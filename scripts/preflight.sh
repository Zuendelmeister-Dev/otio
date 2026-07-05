#!/usr/bin/env bash
set -euo pipefail

while [ "$#" -gt 0 ]; do
  case "$1" in
    --pull)
      shift
      ;;
    *)
      echo "Unknown argument: $1"
      exit 1
      ;;
  esac
done

if ! command -v docker >/dev/null 2>&1; then
  echo "Docker CLI was not found. Install Docker first."
  exit 1
fi

if ! docker info >/dev/null 2>&1; then
  echo "Docker is not running or the current user cannot access Docker."
  exit 1
fi

if ! docker compose version >/dev/null 2>&1; then
  echo "Docker Compose plugin was not found. Install the Docker Compose plugin first."
  exit 1
fi

echo "Docker preflight checks passed."
