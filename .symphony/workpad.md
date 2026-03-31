# BIG-GO-1026 Workpad

## Plan
- Remove the Python `RiskAssessment` round-trip test from `tests/test_reports.py`.
- Validate that the existing Go-native risk coverage in `bigclaw-go/internal/risk/assessment_test.go` already carries that contract slice.
- Re-run the consolidated reports suite and repo-level inventory checks so the branch captures the reduced Python test footprint.
- Commit the scoped semantic reduction and push the branch to the remote.

## Acceptance
- Scope stays limited to the `RiskAssessment` round-trip slice removed from `tests/test_reports.py`.
- Python test coverage for that slice is carried by existing Go-native risk coverage in `bigclaw-go/internal/risk/assessment_test.go`.
- `tests/test_reports.py` shrinks while the consolidated suite still passes.
- Report includes `.py` / `.go` file-count impact and confirms whether `pyproject.toml` / `setup.py` / `setup.cfg` changed.

## Validation
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py -q`
- `go test ./internal/risk`
- `wc -l tests/test_reports.py`
- `git diff --stat`
- `rg --files | rg '\\.py$' | wc -l`
- `rg --files | rg '\\.go$' | wc -l`
- `rg --files | rg '(^|/)(pyproject\\.toml|setup\\.py|setup\\.cfg)$'`
