# Scheduler Policy Report

## Scope

This report summarizes the current Go scheduler policy surface for `OPE-179` / `BIG-GO-004`.

## Implemented policies

- Budget guardrail rejects tasks whose `budget_cents` exceed remaining budget.
- Concurrency quota rejects new work when tenant concurrency is exhausted.
- Multi-tenant fairness windows can throttle dominant tenant low-priority work when peer tenants are also active, with optional shared SQLite-backed coordination across scheduler processes.
- Preemptible capacity allows urgent tasks to pass the concurrency guardrail when `preemptible_executions` is available.
- Backpressure rejects low-priority tasks when queue depth exceeds `max_queue_depth`.
- High-risk tasks default to `kubernetes`.
- GPU-tagged tasks default to `ray`.
- Browser-tagged tasks default to `kubernetes`.
- Explicit `required_executor` still overrides policy routing.

## Evidence

- Policy implementation: `internal/scheduler/scheduler.go` and `internal/scheduler/policy_store.go`
- Unit coverage: `internal/scheduler/scheduler_test.go`
- Runtime emission of `scheduler.routed`: `internal/worker/runtime.go`
- File-backed scheduler policy inspection and reload: `GET /v2/control-center/policy` and `POST /v2/control-center/policy/reload`
- Local benchmark: `docs/reports/benchmark-report.md`

## Fresh benchmark snapshot

- `BenchmarkSchedulerDecide-8 = 51.08 ns/op`

## Remaining gaps

- Shared fairness coordination is now available through an optional SQLite-backed store, but there is still no remote/distributed fairness service beyond shared local SQLite state.
- No active task eviction or live preemption mechanism yet; `preemptible_executions` is a scheduling allowance, not forced runtime cancellation.
- File-backed policy reload is now available, but there is still no distributed/shared external policy backend beyond per-service file reload.
