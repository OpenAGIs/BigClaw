# BigClaw Go MVP Issue Coverage

## Summary

This document maps the current local MVP implementation to the Linear rewrite issues `OPE-176` through `OPE-186`.

## Coverage

- `OPE-176` / `BIG-GO-001`
  - Covered by `docs/adr/0001-go-rewrite-control-plane.md` and `docs/reports/migration-plan-review-notes.md`
- `OPE-177` / `BIG-GO-002`
  - Covered by `internal/domain/task.go`, `internal/domain/state_machine.go`, `docs/reports/task-protocol-spec.md`, and `docs/reports/state-machine-validation-report.md`
- `OPE-178` / `BIG-GO-003`
  - Covered by `internal/queue/queue.go`, `internal/queue/memory_queue.go`, `internal/queue/file_queue.go`, `internal/queue/sqlite_queue.go`, `internal/queue/sqlite_queue_test.go`, `internal/api/server.go`, `internal/api/server_test.go`, `docs/reports/queue-reliability-report.md`, and `docs/reports/lease-recovery-report.md`
- `OPE-179` / `BIG-GO-004`
  - Covered by `internal/scheduler/scheduler.go`, `internal/scheduler/scheduler_test.go`, `internal/orchestrator/loop.go`, and `docs/reports/scheduler-policy-report.md`
- `OPE-180` / `BIG-GO-005`
  - Covered by `internal/worker/runtime.go`, `internal/worker/runtime_test.go`, `internal/api/server.go`, `internal/api/server_test.go`, and `docs/reports/worker-lifecycle-validation-report.md`
- `OPE-181` / `BIG-GO-006`
  - Covered by `internal/executor/kubernetes.go`, `internal/executor/kubernetes_test.go`, `scripts/e2e/kubernetes_smoke.sh`, `docs/e2e-validation.md`, and `docs/reports/kubernetes-live-smoke-report.json`
- `OPE-182` / `BIG-GO-007`
  - Covered by `internal/executor/ray.go`, `internal/executor/ray_test.go`, `scripts/e2e/ray_smoke.sh`, `docs/e2e-validation.md`, and `docs/reports/ray-live-smoke-report.json`
- `OPE-183` / `BIG-GO-008`
  - Covered by `internal/events/bus.go`, `internal/events/recorder_sink.go`, `internal/events/webhook.go`, `internal/events/bus_test.go`, `internal/events/webhook_test.go`, `internal/api/server.go`, `internal/api/server_test.go`, and `docs/reports/event-bus-reliability-report.md`
  - Follow-up takeover caveats are tracked in `docs/reports/subscriber-takeover-executability-follow-up-digest.md`, which now captures the deterministic local harness and the remaining live multi-node gap
- `OPE-184` / `BIG-GO-009`
  - Covered by `internal/observability/recorder.go`, `internal/observability/recorder_test.go`, `internal/observability/audit.go`, `internal/observability/audit_test.go`, `internal/api/server.go`, `internal/api/server_test.go`, `internal/worker/runtime.go`, `docs/reports/go-control-plane-observability-report.md`, `scripts/e2e/run_task_smoke.py`, `scripts/e2e/run_all.sh`, and isolated autostart live-validation reports in `docs/reports/*.json`
  - Follow-up caveats for external tracing backends, span propagation, telemetry pipeline controls, sampling policy, and high-cardinality handling are tracked in `docs/reports/tracing-backend-follow-up-digest.md`, `docs/reports/telemetry-pipeline-controls-follow-up-digest.md`, and `docs/reports/telemetry-sampling-cardinality-evidence-pack.json`
- `OPE-185` / `BIG-GO-010`
  - Covered by `docs/migration.md`, `docs/migration-shadow.md`, `scripts/migration/shadow_compare.py`, `scripts/migration/shadow_matrix.py`, `scripts/migration/live_shadow_scorecard.py`, `examples/shadow-corpus-manifest.json`, `docs/reports/migration-readiness-report.md`, `docs/reports/shadow-compare-report.json`, `docs/reports/shadow-matrix-report.json`, and `docs/reports/live-shadow-mirror-scorecard.json`
  - Follow-up caveats for live shadow traffic comparison, rollback safeguard trigger surfaces, and production corpus coverage are tracked in `docs/reports/live-shadow-comparison-follow-up-digest.md`, `docs/reports/rollback-safeguard-follow-up-digest.md`, and `docs/reports/production-corpus-migration-coverage-digest.md`
- `OPE-186` / `BIG-GO-011`
  - Covered by `internal/queue/benchmark_test.go`, `internal/scheduler/benchmark_test.go`, `docs/benchmark-plan.md`, `docs/reports/benchmark-report.md`, `docs/reports/benchmark-readiness-report.md`, `docs/reports/benchmark-matrix-report.json`, `docs/reports/long-duration-soak-report.md`, `docs/reports/soak-local-report.json`, `docs/reports/soak-local-50x8.json`, `docs/reports/soak-local-100x12.json`, `docs/reports/soak-local-1000x24.json`, `docs/reports/soak-local-2000x24.json`, `docs/reports/live-validation-summary.json`, `scripts/benchmark/run_suite.sh`, `scripts/benchmark/run_matrix.py`, `scripts/benchmark/soak_local.py`, and `scripts/e2e/run_all.sh`

## Parallel follow-up digests

- `OPE-264` / `BIG-PAR-075` — tracing backend and span-propagation caveats are consolidated in `docs/reports/tracing-backend-follow-up-digest.md`.
- `OPE-265` / `BIG-PAR-076` — telemetry pipeline, sampling policy, and high-cardinality caveats are consolidated in `docs/reports/telemetry-pipeline-controls-follow-up-digest.md`, with the current machine-checkable surface in `docs/reports/telemetry-sampling-cardinality-evidence-pack.json`.
- `OPE-266` / `BIG-PAR-092` — repo-native live shadow mirror scorecard and remaining live shadow traffic comparison caveats are consolidated in `docs/reports/live-shadow-comparison-follow-up-digest.md`.
- `OPE-267` / `BIG-PAR-078` — rollback safeguard trigger-surface caveats are consolidated in `docs/reports/rollback-safeguard-follow-up-digest.md`.
- `OPE-268` / `BIG-PAR-079` — production corpus coverage caveats are consolidated in `docs/reports/production-corpus-migration-coverage-digest.md`.
- `OPE-269` / `BIG-PAR-080` — subscriber takeover executability caveats are consolidated in `docs/reports/subscriber-takeover-executability-follow-up-digest.md`.
- `OPE-270` / `BIG-PAR-081` — cross-process coordination caveats are consolidated in `docs/reports/cross-process-coordination-boundary-digest.md`, with the current machine-readable surface in `docs/reports/cross-process-coordination-capability-surface.json`.
- `OPE-271` / `BIG-PAR-082` — validation bundle continuation caveats are consolidated in `docs/reports/validation-bundle-continuation-digest.md`, with the current multi-bundle lineage summarized in `docs/reports/validation-bundle-continuation-scorecard.json` and the current gate result captured in `docs/reports/validation-bundle-continuation-policy-gate.json`.

## Remaining Gaps Before Honest Closure

- Real `Kubernetes` API integration path is implemented and has passed live smoke validation against `kind-ray-local` using `KUBECONFIG=/Users/jxrt/.kube/ray-local-config`.
- Real `Ray Jobs` REST integration path is implemented and has passed live smoke validation against `ray://127.0.0.1:10001` via the live dashboard Jobs API on `127.0.0.1:8265`.
- SQLite-backed durable queue support is implemented; higher-scale external store validation is still pending.
- No dedicated leader-election layer exists yet; current evidence is limited to a local two-node shared-SQLite coordination proof captured in `docs/reports/multi-node-coordination-report.md` plus the repo-native surface summary in `docs/reports/cross-process-coordination-capability-surface.json`.
- Validation bundle continuation now has a repo-native rolling scorecard in `docs/reports/validation-bundle-continuation-scorecard.json` plus a checked-in policy gate in `docs/reports/validation-bundle-continuation-policy-gate.json`; `run_all.sh` refreshes both automatically during closeout, but the surface remains workflow-triggered and non-enforcing by default.
- Multi-subscriber takeover validation now has a deterministic local harness in `docs/reports/multi-subscriber-takeover-validation-report.md`, but a live multi-node subscriber takeover proof is still pending.
- Benchmark output is local bootstrap evidence, not production-grade capacity certification.
- When running multiple local smoke processes with the SQLite backend, use separate `BIGCLAW_QUEUE_SQLITE_PATH` and `BIGCLAW_AUDIT_LOG_PATH` values to avoid local file-lock contention.
- Replay retention, compaction, and aged-out checkpoint semantics for the follow-on parallel durability track are documented in `docs/reports/replay-retention-semantics-report.md` and `docs/openclaw-parallel-gap-analysis.md`.
