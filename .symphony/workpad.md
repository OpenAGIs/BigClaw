# BIG-GO-1135

## Plan
- confirm the `BIG-GO-1135` candidate list against the actual worktree and record the zero-`.py` baseline before any other change
- add one issue-scoped regression guard for the residual sweep 5 candidate set so these deleted Python paths stay absent and their Go or shell replacements stay present
- add issue-local closeout artifacts under `reports/` that capture the candidate mapping, exact validation commands, and the already-materialized Python-free baseline
- run targeted validation for repo-wide `.py` counts, candidate-path scans, the new regression test, and the relevant Go CLI help surfaces
- commit and push the scoped change set

## Acceptance
- lane file list is explicit:
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
- the repo remains Python-free and the lane candidate paths stay absent on disk
- the documented replacement surface remains Go or shell only for benchmark, e2e, migration, and root-script entrypoints
- the repository continues to have no live `.py` files in the worktree
- exact validation commands and outcomes are recorded below
- residual risk explicitly notes that the issue goal of reducing Python file count further is already blocked by the pre-change zero baseline in this workspace

## Validation
- `find . -name '*.py' | wc -l`
- `git ls-tree -r --name-only HEAD | rg '\.py$' | wc -l`
- `rg -n "bigclaw-go/scripts/e2e/.*\.py|scripts/dev_smoke\.py" bigclaw-go/docs/go-cli-script-migration.md`
- `cd bigclaw-go && go test ./internal/regression -run TestPhysicalPythonResidualSweep5`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark capacity-certification --help | head -n 1`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e run-task-smoke --help | head -n 1`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration shadow-compare --help | head -n 1`
- `cd bigclaw-go && go run ./cmd/bigclawctl create-issues --help | head -n 1`
- `cd bigclaw-go && go run ./cmd/bigclawctl dev-smoke --help | head -n 1`
- `git status --short`

## Validation Results
- `find . -name '*.py' | wc -l` -> `0`
- `git ls-tree -r --name-only HEAD | rg '\.py$' | wc -l` -> `0`
- `rg -n "bigclaw-go/scripts/e2e/.*\.py|scripts/dev_smoke\.py" bigclaw-go/docs/go-cli-script-migration.md` -> exit `1` with no matches
- `cd bigclaw-go && go test ./internal/regression -run TestPhysicalPythonResidualSweep5` -> `ok  	bigclaw-go/internal/regression	0.829s`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark capacity-certification --help | head -n 1` -> `usage: bigclawctl automation benchmark capacity-certification [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e run-task-smoke --help | head -n 1` -> `usage: bigclawctl automation e2e run-task-smoke [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration shadow-compare --help | head -n 1` -> `usage: bigclawctl automation migration shadow-compare [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl create-issues --help | head -n 1` -> `usage: bigclawctl create-issues [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl dev-smoke --help | head -n 1` -> `usage: bigclawctl dev-smoke [flags]`
- `git status --short --branch` -> clean on `symphony/BIG-GO-1135...origin/symphony/BIG-GO-1135`

## Residual Risk
- the repo already starts from a zero-`.py` baseline in this worktree, so this issue can only harden deletion enforcement and refresh evidence for the candidate lane; it cannot make the Python file count numerically lower from the current baseline

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
