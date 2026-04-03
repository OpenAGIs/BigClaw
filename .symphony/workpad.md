# BIG-GO-1131

## Plan
- confirm which `src/bigclaw` candidates from the issue context are not yet enforced by existing regression tranches
- preserve the repo's current zero-`.py` baseline and record that the acceptance target is already numerically saturated in this workspace
- add one scoped regression tranche for the remaining residual surfaces: `audit_events.py`, `collaboration.py`, `console_ia.py`, `design_system.py`, `evaluation.py`, `run_detail.py`, `runtime.py`, and the package entrypoints retired under `src/bigclaw/__init__.py` and `src/bigclaw/__main__.py`
- update the cutover handoff note so those residual Python surfaces point at concrete Go owners instead of staying implied backlog-only items
- run targeted validation for the new regression tranche, the full regression package, and repo-wide Python file count checks
- commit and push the scoped change set

## Acceptance
- the BIG-GO-1131 residual candidate set is explicitly covered by regression and handoff evidence
- the worktree continues to contain no live `.py` files
- the new tranche proves compatible Go ownership exists for the retired residual surfaces
- exact validation commands and outcomes are recorded below
- residual risk explicitly records that `find . -name '*.py' | wc -l` already starts at `0` in this workspace, so the count cannot decrease further from the checked-out baseline

## Validation
- `find . -name '*.py' | wc -l`
- `git ls-tree -r --name-only HEAD | rg '\.py$' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run TestTopLevelModulePurgeTranche17`
- `cd bigclaw-go && go test ./internal/regression`
- `git status --short`

## Validation Results
- `find . -name '*.py' | wc -l` -> `0`
- `git ls-tree -r --name-only HEAD | rg '\.py$' | wc -l` -> `0`
- `cd bigclaw-go && go test ./internal/regression -run TestTopLevelModulePurgeTranche17` -> `ok  	bigclaw-go/internal/regression	0.443s`
- `cd bigclaw-go && go test ./internal/regression` -> `ok  	bigclaw-go/internal/regression	0.837s`
- `git status --short` -> modified `.symphony/workpad.md`, `bigclaw-go/internal/regression/top_level_module_purge_tranche16_test.go`; added `bigclaw-go/internal/regression/top_level_module_purge_tranche17_test.go`; modified `docs/go-mainline-cutover-handoff.md`

## Residual Risk
- the repository already materialized to a zero-`.py` baseline before this change, so BIG-GO-1131 can harden deletion enforcement and Go ownership evidence but cannot make the Python file count numerically lower in this checkout
