# BIG-GO-1026 Workpad

## Plan
- Add focused Go coverage for the remaining orchestration ledger extraction contracts that are now already implemented in `internal/reporting`.
- Expand Go assertions around `BuildOrchestrationCanvasFromLedgerEntry` and `BuildOrchestrationPortfolioFromLedger` to match the Python tests exactly.
- Remove only the matching orchestration ledger Python tests from `tests/test_reports.py`.
- Re-run the targeted reports pytest file and the Go tests for `./internal/reporting`.
- Capture the updated repo inventory and confirm `pyproject.toml` / `setup.py` / `setup.cfg` remain unchanged.
- Commit and push the follow-up reduction on `BIG-GO-1026`.

## Acceptance
- Scope stays limited to the remaining orchestration ledger extraction contracts currently exercised by the matching `tests/test_reports.py` cases.
- Go-native coverage in `bigclaw-go/internal/reporting` becomes the source of truth for those ledger-extraction contracts.
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
  `49 passed in 0.18s`
- `go test ./internal/reporting` (run from `bigclaw-go/`)
  `ok  	bigclaw-go/internal/reporting	0.976s`
- `wc -l tests/test_reports.py`
  `2038 tests/test_reports.py`
- `git diff --stat`
  `.symphony/workpad.md | 11 +-\n  bigclaw-go/internal/reporting/reporting_test.go | 167 ++++++++++++++++++++++++\n  tests/test_reports.py | 143 --------------------\n  3 files changed, 172 insertions(+), 149 deletions(-)`
- `rg --files | rg '\.py$' | wc -l`
  `51`
- `rg --files | rg '\.go$' | wc -l`
  `288`
- `rg --files | rg '(^|/)(pyproject\.toml|setup\.py|setup\.cfg)$'`
  no matches; no `pyproject.toml`, `setup.py`, or `setup.cfg` paths were added or changed in this workspace slice.
