package script

import "time"

// StepKind 宏观节点类型，如 shell。
const StepKindShell = "shell"

// Step 流程中的单步：名称、宏观类型、以及已设置好的命令（支持 {{变量}}）。
type Step struct {
	Name    string `json:"name"`
	Kind    string `json:"kind"`    // 如 "shell"
	Command string `json:"command"` // 已设置好的命令字符串，可用 {{var}} 占位，由流程级 inputs 替换
}

// FlowRunRequest 编排一次临时流程执行的请求（不持久化）。
type FlowRunRequest struct {
	WorkDir string            `json:"workDir,omitempty"` // 空则使用 runId 子目录
	Steps   []Step            `json:"steps"`
	Inputs  map[string]string `json:"inputs,omitempty"` // 替换各 step.Command 中的 {{key}}
}

// RunCommandRequest 单条命令执行（如 workdir 下 ls、git branch）。
type RunCommandRequest struct {
	WorkDir string `json:"workDir,omitempty"`
	Command string `json:"command"`
}

// PresetNodeDef 预设节点定义：宏观 kind、展示名、默认命令模板（含 {{var}}）及建议参数。
type PresetNodeDef struct {
	Kind            string     `json:"kind"`
	Label           string     `json:"label"`
	CommandTemplate string     `json:"commandTemplate"` // 默认命令，如 "git clone {{repo}} -b {{branch}} ."
	Params          []ParamDef `json:"params"`          // 占位符对应的参数定义，便于前端做表单
}

// ParamDef 参数定义。
type ParamDef struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Type    string `json:"type"` // string
	Default string `json:"default,omitempty"`
}

// Workflow 持久化的工作流（可复用的 flow 模板）。
type Workflow struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	WorkDir       string            `json:"workDir,omitempty"`       // 默认工作目录，可被运行时覆盖
	Steps         []Step            `json:"steps"`                   // 流程内的步骤
	DefaultInputs map[string]string `json:"defaultInputs,omitempty"` // 默认 inputs，可在运行时合并覆盖
	CreatedAt     time.Time         `json:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`
}

// WorkflowRunRequest 运行已持久化 Workflow 的请求。
type WorkflowRunRequest struct {
	WorkflowID string            `json:"workflowId"`
	WorkDir    string            `json:"workDir,omitempty"` // 若非空，覆盖 Workflow 的默认 workDir
	Inputs     map[string]string `json:"inputs,omitempty"`  // 与 Workflow.DefaultInputs 合并
}

// JobView 脚本任务对外视图。
type JobView struct {
	ID         string            `json:"id"`
	Topic      string            `json:"topic"`
	Status     string            `json:"status"`
	Error      string            `json:"error,omitempty"`
	Logs       string            `json:"logs,omitempty"`
	WorkDir    string            `json:"workDir,omitempty"`
	Meta       map[string]string `json:"meta,omitempty"`
	StartedAt  time.Time         `json:"startedAt,omitempty"`
	FinishedAt time.Time         `json:"finishedAt,omitempty"`
	EnqueuedAt time.Time         `json:"enqueued_at,omitempty"`
}
