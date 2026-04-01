package repo

import (
	"reflect"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
)

func TestPythonParityRepoDiscussionBoardLifecycle(t *testing.T) {
	board := RepoDiscussionBoard{Now: func() time.Time { return time.Date(2026, 4, 1, 8, 0, 0, 0, time.UTC) }}

	post := board.CreatePost("bigclaw-ope-164", "agent-a", "Need reviewer on commit lineage", "run", "run-164", map[string]any{"severity": "p1"})
	reply, err := board.Reply(post.PostID, "reviewer", "I will review this now")
	if err != nil {
		t.Fatalf("reply: %v", err)
	}

	if post.PostID != "post-1" {
		t.Fatalf("unexpected post id: %q", post.PostID)
	}
	if reply.ParentPostID != "post-1" {
		t.Fatalf("unexpected parent post id: %q", reply.ParentPostID)
	}

	runPosts := board.ListPosts("", "run", "run-164")
	if len(runPosts) != 2 {
		t.Fatalf("expected two run posts, got %d", len(runPosts))
	}
	if runPosts[0].Channel != "bigclaw-ope-164" {
		t.Fatalf("unexpected filtered channel: %q", runPosts[0].Channel)
	}
	if runPosts[0].TargetSurface != "run" || runPosts[0].TargetID != "run-164" {
		t.Fatalf("unexpected target filter payload: %+v", runPosts[0])
	}
}

func TestPythonParityPermissionMatrixAndAuditRequirements(t *testing.T) {
	contract := NewPermissionContract()

	if !contract.Check("repo.push", []string{"eng-lead"}) {
		t.Fatal("expected eng-lead to be allowed repo.push")
	}
	if !contract.Check("repo.accept", []string{"reviewer"}) {
		t.Fatal("expected reviewer to be allowed repo.accept")
	}
	if contract.Check("repo.push", []string{"execution-agent"}) {
		t.Fatal("expected execution-agent to be denied repo.push")
	}

	missing := MissingAuditFields("repo.accept", map[string]any{
		"task_id":       "OPE-172",
		"run_id":        "run-172",
		"repo_space_id": "space-1",
		"actor":         "reviewer",
	})
	if !reflect.DeepEqual(missing, []string{"accepted_commit_hash", "reviewer"}) {
		t.Fatalf("unexpected missing audit fields: %+v", missing)
	}
}

func TestPythonParityGatewayNormalizationAndAuditPayload(t *testing.T) {
	commit, err := NormalizeCommit(map[string]any{"commit_hash": "abc123", "title": "feat: add repo plane", "author": "bot"})
	if err != nil {
		t.Fatalf("normalize commit: %v", err)
	}
	if commit.CommitHash != "abc123" {
		t.Fatalf("unexpected commit hash: %+v", commit)
	}

	lineage, err := NormalizeLineage(map[string]any{
		"root_hash": "abc123",
		"lineage":   []map[string]any{{"commit_hash": "abc123", "title": "feat: add repo plane", "author": "bot"}},
		"children":  map[string][]string{"abc123": {"def456"}},
		"leaves":    []string{"def456"},
	})
	if err != nil {
		t.Fatalf("normalize lineage: %v", err)
	}
	if !reflect.DeepEqual(lineage.Leaves, []string{"def456"}) {
		t.Fatalf("unexpected lineage leaves: %+v", lineage.Leaves)
	}

	diff, err := NormalizeDiff(map[string]any{
		"left_hash":     "abc123",
		"right_hash":    "def456",
		"files_changed": 3,
		"insertions":    20,
		"deletions":     4,
		"summary":       "3 files changed",
	})
	if err != nil {
		t.Fatalf("normalize diff: %v", err)
	}
	if diff.FilesChanged != 3 {
		t.Fatalf("unexpected diff summary: %+v", diff)
	}

	payload := RepoAuditPayload("native cloud", "repo.diff", "success", "def456", "space-1")
	if payload["actor"] != "native cloud" || payload["commit_hash"] != "def456" {
		t.Fatalf("unexpected repo audit payload: %+v", payload)
	}

	timeout := NormalizeGatewayError(assertErr("gateway timeout while fetching lineage"))
	if timeout.Code != "timeout" || !timeout.Retryable {
		t.Fatalf("unexpected timeout normalization: %+v", timeout)
	}
	missing := NormalizeGatewayError(assertErr("commit not found"))
	if missing.Code != "not_found" || missing.Retryable {
		t.Fatalf("unexpected not-found normalization: %+v", missing)
	}
}

func TestPythonParityRepoRegistryDeterminism(t *testing.T) {
	registry := RepoRegistry{}
	registry.RegisterSpace(RepoSpace{
		SpaceID:        "space-1",
		ProjectKey:     "BIGCLAW",
		Repo:           "OpenAGIs/BigClaw",
		SidecarURL:     "http://127.0.0.1:4041",
		HealthState:    "healthy",
		SidecarEnabled: true,
	})

	resolved, ok := registry.ResolveSpace("BIGCLAW")
	if !ok || resolved.Repo != "OpenAGIs/BigClaw" {
		t.Fatalf("unexpected resolved space: %+v %t", resolved, ok)
	}

	channel := registry.ResolveDefaultChannel("BIGCLAW", domain.Task{ID: "OPE-141"})
	if channel != "bigclaw-ope-141" {
		t.Fatalf("unexpected channel: %q", channel)
	}

	agent := registry.ResolveAgent("native cloud", "reviewer")
	if agent.RepoAgentID != "agent-native-cloud" {
		t.Fatalf("unexpected repo agent id: %+v", agent)
	}

	restored := RepoRegistry{
		SpacesByProject: map[string]RepoSpace{"BIGCLAW": resolved},
		AgentsByActor:   map[string]RepoAgent{"native cloud": agent},
	}
	if _, ok := restored.ResolveSpace("BIGCLAW"); !ok {
		t.Fatal("expected restored registry to resolve BIGCLAW")
	}
	if got := restored.ResolveAgent("native cloud", "reviewer").RepoAgentID; got != "agent-native-cloud" {
		t.Fatalf("unexpected restored agent id: %q", got)
	}
}

func assertErr(message string) error {
	return parityErr(message)
}

type parityErr string

func (e parityErr) Error() string {
	return string(e)
}
