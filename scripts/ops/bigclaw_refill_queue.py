#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repo_root=$(cd -- "$script_dir/../.." && pwd)

args=("$@")
has_local_issues=0
for arg in "${args[@]}"; do
  case "$arg" in
    --local-issues|--local-issues=*)
      has_local_issues=1
      ;;
  esac
done

if [ "$has_local_issues" -eq 0 ] && [ -f "$repo_root/local-issues.json" ]; then
  args+=(--local-issues "$repo_root/local-issues.json")
fi

exec "$script_dir/bigclawctl" refill "${args[@]}"
