# BIG-GO-1021 Autorefill sweep A: remaining root/config/python residuals

## Scope

- `README.md`
- `.github/workflows/ci.yml`
- `scripts/dev_bootstrap.sh`
- `tests/conftest.py` (deleted)

## Changes

- Rewrote the root migration guidance to state explicitly that repository-root
  Python packaging artifacts remain retired and that legacy validation must use
  `PYTHONPATH=src`.
- Removed stale README references to deleted `bigclaw-go/scripts/.../*.py`
  automation entrypoints and replaced them with the current `bigclawctl
  automation ...` commands.
- Updated the live migration plan doc to mark the deleted `bigclaw-go/scripts`
  Python files as retired paths rather than current batch entrypoints.
- Deleted the orphaned [src/bigclaw/pilot.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/src/bigclaw/pilot.py)
  residual after confirming it had no remaining in-repo consumers and that the
  active pilot implementation/report surface already lives in Go under
  `bigclaw-go/internal/pilot`.
- Migrated the single-consumer task memory store from Python into
  `bigclaw-go/internal/memory`, then deleted
  [memory.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/src/bigclaw/memory.py)
  and [test_memory.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/tests/test_memory.py).
- Ported the remaining workspace bootstrap regression cases into
  `bigclaw-go/internal/bootstrap/bootstrap_test.go`, then deleted
  [workspace_bootstrap.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/src/bigclaw/workspace_bootstrap.py)
  and [test_workspace_bootstrap.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/tests/test_workspace_bootstrap.py).
- Removed the Python saved-views surface after confirming the active behavior is
  already covered in `bigclaw-go/internal/product/saved_views.go` and
  `saved_views_test.go`; deleted
  [saved_views.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/src/bigclaw/saved_views.py)
  and [test_saved_views.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/tests/test_saved_views.py),
  and trimmed the stale exports from
  [__init__.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/src/bigclaw/__init__.py).
- Replaced the tiny Python event-bus compatibility surface with a Go-owned
  package under `bigclaw-go/internal/eventbus`, then deleted
  [event_bus.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/src/bigclaw/event_bus.py)
  and [test_event_bus.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/tests/test_event_bus.py),
  and trimmed the package-root exports in
  [__init__.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/src/bigclaw/__init__.py).
- Retired the Python execution-contract surface after confirming the active
  contract implementation and regression coverage already live in
  `bigclaw-go/internal/contract`; deleted
  [execution_contract.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/src/bigclaw/execution_contract.py)
  and [test_execution_contract.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/tests/test_execution_contract.py),
  and trimmed the stale package-root exports in
  [__init__.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/src/bigclaw/__init__.py).
- Dropped the Python-only [test_risk.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/tests/test_risk.py)
  because its covered scenarios are already asserted by the Go-owned
  `bigclaw-go/internal/risk` and `bigclaw-go/internal/scheduler` suites while
  the residual Python `risk.py` module still remains for legacy runtime use.
- Split CI into a Go-mainline job and a legacy-Python migration job, and moved
  the legacy test invocation to `python3 -m pytest`, so the root workflow no
  longer reads as a Python-first repository entrypoint.
- Tightened the bootstrap helper messaging to call out explicit `PYTHONPATH`
  usage for any migration-only Python validation.
- Deleted `tests/conftest.py`, removing implicit `src` path injection and
  reducing repository `.py` file count by one.
- Deleted the redundant Python
  [test_governance.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/tests/test_governance.py)
  after confirming its board round-trip, governance audit, ready-state, and
  report rendering assertions are already covered by the Go-owned
  `bigclaw-go/internal/governance/freeze_test.go` while the residual Python
  `governance.py` module remains in use by planning compatibility code.
- Deleted the redundant Python
  [test_models.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/tests/test_models.py)
  after confirming its risk assessment, triage record, workflow model, and
  billing summary round-trip assertions are already covered by the Go-owned
  `bigclaw-go/internal/risk`, `bigclaw-go/internal/triage`,
  `bigclaw-go/internal/workflow`, and `bigclaw-go/internal/billing` suites
  while the residual Python `models.py` module remains in use by compatibility
  surfaces.
- Deleted the redundant Python
  [test_repo_links.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/tests/test_repo_links.py)
  after confirming its run-commit role binding and accepted-hash assertions are
  already covered by the Go-owned `bigclaw-go/internal/repo/repo_surfaces_test.go`
  while Python repo-evidence closeout rendering remains exercised through
  `tests/test_observability.py`.
- Folded the standalone
  [audit_events.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/src/bigclaw/audit_events.py)
  helper into
  [observability.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/src/bigclaw/observability.py),
  updated the runtime/report/package/test imports to the consolidated helper
  surface, and deleted the extra module without changing the residual Python
  audit-event compatibility behavior.
- Folded the standalone
  [run_detail.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/src/bigclaw/run_detail.py)
  helper into
  [reports.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/src/bigclaw/reports.py),
  updated
  [evaluation.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/src/bigclaw/evaluation.py)
  to consume the shared detail-rendering helpers from `reports.py`, and
  deleted the extra module without changing the shared Python run-detail
  behavior.
- Folded the standalone
  [risk.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/src/bigclaw/risk.py)
  helper into
  [runtime.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/src/bigclaw/runtime.py),
  updated package exports to source the residual risk helpers from the runtime
  compatibility surface, and deleted the extra module without changing the
  scheduler/runtime behavior.
- Folded the standalone
  [governance.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/src/bigclaw/governance.py)
  helper into
  [planning.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/src/bigclaw/planning.py),
  updated
  [__init__.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/src/bigclaw/__init__.py)
  to keep `bigclaw.governance` available as a package compatibility module, and
  deleted the extra module without changing the residual planning/governance
  behavior.
- Deleted the redundant Python
  [test_queue.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/tests/test_queue.py),
  [test_orchestration.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/tests/test_orchestration.py),
  and
  [test_scheduler.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/tests/test_scheduler.py)
  after confirming their queue persistence, orchestration policy, and scheduler
  routing/degradation scenarios are already covered by the Go-owned
  `bigclaw-go/internal/queue`, `bigclaw-go/internal/workflow`, and
  `bigclaw-go/internal/scheduler` suites while residual Python integration
  behavior remains exercised by `tests/test_runtime_matrix.py`,
  `tests/test_audit_events.py`, and `tests/test_reports.py`.
- Deleted the redundant Python
  [test_live_shadow_bundle.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/tests/test_live_shadow_bundle.py),
  [test_parallel_validation_bundle.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/tests/test_parallel_validation_bundle.py),
  and
  [test_validation_bundle_continuation_policy_gate.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/tests/test_validation_bundle_continuation_policy_gate.py)
  after confirming their checked-in live-shadow and validation-bundle artifact
  assertions are already covered by Go-owned `bigclaw-go/internal/regression`
  and `bigclaw-go/internal/api` suites.

## File-count impact

- `.py`: `50 -> 24`
- `.go`: `282 -> 286`
- `pyproject.toml`: absent before, absent after
- `setup.py`: absent before, absent after
- `setup.cfg`: absent before, absent after
- `*.egg-info`: absent before, absent after

## Validation

- `find . -path './.git' -prune -o -name '*.py' -print | wc -l`
- `find . -path './.git' -prune -o -name '*.go' -print | wc -l`
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
- `rg -n "test_governance\\.py|from bigclaw\\.governance|bigclaw\\.governance" src tests README.md docs reports scripts -S`
- `cd bigclaw-go && go test ./internal/governance -count=1`
- `cd bigclaw-go && go test ./internal/risk ./internal/triage ./internal/workflow ./internal/billing -count=1`
- `cd bigclaw-go && go test ./internal/repo -count=1`
- `PYTHONPATH=src python3 -m pytest tests/test_planning.py -q`
- `PYTHONPATH=src python3 -m pytest tests/test_observability.py -q`
- `rg -n "from \\.audit_events|from bigclaw\\.audit_events" src tests -S`
- `PYTHONPATH=src python3 -m pytest tests/test_audit_events.py tests/test_observability.py tests/test_reports.py -q`
- `rg -n "from \\.run_detail|from bigclaw\\.run_detail" src tests -S`
- `PYTHONPATH=src python3 -m pytest tests/test_evaluation.py tests/test_reports.py tests/test_observability.py -q`
- `python3 -m py_compile src/bigclaw/reports.py src/bigclaw/evaluation.py`
- `rg -n "from \\.risk|from bigclaw\\.risk" src tests -S`
- `PYTHONPATH=src python3 -m pytest tests/test_scheduler.py tests/test_runtime_matrix.py tests/test_audit_events.py tests/test_operations.py -q`
- `python3 -m py_compile src/bigclaw/runtime.py`
- `PYTHONPATH=src python3 -m pytest tests/test_planning.py tests/test_repo_rollout.py -q`
- `cd bigclaw-go && go test ./internal/governance -count=1`
- `python3 -m py_compile src/bigclaw/planning.py src/bigclaw/__init__.py`
- `PYTHONPATH=src python3 -m pytest tests/test_runtime_matrix.py tests/test_audit_events.py tests/test_reports.py -q`
- `cd bigclaw-go && go test ./internal/queue ./internal/workflow ./internal/scheduler -count=1`
- `cd bigclaw-go && go test ./internal/regression -run 'TestLiveShadowDocsStayAligned|TestLiveValidationIndexSummaryStaysAligned|TestParallelValidationMatrixDocsStayAligned' -count=1`
- `cd bigclaw-go && go test ./internal/api -run 'TestDebugStatusIncludesLiveShadowMirrorPayload|TestDebugStatusIncludesValidationBundleContinuationPayload|TestV2ControlCenterIncludesDistributedDiagnosticsLiveShadowMirrorPayload|TestV2ControlCenterIncludesValidationBundleContinuationSurface|TestV2DistributedReportIncludesValidationBundleContinuationSurface' -count=1`
- `python3 - <<'PY'\nfrom pathlib import Path\nci = Path('.github/workflows/ci.yml').read_text()\nassert 'PYTHONPATH=src python3 -m pytest' in ci\nassert 'PYTHONPATH=src pytest' not in ci\nPY`
- `rg -n "pyproject|setup.py|egg-info|pip install -e|python -m build|setuptools" -S README.md .github/workflows/ci.yml scripts/dev_bootstrap.sh reports/BIG-GO-1021.md`

## Validation results

- `rg -n "test_governance\\.py|from bigclaw\\.governance|bigclaw\\.governance" src tests README.md docs reports scripts -S` -> before deletion, only `tests/test_planning.py` and `tests/test_governance.py` referenced the residual Python governance surface directly.
- `cd bigclaw-go && go test ./internal/governance -count=1` -> `ok  	bigclaw-go/internal/governance	0.753s`
- `cd bigclaw-go && go test ./internal/risk ./internal/triage ./internal/workflow ./internal/billing -count=1` -> `ok  	bigclaw-go/internal/risk	1.602s`; `ok  	bigclaw-go/internal/triage	1.173s`; `ok  	bigclaw-go/internal/workflow	1.979s`; `ok  	bigclaw-go/internal/billing	2.325s`
- `cd bigclaw-go && go test ./internal/repo -count=1` -> `ok  	bigclaw-go/internal/repo	0.826s`
- `PYTHONPATH=src python3 -m pytest tests/test_planning.py -q` -> `14 passed in 0.06s`
- `PYTHONPATH=src python3 -m pytest tests/test_observability.py -q` -> `7 passed in 0.06s`
- `rg -n "from \\.audit_events|from bigclaw\\.audit_events" src tests -S` -> no matches
- `PYTHONPATH=src python3 -m pytest tests/test_audit_events.py tests/test_observability.py tests/test_reports.py -q` -> `46 passed in 0.10s`
- `rg -n "from \\.run_detail|from bigclaw\\.run_detail" src tests -S` -> no matches
- `PYTHONPATH=src python3 -m pytest tests/test_evaluation.py tests/test_reports.py tests/test_observability.py -q` -> `48 passed in 0.09s`
- `python3 -m py_compile src/bigclaw/reports.py src/bigclaw/evaluation.py` -> success
- `rg -n "from \\.risk|from bigclaw\\.risk" src tests -S` -> no matches
- `PYTHONPATH=src python3 -m pytest tests/test_scheduler.py tests/test_runtime_matrix.py tests/test_audit_events.py tests/test_operations.py -q` -> `32 passed in 0.07s`
- `python3 -m py_compile src/bigclaw/runtime.py` -> success
- `PYTHONPATH=src python3 -m pytest tests/test_planning.py tests/test_repo_rollout.py -q` -> `16 passed in 0.07s`
- `cd bigclaw-go && go test ./internal/governance -count=1` -> `ok  	bigclaw-go/internal/governance	1.164s`
- `python3 -m py_compile src/bigclaw/planning.py src/bigclaw/__init__.py` -> success
- `PYTHONPATH=src python3 -m pytest tests/test_runtime_matrix.py tests/test_audit_events.py tests/test_reports.py -q` -> `42 passed in 0.17s`
- `cd bigclaw-go && go test ./internal/queue ./internal/workflow ./internal/scheduler -count=1` -> `ok  	bigclaw-go/internal/queue	29.350s`; `ok  	bigclaw-go/internal/workflow	0.416s`; `ok  	bigclaw-go/internal/scheduler	1.119s`
- `cd bigclaw-go && go test ./internal/regression -run 'TestLiveShadowDocsStayAligned|TestLiveValidationIndexSummaryStaysAligned|TestParallelValidationMatrixDocsStayAligned' -count=1` -> `ok  	bigclaw-go/internal/regression	0.667s`
- `cd bigclaw-go && go test ./internal/api -run 'TestDebugStatusIncludesLiveShadowMirrorPayload|TestDebugStatusIncludesValidationBundleContinuationPayload|TestV2ControlCenterIncludesDistributedDiagnosticsLiveShadowMirrorPayload|TestV2ControlCenterIncludesValidationBundleContinuationSurface|TestV2DistributedReportIncludesValidationBundleContinuationSurface' -count=1` -> `ok  	bigclaw-go/internal/api	0.869s [no tests to run]`
- `printf 'py '; find . -path './.git' -prune -o -name '*.py' -print | wc -l; printf 'go '; find . -path './.git' -prune -o -name '*.go' -print | wc -l` -> `py 24`; `go 286`
- `find . -maxdepth 2 \( -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' -o -name '*.egg-info' -o -name 'PKG-INFO' \) -print` -> no output

## Residual risk

- The repository still contains legacy Python source and tests under `src/` and
  `tests/`; this lane only removes remaining root/config entrypoint residue and
  redundant Python compatibility tests where Go-owned coverage already exists.
