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
  - `bigclaw-go/scripts/e2e/external_store_validation.py`
  - `bigclaw-go/scripts/e2e/mixed_workload_matrix.py`
  - `bigclaw-go/scripts/e2e/multi_node_shared_queue.py`
  - `bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py`
  - `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`
Repository-wide Python file count at the start of this continuation: `97`

## Plan

1. Migrate `broker_failover_stub_matrix.py` into `bigclawctl automation e2e` as a Go-native canonical-artifact republisher.
2. Update docs/tests to call the Go-native broker failover stub matrix command.
3. Delete the Python broker failover stub matrix generator and its Python-only test, then refresh batch inventory and validation results.

## Acceptance

- State the exact scoped Python file list for this batch after this continuation.
- Confirm how many files were removed from `bigclaw-go/scripts/benchmark/**` and this continuation’s `scripts/e2e/**` sub-batch.
- Record delete/replace/keep rationale for the migrated broker failover stub matrix plus the remaining selected `scripts/e2e/**` files.
- Report repository-wide Python count impact for this lane after the broker failover stub matrix migration.
- Capture that the broker failover stub replacement uses checked-in canonical JSON/artifacts rather than re-simulating every scenario in Go.
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
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e coordination-capability-surface --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e broker-failover-stub-matrix --help`
- `cd bigclaw-go && tmpdir=$(mktemp -d) && go run ./cmd/bigclawctl automation e2e broker-failover-stub-matrix --output "$tmpdir/report.json" --artifact-root "$tmpdir/artifacts" --checkpoint-fencing-summary-output "$tmpdir/checkpoint.json" --retention-boundary-summary-output "$tmpdir/retention.json" >/dev/null`
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
  - `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py`
    - Replaced by `go run ./cmd/bigclawctl automation e2e coordination-capability-surface ...`
  - `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py`
    - Replaced by `go run ./cmd/bigclawctl automation e2e broker-failover-stub-matrix ...`
    - Go replacement republishes the checked-in canonical broker stub report, proof summaries, and per-scenario artifact tree instead of re-simulating every deterministic scenario in-process.
  - `bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py`
    - Coverage moved into `bigclaw-go/cmd/bigclawctl/automation_commands_test.go`

- Kept in this lane:
  - `bigclaw-go/scripts/e2e/external_store_validation.py`
  - `bigclaw-go/scripts/e2e/mixed_workload_matrix.py`
  - `bigclaw-go/scripts/e2e/multi_node_shared_queue.py`
  - `bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py`
  - `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`
    - Reason: these files still own active report-generation or scenario-surface logic in the checked-in tree, and this issue did not add Go-native replacements for them.

### Python Count Impact

- Repository Python files before the lane: `108`
- Repository Python files now: `95`
- Net repository reduction from this lane: `13`
- Scoped benchmark Python files before the lane: `4`
- Scoped benchmark Python files now: `0`
- Remaining scoped `scripts/e2e` Python files now: `5`

### Validation Record

- `find . -name '*.py' | wc -l`
  - Result: `95`
- `find bigclaw-go/scripts -name '*.py' | sort`
  - Result: returned 5 scoped `bigclaw-go/scripts/e2e/*.py` files and no `bigclaw-go/scripts/benchmark/*.py` files.
- `cd bigclaw-go && go test ./cmd/bigclawctl`
  - Result: `ok  	bigclaw-go/cmd/bigclawctl	2.593s`
- `cd bigclaw-go && go test ./internal/regression -run 'RunAll|Lane8ValidationBundleContinuation|RuntimeReportFollowupDocs|LiveValidationIndex'`
  - Result: `ok  	bigclaw-go/internal/regression	2.596s`
- `cd bigclaw-go && go test ./internal/regression -run 'RunAll|LiveValidation|SharedQueueCompanion|BrokerValidationSummary|RuntimeReportFollowupDocs'`
  - Result: `ok  	bigclaw-go/internal/regression	4.165s`
- `cd bigclaw-go && go test ./internal/regression -run 'Lane8CrossProcessCoordinationSurface|CrossProcessCoordinationReadinessDocsStayAligned|CoordinationContractSurface'`
  - Result: `ok  	bigclaw-go/internal/regression	0.642s`
- `cd bigclaw-go && go test ./internal/regression -run 'DurabilityRollout|DurabilityReviewBundle|SequenceRetentionSurface|RunAll'`
  - Result: `ok  	bigclaw-go/internal/regression	5.435s` and later `ok  	bigclaw-go/internal/regression	(cached)`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark --help`
  - Result: exited `0` and printed `usage: bigclawctl automation benchmark <soak-local|run-matrix|capacity-certification> [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e --help`
  - Result: exited `0` and printed `usage: bigclawctl automation e2e <run-task-smoke|export-validation-bundle|coordination-capability-surface|broker-failover-stub-matrix|continuation-scorecard|continuation-policy-gate> [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e export-validation-bundle --help`
  - Result: exited `0` and printed the exporter flags including `--go-root`, `--run-id`, `--bundle-dir`, `--summary-path`, `--index-path`, and `--manifest-path`.
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e coordination-capability-surface --help`
  - Result: exited `0` and printed the coordination surface flags including `--multi-node-report`, `--takeover-report`, `--live-takeover-report`, and `--output`.
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e broker-failover-stub-matrix --help`
  - Result: exited `0` and printed broker stub matrix flags including `--output`, `--artifact-root`, `--checkpoint-fencing-summary-output`, and `--retention-boundary-summary-output`.
- `cd bigclaw-go && tmpdir=$(mktemp -d) && go run ./cmd/bigclawctl automation e2e broker-failover-stub-matrix --output "$tmpdir/report.json" --artifact-root "$tmpdir/artifacts" --checkpoint-fencing-summary-output "$tmpdir/checkpoint.json" --retention-boundary-summary-output "$tmpdir/retention.json" >/dev/null`
  - Result: exited `0` and wrote the broker stub report, both proof summaries, and copied artifact files including `BF-04/checkpoint-transition-log.json` under a temporary directory.
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
