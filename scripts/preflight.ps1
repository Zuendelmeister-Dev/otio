param(
    [switch]$Pull
)

$ErrorActionPreference = "Stop"

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    throw "Docker CLI was not found. Install Docker Desktop or Docker Engine first."
}

docker info *> $null
if ($LASTEXITCODE -ne 0) {
    throw "Docker is not running or the current user cannot access Docker."
}

docker compose version *> $null
if ($LASTEXITCODE -ne 0) {
    throw "Docker Compose plugin was not found. Install the Docker Compose plugin first."
}

if ($Pull) {
    Write-Host "Docker preflight checks passed."
}
else {
    Write-Host "Docker preflight checks passed."
}
