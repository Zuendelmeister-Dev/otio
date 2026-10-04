#!/usr/bin/env sh
# Python 3.10+ and Docker Compose or PostgreSQL client tools are required.
set -eu
if command -v python3 >/dev/null 2>&1; then
  exec python3 "$(dirname "$0")/historian.py" "$@"
elif command -v python >/dev/null 2>&1; then
  exec python "$(dirname "$0")/historian.py" "$@"
fi
printf '%s\n' 'Install Python 3.10 or newer and add it to PATH.' >&2
exit 1
