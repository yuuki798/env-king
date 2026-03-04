package script

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"yuuki798/env-king/infra"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

const Topic = "script"
const bucketWorkflows = "script_workflows"

type runExtra struct {
	logs    string
	workDir string
	mu      sync.Mutex
}

var (
	queueOnce sync.Once
	storeOnce sync.Once
	queue     *infra.JobQueue
	store     *infra.Store
	baseDir   string
	extraByID sync.Map
)

func getQueue() *infra.JobQueue {
	queueOnce.Do(func() {
		baseDir = strings.TrimSpace(viper.GetString("script.work_dir"))
		if baseDir == "" {
			baseDir = filepath.Join(os.TempDir(), "env-king-script")
		}
		_ = os.MkdirAll(baseDir, 0755)
		queue = infra.NewJobQueue(4, 100)
	})
	return queue
}

func getBaseDir() string {
	getQueue()
	return baseDir
}

func getStore() (*infra.Store, error) {
	getQueue()
	var err error
	storeOnce.Do(func() {
		p := strings.TrimSpace(viper.GetString("script.store_path"))
		if p == "" {
			p = filepath.Join(baseDir, "script.db")
		}
		store, err = infra.OpenStore(p, nil)
	})
	return store, err
}

func setExtra(id, workDir string) {
	extraByID.Store(id, &runExtra{workDir: workDir})
}

func getExtra(id string) *runExtra {
	v, _ := extraByID.Load(id)
	if v == nil {
		return nil
	}
	return v.(*runExtra)
}

func (e *runExtra) appendLog(s string) {
	e.mu.Lock()
	e.logs += s
	e.mu.Unlock()
}

func (e *runExtra) view() (logs, workDir string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.logs, e.workDir
}

func jobViewFrom(j *infra.Job) *JobView {
	v := &JobView{
		ID:         j.ID,
		Topic:      j.Topic,
		Status:     string(j.Status),
		Error:      j.Error,
		StartedAt:  j.StartedAt,
		FinishedAt: j.FinishedAt,
		EnqueuedAt: j.EnqueuedAt,
		Meta:       j.Meta,
	}
	if ex := getExtra(j.ID); ex != nil {
		v.Logs, v.WorkDir = ex.view()
	}
	if j.Meta != nil && v.WorkDir == "" {
		v.WorkDir = j.Meta["work_dir"]
	}
	return v
}

// Presets 返回预设节点定义（宏观 kind + 默认命令模板），供前端编排。
func Presets() []PresetNodeDef {
	return []PresetNodeDef{
		{
			Kind:            StepKindShell,
			Label:           "拉取仓库",
			CommandTemplate: "git clone {{repo}} -b {{branch}} .",
			Params: []ParamDef{
				{Key: "repo", Label: "仓库地址", Type: "string"},
				{Key: "branch", Label: "分支", Type: "string", Default: "master"},
			},
		},
		{
			Kind:            StepKindShell,
			Label:           "打包镜像",
			CommandTemplate: "docker build -f {{dockerfile}} -t {{tag}} .",
			Params: []ParamDef{
				{Key: "tag", Label: "镜像标签", Type: "string"},
				{Key: "dockerfile", Label: "Dockerfile 路径", Type: "string", Default: "Dockerfile"},
			},
		},
		{
			Kind:            StepKindShell,
			Label:           "推送镜像",
			CommandTemplate: "docker push {{tag}}",
			Params: []ParamDef{
				{Key: "tag", Label: "镜像标签", Type: "string"},
			},
		},
		{
			Kind:            StepKindShell,
			Label:           "自定义脚本",
			CommandTemplate: "{{script}}",
			Params: []ParamDef{
				{Key: "script", Label: "脚本内容", Type: "string"},
			},
		},
	}
}

// ListJobs 返回 script 任务列表。
func ListJobs() []*JobView {
	q := getQueue()
	jobs := q.JobsByTopic(Topic)
	out := make([]*JobView, 0, len(jobs))
	for i := range jobs {
		out = append(out, jobViewFrom(&jobs[i]))
	}
	return out
}

// GetJob 按 ID 获取任务。
func GetJob(id string) (*JobView, bool) {
	q := getQueue()
	j, ok := q.Job(id)
	if !ok {
		return nil, false
	}
	return jobViewFrom(&j), true
}

// RunFlow 编排执行：按 steps 顺序执行，遇错即停。
func RunFlow(req FlowRunRequest) (*JobView, error) {
	if len(req.Steps) == 0 {
		return nil, fmt.Errorf("steps required")
	}

	meta := map[string]string{"run_type": "flow"}
	ctx := context.Background()

	id, err := getQueue().Submit(Topic, meta, ctx, func(ctx context.Context, job *infra.Job) error {
		workDir := req.WorkDir
		if workDir == "" {
			workDir = filepath.Join(getBaseDir(), job.ID)
			_ = os.MkdirAll(workDir, 0755)
		}
		setExtra(job.ID, workDir)
		if job.Meta != nil {
			job.Meta["work_dir"] = workDir
		}
		return executeSteps(ctx, job, req.Steps, req.Inputs, workDir)
	})
	if err != nil {
		return nil, err
	}

	j, _ := getQueue().Job(id)
	return jobViewFrom(&j), nil
}

// RunCommand 执行单条命令（如 ls、cd xx && git branch），用于快速操作 workdir。
func RunCommand(req RunCommandRequest) (*JobView, error) {
	cmd := strings.TrimSpace(req.Command)
	if cmd == "" {
		return nil, fmt.Errorf("command required")
	}

	workDir := strings.TrimSpace(req.WorkDir)
	meta := map[string]string{"run_type": "command", "command": cmd}
	if workDir != "" {
		meta["work_dir"] = workDir
	}
	ctx := context.Background()

	id, err := getQueue().Submit(Topic, meta, ctx, func(ctx context.Context, job *infra.Job) error {
		dir := workDir
		if dir == "" {
			dir = getBaseDir()
		}
		setExtra(job.ID, dir)
		if job.Meta != nil {
			job.Meta["work_dir"] = dir
		}
		sh := infra.NewShell().WithDir(dir)
		res := sh.RunCombined(ctx, cmd)
		ex := getExtra(job.ID)
		if ex != nil {
			ex.appendLog(res.Stdout)
			if res.Stderr != "" {
				ex.appendLog("ERROR: " + res.Stderr + "\n")
			}
		}
		if !res.Success() {
			return fmt.Errorf("%s", res.Stderr)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	j, _ := getQueue().Job(id)
	return jobViewFrom(&j), nil
}

func substituteTemplate(body string, values map[string]string) string {
	for k, v := range values {
		body = strings.ReplaceAll(body, "{{"+k+"}}", v)
		body = strings.ReplaceAll(body, "{{ "+k+" }}", v)
	}
	return body
}

func executeSteps(ctx context.Context, job *infra.Job, steps []Step, inputs map[string]string, workDir string) error {
	ex := getExtra(job.ID)
	log := func(s string) {
		if ex != nil {
			ex.appendLog(s)
		}
	}

	runCtx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()

	sh := infra.NewShell().WithDir(workDir)

	for i, step := range steps {
		name := step.Name
		if name == "" {
			name = step.Kind
		}
		log(fmt.Sprintf("\n== step %d: %s (%s) ==\n", i+1, name, step.Kind))

		cmd := substituteTemplate(step.Command, inputs)
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("step %q: command empty after substitution", name)
		}

		switch step.Kind {
		case StepKindShell:
			res := sh.RunCombined(runCtx, cmd)
			log(res.Stdout + "\n")
			if !res.Success() {
				if res.Stderr != "" {
					log("ERROR: " + res.Stderr + "\n")
				}
				return fmt.Errorf("step %q: %s", name, res.Stderr)
			}
		default:
			return fmt.Errorf("unknown step kind: %s", step.Kind)
		}
	}

	return nil
}

// --- Workflows (CRUD, Store) ---

// ListWorkflows 返回所有持久化的 Workflow。
func ListWorkflows() ([]Workflow, error) {
	s, err := getStore()
	if err != nil {
		return nil, err
	}
	keys, err := s.ListKeysString(bucketWorkflows)
	if err != nil {
		return nil, err
	}
	var list []Workflow
	for _, k := range keys {
		b, err := s.GetString(bucketWorkflows, string(k))
		if err != nil {
			continue
		}
		var wf Workflow
		if json.Unmarshal(b, &wf) != nil {
			continue
		}
		list = append(list, wf)
	}
	return list, nil
}

// GetWorkflow 获取单个 Workflow。
func GetWorkflow(id string) (*Workflow, error) {
	s, err := getStore()
	if err != nil {
		return nil, err
	}
	b, err := s.GetString(bucketWorkflows, id)
	if err != nil {
		return nil, err
	}
	var wf Workflow
	if err := json.Unmarshal(b, &wf); err != nil {
		return nil, err
	}
	return &wf, nil
}

// CreateWorkflow 创建 Workflow。
func CreateWorkflow(wf *Workflow) (*Workflow, error) {
	s, err := getStore()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if wf.ID == "" {
		wf.ID = uuid.NewString()
	}
	wf.CreatedAt = now
	wf.UpdatedAt = now
	b, _ := json.Marshal(wf)
	if err := s.PutString(bucketWorkflows, wf.ID, b); err != nil {
		return nil, err
	}
	return wf, nil
}

// UpdateWorkflow 更新 Workflow。
func UpdateWorkflow(wf *Workflow) error {
	s, err := getStore()
	if err != nil {
		return err
	}
	if wf.ID == "" {
		return fmt.Errorf("id required")
	}
	_, err = s.GetString(bucketWorkflows, wf.ID)
	if err != nil {
		return err
	}
	wf.UpdatedAt = time.Now()
	b, _ := json.Marshal(wf)
	return s.PutString(bucketWorkflows, wf.ID, b)
}

// DeleteWorkflow 删除 Workflow。
func DeleteWorkflow(id string) error {
	s, err := getStore()
	if err != nil {
		return err
	}
	return s.DeleteString(bucketWorkflows, id)
}

// InitPresetWorkflows 初始化若干预设 Workflow（幂等），便于开箱即用。
func InitPresetWorkflows() error {
	exists, err := ListWorkflows()
	if err != nil {
		return err
	}
	if len(exists) > 0 {
		return nil
	}

	// 示例：预设一个简单的 clone + build + push workflow
	wf := &Workflow{
		Name: "Clone, Build and Push",
		Steps: []Step{
			{
				Name:    "Clone repo",
				Kind:    StepKindShell,
				Command: "git clone {{repo}} -b {{branch}} .",
			},
			{
				Name:    "Build image",
				Kind:    StepKindShell,
				Command: "docker build -f {{dockerfile}} -t {{tag}} .",
			},
			{
				Name:    "Push image",
				Kind:    StepKindShell,
				Command: "docker push {{tag}}",
			},
		},
		DefaultInputs: map[string]string{
			"branch":     "master",
			"dockerfile": "Dockerfile",
		},
	}
	_, err = CreateWorkflow(wf)
	return err
}

// RunWorkflow 按 Workflow ID 加载并运行工作流。
func RunWorkflow(req WorkflowRunRequest) (*JobView, error) {
	wf, err := GetWorkflow(req.WorkflowID)
	if err != nil || wf == nil {
		return nil, fmt.Errorf("workflow not found")
	}

	// 合并 inputs：默认值 + 本次覆盖
	merged := map[string]string{}
	for k, v := range wf.DefaultInputs {
		merged[k] = v
	}
	for k, v := range req.Inputs {
		merged[k] = v
	}

	meta := map[string]string{
		"run_type":    "workflow",
		"workflow_id": wf.ID,
		"workflow":    wf.Name,
	}
	ctx := context.Background()

	id, err := getQueue().Submit(Topic, meta, ctx, func(ctx context.Context, job *infra.Job) error {
		workDir := req.WorkDir
		if workDir == "" {
			if wf.WorkDir != "" {
				workDir = wf.WorkDir
			} else {
				workDir = filepath.Join(getBaseDir(), job.ID)
			}
		}
		_ = os.MkdirAll(workDir, 0755)
		setExtra(job.ID, workDir)
		if job.Meta != nil {
			job.Meta["work_dir"] = workDir
		}
		return executeSteps(ctx, job, wf.Steps, merged, workDir)
	})
	if err != nil {
		return nil, err
	}

	j, _ := getQueue().Job(id)
	return jobViewFrom(&j), nil
}
