# Review Readiness Matrix

## Distributed Closure Scorecard

This document is the stable closeout scorecard for the distributed readiness slices completed through `2026-03-14`. It links the operator-facing artifact set needed for release, reliability, and milestone review without requiring Linear comments to restate the evidence by hand.

| Surface | Status | Latest evidence | Review-ready meaning |
| --- | --- | --- | --- |
| Executor readiness | ready | `docs/reports/kubernetes-live-smoke-report.json`, `docs/reports/ray-live-smoke-report.json`, `docs/reports/mixed-workload-validation-report.md` | Real Kubernetes and Ray smoke runs passed, and mixed-workload routing proved local / Kubernetes / Ray selection in one control-plane matrix. |
| Validation bundle | ready | `docs/reports/live-validation-index.md`, `docs/reports/live-validation-summary.json`, `docs/reports/live-validation-runs/20260314T164647Z/summary.json` | The latest repo-native bundle (`20260314T164647Z`, generated `2026-03-14T16:46:57.671520+00:00`) exposes canonical bundle paths, per-executor logs, and closeout commands. |
| Closure and coordination | ready | `docs/reports/epic-closure-readiness-report.md`, `docs/reports/multi-node-coordination-report.md`, `docs/reports/long-duration-soak-report.md` | Same-day closure evidence covers longer local soak, mixed workload execution, and concrete two-node shared-queue coordination. |
| Event bus durability rollout | contract_ready | `docs/reports/event-bus-reliability-report.md`, `docs/reports/replicated-event-log-durability-rollout-contract.md`, `docs/reports/broker-failover-fault-injection-validation-pack.md` | The current runtime exposes the replicated durability plan and rollout checks, while the failover pack defines the validation required before any live broker-backed durability claim is made. |
| Review-pack artifact stability | ready | `docs/reports/review-readiness.md`, `docs/reports/epic-closure-readiness-report.md`, `docs/reports/event-bus-reliability-report.md`, `docs/reports/live-validation-index.md` | Distributed closeout can point to one repo-native artifact set instead of ad-hoc summary prose. |

## Stable Artifact Set

- `docs/reports/review-readiness.md`
- `docs/reports/epic-closure-readiness-report.md`
- `docs/reports/live-validation-index.md`
- `docs/reports/live-validation-summary.json`
- `docs/reports/event-bus-reliability-report.md`
- `docs/reports/replicated-event-log-durability-rollout-contract.md`
- `docs/reports/broker-failover-fault-injection-validation-pack.md`
- `docs/reports/mixed-workload-validation-report.md`
- `docs/reports/kubernetes-live-smoke-report.json`
- `docs/reports/ray-live-smoke-report.json`

## Baseline Coverage

- `OPE-176`
  - ADR and migration boundary docs exist.
  - Review notes are captured in `docs/reports/migration-plan-review-notes.md`.
- `OPE-177`
  - Task protocol and state model are codified in `internal/domain/*`.
  - Supporting reports exist in `docs/reports/task-protocol-spec.md` and `docs/reports/state-machine-validation-report.md`.
- `OPE-178`
  - Queue reliability evidence includes dead-letter replay, lease expiry recovery, API replay endpoints, and a `1k` no-duplicate-consumption test.
  - Supporting reports exist in `docs/reports/queue-reliability-report.md` and `docs/reports/lease-recovery-report.md`.
- `OPE-179`
  - Scheduler policy coverage includes budget guardrails, backpressure, preemptible concurrency, and tool-aware routing for GPU/browser workloads.
  - Supporting report: `docs/reports/scheduler-policy-report.md`.
- `OPE-180`
  - Worker runtime exposes lifecycle snapshots and `/debug/status` visibility for heartbeat, latest task, and result counters.
  - Supporting report: `docs/reports/worker-lifecycle-validation-report.md`.
- `OPE-181`
  - Real Kubernetes validation has passed and is persisted in `docs/reports/kubernetes-live-smoke-report.json`.
- `OPE-182`
  - Real Ray validation has passed and is persisted in `docs/reports/ray-live-smoke-report.json`.
- `OPE-183`
  - Event bus evidence includes replay-first subscriptions, webhook fanout, recorder sink coverage, and SSE replay/filter behavior by task or trace.
  - Supporting report: `docs/reports/event-bus-reliability-report.md`.
- `OPE-184`
  - Audit and debug surfaces include trace summary endpoints, trace timeline lookup, worker lifecycle snapshots, and `trace_count` metrics visibility.
  - Supporting report: `docs/reports/go-control-plane-observability-report.md`.
- `OPE-185`
  - Migration evidence includes a shadow matrix across multiple sample tasks with matched terminal states and matched event sequences.
  - Supporting report: `docs/reports/migration-readiness-report.md`.
- `OPE-186`
  - Benchmark evidence includes a repeatable matrix runner plus `50x8`, `100x12`, `1000x24`, and `2000x24` soak runs with zero failures.
  - Supporting reports: `docs/reports/benchmark-readiness-report.md`, `docs/reports/benchmark-matrix-report.json`, and `docs/reports/long-duration-soak-report.md`.
- `OPE-175`
  - Epic-level evidence includes longer-duration soak (`2000x24`), mixed workload validation across `local` / `kubernetes` / `ray`, and a concrete two-node shared-queue coordination proof.
  - Supporting report: `docs/reports/epic-closure-readiness-report.md`.

## Follow-up Hardening

- Production-grade capacity certification can remain a follow-up track beyond the current rewrite closure.
- No dedicated leader-election layer exists yet; current evidence is limited to a local two-node shared-SQLite coordination proof.
- Higher-scale external-store validation is still pending beyond the current SQLite-backed scope.
- Replicated broker-backed durability remains intentionally pre-rollout until the failover and checkpoint-fencing pack is backed by a live adapter.
