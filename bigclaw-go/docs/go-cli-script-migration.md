# Go CLI Script Migration

Issues: `BIG-GO-902`, `BIG-GO-1053`

## Current Go-Only Entrypoints

`bigclaw-go/scripts/e2e/` is now a Python-free operator surface. `BIG-GO-1053`
completed the tranche-2 cleanup by keeping only Go-native
`bigclawctl automation ...` subcommands plus the retained shell wrappers needed
for the bundled live-validation workflow.

## Retired Lane Coverage

The physical Python sweep for `BIG-GO-1138` found this lane already retired in
the materialized worktree: there are no remaining `*.py` files anywhere in the
repository, including the benchmark/e2e/migration candidates that previously
lived under `bigclaw-go/scripts/`.

### Benchmark candidates retired behind Go

- `bigclaw-go/scripts/benchmark/soak_local.py`
- `bigclaw-go/scripts/benchmark/run_matrix.py`
- `bigclaw-go/scripts/benchmark/capacity_certification.py`
- `bigclaw-go/scripts/benchmark/capacity_certification_test.py`

Replacement owner:

- `go run ./cmd/bigclawctl automation benchmark soak-local|run-matrix|capacity-certification ...`

### E2E candidates retired behind Go

- `bigclaw-go/scripts/e2e/run_task_smoke.py`
- `bigclaw-go/scripts/e2e/export_validation_bundle.py`
- `bigclaw-go/scripts/e2e/export_validation_bundle_test.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`
- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py`
- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py`
- `bigclaw-go/scripts/e2e/mixed_workload_matrix.py`
- `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py`
- `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`
- `bigclaw-go/scripts/e2e/external_store_validation.py`
- `bigclaw-go/scripts/e2e/multi_node_shared_queue.py`
- `bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py`
- `bigclaw-go/scripts/e2e/run_all_test.py`

Replacement owner:

- `go run ./cmd/bigclawctl automation e2e ...`
- `./scripts/e2e/run_all.sh`
- `./scripts/e2e/kubernetes_smoke.sh`
- `./scripts/e2e/ray_smoke.sh`

### Migration candidates retired behind Go

- `bigclaw-go/scripts/migration/shadow_compare.py`
- `bigclaw-go/scripts/migration/shadow_matrix.py`
- `bigclaw-go/scripts/migration/live_shadow_scorecard.py`
- `bigclaw-go/scripts/migration/export_live_shadow_bundle.py`

Replacement owner:

- `go run ./cmd/bigclawctl automation migration shadow-compare|shadow-matrix|live-shadow-scorecard|export-live-shadow-bundle ...`

| Active entrypoint | Backing command | Purpose |
| --- | --- | --- |
| `go run ./cmd/bigclawctl automation e2e run-task-smoke ...` | `bigclawctl automation e2e run-task-smoke` | Generic submit/poll smoke helper for local, Kubernetes, and Ray paths |
| `go run ./cmd/bigclawctl automation e2e export-validation-bundle ...` | `bigclawctl automation e2e export-validation-bundle` | Export timestamped validation bundles plus latest-summary/index artifacts |
| `go run ./cmd/bigclawctl automation e2e continuation-scorecard ...` | `bigclawctl automation e2e continuation-scorecard` | Regenerate the continuation scorecard from checked-in bundle evidence |
| `go run ./cmd/bigclawctl automation e2e continuation-policy-gate ...` | `bigclawctl automation e2e continuation-policy-gate` | Evaluate continuation readiness from the scorecard surface |
| `go run ./cmd/bigclawctl automation e2e broker-failover-stub-matrix ...` | `bigclawctl automation e2e broker-failover-stub-matrix` | Build the broker failover stub matrix report |
| `go run ./cmd/bigclawctl automation e2e mixed-workload-matrix ...` | `bigclawctl automation e2e mixed-workload-matrix` | Emit the mixed-workload fairness matrix |
| `go run ./cmd/bigclawctl automation e2e cross-process-coordination-surface ...` | `bigclawctl automation e2e cross-process-coordination-surface` | Render the coordination capability surface |
| `go run ./cmd/bigclawctl automation e2e subscriber-takeover-fault-matrix ...` | `bigclawctl automation e2e subscriber-takeover-fault-matrix` | Generate deterministic takeover-fault coverage |
| `go run ./cmd/bigclawctl automation e2e external-store-validation ...` | `bigclawctl automation e2e external-store-validation` | Exercise the external-store replay/takeover validation lane |
| `go run ./cmd/bigclawctl automation e2e multi-node-shared-queue ...` | `bigclawctl automation e2e multi-node-shared-queue` | Generate the live shared-queue companion proof |
| `./scripts/e2e/run_all.sh` | orchestrates Go subcommands plus smoke wrappers | Run local/Kubernetes/Ray live validation and refresh bundle artifacts |
| `./scripts/e2e/kubernetes_smoke.sh` | wraps `automation e2e run-task-smoke` | Run the Kubernetes smoke lane with repo defaults |
| `./scripts/e2e/ray_smoke.sh` | wraps `automation e2e run-task-smoke` | Run the Ray smoke lane with repo defaults |
| `go run ./cmd/bigclawctl automation benchmark soak-local|run-matrix|capacity-certification ...` | `bigclawctl automation benchmark ...` | Benchmark and capacity-certification surfaces |
| `go run ./cmd/bigclawctl automation migration shadow-compare|shadow-matrix|live-shadow-scorecard|export-live-shadow-bundle ...` | `bigclawctl automation migration ...` | Migration shadow comparison and export surfaces |

## Validation Commands

```bash
cd bigclaw-go
go test ./cmd/bigclawctl/...
go run ./cmd/bigclawctl automation --help
go run ./cmd/bigclawctl automation e2e run-task-smoke --help
go run ./cmd/bigclawctl automation e2e export-validation-bundle --help
go run ./cmd/bigclawctl automation e2e continuation-scorecard --help
go run ./cmd/bigclawctl automation e2e continuation-policy-gate --help
go run ./cmd/bigclawctl automation e2e broker-failover-stub-matrix --help
go run ./cmd/bigclawctl automation e2e mixed-workload-matrix --help
go run ./cmd/bigclawctl automation e2e cross-process-coordination-surface --help
go run ./cmd/bigclawctl automation e2e subscriber-takeover-fault-matrix --help
go run ./cmd/bigclawctl automation e2e external-store-validation --help
go run ./cmd/bigclawctl automation e2e multi-node-shared-queue --help
go run ./cmd/bigclawctl automation benchmark soak-local --help
go run ./cmd/bigclawctl automation benchmark run-matrix --help
go run ./cmd/bigclawctl automation benchmark capacity-certification --help
go run ./cmd/bigclawctl automation migration shadow-compare --help
go run ./cmd/bigclawctl automation migration shadow-matrix --help
go run ./cmd/bigclawctl automation migration live-shadow-scorecard --help
go run ./cmd/bigclawctl automation migration export-live-shadow-bundle --help
```

## Regression Surface

- CLI parsing and root help for `bigclawctl`
- HTTP polling against `/healthz`, `/tasks/:id`, and `/events`
- Temporary `bigclawd` autostart state wiring for smoke and soak commands
- Report serialization compatibility for JSON consumers that previously read the Python script output
## Compatibility Layer Plan

- Keep new behavior in Go-native entrypoints and do not reintroduce Python helpers under `bigclaw-go/scripts/e2e/`.
- Preserve the retained shell wrappers only where they add operator convenience over direct `bigclawctl automation ...` invocation.
- Continue the remaining non-e2e script migrations in follow-up batches without expanding the e2e compatibility layer again.

## Branch And PR Suggestion

- Branch: `feat/BIG-GO-902-go-cli-script-migration`
- PR title: `feat: migrate first Python automation scripts to bigclawctl`

## Risks

- `soak-local` now uses Go worker concurrency; very large counts may stress a single local HTTP backend differently than the old Python thread pool.
- `run-task-smoke --autostart` and `soak-local --autostart` still rely on ephemeral port reservation before `bigclawd` binds, so local port races remain possible.
- The shell wrappers in `scripts/e2e/` remain convenience layers; changes to flag defaults must stay aligned with the underlying Go subcommands.
