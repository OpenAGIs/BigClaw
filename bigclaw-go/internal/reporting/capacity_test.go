package reporting

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildCapacityCertification(t *testing.T) {
	root := t.TempDir()
	writeFixtureJSON(t, filepath.Join(root, "bigclaw-go/docs/reports/benchmark-matrix-report.json"), map[string]any{
		"benchmark": map[string]any{
			"parsed": map[string]any{
				"BenchmarkMemoryQueueEnqueueLease-8": map[string]any{"ns_per_op": 66075.0},
				"BenchmarkFileQueueEnqueueLease-8":   map[string]any{"ns_per_op": 31627767.0},
				"BenchmarkSQLiteQueueEnqueueLease-8": map[string]any{"ns_per_op": 18057898.0},
				"BenchmarkSchedulerDecide-8":         map[string]any{"ns_per_op": 73.98},
			},
		},
		"soak_matrix": []map[string]any{
			{"report_path": "docs/reports/soak-local-50x8.json", "result": map[string]any{"count": 50, "workers": 8, "elapsed_seconds": 8.232, "throughput_tasks_per_sec": 6.074, "succeeded": 50, "failed": 0}},
			{"report_path": "docs/reports/soak-local-100x12.json", "result": map[string]any{"count": 100, "workers": 12, "elapsed_seconds": 10.294, "throughput_tasks_per_sec": 9.714, "succeeded": 100, "failed": 0}},
		},
	})
	writeFixtureJSON(t, filepath.Join(root, "bigclaw-go/docs/reports/mixed-workload-matrix-report.json"), map[string]any{
		"all_ok": true,
		"tasks": []map[string]any{
			{"name": "a", "ok": true, "expected_executor": "local", "routed_executor": "local", "final_state": "succeeded"},
			{"name": "b", "ok": true, "expected_executor": "local", "routed_executor": "local", "final_state": "succeeded"},
			{"name": "c", "ok": true, "expected_executor": "local", "routed_executor": "local", "final_state": "succeeded"},
			{"name": "d", "ok": true, "expected_executor": "local", "routed_executor": "local", "final_state": "succeeded"},
			{"name": "e", "ok": true, "expected_executor": "local", "routed_executor": "local", "final_state": "succeeded"},
		},
	})
	writeFixtureJSON(t, filepath.Join(root, "bigclaw-go/docs/reports/soak-local-1000x24.json"), map[string]any{"count": 1000, "workers": 24, "elapsed_seconds": 104.091, "throughput_tasks_per_sec": 9.607, "succeeded": 1000, "failed": 0})
	writeFixtureJSON(t, filepath.Join(root, "bigclaw-go/docs/reports/soak-local-2000x24.json"), map[string]any{"count": 2000, "workers": 24, "elapsed_seconds": 219.167, "throughput_tasks_per_sec": 9.125, "succeeded": 2000, "failed": 0})

	report, markdown, err := BuildCapacityCertification(root, CapacityCertificationOptions{})
	if err != nil {
		t.Fatalf("build capacity certification: %v", err)
	}
	if asString(asMap(report["evidence_inputs"])["generator_script"]) != CapacityCertificationGenerator {
		t.Fatalf("unexpected generator script: %+v", report["evidence_inputs"])
	}
	summary := asMap(report["summary"])
	if asString(summary["overall_status"]) != "pass" || asInt(summary["passed_lanes"]) != 9 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if !strings.Contains(markdown, "# Capacity Certification Report") || !strings.Contains(markdown, "Recommended Operating Envelopes") {
		t.Fatalf("unexpected markdown: %s", markdown)
	}
}
