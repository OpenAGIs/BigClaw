# BIG-GO-1026 Workpad

## Plan
- Add Go-native dashboard-layout normalization coverage in `bigclaw-go/internal/reporting`, then remove the matching Python test from `tests/test_reports.py`.
- Re-run the targeted reports pytest file and the Go reporting package tests.
- Capture the updated repo inventory and confirm `pyproject.toml` / `setup.py` / `setup.cfg` remain unchanged.
- Commit and push the follow-up reduction on `BIG-GO-1026`.

## Acceptance
- Scope stays limited to the dashboard-layout normalization contract migrated out of `tests/test_reports.py`.
- Go-native coverage in `bigclaw-go/internal/reporting/reporting_test.go` becomes the source of truth for that contract.
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
- `gofmt -w bigclaw-go/internal/reporting/reporting.go bigclaw-go/internal/reporting/reporting_test.go` -> completed
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py -q` -> `141 passed in 0.15s`
- `go test ./internal/reporting` (run from `bigclaw-go/`) -> `ok  	bigclaw-go/internal/reporting	0.451s`
- `wc -l tests/test_reports.py` -> `5528 tests/test_reports.py`
- `git diff --stat` -> `.symphony/workpad.md | 12 +---, bigclaw-go/internal/reporting/reporting.go | 77 +++++++++++++++++++++++++, bigclaw-go/internal/reporting/reporting_test.go | 49 ++++++++++++++++, tests/test_reports.py | 46 ---------------`
- `rg --files | rg '\\.py$' | wc -l` -> `51`
- `rg --files | rg '\\.go$' | wc -l` -> `284`
- `rg --files | rg '(^|/)(pyproject\\.toml|setup\\.py|setup\\.cfg)$' || true` -> no matches; no `pyproject.toml`, `setup.py`, or `setup.cfg` files were touched in this change
