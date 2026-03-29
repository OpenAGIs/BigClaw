# BIG-GO-948 Workpad

## Scope

Second-wave cleanup for the remaining Python tests under `tests/` that already have stable Go-native coverage in `bigclaw-go`.

Planned delete set:
- `tests/test_connectors.py`
- `tests/test_mapping.py`
- `tests/test_repo_governance.py`
- `tests/test_repo_board.py`
- `tests/test_repo_links.py`
- `tests/test_repo_triage.py`
- `tests/test_repo_collaboration.py`
- `tests/test_repo_registry.py`
- `tests/test_repo_rollout.py`
- `tests/test_repo_gateway.py`
- `tests/test_scheduler.py`
- `tests/test_dashboard_run_contract.py`
- `tests/test_risk.py`
- `tests/test_saved_views.py`
- `tests/test_workflow.py`
- `tests/test_orchestration.py`
- `tests/test_audit_events.py`
- `tests/test_observability.py`
- `tests/test_queue.py`

Go coverage used for replacement:
- `bigclaw-go/internal/intake/mapping_test.go`
- `bigclaw-go/internal/repo/governance_test.go`
- `bigclaw-go/internal/repo/repo_surfaces_test.go`
- `bigclaw-go/internal/repo/triage.go`
- `bigclaw-go/internal/triage/*.go`
- `bigclaw-go/internal/repo/registry.go`
- `bigclaw-go/internal/repo/board.go`
- `bigclaw-go/internal/repo/links.go`
- `bigclaw-go/internal/repo/gateway.go`
- `bigclaw-go/internal/product/dashboard_run_contract_test.go`
- `bigclaw-go/internal/product/saved_views_test.go`
- `bigclaw-go/internal/risk/*.go`
- `bigclaw-go/internal/scheduler/scheduler_test.go`
- `bigclaw-go/internal/workflow/*.go`
- `bigclaw-go/internal/observability/*.go`
- `bigclaw-go/internal/queue/*.go`

## Acceptance

- Record the lane file list for this wave in `reports/BIG-GO-948-validation.md`.
- Delete only Python tests whose behavior is already covered by Go-native tests.
- Document the remaining Python tests as explicit follow-up delete/migration plan, not implicit backlog.
- Run targeted Go validation for the deleted surfaces.
- Commit and push the lane branch.

## Validation

- `cd bigclaw-go && go test ./internal/intake -run 'TestConnector|TestMap'`
- `cd bigclaw-go && go test ./internal/repo -run 'TestRepo|TestGovernance'`
- `cd bigclaw-go && go test ./internal/triage`
- `cd bigclaw-go && go test ./internal/product -run 'TestDashboardRunContract|TestSavedView|TestConsole'`
- `cd bigclaw-go && go test ./internal/risk`
- `cd bigclaw-go && go test ./internal/scheduler`
- `cd bigclaw-go && go test ./internal/workflow`
- `cd bigclaw-go && go test ./internal/observability`
- `cd bigclaw-go && go test ./internal/queue`
- `git status --short`

## Risks

- `tests/test_connectors.py` still hits Python connector fetch stubs; deletion is acceptable only if Go intake coverage stays local and deterministic.
- Repo-surface Python tests overlap several Go packages; validation needs to prove the delete set still has coverage across governance, board, links, gateway, registry, and triage behavior.
- Larger Python-only report generators such as `tests/test_reports.py`, `tests/test_ui_review.py`, and `tests/test_operations.py` remain intentionally out of scope for this lane because they do not yet have a tight Go-native replacement boundary.
