# BIG-GO-1061 Workpad

## Scope
- Residual sweep for `src/bigclaw` Python assets in the issue's suggested tranche.
- Current physical targets present in this checkout: `src/bigclaw/__init__.py`, `src/bigclaw/__main__.py`, `src/bigclaw/audit_events.py`, `src/bigclaw/collaboration.py`, `src/bigclaw/console_ia.py`, `src/bigclaw/design_system.py`, `src/bigclaw/evaluation.py`, `src/bigclaw/governance.py`.
- Missing from the suggested list in the current checkout and therefore out of edit scope unless encountered indirectly: `connectors.py`, `cost_control.py`, `dashboard_run_contract.py`, `dsl.py`, `event_bus.py`, `execution_contract.py`, `github_sync.py`.

## Plan
1. Audit residual module usage and existing Go replacement coverage.
2. Collapse low-complexity Python modules into package-level compatibility shims where import paths can stay stable.
3. Delete physical Python files that become redundant.
4. Run targeted Python and Go regression checks for the affected surfaces.
5. Measure Python file-count delta, then commit and push the issue branch.

## Acceptance
- Identify the concrete Python asset list handled in this tranche.
- Remove, replace, or downgrade as many residual Python files as is safe while preserving import compatibility.
- Keep a verifiable Go replacement path or migration note for each removed surface.
- Record exact validation commands and outcomes.
- Report Python file-count impact and any residual risk.

## Validation
- `PYTHONPATH=src python3 -m pytest tests/test_top_level_module_shims.py tests/test_repo_collaboration.py tests/test_observability.py tests/test_planning.py tests/test_evaluation.py -q`
- `(cd bigclaw-go && go test ./internal/governance ./internal/events ./internal/regression -count=1)`
- `find . -name '*.py' | wc -l`
- `git status --short`

## Validation Results
- `PYTHONPATH=src python3 -m pytest tests/test_top_level_module_shims.py tests/test_repo_collaboration.py tests/test_observability.py tests/test_planning.py tests/test_evaluation.py -q` -> `30 passed in 0.11s`
- `(cd bigclaw-go && go test ./internal/governance ./internal/events ./internal/regression -count=1)` -> `ok  	bigclaw-go/internal/governance	0.419s`, `ok  	bigclaw-go/internal/events	0.924s`, `ok  	bigclaw-go/internal/regression	1.126s`
- `find . -name '*.py' | wc -l` -> `43` (pre-change baseline: `45`, net `-2`)
