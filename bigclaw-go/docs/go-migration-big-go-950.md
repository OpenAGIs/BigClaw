# BIG-GO-950 Lane Inventory

This lane uses the concrete script inventory under `bigclaw-go/scripts/e2e` and
`bigclaw-go/scripts/migration` as the source of truth because
`reports/go-migration-lanes-2026-03-29.md` is not present in this worktree.

## Migrated In This Lane

| Path | Replacement | Action |
| --- | --- | --- |
| `scripts/e2e/run_task_smoke.py` | `go run ./cmd/bigclawctl automation e2e run-task-smoke ...` | removed Python wrapper |
| `scripts/migration/shadow_compare.py` | `go run ./cmd/bigclawctl automation migration shadow-compare ...` | removed Python wrapper |
| `scripts/e2e/kubernetes_smoke.sh` | same shell entrypoint, now calls `bigclawctl` | rewired to Go |
| `scripts/e2e/ray_smoke.sh` | same shell entrypoint, now calls `bigclawctl` | rewired to Go |
| `scripts/e2e/run_all.sh` | same shell entrypoint, local smoke lane now calls `bigclawctl` | rewired to Go |

## Remaining Lane Backlog

| Path | Recommendation |
| --- | --- |
| `scripts/e2e/export_validation_bundle.py` | rewrite in Go as bundle exporter CLI |
| `scripts/e2e/validation_bundle_continuation_scorecard.py` | rewrite in Go as scorecard/report subcommand |
| `scripts/e2e/validation_bundle_continuation_policy_gate.py` | rewrite in Go as policy gate subcommand |
| `scripts/e2e/multi_node_shared_queue.py` | rewrite in Go using existing queue/api packages |
| `scripts/e2e/mixed_workload_matrix.py` | rewrite in Go using `bigclawctl` smoke primitives |
| `scripts/e2e/external_store_validation.py` | rewrite in Go using event-log service packages |
| `scripts/e2e/cross_process_coordination_surface.py` | rewrite in Go as report surface generator |
| `scripts/e2e/broker_failover_stub_matrix.py` | rewrite in Go with deterministic broker harness types |
| `scripts/e2e/subscriber_takeover_fault_matrix.py` | rewrite in Go against lease/checkpoint packages |
| `scripts/migration/shadow_matrix.py` | rewrite in Go on top of `automation migration shadow-compare` primitives |
| `scripts/migration/live_shadow_scorecard.py` | rewrite in Go as migration scorecard CLI |
| `scripts/migration/export_live_shadow_bundle.py` | rewrite in Go as migration bundle exporter CLI |

## Verification Commands

```bash
cd bigclaw-go
go test ./cmd/bigclawctl/...
python3 scripts/e2e/run_all_test.py
go run ./cmd/bigclawctl automation e2e run-task-smoke --help
go run ./cmd/bigclawctl automation migration shadow-compare --help
```

## Residual Risk

- The remaining Python scripts are report generators and deterministic harnesses
  with no one-to-one Go CLI yet, so Python still exists in this lane after the
  wrapper removal.
