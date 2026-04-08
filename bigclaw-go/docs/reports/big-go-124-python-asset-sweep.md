# BIG-GO-124 Python Asset Sweep

## Scope

This sweep covers the residual migration-script and task-entrypoint paths that
still advertised Python execution after the Go-native automation surface had
already landed:

- `bigclaw-go/scripts/migration/export_live_shadow_bundle`
- `bigclaw-go/scripts/migration/export_live_shadow_bundle.py`
- `bigclaw-go/scripts/migration/live_shadow_scorecard.py`
- `bigclaw-go/scripts/migration/shadow_compare.py`
- `bigclaw-go/scripts/migration/shadow_matrix.py`

## Sweep Result

- Deleted the final extensionless Python-backed migration wrapper at
  `bigclaw-go/scripts/migration/export_live_shadow_bundle`.
- Kept the replacement on the existing Go-native `bigclawctl automation
  migration` command family instead of introducing another shell shim.
- Updated the migration-shadow operator docs and checked-in live-shadow bundle
  metadata so the active closeout path no longer tells operators to invoke
  `python3`.

## Go Or Native Replacement Paths

- `go run ./cmd/bigclawctl automation migration shadow-compare`
- `go run ./cmd/bigclawctl automation migration shadow-matrix`
- `go run ./cmd/bigclawctl automation migration live-shadow-scorecard`
- `go run ./cmd/bigclawctl automation migration export-live-shadow-bundle`
- `bigclaw-go/cmd/bigclawctl/automation_commands.go`
- `bigclaw-go/cmd/bigclawctl/automation_commands_test.go`
- `bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go`

## Validation Commands And Results

- `rg -n "python3 scripts/migration|scripts/migration/(shadow_compare|shadow_matrix|live_shadow_scorecard)\.py|scripts/migration/export_live_shadow_bundle" bigclaw-go/docs/migration-shadow.md bigclaw-go/docs/reports/migration-readiness-report.md bigclaw-go/docs/reports/live-shadow-index.md bigclaw-go/docs/reports/live-shadow-index.json bigclaw-go/docs/reports/live-shadow-summary.json bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/README.md bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go`
  Result: no output; `rg` exited `1`, confirming the scoped migration surfaces no longer advertise Python entrypoints.
- `test ! -e bigclaw-go/scripts/migration/export_live_shadow_bundle`
  Result: exit `0`; the residual extensionless Python wrapper path is absent.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'Test(LiveShadowScorecardBundleStaysAligned|LiveShadowBundleSummaryAndIndexStayAligned|BIGGO124(TargetResidualPythonPathsAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)|BIGGO1577(TargetResidualPythonPathsAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState))$'`
  Result: `ok  	bigclaw-go/internal/regression	0.185s`

## Residual Risk

- Historical sweep artifacts outside the scoped migration/live-shadow surfaces
  still mention the retired shim as prior-lane evidence. Those references are
  intentionally preserved as historical records rather than active operator
  guidance.
