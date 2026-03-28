package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/testharness"
)

func TestExportValidationBundleGeneratesLatestReportsAndIndex(t *testing.T) {
	root := filepath.Join(t.TempDir(), "repo")
	bundleDir := filepath.Join(root, "docs", "reports", "live-validation-runs", "20260315T120000Z")
	stateDir := filepath.Join(t.TempDir(), "state-local")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	writeTextFile(t, filepath.Join(stateDir, "audit.jsonl"), "{\"event\":\"ok\"}\n")
	serviceLog := filepath.Join(t.TempDir(), "local-service.log")
	writeTextFile(t, serviceLog, "local service ready\n")

	writeLiveValidationJSON(t, filepath.Join(bundleDir, "sqlite-smoke-report.json"), map[string]any{
		"base_url":    "http://127.0.0.1:19090",
		"task":        map[string]any{"id": "local-1"},
		"status":      map[string]any{"state": "succeeded"},
		"state_dir":   stateDir,
		"service_log": serviceLog,
	})
	writeLiveValidationJSON(t, filepath.Join(bundleDir, "kubernetes-live-smoke-report.json"), map[string]any{
		"task":   map[string]any{"id": "k8s-1"},
		"status": map[string]any{"state": "succeeded"},
	})
	writeLiveValidationJSON(t, filepath.Join(bundleDir, "ray-live-smoke-report.json"), map[string]any{
		"task":   map[string]any{"id": "ray-1"},
		"status": map[string]any{"state": "succeeded"},
	})

	reportsDir := filepath.Join(root, "docs", "reports")
	writeLiveValidationJSON(t, filepath.Join(reportsDir, "multi-node-shared-queue-report.json"), map[string]any{
		"generated_at":              "2026-03-15T11:55:00Z",
		"count":                     12,
		"submitted_by_node":         map[string]any{"node-a": 6, "node-b": 6},
		"completed_by_node":         map[string]any{"node-a": 5, "node-b": 7},
		"cross_node_completions":    4,
		"duplicate_started_tasks":   []any{},
		"duplicate_completed_tasks": []any{},
		"missing_completed_tasks":   []any{},
		"all_ok":                    true,
		"nodes":                     []map[string]any{{"name": "node-a"}, {"name": "node-b"}},
	})
	writeTextFile(t, filepath.Join(reportsDir, "validation-bundle-continuation-scorecard.json"), "{}\n")
	writeLiveValidationJSON(t, filepath.Join(reportsDir, "validation-bundle-continuation-policy-gate.json"), map[string]any{
		"status":         "policy-hold",
		"recommendation": "hold",
		"failing_checks": []string{"latest_bundle_all_executor_tracks_succeeded"},
		"enforcement":    map[string]any{"mode": "review"},
		"summary":        map[string]any{"latest_bundle_age_hours": 12.5},
		"reviewer_path":  map[string]any{"digest": "docs/reports/validation-bundle-continuation-digest.md"},
		"next_actions":   []string{"rerun ./scripts/e2e/run_all.sh"},
	})
	writeTextFile(t, filepath.Join(reportsDir, "validation-bundle-continuation-digest.md"), "# digest\n")

	localStdout := filepath.Join(t.TempDir(), "local.stdout")
	localStderr := filepath.Join(t.TempDir(), "local.stderr")
	k8sStdout := filepath.Join(t.TempDir(), "k8s.stdout")
	k8sStderr := filepath.Join(t.TempDir(), "k8s.stderr")
	rayStdout := filepath.Join(t.TempDir(), "ray.stdout")
	rayStderr := filepath.Join(t.TempDir(), "ray.stderr")
	for path, content := range map[string]string{
		localStdout: "local ok\n",
		localStderr: "",
		k8sStdout:   "k8s ok\n",
		k8sStderr:   "",
		rayStdout:   "ray ok\n",
		rayStderr:   "",
	} {
		writeTextFile(t, path, content)
	}

	cmd := exec.Command(
		testharness.PythonExecutable(t),
		testharness.JoinRepoRoot(t, "scripts", "e2e", "export_validation_bundle.py"),
		"--go-root", root,
		"--run-id", "20260315T120000Z",
		"--bundle-dir", "docs/reports/live-validation-runs/20260315T120000Z",
		"--summary-path", "docs/reports/live-validation-summary.json",
		"--index-path", "docs/reports/live-validation-index.md",
		"--manifest-path", "docs/reports/live-validation-index.json",
		"--run-local", "1",
		"--run-kubernetes", "1",
		"--run-ray", "1",
		"--validation-status", "0",
		"--local-report-path", "docs/reports/live-validation-runs/20260315T120000Z/sqlite-smoke-report.json",
		"--local-stdout-path", localStdout,
		"--local-stderr-path", localStderr,
		"--kubernetes-report-path", "docs/reports/live-validation-runs/20260315T120000Z/kubernetes-live-smoke-report.json",
		"--kubernetes-stdout-path", k8sStdout,
		"--kubernetes-stderr-path", k8sStderr,
		"--ray-report-path", "docs/reports/live-validation-runs/20260315T120000Z/ray-live-smoke-report.json",
		"--ray-stdout-path", rayStdout,
		"--ray-stderr-path", rayStderr,
	)
	cmd.Dir = testharness.ProjectRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("export validation bundle: %v (%s)", err, string(output))
	}

	var summary struct {
		Status string `json:"status"`
		Local  struct {
			CanonicalReportPath string `json:"canonical_report_path"`
			AuditLogPath        string `json:"audit_log_path"`
			ServiceLogPath      string `json:"service_log_path"`
		} `json:"local"`
		SharedQueueCompanion struct {
			Status               string   `json:"status"`
			CrossNodeCompletions int      `json:"cross_node_completions"`
			CanonicalSummaryPath string   `json:"canonical_summary_path"`
			BundleSummaryPath    string   `json:"bundle_summary_path"`
			Nodes                []string `json:"nodes"`
		} `json:"shared_queue_companion"`
		ContinuationGate struct {
			Status         string   `json:"status"`
			Recommendation string   `json:"recommendation"`
			FailingChecks  []string `json:"failing_checks"`
			NextActions    []string `json:"next_actions"`
			Summary        struct {
				LatestBundleAgeHours float64 `json:"latest_bundle_age_hours"`
			} `json:"summary"`
		} `json:"continuation_gate"`
	}
	readJSONFile(t, filepath.Join(reportsDir, "live-validation-summary.json"), &summary)
	if summary.Status != "succeeded" ||
		summary.Local.CanonicalReportPath != "docs/reports/sqlite-smoke-report.json" ||
		!strings.HasSuffix(summary.Local.AuditLogPath, "local.audit.jsonl") ||
		!strings.HasSuffix(summary.Local.ServiceLogPath, "local.service.log") {
		t.Fatalf("unexpected live validation summary local payload: %+v", summary)
	}
	if summary.SharedQueueCompanion.Status != "succeeded" ||
		summary.SharedQueueCompanion.CrossNodeCompletions != 4 ||
		summary.SharedQueueCompanion.CanonicalSummaryPath != "docs/reports/shared-queue-companion-summary.json" ||
		!strings.HasSuffix(summary.SharedQueueCompanion.BundleSummaryPath, "shared-queue-companion-summary.json") {
		t.Fatalf("unexpected shared queue companion payload: %+v", summary.SharedQueueCompanion)
	}
	if summary.ContinuationGate.Status != "policy-hold" ||
		summary.ContinuationGate.Recommendation != "hold" ||
		summary.ContinuationGate.Summary.LatestBundleAgeHours != 12.5 ||
		!equalStringSlices(summary.ContinuationGate.FailingChecks, []string{"latest_bundle_all_executor_tracks_succeeded"}) ||
		!equalStringSlices(summary.ContinuationGate.NextActions, []string{"rerun ./scripts/e2e/run_all.sh"}) {
		t.Fatalf("unexpected continuation gate payload: %+v", summary.ContinuationGate)
	}

	var latestLocal struct {
		Task struct {
			ID string `json:"id"`
		} `json:"task"`
	}
	readJSONFile(t, filepath.Join(reportsDir, "sqlite-smoke-report.json"), &latestLocal)
	if latestLocal.Task.ID != "local-1" {
		t.Fatalf("unexpected canonical local report: %+v", latestLocal)
	}

	var sharedQueueSummary struct {
		Nodes            []string `json:"nodes"`
		BundleReportPath string   `json:"bundle_report_path"`
	}
	readJSONFile(t, filepath.Join(reportsDir, "shared-queue-companion-summary.json"), &sharedQueueSummary)
	if !equalStringSlices(sharedQueueSummary.Nodes, []string{"node-a", "node-b"}) ||
		!strings.HasSuffix(sharedQueueSummary.BundleReportPath, "multi-node-shared-queue-report.json") {
		t.Fatalf("unexpected shared queue summary: %+v", sharedQueueSummary)
	}

	indexText := readRepoFile(t, root, filepath.ToSlash(filepath.Join("docs", "reports", "live-validation-index.md")))
	for _, needle := range []string{
		"Live Validation Index",
		"20260315T120000Z",
		"docs/reports/live-validation-runs/20260315T120000Z",
		"shared-queue companion",
		"docs/reports/shared-queue-companion-summary.json",
		"docs/reports/multi-node-shared-queue-report.json",
		"docs/reports/validation-bundle-continuation-scorecard.json",
		"docs/reports/validation-bundle-continuation-policy-gate.json",
		"docs/reports/validation-bundle-continuation-digest.md",
	} {
		if !strings.Contains(indexText, needle) {
			t.Fatalf("live validation index missing %q in %s", needle, indexText)
		}
	}

	var manifest struct {
		Latest struct {
			RunID                string `json:"run_id"`
			SharedQueueCompanion struct {
				Nodes []string `json:"nodes"`
			} `json:"shared_queue_companion"`
		} `json:"latest"`
		RecentRuns []struct {
			RunID string `json:"run_id"`
		} `json:"recent_runs"`
	}
	readJSONFile(t, filepath.Join(reportsDir, "live-validation-index.json"), &manifest)
	if manifest.Latest.RunID != "20260315T120000Z" ||
		!equalStringSlices(manifest.Latest.SharedQueueCompanion.Nodes, []string{"node-a", "node-b"}) ||
		len(manifest.RecentRuns) == 0 ||
		manifest.RecentRuns[0].RunID != "20260315T120000Z" {
		t.Fatalf("unexpected validation manifest: %+v", manifest)
	}
}

func writeLiveValidationJSON(t *testing.T, path string, payload map[string]any) {
	t.Helper()
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	writeTextFile(t, path, string(body)+"\n")
}

func writeTextFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func equalStringSlices(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
