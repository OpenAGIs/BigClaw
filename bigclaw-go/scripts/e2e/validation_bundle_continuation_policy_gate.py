#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
PROJECT_ROOT="$(cd "$ROOT/.." && pwd)"

cd "$ROOT"
exec go run ./scripts/e2e/validation_bundle_continuation_policy_gate.go --repo-root "$PROJECT_ROOT" "$@"
