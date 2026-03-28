# Runtime / Scheduler / Orchestration Migration Report

## Scope

This report tracks the Go-only migration status for the Python test assets:

- `tests/test_runtime.py`
- `tests/test_scheduler.py`
- `tests/test_orchestration.py`

Issue: `BIG-GO-925`

## Asset Inventory

### Runtime

- Python asset: `tests/test_runtime.py`
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

- Python asset: `tests/test_scheduler.py`
- Legacy Python surface:
  - medium-based routing (`browser`, `docker`, `vm`, `none`)
  - budget downgrade and pause semantics
- Go replacement surface:
  - `internal/scheduler/scheduler.go`
  - `internal/scheduler/scheduler_test.go`
  - `internal/worker/runtime_test.go`

### Orchestration

- Python asset: `tests/test_orchestration.py`
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
| `test_cross_department_orchestrator_routes_security_data_and_customer_work` | `TestCrossDepartmentOrchestratorMatchesPythonMigrationCase` and `TestCrossDepartmentOrchestratorPlansHandoffs` | migrated |
| `test_standard_policy_limits_advanced_cross_department_routing` | `TestPremiumOrchestrationPolicyMatchesPythonMigrationCase` and `TestPremiumOrchestrationPolicyConstrainsStandardTier` | migrated |
| `test_render_orchestration_plan_lists_handoffs_and_policy` | `RenderOrchestrationPlan` and `TestRenderOrchestrationPlanListsHandoffsAndPolicy` | migrated |
| `test_scheduler_execution_records_orchestration_plan_and_policy` | `TestSchedulerAssessmentCarriesStandardIncludedPolicyForBrowserOpsTask` plus runtime event coverage in `TestRuntimePublishesOrchestrationAssessmentOnRoutedEvent` | migrated |
| `test_scheduler_creates_handoff_for_policy_or_approval_blockers` | `TestSchedulerAssessmentBuildsUpgradeHandoffForStandardTier`, `TestSchedulerAssessmentBuildsSecurityHandoffForRejectedDecision`, `TestRuntimePublishesRejectedDecisionHandoffBeforeRetry` | migrated |
| `test_scheduler_high_risk_requires_approval` | `TestSchedulerRoutesHighRiskToKubernetes` with Go risk/executor semantics | migrated with semantic adaptation |
| `test_scheduler_browser_task_routes_browser` | `TestSchedulerRoutesBrowserToKubernetes` with Go executor routing semantics | migrated with semantic adaptation |
| `test_scheduler_over_budget_degrades_browser_task_to_docker` | no 1:1 replacement; Go rejects on budget rather than degrading executor | intentional semantic divergence |
| `test_scheduler_over_budget_pauses_task` | `TestSchedulerBudgetGuardrail` and runtime retry/handoff coverage | migrated with semantic adaptation |
| `test_sandbox_router_maps_execution_media` | no 1:1 replacement; Go uses executor assignment instead of Python sandbox media objects | superseded |
| `test_tool_runtime_blocks_disallowed_tool_and_audits` | no 1:1 replacement; tool-policy abstraction is superseded by executor/runtime event model | superseded |
| `test_worker_runtime_returns_tool_results_for_approved_task` | runtime execution lifecycle coverage in `internal/worker/runtime_test.go` | migrated with semantic adaptation |
| `test_scheduler_records_worker_runtime_results_and_waits_on_high_risk` | `TestRuntimePublishesRejectedDecisionHandoffBeforeRetry` and scheduler assessment handoff coverage | migrated with semantic adaptation |
| `test_scheduler_pauses_execution_when_budget_cannot_cover_docker` | `TestSchedulerBudgetGuardrail` plus runtime rejection path coverage | migrated with semantic adaptation |

## Intentional Semantic Differences

- Go scheduler routes to executors (`local`, `kubernetes`, `ray`) instead of Python mediums (`docker`, `browser`, `vm`, `none`).
- Go budget handling rejects work when `budget_cents` exceeds remaining budget; it does not downgrade browser work to docker.
- Go runtime policy signaling is emitted through routed/handoff/task lifecycle events and workpad journals rather than Python `ToolRuntime` audit objects.

## Deletion Conditions

- Delete `tests/test_orchestration.py` after the Go workflow, scheduler, and worker suites remain green for the migrated plan/policy/render/handoff cases.
- Delete `tests/test_scheduler.py` after the team accepts the Go budget semantics as authoritative or adds a Go downgrade policy intentionally.
- Delete `tests/test_runtime.py` after Python-only sandbox/tool-policy abstractions are fully retired from the repository surface.

## Validation Commands

- `go test ./internal/workflow`
- `go test ./internal/scheduler`
- `go test ./internal/worker`
- `go test ./internal/worker ./internal/scheduler ./internal/workflow`
