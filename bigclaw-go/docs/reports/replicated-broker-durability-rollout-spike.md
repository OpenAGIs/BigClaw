# Replicated Broker Durability Rollout Spike

## Scope

This spike is the repo-native planning artifact for `OPE-4` / `BIG-PAR-099`. It defines the minimum adapter contract BigClaw must satisfy before the project can honestly claim broker-backed or quorum-backed event durability beyond the current SQLite and process-local proof boundary.

The goal here is not to ship a production adapter in this slice. The goal is to freeze the rollout gates, the current evidence boundary, and the smallest follow-on implementation slices so later work does not have to rediscover them.

## Current proof boundary

### What is already proven

- `memory` and recorder-backed replay semantics are implemented in the current process-local bus.
- `sqlite` and local checkpoint durability semantics are covered as single-node evidence, not replicated durability evidence.
- The runtime already exposes `event_durability`, `event_durability_rollout`, and broker bootstrap readiness through control-plane/debug payloads.
- Checked-in reviewer artifacts already cover failover-stub scenarios, retention-boundary handling, checkpoint fencing, and the current rollout scorecard.

### What is not yet proven

- No live replicated broker adapter is wired into publish, replay, or checkpoint paths.
- No current runtime path proves replicated publish acknowledgement semantics.
- No current runtime path proves durable replay/checkpoint alignment against a real broker failover boundary.
- SQLite-backed proofs must not be described as equivalent to broker- or quorum-backed durability.

## Minimum replicated adapter contract

| Contract area | Minimum requirement | Current status |
| --- | --- | --- |
| Publish acknowledgement | Success must mean replicated commit or explicit `unknown_commit` classification | `contract_only` |
| Replay sequence mapping | Durable broker offsets must map back to one monotonic portable sequence domain | `contract_only` |
| Checkpoint fencing | Stale writers must be rejected after takeover using durable sequence and ownership epoch metadata | `harness_proven` |
| Retention boundaries | Oldest/newest retained replay boundaries must be operator-visible and expired checkpoints must fail closed | `harness_proven` |
| Live fanout isolation | Replay catch-up and broker lag must not stall live SSE/process-local subscribers | `harness_proven` |
| Bootstrap readiness | Driver, broker URLs, topic, consumer group, and timing knobs must validate before rollout claims | `repo_surface_ready` |

## Required rollout gates

### Gate 1: publish acknowledgement

- Replicated publish must distinguish `committed`, `rejected`, and `unknown_commit`.
- Client-visible success cannot mean leader-local enqueue only.
- Audit/replay evidence must let operators reconcile ambiguous publish attempts after failover.

### Gate 2: replay and checkpoint sequence alignment

- Replay cursors and checkpoint commits must reference the same durable sequence space.
- Failover cannot reinterpret the cursor domain or allow stale ownership to regress progress.
- Partition-aware providers must still expose one portable monotonic sequence contract to callers.

### Gate 3: retention and recovery visibility

- Operators must be able to see the current oldest/newest retained replay boundaries.
- Expired checkpoints must fail closed with reset guidance instead of silently fast-forwarding.
- Retention policy must be called out separately from checkpoint inactivity cleanup.

### Gate 4: live delivery isolation

- Replay catch-up must stay on a different lane from live delivery.
- Broker lag, backfill, or consumer recovery cannot become a hidden source of SSE latency.
- The runtime needs one explicit proof that live-only subscribers still receive prompt delivery while replay drains.

## Current evidence map

| Gate | Current evidence | Boundary |
| --- | --- | --- |
| Publish acknowledgement | `docs/reports/replicated-event-log-durability-rollout-contract.md`, `docs/reports/broker-durability-rollout-scorecard.json` | contract only |
| Replay/checkpoint alignment | `docs/reports/broker-failover-fault-injection-validation-pack.md`, `docs/reports/broker-checkpoint-fencing-proof-summary.json` | deterministic stub proof |
| Retention visibility | `docs/reports/replay-retention-semantics-report.md`, `docs/reports/broker-retention-boundary-proof-summary.json` | deterministic stub proof |
| Live fanout isolation | `docs/reports/broker-stub-live-fanout-isolation-evidence-pack.json` | local broker-stub proof |
| Runtime posture | `internal/events/durability.go`, `docs/reports/event-bus-reliability-report.md`, `docs/reports/durability-rollout-scorecard.json` | repo-native contract and scorecard only |

## Follow-on implementation slices

1. Publish-ack outcome ledger
   Freeze the adapter result contract for `committed`, `rejected`, and `unknown_commit`, and thread it through publish/audit surfaces.
2. Durable sequence bridge
   Define how broker offsets or quorum positions map into `Position.Sequence` and checkpoint ownership metadata.
3. Retention watermark and expiry surface
   Expose provider retention watermarks and make expired-checkpoint handling provider-backed rather than stub-backed.
4. Provider-backed live-handoff proof
   Re-run the live fanout isolation drill against the first real broker adapter instead of the local stub.

These slices are intentionally small and independently verifiable so they can be run in parallel without merge-heavy overlap.

## Honest rollout statement

Until a real replicated adapter passes those follow-on slices, BigClaw can accurately claim:

- the replicated durability contract is defined;
- the rollout gates are visible in runtime and checked-in reviewer artifacts;
- some failure scenarios are proven with deterministic broker-stub evidence;
- current single-node SQLite and local replay proofs are not equivalent to replicated durability.

It cannot yet accurately claim:

- replicated broker durability is implemented;
- publish success means replicated commit;
- failover-safe replay and checkpoint behavior is proven against a real provider backend.

## Repo evidence

- `internal/events/durability.go`
- `docs/reports/event-bus-reliability-report.md`
- `docs/reports/replicated-event-log-durability-rollout-contract.md`
- `docs/reports/broker-durability-rollout-scorecard.json`
- `docs/reports/durability-rollout-scorecard.json`
- `docs/reports/broker-checkpoint-fencing-proof-summary.json`
- `docs/reports/broker-retention-boundary-proof-summary.json`
- `docs/reports/broker-stub-live-fanout-isolation-evidence-pack.json`
