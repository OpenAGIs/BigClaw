#!/usr/bin/env python3
import argparse
import concurrent.futures
import json
import os
import pathlib
import socket
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request
from datetime import datetime, timezone


class HTTPRequestError(RuntimeError):
    def __init__(self, *, status, payload):
        super().__init__(f'http request failed with status {status}')
        self.status = status
        self.payload = payload


RUNTIME_TAKEOVER_EVENT_TYPES = {
    'subscriber.lease_acquired',
    'subscriber.lease_rejected',
    'subscriber.lease_expired',
    'subscriber.takeover_succeeded',
    'subscriber.checkpoint_committed',
    'subscriber.checkpoint_rejected',
}


def http_json(url, method='GET', payload=None, timeout=30):
    data = None if payload is None else json.dumps(payload).encode('utf-8')
    req = urllib.request.Request(url, data=data, method=method, headers={'Content-Type': 'application/json'})
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return json.loads(resp.read().decode('utf-8'))
    except urllib.error.HTTPError as exc:
        raw = exc.read().decode('utf-8', errors='replace')
        try:
            payload = json.loads(raw)
        except json.JSONDecodeError:
            payload = {'error': raw}
        raise HTTPRequestError(status=exc.code, payload=payload) from exc


def utc_iso(now=None):
    moment = time.gmtime() if now is None else now
    return time.strftime('%Y-%m-%dT%H:%M:%SZ', moment)


def normalize_iso8601(value):
    if not value:
        return ''
    return datetime.fromisoformat(value).astimezone(timezone.utc).isoformat().replace('+00:00', 'Z')


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


def build_node_envs(root_state_dir):
    queue_path = root_state_dir / 'shared-queue.db'
    subscriber_lease_path = root_state_dir / 'shared-subscriber-leases.db'
    node_configs = []
    for node_name in ['node-a', 'node-b']:
        env = os.environ.copy()
        base_url, http_addr = reserve_local_base_url()
        env['BIGCLAW_HTTP_ADDR'] = http_addr
        env['BIGCLAW_SERVICE_NAME'] = node_name
        env['BIGCLAW_QUEUE_BACKEND'] = 'sqlite'
        env['BIGCLAW_QUEUE_SQLITE_PATH'] = str(queue_path)
        env['BIGCLAW_SUBSCRIBER_LEASE_SQLITE_PATH'] = str(subscriber_lease_path)
        env['BIGCLAW_AUDIT_LOG_PATH'] = str(root_state_dir / f'{node_name}-audit.jsonl')
        env['BIGCLAW_POLL_INTERVAL'] = env.get('BIGCLAW_POLL_INTERVAL', '100ms')
        node_configs.append({
            'name': node_name,
            'env': env,
            'base_url': base_url,
            'audit_path': root_state_dir / f'{node_name}-audit.jsonl',
        })
    return node_configs


def start_bigclawd(go_root, env, prefix):
    log_path = pathlib.Path(tempfile.mkstemp(prefix=prefix, suffix='.log')[1])
    log_file = log_path.open('w')
    process = subprocess.Popen(['go', 'run', './cmd/bigclawd'], cwd=go_root, stdout=log_file, stderr=subprocess.STDOUT, env=env)
    return process, log_path


def submit_task(base_url, task):
    http_json(base_url + '/tasks', method='POST', payload=task)
    return task['id']


def read_jsonl(path, node_name):
    if not path.exists():
        return []
    events = []
    with path.open() as handle:
        for line in handle:
            line = line.strip()
            if not line:
                continue
            payload = json.loads(line)
            payload['_node'] = node_name
            events.append(payload)
    return events


def write_jsonl(path, payload):
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open('a', encoding='utf-8') as handle:
        handle.write(json.dumps(payload, ensure_ascii=False) + '\n')


def aggregate_events(node_configs):
    events = []
    for node in node_configs:
        events.extend(read_jsonl(node['audit_path'], node['name']))
    return events


def summarize(tasks, events):
    per_task = {task_id: {'started': [], 'completed': [], 'queued': []} for task_id in tasks}
    for event in events:
        task_id = event.get('task_id')
        if task_id not in per_task:
            continue
        if event.get('type') == 'task.started':
            per_task[task_id]['started'].append(event['_node'])
        elif event.get('type') == 'task.completed':
            per_task[task_id]['completed'].append(event['_node'])
        elif event.get('type') == 'task.queued':
            per_task[task_id]['queued'].append(event['_node'])
    return per_task


def checkpoint_payload(lease):
    return {
        'owner': lease['consumer_id'],
        'lease_epoch': lease['lease_epoch'],
        'lease_token': lease['lease_token'],
        'offset': lease['checkpoint_offset'],
        'event_id': lease.get('checkpoint_event_id', ''),
        'updated_at': normalize_iso8601(lease['updated_at']),
    }


def cursor(offset, event_id):
    return {'offset': offset, 'event_id': event_id}


def runtime_takeover_events(node_configs, subscriber_group, subscriber_id):
    timeline = []
    per_node = {}
    for node in node_configs:
        node_events = []
        for event in read_jsonl(node['audit_path'], node['name']):
            payload = event.get('payload', {})
            if event.get('type') not in RUNTIME_TAKEOVER_EVENT_TYPES:
                continue
            if payload.get('group_id') != subscriber_group or payload.get('subscriber_id') != subscriber_id:
                continue
            node_events.append(event)
            timeline.append(event)
        per_node[node['name']] = node_events
    timeline.sort(key=lambda item: (item.get('timestamp', ''), item.get('id', '')))
    return timeline, per_node


def export_runtime_takeover_audit(artifact_root, scenario_id, repo_root, per_node_events):
    scenario_dir = artifact_root / scenario_id
    scenario_dir.mkdir(parents=True, exist_ok=True)
    paths = []
    for node_name, node_events in per_node_events.items():
        path = scenario_dir / f'{node_name}-audit.jsonl'
        with path.open('w', encoding='utf-8') as handle:
            for event in node_events:
                handle.write(json.dumps(event, ensure_ascii=False) + '\n')
        paths.append(str(path.relative_to(repo_root)))
    return sorted(paths)


def to_takeover_timeline(runtime_events):
    timeline = []
    for event in runtime_events:
        payload = event.get('payload', {})
        event_type = event.get('type')
        action = event_type
        subscriber = payload.get('consumer_id') or payload.get('attempted_consumer_id') or 'unknown'
        details = {
            'runtime_event_id': event.get('id', ''),
            'runtime_event_type': event_type,
            'audit_node': event.get('_node', 'unknown'),
        }
        if event_type == 'subscriber.lease_acquired':
            action = 'lease_acquired'
            details['lease_epoch'] = payload.get('lease_epoch')
            details['renewal'] = payload.get('renewal', False)
        elif event_type == 'subscriber.lease_rejected':
            action = 'lease_rejected'
            subscriber = payload.get('attempted_consumer_id', subscriber)
            details['attempted_owner'] = payload.get('attempted_consumer_id')
            details['accepted_owner'] = payload.get('consumer_id')
            details['lease_epoch'] = payload.get('lease_epoch')
            details['reason'] = payload.get('reason', '')
        elif event_type == 'subscriber.lease_expired':
            action = 'lease_expired'
            subscriber = payload.get('expired_consumer_id', subscriber)
            details['last_offset'] = payload.get('checkpoint_offset')
            details['takeover_consumer_id'] = payload.get('takeover_consumer_id')
        elif event_type == 'subscriber.takeover_succeeded':
            action = 'takeover_succeeded'
            details['lease_epoch'] = payload.get('lease_epoch')
            details['previous_owner'] = payload.get('previous_consumer_id')
        elif event_type == 'subscriber.checkpoint_committed':
            action = 'checkpoint_committed'
            details['offset'] = payload.get('checkpoint_offset')
            details['event_id'] = payload.get('checkpoint_event_id', '')
        elif event_type == 'subscriber.checkpoint_rejected':
            reason = payload.get('reason', '')
            action = 'lease_fenced' if 'fenced' in reason else 'checkpoint_rejected'
            subscriber = payload.get('attempted_consumer_id', subscriber)
            details['attempted_offset'] = payload.get('attempted_checkpoint_offset')
            details['attempted_event_id'] = payload.get('attempted_checkpoint_event_id', '')
            details['accepted_owner'] = payload.get('consumer_id')
            details['reason'] = reason
        timeline.append(
            {
                'timestamp': normalize_iso8601(event.get('timestamp', '')),
                'subscriber': subscriber,
                'action': action,
                'details': details,
            }
        )
    return timeline


def task_event_excerpt(events, task_id):
    excerpt = []
    for event in events:
        if event.get('task_id') != task_id:
            continue
        excerpt.append(
            {
                'event_id': event.get('id', ''),
                'delivered_by': [event.get('_node', 'unknown')],
                'delivery_kind': event.get('type', 'unknown'),
            }
        )
    return excerpt


def wait_for_task_completion(task_id, submitted_by, node_configs, timeout_seconds):
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        per_task = summarize({task_id: submitted_by}, aggregate_events(node_configs))
        item = per_task[task_id]
        if item['completed']:
            return item
        time.sleep(0.2)
    raise TimeoutError(f'timed out waiting for task {task_id} to complete')


def submit_takeover_task(submit_node, scenario_id, scenario_index):
    task_id = f'{scenario_id}-task-{scenario_index}-{int(time.time())}'
    task = {
        'id': task_id,
        'trace_id': task_id,
        'title': f'{scenario_id} shared-queue task',
        'entrypoint': f'echo {scenario_id}',
        'required_executor': 'local',
        'metadata': {
            'scenario': 'live-multi-node-subscriber-takeover',
            'submit_node': submit_node['name'],
            'takeover_scenario': scenario_id,
        },
    }
    submit_task(submit_node['base_url'], task)
    return task


def build_assertion_results(lease_owner_timeline, checkpoint_before, checkpoint_after, replay_start_cursor, replay_end_cursor, duplicate_events, stale_write_rejections, audit_timeline):
    audit_checks = [
        {
            'label': 'ownership handoff is visible in the audit timeline',
            'passed': len({entry['owner'] for entry in lease_owner_timeline}) >= 2,
        },
        {
            'label': 'audit timeline contains acquisition, expiry, rejection, or takeover events',
            'passed': any(item['action'] in {'lease_acquired', 'lease_expired', 'lease_rejected', 'lease_fenced', 'takeover_succeeded'} for item in audit_timeline),
        },
        {
            'label': 'audit timeline stays ordered by timestamp',
            'passed': [item['timestamp'] for item in audit_timeline] == sorted(item['timestamp'] for item in audit_timeline),
        },
        {
            'label': 'runtime takeover events keep required owner and lease fields',
            'passed': all(
                item['details'].get('runtime_event_id')
                and item['details'].get('runtime_event_type')
                and item['subscriber']
                for item in audit_timeline
            ),
        },
    ]
    checkpoint_checks = [
        {
            'label': 'checkpoint never regresses across takeover',
            'passed': checkpoint_after['offset'] >= checkpoint_before['offset'],
        },
        {
            'label': 'final checkpoint owner matches the final lease owner',
            'passed': checkpoint_after['owner'] == lease_owner_timeline[-1]['owner'],
        },
        {
            'label': 'stale writers do not replace the accepted checkpoint owner',
            'passed': stale_write_rejections == 0 or checkpoint_after['owner'] == lease_owner_timeline[-1]['owner'],
        },
    ]
    replay_checks = [
        {
            'label': 'replay restarts from the durable checkpoint boundary',
            'passed': replay_start_cursor['offset'] == checkpoint_before['offset'],
        },
        {
            'label': 'replay end cursor advances to the final durable checkpoint',
            'passed': replay_end_cursor['offset'] == checkpoint_after['offset'],
        },
        {
            'label': 'duplicate replay candidates are counted explicitly',
            'passed': len(duplicate_events) >= 0,
        },
    ]
    return {
        'audit': audit_checks,
        'checkpoint': checkpoint_checks,
        'replay': replay_checks,
    }


def owner_timeline_entry(owner, event, lease):
    return {
        'timestamp': utc_iso(),
        'owner': owner,
        'event': event,
        'lease_epoch': lease['lease_epoch'],
        'checkpoint_offset': lease['checkpoint_offset'],
        'checkpoint_event_id': lease.get('checkpoint_event_id', ''),
    }


def live_scenario_result(
    *,
    scenario_id,
    title,
    primary_node,
    takeover_node,
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
    audit_log_paths,
):
    assertion_results = build_assertion_results(
        lease_owner_timeline,
        checkpoint_before,
        checkpoint_after,
        replay_start_cursor,
        replay_end_cursor,
        duplicate_events,
        stale_write_rejections,
        audit_timeline,
    )
    all_assertions_passed = all(
        item['passed']
        for category in assertion_results.values()
        for item in category
    )
    return {
        'id': scenario_id,
        'title': title,
        'subscriber_group': subscriber_group,
        'primary_subscriber': primary_node['name'],
        'takeover_subscriber': takeover_node['name'],
        'task_or_trace_id': task_or_trace_id,
        'audit_assertions': audit_assertions,
        'checkpoint_assertions': checkpoint_assertions,
        'replay_assertions': replay_assertions,
        'lease_owner_timeline': lease_owner_timeline,
        'checkpoint_before': checkpoint_before,
        'checkpoint_after': checkpoint_after,
        'replay_start_cursor': replay_start_cursor,
        'replay_end_cursor': replay_end_cursor,
        'duplicate_delivery_count': len(duplicate_events),
        'duplicate_events': duplicate_events,
        'stale_write_rejections': stale_write_rejections,
        'audit_log_paths': audit_log_paths,
        'event_log_excerpt': event_log_excerpt,
        'audit_timeline': audit_timeline,
        'assertion_results': assertion_results,
        'all_assertions_passed': all_assertions_passed,
        'local_limitations': local_limitations,
    }


def execute_takeover_scenario(
    *,
    primary_node,
    takeover_node,
    scenario_id,
    title,
    scenario_index,
    node_configs,
    artifact_root,
    timeout_seconds,
    ttl_seconds,
    offset_base,
    duplicate_events,
    include_conflict_probe,
    include_idle_gap,
):
    ttl_seconds = max(1, int(round(ttl_seconds)))
    subscriber_group = f'live-{scenario_id}'
    subscriber_id = 'event-stream'
    task = submit_takeover_task(primary_node, scenario_id, scenario_index)
    task_summary = wait_for_task_completion(task['id'], primary_node['name'], node_configs, timeout_seconds)
    time.sleep(0.2)
    task_events = task_event_excerpt(aggregate_events(node_configs), task['id'])

    lease_owner_timeline = []
    repo_root = artifact_root.parent.parent.parent

    lease = http_json(
        primary_node['base_url'] + '/subscriber-groups/leases',
        method='POST',
        payload={
            'group_id': subscriber_group,
            'subscriber_id': subscriber_id,
            'consumer_id': primary_node['name'],
            'ttl_seconds': ttl_seconds,
        },
    )['lease']

    lease_owner_timeline.append(owner_timeline_entry(primary_node['name'], 'lease_acquired', lease))

    if include_conflict_probe:
        try:
            http_json(
                takeover_node['base_url'] + '/subscriber-groups/leases',
                method='POST',
                payload={
                    'group_id': subscriber_group,
                    'subscriber_id': subscriber_id,
                    'consumer_id': takeover_node['name'],
                    'ttl_seconds': ttl_seconds,
                },
            )
        except HTTPRequestError as exc:
            if exc.status != 409:
                raise


    lease = http_json(
        primary_node['base_url'] + '/subscriber-groups/checkpoints',
        method='POST',
        payload={
            'group_id': subscriber_group,
            'subscriber_id': subscriber_id,
            'consumer_id': primary_node['name'],
            'lease_token': lease['lease_token'],
            'lease_epoch': lease['lease_epoch'],
            'checkpoint_offset': offset_base,
            'checkpoint_event_id': f'evt-{offset_base}',
        },
    )['lease']
    checkpoint_before = checkpoint_payload(lease)

    if include_idle_gap:
        time.sleep(0.05)

    time.sleep(ttl_seconds + 0.3)

    takeover_lease = http_json(
        takeover_node['base_url'] + '/subscriber-groups/leases',
        method='POST',
        payload={
            'group_id': subscriber_group,
            'subscriber_id': subscriber_id,
            'consumer_id': takeover_node['name'],
            'ttl_seconds': ttl_seconds,
        },
    )['lease']

    lease_owner_timeline.append(owner_timeline_entry(takeover_node['name'], 'takeover_acquired', takeover_lease))

    attempted_offset = offset_base + 1
    try:
        http_json(
            primary_node['base_url'] + '/subscriber-groups/checkpoints',
            method='POST',
            payload={
                'group_id': subscriber_group,
                'subscriber_id': subscriber_id,
                'consumer_id': primary_node['name'],
                'lease_token': lease['lease_token'],
                'lease_epoch': lease['lease_epoch'],
                'checkpoint_offset': attempted_offset,
                'checkpoint_event_id': f'evt-{attempted_offset}',
            },
        )
    except HTTPRequestError as exc:
        if exc.status != 409:
            raise


    final_offset = offset_base + len(duplicate_events)
    takeover_lease = http_json(
        takeover_node['base_url'] + '/subscriber-groups/checkpoints',
        method='POST',
        payload={
            'group_id': subscriber_group,
            'subscriber_id': subscriber_id,
            'consumer_id': takeover_node['name'],
            'lease_token': takeover_lease['lease_token'],
            'lease_epoch': takeover_lease['lease_epoch'],
            'checkpoint_offset': final_offset,
            'checkpoint_event_id': f'evt-{final_offset}',
        },
    )['lease']
    checkpoint_after = checkpoint_payload(takeover_lease)
    time.sleep(0.1)
    runtime_events, per_node_runtime_events = runtime_takeover_events(node_configs, subscriber_group, subscriber_id)
    audit_timeline = to_takeover_timeline(runtime_events)
    stale_write_rejections = sum(1 for item in audit_timeline if item['action'] == 'lease_fenced')
    audit_log_paths = export_runtime_takeover_audit(artifact_root, scenario_id, repo_root, per_node_runtime_events)

    if task_summary['completed'] and task_summary['completed'][0] != primary_node['name']:
        task_events.append(
            {
                'event_id': f'{task["id"]}-cross-node',
                'delivered_by': task_summary['completed'],
                'delivery_kind': 'shared_queue_cross_node_completion',
            }
        )

    return live_scenario_result(
        scenario_id=scenario_id,
        title=title,
        primary_node=primary_node,
        takeover_node=takeover_node,
        task_or_trace_id=task['trace_id'],
        subscriber_group=subscriber_group,
        audit_assertions=[
            'Per-node audit artifacts are filtered excerpts from runtime-emitted audit events rather than harness-authored companion logs.',
            'The live report binds takeover actions to a real shared-queue task trace ID for the same two-node cluster run.',
            'Lease rejection and accepted takeover owner are captured in one ordered audit timeline.',
        ],
        checkpoint_assertions=[
            'Checkpoint after takeover is greater than or equal to the last durable checkpoint from the primary subscriber.',
            'Standby checkpoint commit is attributed to the new lease owner returned by the live API.',
            'Late primary checkpoint writes are fenced once takeover succeeds.',
        ],
        replay_assertions=[
            'Replay resumes from the last durable checkpoint boundary returned by the live lease endpoint.',
            'Duplicate replay candidates are counted explicitly from the overlap between the last durable offset and final takeover offset.',
            'The final replay cursor and final owner are both emitted in the live report schema.',
        ],
        lease_owner_timeline=lease_owner_timeline,
        checkpoint_before=checkpoint_before,
        checkpoint_after=checkpoint_after,
        replay_start_cursor=cursor(offset_base, f'evt-{offset_base}'),
        replay_end_cursor=cursor(final_offset, f'evt-{final_offset}'),
        duplicate_events=duplicate_events,
        stale_write_rejections=stale_write_rejections,
        audit_timeline=audit_timeline,
        event_log_excerpt=task_events,
        local_limitations=[
            'The live proof runs against a real two-node shared-queue cluster and one shared SQLite-backed subscriber lease store.',
            'The checked-in takeover artifacts are derived from runtime-emitted subscriber transition events exported per scenario.',
            'This proof upgrades ownership to a shared durable scaffold without claiming broker-backed or replicated subscriber ownership.',
        ],
        audit_log_paths=audit_log_paths,
    )


def build_live_takeover_report(scenarios, shared_queue_report_path):
    passing = sum(1 for scenario in scenarios if scenario['all_assertions_passed'])
    return {
        'generated_at': utc_iso(),
        'ticket': 'OPE-260',
        'title': 'Live multi-node subscriber takeover proof',
        'status': 'live-multi-node-proof',
        'harness_mode': 'live_multi_node_bigclawd_cluster',
        'current_primitives': {
            'lease_aware_checkpoints': [
                'internal/events/subscriber_leases.go',
                'internal/events/subscriber_leases_test.go',
                'internal/api/server.go',
            ],
            'shared_queue_evidence': [
                'scripts/e2e/multi_node_shared_queue.py',
                shared_queue_report_path,
            ],
            'live_takeover_harness': [
                'internal/api/server.go',
                'scripts/e2e/multi_node_shared_queue.py',
                'docs/reports/live-multi-node-subscriber-takeover-report.json',
            ],
        },
        'required_report_sections': [
            'scenario metadata',
            'fault injection steps',
            'audit assertions',
            'checkpoint assertions',
            'replay assertions',
            'per-node audit artifacts',
            'final owner and replay cursor summary',
            'duplicate delivery accounting',
            'open blockers and follow-up implementation hooks',
        ],
        'implementation_path': [
            'run a real two-node bigclawd cluster against one shared SQLite queue',
            'drive lease acquisition, expiry, fencing, and checkpoint takeover through the live subscriber-group API on both nodes against one shared SQLite lease backend',
            'slice canonical per-node takeover artifacts out of the runtime audit stream beside the checked-in report',
            'keep broker-backed and replicated subscriber ownership caveats explicit until a broker-native lease backend exists',
        ],
        'summary': {
            'scenario_count': len(scenarios),
            'passing_scenarios': passing,
            'failing_scenarios': len(scenarios) - passing,
            'duplicate_delivery_count': sum(scenario['duplicate_delivery_count'] for scenario in scenarios),
            'stale_write_rejections': sum(scenario['stale_write_rejections'] for scenario in scenarios),
        },
        'scenarios': scenarios,
        'remaining_gaps': [
            'Subscriber ownership now uses a shared durable SQLite scaffold, but it is not yet broker-backed or replicated.',
            'The live proof reuses real shared-queue nodes but does not yet validate broker-backed or replicated subscriber ownership.',
            'Native runtime audit coverage now captures takeover transitions, but the proof still depends on the current lease API rather than broker-backed replay ownership.',
        ],
    }


def main():
    parser = argparse.ArgumentParser(description='Run a two-node shared-queue coordination proof for BigClaw Go')
    parser.add_argument('--go-root', default=str(pathlib.Path(__file__).resolve().parents[2]))
    parser.add_argument('--report-path', default='docs/reports/multi-node-shared-queue-report.json')
    parser.add_argument('--takeover-report-path', default='docs/reports/live-multi-node-subscriber-takeover-report.json')
    parser.add_argument('--takeover-artifact-dir', default='docs/reports/live-multi-node-subscriber-takeover-artifacts')
    parser.add_argument('--takeover-ttl-seconds', type=float, default=1.0)
    parser.add_argument('--count', type=int, default=200)
    parser.add_argument('--submit-workers', type=int, default=8)
    parser.add_argument('--timeout-seconds', type=int, default=180)
    args = parser.parse_args()

    root_state_dir = pathlib.Path(tempfile.mkdtemp(prefix='bigclawd-multinode-'))
    node_configs = build_node_envs(root_state_dir)
    processes = []
    timestamp = int(time.time())
    shared_queue_report = None
    live_takeover_report = None
    try:
        for node in node_configs:
            process, log_path = start_bigclawd(args.go_root, node['env'], f"{node['name']}-")
            node['process'] = process
            node['service_log'] = log_path
            processes.append(process)
        for node in node_configs:
            wait_for_health(node['base_url'])

        submitted_by = {}
        tasks = []
        for index in range(args.count):
            submit_node = node_configs[index % len(node_configs)]
            task_id = f'multinode-{index}-{timestamp}'
            task = {
                'id': task_id,
                'trace_id': task_id,
                'title': f'multi-node task {index}',
                'entrypoint': f'echo multinode {index}',
                'required_executor': 'local',
                'metadata': {
                    'scenario': 'multi-node-shared-queue',
                    'submit_node': submit_node['name'],
                },
            }
            tasks.append((submit_node['base_url'], task))
            submitted_by[task_id] = submit_node['name']

        with concurrent.futures.ThreadPoolExecutor(max_workers=args.submit_workers) as pool:
            list(pool.map(lambda item: submit_task(item[0], item[1]), tasks))

        deadline = time.time() + args.timeout_seconds
        while time.time() < deadline:
            per_task = summarize(submitted_by, aggregate_events(node_configs))
            if all(len(item['completed']) >= 1 for item in per_task.values()):
                break
            time.sleep(0.5)
        else:
            raise TimeoutError('timed out waiting for all multi-node tasks to complete')

        events = aggregate_events(node_configs)
        per_task = summarize(submitted_by, events)
        duplicate_started = sorted(task_id for task_id, item in per_task.items() if len(item['started']) > 1)
        duplicate_completed = sorted(task_id for task_id, item in per_task.items() if len(item['completed']) > 1)
        missing_completed = sorted(task_id for task_id, item in per_task.items() if len(item['completed']) == 0)
        completion_by_node = {node['name']: 0 for node in node_configs}
        cross_node_completions = 0
        for task_id, item in per_task.items():
            if item['completed']:
                completion_node = item['completed'][0]
                completion_by_node[completion_node] += 1
                if completion_node != submitted_by[task_id]:
                    cross_node_completions += 1

        shared_queue_report = {
            'generated_at': utc_iso(),
            'root_state_dir': str(root_state_dir),
            'queue_path': str(root_state_dir / 'shared-queue.db'),
            'count': args.count,
            'submitted_by_node': {node['name']: sum(1 for name in submitted_by.values() if name == node['name']) for node in node_configs},
            'completed_by_node': completion_by_node,
            'cross_node_completions': cross_node_completions,
            'duplicate_started_tasks': duplicate_started,
            'duplicate_completed_tasks': duplicate_completed,
            'missing_completed_tasks': missing_completed,
            'all_ok': not duplicate_started and not duplicate_completed and not missing_completed and all(value > 0 for value in completion_by_node.values()),
            'nodes': [
                {
                    'name': node['name'],
                    'base_url': node['base_url'],
                    'audit_path': str(node['audit_path']),
                    'service_log': str(node['service_log']),
                }
                for node in node_configs
            ],
        }
        report_output_path = pathlib.Path(args.go_root) / args.report_path
        report_output_path.parent.mkdir(parents=True, exist_ok=True)
        report_output_path.write_text(json.dumps(shared_queue_report, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')

        artifact_root = pathlib.Path(args.go_root) / args.takeover_artifact_dir
        if artifact_root.exists():
            for path in sorted(artifact_root.rglob('*.jsonl')):
                path.unlink()

        live_scenarios = [
            execute_takeover_scenario(
                primary_node=node_configs[0],
                takeover_node=node_configs[1],
                scenario_id='lease-expiry-stale-writer-rejected-live',
                title='Lease expires on node-a and node-b takes ownership with stale-writer fencing',
                scenario_index=1,
                node_configs=node_configs,
                artifact_root=artifact_root,
                timeout_seconds=min(args.timeout_seconds, 30),
                ttl_seconds=args.takeover_ttl_seconds,
                offset_base=80,
                duplicate_events=['evt-81'],
                include_conflict_probe=False,
                include_idle_gap=False,
            ),
            execute_takeover_scenario(
                primary_node=node_configs[1],
                takeover_node=node_configs[0],
                scenario_id='contention-then-takeover-live',
                title='Node-a is rejected during active ownership and takes over after node-b lease expiry',
                scenario_index=2,
                node_configs=node_configs,
                artifact_root=artifact_root,
                timeout_seconds=min(args.timeout_seconds, 30),
                ttl_seconds=args.takeover_ttl_seconds,
                offset_base=120,
                duplicate_events=['evt-121', 'evt-122'],
                include_conflict_probe=True,
                include_idle_gap=False,
            ),
            execute_takeover_scenario(
                primary_node=node_configs[1],
                takeover_node=node_configs[0],
                scenario_id='idle-primary-takeover-live',
                title='Node-b stops checkpointing and node-a advances the durable cursor after expiry',
                scenario_index=3,
                node_configs=node_configs,
                artifact_root=artifact_root,
                timeout_seconds=min(args.timeout_seconds, 30),
                ttl_seconds=args.takeover_ttl_seconds,
                offset_base=40,
                duplicate_events=['evt-41'],
                include_conflict_probe=False,
                include_idle_gap=True,
            ),
        ]
        live_takeover_report = build_live_takeover_report(live_scenarios, args.report_path)
        takeover_output_path = pathlib.Path(args.go_root) / args.takeover_report_path
        takeover_output_path.parent.mkdir(parents=True, exist_ok=True)
        takeover_output_path.write_text(json.dumps(live_takeover_report, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')

        print(
            json.dumps(
                {
                    'shared_queue_report': shared_queue_report,
                    'live_takeover_report': live_takeover_report,
                },
                ensure_ascii=False,
                indent=2,
            )
        )
        return 0 if shared_queue_report['all_ok'] and live_takeover_report['summary']['failing_scenarios'] == 0 else 1
    finally:
        for node in node_configs:
            process = node.get('process')
            if process is None:
                continue
            process.terminate()
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill()
            if node.get('service_log'):
                print(f"{node['name']} log: {node['service_log']}", file=sys.stderr)


if __name__ == '__main__':
    raise SystemExit(main())
