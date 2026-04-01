# BIG-GO-1061 Workpad

## Scope
- Residual sweep for `src/bigclaw` Python assets in the issue's suggested tranche.
- Physical targets still present in this checkout from the suggested list: `src/bigclaw/__init__.py`, `src/bigclaw/__main__.py`.
- Adjacent residual helper likely removable within the same compatibility slice: `src/bigclaw/deprecation.py`.
- Already absent from the suggested list in the current checkout and therefore out of edit scope unless encountered indirectly: `audit_events.py`, `collaboration.py`, `connectors.py`, `console_ia.py`, `cost_control.py`, `dashboard_run_contract.py`, `design_system.py`, `dsl.py`, `evaluation.py`, `event_bus.py`, `execution_contract.py`, `github_sync.py`, `governance.py`.
- Current Go validation drift to fix during this tranche: `bigclaw-go/internal/legacyshim/compilecheck.go` and related tests still reference deleted `src/bigclaw/service.py`.

## Plan
1. Refresh this workpad to match the actual residual Python asset list and stale validation surfaces in the checkout.
2. Delete `src/bigclaw/__main__.py` if no live validation or import path requires it, preserving Go-first replacement guidance in docs instead of Python code.
3. Inline the tiny deprecation helper into a surviving module if that safely removes `src/bigclaw/deprecation.py`.
4. Update Go compile-check coverage, regression tests, and current compatibility docs/manifest to the reduced shim list.
5. Run targeted Python and Go regression checks, measure Python file-count delta, then commit and push the issue branch.

## Acceptance
- Identify the concrete Python asset list handled in this tranche and separate it from already-absent files.
- Remove, replace, or downgrade redundant residual Python files while preserving the remaining package compatibility surface.
- Keep a verifiable Go replacement path or migration note for each removed entry or shim.
- Record exact validation commands and outcomes.
- Report Python file-count impact and any residual risk.

## Validation
- `PYTHONPATH=src python3 -m pytest tests/test_top_level_module_shims.py tests/test_repo_collaboration.py tests/test_observability.py tests/test_planning.py tests/test_evaluation.py tests/test_operations.py tests/test_design_system.py tests/test_console_ia.py -q`
- `(cd bigclaw-go && go test ./internal/legacyshim ./internal/regression ./cmd/bigclawctl -count=1)`
- `(cd bigclaw-go && go test ./internal/regression -run TestTopLevelModulePurgeTranche14 -count=1)`
- `bash scripts/ops/bigclawctl legacy-python compile-check --json`
- `bash scripts/ops/bigclawctl github-sync status --json`
- `find . -name '*.py' | wc -l`
- `git status --short`

## Validation Results
- `PYTHONPATH=src python3 -m pytest tests/test_top_level_module_shims.py tests/test_repo_collaboration.py tests/test_observability.py tests/test_planning.py tests/test_evaluation.py tests/test_operations.py tests/test_design_system.py tests/test_console_ia.py -q` -> `76 passed in 0.28s`
- `(cd bigclaw-go && go test ./internal/legacyshim ./internal/regression ./cmd/bigclawctl -count=1)` -> `ok  	bigclaw-go/internal/legacyshim	1.316s`, `ok  	bigclaw-go/internal/regression	1.569s`, `ok  	bigclaw-go/cmd/bigclawctl	4.678s`
- `(cd bigclaw-go && go test ./internal/regression -run TestTopLevelModulePurgeTranche14 -count=1)` -> `ok  	bigclaw-go/internal/regression	1.893s`
- `bash scripts/ops/bigclawctl legacy-python compile-check --json` -> `status: ok`; files: `src/bigclaw/__init__.py`, `src/bigclaw/legacy_shim.py`, `src/bigclaw/runtime.py`
- `bash scripts/ops/bigclawctl github-sync status --json` -> `branch: big-go-1061-residual-sweep`, `local_sha: 9df504144ab1e1dc0ca026e6992b6b6459a56b73`, `remote_sha: 9df504144ab1e1dc0ca026e6992b6b6459a56b73`, `status: ok`, `synced: true`
- `find . -name '*.py' | wc -l` -> `38` (pre-change baseline: `40`, net `-2`)
- `rg --files src/bigclaw | rg '\.py$' | wc -l` -> `11` (pre-change baseline: `13`, net `-2`)
- `git status --short` -> clean after pushing `origin/big-go-1061-residual-sweep`
