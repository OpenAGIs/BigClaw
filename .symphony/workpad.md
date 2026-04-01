# BIG-GO-1042 Workpad

## Plan

1. Verify which `src/bigclaw/*.py` top-level modules already have canonical Go owners and still have only limited Python compatibility references.
2. Replace the package-level compatibility for the tranche in `src/bigclaw/__init__.py` so the standalone Python files are no longer required.
3. Delete the retired Python tranche files from `src/bigclaw/` and add or extend focused Go tests for their canonical owners under `bigclaw-go/internal/...`.
4. Run targeted validation for the touched Python and Go surfaces, plus before/after Python file counts, and record exact commands and results.
5. Commit the scoped change with a message that lists the deleted Python files and added Go files/tests, then push the branch.

## Acceptance

- The repository-wide `*.py` file count decreases from the starting count for this issue.
- The deleted tranche is limited to top-level `src/bigclaw/*.py` modules already assigned to canonical Go owners.
- No new Python files are added.
- The final commit message names the deleted Python files and the added Go file(s) and Go test file(s).
- Targeted Python compatibility checks and targeted Go tests pass for the touched surfaces.

## Validation

- `find . -name "*.py" | wc -l`
- `PYTHONPATH=src python3 -m pytest tests/test_dsl.py tests/test_memory.py tests/test_runtime_matrix.py -q`
- `cd bigclaw-go && go test ./internal/intake ./internal/workflow ./internal/regression`
- `git status --short`

## Validation Results

- `find . -name "*.py" | wc -l`
  - `78`
- `PYTHONPATH=src python3 -m pytest tests/test_dsl.py tests/test_memory.py tests/test_runtime_matrix.py -q`
  - `8 passed in 0.21s`
- `cd bigclaw-go && go test ./internal/intake ./internal/workflow ./internal/regression`
  - `ok  	bigclaw-go/internal/intake	0.464s`
  - `ok  	bigclaw-go/internal/workflow	0.916s`
  - `ok  	bigclaw-go/internal/regression	1.251s`
- `git status --short`
  - `M .symphony/workpad.md`
  - `M src/bigclaw/__init__.py`
  - `D src/bigclaw/connectors.py`
  - `D src/bigclaw/dsl.py`
  - `D src/bigclaw/mapping.py`
  - `?? bigclaw-go/internal/regression/top_level_module_purge_tranche2_test.go`
