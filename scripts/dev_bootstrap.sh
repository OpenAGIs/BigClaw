#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)

(
  cd "$repo_root/bigclaw-go"
  go test ./cmd/bigclawctl
)

if [ "${BIGCLAW_ENABLE_LEGACY_PYTHON:-0}" = "1" ]; then
  bash "$repo_root/scripts/ops/bigclawctl" dev-smoke
  echo "BigClaw Go environment is ready, and the legacy Python migration smoke path was validated with bigclawctl dev-smoke."
else
  echo "BigClaw Go development environment is ready."
  echo "Set BIGCLAW_ENABLE_LEGACY_PYTHON=1 to validate the legacy Python migration smoke path."
fi
