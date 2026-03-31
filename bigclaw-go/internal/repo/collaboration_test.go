package repo

import "testing"

func TestRepoPostToCollaborationCommentPreservesAnchorAndResolvedState(t *testing.T) {
	post := RepoPost{
		PostID:        "post-1",
		Author:        "agent-a",
		Body:          "Need reviewer on commit lineage",
		TargetSurface: "run",
		TargetID:      "run-164",
		CreatedAt:     "2026-03-20T10:00:00Z",
		Metadata:      map[string]any{"resolved": true},
	}

	comment := post.ToCollaborationComment()
	if comment.CommentID != "repo-post-1" || comment.Anchor != "run:run-164" || comment.Status != "resolved" {
		t.Fatalf("unexpected collaboration comment: %+v", comment)
	}
	if comment.Body != post.Body || comment.Author != post.Author || comment.CreatedAt != post.CreatedAt {
		t.Fatalf("expected comment fields to mirror post, got %+v", comment)
	}
}

func TestMergeCollaborationThreadsCombinesNativeAndRepoSurfaces(t *testing.T) {
	native := BuildCollaborationThread(
		"run",
		"run-165",
		[]CollaborationComment{{CommentID: "c1", Author: "ops", Body: "native note", CreatedAt: "2026-03-12T10:00:00Z"}},
		[]DecisionNote{{DecisionID: "d1", Author: "lead", Outcome: "approved", Summary: "native decision", RecordedAt: "2026-03-12T10:05:00Z"}},
	)

	board := RepoDiscussionBoard{}
	repoPost := board.CreatePost("bigclaw-ope-165", "repo-agent", "repo board context", "run", "run-165", nil)
	repoThread := BuildCollaborationThread(
		"repo-board",
		"run-165",
		[]CollaborationComment{repoPost.ToCollaborationComment()},
		nil,
	)

	merged := MergeCollaborationThreads("run-165", &native, &repoThread)
	if merged == nil {
		t.Fatal("expected merged collaboration thread")
	}
	if merged.Surface != "merged" || len(merged.Comments) != 2 || len(merged.Decisions) != 1 {
		t.Fatalf("unexpected merged collaboration thread: %+v", merged)
	}
	if merged.Comments[1].Body != "repo board context" {
		t.Fatalf("expected repo board comment to sort after native comment, got %+v", merged.Comments)
	}
}
