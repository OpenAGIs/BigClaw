# Replicated Event-Log Durability Rollout Contract

## Scope

This report defines the rollout contract for `OPE-222` / `BIG-PAR-035`: the minimum runtime, operator, and validation expectations BigClaw must satisfy before claiming a replicated broker-backed or quorum-backed event log.

It builds on the provider-neutral adapter boundary in `docs/reports/broker-event-log-adapter-contract.md`, the retention semantics in `docs/reports/replay-retention-semantics-report.md`, and the failover scenarios in `docs/reports/broker-failover-fault-injection-validation-pack.md`.

## Current baseline

- `internal/events/durability.go` already declares `broker_replicated` as the target backend and surfaces the active durability plan through bootstrap and debug payloads, including broker bootstrap readiness derived from configured driver / URLs / topic settings.
- The same payload now carries a `broker_probe` dry-run summary so operators can inspect expected broker capabilities, failure modes, and config-validation results before a live provider adapter is wired.
- `cmd/bigclawd/main.go` validates broker runtime config but intentionally stops before instantiating a live replicated adapter.
- `docs/reports/event-bus-reliability-report.md` and `docs/reports/broker-failover-fault-injection-validation-pack.md` describe the portability and validation direction, but prior to this slice the rollout gate itself was not captured as one explicit contract.

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
- dry-run broker capability expectations and config-validation results when the replicated target is selected
- rollout checks and their failure modes
- failure-domain summaries
- references to the supporting validation pack and rollout contract documents

The current repo-native source for these signals is the `event_durability` payload exposed through `GET /debug/status`.

## Validation evidence required before a live adapter lands

- debug/control-plane payload proving the active runtime advertises the replicated rollout contract and broker bootstrap readiness state
- failover validation artifacts matching the scenario matrix in `docs/reports/broker-failover-fault-injection-validation-pack.md`
- replay retention diagnostics proving expired checkpoints are surfaced explicitly
- checkpoint takeover evidence proving stale writers cannot regress durable progress

## Repo evidence

- `internal/events/durability.go`
- `internal/events/durability_test.go`
- `internal/api/server_test.go`
- `cmd/bigclawd/main.go`
- `docs/reports/event-bus-reliability-report.md`
- `docs/reports/broker-failover-fault-injection-validation-pack.md`
- `docs/reports/replay-retention-semantics-report.md`
