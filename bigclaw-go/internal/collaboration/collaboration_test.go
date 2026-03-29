package collaboration

import (
	"testing"

	"bigclaw-go/internal/repo"
)

func TestMergeCollaborationThreadsCombinesNativeAndRepoSurfaces(t *testing.T) {
	native := BuildThread(
		"run",
		"run-165",
		[]CollaborationComment{
			{CommentID: "c1", Author: "ops", Body: "native note", CreatedAt: "2026-03-12T10:00:00Z"},
		},
		[]DecisionNote{
			{DecisionID: "d1", Author: "lead", Outcome: "approved", Summary: "native decision", RecordedAt: "2026-03-12T10:05:00Z"},
		},
	)

	board := repo.RepoDiscussionBoard{}
	repoPost := board.CreatePost(
		"bigclaw-ope-165",
		"repo-agent",
		"repo board context",
		"run",
		"run-165",
		nil,
	)
	repoThread := BuildThread(
		"repo-board",
		"run-165",
		[]CollaborationComment{
			{
				CommentID: repoPost.ToCollaborationComment().CommentID,
				Author:    repoPost.ToCollaborationComment().Author,
				Body:      repoPost.ToCollaborationComment().Body,
				CreatedAt: repoPost.ToCollaborationComment().CreatedAt,
				Anchor:    repoPost.ToCollaborationComment().Anchor,
				Status:    repoPost.ToCollaborationComment().Status,
			},
		},
		nil,
	)

	merged := MergeThreads("run-165", &native, &repoThread)
	if merged == nil {
		t.Fatal("expected merged thread")
	}
	if merged.Surface != "merged" || len(merged.Comments) != 2 || len(merged.Decisions) != 1 {
		t.Fatalf("unexpected merged thread: %+v", merged)
	}
	if merged.Comments[1].Body != "repo board context" {
		t.Fatalf("expected repo board comment to be appended in timestamp order, got %+v", merged.Comments)
	}
}
