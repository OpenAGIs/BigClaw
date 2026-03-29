# BIG-GO-967 Validation

## Batch File List

- `tests/test_mapping.py`
- `tests/test_connectors.py`
- `tests/test_workspace_bootstrap.py`
- `tests/test_saved_views.py`
- `tests/test_console_ia.py`
- `tests/test_dashboard_run_contract.py`

## Delete / Replace / Retain Rationale

- `tests/test_mapping.py`
  - Deleted.
  - Replaced by `bigclaw-go/internal/intake/mapping_test.go`, which already covers priority mapping and source issue to task conversion.
- `tests/test_connectors.py`
  - Deleted.
  - Replaced by `bigclaw-go/internal/intake/connector_test.go`, which covers minimum fetch behavior and adds Go-native connector lookup and ClawHost inventory assertions.
- `tests/test_workspace_bootstrap.py`
  - Deleted.
  - Replaced by `bigclaw-go/internal/bootstrap/bootstrap_test.go`.
  - This issue added missing Go coverage for cache-root selection, warm-cache reuse, existing workspace reuse, cleanup-preserved cache reuse, stale-seed recovery, and validation-report summary generation.
- `tests/test_saved_views.py`
  - Deleted.
  - Replaced by `bigclaw-go/internal/product/saved_views_test.go`, which exercises saved-view catalog generation, scope handling, ordering, and digest recipients.
- `tests/test_console_ia.py`
  - Deleted.
  - Replaced by `bigclaw-go/internal/product/console_test.go`, which covers console navigation, role-based home composition, aggregate behavior, and summary formatting.
- `tests/test_dashboard_run_contract.py`
  - Deleted.
  - Replaced by `bigclaw-go/internal/product/dashboard_run_contract_test.go`, which covers default contract readiness, audit gap detection, report rendering, and nested path traversal.

Retained outside this batch:

- Remaining `tests/*.py` files were retained because they still exercise Python-only runtime, report-generation, CLI shim, or broader product/control-plane behavior that does not yet have a sufficiently tight Go-native replacement surface in this branch.
- This lane intentionally stayed within already-owned Go packages: `internal/intake`, `internal/bootstrap`, and `internal/product`.

## Python File Count Impact

- `tests/**` Python files: `43 -> 37` (`-6`)
- Repo-wide Python files: `123 -> 117` (`-6`)

## Go Replacement Files

- `bigclaw-go/internal/intake/mapping_test.go`
- `bigclaw-go/internal/intake/connector_test.go`
- `bigclaw-go/internal/bootstrap/bootstrap_test.go`
- `bigclaw-go/internal/product/saved_views_test.go`
- `bigclaw-go/internal/product/console_test.go`
- `bigclaw-go/internal/product/dashboard_run_contract_test.go`

## Validation Commands

- `cd bigclaw-go && go test ./internal/intake -run 'TestMapPriority|TestMapSourceIssueToTask|TestConnectorsFetchMinimumIssue|TestConnectorByNameReturnsKnownConnectors|TestClawHostConnectorFetchesInventoryMetadata'`
- `cd bigclaw-go && go test ./internal/product -run 'TestBuildSavedViewCatalog|TestNavigationIncludesCoreConsoleSections|TestHomeForRoleUsesRoleSpecificCards|TestDefaultDesignSystemAndConsoleHelpers|TestBuildDefaultDashboardRunContractIsReleaseReady|TestDashboardRunContractAuditDetectsMissingPaths|TestRenderDashboardRunContractReport|TestContractPathExistsTraversesNestedObjectsAndLists|TestContractPathExistsSkipsNonMapListEntries|TestDashboardContractFormattingHelpers'`
- `cd bigclaw-go && go test ./internal/bootstrap -run 'TestRepoCacheKeyDerivesFromRepoLocator|TestCacheRootForRepoUsesRepoSpecificDirectory|TestBootstrapWorkspaceCreatesSharedWorktreeFromLocalSeed|TestSecondWorkspaceReusesWarmCacheWithoutFullClone|TestBootstrapWorkspaceReusesExistingIssueWorktree|TestCleanupWorkspacePreservesSharedCacheForFutureReuse|TestBootstrapRecoversFromStaleSeedDirectoryWithoutRemoteReclone|TestCleanupWorkspacePrunesWorktreeAndBootstrapBranch|TestValidationReportCoversThreeWorkspacesWithOneCache|TestWithCacheLockSerializesAcrossProcesses|TestBootstrapHelpersUniqueJoinAndPathExists'`
- `git status --short`

## Validation Results

- `cd bigclaw-go && go test ./internal/intake -run 'TestMapPriority|TestMapSourceIssueToTask|TestConnectorsFetchMinimumIssue|TestConnectorByNameReturnsKnownConnectors|TestClawHostConnectorFetchesInventoryMetadata'`
  - Result: `ok  	bigclaw-go/internal/intake	1.311s`
- `cd bigclaw-go && go test ./internal/product -run 'TestBuildSavedViewCatalog|TestNavigationIncludesCoreConsoleSections|TestHomeForRoleUsesRoleSpecificCards|TestDefaultDesignSystemAndConsoleHelpers|TestBuildDefaultDashboardRunContractIsReleaseReady|TestDashboardRunContractAuditDetectsMissingPaths|TestRenderDashboardRunContractReport|TestContractPathExistsTraversesNestedObjectsAndLists|TestContractPathExistsSkipsNonMapListEntries|TestDashboardContractFormattingHelpers'`
  - Result: `ok  	bigclaw-go/internal/product	1.629s`
- `cd bigclaw-go && go test ./internal/bootstrap -run 'TestRepoCacheKeyDerivesFromRepoLocator|TestCacheRootForRepoUsesRepoSpecificDirectory|TestBootstrapWorkspaceCreatesSharedWorktreeFromLocalSeed|TestSecondWorkspaceReusesWarmCacheWithoutFullClone|TestBootstrapWorkspaceReusesExistingIssueWorktree|TestCleanupWorkspacePreservesSharedCacheForFutureReuse|TestBootstrapRecoversFromStaleSeedDirectoryWithoutRemoteReclone|TestCleanupWorkspacePrunesWorktreeAndBootstrapBranch|TestValidationReportCoversThreeWorkspacesWithOneCache|TestWithCacheLockSerializesAcrossProcesses|TestBootstrapHelpersUniqueJoinAndPathExists'`
  - Result: `ok  	bigclaw-go/internal/bootstrap	4.272s`
