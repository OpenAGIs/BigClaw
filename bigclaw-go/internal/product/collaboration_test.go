package product

import (
	"testing"

	"bigclaw-go/internal/repo"
)

func TestMergeCollaborationThreadsCombinesNativeAndRepoSurfaces(t *testing.T) {
	native := BuildCollaborationThread("run", "run-165", []CollaborationComment{
		{CommentID: "c1", Author: "ops", Body: "native note", CreatedAt: "2026-03-12T10:00:00Z"},
	}, []DecisionNote{
		{DecisionID: "d1", Author: "lead", Outcome: "approved", Summary: "native decision", RecordedAt: "2026-03-12T10:05:00Z"},
	})

	board := repo.RepoDiscussionBoard{}
	post := board.CreatePost("bigclaw-ope-165", "repo-agent", "repo board context", "run", "run-165", nil)
	repoThread := BuildCollaborationThread("repo-board", "run-165", []CollaborationComment{
		{
			CommentID: post.PostID,
			Author:    post.Author,
			Body:      post.Body,
			CreatedAt: post.CreatedAt,
			Anchor:    "run:" + post.TargetID,
		},
	}, nil)

	merged := MergeCollaborationThreads("run-165", &native, &repoThread)
	if merged == nil {
		t.Fatal("expected merged thread")
	}
	if merged.Surface != "merged" || len(merged.Comments) != 2 || len(merged.Decisions) != 1 {
		t.Fatalf("unexpected merged thread: %+v", merged)
	}
	if merged.Comments[1].Body != "repo board context" {
		t.Fatalf("expected repo board comment in merged output, got %+v", merged.Comments)
	}
}
