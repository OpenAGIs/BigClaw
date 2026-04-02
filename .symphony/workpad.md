# BIG-GO-985 Workpad

## Scope
- Final sweep E for remaining `src/bigclaw/**` Python modules, batch 3.
- Keep changes scoped to package-surface consolidation only.

## Batch Inventory
- Candidate Python modules in this batch:
  - `src/bigclaw/audit_events.py`
  - `src/bigclaw/collaboration.py`
  - `src/bigclaw/dashboard_run_contract.py`
  - `src/bigclaw/design_system.py`
  - `src/bigclaw/run_detail.py`
  - `src/bigclaw/ui_review.py`
- Expected retained package entry surfaces:
  - `src/bigclaw/__init__.py`
  - `src/bigclaw/__main__.py`
  - `src/bigclaw/runtime.py`
  - other still-live domain hosts needed outside this batch

## Plan
1. Inspect import/test usage for the batch modules and confirm they can collapse into surviving host modules without changing public imports.
2. Move owned symbols into surviving host modules and install compatibility submodules from `bigclaw.__init__` or another retained host.
3. Delete redundant batch files and update package exports.
4. Run targeted validation, record exact commands/results, measure Python file-count impact, then commit and push.

## Acceptance
- Batch file list is explicitly recorded.
- Python file count under `src/bigclaw` goes down for this batch if compatibility collapse is safe.
- Each deleted/replaced/retained file has a stated basis.
- Repo-wide Python file count impact is measured and recorded.
- Changes are committed and pushed to the remote branch.

## Validation Plan
- Use `rg` to confirm no live code paths still require deleted files on disk.
- Run targeted `pytest` for affected module coverage.
- Run `python3 -m compileall src/bigclaw`.
- Run `git diff --check`.
- Record exact file-count commands before/after.

## Validation Results
- Inventory basis:
  - Batch deleted: `src/bigclaw/audit_events.py`, `src/bigclaw/collaboration.py`, `src/bigclaw/dashboard_run_contract.py`, `src/bigclaw/run_detail.py`
  - Replaced by retained hosts:
    - `audit_events.py` -> `src/bigclaw/observability.py`
    - `collaboration.py` -> `src/bigclaw/observability.py`
    - `dashboard_run_contract.py` -> `src/bigclaw/reports.py`
    - `run_detail.py` -> `src/bigclaw/reports.py`
  - Retained because still live outside this batch: `src/bigclaw/__init__.py`, `src/bigclaw/__main__.py`, `src/bigclaw/runtime.py`, plus remaining domain hosts such as `design_system.py` and `ui_review.py`
- Compatibility basis:
  - `src/bigclaw/__init__.py` now installs synthetic compatibility submodules for `bigclaw.audit_events`, `bigclaw.collaboration`, `bigclaw.dashboard_run_contract`, and `bigclaw.run_detail`
  - Deleted import paths continue to resolve without keeping separate `.py` files on disk
- Count impact:
  - Repo-wide Python files: `112 -> 108` via `find . -name '*.py' | wc -l`
  - `src/bigclaw` top-level Python files: `42 -> 38` via `find src/bigclaw -maxdepth 1 -name '*.py' | wc -l`
- Exact commands and results:
  - `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/observability.py src/bigclaw/reports.py src/bigclaw/runtime.py src/bigclaw/repo_board.py src/bigclaw/evaluation.py` -> success
  - `python3 - <<'PY' ... import bigclaw.audit_events ... import bigclaw.run_detail ... PY` -> `compat_imports_ok AuditEventSpec CollaborationThread DashboardRunContract RunDetailEvent`
  - `PYTHONPATH=src python3 -m pytest tests/test_audit_events.py tests/test_repo_collaboration.py tests/test_dashboard_run_contract.py tests/test_reports.py tests/test_observability.py -q` -> `50 passed in 0.17s`
  - `PYTHONPATH=src python3 -m pytest tests/test_evaluation.py -q` -> `7 passed in 0.08s`
  - `PYTHONPATH=src python3 -m pytest tests/test_console_ia.py tests/test_design_system.py -q` -> `26 passed in 0.10s`
  - `python3 -m compileall src/bigclaw` -> success
  - `git diff --check` -> clean
