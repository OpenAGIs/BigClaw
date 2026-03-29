#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)

out_args=()
args=("$@")
i=0
while [ "$i" -lt "${#args[@]}" ]; do
  arg=${args[$i]}
  case "$arg" in
    --report-file)
      i=$((i + 1))
      out_args+=(--report "${args[$i]}")
      ;;
    --report-file=*)
      out_args+=(--report "${arg#--report-file=}")
      ;;
    --no-cleanup)
      out_args+=(--cleanup=false)
      ;;
    --issues)
      i=$((i + 1))
      issues=()
      while [ "$i" -lt "${#args[@]}" ]; do
        next=${args[$i]}
        case "$next" in
          -*)
            i=$((i - 1))
            break
            ;;
          *)
            issues+=("$next")
            ;;
        esac
        i=$((i + 1))
      done
      if [ "${#issues[@]}" -gt 0 ]; then
        issue_csv=$(IFS=,; printf '%s' "${issues[*]}")
        out_args+=(--issues "$issue_csv")
      else
        out_args+=(--issues "")
      fi
      ;;
    *)
      out_args+=("$arg")
      ;;
  esac
  i=$((i + 1))
done

exec bash "$script_dir/bigclawctl" workspace validate "${out_args[@]}"
