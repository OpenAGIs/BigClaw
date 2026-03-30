#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)

(
  cd "$repo_root/bigclaw-go"
  go test ./cmd/bigclawctl
)

if [ "${BIGCLAW_ENABLE_LEGACY_PYTHON:-0}" = "1" ]; then
  bash "$repo_root/scripts/ops/bigclawctl" dev-smoke

  if python3 -m pytest --version >/dev/null 2>&1; then
    PYTHONPATH="$repo_root/src" python3 -m pytest \
      "$repo_root/tests/test_workspace_bootstrap.py" \
      "$repo_root/tests/test_planning.py"
    echo "BigClaw Go environment is ready, and the legacy Python migration smoke suite was validated from source."
  else
    echo "BigClaw Go environment is ready."
    echo "Legacy Python migration validation was limited to bigclawctl dev-smoke because pytest is not installed in the active environment."
  fi
else
  echo "BigClaw Go development environment is ready."
  echo "Set BIGCLAW_ENABLE_LEGACY_PYTHON=1 to validate the legacy Python migration smoke suite with PYTHONPATH only."
fi
