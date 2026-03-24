# BIGCLAW-172

## Plan
- Inspect shared queue lease paths across `MemoryQueue`, `FileQueue`, and `SQLiteQueue`.
- Tighten lease acquire/renew/release state transitions so expired or stale leases cannot mutate task ownership.
- Add regression tests for local backends and shared SQLite clients covering duplicate consumption, expiry takeover, and stale writer rejection.
- Run targeted queue tests, record exact commands and results, then commit and push the scoped change set.

## Acceptance Mapping
- 明确 lease 获取/续约/释放状态机:
  centralize and document lease mutation validation in queue internals, and make backends enforce the same stale/expired ownership rules.
- 重复消费与网络抖动下不出现双执行:
  reject stale or expired ack/requeue/dead-letter mutations after takeover or renewal races, including shared SQLite clients.
- 增加回归测试覆盖 local + distributed 场景:
  extend file/local queue tests and shared SQLite cross-client tests for expiry, takeover, and duplicate-consumption safety.

## Validation
- `cd /Users/openagi/code/bigclaw-workspaces/BIGCLAW-172/bigclaw-go && go test ./internal/queue`
- If needed, run focused reruns for failing queue tests while iterating.
