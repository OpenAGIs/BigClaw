# BIGCLAW-172 Workpad

## Plan

1. Verify the current queue and subscriber lease changes already in the worktree against the issue acceptance for shared coordination, stale lease fencing, and release safety.
2. Keep the implementation scoped to `bigclaw-go/internal/events` and `bigclaw-go/internal/queue`, only adjusting behavior required to formalize acquire, renew, release, expiry, and stale-owner handling.
3. Extend focused regression coverage for local memory and SQLite-backed shared-store flows, including expired mutations, takeover fencing, and release-with-checkpoint preservation.
4. Run targeted Go and Python validation commands, record exact commands and pass/fail results, then commit and push the branch.

## Acceptance

- Lease acquire, renew, commit, and release paths use an explicit state model so vacant, active, and expired leases are handled consistently.
- Duplicate consumption, stale renewals, stale releases, and expired ack/requeue/dead-letter attempts do not allow a second executor to win after takeover or expiry.
- Regression coverage includes both local memory and distributed SQLite-backed queue or lease-store paths.

## Validation

- `cd bigclaw-go && go test ./internal/events ./internal/queue ./internal/worker`
- `cd bigclaw-go && python3 -m unittest scripts/e2e/multi_node_shared_queue_test.py`

Record exact command lines and results in the final closeout.
