package main

import "testing"

func TestDefaultMixedTasksMatchExpectedProfiles(t *testing.T) {
	tasks := defaultMixedTasks(42)
	if len(tasks) != 5 {
		t.Fatalf("expected 5 tasks, got %d", len(tasks))
	}
	expected := []struct {
		name     string
		executor string
		taskID   string
		profile  string
	}{
		{"local-default", "local", "mixed-local-42", "local-default"},
		{"browser-auto", "kubernetes", "mixed-browser-42", "browser-auto"},
		{"gpu-auto", "ray", "mixed-gpu-42", "gpu-auto"},
		{"high-risk-auto", "kubernetes", "mixed-risk-42", "high-risk-auto"},
		{"required-ray", "ray", "mixed-required-ray-42", "required-ray"},
	}
	for index, item := range expected {
		task := tasks[index]
		if task.Name != item.name || task.ExpectedExecutor != item.executor {
			t.Fatalf("unexpected task header at %d: %+v", index, task)
		}
		if asString(task.Task["id"]) != item.taskID || asString(task.Task["trace_id"]) != item.taskID {
			t.Fatalf("unexpected task ids at %d: %+v", index, task.Task)
		}
		metadata := asMap(task.Task["metadata"])
		if asString(metadata["scenario"]) != "mixed-workload" || asString(metadata["profile"]) != item.profile {
			t.Fatalf("unexpected metadata at %d: %+v", index, metadata)
		}
	}
}

func TestFirstRoutedEventReturnsFirstSchedulerRoute(t *testing.T) {
	events := []map[string]any{
		{"type": "task.queued"},
		{"type": "scheduler.routed", "payload": map[string]any{"executor": "kubernetes"}},
		{"type": "scheduler.routed", "payload": map[string]any{"executor": "ray"}},
	}
	routed := firstRoutedEvent(events)
	if routed == nil {
		t.Fatal("expected routed event")
	}
	if asString(asMap(routed["payload"])["executor"]) != "kubernetes" {
		t.Fatalf("unexpected routed payload: %+v", routed)
	}
}

func TestLatestEventTypeHandlesMissingLatestEvent(t *testing.T) {
	if latestEventType(map[string]any{}) != "" {
		t.Fatal("expected empty latest event type")
	}
	if latestEventType(map[string]any{"latest_event": map[string]any{"type": "task.completed"}}) != "task.completed" {
		t.Fatal("expected latest event type to be extracted")
	}
}
