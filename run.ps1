param(
    [switch]$Demo,
    [switch]$Cli,
    [switch]$History,
    [string]$Dir = ''
)
$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot
$traceCompiler = Join-Path $env:LOCALAPPDATA 'codex-cmd-trace\tools\w64devkit\bin\gcc.exe'
if (-not (Get-Command gcc -ErrorAction SilentlyContinue) -and (Test-Path -LiteralPath $traceCompiler)) {
    $env:CC = $traceCompiler
}
$env:CGO_ENABLED = '1'
$traceArgs = @()
if ($Demo) { $traceArgs += '-demo' }
if ($Cli) { $traceArgs += '-cli' }
if ($History) { $traceArgs += '-history' }
if ($Dir) { $traceArgs += @('-dir', $Dir) }
go run . @traceArgs
exit $LASTEXITCODE
