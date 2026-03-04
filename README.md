# Env King

专属于开发团队的**智能环境管家与 DevOps Agent**。集成 Clash 代理、GitHub 流水线、Agent 沙盒、MCP/Skills，支持自我修改与自主部署。

## 愿景

让每个开发团队都拥有一个懂自己上下文的专属智能运维助手。

- **加速访问**：Clash 代理 + Docker 镜像测速
- **流水线自动化**：GitHub → Docker → Harbor
- **MCP & Skills**：可安装、可扩展
- **Agent 沙盒**：cron、自我修改、git、自主部署

## 核心特性

- **DevOps 流水线**：从 GitHub（含私有仓库）拉代码，Docker build 并推送到 Harbor，可统一配置
- **Clash 代理**：集成 clash-for-linux-install，git clone / docker pull 走代理加速
- **智能 Agent**：workspace 沙盒、cron 管理、自我修改日志、git commit/push、go build 部署
- **MCP & Skills**：安装/卸载、Token/模型负载均衡、长短期记忆接口
- **前端管理控制台**：Dashboard、Pipeline、Clash、Agent、Skills 全 UI 管理（`web/`）

## 项目结构

```
env-king/
├── main.go
├── cli/                  # Cobra CLI
├── initial/
├── internal/api/         # Gin API
├── biz/
│   ├── pipeline/        # GitHub → Docker → Harbor
│   ├── clash/
│   ├── agent/
│   └── skills/
├── web/                  # 管理控制台 (pnpm + Vite + React)
├── community/            # 开源社区文档站 (VitePress)
├── clash-for-linux-install/
└── config.dev.yaml
```

## 快速开始

### 配置

编辑 `config.dev.yaml`：

```yaml
github:
  token: "ghp_xxx"
harbor:
  host: "harbor.example.com"
  project: library
```

### 启动

```bash
# 后端
go run . server

# 前端控制台（另开终端）
cd web && pnpm install && pnpm dev
```

- API：`http://localhost:8080`
- 控制台：`http://localhost:3001`

### CLI

```bash
./env-king pipeline trigger --repo owner/repo --branch main
./env-king clash install   # 仅 Linux
./env-king clash on
./env-king agent chat "你好"
./env-king skills list
```

## 路线图（见 todo.md）

1. Clash + Git 拉代码 + DevOps 流水线，集成配置中心
2. Docker 镜像测速与代理设置
3. 爬虫工作流
4. 前端完全 UI 管理
5. Agent 对话 + MCP/Skill 管理
6. Agent 自我修改 + git 规范化 + 自主部署
7. 服务器指标、K8s 分布式部署与管理面板

## 社区与文档

文档站在 `community/`，使用 [VitePress](https://vitepress.dev/) 构建，与 `web/` 管理控制台分离。

```bash
cd community
pnpm install
pnpm docs:dev    # http://localhost:5173
pnpm docs:build  # 产出 .vitepress/dist
```

## 开发

- 后端：Go 1.24+，`go build -o env-king .`
- 前端：`cd web && pnpm build`
- 文档：`cd community && pnpm docs:build`

## 社区贡献

- **Issue / 需求反馈**：遇到 Bug、性能问题或新想法，先开 Issue 说明背景和复现步骤。
- **Pull Request**：一个 PR 尽量只做一类事情（修 Bug / 加特性 / 改文档），保证能 `go build`、`pnpm build` 通过。
- **代码风格**：Go 使用 `gofmt`；前端遵循现有 `eslint` / `tsconfig` 规则，命名尽量清晰、语义化。
- **文档贡献**：文档站在 `community/` 下，新增功能建议顺手补一页 Guide 或 Architecture。
- **交流方式**：
  - 交流 QQ 群：**Env King 交流群（QQ：1079845993）**。

## License

本项目使用 **MIT License** 开源协议。详细条款见仓库根目录的 `LICENSE` 文件。
