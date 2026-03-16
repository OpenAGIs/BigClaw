# OpenClaw Comparison and Parallel Gap Analysis

## Context

- Comparison date: 2026-03-14
- Compared repo: `openclaw/openclaw`
- Local repo: `OpenAGIs/BigClaw`

## What BigClaw should borrow from OpenClaw

- Treat the control plane as a durable, always-on service boundary rather than a single-process demo harness.
- Make multi-worker and multi-node visibility first-class API payloads so UI and operational review surfaces can reason about distributed state directly.
- Keep validation artifacts isolated per run so concurrent live verification does not collapse into shared-state ambiguity.
- Package cluster and executor health as repo-native evidence that planning, review, and Linear execution slices can reference directly.

## What BigClaw should not borrow

- End-user messaging channel product scope.
- Consumer assistant UX assumptions.
- Personal workspace and device-pairing abstractions that do not map to the execution control plane.

## Replay and checkpoint durability track

The current BigClaw Go event plane now has replay-capable APIs, subscriber-group fencing, scheduler coordination, and service-style event-log integration points, but the execution path still needs a stronger distributed durability contract.

### Closed baseline

- `OPE-199` introduced the durable-event-log direction and backend capability framing.
- `OPE-203` added subscriber checkpoint and resume semantics.
- `OPE-205` tightened monotonic checkpoint expectations.
- `OPE-210` introduced subscriber-group lease coordination so stale writers cannot move shared progress backward.
- `OPE-212` through `OPE-217` defined replay compaction, capability probing, dedup semantics, expired-cursor fallback, and takeover validation evidence.

### Remaining gaps

- Replay retention watermarks are now visible in runtime payloads, SQLite-backed logs now persist trimmed replay boundaries across restarts, expired durable checkpoints now fail closed with reset guidance, and checkpoint resets now leave a persisted operator history trail; memory-only deployments are still bounded by in-process history and broker/quorum retention remains future work.
- Service-style SQLite and HTTP-backed coordination improve sharing, but replicated broker or quorum-backed durability is still future work.
- Downstream consumers still need idempotent handlers and durable dedupe stores; the system remains replay-safe, not globally exactly-once.
- Parallel validation for Kubernetes, Ray, and shared-queue takeover should continue to be bundled as repo-native evidence.

### Current rollout gate

- `OPE-222` now makes the replicated durability rollout contract explicit in repo-native form:
  - rollout metadata lives in `bigclaw-go/internal/events/durability.go` so debug/control-plane payloads can advertise checks, failure domains, evidence links, and broker bootstrap readiness;
  - `bigclaw-go/docs/reports/replicated-event-log-durability-rollout-contract.md` defines the minimum publish-ack, replay/checkpoint, retention-boundary, and failover expectations before a replicated adapter can be called rollout-ready.

## Recommended BigClaw parallel mainline

1. Multi-worker and multi-node control-plane observability.
2. Shared-queue coordination and lease-safety hardening.
3. Parallel validation matrix and evidence bundling for local, Kubernetes, and Ray execution.
4. Distributed scheduler and executor diagnostics for capacity, routing, and recovery visibility.

## Distributed diagnostics follow-up digests

- `OPE-264` / `BIG-PAR-075` tracks the distributed trace export bundle path in `bigclaw-go/internal/api/distributed.go` and the remaining external tracing backend and span-propagation caveats in `bigclaw-go/docs/reports/tracing-backend-follow-up-digest.md`.
- `OPE-265` / `BIG-PAR-076` tracks the remaining telemetry pipeline, sampling policy, and high-cardinality caveats in `bigclaw-go/docs/reports/telemetry-pipeline-controls-follow-up-digest.md`.

## Migration follow-up digests

- `OPE-266` / `BIG-PAR-092` tracks the repo-native live shadow mirror parity drift scorecard and the remaining live shadow traffic comparison caveats in `bigclaw-go/docs/reports/live-shadow-comparison-follow-up-digest.md`.
- `OPE-267` / `BIG-PAR-078` tracks the remaining rollback safeguard caveats in `bigclaw-go/docs/reports/rollback-safeguard-follow-up-digest.md`.
- `OPE-268` / `BIG-PAR-079` tracks the remaining production corpus migration coverage caveats in `bigclaw-go/docs/reports/production-corpus-migration-coverage-digest.md`.
- `OPE-269` / `BIG-PAR-080` tracks the remaining subscriber takeover executability caveats in `bigclaw-go/docs/reports/subscriber-takeover-executability-follow-up-digest.md`.
- `OPE-270` / `BIG-PAR-081` tracks the remaining cross-process coordination caveats in `bigclaw-go/docs/reports/cross-process-coordination-boundary-digest.md`.
- `OPE-271` / `BIG-PAR-082` tracks the remaining validation bundle continuation caveats in `bigclaw-go/docs/reports/validation-bundle-continuation-digest.md`, with the current local lineage overlay summarized in `bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json`, the shared-queue companion exported in `bigclaw-go/docs/reports/shared-queue-companion-summary.json`, and the current policy result captured in `bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json`.
