package reporting

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

func TestRunMixedWorkloadMatrix(t *testing.T) {
	taskPolls := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/healthz":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		case r.Method == http.MethodPost && r.URL.Path == "/tasks":
			var task map[string]any
			if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
				t.Fatalf("decode task: %v", err)
			}
			taskPolls[asString(task["id"])] = 0
			_ = json.NewEncoder(w).Encode(map[string]any{"accepted": true})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/tasks/"):
			taskID := filepath.Base(r.URL.Path)
			taskPolls[taskID]++
			expected := expectedMixedExecutor(taskID)
			state := "running"
			if taskPolls[taskID] > 1 {
				state = "succeeded"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":           taskID,
				"state":        state,
				"latest_event": map[string]any{"type": "task.completed"},
				"executor":     expected,
			})
		case r.Method == http.MethodGet && r.URL.Path == "/events":
			taskID := r.URL.Query().Get("task_id")
			expected := expectedMixedExecutor(taskID)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"events": []map[string]any{
					{"id": taskID + "-1", "type": "task.started"},
					{"id": taskID + "-2", "type": "scheduler.routed", "payload": map[string]any{"executor": expected, "reason": "test-route"}},
					{"id": taskID + "-3", "type": "task.completed"},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	root := t.TempDir()
	report, exitCode, err := RunMixedWorkloadMatrix(MixedWorkloadMatrixOptions{
		GoRoot:         root,
		ReportPath:     "docs/reports/mixed-workload-matrix-report.json",
		TimeoutSeconds: 5,
		Autostart:      false,
		BaseURL:        server.URL,
		TimeNow: func() time.Time {
			return time.Unix(1712360000, 0).UTC()
		},
	})
	if err != nil {
		t.Fatalf("run mixed workload matrix: %v", err)
	}
	if exitCode != 0 || !asBool(report["all_ok"]) {
		t.Fatalf("unexpected matrix result: exit=%d report=%+v", exitCode, report)
	}
	tasks := anyToMapSlice(report["tasks"])
	if len(tasks) != 5 {
		t.Fatalf("unexpected task count: %+v", tasks)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/reports/mixed-workload-matrix-report.json")); err != nil {
		t.Fatalf("expected written report: %v", err)
	}
}

func expectedMixedExecutor(taskID string) string {
	switch {
	case strings.Contains(taskID, "browser"), strings.Contains(taskID, "risk"):
		return "kubernetes"
	case strings.Contains(taskID, "gpu"), strings.Contains(taskID, "required-ray"):
		return "ray"
	default:
		return "local"
	}
}
