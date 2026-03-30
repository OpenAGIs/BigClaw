Issue: BIG-GO-1019

Plan
- Port `subscriber_takeover_fault_matrix.py` into a Go-native deterministic harness command under `bigclawctl automation e2e`.
- Preserve the checked-in takeover report schema and scenario semantics while swapping the harness reference from Python to the Go entrypoint.
- Update directly coupled docs, migration inventory, and regression text to reflect the removed Python generator.
- Run targeted validation for the subscriber-takeover tranche, capture exact commands and results, then commit and push the scoped change set.

Acceptance
- Changes stay scoped to `bigclaw-go/scripts/**` residual Python assets plus directly coupled tests/docs.
- `.py` file count under `bigclaw-go/scripts/e2e/**` is reduced for this tranche.
- Subscriber-takeover validation remains invokable through a Go-native CLI path that writes the same canonical report surface.
- Final report states the impact on `py files`, `go files`, `pyproject.toml`, and `setup.py`.

Validation
- `find bigclaw-go/scripts/e2e -maxdepth 1 \( -name '*.py' -o -name '*.go' -o -name '*.sh' \) | sort`
- `gofmt -w bigclaw-go/cmd/bigclawctl/automation_e2e_takeover_matrix_command.go bigclaw-go/cmd/bigclawctl/automation_e2e_takeover_matrix_command_test.go bigclaw-go/cmd/bigclawctl/automation_commands.go`
- `go test ./cmd/bigclawctl -run 'TestAutomationSubscriberTakeoverFaultMatrixBuildsReport'`
- `go run ./cmd/bigclawctl automation e2e subscriber-takeover-fault-matrix --help`
- `go run ./cmd/bigclawctl automation e2e subscriber-takeover-fault-matrix >/tmp/bigclaw-takeover-report.json`
- `go test ./internal/regression -run 'TestLane8SubscriberTakeoverHarnessStaysAligned|TestLane8FollowupDigestsStayAligned'`
- `git diff --stat && git status --short`
