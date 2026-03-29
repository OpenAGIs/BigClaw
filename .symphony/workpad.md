# BIG-GO-979 Workpad

## Scope

Targeted continuation migration batch under `bigclaw-go/scripts/e2e/`:

- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`
- `bigclaw-go/scripts/e2e/run_all_test.py`
- `bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py`

Replacement paths for this batch:

- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.go`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_internal_test.go`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.go`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard_internal_test.go`
- `bigclaw-go/scripts/e2e/run_all_internal_test.go`
- `bigclaw-go/scripts/e2e/multi_node_shared_queue_internal_test.go`

Current repository Python file count before this sub-batch: `112`
Current `bigclaw-go/scripts/e2e/**` Python file count before this sub-batch: `11`

## Plan

1. Port the `multi_node_shared_queue.py` lightweight regression coverage from Python to Go while continuing to validate the current Python script behavior.
2. Remove `bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py` after the Go replacement is in place.
3. Re-run the adjacent `run_all` and continuation targeted tests to keep the batch cohesive.
4. Record the updated batch file list, replacement paths, and Python file-count impact.
5. Commit and push the scoped changes for `BIG-GO-979`.

## Acceptance

- Produce the exact `BIG-GO-979` batch file list under `bigclaw-go/scripts/e2e/**`.
- Reduce Python files in the targeted directory by removing the selected shared-queue test batch and replacing it with Go-native paths.
- Keep changes scoped to the validation-bundle continuation migration batch only.
- Report before/after repository-wide and `bigclaw-go/scripts/e2e/**` Python file counts.

## Validation

- `cd bigclaw-go && go test ./scripts/e2e/validation_bundle_continuation_policy_gate.go ./scripts/e2e/validation_bundle_continuation_policy_gate_internal_test.go`
- `cd bigclaw-go && go test ./scripts/e2e/validation_bundle_continuation_scorecard.go ./scripts/e2e/validation_bundle_continuation_scorecard_internal_test.go`
- `cd bigclaw-go && go run ./scripts/e2e/validation_bundle_continuation_scorecard.go --output bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json`
- `cd bigclaw-go && go run ./scripts/e2e/validation_bundle_continuation_policy_gate.go --scorecard bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json --output bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json`
- `cd bigclaw-go && go test ./scripts/e2e/run_all_internal_test.go`
- `cd bigclaw-go && go test ./scripts/e2e/multi_node_shared_queue_internal_test.go`
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

### Python File Count Impact

- Repository Python files before first sub-batch: `116`
- Repository Python files after current sub-batch: `111`
- `bigclaw-go/scripts/e2e/**` Python files before first sub-batch: `15`
- `bigclaw-go/scripts/e2e/**` Python files after current sub-batch: `10`
- Net reduction across this issue so far: `5`
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
  - Result: `ok  	command-line-arguments	1.637s`
- `git status --short`
  - Result: only the scoped `BIG-GO-979` files above were modified before commit.
