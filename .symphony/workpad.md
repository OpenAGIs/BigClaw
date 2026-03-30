# BIG-GO-989 Workpad

## Scope

Targeted Python batch for this lane:

- `bigclaw-go/scripts/benchmark/capacity_certification.py`
- `bigclaw-go/scripts/benchmark/capacity_certification_test.py`
- `bigclaw-go/scripts/benchmark/run_matrix.py`
- `bigclaw-go/scripts/benchmark/soak_local.py`
- `bigclaw-go/scripts/e2e/run_task_smoke.py`

Supporting in-scope references likely requiring updates:

- `bigclaw-go/cmd/bigclawctl/automation_commands.go`
- `bigclaw-go/cmd/bigclawctl/automation_commands_test.go`
- `bigclaw-go/scripts/e2e/kubernetes_smoke.sh`
- `bigclaw-go/scripts/e2e/ray_smoke.sh`
- `bigclaw-go/scripts/e2e/run_all.sh`
- `bigclaw-go/README.md`
- `bigclaw-go/docs/e2e-validation.md`
- `bigclaw-go/docs/reports/*.md`
- `bigclaw-go/docs/reports/*.json`

Current repository Python file count before this lane: `116`
Current targeted batch size before this lane: `5`

## Plan

1. Confirm the selected benchmark and `e2e` scripts are either already thin shims or are cleanly replaceable with `bigclawctl automation` Go entrypoints.
2. Add any missing Go automation subcommands needed to cover the benchmark report generation scripts.
3. Update shell wrappers, docs, checked-in report metadata, and regression tests to point at the Go-native commands.
4. Delete the replaced Python files from the targeted batch only.
5. Run targeted Go tests and direct command validations, then record exact results and Python file-count impact.
6. Commit and push the scoped changes.

## Acceptance

- Produce the exact `BIG-GO-989` batch file list handled in this lane.
- Reduce Python files in `bigclaw-go/scripts/benchmark/**` and the selected `bigclaw-go/scripts/e2e/**` slice where Go-native replacements exist.
- For each targeted file, record whether it was replaced, deleted, or retained, with a concrete reason.
- Keep changes scoped to benchmark/e2e script migration for this batch.
- Report repository-wide Python file counts before and after the lane.

## Validation

- `cd bigclaw-go && go test ./cmd/bigclawctl`
- `cd bigclaw-go && go test ./internal/regression`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark run-matrix --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark capacity-certification --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e run-task-smoke --help`
- `rg --files -g '*.py' | wc -l`
- `git status --short`

## Results

### File Disposition

- `bigclaw-go/scripts/benchmark/capacity_certification.py`
  - Deleted.
  - Replaced by `go run ./cmd/bigclawctl automation benchmark capacity-certification`.
- `bigclaw-go/scripts/benchmark/capacity_certification_test.py`
  - Deleted.
  - Replaced by Go coverage in `bigclaw-go/cmd/bigclawctl/automation_benchmark_reports_test.go`.
- `bigclaw-go/scripts/benchmark/run_matrix.py`
  - Deleted.
  - Replaced by `go run ./cmd/bigclawctl automation benchmark run-matrix`.
- `bigclaw-go/scripts/benchmark/soak_local.py`
  - Deleted.
  - Replaced by `go run ./cmd/bigclawctl automation benchmark soak-local`.
- `bigclaw-go/scripts/e2e/run_task_smoke.py`
  - Deleted.
  - Replaced by `go run ./cmd/bigclawctl automation e2e run-task-smoke`.

### Retained Python In Adjacent Scope

- `bigclaw-go/scripts/e2e/export_validation_bundle.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
- `bigclaw-go/scripts/e2e/mixed_workload_matrix.py`
- `bigclaw-go/scripts/e2e/multi_node_shared_queue.py`
- `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`
- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py`
- `bigclaw-go/scripts/e2e/external_store_validation.py`
- `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py`

Reason retained: these files still hold substantive checked-in evidence generation, continuation gating, or deterministic `e2e` harness logic with no Go-native replacement in this batch.

### Python File Count Impact

- Repository Python files before: `116`
- Repository Python files after: `111`
- Targeted batch size before: `5`
- Targeted batch size after: `0`
- Net reduction: `5`

### Validation Record

- `cd bigclaw-go && go test ./cmd/bigclawctl`
  - Result: `ok  	bigclaw-go/cmd/bigclawctl	2.660s`
- `cd bigclaw-go && python3 scripts/e2e/run_all_test.py`
  - Result: `Ran 3 tests in 4.272s, OK`
- `cd bigclaw-go && go test ./internal/regression -run 'TestE2EValidationDocsStayAligned|TestReadinessFollowUpIndexDocsStayAligned|TestExternalStoreValidationReportStaysAligned'`
  - Result: `ok  	bigclaw-go/internal/regression	1.175s`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark run-matrix --help`
  - Result: exit `0`; help text rendered with `--scenario`, `--report-path`, and `--timeout-seconds`.
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark capacity-certification --help`
  - Result: exit `0`; help text rendered with benchmark, mixed-workload, output, and markdown flags.
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e run-task-smoke --help`
  - Result: exit `0`; help text rendered with autostart, executor, report, and metadata flags.
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark capacity-certification --go-root . --output docs/reports/capacity-certification-matrix.json --markdown-output docs/reports/capacity-certification-report.md`
  - Result: exit `0`; regenerated the checked-in capacity certification JSON and Markdown artifacts from the Go implementation.
