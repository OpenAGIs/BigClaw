# Go CLI Script Migration

Issue: `BIG-GO-902`

## Implemented In This Slice

| Legacy script | Go CLI replacement | Status |
| --- | --- | --- |
| `bigclaw-go/scripts/e2e/run_task_smoke.py` | `go run ./cmd/bigclawctl automation e2e run-task-smoke ...` | migrated with Python compatibility shim |
| `bigclaw-go/scripts/benchmark/run_matrix.py` | `go run ./cmd/bigclawctl automation benchmark run-matrix ...` | migrated (BIG-GO-978) |
| `bigclaw-go/scripts/benchmark/soak_local.py` | `go run ./cmd/bigclawctl automation benchmark soak-local ...` | migrated (BIG-GO-978) |
| `bigclaw-go/scripts/benchmark/capacity_certification.py` | `go run ./cmd/bigclawctl automation benchmark capacity-certification ...` | migrated (BIG-GO-978) |
| `bigclaw-go/scripts/migration/shadow_compare.py` | `go run ./cmd/bigclawctl automation migration shadow-compare ...` | migrated with Python compatibility shim |

## Remaining Python Script Backlog

- `bigclaw-go/scripts/e2e/export_validation_bundle.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
- `bigclaw-go/scripts/e2e/multi_node_shared_queue.py`
- `bigclaw-go/scripts/e2e/mixed_workload_matrix.py`
- `bigclaw-go/scripts/e2e/external_store_validation.py`
- `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py`
- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py`
- `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`
- `bigclaw-go/scripts/migration/export_live_shadow_bundle.py`
- `bigclaw-go/scripts/migration/live_shadow_scorecard.py`
- `bigclaw-go/scripts/migration/shadow_matrix.py`

BIG-GO-978 closed the benchmark batch backlog by fully replacing `bigclaw-go/scripts/benchmark/run_matrix.py` and `bigclaw-go/scripts/benchmark/capacity_certification.py` with `bigclawctl automation benchmark run-matrix` and `bigclawctl automation benchmark capacity-certification`, so those Python helpers can now be deleted after downstream references re-target the Go CLI.

## Validation Commands

```bash
cd bigclaw-go
go test ./cmd/bigclawctl/...
go run ./cmd/bigclawctl automation --help
go run ./cmd/bigclawctl automation e2e run-task-smoke --help
go run ./cmd/bigclawctl automation benchmark run-matrix --help
go run ./cmd/bigclawctl automation benchmark soak-local --help
go run ./cmd/bigclawctl automation benchmark capacity-certification --help
go run ./cmd/bigclawctl automation migration shadow-compare --help
```

## Regression Surface

- CLI parsing and root help for `bigclawctl`
- HTTP polling against `/healthz`, `/tasks/:id`, and `/events`
- Temporary `bigclawd` autostart state wiring for smoke and soak commands
- Report serialization compatibility for JSON consumers that previously read the Python script output
- Python shim forwarding for operators still calling the legacy script paths
- Automation benchmark subcommand tests now live in `cmd/bigclawctl/automation_benchmark_reports_test.go`, replacing the old `scripts/benchmark/capacity_certification_test.py` coverage.

## Compatibility Layer Plan

- Keep the migrated Python entrypoints as thin shims that only forward to `bigclawctl`.
- Do not add new behavior to the Python copies; all new logic belongs in Go.
- Migrate the remaining reporting/export scripts in follow-up batches grouped by shared payload shape:
  - validation bundle generators
  - benchmark matrices
  - migration scorecards/bundle exporters
- Remove each Python shim only after the corresponding Go command is referenced by docs, CI, and operators for one full rollout cycle.

## Branch And PR Suggestion

- Branch: `feat/BIG-GO-902-go-cli-script-migration`
- PR title: `feat: migrate first Python automation scripts to bigclawctl`

## Risks

- `soak-local` now uses Go worker concurrency; very large counts may stress a single local HTTP backend differently than the old Python thread pool.
- `run-task-smoke --autostart` and `soak-local --autostart` still rely on ephemeral port reservation before `bigclawd` binds, so local port races remain possible.
- Remaining Python report generators still exist, so automation ownership is split until later migration batches land.
