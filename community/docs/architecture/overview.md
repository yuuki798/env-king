# 架构总览

## 模块

| 模块 | 路径 | 说明 |
|------|------|------|
| Pipeline | `biz/pipeline/` | GitHub clone → Docker build → Harbor push |
| Mihomo | `biz/mihomo/` | 代理启停、配置托管、状态（子进程） |
| Agent | `biz/agent/` | 沙盒、cron、自我修改、git、deploy |
| Skills | `biz/skills/` | MCP/Skills 安装、负载均衡、记忆 |

## API

- `/api/pipeline/jobs`、`/api/pipeline/trigger`
- `/api/mihomo/status`、`/api/mihomo/start`、`/api/mihomo/stop`、`/api/mihomo/config`、`/api/mihomo/restart`
- `/api/agent/state`、`/api/agent/chat`、`/api/agent/cron`、`/api/agent/self-modify`、`/api/agent/git/*`、`/api/agent/deploy`
- `/api/skills/mcp`、`/api/skills/list`、`/api/skills/install`、`/api/skills/load-balance`

## CLI

- `env-king server` — 启动 HTTP 服务
- `env-king pipeline trigger`
- `env-king mihomo install|on|off`
- `env-king agent chat|cron-list|deploy`
- `env-king skills list|mcp-list`

## 前端

- `web/`：pnpm + Vite + React + Tailwind + Radix UI，管理控制台，通过 proxy 访问后端 `/api`。
