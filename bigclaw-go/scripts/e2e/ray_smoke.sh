#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
ENTRYPOINT="${BIGCLAW_RAY_SMOKE_ENTRYPOINT:-python -c \"print('hello from ray')\"}"
RUNTIME_ENV_JSON="${BIGCLAW_RAY_RUNTIME_ENV_JSON:-}"
REPORT_PATH="${BIGCLAW_RAY_SMOKE_REPORT_PATH:-docs/reports/ray-live-smoke-report.json}"
ARGS=(
  --autostart
  --go-root "$ROOT"
  --executor ray
  --title "Ray smoke test"
  --entrypoint "$ENTRYPOINT"
  --report-path "$REPORT_PATH"
)
if [[ -n "$RUNTIME_ENV_JSON" ]]; then
  ARGS+=(--runtime-env-json "$RUNTIME_ENV_JSON")
fi
go run "$ROOT/cmd/bigclawctl" automation e2e run-task-smoke "${ARGS[@]}"
