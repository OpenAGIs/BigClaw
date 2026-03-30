Issue: BIG-GO-1019

Plan
- Port `multi_node_shared_queue.py` into a Go-native `bigclawctl automation e2e multi-node-shared-queue` command.
- Preserve the live two-node shared-queue proof plus the companion live takeover report and per-scenario audit artifact exports.
- Update directly coupled docs, generated report references, and migration inventory to replace the final Python entrypoint.
- Run targeted validation for the shared-queue tranche, capture exact commands and results, then commit and push the scoped change set.

Acceptance
- Changes stay scoped to `bigclaw-go/scripts/**` residual Python assets plus directly coupled tests/docs.
- `.py` file count under `bigclaw-go/scripts/e2e/**` is reduced for this tranche.
- Shared-queue validation remains invokable through a Go-native CLI path that writes the same canonical shared-queue and live takeover report surfaces.
- Final report states the impact on `py files`, `go files`, `pyproject.toml`, and `setup.py`.

Validation
- `find bigclaw-go/scripts/e2e -maxdepth 1 \( -name '*.py' -o -name '*.go' -o -name '*.sh' \) | sort`
- `gofmt -w bigclaw-go/cmd/bigclawctl/automation_e2e_multi_node_shared_queue_command.go bigclaw-go/cmd/bigclawctl/automation_e2e_multi_node_shared_queue_command_test.go bigclaw-go/cmd/bigclawctl/automation_commands.go`
- `go test ./cmd/bigclawctl -run 'TestAutomationMultiNodeSharedQueueBuildLiveTakeoverReport|TestAutomationMultiNodeSharedQueueSummarize'`
- `go run ./cmd/bigclawctl automation e2e multi-node-shared-queue --help`
- `go run ./cmd/bigclawctl automation e2e multi-node-shared-queue --report-path /tmp/bigclaw-multi-node-shared-queue-report.json --takeover-report-path /tmp/bigclaw-live-multi-node-subscriber-takeover-report.json --takeover-artifact-dir /tmp/bigclaw-live-multi-node-subscriber-takeover-artifacts`
- `go test ./internal/regression -run 'TestSharedQueueReportStaysAligned|TestLiveTakeoverReportStaysAligned|TestLiveMultiNodeSubscriberTakeoverProofReport'`
- `find bigclaw-go/scripts -name '*.py' | sort | wc -l`
- `git diff --stat && git status --short`
