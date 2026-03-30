package regression

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCapacityCertificationScriptMatchesCheckedInEvidence(t *testing.T) {
	repoRoot := repoRoot(t)
	tmpDir := t.TempDir()
	jsonOutput := filepath.Join(tmpDir, "capacity-certification-matrix.json")
	markdownOutput := filepath.Join(tmpDir, "capacity-certification-report.md")
	scriptPath := filepath.Join(repoRoot, "scripts", "benchmark", "capacity_certification.py")

	cmd := exec.Command("python3", scriptPath, "--output", jsonOutput, "--markdown-output", markdownOutput)
	cmd.Dir = repoRoot
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run capacity certification script: %v\n%s", err, output)
	}

	var report struct {
		GeneratedAt string `json:"generated_at"`
		Summary     struct {
			OverallStatus string   `json:"overall_status"`
			FailedLanes   []string `json:"failed_lanes"`
		} `json:"summary"`
		SaturationIndicator struct {
			Status string `json:"status"`
		} `json:"saturation_indicator"`
		MixedWorkload struct {
			Status string `json:"status"`
		} `json:"mixed_workload"`
		SoakMatrix []struct {
			Lane string `json:"lane"`
		} `json:"soak_matrix"`
	}
	readJSONFile(t, jsonOutput, &report)

	if report.Summary.OverallStatus != "pass" {
		t.Fatalf("unexpected overall_status: %+v", report.Summary)
	}
	if len(report.Summary.FailedLanes) != 0 {
		t.Fatalf("expected no failed lanes, got %+v", report.Summary.FailedLanes)
	}
	if report.SaturationIndicator.Status != "pass" {
		t.Fatalf("unexpected saturation indicator: %+v", report.SaturationIndicator)
	}
	if report.MixedWorkload.Status != "pass" {
		t.Fatalf("unexpected mixed workload status: %+v", report.MixedWorkload)
	}

	found1000x24 := false
	for _, lane := range report.SoakMatrix {
		if lane.Lane == "1000x24" {
			found1000x24 = true
			break
		}
	}
	if !found1000x24 {
		t.Fatalf("expected 1000x24 soak lane, got %+v", report.SoakMatrix)
	}
	if report.GeneratedAt != "2026-03-13T09:44:42.458392Z" {
		t.Fatalf("unexpected generated_at: %s", report.GeneratedAt)
	}

	markdown, err := os.ReadFile(markdownOutput)
	if err != nil {
		t.Fatalf("read markdown output: %v", err)
	}
	if !bytes.Contains(markdown, []byte("## Admission Policy Summary")) {
		t.Fatalf("markdown missing admission policy summary: %s", markdown)
	}
	if !strings.Contains(string(markdown), "Runtime enforcement: `none`") {
		t.Fatalf("markdown missing runtime enforcement note: %s", markdown)
	}
}

func TestCapacityCertificationCheckedInArtifactStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	reportPath := filepath.Join(repoRoot, "docs", "reports", "capacity-certification-matrix.json")

	var report struct {
		GeneratedAt string `json:"generated_at"`
		Summary     struct {
			OverallStatus string   `json:"overall_status"`
			FailedLanes   []string `json:"failed_lanes"`
		} `json:"summary"`
		SaturationIndicator struct {
			Status string `json:"status"`
		} `json:"saturation_indicator"`
		MixedWorkload struct {
			Status string `json:"status"`
		} `json:"mixed_workload"`
		EvidenceInputs struct {
			GeneratorScript string `json:"generator_script"`
		} `json:"evidence_inputs"`
		SoakMatrix []struct {
			Lane string `json:"lane"`
		} `json:"soak_matrix"`
	}
	readJSONFile(t, reportPath, &report)

	if report.Summary.OverallStatus != "pass" || len(report.Summary.FailedLanes) != 0 {
		t.Fatalf("unexpected checked-in summary: %+v", report.Summary)
	}
	if report.SaturationIndicator.Status != "pass" || report.MixedWorkload.Status != "pass" {
		t.Fatalf("unexpected checked-in statuses: saturation=%+v mixed=%+v", report.SaturationIndicator, report.MixedWorkload)
	}
	if report.EvidenceInputs.GeneratorScript != "bigclaw-go/scripts/benchmark/capacity_certification.py" {
		t.Fatalf("unexpected generator script: %s", report.EvidenceInputs.GeneratorScript)
	}
	if report.GeneratedAt != "2026-03-13T09:44:42.458392Z" {
		t.Fatalf("unexpected checked-in generated_at: %s", report.GeneratedAt)
	}

	found1000x24 := false
	for _, lane := range report.SoakMatrix {
		if lane.Lane == "1000x24" {
			found1000x24 = true
			break
		}
	}
	if !found1000x24 {
		payload, _ := json.Marshal(report.SoakMatrix)
		t.Fatalf("expected checked-in 1000x24 lane, got %s", payload)
	}
}
