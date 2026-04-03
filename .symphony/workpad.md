# BIG-GO-1136

## Plan
- confirm the lane-owned Python candidate paths from the issue context against the actual worktree baseline
- record the pre-change zero-`.py` state so the acceptance constraint is explicit for this lane
- add scoped regression coverage that keeps the candidate Python entrypoints deleted and verifies the active Go or shell replacement surfaces still exist
- run targeted validation for the new regression tranche, the broader regression package, and repo Python-file counts
- commit and push the issue-scoped change set to a remote branch

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
- each listed Python path stays absent from the repo worktree
- the Go or shell replacement surfaces for those entrypoints still exist
- exact validation commands and outcomes are recorded below
- residual risk explicitly notes that this workspace already starts at `find . -name '*.py' | wc -l == 0`, so the issue can harden the zero baseline but cannot reduce it numerically further from this branch state

## Validation
- `find . -name '*.py' | wc -l`
- `git ls-tree -r --name-only HEAD | rg '\.py$'`
- `cd bigclaw-go && go test ./internal/regression -run TestPhysicalPythonResidualSweep6`
- `cd bigclaw-go && go test ./internal/regression`
- `git status --short`

## Validation Results
- `find . -name '*.py' | wc -l` -> `0`
- `git ls-tree -r --name-only HEAD | rg '\.py$'` -> exit `1` with no tracked Python files
- `cd bigclaw-go && go test ./internal/regression -run TestPhysicalPythonResidualSweep6` -> `ok  	bigclaw-go/internal/regression	0.848s`
- `cd bigclaw-go && go test ./internal/regression` -> `ok  	bigclaw-go/internal/regression	0.470s`
- `git status --short` -> modified `.symphony/workpad.md`; added `bigclaw-go/internal/regression/physical_python_residual_sweep6_test.go`

## Residual Risk
- the repo already starts from a zero-`.py` baseline in this workspace, so this issue can only harden the zero baseline and replacement-path coverage; it cannot make the Python file count numerically lower from the current branch state
