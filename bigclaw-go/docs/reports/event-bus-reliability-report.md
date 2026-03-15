# Event Bus Reliability Report

## Scope

This report summarizes the current event bus reliability evidence for `OPE-183` / `BIG-GO-008`.

## Implemented surfaces

- In-process publish/subscribe bus with replay history
- Recorder sink integration for audit/debug persistence
- Webhook sink integration for external fanout
- SSE stream via `GET /stream/events`
- Optional SSE replay and filtering via `replay=1`, `after_id`, `Last-Event-ID`, `task_id`, and `trace_id`
- Replay cursor diagnostics via `X-Replay-*` headers and JSON `cursor` metadata on `GET /events`

## Validated behaviors

- Published events are retained in replay history.
- New subscribers can request replayed events before switching to live events.
- Replay resumes can detect when the available replay window has moved past the requested cursor and fall back to the oldest retained event.
- Webhook sink receives serialized domain events.
- SSE streaming can deliver live events.
- SSE replay can filter to one trace without leaking unrelated events.

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
- Expired cursor detection is bounded by the current in-process replay window; unknown cursors and fully empty windows still need durable backend semantics for stronger guarantees.
- No delivery acknowledgement protocol beyond sink-level best effort.
- No partitioned topic model or cross-process subscriber coordination yet.
