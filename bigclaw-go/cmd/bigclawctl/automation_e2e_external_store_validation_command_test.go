package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAutomationExternalStoreValidationBuildBackendMatrix(t *testing.T) {
	report := externalStoreBuildBackendMatrix("http", true)
	summary, _ := report["summary"].(map[string]any)
	if asInt(summary["live_validated_lanes"]) != 1 || asInt(summary["not_configured_lanes"]) != 1 || asInt(summary["contract_only_lanes"]) != 1 {
		t.Fatalf("unexpected backend matrix summary: %+v", summary)
	}
	lanes, _ := report["lanes"].([]any)
	if len(lanes) != 3 {
		t.Fatalf("unexpected backend matrix lanes: %+v", lanes)
	}
}

func TestAutomationExternalStoreValidationWritesReport(t *testing.T) {
	root := t.TempDir()
	report := externalStoreBuildReport(
		time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC),
		"2s",
		externalStoreReplayTaskID,
		externalStoreReplayTraceID,
		map[string]any{"state": "succeeded", "id": externalStoreReplayTaskID},
		map[string]any{"backend": "http", "durable": true},
		[]map[string]any{{"id": "evt-1", "type": "task.queued"}, {"id": "evt-2", "type": "task.started"}, {"id": "evt-3", "type": "task.completed"}},
		"evt-3",
		map[string]any{"checkpoint": map[string]any{"event_id": "evt-3"}},
		map[string]any{"checkpoint": map[string]any{"event_id": "evt-3"}},
		map[string]any{"history": []any{map[string]any{"event_id": "evt-3"}}},
		map[string]any{},
		[]map[string]any{{"id": "evt-external-retention-new"}},
		map[string]any{"history_truncated": true, "persisted_boundary": true, "trimmed_through_event_id": "evt-external-retention-old"},
		map[string]any{"lease": map[string]any{"consumer_id": "node-a", "lease_epoch": 1}},
		map[string]any{"lease": map[string]any{"checkpoint_offset": 11}},
		409,
		map[string]any{"lease": map[string]any{"consumer_id": "node-b", "lease_epoch": 2}},
		409,
		map[string]any{"lease": map[string]any{"checkpoint_offset": 15}},
		map[string]any{"lease": map[string]any{"consumer_id": "node-b"}},
	)
	if err := automationWriteReport(root, "tmp/report.json", report); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "tmp/report.json")); err != nil {
		t.Fatalf("expected output file: %v", err)
	}
	summary, _ := report["summary"].(map[string]any)
	if summary["remote_replay_backend"] != "http" || summary["stale_writer_rejected"] != true {
		t.Fatalf("unexpected report summary: %+v", summary)
	}
}
