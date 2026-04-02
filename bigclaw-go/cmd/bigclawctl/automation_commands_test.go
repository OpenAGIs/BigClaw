package main

import (
	"encoding/json"
	"fmt"
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

func TestRunAutomationE2EHelpIncludesContinuationCommands(t *testing.T) {
	output, err := captureStdout(t, func() error {
		return runAutomation([]string{"e2e", "--help"})
	})
	if err != nil {
		t.Fatalf("e2e help: %v", err)
	}
	text := string(output)
	if !strings.Contains(text, "run-task-smoke|continuation-scorecard|continuation-policy-gate") {
		t.Fatalf("unexpected e2e help: %s", text)
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

func TestRunAutomationBenchmarkHelpIncludesMigratedCommands(t *testing.T) {
	output, err := captureStdout(t, func() error {
		return runAutomation([]string{"benchmark", "--help"})
	})
	if err != nil {
		t.Fatalf("benchmark help: %v", err)
	}
	text := string(output)
	if !strings.Contains(text, "soak-local|run-matrix|capacity-certification") {
		t.Fatalf("unexpected benchmark help: %s", text)
	}
}

func TestAutomationBenchmarkRunMatrixWritesReport(t *testing.T) {
	goRoot := t.TempDir()
	reportPath := "docs/reports/benchmark-matrix-report.json"
	report, err := automationBenchmarkRunMatrix(automationBenchmarkRunMatrixOptions{
		GoRoot:         goRoot,
		ReportPath:     reportPath,
		TimeoutSeconds: 30,
		Scenarios:      []string{"50:8"},
		RunBenchmarks: func(string) (string, error) {
			return "BenchmarkMemoryQueueEnqueueLease-8\t1\t66075 ns/op\nBenchmarkSchedulerDecide-8\t1\t73.98 ns/op\n", nil
		},
		RunSoak: func(opts automationSoakLocalOptions) (*automationSoakLocalReport, int, error) {
			if opts.Count != 50 || opts.Workers != 8 || !opts.Autostart {
				t.Fatalf("unexpected soak options: %+v", opts)
			}
			return &automationSoakLocalReport{
				Count:                 50,
				Workers:               8,
				ElapsedSeconds:        8.232,
				ThroughputTasksPerSec: 6.074,
				Succeeded:             50,
				Failed:                0,
			}, 0, nil
		},
	})
	if err != nil {
		t.Fatalf("run benchmark matrix: %v", err)
	}
	parsed, _ := lookupMap(report, "benchmark", "parsed").(map[string]any)
	if asFloat(lookupMap(parsed, "BenchmarkMemoryQueueEnqueueLease-8", "ns_per_op")) != 66075 {
		t.Fatalf("unexpected parsed benchmark payload: %+v", report)
	}
	entries, _ := report["soak_matrix"].([]map[string]any)
	if len(entries) != 1 {
		rawEntries, _ := report["soak_matrix"].([]any)
		if len(rawEntries) != 1 {
			t.Fatalf("unexpected soak matrix entries: %+v", report["soak_matrix"])
		}
	}
	body, err := os.ReadFile(filepath.Join(goRoot, reportPath))
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(body), "\"report_path\": \"docs/reports/soak-local-50x8.json\"") {
		t.Fatalf("unexpected report body: %s", string(body))
	}
}

func TestAutomationCapacityCertificationMatchesCheckedInEvidence(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	outputPath := filepath.Join(t.TempDir(), "capacity-certification-matrix.json")
	markdownPath := filepath.Join(t.TempDir(), "capacity-certification-report.md")
	report, err := automationCapacityCertification(automationCapacityCertificationOptions{
		BenchmarkReportPath:        filepath.Join(repoRoot, "docs", "reports", "benchmark-matrix-report.json"),
		MixedWorkloadReportPath:    filepath.Join(repoRoot, "docs", "reports", "mixed-workload-matrix-report.json"),
		SupplementalSoakReportPath: []string{filepath.Join(repoRoot, "docs", "reports", "soak-local-1000x24.json"), filepath.Join(repoRoot, "docs", "reports", "soak-local-2000x24.json")},
		OutputPath:                 outputPath,
		MarkdownOutputPath:         markdownPath,
		Now:                        func() time.Time { return time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("build capacity certification: %v", err)
	}
	summary, _ := report["summary"].(map[string]any)
	if summary["overall_status"] != "pass" {
		t.Fatalf("unexpected overall status: %+v", report)
	}
	if report["generated_at"] != "2026-03-13T09:44:42.458392Z" {
		t.Fatalf("unexpected generated_at: %+v", report["generated_at"])
	}
	evidenceInputs, _ := report["evidence_inputs"].(map[string]any)
	if evidenceInputs["generator_script"] != "go run ./cmd/bigclawctl automation benchmark capacity-certification" {
		t.Fatalf("unexpected generator script: %+v", evidenceInputs)
	}
	if !strings.Contains(fmt.Sprint(report["markdown"]), "## Admission Policy Summary") ||
		!strings.Contains(fmt.Sprint(report["markdown"]), "Runtime enforcement: `none`") {
		t.Fatalf("unexpected markdown: %v", report["markdown"])
	}
	body, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output json: %v", err)
	}
	if !strings.Contains(string(body), "\"repo-native-capacity-certification\"") {
		t.Fatalf("unexpected output body: %s", string(body))
	}
	markdownBody, err := os.ReadFile(markdownPath)
	if err != nil {
		t.Fatalf("read markdown output: %v", err)
	}
	if !strings.Contains(string(markdownBody), "# Capacity Certification Report") {
		t.Fatalf("unexpected markdown output: %s", string(markdownBody))
	}
}

func TestAutomationContinuationScorecardMatchesCheckedInEvidence(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	goRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	outputPath := filepath.Join(t.TempDir(), "validation-bundle-continuation-scorecard.json")
	report, err := automationContinuationScorecard(automationContinuationScorecardOptions{
		GoRoot:                goRoot,
		IndexManifestPath:     "bigclaw-go/docs/reports/live-validation-index.json",
		BundleRootPath:        "bigclaw-go/docs/reports/live-validation-runs",
		LatestSummaryPath:     "bigclaw-go/docs/reports/live-validation-summary.json",
		SharedQueueReportPath: "bigclaw-go/docs/reports/multi-node-shared-queue-report.json",
		OutputPath:            outputPath,
		Now:                   func() time.Time { return time.Date(2026, 3, 17, 4, 32, 49, 251910000, time.UTC) },
	})
	if err != nil {
		t.Fatalf("build continuation scorecard: %v", err)
	}
	if report["ticket"] != "BIG-PAR-086-local-prework" || report["status"] != "local-continuation-scorecard" {
		t.Fatalf("unexpected continuation scorecard identity: %+v", report)
	}
	evidenceInputs, _ := report["evidence_inputs"].(map[string]any)
	if evidenceInputs["generator_script"] != "go run ./cmd/bigclawctl automation e2e continuation-scorecard" {
		t.Fatalf("unexpected generator script: %+v", evidenceInputs)
	}
	summary, _ := report["summary"].(map[string]any)
	if automationInt(summary["recent_bundle_count"], 0) != 3 ||
		fmt.Sprint(summary["latest_run_id"]) != "20260316T140138Z" ||
		!automationBool(summary["latest_all_executor_tracks_succeeded"]) ||
		!automationBool(summary["recent_bundle_chain_has_no_failures"]) ||
		!automationBool(summary["all_executor_tracks_have_repeated_recent_coverage"]) {
		t.Fatalf("unexpected continuation scorecard summary: %+v", summary)
	}
	if asFloat(summary["latest_bundle_age_hours"]) <= 0 || asFloat(summary["latest_bundle_age_hours"]) >= 1 {
		t.Fatalf("unexpected continuation scorecard age: %+v", summary)
	}
	sharedQueue, _ := report["shared_queue_companion"].(map[string]any)
	if !automationBool(sharedQueue["available"]) ||
		automationInt(sharedQueue["cross_node_completions"], 0) != 99 ||
		automationInt(sharedQueue["duplicate_completed_tasks"], 0) != 0 ||
		fmt.Sprint(sharedQueue["mode"]) != "bundle-companion-summary" {
		t.Fatalf("unexpected continuation shared queue companion: %+v", sharedQueue)
	}
	body, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read continuation scorecard output: %v", err)
	}
	if !strings.Contains(string(body), "\"go run ./cmd/bigclawctl automation e2e continuation-scorecard\"") {
		t.Fatalf("unexpected continuation scorecard output: %s", string(body))
	}
}

func TestAutomationContinuationPolicyGateMatchesCheckedInEvidence(t *testing.T) {
	scorecardPath := filepath.Join(t.TempDir(), "validation-bundle-continuation-scorecard.json")
	if err := os.WriteFile(scorecardPath, []byte(`{
  "summary": {
    "latest_run_id": "20260316T140138Z",
    "latest_bundle_age_hours": 0.01,
    "recent_bundle_count": 3,
    "latest_all_executor_tracks_succeeded": true,
    "recent_bundle_chain_has_no_failures": true,
    "all_executor_tracks_have_repeated_recent_coverage": true
  },
  "shared_queue_companion": {
    "available": true,
    "cross_node_completions": 99,
    "duplicate_completed_tasks": 0,
    "duplicate_started_tasks": 0,
    "mode": "bundle-companion-summary"
  }
}`), 0o644); err != nil {
		t.Fatalf("write scorecard fixture: %v", err)
	}
	outputPath := filepath.Join(t.TempDir(), "validation-bundle-continuation-policy-gate.json")
	report, exitCode, err := automationContinuationPolicyGate(automationContinuationPolicyGateOptions{
		GoRoot:                      filepath.Join("..", ".."),
		ScorecardPath:               scorecardPath,
		OutputPath:                  outputPath,
		MaxLatestAgeHours:           72,
		MinRecentBundles:            2,
		RequireRepeatedLaneCoverage: true,
		EnforcementMode:             "review",
		Now:                         func() time.Time { return time.Date(2026, 3, 17, 4, 32, 49, 251910000, time.UTC) },
	})
	if err != nil {
		t.Fatalf("build continuation policy gate: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("unexpected continuation policy gate exit code: %d", exitCode)
	}
	if report["ticket"] != "OPE-262" || report["status"] != "policy-go" || report["recommendation"] != "go" {
		t.Fatalf("unexpected continuation policy gate identity: %+v", report)
	}
	evidenceInputs, _ := report["evidence_inputs"].(map[string]any)
	if evidenceInputs["generator_script"] != "go run ./cmd/bigclawctl automation e2e continuation-policy-gate" {
		t.Fatalf("unexpected generator script: %+v", evidenceInputs)
	}
	enforcement, _ := report["enforcement"].(map[string]any)
	if enforcement["mode"] != "review" || enforcement["outcome"] != "pass" || automationInt(enforcement["exit_code"], 0) != 0 {
		t.Fatalf("unexpected continuation policy gate enforcement: %+v", enforcement)
	}
	reviewerPath, _ := report["reviewer_path"].(map[string]any)
	digestIssue, _ := reviewerPath["digest_issue"].(map[string]any)
	if reviewerPath["index_path"] != "docs/reports/live-validation-index.md" ||
		reviewerPath["digest_path"] != "docs/reports/validation-bundle-continuation-digest.md" ||
		digestIssue["id"] != "OPE-271" ||
		digestIssue["slug"] != "BIG-PAR-082" {
		t.Fatalf("unexpected continuation policy gate reviewer path: %+v", reviewerPath)
	}
	body, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read continuation policy gate output: %v", err)
	}
	if !strings.Contains(string(body), "\"go run ./cmd/bigclawctl automation e2e continuation-policy-gate\"") {
		t.Fatalf("unexpected continuation policy gate output: %s", string(body))
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
