package refill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIssueStateMapRecordsIdentifiers(t *testing.T) {
	issues := []TrackedIssue{
		{Identifier: "BIG-GOM-301", StateName: "Todo"},
		{Identifier: "BIG-GOM-302", StateName: "Todo"},
	}
	stateMap := IssueStateMap(issues)
	if stateMap["BIG-GOM-301"] != "Todo" || stateMap["BIG-GOM-302"] != "Todo" {
		t.Fatalf("unexpected state map: %+v", stateMap)
	}
}

func TestParallelIssueQueueRunnableCountTreatsFullyDoneQueueAsDrained(t *testing.T) {
	queue := &ParallelIssueQueue{
		payload: QueuePayload{
			IssueOrder: []string{"BIG-PAR-001", "BIG-PAR-002"},
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-001", Status: "Done"},
				{Identifier: "BIG-PAR-002", Status: "Done"},
			},
		},
	}
	if got := queue.RunnableCount(); got != 0 {
		t.Fatalf("expected drained runnable count, got %d", got)
	}
}

func TestParallelIssueQueueRunnableCountDoesNotDrainWhenMetadataMissing(t *testing.T) {
	queue := &ParallelIssueQueue{
		payload: QueuePayload{
			IssueOrder: []string{"BIG-PAR-001", "BIG-PAR-002"},
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-001", Status: "Done"},
			},
		},
	}
	if got := queue.RunnableCount(); got == 0 {
		t.Fatalf("expected runnable count for missing metadata, got %d", got)
	}
}

func TestParallelIssueQueueRunnableCountForStatesPrefersLiveStateMap(t *testing.T) {
	queue := &ParallelIssueQueue{
		payload: QueuePayload{
			IssueOrder: []string{"BIG-PAR-224", "BIG-PAR-225", "BIG-PAR-226", "BIG-PAR-227"},
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-224", Status: "In Progress"},
				{Identifier: "BIG-PAR-225", Status: "In Progress"},
				{Identifier: "BIG-PAR-226", Status: "Todo"},
				{Identifier: "BIG-PAR-227", Status: "Todo"},
			},
		},
	}
	liveStates := map[string]string{
		"BIG-PAR-224": "Done",
		"BIG-PAR-225": "Done",
		"BIG-PAR-226": "Done",
		"BIG-PAR-227": "Done",
	}
	if got := queue.RunnableCountForStates(liveStates); got != 0 {
		t.Fatalf("expected live state map to mark queue drained, got %d", got)
	}
}

func TestParallelIssueQueueRefreshRecentBatchesFromStates(t *testing.T) {
	queue := &ParallelIssueQueue{
		payload: QueuePayload{
			IssueOrder: []string{"BIG-PAR-241", "BIG-PAR-242", "BIG-PAR-243"},
		},
	}
	queue.payload.Policy.RefillStates = []string{"Todo", "Backlog"}
	states := map[string]string{
		"BIG-PAR-241": "In Progress",
		"BIG-PAR-242": "Done",
		"BIG-PAR-243": "Backlog",
	}
	updated := queue.RefreshRecentBatchesFromStates(states)
	if !updated {
		t.Fatalf("expected recent batches to report a change")
	}
	if !equalStringSlices(queue.payload.RecentBatches.Active, []string{"BIG-PAR-241"}) {
		t.Fatalf("unexpected active batches: %v", queue.payload.RecentBatches.Active)
	}
	if !equalStringSlices(queue.payload.RecentBatches.Completed, []string{"BIG-PAR-242"}) {
		t.Fatalf("unexpected completed batches: %v", queue.payload.RecentBatches.Completed)
	}
	if !equalStringSlices(queue.payload.RecentBatches.Standby, []string{"BIG-PAR-243"}) {
		t.Fatalf("unexpected standby batches: %v", queue.payload.RecentBatches.Standby)
	}
	if queue.RefreshRecentBatchesFromStates(states) {
		t.Fatalf("expected no change when data is already fresh")
	}
}

func TestParallelIssueQueueSavePreservesBlockedReasonAndRecentBatches(t *testing.T) {
	queuePath := filepath.Join(t.TempDir(), "queue.json")
	queue := &ParallelIssueQueue{
		queuePath: queuePath,
		payload: QueuePayload{
			IssueOrder: []string{"BIG-PAR-230"},
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-230", Status: "Todo"},
			},
		},
	}
	queue.payload.Policy.BlockedReason = "local tracker owns live issue state"
	queue.payload.RecentBatches.Completed = []string{"BIG-PAR-229"}
	queue.payload.RecentBatches.Active = []string{"BIG-PAR-230"}
	queue.payload.RecentBatches.Standby = []string{}

	if err := queue.Save(); err != nil {
		t.Fatalf("save queue: %v", err)
	}

	body, err := os.ReadFile(queuePath)
	if err != nil {
		t.Fatalf("read queue: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, `"blocked_reason": "local tracker owns live issue state"`) {
		t.Fatalf("expected blocked_reason to persist, got %s", text)
	}
	if !strings.Contains(text, `"recent_batches"`) {
		t.Fatalf("expected recent_batches to persist, got %s", text)
	}
	if !strings.Contains(text, `"completed": [`) || !strings.Contains(text, `"active": [`) || !strings.Contains(text, `"standby": []`) {
		t.Fatalf("expected recent_batches fields to persist, got %s", text)
	}
}
