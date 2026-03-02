package skills

import (
	"os"
	"path/filepath"
)

var defaultSkillsDir = filepath.Join(os.TempDir(), "env-king-skills")

func resolveSkillsDir() string {
	if d := os.Getenv("ENV_KING_SKILLS_DIR"); d != "" {
		return d
	}
	return defaultSkillsDir
}

func resolveMcpDir() string {
	// TODO: 从配置读取
	return filepath.Join(resolveSkillsDir(), "mcp")
}

// ListMCP 返回已安装的 MCP 列表
func ListMCP() []string {
	// TODO: 从 mcp 目录扫描
	dir := resolveMcpDir()
	_ = os.MkdirAll(dir, 0755)
	return []string{}
}

// InstallMCP 安装 MCP
func InstallMCP(source string) error {
	// TODO: 支持 npm/url 安装，参考 MCP 规范
	_ = source
	return nil
}

// UninstallMCP 卸载 MCP
func UninstallMCP(name string) error {
	// TODO
	_ = name
	return nil
}

// ListSkills 返回已安装的 Skills（参考 openclaw skills）
func ListSkills() []string {
	dir := resolveSkillsDir()
	_ = os.MkdirAll(dir, 0755)
	d, _ := os.ReadDir(dir)
	out := make([]string, 0, len(d))
	for _, e := range d {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out
}

// InstallSkill 安装 Skill
func InstallSkill(source string) error {
	// TODO: 支持 npm/url，校验 openclaw 兼容格式
	_ = source
	return nil
}

// UninstallSkill 卸载 Skill
func UninstallSkill(name string) error {
	// TODO
	_ = name
	return nil
}

// LoadBalanceConfig Token/模型负载均衡配置
type LoadBalanceConfig struct {
	Models []ModelEndpoint `json:"models"`
}

type ModelEndpoint struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	APIKey string `json:"apiKey,omitempty"`
	Weight int    `json:"weight"` // 负载权重
}

// GetLoadBalanceConfig 获取负载配置
func GetLoadBalanceConfig() (*LoadBalanceConfig, error) {
	// TODO: 从配置读取
	return &LoadBalanceConfig{Models: []ModelEndpoint{}}, nil
}

// SetLoadBalanceConfig 设置负载配置
func SetLoadBalanceConfig(cfg *LoadBalanceConfig) error {
	// TODO
	_ = cfg
	return nil
}

// MemoryStore 长短期记忆存储
type MemoryStore interface {
	AddShortTerm(k, v string) error
	GetShortTerm(k string) (string, bool)
	AddLongTerm(k, v string) error
	GetLongTerm(k string) (string, bool)
}

// NewMemoryStore 创建记忆存储（占位）
func NewMemoryStore() MemoryStore {
	// TODO: 实现基于 sqlite/redis 的记忆
	return &memoryStoreImpl{}
}

type memoryStoreImpl struct{}

func (m *memoryStoreImpl) AddShortTerm(k, v string) error       { return nil }
func (m *memoryStoreImpl) GetShortTerm(k string) (string, bool) { return "", false }
func (m *memoryStoreImpl) AddLongTerm(k, v string) error        { return nil }
func (m *memoryStoreImpl) GetLongTerm(k string) (string, bool)  { return "", false }
