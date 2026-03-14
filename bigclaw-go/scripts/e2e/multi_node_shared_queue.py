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


def build_node_envs(root_state_dir):
    queue_path = root_state_dir / 'shared-queue.db'
    node_configs = []
    for node_name in ['node-a', 'node-b']:
        env = os.environ.copy()
        base_url, http_addr = reserve_local_base_url()
        env['BIGCLAW_HTTP_ADDR'] = http_addr
        env['BIGCLAW_SERVICE_NAME'] = node_name
        env['BIGCLAW_QUEUE_BACKEND'] = 'sqlite'
        env['BIGCLAW_QUEUE_SQLITE_PATH'] = str(queue_path)
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


def main():
    parser = argparse.ArgumentParser(description='Run a two-node shared-queue coordination proof for BigClaw Go')
    parser.add_argument('--go-root', default=str(pathlib.Path(__file__).resolve().parents[2]))
    parser.add_argument('--report-path', default='docs/reports/multi-node-shared-queue-report.json')
    parser.add_argument('--count', type=int, default=200)
    parser.add_argument('--submit-workers', type=int, default=8)
    parser.add_argument('--timeout-seconds', type=int, default=180)
    args = parser.parse_args()

    root_state_dir = pathlib.Path(tempfile.mkdtemp(prefix='bigclawd-multinode-'))
    node_configs = build_node_envs(root_state_dir)
    processes = []
    timestamp = int(time.time())
    report = None
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

        report = {
            'generated_at': time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime()),
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
        output_path = pathlib.Path(args.go_root) / args.report_path
        output_path.parent.mkdir(parents=True, exist_ok=True)
        output_path.write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
        print(json.dumps(report, ensure_ascii=False, indent=2))
        return 0 if report['all_ok'] else 1
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
