# BIG-GO-115 Python Asset Sweep

## Scope

This sweep removes the final residual live-shadow bundle helper that still
executed Python semantics inside the Go module:

- `bigclaw-go/scripts/migration/export_live_shadow_bundle`

It also realigns the checked-in live-shadow bundle surfaces that still pointed
reviewers at Python migration commands:

- `bigclaw-go/docs/migration-shadow.md`
- `bigclaw-go/docs/reports/migration-readiness-report.md`
- `bigclaw-go/docs/reports/live-shadow-index.md`
- `bigclaw-go/docs/reports/live-shadow-index.json`
- `bigclaw-go/docs/reports/live-shadow-summary.json`
- `bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/README.md`
- `bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json`
- `bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go`

## Sweep Result

- Deleted the extensionless compatibility wrapper `bigclaw-go/scripts/migration/export_live_shadow_bundle`.
- Kept the canonical live-shadow scorecard and bundle/index workflow on the
  Go-native automation commands:
  - `go run ./cmd/bigclawctl automation migration live-shadow-scorecard`
  - `go run ./cmd/bigclawctl automation migration export-live-shadow-bundle`
- Updated the checked-in live-shadow bundle metadata and reviewer-facing docs so
  the closeout path no longer recommends `python3 scripts/migration/...`.
- Closed the historical `BIG-GO-1577` shim dependency by updating its report and
  guard to expect the native automation implementation instead of the deleted
  compatibility path.

## Go Or Native Replacement Paths

- `bigclaw-go/cmd/bigclawctl/automation_commands.go`
- `bigclaw-go/docs/reports/live-shadow-mirror-scorecard.json`
- `bigclaw-go/docs/reports/live-shadow-summary.json`
- `bigclaw-go/docs/reports/live-shadow-index.json`
- `bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go`
- `bigclaw-go/internal/regression/big_go_115_zero_python_guard_test.go`

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no repository `.py` files remain.
- `find bigclaw-go/scripts bigclaw-go/docs bigclaw-go/internal/regression -type f -name '*.py' 2>/dev/null | sort`
  Result: no Python files remain in the targeted Go-module tooling, docs, or regression directories.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'Test(LiveShadowScorecardBundleStaysAligned|LiveShadowBundleSummaryAndIndexStayAligned|BIGGO115(TargetResidualPythonPathsAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState))$'`
  Result: targeted live-shadow regression coverage passes with the deleted wrapper path held absent and the Go-native bundle surface aligned.

## Residual Risk

- `docs/migration-shadow.md` and `docs/reports/migration-readiness-report.md`
  still mention the older Python `shadow_compare.py` and `shadow_matrix.py`
  examples outside this issue’s narrow residual bundle-helper scope. This lane
  only removes the last live Python-backed exporter path and the stale
  scorecard/bundle command references tied to it.
