# BIG-GO-948 Validation

## Lane File List

- `tests/test_cross_process_coordination_surface.py`
- `tests/test_followup_digests.py`
- `tests/test_live_shadow_scorecard.py`
- `tests/test_shadow_matrix_corpus.py`
- `tests/test_subscriber_takeover_harness.py`
- `tests/test_validation_bundle_continuation_scorecard.py`
- `tests/test_parallel_refill.py`
- `tests/test_roadmap.py`

## Go Replacements

- `bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go`
  - `TestLane8CrossProcessCoordinationSurfaceStaysAligned`
  - `TestLane8ValidationBundleContinuationScorecardStaysAligned`
  - `TestLane8LiveShadowScorecardStaysAligned`
  - `TestLane8ShadowMatrixCorpusCoverageStaysAligned`
  - `TestLane8SubscriberTakeoverHarnessStaysAligned`
  - `TestLane8FollowupDigestsStayAligned`
- `bigclaw-go/internal/refill/queue_repo_fixture_test.go`
  - `TestParallelIssueQueueRepoFixtureSelectionStaysAligned`
- `bigclaw-go/internal/regression/roadmap_contract_test.go`
  - `TestExecutionPackRoadmapDocsStayAligned`
  - `TestExecutionPackRoadmapUniqueOwnersContract`

The deleted Python tests were either:
- report and digest regressions over checked-in `bigclaw-go/docs/reports/*` artifacts, now covered in Go under `bigclaw-go/internal/regression`
- refill queue fixture assertions over `docs/parallel-refill-queue.json`, now covered in Go under `bigclaw-go/internal/refill`

This lane removes redundant Python-only coverage without expanding into unrelated runtime migration domains.

## Validation Commands

- `cd bigclaw-go && go test ./internal/regression -run 'TestLane8|TestCrossProcessCoordinationReadinessDocsStayAligned|TestLiveShadowScorecardBundleStaysAligned|TestProductionCorpus|TestLocalTakeoverReportStaysAligned|TestLiveValidationIndexStaysAligned|TestLiveValidationSummaryStaysAligned|TestFollowUpLaneDocsStayAligned'`
- `cd bigclaw-go && go test ./internal/refill -run TestParallelIssueQueueRepoFixtureSelectionStaysAligned`
- `cd bigclaw-go && go test ./internal/regression -run 'TestExecutionPackRoadmapDocsStayAligned|TestExecutionPackRoadmapUniqueOwnersContract'`
- `git status --short`

## Residual Risks

- This lane intentionally leaves other remaining `tests/*.py` files untouched when they do not yet have a tight Go regression home or require broader production code migration.
- `tests/test_parallel_validation_bundle.py` and other script-execution Python tests remain outside this scoped delete set because they exercise dynamic script behavior rather than only checked-in report fixtures.
- `tests/test_cost_control.py`, `tests/test_service.py`, `tests/test_deprecation.py`, `tests/test_control_center.py`, `tests/test_operations.py`, and `tests/test_ui_review.py` still need broader Go-native implementation or contract surfaces before their Python tests can be removed safely.
