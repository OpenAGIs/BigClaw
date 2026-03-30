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
