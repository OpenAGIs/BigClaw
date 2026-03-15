#!/usr/bin/env python3
import argparse
import json
import pathlib
import time


def scenario(
    scenario_id,
    title,
    fault,
    current_evidence,
    audit_assertions,
    checkpoint_assertions,
    replay_assertions,
    harness_outputs,
    blockers,
):
    return {
        "id": scenario_id,
        "title": title,
        "fault": fault,
        "current_evidence": current_evidence,
        "audit_assertions": audit_assertions,
        "checkpoint_assertions": checkpoint_assertions,
        "replay_assertions": replay_assertions,
        "minimum_harness_outputs": harness_outputs,
        "implementation_blockers": blockers,
    }


def build_report():
    harness_outputs = [
        "scenario_id",
        "subscriber_group",
        "primary_subscriber",
        "takeover_subscriber",
        "task_or_trace_id",
        "lease_owner_timeline",
        "checkpoint_before",
        "checkpoint_after",
        "replay_start_cursor",
        "replay_end_cursor",
        "duplicate_delivery_count",
        "stale_write_rejections",
        "audit_log_paths",
        "event_log_excerpt",
        "all_assertions_passed",
    ]
    scenarios = [
        scenario(
            "takeover-after-primary-crash",
            "Primary subscriber crashes after processing but before checkpoint flush",
            {
                "inject": [
                    "start two subscribers against one subscriber group",
                    "let primary process at least one replayable event",
                    "terminate primary before checkpoint acknowledgement is persisted",
                    "allow standby subscriber to acquire ownership",
                ],
                "target_gap": "prove takeover replays the uncheckpointed tail without losing or double-committing progress",
            },
            [
                "Current shared-SQLite evidence proves cross-node completion without duplicate terminal work.",
                "Current event replay evidence proves replay-first subscription ordering inside one process.",
            ],
            [
                "Audit log shows one ownership handoff from primary to standby.",
                "Audit log records the primary interruption reason before standby completion.",
                "Audit log links takeover to the same task or trace identifier across both subscribers.",
            ],
            [
                "Checkpoint after takeover is greater than or equal to the last durable checkpoint from the primary.",
                "Standby checkpoint commit is attributed to the new lease owner.",
                "No checkpoint update is accepted from the crashed primary after takeover.",
            ],
            [
                "Replay resumes from the last durable checkpoint, not from the last in-memory event processed by the crashed primary.",
                "At most one duplicate delivery is tolerated for the uncheckpointed tail and it is visible in the report.",
                "Replay window closes once the standby checkpoint advances past the tail.",
            ],
            harness_outputs,
            [
                "Subscriber-group checkpoint leases are not implemented yet.",
                "No audit event schema currently records subscriber-group ownership transfers.",
            ],
        ),
        scenario(
            "lease-expiry-stale-writer-rejected",
            "Lease expires and the former owner attempts a stale checkpoint write",
            {
                "inject": [
                    "start primary subscriber and allow it to own the checkpoint lease",
                    "block renewals or pause the primary long enough to expire its lease",
                    "let standby subscriber acquire the replacement lease",
                    "resume the primary and force a late checkpoint acknowledgement",
                ],
                "target_gap": "prove stale writers cannot move a subscriber-group checkpoint backwards after takeover",
            },
            [
                "Current monotonic checkpoint work covers stale acknowledgement rejection for one logical subscriber.",
                "Current lease recovery evidence proves expired leases can be reacquired by another worker.",
            ],
            [
                "Audit log records lease expiry for the former owner and acquisition by the standby.",
                "Audit log records the stale write rejection with both attempted and accepted owners.",
                "Audit log keeps the rejection and accepted takeover in the same ordered timeline.",
            ],
            [
                "Checkpoint sequence never decreases after the standby acquires ownership.",
                "Late primary acknowledgement is rejected or ignored without mutating durable checkpoint state.",
                "Accepted checkpoint owner always matches the active lease holder.",
            ],
            [
                "Replay after stale write rejection starts from the accepted durable checkpoint only.",
                "No event acknowledged only by the stale writer disappears from the replay timeline.",
                "Replay report exposes any duplicate event IDs caused by the overlap window.",
            ],
            harness_outputs,
            [
                "Checkpoint ownership is not fenced by lease metadata yet.",
                "No current API/report payload exposes stale checkpoint rejection counts.",
            ],
        ),
        scenario(
            "split-brain-dual-replay-window",
            "Two subscribers briefly believe they can replay the same tail",
            {
                "inject": [
                    "start a primary subscriber and a standby subscriber with one shared subscriber group",
                    "introduce a control-path delay so both nodes believe takeover is permitted",
                    "let both nodes attempt replay of the same tail window",
                    "restore coordination and keep only one active owner",
                ],
                "target_gap": "prove audit evidence is strong enough to diagnose duplicate replay attempts and the winning lease owner",
            },
            [
                "Current event bus evidence is process-local and does not prove cross-node subscriber fencing.",
                "Current multi-node queue proof already aggregates audit files from multiple nodes into one report.",
            ],
            [
                "Combined audit timeline shows overlapping replay attempts and identifies the surviving owner.",
                "Audit evidence includes per-node file paths and normalized subscriber identities.",
                "The final report highlights whether duplicate replay attempts were observed or only simulated.",
            ],
            [
                "Only the winning owner can advance the durable checkpoint.",
                "Losing owner leaves durable checkpoint unchanged once fencing is applied.",
                "Report includes the exact checkpoint sequence where overlap began and ended.",
            ],
            [
                "Replay output groups duplicate candidate deliveries by event ID.",
                "Final replay cursor belongs to the winning owner only.",
                "Validation reports whether overlapping replay created observable duplicate deliveries.",
            ],
            harness_outputs,
            [
                "No subscriber-group membership or lease coordinator exists yet.",
                "Replay reports do not currently aggregate duplicate candidates by event ID across nodes.",
            ],
        ),
    ]
    return {
        "generated_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "ticket": "OPE-217",
        "title": "Multi-subscriber takeover fault-injection validation matrix",
        "status": "planning-ready",
        "current_primitives": {
            "event_replay": [
                "internal/events/bus.go",
                "internal/events/bus_test.go",
                "docs/reports/event-bus-reliability-report.md",
            ],
            "cross_node_audit_aggregation": [
                "scripts/e2e/multi_node_shared_queue.py",
                "docs/reports/multi-node-shared-queue-report.json",
            ],
            "lease_recovery": [
                "docs/reports/lease-recovery-report.md",
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
            "add subscriber-group lease ownership metadata to checkpoint writes",
            "emit normalized audit events for lease acquire, renew, expire, reject, and takeover",
            "extend the shared multi-node e2e harness with takeover-specific fault injection toggles",
            "export grouped replay duplicates and stale-writer rejection counters in the generated report",
        ],
        "scenarios": scenarios,
    }


def main():
    parser = argparse.ArgumentParser(
        description="Generate the canonical takeover fault-injection matrix for OPE-217"
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
