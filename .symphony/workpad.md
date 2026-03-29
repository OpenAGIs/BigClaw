# BIG-GO-975 Workpad

## Scope

Targeted remaining Python test batch under `tests/` for this lane:

- `tests/test_connectors.py`
- `tests/test_mapping.py`

Existing Go-native replacement paths:

- `bigclaw-go/internal/intake/connector_test.go`
- `bigclaw-go/internal/intake/mapping_test.go`

Current repository Python file count before this lane: `119`
Current `tests/**` Python file count before this lane: `43`

## Plan

1. Confirm the selected batch maps cleanly to existing Go-native intake coverage.
2. Remove the redundant Python test files for connectors and mapping.
3. Run the targeted Go intake tests that now serve as the replacement coverage.
4. Record the exact file list, replacement paths, validation commands, and Python file-count impact.
5. Commit and push the scoped lane changes.

## Acceptance

- Produce the exact `BIG-GO-975` batch file list.
- Reduce Python files in `tests/**` by removing the selected batch or clearly document the Go replacement path.
- Keep changes scoped to the intake test migration batch only.
- Report before/after repository-wide and `tests/**` Python file counts.

## Validation

- `cd bigclaw-go && go test ./internal/intake`
- `git status --short`

## Results

### File Disposition

- `tests/test_connectors.py`
  - Deleted.
  - Reason: replaced by existing Go-native intake coverage in `bigclaw-go/internal/intake/connector_test.go`.
- `tests/test_mapping.py`
  - Deleted.
  - Reason: replaced by existing Go-native intake coverage in `bigclaw-go/internal/intake/mapping_test.go`.

### Python File Count Impact

- Repository Python files before: `119`
- Repository Python files after: `117`
- `tests/**` Python files before: `43`
- `tests/**` Python files after: `41`
- Net reduction: `2`

### Validation Record

- `cd bigclaw-go && go test ./internal/intake`
  - Result: `ok  	bigclaw-go/internal/intake	0.461s`
- `git status --short`
  - Result: only `.symphony/workpad.md`, `tests/test_connectors.py`, and `tests/test_mapping.py` changed before commit.
