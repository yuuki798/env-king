# 流水线（Pipeline）

从 GitHub 拉取仓库，执行 Docker 构建并推送到 Harbor。

## 配置

在 `config.dev.yaml` 中设置：

- `github.token`：GitHub Personal Access Token，用于拉取私有仓库
- `harbor.host`、`harbor.project`：Harbor 地址与项目名
- `pipeline.work_dir`：构建时的临时工作目录

## 使用方式

### 控制台

在 **Pipeline** 页面输入 `owner/repo` 和分支（默认 main），点击「触发构建」。

### CLI

```bash
./env-king pipeline trigger --repo owner/repo --branch main
```

## 流程

1. 使用配置的 token 或 SSH 克隆仓库到临时目录
2. 检测仓库根目录的 `Dockerfile`
3. 执行 `docker build`，镜像 tag 含分支与时间戳
4. 若配置了 Harbor，执行 `docker push`

构建记录在控制台「构建历史」中查看。
