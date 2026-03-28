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
  Migration status: `tests/test_runtime.py` migrated and removed in this issue; remaining Python runtime coverage lives in `tests/test_runtime_matrix.py` because legacy `src/bigclaw/runtime.py` and `src/bigclaw/scheduler.py` still exist outside the Go mainline.
- `tests/test_scheduler.py`
  Python coverage focuses on high-risk approval routing, browser routing, budget degradation/pause behavior.
  Go replacement surface is `bigclaw-go/internal/scheduler/scheduler.go` and `bigclaw-go/internal/scheduler/scheduler_test.go`.
  Migration status: migrated to Go and Python test asset removed in this issue; the one material semantic change is that Go rejects budget-blocked browser work instead of downgrading it to docker.
- `tests/test_orchestration.py`
  Python coverage focuses on cross-department planning, premium policy constraints, markdown rendering, and scheduler execution traces for orchestration policy/handoffs.
  Go replacement surface is `bigclaw-go/internal/workflow/orchestration.go`, `bigclaw-go/internal/workflow/orchestration_test.go`, `bigclaw-go/internal/scheduler/scheduler_test.go`, and `bigclaw-go/internal/worker/runtime_test.go`.
  Migration status: migrated to Go and Python test asset removed in this issue after replacement coverage landed in Go.

## First-pass Go Migration
- Added `RenderOrchestrationPlan` in `bigclaw-go/internal/workflow/orchestration.go` to replace the Python markdown rendering helper.
- Added `bigclaw-go/docs/reports/runtime-scheduler-orchestration-migration-report.md` as the repo-local migration inventory and Python-to-Go mapping report.
- Added `TestCrossDepartmentOrchestratorMatchesPythonMigrationCase` to preserve the original `tests/test_orchestration.py` planning scenario with Go inputs and outputs.
- Added `TestPremiumOrchestrationPolicyMatchesPythonMigrationCase` to lock the standard-tier blocked-department/cost semantics from `tests/test_orchestration.py`.
- Added `TestRenderOrchestrationPlanListsHandoffsAndPolicy` to lock the migrated markdown output from `tests/test_orchestration.py`.
- Added `TestSchedulerAssessmentCarriesStandardIncludedPolicyForBrowserOpsTask` to lock the orchestration-aware scheduler assessment semantics that Python previously asserted through scheduler execution.
- Added `TestSchedulerRejectsBudgetBlockedBrowserTaskInsteadOfDowngrading` to make the Go budget behavior explicit where Python used browser-to-docker downgrade semantics.
- Folded `tests/test_runtime.py` legacy-only sandbox and scheduler execution assertions into `tests/test_runtime_matrix.py`, then removed `tests/test_runtime.py`.
- Removed `tests/test_orchestration.py` after the Go replacement coverage and validation commands were in place.
- Removed `tests/test_scheduler.py` after the Go scheduler suite covered the migrated routing and budget guardrail scenarios, with the budget downgrade difference documented as intentional.
- Updated `tests/test_planning.py`, `src/bigclaw/planning.py`, and `reports/OPE-91-validation.md` to point migrated runtime/orchestration validation at Go packages instead of the removed Python test files.

## Python Asset Deletion Conditions
- `tests/test_orchestration.py` removed after the Go workflow/scheduler/runtime suites remained green for the migrated planning, policy, render, and handoff scenarios covered here.
- `tests/test_scheduler.py` removed after the Go scheduler suite covered the routing and budget guardrail behaviors, with the Python browser-to-docker downgrade replaced by explicit Go budget rejection semantics.
- `tests/test_runtime.py` removed after its issue-scoped assertions were migrated to Go or absorbed into `tests/test_runtime_matrix.py`; full retirement of Python runtime abstractions is still blocked by legacy `src/bigclaw/runtime.py` and `src/bigclaw/scheduler.py`.

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
- `go test ./internal/workflow`
  Result: `ok  	bigclaw-go/internal/workflow	0.424s`
- `go test ./internal/scheduler ./internal/worker`
  Result: `ok  	bigclaw-go/internal/scheduler	(cached)`; `ok  	bigclaw-go/internal/worker	(cached)`
- `go test ./internal/workflow ./internal/scheduler ./internal/worker`
  Result: `ok  	bigclaw-go/internal/workflow	(cached)`; `ok  	bigclaw-go/internal/scheduler	(cached)`; `ok  	bigclaw-go/internal/worker	(cached)`
- `PYTHONPATH=src python3 -m pytest tests/test_planning.py -q`
  Result: `.............. [100%]`
- `go test ./internal/worker ./internal/scheduler ./internal/workflow`
  Result: `ok  	bigclaw-go/internal/worker	(cached)`; `ok  	bigclaw-go/internal/scheduler	(cached)`; `ok  	bigclaw-go/internal/workflow	(cached)`
- `PYTHONPATH=src python3 -m pytest tests/test_runtime_matrix.py tests/test_planning.py -q`
  Result: `................... [100%]`
