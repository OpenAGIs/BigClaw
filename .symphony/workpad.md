Issue: BIG-GO-1027

Plan
- Finish a narrow residual tranche by removing the stale script-bound `tests/test_parallel_validation_bundle.py`.
- Rely on existing Go-native live-validation summary/index/shared-queue/continuation-gate regressions instead of the removed Python exporter path.
- Delete `tests/test_parallel_validation_bundle.py` once the checked-in live-validation bundle surfaces are validated in Go.
- Run targeted Go validation plus file-count/package-surface checks, then commit and push the branch.

Acceptance
- Changes remain scoped to the stale live-validation residual Python test tranche and corresponding Go regression/doc coverage.
- Repository `.py` file count decreases after removing `tests/test_parallel_validation_bundle.py`.
- Targeted Go tests cover the checked-in live-validation summary, index, shared-queue companion, and continuation-gate surfaces that replaced the Python exporter path.
- Final report includes the exact impact on `.py` count, `.go` count, and confirms whether `pyproject.toml` / `setup.py` changed.

Validation
- `find tests -maxdepth 1 -name '*.py' | sort`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'Test(LiveValidationSummaryStaysAligned|LiveValidationIndexStaysAligned|LiveValidationIndexSummaryPointers|SharedQueueCompanionSummaryStaysAligned)' -count=1`
- `cd bigclaw-go && go test ./internal/api -run 'TestDebugStatusIncludesValidationBundleContinuationGate' -count=1`
- `git diff --stat`
- `git status --short`
