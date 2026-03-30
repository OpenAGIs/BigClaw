# BIG-GO-1005 Batch Report

## Scope
- Batch target: residual Python modules under `src/bigclaw/**`
- Baseline batch count: `31`
- Final batch count: `29`
- Baseline repo-wide Python count: `94`
- Final repo-wide Python count: `92`
- Net reduction: `-2` batch files, `-2` repo-wide Python files

## Deleted Or Replaced
- `src/bigclaw/deprecation.py`
  Replaced by direct warning helpers in the remaining migration-only callers: `src/bigclaw/__main__.py`, `src/bigclaw/runtime.py`, and `scripts/dev_smoke.py`. No tests or scripts import the module anymore.
- `src/bigclaw/legacy_shim.py`
  Replaced by direct `bigclawctl` wrapper logic inside the operator compatibility scripts under `scripts/ops/`. No tests or scripts import the module anymore.

## Residual Inventory And Keep Basis
- `src/bigclaw/__init__.py`
  Keep. Package root still re-exports the migration compatibility surface used across the remaining Python tests and import paths.
- `src/bigclaw/__main__.py`
  Keep. Legacy `python -m bigclaw` entrypoint still provides the migration-only CLI wrapper for `serve` and `repo-sync-audit`.
- `src/bigclaw/audit_events.py`
  Keep. Directly covered by `tests/test_audit_events.py`.
- `src/bigclaw/collaboration.py`
  Keep. Directly covered by `tests/test_observability.py`, `tests/test_repo_collaboration.py`, and `tests/test_reports.py`.
- `src/bigclaw/connectors.py`
  Keep. Still exported from `src/bigclaw/__init__.py` and consumed by the package-level mapping surface.
- `src/bigclaw/console_ia.py`
  Keep. Directly covered by `tests/test_console_ia.py`.
- `src/bigclaw/dashboard_run_contract.py`
  Keep. Directly covered by `tests/test_dashboard_run_contract.py`.
- `src/bigclaw/design_system.py`
  Keep. Directly covered by `tests/test_console_ia.py` and `tests/test_design_system.py`.
- `src/bigclaw/dsl.py`
  Keep. Directly covered by `tests/test_dsl.py`.
- `src/bigclaw/evaluation.py`
  Keep. Directly covered by `tests/test_evaluation.py` and `tests/test_operations.py`.
- `src/bigclaw/event_bus.py`
  Keep. Directly covered by `tests/test_event_bus.py`.
- `src/bigclaw/execution_contract.py`
  Keep. Directly covered by `tests/test_execution_contract.py`.
- `src/bigclaw/github_sync.py`
  Keep. Directly covered by `tests/test_github_sync.py`.
- `src/bigclaw/governance.py`
  Keep. Directly covered by `tests/test_governance.py` and `tests/test_planning.py`.
- `src/bigclaw/issue_archive.py`
  Keep. Still exported from `src/bigclaw/__init__.py`; deleting it would break the package compatibility surface even though it no longer has dedicated tests.
- `src/bigclaw/memory.py`
  Keep. Directly covered by `tests/test_memory.py`.
- `src/bigclaw/models.py`
  Keep. Core domain module with broad direct coverage across the Python compatibility tests.
- `src/bigclaw/observability.py`
  Keep. Shared ledger and audit surface with broad direct coverage across the Python compatibility tests.
- `src/bigclaw/operations.py`
  Keep. Directly covered by `tests/test_control_center.py` and `tests/test_operations.py`.
- `src/bigclaw/planning.py`
  Keep. Directly covered by `tests/test_planning.py` and `tests/test_repo_rollout.py`.
- `src/bigclaw/repo_board.py`
  Keep. Directly covered by `tests/test_repo_board.py` and `tests/test_repo_collaboration.py`.
- `src/bigclaw/reports.py`
  Keep. Shared rendering surface with direct coverage in `tests/test_audit_events.py`, `tests/test_control_center.py`, `tests/test_observability.py`, `tests/test_operations.py`, `tests/test_repo_rollout.py`, and `tests/test_reports.py`.
- `src/bigclaw/risk.py`
  Keep. Directly covered by `tests/test_risk.py`.
- `src/bigclaw/run_detail.py`
  Keep. Still imported by `src/bigclaw/evaluation.py` and `src/bigclaw/reports.py` for run-detail rendering primitives.
- `src/bigclaw/runtime.py`
  Keep. Directly covered by `tests/test_runtime_matrix.py`.
- `src/bigclaw/saved_views.py`
  Keep. Directly covered by `tests/test_saved_views.py`.
- `src/bigclaw/ui_review.py`
  Keep. Directly covered by `tests/test_ui_review.py`.
- `src/bigclaw/workspace_bootstrap.py`
  Keep. Directly covered by `tests/test_workspace_bootstrap.py` and still imported by `src/bigclaw/workspace_bootstrap_validation.py`.
- `src/bigclaw/workspace_bootstrap_validation.py`
  Keep. Directly covered by `tests/test_workspace_bootstrap.py`.

## Notes
- This batch could not safely remove the remaining tested compatibility modules without simultaneously migrating or deleting their dependent tests and package exports, which is outside the narrow issue scope.
- The residual list is now concentrated in either actively tested Python compatibility surfaces or package-level compatibility exports that would become breaking changes if removed outright.
