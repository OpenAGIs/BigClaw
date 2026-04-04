package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildValidationBundleContinuationScorecard(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "bigclaw-go", "docs", "reports", "live-validation-runs", "run-a"), 0o755); err != nil {
		t.Fatalf("mkdir repo fixtures: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, "bigclaw-go", "docs", "reports", "live-validation-runs", "run-b"), 0o755); err != nil {
		t.Fatalf("mkdir repo fixtures: %v", err)
	}
	writeJSONFixture(t, filepath.Join(repoRoot, "bigclaw-go", "docs", "reports", "live-validation-index.json"), map[string]any{
		"latest": map[string]any{
			"run_id":       "run-a",
			"status":       "succeeded",
			"generated_at": "2026-03-17T04:32:13.251910+00:00",
		},
		"recent_runs": []any{
			map[string]any{"summary_path": "bigclaw-go/docs/reports/live-validation-runs/run-a/summary.json"},
			map[string]any{"summary_path": "bigclaw-go/docs/reports/live-validation-runs/run-b/summary.json"},
		},
	})
	writeJSONFixture(t, filepath.Join(repoRoot, "bigclaw-go", "docs", "reports", "live-validation-runs", "run-a", "summary.json"), map[string]any{
		"generated_at": "2026-03-17T04:32:13.251910+00:00",
		"status":       "succeeded",
		"local":        map[string]any{"enabled": true, "status": "succeeded"},
		"kubernetes":   map[string]any{"enabled": true, "status": "succeeded"},
		"ray":          map[string]any{"enabled": true, "status": "succeeded"},
	})
	writeJSONFixture(t, filepath.Join(repoRoot, "bigclaw-go", "docs", "reports", "live-validation-runs", "run-b", "summary.json"), map[string]any{
		"generated_at": "2026-03-16T04:32:13.251910+00:00",
		"status":       "succeeded",
		"local":        map[string]any{"enabled": true, "status": "succeeded"},
		"kubernetes":   map[string]any{"enabled": true, "status": "succeeded"},
		"ray":          map[string]any{"enabled": true, "status": "succeeded"},
	})
	writeJSONFixture(t, filepath.Join(repoRoot, "bigclaw-go", "docs", "reports", "live-validation-summary.json"), map[string]any{
		"local":                  map[string]any{"status": "succeeded"},
		"kubernetes":             map[string]any{"status": "succeeded"},
		"ray":                    map[string]any{"status": "succeeded"},
		"shared_queue_companion": map[string]any{"available": true, "cross_node_completions": 12},
	})
	writeJSONFixture(t, filepath.Join(repoRoot, "bigclaw-go", "docs", "reports", "multi-node-shared-queue-report.json"), map[string]any{
		"all_ok": true,
	})

	report, err := buildValidationBundleContinuationScorecard(repoRoot, "bigclaw-go/docs/reports/live-validation-index.json", "bigclaw-go/docs/reports/live-validation-runs", "bigclaw-go/docs/reports/live-validation-summary.json", "bigclaw-go/docs/reports/multi-node-shared-queue-report.json")
	if err != nil {
		t.Fatalf("build scorecard: %v", err)
	}

	if report["status"] != "local-continuation-scorecard" {
		t.Fatalf("unexpected status: %+v", report)
	}
	summary := mapAt(report, "summary")
	if !boolAt(summary, "latest_all_executor_tracks_succeeded") || !boolAt(summary, "all_executor_tracks_have_repeated_recent_coverage") {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	evidenceInputs := mapAt(report, "evidence_inputs")
	if evidenceInputs["generator_script"] != "go run ./cmd/bigclawctl e2e validation-bundle-continuation-scorecard" {
		t.Fatalf("unexpected generator script: %+v", evidenceInputs)
	}
}

func TestBuildValidationBundleContinuationPolicyGate(t *testing.T) {
	repoRoot := t.TempDir()
	writeJSONFixture(t, filepath.Join(repoRoot, "bigclaw-go", "docs", "reports", "validation-bundle-continuation-scorecard.json"), map[string]any{
		"summary": map[string]any{
			"latest_run_id":                                     "run-a",
			"latest_bundle_age_hours":                           1.25,
			"recent_bundle_count":                               3,
			"latest_all_executor_tracks_succeeded":              true,
			"recent_bundle_chain_has_no_failures":               true,
			"all_executor_tracks_have_repeated_recent_coverage": true,
		},
		"shared_queue_companion": map[string]any{
			"available":              true,
			"cross_node_completions": 12,
		},
	})

	report, err := buildValidationBundleContinuationPolicyGate(repoRoot, "bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json", 72, 2, true, "review", false)
	if err != nil {
		t.Fatalf("build policy gate: %v", err)
	}
	if report["recommendation"] != "go" || mapAt(report, "enforcement")["outcome"] != "pass" {
		t.Fatalf("unexpected gate payload: %+v", report)
	}
	reviewer := mapAt(report, "reviewer_path")
	if mapAt(reviewer, "digest_issue")["id"] != "OPE-271" {
		t.Fatalf("unexpected reviewer path: %+v", reviewer)
	}
}

func TestRunE2EExportValidationBundle(t *testing.T) {
	root := t.TempDir()
	writeJSONFixture(t, filepath.Join(root, "docs", "reports", "multi-node-shared-queue-report.json"), map[string]any{
		"generated_at":              "2026-03-13T09:45:19Z",
		"all_ok":                    true,
		"count":                     12,
		"cross_node_completions":    6,
		"duplicate_started_tasks":   []any{},
		"duplicate_completed_tasks": []any{},
		"missing_completed_tasks":   []any{},
		"submitted_by_node":         map[string]any{"node-a": 6, "node-b": 6},
		"completed_by_node":         map[string]any{"node-a": 4, "node-b": 8},
		"nodes":                     []any{map[string]any{"name": "node-a"}, map[string]any{"name": "node-b"}},
	})
	writeJSONFixture(t, filepath.Join(root, "docs", "reports", "validation-bundle-continuation-policy-gate.json"), map[string]any{
		"status":         "policy-go",
		"recommendation": "go",
		"failing_checks": []any{},
		"enforcement":    map[string]any{"mode": "review", "outcome": "pass", "exit_code": 0},
		"summary":        map[string]any{"latest_run_id": "run-1", "failing_check_count": 0, "workflow_exit_code": 0},
		"reviewer_path":  map[string]any{"index_path": "docs/reports/live-validation-index.md", "digest_path": "docs/reports/validation-bundle-continuation-digest.md"},
		"next_actions":   []any{"set BIGCLAW_E2E_CONTINUATION_GATE_MODE=fail when workflow closeout should stop on continuation regressions"},
	})
	writeJSONFixture(t, filepath.Join(root, "run-1", "local.json"), map[string]any{
		"base_url": "http://127.0.0.1:1",
		"task":     map[string]any{"id": "local-1", "required_executor": "local"},
		"status":   map[string]any{"state": "succeeded"},
	})
	writeJSONFixture(t, filepath.Join(root, "run-1", "k8s.json"), map[string]any{
		"base_url": "http://127.0.0.1:2",
		"task":     map[string]any{"id": "k8s-1", "required_executor": "kubernetes"},
		"status":   map[string]any{"state": "succeeded"},
	})
	writeJSONFixture(t, filepath.Join(root, "run-1", "ray.json"), map[string]any{
		"base_url": "http://127.0.0.1:3",
		"task":     map[string]any{"id": "ray-1", "required_executor": "ray"},
		"status":   map[string]any{"state": "succeeded"},
	})
	if err := os.WriteFile(filepath.Join(root, "local.out"), []byte("ok\n"), 0o644); err != nil {
		t.Fatalf("write local stdout: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "local.err"), nil, 0o644); err != nil {
		t.Fatalf("write local stderr: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "k8s.out"), []byte("ok\n"), 0o644); err != nil {
		t.Fatalf("write k8s stdout: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "k8s.err"), nil, 0o644); err != nil {
		t.Fatalf("write k8s stderr: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "ray.out"), []byte("ok\n"), 0o644); err != nil {
		t.Fatalf("write ray stdout: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "ray.err"), nil, 0o644); err != nil {
		t.Fatalf("write ray stderr: %v", err)
	}

	if err := runE2EExportValidationBundle([]string{
		"--go-root", root,
		"--run-id", "run-1",
		"--bundle-dir", "docs/reports/live-validation-runs/run-1",
		"--local-report-path", "run-1/local.json",
		"--local-stdout-path", filepath.Join(root, "local.out"),
		"--local-stderr-path", filepath.Join(root, "local.err"),
		"--kubernetes-report-path", "run-1/k8s.json",
		"--kubernetes-stdout-path", filepath.Join(root, "k8s.out"),
		"--kubernetes-stderr-path", filepath.Join(root, "k8s.err"),
		"--ray-report-path", "run-1/ray.json",
		"--ray-stdout-path", filepath.Join(root, "ray.out"),
		"--ray-stderr-path", filepath.Join(root, "ray.err"),
	}); err != nil {
		t.Fatalf("run export validation bundle: %v", err)
	}

	summaryBody, err := os.ReadFile(filepath.Join(root, "docs", "reports", "live-validation-summary.json"))
	if err != nil {
		t.Fatalf("read summary: %v", err)
	}
	if !strings.Contains(string(summaryBody), `"run_id": "run-1"`) || !strings.Contains(string(summaryBody), `"canonical_summary_path": "docs/reports/shared-queue-companion-summary.json"`) {
		t.Fatalf("unexpected summary body: %s", string(summaryBody))
	}
	indexBody, err := os.ReadFile(filepath.Join(root, "docs", "reports", "live-validation-index.md"))
	if err != nil {
		t.Fatalf("read index: %v", err)
	}
	if !strings.Contains(string(indexBody), "## Continuation gate") || !strings.Contains(string(indexBody), "## Workflow closeout commands") {
		t.Fatalf("unexpected index body: %s", string(indexBody))
	}
}

func TestRunE2ETaskSmokeWithoutAutostartWritesReport(t *testing.T) {
	var taskID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/healthz":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		case r.URL.Path == "/tasks" && r.Method == http.MethodPost:
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode task payload: %v", err)
			}
			taskID = payload["id"].(string)
			_ = json.NewEncoder(w).Encode(map[string]any{"task": payload})
		case strings.HasPrefix(r.URL.Path, "/tasks/"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": taskID, "state": "succeeded"})
		case r.URL.Path == "/events":
			_ = json.NewEncoder(w).Encode(map[string]any{"events": []any{map[string]any{"id": "evt-1", "type": "task.completed"}}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	goRoot := t.TempDir()
	reportPath := filepath.Join("docs", "reports", "smoke.json")
	if err := runE2ETaskSmoke([]string{
		"--executor", "local",
		"--title", "Smoke",
		"--entrypoint", "echo ok",
		"--base-url", server.URL,
		"--go-root", goRoot,
		"--report-path", reportPath,
		"--timeout-seconds", "2",
		"--poll-interval", "10ms",
	}); err != nil {
		t.Fatalf("run task smoke: %v", err)
	}

	reportBody, err := os.ReadFile(filepath.Join(goRoot, reportPath))
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(reportBody), `"autostarted": false`) || !strings.Contains(string(reportBody), `"state": "succeeded"`) {
		t.Fatalf("unexpected report body: %s", string(reportBody))
	}
}

func TestBuildSubscriberTakeoverFaultMatrixReport(t *testing.T) {
	report := buildSubscriberTakeoverFaultMatrixReport(time.Date(2026, 3, 16, 10, 20, 20, 246671000, time.UTC))
	if report["ticket"] != "OPE-269" || report["status"] != "local-executable" {
		t.Fatalf("unexpected report identity: %+v", report)
	}
	summary := mapAt(report, "summary")
	if intAt(summary, "scenario_count") != 3 || intAt(summary, "passing_scenarios") != 3 || intAt(summary, "duplicate_delivery_count") != 4 || intAt(summary, "stale_write_rejections") != 2 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	currentPrimitives := mapAt(report, "current_primitives")
	takeoverHarness, ok := currentPrimitives["takeover_harness"].([]string)
	if !ok || len(takeoverHarness) != 2 || takeoverHarness[0] != "cmd/bigclawctl/e2e.go" {
		t.Fatalf("unexpected takeover harness primitive: %+v", currentPrimitives)
	}
}

func writeJSONFixture(t *testing.T, path string, payload any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture parent: %v", err)
	}
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	if err := os.WriteFile(path, append(body, '\n'), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}
