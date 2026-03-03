package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
)

type State struct {
	WorkspaceDir string   `json:"workspaceDir"`
	CronJobs     []string `json:"cronJobs"`
	LastActivity string   `json:"lastActivity,omitempty"`
}

var (
	lastActivity time.Time
	mu           sync.Mutex
)

func GetState() State {
	wd := resolveWorkspaceDir()
	_ = ensureAgentDirs(wd)
	cronDir := filepath.Join(wd, "cron")
	crons := listCronJobs(cronDir)
	mu.Lock()
	la := ""
	if !lastActivity.IsZero() {
		la = lastActivity.Format(time.RFC3339)
	}
	mu.Unlock()
	return State{WorkspaceDir: wd, CronJobs: crons, LastActivity: la}
}

func resolveWorkspaceDir() string {
	wd := strings.TrimSpace(viper.GetString("agent.workspace_dir"))
	if wd == "" {
		cwd, _ := os.Getwd()
		wd = filepath.Join(cwd, ".env-king-agent")
	}
	if !filepath.IsAbs(wd) {
		cwd, _ := os.Getwd()
		wd = filepath.Join(cwd, wd)
	}
	return wd
}

func ensureAgentDirs(wd string) error {
	for _, d := range []string{"", "cron", "logs"} {
		if err := os.MkdirAll(filepath.Join(wd, d), 0o755); err != nil {
			return err
		}
	}
	return nil
}

func listCronJobs(cronDir string) []string {
	crons := []string{}
	if d, err := os.ReadDir(cronDir); err == nil {
		for _, e := range d {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
				crons = append(crons, strings.TrimSuffix(e.Name(), ".json"))
			}
		}
	}
	sort.Strings(crons)
	return crons
}

// Chat 接入 LLM，支持沙盒内执行、自我修改、git push、部署
func Chat(msg string) (string, error) {
	mu.Lock()
	lastActivity = time.Now()
	mu.Unlock()
	if strings.TrimSpace(msg) == "" {
		return "请提供消息内容。", nil
	}
	return "Agent 功能开发中：已接收消息，后续将接入 LLM 执行链路。", nil
}

// CronAdd 添加 cron 定时任务（最小可用：落盘保存）
func CronAdd(spec, script string) error {
	if strings.TrimSpace(spec) == "" || strings.TrimSpace(script) == "" {
		return errors.New("spec 和 script 不能为空")
	}
	wd := resolveWorkspaceDir()
	if err := ensureAgentDirs(wd); err != nil {
		return err
	}
	name := fmt.Sprintf("job-%d", time.Now().UnixNano())
	job := CronJob{Name: name, Spec: strings.TrimSpace(spec), Script: script}
	raw, _ := json.MarshalIndent(job, "", "  ")
	return os.WriteFile(filepath.Join(wd, "cron", name+".json"), raw, 0o644)
}

// CronRemove 移除 cron 任务
func CronRemove(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("name 不能为空")
	}
	wd := resolveWorkspaceDir()
	return os.Remove(filepath.Join(wd, "cron", name+".json"))
}

// CronList 列出所有 cron 任务
func CronList() ([]CronJob, error) {
	wd := resolveWorkspaceDir()
	if err := ensureAgentDirs(wd); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(filepath.Join(wd, "cron"))
	if err != nil {
		return nil, err
	}
	out := make([]CronJob, 0)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(wd, "cron", e.Name()))
		if err != nil {
			continue
		}
		var job CronJob
		if err := json.Unmarshal(raw, &job); err != nil {
			continue
		}
		out = append(out, job)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

type CronJob struct {
	Name   string `json:"name"`
	Spec   string `json:"spec"`
	Script string `json:"script"`
}

// SelfModify 自我修改：记录补丁到日志文件
func SelfModify(diff string) error {
	diff = strings.TrimSpace(diff)
	if diff == "" {
		return errors.New("diff 不能为空")
	}
	wd := resolveWorkspaceDir()
	if err := ensureAgentDirs(wd); err != nil {
		return err
	}
	logPath := filepath.Join(wd, "logs", "self-modify.log")
	entry := fmt.Sprintf("\n[%s]\n%s\n", time.Now().Format(time.RFC3339), diff)
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(entry)
	return err
}

func ensureGitRepo(wd string) error {
	cmd := exec.Command("git", "-C", wd, "rev-parse", "--is-inside-work-tree")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("不是 git 仓库: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// GitCommit 在 workspace 执行 git commit
func GitCommit(msg string) error {
	wd := resolveWorkspaceDir()
	if err := ensureGitRepo(wd); err != nil {
		return err
	}
	if strings.TrimSpace(msg) == "" {
		msg = "agent: update workspace"
	}
	if out, err := exec.Command("git", "-C", wd, "add", ".").CombinedOutput(); err != nil {
		return fmt.Errorf("git add 失败: %s", strings.TrimSpace(string(out)))
	}
	if out, err := exec.Command("git", "-C", wd, "commit", "-m", msg).CombinedOutput(); err != nil {
		return fmt.Errorf("git commit 失败: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// GitPush 推送到远程
func GitPush() error {
	wd := resolveWorkspaceDir()
	if err := ensureGitRepo(wd); err != nil {
		return err
	}
	if out, err := exec.Command("git", "-C", wd, "push").CombinedOutput(); err != nil {
		return fmt.Errorf("git push 失败: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// Deploy 打包并部署新 env-king（最小可用：执行 go build）
func Deploy() error {
	wd := resolveWorkspaceDir()
	cmd := exec.Command("go", "build", "-o", "env-king", ".")
	cmd.Dir = wd
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("deploy(build) 失败: %s", strings.TrimSpace(string(out)))
	}
	return nil
}
