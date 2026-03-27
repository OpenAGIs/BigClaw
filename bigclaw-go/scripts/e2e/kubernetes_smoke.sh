#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
IMAGE="${BIGCLAW_KUBERNETES_SMOKE_IMAGE:-${BIGCLAW_KUBERNETES_IMAGE:-alpine:3.20}}"
ENTRYPOINT="${BIGCLAW_KUBERNETES_SMOKE_ENTRYPOINT:-echo hello from kubernetes}"
REPORT_PATH="${BIGCLAW_KUBERNETES_SMOKE_REPORT_PATH:-docs/reports/kubernetes-live-smoke-report.json}"
go run "$ROOT/scripts/e2e/run_task_smoke.go" \
  --autostart \
  --go-root "$ROOT" \
  --executor kubernetes \
  --title "Kubernetes smoke test" \
  --image "$IMAGE" \
  --entrypoint "$ENTRYPOINT" \
  --report-path "$REPORT_PATH"
