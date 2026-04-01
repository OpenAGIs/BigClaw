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
  echo "BigClaw Go environment is ready."
  echo "Legacy Python test sweep B removed the remaining bootstrap-linked Python migration suite; Go smoke and bootstrap checks are now the supported verification path."
else
  echo "BigClaw Go development environment is ready."
  echo "Set BIGCLAW_ENABLE_LEGACY_PYTHON=1 to include the Go bootstrap verification after the default Go smoke coverage."
fi
