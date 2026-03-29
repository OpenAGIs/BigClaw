# BIG-GO-948 Validation

## Completed Work

This continuation closes the `test_operations.py` portion of the remaining Lane 8 Python test debt and materially reduces `tests/test_reports.py` by moving its bounded report-studio, pilot, checklist, issue-closure, and shared-view assertions into a dedicated Go-native parity owner.

Completed in this change:
- Deleted `tests/test_operations.py`.
- Added `bigclaw-go/internal/reportingparity/reportingparity.go`.
- Added `bigclaw-go/internal/reportingparity/reportingparity_test.go`.
- Removed the migrated report-studio, pilot, checklist, issue-closure, and shared-view assertions from `tests/test_reports.py`.
- Updated `src/bigclaw/planning.py` so the `candidate-ops-hardening` validation command now points at Go packages instead of removed pytest suites.
- Updated `bigclaw-go/internal/planningparity/planningparity.go` to mirror the new Go-native validation command and evidence links.
- Updated `bigclaw-go/internal/planningparity/planningparity_test.go` so parity assertions match the Go-native command and evidence targets.

## Lane File List

Lane 8 Python files remaining at the start of this continuation:
- `tests/test_operations.py`
- `tests/test_reports.py`
- `tests/test_ui_review.py`
- `tests/conftest.py`

Status after this continuation:
- `tests/test_operations.py`
  - Status: deleted.
  - Replacement: Go-native reporting coverage in `bigclaw-go/internal/reporting/reporting_test.go`.
- `tests/test_reports.py`
  - Status: retained but reduced.
  - Migrated to Go: report-studio, pilot scorecard/portfolio, launch checklist, final-delivery checklist, issue-closure, validation-report write/existence, and shared-view collaboration assertions.
  - Remaining in Python: auto-triage, takeover queue, orchestration canvas/portfolio, billing entitlements, and timezone-specific validation-report timestamp checks.
- `tests/test_ui_review.py`
  - Status: retained.
  - Reason: no Go-native `ui_review` implementation or parity/regression owner exists in this branch.
- `tests/conftest.py`
  - Status: retained.
  - Reason: still required bootstrap for the remaining Python test files.

## Go Replacement

`tests/test_operations.py` is replaced by existing Go coverage in:
- `bigclaw-go/internal/reporting/reporting_test.go`
  - `TestBuildWeeklyReportRendersExpandedMarkdown`
  - `TestBuildOperationsMetricSpec`
  - `TestWriteWeeklyOperationsBundle`
  - `TestAuditDashboardBuilderFlagsGovernanceGaps`
  - `TestRenderAndWriteDashboardBuilderBundle`
  - `TestBuildEngineeringOverviewFromTasksAndEvents`
  - `TestRenderAndWriteEngineeringOverviewBundle`
  - `TestBuildPolicyPromptVersionCenterSummarizesRevisionDiffs`
  - `TestRenderAndWritePolicyPromptVersionCenterBundle`
  - `TestWriteWeeklyOperationsBundleWithVersionCenter`
  - `TestWriteWeeklyOperationsBundleWithCenters`
  - `TestRenderAndWriteRegressionCenterBundle`

Supporting Go-native planning traceability updated in:
- `bigclaw-go/internal/planningparity/planningparity.go`
- `bigclaw-go/internal/planningparity/planningparity_test.go`

`tests/test_reports.py` is partially replaced by:
- `bigclaw-go/internal/reportingparity/reportingparity_test.go`
  - `TestRenderAndWriteReport`
  - `TestConsoleActionStateReflectsEnabledFlag`
  - `TestReportStudioRendersNarrativeSectionsAndExportBundle`
  - `TestReportStudioRequiresSummaryAndCompleteSections`
  - `TestRenderPilotScorecardIncludesROIAndRecommendation`
  - `TestPilotScorecardReturnsHoldWhenValueIsNegative`
  - `TestIssueClosureRequiresNonEmptyValidationReport`
  - `TestIssueClosureBlocksFailedValidationReport`
  - `TestIssueClosureAllowsCompletedValidationReport`
  - `TestLaunchChecklistAutoLinksDocumentationStatus`
  - `TestFinalDeliveryChecklistTracksRequiredOutputsAndRecommendedDocs`
  - `TestIssueClosureBlocksIncompleteLinkedLaunchChecklist`
  - `TestIssueClosureBlocksMissingRequiredFinalDeliveryOutputs`
  - `TestIssueClosureAllowsWhenRequiredFinalDeliveryOutputsExist`
  - `TestIssueClosureAllowsWhenLinkedLaunchChecklistIsReady`
  - `TestRenderPilotPortfolioReportSummarizesCommercialReadiness`
  - `TestRenderSharedViewContextIncludesCollaborationAnnotations`

## Delete Or Migration Plan

- `tests/test_reports.py`
  - Plan: migrate the remaining auto-triage, takeover/orchestration, billing-entitlements, and timezone-specific report assertions into bounded Go owners before deleting the file.
- `tests/test_ui_review.py`
  - Plan: introduce a dedicated Go-native review-pack package or a Go regression owner for the review-pack contract before deleting the Python suite.

## Validation Commands

- `cd bigclaw-go && go test ./internal/reporting`
- `cd bigclaw-go && go test ./internal/planningparity`
- `cd bigclaw-go && go test ./internal/reportingparity`
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py -q`
- `git status --short`

## Latest Validation Result

- `cd bigclaw-go && go test ./internal/reporting`
  - Result: `ok  	bigclaw-go/internal/reporting	1.407s`
- `cd bigclaw-go && go test ./internal/planningparity`
  - Result: `ok  	bigclaw-go/internal/planningparity	0.839s`
- `cd bigclaw-go && go test ./internal/reportingparity`
  - Result: `ok  	bigclaw-go/internal/reportingparity	0.873s`
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py -q`
  - Result: `17 passed in 0.09s`

## Residual Risks

- `tests/test_reports.py` is smaller now, but its remaining assertions still span multiple owners across triage, flow/orchestration, and billing/report APIs.
- `tests/test_ui_review.py` remains Python-owned because this branch does not yet contain a Go-native review-pack surface to replace it.
- `tests/conftest.py` cannot be removed until the remaining Python tests are migrated or deleted.
