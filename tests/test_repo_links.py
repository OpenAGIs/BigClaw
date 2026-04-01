from pathlib import Path

from bigclaw.models import Task
from bigclaw.observability import RunCommitLink, TaskRun, bind_run_commits


def test_run_closeout_supports_commit_roles_and_accepted_hash():
    task = Task(task_id="OPE-143", source="linear", title="run links", description="")
    run = TaskRun.from_task(task, run_id="run-143", medium="docker")

    links = [
        RunCommitLink(run_id=run.run_id, commit_hash="aaa111", role="source", repo_space_id="space-1"),
        RunCommitLink(run_id=run.run_id, commit_hash="bbb222", role="candidate", repo_space_id="space-1"),
        RunCommitLink(run_id=run.run_id, commit_hash="ccc333", role="accepted", repo_space_id="space-1"),
    ]

    binding = bind_run_commits(links)
    assert binding.accepted_commit_hash == "ccc333"

    run.record_closeout(
        validation_evidence=["pytest tests/test_repo_links.py"],
        git_push_succeeded=True,
        git_log_stat_output="commit ccc333",
        run_commit_links=links,
    )

    assert run.closeout.accepted_commit_hash == "ccc333"
    restored = TaskRun.from_dict(run.to_dict())
    assert restored.closeout.accepted_commit_hash == "ccc333"
    assert restored.closeout.run_commit_links[1].role == "candidate"


def test_workspace_python_wrappers_are_removed_from_active_entrypoints():
    repo_root = Path(__file__).resolve().parents[1]
    deleted_wrappers = [
        repo_root / "scripts/ops/bigclaw_workspace_bootstrap.py",
        repo_root / "scripts/ops/symphony_workspace_bootstrap.py",
        repo_root / "scripts/ops/symphony_workspace_validate.py",
    ]

    for wrapper in deleted_wrappers:
        assert not wrapper.exists()

    active_files = [
        repo_root / "README.md",
        repo_root / "workflow.md",
        repo_root / ".github/workflows/ci.yml",
        repo_root / ".githooks/post-commit",
        repo_root / ".githooks/post-rewrite",
    ]
    deleted_names = {path.name for path in deleted_wrappers}

    for active_file in active_files:
        content = active_file.read_text()
        for deleted_name in deleted_names:
            assert deleted_name not in content

    workflow_content = (repo_root / "workflow.md").read_text()
    assert "bash \"$SYMPHONY_WORKFLOW_DIR/scripts/ops/bigclawctl\" workspace bootstrap" in workflow_content

    ci_content = (repo_root / ".github/workflows/ci.yml").read_text()
    assert "bash scripts/ops/bigclawctl github-sync --help >/dev/null" in ci_content
