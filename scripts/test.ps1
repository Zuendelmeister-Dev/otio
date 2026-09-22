$ErrorActionPreference = "Stop"
$root = Resolve-Path (Join-Path $PSScriptRoot "..")

Write-Host "Running Go tests for OT.io monorepo..." -ForegroundColor Cyan

$modules = @(
  "apps/presense-modbus",
  "apps/presense-opcua",
  "apps/sense",
  "apps/lense",
  "apps/dispense",
  "apps/plc4go-modbus",
  "apps/protocol-lab",
  "apps/ha-agent",
  "shared/mqttx"
)
foreach ($module in $modules) {
  Write-Host ""
  Write-Host "==> $module" -ForegroundColor Yellow
  Push-Location (Join-Path $root $module)
  try {
    go test ./...
    if ($LASTEXITCODE -ne 0) { throw "Go tests failed in $module (exit $LASTEXITCODE)." }
  }
  finally {
    Pop-Location
  }
}

Write-Host ""
Write-Host "Running shared JavaScript tests..." -ForegroundColor Yellow
$nodeCommand = Get-Command node -ErrorAction SilentlyContinue
$nodePath = $null
if ($nodeCommand) {
  $nodePath = $nodeCommand.Source
}
else {
  $bundledNode = Join-Path $env:USERPROFILE ".cache\codex-runtimes\codex-primary-runtime\dependencies\node\bin\node.exe"
  if (Test-Path $bundledNode) {
    $nodePath = $bundledNode
  }
}
if (-not $nodePath) {
  throw "Node.js was not found. Install Node.js or set it on PATH to run the shared JavaScript tests."
}
& $nodePath (Join-Path $root "shared/web/iot-ui.test.js")
if ($LASTEXITCODE -ne 0) { throw "iot-ui JavaScript tests failed (exit $LASTEXITCODE)." }
& $nodePath (Join-Path $root "shared/web/standard-chart.test.js")
if ($LASTEXITCODE -ne 0) { throw "standard-chart JavaScript tests failed (exit $LASTEXITCODE)." }
& $nodePath (Join-Path $root "scripts/kubernetes-config.test.js")
if ($LASTEXITCODE -ne 0) { throw "Kubernetes configuration tests failed (exit $LASTEXITCODE)." }

Write-Host ""
& $nodePath (Join-Path $root "shared/web/workspace-ui.test.js")
if ($LASTEXITCODE -ne 0) { throw "Workspace UI tests failed." }
Write-Host "All tests finished successfully." -ForegroundColor Green
