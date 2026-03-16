# Lease and Takeover Readiness Digest

## Scope

This digest consolidates the current reviewer-facing evidence for lease recovery, shared-queue coordination, and takeover-readiness planning for `OPE-246`.

It separates implemented proof from takeover assertions that are still planned so the repo does not imply multi-subscriber takeover automation that has not been shipped yet.

## Implemented Evidence

### Lease Recovery

- Artifact: `docs/reports/lease-recovery-report.md`
- Current proof:
  - automated queue tests cover lease expiry, reacquisition, and `1k` concurrent processing without duplicate lease consumption
  - worker runtime renews active leases on a heartbeat while work is still executing
  - replay and dead-letter flows clear lease ownership so work can re-enter an actionable state
- Reviewer takeaway:
  - the current Go queue layer has direct automated evidence for worker interruption and lease reacquisition behavior

### Shared-Queue Multi-Node Coordination

- Artifact: `docs/reports/multi-node-coordination-report.md`
- Supporting data: `docs/reports/multi-node-shared-queue-report.json`
- Current proof:
  - two independent `bigclawd` processes shared one SQLite-backed queue in a `200` task run
  - cross-node completions occurred on both nodes with `0` duplicate `task.started`, `0` duplicate `task.completed`, and `0` missing terminal completions
- Reviewer takeaway:
  - the current local topology has a concrete two-node coordination proof for queue consumption
  - this is coordination evidence, not a dedicated leader-election or subscriber-takeover implementation

## Planned Takeover Readiness

### Canonical Takeover Matrix

- Artifacts:
  - `docs/reports/multi-subscriber-takeover-validation-report.md`
  - `docs/reports/multi-subscriber-takeover-validation-report.json`
- Current planning coverage:
  - primary crash before durable checkpoint flush
  - lease expiry followed by stale-writer rejection
  - split-brain replay overlap with one surviving owner
- Required end-state assertions:
  - only the active lease owner can advance the durable checkpoint
  - durable checkpoints stay monotonic across takeovers
  - takeover replay starts from the last durable checkpoint and reports duplicate tail deliveries explicitly
  - audit evidence preserves ordered ownership transitions and stale-writer rejection details
- Reviewer takeaway:
  - the repo now has a stable, reviewable takeover evidence contract and scenario matrix
  - these files define what must be proven; they are not proof that subscriber-group takeover is implemented today

## Current Blockers

- Subscriber-group checkpoint leases are not implemented yet.
- No normalized audit schema currently records subscriber-group ownership acquire, renew, expire, reject, and takeover transitions.
- The shared multi-node harness does not yet execute the takeover matrix end to end.
- Replay reports do not yet aggregate duplicate candidate deliveries and stale-writer rejection counters across takeover scenarios.

## Honest Readiness Summary

- Implemented today:
  - worker/task lease recovery evidence
  - two-node shared-queue coordination evidence
  - a generated takeover validation contract for future fault-injection coverage
- Not implemented today:
  - lease-aware subscriber-group checkpoint fencing
  - automated subscriber takeover execution with durable checkpoint ownership enforcement
  - end-to-end takeover reports that prove stale-writer rejection and takeover replay behavior

## Reviewer Artifact Order

1. Read `docs/reports/lease-recovery-report.md` for the current automated lease-recovery proof.
2. Read `docs/reports/multi-node-coordination-report.md` and `docs/reports/multi-node-shared-queue-report.json` for concrete cross-node queue coordination evidence.
3. Read `docs/reports/multi-subscriber-takeover-validation-report.md` and `docs/reports/multi-subscriber-takeover-validation-report.json` for the planned takeover matrix, expected assertions, and remaining implementation blockers.
