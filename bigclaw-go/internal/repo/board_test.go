package repo

import (
	"reflect"
	"testing"
	"time"
)

func TestRepoPostRoundTripAndCollaborationComment(t *testing.T) {
	post := RepoPostFromMap(map[string]any{
		"post_id":        "post-1",
		"channel":        "bigclaw-ope-164",
		"author":         "agent-a",
		"body":           "Need reviewer on commit lineage",
		"target_id":      "run-164",
		"created_at":     "2026-03-27T10:00:00Z",
		"metadata":       map[string]any{"resolved": true},
		"parent_post_id": "",
	})

	if post.TargetSurface != "task" {
		t.Fatalf("expected default target surface task, got %q", post.TargetSurface)
	}

	if got := post.ToMap(); !reflect.DeepEqual(got, map[string]any{
		"post_id":        "post-1",
		"channel":        "bigclaw-ope-164",
		"author":         "agent-a",
		"body":           "Need reviewer on commit lineage",
		"target_surface": "task",
		"target_id":      "run-164",
		"parent_post_id": "",
		"created_at":     "2026-03-27T10:00:00Z",
		"metadata":       map[string]any{"resolved": true},
	}) {
		t.Fatalf("unexpected repo post map: %+v", got)
	}

	comment := post.ToCollaborationComment()
	if comment.CommentID != "repo-post-1" || comment.Anchor != "task:run-164" || comment.Status != "resolved" {
		t.Fatalf("unexpected collaboration comment: %+v", comment)
	}
}

func TestRepoDiscussionBoardMirrorsPythonCreateReplyAndFiltering(t *testing.T) {
	board := RepoDiscussionBoard{Now: func() time.Time { return time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC) }}
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

	if post.PostID != "post-1" || reply.ParentPostID != "post-1" {
		t.Fatalf("expected sequential post ids, got post=%+v reply=%+v", post, reply)
	}

	runPosts := board.ListPosts("", "run", "run-164")
	if len(runPosts) != 2 || runPosts[0].Channel != "bigclaw-ope-164" {
		t.Fatalf("unexpected filtered repo posts: %+v", runPosts)
	}

	comment := runPosts[0].ToCollaborationComment()
	if comment.Anchor != "run:run-164" || comment.Body != "Need reviewer on commit lineage" {
		t.Fatalf("unexpected collaboration comment conversion: %+v", comment)
	}
}

func TestRepoDiscussionBoardReplyErrorAndMapDefaults(t *testing.T) {
	board := RepoDiscussionBoard{}
	if _, err := board.Reply("missing-post", "reviewer", "hello"); err == nil || err.Error() != "unknown parent post: missing-post" {
		t.Fatalf("expected missing parent error, got %v", err)
	}

	post := RepoPostFromMap(map[string]any{
		"post_id": "post-2",
	})
	if post.CreatedAt == "" {
		t.Fatal("expected created_at fallback")
	}
	if _, err := time.Parse(time.RFC3339, post.CreatedAt); err != nil {
		t.Fatalf("expected RFC3339 created_at fallback, got %q (%v)", post.CreatedAt, err)
	}
	if post.TargetSurface != "task" {
		t.Fatalf("expected task fallback target surface, got %q", post.TargetSurface)
	}
}
