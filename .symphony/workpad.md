Issue: BIG-GO-1024

Plan
- Fold `workspace_bootstrap.py` into `src/bigclaw/__init__.py` as a dynamic support submodule while preserving `bigclaw.workspace_bootstrap` imports.
- Delete that standalone Python file from `src/bigclaw`.
- Run targeted workspace bootstrap validation plus file-count checks, then commit and push the issue branch.

Acceptance
- `src/bigclaw/workspace_bootstrap.py` is removed, reducing physical Python file count in `src/bigclaw`.
- `bigclaw.workspace_bootstrap` remains importable and preserves the tested bootstrap and cleanup behavior.
- No `pyproject.toml`, `setup.py`, or `setup.cfg` changes are introduced unless directly required by this tranche.

Validation
- `find src/bigclaw -maxdepth 1 -name '*.py' | sort | wc -l`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `python3 -m py_compile src/bigclaw/__init__.py tests/test_workspace_bootstrap.py`
- `PYTHONPATH=src python3 -m pytest tests/test_workspace_bootstrap.py -q`
- `PYTHONPATH=src python3 - <<'PY'`
- `import bigclaw.workspace_bootstrap`
- `print("module smoke checks passed")`
- `PY`
- `git status --short`
- `git diff --stat`
