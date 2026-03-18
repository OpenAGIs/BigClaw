package triage

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestTriageRecordRoundTripPreservesQueueLabelsAndActions(t *testing.T) {
	record := TriageRecord{
		TriageID: "triage-1",
		TaskID:   "OPE-130",
		Status:   StatusEscalated,
		Queue:    "risk-review",
		Owner:    "ops",
		Summary:  "High-risk flow needs billing review",
		Labels: []Label{
			{Name: "risk", Confidence: 0.9, Source: "heuristic"},
			{Name: "billing", Confidence: 0.8, Source: "classifier"},
		},
		RelatedRunID:     "run-1",
		EscalationTarget: "finance",
		Actions:          []string{"route-to-finance", "request-approval"},
	}

	payload, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal triage record: %v", err)
	}

	var restored TriageRecord
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal triage record: %v", err)
	}

	if !reflect.DeepEqual(restored, record) {
		t.Fatalf("triage record mismatch: restored=%+v want=%+v", restored, record)
	}
	if restored.Labels[0].Name != "risk" || restored.Labels[1].Name != "billing" {
		t.Fatalf("expected labels to survive round trip, got %+v", restored)
	}
}
