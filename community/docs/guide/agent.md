# 智能 Agent

Agent 在指定 **workspace** 目录下工作，支持 cron、自我修改、git、部署。

## 工作目录

默认：`./.env-king-agent`，可通过 `config.dev.yaml` 的 `agent.workspace_dir` 修改。

目录结构示例：

- `cron/` — cron 任务 JSON
- `logs/` — 自我修改等日志

## 功能

- **对话**：当前为占位回复，后续接入 LLM 与 MCP/Skills
- **Cron**：添加/删除/列表，任务以 JSON 形式保存在 `cron/` 下
- **自我修改**：提交的 diff 追加写入 `logs/self-modify.log`
- **Git**：在 workspace 内执行 `git add`、`commit`、`push`（需该目录为 git 仓库）
- **部署**：在 workspace 内执行 `go build -o env-king .`

## 控制台

在 **Agent** 页面可：发送消息、添加/删除 cron、填写 commit message 与 diff、执行 Push 与部署。

## CLI

```bash
./env-king agent chat "你好"
./env-king agent cron-list
./env-king agent deploy
```
