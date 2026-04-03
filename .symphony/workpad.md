# BIG-GO-1038 Workpad

## Plan

1. Inspect the two remaining Python tests under `tests/` and map them to existing Go coverage.
2. Delete `tests/test_reports.py` after confirming equivalent Go coverage exists or adding the missing
   `_test.go` assertions inside `bigclaw-go/internal/reporting`.
3. Add a Go-native `bigclaw-go/internal/uireview` package and tests to replace the remaining
   `tests/test_ui_review.py` coverage.
4. Delete `tests/test_ui_review.py` after the Go replacement passes.
5. Run targeted validation for touched Go packages, repo Python-file counts, and Python packaging-file
   absence, then record exact commands and outcomes here.
6. Commit only this issue's changes and push the current branch.

## Acceptance

- The number of Python files under `tests/` decreases.
- Any deleted Python coverage is replaced by Go tests in `bigclaw-go/internal/reporting` or another
  directly corresponding Go package.
- No new Python files are introduced.
- `pyproject.toml` and `setup.py` are absent after the change.
- Final notes can name which Python files were removed and which Go test files were added or expanded.

## Validation

- `find tests -maxdepth 1 -name '*.py' | sort`
- `cd bigclaw-go && go test ./internal/reporting ./internal/uireview`
- `find . \( -name pyproject.toml -o -name setup.py \) -print | sort`
- `git status --short`

## Validation Results

- `find tests -maxdepth 1 -name '*.py' | sort`
  - no output
- `cd bigclaw-go && go test ./internal/reporting ./internal/uireview`
  - `ok  	bigclaw-go/internal/reporting	(cached)`
  - `ok  	bigclaw-go/internal/uireview	(cached)`
- `find . \( -name pyproject.toml -o -name setup.py \) -print | sort`
  - no output
- `git status --short`
  - ` M docs/BigClaw-AgentHub-Integration-Alignment.md`
  - ` D tests/test_ui_review.py`
  - `?? bigclaw-go/internal/uireview/`
