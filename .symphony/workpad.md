# BIG-GO-1008

## Plan
1. Inventory every remaining `tests/**` Python file and group them by replacement status: Go-covered, still Python-owned, or blocked pending migration.
2. Inspect the current overflow batch and delete only the Python tests whose assertions are already covered by Go-native tests or repo-native regression fixtures.
3. Record delete/replace/keep rationale for the full batch in an issue report so the remaining tail is explicit instead of implicit.
4. Run targeted validation for the Go packages that replace deleted tests, then re-count remaining `tests/**` and repo-wide Python files.
5. Commit the scoped issue changes and push the branch to the remote.

## Acceptance
- Produce the explicit `tests/**` Python file list for this overflow batch.
- Reduce the number of Python files where safe in this issue scope.
- Capture delete/replace/keep rationale for each file in the batch.
- Report the delta for remaining `tests/**` Python files and total repo Python files.

## Validation
- `rg --files tests -g '*.py'`
- `rg --files . -g '*.py' | wc -l`
- `cd bigclaw-go && go test ...` for the Go packages replacing deleted Python coverage
- `git status --short`

## Results

### Deleted

- `tests/test_saved_views.py`
- `tests/test_dashboard_run_contract.py`
- `tests/test_repo_gateway.py`
- `tests/test_repo_registry.py`
- `tests/test_repo_board.py`
- `tests/test_repo_governance.py`
- `tests/test_repo_triage.py`
- `tests/test_repo_links.py`
- `tests/test_github_sync.py`
- `tests/test_risk.py`
- `tests/test_queue.py`

### Kept

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

### Count Impact

- Repository Python files before: `108`
- Repository Python files after: `97`
- `tests/**` Python files before: `38`
- `tests/**` Python files after: `27`
- Net reduction: `11`

### Validation Record

- `rg --files tests -g '*.py' | sort`
  - Result: `27` remaining files
- `rg --files . -g '*.py' | wc -l`
  - Result: `97`
- `cd bigclaw-go && go test ./internal/product ./internal/repo ./internal/triage ./internal/githubsync ./internal/risk ./internal/queue`
  - Result: all packages passed
- `git status --short`
  - Result: only `.symphony/workpad.md`, `reports/BIG-GO-1008-validation.md`, and the scoped test deletions are modified
