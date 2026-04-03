# BIG-GO-1150

## Plan
- confirm the lane-owned candidate Python paths against the actual worktree and record the pre-change baseline
- keep the change scoped to regression hardening because this workspace already materializes with zero live `.py` files
- add explicit regression coverage for the remaining BIG-GO-1150 benchmark, e2e test-helper, and migration candidate Python paths so they stay absent
- assert the corresponding Go replacement or compatibility surfaces still exist for benchmark, e2e, and migration entrypoints
- run targeted validation for Python-file counts, the new regression tranche, the updated e2e regression, and the relevant Go CLI help entrypoints
- record exact commands and outcomes, then commit and push the scoped branch changes

## Acceptance
- the BIG-GO-1150 candidate paths are explicitly covered, including:
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
- Go-backed replacements or compatibility surfaces are asserted for the lane
- exact validation commands and outcomes are recorded below
- residual risk explicitly notes that the repo already starts at a zero-`.py` baseline, so the file count cannot decrease numerically in this workspace

## Validation
- `find . -name '*.py' | wc -l`
- `git ls-tree -r --name-only HEAD | rg '\.py$'`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche17|TestE2EMigrationDocListsOnlyActiveEntrypoints'`
- `cd bigclaw-go && go test ./cmd/bigclawctl -run TestBenchmarkScriptsStayGoOnly`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark soak-local --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark run-matrix --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark capacity-certification --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration shadow-compare --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration shadow-matrix --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration live-shadow-scorecard --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration export-live-shadow-bundle --help`
- `bash scripts/ops/bigclawctl create-issues --help`
- `bash scripts/ops/bigclawctl dev-smoke --help`
- `git status --short`

## Validation Results
- `find . -name '*.py' | wc -l` -> `0`
- `git ls-tree -r --name-only HEAD | rg '\.py$'` -> exit `1` with no tracked Python files
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche17|TestE2EMigrationDocListsOnlyActiveEntrypoints'` -> `ok  	bigclaw-go/internal/regression	0.478s`
- `cd bigclaw-go && go test ./cmd/bigclawctl -run TestBenchmarkScriptsStayGoOnly` -> `ok  	bigclaw-go/cmd/bigclawctl	1.700s`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark soak-local --help` -> exit `0`; printed `usage: bigclawctl automation benchmark soak-local [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark run-matrix --help` -> exit `0`; printed `usage: bigclawctl automation benchmark run-matrix [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark capacity-certification --help` -> exit `0`; printed `usage: bigclawctl automation benchmark capacity-certification [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration shadow-compare --help` -> exit `0`; printed `usage: bigclawctl automation migration shadow-compare [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration shadow-matrix --help` -> exit `0`; printed `usage: bigclawctl automation migration shadow-matrix [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration live-shadow-scorecard --help` -> exit `0`; printed `usage: bigclawctl automation migration live-shadow-scorecard [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration export-live-shadow-bundle --help` -> exit `0`; printed `usage: bigclawctl automation migration export-live-shadow-bundle [flags]`
- `bash scripts/ops/bigclawctl create-issues --help` -> exit `0`; printed `usage: bigclawctl create-issues [flags]`
- `bash scripts/ops/bigclawctl dev-smoke --help` -> exit `0`; printed `usage: bigclawctl dev-smoke [flags]`
- `git status --short` -> modified `.symphony/workpad.md`; modified `bigclaw-go/internal/regression/e2e_entrypoint_migration_test.go`; added `bigclaw-go/internal/regression/top_level_module_purge_tranche17_test.go`

## Residual Risk
- the repo already starts from a zero-`.py` baseline in this worktree, so this issue can only harden deletion enforcement for the lane; it cannot make the Python file count numerically lower from the current baseline

# BIG-GO-1133

## Plan
- confirm the lane-owned candidate root script paths from the issue against the actual worktree
- record the repo's pre-change zero-`.py` baseline so the acceptance limitation is explicit before implementation
- add scoped regression coverage for the retired Python script paths still named by this lane:
- `scripts/create_issues.py`
- `scripts/dev_smoke.py`
- `scripts/ops/bigclaw_github_sync.py`
- `scripts/ops/bigclaw_refill_queue.py`
- `scripts/ops/bigclaw_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_validate.py`
- verify each retired path still has a live Go-backed replacement or compatibility wrapper in the repo
- run targeted validation for the Python-file count baseline, the new regression tranche, and the active Go replacement entrypoints
- commit and push the scoped change set

## Acceptance
- the lane candidate root script paths are explicitly covered and remain absent from disk
- Go replacements or compatibility wrappers are asserted for the lane:
- `bigclaw-go/cmd/bigclawctl/main.go`
- `scripts/ops/bigclawctl`
- `scripts/ops/bigclaw-symphony`
- `scripts/dev_bootstrap.sh`
- the repository remains at zero live `.py` files in the worktree
- exact validation commands and outcomes are recorded below
- residual risk explicitly notes that the issue goal of lowering the Python-file count further is blocked by the pre-change zero baseline in this workspace

## Validation
- `find . -name '*.py' | wc -l`
- `git ls-tree -r --name-only HEAD | rg '\.py$'`
- `cd bigclaw-go && go test ./internal/regression -run TestTopLevelModulePurgeTranche16`
- `bash scripts/ops/bigclawctl create-issues --help`
- `bash scripts/ops/bigclawctl dev-smoke --help`
- `bash scripts/ops/bigclawctl github-sync --help`
- `bash scripts/ops/bigclawctl refill --help`
- `bash scripts/ops/bigclawctl workspace validate --help`
- `bash scripts/ops/bigclaw-symphony --help`
- `bash scripts/ops/bigclawctl legacy-python compile-check --json`
- `git status --short`

## Validation Results
- `find . -name '*.py' | wc -l` -> `0`
- `git ls-tree -r --name-only HEAD | rg '\.py$'` -> exit `1` with no tracked Python files
- `cd bigclaw-go && go test ./internal/regression -run TestTopLevelModulePurgeTranche16` -> `ok  	bigclaw-go/internal/regression	0.442s`
- `bash scripts/ops/bigclawctl create-issues --help` -> exit `0`; printed `usage: bigclawctl create-issues [flags]`
- `bash scripts/ops/bigclawctl dev-smoke --help` -> exit `0`; printed `usage: bigclawctl dev-smoke [flags]`
- `bash scripts/ops/bigclawctl github-sync --help` -> exit `0`; printed `usage: bigclawctl github-sync <install|status|sync> [flags]`
- `bash scripts/ops/bigclawctl refill --help` -> exit `0`; printed `usage: bigclawctl refill [flags]` and `bigclawctl refill seed [flags]`
- `bash scripts/ops/bigclawctl workspace validate --help` -> exit `0`; printed `usage: bigclawctl workspace validate [flags]`
- `bash scripts/ops/bigclaw-symphony --help` -> exit `0`; printed `usage: bigclawctl symphony [flags] [args...]`
- `bash scripts/ops/bigclawctl legacy-python compile-check --json` -> exit `0`; JSON reported `status: ok`, `python: python3`, and `files: []`
- `git status --short` -> modified `.symphony/workpad.md`; added `bigclaw-go/internal/regression/top_level_module_purge_tranche16_test.go`

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
