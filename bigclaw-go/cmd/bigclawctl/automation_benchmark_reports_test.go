package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAutomationRunMatrixWritesReport(t *testing.T) {
	goRoot := t.TempDir()
	report, err := automationRunMatrix(automationRunMatrixOptions{
		GoRoot:         goRoot,
		ReportPath:     "docs/reports/benchmark-matrix-report.json",
		TimeoutSeconds: 30,
		Scenarios:      []string{"3:2"},
		RunBenchmarks: func(string) (string, error) {
			return "BenchmarkSchedulerDecide-8    1  73.98 ns/op\n", nil
		},
		RunSoak: func(opts automationSoakLocalOptions) (*automationSoakLocalReport, int, error) {
			if opts.Count != 3 || opts.Workers != 2 || !opts.Autostart {
				t.Fatalf("unexpected soak opts: %+v", opts)
			}
			return &automationSoakLocalReport{
				Count:                 3,
				Workers:               2,
				ElapsedSeconds:        1.5,
				ThroughputTasksPerSec: 2,
				Succeeded:             3,
				Failed:                0,
			}, 0, nil
		},
	})
	if err != nil {
		t.Fatalf("run matrix: %v", err)
	}
	if got := report.Benchmark.Parsed["BenchmarkSchedulerDecide-8"].NSPerOp; got != 73.98 {
		t.Fatalf("unexpected benchmark parse: %+v", report.Benchmark.Parsed)
	}
	body, err := os.ReadFile(filepath.Join(goRoot, "docs/reports/benchmark-matrix-report.json"))
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	soakMatrix, _ := payload["soak_matrix"].([]any)
	if len(soakMatrix) != 1 {
		t.Fatalf("unexpected soak matrix: %+v", payload)
	}
}

func TestAutomationCapacityCertificationMatchesCheckedInEvidence(t *testing.T) {
	goRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	report, markdown, err := automationCapacityCertification(automationCapacityCertificationOptions{
		GoRoot:                  goRoot,
		BenchmarkReportPath:     "docs/reports/benchmark-matrix-report.json",
		MixedWorkloadReportPath: "docs/reports/mixed-workload-matrix-report.json",
		OutputPath:              filepath.Join(t.TempDir(), "capacity-certification-matrix.json"),
		MarkdownOutputPath:      filepath.Join(t.TempDir(), "capacity-certification-report.md"),
	})
	if err != nil {
		t.Fatalf("capacity certification: %v", err)
	}
	if got := report.Summary["overall_status"]; got != "pass" {
		t.Fatalf("unexpected overall status: %+v", report.Summary)
	}
	failedLanes, _ := report.Summary["failed_lanes"].([]string)
	if len(failedLanes) != 0 {
		t.Fatalf("unexpected failed lanes: %+v", report.Summary)
	}
	if report.SaturationIndicator.Status != "pass" {
		t.Fatalf("unexpected saturation indicator: %+v", report.SaturationIndicator)
	}
	if report.MixedWorkload.Status != "pass" {
		t.Fatalf("unexpected mixed workload status: %+v", report.MixedWorkload)
	}
	if report.GeneratedAt != "2026-03-13T09:44:42.458392Z" {
		t.Fatalf("unexpected generated_at: %s", report.GeneratedAt)
	}
	if report.EvidenceInputs["generator_script"] != "go run ./cmd/bigclawctl automation benchmark capacity-certification" {
		t.Fatalf("unexpected generator script: %+v", report.EvidenceInputs)
	}
	if !strings.Contains(markdown, "## Admission Policy Summary") || !strings.Contains(markdown, "Runtime enforcement: `none`") {
		t.Fatalf("unexpected markdown: %s", markdown)
	}
}
