package regression

import (
	"encoding/json"
	"testing"
)

func TestLiveValidationIndexSummaryPointers(t *testing.T) {
	repoRoot := repoRoot(t)
	contents := readRepoFile(t, repoRoot, "docs/reports/live-validation-index.json")

	var payload struct {
		Latest struct {
			Broker struct {
				BundleSummaryPath    string `json:"bundle_summary_path"`
				CanonicalSummaryPath string `json:"canonical_summary_path"`
			} `json:"broker"`
			SharedQueueCompanion struct {
				BundleReportPath     string `json:"bundle_report_path"`
				BundleSummaryPath    string `json:"bundle_summary_path"`
				CanonicalReportPath  string `json:"canonical_report_path"`
				CanonicalSummaryPath string `json:"canonical_summary_path"`
			} `json:"shared_queue_companion"`
		} `json:"latest"`
	}

	if err := json.Unmarshal([]byte(contents), &payload); err != nil {
		t.Fatalf("parse live validation index: %v", err)
	}

	broker := payload.Latest.Broker
	if broker.BundleSummaryPath != "docs/reports/live-validation-runs/20260316T140138Z/broker-validation-summary.json" {
		t.Fatalf("unexpected broker bundle summary path: %s", broker.BundleSummaryPath)
	}
	if broker.CanonicalSummaryPath != "docs/reports/broker-validation-summary.json" {
		t.Fatalf("unexpected broker canonical summary path: %s", broker.CanonicalSummaryPath)
	}

	shared := payload.Latest.SharedQueueCompanion
	if shared.BundleReportPath != "docs/reports/live-validation-runs/20260316T140138Z/multi-node-shared-queue-report.json" {
		t.Fatalf("unexpected shared queue bundle report path: %s", shared.BundleReportPath)
	}
	if shared.CanonicalReportPath != "docs/reports/multi-node-shared-queue-report.json" {
		t.Fatalf("unexpected shared queue canonical report path: %s", shared.CanonicalReportPath)
	}
	if shared.BundleSummaryPath != "docs/reports/live-validation-runs/20260316T140138Z/shared-queue-companion-summary.json" {
		t.Fatalf("unexpected shared queue bundle summary path: %s", shared.BundleSummaryPath)
	}
	if shared.CanonicalSummaryPath != "docs/reports/shared-queue-companion-summary.json" {
		t.Fatalf("unexpected shared queue canonical summary path: %s", shared.CanonicalSummaryPath)
	}
}
