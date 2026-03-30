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
- `tests/test_validation_policy.py`
- `tests/test_memory.py`
- `tests/test_runtime_matrix.py`
- `tests/test_control_center.py`
- `tests/test_event_bus.py`
- `tests/test_design_system.py`
- `tests/test_console_ia.py`
- `tests/test_workspace_bootstrap.py`
- `tests/test_parallel_validation_bundle.py`
- `tests/test_control_center.py`

Go replacement inventory:

- `bigclaw-go/internal/product/dashboard_run_contract_test.go`
- `bigclaw-go/internal/contract/execution_test.go`
- `bigclaw-go/internal/githubsync/sync_test.go`
- `bigclaw-go/internal/governance/freeze_test.go`
- `bigclaw-go/internal/queue/memory_queue_test.go`
- `bigclaw-go/internal/queue/sqlite_queue_test.go`
- `bigclaw-go/internal/queue/file_queue_test.go`
- `bigclaw-go/internal/repo/repo_surfaces_test.go`
- `bigclaw-go/internal/validationpolicy/policy_test.go`
- `bigclaw-go/internal/taskmemory/store_test.go`
- `bigclaw-go/internal/runtimecompat/runtime_test.go`
- `bigclaw-go/internal/controlcentercompat/control_center_test.go`
- `bigclaw-go/internal/eventbuscompat/event_bus_test.go`
- `bigclaw-go/internal/designsystemcompat/design_system_test.go`
- `bigclaw-go/internal/consoleiacompat/console_ia_test.go`
- `bigclaw-go/internal/bootstrap/bootstrap_test.go`
- `bigclaw-go/scripts/e2e/export_validation_bundle_test.py`
- `bigclaw-go/internal/regression/live_validation_summary_test.go`
- `bigclaw-go/internal/regression/live_validation_index_test.go`
- `bigclaw-go/internal/controlcentercompat/control_center_test.go`
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
6. Add a minimal Go-native replacement for the residual validation-report
   policy contract, then delete the root Python test once parity is covered.
7. Add a minimal Go-native replacement for the residual task-memory reuse
   contract, then delete the root Python test once parity is covered.
8. Add a minimal Go-native replacement for the residual runtime-matrix
   contract, then delete the root Python test once parity is covered.
9. Add a minimal Go-native replacement for the residual control-center queue
   contract, then delete the root Python test once parity is covered.
10. Add a minimal Go-native replacement for the residual event-bus transition
    contract, then delete the root Python test once parity is covered.
11. Add a minimal Go-native replacement for the residual design-system and UI
    acceptance contract, then delete the root Python test once parity is
    covered.
12. Add a minimal Go-native replacement for the residual console information
    architecture and interaction-draft contract, then delete the root Python
    test once parity is covered.
13. Extend the Go workspace bootstrap coverage to match the residual
    cache/worktree validation contract, then delete the root Python test once
    parity is covered.
14. Delete the residual root validation-bundle exporter test once adjacent
    script-local exporter coverage and checked-in regression surfaces fully
    cover the same emitted summary and index contracts.

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
- `cd bigclaw-go && go test ./internal/validationpolicy`
- `cd bigclaw-go && go test ./internal/taskmemory`
- `cd bigclaw-go && go test ./internal/runtimecompat`
- `cd bigclaw-go && go test ./internal/controlcentercompat`
- `cd bigclaw-go && go test ./internal/eventbuscompat`
- `cd bigclaw-go && go test ./internal/designsystemcompat`
- `cd bigclaw-go && go test ./internal/consoleiacompat`
- `cd bigclaw-go && go test ./internal/bootstrap`
- `cd bigclaw-go && python3 scripts/e2e/export_validation_bundle_test.py`
- `cd bigclaw-go && go test ./internal/regression -run 'TestLiveValidationSummaryStaysAligned|TestLiveValidationIndexStaysAligned'`
- `cd bigclaw-go && go test ./internal/controlcentercompat`

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
- `tests/test_validation_policy.py`
  - Deleted.
  - Reason: its required-report closeout policy is now covered directly in
    `bigclaw-go/internal/validationpolicy/policy_test.go`.
- `tests/test_memory.py`
  - Deleted.
  - Reason: its task-pattern reuse contract is now covered directly in
    `bigclaw-go/internal/taskmemory/store_test.go`.
- `tests/test_runtime_matrix.py`
  - Deleted.
  - Reason: its legacy runtime matrix contract is now covered directly in
    `bigclaw-go/internal/runtimecompat/runtime_test.go`.
- `tests/test_control_center.py`
  - Deleted.
  - Reason: its queue peeking, control-center rollup, and shared-view empty
    state contract is now covered directly in
    `bigclaw-go/internal/controlcentercompat/control_center_test.go`.
- `tests/test_event_bus.py`
  - Deleted.
  - Reason: its event-bus transition and persisted ledger audit contract is now
    covered directly in `bigclaw-go/internal/eventbuscompat/event_bus_test.go`.
- `tests/test_design_system.py`
  - Deleted.
  - Reason: its design token, console chrome, information architecture, and UI
    acceptance contract coverage is now covered directly in
    `bigclaw-go/internal/designsystemcompat/design_system_test.go`.
- `tests/test_console_ia.py`
  - Deleted.
  - Reason: its console information architecture, surface-state audit, critical
    page interaction-draft, and `BIG-4203` release-ready builder coverage is
    now covered directly in
    `bigclaw-go/internal/consoleiacompat/console_ia_test.go`.
- `tests/test_workspace_bootstrap.py`
  - Deleted.
  - Reason: its repo-cache derivation, warm-cache reuse, stale-seed recovery,
    cleanup preservation, and bootstrap validation-summary coverage is now
    covered directly in `bigclaw-go/internal/bootstrap/bootstrap_test.go`.
- `tests/test_parallel_validation_bundle.py`
  - Deleted.
  - Reason: its root-level validation bundle exporter coverage is now split
    between adjacent script-local coverage in
    `bigclaw-go/scripts/e2e/export_validation_bundle_test.py` and checked-in
    summary/index regression coverage in
    `bigclaw-go/internal/regression/live_validation_summary_test.go` and
    `bigclaw-go/internal/regression/live_validation_index_test.go`.

### Impact Summary

- `tests/*.py` files before: `38`
- `tests/*.py` files after: `14`
- Net `tests/*.py` reduction: `24`
- Repo `*.py` files before: `108`
- Repo `*.py` files after: `84`
- Net repo `*.py` reduction: `24`
- Repo `*.go` files before: `267`
- Repo `*.go` files after: `281`
- Net repo `*.go` increase: `14`
- Root `pyproject.toml`: absent before, absent after
- Root `setup.py`: absent before, absent after

### Validation Record

- `find tests -name '*.py' | sort | wc -l`
  - Result: `14`
- `find . -name '*.py' | sort | wc -l`
  - Result: `84`
- `find . -name '*.go' | sort | wc -l`
  - Result: `281`
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
  - Result: `.symphony/workpad.md` modified,
    `bigclaw-go/scripts/e2e/export_validation_bundle.py` modified, and
    `tests/test_parallel_validation_bundle.py` deleted
- `cd bigclaw-go && go test ./internal/validationpolicy`
  - Result: `ok  	bigclaw-go/internal/validationpolicy	1.154s`
- `cd bigclaw-go && go test ./internal/taskmemory`
  - Result: `ok  	bigclaw-go/internal/taskmemory	1.240s`
- `cd bigclaw-go && go test ./internal/runtimecompat`
  - Result: `ok  	bigclaw-go/internal/runtimecompat	1.103s`
- `cd bigclaw-go && go test ./internal/controlcentercompat`
  - Result: `ok  	bigclaw-go/internal/controlcentercompat	0.138s`
- `cd bigclaw-go && go test ./internal/eventbuscompat`
  - Result: `ok  	bigclaw-go/internal/eventbuscompat	0.469s`
- `cd bigclaw-go && go test ./internal/designsystemcompat`
  - Result: `ok  	bigclaw-go/internal/designsystemcompat	0.514s`
- `cd bigclaw-go && go test ./internal/consoleiacompat`
  - Result: `ok  	bigclaw-go/internal/consoleiacompat	0.902s`
- `cd bigclaw-go && go test ./internal/bootstrap`
  - Result: `ok  	bigclaw-go/internal/bootstrap	4.163s`
- `cd bigclaw-go && python3 scripts/e2e/export_validation_bundle_test.py`
  - Result: `Ran 4 tests in 0.005s` / `OK`
- `cd bigclaw-go && go test ./internal/regression -run 'TestLiveValidationSummaryStaysAligned|TestLiveValidationIndexStaysAligned'`
  - Result: `ok  	bigclaw-go/internal/regression	1.349s`
