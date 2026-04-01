package collaboration

import "testing"

func TestMergeCollaborationThreadsCombinesNativeAndRepoSurfaces(t *testing.T) {
	native := BuildCollaborationThread(
		"run",
		"run-165",
		[]Comment{{CommentID: "c1", Author: "ops", Body: "native note", CreatedAt: "2026-03-12T10:00:00Z"}},
		[]Decision{{DecisionID: "d1", Author: "lead", Outcome: "approved", Summary: "native decision", RecordedAt: "2026-03-12T10:05:00Z"}},
	)

	board := &RepoDiscussionBoard{}
	post := board.CreatePost("bigclaw-ope-165", "repo-agent", "repo board context", "run", "run-165")
	repoThread := BuildCollaborationThread("repo-board", "run-165", []Comment{post.ToComment()}, nil)

	merged := MergeCollaborationThreads("run-165", &native, &repoThread)
	if merged == nil {
		t.Fatal("expected merged thread")
	}
	if merged.Surface != "merged" {
		t.Fatalf("expected merged surface, got %+v", merged)
	}
	if len(merged.Comments) != 2 || len(merged.Decisions) != 1 {
		t.Fatalf("unexpected merged thread: %+v", merged)
	}
	if merged.Comments[1].Body != "repo board context" {
		t.Fatalf("expected repo board comment to survive merge, got %+v", merged.Comments)
	}
}
