# Go CLI Script Migration

Issues: `BIG-GO-902`, `BIG-GO-950`

## Implemented In This Slice

| Legacy script | Go CLI replacement | Status |
| --- | --- | --- |
| `bigclaw-go/scripts/e2e/run_task_smoke.py` | `go run ./cmd/bigclawctl automation e2e run-task-smoke ...` | migrated and Python wrapper removed |
| `bigclaw-go/scripts/benchmark/soak_local.py` | `go run ./cmd/bigclawctl automation benchmark soak-local ...` | migrated with Python compatibility shim |
| `bigclaw-go/scripts/migration/shadow_compare.py` | `go run ./cmd/bigclawctl automation migration shadow-compare ...` | migrated and Python wrapper removed |

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
- `bigclaw-go/scripts/benchmark/capacity_certification.py`
- `bigclaw-go/scripts/benchmark/run_matrix.py`
- `bigclaw-go/scripts/migration/export_live_shadow_bundle.py`
- `bigclaw-go/scripts/migration/live_shadow_scorecard.py`
- `bigclaw-go/scripts/migration/shadow_matrix.py`

## Validation Commands

```bash
cd bigclaw-go
go test ./cmd/bigclawctl/...
go run ./cmd/bigclawctl automation --help
go run ./cmd/bigclawctl automation e2e run-task-smoke --help
go run ./cmd/bigclawctl automation benchmark soak-local --help
go run ./cmd/bigclawctl automation migration shadow-compare --help
```

## Regression Surface

- CLI parsing and root help for `bigclawctl`
- HTTP polling against `/healthz`, `/tasks/:id`, and `/events`
- Temporary `bigclawd` autostart state wiring for smoke and soak commands
- Report serialization compatibility for JSON consumers that previously read the Python script output
- Direct Go/native command usage in operator docs after removing the `run_task_smoke.py` and `shadow_compare.py` wrappers

## Compatibility Layer Plan

- Keep any future compatibility shim temporary; this lane already removed the `run_task_smoke.py` and `shadow_compare.py` wrappers.
- Do not add new behavior to Python copies; all new logic belongs in Go.
- Migrate the remaining reporting/export scripts in follow-up batches grouped by shared payload shape:
  - validation bundle generators
  - benchmark matrices
  - migration scorecards/bundle exporters
- Remove each remaining Python shim only after the corresponding Go command is referenced by docs, CI, and operators.

## Risks

- `soak-local` now uses Go worker concurrency; very large counts may stress a single local HTTP backend differently than the old Python thread pool.
- `run-task-smoke --autostart` and `soak-local --autostart` still rely on ephemeral port reservation before `bigclawd` binds, so local port races remain possible.
- Remaining Python report generators still exist, so automation ownership is split until later migration batches land.
