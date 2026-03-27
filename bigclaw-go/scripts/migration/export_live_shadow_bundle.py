#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

cd "$ROOT"
exec go run ./scripts/migration/export_live_shadow_bundle.go --go-root "$ROOT" "$@"
