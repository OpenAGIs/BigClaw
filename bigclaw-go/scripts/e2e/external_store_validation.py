#!/usr/bin/env python3
from __future__ import annotations
import argparse
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
from datetime import datetime, timedelta, timezone


REPLAY_TASK_ID = "external-store-smoke-task"
REPLAY_TRACE_ID = "external-store-smoke-trace"
RETENTION_TASK_ID = "external-store-retention-task"
RETENTION_TRACE_ID = "external-store-retention-trace"
CHECKPOINT_SUBSCRIBER_ID = "subscriber-external-store"
LEASE_GROUP_ID = "group-external-store"
LEASE_SUBSCRIBER_ID = "subscriber-external-store"


def utc_now() -> datetime:
    return datetime.now(timezone.utc)


def to_rfc3339(value: datetime) -> str:
    return value.astimezone(timezone.utc).isoformat().replace("+00:00", "Z")


class HTTPStatusError(RuntimeError):
    def __init__(self, status_code: int, body: str):
        self.status_code = status_code
        self.body = body
        super().__init__(f"HTTP {status_code}: {body}")


def http_json(url, method="GET", payload=None, timeout=10):
    data = None
    headers = {"Content-Type": "application/json"}
    if payload is not None:
        data = json.dumps(payload).encode("utf-8")
    request = urllib.request.Request(url, data=data, method=method, headers=headers)
    try:
        with urllib.request.urlopen(request, timeout=timeout) as response:
            return json.loads(response.read().decode("utf-8"))
    except urllib.error.HTTPError as exc:
        body = exc.read().decode("utf-8", errors="replace")
        raise HTTPStatusError(exc.code, body) from exc


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


def build_node_env(*, state_dir: pathlib.Path, queue_backend: str, service_name: str, event_log_sqlite_path: pathlib.Path | None = None, event_log_remote_url: str = "", subscriber_lease_sqlite_path: pathlib.Path | None = None, event_retention: str = ""):
    env = os.environ.copy()
    env["BIGCLAW_QUEUE_BACKEND"] = queue_backend
    if queue_backend == "sqlite":
        env["BIGCLAW_QUEUE_SQLITE_PATH"] = str(state_dir / "queue.db")
    elif queue_backend == "file":
        env["BIGCLAW_QUEUE_FILE"] = str(state_dir / "queue.json")
    env["BIGCLAW_AUDIT_LOG_PATH"] = str(state_dir / "audit.jsonl")
    env["BIGCLAW_SERVICE_NAME"] = service_name
    env["BIGCLAW_BOOTSTRAP_TASKS"] = "0"
    env["BIGCLAW_MAX_CONCURRENT_RUNS"] = "2"
    if event_log_sqlite_path is not None:
        env["BIGCLAW_EVENT_LOG_SQLITE_PATH"] = str(event_log_sqlite_path)
    else:
        env.pop("BIGCLAW_EVENT_LOG_SQLITE_PATH", None)
    if event_log_remote_url:
        env["BIGCLAW_EVENT_LOG_REMOTE_URL"] = event_log_remote_url
    else:
        env.pop("BIGCLAW_EVENT_LOG_REMOTE_URL", None)
    if subscriber_lease_sqlite_path is not None:
        env["BIGCLAW_SUBSCRIBER_LEASE_SQLITE_PATH"] = str(subscriber_lease_sqlite_path)
    else:
        env.pop("BIGCLAW_SUBSCRIBER_LEASE_SQLITE_PATH", None)
    if event_retention:
        env["BIGCLAW_EVENT_RETENTION"] = event_retention
    else:
        env.pop("BIGCLAW_EVENT_RETENTION", None)
    base_url, http_addr = reserve_local_base_url()
    env["BIGCLAW_HTTP_ADDR"] = http_addr
    return env, base_url


def start_bigclawd(go_root: pathlib.Path, env: dict[str, str], log_name: str):
    log_path = go_root / "docs" / "reports" / f"{log_name}.log"
    log_path.parent.mkdir(parents=True, exist_ok=True)
    log_file = log_path.open("w")
    process = subprocess.Popen(["go", "run", "./cmd/bigclawd"], cwd=go_root, stdout=log_file, stderr=subprocess.STDOUT, env=env)
    return process, log_file, log_path


def terminate_process(process, log_file):
    if process is None:
        return
    process.terminate()
    try:
        process.wait(timeout=5)
    except subprocess.TimeoutExpired:
        process.kill()
        process.wait(timeout=5)
    log_file.close()


def submit_task(base_url: str, task: dict):
    return http_json(base_url + "/tasks", method="POST", payload=task)["task"]


def fetch_status(base_url: str, task_id: str):
    return http_json(base_url + f"/tasks/{task_id}")


def fetch_events_payload(base_url: str, query: str):
    return http_json(base_url + "/events?" + query)


def terminal(state: str) -> bool:
    return state in {"succeeded", "dead_letter", "cancelled", "failed"}


def wait_for_task(base_url: str, task_id: str, timeout_seconds: int, poll_interval: float):
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        status = fetch_status(base_url, task_id)
        if terminal(status.get("state", "")):
            return status
        time.sleep(poll_interval)
    raise RuntimeError(f"task {task_id} did not reach terminal state before timeout")


def check(condition: bool, message: str):
    if not condition:
        raise RuntimeError(message)


def write_report(go_root: pathlib.Path, report_path: str, payload: dict):
    output_path = go_root / report_path
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def build_backend_matrix(*, replay_backend: str, retention_boundary_visible: bool):
    return {
        "status_definitions": {
            "live_validated": "This checked-in repo-native lane executed the backend path and captured evidence.",
            "not_configured": "The backend lane is intentionally not configured in the current runtime proof and remains a placeholder.",
            "contract_only": "Only the rollout contract defines the expected backend semantics today.",
        },
        "summary": {
            "live_validated_lanes": 1,
            "not_configured_lanes": 1,
            "contract_only_lanes": 1,
        },
        "lanes": [
            {
                "backend": "http_remote_service",
                "role": "runtime_event_log",
                "validation_status": "live_validated",
                "configuration_state": "configured",
                "proof_kind": "repo_native_e2e",
                "replay_backend": replay_backend,
                "checkpoint_backend": "http_remote_service",
                "retention_boundary_visible": retention_boundary_visible,
                "takeover_backend": "sqlite_shared_lease",
                "report_links": [
                    "docs/e2e-validation.md",
                    "docs/reports/replay-retention-semantics-report.md",
                    "docs/reports/epic-closure-readiness-report.md",
                ],
                "notes": "Replay, checkpoint state, and retention-boundary visibility are validated through the remote HTTP event-log service boundary.",
            },
            {
                "backend": "broker_replicated",
                "role": "runtime_event_log",
                "validation_status": "not_configured",
                "configuration_state": "not_configured",
                "proof_kind": "placeholder",
                "reason": "not_configured",
                "report_links": [
                    "docs/reports/broker-failover-fault-injection-validation-pack.md",
                    "docs/reports/broker-failover-stub-report.json",
                ],
                "notes": "The checked-in repo-native external-store lane does not start a live broker-backed event-log adapter yet.",
            },
            {
                "backend": "quorum_replicated",
                "role": "runtime_event_log",
                "validation_status": "contract_only",
                "configuration_state": "contract_documented",
                "proof_kind": "placeholder",
                "reason": "contract_only",
                "report_links": [
                    "docs/reports/replicated-event-log-durability-rollout-contract.md",
                    "docs/reports/replicated-broker-durability-rollout-spike.md",
                ],
                "notes": "Quorum-backed durability expectations are documented, but no executable quorum lane or adapter is checked in.",
            },
        ],
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="Validate the remote external-store event-log lane")
    parser.add_argument("--go-root", default=str(pathlib.Path(__file__).resolve().parents[2]))
    parser.add_argument("--report-path", default="docs/reports/external-store-validation-report.json")
    parser.add_argument("--timeout-seconds", type=int, default=120)
    parser.add_argument("--poll-interval", type=float, default=0.5)
    parser.add_argument("--retention", default="2s")
    args = parser.parse_args()

    go_root = pathlib.Path(args.go_root)
    runtime_root = pathlib.Path(tempfile.mkdtemp(prefix="bigclaw-external-store-"))
    service_state = runtime_root / "service"
    node_a_state = runtime_root / "node-a"
    node_b_state = runtime_root / "node-b"
    for path in (service_state, node_a_state, node_b_state):
        path.mkdir(parents=True, exist_ok=True)
    shared_lease_db = runtime_root / "shared-subscriber-leases.db"

    service_env, service_base = build_node_env(
        state_dir=service_state,
        queue_backend="file",
        service_name="bigclawd-external-store-service",
        event_log_sqlite_path=service_state / "event-log.db",
        event_retention=args.retention,
    )
    service_process = service_log_file = service_log_path = None
    node_a_process = node_a_log_file = node_a_log_path = None
    node_b_process = node_b_log_file = node_b_log_path = None

    try:
        service_process, service_log_file, service_log_path = start_bigclawd(go_root, service_env, "external-store-service")
        wait_for_health(service_base)
        remote_event_log_url = service_base + "/internal/events/log"

        node_a_env, node_a_base = build_node_env(
            state_dir=node_a_state,
            queue_backend="sqlite",
            service_name="bigclawd-external-store-node-a",
            event_log_remote_url=remote_event_log_url,
            subscriber_lease_sqlite_path=shared_lease_db,
        )
        node_b_env, node_b_base = build_node_env(
            state_dir=node_b_state,
            queue_backend="sqlite",
            service_name="bigclawd-external-store-node-b",
            event_log_remote_url=remote_event_log_url,
            subscriber_lease_sqlite_path=shared_lease_db,
        )
        node_a_process, node_a_log_file, node_a_log_path = start_bigclawd(go_root, node_a_env, "external-store-node-a")
        node_b_process, node_b_log_file, node_b_log_path = start_bigclawd(go_root, node_b_env, "external-store-node-b")
        wait_for_health(node_a_base)
        wait_for_health(node_b_base)

        task = {
            "id": REPLAY_TASK_ID,
            "trace_id": REPLAY_TRACE_ID,
            "title": "External-store remote event-log smoke",
            "required_executor": "local",
            "entrypoint": "echo hello from remote event log",
            "execution_timeout_seconds": args.timeout_seconds,
            "metadata": {
                "scenario": "external-store-validation",
                "lane": "remote_http_event_log",
            },
        }
        submitted = submit_task(node_a_base, task)
        final_status = wait_for_task(node_a_base, submitted["id"], args.timeout_seconds, args.poll_interval)
        replay_payload = fetch_events_payload(node_a_base, f"task_id={submitted['id']}&limit=100")
        replay_events = replay_payload.get("events", [])
        check(final_status.get("state") == "succeeded", f"task did not succeed: {final_status}")
        check(replay_payload.get("backend") == "http", f"expected remote replay backend http, got {replay_payload.get('backend')}")
        check(replay_payload.get("durable") is True, f"expected durable replay payload, got {replay_payload}")
        check(len(replay_events) >= 3, f"expected replay events for smoke task, got {replay_events}")
        latest_event_id = replay_events[-1]["id"]

        checkpoint_payload = http_json(
            node_a_base + f"/stream/events/checkpoints/{CHECKPOINT_SUBSCRIBER_ID}",
            method="POST",
            payload={"event_id": latest_event_id},
        )
        checkpoint_read = http_json(node_a_base + f"/stream/events/checkpoints/{CHECKPOINT_SUBSCRIBER_ID}")
        http_json(node_a_base + f"/stream/events/checkpoints/{CHECKPOINT_SUBSCRIBER_ID}", method="DELETE")
        checkpoint_history = http_json(node_a_base + f"/stream/events/checkpoints/{CHECKPOINT_SUBSCRIBER_ID}/history?limit=10")
        check(checkpoint_payload["checkpoint"]["event_id"] == latest_event_id, f"expected checkpoint event {latest_event_id}, got {checkpoint_payload}")
        check(checkpoint_read["checkpoint"]["event_id"] == latest_event_id, f"expected checkpoint readback {latest_event_id}, got {checkpoint_read}")
        check(len(checkpoint_history.get("history", [])) >= 1, f"expected checkpoint reset history, got {checkpoint_history}")

        retention_now = utc_now()
        http_json(
            remote_event_log_url + "/record",
            method="POST",
            payload={
                "id": "evt-external-retention-old",
                "type": "task.queued",
                "task_id": RETENTION_TASK_ID,
                "trace_id": RETENTION_TRACE_ID,
                "timestamp": to_rfc3339(retention_now - timedelta(seconds=10)),
            },
        )
        http_json(
            remote_event_log_url + "/record",
            method="POST",
            payload={
                "id": "evt-external-retention-new",
                "type": "task.started",
                "task_id": RETENTION_TASK_ID,
                "trace_id": RETENTION_TRACE_ID,
                "timestamp": to_rfc3339(retention_now),
            },
        )
        retention_payload = fetch_events_payload(node_a_base, f"trace_id={RETENTION_TRACE_ID}&limit=10")
        retention_events = retention_payload.get("events", [])
        retention_watermark = retention_payload.get("retention_watermark", {})
        check(len(retention_events) == 1 and retention_events[0]["id"] == "evt-external-retention-new", f"expected only retained external-store event, got {retention_events}")
        check(retention_watermark.get("history_truncated") is True, f"expected history truncation, got {retention_watermark}")
        check(retention_watermark.get("persisted_boundary") is True, f"expected persisted boundary, got {retention_watermark}")
        check(retention_watermark.get("trimmed_through_event_id") == "evt-external-retention-old", f"unexpected trimmed boundary: {retention_watermark}")

        lease_a = http_json(
            node_a_base + "/subscriber-groups/leases",
            method="POST",
            payload={
                "group_id": LEASE_GROUP_ID,
                "subscriber_id": LEASE_SUBSCRIBER_ID,
                "consumer_id": "node-a",
                "ttl_seconds": 2,
            },
        )
        checkpoint_a = http_json(
            node_a_base + "/subscriber-groups/checkpoints",
            method="POST",
            payload={
                "group_id": LEASE_GROUP_ID,
                "subscriber_id": LEASE_SUBSCRIBER_ID,
                "consumer_id": "node-a",
                "lease_token": lease_a["lease"]["lease_token"],
                "lease_epoch": lease_a["lease"]["lease_epoch"],
                "checkpoint_offset": 11,
                "checkpoint_event_id": latest_event_id,
            },
        )
        conflict_status = None
        conflict_body = ""
        try:
            http_json(
                node_b_base + "/subscriber-groups/leases",
                method="POST",
                payload={
                    "group_id": LEASE_GROUP_ID,
                    "subscriber_id": LEASE_SUBSCRIBER_ID,
                    "consumer_id": "node-b",
                    "ttl_seconds": 2,
                },
            )
        except HTTPStatusError as exc:
            conflict_status = exc.status_code
            conflict_body = exc.body
        check(conflict_status == 409, f"expected active leader conflict 409, got {conflict_status} {conflict_body}")
        time.sleep(2.2)
        lease_b = http_json(
            node_b_base + "/subscriber-groups/leases",
            method="POST",
            payload={
                "group_id": LEASE_GROUP_ID,
                "subscriber_id": LEASE_SUBSCRIBER_ID,
                "consumer_id": "node-b",
                "ttl_seconds": 2,
            },
        )
        stale_status = None
        stale_body = ""
        try:
            http_json(
                node_a_base + "/subscriber-groups/checkpoints",
                method="POST",
                payload={
                    "group_id": LEASE_GROUP_ID,
                    "subscriber_id": LEASE_SUBSCRIBER_ID,
                    "consumer_id": "node-a",
                    "lease_token": lease_a["lease"]["lease_token"],
                    "lease_epoch": lease_a["lease"]["lease_epoch"],
                    "checkpoint_offset": 12,
                    "checkpoint_event_id": latest_event_id,
                },
            )
        except HTTPStatusError as exc:
            stale_status = exc.status_code
            stale_body = exc.body
        check(stale_status == 409, f"expected stale writer conflict 409, got {stale_status} {stale_body}")
        checkpoint_b = http_json(
            node_b_base + "/subscriber-groups/checkpoints",
            method="POST",
            payload={
                "group_id": LEASE_GROUP_ID,
                "subscriber_id": LEASE_SUBSCRIBER_ID,
                "consumer_id": "node-b",
                "lease_token": lease_b["lease"]["lease_token"],
                "lease_epoch": lease_b["lease"]["lease_epoch"],
                "checkpoint_offset": 15,
                "checkpoint_event_id": latest_event_id,
            },
        )
        lease_status = http_json(node_b_base + f"/subscriber-groups/{LEASE_GROUP_ID}/subscribers/{LEASE_SUBSCRIBER_ID}")

        report = {
            "generated_at": to_rfc3339(utc_now()),
            "ticket": "BIG-PAR-102",
            "title": "External-store validation backend matrix and broker placeholders",
            "status": "validated",
            "lane": {
                "service_backend": "sqlite_event_log_service",
                "runtime_event_log_backend": "http_remote_service",
                "queue_backend": "sqlite",
                "subscriber_lease_backend": "sqlite_shared",
                "retention": args.retention,
                "node_count": 3,
            },
            "summary": {
                "task_succeeded": final_status.get("state") == "succeeded",
                "remote_replay_backend": replay_payload.get("backend"),
                "replay_event_count": len(replay_events),
                "checkpoint_acknowledged": checkpoint_payload["checkpoint"]["event_id"] == latest_event_id,
                "checkpoint_reset_recorded": len(checkpoint_history.get("history", [])) >= 1,
                "retention_boundary_visible": retention_watermark.get("history_truncated", False),
                "retained_event_count": len(retention_events),
                "takeover_conflict_rejected": conflict_status == 409,
                "takeover_after_expiry": lease_b["lease"]["lease_epoch"] == 2,
                "stale_writer_rejected": stale_status == 409,
            },
            "backend_matrix": build_backend_matrix(
                replay_backend=replay_payload.get("backend"),
                retention_boundary_visible=retention_watermark.get("history_truncated", False),
            ),
            "replay_validation": {
                "task_id": submitted["id"],
                "trace_id": submitted["trace_id"],
                "backend": replay_payload.get("backend"),
                "durable": replay_payload.get("durable"),
                "latest_event_id": latest_event_id,
                "latest_event_type": replay_events[-1]["type"],
            },
            "checkpoint_validation": {
                "subscriber_id": CHECKPOINT_SUBSCRIBER_ID,
                "acked_event_id": checkpoint_payload["checkpoint"]["event_id"],
                "checkpoint_event_id": checkpoint_read["checkpoint"]["event_id"],
                "reset_history_entries": len(checkpoint_history.get("history", [])),
            },
            "retention_validation": {
                "trace_id": RETENTION_TRACE_ID,
                "history_truncated": retention_watermark.get("history_truncated"),
                "persisted_boundary": retention_watermark.get("persisted_boundary"),
                "trimmed_through_event_id": retention_watermark.get("trimmed_through_event_id"),
                "oldest_event_id": retention_watermark.get("oldest_event_id"),
                "newest_event_id": retention_watermark.get("newest_event_id"),
            },
            "takeover_validation": {
                "group_id": LEASE_GROUP_ID,
                "subscriber_id": LEASE_SUBSCRIBER_ID,
                "initial_consumer": lease_a["lease"]["consumer_id"],
                "initial_epoch": lease_a["lease"]["lease_epoch"],
                "initial_checkpoint_offset": checkpoint_a["lease"]["checkpoint_offset"],
                "conflict_status": conflict_status,
                "takeover_consumer": lease_b["lease"]["consumer_id"],
                "takeover_epoch": lease_b["lease"]["lease_epoch"],
                "stale_writer_status": stale_status,
                "final_checkpoint_offset": checkpoint_b["lease"]["checkpoint_offset"],
                "final_lease_consumer": lease_status["lease"]["consumer_id"],
            },
            "artifacts": {
                "e2e_doc": "docs/e2e-validation.md",
                "retention_report": "docs/reports/replay-retention-semantics-report.md",
                "epic_report": "docs/reports/epic-closure-readiness-report.md",
            },
            "limitations": [
                "The backend matrix marks the HTTP remote-service lane as live validated, while broker-backed and quorum-backed durability remain explicit placeholders.",
                "Event replay and checkpoint storage are validated through the remote HTTP event-log service, while shared-queue coordination and takeover still rely on the current shared SQLite lease store.",
            ],
        }
        write_report(go_root, args.report_path, report)
        print(json.dumps(report, ensure_ascii=False, indent=2))
        return 0
    finally:
        terminate_process(node_b_process, node_b_log_file)
        terminate_process(node_a_process, node_a_log_file)
        terminate_process(service_process, service_log_file)


if __name__ == "__main__":
    sys.exit(main())
