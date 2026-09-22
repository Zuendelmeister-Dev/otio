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
  "apps/protocol-lab",
  "apps/ha-agent",
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
    if ($LASTEXITCODE -ne 0) { throw "Go coverage tests failed in $module (exit $LASTEXITCODE)." }
    go tool cover -func $profile | Tee-Object -FilePath (Join-Path $coverageDir "$name.coverage.txt")
    if ($LASTEXITCODE -ne 0) { throw "Coverage summary failed for $module (exit $LASTEXITCODE)." }
    $totalProfiles += $profile
  }
  finally {
    Pop-Location
  }
}

$merged = Join-Path $coverageDir "merged.coverprofile"
"mode: atomic" | Set-Content -Encoding ascii $merged
foreach ($profile in $totalProfiles) {
  Get-Content $profile | Select-Object -Skip 1 | Add-Content -Encoding ascii $merged
}

Push-Location $root
try {
  go tool cover -func $merged | Tee-Object -FilePath (Join-Path $coverageDir "merged.coverage.txt")
  if ($LASTEXITCODE -ne 0) { throw "Merged coverage summary failed (exit $LASTEXITCODE)." }
  go tool cover -html $merged -o (Join-Path $coverageDir "coverage.html")
  if ($LASTEXITCODE -ne 0) { throw "Coverage HTML generation failed (exit $LASTEXITCODE)." }
}
finally {
  Pop-Location
}

Write-Host ""
Write-Host "Coverage report written to coverage/coverage.html" -ForegroundColor Green
