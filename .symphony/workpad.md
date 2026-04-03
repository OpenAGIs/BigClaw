# BIG-GO-1113

## Plan
- baseline the current worktree to confirm whether the lane's candidate `src/bigclaw` Python files still physically exist
- document the exact lane file list for this issue and the current repository Python-file baseline
- add regression coverage for lane-owned candidate paths that are already deleted but are not yet explicitly guarded by Go-side purge tests
- run targeted validation for repo Python-file count, lane path absence, and the affected regression package
- commit the scoped closeout and push the branch

## Acceptance
- lane coverage is explicit for `src/bigclaw/__init__.py`, `src/bigclaw/__main__.py`, `src/bigclaw/audit_events.py`, `src/bigclaw/collaboration.py`, `src/bigclaw/connectors.py`, `src/bigclaw/console_ia.py`, `src/bigclaw/cost_control.py`, `src/bigclaw/dashboard_run_contract.py`, `src/bigclaw/design_system.py`, `src/bigclaw/dsl.py`, `src/bigclaw/evaluation.py`, and `src/bigclaw/event_bus.py`
- the current worktree fact is captured: those candidate Python files are already absent and `find . -name '*.py' | wc -l` is `0`
- the change is scoped to this issue and improves enforceable protection by adding regression coverage for the candidate paths that did not yet have explicit purge assertions
- exact validation commands and outcomes are recorded below

## Validation
- `find . -name '*.py' | wc -l`
- `find . -path '*/src/bigclaw/*.py' | sort`
- `rg -n "src/bigclaw/(__init__|__main__|audit_events|collaboration|connectors|console_ia|cost_control|dashboard_run_contract|design_system|dsl|evaluation|event_bus)\\.py" bigclaw-go/internal/regression`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche(1|4|9|12|13|15)$'`
- `git status --short`

## Validation Results
- `find . -name '*.py' | wc -l` -> `0`
- `find . -path '*/src/bigclaw/*.py' | sort` -> no output
- `rg -n "src/bigclaw/(__init__|__main__|audit_events|collaboration|connectors|console_ia|cost_control|dashboard_run_contract|design_system|dsl|evaluation|event_bus)\\.py" bigclaw-go/internal/regression` -> exit `0`; matches now come from `top_level_module_purge_tranche1_test.go`, `top_level_module_purge_tranche4_test.go`, `top_level_module_purge_tranche9_test.go`, `top_level_module_purge_tranche12_test.go`, `top_level_module_purge_tranche13_test.go`, and the new `top_level_module_purge_tranche15_test.go`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche(1|4|9|12|13|15)$'` -> `ok   bigclaw-go/internal/regression 0.796s`
- `git status --short` -> modified `.symphony/workpad.md`; added `bigclaw-go/internal/regression/top_level_module_purge_tranche15_test.go`
