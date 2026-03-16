import subprocess
from pathlib import Path

from bigclaw.workspace_bootstrap import bootstrap_workspace, cleanup_workspace


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


def test_bootstrap_workspace_creates_shared_worktree_from_local_seed(tmp_path: Path) -> None:
    remote = init_remote_with_main(tmp_path)
    cache_root = tmp_path / "cache"
    workspace = tmp_path / "workspaces" / "OPE-321"

    status = bootstrap_workspace(workspace, "OPE-321", str(remote), cache_root=cache_root)

    assert status.reused is False
    assert status.branch == "symphony/OPE-321"
    assert (cache_root / "mirror.git" / "HEAD").exists()
    assert (cache_root / "seed" / ".git").exists()
    assert workspace.exists()
    assert (workspace / ".git").exists()
    assert git(workspace, "branch", "--show-current") == "symphony/OPE-321"
    assert (workspace / "README.md").read_text() == "hello\n"
    assert Path(git(workspace, "rev-parse", "--git-common-dir")).resolve() == (cache_root / "seed" / ".git").resolve()
    assert git(cache_root / "seed", "remote", "get-url", "origin") == str(remote)
    assert git(cache_root / "seed", "remote", "get-url", "cache") == str((cache_root / "mirror.git").resolve())


def test_bootstrap_workspace_reuses_existing_issue_worktree(tmp_path: Path) -> None:
    remote = init_remote_with_main(tmp_path)
    cache_root = tmp_path / "cache"
    workspace = tmp_path / "workspaces" / "OPE-322"

    first = bootstrap_workspace(workspace, "OPE-322", str(remote), cache_root=cache_root)
    second = bootstrap_workspace(workspace, "OPE-322", str(remote), cache_root=cache_root)

    assert first.reused is False
    assert second.reused is True
    assert second.branch == "symphony/OPE-322"


def test_cleanup_workspace_prunes_worktree_and_bootstrap_branch(tmp_path: Path) -> None:
    remote = init_remote_with_main(tmp_path)
    cache_root = tmp_path / "cache"
    workspace = tmp_path / "workspaces" / "OPE-323"

    bootstrap_workspace(workspace, "OPE-323", str(remote), cache_root=cache_root)
    status = cleanup_workspace(workspace, "OPE-323", str(remote), cache_root=cache_root)

    assert status.removed is True
    assert not workspace.exists()
    assert "symphony/OPE-323" not in git(cache_root / "seed", "branch", "--format", "%(refname:short)").splitlines()
    assert str(workspace.resolve()) not in git(cache_root / "seed", "worktree", "list", "--porcelain")
