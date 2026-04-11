package collaboration

import "testing"

func TestMergeThreadsCombinesNativeAndRepoSurfaces(t *testing.T) {
	native := BuildThread(
		"run",
		"run-165",
		[]Comment{{CommentID: "c1", Author: "ops", Body: "native note", CreatedAt: "2026-03-12T10:00:00Z"}},
		[]Decision{{DecisionID: "d1", Author: "lead", Outcome: "approved", Summary: "native decision", RecordedAt: "2026-03-12T10:05:00Z"}},
	)

	repoThread := BuildThread(
		"repo-board",
		"run-165",
		[]Comment{{CommentID: "repo-post-1", Author: "repo-agent", Body: "repo board context", CreatedAt: "2026-03-12T10:02:00Z", Anchor: "run:run-165"}},
		nil,
	)

	merged := MergeThreads("run-165", &native, &repoThread)
	if merged == nil {
		t.Fatal("expected merged thread")
	}
	if merged.Surface != "merged" || merged.TargetID != "run-165" {
		t.Fatalf("unexpected merged thread header: %+v", merged)
	}
	if len(merged.Comments) != 2 || len(merged.Decisions) != 1 {
		t.Fatalf("unexpected merged thread shape: %+v", merged)
	}
	if merged.Comments[1].Body != "repo board context" {
		t.Fatalf("expected repo comment to be sorted after native comment, got %+v", merged.Comments)
	}
}
