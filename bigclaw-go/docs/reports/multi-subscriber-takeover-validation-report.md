# Multi-Subscriber Takeover Validation Report

## Scope

This report captures the executable local takeover harness for `OPE-269` / `BIG-PAR-080`.

## Current Evidence Inputs

- `internal/events/subscriber_leases.go`
- `internal/events/subscriber_leases_test.go`
- `docs/reports/event-bus-reliability-report.md`
- `scripts/e2e/subscriber_takeover_fault_matrix.py`
- `scripts/e2e/multi_node_shared_queue.py`
- `docs/reports/multi-node-shared-queue-report.json`

## Executed Fault Scenarios

- Primary subscriber crashes after processing an event but before its checkpoint is durably advanced.
- Lease ownership expires, a standby takes over, and the former owner attempts a stale checkpoint write.
- A brief split-brain window creates overlapping replay candidates before fencing converges on one owner.

## Required Assertions

- Audit assertions:
  - ownership acquisition, expiry, rejection, and takeover form one ordered timeline per subscriber group
  - normalized audit paths stay stable enough that future live per-node artifacts can adopt the same schema
  - stale-writer rejections identify the attempted owner and the accepted owner
- Checkpoint assertions:
  - durable checkpoints stay monotonic across takeovers
  - only the active lease owner can advance the durable checkpoint
  - takeover does not allow a late writer to move the checkpoint backwards
- Replay assertions:
  - takeover replay starts from the last durable checkpoint
  - duplicate deliveries in the uncheckpointed tail are counted explicitly
  - the final replay cursor and final owner are both reported

## Minimum Harness Output

The canonical executable report lives in `docs/reports/multi-subscriber-takeover-validation-report.json` and defines the minimum machine-readable fields for repeatable takeover evidence:

- `scenario_id`
- `subscriber_group`
- `primary_subscriber`
- `takeover_subscriber`
- `task_or_trace_id`
- `lease_owner_timeline`
- `checkpoint_before`
- `checkpoint_after`
- `replay_start_cursor`
- `replay_end_cursor`
- `duplicate_delivery_count`
- `duplicate_events`
- `stale_write_rejections`
- `audit_log_paths`
- `event_log_excerpt`
- `assertion_results`
- `all_assertions_passed`
- `local_limitations`

## Current Result

- The repo now ships a deterministic local harness that executes the canonical takeover scenarios instead of only describing them as a future matrix.
- The generated report proves lease-aware ownership handoff, stale-writer fencing, checkpoint monotonicity, and duplicate replay accounting at the local harness level.
- The next implementation slice should wire the same schema into the shared multi-node harness so live `bigclawd` processes emit the same proof contract.
- The remaining live multi-node caveats are consolidated in `docs/reports/subscriber-takeover-executability-follow-up-digest.md`.
