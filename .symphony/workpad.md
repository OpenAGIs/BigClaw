# BIG-GO-1129

## Plan
- confirm the issue candidate paths against the actual worktree and record the current zero-`.py` baseline before changing code
- keep the change set scoped to active regression and migration-plan surfaces that still encode deleted Python entrypoints from this lane
- update regression coverage so it verifies the Go-owned E2E entrypoints without reintroducing stale deleted-file references
- expand the migration plan to list the current Go replacements for the benchmark, E2E, migration, and repo-root candidates in this issue
- run targeted validation for repo Python count, migration regression coverage, and representative `bigclawctl automation ... --help` replacement paths
- commit the scoped changes and push the issue branch

## Acceptance
- the issue candidate surfaces are explicitly covered:
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
- active regression and plan docs stop naming deleted Python files as current entrypoints and instead point at the Go replacements
- Go replacement and compatibility paths are validated with exact commands and results recorded below
- the repository still reports zero live `.py` files in the worktree
- residual risk records that the numeric `find . -name '*.py' | wc -l` acceptance cannot decrease further because the pre-change baseline is already zero in this workspace

## Validation
- `find . -name '*.py' | wc -l`
- `git ls-tree -r --name-only HEAD | rg '\.py$'`
- `cd bigclaw-go && go test ./internal/regression -run 'TestE2EScriptDirectoryStaysPythonFree|TestE2EMigrationDocListsOnlyActiveEntrypoints'`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e run-task-smoke --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark capacity-certification --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration export-live-shadow-bundle --help`
- `bash scripts/ops/bigclawctl create-issues --help`
- `bash scripts/ops/bigclawctl dev-smoke --help`
- `git status --short`

## Validation Results
- `find . -name '*.py' | wc -l` -> `0`
- `git ls-tree -r --name-only HEAD | rg '\.py$'` -> exit `1` with no tracked Python files
- `cd bigclaw-go && go test ./internal/regression -run 'TestE2EScriptDirectoryStaysPythonFree|TestE2EMigrationDocListsOnlyActiveEntrypoints'` -> `ok  	bigclaw-go/internal/regression	0.446s`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e run-task-smoke --help` -> exit `0`; usage printed for `bigclawctl automation e2e run-task-smoke`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark capacity-certification --help` -> exit `0`; usage printed for `bigclawctl automation benchmark capacity-certification`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration export-live-shadow-bundle --help` -> exit `0`; usage printed for `bigclawctl automation migration export-live-shadow-bundle`
- `bash scripts/ops/bigclawctl create-issues --help` -> exit `0`; usage printed for `bigclawctl create-issues`
- `bash scripts/ops/bigclawctl dev-smoke --help` -> exit `0`; usage printed for `bigclawctl dev-smoke`
- `git status --short` -> modified `.symphony/workpad.md`, `bigclaw-go/internal/regression/e2e_entrypoint_migration_test.go`, and `docs/go-cli-script-migration-plan.md`

## Residual Risk
- the repo already starts from a zero-`.py` baseline in this worktree, so this issue can only harden active regression/docs truthfulness for the lane; it cannot make the Python file count numerically lower from the current baseline

## Archived Workpads
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
