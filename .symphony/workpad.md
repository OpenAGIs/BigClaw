# BIG-GO-1026 Workpad

## Plan
- Extend `bigclaw-go/internal/reporting` with the remaining pilot scorecard renderer contract still covered only in Python.
- Reuse the existing Go pilot types to add the markdown scorecard renderer and focused assertions around payoff/recommendation behavior.
- Add focused Go coverage for the remaining pilot scorecard tests in `tests/test_reports.py`.
- Remove only the matching Python pilot scorecard tests from `tests/test_reports.py`.
- Re-run the targeted reports pytest file and the Go tests for `./internal/reporting`.
- Capture the updated repo inventory and confirm `pyproject.toml` / `setup.py` / `setup.cfg` remain unchanged.
- Commit and push the follow-up reduction on `BIG-GO-1026`.

## Acceptance
- Scope stays limited to the remaining pilot scorecard reporting contracts currently exercised by the matching `tests/test_reports.py` cases.
- Go-native coverage in `bigclaw-go/internal/reporting` becomes the source of truth for those pilot scorecard contracts.
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
  `42 passed in 0.15s`
- `go test ./internal/reporting` (run from `bigclaw-go/`)
  `ok  	bigclaw-go/internal/reporting	1.387s`
- `wc -l tests/test_reports.py`
  `1740 tests/test_reports.py`
- `git diff --stat`
  `.symphony/workpad.md | 12 ++---\n  bigclaw-go/internal/reporting/reporting.go | 40 +++++++++++++++\n  bigclaw-go/internal/reporting/reporting_test.go | 66 +++++++++++++++++++++++++\n  tests/test_reports.py | 57 ---------------------\n  4 files changed, 112 insertions(+), 63 deletions(-)`
- `rg --files | rg '\.py$' | wc -l`
  `51`
- `rg --files | rg '\.go$' | wc -l`
  `288`
- `rg --files | rg '(^|/)(pyproject\.toml|setup\.py|setup\.cfg)$'`
  no matches; no `pyproject.toml`, `setup.py`, or `setup.cfg` paths were added or changed in this workspace slice.
