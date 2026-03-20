package domain

import (
	"encoding/json"
	"testing"
)

func TestTaskJSONSupportsLegacyTaskIDAndBudgetOverrides(t *testing.T) {
	payload := []byte(`{"task_id":"BIG-401","source":"linear","title":"Budget override","budget_override_actor":"lead","budget_override_reason":"approved","budget_override_amount":12.5}`)

	var task Task
	if err := json.Unmarshal(payload, &task); err != nil {
		t.Fatalf("unmarshal task: %v", err)
	}
	if task.ID != "BIG-401" {
		t.Fatalf("expected task id from legacy field, got %+v", task)
	}
	if task.BudgetOverrideActor != "lead" || task.BudgetOverrideReason != "approved" || task.BudgetOverrideAmount != 12.5 {
		t.Fatalf("expected budget override fields to round-trip, got %+v", task)
	}

	encoded, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decode marshaled task: %v", err)
	}
	if decoded["id"] != "BIG-401" || decoded["task_id"] != "BIG-401" {
		t.Fatalf("expected both id and task_id in JSON, got %+v", decoded)
	}
	if decoded["budget_override_actor"] != "lead" || decoded["budget_override_reason"] != "approved" || decoded["budget_override_amount"] != 12.5 {
		t.Fatalf("expected budget override JSON fields, got %+v", decoded)
	}
}

func TestTaskJSONSupportsLegacyBudgetField(t *testing.T) {
	payload := []byte(`{"task_id":"BIG-402","source":"linear","title":"Legacy budget","budget":12.34}`)

	var task Task
	if err := json.Unmarshal(payload, &task); err != nil {
		t.Fatalf("unmarshal task: %v", err)
	}
	if task.BudgetCents != 1234 {
		t.Fatalf("expected budget cents from legacy budget field, got %+v", task)
	}

	encoded, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decode marshaled task: %v", err)
	}
	if decoded["budget"] != 12.34 {
		t.Fatalf("expected legacy budget field in JSON, got %+v", decoded)
	}
	if decoded["budget_cents"] != float64(1234) {
		t.Fatalf("expected canonical budget_cents field in JSON, got %+v", decoded)
	}
}
