# BIG-GO-925

## Plan
- Inventory current Python tests in `tests/test_runtime.py`, `tests/test_scheduler.py`, and `tests/test_orchestration.py`.
- Map each behavior to existing Go packages under `bigclaw-go/internal/worker`, `bigclaw-go/internal/scheduler`, and `bigclaw-go/internal/workflow`.
- Implement first-pass Go replacements for uncovered behavior while keeping changes scoped to runtime/scheduler/orchestration migration.
- Run targeted Go tests, record exact commands and results, then commit and push the branch.

## Acceptance
- Produce a concrete inventory of Python/non-Go assets for runtime/scheduler/orchestration tests.
- Add Go replacement implementation and/or migration coverage for the first batch of behaviors.
- State deletion conditions for legacy Python assets.
- Capture regression validation commands and results.

## Inventory
- `tests/test_runtime.py`
  Python coverage centers on legacy `SandboxRouter`, `ToolRuntime`, `ClawWorkerRuntime`, and `Scheduler.execute` integration behavior.
  Go replacement surface is `bigclaw-go/internal/worker/runtime.go`, `bigclaw-go/internal/worker/runtime_runonce.go`, and `bigclaw-go/internal/worker/runtime_test.go`.
  Migration status: existing Go runtime tests already cover routed decisions, rejected-decision handoff/retry, workpad journaling, executor artifacts, and lifecycle events; the Python-only sandbox/tool-policy abstractions do not have a 1:1 Go type and are superseded by executor assignment plus routed/handoff payloads.
- `tests/test_scheduler.py`
  Python coverage focuses on high-risk approval routing, browser routing, budget degradation/pause behavior.
  Go replacement surface is `bigclaw-go/internal/scheduler/scheduler.go` and `bigclaw-go/internal/scheduler/scheduler_test.go`.
  Migration status: Go already covers high-risk routing, browser routing, budget guardrails, backpressure, preemption, policy stores, fairness, and now includes an explicit standard-tier included-policy assessment case mirroring orchestration-aware execution.
- `tests/test_orchestration.py`
  Python coverage focuses on cross-department planning, premium policy constraints, markdown rendering, and scheduler execution traces for orchestration policy/handoffs.
  Go replacement surface is `bigclaw-go/internal/workflow/orchestration.go`, `bigclaw-go/internal/workflow/orchestration_test.go`, `bigclaw-go/internal/scheduler/scheduler_test.go`, and `bigclaw-go/internal/worker/runtime_test.go`.
  Migration status: this change adds direct Go coverage for the Python premium-blocking case and markdown rendering output; existing Go runtime/scheduler tests already cover orchestration payloads and handoff events.

## First-pass Go Migration
- Added `RenderOrchestrationPlan` in `bigclaw-go/internal/workflow/orchestration.go` to replace the Python markdown rendering helper.
- Added `TestPremiumOrchestrationPolicyMatchesPythonMigrationCase` to lock the standard-tier blocked-department/cost semantics from `tests/test_orchestration.py`.
- Added `TestRenderOrchestrationPlanListsHandoffsAndPolicy` to lock the migrated markdown output from `tests/test_orchestration.py`.
- Added `TestSchedulerAssessmentCarriesStandardIncludedPolicyForBrowserOpsTask` to lock the orchestration-aware scheduler assessment semantics that Python previously asserted through scheduler execution.

## Python Asset Deletion Conditions
- Delete `tests/test_orchestration.py` after the Go workflow/scheduler/runtime suites remain green for the migrated planning, policy, render, and handoff scenarios covered here.
- Delete `tests/test_scheduler.py` after the remaining Python-only budget degradation semantics are either intentionally retired or re-expressed against Go scheduler rules; current Go scheduler uses executor routing and budget rejection rather than the Python medium downgrade model.
- Delete `tests/test_runtime.py` only after the repository no longer relies on Python `SandboxRouter`/`ToolRuntime` runtime abstractions, or after any still-required policy semantics are restated against Go executor/runtime events.

## Validation
- `go test ./internal/worker ./internal/scheduler ./internal/workflow`
- Additional focused `go test` commands for newly added test cases as needed.

## Validation Results
- `go test ./internal/workflow`
  Result: `ok  	bigclaw-go/internal/workflow	3.166s`
- `go test ./internal/scheduler`
  Result: `ok  	bigclaw-go/internal/scheduler	1.275s`
- `go test ./internal/worker`
  Result: `ok  	bigclaw-go/internal/worker	2.075s`
- `go test ./internal/worker ./internal/scheduler ./internal/workflow`
  Result: `ok  	bigclaw-go/internal/worker	(cached)`; `ok  	bigclaw-go/internal/scheduler	(cached)`; `ok  	bigclaw-go/internal/workflow	(cached)`
