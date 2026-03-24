# BIGCLAW-189

## Plan
- inspect the current worker runtime, pool, queue lease lifecycle, and worker-pool diagnostics surfaces
- add a worker health probe in the pool that runs once per orchestration cycle and records stale heartbeat / recovery telemetry
- add a queue self-heal hook that proactively requeues expired leased tasks so orphaned work is redispatched without waiting for new lease traffic
- expose the probe and remediation counters through worker snapshots and API worker-pool health payloads
- cover the new behavior with targeted queue, worker, and API tests
- run targeted tests, record exact commands and results, then commit and push `BIGCLAW-189`

## Acceptance
- worker pool executes a health probe during normal loop cycles without changing unrelated worker execution paths
- expired leased tasks are proactively recovered and become available for redispatch through a self-heal path
- worker-pool health output includes probe/remediation telemetry that operators can inspect
- targeted automated tests verify stale worker detection, lease recovery, and API reporting

## Validation
- `cd /Users/openagi/code/bigclaw-workspaces/BIGCLAW-189/bigclaw-go && go test ./internal/queue ./internal/worker ./internal/api`
- if failures indicate narrower scope is better, rerun focused packages or tests and record exact commands plus pass/fail status
