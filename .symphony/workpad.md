# BIG-GO-1144

## Plan
- confirm the lane-owned candidate Python paths against the current worktree and record the pre-change repo baseline
- replace prior generic assumptions with issue-specific regression coverage for the benchmark, e2e, and migration script candidates owned by this lane
- assert that the deleted Python entrypoints remain absent and that the repo keeps the Go-native replacement surface in `cmd/bigclawctl`, shell wrappers, and migration docs
- run targeted validation for the Python-file baseline, the new regression tranche, existing benchmark/e2e migration checks, and representative Go help surfaces
- commit and push the scoped change set

## Acceptance
- the `BIG-GO-1144` candidate paths are explicitly covered and remain absent from disk:
- `bigclaw-go/scripts/benchmark/capacity_certification.py`
- `bigclaw-go/scripts/benchmark/capacity_certification_test.py`
- `bigclaw-go/scripts/benchmark/run_matrix.py`
- `bigclaw-go/scripts/benchmark/soak_local.py`
- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py`
- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py`
- `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py`
- `bigclaw-go/scripts/e2e/export_validation_bundle.py`
- `bigclaw-go/scripts/e2e/export_validation_bundle_test.py`
- `bigclaw-go/scripts/e2e/external_store_validation.py`
- `bigclaw-go/scripts/e2e/mixed_workload_matrix.py`
- `bigclaw-go/scripts/e2e/multi_node_shared_queue.py`
- `bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py`
- `bigclaw-go/scripts/e2e/run_all_test.py`
- `bigclaw-go/scripts/e2e/run_task_smoke.py`
- `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`
- `bigclaw-go/scripts/migration/export_live_shadow_bundle.py`
- `bigclaw-go/scripts/migration/live_shadow_scorecard.py`
- `bigclaw-go/scripts/migration/shadow_compare.py`
- `bigclaw-go/scripts/migration/shadow_matrix.py`
- Go-native replacements or compatibility surfaces are asserted for this lane:
- `bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go`
- `bigclaw-go/cmd/bigclawctl/automation_commands.go`
- `bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands.go`
- `bigclaw-go/cmd/bigclawctl/automation_e2e_coordination_surface_command.go`
- `bigclaw-go/cmd/bigclawctl/automation_e2e_external_store_validation_command.go`
- `bigclaw-go/cmd/bigclawctl/automation_e2e_mixed_workload_command.go`
- `bigclaw-go/cmd/bigclawctl/automation_e2e_multi_node_shared_queue_command.go`
- `bigclaw-go/cmd/bigclawctl/automation_e2e_takeover_matrix_command.go`
- `bigclaw-go/cmd/bigclawctl/automation_e2e_broker_stub_command.go`
- `bigclaw-go/scripts/benchmark/run_suite.sh`
- `bigclaw-go/scripts/e2e/run_all.sh`
- `bigclaw-go/scripts/e2e/kubernetes_smoke.sh`
- `bigclaw-go/scripts/e2e/ray_smoke.sh`
- `bigclaw-go/docs/go-cli-script-migration.md`
- the repository remains at zero live `.py` files in the current worktree
- exact validation commands and outcomes are recorded below
- residual risk explicitly notes that the numeric Python-file-count acceptance cannot decrease further because the branch already starts at zero

## Validation
- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run TestTopLevelModulePurgeTranche17`
- `cd bigclaw-go && go test ./internal/regression -run 'TestE2EScriptDirectoryStaysPythonFree|TestE2EMigrationDocListsOnlyActiveEntrypoints'`
- `cd bigclaw-go && go test ./cmd/bigclawctl -run TestBenchmarkScriptsStayGoOnly`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration --help`
- `git status --short`

## Validation Results
- `find . -name '*.py' | wc -l` -> `0`
- `cd bigclaw-go && go test ./internal/regression -run TestTopLevelModulePurgeTranche17` -> `ok  	bigclaw-go/internal/regression	0.483s`
- `cd bigclaw-go && go test ./internal/regression -run 'TestE2EScriptDirectoryStaysPythonFree|TestE2EMigrationDocListsOnlyActiveEntrypoints'` -> `ok  	bigclaw-go/internal/regression	0.244s`
- `cd bigclaw-go && go test ./cmd/bigclawctl -run TestBenchmarkScriptsStayGoOnly` -> `ok  	bigclaw-go/cmd/bigclawctl	3.395s`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark --help` -> exit `0`; printed `usage: bigclawctl automation benchmark <soak-local|run-matrix|capacity-certification> [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e --help` -> exit `0`; printed `usage: bigclawctl automation e2e <run-task-smoke|export-validation-bundle|continuation-scorecard|continuation-policy-gate|broker-failover-stub-matrix|mixed-workload-matrix|cross-process-coordination-surface|subscriber-takeover-fault-matrix|external-store-validation|multi-node-shared-queue> [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration --help` -> exit `0`; printed `usage: bigclawctl automation migration <shadow-compare|shadow-matrix|live-shadow-scorecard|export-live-shadow-bundle> [flags]`
- `git status --short` -> modified `.symphony/workpad.md`; added `bigclaw-go/internal/regression/top_level_module_purge_tranche17_test.go`

## Residual Risk
- the repo already starts from a zero-`.py` baseline in this workspace, so this issue can only harden deletion and replacement coverage for the lane; it cannot make the Python file count numerically lower from the current baseline
