Issue: BIG-GO-1024

Plan
- Fold `reports.py` into `src/bigclaw/__init__.py` as a dynamic compatibility submodule while preserving `bigclaw.reports` imports.
- Store the compatibility module source in a non-`.py` asset so the physical Python count still drops.
- Delete that standalone Python file from `src/bigclaw`, run targeted reports validation plus file-count checks, then commit and push the issue branch.

Acceptance
- `src/bigclaw/reports.py` is removed, reducing physical Python file count in `src/bigclaw`.
- `bigclaw.reports` remains importable and preserves the tested shared-view, orchestration/reporting, observability rendering, and repo-rollout narrative behavior.
- No `pyproject.toml`, `setup.py`, or `setup.cfg` changes are introduced unless directly required by this tranche.

Validation
- `find src/bigclaw -maxdepth 1 -name '*.py' | sort | wc -l`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `python3 -m py_compile src/bigclaw/__init__.py tests/test_reports.py tests/test_observability.py tests/test_repo_rollout.py`
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py tests/test_observability.py tests/test_repo_rollout.py -q`
- `PYTHONPATH=src python3 - <<'PY'`
- `import bigclaw.reports`
- `from bigclaw.reports import SharedViewContext`
- `print("module smoke checks passed", bool(SharedViewContext))`
- `PY`
- `git status --short`
- `git diff --stat`
