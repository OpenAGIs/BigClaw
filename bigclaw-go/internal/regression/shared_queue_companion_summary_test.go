package regression

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestSharedQueueCompanionSummaryStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	summaryPath := filepath.Join(repoRoot, "docs", "reports", "shared-queue-companion-summary.json")
	bundleSummaryPath := filepath.Join(repoRoot, "docs", "reports", "live-validation-runs", "20260316T140138Z", "shared-queue-companion-summary.json")
	liveSummaryPath := filepath.Join(repoRoot, "docs", "reports", "live-validation-summary.json")

	var summary struct {
		Available            bool           `json:"available"`
		CanonicalReportPath  string         `json:"canonical_report_path"`
		CanonicalSummaryPath string         `json:"canonical_summary_path"`
		BundleReportPath     string         `json:"bundle_report_path"`
		BundleSummaryPath    string         `json:"bundle_summary_path"`
		Status               string         `json:"status"`
		GeneratedAt          string         `json:"generated_at"`
		Count                int            `json:"count"`
		CrossNodeCompletions int            `json:"cross_node_completions"`
		DuplicateStarted     int            `json:"duplicate_started_tasks"`
		DuplicateCompleted   int            `json:"duplicate_completed_tasks"`
		MissingCompleted     int            `json:"missing_completed_tasks"`
		SubmittedByNode      map[string]int `json:"submitted_by_node"`
		CompletedByNode      map[string]int `json:"completed_by_node"`
		Nodes                []string       `json:"nodes"`
	}
	readJSONFile(t, summaryPath, &summary)

	if !summary.Available || summary.CanonicalReportPath != "docs/reports/multi-node-shared-queue-report.json" || summary.CanonicalSummaryPath != "docs/reports/shared-queue-companion-summary.json" {
		t.Fatalf("unexpected shared queue canonical paths: %+v", summary)
	}
	if summary.BundleReportPath != "docs/reports/live-validation-runs/20260316T140138Z/multi-node-shared-queue-report.json" || summary.BundleSummaryPath != "docs/reports/live-validation-runs/20260316T140138Z/shared-queue-companion-summary.json" || summary.Status != "succeeded" {
		t.Fatalf("unexpected shared queue bundle paths/status: %+v", summary)
	}
	if summary.GeneratedAt != "2026-03-13T09:45:19Z" || summary.Count != 200 || summary.CrossNodeCompletions != 99 || summary.DuplicateStarted != 0 || summary.DuplicateCompleted != 0 || summary.MissingCompleted != 0 {
		t.Fatalf("unexpected shared queue counts: %+v", summary)
	}
	if len(summary.Nodes) != 2 || summary.Nodes[0] != "node-a" || summary.Nodes[1] != "node-b" {
		t.Fatalf("unexpected shared queue nodes: %+v", summary.Nodes)
	}
	if summary.SubmittedByNode["node-a"] != 100 || summary.SubmittedByNode["node-b"] != 100 || summary.CompletedByNode["node-a"] != 73 || summary.CompletedByNode["node-b"] != 127 {
		t.Fatalf("unexpected shared queue per-node counts: submitted=%+v completed=%+v", summary.SubmittedByNode, summary.CompletedByNode)
	}

	var bundleSummary struct {
		Available            bool           `json:"available"`
		CanonicalReportPath  string         `json:"canonical_report_path"`
		CanonicalSummaryPath string         `json:"canonical_summary_path"`
		BundleReportPath     string         `json:"bundle_report_path"`
		BundleSummaryPath    string         `json:"bundle_summary_path"`
		Status               string         `json:"status"`
		GeneratedAt          string         `json:"generated_at"`
		Count                int            `json:"count"`
		CrossNodeCompletions int            `json:"cross_node_completions"`
		DuplicateStarted     int            `json:"duplicate_started_tasks"`
		DuplicateCompleted   int            `json:"duplicate_completed_tasks"`
		MissingCompleted     int            `json:"missing_completed_tasks"`
		SubmittedByNode      map[string]int `json:"submitted_by_node"`
		CompletedByNode      map[string]int `json:"completed_by_node"`
		Nodes                []string       `json:"nodes"`
	}
	readJSONFile(t, bundleSummaryPath, &bundleSummary)

	if !reflect.DeepEqual(bundleSummary, summary) {
		t.Fatalf("bundle summary drift: %+v vs %+v", bundleSummary, summary)
	}

	var liveSummary struct {
		SharedQueueCompanion struct {
			Available            bool           `json:"available"`
			CanonicalReportPath  string         `json:"canonical_report_path"`
			CanonicalSummaryPath string         `json:"canonical_summary_path"`
			BundleReportPath     string         `json:"bundle_report_path"`
			BundleSummaryPath    string         `json:"bundle_summary_path"`
			Status               string         `json:"status"`
			Count                int            `json:"count"`
			CrossNodeCompletions int            `json:"cross_node_completions"`
			DuplicateStarted     int            `json:"duplicate_started_tasks"`
			DuplicateCompleted   int            `json:"duplicate_completed_tasks"`
			MissingCompleted     int            `json:"missing_completed_tasks"`
			SubmittedByNode      map[string]int `json:"submitted_by_node"`
			CompletedByNode      map[string]int `json:"completed_by_node"`
			Nodes                []string       `json:"nodes"`
		} `json:"shared_queue_companion"`
	}
	readJSONFile(t, liveSummaryPath, &liveSummary)

	if liveSummary.SharedQueueCompanion.Available != summary.Available ||
		liveSummary.SharedQueueCompanion.CanonicalReportPath != summary.CanonicalReportPath ||
		liveSummary.SharedQueueCompanion.CanonicalSummaryPath != summary.CanonicalSummaryPath ||
		liveSummary.SharedQueueCompanion.BundleReportPath != summary.BundleReportPath ||
		liveSummary.SharedQueueCompanion.BundleSummaryPath != summary.BundleSummaryPath ||
		liveSummary.SharedQueueCompanion.Status != summary.Status ||
		liveSummary.SharedQueueCompanion.Count != summary.Count ||
		liveSummary.SharedQueueCompanion.CrossNodeCompletions != summary.CrossNodeCompletions ||
		liveSummary.SharedQueueCompanion.DuplicateStarted != summary.DuplicateStarted ||
		liveSummary.SharedQueueCompanion.DuplicateCompleted != summary.DuplicateCompleted ||
		liveSummary.SharedQueueCompanion.MissingCompleted != summary.MissingCompleted {
		t.Fatalf("live validation shared queue drift: %+v vs %+v", liveSummary.SharedQueueCompanion, summary)
	}
	if len(liveSummary.SharedQueueCompanion.Nodes) != len(summary.Nodes) || liveSummary.SharedQueueCompanion.Nodes[0] != summary.Nodes[0] || liveSummary.SharedQueueCompanion.Nodes[1] != summary.Nodes[1] {
		t.Fatalf("live validation shared queue nodes drift: %+v vs %+v", liveSummary.SharedQueueCompanion.Nodes, summary.Nodes)
	}
	if liveSummary.SharedQueueCompanion.SubmittedByNode["node-a"] != summary.SubmittedByNode["node-a"] ||
		liveSummary.SharedQueueCompanion.SubmittedByNode["node-b"] != summary.SubmittedByNode["node-b"] ||
		liveSummary.SharedQueueCompanion.CompletedByNode["node-a"] != summary.CompletedByNode["node-a"] ||
		liveSummary.SharedQueueCompanion.CompletedByNode["node-b"] != summary.CompletedByNode["node-b"] {
		t.Fatalf("live validation shared queue per-node drift: submitted=%+v/%+v completed=%+v/%+v", liveSummary.SharedQueueCompanion.SubmittedByNode, summary.SubmittedByNode, liveSummary.SharedQueueCompanion.CompletedByNode, summary.CompletedByNode)
	}
}
