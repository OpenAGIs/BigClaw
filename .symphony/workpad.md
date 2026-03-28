# BIG-GO-923

## Plan

1. Reconfirm the live pytest/conftest surface under `tests/`, `pyproject.toml`, `src/bigclaw/ui_review.py`, and `bigclaw-go/internal/testharness`.
2. Complete the Go-native `bigclaw-go/internal/uireview` surface needed to replace `tests/test_ui_review.py`, including the BIG-4204 fixture/builder and the missing board/log renderers used by the legacy test pack.
3. Add Go coverage for the migrated UI review contracts, then delete `tests/test_ui_review.py` once the Go suite protects the same slice.
4. Refresh `bigclaw-go/docs/reports/pytest-harness-status.json` and `bigclaw-go/docs/reports/pytest-harness-migration.md` to the post-migration repo state and deletion conditions.
5. Run targeted validation, record exact commands and outcomes here, then commit and push `BIG-GO-923-go-test-harness`.

## Acceptance

- Current Python and non-Go pytest harness assets are explicitly inventoried against the live worktree.
- A Go-native replacement exists for the remaining pytest harness consumer that depended on the legacy Python bootstrap.
- The first-batch Go implementation for the UI review slice is landed and the corresponding Python pytest module is removed.
- Conditions for deleting old Python harness assets and the regression verification commands are documented.
- Exact validation commands and outcomes are recorded in this workpad and reflected in the final closeout.

## Validation

- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/uireview`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/testharness ./internal/regression ./cmd/bigclawctl`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go run ./cmd/bigclawctl pytest-harness --project-root .. --report-path docs/reports/pytest-harness-status.json --json`

## Validation Results

- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/uireview`
  - `ok  	bigclaw-go/internal/uireview`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/testharness ./internal/regression ./cmd/bigclawctl`
  - `ok  	bigclaw-go/internal/testharness`
  - `ok  	bigclaw-go/internal/regression`
  - `ok  	bigclaw-go/cmd/bigclawctl`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go run ./cmd/bigclawctl pytest-harness --project-root .. --report-path docs/reports/pytest-harness-status.json --json`
  - `status=ok`
  - `inventory_summary=tests=0 bigclaw_imports=0 pytest_imports=0 pytest_command_refs=0`
  - `conftest_delete_ready=true blockers=none`
  - `legacy_pytest_delete_ready=true blockers=none`

## Notes

- Live inventory at closeout: `tests/conftest.py` remains deleted; `pyproject.toml` still excludes pytest from the default baseline; `tests/` now contains no remaining pytest modules.
- `tests/test_ui_review.py` has been replaced by Go-owned coverage in `bigclaw-go/internal/uireview/uireview_test.go`, together with Go-native BIG-4204 builder/report/bundle implementations under `bigclaw-go/internal/uireview`.
- Keep scope constrained to the pytest/conftest migration surface for this issue.
- Execution update: migrated the remaining `tests/test_ui_review.py` contract into `bigclaw-go/internal/uireview` and refreshed the pytest-harness snapshot/tests from `tests=1` to `tests=0`.
