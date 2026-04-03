# BIG-GO-1151

## Plan
- confirm which `src/bigclaw/*.py` candidates from this lane still lack explicit regression coverage and record the pre-change Python-file baseline
- replace the stale active workpad section with this issue's scoped plan before any code changes
- add one scoped regression tranche that locks the uncovered residual top-level Python modules to absent-on-disk and asserts the active Go replacement coverage for the same domains
- validate the new tranche plus repo-wide Python-count checks and a Go CLI compatibility path
- record exact commands and outcomes, then commit and push the scoped change set

## Acceptance
- the lane candidate Python module paths are explicitly covered and remain absent from disk:
- `src/bigclaw/__init__.py`
- `src/bigclaw/__main__.py`
- `src/bigclaw/audit_events.py`
- `src/bigclaw/collaboration.py`
- `src/bigclaw/console_ia.py`
- `src/bigclaw/design_system.py`
- `src/bigclaw/evaluation.py`
- `src/bigclaw/run_detail.py`
- `src/bigclaw/runtime.py`
- Go replacements or compatibility surfaces are asserted for this lane through concrete repo-native files under `bigclaw-go/internal` and `bigclaw-go/cmd`
- the repository remains at zero live `.py` files in the worktree
- exact validation commands and outcomes are recorded below
- residual risk explicitly notes that this workspace already starts at a zero-`.py` baseline, so the count cannot numerically decrease further here

## Validation
- `find . -name '*.py' | wc -l`
- `git ls-tree -r --name-only HEAD | rg '\.py$'`
- `cd bigclaw-go && go test ./internal/regression -run TestTopLevelModulePurgeTranche17`
- `bash scripts/ops/bigclawctl automation migration live-shadow-scorecard --help`
- `bash scripts/ops/bigclawctl legacy-python compile-check --json`
- `git status --short`

## Validation Results
- `find . -name '*.py' | wc -l` -> `0`
- `git ls-tree -r --name-only HEAD | rg '\.py$'` -> exit `1` with no tracked Python files
- `cd bigclaw-go && go test ./internal/regression -run TestTopLevelModulePurgeTranche17` -> `ok  	bigclaw-go/internal/regression	1.486s`
- `bash scripts/ops/bigclawctl automation migration live-shadow-scorecard --help` -> exit `0`; printed `usage: bigclawctl automation migration live-shadow-scorecard [flags]`
- `bash scripts/ops/bigclawctl legacy-python compile-check --json` -> exit `0`; JSON reported `status: ok`, `python: python3`, and `files: []`
- `git status --short` -> modified `.symphony/workpad.md`; added `bigclaw-go/internal/regression/top_level_module_purge_tranche17_test.go`

## Residual Risk
- the repo already starts from a zero-`.py` baseline in this worktree, so this issue can only harden deletion enforcement for the candidate lane; it cannot make the Python file count numerically lower from the current baseline

## Archived Workpads
### BIG-GO-1155

#### Plan
- Verify the repository-wide Python file count.
- Check whether the issue's candidate Python paths still exist in this workspace.
- Confirm the available Go or shell execution paths in the affected script areas.
- Keep the change scoped to this issue by recording the audit and validation only if the lane is already complete.
- Commit and push the resulting issue-scoped artifact.

#### Acceptance
- Real Python files in this lane are covered or confirmed absent.
- Go replacement or compatible non-Python execution paths are verified for the relevant script areas.
- `find . -name '*.py' | wc -l` is confirmed and does not regress.

#### Validation
- `find . -name '*.py' | wc -l`
- `find bigclaw-go/scripts -maxdepth 3 -type f | sort`
- `find scripts -maxdepth 2 -type f | sort`
- `go test ./internal/regression -run 'TestTopLevelModulePurgeTranche16|TestE2EScriptDirectoryStaysPythonFree|TestE2EMigrationDocListsOnlyActiveEntrypoints'`
- `git status --short`

#### Notes
- Initial inspection in this workspace shows zero Python files repository-wide.
- The candidate Python files listed in the issue are not present in this checkout.
- Existing compatible paths observed during inspection:
- `bigclaw-go/scripts/e2e/broker_bootstrap_summary.go`
- `bigclaw-go/scripts/e2e/run_all.sh`
- `bigclaw-go/scripts/e2e/kubernetes_smoke.sh`
- `bigclaw-go/scripts/e2e/ray_smoke.sh`
- `bigclaw-go/scripts/benchmark/run_suite.sh`
- `scripts/dev_bootstrap.sh`

#### Results
- `find . -name '*.py' | wc -l`
- Result: `0`
- `find bigclaw-go/scripts -maxdepth 3 -type f | sort`
- Result:
- `bigclaw-go/scripts/benchmark/run_suite.sh`
- `bigclaw-go/scripts/e2e/broker_bootstrap_summary.go`
- `bigclaw-go/scripts/e2e/kubernetes_smoke.sh`
- `bigclaw-go/scripts/e2e/ray_smoke.sh`
- `bigclaw-go/scripts/e2e/run_all.sh`
- `find scripts -maxdepth 2 -type f | sort`
- Result:
- `scripts/dev_bootstrap.sh`
- `scripts/ops/bigclaw-issue`
- `scripts/ops/bigclaw-panel`
- `scripts/ops/bigclaw-symphony`
- `scripts/ops/bigclawctl`
- `go test ./internal/regression -run 'TestTopLevelModulePurgeTranche16|TestE2EScriptDirectoryStaysPythonFree|TestE2EMigrationDocListsOnlyActiveEntrypoints'`
- Result: `ok  	bigclaw-go/internal/regression	3.211s`

#### Residual Risk
- The repository already starts from a zero-`.py` baseline in this workspace, so this issue can only document and validate the completed lane state; it cannot make the Python file count numerically lower from the current baseline.

### BIG-GO-1153

#### Plan
- confirm the root-script candidate list against the actual worktree and record the pre-change Python-file baseline
- replace the stale active workpad section with this issue's scoped plan before any code changes
- add a regression tranche that locks the retired root Python script paths to absent-on-disk and asserts the documented Go or shell compatibility entrypoints for the same lane
- validate the new tranche plus repo-wide Python-count checks and targeted `bigclawctl` help/compatibility commands
- record exact commands and outcomes, then commit and push the scoped change set

#### Acceptance
- the lane candidate Python files are explicitly covered and remain absent from disk:
- `scripts/create_issues.py`
- `scripts/dev_smoke.py`
- `scripts/ops/bigclaw_github_sync.py`
- `scripts/ops/bigclaw_refill_queue.py`
- `scripts/ops/bigclaw_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_validate.py`
- Go replacements or compatibility entrypoints are asserted for this lane through concrete files and docs under `bigclaw-go/cmd`, `bigclaw-go/internal`, `scripts/ops`, `scripts/dev_bootstrap.sh`, `docs/go-cli-script-migration-plan.md`, and `README.md`

### BIG-GO-1142

#### Plan
- confirm the lane-owned `tests/*.py` candidate list against the actual worktree and record the pre-change Python-file baseline
- replace the stale active workpad section with this issue's scoped plan before any code changes
- add a regression tranche that locks the retired Python test paths to absent-on-disk and asserts the active Go replacement coverage for the same domains
- validate the new tranche plus repo-wide Python-count checks and a live Go compatibility path
- record exact commands and outcomes, then commit and push the scoped change set

#### Acceptance
- the lane candidate Python test paths are explicitly covered and remain absent from disk:
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
- Go replacements or compatibility surfaces are asserted for this lane through concrete repo-native files under `bigclaw-go/internal`, `bigclaw-go/cmd`, and `docs/reports`
- the repository remains at zero live `.py` files in the worktree
- exact validation commands and outcomes are recorded below
- residual risk explicitly notes that this workspace already starts at a zero-`.py` baseline, so the count cannot numerically decrease further here

#### Validation
- `find . -name '*.py' | wc -l`
- `git ls-tree -r --name-only HEAD | rg '\.py$'`
- `cd bigclaw-go && go test ./internal/regression -run TestRootScriptResidualSweep`
- `bash scripts/ops/bigclawctl create-issues --help`
- `bash scripts/ops/bigclawctl dev-smoke --help`
- `bash scripts/ops/bigclawctl github-sync --help`
- `bash scripts/ops/bigclawctl refill --help`
- `bash scripts/ops/bigclawctl workspace validate --help`
- `bash scripts/ops/bigclawctl legacy-python compile-check --json`
- `git status --short`

#### Validation Results
- `find . -name '*.py' | wc -l` -> `0`
- `git ls-tree -r --name-only HEAD | rg '\.py$'` -> exit `1` with no tracked Python files
- `cd bigclaw-go && go test ./internal/regression -run TestRootScriptResidualSweep` -> `ok  	bigclaw-go/internal/regression	0.499s`
- `bash scripts/ops/bigclawctl create-issues --help` -> exit `0`; printed `usage: bigclawctl create-issues [flags]`
- `bash scripts/ops/bigclawctl dev-smoke --help` -> exit `0`; printed `usage: bigclawctl dev-smoke [flags]`
- `bash scripts/ops/bigclawctl github-sync --help` -> exit `0`; printed `usage: bigclawctl github-sync <install|status|sync> [flags]`
- `bash scripts/ops/bigclawctl refill --help` -> exit `0`; printed `usage: bigclawctl refill [flags]`
- `bash scripts/ops/bigclawctl workspace validate --help` -> exit `0`; printed `usage: bigclawctl workspace validate [flags]`
- `bash scripts/ops/bigclawctl legacy-python compile-check --json` -> exit `0`; JSON reported `status: ok`, `python: python3`, and `files: []`
- `git status --short` -> modified `.symphony/workpad.md`; added `bigclaw-go/internal/regression/root_script_residual_sweep_test.go`

#### Residual Risk
- the repo already starts from a zero-`.py` baseline in this worktree, so this issue can only harden deletion enforcement for the candidate lane; it cannot make the Python file count numerically lower from the current baseline

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
