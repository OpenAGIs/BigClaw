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
    runtime_readiness,
    live_local_proof,
    deterministic_local_harness,
    contract_defined_target,
    notes,
):
    return {
        'capability': capability,
        'current_state': current_state,
        'runtime_readiness': runtime_readiness,
        'live_local_proof': live_local_proof,
        'deterministic_local_harness': deterministic_local_harness,
        'contract_defined_target': contract_defined_target,
        'notes': notes,
    }


def target_contract_row(
    *,
    capability,
    contract_anchor,
    runtime_status,
    partitioning=None,
    ownership=None,
    guarantees=None,
):
    return {
        'capability': capability,
        'contract_anchor': contract_anchor,
        'runtime_status': runtime_status,
        'partitioning': partitioning or {},
        'ownership': ownership or {},
        'guarantees': guarantees or [],
    }


def build_report(
    multi_node_report_path='bigclaw-go/docs/reports/multi-node-shared-queue-report.json',
    takeover_report_path='bigclaw-go/docs/reports/multi-subscriber-takeover-validation-report.json',
    live_takeover_report_path='bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-report.json',
):
    repo_root = pathlib.Path(__file__).resolve().parents[3]
    multi_node = load_json(resolve_repo_path(repo_root, multi_node_report_path))
    takeover = load_json(resolve_repo_path(repo_root, takeover_report_path))
    live_takeover = load_json(resolve_repo_path(repo_root, live_takeover_report_path))

    summary = {
        'shared_queue_total_tasks': multi_node['count'],
        'shared_queue_cross_node_completions': multi_node['cross_node_completions'],
        'shared_queue_duplicate_completed_tasks': len(multi_node['duplicate_completed_tasks']),
        'shared_queue_duplicate_started_tasks': len(multi_node['duplicate_started_tasks']),
        'takeover_scenario_count': takeover['summary']['scenario_count'],
        'takeover_passing_scenarios': takeover['summary']['passing_scenarios'],
        'takeover_duplicate_delivery_count': takeover['summary']['duplicate_delivery_count'],
        'takeover_stale_write_rejections': takeover['summary']['stale_write_rejections'],
        'live_takeover_scenario_count': live_takeover['summary']['scenario_count'],
        'live_takeover_passing_scenarios': live_takeover['summary']['passing_scenarios'],
        'live_takeover_stale_write_rejections': live_takeover['summary']['stale_write_rejections'],
    }

    target_contracts = [
        target_contract_row(
            capability='partitioned_topic_routing',
            contract_anchor='events.SubscriptionRequest.PartitionRoute',
            runtime_status='contract_only',
            partitioning={
                'topic': 'provider-defined shared event stream',
                'supported_partition_keys': ['trace_id', 'task_id', 'event_type'],
                'ordering_scope': 'sequence remains portable within the selected partition route',
                'filter_alignment': 'ReplayRequest task_id/trace_id filters must remain valid when a backend introduces partition routing.',
            },
            guarantees=[
                'Partition keys are provider-neutral and map to existing trace/task/event_type selectors.',
                'Partition metadata may vary by backend, but portable replay ordering still uses Position.Sequence.',
                'No runtime implementation is shipped yet; this row defines the future adapter contract only.',
            ],
        ),
        target_contract_row(
            capability='broker_backed_subscriber_ownership',
            contract_anchor='events.SubscriptionRequest.OwnershipContract',
            runtime_status='contract_only',
            ownership={
                'subscriber_group': 'shared durable consumer identity',
                'mode': 'exclusive',
                'lease_fields': ['epoch', 'lease_token'],
                'partition_hints': 'optional partition affinity for future broker-backed consumers',
            },
            guarantees=[
                'Checkpoint commits remain fenced by epoch plus lease token after ownership transfer.',
                'Ownership metadata travels through the neutral subscription contract instead of provider-specific APIs.',
                'No broker-backed runtime implementation is shipped yet; this row defines the future ownership contract only.',
            ],
        ),
    ]

    capabilities = [
        capability_row(
            capability='shared_queue_task_coordination',
            current_state='implemented',
            runtime_readiness='live_proven',
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
            current_state='implemented_with_process_local_boundary',
            runtime_readiness='harness_proven',
            live_local_proof=True,
            deterministic_local_harness=True,
            contract_defined_target=True,
            notes=[
                'Lease handoff and duplicate replay accounting are exercised by both the deterministic harness and the live two-node companion proof.',
                'Runtime ownership remains bounded by process-local coordination, so the provider-neutral broker-backed ownership contract is still not runtime-proven.',
            ],
        ),
        capability_row(
            capability='cross_process_replay_coordination',
            current_state='contract_defined',
            runtime_readiness='harness_proven',
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
            current_state='implemented_with_process_local_boundary',
            runtime_readiness='harness_proven',
            live_local_proof=True,
            deterministic_local_harness=True,
            contract_defined_target=True,
            notes=[
                'The local takeover harness proves stale checkpoint writers are fenced after ownership transfer.',
                'Shared durable subscriber ownership is still missing, so the broker-backed ownership contract remains contract-only even though local proof artifacts exist.',
            ],
        ),
        capability_row(
            capability='partitioned_topic_routing',
            current_state='not_available',
            runtime_readiness='contract_only',
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
            runtime_readiness='contract_only',
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
            runtime_readiness='supporting_surface',
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
        'no shared durable subscriber ownership backend',
    ]

    next_hooks = [
        'back subscriber ownership with a shared durable backend instead of process-local lease state',
        'emit native takeover transition audit events from the runtime instead of harness-authored artifacts',
        'validate broker-backed replay and ownership semantics against the same report schema',
    ]

    return {
        'generated_at': utc_iso(),
        'ticket': 'BIG-PAR-085-local-prework',
        'title': 'Cross-process coordination capability surface',
        'status': 'local-capability-surface',
        'target_contract_surface_version': '2026-03-17',
        'runtime_readiness_levels': {
            'live_proven': 'Shipped runtime behavior with checked-in live cross-process proof.',
            'harness_proven': 'Deterministic executable harness coverage exists, but no live multi-node proof is checked in.',
            'contract_only': 'Only target contracts or rollout docs define the expected semantics today.',
            'supporting_surface': 'The repo exposes reporting or metadata surfaces that describe runtime readiness without proving the coordination behavior itself.',
        },
        'evidence_inputs': {
            'shared_queue_report': multi_node_report_path,
            'takeover_harness_report': takeover_report_path,
            'live_takeover_report': live_takeover_report_path,
            'supporting_docs': [
                'bigclaw-go/docs/reports/event-bus-reliability-report.md',
                'bigclaw-go/docs/reports/replicated-event-log-durability-rollout-contract.md',
                'bigclaw-go/docs/reports/broker-event-log-adapter-contract.md',
            ],
        },
        'target_contracts': target_contracts,
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
