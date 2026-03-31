# BIG-GO-1026 Workpad

## Plan
- Add Go-native `DashboardBuilder` round-trip coverage in `bigclaw-go/internal/reporting/reporting_test.go`.
- Remove the Python dashboard-builder round-trip test from `tests/test_reports.py`.
- Re-run the consolidated reports suite and the matching Go package tests, then capture repo-level inventory checks so the branch records the reduced Python test footprint.
- Commit the scoped semantic reduction and push the branch to the remote.

## Acceptance
- Scope stays limited to the dashboard-builder round-trip slice migrated out of `tests/test_reports.py`.
- Go-native coverage in `bigclaw-go/internal/reporting/reporting_test.go` carries the dashboard-builder round-trip contract.
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
