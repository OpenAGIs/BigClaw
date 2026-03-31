# BIG-GO-1032 Workpad

## Plan
1. Inspect the existing `scripts/ops/*.py` scripts and the current Go CLI layout to determine whether each Python entrypoint already has a Go equivalent or needs a new one.
2. Implement missing Go CLI replacements in the existing Go module, keeping behavior scoped to the script responsibilities needed by the repository.
3. Update repository entrypoints or wrappers as needed so the replaced workflows invoke Go binaries instead of Python scripts.
4. Delete the migrated Python files in `scripts/ops/`.
5. Run targeted validation for the new Go commands and any impacted tests, then record exact commands and outcomes.
6. Commit the changes with an issue-scoped message and push the branch to `origin`.

## Acceptance
- The count of Python files in `scripts/ops/*.py` decreases from the current baseline of 5.
- Replacement implementations are present as `.go` files and/or Go tests inside `bigclaw-go`.
- Any affected repository wrappers or docs point at the Go replacements instead of the deleted Python scripts.
- No new Python implementation is added for this migration.
- The final change can explicitly list deleted Python files and added Go files.

## Validation
- `go test ./...` from `bigclaw-go` if the impacted packages allow it; otherwise run targeted package tests covering the touched commands.
- Execute each migrated CLI with `--help` or equivalent non-destructive validation command.
- Recount Python files in `scripts/ops/*.py` after the migration to confirm the decrease.
- Capture exact commands and exit results in the final summary.

## Validation Results
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim ./internal/regression` -> passed
- `bash scripts/ops/bigclaw-github-sync status --json` -> passed
- `bash scripts/ops/bigclaw-refill-queue --help` -> passed
- `bash scripts/ops/bigclaw-workspace-bootstrap --help` -> passed
- `bash scripts/ops/symphony-workspace-bootstrap --help` -> passed
- `bash scripts/ops/symphony-workspace-validate --issues BIG-GO-1032 BIG-GO-1033 --report-file /tmp/big-go-1032-report.json --no-cleanup --help` -> passed
- `find scripts/ops -maxdepth 1 -name '*.py' | wc -l` -> `0`
- `find . -name pyproject.toml -o -name setup.py | wc -l` -> `0`
