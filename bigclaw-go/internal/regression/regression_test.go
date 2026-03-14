package regression

import (
	"testing"
	"time"

	"bigclaw-go/internal/domain"
)

func TestBuildRegressionCenter(t *testing.T) {
	base := time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC)
	records := []Record{
		{
			Task:   domain.Task{ID: "task-reg-1", TraceID: "trace-reg-1", Title: "Deploy regression", State: domain.TaskDeadLetter, Labels: []string{"regression", "prod"}, RequiredTools: []string{"deploy"}, Metadata: map[string]string{"team": "platform", "workflow": "deploy", "template": "release", "service": "api", "regression_count": "2", "regression_source": "security scan failed"}, UpdatedAt: base},
			Events: []domain.Event{{Type: domain.EventTaskRetried, Timestamp: base.Add(time.Minute), Payload: map[string]any{"reason": "retry deploy"}}},
		},
		{
			Task:   domain.Task{ID: "task-reg-2", TraceID: "trace-reg-2", Title: "Prompt regression", State: domain.TaskBlocked, Labels: []string{"regression"}, Metadata: map[string]string{"team": "growth", "workflow": "prompt-tune", "template": "triage-system", "service": "assistant", "regression_count": "1", "regression_source": "prompt drift"}, UpdatedAt: base.Add(24 * time.Hour)},
			Events: nil,
		},
	}
	center := Build(records)
	if center.Summary.TotalRegressions != 3 || center.Summary.AffectedTasks != 2 || center.Summary.CriticalRegressions != 1 || center.Summary.ReworkEvents != 1 {
		t.Fatalf("unexpected regression summary: %+v", center.Summary)
	}
	if center.Summary.TopSource != "security scan failed" || center.Summary.TopWorkflow != "deploy" {
		t.Fatalf("unexpected top breakdown pointers: %+v", center.Summary)
	}
	if len(center.WorkflowBreakdown) == 0 || center.WorkflowBreakdown[0].Key != "deploy" || center.WorkflowBreakdown[0].TotalRegressions != 2 {
		t.Fatalf("unexpected workflow breakdown: %+v", center.WorkflowBreakdown)
	}
	if len(center.Findings) != 2 || center.Findings[0].TaskID != "task-reg-1" || center.Findings[0].Severity != "critical" {
		t.Fatalf("unexpected regression findings: %+v", center.Findings)
	}
	if len(center.Hotspots) == 0 {
		t.Fatalf("expected hotspots, got %+v", center.Hotspots)
	}
	trend := Trend(center.Findings, base, base.Add(24*time.Hour), "day")
	if len(trend) != 2 || trend[0].TotalRegressions != 2 || trend[1].TotalRegressions != 1 {
		t.Fatalf("unexpected regression trend: %+v", trend)
	}
}
