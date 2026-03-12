from __future__ import annotations

import json
import os
import threading
import time
from collections import deque
from dataclasses import dataclass, field
from http import HTTPStatus
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
from typing import Deque, Dict, List


@dataclass
class ServerMonitoring:
    start_time: float = field(default_factory=time.time)
    request_total: int = 0
    error_total: int = 0
    recent_requests: Deque[Dict[str, str]] = field(default_factory=lambda: deque(maxlen=20))
    minute_buckets: Deque[Dict[str, int]] = field(default_factory=lambda: deque(maxlen=5))
    _lock: threading.Lock = field(default_factory=threading.Lock)

    def _now_minute(self) -> int:
        return int(time.time() // 60)

    def _ensure_bucket(self, minute: int) -> Dict[str, int]:
        if not self.minute_buckets or self.minute_buckets[-1]["minute"] != minute:
            self.minute_buckets.append({"minute": minute, "requests": 0, "errors": 0})
        return self.minute_buckets[-1]

    def record(self, path: str, status: int) -> None:
        ts = time.time()
        minute = int(ts // 60)
        with self._lock:
            self.request_total += 1
            if status >= 400:
                self.error_total += 1
            self.recent_requests.append({"path": path, "status": str(status), "ts": f"{ts:.3f}"})
            bucket = self._ensure_bucket(minute)
            bucket["requests"] += 1
            if status >= 400:
                bucket["errors"] += 1

    def _rolling(self) -> List[Dict[str, int]]:
        return [dict(bucket) for bucket in self.minute_buckets]

    def health_payload(self) -> Dict[str, object]:
        uptime = max(0.0, time.time() - self.start_time)
        with self._lock:
            return {
                "status": "ok",
                "uptime_seconds": round(uptime, 3),
                "request_total": self.request_total,
                "error_total": self.error_total,
                "recent_requests": list(self.recent_requests),
                "rolling_5m": self._rolling(),
            }

    def metrics_payload(self) -> Dict[str, object]:
        uptime = max(0.0, time.time() - self.start_time)
        with self._lock:
            error_rate = (self.error_total / self.request_total) if self.request_total else 0.0
            summary = "healthy"
            if error_rate >= 0.2:
                summary = "critical"
            elif error_rate >= 0.05:
                summary = "degraded"
            return {
                "bigclaw_uptime_seconds": round(uptime, 3),
                "bigclaw_http_requests_total": self.request_total,
                "bigclaw_http_errors_total": self.error_total,
                "bigclaw_http_error_rate": round(error_rate, 4),
                "health_summary": summary,
                "recent_requests": list(self.recent_requests),
                "rolling_5m": self._rolling(),
            }

    def alerts_payload(self) -> Dict[str, object]:
        metrics = self.metrics_payload()
        error_rate = float(metrics["bigclaw_http_error_rate"])
        level = "ok"
        message = "System healthy"
        if error_rate >= 0.2:
            level = "critical"
            message = "High HTTP error rate detected"
        elif error_rate >= 0.05:
            level = "warn"
            message = "Elevated HTTP error rate detected"
        return {
            "level": level,
            "message": message,
            "error_rate": error_rate,
            "request_total": metrics["bigclaw_http_requests_total"],
            "error_total": metrics["bigclaw_http_errors_total"],
        }

    def metrics_text(self) -> str:
        uptime = max(0.0, time.time() - self.start_time)
        with self._lock:
            lines = [
                "# HELP bigclaw_uptime_seconds process uptime in seconds",
                "# TYPE bigclaw_uptime_seconds gauge",
                f"bigclaw_uptime_seconds {uptime:.3f}",
                "# HELP bigclaw_http_requests_total total HTTP requests",
                "# TYPE bigclaw_http_requests_total counter",
                f"bigclaw_http_requests_total {self.request_total}",
                "# HELP bigclaw_http_errors_total total HTTP error responses (>=400)",
                "# TYPE bigclaw_http_errors_total counter",
                f"bigclaw_http_errors_total {self.error_total}",
            ]
            for bucket in self.minute_buckets:
                lines.append(
                    f"bigclaw_http_requests_minute{{minute=\"{bucket['minute']}\"}} {bucket['requests']}"
                )
                lines.append(
                    f"bigclaw_http_errors_minute{{minute=\"{bucket['minute']}\"}} {bucket['errors']}"
                )
            return "\n".join(lines) + "\n"


def _monitor_page(stats: Dict[str, object]) -> str:
    rows = "".join(
        f"<tr><td>{item['ts']}</td><td>{item['path']}</td><td>{item['status']}</td></tr>"
        for item in stats["recent_requests"]
    ) or "<tr><td colspan='3'>No requests yet</td></tr>"

    rolling_rows = "".join(
        f"<tr><td>{bucket['minute']}</td><td>{bucket['requests']}</td><td>{bucket['errors']}</td></tr>"
        for bucket in stats.get("rolling_5m", [])
    ) or "<tr><td colspan='3'>No rolling data yet</td></tr>"

    return f"""<!doctype html>
<html>
<head>
  <meta charset='utf-8'>
  <meta name='viewport' content='width=device-width, initial-scale=1'>
  <title>BigClaw Monitor</title>
  <style>
    body {{ font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; margin: 0; background:#f6f7fb; color:#0f172a; }}
    .container {{ max-width: 1040px; margin: 24px auto; padding: 0 16px; }}
    .cards {{ display:grid; grid-template-columns: repeat(auto-fit,minmax(180px,1fr)); gap:12px; }}
    .card {{ background:#fff; border:1px solid #e2e8f0; border-radius:12px; padding:12px; }}
    .label {{ color:#64748b; font-size:12px; }}
    .value {{ font-size:24px; font-weight:700; margin-top:4px; }}
    table {{ width:100%; border-collapse: collapse; background:#fff; border:1px solid #e2e8f0; border-radius:12px; overflow:hidden; }}
    th,td {{ border-bottom:1px solid #e2e8f0; padding:8px 10px; text-align:left; font-size:13px; }}
    h1,h2 {{ margin: 0 0 10px; }}
    section {{ margin-top: 16px; }}
    .muted {{ color:#64748b; font-size:12px; }}
  </style>
</head>
<body>
  <div class='container'>
    <h1>BigClaw Monitor</h1>
    <p class='muted'>Auto refresh every 5s · endpoint: /metrics.json</p>
    <div class='cards'>
      <div class='card'><div class='label'>Uptime (s)</div><div class='value' id='uptime'>{stats['bigclaw_uptime_seconds']}</div></div>
      <div class='card'><div class='label'>Requests</div><div class='value' id='requests'>{stats['bigclaw_http_requests_total']}</div></div>
      <div class='card'><div class='label'>Errors</div><div class='value' id='errors'>{stats['bigclaw_http_errors_total']}</div></div>
      <div class='card'><div class='label'>Error Rate</div><div class='value' id='error-rate'>{stats['bigclaw_http_error_rate']}</div></div>
      <div class='card'><div class='label'>Health</div><div class='value' id='health-summary'>{stats['health_summary']}</div></div>
    </div>

    <section>
      <h2>Rolling 5m</h2>
      <table id='rolling-table'>
        <thead><tr><th>minute</th><th>requests</th><th>errors</th></tr></thead>
        <tbody>{rolling_rows}</tbody>
      </table>
    </section>

    <section>
      <h2>Recent Requests</h2>
      <table id='recent-table'>
        <thead><tr><th>ts</th><th>path</th><th>status</th></tr></thead>
        <tbody>{rows}</tbody>
      </table>
    </section>
  </div>
  <script>
    async function refreshMonitor() {{
      try {{
        const res = await fetch('/metrics.json', {{ cache: 'no-store' }});
        const data = await res.json();
        document.getElementById('uptime').textContent = data.bigclaw_uptime_seconds;
        document.getElementById('requests').textContent = data.bigclaw_http_requests_total;
        document.getElementById('errors').textContent = data.bigclaw_http_errors_total;
        document.getElementById('error-rate').textContent = data.bigclaw_http_error_rate;
        document.getElementById('health-summary').textContent = data.health_summary;

        const rollingBody = document.querySelector('#rolling-table tbody');
        rollingBody.innerHTML = (data.rolling_5m || []).map((b) =>
          `<tr><td>${{b.minute}}</td><td>${{b.requests}}</td><td>${{b.errors}}</td></tr>`
        ).join('') || "<tr><td colspan='3'>No rolling data yet</td></tr>";

        const recentBody = document.querySelector('#recent-table tbody');
        recentBody.innerHTML = (data.recent_requests || []).map((r) =>
          `<tr><td>${{r.ts}}</td><td>${{r.path}}</td><td>${{r.status}}</td></tr>`
        ).join('') || "<tr><td colspan='3'>No requests yet</td></tr>";
      }} catch (e) {{
        console.error('monitor refresh failed', e);
      }}
    }}
    setInterval(refreshMonitor, 5000);
  </script>
</body>
</html>"""


def _handler_factory(*, directory: str, monitoring: ServerMonitoring):
    class BigClawHandler(SimpleHTTPRequestHandler):
        def __init__(self, *args, **kwargs):
            super().__init__(*args, directory=directory, **kwargs)

        def do_GET(self) -> None:  # noqa: N802
            if self.path == "/health":
                payload = monitoring.health_payload()
                body = json.dumps(payload).encode("utf-8")
                self.send_response(HTTPStatus.OK)
                self.send_header("Content-Type", "application/json; charset=utf-8")
                self.send_header("Content-Length", str(len(body)))
                self.end_headers()
                self.wfile.write(body)
                return

            if self.path == "/metrics":
                body = monitoring.metrics_text().encode("utf-8")
                self.send_response(HTTPStatus.OK)
                self.send_header("Content-Type", "text/plain; version=0.0.4")
                self.send_header("Content-Length", str(len(body)))
                self.end_headers()
                self.wfile.write(body)
                return

            if self.path == "/metrics.json":
                body = json.dumps(monitoring.metrics_payload()).encode("utf-8")
                self.send_response(HTTPStatus.OK)
                self.send_header("Content-Type", "application/json; charset=utf-8")
                self.send_header("Content-Length", str(len(body)))
                self.end_headers()
                self.wfile.write(body)
                return

            if self.path == "/monitor":
                html = _monitor_page(monitoring.metrics_payload())
                body = html.encode("utf-8")
                self.send_response(HTTPStatus.OK)
                self.send_header("Content-Type", "text/html; charset=utf-8")
                self.send_header("Content-Length", str(len(body)))
                self.end_headers()
                self.wfile.write(body)
                return

            if self.path == "/alerts":
                body = json.dumps(monitoring.alerts_payload()).encode("utf-8")
                self.send_response(HTTPStatus.OK)
                self.send_header("Content-Type", "application/json; charset=utf-8")
                self.send_header("Content-Length", str(len(body)))
                self.end_headers()
                self.wfile.write(body)
                return

            super().do_GET()

        def log_message(self, format: str, *args) -> None:
            super().log_message(format, *args)

        def send_response(self, code: int, message=None):  # type: ignore[override]
            super().send_response(code, message)
            path = getattr(self, "path", "-")
            monitoring.record(path, int(code))

    return BigClawHandler


def create_server(host: str = "127.0.0.1", port: int = 8008, directory: str = "."):
    directory = os.path.abspath(directory)
    monitoring = ServerMonitoring()
    handler = _handler_factory(directory=directory, monitoring=monitoring)
    server = ThreadingHTTPServer((host, port), handler)
    return server, monitoring


def run_server(host: str = "127.0.0.1", port: int = 8008, directory: str = ".") -> None:
    server, _ = create_server(host=host, port=port, directory=directory)
    print(f"BigClaw server running at http://{host}:{port} (dir={os.path.abspath(directory)})")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        server.server_close()


@dataclass
class RepoGovernancePolicy:
    max_bundle_bytes: int = 50 * 1024 * 1024
    max_push_per_hour: int = 20
    max_diff_per_hour: int = 120
    sidecar_required: bool = True


@dataclass
class RepoGovernanceResult:
    allowed: bool
    mode: str
    reason: str = ""


class RepoGovernanceEnforcer:
    def __init__(self, policy: RepoGovernancePolicy):
        self.policy = policy
        self.push_count = 0
        self.diff_count = 0

    def evaluate(self, *, action: str, bundle_bytes: int = 0, sidecar_available: bool = True) -> RepoGovernanceResult:
        if self.policy.sidecar_required and not sidecar_available:
            return RepoGovernanceResult(allowed=False, mode="degraded", reason="repo sidecar unavailable")

        if action == "push":
            if bundle_bytes > self.policy.max_bundle_bytes:
                return RepoGovernanceResult(allowed=False, mode="blocked", reason="bundle exceeds max size")
            if self.push_count >= self.policy.max_push_per_hour:
                return RepoGovernanceResult(allowed=False, mode="blocked", reason="push quota exceeded")
            self.push_count += 1
            return RepoGovernanceResult(allowed=True, mode="allow")

        if action == "diff":
            if self.diff_count >= self.policy.max_diff_per_hour:
                return RepoGovernanceResult(allowed=False, mode="blocked", reason="diff quota exceeded")
            self.diff_count += 1
            return RepoGovernanceResult(allowed=True, mode="allow")

        return RepoGovernanceResult(allowed=True, mode="allow")
