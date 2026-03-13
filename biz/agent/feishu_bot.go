package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"yuuki798/env-king/infra"
)

const feishuBucketSessions = "feishu_sessions"

// FeishuBucketJobNotify 是 job_id -> chat_id 映射的 bbolt bucket 名（导出供 biz/feishu 包读取）。
const FeishuBucketJobNotify = "feishu_job_notify"

// feishuBucketJobNotify 包内用的小写别名
const feishuBucketJobNotify = FeishuBucketJobNotify

// GetFeishuStoreExported 返回飞书用 bbolt store（导出给 biz/feishu 包读取 job 通知映射）。
func GetFeishuStoreExported() (*infra.Store, error) {
	return getFeishuStore()
}

var (
	feishuStoreOnce sync.Once
	feishuStore     *infra.Store
	feishuStoreErr  error
)

// getFeishuStore 打开用于存储 Feishu 相关状态的本地 Store。
// 目前仅用于 chat_id -> agent 会话 sessionID 的映射。
func getFeishuStore() (*infra.Store, error) {
	feishuStoreOnce.Do(func() {
		base := resolveWorkspaceDir()
		dir := filepath.Join(base, "store")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			feishuStoreErr = fmt.Errorf("创建 feishu store 目录失败: %w", err)
			return
		}
		feishuStore, feishuStoreErr = infra.OpenStore(filepath.Join(dir, "feishu.db"), nil)
	})
	return feishuStore, feishuStoreErr
}

// FeishuBot 将 Feishu 群聊与 env-king Agent 对话打通：
// - 为每个 Feishu chat_id 维护一个独立的会话 session；
// - 将群消息转为 Agent 输入，使用 ChatStream 生成回复；
// - 会话持久化到 workdir/.env-king-agent/conversations 下。
//
// 注意：本对象只负责“会话与回复”的逻辑，不直接依赖飞书 SDK。
// 调用方负责通过长连接 / Webhook 接入飞书，并在收到消息时调用 HandleIncomingMessage。
type FeishuBot struct{}

func NewFeishuBot() *FeishuBot {
	return &FeishuBot{}
}

// HandleIncomingMessage 处理来自某个 Feishu 群（chatID）的文本消息：
// - 根据 chatID 查找/创建对应的 Agent 会话；
// - 复用 CompressSessionHistory 控制上下文长度；
// - 调用 ChatStream 生成回复，并写回会话；
// - 返回最终回复文本，由调用方发送到飞书群。
func (b *FeishuBot) HandleIncomingMessage(ctx context.Context, chatID, userText string) (string, error) {
	chatID = strings.TrimSpace(chatID)
	userText = strings.TrimSpace(userText)
	if chatID == "" || userText == "" {
		return "", fmt.Errorf("chatID 和 userText 不能为空")
	}

	st, err := getFeishuStore()
	if err != nil {
		return "", fmt.Errorf("打开 Feishu store 失败: %w", err)
	}

	// 1. 尝试读取该群对应的会话 ID
	var sessionID string
	if raw, err := st.GetString(feishuBucketSessions, chatID); err == nil && len(raw) > 0 {
		sessionID = string(raw)
	}

	// 2. 基于 sessionID 构造 history（带压缩）
	history := make([]struct{ Role, Content string }, 0)
	if sessionID != "" {
		if sess, err := GetConversation(sessionID); err == nil && sess != nil {
			if sum, turns, changed, cerr := CompressSessionHistory(ctx, sess); cerr == nil {
				// 将摘要作为一条虚拟助手消息加入 history 开头，保持与 HTTP 层一致的语义
				if s := strings.TrimSpace(sum); s != "" {
					history = append(history, struct{ Role, Content string }{
						Role:    "assistant",
						Content: "（以下是本会话的简要摘要，请参考其中的事实与约束。）\n" + s,
					})
				}
				for _, t := range turns {
					history = append(history, struct{ Role, Content string }{Role: t.Role, Content: t.Content})
				}
				if changed {
					_ = SaveConversation(sess)
				}
			} else {
				// 压缩失败时退回到完整历史
				for _, m := range sess.Messages {
					history = append(history, struct{ Role, Content string }{Role: m.Role, Content: m.Content})
				}
			}
		}
	}

	// 3. 使用持久化的默认技能，请求 LLM 时带上
	//    同时将 chatID 注入 context，供 HTTP 工具层记录 job_id -> chat_id 映射
	activeSkills, _ := GetDefaultSkillIDs()
	if activeSkills == nil {
		activeSkills = []string{}
	}
	ctxWithChat := WithChatID(ctx, chatID)
	fullReply, _, err := ChatStream(ctxWithChat, "", userText, history, activeSkills, nil)
	if err != nil {
		return "", err
	}

	// 4. 将本轮对话写回会话，并在无会话时创建新会话
	if sessionID != "" {
		if _, err := AppendToConversation(sessionID, userText, fullReply); err != nil {
			return "", err
		}
	} else {
		sess, err := CreateConversation(userText, fullReply)
		if err != nil {
			return "", err
		}
		sessionID = sess.ID
		// 建立 chatID -> sessionID 的映射，后续消息复用上下文
		if err := st.PutString(feishuBucketSessions, chatID, []byte(sessionID)); err != nil {
			return "", fmt.Errorf("保存 Feishu 会话映射失败: %w", err)
		}
	}

	return fullReply, nil
}

