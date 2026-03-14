# Lease Recovery Report

## Scope

This report captures the current automated evidence for lease expiry and recovery behavior in the Go queue layer.

## Automated Evidence

- `internal/queue/sqlite_queue_test.go::TestSQLiteQueueLeaseExpiresAndCanBeReacquired`
- `internal/queue/sqlite_queue_test.go::TestSQLiteQueueProcesses1000TasksWithoutDuplicateLease`
- `internal/worker/runtime.go`
- `internal/worker/runtime_test.go`

## Verified Behaviors

- Worker leases carry explicit owner, attempt, acquisition time, and expiry metadata.
- Worker runtime renews leases on a heartbeat interval while execution is active.
- Expired SQLite leases become available for reacquisition by a different worker.
- Replay and dead-letter operations clear lease ownership and return the task to an actionable state.
- A `1k` task concurrent processing scenario now completes without duplicate consumption.

## Current Result

- Lease recovery is directly covered by automated tests instead of only being implied by implementation.
- The SQLite queue no longer surfaces the previous `database is locked` failure under the added concurrent reliability test.
- The system supports the core recovery loop expected for worker interruption scenarios.
