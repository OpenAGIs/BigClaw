package observability

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/repo"
)

func TestTaskRunCapturesLogsTraceArtifactsAndAudits(t *testing.T) {
	tmp := t.TempDir()
	artifact := filepath.Join(tmp, "validation.md")
	if err := os.WriteFile(artifact, []byte("validation ok"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	sum := sha256.Sum256([]byte("validation ok"))
	expectedDigest := hex.EncodeToString(sum[:])

	run := NewTaskRunFromTask(domain.Task{
		ID:     "BIG-502",
		Source: "linear",
		Title:  "Add observability",
	}, "run-1", "docker")
	run.Log("info", "task accepted", map[string]any{"queue": "primary"})
	run.Trace("scheduler.decide", "ok", map[string]any{"approved": true})
	run.RegisterArtifact("validation-report", "report", artifact, map[string]any{"environment": "sandbox"})
	run.Audit("scheduler.approved", "system", "success", map[string]any{"reason": "default low risk path"})
	run.RecordCloseout(
		[]string{"pytest", "validation-report"},
		true,
		"Everything up-to-date",
		"commit abc123\n 1 file changed, 2 insertions(+)",
		nil,
		nil,
	)
	run.Finalize("succeeded", "validation passed")

	ledger := ObservabilityLedger{StoragePath: filepath.Join(tmp, "observability.json")}
	if err := ledger.Append(run); err != nil {
		t.Fatalf("append run: %v", err)
	}
	entries, err := ledger.Load()
	if err != nil {
		t.Fatalf("load entries: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0]["status"] != "succeeded" {
		t.Fatalf("expected succeeded status, got %#v", entries[0]["status"])
	}
	logs := entries[0]["logs"].([]any)
	if logs[0].(map[string]any)["context"].(map[string]any)["queue"] != "primary" {
		t.Fatalf("expected queue context in logs, got %#v", logs[0])
	}
	traces := entries[0]["traces"].([]any)
	if traces[0].(map[string]any)["attributes"].(map[string]any)["approved"] != true {
		t.Fatalf("expected approved trace attribute, got %#v", traces[0])
	}
	artifacts := entries[0]["artifacts"].([]any)
	if artifacts[0].(map[string]any)["sha256"] != expectedDigest {
		t.Fatalf("expected digest %s, got %#v", expectedDigest, artifacts[0].(map[string]any)["sha256"])
	}
	audits := entries[0]["audits"].([]any)
	actions := make(map[string]bool, len(audits))
	for _, value := range audits {
		actions[value.(map[string]any)["action"].(string)] = true
	}
	for _, want := range []string{"artifact.registered", "closeout.recorded", "scheduler.approved"} {
		if !actions[want] {
			t.Fatalf("expected audit action %q in %#v", want, actions)
		}
	}
	closeout := entries[0]["closeout"].(map[string]any)
	if closeout["complete"] != true {
		t.Fatalf("expected complete closeout, got %#v", closeout)
	}
}

func TestTaskRunCloseoutSerializesRepoSyncAudit(t *testing.T) {
	run := NewTaskRunFromTask(domain.Task{
		ID:     "BIG-sync",
		Source: "linear",
		Title:  "Repo sync closeout",
	}, "run-sync", "docker")
	prNumber := 219
	repoSyncAudit := &RepoSyncAudit{
		Sync: GitSyncTelemetry{
			Status:          "failed",
			FailureCategory: "dirty",
			Summary:         "worktree has local changes",
			Branch:          "feature/OPE-219",
			RemoteRef:       "origin/feature/OPE-219",
			DirtyPaths:      []string{"src/bigclaw/workflow.py"},
		},
		PullRequest: PullRequestFreshness{
			PRNumber:           &prNumber,
			PRURL:              "https://github.com/OpenAGIs/BigClaw/pull/219",
			BranchState:        "out-of-sync",
			BodyState:          "drifted",
			BranchHeadSHA:      "abc123",
			PRHeadSHA:          "def456",
			ExpectedBodyDigest: "body-expected",
			ActualBodyDigest:   "body-actual",
		},
	}
	run.RecordCloseout(
		[]string{"pytest"},
		false,
		"push rejected",
		"commit abc123\n 1 file changed, 2 insertions(+)",
		repoSyncAudit,
		nil,
	)

	ledger := ObservabilityLedger{StoragePath: filepath.Join(t.TempDir(), "observability.json")}
	if err := ledger.Append(run); err != nil {
		t.Fatalf("append run: %v", err)
	}
	loadedRuns, err := ledger.LoadRuns()
	if err != nil {
		t.Fatalf("load runs: %v", err)
	}

	if len(loadedRuns) != 1 || loadedRuns[0].Closeout.RepoSyncAudit == nil {
		t.Fatalf("expected repo sync audit in loaded run, got %+v", loadedRuns)
	}
	if loadedRuns[0].Closeout.RepoSyncAudit.Sync.FailureCategory != "dirty" {
		t.Fatalf("expected dirty failure category, got %+v", loadedRuns[0].Closeout.RepoSyncAudit.Sync)
	}
	if loadedRuns[0].Closeout.RepoSyncAudit.PullRequest.BodyState != "drifted" {
		t.Fatalf("expected drifted PR body state, got %+v", loadedRuns[0].Closeout.RepoSyncAudit.PullRequest)
	}
}

func TestObservabilityLedgerLoadRunsRoundTripsEntries(t *testing.T) {
	run := NewTaskRunFromTask(domain.Task{
		ID:     "BIG-502-roundtrip",
		Source: "linear",
		Title:  "Round trip",
	}, "run-roundtrip", "docker")
	run.Log("info", "persisted", nil)
	run.Trace("scheduler.decide", "ok", nil)
	run.Audit("scheduler.decision", "scheduler", "approved", map[string]any{"reason": "default low risk path"})
	run.AddComment("ops", "Need @eng confirmation on the retry plan.", []string{"eng"}, "timeline")
	run.Finalize("approved", "default low risk path")

	ledger := ObservabilityLedger{StoragePath: filepath.Join(t.TempDir(), "observability.json")}
	if err := ledger.Append(run); err != nil {
		t.Fatalf("append run: %v", err)
	}
	loadedRuns, err := ledger.LoadRuns()
	if err != nil {
		t.Fatalf("load runs: %v", err)
	}

	if len(loadedRuns) != 1 {
		t.Fatalf("expected 1 loaded run, got %d", len(loadedRuns))
	}
	loaded := loadedRuns[0]
	if loaded.RunID != "run-roundtrip" {
		t.Fatalf("expected run id run-roundtrip, got %q", loaded.RunID)
	}
	if loaded.Logs[0].Message != "persisted" {
		t.Fatalf("expected persisted log message, got %+v", loaded.Logs)
	}
	if loaded.Traces[0].Span != "scheduler.decide" {
		t.Fatalf("expected scheduler.decide trace span, got %+v", loaded.Traces)
	}
	if loaded.Audits[0].Details["reason"] != "default low risk path" {
		t.Fatalf("expected scheduler decision reason, got %+v", loaded.Audits[0].Details)
	}
	mentions := 0
	var commentBody string
	for _, audit := range loaded.Audits {
		if audit.Action != "collaboration.comment" {
			continue
		}
		values, _ := audit.Details["mentions"].([]any)
		mentions += len(values)
		commentBody, _ = audit.Details["body"].(string)
	}
	if mentions != 1 {
		t.Fatalf("expected 1 collaboration mention, got %d in %+v", mentions, loaded.Audits)
	}
	if commentBody != "Need @eng confirmation on the retry plan." {
		t.Fatalf("expected collaboration comment body, got %q", commentBody)
	}
}

func TestTaskRunCloseoutBindsAcceptedCommitHash(t *testing.T) {
	run := NewTaskRunFromTask(domain.Task{ID: "BIG-commit", Title: "Commit closeout"}, "run-3", "browser")
	run.RecordCloseout(
		[]string{"pytest"},
		true,
		"main -> origin/main",
		"commit fedcba\n 1 file changed, 1 insertion(+)",
		nil,
		[]repo.RunCommitLink{
			{RunID: "run-3", CommitHash: "abc111", Role: "candidate", RepoSpaceID: "space-1"},
			{RunID: "run-3", CommitHash: "fedcba", Role: "accepted", RepoSpaceID: "space-1"},
		},
	)
	if run.Closeout.AcceptedCommitHash != "fedcba" {
		t.Fatalf("expected accepted commit hash fedcba, got %q", run.Closeout.AcceptedCommitHash)
	}
}
