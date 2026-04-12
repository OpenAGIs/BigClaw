# BIG-GO-1613 Python Asset Sweep

## Scope

This sweep closes the remaining `bigclaw-go/scripts/**/*.py` runner bucket that
was historically used for benchmark, migration, and e2e orchestration.

The targeted retired runner paths are:

- `bigclaw-go/scripts/benchmark/capacity_certification.py`
- `bigclaw-go/scripts/benchmark/run_matrix.py`
- `bigclaw-go/scripts/benchmark/soak_local.py`
- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py`
- `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py`
- `bigclaw-go/scripts/e2e/export_validation_bundle.py`
- `bigclaw-go/scripts/e2e/external_store_validation.py`
- `bigclaw-go/scripts/e2e/mixed_workload_matrix.py`
- `bigclaw-go/scripts/e2e/multi_node_shared_queue.py`
- `bigclaw-go/scripts/e2e/run_task_smoke.py`
- `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`
- `bigclaw-go/scripts/migration/export_live_shadow_bundle.py`
- `bigclaw-go/scripts/migration/live_shadow_scorecard.py`
- `bigclaw-go/scripts/migration/shadow_compare.py`
- `bigclaw-go/scripts/migration/shadow_matrix.py`

## Sweep Result

- Repository-wide Python file count: `0`.
- `bigclaw-go/scripts/benchmark`: `0` Python files.
- `bigclaw-go/scripts/e2e`: `0` Python files.
- `bigclaw-go/scripts/migration`: retired directory absent.
- The current branch baseline was already Python-free for these buckets, so
  BIG-GO-1613 hardens the deletion state with regression coverage and fresh
  lane evidence instead of removing in-branch `.py` files.

## Go Or Native Replacement Paths

- `bigclaw-go/cmd/bigclawctl/automation_commands.go`
- `bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go`
- `bigclaw-go/cmd/bigclawctl/automation_e2e_broker_stub_command.go`
- `bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands.go`
- `bigclaw-go/cmd/bigclawctl/automation_e2e_coordination_surface_command.go`
- `bigclaw-go/cmd/bigclawctl/automation_e2e_external_store_validation_command.go`
- `bigclaw-go/cmd/bigclawctl/automation_e2e_mixed_workload_command.go`
- `bigclaw-go/cmd/bigclawctl/automation_e2e_multi_node_shared_queue_command.go`
- `bigclaw-go/cmd/bigclawctl/automation_e2e_takeover_matrix_command.go`
- `bigclaw-go/cmd/bigclawctl/automation_commands_test.go`
- `bigclaw-go/docs/go-cli-script-migration.md`
- `bigclaw-go/scripts/benchmark/run_suite.sh`
- `bigclaw-go/scripts/e2e/run_all.sh`
- `bigclaw-go/scripts/e2e/kubernetes_smoke.sh`
- `bigclaw-go/scripts/e2e/ray_smoke.sh`

## Supported Go Command Surface

- `bigclawctl automation benchmark soak-local|run-matrix|capacity-certification`
- `bigclawctl automation e2e run-task-smoke|export-validation-bundle|continuation-scorecard|continuation-policy-gate|broker-failover-stub-matrix|mixed-workload-matrix|cross-process-coordination-surface|subscriber-takeover-fault-matrix|external-store-validation|multi-node-shared-queue`
- `bigclawctl automation migration shadow-compare|shadow-matrix|live-shadow-scorecard|export-live-shadow-bundle`

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
  Result: `none`
- `find bigclaw-go/scripts/benchmark bigclaw-go/scripts/e2e bigclaw-go/scripts/migration -type f -name '*.py' 2>/dev/null | sort`
  Result: `none`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1613(RepositoryHasNoPythonFiles|RemainingScriptBucketsStayPythonFree|RetiredPythonRunnersRemainAbsent|ReplacementSurfacesRemainAvailable|LaneReportCapturesSweepState)$'`
  Result: recorded in `reports/BIG-GO-1613-validation.md`

## Residual Risk

- The lane validates the current zero-Python baseline and the availability of
  replacement entrypoints, but it does not re-run every benchmark, migration,
  and e2e workflow end-to-end.
