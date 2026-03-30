# BIG-GO-1008 Validation

## Batch File List

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
- `tests/test_runtime_matrix.py`
- `tests/test_saved_views.py`
- `tests/test_scheduler.py`
- `tests/test_ui_review.py`
- `tests/test_validation_bundle_continuation_policy_gate.py`
- `tests/test_validation_policy.py`
- `tests/test_workspace_bootstrap.py`

## Deleted Python Tests

- `tests/test_saved_views.py`
  - Deleted.
  - Replaced by `bigclaw-go/internal/product/saved_views.go`, `bigclaw-go/internal/product/saved_views_test.go`, and `bigclaw-go/internal/api/expansion_test.go`.
  - Basis: Go-native catalog building, audit, report rendering, and export/API coverage already exist.
- `tests/test_dashboard_run_contract.py`
  - Deleted.
  - Replaced by `bigclaw-go/internal/product/dashboard_run_contract.go`, `bigclaw-go/internal/product/dashboard_run_contract_test.go`, and `bigclaw-go/internal/api/expansion_test.go`.
  - Basis: default contract, audit gap detection, report rendering, and export/API coverage are already asserted in Go.
- `tests/test_repo_gateway.py`
  - Deleted.
  - Replaced by `bigclaw-go/internal/repo/gateway.go` and `bigclaw-go/internal/repo/repo_surfaces_test.go`.
  - Basis: Go tests already cover payload normalization, gateway error normalization, and deterministic audit payload output.
- `tests/test_repo_registry.py`
  - Deleted.
  - Replaced by `bigclaw-go/internal/repo/registry.go` and `bigclaw-go/internal/repo/repo_surfaces_test.go`.
  - Basis: Go tests already cover deterministic space resolution, default channel generation, and agent resolution.
- `tests/test_repo_board.py`
  - Deleted.
  - Replaced by `bigclaw-go/internal/repo/board.go` and `bigclaw-go/internal/repo/repo_surfaces_test.go`.
  - Basis: Go tests already cover create/reply/filter behavior and post metadata handling.
- `tests/test_repo_governance.py`
  - Deleted.
  - Replaced by `bigclaw-go/internal/repo/governance.go` and `bigclaw-go/internal/repo/governance_test.go`.
  - Basis: Go tests already cover role permission checks and deterministic required-audit-field calculation.
- `tests/test_repo_triage.py`
  - Deleted.
  - Replaced by `bigclaw-go/internal/triage/repo.go`, `bigclaw-go/internal/triage/repo_test.go`, and `bigclaw-go/internal/api/server_test.go`.
  - Basis: Go tests already cover lineage-aware recommendations and approval evidence packet generation.
- `tests/test_repo_links.py`
  - Deleted.
  - Replaced by `bigclaw-go/internal/repo/commits.go`, `bigclaw-go/internal/repo/repo_surfaces_test.go`, and `bigclaw-go/internal/api/server_test.go`.
  - Basis: Go tests already cover run-commit binding, accepted hash resolution, and closeout payload exposure.
- `tests/test_github_sync.py`
  - Deleted.
  - Replaced by `bigclaw-go/internal/githubsync/sync.go` and `bigclaw-go/internal/githubsync/sync_test.go`.
  - Basis: Go tests already cover hook installation, push/sync behavior, dirty worktrees, and fast-forward/default-branch cases.
- `tests/test_risk.py`
  - Deleted.
  - Replaced by `bigclaw-go/internal/risk/risk.go`, `bigclaw-go/internal/risk/risk_test.go`, and `bigclaw-go/internal/scheduler/scheduler_test.go`.
  - Basis: Go tests already cover low/medium/high scoring, approval requirements, and risk-aware scheduling outcomes.
- `tests/test_queue.py`
  - Deleted.
  - Replaced by `bigclaw-go/internal/queue/file_queue.go`, `bigclaw-go/internal/queue/file_queue_test.go`, `bigclaw-go/internal/queue/memory_queue_test.go`, and `bigclaw-go/internal/queue/sqlite_queue_test.go`.
  - Basis: Go tests already cover persistence, priority order, dead-letter replay, and reload behavior across queue backends.

## Kept Python Tests

- `tests/conftest.py`
- `tests/test_audit_events.py`
- `tests/test_console_ia.py`
- `tests/test_control_center.py`
- `tests/test_design_system.py`
- `tests/test_dsl.py`
- `tests/test_evaluation.py`
- `tests/test_event_bus.py`
- `tests/test_execution_contract.py`
- `tests/test_governance.py`
- `tests/test_live_shadow_bundle.py`
- `tests/test_memory.py`
- `tests/test_models.py`
- `tests/test_observability.py`
- `tests/test_operations.py`
- `tests/test_orchestration.py`
- `tests/test_parallel_validation_bundle.py`
- `tests/test_planning.py`
- `tests/test_repo_collaboration.py`
- `tests/test_repo_rollout.py`
- `tests/test_reports.py`
- `tests/test_runtime_matrix.py`
- `tests/test_scheduler.py`
- `tests/test_ui_review.py`
- `tests/test_validation_bundle_continuation_policy_gate.py`
- `tests/test_validation_policy.py`
- `tests/test_workspace_bootstrap.py`

## Keep Basis

- These files still exercise Python-owned behavior, Python-only report/rendering surfaces, or script orchestration that does not yet have a tight checked-in Go-native contract in this repo.
- `tests/test_scheduler.py` was kept even though Go scheduler tests exist because the surviving Python assertions are not a direct semantic match to the current Go routing behavior.
- `tests/test_event_bus.py`, `tests/test_operations.py`, `tests/test_ui_review.py`, `tests/test_parallel_validation_bundle.py`, `tests/test_runtime_matrix.py`, and related report/script tests were kept because they still validate Python-specific ledger, report, or script surfaces rather than already-migrated Go packages.

## Python File Count Impact

- Repository Python files before: `108`
- Repository Python files after: `97`
- Targeted `tests/**` Python files before: `38`
- Targeted `tests/**` Python files after: `27`
- Net reduction: `11`

## Validation Commands

- `rg --files tests -g '*.py' | sort`
- `rg --files . -g '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/product ./internal/repo ./internal/triage ./internal/githubsync ./internal/risk ./internal/queue`
- `git status --short`

## Validation Results

- `rg --files tests -g '*.py' | sort`
  - Result: `27` remaining Python files under `tests/`.
- `rg --files . -g '*.py' | wc -l`
  - Result: `97`
- `cd bigclaw-go && go test ./internal/product ./internal/repo ./internal/triage ./internal/githubsync ./internal/risk ./internal/queue`
  - Result:
    - `ok  	bigclaw-go/internal/product	1.550s`
    - `ok  	bigclaw-go/internal/repo	1.986s`
    - `ok  	bigclaw-go/internal/triage	1.128s`
    - `ok  	bigclaw-go/internal/githubsync	5.336s`
    - `ok  	bigclaw-go/internal/risk	2.795s`
    - `ok  	bigclaw-go/internal/queue	28.490s`
- `git status --short`
  - Result: `.symphony/workpad.md`, this validation report, and the scoped Python test deletions only.
