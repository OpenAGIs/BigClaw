# BIG-GO-1126

## Plan
- confirm the BIG-GO-1126 candidate Python entrypoints are already absent in this materialized worktree
- record the pre-change zero-`.py` repository baseline so the acceptance constraint is explicit
- add a scoped regression test that locks the candidate entrypoints to absent-on-disk plus Go or retained shell compatibility owners present
- keep the change set limited to BIG-GO-1126 enforcement and validation evidence
- run targeted validation and record the exact commands and outcomes here before commit/push

## Acceptance
- lane coverage is explicit for:
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
- `scripts/create_issues.py`
- `scripts/dev_smoke.py`
- regression coverage verifies those Python entrypoints stay deleted while their Go or retained shell owners still exist
- exact validation commands and outcomes are recorded below
- residual risk explicitly notes that this checkout already starts at `find . -name '*.py' | wc -l -> 0`, so this lane can only harden the zero-Python state instead of lowering the count numerically

## Validation
- `find . -name '*.py' | wc -l`
- `git ls-tree -r --name-only HEAD | rg '\.py$'`
- `cd bigclaw-go && go test ./internal/regression -run TestBIGGO1126ScriptMigrationSurface`
- `cd bigclaw-go && go test ./internal/regression`
- `git status --short`

## Validation Results
- `find . -name '*.py' | wc -l` -> `0`
- `git ls-tree -r --name-only HEAD | rg '\.py$'` -> exit `1` with no tracked Python files
- `cd bigclaw-go && go test ./internal/regression -run TestBIGGO1126ScriptMigrationSurface` -> `ok  	bigclaw-go/internal/regression	0.479s`
- `cd bigclaw-go && go test ./internal/regression` -> `ok  	bigclaw-go/internal/regression	0.289s`
- `git status --short` -> modified `.symphony/workpad.md`; added `bigclaw-go/internal/regression/big_go_1126_script_migration_surface_test.go`

## Residual Risk
- the repo already materialized with zero `.py` files in this workspace, so the issue acceptance item about decreasing the Python count cannot move numerically lower from the observed baseline
