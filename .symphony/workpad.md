# BIG-GO-1485

## Plan
- Establish the tracked Python-file baseline and identify the remaining reporting/observability Python helpers still used by checked-in live-shadow evidence flows.
- Replace the scoped Python helpers with Go-owned implementations in `bigclaw-go`, update docs/tests/references to the Go entrypoints, and delete the retired Python files.
- Rebuild the checked-in live-shadow bundle artifacts with the Go tools, run targeted validation, and capture exact before/after inventory evidence.
- Commit the scoped change set on `BIG-GO-1485` and push the branch to `origin`.

## Acceptance
- Repository tracked Python file count decreases from the measured baseline.
- The remaining live-shadow reporting/bundle workflow no longer depends on `bigclaw-go/scripts/migration/live_shadow_scorecard.py` or `bigclaw-go/scripts/migration/export_live_shadow_bundle.py`.
- Checked-in docs, JSON artifacts, and regression tests reference the Go-owned workflow.
- Targeted tests covering the migrated live-shadow reporting/bundle surfaces pass.

## Validation
- `rg --files -g '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'TestLiveShadowBundleSurface|TestProductionCorpusSurface'`
- `cd bigclaw-go && go test ./...`
- Regenerate the live-shadow scorecard/bundle artifacts with the new Go command(s) and confirm the updated checked-in files match the expected shape.

## Results
- Baseline repository Python inventory: `138` via `rg --files -g '*.py' | wc -l`
- Final repository Python inventory: `132` via `rg --files -g '*.py' | wc -l`
- Replaced the live-shadow scorecard/bundle helpers and the validation-bundle continuation scorecard helper with Go-owned `bigclawctl` commands, migrated helper-specific coverage into `bigclaw-go/internal/liveshadow` and `bigclaw-go/internal/validationbundle`, and removed the retired Python-only helper files/tests from the workspace inventory.
- Regenerated the checked-in live-shadow scorecard and bundle artifacts with fixed historical `--generated-at` timestamps so the migration changes ownership/tooling without drifting the underlying evidence semantics.

## Validation Results
- `cd bigclaw-go && go test ./internal/liveshadow ./cmd/bigclawctl` -> `ok`
- `cd bigclaw-go && go test ./internal/regression -run 'TestBundleFollowUpIndexDocsStayAligned|TestLiveShadowRuntimeDocsStayAligned|TestLiveShadowScorecardBundleStaysAligned|TestLiveShadowBundleSummaryAndIndexStayAligned|TestProductionCorpusMatrixManifestAlignment|TestProductionCorpusDriftRollupStaysAligned'` -> `ok`
- `cd bigclaw-go && go test ./...` -> `ok`
