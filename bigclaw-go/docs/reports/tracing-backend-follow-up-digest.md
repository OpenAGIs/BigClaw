# Tracing Backend Follow-up Digest

## Scope

This digest consolidates the remaining external tracing-backend and span-propagation caveats for `OPE-264` / `BIG-PAR-075`.

## Current Repo-Backed Evidence

- `docs/reports/go-control-plane-observability-report.md` captures the currently shipped debug, metrics, and audit surfaces.
- `docs/reports/review-readiness.md` records which observability claims are already safe to treat as closure-ready.
- `docs/reports/issue-coverage.md` maps the shipped observability implementation back to the original rewrite issues.
- `internal/observability/recorder.go` preserves in-memory task / trace grouping and audit persistence hooks.
- `internal/api/server.go` exposes the operator-facing `/events`, `/debug/traces`, `/debug/status`, and `/metrics` surfaces.

## Reviewer Digest

- Trace visibility is still repo-local: `trace_id` groups timelines inside the recorder and debug APIs, but there is no external tracing backend.
- Span propagation stops at request / task correlation: there is no cross-process span propagation beyond in-memory `trace_id` grouping.
- Debug surfaces are operator-readable snapshots rather than a distributed trace search plane.
- Audit persistence is file-backed JSONL evidence, not a multi-tenant tracing store with indexed query, retention, or fan-out controls.

## Current Blockers

- No OTLP / Jaeger / Tempo / Zipkin exporter path exists yet.
- No background batching or retry contract exists for trace envelopes.
- No shared-queue, Kubernetes, or Ray span propagation contract exists yet.
- No backend-side sampling or span-budget policy exists beyond lightweight in-memory usage.

## Lightweight Consistency Check

- Keep this digest aligned with `docs/reports/go-control-plane-observability-report.md`.
- Repeat the `no external tracing backend` and `no cross-process span propagation beyond in-memory trace grouping` caveats anywhere review-ready observability claims are summarized.
- When tracing backend support lands, update this digest, `docs/reports/review-readiness.md`, and `docs/reports/issue-coverage.md` together.
