# BIGCLAW-190 Workpad

## Context
- Issue: `BIGCLAW-190`
- Focus: regressions around multi-tenant parallel fairness and queue starvation
- Constraint: keep changes scoped to this issue
- Environment issue encountered: workspace arrived empty, and initial remote retrieval attempts have been unstable

## Plan
1. Rebuild the local git metadata for this workspace and fetch a minimal `main` baseline.
2. Locate scheduler / queue / tenant fairness code paths and existing tests covering parallel execution.
3. Reproduce the regression with targeted tests or by adding a focused failing test.
4. Implement the smallest fix that restores fairness and prevents starvation without broad refactors.
5. Run targeted validation, capture exact commands and results, then commit and push.

## Acceptance Criteria
- Multi-tenant queued work no longer allows one tenant to starve others under parallel execution.
- Fair scheduling behavior is covered by targeted automated tests.
- Existing adjacent behavior remains intact for the touched scheduler / queue code paths.
- All changes remain scoped to code and tests directly relevant to `BIGCLAW-190`.

## Validation Plan
- Inspect existing queue / scheduler tests and run the most relevant subset first.
- Add or update focused tests for fairness / starvation scenarios.
- Run the smallest command set that proves the fix, and record exact commands plus pass/fail results here after execution.

## Validation Results
- `gofmt -w bigclaw-go/internal/queue/memory_queue.go bigclaw-go/internal/queue/sqlite_queue.go bigclaw-go/internal/queue/memory_queue_test.go bigclaw-go/internal/queue/sqlite_queue_test.go bigclaw-go/internal/worker/pool_test.go`
  - Result: passed
- `cd bigclaw-go && go test ./internal/queue ./internal/worker`
  - Result: passed
  - Notes: `ok  	bigclaw-go/internal/queue	111.800s`, `ok  	bigclaw-go/internal/worker	(cached)`
