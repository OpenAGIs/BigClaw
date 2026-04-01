# BIG-GO-1026 Workpad

## Plan
- Add Go-native coverage for the remaining dashboard-builder round-trip and engineering-overview permission-rendering contracts in `bigclaw-go/internal/reporting/reporting_test.go`.
- Remove the matching Python tests from `tests/test_reports.py`.
- Re-run the consolidated reports suite and the matching Go package tests, then capture repo-level inventory checks so the branch records the reduced Python test footprint.
- Commit the scoped semantic reduction and push the branch to the remote.

## Acceptance
- Scope stays limited to the dashboard-builder round-trip and engineering-overview permission-rendering slices migrated out of `tests/test_reports.py`.
- Go-native coverage in `bigclaw-go/internal/reporting/reporting_test.go` carries both migrated contracts.
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
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py -q` -> `145 passed in 0.21s`
- `go test ./internal/reporting` (run from `bigclaw-go/`) -> `ok  	bigclaw-go/internal/reporting	1.406s`
- `wc -l tests/test_reports.py` -> `5685 tests/test_reports.py`
- `git diff --stat` -> `.symphony/workpad.md | 8 +-, bigclaw-go/internal/reporting/reporting_test.go | 111 ++++++++++++++++++++++++, tests/test_reports.py | 39 ---------`
- `rg --files | rg '\\.py$' | wc -l` -> `51`
- `rg --files | rg '\\.go$' | wc -l` -> `284`
- `rg --files | rg '(^|/)(pyproject\\.toml|setup\\.py|setup\\.cfg)$' || true` -> no matches; no `pyproject.toml`, `setup.py`, or `setup.cfg` files were touched in this change
