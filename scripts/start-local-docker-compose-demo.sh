#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
EXAMPLE_DIR="$REPO_ROOT/examples/01-local-docker-compose"

cd "$REPO_ROOT"

echo "Running OT.io preflight checks..."
bash "$SCRIPT_DIR/preflight.sh" --pull

echo
echo "Starting OT.io local demo stack..."
cd "$EXAMPLE_DIR"
docker compose up --build
