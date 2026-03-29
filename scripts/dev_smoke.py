#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repo_root=$(cd -- "$script_dir/.." && pwd)

printf '%s\n' "scripts/dev_smoke.py is a legacy wrapper; use bash scripts/ops/bigclawctl dev-smoke." >&2
exec bash "$repo_root/scripts/ops/bigclawctl" dev-smoke "$@"
