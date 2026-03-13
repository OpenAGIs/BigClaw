package triage

import (
	"testing"

	"bigclaw-go/internal/domain"
)

func TestBuildAutoTriageCenter(t *testing.T) {
	records := []Record{
		{
			Task:   domain.Task{ID: "task-browser", TraceID: "run-browser", Title: "Browser replay failure", State: domain.TaskDeadLetter, RequiredTools: []string{"browser"}, Metadata: map[string]string{"team": "platform", "project": "alpha"}},
			Events: []domain.Event{{Type: domain.EventTaskDeadLetter, Payload: map[string]any{"message": "browser session crashed"}}},
		},
		{
			Task:   domain.Task{ID: "task-risk", TraceID: "run-risk", Title: "Security approval", State: domain.TaskBlocked, Priority: 1, Labels: []string{"security", "prod"}, RequiredTools: []string{"deploy"}, Metadata: map[string]string{"team": "platform", "project": "alpha"}},
			Events: []domain.Event{{Type: domain.EventRunTakeover, Payload: map[string]any{"reason": "requires approval for high-risk task"}}},
		},
		{
			Task:   domain.Task{ID: "task-browser-similar", TraceID: "run-browser-2", Title: "Browser replay failure", State: domain.TaskSucceeded, RequiredTools: []string{"browser"}, Metadata: map[string]string{"team": "platform", "project": "alpha"}},
			Events: []domain.Event{{Type: domain.EventTaskCompleted, Payload: map[string]any{"message": "browser session crashed"}}},
		},
	}
	center := Build(records)
	if center.FlaggedRuns != 2 || center.InboxSize != 2 {
		t.Fatalf("expected 2 flagged triage items, got %+v", center)
	}
	if center.SeverityCounts["critical"] != 1 || center.SeverityCounts["high"] != 1 {
		t.Fatalf("unexpected severity counts: %+v", center.SeverityCounts)
	}
	if center.OwnerCounts["engineering"] != 1 || center.OwnerCounts["security"] != 1 {
		t.Fatalf("unexpected owner counts: %+v", center.OwnerCounts)
	}
	if center.Recommendation != "immediate-attention" {
		t.Fatalf("expected immediate-attention recommendation, got %+v", center)
	}
	if center.Findings[0].TaskID != "task-browser" || center.Findings[0].SuggestedWorkflow != "run-replay" {
		t.Fatalf("expected browser failure first, got %+v", center.Findings)
	}
	if len(center.Findings[0].SimilarCases) == 0 || center.Findings[0].SimilarCases[0].TaskID != "task-browser-similar" {
		t.Fatalf("expected similarity evidence, got %+v", center.Findings[0])
	}
	if center.Findings[1].SuggestedWorkflow != "security-review" || center.Findings[1].SuggestedPriority != "P1" {
		t.Fatalf("expected security approval workflow, got %+v", center.Findings[1])
	}
	if len(center.Clusters) != 2 {
		t.Fatalf("expected two triage clusters, got %+v", center.Clusters)
	}
}
