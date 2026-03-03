# MCP & Skills

管理 MCP 与 Skills 的安装、卸载，以及模型负载均衡配置。

## 目录

- MCP：`ENV_KING_SKILLS_DIR/mcp`（未设置时使用临时目录下的 `env-king-skills/mcp`）
- Skills：同根目录下的 `skills/`
- 负载配置：`load-balance.json`

## 控制台

- **安装**：输入 npm 包名或 URL，点击「安装 MCP」或「安装 Skill」
- **已安装**：列表展示，可点击删除卸载
- **负载均衡**：添加/删除模型端点（名称、URL、权重），自动持久化

## 最小实现说明

当前安装为「目录 + source 文件」的占位实现，便于扩展为真实 npm/URL 安装与 MCP 协议对接。负载均衡配置已落盘，可供后续 LLM 路由使用。
