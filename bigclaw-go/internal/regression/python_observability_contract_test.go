package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLane8PythonObservabilityContractStaysAligned(t *testing.T) {
	goRepoRoot := repoRoot(t)
	repoRoot := filepath.Clean(filepath.Join(goRepoRoot, ".."))
	scriptPath := filepath.Join(t.TempDir(), "observability_contract.py")
	script := `import hashlib
import json
import sys
import tempfile
from pathlib import Path

repo_root = Path(sys.argv[1])
sys.path.insert(0, str(repo_root / "src"))

from bigclaw.collaboration import build_collaboration_thread_from_audits
from bigclaw.models import Priority, Task
from bigclaw.observability import GitSyncTelemetry, ObservabilityLedger, PullRequestFreshness, RepoSyncAudit, TaskRun
from bigclaw.reports import render_repo_sync_audit_report, render_task_run_detail_page, render_task_run_report
from bigclaw.repo_plane import RunCommitLink

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    artifact = td / "validation.md"
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

    ledger = ObservabilityLedger(str(td / "observability.json"))
    ledger.append(run)
    entries = ledger.load()
    captured = {
        "entry_count": len(entries),
        "status": entries[0]["status"],
        "queue": entries[0]["logs"][0]["context"]["queue"],
        "trace_approved": entries[0]["traces"][0]["attributes"]["approved"],
        "artifact_sha": entries[0]["artifacts"][0]["sha256"],
        "audit_actions": [item["action"] for item in entries[0]["audits"]],
        "closeout_complete": entries[0]["closeout"]["complete"],
        "expected_digest": expected_digest,
    }

with tempfile.TemporaryDirectory() as td:
    task = Task(task_id="BIG-sync", source="linear", title="Repo sync closeout", description="")
    run = TaskRun.from_task(task, run_id="run-sync", medium="docker")
    repo_sync_audit = RepoSyncAudit(
        sync=GitSyncTelemetry(
            status="failed",
            failure_category="dirty",
            summary="worktree has local changes",
            branch="feature/OPE-219",
            remote_ref="origin/feature/OPE-219",
            dirty_paths=["src/bigclaw/workflow.py"],
        ),
        pull_request=PullRequestFreshness(
            pr_number=219,
            pr_url="https://github.com/OpenAGIs/BigClaw/pull/219",
            branch_state="out-of-sync",
            body_state="drifted",
            branch_head_sha="abc123",
            pr_head_sha="def456",
            expected_body_digest="body-expected",
            actual_body_digest="body-actual",
        ),
    )
    run.record_closeout(
        validation_evidence=["pytest"],
        git_push_succeeded=False,
        git_push_output="push rejected",
        git_log_stat_output="commit abc123\n 1 file changed, 2 insertions(+)",
        repo_sync_audit=repo_sync_audit,
    )
    ledger = ObservabilityLedger(str(Path(td) / "observability.json"))
    ledger.append(run)
    loaded_run = ledger.load_runs()[0]
    closeout = {
        "has_repo_sync": loaded_run.closeout.repo_sync_audit is not None,
        "failure_category": loaded_run.closeout.repo_sync_audit.sync.failure_category,
        "body_state": loaded_run.closeout.repo_sync_audit.pull_request.body_state,
    }

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    artifact = td / "artifact.txt"
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
    task_run_report = {
        "has_run_id": "Run ID: run-2" in report,
        "has_logs": "## Logs" in report,
        "has_trace": "## Trace" in report,
        "has_artifacts": "## Artifacts" in report,
        "has_audit": "## Audit" in report,
        "has_closeout": "## Closeout" in report,
        "has_git_push": "Git Push Succeeded: True" in report,
        "has_actions": "## Actions" in report,
        "has_retry": "Retry [retry] state=disabled target=run-2 reason=retry is available for failed or approval-blocked runs" in report,
        "has_collaboration": "## Collaboration" in report,
        "has_comment": "Need @security sign-off before we clear this run." in report,
        "has_decision": "Approved release after manual review." in report,
    }

audit = RepoSyncAudit(
    sync=GitSyncTelemetry(
        status="failed",
        failure_category="auth",
        summary="github token expired",
        branch="dcjcloud/ope-219",
        remote_ref="origin/dcjcloud/ope-219",
        auth_target="github.com/OpenAGIs/BigClaw.git",
    ),
    pull_request=PullRequestFreshness(
        pr_number=219,
        pr_url="https://github.com/OpenAGIs/BigClaw/pull/219",
        branch_state="in-sync",
        body_state="drifted",
        branch_head_sha="abc123",
        pr_head_sha="abc123",
        expected_body_digest="expected",
        actual_body_digest="actual",
    ),
)
repo_sync_report = render_repo_sync_audit_report(audit)
repo_sync = {
    "has_title": "# Repo Sync Audit" in repo_sync_report,
    "has_failure_category": "Failure Category: auth" in repo_sync_report,
    "has_branch_state": "Branch State: in-sync" in repo_sync_report,
    "has_body_state": "Body State: drifted" in repo_sync_report,
    "has_summary_line": "sync=failed, failure=auth, pr-branch=in-sync, pr-body=drifted" in repo_sync_report,
}

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    artifact = td / "artifact.txt"
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
        run_commit_links=[
            RunCommitLink(run_id="run-3", commit_hash="abc111", role="candidate", repo_space_id="space-1"),
            RunCommitLink(run_id="run-3", commit_hash="fedcba", role="accepted", repo_space_id="space-1"),
        ],
    )
    run.finalize("approved", "detail page ready")
    page = render_task_run_detail_page(run)
    detail_page = {
        "has_title": "<title>Task Run Detail" in page,
        "has_timeline": "Timeline / Log Sync" in page,
        "has_data_detail": "data-detail=\"title\"" in page,
        "has_reports": "Reports" in page,
        "has_log": "opened detail page" in page,
        "has_trace": "playback.render" in page,
        "has_artifact": str(artifact) in page,
        "has_summary": "detail page ready" in page,
        "has_closeout": "Closeout" in page,
        "has_complete": "complete" in page,
        "has_repo_evidence": "Repo Evidence" in page,
        "has_commit": "fedcba" in page,
        "has_actions": "Actions" in page,
        "has_pause": "Pause [pause] state=disabled target=run-3 reason=completed or failed runs cannot be paused" in page,
        "has_collaboration": "Collaboration" in page,
        "has_comment": "Loop in @design before we publish the replay." in page,
        "has_decision": "Replay copy approved for external review." in page,
    }

task = Task(task_id="BIG-escape", source="linear", title="Escape check", description="")
run = TaskRun.from_task(task, run_id="run-escape", medium="browser")
run.log("info", "contains </script> marker")
run.finalize("approved", "ok")
escaped_page = render_task_run_detail_page(run)
escaped = {"has_escaped_script": "contains <\\/script> marker" in escaped_page}

with tempfile.TemporaryDirectory() as td:
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
    ledger = ObservabilityLedger(str(Path(td) / "observability.json"))
    ledger.append(run)
    loaded_runs = ledger.load_runs()
    collaboration = build_collaboration_thread_from_audits(
        [entry.to_dict() for entry in loaded_runs[0].audits],
        surface="run",
        target_id=loaded_runs[0].run_id,
    )
    roundtrip = {
        "count": len(loaded_runs),
        "run_id": loaded_runs[0].run_id,
        "log_message": loaded_runs[0].logs[0].message,
        "trace_span": loaded_runs[0].traces[0].span,
        "audit_reason": loaded_runs[0].audits[0].details["reason"],
        "has_collaboration": collaboration is not None,
        "mention_count": collaboration.mention_count,
        "comment_body": collaboration.comments[0].body,
    }

print(json.dumps({
    "captured": captured,
    "closeout": closeout,
    "task_run_report": task_run_report,
    "repo_sync": repo_sync,
    "detail_page": detail_page,
    "escaped": escaped,
    "roundtrip": roundtrip,
}))
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write observability contract script: %v", err)
	}

	cmd := exec.Command("python3", scriptPath, repoRoot)
	cmd.Dir = goRepoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run observability contract script: %v\n%s", err, string(output))
	}

	var decoded struct {
		Captured struct {
			EntryCount       int      `json:"entry_count"`
			Status           string   `json:"status"`
			Queue            string   `json:"queue"`
			TraceApproved    bool     `json:"trace_approved"`
			ArtifactSHA      string   `json:"artifact_sha"`
			AuditActions     []string `json:"audit_actions"`
			CloseoutComplete bool     `json:"closeout_complete"`
			ExpectedDigest   string   `json:"expected_digest"`
		} `json:"captured"`
		Closeout struct {
			HasRepoSync     bool   `json:"has_repo_sync"`
			FailureCategory string `json:"failure_category"`
			BodyState       string `json:"body_state"`
		} `json:"closeout"`
		TaskRunReport struct {
			HasRunID        bool `json:"has_run_id"`
			HasLogs         bool `json:"has_logs"`
			HasTrace        bool `json:"has_trace"`
			HasArtifacts    bool `json:"has_artifacts"`
			HasAudit        bool `json:"has_audit"`
			HasCloseout     bool `json:"has_closeout"`
			HasGitPush      bool `json:"has_git_push"`
			HasActions      bool `json:"has_actions"`
			HasRetry        bool `json:"has_retry"`
			HasCollaboration bool `json:"has_collaboration"`
			HasComment      bool `json:"has_comment"`
			HasDecision     bool `json:"has_decision"`
		} `json:"task_run_report"`
		RepoSync struct {
			HasTitle           bool `json:"has_title"`
			HasFailureCategory bool `json:"has_failure_category"`
			HasBranchState     bool `json:"has_branch_state"`
			HasBodyState       bool `json:"has_body_state"`
			HasSummaryLine     bool `json:"has_summary_line"`
		} `json:"repo_sync"`
		DetailPage struct {
			HasTitle         bool `json:"has_title"`
			HasTimeline      bool `json:"has_timeline"`
			HasDataDetail    bool `json:"has_data_detail"`
			HasReports       bool `json:"has_reports"`
			HasLog           bool `json:"has_log"`
			HasTrace         bool `json:"has_trace"`
			HasArtifact      bool `json:"has_artifact"`
			HasSummary       bool `json:"has_summary"`
			HasCloseout      bool `json:"has_closeout"`
			HasComplete      bool `json:"has_complete"`
			HasRepoEvidence  bool `json:"has_repo_evidence"`
			HasCommit        bool `json:"has_commit"`
			HasActions       bool `json:"has_actions"`
			HasPause         bool `json:"has_pause"`
			HasCollaboration bool `json:"has_collaboration"`
			HasComment       bool `json:"has_comment"`
			HasDecision      bool `json:"has_decision"`
		} `json:"detail_page"`
		Escaped struct {
			HasEscapedScript bool `json:"has_escaped_script"`
		} `json:"escaped"`
		Roundtrip struct {
			Count            int    `json:"count"`
			RunID            string `json:"run_id"`
			LogMessage       string `json:"log_message"`
			TraceSpan        string `json:"trace_span"`
			AuditReason      string `json:"audit_reason"`
			HasCollaboration bool   `json:"has_collaboration"`
			MentionCount     int    `json:"mention_count"`
			CommentBody      string `json:"comment_body"`
		} `json:"roundtrip"`
	}
	if err := json.Unmarshal(output, &decoded); err != nil {
		t.Fatalf("decode observability contract output: %v\n%s", err, string(output))
	}

	if decoded.Captured.EntryCount != 1 || decoded.Captured.Status != "succeeded" || decoded.Captured.Queue != "primary" || !decoded.Captured.TraceApproved || decoded.Captured.ArtifactSHA != decoded.Captured.ExpectedDigest || !containsAuditAction(decoded.Captured.AuditActions, "artifact.registered") || !containsAuditAction(decoded.Captured.AuditActions, "closeout.recorded") || !containsAuditAction(decoded.Captured.AuditActions, "scheduler.approved") || !decoded.Captured.CloseoutComplete {
		t.Fatalf("unexpected captured observability payload: %+v", decoded.Captured)
	}
	if !decoded.Closeout.HasRepoSync || decoded.Closeout.FailureCategory != "dirty" || decoded.Closeout.BodyState != "drifted" {
		t.Fatalf("unexpected closeout serialization payload: %+v", decoded.Closeout)
	}
	if !decoded.TaskRunReport.HasRunID || !decoded.TaskRunReport.HasLogs || !decoded.TaskRunReport.HasTrace || !decoded.TaskRunReport.HasArtifacts || !decoded.TaskRunReport.HasAudit || !decoded.TaskRunReport.HasCloseout || !decoded.TaskRunReport.HasGitPush || !decoded.TaskRunReport.HasActions || !decoded.TaskRunReport.HasRetry || !decoded.TaskRunReport.HasCollaboration || !decoded.TaskRunReport.HasComment || !decoded.TaskRunReport.HasDecision {
		t.Fatalf("unexpected task run report payload: %+v", decoded.TaskRunReport)
	}
	if !decoded.RepoSync.HasTitle || !decoded.RepoSync.HasFailureCategory || !decoded.RepoSync.HasBranchState || !decoded.RepoSync.HasBodyState || !decoded.RepoSync.HasSummaryLine {
		t.Fatalf("unexpected repo sync report payload: %+v", decoded.RepoSync)
	}
	if !decoded.DetailPage.HasTitle || !decoded.DetailPage.HasTimeline || !decoded.DetailPage.HasDataDetail || !decoded.DetailPage.HasReports || !decoded.DetailPage.HasLog || !decoded.DetailPage.HasTrace || !decoded.DetailPage.HasArtifact || !decoded.DetailPage.HasSummary || !decoded.DetailPage.HasCloseout || !decoded.DetailPage.HasComplete || !decoded.DetailPage.HasRepoEvidence || !decoded.DetailPage.HasCommit || !decoded.DetailPage.HasActions || !decoded.DetailPage.HasPause || !decoded.DetailPage.HasCollaboration || !decoded.DetailPage.HasComment || !decoded.DetailPage.HasDecision {
		t.Fatalf("unexpected detail page payload: %+v", decoded.DetailPage)
	}
	if !decoded.Escaped.HasEscapedScript {
		t.Fatalf("unexpected detail page escaping payload: %+v", decoded.Escaped)
	}
	if decoded.Roundtrip.Count != 1 || decoded.Roundtrip.RunID != "run-roundtrip" || decoded.Roundtrip.LogMessage != "persisted" || decoded.Roundtrip.TraceSpan != "scheduler.decide" || decoded.Roundtrip.AuditReason != "default low risk path" || !decoded.Roundtrip.HasCollaboration || decoded.Roundtrip.MentionCount != 1 || decoded.Roundtrip.CommentBody != "Need @eng confirmation on the retry plan." {
		t.Fatalf("unexpected round-trip payload: %+v", decoded.Roundtrip)
	}
}
