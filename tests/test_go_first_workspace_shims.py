import importlib.util
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


def load_script_module(name: str, relative_path: str):
    path = ROOT / relative_path
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(module)
    return module


def test_symphony_workspace_bootstrap_shim_forwards_to_go_wrapper(monkeypatch) -> None:
    module = load_script_module("symphony_workspace_bootstrap", "scripts/ops/symphony_workspace_bootstrap.py")
    captured = {}

    class Completed:
        returncode = 0

    def fake_run(command, cwd):
        captured["command"] = command
        captured["cwd"] = cwd
        return Completed()

    monkeypatch.setattr(module.subprocess, "run", fake_run)

    exit_code = module.main(["bootstrap", "--workspace", "/tmp/workspace", "--issue", "BIG-GOM-307"])

    assert exit_code == 0
    assert captured["command"] == [
        "bash",
        str(ROOT / "scripts/ops/bigclawctl"),
        "workspace",
        "bootstrap",
        "--workspace",
        "/tmp/workspace",
        "--issue",
        "BIG-GOM-307",
    ]
    assert captured["cwd"] == ROOT


def test_bigclaw_workspace_bootstrap_shim_adds_repo_defaults(monkeypatch) -> None:
    module = load_script_module("bigclaw_workspace_bootstrap", "scripts/ops/bigclaw_workspace_bootstrap.py")
    captured = {}

    class Completed:
        returncode = 0

    def fake_run(command, cwd):
        captured["command"] = command
        captured["cwd"] = cwd
        return Completed()

    monkeypatch.setattr(module.subprocess, "run", fake_run)

    exit_code = module.main(["bootstrap", "--workspace", "/tmp/workspace", "--issue", "BIG-GOM-307"])

    assert exit_code == 0
    assert captured["command"] == [
        "bash",
        str(ROOT / "scripts/ops/bigclawctl"),
        "workspace",
        "bootstrap",
        "--workspace",
        "/tmp/workspace",
        "--issue",
        "BIG-GOM-307",
        "--repo-url",
        "git@github.com:OpenAGIs/BigClaw.git",
        "--cache-key",
        "openagis-bigclaw",
    ]
    assert captured["cwd"] == ROOT


def test_bigclaw_workspace_bootstrap_shim_respects_explicit_overrides(monkeypatch) -> None:
    module = load_script_module("bigclaw_workspace_bootstrap_overrides", "scripts/ops/bigclaw_workspace_bootstrap.py")
    captured = {}

    class Completed:
        returncode = 0

    def fake_run(command, cwd):
        captured["command"] = command
        captured["cwd"] = cwd
        return Completed()

    monkeypatch.setattr(module.subprocess, "run", fake_run)

    exit_code = module.main(
        [
            "cleanup",
            "--workspace",
            "/tmp/workspace",
            "--repo-url=https://example.com/repo.git",
            "--cache-key=custom-cache",
        ]
    )

    assert exit_code == 0
    assert captured["command"] == [
        "bash",
        str(ROOT / "scripts/ops/bigclawctl"),
        "workspace",
        "cleanup",
        "--workspace",
        "/tmp/workspace",
        "--repo-url=https://example.com/repo.git",
        "--cache-key=custom-cache",
    ]
    assert captured["cwd"] == ROOT


def test_symphony_workspace_validate_shim_forwards_to_go_wrapper(monkeypatch) -> None:
    module = load_script_module("symphony_workspace_validate", "scripts/ops/symphony_workspace_validate.py")
    captured = {}

    class Completed:
        returncode = 0

    def fake_run(command, cwd):
        captured["command"] = command
        captured["cwd"] = cwd
        return Completed()

    monkeypatch.setattr(module.subprocess, "run", fake_run)

    exit_code = module.main(
        [
            "--workspace-root",
            "/tmp/workspaces",
            "--repo-url",
            "git@example.com:repo.git",
            "--issues",
            "BIG-1",
            "BIG-2",
            "--json",
        ]
    )

    assert exit_code == 0
    assert captured["command"] == [
        "bash",
        str(ROOT / "scripts/ops/bigclawctl"),
        "workspace",
        "validate",
        "--workspace-root",
        "/tmp/workspaces",
        "--repo-url",
        "git@example.com:repo.git",
        "--issues",
        "BIG-1",
        "BIG-2",
        "--json",
    ]
    assert captured["cwd"] == ROOT


def test_bigclaw_github_sync_shim_forwards_to_go_wrapper(monkeypatch) -> None:
    module = load_script_module("bigclaw_github_sync", "scripts/ops/bigclaw_github_sync.py")
    captured = {}

    class Completed:
        returncode = 0

    def fake_run(command, cwd):
        captured["command"] = command
        captured["cwd"] = cwd
        return Completed()

    monkeypatch.setattr(module.subprocess, "run", fake_run)

    exit_code = module.main(["status", "--require-synced", "--json"])

    assert exit_code == 0
    assert captured["command"] == [
        "bash",
        str(ROOT / "scripts/ops/bigclawctl"),
        "github-sync",
        "status",
        "--require-synced",
        "--json",
    ]
    assert captured["cwd"] == ROOT


def test_bigclaw_refill_queue_shim_forwards_to_go_wrapper(monkeypatch) -> None:
    module = load_script_module("bigclaw_refill_queue", "scripts/ops/bigclaw_refill_queue.py")
    captured = {}

    class Completed:
        returncode = 0

    def fake_run(command, cwd):
        captured["command"] = command
        captured["cwd"] = cwd
        return Completed()

    monkeypatch.setattr(module.subprocess, "run", fake_run)

    exit_code = module.main(["--apply", "--watch", "--local-issues", "local-issues.json"])

    assert exit_code == 0
    assert captured["command"] == [
        "bash",
        str(ROOT / "scripts/ops/bigclawctl"),
        "refill",
        "--apply",
        "--watch",
        "--local-issues",
        "local-issues.json",
    ]
    assert captured["cwd"] == ROOT
