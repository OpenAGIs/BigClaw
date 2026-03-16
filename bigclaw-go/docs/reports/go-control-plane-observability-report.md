# Go Control Plane Observability Report

## Scope

This report summarizes the current observability/debug evidence for `OPE-184` / `BIG-GO-009`.

## Implemented surfaces

- Event counters and queue-size metrics via `GET /metrics` JSON plus Prometheus-style text exposition via `GET /metrics?format=prometheus`
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
- Metrics surface keeps `trace_count` JSON visibility and now exposes scrape-friendly queue, event, executor, worker-pool, and control-plane gauges.
- Audit sink writes JSONL event records for later inspection.

## Evidence

- `internal/observability/recorder.go`
- `internal/observability/recorder_test.go`
- `internal/observability/audit.go`
- `internal/observability/audit_test.go`
- `internal/api/server.go`
- `internal/api/metrics.go`
- `internal/api/server_test.go`
- `internal/worker/runtime.go`
- `docs/reports/telemetry-sampling-cardinality-evidence-pack.json`

## Remaining gaps

- No external tracing backend or span propagation beyond in-memory `trace_id` grouping; see `docs/reports/tracing-backend-follow-up-digest.md`.
- Prometheus-style text exposition is now available, but there is still no full OpenTelemetry-native metrics / tracing pipeline; see `docs/reports/telemetry-pipeline-controls-follow-up-digest.md`.
- No configurable sampling policy or production-grade high-cardinality controls exist beyond lightweight in-memory usage; the current repo-native evidence pack is `docs/reports/telemetry-sampling-cardinality-evidence-pack.json` and the follow-up digest remains `docs/reports/telemetry-pipeline-controls-follow-up-digest.md`.

## Parallel follow-up digests

- `OPE-264` / `BIG-PAR-075` — `docs/reports/tracing-backend-follow-up-digest.md`
- `OPE-265` / `BIG-PAR-076` — `docs/reports/telemetry-pipeline-controls-follow-up-digest.md` and `docs/reports/telemetry-sampling-cardinality-evidence-pack.json`
