package regression

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMixedWorkloadMatrixScriptBuildsExpectedReport(t *testing.T) {
	repoRoot := repoRoot(t)
	taskExecutors := map[string]string{
		"mixed-local-":        "local",
		"mixed-browser-":      "kubernetes",
		"mixed-gpu-":          "ray",
		"mixed-risk-":         "kubernetes",
		"mixed-required-ray-": "ray",
	}
	taskReasons := map[string]string{
		"local":      "default local executor for low/medium risk",
		"kubernetes": "browser workloads default to kubernetes executor",
		"ray":        "gpu workloads default to ray executor",
	}
	taskTitles := map[string]string{}
	taskStates := map[string]int{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/healthz":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		case r.Method == http.MethodPost && r.URL.Path == "/tasks":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode task payload: %v", err)
			}
			taskID := payload["id"].(string)
			taskTitles[taskID] = payload["title"].(string)
			taskStates[taskID] = 0
			_ = json.NewEncoder(w).Encode(map[string]any{"id": taskID})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/tasks/"):
			taskID := strings.TrimPrefix(r.URL.Path, "/tasks/")
			executor := expectedExecutor(taskExecutors, taskID)
			taskStates[taskID]++
			state := "running"
			if taskStates[taskID] > 1 {
				state = "succeeded"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    taskID,
				"state": state,
				"latest_event": map[string]any{
					"type": map[string]string{"running": "task.started", "succeeded": "task.completed"}[state],
				},
				"executor": executor,
			})
		case r.Method == http.MethodGet && r.URL.Path == "/events":
			taskID := r.URL.Query().Get("task_id")
			executor := expectedExecutor(taskExecutors, taskID)
			reason := taskReasons[executor]
			if strings.HasPrefix(taskID, "mixed-risk-") {
				reason = "high-risk tasks default to isolated executor"
			}
			if strings.HasPrefix(taskID, "mixed-required-ray-") {
				reason = "required executor requested by task"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"events": []map[string]any{
					{
						"id":        taskID + "-queued",
						"type":      "task.queued",
						"task_id":   taskID,
						"trace_id":  taskID,
						"timestamp": "2026-03-13T17:44:26.107696+08:00",
						"payload": map[string]any{
							"executor": "",
							"title":    taskTitles[taskID],
						},
					},
					{
						"id":        taskID + "-routed",
						"type":      "scheduler.routed",
						"task_id":   taskID,
						"trace_id":  taskID,
						"timestamp": "2026-03-13T17:44:26.195926+08:00",
						"payload": map[string]any{
							"executor": executor,
							"reason":   reason,
						},
					},
					{
						"id":        taskID + "-completed",
						"type":      "task.completed",
						"task_id":   taskID,
						"trace_id":  taskID,
						"timestamp": "2026-03-13T17:44:26.206918+08:00",
						"payload": map[string]any{
							"message": executor + " execution completed",
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	outputPath := filepath.Join(t.TempDir(), "mixed-workload-matrix-report.json")
	scriptPath := filepath.Join(repoRoot, "scripts", "e2e", "mixed-workload-matrix")
	cmd := exec.Command("bash", scriptPath, "--autostart=false", "--report-path", outputPath)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "BIGCLAW_ADDR="+server.URL)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run mixed workload matrix script: %v\n%s", err, output)
	}

	var report struct {
		BaseURL    string `json:"base_url"`
		StateDir   string `json:"state_dir"`
		ServiceLog string `json:"service_log"`
		AllOK      bool   `json:"all_ok"`
		Tasks      []struct {
			Name             string `json:"name"`
			ExpectedExecutor string `json:"expected_executor"`
			RoutedExecutor   string `json:"routed_executor"`
			FinalState       string `json:"final_state"`
			LatestEventType  string `json:"latest_event_type"`
			OK               bool   `json:"ok"`
		} `json:"tasks"`
	}
	readJSONFile(t, outputPath, &report)

	if report.BaseURL != server.URL || report.StateDir != "" || report.ServiceLog != "" {
		t.Fatalf("unexpected report runtime fields: %+v", report)
	}
	if !report.AllOK || len(report.Tasks) != 5 {
		t.Fatalf("unexpected report summary: %+v", report)
	}
	expectedByName := map[string]string{
		"local-default":  "local",
		"browser-auto":   "kubernetes",
		"gpu-auto":       "ray",
		"high-risk-auto": "kubernetes",
		"required-ray":   "ray",
	}
	for _, item := range report.Tasks {
		if item.ExpectedExecutor != expectedByName[item.Name] || item.RoutedExecutor != expectedByName[item.Name] {
			t.Fatalf("unexpected routing for %s: %+v", item.Name, item)
		}
		if item.FinalState != "succeeded" || item.LatestEventType != "task.completed" || !item.OK {
			t.Fatalf("unexpected task outcome for %s: %+v", item.Name, item)
		}
	}
}

func expectedExecutor(prefixes map[string]string, taskID string) string {
	for prefix, executor := range prefixes {
		if strings.HasPrefix(taskID, prefix) {
			return executor
		}
	}
	return ""
}
