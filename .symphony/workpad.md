# BIG-GO-1026 Workpad

## Plan
- Add Go-native coverage for the workspace validation summary slice in `bigclaw-go/internal/bootstrap/bootstrap_test.go`.
- Remove the Python `build_validation_report` summary test from `tests/test_reports.py` once the equivalent Go test exists.
- Re-run the consolidated reports suite and repo-level inventory checks so the branch captures the reduced Python test footprint.
- Commit the scoped semantic reduction and push the branch to the remote.

## Acceptance
- Scope stays limited to the workspace validation summary slice removed from `tests/test_reports.py`.
- Python test coverage for that slice is carried by new Go-native bootstrap coverage in `bigclaw-go/internal/bootstrap/bootstrap_test.go`.
- `tests/test_reports.py` shrinks while the consolidated suite still passes.
- Report includes `.py` / `.go` file-count impact and confirms whether `pyproject.toml` / `setup.py` / `setup.cfg` changed.

## Validation
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py -q`
- `wc -l tests/test_reports.py`
- `git diff --stat`
- `rg --files | rg '\\.py$' | wc -l`
- `rg --files | rg '\\.go$' | wc -l`
- `rg --files | rg '(^|/)(pyproject\\.toml|setup\\.py|setup\\.cfg)$'`
