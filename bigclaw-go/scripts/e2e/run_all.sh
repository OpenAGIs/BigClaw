#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SUMMARY_REPORT_PATH="${BIGCLAW_E2E_SUMMARY_REPORT_PATH:-docs/reports/live-validation-summary.json}"
RUN_LOCAL="${BIGCLAW_E2E_RUN_LOCAL:-1}"
RUN_KUBERNETES="${BIGCLAW_E2E_RUN_KUBERNETES:-1}"
RUN_RAY="${BIGCLAW_E2E_RUN_RAY:-1}"

LOCAL_OUT="$(mktemp -t bigclaw-local-e2e-out.XXXXXX)"
LOCAL_ERR="$(mktemp -t bigclaw-local-e2e-err.XXXXXX)"
K8S_OUT="$(mktemp -t bigclaw-k8s-e2e-out.XXXXXX)"
K8S_ERR="$(mktemp -t bigclaw-k8s-e2e-err.XXXXXX)"
RAY_OUT="$(mktemp -t bigclaw-ray-e2e-out.XXXXXX)"
RAY_ERR="$(mktemp -t bigclaw-ray-e2e-err.XXXXXX)"
trap 'rm -f "$LOCAL_OUT" "$LOCAL_ERR" "$K8S_OUT" "$K8S_ERR" "$RAY_OUT" "$RAY_ERR"' EXIT

status=0
if [[ "$RUN_LOCAL" == "1" ]]; then
  if ! BIGCLAW_QUEUE_BACKEND=sqlite \
    python3 "$ROOT/scripts/e2e/run_task_smoke.py" \
      --autostart \
      --go-root "$ROOT" \
      --executor local \
      --title "SQLite smoke" \
      --entrypoint "echo hello from sqlite" \
      --report-path docs/reports/sqlite-smoke-report.json \
      >"$LOCAL_OUT" 2>"$LOCAL_ERR"; then
    status=1
  fi
fi

pids=()
if [[ "$RUN_KUBERNETES" == "1" ]]; then
  "$ROOT/scripts/e2e/kubernetes_smoke.sh" >"$K8S_OUT" 2>"$K8S_ERR" &
  pids+=("$!")
fi
if [[ "$RUN_RAY" == "1" ]]; then
  "$ROOT/scripts/e2e/ray_smoke.sh" >"$RAY_OUT" 2>"$RAY_ERR" &
  pids+=("$!")
fi

if (( ${#pids[@]} > 0 )); then
  for pid in "${pids[@]}"; do
    if ! wait "$pid"; then
      status=1
    fi
  done
fi
if [[ "$RUN_LOCAL" == "1" && ! -s "$LOCAL_OUT" ]]; then
  status=1
fi

ROOT_PATH="$ROOT" SUMMARY_REPORT_PATH="$SUMMARY_REPORT_PATH" \
LOCAL_OUT="$LOCAL_OUT" LOCAL_ERR="$LOCAL_ERR" \
K8S_OUT="$K8S_OUT" K8S_ERR="$K8S_ERR" \
RAY_OUT="$RAY_OUT" RAY_ERR="$RAY_ERR" \
RUN_LOCAL="$RUN_LOCAL" RUN_KUBERNETES="$RUN_KUBERNETES" RUN_RAY="$RUN_RAY" \
VALIDATION_STATUS="$status" \
python3 - <<'PY'
import json
import os
import pathlib
import sys
from datetime import datetime, timezone

root = pathlib.Path(os.environ['ROOT_PATH'])
summary_path = root / os.environ['SUMMARY_REPORT_PATH']

def read_json(path_str):
    path = pathlib.Path(path_str)
    if not path.exists() or path.stat().st_size == 0:
        return None
    return json.loads(path.read_text())

def section(name, enabled, report_rel, stdout_path, stderr_path):
    payload = {
        'enabled': enabled,
        'report_path': report_rel,
        'stdout_path': stdout_path,
        'stderr_path': stderr_path,
    }
    if not enabled:
        payload['status'] = 'skipped'
        return payload
    report = read_json(root / report_rel)
    payload['report'] = report
    if report and isinstance(report, dict):
        payload['status'] = report.get('status', {}).get('state', 'unknown')
        payload['task_id'] = report.get('task', {}).get('id')
        payload['base_url'] = report.get('base_url')
        payload['state_dir'] = report.get('state_dir')
        payload['service_log'] = report.get('service_log')
    else:
        payload['status'] = 'missing_report'
    stderr = pathlib.Path(stderr_path).read_text() if pathlib.Path(stderr_path).exists() else ''
    if stderr.strip():
        payload['stderr_tail'] = stderr.strip().splitlines()[-10:]
    return payload

summary = {
    'generated_at': datetime.now(timezone.utc).isoformat(),
    'status': 'succeeded' if os.environ['VALIDATION_STATUS'] == '0' else 'failed',
    'local': section('local', os.environ['RUN_LOCAL'] == '1', 'docs/reports/sqlite-smoke-report.json', os.environ['LOCAL_OUT'], os.environ['LOCAL_ERR']),
    'kubernetes': section('kubernetes', os.environ['RUN_KUBERNETES'] == '1', 'docs/reports/kubernetes-live-smoke-report.json', os.environ['K8S_OUT'], os.environ['K8S_ERR']),
    'ray': section('ray', os.environ['RUN_RAY'] == '1', 'docs/reports/ray-live-smoke-report.json', os.environ['RAY_OUT'], os.environ['RAY_ERR']),
}
summary_path.parent.mkdir(parents=True, exist_ok=True)
summary_path.write_text(json.dumps(summary, ensure_ascii=False, indent=2) + '\n')
print(json.dumps(summary, ensure_ascii=False, indent=2))
sys.exit(0 if summary['status'] == 'succeeded' else 1)
PY
