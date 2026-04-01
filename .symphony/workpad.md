# BIG-GO-1038 Workpad

## Plan

1. Extend `internal/designsystem` to cover the behavior from `tests/test_design_system.py`,
   including component-library auditing, console top-bar auditing, information-architecture
   auditing, UI-acceptance auditing, and the corresponding reports, then delete that Python test.
2. Preserve compatibility with the newly added console IA tranche while expanding the shared
   design-system package rather than creating another overlapping UI package.
3. Run targeted Go validation for `./internal/designsystem`, plus repo-level file-count checks.
4. Commit the scoped migration changes and push the branch to the remote.

## Acceptance

- The number of Python files under `tests/` decreases again in this tranche.
- Deleted Python tests are covered by checked-in or expanded Go tests in `bigclaw-go/`.
- No new Python tests are introduced.
- `pyproject.toml` and `setup.py` remain absent.
- The final change can name the deleted Python files and the added or expanded Go test files.

## Validation

- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
- `cd bigclaw-go && go test ./internal/designsystem`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
- `git status --short`

## Validation Results
- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
  - Result: `4`
- `cd bigclaw-go && go test ./internal/designsystem`
  - Result:
    - `ok  	bigclaw-go/internal/designsystem	0.471s`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
  - Result: no output
