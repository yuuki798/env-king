# 路线图

基于项目 `todo.md` 的规划。

1. **Clash + Git + DevOps 流水线**  
   拉代码存本地，Docker build 与 push 到 Harbor，统一配置与配置中心集成。

2. **Docker 镜像测速与代理**  
   镜像测速、镜像与代理设置等。

3. **爬虫工作流**  
   支持简单爬虫任务编排。

4. **前端完全 UI 管理**  
   所有能力均可通过 Web 控制台完成（当前已覆盖 Pipeline、Clash、Agent、Skills）。

5. **Agent 对话与 MCP/Skill**  
   对话协助执行命令，MCP 与 Skill 的管理、自主安装与使用。

6. **Agent 自我修改与自主部署**  
   对话中修改代码（含自身），git 规范化版本管理，自主部署新版本 Env King。

7. **服务器指标与 K8s**  
   服务器指标展示，分布式部署（K8s）及 K8s 调度管理面板。
