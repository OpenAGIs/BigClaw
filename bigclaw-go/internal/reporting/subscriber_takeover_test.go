package reporting

import (
	"path/filepath"
	"testing"
	"time"
)

func TestBuildSubscriberTakeoverReport(t *testing.T) {
	report := BuildSubscriberTakeoverReport(time.Date(2026, 3, 16, 10, 20, 20, 246671000, time.UTC))
	if asString(report["ticket"]) != "OPE-269" || asString(report["status"]) != "local-executable" {
		t.Fatalf("unexpected takeover report metadata: %+v", report)
	}
	summary := asMap(report["summary"])
	if asInt(summary["scenario_count"]) != 3 || asInt(summary["passing_scenarios"]) != 3 || asInt(summary["failing_scenarios"]) != 0 {
		t.Fatalf("unexpected takeover summary: %+v", summary)
	}
	if asInt(summary["stale_write_rejections"]) != 2 || asInt(summary["duplicate_delivery_count"]) != 4 {
		t.Fatalf("unexpected takeover counters: %+v", summary)
	}
	primitives := asMap(report["current_primitives"])
	takeoverHarness := stringSliceFromAny(primitives["takeover_harness"])
	if len(takeoverHarness) != 2 || takeoverHarness[0] != "scripts/e2e/subscriber_takeover_fault_matrix/main.go" {
		t.Fatalf("unexpected takeover harness primitive: %+v", takeoverHarness)
	}
	scenarios := anyToMapSlice(report["scenarios"])
	if len(scenarios) != 3 {
		t.Fatalf("unexpected scenarios: %+v", scenarios)
	}
	var staleWriter map[string]any
	for _, scenario := range scenarios {
		if asString(scenario["id"]) == "lease-expiry-stale-writer-rejected" {
			staleWriter = scenario
			break
		}
	}
	if staleWriter == nil {
		t.Fatal("missing stale writer scenario")
	}
	if asInt(staleWriter["stale_write_rejections"]) != 1 {
		t.Fatalf("unexpected stale writer rejection count: %+v", staleWriter)
	}
	checkpointAfter := asMap(staleWriter["checkpoint_after"])
	if asString(checkpointAfter["owner"]) != asString(staleWriter["takeover_subscriber"]) {
		t.Fatalf("unexpected stale writer checkpoint owner: %+v", checkpointAfter)
	}
	duplicates := stringSliceFromAny(staleWriter["duplicate_events"])
	if len(duplicates) != 1 || duplicates[0] != "evt-81" || !asBool(staleWriter["all_assertions_passed"]) {
		t.Fatalf("unexpected stale writer scenario details: %+v", staleWriter)
	}
}

func TestWriteSubscriberTakeoverArtifacts(t *testing.T) {
	root := t.TempDir()
	if err := WriteSubscriberTakeoverArtifacts(root, SubscriberTakeoverOptions{}); err != nil {
		t.Fatalf("write takeover artifacts: %v", err)
	}
	if !pathExists(filepath.Join(root, "bigclaw-go/docs/reports/multi-subscriber-takeover-validation-report.json")) {
		t.Fatal("expected takeover report artifact")
	}
}
