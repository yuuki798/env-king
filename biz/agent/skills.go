package agent

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Skill 代表一个已发现的 skill
type Skill struct {
	ID          string   `json:"id"`                // 目录名，如 create-workflow
	Name        string   `json:"name"`              // SKILL.md frontmatter 中的 name
	Description string   `json:"description"`       // SKILL.md frontmatter 中的 description
	Pages       []string `json:"pages"`             // 适用页面，空表示全局
	Content     string   `json:"content,omitempty"` // SKILL.md 完整内容（供 LLM 使用）
}

// skillsDir 返回统一的 skills 目录：workdir/.agents/skills/
// biz/skills 模块通过 skills.work_dir=./workdir/.agents 也写到这里，两者共用同一目录
func skillsDir() string {
	return filepath.Join(resolveWorkspaceDir(), "skills")
}

// ListSkills 扫描 workdir/.agents/skills/*/SKILL.md 返回全部技能
func ListSkills() ([]Skill, error) {
	dir := skillsDir()
	_ = os.MkdirAll(dir, 0o755)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var result []Skill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name(), "SKILL.md"))
		if err != nil {
			continue
		}
		result = append(result, parseSkillMd(e.Name(), string(data)))
	}
	return result, nil
}

// LoadSkillContent 加载指定 skill 的 SKILL.md 内容
func LoadSkillContent(id string) (string, error) {
	p := filepath.Join(skillsDir(), id, "SKILL.md")
	data, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// parseSkillMd 解析 SKILL.md 的 YAML frontmatter（--- ... ---）
func parseSkillMd(id, content string) Skill {
	skill := Skill{ID: id, Name: id, Content: content}
	if !strings.HasPrefix(strings.TrimSpace(content), "---") {
		return skill
	}
	// 提取 frontmatter
	lines := strings.Split(content, "\n")
	inFront := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if !inFront {
				inFront = true
				continue
			}
			break // 结束 frontmatter
		}
		if !inFront {
			continue
		}
		// 简单 key: value 解析
		if k, v, ok := strings.Cut(trimmed, ":"); ok {
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)
			switch k {
			case "name":
				skill.Name = v
			case "description":
				skill.Description = v
			case "pages":
				// pages: [script, dashboard] 或 pages: script
				v = strings.Trim(v, "[]")
				for _, p := range strings.Split(v, ",") {
					if pg := strings.TrimSpace(p); pg != "" {
						skill.Pages = append(skill.Pages, pg)
					}
				}
			}
		}
	}
	return skill
}

// BuildSystemPromptFromSkills 根据激活的 skills 构建 system prompt
// 若无 activeSkills，回退到基础提示
func BuildSystemPromptFromSkills(currentPage string, activeSkillIDs []string) string {
	var sb strings.Builder

	sb.WriteString("你是 Env King 的智能助理，帮助用户在控制台里完成任务。\n")
	if currentPage != "" {
		sb.WriteString("当前用户所在页面: " + currentPage + "。\n")
	}
	sb.WriteString("\n")

	if len(activeSkillIDs) == 0 {
		sb.WriteString("请用中文回复，简洁友好。\n")
		return sb.String()
	}

	sb.WriteString("你已启用以下技能，请根据技能说明辅助用户完成对应任务：\n\n")
	for _, id := range activeSkillIDs {
		content, err := LoadSkillContent(id)
		if err != nil {
			continue
		}
		// 去掉 frontmatter，只保留正文给 LLM
		body := stripFrontmatter(content)
		sb.WriteString("## 技能：" + id + "\n")
		sb.WriteString(body)
		sb.WriteString("\n---\n\n")
	}

	// 通用填表指令（所有 skill 共用）
	sb.WriteString(`当需要自动填写页面表单时，请先简短说明你要填什么，然后在回复末尾单独写：
[FORM_FILL:页面名.表单名]
{"字段": "值", ...}

请用中文回复，简洁友好。`)
	return sb.String()
}

// stripFrontmatter 去掉 SKILL.md 开头的 YAML frontmatter（--- ... ---）
func stripFrontmatter(content string) string {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "---") {
		return content
	}
	scanner := bufio.NewScanner(strings.NewReader(trimmed))
	scanner.Scan() // skip first ---
	count := 0
	var lines []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			count++
			if count == 1 {
				continue // 跳过结束 ---
			}
		}
		if count >= 1 {
			lines = append(lines, line)
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
