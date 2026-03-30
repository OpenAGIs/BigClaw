# BIG-GO-1010 Workpad

## Plan

1. Migrate the continuation scorecard and policy gate generators from Python into `bigclawctl automation e2e` subcommands.
2. Repoint `scripts/e2e/run_all.sh`, script tests, and docs to the new Go-native continuation commands.
3. Delete the two migrated Python generators and refresh the final gap report plus migration docs with the reduced residual list and updated counts.
4. Run targeted validation and record the exact commands and outcomes.
5. Commit and push the scoped `BIG-GO-1010` follow-up change set.

## Acceptance

- Reduce the residual Python file list for this batch by migrating the continuation scorecard and policy gate generators to Go.
- Keep `migration/**` at zero residual Python files in the current checkout.
- Update delete/replace/keep rationale for the batch files covered by `BIG-GO-1010`.
- Report the repository-wide Python count impact and the remaining `scripts/e2e` Python count after the extra migration.

## Validation

- `find . -name '*.py' | wc -l`
- `find bigclaw-go/scripts/e2e -name '*.py' | wc -l`
- `rg --files | rg '(^|/)migration/|/migration/'`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./scripts/e2e ./internal/regression`
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
- `migration/**` does not exist in this checkout, so there are no remaining Python files there.
- Current repository Python file count: `101`
- Current `bigclaw-go/scripts/e2e/**` Python file count: `7`
- Historical `BIG-GO-1010` removal commit confirmed by `git log`: `76a14bc feat: finalize BIG-GO-1010 python gap sweep`
- Extra migration completed in this continuation:
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`

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
  - continuation scorecard and policy gate are now Go-native CLI commands and no longer part of the Python backlog

## Results

- Residual Python list for the batch is verified and reduced to seven generators.
- `migration/**` contributes zero Python files in the current checkout.
- Historical impact of `BIG-GO-1010` remains:
  - repository Python files `108 -> 101`
  - targeted batch Python files `14 -> 7`
  - net reduction `7`
- This continuation migrated:
  - `go run ./cmd/bigclawctl automation e2e validation-bundle-continuation-scorecard`
  - `go run ./cmd/bigclawctl automation e2e validation-bundle-continuation-policy-gate`
- No additional safe Python deletions remain in the scoped batch without implementing new Go-native report generators.

## Validation Results

- `find . -name '*.py' | wc -l`
  - Result: `101`
- `find bigclaw-go/scripts/e2e -name '*.py' | wc -l`
  - Result: `7`
- `rg --files | rg '(^|/)migration/|/migration/'`
  - Result: no matches
- `cd bigclaw-go && go test ./cmd/bigclawctl ./scripts/e2e ./internal/regression`
  - Result: `ok   bigclaw-go/cmd/bigclawctl  (cached)`
  - Result: `ok   bigclaw-go/scripts/e2e  2.901s`
  - Result: `ok   bigclaw-go/internal/regression  (cached)`
