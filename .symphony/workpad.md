Issue: BIG-GO-1027

Plan
- Finish a narrow residual tranche by removing the stale script-bound `tests/test_live_shadow_bundle.py`.
- Rely on existing Go-native live-shadow bundle/index regressions and migration docs that now point to `bigclawctl automation migration export-live-shadow-bundle` instead of the removed Python exporter path.
- Delete `tests/test_live_shadow_bundle.py` once the checked-in live-shadow bundle/index surface coverage is validated in Go.
- Run targeted Go validation plus file-count/package-surface checks, then commit and push the branch.

Acceptance
- Changes remain scoped to the stale live-shadow residual Python test tranche and corresponding Go regression/doc coverage.
- Repository `.py` file count decreases after removing `tests/test_live_shadow_bundle.py`.
- Targeted Go tests cover the checked-in live-shadow scorecard, summary, index, and drift-rollup surfaces that replaced the Python exporter path.
- Final report includes the exact impact on `.py` count, `.go` count, and confirms whether `pyproject.toml` / `setup.py` changed.

Validation
- `find tests -maxdepth 1 -name '*.py' | sort`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'Test(Lane8LiveShadowScorecardStaysAligned|LiveShadowBundleSurfaceStaysAligned)' -count=1`
- `git diff --stat`
- `git status --short`
