#!/usr/bin/env python3
import argparse
import concurrent.futures
import json
import os
import pathlib
import subprocess
import tempfile
import time
import urllib.request


def request_json(base_url, method, path, payload=None):
    data = None if payload is None else json.dumps(payload).encode('utf-8')
    req = urllib.request.Request(base_url.rstrip('/') + path, data=data, method=method, headers={'Content-Type': 'application/json'})
    with urllib.request.urlopen(req, timeout=30) as resp:
        return json.loads(resp.read().decode('utf-8'))


def wait_health(base_url, timeout_seconds=60):
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        try:
            if request_json(base_url, 'GET', '/healthz').get('ok'):
                return
        except Exception:
            pass
        time.sleep(1)
    raise TimeoutError('health check timeout')


def start_service(go_root, env):
    log_path = pathlib.Path(tempfile.mkstemp(prefix='bigclawd-soak-', suffix='.log')[1])
    log_file = log_path.open('w')
    proc = subprocess.Popen(['go', 'run', './cmd/bigclawd'], cwd=go_root, stdout=log_file, stderr=subprocess.STDOUT, env=env)
    return proc, log_path


def submit(base_url, task):
    request_json(base_url, 'POST', '/tasks', task)
    return task['id']


def wait_task(base_url, task_id, timeout_seconds):
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        status = request_json(base_url, 'GET', f'/tasks/{task_id}')
        if status['state'] in {'succeeded', 'dead_letter', 'failed', 'cancelled'}:
            return status
        time.sleep(0.25)
    raise TimeoutError(task_id)


def main():
    parser = argparse.ArgumentParser(description='Local soak runner for BigClaw')
    parser.add_argument('--count', type=int, default=50)
    parser.add_argument('--workers', type=int, default=8)
    parser.add_argument('--base-url', default='http://127.0.0.1:8080')
    parser.add_argument('--go-root', default=str(pathlib.Path(__file__).resolve().parents[2]))
    parser.add_argument('--timeout-seconds', type=int, default=180)
    parser.add_argument('--autostart', action='store_true')
    parser.add_argument('--report-path', default='docs/reports/soak-local-report.json')
    args = parser.parse_args()

    proc = None
    log_path = None
    start = time.time()
    try:
        if args.autostart:
            env = os.environ.copy()
            proc, log_path = start_service(args.go_root, env)
        wait_health(args.base_url)
        tasks = [{
            'id': f'soak-{i}-{int(start)}',
            'title': f'soak task {i}',
            'required_executor': 'local',
            'entrypoint': f'echo soak {i}',
            'execution_timeout_seconds': args.timeout_seconds,
        } for i in range(args.count)]
        with concurrent.futures.ThreadPoolExecutor(max_workers=args.workers) as pool:
            ids = list(pool.map(lambda task: submit(args.base_url, task), tasks))
            statuses = list(pool.map(lambda task_id: wait_task(args.base_url, task_id, args.timeout_seconds), ids))
        elapsed = time.time() - start
        succeeded = sum(1 for status in statuses if status['state'] == 'succeeded')
        report = {
            'count': args.count,
            'workers': args.workers,
            'elapsed_seconds': elapsed,
            'throughput_tasks_per_sec': args.count / elapsed if elapsed else 0,
            'succeeded': succeeded,
            'failed': args.count - succeeded,
            'sample_status': statuses[:3],
        }
        report_path = pathlib.Path(args.go_root) / args.report_path
        report_path.parent.mkdir(parents=True, exist_ok=True)
        report_path.write_text(json.dumps(report, ensure_ascii=False, indent=2))
        print(json.dumps(report, ensure_ascii=False, indent=2))
        return 0 if succeeded == args.count else 1
    finally:
        if proc is not None:
            proc.terminate()
            try:
                proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                proc.kill()
            if log_path:
                print(f'service-log: {log_path}')

if __name__ == '__main__':
    raise SystemExit(main())
