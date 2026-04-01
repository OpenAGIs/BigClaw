# BIG-GO-1038 Workpad

## Plan

1. Delete the repo-surface and risk Python tests that already have direct Go replacements:
   `tests/test_repo_governance.py`, `tests/test_repo_board.py`, `tests/test_repo_triage.py`,
   `tests/test_repo_links.py`, `tests/test_repo_registry.py`, `tests/test_repo_gateway.py`,
   `tests/test_risk.py`, and `tests/test_dsl.py`.
2. Extend `bigclaw-go/internal/repo/repo_surfaces_test.go` with the missing registry JSON round-trip coverage that replaces the remaining Python-only round-trip assertion.
3. Run targeted Go validation for `./internal/repo`, `./internal/risk`, and `./internal/workflow`, plus repo-level file-count checks.
4. Commit the scoped migration changes and push the branch to the remote.

## Acceptance

- The number of Python files under `tests/` decreases in this issue scope.
- Any deleted Python test has a checked-in Go replacement test in `bigclaw-go/`.
- No new Python tests are introduced.
- `pyproject.toml` and `setup.py` remain absent.
- The final change can name the deleted Python files and the added or expanded Go test files.

## Validation

- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
- `cd bigclaw-go && go test ./internal/repo ./internal/risk ./internal/workflow`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
- `git status --short`

## Validation Results

- `cd bigclaw-go && go test ./internal/repo ./internal/risk ./internal/workflow`
  - `ok  	bigclaw-go/internal/repo	1.151s`
  - `ok  	bigclaw-go/internal/risk	2.022s`
  - `ok  	bigclaw-go/internal/workflow	1.574s`
- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
  - `23`
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
