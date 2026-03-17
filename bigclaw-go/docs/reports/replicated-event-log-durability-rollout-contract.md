# Replicated Event-Log Durability Rollout Contract

## Scope

This report defines the rollout contract for `OPE-222` / `BIG-PAR-035`: the minimum runtime, operator, and validation expectations BigClaw must satisfy before claiming a replicated broker-backed or quorum-backed event log.

It builds on the provider-neutral adapter boundary in `docs/reports/broker-event-log-adapter-contract.md`, the retention semantics in `docs/reports/replay-retention-semantics-report.md`, the failover scenarios in `docs/reports/broker-failover-fault-injection-validation-pack.md`, the reviewer-facing `BF-05` proof in `docs/reports/ambiguous-publish-outcome-proof-summary.json`, and the planning-oriented rollout spike at `docs/reports/replicated-broker-durability-rollout-spike.md`.

## Current baseline

- `internal/events/durability.go` already declares `broker_replicated` as the target backend and surfaces the active durability plan through bootstrap and debug payloads, including broker bootstrap readiness derived from configured driver / URLs / topic settings.
- `docs/reports/broker-validation-summary.json` now exports the same broker bootstrap review summary into the validation bundle path, including driver / URL / topic / consumer-group completeness, validation errors, and whether the runtime posture is `contract_only`, `stub_driver_only`, or `fail_closed_until_adapter_exists`.
- `docs/reports/broker-durability-rollout-scorecard.json` now captures the same rollout posture as one checked-in machine-readable scorecard, including blockers, missing evidence, and broker bootstrap readiness.
- `cmd/bigclawd/main.go` validates broker runtime config but intentionally stops before instantiating a live replicated adapter.
- `docs/reports/event-bus-reliability-report.md` and `docs/reports/broker-failover-fault-injection-validation-pack.md` describe the portability and validation direction, but prior to this slice the rollout gate itself was not captured as one explicit contract.
- `docs/reports/replicated-broker-durability-rollout-spike.md` now makes the current proof boundary explicit so SQLite and process-local evidence do not get overstated as replicated durability evidence.
- `docs/reports/broker-checkpoint-fencing-proof-summary.json`, `docs/reports/broker-retention-boundary-proof-summary.json`, and `docs/reports/ambiguous-publish-outcome-proof-summary.json` now split the deterministic stub matrix into reviewable rollout-gate proofs for checkpoint fencing, retention expiry handling, and ambiguous publish classification.

## Runtime contract

### Publish

- Success must mean the event reached the configured replicated durability boundary, not merely a leader-local buffer.
- Ambiguous publish outcomes must be classifiable as `committed`, `rejected`, or `unknown_commit` using replay and audit evidence.
- Event identity fields (`id`, `task_id`, `trace_id`, `event_type`) must remain stable across append, replay, and duplicate delivery windows.

### Replay and live handoff

- Replay order must be monotonic within the provider's durable ordering scope and mapped back to the portable `Position.Sequence`.
- Live fanout must remain decoupled from broker catch-up lag so replay recovery does not stall process-local subscribers or SSE clients.
- Replay resume and checkpoint acknowledgement must reference the same durable sequence space after failover, reconnect, or consumer takeover.

### Checkpoints

- Checkpoints must remain monotonic by durable sequence, not by wall-clock arrival order.
- Stale writers must be fenced with lease/epoch metadata before a replicated backend is considered rollout-safe.
- Retention or compaction must not silently reinterpret an expired checkpoint as a valid later resume point.

## Rollout phases

| Phase | Goal | Exit condition |
| --- | --- | --- |
| `contract` | keep provider-neutral append/replay/checkpoint expectations stable | runtime plan and docs expose rollout checks, failure domains, and verification evidence |
| `stubbed_validation` | prove failover and checkpoint accounting with deterministic harnesses | scenario runner can emit the report shape in `broker-failover-fault-injection-validation-pack.md` |
| `single_backend_trial` | wire one replicated adapter without changing callers | one provider backend can publish, replay, and checkpoint behind the existing event-log contract |
| `rollout_ready` | claim operator-safe replicated durability | pass failover, retention-boundary, and checkpoint-fencing evidence for the chosen provider |

## Failure domains

### Broker leader or quorum loss

- Risk: acknowledged writes may be ambiguous until leadership stabilizes.
- Required mitigation: replicated publish acknowledgements plus replay-visible reconciliation of ambiguous outcomes.

### Checkpoint store failover

- Risk: stale consumers can overwrite newer progress after ownership transfer.
- Required mitigation: durable checkpoint fencing using sequence monotonicity and lease epoch metadata.

### Retention or compaction drift

- Risk: replay cursors or checkpoints may reference trimmed history even though their shape is still valid.
- Required mitigation: expose retention watermarks and fail closed on expired cursors until an explicit reset policy is invoked.

## Required operator signals

- active backend and target backend
- replication factor or quorum expectation
- whether publisher acknowledgement is required before success is reported
- rollout checks and their failure modes
- failure-domain summaries
- references to the supporting validation pack and rollout contract documents

The current repo-native sources for these signals are the `event_durability` payload and its nested `rollout_scorecard`, plus the top-level `event_durability_rollout` alias exposed through `GET /debug/status` and `/metrics`. Checked-in reviewer artifacts live at `docs/reports/broker-durability-rollout-scorecard.json`, `docs/reports/durability-rollout-scorecard.json`, `docs/reports/broker-checkpoint-fencing-proof-summary.json`, and `docs/reports/broker-retention-boundary-proof-summary.json`.

`docs/reports/broker-validation-summary.json` is a reviewer-facing readiness surface, not evidence of a live broker adapter. A `ready` bootstrap summary only means the configured driver / URLs / topic / consumer group satisfy the pre-adapter contract; it does not prove replicated publish durability, live replay correctness, or a non-stub broker implementation.


## Validation evidence required before a live adapter lands

- debug/control-plane payload proving the active runtime advertises the replicated rollout contract and broker bootstrap readiness state
- failover validation artifacts matching the scenario matrix in `docs/reports/broker-failover-fault-injection-validation-pack.md`
- ambiguous-publish proof summary at `docs/reports/ambiguous-publish-outcome-proof-summary.json`
- checkpoint-fencing proof summary at `docs/reports/broker-checkpoint-fencing-proof-summary.json`
- retention-boundary proof summary at `docs/reports/broker-retention-boundary-proof-summary.json`
- replay retention diagnostics proving expired checkpoints are surfaced explicitly
- checkpoint takeover evidence proving stale writers cannot regress durable progress

## Repo evidence

- `internal/events/durability.go`
- `internal/events/durability_test.go`
- `internal/api/server_test.go`
- `cmd/bigclawd/main.go`
- `docs/reports/event-bus-reliability-report.md`
- `docs/reports/ambiguous-publish-outcome-proof-summary.json`
- `docs/reports/replicated-broker-durability-rollout-spike.md`
- `docs/reports/broker-checkpoint-fencing-proof-summary.json`
- `docs/reports/broker-retention-boundary-proof-summary.json`
- `docs/reports/broker-failover-fault-injection-validation-pack.md`
- `docs/reports/replay-retention-semantics-report.md`
