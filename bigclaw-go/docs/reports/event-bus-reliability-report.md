# Event Bus Reliability Report

## Scope

This report summarizes the current event bus reliability evidence and the next replicated-durability integration plan for `OPE-183` / `BIG-GO-008` and `OPE-206`.

## Implemented surfaces

- In-process publish/subscribe bus with replay history
- Recorder sink integration for audit/debug persistence
- Webhook sink integration for external fanout
- SSE stream via `GET /stream/events`
- Optional SSE replay and filtering via `replay=1`, `task_id`, and `trace_id`
<<<<<<< HEAD
- Replay-safe consumer delivery metadata via `EventDelivery`, including additive `delivery.mode`, `delivery.replay`, and `delivery.idempotency_key` fields
- Consumer dedup ledger/result contract covering duplicate, retryable-failure, and already-applied outcomes
- Subscriber-group checkpoint lease coordination via `/subscriber-groups/leases` and `/subscriber-groups/checkpoints`
=======
- Event backend capability and config-validation contract via `internal/events/backend_contract.go`
>>>>>>> origin/dcjcloud/ope-213-big-par-025-durability-capability-matrix-与后端配置校验

## Validated behaviors

- Published events are retained in replay history.
- New subscribers can request replayed events before switching to live events.
- Webhook sink receives serialized domain events.
- SSE streaming can deliver live events.
- SSE replay can filter to one trace without leaking unrelated events.
- Replay and live deliveries preserve the original event id while exposing an explicit delivery mode and stable downstream idempotency key.
- Subscriber-group checkpoint commits are fenced by lease token + epoch, so stale writers cannot advance ownership after takeover.
- Checkpoint offsets remain monotonic within a subscriber group and reject rollback writes.

## Evidence

- `internal/events/bus.go`
<<<<<<< HEAD
- `internal/events/durability.go`
=======
- `internal/events/backend_contract.go`
- `internal/events/backend_contract_test.go`
>>>>>>> origin/dcjcloud/ope-213-big-par-025-durability-capability-matrix-与后端配置校验
- `internal/events/bus_test.go`
- `internal/events/delivery.go`
- `internal/events/delivery_test.go`
- `internal/domain/consumer_dedup.go`
- `internal/domain/consumer_dedup_test.go`
- `internal/events/webhook.go`
- `internal/events/webhook_test.go`
- `internal/events/recorder_sink.go`
- `internal/events/subscriber_leases.go`
- `internal/events/subscriber_leases_test.go`
- `internal/api/server.go`
- `internal/api/server_test.go`
- `cmd/bigclawd/main.go`
- `internal/config/config.go`

<<<<<<< HEAD
## Current durability shape

- Runtime publish/subscribe remains in-process.
- Audit/debug persistence is recorder-backed, with optional JSONL sinking.
- The `events.DurabilityPlan` surface makes the active backend and the next replicated target explicit in bootstrap and `GET /debug/status`.
- Default plan is `memory -> broker_replicated` with replication factor `3`, and env overrides exist for:
  - `BIGCLAW_EVENT_LOG_BACKEND`
  - `BIGCLAW_EVENT_LOG_TARGET_BACKEND`
  - `BIGCLAW_EVENT_LOG_REPLICATION_FACTOR`

## Next backend targets

- `sqlite`: durable single-node append log with monotonic checkpoints but no replica quorum.
- `http`: shared service-backed append log with single-writer ordering and shared subscriber state.
- `broker_replicated`: quorum or partition-backed log with shared replay, replicated durability, and publisher ack requirements.

## Repo-native integration points

- `cmd/bigclawd/main.go`: bootstrap backend selection, capability validation, and future broker client wiring.
- `internal/events/bus.go`: publish path remains the place to insert append/ack behavior ahead of live fanout.
- `internal/api/server.go`: operational reporting for current and target durability mode.
- Subscriber checkpoint persistence and replay endpoints: preserve resume semantics while moving state out of process-local memory.

## Migration and compatibility constraints

- Preserve append-only replay semantics across backend cutover.
- Keep subscriber checkpoints monotonic during dual-write or backfill.
- Keep `task_id`, `trace_id`, and `event_type` stable so partitioning and replay filters remain compatible.
- Decouple SSE live fanout from broker consumer lag so replay catch-up does not stall live delivery.

## Implementation-ready follow-up plan

1. Add a concrete event-log interface alongside the in-process bus sink contract so append, replay, and checkpoint operations can be backed by SQLite, HTTP, or broker implementations.
2. Introduce a dual-write migration phase from the current publish path into the new event-log backend while keeping recorder/audit output unchanged.
3. Add checkpoint-backed replay endpoints that read from the shared event log instead of recorder-only history.
4. Add a broker-backed implementation with partition-key rules for `trace_id` and explicit publisher ack / durability error handling.
5. Validate cutover with replay, checkpoint monotonicity, SSE handoff, and capability-matrix regression coverage under shared multi-node conditions.

## Durability capability matrix

| Backend | Implemented in bootstrap | Durable history | Publish | Replay | Checkpoint | Filtering | Required config |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `memory` | yes | no | native | native | unsupported | native | none |
| `sqlite` | no | yes | native | native | native | derived | `BIGCLAW_EVENT_LOG_DSN`, `BIGCLAW_EVENT_CHECKPOINT_DSN`, `BIGCLAW_EVENT_RETENTION` |
| `http` | no | yes | native | native | native | derived | `BIGCLAW_EVENT_LOG_DSN`, `BIGCLAW_EVENT_CHECKPOINT_DSN`, `BIGCLAW_EVENT_RETENTION` |
| `broker` | no | yes | native | native | native | derived | `BIGCLAW_EVENT_LOG_DSN`, `BIGCLAW_EVENT_CHECKPOINT_DSN`, `BIGCLAW_EVENT_RETENTION` |

## Validation contract

- Startup validates `BIGCLAW_EVENT_BACKEND` against the backend catalog before queue/bootstrap wiring begins.
- Durable backends must provide explicit event-log DSN, checkpoint DSN, and positive retention.
- `BIGCLAW_EVENT_REQUIRE_REPLAY`, `BIGCLAW_EVENT_REQUIRE_CHECKPOINT`, and `BIGCLAW_EVENT_REQUIRE_FILTERING` express the runtime features operators expect from the selected backend.
- Unsupported combinations fail fast with field-specific errors instead of silently downgrading runtime behavior.
- Backends declared in the matrix but not yet wired into the bootstrap runtime are rejected explicitly so planning assumptions cannot masquerade as implemented support.

## Remaining gaps

- No concrete durable external event log exists yet in this checkout; replay still depends on process-local history plus the documented integration plan.
- Dedup ledger semantics are defined, but there is no persistent ledger store or handler middleware enforcing them yet.
- No delivery acknowledgement protocol exists beyond sink-level best effort.
- Lease coordination is currently in-memory and single-process; shared multi-node subscriber groups still need a durable backend.
- No partitioned topic model or broker-backed cross-process subscriber coordination exists yet.
- No retention watermark or expired-cursor contract exists in the runtime yet; the target compaction semantics are defined in `docs/reports/replay-retention-semantics-report.md`.
- Consumers still need their own dedupe store keyed by `delivery.idempotency_key`; this change does not introduce exactly-once execution.

## Next adapter boundary

- `internal/events/log.go` now defines the provider-neutral event-log and checkpoint contract for future broker-backed adapters.
- `internal/events/memory_log.go` provides the contract-compatible in-memory baseline while BigClaw remains on local fanout.
- Broker-facing runtime knobs are reserved behind `BIGCLAW_EVENT_LOG_*` env vars so a first provider adapter can land without changing publish/replay/checkpoint callers.
