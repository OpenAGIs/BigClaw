#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo_root/bigclaw-go"
exec go run ./cmd/bigclawctl dev bootstrap --repo-root "$repo_root" "$@"
