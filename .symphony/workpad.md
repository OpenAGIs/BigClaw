# BIG-GO-1038 Workpad

## Plan

1. Add a Go-native run-history / observability surface that covers the behaviors from
   `tests/test_observability.py`, including task-run capture, ledger persistence, repo sync audit
   rendering, detail-page rendering, and collaboration extraction from audits, then delete that
   Python test.
2. Reuse existing repo commit-link types where it helps, but keep the task-run and rendering
   implementation scoped to this issue.
3. Run targeted Go validation for the touched packages, plus repo-level file-count checks.
4. Commit the scoped migration changes and push the branch to the remote.

## Acceptance

- The number of Python files under `tests/` decreases again in this tranche.
- Deleted Python tests are covered by checked-in or expanded Go tests in `bigclaw-go/`.
- No new Python tests are introduced.
- `pyproject.toml` and `setup.py` remain absent.
- The final change can name the deleted Python files and the added or expanded Go test files.

## Validation

- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
- `cd bigclaw-go && go test ./internal/observability ./internal/collaboration`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
- `git status --short`

## Validation Results
- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
  - Result: `6`
- `cd bigclaw-go && go test ./internal/observability ./internal/collaboration`
  - Result:
    - `ok  	bigclaw-go/internal/observability	0.429s`
    - `ok  	bigclaw-go/internal/collaboration	(cached)`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
  - Result: no output
