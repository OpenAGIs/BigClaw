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
- `cd bigclaw-go && go test ./cmd/bigclawctl -run 'TestAutomationExportValidationBundle|TestAutomationContinuationPolicyGate|TestRunAllUsesGoBundleCommandsAndDefaultsHoldMode|TestRunAutomationExportValidationBundleCommandAllowsDisabledLanePaths'`
  Result: `ok  	bigclaw-go/cmd/bigclawctl	1.639s`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e export-validation-bundle --go-root . --run-id 20260316T140138Z --bundle-dir docs/reports/live-validation-runs/20260316T140138Z --summary-path docs/reports/live-validation-summary.json --index-path docs/reports/live-validation-index.md --manifest-path docs/reports/live-validation-index.json --run-local=true --run-kubernetes=true --run-ray=true --run-broker=false --broker-bootstrap-summary-path docs/reports/broker-bootstrap-review-summary.json --validation-status 0 --local-report-path docs/reports/live-validation-runs/20260316T140138Z/sqlite-smoke-report.json --local-stdout-path docs/reports/live-validation-runs/20260316T140138Z/local.stdout.log --local-stderr-path docs/reports/live-validation-runs/20260316T140138Z/local.stderr.log --kubernetes-report-path docs/reports/live-validation-runs/20260316T140138Z/kubernetes-live-smoke-report.json --kubernetes-stdout-path docs/reports/live-validation-runs/20260316T140138Z/kubernetes.stdout.log --kubernetes-stderr-path docs/reports/live-validation-runs/20260316T140138Z/kubernetes.stderr.log --ray-report-path docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json --ray-stdout-path docs/reports/live-validation-runs/20260316T140138Z/ray.stdout.log --ray-stderr-path docs/reports/live-validation-runs/20260316T140138Z/ray.stderr.log --json=false`
  Result: `exit 0`; regenerated canonical and bundled `live-validation-summary`, `live-validation-index`, and `parallel-validation-evidence-bundle` surfaces with `root_cause_location_kind` and matrix/index markdown updates.
- `cd bigclaw-go && go test ./internal/regression -run 'TestLiveValidationSummaryStaysAligned|TestLiveValidationIndexStaysAligned|TestLiveValidationIndexSummaryPointers|TestParallelValidationEvidenceBundleStaysAligned|TestParallelValidationMatrixDocsStayAligned|TestE2EValidationDocsStayAligned'`
  Result: `ok  	bigclaw-go/internal/regression	0.504s`
