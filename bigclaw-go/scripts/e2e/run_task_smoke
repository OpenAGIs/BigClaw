#!/usr/bin/env python3
import argparse
import json
import os
import pathlib
import socket
import subprocess
import sys
import tempfile
import time
import urllib.request


def http_json(url, method="GET", payload=None, timeout=10):
    data = None
    headers = {"Content-Type": "application/json"}
    if payload is not None:
        data = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(url, data=data, method=method, headers=headers)
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return json.loads(resp.read().decode("utf-8"))


def wait_for_health(base_url, attempts=60, interval=1.0):
    last_error = None
    for _ in range(attempts):
        try:
            payload = http_json(base_url + "/healthz")
            if payload.get("ok"):
                return
        except Exception as exc:
            last_error = exc
        time.sleep(interval)
    raise RuntimeError(f"service did not become healthy: {last_error}")


def reserve_local_base_url():
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        sock.listen(1)
        port = sock.getsockname()[1]
    return f"http://127.0.0.1:{port}", f"127.0.0.1:{port}"


def build_autostart_env():
    env = os.environ.copy()
    state_dir = pathlib.Path(tempfile.mkdtemp(prefix="bigclawd-state-"))
    queue_backend = env.get("BIGCLAW_QUEUE_BACKEND", "file") or "file"
    if queue_backend == "sqlite":
        env["BIGCLAW_QUEUE_SQLITE_PATH"] = str(state_dir / "queue.db")
    elif queue_backend == "file":
        env["BIGCLAW_QUEUE_FILE"] = str(state_dir / "queue.json")
    env["BIGCLAW_AUDIT_LOG_PATH"] = str(state_dir / "audit.jsonl")
    base_url, http_addr = reserve_local_base_url()
    env["BIGCLAW_HTTP_ADDR"] = http_addr
    return env, base_url, state_dir


def start_bigclawd(go_root, env):
    log_path = pathlib.Path(tempfile.mkstemp(prefix="bigclawd-e2e-", suffix=".log")[1])
    log_file = log_path.open("w")
    process = subprocess.Popen(["go", "run", "./cmd/bigclawd"], cwd=go_root, stdout=log_file, stderr=subprocess.STDOUT, env=env)
    return process, log_path


def submit_task(base_url, task):
    payload = http_json(base_url + "/tasks", method="POST", payload=task)
    return payload["task"]


def fetch_status(base_url, task_id):
    return http_json(base_url + f"/tasks/{task_id}")


def fetch_events(base_url, task_id):
    return http_json(base_url + f"/events?task_id={task_id}&limit=100")["events"]


def terminal(state):
    return state in {"succeeded", "dead_letter", "cancelled", "failed"}


def write_report(go_root, report_path, payload):
    output_path = pathlib.Path(go_root) / report_path
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n")


def main():
    parser = argparse.ArgumentParser(description="Submit and verify a BigClaw task end-to-end")
    parser.add_argument("--executor", required=True, choices=["local", "kubernetes", "ray"])
    parser.add_argument("--title", required=True)
    parser.add_argument("--entrypoint", required=True)
    parser.add_argument("--image", default="")
    parser.add_argument("--base-url", default=os.environ.get("BIGCLAW_ADDR", "http://127.0.0.1:8080"))
    parser.add_argument("--go-root", default=str(pathlib.Path(__file__).resolve().parents[2]))
    parser.add_argument("--timeout-seconds", type=int, default=180)
    parser.add_argument("--poll-interval", type=float, default=1.0)
    parser.add_argument("--runtime-env-json", default="")
    parser.add_argument("--metadata-json", default="")
    parser.add_argument("--report-path", default="")
    parser.add_argument("--autostart", action="store_true")
    args = parser.parse_args()

    process = None
    log_path = None
    state_dir = None
    active_base_url = args.base_url
    report = None
    try:
        if args.autostart:
            try:
                wait_for_health(active_base_url, attempts=2, interval=0.2)
            except Exception:
                env, active_base_url, state_dir = build_autostart_env()
                process, log_path = start_bigclawd(args.go_root, env)
                wait_for_health(active_base_url)
        else:
            wait_for_health(active_base_url)

        task_id = f"{args.executor}-smoke-{int(time.time())}"
        task = {
            "id": task_id,
            "title": args.title,
            "required_executor": args.executor,
            "entrypoint": args.entrypoint,
            "execution_timeout_seconds": args.timeout_seconds,
            "metadata": {"smoke_test": "true", "executor": args.executor},
        }
        if args.image:
            task["container_image"] = args.image
        if args.runtime_env_json:
            task["runtime_env"] = json.loads(args.runtime_env_json)
        if args.metadata_json:
            task["metadata"].update(json.loads(args.metadata_json))

        submitted = submit_task(active_base_url, task)
        deadline = time.time() + args.timeout_seconds
        while time.time() < deadline:
            status = fetch_status(active_base_url, submitted["id"])
            state = status.get("state")
            if terminal(state):
                report = {
                    "autostarted": process is not None,
                    "base_url": active_base_url,
                    "task": submitted,
                    "status": status,
                    "events": fetch_events(active_base_url, submitted["id"]),
                }
                if state_dir is not None:
                    report["state_dir"] = str(state_dir)
                if log_path is not None:
                    report["service_log"] = str(log_path)
                if args.report_path:
                    write_report(args.go_root, args.report_path, report)
                print(json.dumps(report, ensure_ascii=False, indent=2))
                return 0 if state == "succeeded" else 1
            time.sleep(args.poll_interval)

        report = {
            "autostarted": process is not None,
            "base_url": active_base_url,
            "task": submitted,
            "status": fetch_status(active_base_url, submitted["id"]),
            "events": fetch_events(active_base_url, submitted["id"]),
            "error": "timeout waiting for terminal state",
        }
        if state_dir is not None:
            report["state_dir"] = str(state_dir)
        if log_path is not None:
            report["service_log"] = str(log_path)
        if args.report_path:
            write_report(args.go_root, args.report_path, report)
        print(json.dumps(report, ensure_ascii=False, indent=2), file=sys.stderr)
        return 1
    finally:
        if process is not None:
            process.terminate()
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill()
            if log_path and log_path.exists():
                print(f"bigclawd log: {log_path}", file=sys.stderr)


if __name__ == "__main__":
    sys.exit(main())
