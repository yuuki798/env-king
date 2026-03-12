package skills

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

// OpenClawMemoryConfig 委托 openclaw 时的配置
type OpenClawMemoryConfig struct {
	Enabled bool   // 是否启用 openclaw 记忆
	Bin     string // openclaw 可执行路径，如 "./openclaw/cli.js" 或 "node" + "openclaw/dist/cli.js"
	WorkDir string // 工作区目录，与 agent.workspace_dir 对齐
	AgentID string // agent 标识，默认 "main"
}

// NewOpenClawMemorySearchManager 创建委托 openclaw 的记忆检索实现（若 bin 不可用则返回 nil）
// 设计对齐 openclaw：memory 来源 = MEMORY.md + memory/*.md，sessions 来源 = 会话转录。
func NewOpenClawMemorySearchManager(cfg OpenClawMemoryConfig) (MemorySearchManager, error) {
	if !cfg.Enabled || strings.TrimSpace(cfg.Bin) == "" {
		return nil, nil
	}
	workDir := strings.TrimSpace(cfg.WorkDir)
	if workDir == "" {
		return nil, nil
	}
	agentID := strings.TrimSpace(cfg.AgentID)
	if agentID == "" {
		agentID = "main"
	}
	return &openclawMemoryAdapter{
		bin:     strings.Fields(cfg.Bin),
		workDir: workDir,
		agentID: agentID,
	}, nil
}

type openclawMemoryAdapter struct {
	bin     []string // 第一个为命令，后续为默认参数
	workDir string
	agentID string
}

func (a *openclawMemoryAdapter) Search(ctx context.Context, query string, opts *SearchOptions) ([]MemorySearchResult, error) {
	// 调用 openclaw memory search（若 openclaw 暴露了对应 CLI）
	// 示例：openclaw memory search --query "..." [--max-results 6] [--min-score 0.35]
	args := append([]string{}, a.bin[1:]...)
	args = append(args, "memory", "search", "--query", query)
	if opts != nil {
		if opts.MaxResults > 0 {
			args = append(args, "--max-results", strconv.Itoa(opts.MaxResults))
		}
		if opts.MinScore > 0 {
			args = append(args, "--min-score", strconv.FormatFloat(opts.MinScore, 'f', -1, 64))
		}
	}
	cmd := exec.CommandContext(ctx, a.bin[0], args...)
	cmd.Dir = a.workDir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var raw struct {
		Results []MemorySearchResult `json:"results"`
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, err
	}
	return raw.Results, nil
}

func (a *openclawMemoryAdapter) ReadFile(ctx context.Context, relPath string, from, lines *int) (text, path string, err error) {
	args := append([]string{}, a.bin[1:]...)
	args = append(args, "memory", "get", "--path", relPath)
	if from != nil {
		args = append(args, "--from", strconv.Itoa(*from))
	}
	if lines != nil {
		args = append(args, "--lines", strconv.Itoa(*lines))
	}
	cmd := exec.CommandContext(ctx, a.bin[0], args...)
	cmd.Dir = a.workDir
	out, err := cmd.Output()
	if err != nil {
		return "", relPath, err
	}
	var raw struct {
		Text string `json:"text"`
		Path string `json:"path"`
	}
	_ = json.Unmarshal(out, &raw)
	return raw.Text, raw.Path, nil
}

// GetMemorySearchManager 根据配置返回记忆检索实现：若启用且可用的 openclaw 则委托 openclaw，否则返回 NoopMemorySearchManager
// workspaceDir 建议与 agent.workspace_dir 一致，用于 MEMORY.md / memory/*.md 与 openclaw 工作区对齐
func GetMemorySearchManager(workspaceDir string) MemorySearchManager {
	if !viper.GetBool("memory.enabled") {
		return NoopMemorySearchManager{}
	}
	cfg := OpenClawMemoryConfig{
		Enabled: viper.GetBool("memory.openclaw.enabled"),
		Bin:     strings.TrimSpace(viper.GetString("memory.openclaw.bin")),
		WorkDir: workspaceDir,
		AgentID: strings.TrimSpace(viper.GetString("memory.openclaw.agent_id")),
	}
	if cfg.Bin != "" && !filepath.IsAbs(cfg.Bin) && !strings.HasPrefix(cfg.Bin, "node") {
		cfg.Bin = ResolveOpenClawBin(cfg.Bin, filepath.Dir(workspaceDir))
	}
	mgr, err := NewOpenClawMemorySearchManager(cfg)
	if err != nil || mgr == nil {
		return NoopMemorySearchManager{}
	}
	return mgr
}

// ResolveOpenClawBin 解析 openclaw 可执行路径：若为相对路径则基于 workdir 解析
func ResolveOpenClawBin(bin, workDir string) string {
	bin = strings.TrimSpace(bin)
	if bin == "" {
		return ""
	}
	parts := strings.Fields(bin)
	if len(parts) == 0 {
		return ""
	}
	first := parts[0]
	if filepath.IsAbs(first) || first == "node" || first == "npx" {
		return bin
	}
	abs := filepath.Join(workDir, first)
	if _, err := os.Stat(abs); err == nil {
		return abs + " " + strings.Join(parts[1:], " ")
	}
	return bin
}
