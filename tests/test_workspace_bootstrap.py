import subprocess
from pathlib import Path

from bigclaw.workspace_bootstrap import (
    bootstrap_workspace,
    cache_root_for_repo,
    cleanup_workspace,
    repo_cache_key,
)


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
    assert (expected_cache_root / "mirror.git" / "HEAD").exists()
    assert (expected_cache_root / "seed" / ".git").exists()
    assert workspace.exists()
    assert (workspace / ".git").exists()
    assert git(workspace, "branch", "--show-current") == "symphony/OPE-321"
    assert (workspace / "README.md").read_text() == "hello\n"
    assert Path(git(workspace, "rev-parse", "--git-common-dir")).resolve() == (expected_cache_root / "seed" / ".git").resolve()
    assert git(expected_cache_root / "seed", "remote", "get-url", "origin") == str(remote)
    assert git(expected_cache_root / "seed", "remote", "get-url", "cache") == str((expected_cache_root / "mirror.git").resolve())


def test_bootstrap_workspace_reuses_existing_issue_worktree(tmp_path: Path) -> None:
    remote = init_remote_with_main(tmp_path)
    cache_base = tmp_path / "repos"
    workspace = tmp_path / "workspaces" / "OPE-322"

    first = bootstrap_workspace(workspace, "OPE-322", str(remote), cache_base=cache_base)
    second = bootstrap_workspace(workspace, "OPE-322", str(remote), cache_base=cache_base)

    assert first.reused is False
    assert second.reused is True
    assert second.branch == "symphony/OPE-322"


def test_cleanup_workspace_prunes_worktree_and_bootstrap_branch(tmp_path: Path) -> None:
    remote = init_remote_with_main(tmp_path)
    cache_base = tmp_path / "repos"
    workspace = tmp_path / "workspaces" / "OPE-323"
    cache_root = cache_root_for_repo(str(remote), cache_base=cache_base)

    bootstrap_workspace(workspace, "OPE-323", str(remote), cache_base=cache_base)
    status = cleanup_workspace(workspace, "OPE-323", str(remote), cache_base=cache_base)

    assert status.removed is True
    assert not workspace.exists()
    assert "symphony/OPE-323" not in git(cache_root / "seed", "branch", "--format", "%(refname:short)").splitlines()
    assert str(workspace.resolve()) not in git(cache_root / "seed", "worktree", "list", "--porcelain")
