import importlib.util
import subprocess
import sys
from pathlib import Path

MODULE_PATH = Path(__file__).resolve().parents[1] / "src" / "bigclaw" / "workspace_bootstrap.py"
SPEC = importlib.util.spec_from_file_location("bigclaw_workspace_bootstrap_test_module", MODULE_PATH)
assert SPEC is not None and SPEC.loader is not None
WORKSPACE_BOOTSTRAP = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = WORKSPACE_BOOTSTRAP
SPEC.loader.exec_module(WORKSPACE_BOOTSTRAP)

bootstrap_workspace = WORKSPACE_BOOTSTRAP.bootstrap_workspace
build_validation_report = WORKSPACE_BOOTSTRAP.build_validation_report
cache_root_for_repo = WORKSPACE_BOOTSTRAP.cache_root_for_repo
cleanup_workspace = WORKSPACE_BOOTSTRAP.cleanup_workspace
repo_cache_key = WORKSPACE_BOOTSTRAP.repo_cache_key


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
    assert report["summary"]["clone_suppressed_after_first"] is True
    assert report["summary"]["cache_reused_after_first"] is True
    assert report["summary"]["cleanup_preserved_cache"] is True
