# BIG-GO-1038 Workpad

## Plan

1. Delete `tests/test_orchestration.py` after closing its remaining Go gap with a repo-native
   orchestration plan renderer test in `bigclaw-go/internal/workflow`.
2. Add a small Go-native validation policy package and tests that replace
   `tests/test_validation_policy.py`, then delete that Python test.
3. Run targeted Go validation for `./internal/workflow`, `./internal/scheduler`,
   `./internal/worker`, and `./internal/validationpolicy`, plus repo-level file-count checks.
4. Commit the scoped migration changes and push the branch to the remote.

## Acceptance

- The number of Python files under `tests/` decreases in this tranche.
- Deleted Python tests are covered by checked-in or expanded Go tests in `bigclaw-go/`.
- No new Python tests are introduced.
- `pyproject.toml` and `setup.py` remain absent.
- The final change can name the deleted Python files and the added or expanded Go test files.

## Validation

- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
- `cd bigclaw-go && go test ./internal/workflow ./internal/scheduler ./internal/worker ./internal/validationpolicy`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
- `git status --short`

## Validation Results

- `cd bigclaw-go && go test ./internal/workflow ./internal/scheduler ./internal/worker ./internal/validationpolicy`
  - `ok  	bigclaw-go/internal/workflow	0.488s`
  - `ok  	bigclaw-go/internal/scheduler	1.885s`
  - `ok  	bigclaw-go/internal/worker	1.161s`
  - `ok  	bigclaw-go/internal/validationpolicy	1.099s`
- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
  - `18`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
  - no output
