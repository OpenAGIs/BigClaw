# BIG-GO-1038 Workpad

## Plan

1. Keep this tranche scoped to the remaining Python tests now replaced by Go-native coverage on
   this branch: `tests/test_models.py`, `tests/test_repo_gateway.py`, `tests/test_repo_registry.py`,
   `tests/test_risk.py`, `tests/test_orchestration.py`, and `tests/test_validation_policy.py`.
2. Extend the matching Go-native tests and packages only where a deleted Python assertion still
   needs an explicit repo-native replacement, without touching unrelated product areas.
3. Validate the touched Go packages and record the exact commands and results.
4. Commit the scoped migration changes and push the branch to the remote.

## Acceptance

- The number of Python files under `tests/` decreases in this tranche.
- Deleted Python tests are covered by checked-in or expanded Go tests in `bigclaw-go/`.
- No new Python tests are introduced.
- `pyproject.toml` and `setup.py` remain absent.
- The final change can name the deleted Python files and the added or expanded Go test files.

## Validation

- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
- `cd bigclaw-go && go test ./internal/workflow ./internal/scheduler ./internal/worker ./internal/validationpolicy ./internal/repo ./internal/risk ./internal/billing ./internal/triage`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
- `git status --short`

## Validation Results

- `cd bigclaw-go && go test ./internal/workflow ./internal/scheduler ./internal/worker ./internal/validationpolicy ./internal/repo ./internal/risk ./internal/billing ./internal/triage`
  - `ok  	bigclaw-go/internal/workflow	(cached)`
  - `ok  	bigclaw-go/internal/scheduler	(cached)`
  - `ok  	bigclaw-go/internal/worker	(cached)`
  - `ok  	bigclaw-go/internal/validationpolicy	(cached)`
  - `ok  	bigclaw-go/internal/repo	(cached)`
  - `ok  	bigclaw-go/internal/risk	(cached)`
  - `ok  	bigclaw-go/internal/billing	(cached)`
  - `ok  	bigclaw-go/internal/triage	(cached)`
- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
  - `20`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
  - no output
