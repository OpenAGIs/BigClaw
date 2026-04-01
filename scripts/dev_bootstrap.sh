#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)

(
  cd "$repo_root/bigclaw-go"
  go test ./cmd/bigclawctl
)

if [ "${BIGCLAW_ENABLE_LEGACY_PYTHON:-0}" = "1" ]; then
  bash "$repo_root/scripts/ops/bigclawctl" dev-smoke
  (
    cd "$repo_root/bigclaw-go"
    go test ./internal/bootstrap
  )

  if python3 -m pytest --version >/dev/null 2>&1; then
    PYTHONPATH="$repo_root/src" python3 -m pytest "$repo_root/tests/test_planning.py"
    echo "BigClaw Go environment is ready, and the remaining Python migration smoke suite was validated from source."
  else
    echo "BigClaw Go environment is ready."
    echo "Source-level Python migration validation was skipped because pytest is not installed in the active environment; Go smoke and bootstrap checks still ran."
  fi
else
  echo "BigClaw Go development environment is ready."
  echo "Set BIGCLAW_ENABLE_LEGACY_PYTHON=1 to add the remaining source-level Python migration suite after the default Go smoke and bootstrap coverage."
fi
