# BIG-GO-1038 Workpad

## Plan

1. Add Go-native `designsystem` and `consoleia` packages to cover the behavior from
   `tests/test_console_ia.py`, including console IA manifests, top-bar auditing, interaction-draft
   auditing, static BIG-4203 draft construction, and report rendering, then delete that Python
   test.
2. Keep the shared design-system types limited to the console IA tranche so future remaining UI
   migrations can build on them without widening this diff unnecessarily.
3. Run targeted Go validation for `./internal/designsystem` and `./internal/consoleia`, plus
   repo-level file-count checks.
4. Commit the scoped migration changes and push the branch to the remote.

## Acceptance

- The number of Python files under `tests/` decreases again in this tranche.
- Deleted Python tests are covered by checked-in or expanded Go tests in `bigclaw-go/`.
- No new Python tests are introduced.
- `pyproject.toml` and `setup.py` remain absent.
- The final change can name the deleted Python files and the added or expanded Go test files.

## Validation

- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
- `cd bigclaw-go && go test ./internal/designsystem ./internal/consoleia`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
- `git status --short`

## Validation Results
- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
  - Result: `5`
- `cd bigclaw-go && go test ./internal/designsystem ./internal/consoleia`
  - Result:
    - `ok  	bigclaw-go/internal/designsystem	0.414s`
    - `ok  	bigclaw-go/internal/consoleia	0.838s`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
  - Result: no output
