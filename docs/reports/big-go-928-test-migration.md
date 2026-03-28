# BIG-GO-928 Test Migration

## Inventory

- `tests/test_mapping.py`
  - Legacy Python source: `src/bigclaw/mapping.py`
  - Go replacement: `bigclaw-go/internal/intake/mapping_test.go`
  - Migration status: covered in Go already; Python test removed in this issue.
- `tests/test_workspace_bootstrap.py`
  - Legacy Python sources: `src/bigclaw/workspace_bootstrap.py`, `src/bigclaw/workspace_bootstrap_validation.py`
  - Go replacement: `bigclaw-go/internal/bootstrap/bootstrap_test.go`
  - Migration status: Go coverage now includes cache-root derivation, warm-cache reuse, workspace reuse, cleanup cache preservation, stale-seed recovery, and validation report reuse.
- `tests/test_planning.py`
  - Legacy Python source: `src/bigclaw/planning.py`
  - Go replacement: `bigclaw-go/internal/planning/planning_test.go`
  - Migration status: Go coverage now includes planning model round-trips, ranking/scoring, gate evaluation, builders, four-week execution rollups, and report rendering.

## Go Replacements Landed

- Added `bigclaw-go/internal/planning/planning.go` with Go-native planning models, builders, gate evaluation, and markdown reports.
- Extended `bigclaw-go/internal/bootstrap/bootstrap_test.go` to cover the remaining bootstrap migration scenarios from Python.
- Removed migrated Python test files:
  - `tests/test_mapping.py`
  - `tests/test_workspace_bootstrap.py`
  - `tests/test_planning.py`

## Legacy Python Deletion Conditions

- `src/bigclaw/mapping.py` can be deleted once runtime callers have been moved to the Go intake path and no Python entrypoints import the mapping helpers.
- `src/bigclaw/workspace_bootstrap.py` and `src/bigclaw/workspace_bootstrap_validation.py` can be deleted once the repository bootstrap entrypoints use `bigclaw-go/internal/bootstrap` exclusively.
- `src/bigclaw/planning.py` can be deleted once any remaining Python consumers of planning builders/reports are redirected to the Go implementation or retired.

## Regression Commands

- `go test ./internal/bootstrap ./internal/intake ./internal/planning ./internal/governance`

## Reference Audit

- No active CI workflow, script, or non-historical execution doc still invokes `tests/test_mapping.py`, `tests/test_workspace_bootstrap.py`, or `tests/test_planning.py`.
- Remaining references are limited to historical validation reports under `reports/` and migration tracking documents that intentionally preserve prior-state context.
