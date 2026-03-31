Issue: BIG-GO-1024

Plan
- Fold `execution_contract.py` into `src/bigclaw/__init__.py` as a dynamic support submodule while preserving `bigclaw.execution_contract` imports.
- Delete that standalone Python file from `src/bigclaw`.
- Run targeted execution-contract validation plus file-count checks, then commit and push the issue branch.

Acceptance
- `src/bigclaw/execution_contract.py` is removed, reducing physical Python file count in `src/bigclaw`.
- `bigclaw.execution_contract` remains importable and preserves the tested contract, permission, audit, and report behavior.
- No `pyproject.toml`, `setup.py`, or `setup.cfg` changes are introduced unless directly required by this tranche.

Validation
- `find src/bigclaw -maxdepth 1 -name '*.py' | sort | wc -l`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `python3 -m py_compile src/bigclaw/__init__.py tests/test_execution_contract.py`
- `PYTHONPATH=src python3 -m pytest tests/test_execution_contract.py tests/test_reports.py -q`
- `PYTHONPATH=src python3 - <<'PY'`
- `import bigclaw.execution_contract`
- `print("module smoke checks passed")`
- `PY`
- `git status --short`
- `git diff --stat`
