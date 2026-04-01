# BIG-GO-1038 Workpad

## Plan

1. Add a small Go-native memory package that covers the persisted successful-task pattern reuse
   behavior from `tests/test_memory.py`, then delete that Python test.
2. Add a small Go-native runtime matrix package that covers multi-tool execution, routing
   expectations, and tool policy auditing from `tests/test_runtime_matrix.py`, then delete that
   Python test.
3. Run targeted Go validation for `./internal/memory` and `./internal/runtimematrix`, plus
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
- `cd bigclaw-go && go test ./internal/memory ./internal/runtimematrix`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
- `git status --short`

## Validation Results
- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
  - Result: `9`
- `cd bigclaw-go && go test ./internal/memory ./internal/runtimematrix`
  - Result:
    - `ok  	bigclaw-go/internal/memory	0.936s`
    - `ok  	bigclaw-go/internal/runtimematrix	0.462s`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
  - Result: no output
