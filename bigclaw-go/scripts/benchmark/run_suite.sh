#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
mkdir -p "$ROOT/docs/reports"
(
  cd "$ROOT"
  go test -bench . ./internal/queue ./internal/scheduler | tee "$ROOT/docs/reports/benchmark-report.md"
  go run ./cmd/bigclawctl automation benchmark run-matrix \
    --scenario 50:8 \
    --scenario 100:12 \
    --report-path docs/reports/benchmark-matrix-report.json >/dev/null
)
