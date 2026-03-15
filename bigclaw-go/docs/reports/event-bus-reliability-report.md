# Event Bus Reliability Report

## Scope

This report summarizes the current event bus reliability evidence for `OPE-183` / `BIG-GO-008`.

## Implemented surfaces

- In-process publish/subscribe bus with replay history
- Optional SQLite-backed durable event log for cross-process replay
- Optional HTTP-backed remote event log service/client for cross-node event durability and replay coordination
- Durable subscriber checkpoints for acknowledged consumer resume positions
- Recorder sink integration for audit/debug persistence
- Webhook sink integration for external fanout
- SSE stream via `GET /stream/events`
- Optional SSE replay and filtering via `replay=1`, `after_id`, `Last-Event-ID`, `subscriber_id`, `task_id`, `trace_id`, and `event_type`
- Topic-scoped in-process subscriptions via task / trace / event-type filters

## Validated behaviors

- Published events are retained in replay history.
- New subscribers can request replayed events before switching to live events.
- Webhook sink receives serialized domain events.
- SSE streaming can deliver live events.
- SSE replay can filter to one trace without leaking unrelated events.
- Durable event log replay can serve task/trace history across process restarts when the shared SQLite log is configured.
- Cursor-based replay can resume `/events` and SSE consumers from a prior event id without replaying the full stream.
- SSE reconnects can recover missed trace/task events by honoring `Last-Event-ID` against the durable event log.
- SSE now subscribes before replay and deduplicates overlap so replay/live handoff stays gap-free across reconnect catch-up.
- Subscriber checkpoints can be acknowledged explicitly and reused across process restarts to resume `/events` and SSE streams from shared durable state.
- Remote event-log clients can publish, replay, and checkpoint through a shared HTTP service instead of depending on a shared local SQLite path on every node.
- Checkpoint acknowledgements are monotonic by event sequence so stale or duplicate acknowledgements cannot move consumer progress backwards.
- Topic-scoped subscriptions and `event_type` filters prevent unrelated events from being replayed or fanned out to filtered consumers.

## Evidence

- `internal/events/bus.go`
- `internal/events/bus_test.go`
- `internal/events/webhook.go`
- `internal/events/webhook_test.go`
- `internal/events/recorder_sink.go`
- `internal/events/sqlite_log.go`
- `internal/events/sqlite_log_test.go`
- `internal/api/server.go`
- `internal/api/server_test.go`

## Remaining gaps

- Remote coordination is now service-style, but durability still ultimately depends on a single SQLite-backed log rather than a replicated broker or quorum-backed event store.
- Monotonic checkpoints prevent regressions, but downstream consumers still need idempotent handlers because the system remains replay-capable rather than globally exactly-once.
