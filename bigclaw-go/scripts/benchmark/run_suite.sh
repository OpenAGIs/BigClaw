#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
mkdir -p "$ROOT/docs/reports"
(
  cd "$ROOT"
  go test -bench . ./internal/queue ./internal/scheduler | tee "$ROOT/docs/reports/benchmark-report.md"
)
