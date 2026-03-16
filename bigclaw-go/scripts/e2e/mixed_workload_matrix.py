#!/usr/bin/env python3
import argparse
import json
import os
import pathlib
import shutil
import socket
import subprocess
import sys
import tempfile
import time
import urllib.request


def http_json(url, method='GET', payload=None, timeout=30):
    data = None if payload is None else json.dumps(payload).encode('utf-8')
    req = urllib.request.Request(url, data=data, method=method, headers={'Content-Type': 'application/json'})
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return json.loads(resp.read().decode('utf-8'))


def wait_for_health(base_url, attempts=60, interval=1.0):
    last_error = None
    for _ in range(attempts):
        try:
            if http_json(base_url + '/healthz').get('ok'):
                return
        except Exception as exc:
            last_error = exc
        time.sleep(interval)
    raise RuntimeError(f'service did not become healthy: {last_error}')


def reserve_local_base_url():
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(('127.0.0.1', 0))
        sock.listen(1)
        port = sock.getsockname()[1]
    return f'http://127.0.0.1:{port}', f'127.0.0.1:{port}'


def build_autostart_env():
    env = os.environ.copy()
    state_dir = pathlib.Path(tempfile.mkdtemp(prefix='bigclawd-mixed-state-'))
    base_url, http_addr = reserve_local_base_url()
    env['BIGCLAW_HTTP_ADDR'] = http_addr
    env['BIGCLAW_QUEUE_BACKEND'] = env.get('BIGCLAW_QUEUE_BACKEND', 'sqlite') or 'sqlite'
    env['BIGCLAW_QUEUE_SQLITE_PATH'] = str(state_dir / 'queue.db')
    env['BIGCLAW_AUDIT_LOG_PATH'] = str(state_dir / 'audit.jsonl')
    env['BIGCLAW_SERVICE_NAME'] = env.get('BIGCLAW_SERVICE_NAME', 'bigclawd-mixed')
    return env, base_url, state_dir


def start_bigclawd(go_root, env):
    log_path = pathlib.Path(tempfile.mkstemp(prefix='bigclawd-mixed-', suffix='.log')[1])
    log_file = log_path.open('w')
    process = subprocess.Popen(['go', 'run', './cmd/bigclawd'], cwd=go_root, stdout=log_file, stderr=subprocess.STDOUT, env=env)
    return process, log_path


def terminal(state):
    return state in {'succeeded', 'dead_letter', 'cancelled', 'failed'}


def wait_task(base_url, task_id, timeout_seconds):
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        status = http_json(base_url + f'/tasks/{task_id}')
        if terminal(status['state']):
            return status
        time.sleep(0.5)
    raise TimeoutError(task_id)


def fetch_events(base_url, task_id):
    return http_json(base_url + f'/events?task_id={task_id}&limit=100')['events']


def routed_event(events):
    for event in events:
        if event.get('type') == 'scheduler.routed':
            return event
    return None


def default_tasks(timestamp):
    return [
        {
            'name': 'local-default',
            'expected_executor': 'local',
            'expectation_label': 'default low/medium risk route',
            'expectation_rule': 'unscoped low-risk tasks stay on the local executor',
            'task': {
                'id': f'mixed-local-{timestamp}',
                'trace_id': f'mixed-local-{timestamp}',
                'title': 'Mixed workload local default',
                'entrypoint': 'echo local default',
                'metadata': {'scenario': 'mixed-workload', 'profile': 'local-default'},
            },
        },
        {
            'name': 'browser-auto',
            'expected_executor': 'kubernetes',
            'expectation_label': 'browser tool route',
            'expectation_rule': 'browser-required workloads should route to kubernetes',
            'task': {
                'id': f'mixed-browser-{timestamp}',
                'trace_id': f'mixed-browser-{timestamp}',
                'title': 'Mixed workload browser auto-route',
                'required_tools': ['browser'],
                'container_image': 'alpine:3.20',
                'entrypoint': 'echo browser via kubernetes',
                'metadata': {'scenario': 'mixed-workload', 'profile': 'browser-auto'},
            },
        },
        {
            'name': 'gpu-auto',
            'expected_executor': 'ray',
            'expectation_label': 'gpu tool route',
            'expectation_rule': 'gpu-required workloads should route to ray',
            'task': {
                'id': f'mixed-gpu-{timestamp}',
                'trace_id': f'mixed-gpu-{timestamp}',
                'title': 'Mixed workload gpu auto-route',
                'required_tools': ['gpu'],
                'entrypoint': "python -c \"print('gpu via ray')\"",
                'metadata': {'scenario': 'mixed-workload', 'profile': 'gpu-auto'},
            },
        },
        {
            'name': 'high-risk-auto',
            'expected_executor': 'kubernetes',
            'expectation_label': 'high-risk isolation route',
            'expectation_rule': 'high-risk workloads should route to the isolated kubernetes executor',
            'task': {
                'id': f'mixed-risk-{timestamp}',
                'trace_id': f'mixed-risk-{timestamp}',
                'title': 'Mixed workload high-risk auto-route',
                'risk_level': 'high',
                'container_image': 'alpine:3.20',
                'entrypoint': 'echo high risk via kubernetes',
                'metadata': {'scenario': 'mixed-workload', 'profile': 'high-risk-auto'},
            },
        },
        {
            'name': 'required-ray',
            'expected_executor': 'ray',
            'expectation_label': 'explicit required executor route',
            'expectation_rule': 'required_executor=ray must override automatic routing',
            'task': {
                'id': f'mixed-required-ray-{timestamp}',
                'trace_id': f'mixed-required-ray-{timestamp}',
                'title': 'Mixed workload explicit ray',
                'required_executor': 'ray',
                'entrypoint': "python -c \"print('required ray')\"",
                'metadata': {'scenario': 'mixed-workload', 'profile': 'required-ray'},
            },
        },
    ]


def write_report(go_root, report_path, payload):
    output_path = pathlib.Path(go_root) / report_path
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')


def copy_artifact(source, destination):
    source_path = pathlib.Path(source)
    if not source_path.exists():
        return ''
    destination.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(source_path, destination)
    return str(destination)


def relpath(path, go_root):
    try:
        return str(pathlib.Path(path).resolve().relative_to(pathlib.Path(go_root).resolve()))
    except ValueError:
        return str(path)


def event_summary(event):
    payload = event.get('payload') if isinstance(event.get('payload'), dict) else {}
    return {
        'id': event.get('id', ''),
        'type': event.get('type', ''),
        'timestamp': event.get('timestamp', ''),
        'executor': payload.get('executor', ''),
        'reason': payload.get('reason', ''),
        'message': payload.get('message', ''),
    }


def build_task_result(spec, status, events):
    routed = routed_event(events)
    routed_executor = None
    routed_reason = None
    if routed and isinstance(routed.get('payload'), dict):
        routed_executor = routed['payload'].get('executor')
        routed_reason = routed['payload'].get('reason')
    latest_event_type = status['latest_event']['type'] if status.get('latest_event') else ''
    return {
        'name': spec['name'],
        'task_id': spec['task']['id'],
        'trace_id': spec['task']['trace_id'],
        'expectation': {
            'executor': spec['expected_executor'],
            'label': spec['expectation_label'],
            'rule': spec['expectation_rule'],
        },
        'observed': {
            'routed_executor': routed_executor,
            'routed_reason': routed_reason,
            'final_state': status['state'],
            'latest_event_type': latest_event_type,
        },
        'event_highlights': [event_summary(event) for event in events],
        'event_types': [event.get('type', '') for event in events],
        'event_count': len(events),
        'ok': status['state'] == 'succeeded' and routed_executor == spec['expected_executor'],
        'task': spec['task'],
        'events': events,
    }


def render_bundle_readme(report):
    lines = [
        '# Mixed Workload Validation Bundle',
        '',
        f"- Run ID: `{report['run_id']}`",
        f"- Generated at: `{report['generated_at']}`",
        f"- Status: `{'succeeded' if report['all_ok'] else 'failed'}`",
        f"- Latest summary: `{report['latest_report_path']}`",
        '',
        '## Scenario drilldowns',
        '',
    ]
    for task in report['tasks']:
        lines.append(f"### {task['name']}")
        lines.append(f"- Expectation: `{task['expectation']['label']}`")
        lines.append(f"- Expected executor: `{task['expectation']['executor']}`")
        lines.append(f"- Routed executor: `{task['observed']['routed_executor']}`")
        lines.append(f"- Final state: `{task['observed']['final_state']}`")
        lines.append(f"- Drilldown JSON: `{task['drilldown']['json_path']}`")
        lines.append('')
    lines.extend(
        [
            '## Shared artifacts',
            '',
            f"- Bundle summary: `{report['bundle']['summary_path']}`",
            f"- Canonical report: `{report['latest_report_path']}`",
        ]
    )
    if report['bundle'].get('audit_log_path'):
        lines.append(f"- Audit log copy: `{report['bundle']['audit_log_path']}`")
    if report['bundle'].get('service_log_path'):
        lines.append(f"- Service log copy: `{report['bundle']['service_log_path']}`")
    lines.append('')
    return '\n'.join(lines)


def materialize_bundle(go_root, bundle_root, run_id, report, state_dir, log_path):
    root = pathlib.Path(go_root)
    bundle_dir = root / bundle_root / run_id
    drilldown_dir = bundle_dir / 'tasks'
    drilldown_dir.mkdir(parents=True, exist_ok=True)

    for task in report['tasks']:
        drilldown_payload = {
            'generated_at': report['generated_at'],
            'run_id': report['run_id'],
            'name': task['name'],
            'task_id': task['task_id'],
            'trace_id': task['trace_id'],
            'expectation': task['expectation'],
            'observed': task['observed'],
            'event_types': task['event_types'],
            'task': task['task'],
            'events': task['events'],
            'ok': task['ok'],
        }
        drilldown_path = drilldown_dir / f"{task['name']}.json"
        write_report(go_root, relpath(drilldown_path, go_root), drilldown_payload)
        task['drilldown'] = {'json_path': relpath(drilldown_path, go_root)}
        task.pop('task', None)
        task.pop('events', None)

    summary_path = bundle_dir / 'summary.json'
    readme_path = bundle_dir / 'README.md'
    report['bundle'] = {
        'path': relpath(bundle_dir, go_root),
        'summary_path': relpath(summary_path, go_root),
        'readme_path': relpath(readme_path, go_root),
    }
    if state_dir:
        copied_audit = copy_artifact(pathlib.Path(state_dir) / 'audit.jsonl', bundle_dir / 'audit.jsonl')
        if copied_audit:
            report['bundle']['audit_log_path'] = relpath(copied_audit, go_root)
    if log_path:
        copied_service_log = copy_artifact(log_path, bundle_dir / 'service.log')
        if copied_service_log:
            report['bundle']['service_log_path'] = relpath(copied_service_log, go_root)

    write_report(go_root, relpath(summary_path, go_root), report)
    readme_path.write_text(render_bundle_readme(report) + '\n', encoding='utf-8')
    return report


def main():
    parser = argparse.ArgumentParser(description='Run a mixed workload matrix against one BigClaw Go control plane')
    parser.add_argument('--go-root', default=str(pathlib.Path(__file__).resolve().parents[2]))
    parser.add_argument('--report-path', default='docs/reports/mixed-workload-matrix-report.json')
    parser.add_argument('--bundle-root', default='docs/reports/mixed-workload-runs')
    parser.add_argument('--timeout-seconds', type=int, default=240)
    parser.add_argument('--autostart', action='store_true', default=True)
    args = parser.parse_args()

    process = None
    log_path = None
    state_dir = None
    report = None
    base_url = os.environ.get('BIGCLAW_ADDR', 'http://127.0.0.1:8080')
    try:
        if args.autostart:
            env, base_url, state_dir = build_autostart_env()
            process, log_path = start_bigclawd(args.go_root, env)
        wait_for_health(base_url)
        timestamp = int(time.time())
        task_specs = default_tasks(timestamp)
        results = []
        for spec in task_specs:
            task = spec['task']
            http_json(base_url + '/tasks', method='POST', payload=task)
            status = wait_task(base_url, task['id'], args.timeout_seconds)
            events = fetch_events(base_url, task['id'])
            results.append(build_task_result(spec, status, events))
        run_id = time.strftime('%Y%m%dT%H%M%SZ', time.gmtime())
        report = {
            'generated_at': time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime()),
            'run_id': run_id,
            'base_url': base_url,
            'state_dir': str(state_dir) if state_dir else '',
            'service_log': str(log_path) if log_path else '',
            'all_ok': all(item['ok'] for item in results),
            'tasks': results,
        }
        report['latest_report_path'] = args.report_path
        report = materialize_bundle(args.go_root, args.bundle_root, run_id, report, state_dir, log_path)
        write_report(args.go_root, args.report_path, report)
        print(json.dumps(report, ensure_ascii=False, indent=2))
        return 0 if report['all_ok'] else 1
    finally:
        if process is not None:
            process.terminate()
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill()
            if log_path and log_path.exists():
                print(f'bigclawd log: {log_path}', file=sys.stderr)


if __name__ == '__main__':
    raise SystemExit(main())
