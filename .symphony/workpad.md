# BIG-GO-964 Workpad

## Plan

1. Inventory the remaining physical Python modules under `src/bigclaw/` and confirm the last removable application-host candidate for this batch.
2. Fold `src/bigclaw/scheduler.py` into `src/bigclaw/execution_contract.py`, then fold `src/bigclaw/execution_contract.py` into `src/bigclaw/__init__.py` so the legacy Python package keeps a single implementation host plus the package entrypoint.
3. Update package exports and legacy module aliases in `src/bigclaw/__init__.py`, and redirect `src/bigclaw/__main__.py` to the consolidated host.
4. Delete `src/bigclaw/scheduler.py` and `src/bigclaw/execution_contract.py`, then run targeted import, pytest, and compile validation for the touched surfaces.
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
  - `bigclaw`
  - `bigclaw.execution_contract`
  - `bigclaw.scheduler`
  - `bigclaw.runtime`
  - `bigclaw.reports`
- `PYTHONPATH=src python3 -m pytest tests/test_scheduler.py tests/test_runtime.py tests/test_execution_flow.py tests/test_orchestration.py tests/test_operations.py tests/test_evaluation.py tests/test_risk.py tests/test_audit_events.py tests/test_execution_contract.py tests/test_github_sync.py tests/test_workspace_bootstrap.py`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/__main__.py`
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
  - Delete. Its remaining runtime, queue, orchestration, reporting, and compatibility helpers were first consolidated into `src/bigclaw/execution_contract.py`, then into `src/bigclaw/__init__.py`; keeping a separate module no longer reduced migration risk.
- `src/bigclaw/execution_contract.py`
  - Delete/replace. It temporarily hosted the merged execution-contract and scheduler surface, then was folded into `src/bigclaw/__init__.py`; `bigclaw.execution_contract` is now a compatibility alias to the package root.
- `src/bigclaw/__init__.py`
  - Retain/expand. It now owns the remaining legacy Python implementation, package-level re-exports, and legacy module alias registration; `bigclaw.execution_contract`, `bigclaw.scheduler`, `bigclaw.runtime`, and `bigclaw.reports` all resolve to the package root.
- `src/bigclaw/__main__.py`
  - Retain. `python -m bigclaw` still requires a physical `__main__.py`; this file now imports required symbols directly from the package root.

## Results

- `PYTHONPATH=src python3 - <<'PY'`
  - Result:
    - `bigclaw.execution_contract -> bigclaw`
    - `bigclaw.scheduler -> bigclaw`
    - `bigclaw.runtime -> bigclaw`
    - `bigclaw.reports -> bigclaw`
    - `symbols -> Scheduler RepoSyncAudit`
- `PYTHONPATH=src python3 -m pytest tests/test_scheduler.py tests/test_runtime.py tests/test_execution_flow.py tests/test_orchestration.py tests/test_operations.py tests/test_evaluation.py tests/test_risk.py tests/test_audit_events.py tests/test_execution_contract.py tests/test_github_sync.py tests/test_workspace_bootstrap.py`
  - Result: `72 passed in 4.44s`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/__main__.py`
  - Result: passed with no output
- `find src/bigclaw -type f -name '*.py' | wc -l`
  - Result: `2`
- `find . -type f -name '*.py' | wc -l`
  - Result: `76`
- `git status --short`
  - Result:
    - `M .symphony/workpad.md`
    - `M src/bigclaw/__init__.py`
    - `M src/bigclaw/__main__.py`
    - `D src/bigclaw/execution_contract.py`
