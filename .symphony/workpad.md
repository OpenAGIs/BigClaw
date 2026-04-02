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
  - `bigclaw-go/scripts/e2e/external_store_validation.py`
  - `bigclaw-go/scripts/e2e/mixed_workload_matrix.py`
  - `bigclaw-go/scripts/e2e/multi_node_shared_queue.py`
  - `bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py`
  - `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`
Repository-wide Python file count at the start of this continuation: `100`

## Plan

1. Migrate `export_validation_bundle.py` into `bigclawctl automation e2e`.
2. Update `scripts/e2e/run_all.sh`, regression fixtures, and docs/tests to call the Go-native export command.
3. Delete the Python exporter and its Python-only test, then refresh batch inventory and validation results.

## Acceptance

- State the exact scoped Python file list for this batch after this continuation.
- Confirm how many files were removed from `bigclaw-go/scripts/benchmark/**` and this continuation’s `scripts/e2e/**` sub-batch.
- Record delete/replace/keep rationale for the migrated exporter plus the remaining selected `scripts/e2e/**` files.
- Report repository-wide Python count impact for this lane after the exporter migration.
- Capture exact validation commands and results.

## Validation

- `find . -name '*.py' | wc -l`
- `find bigclaw-go/scripts -name '*.py' | sort`
- `cd bigclaw-go && go test ./cmd/bigclawctl`
- `cd bigclaw-go && go test ./internal/regression -run RunAll`
- `cd bigclaw-go && go test ./internal/regression -run Lane8ValidationBundleContinuation`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e export-validation-bundle --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e continuation-scorecard --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e continuation-policy-gate --help`
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
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`
    - Replaced by `go run ./cmd/bigclawctl automation e2e continuation-scorecard ...`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
    - Replaced by `go run ./cmd/bigclawctl automation e2e continuation-policy-gate ...`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`
    - Coverage moved into `bigclaw-go/cmd/bigclawctl/automation_commands_test.go`
  - `bigclaw-go/scripts/e2e/export_validation_bundle.py`
    - Replaced by `go run ./cmd/bigclawctl automation e2e export-validation-bundle ...`
  - `bigclaw-go/scripts/e2e/export_validation_bundle_test.py`
    - Coverage moved into `bigclaw-go/cmd/bigclawctl/automation_commands_test.go`

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
    - Reason: these files still own active report-generation or scenario-surface logic in the checked-in tree, and this issue did not add Go-native replacements for them.

### Python Count Impact

- Repository Python files before the lane: `108`
- Repository Python files now: `98`
- Net repository reduction from this lane: `10`
- Scoped benchmark Python files before the lane: `4`
- Scoped benchmark Python files now: `0`
- Remaining scoped `scripts/e2e` Python files now: `8`

### Validation Record

- `find . -name '*.py' | wc -l`
  - Result: `98`
- `find bigclaw-go/scripts -name '*.py' | sort`
  - Result: returned 8 scoped `bigclaw-go/scripts/e2e/*.py` files and no `bigclaw-go/scripts/benchmark/*.py` files.
- `cd bigclaw-go && go test ./cmd/bigclawctl`
  - Result: `ok  	bigclaw-go/cmd/bigclawctl	5.908s` and later `ok  	bigclaw-go/cmd/bigclawctl	(cached)`
- `cd bigclaw-go && go test ./internal/regression -run 'RunAll|Lane8ValidationBundleContinuation|RuntimeReportFollowupDocs|LiveValidationIndex'`
  - Result: `ok  	bigclaw-go/internal/regression	2.596s`
- `cd bigclaw-go && go test ./internal/regression -run 'RunAll|LiveValidation|SharedQueueCompanion|BrokerValidationSummary|RuntimeReportFollowupDocs'`
  - Result: `ok  	bigclaw-go/internal/regression	4.165s`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark --help`
  - Result: exited `0` and printed `usage: bigclawctl automation benchmark <soak-local|run-matrix|capacity-certification> [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e --help`
  - Result: exited `0` and printed `usage: bigclawctl automation e2e <run-task-smoke|export-validation-bundle|continuation-scorecard|continuation-policy-gate> [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e export-validation-bundle --help`
  - Result: exited `0` and printed the exporter flags including `--go-root`, `--run-id`, `--bundle-dir`, `--summary-path`, `--index-path`, and `--manifest-path`.
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e continuation-scorecard --help`
  - Result: exited `0` and printed the continuation scorecard flags including `--go-root`, `--index-manifest`, `--bundle-root`, and `--output`.
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e continuation-policy-gate --help`
  - Result: exited `0` and printed the continuation policy gate flags including `--scorecard`, `--enforcement-mode`, `--max-latest-age-hours`, and `--output`.
- `cd bigclaw-go && tmpdir=$(mktemp -d) && ... && go run ./cmd/bigclawctl automation e2e export-validation-bundle ...`
  - Result: command returned `0` against a temporary evidence fixture and wrote `live-validation-summary.json` plus `live-validation-index.md` with local, kubernetes, ray, broker, and validation-matrix sections.
- `cd bigclaw-go && tmpdir=$(mktemp -d) && go run ./cmd/bigclawctl automation e2e continuation-scorecard --output "$tmpdir/scorecard.json" >/dev/null && go run ./cmd/bigclawctl automation e2e continuation-policy-gate --scorecard "$tmpdir/scorecard.json" --enforcement-mode review --output "$tmpdir/gate.json" >/dev/null`
  - Result: both commands returned `0`; the generated scorecard carried `generator_script = go run ./cmd/bigclawctl automation e2e continuation-scorecard`, and the generated policy gate carried `generator_script = go run ./cmd/bigclawctl automation e2e continuation-policy-gate`.
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark run-matrix --help`
  - Result: exited `0` and printed the `run-matrix` flags including `--scenario`, `--report-path`, and `--timeout-seconds`.
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark capacity-certification --help`
  - Result: exited `0` and printed the `capacity-certification` flags including `--benchmark-report`, `--mixed-workload-report`, `--supplemental-soak-report`, `--output`, and `--markdown-output`.
