package observability

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/domain"
)

func TestTaskRunCapturesLogsTraceArtifactsAuditsAndCloseout(t *testing.T) {
	artifactPath := filepath.Join(t.TempDir(), "validation.md")
	if err := os.WriteFile(artifactPath, []byte("validation ok"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	run := NewTaskRun(domain.Task{
		ID:       "BIG-502",
		Source:   "linear",
		Title:    "Add observability",
		Priority: 0,
	}, "run-1", "docker")
	run.Log("info", "task accepted", map[string]any{"queue": "primary"})
	run.Trace("scheduler.decide", "ok", map[string]any{"approved": true})
	if err := run.RegisterArtifact("validation-report", "report", artifactPath); err != nil {
		t.Fatalf("register artifact: %v", err)
	}
	run.Audit("scheduler.approved", "system", "success", map[string]any{"reason": "default low risk path"})
	run.RecordCloseout([]string{"pytest", "validation-report"}, true, "Everything up-to-date", "commit abc123\n 1 file changed, 2 insertions(+)", nil, nil)
	run.Finalize("succeeded", "validation passed")

	ledger := NewRunLedger(filepath.Join(t.TempDir(), "observability.json"))
	if err := ledger.Append(run); err != nil {
		t.Fatalf("append run: %v", err)
	}
	runs, err := ledger.LoadRuns()
	if err != nil {
		t.Fatalf("load runs: %v", err)
	}
	if len(runs) != 1 || runs[0].Status != "succeeded" {
		t.Fatalf("unexpected persisted runs: %+v", runs)
	}
	if runs[0].Logs[0].Context["queue"] != "primary" || runs[0].Traces[0].Attributes["approved"] != true || runs[0].Artifacts[0].SHA256 == "" {
		t.Fatalf("expected persisted log/trace/artifact data, got %+v", runs[0])
	}
	if !hasAudit(runs[0].Audits, "artifact.registered") || !hasAudit(runs[0].Audits, "closeout.recorded") || !hasAudit(runs[0].Audits, "scheduler.approved") {
		t.Fatalf("expected persisted audits, got %+v", runs[0].Audits)
	}
	if !runs[0].Closeout.Complete {
		t.Fatalf("expected complete closeout, got %+v", runs[0].Closeout)
	}
}

func TestTaskRunCloseoutSerializesRepoSyncAudit(t *testing.T) {
	run := NewTaskRun(domain.Task{ID: "BIG-sync", Source: "linear", Title: "Repo sync closeout"}, "run-sync", "docker")
	run.RecordCloseout([]string{"pytest"}, false, "push rejected", "commit abc123\n 1 file changed, 2 insertions(+)", &RepoSyncAudit{
		Sync: GitSyncTelemetry{
			Status:          "failed",
			FailureCategory: "dirty",
			Summary:         "worktree has local changes",
			Branch:          "feature/OPE-219",
			RemoteRef:       "origin/feature/OPE-219",
			DirtyPaths:      []string{"src/bigclaw/workflow.py"},
		},
		PullRequest: PullRequestFreshness{
			PRNumber:           219,
			PRURL:              "https://github.com/OpenAGIs/BigClaw/pull/219",
			BranchState:        "out-of-sync",
			BodyState:          "drifted",
			BranchHeadSHA:      "abc123",
			PRHeadSHA:          "def456",
			ExpectedBodyDigest: "body-expected",
			ActualBodyDigest:   "body-actual",
		},
	}, nil)

	ledger := NewRunLedger(filepath.Join(t.TempDir(), "observability.json"))
	if err := ledger.Append(run); err != nil {
		t.Fatalf("append run: %v", err)
	}
	runs, err := ledger.LoadRuns()
	if err != nil {
		t.Fatalf("load runs: %v", err)
	}
	if runs[0].Closeout.RepoSyncAudit == nil || runs[0].Closeout.RepoSyncAudit.Sync.FailureCategory != "dirty" || runs[0].Closeout.RepoSyncAudit.PullRequest.BodyState != "drifted" {
		t.Fatalf("unexpected repo sync audit payload: %+v", runs[0].Closeout.RepoSyncAudit)
	}
}

func TestRenderTaskRunReportAndRepoSyncAuditReport(t *testing.T) {
	artifactPath := filepath.Join(t.TempDir(), "artifact.txt")
	if err := os.WriteFile(artifactPath, []byte("audit trail"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	run := NewTaskRun(domain.Task{ID: "BIG-502", Source: "linear", Title: "Observe execution"}, "run-2", "vm")
	run.Log("warn", "approval required", nil)
	run.Trace("risk.review", "pending", nil)
	if err := run.RegisterArtifact("approval-note", "note", artifactPath); err != nil {
		t.Fatalf("register artifact: %v", err)
	}
	run.Audit("risk.review", "reviewer", "approved", nil)
	comment := run.AddComment("ops-lead", "Need @security sign-off before we clear this run.", []string{"security"}, "closeout")
	run.AddDecisionNote("security-reviewer", "Approved release after manual review.", "approved", []string{"ops-lead"}, "Share decision in the weekly review.")
	run.RecordCloseout([]string{"pytest"}, true, "main -> origin/main", "commit def456\n 1 file changed, 3 insertions(+)", nil, nil)
	run.Finalize("completed", "manual approval granted")

	report := RenderTaskRunReport(*run)
	for _, fragment := range []string{"Run ID: run-2", "## Logs", "## Trace", "## Artifacts", "## Audit", "## Closeout", "Git Push Succeeded: true", "## Actions", "Retry [retry] state=disabled target=run-2 reason=retry is available for failed or approval-blocked runs", "## Collaboration", comment.Body, "Approved release after manual review."} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in task run report, got %s", fragment, report)
		}
	}

	repoReport := RenderRepoSyncAuditReport(RepoSyncAudit{
		Sync: GitSyncTelemetry{
			Status:          "failed",
			FailureCategory: "auth",
			Summary:         "github token expired",
			Branch:          "dcjcloud/ope-219",
			RemoteRef:       "origin/dcjcloud/ope-219",
			AuthTarget:      "github.com/OpenAGIs/BigClaw.git",
		},
		PullRequest: PullRequestFreshness{
			PRNumber:           219,
			PRURL:              "https://github.com/OpenAGIs/BigClaw/pull/219",
			BranchState:        "in-sync",
			BodyState:          "drifted",
			BranchHeadSHA:      "abc123",
			PRHeadSHA:          "abc123",
			ExpectedBodyDigest: "expected",
			ActualBodyDigest:   "actual",
		},
	})
	for _, fragment := range []string{"# Repo Sync Audit", "Failure Category: auth", "Branch State: in-sync", "Body State: drifted", "sync=failed, failure=auth, pr-branch=in-sync, pr-body=drifted"} {
		if !strings.Contains(repoReport, fragment) {
			t.Fatalf("expected %q in repo sync report, got %s", fragment, repoReport)
		}
	}
}

func TestRenderTaskRunDetailPage(t *testing.T) {
	artifactPath := filepath.Join(t.TempDir(), "artifact.txt")
	if err := os.WriteFile(artifactPath, []byte("audit trail"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	run := NewTaskRun(domain.Task{ID: "BIG-502", Source: "linear", Title: "Observe execution"}, "run-3", "browser")
	run.Log("info", "opened detail page", nil)
	run.Trace("playback.render", "ok", nil)
	if err := run.RegisterArtifact("approval-note", "note", artifactPath); err != nil {
		t.Fatalf("register artifact: %v", err)
	}
	run.Audit("playback.render", "reviewer", "success", nil)
	run.AddComment("pm", "Loop in @design before we publish the replay.", []string{"design"}, "overview")
	run.AddDecisionNote("design", "Replay copy approved for external review.", "approved", []string{"pm"}, "")
	run.RecordCloseout([]string{"pytest", "playback-smoke"}, true, "main -> origin/main", "commit fedcba\n 1 file changed, 1 insertion(+)", nil, []RunCommitLink{
		{RunID: "run-3", CommitHash: "abc111", Role: "candidate", RepoSpaceID: "space-1"},
		{RunID: "run-3", CommitHash: "fedcba", Role: "accepted", RepoSpaceID: "space-1"},
	})
	run.Finalize("approved", "detail page ready")

	page := RenderTaskRunDetailPage(*run)
	for _, fragment := range []string{"<title>Task Run Detail", "Timeline / Log Sync", "data-detail=\"title\"", "Reports", "opened detail page", "playback.render", artifactPath, "detail page ready", "Closeout", "complete", "Repo Evidence", "fedcba", "Actions", "Pause [pause] state=disabled target=run-3 reason=completed or failed runs cannot be paused"} {
		if !strings.Contains(page, fragment) {
			t.Fatalf("expected %q in task run detail page, got %s", fragment, page)
		}
	}
}

func hasAudit(audits []AuditItem, action string) bool {
	for _, audit := range audits {
		if audit.Action == action {
			return true
		}
	}
	return false
}
