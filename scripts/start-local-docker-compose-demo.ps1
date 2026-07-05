$ErrorActionPreference = "Stop"

$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$ExampleDir = Join-Path $RepoRoot "examples\01-local-docker-compose"

Write-Host "Running OT.io preflight checks..."
Push-Location $RepoRoot
try {
    & "$PSScriptRoot\preflight.ps1" -Pull
    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }
}
finally {
    Pop-Location
}

Write-Host ""
Write-Host "Starting OT.io local demo stack..."
Push-Location $ExampleDir
try {
    docker compose up --build
    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }
}
finally {
    Pop-Location
}
