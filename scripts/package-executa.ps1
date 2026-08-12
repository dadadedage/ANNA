param(
  [string]$OutputDir = "dist-anna",
  [string]$Platform = ""
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$meta = Get-Content (Join-Path $root "executa\executa.json") -Raw | ConvertFrom-Json
$tool = $meta.tool_id
$version = $meta.version
$go = (Get-Command go -ErrorAction Stop).Source

$os = if ($env:OS -eq "Windows_NT") { "windows" } elseif ([System.Runtime.InteropServices.RuntimeInformation]::OSDescription -match "Darwin") { "darwin" } else { "linux" }
$arch = switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant()) {
  "x64" { "x86_64" }
  "arm64" { "arm64" }
  default { throw "Unsupported host architecture" }
}
$hostPlatform = "$os-$arch"
if (-not $Platform) { $Platform = $hostPlatform }

$supported = @("darwin-arm64", "darwin-x86_64", "windows-x86_64")
if ($supported -notcontains $Platform) { throw "Unsupported platform '$Platform'. Supported: $($supported -join ', ')" }
if ($Platform -ne $hostPlatform) { throw "Cross-platform builds are not supported locally. Current host is '$hostPlatform'." }
$targetOs, $targetArch = $Platform -split "-", 2
$stage = Join-Path $root (Join-Path $OutputDir "staging-$Platform")
$archive = Join-Path $root (Join-Path $OutputDir "$tool-$Platform.$(if ($targetOs -eq 'windows') { 'zip' } else { 'tar.gz' })")
Remove-Item $stage -Recurse -Force -ErrorAction SilentlyContinue
New-Item (Join-Path $stage "bin") -ItemType Directory -Force | Out-Null
New-Item (Split-Path $archive) -ItemType Directory -Force | Out-Null

$binary = Join-Path $stage "bin/$tool$(if ($targetOs -eq 'windows') { '.exe' } else { '' })"
$env:GOOS = $targetOs
$env:GOARCH = if ($targetArch -eq "x86_64") { "amd64" } else { "arm64" }
$env:CGO_ENABLED = "0"
Push-Location (Join-Path $root "executa")
try { & $go build -trimpath -ldflags "-s -w" -o $binary ./cmd/anna-notes-tool }
finally { Pop-Location }

$archiveManifest = @{
  name = $tool; display_name = $meta.display_name; version = $version
  description = $meta.description
  host_capabilities = @("llm.sample")
  runtime = @{ binary = @{ entrypoint = "bin/$tool$(if ($targetOs -eq 'windows') { '.exe' } else { '' })"; permissions = @{ "bin/$tool$(if ($targetOs -eq 'windows') { '.exe' } else { '' })" = "0o755" } } }
} | ConvertTo-Json -Depth 8
$archiveManifest | Set-Content (Join-Path $stage "manifest.json") -Encoding utf8NoBOM

if (Test-Path $archive) { Remove-Item $archive -Force }
if ($targetOs -eq "windows") { Compress-Archive -Path (Join-Path $stage "*") -DestinationPath $archive }
else { tar -czf $archive -C $stage . }
Write-Output $archive
