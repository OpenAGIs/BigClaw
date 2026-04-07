package regression

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type sharedQueueCompanionSummary struct {
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

func TestSharedQueueCompanionSummaryStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	canonicalPath := filepath.Join(repoRoot, "docs", "reports", "shared-queue-companion-summary.json")
	bundlePath := filepath.Join(repoRoot, "docs", "reports", "live-validation-runs", "20260316T140138Z", "shared-queue-companion-summary.json")

	var canonical sharedQueueCompanionSummary
	readJSONFile(t, canonicalPath, &canonical)
	assertSharedQueueCompanionSummary(t, canonicalPath, canonical)

	var bundled sharedQueueCompanionSummary
	readJSONFile(t, bundlePath, &bundled)
	assertSharedQueueCompanionSummary(t, bundlePath, bundled)

	if !reflect.DeepEqual(canonical, bundled) {
		t.Fatalf("canonical and bundled companion summaries drifted: canonical=%+v bundled=%+v", canonical, bundled)
	}

	for _, candidate := range []string{canonical.CanonicalReportPath, canonical.BundleReportPath} {
		if _, err := os.Stat(resolveRepoPath(repoRoot, candidate)); err != nil {
			t.Fatalf("expected referenced report %q to exist: %v", candidate, err)
		}
	}
}

func assertSharedQueueCompanionSummary(t *testing.T, path string, summary sharedQueueCompanionSummary) {
	t.Helper()

	if !summary.Available || summary.Status != "succeeded" || summary.GeneratedAt != "2026-03-13T09:45:19Z" {
		t.Fatalf("%s unexpected availability metadata: %+v", path, summary)
	}
	if summary.CanonicalReportPath != "docs/reports/multi-node-shared-queue-report.json" || summary.CanonicalSummaryPath != "docs/reports/shared-queue-companion-summary.json" {
		t.Fatalf("%s unexpected canonical pointers: %+v", path, summary)
	}
	if summary.BundleReportPath != "docs/reports/live-validation-runs/20260316T140138Z/multi-node-shared-queue-report.json" || summary.BundleSummaryPath != "docs/reports/live-validation-runs/20260316T140138Z/shared-queue-companion-summary.json" {
		t.Fatalf("%s unexpected bundle pointers: %+v", path, summary)
	}
	if summary.Count != 200 || summary.CrossNodeCompletions != 99 || summary.DuplicateStarted != 0 || summary.DuplicateCompleted != 0 || summary.MissingCompleted != 0 {
		t.Fatalf("%s unexpected completion counters: %+v", path, summary)
	}
	if len(summary.Nodes) != 2 || summary.Nodes[0] != "node-a" || summary.Nodes[1] != "node-b" {
		t.Fatalf("%s unexpected nodes: %+v", path, summary.Nodes)
	}
	if summary.SubmittedByNode["node-a"] != 100 || summary.SubmittedByNode["node-b"] != 100 {
		t.Fatalf("%s unexpected submitted_by_node counts: %+v", path, summary.SubmittedByNode)
	}
	if summary.CompletedByNode["node-a"] != 73 || summary.CompletedByNode["node-b"] != 127 {
		t.Fatalf("%s unexpected completed_by_node counts: %+v", path, summary.CompletedByNode)
	}
}
