#!/usr/bin/env bash
set -euo pipefail

PULL=false

for arg in "$@"; do
  case "$arg" in
    --pull|-p)
      PULL=true
      ;;
    *)
      echo "Unknown argument: $arg"
      echo "Usage: scripts/preflight.sh [--pull]"
      exit 1
      ;;
  esac
done

step() {
  echo
  echo "==> $1"
}

fail() {
  echo
  echo "Preflight check failed:"
  echo "$1"
  exit 1
}

compose_cmd() {
  if docker compose version >/dev/null 2>&1; then
    echo "docker compose"
    return
  fi
  if command -v docker-compose >/dev/null 2>&1; then
    echo "docker-compose"
    return
  fi
  fail "Docker Compose is not available. Install Docker Compose v2 or docker-compose."
}

step "Checking Docker CLI"
command -v docker >/dev/null 2>&1 || fail "Docker CLI is not installed or not available in PATH."

step "Checking Docker daemon"
docker info >/dev/null 2>&1 || fail "Docker daemon is not reachable. Start Docker and run this script again."

step "Checking Docker Compose"
COMPOSE="$(compose_cmd)"
echo "Using: $COMPOSE"

step "Checking DNS resolution for Docker Hub"
if command -v getent >/dev/null 2>&1; then
  getent hosts auth.docker.io >/dev/null 2>&1 || fail "Cannot resolve auth.docker.io. Check network, VPN, proxy or DNS settings."
elif command -v nslookup >/dev/null 2>&1; then
  nslookup auth.docker.io >/dev/null 2>&1 || fail "Cannot resolve auth.docker.io. Check network, VPN, proxy or DNS settings."
elif command -v dig >/dev/null 2>&1; then
  dig auth.docker.io >/dev/null 2>&1 || fail "Cannot resolve auth.docker.io. Check network, VPN, proxy or DNS settings."
else
  echo "No DNS helper found. Skipping explicit DNS check."
fi

step "Checking Docker access to base images"
images=(
  "alpine:3.20"
  "golang:1.22-alpine"
  "postgres:16"
  "eclipse-mosquitto:2"
)

for image in "${images[@]}"; do
  echo "Checking $image"
  if docker image inspect "$image" >/dev/null 2>&1; then
    echo "  available locally"
    continue
  fi

  if [ "$PULL" = false ]; then
    echo "  not available locally"
    continue
  fi

  docker pull "$image" || fail "Docker could not pull $image. Docker cannot reach Docker Hub or authentication failed."
done

if [ "$PULL" = false ]; then
  echo
  echo "Base image check completed. Some images may not be cached locally."
  echo "Run scripts/preflight.sh --pull to download required base images before the first build."
else
  echo
  echo "All required base images are available."
fi

step "Checking Go workspace"
if [ -f "go.work" ]; then
  if command -v go >/dev/null 2>&1; then
    go work sync
  else
    echo "Go is not installed locally. Skipping go work sync."
    echo "Docker builds still work because Go runs inside the build containers."
  fi
else
  fail "go.work not found. Run this script from the repository root."
fi

step "Preflight completed"
echo "You can now run: $COMPOSE up --build"
