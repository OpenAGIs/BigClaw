Issue: BIG-GO-1024

Plan
- Fold `planning.py` into `src/bigclaw/__init__.py` as a dynamic support submodule while preserving `bigclaw.planning` imports.
- Delete that standalone Python file from `src/bigclaw`.
- Run targeted planning and repo-rollout validation plus file-count checks, then commit and push the issue branch.

Acceptance
- `src/bigclaw/planning.py` is removed, reducing physical Python file count in `src/bigclaw`.
- `bigclaw.planning` remains importable and preserves the tested planning, candidate-gate, and execution-plan behavior.
- No `pyproject.toml`, `setup.py`, or `setup.cfg` changes are introduced unless directly required by this tranche.

Validation
- `find src/bigclaw -maxdepth 1 -name '*.py' | sort | wc -l`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `python3 -m py_compile src/bigclaw/__init__.py tests/test_planning.py tests/test_repo_rollout.py`
- `PYTHONPATH=src python3 -m pytest tests/test_planning.py tests/test_repo_rollout.py -q`
- `PYTHONPATH=src python3 - <<'PY'`
- `import bigclaw.planning`
- `print("module smoke checks passed")`
- `PY`
- `git status --short`
- `git diff --stat`
