# BIG-GO-1038 Workpad

## Plan

1. Delete `tests/test_live_shadow_bundle.py` because the Go-native live-shadow bundle export path
   is already covered by `bigclawctl` automation command tests and the checked-in live-shadow
   bundle surfaces are locked by regression tests.
2. Run targeted Go validation for `./cmd/bigclawctl` and `./internal/regression`, plus repo-level
   file-count checks.
3. Commit the scoped migration changes and push the branch to the remote.

## Acceptance

- The number of Python files under `tests/` decreases in this tranche.
- Deleted Python tests are covered by checked-in or expanded Go tests in `bigclaw-go/`.
- No new Python tests are introduced.
- `pyproject.toml` and `setup.py` remain absent.
- The final change can name the deleted Python files and the added or expanded Go test files.

## Validation

- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/regression`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
- `git status --short`

## Validation Results

- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/regression`
  - `ok  	bigclaw-go/cmd/bigclawctl	(cached)`
  - `ok  	bigclaw-go/internal/regression	(cached)`
- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
  - `15`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
  - no output
