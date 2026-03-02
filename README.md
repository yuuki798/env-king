# Env King

专属于开发团队的智能助理。集成 GitHub 流水线、Clash 代理加速、Agent 沙盒、MCP/Skills。

## 功能

- **构建流水线**：从 GitHub 拉取私有仓库，Docker build 并推送到 Harbor
- **Clash 代理**：集成 clash-for-linux-install，git clone 走代理加速
- **智能 Agent**：目录内 root 权限，支持 cron、自我修改、git push、打包部署（规划中）
- **MCP & Skills**：支持 MCP 安装、Skill 安装、Token/模型负载均衡、长短期记忆（规划中）

## 项目结构

```
env-king/
├── main.go              # 入口
├── cli/                  # Cobra CLI
├── initial/              # 初始化
├── internal/api/         # Web API (Gin)
├── biz/                  # 业务逻辑
│   ├── pipeline/         # GitHub → Docker → Harbor
│   ├── clash/            # Clash 代理
│   ├── agent/            # Agent 沙盒
│   └── skills/           # MCP & Skills
├── web/                  # 前端 (pnpm + Vite + React + Tailwind + Radix)
└── clash-for-linux-install/  # Clash 安装脚本
```

## 快速开始

### 1. 配置

复制并编辑 `config.dev.yaml`，配置 GitHub Token、Harbor 等：

```yaml
github:
  token: "ghp_xxx"  # GitHub Personal Access Token
harbor:
  host: "harbor.example.com"
  project: library
```

### 2. 启动服务

```bash
go run . server
# 或
./env-king server
```

API 在 `http://localhost:8080`

### 3. 启动前端（开发）

```bash
cd web && pnpm dev
```

前端在 `http://localhost:3001`，通过 Vite proxy 访问后端 `/api`。

### 4. CLI 命令

```bash
# 构建流水线
./env-king pipeline trigger --repo owner/repo --branch main

# Clash（仅 Linux）
./env-king clash install   # 安装
./env-king clash on        # 启动
./env-king clash off       # 关闭

# Agent（骨架已搭，函数填空）
./env-king agent chat "你好"
./env-king agent cron-list
./env-king agent deploy

# Skills
./env-king skills list     # 列出 Skills
./env-king skills mcp-list # 列出 MCP

# 其他
./env-king ek speed        # Docker 镜像站测速
./env-king blog ...        # 博客相关
```

## 开发

- 后端：Go 1.24+
- 前端：pnpm + Vite + React + Tailwind v4 + Radix UI

### 生产构建

```bash
# 后端
go build -o env-king .

# 前端
cd web && pnpm build
# 将 web/dist 放到后端可托管的静态目录，或通过 nginx 反向代理
```
