# BIG-GO-979 Workpad

## Scope

Targeted first migration batch under `bigclaw-go/scripts/e2e/`:

- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`

Replacement paths for this batch:

- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.go`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_internal_test.go`

Current repository Python file count before this lane: `116`
Current `bigclaw-go/scripts/e2e/**` Python file count before this lane: `15`

## Plan

1. Port the continuation policy gate generator from Python to Go with matching CLI behavior and JSON output structure.
2. Recreate the Python unit coverage as Go tests against the Go implementation.
3. Remove the migrated Python script and Python test file.
4. Run targeted Go tests and a targeted generator invocation to validate behavior.
5. Record the exact batch file list, replacement paths, and Python file-count impact.
6. Commit and push the scoped changes for `BIG-GO-979`.

## Acceptance

- Produce the exact `BIG-GO-979` batch file list under `bigclaw-go/scripts/e2e/**`.
- Reduce Python files in the targeted directory by removing the selected batch and replacing it with Go-native paths.
- Keep changes scoped to the continuation policy gate migration batch only.
- Report before/after repository-wide and `bigclaw-go/scripts/e2e/**` Python file counts.

## Validation

- `cd bigclaw-go && go test ./scripts/e2e/validation_bundle_continuation_policy_gate.go ./scripts/e2e/validation_bundle_continuation_policy_gate_internal_test.go`
- `cd bigclaw-go && go run ./scripts/e2e/validation_bundle_continuation_policy_gate.go --scorecard bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json --output bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json`
- `cd bigclaw-go && python3 scripts/e2e/run_all_test.py`
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

### Python File Count Impact

- Repository Python files before: `116`
- Repository Python files after: `114`
- `bigclaw-go/scripts/e2e/**` Python files before: `15`
- `bigclaw-go/scripts/e2e/**` Python files after: `13`
- Net reduction: `2`

### Validation Record

- `cd bigclaw-go && go test ./scripts/e2e/validation_bundle_continuation_policy_gate.go ./scripts/e2e/validation_bundle_continuation_policy_gate_internal_test.go`
  - Result: `ok  	command-line-arguments	1.416s`
- `cd bigclaw-go && python3 scripts/e2e/run_all_test.py`
  - Result: `Ran 3 tests in 7.094s` and `OK`
- `cd bigclaw-go && go test ./internal/regression -run 'TestLane8FollowupDigestsStayAligned'`
  - Result: `ok  	bigclaw-go/internal/regression	(cached)`
- `cd bigclaw-go && go run ./scripts/e2e/validation_bundle_continuation_policy_gate.go --scorecard bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json --output bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json`
  - Result: exit code `0`
- `git status --short`
  - Result: only the scoped `BIG-GO-979` files above were modified before commit.
