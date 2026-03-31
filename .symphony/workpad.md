Issue: BIG-GO-1024

Plan
- Fold `console_ia.py` into `src/bigclaw/__init__.py` as a dynamic support submodule while preserving `bigclaw.console_ia` imports.
- Delete that standalone Python file from `src/bigclaw`.
- Run targeted console IA validation plus file-count checks, then commit and push the issue branch.

Acceptance
- `src/bigclaw/console_ia.py` is removed, reducing physical Python file count in `src/bigclaw`.
- `bigclaw.console_ia` remains importable and preserves the tested IA and interaction-audit behavior.
- No `pyproject.toml`, `setup.py`, or `setup.cfg` changes are introduced unless directly required by this tranche.

Validation
- `find src/bigclaw -maxdepth 1 -name '*.py' | sort | wc -l`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `python3 -m py_compile src/bigclaw/__init__.py tests/test_console_ia.py`
- `PYTHONPATH=src python3 -m pytest tests/test_console_ia.py tests/test_design_system.py -q`
- `PYTHONPATH=src python3 - <<'PY'`
- `import bigclaw.console_ia`
- `print("module smoke checks passed")`
- `PY`
- `git status --short`
- `git diff --stat`
