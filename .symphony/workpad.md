Issue: BIG-GO-1024

Plan
- Fold `operations.py` into `src/bigclaw/__init__.py` as a dynamic compatibility submodule while preserving `bigclaw.operations` imports.
- Store the compatibility module source in a non-`.py` asset so the physical Python count still drops.
- Delete that standalone Python file from `src/bigclaw`, run targeted operations validation plus file-count checks, then commit and push the issue branch.

Acceptance
- `src/bigclaw/operations.py` is removed, reducing physical Python file count in `src/bigclaw`.
- `bigclaw.operations` remains importable and preserves the tested operations analytics, queue control center, dashboard builder, regression center, and bundle/report behavior.
- No `pyproject.toml`, `setup.py`, or `setup.cfg` changes are introduced unless directly required by this tranche.

Validation
- `find src/bigclaw -maxdepth 1 -name '*.py' | sort | wc -l`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `python3 -m py_compile src/bigclaw/__init__.py tests/test_operations.py tests/test_control_center.py`
- `PYTHONPATH=src python3 -m pytest tests/test_operations.py tests/test_control_center.py -q`
- `PYTHONPATH=src python3 - <<'PY'`
- `import bigclaw.operations`
- `from bigclaw.operations import OperationsAnalytics`
- `print("module smoke checks passed", bool(OperationsAnalytics))`
- `PY`
- `git status --short`
- `git diff --stat`
