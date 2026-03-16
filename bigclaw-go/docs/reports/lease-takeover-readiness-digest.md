# Lease And Takeover Readiness Digest

## Scope

This digest consolidates the current lease recovery, shared-queue coordination, and takeover-readiness evidence for `OPE-246` into one reviewer-facing report.

## Current Evidence Snapshot

- Lease recovery report: `docs/reports/lease-recovery-report.md`
- Multi-node coordination report: `docs/reports/multi-node-coordination-report.md`
- Coordination artifact: `docs/reports/multi-node-shared-queue-report.json`
- Takeover validation contract: `docs/reports/multi-subscriber-takeover-validation-report.md`
- Takeover validation matrix: `docs/reports/multi-subscriber-takeover-validation-report.json`

## Lease Recovery Evidence

- Worker leases carry explicit owner, attempt, acquisition time, and expiry metadata.
- Worker runtime renews leases on a heartbeat interval while execution is active.
- Expired SQLite leases become available for reacquisition by a different worker.
- Replay and dead-letter operations clear lease ownership and return the task to an actionable state.
- A `1k` task concurrent processing scenario now completes without duplicate consumption.

## Shared-Queue Coordination Evidence

- Run date: `2026-03-13`
- Command: ``python3 scripts/e2e/multi_node_shared_queue.py --count 200 --submit-workers 8 --timeout-seconds 180 --report-path docs/reports/multi-node-shared-queue-report.json``
- Total tasks: `200`
- Submitted by `node-a`: `100`
- Submitted by `node-b`: `100`
- Completed by `node-a`: `73`
- Completed by `node-b`: `127`
- Cross-node completions: `99`
- Duplicate `task.started`: `0`
- Duplicate `task.completed`: `0`
- Missing terminal completions: `0`
- Overall result: `pass`

## Takeover Readiness Contract

- Current status: `planning-ready`
- Source ticket: `OPE-217`
- Generated matrix timestamp: `2026-03-15T09:19:19Z`
- Required report sections: `scenario metadata`, `fault injection steps`, `audit assertions`, `checkpoint assertions`, `replay assertions`, `per-node audit artifacts`, `final owner and replay cursor summary`, `duplicate delivery accounting`, `open blockers and follow-up implementation hooks`

The takeover matrix is planning-ready evidence. It defines the assertions and report shape reviewers should expect once shared multi-node subscriber-group takeover automation exists, but it does not claim the end-to-end fault injection is implemented today.

## Scenario Assertions

### `takeover-after-primary-crash`

- Title: Primary subscriber crashes after processing but before checkpoint flush
- Target gap: prove takeover replays the uncheckpointed tail without losing or double-committing progress
- Audit assertions:
  - Audit log shows one ownership handoff from primary to standby.
  - Audit log records the primary interruption reason before standby completion.
  - Audit log links takeover to the same task or trace identifier across both subscribers.
- Checkpoint assertions:
  - Checkpoint after takeover is greater than or equal to the last durable checkpoint from the primary.
  - Standby checkpoint commit is attributed to the new lease owner.
  - No checkpoint update is accepted from the crashed primary after takeover.
- Replay assertions:
  - Replay resumes from the last durable checkpoint, not from the last in-memory event processed by the crashed primary.
  - At most one duplicate delivery is tolerated for the uncheckpointed tail and it is visible in the report.
  - Replay window closes once the standby checkpoint advances past the tail.
- Current blockers:
  - Subscriber-group checkpoint leases are not implemented yet.
  - No audit event schema currently records subscriber-group ownership transfers.

### `lease-expiry-stale-writer-rejected`

- Title: Lease expires and the former owner attempts a stale checkpoint write
- Target gap: prove stale writers cannot move a subscriber-group checkpoint backwards after takeover
- Audit assertions:
  - Audit log records lease expiry for the former owner and acquisition by the standby.
  - Audit log records the stale write rejection with both attempted and accepted owners.
  - Audit log keeps the rejection and accepted takeover in the same ordered timeline.
- Checkpoint assertions:
  - Checkpoint sequence never decreases after the standby acquires ownership.
  - Late primary acknowledgement is rejected or ignored without mutating durable checkpoint state.
  - Accepted checkpoint owner always matches the active lease holder.
- Replay assertions:
  - Replay after stale write rejection starts from the accepted durable checkpoint only.
  - No event acknowledged only by the stale writer disappears from the replay timeline.
  - Replay report exposes any duplicate event IDs caused by the overlap window.
- Current blockers:
  - Checkpoint ownership is not fenced by lease metadata yet.
  - No current API/report payload exposes stale checkpoint rejection counts.

### `split-brain-dual-replay-window`

- Title: Two subscribers briefly believe they can replay the same tail
- Target gap: prove audit evidence is strong enough to diagnose duplicate replay attempts and the winning lease owner
- Audit assertions:
  - Combined audit timeline shows overlapping replay attempts and identifies the surviving owner.
  - Audit evidence includes per-node file paths and normalized subscriber identities.
  - The final report highlights whether duplicate replay attempts were observed or only simulated.
- Checkpoint assertions:
  - Only the winning owner can advance the durable checkpoint.
  - Losing owner leaves durable checkpoint unchanged once fencing is applied.
  - Report includes the exact checkpoint sequence where overlap began and ended.
- Replay assertions:
  - Replay output groups duplicate candidate deliveries by event ID.
  - Final replay cursor belongs to the winning owner only.
  - Validation reports whether overlapping replay created observable duplicate deliveries.
- Current blockers:
  - No subscriber-group membership or lease coordinator exists yet.
  - Replay reports do not currently aggregate duplicate candidates by event ID across nodes.

## Review Guidance

- Lease recovery is directly covered by automated tests instead of only being implied by implementation.
- The SQLite queue no longer surfaces the previous `database is locked` failure under the added concurrent reliability test.
- The system supports the core recovery loop expected for worker interruption scenarios.
- The repo now has a generated, reviewable scenario matrix for takeover fault injection instead of an implied TODO.
- Existing evidence is sufficient to define the report contract, but not yet to execute the takeover scenarios end to end.
- The next implementation slice should add lease-aware checkpoint ownership metadata and normalized audit events so the shared multi-node harness can execute this matrix directly.
- Current shared-queue evidence proves lease expiry recovery and two-node coordination, but not durable cross-node subscriber-group takeover fencing.
- `docs/reports/event-bus-reliability-report.md` documents subscriber-group lease concepts and current API/reporting boundaries; reviewers should treat the takeover matrix here as the stricter cross-node readiness contract.

## Validation

- Regenerate with `python3 scripts/e2e/lease_takeover_readiness_digest.py --write`.
- Verify consistency with `python3 scripts/e2e/lease_takeover_readiness_digest.py --check`.

