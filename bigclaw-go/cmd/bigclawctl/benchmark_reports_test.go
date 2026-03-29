package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseBenchmarkStdout(t *testing.T) {
	parsed := parseBenchmarkStdout("BenchmarkSchedulerDecide-8   \t16466796\t        73.98 ns/op\nPASS\n")
	if got := parsed["BenchmarkSchedulerDecide-8"]["ns_per_op"]; got != 73.98 {
		t.Fatalf("expected 73.98 ns/op, got %v", got)
	}
}

func TestAutomationBenchmarkRunMatrixWritesReport(t *testing.T) {
	goRoot := t.TempDir()
	reportPath := "docs/reports/benchmark-matrix-report.json"
	report, err := automationBenchmarkRunMatrix(automationBenchmarkMatrixOptions{
		GoRoot:         goRoot,
		ReportPath:     reportPath,
		TimeoutSeconds: 3,
		Scenarios: []benchmarkScenario{
			{Count: 7, Workers: 3},
		},
		BenchmarkRunner: func(string) (string, map[string]map[string]float64, error) {
			return "BenchmarkSchedulerDecide-8\t1\t73.98 ns/op\n", map[string]map[string]float64{
				"BenchmarkSchedulerDecide-8": {"ns_per_op": 73.98},
			}, nil
		},
		SoakRunner: func(opts automationSoakLocalOptions) (*automationSoakLocalReport, int, error) {
			if !opts.Autostart || opts.Count != 7 || opts.Workers != 3 {
				t.Fatalf("unexpected soak options: %+v", opts)
			}
			return &automationSoakLocalReport{
				Count:                 7,
				Workers:               3,
				ElapsedSeconds:        1.4,
				ThroughputTasksPerSec: 5,
				Succeeded:             7,
				Failed:                0,
			}, 0, nil
		},
	})
	if err != nil {
		t.Fatalf("run matrix: %v", err)
	}
	if len(report["soak_matrix"].([]map[string]any)) != 1 {
		t.Fatalf("expected one soak lane, got %+v", report)
	}
	body, err := os.ReadFile(filepath.Join(goRoot, reportPath))
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(body), "\"workers\": 3") {
		t.Fatalf("unexpected report body: %s", string(body))
	}
}

func TestBuildCapacityCertificationReportPassesCheckedInEvidence(t *testing.T) {
	goRoot := filepath.Clean(filepath.Join("..", ".."))
	report, err := buildCapacityCertificationReport(capacityCertificationOptions{
		GoRoot:                      goRoot,
		BenchmarkReportPath:         "docs/reports/benchmark-matrix-report.json",
		MixedWorkloadReportPath:     "docs/reports/mixed-workload-matrix-report.json",
		SupplementalSoakReportPaths: []string{"docs/reports/soak-local-1000x24.json", "docs/reports/soak-local-2000x24.json"},
		GeneratorPath:               "scripts/benchmark/capacity_certification.sh",
	})
	if err != nil {
		t.Fatalf("build report: %v", err)
	}
	summary := getMap(report, "summary")
	if summary["overall_status"] != "pass" {
		t.Fatalf("expected pass summary, got %+v", report)
	}
	if report["generated_at"] != "2026-03-13T09:44:42.458392Z" {
		t.Fatalf("unexpected generated_at: %v", report["generated_at"])
	}
	if !strings.Contains(report["markdown"].(string), "## Admission Policy Summary") {
		t.Fatalf("missing markdown section: %v", report["markdown"])
	}
}
