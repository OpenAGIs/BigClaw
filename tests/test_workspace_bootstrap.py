import json
import subprocess
from pathlib import Path

from bigclaw.workspace_bootstrap import (
    bootstrap_workspace,
    cache_root_for_repo,
    cleanup_workspace,
    prewarm_workspaces,
    prepare_shared_snapshot,
    repo_cache_key,
)
from bigclaw.workspace_bootstrap_cli import main as workspace_bootstrap_cli_main
from bigclaw.workspace_bootstrap_validation import build_validation_report


def git(repo: Path, *args: str) -> str:
    completed = subprocess.run(
        ["git", *args],
        cwd=repo,
        text=True,
        capture_output=True,
        check=True,
    )
    return completed.stdout.strip()


def init_repo(repo: Path, branch: str = "main") -> None:
    git(repo, "init", "-b", branch)
    git(repo, "config", "user.email", "test@example.com")
    git(repo, "config", "user.name", "Test User")


def commit_file(repo: Path, name: str, content: str, message: str) -> str:
    (repo / name).write_text(content)
    git(repo, "add", name)
    git(repo, "commit", "-m", message)
    return git(repo, "rev-parse", "HEAD")


def init_remote_with_main(tmp_path: Path) -> Path:
    remote = tmp_path / "remote.git"
    subprocess.run(
        ["git", "init", "--bare", "--initial-branch=main", str(remote)],
        check=True,
        capture_output=True,
        text=True,
    )

    source = tmp_path / "source"
    source.mkdir()
    init_repo(source)
    git(source, "remote", "add", "origin", str(remote))
    commit_file(source, "README.md", "hello\n", "initial")
    git(source, "push", "-u", "origin", "main")
    return remote


def test_repo_cache_key_derives_from_repo_locator() -> None:
    assert repo_cache_key("git@github.com:OpenAGIs/BigClaw.git") == "github.com-openagis-bigclaw"
    assert repo_cache_key("https://github.com/OpenAGIs/BigClaw.git") == "github.com-openagis-bigclaw"
    assert repo_cache_key("git@github.com:OpenAGIs/BigClaw.git", cache_key="Team/BigClaw") == "team-bigclaw"


def test_cache_root_for_repo_uses_repo_specific_directory(tmp_path: Path) -> None:
    cache_root = cache_root_for_repo(
        "git@github.com:OpenAGIs/BigClaw.git",
        cache_base=tmp_path / "repos",
    )

    assert cache_root == tmp_path / "repos" / "github.com-openagis-bigclaw"


def test_bootstrap_workspace_creates_shared_worktree_from_local_seed(tmp_path: Path) -> None:
    remote = init_remote_with_main(tmp_path)
    cache_base = tmp_path / "repos"
    workspace = tmp_path / "workspaces" / "OPE-321"

    status = bootstrap_workspace(workspace, "OPE-321", str(remote), cache_base=cache_base)

    expected_cache_root = cache_root_for_repo(str(remote), cache_base=cache_base)
    assert status.reused is False
    assert status.branch == "symphony/OPE-321"
    assert status.workspace_mode == "worktree_created"
    assert status.cache_reused is False
    assert status.clone_suppressed is False
    assert status.mirror_created is True
    assert status.seed_created is True
    assert Path(status.cache_root) == expected_cache_root
    assert (expected_cache_root / "mirror.git" / "HEAD").exists()
    assert (expected_cache_root / "seed" / ".git").exists()
    assert workspace.exists()
    assert (workspace / ".git").exists()
    assert git(workspace, "branch", "--show-current") == "symphony/OPE-321"
    assert (workspace / "README.md").read_text() == "hello\n"
    assert Path(git(workspace, "rev-parse", "--git-common-dir")).resolve() == (expected_cache_root / "seed" / ".git").resolve()
    assert git(expected_cache_root / "seed", "remote", "get-url", "origin") == str(remote)
    assert git(expected_cache_root / "seed", "remote", "get-url", "cache") == str((expected_cache_root / "mirror.git").resolve())


def test_second_workspace_reuses_warm_cache_without_full_clone(tmp_path: Path) -> None:
    remote = init_remote_with_main(tmp_path)
    cache_base = tmp_path / "repos"

    first = bootstrap_workspace(tmp_path / "workspaces" / "OPE-322", "OPE-322", str(remote), cache_base=cache_base)
    second = bootstrap_workspace(tmp_path / "workspaces" / "OPE-323", "OPE-323", str(remote), cache_base=cache_base)

    assert first.cache_root == second.cache_root
    assert second.cache_reused is True
    assert second.clone_suppressed is True
    assert second.mirror_created is False
    assert second.seed_created is False
    assert second.workspace_mode == "worktree_created"


def test_prepare_shared_snapshot_serializes_reusable_cache_configuration(tmp_path: Path) -> None:
    remote = init_remote_with_main(tmp_path)
    snapshot = prepare_shared_snapshot(str(remote), cache_base=tmp_path / "repos")
    restored = type(snapshot).from_dict(snapshot.to_dict())

    assert restored.repo_url == str(remote)
    assert restored.repo_locator.endswith("remote")
    assert restored.default_branch == "main"
    assert Path(restored.cache_root) == cache_root_for_repo(str(remote), cache_base=tmp_path / "repos")
    assert Path(restored.mirror_path).joinpath("HEAD").exists()
    assert Path(restored.seed_path).joinpath(".git").exists()


def test_bootstrap_workspace_reuses_existing_issue_worktree(tmp_path: Path) -> None:
    remote = init_remote_with_main(tmp_path)
    cache_base = tmp_path / "repos"
    workspace = tmp_path / "workspaces" / "OPE-324"

    first = bootstrap_workspace(workspace, "OPE-324", str(remote), cache_base=cache_base)
    second = bootstrap_workspace(workspace, "OPE-324", str(remote), cache_base=cache_base)

    assert first.reused is False
    assert second.reused is True
    assert second.workspace_mode == "workspace_reused"
    assert second.cache_reused is True
    assert second.clone_suppressed is True
    assert second.branch == "symphony/OPE-324"


def test_bootstrap_workspace_can_reuse_prepared_snapshot(tmp_path: Path) -> None:
    remote = init_remote_with_main(tmp_path)
    snapshot = prepare_shared_snapshot(str(remote), cache_base=tmp_path / "repos")

    first = bootstrap_workspace(
        tmp_path / "workspaces" / "OPE-324A",
        "OPE-324A",
        str(remote),
        shared_snapshot=snapshot.to_dict(),
    )
    second = bootstrap_workspace(
        tmp_path / "workspaces" / "OPE-324B",
        "OPE-324B",
        str(remote),
        shared_snapshot=snapshot,
    )

    assert first.snapshot_reused is True
    assert second.snapshot_reused is True
    assert first.mirror_created is False
    assert first.seed_created is False
    assert first.cache_reused is True
    assert first.clone_suppressed is True
    assert second.cache_reused is True
    assert second.clone_suppressed is True


def test_cleanup_workspace_preserves_shared_cache_for_future_reuse(tmp_path: Path) -> None:
    remote = init_remote_with_main(tmp_path)
    cache_base = tmp_path / "repos"
    cache_root = cache_root_for_repo(str(remote), cache_base=cache_base)
    workspace = tmp_path / "workspaces" / "OPE-325"

    bootstrap_workspace(workspace, "OPE-325", str(remote), cache_base=cache_base)
    status = cleanup_workspace(workspace, "OPE-325", str(remote), cache_base=cache_base)
    follow_up = bootstrap_workspace(tmp_path / "workspaces" / "OPE-326", "OPE-326", str(remote), cache_base=cache_base)

    assert status.removed is True
    assert not workspace.exists()
    assert (cache_root / "mirror.git" / "HEAD").exists()
    assert (cache_root / "seed" / ".git").exists()
    assert follow_up.cache_reused is True
    assert follow_up.clone_suppressed is True
    assert follow_up.mirror_created is False
    assert follow_up.seed_created is False


def test_bootstrap_recovers_from_stale_seed_directory_without_remote_reclone(tmp_path: Path) -> None:
    remote = init_remote_with_main(tmp_path)
    cache_base = tmp_path / "repos"
    first = bootstrap_workspace(tmp_path / "workspaces" / "OPE-327", "OPE-327", str(remote), cache_base=cache_base)
    cache_root = Path(first.cache_root)

    cleanup_workspace(tmp_path / "workspaces" / "OPE-327", "OPE-327", str(remote), cache_base=cache_base)
    seed_path = cache_root / "seed"
    if seed_path.exists():
        subprocess.run(["rm", "-rf", str(seed_path)], check=True)
    seed_path.mkdir(parents=True, exist_ok=True)
    (seed_path / "stale.txt").write_text("stale\n")

    recovered = bootstrap_workspace(tmp_path / "workspaces" / "OPE-328", "OPE-328", str(remote), cache_base=cache_base)

    assert recovered.cache_reused is False
    assert recovered.clone_suppressed is True
    assert recovered.mirror_created is False
    assert recovered.seed_created is True
    assert (cache_root / "mirror.git" / "HEAD").exists()
    assert (cache_root / "seed" / ".git").exists()


def test_prewarm_workspaces_prepares_multiple_worktrees_from_one_snapshot(tmp_path: Path) -> None:
    remote = init_remote_with_main(tmp_path)
    workspaces = [
        (tmp_path / "workspaces" / "OPE-401", "OPE-401"),
        (tmp_path / "workspaces" / "OPE-402", "OPE-402"),
        (tmp_path / "workspaces" / "OPE-403", "OPE-403"),
    ]

    prewarm = prewarm_workspaces(workspaces, str(remote), cache_base=tmp_path / "repos")

    assert prewarm.snapshot.mirror_created is True
    assert prewarm.snapshot.seed_created is True
    assert prewarm.to_dict()["summary"]["snapshot_reused_for_all"] is True
    assert len(prewarm.workspaces) == 3
    assert all(status.snapshot_reused for status in prewarm.workspaces)
    assert all(status.workspace_mode == "worktree_created" for status in prewarm.workspaces)
    assert all(status.cache_root == prewarm.workspaces[0].cache_root for status in prewarm.workspaces)
    assert prewarm.workspaces[1].cache_reused is True
    assert prewarm.workspaces[2].clone_suppressed is True


def test_cleanup_workspace_prunes_worktree_and_bootstrap_branch(tmp_path: Path) -> None:
    remote = init_remote_with_main(tmp_path)
    cache_base = tmp_path / "repos"
    workspace = tmp_path / "workspaces" / "OPE-329"
    cache_root = cache_root_for_repo(str(remote), cache_base=cache_base)

    bootstrap_workspace(workspace, "OPE-329", str(remote), cache_base=cache_base)
    status = cleanup_workspace(workspace, "OPE-329", str(remote), cache_base=cache_base)

    assert status.removed is True
    assert not workspace.exists()
    assert "symphony/OPE-329" not in git(cache_root / "seed", "branch", "--format", "%(refname:short)").splitlines()
    assert str(workspace.resolve()) not in git(cache_root / "seed", "worktree", "list", "--porcelain")


def test_validation_report_covers_three_workspaces_with_one_cache(tmp_path: Path) -> None:
    remote = init_remote_with_main(tmp_path)
    report = build_validation_report(
        repo_url=str(remote),
        workspace_root=tmp_path / "validation-workspaces",
        issue_identifiers=["OPE-272", "OPE-273", "OPE-274"],
        cache_base=tmp_path / "repos",
        cleanup=True,
    )

    assert report["summary"]["workspace_count"] == 3
    assert report["summary"]["single_cache_root_reused"] is True
    assert report["summary"]["single_mirror_reused"] is True
    assert report["summary"]["single_seed_reused"] is True
    assert report["summary"]["mirror_creations"] == 1
    assert report["summary"]["seed_creations"] == 1
    assert report["summary"]["shared_snapshot_reused_for_all"] is True
    assert report["summary"]["clone_suppressed_after_first"] is True
    assert report["summary"]["cache_reused_after_first"] is True
    assert report["summary"]["cleanup_preserved_cache"] is True
    assert report["shared_snapshot"]["default_branch"] == "main"


def test_cli_prepare_snapshot_writes_snapshot_file(tmp_path: Path, capsys) -> None:
    remote = init_remote_with_main(tmp_path)
    snapshot_path = tmp_path / "snapshot.json"

    exit_code = workspace_bootstrap_cli_main(
        [
            "prepare-snapshot",
            "--repo-url",
            str(remote),
            "--cache-base",
            str(tmp_path / "repos"),
            "--snapshot-file",
            str(snapshot_path),
            "--json",
        ]
    )

    captured = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert captured["status"] == "ok"
    assert captured["snapshot_file"] == str(snapshot_path.resolve())
    assert captured["snapshot"]["repo_url"] == str(remote)
    assert json.loads(snapshot_path.read_text())["repo_url"] == str(remote)


def test_cli_bootstrap_uses_snapshot_file(tmp_path: Path, capsys) -> None:
    remote = init_remote_with_main(tmp_path)
    snapshot_path = tmp_path / "snapshot.json"
    snapshot = prepare_shared_snapshot(str(remote), cache_base=tmp_path / "repos")
    snapshot_path.write_text(json.dumps(snapshot.to_dict(), ensure_ascii=False, indent=2))

    workspace = tmp_path / "workspaces" / "BIG-197-A"
    exit_code = workspace_bootstrap_cli_main(
        [
            "bootstrap",
            "--workspace",
            str(workspace),
            "--issue",
            "BIG-197-A",
            "--repo-url",
            str(remote),
            "--snapshot-file",
            str(snapshot_path),
            "--json",
        ]
    )

    captured = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert captured["status"] == "ok"
    assert captured["snapshot_reused"] is True
    assert captured["cache_reused"] is True
    assert workspace.exists()


def test_cli_prewarm_creates_multiple_workspaces_from_one_snapshot(tmp_path: Path, capsys) -> None:
    remote = init_remote_with_main(tmp_path)
    snapshot_path = tmp_path / "snapshot.json"

    exit_code = workspace_bootstrap_cli_main(
        [
            "prewarm",
            "--workspace-root",
            str(tmp_path / "workspaces"),
            "--issues",
            "BIG-197-1",
            "BIG-197-2",
            "BIG-197-3",
            "--repo-url",
            str(remote),
            "--cache-base",
            str(tmp_path / "repos"),
            "--snapshot-file",
            str(snapshot_path),
            "--json",
        ]
    )

    captured = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert captured["status"] == "ok"
    assert captured["summary"]["workspace_count"] == 3
    assert captured["summary"]["snapshot_reused_for_all"] is True
    assert Path(captured["snapshot"]["mirror_path"]).joinpath("HEAD").exists()
    assert all(Path(workspace["workspace"]).exists() for workspace in captured["workspaces"])
    assert json.loads(snapshot_path.read_text())["cache_root"] == captured["snapshot"]["cache_root"]
