# Queue Reliability Report

## Scope

This report summarizes the current reliability evidence for the Go queue layer across memory, file, and SQLite backends.

## Automated Evidence

- `internal/queue/memory_queue_test.go`
- `internal/queue/file_queue_test.go`
- `internal/queue/sqlite_queue_test.go`
- `internal/api/server_test.go`
- `docs/reports/live-validation-summary.json`

## Verified Behaviors

- Priority ordering works for the in-memory queue.
- File-backed queue persists tasks across reload and persists dead-letter replay behavior across reload.
- SQLite-backed queue persists tasks across reopen, supports dead-letter listing, supports replay back into the runnable queue, and now passes a `1k` task no-duplicate-consumption test.
- API-level dead-letter listing and replay are available through `GET /deadletters` and `POST /deadletters/{id}/replay`.
- SQLite local reliability has been hardened by constraining the local connection pool to a single open/idle connection, removing the `database is locked` failure that appeared during concurrent queue validation.

## Current Result

- Queue implementations now support dead-letter retrieval and replay instead of only marking terminal failure.
- Lease recovery and replay paths are directly testable and inspectable through the API.
- The queue layer is materially closer to the original reliability target and is ready for another review pass.
- A larger `10k` reliability matrix is still a reasonable next follow-up if stricter closure criteria are desired.

## Follow-Up Digest

- `docs/reports/scale-validation-follow-up-digest.md` consolidates the remaining queue scale follow-up with the benchmark and longer-duration soak caveats.
