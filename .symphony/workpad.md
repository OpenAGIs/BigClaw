# BIG-GO-964 Workpad

## Plan

1. Inventory the remaining physical Python modules under `src/bigclaw/` and confirm the last removable application-host candidate for this batch.
2. Fold `src/bigclaw/scheduler.py` into `src/bigclaw/execution_contract.py` so the legacy Python application surface keeps a single non-entrypoint implementation host.
3. Update package exports and legacy module aliases in `src/bigclaw/__init__.py`, and redirect `src/bigclaw/__main__.py` to the consolidated host.
4. Delete `src/bigclaw/scheduler.py`, then run targeted import, pytest, and compile validation for the touched surfaces.
5. Recount `src/bigclaw/*.py` and repo-wide `*.py`, record exact commands/results, then commit and push the issue branch.

## Acceptance

- Explicit handled-file list for this batch:
  - `src/bigclaw/scheduler.py`
  - `src/bigclaw/execution_contract.py`
  - `src/bigclaw/__init__.py`
  - `src/bigclaw/__main__.py`
- Reduce the number of Python files under `src/bigclaw/`.
- Document delete / replace / retain reasons for the handled files.
- Report the before/after impact on both `src/bigclaw/*.py` and total repo `*.py` counts.

## Validation

- `PYTHONPATH=src python3 - <<'PY'` import check for:
  - `bigclaw.scheduler`
  - `bigclaw.runtime`
  - `bigclaw.reports`
  - `bigclaw.execution_contract`
- `PYTHONPATH=src python3 -m pytest tests/test_scheduler.py tests/test_runtime.py tests/test_execution_flow.py tests/test_orchestration.py tests/test_operations.py tests/test_evaluation.py tests/test_risk.py tests/test_audit_events.py tests/test_execution_contract.py`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/__main__.py src/bigclaw/execution_contract.py`
- `find src/bigclaw -type f -name '*.py' | wc -l`
- `find . -type f -name '*.py' | wc -l`
- `git status --short`

## Baseline

- `find src/bigclaw -type f -name '*.py' | wc -l`
  - Result: `4`
- `find . -type f -name '*.py' | wc -l`
  - Result: `78`

## Decisions

- `src/bigclaw/scheduler.py`
  - Delete. Its remaining runtime, queue, orchestration, reporting, and compatibility helpers were consolidated into `src/bigclaw/execution_contract.py`, so keeping a second implementation host no longer reduced migration risk.
- `src/bigclaw/execution_contract.py`
  - Replace/expand. It now hosts both the execution-contract layer and the last surviving scheduler/runtime/reporting surface, making it the single legacy Python implementation module behind the package entrypoints.
- `src/bigclaw/__init__.py`
  - Retain. It still owns package-level re-exports and legacy module alias registration, and now aliases `bigclaw.scheduler` directly to `bigclaw.execution_contract` before aliases such as `runtime`, `reports`, and `operations`.
- `src/bigclaw/__main__.py`
  - Retain. `python -m bigclaw` still requires a physical `__main__.py`; this file now imports its report helpers from the consolidated host instead of the removed `scheduler.py`.

## Results

- `PYTHONPATH=src python3 - <<'PY'`
  - Result:
    - `bigclaw.scheduler -> bigclaw.execution_contract`
    - `bigclaw.runtime -> bigclaw.execution_contract`
    - `bigclaw.reports -> bigclaw.execution_contract`
    - `bigclaw.execution_contract -> bigclaw.execution_contract`
- `PYTHONPATH=src python3 -m pytest tests/test_scheduler.py tests/test_runtime.py tests/test_execution_flow.py tests/test_orchestration.py tests/test_operations.py tests/test_evaluation.py tests/test_risk.py tests/test_audit_events.py tests/test_execution_contract.py`
  - Result: `58 passed in 0.12s`
- `PYTHONPATH=src python3 -m pytest tests/test_github_sync.py tests/test_workspace_bootstrap.py`
  - Result: `14 passed in 4.11s`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/__main__.py src/bigclaw/execution_contract.py`
  - Result: passed with no output
- `find src/bigclaw -type f -name '*.py' | wc -l`
  - Result: `3`
- `find . -type f -name '*.py' | wc -l`
  - Result: `77`
- `git status --short`
  - Result:
    - `M .symphony/workpad.md`
    - `M src/bigclaw/__init__.py`
    - `M src/bigclaw/__main__.py`
    - `M src/bigclaw/execution_contract.py`
    - `D src/bigclaw/scheduler.py`
