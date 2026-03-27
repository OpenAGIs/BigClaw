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

func TestAutomationValidationBundleScorecardBuildsReport(t *testing.T) {
	repoRoot := t.TempDir()
	goRoot := filepath.Join(repoRoot, "bigclaw-go")
	reportsDir := filepath.Join(goRoot, "docs", "reports")
	runDir := filepath.Join(reportsDir, "live-validation-runs", "run-1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}

	latestSummary := `{
  "run_id": "run-1",
  "generated_at": "2026-03-28T01:00:00Z",
  "status": "succeeded",
  "local": {"enabled": true, "status": "succeeded"},
  "kubernetes": {"enabled": true, "status": "succeeded"},
  "ray": {"enabled": true, "status": "succeeded"},
  "shared_queue_companion": {
    "available": true,
    "canonical_report_path": "docs/reports/multi-node-shared-queue-report.json",
    "canonical_summary_path": "docs/reports/shared-queue-companion-summary.json",
    "bundle_report_path": "docs/reports/live-validation-runs/run-1/multi-node-shared-queue-report.json",
    "bundle_summary_path": "docs/reports/live-validation-runs/run-1/shared-queue-companion-summary.json",
    "cross_node_completions": 5,
    "duplicate_completed_tasks": 0,
    "duplicate_started_tasks": 0
  }
}`
	if err := os.WriteFile(filepath.Join(runDir, "summary.json"), []byte(latestSummary), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportsDir, "live-validation-summary.json"), []byte(latestSummary), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportsDir, "live-validation-index.json"), []byte(`{"latest":{"run_id":"run-1","generated_at":"2026-03-28T01:00:00Z","status":"succeeded"},"recent_runs":[{"summary_path":"bigclaw-go/docs/reports/live-validation-runs/run-1/summary.json"},{"summary_path":"bigclaw-go/docs/reports/live-validation-runs/run-1/summary.json"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportsDir, "multi-node-shared-queue-report.json"), []byte(`{"all_ok":true,"cross_node_completions":5,"duplicate_completed_tasks":[],"duplicate_started_tasks":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	report, _, err := automationValidationBundleScorecard(automationValidationBundleScorecardOptions{
		RepoRoot:              repoRoot,
		IndexManifestPath:     "bigclaw-go/docs/reports/live-validation-index.json",
		BundleRootPath:        "bigclaw-go/docs/reports/live-validation-runs",
		SummaryPath:           "bigclaw-go/docs/reports/live-validation-summary.json",
		SharedQueueReportPath: "bigclaw-go/docs/reports/multi-node-shared-queue-report.json",
		OutputPath:            "bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json",
		Now:                   func() time.Time { return time.Date(2026, 3, 28, 3, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("build scorecard: %v", err)
	}
	summary := report["summary"].(map[string]any)
	if summary["recent_bundle_count"] != 2 || summary["latest_all_executor_tracks_succeeded"] != true {
		t.Fatalf("unexpected scorecard summary: %+v", summary)
	}
	lanes := report["executor_lanes"].([]any)
	if len(lanes) != 3 {
		t.Fatalf("expected 3 lane scorecards, got %+v", lanes)
	}
}

func TestAutomationValidationBundlePolicyGateRespectsLegacyEnforce(t *testing.T) {
	repoRoot := t.TempDir()
	scorecardPath := filepath.Join(repoRoot, "bigclaw-go", "docs", "reports", "validation-bundle-continuation-scorecard.json")
	if err := os.MkdirAll(filepath.Dir(scorecardPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scorecardPath, []byte(`{
  "summary": {
    "latest_run_id": "run-1",
    "latest_bundle_age_hours": 96.0,
    "recent_bundle_count": 1,
    "latest_all_executor_tracks_succeeded": false,
    "recent_bundle_chain_has_no_failures": false,
    "all_executor_tracks_have_repeated_recent_coverage": false
  },
  "shared_queue_companion": {
    "available": false,
    "cross_node_completions": 0
  }
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	report, exitCode, err := automationValidationBundlePolicyGate(automationValidationBundlePolicyGateOptions{
		RepoRoot:                    repoRoot,
		ScorecardPath:               "bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json",
		OutputPath:                  "bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json",
		MaxLatestAgeHours:           72,
		MinRecentBundles:            2,
		RequireRepeatedLaneCoverage: true,
		LegacyEnforce:               true,
		Now:                         func() time.Time { return time.Date(2026, 3, 28, 3, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("build policy gate: %v", err)
	}
	enforcement := report["enforcement"].(map[string]any)
	if enforcement["mode"] != "fail" || exitCode != 1 {
		t.Fatalf("unexpected enforcement: exit=%d report=%+v", exitCode, enforcement)
	}
	failingChecks := report["failing_checks"].([]any)
	if len(failingChecks) == 0 {
		t.Fatalf("expected failing checks, got %+v", report)
	}
}

func TestAutomationBenchmarkMatrixParsesBenchAndWritesReport(t *testing.T) {
	goRoot := t.TempDir()
	report, exitCode, err := automationBenchmarkMatrix(automationBenchmarkMatrixOptions{
		GoRoot:         goRoot,
		ReportPath:     "docs/reports/benchmark-matrix-report.json",
		TimeoutSeconds: 180,
		Scenarios:      []string{"50:8", "100:12"},
		RunCommand: func(name string, args []string, dir string) ([]byte, error) {
			if name != "go" {
				t.Fatalf("unexpected command: %s", name)
			}
			return []byte("BenchmarkSchedulerDecide-8  1000  800 ns/op\nBenchmarkFileQueueEnqueueLease-8  10  12000000 ns/op\n"), nil
		},
		RunSoak: func(count int, workers int, timeoutSeconds int, reportPath string) (map[string]any, error) {
			return map[string]any{
				"count":                    count,
				"workers":                  workers,
				"elapsed_seconds":          10.0,
				"throughput_tasks_per_sec": 9.5,
				"succeeded":                count,
				"failed":                   0,
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("run matrix: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("unexpected exit code: %d", exitCode)
	}
	benchmark := report["benchmark"].(map[string]any)
	parsed := benchmark["parsed"].(map[string]any)
	if parsed["BenchmarkSchedulerDecide-8"].(map[string]any)["ns_per_op"] != 800.0 {
		t.Fatalf("unexpected parsed benchmark payload: %+v", parsed)
	}
	soakMatrix := report["soak_matrix"].([]any)
	if len(soakMatrix) != 2 {
		t.Fatalf("expected 2 soak scenarios, got %+v", soakMatrix)
	}
	body, err := os.ReadFile(filepath.Join(goRoot, "docs/reports/benchmark-matrix-report.json"))
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(body), "\"throughput_tasks_per_sec\": 9.5") {
		t.Fatalf("unexpected report body: %s", string(body))
	}
}

func TestAutomationCapacityCertificationBuildsMatrixAndMarkdown(t *testing.T) {
	repoRoot := t.TempDir()
	reportsDir := filepath.Join(repoRoot, "bigclaw-go", "docs", "reports")
	if err := os.MkdirAll(reportsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	benchmarkReport := `{
  "benchmark": {
    "parsed": {
      "BenchmarkMemoryQueueEnqueueLease-8": {"ns_per_op": 90000},
      "BenchmarkFileQueueEnqueueLease-8": {"ns_per_op": 12000000},
      "BenchmarkSQLiteQueueEnqueueLease-8": {"ns_per_op": 11000000},
      "BenchmarkSchedulerDecide-8": {"ns_per_op": 800}
    }
  },
  "soak_matrix": [
    {"report_path":"docs/reports/soak-local-50x8.json","result":{"count":50,"workers":8,"elapsed_seconds":8.1,"throughput_tasks_per_sec":6.2,"succeeded":50,"failed":0,"generated_at":"2026-03-13T09:01:00.000001Z"}},
    {"report_path":"docs/reports/soak-local-100x12.json","result":{"count":100,"workers":12,"elapsed_seconds":10.0,"throughput_tasks_per_sec":9.1,"succeeded":100,"failed":0,"generated_at":"2026-03-13T09:02:00.000001Z"}}
  ],
  "generated_at":"2026-03-13T09:02:00.000001Z"
}`
	mixedWorkloadReport := `{
  "all_ok": true,
  "generated_at":"2026-03-13T09:30:00.000001Z",
  "tasks": [
    {"name":"local","ok":true,"expected_executor":"local","routed_executor":"local","final_state":"succeeded"},
    {"name":"k8s","ok":true,"expected_executor":"kubernetes","routed_executor":"kubernetes","final_state":"succeeded"},
    {"name":"ray","ok":true,"expected_executor":"ray","routed_executor":"ray","final_state":"succeeded"},
    {"name":"browser","ok":true,"expected_executor":"kubernetes","routed_executor":"kubernetes","final_state":"succeeded"},
    {"name":"python","ok":true,"expected_executor":"local","routed_executor":"local","final_state":"succeeded"}
  ]
}`
	if err := os.WriteFile(filepath.Join(reportsDir, "benchmark-matrix-report.json"), []byte(benchmarkReport), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportsDir, "mixed-workload-matrix-report.json"), []byte(mixedWorkloadReport), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportsDir, "soak-local-1000x24.json"), []byte(`{"count":1000,"workers":24,"elapsed_seconds":100,"throughput_tasks_per_sec":9.5,"succeeded":1000,"failed":0,"generated_at":"2026-03-13T09:44:42.458392Z"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportsDir, "soak-local-2000x24.json"), []byte(`{"count":2000,"workers":24,"elapsed_seconds":215,"throughput_tasks_per_sec":9.1,"succeeded":2000,"failed":0,"generated_at":"2026-03-13T09:43:42.458392Z"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	report, _, err := automationCapacityCertification(automationCapacityCertificationOptions{
		RepoRoot:                repoRoot,
		BenchmarkReportPath:     "bigclaw-go/docs/reports/benchmark-matrix-report.json",
		MixedWorkloadReportPath: "bigclaw-go/docs/reports/mixed-workload-matrix-report.json",
		SupplementalSoakReports: []string{
			"bigclaw-go/docs/reports/soak-local-1000x24.json",
			"bigclaw-go/docs/reports/soak-local-2000x24.json",
		},
		OutputPath:         "bigclaw-go/docs/reports/capacity-certification-matrix.json",
		MarkdownOutputPath: "bigclaw-go/docs/reports/capacity-certification-report.md",
	})
	if err != nil {
		t.Fatalf("capacity certification: %v", err)
	}
	summary := report["summary"].(map[string]any)
	if summary["overall_status"] != "pass" {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	saturation := report["saturation_indicator"].(map[string]any)
	if saturation["status"] != "pass" {
		t.Fatalf("unexpected saturation indicator: %+v", saturation)
	}
	mixed := report["mixed_workload"].(map[string]any)
	if mixed["status"] != "pass" {
		t.Fatalf("unexpected mixed workload lane: %+v", mixed)
	}
	if report["generated_at"] != "2026-03-13T09:44:42.458392Z" {
		t.Fatalf("unexpected generated_at: %+v", report["generated_at"])
	}
	markdownBody, err := os.ReadFile(filepath.Join(reportsDir, "capacity-certification-report.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(markdownBody), "## Admission Policy Summary") || !strings.Contains(string(markdownBody), "Runtime enforcement: `none`") {
		t.Fatalf("unexpected markdown body: %s", string(markdownBody))
	}
}

func TestAutomationLiveShadowScorecardBuildsReport(t *testing.T) {
	repoRoot := t.TempDir()
	reportsDir := filepath.Join(repoRoot, "bigclaw-go", "docs", "reports")
	if err := os.MkdirAll(reportsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	compare := `{
  "trace_id":"trace-1",
  "primary":{"task_id":"primary-1","events":[{"timestamp":"2026-03-13T09:40:00Z"},{"timestamp":"2026-03-13T09:41:00Z"}]},
  "shadow":{"task_id":"shadow-1","events":[{"timestamp":"2026-03-13T09:40:02Z"},{"timestamp":"2026-03-13T09:41:00Z"}]},
  "diff":{"state_equal":true,"event_types_equal":true,"event_count_delta":0,"primary_timeline_seconds":1.0,"shadow_timeline_seconds":1.1}
}`
	matrix := `{
  "total":1,
  "matched":1,
  "mismatched":0,
  "corpus_coverage":{"corpus_slice_count":2,"uncovered_corpus_slice_count":0},
  "results":[
    {
      "trace_id":"trace-2",
      "source_file":"fixture.json",
      "source_kind":"fixture",
      "task_shape":"executor:local",
      "primary":{"task_id":"primary-2","events":[{"timestamp":"2026-03-13T09:42:00Z"}]},
      "shadow":{"task_id":"shadow-2","events":[{"timestamp":"2026-03-13T09:42:00Z"}]},
      "diff":{"state_equal":true,"event_types_equal":true,"event_count_delta":0,"primary_timeline_seconds":1.0,"shadow_timeline_seconds":1.0}
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(reportsDir, "shadow-compare-report.json"), []byte(compare), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportsDir, "shadow-matrix-report.json"), []byte(matrix), 0o644); err != nil {
		t.Fatal(err)
	}
	report, _, err := automationLiveShadowScorecard(automationLiveShadowScorecardOptions{
		RepoRoot:            repoRoot,
		ShadowCompareReport: "bigclaw-go/docs/reports/shadow-compare-report.json",
		ShadowMatrixReport:  "bigclaw-go/docs/reports/shadow-matrix-report.json",
		OutputPath:          "bigclaw-go/docs/reports/live-shadow-mirror-scorecard.json",
		Now:                 func() time.Time { return time.Date(2026, 3, 13, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("live shadow scorecard: %v", err)
	}
	summary := report["summary"].(map[string]any)
	if summary["parity_ok_count"] != 2 || summary["drift_detected_count"] != 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if summary["fresh_inputs"] != 2 {
		t.Fatalf("unexpected freshness summary: %+v", summary)
	}
}

func TestAutomationExportLiveShadowBundleWritesManifest(t *testing.T) {
	root := t.TempDir()
	reportsDir := filepath.Join(root, "docs", "reports")
	if err := os.MkdirAll(filepath.Join(reportsDir, "live-shadow-runs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportsDir, "shadow-compare-report.json"), []byte(`{"trace_id":"trace-1","primary":{"events":[{"timestamp":"2026-03-13T09:40:00Z"}]},"shadow":{"events":[{"timestamp":"2026-03-13T09:40:01Z"}]},"diff":{"state_equal":true,"event_types_equal":true}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportsDir, "shadow-matrix-report.json"), []byte(`{"results":[{"trace_id":"trace-2"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportsDir, "live-shadow-mirror-scorecard.json"), []byte(`{
  "summary":{"latest_evidence_timestamp":"2026-03-13T08:56:55Z","total_evidence_runs":2,"parity_ok_count":2,"drift_detected_count":0,"matrix_total":1,"matrix_mismatched":0,"stale_inputs":0,"fresh_inputs":2},
  "freshness":[],
  "cutover_checkpoints":[{"name":"ok","passed":true}]
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportsDir, "rollback-trigger-surface.json"), []byte(`{"summary":{"status":"ok","automation_boundary":"manual-only","automated_rollback_trigger":false,"distinctions":{"cutover":"manual"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	report, _, err := automationExportLiveShadowBundle(automationExportLiveShadowBundleOptions{
		GoRoot:              root,
		ShadowCompareReport: "docs/reports/shadow-compare-report.json",
		ShadowMatrixReport:  "docs/reports/shadow-matrix-report.json",
		ScorecardReport:     "docs/reports/live-shadow-mirror-scorecard.json",
		BundleRoot:          "docs/reports/live-shadow-runs",
		SummaryPath:         "docs/reports/live-shadow-summary.json",
		IndexPath:           "docs/reports/live-shadow-index.md",
		ManifestPath:        "docs/reports/live-shadow-index.json",
		RollupPath:          "docs/reports/live-shadow-drift-rollup.json",
		Now:                 func() time.Time { return time.Date(2026, 3, 13, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("export live shadow bundle: %v", err)
	}
	latest := report["latest"].(map[string]any)
	if latest["run_id"] != "20260313T085655Z" {
		t.Fatalf("unexpected run id: %+v", latest)
	}
	indexBody, err := os.ReadFile(filepath.Join(reportsDir, "live-shadow-index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(indexBody), "## Latest bundle artifacts") {
		t.Fatalf("unexpected index body: %s", string(indexBody))
	}
	if _, err := os.Stat(filepath.Join(reportsDir, "live-shadow-runs", "20260313T085655Z", "README.md")); err != nil {
		t.Fatalf("expected bundle README: %v", err)
	}
}

func TestAutomationShadowMatrixBuildsCoverageReport(t *testing.T) {
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/healthz":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		case r.Method == http.MethodPost && r.URL.Path == "/tasks":
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/tasks/"):
			_ = json.NewEncoder(w).Encode(map[string]any{"state": "succeeded"})
		case r.Method == http.MethodGet && r.URL.Path == "/events":
			_ = json.NewEncoder(w).Encode(map[string]any{"events": []map[string]any{{"type": "queued", "timestamp": "2026-03-28T00:00:00Z"}, {"type": "succeeded", "timestamp": "2026-03-28T00:00:01Z"}}})
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
			_ = json.NewEncoder(w).Encode(map[string]any{"state": "succeeded"})
		case r.Method == http.MethodGet && r.URL.Path == "/events":
			_ = json.NewEncoder(w).Encode(map[string]any{"events": []map[string]any{{"type": "queued", "timestamp": "2026-03-28T00:00:00Z"}, {"type": "succeeded", "timestamp": "2026-03-28T00:00:01Z"}}})
		default:
			t.Fatalf("unexpected shadow request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer shadow.Close()

	root := t.TempDir()
	taskFile := filepath.Join(root, "task.json")
	if err := os.WriteFile(taskFile, []byte(`{"id":"matrix","title":"matrix","entrypoint":"echo hi","execution_timeout_seconds":1,"required_executor":"local"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(`{"name":"shadow-corpus","slices":[{"slice_id":"slice-1","title":"Slice 1","weight":2,"task":{"id":"corpus","title":"corpus","entrypoint":"echo hi","execution_timeout_seconds":1,"required_executor":"local"},"task_shape":"executor:local"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	reportPath := filepath.Join(root, "shadow-matrix-report.json")

	report, exitCode, err := automationShadowMatrix(automationShadowMatrixOptions{
		PrimaryBaseURL:       primary.URL,
		ShadowBaseURL:        shadow.URL,
		TaskFiles:            []string{taskFile},
		CorpusManifest:       manifestPath,
		ReplayCorpusSlices:   true,
		TimeoutSeconds:       1,
		HealthTimeoutSeconds: 1,
		ReportPath:           reportPath,
	})
	if err != nil {
		t.Fatalf("shadow matrix: %v", err)
	}
	if exitCode != 0 || report["matched"] != 2 {
		t.Fatalf("unexpected matrix report: exit=%d report=%+v", exitCode, report)
	}
	coverage := report["corpus_coverage"].(map[string]any)
	if coverage["corpus_slice_count"] != 1 || coverage["covered_corpus_slice_count"] != 1 {
		t.Fatalf("unexpected corpus coverage: %+v", coverage)
	}
}

func TestAutomationCrossProcessCoordinationSurfaceBuildsReport(t *testing.T) {
	repoRoot := t.TempDir()
	reportsDir := filepath.Join(repoRoot, "bigclaw-go", "docs", "reports")
	if err := os.MkdirAll(reportsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportsDir, "multi-node-shared-queue-report.json"), []byte(`{"count":200,"cross_node_completions":40,"duplicate_completed_tasks":[],"duplicate_started_tasks":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportsDir, "multi-subscriber-takeover-validation-report.json"), []byte(`{"summary":{"scenario_count":4,"passing_scenarios":4,"duplicate_delivery_count":0,"stale_write_rejections":2}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportsDir, "live-multi-node-subscriber-takeover-report.json"), []byte(`{"summary":{"scenario_count":2,"passing_scenarios":2,"stale_write_rejections":1}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	report, _, err := automationCrossProcessCoordinationSurface(automationCrossProcessCoordinationSurfaceOptions{
		RepoRoot:               repoRoot,
		MultiNodeReportPath:    "bigclaw-go/docs/reports/multi-node-shared-queue-report.json",
		TakeoverReportPath:     "bigclaw-go/docs/reports/multi-subscriber-takeover-validation-report.json",
		LiveTakeoverReportPath: "bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-report.json",
		OutputPath:             "bigclaw-go/docs/reports/cross-process-coordination-capability-surface.json",
	})
	if err != nil {
		t.Fatalf("coordination surface: %v", err)
	}
	summary := report["summary"].(map[string]any)
	if summary["shared_queue_total_tasks"] != 200 || summary["takeover_passing_scenarios"] != 4 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	capabilities := report["capabilities"].([]any)
	if len(capabilities) < 6 {
		t.Fatalf("expected capability rows, got %+v", capabilities)
	}
}

func TestAutomationMixedWorkloadMatrixWritesReport(t *testing.T) {
	var mu sync.Mutex
	routed := map[string]string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/healthz":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		case r.Method == http.MethodPost && r.URL.Path == "/tasks":
			var task map[string]any
			if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
				t.Fatalf("decode task: %v", err)
			}
			taskID, _ := task["id"].(string)
			executor := "local"
			switch {
			case strings.Contains(taskID, "browser"), strings.Contains(taskID, "risk"):
				executor = "kubernetes"
			case strings.Contains(taskID, "gpu"), strings.Contains(taskID, "required-ray"):
				executor = "ray"
			}
			mu.Lock()
			routed[taskID] = executor
			mu.Unlock()
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{"task": task})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/tasks/"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    strings.TrimPrefix(r.URL.Path, "/tasks/"),
				"state": "succeeded",
				"latest_event": map[string]any{
					"type": "task.completed",
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/events":
			taskID := r.URL.Query().Get("task_id")
			mu.Lock()
			executor := routed[taskID]
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{
				"events": []map[string]any{
					{
						"type":      "scheduler.routed",
						"timestamp": "2026-03-28T00:00:00Z",
						"payload": map[string]any{
							"executor": executor,
							"reason":   "test route",
						},
					},
					{
						"type":      "task.completed",
						"timestamp": "2026-03-28T00:00:01Z",
					},
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	root := t.TempDir()
	reportPath := "docs/reports/mixed-workload-matrix-report.json"
	report, exitCode, err := automationMixedWorkloadMatrix(automationMixedWorkloadMatrixOptions{
		BaseURL:        server.URL,
		GoRoot:         root,
		ReportPath:     reportPath,
		TimeoutSeconds: 1,
		Autostart:      false,
		HTTPClient:     server.Client(),
		Now:            func() time.Time { return time.Unix(1711584000, 0).UTC() },
		Sleep:          func(time.Duration) {},
	})
	if err != nil {
		t.Fatalf("run mixed workload matrix: %v", err)
	}
	if exitCode != 0 || !report.AllOK || len(report.Tasks) != 5 {
		t.Fatalf("unexpected report: exit=%d report=%+v", exitCode, report)
	}
	body, err := os.ReadFile(filepath.Join(root, reportPath))
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(body), "\"all_ok\": true") {
		t.Fatalf("unexpected report body: %s", string(body))
	}
}
