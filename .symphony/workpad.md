# BIG-GO-1121

## Plan
- confirm the materialized workspace baseline for real Python files and record the zero-`.py` starting state explicitly
- inventory existing `top_level_module_purge_tranche*` regression coverage against the issue candidate list
- add one scoped regression tranche for the candidate modules still not explicitly covered by the existing purge tests
- validate the Go replacement paths for the newly covered modules using already-materialized Go ownership files
- run targeted validation plus repo Python-count checks and record exact commands and outcomes
- commit and push the scoped change set

## Acceptance
- cover the issue-owned candidate modules that were still missing explicit purge-tranche enforcement in this workspace:
- `src/bigclaw/__init__.py`
- `src/bigclaw/__main__.py`
- `src/bigclaw/audit_events.py`
- `src/bigclaw/collaboration.py`
- `src/bigclaw/console_ia.py`
- `src/bigclaw/design_system.py`
- `src/bigclaw/evaluation.py`
- `src/bigclaw/runtime.py`
- verify a Go replacement or compatibility owner exists for each newly covered module
- keep the change scoped to regression enforcement for `BIG-GO-1121`
- record the actual Python baseline and resulting count after the change
- exact validation commands and outcomes are recorded below
- residual risk explicitly notes that the workspace already starts at `0` real `.py` files, so the numeric file count cannot decrease further here

## Validation
- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run TestTopLevelModulePurgeTranche16`
- `cd bigclaw-go && go test ./internal/regression`
- `git status --short`

## Validation Results
- `find . -name '*.py' | wc -l` -> `0`
- `git ls-tree -r --name-only HEAD | rg '\.py$' | wc -l` -> `0`
- `cd bigclaw-go && go test ./internal/regression -run TestTopLevelModulePurgeTranche16` -> `ok   bigclaw-go/internal/regression (cached)`
- `cd bigclaw-go && go test ./internal/regression` -> `ok   bigclaw-go/internal/regression (cached)`
- `git status --short` before commit -> `M bigclaw-go/internal/regression/top_level_module_purge_tranche16_test.go`

## Residual Risk
- this materialized workspace already starts from `0` real `.py` files, so `BIG-GO-1121` can only harden deletion enforcement and Go replacement verification; it cannot make the Python file count numerically lower from the current baseline
