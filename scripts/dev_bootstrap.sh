#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)

(
  cd "$repo_root/bigclaw-go"
  go test ./cmd/bigclawctl
)

bash "$repo_root/scripts/ops/bigclawctl" dev-smoke
(
  cd "$repo_root/bigclaw-go"
  go test ./internal/bootstrap
)

echo "BigClaw Go development environment is ready."
