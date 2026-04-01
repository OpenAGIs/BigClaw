# BIG-GO-1038 Workpad

## Plan

1. Extend the existing Go queue and reporting surfaces to cover the remaining
   `tests/test_control_center.py` behaviors, including queue peeking and shared-view empty-state
   rendering, then delete that Python test.
2. Delete `tests/conftest.py` once no remaining top-level Python tests depend on it.
3. Run targeted Go validation for the touched queue and reporting packages, plus repo-level
   file-count checks.
4. Commit the scoped migration changes and push the branch to the remote.

## Acceptance

- The number of Python files under `tests/` decreases again in this tranche.
- Deleted Python tests are covered by checked-in or expanded Go tests in `bigclaw-go/`.
- No new Python tests are introduced.
- `pyproject.toml` and `setup.py` remain absent.
- The final change can name the deleted Python files and the added or expanded Go test files.

## Validation

- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
- `cd bigclaw-go && go test ./internal/queue ./internal/reporting`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
- `git status --short`

## Validation Results
- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
  - Result: `8`
- `cd bigclaw-go && go test ./internal/queue ./internal/reporting`
  - Result:
    - `ok  	bigclaw-go/internal/queue	(cached)`
    - `ok  	bigclaw-go/internal/reporting	0.402s`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
  - Result: no output
