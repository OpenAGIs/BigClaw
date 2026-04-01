# BIG-GO-1038 Workpad

## Plan

1. Add a Go-native planning package that covers the candidate backlog, entry-gate evaluation, and
   four-week execution plan behavior from `tests/test_planning.py`, then delete that Python test.
2. Keep the implementation self-contained and reuse the existing governance freeze audit only for
   baseline readiness / readiness score checks.
3. Run targeted Go validation for the new planning package, plus repo-level file-count checks.
4. Commit the scoped migration changes and push the branch to the remote.

## Acceptance

- The number of Python files under `tests/` decreases again in this tranche.
- Deleted Python tests are covered by checked-in or expanded Go tests in `bigclaw-go/`.
- No new Python tests are introduced.
- `pyproject.toml` and `setup.py` remain absent.
- The final change can name the deleted Python files and the added or expanded Go test files.

## Validation

- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
- `cd bigclaw-go && go test ./internal/planning`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
- `git status --short`

## Validation Results
- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
  - Result: `7`
- `cd bigclaw-go && go test ./internal/planning`
  - Result:
    - `ok  	bigclaw-go/internal/planning	1.120s`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
  - Result: no output
