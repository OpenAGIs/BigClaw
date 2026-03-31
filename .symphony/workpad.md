# BIG-GO-1026 Workpad

## Plan
- Inline the remaining `tests/conftest.py` path bootstrap into the surviving reports suite so the test layer keeps the same behavior with one fewer Python file.
- Remove `tests/conftest.py` after confirming direct `pytest` and `PYTHONPATH` invocations still collect the consolidated suite cleanly.
- Run targeted Python validation for the consolidated reports suite plus repo-level grep/count checks, then record exact commands and results.
- Commit the scoped changes and push the branch to the remote.

## Acceptance
- Scope stays limited to the `tests/conftest.py` consolidation tranche for this issue.
- `.py` file count decreases from the current baseline.
- Coverage formerly provided by `tests/conftest.py` now lives in `tests/test_reports.py`.
- Any references to `tests/conftest.py` are updated or eliminated.
- Report includes `.py` / `.go` file-count impact and confirms whether `pyproject.toml` / `setup.py` / `setup.cfg` changed.

## Validation
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py -q`
- `python3 -m pytest tests/test_reports.py -q`
- `rg -n "conftest\\.py|tests/conftest\\.py" README.md scripts tests src .symphony`
- `rg --files | rg '\\.py$' | wc -l`
- `rg --files | rg '\\.go$' | wc -l`
- `rg --files | rg '(^|/)(pyproject\\.toml|setup\\.py|setup\\.cfg)$'`
