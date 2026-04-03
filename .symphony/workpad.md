# BIG-GO-1040 Workpad

## Plan

1. Delete every remaining legacy Python package module under `src/bigclaw` by folding the compatibility surfaces into staged package-root cutovers and then removing the physical `.py` files.
2. Tighten Go-side regression coverage after each deletion so the repo inventory explicitly converges to zero Python files and continues to reject `pyproject.toml` / `setup.py`.
3. Validate with targeted Go regression tests plus repo-wide file inventory checks, then commit and push the scoped issue branch.

## Acceptance

- Repository `.py` file count drops to zero within this issue scope.
- No new Python files are introduced.
- `pyproject.toml` and `setup.py` remain absent.
- The change can name which Python files were removed and which Go tests were added to pin the zero-Python state.

## Validation

- `find . -name '*.py' | sort | wc -l`
- `find . -name '*.py' | sort`
- `find . \( -name pyproject.toml -o -name setup.py \) -print | sort`
- `cd bigclaw-go && go test ./internal/regression`
- `git status --short`

## Validation Results

- Removed Python files:
  - `src/bigclaw/evaluation.py`
  - `src/bigclaw/models.py`
  - `src/bigclaw/operations.py`
  - `src/bigclaw/observability.py`
  - `src/bigclaw/reports.py`
  - `src/bigclaw/runtime.py`
  - `src/bigclaw/__init__.py`
- Added Go files:
  - `bigclaw-go/internal/regression/python_inventory_evaluation_cutover_test.go`
  - `bigclaw-go/internal/regression/python_inventory_models_cutover_test.go`
  - `bigclaw-go/internal/regression/python_inventory_operations_cutover_test.go`
  - `bigclaw-go/internal/regression/python_inventory_observability_cutover_test.go`
  - `bigclaw-go/internal/regression/python_inventory_reports_cutover_test.go`
  - `bigclaw-go/internal/regression/python_inventory_runtime_cutover_test.go`
  - `bigclaw-go/internal/regression/python_inventory_init_cutover_test.go`
- Updated Go files:
  - `bigclaw-go/internal/regression/python_inventory_test.go`
- Validation commands and results:
  - `cd bigclaw-go && go test ./internal/regression`
    - `ok  	bigclaw-go/internal/regression	0.649s`
  - `find . -name '*.py' | sort | wc -l`
    - `0`
  - `find . -name '*.py' | sort`
    - no output
  - `find . \( -name pyproject.toml -o -name setup.py \) -print | sort`
    - no output
  - `rmdir src/bigclaw`
    - directory removed; `find src -maxdepth 2 -type d | sort` returned only `src`
