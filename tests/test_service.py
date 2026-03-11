import json
import threading
import urllib.request
from pathlib import Path

from bigclaw.service import create_server


def _get(url: str) -> str:
    with urllib.request.urlopen(url, timeout=3) as resp:
        return resp.read().decode("utf-8")


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
        assert monitoring.request_total >= 3
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=2)
