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

  if command -v python3 >/dev/null 2>&1; then
    bash "$repo_root/scripts/ops/bigclawctl" legacy-python compile-check --repo "$repo_root" --python python3 --json
    echo "BigClaw Go environment is ready, and the remaining Python compatibility shim was compile-checked from source."
  else
    echo "BigClaw Go environment is ready."
    echo "Legacy Python shim compile-check was skipped because python3 is not installed in the active environment; Go smoke and bootstrap checks still ran."
  fi
else
  echo "BigClaw Go development environment is ready."
  echo "Set BIGCLAW_ENABLE_LEGACY_PYTHON=1 to add the remaining legacy Python shim compile-check after the default Go smoke and bootstrap coverage."
fi
