package collaboration_test

import (
	"testing"

	"bigclaw-go/internal/collaboration"
	"bigclaw-go/internal/repo"
)

func TestMergeCollaborationThreadsCombinesNativeAndRepoSurfaces(t *testing.T) {
	native := collaboration.BuildThread(
		"run",
		"run-165",
		[]collaboration.Comment{{CommentID: "c1", Author: "ops", Body: "native note", CreatedAt: "2026-03-12T10:00:00Z"}},
		[]collaboration.Decision{{DecisionID: "d1", Author: "lead", Outcome: "approved", Summary: "native decision", RecordedAt: "2026-03-12T10:05:00Z"}},
	)

	board := repo.DiscussionBoard{}
	repoPost := board.CreatePost(
		"bigclaw-ope-165",
		"repo-agent",
		"repo board context",
		"run",
		"run-165",
	)
	repoThread := collaboration.BuildThread(
		"repo-board",
		"run-165",
		[]collaboration.Comment{repoPost.ToCollaborationComment()},
		nil,
	)

	merged := collaboration.MergeThreads("run-165", native, repoThread)

	if merged == nil {
		t.Fatal("expected merged thread")
	}
	if merged.Surface != "merged" || len(merged.Comments) != 2 || len(merged.Decisions) != 1 || merged.Comments[1].Body != "repo board context" {
		t.Fatalf("unexpected merged thread: %+v", merged)
	}
}
