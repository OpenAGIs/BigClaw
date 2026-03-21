# Broker-Backed Event Log Adapter Contract

## Scope

This note captures the implementation-ready boundary for `OPE-208` / `BIG-PAR-021` and the contract-surface follow-up in `OPE-257` / `BIG-PAR-095`: a provider-agnostic event-log contract that can move BigClaw beyond the current in-process replay history while preserving publish, replay, checkpoint, partition-routing, and subscriber-ownership semantics.

## Contract surface

- `internal/events/log.go` defines the provider-neutral event-log boundary:
  - `EventLog` for publish, replay, and live subscription.
  - `CheckpointStore` for consumer resume state.
  - `Record` and `Position` so replay can be anchored to a portable sequence/offset token instead of an implementation-specific cursor.
  - `Capabilities` so API/bootstrap surfaces can report whether a backend is durable, ordered, checkpoint-aware, and broker-backed.
  - `PartitionRoute` so future broker-backed adapters can describe topic routing with provider-neutral `trace_id`, `task_id`, and `event_type` partition keys.
  - `SubscriberOwnershipContract` so future broker-backed adapters can describe exclusive subscriber-group ownership, lease fencing, and partition affinity without changing caller-facing subscription APIs.
- `internal/events/memory_log.go` is the first implementation of that contract and acts as the compatibility baseline for future broker adapters.

## Contract-only target semantics

- Partitioned topic routing:
  - `SubscriptionRequest.PartitionRoute` reserves one neutral route description for future backends.
  - `PartitionRoute.Topic` names the provider-defined stream or topic family.
  - `PartitionRoute.PartitionKey` is limited to existing BigClaw selectors: `trace_id`, `task_id`, and `event_type`.
  - `PartitionRoute.OrderingScope` is descriptive only; callers still use `Position.Sequence` as the portable replay token.
- Broker-backed subscriber ownership:
  - `SubscriptionRequest.OwnershipContract` reserves one neutral ownership description for future backends.
  - `SubscriberOwnershipContract` carries `subscriber_group`, `consumer`, `mode`, `epoch`, `lease_token`, and optional `partition_hints`.
  - The contract is explicit that epoch plus lease token fencing is required before checkpoint writes can claim cross-process ownership transfer safety.
- These fields are contract-only in the current checkout. No broker-backed adapter, partitioned topic runtime, or shared durable ownership backend is shipped yet.

## Portable semantics

- Publish:
  - every accepted event must produce a monotonic `Position.Sequence` within the selected backend scope;
  - broker-backed backends may also attach `Partition` and `Offset`, but callers should treat sequence as the portable ordering token.
- Replay:
  - replay must return records in append order;
  - replay after a checkpoint uses `ReplayRequest.After` so consumers do not need provider-specific offset parsing in the core runtime;
  - task and trace filters remain stable across backends.
- Live subscribe:
  - subscriptions may include a replay prefix before live delivery, matching the current replay-then-live behavior already used by SSE/event consumers.
- Checkpoints:
  - checkpoints store the last durable `Position` per consumer identity;
  - committing a checkpoint acknowledges replay progress, not downstream business success beyond that consumer boundary;
  - future lease/fencing work can layer ownership semantics on top of the same portable position record without changing the event-log API.

## Runtime knobs for broker-backed backends

- `BIGCLAW_EVENT_LOG_BACKEND=broker` selects the broker-backed adapter family.
- `BIGCLAW_EVENT_LOG_BROKER_DRIVER` identifies the provider-specific adapter (`kafka`, `nats-jetstream`, `redis-streams`, etc.).
- `BIGCLAW_EVENT_LOG_BROKER_URLS` lists the broker bootstrap endpoints.
- `BIGCLAW_EVENT_LOG_BROKER_TOPIC` identifies the shared event stream/topic/subject family.
- `BIGCLAW_EVENT_LOG_CONSUMER_GROUP` provides the default shared subscriber identity when a backend needs one.
- `BIGCLAW_EVENT_LOG_PUBLISH_TIMEOUT` bounds append latency.
- `BIGCLAW_EVENT_LOG_REPLAY_LIMIT` sets the default catch-up page size for replay-oriented consumers.
- `BIGCLAW_EVENT_LOG_CHECKPOINT_INTERVAL` sets the default checkpoint flush cadence for streaming consumers.

## First backend implementation path

- Keep the existing in-process `Bus` for local fanout and SSE/live delivery.
- Use the event-log adapter as the append/replay/checkpoint boundary behind that bus.
- Implement one broker adapter behind `EventLog` plus `CheckpointStore`, then leave API handlers and worker/runtime code consuming the neutral contract.
- Preserve `RecorderSink` as audit/report evidence even after a broker adapter exists; it is a reporting sink, not the durability boundary.
