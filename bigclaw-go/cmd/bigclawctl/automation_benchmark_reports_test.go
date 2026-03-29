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
	reportPath := "docs/reports/benchmark-matrix-report.json"

	report, err := automationRunMatrix(automationRunMatrixOptions{
		GoRoot:         goRoot,
		ReportPath:     reportPath,
		TimeoutSeconds: 30,
		Scenarios:      []string{"5:2", "9:3"},
		RunBenchmarks: func(string) (string, map[string]map[string]float64, error) {
			stdout := "BenchmarkMemoryQueueEnqueueLease-8 100 321.0 ns/op\nBenchmarkSchedulerDecide-8 200 99.0 ns/op\n"
			return stdout, automationParseBenchmarkStdout(stdout), nil
		},
		RunSoakScenario: func(opts automationSoakLocalOptions) (*automationSoakLocalReport, int, error) {
			return &automationSoakLocalReport{
				Count:                 opts.Count,
				Workers:               opts.Workers,
				ElapsedSeconds:        2,
				ThroughputTasksPerSec: float64(opts.Count) / 2,
				Succeeded:             opts.Count,
				Failed:                0,
			}, 0, nil
		},
	})
	if err != nil {
		t.Fatalf("run matrix: %v", err)
	}
	if len(report.SoakMatrix) != 2 {
		t.Fatalf("expected 2 soak scenarios, got %d", len(report.SoakMatrix))
	}
	if report.SoakMatrix[0].ReportPath != "docs/reports/soak-local-5x2.json" {
		t.Fatalf("unexpected first report path: %+v", report.SoakMatrix[0])
	}
	if report.Benchmark.Parsed["BenchmarkSchedulerDecide-8"]["ns_per_op"] != 99 {
		t.Fatalf("unexpected benchmark parse: %+v", report.Benchmark.Parsed)
	}

	body, err := os.ReadFile(filepath.Join(goRoot, reportPath))
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(body), "\"report_path\": \"docs/reports/soak-local-9x3.json\"") {
		t.Fatalf("unexpected report body: %s", string(body))
	}
}

func TestAutomationBuildCapacityCertificationReportPassesCheckedInEvidence(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "capacity-certification-matrix.json")
	markdownPath := filepath.Join(t.TempDir(), "capacity-certification-report.md")

	report, err := automationBuildCapacityCertificationReport(automationCapacityCertificationOptions{
		RepoRoot:           repoRootFromWorkingDir(),
		OutputPath:         outputPath,
		MarkdownOutputPath: markdownPath,
	})
	if err != nil {
		t.Fatalf("build report: %v", err)
	}

	if report.Summary.OverallStatus != "pass" {
		t.Fatalf("unexpected overall status: %+v", report.Summary)
	}
	if len(report.Summary.FailedLanes) != 0 {
		t.Fatalf("unexpected failed lanes: %+v", report.Summary.FailedLanes)
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
	if !strings.Contains(report.Markdown, "## Admission Policy Summary") || !strings.Contains(report.Markdown, "Runtime enforcement: `none`") {
		t.Fatalf("unexpected markdown: %s", report.Markdown)
	}
	if report.EvidenceInputs.GeneratorScript != "bigclaw-go/cmd/bigclawctl automation benchmark capacity-certification" {
		t.Fatalf("unexpected generator script: %+v", report.EvidenceInputs)
	}

	found1000x24 := false
	for _, lane := range report.SoakMatrix {
		if lane.Lane == "1000x24" {
			found1000x24 = true
		}
	}
	if !found1000x24 {
		t.Fatalf("missing 1000x24 lane: %+v", report.SoakMatrix)
	}

	body, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read json output: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode json output: %v", err)
	}
	if payload["ticket"] != "BIG-PAR-098" {
		t.Fatalf("unexpected output payload: %+v", payload)
	}

	markdown, err := os.ReadFile(markdownPath)
	if err != nil {
		t.Fatalf("read markdown output: %v", err)
	}
	if !strings.Contains(string(markdown), "# Capacity Certification Report") {
		t.Fatalf("unexpected markdown output: %s", string(markdown))
	}
}
