# BIG-GO-105 Workpad

## Context
- Issue: `BIG-GO-105`
- Goal: reduce residual Python tooling references in the migration/live-shadow helper path without broad unrelated cleanup.
- Current repo state on entry: clean working tree.

## Scope
- `bigclaw-go/scripts/migration/export_live_shadow_bundle`
- `bigclaw-go/docs/migration-shadow.md`
- `bigclaw-go/docs/reports/migration-readiness-report.md`
- Checked-in live-shadow bundle artifacts under `bigclaw-go/docs/reports/`
- Targeted regression coverage for the live-shadow bundle surface

## Plan
1. Replace the remaining Python-based `export_live_shadow_bundle` helper with a repo-native shell wrapper that dispatches to the existing Go CLI command.
2. Update migration docs and checked-in live-shadow bundle artifacts so they reference Go automation commands instead of Python entrypoints.
3. Adjust targeted regression expectations to match the Go-native surfaces.
4. Run focused validation, record exact commands and results, then commit and push the branch.

## Acceptance
- No remaining Python interpreter dependency in `bigclaw-go/scripts/migration/export_live_shadow_bundle`.
- Live-shadow migration docs point operators to `go run ./cmd/bigclawctl automation migration ...` commands.
- Checked-in live-shadow bundle/index artifacts no longer advertise Python closeout commands.
- Targeted regression tests pass against the touched migration/live-shadow surfaces.

## Validation
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestLiveShadow(ScorecardBundleStaysAligned|BundleSummaryAndIndexStayAligned)$|TestBIGGO1160MigrationDocsListGoReplacements'`
- `cd bigclaw-go && go test -count=1 ./cmd/bigclawctl -run 'TestAutomation(MigrationLiveShadowScorecard|ExportLiveShadowBundle)'`
