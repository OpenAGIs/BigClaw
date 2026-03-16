# Event Bus Reliability Report

## Scope

This report summarizes the current event-bus reliability evidence and the replicated-durability rollout posture for `OPE-183` / `BIG-GO-008` plus the follow-on distributed slices through `OPE-246`.

## Implemented Surfaces

- In-process publish/subscribe bus with replay history
- Recorder sink integration for audit/debug persistence
- Webhook sink integration for external fanout
- SSE stream via `GET /stream/events`
- Optional SSE replay and filtering via `replay=1`, `after_id`, `Last-Event-ID`, `task_id`, and `trace_id`
- Replay cursor diagnostics via `X-Replay-*` headers and JSON `cursor` metadata on `GET /events`
- Retention watermark / replay horizon visibility through API debug payloads and event-log service surfaces
- Expired checkpoint diagnostics, checkpoint reset surface, and persisted operator history through `GET/DELETE /stream/events/checkpoints/{subscriber_id}` plus `GET /stream/events/checkpoints/{subscriber_id}/history` and conflict payloads on resume attempts
- SQLite retention bootstrap with persisted truncation boundaries that survive process restarts when a replay window is configured
- Replay-safe consumer delivery metadata via `EventDelivery`, including additive `delivery.mode`, `delivery.replay`, and `delivery.idempotency_key` fields
- Consumer dedup ledger/result contract covering duplicate, retryable-failure, and already-applied outcomes
- Replay-safe consumer dedup ledger contract with stable storage key and result semantics
- SQLite-backed durable consumer dedup ledger bootstrap for replay-safe side-effect persistence across restarts
- Subscriber-group checkpoint lease coordination via `/subscriber-groups/leases` and `/subscriber-groups/checkpoints`
- Event backend capability and config-validation contract via `internal/events/backend_contract.go`
- Event-log backend capability probe surfaced through control/debug responses before replay-oriented dispatch

## Current Evidence Pack

- Automated coverage:
  - `internal/events/bus_test.go`
  - `internal/events/delivery_test.go`
  - `internal/events/dedup_ledger_test.go`
  - `internal/events/subscriber_leases_test.go`
  - `internal/events/backend_contract_test.go`
  - `internal/api/server_test.go`
- Supporting closeout reports:
  - `docs/reports/live-validation-index.md`
  - `docs/reports/replay-retention-semantics-report.md`
  - `docs/reports/multi-subscriber-takeover-validation-report.md`
  - `docs/reports/replicated-event-log-durability-rollout-contract.md`
  - `docs/reports/broker-failover-fault-injection-validation-pack.md`

## Validated Behaviors

- Published events are retained in replay history.
- New subscribers can request replayed events before switching to live events.
- Webhook sink receives serialized domain events.
- SSE streaming can deliver live events.
- SSE replay can filter to one trace without leaking unrelated events.
- Replay and live deliveries preserve the original event id while exposing an explicit delivery mode and stable downstream idempotency key.
- Subscriber-group checkpoint commits are fenced by lease token plus epoch, so stale writers cannot advance ownership after takeover.
- Checkpoint offsets remain monotonic within a subscriber group and reject rollback writes.
- Operators can inspect backend capability support before dispatching replay-oriented operations.
- Operator-facing capability payloads distinguish durable consumer dedup support from process-local replay/checkpoint support.

## Rollout Posture

- Local event distribution is implemented and test-backed today.
- Durable replay semantics, checkpoint fencing, and retention boundaries now have repo-native contracts and validation references.
- `docs/reports/live-validation-index.md` normalizes the latest local, Kubernetes, and Ray validation bundle so event-bus review can point to one stable runtime evidence surface.
- `docs/reports/multi-subscriber-takeover-validation-report.md` defines the next takeover-focused reliability matrix, but that plan is still ahead of a durable shared lease backend.

## Remaining Gaps

- No concrete broker-backed external event log exists yet in this checkout; replay still depends on in-process or local durable history plus the documented adapter plan.
- Only the SQLite durable consumer dedup backend exists today; HTTP and broker-backed dedup persistence are still future work.
- Lease coordination is still single-process and not yet backed by a shared durable store.
- No partitioned topic model or broker-backed cross-process subscriber coordination exists yet.
- Multi-subscriber takeover fault injection remains a planned validation pack rather than an implemented executable suite.

## Artifacts

- `docs/reports/live-validation-index.md`
- `docs/reports/live-validation-index.json`
- `docs/reports/replay-retention-semantics-report.md`
- `docs/reports/multi-subscriber-takeover-validation-report.md`
- `docs/reports/replicated-event-log-durability-rollout-contract.md`
- `docs/reports/broker-failover-fault-injection-validation-pack.md`
- `docs/reports/event-bus-reliability-report.md`
