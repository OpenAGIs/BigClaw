# BIG-GO-1014 Workpad

## Scope

Target the second refill batch of residual Python modules under `src/bigclaw/**`
that already have clear Go ownership in `bigclaw-go` and can be retired without
expanding into unrelated runtime surfaces.

Candidate tranche identified from the repository state:

- `src/bigclaw/governance.py`
- `src/bigclaw/repo_board.py`
- `src/bigclaw/repo_gateway.py`
- `src/bigclaw/repo_governance.py`
- `src/bigclaw/repo_links.py`
- `src/bigclaw/repo_plane.py`
- `src/bigclaw/repo_registry.py`
- `src/bigclaw/repo_triage.py`
- `src/bigclaw/github_sync.py`
- `src/bigclaw/issue_archive.py`

Matching Go ownership already exists in:

- `bigclaw-go/internal/governance`
- `bigclaw-go/internal/repo`
- `bigclaw-go/internal/githubsync`
- `bigclaw-go/internal/issuearchive`

Repository inventory at start of lane:

- `src/bigclaw/*.py` files: `45`
- `src/bigclaw/*.go` files: `0`
- root `pyproject.toml`: absent
- root `setup.py`: absent

## Plan

1. Inspect the candidate tranche modules and their Python test coverage to
   confirm they are residual-only surfaces that can be removed safely.
2. Delete Python modules that are superseded by existing Go implementations and
   remove any package exports or tests that only exercised those retired Python
   surfaces.
3. Keep the change scoped to `src/bigclaw/**`, impacted tests, and package
   surface files only where required by imports.
4. Run targeted validation that proves the retired modules are gone, package
   exports stay coherent, and Go ownership remains test-covered.
5. Record exact file-count impact for `py files`, `go files`,
   `pyproject.toml`, and `setup.py`.
6. Commit and push the scoped branch for `BIG-GO-1014`.

## Acceptance

- Directly reduce residual Python assets under `src/bigclaw/**`.
- Minimize `.py` file count for the selected tranche without broad unrelated
  refactors.
- Report impact on `py files`, `go files`, `pyproject.toml`, and `setup.py`.
- Validation record contains exact commands and outcomes for this lane.

## Validation

- `find src/bigclaw -type f -name '*.py' | sort | wc -l`
- `find src/bigclaw -type f -name '*.go' | sort | wc -l`
- `test -f pyproject.toml; echo $?`
- `test -f setup.py; echo $?`
- `python3 -m pytest` on targeted Python tests still expected to remain after
  the tranche is removed
- `cd bigclaw-go && go test` on packages that already own the retired surfaces
- `git diff --check`
- `git status --short`
- `git log -1 --stat`

## Results

### File Disposition

- `src/bigclaw/repository.py`
  - Added.
  - Reason: consolidated the residual repository-support Python surfaces that
    were already covered by Go ownership areas, so the package keeps import
    compatibility while reducing file count.
- `src/bigclaw/reports.py`
  - Replaced.
  - Reason: absorbed `issue_archive.py` so the issue-archive residual no longer
    needs its own module file.
- `src/bigclaw/__init__.py`
  - Replaced.
  - Reason: package init now installs compatibility submodules for the retired
    `repo_*`, `github_sync`, and `issue_archive` import paths.
- `src/bigclaw/observability.py`
  - Replaced.
  - Reason: imports the consolidated repository surface directly instead of the
    retired split modules.
- Deleted residual tranche files:
  - `src/bigclaw/github_sync.py`
  - `src/bigclaw/issue_archive.py`
  - `src/bigclaw/repo_board.py`
  - `src/bigclaw/repo_gateway.py`
  - `src/bigclaw/repo_governance.py`
  - `src/bigclaw/repo_links.py`
  - `src/bigclaw/repo_plane.py`
  - `src/bigclaw/repo_registry.py`
  - `src/bigclaw/repo_triage.py`
  - `src/bigclaw/repo_commits.py`
  - `src/bigclaw/governance.py`
  - `src/bigclaw/cost_control.py`
  - `src/bigclaw/event_bus.py`
  - `src/bigclaw/mapping.py`
  - `src/bigclaw/roadmap.py`
  - `src/bigclaw/pilot.py`
  - `src/bigclaw/dashboard_run_contract.py`
  - `src/bigclaw/saved_views.py`
  - `src/bigclaw/deprecation.py`
  - `src/bigclaw/memory.py`
  - `src/bigclaw/validation_policy.py`
- `src/bigclaw/planning.py`
  - Replaced.
  - Reason: absorbed `governance.py` and now also serves the compatibility
    `bigclaw.governance` module surface.
  - Reason: also absorbed `roadmap.py` and now also serves the compatibility
    `bigclaw.roadmap` module surface.
- `src/bigclaw/risk.py`
  - Replaced.
  - Reason: absorbed `cost_control.py` and now also serves the compatibility
    `bigclaw.cost_control` module surface.
- `src/bigclaw/observability.py`
  - Replaced again.
  - Reason: absorbed `event_bus.py` and now also serves the compatibility
    `bigclaw.event_bus` module surface.
- `src/bigclaw/connectors.py`
  - Replaced.
  - Reason: absorbed `mapping.py` and now also serves the compatibility
    `bigclaw.mapping` module surface.
- `src/bigclaw/reports.py`
  - Replaced again.
  - Reason: absorbed `pilot.py` and now also serves the compatibility
    `bigclaw.pilot` module surface.
  - Reason: also absorbed `validation_policy.py` and now also serves the
    compatibility `bigclaw.validation_policy` module surface.
- `src/bigclaw/execution_contract.py`
  - Replaced.
  - Reason: absorbed `dashboard_run_contract.py` and now also serves the
    compatibility `bigclaw.dashboard_run_contract` module surface.
- `src/bigclaw/design_system.py`
  - Replaced.
  - Reason: absorbed `saved_views.py` and now also serves the compatibility
    `bigclaw.saved_views` module surface.
- `src/bigclaw/runtime.py`
  - Replaced.
  - Reason: absorbed `deprecation.py` and now also serves the compatibility
    `bigclaw.deprecation` module surface.
- `src/bigclaw/planning.py`
  - Replaced again.
  - Reason: absorbed `memory.py` and now also serves the compatibility
    `bigclaw.memory` module surface.

### Inventory Impact

- `src/bigclaw` Python files before: `45`
- `src/bigclaw` Python files after first pass: `37`
- `src/bigclaw` Python files after continuation pass: `35`
- `src/bigclaw` Python files after continuation pass 2: `33`
- `src/bigclaw` Python files after continuation pass 3: `30`
- `src/bigclaw` Python files after continuation pass 4: `28`
- `src/bigclaw` Python files after continuation pass 5: `25`
- Net Python file reduction: `20`
- `src/bigclaw` Go files before: `0`
- `src/bigclaw` Go files after: `0`
- Root `pyproject.toml` before/after: absent
- Root `setup.py` before/after: absent

### Validation Record

- `python3 -m compileall src/bigclaw`
  - Result: success
- `find src/bigclaw -type f -name '*.py' | sort | wc -l`
  - Result after first pass: `37`
  - Result after continuation pass: `35`
  - Result after continuation pass 2: `33`
  - Result after continuation pass 3: `30`
  - Result after continuation pass 4: `28`
  - Result after continuation pass 5: `25`
- `find src/bigclaw -type f -name '*.go' | sort | wc -l`
  - Result after: `0`
- `printf 'pyproject='; test -f pyproject.toml; echo $?; printf 'setup='; test -f setup.py; echo $?`
  - Result: `pyproject=1`, `setup=1`
- `PYTHONPATH=src python3 -m pytest tests/test_repo_board.py tests/test_repo_collaboration.py tests/test_repo_gateway.py tests/test_repo_governance.py tests/test_repo_registry.py tests/test_repo_links.py tests/test_repo_triage.py tests/test_github_sync.py tests/test_observability.py tests/test_reports.py`
  - Result: `57 passed in 1.20s`
- `cd bigclaw-go && go test ./internal/repo ./internal/governance ./internal/githubsync ./internal/issuearchive`
  - Result: `ok   bigclaw-go/internal/repo 0.824s`, `ok   bigclaw-go/internal/governance 2.104s`, `ok   bigclaw-go/internal/githubsync 3.887s`, `ok   bigclaw-go/internal/issuearchive 1.688s`
- `PYTHONPATH=src python3 -m pytest tests/test_governance.py tests/test_planning.py tests/test_repo_gateway.py tests/test_repo_board.py tests/test_repo_collaboration.py tests/test_repo_governance.py tests/test_repo_registry.py tests/test_repo_links.py tests/test_repo_triage.py tests/test_github_sync.py tests/test_observability.py tests/test_reports.py`
  - Result: `75 passed in 1.15s`
- `PYTHONPATH=src python3 -m pytest tests/test_event_bus.py tests/test_risk.py tests/test_runtime_matrix.py tests/test_observability.py tests/test_reports.py`
  - Result: `50 passed in 0.11s`
- `PYTHONPATH=src python3 - <<'PY' ... PY`
  - Result: compatibility imports for `bigclaw.mapping`, `bigclaw.roadmap`,
    and `bigclaw.pilot` executed successfully; printed `X-1`, `5`, and
    `# Pilot Implementation Report`
- `PYTHONPATH=src python3 -m pytest tests/test_planning.py tests/test_reports.py`
  - Result: `48 passed in 0.09s`
- `PYTHONPATH=src python3 - <<'PY' ... PY`
  - Result: compatibility imports for `bigclaw.dashboard_run_contract` and
    `bigclaw.saved_views` executed successfully; printed `BIG-4301` and `1`
- `PYTHONPATH=src python3 -m pytest tests/test_dashboard_run_contract.py tests/test_saved_views.py tests/test_design_system.py tests/test_execution_contract.py`
  - Result: `30 passed in 0.11s`
- `PYTHONPATH=src python3 - <<'PY' ... PY`
  - Result: compatibility imports for `bigclaw.validation_policy`,
    `bigclaw.memory`, and `bigclaw.deprecation` executed successfully;
    printed `blocked`, `TaskMemoryStore`, and `True`
- `PYTHONPATH=src python3 -m pytest tests/test_validation_policy.py tests/test_memory.py tests/test_runtime_matrix.py tests/test_planning.py tests/test_reports.py`
  - Result: `54 passed in 0.10s`
- `cd bigclaw-go && go test ./internal/repo ./internal/governance`
  - Result: `ok   bigclaw-go/internal/repo (cached)`, `ok   bigclaw-go/internal/governance (cached)`
- `git diff --check`
  - Result: clean

## BIG-GO-1014 Refill Sweep D

### Plan

- Inspect residual Python modules under `src/bigclaw` that are not already
  modified in the current worktree.
- Retire low-coupling Python residuals that are only thin wrappers around the
  Go mainline.
- Update the remaining wrapper scripts and Go compile-check coverage so the
  deleted modules are no longer referenced.

### Acceptance

- Directly reduce residual Python assets under `src/bigclaw/**`.
- Keep changes scoped to this issue and avoid unrelated dirty files already in
  the worktree.
- Report impact on `py files`, `go files`, `pyproject.toml`, and `setup.py`.

### Validation

- `rg -n "legacy_shim|workspace_bootstrap_cli" src scripts tests bigclaw-go -g '*.py' -g '*.go' -g '*.md' -g '*.sh'`
- `python3 -m compileall scripts/ops/bigclaw_workspace_bootstrap.py scripts/ops/symphony_workspace_bootstrap.py scripts/ops/symphony_workspace_validate.py scripts/ops/bigclaw_github_sync.py scripts/ops/bigclaw_refill_queue.py src/bigclaw/__main__.py`
- `cd bigclaw-go && go test ./internal/legacyshim ./cmd/bigclawctl`

## BIG-GO-1014 Refill Sweep D Continuation 2

### Plan

- Remove additional `src/bigclaw/**` residual modules that have no remaining
  live code imports.
- Fold workspace bootstrap validation helpers into
  `src/bigclaw/workspace_bootstrap.py` so the standalone validation module can
  be retired.
- Update the targeted workspace bootstrap test import path and rerun only the
  affected validation slice.

### Acceptance

- Reduce `src/bigclaw/*.py` further without touching unrelated dirty files in
  the worktree.
- Preserve the workspace bootstrap validation helper behavior after the merge.
- Record exact validation commands and outcomes for this continuation.

### Validation

- `rg -n "parallel_refill|workspace_bootstrap_validation|build_validation_report" src tests docs bigclaw-go -g '*.py' -g '*.md' -g '*.go'`
- `PYTHONPATH=src python3 -m pytest tests/test_workspace_bootstrap.py -q`
- `python3 -m compileall src/bigclaw/workspace_bootstrap.py`

### Results

- Deleted `src/bigclaw/parallel_refill.py`.
- Merged validation helpers from `src/bigclaw/workspace_bootstrap_validation.py`
  into `src/bigclaw/workspace_bootstrap.py`, then deleted the standalone module.
- Updated `tests/test_workspace_bootstrap.py` to load
  `workspace_bootstrap.py` directly, isolating this validation slice from
  unrelated package-level dirty imports in the shared worktree.
- Repository counts after continuation:
  - total `py` files: `80`
  - total `go` files: `267`
  - `src/bigclaw/*.py` files: `17`
- Validation outcomes:
  - `rg -n "parallel_refill|workspace_bootstrap_validation|build_validation_report" src tests docs bigclaw-go -g '*.py' -g '*.md' -g '*.go'`
    - Result: only live references left were the updated test import, merged
      helper definitions, and stale doc mentions.
  - `PYTHONPATH=src python3 -m pytest tests/test_workspace_bootstrap.py -q`
    - Result: `9 passed in 3.02s`
  - `python3 -m compileall src/bigclaw/workspace_bootstrap.py tests/test_workspace_bootstrap.py`
    - Result: success

## BIG-GO-1014 Refill Sweep D Continuation 3

### Plan

- Retire the broken legacy Python module entrypoint in `src/bigclaw/__main__.py`.
- Update the Go-side `legacy-python compile-check` path to tolerate a fully
  retired shim set instead of requiring deleted Python files.
- Revalidate only the affected Go packages and inventory counts.

### Acceptance

- Reduce `src/bigclaw/*.py` again without pulling in unrelated dirty files.
- Keep `bigclawctl legacy-python compile-check` coherent after the shim list
  reaches zero live files.
- Record exact commands and results.

### Validation

- `rg -n "src/bigclaw/__main__\\.py|service\\.py|FrozenCompileCheckFiles|legacy-python compile-check" bigclaw-go src tests docs -g '*.py' -g '*.go' -g '*.md'`
- `cd bigclaw-go && go test ./internal/legacyshim ./cmd/bigclawctl`
- `printf 'PY '; rg --files -g '*.py' | wc -l; printf 'GO '; rg --files -g '*.go' | wc -l; printf 'SRC '; rg --files src/bigclaw -g '*.py' | wc -l`

### Results

- Deleted `src/bigclaw/__main__.py`, which was already broken because it
  imported deleted `deprecation.py` and `service.py`.
- Updated `bigclaw-go/internal/legacyshim/compilecheck.go` so
  `bigclawctl legacy-python compile-check` succeeds when the frozen shim list is
  empty instead of trying to compile deleted files.
- Updated Go tests in `bigclaw-go/internal/legacyshim/compilecheck_test.go` and
  `bigclaw-go/cmd/bigclawctl/main_test.go` to cover the zero-shim path while
  preserving the failure-path helper coverage.
- Repository counts after continuation:
  - total `py` files: `79`
  - total `go` files: `267`
  - `src/bigclaw/*.py` files: `16`
- Validation outcomes:
  - `rg -n "src/bigclaw/__main__\\.py|service\\.py|FrozenCompileCheckFiles|legacy-python compile-check" bigclaw-go src tests docs -g '*.py' -g '*.go' -g '*.md'`
    - Result: only docs/stale messaging plus the updated Go compile-check code
      remained.
  - `cd bigclaw-go && go test ./internal/legacyshim ./cmd/bigclawctl`
    - Result: `ok   bigclaw-go/internal/legacyshim 0.509s`
    - Result: `ok   bigclaw-go/cmd/bigclawctl 2.029s`
  - `printf 'PY '; rg --files -g '*.py' | wc -l; printf 'GO '; rg --files -g '*.go' | wc -l; printf 'SRC '; rg --files src/bigclaw -g '*.py' | wc -l`
    - Result: `PY 79`, `GO 267`, `SRC 16`

## BIG-GO-1014 Refill Sweep D Continuation 4

### Plan

- Fold the low-coupling connector stub surface from `src/bigclaw/connectors.py`
  into `src/bigclaw/models.py`.
- Install compatibility submodules from `src/bigclaw/__init__.py` so
  `bigclaw.connectors` and `bigclaw.mapping` still resolve after file removal.
- Validate via direct module loading and syntax checks, avoiding unrelated
  package-import failures from other dirty files in the shared worktree.

### Acceptance

- Reduce `src/bigclaw/*.py` by one more file.
- Preserve exported connector and mapping symbols after the merge.
- Record exact validation commands and results.

### Validation

- `python3 -m compileall src/bigclaw/models.py src/bigclaw/__init__.py`
- `python3 - <<'PY' ... PY` direct-load `src/bigclaw/models.py` and assert connector symbols work
- `printf 'PY '; rg --files -g '*.py' | wc -l; printf 'GO '; rg --files -g '*.go' | wc -l; printf 'SRC '; rg --files src/bigclaw -g '*.py' | wc -l`

### Results

- Deleted `src/bigclaw/connectors.py`.
- Merged connector stub types and mapping helpers into
  `src/bigclaw/models.py`.
- Updated `src/bigclaw/__init__.py` to expose compatibility `connectors` and
  `mapping` submodules from the merged `models` surface.
- Repository counts after continuation:
  - total `py` files: `78`
  - total `go` files: `267`
  - `src/bigclaw/*.py` files: `15`
- Validation outcomes:
  - `python3 -m compileall src/bigclaw/models.py src/bigclaw/__init__.py`
    - Result: success
  - `python3 - <<'PY' ... PY`
    - Result: direct-loaded `models.py`, built a `SourceIssue`, mapped it to a
      `Task`, and exercised `GitHubConnector.fetch_issues()` successfully;
      printed `demo#1`, `demo#1`, `Priority.P0`, `In Progress`, `github`
  - `printf 'PY '; rg --files -g '*.py' | wc -l; printf 'GO '; rg --files -g '*.go' | wc -l; printf 'SRC '; rg --files src/bigclaw -g '*.py' | wc -l`
    - Result: `PY 78`, `GO 267`, `SRC 15`

## BIG-GO-1014 Refill Sweep D Continuation 5

### Plan

- Merge the low-coupling risk and cost-control helpers from `src/bigclaw/risk.py`
  into `src/bigclaw/models.py`.
- Update `src/bigclaw/runtime.py` and `src/bigclaw/__init__.py` to source the
  moved symbols from `models.py`, then delete `risk.py`.
- Validate with syntax checks, import-reference grep, and direct module loading
  to avoid unrelated package-import failures from other dirty files.

### Acceptance

- Reduce `src/bigclaw/*.py` by one more file.
- Preserve `RiskScorer`, `RiskScore`, `BudgetDecision`, `CostController`, and
  `RiskFactor` exports after the merge.
- Record exact validation commands and results.

### Validation

- `python3 -m compileall src/bigclaw/models.py src/bigclaw/runtime.py src/bigclaw/__init__.py`
- `python3 - <<'PY' ... PY` direct-load `src/bigclaw/models.py` and exercise `RiskScorer`/`CostController`
- `rg -n "from \\.risk import|from bigclaw\\.risk|RiskScorer|CostController" src tests docs bigclaw-go -g '*.py' -g '*.md' -g '*.go'`

### Results

- Deleted `src/bigclaw/risk.py`.
- Merged the risk and cost-control helpers into `src/bigclaw/models.py`.
- Updated `src/bigclaw/runtime.py` and `src/bigclaw/__init__.py` to source the
  moved symbols from `models.py`, and installed a compatibility `bigclaw.risk`
  surface from the merged module.
- Repository counts after continuation:
  - total `py` files: `77`
  - total `go` files: `267`
  - `src/bigclaw/*.py` files: `14`
- Validation outcomes:
  - `python3 -m compileall src/bigclaw/models.py src/bigclaw/runtime.py src/bigclaw/__init__.py`
    - Result: success
  - `python3 - <<'PY' ... PY`
    - Result: direct-loaded `models.py`, scored a high-risk task, and evaluated
      budget control successfully; printed `high`, `70`, `True`, `allow`, `3.0`
  - `rg -n "from \\.risk import|from bigclaw\\.risk|RiskScorer|CostController" src tests docs bigclaw-go -g '*.py' -g '*.md' -g '*.go'`
    - Result: live imports now point at `models.py`/compatibility surfaces;
      only test/docs references to `bigclaw.risk` remain as expected

## BIG-GO-1014 Refill Sweep D Continuation 6

### Plan

- Merge the console IA contracts from `src/bigclaw/console_ia.py` into
  `src/bigclaw/design_system.py`.
- Update `src/bigclaw/__init__.py` to import the moved symbols from
  `design_system.py` and install a compatibility `bigclaw.console_ia` surface.
- Validate with syntax checks and direct loading of `design_system.py`, keeping
  the verification isolated from unrelated package-level dirty imports.

### Acceptance

- Reduce `src/bigclaw/*.py` by one more file.
- Preserve all `ConsoleIA*` and console interaction exports after the merge.
- Record exact validation commands and results.

### Validation

- `python3 -m compileall src/bigclaw/design_system.py src/bigclaw/__init__.py`
- `python3 - <<'PY' ... PY` direct-load `src/bigclaw/design_system.py` and exercise `ConsoleIAAuditor`
- `printf 'PY '; rg --files -g '*.py' | wc -l; printf 'GO '; rg --files -g '*.go' | wc -l; printf 'SRC '; rg --files src/bigclaw -g '*.py' | wc -l`

### Results

- Deleted `src/bigclaw/console_ia.py`.
- Merged the console IA models, auditors, reports, and draft builder into
  `src/bigclaw/design_system.py`.
- Updated `src/bigclaw/__init__.py` to install a compatibility
  `bigclaw.console_ia` submodule backed by `design_system.py`.
- Repository counts after continuation:
  - total `py` files: `75`
  - total `go` files: `267`
  - `src/bigclaw/*.py` files: `12`
- Validation outcomes:
  - `python3 -m compileall src/bigclaw/design_system.py src/bigclaw/__init__.py`
    - Result: success
  - `python3 - <<'PY' ... PY`
    - Result: direct-loaded `design_system.py`, built a minimal `ConsoleIA`,
      audited it, and rendered a console IA report successfully; printed `1`,
      `1`, `0.0`, `True`
  - `printf 'PY '; rg --files -g '*.py' | wc -l; printf 'GO '; rg --files -g '*.go' | wc -l; printf 'SRC '; rg --files src/bigclaw -g '*.py' | wc -l`
    - Result: `PY 75`, `GO 267`, `SRC 12`

## BIG-GO-1014 Refill Sweep D Continuation 7

### Plan

- Merge the isolated workspace bootstrap helpers from
  `src/bigclaw/workspace_bootstrap.py` into `src/bigclaw/models.py`.
- Install a compatibility `bigclaw.workspace_bootstrap` submodule from
  `src/bigclaw/__init__.py` and retarget the direct-load bootstrap tests to the
  merged module.
- Validate via syntax checks and the focused workspace bootstrap pytest slice.

### Acceptance

- Reduce `src/bigclaw/*.py` by one more file.
- Preserve the bootstrap helper API after the move.
- Record exact validation commands and results.

### Validation

- `python3 -m compileall src/bigclaw/models.py src/bigclaw/__init__.py tests/test_workspace_bootstrap.py`
- `PYTHONPATH=src python3 -m pytest tests/test_workspace_bootstrap.py -q`
- `printf 'PY '; rg --files -g '*.py' | wc -l; printf 'GO '; rg --files -g '*.go' | wc -l; printf 'SRC '; rg --files src/bigclaw -g '*.py' | wc -l`

### Results

- Deleted `src/bigclaw/workspace_bootstrap.py`.
- Merged the workspace bootstrap/cache validation helpers into
  `src/bigclaw/models.py`.
- Updated `src/bigclaw/__init__.py` to install a compatibility
  `bigclaw.workspace_bootstrap` submodule from the merged `models` surface.
- Retargeted `tests/test_workspace_bootstrap.py` to direct-load
  `src/bigclaw/models.py`.
- Repository counts after continuation:
  - total `py` files: `76`
  - total `go` files: `267`
  - `src/bigclaw/*.py` files: `13`
- Validation outcomes:
  - `python3 -m compileall src/bigclaw/models.py src/bigclaw/__init__.py tests/test_workspace_bootstrap.py`
    - Result: success
  - `PYTHONPATH=src python3 -m pytest tests/test_workspace_bootstrap.py -q`
    - Result: `9 passed in 2.99s`
  - `printf 'PY '; rg --files -g '*.py' | wc -l; printf 'GO '; rg --files -g '*.go' | wc -l; printf 'SRC '; rg --files src/bigclaw -g '*.py' | wc -l`
    - Result: `PY 76`, `GO 267`, `SRC 13`

## BIG-GO-1014 Refill Sweep D Continuation 8

### Plan

- Merge the benchmark/replay surface from `src/bigclaw/evaluation.py` into
  `src/bigclaw/operations.py`.
- Update `src/bigclaw/__init__.py` to source evaluation exports from
  `operations.py` and install a compatibility `bigclaw.evaluation` submodule.
- Validate with syntax checks plus a direct-load benchmark script that avoids
  the shared package import path.

### Acceptance

- Reduce `src/bigclaw/*.py` by one more file.
- Preserve benchmark/replay exports after the move.
- Record exact validation commands and results.

### Validation

- `python3 -m compileall src/bigclaw/operations.py src/bigclaw/__init__.py`
- `python3 - <<'PY' ... PY` fake-package load `bigclaw.operations` and exercise `BenchmarkSuiteResult`/`render_benchmark_suite_report`
- `printf 'PY '; rg --files -g '*.py' | wc -l; printf 'GO '; rg --files -g '*.go' | wc -l; printf 'SRC '; rg --files src/bigclaw -g '*.py' | wc -l`

### Results

- Deleted `src/bigclaw/evaluation.py`.
- Merged the benchmark models, runner, and replay/report render helpers into
  `src/bigclaw/operations.py`.
- Updated `src/bigclaw/__init__.py` to source evaluation exports from
  `operations.py` and install a compatibility `bigclaw.evaluation` submodule.
- Repository counts after continuation:
  - total `py` files: `74`
  - total `go` files: `267`
  - `src/bigclaw/*.py` files: `11`
- Root manifest impact:
  - `pyproject.toml`: absent
  - `setup.py`: absent
- Validation outcomes:
  - `python3 -m compileall src/bigclaw/operations.py src/bigclaw/__init__.py`
    - Result: success
  - `python3 - <<'PY' ... PY`
    - Result: fake-package loaded `bigclaw.operations`, executed
      `BenchmarkRunner.run_case`, rendered the benchmark suite report, replay
      detail page, and run replay index page successfully; printed
      `evaluation merge validation ok`
  - `printf 'PY '; rg --files -g '*.py' | wc -l; printf 'GO '; rg --files -g '*.go' | wc -l; printf 'SRC '; rg --files src/bigclaw -g '*.py' | wc -l`
    - Result: `PY 74`, `GO 267`, `SRC 11`
  - `printf 'pyproject='; test -f pyproject.toml; echo $?; printf 'setup='; test -f setup.py; echo $?`
    - Result: `pyproject=1`, `setup=1`
  - `git diff --check`
    - Result: clean
