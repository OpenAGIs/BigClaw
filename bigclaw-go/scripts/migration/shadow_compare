#!/usr/bin/env python3
import argparse
import datetime
import json
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
    args = parser.parse_args()

    task = json.load(open(args.task_file))
    report = compare_task(args.primary, args.shadow, task, args.timeout_seconds, args.health_timeout_seconds)
    if args.report_path:
        with open(args.report_path, 'w', encoding='utf-8') as fh:
            json.dump(report, fh, ensure_ascii=False, indent=2)
    print(json.dumps(report, ensure_ascii=False, indent=2))
    return 0 if report['diff']['state_equal'] and report['diff']['event_types_equal'] else 1


if __name__ == '__main__':
    sys.exit(main())
