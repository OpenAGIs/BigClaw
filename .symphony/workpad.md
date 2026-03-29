# BIG-GO-967 Workpad

## Plan

1. Confirm the direct migration batch within remaining `tests/**` Python files by selecting files that already have clear Go-native ownership in `bigclaw-go`.
2. Close the remaining parity gap for `tests/test_workspace_bootstrap.py` by adding targeted Go tests in `bigclaw-go/internal/bootstrap/bootstrap_test.go`.
3. Delete the Python tests whose behavior is now covered by Go tests, while leaving unrelated Python tests untouched.
4. Record the lane file list, delete/replace/retain rationale, and Python file-count impact in an issue-scoped validation report.
5. Run targeted `go test` commands for the touched Go packages, verify git state, then commit and push the branch.

## Acceptance

- Produce an explicit file list for this `BIG-GO-967` batch.
- Reduce the Python file count in `tests/` by deleting only files with Go-native replacements.
- For each file in the batch, state whether it was deleted, replaced, or retained and why.
- Report the before/after overall Python file count impact.
- Record exact validation commands and results.

## Validation

- `cd bigclaw-go && go test ./internal/intake -run 'TestMapPriority|TestMapSourceIssueToTask|TestConnectorsFetchMinimumIssue|TestConnectorByNameReturnsKnownConnectors|TestClawHostConnectorFetchesInventoryMetadata'`
- `cd bigclaw-go && go test ./internal/product -run 'TestBuildSavedViewCatalog|TestNavigationIncludesCoreConsoleSections|TestHomeForRoleUsesRoleSpecificCards|TestDefaultDesignSystemAndConsoleHelpers|TestBuildDefaultDashboardRunContractIsReleaseReady|TestDashboardRunContractAuditDetectsMissingPaths|TestRenderDashboardRunContractReport|TestContractPathExistsTraversesNestedObjectsAndLists|TestContractPathExistsSkipsNonMapListEntries|TestDashboardContractFormattingHelpers'`
- `cd bigclaw-go && go test ./internal/bootstrap -run 'TestRepoCacheKeyDerivesFromRepoLocator|TestCacheRootForRepoUsesRepoSpecificDirectory|TestBootstrapWorkspaceCreatesSharedWorktreeFromLocalSeed|TestSecondWorkspaceReusesWarmCacheWithoutFullClone|TestBootstrapWorkspaceReusesExistingIssueWorktree|TestCleanupWorkspacePreservesSharedCacheForFutureReuse|TestBootstrapRecoversFromStaleSeedDirectoryWithoutRemoteReclone|TestCleanupWorkspacePrunesWorktreeAndBootstrapBranch|TestValidationReportCoversThreeWorkspacesWithOneCache|TestWithCacheLockSerializesAcrossProcesses|TestBootstrapHelpersUniqueJoinAndPathExists'`
- `git status --short`
