package skills

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

var defaultSkillsDir = filepath.Join(os.TempDir(), "env-king-skills")

func resolveSkillsDir() string {
	if d := os.Getenv("ENV_KING_SKILLS_DIR"); d != "" {
		return d
	}
	return defaultSkillsDir
}

func resolveMcpDir() string   { return filepath.Join(resolveSkillsDir(), "mcp") }
func resolveSkillDir() string { return filepath.Join(resolveSkillsDir(), "skills") }
func resolveLBPath() string   { return filepath.Join(resolveSkillsDir(), "load-balance.json") }

var nameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func normalizeName(source string) string {
	s := strings.TrimSpace(source)
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.Trim(s, "/")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "@", "")
	s = nameSanitizer.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-.")
	if s == "" {
		s = "item"
	}
	return s
}

func listDirs(dir string) []string {
	_ = os.MkdirAll(dir, 0o755)
	d, err := os.ReadDir(dir)
	if err != nil {
		return []string{}
	}
	out := make([]string, 0, len(d))
	for _, e := range d {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

// ListMCP 返回已安装的 MCP 列表
func ListMCP() []string { return listDirs(resolveMcpDir()) }

// InstallMCP 安装 MCP（最小可用：目录+source 文件）
func InstallMCP(source string) error {
	source = strings.TrimSpace(source)
	if source == "" {
		return errors.New("source 不能为空")
	}
	name := normalizeName(source)
	target := filepath.Join(resolveMcpDir(), name)
	if err := os.MkdirAll(target, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(target, "source.txt"), []byte(source+"\n"), 0o644)
}

// UninstallMCP 卸载 MCP
func UninstallMCP(name string) error {
	name = normalizeName(name)
	return os.RemoveAll(filepath.Join(resolveMcpDir(), name))
}

// ListSkills 返回已安装的 Skills（参考 openclaw skills）
func ListSkills() []string { return listDirs(resolveSkillDir()) }

// InstallSkill 安装 Skill（最小可用：目录+source 文件）
func InstallSkill(source string) error {
	source = strings.TrimSpace(source)
	if source == "" {
		return errors.New("source 不能为空")
	}
	name := normalizeName(source)
	target := filepath.Join(resolveSkillDir(), name)
	if err := os.MkdirAll(target, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(target, "source.txt"), []byte(source+"\n"), 0o644)
}

// UninstallSkill 卸载 Skill
func UninstallSkill(name string) error {
	name = normalizeName(name)
	return os.RemoveAll(filepath.Join(resolveSkillDir(), name))
}

// LoadBalanceConfig Token/模型负载均衡配置
type LoadBalanceConfig struct {
	Models []ModelEndpoint `json:"models"`
}

type ModelEndpoint struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	APIKey string `json:"apiKey,omitempty"`
	Weight int    `json:"weight"`
}

// GetLoadBalanceConfig 获取负载配置
func GetLoadBalanceConfig() (*LoadBalanceConfig, error) {
	_ = os.MkdirAll(resolveSkillsDir(), 0o755)
	path := resolveLBPath()
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &LoadBalanceConfig{Models: []ModelEndpoint{}}, nil
		}
		return nil, err
	}
	var cfg LoadBalanceConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return &LoadBalanceConfig{Models: []ModelEndpoint{}}, nil
	}
	if cfg.Models == nil {
		cfg.Models = []ModelEndpoint{}
	}
	return &cfg, nil
}

// SetLoadBalanceConfig 设置负载配置
func SetLoadBalanceConfig(cfg *LoadBalanceConfig) error {
	if cfg == nil {
		return errors.New("cfg 不能为空")
	}
	_ = os.MkdirAll(resolveSkillsDir(), 0o755)
	raw, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(resolveLBPath(), raw, 0o644)
}

// MemoryStore 长短期记忆存储
type MemoryStore interface {
	AddShortTerm(k, v string) error
	GetShortTerm(k string) (string, bool)
	AddLongTerm(k, v string) error
	GetLongTerm(k string) (string, bool)
}

// NewMemoryStore 创建记忆存储（最小可用：进程内 map）
func NewMemoryStore() MemoryStore {
	return &memoryStoreImpl{short: map[string]string{}, long: map[string]string{}}
}

type memoryStoreImpl struct {
	mu    sync.RWMutex
	short map[string]string
	long  map[string]string
}

func (m *memoryStoreImpl) AddShortTerm(k, v string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.short[k] = v
	return nil
}

func (m *memoryStoreImpl) GetShortTerm(k string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.short[k]
	return v, ok
}

func (m *memoryStoreImpl) AddLongTerm(k, v string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.long[k] = v
	return nil
}

func (m *memoryStoreImpl) GetLongTerm(k string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.long[k]
	return v, ok
}
