import json
import threading
import urllib.request
from pathlib import Path

from bigclaw.service import RepoGovernanceEnforcer, RepoGovernancePolicy, create_server


def _get(url: str) -> str:
    with urllib.request.urlopen(url, timeout=3) as resp:
        return resp.read().decode("utf-8")


def test_repo_governance_enforcer_blocks_quota_and_sidecar_failures():
    enforcer = RepoGovernanceEnforcer(
        RepoGovernancePolicy(max_bundle_bytes=10, max_push_per_hour=1, max_diff_per_hour=1, sidecar_required=True)
    )

    ok = enforcer.evaluate(action="push", bundle_bytes=8, sidecar_available=True)
    assert ok.allowed is True

    too_large = enforcer.evaluate(action="push", bundle_bytes=12, sidecar_available=True)
    assert too_large.allowed is False
    assert too_large.mode == "blocked"

    over_quota = enforcer.evaluate(action="push", bundle_bytes=8, sidecar_available=True)
    assert over_quota.allowed is False
    assert "quota" in over_quota.reason

    degraded = enforcer.evaluate(action="diff", sidecar_available=False)
    assert degraded.allowed is False
    assert degraded.mode == "degraded"


def test_server_entry_health_metrics(tmp_path: Path):
    (tmp_path / "index.html").write_text("<h1>ok</h1>", encoding="utf-8")

    server, monitoring = create_server(host="127.0.0.1", port=0, directory=str(tmp_path))
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        host, port = server.server_address
        body = _get(f"http://{host}:{port}/")
        assert "ok" in body

        health = json.loads(_get(f"http://{host}:{port}/health"))
        assert health["status"] == "ok"
        assert health["request_total"] >= 1

        metrics = _get(f"http://{host}:{port}/metrics")
        assert "bigclaw_http_requests_total" in metrics
        assert "bigclaw_uptime_seconds" in metrics

        metrics_json = json.loads(_get(f"http://{host}:{port}/metrics.json"))
        assert "bigclaw_http_requests_total" in metrics_json
        assert "recent_requests" in metrics_json
        assert "rolling_5m" in metrics_json
        assert "bigclaw_http_error_rate" in metrics_json
        assert "health_summary" in metrics_json

        alerts = json.loads(_get(f"http://{host}:{port}/alerts"))
        assert alerts["level"] in {"ok", "warn", "critical"}
        assert "error_rate" in alerts

        monitor_html = _get(f"http://{host}:{port}/monitor")
        assert "BigClaw Monitor" in monitor_html
        assert "Requests" in monitor_html
        assert "Error Rate" in monitor_html
        assert "Auto refresh every 5s" in monitor_html

        assert monitoring.request_total >= 6
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=2)
