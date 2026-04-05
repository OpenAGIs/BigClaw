package reporting

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestParseBenchmarkStdout(t *testing.T) {
	parsed := parseBenchmarkStdout("BenchmarkMemoryQueueEnqueueLease-8    \t20000\t66075 ns/op\nBenchmarkSchedulerDecide-8 100 73.98 ns/op\n")
	if asFloat(asMap(parsed["BenchmarkMemoryQueueEnqueueLease-8"])["ns_per_op"]) != 66075 {
		t.Fatalf("unexpected parsed bench result: %+v", parsed)
	}
	if asFloat(asMap(parsed["BenchmarkSchedulerDecide-8"])["ns_per_op"]) != 73.98 {
		t.Fatalf("unexpected scheduler bench result: %+v", parsed)
	}
}

func TestRunLocalSoak(t *testing.T) {
	var states sync.Map
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
			states.Store(asString(task["id"]), 0)
			_ = json.NewEncoder(w).Encode(map[string]any{"accepted": true})
		case r.Method == http.MethodGet && len(r.URL.Path) > len("/tasks/") && r.URL.Path[:7] == "/tasks/":
			taskID := filepath.Base(r.URL.Path)
			currentAny, _ := states.Load(taskID)
			current, _ := currentAny.(int)
			current++
			states.Store(taskID, current)
			state := "running"
			if current > 1 {
				state = "succeeded"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": taskID, "state": state})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	root := t.TempDir()
	now := time.Unix(1712360000, 0).UTC()
	report, exitCode, err := RunLocalSoak(LocalSoakOptions{
		Count:          5,
		Workers:        2,
		BaseURL:        server.URL,
		GoRoot:         root,
		TimeoutSeconds: 5,
		ReportPath:     "docs/reports/soak-local-report.json",
		TimeNow: func() time.Time {
			return now
		},
	})
	if err != nil {
		t.Fatalf("run local soak: %v", err)
	}
	if exitCode != 0 || asInt(report["succeeded"]) != 5 || asInt(report["failed"]) != 0 {
		t.Fatalf("unexpected soak result: exit=%d report=%+v", exitCode, report)
	}
	if len(anyToMapSlice(report["sample_status"])) != 3 {
		t.Fatalf("unexpected sample statuses: %+v", report["sample_status"])
	}
	if _, err := os.Stat(filepath.Join(root, "docs/reports/soak-local-report.json")); err != nil {
		t.Fatalf("expected written soak report: %v", err)
	}
}

func TestRunBenchmarkMatrix(t *testing.T) {
	root := t.TempDir()
	report, err := RunBenchmarkMatrix(BenchmarkMatrixOptions{
		GoRoot:         root,
		ReportPath:     "docs/reports/benchmark-matrix-report.json",
		TimeoutSeconds: 180,
		Scenarios:      []string{"50:8", "100:12"},
		BenchmarkRunner: func(string) (map[string]any, error) {
			return map[string]any{
				"stdout": "BenchmarkMemoryQueueEnqueueLease-8 1 66075 ns/op\n",
				"parsed": map[string]any{"BenchmarkMemoryQueueEnqueueLease-8": map[string]any{"ns_per_op": 66075.0}},
			}, nil
		},
		SoakRunner: func(_ string, count int, workers int, _ int, reportPath string) (map[string]any, error) {
			payload := map[string]any{"count": count, "workers": workers, "elapsed_seconds": 1.5, "throughput_tasks_per_sec": 10.0, "succeeded": count, "failed": 0}
			if err := WriteJSON(filepath.Join(root, reportPath), payload); err != nil {
				return nil, err
			}
			return payload, nil
		},
	})
	if err != nil {
		t.Fatalf("run benchmark matrix: %v", err)
	}
	if len(anyToMapSlice(report["soak_matrix"])) != 2 {
		t.Fatalf("unexpected soak matrix: %+v", report)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/reports/benchmark-matrix-report.json")); err != nil {
		t.Fatalf("expected matrix report: %v", err)
	}
}
