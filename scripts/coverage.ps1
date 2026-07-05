$ErrorActionPreference = "Stop"

Write-Host "Running Go tests with coverage for OT.io monorepo..." -ForegroundColor Cyan

$root = Resolve-Path (Join-Path $PSScriptRoot "..")
$coverageDir = Join-Path $root "coverage"
New-Item -ItemType Directory -Force -Path $coverageDir | Out-Null

$modules = @(
  "apps/presense-modbus",
  "apps/presense-opcua",
  "apps/sense",
  "apps/lense",
  "apps/dispense",
  "apps/plc4go-modbus",
  "shared/mqttx"
)
$totalProfiles = @()
foreach ($module in $modules) {
  $name = $module.Replace("/", "-")
  $profile = Join-Path $coverageDir "$name.coverprofile"
  Write-Host ""
  Write-Host "==> $module" -ForegroundColor Yellow
  Push-Location (Join-Path $root $module)
  try {
    go test ./... -coverprofile $profile -covermode atomic
    go tool cover -func $profile | Tee-Object -FilePath (Join-Path $coverageDir "$name.coverage.txt")
    $totalProfiles += $profile
  }
  finally {
    Pop-Location
  }
}

$merged = Join-Path $coverageDir "merged.coverprofile"
"mode: atomic" | Set-Content $merged
foreach ($profile in $totalProfiles) {
  Get-Content $profile | Select-Object -Skip 1 | Add-Content $merged
}

go tool cover -func $merged | Tee-Object -FilePath (Join-Path $coverageDir "merged.coverage.txt")
go tool cover -html $merged -o (Join-Path $coverageDir "coverage.html")

Write-Host ""
Write-Host "Coverage report written to coverage/coverage.html" -ForegroundColor Green
