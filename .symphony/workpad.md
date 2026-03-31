Issue: BIG-GO-1028

Plan
- Retire `tests/test_scheduler.py` and `tests/test_risk.py` because their assertions are already covered by the Go-native scheduler and risk package tests.
- Retire `tests/test_live_shadow_bundle.py` and `tests/test_parallel_validation_bundle.py` because the Go regression suite already validates the canonical live-shadow bundle and live-validation summary/index surfaces directly from repository artifacts.
- Keep the change set scoped to deleting the migrated Python tests plus the required workpad update only.
- Delete the migrated Python test files so this tranche reduces repository `.py` inventory immediately.
- Run targeted formatting, file-count checks, and Go tests; record exact commands and outcomes for final closeout.
- Commit only the scoped issue changes and push the branch to the remote.

Acceptance
- Changes remain scoped to the selected tranche-3 Python test deletions.
- Repository `.py` file count decreases by deleting the four migrated Python test files.
- Repository `.go` file count remains unchanged.
- `pyproject.toml`, `setup.py`, and `setup.cfg` remain unchanged.
- Final report includes the impact on `.py` count, `.go` count, and `pyproject/setup*` files.

Validation
- `find tests -maxdepth 1 -name 'test_*.py' | sort | wc -l`
- `find . -path './.git' -prune -o -name '*.py' -print | sort | wc -l`
- `find . -path './.git' -prune -o -name '*.go' -print | sort | wc -l`
- `cd bigclaw-go && go test ./internal/scheduler -run 'TestSchedulerRoutesHighRiskToKubernetes|TestSchedulerRoutesComputedHighRiskToKubernetes|TestSchedulerRoutesBrowserToKubernetes|TestSchedulerBudgetGuardrail'`
- `cd bigclaw-go && go test ./internal/risk -run 'TestScoreTaskKeepsSimpleLowRiskWorkLow|TestScoreTaskElevatesProdBrowserWork|TestScoreTaskUsesFailuresRetriesAndRegressions|TestScoreTaskFlagsNegativeBudgetForManualReview'`
- `cd bigclaw-go && go test ./internal/regression -run 'TestLiveShadowScorecardBundleStaysAligned|TestLiveShadowBundleSummaryAndIndexStayAligned|TestLiveValidationSummaryStaysAligned|TestLiveValidationIndexStaysAligned'`
- `git diff --stat`
- `git status --short`
