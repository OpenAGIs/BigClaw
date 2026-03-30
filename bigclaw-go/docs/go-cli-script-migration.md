# Go CLI Script Migration

Issue: `BIG-GO-902`

## Implemented In This Slice

| Legacy script | Go CLI replacement | Status |
| --- | --- | --- |
| `bigclaw-go/scripts/e2e/run_task_smoke.py` | `go run ./cmd/bigclawctl automation e2e run-task-smoke ...` | migrated and Python shim removed |
| `bigclaw-go/scripts/e2e/export_validation_bundle.py` | `go run ./cmd/bigclawctl automation e2e export-validation-bundle ...` | migrated and Python shim removed |
| `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py` | `go run ./cmd/bigclawctl automation e2e continuation-scorecard ...` | migrated and Python shim removed |
| `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py` | `go run ./cmd/bigclawctl automation e2e continuation-policy-gate ...` | migrated and Python shim removed |
| `bigclaw-go/scripts/benchmark/soak_local.py` | `go run ./cmd/bigclawctl automation benchmark soak-local ...` | migrated and Python shim removed |
| `bigclaw-go/scripts/benchmark/run_matrix.py` | `go run ./cmd/bigclawctl automation benchmark run-matrix ...` | migrated and Python shim removed |
| `bigclaw-go/scripts/benchmark/capacity_certification.py` | `go run ./cmd/bigclawctl automation benchmark capacity-certification ...` | migrated and Python shim removed |
| `bigclaw-go/scripts/migration/shadow_compare.py` | `go run ./cmd/bigclawctl automation migration shadow-compare ...` | migrated and Python shim removed |
| `bigclaw-go/scripts/migration/shadow_matrix.py` | `go run ./cmd/bigclawctl automation migration shadow-matrix ...` | migrated and Python shim removed |
| `bigclaw-go/scripts/migration/live_shadow_scorecard.py` | `go run ./cmd/bigclawctl automation migration live-shadow-scorecard ...` | migrated and Python shim removed |
| `bigclaw-go/scripts/migration/export_live_shadow_bundle.py` | `go run ./cmd/bigclawctl automation migration export-live-shadow-bundle` | migrated and Python shim removed |

## Remaining Python Script Backlog

- `bigclaw-go/scripts/e2e/multi_node_shared_queue.py`
- `bigclaw-go/scripts/e2e/mixed_workload_matrix.py`
- `bigclaw-go/scripts/e2e/external_store_validation.py`
- `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py`
- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py`
- `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`
## Validation Commands

```bash
cd bigclaw-go
go test ./cmd/bigclawctl/...
go run ./cmd/bigclawctl automation --help
go run ./cmd/bigclawctl automation e2e run-task-smoke --help
go run ./cmd/bigclawctl automation e2e export-validation-bundle --help
go run ./cmd/bigclawctl automation e2e continuation-scorecard --help
go run ./cmd/bigclawctl automation e2e continuation-policy-gate --help
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

- Keep new behavior in Go-native entrypoints and reserve Python only for batches that are not yet migrated.
- Migrate the remaining reporting/export scripts in follow-up batches grouped by shared payload shape:
  - migration scorecards/bundle exporters
- Remaining Python generators still need native replacements before they can be removed.

## Branch And PR Suggestion

- Branch: `feat/BIG-GO-902-go-cli-script-migration`
- PR title: `feat: migrate first Python automation scripts to bigclawctl`

## Risks

- `soak-local` now uses Go worker concurrency; very large counts may stress a single local HTTP backend differently than the old Python thread pool.
- `run-task-smoke --autostart` and `soak-local --autostart` still rely on ephemeral port reservation before `bigclawd` binds, so local port races remain possible.
- Remaining Python report generators still exist, so automation ownership is split until later migration batches land.
