# BIG-GO-979 Workpad

## Scope

Targeted continuation migration batch under `bigclaw-go/scripts/e2e/`:

- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`
- `bigclaw-go/scripts/e2e/run_all_test.py`
- `bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py`
- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py`
- `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py`
- `bigclaw-go/scripts/e2e/run_task_smoke.py`

Replacement paths for this batch:

- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.go`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_internal_test.go`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.go`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard_internal_test.go`
- `bigclaw-go/scripts/e2e/run_all_internal_test.go`
- `bigclaw-go/scripts/e2e/multi_node_shared_queue_internal_test.go`
- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix_internal_test.go`
- `bigclaw-go/scripts/e2e/cross_process_coordination_surface.go`
- `go run ./cmd/bigclawctl automation e2e run-task-smoke ...`

Current repository Python file count before this sub-batch: `109`
Current `bigclaw-go/scripts/e2e/**` Python file count before this sub-batch: `8`

## Plan

1. Remove the `run_task_smoke.py` compatibility shim and switch repo-local callers to `bigclawctl automation e2e run-task-smoke`.
2. Update shell wrappers, docs, and harness tests to call the Go CLI directly.
3. Re-run the existing `run_all` harness tests and smoke-wrapper checks against the direct Go invocation path.
4. Record the updated batch file list, replacement paths, and Python file-count impact.
5. Commit and push the scoped changes for `BIG-GO-979`.

## Acceptance

- Produce the exact `BIG-GO-979` batch file list under `bigclaw-go/scripts/e2e/**`.
- Reduce Python files in the targeted directory by removing the selected smoke-shim batch and replacing it with direct Go invocation paths.
- Keep changes scoped to the validation-bundle continuation migration batch only.
- Report before/after repository-wide and `bigclaw-go/scripts/e2e/**` Python file counts.

## Validation

- `cd bigclaw-go && go test ./scripts/e2e/validation_bundle_continuation_policy_gate.go ./scripts/e2e/validation_bundle_continuation_policy_gate_internal_test.go`
- `cd bigclaw-go && go test ./scripts/e2e/validation_bundle_continuation_scorecard.go ./scripts/e2e/validation_bundle_continuation_scorecard_internal_test.go`
- `cd bigclaw-go && go run ./scripts/e2e/validation_bundle_continuation_scorecard.go --output bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json`
- `cd bigclaw-go && go run ./scripts/e2e/validation_bundle_continuation_policy_gate.go --scorecard bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json --output bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json`
- `cd bigclaw-go && go test ./scripts/e2e/run_all_internal_test.go`
- `cd bigclaw-go && go test ./scripts/e2e/multi_node_shared_queue_internal_test.go`
- `cd bigclaw-go && go test ./scripts/e2e/broker_failover_stub_matrix_internal_test.go`
- `cd bigclaw-go && go run ./scripts/e2e/cross_process_coordination_surface.go --output bigclaw-go/docs/reports/cross-process-coordination-capability-surface.json`
- `cd bigclaw-go && go test ./internal/regression -run 'TestLane8CrossProcessCoordinationSurfaceStaysAligned|TestLane8FollowupDigestsStayAligned'`
- `cd bigclaw-go && go test ./scripts/e2e/run_all_internal_test.go`
- `cd bigclaw-go && go test ./internal/regression -run 'TestLane8ValidationBundleContinuationScorecardStaysAligned|TestLane8FollowupDigestsStayAligned'`
- `cd bigclaw-go && go test ./internal/regression -run 'TestLane8FollowupDigestsStayAligned'`
- `git status --short`

## Results

### File Disposition

- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
  - Deleted.
  - Replaced by `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.go`.
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`
  - Deleted.
  - Replaced by `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_internal_test.go`.
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`
  - Deleted.
  - Replaced by `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.go`.
- `bigclaw-go/scripts/e2e/run_all_test.py`
  - Deleted.
  - Replaced by `bigclaw-go/scripts/e2e/run_all_internal_test.go`.
- `bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py`
  - Deleted.
  - Replaced by `bigclaw-go/scripts/e2e/multi_node_shared_queue_internal_test.go`.
- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py`
- `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py`
  - Deleted.
  - Replaced by `bigclaw-go/scripts/e2e/cross_process_coordination_surface.go`.
- `bigclaw-go/scripts/e2e/run_task_smoke.py`
  - Deleted.
  - Replaced by direct Go invocation through `go run ./cmd/bigclawctl automation e2e run-task-smoke ...`.

### Python File Count Impact

- Repository Python files before first sub-batch: `116`
- Repository Python files after current sub-batch: `108`
- `bigclaw-go/scripts/e2e/**` Python files before first sub-batch: `15`
- `bigclaw-go/scripts/e2e/**` Python files after current sub-batch: `7`
- Net reduction across this issue so far: `8`
- Net reduction in this continuation sub-batch: `1`

### Validation Record

- `cd bigclaw-go && go test ./scripts/e2e/validation_bundle_continuation_policy_gate.go ./scripts/e2e/validation_bundle_continuation_policy_gate_internal_test.go`
  - Result: `ok  	command-line-arguments	0.773s`
- `cd bigclaw-go && go test ./scripts/e2e/validation_bundle_continuation_scorecard.go ./scripts/e2e/validation_bundle_continuation_scorecard_internal_test.go`
  - Result: `ok  	command-line-arguments	1.582s`
- `cd bigclaw-go && python3 scripts/e2e/run_all_test.py`
  - Result: `Ran 3 tests in 8.460s` and `OK`
- `cd bigclaw-go && go test ./internal/regression -run 'TestLane8ValidationBundleContinuationScorecardStaysAligned|TestLane8FollowupDigestsStayAligned'`
  - Result: `ok  	bigclaw-go/internal/regression	0.496s`
- `cd bigclaw-go && go run ./scripts/e2e/validation_bundle_continuation_scorecard.go --output bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json`
  - Result: exit code `0`
- `cd bigclaw-go && go test ./scripts/e2e/run_all_internal_test.go`
  - Result: `ok  	command-line-arguments	9.123s`
- `cd bigclaw-go && go test ./scripts/e2e/multi_node_shared_queue_internal_test.go`
  - Result: `ok  	command-line-arguments	1.164s`
- `cd bigclaw-go && go test ./scripts/e2e/broker_failover_stub_matrix_internal_test.go`
  - Result: `ok  	command-line-arguments	1.652s`
- `cd bigclaw-go && go run ./scripts/e2e/cross_process_coordination_surface.go --output bigclaw-go/docs/reports/cross-process-coordination-capability-surface.json`
  - Result: exit code `0`
- `cd bigclaw-go && go test ./internal/regression -run 'TestLane8CrossProcessCoordinationSurfaceStaysAligned|TestLane8FollowupDigestsStayAligned|TestCrossProcessCoordinationReadinessDocsStayAligned'`
  - Result: `ok  	bigclaw-go/internal/regression	0.952s`
- `cd bigclaw-go && go test ./scripts/e2e/run_all_internal_test.go`
  - Result: `ok  	command-line-arguments	9.353s`
- `git status --short`
  - Result: only the scoped `BIG-GO-979` files above were modified before commit.

## Continuation Slice: subscriber_takeover_fault_matrix

### Scope

- `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`

Replacement path for this slice:

- `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.go`

Current repository Python file count before this slice: `108`
Current `bigclaw-go/scripts/e2e/**` Python file count before this slice: `7`

### Plan

1. Replace the deterministic local takeover report generator with a Go-native entrypoint that preserves the checked-in report contract.
2. Add a focused Go test for the report summary and scenario schema so the replacement is no longer validated through Python.
3. Update docs and regression references from the Python path to the Go path.
4. Regenerate the takeover report, record the Python-count deltas, then commit and push the scoped slice.

### Acceptance

- Remove `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py` from the remaining `scripts/e2e` Python backlog by replacing it with `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.go`.
- Keep `docs/reports/multi-subscriber-takeover-validation-report.json` and linked takeover docs aligned with the new Go generator.
- Record exact targeted validation commands and before/after Python counts for both the repo and `bigclaw-go/scripts/e2e/**`.

### Validation

- `cd bigclaw-go && go test ./scripts/e2e/subscriber_takeover_fault_matrix.go ./scripts/e2e/subscriber_takeover_fault_matrix_internal_test.go`
- `cd bigclaw-go && go run ./scripts/e2e/subscriber_takeover_fault_matrix.go --output bigclaw-go/docs/reports/multi-subscriber-takeover-validation-report.json`
- `cd bigclaw-go && go test ./internal/regression -run 'TestLane8CrossProcessCoordinationSurfaceStaysAligned|TestLane8FollowupDigestsStayAligned|TestCrossProcessCoordinationReadinessDocsStayAligned'`

### Outcome

- Replaced `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py` with `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.go`.
- Added `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix_internal_test.go` to cover the stable report summary, scenario ids, and Go-path normalization.
- Updated takeover docs and regression references to point at the Go entrypoint.
- Regenerated `bigclaw-go/docs/reports/multi-subscriber-takeover-validation-report.json` from the Go entrypoint.

### Python File Count Impact

- Repository Python files before this slice: `108`
- Repository Python files after this slice: `107`
- `bigclaw-go/scripts/e2e/**` Python files before this slice: `7`
- `bigclaw-go/scripts/e2e/**` Python files after this slice: `6`
- Net reduction across this issue so far: `9`
- Net reduction in this slice: `1`

### Validation Record

- `cd bigclaw-go && go test ./scripts/e2e/subscriber_takeover_fault_matrix.go ./scripts/e2e/subscriber_takeover_fault_matrix_internal_test.go`
  - Result: `ok  	command-line-arguments	0.451s`
- `cd bigclaw-go && go run ./scripts/e2e/subscriber_takeover_fault_matrix.go --output bigclaw-go/docs/reports/multi-subscriber-takeover-validation-report.json`
  - Result: exit code `0`
- `cd bigclaw-go && go test ./internal/regression -run 'TestLane8CrossProcessCoordinationSurfaceStaysAligned|TestLane8FollowupDigestsStayAligned|TestCrossProcessCoordinationReadinessDocsStayAligned'`
  - Result: `ok  	bigclaw-go/internal/regression	0.525s`
- `python3 - <<'PY' ...`
  - Result: repository Python file count `107`; `bigclaw-go/scripts/e2e/**` Python file count `6`.

## Continuation Slice: broker_failover_stub_matrix

### Scope

- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py`

Replacement path for this slice:

- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.go`

Current repository Python file count before this slice: `107`
Current `bigclaw-go/scripts/e2e/**` Python file count before this slice: `6`

### Plan

1. Replace the deterministic broker-failover stub generator with a Go-native entrypoint that preserves the checked-in report, proof-summary, and artifact contract.
2. Rewrite the existing Go tests to validate the Go-native helpers directly instead of importing the Python module.
3. Update docs and backlog references from the Python path to the Go path.
4. Regenerate the checked-in broker failover report surfaces, record the Python-count deltas, then commit and push the scoped slice.

### Acceptance

- Remove `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py` from the remaining `scripts/e2e` Python backlog by replacing it with `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.go`.
- Keep `docs/reports/broker-failover-stub-report.json`, `docs/reports/broker-checkpoint-fencing-proof-summary.json`, and `docs/reports/broker-retention-boundary-proof-summary.json` aligned with the new Go generator.
- Record exact targeted validation commands and before/after Python counts for both the repo and `bigclaw-go/scripts/e2e/**`.

### Validation

- `cd bigclaw-go && go test ./scripts/e2e/broker_failover_stub_matrix.go ./scripts/e2e/broker_failover_stub_matrix_internal_test.go`
- `cd bigclaw-go && go run ./scripts/e2e/broker_failover_stub_matrix.go --output bigclaw-go/docs/reports/broker-failover-stub-report.json --artifact-root bigclaw-go/docs/reports/broker-failover-stub-artifacts --checkpoint-fencing-summary-output bigclaw-go/docs/reports/broker-checkpoint-fencing-proof-summary.json --retention-boundary-summary-output bigclaw-go/docs/reports/broker-retention-boundary-proof-summary.json`
- `python3 - <<'PY' ...`

### Outcome

- Replaced `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py` with `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.go`.
- Reworked `bigclaw-go/scripts/e2e/broker_failover_stub_matrix_internal_test.go` to validate the Go-native helpers directly instead of importing the Python module.
- Updated broker failover docs and backlog references to point at the Go entrypoint.
- Regenerated `bigclaw-go/docs/reports/broker-failover-stub-report.json`, `bigclaw-go/docs/reports/broker-checkpoint-fencing-proof-summary.json`, `bigclaw-go/docs/reports/broker-retention-boundary-proof-summary.json`, and the checked-in per-scenario raw artifact files from the Go entrypoint.

### Python File Count Impact

- Repository Python files before this slice: `107`
- Repository Python files after this slice: `106`
- `bigclaw-go/scripts/e2e/**` Python files before this slice: `6`
- `bigclaw-go/scripts/e2e/**` Python files after this slice: `5`
- Net reduction across this issue so far: `10`
- Net reduction in this slice: `1`

### Validation Record

- `cd bigclaw-go && go test ./scripts/e2e/broker_failover_stub_matrix.go ./scripts/e2e/broker_failover_stub_matrix_internal_test.go`
  - Result: `ok  	command-line-arguments	1.524s`
- `cd bigclaw-go && go run ./scripts/e2e/broker_failover_stub_matrix.go --output bigclaw-go/docs/reports/broker-failover-stub-report.json --artifact-root bigclaw-go/docs/reports/broker-failover-stub-artifacts --checkpoint-fencing-summary-output bigclaw-go/docs/reports/broker-checkpoint-fencing-proof-summary.json --retention-boundary-summary-output bigclaw-go/docs/reports/broker-retention-boundary-proof-summary.json`
  - Result: exit code `0`
- `cd bigclaw-go && go test ./internal/regression -run 'TestDurabilityRolloutProofSummariesStayAligned|TestDurabilityReviewBundleStaysAligned'`
  - Result: `ok  	bigclaw-go/internal/regression	1.286s`
- `python3 - <<'PY' ...`
  - Result: repository Python file count `106`; `bigclaw-go/scripts/e2e/**` Python file count `5`.

## Continuation Slice: export_validation_bundle

### Scope

- `bigclaw-go/scripts/e2e/export_validation_bundle.py`
- `bigclaw-go/scripts/e2e/export_validation_bundle_test.py`

Replacement paths for this slice:

- `bigclaw-go/scripts/e2e/export_validation_bundle.go`
- `bigclaw-go/scripts/e2e/export_validation_bundle_internal_test.go`

Current repository Python file count before this slice: `106`
Current `bigclaw-go/scripts/e2e/**` Python file count before this slice: `5`

### Plan

1. Replace the validation-bundle exporter with a Go-native entrypoint that preserves the bundle summary, manifest, README/index, broker summary, and shared-queue companion behavior.
2. Port the existing Python unit coverage to Go around broker summary handling, component failure extraction, validation-matrix rows, and rendered index text.
3. Switch `scripts/e2e/run_all.sh` and its harness test to invoke the Go exporter path.
4. Record targeted validation commands and Python-count deltas, then commit and push the scoped slice.

### Acceptance

- Remove `bigclaw-go/scripts/e2e/export_validation_bundle.py` and `bigclaw-go/scripts/e2e/export_validation_bundle_test.py` from the remaining `scripts/e2e` Python backlog by replacing them with Go-native files.
- Keep `scripts/e2e/run_all.sh` bundle export behavior aligned with the Go exporter.
- Record exact targeted validation commands and before/after Python counts for both the repo and `bigclaw-go/scripts/e2e/**`.

### Validation

- `cd bigclaw-go && go test ./scripts/e2e/export_validation_bundle.go ./scripts/e2e/export_validation_bundle_internal_test.go`
- `cd bigclaw-go && go test ./scripts/e2e/run_all_internal_test.go`
- `cd bigclaw-go && rg -n "export_validation_bundle\\.py|export_validation_bundle\\.go" README.md docs/go-cli-script-migration.md scripts/e2e/run_all.sh scripts/e2e/run_all_internal_test.go`
- `python3 - <<'PY' ...`

### Outcome

- Replaced `bigclaw-go/scripts/e2e/export_validation_bundle.py` with `bigclaw-go/scripts/e2e/export_validation_bundle.go`.
- Replaced `bigclaw-go/scripts/e2e/export_validation_bundle_test.py` with `bigclaw-go/scripts/e2e/export_validation_bundle_internal_test.go`.
- Switched `bigclaw-go/scripts/e2e/run_all.sh` and its harness test to invoke the Go exporter directly.
- Updated `bigclaw-go/README.md` and `bigclaw-go/docs/go-cli-script-migration.md` to point at the Go-native exporter and removed the legacy Python exporter from the remaining backlog list.

### Python File Count Impact

- Repository Python files before this slice: `13`
- Repository Python files after this slice: `11`
- `bigclaw-go/scripts/e2e/**` Python files before this slice: `5`
- `bigclaw-go/scripts/e2e/**` Python files after this slice: `3`
- Net reduction across this issue so far: `12`
- Net reduction in this slice: `2`

### Validation Record

- `cd bigclaw-go && go test ./scripts/e2e/export_validation_bundle.go ./scripts/e2e/export_validation_bundle_internal_test.go`
  - Result: `ok  	command-line-arguments	1.466s`
- `cd bigclaw-go && go test ./scripts/e2e/run_all_internal_test.go`
  - Result: `ok  	command-line-arguments	11.421s`
- `cd bigclaw-go && rg -n "export_validation_bundle\\.py|export_validation_bundle\\.go" README.md docs/go-cli-script-migration.md scripts/e2e/run_all.sh scripts/e2e/run_all_internal_test.go`
  - Result: `scripts/e2e/run_all.sh`, `scripts/e2e/run_all_internal_test.go`, `README.md`, and `docs/go-cli-script-migration.md` now reference `export_validation_bundle.go`; the only remaining `.py` hit is the migration table row documenting the legacy-to-Go replacement.
- `cd bigclaw-go && find . -name '*.py' | wc -l`
  - Result: `11`
- `cd bigclaw-go && find scripts/e2e -name '*.py' | wc -l`
  - Result: `3`
