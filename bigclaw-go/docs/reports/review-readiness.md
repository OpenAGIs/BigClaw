# Review Readiness Matrix

## Current Distributed Platform Refresh Pack

- `OPE-254`
  - Long-duration soak and benchmark closeout has been refreshed around the canonical benchmark pack.
  - Primary reports: `docs/reports/benchmark-readiness-report.md`, `docs/reports/long-duration-soak-report.md`, and `docs/reports/epic-closure-readiness-report.md`.
- `OPE-255`
  - Operations foundation evidence now rolls up control-center, run-detail, policy, and collaboration surfaces through one repo-native summary.
  - Primary reports: `docs/reports/v2-phase1-operations-foundation-report.md` and `docs/reports/linear-project-sync-summary.md`.
- `OPE-256`
  - Scheduler policy closeout references now live with the latest routing, fairness, and preemption evidence instead of older review notes.
  - Primary reports: `docs/reports/scheduler-policy-report.md` and `docs/reports/epic-closure-readiness-report.md`.
- `OPE-257`
  - Review and closeout navigation now links the latest distributed-platform refresh set from this matrix and the epic closure report.
  - Primary reports: `docs/reports/review-readiness.md`, `docs/reports/epic-closure-readiness-report.md`, and `docs/reports/linear-project-sync-summary.md`.

## Done

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

## Reviewer Navigation

- Start with `docs/reports/review-readiness.md` for the current refresh matrix.
- Use `docs/reports/epic-closure-readiness-report.md` for the benchmark, mixed-workload, and multi-node closeout spine.
- Use `docs/reports/v2-phase1-operations-foundation-report.md` for operations foundation and control-center evidence.
- Use `docs/reports/scheduler-policy-report.md` for scheduler fairness, routing, and preemption evidence.
- Use `docs/reports/linear-project-sync-summary.md` for the Linear project and issue crosswalk.

## Follow-up Hardening

- Production-grade capacity certification can remain a follow-up track beyond the current rewrite closure.
- No dedicated leader-election layer exists yet; current evidence is limited to a local two-node shared-SQLite coordination proof.
- Higher-scale external-store validation is still pending beyond the current SQLite-backed scope.
