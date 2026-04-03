package regression

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestParallelValidationEvidenceBundleStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	reportPath := filepath.Join(repoRoot, "docs", "reports", "parallel-validation-evidence-bundle.json")

	var report struct {
		Ticket  string `json:"ticket"`
		Status  string `json:"status"`
		RunID   string `json:"run_id"`
		Summary struct {
			LaneCount        int `json:"lane_count"`
			EnabledLaneCount int `json:"enabled_lane_count"`
			SucceededCount   int `json:"succeeded_lane_count"`
			FailingCount     int `json:"failing_lane_count"`
			SkippedCount     int `json:"skipped_lane_count"`
			RootCauseCount   int `json:"root_cause_count"`
		} `json:"summary"`
		EvidenceInputs struct {
			LiveValidationSummary string   `json:"live_validation_summary"`
			LiveValidationIndex   string   `json:"live_validation_index"`
			Executors             []string `json:"executors"`
		} `json:"evidence_inputs"`
		ValidationMatrix []struct {
			Lane               string `json:"lane"`
			Executor           string `json:"executor"`
			Status             string `json:"status"`
			BundleReportPath   string `json:"bundle_report_path"`
			RootCauseEventType string `json:"root_cause_event_type"`
			RootCauseLocation  string `json:"root_cause_location"`
		} `json:"validation_matrix"`
		Lanes []struct {
			Lane             string `json:"lane"`
			Executor         string `json:"executor"`
			Status           string `json:"status"`
			BundleReportPath string `json:"bundle_report_path"`
			FailureRootCause struct {
				Status    string `json:"status"`
				EventType string `json:"event_type"`
				Location  string `json:"location"`
			} `json:"failure_root_cause"`
		} `json:"lanes"`
	}
	readJSONFile(t, reportPath, &report)

	if report.Ticket != "BIGCLAW-173" || report.Status != "succeeded" || report.RunID != "20260316T140138Z" {
		t.Fatalf("unexpected parallel evidence metadata: %+v", report)
	}
	if report.Summary.LaneCount != 3 || report.Summary.EnabledLaneCount != 3 || report.Summary.SucceededCount != 3 || report.Summary.FailingCount != 0 || report.Summary.SkippedCount != 0 || report.Summary.RootCauseCount != 3 {
		t.Fatalf("unexpected parallel evidence summary: %+v", report.Summary)
	}
	if report.EvidenceInputs.LiveValidationSummary != "docs/reports/live-validation-runs/20260316T140138Z/summary.json" || report.EvidenceInputs.LiveValidationIndex != "docs/reports/live-validation-index.md" {
		t.Fatalf("unexpected evidence inputs: %+v", report.EvidenceInputs)
	}
	if len(report.EvidenceInputs.Executors) != 3 || report.EvidenceInputs.Executors[0] != "local" || report.EvidenceInputs.Executors[1] != "kubernetes" || report.EvidenceInputs.Executors[2] != "ray" {
		t.Fatalf("unexpected executors: %+v", report.EvidenceInputs.Executors)
	}
	if len(report.ValidationMatrix) != 3 || report.ValidationMatrix[0].Lane != "local" || report.ValidationMatrix[1].Lane != "k8s" || report.ValidationMatrix[2].Lane != "ray" {
		t.Fatalf("unexpected validation matrix lanes: %+v", report.ValidationMatrix)
	}
	for _, row := range report.ValidationMatrix {
		if row.Status != "succeeded" || row.BundleReportPath == "" || row.RootCauseLocation == "" {
			t.Fatalf("unexpected validation matrix row: %+v", row)
		}
	}
	if len(report.Lanes) != 3 || report.Lanes[1].FailureRootCause.Status != "not_triggered" || report.Lanes[1].FailureRootCause.Location == "" {
		t.Fatalf("unexpected lane details: %+v", report.Lanes)
	}

	body := readRepoFile(t, repoRoot, "docs/reports/parallel-validation-evidence-bundle.md")
	for _, needle := range []string{
		"# Parallel Validation Evidence Bundle",
		"Lane `k8s` executor=`kubernetes` status=`succeeded` enabled=`true`",
		"Lane `k8s` root cause: event=`task.completed`",
		"### ray",
		"Failure root cause: status=`not_triggered` event=`task.completed`",
	} {
		if !strings.Contains(body, needle) {
			t.Fatalf("parallel validation evidence markdown missing substring %q", needle)
		}
	}
}
