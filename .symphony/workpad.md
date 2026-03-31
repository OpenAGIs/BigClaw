Issue: BIG-GO-1027

Plan
- Finish a narrow residual tranche by replacing `tests/test_scheduler.py` with Go-native legacy medium-decision coverage.
- Add a small compatibility helper in `internal/scheduler` that preserves the Python scheduler’s `medium`/budget-floor decision contract without changing the main executor-routing scheduler path.
- Delete `tests/test_scheduler.py` once equivalent Go-native high-risk approval, browser routing, browser-to-docker budget degradation, and pause-on-low-budget behaviors are covered.
- Run targeted Go validation plus file-count/package-surface checks, then commit and push the branch.

Acceptance
- Changes remain scoped to the scheduler residual Python test tranche and corresponding Go scheduler compatibility coverage.
- Repository `.py` file count decreases after removing `tests/test_scheduler.py`.
- Targeted Go tests cover high-risk approval gating, browser routing, budget degradation from browser to docker, and pause-on-low-budget behavior for the legacy medium contract.
- Final report includes the exact impact on `.py` count, `.go` count, and confirms whether `pyproject.toml` / `setup.py` changed.

Validation
- `find tests -maxdepth 1 -name '*.py' | sort`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `gofmt -w bigclaw-go/internal/scheduler/legacy_medium.go bigclaw-go/internal/scheduler/legacy_medium_test.go`
- `cd bigclaw-go && go test ./internal/scheduler -run 'TestLegacyMediumDecision' -count=1`
- `git diff --stat`
- `git status --short`
