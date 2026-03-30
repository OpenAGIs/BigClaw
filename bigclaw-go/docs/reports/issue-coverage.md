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
  - Follow-up takeover caveats are tracked in `docs/reports/subscriber-takeover-executability-follow-up-digest.md`, which now captures the deterministic local harness, the live two-node proof, and the remaining `OPE-269` / `BIG-PAR-080` shared-durable ownership gap
- `OPE-184` / `BIG-GO-009`
  - Covered by `internal/observability/recorder.go`, `internal/observability/recorder_test.go`, `internal/observability/audit.go`, `internal/observability/audit_test.go`, `internal/api/server.go`, `internal/api/server_test.go`, `internal/worker/runtime.go`, `docs/reports/go-control-plane-observability-report.md`, `cmd/bigclawctl/automation_commands.go`, `scripts/e2e/run_all.sh`, and isolated autostart live-validation reports in `docs/reports/*.json`
  - Follow-up caveats for `OPE-264` / `BIG-PAR-075` external tracing backends and span propagation plus `OPE-265` / `BIG-PAR-076` telemetry pipeline controls, sampling policy, and high-cardinality handling are tracked in `docs/reports/tracing-backend-follow-up-digest.md`, `docs/reports/telemetry-pipeline-controls-follow-up-digest.md`, and `docs/reports/telemetry-sampling-cardinality-evidence-pack.json`
- `OPE-185` / `BIG-GO-010`
  - Covered by `docs/migration.md`, `docs/migration-shadow.md`, `scripts/migration/shadow_compare.py`, `scripts/migration/shadow_matrix.py`, `scripts/migration/live_shadow_scorecard.py`, `examples/shadow-corpus-manifest.json`, `docs/reports/migration-readiness-report.md`, `docs/reports/shadow-compare-report.json`, `docs/reports/shadow-matrix-report.json`, and `docs/reports/live-shadow-mirror-scorecard.json`
  - Follow-up caveats for `OPE-266` / `BIG-PAR-092` live shadow traffic comparison, `OPE-254` / `BIG-PAR-088` rollback safeguard trigger surfaces, and `OPE-268` / `BIG-PAR-079` production corpus coverage are tracked in `docs/reports/live-shadow-comparison-follow-up-digest.md`, `docs/reports/rollback-safeguard-follow-up-digest.md`, `docs/reports/rollback-trigger-surface.json`, and `docs/reports/production-corpus-migration-coverage-digest.md`
- `OPE-186` / `BIG-GO-011`
  - Covered by `internal/queue/benchmark_test.go`, `internal/scheduler/benchmark_test.go`, `docs/benchmark-plan.md`, `docs/reports/benchmark-report.md`, `docs/reports/benchmark-readiness-report.md`, `docs/reports/benchmark-matrix-report.json`, `docs/reports/long-duration-soak-report.md`, `docs/reports/soak-local-report.json`, `docs/reports/soak-local-50x8.json`, `docs/reports/soak-local-100x12.json`, `docs/reports/soak-local-1000x24.json`, `docs/reports/soak-local-2000x24.json`, `docs/reports/live-validation-summary.json`, `scripts/benchmark/run_suite.sh`, `cmd/bigclawctl/automation_benchmark_reports.go`, `cmd/bigclawctl/automation_commands.go`, and `scripts/e2e/run_all.sh`
- `OPE-3` / `BIG-PAR-098`
  - Covered by `docs/benchmark-plan.md`, `docs/reports/capacity-certification-report.md`, `docs/reports/capacity-certification-matrix.json`, `docs/reports/benchmark-readiness-report.md`, `docs/reports/long-duration-soak-report.md`, `docs/reports/mixed-workload-matrix-report.json`, `cmd/bigclawctl/automation_benchmark_reports.go`, and `cmd/bigclawctl/automation_benchmark_reports_test.go`

## Parallel Follow-up Index

- `docs/reports/parallel-follow-up-index.md` is the canonical index for the
  remaining `BIG-PAR-*` follow-up digests, capability surfaces, and rollout
  contracts.
- Use `docs/reports/parallel-validation-matrix.md` first for executor-lane
  validation commands and checked-in local/Kubernetes/Ray evidence, then use
  the follow-up index for the unfinished hardening tracks behind those lanes.
- Coordination and continuation follow-ups referenced across the evidence set
  include `docs/reports/cross-process-coordination-boundary-digest.md` and
  `docs/reports/validation-bundle-continuation-digest.md`.
- The current bundle-continuation lane is tracked under `OPE-271` / `BIG-PAR-082` in `docs/reports/validation-bundle-continuation-digest.md`.
- The runtime capability matrix in
  `docs/reports/cross-process-coordination-capability-surface.json` makes the
  remaining coordination posture explicit as `live_proven`, `harness_proven`, and `contract_only`.

## Remaining Gaps Before Honest Closure

- Real `Kubernetes` API integration path is implemented and has passed live smoke validation against `kind-ray-local` using `KUBECONFIG=/Users/jxrt/.kube/ray-local-config`.
- Real `Ray Jobs` REST integration path is implemented and has passed live smoke validation against `ray://127.0.0.1:10001` via the live dashboard Jobs API on `127.0.0.1:8265`.
- SQLite-backed durable queue support is implemented, and a repo-native external-store validation lane now exists in `docs/reports/external-store-validation-report.json`, proving replay, checkpoint reset history, persisted retention boundaries, and shared-lease takeover behavior through the remote HTTP event-log service boundary. Its backend matrix now marks the remaining distributed durability lanes explicitly as `broker_replicated=not_configured` and `quorum_replicated=contract_only` beyond that first external-store lane.
- A repo-native leader-election scaffold now exists through the subscriber-lease-backed `/coordination/leader` surface and the matching debug/control-center payloads, while the current runtime proof remains a `live_proven` local two-node shared-SQLite coordination result captured in `docs/reports/multi-node-coordination-report.md`.
- `docs/reports/leader-election-capability-surface.json` now separates the backend posture directly: shared SQLite is `live_proven`, shared-store takeover hardening is `harness_proven`, and broker/quorum ownership remains `contract_only`.
- Takeover and replay semantics remain `harness_proven` and broker-backed ownership remains `contract_only` in `docs/reports/cross-process-coordination-capability-surface.json`.
- Validation bundle continuation now has a repo-native rolling scorecard in `docs/reports/validation-bundle-continuation-scorecard.json` plus a checked-in policy gate in `docs/reports/validation-bundle-continuation-policy-gate.json`; `run_all.sh` refreshes both automatically during closeout, but the surface remains workflow-triggered and non-enforcing by default.
- Multi-subscriber takeover validation now has both a deterministic local harness in `docs/reports/multi-subscriber-takeover-validation-report.md` and a live two-node companion proof in `docs/reports/live-multi-node-subscriber-takeover-report.json`, but shared-durable subscriber ownership is still pending.
- Benchmark output now separates local bootstrap evidence from the repo-native capacity certification matrix; both remain reviewer artifacts rather than runtime-enforced policy.
- When running multiple local smoke processes with the SQLite backend, use separate `BIGCLAW_QUEUE_SQLITE_PATH` and `BIGCLAW_AUDIT_LOG_PATH` values to avoid local file-lock contention.
- Replay retention, compaction, and aged-out checkpoint semantics for the follow-on parallel durability track are documented in `docs/reports/replay-retention-semantics-report.md`, `docs/reports/replicated-event-log-durability-rollout-contract.md`, and `docs/reports/replicated-broker-durability-rollout-spike.md`.
