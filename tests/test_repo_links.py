from bigclaw.models import Task
from bigclaw.observability import TaskRun
from bigclaw.repo_links import bind_run_commits
from bigclaw.repo_plane import RunCommitLink


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
