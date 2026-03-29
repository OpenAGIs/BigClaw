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

- Kept for later lanes:
  - `tests/test_repo_links.py`
    - Reason: it still crosses into broader run closeout / observability payload round-trip behavior, beyond the minimal repo-surface scope handled here.
  - `tests/test_github_sync.py`
    - Reason: Go `githubsync` uses a different `Pushed` status semantic in clean fast-forward/default-head cases, so direct parity would require a wider behavior decision rather than a scoped migration.
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

- Python file count impact:
  - `tests/**` Python files: `43 -> 30` (`-13`)
  - Repository-wide Python files: `123 -> 110` (`-13`)

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
