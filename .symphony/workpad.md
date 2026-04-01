# BIG-GO-1038 Workpad

## Plan

1. Add a small Go-native event bus package with a file-backed run ledger that covers the remaining
   PR approval, CI completion, and task-failure transition tests, then delete
   `tests/test_event_bus.py`.
2. Add a small Go-native evaluation package that covers benchmark case execution, replay mismatch
   detection, suite comparison, and the replay/detail page renderers, then delete
   `tests/test_evaluation.py`.
3. Run targeted Go validation for `./internal/eventbus` and `./internal/evaluation`, plus
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
- `cd bigclaw-go && go test ./internal/eventbus ./internal/evaluation`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
- `git status --short`

## Validation Results

- `cd bigclaw-go && go test ./internal/eventbus ./internal/evaluation`
  - `ok  	bigclaw-go/internal/eventbus	0.799s`
  - `ok  	bigclaw-go/internal/evaluation	0.395s`
- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
  - `11`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
  - no output
