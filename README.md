# Anna Mini Notes

Local Anna App demonstrating the required platform path:

```text
iframe -> AnnaAppRuntime.connect() -> anna.storage.get/set
       -> anna.tools.invoke -> Go Executa -> sampling/createMessage
       -> host or mock sampling -> summary UI
```

No browser persistence, HTTP backend, Anna login, LLM API key, or cloud database is required.

## Structure

- `src/`: Vite + TypeScript UI, domain rules, and Anna Host API adapter.
- `executa/`: long-running Go JSON-RPC Tool.
- `fixtures/sampling.jsonl`: deterministic local sampling response.
- `scripts/`: binary packaging and binary protocol smoke test.
- `.github/workflows/release.yml`: three-platform release builder.

## Setup and UI harness

```powershell
npm install
npm run build
npm run validate
npm run dev
```

Open `http://localhost:5180`. Create a note, confirm `storage.set` in the Harness RPC log, refresh the iframe, then confirm `storage.get` returns it. Delete a note and confirm another `storage.set`.

`npm run dev` intentionally uses `anna-app dev --no-llm`. A Summarize click must show `tools.invoke` in the RPC log, then an equivalent `[-32603] ... llm_disabled` error. That is the expected UI wiring result: the tool was invoked but the harness intentionally suppresses sampling. It is not an LLM failure.

## Tool sampling test

Run this from the repository root:

```powershell
anna-app executa dev --dir .\executa --mock-sampling .\fixtures\sampling.jsonl --invoke summarize --args '{"notes":[{"id":"note-1","content":"Follow up with the client","order":1}]}' --json
```

Expected output contains the fixture summary. This command is evidence that `invoke` sent `sampling/createMessage`: the fixture is only selected for `ns: "sampling"`, `method: "createMessage"` with matching prompt text. A missing reverse RPC would instead fail or return `(mock) no fixture matched`.

## Manual JSON-RPC check

Run the source tool, then paste one JSON line at a time:

```powershell
cd executa
go run ./cmd/anna-notes-tool
```

```json
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2.0"}}
{"jsonrpc":"2.0","id":2,"method":"describe","params":{}}
{"jsonrpc":"2.0","id":3,"method":"health","params":{}}
```

`initialize` returns protocol v2 and `client_capabilities.sampling`; `describe` returns the Tool manifest; `health` returns `{ "ok": true }`. `invoke` needs a host or the mock runner above because it waits for the reverse sampling response.

## Sampling capability compatibility

The published Executa contract declares `host_capabilities: ["llm.sample"]`, as required by Anna sampling documentation. The local CLI version bundled for this project currently routes reverse sampling through its internal `llm.complete` bridge, so the Tool also declares `llm.complete` for local harness compatibility. The UI never calls `anna.llm.complete` directly.

## Binary packaging

The release artifact has this archive-root layout:

```text
bin/tool-local-anna-mini-notes[.exe]
manifest.json
```

`manifest.json` declares the matching Tool name, binary entrypoint, and executable permission. Build only the current platform locally:

```powershell
./scripts/package-executa.ps1
```

To create the Windows archive explicitly:

```powershell
./scripts/package-executa.ps1 -Platform windows-x86_64
./scripts/smoke-executa.ps1 -Binary .\dist-anna\staging-windows-x86_64\bin\tool-local-anna-mini-notes.exe
```

The script rejects cross-platform builds on the wrong host. GitHub Actions builds all required assets: `*-darwin-arm64.tar.gz`, `*-darwin-x86_64.tar.gz`, and `*-windows-x86_64.zip`.

## Release workflow

`.github/workflows/release.yml` runs on `workflow_dispatch` or a `v*` tag. It builds each native binary, sends `initialize`, `describe`, and `health` JSON-RPC lines to it as a smoke test, then uploads all three archives as GitHub Release assets. Publishing requires this project to be placed in a GitHub repository; no Anna account is needed for the local checks above.
