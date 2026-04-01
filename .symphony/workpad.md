# BIG-GO-1038 Workpad

## Plan

1. Add a small Go-native collaboration package that covers thread construction and merge behavior,
   then delete `tests/test_repo_collaboration.py`.
2. Add a small Go-native repo rollout package that covers the pilot rollout scorecard, candidate
   gate evaluation, and repo narrative export rendering, then delete `tests/test_repo_rollout.py`.
3. Run targeted Go validation for `./internal/collaboration` and `./internal/reporollout`, plus
   repo-level file-count checks.
4. Commit the scoped migration changes and push the branch to the remote.

## Acceptance

- The number of Python files under `tests/` decreases in this tranche.
- Deleted Python tests are covered by checked-in or expanded Go tests in `bigclaw-go/`.
- No new Python tests are introduced.
- `pyproject.toml` and `setup.py` remain absent.
- The final change can name the deleted Python files and the added or expanded Go test files.

## Validation

- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
- `cd bigclaw-go && go test ./internal/collaboration ./internal/reporollout`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
- `git status --short`

## Validation Results

- `cd bigclaw-go && go test ./internal/collaboration ./internal/reporollout`
  - `ok  	bigclaw-go/internal/collaboration	0.380s`
  - `ok  	bigclaw-go/internal/reporollout	0.769s`
- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
  - `13`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
  - no output
