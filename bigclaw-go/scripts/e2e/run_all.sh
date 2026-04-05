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
RUN_BROKER="${BIGCLAW_E2E_RUN_BROKER:-0}"
BROKER_BACKEND="${BIGCLAW_E2E_BROKER_BACKEND:-}"
BROKER_REPORT_PATH="${BIGCLAW_E2E_BROKER_REPORT_PATH:-}"
BROKER_BOOTSTRAP_SUMMARY_PATH="${BIGCLAW_E2E_BROKER_BOOTSTRAP_SUMMARY_PATH:-docs/reports/broker-bootstrap-review-summary.json}"
REFRESH_CONTINUATION="${BIGCLAW_E2E_REFRESH_CONTINUATION:-1}"
ENFORCE_CONTINUATION_GATE="${BIGCLAW_E2E_ENFORCE_CONTINUATION_GATE:-0}"
CONTINUATION_GATE_MODE="${BIGCLAW_E2E_CONTINUATION_GATE_MODE:-}"
# NOTE: The continuation scripts resolve output paths relative to the repo root,
# so we must include the bigclaw-go prefix here to avoid emitting repo-root
# duplicates under docs/reports/.
CONTINUATION_SCORECARD_PATH="${BIGCLAW_E2E_CONTINUATION_SCORECARD_PATH:-bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json}"
CONTINUATION_POLICY_GATE_PATH="${BIGCLAW_E2E_CONTINUATION_POLICY_GATE_PATH:-bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json}"

if [[ -z "$CONTINUATION_GATE_MODE" ]]; then
  if [[ "$ENFORCE_CONTINUATION_GATE" == "1" ]]; then
    CONTINUATION_GATE_MODE="fail"
  else
    CONTINUATION_GATE_MODE="hold"
  fi
fi

mkdir -p "$ROOT/$BUNDLE_DIR_REL"

LOCAL_REPORT_REL="$BUNDLE_DIR_REL/sqlite-smoke-report.json"
K8S_REPORT_REL="$BUNDLE_DIR_REL/kubernetes-live-smoke-report.json"
RAY_REPORT_REL="$BUNDLE_DIR_REL/ray-live-smoke-report.json"

LOCAL_OUT="$(mktemp -t bigclaw-local-e2e-out.XXXXXX)"
LOCAL_ERR="$(mktemp -t bigclaw-local-e2e-err.XXXXXX)"
K8S_OUT="$(mktemp -t bigclaw-k8s-e2e-out.XXXXXX)"
K8S_ERR="$(mktemp -t bigclaw-k8s-e2e-err.XXXXXX)"
RAY_OUT="$(mktemp -t bigclaw-ray-e2e-out.XXXXXX)"
RAY_ERR="$(mktemp -t bigclaw-ray-e2e-err.XXXXXX)"
trap 'rm -f "$LOCAL_OUT" "$LOCAL_ERR" "$K8S_OUT" "$K8S_ERR" "$RAY_OUT" "$RAY_ERR"' EXIT

status=0
pids=()

if [[ "$RUN_LOCAL" == "1" ]]; then
  (
    BIGCLAW_QUEUE_BACKEND=sqlite \
      go run "$ROOT/scripts/e2e/run_task_smoke" \
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

if (( ${#pids[@]} > 0 )); then
  for pid in "${pids[@]}"; do
    if ! wait "$pid"; then
      status=1
    fi
  done
fi

export_bundle() {
  go run "$ROOT/scripts/e2e/broker_bootstrap_summary.go" \
    --output "$ROOT/$BROKER_BOOTSTRAP_SUMMARY_PATH"
  go run "$ROOT/scripts/e2e/export_validation_bundle" \
    --go-root "$ROOT" \
    --run-id "$RUN_ID" \
    --bundle-dir "$BUNDLE_DIR_REL" \
    --summary-path "$SUMMARY_REPORT_PATH" \
    --index-path "$INDEX_REPORT_PATH" \
    --manifest-path "$MANIFEST_REPORT_PATH" \
    --run-local "$RUN_LOCAL" \
    --run-kubernetes "$RUN_KUBERNETES" \
    --run-ray "$RUN_RAY" \
    --run-broker "$RUN_BROKER" \
    --broker-backend "$BROKER_BACKEND" \
    --broker-report-path "$BROKER_REPORT_PATH" \
    --broker-bootstrap-summary-path "$BROKER_BOOTSTRAP_SUMMARY_PATH" \
    --validation-status "$status" \
    --local-report-path "$LOCAL_REPORT_REL" \
    --local-stdout-path "$LOCAL_OUT" \
    --local-stderr-path "$LOCAL_ERR" \
    --kubernetes-report-path "$K8S_REPORT_REL" \
    --kubernetes-stdout-path "$K8S_OUT" \
    --kubernetes-stderr-path "$K8S_ERR" \
    --ray-report-path "$RAY_REPORT_REL" \
    --ray-stdout-path "$RAY_OUT" \
    --ray-stderr-path "$RAY_ERR"
}

export_status=0
if ! export_bundle; then
  export_status=$?
fi

if [[ "$REFRESH_CONTINUATION" == "1" ]]; then
  go run "$ROOT/scripts/e2e/validation_bundle_continuation_scorecard" \
    --output "$CONTINUATION_SCORECARD_PATH"

  gate_status=0
  if ! go run "$ROOT/scripts/e2e/validation_bundle_continuation_policy_gate" \
    --scorecard "$CONTINUATION_SCORECARD_PATH" \
    --enforcement-mode "$CONTINUATION_GATE_MODE" \
    --output "$CONTINUATION_POLICY_GATE_PATH"; then
    gate_status=$?
  fi

  rerender_status=0
  if ! export_bundle; then
    rerender_status=$?
  fi
  if [[ "$export_status" -eq 0 && "$rerender_status" -ne 0 ]]; then
    export_status=$rerender_status
  fi

  if [[ "$gate_status" -ne 0 ]]; then
    exit "$gate_status"
  fi
fi

exit "$export_status"
