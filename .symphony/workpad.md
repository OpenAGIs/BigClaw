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
