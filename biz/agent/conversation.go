package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ConversationMessage 单条对话消息
type ConversationMessage struct {
	Role      string `json:"role"`      // user | assistant
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"` // ISO8601
}

// Conversation 一次会话（持久化到 workdir/.agents/conversations/{id}.json）
type Conversation struct {
	ID        string                `json:"id"`
	Title     string                `json:"title"`              // 首条用户消息截断，便于列表展示
	CreatedAt string                `json:"createdAt"`
	UpdatedAt string                `json:"updatedAt"`
	Messages  []ConversationMessage `json:"messages"`
	// Summary 为当前会话的压缩摘要，用于控制上下文长度。
	// 早期轮次会被浓缩进 Summary，并仅保留最近若干轮的完整消息发送给 LLM。
	Summary string `json:"summary,omitempty"`
}

func conversationsDir() string {
	return filepath.Join(resolveWorkspaceDir(), "conversations")
}

// ListConversations 返回所有会话（按 UpdatedAt 倒序）
func ListConversations() ([]Conversation, error) {
	dir := conversationsDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var list []Conversation
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var c Conversation
		if err := json.Unmarshal(data, &c); err != nil {
			continue
		}
		list = append(list, c)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].UpdatedAt > list[j].UpdatedAt
	})
	return list, nil
}

// GetConversation 按 ID 读取会话
func GetConversation(id string) (*Conversation, error) {
	if id == "" {
		return nil, os.ErrNotExist
	}
	path := filepath.Join(conversationsDir(), id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Conversation
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// SaveConversation 写入会话文件
func SaveConversation(c *Conversation) error {
	dir := conversationsDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, c.ID+".json")
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func nowISO() string { return time.Now().UTC().Format(time.RFC3339) }

func truncateTitle(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// CreateConversation 创建新会话并写入第一条用户消息与助手回复
func CreateConversation(userMessage, assistantContent string) (*Conversation, error) {
	now := nowISO()
	c := &Conversation{
		ID:        uuid.Must(uuid.NewRandom()).String(),
		Title:     truncateTitle(userMessage, 50),
		CreatedAt: now,
		UpdatedAt: now,
		Messages: []ConversationMessage{
			{Role: "user", Content: userMessage, CreatedAt: now},
			{Role: "assistant", Content: assistantContent, CreatedAt: now},
		},
	}
	return c, SaveConversation(c)
}

// AppendToConversation 在已有会话末尾追加一轮对话并保存
func AppendToConversation(id, userMessage, assistantContent string) (*Conversation, error) {
	c, err := GetConversation(id)
	if err != nil {
		return nil, err
	}
	now := nowISO()
	c.Messages = append(c.Messages,
		ConversationMessage{Role: "user", Content: userMessage, CreatedAt: now},
		ConversationMessage{Role: "assistant", Content: assistantContent, CreatedAt: now},
	)
	c.UpdatedAt = now
	return c, SaveConversation(c)
}

// DeleteConversation 删除会话文件
func DeleteConversation(id string) error {
	if id == "" {
		return os.ErrNotExist
	}
	path := filepath.Join(conversationsDir(), id+".json")
	return os.Remove(path)
}
