# BIG-GO-1026 Workpad

## Plan
- Fold the `console_ia` Python tests into the adjacent design-system suite so the repo keeps the same coverage with one fewer test file.
- Remove `tests/test_console_ia.py` and update planner/report references that still point to the deleted file.
- Run targeted Python validation for the consolidated design-system/planning suites plus repo-level grep/count checks, then record exact commands and results.
- Commit the scoped changes and push the branch to the remote.

## Acceptance
- Scope stays limited to the `tests/test_console_ia.py` consolidation tranche for this issue.
- `.py` file count decreases from the current baseline.
- Coverage formerly in `tests/test_console_ia.py` now lives in `tests/test_design_system.py`.
- Any references to `tests/test_console_ia.py` are updated or eliminated.
- Report includes `.py` / `.go` file-count impact and confirms whether `pyproject.toml` / `setup.py` / `setup.cfg` changed.

## Validation
- `PYTHONPATH=src python3 -m pytest tests/test_design_system.py -q`
- `PYTHONPATH=src python3 -m pytest tests/test_planning.py -q`
- `rg -n "test_console_ia\\.py" README.md scripts tests src .symphony`
- `rg --files | rg '\\.py$' | wc -l`
- `rg --files | rg '\\.go$' | wc -l`
- `rg --files | rg '(^|/)(pyproject\\.toml|setup\\.py|setup\\.cfg)$'`
