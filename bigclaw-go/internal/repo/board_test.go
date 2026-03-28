package repo

import (
	"testing"
	"time"
)

func TestRepoBoardCreateReplyAndTargetFiltering(t *testing.T) {
	board := RepoDiscussionBoard{
		Now: func() time.Time {
			return time.Date(2026, 3, 28, 2, 3, 4, 0, time.UTC)
		},
	}

	post := board.CreatePost(
		"bigclaw-ope-164",
		"agent-a",
		"Need reviewer on commit lineage",
		"run",
		"run-164",
		map[string]any{"severity": "p1"},
	)
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
		t.Fatalf("expected 2 run posts, got %d", len(runPosts))
	}
	if runPosts[0].Channel != "bigclaw-ope-164" {
		t.Fatalf("unexpected channel: %q", runPosts[0].Channel)
	}

	comment := runPosts[0].ToCollaborationComment()
	if comment.Anchor != "run:run-164" {
		t.Fatalf("unexpected collaboration anchor: %q", comment.Anchor)
	}
	if comment.Body != "Need reviewer on commit lineage" {
		t.Fatalf("unexpected collaboration body: %q", comment.Body)
	}
	if comment.Status != "open" {
		t.Fatalf("unexpected collaboration status: %q", comment.Status)
	}
}

func TestRepoPostToCollaborationCommentResolvedStatus(t *testing.T) {
	post := RepoPost{
		PostID:        "post-3",
		Author:        "reviewer",
		Body:          "Approved",
		CreatedAt:     "2026-03-28T02:03:04Z",
		TargetSurface: "task",
		TargetID:      "BIG-924",
		Metadata:      map[string]any{"resolved": true},
	}

	comment := post.ToCollaborationComment()
	if comment.CommentID != "repo-post-3" {
		t.Fatalf("unexpected comment id: %q", comment.CommentID)
	}
	if comment.Status != "resolved" {
		t.Fatalf("unexpected status: %q", comment.Status)
	}
	if comment.Anchor != "task:BIG-924" {
		t.Fatalf("unexpected anchor: %q", comment.Anchor)
	}
}
