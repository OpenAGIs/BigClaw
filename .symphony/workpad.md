Issue: BIG-GO-1019

Plan
- Verify the remaining Python-only `multi_node_shared_queue_test.py` coverage is redundant with checked-in Go regression coverage for the shared-queue and live takeover reports.
- Remove the residual Python test asset if the existing Go regression surface already covers the same report summaries and path invariants.
- Keep changes scoped to the residual Python asset plus the tranche workpad update.
- Run targeted regression coverage, capture exact commands and results, then commit and push the scoped change set.

Acceptance
- Changes stay scoped to `bigclaw-go/scripts/**` residual Python assets plus directly coupled tests/docs.
- `.py` file count under `bigclaw-go/scripts/e2e/**` is reduced for this tranche.
- Shared-queue and live takeover proof coverage remains enforced by Go regression tests after the redundant Python test is removed.
- Final report states the impact on `py files`, `go files`, `pyproject.toml`, and `setup.py`.

Validation
- `find bigclaw-go/scripts/e2e -maxdepth 1 \( -name '*.py' -o -name '*.go' -o -name '*.sh' \) | sort`
- `go test ./internal/regression -run 'TestSharedQueueReportStaysAligned|TestLiveMultiNodeSubscriberTakeoverProofReport|TestLiveTakeoverReportStaysAligned'`
- `find bigclaw-go/scripts -name '*.py' | sort | wc -l`
- `git diff --stat && git status --short`
