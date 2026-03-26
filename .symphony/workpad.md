# BIGCLAW-196 Workpad

## Plan

1. Confirm the active Go surfaces for node heartbeat degradation, worker-pool health, and queue task reassignment.
2. Extend the worker-pool/control-plane API payloads so degraded nodes expose the tasks still leased on them and the tasks already requeued from them, closing the observability loop without changing scheduler policy.
3. Add focused Go tests for node-aware worker-pool summaries and health payloads that prove degraded-node task reassignment is visible.
4. Run targeted Go tests, capture exact commands and results, then commit and push the issue branch.

## Acceptance

- The active Go mainline reports degraded node state in worker-pool health surfaces.
- The same API payloads show the tasks affected by degraded-node reassignment, including still-leased work and requeued work, so operators can verify the closed loop.
- Existing reassignment behavior remains scoped to degraded nodes and no unrelated scheduler behavior changes.
- Targeted Go tests cover the degraded-node reassignment visibility path.
- The branch contains a committed, pushed implementation for BIGCLAW-196.

## Validation

- Run focused Go tests for `bigclaw-go/internal/api` and `bigclaw-go/internal/worker`.
- Record the exact commands and pass/fail results.
- Verify the final commit SHA is pushed and matches the remote branch SHA.
