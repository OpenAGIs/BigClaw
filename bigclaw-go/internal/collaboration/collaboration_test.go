package collaboration_test

import (
	"testing"
	"time"

	"bigclaw-go/internal/collaboration"
	"bigclaw-go/internal/repo"
)

func TestMergeThreadsCombinesNativeAndRepoSurfaces(t *testing.T) {
	native := collaboration.BuildThread("run", "run-165", []collaboration.Comment{{
		CommentID: "c1",
		Author:    "ops",
		Body:      "native note",
		CreatedAt: "2026-03-12T10:00:00Z",
	}}, []collaboration.Decision{{
		DecisionID: "d1",
		Author:     "lead",
		Outcome:    "approved",
		Summary:    "native decision",
		RecordedAt: "2026-03-12T10:05:00Z",
	}})

	board := repo.RepoDiscussionBoard{
		Now: func() time.Time { return time.Date(2026, 3, 12, 10, 1, 0, 0, time.UTC) },
	}
	repoPost := board.CreatePost("bigclaw-ope-165", "repo-agent", "repo board context", "run", "run-165", nil)
	repoThread := collaboration.BuildThread("repo-board", "run-165", []collaboration.Comment{
		repoPost.ToCollaborationComment(),
	}, nil)

	merged := collaboration.MergeThreads("run-165", &native, &repoThread)
	if merged == nil {
		t.Fatal("expected merged thread")
	}
	if merged.Surface != "merged" || merged.TargetID != "run-165" {
		t.Fatalf("unexpected merged header: %+v", merged)
	}
	if len(merged.Comments) != 2 || len(merged.Decisions) != 1 {
		t.Fatalf("unexpected merged counts: %+v", merged)
	}
	if merged.Comments[1].Body != "repo board context" {
		t.Fatalf("unexpected repo comment ordering: %+v", merged.Comments)
	}
	if merged.Comments[1].Anchor != "run:run-165" {
		t.Fatalf("unexpected repo comment anchor: %+v", merged.Comments[1])
	}
}
