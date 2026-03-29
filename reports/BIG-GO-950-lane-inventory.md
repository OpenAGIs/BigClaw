# BIG-GO-950 Lane Inventory

Source of truth for this lane is the checked-out repository contents under `bigclaw-go/scripts/e2e` and `bigclaw-go/scripts/migration`. The referenced report `reports/go-migration-lanes-2026-03-29.md` is not present in this worktree, so this inventory is based on the actual files on disk.

## Delivered In This Change

| Path | Current status | Go/native replacement |
| --- | --- | --- |
| `bigclaw-go/scripts/e2e/run_task_smoke.py` | deleted | `go run ./cmd/bigclawctl automation e2e run-task-smoke ...` |
| `bigclaw-go/scripts/migration/shadow_compare.py` | deleted | `go run ./cmd/bigclawctl automation migration shadow-compare ...` |
| `bigclaw-go/scripts/e2e/kubernetes_smoke.sh` | retained shell wrapper | calls `go run ./cmd/bigclawctl automation e2e run-task-smoke ...` |
| `bigclaw-go/scripts/e2e/ray_smoke.sh` | retained shell wrapper | calls `go run ./cmd/bigclawctl automation e2e run-task-smoke ...` |
| `bigclaw-go/scripts/e2e/run_all.sh` | retained shell workflow | local smoke lane now calls `go run ./cmd/bigclawctl automation e2e run-task-smoke ...` |

## Remaining Lane Files

### `bigclaw-go/scripts/e2e`

| Path | Role | Planned Go/native end state |
| --- | --- | --- |
| `bigclaw-go/scripts/e2e/broker_bootstrap_summary.go` | already Go | keep as-is |
| `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py` | deterministic broker failover proof generator | migrate to Go harness using `internal/events` plus broker/reporting structs |
| `bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py` | Python unit test for the generator | delete after Go harness gets Go tests |
| `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py` | JSON/report surface aggregator | migrate to Go report generator reusing `internal/api/coordination_surface.go` |
| `bigclaw-go/scripts/e2e/export_validation_bundle.py` | validation bundle exporter | migrate to Go report generator, then remove Python entrypoint |
| `bigclaw-go/scripts/e2e/export_validation_bundle_test.py` | Python unit test for exporter | delete after Go exporter has Go tests |
| `bigclaw-go/scripts/e2e/external_store_validation.py` | live external store validation harness | migrate to Go harness using runtime and API packages |
| `bigclaw-go/scripts/e2e/kubernetes_smoke.sh` | native shell wrapper | keep shell wrapper |
| `bigclaw-go/scripts/e2e/mixed_workload_matrix.py` | mixed workload matrix generator | migrate to Go harness/report generator |
| `bigclaw-go/scripts/e2e/multi_node_shared_queue.py` | live shared queue and takeover proof harness | migrate to Go harness using `internal/api`, `internal/worker`, and `internal/events` |
| `bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py` | Python unit test for shared queue helpers | delete after Go harness has Go tests |
| `bigclaw-go/scripts/e2e/ray_smoke.sh` | native shell wrapper | keep shell wrapper |
| `bigclaw-go/scripts/e2e/run_all.sh` | native shell orchestrator | keep shell wrapper while downstream generators are still Python |
| `bigclaw-go/scripts/e2e/run_all_test.py` | Python orchestration regression test | convert to Go or shell-level regression test after bundle generators move to Go |
| `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py` | deterministic takeover proof generator | migrate to Go harness using `internal/events` and `internal/api` |
| `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py` | continuation policy gate generator | migrate to Go report generator reusing continuation document structs |
| `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py` | Python unit test for gate | delete after Go gate has Go tests |
| `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py` | continuation scorecard generator | migrate to Go report generator reusing continuation document structs |

### `bigclaw-go/scripts/migration`

| Path | Role | Planned Go/native end state |
| --- | --- | --- |
| `bigclaw-go/scripts/migration/export_live_shadow_bundle.py` | live shadow bundle exporter | migrate to Go report generator reusing `internal/api/live_shadow_surface.go` data shapes |
| `bigclaw-go/scripts/migration/live_shadow_scorecard.py` | live shadow scorecard generator | migrate to Go report generator reusing `internal/api/live_shadow_surface.go` data shapes |
| `bigclaw-go/scripts/migration/shadow_matrix.py` | matrix runner over shadow-compare inputs | migrate to `bigclawctl automation migration shadow-matrix` |

## Validation Commands For This Lane

```bash
cd bigclaw-go
bash -n scripts/e2e/run_all.sh scripts/e2e/kubernetes_smoke.sh scripts/e2e/ray_smoke.sh
go test ./cmd/bigclawctl/...
python3 scripts/e2e/run_all_test.py
```

## Residual Risks

- `run_all.sh` still depends on Python exporters for validation bundle and continuation artifacts, so Python remains on the closeout path for full bundle generation.
- `migration-shadow.md` and `e2e-validation.md` still document several Python-only generators because those generators have not been ported yet.
- The largest remaining scripts are live harnesses, not thin wrappers; deleting them before a Go harness exists would remove executable coverage.
