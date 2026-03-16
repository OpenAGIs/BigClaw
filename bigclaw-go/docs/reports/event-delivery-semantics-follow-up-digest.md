# Event Delivery Semantics Follow-up Digest

## Scope

This digest consolidates the remaining event-delivery semantics for `OPE-262` so reviewers can verify the current durable dedupe, acknowledgement, and exactly-once posture from one repo-native source.

## Canonical delivery contract

| Topic | Current contract | Current evidence | Remaining follow-up |
| --- | --- | --- | --- |
| Durable dedupe | Replay-safe consumers still need a durable dedupe store keyed by `delivery.idempotency_key` before side effects can survive retries or takeover without reapplication. | `internal/events/delivery.go`, `internal/domain/consumer_dedup.go`, `internal/events/dedup_ledger.go`, `docs/reports/event-bus-reliability-report.md` | Only the SQLite durable dedupe backend exists in this checkout; HTTP and broker-backed dedupe persistence are still future work. |
| Delivery acknowledgement | Publish and checkpoint acknowledgements currently prove transport or replay-progress boundaries only. No end-to-end delivery acknowledgement protocol exists beyond sink-level best-effort delivery. | `docs/reports/event-bus-reliability-report.md`, `docs/reports/replicated-event-log-durability-rollout-contract.md`, `docs/reports/multi-subscriber-takeover-validation-report.md` | Replicated adapters still need committed/rejected/ambiguous acknowledgement handling plus takeover-safe checkpoint ownership across nodes. |
| Exactly-once posture | BigClaw remains replay-safe, not globally exactly-once. Durable replay, checkpointing, and dedupe reduce duplicate side effects, but they do not prove one-and-only-once business execution across distributed sinks. | `docs/openclaw-parallel-gap-analysis.md`, `docs/reports/event-bus-reliability-report.md`, `docs/reports/review-readiness.md` | Broker/quorum durability, cross-node coordination, and durable downstream dedupe need to land before a stronger exactly-once claim is even reviewable. |

## Reviewer guidance

- Treat `delivery.idempotency_key` as the downstream dedupe boundary, not as proof that the platform has already enforced exactly-once side effects.
- Treat checkpoint acknowledgement as proof of replay progress for the active consumer boundary, not proof that downstream business work completed successfully.
- Treat publish acknowledgement as a transport-level outcome that still needs replicated committed/rejected/ambiguous handling before a broker-backed adapter can be called rollout-ready.
- Use `docs/reports/replicated-event-log-durability-rollout-contract.md` for the replicated publish/replay gate and `docs/reports/multi-subscriber-takeover-validation-report.md` for the cross-node takeover validation shape.

## Linked report surfaces

- `docs/reports/event-bus-reliability-report.md`
- `docs/reports/issue-coverage.md`
- `docs/reports/review-readiness.md`
- `docs/reports/multi-subscriber-takeover-validation-report.md`
- `docs/openclaw-parallel-gap-analysis.md`
