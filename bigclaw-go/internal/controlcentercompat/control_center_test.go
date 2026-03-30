package controlcentercompat

import (
	"strings"
	"testing"

	"bigclaw-go/internal/domain"
)

func TestQueuePeekTasksReturnsPriorityOrder(t *testing.T) {
	var queue Queue
	queue.Enqueue(domain.Task{ID: "p2", Source: "linear", Title: "low", Priority: 2})
	queue.Enqueue(domain.Task{ID: "p0", Source: "linear", Title: "top", Priority: 0})
	queue.Enqueue(domain.Task{ID: "p1", Source: "linear", Title: "mid", Priority: 1})

	got := queue.PeekTasks()
	if len(got) != 3 || got[0].ID != "p0" || got[1].ID != "p1" || got[2].ID != "p2" {
		t.Fatalf("unexpected queue order: %+v", got)
	}
}

func TestQueueControlCenterSummarizesQueueAndExecutionMedia(t *testing.T) {
	var queue Queue
	queue.Enqueue(domain.Task{ID: "BIG-802-1", Source: "linear", Title: "top", Priority: 0, RiskLevel: domain.RiskHigh})
	queue.Enqueue(domain.Task{ID: "BIG-802-2", Source: "linear", Title: "mid", Priority: 1, RiskLevel: domain.RiskMedium})
	queue.Enqueue(domain.Task{ID: "BIG-802-3", Source: "linear", Title: "low", Priority: 2, RiskLevel: domain.RiskLow})

	center := Analytics{}.BuildQueueControlCenter(&queue, []RunSnapshot{
		{TaskID: "BIG-802-1", Status: "needs-approval", Medium: "vm"},
		{TaskID: "BIG-802-2", Status: "approved", Medium: "browser"},
		{TaskID: "BIG-802-4", Status: "approved", Medium: "docker"},
	})

	report := RenderQueueControlCenter(center, nil)

	if center.QueueDepth != 3 {
		t.Fatalf("unexpected queue depth: %+v", center)
	}
	if center.QueuedByPriority["P0"] != 1 || center.QueuedByPriority["P1"] != 1 || center.QueuedByPriority["P2"] != 1 {
		t.Fatalf("unexpected queued by priority: %+v", center.QueuedByPriority)
	}
	if center.QueuedByRisk["low"] != 1 || center.QueuedByRisk["medium"] != 1 || center.QueuedByRisk["high"] != 1 {
		t.Fatalf("unexpected queued by risk: %+v", center.QueuedByRisk)
	}
	if center.ExecutionMedia["vm"] != 1 || center.ExecutionMedia["browser"] != 1 || center.ExecutionMedia["docker"] != 1 {
		t.Fatalf("unexpected execution media: %+v", center.ExecutionMedia)
	}
	if center.WaitingApprovalRuns != 1 {
		t.Fatalf("unexpected waiting approval runs: %+v", center)
	}
	if len(center.BlockedTasks) != 1 || center.BlockedTasks[0] != "BIG-802-1" {
		t.Fatalf("unexpected blocked tasks: %+v", center.BlockedTasks)
	}
	if len(center.QueuedTasks) != 3 || center.QueuedTasks[0] != "BIG-802-1" || center.QueuedTasks[1] != "BIG-802-2" || center.QueuedTasks[2] != "BIG-802-3" {
		t.Fatalf("unexpected queued tasks: %+v", center.QueuedTasks)
	}
	actions := center.Actions["BIG-802-1"]
	wantActionIDs := []string{"drill-down", "export", "add-note", "escalate", "retry", "pause", "reassign", "audit"}
	if len(actions) != len(wantActionIDs) {
		t.Fatalf("unexpected actions: %+v", actions)
	}
	for i, want := range wantActionIDs {
		if actions[i].ActionID != want {
			t.Fatalf("unexpected action ids: %+v", actions)
		}
	}
	if !actions[3].Enabled || !actions[4].Enabled || actions[5].Enabled {
		t.Fatalf("unexpected action enablement: %+v", actions)
	}
	for _, want := range []string{
		"# Queue Control Center",
		"- Waiting Approval Runs: 1",
		"- BIG-802-1",
		"BIG-802-1: Drill Down [drill-down]",
		"Escalate [escalate] state=enabled",
		"Pause [pause] state=disabled target=BIG-802-1 reason=approval-blocked tasks should be escalated instead of paused",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in report, got %s", want, report)
		}
	}
}

func TestQueueControlCenterRendersSharedViewEmptyState(t *testing.T) {
	var queue Queue
	center := Analytics{}.BuildQueueControlCenter(&queue, nil)
	resultCount := 0
	report := RenderQueueControlCenter(center, &SharedViewContext{
		Filters:      []SharedViewFilter{{Label: "Team", Value: "operations"}},
		ResultCount:  &resultCount,
		EmptyMessage: "No queued work for the selected team.",
	})

	for _, want := range []string{
		"## View State",
		"- State: empty",
		"- Summary: No queued work for the selected team.",
		"- Team: operations",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in report, got %s", want, report)
		}
	}
}
