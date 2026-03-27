package soaklocal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestWaitHealth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer server.Close()
	if err := WaitHealth(server.URL, 1); err != nil {
		t.Fatalf("wait health: %v", err)
	}
}

func TestRunWritesReport(t *testing.T) {
	var mu sync.Mutex
	statuses := map[string]string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/healthz":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		case r.Method == http.MethodPost && r.URL.Path == "/tasks":
			var task map[string]any
			_ = json.NewDecoder(r.Body).Decode(&task)
			id, _ := task["id"].(string)
			mu.Lock()
			statuses[id] = "succeeded"
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "accepted"})
		case r.Method == http.MethodGet && len(r.URL.Path) > len("/tasks/"):
			id := filepath.Base(r.URL.Path)
			mu.Lock()
			state := statuses[id]
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(TaskStatus{ID: id, State: state})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	goRoot := t.TempDir()
	report, logPath, err := Run(Options{
		Count:          3,
		Workers:        2,
		BaseURL:        server.URL,
		GoRoot:         goRoot,
		TimeoutSeconds: 1,
		ReportPath:     "docs/reports/soak-local-report.json",
	})
	if err != nil {
		t.Fatalf("run soak local: %v", err)
	}
	if logPath != "" {
		t.Fatalf("expected empty log path without autostart, got %s", logPath)
	}
	if report.Succeeded != 3 || report.Failed != 0 {
		t.Fatalf("unexpected report: %+v", report)
	}
	body, err := os.ReadFile(filepath.Join(goRoot, "docs/reports/soak-local-report.json"))
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var decoded Report
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if decoded.Count != 3 {
		t.Fatalf("unexpected decoded report: %+v", decoded)
	}
}
