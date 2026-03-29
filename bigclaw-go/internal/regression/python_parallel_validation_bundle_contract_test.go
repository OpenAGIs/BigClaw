package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLane8PythonExportValidationBundleScriptStaysAligned(t *testing.T) {
	goRepoRoot := repoRoot(t)
	tmp := t.TempDir()
	root := filepath.Join(tmp, "repo")
	bundle := filepath.Join(root, "docs", "reports", "live-validation-runs", "20260315T120000Z")
	stateDir := filepath.Join(tmp, "state-local")

	mustMkdirAll(t, stateDir)
	if err := os.WriteFile(filepath.Join(stateDir, "audit.jsonl"), []byte("{\"event\":\"ok\"}\n"), 0o644); err != nil {
		t.Fatalf("write audit log: %v", err)
	}
	serviceLog := filepath.Join(tmp, "local-service.log")
	if err := os.WriteFile(serviceLog, []byte("local service ready\n"), 0o644); err != nil {
		t.Fatalf("write service log: %v", err)
	}

	writeJSONFixture(t, filepath.Join(bundle, "sqlite-smoke-report.json"), map[string]any{
		"base_url": "http://127.0.0.1:19090",
		"task":     map[string]any{"id": "local-1"},
		"status":   map[string]any{"state": "succeeded"},
		"state_dir": stateDir,
		"service_log": serviceLog,
	})
	writeJSONFixture(t, filepath.Join(bundle, "kubernetes-live-smoke-report.json"), map[string]any{
		"task":   map[string]any{"id": "k8s-1"},
		"status": map[string]any{"state": "succeeded"},
	})
	writeJSONFixture(t, filepath.Join(bundle, "ray-live-smoke-report.json"), map[string]any{
		"task":   map[string]any{"id": "ray-1"},
		"status": map[string]any{"state": "succeeded"},
	})

	writeJSONFixture(t, filepath.Join(root, "docs", "reports", "multi-node-shared-queue-report.json"), map[string]any{
		"generated_at":             "2026-03-15T11:55:00Z",
		"count":                    12,
		"submitted_by_node":        map[string]any{"node-a": 6, "node-b": 6},
		"completed_by_node":        map[string]any{"node-a": 5, "node-b": 7},
		"cross_node_completions":   4,
		"duplicate_started_tasks":  []any{},
		"duplicate_completed_tasks": []any{},
		"missing_completed_tasks":  []any{},
		"all_ok":                   true,
		"nodes":                    []map[string]any{{"name": "node-a"}, {"name": "node-b"}},
	})
	if err := os.WriteFile(filepath.Join(root, "docs", "reports", "validation-bundle-continuation-scorecard.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write continuation scorecard: %v", err)
	}
	writeJSONFixture(t, filepath.Join(root, "docs", "reports", "validation-bundle-continuation-policy-gate.json"), map[string]any{
		"status":         "policy-hold",
		"recommendation": "hold",
		"failing_checks": []string{"latest_bundle_all_executor_tracks_succeeded"},
		"enforcement":    map[string]any{"mode": "review"},
		"summary":        map[string]any{"latest_bundle_age_hours": 12.5},
		"reviewer_path":  map[string]any{"digest": "docs/reports/validation-bundle-continuation-digest.md"},
		"next_actions":   []string{"rerun ./scripts/e2e/run_all.sh"},
	})
	if err := os.WriteFile(filepath.Join(root, "docs", "reports", "validation-bundle-continuation-digest.md"), []byte("# digest\n"), 0o644); err != nil {
		t.Fatalf("write continuation digest: %v", err)
	}

	localStdout := filepath.Join(tmp, "local.stdout")
	localStderr := filepath.Join(tmp, "local.stderr")
	k8sStdout := filepath.Join(tmp, "k8s.stdout")
	k8sStderr := filepath.Join(tmp, "k8s.stderr")
	rayStdout := filepath.Join(tmp, "ray.stdout")
	rayStderr := filepath.Join(tmp, "ray.stderr")
	for path, contents := range map[string]string{
		localStdout: "local ok\n",
		localStderr: "",
		k8sStdout:   "k8s ok\n",
		k8sStderr:   "",
		rayStdout:   "ray ok\n",
		rayStderr:   "",
	} {
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatalf("write log fixture %s: %v", path, err)
		}
	}

	script := filepath.Join(goRepoRoot, "scripts", "e2e", "export_validation_bundle.py")
	wrapper := filepath.Join(tmp, "run_export_validation_bundle.py")
	wrapperSource := `import __future__
import pathlib
import sys

script = pathlib.Path(sys.argv[1])
sys.argv = [str(script)] + sys.argv[2:]
code = script.read_text(encoding="utf-8")
globals_dict = {
    "__name__": "__main__",
    "__file__": str(script),
    "__package__": None,
}
exec(compile(code, str(script), "exec", flags=__future__.annotations.compiler_flag, dont_inherit=True), globals_dict)
`
	if err := os.WriteFile(wrapper, []byte(wrapperSource), 0o644); err != nil {
		t.Fatalf("write export_validation_bundle wrapper: %v", err)
	}
	cmd := exec.Command(
		"python3",
		wrapper,
		script,
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
	cmd.Dir = goRepoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run export_validation_bundle.py: %v\n%s", err, string(output))
	}

	var summary struct {
		Status string `json:"status"`
		Local  struct {
			CanonicalReportPath string `json:"canonical_report_path"`
			AuditLogPath        string `json:"audit_log_path"`
			ServiceLogPath      string `json:"service_log_path"`
		} `json:"local"`
		SharedQueueCompanion struct {
			Status               string `json:"status"`
			CrossNodeCompletions int    `json:"cross_node_completions"`
			CanonicalSummaryPath string `json:"canonical_summary_path"`
			BundleSummaryPath    string `json:"bundle_summary_path"`
		} `json:"shared_queue_companion"`
		ContinuationGate struct {
			Status         string `json:"status"`
			Recommendation string `json:"recommendation"`
			Summary        struct {
				LatestBundleAgeHours float64 `json:"latest_bundle_age_hours"`
			} `json:"summary"`
			FailingChecks []string `json:"failing_checks"`
			NextActions   []string `json:"next_actions"`
		} `json:"continuation_gate"`
	}
	readJSONFixture(t, filepath.Join(root, "docs", "reports", "live-validation-summary.json"), &summary)
	if summary.Status != "succeeded" ||
		summary.Local.CanonicalReportPath != "docs/reports/sqlite-smoke-report.json" ||
		!strings.HasSuffix(summary.Local.AuditLogPath, "local.audit.jsonl") ||
		!strings.HasSuffix(summary.Local.ServiceLogPath, "local.service.log") ||
		summary.SharedQueueCompanion.Status != "succeeded" ||
		summary.SharedQueueCompanion.CrossNodeCompletions != 4 ||
		summary.SharedQueueCompanion.CanonicalSummaryPath != "docs/reports/shared-queue-companion-summary.json" ||
		!strings.HasSuffix(summary.SharedQueueCompanion.BundleSummaryPath, "shared-queue-companion-summary.json") ||
		summary.ContinuationGate.Status != "policy-hold" ||
		summary.ContinuationGate.Recommendation != "hold" ||
		summary.ContinuationGate.Summary.LatestBundleAgeHours != 12.5 ||
		len(summary.ContinuationGate.FailingChecks) != 1 || summary.ContinuationGate.FailingChecks[0] != "latest_bundle_all_executor_tracks_succeeded" ||
		len(summary.ContinuationGate.NextActions) != 1 || summary.ContinuationGate.NextActions[0] != "rerun ./scripts/e2e/run_all.sh" {
		t.Fatalf("unexpected validation summary payload: %+v", summary)
	}

	var latestLocal struct {
		Task struct {
			ID string `json:"id"`
		} `json:"task"`
	}
	readJSONFixture(t, filepath.Join(root, "docs", "reports", "sqlite-smoke-report.json"), &latestLocal)
	if latestLocal.Task.ID != "local-1" {
		t.Fatalf("unexpected latest local report: %+v", latestLocal)
	}

	var sharedQueue struct {
		Nodes            []string `json:"nodes"`
		BundleReportPath string   `json:"bundle_report_path"`
	}
	readJSONFixture(t, filepath.Join(root, "docs", "reports", "shared-queue-companion-summary.json"), &sharedQueue)
	if len(sharedQueue.Nodes) != 2 || sharedQueue.Nodes[0] != "node-a" || sharedQueue.Nodes[1] != "node-b" || !strings.HasSuffix(sharedQueue.BundleReportPath, "multi-node-shared-queue-report.json") {
		t.Fatalf("unexpected shared queue summary: %+v", sharedQueue)
	}

	indexBytes, err := os.ReadFile(filepath.Join(root, "docs", "reports", "live-validation-index.md"))
	if err != nil {
		t.Fatalf("read live validation index markdown: %v", err)
	}
	indexText := string(indexBytes)
	for _, want := range []string{
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
		if !strings.Contains(indexText, want) {
			t.Fatalf("live validation index missing %q\n%s", want, indexText)
		}
	}

	var manifest struct {
		Latest struct {
			RunID string `json:"run_id"`
			SharedQueueCompanion struct {
				Nodes []string `json:"nodes"`
			} `json:"shared_queue_companion"`
		} `json:"latest"`
		RecentRuns []struct {
			RunID string `json:"run_id"`
		} `json:"recent_runs"`
	}
	readJSONFixture(t, filepath.Join(root, "docs", "reports", "live-validation-index.json"), &manifest)
	if manifest.Latest.RunID != "20260315T120000Z" ||
		len(manifest.Latest.SharedQueueCompanion.Nodes) != 2 ||
		manifest.Latest.SharedQueueCompanion.Nodes[0] != "node-a" ||
		manifest.Latest.SharedQueueCompanion.Nodes[1] != "node-b" ||
		len(manifest.RecentRuns) == 0 ||
		manifest.RecentRuns[0].RunID != "20260315T120000Z" {
		t.Fatalf("unexpected validation index manifest: %+v", manifest)
	}
}

func writeJSONFixture(t *testing.T, path string, payload any) {
	t.Helper()
	mustMkdirAll(t, filepath.Dir(path))
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal json fixture %s: %v", path, err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write json fixture %s: %v", path, err)
	}
}

func readJSONFixture(t *testing.T, path string, target any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read json fixture %s: %v", path, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("decode json fixture %s: %v\n%s", path, err, string(data))
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}
