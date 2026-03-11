import hashlib
from pathlib import Path

from bigclaw.collaboration import build_collaboration_thread_from_audits
from bigclaw.models import Priority, Task
from bigclaw.observability import ObservabilityLedger, TaskRun
from bigclaw.reports import render_task_run_detail_page, render_task_run_report


def test_task_run_captures_logs_trace_artifacts_and_audits(tmp_path: Path):
    artifact = tmp_path / "validation.md"
    artifact.write_text("validation ok")
    expected_digest = hashlib.sha256(artifact.read_bytes()).hexdigest()

    task = Task(
        task_id="BIG-502",
        source="linear",
        title="Add observability",
        description="full chain",
        priority=Priority.P0,
    )
    run = TaskRun.from_task(task, run_id="run-1", medium="docker")
    run.log("info", "task accepted", queue="primary")
    run.trace("scheduler.decide", "ok", approved=True)
    run.register_artifact("validation-report", "report", str(artifact), environment="sandbox")
    run.audit("scheduler.approved", "system", "success", reason="default low risk path")
    run.record_closeout(
        validation_evidence=["pytest", "validation-report"],
        git_push_succeeded=True,
        git_push_output="Everything up-to-date",
        git_log_stat_output="commit abc123\n 1 file changed, 2 insertions(+)",
    )
    run.finalize("succeeded", "validation passed")

    ledger = ObservabilityLedger(str(tmp_path / "observability.json"))
    ledger.append(run)
    entries = ledger.load()

    assert len(entries) == 1
    assert entries[0]["status"] == "succeeded"
    assert entries[0]["logs"][0]["context"]["queue"] == "primary"
    assert entries[0]["traces"][0]["attributes"]["approved"] is True
    assert entries[0]["artifacts"][0]["sha256"] == expected_digest
    assert entries[0]["audits"][0]["action"] == "scheduler.approved"
    assert entries[0]["closeout"]["complete"] is True


def test_render_task_run_report(tmp_path: Path):
    artifact = tmp_path / "artifact.txt"
    artifact.write_text("audit trail")

    task = Task(task_id="BIG-502", source="linear", title="Observe execution", description="")
    run = TaskRun.from_task(task, run_id="run-2", medium="vm")
    run.log("warn", "approval required")
    run.trace("risk.review", "pending")
    run.register_artifact("approval-note", "note", str(artifact))
    run.audit("risk.review", "reviewer", "approved")
    comment = run.add_comment(
        author="ops-lead",
        body="Need @security sign-off before we clear this run.",
        mentions=["security"],
        anchor="closeout",
    )
    run.add_decision_note(
        author="security-reviewer",
        summary="Approved release after manual review.",
        outcome="approved",
        mentions=["ops-lead"],
        related_comment_ids=[comment.comment_id],
        follow_up="Share decision in the weekly review.",
    )
    run.record_closeout(
        validation_evidence=["pytest"],
        git_push_succeeded=True,
        git_push_output="main -> origin/main",
        git_log_stat_output="commit def456\n 1 file changed, 3 insertions(+)",
    )
    run.finalize("completed", "manual approval granted")

    report = render_task_run_report(run)

    assert "Run ID: run-2" in report
    assert "## Logs" in report
    assert "## Trace" in report
    assert "## Artifacts" in report
    assert "## Audit" in report
    assert "## Closeout" in report
    assert "Git Push Succeeded: True" in report
    assert "## Actions" in report
    assert "Retry [retry] state=disabled target=run-2 reason=retry is available for failed or approval-blocked runs" in report
    assert "## Collaboration" in report
    assert "Need @security sign-off before we clear this run." in report
    assert "Approved release after manual review." in report


def test_render_task_run_detail_page(tmp_path: Path):
    artifact = tmp_path / "artifact.txt"
    artifact.write_text("audit trail")

    task = Task(task_id="BIG-502", source="linear", title="Observe execution", description="")
    run = TaskRun.from_task(task, run_id="run-3", medium="browser")
    run.log("info", "opened detail page")
    run.trace("playback.render", "ok")
    run.register_artifact("approval-note", "note", str(artifact))
    run.audit("playback.render", "reviewer", "success")
    run.add_comment(
        author="pm",
        body="Loop in @design before we publish the replay.",
        mentions=["design"],
        anchor="overview",
    )
    run.add_decision_note(
        author="design",
        summary="Replay copy approved for external review.",
        outcome="approved",
        mentions=["pm"],
    )
    run.record_closeout(
        validation_evidence=["pytest", "playback-smoke"],
        git_push_succeeded=True,
        git_push_output="main -> origin/main",
        git_log_stat_output="commit fedcba\n 1 file changed, 1 insertion(+)",
    )
    run.finalize("approved", "detail page ready")

    page = render_task_run_detail_page(run)

    assert "<title>Task Run Detail" in page
    assert "Timeline / Log Sync" in page
    assert "data-detail=\"title\"" in page
    assert "Reports" in page
    assert "opened detail page" in page
    assert "playback.render" in page
    assert str(artifact) in page
    assert "detail page ready" in page
    assert "Closeout" in page
    assert "complete" in page
    assert "Actions" in page
    assert "Pause [pause] state=disabled target=run-3 reason=completed or failed runs cannot be paused" in page
    assert "Collaboration" in page
    assert "Loop in @design before we publish the replay." in page
    assert "Replay copy approved for external review." in page


def test_observability_ledger_load_runs_round_trips_entries(tmp_path: Path):
    task = Task(task_id="BIG-502-roundtrip", source="linear", title="Round trip", description="")
    run = TaskRun.from_task(task, run_id="run-roundtrip", medium="docker")
    run.log("info", "persisted")
    run.trace("scheduler.decide", "ok")
    run.audit("scheduler.decision", "scheduler", "approved", reason="default low risk path")
    run.add_comment(
        author="ops",
        body="Need @eng confirmation on the retry plan.",
        mentions=["eng"],
        anchor="timeline",
    )
    run.finalize("approved", "default low risk path")

    ledger = ObservabilityLedger(str(tmp_path / "observability.json"))
    ledger.append(run)

    loaded_runs = ledger.load_runs()

    assert len(loaded_runs) == 1
    assert loaded_runs[0].run_id == "run-roundtrip"
    assert loaded_runs[0].logs[0].message == "persisted"
    assert loaded_runs[0].traces[0].span == "scheduler.decide"
    assert loaded_runs[0].audits[0].details["reason"] == "default low risk path"
    collaboration = build_collaboration_thread_from_audits(
        [entry.to_dict() for entry in loaded_runs[0].audits],
        surface="run",
        target_id=loaded_runs[0].run_id,
    )
    assert collaboration is not None
    assert collaboration.mention_count == 1
    assert collaboration.comments[0].body == "Need @eng confirmation on the retry plan."
