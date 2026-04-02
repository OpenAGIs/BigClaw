# BIG-GO-1009 Workpad

## Scope

Issue: `BIG-GO-1009`
Title: `Terminal sweep I: bigclaw-go benchmark/e2e scripts batch`

This lane is limited to the benchmark batch and the adjacent `scripts/e2e` leftovers referenced by the issue:

- `bigclaw-go/scripts/benchmark/**`
- selected `bigclaw-go/scripts/e2e/**`

Current tree snapshot at the start of this pass:

- `bigclaw-go/scripts/benchmark/**` contains no Python files.
- The scoped `bigclaw-go/scripts/benchmark/**` and selected `bigclaw-go/scripts/e2e/**` batch now contains no Python files.
- Repository-wide Python file count at the start of the final cleanup sub-batch: `93`

## Plan

1. Migrate the remaining shared-queue and subscriber-takeover Python generators into `bigclawctl automation e2e`.
2. Update docs/tests and checked-in evidence metadata to point at Go-native ownership instead of deleted Python paths.
3. Delete the remaining Python generators/tests in the scoped batch, then refresh inventory and validation results.

## Acceptance

- State the exact scoped Python file list for this batch after this continuation.
- Confirm how many files were removed from `bigclaw-go/scripts/benchmark/**` and this continuation’s `scripts/e2e/**` sub-batch.
- Record delete/replace/keep rationale for the final shared-queue and takeover generators.
- Report repository-wide Python count impact for the full lane after the final migration sub-batch.
- Capture that the final shared-queue and takeover replacements use checked-in canonical reports/artifacts rather than re-running the old Python harnesses in Go.
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
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e external-store-validation --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e mixed-workload-matrix --help`
- `cd bigclaw-go && tmpdir=$(mktemp -d) && go run ./cmd/bigclawctl automation e2e mixed-workload-matrix --output "$tmpdir/mixed-workload-matrix-report.json" >/dev/null`
- `cd bigclaw-go && tmpdir=$(mktemp -d) && go run ./cmd/bigclawctl automation e2e external-store-validation --output "$tmpdir/external-store-validation-report.json" >/dev/null`
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
  - `bigclaw-go/scripts/e2e/external_store_validation.py`
    - Replaced by `go run ./cmd/bigclawctl automation e2e external-store-validation ...`
    - Go replacement republishes the checked-in canonical external-store validation report instead of re-running the prior live multi-process harness orchestration in Go.
  - `bigclaw-go/scripts/e2e/mixed_workload_matrix.py`
    - Replaced by `go run ./cmd/bigclawctl automation e2e mixed-workload-matrix ...`
    - Go replacement republishes the checked-in canonical mixed workload matrix report instead of re-running the old mixed-executor routing harness in Go.
  - `bigclaw-go/scripts/e2e/multi_node_shared_queue.py`
    - Replaced by `go run ./cmd/bigclawctl automation e2e multi-node-shared-queue ...`
    - Go replacement republishes the checked-in shared-queue report, live takeover report, and live takeover artifact tree instead of re-running the old two-node Python harness.
  - `bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py`
    - Coverage moved into `bigclaw-go/cmd/bigclawctl/automation_commands_test.go`
  - `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`
    - Replaced by `go run ./cmd/bigclawctl automation e2e subscriber-takeover-harness ...`
    - Go replacement republishes the checked-in deterministic takeover harness report instead of re-running the old Python local simulation.

- Kept in this lane:
  - None in the scoped batch.

### Python Count Impact

- Repository Python files before the lane: `108`
- Repository Python files now: `90`
- Net repository reduction from this lane: `18`
- Scoped benchmark Python files before the lane: `4`
- Scoped benchmark Python files now: `0`
- Remaining scoped `scripts/e2e` Python files now: `0`

### Validation Record

- `find . -name '*.py' | wc -l`
  - Result: `90`
- `find bigclaw-go/scripts -name '*.py' | sort`
  - Result: returned no Python files under `bigclaw-go/scripts`.
- `cd bigclaw-go && go test ./cmd/bigclawctl`
  - Result: `ok  	bigclaw-go/cmd/bigclawctl	2.593s`, later `ok  	bigclaw-go/cmd/bigclawctl	4.374s`, later `ok  	bigclaw-go/cmd/bigclawctl	3.978s`, and finally `ok  	bigclaw-go/cmd/bigclawctl	2.340s`
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
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e external-store-validation --help`
  - Result: exited `0` and printed external-store validation flags including `--source-report` and `--output`.
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e mixed-workload-matrix --help`
  - Result: exited `0` and printed mixed-workload matrix flags including `--source-report` and `--output`.
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e multi-node-shared-queue --help`
  - Result: exited `0` and printed shared queue republisher flags including `--output`, `--takeover-report-output`, and `--takeover-artifact-dir`.
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e subscriber-takeover-harness --help`
  - Result: exited `0` and printed takeover harness republisher flags including `--source-report` and `--output`.
- `cd bigclaw-go && tmpdir=$(mktemp -d) && go run ./cmd/bigclawctl automation e2e mixed-workload-matrix --output "$tmpdir/mixed-workload-matrix-report.json" >/dev/null`
  - Result: exited `0` and wrote `mixed-workload-matrix-report.json` into a temporary directory.
- `cd bigclaw-go && tmpdir=$(mktemp -d) && go run ./cmd/bigclawctl automation e2e multi-node-shared-queue --output "$tmpdir/multi-node-shared-queue-report.json" --takeover-report-output "$tmpdir/live-multi-node-subscriber-takeover-report.json" --takeover-artifact-dir "$tmpdir/live-multi-node-subscriber-takeover-artifacts" >/dev/null && go run ./cmd/bigclawctl automation e2e subscriber-takeover-harness --output "$tmpdir/multi-subscriber-takeover-validation-report.json" >/dev/null`
  - Result: exited `0` and wrote the shared queue report, live takeover report, deterministic takeover report, and copied live takeover audit artifacts into a temporary directory.
- `cd bigclaw-go && tmpdir=$(mktemp -d) && go run ./cmd/bigclawctl automation e2e external-store-validation --output "$tmpdir/external-store-validation-report.json" >/dev/null`
  - Result: exited `0` and wrote `external-store-validation-report.json` into a temporary directory.
- `cd bigclaw-go && tmpdir=$(mktemp -d) && go run ./cmd/bigclawctl automation e2e broker-failover-stub-matrix --output "$tmpdir/report.json" --artifact-root "$tmpdir/artifacts" --checkpoint-fencing-summary-output "$tmpdir/checkpoint.json" --retention-boundary-summary-output "$tmpdir/retention.json" >/dev/null`
  - Result: exited `0` and wrote the broker stub report, both proof summaries, and copied artifact files including `BF-04/checkpoint-transition-log.json` under a temporary directory.
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e continuation-scorecard --help`
  - Result: exited `0` and printed the continuation scorecard flags including `--go-root`, `--index-manifest`, `--bundle-root`, and `--output`.
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e continuation-policy-gate --help`
  - Result: exited `0` and printed the continuation policy gate flags including `--scorecard`, `--enforcement-mode`, `--max-latest-age-hours`, and `--output`.
- `cd bigclaw-go && go test ./internal/regression -run 'ExternalStoreValidation|ProviderLiveHandoff|CrossProcessCoordination|RuntimeReportFollowupDocs|IssueCoverage'`
  - Result: `ok  	bigclaw-go/internal/regression	0.879s`
- `cd bigclaw-go && go test ./internal/regression -run 'IssueCoverage|BenchmarkReadiness|EpicClosure|RuntimeReportFollowupDocs'`
  - Result: `ok  	bigclaw-go/internal/regression	1.635s [no tests to run]`
- `cd bigclaw-go && go test ./internal/regression -run 'SharedQueueReport|TakeoverProof|PythonLane8Remaining|RuntimeReportFollowupDocs|FollowupLaneDocs|LiveMultinodeTakeoverProof|ValidationBundleContinuation'`
  - Result: `ok  	bigclaw-go/internal/regression	0.866s`
- `cd bigclaw-go && tmpdir=$(mktemp -d) && ... && go run ./cmd/bigclawctl automation e2e export-validation-bundle ...`
  - Result: command returned `0` against a temporary evidence fixture and wrote `live-validation-summary.json` plus `live-validation-index.md` with local, kubernetes, ray, broker, and validation-matrix sections.
- `cd bigclaw-go && tmpdir=$(mktemp -d) && go run ./cmd/bigclawctl automation e2e continuation-scorecard --output "$tmpdir/scorecard.json" >/dev/null && go run ./cmd/bigclawctl automation e2e continuation-policy-gate --scorecard "$tmpdir/scorecard.json" --enforcement-mode review --output "$tmpdir/gate.json" >/dev/null`
  - Result: both commands returned `0`; the generated scorecard carried `generator_script = go run ./cmd/bigclawctl automation e2e continuation-scorecard`, and the generated policy gate carried `generator_script = go run ./cmd/bigclawctl automation e2e continuation-policy-gate`.
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark run-matrix --help`
  - Result: exited `0` and printed the `run-matrix` flags including `--scenario`, `--report-path`, and `--timeout-seconds`.
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark capacity-certification --help`
  - Result: exited `0` and printed the `capacity-certification` flags including `--benchmark-report`, `--mixed-workload-report`, `--supplemental-soak-report`, `--output`, and `--markdown-output`.
