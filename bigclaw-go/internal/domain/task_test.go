package domain

import (
	"encoding/json"
	"testing"
)

func TestTaskUnmarshalAcceptsPythonCompatShape(t *testing.T) {
	payload := []byte(`{
		"task_id":"BIG-301",
		"source":"linear",
		"title":"Budget override migration",
		"description":"Port Python task payloads",
		"labels":["migration","prod"],
		"priority":1,
		"state":"Todo",
		"risk_level":"high",
		"budget":12.5,
		"budget_override_actor":"reviewer",
		"budget_override_reason":"approved exception",
		"budget_override_amount":2.25,
		"required_tools":["github"],
		"acceptance_criteria":["parity"],
		"validation_plan":["go test"]
	}`)

	var task Task
	if err := json.Unmarshal(payload, &task); err != nil {
		t.Fatalf("unmarshal python task payload: %v", err)
	}
	if task.ID != "BIG-301" {
		t.Fatalf("expected ID BIG-301, got %q", task.ID)
	}
	if task.BudgetCents != 1250 {
		t.Fatalf("expected budget 1250 cents, got %d", task.BudgetCents)
	}
	if task.BudgetOverrideActor != "reviewer" {
		t.Fatalf("expected override actor reviewer, got %q", task.BudgetOverrideActor)
	}
	if task.BudgetOverrideReason != "approved exception" {
		t.Fatalf("expected override reason preserved, got %q", task.BudgetOverrideReason)
	}
	if task.BudgetOverrideAmountCents != 225 {
		t.Fatalf("expected override amount 225 cents, got %d", task.BudgetOverrideAmountCents)
	}
}

func TestTaskUnmarshalPrefersCanonicalBudgetFieldsWhenPresent(t *testing.T) {
	payload := []byte(`{
		"id":"BIG-302",
		"task_id":"ignored",
		"budget":12.5,
		"budget_cents":900,
		"budget_override_amount":2.25,
		"budget_override_amount_cents":300
	}`)

	var task Task
	if err := json.Unmarshal(payload, &task); err != nil {
		t.Fatalf("unmarshal mixed task payload: %v", err)
	}
	if task.ID != "BIG-302" {
		t.Fatalf("expected canonical ID BIG-302, got %q", task.ID)
	}
	if task.BudgetCents != 900 {
		t.Fatalf("expected canonical budget_cents 900, got %d", task.BudgetCents)
	}
	if task.BudgetOverrideAmountCents != 300 {
		t.Fatalf("expected canonical budget_override_amount_cents 300, got %d", task.BudgetOverrideAmountCents)
	}
}

func TestTaskMarshalIncludesPythonCompatAliases(t *testing.T) {
	task := Task{
		ID:                        "BIG-303",
		Source:                    "linear",
		Title:                     "Emit compat task payloads",
		BudgetCents:               1250,
		BudgetOverrideActor:       "reviewer",
		BudgetOverrideReason:      "manual approval",
		BudgetOverrideAmountCents: 225,
	}

	encoded, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("decode marshalled task: %v", err)
	}
	if payload["id"] != "BIG-303" || payload["task_id"] != "BIG-303" {
		t.Fatalf("expected id/task_id aliases to match, got %#v", payload)
	}
	if payload["budget_cents"] != float64(1250) {
		t.Fatalf("expected budget_cents 1250, got %#v", payload["budget_cents"])
	}
	if payload["budget"] != 12.5 {
		t.Fatalf("expected budget 12.5, got %#v", payload["budget"])
	}
	if payload["budget_override_amount_cents"] != float64(225) {
		t.Fatalf("expected budget_override_amount_cents 225, got %#v", payload["budget_override_amount_cents"])
	}
	if payload["budget_override_amount"] != 2.25 {
		t.Fatalf("expected budget_override_amount 2.25, got %#v", payload["budget_override_amount"])
	}
}
