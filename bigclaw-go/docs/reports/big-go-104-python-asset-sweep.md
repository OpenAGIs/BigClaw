# BIG-GO-104 Python Asset Sweep

## Scope

`BIG-GO-104` closes the residual migration-bundle shim left behind after the
earlier Python asset sweeps. The active replacement scope is:

- `bigclaw-go/scripts/migration/export_live_shadow_bundle`
- `bigclaw-go/docs/migration-shadow.md`
- `bigclaw-go/docs/reports/migration-readiness-report.md`
- `bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go`

## Sweep Result

- Replaced `bigclaw-go/scripts/migration/export_live_shadow_bundle` from a
  Python implementation with a shell-native wrapper that dispatches to
  `bigclawctl automation migration export-live-shadow-bundle`.
- Updated the active migration docs to reference the Go automation migration
  commands for compare, matrix, scorecard, and bundle export flows.
- Refreshed the checked-in live-shadow bundle artifacts so the closeout command
  surface now advertises Go migration commands instead of Python commands.
- Updated the live-shadow regression surface to assert the active Go-native
  generator and closeout commands.

## Go Or Native Replacement Paths

- `bigclaw-go/scripts/migration/export_live_shadow_bundle`
- `bigclaw-go/cmd/bigclawctl/automation_commands.go`
- `bigclaw-go/docs/migration-shadow.md`
- `bigclaw-go/docs/reports/migration-readiness-report.md`
- `bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go`
- `bigclaw-go/internal/regression/big_go_104_zero_python_guard_test.go`

## Validation Commands And Results

- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration export-live-shadow-bundle --go-root .`
  Result: command completed successfully and rewrote the checked-in live-shadow
  summary/index bundle artifacts with Go closeout commands.
- `cd bigclaw-go && go test -count=1 ./cmd/bigclawctl -run TestAutomationExportLiveShadowBundleBuildsManifest`
  Result: pending in validation artifact.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO104(ExportWrapperIsShellNative|LaneReportCapturesSweepState)$|TestLiveShadow(ScorecardBundleStaysAligned|BundleSummaryAndIndexStayAligned|RuntimeDocsStayAligned)$'`
  Result: pending in validation artifact.
