# BIG-GO-1026 Workpad

## Plan
- Remove Python tests whose surfaces now have direct Go coverage in `bigclaw-go/internal/*`, starting with bundle/report fixtures and queue control-center coverage.
- Update in-repo references that still point to removed Python test files so validation guidance stays accurate.
- Run targeted Go tests for the replacement coverage and repo-level grep/count checks, then record exact commands and results.
- Commit the scoped changes and push the branch to the remote.

## Acceptance
- Scope stays limited to the remaining Python test tranche for this issue.
- `.py` file count decreases from the current baseline.
- Go coverage exists for each removed Python test surface.
- Any references to removed Python tests are updated or eliminated.
- Report includes `.py` / `.go` file-count impact and confirms whether `pyproject.toml` / `setup.py` / `setup.cfg` changed.

## Validation
- `go test ./internal/regression -run 'TestLiveShadowScorecardBundleStaysAligned|TestLiveShadowBundleSummaryAndIndexStayAligned|TestLiveValidationSummaryStaysAligned|TestLiveValidationIndexStaysAligned|TestSharedQueueCompanionSummaryStaysAligned|TestLane8ValidationBundleContinuationScorecardStaysAligned|TestLane8LiveShadowScorecardStaysAligned|TestLane8ShadowMatrixCorpusCoverageStaysAligned'`
- `go test ./internal/queue ./internal/reporting`
- `rg -n "test_live_shadow_bundle\\.py|test_parallel_validation_bundle\\.py|test_control_center\\.py" .`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `find . -name pyproject.toml -o -name setup.py -o -name setup.cfg`
