# Runtime / Scheduler / Orchestration Migration Report

## Scope

This report tracks the Go-only migration status for the Python test assets:

- `tests/test_runtime.py` (removed in this issue after Go migration)
- `tests/test_scheduler.py` (removed in this issue after Go migration)
- `tests/test_orchestration.py` (removed in this issue after Go migration)

Issue: `BIG-GO-925`

## Asset Inventory

### Runtime

- Python asset: `tests/test_runtime.py` (removed)
- Legacy Python surface:
  - `SandboxRouter`
  - `ToolRuntime`
  - `ToolPolicy`
  - `ClawWorkerRuntime`
  - `Scheduler.execute` integration via Python observability ledger
- Go replacement surface:
  - `internal/worker/runtime.go`
  - `internal/worker/runtime_runonce.go`
  - `internal/worker/runtime_test.go`
  - `internal/scheduler/scheduler.go`

### Scheduler

- Python asset: `tests/test_scheduler.py` (removed)
- Legacy Python surface:
  - medium-based routing (`browser`, `docker`, `vm`, `none`)
  - budget downgrade and pause semantics
- Go replacement surface:
  - `internal/scheduler/scheduler.go`
  - `internal/scheduler/scheduler_test.go`
  - `internal/worker/runtime_test.go`

### Orchestration

- Python asset: `tests/test_orchestration.py` (removed)
- Legacy Python surface:
  - cross-department planning
  - premium-policy gating
  - markdown rendering helper
  - scheduler execution traces and handoff creation
- Go replacement surface:
  - `internal/workflow/orchestration.go`
  - `internal/workflow/orchestration_test.go`
  - `internal/scheduler/scheduler_test.go`
  - `internal/worker/runtime_test.go`

## Python to Go Mapping

| Python test | Go replacement | Status |
| --- | --- | --- |
| `test_cross_department_orchestrator_routes_security_data_and_customer_work` | `TestCrossDepartmentOrchestratorMatchesPythonMigrationCase` and `TestCrossDepartmentOrchestratorPlansHandoffs` | migrated, Python asset removed |
| `test_standard_policy_limits_advanced_cross_department_routing` | `TestPremiumOrchestrationPolicyMatchesPythonMigrationCase` and `TestPremiumOrchestrationPolicyConstrainsStandardTier` | migrated, Python asset removed |
| `test_render_orchestration_plan_lists_handoffs_and_policy` | `RenderOrchestrationPlan` and `TestRenderOrchestrationPlanListsHandoffsAndPolicy` | migrated, Python asset removed |
| `test_scheduler_execution_records_orchestration_plan_and_policy` | `TestSchedulerAssessmentCarriesStandardIncludedPolicyForBrowserOpsTask` plus runtime event coverage in `TestRuntimePublishesOrchestrationAssessmentOnRoutedEvent` | migrated, Python asset removed |
| `test_scheduler_creates_handoff_for_policy_or_approval_blockers` | `TestSchedulerAssessmentBuildsUpgradeHandoffForStandardTier`, `TestSchedulerAssessmentBuildsSecurityHandoffForRejectedDecision`, `TestRuntimePublishesRejectedDecisionHandoffBeforeRetry` | migrated, Python asset removed |
| `test_scheduler_high_risk_requires_approval` | `TestSchedulerRoutesHighRiskToKubernetes` with Go risk/executor semantics | migrated, Python asset removed |
| `test_scheduler_browser_task_routes_browser` | `TestSchedulerRoutesBrowserToKubernetes` with Go executor routing semantics | migrated, Python asset removed |
| `test_scheduler_over_budget_degrades_browser_task_to_docker` | `TestSchedulerRejectsBudgetBlockedBrowserTaskInsteadOfDowngrading` | migrated with intentional semantic divergence, Python asset removed |
| `test_scheduler_over_budget_pauses_task` | `TestSchedulerBudgetGuardrail` and runtime retry/handoff coverage | migrated, Python asset removed |
| `test_sandbox_router_maps_execution_media` | `tests/test_runtime_matrix.py::test_runtime_sandbox_router_profiles_are_stable` plus Go executor-routing replacement in `internal/worker/runtime_test.go` | migrated, Python asset removed |
| `test_tool_runtime_blocks_disallowed_tool_and_audits` | `tests/test_runtime_matrix.py::test_big303_tool_runtime_policy_and_audit_chain` plus Go runtime event model | migrated, Python asset removed |
| `test_worker_runtime_returns_tool_results_for_approved_task` | `tests/test_runtime_matrix.py::test_big301_worker_lifecycle_is_stable_with_multiple_tools` and runtime execution lifecycle coverage in `internal/worker/runtime_test.go` | migrated, Python asset removed |
| `test_scheduler_records_worker_runtime_results_and_waits_on_high_risk` | `tests/test_runtime_matrix.py::test_big302_scheduler_execution_records_pending_and_budget_paused_states` and `TestRuntimePublishesRejectedDecisionHandoffBeforeRetry` | migrated, Python asset removed |
| `test_scheduler_pauses_execution_when_budget_cannot_cover_docker` | `tests/test_runtime_matrix.py::test_big302_scheduler_execution_records_pending_and_budget_paused_states` plus `TestSchedulerBudgetGuardrail` | migrated, Python asset removed |

## Intentional Semantic Differences

- Go scheduler routes to executors (`local`, `kubernetes`, `ray`) instead of Python mediums (`docker`, `browser`, `vm`, `none`).
- Go budget handling rejects work when `budget_cents` exceeds remaining budget; it does not downgrade browser work to docker.
- Go runtime policy signaling is emitted through routed/handoff/task lifecycle events and workpad journals rather than Python `ToolRuntime` audit objects.

## Deletion Conditions

- `tests/test_orchestration.py` has been removed because the Go workflow, scheduler, and worker suites cover the migrated plan, policy, render, and handoff cases.
- `tests/test_scheduler.py` has been removed because the Go scheduler suite now covers the routing and budget guardrail cases, and the browser downgrade difference is documented as an intentional Go behavior.
- `tests/test_runtime.py` has been removed for this issue. Full retirement of Python runtime abstractions is still blocked by legacy `src/bigclaw/runtime.py`, `src/bigclaw/scheduler.py`, and `tests/test_runtime_matrix.py`.

## Validation Commands

- `go test ./internal/workflow`
- `go test ./internal/scheduler`
- `go test ./internal/worker`
- `go test ./internal/worker ./internal/scheduler ./internal/workflow`
