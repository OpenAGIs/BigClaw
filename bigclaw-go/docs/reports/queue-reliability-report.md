# Queue Reliability Report

## Scope

This report summarizes the current queue reliability evidence for the Go control plane across memory, file, and SQLite backends, plus the latest shared-queue coordination proof used for distributed closeout.

## Automated Evidence

- `internal/queue/memory_queue_test.go`
- `internal/queue/file_queue_test.go`
- `internal/queue/sqlite_queue_test.go`
- `internal/api/server_test.go`
- `docs/reports/lease-recovery-report.md`
- `docs/reports/multi-node-coordination-report.md`
- `docs/reports/live-validation-summary.json`

## Verified Behaviors

- Priority ordering works for the in-memory queue.
- File-backed queue persists tasks across reload and preserves dead-letter replay behavior across reload.
- SQLite-backed queue persists tasks across reopen, supports dead-letter listing, and supports replay back into the runnable queue.
- SQLite queue coverage includes a `1k` task no-duplicate-consumption test plus lease-expiry reacquisition coverage.
- API-level dead-letter listing and replay are available through `GET /deadletters` and `POST /deadletters/{id}/replay`.
- Two `bigclawd` processes can coordinate against one SQLite queue with `0` duplicate terminal executions in the latest shared-queue report.
- SQLite local reliability has been hardened by constraining the local connection pool to a single open and idle connection, removing the `database is locked` failure from concurrent queue validation.

## Current Result

- Queue implementations now support dead-letter retrieval and replay instead of only marking terminal failure.
- Lease recovery and replay paths are directly testable and inspectable through the API.
- Shared-queue coordination evidence now complements the single-process reliability tests, so the closeout pack no longer relies on one-node queue proofs only.
- The queue layer is review-ready for the implemented local and shared-SQLite topology.

## Remaining Gaps

- A larger `10k` reliability matrix is still a reasonable next follow-up if stricter closure criteria are desired.
- Queue coordination is proven for shared SQLite, not for a broker-backed or quorum-backed distributed queue.
- Local SQLite evidence does not replace leader election, durable lease fencing across independent stores, or multi-region queue validation.

## Artifacts

- `docs/reports/lease-recovery-report.md`
- `docs/reports/multi-node-coordination-report.md`
- `docs/reports/multi-node-shared-queue-report.json`
- `docs/reports/live-validation-summary.json`
