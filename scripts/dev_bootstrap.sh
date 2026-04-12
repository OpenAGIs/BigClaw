#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)

(
  cd "$repo_root/bigclaw-go"
  go test ./...
)

if [ "${BIGCLAW_ENABLE_LEGACY_PYTHON:-0}" = "1" ]; then
  python3 -m venv "$repo_root/.venv"
  # shellcheck disable=SC1091
  source "$repo_root/.venv/bin/activate"
  python -m pip install -U pip
  pip install -e "$repo_root[dev]"
  python -m pytest
  echo "BigClaw Go and legacy Python migration environments are ready."
else
  echo "BigClaw Go development environment is ready."
  echo "Set BIGCLAW_ENABLE_LEGACY_PYTHON=1 to bootstrap the legacy Python migration surface too."
fi
