# BIG-GO-115 Workpad

## Context
- Issue: `BIG-GO-115`
- Goal: remove the residual Python-backed live-shadow bundle helper surface and align checked-in docs, reports, and regression coverage to the Go-native `bigclawctl` automation commands.
- Current repo state on entry: the only remaining executable Python tooling artifact is `bigclaw-go/scripts/migration/export_live_shadow_bundle`, an extensionless compatibility wrapper that still contains Python and is still referenced by checked-in live-shadow docs and regression fixtures.

## Scope
- `.symphony/workpad.md`
- `bigclaw-go/scripts/migration/export_live_shadow_bundle`
- `bigclaw-go/docs/migration-shadow.md`
- `bigclaw-go/docs/reports/migration-readiness-report.md`
- `bigclaw-go/docs/reports/live-shadow-index.md`
- `bigclaw-go/docs/reports/live-shadow-index.json`
- `bigclaw-go/docs/reports/live-shadow-summary.json`
- `bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/README.md`
- `bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json`
- `bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go`
- `bigclaw-go/internal/regression/big_go_115_zero_python_guard_test.go`
- `bigclaw-go/docs/reports/big-go-115-python-asset-sweep.md`
- `reports/BIG-GO-115-status.json`
- `reports/BIG-GO-115-validation.md`

## Plan
1. Remove the extensionless Python compatibility wrapper for live-shadow bundle export and keep the checked-in automation surface on the Go-native `bigclawctl` commands.
2. Update checked-in migration docs, live-shadow bundle summaries, and regression expectations so they no longer advertise Python invocations for scorecard and bundle generation.
3. Add lane-specific sweep artifacts and targeted regression coverage that prove the retired Python path stays absent and the Go replacement surface remains available.
4. Run targeted repository inventory and regression commands, record exact commands and exact results, then commit and push the lane branch.

## Acceptance
- `bigclaw-go/scripts/migration/export_live_shadow_bundle` is removed and no checked-in live-shadow docs or bundle metadata still recommend Python execution for scorecard or bundle generation.
- Regression coverage verifies the retired Python path stays absent, the Go-native replacement commands remain referenced, and the lane report captures the sweep state.
- Validation records exact commands and exact results for repository Python inventory, residual directory inventory, and the targeted regression suite used for this issue.
- Changes remain scoped to the residual tooling sweep for `BIG-GO-115`.

## Validation
- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find bigclaw-go/scripts bigclaw-go/docs bigclaw-go/internal/regression -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'Test(LiveShadowScorecardBundleStaysAligned|LiveShadowBundleSummaryAndIndexStayAligned|BIGGO115(TargetResidualPythonPathsAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState))$'`
