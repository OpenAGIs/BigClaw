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
	if decoded["state"] != string(TaskQueued) || decoded["risk_level"] != string(RiskLow) {
		t.Fatalf("expected default state and risk level in JSON, got %+v", decoded)
	}
	if labels, ok := decoded["labels"].([]any); !ok || len(labels) != 0 {
		t.Fatalf("expected empty labels in JSON, got %+v", decoded["labels"])
	}
	if requiredTools, ok := decoded["required_tools"].([]any); !ok || len(requiredTools) != 0 {
		t.Fatalf("expected empty required_tools in JSON, got %+v", decoded["required_tools"])
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

func TestTaskJSONEmitsPythonContractDefaults(t *testing.T) {
	task := Task{ID: "BIG-404", Title: "Default contract"}

	encoded, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decode marshaled task: %v", err)
	}
	for _, key := range []string{"source", "description", "labels", "priority", "state", "risk_level", "budget", "budget_override_actor", "budget_override_reason", "budget_override_amount", "required_tools", "acceptance_criteria", "validation_plan"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("expected key %q in task JSON, got %+v", key, decoded)
		}
	}
	if decoded["state"] != string(TaskQueued) || decoded["risk_level"] != string(RiskLow) || decoded["budget"] != float64(0) {
		t.Fatalf("expected Python-style defaults in task JSON, got %+v", decoded)
	}
}

func TestTaskJSONNormalizesLegacyTaskStates(t *testing.T) {
	tests := map[string]TaskState{
		"Todo":        TaskQueued,
		"In Progress": TaskRunning,
		"Blocked":     TaskBlocked,
		"Done":        TaskSucceeded,
		"Failed":      TaskFailed,
	}

	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			payload := []byte(`{"task_id":"BIG-403","source":"linear","title":"Legacy state","state":"` + input + `"}`)

			var task Task
			if err := json.Unmarshal(payload, &task); err != nil {
				t.Fatalf("unmarshal task: %v", err)
			}
			if task.State != want {
				t.Fatalf("expected normalized state %q, got %+v", want, task)
			}
			if task.Labels == nil || task.RequiredTools == nil || task.AcceptanceCriteria == nil || task.ValidationPlan == nil {
				t.Fatalf("expected non-nil task collections, got %+v", task)
			}
		})
	}
}
