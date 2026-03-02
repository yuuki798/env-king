package agent

import (
	"os"
	"path/filepath"
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
	cronDir := filepath.Join(wd, "cron")
	crons := listCronJobs(cronDir)
	mu.Lock()
	la := lastActivity.Format(time.RFC3339)
	mu.Unlock()
	return State{
		WorkspaceDir: wd,
		CronJobs:     crons,
		LastActivity: la,
	}
}

func resolveWorkspaceDir() string {
	wd := viper.GetString("agent.workspace_dir")
	if wd == "" {
		cwd, _ := os.Getwd()
		wd = filepath.Join(cwd, ".env-king-agent")
	}
	return wd
}

func listCronJobs(cronDir string) []string {
	crons := []string{}
	if d, err := os.ReadDir(cronDir); err == nil {
		for _, e := range d {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".sh") {
				crons = append(crons, e.Name())
			}
		}
	}
	return crons
}

// Chat 接入 LLM，支持沙盒内执行、自我修改、git push、部署
func Chat(msg string) (string, error) {
	mu.Lock()
	lastActivity = time.Now()
	mu.Unlock()
	// TODO: 接入 LLM、MCP、Skills，在 workspace 目录执行命令
	return "Agent 功能开发中。", nil
}

// CronAdd 添加 cron 定时任务
func CronAdd(spec, script string) error {
	// TODO: 写入 cron 目录，注册 cron
	_ = spec
	_ = script
	return nil
}

// CronRemove 移除 cron 任务
func CronRemove(name string) error {
	// TODO
	_ = name
	return nil
}

// CronList 列出所有 cron 任务
func CronList() ([]CronJob, error) {
	// TODO: 从 cron 目录读取
	return []CronJob{}, nil
}

type CronJob struct {
	Name   string `json:"name"`
	Spec   string `json:"spec"`
	Script string `json:"script"`
}

// SelfModify 自我修改：在 workspace 内修改 env-king 自身代码
func SelfModify(diff string) error {
	// TODO: 应用 diff，或通过 LLM 生成修改
	_ = diff
	return nil
}

// GitCommit 在 workspace 执行 git commit
func GitCommit(msg string) error {
	// TODO: git add . && git commit -m msg
	_ = msg
	return nil
}

// GitPush 推送到远程
func GitPush() error {
	// TODO: git push
	return nil
}

// Deploy 打包并部署新 env-king
func Deploy() error {
	// TODO: go build && docker build && push
	return nil
}
