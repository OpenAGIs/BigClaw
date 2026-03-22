# Event Bus Reliability Report

## Scope

This report summarizes the current event bus reliability evidence and the next replicated-durability integration plan for `OPE-183` / `BIG-GO-008` and `OPE-206`.

## Implemented surfaces

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
- Delivery acknowledgement readiness surface via `docs/reports/delivery-ack-readiness-surface.json` and reviewer-facing debug/distributed diagnostics
- Broker stub live fanout isolation evidence pack via `docs/reports/broker-stub-live-fanout-isolation-evidence-pack.json` and reviewer-facing debug/distributed diagnostics

## Validated behaviors

- Published events are retained in replay history.
- New subscribers can request replayed events before switching to live events.
- Webhook sink receives serialized domain events.
- SSE streaming can deliver live events.
- SSE replay can filter to one trace without leaking unrelated events.
- Replay and live deliveries preserve the original event id while exposing an explicit delivery mode and stable downstream idempotency key.
- Subscriber-group checkpoint commits are fenced by lease token + epoch, so stale writers cannot advance ownership after takeover.
- Checkpoint offsets remain monotonic within a subscriber group and reject rollback writes.
- Operators can inspect backend capability support before dispatching replay-oriented operations.
- Operator-facing capability payloads now distinguish durable consumer dedup support from process-local replay/checkpoint support.

## Evidence

- `internal/events/bus.go`
- `internal/events/bus_test.go`
- `internal/events/capabilities.go`
- `internal/events/durability.go`
- `internal/events/delivery.go`
- `internal/events/delivery_test.go`
- `internal/domain/consumer_dedup.go`
- `internal/domain/consumer_dedup_test.go`
- `internal/events/webhook.go`
- `internal/events/webhook_test.go`
- `internal/events/recorder_sink.go`
- `internal/events/subscriber_leases.go`
- `internal/events/subscriber_leases_test.go`
- `internal/events/backend_contract.go`
- `internal/events/backend_contract_test.go`
- `internal/events/dedup_ledger.go`
- `internal/events/dedup_ledger_test.go`
- `internal/events/sqlite_log.go`
- `internal/api/server.go`
- `internal/api/server_test.go`
- `cmd/bigclawd/main.go`
- `internal/config/config.go`
- `docs/reports/delivery-ack-readiness-surface.json`
- `docs/reports/broker-stub-live-fanout-isolation-evidence-pack.json`

## Current durability shape

- Runtime publish/subscribe remains in-process.
- Audit/debug persistence is recorder-backed, with optional JSONL sinking.
- The `events.DurabilityPlan` surface makes the active backend and the next replicated target explicit in bootstrap and `GET /debug/status`, including broker bootstrap readiness when a replicated target is configured.
- Default plan is `memory -> broker_replicated` with replication factor `3`, and env overrides exist for:
  - `BIGCLAW_EVENT_LOG_BACKEND`
  - `BIGCLAW_EVENT_LOG_TARGET_BACKEND`
  - `BIGCLAW_EVENT_LOG_REPLICATION_FACTOR`

## Next backend targets

- `sqlite`: durable single-node append log with monotonic checkpoints but no replica quorum.
- `http`: shared service-backed append log with single-writer ordering and shared subscriber state.
- `broker_replicated`: quorum or partition-backed log with shared replay, replicated durability, and publisher ack requirements.

## Repo-native integration points

- `cmd/bigclawd/main.go`: bootstrap backend selection, capability validation, dedup ledger contract exposure, and future broker client wiring.
- `internal/events/bus.go`: publish path remains the place to insert append/ack behavior ahead of live fanout.
- `internal/api/server.go`: operational reporting for current and target durability mode plus runtime capability probes.
- Subscriber checkpoint persistence, replay endpoints, and dedup ledger surfaces preserve resume and idempotency semantics while moving state out of process-local memory.
- `internal/events/durability.go`: rollout-facing contract for replicated durability phases, failure domains, and required verification evidence.
- `docs/reports/broker-durability-rollout-scorecard.json`: checked-in reviewer-facing rollout scorecard for readiness, blockers, and missing evidence.
- `docs/reports/durability-rollout-scorecard.json`: generic mirror of the same rollout scorecard payload for bootstrap and automation workflows.
- `internal/events/log.go`: provider-neutral `PartitionRoute` and `SubscriberOwnershipContract` fields now reserve the future partitioned-routing and broker-backed ownership contract surface, and current local runtimes reject those requests explicitly at subscribe time instead of silently ignoring them.

## Migration and compatibility constraints

- Preserve append-only replay semantics across backend cutover.
- Keep subscriber checkpoints monotonic during dual-write or backfill.
- Keep `task_id`, `trace_id`, and `event_type` stable so partitioning and replay filters remain compatible.
- Decouple SSE live fanout from broker consumer lag so replay catch-up does not stall live delivery.

## Implementation-ready follow-up plan

1. Add a concrete event-log interface alongside the in-process bus sink contract so append, replay, and checkpoint operations can be backed by SQLite, HTTP, or broker implementations.
2. Introduce a dual-write migration phase from the current publish path into the new event-log backend while keeping recorder/audit output unchanged.
3. Add checkpoint-backed replay endpoints that read from the shared event log instead of recorder-only history.
4. Add a broker-backed implementation with partition-key rules for `trace_id` and explicit publisher ack / durability error handling.
5. Validate cutover with replay, checkpoint monotonicity, SSE handoff, capability-matrix regression coverage, dedup-ledger persistence coverage, backend-capability probe validation, and multi-subscriber takeover fault-injection evidence under shared multi-node conditions.

## Durability capability matrix

| Backend | Implemented in bootstrap | Durable history | Publish | Replay | Checkpoint | Filtering | Required config |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `memory` | yes | no | native | native | unsupported | native | none |
| `sqlite` | no | yes | native | native | native | derived | `BIGCLAW_EVENT_LOG_DSN`, `BIGCLAW_EVENT_CHECKPOINT_DSN`, `BIGCLAW_EVENT_RETENTION` |
| `http` | no | yes | native | native | native | derived | `BIGCLAW_EVENT_LOG_DSN`, `BIGCLAW_EVENT_CHECKPOINT_DSN`, `BIGCLAW_EVENT_RETENTION` |
| `broker` | no | yes | native | native | native | derived | `BIGCLAW_EVENT_LOG_DSN`, `BIGCLAW_EVENT_CHECKPOINT_DSN`, `BIGCLAW_EVENT_RETENTION` |

## Validation contract

- Startup validates `BIGCLAW_EVENT_BACKEND` against the backend catalog before queue/bootstrap wiring begins.
- Durable backends must provide explicit event-log DSN, checkpoint DSN, and positive retention.
- `BIGCLAW_EVENT_REQUIRE_REPLAY`, `BIGCLAW_EVENT_REQUIRE_CHECKPOINT`, and `BIGCLAW_EVENT_REQUIRE_FILTERING` express the runtime features operators expect from the selected backend.
- Unsupported combinations fail fast with field-specific errors instead of silently downgrading runtime behavior.
- Backends declared in the matrix but not yet wired into the bootstrap runtime are rejected explicitly so planning assumptions cannot masquerade as implemented support.

## Consumer dedup ledger contract

- Stable storage keys use `v1/<consumer_id>/<event_id>` so durable backends can share one persistence layout while still rejecting conflicting event metadata for the same consumer/event tuple.
- Collision protection uses a fingerprint over `consumer_id`, `event_id`, `event_type`, `task_id`, `trace_id`, and `run_id`; the same storage key cannot be reused for a different event payload shape.
- Reservation semantics distinguish first-writer `reserved`, repeated in-flight `duplicate`, and terminal `already_applied` outcomes.
- Applied side effects persist `handler`, `applied_at`, `effect_id`, `effect_sequence`, `effect_fingerprint`, `summary`, and stable metadata so duplicate deliveries can return the prior applied result instead of replaying the side effect.
- Once a record reaches `applied`, backends must treat a different result fingerprint as a conflict instead of silently overwriting the prior side effect evidence.
- The first durable bootstrap is SQLite-backed, stores the full normalized dedup record as durable JSON, and indexes state/update timestamps so lifecycle cleanup can evolve without changing caller contracts.
- Control-plane capability payloads expose `dedup` separately so operators can see whether replay-safe consumers are backed by durable dedup state or process memory only.

## Remaining gaps

- No concrete durable external event log exists yet in this checkout; replay still depends on process-local history plus the documented integration plan.
- Only the SQLite durable consumer dedup backend exists yet; HTTP and broker-backed dedup persistence still need concrete implementations.
- Memory bus delivery acknowledgements remain sink-level best effort; explicit commit acknowledgements exist for SQLite, the remote HTTP event-log service, and the local broker stub, while replicated broker acknowledgements remain contract-only.
- Lease coordination now has a shared durable SQLite-backed scaffold for shared multi-node subscriber groups, but broker-backed and replicated ownership semantics still need a durable backend beyond SQLite.
- No runtime partitioned topic model or broker-backed cross-process subscriber coordination exists yet; `PartitionRoute` and `SubscriberOwnershipContract` remain future-facing contracts, and current local subscription runtimes fail closed when callers request them. See `docs/reports/cross-process-coordination-boundary-digest.md`.
- Retention watermarks are now exposed for in-memory and durable event-log backends, SQLite-backed logs persist trimmed replay boundaries across restarts, and expired checkpoint resumes now fail closed with explicit reset guidance; the broader compaction semantics remain documented in `docs/reports/replay-retention-semantics-report.md`.
- Consumers still need their own dedupe store keyed by `delivery.idempotency_key`; this change does not introduce exactly-once execution.
- Multi-subscriber takeover validation now has both a deterministic local harness in `docs/reports/multi-subscriber-takeover-validation-report.md` / `docs/reports/multi-subscriber-takeover-validation-report.json` and a live two-node companion proof in `docs/reports/live-multi-node-subscriber-takeover-report.json`; see `docs/reports/subscriber-takeover-executability-follow-up-digest.md` for the remaining broker-backed and native-audit caveats.
- `docs/reports/broker-stub-live-fanout-isolation-evidence-pack.json` now captures live fanout isolation for the local broker stub: replay catch-up is deliberately slowed while a live-only subscriber still receives the next publish inside the 50ms drill window, proving the live lane stays separate from replay drain in the process-local path.
- `docs/reports/cross-process-coordination-capability-surface.json` now acts as the runtime capability matrix, summarizing which coordination guarantees are `live_proven`, which are `harness_proven`, and which remain `contract_only`.

## Replicated rollout contract

- `docs/reports/replicated-event-log-durability-rollout-contract.md` now captures the minimum rollout gates for a broker-backed or quorum-backed adapter, and `event_durability` now includes broker bootstrap readiness for those targets:
- `docs/reports/replicated-broker-durability-rollout-spike.md` is the ticket-specific spike artifact that separates the current SQLite/process-local proof boundary from future replicated-backend claims and names the next smallest implementation slices.
- `docs/reports/broker-durability-rollout-scorecard.json` mirrors the runtime contract in one checked-in JSON scorecard so reviewers can inspect readiness, blockers, and missing evidence without reconstructing them from prose.
- `docs/reports/durability-rollout-scorecard.json` keeps the same payload under a repo-agnostic filename for queue bootstrap and automation flows.
- `docs/reports/broker-checkpoint-fencing-proof-summary.json` isolates the stub-matrix scenarios that prove replay/checkpoint sequence monotonicity and stale-writer fencing during takeover.
- `docs/reports/broker-retention-boundary-proof-summary.json` isolates the retention-expiry scenario that proves aged-out checkpoints fail closed until an explicit reset is recorded.
  - replicated publish acknowledgements must distinguish committed, rejected, and ambiguous outcomes;
  - replay and checkpoint state must share the same durable sequence domain across failover;
  - retention boundaries must be operator-visible before resumable recovery is claimed;
  - live fanout must remain isolated from broker catch-up lag; `docs/reports/broker-stub-live-fanout-isolation-evidence-pack.json` now captures the repo-native local proof for the broker stub path.
- The same contract is surfaced in `events.DurabilityPlan`; the nested `rollout_scorecard` field and top-level `event_durability_rollout` alias now publish the derived status through `GET /debug/status` and `/metrics`, with matching checked-in artifacts at `docs/reports/broker-durability-rollout-scorecard.json` and `docs/reports/durability-rollout-scorecard.json`.

## Next adapter boundary

- `internal/events/log.go` now defines the provider-neutral event-log and checkpoint contract for future broker-backed adapters.
- `internal/events/log.go` also defines the future-facing `PartitionRoute` and `SubscriberOwnershipContract` surface for partitioned topic routing and broker-backed subscriber ownership while rejecting those requests in current local subscription runtimes.
- `internal/events/memory_log.go` provides the contract-compatible in-memory baseline while BigClaw remains on local fanout.
- Broker-facing runtime knobs are reserved behind `BIGCLAW_EVENT_LOG_*` env vars so a first provider adapter can land without changing publish/replay/checkpoint callers.
- No durable external event log yet; replay is process-local history.
- No delivery acknowledgement protocol beyond sink-level best effort.
- No runtime partitioned topic model or cross-process subscriber coordination yet; `PartitionRoute` and `SubscriberOwnershipContract` still target future backends, and current local subscription runtimes reject them explicitly. See `docs/reports/cross-process-coordination-boundary-digest.md`.
- Multi-subscriber takeover validation now has both a deterministic local harness in `docs/reports/multi-subscriber-takeover-validation-report.md` / `docs/reports/multi-subscriber-takeover-validation-report.json` and a live two-node companion proof in `docs/reports/live-multi-node-subscriber-takeover-report.json`; see `docs/reports/subscriber-takeover-executability-follow-up-digest.md` for the remaining shared-durable and broker-backed ownership caveats.
- `docs/reports/cross-process-coordination-capability-surface.json` now acts as the runtime capability matrix, summarizing which coordination guarantees are `live_proven`, which are `harness_proven`, and which remain `contract_only`.

## Parallel Follow-up Index

- `docs/reports/parallel-follow-up-index.md` is the canonical index for the
  remaining event-bus takeover, coordination-boundary, and rollout follow-up
  tracks.
- Use `docs/reports/parallel-validation-matrix.md` for the checked-in executor
  validation entrypoint when the event-bus review needs lane-specific proof.
