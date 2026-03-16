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
  - Follow-up digest captures the remaining subscriber takeover executability caveats in `docs/reports/subscriber-takeover-executability-follow-up-digest.md`.
- `OPE-184`
  - Audit and debug surfaces include trace summary endpoints, trace timeline lookup, worker lifecycle snapshots, and `trace_count` metrics visibility.
  - Supporting report: `docs/reports/go-control-plane-observability-report.md`.
  - Follow-up digests capture the remaining tracing-backend and telemetry-pipeline caveats in `docs/reports/tracing-backend-follow-up-digest.md` and `docs/reports/telemetry-pipeline-controls-follow-up-digest.md`.
- `OPE-185`
  - Migration evidence includes a shadow matrix across multiple sample tasks with matched terminal states, matched event sequences, and an anonymized corpus coverage scorecard.
  - Supporting report: `docs/reports/migration-readiness-report.md`.
  - Follow-up digests capture the remaining live shadow comparison, rollback safeguard, and production corpus coverage caveats in `docs/reports/live-shadow-comparison-follow-up-digest.md`, `docs/reports/rollback-safeguard-follow-up-digest.md`, and `docs/reports/production-corpus-migration-coverage-digest.md`.
- `OPE-186`
  - Benchmark evidence includes a repeatable matrix runner plus `50x8`, `100x12`, `1000x24`, and `2000x24` soak runs with zero failures.
  - Supporting reports: `docs/reports/benchmark-readiness-report.md`, `docs/reports/benchmark-matrix-report.json`, and `docs/reports/long-duration-soak-report.md`.
- `OPE-175`
  - Epic-level evidence includes longer-duration soak (`2000x24`), mixed workload validation across `local` / `kubernetes` / `ray`, and a concrete two-node shared-queue coordination proof.
  - Supporting report: `docs/reports/epic-closure-readiness-report.md`.
  - Follow-up digests capture the remaining cross-process coordination and validation bundle continuation caveats in `docs/reports/cross-process-coordination-boundary-digest.md` and `docs/reports/validation-bundle-continuation-digest.md`.

## Follow-up Hardening

- Production-grade capacity certification can remain a follow-up track beyond the current rewrite closure.
- No dedicated leader-election layer exists yet; current evidence is limited to a local two-node shared-SQLite coordination proof.
- Higher-scale external-store validation is still pending beyond the current SQLite-backed scope.

## Parallel follow-up digests

- `OPE-264` / `BIG-PAR-075` — external tracing backend and span-propagation caveats are consolidated in `docs/reports/tracing-backend-follow-up-digest.md`.
- `OPE-265` / `BIG-PAR-076` — telemetry pipeline, sampling policy, and high-cardinality caveats are consolidated in `docs/reports/telemetry-pipeline-controls-follow-up-digest.md`.
- `OPE-266` / `BIG-PAR-077` — live shadow traffic comparison caveats are consolidated in `docs/reports/live-shadow-comparison-follow-up-digest.md`.
- `OPE-267` / `BIG-PAR-078` — rollback safeguard caveats are consolidated in `docs/reports/rollback-safeguard-follow-up-digest.md`.
- `OPE-268` / `BIG-PAR-079` — production corpus coverage caveats are consolidated in `docs/reports/production-corpus-migration-coverage-digest.md`.
- `OPE-269` / `BIG-PAR-080` — subscriber takeover executability caveats are consolidated in `docs/reports/subscriber-takeover-executability-follow-up-digest.md`.
- `OPE-270` / `BIG-PAR-081` — cross-process coordination caveats are consolidated in `docs/reports/cross-process-coordination-boundary-digest.md`.
- `OPE-271` / `BIG-PAR-082` — validation bundle continuation caveats are consolidated in `docs/reports/validation-bundle-continuation-digest.md`.
