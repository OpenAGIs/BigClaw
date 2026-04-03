# BIG-GO-1040 Workpad

## Plan

1. Delete one remaining legacy package module by folding the `src/bigclaw/evaluation.py` surface into `src/bigclaw/operations.py` and exposing a compatibility `bigclaw.evaluation` module from package init.
2. Remove the physical Python file and tighten Go-side regression coverage so the repo inventory explicitly expects one fewer `.py` file.
3. Validate with targeted Go tests plus repo-wide Python inventory checks, then commit and push the scoped issue branch.

## Acceptance

- Repository `.py` file count drops within this issue scope.
- No new Python files are introduced.
- `src/bigclaw/evaluation.py` is deleted while `bigclaw.evaluation` remains import-compatible through the package surface.
- Go regression coverage is updated to pin the new lower Python inventory.
- `pyproject.toml` and `setup.py` remain absent.

## Validation

- `find src/bigclaw -maxdepth 1 -name '*.py' | sort`
- `find . \( -name pyproject.toml -o -name setup.py \) -print | sort`
- `cd bigclaw-go && go test ./internal/regression`
- `PYTHONPATH=src python3 - <<'PY'`
  `import bigclaw`
  `import bigclaw.evaluation as evaluation`
  `print(evaluation.BenchmarkRunner.__name__)`
  `print(bigclaw.BenchmarkSuiteResult.__name__)`
  `PY`
- `git status --short`

## Validation Results

- Removed Python files:
  - `src/bigclaw/evaluation.py`
- Added Go files:
  - `bigclaw-go/internal/regression/python_inventory_evaluation_cutover_test.go`
- Updated files:
  - `src/bigclaw/operations.py`
  - `src/bigclaw/__init__.py`
  - `bigclaw-go/internal/regression/python_inventory_test.go`
- `find . -name '*.py' | sort | wc -l`
  - `6`
- `find . \( -name pyproject.toml -o -name setup.py \) -print | sort`
  - no output
- `cd bigclaw-go && go test ./internal/regression`
  - `ok  	bigclaw-go/internal/regression	(cached)`
- `PYTHONPATH=src python3 - <<'PY' ...`
  - `BenchmarkRunner`
  - `BenchmarkRunner`
  - `bigclaw-go/internal/regression/python_inventory_evaluation_cutover_test.go`
- `git status --short`
  - `M .symphony/workpad.md`
  - `M bigclaw-go/internal/regression/python_inventory_test.go`
  - `M src/bigclaw/__init__.py`
  - `D src/bigclaw/evaluation.py`
  - `M src/bigclaw/operations.py`
  - `?? bigclaw-go/internal/regression/python_inventory_evaluation_cutover_test.go`

## Validation Results

- Removed Python files:
  - `src/bigclaw/evaluation.py`
- Added Go files:
  - `bigclaw-go/internal/regression/python_inventory_evaluation_cutover_test.go`
- Updated Go files:
  - `bigclaw-go/internal/regression/python_inventory_test.go`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/operations.py src/bigclaw/models.py src/bigclaw/observability.py src/bigclaw/reports.py src/bigclaw/runtime.py`
  - no output
- `gofmt -w bigclaw-go/internal/regression/python_inventory_test.go bigclaw-go/internal/regression/python_inventory_evaluation_cutover_test.go`
  - no output
- `PYTHONPATH=src python3 - <<'PY'`
  - `BenchmarkRunner`
  - `BenchmarkSuiteResult`
- `cd bigclaw-go && go test ./internal/regression`
  - `ok  	bigclaw-go/internal/regression	(cached)`
- `find src/bigclaw -maxdepth 1 -name '*.py' | sort`
  - `src/bigclaw/__init__.py`
  - `src/bigclaw/models.py`
  - `src/bigclaw/observability.py`
  - `src/bigclaw/operations.py`
  - `src/bigclaw/reports.py`
  - `src/bigclaw/runtime.py`
- `find . -name '*.py' | sort | wc -l`
  - `6`
- `find . \( -name pyproject.toml -o -name setup.py \) -print | sort`
  - no output
- `git status --short`
  - `M .symphony/workpad.md`
  - `M bigclaw-go/internal/regression/python_inventory_test.go`
  - `M src/bigclaw/__init__.py`
  - `D src/bigclaw/evaluation.py`
  - `M src/bigclaw/operations.py`
  - `?? bigclaw-go/internal/regression/python_inventory_evaluation_cutover_test.go`
