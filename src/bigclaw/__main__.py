from __future__ import annotations

import argparse
import json
import os
import shutil
import stat
import subprocess
import threading
import time
import warnings
from collections import deque
from dataclasses import asdict, dataclass, field
from http import HTTPStatus
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from typing import Any, Deque, Dict, Iterable, List, Sequence
from urllib.parse import urlparse

from .execution_contract import RepoSyncAudit
from .reports import render_repo_sync_audit_report, write_report


GO_MAINLINE_REPLACEMENT = "bigclaw-go/cmd/bigclawd/main.go"
LEGACY_MAINLINE_STATUS = (
    "bigclaw-go is the sole implementation mainline for active development; "
    "__main__.py remains migration-only compatibility scaffolding."
)


def warn_legacy_service_surface(surface: str = "python -m bigclaw serve") -> str:
    return warn_legacy_runtime_surface(surface, "go run ./bigclaw-go/cmd/bigclawd")


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
                lines.append(f"bigclaw_http_requests_minute{{minute=\"{bucket['minute']}\"}} {bucket['requests']}")
                lines.append(f"bigclaw_http_errors_minute{{minute=\"{bucket['minute']}\"}} {bucket['errors']}")
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
    warn_legacy_service_surface()
    server, _ = create_server(host=host, port=port, directory=directory)
    print(f"BigClaw server running at http://{host}:{port} (dir={os.path.abspath(directory)})")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        server.server_close()


def package_main() -> None:
    warn_legacy_runtime_surface("python -m bigclaw", "bash scripts/ops/bigclawctl or go run ./bigclaw-go/cmd/bigclawd")
    parser = argparse.ArgumentParser(prog="bigclaw", description="BigClaw developer utilities")
    sub = parser.add_subparsers(dest="command")

    serve = sub.add_parser("serve", help="Run legacy migration-only static web server")
    serve.add_argument("--host", default="127.0.0.1")
    serve.add_argument("--port", type=int, default=8008)
    serve.add_argument("--dir", default="reports")

    repo_sync_audit = sub.add_parser("repo-sync-audit", help="Render a repo sync and PR freshness audit report")
    repo_sync_audit.add_argument("--input", required=True, help="Path to JSON payload for RepoSyncAudit")
    repo_sync_audit.add_argument("--output", required=True, help="Path to markdown report output")

    args = parser.parse_args()

    if args.command == "serve":
        run_server(host=args.host, port=args.port, directory=args.dir)
        return

    if args.command == "repo-sync-audit":
        payload = json.loads(Path(args.input).read_text())
        audit = RepoSyncAudit.from_dict(payload)
        write_report(args.output, render_repo_sync_audit_report(audit))
        return

    parser.print_help()


class WorkspaceBootstrapError(RuntimeError):
    """Raised when the shared-worktree bootstrap flow cannot complete."""


class GitSyncError(RuntimeError):
    """Raised when repository sync automation cannot complete safely."""


@dataclass
class CacheBootstrapState:
    cache_root: str
    cache_key: str
    mirror_path: str
    seed_path: str
    mirror_created: bool
    seed_created: bool

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)


@dataclass
class WorkspaceBootstrapStatus:
    workspace: str
    branch: str
    cache_root: str
    cache_key: str
    mirror_path: str
    seed_path: str
    reused: bool
    cache_reused: bool
    clone_suppressed: bool
    mirror_created: bool = False
    seed_created: bool = False
    workspace_mode: str = "worktree_created"
    removed: bool = False

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)


@dataclass
class RepoSyncStatus:
    branch: str
    local_sha: str
    remote_sha: str
    dirty: bool
    remote_exists: bool
    synced: bool
    pushed: bool = False

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)


@dataclass
class CommandResult:
    stdout: str
    stderr: str
    returncode: int


CACHE_REMOTE = "cache"
BOOTSTRAP_BRANCH_PREFIX = "symphony"
DEFAULT_CACHE_BASE = Path("~/.cache/symphony/repos")
DEFAULT_CLI_CACHE_BASE = "~/.cache/symphony/repos"
EXECUTABLE_BITS = stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH
LEGACY_PYTHON_WRAPPER_NOTICE = (
    "Legacy Python operator wrapper: use scripts/ops/bigclawctl for the Go mainline. "
    "This Python path remains only as a compatibility shim during migration."
)
LEGACY_RUNTIME_GUIDANCE = (
    "bigclaw-go is the sole implementation mainline for active development; "
    "the legacy Python runtime surface remains migration-only."
)


def build_parser(
    description: str,
    default_repo_url: str,
    default_branch: str,
    default_cache_root: str | None,
    default_cache_base: str,
    default_cache_key: str | None,
) -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=description)
    parser.add_argument("command", choices=["bootstrap", "cleanup"])
    parser.add_argument("--workspace", default=".", help="Workspace path managed by Symphony.")
    parser.add_argument("--issue", default="", help="Linear issue identifier used for the bootstrap branch.")
    parser.add_argument("--repo-url", default=default_repo_url, help="Canonical remote repository URL.")
    parser.add_argument("--default-branch", default=default_branch, help="Default branch used as the bootstrap base.")
    parser.add_argument(
        "--cache-root",
        default=default_cache_root,
        help="Full cache root that contains mirror.git and seed. Overrides --cache-base/--cache-key.",
    )
    parser.add_argument(
        "--cache-base",
        default=default_cache_base,
        help="Base directory that stores per-repo cache roots.",
    )
    parser.add_argument(
        "--cache-key",
        default=default_cache_key,
        help="Optional stable key for the per-repo cache directory.",
    )
    parser.add_argument("--json", action="store_true", help="Emit machine-readable JSON output.")
    return parser


def emit(payload: dict, as_json: bool) -> None:
    if as_json:
        print(json.dumps(payload, ensure_ascii=False, indent=2))
        return
    for key, value in payload.items():
        print(f"{key}={value}")


def append_missing_flag(args: Sequence[str], flag: str, value: str) -> List[str]:
    flag_prefix = flag + "="
    if any(arg == flag or arg.startswith(flag_prefix) for arg in args):
        return list(args)
    return [*args, flag, value]


def legacy_runtime_message(surface: str, replacement: str) -> str:
    return f"{surface} is frozen for migration-only use. {LEGACY_RUNTIME_GUIDANCE} Use {replacement} instead."


def warn_legacy_runtime_surface(surface: str, replacement: str) -> str:
    message = legacy_runtime_message(surface, replacement)
    warnings.warn(message, DeprecationWarning, stacklevel=2)
    return message


def build_bigclawctl_exec_args(repo_root: Path, command: Iterable[str], forwarded: Sequence[str]) -> List[str]:
    return ["bash", str(repo_root / "scripts/ops/bigclawctl"), *command, *forwarded]


def repo_root_from_script(script_path: str) -> Path:
    return Path(script_path).resolve().parents[2]


def run_bigclawctl_shim(script_path: str, command: Iterable[str], forwarded: Sequence[str]) -> int:
    repo_root = repo_root_from_script(script_path)
    argv = build_bigclawctl_exec_args(repo_root, command, forwarded)
    return subprocess.call(argv, cwd=repo_root)


def build_workspace_bootstrap_args(repo_root: Path, forwarded: Sequence[str]) -> List[str]:
    args = list(forwarded)
    args = append_missing_flag(args, "--repo-url", "git@github.com:OpenAGIs/BigClaw.git")
    args = append_missing_flag(args, "--cache-key", "openagis-bigclaw")
    return build_bigclawctl_exec_args(repo_root, ["workspace", "bootstrap"], args)


def translate_workspace_validate_args(forwarded: Sequence[str]) -> List[str]:
    translated: List[str] = []
    i = 0
    while i < len(forwarded):
        arg = forwarded[i]
        if arg == "--report-file":
            translated.extend(["--report", forwarded[i + 1]])
            i += 2
            continue
        if arg.startswith("--report-file="):
            translated.append("--report=" + arg.split("=", 1)[1])
            i += 1
            continue
        if arg == "--no-cleanup":
            translated.append("--cleanup=false")
            i += 1
            continue
        if arg == "--issues":
            issues: List[str] = []
            i += 1
            while i < len(forwarded) and not forwarded[i].startswith("-"):
                issues.append(forwarded[i])
                i += 1
            translated.extend(["--issues", ",".join(issues)])
            continue
        translated.append(arg)
        i += 1
    return translated


def build_workspace_validate_args(repo_root: Path, forwarded: Sequence[str]) -> List[str]:
    return build_bigclawctl_exec_args(repo_root, ["workspace", "validate"], translate_workspace_validate_args(forwarded))


def build_github_sync_args(repo_root: Path, forwarded: Sequence[str]) -> List[str]:
    return build_bigclawctl_exec_args(repo_root, ["github-sync"], list(forwarded))


def build_refill_args(repo_root: Path, forwarded: Sequence[str]) -> List[str]:
    return build_bigclawctl_exec_args(repo_root, ["refill"], list(forwarded))


def build_workspace_runtime_bootstrap_args(repo_root: Path, forwarded: Sequence[str]) -> List[str]:
    return build_bigclawctl_exec_args(repo_root, ["workspace"], list(forwarded))


def _run(command: Sequence[str], cwd: Path) -> CommandResult:
    completed = subprocess.run(
        list(command),
        cwd=cwd,
        text=True,
        capture_output=True,
        check=False,
    )
    return CommandResult(
        stdout=completed.stdout.strip(),
        stderr=completed.stderr.strip(),
        returncode=completed.returncode,
    )


def _git(repo: Path, *args: str) -> CommandResult:
    return _run(["git", *args], repo)


def _require_git(repo: Path, *args: str) -> str:
    result = _git(repo, *args)
    if result.returncode != 0:
        detail = result.stderr or result.stdout or f"git {' '.join(args)} failed"
        raise WorkspaceBootstrapError(detail)
    return result.stdout


def _dirty(repo: Path) -> bool:
    return bool(_require_git(repo, "status", "--porcelain"))


def _remote_default_branch(repo: Path, remote: str) -> str:
    symbolic_ref = _git(repo, "symbolic-ref", "--quiet", f"refs/remotes/{remote}/HEAD")
    if symbolic_ref.returncode == 0 and symbolic_ref.stdout:
        prefix = f"refs/remotes/{remote}/"
        if symbolic_ref.stdout.startswith(prefix):
            return symbolic_ref.stdout[len(prefix) :]

    symref_result = _git(repo, "ls-remote", "--symref", remote, "HEAD")
    if symref_result.returncode != 0:
        detail = symref_result.stderr or symref_result.stdout or f"git ls-remote --symref failed for {remote}/HEAD"
        raise GitSyncError(detail)

    for line in symref_result.stdout.splitlines():
        if line.startswith("ref: ") and line.endswith("\tHEAD"):
            ref = line.split()[1]
            prefix = "refs/heads/"
            if ref.startswith(prefix):
                return ref[len(prefix) :]

    raise GitSyncError(f"Could not determine default branch for remote {remote}")


def _remote_branch_sha(repo: Path, remote: str, branch: str) -> str:
    local_ref = _git(repo, "rev-parse", f"refs/remotes/{remote}/{branch}")
    if local_ref.returncode == 0 and local_ref.stdout:
        return local_ref.stdout

    remote_result = _git(repo, "ls-remote", "--heads", remote, branch)
    if remote_result.returncode != 0:
        detail = remote_result.stderr or remote_result.stdout or f"git ls-remote failed for {remote}/{branch}"
        raise GitSyncError(detail)

    return remote_result.stdout.split()[0] if remote_result.stdout else ""


def _matches_remote_default_branch(repo: Path, remote: str, local_sha: str) -> bool:
    try:
        default_branch = _remote_default_branch(repo, remote)
        default_sha = _remote_branch_sha(repo, remote, default_branch)
    except GitSyncError:
        return False

    return bool(default_sha) and local_sha == default_sha


def sanitize_issue_identifier(identifier: str | None) -> str:
    raw = (identifier or "issue").strip() or "issue"
    return "".join(character if character.isalnum() or character in ".-_" else "_" for character in raw)


def bootstrap_branch_name(identifier: str | None) -> str:
    return f"{BOOTSTRAP_BRANCH_PREFIX}/{sanitize_issue_identifier(identifier)}"


def default_cache_base(path: str | Path | None = None) -> Path:
    if path is None:
        return DEFAULT_CACHE_BASE.expanduser().resolve()
    return Path(path).expanduser().resolve()


def normalize_repo_locator(repo_url: str) -> str:
    raw = repo_url.strip()

    if "://" in raw:
        parsed = urlparse(raw)
        locator = f"{parsed.netloc}{parsed.path}"
    elif ":" in raw and "@" in raw.split(":", 1)[0]:
        user_host, repo_path = raw.split(":", 1)
        host = user_host.split("@", 1)[-1]
        locator = f"{host}/{repo_path}"
    else:
        locator = raw

    return locator.strip().rstrip("/").removesuffix(".git")


def repo_cache_key(repo_url: str, cache_key: str | None = None) -> str:
    raw = (cache_key or normalize_repo_locator(repo_url)).strip().lower()
    sanitized = "".join(character if character.isalnum() or character in ".-_" else "-" for character in raw)
    compact = "-".join(segment for segment in sanitized.split("-") if segment)
    return compact or "repo"


def cache_root_for_repo(
    repo_url: str,
    cache_base: str | Path | None = None,
    cache_key: str | None = None,
) -> Path:
    return default_cache_base(cache_base) / repo_cache_key(repo_url, cache_key)


def resolve_cache_root(
    repo_url: str,
    cache_root: str | Path | None = None,
    cache_base: str | Path | None = None,
    cache_key: str | None = None,
) -> Path:
    if cache_root is not None:
        return Path(cache_root).expanduser().resolve()
    return cache_root_for_repo(repo_url, cache_base=cache_base, cache_key=cache_key)


def default_cache_root(path: str | Path | None = None) -> Path:
    return default_cache_base(path)


def build_validation_report(
    *,
    repo_url: str,
    workspace_root: str | Path,
    issue_identifiers: Sequence[str],
    default_branch: str = "main",
    cache_root: str | Path | None = None,
    cache_base: str | Path | None = None,
    cache_key: str | None = None,
    cleanup: bool = True,
) -> dict[str, Any]:
    workspace_root_path = Path(workspace_root).expanduser().resolve()
    workspace_root_path.mkdir(parents=True, exist_ok=True)

    bootstrap_results = []
    for issue_identifier in issue_identifiers:
        workspace_path = workspace_root_path / issue_identifier
        status = bootstrap_workspace(
            workspace=workspace_path,
            issue_identifier=issue_identifier,
            repo_url=repo_url,
            default_branch=default_branch,
            cache_root=cache_root,
            cache_base=cache_base,
            cache_key=cache_key,
        )
        bootstrap_results.append(status.to_dict())

    cache_roots = sorted({result["cache_root"] for result in bootstrap_results})
    mirror_paths = sorted({result["mirror_path"] for result in bootstrap_results})
    seed_paths = sorted({result["seed_path"] for result in bootstrap_results})
    cleanup_results = []

    if cleanup:
        for issue_identifier in issue_identifiers:
            workspace_path = workspace_root_path / issue_identifier
            status = cleanup_workspace(
                workspace=workspace_path,
                issue_identifier=issue_identifier,
                repo_url=repo_url,
                default_branch=default_branch,
                cache_root=cache_root,
                cache_base=cache_base,
                cache_key=cache_key,
            )
            cleanup_results.append(status.to_dict())

    return {
        "repo_url": repo_url,
        "default_branch": default_branch,
        "workspace_root": str(workspace_root_path),
        "issue_identifiers": list(issue_identifiers),
        "bootstrap_results": bootstrap_results,
        "cleanup_results": cleanup_results,
        "summary": {
            "workspace_count": len(bootstrap_results),
            "unique_cache_roots": cache_roots,
            "unique_mirror_paths": mirror_paths,
            "unique_seed_paths": seed_paths,
            "single_cache_root_reused": len(cache_roots) == 1,
            "single_mirror_reused": len(mirror_paths) == 1,
            "single_seed_reused": len(seed_paths) == 1,
            "mirror_creations": sum(1 for result in bootstrap_results if result["mirror_created"]),
            "seed_creations": sum(1 for result in bootstrap_results if result["seed_created"]),
            "clone_suppressed_after_first": all(result["clone_suppressed"] for result in bootstrap_results[1:]),
            "cache_reused_after_first": all(result["cache_reused"] for result in bootstrap_results[1:]),
            "all_workspaces_created_via_worktree": all(
                result["workspace_mode"] in {"worktree_created", "workspace_reused"}
                for result in bootstrap_results
            ),
            "cleanup_preserved_cache": bool(bootstrap_results)
            and Path(bootstrap_results[0]["mirror_path"]).exists()
            and Path(bootstrap_results[0]["seed_path"]).joinpath(".git").exists(),
        },
    }


def render_validation_markdown(report: dict[str, Any]) -> str:
    summary = report["summary"]
    lines = [
        "# Symphony bootstrap cache validation",
        "",
        f"- Repo: `{report['repo_url']}`",
        f"- Workspace root: `{report['workspace_root']}`",
        f"- Workspaces: `{summary['workspace_count']}`",
        f"- Single cache root reused: `{summary['single_cache_root_reused']}`",
        f"- Mirror creations: `{summary['mirror_creations']}`",
        f"- Seed creations: `{summary['seed_creations']}`",
        f"- Clone suppressed after first workspace: `{summary['clone_suppressed_after_first']}`",
        f"- Cleanup preserved cache: `{summary['cleanup_preserved_cache']}`",
        "",
        "## Bootstrap Results",
        "",
    ]

    for result in report["bootstrap_results"]:
        lines.extend(
            [
                f"- `{result['workspace']}`",
                f"  - `cache_root={result['cache_root']}`",
                f"  - `cache_key={result['cache_key']}`",
                f"  - `workspace_mode={result['workspace_mode']}`",
                f"  - `cache_reused={result['cache_reused']}`",
                f"  - `clone_suppressed={result['clone_suppressed']}`",
                f"  - `mirror_created={result['mirror_created']}`",
                f"  - `seed_created={result['seed_created']}`",
            ]
        )

    return "\n".join(lines) + "\n"


def write_validation_report(report: dict[str, Any], path: str | Path) -> Path:
    target = Path(path).expanduser().resolve()
    target.parent.mkdir(parents=True, exist_ok=True)

    if target.suffix.lower() == ".md":
        target.write_text(render_validation_markdown(report))
    else:
        target.write_text(json.dumps(report, ensure_ascii=False, indent=2))
    return target


def inspect_repo_sync(repo: Path | str, remote: str = "origin") -> RepoSyncStatus:
    repo_path = Path(repo).resolve()
    branch = _require_git(repo_path, "branch", "--show-current")
    if not branch:
        raise GitSyncError("Detached HEAD does not support issue branch sync automation")

    local_sha = _require_git(repo_path, "rev-parse", "HEAD")
    remote_result = _git(repo_path, "ls-remote", "--heads", remote, branch)
    if remote_result.returncode != 0:
        detail = remote_result.stderr or remote_result.stdout or f"git ls-remote failed for {remote}/{branch}"
        raise GitSyncError(detail)

    remote_sha = remote_result.stdout.split()[0] if remote_result.stdout else ""
    dirty = _dirty(repo_path)
    remote_exists = bool(remote_sha)
    synced = remote_exists and local_sha == remote_sha

    if not remote_exists and _matches_remote_default_branch(repo_path, remote, local_sha):
        synced = True

    return RepoSyncStatus(
        branch=branch,
        local_sha=local_sha,
        remote_sha=remote_sha,
        dirty=dirty,
        remote_exists=remote_exists,
        synced=synced,
    )


def install_git_hooks(repo: Path | str, hooks_path: str = ".githooks") -> Path:
    repo_path = Path(repo).resolve()
    hooks_dir = repo_path / hooks_path
    if not hooks_dir.is_dir():
        raise GitSyncError(f"Missing hooks directory: {hooks_dir}")

    _require_git(repo_path, "config", "core.hooksPath", hooks_path)
    for hook in hooks_dir.iterdir():
        if hook.is_file():
            hook.chmod(hook.stat().st_mode | EXECUTABLE_BITS)
    return hooks_dir


def ensure_repo_sync(
    repo: Path | str,
    remote: str = "origin",
    auto_push: bool = True,
    allow_dirty: bool = False,
) -> RepoSyncStatus:
    repo_path = Path(repo).resolve()
    status = inspect_repo_sync(repo_path, remote=remote)

    if status.dirty:
        if allow_dirty:
            return status
        raise GitSyncError("Working tree is dirty; commit or stash changes before syncing")

    if status.remote_exists and not status.synced:
        fetch_result = _git(repo_path, "fetch", remote, status.branch)
        if fetch_result.returncode != 0:
            detail = fetch_result.stderr or fetch_result.stdout or f"git fetch {remote} {status.branch} failed"
            raise GitSyncError(detail)

        ff_result = _git(repo_path, "pull", "--ff-only", remote, status.branch)
        if ff_result.returncode != 0:
            detail = ff_result.stderr or ff_result.stdout or f"git pull --ff-only {remote} {status.branch} failed"
            raise GitSyncError(detail)
        status = inspect_repo_sync(repo_path, remote=remote)

    if not auto_push or status.synced:
        return status

    push_args = ["push", remote, "HEAD"]
    if not status.remote_exists:
        push_args = ["push", "-u", remote, "HEAD"]

    push_result = _git(repo_path, *push_args)
    if push_result.returncode != 0:
        detail = push_result.stderr or push_result.stdout or f"git {' '.join(push_args)} failed"
        raise GitSyncError(detail)

    refreshed = inspect_repo_sync(repo_path, remote=remote)
    refreshed.pushed = True
    if not refreshed.synced:
        raise GitSyncError(
            f"Remote SHA mismatch after push: local={refreshed.local_sha} remote={refreshed.remote_sha or 'missing'}"
        )
    return refreshed


def main(
    argv: Sequence[str] | None = None,
    *,
    description: str = "Bootstrap Symphony workspaces from a shared local mirror.",
    default_repo_url: str = "",
    default_branch: str = "main",
    default_cache_root: str | None = None,
    default_cache_base: str = DEFAULT_CLI_CACHE_BASE,
    default_cache_key: str | None = None,
) -> int:
    parser = build_parser(
        description=description,
        default_repo_url=default_repo_url,
        default_branch=default_branch,
        default_cache_root=default_cache_root,
        default_cache_base=default_cache_base,
        default_cache_key=default_cache_key,
    )
    args = parser.parse_args(argv)
    workspace = Path(args.workspace).expanduser().resolve()

    try:
        payload = dict(
            workspace=workspace,
            issue_identifier=args.issue,
            repo_url=args.repo_url,
            default_branch=args.default_branch,
            cache_root=args.cache_root,
            cache_base=args.cache_base,
            cache_key=args.cache_key,
        )
        if args.command == "bootstrap":
            status = bootstrap_workspace(**payload)
        else:
            status = cleanup_workspace(**payload)
        emit({"status": "ok", **status.to_dict()}, args.json)
        return 0
    except WorkspaceBootstrapError as exc:
        emit({"status": "error", "workspace": str(workspace), "error": str(exc)}, args.json)
        return 1


def _remove_path(path: Path) -> None:
    if path.is_dir() and not path.is_symlink():
        shutil.rmtree(path)
    elif path.exists() or path.is_symlink():
        path.unlink()


def _cache_state(
    repo_url: str,
    repo_cache_root: Path,
    cache_key: str | None = None,
    *,
    mirror_created: bool = False,
    seed_created: bool = False,
) -> CacheBootstrapState:
    return CacheBootstrapState(
        cache_root=str(repo_cache_root),
        cache_key=repo_cache_key(repo_url, cache_key),
        mirror_path=str(repo_cache_root / "mirror.git"),
        seed_path=str(repo_cache_root / "seed"),
        mirror_created=mirror_created,
        seed_created=seed_created,
    )


def ensure_mirror(
    repo_url: str,
    cache_root: str | Path | None = None,
    cache_base: str | Path | None = None,
    cache_key: str | None = None,
) -> CacheBootstrapState:
    repo_cache_root = resolve_cache_root(repo_url, cache_root=cache_root, cache_base=cache_base, cache_key=cache_key)
    mirror_path = repo_cache_root / "mirror.git"
    mirror_path.parent.mkdir(parents=True, exist_ok=True)
    mirror_created = False

    if not (mirror_path / "HEAD").exists():
        if mirror_path.exists():
            _remove_path(mirror_path)
        result = subprocess.run(
            ["git", "clone", "--mirror", repo_url, str(mirror_path)],
            text=True,
            capture_output=True,
            check=False,
        )
        if result.returncode != 0:
            detail = result.stderr.strip() or result.stdout.strip() or "git clone --mirror failed"
            raise WorkspaceBootstrapError(detail)
        mirror_created = True
    else:
        _require_git(mirror_path, "remote", "set-url", "origin", repo_url)
        _require_git(mirror_path, "fetch", "--prune", "origin")

    return _cache_state(repo_url, repo_cache_root, cache_key, mirror_created=mirror_created)


def ensure_seed(
    repo_url: str,
    default_branch: str,
    cache_root: str | Path | None = None,
    cache_base: str | Path | None = None,
    cache_key: str | None = None,
) -> CacheBootstrapState:
    cache_state = ensure_mirror(
        repo_url,
        cache_root=cache_root,
        cache_base=cache_base,
        cache_key=cache_key,
    )
    seed_path = Path(cache_state.seed_path)
    seed_created = False

    if not (seed_path / ".git").exists():
        if seed_path.exists():
            _remove_path(seed_path)
        result = subprocess.run(
            ["git", "clone", cache_state.mirror_path, str(seed_path)],
            text=True,
            capture_output=True,
            check=False,
        )
        if result.returncode != 0:
            detail = result.stderr.strip() or result.stdout.strip() or "git clone seed failed"
            raise WorkspaceBootstrapError(detail)
        seed_created = True

    configure_seed_remotes(seed_path, repo_url, Path(cache_state.mirror_path))
    _require_git(seed_path, "fetch", "--prune", CACHE_REMOTE)
    _require_git(seed_path, "worktree", "prune")
    _require_git(seed_path, "checkout", "-B", default_branch, f"{CACHE_REMOTE}/{default_branch}")
    return _cache_state(
        repo_url,
        Path(cache_state.cache_root),
        cache_key,
        mirror_created=cache_state.mirror_created,
        seed_created=seed_created,
    )


def configure_seed_remotes(seed_path: Path, repo_url: str, mirror_path: Path) -> None:
    remotes = set(_require_git(seed_path, "remote").splitlines())

    if CACHE_REMOTE not in remotes and "origin" in remotes:
        current_origin = _require_git(seed_path, "remote", "get-url", "origin")
        if Path(current_origin).expanduser().resolve() == mirror_path.resolve():
            _require_git(seed_path, "remote", "rename", "origin", CACHE_REMOTE)
            remotes = set(_require_git(seed_path, "remote").splitlines())

    if CACHE_REMOTE not in remotes:
        _require_git(seed_path, "remote", "add", CACHE_REMOTE, str(mirror_path))
    else:
        _require_git(seed_path, "remote", "set-url", CACHE_REMOTE, str(mirror_path))

    remotes = set(_require_git(seed_path, "remote").splitlines())
    if "origin" not in remotes:
        _require_git(seed_path, "remote", "add", "origin", repo_url)
    else:
        _require_git(seed_path, "remote", "set-url", "origin", repo_url)

    _require_git(seed_path, "config", "remote.pushDefault", "origin")


def bootstrap_workspace(
    workspace: str | Path,
    issue_identifier: str | None,
    repo_url: str,
    default_branch: str = "main",
    cache_root: str | Path | None = None,
    cache_base: str | Path | None = None,
    cache_key: str | None = None,
) -> WorkspaceBootstrapStatus:
    workspace_path = Path(workspace).expanduser().resolve()
    cache_state = ensure_seed(
        repo_url,
        default_branch,
        cache_root=cache_root,
        cache_base=cache_base,
        cache_key=cache_key,
    )
    seed_path = Path(cache_state.seed_path)
    branch = bootstrap_branch_name(issue_identifier or workspace_path.name)
    cache_reused = not cache_state.mirror_created and not cache_state.seed_created
    clone_suppressed = not cache_state.mirror_created

    git_dir = workspace_path / ".git"
    if git_dir.exists():
        current_branch = _require_git(workspace_path, "branch", "--show-current")
        return WorkspaceBootstrapStatus(
            workspace=str(workspace_path),
            branch=current_branch or branch,
            cache_root=cache_state.cache_root,
            cache_key=cache_state.cache_key,
            mirror_path=cache_state.mirror_path,
            seed_path=cache_state.seed_path,
            reused=True,
            cache_reused=cache_reused,
            clone_suppressed=clone_suppressed,
            mirror_created=cache_state.mirror_created,
            seed_created=cache_state.seed_created,
            workspace_mode="workspace_reused",
        )

    parent = workspace_path.parent
    parent.mkdir(parents=True, exist_ok=True)

    if workspace_path.exists() and workspace_path.is_dir() and any(workspace_path.iterdir()):
        raise WorkspaceBootstrapError(f"Workspace is not empty: {workspace_path}")

    _require_git(
        seed_path,
        "worktree",
        "add",
        "--force",
        "-B",
        branch,
        str(workspace_path),
        f"{CACHE_REMOTE}/{default_branch}",
    )

    return WorkspaceBootstrapStatus(
        workspace=str(workspace_path),
        branch=branch,
        cache_root=cache_state.cache_root,
        cache_key=cache_state.cache_key,
        mirror_path=cache_state.mirror_path,
        seed_path=cache_state.seed_path,
        reused=False,
        cache_reused=cache_reused,
        clone_suppressed=clone_suppressed,
        mirror_created=cache_state.mirror_created,
        seed_created=cache_state.seed_created,
        workspace_mode="worktree_created",
    )


def cleanup_workspace(
    workspace: str | Path,
    issue_identifier: str | None,
    repo_url: str,
    default_branch: str = "main",
    cache_root: str | Path | None = None,
    cache_base: str | Path | None = None,
    cache_key: str | None = None,
) -> WorkspaceBootstrapStatus:
    workspace_path = Path(workspace).expanduser().resolve()
    repo_cache_root = resolve_cache_root(repo_url, cache_root=cache_root, cache_base=cache_base, cache_key=cache_key)
    cache_state = _cache_state(repo_url, repo_cache_root, cache_key)
    seed_path = Path(cache_state.seed_path)
    mirror_path = Path(cache_state.mirror_path)
    branch = bootstrap_branch_name(issue_identifier or workspace_path.name)

    if not (seed_path / ".git").exists() or not workspace_path.exists():
        return WorkspaceBootstrapStatus(
            workspace=str(workspace_path),
            branch=branch,
            cache_root=cache_state.cache_root,
            cache_key=cache_state.cache_key,
            mirror_path=cache_state.mirror_path,
            seed_path=cache_state.seed_path,
            reused=False,
            cache_reused=seed_path.exists() or mirror_path.exists(),
            clone_suppressed=True,
            workspace_mode="cleanup",
            removed=False,
        )

    configure_seed_remotes(seed_path, repo_url, mirror_path)

    if _git(workspace_path, "rev-parse", "--git-dir").returncode == 0:
        current_branch = _require_git(workspace_path, "branch", "--show-current")
        branch = current_branch or branch

    worktree_list = _require_git(seed_path, "worktree", "list", "--porcelain")
    registered = f"worktree {workspace_path}" in worktree_list
    if registered:
        _require_git(seed_path, "worktree", "remove", "--force", str(workspace_path))
        _require_git(seed_path, "worktree", "prune")

    local_branches = set(_require_git(seed_path, "branch", "--format", "%(refname:short)").splitlines())
    if branch.startswith(f"{BOOTSTRAP_BRANCH_PREFIX}/") and branch in local_branches:
        _require_git(seed_path, "branch", "-D", branch)

    _require_git(seed_path, "checkout", "-B", default_branch, f"{CACHE_REMOTE}/{default_branch}")
    return WorkspaceBootstrapStatus(
        workspace=str(workspace_path),
        branch=branch,
        cache_root=cache_state.cache_root,
        cache_key=cache_state.cache_key,
        mirror_path=cache_state.mirror_path,
        seed_path=cache_state.seed_path,
        reused=False,
        cache_reused=True,
        clone_suppressed=True,
        workspace_mode="cleanup",
        removed=registered,
    )


def status_as_json(status: WorkspaceBootstrapStatus) -> str:
    return json.dumps(status.to_dict(), ensure_ascii=False, indent=2)


if __name__ == "__main__":
    package_main()
