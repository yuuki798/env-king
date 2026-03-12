package skills

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

// InstallSkillRequest 安装请求：支持完整命令或 source+skill
type InstallSkillRequest struct {
	Command string `json:"command"` // 完整命令，如 npx skills add https://github.com/vercel-labs/skills --skill find-skills
	Source  string `json:"source"`  // 或单独传 source
	Skill   string `json:"skill"`  // 与 source 配合使用
}

var defaultSkillsDir = filepath.Join(os.TempDir(), "env-king-skills")

func resolveSkillsDir() string {
	if d := os.Getenv("ENV_KING_SKILLS_DIR"); d != "" {
		return d
	}
	if d := strings.TrimSpace(viper.GetString("skills.work_dir")); d != "" {
		abs, err := filepath.Abs(d)
		if err == nil {
			return abs
		}
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

// extractSkillName 从命令或 source/skill 参数中提取简洁的技能目录名
// 优先级: 显式 skill 参数 > 命令中 --skill 或 -s 参数 > normalizeName(commandOrSource)
func extractSkillName(commandOrSource, skill string) string {
	if skill != "" {
		return normalizeName(skill)
	}
	// 尝试从命令字符串中找 --skill xxx 或 -s xxx
	parts := strings.Fields(commandOrSource)
	for i, p := range parts {
		if (p == "--skill" || p == "-s") && i+1 < len(parts) {
			return normalizeName(parts[i+1])
		}
	}
	return normalizeName(commandOrSource)
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

// InstallSkill 执行 npx skills add 安装 Skill（支持完整命令或 source+skill）
// 自动添加 -y 跳过交互，并通过 stdin 持续发送回车应对意外提示
func InstallSkill(commandOrSource, skill string) error {
	cmdStr := strings.TrimSpace(commandOrSource)
	if cmdStr == "" {
		return errors.New("command 或 source 不能为空")
	}
	// 若为 source+skill 形式，构造完整命令
	if skill != "" {
		cmdStr = "npx skills add " + cmdStr + " --skill " + skill
	}
	// 确保非交互：无 -y/--yes 时追加
	if !strings.Contains(cmdStr, "-y") && !strings.Contains(cmdStr, "--yes") {
		cmdStr += " -y"
	}
	// 追加 -g -a cursor 安装到 Cursor 全局目录
	if !strings.Contains(cmdStr, "-g") && !strings.Contains(cmdStr, "--global") {
		cmdStr += " -g -a cursor"
	}
	// skillsHome：技能仓库根目录（用于 SKILLS_DIR、元数据等）
	skillsHome := resolveSkillsDir()
	_ = os.MkdirAll(skillsHome, 0o755)
	// 默认在 workdir 下执行：即 skillsHome 的上一级目录
	execDir := filepath.Dir(skillsHome)
	cmd := exec.Command("sh", "-c", cmdStr)
	cmd.Dir = execDir
	cmd.Env = append(os.Environ(), "SKILLS_DIR="+skillsHome, "DISABLE_TELEMETRY=1")
	// 持续发送回车以应对任何交互提示
	cmd.Stdin = bytes.NewReader(bytes.Repeat([]byte("\n"), 200))
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		log.Printf("[skills] npx skills add failed: %v stdout=%s stderr=%s", err, out.String(), errOut.String())
		return errors.New("npx skills add 执行失败: " + errOut.String())
	}
	// 记录安装到本地 registry
	// 优先级：显式 skill 参数 > 从命令中提取 --skill 参数值 > 规范化完整命令
	name := extractSkillName(commandOrSource, skill)
	target := filepath.Join(resolveSkillDir(), name)
	_ = os.MkdirAll(target, 0o755)
	manifest := map[string]string{"command": cmdStr, "id": uuid.Must(uuid.NewRandom()).String()}
	manifestRaw, _ := json.MarshalIndent(manifest, "", "  ")
	_ = os.WriteFile(filepath.Join(target, "manifest.json"), manifestRaw, 0o644)

	// 从 npx 输出中提取 SKILL.md 内容（或构造最简 SKILL.md），供 Agent 发现使用
	// npx skills add 会将文件安装到 ~/.cursor/skills/{name}/SKILL.md
	// 尝试从该路径读取并保存到本地 registry，若不存在则写一个最简占位文件
	skillMdPath := filepath.Join(target, "SKILL.md")
	if _, err := os.Stat(skillMdPath); os.IsNotExist(err) {
		displayName := skill
		if displayName == "" {
			displayName = name
		}
		// 仅在 workdir 内查找：从本项目的 .agents/skills 目录中复制“真正的” SKILL 定义
		if cwd, err2 := os.Getwd(); err2 == nil {
			projectRoot := filepath.Dir(cwd)
			candidates := []string{
				filepath.Join(projectRoot, ".agents", "skills", displayName, "SKILL.md"),
				filepath.Join(projectRoot, ".agents", "skills", name, "SKILL.md"),
			}
			var skillMdContent []byte
			for _, c := range candidates {
				if data, err := os.ReadFile(c); err == nil {
					skillMdContent = data
					break
				}
			}
			if len(skillMdContent) == 0 {
				// 没找到则写最简占位（无 pages = 全局生效）
				skillMdContent = []byte("---\nname: " + displayName + "\ndescription: 通过 npx skills add 安装的技能\n---\n\n# " + displayName + "\n\n此技能通过 `" + cmdStr + "` 安装。\n")
			}
			_ = os.WriteFile(skillMdPath, skillMdContent, 0o644)
			return nil
		}
		// 如果连项目根目录都拿不到，兜底写占位
		skillMdContent := []byte("---\nname: " + displayName + "\ndescription: 通过 npx skills add 安装的技能\n---\n\n# " + displayName + "\n\n此技能通过 `" + cmdStr + "` 安装。\n")
		_ = os.WriteFile(skillMdPath, skillMdContent, 0o644)
	}
	return nil
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

// GetConfigModels 从 config 的 agent.llm / agent.llms 读取模型列表（用于负载均衡）
func GetConfigModels() []ModelEndpoint {
	var out []ModelEndpoint
	// agent.llms 数组优先
	llms := viper.Get("agent.llms")
	if arr, ok := llms.([]interface{}); ok && len(arr) > 0 {
		for _, v := range arr {
			m := viperMapToModel(v)
			if m != nil {
				out = append(out, *m)
			}
		}
		return out
	}
	// 回退到 agent.llm 单条
	m := viperMapToModel(viper.Get("agent.llm"))
	if m != nil {
		out = append(out, *m)
	}
	return out
}

func viperMapToModel(v interface{}) *ModelEndpoint {
	mp, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	baseURL := getMapStr(mp, "base_url", "baseUrl")
	if baseURL == "" {
		return nil
	}
	name := getMapStr(mp, "name")
	if name == "" {
		name = getMapStr(mp, "model")
	}
	if name == "" {
		name = "glm"
	}
	return &ModelEndpoint{
		Name:   name,
		URL:    baseURL,
		APIKey: getMapStr(mp, "api_key", "apiKey"),
		Weight: 1,
	}
}

func getMapStr(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			if s, ok := v.(string); ok {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

// GetLoadBalanceConfig 获取负载配置，合并 config 中的模型列表
func GetLoadBalanceConfig() (*LoadBalanceConfig, error) {
	_ = os.MkdirAll(resolveSkillsDir(), 0o755)
	path := resolveLBPath()
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &LoadBalanceConfig{Models: append([]ModelEndpoint{}, GetConfigModels()...)}, nil
		}
		return nil, err
	}
	var cfg LoadBalanceConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return &LoadBalanceConfig{Models: GetConfigModels()}, nil
	}
	if cfg.Models == nil {
		cfg.Models = []ModelEndpoint{}
	}
	// config 中的模型放在前面（若 load-balance.json 无同名则追加）
	configModels := GetConfigModels()
	seen := make(map[string]bool)
	for _, m := range cfg.Models {
		seen[m.Name] = true
	}
	for _, m := range configModels {
		if !seen[m.Name] {
			cfg.Models = append([]ModelEndpoint{m}, cfg.Models...)
			seen[m.Name] = true
		}
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

var (
	defaultMemoryStore     MemoryStore
	defaultMemoryStoreOnce sync.Once
)

// GetMemoryStore 返回全局单例 MemoryStore，供 Agent 对话后写入长短记忆使用
func GetMemoryStore() MemoryStore {
	defaultMemoryStoreOnce.Do(func() {
		defaultMemoryStore = NewMemoryStore()
	})
	return defaultMemoryStore
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
