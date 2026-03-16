# Telemetry Pipeline Controls Follow-up Digest

## Scope

This digest consolidates the remaining telemetry-pipeline hardening caveats for `OPE-265` / `BIG-PAR-076`.

## Current Repo-Backed Evidence

- `docs/reports/go-control-plane-observability-report.md` captures the metrics / audit surfaces already exposed by the Go control plane.
- `docs/reports/review-readiness.md` records which observability claims are already closure-safe.
- `internal/api/server.go` exposes JSON and Prometheus-style metrics surfaces.
- `internal/observability/recorder.go` captures recorder state, counters, and audit persistence hooks.
- `internal/worker/runtime.go` contributes worker-lifecycle state that feeds the current metrics / debug snapshots.

## Reviewer Digest

- Prometheus-style exposition exists, but there is still no full OpenTelemetry-native metrics / tracing pipeline.
- Sampling policy is still implicit: the repo does not define a configurable sampling contract for traces, events, or audit fan-out.
- High-cardinality controls are still lightweight and local; there is no label-budget, dimension allowlist, or exporter-side aggregation policy.
- The current telemetry path is best treated as local diagnostics and rollout evidence, not a production-grade observability pipeline.

## Current Blockers

- No OpenTelemetry collector / OTLP exporter integration exists yet.
- No configurable sampling policy exists for traces or event-derived metrics.
- No explicit high-cardinality guardrails exist for task, trace, tenant, or executor dimensions.
- No remote-write or backend retention contract exists for the current metrics snapshots.

## Lightweight Consistency Check

- Keep this digest aligned with `docs/reports/go-control-plane-observability-report.md`.
- Repeat the `no full OpenTelemetry-native metrics / tracing pipeline` and `no configurable sampling or high-cardinality controls` caveats anywhere telemetry readiness is summarized.
- When telemetry controls evolve, update this digest, `docs/reports/review-readiness.md`, and `docs/reports/issue-coverage.md` together.
