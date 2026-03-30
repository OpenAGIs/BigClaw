# BIG-GO-985 Workpad

## Scope

Targeted remaining Python batch under `src/bigclaw/`:

- `src/bigclaw/workspace_bootstrap.py`
- `src/bigclaw/workspace_bootstrap_cli.py`
- `src/bigclaw/workspace_bootstrap_validation.py`

Direct Go-mainline replacement path:

- `bigclaw-go/cmd/bigclawctl/main.go`
- `bigclaw-go/cmd/bigclawctl/main_test.go`

Why this batch:

- Operator-facing workspace bootstrap and validation flows already route through `bigclawctl workspace ...`.
- The three Python modules are no longer imported by runtime/operator entrypoints; remaining coupling is limited to legacy Python tests.
- The Go CLI already carries workspace bootstrap/validate coverage, making this a defensible removal slice.

Current repository Python file count before this lane: `116`
Current `src/bigclaw/**` Python file count before this lane: `45`

## Plan

1. Remove the three legacy workspace bootstrap Python modules from `src/bigclaw/`.
2. Remove directly coupled legacy Python tests that only validate those deleted modules.
3. Run targeted Go CLI validation for the replacement workspace flows plus a focused repo sanity check.
4. Record exact file disposition, replacement rationale, validation commands, and before/after Python counts.
5. Commit and push the scoped issue branch changes.

## Acceptance

- Produce the exact `BIG-GO-985` batch file list under `src/bigclaw/**`.
- Reduce Python file count in `src/bigclaw/**` by removing the selected legacy modules.
- Give explicit keep/replace/delete rationale for each batch file.
- Report repository-wide and `src/bigclaw/**` Python file counts before and after.

## Validation

- `cd bigclaw-go && go test ./cmd/bigclawctl`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/bootstrap`
- `cd bigclaw-go && go test ./internal/bootstrap -run 'Test.*(Workspace|Validation|Cache|Cleanup)'`
- `git status --short`

## Results

### File Disposition

- `src/bigclaw/workspace_bootstrap.py`
  - Deleted.
  - Reason: replaced by Go workspace bootstrap implementation and tests in `bigclaw-go/internal/bootstrap` surfaced through `bigclaw-go/cmd/bigclawctl/main.go`.
- `src/bigclaw/workspace_bootstrap_cli.py`
  - Deleted.
  - Reason: legacy Python CLI wrapper superseded by `bigclawctl workspace <bootstrap|cleanup|validate>`.
- `src/bigclaw/workspace_bootstrap_validation.py`
  - Deleted.
  - Reason: replaced by Go validation report generation in `bigclaw-go/internal/bootstrap` and CLI entrypoints in `bigclaw-go/cmd/bigclawctl/main.go`.
- `tests/test_workspace_bootstrap.py`
  - Deleted.
  - Reason: direct legacy-only test coverage for the deleted Python implementation; replacement coverage already exists in Go.

### Python File Count Impact

- Repository Python files before: `116`
- Repository Python files after: `112`
- `src/bigclaw/**` Python files before: `45`
- `src/bigclaw/**` Python files after: `42`
- Net reduction: `4`

### Validation Record

- `cd bigclaw-go && go test ./cmd/bigclawctl`
  - Result: `ok  	bigclaw-go/cmd/bigclawctl	2.165s`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/bootstrap`
  - Result: `ok  	bigclaw-go/cmd/bigclawctl	(cached)` and `ok  	bigclaw-go/internal/bootstrap	1.307s`
- `cd bigclaw-go && go test ./internal/bootstrap -run 'Test.*(Workspace|Validation|Cache|Cleanup)'`
  - Result: `ok  	bigclaw-go/internal/bootstrap	1.076s`
- `python3 -m pytest tests/test_legacy_shim.py tests/test_deprecation.py`
  - Result: failed before execution because neither file exists in this checkout.
- `python3 -m pytest tests/test_deprecation.py`
  - Result: failed before execution because the file does not exist in this checkout.
- `git status --short`
  - Result before commit: only `.symphony/workpad.md`, the three deleted `src/bigclaw/workspace_bootstrap*.py` files, and deleted `tests/test_workspace_bootstrap.py` were changed.
