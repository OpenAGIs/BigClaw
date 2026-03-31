# BIG-GO-1026 Workpad

## Plan
- Fold the `design_system` Python tests into the adjacent reports suite so the repo keeps the same coverage with one fewer test file.
- Remove `tests/test_design_system.py` and update any in-repo references that still point to the deleted file.
- Run targeted Python validation for the consolidated reports suite plus repo-level grep/count checks, then record exact commands and results.
- Commit the scoped changes and push the branch to the remote.

## Acceptance
- Scope stays limited to the `tests/test_design_system.py` consolidation tranche for this issue.
- `.py` file count decreases from the current baseline.
- Coverage formerly in `tests/test_design_system.py` now lives in `tests/test_reports.py`.
- Any references to `tests/test_design_system.py` are updated or eliminated.
- Report includes `.py` / `.go` file-count impact and confirms whether `pyproject.toml` / `setup.py` / `setup.cfg` changed.

## Validation
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py -q`
- `python3 -m pytest tests/test_reports.py -q`
- `rg -n "test_design_system\\.py" README.md scripts tests src .symphony`
- `rg --files | rg '\\.py$' | wc -l`
- `rg --files | rg '\\.go$' | wc -l`
- `rg --files | rg '(^|/)(pyproject\\.toml|setup\\.py|setup\\.cfg)$'`
