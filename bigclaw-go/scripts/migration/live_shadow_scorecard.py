#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
PROJECT_ROOT="$(cd "$ROOT/.." && pwd)"

cd "$ROOT"
exec go run ./scripts/migration/live_shadow_scorecard.go --repo-root "$PROJECT_ROOT" "$@"
