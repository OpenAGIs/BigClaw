#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)

out_args=()
has_repo_url=0
has_cache_key=0
for arg in "$@"; do
  case "$arg" in
    --repo-url|--repo-url=*)
      has_repo_url=1
      ;;
    --cache-key|--cache-key=*)
      has_cache_key=1
      ;;
  esac
  out_args+=("$arg")
done

if [ "$has_repo_url" -eq 0 ]; then
  out_args+=(--repo-url "${BIGCLAW_BOOTSTRAP_REPO_URL:-git@github.com:OpenAGIs/BigClaw.git}")
fi
if [ "$has_cache_key" -eq 0 ]; then
  out_args+=(--cache-key "${BIGCLAW_BOOTSTRAP_CACHE_KEY:-openagis-bigclaw}")
fi

exec bash "$script_dir/bigclawctl" workspace "${out_args[@]}"
