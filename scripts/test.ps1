$ErrorActionPreference = "Stop"

Write-Host "Running Go tests for OT.io monorepo..." -ForegroundColor Cyan

$modules = @(
  "apps/presense-modbus",
  "apps/presense-opcua",
  "apps/sense",
  "apps/lense",
  "apps/dispense",
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
node shared/web/iot-ui.test.js
node shared/web/standard-chart.test.js

Write-Host ""
Write-Host "All tests finished successfully." -ForegroundColor Green
