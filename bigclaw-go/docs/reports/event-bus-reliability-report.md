# Event Bus Reliability Report

## Scope

This report summarizes the current event bus reliability evidence for `OPE-183` / `BIG-GO-008`.

## Implemented surfaces

- In-process publish/subscribe bus with replay history
- Recorder sink integration for audit/debug persistence
- Webhook sink integration for external fanout
- SSE stream via `GET /stream/events`
- Optional SSE replay and filtering via `replay=1`, `task_id`, and `trace_id`
- Event backend capability and config-validation contract via `internal/events/backend_contract.go`

## Validated behaviors

- Published events are retained in replay history.
- New subscribers can request replayed events before switching to live events.
- Webhook sink receives serialized domain events.
- SSE streaming can deliver live events.
- SSE replay can filter to one trace without leaking unrelated events.

## Evidence

- `internal/events/bus.go`
- `internal/events/backend_contract.go`
- `internal/events/backend_contract_test.go`
- `internal/events/bus_test.go`
- `internal/events/webhook.go`
- `internal/events/webhook_test.go`
- `internal/events/recorder_sink.go`
- `internal/api/server.go`
- `internal/api/server_test.go`
- `cmd/bigclawd/main.go`
- `internal/config/config.go`

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

- No durable external event log is wired into the runtime yet; replay is still process-local history.
- No delivery acknowledgement protocol beyond sink-level best effort.
- No partitioned topic model or cross-process subscriber coordination yet.
