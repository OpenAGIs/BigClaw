package collaboration

import "testing"

func TestBuildCollaborationThreadCountsMentionsAcrossCommentsAndDecisions(t *testing.T) {
	thread := BuildCollaborationThread(
		"dashboard",
		"ops-overview",
		[]Comment{{CommentID: "c1", Author: "pm", Body: "Please review with @ops and @eng."}},
		[]Decision{{DecisionID: "d1", Author: "ops", Outcome: "approved", Summary: "Keep @pm posted.", RecordedAt: "Follow up with @design."}},
	)

	if thread.Surface != "dashboard" || thread.TargetID != "ops-overview" {
		t.Fatalf("unexpected thread identity: %+v", thread)
	}
	if thread.MentionCount != 4 {
		t.Fatalf("expected four mentions, got %+v", thread)
	}
}

func TestBuildCollaborationThreadFromAuditsBuildsCommentAndDecisionRecords(t *testing.T) {
	thread := BuildCollaborationThreadFromAudits([]map[string]any{
		{
			"action": "collaboration.comment",
			"details": map[string]any{
				"comment_id": "flow-comment-1",
				"author":     "ops-lead",
				"body":       "Route @eng after review.",
				"anchor":     "handoff-lane",
			},
		},
		{
			"action": "collaboration.decision",
			"details": map[string]any{
				"decision_id": "flow-decision-1",
				"author":      "eng-manager",
				"outcome":     "accepted",
				"summary":     "Engineering owns the next handoff.",
				"follow_up":   "Notify @ops after deploy.",
			},
		},
	}, "flow", "run-113")
	if thread == nil {
		t.Fatal("expected thread from audits")
	}
	if thread.Surface != "flow" || thread.TargetID != "run-113" {
		t.Fatalf("unexpected thread identity: %+v", thread)
	}
	if len(thread.Comments) != 1 || len(thread.Decisions) != 1 {
		t.Fatalf("unexpected thread contents: %+v", thread)
	}
	if thread.Comments[0].CommentID != "flow-comment-1" || thread.Comments[0].CreatedAt != "handoff-lane" {
		t.Fatalf("unexpected comment mapping: %+v", thread.Comments[0])
	}
	if thread.Decisions[0].DecisionID != "flow-decision-1" || thread.Decisions[0].RecordedAt != "Notify @ops after deploy." {
		t.Fatalf("unexpected decision mapping: %+v", thread.Decisions[0])
	}
	if thread.MentionCount != 2 {
		t.Fatalf("expected two mentions, got %+v", thread)
	}
}

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
