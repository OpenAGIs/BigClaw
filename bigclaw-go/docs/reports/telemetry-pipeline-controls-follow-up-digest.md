# Telemetry Pipeline Controls Follow-up Digest

## Scope

This digest consolidates the remaining telemetry-pipeline hardening caveats for `OPE-265` / `BIG-PAR-095`.

## Current Repo-Backed Evidence

- `docs/reports/go-control-plane-observability-report.md` captures the metrics / audit surfaces already exposed by the Go control plane.
- `docs/reports/telemetry-sampling-cardinality-evidence-pack.json` makes the current sampling posture, cardinality guardrails, and reviewer links machine-checkable.
- `docs/reports/review-readiness.md` records which observability claims are already closure-safe.
- `internal/api/server.go` exposes JSON and Prometheus-style metrics surfaces.
- `internal/api/metrics.go` shows which dimensions are emitted as metrics labels versus left in debug payloads.
- `internal/observability/recorder.go` captures recorder state, counters, and audit persistence hooks.
- `internal/worker/runtime.go` contributes worker-lifecycle state that feeds the current metrics / debug snapshots.

## Reviewer Digest

- Prometheus-style exposition exists, but there is still no full OpenTelemetry-native metrics / tracing pipeline.
- Sampling policy is still implicit: the repo records full in-process event history and derives traces/metrics from that local state, but it does not define a configurable sampling contract for traces, events, or audit fan-out.
- High-cardinality controls are still lightweight and local: metrics stay on low-cardinality aggregate labels while task and trace identifiers remain in event/debug payloads, but there is no label-budget, dimension allowlist, or exporter-side aggregation policy.
- The current telemetry path is best treated as local diagnostics and rollout evidence, not a production-grade observability pipeline.

## Current Blockers

- No OpenTelemetry collector / OTLP exporter integration exists yet.
- No configurable sampling policy exists for traces or event-derived metrics.
- No explicit high-cardinality guardrails exist for task, trace, tenant, or executor dimensions.
- No remote-write or backend retention contract exists for the current metrics snapshots.

## Lightweight Consistency Check

- Keep this digest aligned with `docs/reports/go-control-plane-observability-report.md` and `docs/reports/telemetry-sampling-cardinality-evidence-pack.json`.
- Repeat the `no full OpenTelemetry-native metrics / tracing pipeline` and `no configurable sampling or high-cardinality controls` caveats anywhere telemetry readiness is summarized.
- When telemetry controls evolve, update this digest, `docs/reports/review-readiness.md`, and `docs/reports/issue-coverage.md` together.
