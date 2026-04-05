package reporting

import (
	"testing"
	"time"
)

func TestBuildExternalStoreBackendMatrix(t *testing.T) {
	matrix := buildExternalStoreBackendMatrix("http", true)
	summary := asMap(matrix["summary"])
	if asInt(summary["live_validated_lanes"]) != 1 || asInt(summary["not_configured_lanes"]) != 1 || asInt(summary["contract_only_lanes"]) != 1 {
		t.Fatalf("unexpected backend matrix summary: %+v", summary)
	}
	lanes := anyToMapSlice(matrix["lanes"])
	if len(lanes) != 3 {
		t.Fatalf("unexpected backend matrix lanes: %+v", lanes)
	}
	if asString(lanes[0]["backend"]) != "http_remote_service" || asString(lanes[0]["replay_backend"]) != "http" || !asBool(lanes[0]["retention_boundary_visible"]) {
		t.Fatalf("unexpected http lane: %+v", lanes[0])
	}
	if asString(lanes[1]["validation_status"]) != "not_configured" || asString(lanes[2]["validation_status"]) != "contract_only" {
		t.Fatalf("unexpected placeholder lanes: %+v", lanes)
	}
}

func TestBuildExternalStoreValidationReport(t *testing.T) {
	report := buildExternalStoreValidationReport(time.Date(2026, 3, 17, 11, 15, 53, 753800000, time.UTC), "2s", externalStoreRuntimeArtifacts{
		replayPayload: map[string]any{"backend": "http", "durable": true},
		replayEvents: []map[string]any{
			{"id": "external-store-smoke-task-queued", "type": "task.queued"},
			{"id": "external-store-smoke-task-completed", "type": "task.completed"},
		},
		checkpointWrite:   map[string]any{"checkpoint": map[string]any{"event_id": "external-store-smoke-task-completed"}},
		checkpointRead:    map[string]any{"checkpoint": map[string]any{"event_id": "external-store-smoke-task-completed"}},
		checkpointHistory: map[string]any{"history": []any{map[string]any{"event_id": "external-store-smoke-task-completed"}}},
		retentionEvents:   []map[string]any{{"id": "evt-external-retention-new"}},
		retentionMark: map[string]any{
			"history_truncated":        true,
			"persisted_boundary":       true,
			"trimmed_through_event_id": "evt-external-retention-old",
			"oldest_event_id":          "external-store-smoke-task-queued",
			"newest_event_id":          "evt-external-retention-new",
		},
		submittedTask:  map[string]any{"id": externalStoreReplayTaskID, "trace_id": externalStoreReplayTraceID},
		finalStatus:    map[string]any{"state": "succeeded"},
		leaseA:         map[string]any{"lease": map[string]any{"consumer_id": "node-a", "lease_epoch": 1}},
		checkpointA:    map[string]any{"lease": map[string]any{"checkpoint_offset": 11}},
		leaseB:         map[string]any{"lease": map[string]any{"consumer_id": "node-b", "lease_epoch": 2}},
		checkpointB:    map[string]any{"lease": map[string]any{"checkpoint_offset": 15}},
		leaseStatus:    map[string]any{"lease": map[string]any{"consumer_id": "node-b"}},
		conflictStatus: 409,
		staleStatus:    409,
	})
	if asString(report["ticket"]) != "BIG-PAR-102" || asString(report["status"]) != "validated" {
		t.Fatalf("unexpected external store report metadata: %+v", report)
	}
	summary := asMap(report["summary"])
	if !asBool(summary["task_succeeded"]) || asString(summary["remote_replay_backend"]) != "http" || !asBool(summary["stale_writer_rejected"]) {
		t.Fatalf("unexpected external store summary: %+v", summary)
	}
	takeover := asMap(report["takeover_validation"])
	if asString(takeover["takeover_consumer"]) != "node-b" || asInt(takeover["takeover_epoch"]) != 2 || asInt(takeover["final_checkpoint_offset"]) != 15 {
		t.Fatalf("unexpected takeover validation: %+v", takeover)
	}
}
