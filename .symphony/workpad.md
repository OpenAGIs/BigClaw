# BIG-GO-990 Workpad

## Scope

Target the remaining Python files under:

- `bigclaw-go/scripts/e2e/**`
- `migration/**`

Current batch file list:

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
Current targeted batch Python file count before this lane: `14`
Current migration batch Python file count before this lane: `0`

## Plan

1. Confirm which batch files already have repo-native Go replacements or can be routed through existing Go automation commands.
2. Delete redundant Python e2e scripts and tests where Go replacements already exist, then update shell/docs/tests that still reference the Python entrypoints.
3. Run targeted validation for the touched Go automation flows and repository assertions.
4. Record exact disposition, rationale, and Python count deltas for this lane.
5. Commit and push the scoped `BIG-GO-990` changes.

## Acceptance

- Produce the exact `BIG-GO-990` batch file list for `scripts/e2e` and `migration`.
- Reduce the number of Python files in the targeted directories as far as practical within this lane.
- Document keep/replace/delete rationale for every targeted file touched in this lane.
- Report the repository-wide Python file count impact.

## Validation

- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/regression`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e validation-bundle-continuation-scorecard --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e validation-bundle-continuation-policy-gate --help`
- `git status --short`
- `git log -1 --stat`

## Results

### File Disposition

- `migration/**`
  - No files present in the current worktree.
  - Reason: the migration-side Python batch for this issue was already cleared before this lane; no scoped migration edits were required.
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`
  - Deleted.
  - Reason: `go run ./cmd/bigclawctl automation e2e validation-bundle-continuation-scorecard ...` already provides the repo-native replacement and `run_all.sh` now calls it directly.
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
  - Deleted.
  - Reason: `go run ./cmd/bigclawctl automation e2e validation-bundle-continuation-policy-gate ...` already provides the repo-native replacement and `run_all.sh` now calls it directly.
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`
  - Deleted.
  - Reason: it only validated the removed Python entrypoint; coverage now lives in Go CLI tests plus the `run_all.sh` regression harness.
- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py`
  - Kept.
  - Reason: still generates the checked-in broker failover proof pack and no Go entrypoint produces the same scenario matrix/report surface.
- `bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py`
  - Kept.
  - Reason: it validates the retained Python broker failover harness.
- `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py`
  - Kept.
  - Reason: still generates the cross-process coordination capability surface JSON consumed by docs/regression checks; no Go replacement exists yet.
- `bigclaw-go/scripts/e2e/export_validation_bundle.py`
  - Kept.
  - Reason: `run_all.sh` still depends on it to build the live validation bundle/index/README export surface; no Go-native exporter exists yet.
- `bigclaw-go/scripts/e2e/export_validation_bundle_test.py`
  - Kept.
  - Reason: it validates the retained Python bundle exporter.
- `bigclaw-go/scripts/e2e/external_store_validation.py`
  - Kept.
  - Reason: still authors the external-store validation evidence pack and no Go CLI command replaces that report generator yet.
- `bigclaw-go/scripts/e2e/mixed_workload_matrix.py`
  - Kept.
  - Reason: still generates the mixed workload validation matrix and has no repo-native Go replacement.
- `bigclaw-go/scripts/e2e/multi_node_shared_queue.py`
  - Kept.
  - Reason: still generates the shared-queue and live takeover companion proofs referenced by checked-in reports and docs; no Go-native harness exists yet.
- `bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py`
  - Kept.
  - Reason: it validates the retained Python shared-queue harness.
- `bigclaw-go/scripts/e2e/run_all_test.py`
  - Kept.
  - Reason: it regression-tests the retained shell workflow wrapper `run_all.sh`; this is test coverage, not an obsolete automation entrypoint.
- `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`
  - Kept.
  - Reason: still generates the deterministic subscriber takeover harness report and no Go replacement exists yet.

### Python File Count Impact

- Repository Python files before: `108`
- Repository Python files after: `105`
- Targeted batch Python files before: `14`
- Targeted batch Python files after: `11`
- Net reduction: `3`

### Validation Record

- `cd bigclaw-go && python3 scripts/e2e/run_all_test.py`
  - Result: `Ran 3 tests in 4.012s` and `OK`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/regression`
  - Result: `ok  	bigclaw-go/cmd/bigclawctl	2.853s` and `ok  	bigclaw-go/internal/regression	0.763s`
- `cd bigclaw-go && go test ./internal/regression -run 'TestLane8FollowupDigestsStayAligned'`
  - Result: `ok  	bigclaw-go/internal/regression	1.358s`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e validation-bundle-continuation-scorecard --help`
  - Result: usage text printed with the expected continuation scorecard flags.
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e validation-bundle-continuation-policy-gate --help`
  - Result: usage text printed with the expected continuation policy-gate flags.
- `git diff --check`
  - Result: no whitespace or merge-marker errors.
- `find .. -name '*.py' | wc -l`
  - Result: `105`
