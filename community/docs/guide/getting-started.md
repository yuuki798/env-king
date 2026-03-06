# 快速开始

## 环境要求

- Go 1.24+
- Node.js 18+（用于前端与文档站）
- pnpm

## 安装与运行

```bash
# 克隆仓库后进入项目根目录
cd env-king

# 启动后端（默认 :8080）
go run . server

# 另开终端：启动管理控制台（默认 :3001）
cd web
pnpm install
pnpm dev
```

- **控制台**：http://localhost:3001  
- **API**：http://localhost:8080，前缀 `/api`

## 配置

复制或编辑 `config.dev.yaml`：

```yaml
github:
  token: "ghp_xxx"   # 拉取私有仓库用

harbor:
  host: "harbor.example.com"
  project: library

pipeline:
  work_dir: /tmp/env-king-pipeline

agent:
  workspace_dir: ./.env-king-agent
```

## 下一步

- 在控制台 **Pipeline** 页触发一次构建
- 在 **Mihomo 代理** 页查看/启停代理、编辑配置
- 在 **Agent** 页添加 cron、对话、自我修改
- 在 **Skills** 页安装 MCP/Skill、配置负载均衡
