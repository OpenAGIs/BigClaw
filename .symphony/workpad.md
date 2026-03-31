# BIG-GO-1038 Workpad

## Plan

1. Inventory remaining `tests/*.py` files and identify the tranche with clear Go-native replacements already present in `bigclaw-go/`.
2. Add or extend targeted Go tests where Python coverage still lacks a direct Go home but the production contract already exists in Go.
3. Delete the replaced Python test files and remove `tests/conftest.py` if no remaining Python tests require it.
4. Run targeted Go validation for the touched packages and record exact commands and results.
5. Commit the scoped migration changes and push the branch to the remote.

## Acceptance

- The number of Python files under `tests/` decreases in this issue scope.
- Any deleted Python test has a checked-in Go replacement test in `bigclaw-go/`.
- No new Python tests are introduced.
- `pyproject.toml` and `setup.py` remain absent.
- The final change can name the deleted Python files and the added or expanded Go test files.

## Validation

- `find tests -maxdepth 1 -name '*.py' | sort`
- Targeted `go test` commands for each touched Go package
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
- `git status --short`

## Validation Results

- `cd bigclaw-go && go test ./internal/bootstrap`
  - `ok  	bigclaw-go/internal/bootstrap	4.862s`
- `cd bigclaw-go && go test ./internal/product`
  - `ok  	bigclaw-go/internal/product	2.728s`
- `cd bigclaw-go && go test ./internal/contract`
  - `ok  	bigclaw-go/internal/contract	1.370s`
- `cd bigclaw-go && go test ./internal/githubsync`
  - `ok  	bigclaw-go/internal/githubsync	3.702s`
- `cd bigclaw-go && go test ./internal/governance`
  - `ok  	bigclaw-go/internal/governance	0.534s`
- `cd bigclaw-go && go test ./internal/observability`
  - `ok  	bigclaw-go/internal/observability	1.891s`
- `PYTHONPATH=src python3 -m pytest tests/test_planning.py -q`
  - `14 passed in 0.18s`
- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
  - `31`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
  - no output
