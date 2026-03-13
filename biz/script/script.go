package script

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"yuuki798/env-king/infra"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

const Topic = "script"
const bucketWorkflows = "script_workflows"
const bucketJobs = "script_jobs"

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
	cancelByID sync.Map // jobID -> context.CancelFunc
)

func getQueue() *infra.JobQueue {
	queueOnce.Do(func() {
		baseDir = strings.TrimSpace(viper.GetString("script.work_dir"))
		if baseDir == "" {
			// 默认使用当前目录下的 workdir
			cwd, _ := os.Getwd()
			baseDir = filepath.Join(cwd, "workdir")
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
			storeDir := filepath.Join(baseDir, "store")
			_ = os.MkdirAll(storeDir, 0755)
			p = filepath.Join(storeDir, "script.db")
		} else {
			_ = os.MkdirAll(filepath.Dir(p), 0755)
		}
		store, err = infra.OpenStore(p, nil)
	})
	return store, err
}

func registerCancel(id string, cancel context.CancelFunc) {
	if id == "" || cancel == nil {
		return
	}
	cancelByID.Store(id, cancel)
}

func tryCancel(id string) bool {
	if id == "" {
		return false
	}
	v, ok := cancelByID.LoadAndDelete(id)
	if !ok {
		return false
	}
	if cancel, ok := v.(context.CancelFunc); ok && cancel != nil {
		cancel()
		return true
	}
	return false
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

// ListJobs 返回 script 任务列表（内存队列 + 持久化历史）。
func ListJobs() []*JobView {
	// 1. 从持久化 store 读取历史任务
	history, err := loadAllJobSnapshots()
	if err != nil {
		// 读失败时至少保证还能看到内存任务
		history = nil
	}
	jobMap := make(map[string]*JobView, len(history))
	for _, hv := range history {
		if hv == nil || hv.ID == "" {
			continue
		}
		jobMap[hv.ID] = hv
	}

	// 2. 用内存队列的最新状态覆盖（包含进行中的任务）
	q := getQueue()
	live := q.JobsByTopic(Topic)
	for i := range live {
		j := live[i]
		view := jobViewFrom(&j)
		jobMap[view.ID] = view
	}

	// 3. 输出并按时间排序（最近在前）
	out := make([]*JobView, 0, len(jobMap))
	for _, v := range jobMap {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		// 优先按 EnqueuedAt，其次 FinishedAt
		ai, aj := out[i].EnqueuedAt, out[j].EnqueuedAt
		if !ai.Equal(aj) {
			return ai.After(aj)
		}
		return out[i].FinishedAt.After(out[j].FinishedAt)
	})
	return out
}

// GetJob 按 ID 获取任务（先查内存，再查历史）。
func GetJob(id string) (*JobView, bool) {
	q := getQueue()
	if j, ok := q.Job(id); ok {
		return jobViewFrom(&j), true
	}
	// 不在内存队列里（可能是历史任务），尝试从 store 读取
	s, err := getStore()
	if err != nil || s == nil {
		return nil, false
	}
	raw, err := s.GetString(bucketJobs, id)
	if err != nil {
		return nil, false
	}
	var v JobView
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, false
	}
	if v.ID == "" {
		return nil, false
	}
	return &v, true
}

// CancelJob 通过 ID 中断一个仍在 pending/running 的 Job。
func CancelJob(id string) (*JobView, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id required")
	}
	q := getQueue()
	j, ok := q.Job(id)
	if !ok {
		return nil, fmt.Errorf("job not found")
	}
	if j.Status != infra.JobPending && j.Status != infra.JobRunning {
		return nil, fmt.Errorf("job already finished")
	}
	if !tryCancel(id) {
		return nil, fmt.Errorf("job cannot be canceled")
	}
	// 返回最新视图（可能仍然是 running/pending，前端会继续轮询直至变为 canceled）
	if jj, ok := q.Job(id); ok {
		view := jobViewFrom(&jj)
		return view, nil
	}
	if view, ok := GetJob(id); ok {
		return view, nil
	}
	return nil, fmt.Errorf("job not found")
}

// RunFlow 编排执行：按 steps 顺序执行，遇错即停。
func RunFlow(req FlowRunRequest) (*JobView, error) {
	if len(req.Steps) == 0 {
		return nil, fmt.Errorf("steps required")
	}

	meta := map[string]string{"run_type": "flow"}
	ctx, cancel := context.WithCancel(context.Background())

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
		cancel()
		return nil, err
	}

	registerCancel(id, cancel)

	persistJobWhenDone(id)

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
	ctx, cancel := context.WithCancel(context.Background())

	id, err := getQueue().Submit(Topic, meta, ctx, func(ctx context.Context, job *infra.Job) error {
		dir := workDir
		if dir == "" {
			dir = getBaseDir()
		}
		setExtra(job.ID, dir)
		if job.Meta != nil {
			job.Meta["work_dir"] = dir
		}
		safeCmd, err := sanitizeShellCommand(cmd, dir)
		if err != nil {
			return err
		}
		sh := infra.NewShell().WithDir(dir)
		ex := getExtra(job.ID)
		res := sh.RunCombinedStreaming(ctx, safeCmd, func(chunk []byte) {
			if len(chunk) > 0 && ex != nil {
				ex.appendLog(string(chunk))
			}
		})
		// 输出已流式写入，失败时不再重复追加
		if !res.Success() {
			return fmt.Errorf("%s", res.Stdout)
		}
		return nil
	})
	if err != nil {
		cancel()
		return nil, err
	}

	registerCancel(id, cancel)

	persistJobWhenDone(id)

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

// sanitizeShellCommand 对 shell 命令做安全检查：
// 1) 禁止出现 ../
// 2) 限制 rm 的目标只能落在脚本工作目录下（即 script.work_dir）
func sanitizeShellCommand(cmd, workDir string) (string, error) {
	if strings.Contains(cmd, "../") {
		return "", fmt.Errorf("安全限制：命令中禁止包含 \"../\" 路径片段")
	}

	fields := strings.Fields(strings.TrimSpace(cmd))
	if len(fields) == 0 {
		return cmd, nil
	}
	if fields[0] != "rm" {
		return cmd, nil
	}

	base := getBaseDir()

	// 遍历 rm 的非选项参数，确保删除目标在 base 之下
	for _, arg := range fields[1:] {
		if arg == "--" || strings.HasPrefix(arg, "-") {
			continue
		}
		// 为简单起见，禁止通配符，避免误删
		if strings.ContainsAny(arg, "*?") {
			return "", fmt.Errorf("安全限制：rm 命令不允许使用通配符（* 或 ?）")
		}
		target := arg
		if !filepath.IsAbs(target) {
			target = filepath.Join(workDir, target)
		}
		target = filepath.Clean(target)
		rel, err := filepath.Rel(base, target)
		if err != nil || strings.HasPrefix(rel, "..") {
			return "", fmt.Errorf("安全限制：rm 仅允许删除 %s 下的文件或目录", base)
		}
	}

	return cmd, nil
}

// persistJobWhenDone 在 Job 结束后将其快照持久化到 store。
func persistJobWhenDone(id string) {
	go func() {
		q := getQueue()
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			j, ok := q.Job(id)
			if !ok {
				// 队列中已不存在，放弃
				return
			}
			if j.Status == infra.JobPending || j.Status == infra.JobRunning {
				<-ticker.C
				continue
			}
			cancelByID.Delete(id)
			_ = saveJobSnapshot(&j)
			return
		}
	}()
}

// saveJobSnapshot 将单个 Job 的视图写入持久化存储。
func saveJobSnapshot(j *infra.Job) error {
	s, err := getStore()
	if err != nil || s == nil {
		return err
	}
	view := jobViewFrom(j)
	raw, err := json.Marshal(view)
	if err != nil {
		return err
	}
	return s.PutString(bucketJobs, view.ID, raw)
}

// loadAllJobSnapshots 读取所有已持久化的 Job 视图。
func loadAllJobSnapshots() ([]*JobView, error) {
	s, err := getStore()
	if err != nil || s == nil {
		return nil, err
	}
	keys, err := s.ListKeysString(bucketJobs)
	if err != nil {
		return nil, err
	}
	out := make([]*JobView, 0, len(keys))
	for _, k := range keys {
		id := string(k)
		raw, err := s.GetString(bucketJobs, id)
		if err != nil {
			continue
		}
		var v JobView
		if err := json.Unmarshal(raw, &v); err != nil || v.ID == "" {
			continue
		}
		out = append(out, &v)
	}
	return out, nil
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

		// 基本安全策略：禁止 ../，限制 rm 作用范围在脚本工作目录内
		safeCmd, err := sanitizeShellCommand(cmd, workDir)
		if err != nil {
			return fmt.Errorf("step %q: %w", name, err)
		}

		switch step.Kind {
		case StepKindShell:
			res := sh.RunCombinedStreaming(runCtx, safeCmd, func(chunk []byte) {
				if len(chunk) > 0 {
					log(string(chunk))
				}
			})
			if !res.Success() {
				// 输出已流式写入，仅追加失败提示
				log("\n[step failed]\n")
				return fmt.Errorf("step %q: %s", name, res.Stdout)
			}
			log("\n")
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
	ctx, cancel := context.WithCancel(context.Background())

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
		cancel()
		return nil, err
	}

	registerCancel(id, cancel)

	persistJobWhenDone(id)

	j, _ := getQueue().Job(id)
	return jobViewFrom(&j), nil
}
