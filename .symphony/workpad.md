Issue: BIG-GO-1024

Plan
- Fold `runtime.py` into `src/bigclaw/__init__.py` as a dynamic compatibility submodule while preserving `bigclaw.runtime` imports and dependent legacy surfaces.
- Delete that standalone Python file from `src/bigclaw`.
- Run targeted runtime validation plus file-count checks, then commit and push the issue branch.

Acceptance
- `src/bigclaw/runtime.py` is removed, reducing physical Python file count in `src/bigclaw`.
- `bigclaw.runtime` remains importable and preserves the tested worker-runtime, scheduler, workflow, service, and governance compatibility behavior.
- No `pyproject.toml`, `setup.py`, or `setup.cfg` changes are introduced unless directly required by this tranche.

Validation
- `find src/bigclaw -maxdepth 1 -name '*.py' | sort | wc -l`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `python3 -m py_compile src/bigclaw/__init__.py tests/test_runtime_matrix.py`
- `PYTHONPATH=src python3 -m pytest tests/test_runtime_matrix.py -q`
- `PYTHONPATH=src python3 - <<'PY'`
- `import bigclaw.runtime`
- `from bigclaw.service import create_server`
- `from bigclaw.scheduler import Scheduler`
- `print("module smoke checks passed", bool(create_server), bool(Scheduler))`
- `PY`
- `git status --short`
- `git diff --stat`
