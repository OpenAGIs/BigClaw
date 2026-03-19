#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)

args=("$@")
has_repo_url=0
has_cache_key=0
for arg in "${args[@]}"; do
  case "$arg" in
    --repo-url|--repo-url=*)
      has_repo_url=1
      ;;
    --cache-key|--cache-key=*)
      has_cache_key=1
      ;;
  esac
done

if [ "$has_repo_url" -eq 0 ]; then
  args+=(--repo-url "git@github.com:OpenAGIs/BigClaw.git")
fi
if [ "$has_cache_key" -eq 0 ]; then
  args+=(--cache-key "openagis-bigclaw")
fi

exec "$script_dir/bigclawctl" workspace "${args[@]}"
