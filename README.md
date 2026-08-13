# Anna Mini Notes

这是一个本地运行的 Anna App：创建、查看、删除笔记，并通过本地 Go Executa Tool 请求 Host sampling 生成摘要。

项目支持三种启动模式：无登录的 `--no-llm` UI 验收、无登录的 mock 摘要、登录后的 APS KV/真实 Host LLM。三种模式都使用同一套 Anna Runtime、storage 和 Executa 链路，应用代码不持有 API Key。

## 一、平台链路

```text
Anna App iframe
  -> AnnaAppRuntime.connect()
  -> anna.storage.get / anna.storage.set
  -> anna.tools.invoke
  -> 本地 Go Executa Tool（JSON-RPC over stdio）
  -> reverse JSON-RPC sampling/createMessage
  -> Host LLM 或本地 mock sampling fixture
  -> summary 返回 iframe
```

前端不直接调用模型，也不调用自建 HTTP API。Executa Tool 不保存 API Key，也不使用本地规则伪造摘要；它依照 Sampling v2 使用 `sampling/createMessage`，传入 `content: { type: "text", text: ... }`、`maxTokens`、`includeContext: "none"` 和 `metadata.executa_invoke_id`。

## 二、项目结构

```text
.
├── manifest.json                 # Anna App manifest、权限和 Tool 依赖
├── package.json                  # Node 依赖和开发命令
├── src/
│   ├── app/app.ts                # 页面渲染和用户交互
│   ├── domain/notes.ts           # 笔记校验、创建、排序、删除
│   ├── host/anna.ts              # Anna Runtime、storage、Tool 适配器
│   ├── main.ts                   # 前端启动入口
│   └── style.css                 # 页面样式
├── executa/
│   ├── cmd/anna-notes-tool/      # Go JSON-RPC Tool 入口
│   ├── executa.json              # Tool 身份和本地启动命令
│   └── go.mod                    # Go 模块配置
├── fixtures/sampling.jsonl       # 离线 sampling/createMessage fixture
├── fixtures/harness-llm.jsonl    # 无 login 的 UI mock tools.invoke fixture
├── scripts/
│   ├── package-executa.ps1       # 当前平台二进制打包脚本
│   ├── smoke-executa.ps1         # 二进制 JSON-RPC smoke test
│   └── test-mock-sampling.cmd    # Windows PowerShell 5 兼容 sampling 测试
└── .github/workflows/release.yml # 三平台 GitHub Release workflow
```

## 三、安装依赖

需要安装 Node.js 20+、Go 1.23+、Anna CLI `anna-app`。项目也可通过本地依赖的 `npx anna-app` 使用 CLI；Windows 打包建议使用 PowerShell 7，PowerShell 5 可运行 `.cmd` 测试脚本。

```powershell
npm install
```

无登录验收不需要执行 `anna-app login`，也不需要设置模型 API Key。

## 三种启动方式

### A. 无登录、禁止 LLM

```powershell
npm run dev
```

该命令使用 `anna-app dev --no-llm`。笔记仍通过 `anna.storage.get/set` 保存到当前本地 Harness 的 legacy in-memory runtime state；同一个 Harness 会话内可以读回，但停止或重启 `anna-app dev`、重建 Harness 或刷新外层 dashboard 后不保证保留。RPC Log 应出现 `storage.get/set`。点击 Summarize 仍调用 `anna.tools.invoke`，但出现 `[-32603] harness started with --no-llm` 或等价错误是预期结果。

### B. 无登录、固定 mock 摘要

```powershell
npm run dev:mock
```

该命令使用 `--mock-llm fixtures/harness-llm.jsonl`。笔记存储行为与 A 模式相同：当前本地 Harness 会话内可读写，重启服务或外层 Harness 后不保证保留。Harness 匹配 `tool-local-anna-mini-notes/summarize` 并返回固定摘要，只用于验证 UI 结果展示，不替代后端 sampling 测试。修改 fixture 或启动参数后必须重启 Harness。

### C. 登录 Anna、APS KV 和真实 Host LLM

```powershell
npx anna-app login --host <Anna-Nexus-服务地址>
npx anna-app whoami
npm run dev:aps
```

该命令使用 `--storage aps`，把 `anna.storage.get/set` 落到每用户 APS KV；Summarize 通过 Executa reverse RPC 请求 Host LLM。真实 sampling 还必须在 Anna Admin 为该 Executa 开启 sampling grant。API Key 不写入前端、manifest 或 Executa。

## 三点五、正式 APS KV 持久化

笔记已经由 `src/host/anna.ts` 的 `NotesGateway` 通过 `anna.storage.get` / `anna.storage.set` 保存。存储后端由 Host 启动模式决定：无 login 时是本地 Harness 的 legacy in-memory runtime state；login 后使用 Anna 的每用户 APS KV。应用代码不更换 API，也不自行实现数据库。

```powershell
npx anna-app login --host <Anna-Nexus-服务地址>
npx anna-app whoami
npm run dev:aps
```

`dev:aps` 实际使用 `anna-app dev --storage aps`。此时创建、刷新 iframe、停止并重新启动开发服务后，仍会使用同一个 `mini-notes.notes` key 从 Anna Host 的 APS KV 读回笔记。不能将模型 API key 或 APS 凭据放入 `.env`、前端 bundle、`manifest.json` 或 Executa。

若出现认证或存储授权失败，先执行 `npx anna-app whoami` 确认本机登录账户；本项目不应通过 `localStorage`、IndexedDB 或自建 HTTP 服务绕过 Host API。无 login 模式的数据只属于当前本地 Harness 会话，重启服务或外层 Harness 后消失是预期行为，不应据此判断 `anna.storage.*` 未生效。

## 四、构建前端 bundle

```powershell
npm run build
```

Vite 将 `src/` 编译到 `bundle/`，生成 `bundle/index.html`、`bundle/assets/*.js`、`bundle/assets/*.css`。`manifest.json` 的 `ui.bundle.entry` 为 `index.html`，Anna 会加载 `bundle/index.html`；资源使用相对路径以适配 iframe 嵌套路由。

## 五、运行严格校验

```powershell
npm run validate
```

该脚本实际执行：

```powershell
anna-app validate --strict
```

校验 Anna App manifest、bundle 入口和声明的 Tool 依赖。

## 六、启动和验收 UI Harness

```powershell
npm run dev
```

该脚本使用：

```powershell
anna-app dev --no-llm --slug anna-mini-notes --executa "dir=./executa,tool_id=tool-local-anna-mini-notes,command=go run ./cmd/anna-notes-tool"
```

打开 `http://localhost:5180`。

### 笔记验收

1. 输入非空内容，点击 **Save note**。
2. 确认输入框清空，列表显示笔记顺序和内容。
3. 查看 Harness 右侧 RPC Log，确认出现 `storage.set`，key 为 `mini-notes.notes`。
4. 刷新 iframe，确认出现 `storage.get`，笔记被重新加载。
5. 点击 **Delete**，确认列表立即更新，并再次出现 `storage.set`。
6. 尝试保存空内容，确认保存被拒绝且不会新增笔记。

`src/host/anna.ts` 中的 `NotesGateway` 只调用 `anna.storage.get` 和 `anna.storage.set`，没有使用 `localStorage`、IndexedDB、浏览器文件系统或 HTTP 后端。

### 无登录模拟 LLM

如果需要在不登录 Anna 的情况下看到 Summarize 的模拟结果，运行：

```powershell
npm run dev:mock
```

这与官方 LLM Demo 的 mock 模式一致：Harness 使用 `fixtures/harness-llm.jsonl` 拦截 `anna.tools.invoke` 并返回固定 fixture，前端仍然走 `anna.storage.get` 和 `anna.tools.invoke`。它不调用真实模型，也不代表 Executa 已完成 sampling；后端 reverse sampling 仍用第七节的 `--mock-sampling` 单独验证。

### Summarize 验收

保存至少一条笔记后点击 **Summarize**。RPC Log 应出现：

```text
storage.get -> tools.invoke
```

随后 UI 显示类似：

```text
[-32603] sampling failed: harness started with --no-llm
```

这是正确的 Harness 验收结果：`--no-llm` 禁用了 Host LLM。它证明 UI 已通过 `anna.tools.invoke` 调用本地 Executa，并已到达 sampling 边界；这不表示前端或 Executa wiring 失败。

本地 Harness 使用 legacy in-memory runtime state：笔记在当前 Harness 会话中可以保存和读回，但刷新外层 dashboard、停止/重启 `anna-app dev` 或重建 Harness 后不保证继续保留。需要跨会话持久化时，使用 C 模式的 APS KV。

## 七、单独测试 Executa sampling

后端 sampling 必须与 UI Harness 分开测试，fixture 不用于伪造 UI 结果。

### Windows PowerShell 5 推荐方式

在仓库根目录执行：

```powershell
.\scripts\test-mock-sampling.cmd
```

也可以在 `scripts` 目录执行：

```powershell
.\test-mock-sampling.cmd
```

该脚本自动切换到仓库根目录，并调用 `anna-app executa dev --dir executa --mock-sampling fixtures\sampling.jsonl --invoke summarize ... --json`。

### PowerShell 7 直接方式

```powershell
$PSNativeCommandArgumentPassing = 'Standard'
$argsJson = '{"notes":[{"id":"note-1","content":"Follow up with the client","order":1}]}'
npx anna-app executa dev --dir .\executa --mock-sampling .\fixtures\sampling.jsonl --invoke summarize --args $argsJson --json
```

PowerShell 5 不支持 `Standard` 参数模式，应使用 `test-mock-sampling.cmd`，避免 JSON 双引号被剥掉。

预期输出：

```json
{"summary":"Follow up with the client, fix the login bug, and turn workshop ideas into an agenda."}
```

`fixtures/sampling.jsonl` 只匹配 `ns: "sampling"`、`method: "createMessage"` 和总结 prompt。得到 fixture 摘要证明 Tool 发起了 `sampling/createMessage`；若没有发起 reverse RPC，测试会失败或返回 `(mock) no fixture matched`。

## 八、手动测试 Executa JSON-RPC

进入 Go 模块目录并启动 Tool：

```powershell
cd executa
go run ./cmd/anna-notes-tool
```

逐行输入：

```json
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2.0"}}
{"jsonrpc":"2.0","id":2,"method":"describe","params":{}}
{"jsonrpc":"2.0","id":3,"method":"health","params":{}}
{"jsonrpc":"2.0","id":4,"method":"shutdown","params":{}}
```

- `initialize` 返回 protocol `2.0` 和 `client_capabilities.sampling`。
- `describe` 返回裸 Tool manifest，含 `name`、`display_name`、`version`、`description`、`host_capabilities`、`tools`、`runtime`。
- `health` 返回 `{ "ok": true }`。
- `shutdown` 返回 `{ "ok": true }`。

`invoke` 需要 Host 返回 reverse sampling response，因此使用第七节 `--mock-sampling` 命令验收 `invoke`。Tool 持续读取 stdin，普通请求和 reverse response 共用同一个 reader；每条响应写入 stdout 后立即 flush，诊断日志只写 stderr。

## 九、确认 summary 链路证据

前端 `src/host/anna.ts` 的 `SummaryGateway` 调用：

```text
anna.tools.invoke(tool-local-anna-mini-notes, summarize)
```

Go Tool 的 `invoke` 随后发出：

```text
sampling/createMessage
```

确认方式：查看 UI Harness RPC Log 的 `tools.invoke`；运行 `test-mock-sampling.cmd` 并确认 fixture 返回 summary；检查 Go Tool `sample()` 中包含 notes、prompt 和 `metadata.invoke_id`。

## 十、本机二进制打包

```powershell
.\scripts\package-executa.ps1
```

Windows x86_64 可显式执行：

```powershell
.\scripts\package-executa.ps1 -Platform windows-x86_64
.\scripts\smoke-executa.ps1 -Binary .\dist-anna\staging-windows-x86_64\bin\tool-local-anna-mini-notes.exe
```

输出目录是 `dist-anna/`。archive 结构必须是：

```text
bin/tool-local-anna-mini-notes.exe
manifest.json
```

archive 根目录 `manifest.json` 声明 Tool 名称、二进制 entrypoint 和执行权限。脚本会识别本机平台并拒绝错误平台的本地交叉构建。

## 十一、GitHub Actions Release

workflow 为 `.github/workflows/release.yml`，支持：

- GitHub Actions 页面：`Actions -> Executa binaries -> Run workflow`。
- 推送匹配 `v*` 的 Git tag 自动触发。

三平台矩阵：

```text
macos-14       -> darwin-arm64
macos-15-intel -> darwin-x86_64
windows-latest -> windows-x86_64
```

每个平台运行二进制 JSON-RPC smoke test，Release job 上传以下 GitHub Release assets：

```text
tool-local-anna-mini-notes-darwin-arm64.tar.gz
tool-local-anna-mini-notes-darwin-x86_64.tar.gz
tool-local-anna-mini-notes-windows-x86_64.zip
```

workflow artifacts 只是 build job 到 release job 的中间传递，最终交付物是 GitHub Release assets。

正式版本发布示例：

```powershell
git tag v0.1.0
git push origin v0.1.0
```

## 十二、各部分关系

- **manifest**：描述 Anna App 的 bundle、视图、Host API 授权和 required Executa 身份。
- **bundled Executa**：`app.json` 将 `mini-notes` handle 映射到本地 `executa/`；开发 Harness 与发布流程将 `bundled:mini-notes` 替换成实际 Tool ID，并生成 `bundle/anna-tool-ids.js`，前端从该运行时映射读取 ID，避免硬编码本地或发布后的 Tool ID。
- **bundle**：Vite 从 `src/` 构建出的 iframe 静态页面，由 Anna 加载。
- **Executa**：独立运行的 Go 进程，通过 JSON-RPC over stdio 提供 `summarize` Tool。
- **Anna storage / APS KV**：前端调用的 `anna.storage.*` 抽象；无登录的 `npm run dev` 使用 legacy in-memory runtime state，已登录的 `npm run dev:aps` 使用 Host 的每用户 APS KV。笔记不会写入浏览器本地存储或自建数据库。
- **sampling**：Executa 通过 reverse `sampling/createMessage` 借用 Host LLM，模型访问权属于 Host，不属于前端或 Tool。
- **binary archive**：包含原生可执行文件和根目录 manifest 的发布压缩包，安装后不要求用户安装 Go。

## 十三、完整验收顺序

```powershell
npm install
npm run build
npm run validate
npm run dev
```

然后按第六节验收 UI，按第七节验收 mock sampling，第八节验收 JSON-RPC，第十节验收本机 archive，最后在 GitHub Actions 检查三平台 Release assets。
