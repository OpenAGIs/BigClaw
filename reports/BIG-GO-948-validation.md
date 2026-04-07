# BIG-GO-948 Validation

## Lane File List

- `tests/test_cross_process_coordination_surface.py`
- `tests/test_followup_digests.py`
- `tests/test_live_shadow_scorecard.py`
- `tests/test_shadow_matrix_corpus.py`
- `tests/test_subscriber_takeover_harness.py`
- `tests/test_validation_bundle_continuation_scorecard.py`
- `tests/test_parallel_refill.py`
- `tests/test_roadmap.py`
- `tests/test_cost_control.py`
- `tests/test_deprecation.py`
- `tests/test_legacy_shim.py`
- `tests/test_service.py`
- `tests/test_pilot.py`
- `tests/test_issue_archive.py`

## Go Replacements

- `bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go`
  - `TestLane8CrossProcessCoordinationSurfaceStaysAligned`
  - `TestLane8ValidationBundleContinuationScorecardStaysAligned`
  - `TestLane8LiveShadowScorecardStaysAligned`
  - `TestLane8ShadowMatrixCorpusCoverageStaysAligned`
  - `TestLane8SubscriberTakeoverHarnessStaysAligned`
  - `TestLane8FollowupDigestsStayAligned`
- `bigclaw-go/internal/refill/queue_repo_fixture_test.go`
  - `TestParallelIssueQueueRepoFixtureSelectionStaysAligned`
- `bigclaw-go/internal/regression/roadmap_contract_test.go`
  - `TestExecutionPackRoadmapDocsStayAligned`
  - `TestExecutionPackRoadmapUniqueOwnersContract`
- `bigclaw-go/internal/regression/deprecation_contract_test.go`
  - `TestLegacyMainlineCompatibilityManifestStaysAligned`
- `bigclaw-go/internal/costcontrol/controller.go`
- `bigclaw-go/internal/costcontrol/controller_test.go`
  - `TestControllerDegradesWhenHighMediumGoesOverBudget`
  - `TestControllerPausesWhenEvenDockerExceedsBudget`
  - `TestControllerRespectsBudgetOverrideAmount`
- `bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json`
- `bigclaw-go/internal/legacyshim/wrappers.go`
- `bigclaw-go/internal/legacyshim/wrappers_test.go`
  - `TestAppendMissingFlagPreservesExistingValues`
  - `TestWorkspaceBootstrapWrapperInjectsGoDefaults`
  - `TestWorkspaceValidateWrapperTranslatesLegacyFlags`
  - `TestGitHubSyncAndRefillWrappersTargetGoShim`
  - `TestWorkspaceRuntimeWrapperTargetsGoShim`
  - `TestRepoRootFromScriptClimbsToRepositoryRoot`
- `bigclaw-go/cmd/bigclawctl/legacy_shim_help_test.go`
  - `TestRunGitHubSyncHelpPrintsUsageAndExitsZero`
  - `TestRunWorkspaceHelpPrintsUsageAndExitsZero`
  - `TestRunCreateIssuesHelpPrintsUsageAndExitsZero`
  - `TestRunDevSmokeHelpPrintsUsageAndExitsZero`
- `bigclaw-go/internal/service/server.go`
- `bigclaw-go/internal/service/server_test.go`
  - `TestRepoGovernanceEnforcerBlocksQuotaAndSidecarFailures`
  - `TestServerEntryHealthMetrics`
- `bigclaw-go/internal/pilot/report.go`
- `bigclaw-go/internal/pilot/report_test.go`
  - `TestImplementationResultReadyWhenKPIsPassAndNoIncidents`
  - `TestRenderPilotImplementationReportContainsReadinessFields`
- `bigclaw-go/internal/issuearchive/archive.go`
- `bigclaw-go/internal/issuearchive/archive_test.go`
  - `TestIssuePriorityArchiveRoundTripPreservesManifestShape`
  - `TestIssuePriorityArchiveAuditFlagsOwnerPriorityCategoryAndOpenP0Gaps`
  - `TestIssuePriorityArchiveAuditRoundTripAndReadyState`
  - `TestRenderIssuePriorityArchiveReportSummarizesFindingsAndRollups`

The deleted Python tests were either:
- report and digest regressions over checked-in `bigclaw-go/docs/reports/*` artifacts, now covered in Go under `bigclaw-go/internal/regression`
- refill queue fixture assertions over `docs/parallel-refill-queue.json`, now covered in Go under `bigclaw-go/internal/refill`

This lane removes redundant Python-only coverage without expanding into unrelated runtime migration domains.

## Validation Commands

- `cd bigclaw-go && go test ./internal/regression -run 'TestLane8|TestCrossProcessCoordinationReadinessDocsStayAligned|TestLiveShadowScorecardBundleStaysAligned|TestProductionCorpus|TestLocalTakeoverReportStaysAligned|TestLiveValidationIndexStaysAligned|TestLiveValidationSummaryStaysAligned|TestFollowUpLaneDocsStayAligned'`
- `cd bigclaw-go && go test ./internal/refill -run TestParallelIssueQueueRepoFixtureSelectionStaysAligned`
- `cd bigclaw-go && go test ./internal/regression -run 'TestExecutionPackRoadmapDocsStayAligned|TestExecutionPackRoadmapUniqueOwnersContract'`
- `cd bigclaw-go && go test ./internal/costcontrol -run TestController`
- `cd bigclaw-go && go test ./internal/regression -run TestLegacyMainlineCompatibilityManifestStaysAligned`
- `cd bigclaw-go && go test ./internal/legacyshim -run 'TestAppendMissingFlagPreservesExistingValues|TestWorkspaceBootstrapWrapperInjectsGoDefaults|TestWorkspaceValidateWrapperTranslatesLegacyFlags|TestGitHubSyncAndRefillWrappersTargetGoShim|TestWorkspaceRuntimeWrapperTargetsGoShim|TestRepoRootFromScriptClimbsToRepositoryRoot'`
- `cd bigclaw-go && go test ./cmd/bigclawctl -run 'TestRunGitHubSyncHelpPrintsUsageAndExitsZero|TestRunWorkspaceHelpPrintsUsageAndExitsZero|TestRunCreateIssuesHelpPrintsUsageAndExitsZero|TestRunDevSmokeHelpPrintsUsageAndExitsZero'`
- `cd bigclaw-go && go test ./internal/service -run 'TestRepoGovernanceEnforcerBlocksQuotaAndSidecarFailures|TestServerEntryHealthMetrics'`
- `cd bigclaw-go && go test ./internal/pilot -run 'TestImplementationResultReadyWhenKPIsPassAndNoIncidents|TestRenderPilotImplementationReportContainsReadinessFields'`
- `cd bigclaw-go && go test ./internal/issuearchive -run 'TestIssuePriorityArchiveRoundTripPreservesManifestShape|TestIssuePriorityArchiveAuditFlagsOwnerPriorityCategoryAndOpenP0Gaps|TestIssuePriorityArchiveAuditRoundTripAndReadyState|TestRenderIssuePriorityArchiveReportSummarizesFindingsAndRollups'`
- `git status --short`

## Latest Validation Result

- `cd bigclaw-go && go test ./internal/pilot -run 'TestImplementationResultReadyWhenKPIsPassAndNoIncidents|TestRenderPilotImplementationReportContainsReadinessFields'`
  - Result: `ok  	bigclaw-go/internal/pilot	0.789s`
- `cd bigclaw-go && go test ./internal/issuearchive -run 'TestIssuePriorityArchiveRoundTripPreservesManifestShape|TestIssuePriorityArchiveAuditFlagsOwnerPriorityCategoryAndOpenP0Gaps|TestIssuePriorityArchiveAuditRoundTripAndReadyState|TestRenderIssuePriorityArchiveReportSummarizesFindingsAndRollups'`
  - Result: `ok  	bigclaw-go/internal/issuearchive	0.445s`

## Residual Risks

- This lane intentionally leaves other remaining `tests/*.py` files untouched when they do not yet have a tight Go regression home or require broader production code migration.
- `tests/test_parallel_validation_bundle.py` and other script-execution Python tests remain outside this scoped delete set because they exercise dynamic script behavior rather than only checked-in report fixtures.
- `tests/test_control_center.py`, `tests/test_operations.py`, and `tests/test_ui_review.py` still need broader Go-native implementation or contract surfaces before their Python tests can be removed safely.

## Remaining Python Test Plan

- `tests/test_parallel_validation_bundle.py`
  - Plan: replace with a Go test once the validation bundle export path moves from Python script orchestration to a Go-native exporter or a stable CLI/API wrapper.
- `tests/test_control_center.py`
  - Plan: attach to an existing Go API/export surface only after the control-center payload contract is fully represented in `bigclaw-go/internal/api`.
- `tests/test_operations.py`
  - Plan: split by feature and migrate only after the operations report and control-center contracts are each represented in Go.
- `tests/test_ui_review.py`
  - Plan: split into smaller report/contract slices and migrate only the parts that already have checked-in Go report surfaces.
- `tests/test_design_system.py`
  - Plan: migrate only after a Go-owned static report/contract exists; otherwise leave until the Python design-system generator is retired.
- `tests/test_dsl.py`
  - Plan: requires a Go-native DSL parser/validator or an explicit decision to retire the Python DSL surface.
- `tests/test_evaluation.py`
  - Plan: requires a Go-native evaluation/report builder or a narrow checked-in report contract that can be asserted in Go.
- `tests/test_issue_archive.py`
  - Completed: replaced by `bigclaw-go/internal/issuearchive/archive.go` and `bigclaw-go/internal/issuearchive/archive_test.go`; Python test deleted.
- `tests/test_pilot.py`
  - Completed: replaced by `bigclaw-go/internal/pilot/report.go` and `bigclaw-go/internal/pilot/report_test.go`; Python test deleted.

The remaining low-size files are not automatically low-risk deletes: they still encode behavior that does not yet exist as a Go-native contract in the repository. Their safe removal depends on implementation migration, not only on test translation.
