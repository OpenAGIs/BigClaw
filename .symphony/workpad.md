# BIG-GO-1009 Workpad

## Scope

Issue: `BIG-GO-1009`
Title: `Terminal sweep I: bigclaw-go benchmark/e2e scripts batch`

This lane is limited to the benchmark batch and the adjacent `scripts/e2e` leftovers referenced by the issue:

- `bigclaw-go/scripts/benchmark/**`
- selected `bigclaw-go/scripts/e2e/**`

Current tree snapshot at the start of this pass:

- `bigclaw-go/scripts/benchmark/**` contains no Python files.
- The only Python files still in the scoped area are:
  - `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py`
  - `bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py`
  - `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py`
  - `bigclaw-go/scripts/e2e/export_validation_bundle.py`
  - `bigclaw-go/scripts/e2e/export_validation_bundle_test.py`
  - `bigclaw-go/scripts/e2e/external_store_validation.py`
  - `bigclaw-go/scripts/e2e/mixed_workload_matrix.py`
  - `bigclaw-go/scripts/e2e/multi_node_shared_queue.py`
  - `bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py`
  - `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`

Repository-wide Python file count at the start of this pass: `103`

## Plan

1. Reconcile the branch contents with the issue acceptance so the work record matches the actual repository state.
2. Re-run targeted validation for the benchmark migration surfaces already moved to Go and shell.
3. Record the exact delete/replace/keep rationale for this batch, then commit and push the refreshed issue record.

## Acceptance

- State the exact scoped Python file list for this batch.
- Confirm how many files were removed from `bigclaw-go/scripts/benchmark/**`.
- Record delete/replace/keep rationale for the benchmark batch and the remaining selected `scripts/e2e/**` files.
- Report repository-wide Python count impact for this lane.
- Capture exact validation commands and results.

## Validation

- `find . -name '*.py' | wc -l`
- `find bigclaw-go/scripts -name '*.py' | sort`
- `cd bigclaw-go && go test ./cmd/bigclawctl`
- `cd bigclaw-go && go test ./internal/regression -run RunAll`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark run-matrix --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark capacity-certification --help`
- `git status --short`
- `git log -1 --stat`

## Results

### Batch Disposition

- Deleted and replaced in this lane:
  - `bigclaw-go/scripts/benchmark/soak_local.py`
    - Replaced by `go run ./cmd/bigclawctl automation benchmark soak-local ...`
  - `bigclaw-go/scripts/benchmark/run_matrix.py`
    - Replaced by `go run ./cmd/bigclawctl automation benchmark run-matrix ...`
  - `bigclaw-go/scripts/benchmark/capacity_certification.py`
    - Replaced by `go run ./cmd/bigclawctl automation benchmark capacity-certification ...`
  - `bigclaw-go/scripts/benchmark/capacity_certification_test.py`
    - Coverage moved into `bigclaw-go/cmd/bigclawctl/automation_commands_test.go`
  - `bigclaw-go/scripts/e2e/run_all_test.py`
    - Coverage moved into `bigclaw-go/internal/regression/run_all_script_test.go`

- Kept in this lane:
  - `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py`
  - `bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py`
  - `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py`
  - `bigclaw-go/scripts/e2e/export_validation_bundle.py`
  - `bigclaw-go/scripts/e2e/export_validation_bundle_test.py`
  - `bigclaw-go/scripts/e2e/external_store_validation.py`
  - `bigclaw-go/scripts/e2e/mixed_workload_matrix.py`
  - `bigclaw-go/scripts/e2e/multi_node_shared_queue.py`
  - `bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py`
  - `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`
    - Reason: these files still own active report-generation or scenario-surface logic in the checked-in tree, and this issue did not add Go-native replacements for them.

### Python Count Impact

- Repository Python files before the lane: `108`
- Repository Python files now: `103`
- Net repository reduction from this lane: `5`
- Scoped benchmark Python files before the lane: `4`
- Scoped benchmark Python files now: `0`
- Remaining scoped `scripts/e2e` Python files now: `13`

### Validation Record

- `find . -name '*.py' | wc -l`
  - Result: `103`
- `find bigclaw-go/scripts -name '*.py' | sort`
  - Result: returned the 13 scoped `bigclaw-go/scripts/e2e/*.py` files listed above and no `bigclaw-go/scripts/benchmark/*.py` files.
- `cd bigclaw-go && go test ./cmd/bigclawctl`
  - Result: `ok  	bigclaw-go/cmd/bigclawctl	5.731s`
- `cd bigclaw-go && go test ./internal/regression -run RunAll`
  - Result: `ok  	bigclaw-go/internal/regression	8.353s`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark --help`
  - Result: exited `0` and printed `usage: bigclawctl automation benchmark <soak-local|run-matrix|capacity-certification> [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark run-matrix --help`
  - Result: exited `0` and printed the `run-matrix` flags including `--scenario`, `--report-path`, and `--timeout-seconds`.
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark capacity-certification --help`
  - Result: exited `0` and printed the `capacity-certification` flags including `--benchmark-report`, `--mixed-workload-report`, `--supplemental-soak-report`, `--output`, and `--markdown-output`.
