# Go CLI Script Migration

Issue: `BIG-GO-902`

## Executable Migration Inventory

```bash
cd bigclaw-go
go run ./cmd/bigclawctl legacy-python inventory --json
```

The command above is the source of truth for the current script migration inventory,
including migrated shims, pending Python-native scripts, recommended wave ordering,
validation commands, compatibility policy, and branch / PR suggestions.

## Implemented In This Slice

| Legacy script | Go CLI replacement | Status |
| --- | --- | --- |
| `bigclaw-go/scripts/e2e/run_task_smoke.py` | `go run ./cmd/bigclawctl automation e2e run-task-smoke ...` | migrated with Python compatibility shim |
| `bigclaw-go/scripts/e2e/export_validation_bundle.py` | `go run ./cmd/bigclawctl automation e2e export-validation-bundle ...` | migrated with Python compatibility shim |
| `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py` | `go run ./cmd/bigclawctl automation e2e validation-bundle-scorecard ...` | migrated with Python compatibility shim |
| `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py` | `go run ./cmd/bigclawctl automation e2e validation-bundle-policy-gate ...` | migrated with Python compatibility shim |
| `bigclaw-go/scripts/benchmark/soak_local.py` | `go run ./cmd/bigclawctl automation benchmark soak-local ...` | migrated with Python compatibility shim |
| `bigclaw-go/scripts/migration/shadow_compare.py` | `go run ./cmd/bigclawctl automation migration shadow-compare ...` | migrated with Python compatibility shim |

## First-Batch Migration / Adaptation Queue

### Wave 1: benchmark orchestration

- `bigclaw-go/scripts/benchmark/run_matrix.py`
- `bigclaw-go/scripts/benchmark/capacity_certification.py`

Why first remaining wave:

- They already depend on the migrated `soak-local` execution path.
- Their main risk is aggregation/report shape, not task execution semantics.

### Wave 2: migration scorecards and bundle exporters

- `bigclaw-go/scripts/migration/shadow_matrix.py`
- `bigclaw-go/scripts/migration/live_shadow_scorecard.py`
- `bigclaw-go/scripts/migration/export_live_shadow_bundle.py`

Why second:

- These scripts are report-heavy and can reuse the Wave 1 exporter patterns.
- They affect migration evidence rather than core operator flows.

### Deferred longer-tail e2e matrices

- `bigclaw-go/scripts/e2e/multi_node_shared_queue.py`
- `bigclaw-go/scripts/e2e/mixed_workload_matrix.py`
- `bigclaw-go/scripts/e2e/external_store_validation.py`
- `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py`
- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py`
- `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`

Why deferred:

- They are scenario-heavy harnesses rather than common operator entrypoints.
- They likely need shared Go helper packages before a clean CLI migration is worth doing.

## Validation Commands

```bash
cd bigclaw-go
go test ./cmd/bigclawctl/...
go run ./cmd/bigclawctl legacy-python inventory --json
go run ./cmd/bigclawctl automation --help
go run ./cmd/bigclawctl automation e2e run-task-smoke --help
go run ./cmd/bigclawctl automation e2e export-validation-bundle --help
go run ./cmd/bigclawctl automation e2e validation-bundle-scorecard --help
go run ./cmd/bigclawctl automation e2e validation-bundle-policy-gate --help
go run ./cmd/bigclawctl automation benchmark soak-local --help
go run ./cmd/bigclawctl automation migration shadow-compare --help
```

## Regression Surface

- CLI parsing and root help for `bigclawctl`
- HTTP polling against `/healthz`, `/tasks/:id`, and `/events`
- Temporary `bigclawd` autostart state wiring for smoke and soak commands
- Validation bundle export semantics under `docs/reports/live-validation-runs/`
- Continuation scorecard and policy gate decision logic for bundle freshness / lane coverage
- Report serialization compatibility for JSON consumers that previously read the Python script output
- Python shim forwarding for operators still calling the legacy script paths

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
- PR title: `feat: migrate script-layer automation to bigclawctl`

## Risks

- `soak-local` now uses Go worker concurrency; very large counts may stress a single local HTTP backend differently than the old Python thread pool.
- `run-task-smoke --autostart` and `soak-local --autostart` still rely on ephemeral port reservation before `bigclawd` binds, so local port races remain possible.
- `export-validation-bundle` preserves the repo-native JSON and markdown shape, but downstream tooling may still have hidden assumptions about field ordering or omitted keys.
- The continuation scripts now run through Go with repository-root path resolution; callers that depended on importing Python helpers directly must switch to CLI invocation.
- Remaining Python report generators still exist, so automation ownership is split until later migration batches land.
