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
  - Follow-up digest captures the deterministic local harness, the live two-node companion proof, and the remaining shared-durable ownership caveats in `docs/reports/subscriber-takeover-executability-follow-up-digest.md`.
- `OPE-184`
  - Audit and debug surfaces include trace summary endpoints, trace timeline lookup, worker lifecycle snapshots, and `trace_count` metrics visibility.
  - Supporting report: `docs/reports/go-control-plane-observability-report.md`.
  - The distributed diagnostics export now packages recent trace summaries, reviewer navigation, validation-artifact references, the `BF-05` ambiguous publish proof in `docs/reports/ambiguous-publish-outcome-proof-summary.json`, and backend limitations through `GET /v2/reports/distributed/export`.
  - Follow-up evidence captures the remaining tracing-backend and telemetry-pipeline caveats in `docs/reports/tracing-backend-follow-up-digest.md`, `docs/reports/telemetry-pipeline-controls-follow-up-digest.md`, and the machine-checkable `docs/reports/telemetry-sampling-cardinality-evidence-pack.json`.
- `OPE-185`
  - Migration evidence includes a shadow matrix across multiple sample tasks with matched terminal states, matched event sequences, an anonymized corpus coverage scorecard, and a repo-native live shadow mirror scorecard that summarizes parity drift and evidence freshness across the checked-in compare and matrix artifacts.
  - Supporting report: `docs/reports/migration-readiness-report.md`.
  - Follow-up digests capture the remaining live shadow comparison, rollback safeguard trigger surface, and production corpus coverage caveats in `docs/reports/live-shadow-comparison-follow-up-digest.md`, `docs/reports/rollback-safeguard-follow-up-digest.md`, `docs/reports/rollback-trigger-surface.json`, and `docs/reports/production-corpus-migration-coverage-digest.md`.
- `OPE-186`
  - Benchmark evidence includes a repeatable matrix runner plus `50x8`, `100x12`, `1000x24`, and `2000x24` soak runs with zero failures.
  - Supporting reports: `docs/reports/benchmark-readiness-report.md`, `docs/reports/benchmark-matrix-report.json`, `docs/reports/long-duration-soak-report.md`, `docs/reports/capacity-certification-report.md`, and `docs/reports/capacity-certification-matrix.json`.
- `OPE-175`
  - Epic-level evidence includes longer-duration soak (`2000x24`), mixed workload validation across `local` / `kubernetes` / `ray`, and a concrete two-node shared-queue coordination proof.
  - Supporting report: `docs/reports/epic-closure-readiness-report.md`.
  - Follow-up digests capture the remaining cross-process coordination and validation bundle continuation caveats in `docs/reports/cross-process-coordination-boundary-digest.md` and `docs/reports/validation-bundle-continuation-digest.md`, with the current multi-bundle lineage summarized in `docs/reports/validation-bundle-continuation-scorecard.json` and the current gate result captured in `docs/reports/validation-bundle-continuation-policy-gate.json`.
  - The coordination runtime capability matrix in `docs/reports/cross-process-coordination-capability-surface.json` makes the remaining coordination claims explicit as `live_proven`, `harness_proven`, and `contract_only` instead of treating all cross-process semantics as equally shipped.
  - Replicated broker durability remains a follow-up track with an explicit rollout spike in `docs/reports/replicated-broker-durability-rollout-spike.md`, the governing contract in `docs/reports/replicated-event-log-durability-rollout-contract.md`, and the current readiness posture in `docs/reports/broker-durability-rollout-scorecard.json`.

## Follow-up Hardening

- The capacity certification matrix is now repo-native and reviewer-facing, but it remains a single-instance evidence slice rather than a multi-tenant production admission control policy.
- A repo-native leader-election scaffold now exists through the subscriber-lease-backed `/coordination/leader` surface plus matching debug/control-center payloads, while the underlying proof remains local/shared-store scoped rather than broker-backed or quorum-backed.
- `docs/reports/leader-election-capability-surface.json` now captures the leader-election backend posture explicitly: shared SQLite is `live_proven`, shared-store takeover hardening is `harness_proven`, and broker/quorum ownership remains `contract_only`.
- A repo-native external-store validation lane now exists in `docs/reports/external-store-validation-report.json`, proving replay, checkpoint reset history, persisted retention boundaries, and shared-lease takeover behavior through the remote HTTP event-log service boundary. Its backend matrix now makes the remaining posture explicit as `http_remote_service=live_validated`, `broker_replicated=not_configured`, and `quorum_replicated=contract_only` instead of leaving broker-backed or quorum-backed durability as prose-only caveats.

## Parallel follow-up digests

- `OPE-264` / `BIG-PAR-075` — external tracing backend and span-propagation caveats are consolidated in `docs/reports/tracing-backend-follow-up-digest.md`, while the current reviewer bundle lives in `GET /v2/reports/distributed/export`.
- `OPE-265` / `BIG-PAR-076` — telemetry pipeline, sampling policy, and high-cardinality caveats are consolidated in `docs/reports/telemetry-pipeline-controls-follow-up-digest.md`, with the current review surface summarized in `docs/reports/telemetry-sampling-cardinality-evidence-pack.json`.
- `OPE-266` / `BIG-PAR-092` — repo-native live shadow mirror scorecard and remaining live shadow traffic comparison caveats are consolidated in `docs/reports/live-shadow-comparison-follow-up-digest.md`.
- `OPE-266` / `BIG-PAR-092` — the same checked-in mirror evidence is now visible at runtime via `GET /debug/status` (`live_shadow_mirror_scorecard`) and `GET /v2/control-center` (`distributed_diagnostics.live_shadow_mirror_scorecard`) so reviewers can inspect parity drift, freshness, and report links without opening the bundle first.
- `OPE-254` / `BIG-PAR-088` — rollback safeguard trigger-surface caveats are consolidated in `docs/reports/rollback-safeguard-follow-up-digest.md`, with the machine-checkable reviewer surface in `docs/reports/rollback-trigger-surface.json`. The same payload is now visible at runtime via `GET /debug/status` (`rollback_trigger_surface`) and `GET /v2/control-center` (`distributed_diagnostics.migration_review_pack.rollback_trigger_surface`).
- `OPE-268` / `BIG-PAR-079` — production corpus coverage caveats are consolidated in `docs/reports/production-corpus-migration-coverage-digest.md`.
- `OPE-269` / `BIG-PAR-080` — subscriber takeover executability caveats are consolidated in `docs/reports/subscriber-takeover-executability-follow-up-digest.md`.
- `OPE-261` / `BIG-PAR-085` — cross-process coordination caveats are consolidated in `docs/reports/cross-process-coordination-boundary-digest.md`, with the current runtime capability matrix summarized in `docs/reports/cross-process-coordination-capability-surface.json` using `live_proven`, `harness_proven`, and `contract_only` readiness labels.
- `OPE-271` / `BIG-PAR-082` — validation bundle continuation caveats are consolidated in `docs/reports/validation-bundle-continuation-digest.md`, with the latest multi-bundle lineage summarized in `docs/reports/validation-bundle-continuation-scorecard.json` and the latest gate result captured in `docs/reports/validation-bundle-continuation-policy-gate.json`.
