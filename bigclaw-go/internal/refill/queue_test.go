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
