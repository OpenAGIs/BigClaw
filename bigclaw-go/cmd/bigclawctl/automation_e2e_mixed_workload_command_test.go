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

func TestAutomationMixedWorkloadMatrixBuildsReport(t *testing.T) {
	type taskState struct {
		Expected string
	}
	tasks := map[string]taskState{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/healthz":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		case r.Method == http.MethodPost && r.URL.Path == "/tasks":
			var task map[string]any
			if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
				t.Fatalf("decode task: %v", err)
			}
			id := task["id"].(string)
			expected := "local"
			switch {
			case strings.Contains(id, "browser"), strings.Contains(id, "risk"):
				expected = "kubernetes"
			case strings.Contains(id, "gpu"), strings.Contains(id, "required-ray"):
				expected = "ray"
			}
			tasks[id] = taskState{Expected: expected}
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{"task": task})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/tasks/"):
			taskID := strings.TrimPrefix(r.URL.Path, "/tasks/")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    taskID,
				"state": "succeeded",
				"latest_event": map[string]any{
					"type": "task.completed",
				},
			})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/events"):
			taskID := r.URL.Query().Get("task_id")
			expected := tasks[taskID].Expected
			_ = json.NewEncoder(w).Encode(map[string]any{
				"events": []map[string]any{
					{
						"type": "scheduler.routed",
						"payload": map[string]any{
							"executor": expected,
							"reason":   "stub route",
						},
					},
					{
						"type": "task.completed",
					},
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	root := t.TempDir()
	report, exitCode, err := automationMixedWorkloadMatrix(automationMixedWorkloadMatrixOptions{
		GoRoot:         root,
		BaseURL:        server.URL,
		ReportPath:     "docs/reports/mixed-workload-matrix-report.json",
		TimeoutSeconds: 2,
		Autostart:      false,
		HTTPClient:     server.Client(),
		Now:            func() time.Time { return time.Date(2026, 3, 30, 14, 40, 0, 0, time.UTC) },
		Sleep:          func(time.Duration) {},
	})
	if err != nil {
		t.Fatalf("run mixed workload matrix: %v", err)
	}
	if exitCode != 0 || report["all_ok"] != true {
		t.Fatalf("unexpected report: exit=%d report=%+v", exitCode, report)
	}
	tasksOut, _ := report["tasks"].([]any)
	if len(tasksOut) != 5 {
		t.Fatalf("expected 5 tasks, got %+v", tasksOut)
	}
	body, err := os.ReadFile(filepath.Join(root, "docs/reports/mixed-workload-matrix-report.json"))
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(body), "\"all_ok\": true") || !strings.Contains(string(body), "\"routed_executor\": \"ray\"") {
		t.Fatalf("unexpected report body: %s", string(body))
	}
}

func TestDefaultMixedWorkloadTasksUseNoPythonEntrypoints(t *testing.T) {
	for _, spec := range defaultMixedWorkloadTasks(1234) {
		if strings.Contains(spec.Task.Entrypoint, "python") {
			t.Fatalf("expected mixed workload default entrypoint to avoid python helper residue: %+v", spec)
		}
	}
}
