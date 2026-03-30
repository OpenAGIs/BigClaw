# BIG-GO-1010 Workpad

## Scope

Final sweep for Python residue under:

- `bigclaw-go/scripts/e2e/**`
- `migration/**` related backlog verification

Observed targeted Python files at start of lane:

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

Verification at start of lane:

- `migration/**` has no remaining Python files in this checkout.
- Repository Python file count before changes: `108`
- Targeted batch Python file count before changes: `14`

## Plan

1. Confirm which targeted files are still authoritative generators versus redundant Python-only tests.
2. Replace redundant Python test coverage with Go-native tests where coverage must remain, then delete the Python test files.
3. Update migration/gap-report documentation to reflect the real remaining Python backlog and the rationale for each keep/delete decision.
4. Run targeted validation and record exact commands plus results.
5. Commit and push the scoped `BIG-GO-1010` change set.

## Acceptance

- Produce the exact residual Python file list for this batch.
- Reduce the targeted Python file count where direct removal is safe.
- Document delete/replace/keep rationale for every targeted Python file.
- Report total repository Python count impact after the sweep.

## Validation

- `find . -name '*.py' | wc -l`
- Targeted `go test` commands covering any migrated test paths
- `git status --short`
- `git log -1 --stat`

## Notes

- `bigclaw-go/cmd/bigclawctl automation e2e` currently exposes only `run-task-smoke`; no Go CLI replacements exist yet for the remaining report generators.
- Existing Go regression tests already pin many checked-in report artifacts, which makes Python unit-test deletion viable if equivalent or stronger Go coverage remains.

## Results

### File Disposition

- Deleted:
  - `bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py`
  - `bigclaw-go/scripts/e2e/export_validation_bundle_test.py`
  - `bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py`
  - `bigclaw-go/scripts/e2e/run_all_test.py`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`
- Added Go replacements:
  - `bigclaw-go/scripts/e2e/run_all_test.go`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.go`
- Kept as remaining Go-only gap:
  - `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py`
  - `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py`
  - `bigclaw-go/scripts/e2e/export_validation_bundle.py`
  - `bigclaw-go/scripts/e2e/external_store_validation.py`
  - `bigclaw-go/scripts/e2e/mixed_workload_matrix.py`
  - `bigclaw-go/scripts/e2e/multi_node_shared_queue.py`
  - `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`

### Count Impact

- Repository Python files before: `108`
- Repository Python files after: `103`
- Targeted batch Python files before: `14`
- Targeted batch Python files after: `9`
- Net reduction: `5`

### Validation Record

- `gofmt -w bigclaw-go/scripts/e2e/run_all_test.go bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.go`
  - Result: success
- `cd bigclaw-go && go test ./scripts/e2e`
  - Result: `ok  	bigclaw-go/scripts/e2e	3.673s`
- `cd bigclaw-go && go test ./internal/regression -run 'TestLane8ValidationBundleContinuationScorecardStaysAligned|TestLane8SubscriberTakeoverHarnessStaysAligned|TestLiveValidation(Index|Summary)|TestSharedQueueReport|TestSharedQueueCompanionSummary|TestBrokerValidationSummary'`
  - Result: `ok  	bigclaw-go/internal/regression	1.279s`
- `cd bigclaw-go && find .. -name '*.py' | wc -l && find scripts/e2e -name '*.py' | wc -l`
  - Result:
    - repository count: `103`
    - `scripts/e2e` count: `9`
