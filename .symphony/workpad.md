# BIG-GO-1021

## Plan
- Inventory root/config/packaging-related Python assets, with emphasis on repo-root entrypoints and `scripts/ops`.
- Replace remaining root-level operational Python entrypoints with non-Python wrappers or existing Go binaries where possible.
- Validate targeted commands and measure repository `*.py` / `*.go` counts plus packaging-file impact.
- Commit scoped changes and push the issue branch.

## Acceptance
- Repository physical-layer Python residuals are reduced within this issue scope.
- Root/config/python residuals are addressed without using tracker-only closure.
- Report includes `*.py` / `*.go` counts and confirms `pyproject/setup/egg-info` impact.
- Targeted validation commands and exact results are recorded.

## Validation
- `find . -path './.git' -prune -o -name '*.py' -print | wc -l`
- `find . -path './.git' -prune -o -name '*.go' -print | wc -l`
- Targeted execution of affected operational entrypoints and their tests, based on changed files.

## Results
- `cd bigclaw-go && go test ./internal/legacyshim ./cmd/bigclawctl` -> `ok  	bigclaw-go/internal/legacyshim	1.098s` and `ok  	bigclaw-go/cmd/bigclawctl	5.977s`
- `bash scripts/ops/bigclawctl legacy-python compile-check --json` -> `status: ok` for `src/bigclaw/__main__.py` and `src/bigclaw/__init__.py`
- `bash scripts/ops/bigclawctl github-sync --help` -> `usage: bigclawctl github-sync <install|status|sync> [flags]`
- `bash scripts/ops/bigclawctl workspace validate --help` -> `usage: bigclawctl workspace validate [flags]`
- `git ls-tree -r --name-only HEAD | rg '\.py$' | wc -l` -> `88`; `find . -path './.git' -prune -o -name '*.py' -print | wc -l` -> `82`
- `git ls-tree -r --name-only HEAD | rg '\.go$' | wc -l` -> `282`; `find . -path './.git' -prune -o -name '*.go' -print | wc -l` -> `282`
- `git ls-tree -r --name-only HEAD | rg '(^|/)(pyproject\.toml|setup\.py|setup\.cfg|[^/]+\.egg-info|PKG-INFO)$' | wc -l` -> `0`; `find . -path './.git' -prune -o \( -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' -o -name '*.egg-info' -o -name 'PKG-INFO' \) -print | wc -l` -> `0`
- `cd bigclaw-go && go test ./internal/regression -run TestLegacyMainlineCompatibilityManifestStaysAligned -count=1` -> `ok  	bigclaw-go/internal/regression	1.218s`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/workspace_bootstrap.py src/bigclaw/workspace_bootstrap_validation.py src/bigclaw/github_sync.py` -> success
- `find . -path './.git' -prune -o -name '*.py' -print | wc -l` -> `81`
- `cd bigclaw-go && go test ./internal/legacyshim ./cmd/bigclawctl ./internal/regression -run 'TestLegacyMainlineCompatibilityManifestStaysAligned|TestRunLegacyPythonCompileCheckJSONOutputDoesNotEscapeArrowTokens|TestFrozenCompileCheckFilesUsesFrozenShimList|TestCompileCheckRunsPyCompileAgainstFrozenShimList|TestCompileCheckReturnsCompilerOutputOnFailure' -count=1` -> `ok  	bigclaw-go/internal/legacyshim	1.087s`, `ok  	bigclaw-go/cmd/bigclawctl	2.237s`, `ok  	bigclaw-go/internal/regression	1.489s`
- `bash scripts/ops/bigclawctl legacy-python compile-check --json` -> `status: ok` for `src/bigclaw/__init__.py`
- `find . -path './.git' -prune -o -name '*.py' -print | wc -l` -> `80`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/runtime.py src/bigclaw/workspace_bootstrap.py src/bigclaw/workspace_bootstrap_validation.py src/bigclaw/github_sync.py` -> success
- `find . -path './.git' -prune -o -name '*.py' -print | wc -l` -> `79` after retiring `src/bigclaw/cost_control.py`
- `find . -path './.git' -prune -o -name '*.py' -print | wc -l` -> `78` after retiring `src/bigclaw/parallel_refill.py`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/runtime.py src/bigclaw/repo_gateway.py src/bigclaw/workspace_bootstrap.py src/bigclaw/workspace_bootstrap_validation.py src/bigclaw/github_sync.py` -> success
- `find . -path './.git' -prune -o -name '*.py' -print | wc -l` -> `77` after retiring `src/bigclaw/issue_archive.py`
- `cd bigclaw-go && go test ./internal/repo -count=1` -> `ok  	bigclaw-go/internal/repo	0.782s`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/runtime.py src/bigclaw/repo_commits.py src/bigclaw/workspace_bootstrap.py src/bigclaw/workspace_bootstrap_validation.py src/bigclaw/github_sync.py` -> success
- `find . -path './.git' -prune -o -name '*.py' -print | wc -l` -> `75` after retiring `src/bigclaw/repo_gateway.py` and `tests/test_repo_gateway.py`
- `cd bigclaw-go && go test ./internal/product -run 'TestBuildDefaultDashboardRunContractIsReleaseReady|TestDashboardRunContractAuditDetectsMissingPaths|TestRenderDashboardRunContractReport' -count=1` -> `ok  	bigclaw-go/internal/product	0.446s`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/runtime.py src/bigclaw/reports.py src/bigclaw/operations.py src/bigclaw/run_detail.py src/bigclaw/workspace_bootstrap.py src/bigclaw/workspace_bootstrap_validation.py src/bigclaw/github_sync.py` -> success
- `find . -path './.git' -prune -o -name '*.py' -print | wc -l` -> `71` after retiring `src/bigclaw/dashboard_run_contract.py`, `src/bigclaw/validation_policy.py`, `tests/test_dashboard_run_contract.py`, and `tests/test_validation_policy.py`
- `cd bigclaw-go && go test ./internal/repo -count=1` -> `ok  	bigclaw-go/internal/repo	1.149s`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/runtime.py src/bigclaw/repo_plane.py src/bigclaw/repo_commits.py src/bigclaw/repo_links.py src/bigclaw/workspace_bootstrap.py src/bigclaw/workspace_bootstrap_validation.py src/bigclaw/github_sync.py` -> success
- `find . -path './.git' -prune -o -name '*.py' -print | wc -l` -> `67` after retiring `src/bigclaw/repo_governance.py`, `src/bigclaw/repo_registry.py`, `tests/test_repo_governance.py`, and `tests/test_repo_registry.py`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/runtime.py src/bigclaw/repo_plane.py src/bigclaw/repo_commits.py src/bigclaw/repo_links.py src/bigclaw/workspace_bootstrap.py src/bigclaw/workspace_bootstrap_validation.py src/bigclaw/github_sync.py` -> success
- `find . -path './.git' -prune -o -name '*.py' -print | wc -l` -> `66` after retiring `src/bigclaw/roadmap.py`
- `cd bigclaw-go && go test ./internal/repo -count=1` -> `ok  	bigclaw-go/internal/repo	0.727s`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/runtime.py src/bigclaw/repo_plane.py src/bigclaw/repo_links.py src/bigclaw/workspace_bootstrap.py src/bigclaw/github_sync.py tests/test_workspace_bootstrap.py` -> success
- `find . -path './.git' -prune -o -name '*.py' -print | wc -l` -> `64` after retiring `src/bigclaw/repo_commits.py` and `src/bigclaw/workspace_bootstrap_validation.py`
- `cd bigclaw-go && go test ./internal/githubsync -count=1` -> `ok  	bigclaw-go/internal/githubsync	3.612s`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/runtime.py src/bigclaw/repo_plane.py src/bigclaw/repo_links.py src/bigclaw/workspace_bootstrap.py tests/test_workspace_bootstrap.py` -> success
- `find . -path './.git' -prune -o -name '*.py' -print | wc -l` -> `62` after retiring `src/bigclaw/github_sync.py` and `tests/test_github_sync.py`
- `cd bigclaw-go && go test ./internal/repo ./internal/triage -count=1` -> `ok  	bigclaw-go/internal/repo	0.620s` and `ok  	bigclaw-go/internal/triage	1.032s`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/runtime.py src/bigclaw/repo_plane.py src/bigclaw/repo_links.py src/bigclaw/workspace_bootstrap.py tests/test_workspace_bootstrap.py` -> success
- `find . -path './.git' -prune -o -name '*.py' -print | wc -l` -> `60` after retiring `src/bigclaw/repo_triage.py` and `tests/test_repo_triage.py`
- `cd bigclaw-go && go test ./internal/intake ./internal/workflow -count=1` -> `ok  	bigclaw-go/internal/intake	0.850s` and `ok  	bigclaw-go/internal/workflow	1.304s`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/runtime.py src/bigclaw/repo_plane.py src/bigclaw/repo_links.py src/bigclaw/workspace_bootstrap.py tests/test_workspace_bootstrap.py` -> success
- `find . -path './.git' -prune -o -name '*.py' -print | wc -l` -> `56` after retiring `src/bigclaw/connectors.py`, `src/bigclaw/dsl.py`, `src/bigclaw/mapping.py`, and `tests/test_dsl.py`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/runtime.py src/bigclaw/repo_plane.py src/bigclaw/repo_links.py src/bigclaw/workspace_bootstrap.py tests/test_workspace_bootstrap.py` -> success
- `find . -path './.git' -prune -o -name '*.py' -print | wc -l` -> `55` after retiring `src/bigclaw/deprecation.py`
- `cd bigclaw-go && go test ./internal/repo -count=1` -> `ok  	bigclaw-go/internal/repo	0.882s`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/runtime.py src/bigclaw/collaboration.py src/bigclaw/repo_plane.py src/bigclaw/repo_links.py src/bigclaw/workspace_bootstrap.py tests/test_repo_collaboration.py tests/test_workspace_bootstrap.py` -> success
- `find . -path './.git' -prune -o -name '*.py' -print | wc -l` -> `53` after retiring `src/bigclaw/repo_board.py` and `tests/test_repo_board.py`
- `rg -n "repo_plane|repo_links|RunCommitLink|bind_run_commits" -S src tests docs` -> only `src/bigclaw/observability.py`, affected tests, and migration docs still referenced the run-commit compatibility surface before this slice
- `PYTHONPATH=src python3 -m pytest tests/test_repo_links.py tests/test_observability.py tests/test_reports.py -q` -> `42 passed in 0.18s`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/observability.py src/bigclaw/reports.py src/bigclaw/workspace_bootstrap.py tests/test_repo_links.py tests/test_observability.py` -> success
- `cd bigclaw-go && go test ./internal/repo -count=1` -> `ok  	bigclaw-go/internal/repo	0.464s`
- `find . -path './.git' -prune -o -name '*.py' -print | wc -l` -> `51` after retiring `src/bigclaw/repo_plane.py` and `src/bigclaw/repo_links.py`
- `find . -path './.git' -prune -o -name '*.go' -print | wc -l` -> `282`
- `find . -path './.git' -prune -o \( -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' -o -name '*.egg-info' -o -name 'PKG-INFO' \) -print | wc -l` -> `0`
- `rg -n "bigclaw\.collaboration|src/bigclaw/collaboration.py|from \.collaboration|from bigclaw\.collaboration" src tests docs -S` -> no matches after folding collaboration helpers into `src/bigclaw/observability.py`
- `PYTHONPATH=src python3 -m pytest tests/test_repo_collaboration.py tests/test_reports.py tests/test_observability.py tests/test_repo_links.py -q` -> `43 passed in 0.15s`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/observability.py src/bigclaw/reports.py tests/test_repo_collaboration.py tests/test_reports.py tests/test_observability.py tests/test_repo_links.py` -> success
- `find . -path './.git' -prune -o -name '*.py' -print | wc -l` -> `50` after retiring `src/bigclaw/collaboration.py`
- `find . -path './.git' -prune -o -name '*.go' -print | wc -l` -> `282`
- `find . -path './.git' -prune -o \( -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' -o -name '*.egg-info' -o -name 'PKG-INFO' \) -print | wc -l` -> `0`

## 2026-03-31 Sweep A Addendum

### Plan
- Inspect repository-root and config-layer Python residuals, with emphasis on packaging entrypoints and root-level workflow/docs references.
- Remove or rewrite root/config references that still imply `pyproject.toml`, `setup.py`, editable installs, or Python-first execution where the Go surface should be primary.
- Record repository file-count impact for `.py` and `.go`, then run targeted validation for the touched surfaces.
- Commit scoped changes and push the issue branch.

### Acceptance
- Repository-root/config residuals for Python packaging are reduced without widening scope beyond this issue.
- Root and config surfaces no longer present stale `pyproject/setup/egg-info` style entrypoint guidance.
- Report includes `.py`/`.go` counts and the effect on `pyproject/setup` presence.
- Validation is executed with exact commands and results captured.

### Validation
- `printf 'py '; find . -path './.git' -prune -o -name '*.py' -print | wc -l; printf 'go '; find . -path './.git' -prune -o -name '*.go' -print | wc -l`
- `find . -maxdepth 2 \( -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' -o -name '*.egg-info' \) -print`
- `PYTHONPATH=src python3 -m pytest tests/test_workspace_bootstrap.py tests/test_planning.py -q`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./cmd/bigclawd`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation --help`
- `rg -n "bigclaw-go/scripts/.+\\.py" README.md docs/go-cli-script-migration-plan.md -S`
- `rg -n "PilotKPI|PilotImplementationResult|render_pilot_implementation_report|from bigclaw\\.pilot|bigclaw\\.pilot" src tests README.md docs reports scripts -S`
- `cd bigclaw-go && go test ./internal/pilot -run 'TestImplementationResultReadyWhenKPIsPassAndNoIncidents|TestRenderPilotImplementationReportContainsReadinessFields'`
- `rg -n "TaskMemoryStore|MemoryPattern|from bigclaw\\.memory|bigclaw\\.memory" src tests README.md docs reports scripts bigclaw-go -S`
- `cd bigclaw-go && go test ./internal/memory -run TestTaskStoreReusesHistoryAndInjectsRules`
- `cd bigclaw-go && go test ./internal/bootstrap -run 'TestCacheRootForRepoUsesRepoSpecificDirectory|TestSecondWorkspaceReusesWarmCacheWithoutFullClone|TestBootstrapWorkspaceReusesExistingIssueWorktree|TestCleanupWorkspacePreservesSharedCacheForFutureReuse|TestBootstrapRecoversFromStaleSeedDirectoryWithoutRemoteReclone'`
- `rg -n "from bigclaw\\.workspace_bootstrap|bigclaw\\.workspace_bootstrap|workspace_bootstrap import" src tests README.md docs reports scripts -S`
- `cd bigclaw-go && go test ./internal/product -run 'TestAuditSavedViewCatalogAndRenderReport|TestSavedViewCatalogJSONRoundTrip|TestRenderSavedViewReportEmptyState|TestRenderSavedViewReportPopulatedRowsUseFallbacks'`
- `rg -n "from bigclaw\\.saved_views|bigclaw\\.saved_views|SavedViewLibrary|render_saved_view_report" src tests README.md docs reports scripts -S`
- `cd bigclaw-go && go test ./internal/eventbus -run 'TestEventBusPRCommentApprovesWaitingRunAndPersistsLedger|TestEventBusCICompletedMarksRunCompleted|TestEventBusTaskFailedMarksRunFailed'`
- `rg -n "from bigclaw\\.event_bus|bigclaw\\.event_bus|EventBus|BusEvent|PULL_REQUEST_COMMENT_EVENT|CI_COMPLETED_EVENT|TASK_FAILED_EVENT" src tests README.md docs reports scripts -S`
- `cd bigclaw-go && go test ./internal/contract -run 'TestExecutionContractAuditAcceptsWellFormedContract|TestExecutionContractAuditSurfacesContractGaps|TestExecutionContractRoundTripAndPermissionMatrix|TestRenderExecutionContractReportIncludesRoleMatrix|TestOperationsAPIContractDraftIsReleaseReady|TestOperationsAPIContractPermissionsCoverReadAndActionPaths|TestExecutionContractAuditRequiresPersonaScopeAndEscalationMetadata'`
- `rg -n "from bigclaw\\.execution_contract|bigclaw\\.execution_contract|ExecutionContractLibrary|render_execution_contract_report|build_operations_api_contract|ExecutionPermissionMatrix|ExecutionRole" src tests README.md docs reports scripts -S`
- `cd bigclaw-go && go test ./internal/risk ./internal/scheduler`
- `rg -n "from bigclaw\\.risk|RiskScorer|Scheduler\\(\\)\\.execute|test_risk\\.py" src tests README.md docs reports scripts -S`
- `python3 - <<'PY' ... assert 'PYTHONPATH=src python3 -m pytest' in .github/workflows/ci.yml ... PY`
- `rg -n "pyproject|setup.py|egg-info|pip install -e|python -m build|setuptools" -S README.md .github/workflows/ci.yml scripts/dev_bootstrap.sh reports/BIG-GO-1021.md`
