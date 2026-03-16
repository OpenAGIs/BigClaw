#!/usr/bin/env python3
import argparse
import json
import pathlib
from datetime import datetime, timezone


def utc_iso(now=None):
    moment = now or datetime.now(timezone.utc)
    return moment.isoformat().replace('+00:00', 'Z')


def load_json(path):
    return json.loads(pathlib.Path(path).read_text(encoding='utf-8'))


def resolve_repo_path(repo_root, path):
    candidate = pathlib.Path(path)
    if candidate.is_absolute():
        return candidate
    return repo_root / candidate


def capability_row(
    *,
    capability,
    current_state,
    live_local_proof,
    deterministic_local_harness,
    contract_defined_target,
    notes,
):
    return {
        'capability': capability,
        'current_state': current_state,
        'live_local_proof': live_local_proof,
        'deterministic_local_harness': deterministic_local_harness,
        'contract_defined_target': contract_defined_target,
        'notes': notes,
    }


def build_report(
    multi_node_report_path='bigclaw-go/docs/reports/multi-node-shared-queue-report.json',
    takeover_report_path='bigclaw-go/docs/reports/multi-subscriber-takeover-validation-report.json',
):
    repo_root = pathlib.Path(__file__).resolve().parents[3]
    multi_node = load_json(resolve_repo_path(repo_root, multi_node_report_path))
    takeover = load_json(resolve_repo_path(repo_root, takeover_report_path))

    summary = {
        'shared_queue_total_tasks': multi_node['count'],
        'shared_queue_cross_node_completions': multi_node['cross_node_completions'],
        'shared_queue_duplicate_completed_tasks': len(multi_node['duplicate_completed_tasks']),
        'shared_queue_duplicate_started_tasks': len(multi_node['duplicate_started_tasks']),
        'takeover_scenario_count': takeover['summary']['scenario_count'],
        'takeover_passing_scenarios': takeover['summary']['passing_scenarios'],
        'takeover_duplicate_delivery_count': takeover['summary']['duplicate_delivery_count'],
        'takeover_stale_write_rejections': takeover['summary']['stale_write_rejections'],
    }

    capabilities = [
        capability_row(
            capability='shared_queue_task_coordination',
            current_state='implemented',
            live_local_proof=True,
            deterministic_local_harness=False,
            contract_defined_target=True,
            notes=[
                'Two independent bigclawd processes share one SQLite-backed queue without duplicate terminal execution.',
                'Current proof is local and SQLite-backed rather than broker-backed or replicated.',
            ],
        ),
        capability_row(
            capability='subscriber_takeover_semantics',
            current_state='local_executable_only',
            live_local_proof=False,
            deterministic_local_harness=True,
            contract_defined_target=True,
            notes=[
                'Lease handoff, stale-writer fencing, and duplicate replay accounting are covered by a deterministic local harness.',
                'The same schema is not yet emitted by a live multi-node run.',
            ],
        ),
        capability_row(
            capability='cross_process_replay_coordination',
            current_state='contract_defined',
            live_local_proof=False,
            deterministic_local_harness=True,
            contract_defined_target=True,
            notes=[
                'Replay cursor and checkpoint expectations are codified across the local takeover harness and durability rollout contract.',
                'No broker-backed or partitioned live replay proof exists yet.',
            ],
        ),
        capability_row(
            capability='stale_writer_fencing',
            current_state='local_executable_only',
            live_local_proof=False,
            deterministic_local_harness=True,
            contract_defined_target=True,
            notes=[
                'The local takeover harness proves stale checkpoint writers are rejected after ownership transfer.',
                'A live multi-process subscriber group still needs to emit the same rejection evidence.',
            ],
        ),
        capability_row(
            capability='partitioned_topic_routing',
            current_state='not_available',
            live_local_proof=False,
            deterministic_local_harness=False,
            contract_defined_target=True,
            notes=[
                'No partitioned topic model exists yet in the runtime.',
                'Broker-backed target docs reserve partition or quorum log ordering as the future coordination scope.',
            ],
        ),
        capability_row(
            capability='broker_backed_subscriber_ownership',
            current_state='not_available',
            live_local_proof=False,
            deterministic_local_harness=False,
            contract_defined_target=True,
            notes=[
                'No broker-backed cross-process subscriber ownership model exists yet.',
                'The durability rollout contract defines the expected checkpoint fencing and failover guarantees before rollout-safe claims.',
            ],
        ),
        capability_row(
            capability='operator_capability_surface',
            current_state='implemented',
            live_local_proof=False,
            deterministic_local_harness=False,
            contract_defined_target=True,
            notes=[
                'The repo exposes provider-neutral durability and rollout metadata through docs and runtime-facing event_durability surfaces.',
                'This report adds a coordination-specific surface tying together live local proof, deterministic local harnesses, and future targets.',
            ],
        ),
    ]

    current_ceiling = [
        'no partitioned topic model',
        'no broker-backed cross-process subscriber coordination',
        'no live multi-node subscriber takeover proof',
    ]

    next_hooks = [
        'wire the takeover harness schema into scripts/e2e/multi_node_shared_queue.py output',
        'attach real per-node audit artifacts to live multi-process takeover runs',
        'back subscriber ownership with a shared durable backend instead of process-local lease state',
        'validate broker-backed replay and ownership semantics against the same report schema',
    ]

    return {
        'generated_at': utc_iso(),
        'ticket': 'BIG-PAR-085-local-prework',
        'title': 'Cross-process coordination capability surface',
        'status': 'local-capability-surface',
        'evidence_inputs': {
            'shared_queue_report': multi_node_report_path,
            'takeover_harness_report': takeover_report_path,
            'supporting_docs': [
                'bigclaw-go/docs/reports/event-bus-reliability-report.md',
                'bigclaw-go/docs/reports/replicated-event-log-durability-rollout-contract.md',
                'bigclaw-go/docs/reports/broker-event-log-adapter-contract.md',
            ],
        },
        'summary': summary,
        'capabilities': capabilities,
        'current_ceiling': current_ceiling,
        'next_runtime_hooks': next_hooks,
    }


def main():
    parser = argparse.ArgumentParser(description='Generate the cross-process coordination capability surface report')
    parser.add_argument('--output', default='bigclaw-go/docs/reports/cross-process-coordination-capability-surface.json')
    parser.add_argument('--pretty', action='store_true')
    args = parser.parse_args()

    repo_root = pathlib.Path(__file__).resolve().parents[3]
    report = build_report()
    output_path = repo_root / args.output
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(json.dumps(report, indent=2) + '\n', encoding='utf-8')
    if args.pretty:
        print(json.dumps(report, indent=2))


if __name__ == '__main__':
    main()
