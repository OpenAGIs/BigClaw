package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAutomationBenchmarkReportContractIncludesParsedBenchmarksAndSoakOutputs(t *testing.T) {
	root := t.TempDir()
	report, err := automationBenchmarkRunMatrix(automationBenchmarkRunMatrixOptions{
		GoRoot:         root,
		ReportPath:     "docs/reports/benchmark-matrix-report.json",
		TimeoutSeconds: 90,
		Scenarios:      []string{"3:2"},
		RunBenchmark: func(string) (string, error) {
			return "BenchmarkSchedulerDecide-8\t100\t73.98 ns/op\n", nil
		},
		RunSoak: func(_ string, count, workers, timeoutSeconds int, reportPath string) (map[string]any, error) {
			return map[string]any{
				"count":                    count,
				"workers":                  workers,
				"elapsed_seconds":          1.5,
				"throughput_tasks_per_sec": 2.0,
				"succeeded":                count,
				"failed":                   0,
				"timeout_seconds":          timeoutSeconds,
				"report_path":              reportPath,
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("run benchmark matrix: %v", err)
	}

	benchmark, ok := report["benchmark"].(map[string]any)
	if !ok {
		t.Fatalf("expected benchmark payload, got %+v", report)
	}
	parsed, ok := benchmark["parsed"].(map[string]any)
	if !ok || len(parsed) != 1 {
		t.Fatalf("expected parsed benchmark summary, got %+v", benchmark)
	}
	schedulerLane, ok := parsed["BenchmarkSchedulerDecide-8"].(map[string]any)
	if !ok || schedulerLane["ns_per_op"] != 73.98 {
		t.Fatalf("expected parsed scheduler benchmark, got %+v", parsed)
	}
	soakMatrix, ok := report["soak_matrix"].([]any)
	if !ok || len(soakMatrix) != 1 {
		t.Fatalf("expected single soak scenario, got %+v", report)
	}

	body, err := os.ReadFile(filepath.Join(root, "docs/reports/benchmark-matrix-report.json"))
	if err != nil {
		t.Fatalf("read matrix report: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, `"BenchmarkSchedulerDecide-8"`) || !strings.Contains(text, `"report_path": "docs/reports/soak-local-3x2.json"`) {
		t.Fatalf("unexpected matrix report body: %s", text)
	}
}

func TestAutomationCapacityCertificationContractIncludesBenchmarkAndMixedWorkloadSignals(t *testing.T) {
	root := t.TempDir()
	for path, body := range map[string]string{
		filepath.Join(root, "docs/reports/benchmark-matrix-report.json"): `{
  "benchmark": {
    "parsed": {
      "BenchmarkMemoryQueueEnqueueLease-8": {"ns_per_op": 66075.0},
      "BenchmarkFileQueueEnqueueLease-8": {"ns_per_op": 31627767.0},
      "BenchmarkSQLiteQueueEnqueueLease-8": {"ns_per_op": 18057898.0},
      "BenchmarkSchedulerDecide-8": {"ns_per_op": 73.98}
    }
  },
  "soak_matrix": [
    {"report_path": "docs/reports/soak-local-50x8.json", "result": {"count": 50, "workers": 8, "elapsed_seconds": 5.0, "throughput_tasks_per_sec": 10.0, "succeeded": 50, "failed": 0, "generated_at": "2026-03-13T09:44:00Z"}},
    {"report_path": "docs/reports/soak-local-100x12.json", "result": {"count": 100, "workers": 12, "elapsed_seconds": 10.0, "throughput_tasks_per_sec": 9.6, "succeeded": 100, "failed": 0, "generated_at": "2026-03-13T09:44:20Z"}}
  ]
}`,
		filepath.Join(root, "docs/reports/mixed-workload-matrix-report.json"): `{
  "all_ok": true,
  "generated_at": "2026-03-13T09:44:42.458392Z",
  "tasks": [
    {"name": "browser-a", "ok": true, "expected_executor": "kubernetes", "routed_executor": "kubernetes", "final_state": "succeeded"}
  ]
}`,
		filepath.Join(root, "docs/reports/soak-local-100x12.json"):  `{"count":100,"workers":12,"elapsed_seconds":10.0,"throughput_tasks_per_sec":9.6,"succeeded":100,"failed":0,"generated_at":"2026-03-13T09:44:20Z"}`,
		filepath.Join(root, "docs/reports/soak-local-1000x24.json"): `{"count":1000,"workers":24,"elapsed_seconds":100.0,"throughput_tasks_per_sec":10.0,"succeeded":1000,"failed":0,"generated_at":"2026-03-13T09:44:30Z"}`,
		filepath.Join(root, "docs/reports/soak-local-2000x24.json"): `{"count":2000,"workers":24,"elapsed_seconds":205.0,"throughput_tasks_per_sec":9.75,"succeeded":2000,"failed":0,"generated_at":"2026-03-13T09:44:40Z"}`,
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	report, err := automationBenchmarkCapacityCertification(automationBenchmarkCapacityCertificationOptions{
		GoRoot:                  root,
		OutputPath:              "docs/reports/capacity-certification-matrix.json",
		MarkdownOutputPath:      "docs/reports/capacity-certification-report.md",
		BenchmarkReportPath:     "docs/reports/benchmark-matrix-report.json",
		MixedWorkloadReportPath: "docs/reports/mixed-workload-matrix-report.json",
	})
	if err != nil {
		t.Fatalf("build capacity certification: %v", err)
	}

	summary, ok := report["summary"].(map[string]any)
	if !ok || summary["overall_status"] != "pass" {
		t.Fatalf("expected passing summary, got %+v", report)
	}
	inputs, ok := report["evidence_inputs"].(map[string]any)
	if !ok || inputs["benchmark_report_path"] != "docs/reports/benchmark-matrix-report.json" || inputs["mixed_workload_report_path"] != "docs/reports/mixed-workload-matrix-report.json" {
		t.Fatalf("expected preserved input report paths, got %+v", report)
	}

	markdownBody, err := os.ReadFile(filepath.Join(root, "docs/reports/capacity-certification-report.md"))
	if err != nil {
		t.Fatalf("read capacity certification markdown: %v", err)
	}
	text := string(markdownBody)
	if !strings.Contains(text, "# Capacity Certification Report") || !strings.Contains(text, "Runtime enforcement: `none`") {
		t.Fatalf("unexpected capacity certification markdown: %s", text)
	}
}
