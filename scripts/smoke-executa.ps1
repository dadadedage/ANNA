param([string]$Binary)
$ErrorActionPreference = "Stop"
if (-not (Test-Path $Binary)) { throw "Binary not found: $Binary" }
$input = @(
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2.0"}}',
  '{"jsonrpc":"2.0","id":2,"method":"describe","params":{}}',
  '{"jsonrpc":"2.0","id":3,"method":"health","params":{}}'
) -join [Environment]::NewLine
$output = $input | & $Binary
if ($LASTEXITCODE -ne 0) { throw "Binary exited with code $LASTEXITCODE" }
$lines = @($output | Where-Object { $_ })
if ($lines.Count -lt 3 -or ($lines -join "`n") -notmatch '"name"\s*:\s*"tool-local-anna-mini-notes"' -or ($lines -join "`n") -notmatch '"ok"\s*:\s*true') { throw "Unexpected smoke output: $($lines -join ' | ')" }
Write-Output "smoke passed"
