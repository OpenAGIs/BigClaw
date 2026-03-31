# BIG-GO-1026 Workpad

## Plan
- Identify the next bounded slice inside `tests/test_reports.py` with direct Go-native reporting coverage.
- Remove the Python operations metric / dashboard builder / engineering overview tests whose behaviors are already covered in `bigclaw-go/internal/reporting/reporting_test.go`.
- Re-run the consolidated reports suite and repo-level inventory checks so the branch captures the reduced Python test footprint.
- Commit the scoped semantic reduction and push the branch to the remote.

## Acceptance
- Scope stays limited to the operations metric / dashboard builder / engineering overview slice removed from `tests/test_reports.py`.
- Python test coverage for that slice is carried by existing Go-native reporting tests in `bigclaw-go/internal/reporting/reporting_test.go`.
- `tests/test_reports.py` shrinks while the consolidated suite still passes.
- Report includes `.py` / `.go` file-count impact and confirms whether `pyproject.toml` / `setup.py` / `setup.cfg` changed.

## Validation
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py -q`
- `wc -l tests/test_reports.py`
- `git diff --stat`
- `rg --files | rg '\\.py$' | wc -l`
- `rg --files | rg '\\.go$' | wc -l`
- `rg --files | rg '(^|/)(pyproject\\.toml|setup\\.py|setup\\.cfg)$'`
