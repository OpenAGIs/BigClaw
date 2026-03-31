Issue: BIG-GO-1028

Plan
- Identify a small tranche of remaining Python test files that already have clear Go-native ownership or need only minimal Go-side test coverage.
- Port the selected Python assertions into Go tests under existing `bigclaw-go/internal/**` packages, adding only the smallest supporting Go code required to preserve behavior.
- Delete the migrated Python test files so this issue measurably reduces repository `.py` test inventory without expanding scope into unrelated Python modules.
- Run targeted Go validation for the migrated tranche, capture exact commands and outcomes, then commit and push the scoped branch changes.

Acceptance
- Changes stay scoped to the remaining Python tests tranche for this issue plus directly coupled Go test/support files.
- Repository `.py` file count decreases by removing migrated Python test files.
- Repository `.go` file count only changes where needed to host migrated Go-native coverage.
- `pyproject.toml`, `setup.py`, and `setup.cfg` remain unchanged unless the selected tranche proves they are directly affected.
- Final report includes the impact on `.py` count, `.go` count, and `pyproject/setup*` files.

Validation
- `find tests -maxdepth 1 -name 'test_*.py' | sort | wc -l`
- `find . -path './.git' -prune -o -name '*.py' -print | sort | wc -l`
- `find . -path './.git' -prune -o -name '*.go' -print | sort | wc -l`
- `gofmt -w bigclaw-go/internal/repo/board.go bigclaw-go/internal/repo/collaboration.go bigclaw-go/internal/repo/collaboration_test.go bigclaw-go/internal/repo/repo_surfaces_test.go bigclaw-go/internal/queue/file_queue.go bigclaw-go/internal/queue/file_queue_test.go bigclaw-go/internal/validationpolicy/policy.go bigclaw-go/internal/validationpolicy/policy_test.go bigclaw-go/internal/memory/store.go bigclaw-go/internal/memory/store_test.go bigclaw-go/internal/workflow/definition.go bigclaw-go/internal/workflow/definition_runner_test.go`
- `go test ./internal/repo -run 'TestPermissionMatrixResolvesRoles|TestAuditFieldContractIsDeterministic|TestRepoRegistryResolvesSpaceChannelAndAgent|TestRepoRegistryJSONRoundTripPreservesSpacesAndAgents|TestRepoPostToCollaborationCommentPreservesAnchorAndResolvedState|TestMergeCollaborationThreadsCombinesNativeAndRepoSurfaces|TestRepoDiscussionBoardCreateReplyAndFilter|TestNormalizeGatewayPayloadsAndErrors|TestNormalizeGatewayPayloadsReturnDecodeErrors|TestBindRunCommitsAndAcceptedHash|TestBuildApprovalEvidencePacket'`
- `go test ./internal/triage -run 'TestRecommendRepoActionFollowsLineageAndDiscussionEvidence|TestApprovalEvidencePacketCapturesAcceptedAndCandidateLinks'`
- `go test ./internal/queue -run 'TestFileQueuePersistsAcrossReload|TestFileQueueCreatesParentDirectoryAndPreservesTaskPayload|TestFileQueueDeadLetterReplayPersistsAcrossReload|TestFileQueueLoadsLegacyListStorage'`
- `go test ./internal/validationpolicy -run 'TestValidationPolicyBlocksIssueCloseWithoutRequiredReports|TestValidationPolicyAllowsIssueCloseWhenReportsComplete'`
- `go test ./internal/memory -run 'TestStoreReusesHistoryAndInjectsRules'`
- `go test ./internal/workflow -run 'TestDefinitionParsesAndRendersTemplates|TestDefinitionEngineRunsDefinitionEndToEnd|TestDefinitionEngineRejectsUnknownStepKind|TestDefinitionEngineManualApprovalClosesHighRiskTask'`
- `go test ./cmd/bigclawctl -run 'TestAutomationContinuationPolicyGateReturnsPolicyGoWhenInputsPass|TestAutomationContinuationPolicyGateReturnsPolicyHoldWithFailures'`
- `go test ./internal/regression -run 'TestContinuationPolicyGateReviewerMetadata'`
- `gofmt -w bigclaw-go/internal/workflow/orchestration.go bigclaw-go/internal/workflow/orchestration_test.go`
- `go test ./internal/workflow -run 'TestCrossDepartmentOrchestratorPlansHandoffs|TestPremiumOrchestrationPolicyConstrainsStandardTier|TestBuildHandoffRequest|TestRenderOrchestrationPlanListsHandoffsAndPolicy'`
- `go test ./internal/scheduler -run 'TestSchedulerAssessmentBuildsUpgradeHandoffForStandardTier|TestSchedulerAssessmentOmitsHandoffWhenStandardPlanFits|TestSchedulerAssessmentBuildsSecurityHandoffForRejectedDecision'`
- `go test ./internal/worker -run 'TestRuntimePublishesOrchestrationAssessmentOnRoutedEvent|TestRuntimePublishesRejectedDecisionHandoffBeforeRetry'`
- `git diff --stat`
- `git status --short`

Latest tranche results
- Removed `tests/test_orchestration.py` after adding Go-native orchestration plan rendering coverage in `bigclaw-go/internal/workflow`.
- `find tests -maxdepth 1 -name 'test_*.py' | sort | wc -l` -> `18`
- `find . -path './.git' -prune -o -name '*.py' -print | sort | wc -l` -> `69`
- `find . -path './.git' -prune -o -name '*.go' -print | sort | wc -l` -> `289`
- `go test ./internal/workflow -run 'TestCrossDepartmentOrchestratorPlansHandoffs|TestPremiumOrchestrationPolicyConstrainsStandardTier|TestBuildHandoffRequest|TestRenderOrchestrationPlanListsHandoffsAndPolicy'` -> `ok   bigclaw-go/internal/workflow`
- `go test ./internal/scheduler -run 'TestSchedulerAssessmentBuildsUpgradeHandoffForStandardTier|TestSchedulerAssessmentOmitsHandoffWhenStandardPlanFits|TestSchedulerAssessmentBuildsSecurityHandoffForRejectedDecision'` -> `ok   bigclaw-go/internal/scheduler`
- `go test ./internal/worker -run 'TestRuntimePublishesOrchestrationAssessmentOnRoutedEvent|TestRuntimePublishesRejectedDecisionHandoffBeforeRetry'` -> `ok   bigclaw-go/internal/worker`

- Removed `tests/test_workspace_bootstrap.py` after porting the remaining workspace cache/bootstrap lifecycle coverage into `bigclaw-go/internal/bootstrap/bootstrap_test.go`.
- `find tests -maxdepth 1 -name 'test_*.py' | sort | wc -l` -> `17`
- `find . -path './.git' -prune -o -name '*.py' -print | sort | wc -l` -> `68`
- `find . -path './.git' -prune -o -name '*.go' -print | sort | wc -l` -> `289`
- `gofmt -w bigclaw-go/internal/bootstrap/bootstrap_test.go` -> `ok`
- `go test ./internal/bootstrap -run 'TestRepoCacheKeyDerivesFromRepoLocator|TestCacheRootForRepoUsesRepoSpecificDirectory|TestBootstrapWorkspaceCreatesSharedWorktreeFromLocalSeed|TestSecondWorkspaceReusesWarmCacheWithoutFullClone|TestBootstrapWorkspaceReusesExistingIssueWorktree|TestCleanupWorkspacePreservesSharedCacheForFutureReuse|TestBootstrapRecoversFromStaleSeedDirectoryWithoutRemoteReclone|TestCleanupWorkspacePrunesWorktreeAndBootstrapBranch|TestValidationReportCoversThreeWorkspacesWithOneCache'` -> `ok   bigclaw-go/internal/bootstrap`
- Removed `tests/test_control_center.py` after porting queue-control shared view, reassign action, run-aware execution media, and persistent queue priority coverage into `bigclaw-go/internal/reporting` and `bigclaw-go/internal/queue`.
- `find tests -maxdepth 1 -name 'test_*.py' | sort | wc -l` -> `16`
- `find . -path './.git' -prune -o -name '*.py' -print | sort | wc -l` -> `67`
- `find . -path './.git' -prune -o -name '*.go' -print | sort | wc -l` -> `289`
- `gofmt -w bigclaw-go/internal/reporting/reporting.go bigclaw-go/internal/reporting/reporting_test.go bigclaw-go/internal/queue/file_queue_test.go` -> `ok`
- `go test ./internal/reporting -run 'TestBuildRenderAndWriteQueueControlCenterBundle|TestBuildQueueControlCenterWithRunsSummarizesQueueAndExecutionMedia|TestRenderQueueControlCenterWithSharedViewEmptyState'` -> `ok   bigclaw-go/internal/reporting`
- `go test ./internal/queue -run 'TestFileQueueLeasesByPriority|TestFileQueuePersistsAcrossReload|TestFileQueueCreatesParentDirectoryAndPreservesTaskPayload|TestFileQueueDeadLetterReplayPersistsAcrossReload|TestFileQueueLoadsLegacyListStorage'` -> `ok   bigclaw-go/internal/queue`
