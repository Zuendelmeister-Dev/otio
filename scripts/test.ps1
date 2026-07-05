$ErrorActionPreference = "Stop"

Write-Host "Running Go tests for OT.io monorepo..." -ForegroundColor Cyan

$modules = @(
  "apps/presense-modbus",
  "apps/presense-opcua",
  "apps/sense",
  "apps/lense",
  "apps/dispense",
  "apps/plc4go-modbus",
  "shared/mqttx"
)
foreach ($module in $modules) {
  Write-Host ""
  Write-Host "==> $module" -ForegroundColor Yellow
  Push-Location $module
  try {
    go test ./...
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
& $nodePath shared/web/iot-ui.test.js
& $nodePath shared/web/standard-chart.test.js

Write-Host ""
Write-Host "All tests finished successfully." -ForegroundColor Green
