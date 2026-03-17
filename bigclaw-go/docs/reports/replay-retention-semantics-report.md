# Replay Retention and Checkpoint Compaction Semantics

## Scope

This report defines the retention and compaction contract for the replay-capable event surfaces being extended under `OPE-212` / `BIG-PAR-026`.

The current Go runtime still uses in-process replay history in `internal/events/bus.go` and recorder-backed timeline lookup in `internal/api/server.go`. This document defines the durability-safe behavior that a future retained event-log backend must preserve when replay history and subscriber checkpoints age out.

## Current baseline

- `Bus.SubscribeReplay` replays the tail of the in-memory append history before switching the subscriber to live events.
- `GET /events`, `GET /replay/{id}`, and `GET /stream/events?replay=1` expose replay-oriented views over recorder history.
- No durable retention watermark, checkpoint expiration signal, or history compaction rule exists yet.

## Retention model

- Replay history is retained as a contiguous suffix of the event log.
- Compaction may advance the oldest retained boundary, but it must never create holes inside the retained replay window.
- A backend must expose or be able to derive an oldest retained cursor and newest retained cursor for operator diagnostics.
- Cursor validity is determined against the retained window, not only by syntactic shape. A well-formed cursor may still be expired if it points before the oldest retained boundary.

## Compaction boundaries

- History may only be compacted on prefix boundaries where every removed event is older than the new oldest retained cursor.
- Compaction must not discard an event while still claiming that the event's cursor can be resumed.
- Checkpoint cleanup and history compaction are related but distinct operations: compacting history advances replay availability, while checkpoint cleanup removes stale subscriber metadata.
- Backends should prefer monotonic compaction watermarks so repeated resume attempts observe stable or forward-only retention behavior.

## Checkpoint interaction

- A subscriber checkpoint represents the last fully processed event cursor for that subscriber identity or lease domain.
- A checkpoint is resumable only when its cursor is still within the retained replay window.
- If a checkpoint points before the oldest retained cursor, the backend must reject resume as expired instead of silently starting from an arbitrary later event.
- If a checkpoint points inside the retained window but the exact event has been compacted incorrectly, that is a backend correctness bug rather than a fallback case.
- Active subscriber checkpoints should be retained even when the subscriber is idle; cleanup eligibility should be driven by inactivity policy, lease expiry, or explicit deletion, not by compaction alone.

## Expired cursor fallback contract

- Resume requests against aged-out checkpoints must surface an explicit expired-cursor result.
- The current API surface now returns checkpoint diagnostics plus a reset path through `GET/DELETE /stream/events/checkpoints/{subscriber_id}` and a persisted review trail through `GET /stream/events/checkpoints/{subscriber_id}/history` when a saved cursor falls behind the retained boundary.
- The result should include the subscriber or lease identity, the requested checkpoint cursor, and the oldest/newest retained cursors that were available at evaluation time.
- Operator-facing diagnostics should describe whether recovery can restart from the earliest retained event, from the latest live edge, or requires manual checkpoint reset.
- Automatic fallback must be policy-driven. The default safe behavior is fail-closed with diagnostics rather than silently skipping truncated history.

## Cleanup path

- Retention-aware checkpoint cleanup should only remove checkpoints that are both inactive and no longer useful for replay recovery.
- Cleanup should record enough metadata to explain why a checkpoint disappeared, including inactivity age, lease status, and retention watermark at deletion time.
- A future durable backend should separate:
  - history retention policy,
  - checkpoint inactivity TTL,
  - operator override/reset actions.
- This separation lets operators keep short replay windows without forcing immediate loss of subscriber ownership metadata.

## Required operator signals

- Current oldest retained cursor
- Current newest retained cursor
- Subscriber checkpoint cursor and last update time
- Resume failure reason: expired cursor, unknown subscriber, or backend mismatch
- Suggested recovery action: restart from earliest retained, reset checkpoint, or investigate backend corruption

## Forward path

- `OPE-212` establishes the compaction and retention contract.
- `OPE-216` established the expired replay cursor semantics, `OPE-226` added the concrete checkpoint diagnostics / reset surface for durable checkpoint resumes, and `OPE-228` extends that flow with persisted reset audit history.
- `docs/reports/broker-retention-boundary-proof-summary.json` now captures the deterministic broker-stub scenario where a stale checkpoint falls behind the retention floor and must be reset explicitly.
- Durable backends extending `internal/events` should expose retention watermarks before replay-aware checkpoint cleanup is implemented.
- SQLite-backed durable logs now persist trimmed replay boundaries across restarts when a retention window is configured, giving operators a stable replay horizon even after reboot.
- The remote HTTP event-log validation lane in `docs/reports/external-store-validation-report.json` now proves that the same persisted retention-boundary metadata stays visible through a repo-native external-store service boundary, not only when the log is embedded locally.
- That report's backend matrix keeps the lane split explicit: `http_remote_service` is the checked-in proof, while `broker_replicated` remains `not_configured` and `quorum_replicated` remains `contract_only`.

## Repo evidence

- `internal/events/bus.go`
- `internal/events/bus_test.go`
- `internal/api/server.go`
- `internal/api/server_test.go`
- `docs/reports/event-bus-reliability-report.md`
- `docs/reports/broker-retention-boundary-proof-summary.json`
