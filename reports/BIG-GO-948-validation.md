# BIG-GO-948 Validation

## Lane File List

- `tests/test_cross_process_coordination_surface.py`
- `tests/test_followup_digests.py`
- `tests/test_live_shadow_scorecard.py`
- `tests/test_shadow_matrix_corpus.py`
- `tests/test_subscriber_takeover_harness.py`
- `tests/test_validation_bundle_continuation_scorecard.py`

## Go Replacements

- `bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go`
  - `TestLane8CrossProcessCoordinationSurfaceStaysAligned`
  - `TestLane8ValidationBundleContinuationScorecardStaysAligned`
  - `TestLane8LiveShadowScorecardStaysAligned`
  - `TestLane8ShadowMatrixCorpusCoverageStaysAligned`
  - `TestLane8SubscriberTakeoverHarnessStaysAligned`
  - `TestLane8FollowupDigestsStayAligned`

The deleted Python tests were report and digest regressions over checked-in `bigclaw-go/docs/reports/*` artifacts. Their assertions now live in Go regression tests under `bigclaw-go/internal/regression`, so this lane removes redundant Python-only coverage without expanding into unrelated test domains.

## Validation Commands

- `cd bigclaw-go && go test ./internal/regression -run 'TestLane8|TestCrossProcessCoordinationReadinessDocsStayAligned|TestLiveShadowScorecardBundleStaysAligned|TestProductionCorpus|TestLocalTakeoverReportStaysAligned|TestLiveValidationIndexStaysAligned|TestLiveValidationSummaryStaysAligned|TestFollowUpLaneDocsStayAligned'`
- `git status --short`

## Residual Risks

- This lane intentionally leaves other remaining `tests/*.py` files untouched when they do not yet have a tight Go regression home or require broader production code migration.
- `tests/test_parallel_validation_bundle.py` and other script-execution Python tests remain outside this scoped delete set because they exercise dynamic script behavior rather than only checked-in report fixtures.
