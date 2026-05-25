param(
    [switch]$Pull
)

$ErrorActionPreference = "Stop"

function Write-Step {
    param([string]$Message)
    Write-Host ""
    Write-Host "==> $Message" -ForegroundColor Cyan
}

function Fail {
    param([string]$Message)
    Write-Host ""
    Write-Host "Preflight check failed:" -ForegroundColor Red
    Write-Host $Message -ForegroundColor Red
    exit 1
}

Write-Step "Checking Docker CLI"
try {
    docker version | Out-Null
} catch {
    Fail "Docker CLI or Docker Desktop is not available. Start Docker Desktop and run this script again."
}

Write-Step "Checking Docker daemon"
try {
    docker info | Out-Null
} catch {
    Fail "Docker daemon is not reachable. Start or restart Docker Desktop and run this script again."
}

Write-Step "Checking DNS resolution for Docker Hub"
try {
    Resolve-DnsName auth.docker.io -ErrorAction Stop | Out-Null
} catch {
    Fail "Windows cannot resolve auth.docker.io. Check your network, VPN, proxy or DNS settings."
}

Write-Step "Checking Docker access to base images"

$images = @(
    "alpine:3.20",
    "golang:1.22-alpine",
    "postgres:16",
    "eclipse-mosquitto:2"
)

foreach ($image in $images) {
    Write-Host "Checking $image"
    $inspect = docker image inspect $image 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  available locally"
        continue
    }

    if (-not $Pull) {
        Write-Host "  not available locally"
        continue
    }

    docker pull $image
    if ($LASTEXITCODE -ne 0) {
        Fail "Docker could not pull $image. This usually means Docker Desktop cannot reach Docker Hub."
    }
}

if (-not $Pull) {
    Write-Host ""
    Write-Host "Base image check completed. Some images may not be cached locally." -ForegroundColor Yellow
    Write-Host "Run scripts/preflight.ps1 -Pull to download required base images before the first build." -ForegroundColor Yellow
} else {
    Write-Host ""
    Write-Host "All required base images are available." -ForegroundColor Green
}

Write-Step "Checking Go workspace"
if (Test-Path "go.work") {
    go work sync
} else {
    Fail "go.work not found. Run this script from the repository root."
}

Write-Step "Preflight completed"
Write-Host "You can now run: docker compose up --build" -ForegroundColor Green
