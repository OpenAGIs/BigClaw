Issue: BIG-GO-1027

Plan
- Finish a narrow residual tranche by replacing `tests/test_control_center.py` with Go-native queue/control-center coverage.
- Extend `bigclaw-go/internal/reporting` only for the missing shared-view empty-state rendering that the Python test still pins.
- Delete `tests/test_control_center.py` once queue ordering, control-center report content, and empty shared-view state are all covered in Go.
- Run targeted Go validation plus file-count/package-surface checks, then commit and push the branch.

Acceptance
- Changes remain scoped to the control-center residual Python test tranche and directly corresponding Go reporting/queue coverage.
- Repository `.py` file count decreases after removing `tests/test_control_center.py`.
- Targeted Go tests cover queue priority ordering, queue control center report content, and shared-view empty-state rendering.
- Final report includes the exact impact on `.py` count, `.go` count, and confirms whether `pyproject.toml` / `setup.py` changed.

Validation
- `find tests -maxdepth 1 -name '*.py' | sort`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `gofmt -w bigclaw-go/internal/reporting/reporting.go bigclaw-go/internal/reporting/reporting_test.go`
- `cd bigclaw-go && go test ./internal/reporting ./internal/queue -run 'Test(BuildQueueControlCenter|MemoryQueueLeasesByPriority)' -count=1`
- `git diff --stat`
- `git status --short`
