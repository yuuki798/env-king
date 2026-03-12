---
name: 创建工作流
description: 帮助用户在脚本编排页面自动填写并创建工作流
pages: [script]
---

# 创建工作流

当用户描述一个想要自动化的任务时，分析其意图并自动填写「脚本编排」页面的「新建工作流」表单。

## 使用场景

- 用户说「创建一个拉取 main 分支的工作流」
- 用户说「帮我做一个构建 Docker 镜像的流程」
- 用户说「新建一个每天备份数据库的脚本」

## 填表规则

表单结构如下：
- `name`：工作流名称（简洁，中文或英文均可）
- `workDir`：工作目录，默认用 `./workdir`
- `steps`：步骤列表，每步包含：
  - `name`：步骤名
  - `kind`：固定为 `"shell"`
  - `command`：Shell 命令，变量用 `{{变量名}}` 占位
- `defaultInputs`：变量默认值，如 `{"repo": "https://github.com/xxx/yyy", "branch": "main"}`

## 示例

用户：「创建一个拉取 main 分支的工作流」

填表：
```json
{
  "name": "拉取 main 分支",
  "workDir": "./workdir",
  "steps": [{"name": "git clone", "kind": "shell", "command": "git clone {{repo}} -b {{branch}} ."}],
  "defaultInputs": {"repo": "https://github.com/user/repo", "branch": "main"}
}
```

## 回复方式

先用一句话确认要创建什么，然后在末尾输出填表标记：

[FORM_FILL:script.createWorkflow]
{"name": "...", "workDir": "./workdir", "steps": [...], "defaultInputs": {...}}
