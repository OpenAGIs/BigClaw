package reporting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExportLiveValidationBundle(t *testing.T) {
	root := t.TempDir()
	bundle := filepath.Join(root, "docs", "reports", "live-validation-runs", "20260315T120000Z")
	stateDir := filepath.Join(root, "state-local")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "audit.jsonl"), []byte("{\"event\":\"ok\"}\n"), 0o644); err != nil {
		t.Fatalf("write audit log: %v", err)
	}
	serviceLog := filepath.Join(root, "local-service.log")
	if err := os.WriteFile(serviceLog, []byte("local service ready\n"), 0o644); err != nil {
		t.Fatalf("write service log: %v", err)
	}

	writeFixtureJSON(t, filepath.Join(bundle, "sqlite-smoke-report.json"), map[string]any{
		"base_url":    "http://127.0.0.1:19090",
		"task":        map[string]any{"id": "local-1"},
		"status":      map[string]any{"state": "succeeded"},
		"state_dir":   stateDir,
		"service_log": serviceLog,
	})
	writeFixtureJSON(t, filepath.Join(bundle, "kubernetes-live-smoke-report.json"), map[string]any{"task": map[string]any{"id": "k8s-1"}, "status": map[string]any{"state": "succeeded"}})
	writeFixtureJSON(t, filepath.Join(bundle, "ray-live-smoke-report.json"), map[string]any{"task": map[string]any{"id": "ray-1"}, "status": map[string]any{"state": "succeeded"}})
	writeFixtureJSON(t, filepath.Join(root, "docs", "reports", "multi-node-shared-queue-report.json"), map[string]any{
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
	writeFixtureJSON(t, filepath.Join(root, "docs", "reports", "validation-bundle-continuation-scorecard.json"), map[string]any{})
	writeFixtureJSON(t, filepath.Join(root, "docs", "reports", "validation-bundle-continuation-policy-gate.json"), map[string]any{
		"status":         "policy-hold",
		"recommendation": "hold",
		"failing_checks": []any{"latest_bundle_all_executor_tracks_succeeded"},
		"enforcement":    map[string]any{"mode": "review"},
		"summary":        map[string]any{"latest_bundle_age_hours": 12.5},
		"reviewer_path":  map[string]any{"digest_path": "docs/reports/validation-bundle-continuation-digest.md"},
		"next_actions":   []any{"rerun ./scripts/e2e/run_all.sh"},
	})
	if err := os.WriteFile(filepath.Join(root, "docs", "reports", "validation-bundle-continuation-digest.md"), []byte("# digest\n"), 0o644); err != nil {
		t.Fatalf("write digest: %v", err)
	}

	localStdout := filepath.Join(root, "local.stdout")
	localStderr := filepath.Join(root, "local.stderr")
	k8sStdout := filepath.Join(root, "k8s.stdout")
	k8sStderr := filepath.Join(root, "k8s.stderr")
	rayStdout := filepath.Join(root, "ray.stdout")
	rayStderr := filepath.Join(root, "ray.stderr")
	for path, content := range map[string]string{
		localStdout: "local ok\n",
		localStderr: "",
		k8sStdout:   "k8s ok\n",
		k8sStderr:   "",
		rayStdout:   "ray ok\n",
		rayStderr:   "",
	} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write log %s: %v", path, err)
		}
	}

	summary, indexText, manifest, err := ExportLiveValidationBundle(root, LiveValidationBundleOptions{
		RunID:                "20260315T120000Z",
		BundleDir:            "docs/reports/live-validation-runs/20260315T120000Z",
		SummaryPath:          "docs/reports/live-validation-summary.json",
		IndexPath:            "docs/reports/live-validation-index.md",
		ManifestPath:         "docs/reports/live-validation-index.json",
		RunLocal:             true,
		RunKubernetes:        true,
		RunRay:               true,
		ValidationStatus:     0,
		LocalReportPath:      "docs/reports/live-validation-runs/20260315T120000Z/sqlite-smoke-report.json",
		LocalStdoutPath:      localStdout,
		LocalStderrPath:      localStderr,
		KubernetesReportPath: "docs/reports/live-validation-runs/20260315T120000Z/kubernetes-live-smoke-report.json",
		KubernetesStdoutPath: k8sStdout,
		KubernetesStderrPath: k8sStderr,
		RayReportPath:        "docs/reports/live-validation-runs/20260315T120000Z/ray-live-smoke-report.json",
		RayStdoutPath:        rayStdout,
		RayStderrPath:        rayStderr,
		Now:                  time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("export live validation bundle: %v", err)
	}

	local := asMap(summary["local"])
	if asString(summary["status"]) != "succeeded" || asString(local["canonical_report_path"]) != "docs/reports/sqlite-smoke-report.json" {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if !strings.HasSuffix(asString(local["audit_log_path"]), "local.audit.jsonl") || !strings.HasSuffix(asString(local["service_log_path"]), "local.service.log") {
		t.Fatalf("unexpected local artifacts: %+v", local)
	}
	sharedQueue := asMap(summary["shared_queue_companion"])
	if asString(sharedQueue["status"]) != "succeeded" || asInt(sharedQueue["cross_node_completions"]) != 4 || asString(sharedQueue["canonical_summary_path"]) != "docs/reports/shared-queue-companion-summary.json" {
		t.Fatalf("unexpected shared queue summary: %+v", sharedQueue)
	}
	continuationGate := asMap(summary["continuation_gate"])
	if asString(continuationGate["status"]) != "policy-hold" || asString(continuationGate["recommendation"]) != "hold" {
		t.Fatalf("unexpected continuation gate: %+v", continuationGate)
	}
	if !strings.Contains(indexText, "Live Validation Index") || !strings.Contains(indexText, "docs/reports/shared-queue-companion-summary.json") || !strings.Contains(indexText, "docs/reports/validation-bundle-continuation-policy-gate.json") {
		t.Fatalf("unexpected index text: %s", indexText)
	}
	if asString(asMap(manifest["latest"])["run_id"]) != "20260315T120000Z" {
		t.Fatalf("unexpected manifest latest: %+v", manifest["latest"])
	}
	if asString(asMap(manifest["recent_runs"].([]map[string]any)[0])["run_id"]) != "20260315T120000Z" {
		t.Fatalf("unexpected recent runs: %+v", manifest["recent_runs"])
	}
}
