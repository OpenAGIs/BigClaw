# BIG-GO-1038 Workpad

## Plan

1. Delete `tests/test_validation_bundle_continuation_policy_gate.py` because the Go-native
   `bigclawctl automation e2e continuation-policy-gate` tests already cover the policy-go and
   policy-hold paths for the same continuation bundle surface.
2. Delete `tests/test_parallel_validation_bundle.py` because the Go-native automation bundle
   command and regression tests already cover live validation bundle export, summary generation,
   index generation, and shared-queue companion paths.
3. Run targeted Go validation for `./cmd/bigclawctl` and `./internal/regression`, plus repo-level
   file-count checks.
4. Commit the scoped migration changes and push the branch to the remote.

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
  - `ok  	bigclaw-go/cmd/bigclawctl	3.903s`
  - `ok  	bigclaw-go/internal/regression	1.064s`
- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
  - `16`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
  - no output
