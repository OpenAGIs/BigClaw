# BIG-GO-1026 Workpad

## Plan
- Extend `bigclaw-go/internal/reporting` with the small issue-validation contract still covered only in Python.
- Add a Go-native `RenderIssueValidationReport` that preserves the existing markdown shape and UTC timestamp contract from `tests/test_reports.py`.
- Add focused Go tests for `WriteReport`, `ConsoleAction.State()`, and `RenderIssueValidationReport` timestamp/content behavior.
- Remove only the matching short Python reporting tests from `tests/test_reports.py` after Go-native coverage is in place.
- Re-run the targeted reports pytest file and the Go tests for `./internal/reporting`.
- Capture the updated repo inventory and confirm `pyproject.toml` / `setup.py` / `setup.cfg` remain unchanged.
- Commit and push the follow-up reduction on `BIG-GO-1026`.

## Acceptance
- Scope stays limited to the short reporting contracts currently exercised by `test_render_and_write_report`, `test_console_action_state_reflects_enabled_flag`, and `test_issue_validation_report_uses_timezone_aware_utc_timestamp` in `tests/test_reports.py`.
- Go-native coverage in `bigclaw-go/internal/reporting` becomes the source of truth for those contracts.
- `tests/test_reports.py` shrinks while the consolidated suite still passes.
- Report includes `.py` / `.go` file-count impact and confirms whether `pyproject.toml` / `setup.py` / `setup.cfg` changed.

## Validation
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py -q`
- `go test ./internal/uigovernance`
- `wc -l tests/test_reports.py`
- `git diff --stat`
- `rg --files | rg '\\.py$' | wc -l`
- `rg --files | rg '\\.go$' | wc -l`
- `rg --files | rg '(^|/)(pyproject\\.toml|setup\\.py|setup\\.cfg)$'`

## Validation Results
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py -q`
  `64 passed in 0.19s`
- `go test ./internal/reporting` (run from `bigclaw-go/`)
  `ok  	bigclaw-go/internal/reporting	1.540s`
- `wc -l tests/test_reports.py`
  `2733 tests/test_reports.py`
- `git diff --stat`
  `.symphony/workpad.md | 15 ++++----`
  `bigclaw-go/internal/reporting/reporting.go | 10 +++++`
  `bigclaw-go/internal/reporting/reporting_test.go | 50 +++++++++++++++++++++++++`
  `tests/test_reports.py | 29 --------------`
  `4 files changed, 67 insertions(+), 37 deletions(-)`
- `rg --files | rg '\.py$' | wc -l`
  `51`
- `rg --files | rg '\.go$' | wc -l`
  `288`
- `rg --files | rg '(^|/)(pyproject\.toml|setup\.py|setup\.cfg)$'`
  no matches; no `pyproject.toml`, `setup.py`, or `setup.cfg` paths were added or changed in this workspace slice.
