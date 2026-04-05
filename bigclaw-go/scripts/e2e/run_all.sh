#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
exec go run "$ROOT/cmd/bigclawctl" automation e2e run-all --go-root "$ROOT" "$@"
