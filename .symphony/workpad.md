# BIG-GO-1110

## Plan
- inventory the live `src/bigclaw/*.py` surface and keep only the frozen Python compatibility shim that still has a Go compile-check contract
- remove the residual top-level Python product/domain modules now superseded by `bigclaw-go/internal/...`
- add a regression tranche that locks the deleted Python file set and asserts representative Go replacements remain present
- run targeted repository-count, grep, and Go regression/legacy-shim validation
- commit and push the scoped change set to the issue branch

## Acceptance
- lane coverage is explicit: all live residual Python files under `src/bigclaw` are accounted for, with only `src/bigclaw/legacy_shim.py` retained
- real Python assets are physically removed, not just documented, and `find . -name '*.py' | wc -l` decreases from the pre-change baseline
- Go regression coverage prevents the deleted tranche from reappearing
- exact validation commands and results are recorded here

## Validation
- `find . -name '*.py' | wc -l`
- `find src/bigclaw -maxdepth 1 -type f -name '*.py' | sort`
- `rg --files src/bigclaw`
- `cd bigclaw-go && go test ./internal/regression ./internal/legacyshim ./cmd/bigclawctl`
- `bash scripts/ops/bigclawctl legacy-python compile-check --json`
- `git status --short`

## Validation Results
- pre-change inventory: `find . -name '*.py' | wc -l` -> `17`
- `find . -name '*.py' | wc -l` -> `1`
- `find src/bigclaw -maxdepth 1 -type f -name '*.py' | sort` -> `src/bigclaw/legacy_shim.py`
- `rg --files src/bigclaw` -> `src/bigclaw/legacy_shim.py`
- `cd bigclaw-go && go test ./internal/regression ./internal/legacyshim ./cmd/bigclawctl` -> `ok   bigclaw-go/internal/regression 0.431s`; `ok   bigclaw-go/internal/legacyshim 0.790s`; `ok   bigclaw-go/cmd/bigclawctl 3.881s`
- `bash scripts/ops/bigclawctl legacy-python compile-check --json` -> exit `0`; JSON reported `status: ok`, `python: python3`, and the single checked file `/Users/openagi/code/bigclaw-workspaces/BIG-GO-1110/src/bigclaw/legacy_shim.py`
- `git status --short` -> modified `.symphony/workpad.md`, `README.md`, `bigclaw-go/internal/planning/planning.go`, `bigclaw-go/internal/planning/planning_test.go`, `docs/go-domain-intake-parity-matrix.md`, `docs/go-mainline-cutover-handoff.md`; deleted `src/bigclaw/{audit_events,collaboration,console_ia,deprecation,design_system,evaluation,governance,models,observability,operations,planning,reports,risk,run_detail,runtime,ui_review}.py`; added `bigclaw-go/internal/regression/top_level_module_purge_tranche14_test.go`
