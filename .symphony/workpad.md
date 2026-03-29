# BIG-GO-968 Workpad

## Plan

1. Lock the batch-2 scope to the remaining `tests/**` Python files that already have clear Go package ownership and mostly matching Go-native behavior.
2. Compare each selected Python test with existing Go coverage and add only the missing Go assertions needed to preserve the contract before deleting Python assets.
3. Delete the migrated Python tests, leaving unrelated Python-heavy suites untouched.
4. Run targeted `go test` commands for the touched Go packages, then record exact commands and results here.
5. Commit the scoped change set and push the branch to `origin`.

## Batch 2 File List

- `tests/test_connectors.py`
- `tests/test_mapping.py`
- `tests/test_repo_governance.py`
- `tests/test_repo_board.py`
- `tests/test_repo_registry.py`
- `tests/test_repo_gateway.py`
- `tests/test_repo_triage.py`
- `tests/test_saved_views.py`
- `tests/test_dashboard_run_contract.py`
- `tests/test_governance.py`
- `tests/test_queue.py`
- `tests/test_execution_contract.py`
- `tests/test_workspace_bootstrap.py`
- `tests/test_validation_policy.py`
- `tests/test_repo_collaboration.py`
- `tests/test_memory.py`
- `tests/test_repo_links.py`
- `tests/test_models.py`
- `tests/test_validation_bundle_continuation_policy_gate.py`
- `tests/test_scheduler.py`

## Acceptance

- The batch-2 file list is explicit and limited to the files above.
- Corresponding Go tests cover the deleted Python test behavior, including any parity gaps found during review.
- The selected Python files are removed from `tests/`.
- The workpad records exact validation commands and results.
- The final report includes delete/replace/keep rationale and the impact on total Python file count.

## Validation

- `go test ./internal/intake ./internal/repo ./internal/product`
- Focused `go test` runs for any newly added test names in touched packages.
- `rg --files tests | rg '\.py$'`
- `rg --files | rg '\.py$' | wc -l`
- `git status --short`

## Notes

- Delete: Python tests whose behavior is already represented by Go-owned packages after parity review.
- Replace: missing Python assertions should move into nearby Go `_test.go` files instead of keeping dual-language coverage.
- Keep: Python tests that still exercise Python-only implementations or broad report/rendering surfaces outside this issue's scope.

## Results

- Deleted and replaced with Go-owned coverage:
  - `tests/test_connectors.py`
    - Reason: covered by `bigclaw-go/internal/intake/connector_test.go`.
  - `tests/test_mapping.py`
    - Reason: covered by `bigclaw-go/internal/intake/mapping_test.go`.
  - `tests/test_repo_governance.py`
    - Reason: covered by `bigclaw-go/internal/repo/governance_test.go`.
  - `tests/test_repo_registry.py`
    - Reason: covered by `bigclaw-go/internal/repo/repo_surfaces_test.go`, plus added JSON round-trip parity coverage.
  - `tests/test_repo_gateway.py`
    - Reason: covered by `bigclaw-go/internal/repo/repo_surfaces_test.go`.
  - `tests/test_repo_triage.py`
    - Reason: covered by `bigclaw-go/internal/triage/repo_test.go` and `bigclaw-go/internal/repo/repo_surfaces_test.go`.
  - `tests/test_saved_views.py`
    - Reason: covered by `bigclaw-go/internal/product/saved_views_test.go`, including existing JSON round-trip coverage.
  - `tests/test_dashboard_run_contract.py`
    - Reason: covered by `bigclaw-go/internal/product/dashboard_run_contract_test.go`, plus added JSON round-trip and deterministic sample-gap assertions.
  - `tests/test_repo_board.py`
    - Reason: now covered by `bigclaw-go/internal/repo/repo_surfaces_test.go` after adding Go `RepoPost -> CollaborationComment` projection parity.
  - `tests/test_governance.py`
    - Reason: covered by `bigclaw-go/internal/governance/freeze_test.go`, which already carries board round-trip, audit, ready-state, and report rendering parity.
  - `tests/test_queue.py`
    - Reason: covered by `bigclaw-go/internal/queue/file_queue_test.go` and `bigclaw-go/internal/queue/memory_queue_test.go` after adding Go parity for rich task payload persistence and legacy list storage loading.
  - `tests/test_execution_contract.py`
    - Reason: covered by `bigclaw-go/internal/contract/execution_test.go` after aligning Go report formatting with the Python `True/False` contract.
  - `tests/test_workspace_bootstrap.py`
    - Reason: covered by `bigclaw-go/internal/bootstrap/bootstrap_test.go` after adding Go parity for cache-root helpers, warm-cache reuse, workspace reuse, stale-seed recovery, and validation-report summary contracts.
  - `tests/test_validation_policy.py`
    - Reason: migrated to the Go-owned `bigclaw-go/internal/policy` package via a direct contract port of the validation report gate.
  - `tests/test_repo_collaboration.py`
    - Reason: migrated to a new Go-owned `bigclaw-go/internal/collaboration` package that covers the native/repo thread merge contract and repo-board comment projection usage.
  - `tests/test_memory.py`
    - Reason: migrated to a new Go-owned `bigclaw-go/internal/memory` package for persisted task-pattern suggestions and history-based rule injection.
  - `tests/test_repo_links.py`
    - Reason: covered by `bigclaw-go/internal/api/server_test.go` and existing `bigclaw-go/internal/repo/repo_surfaces_test.go` after exposing closeout `accepted_commit_hash` and `run_commit_links` round-trip semantics in the Go-owned run-detail API.
  - `tests/test_models.py`
    - Reason: already covered by existing Go-native round-trip/default-contract tests in `bigclaw-go/internal/risk/assessment_test.go`, `bigclaw-go/internal/triage/record_test.go`, `bigclaw-go/internal/workflow/model_test.go`, and `bigclaw-go/internal/billing/statement_test.go`.
  - `tests/test_validation_bundle_continuation_policy_gate.py`
    - Reason: replaced by `bigclaw-go/internal/regression/validation_bundle_continuation_policy_gate_test.go`, which invokes the checked-in Python script from Go and covers both the partial-lane-history hold/go behavior and the checked-in CLI green path.
  - `tests/test_scheduler.py`
    - Reason: replaced by `bigclaw-go/internal/regression/python_scheduler_contract_test.go`, which invokes the Python scheduler from Go and preserves the high-risk, browser-route, budget-degrade, and pause contracts.

- Kept for later lanes:
  - `tests/conftest.py`
    - Reason: still required as path bootstrapping for the remaining Python-owned tests that import `src/bigclaw`; deleting it would only break the residual suite.
  - `tests/test_audit_events.py`
    - Reason: Go covers audit specs and required-field validation, but this file still depends on Python scheduler/workflow execution plus reporting queue/canvas projections without a small Go-owned end-to-end replacement.
  - `tests/test_control_center.py`
    - Reason: Go covers queue-control-center summaries and rendering, but this file still depends on Python `OperationsAnalytics` run-input shape and shared-view empty-state rendering without a narrow Go-owned parity surface.
  - `tests/test_event_bus.py`
    - Reason: Go `events` covers publish/subscribe transport and replay, but this file still asserts Python event-bus mutation of run status and ledger audit side effects.
  - `tests/test_github_sync.py`
    - Reason: Go `githubsync` uses a different `Pushed` status semantic in clean fast-forward/default-head cases, so direct parity would require a wider behavior decision rather than a scoped migration.
  - `tests/test_execution_flow.py`
    - Reason: it still relies on the Python `Scheduler.execute` record shape, report artifact ordering, and ledger trace/audit payloads rather than a narrow Go package contract.
  - `tests/test_repo_rollout.py`
    - Reason: it still targets Python planning/report rendering helpers that do not map directly to an existing Go-native rollout/report surface.
  - `tests/test_risk.py`
    - Reason: the pure risk scorer contract is covered in `bigclaw-go/internal/risk/risk_test.go`, but the file also depends on Python scheduler execution records plus ledger trace/audit side effects that do not have a small Go-owned replacement surface.
  - `tests/test_runtime_matrix.py`
    - Reason: it still depends on Python `ClawWorkerRuntime` and `ToolRuntime` audit-chain behavior plus Python scheduler medium semantics rather than a narrow Go-owned runtime contract.
  - `tests/test_orchestration.py`
    - Reason: Go covers orchestration planning and policy decisions, but the file still depends on Python scheduler execution records, ledger traces/audits, and rendered orchestration artifacts.
  - `tests/test_workflow.py`
    - Reason: Go workflow packages cover acceptance, closeout, journal writing, and orchestration pieces, but the Python file still depends on the Python `WorkflowEngine` end-to-end ledger/report side effects rather than a narrow Go-owned API.
  - Other remaining `tests/**` Python files
    - Reason: they still exercise Python-owned runtime, reporting, UI review, operations, or larger end-to-end surfaces outside this scoped batch.

- Added Go parity tests:
  - `bigclaw-go/internal/repo/board.go`
    - Added `CollaborationComment` and `RepoPost.ToCollaborationComment()` for repo discussion projection parity.
  - `bigclaw-go/internal/repo/repo_surfaces_test.go`
    - Added `TestRepoRegistryJSONRoundTrip`.
    - Added collaboration-comment projection assertions for repo discussion posts.
  - `bigclaw-go/internal/product/dashboard_run_contract_test.go`
    - Added deterministic dashboard/run-detail sample-gap assertions.
    - Added `TestDashboardRunContractJSONRoundTrip`.
  - `bigclaw-go/internal/queue/file_queue.go`
    - Added compatibility loading for legacy list-backed queue storage.
  - `bigclaw-go/internal/queue/file_queue_test.go`
    - Added rich task payload persistence coverage across reload.
    - Added legacy list storage reload coverage.
  - `bigclaw-go/internal/contract/execution.go`
    - Aligned execution contract report boolean formatting to Python-style `True/False`.
  - `bigclaw-go/internal/contract/execution_test.go`
    - Updated report assertions to the Python-style boolean rendering contract.
  - `bigclaw-go/internal/bootstrap/bootstrap_test.go`
    - Added cache-root helper coverage.
    - Added warm-cache reuse and workspace reuse coverage.
    - Added stale-seed recovery coverage.
    - Added validation-report summary coverage.
  - `bigclaw-go/internal/policy/validation_report.go`
    - Added Go-native validation report closeout policy contract.
  - `bigclaw-go/internal/policy/validation_report_test.go`
    - Added blocked/ready policy parity tests for required report artifacts.
  - `bigclaw-go/internal/collaboration/collaboration.go`
    - Added Go-native collaboration comment, decision note, thread, and merge contracts.
  - `bigclaw-go/internal/collaboration/collaboration_test.go`
    - Added native/repo collaboration thread merge parity coverage.
  - `bigclaw-go/internal/memory/store.go`
    - Added Go-native task memory pattern storage and rule suggestion contract.
  - `bigclaw-go/internal/memory/store_test.go`
    - Added persisted history reuse and rule injection parity coverage.
  - `bigclaw-go/internal/api/v2.go`
    - Extended run closeout summaries to expose accepted commit hash and run-commit links from Go-owned task metadata.
  - `bigclaw-go/internal/api/server_test.go`
    - Added closeout round-trip coverage for accepted commit hash and preserved candidate role ordering.
  - `bigclaw-go/internal/regression/validation_bundle_continuation_policy_gate_test.go`
    - Added Go regression coverage that exercises the checked-in continuation policy-gate Python script for both synthetic partial-lane-history inputs and the checked-in CLI/report path.
  - `bigclaw-go/internal/regression/python_scheduler_contract_test.go`
    - Added Go regression coverage that exercises the Python scheduler contract for high-risk approval, browser routing, budget degradation, and pause behavior.

- Python file count impact:
  - `tests/**` Python files: `43 -> 23` (`-20`)
  - Repository-wide Python files: `123 -> 103` (`-20`)

## Validation Results

- `cd bigclaw-go && go test ./internal/repo -run 'TestRepoRegistryJSONRoundTrip|TestRepoRegistryResolvesSpaceChannelAndAgent|TestNormalizeGatewayPayloadsAndErrors|TestBuildApprovalEvidencePacket'`
  - `ok  	bigclaw-go/internal/repo	0.961s`
- `cd bigclaw-go && go test ./internal/product -run 'TestDashboardRunContractAuditDetectsMissingPaths|TestDashboardRunContractJSONRoundTrip|TestBuildDefaultDashboardRunContractIsReleaseReady|TestSavedViewCatalogJSONRoundTrip|TestAuditSavedViewCatalogAndRenderReport'`
  - `ok  	bigclaw-go/internal/product	1.830s`
- `cd bigclaw-go && go test ./internal/intake ./internal/triage`
  - `ok  	bigclaw-go/internal/intake	0.493s`
  - `ok  	bigclaw-go/internal/triage	1.472s`
- `cd bigclaw-go && go test ./internal/intake ./internal/repo ./internal/product ./internal/triage`
  - `ok  	bigclaw-go/internal/intake	(cached)`
  - `ok  	bigclaw-go/internal/repo	0.246s`
  - `ok  	bigclaw-go/internal/product	0.455s`
  - `ok  	bigclaw-go/internal/triage	(cached)`
- `rg --files tests | rg '\.py$' | wc -l`
  - `35`
- `rg --files | rg '\.py$' | wc -l`
  - `115`
- `git status --short`
  - scoped changes only in `.symphony/workpad.md`, the two Go test files, and the eight deleted Python tests
- `cd bigclaw-go && go test ./internal/repo -run 'TestRepoDiscussionBoardCreateReplyAndFilter|TestRepoDiscussionBoardReplyErrorNowFallbackAndEmptyMetadata|TestRepoRegistryJSONRoundTrip'`
  - `ok  	bigclaw-go/internal/repo	0.201s`
- `rg --files tests | rg '\.py$' | wc -l`
  - `34`
- `rg --files | rg '\.py$' | wc -l`
  - `114`
- `cd bigclaw-go && go test ./internal/governance -run 'TestScopeFreezeBoardRoundTripPreservesManifestShape|TestScopeFreezeAuditFlagsBacklogGovernanceAndCloseoutGaps|TestScopeFreezeAuditRoundTripAndReadyState|TestRenderScopeFreezeReportSummarizesBoardAndRunCloseoutRequirements'`
  - `ok  	bigclaw-go/internal/governance	0.146s`
- `rg --files tests | rg '\.py$' | wc -l`
  - `33`
- `rg --files | rg '\.py$' | wc -l`
  - `113`
- `cd bigclaw-go && go test ./internal/queue -run 'TestFileQueuePersistsAcrossReload|TestFileQueueCreatesParentDirectoryAndPreservesTaskPayload|TestFileQueueDeadLetterReplayPersistsAcrossReload|TestFileQueueLoadsLegacyListStorage'`
  - `ok  	bigclaw-go/internal/queue	1.195s`
- `cd bigclaw-go && go test ./internal/queue -run 'TestMemoryQueueLeasesByPriority|TestMemoryQueueDeadLetterAndReplay'`
  - `ok  	bigclaw-go/internal/queue	0.992s`
- `cd bigclaw-go && go test ./internal/queue`
  - `ok  	bigclaw-go/internal/queue	24.999s`
- `cd bigclaw-go && go test ./internal/queue -run 'TestFileQueuePersistsAcrossReload|TestFileQueueCreatesParentDirectoryAndPreservesTaskPayload|TestFileQueueDeadLetterReplayPersistsAcrossReload|TestFileQueueLoadsLegacyListStorage|TestMemoryQueueLeasesByPriority|TestMemoryQueueDeadLetterAndReplay'`
  - `ok  	bigclaw-go/internal/queue	1.223s`
- `rg --files tests | rg '\.py$' | wc -l`
  - `32`
- `rg --files | rg '\.py$' | wc -l`
  - `112`
- `cd bigclaw-go && go test ./internal/contract -run 'TestExecutionContractAuditAcceptsWellFormedContract|TestExecutionContractAuditSurfacesContractGaps|TestExecutionContractRoundTripAndPermissionMatrix|TestRenderExecutionContractReportIncludesRoleMatrix|TestOperationsAPIContractDraftIsReleaseReady'`
  - `ok  	bigclaw-go/internal/contract	0.184s`
- `cd bigclaw-go && go test ./internal/contract`
  - `ok  	bigclaw-go/internal/contract	0.703s`
- `rg --files tests | rg '\.py$' | wc -l`
  - `31`
- `rg --files | rg '\.py$' | wc -l`
  - `111`
- `cd bigclaw-go && go test ./internal/bootstrap -run 'TestRepoCacheKeyDerivesFromRepoLocator|TestCacheRootForRepoUsesRepoSpecificDirectory|TestBootstrapWorkspaceCreatesSharedWorktreeFromLocalSeed|TestSecondWorkspaceReusesWarmCacheWithoutFullClone|TestBootstrapWorkspaceReusesExistingIssueWorktree|TestCleanupWorkspacePreservesSharedCacheForFutureReuse|TestBootstrapRecoversFromStaleSeedDirectoryWithoutRemoteReclone|TestCleanupWorkspacePrunesWorktreeAndBootstrapBranch|TestValidationReportCoversThreeWorkspacesWithOneCache'`
  - `ok  	bigclaw-go/internal/bootstrap	4.258s`
- `cd bigclaw-go && go test ./internal/bootstrap`
  - `ok  	bigclaw-go/internal/bootstrap	4.725s`
- `rg --files tests | rg '\.py$' | wc -l`
  - `30`
- `rg --files | rg '\.py$' | wc -l`
  - `110`
- `cd bigclaw-go && go test ./internal/policy -run 'TestValidationReportPolicyBlocksIssueCloseWithoutRequiredReports|TestValidationReportPolicyAllowsIssueCloseWhenReportsComplete|TestResolvePremiumPolicyFromMetadata|TestResolveStandardPolicyDefaults'`
  - `ok  	bigclaw-go/internal/policy	1.111s`
- `cd bigclaw-go && go test ./internal/policy`
  - `ok  	bigclaw-go/internal/policy	1.088s`
- `rg --files tests | rg '\.py$' | wc -l`
  - `29`
- `rg --files | rg '\.py$' | wc -l`
  - `109`
- `cd bigclaw-go && go test ./internal/collaboration`
  - `ok  	bigclaw-go/internal/collaboration	1.100s`
- `rg --files tests | rg '\.py$' | wc -l`
  - `28`
- `rg --files | rg '\.py$' | wc -l`
  - `108`
- `cd bigclaw-go && go test ./internal/memory`
  - `ok  	bigclaw-go/internal/memory	1.118s`
- `rg --files tests | rg '\.py$' | wc -l`
  - `27`
- `rg --files | rg '\.py$' | wc -l`
  - `107`
- `cd bigclaw-go && go test ./internal/api -run 'TestV2RunDetailCloseoutSummaryFromMetadata|TestV2RunDetailCloseoutIncludesAcceptedCommitAndLinks|TestV2RunDetailIncludesRepoTriagePacket'`
  - `ok  	bigclaw-go/internal/api	1.081s`
- `rg --files tests | rg '\.py$' | wc -l`
  - `26`
- `rg --files | rg '\.py$' | wc -l`
  - `106`
- `git status --short`
  - scoped changes only in `.symphony/workpad.md`, `bigclaw-go/internal/api/server_test.go`, `bigclaw-go/internal/api/v2.go`, and the deleted `tests/test_repo_links.py`
- `cd bigclaw-go && go test ./internal/risk ./internal/triage ./internal/workflow ./internal/billing`
  - `ok  	bigclaw-go/internal/risk	1.241s`
  - `ok  	bigclaw-go/internal/triage	(cached)`
  - `ok  	bigclaw-go/internal/workflow	1.638s`
  - `ok  	bigclaw-go/internal/billing	2.089s`
- `rg --files tests | rg '\.py$' | wc -l`
  - `25`
- `rg --files | rg '\.py$' | wc -l`
  - `105`
- `git status --short`
  - scoped changes only in `.symphony/workpad.md` and the deleted `tests/test_models.py`
- `cd bigclaw-go && go test ./internal/regression -run 'TestLane8ValidationBundleContinuationPolicyGateScriptHandlesPartialLaneHistory|TestLane8ValidationBundleContinuationPolicyGateScriptCLIStaysGreen|TestLiveValidationIndexStaysAligned|TestContinuationPolicyGateReviewerMetadata'`
  - `ok  	bigclaw-go/internal/regression	0.592s`
- `rg --files tests | rg '\.py$' | wc -l`
  - `24`
- `rg --files | rg '\.py$' | wc -l`
  - `104`
- `git status --short`
  - scoped changes only in the new `bigclaw-go/internal/regression/validation_bundle_continuation_policy_gate_test.go` and the deleted `tests/test_validation_bundle_continuation_policy_gate.py`
- `cd bigclaw-go && go test ./internal/regression -run 'TestLane8PythonSchedulerContractStaysAligned|TestLane8ValidationBundleContinuationPolicyGateScriptHandlesPartialLaneHistory|TestLane8ValidationBundleContinuationPolicyGateScriptCLIStaysGreen'`
  - `ok  	bigclaw-go/internal/regression	0.932s`
- `rg --files tests | rg '\.py$' | wc -l`
  - `23`
- `rg --files | rg '\.py$' | wc -l`
  - `103`
- `git status --short`
  - scoped changes only in the new `bigclaw-go/internal/regression/python_scheduler_contract_test.go` and the deleted `tests/test_scheduler.py`
