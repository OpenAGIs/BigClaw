Issue: BIG-GO-1024

Plan
- Fold `observability.py` into `src/bigclaw/__init__.py` as a dynamic support submodule while preserving `bigclaw.observability` imports.
- Delete that standalone Python file from `src/bigclaw`.
- Run targeted observability validation plus file-count checks, then commit and push the issue branch.

Acceptance
- `src/bigclaw/observability.py` is removed, reducing physical Python file count in `src/bigclaw`.
- `bigclaw.observability` remains importable and preserves the tested ledger, closeout, and repo-sync behavior.
- No `pyproject.toml`, `setup.py`, or `setup.cfg` changes are introduced unless directly required by this tranche.

Validation
- `find src/bigclaw -maxdepth 1 -name '*.py' | sort | wc -l`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `python3 -m py_compile src/bigclaw/__init__.py tests/test_observability.py`
- `PYTHONPATH=src python3 -m pytest tests/test_observability.py tests/test_repo_links.py tests/test_reports.py -q`
- `PYTHONPATH=src python3 - <<'PY'`
- `import bigclaw.observability`
- `print("module smoke checks passed")`
- `PY`
- `git status --short`
- `git diff --stat`
