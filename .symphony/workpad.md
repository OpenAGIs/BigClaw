# BIG-GO-1026 Workpad

## Plan
- Extend `bigclaw-go/internal/reporting` with the small ledger-to-orchestration contract still covered only in Python.
- Add Go-native `BuildOrchestrationCanvasFromLedgerEntry` and `BuildTakeoverQueueFromLedger` helpers that preserve canonical manual-takeover/handoff parsing and approval propagation.
- Add focused Go coverage for the canonical handoff/takeover event path exercised by `test_reports_accept_canonical_handoff_and_takeover_events`.
- Remove only the matching Python handoff/takeover contract test from `tests/test_reports.py` after Go-native coverage is in place.
- Re-run the targeted reports pytest file and the Go tests for `./internal/reporting`.
- Capture the updated repo inventory and confirm `pyproject.toml` / `setup.py` / `setup.cfg` remain unchanged.
- Commit and push the follow-up reduction on `BIG-GO-1026`.

## Acceptance
- Scope stays limited to the canonical handoff/takeover reporting contract currently exercised by `test_reports_accept_canonical_handoff_and_takeover_events` in `tests/test_reports.py`.
- Go-native coverage in `bigclaw-go/internal/reporting` becomes the source of truth for that ledger parsing contract.
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
  `63 passed in 0.19s`
- `go test ./internal/reporting` (run from `bigclaw-go/`)
  `ok  	bigclaw-go/internal/reporting	1.071s`
- `wc -l tests/test_reports.py`
  `2694 tests/test_reports.py`
- `git diff --stat`
  `.symphony/workpad.md | 12 +-`
  `bigclaw-go/internal/reporting/reporting.go | 149 ++++++++++++++++++++++++`
  `bigclaw-go/internal/reporting/reporting_test.go | 44 +++++++`
  `tests/test_reports.py | 39 -------`
  `4 files changed, 199 insertions(+), 45 deletions(-)`
- `rg --files | rg '\.py$' | wc -l`
  `51`
- `rg --files | rg '\.go$' | wc -l`
  `288`
- `rg --files | rg '(^|/)(pyproject\.toml|setup\.py|setup\.cfg)$'`
  no matches; no `pyproject.toml`, `setup.py`, or `setup.cfg` paths were added or changed in this workspace slice.
