# BIG-GO-124 Workpad

## Context
- Issue: `BIG-GO-124`
- Title: `Residual scripts Python sweep H`
- Goal: remove or replace the remaining Python migration wrapper, wrapper references, and task entrypoints that still advertise Python execution in the Go-only repository.
- Current repo state on entry: the repository is already extension-level Python-free for `*.py`, but `bigclaw-go/scripts/migration/export_live_shadow_bundle` was still an extensionless Python script and the migration/live-shadow docs and checked-in artifacts still pointed at Python-based commands.

## Scope
- `.symphony/workpad.md`
- `bigclaw-go/scripts/migration/export_live_shadow_bundle`
- `bigclaw-go/docs/migration-shadow.md`
- `bigclaw-go/docs/reports/migration-readiness-report.md`
- `bigclaw-go/docs/reports/live-shadow-mirror-scorecard.json`
- `bigclaw-go/docs/reports/live-shadow-summary.json`
- `bigclaw-go/docs/reports/live-shadow-index.json`
- `bigclaw-go/docs/reports/live-shadow-index.md`
- `bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/live-shadow-mirror-scorecard.json`
- `bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json`
- `bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/README.md`
- `bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go`
- `bigclaw-go/internal/regression/big_go_124_zero_python_guard_test.go`
- `bigclaw-go/internal/regression/big_go_1577_zero_python_guard_test.go`
- `bigclaw-go/docs/reports/big-go-124-python-asset-sweep.md`
- `reports/BIG-GO-124-status.json`
- `reports/BIG-GO-124-validation.md`

## Plan
1. Replace the stale workpad with this issue-specific plan, acceptance criteria, and validation targets before editing code or docs.
2. Remove the residual Python migration wrapper and update checked-in migration/live-shadow surfaces to use the existing Go automation entrypoints exposed through `bigclawctl`.
3. Add lane-specific regression coverage and sweep/report artifacts documenting the retired wrapper path and the required Go/native replacements, including the historical `BIG-GO-1577` guard that still referenced the deleted wrapper.
4. Run targeted commands, record exact commands and results, then commit and push the scoped lane changes to the remote branch.

## Acceptance
- `bigclaw-go/scripts/migration/export_live_shadow_bundle` no longer exists as a Python task entrypoint.
- Migration shadow docs and checked-in live-shadow artifacts refer to `bigclawctl automation migration ...` instead of Python script commands.
- Regression coverage enforces the retired residual path stays absent and the Go migration replacement paths remain available and aligned with checked-in artifacts.
- Validation captures exact commands and exact results for the scoped Python-reference sweep and the targeted regression tests.
- Changes stay scoped to `BIG-GO-124` residual migration-script cleanup artifacts plus the directly impacted historical `BIG-GO-1577` guard/report alignment.

## Validation
- `rg -n "python3 scripts/migration|scripts/migration/(shadow_compare|shadow_matrix|live_shadow_scorecard)\\.py|scripts/migration/export_live_shadow_bundle" bigclaw-go/docs/migration-shadow.md bigclaw-go/docs/reports/migration-readiness-report.md bigclaw-go/docs/reports/live-shadow-index.md bigclaw-go/docs/reports/live-shadow-index.json bigclaw-go/docs/reports/live-shadow-summary.json bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/README.md bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go`
- `test ! -e bigclaw-go/scripts/migration/export_live_shadow_bundle`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'Test(LiveShadowScorecardBundleStaysAligned|LiveShadowBundleSummaryAndIndexStayAligned|BIGGO124(TargetResidualPythonPathsAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)|BIGGO1577(TargetResidualPythonPathsAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState))$'`
