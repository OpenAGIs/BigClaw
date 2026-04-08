# BIG-GO-120 Workpad

## Plan

1. Audit the remaining migration-shadow bundle export surface and identify any paths,
   docs, or checked-in artifacts that still depend on Python execution.
2. Replace the `bigclaw-go/scripts/migration/export_live_shadow_bundle` Python
   implementation with a non-Python compatibility wrapper that delegates to the
   existing Go CLI command.
3. Update the narrow regression/docs/artifact expectations that still pin
   `python3 ... export_live_shadow_bundle` so they reflect the Go-native entrypoint.
4. Run targeted regression and command tests, record exact commands and results,
   then commit and push the branch.

## Acceptance

- The repository no longer contains a Python-implemented
  `bigclaw-go/scripts/migration/export_live_shadow_bundle` shim.
- The supported bundle export path is Go-native via
  `bigclawctl automation migration export-live-shadow-bundle`, with the
  compatibility wrapper remaining non-Python.
- Checked-in migration-shadow docs, bundle summaries, and regression tests align on
  the Go-native command surface.
- Targeted tests covering the migration automation command and live-shadow
  regression surfaces pass.

## Validation

- `cd bigclaw-go && go test -count=1 ./cmd/bigclawctl -run 'TestAutomationExportLiveShadowBundleBuildsManifest$'`
  Result: `ok  	bigclaw-go/cmd/bigclawctl	3.119s`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'Test(BIGGO1577GoReplacementPathsRemainAvailable|LiveShadowBundleSummaryAndIndexStayAligned|LiveShadowRuntimeDocsStayAligned)$'`
  Result: `ok  	bigclaw-go/internal/regression	2.265s`
- `find bigclaw-go/scripts/migration -maxdepth 1 -type f | sort`
  Result: `bigclaw-go/scripts/migration/export_live_shadow_bundle`
