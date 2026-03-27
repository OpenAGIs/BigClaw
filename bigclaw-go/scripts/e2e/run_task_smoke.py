#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

cd "$ROOT"
exec go run ./scripts/e2e/run_task_smoke.go "$@"
