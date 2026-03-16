package regression

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type telemetryEvidencePack struct {
	Ticket         string `json:"ticket"`
	Track          string `json:"track"`
	Status         string `json:"status"`
	EvidenceInputs struct {
		ObservabilityReport string   `json:"observability_report"`
		FollowUpDigest      string   `json:"follow_up_digest"`
		ReviewReadiness     string   `json:"review_readiness"`
		IssueCoverage       string   `json:"issue_coverage"`
		CodePaths           []string `json:"code_paths"`
		ValidationTests     []string `json:"validation_tests"`
	} `json:"evidence_inputs"`
	CurrentControls struct {
		SamplingPolicy struct {
			State string `json:"state"`
		} `json:"sampling_policy"`
		CardinalityControls struct {
			State                   string   `json:"state"`
			BlockedMetricDimensions []string `json:"blocked_metric_dimensions"`
		} `json:"cardinality_controls"`
	} `json:"current_controls"`
	ReviewerPosture struct {
		CurrentRuntimePosture string   `json:"current_runtime_posture"`
		IntendedProduction    []string `json:"intended_production_posture"`
	} `json:"reviewer_posture"`
}

func TestTelemetrySamplingCardinalityEvidencePack(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	reportPath := filepath.Join(repoRoot, "docs", "reports", "telemetry-sampling-cardinality-evidence-pack.json")
	contents, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read telemetry evidence pack: %v", err)
	}

	var report telemetryEvidencePack
	if err := json.Unmarshal(contents, &report); err != nil {
		t.Fatalf("decode telemetry evidence pack: %v", err)
	}

	if report.Ticket != "OPE-265" || report.Track != "BIG-PAR-095" || report.Status != "repo-evidence-pack" {
		t.Fatalf("unexpected report identity: %+v", report)
	}
	if report.CurrentControls.SamplingPolicy.State != "implicit_full_capture" {
		t.Fatalf("expected implicit sampling posture, got %+v", report.CurrentControls.SamplingPolicy)
	}
	if report.CurrentControls.CardinalityControls.State != "low_cardinality_metrics_only" {
		t.Fatalf("expected low-cardinality metrics posture, got %+v", report.CurrentControls.CardinalityControls)
	}
	if !contains(report.CurrentControls.CardinalityControls.BlockedMetricDimensions, "trace_id") || !contains(report.CurrentControls.CardinalityControls.BlockedMetricDimensions, "task_id") {
		t.Fatalf("expected trace_id and task_id to remain blocked metric dimensions, got %+v", report.CurrentControls.CardinalityControls.BlockedMetricDimensions)
	}
	if !strings.Contains(report.ReviewerPosture.CurrentRuntimePosture, "local diagnostics and rollout evidence") {
		t.Fatalf("expected explicit non-production posture, got %q", report.ReviewerPosture.CurrentRuntimePosture)
	}
	if len(report.ReviewerPosture.IntendedProduction) < 3 {
		t.Fatalf("expected intended production posture details, got %+v", report.ReviewerPosture.IntendedProduction)
	}

	requiredPaths := []string{
		report.EvidenceInputs.ObservabilityReport,
		report.EvidenceInputs.FollowUpDigest,
		report.EvidenceInputs.ReviewReadiness,
		report.EvidenceInputs.IssueCoverage,
	}
	requiredPaths = append(requiredPaths, report.EvidenceInputs.CodePaths...)
	requiredPaths = append(requiredPaths, report.EvidenceInputs.ValidationTests...)
	for _, candidate := range requiredPaths {
		if candidate == "" {
			t.Fatal("evidence pack contains an empty path")
		}
		if _, err := os.Stat(resolveRepoPath(repoRoot, candidate)); err != nil {
			t.Fatalf("expected referenced path %q to exist: %v", candidate, err)
		}
	}

	docsToCheck := []string{
		report.EvidenceInputs.ObservabilityReport,
		report.EvidenceInputs.FollowUpDigest,
		report.EvidenceInputs.ReviewReadiness,
		report.EvidenceInputs.IssueCoverage,
	}
	for _, candidate := range docsToCheck {
		doc, err := os.ReadFile(resolveRepoPath(repoRoot, candidate))
		if err != nil {
			t.Fatalf("read linked doc %q: %v", candidate, err)
		}
		body := string(doc)
		if !strings.Contains(body, "telemetry-sampling-cardinality-evidence-pack.json") {
			t.Fatalf("expected %q to reference the telemetry evidence pack", candidate)
		}
	}

	digestBody, err := os.ReadFile(resolveRepoPath(repoRoot, report.EvidenceInputs.FollowUpDigest))
	if err != nil {
		t.Fatalf("read follow-up digest: %v", err)
	}
	if !strings.Contains(string(digestBody), "BIG-PAR-095") {
		t.Fatalf("expected follow-up digest to use the current Linear track id, got %s", string(digestBody))
	}
}

func resolveRepoPath(repoRoot, candidate string) string {
	return filepath.Join(repoRoot, strings.TrimPrefix(candidate, "bigclaw-go/"))
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
