# Multi-Subscriber Takeover Validation Report

## Scope

This report captures the remaining fault-injection and evidence contract for `OPE-217` after lease-aware subscriber-group checkpoint coordination landed for the in-process event bus, but before a durable shared multi-node coordination backend exists.

## Current Evidence Inputs

- `internal/events/bus.go`
- `internal/events/bus_test.go`
- `docs/reports/event-bus-reliability-report.md`
- `docs/reports/lease-recovery-report.md`
- `scripts/e2e/multi_node_shared_queue.py`
- `docs/reports/multi-node-shared-queue-report.json`

## Planned Fault Scenarios

- Primary subscriber crashes after processing an event but before its checkpoint is durably advanced.
- Lease ownership expires, a standby takes over, and the former owner attempts a stale checkpoint write.
- A brief split-brain window allows two subscribers to attempt replay of the same tail before fencing converges.

## Required Assertions

- Audit assertions:
  - ownership acquisition, expiry, rejection, and takeover must form one ordered timeline per subscriber group
  - per-node audit paths must be preserved so cross-node evidence can be inspected directly
  - stale-writer rejections must identify the attempted owner and accepted owner
- Checkpoint assertions:
  - durable checkpoints stay monotonic across takeovers
  - only the active lease owner can advance the durable checkpoint
  - takeover must not allow a late writer to move the checkpoint backwards
- Replay assertions:
  - takeover replay starts from the last durable checkpoint
  - duplicate deliveries in the uncheckpointed tail are counted explicitly
  - the final replay cursor and final owner are both reported

## Minimum Harness Output

The canonical generated matrix lives in `docs/reports/multi-subscriber-takeover-validation-report.json` and defines the minimum report fields required for repeatable evidence:

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
- `stale_write_rejections`
- `audit_log_paths`
- `event_log_excerpt`
- `all_assertions_passed`

## Current Result

- The repo now has a generated, reviewable scenario matrix for takeover fault injection instead of an implied TODO.
- Existing evidence is sufficient to define the report contract, but not yet to execute the takeover scenarios end to end under shared multi-node conditions.
- The next implementation slice should carry the existing lease ownership and normalized audit semantics into a durable shared multi-node backend so the harness can execute this matrix directly.
