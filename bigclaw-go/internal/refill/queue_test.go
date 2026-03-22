package refill

import "testing"

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
