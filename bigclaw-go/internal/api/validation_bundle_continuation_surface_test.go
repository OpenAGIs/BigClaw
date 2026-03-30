package api

import (
	"path/filepath"
	"testing"
	"time"
)

func TestBuildValidationBundleContinuationGateReportHoldsForPartialLaneHistory(t *testing.T) {
	scorecard := validationBundleContinuationGateDocument{}
	scorecard.Summary.LatestRunID = "synthetic-run"
	scorecard.Summary.LatestBundleAgeHours = 1.5
	scorecard.Summary.RecentBundleCount = 2
	scorecard.Summary.LatestAllExecutorTracksSucceeded = true
	scorecard.Summary.RecentBundleChainHasNoFailures = true
	scorecard.Summary.AllExecutorTracksHaveRepeatedRecentCoverage = false
	scorecard.SharedQueueCompanion.Available = true
	scorecard.SharedQueueCompanion.CrossNodeCompletions = 99
	scorecard.SharedQueueCompanion.Mode = "standalone-proof"
	scorecard.SharedQueueCompanion.ReportPath = "bigclaw-go/docs/reports/multi-node-shared-queue-report.json"

	report := buildValidationBundleContinuationGateReport(scorecard, validationBundleContinuationGateConfig{
		MaxLatestAgeHours:           72,
		MinRecentBundles:            2,
		RequireRepeatedLaneCoverage: true,
		Now:                         func() time.Time { return time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC) },
	})

	if report.Status != "policy-hold" || report.Recommendation != "hold" {
		t.Fatalf("expected hold report, got %+v", report)
	}
	if len(report.FailingChecks) != 1 || report.FailingChecks[0] != "repeated_lane_coverage_meets_policy" {
		t.Fatalf("expected repeated lane coverage failure, got %+v", report.FailingChecks)
	}
	if report.Summary.FailingCheckCount != 1 {
		t.Fatalf("expected one failing check, got %+v", report.Summary)
	}
}

func TestBuildValidationBundleContinuationGateReportAllowsPartialLaneHistory(t *testing.T) {
	scorecard := validationBundleContinuationGateDocument{}
	scorecard.Summary.LatestRunID = "synthetic-run"
	scorecard.Summary.LatestBundleAgeHours = 1.5
	scorecard.Summary.RecentBundleCount = 2
	scorecard.Summary.LatestAllExecutorTracksSucceeded = true
	scorecard.Summary.RecentBundleChainHasNoFailures = true
	scorecard.Summary.AllExecutorTracksHaveRepeatedRecentCoverage = false
	scorecard.SharedQueueCompanion.Available = true
	scorecard.SharedQueueCompanion.CrossNodeCompletions = 99
	scorecard.SharedQueueCompanion.Mode = "standalone-proof"
	scorecard.SharedQueueCompanion.ReportPath = "bigclaw-go/docs/reports/multi-node-shared-queue-report.json"

	report := buildValidationBundleContinuationGateReport(scorecard, validationBundleContinuationGateConfig{
		MaxLatestAgeHours:           72,
		MinRecentBundles:            2,
		RequireRepeatedLaneCoverage: false,
		Now:                         func() time.Time { return time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC) },
	})

	if report.Status != "policy-go" || report.Recommendation != "go" {
		t.Fatalf("expected go report, got %+v", report)
	}
	if len(report.FailingChecks) != 0 {
		t.Fatalf("expected no failures, got %+v", report.FailingChecks)
	}
}

func TestCheckedInValidationBundleContinuationPolicyGateMatchesExpectedShape(t *testing.T) {
	scorecardPath := filepath.Join("..", "..", "docs", "reports", "validation-bundle-continuation-scorecard.json")
	scorecard, err := loadValidationBundleContinuationScorecard(scorecardPath)
	if err != nil {
		t.Fatalf("load scorecard: %v", err)
	}
	report := buildValidationBundleContinuationGateReport(scorecard, validationBundleContinuationGateConfig{
		MaxLatestAgeHours:           72,
		MinRecentBundles:            2,
		RequireRepeatedLaneCoverage: true,
		Now:                         func() time.Time { return time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC) },
	})

	if report.Status != "policy-go" || report.Recommendation != "go" {
		t.Fatalf("expected checked-in report to be go, got %+v", report)
	}
	if report.Summary.LatestRunID != "20260316T140138Z" {
		t.Fatalf("expected latest run id 20260316T140138Z, got %+v", report.Summary)
	}
	if len(report.FailingChecks) != 0 {
		t.Fatalf("expected no failing checks, got %+v", report.FailingChecks)
	}
}

func TestCheckedInValidationBundleContinuationGateDocumentMatchesExpectedShape(t *testing.T) {
	gatePath := filepath.Join("..", "..", "docs", "reports", "validation-bundle-continuation-policy-gate.json")
	report, err := loadValidationBundleContinuationGateDocument(gatePath)
	if err != nil {
		t.Fatalf("load gate document: %v", err)
	}
	if report.Status != "policy-go" || report.Recommendation != "go" {
		t.Fatalf("expected checked-in gate document to be go, got %+v", report)
	}
	if report.Summary.LatestRunID != "20260316T140138Z" {
		t.Fatalf("expected latest run id 20260316T140138Z, got %+v", report.Summary)
	}
	if len(report.FailingChecks) != 0 {
		t.Fatalf("expected no failing checks, got %+v", report.FailingChecks)
	}
}
