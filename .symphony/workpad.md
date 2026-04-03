# BIG-GO-1124

## Plan
- confirm whether the issue candidate Python scripts still exist in this worktree
- record the current repo baseline before edits, including the effective `*.py` count
- document the Go replacement and compatibility path already covering the benchmark, e2e, and migration script families
- keep the change scoped to issue evidence and validation for the residual `bigclaw-go/scripts` sweep
- run targeted validation for the Go command surfaces plus repo-wide Python-file checks
- commit and push the scoped change set

## Acceptance
- the candidate `bigclaw-go/scripts/benchmark/*.py`, `bigclaw-go/scripts/e2e/*.py`, and `bigclaw-go/scripts/migration/*.py` assets are verified absent in the current worktree
- the Go replacement path is explicit for benchmark, e2e, and migration command families
- `bigclaw-go/scripts/` remains Python-free, with only the retained shell wrappers and Go-owned entrypoints documented
- exact validation commands and outcomes are recorded below
- residual risk explicitly notes that the issue goal of lowering the repo Python count is blocked by the pre-change zero baseline in this workspace

## Validation
- `find . -name '*.py' | wc -l`
- `find bigclaw-go/scripts -type f | sort`
- `cd bigclaw-go && go test ./cmd/bigclawctl/...`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark capacity-certification --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark run-matrix --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark soak-local --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e broker-failover-stub-matrix --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e cross-process-coordination-surface --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e export-validation-bundle --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e external-store-validation --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e mixed-workload-matrix --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e multi-node-shared-queue --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e run-task-smoke --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e subscriber-takeover-fault-matrix --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e continuation-policy-gate --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e continuation-scorecard --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration export-live-shadow-bundle --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration live-shadow-scorecard --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration shadow-compare --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration shadow-matrix --help`
- `git status --short`

## Validation Results
- `find . -name '*.py' | wc -l` -> `0`
- `find bigclaw-go/scripts -type f | sort` -> `bigclaw-go/scripts/benchmark/run_suite.sh`, `bigclaw-go/scripts/e2e/broker_bootstrap_summary.go`, `bigclaw-go/scripts/e2e/kubernetes_smoke.sh`, `bigclaw-go/scripts/e2e/ray_smoke.sh`, `bigclaw-go/scripts/e2e/run_all.sh`
- `cd bigclaw-go && go test ./cmd/bigclawctl/...` -> `ok   bigclaw-go/cmd/bigclawctl 3.094s`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark capacity-certification --help` -> exit `0`; first line `usage: bigclawctl automation benchmark capacity-certification [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark run-matrix --help` -> exit `0`; first line `usage: bigclawctl automation benchmark run-matrix [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark soak-local --help` -> exit `0`; first line `usage: bigclawctl automation benchmark soak-local [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e broker-failover-stub-matrix --help` -> exit `0`; first line `usage: bigclawctl automation e2e broker-failover-stub-matrix [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e cross-process-coordination-surface --help` -> exit `0`; first line `usage: bigclawctl automation e2e cross-process-coordination-surface [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e export-validation-bundle --help` -> exit `0`; first line `usage: bigclawctl automation e2e export-validation-bundle [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e external-store-validation --help` -> exit `0`; first line `usage: bigclawctl automation e2e external-store-validation [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e mixed-workload-matrix --help` -> exit `0`; first line `usage: bigclawctl automation e2e mixed-workload-matrix [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e multi-node-shared-queue --help` -> exit `0`; first line `usage: bigclawctl automation e2e multi-node-shared-queue [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e run-task-smoke --help` -> exit `0`; first line `usage: bigclawctl automation e2e run-task-smoke [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e subscriber-takeover-fault-matrix --help` -> exit `0`; first line `usage: bigclawctl automation e2e subscriber-takeover-fault-matrix [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e continuation-policy-gate --help` -> exit `0`; first line `usage: bigclawctl automation e2e continuation-policy-gate [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e continuation-scorecard --help` -> exit `0`; first line `usage: bigclawctl automation e2e continuation-scorecard [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration export-live-shadow-bundle --help` -> exit `0`; first line `usage: bigclawctl automation migration export-live-shadow-bundle [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration live-shadow-scorecard --help` -> exit `0`; first line `usage: bigclawctl automation migration live-shadow-scorecard [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration shadow-compare --help` -> exit `0`; first line `usage: bigclawctl automation migration shadow-compare [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration shadow-matrix --help` -> exit `0`; first line `usage: bigclawctl automation migration shadow-matrix [flags]`
- `git status --short` -> pending final commit state at closeout

## Residual Risk
- the repo already starts from a zero-`.py` baseline in this worktree, so this issue can only record and preserve the Python-free `bigclaw-go/scripts` state; it cannot make the Python file count numerically lower from the current baseline

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
