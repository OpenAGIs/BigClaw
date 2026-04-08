package executor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
)

func TestRayRunnerExecuteUsesJobsAPI(t *testing.T) {
	statuses := []string{"PENDING", "RUNNING", "SUCCEEDED"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/jobs/":
			_ = json.NewEncoder(w).Encode(map[string]any{"job_id": "raysubmit_test_123"})
		case r.Method == http.MethodGet && r.URL.Path == "/api/jobs/raysubmit_test_123":
			status := statuses[0]
			statuses = statuses[1:]
			_ = json.NewEncoder(w).Encode(map[string]any{"job_id": "raysubmit_test_123", "status": status, "message": "ok"})
		case r.Method == http.MethodGet && r.URL.Path == "/api/jobs/raysubmit_test_123/logs":
			_ = json.NewEncoder(w).Encode(map[string]any{"logs": "ray job output"})
		default:
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	runner := NewRayRunnerWithClient(RayConfig{Address: server.URL, PollInterval: time.Millisecond, HTTPTimeout: time.Second}, server.Client(), parsed)
	result := runner.Execute(context.Background(), domain.Task{ID: "OPE-182", Title: "run on ray", Entrypoint: "sh -c 'echo hello from ray'"})
	if !result.Success {
		t.Fatalf("expected success, got %+v", result)
	}
	if !strings.Contains(result.Message, "ray job") {
		t.Fatalf("expected ray job message, got %s", result.Message)
	}
}

func TestRayRunnerStopsJobOnCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/jobs/":
			_ = json.NewEncoder(w).Encode(map[string]any{"job_id": "raysubmit_stop_123"})
		case r.Method == http.MethodGet && r.URL.Path == "/api/jobs/raysubmit_stop_123":
			_ = json.NewEncoder(w).Encode(map[string]any{"job_id": "raysubmit_stop_123", "status": "RUNNING", "message": "running"})
		case r.Method == http.MethodPost && r.URL.Path == "/api/jobs/raysubmit_stop_123/stop":
			_ = json.NewEncoder(w).Encode(map[string]any{"stopped": true})
		default:
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	runner := NewRayRunnerWithClient(RayConfig{Address: server.URL, PollInterval: time.Millisecond, HTTPTimeout: time.Second}, server.Client(), parsed)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()
	result := runner.Execute(ctx, domain.Task{ID: "OPE-182", Title: "cancel ray"})
	if !result.ShouldRetry {
		t.Fatalf("expected retryable cancellation, got %+v", result)
	}
}
