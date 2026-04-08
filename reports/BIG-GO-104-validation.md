# BIG-GO-104 Validation

Date: 2026-04-08

## Scope

Issue: `BIG-GO-104`

Title: `Residual scripts Python sweep F`

This lane removes the last active Python implementation behind the live-shadow
bundle export wrapper and aligns the directly coupled docs, checked-in bundle
artifacts, and regression coverage with the Go-native migration command family.

## Residual Script Surface

- Replaced `bigclaw-go/scripts/migration/export_live_shadow_bundle` with a
  shell-native wrapper.
- Active command replacement: `go run ./cmd/bigclawctl automation migration export-live-shadow-bundle`
- Updated docs:
  - `bigclaw-go/docs/migration-shadow.md`
  - `bigclaw-go/docs/reports/migration-readiness-report.md`
- Updated regression coverage:
  - `bigclaw-go/internal/regression/big_go_104_zero_python_guard_test.go`
  - `bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go`

## Validation Commands

- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-104/bigclaw-go && go run ./cmd/bigclawctl automation migration export-live-shadow-bundle --go-root .`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-104/bigclaw-go && go test -count=1 ./cmd/bigclawctl -run TestAutomationExportLiveShadowBundleBuildsManifest`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-104/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO104(ExportWrapperIsShellNative|LaneReportCapturesSweepState)$|TestLiveShadow(ScorecardBundleStaysAligned|BundleSummaryAndIndexStayAligned|RuntimeDocsStayAligned)$'`

## Validation Results

### Live-shadow export refresh

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-104/bigclaw-go && go run ./cmd/bigclawctl automation migration export-live-shadow-bundle --go-root .
```

Result:

```text
exit 0; refreshed live-shadow summary/index bundle artifacts with Go-native closeout commands and follow-up digest references
```

### Targeted automation command test

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-104/bigclaw-go && go test -count=1 ./cmd/bigclawctl -run TestAutomationExportLiveShadowBundleBuildsManifest
```

Result:

```text
ok  	bigclaw-go/cmd/bigclawctl	5.137s
```

### Targeted regression surface

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-104/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO104(ExportWrapperIsShellNative|LaneReportCapturesSweepState)$|TestLiveShadow(ScorecardBundleStaysAligned|BundleSummaryAndIndexStayAligned|RuntimeDocsStayAligned)$'
```

Result:

```text
ok  	bigclaw-go/internal/regression	5.468s
```

## Git

- Branch: `BIG-GO-104`
- Baseline HEAD before lane commit: `11d9f75`
- Push target: `origin/BIG-GO-104`

## Residual Risk

- The live-shadow export flow still preserves checked-in fixture timestamps and
  bundle IDs, so refreshing the bundle rewrites generated artifact metadata
  rather than introducing new live evidence. That is expected for this repo-native
  migration surface.
