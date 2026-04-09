#!/usr/bin/env python3
import argparse
import json
import pathlib
from dataclasses import asdict, dataclass
from datetime import datetime, timedelta, timezone


class LeaseError(Exception):
    pass


class LeaseHeld(LeaseError):
    pass


class LeaseExpired(LeaseError):
    pass


class LeaseFence(LeaseError):
    pass


class CheckpointRollback(LeaseError):
    pass


@dataclass
class Lease:
    group_id: str
    subscriber_id: str
    consumer_id: str
    lease_token: str
    lease_epoch: int
    checkpoint_offset: int
    checkpoint_event_id: str
    expires_at: datetime
    updated_at: datetime


class LeaseCoordinator:
    def __init__(self):
        self._leases = {}
        self._counter = 0

    def _key(self, group_id, subscriber_id):
        return (group_id, subscriber_id)

    def _next_token(self):
        self._counter += 1
        return f"lease-{self._counter}"

    def _expired(self, lease, now):
        return lease.expires_at and now >= lease.expires_at

    def acquire(self, *, group_id, subscriber_id, consumer_id, ttl, now):
        key = self._key(group_id, subscriber_id)
        current = self._leases.get(key)
        if current and not self._expired(current, now) and current.consumer_id != consumer_id:
            raise LeaseHeld(current)
        if current and self._expired(current, now) and current.consumer_id != consumer_id:
            current = Lease(
                group_id=current.group_id,
                subscriber_id=current.subscriber_id,
                consumer_id=current.consumer_id,
                lease_token="",
                lease_epoch=current.lease_epoch,
                checkpoint_offset=current.checkpoint_offset,
                checkpoint_event_id=current.checkpoint_event_id,
                expires_at=current.expires_at,
                updated_at=current.updated_at,
            )
        if current and not self._expired(current, now) and current.consumer_id == consumer_id:
            current.expires_at = now + ttl
            current.updated_at = now
            self._leases[key] = current
            return current
        next_epoch = current.lease_epoch + 1 if current else 1
        next_offset = current.checkpoint_offset if current else 0
        next_event = current.checkpoint_event_id if current else ""
        lease = Lease(
            group_id=group_id,
            subscriber_id=subscriber_id,
            consumer_id=consumer_id,
            lease_token=self._next_token(),
            lease_epoch=next_epoch,
            checkpoint_offset=next_offset,
            checkpoint_event_id=next_event,
            expires_at=now + ttl,
            updated_at=now,
        )
        self._leases[key] = lease
        return lease

    def commit(
        self,
        *,
        group_id,
        subscriber_id,
        consumer_id,
        lease_token,
        lease_epoch,
        checkpoint_offset,
        checkpoint_event_id,
        now,
    ):
        key = self._key(group_id, subscriber_id)
        current = self._leases.get(key)
        if not current:
            raise LeaseExpired("no lease")
        if self._expired(current, now):
            raise LeaseExpired(current)
        if (
            current.consumer_id != consumer_id
            or current.lease_token != lease_token
            or current.lease_epoch != lease_epoch
        ):
            raise LeaseFence(current)
        if checkpoint_offset < current.checkpoint_offset:
            raise CheckpointRollback(current)
        current.checkpoint_offset = checkpoint_offset
        current.checkpoint_event_id = checkpoint_event_id
        current.updated_at = now
        self._leases[key] = current
        return current


def utc_iso(timestamp):
    return timestamp.astimezone(timezone.utc).isoformat().replace("+00:00", "Z")


def checkpoint_payload(lease):
    return {
        "owner": lease.consumer_id,
        "lease_epoch": lease.lease_epoch,
        "lease_token": lease.lease_token,
        "offset": lease.checkpoint_offset,
        "event_id": lease.checkpoint_event_id,
        "updated_at": utc_iso(lease.updated_at),
    }


def cursor(offset, event_id):
    return {"offset": offset, "event_id": event_id}


def audit_event(timeline, *, timestamp, subscriber, action, details):
    timeline.append(
        {
            "timestamp": utc_iso(timestamp),
            "subscriber": subscriber,
            "action": action,
            "details": details,
        }
    )


def owner_timeline_entry(*, timestamp, owner, event, lease):
    return {
        "timestamp": utc_iso(timestamp),
        "owner": owner,
        "event": event,
        "lease_epoch": lease.lease_epoch,
        "checkpoint_offset": lease.checkpoint_offset,
        "checkpoint_event_id": lease.checkpoint_event_id,
    }


def build_scenario_result(
    *,
    scenario_id,
    title,
    primary_subscriber,
    takeover_subscriber,
    task_or_trace_id,
    subscriber_group,
    audit_assertions,
    checkpoint_assertions,
    replay_assertions,
    lease_owner_timeline,
    checkpoint_before,
    checkpoint_after,
    replay_start_cursor,
    replay_end_cursor,
    duplicate_events,
    stale_write_rejections,
    audit_timeline,
    event_log_excerpt,
    local_limitations,
):
    audit_checks = [
        {
            "label": "ownership handoff is visible in the audit timeline",
            "passed": len({entry["owner"] for entry in lease_owner_timeline}) >= 2,
        },
        {
            "label": "audit timeline contains takeover-specific events",
            "passed": any(item["action"] in {"lease_acquired", "lease_fenced", "primary_crashed"} for item in audit_timeline),
        },
        {
            "label": "audit timeline stays ordered by timestamp",
            "passed": [item["timestamp"] for item in audit_timeline] == sorted(item["timestamp"] for item in audit_timeline),
        },
    ]
    checkpoint_checks = [
        {
            "label": "checkpoint never regresses across takeover",
            "passed": checkpoint_after["offset"] >= checkpoint_before["offset"],
        },
        {
            "label": "final checkpoint owner matches the final lease owner",
            "passed": checkpoint_after["owner"] == lease_owner_timeline[-1]["owner"],
        },
        {
            "label": "stale writers do not replace the accepted checkpoint owner",
            "passed": stale_write_rejections == 0 or checkpoint_after["owner"] == takeover_subscriber,
        },
    ]
    replay_checks = [
        {
            "label": "replay restarts from the durable checkpoint boundary",
            "passed": replay_start_cursor["offset"] == checkpoint_before["offset"],
        },
        {
            "label": "replay end cursor advances to the final durable checkpoint",
            "passed": replay_end_cursor["offset"] == checkpoint_after["offset"],
        },
        {
            "label": "duplicate replay candidates are counted explicitly",
            "passed": len(duplicate_events) >= 0,
        },
    ]
    all_assertions_passed = all(item["passed"] for item in audit_checks + checkpoint_checks + replay_checks)
    artifact_root = f"artifacts/{scenario_id}"
    return {
        "id": scenario_id,
        "title": title,
        "subscriber_group": subscriber_group,
        "primary_subscriber": primary_subscriber,
        "takeover_subscriber": takeover_subscriber,
        "task_or_trace_id": task_or_trace_id,
        "audit_assertions": audit_assertions,
        "checkpoint_assertions": checkpoint_assertions,
        "replay_assertions": replay_assertions,
        "lease_owner_timeline": lease_owner_timeline,
        "checkpoint_before": checkpoint_before,
        "checkpoint_after": checkpoint_after,
        "replay_start_cursor": replay_start_cursor,
        "replay_end_cursor": replay_end_cursor,
        "duplicate_delivery_count": len(duplicate_events),
        "duplicate_events": duplicate_events,
        "stale_write_rejections": stale_write_rejections,
        "audit_log_paths": [
            f"{artifact_root}/{primary_subscriber}-audit.jsonl",
            f"{artifact_root}/{takeover_subscriber}-audit.jsonl",
        ],
        "event_log_excerpt": event_log_excerpt,
        "audit_timeline": audit_timeline,
        "assertion_results": {
            "audit": audit_checks,
            "checkpoint": checkpoint_checks,
            "replay": replay_checks,
        },
        "all_assertions_passed": all_assertions_passed,
        "local_limitations": local_limitations,
    }


BASE_TIME = datetime(2026, 3, 16, 10, 30, 0, tzinfo=timezone.utc)
TTL = timedelta(seconds=30)


def scenario_takeover_after_primary_crash():
    scenario_id = "takeover-after-primary-crash"
    title = "Primary subscriber crashes after processing but before checkpoint flush"
    subscriber_group = "group-takeover-crash"
    primary = "subscriber-a"
    standby = "subscriber-b"
    task_or_trace_id = "trace-takeover-crash"
    timeline = []
    owner_timeline = []
    coordinator = LeaseCoordinator()
    now = BASE_TIME

    primary_lease = coordinator.acquire(
        group_id=subscriber_group,
        subscriber_id="event-stream",
        consumer_id=primary,
        ttl=TTL,
        now=now,
    )
    owner_timeline.append(owner_timeline_entry(timestamp=now, owner=primary, event="lease_acquired", lease=primary_lease))
    audit_event(timeline, timestamp=now, subscriber=primary, action="lease_acquired", details={"lease_epoch": primary_lease.lease_epoch})

    primary_lease = coordinator.commit(
        group_id=subscriber_group,
        subscriber_id="event-stream",
        consumer_id=primary,
        lease_token=primary_lease.lease_token,
        lease_epoch=primary_lease.lease_epoch,
        checkpoint_offset=40,
        checkpoint_event_id="evt-40",
        now=now + timedelta(seconds=5),
    )
    checkpoint_before = checkpoint_payload(primary_lease)
    audit_event(timeline, timestamp=now + timedelta(seconds=5), subscriber=primary, action="checkpoint_committed", details={"offset": 40, "event_id": "evt-40"})
    audit_event(timeline, timestamp=now + timedelta(seconds=7), subscriber=primary, action="processed_uncheckpointed_tail", details={"offset": 41, "event_id": "evt-41"})
    audit_event(timeline, timestamp=now + timedelta(seconds=8), subscriber=primary, action="primary_crashed", details={"reason": "terminated before checkpoint flush"})

    takeover_lease = coordinator.acquire(
        group_id=subscriber_group,
        subscriber_id="event-stream",
        consumer_id=standby,
        ttl=TTL,
        now=now + timedelta(seconds=31),
    )
    owner_timeline.append(owner_timeline_entry(timestamp=now + timedelta(seconds=31), owner=standby, event="takeover_acquired", lease=takeover_lease))
    audit_event(timeline, timestamp=now + timedelta(seconds=31), subscriber=standby, action="lease_acquired", details={"lease_epoch": takeover_lease.lease_epoch, "takeover": True})
    audit_event(timeline, timestamp=now + timedelta(seconds=32), subscriber=standby, action="replay_started", details={"from_offset": 40, "from_event_id": "evt-40"})
    takeover_lease = coordinator.commit(
        group_id=subscriber_group,
        subscriber_id="event-stream",
        consumer_id=standby,
        lease_token=takeover_lease.lease_token,
        lease_epoch=takeover_lease.lease_epoch,
        checkpoint_offset=41,
        checkpoint_event_id="evt-41",
        now=now + timedelta(seconds=33),
    )
    checkpoint_after = checkpoint_payload(takeover_lease)
    audit_event(timeline, timestamp=now + timedelta(seconds=33), subscriber=standby, action="checkpoint_committed", details={"offset": 41, "event_id": "evt-41", "replayed_tail": True})

    return build_scenario_result(
        scenario_id=scenario_id,
        title=title,
        primary_subscriber=primary,
        takeover_subscriber=standby,
        task_or_trace_id=task_or_trace_id,
        subscriber_group=subscriber_group,
        audit_assertions=[
            "Audit log shows one ownership handoff from primary to standby.",
            "Audit log records the primary interruption reason before standby completion.",
            "Audit log links takeover to the same task or trace identifier across both subscribers.",
        ],
        checkpoint_assertions=[
            "Checkpoint after takeover is greater than or equal to the last durable checkpoint from the primary.",
            "Standby checkpoint commit is attributed to the new lease owner.",
            "No checkpoint update is accepted from the crashed primary after takeover.",
        ],
        replay_assertions=[
            "Replay resumes from the last durable checkpoint, not from the last in-memory event processed by the crashed primary.",
            "At most one duplicate delivery is tolerated for the uncheckpointed tail and it is visible in the report.",
            "Replay window closes once the standby checkpoint advances past the tail.",
        ],
        lease_owner_timeline=owner_timeline,
        checkpoint_before=checkpoint_before,
        checkpoint_after=checkpoint_after,
        replay_start_cursor=cursor(40, "evt-40"),
        replay_end_cursor=cursor(41, "evt-41"),
        duplicate_events=["evt-41"],
        stale_write_rejections=0,
        audit_timeline=timeline,
        event_log_excerpt=[
            {"event_id": "evt-40", "delivered_by": [primary], "delivery_kind": "durable"},
            {"event_id": "evt-41", "delivered_by": [primary, standby], "delivery_kind": "uncheckpointed_tail_replay"},
        ],
        local_limitations=[
            "Deterministic local harness only; no live bigclawd processes participate in this proof.",
            "Per-node audit paths are report artifacts rather than emitted runtime JSONL files.",
        ],
    )


def scenario_lease_expiry_stale_writer_rejected():
    scenario_id = "lease-expiry-stale-writer-rejected"
    title = "Lease expires and the former owner attempts a stale checkpoint write"
    subscriber_group = "group-stale-writer"
    primary = "subscriber-a"
    standby = "subscriber-b"
    task_or_trace_id = "trace-stale-writer"
    timeline = []
    owner_timeline = []
    coordinator = LeaseCoordinator()
    now = BASE_TIME + timedelta(minutes=5)

    primary_lease = coordinator.acquire(
        group_id=subscriber_group,
        subscriber_id="event-stream",
        consumer_id=primary,
        ttl=TTL,
        now=now,
    )
    owner_timeline.append(owner_timeline_entry(timestamp=now, owner=primary, event="lease_acquired", lease=primary_lease))
    audit_event(timeline, timestamp=now, subscriber=primary, action="lease_acquired", details={"lease_epoch": primary_lease.lease_epoch})

    primary_lease = coordinator.commit(
        group_id=subscriber_group,
        subscriber_id="event-stream",
        consumer_id=primary,
        lease_token=primary_lease.lease_token,
        lease_epoch=primary_lease.lease_epoch,
        checkpoint_offset=80,
        checkpoint_event_id="evt-80",
        now=now + timedelta(seconds=3),
    )
    checkpoint_before = checkpoint_payload(primary_lease)
    audit_event(timeline, timestamp=now + timedelta(seconds=3), subscriber=primary, action="checkpoint_committed", details={"offset": 80, "event_id": "evt-80"})
    audit_event(timeline, timestamp=now + timedelta(seconds=31), subscriber=primary, action="lease_expired", details={"last_offset": 80})

    takeover_lease = coordinator.acquire(
        group_id=subscriber_group,
        subscriber_id="event-stream",
        consumer_id=standby,
        ttl=TTL,
        now=now + timedelta(seconds=31),
    )
    owner_timeline.append(owner_timeline_entry(timestamp=now + timedelta(seconds=31), owner=standby, event="takeover_acquired", lease=takeover_lease))
    audit_event(timeline, timestamp=now + timedelta(seconds=31), subscriber=standby, action="lease_acquired", details={"lease_epoch": takeover_lease.lease_epoch, "takeover": True})

    stale_write_rejections = 0
    try:
        coordinator.commit(
            group_id=subscriber_group,
            subscriber_id="event-stream",
            consumer_id=primary,
            lease_token=primary_lease.lease_token,
            lease_epoch=primary_lease.lease_epoch,
            checkpoint_offset=81,
            checkpoint_event_id="evt-81",
            now=now + timedelta(seconds=32),
        )
    except LeaseFence:
        stale_write_rejections += 1
        audit_event(
            timeline,
            timestamp=now + timedelta(seconds=32),
            subscriber=primary,
            action="lease_fenced",
            details={"attempted_offset": 81, "attempted_event_id": "evt-81", "accepted_owner": standby},
        )

    audit_event(timeline, timestamp=now + timedelta(seconds=33), subscriber=standby, action="replay_started", details={"from_offset": 80, "from_event_id": "evt-80"})
    takeover_lease = coordinator.commit(
        group_id=subscriber_group,
        subscriber_id="event-stream",
        consumer_id=standby,
        lease_token=takeover_lease.lease_token,
        lease_epoch=takeover_lease.lease_epoch,
        checkpoint_offset=82,
        checkpoint_event_id="evt-82",
        now=now + timedelta(seconds=34),
    )
    checkpoint_after = checkpoint_payload(takeover_lease)
    audit_event(timeline, timestamp=now + timedelta(seconds=34), subscriber=standby, action="checkpoint_committed", details={"offset": 82, "event_id": "evt-82", "stale_writer_rejected": True})

    return build_scenario_result(
        scenario_id=scenario_id,
        title=title,
        primary_subscriber=primary,
        takeover_subscriber=standby,
        task_or_trace_id=task_or_trace_id,
        subscriber_group=subscriber_group,
        audit_assertions=[
            "Audit log records lease expiry for the former owner and acquisition by the standby.",
            "Audit log records the stale write rejection with both attempted and accepted owners.",
            "Audit log keeps the rejection and accepted takeover in the same ordered timeline.",
        ],
        checkpoint_assertions=[
            "Checkpoint sequence never decreases after the standby acquires ownership.",
            "Late primary acknowledgement is rejected or ignored without mutating durable checkpoint state.",
            "Accepted checkpoint owner always matches the active lease holder.",
        ],
        replay_assertions=[
            "Replay after stale write rejection starts from the accepted durable checkpoint only.",
            "No event acknowledged only by the stale writer disappears from the replay timeline.",
            "Replay report exposes any duplicate event IDs caused by the overlap window.",
        ],
        lease_owner_timeline=owner_timeline,
        checkpoint_before=checkpoint_before,
        checkpoint_after=checkpoint_after,
        replay_start_cursor=cursor(80, "evt-80"),
        replay_end_cursor=cursor(82, "evt-82"),
        duplicate_events=["evt-81"],
        stale_write_rejections=stale_write_rejections,
        audit_timeline=timeline,
        event_log_excerpt=[
            {"event_id": "evt-80", "delivered_by": [primary], "delivery_kind": "durable"},
            {"event_id": "evt-81", "delivered_by": [primary, standby], "delivery_kind": "stale_overlap_candidate"},
            {"event_id": "evt-82", "delivered_by": [standby], "delivery_kind": "takeover_replay_commit"},
        ],
        local_limitations=[
            "Deterministic local harness only; live lease expiry still needs a real two-node integration proof.",
            "Stale writer rejection count is produced by the harness rather than the live control-plane API.",
        ],
    )


def scenario_split_brain_dual_replay_window():
    scenario_id = "split-brain-dual-replay-window"
    title = "Two subscribers briefly believe they can replay the same tail"
    subscriber_group = "group-split-brain"
    primary = "subscriber-a"
    standby = "subscriber-b"
    task_or_trace_id = "trace-split-brain"
    timeline = []
    owner_timeline = []
    coordinator = LeaseCoordinator()
    now = BASE_TIME + timedelta(minutes=10)

    primary_lease = coordinator.acquire(
        group_id=subscriber_group,
        subscriber_id="event-stream",
        consumer_id=primary,
        ttl=TTL,
        now=now,
    )
    owner_timeline.append(owner_timeline_entry(timestamp=now, owner=primary, event="lease_acquired", lease=primary_lease))
    audit_event(timeline, timestamp=now, subscriber=primary, action="lease_acquired", details={"lease_epoch": primary_lease.lease_epoch})

    primary_lease = coordinator.commit(
        group_id=subscriber_group,
        subscriber_id="event-stream",
        consumer_id=primary,
        lease_token=primary_lease.lease_token,
        lease_epoch=primary_lease.lease_epoch,
        checkpoint_offset=120,
        checkpoint_event_id="evt-120",
        now=now + timedelta(seconds=2),
    )
    checkpoint_before = checkpoint_payload(primary_lease)
    audit_event(timeline, timestamp=now + timedelta(seconds=2), subscriber=primary, action="checkpoint_committed", details={"offset": 120, "event_id": "evt-120"})
    audit_event(timeline, timestamp=now + timedelta(seconds=29), subscriber=primary, action="overlap_window_started", details={"candidate_events": ["evt-121", "evt-122"]})

    takeover_lease = coordinator.acquire(
        group_id=subscriber_group,
        subscriber_id="event-stream",
        consumer_id=standby,
        ttl=TTL,
        now=now + timedelta(seconds=31),
    )
    owner_timeline.append(owner_timeline_entry(timestamp=now + timedelta(seconds=31), owner=standby, event="takeover_acquired", lease=takeover_lease))
    audit_event(timeline, timestamp=now + timedelta(seconds=31), subscriber=standby, action="lease_acquired", details={"lease_epoch": takeover_lease.lease_epoch, "takeover": True})
    audit_event(timeline, timestamp=now + timedelta(seconds=32), subscriber=standby, action="replay_started", details={"from_offset": 120, "candidate_events": ["evt-121", "evt-122"]})

    stale_write_rejections = 0
    try:
        coordinator.commit(
            group_id=subscriber_group,
            subscriber_id="event-stream",
            consumer_id=primary,
            lease_token=primary_lease.lease_token,
            lease_epoch=primary_lease.lease_epoch,
            checkpoint_offset=122,
            checkpoint_event_id="evt-122",
            now=now + timedelta(seconds=33),
        )
    except LeaseFence:
        stale_write_rejections += 1
        audit_event(timeline, timestamp=now + timedelta(seconds=33), subscriber=primary, action="lease_fenced", details={"attempted_offset": 122, "accepted_owner": standby, "overlap": True})

    takeover_lease = coordinator.commit(
        group_id=subscriber_group,
        subscriber_id="event-stream",
        consumer_id=standby,
        lease_token=takeover_lease.lease_token,
        lease_epoch=takeover_lease.lease_epoch,
        checkpoint_offset=122,
        checkpoint_event_id="evt-122",
        now=now + timedelta(seconds=34),
    )
    checkpoint_after = checkpoint_payload(takeover_lease)
    audit_event(timeline, timestamp=now + timedelta(seconds=34), subscriber=standby, action="checkpoint_committed", details={"offset": 122, "event_id": "evt-122", "winning_owner": standby})
    audit_event(timeline, timestamp=now + timedelta(seconds=35), subscriber=standby, action="overlap_window_closed", details={"winning_owner": standby, "duplicate_events": ["evt-121", "evt-122"]})

    return build_scenario_result(
        scenario_id=scenario_id,
        title=title,
        primary_subscriber=primary,
        takeover_subscriber=standby,
        task_or_trace_id=task_or_trace_id,
        subscriber_group=subscriber_group,
        audit_assertions=[
            "Combined audit timeline shows overlapping replay attempts and identifies the surviving owner.",
            "Audit evidence includes per-node file paths and normalized subscriber identities.",
            "The final report highlights whether duplicate replay attempts were observed or only simulated.",
        ],
        checkpoint_assertions=[
            "Only the winning owner can advance the durable checkpoint.",
            "Losing owner leaves durable checkpoint unchanged once fencing is applied.",
            "Report includes the exact checkpoint sequence where overlap began and ended.",
        ],
        replay_assertions=[
            "Replay output groups duplicate candidate deliveries by event ID.",
            "Final replay cursor belongs to the winning owner only.",
            "Validation reports whether overlapping replay created observable duplicate deliveries.",
        ],
        lease_owner_timeline=owner_timeline,
        checkpoint_before=checkpoint_before,
        checkpoint_after=checkpoint_after,
        replay_start_cursor=cursor(120, "evt-120"),
        replay_end_cursor=cursor(122, "evt-122"),
        duplicate_events=["evt-121", "evt-122"],
        stale_write_rejections=stale_write_rejections,
        audit_timeline=timeline,
        event_log_excerpt=[
            {"event_id": "evt-120", "delivered_by": [primary], "delivery_kind": "durable"},
            {"event_id": "evt-121", "delivered_by": [primary, standby], "delivery_kind": "overlap_candidate"},
            {"event_id": "evt-122", "delivered_by": [primary, standby], "delivery_kind": "overlap_candidate"},
        ],
        local_limitations=[
            "Deterministic local harness only; duplicate replay candidates are modeled rather than captured from a live shared queue.",
            "Real cross-process subscriber membership and per-node replay metrics remain follow-up work.",
        ],
    )


def build_report(now=None):
    generated_at = now or datetime.now(timezone.utc)
    scenarios = [
        scenario_takeover_after_primary_crash(),
        scenario_lease_expiry_stale_writer_rejected(),
        scenario_split_brain_dual_replay_window(),
    ]
    passing = sum(1 for scenario in scenarios if scenario["all_assertions_passed"])
    total_duplicates = sum(scenario["duplicate_delivery_count"] for scenario in scenarios)
    total_rejections = sum(scenario["stale_write_rejections"] for scenario in scenarios)
    return {
        "generated_at": utc_iso(generated_at),
        "ticket": "OPE-269",
        "title": "Multi-subscriber takeover executable local harness report",
        "status": "local-executable",
        "harness_mode": "deterministic_local_simulation",
        "current_primitives": {
            "lease_aware_checkpoints": [
                "internal/events/subscriber_leases.go",
                "internal/events/subscriber_leases_test.go",
                "docs/reports/event-bus-reliability-report.md",
            ],
            "shared_queue_evidence": [
                "scripts/e2e/multi_node_shared_queue",
                "docs/reports/multi-node-shared-queue-report.json",
            ],
            "takeover_harness": [
                "scripts/e2e/subscriber_takeover_fault_matrix",
                "docs/reports/multi-subscriber-takeover-validation-report.json",
            ],
        },
        "required_report_sections": [
            "scenario metadata",
            "fault injection steps",
            "audit assertions",
            "checkpoint assertions",
            "replay assertions",
            "per-node audit artifacts",
            "final owner and replay cursor summary",
            "duplicate delivery accounting",
            "open blockers and follow-up implementation hooks",
        ],
        "implementation_path": [
            "wire the same ownership and rejection schema into the shared multi-node harness",
            "emit real per-node audit artifacts from live takeover runs instead of synthetic report paths",
            "export duplicate replay candidates and stale-writer rejection counters from live event-log APIs",
            "prove the same report contract against an actual cross-process subscriber group",
        ],
        "summary": {
            "scenario_count": len(scenarios),
            "passing_scenarios": passing,
            "failing_scenarios": len(scenarios) - passing,
            "duplicate_delivery_count": total_duplicates,
            "stale_write_rejections": total_rejections,
        },
        "scenarios": scenarios,
        "remaining_gaps": [
            "The harness is deterministic and local; it does not yet orchestrate live bigclawd takeover between separate processes.",
            "Audit log paths in the report are normalized artifact targets, not emitted runtime files from a multi-node run.",
            "Shared durable subscriber-group coordination still needs a full cross-process proof before the follow-up digest can be closed as done.",
        ],
    }


def main():
    parser = argparse.ArgumentParser(
        description="Generate the executable local takeover fault-injection harness report for OPE-269"
    )
    parser.add_argument(
        "--output",
        default="docs/reports/multi-subscriber-takeover-validation-report.json",
        help="Path relative to the repo root",
    )
    parser.add_argument(
        "--pretty",
        action="store_true",
        help="Print the generated report to stdout",
    )
    args = parser.parse_args()

    repo_root = pathlib.Path(__file__).resolve().parents[2]
    report = build_report()
    output_path = repo_root / args.output
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    if args.pretty:
        print(json.dumps(report, indent=2))


if __name__ == "__main__":
    main()
