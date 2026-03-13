package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// chatIDContextKey 用于在 context 中携带飞书 chat_id，供工具执行层写通知映射。
type chatIDContextKey struct{}

// WithChatID 将飞书 chat_id 注入 context，供 HTTP 工具执行时读取。
func WithChatID(ctx context.Context, chatID string) context.Context {
	return context.WithValue(ctx, chatIDContextKey{}, chatID)
}

func chatIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(chatIDContextKey{}).(string)
	return v
}

// ToolCall 是 [TOOL_CALL]...[/TOOL_CALL] 块解析出的结构。
type ToolCall struct {
	Method string      `json:"method"`
	Path   string      `json:"path"`
	Body   interface{} `json:"body,omitempty"`
}

var toolCallRe = regexp.MustCompile(`(?s)\[TOOL_CALL\](.*?)\[/TOOL_CALL\]`)

// ParseToolCalls 提取 LLM 输出中所有 [TOOL_CALL]...[/TOOL_CALL] 块。
func ParseToolCalls(text string) []ToolCall {
	matches := toolCallRe.FindAllStringSubmatch(text, -1)
	var calls []ToolCall
	for _, m := range matches {
		raw := strings.TrimSpace(m[1])
		var tc ToolCall
		if err := json.Unmarshal([]byte(raw), &tc); err == nil && tc.Path != "" {
			calls = append(calls, tc)
		}
	}
	return calls
}

// StripToolCalls 从 LLM 输出中去掉所有 [TOOL_CALL]...[/TOOL_CALL] 块。
func StripToolCalls(text string) string {
	return strings.TrimSpace(toolCallRe.ReplaceAllString(text, ""))
}

// envKingBaseURL 返回 env-king 自身 API 的 baseURL（http://127.0.0.1:<port>/api）。
func envKingBaseURL() string {
	addr := strings.TrimSpace(viper.GetString("server.addr"))
	if addr == "" {
		addr = "0.0.0.0:8080"
	}
	addr = strings.ReplaceAll(addr, "0.0.0.0", "127.0.0.1")
	addr = strings.ReplaceAll(addr, "[::]", "127.0.0.1")
	if !strings.HasPrefix(addr, "http") {
		return "http://" + addr + "/api"
	}
	return addr + "/api"
}

var toolHTTPClient = &http.Client{Timeout: 30 * time.Second}

// ExecuteToolCall 向 env-king 自身 API 发起一次 HTTP 请求，并返回格式化的结果字符串。
// 若是执行工作流的请求（返回含 job_id 的响应），会自动将 job_id -> chat_id 存入 bbolt，
// 供任务完成后主动推送飞书通知使用。
func ExecuteToolCall(ctx context.Context, tc ToolCall) (string, error) {
	base := envKingBaseURL()
	url := base + tc.Path

	var bodyReader io.Reader
	if tc.Body != nil {
		bodyBytes, err := json.Marshal(tc.Body)
		if err != nil {
			return "", fmt.Errorf("序列化请求体失败: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	method := strings.ToUpper(strings.TrimSpace(tc.Method))
	if method == "" {
		method = "GET"
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return "", fmt.Errorf("构建 HTTP 请求失败: %w", err)
	}
	if tc.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := toolHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	result := fmt.Sprintf("HTTP %d\n%s", resp.StatusCode, string(respBody))

	// 执行工作流后，若响应包含 job id，存储 job_id -> chat_id 映射，用于任务完成通知
	if method == "POST" && strings.Contains(tc.Path, "/run") &&
		resp.StatusCode >= 200 && resp.StatusCode < 300 {
		chatID := chatIDFromContext(ctx)
		if chatID != "" {
			var jobResp struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(respBody, &jobResp); err == nil && jobResp.ID != "" {
				if st, serr := getFeishuStore(); serr == nil {
					_ = st.PutString(feishuBucketJobNotify, jobResp.ID, []byte(chatID))
				}
			}
		}
	}

	return result, nil
}
