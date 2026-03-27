package shadowcompare

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCompareTask(t *testing.T) {
	t.Parallel()

	primary := newShadowTestServer(t, "primary", []map[string]any{
		{"id": "queued", "type": "task.queued", "timestamp": "2026-03-13T15:53:21.384193+08:00"},
		{"id": "leased", "type": "task.leased", "timestamp": "2026-03-13T15:53:21.391943+08:00"},
		{"id": "completed", "type": "task.completed", "timestamp": "2026-03-13T15:53:21.403334+08:00"},
	})
	defer primary.Close()
	shadow := newShadowTestServer(t, "shadow", []map[string]any{
		{"id": "queued", "type": "task.queued", "timestamp": "2026-03-13T15:53:21.385860+08:00"},
		{"id": "leased", "type": "task.leased", "timestamp": "2026-03-13T15:53:21.392426+08:00"},
		{"id": "completed", "type": "task.completed", "timestamp": "2026-03-13T15:53:21.403765+08:00"},
	})
	defer shadow.Close()

	report, err := CompareTask(CompareOptions{
		PrimaryBaseURL:     primary.URL,
		ShadowBaseURL:      shadow.URL,
		Task:               map[string]any{"id": "shadow-compare-sample"},
		Timeout:            time.Second,
		HealthTimeout:      time.Second,
		PollInterval:       time.Millisecond,
		HealthPollInterval: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("CompareTask returned error: %v", err)
	}

	if got := stringValue(report["trace_id"], ""); got != "shadow-compare-sample" {
		t.Fatalf("trace_id = %q, want shadow-compare-sample", got)
	}
	if got := stringValue(nestedMap(report, "primary")["task_id"], ""); got != "shadow-compare-sample-primary" {
		t.Fatalf("primary.task_id = %q, want shadow-compare-sample-primary", got)
	}
	if got := stringValue(nestedMap(report, "shadow")["task_id"], ""); got != "shadow-compare-sample-shadow" {
		t.Fatalf("shadow.task_id = %q, want shadow-compare-sample-shadow", got)
	}
	diff := nestedMap(report, "diff")
	if !boolValue(diff["state_equal"], false) || !boolValue(diff["event_types_equal"], false) {
		t.Fatalf("unexpected diff payload: %+v", diff)
	}
	if got := ExitCode(report); got != 0 {
		t.Fatalf("ExitCode = %d, want 0", got)
	}
}

type shadowTestServer struct {
	t      *testing.T
	mu     sync.Mutex
	events []map[string]any
	tasks  map[string]map[string]any
}

func newShadowTestServer(t *testing.T, suffix string, template []map[string]any) *httptest.Server {
	state := &shadowTestServer{t: t, events: template, tasks: map[string]map[string]any{}}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/healthz":
			writeJSON(t, w, map[string]any{"ok": true})
		case r.Method == http.MethodPost && r.URL.Path == "/tasks":
			var task map[string]any
			if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
				t.Fatalf("decode task: %v", err)
			}
			taskID := stringValue(task["id"], "")
			traceID := stringValue(task["trace_id"], "")
			state.mu.Lock()
			state.tasks[taskID] = map[string]any{
				"state":    "succeeded",
				"task_id":  taskID,
				"trace_id": traceID,
			}
			state.mu.Unlock()
			writeJSON(t, w, map[string]any{"ok": true})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/tasks/"):
			taskID := strings.TrimPrefix(r.URL.Path, "/tasks/")
			state.mu.Lock()
			task := state.tasks[taskID]
			state.mu.Unlock()
			writeJSON(t, w, task)
		case r.Method == http.MethodGet && r.URL.Path == "/events":
			taskID := r.URL.Query().Get("task_id")
			traceID := strings.TrimSuffix(strings.TrimSuffix(taskID, "-primary"), "-shadow")
			events := make([]map[string]any, 0, len(state.events))
			for _, event := range state.events {
				cloned := cloneMap(event)
				cloned["task_id"] = taskID
				cloned["trace_id"] = traceID
				cloned["id"] = traceID + "-" + suffix + "-" + stringValue(event["id"], "")
				events = append(events, cloned)
			}
			writeJSON(t, w, map[string]any{"events": events})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	return httptest.NewServer(handler)
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload map[string]any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}
