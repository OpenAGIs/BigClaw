# Event Bus Reliability Report

## Scope

This report summarizes the current event bus reliability evidence for `OPE-183` / `BIG-GO-008`.

## Implemented surfaces

- In-process publish/subscribe bus with replay history
- Recorder sink integration for audit/debug persistence
- Webhook sink integration for external fanout
- SSE stream via `GET /stream/events`
- Optional SSE replay and filtering via `replay=1`, `task_id`, and `trace_id`
- Replay-safe consumer dedup ledger contract with stable storage key and result semantics

## Validated behaviors

- Published events are retained in replay history.
- New subscribers can request replayed events before switching to live events.
- Webhook sink receives serialized domain events.
- SSE streaming can deliver live events.
- SSE replay can filter to one trace without leaking unrelated events.

## Evidence

- `internal/events/bus.go`
- `internal/events/dedup_ledger.go`
- `internal/events/dedup_ledger_test.go`
- `internal/events/bus_test.go`
- `internal/events/webhook.go`
- `internal/events/webhook_test.go`
- `internal/events/recorder_sink.go`
- `internal/domain/dedup.go`
- `internal/domain/dedup_test.go`
- `internal/api/server.go`
- `internal/api/server_test.go`

## Remaining gaps

- No durable external event log yet; replay is process-local history.
- No durable consumer dedup backend yet; only the backend abstraction, stable key layout, and reference in-memory semantics are implemented.
- No delivery acknowledgement protocol beyond sink-level best effort.
- No partitioned topic model or cross-process subscriber coordination yet.

## Consumer dedup ledger contract

- Stable storage keys use `v1/<consumer_id>/<event_id>` so durable backends can share one persistence layout while still rejecting conflicting event metadata for the same consumer/event tuple.
- Collision protection uses a fingerprint over `consumer_id`, `event_id`, `event_type`, `task_id`, `trace_id`, and `run_id`; the same storage key cannot be reused for a different event payload shape.
- Reservation semantics distinguish first-writer `reserved`, repeated in-flight `duplicate`, and terminal `already_applied` outcomes.
- Applied side effects persist `handler`, `applied_at`, `effect_id`, `effect_sequence`, `effect_fingerprint`, `summary`, and stable metadata so duplicate deliveries can return the prior applied result instead of replaying the side effect.
- Once a record reaches `applied`, backends must treat a different result fingerprint as a conflict instead of silently overwriting the prior side effect evidence.
