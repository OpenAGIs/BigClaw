# BIG-GO-1010 Workpad

## Plan

1. Verify the actual Python residue under `bigclaw-go/scripts/e2e/**` and confirm whether any `migration/**` Python files still exist in this checkout.
2. Cross-check the surviving scripts against Go tests, regression coverage, and repo call sites so the final gap report distinguishes removed coverage from still-blocking generators.
3. Refresh the issue-facing documentation with the verified residual list, delete/replace/keep rationale, and Python file-count impact.
4. Run targeted validation and record the exact commands and outcomes.
5. Commit and push the scoped documentation/report update.

## Acceptance

- Produce the exact residual Python file list for this batch.
- State whether `migration/**` still contains Python in the current checkout.
- Document delete/replace/keep rationale for the batch files covered by `BIG-GO-1010`.
- Report the repository-wide Python count impact and the remaining `scripts/e2e` Python count.

## Validation

- `find . -name '*.py' | wc -l`
- `find bigclaw-go/scripts/e2e -name '*.py' | wc -l`
- `rg --files | rg '(^|/)migration/|/migration/'`
- `cd bigclaw-go && go test ./scripts/e2e ./internal/regression`
- `git status --short`
- `git log --oneline -1`

## Baseline

- Current residual Python files under `bigclaw-go/scripts/e2e/**`:
  - `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py`
  - `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py`
  - `bigclaw-go/scripts/e2e/export_validation_bundle.py`
  - `bigclaw-go/scripts/e2e/external_store_validation.py`
  - `bigclaw-go/scripts/e2e/mixed_workload_matrix.py`
  - `bigclaw-go/scripts/e2e/multi_node_shared_queue.py`
  - `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`
- `migration/**` does not exist in this checkout, so there are no remaining Python files there.
- Current repository Python file count: `103`
- Current `bigclaw-go/scripts/e2e/**` Python file count: `9`
- Historical `BIG-GO-1010` removal commit confirmed by `git log`: `76a14bc feat: finalize BIG-GO-1010 python gap sweep`

## File Disposition

- Removed earlier in `BIG-GO-1010` and confirmed via git history:
  - `bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py`
  - `bigclaw-go/scripts/e2e/export_validation_bundle_test.py`
  - `bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py`
  - `bigclaw-go/scripts/e2e/run_all_test.py`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`
- Go-native replacement coverage retained:
  - `bigclaw-go/scripts/e2e/run_all_test.go`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.go`
- Still blocked on Go-native implementation:
  - `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py`
  - `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py`
  - `bigclaw-go/scripts/e2e/export_validation_bundle.py`
  - `bigclaw-go/scripts/e2e/external_store_validation.py`
  - `bigclaw-go/scripts/e2e/mixed_workload_matrix.py`
  - `bigclaw-go/scripts/e2e/multi_node_shared_queue.py`
  - `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`

## Results

- Residual Python list for the batch is verified and reduced to the nine generators above.
- `migration/**` contributes zero Python files in the current checkout.
- Historical impact of `BIG-GO-1010` remains:
  - repository Python files `108 -> 103`
  - targeted batch Python files `14 -> 9`
  - net reduction `5`
- No additional safe Python deletions remain in the scoped batch without implementing new Go-native report generators.

## Validation Results

- `find . -name '*.py' | wc -l`
  - Result: `103`
- `find bigclaw-go/scripts/e2e -name '*.py' | wc -l`
  - Result: `9`
- `rg --files | rg '(^|/)migration/|/migration/'`
  - Result: no matches
- `cd bigclaw-go && go test ./scripts/e2e ./internal/regression`
  - Result: `ok   bigclaw-go/scripts/e2e  4.224s`
  - Result: `ok   bigclaw-go/internal/regression  1.186s`
