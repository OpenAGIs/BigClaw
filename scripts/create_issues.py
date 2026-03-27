#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)

args=("$@")
if [ "${#args[@]}" -gt 0 ] && [[ "${args[0]}" != -* ]]; then
  exec bash "$repo_root/scripts/ops/bigclawctl" issue-bootstrap sync "${args[@]}"
fi

exec bash "$repo_root/scripts/ops/bigclawctl" issue-bootstrap sync "${args[@]}"
