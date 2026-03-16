# Scheduler Policy Report

## Scope

- Run date: 2026-03-16
- Goal: refresh `OPE-179` scheduler closeout evidence so policy guardrails, fairness behavior, preemption, and executor routing can be reviewed from one current repo-native summary.

## Admission And Fairness Policy

- Budget guardrail rejects a task when `budget_cents` exceeds `BudgetRemaining`.
- Queue backpressure rejects non-exempt work when `QueueDepth >= MaxQueueDepth`.
- Tenant concurrency rejects new work when `CurrentRunning >= ConcurrentLimit`.
- Urgent work can reclaim preemptible capacity when `PreemptibleExecutions > 0`, returning a decision with `Preemption.Required=true`.
- Fairness windows only throttle non-urgent, non-high-risk work and only when another tenant is active inside the configured window.
- Fairness state can run in process memory, shared SQLite, or through a remote HTTP fairness service.

## Routing Order

The scheduler applies executor routing in this order:

1. `required_executor` hard override
2. tool-aware routing from `tool_executors`
3. computed high-risk routing to `high_risk_executor`
4. fallback routing to `default_executor`

Current default routing evidence from `DefaultRoutingRules()`:

- `gpu -> ray`
- `browser -> kubernetes`
- high-risk tasks -> `kubernetes`
- low/medium-risk fallback -> `local`

## Runtime Evidence

- Core routing and admission logic: `internal/scheduler/scheduler.go`
- Policy loading, reload, and shared SQLite policy replication: `internal/scheduler/policy_store.go`
- Fairness backends: `internal/scheduler/fairness.go`, `internal/scheduler/sqlite_fairness.go`, and `internal/scheduler/http_fairness.go`
- Scheduler policy API snapshot: `GET /v2/control-center/policy`
- Scheduler policy reload API: `POST /v2/control-center/policy/reload`
- Remote fairness service surface: `/internal/scheduler/fairness/{throttle,record,snapshot}`
- Unit coverage: `internal/scheduler/scheduler_test.go`
- Policy endpoint coverage: `internal/api/server_test.go`
- Runtime routing/preemption events: `internal/worker/runtime.go`

## Benchmark Reference

- Baseline microbenchmark reference: `docs/reports/benchmark-report.md`
- Current scheduler microbenchmark label: `BenchmarkSchedulerDecide-8`

The scheduler closeout report intentionally links the stable benchmark reference instead of copying a stale performance narrative into policy docs.

## Closeout Summary

The current scheduler surface now has repo-native evidence for:

- budget rejection and queue backpressure
- tenant fairness with memory, SQLite, and HTTP coordination modes
- live urgent-task preemption when reclaimable capacity exists
- deterministic executor routing precedence across explicit pins, tool requirements, computed risk, and default fallback

That is sufficient to review the current scheduler policy scope without reconstructing behavior from scattered code paths or older benchmark notes.
