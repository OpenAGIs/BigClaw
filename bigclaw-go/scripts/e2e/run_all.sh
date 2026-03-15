#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SUMMARY_REPORT_PATH="${BIGCLAW_E2E_SUMMARY_REPORT_PATH:-docs/reports/live-validation-summary.json}"
INDEX_REPORT_PATH="${BIGCLAW_E2E_INDEX_PATH:-docs/reports/live-validation-index.md}"
MANIFEST_REPORT_PATH="${BIGCLAW_E2E_MANIFEST_PATH:-docs/reports/live-validation-index.json}"
ARTIFACT_ROOT_REL="${BIGCLAW_E2E_ARTIFACT_ROOT:-docs/reports/live-validation-runs}"
RUN_ID="${BIGCLAW_E2E_RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
BUNDLE_DIR_REL="${ARTIFACT_ROOT_REL}/${RUN_ID}"
RUN_LOCAL="${BIGCLAW_E2E_RUN_LOCAL:-1}"
RUN_KUBERNETES="${BIGCLAW_E2E_RUN_KUBERNETES:-1}"
RUN_RAY="${BIGCLAW_E2E_RUN_RAY:-1}"
RUN_SHARED_QUEUE="${BIGCLAW_E2E_RUN_SHARED_QUEUE:-1}"

mkdir -p "$ROOT/$BUNDLE_DIR_REL"

LOCAL_REPORT_REL="$BUNDLE_DIR_REL/sqlite-smoke-report.json"
K8S_REPORT_REL="$BUNDLE_DIR_REL/kubernetes-live-smoke-report.json"
RAY_REPORT_REL="$BUNDLE_DIR_REL/ray-live-smoke-report.json"
SHARED_QUEUE_REPORT_REL="$BUNDLE_DIR_REL/multi-node-shared-queue-report.json"

LOCAL_OUT="$(mktemp -t bigclaw-local-e2e-out.XXXXXX)"
LOCAL_ERR="$(mktemp -t bigclaw-local-e2e-err.XXXXXX)"
K8S_OUT="$(mktemp -t bigclaw-k8s-e2e-out.XXXXXX)"
K8S_ERR="$(mktemp -t bigclaw-k8s-e2e-err.XXXXXX)"
RAY_OUT="$(mktemp -t bigclaw-ray-e2e-out.XXXXXX)"
RAY_ERR="$(mktemp -t bigclaw-ray-e2e-err.XXXXXX)"
SHARED_QUEUE_OUT="$(mktemp -t bigclaw-shared-queue-e2e-out.XXXXXX)"
SHARED_QUEUE_ERR="$(mktemp -t bigclaw-shared-queue-e2e-err.XXXXXX)"
trap 'rm -f "$LOCAL_OUT" "$LOCAL_ERR" "$K8S_OUT" "$K8S_ERR" "$RAY_OUT" "$RAY_ERR" "$SHARED_QUEUE_OUT" "$SHARED_QUEUE_ERR"' EXIT

status=0
pids=()

if [[ "$RUN_LOCAL" == "1" ]]; then
  (
    BIGCLAW_QUEUE_BACKEND=sqlite \
      python3 "$ROOT/scripts/e2e/run_task_smoke.py" \
        --autostart \
        --go-root "$ROOT" \
        --executor local \
        --title "SQLite smoke" \
        --entrypoint "echo hello from sqlite" \
        --report-path "$LOCAL_REPORT_REL"
  ) >"$LOCAL_OUT" 2>"$LOCAL_ERR" &
  pids+=("$!")
fi

if [[ "$RUN_KUBERNETES" == "1" ]]; then
  BIGCLAW_KUBERNETES_SMOKE_REPORT_PATH="$K8S_REPORT_REL" \
    "$ROOT/scripts/e2e/kubernetes_smoke.sh" >"$K8S_OUT" 2>"$K8S_ERR" &
  pids+=("$!")
fi

if [[ "$RUN_RAY" == "1" ]]; then
  BIGCLAW_RAY_SMOKE_REPORT_PATH="$RAY_REPORT_REL" \
    "$ROOT/scripts/e2e/ray_smoke.sh" >"$RAY_OUT" 2>"$RAY_ERR" &
  pids+=("$!")
fi

if [[ "$RUN_SHARED_QUEUE" == "1" ]]; then
  (
    python3 "$ROOT/scripts/e2e/multi_node_shared_queue.py" \
      --go-root "$ROOT" \
      --report-path "$SHARED_QUEUE_REPORT_REL"
  ) >"$SHARED_QUEUE_OUT" 2>"$SHARED_QUEUE_ERR" &
  pids+=("$!")
fi

if (( ${#pids[@]} > 0 )); then
  for pid in "${pids[@]}"; do
    if ! wait "$pid"; then
      status=1
    fi
  done
fi

python3 "$ROOT/scripts/e2e/export_validation_bundle.py" \
  --go-root "$ROOT" \
  --run-id "$RUN_ID" \
  --bundle-dir "$BUNDLE_DIR_REL" \
  --summary-path "$SUMMARY_REPORT_PATH" \
  --index-path "$INDEX_REPORT_PATH" \
  --manifest-path "$MANIFEST_REPORT_PATH" \
  --run-local "$RUN_LOCAL" \
  --run-kubernetes "$RUN_KUBERNETES" \
  --run-ray "$RUN_RAY" \
  --run-shared-queue "$RUN_SHARED_QUEUE" \
  --validation-status "$status" \
  --local-report-path "$LOCAL_REPORT_REL" \
  --local-stdout-path "$LOCAL_OUT" \
  --local-stderr-path "$LOCAL_ERR" \
  --kubernetes-report-path "$K8S_REPORT_REL" \
  --kubernetes-stdout-path "$K8S_OUT" \
  --kubernetes-stderr-path "$K8S_ERR" \
  --ray-report-path "$RAY_REPORT_REL" \
  --ray-stdout-path "$RAY_OUT" \
  --ray-stderr-path "$RAY_ERR" \
  --shared-queue-report-path "$SHARED_QUEUE_REPORT_REL" \
  --shared-queue-stdout-path "$SHARED_QUEUE_OUT" \
  --shared-queue-stderr-path "$SHARED_QUEUE_ERR"
