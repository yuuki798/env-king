package agent

import (
	"context"
	"encoding/json"
	"log"
	"regexp"
	"strings"
	"sync"

	"github.com/spf13/viper"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
)

const formFillPrefix = "[FORM_FILL:"

var (
	llmOnce sync.Once
	llm     llms.Model
	llmErr  error
)

func getLLM() (llms.Model, error) {
	llmOnce.Do(func() {
		typ := strings.TrimSpace(strings.ToLower(viper.GetString("agent.llm.type")))
		if typ == "" {
			typ = "ollama"
		}
		model := strings.TrimSpace(viper.GetString("agent.llm.model"))

		switch typ {
		case "openai":
			// OpenAI 兼容 API：智谱 GLM、OpenAI、Azure 等
			apiKey := strings.TrimSpace(viper.GetString("agent.llm.api_key"))
			baseURL := strings.TrimSpace(viper.GetString("agent.llm.base_url"))
			if baseURL == "" {
				baseURL = "https://api.openai.com/v1"
			}
			if model == "" {
				model = "gpt-4o-mini"
			}
			opts := []openai.Option{openai.WithToken(apiKey), openai.WithBaseURL(baseURL), openai.WithModel(model)}
			llm, llmErr = openai.New(opts...)
		default:
			// Ollama 本地服务
			base := strings.TrimSpace(viper.GetString("agent.llm.ollama_base"))
			if base == "" {
				base = "http://127.0.0.1:11434"
			}
			if model == "" {
				model = "qwen2.5:7b"
			}
			llm, llmErr = ollama.New(
				ollama.WithServerURL(base),
				ollama.WithModel(model),
			)
		}
	})
	return llm, llmErr
}

// FormFillAction 供前端自动填表的结构。
type FormFillAction struct {
	Page    string          `json:"page"`    // 如 script, config
	Form    string          `json:"form"`    // 如 createWorkflow
	Payload json.RawMessage `json:"payload"` // 表单字段 JSON，由前端按表单解析
}

// ChatStream 流式对话：对 streamFn 逐 chunk 回调，返回完整回复与解析出的 form_fill（若有）。
// activeSkills 为前端传入的已激活 skill ID 列表，空时退回基础 prompt。
func ChatStream(ctx context.Context, currentPage, userMessage string, history []struct{ Role, Content string }, activeSkills []string, streamFn func(ctx context.Context, chunk []byte) error) (fullReply string, formFill *FormFillAction, err error) {
	model, err := getLLM()
	if err != nil {
		return "", nil, err
	}

	systemPrompt := BuildSystemPromptFromSkills(currentPage, activeSkills)
	log.Printf("[agent] chat page=%s activeSkills=%v", currentPage, activeSkills)
	messages := buildMessages(systemPrompt, userMessage, history)

	var full strings.Builder
	streamFunc := func(ctx context.Context, chunk []byte) error {
		full.Write(chunk)
		if streamFn != nil {
			return streamFn(ctx, chunk)
		}
		return nil
	}

	opts := []llms.CallOption{llms.WithStreamingFunc(streamFunc)}
	_, err = model.GenerateContent(ctx, messages, opts...)
	if err != nil {
		return full.String(), nil, err
	}

	reply := full.String()
	formFill = parseFormFill(reply)
	// 对话日志（便于排查与审计）
	log.Printf("[agent] chat user=%q", truncateForLog(userMessage, 200))
	log.Printf("[agent] chat reply=%q form_fill=%v", truncateForLog(reply, 500), formFill != nil)
	return reply, formFill, nil
}

func truncateForLog(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}


func buildMessages(systemPrompt, userMessage string, history []struct{ Role, Content string }) []llms.MessageContent {
	messages := make([]llms.MessageContent, 0, len(history)+3)
	messages = append(messages, llms.MessageContent{
		Role:  llms.ChatMessageTypeSystem,
		Parts: []llms.ContentPart{llms.TextContent{Text: systemPrompt}},
	})
	for _, h := range history {
		role := llms.ChatMessageTypeHuman
		if h.Role == "assistant" || h.Role == "ai" {
			role = llms.ChatMessageTypeAI
		}
		messages = append(messages, llms.MessageContent{
			Role:  role,
			Parts: []llms.ContentPart{llms.TextContent{Text: h.Content}},
		})
	}
	messages = append(messages, llms.MessageContent{
		Role:  llms.ChatMessageTypeHuman,
		Parts: []llms.ContentPart{llms.TextContent{Text: userMessage}},
	})
	return messages
}

var formFillHeadRe = regexp.MustCompile(`\[FORM_FILL:([^\]]+)\]\s*\n`)

func parseFormFill(reply string) *FormFillAction {
	idx := strings.Index(reply, formFillPrefix)
	if idx < 0 {
		return nil
	}
	rest := reply[idx:]
	loc := formFillHeadRe.FindStringIndex(rest)
	if loc == nil {
		return nil
	}
	pageForm := strings.TrimSpace(formFillHeadRe.FindStringSubmatch(rest)[1])
	jsonStart := loc[1]
	// 找第一个 {，再匹配到对应的 }
	brace := strings.Index(rest[jsonStart:], "{")
	if brace < 0 {
		return nil
	}
	start := jsonStart + brace
	depth := 0
	for i := start; i < len(rest); i++ {
		switch rest[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				payloadStr := rest[start : i+1]
				var payload json.RawMessage
				if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
					return nil
				}
				parts := strings.SplitN(pageForm, ".", 2)
				page := pageForm
				form := ""
				if len(parts) == 2 {
					page = strings.TrimSpace(parts[0])
					form = strings.TrimSpace(parts[1])
				}
				return &FormFillAction{Page: page, Form: form, Payload: payload}
			}
		}
	}
	return nil
}
