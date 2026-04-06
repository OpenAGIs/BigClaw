package reporting

import "testing"

func TestBuildLiveTakeoverReportUsesGoEntrypoint(t *testing.T) {
	report := buildLiveTakeoverReport([]map[string]any{
		{"all_assertions_passed": true, "duplicate_delivery_count": 1, "stale_write_rejections": 1},
		{"all_assertions_passed": true, "duplicate_delivery_count": 2, "stale_write_rejections": 1},
		{"all_assertions_passed": true, "duplicate_delivery_count": 1, "stale_write_rejections": 1},
	}, "docs/reports/multi-node-shared-queue-report.json")
	primitives := asMap(report["current_primitives"])
	sharedQueue := stringSliceFromAny(primitives["shared_queue_evidence"])
	liveHarness := stringSliceFromAny(primitives["live_takeover_harness"])
	if len(sharedQueue) != 2 || sharedQueue[0] != "scripts/e2e/multi_node_shared_queue/main.go" {
		t.Fatalf("unexpected shared queue evidence: %+v", sharedQueue)
	}
	if len(liveHarness) != 3 || liveHarness[1] != "scripts/e2e/multi_node_shared_queue/main.go" {
		t.Fatalf("unexpected live takeover harness: %+v", liveHarness)
	}
	summary := asMap(report["summary"])
	if asInt(summary["scenario_count"]) != 3 || asInt(summary["passing_scenarios"]) != 3 || asInt(summary["duplicate_delivery_count"]) != 4 || asInt(summary["stale_write_rejections"]) != 3 {
		t.Fatalf("unexpected live takeover summary: %+v", summary)
	}
}
