# BIG-GO-966 Workpad

## Plan

1. Inventory the Python tests in the issue lane: `repo`, `governance`, `reporting`, `risk`, `planning`, `mapping`, `memory`, `operations`, and `observability`; map each file to existing or missing Go coverage.
2. Rewrite or add the smallest Go-native replacements needed for lane-owned Python tests that are already covered or are cheap to migrate directly in this issue.
3. Delete only the Python tests that now have clear Go replacements, and keep broader files in place with explicit retention reasons.
4. Run targeted validation for each touched Go package, then measure the exact repository-wide Python file count impact.
5. Commit the scoped change set and push the branch to the remote branch.

## Acceptance

- Produce an explicit file list for the `BIG-GO-966` lane and state per-file delete/replace/keep reasoning.
- Reduce the number of Python files in the targeted lane as much as possible without widening scope.
- Record exact validation commands and outcomes for the touched Go packages.
- Report the before/after Python file counts for both the lane and the repository.

## Validation

- `go test` for each touched `bigclaw-go/internal/...` package in this lane.
- `git status --short` before commit to confirm the issue-scoped file set.
- `find . -type f -name '*.py' | wc -l` before and after the migration to report global impact.

## Notes

- Existing Go coverage already appears to replace the Python tests for `governance`, `risk`, `observability`, and most `repo/*` surfaces.
- `planning`, `operations`, and `reporting` still need selective keep-vs-migrate decisions based on whether the current Go code actually covers the Python contracts.

## Results

- Deleted Python tests in scope:
  - `tests/test_governance.py`
  - `tests/test_risk.py`
  - `tests/test_observability.py`
  - `tests/test_mapping.py`
  - `tests/test_memory.py`
  - `tests/test_planning.py`
  - `tests/test_repo_board.py`
  - `tests/test_repo_collaboration.py`
  - `tests/test_repo_gateway.py`
  - `tests/test_repo_governance.py`
  - `tests/test_repo_links.py`
  - `tests/test_repo_rollout.py`
  - `tests/test_repo_registry.py`
  - `tests/test_repo_triage.py`
- Added Go replacement coverage in:
  - `bigclaw-go/internal/memory/store.go`
  - `bigclaw-go/internal/memory/store_test.go`
  - `bigclaw-go/internal/planning/planning.go`
  - `bigclaw-go/internal/planning/planning_test.go`
  - Existing Go replacements already present in `internal/governance`, `internal/risk`, `internal/observability`, `internal/repo`, and `internal/intake`
- Kept Python tests in scope:
  - `tests/test_reports.py`
  - `tests/test_operations.py`
- Python file count impact:
  - Repository-wide before: `123`
  - Repository-wide after: `109`
  - Delta: `-14`

## Validation Results

- `cd bigclaw-go && go test ./internal/governance ./internal/reporting ./internal/risk ./internal/observability ./internal/repo ./internal/intake ./internal/memory`
  - `ok  	bigclaw-go/internal/governance	1.173s`
  - `ok  	bigclaw-go/internal/reporting	1.481s`
  - `ok  	bigclaw-go/internal/risk	1.967s`
  - `ok  	bigclaw-go/internal/observability	2.850s`
  - `ok  	bigclaw-go/internal/repo	3.289s`
  - `ok  	bigclaw-go/internal/intake	3.748s`
  - `ok  	bigclaw-go/internal/memory	2.412s`
- `cd bigclaw-go && go test ./internal/planning`
  - `ok  	bigclaw-go/internal/planning	3.164s`
- `find . -type f -name '*.py' | wc -l`
  - `109`
- `git status --short`
  - scoped changes only for `.symphony/workpad.md`, `bigclaw-go/internal/memory`, `bigclaw-go/docs/reports/big-go-966-material-pass.md`, and the deleted lane Python tests
