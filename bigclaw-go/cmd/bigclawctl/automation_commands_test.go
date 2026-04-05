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

func TestBenchmarkScriptsStayGoOnly(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	goRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	benchmarkDir := filepath.Join(goRoot, "scripts", "benchmark")
	entries, err := os.ReadDir(benchmarkDir)
	if err != nil {
		t.Fatalf("read benchmark script directory: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected benchmark wrapper files in %s", benchmarkDir)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".py") {
			t.Fatalf("benchmark directory must not contain Python helpers: %s", entry.Name())
		}
	}
	body, err := os.ReadFile(filepath.Join(benchmarkDir, "run_suite.sh"))
	if err != nil {
		t.Fatalf("read benchmark wrapper: %v", err)
	}
	wrapper := string(body)
	for _, want := range []string{
		"go test -bench . ./internal/queue ./internal/scheduler",
		"go run ./cmd/bigclawctl automation benchmark run-matrix",
	} {
		if !strings.Contains(wrapper, want) {
			t.Fatalf("benchmark wrapper missing %q: %s", want, wrapper)
		}
	}
}

func TestAutomationUsageListsBIGGO1160GoReplacements(t *testing.T) {
	cases := []struct {
		args    []string
		needles []string
	}{
		{
			args: []string{"e2e"},
			needles: []string{
				"run-task-smoke",
				"run-all",
				"export-validation-bundle",
				"continuation-scorecard",
				"continuation-policy-gate",
				"broker-failover-stub-matrix",
				"mixed-workload-matrix",
				"cross-process-coordination-surface",
				"subscriber-takeover-fault-matrix",
				"external-store-validation",
				"multi-node-shared-queue",
			},
		},
		{
			args: []string{"benchmark"},
			needles: []string{
				"soak-local",
				"run-matrix",
				"capacity-certification",
			},
		},
		{
			args: []string{"migration"},
			needles: []string{
				"shadow-compare",
				"shadow-matrix",
				"live-shadow-scorecard",
				"export-live-shadow-bundle",
			},
		},
	}

	for _, tc := range cases {
		output, err := captureStdout(t, func() error {
			return runAutomation(tc.args)
		})
		if err != nil {
			t.Fatalf("run automation usage for %v: %v", tc.args, err)
		}
		usage := string(output)
		for _, needle := range tc.needles {
			if !strings.Contains(usage, needle) {
				t.Fatalf("automation usage for %v missing %q: %s", tc.args, needle, usage)
			}
		}
	}
}

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

func TestAutomationShadowMatrixBuildsCorpusCoverage(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	})
	primary := httptest.NewServer(handler)
	defer primary.Close()
	shadow := httptest.NewServer(handler)
	defer shadow.Close()

	root := t.TempDir()
	fixturePath := filepath.Join(root, "task.json")
	if err := os.WriteFile(fixturePath, []byte(`{"id":"compare","title":"compare","entrypoint":"echo hi","execution_timeout_seconds":1,"required_executor":"local"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, "manifest.json")
	manifest := `{
  "name": "anonymized-production-corpus-v1",
  "generated_at": "2026-03-28T00:00:00Z",
  "slices": [
    {"slice_id": "fixture-covered", "title": "fixture covered", "weight": 3, "task_shape": "executor:local"},
    {"slice_id": "browser-human-review", "title": "browser human review", "weight": 2, "task_shape": "executor:browser"}
  ]
}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	reportPath := filepath.Join(root, "shadow-matrix-report.json")

	report, exitCode, err := automationShadowMatrix(automationShadowMatrixOptions{
		PrimaryBaseURL:       primary.URL,
		ShadowBaseURL:        shadow.URL,
		TaskFiles:            []string{fixturePath},
		CorpusManifest:       manifestPath,
		TimeoutSeconds:       1,
		HealthTimeoutSeconds: 1,
		ReportPath:           reportPath,
		HTTPClient:           primary.Client(),
		Sleep:                func(time.Duration) {},
	})
	if err != nil {
		t.Fatalf("run shadow matrix: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("expected success exit code, got %d", exitCode)
	}
	if report["matched"] != 1 || report["mismatched"] != 0 {
		t.Fatalf("unexpected matrix summary: %+v", report)
	}
	coverage, ok := report["corpus_coverage"].(map[string]any)
	if !ok || coverage["manifest_name"] != "anonymized-production-corpus-v1" || coverage["uncovered_corpus_slice_count"] != 1 {
		t.Fatalf("unexpected corpus coverage: %+v", coverage)
	}
	body, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(body), "\"browser-human-review\"") {
		t.Fatalf("expected uncovered slice in report, got %s", string(body))
	}
}

func TestAutomationLiveShadowScorecardBuildsReport(t *testing.T) {
	root := t.TempDir()
	comparePath := filepath.Join(root, "shadow-compare-report.json")
	matrixPath := filepath.Join(root, "shadow-matrix-report.json")
	outputPath := filepath.Join(root, "live-shadow-mirror-scorecard.json")
	compare := `{
  "trace_id": "shadow-compare-sample",
  "primary": {"task_id": "primary-1", "events": [{"type":"queued","timestamp":"2026-03-28T00:00:00Z"},{"type":"succeeded","timestamp":"2026-03-28T00:00:03Z"}]},
  "shadow": {"task_id": "shadow-1", "events": [{"type":"queued","timestamp":"2026-03-28T00:00:00Z"},{"type":"succeeded","timestamp":"2026-03-28T00:00:03Z"}]},
  "diff": {"state_equal": true, "event_types_equal": true, "event_count_delta": 0, "primary_timeline_seconds": 3.0, "shadow_timeline_seconds": 3.0}
}`
	matrix := `{
  "total": 1,
  "matched": 1,
  "mismatched": 0,
  "corpus_coverage": {"uncovered_corpus_slice_count": 1, "corpus_slice_count": 2},
  "results": [{
    "trace_id": "shadow-compare-sample-m1",
    "source_file": "task.json",
    "source_kind": "fixture",
    "task_shape": "executor:local",
    "primary": {"task_id": "primary-m1", "events": [{"type":"queued","timestamp":"2026-03-28T00:00:00Z"},{"type":"succeeded","timestamp":"2026-03-28T00:00:03Z"}]},
    "shadow": {"task_id": "shadow-m1", "events": [{"type":"queued","timestamp":"2026-03-28T00:00:00Z"},{"type":"succeeded","timestamp":"2026-03-28T00:00:03Z"}]},
    "diff": {"state_equal": true, "event_types_equal": true, "event_count_delta": 0, "primary_timeline_seconds": 3.0, "shadow_timeline_seconds": 3.0}
  }]
}`
	for path, body := range map[string]string{comparePath: compare, matrixPath: matrix} {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	report, err := automationLiveShadowScorecard(automationLiveShadowScorecardOptions{
		ShadowCompareReportPath: comparePath,
		ShadowMatrixReportPath:  matrixPath,
		OutputPath:              outputPath,
		Now:                     func() time.Time { return time.Date(2026, 3, 28, 6, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("build live shadow scorecard: %v", err)
	}
	summary, _ := report["summary"].(map[string]any)
	if summary["total_evidence_runs"] != 2 || summary["parity_ok_count"] != 2 || summary["stale_inputs"] != 0 {
		t.Fatalf("unexpected scorecard summary: %+v", summary)
	}
	inputs, _ := report["evidence_inputs"].(map[string]any)
	if inputs["generator_script"] != "go run ./cmd/bigclawctl automation migration live-shadow-scorecard" {
		t.Fatalf("unexpected evidence inputs: %+v", inputs)
	}
	body, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if !strings.Contains(string(body), "\"repo-native-live-shadow-scorecard\"") {
		t.Fatalf("unexpected output body: %s", string(body))
	}
}

func TestAutomationExportLiveShadowBundleBuildsManifest(t *testing.T) {
	root := t.TempDir()
	for path, body := range map[string]string{
		filepath.Join(root, "docs/reports/shadow-compare-report.json"):        `{"trace_id":"shadow-compare-sample","primary":{"task_id":"primary-1","events":[{"type":"queued","timestamp":"2026-03-13T07:53:21Z"},{"type":"succeeded","timestamp":"2026-03-13T07:53:24Z"}]},"shadow":{"task_id":"shadow-1","events":[{"type":"queued","timestamp":"2026-03-13T07:53:21Z"},{"type":"succeeded","timestamp":"2026-03-13T07:53:24Z"}]},"diff":{"state_equal":true,"event_types_equal":true,"event_count_delta":0,"primary_timeline_seconds":3.0,"shadow_timeline_seconds":3.0}}`,
		filepath.Join(root, "docs/reports/shadow-matrix-report.json"):         `{"total":1,"matched":1,"mismatched":0,"results":[{"trace_id":"shadow-compare-sample-m1","primary":{"task_id":"primary-m1","events":[{"type":"queued","timestamp":"2026-03-13T08:56:55Z"},{"type":"succeeded","timestamp":"2026-03-13T08:56:58Z"}]},"shadow":{"task_id":"shadow-m1","events":[{"type":"queued","timestamp":"2026-03-13T08:56:55Z"},{"type":"succeeded","timestamp":"2026-03-13T08:56:58Z"}]},"diff":{"state_equal":true,"event_types_equal":true,"event_count_delta":0,"primary_timeline_seconds":3.0,"shadow_timeline_seconds":3.0}}]}`,
		filepath.Join(root, "docs/reports/live-shadow-mirror-scorecard.json"): `{"summary":{"total_evidence_runs":2,"parity_ok_count":2,"drift_detected_count":0,"matrix_total":1,"matrix_mismatched":0,"stale_inputs":0,"fresh_inputs":2,"latest_evidence_timestamp":"2026-03-13T08:56:55Z"},"freshness":[{"name":"shadow-compare-report","status":"fresh","report_path":"bigclaw-go/docs/reports/shadow-compare-report.json"},{"name":"shadow-matrix-report","status":"fresh","report_path":"bigclaw-go/docs/reports/shadow-matrix-report.json"}],"cutover_checkpoints":[{"name":"checkpoint","passed":true,"detail":"ok"}]}`,
		filepath.Join(root, "docs/reports/rollback-trigger-surface.json"):     `{"summary":{"status":"manual-review-required","automation_boundary":"manual_only","automated_rollback_trigger":false,"distinctions":{"blockers":3,"warnings":1,"manual_only_paths":2}},"issue":{"id":"OPE-254","slug":"BIG-PAR-088"},"digest_path":"docs/reports/rollback-safeguard-follow-up-digest.md"}`,
		filepath.Join(root, "docs/migration-shadow.md"):                       `shadow docs`,
		filepath.Join(root, "docs/reports/migration-readiness-report.md"):     `readiness docs`,
		filepath.Join(root, "docs/reports/migration-plan-review-notes.md"):    `review notes`,
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	manifest, err := automationExportLiveShadowBundle(automationExportLiveShadowBundleOptions{
		GoRoot:            root,
		ShadowComparePath: "docs/reports/shadow-compare-report.json",
		ShadowMatrixPath:  "docs/reports/shadow-matrix-report.json",
		ScorecardPath:     "docs/reports/live-shadow-mirror-scorecard.json",
		BundleRoot:        "docs/reports/live-shadow-runs",
		SummaryPath:       "docs/reports/live-shadow-summary.json",
		IndexPath:         "docs/reports/live-shadow-index.md",
		ManifestPath:      "docs/reports/live-shadow-index.json",
		RollupPath:        "docs/reports/live-shadow-drift-rollup.json",
		Now:               func() time.Time { return time.Date(2026, 3, 17, 2, 35, 33, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("export live shadow bundle: %v", err)
	}
	latest, _ := manifest["latest"].(map[string]any)
	if latest["run_id"] != "20260313T085655Z" {
		t.Fatalf("unexpected latest run id: %+v", latest)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/reports/live-shadow-runs/20260313T085655Z/README.md")); err != nil {
		t.Fatalf("expected bundled README: %v", err)
	}
	indexBody, err := os.ReadFile(filepath.Join(root, "docs/reports/live-shadow-index.md"))
	if err != nil {
		t.Fatalf("read index: %v", err)
	}
	if !strings.Contains(string(indexBody), "export-live-shadow-bundle") {
		t.Fatalf("unexpected index body: %s", string(indexBody))
	}
}

func TestAutomationBenchmarkRunMatrixBuildsReport(t *testing.T) {
	root := t.TempDir()
	report, err := automationBenchmarkRunMatrix(automationBenchmarkRunMatrixOptions{
		GoRoot:         root,
		ReportPath:     "docs/reports/benchmark-matrix-report.json",
		TimeoutSeconds: 123,
		Scenarios:      []string{"3:2", "5:4"},
		RunBenchmark: func(string) (string, error) {
			return "BenchmarkAlpha-8    \t100\t123.45 ns/op\nBenchmarkBeta-8    \t200\t678.90 ns/op\n", nil
		},
		RunSoak: func(_ string, count, workers, timeoutSeconds int, reportPath string) (map[string]any, error) {
			return map[string]any{
				"count":                    count,
				"workers":                  workers,
				"elapsed_seconds":          1.25,
				"throughput_tasks_per_sec": 9.5,
				"succeeded":                count,
				"failed":                   0,
				"timeout_seconds":          timeoutSeconds,
				"report_path":              reportPath,
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("run benchmark matrix: %v", err)
	}
	benchmark, _ := report["benchmark"].(map[string]any)
	parsed, _ := benchmark["parsed"].(map[string]any)
	if len(parsed) != 2 {
		t.Fatalf("expected parsed benchmarks, got %+v", parsed)
	}
	soakMatrix, _ := report["soak_matrix"].([]any)
	if len(soakMatrix) != 2 {
		t.Fatalf("expected 2 soak scenarios, got %+v", soakMatrix)
	}
	body, err := os.ReadFile(filepath.Join(root, "docs/reports/benchmark-matrix-report.json"))
	if err != nil {
		t.Fatalf("read matrix report: %v", err)
	}
	if !strings.Contains(string(body), "\"report_path\": \"docs/reports/soak-local-3x2.json\"") {
		t.Fatalf("unexpected report body: %s", string(body))
	}
}

func TestAutomationBenchmarkCapacityCertificationBuildsReport(t *testing.T) {
	root := t.TempDir()
	for path, body := range map[string]string{
		filepath.Join(root, "docs/reports/benchmark-matrix-report.json"): `{
  "benchmark": {
    "parsed": {
      "BenchmarkMemoryQueueEnqueueLease-8": {"ns_per_op": 66075.0},
      "BenchmarkFileQueueEnqueueLease-8": {"ns_per_op": 31627767.0},
      "BenchmarkSQLiteQueueEnqueueLease-8": {"ns_per_op": 18057898.0},
      "BenchmarkSchedulerDecide-8": {"ns_per_op": 73.98}
    }
  },
  "soak_matrix": [
    {"report_path": "docs/reports/soak-local-50x8.json", "result": {"count": 50, "workers": 8, "elapsed_seconds": 5.0, "throughput_tasks_per_sec": 10.0, "succeeded": 50, "failed": 0, "generated_at": "2026-03-13T09:44:00Z"}},
    {"report_path": "docs/reports/soak-local-100x12.json", "result": {"count": 100, "workers": 12, "elapsed_seconds": 10.0, "throughput_tasks_per_sec": 9.6, "succeeded": 100, "failed": 0, "generated_at": "2026-03-13T09:44:20Z"}}
  ]
}`,
		filepath.Join(root, "docs/reports/mixed-workload-matrix-report.json"): `{
  "all_ok": true,
  "generated_at": "2026-03-13T09:44:42.458392Z",
  "tasks": [
    {"name": "local-a", "ok": true, "expected_executor": "local", "routed_executor": "local", "final_state": "succeeded"},
    {"name": "local-b", "ok": true, "expected_executor": "local", "routed_executor": "local", "final_state": "succeeded"},
    {"name": "k8s-a", "ok": true, "expected_executor": "kubernetes", "routed_executor": "kubernetes", "final_state": "succeeded"},
    {"name": "ray-a", "ok": true, "expected_executor": "ray", "routed_executor": "ray", "final_state": "succeeded"},
    {"name": "ray-b", "ok": true, "expected_executor": "ray", "routed_executor": "ray", "final_state": "succeeded"}
  ]
}`,
		filepath.Join(root, "docs/reports/soak-local-1000x24.json"): `{"count":1000,"workers":24,"elapsed_seconds":100.0,"throughput_tasks_per_sec":9.8,"succeeded":1000,"failed":0,"generated_at":"2026-03-13T09:44:30Z"}`,
		filepath.Join(root, "docs/reports/soak-local-2000x24.json"): `{"count":2000,"workers":24,"elapsed_seconds":205.0,"throughput_tasks_per_sec":9.1,"succeeded":2000,"failed":0,"generated_at":"2026-03-13T09:44:40Z"}`,
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	report, err := automationBenchmarkCapacityCertification(automationBenchmarkCapacityCertificationOptions{
		GoRoot:                  root,
		OutputPath:              "docs/reports/capacity-certification-matrix.json",
		MarkdownOutputPath:      "docs/reports/capacity-certification-report.md",
		BenchmarkReportPath:     "docs/reports/benchmark-matrix-report.json",
		MixedWorkloadReportPath: "docs/reports/mixed-workload-matrix-report.json",
	})
	if err != nil {
		t.Fatalf("build capacity certification: %v", err)
	}
	summary, _ := report["summary"].(map[string]any)
	if summary["overall_status"] != "pass" {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if report["generated_at"] != "2026-03-13T09:44:42.458392Z" {
		t.Fatalf("unexpected generated_at: %+v", report["generated_at"])
	}
	body, err := os.ReadFile(filepath.Join(root, "docs/reports/capacity-certification-matrix.json"))
	if err != nil {
		t.Fatalf("read certification json: %v", err)
	}
	if strings.Contains(string(body), "\"markdown\"") {
		t.Fatalf("json output should not embed markdown: %s", string(body))
	}
	if !strings.Contains(string(body), "automation benchmark capacity-certification") {
		t.Fatalf("unexpected certification json: %s", string(body))
	}
	markdownBody, err := os.ReadFile(filepath.Join(root, "docs/reports/capacity-certification-report.md"))
	if err != nil {
		t.Fatalf("read certification markdown: %v", err)
	}
	if !strings.Contains(string(markdownBody), "Runtime enforcement: `none`") {
		t.Fatalf("unexpected markdown output: %s", string(markdownBody))
	}
}
