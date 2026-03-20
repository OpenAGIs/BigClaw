#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)

translated=()
issues=()
while [ "$#" -gt 0 ]; do
  case "$1" in
    --issues)
      shift
      while [ "$#" -gt 0 ] && [[ "$1" != --* ]]; do
        issues+=("$1")
        shift
      done
      continue
      ;;
    --report-file)
      translated+=(--report "$2")
      shift 2
      continue
      ;;
    --no-cleanup)
      translated+=(--cleanup=false)
      shift
      continue
      ;;
    *)
      translated+=("$1")
      shift
      ;;
  esac
done

if [ "${#issues[@]}" -gt 0 ]; then
  issue_csv=$(IFS=,; printf '%s' "${issues[*]}")
  translated+=(--issues "$issue_csv")
fi

exec bash "$script_dir/bigclawctl" workspace validate "${translated[@]}"
