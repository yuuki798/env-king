package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AppendLongTermNote 将一轮对话摘要以 Markdown 形式追加到长期记忆文件中。
// workspaceDir 建议为 agent.workspace_dir，对应的长期记忆目录为 {workspaceDir}/memory。
// 当前实现使用单一文件 long-term.md，后续可按需拆分。
func AppendLongTermNote(workspaceDir, sessionID, userMsg, assistantReply string) error {
	workspaceDir = strings.TrimSpace(workspaceDir)
	if workspaceDir == "" {
		return nil
	}
	memDir := filepath.Join(workspaceDir, "memory")
	if err := os.MkdirAll(memDir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(memDir, "long-term.md")

	now := time.Now().UTC().Format(time.RFC3339)
	var b strings.Builder
	// 标题包含时间戳和可选的 session ID，便于后续检索
	if sessionID != "" {
		fmt.Fprintf(&b, "## %s (session %s)\n\n", now, sessionID)
	} else {
		fmt.Fprintf(&b, "## %s\n\n", now)
	}
	userMsg = strings.TrimSpace(userMsg)
	assistantReply = strings.TrimSpace(assistantReply)
	if userMsg != "" {
		fmt.Fprintf(&b, "**User:** %s\n\n", userMsg)
	}
	if assistantReply != "" {
		fmt.Fprintf(&b, "**Assistant:** %s\n\n", assistantReply)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(b.String()); err != nil {
		return err
	}
	return nil
}

