#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"
go run ./cmd/bigclawctl automation benchmark run-matrix "$@"
