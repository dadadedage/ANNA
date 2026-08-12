# Anna App Mini Notes: 方案设计

## 方案 A（已选择）

Vite + TypeScript 前端，Go Executa Tool。

### 1. 大致怎么做

前端负责 Anna Runtime、storage 与 tools 调用；Go 实现持续运行的 stdio JSON-RPC Tool、reverse sampling 和原生跨平台打包。GitHub Actions 分别在 macOS/Windows runner 构建 release archive。

### 2. 主要优点

Go 原生交叉编译和单文件二进制最适合三平台发布；stdio 并发读写与 reverse RPC 更可控。前端和 Tool 各自职责清晰，满足验收路径即可，不引入额外服务。

### 3. 主要风险

需要维护 TypeScript 与 Go 两套工程；reverse RPC 的请求关联、响应分发仍需严格按协议实现。

### 4. 实现复杂度

中等。

### 5. 可测试性

高：前端可独立跑 `anna-app dev --no-llm`；Tool 可独立以 fixture 验证 sampling；二进制可直接发送 `describe` 做 smoke test。

## 方案 B

Vite + TypeScript 前端，Node.js Executa Tool 并打包成二进制。

### 1. 大致怎么做

前端与 Tool 都用 TypeScript/JavaScript；Tool 用 Node 的 stdio 能力实现协议，再使用 Node 二进制打包方案生成平台产物。

### 2. 主要优点

语言统一，团队切换成本低；前后端共享类型、常量和协议测试辅助代码更方便。

### 3. 主要风险

Node 二进制打包、依赖收集、不同平台运行时兼容性通常比 Go 更脆弱；release archive 与 CI smoke test 容易成为主要不稳定点。

### 4. 实现复杂度

中等偏高。

### 5. 可测试性

较高，但必须额外覆盖“源码运行成功、打包二进制也成功”的差异。

## 方案 C

Vite + TypeScript 前端，Python Executa Tool + PyInstaller。

### 1. 大致怎么做

前端维持工程化 TypeScript；Python 实现 Tool 与 sampling，再由 PyInstaller 在不同 CI runner 打包。

### 2. 主要优点

JSON-RPC 和 fixture 测试开发速度快；参考仓库已有 Python sampling、PyInstaller 与多文件 binary 示例。

### 3. 主要风险

PyInstaller 必须在目标平台构建，依赖与 archive 内容更容易出现平台差异；跨平台发布链路的排障成本较高。

### 4. 实现复杂度

中等偏高。

### 5. 可测试性

功能协议测试高；最终交付物的跨平台测试中等，需要严格依赖 CI runner 验证。

## 选定结论

选择方案 A：Vite + TypeScript 前端，Go Executa Tool。

原因：它以 Go 原生交叉编译降低三平台独立二进制发布风险，同时保留 Vite + TypeScript 的稳定前端工程体验，最符合“稳定交付优先”的目标。

不选择方案 B：Node 二进制打包引入额外兼容性风险，语言统一不足以抵消发布不确定性。

不选择方案 C：Python 开发快，但三平台可复现二进制打包与分发的稳定性较弱。
