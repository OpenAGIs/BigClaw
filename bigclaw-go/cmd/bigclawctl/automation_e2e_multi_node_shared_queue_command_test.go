package main

import "testing"

func TestAutomationMultiNodeSharedQueueSummarize(t *testing.T) {
	tasks := map[string]string{"task-a": "node-a", "task-b": "node-b"}
	events := []map[string]any{
		{"task_id": "task-a", "type": "task.queued", "_node": "node-a"},
		{"task_id": "task-a", "type": "task.started", "_node": "node-b"},
		{"task_id": "task-a", "type": "task.completed", "_node": "node-b"},
		{"task_id": "task-b", "type": "task.queued", "_node": "node-b"},
		{"task_id": "task-b", "type": "task.started", "_node": "node-b"},
		{"task_id": "task-b", "type": "task.completed", "_node": "node-b"},
	}
	summary := summarizeSharedQueue(tasks, events)
	if len(summary["task-a"]["completed"]) != 1 || summary["task-a"]["completed"][0] != "node-b" {
		t.Fatalf("unexpected task-a summary: %+v", summary["task-a"])
	}
	if len(summary["task-b"]["queued"]) != 1 || summary["task-b"]["queued"][0] != "node-b" {
		t.Fatalf("unexpected task-b summary: %+v", summary["task-b"])
	}
}

func TestAutomationMultiNodeSharedQueueBuildLiveTakeoverReport(t *testing.T) {
	report := buildLiveTakeoverReportSharedQueue([]map[string]any{
		{"all_assertions_passed": true, "duplicate_delivery_count": 1, "stale_write_rejections": 1},
		{"all_assertions_passed": true, "duplicate_delivery_count": 2, "stale_write_rejections": 1},
	}, "docs/reports/multi-node-shared-queue-report.json")
	summary, _ := report["summary"].(map[string]any)
	if asInt(summary["scenario_count"]) != 2 || asInt(summary["passing_scenarios"]) != 2 || asInt(summary["duplicate_delivery_count"]) != 3 || asInt(summary["stale_write_rejections"]) != 2 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	currentPrimitives, _ := report["current_primitives"].(map[string]any)
	sharedQueueEvidence, _ := currentPrimitives["shared_queue_evidence"].([]any)
	if len(sharedQueueEvidence) != 2 || asString(sharedQueueEvidence[0]) != "cmd/bigclawctl/automation_e2e_multi_node_shared_queue_command.go" {
		t.Fatalf("unexpected current primitives: %+v", currentPrimitives)
	}
}
