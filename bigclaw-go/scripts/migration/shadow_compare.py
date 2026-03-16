#!/usr/bin/env python3
import argparse
import datetime
import json
import pathlib
import sys
import time
import urllib.request


def request_json(base_url, method, path, payload=None):
    data = None if payload is None else json.dumps(payload).encode('utf-8')
    req = urllib.request.Request(base_url.rstrip('/') + path, data=data, method=method, headers={'Content-Type': 'application/json'})
    with urllib.request.urlopen(req, timeout=30) as resp:
        return json.loads(resp.read().decode('utf-8'))


def wait_health(base_url, timeout_seconds):
    deadline = time.time() + timeout_seconds
    last_error = None
    while time.time() < deadline:
        try:
            payload = request_json(base_url, 'GET', '/healthz')
            if payload.get('ok'):
                return
        except Exception as exc:
            last_error = exc
        time.sleep(1)
    raise TimeoutError(f'timeout waiting for health on {base_url}: {last_error}')


def wait_terminal(base_url, task_id, timeout_seconds):
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        status = request_json(base_url, 'GET', f'/tasks/{task_id}')
        if status['state'] in {'succeeded', 'dead_letter', 'cancelled', 'failed'}:
            return status
        time.sleep(1)
    raise TimeoutError(f'timeout waiting for {task_id} on {base_url}')


def event_types(events):
    return [event.get('type') for event in events]


def timeline_seconds(events):
    if len(events) < 2:
        return 0
    start = events[0].get('timestamp')
    end = events[-1].get('timestamp')
    if not start or not end:
        return 0
    try:
        start_ts = datetime.datetime.fromisoformat(start.replace('Z', '+00:00'))
        end_ts = datetime.datetime.fromisoformat(end.replace('Z', '+00:00'))
    except ValueError:
        return 0
    return max((end_ts - start_ts).total_seconds(), 0)


def write_json(path, payload):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')


def relpath(path, root):
    try:
        return str(path.relative_to(root))
    except ValueError:
        return str(path)


def comparison_matched(comparison):
    diff = comparison.get('diff', {})
    return diff.get('state_equal') and diff.get('event_types_equal')


def comparison_summary(comparison):
    diff = comparison.get('diff', {})
    primary = comparison.get('primary', {})
    shadow = comparison.get('shadow', {})
    primary_status = primary.get('status', {})
    shadow_status = shadow.get('status', {})
    return {
        'trace_id': comparison.get('trace_id', ''),
        'primary_task_id': primary.get('task_id', ''),
        'shadow_task_id': shadow.get('task_id', ''),
        'primary_state': primary_status.get('state', 'unknown'),
        'shadow_state': shadow_status.get('state', 'unknown'),
        'primary_event_count': len(primary.get('events', [])),
        'shadow_event_count': len(shadow.get('events', [])),
        'event_count_delta': diff.get('event_count_delta', 0),
        'event_types_equal': bool(diff.get('event_types_equal')),
        'state_equal': bool(diff.get('state_equal')),
        'matched': comparison_matched(comparison),
    }


def build_bundle_report(*, report_kind, comparisons, task_sources=None, closeout_commands=None, legacy_single_result=None):
    matched = sum(1 for comparison in comparisons if comparison_matched(comparison))
    report = {
        'artifact_type': report_kind,
        'generated_at': datetime.datetime.now(datetime.timezone.utc).isoformat(),
        'status': 'matched' if matched == len(comparisons) else 'mismatched',
        'bundle_path': '',
        'summary_path': '',
        'report_path': '',
        'bundle_report_path': '',
        'comparison_totals': {
            'total': len(comparisons),
            'matched': matched,
            'mismatched': len(comparisons) - matched,
        },
        'task_sources': list(task_sources or []),
        'closeout_commands': list(
            closeout_commands
            or [
                'cd bigclaw-go && python3 -m unittest scripts.migration.test_shadow_reports',
                'cd bigclaw-go && python3 scripts/migration/shadow_compare.py --primary <url> --shadow <url> --task-file <task.json> --report-path docs/reports/shadow-compare-report.json',
                'cd bigclaw-go && python3 scripts/migration/shadow_matrix.py --primary <url> --shadow <url> --task-file <task.json> --report-path docs/reports/shadow-matrix-report.json',
            ]
        ),
        'comparison_summaries': [comparison_summary(comparison) for comparison in comparisons],
        'comparisons': comparisons,
    }
    if legacy_single_result is not None:
        report['comparison'] = legacy_single_result
        report.update(legacy_single_result)
    return report


def build_bundle_summary(report):
    return {
        'artifact_type': report['artifact_type'],
        'generated_at': report['generated_at'],
        'status': report['status'],
        'bundle_path': report['bundle_path'],
        'summary_path': report['summary_path'],
        'report_path': report['report_path'],
        'bundle_report_path': report['bundle_report_path'],
        'comparison_totals': report['comparison_totals'],
        'task_sources': report['task_sources'],
        'closeout_commands': report['closeout_commands'],
        'comparison_summaries': report['comparison_summaries'],
    }


def write_bundle_artifacts(report, *, report_path, bundle_dir=None, bundle_summary_name=None):
    root = pathlib.Path(__file__).resolve().parents[2].resolve()
    canonical_report_path = pathlib.Path(report_path)
    if not canonical_report_path.is_absolute():
        canonical_report_path = root / canonical_report_path
    canonical_report_path = canonical_report_path.resolve()
    resolved_bundle_dir = pathlib.Path(bundle_dir) if bundle_dir else canonical_report_path.parent / 'migration-shadow-bundle'
    if not resolved_bundle_dir.is_absolute():
        resolved_bundle_dir = root / resolved_bundle_dir
    resolved_bundle_dir = resolved_bundle_dir.resolve()
    summary_name = bundle_summary_name or f"{report['artifact_type']}-summary.json"
    bundle_report_path = resolved_bundle_dir / canonical_report_path.name
    bundle_summary_path = resolved_bundle_dir / summary_name

    report['bundle_path'] = relpath(resolved_bundle_dir, root)
    report['summary_path'] = relpath(bundle_summary_path, root)
    report['report_path'] = relpath(canonical_report_path, root)
    report['bundle_report_path'] = relpath(bundle_report_path, root)

    write_json(bundle_report_path, report)
    write_json(canonical_report_path, report)
    write_json(bundle_summary_path, build_bundle_summary(report))


def compare_task(primary_base_url, shadow_base_url, task, timeout_seconds, health_timeout_seconds):
    wait_health(primary_base_url, health_timeout_seconds)
    wait_health(shadow_base_url, health_timeout_seconds)

    primary_task = dict(task)
    shadow_task = dict(task)
    base_id = task.get('id', f'shadow-{int(time.time())}')
    trace_id = task.get('trace_id', base_id)
    primary_task['id'] = base_id + '-primary'
    shadow_task['id'] = base_id + '-shadow'
    primary_task['trace_id'] = trace_id
    shadow_task['trace_id'] = trace_id

    request_json(primary_base_url, 'POST', '/tasks', primary_task)
    request_json(shadow_base_url, 'POST', '/tasks', shadow_task)

    primary_status = wait_terminal(primary_base_url, primary_task['id'], timeout_seconds)
    shadow_status = wait_terminal(shadow_base_url, shadow_task['id'], timeout_seconds)
    primary_events = request_json(primary_base_url, 'GET', f"/events?task_id={primary_task['id']}&limit=100")['events']
    shadow_events = request_json(shadow_base_url, 'GET', f"/events?task_id={shadow_task['id']}&limit=100")['events']

    return {
        'trace_id': trace_id,
        'primary': {'task_id': primary_task['id'], 'status': primary_status, 'events': primary_events},
        'shadow': {'task_id': shadow_task['id'], 'status': shadow_status, 'events': shadow_events},
        'diff': {
            'state_equal': primary_status.get('state') == shadow_status.get('state'),
            'event_count_delta': len(primary_events) - len(shadow_events),
            'event_types_equal': event_types(primary_events) == event_types(shadow_events),
            'primary_event_types': event_types(primary_events),
            'shadow_event_types': event_types(shadow_events),
            'primary_timeline_seconds': timeline_seconds(primary_events),
            'shadow_timeline_seconds': timeline_seconds(shadow_events),
        },
    }


def main():
    parser = argparse.ArgumentParser(description='Compare the same task on two BigClaw endpoints')
    parser.add_argument('--primary', required=True)
    parser.add_argument('--shadow', required=True)
    parser.add_argument('--task-file', required=True)
    parser.add_argument('--timeout-seconds', type=int, default=180)
    parser.add_argument('--health-timeout-seconds', type=int, default=60)
    parser.add_argument('--report-path')
    parser.add_argument('--bundle-dir')
    args = parser.parse_args()

    task = json.load(open(args.task_file))
    comparison = compare_task(args.primary, args.shadow, task, args.timeout_seconds, args.health_timeout_seconds)
    report = build_bundle_report(
        report_kind='shadow_compare',
        comparisons=[comparison],
        task_sources=[args.task_file],
        legacy_single_result=comparison,
    )
    if args.report_path:
        write_bundle_artifacts(
            report,
            report_path=args.report_path,
            bundle_dir=args.bundle_dir,
            bundle_summary_name='shadow-compare-summary.json',
        )
    print(json.dumps(report, ensure_ascii=False, indent=2))
    return 0 if report['comparison_totals']['mismatched'] == 0 else 1


if __name__ == '__main__':
    sys.exit(main())
