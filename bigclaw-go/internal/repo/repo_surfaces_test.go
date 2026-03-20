package repo

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
)

func TestBindRunCommitsAndAcceptedHash(t *testing.T) {
	binding, err := BindRunCommits([]RunCommitLink{
		{RunID: "run-1", CommitHash: "abc123", Role: "source", RepoSpaceID: "space-1"},
		{RunID: "run-1", CommitHash: "def456", Role: "accepted", RepoSpaceID: "space-1"},
	})
	if err != nil {
		t.Fatalf("bind run commits: %v", err)
	}
	if got := binding.AcceptedCommitHash(); got != "def456" {
		t.Fatalf("unexpected accepted hash: %q", got)
	}
}

func TestBindRunCommitsRejectsUnsupportedRoles(t *testing.T) {
	_, err := BindRunCommits([]RunCommitLink{{RunID: "run-1", CommitHash: "abc123", Role: "merge", RepoSpaceID: "space-1"}})
	if err == nil || !strings.Contains(err.Error(), "unsupported run commit roles: merge") {
		t.Fatalf("expected unsupported role error, got %v", err)
	}
}

func TestRepoRegistryResolvesSpaceChannelAndAgent(t *testing.T) {
	registry := RepoRegistry{}
	registry.RegisterSpace(RepoSpace{SpaceID: "space-1", ProjectKey: "ALPHA", Repo: "OpenAGIs/BigClaw", SidecarEnabled: true})

	space, ok := registry.ResolveSpace("ALPHA")
	if !ok || space.SpaceID != "space-1" {
		t.Fatalf("expected resolved space, got %+v %t", space, ok)
	}
	channel := registry.ResolveDefaultChannel("ALPHA", domain.Task{ID: "BIG-401/review closeout"})
	if channel != "alpha-big-401-review-closeout" {
		t.Fatalf("unexpected channel: %q", channel)
	}
	agent := registry.ResolveAgent("alice@example.com", "executor")
	if agent.RepoAgentID != "agent-alice-example-com" || !reflect.DeepEqual(agent.Roles, []string{"executor"}) {
		t.Fatalf("unexpected agent: %+v", agent)
	}
}

func TestRepoDiscussionBoardCreateReplyAndFilter(t *testing.T) {
	board := RepoDiscussionBoard{Now: func() time.Time { return time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC) }}
	post := board.CreatePost("alpha-release", "alice", "Need reviewer eyes", "task", "BIG-401", map[string]any{"resolved": false})
	reply, err := board.Reply(post.PostID, "bob", "I am on it")
	if err != nil {
		t.Fatalf("reply: %v", err)
	}
	if reply.Channel != "alpha-release" || reply.TargetID != "BIG-401" || reply.ParentPostID != post.PostID {
		t.Fatalf("unexpected reply: %+v", reply)
	}
	filtered := board.ListPosts("alpha-release", "task", "BIG-401")
	if len(filtered) != 2 {
		t.Fatalf("expected filtered posts, got %+v", filtered)
	}
}

func TestNormalizeGatewayPayloadsAndErrors(t *testing.T) {
	commit, err := NormalizeCommit(map[string]any{"commit_hash": "abc123", "title": "Ship cutover", "author": "alice"})
	if err != nil {
		t.Fatalf("normalize commit: %v", err)
	}
	if commit.CommitHash != "abc123" || commit.Title != "Ship cutover" {
		t.Fatalf("unexpected commit: %+v", commit)
	}

	lineage, err := NormalizeLineage(map[string]any{
		"root_hash": "abc123",
		"lineage":   []map[string]any{{"commit_hash": "abc123", "title": "root"}, {"commit_hash": "def456", "title": "child"}},
		"children":  map[string][]string{"abc123": {"def456"}},
		"leaves":    []string{"def456"},
	})
	if err != nil {
		t.Fatalf("normalize lineage: %v", err)
	}
	if lineage.RootHash != "abc123" || len(lineage.Lineage) != 2 || len(lineage.Children["abc123"]) != 1 {
		t.Fatalf("unexpected lineage: %+v", lineage)
	}

	diff, err := NormalizeDiff(map[string]any{"left_hash": "abc123", "right_hash": "def456", "files_changed": 3, "insertions": 12, "deletions": 4})
	if err != nil {
		t.Fatalf("normalize diff: %v", err)
	}
	if diff.FilesChanged != 3 || diff.Insertions != 12 || diff.Deletions != 4 {
		t.Fatalf("unexpected diff: %+v", diff)
	}

	timeout := NormalizeGatewayError(errors.New("request timeout during fetch"))
	notFound := NormalizeGatewayError(errors.New("bundle not found"))
	other := NormalizeGatewayError(errors.New("permission denied"))
	if timeout.Code != "timeout" || !timeout.Retryable || notFound.Code != "not_found" || other.Code != "gateway_error" {
		t.Fatalf("unexpected normalized errors: timeout=%+v notfound=%+v other=%+v", timeout, notFound, other)
	}
}

func TestRepoAuditPayloadIsDeterministic(t *testing.T) {
	payload := RepoAuditPayload("alice", "repo.diff", "ok", "abc123", "space-1")
	if !reflect.DeepEqual(payload, map[string]any{
		"actor":         "alice",
		"action":        "repo.diff",
		"outcome":       "ok",
		"commit_hash":   "abc123",
		"repo_space_id": "space-1",
	}) {
		t.Fatalf("unexpected audit payload: %+v", payload)
	}
}
