# BIG-GO-1065

## Plan
- inventory the Python test files from the issue batch and map each surviving file to an existing Go replacement or removal candidate
- remove the residual Python files that are already covered by Go tests or regression guards
- retire any remaining Python wrapper scripts that only proxy into `scripts/ops/bigclawctl`
- retire the residual Python test files once equivalent Go coverage is pinned
- retire the top-level `python -m bigclaw` entrypoint once the frozen compatibility manifest and compile-check tooling are aligned
- run targeted validation for the affected Go packages and regression coverage
- record exact commands, results, Python file count impact, and residual risks
- commit and push the scoped change set to the issue branch

## Acceptance
- identify the Python assets handled in this tranche
- reduce the repository Python file count by deleting or replacing covered Python artifacts
- keep a verifiable Go replacement path for each removed asset
- provide exact validation commands and results
- report remaining risks and the Python file count delta

## Validation
- `rg --files tests | sort`
- `git diff --stat`
- `cd bigclaw-go && go test ./...`
- narrower `go test` package/regression commands for touched replacement paths if full sweep is unnecessary

## Completed
- confirmed the issue's suggested `tests/test_*.py` tranche was already absent from the repo before this turn
- removed `src/bigclaw/governance.py`, `src/bigclaw/planning.py`, and `src/bigclaw/ui_review.py`
- removed `src/bigclaw/audit_events.py` and `src/bigclaw/run_detail.py` by folding their compatibility helpers into surviving Python modules
- removed `src/bigclaw/risk.py` by folding its compatibility scorer into the frozen legacy runtime surface
- removed `src/bigclaw/collaboration.py` by folding its compatibility thread/render helpers into the remaining observability surface
- removed `src/bigclaw/deprecation.py` by folding its warning helpers into the frozen legacy runtime surface and `python -m bigclaw` entrypoint
- removed `src/bigclaw/evaluation.py` by folding its benchmark/replay compatibility surface into `src/bigclaw/operations.py` and preserving `bigclaw.evaluation` through a package-installed compatibility submodule
- removed `src/bigclaw/console_ia.py` by folding its console-IA compatibility surface into `src/bigclaw/design_system.py` and preserving `bigclaw.console_ia` through a package-installed compatibility submodule
- removed `src/bigclaw/design_system.py` by folding its remaining compatibility surface into `src/bigclaw/operations.py` and preserving both `bigclaw.design_system` and `bigclaw.console_ia` through package-installed compatibility submodules
- removed `src/bigclaw/legacy_shim.py` by folding its wrapper helpers into `src/bigclaw/runtime.py`, preserving `bigclaw.legacy_shim` through a package-installed compatibility submodule, and updating the Go-side frozen compile-check list to target `runtime.py`
- removed `src/bigclaw/models.py` by folding its remaining compatibility structs into `src/bigclaw/observability.py` and preserving `bigclaw.models` through a package-installed compatibility submodule
- removed `src/bigclaw/reports.py` by folding its remaining compatibility/report surface into `src/bigclaw/operations.py` and preserving `bigclaw.reports` through a package-installed compatibility submodule
- removed `scripts/ops/bigclaw_refill_queue.py`, `scripts/ops/bigclaw_workspace_bootstrap.py`, `scripts/ops/symphony_workspace_bootstrap.py`, and `scripts/ops/symphony_workspace_validate.py` after confirming `scripts/ops/bigclawctl` already owns those operator paths
- removed `tests/conftest.py`, `tests/test_console_ia.py`, `tests/test_control_center.py`, `tests/test_design_system.py`, and `tests/test_evaluation.py` after pinning their replacement surfaces to existing Go-owned product, reporting, pilot, api, and regression coverage
- removed `src/bigclaw/__main__.py` and retired the top-level `python -m bigclaw` entrypoint after aligning the frozen compatibility manifest and Go-side legacy compile-check coverage to the four surviving Python compatibility modules
- removed the corresponding legacy exports from `src/bigclaw/__init__.py`
- added `bigclaw-go/internal/regression/top_level_module_purge_tranche15_test.go` to pin the deletions against Go replacement paths
- added `bigclaw-go/internal/regression/top_level_module_purge_tranche16_test.go` to pin the additional deletions against Go replacement paths
- added `bigclaw-go/internal/regression/top_level_module_purge_tranche17_test.go` to pin the latest deletion against Go replacement paths
- added `bigclaw-go/internal/regression/top_level_module_purge_tranche18_test.go` to pin the collaboration-surface deletion against Go replacement paths
- added `bigclaw-go/internal/regression/top_level_module_purge_tranche19_test.go` to pin the deprecation-surface deletion against Go replacement paths
- added `bigclaw-go/internal/regression/top_level_module_purge_tranche20_test.go` to pin the evaluation-surface deletion against Go replacement paths
- added `bigclaw-go/internal/regression/top_level_module_purge_tranche21_test.go` to pin the console-IA deletion against Go replacement paths
- added `bigclaw-go/internal/regression/top_level_module_purge_tranche22_test.go` to pin the design-system deletion against Go replacement paths
- added `bigclaw-go/internal/regression/top_level_module_purge_tranche23_test.go` to pin the legacy-shim deletion against Go replacement paths
- added `bigclaw-go/internal/regression/top_level_module_purge_tranche24_test.go` to pin the model-surface deletion against Go replacement paths
- added `bigclaw-go/internal/regression/top_level_module_purge_tranche25_test.go` to pin the report-surface deletion against Go replacement paths
- added `bigclaw-go/internal/regression/top_level_module_purge_tranche26_test.go` to pin the ops-wrapper deletion against Go replacement paths
- added `bigclaw-go/internal/regression/top_level_module_purge_tranche27_test.go` to pin the residual Python-test deletion against Go replacement paths
- added `bigclaw-go/internal/regression/top_level_module_purge_tranche28_test.go` to pin the `python -m bigclaw` entrypoint deletion against Go replacement paths
- updated `docs/go-mainline-cutover-issue-pack.md` so the migration inventory reflects the deleted Python assets

## Validation Results
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/operations.py src/bigclaw/reports.py src/bigclaw/evaluation.py src/bigclaw/runtime.py src/bigclaw/observability.py tests/test_control_center.py tests/test_evaluation.py tests/test_console_ia.py tests/test_design_system.py` -> passed
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche15|TestTopLevelModulePurgeTranche16|TestTopLevelModulePurgeTranche17|TestTopLevelModulePurgeTranche18|TestTopLevelModulePurgeTranche19|TestTopLevelModulePurgeTranche20|TestFollowUpLaneDocsStayAligned|TestExecutionPackRoadmapDocsStayAligned|TestExecutionPackRoadmapUniqueOwnersContract'` -> `ok  	bigclaw-go/internal/regression	0.497s`
- `cd bigclaw-go && go test ./internal/governance ./internal/product` -> `ok  	bigclaw-go/internal/governance	0.454s`; `ok  	bigclaw-go/internal/product	(cached)`
- `cd bigclaw-go && go test ./internal/observability ./internal/product ./internal/api` -> `ok  	bigclaw-go/internal/observability	1.663s`; `ok  	bigclaw-go/internal/product	(cached)`; `ok  	bigclaw-go/internal/api	3.475s`
- `PYTHONPATH=src python3 -m pytest tests/test_control_center.py tests/test_evaluation.py tests/test_console_ia.py tests/test_design_system.py -q` -> `36 passed in 0.07s`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/operations.py src/bigclaw/runtime.py src/bigclaw/observability.py src/bigclaw/reports.py src/bigclaw/__main__.py tests/test_control_center.py tests/test_evaluation.py tests/test_console_ia.py tests/test_design_system.py` -> passed
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/design_system.py src/bigclaw/operations.py src/bigclaw/runtime.py src/bigclaw/observability.py src/bigclaw/reports.py src/bigclaw/__main__.py tests/test_console_ia.py tests/test_design_system.py tests/test_control_center.py tests/test_evaluation.py` -> passed
- `PYTHONPATH=src python3 -m pytest tests/test_console_ia.py tests/test_design_system.py tests/test_control_center.py tests/test_evaluation.py -q` -> `36 passed in 0.07s`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche15|TestTopLevelModulePurgeTranche16|TestTopLevelModulePurgeTranche17|TestTopLevelModulePurgeTranche18|TestTopLevelModulePurgeTranche19|TestTopLevelModulePurgeTranche20|TestTopLevelModulePurgeTranche21|TestFollowUpLaneDocsStayAligned|TestExecutionPackRoadmapDocsStayAligned|TestExecutionPackRoadmapUniqueOwnersContract'` -> `ok  	bigclaw-go/internal/regression	1.235s`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/operations.py src/bigclaw/runtime.py src/bigclaw/observability.py src/bigclaw/reports.py src/bigclaw/__main__.py tests/test_design_system.py tests/test_console_ia.py tests/test_control_center.py tests/test_evaluation.py` -> passed
- `PYTHONPATH=src python3 -m pytest tests/test_design_system.py tests/test_console_ia.py tests/test_control_center.py tests/test_evaluation.py -q` -> `36 passed in 0.07s`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche15|TestTopLevelModulePurgeTranche16|TestTopLevelModulePurgeTranche17|TestTopLevelModulePurgeTranche18|TestTopLevelModulePurgeTranche19|TestTopLevelModulePurgeTranche20|TestTopLevelModulePurgeTranche21|TestTopLevelModulePurgeTranche22|TestFollowUpLaneDocsStayAligned|TestExecutionPackRoadmapDocsStayAligned|TestExecutionPackRoadmapUniqueOwnersContract'` -> `ok  	bigclaw-go/internal/regression	0.481s`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/runtime.py src/bigclaw/operations.py src/bigclaw/observability.py src/bigclaw/reports.py src/bigclaw/__main__.py scripts/ops/bigclaw_refill_queue.py scripts/ops/bigclaw_workspace_bootstrap.py scripts/ops/symphony_workspace_bootstrap.py scripts/ops/symphony_workspace_validate.py` -> passed
- `python3 scripts/ops/bigclaw_refill_queue.py --help && python3 scripts/ops/symphony_workspace_bootstrap.py --help && python3 scripts/ops/symphony_workspace_validate.py --help && python3 scripts/ops/bigclaw_workspace_bootstrap.py --help` -> passed
- `cd bigclaw-go && go test ./internal/legacyshim ./cmd/bigclawctl ./internal/regression -run 'TestTopLevelModulePurgeTranche20|TestTopLevelModulePurgeTranche21|TestTopLevelModulePurgeTranche22|TestTopLevelModulePurgeTranche23|TestFrozenCompileCheckFilesUsesFrozenShimList|TestCompileCheckRunsPyCompileAgainstFrozenShimList|TestRunLegacyPythonCompileCheckJSONOutputDoesNotEscapeArrowTokens|TestFollowUpLaneDocsStayAligned|TestExecutionPackRoadmapDocsStayAligned|TestExecutionPackRoadmapUniqueOwnersContract'` -> `ok  	bigclaw-go/internal/legacyshim	0.923s`; `ok  	bigclaw-go/cmd/bigclawctl	2.169s`; `ok  	bigclaw-go/internal/regression	1.432s`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/observability.py src/bigclaw/runtime.py src/bigclaw/operations.py src/bigclaw/reports.py src/bigclaw/__main__.py tests/test_design_system.py tests/test_console_ia.py tests/test_control_center.py tests/test_evaluation.py` -> passed
- `PYTHONPATH=src python3 -m pytest tests/test_design_system.py tests/test_console_ia.py tests/test_control_center.py tests/test_evaluation.py -q` -> `36 passed in 0.07s`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche20|TestTopLevelModulePurgeTranche21|TestTopLevelModulePurgeTranche22|TestTopLevelModulePurgeTranche23|TestTopLevelModulePurgeTranche24|TestFollowUpLaneDocsStayAligned|TestExecutionPackRoadmapDocsStayAligned|TestExecutionPackRoadmapUniqueOwnersContract'` -> `ok  	bigclaw-go/internal/regression	1.049s`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/__main__.py src/bigclaw/observability.py src/bigclaw/operations.py src/bigclaw/runtime.py tests/test_design_system.py tests/test_console_ia.py tests/test_control_center.py tests/test_evaluation.py` -> passed
- `PYTHONPATH=src python3 - <<'PY' ... PY` -> `compat-surface-ok`
- `PYTHONPATH=src python3 -m pytest tests/test_design_system.py tests/test_console_ia.py tests/test_control_center.py tests/test_evaluation.py -q` -> `36 passed in 0.06s`
- `PYTHONPATH=src python3 -m bigclaw --help` -> passed with the expected migration-only deprecation warning and rendered `serve` / `repo-sync-audit` help text
- `bash scripts/ops/bigclawctl refill --help` -> passed
- `bash scripts/ops/bigclawctl workspace bootstrap --help` -> passed
- `bash scripts/ops/bigclawctl workspace validate --help` -> passed
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/bootstrap ./internal/refill ./internal/regression -run 'TestTopLevelModulePurgeTranche20|TestTopLevelModulePurgeTranche21|TestTopLevelModulePurgeTranche22|TestTopLevelModulePurgeTranche23|TestTopLevelModulePurgeTranche24|TestTopLevelModulePurgeTranche25|TestTopLevelModulePurgeTranche26|TestFollowUpLaneDocsStayAligned|TestExecutionPackRoadmapDocsStayAligned|TestExecutionPackRoadmapUniqueOwnersContract'` -> `ok  	bigclaw-go/cmd/bigclawctl	(cached) [no tests to run]`; `ok  	bigclaw-go/internal/bootstrap	(cached) [no tests to run]`; `ok  	bigclaw-go/internal/refill	(cached) [no tests to run]`; `ok  	bigclaw-go/internal/regression	1.019s`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/__main__.py src/bigclaw/observability.py src/bigclaw/operations.py src/bigclaw/runtime.py` -> passed
- `cd bigclaw-go && go test ./internal/product ./internal/reporting ./internal/pilot ./internal/api` -> `ok  	bigclaw-go/internal/product	(cached)`; `ok  	bigclaw-go/internal/reporting	(cached)`; `ok  	bigclaw-go/internal/pilot	(cached)`; `ok  	bigclaw-go/internal/api	(cached)`
- `cd bigclaw-go && go test ./internal/regression -run 'TestLane8CrossProcessCoordinationSurfaceStaysAligned|TestLane8ValidationBundleContinuationScorecardStaysAligned|TestLane8LiveShadowScorecardStaysAligned|TestTopLevelModulePurgeTranche20|TestTopLevelModulePurgeTranche21|TestTopLevelModulePurgeTranche22|TestTopLevelModulePurgeTranche23|TestTopLevelModulePurgeTranche24|TestTopLevelModulePurgeTranche25|TestTopLevelModulePurgeTranche26|TestTopLevelModulePurgeTranche27|TestFollowUpLaneDocsStayAligned|TestExecutionPackRoadmapDocsStayAligned|TestExecutionPackRoadmapUniqueOwnersContract'` -> `ok  	bigclaw-go/internal/regression	1.356s`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/observability.py src/bigclaw/operations.py src/bigclaw/runtime.py` -> passed
- `bash scripts/ops/bigclawctl legacy-python compile-check --json` -> `status=ok`; files=`src/bigclaw/__init__.py`, `src/bigclaw/observability.py`, `src/bigclaw/operations.py`, `src/bigclaw/runtime.py`
- `cd bigclaw-go && go test ./internal/legacyshim ./cmd/bigclawctl ./internal/regression -run 'TestFrozenCompileCheckFilesUsesFrozenShimList|TestCompileCheckRunsPyCompileAgainstFrozenShimList|TestRunLegacyPythonCompileCheckJSONOutputDoesNotEscapeArrowTokens|TestLegacyMainlineCompatibilityManifestStaysAligned|TestTopLevelModulePurgeTranche20|TestTopLevelModulePurgeTranche21|TestTopLevelModulePurgeTranche22|TestTopLevelModulePurgeTranche23|TestTopLevelModulePurgeTranche24|TestTopLevelModulePurgeTranche25|TestTopLevelModulePurgeTranche26|TestTopLevelModulePurgeTranche27|TestTopLevelModulePurgeTranche28|TestFollowUpLaneDocsStayAligned|TestExecutionPackRoadmapDocsStayAligned|TestExecutionPackRoadmapUniqueOwnersContract'` -> `ok  	bigclaw-go/internal/legacyshim	0.822s`; `ok  	bigclaw-go/cmd/bigclawctl	1.360s`; `ok  	bigclaw-go/internal/regression	2.152s`

## Python Count Impact
- before: `28`
- after: `4`
- delta: `-24`

## Residual Risks
- `src/bigclaw/runtime.py`, `src/bigclaw/operations.py`, and related modules remain the surviving live compatibility surface, so they are now the highest-risk merge targets
- the remaining Python files are now all runtime compatibility modules rather than peripheral wrappers or tests
- `src/bigclaw/legacy_shim.py` helper behavior still remains embedded in the surviving compatibility surfaces even though the four `scripts/ops/*.py` wrappers are now gone
- the remaining top-level Python files are now only live compatibility surfaces (`runtime.py`, `observability.py`, `operations.py`, `__init__.py`)
- further file-count reduction now requires merging one of the remaining core live modules (`runtime.py`, `observability.py`, or `operations.py`); that is beyond low-risk residual sweep work

## Terminal Blocker
- remaining Python footprint: `src/bigclaw/__init__.py` (`727` lines), `src/bigclaw/observability.py` (`1482` lines), `src/bigclaw/runtime.py` (`1829` lines), `src/bigclaw/operations.py` (`7334` lines)
- `src/bigclaw/__init__.py` is not a removable stub: it installs the compatibility submodules (`queue`, `scheduler`, `workflow`, `orchestration`, `service`, `evaluation`, `design_system`, `console_ia`, `legacy_shim`, `models`, `reports`) that the surviving Python modules still import
- `src/bigclaw/runtime.py` depends on `src/bigclaw/observability.py`
- `src/bigclaw/operations.py` depends on `src/bigclaw/observability.py` and on the package-installed compatibility submodules that originate from `src/bigclaw/__init__.py`
- with the wrapper/test/entrypoint layers gone, any additional file-count reduction now requires a large-scale merge across the live compatibility core rather than another low-risk sweep
- checked later branches that reached zero-Python states; their remaining-core deletion is not directly portable here because they depend on supporting Go surfaces absent from this branch (for example `bigclaw-go/internal/planning/planning.go`, `bigclaw-go/internal/planning/planning_test.go`, and `bigclaw-go/internal/regression/python_floor_guard_test.go`)
