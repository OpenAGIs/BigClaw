# BIG-GO-1009 Workpad

## Scope

Target the remaining Python scripts under:

- `bigclaw-go/scripts/benchmark/**`
- selected `bigclaw-go/scripts/e2e/**` files that can be removed or replaced within the same benchmark/e2e batch

Initial batch file list:

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
- `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`

Current repository Python file count before this lane: `108`
Current targeted batch Python file count before this lane: `18`
Current benchmark-batch Python file count before this lane: `4`

## Plan

1. Inspect the benchmark scripts and the adjacent `e2e` Python files to find existing Go or shell entrypoints already covering their behavior.
2. Port benchmark-only Python orchestration into `bigclawctl` or shell where practical, then delete the Python originals.
3. Remove any selected `e2e` Python files that are already redundant because a checked-in shell/Go surface is the canonical path.
4. Update docs/tests only where required by the scoped migration.
5. Run targeted validation, capture exact commands/results, and report the before/after Python counts plus per-file disposition.

## Acceptance

- Produce the exact `BIG-GO-1009` batch file list for `scripts/benchmark/**` and the touched subset of `scripts/e2e/**`.
- Reduce Python file count in this batch as far as practical without broadening scope.
- Document delete/replace/keep rationale for every file in the batch.
- Report repository-wide Python count impact.

## Validation

- `find . -name '*.py' | wc -l`
- targeted `go test` and any script-level verification needed for touched benchmark/e2e replacements
- `git status --short`
- `git log -1 --stat`

## Results

### File Disposition

- `bigclaw-go/scripts/benchmark/soak_local.py`
  - Deleted.
  - Reason: fully replaced by `go run ./cmd/bigclawctl automation benchmark soak-local ...`; the compatibility shim is no longer needed.
- `bigclaw-go/scripts/benchmark/run_matrix.py`
  - Deleted.
  - Reason: replaced in this lane by `go run ./cmd/bigclawctl automation benchmark run-matrix ...`, with benchmark stdout parsing, soak scenario orchestration, and JSON report writing moved into `cmd/bigclawctl`.
- `bigclaw-go/scripts/benchmark/capacity_certification.py`
  - Deleted.
  - Reason: replaced in this lane by `go run ./cmd/bigclawctl automation benchmark capacity-certification ...`, with report generation and markdown rendering moved into `cmd/bigclawctl`.
- `bigclaw-go/scripts/benchmark/capacity_certification_test.py`
  - Deleted.
  - Reason: coverage moved into `bigclaw-go/cmd/bigclawctl/automation_commands_test.go`.
- `bigclaw-go/scripts/e2e/run_all_test.py`
  - Deleted.
  - Reason: coverage moved into `bigclaw-go/internal/regression/run_all_script_test.go`, so `run_all.sh` behavior is now exercised by Go tests instead of Python unittest.
- Remaining targeted `e2e` Python files
  - Kept for now.
  - Reason: they still own live/report-generation logic with no checked-in Go-native replacement in this slice.

### Remaining Targeted Python Files

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

### Python File Count Impact

- Repository Python files before: `108`
- Repository Python files after: `103`
- Targeted batch Python files before: `18`
- Targeted batch Python files after: `13`
- Benchmark-batch Python files before: `4`
- Benchmark-batch Python files after: `0`
- Net reduction: `5`

### Validation Record

- `cd bigclaw-go && go test ./cmd/bigclawctl`
  - Result: `ok  	bigclaw-go/cmd/bigclawctl	2.260s`
- `cd bigclaw-go && go test ./internal/regression -run RunAll`
  - Result: `ok  	bigclaw-go/internal/regression	3.981s`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark --help`
  - Result: `usage: bigclawctl automation benchmark <soak-local|run-matrix|capacity-certification> [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark run-matrix --help`
  - Result: command returned `0` and printed the new `run-matrix` flags, including `--scenario` and `--report-path`.
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark capacity-certification --help`
  - Result: command returned `0` and printed the new `capacity-certification` flags, including `--benchmark-report`, `--supplemental-soak-report`, `--output`, and `--markdown-output`.
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark capacity-certification --output docs/reports/capacity-certification-matrix.json --markdown-output docs/reports/capacity-certification-report.md`
  - Result: command returned `0` and regenerated the checked-in certification JSON/Markdown with `generator_script` set to the new Go CLI entrypoint.
- `find . -name '*.py' | wc -l`
  - Result: `103`
