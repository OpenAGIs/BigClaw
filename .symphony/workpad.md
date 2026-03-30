# BIG-GO-1016 Workpad

## Scope

Target the first `tests/**` Python residual tranche that already has direct
Go-owned product equivalents, so the lane can delete Python files instead of
adding parallel duplicate coverage.

Batch file list:

- `tests/test_saved_views.py`
- `tests/test_dashboard_run_contract.py`
- `bigclaw-go/internal/product/saved_views_test.go`
- `bigclaw-go/internal/product/dashboard_run_contract_test.go`

Repository inventory at start of lane:

- Repository `py` files: `108`
- Repository `go` files: `267`
- Root `pyproject.toml`: absent
- Root `setup.py`: absent

## Plan

1. Confirm the selected Python tests map cleanly onto existing Go product
   packages and keep the lane scoped to that tranche only.
2. Port any Python-only assertions that are still missing in the Go test files.
3. Delete the migrated Python tests from `tests/**`.
4. Run targeted Go tests for the touched package and record exact commands and
   results.
5. Recount repository `py` and `go` files and record impact on
   `pyproject.toml` / `setup.py`.
6. Commit and push the scoped branch for `BIG-GO-1016`.

## Acceptance

- Directly reduce repository-level Python residuals under `tests/**`.
- Keep changes limited to the selected tranche files.
- Preserve or improve coverage by moving any Python-only assertions into Go
  tests before deleting the Python files.
- Report the impact on repository `py files`, `go files`, `pyproject.toml`, and
  `setup.py`.
- Do not treat tracker state as the deliverable; the repository result is the
  deliverable.

## Validation

- `go test ./bigclaw-go/internal/product`
- `git diff --check`
- `git status --short`
- `find . -name '*.py' | sort | wc -l`
- `find . -name '*.go' | sort | wc -l`
- `test -f pyproject.toml; echo $?`
- `test -f setup.py; echo $?`

## Results

### File Disposition

- `tests/test_saved_views.py`
  - Deleted.
  - Reason: its report and audit assertions are now covered by
    `bigclaw-go/internal/product/saved_views_test.go`.
- `tests/test_dashboard_run_contract.py`
  - Deleted.
  - Reason: its release-ready, missing-path, and round-trip assertions are now
    covered by `bigclaw-go/internal/product/dashboard_run_contract_test.go`.
- `bigclaw-go/internal/product/saved_views_test.go`
  - Replaced.
  - Reason: added exact saved-view report assertions so the Python test can be
    removed without losing coverage.
- `bigclaw-go/internal/product/dashboard_run_contract_test.go`
  - Replaced.
  - Reason: added exact missing-path assertions and JSON round-trip checks so
    the Python test can be removed without losing coverage.

### Impact

- Repository `py` files before: `108`
- Repository `py` files after: `106`
- Net `py` reduction: `2`
- Repository `go` files before: `267`
- Repository `go` files after: `267`
- Net `go` reduction: `0`
- Root `pyproject.toml`: absent before and after
- Root `setup.py`: absent before and after

### Validation Record

- `go test ./bigclaw-go/internal/product`
  - Result: failed from repo root because Go module root is `bigclaw-go/`
    (`go: cannot find main module`)
- `go test ./internal/product`
  - Working directory: `bigclaw-go/`
  - Result: `ok   bigclaw-go/internal/product  0.453s`
- `git diff --check`
  - Result: clean
- `git status --short`
  - Result before commit:
    - `M .symphony/workpad.md`
    - `M bigclaw-go/internal/product/dashboard_run_contract_test.go`
    - `M bigclaw-go/internal/product/saved_views_test.go`
    - `D tests/test_dashboard_run_contract.py`
    - `D tests/test_saved_views.py`
- `find . -name '*.py' | sort | wc -l`
  - Result after: `106`
- `find . -name '*.go' | sort | wc -l`
  - Result after: `267`
- `test -f pyproject.toml; echo $?`
  - Result: `1`
- `test -f setup.py; echo $?`
  - Result: `1`
