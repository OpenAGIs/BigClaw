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

func TestTriageRecordJSONEmitsPythonContractDefaults(t *testing.T) {
	record := TriageRecord{TriageID: "triage-2", TaskID: "OPE-131"}

	payload, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal triage record: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode triage record: %v", err)
	}

	for _, key := range []string{"status", "queue", "owner", "summary", "labels", "related_run_id", "escalation_target", "actions"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("expected key %q in triage JSON, got %+v", key, decoded)
		}
	}
	if decoded["status"] != string(StatusOpen) || decoded["queue"] != "default" {
		t.Fatalf("expected default status/queue in JSON, got %+v", decoded)
	}

	labelPayload, err := json.Marshal(Label{Name: "risk"})
	if err != nil {
		t.Fatalf("marshal label: %v", err)
	}
	var decodedLabel map[string]any
	if err := json.Unmarshal(labelPayload, &decodedLabel); err != nil {
		t.Fatalf("decode label: %v", err)
	}
	if decodedLabel["confidence"] != float64(1) {
		t.Fatalf("expected default confidence in JSON, got %+v", decodedLabel)
	}
	if _, ok := decodedLabel["source"]; !ok {
		t.Fatalf("expected source key in label JSON, got %+v", decodedLabel)
	}
}
