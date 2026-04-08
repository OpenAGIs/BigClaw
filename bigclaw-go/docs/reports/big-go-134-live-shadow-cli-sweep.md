# BIG-GO-134 Live Shadow CLI Sweep

## Summary

- Removed the remaining live Python wrapper at `bigclaw-go/scripts/migration/export_live_shadow_bundle`.
- Standardized the live-shadow operator workflow on the Go-native automation commands:
  - `go run ./cmd/bigclawctl automation migration shadow-compare`
  - `go run ./cmd/bigclawctl automation migration shadow-matrix`
  - `go run ./cmd/bigclawctl automation migration live-shadow-scorecard`
  - `go run ./cmd/bigclawctl automation migration export-live-shadow-bundle`
- Refreshed the checked-in live-shadow scorecard and bundle artifacts so their generator metadata and closeout commands match the Go CLI surface.

## Validation

- `go test ./internal/regression -run 'TestBIGGO134|TestLiveShadowScorecardBundleStaysAligned|TestLiveShadowBundleSummaryAndIndexStayAligned'`
- `go run ./cmd/bigclawctl automation migration live-shadow-scorecard`
- `go run ./cmd/bigclawctl automation migration export-live-shadow-bundle`

## Notes

- This change is scoped to the residual live-shadow migration helper surface.
- Historical migration reports still mention prior Python paths where they document earlier sweep evidence; this issue removes the last active wrapper and current operator guidance.
