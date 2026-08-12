# Anna App Mini Notes: 系统骨架

选定方案：Vite + TypeScript 前端，Go Executa Tool。

## 1. 模块拆分

| 模块 | 职责 | 主要 seam |
| --- | --- | --- |
| App UI | 输入、列表、删除、总结状态与结果展示 | 调用 Notes 用例，不直接触碰 Anna API |
| Anna Host Adapter | 建立 Runtime 连接，封装 storage 与 tools 调用 | `NotesGateway`、`SummaryGateway` |
| Notes 用例 | 校验、排序、创建、加载、删除 | 面向 UI 的 `NotesService` |
| Summary 用例 | 读取当前 notes、调用 summarize、管理结果 | 面向 UI 的 `SummaryService` |
| Executa RPC Server | stdio JSON-RPC 生命周期、请求/响应分发 | `initialize`、`describe`、`invoke` |
| Sampling Orchestrator | 组装 sampling 请求、关联 `invoke_id`、等待 reverse RPC 响应 | Tool 内部接口 |
| Distribution | Executa manifest、当前平台打包、release archives、CI 发布 | 构建与发布脚本 |

## 2. 目录结构

```text
.
├── manifest.json
├── package.json
├── src/
│   ├── app/                 # 页面状态与界面
│   ├── domain/              # Note、用例与校验规则
│   ├── host/                # Anna Runtime / storage / tools 适配
│   └── main.tsx
├── executa/
│   ├── cmd/anna-notes-tool/ # Go 进程入口
│   ├── internal/rpc/        # JSON-RPC stdio 与消息分发
│   ├── internal/sampling/   # reverse sampling 编排
│   ├── manifest.json        # Executa 分发 manifest
│   └── executa.json         # 本地 Tool 启动配置
├── fixtures/
│   └── sampling.jsonl
├── scripts/
│   └── package-executa.*
├── .github/workflows/
│   └── release.yml
├── notes/
└── README.md
```

## 3. 各模块职责

- `src/app`：仅处理展示与用户操作状态；不得直接使用浏览器持久化或 HTTP 总结接口。
- `src/domain`：定义笔记顺序、空内容校验和不可变更新；不依赖 Anna Runtime。
- `src/host`：唯一允许调用 `AnnaAppRuntime.connect()`、`anna.storage.*`、`anna.tools.invoke(...)` 的前端位置。
- `executa/internal/rpc`：持续读取 stdin，stdout 只写 JSON-RPC，日志写 stderr；同时分发 host 请求及 reverse RPC 响应。
- `executa/internal/sampling`：唯一允许发送 `sampling/createMessage` 的位置；将 sampling 返回内容转换为 summary 结果。
- `scripts` 与 workflow：只负责构建、归档、smoke test 与发布，不参与应用运行逻辑。

## 4. 核心接口

前端模块接口：

```ts
type NotesGateway = {
  load(): Promise<Note[]>
  save(notes: Note[]): Promise<void>
}

type SummaryGateway = {
  summarize(input: SummarizeInput): Promise<SummaryResult>
}

type NotesService = {
  list(): Promise<Note[]>
  add(content: string): Promise<Note[]>
  remove(id: string): Promise<Note[]>
}

type SummaryService = {
  summarize(): Promise<SummaryResult>
}
```

Tool 协议入口：`initialize`、`describe`、`invoke`；`invoke` 中只处理约定的 `summarize` tool identity。Tool 内部由 `SamplingOrchestrator.CreateMessage(invokeID, notes)` 返回 sampling 的文本结果。

**待核对**：Tool name、`invoke` 参数包装形式、`initialize` v2 协商字段、sampling payload 与 metadata 的精确 schema 必须以 Anna 当前文档和示例为准。

## 5. 关键数据结构

```ts
type Note = {
  id: string
  content: string
  order: number
}

type StoredNotes = { notes: Note[] }

type SummarizeInput = { notes: Note[] }

type SummaryResult = {
  text: string
  invokeId?: string
}
```

- `order` 是添加顺序的唯一依据，创建时单调递增；删除不改变其余笔记的既有顺序。
- storage 的单一值为 `StoredNotes`，读取、创建、删除均经 Host storage adapter。
- Tool 输入只接受总结所需的 notes 快照；Tool 不拥有笔记存储权限或状态。

## 6. 主链路时序

```text
创建 / 删除：
UI -> NotesService -> NotesGateway -> Anna Runtime -> anna.storage.get/set -> UI

总结：
UI -> SummaryService -> NotesService.list -> anna.storage.get
   -> SummaryGateway -> anna.tools.invoke
   -> Executa invoke -> sampling/createMessage
   -> Host LLM 或 mock fixture -> reverse RPC response
   -> Executa response -> anna.tools.invoke response -> UI
```

`--no-llm` UI harness 路径在 `anna.tools.invoke` 后允许返回预期的 LLM 禁用错误；这验证前端路由，不取代 Tool 的 mock-sampling 验证。

## 7. 权限、隔离和边界

- 前端权限只声明并使用 Anna 的 storage 与 tools Host API；不接触 LLM Host API，也不创建自建 HTTP 后端。
- iframe UI 与 host 的唯一通信路径是 Anna Runtime；笔记不写 `localStorage`、IndexedDB 或文件系统。
- Tool 与 host 的唯一通信路径是 stdio JSON-RPC；Tool 不直接调用外部 LLM、数据库或浏览器能力。
- Tool 的 stdout 是协议通道，所有诊断输出必须隔离到 stderr。
- Executa manifest、App `required_executas`、`ui.host_api.tools`、前端常量、`describe` 裸 manifest 必须由同一套已核对的身份定义保持一致。
- mock fixture 只隔离后端 sampling 测试；不得成为 UI 总结结果的替代来源。

**边界未稳**：Anna manifest 的完整字段、storage key 的命名/作用域、Executa binary archive 的根目录与权限规则、以及 CLI 版本兼容矩阵，需要在编码前依据官方文档和示例锁定。
