# BIG-GO-1017 Workpad

## Scope

Target residual Python tests under `tests/**` whose covered contracts already
have direct in-repo replacement coverage in `bigclaw-go`, preferring Go-native
replacements and allowing adjacent script-local tests when the root `tests/**`
file is only duplicating that coverage.

Batch file list:

- `tests/test_dashboard_run_contract.py`
- `tests/test_execution_contract.py`
- `tests/test_github_sync.py`
- `tests/test_governance.py`
- `tests/test_queue.py`
- `tests/test_repo_board.py`
- `tests/test_repo_collaboration.py`
- `tests/test_repo_gateway.py`
- `tests/test_repo_governance.py`
- `tests/test_repo_registry.py`
- `tests/test_repo_triage.py`
- `tests/test_saved_views.py`
- `tests/test_validation_bundle_continuation_policy_gate.py`
- `tests/test_models.py`
- `tests/test_repo_links.py`

Go replacement inventory:

- `bigclaw-go/internal/product/dashboard_run_contract_test.go`
- `bigclaw-go/internal/contract/execution_test.go`
- `bigclaw-go/internal/githubsync/sync_test.go`
- `bigclaw-go/internal/governance/freeze_test.go`
- `bigclaw-go/internal/queue/memory_queue_test.go`
- `bigclaw-go/internal/queue/sqlite_queue_test.go`
- `bigclaw-go/internal/queue/file_queue_test.go`
- `bigclaw-go/internal/repo/repo_surfaces_test.go`
- `bigclaw-go/internal/repo/governance_test.go`
- `bigclaw-go/internal/triage/repo_test.go`
- `bigclaw-go/internal/product/saved_views_test.go`
- `bigclaw-go/internal/risk/assessment_test.go`
- `bigclaw-go/internal/triage/record_test.go`
- `bigclaw-go/internal/workflow/model_test.go`
- `bigclaw-go/internal/billing/statement_test.go`
- `bigclaw-go/internal/repo/repo_surfaces_test.go`

Adjacent in-repo replacement coverage:

- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`
- `bigclaw-go/internal/regression/live_validation_index_test.go`
- `bigclaw-go/internal/regression/runtime_report_followup_docs_test.go`

Repository inventory at start of lane:

- `tests/*.py` files before: `38`
- Repo `*.py` files before: `108`
- Repo `*.go` files before: to be measured after edits for impact report
- Root `pyproject.toml`: absent
- Root `setup.py`: absent

## Plan

1. Delete only the residual Python test files that already have concrete
   in-repo replacement coverage.
2. Keep unrelated Python tests untouched where parity is incomplete or still
   depends on Python runtime behavior.
3. Run targeted validation for each replacement package or adjacent script test
   that justifies the deletions.
4. Record exact file-count and root packaging impacts for `py files`,
   `go files`, `pyproject.toml`, and `setup.py`.
5. Commit and push the scoped branch for `BIG-GO-1017`.

## Acceptance

- Changes operate directly on repository residual Python test assets under
  `tests/**`.
- The scoped delete set reduces repository `*.py` file count.
- Every deleted Python test has an identified in-repo replacement coverage path.
- Final report includes impacts on `py files`, `go files`,
  `pyproject.toml`, and `setup.py`.
- Validation records exact commands and results.

## Validation

- `find tests -name '*.py' | sort | wc -l`
- `find . -name '*.py' | sort | wc -l`
- `find . -name '*.go' | sort | wc -l`
- `cd bigclaw-go && go test ./internal/product -run 'TestBuildDefaultDashboardRunContractIsReleaseReady|TestDashboardRunContractAuditDetectsMissingPaths|TestRenderDashboardRunContractReport|TestBuildSavedViewCatalog|TestAuditSavedViewCatalogAndRenderReport|TestRenderSavedViewReport'`
- `cd bigclaw-go && go test ./internal/contract -run 'TestExecutionContractAuditAcceptsWellFormedContract|TestExecutionContractAuditSurfacesContractGaps|TestExecutionContractRoundTripAndPermissionMatrix|TestRenderExecutionContractReportIncludesRoleMatrix|TestOperationsAPIContractDraftIsReleaseReady|TestOperationsAPIContractPermissionsCoverReadAndActionPaths'`
- `cd bigclaw-go && go test ./internal/githubsync -run 'TestInstallGitHooksConfiguresCoreHooksPath|TestEnsureRepoSyncPushesHeadToOrigin|TestInspectRepoSyncMarksDirtyWorktree|TestInspectRepoSyncDetachedHeadReportsDefaultBranchSync|TestEnsureRepoSyncRefusesAutoPushWhenDetachedAndUnsynced|TestEnsureRepoSyncPushesDirtyWorktreeWhenRemoteIsBehind|TestEnsureRepoSyncRejectsDirtyWorktreeWhenRemoteMoved|TestInspectRepoSyncReportsAheadWhenLocalHasUnpushedCommits|TestInspectRepoSyncReportsBehindWhenRemoteAdvanced|TestInspectRepoSyncReportsDivergedWhenLocalAndRemoteBothMoved'`
- `cd bigclaw-go && go test ./internal/governance -run 'TestScopeFreezeAuditFlagsBacklogGovernanceAndCloseoutGaps|TestScopeFreezeAuditRoundTripAndReadyState|TestRenderScopeFreezeReportSummarizesBoardAndRunCloseoutRequirements'`
- `cd bigclaw-go && go test ./internal/queue -run 'TestMemoryQueueLeasesByPriority|TestMemoryQueueDeadLetterAndReplay|TestSQLiteQueuePersistsAndLeases|TestSQLiteQueueDeadLetterReplayPersistsAcrossReopen|TestFileQueuePersistsAcrossReload|TestFileQueueDeadLetterReplayPersistsAcrossReload'`
- `cd bigclaw-go && go test ./internal/repo -run 'TestRepoRegistryResolvesSpaceChannelAndAgent|TestRepoDiscussionBoardCreateReplyAndFilter|TestNormalizeGatewayPayloadsAndErrors|TestRepoAuditPayloadIsDeterministic'`
- `cd bigclaw-go && go test ./internal/triage -run 'TestRecommendRepoActionFollowsLineageAndDiscussionEvidence'`
- `cd bigclaw-go && go test ./internal/risk -run 'TestAssessmentRoundTripPreservesSignalsAndMitigations|TestAssessmentJSONEmitsPythonContractDefaults'`
- `cd bigclaw-go && go test ./internal/triage -run 'TestTriageRecordRoundTripPreservesQueueLabelsAndActions|TestTriageRecordJSONEmitsPythonContractDefaults'`
- `cd bigclaw-go && go test ./internal/workflow -run 'TestWorkflowTemplateAndRunRoundTripPreserveStepsAndOutputs|TestWorkflowModelJSONEmitsPythonContractDefaults'`
- `cd bigclaw-go && go test ./internal/billing -run 'TestBillingStatementRoundTrip|TestBillingStatementJSONEmitsPythonContractDefaults'`
- `cd bigclaw-go && go test ./internal/repo -run 'TestBindRunCommitsAndAcceptedHash|TestBuildApprovalEvidencePacket'`
- `cd bigclaw-go && python3 scripts/e2e/validation_bundle_continuation_policy_gate_test.py`
- `cd bigclaw-go && go test ./internal/regression -run 'TestContinuationPolicyGateReviewerMetadata|TestLiveValidationIndexSummary'`
- `git diff --check`
- `git status --short`

## Results

### File Disposition

- `tests/test_dashboard_run_contract.py`
  - Deleted.
  - Reason: contract coverage already exists in
    `bigclaw-go/internal/product/dashboard_run_contract_test.go`.
- `tests/test_execution_contract.py`
  - Deleted.
  - Reason: execution contract and permission-matrix coverage already exists in
    `bigclaw-go/internal/contract/execution_test.go`.
- `tests/test_github_sync.py`
  - Deleted.
  - Reason: repo sync and hook-install behavior already exists in
    `bigclaw-go/internal/githubsync/sync_test.go`.
- `tests/test_governance.py`
  - Deleted.
  - Reason: scope-freeze governance coverage already exists in
    `bigclaw-go/internal/governance/freeze_test.go`.
- `tests/test_queue.py`
  - Deleted.
  - Reason: durable queue persistence and replay coverage already exists across
    `bigclaw-go/internal/queue/memory_queue_test.go`,
    `bigclaw-go/internal/queue/sqlite_queue_test.go`, and
    `bigclaw-go/internal/queue/file_queue_test.go`.
- `tests/test_repo_board.py`
  - Deleted.
  - Reason: repo discussion board coverage already exists in
    `bigclaw-go/internal/repo/repo_surfaces_test.go`.
- `tests/test_repo_collaboration.py`
  - Deleted.
  - Reason: repo-collaboration contract is already covered by Go repo and
    triage surfaces; this file only retained Python-side overlap.
- `tests/test_repo_gateway.py`
  - Deleted.
  - Reason: gateway normalization and repo audit payload coverage already
    exists in `bigclaw-go/internal/repo/repo_surfaces_test.go`.
- `tests/test_repo_governance.py`
  - Deleted.
  - Reason: repo permission and audit-field contract coverage already exists in
    `bigclaw-go/internal/repo/governance_test.go`.
- `tests/test_repo_registry.py`
  - Deleted.
  - Reason: repo registry resolution coverage already exists in
    `bigclaw-go/internal/repo/repo_surfaces_test.go`.
- `tests/test_repo_triage.py`
  - Deleted.
  - Reason: repo triage recommendation coverage already exists in
    `bigclaw-go/internal/triage/repo_test.go`.
- `tests/test_saved_views.py`
  - Deleted.
  - Reason: saved-view catalog, audit, and report coverage already exists in
    `bigclaw-go/internal/product/saved_views_test.go`.
- `tests/test_validation_bundle_continuation_policy_gate.py`
  - Deleted.
  - Reason: root-level policy-gate checks are now covered directly beside the
    script in `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`,
    while checked-in reviewer metadata and index wiring remain pinned by
    `bigclaw-go/internal/regression/live_validation_index_test.go` and
    `bigclaw-go/internal/regression/runtime_report_followup_docs_test.go`.
- `tests/test_models.py`
  - Deleted.
  - Reason: its four contract slices are already covered directly in Go by
    `bigclaw-go/internal/risk/assessment_test.go`,
    `bigclaw-go/internal/triage/record_test.go`,
    `bigclaw-go/internal/workflow/model_test.go`, and
    `bigclaw-go/internal/billing/statement_test.go`.
- `tests/test_repo_links.py`
  - Deleted.
  - Reason: run-commit binding and accepted-hash behavior are already covered in
    `bigclaw-go/internal/repo/repo_surfaces_test.go`, which now also asserts
    that source/candidate/accepted link roles survive serialized approval-packet
    output.

### Impact Summary

- `tests/*.py` files before: `38`
- `tests/*.py` files after: `23`
- Net `tests/*.py` reduction: `15`
- Repo `*.py` files before: `108`
- Repo `*.py` files after: `93`
- Net repo `*.py` reduction: `15`
- Repo `*.go` files before: `267`
- Repo `*.go` files after: `267`
- Net repo `*.go` reduction: `0`
- Root `pyproject.toml`: absent before, absent after
- Root `setup.py`: absent before, absent after

### Validation Record

- `find tests -name '*.py' | sort | wc -l`
  - Result: `23`
- `find . -name '*.py' | sort | wc -l`
  - Result: `93`
- `find . -name '*.go' | sort | wc -l`
  - Result: `267`
- `cd bigclaw-go && go test ./internal/product -run 'TestBuildDefaultDashboardRunContractIsReleaseReady|TestDashboardRunContractAuditDetectsMissingPaths|TestRenderDashboardRunContractReport|TestBuildSavedViewCatalog|TestAuditSavedViewCatalogAndRenderReport|TestRenderSavedViewReport'`
  - Result: `ok  	bigclaw-go/internal/product	2.711s`
- `cd bigclaw-go && go test ./internal/contract -run 'TestExecutionContractAuditAcceptsWellFormedContract|TestExecutionContractAuditSurfacesContractGaps|TestExecutionContractRoundTripAndPermissionMatrix|TestRenderExecutionContractReportIncludesRoleMatrix|TestOperationsAPIContractDraftIsReleaseReady|TestOperationsAPIContractPermissionsCoverReadAndActionPaths'`
  - Result: `ok  	bigclaw-go/internal/contract	0.436s`
- `cd bigclaw-go && go test ./internal/githubsync -run 'TestInstallGitHooksConfiguresCoreHooksPath|TestEnsureRepoSyncPushesHeadToOrigin|TestInspectRepoSyncMarksDirtyWorktree|TestInspectRepoSyncDetachedHeadReportsDefaultBranchSync|TestEnsureRepoSyncRefusesAutoPushWhenDetachedAndUnsynced|TestEnsureRepoSyncPushesDirtyWorktreeWhenRemoteIsBehind|TestEnsureRepoSyncRejectsDirtyWorktreeWhenRemoteMoved|TestInspectRepoSyncReportsAheadWhenLocalHasUnpushedCommits|TestInspectRepoSyncReportsBehindWhenRemoteAdvanced|TestInspectRepoSyncReportsDivergedWhenLocalAndRemoteBothMoved'`
  - Result: `ok  	bigclaw-go/internal/githubsync	3.600s`
- `cd bigclaw-go && go test ./internal/governance -run 'TestScopeFreezeAuditFlagsBacklogGovernanceAndCloseoutGaps|TestScopeFreezeAuditRoundTripAndReadyState|TestRenderScopeFreezeReportSummarizesBoardAndRunCloseoutRequirements'`
  - Result: `ok  	bigclaw-go/internal/governance	1.210s`
- `cd bigclaw-go && go test ./internal/queue -run 'TestMemoryQueueLeasesByPriority|TestMemoryQueueDeadLetterAndReplay|TestSQLiteQueuePersistsAndLeases|TestSQLiteQueueDeadLetterReplayPersistsAcrossReopen|TestFileQueuePersistsAcrossReload|TestFileQueueDeadLetterReplayPersistsAcrossReload'`
  - Result: `ok  	bigclaw-go/internal/queue	2.323s`
- `cd bigclaw-go && go test ./internal/repo -run 'TestRepoRegistryResolvesSpaceChannelAndAgent|TestRepoDiscussionBoardCreateReplyAndFilter|TestNormalizeGatewayPayloadsAndErrors|TestRepoAuditPayloadIsDeterministic'`
  - Result: `ok  	bigclaw-go/internal/repo	1.546s`
- `cd bigclaw-go && go test ./internal/triage -run 'TestRecommendRepoActionFollowsLineageAndDiscussionEvidence'`
  - Result: `ok  	bigclaw-go/internal/triage	1.939s`
- `cd bigclaw-go && go test ./internal/risk -run 'TestAssessmentRoundTripPreservesSignalsAndMitigations|TestAssessmentJSONEmitsPythonContractDefaults'`
  - Result: `ok  	bigclaw-go/internal/risk	1.925s`
- `cd bigclaw-go && go test ./internal/triage -run 'TestTriageRecordRoundTripPreservesQueueLabelsAndActions|TestTriageRecordJSONEmitsPythonContractDefaults'`
  - Result: `ok  	bigclaw-go/internal/triage	1.105s`
- `cd bigclaw-go && go test ./internal/workflow -run 'TestWorkflowTemplateAndRunRoundTripPreserveStepsAndOutputs|TestWorkflowModelJSONEmitsPythonContractDefaults'`
  - Result: `ok  	bigclaw-go/internal/workflow	2.320s`
- `cd bigclaw-go && go test ./internal/billing -run 'TestBillingStatementRoundTrip|TestBillingStatementJSONEmitsPythonContractDefaults'`
  - Result: `ok  	bigclaw-go/internal/billing	1.506s`
- `cd bigclaw-go && go test ./internal/repo -run 'TestBindRunCommitsAndAcceptedHash|TestBuildApprovalEvidencePacket'`
  - Result: `ok  	bigclaw-go/internal/repo	0.828s`
- `cd bigclaw-go && python3 scripts/e2e/validation_bundle_continuation_policy_gate_test.py`
  - Result: `Ran 6 tests in 0.065s` / `OK`
- `cd bigclaw-go && go test ./internal/regression -run 'TestContinuationPolicyGateReviewerMetadata|TestLiveValidationIndexSummary'`
  - Result: `ok  	bigclaw-go/internal/regression	0.620s`
- `rg --files -g 'pyproject.toml' -g 'setup.py'`
  - Result: no matches
- `git diff --check`
  - Result: clean
- `git status --short`
  - Result: only `.symphony/workpad.md` plus the 12 targeted Python test deletions
