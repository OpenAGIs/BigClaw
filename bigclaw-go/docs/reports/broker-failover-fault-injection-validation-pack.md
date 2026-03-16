# Broker Failover And Replay Fault-Injection Validation Pack

## Canonical Evidence Bundle

Use `docs/reports/broker-failover-evidence-bundle.md` and `docs/reports/broker-failover-evidence-bundle.json` as the stable review-pack export surface for this validation work.

This document remains the detailed scenario matrix behind that bundle and should not be referenced as the only broker-failover artifact set anymore.

## Scope

This pack defines the minimum provider-neutral validation required before BigClaw claims broker-backed durability beyond the current local SQLite coordination proof.

It turns the current durability planning work into an implementation-ready checklist for future live validation across replicated logs, broker-backed publish paths, and externally coordinated checkpoints.

## Current Repo Baseline

- `docs/e2e-validation.md` covers local SQLite smoke, mixed workload runs, and two-node shared queue proof.
- `docs/reports/event-bus-reliability-report.md` proves replay only for the in-process event bus.
- `docs/reports/queue-reliability-report.md` and `docs/reports/lease-recovery-report.md` prove lease recovery and dead-letter replay for local queue backends.
- No broker-backed event log, provider failover harness, or cross-backend checkpoint recovery report exists yet.

## Validation Objectives

- prove publish durability across broker leader or node loss
- prove replay continuity after broker failover or client reconnect
- prove checkpoint safety so stale writers and partial acknowledgements cannot move consumer progress backward
- prove operator-visible evidence is sufficient to explain data loss, duplication, or recovery gaps
- keep the pack portable across candidate backends such as Redis Streams, NATS JetStream, Kafka-style logs, or a quorum-backed service

## Scenario Matrix

| Scenario ID | Fault injected | Setup | Required assertions |
| --- | --- | --- | --- |
| `BF-01` | active broker leader restarts during publish burst | one producer, one consumer group, monotonic event ids, steady publish load | accepted publishes remain replayable after failover; replay shows no gap in committed sequence; duplicates are either absent or explicitly flagged as replay-safe duplicates |
| `BF-02` | broker follower or replica loss during publish and replay | replicated backend with one non-leader node removed | no committed event becomes unreadable; publish latency spike is recorded; replay cursor resumes from last durable sequence |
| `BF-03` | consumer process crashes after handling an event but before checkpoint write is confirmed | one consumer group with explicit checkpoint acknowledgements | replay redelivers at most the uncheckpointed suffix; dedup metadata is sufficient to identify duplicate handling |
| `BF-04` | checkpoint store leader changes while two consumers contend for one subscriber group | two consumers, one active lease owner, one standby | stale checkpoint writer is fenced; newer owner does not regress to an older checkpoint; lease transition is visible in artifacts |
| `BF-05` | producer timeout after broker commit ambiguity | injected timeout between client ack and broker durability response | report distinguishes unknown commit from rejected publish; replay/audit evidence resolves whether the event landed |
| `BF-06` | replay client disconnects during catch-up and reconnects to a new broker node | replay from `after_id` or equivalent cursor across reconnect | resumed replay starts from the last confirmed cursor without skipping committed events |
| `BF-07` | retention or compaction boundary intersects a stale checkpoint | replay request starts behind retained history | system surfaces explicit truncation or reset signal; operator guidance identifies the safe recovery path instead of silently starting from an unsafe cursor |
| `BF-08` | split-brain style duplicate delivery window during failover | provider or proxy allows duplicate fetch / delivery after election | duplicate deliveries preserve event identity fields; checkpoint and dedup logic keep terminal processing idempotent |

## Assertions By Surface

### Publish Assertions

- every accepted publish has a stable event identity, trace linkage, and broker durability marker or equivalent commit proof
- every rejected or ambiguous publish is audit-visible and classifiable as `rejected`, `unknown_commit`, or `committed`
- post-failover replay can account for every event accepted before the fault window closed

### Replay Assertions

- replay cursors remain monotonic across reconnect, broker election, and consumer restart
- replayed events preserve identity fields needed for downstream dedup
- handoff from replay to live delivery does not skip committed events at the failover boundary
- duplicate replay is allowed only when event identity makes dedup and operator explanation possible

### Checkpoint Assertions

- checkpoint writes are monotonic by durable event sequence, not by arrival order
- stale owners cannot overwrite newer checkpoints after lease expiry or coordinator failover
- recovered consumers can prove which event was the last safely checkpointed item before the fault
- checkpoint artifacts distinguish `leased`, `acked_pending_commit`, `committed`, `replayed`, and `fenced` transitions

## Minimum Harness Requirements

- start at least three backend roles when the provider supports it: leader, follower/replica, and client-facing producer/consumer
- drive one publish stream and one replay-capable consumer group with deterministic event ids
- inject broker restart, network cut, process kill, delayed ack, and forced reconnect faults on demand
- record wall-clock timestamps and durable event sequences before, during, and after each fault
- export one machine-readable report per scenario plus raw event/checkpoint artifacts for investigation

## Minimum Report Schema

Each scenario report should include at least:

- `scenario_id`
- `backend`
- `topology`
- `fault_window`
- `published_count`
- `committed_count`
- `replayed_count`
- `duplicate_count`
- `missing_event_ids`
- `checkpoint_before_fault`
- `checkpoint_after_recovery`
- `lease_transitions`
- `publish_outcomes`
- `replay_resume_cursor`
- `artifacts`
- `result`

## Required Raw Artifacts

- publish attempt ledger with event id, trace id, attempt number, and client outcome
- replay capture with durable sequence, event id, replay/live flag, and source node
- checkpoint transition log with owner id, lease id, prior sequence, next sequence, and fence reason
- fault timeline with exact injected action and node target
- backend health snapshot before and after recovery

## Pass And Fail Rules

### Pass

- zero unexplained missing committed events
- zero checkpoint regressions
- duplicate delivery count is zero or fully matched by dedup-safe identities
- every ambiguous publish outcome is resolvable from replay plus audit evidence

### Fail

- any committed event disappears from replay after failover
- any stale consumer can overwrite a newer checkpoint
- any report cannot explain whether an event was committed, duplicated, or lost
- recovery requires operator guesswork instead of explicit evidence

## Implementation Order

1. Add a provider-neutral scenario runner that can emit the report schema above against a fake broker or deterministic stub.
2. Reuse the existing replay and checkpoint semantics already documented in `docs/e2e-validation.md`, `docs/reports/queue-reliability-report.md`, and `docs/reports/lease-recovery-report.md`.
3. Introduce one backend adapter at a time behind the same scenario ids so result diffs stay comparable.
4. Promote the pack into live broker validation only after the stubbed harness can prove sequence accounting and checkpoint fencing deterministically.

## Live Validation Follow-On Path

- add `scripts/e2e/broker_failover_validation.py` or equivalent once a broker-backed event log exists
- write reports to `docs/reports/broker-failover-<backend>-report.json`
- refresh `docs/reports/broker-failover-evidence-bundle.json` as the canonical machine-readable manifest
- keep `docs/reports/broker-failover-evidence-bundle.md` as the stable markdown summary across backends
- link the canonical bundle from `docs/e2e-validation.md` beside the current SQLite and multi-node validation commands

## Exit Criteria For Future Implementation Ticket

- at least one broker-backed backend passes `BF-01` through `BF-06`
- retention-aware coverage exists for `BF-07` when compaction work lands
- duplicate-delivery coverage exists for `BF-08` when subscriber dedup semantics land
- report artifacts are sufficient for Linear closeout without manual log spelunking
