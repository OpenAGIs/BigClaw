Issue: BIG-GO-1027

Plan
- Finish a narrow residual tranche by replacing `tests/test_event_bus.py` with Go-native event-bus compatibility coverage.
- Add a small `internal/eventbuscompat` package with a minimal run record, JSON ledger, and event bus that preserves the frozen Python transition/comment behavior for pull-request comments, CI completion, and task failure events.
- Delete `tests/test_event_bus.py` once equivalent Go-native transition, summary, subscriber callback, comment, and ledger-persistence behaviors are covered.
- Run targeted Go validation plus file-count/package-surface checks, then commit and push the branch.

Acceptance
- Changes remain scoped to the event-bus residual Python test tranche and corresponding Go compatibility coverage.
- Repository `.py` file count decreases after removing `tests/test_event_bus.py`.
- Targeted Go tests cover PR-comment approval transition, CI completion transition, task-failed transition, subscriber callback behavior, collaboration comment recording, and ledger persistence.
- Final report includes the exact impact on `.py` count, `.go` count, and confirms whether `pyproject.toml` / `setup.py` changed.

Validation
- `find tests -maxdepth 1 -name '*.py' | sort`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `gofmt -w bigclaw-go/internal/eventbuscompat/eventbus.go bigclaw-go/internal/eventbuscompat/eventbus_test.go`
- `cd bigclaw-go && go test ./internal/eventbuscompat -run 'Test(EventBusPRCommentApprovesWaitingRunAndPersistsLedger|EventBusCICompletedMarksRunCompleted|EventBusTaskFailedMarksRunFailed)' -count=1`
- `git diff --stat`
- `git status --short`
