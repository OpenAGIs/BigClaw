# BIGCLAW-173 Workpad

## Plan
- Inspect the existing `automation e2e export-validation-bundle` flow, lane summary structure, and regression coverage in `bigclaw-go`.
- Extend the reusable bundle exporter so `local`, `kubernetes`, and `ray` are represented as an explicit executor validation matrix in a unified evidence bundle.
- Export the evidence bundle in both JSON and Markdown, with lane-level failure root-cause fields suitable for reviewer triage.
- Add/adjust focused unit tests for the bundle exporter and targeted regression/doc assertions.
- Run targeted tests, record exact commands and outcomes here, then commit and push the branch.

## Acceptance
- Support validation matrix coverage for `local`, `kubernetes`, and `ray` executors.
- Emit one unified evidence bundle with both JSON and Markdown outputs.
- Include failure root-cause location fields in the exported report surface.
- Keep changes scoped to the validation bundle/export workflow for this issue.

## Validation
- Targeted Go tests for `cmd/bigclawctl` bundle export logic.
- Targeted regression tests for checked-in report/docs surfaces that depend on the exported bundle.
- Manual verification of generated paths/fields from test fixtures where appropriate.

## Test Log
- `cd bigclaw-go && go test ./cmd/bigclawctl -run 'TestAutomationExportValidationBundle|TestAutomationContinuationPolicyGate|TestRunAllUsesGoBundleCommandsAndDefaultsHoldMode'`
  Result: `ok  	bigclaw-go/cmd/bigclawctl	(cached)`
- `cd bigclaw-go && go test ./internal/regression -run 'TestLiveValidationSummaryStaysAligned|TestLiveValidationIndexJsonExportsBundledSummaries|TestParallelValidationMatrixDocsStayAligned|TestE2EValidationDocsStayAligned|TestParallelValidationEvidenceBundleStaysAligned'`
  Result: `ok  	bigclaw-go/internal/regression	0.470s`
