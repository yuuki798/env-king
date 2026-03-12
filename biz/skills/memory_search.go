package skills

import "context"

// MemorySearchResult 与 openclaw MemorySearchResult 对齐，用于语义检索返回的片段
type MemorySearchResult struct {
	Path      string  `json:"path"`
	StartLine int     `json:"startLine"`
	EndLine   int     `json:"endLine"`
	Score     float64 `json:"score"`
	Snippet   string  `json:"snippet"`
	Source    string  `json:"source"` // "memory" | "sessions"
}

// SearchOptions 记忆检索选项
type SearchOptions struct {
	MaxResults int     // 默认 6
	MinScore   float64 // 默认 0.35
	SessionKey string  // 可选，会话维度或 warm 用
}

// MemorySearchManager 长短期记忆的语义检索与按路径读（对应 openclaw 的 memory_search + memory_get）
// 长期来源 memory：MEMORY.md、memory/*.md；短期来源 sessions：会话转录（可选）。
// 实现可以是委托 openclaw，或 Go 内 SQLite + Embedding API。
type MemorySearchManager interface {
	Search(ctx context.Context, query string, opts *SearchOptions) ([]MemorySearchResult, error)
	ReadFile(ctx context.Context, relPath string, from, lines *int) (text, path string, err error)
}

// NoopMemorySearchManager 空实现：未配置记忆检索时返回空结果，不报错
type NoopMemorySearchManager struct{}

func (NoopMemorySearchManager) Search(ctx context.Context, _ string, _ *SearchOptions) ([]MemorySearchResult, error) {
	return nil, nil
}

func (NoopMemorySearchManager) ReadFile(ctx context.Context, relPath string, _, _ *int) (text, path string, err error) {
	return "", relPath, nil
}
