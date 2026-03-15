# Event Bus Reliability Report

## Scope

This report summarizes the current event bus reliability evidence for `OPE-183` / `BIG-GO-008`.

## Implemented surfaces

- In-process publish/subscribe bus with replay history
- Recorder sink integration for audit/debug persistence
- Webhook sink integration for external fanout
- SSE stream via `GET /stream/events`
- Optional SSE replay and filtering via `replay=1`, `task_id`, and `trace_id`
- Subscriber-group checkpoint lease coordination via `/subscriber-groups/leases` and `/subscriber-groups/checkpoints`

## Validated behaviors

- Published events are retained in replay history.
- New subscribers can request replayed events before switching to live events.
- Webhook sink receives serialized domain events.
- SSE streaming can deliver live events.
- SSE replay can filter to one trace without leaking unrelated events.
- Subscriber-group checkpoint commits are fenced by lease token + epoch, so stale writers cannot advance ownership after takeover.
- Checkpoint offsets remain monotonic within a subscriber group and reject rollback writes.

## Evidence

- `internal/events/bus.go`
- `internal/events/bus_test.go`
- `internal/events/webhook.go`
- `internal/events/webhook_test.go`
- `internal/events/recorder_sink.go`
- `internal/events/subscriber_leases.go`
- `internal/events/subscriber_leases_test.go`
- `internal/api/server.go`
- `internal/api/server_test.go`

## Remaining gaps

- No durable external event log yet; replay is process-local history.
- No delivery acknowledgement protocol beyond sink-level best effort.
- Lease coordination is currently in-memory and single-process; shared multi-node subscriber groups still need a durable backend.
- No partitioned topic model yet.
