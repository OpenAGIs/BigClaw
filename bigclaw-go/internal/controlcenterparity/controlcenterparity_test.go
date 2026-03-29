package controlcenterparity

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestQueuePeekTasksReturnsPriorityOrder(t *testing.T) {
	t.Parallel()

	queue, err := NewQueue(filepath.Join(t.TempDir(), "queue.json"))
	if err != nil {
		t.Fatalf("new queue: %v", err)
	}
	for _, task := range []Task{
		{ID: "p2", Source: "linear", Title: "low", Priority: 2, RiskLevel: RiskLow},
		{ID: "p0", Source: "linear", Title: "top", Priority: 0, RiskLevel: RiskLow},
		{ID: "p1", Source: "linear", Title: "mid", Priority: 1, RiskLevel: RiskLow},
	} {
		if err := queue.Enqueue(task); err != nil {
			t.Fatalf("enqueue %s: %v", task.ID, err)
		}
	}

	peeked := queue.PeekTasks()
	if len(peeked) != 3 || peeked[0].ID != "p0" || peeked[1].ID != "p1" || peeked[2].ID != "p2" {
		t.Fatalf("unexpected queue order: %+v", peeked)
	}
}

func TestQueueControlCenterSummarizesQueueAndExecutionMedia(t *testing.T) {
	t.Parallel()

	queue, err := NewQueue(filepath.Join(t.TempDir(), "queue.json"))
	if err != nil {
		t.Fatalf("new queue: %v", err)
	}
	for _, task := range []Task{
		{ID: "BIG-802-1", Source: "linear", Title: "top", Priority: 0, RiskLevel: RiskHigh},
		{ID: "BIG-802-2", Source: "linear", Title: "mid", Priority: 1, RiskLevel: RiskMedium},
		{ID: "BIG-802-3", Source: "linear", Title: "low", Priority: 2, RiskLevel: RiskLow},
	} {
		if err := queue.Enqueue(task); err != nil {
			t.Fatalf("enqueue %s: %v", task.ID, err)
		}
	}

	center := BuildQueueControlCenter(queue, []map[string]string{
		{"task_id": "BIG-802-1", "status": "needs-approval", "medium": "vm"},
		{"task_id": "BIG-802-2", "status": "approved", "medium": "browser"},
		{"task_id": "BIG-802-4", "status": "approved", "medium": "docker"},
	})
	report := RenderQueueControlCenter(center, nil)

	if center.QueueDepth != 3 {
		t.Fatalf("queue depth = %d, want 3", center.QueueDepth)
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
		t.Fatalf("waiting approval runs = %d, want 1", center.WaitingApprovalRuns)
	}
	if strings.Join(center.BlockedTasks, ",") != "BIG-802-1" {
		t.Fatalf("unexpected blocked tasks: %+v", center.BlockedTasks)
	}
	if strings.Join(center.QueuedTasks, ",") != "BIG-802-1,BIG-802-2,BIG-802-3" {
		t.Fatalf("unexpected queued tasks: %+v", center.QueuedTasks)
	}
	actionIDs := make([]string, 0, len(center.Actions["BIG-802-1"]))
	for _, action := range center.Actions["BIG-802-1"] {
		actionIDs = append(actionIDs, action.ActionID)
	}
	if strings.Join(actionIDs, ",") != "drill-down,export,add-note,escalate,retry,pause,reassign,audit" {
		t.Fatalf("unexpected action ids: %v", actionIDs)
	}
	if !center.Actions["BIG-802-1"][3].Enabled || !center.Actions["BIG-802-1"][4].Enabled || center.Actions["BIG-802-1"][5].Enabled {
		t.Fatalf("unexpected blocked task actions: %+v", center.Actions["BIG-802-1"])
	}
	for _, fragment := range []string{
		"# Queue Control Center",
		"- Waiting Approval Runs: 1",
		"- BIG-802-1",
		"BIG-802-1: Drill Down [drill-down]",
		"Escalate [escalate] state=enabled",
		"Pause [pause] state=disabled target=BIG-802-1 reason=approval-blocked tasks should be escalated instead of paused",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}

func TestQueueControlCenterRendersSharedViewEmptyState(t *testing.T) {
	t.Parallel()

	queue, err := NewQueue(filepath.Join(t.TempDir(), "queue.json"))
	if err != nil {
		t.Fatalf("new queue: %v", err)
	}
	center := BuildQueueControlCenter(queue, nil)
	report := RenderQueueControlCenter(center, &SharedViewContext{
		Filters:      []SharedViewFilter{{Label: "Team", Value: "operations"}},
		ResultCount:  0,
		EmptyMessage: "No queued work for the selected team.",
	})

	for _, fragment := range []string{
		"## View State",
		"- State: empty",
		"- Summary: No queued work for the selected team.",
		"- Team: operations",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}
