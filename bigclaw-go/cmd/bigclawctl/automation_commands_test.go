package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRunAutomationRunTaskSmokeJSONOutput(t *testing.T) {
	var mu sync.Mutex
	statusCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/healthz":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		case r.Method == http.MethodPost && r.URL.Path == "/tasks":
			var task map[string]any
			if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
				t.Fatalf("decode task: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"task": task})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/tasks/"):
			mu.Lock()
			statusCalls++
			call := statusCalls
			mu.Unlock()
			state := "running"
			if call >= 2 {
				state = "succeeded"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": strings.TrimPrefix(r.URL.Path, "/tasks/"), "state": state})
		case r.Method == http.MethodGet && r.URL.Path == "/events":
			_ = json.NewEncoder(w).Encode(map[string]any{"events": []map[string]any{
				{"type": "queued", "timestamp": "2026-03-28T00:00:00Z"},
				{"type": "succeeded", "timestamp": "2026-03-28T00:00:02Z"},
			}})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	output, err := captureStdout(t, func() error {
		return runAutomation([]string{
			"e2e", "run-task-smoke",
			"--executor", "local",
			"--title", "smoke",
			"--entrypoint", "echo hi",
			"--base-url", server.URL,
			"--timeout-seconds", "2",
			"--poll-interval", "1ms",
			"--json",
		})
	})
	if err != nil {
		t.Fatalf("run automation smoke: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(output, &payload); err != nil {
		t.Fatalf("decode output: %v (%s)", err, string(output))
	}
	if payload["base_url"] != server.URL {
		t.Fatalf("unexpected base_url payload: %+v", payload)
	}
	status, _ := payload["status"].(map[string]any)
	if status["state"] != "succeeded" {
		t.Fatalf("expected succeeded status, got %+v", payload)
	}
}

func TestAutomationSoakLocalWritesReport(t *testing.T) {
	var mu sync.Mutex
	states := map[string]string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/healthz":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		case r.Method == http.MethodPost && r.URL.Path == "/tasks":
			var task map[string]any
			if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
				t.Fatalf("decode task: %v", err)
			}
			taskID := task["id"].(string)
			mu.Lock()
			states[taskID] = "succeeded"
			mu.Unlock()
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{"task": task})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/tasks/"):
			taskID := strings.TrimPrefix(r.URL.Path, "/tasks/")
			mu.Lock()
			state := states[taskID]
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"id": taskID, "state": state})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	goRoot := t.TempDir()
	reportPath := "docs/reports/soak-local-report.json"
	report, exitCode, err := automationSoakLocal(automationSoakLocalOptions{
		Count:          4,
		Workers:        2,
		BaseURL:        server.URL,
		GoRoot:         goRoot,
		TimeoutSeconds: 1,
		ReportPath:     reportPath,
		HTTPClient:     server.Client(),
		Sleep:          func(time.Duration) {},
	})
	if err != nil {
		t.Fatalf("run soak local: %v", err)
	}
	if exitCode != 0 || report.Succeeded != 4 {
		t.Fatalf("unexpected report: exit=%d report=%+v", exitCode, report)
	}
	body, err := os.ReadFile(filepath.Join(goRoot, reportPath))
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(body), "\"succeeded\": 4") {
		t.Fatalf("unexpected report body: %s", string(body))
	}
}

func TestAutomationShadowCompareDetectsMismatch(t *testing.T) {
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/healthz":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		case r.Method == http.MethodPost && r.URL.Path == "/tasks":
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/tasks/"):
			_ = json.NewEncoder(w).Encode(map[string]any{"state": "succeeded"})
		case r.Method == http.MethodGet && r.URL.Path == "/events":
			_ = json.NewEncoder(w).Encode(map[string]any{"events": []map[string]any{
				{"type": "queued", "timestamp": "2026-03-28T00:00:00Z"},
				{"type": "succeeded", "timestamp": "2026-03-28T00:00:03Z"},
			}})
		default:
			t.Fatalf("unexpected primary request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer primary.Close()

	shadow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/healthz":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		case r.Method == http.MethodPost && r.URL.Path == "/tasks":
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/tasks/"):
			_ = json.NewEncoder(w).Encode(map[string]any{"state": "failed"})
		case r.Method == http.MethodGet && r.URL.Path == "/events":
			_ = json.NewEncoder(w).Encode(map[string]any{"events": []map[string]any{
				{"type": "queued", "timestamp": "2026-03-28T00:00:00Z"},
				{"type": "failed", "timestamp": "2026-03-28T00:00:04Z"},
			}})
		default:
			t.Fatalf("unexpected shadow request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer shadow.Close()

	taskFile := filepath.Join(t.TempDir(), "task.json")
	if err := os.WriteFile(taskFile, []byte(`{"id":"compare","title":"compare","entrypoint":"echo hi","execution_timeout_seconds":1}`), 0o644); err != nil {
		t.Fatal(err)
	}

	report, exitCode, err := automationShadowCompare(automationShadowCompareOptions{
		PrimaryBaseURL:       primary.URL,
		ShadowBaseURL:        shadow.URL,
		TaskFile:             taskFile,
		TimeoutSeconds:       1,
		HealthTimeoutSeconds: 1,
		HTTPClient:           primary.Client(),
		Sleep:                func(time.Duration) {},
	})
	if err != nil {
		t.Fatalf("run shadow compare: %v", err)
	}
	if exitCode != 1 {
		t.Fatalf("expected mismatch exit code, got %d", exitCode)
	}
	if report.Diff.StateEqual || report.Diff.EventTypesEqual {
		t.Fatalf("expected mismatch diff, got %+v", report.Diff)
	}
}

func TestAutomationExportValidationBundleWritesBundleArtifacts(t *testing.T) {
	root := t.TempDir()
	bundleDir := filepath.Join("docs", "reports", "live-validation-runs", "run-1")

	localReport := filepath.Join(root, "tmp", "local-report.json")
	k8sReport := filepath.Join(root, "tmp", "k8s-report.json")
	rayReport := filepath.Join(root, "tmp", "ray-report.json")
	localStdout := filepath.Join(root, "tmp", "local.stdout.log")
	localStderr := filepath.Join(root, "tmp", "local.stderr.log")
	k8sStdout := filepath.Join(root, "tmp", "k8s.stdout.log")
	k8sStderr := filepath.Join(root, "tmp", "k8s.stderr.log")
	rayStdout := filepath.Join(root, "tmp", "ray.stdout.log")
	rayStderr := filepath.Join(root, "tmp", "ray.stderr.log")

	for _, path := range []string{localReport, k8sReport, rayReport, localStdout, localStderr, k8sStdout, k8sStderr, rayStdout, rayStderr} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.WriteFile(localReport, []byte(`{"task":{"id":"local-1","required_executor":"local"},"status":{"state":"succeeded","latest_event":{"type":"task.completed","timestamp":"2026-03-28T00:00:02Z"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(k8sReport, []byte(`{"task":{"id":"k8s-1","required_executor":"kubernetes"},"status":{"state":"dead_letter","latest_event":{"id":"evt-dead","type":"task.dead_letter","timestamp":"2026-03-28T00:00:03Z","payload":{"message":"lease lost"}}},"events":[{"id":"evt-routed","type":"scheduler.routed","timestamp":"2026-03-28T00:00:01Z","payload":{"reason":"browser workloads default to kubernetes executor"}},{"id":"evt-dead","type":"task.dead_letter","timestamp":"2026-03-28T00:00:03Z","payload":{"message":"lease lost"}}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rayReport, []byte(`{"task":{"id":"ray-1","required_executor":"ray"},"status":{"state":"succeeded","latest_event":{"type":"task.completed","timestamp":"2026-03-28T00:00:04Z"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	for path, body := range map[string]string{
		localStdout: "local ok\n",
		localStderr: "",
		k8sStdout:   "k8s start\n",
		k8sStderr:   "lease lost\n",
		rayStdout:   "ray ok\n",
		rayStderr:   "",
	} {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.MkdirAll(filepath.Join(root, "docs", "reports"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "reports", "multi-node-shared-queue-report.json"), []byte(`{"all_ok":true,"cross_node_completions":3,"duplicate_started_tasks":[],"duplicate_completed_tasks":[],"missing_completed_tasks":[],"nodes":[{"name":"node-a"},{"name":"node-b"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "reports", "validation-bundle-continuation-policy-gate.json"), []byte(`{"status":"policy-hold","recommendation":"hold","summary":{"latest_run_id":"run-1","failing_check_count":2,"workflow_exit_code":2},"enforcement":{"mode":"hold","outcome":"hold"},"reviewer_path":{"digest_path":"docs/reports/validation-bundle-continuation-digest.md"},"next_actions":["rerun run_all"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "reports", "validation-bundle-continuation-scorecard.json"), []byte(`{"status":"warn"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "reports", "validation-bundle-continuation-digest.md"), []byte("digest\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "reports", "broker-bootstrap-source.json"), []byte(`{"ready":false,"runtime_posture":"contract_only","live_adapter_implemented":false,"proof_boundary":"contract surface","config_completeness":{"driver":false,"urls":false,"topic":false,"consumer_group":false},"validation_errors":["missing broker config"]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	report, exitCode, err := automationExportValidationBundle(automationExportValidationBundleOptions{
		GoRoot:                     root,
		RunID:                      "run-1",
		BundleDir:                  bundleDir,
		SummaryPath:                "docs/reports/live-validation-summary.json",
		IndexPath:                  "docs/reports/live-validation-index.md",
		ManifestPath:               "docs/reports/live-validation-index.json",
		RunLocal:                   true,
		RunKubernetes:              true,
		RunRay:                     true,
		ValidationStatus:           1,
		RunBroker:                  false,
		BrokerBootstrapSummaryPath: "docs/reports/broker-bootstrap-source.json",
		LocalReportPath:            "tmp/local-report.json",
		LocalStdoutPath:            localStdout,
		LocalStderrPath:            localStderr,
		KubernetesReportPath:       "tmp/k8s-report.json",
		KubernetesStdoutPath:       k8sStdout,
		KubernetesStderrPath:       k8sStderr,
		RayReportPath:              "tmp/ray-report.json",
		RayStdoutPath:              rayStdout,
		RayStderrPath:              rayStderr,
		Now:                        func() time.Time { return time.Date(2026, 3, 28, 1, 2, 3, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("export validation bundle: %v", err)
	}
	if exitCode != 1 || report["status"] != "failed" {
		t.Fatalf("unexpected report status: exit=%d report=%+v", exitCode, report)
	}
	k8sSection := report["kubernetes"].(map[string]any)
	rootCause := k8sSection["failure_root_cause"].(map[string]any)
	if rootCause["event_type"] != "task.dead_letter" || rootCause["message"] != "lease lost" {
		t.Fatalf("unexpected k8s root cause: %+v", rootCause)
	}
	indexBody, err := os.ReadFile(filepath.Join(root, "docs", "reports", "live-validation-index.md"))
	if err != nil {
		t.Fatal(err)
	}
	indexText := string(indexBody)
	if !strings.Contains(indexText, "## Validation matrix") || !strings.Contains(indexText, "- Workflow mode: `hold`") {
		t.Fatalf("unexpected index body: %s", indexText)
	}
	if _, err := os.Stat(filepath.Join(root, bundleDir, "README.md")); err != nil {
		t.Fatalf("expected bundle README: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "reports", "sqlite-smoke-report.json")); err != nil {
		t.Fatalf("expected canonical local report copy: %v", err)
	}
}
