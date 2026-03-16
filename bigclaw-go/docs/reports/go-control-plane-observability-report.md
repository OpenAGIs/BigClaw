# Go Control Plane Observability Report

## Scope

This report summarizes the current observability/debug evidence for `OPE-184` / `BIG-GO-009`.

## Implemented surfaces

- Metrics snapshot via `GET /metrics` with concrete JSON fields for `queue_size`, `events`, `trace_count`, `registered_executors`, `event_durability`, `event_log`, and `retention_watermark`
- Prometheus-style text exposition via `GET /metrics?format=prometheus` for queue, trace, event-type, registered-executor, worker-pool, per-worker, and control-plane gauges/counters
- Task timeline lookup via `GET /events?task_id=...`
- Trace timeline lookup via `GET /events?trace_id=...`
- Trace summary listing via `GET /debug/traces`
- Trace detail timeline via `GET /debug/traces/{trace_id}`
- Worker and control-plane snapshot via `GET /debug/status`
- Replay/checkpoint diagnostics via `/events`, `/stream/events`, `/stream/events/checkpoints/{subscriber_id}`, and `/debug/status` retention/reset summaries
- Audit persistence via `internal/observability/JSONLAuditSink`

## Validated behaviors

- Task and trace timelines are queryable from the recorder.
- Recent traces can be summarized with first/last timestamps, duration, event counts, and task ids.
- Debug status exposes worker snapshots, worker-pool totals, event-log capability data, retention watermark details, and recent checkpoint reset summaries.
- Metrics surface keeps `trace_count` JSON visibility and now exposes concrete JSON fields plus Prometheus gauges/counters for queue size, recorded event types, registered executors, worker-pool totals, per-worker activity, and control-plane pause/takeover state.
- Audit sink writes JSONL event records for later inspection.

## Evidence

- `internal/observability/recorder.go`
- `internal/observability/recorder_test.go`
- `internal/observability/audit.go`
- `internal/observability/audit_test.go`
- `internal/api/metrics.go`
- `internal/api/server.go`
- `internal/api/server_test.go`
- `internal/worker/runtime.go`

## Remaining gaps

- No external tracing backend or span propagation beyond in-memory `trace_id` grouping.
- Prometheus-style text exposition is now available, but there is still no full OpenTelemetry-native metrics/tracing pipeline.
- No sampling policy or high-cardinality controls beyond lightweight in-memory usage.
