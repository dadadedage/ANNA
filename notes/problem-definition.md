# Anna App Mini Notes: 问题定义

## 1. 最终交付物

一个可提交 GitHub 链接的 Anna App 仓库：工程化 Mini Notes 前端、`manifest.json`、本地 Executa Tool、mock sampling fixture、测试与打包脚本、三平台发布的 GitHub Actions、完整 README。

## 2. 硬约束

- 必须经 Anna Runtime 使用 `anna.storage.*` 持久化笔记。
- 总结链路必须是 `anna.tools.invoke -> Executa -> sampling/createMessage`。
- Executa 必须是持续运行的 JSON-RPC 2.0 stdio 服务，stdout 仅输出协议响应。
- UI 用 `anna-app dev --no-llm` 验证；后端 sampling 用 `executa dev --mock-sampling` 单测。
- 必须可构建三平台二进制 archive，并由 Actions 上传为 GitHub Release assets。
- 不得依赖真实账号、LLM key、云端数据库或普通 HTTP 后端替代平台链路。

## 3. 隐含要求

- 各 manifest、tool identity、权限、前端调用常量必须严格一致。
- 必须提供可审计证据，证明 storage 与 sampling 链路实际发生。
- README 本身是验收的一部分，需让审阅者可复现全部流程。
- 二进制 archive 结构须符合 Anna 文档。具体细节需查官方文档，当前任务描述未展开。

## 4. 什么叫完成

审阅者能从零安装、构建、通过严格校验，启动 UI 创建/删除笔记并确认 Host storage 调用；点击总结能触发 Tool 路由并在无 LLM 模式得到预期错误；随后能借 mock fixture 独立验证 Tool 发起 sampling；能打包并检查三平台产物，Actions 能发布三类 Release assets。

## 5. 最容易翻车的风险点

- 用浏览器存储、HTTP 服务或固定文本伪造 Anna 平台链路。
- `--no-llm` 的预期 sampling 错误被误判为 UI wiring 失败。
- reverse JSON-RPC 的请求与响应共用 stdin 时处理错误。
- Tool 标识、参数 schema、权限或 manifest 字段不一致。
- archive 结构、平台 key、入口权限不符合官方二进制分发规范。
- 不确定项：Anna 官方文档中严格 manifest/协议/archive 格式及 CLI 的实际版本兼容性。
