$ErrorActionPreference = "Stop"

Write-Host "Running OT.io preflight checks..." -ForegroundColor Cyan
& "$PSScriptRoot\preflight.ps1" -Pull

if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}

Write-Host ""
Write-Host "Starting OT.io demo stack..." -ForegroundColor Cyan
docker compose up --build
