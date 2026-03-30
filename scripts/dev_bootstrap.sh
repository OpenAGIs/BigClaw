#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)

make -C "$repo_root" test

if [ "${BIGCLAW_ENABLE_LEGACY_PYTHON:-0}" = "1" ]; then
  python3 -m venv "$repo_root/.venv"
  # shellcheck disable=SC1091
  source "$repo_root/.venv/bin/activate"
  python -m pip install -U pip
  python -m pip install pytest ruff pre-commit
  PYTHONPATH="$repo_root/src" python -m pytest
  echo "BigClaw Go environment is ready, and the legacy Python migration surface was validated without editable install."
else
  echo "BigClaw Go development environment is ready."
  echo "Set BIGCLAW_ENABLE_LEGACY_PYTHON=1 to validate the legacy Python migration surface with PYTHONPATH only."
fi
