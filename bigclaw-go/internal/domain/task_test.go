package domain

import (
	"encoding/json"
	"testing"
	"time"
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

func TestTaskJSONMarshalIncludesOptionalExecutionFields(t *testing.T) {
	createdAt := time.Date(2026, time.March, 25, 18, 12, 13, 0, time.UTC)
	updatedAt := createdAt.Add(2 * time.Hour)
	task := Task{
		ID:                      "BIG-405",
		TraceID:                 "trace-405",
		Source:                  "tracker",
		Title:                   "Optional execution fields",
		RequiredExecutor:        ExecutorRay,
		IdempotencyKey:          "idem-405",
		TenantID:                "tenant-405",
		ContainerImage:          "ghcr.io/openagis/bigclaw:latest",
		Entrypoint:              "/bin/run-task",
		Command:                 []string{"run"},
		Args:                    []string{"--json", "--verbose"},
		Environment:             map[string]string{"MODE": "prod"},
		RuntimeEnv:              map[string]any{"cpu": float64(2), "team": "ops"},
		Metadata:                map[string]string{"ticket": "BIG-PAR-397"},
		WorkingDir:              "/workspace",
		ExecutionTimeoutSeconds: 900,
		CreatedAt:               createdAt,
		UpdatedAt:               updatedAt,
	}

	encoded, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decode marshaled task: %v", err)
	}
	if decoded["trace_id"] != task.TraceID {
		t.Fatalf("expected trace_id in JSON, got %+v", decoded)
	}
	if decoded["required_executor"] != string(task.RequiredExecutor) {
		t.Fatalf("expected required_executor in JSON, got %+v", decoded)
	}
	if decoded["idempotency_key"] != task.IdempotencyKey || decoded["tenant_id"] != task.TenantID {
		t.Fatalf("expected idempotency and tenant fields in JSON, got %+v", decoded)
	}
	if decoded["container_image"] != task.ContainerImage || decoded["entrypoint"] != task.Entrypoint {
		t.Fatalf("expected container execution fields in JSON, got %+v", decoded)
	}
	if command, ok := decoded["command"].([]any); !ok || len(command) != 1 || command[0] != "run" {
		t.Fatalf("expected command in JSON, got %+v", decoded["command"])
	}
	if args, ok := decoded["args"].([]any); !ok || len(args) != 2 || args[0] != "--json" || args[1] != "--verbose" {
		t.Fatalf("expected args in JSON, got %+v", decoded["args"])
	}
	if environment, ok := decoded["environment"].(map[string]any); !ok || environment["MODE"] != "prod" {
		t.Fatalf("expected environment in JSON, got %+v", decoded["environment"])
	}
	if runtimeEnv, ok := decoded["runtime_env"].(map[string]any); !ok || runtimeEnv["cpu"] != float64(2) || runtimeEnv["team"] != "ops" {
		t.Fatalf("expected runtime_env in JSON, got %+v", decoded["runtime_env"])
	}
	if metadata, ok := decoded["metadata"].(map[string]any); !ok || metadata["ticket"] != "BIG-PAR-397" {
		t.Fatalf("expected metadata in JSON, got %+v", decoded["metadata"])
	}
	if decoded["working_dir"] != task.WorkingDir || decoded["execution_timeout_seconds"] != float64(task.ExecutionTimeoutSeconds) {
		t.Fatalf("expected working_dir and execution timeout in JSON, got %+v", decoded)
	}
	if decoded["created_at"] != createdAt.Format(time.RFC3339) || decoded["updated_at"] != updatedAt.Format(time.RFC3339) {
		t.Fatalf("expected created_at and updated_at in JSON, got %+v", decoded)
	}
}

func TestTaskJSONUnmarshalRejectsInvalidJSON(t *testing.T) {
	var task Task
	if err := json.Unmarshal([]byte(`{"task_id":"BIG-ERR","title":"Bad budget","budget":"oops"}`), &task); err == nil {
		t.Fatal("expected invalid task payload to fail")
	}
}

func TestTaskJSONUnmarshalPrefersCanonicalFieldsAndNormalizesCollections(t *testing.T) {
	payload := []byte(`{
		"id":"BIG-406",
		"task_id":"LEGACY-406",
		"source":"tracker",
		"title":"Canonical task id",
		"budget_cents":4321,
		"budget":99.99,
		"labels":null,
		"required_tools":null,
		"acceptance_criteria":null,
		"validation_plan":null
	}`)

	var task Task
	if err := json.Unmarshal(payload, &task); err != nil {
		t.Fatalf("unmarshal task: %v", err)
	}
	if task.ID != "BIG-406" {
		t.Fatalf("expected canonical id to win over task_id, got %+v", task)
	}
	if task.BudgetCents != 4321 {
		t.Fatalf("expected budget_cents to win over legacy budget, got %+v", task)
	}
	if task.Labels == nil || len(task.Labels) != 0 {
		t.Fatalf("expected labels normalized to empty slice, got %#v", task.Labels)
	}
	if task.RequiredTools == nil || len(task.RequiredTools) != 0 {
		t.Fatalf("expected required_tools normalized to empty slice, got %#v", task.RequiredTools)
	}
	if task.AcceptanceCriteria == nil || len(task.AcceptanceCriteria) != 0 {
		t.Fatalf("expected acceptance_criteria normalized to empty slice, got %#v", task.AcceptanceCriteria)
	}
	if task.ValidationPlan == nil || len(task.ValidationPlan) != 0 {
		t.Fatalf("expected validation_plan normalized to empty slice, got %#v", task.ValidationPlan)
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

func TestTaskStateHelpers(t *testing.T) {
	mapped := map[EventType]TaskState{
		EventTaskQueued:      TaskQueued,
		EventTaskLeased:      TaskLeased,
		EventSchedulerRouted: TaskLeased,
		EventTaskStarted:     TaskRunning,
		EventTaskRetried:     TaskRetrying,
		EventTaskCompleted:   TaskSucceeded,
		EventTaskPreempted:   TaskCancelled,
		EventTaskCancelled:   TaskCancelled,
		EventTaskDeadLetter:  TaskDeadLetter,
	}
	for eventType, want := range mapped {
		if got, ok := TaskStateFromEventType(eventType); !ok || got != want {
			t.Fatalf("expected event %q to map to %q, got %q ok=%v", eventType, want, got, ok)
		}
	}
	if got, ok := TaskStateFromEventType(EventRunAnnotated); ok || got != "" {
		t.Fatalf("expected unmapped event type to fail lookup, got %q ok=%v", got, ok)
	}

	active := map[TaskState]bool{
		TaskQueued:    true,
		TaskLeased:    true,
		TaskRunning:   true,
		TaskBlocked:   true,
		TaskRetrying:  true,
		TaskSucceeded: false,
		TaskCancelled: false,
		TaskDeadLetter:false,
		TaskFailed:    false,
		"":            false,
	}
	for state, want := range active {
		if got := IsActiveTaskState(state); got != want {
			t.Fatalf("expected state %q active=%v, got %v", state, want, got)
		}
	}
}

func TestTaskMarshalRiskAndNormalizeStateHelpers(t *testing.T) {
	if got := marshalRiskLevel(""); got != string(RiskLow) {
		t.Fatalf("expected empty risk level to default low, got %q", got)
	}
	if got := marshalRiskLevel(RiskHigh); got != string(RiskHigh) {
		t.Fatalf("expected explicit risk level to round-trip, got %q", got)
	}

	tests := map[TaskState]TaskState{
		"":               TaskQueued,
		TaskLeased:       TaskLeased,
		TaskRunning:      TaskRunning,
		TaskBlocked:      TaskBlocked,
		TaskRetrying:     TaskRetrying,
		TaskSucceeded:    TaskSucceeded,
		TaskFailed:       TaskFailed,
		TaskDeadLetter:   TaskDeadLetter,
		" canceled ":     TaskCancelled,
		"Closed - done":  TaskSucceeded,
		"Resolved":       TaskSucceeded,
		"failing hard":   TaskFailed,
		"Blocked on QA":  TaskBlocked,
		"custom-status":  TaskState("custom-status"),
	}
	for input, want := range tests {
		if got := normalizeTaskState(input); got != want {
			t.Fatalf("expected normalized state %q for %q, got %q", want, input, got)
		}
	}
}
