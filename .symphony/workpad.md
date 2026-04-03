# BIG-GO-1026 Workpad

## Plan
- Extend `bigclaw-go/internal/reporting` with the remaining takeover queue and orchestration canvas renderer behavior still covered only in Python.
- Add Go-native aggregate helpers for `TakeoverQueue` and `OrchestrationCanvas` that match the Python contracts used by the current tests.
- Add focused Go coverage for the takeover grouping, shared-view error rendering, and orchestration canvas summary assertions still living in `tests/test_reports.py`.
- Remove only the matching takeover/canvas Python tests from `tests/test_reports.py` after Go-native coverage is in place.
- Re-run the targeted reports pytest file and the Go tests for `./internal/reporting`.
- Capture the updated repo inventory and confirm `pyproject.toml` / `setup.py` / `setup.cfg` remain unchanged.
- Commit and push the follow-up reduction on `BIG-GO-1026`.

## Acceptance
- Scope stays limited to the takeover queue and orchestration canvas contracts currently exercised by `test_takeover_queue_from_ledger_groups_pending_handoffs`, `test_takeover_queue_report_renders_shared_view_error_state`, and `test_orchestration_canvas_summarizes_policy_and_handoff` in `tests/test_reports.py`.
- Go-native coverage in `bigclaw-go/internal/reporting` becomes the source of truth for those takeover/canvas contracts.
- `tests/test_reports.py` shrinks while the consolidated suite still passes.
- Report includes `.py` / `.go` file-count impact and confirms whether `pyproject.toml` / `setup.py` / `setup.cfg` changed.

## Validation
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py -q`
- `go test ./internal/reporting`
- `wc -l tests/test_reports.py`
- `git diff --stat`
- `rg --files | rg '\\.py$' | wc -l`
- `rg --files | rg '\\.go$' | wc -l`
- `rg --files | rg '(^|/)(pyproject\\.toml|setup\\.py|setup\\.cfg)$'`

## Validation Results
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py -q`
  `57 passed in 0.18s`
- `go test ./internal/reporting` (run from `bigclaw-go/`)
  `ok  	bigclaw-go/internal/reporting	0.455s`
- `wc -l tests/test_reports.py`
  `2504 tests/test_reports.py`
- `git diff --stat`
  `.symphony/workpad.md | 14 +-\n  bigclaw-go/internal/reporting/reporting.go | 222 ++++++++++++++++++++++--\n  bigclaw-go/internal/reporting/reporting_test.go | 182 +++++++++++++++++++\n  tests/test_reports.py | 126 --------------\n  4 files changed, 394 insertions(+), 150 deletions(-)`
- `rg --files | rg '\.py$' | wc -l`
  `51`
- `rg --files | rg '\.go$' | wc -l`
  `288`
- `rg --files | rg '(^|/)(pyproject\.toml|setup\.py|setup\.cfg)$'`
  no matches; no `pyproject.toml`, `setup.py`, or `setup.cfg` paths were added or changed in this workspace slice.
