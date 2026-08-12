# Anna Mini Notes

一个本地运行的 Anna App：创建、查看、删除笔记，并通过本地 Go Executa Tool 请求 Host sampling 生成摘要。

本项目不需要 Anna 登录、真实 LLM API key、云端 APS 或自建 HTTP 后端。开发时由 `anna-app dev` 提供本地 Harness；笔记使用其 legacy in-memory runtime state。

## Architecture

```text
Anna App iframe
  -> AnnaAppRuntime.connect()
  -> anna.storage.get / anna.storage.set
  -> anna.tools.invoke
  -> local Go Executa Tool (JSON-RPC over stdio)
  -> reverse RPC: sampling/createMessage
  -> Host LLM or local mock sampling fixture
  -> summary returned to the iframe
```

The iframe never calls a model API directly. The Executa Tool never has an API key and never generates a local rule-based summary.

## Project structure

```text
.
├── manifest.json                 # Anna App manifest and Host API grants
├── package.json                  # Node dependencies and local commands
├── src/
│   ├── app/app.ts                # UI rendering and user interactions
│   ├── domain/notes.ts           # Note validation, creation, ordering, deletion
│   ├── host/anna.ts              # Anna Runtime, storage, and Tool adapters
│   ├── main.ts                   # Application bootstrap
│   └── style.css
├── executa/
│   ├── cmd/anna-notes-tool/      # Go JSON-RPC Tool entrypoint
│   ├── executa.json              # Local Tool identity and launch command
│   └── go.mod
├── fixtures/sampling.jsonl       # Offline sampling/createMessage fixture
├── scripts/
│   ├── package-executa.ps1       # Build current-platform binary archive
│   └── smoke-executa.ps1         # Binary JSON-RPC smoke test
└── .github/workflows/release.yml # Three-platform GitHub Release workflow
```

## Prerequisites

- Node.js 20 or later.
- Go 1.23 or later.
- Anna CLI available as `anna-app` (the repository installs `@anna-ai/cli` locally, so `npx anna-app ...` also works).
- PowerShell 7 for the local binary packaging script on Windows.

No `anna-app login`, LLM API key, or database setup is required.

## Install and build

```powershell
npm install
npm run build
npm run validate
```

`npm run build` writes the static Anna bundle to `bundle/`. `manifest.json` declares `ui.bundle.entry: "index.html"`, so Anna loads `bundle/index.html` and its generated relative JS/CSS assets.

`npm run validate` runs:

```powershell
anna-app validate --strict
```

## UI Harness: notes and Tool wiring

Start the local Harness:

```powershell
npm run dev
```

Open `http://localhost:5180`.

### Notes acceptance steps

1. Enter a non-empty note and click **Save note**.
2. Confirm the input clears and the list shows its order and content.
3. In the Harness RPC Log, confirm a `storage.set` request with key `mini-notes.notes`.
4. Refresh the iframe page. Confirm a `storage.get` request and the note is loaded.
5. Click **Delete** for the note. Confirm the list updates immediately and the RPC Log records another `storage.set`.
6. Try saving an empty note. Confirm it is rejected and no note is added.

The source path is `NotesGateway` in `src/host/anna.ts`, which only uses `anna.storage.get` and `anna.storage.set`. It does not use `localStorage`, IndexedDB, the browser filesystem, or an HTTP backend.

### Summarize acceptance in `--no-llm` mode

`npm run dev` deliberately launches:

```powershell
anna-app dev --no-llm ...
```

With at least one saved note, click **Summarize**. The Harness RPC Log must show:

```text
storage.get -> tools.invoke
```

The UI then displays an error equivalent to:

```text
[-32603] sampling failed: harness started with --no-llm
```

This is the expected UI Harness result, not a Tool failure. It proves the browser called `anna.tools.invoke`, the local Executa Tool handled `summarize`, and the Tool reached the Host sampling boundary. The Harness intentionally rejects the actual LLM request because `--no-llm` is enabled.

The local Harness uses legacy in-memory runtime state. Notes are expected to exist within the active Harness session, but persistence after restarting the outer `anna-app dev` process is not required.

## Offline sampling test

Test the Tool's actual reverse sampling path without login or an LLM. On Windows PowerShell 5, use this command from the repository root:

```powershell
.\scripts\test-mock-sampling.cmd
```

The script passes the JSON argument through `cmd.exe`, avoiding the quote-stripping behavior of Windows PowerShell 5. It also adds the standard Go installation path when present.

In PowerShell 7, the equivalent direct command is:

```powershell
$PSNativeCommandArgumentPassing = 'Standard'
$argsJson = '{"notes":[{"id":"note-1","content":"Follow up with the client","order":1}]}'
npx anna-app executa dev --dir .\executa --mock-sampling .\fixtures\sampling.jsonl --invoke summarize --args $argsJson --json
```

The first line is required in PowerShell 7 when its native-command compatibility mode strips JSON quotes. It ensures the CLI receives `$argsJson` as valid JSON instead of `{notes:...}`. Windows PowerShell 5 does not support this setting; use `test-mock-sampling.cmd` instead.

Expected result:

```json
{"summary":"Follow up with the client, fix the login bug, and turn workshop ideas into an agenda."}
```

`fixtures/sampling.jsonl` matches only `ns: "sampling"` and `method: "createMessage"`, with prompt content that includes `Summarize these notes concisely`. Therefore this result is evidence that the Tool sent `sampling/createMessage`; a missing reverse RPC would fail or return `(mock) no fixture matched` instead.

This test is intentionally separate from the `--no-llm` UI Harness. The fixture is never used to fake a UI summary.

## Manual Executa JSON-RPC test

Run the Tool from its Go module directory:

```powershell
cd executa
go run ./cmd/anna-notes-tool
```

Paste these requests one line at a time:

```json
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2.0"}}
{"jsonrpc":"2.0","id":2,"method":"describe","params":{}}
{"jsonrpc":"2.0","id":3,"method":"health","params":{}}
{"jsonrpc":"2.0","id":4,"method":"shutdown","params":{}}
```

Expected behavior:

- `initialize` returns protocol `2.0` and `client_capabilities.sampling`.
- `describe` returns the bare Tool manifest with `name`, `display_name`, `version`, `description`, `host_capabilities`, `tools`, and `runtime`.
- `health` returns `{ "ok": true }`.
- `shutdown` returns `{ "ok": true }`.

To manually exercise `invoke`, it must be run under an Anna Host or the mock command above because it waits for the Host response to its reverse `sampling/createMessage` request. The one-shot mock command is the reproducible `invoke` test.

The Tool reads JSON-RPC requests continuously from stdin until EOF. It keeps one stdin reader for both Host requests and reverse-RPC responses, associates a numeric sampling request ID with the pending invocation, and writes only flushed JSON-RPC lines to stdout. Diagnostic output goes only to stderr.

## Manifest and capability model

`manifest.json` identifies the static bundle, grants `storage.get/set`, registers the required Tool, and grants `ui.host_api.llm.complete` for the local runtime's reverse-sampling bridge.

The Tool's own `describe` manifest declares `host_capabilities: ["llm.sample"]`, which is the Anna sampling contract. It also declares `llm.complete` as a compatibility capability because the current local Anna runtime maps reverse sampling onto the `llm.complete` Host API route. This does not grant the iframe direct model access: the UI only invokes the Tool.

## Package a native Executa archive

The release artifact contains an archive-root `manifest.json` and a native executable under `bin/`:

```text
bin/tool-local-anna-mini-notes[.exe]
manifest.json
```

Build only the current machine's supported platform:

```powershell
./scripts/package-executa.ps1
```

On this Windows x86_64 machine:

```powershell
./scripts/package-executa.ps1 -Platform windows-x86_64
./scripts/smoke-executa.ps1 -Binary .\dist-anna\staging-windows-x86_64\bin\tool-local-anna-mini-notes.exe
```

The archive is written to `dist-anna/`. The script detects the host platform and rejects cross-platform local builds. The archive-root manifest declares the matching Tool name, platform-appropriate entrypoint, and executable permissions.

## GitHub Release workflow

`.github/workflows/release.yml` runs when either:

- You select **Actions -> Executa binaries -> Run workflow**.
- A Git tag matching `v*` is pushed.

The workflow builds native binaries on `macos-14`, `macos-15-intel`, and `windows-latest`, then executes the binary JSON-RPC smoke test. Its Release job uploads these GitHub Release assets:

```text
tool-local-anna-mini-notes-darwin-arm64.tar.gz
tool-local-anna-mini-notes-darwin-x86_64.tar.gz
tool-local-anna-mini-notes-windows-x86_64.zip
```

Workflow artifacts are only an intermediate handoff between build jobs and the Release job; the final deliverables are GitHub Release assets.

For a versioned release:

```powershell
git tag v0.1.0
git push origin v0.1.0
```

## Relationship of the parts

- **Manifest**: describes the Anna App bundle, view, Host API grants, and required Tool identity.
- **Bundle**: the Vite-generated iframe UI loaded by Anna.
- **Executa**: a separate long-running Go process that exposes `summarize` over JSON-RPC stdio.
- **Anna storage / APS KV**: the UI's `anna.storage.*` abstraction. Local Harness testing uses legacy in-memory runtime state; a platform deployment can back the same Host API with APS KV.
- **Sampling**: the Tool asks the Host to complete a prompt through reverse `sampling/createMessage`; the Host, rather than the UI or Tool, owns model access.
- **Binary archive**: a native executable and archive-root manifest that Anna can install without requiring Go on the user's machine.
