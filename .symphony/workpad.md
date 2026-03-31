Issue: BIG-GO-1027

Plan
- Finish a narrow residual tranche by replacing `tests/test_risk.py` with Go-native risk and scheduler-assessment coverage.
- Reuse the existing `internal/risk` score tests for the `0` and `40` expectations, and add one focused `internal/scheduler` regression for the `70/high/requires approval` rejected-assessment case.
- Delete `tests/test_risk.py` once the equivalent Go-native scoring and approval-routing assertions are covered.
- Run targeted Go validation plus file-count/package-surface checks, then commit and push the branch.

Acceptance
- Changes remain scoped to the risk residual Python test tranche and corresponding Go risk/scheduler coverage.
- Repository `.py` file count decreases after removing `tests/test_risk.py`.
- Targeted Go tests cover low-risk baseline scoring, medium prod-browser scoring, and the high-risk approval-required scheduler assessment path.
- Final report includes the exact impact on `.py` count, `.go` count, and confirms whether `pyproject.toml` / `setup.py` changed.

Validation
- `find tests -maxdepth 1 -name '*.py' | sort`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `cd bigclaw-go && go test ./internal/risk -run 'Test(ScoreTaskKeepsSimpleLowRiskWorkLow|ScoreTaskElevatesProdBrowserWork)' -count=1`
- `cd bigclaw-go && go test ./internal/scheduler -run 'TestSchedulerAssessmentComputedHighRiskRequiresApprovalAndSecurityHandoff' -count=1`
- `git diff --stat`
- `git status --short`
