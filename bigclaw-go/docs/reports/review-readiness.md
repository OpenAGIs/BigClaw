# Review Readiness Matrix

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
- `OPE-233`
  - Distributed diagnostics now include Ray executor readiness and latest live-validation references without sending reviewers back to raw run logs first.
  - Supporting reports: `docs/reports/ray-live-smoke-report.json` and `docs/reports/live-validation-index.md`.
- `OPE-234`
  - Replicated event-log rollout readiness is captured as an operator-facing contract with explicit failure domains, rollout phases, and validation gates.
  - Supporting reports: `docs/reports/replicated-event-log-durability-rollout-contract.md` and `docs/reports/event-bus-reliability-report.md`.
- `OPE-236`
  - Distributed diagnostics now include Kubernetes executor readiness and the latest bundled live-validation evidence.
  - Supporting reports: `docs/reports/kubernetes-live-smoke-report.json` and `docs/reports/live-validation-index.md`.
- `OPE-237`
  - Live validation now exposes one normalized, timestamped evidence bundle with stable per-executor canonical report paths and isolated per-run logs.
  - Supporting reports: `docs/reports/live-validation-index.md` and `docs/reports/live-validation-runs/20260314T164647Z/README.md`.
- `OPE-238`
  - The distributed rollout review pack now tells reviewers exactly which local, service-backed, and future replicated-durability artifacts belong in Linear closeout comments and GitHub review attachments.
  - Supporting reports: `docs/reports/review-readiness.md`, `docs/reports/live-validation-index.md`, `docs/reports/event-bus-reliability-report.md`, and `docs/openclaw-parallel-gap-analysis.md`.
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
- The replicated durability track is still contract-first; until a live broker-backed adapter exists, reviewers should treat the normalized validation bundle plus the replicated rollout contract as the release-ready evidence set.
