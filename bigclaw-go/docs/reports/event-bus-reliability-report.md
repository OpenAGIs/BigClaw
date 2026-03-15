# Event Bus Reliability Report

## Scope

This report summarizes the current event bus reliability evidence for `OPE-183` / `BIG-GO-008`.

## Implemented surfaces

- In-process publish/subscribe bus with replay history
- Recorder sink integration for audit/debug persistence
- Webhook sink integration for external fanout
- SSE stream via `GET /stream/events`
- Optional SSE replay and filtering via `replay=1`, `task_id`, and `trace_id`
- Additive delivery metadata for replay-safe downstream consumption via `delivery.mode`, `delivery.replay`, and `delivery.idempotency_key`

## Validated behaviors

- Published events are retained in replay history.
- New subscribers can request replayed events before switching to live events.
- Webhook sink receives serialized domain events.
- SSE streaming can deliver live events.
- SSE replay can filter to one trace without leaking unrelated events.
- Replay and live deliveries preserve the original event id while exposing an explicit delivery mode and stable downstream idempotency key.

## Evidence

- `internal/events/bus.go`
- `internal/events/bus_test.go`
- `internal/events/webhook.go`
- `internal/events/webhook_test.go`
- `internal/events/recorder_sink.go`
- `internal/api/server.go`
- `internal/api/server_test.go`

## Remaining gaps

- No durable external event log yet; replay is process-local history.
- No delivery acknowledgement protocol beyond sink-level best effort.
- No partitioned topic model or cross-process subscriber coordination yet.
- Consumers still need their own dedupe store keyed by `delivery.idempotency_key`; this change does not introduce exactly-once execution.
