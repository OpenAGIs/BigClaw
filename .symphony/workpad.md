# BIG-GO-1038 Workpad

## Plan

1. Keep this continuation tranche scoped to deleting `tests/conftest.py`, which is now only a
   Python import-path shim for the shrinking Python test surface.
2. Expand Go coverage in existing packages only: add focused assertions in
   `bigclaw-go/internal/collaboration` and `bigclaw-go/internal/pilot` for adjacent report-side
   behavior already represented in Go.
3. Run targeted Go validation for `./internal/collaboration` and `./internal/pilot` plus
   repo-level file-count and Python packaging checks, then record exact commands and results here.
4. Commit the scoped migration changes and push the branch to the remote.

## Acceptance

- The number of Python files under `tests/` decreases again in this tranche.
- Deleted Python file `tests/conftest.py` is replaced by expanded Go coverage in existing Go test
  packages while no new Python tests are introduced.
- No new Python tests are introduced.
- `pyproject.toml` and `setup.py` remain absent.
- The final change can name the deleted Python files and the added or expanded Go test files.

## Validation

- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
- `cd bigclaw-go && go test ./internal/collaboration ./internal/pilot`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
- `git status --short`

## Validation Results

- `cd bigclaw-go && go test ./internal/collaboration ./internal/pilot`
  - `ok  	bigclaw-go/internal/collaboration	(cached)`
  - `ok  	bigclaw-go/internal/pilot	0.463s`
- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
  - `2`
- `find . \( -name pyproject.toml -o -name setup.py \) -print | sort`
  - no output
- `git status --short`
  - ` M .symphony/workpad.md`
  - ` M bigclaw-go/internal/collaboration/thread_test.go`
  - ` M bigclaw-go/internal/pilot/report_test.go`
  - ` M docs/BigClaw-AgentHub-Integration-Alignment.md`
  - ` D tests/conftest.py`
