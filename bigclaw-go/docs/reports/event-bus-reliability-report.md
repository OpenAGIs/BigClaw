# Event Bus Reliability Report

## Scope

This report summarizes the current event bus reliability evidence and the next replicated-durability integration plan for `OPE-183` / `BIG-GO-008` and `OPE-206`.

## Implemented surfaces

- In-process publish/subscribe bus with replay history
- Recorder sink integration for audit/debug persistence
- Webhook sink integration for external fanout
- SSE stream via `GET /stream/events`
- Optional SSE replay and filtering via `replay=1`, `task_id`, and `trace_id`
- Replay-safe consumer delivery metadata via `EventDelivery`, including additive `delivery.mode`, `delivery.replay`, and `delivery.idempotency_key` fields
- Consumer dedup ledger/result contract covering duplicate, retryable-failure, and already-applied outcomes

## Validated behaviors

- Published events are retained in replay history.
- New subscribers can request replayed events before switching to live events.
- Webhook sink receives serialized domain events.
- SSE streaming can deliver live events.
- SSE replay can filter to one trace without leaking unrelated events.
- Replay and live deliveries preserve the original event id while exposing an explicit delivery mode and stable downstream idempotency key.

## Evidence

- `internal/events/bus.go`
- `internal/events/durability.go`
- `internal/events/bus_test.go`
- `internal/events/delivery.go`
- `internal/events/delivery_test.go`
- `internal/domain/consumer_dedup.go`
- `internal/domain/consumer_dedup_test.go`
- `internal/events/webhook.go`
- `internal/events/webhook_test.go`
- `internal/events/recorder_sink.go`
- `internal/api/server.go`
- `internal/api/server_test.go`
- `cmd/bigclawd/main.go`
- `internal/config/config.go`

## Current durability shape

- Runtime publish/subscribe remains in-process.
- Audit/debug persistence is recorder-backed, with optional JSONL sinking.
- The new `events.DurabilityPlan` surface makes the active backend and the next replicated target explicit in bootstrap and `GET /debug/status`.
- Default plan is `memory -> broker_replicated` with replication factor `3`, and env overrides now exist for:
  - `BIGCLAW_EVENT_LOG_BACKEND`
  - `BIGCLAW_EVENT_LOG_TARGET_BACKEND`
  - `BIGCLAW_EVENT_LOG_REPLICATION_FACTOR`

## Next backend targets

- `sqlite`: durable single-node append log with monotonic checkpoints but no replica quorum.
- `http`: shared service-backed append log with single-writer ordering and shared subscriber state.
- `broker_replicated`: quorum or partition-backed log with shared replay, replicated durability, and publisher ack requirements.

## Repo-native integration points

- `cmd/bigclawd/main.go`: bootstrap backend selection and future broker client wiring.
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
5. Validate cutover with replay, checkpoint monotonicity, and SSE handoff regression coverage under shared multi-node conditions.

## Remaining gaps

- No concrete durable external event log exists yet in this checkout; replay still depends on process-local history plus the newly documented integration plan.
- Dedup ledger semantics are defined, but there is no persistent ledger store or handler middleware enforcing them yet.
- No delivery acknowledgement protocol exists beyond sink-level best effort.
- No partitioned topic model or broker-backed cross-process subscriber coordination exists yet.
- Consumers still need their own dedupe store keyed by `delivery.idempotency_key`; this change does not introduce exactly-once execution.
