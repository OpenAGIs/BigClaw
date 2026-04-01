package observability

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/collaboration"
	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/repo"
)

func TestTaskRunCapturesLogsTraceArtifactsAndAudits(t *testing.T) {
	artifactPath := filepath.Join(t.TempDir(), "validation.md")
	if err := os.WriteFile(artifactPath, []byte("validation ok"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	sum := sha256.Sum256([]byte("validation ok"))

	task := domain.Task{ID: "BIG-502", Source: "linear", Title: "Add observability", Description: "full chain"}
	run := NewTaskRun(task, "run-1", "docker")
	run.Log("info", "task accepted", map[string]any{"queue": "primary"})
	run.Trace("scheduler.decide", "ok", map[string]any{"approved": true})
	if err := run.RegisterArtifact("validation-report", "report", artifactPath, "sandbox"); err != nil {
		t.Fatalf("register artifact: %v", err)
	}
	run.Audit("scheduler.approved", "system", "success", map[string]any{"reason": "default low risk path"})
	run.RecordCloseout(Closeout{
		ValidationEvidence: []string{"pytest", "validation-report"},
		GitPushSucceeded:   true,
		GitPushOutput:      "Everything up-to-date",
		GitLogStatOutput:   "commit abc123\n 1 file changed, 2 insertions(+)",
	})
	run.Finalize("succeeded", "validation passed")

	ledger := NewLedger(filepath.Join(t.TempDir(), "observability.json"))
	if err := ledger.Append(run); err != nil {
		t.Fatalf("append ledger: %v", err)
	}
	entries, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	if len(entries) != 1 || entries[0]["status"] != "succeeded" {
		t.Fatalf("unexpected ledger entries: %+v", entries)
	}
	logs := entries[0]["logs"].([]any)
	if logs[0].(map[string]any)["context"].(map[string]any)["queue"] != "primary" {
		t.Fatalf("unexpected log context: %+v", logs[0])
	}
	traces := entries[0]["traces"].([]any)
	if traces[0].(map[string]any)["attributes"].(map[string]any)["approved"] != true {
		t.Fatalf("unexpected trace attributes: %+v", traces[0])
	}
	artifacts := entries[0]["artifacts"].([]any)
	if artifacts[0].(map[string]any)["sha256"] != hex.EncodeToString(sum[:]) {
		t.Fatalf("unexpected artifact digest: %+v", artifacts[0])
	}
	audits := entries[0]["audits"].([]any)
	actions := make([]string, 0, len(audits))
	for _, item := range audits {
		actions = append(actions, item.(map[string]any)["action"].(string))
	}
	for _, want := range []string{"artifact.registered", "closeout.recorded", "scheduler.approved"} {
		if !contains(actions, want) {
			t.Fatalf("expected audit action %q, got %v", want, actions)
		}
	}
	closeout := entries[0]["closeout"].(map[string]any)
	if closeout["complete"] != true {
		t.Fatalf("expected complete closeout, got %+v", closeout)
	}
}

func TestTaskRunCloseoutSerializesRepoSyncAudit(t *testing.T) {
	task := domain.Task{ID: "BIG-sync", Source: "linear", Title: "Repo sync closeout"}
	run := NewTaskRun(task, "run-sync", "docker")
	run.RecordCloseout(Closeout{
		ValidationEvidence: []string{"pytest"},
		GitPushSucceeded:   false,
		GitPushOutput:      "push rejected",
		GitLogStatOutput:   "commit abc123\n 1 file changed, 2 insertions(+)",
		RepoSyncAudit: &RepoSyncAudit{
			Sync:        GitSyncTelemetry{Status: "failed", FailureCategory: "dirty", Summary: "worktree has local changes", Branch: "feature/OPE-219", RemoteRef: "origin/feature/OPE-219", DirtyPaths: []string{"src/bigclaw/workflow.py"}},
			PullRequest: PullRequestFreshness{PRNumber: 219, PRURL: "https://github.com/OpenAGIs/BigClaw/pull/219", BranchState: "out-of-sync", BodyState: "drifted", BranchHeadSHA: "abc123", PRHeadSHA: "def456", ExpectedBodyDigest: "body-expected", ActualBodyDigest: "body-actual"},
		},
	})

	ledger := NewLedger(filepath.Join(t.TempDir(), "observability.json"))
	if err := ledger.Append(run); err != nil {
		t.Fatalf("append ledger: %v", err)
	}
	runs, err := ledger.LoadRuns()
	if err != nil {
		t.Fatalf("load runs: %v", err)
	}
	if runs[0].Closeout.RepoSyncAudit == nil || runs[0].Closeout.RepoSyncAudit.Sync.FailureCategory != "dirty" || runs[0].Closeout.RepoSyncAudit.PullRequest.BodyState != "drifted" {
		t.Fatalf("unexpected repo sync audit: %+v", runs[0].Closeout.RepoSyncAudit)
	}
}

func TestRenderTaskRunReport(t *testing.T) {
	artifactPath := filepath.Join(t.TempDir(), "artifact.txt")
	if err := os.WriteFile(artifactPath, []byte("audit trail"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	task := domain.Task{ID: "BIG-502", Source: "linear", Title: "Observe execution"}
	run := NewTaskRun(task, "run-2", "vm")
	run.Log("warn", "approval required", nil)
	run.Trace("risk.review", "pending", nil)
	if err := run.RegisterArtifact("approval-note", "note", artifactPath, ""); err != nil {
		t.Fatalf("register artifact: %v", err)
	}
	run.Audit("risk.review", "reviewer", "approved", nil)
	comment := run.AddComment("ops-lead", "Need @security sign-off before we clear this run.", []string{"security"}, "closeout")
	run.AddDecisionNote("security-reviewer", "Approved release after manual review.", "approved", []string{"ops-lead"}, []string{comment.CommentID}, "Share decision in the weekly review.")
	run.RecordCloseout(Closeout{ValidationEvidence: []string{"pytest"}, GitPushSucceeded: true, GitPushOutput: "main -> origin/main", GitLogStatOutput: "commit def456\n 1 file changed, 3 insertions(+)"})
	run.Finalize("completed", "manual approval granted")

	report := RenderTaskRunReport(*run)
	for _, fragment := range []string{"Run ID: run-2", "## Logs", "## Trace", "## Artifacts", "## Audit", "## Closeout", "Git Push Succeeded: True", "## Actions", "Retry [retry] state=disabled target=run-2 reason=retry is available for failed or approval-blocked runs", "## Collaboration", "Need @security sign-off before we clear this run.", "Approved release after manual review."} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}

func TestRenderRepoSyncAuditReport(t *testing.T) {
	report := RenderRepoSyncAuditReport(RepoSyncAudit{
		Sync:        GitSyncTelemetry{Status: "failed", FailureCategory: "auth", Summary: "github token expired", Branch: "dcjcloud/ope-219", RemoteRef: "origin/dcjcloud/ope-219", AuthTarget: "github.com/OpenAGIs/BigClaw.git"},
		PullRequest: PullRequestFreshness{PRNumber: 219, PRURL: "https://github.com/OpenAGIs/BigClaw/pull/219", BranchState: "in-sync", BodyState: "drifted", BranchHeadSHA: "abc123", PRHeadSHA: "abc123", ExpectedBodyDigest: "expected", ActualBodyDigest: "actual"},
	})

	for _, fragment := range []string{"# Repo Sync Audit", "Failure Category: auth", "Branch State: in-sync", "Body State: drifted", "sync=failed, failure=auth, pr-branch=in-sync, pr-body=drifted"} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}

func TestRenderTaskRunDetailPage(t *testing.T) {
	artifactPath := filepath.Join(t.TempDir(), "artifact.txt")
	if err := os.WriteFile(artifactPath, []byte("audit trail"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	task := domain.Task{ID: "BIG-502", Source: "linear", Title: "Observe execution"}
	run := NewTaskRun(task, "run-3", "browser")
	run.Log("info", "opened detail page", nil)
	run.Trace("playback.render", "ok", nil)
	if err := run.RegisterArtifact("approval-note", "note", artifactPath, ""); err != nil {
		t.Fatalf("register artifact: %v", err)
	}
	run.Audit("playback.render", "reviewer", "success", nil)
	run.AddComment("pm", "Loop in @design before we publish the replay.", []string{"design"}, "overview")
	run.AddDecisionNote("design", "Replay copy approved for external review.", "approved", []string{"pm"}, nil, "")
	run.RecordCloseout(Closeout{
		ValidationEvidence: []string{"pytest", "playback-smoke"},
		GitPushSucceeded:   true,
		GitPushOutput:      "main -> origin/main",
		GitLogStatOutput:   "commit fedcba\n 1 file changed, 1 insertion(+)",
		RunCommitLinks: []repo.RunCommitLink{
			{RunID: "run-3", CommitHash: "abc111", Role: "candidate", RepoSpaceID: "space-1"},
			{RunID: "run-3", CommitHash: "fedcba", Role: "accepted", RepoSpaceID: "space-1"},
		},
	})
	run.Finalize("approved", "detail page ready")

	page := RenderTaskRunDetailPage(*run)
	for _, fragment := range []string{"<title>Task Run Detail", "Timeline / Log Sync", "data-detail=\"title\"", "Reports", "opened detail page", "playback.render", artifactPath, "detail page ready", "Closeout", "complete", "Repo Evidence", "fedcba", "Actions", "Pause [pause] state=disabled target=run-3 reason=completed or failed runs cannot be paused", "Collaboration", "Loop in @design before we publish the replay.", "Replay copy approved for external review."} {
		if !strings.Contains(page, fragment) {
			t.Fatalf("expected %q in detail page, got %s", fragment, page)
		}
	}
}

func TestRenderTaskRunDetailPageEscapesTimelineJSONScriptBreakout(t *testing.T) {
	task := domain.Task{ID: "BIG-escape", Source: "linear", Title: "Escape check"}
	run := NewTaskRun(task, "run-escape", "browser")
	run.Log("info", "contains </script> marker", nil)
	run.Finalize("approved", "ok")

	page := RenderTaskRunDetailPage(*run)
	if !strings.Contains(page, "contains <\\/script> marker") {
		t.Fatalf("expected escaped script marker, got %s", page)
	}
}

func TestObservabilityLedgerLoadRunsRoundTripsEntries(t *testing.T) {
	task := domain.Task{ID: "BIG-502-roundtrip", Source: "linear", Title: "Round trip"}
	run := NewTaskRun(task, "run-roundtrip", "docker")
	run.Log("info", "persisted", nil)
	run.Trace("scheduler.decide", "ok", nil)
	run.Audit("scheduler.decision", "scheduler", "approved", map[string]any{"reason": "default low risk path"})
	run.AddComment("ops", "Need @eng confirmation on the retry plan.", []string{"eng"}, "timeline")
	run.Finalize("approved", "default low risk path")

	ledger := NewLedger(filepath.Join(t.TempDir(), "observability.json"))
	if err := ledger.Append(run); err != nil {
		t.Fatalf("append ledger: %v", err)
	}
	runs, err := ledger.LoadRuns()
	if err != nil {
		t.Fatalf("load runs: %v", err)
	}
	if len(runs) != 1 || runs[0].RunID != "run-roundtrip" || runs[0].Logs[0].Message != "persisted" || runs[0].Traces[0].Span != "scheduler.decide" || runs[0].Audits[0].Details["reason"] != "default low risk path" {
		t.Fatalf("unexpected run round trip: %+v", runs)
	}
	thread := collaboration.BuildCollaborationThreadFromAudits(auditMaps(runs[0].Audits), "run", runs[0].RunID)
	if thread == nil || thread.MentionCount != 1 || thread.Comments[0].Body != "Need @eng confirmation on the retry plan." {
		t.Fatalf("unexpected collaboration thread: %+v", thread)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
