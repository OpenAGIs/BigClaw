# BIG-GO-1152

## Plan
- confirm the lane-owned `tests/*.py` candidate list against the actual worktree and record the pre-change Python-file baseline, which is already zero in this workspace
- replace the stale active workpad section with this issue's scoped plan before any code changes
- extend the existing tranche-17 regression coverage with an explicit repo-wide zero-Python baseline assertion and lane-owned Go compatibility checks so this issue still hardens the residual sweep
- validate the regression extension plus repo-wide Python-count checks and Go compatibility commands
- record exact commands and outcomes, then commit and push the scoped change set

## Acceptance
- the lane candidate Python test paths remain explicitly covered and absent from disk:
- `tests/conftest.py`
- `tests/test_audit_events.py`
- `tests/test_connectors.py`
- `tests/test_console_ia.py`
- `tests/test_control_center.py`
- `tests/test_cost_control.py`
- `tests/test_cross_process_coordination_surface.py`
- `tests/test_dashboard_run_contract.py`
- `tests/test_design_system.py`
- `tests/test_dsl.py`
- `tests/test_evaluation.py`
- `tests/test_event_bus.py`
- `tests/test_execution_contract.py`
- `tests/test_execution_flow.py`
- `tests/test_followup_digests.py`
- `tests/test_github_sync.py`
- `tests/test_governance.py`
- `tests/test_issue_archive.py`
- `tests/test_live_shadow_bundle.py`
- `tests/test_live_shadow_scorecard.py`
- `tests/test_mapping.py`
- `tests/test_memory.py`
- `tests/test_models.py`
- `tests/test_observability.py`
- `tests/test_operations.py`
- `tests/test_orchestration.py`
- `tests/test_parallel_refill.py`
- `tests/test_parallel_validation_bundle.py`
- `tests/test_pilot.py`
- `tests/test_planning.py`
- `tests/test_queue.py`
- `tests/test_repo_board.py`
- `tests/test_repo_collaboration.py`
- `tests/test_repo_gateway.py`
- `tests/test_repo_governance.py`
- `tests/test_repo_links.py`
- `tests/test_repo_registry.py`
- `tests/test_repo_rollout.py`
- `tests/test_repo_triage.py`
- `tests/test_reports.py`
- tranche-17 coverage stays intact and this issue adds a separate regression assertion that the repository has zero live or tracked `.py` files for the residual test sweep
- Go replacements or compatibility surfaces are asserted for this lane through concrete repo-native files under `bigclaw-go/internal`, `bigclaw-go/cmd`, and `docs/reports`
- the repository remains at zero live `.py` files in the worktree and zero tracked `.py` files in `HEAD`
- exact validation commands and outcomes are recorded below
- residual risk explicitly notes that this workspace already starts at a zero-`.py` baseline, so the count cannot numerically decrease further here

## Validation
- `find . -name '*.py' | wc -l`
- `git ls-tree -r --name-only HEAD | rg '\.py$'`
- `cd bigclaw-go && go test ./internal/regression -run 'TestPythonTestTranche17Removed|TestPythonResidualSweepRepoBaseline|TestPythonResidualSweepGoCompatibilitySurfaces'`
- `bash scripts/ops/bigclawctl automation e2e continuation-scorecard --help`
- `bash scripts/ops/bigclawctl automation migration live-shadow-scorecard --help`
- `bash scripts/ops/bigclawctl legacy-python compile-check --json`
- `git status --short`

## Validation Results
- `find . -name '*.py' | wc -l` -> `0`
- `git ls-tree -r --name-only HEAD | rg '\.py$'` -> exit `1` with no tracked Python files
- `cd bigclaw-go && go test ./internal/regression -run 'TestPythonTestTranche17Removed|TestPythonResidualSweepRepoBaseline|TestPythonResidualSweepGoCompatibilitySurfaces'` -> `ok  	bigclaw-go/internal/regression	0.435s`
- `bash scripts/ops/bigclawctl automation e2e continuation-scorecard --help` -> exit `0`; printed `usage: bigclawctl automation e2e continuation-scorecard [flags]`
- `bash scripts/ops/bigclawctl automation migration live-shadow-scorecard --help` -> exit `0`; printed `usage: bigclawctl automation migration live-shadow-scorecard [flags]`
- `bash scripts/ops/bigclawctl legacy-python compile-check --json` -> exit `0`; JSON reported `status: ok`, `python: python3`, and `files: []`
- `git status --short --branch` -> `## BIG-GO-1152`; modified `.symphony/workpad.md`; added `bigclaw-go/internal/regression/python_residual_sweep_test.go`

## Residual Risk
- the repo already starts from a zero-`.py` baseline in this worktree, so this issue can only harden deletion enforcement for the candidate lane; it cannot make the Python file count numerically lower from the current baseline

## Archived Workpads
### BIG-GO-1115

#### Plan
- confirm the lane-owned candidate files from the issue context against the actual worktree
- document the zero-`.py` baseline in this branch so the acceptance risk is explicit before any code change
- add missing regression coverage for the still-uncovered candidate modules `src/bigclaw/planning.py`, `src/bigclaw/queue.py`, `src/bigclaw/reports.py`, and `src/bigclaw/risk.py`
- keep the existing `repo_*` candidate coverage unchanged because `top_level_module_purge_tranche2_test.go` and `top_level_module_purge_tranche10_test.go` already enforce those deletions
- run targeted validation for the new regression tranche plus repo-wide `.py` baseline checks
- commit and push the scoped change set

#### Acceptance
- lane file list is explicit:
- `src/bigclaw/planning.py`
- `src/bigclaw/queue.py`
- `src/bigclaw/repo_board.py`
- `src/bigclaw/repo_commits.py`
- `src/bigclaw/repo_gateway.py`
- `src/bigclaw/repo_governance.py`
- `src/bigclaw/repo_links.py`
- `src/bigclaw/repo_plane.py`
- `src/bigclaw/repo_registry.py`
- `src/bigclaw/repo_triage.py`
- `src/bigclaw/reports.py`
- `src/bigclaw/risk.py`
- the implementation stays scoped to the uncovered tranche for `planning.py`, `queue.py`, `reports.py`, and `risk.py`
- the repository continues to have no live `.py` files in the worktree
- exact validation commands and outcomes are recorded below
- residual risk explicitly notes that the issue goal of reducing Python file count further is already blocked by the pre-change zero baseline in this workspace

#### Validation
- `find . -name '*.py' | wc -l`
- `git ls-tree -r --name-only HEAD | rg '\.py$'`
- `cd bigclaw-go && go test ./internal/regression -run TestTopLevelModulePurgeTranche14`
- `cd bigclaw-go && go test ./internal/regression`
- `git status --short`

#### Validation Results
- `find . -name '*.py' | wc -l` -> `0`
- `git ls-tree -r --name-only HEAD | rg '\.py$'` -> exit `1` with no tracked Python files
- `cd bigclaw-go && go test ./internal/regression -run TestTopLevelModulePurgeTranche14` -> `ok  	bigclaw-go/internal/regression	0.459s`
- `cd bigclaw-go && go test ./internal/regression` -> `ok  	bigclaw-go/internal/regression	0.653s`
- `git status --short` -> modified `.symphony/workpad.md`; added `bigclaw-go/internal/regression/top_level_module_purge_tranche14_test.go`

#### Residual Risk
- the repo already starts from a zero-`.py` baseline in this worktree, so this issue can only harden deletion enforcement for the candidate lane; it cannot make the Python file count numerically lower from the current baseline

### BIG-GO-1114

#### Plan
- inventory the actual lane candidate files and confirm whether any real `.py` assets remain in this worktree
- record the explicit lane file list even if the physical Python surface is already at zero
- replace active planning evidence links that still point at deleted Python files with Go-owned implementation/test targets
- tighten handoff/parity docs so they describe the current Python-free repo state rather than implying active `src/bigclaw/*.py` runtime surfaces remain
- add regression coverage for the BIG-GO-1114 candidate set not yet locked by existing purge tranche tests
- run targeted validation for planning, regression, doc sweeps, legacy compile-check, and Python-file counts
- commit the scoped change set and push it to the remote branch

#### Acceptance
- lane coverage is explicit for `src/bigclaw/execution_contract.py`, `src/bigclaw/github_sync.py`, `src/bigclaw/governance.py`, `src/bigclaw/issue_archive.py`, `src/bigclaw/mapping.py`, `src/bigclaw/memory.py`, `src/bigclaw/models.py`, `src/bigclaw/observability.py`, `src/bigclaw/operations.py`, `src/bigclaw/orchestration.py`, `src/bigclaw/parallel_refill.py`, and `src/bigclaw/pilot.py`
- active planning and handoff artifacts stop pointing at deleted Python sources when a Go owner already exists
- regression coverage locks the remaining lane candidate files to absent-on-disk plus Go-owner-present assertions
- exact validation commands and outcomes are recorded, including the fact that the repo already materialized to zero `.py` files before the change
- residual risk is recorded if the Python-file-count acceptance cannot move further because the baseline is already zero

#### Validation
- `find . -name '*.py' | wc -l`
- `git ls-tree -r --name-only HEAD | rg '\.py$' | wc -l`
- `rg -n "src/bigclaw/(execution_contract|orchestration|saved_views)\\.py" bigclaw-go/internal/planning`
- `rg -n "remaining Python entrypoints|remaining Python compatibility surface|broader Python runtime/reporting/orchestration surface still remains" docs/go-mainline-cutover-handoff.md docs/go-domain-intake-parity-matrix.md`
- `cd bigclaw-go && go test ./internal/planning ./internal/regression ./internal/legacyshim ./cmd/bigclawctl`
- `bash scripts/ops/bigclawctl legacy-python compile-check --json`
- `git status --short`

#### Validation Results
- `find . -name '*.py' | wc -l` -> `0`
- `git ls-tree -r --name-only HEAD | rg '\.py$' | wc -l` -> `0`
- `rg -n "src/bigclaw/(execution_contract|orchestration|saved_views)\\.py" bigclaw-go/internal/planning` -> exit `1` with no matches
- `rg -n "remaining Python entrypoints|remaining Python compatibility surface|broader Python runtime/reporting/orchestration surface still remains" docs/go-mainline-cutover-handoff.md docs/go-domain-intake-parity-matrix.md` -> exit `1` with no matches
- `cd bigclaw-go && go test ./internal/planning ./internal/regression ./internal/legacyshim ./cmd/bigclawctl` -> `ok   bigclaw-go/internal/planning 0.743s`; `ok   bigclaw-go/internal/regression 0.912s`; `ok   bigclaw-go/internal/legacyshim 1.309s`; `ok   bigclaw-go/cmd/bigclawctl 4.646s`
- `bash scripts/ops/bigclawctl legacy-python compile-check --json` -> exit `0`; JSON reported `status: ok`, `python: python3`, and `files: []`
- `git status --short` -> modified `.symphony/workpad.md`, `bigclaw-go/internal/planning/planning.go`, `bigclaw-go/internal/planning/planning_test.go`, `docs/go-domain-intake-parity-matrix.md`, `docs/go-mainline-cutover-handoff.md`; added `bigclaw-go/internal/regression/top_level_module_purge_tranche15_test.go`

#### Residual Risk
- the repo already starts from a zero-`.py` baseline in this worktree, so this issue can only harden deletion enforcement and retire stale Go planning/handoff references; it cannot make the Python file count numerically lower from the current baseline
