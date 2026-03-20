package repo

import (
	"strings"
	"testing"
)

func TestDiscussionBoardCreateReplyAndTargetFiltering(t *testing.T) {
	var board DiscussionBoard
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
		t.Fatalf("expected reply to succeed: %v", err)
	}

	if post.PostID != "post-1" {
		t.Fatalf("unexpected post id: %+v", post)
	}
	if reply.ParentPostID != "post-1" {
		t.Fatalf("unexpected reply parent id: %+v", reply)
	}

	runPosts := board.ListPosts("", "run", "run-164")
	if len(runPosts) != 2 {
		t.Fatalf("expected two run posts, got %d", len(runPosts))
	}
	if runPosts[0].Channel != "bigclaw-ope-164" {
		t.Fatalf("unexpected filtered post channel: %+v", runPosts[0])
	}

	comment := runPosts[0].ToCollaborationComment()
	if comment.Anchor != "run:run-164" {
		t.Fatalf("unexpected collaboration anchor: %+v", comment)
	}
	if !strings.HasPrefix(comment.Body, "Need reviewer") {
		t.Fatalf("unexpected collaboration body: %+v", comment)
	}
}

func TestNormalizeRepoPostDefaultsAndResolvedComment(t *testing.T) {
	post := NormalizeRepoPost(map[string]any{
		"post_id":   "post-9",
		"channel":   "bigclaw-ope-200",
		"author":    "reviewer",
		"body":      "resolved on repo board",
		"target_id": "run-200",
		"metadata":  map[string]any{"resolved": true},
	})
	if post.TargetSurface != "task" {
		t.Fatalf("expected default target surface, got %+v", post)
	}
	if post.CreatedAt == "" {
		t.Fatalf("expected created timestamp to default, got %+v", post)
	}
	comment := post.ToCollaborationComment()
	if comment.Status != "resolved" {
		t.Fatalf("expected resolved comment status, got %+v", comment)
	}
}
