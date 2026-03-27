#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)

args=("$@")
if [ "${BIGCLAW_ENABLE_LEGACY_PYTHON:-0}" = "1" ]; then
  args+=(--include-legacy-python)
fi

exec bash "$repo_root/scripts/ops/bigclawctl" dev-bootstrap "${args[@]}"
