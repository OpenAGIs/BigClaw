# Scheduler Policy Report

## Scope

This report summarizes the current Go scheduler policy surface for `OPE-179` / `BIG-GO-004`.

## Implemented policies

- Budget guardrail rejects tasks whose `budget_cents` exceed remaining budget.
- Concurrency quota rejects new work when tenant concurrency is exhausted.
- Multi-tenant fairness windows can throttle dominant tenant low-priority work when peer tenants are also active, with optional shared SQLite-backed coordination or a remote HTTP fairness service across scheduler processes.
- Preemptible capacity now supports live preemption: urgent tasks can cancel a lower-priority leased/running task to reclaim capacity when `preemptible_executions` is available.
- Backpressure rejects low-priority tasks when queue depth exceeds `max_queue_depth`.
- Tenant isolation can now resolve capacity boundaries from `task.tenant_id` plus configured metadata keys such as `app_id`, and owner matching can enforce per-user bot ownership using configured owner metadata keys.
- High-risk tasks default to `kubernetes`.
- GPU-tagged tasks default to `ray`.
- Browser-tagged tasks default to `kubernetes`.
- Explicit `required_executor` still overrides policy routing.

## Evidence

- Policy implementation: `internal/scheduler/scheduler.go` and `internal/scheduler/policy_store.go`
- Unit coverage: `internal/scheduler/scheduler_test.go` and `internal/worker/runtime_test.go`
- Runtime emission of `scheduler.routed`, `task.preempted`, and in-flight cancellation enforcement: `internal/worker/runtime.go`
- File-backed scheduler policy inspection and reload now optionally replicate through a shared SQLite-backed policy store for multi-process convergence: `GET /v2/control-center/policy` and `POST /v2/control-center/policy/reload`
- Distributed diagnostics now surface cross-tenant boundary pairs and the metadata keys used to resolve tenant/owner isolation in scheduler events and reports.
- Local benchmark: `docs/reports/benchmark-report.md`

## Fresh benchmark snapshot

- `BenchmarkSchedulerDecide-8 = 51.08 ns/op`

## Remaining gaps

- No open fairness-distribution gaps remain in the current scheduler policy scope; fairness can now coordinate through memory, shared SQLite, or a remote HTTP service backend.
