Issue: BIG-GO-1024

Plan
- Fold `design_system.py` into `src/bigclaw/__init__.py` as a dynamic support submodule while preserving `bigclaw.design_system` imports.
- Delete that standalone Python file from `src/bigclaw`.
- Run targeted design-system validation plus file-count checks, then commit and push the issue branch.

Acceptance
- `src/bigclaw/design_system.py` is removed, reducing physical Python file count in `src/bigclaw`.
- `bigclaw.design_system` remains importable and preserves the tested design-system, IA, console-top-bar, and UI-acceptance behavior.
- No `pyproject.toml`, `setup.py`, or `setup.cfg` changes are introduced unless directly required by this tranche.

Validation
- `find src/bigclaw -maxdepth 1 -name '*.py' | sort | wc -l`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `python3 -m py_compile src/bigclaw/__init__.py tests/test_design_system.py`
- `PYTHONPATH=src python3 -m pytest tests/test_design_system.py tests/test_console_ia.py -q`
- `PYTHONPATH=src python3 - <<'PY'`
- `import bigclaw.design_system`
- `print("module smoke checks passed")`
- `PY`
- `git status --short`
- `git diff --stat`
