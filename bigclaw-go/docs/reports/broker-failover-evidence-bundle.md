# Broker Failover Evidence Bundle

- Ticket: `OPE-239`
- Status: `planning-ready`
- Canonical manifest: `docs/reports/broker-failover-evidence-bundle.json`
- Detailed scenario pack: `docs/reports/broker-failover-fault-injection-validation-pack.md`
- Rollout contract: `docs/reports/replicated-event-log-durability-rollout-contract.md`

## Purpose

This bundle is the stable review-pack and rollout-facing entrypoint for broker failover and replay fault-injection evidence.

It normalizes the current planning-only artifacts into one canonical surface without implying that a broker-backed event log, live provider adapter, or executable failover harness already exists in the repo.

## Canonical bundle contents

- `docs/reports/broker-failover-evidence-bundle.json`
  - machine-readable manifest for review packs and future automation
- `docs/reports/broker-failover-fault-injection-validation-pack.md`
  - provider-neutral scenario matrix, assertions, and pass/fail rules
- `docs/reports/replicated-event-log-durability-rollout-contract.md`
  - rollout gates and operator-facing runtime expectations for replicated durability
- `docs/reports/event-bus-reliability-report.md`
  - current baseline reliability evidence and portability constraints
- `docs/reports/replay-retention-semantics-report.md`
  - retention-boundary and stale-checkpoint recovery expectations

## Scenario coverage

| Scenario ID | Coverage | Current readiness |
| --- | --- | --- |
| `BF-01` | leader restart during publish burst | planning-ready |
| `BF-02` | follower or replica loss during publish and replay | planning-ready |
| `BF-03` | consumer crash before checkpoint confirmation | planning-ready |
| `BF-04` | checkpoint leader change during consumer contention | planning-ready |
| `BF-05` | producer timeout with ambiguous commit outcome | planning-ready |
| `BF-06` | replay reconnect to a different broker node | planning-ready |
| `BF-07` | retention boundary intersects a stale checkpoint | planning-ready |
| `BF-08` | duplicate-delivery window during failover | planning-ready |

## Machine-readable report contract

Future executable reports should emit at least:

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

Required raw artifacts remain:

- publish attempt ledger with event id, trace id, attempt number, and client outcome
- replay capture with durable sequence, event id, replay/live flag, and source node
- checkpoint transition log with owner id, lease id, prior sequence, next sequence, and fence reason
- fault timeline with exact injected action and node target
- backend health snapshot before and after recovery

## Future live export shape

When a broker-backed harness exists, the canonical export surface should stay anchored here:

- canonical manifest: `docs/reports/broker-failover-evidence-bundle.json`
- canonical index: `docs/reports/broker-failover-evidence-bundle.md`
- per-run bundle root: `docs/reports/broker-failover-runs/<run_id>/`
- per-backend report pattern: `docs/reports/broker-failover-<backend>-report.json`

## Current usage rules

- review packs may cite this bundle as the canonical broker failover evidence surface
- rollout docs may point here for planned failover, replay, and checkpoint-fencing expectations
- no document should claim live broker validation has passed until executable artifacts populate the future bundle shape above
