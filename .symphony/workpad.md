# BIG-GO-1112 Workpad

## Plan

1. Inspect the candidate `bigclaw-go/scripts/{benchmark,e2e,migration}` lane and confirm which listed Python files still exist versus which were already removed.
2. Update the migration/regression surface so the repo records the lane as Go-only and fails fast if any of the retired candidate Python entrypoints return.
3. Run targeted validation for the affected regression tests and migration-doc assertions.
4. Commit the scoped changes and push the branch.

## Acceptance Mapping

- Explicit lane file list:
  - `bigclaw-go/scripts/benchmark/capacity_certification.py`
  - `bigclaw-go/scripts/benchmark/capacity_certification_test.py`
  - `bigclaw-go/scripts/benchmark/run_matrix.py`
  - `bigclaw-go/scripts/benchmark/soak_local.py`
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
  - `bigclaw-go/scripts/e2e/run_task_smoke.py`
  - `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`
  - `bigclaw-go/scripts/migration/export_live_shadow_bundle.py`
  - `bigclaw-go/scripts/migration/live_shadow_scorecard.py`
  - `bigclaw-go/scripts/migration/shadow_compare.py`
  - `bigclaw-go/scripts/migration/shadow_matrix.py`
- Python file total should remain at zero for the scoped lane and be guarded by regression checks.
- Validation must capture exact commands and outcomes for the regression/doc updates.

## Validation

- `find bigclaw-go -type f -name '*.py' | sort`
- `cd bigclaw-go && go test ./internal/regression -run 'TestScriptDirectoryStaysPythonFree|TestE2EMigrationDocListsOnlyActiveEntrypoints'`
- `rg -n 'capacity_certification\.py|run_matrix\.py|soak_local\.py|broker_failover_stub_matrix\.py|cross_process_coordination_surface\.py|export_validation_bundle\.py|external_store_validation\.py|mixed_workload_matrix\.py|multi_node_shared_queue\.py|run_task_smoke\.py|subscriber_takeover_fault_matrix\.py|validation_bundle_continuation_policy_gate\.py|validation_bundle_continuation_scorecard\.py|export_live_shadow_bundle\.py|live_shadow_scorecard\.py|shadow_compare\.py|shadow_matrix\.py' bigclaw-go/internal bigclaw-go/docs docs -g '!reports/**'`

## Validation Results

- `find bigclaw-go -type f -name '*.py' | sort` -> no output
- `cd bigclaw-go && go test ./internal/regression -run 'TestE2EScriptDirectoryStaysPythonFree|TestE2EMigrationDocListsOnlyActiveEntrypoints'` -> `ok  	bigclaw-go/internal/regression	0.449s`
- `rg -n 'candidate Python files covered by this lane were:|bigclaw-go/scripts/benchmark/capacity_certification\.py|bigclaw-go/scripts/e2e/export_validation_bundle\.py|bigclaw-go/scripts/migration/shadow_compare\.py|bigclaw-go/scripts/e2e/run_task_smoke\.py' bigclaw-go/docs/go-cli-script-migration.md bigclaw-go/internal/regression/e2e_entrypoint_migration_test.go docs/go-cli-script-migration-plan.md` -> expected matches only in the updated migration docs and regression guard
- `git rev-parse HEAD` -> `2246b090d2b80c554ce81d23841f4321927e9c5a`
- `git rev-parse --abbrev-ref --symbolic-full-name @{u}` -> `origin/symphony/BIG-GO-1112`

## Residual Risk

- This materialized workspace was already at a zero-`.py` baseline for `bigclaw-go`, so this lane could only harden the delete gate and migration record instead of numerically reducing the Python count below zero.
