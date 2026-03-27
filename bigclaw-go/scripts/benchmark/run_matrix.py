#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

cd "$ROOT"
exec go run ./scripts/benchmark/run_matrix.go --go-root "$ROOT" "$@"
