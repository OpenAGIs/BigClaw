# Go Control Plane Observability Report

## Scope

This report summarizes the current observability/debug evidence for `OPE-184` / `BIG-GO-009`.

## Implemented surfaces

- Event counters and queue-size metrics via `GET /metrics`
- Task timeline lookup via `GET /events?task_id=...`
- Trace timeline lookup via `GET /events?trace_id=...`
- Trace summary listing via `GET /debug/traces`
- Trace detail timeline via `GET /debug/traces/{trace_id}`
- Worker lifecycle snapshot via `GET /debug/status`
- Audit persistence via `internal/observability/JSONLAuditSink`

## Validated behaviors

- Task and trace timelines are queryable from the recorder.
- Recent traces can be summarized with first/last timestamps, duration, event counts, and task ids.
- Debug status exposes the current worker snapshot and aggregate counters.
- Metrics surface includes `trace_count` in addition to queue/event counters.
- Audit sink writes JSONL event records for later inspection.

## Evidence

- `internal/observability/recorder.go`
- `internal/observability/recorder_test.go`
- `internal/observability/audit.go`
- `internal/observability/audit_test.go`
- `internal/api/server.go`
- `internal/api/server_test.go`
- `internal/worker/runtime.go`

## Remaining gaps

- No external tracing backend or span propagation beyond in-memory `trace_id` grouping.
- Metrics remain JSON debug output rather than Prometheus/OpenTelemetry-native exposition.
- No sampling policy or high-cardinality controls beyond lightweight in-memory usage.
