# BIG-GO-988

## Plan
- Inventory remaining `tests/**` Python files and confirm this overflow batch scope.
- Compare Python tests against existing Go test coverage and repository layout.
- Remove or replace redundant Python tests where a Go-only path already exists or the Python harness is obsolete.
- Keep changes scoped to the overflow batch and avoid unrelated migration work.
- Run targeted validation and record exact commands and outcomes.

## Acceptance
- Produce the explicit file list for the `tests/**` Python overflow batch.
- Reduce the number of Python files in the affected test directory where justified.
- Document rationale for each removed, replaced, or retained file.
- Report the impact on the repository-wide Python file count.

## Validation
- Recount repository Python files before and after changes.
- Run targeted tests for any touched or replacement coverage.
- Record exact validation commands and exact results in the final report.

## Batch Inventory
- Starting repository Python file count: `116`
- Starting `tests/**` Python file count: `41`
- Overflow batch files in scope:
  - `tests/conftest.py`
  - `tests/test_audit_events.py`
  - `tests/test_console_ia.py`
  - `tests/test_control_center.py`
  - `tests/test_dashboard_run_contract.py`
  - `tests/test_design_system.py`
  - `tests/test_dsl.py`
  - `tests/test_evaluation.py`
  - `tests/test_event_bus.py`
  - `tests/test_execution_contract.py`
  - `tests/test_execution_flow.py`
  - `tests/test_github_sync.py`
  - `tests/test_governance.py`
  - `tests/test_live_shadow_bundle.py`
  - `tests/test_memory.py`
  - `tests/test_models.py`
  - `tests/test_observability.py`
  - `tests/test_operations.py`
  - `tests/test_orchestration.py`
  - `tests/test_parallel_validation_bundle.py`
  - `tests/test_planning.py`
  - `tests/test_queue.py`
  - `tests/test_repo_board.py`
  - `tests/test_repo_collaboration.py`
  - `tests/test_repo_gateway.py`
  - `tests/test_repo_governance.py`
  - `tests/test_repo_links.py`
  - `tests/test_repo_registry.py`
  - `tests/test_repo_rollout.py`
  - `tests/test_repo_triage.py`
  - `tests/test_reports.py`
  - `tests/test_risk.py`
  - `tests/test_runtime.py`
  - `tests/test_runtime_matrix.py`
  - `tests/test_saved_views.py`
  - `tests/test_scheduler.py`
  - `tests/test_validation_bundle_continuation_policy_gate.py`
  - `tests/test_validation_policy.py`
  - `tests/test_workflow.py`
  - `tests/test_workspace_bootstrap.py`
  - `tests/test_ui_review.py`

## Decision Log
- Delete `tests/test_dashboard_run_contract.py`: replaced by Go-native dashboard contract coverage in `bigclaw-go/internal/product/dashboard_run_contract_test.go`.
- Delete `tests/test_saved_views.py`: replaced by Go-native saved-view catalog/audit/report coverage in `bigclaw-go/internal/product/saved_views_test.go`.
- Delete `tests/test_workspace_bootstrap.py`: replaced by Go-native bootstrap/cache/worktree coverage in `bigclaw-go/internal/bootstrap/bootstrap_test.go`.
- Delete `tests/test_validation_bundle_continuation_policy_gate.py`: replaced by adjacent script regression coverage in `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`.
- Retain the remaining `tests/**` Python files: they still exercise legacy Python production behavior or broader runtime/report surfaces without a tight Go-native replacement in this issue slice.
