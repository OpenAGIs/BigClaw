# BIG-GO-1026 Workpad

## Plan
- Remove the remaining Python engineering-overview permission-rendering test from `tests/test_reports.py`, keeping scope limited to the duplicate contract already covered in Go.
- Re-run the targeted reports pytest file and the Go reporting package tests.
- Capture the updated repo inventory and confirm `pyproject.toml` / `setup.py` / `setup.cfg` remain unchanged.
- Commit and push the follow-up reduction on `BIG-GO-1026`.
- Re-run the consolidated reports suite and the matching Go package tests, then capture repo-level inventory checks so the branch records the reduced Python test footprint.
- Commit the scoped semantic reduction and push the branch to the remote.

## Acceptance
- Scope stays limited to removing the Python engineering-overview permission-rendering duplicate.
- Go-native coverage in `bigclaw-go/internal/reporting/reporting_test.go` remains the source of truth for that contract.
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
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py -q` -> `144 passed in 0.44s`
- `go test ./internal/reporting` (run from `bigclaw-go/`) -> `ok  	bigclaw-go/internal/reporting	(cached)`
- `wc -l tests/test_reports.py` -> `5652 tests/test_reports.py`
- `rg --files | rg '\\.py$' | wc -l` -> `51`
- `rg --files | rg '\\.go$' | wc -l` -> `284`
- `rg --files | rg '(^|/)(pyproject\\.toml|setup\\.py|setup\\.cfg)$' || true` -> no matches; no `pyproject.toml`, `setup.py`, or `setup.cfg` files were touched in this change
