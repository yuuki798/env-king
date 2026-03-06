# Env King

Env King 是面向开发团队的**智能环境管家与 DevOps Agent**。

## 能做什么

- **Mihomo 代理 + Git**：拉代码走代理加速，嵌入 mihomo 内核（子进程 + 配置托管）
- **流水线**：GitHub（含私有仓库）→ Docker build → 推送到 Harbor
- **智能 Agent**：沙盒内 cron、自我修改、git commit/push、自主部署
- **MCP & Skills**：安装/卸载、模型负载均衡、长短期记忆

## 快速开始

1. 克隆仓库，进入项目根目录
2. 配置 `config.dev.yaml`（GitHub token、Harbor 等）
3. 启动后端：`go run . server`
4. 启动控制台：`cd web && pnpm install && pnpm dev`
5. 访问 `http://localhost:3001`

详细步骤见 [快速开始](/guide/getting-started)。

## 文档导航

- [指南](/guide/getting-started) — 安装、配置、流水线、Agent、Skills
- [架构](/architecture/overview) — 模块与 API 总览
- [路线图](/roadmap) — 规划中的能力
