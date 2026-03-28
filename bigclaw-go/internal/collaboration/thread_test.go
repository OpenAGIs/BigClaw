package collaboration

import "testing"

func TestMergeCollaborationThreadsCombinesNativeAndRepoSurfaces(t *testing.T) {
	native := BuildCollaborationThread(
		"run",
		"run-165",
		[]CollaborationComment{
			{CommentID: "c1", Author: "ops", Body: "native note", CreatedAt: "2026-03-12T10:00:00Z"},
		},
		[]DecisionNote{
			{DecisionID: "d1", Author: "lead", Outcome: "approved", Summary: "native decision", RecordedAt: "2026-03-12T10:05:00Z"},
		},
	)

	repoThread := BuildCollaborationThread(
		"repo-board",
		"run-165",
		[]CollaborationComment{
			{CommentID: "repo-post-1", Author: "repo-agent", Body: "repo board context", CreatedAt: "2026-03-12T10:01:00Z"},
		},
		nil,
	)

	merged := MergeCollaborationThreads("run-165", &native, &repoThread)
	if merged == nil {
		t.Fatal("expected merged thread")
	}
	if merged.Surface != "merged" {
		t.Fatalf("unexpected surface: %q", merged.Surface)
	}
	if len(merged.Comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(merged.Comments))
	}
	if len(merged.Decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(merged.Decisions))
	}
	if merged.Comments[1].Body != "repo board context" {
		t.Fatalf("unexpected merged repo comment: %q", merged.Comments[1].Body)
	}
}

func TestMergeCollaborationThreadsReturnsNilWhenEmpty(t *testing.T) {
	if merged := MergeCollaborationThreads("run-165", nil, nil); merged != nil {
		t.Fatalf("expected nil merge, got %+v", merged)
	}
}
