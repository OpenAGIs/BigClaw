# BIG-GO-1026 Workpad

## Plan
- Fold the `workspace_bootstrap` Python tests into the adjacent planning suite so the repo keeps the same Python coverage with one fewer test file.
- Remove `tests/test_workspace_bootstrap.py` and update bootstrap docs/scripts that still reference the deleted file.
- Run targeted Python validation for the consolidated planning suite plus repo-level grep/count checks, then record exact commands and results.
- Commit the scoped changes and push the branch to the remote.

## Acceptance
- Scope stays limited to the `tests/test_workspace_bootstrap.py` consolidation tranche for this issue.
- `.py` file count decreases from the current baseline.
- Coverage formerly in `tests/test_workspace_bootstrap.py` now lives in `tests/test_planning.py`.
- Any references to `tests/test_workspace_bootstrap.py` are updated or eliminated.
- Report includes `.py` / `.go` file-count impact and confirms whether `pyproject.toml` / `setup.py` / `setup.cfg` changed.

## Validation
- `PYTHONPATH=src python3 -m pytest tests/test_planning.py -q`
- `python3 -m pytest tests/test_planning.py -q`
- `rg -n "test_workspace_bootstrap\\.py" README.md scripts tests src .symphony`
- `rg --files | rg '\\.py$' | wc -l`
- `rg --files | rg '\\.go$' | wc -l`
- `rg --files | rg '(^|/)(pyproject\\.toml|setup\\.py|setup\\.cfg)$'`
