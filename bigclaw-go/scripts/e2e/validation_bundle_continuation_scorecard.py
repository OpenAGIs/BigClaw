#!/usr/bin/env python3
import argparse
import json
import pathlib
from datetime import datetime, timezone


EXECUTOR_LANES = ('local', 'kubernetes', 'ray')


def utc_iso(moment=None):
    value = moment or datetime.now(timezone.utc)
    return value.isoformat().replace('+00:00', 'Z')


def parse_time(value):
    return datetime.fromisoformat(str(value).replace('Z', '+00:00'))


def load_json(path):
    return json.loads(path.read_text(encoding='utf-8'))


def resolve_repo_path(repo_root, path):
    candidate = pathlib.Path(path)
    if candidate.is_absolute():
        return candidate
    return repo_root / candidate


def resolve_evidence_path(repo_root, bigclaw_go_root, path):
    candidate = pathlib.Path(path)
    if candidate.is_absolute():
        return candidate

    search_roots = [repo_root]
    if candidate.parts[:1] != ('bigclaw-go',):
        search_roots.append(bigclaw_go_root)

    for root in search_roots:
        resolved = root / candidate
        if resolved.exists():
            return resolved
    return search_roots[0] / candidate


def consecutive_successes(statuses):
    count = 0
    for status in statuses:
        if status == 'succeeded':
            count += 1
        else:
            break
    return count


def build_lane_scorecard(runs, lane):
    statuses = []
    enabled_runs = 0
    succeeded_runs = 0
    for run in runs:
        section = run.get(lane, {}) if isinstance(run, dict) else {}
        enabled = bool(section.get('enabled'))
        status = section.get('status', 'missing') if enabled else 'disabled'
        statuses.append(status)
        if enabled:
            enabled_runs += 1
        if status == 'succeeded':
            succeeded_runs += 1
    latest = runs[0].get(lane, {}) if runs else {}
    return {
        'lane': lane,
        'latest_enabled': bool(latest.get('enabled')),
        'latest_status': latest.get('status', 'missing') if latest else 'missing',
        'recent_statuses': statuses,
        'enabled_runs': enabled_runs,
        'succeeded_runs': succeeded_runs,
        'consecutive_successes': consecutive_successes(statuses),
        'all_recent_runs_succeeded': enabled_runs > 0 and enabled_runs == succeeded_runs,
    }


def check(name, passed, detail):
    return {'name': name, 'passed': passed, 'detail': detail}


def build_report(
    index_manifest_path='bigclaw-go/docs/reports/live-validation-index.json',
    bundle_root_path='bigclaw-go/docs/reports/live-validation-runs',
    summary_path='bigclaw-go/docs/reports/live-validation-summary.json',
    shared_queue_report_path='bigclaw-go/docs/reports/multi-node-shared-queue-report.json',
):
    repo_root = pathlib.Path(__file__).resolve().parents[3]
    bigclaw_go_root = repo_root / 'bigclaw-go'
    manifest = load_json(resolve_repo_path(repo_root, index_manifest_path))
    latest = manifest['latest']
    recent_runs_meta = manifest.get('recent_runs', [])
    recent_runs = []
    recent_run_inputs = []
    for item in recent_runs_meta:
        summary_file = resolve_evidence_path(repo_root, bigclaw_go_root, item['summary_path'])
        recent_runs.append(load_json(summary_file))
        recent_run_inputs.append(str(summary_file.relative_to(repo_root)))

    latest_summary = load_json(resolve_repo_path(repo_root, summary_path))
    shared_queue = load_json(resolve_repo_path(repo_root, shared_queue_report_path))
    bundle_root = resolve_repo_path(repo_root, bundle_root_path)

    lane_scorecards = [build_lane_scorecard(recent_runs, lane) for lane in EXECUTOR_LANES]
    latest_generated_at = parse_time(latest['generated_at'])
    previous_generated_at = parse_time(recent_runs[1]['generated_at']) if len(recent_runs) > 1 else None
    generated_at = datetime.now(timezone.utc)
    latest_age_hours = round((generated_at - latest_generated_at).total_seconds() / 3600, 2)
    bundle_gap_minutes = None
    if previous_generated_at is not None:
        bundle_gap_minutes = round((latest_generated_at - previous_generated_at).total_seconds() / 60, 2)

    latest_lane_statuses = {lane: latest_summary[lane]['status'] for lane in EXECUTOR_LANES}
    latest_all_succeeded = all(status == 'succeeded' for status in latest_lane_statuses.values())
    recent_all_succeeded = all(run.get('status') == 'succeeded' for run in recent_runs)
    repeated_lane_coverage = all(item['enabled_runs'] >= 2 for item in lane_scorecards)
    enabled_runs_by_lane = {item['lane']: item['enabled_runs'] for item in lane_scorecards}

    continuation_checks = [
        check(
            'latest_bundle_all_executor_tracks_succeeded',
            latest_all_succeeded,
            f"latest lane statuses={latest_lane_statuses}",
        ),
        check(
            'recent_bundle_chain_has_multiple_runs',
            len(recent_runs) >= 2,
            f"recent bundle count={len(recent_runs)}",
        ),
        check(
            'recent_bundle_chain_has_no_failures',
            recent_all_succeeded,
            f"recent bundle statuses={[run.get('status', 'unknown') for run in recent_runs]}",
        ),
        check(
            'all_executor_tracks_have_repeated_recent_coverage',
            repeated_lane_coverage,
            f"enabled_runs_by_lane={enabled_runs_by_lane}",
        ),
        check(
            'shared_queue_companion_proof_available',
            bool(shared_queue.get('all_ok')),
            f"cross_node_completions={shared_queue.get('cross_node_completions')}",
        ),
        check(
            'continuation_surface_is_workflow_triggered',
            True,
            'run_all closeout now refreshes the scorecard and gate automatically, but continuation still depends on explicit workflow execution instead of an always-on service',
        ),
    ]

    shared_queue_companion = {
        'available': bool(shared_queue.get('all_ok')),
        'report_path': shared_queue_report_path,
        'cross_node_completions': shared_queue.get('cross_node_completions', 0),
        'duplicate_completed_tasks': len(shared_queue.get('duplicate_completed_tasks', [])),
        'duplicate_started_tasks': len(shared_queue.get('duplicate_started_tasks', [])),
        'mode': 'standalone-proof',
    }

    current_ceiling = [
        'continuation across future validation bundles remains workflow-triggered',
        'shared-queue coordination proof still lives outside the canonical live validation bundle',
        'recent history is bounded to the exported bundle index and not an always-on service',
    ]
    if not repeated_lane_coverage:
        current_ceiling.append('not every executor lane is enabled across every indexed bundle in the current recent window')

    next_runtime_hooks = [
        'enable BIGCLAW_E2E_ENFORCE_CONTINUATION_GATE=1 in workflow closeout when continuation holds should fail the run',
        'fold shared-queue coordination proof into the same bundle lineage or adjacent bundle metadata',
        'extend the automatic continuation refresh beyond run_all.sh into broader workflow orchestrators',
        'extend the scorecard beyond the latest recent_runs window when more longitudinal evidence exists',
    ]

    return {
        'generated_at': utc_iso(generated_at),
        'ticket': 'BIG-PAR-086-local-prework',
        'title': 'Validation bundle continuation scorecard',
        'status': 'local-continuation-scorecard',
        'evidence_inputs': {
            'manifest_path': index_manifest_path,
            'latest_summary_path': summary_path,
            'bundle_root': bundle_root_path,
            'recent_run_summaries': recent_run_inputs,
            'shared_queue_report_path': shared_queue_report_path,
            'generator_script': 'bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py',
        },
        'summary': {
            'recent_bundle_count': len(recent_runs),
            'latest_run_id': latest['run_id'],
            'latest_status': latest['status'],
            'latest_bundle_age_hours': latest_age_hours,
            'latest_all_executor_tracks_succeeded': latest_all_succeeded,
            'recent_bundle_chain_has_no_failures': recent_all_succeeded,
            'all_executor_tracks_have_repeated_recent_coverage': repeated_lane_coverage,
            'bundle_gap_minutes': bundle_gap_minutes,
            'bundle_root_exists': bundle_root.exists(),
        },
        'executor_lanes': lane_scorecards,
        'shared_queue_companion': shared_queue_companion,
        'continuation_checks': continuation_checks,
        'current_ceiling': current_ceiling,
        'next_runtime_hooks': next_runtime_hooks,
    }


def main():
    parser = argparse.ArgumentParser(description='Generate the validation bundle continuation scorecard')
    parser.add_argument('--output', default='bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json')
    parser.add_argument('--pretty', action='store_true')
    args = parser.parse_args()

    repo_root = pathlib.Path(__file__).resolve().parents[3]
    report = build_report()
    output_path = resolve_repo_path(repo_root, args.output)
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(json.dumps(report, indent=2) + '\n', encoding='utf-8')
    if args.pretty:
        print(json.dumps(report, indent=2))


if __name__ == '__main__':
    main()
