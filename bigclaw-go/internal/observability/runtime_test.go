package observability

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/repo"
)

func TestTaskRunRecordCloseoutAndRepoSyncSummary(t *testing.T) {
	now := time.Date(2026, 3, 29, 8, 0, 0, 0, time.UTC)
	run := NewTaskRunFromTask(domain.Task{ID: "BIG-944", Source: "github", Title: "Lane closeout"}, "run-944", "docker", now)
	audit := &RepoSyncAudit{
		Sync:        GitSyncTelemetry{Status: "synced", Branch: "big-go-944", Remote: "origin", Timestamp: now},
		PullRequest: PullRequestFreshness{BranchState: "in-sync", BodyState: "fresh", CheckedAt: now},
	}
	err := run.RecordCloseout(
		[]string{"go test ./internal/planning"},
		true,
		"pushed",
		"abc123 big-go-944",
		audit,
		[]repo.RunCommitLink{
			{RunID: "run-944", CommitHash: "aaa111", Role: "candidate", RepoSpaceID: "repo-space"},
			{RunID: "run-944", CommitHash: "bbb222", Role: "accepted", RepoSpaceID: "repo-space"},
		},
		now.Add(time.Minute),
	)
	if err != nil {
		t.Fatalf("record closeout: %v", err)
	}
	if !run.Closeout.Complete() || run.Closeout.AcceptedCommitHash != "bbb222" {
		t.Fatalf("unexpected closeout: %+v", run.Closeout)
	}
	if got := audit.Summary(); got != "sync=synced, pr-branch=in-sync, pr-body=fresh" {
		t.Fatalf("unexpected repo sync summary: %s", got)
	}
}

func TestTaskRunRegisterArtifactAndLedgerRoundTrip(t *testing.T) {
	now := time.Date(2026, 3, 29, 8, 0, 0, 0, time.UTC)
	root := t.TempDir()
	artifactPath := filepath.Join(root, "report.md")
	if err := os.WriteFile(artifactPath, []byte("lane4"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	run := NewTaskRunFromTask(domain.Task{ID: "BIG-944", Title: "Lane closeout"}, "run-944", "docker", now)
	if err := run.RegisterArtifact("weekly-report", "markdown", artifactPath, now, map[string]any{"lane": 4}); err != nil {
		t.Fatalf("register artifact: %v", err)
	}
	if len(run.Artifacts) != 1 || run.Artifacts[0].SHA256 == "" {
		t.Fatalf("expected artifact digest, got %+v", run.Artifacts)
	}
	if len(run.Audits) != 1 || run.Audits[0].Action != "artifact.registered" {
		t.Fatalf("expected artifact audit, got %+v", run.Audits)
	}
	ledger := NewLedger(filepath.Join(root, "ledger", "runs.json"))
	if err := ledger.Append(run); err != nil {
		t.Fatalf("append run: %v", err)
	}
	run.Finalize("succeeded", "closeout finished", now.Add(2*time.Minute))
	if err := ledger.Upsert(run); err != nil {
		t.Fatalf("upsert run: %v", err)
	}
	runs, err := ledger.LoadRuns()
	if err != nil {
		t.Fatalf("load runs: %v", err)
	}
	if len(runs) != 1 || runs[0].Status != "succeeded" || runs[0].Summary != "closeout finished" {
		t.Fatalf("unexpected ledger runs: %+v", runs)
	}
}

func TestTaskRunRejectsUnsupportedCommitRoles(t *testing.T) {
	run := NewTaskRunFromTask(domain.Task{ID: "BIG-944", Title: "Lane closeout"}, "run-944", "docker", time.Now())
	err := run.RecordCloseout(nil, true, "", "git log", nil, []repo.RunCommitLink{{RunID: "run-944", CommitHash: "aaa111", Role: "invalid", RepoSpaceID: "repo-space"}}, time.Now())
	if err == nil || !strings.Contains(err.Error(), "unsupported run commit roles") {
		t.Fatalf("expected invalid role error, got %v", err)
	}
}
