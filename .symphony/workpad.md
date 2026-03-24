# BIGCLAW-181

Title: BIG-vNext-011 worker池自适应分流与容量保护回归

## Plan
- Inspect worker pool routing and capacity protection paths in the Go control plane.
- Implement a scoped fix for the worker-pool adaptive dispatch regression so pool ticks do not oversubscribe scheduler quota.
- Add targeted regression coverage for concurrent-limit and preemptible-capacity behavior.
- Run focused tests, record exact commands and results, then commit and push.

## Acceptance
- Worker pool dispatch honors remaining scheduler capacity instead of fanning out all workers against the same quota snapshot.
- Preemptible capacity still allows limited urgent overflow without allowing the whole pool to bypass protection.
- Targeted regression tests covering the worker pool pass.

## Validation
- Inspect `bigclaw-go/internal/worker/pool.go` and adjacent runtime/scheduler code paths.
- Run focused `go test` commands for `bigclaw-go/internal/worker`.
- Record exact commands and outcomes in this file after validation.

## Results
- Implemented pool-level quota reservation in `bigclaw-go/internal/worker/pool.go` so each worker tick receives an adjusted capacity snapshot instead of the original shared snapshot.
- Added worker-pool regression coverage for remaining concurrency protection and capped preemptible overflow dispatch in `bigclaw-go/internal/worker/pool_test.go`.
- Validation:
- `cd /Users/openagi/code/bigclaw-workspaces/BIGCLAW-181/bigclaw-go && go test ./internal/worker` -> `ok  	bigclaw-go/internal/worker	1.187s`
- `cd /Users/openagi/code/bigclaw-workspaces/BIGCLAW-181/bigclaw-go && go test ./internal/worker -run 'TestPool'` -> `ok  	bigclaw-go/internal/worker	4.924s`
