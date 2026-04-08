# BIG-GO-104 Workpad

## Context
- Issue: `BIG-GO-104`
- Goal: remove or replace remaining Python scripts and operational wrappers while keeping changes scoped to the active residual surface.
- Active residual identified in this checkout: `bigclaw-go/scripts/migration/export_live_shadow_bundle` is still implemented in Python and some docs/tests still describe Python-based migration bundle commands.

## Scope
- `bigclaw-go/scripts/migration/export_live_shadow_bundle`
- `bigclaw-go/docs/migration-shadow.md`
- `bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go`
- Any directly coupled Go/tests/docs needed to keep the exported bundle surface aligned after replacing the Python wrapper

## Plan
1. Replace the Python live-shadow export wrapper with a repo-native shell entrypoint that dispatches to the existing Go migration automation command.
2. Update the directly coupled docs and regression assertions so the active operator path is Go-only and no longer advertises Python commands.
3. Run targeted tests for the migration automation and regression surface touched by the change.
4. Commit the scoped change set and push branch `BIG-GO-104` to `origin`.

## Acceptance
- No active Python implementation remains at `bigclaw-go/scripts/migration/export_live_shadow_bundle`.
- Active docs and regression surfaces point to `bigclawctl automation migration export-live-shadow-bundle` rather than Python execution.
- Validation records exact commands and results for the touched migration/regression surface.
- Changes stay scoped to this residual script sweep.

## Validation
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration export-live-shadow-bundle --go-root .`
  - Result: command succeeded and refreshed the checked-in live-shadow summary/index bundle artifacts with Go closeout commands.
- `cd bigclaw-go && go test -count=1 ./cmd/bigclawctl -run TestAutomationExportLiveShadowBundleBuildsManifest`
  - Result: `ok  	bigclaw-go/cmd/bigclawctl	5.137s`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO104(ExportWrapperIsShellNative|LaneReportCapturesSweepState)$|TestLiveShadow(ScorecardBundleStaysAligned|BundleSummaryAndIndexStayAligned|RuntimeDocsStayAligned)$'`
  - Result: `ok  	bigclaw-go/internal/regression	5.468s`
