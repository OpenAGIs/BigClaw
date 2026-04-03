# BIG-GO-1026 Workpad

## Plan
- Extend `bigclaw-go/internal/reporting` with the remaining pilot portfolio reporting contract still covered only in Python.
- Add Go-native pilot metric, scorecard, and portfolio helpers plus the markdown portfolio renderer needed by the current test.
- Add focused Go coverage for `test_render_pilot_portfolio_report_summarizes_commercial_readiness`.
- Remove only the matching Python pilot portfolio test from `tests/test_reports.py`.
- Re-run the targeted reports pytest file and the Go tests for `./internal/reporting`.
- Capture the updated repo inventory and confirm `pyproject.toml` / `setup.py` / `setup.cfg` remain unchanged.
- Commit and push the follow-up reduction on `BIG-GO-1026`.

## Acceptance
- Scope stays limited to the remaining pilot portfolio reporting contract currently exercised by the matching `tests/test_reports.py` case.
- Go-native coverage in `bigclaw-go/internal/reporting` becomes the source of truth for that pilot portfolio contract.
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
  `44 passed in 0.15s`
- `go test ./internal/reporting` (run from `bigclaw-go/`)
  `ok  	bigclaw-go/internal/reporting	1.269s`
- `wc -l tests/test_reports.py`
  `1797 tests/test_reports.py`
- `git diff --stat`
  `.symphony/workpad.md | 12 +-\n  bigclaw-go/internal/reporting/reporting.go | 148 ++++++++++++++++++++++++\n  bigclaw-go/internal/reporting/reporting_test.go | 62 ++++++++++\n  tests/test_reports.py | 41 -------\n  4 files changed, 216 insertions(+), 47 deletions(-)`
- `rg --files | rg '\.py$' | wc -l`
  `51`
- `rg --files | rg '\.go$' | wc -l`
  `288`
- `rg --files | rg '(^|/)(pyproject\.toml|setup\.py|setup\.cfg)$'`
  no matches; no `pyproject.toml`, `setup.py`, or `setup.cfg` paths were added or changed in this workspace slice.
