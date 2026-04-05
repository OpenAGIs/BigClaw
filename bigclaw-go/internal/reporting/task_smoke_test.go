package reporting

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunTaskSmoke(t *testing.T) {
	tasks := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/healthz":
			_, _ = w.Write([]byte(`{"ok":true}`))
		case r.Method == http.MethodPost && r.URL.Path == "/tasks":
			var task map[string]any
			if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
				t.Fatalf("decode task: %v", err)
			}
			if task["required_executor"] != "local" {
				t.Fatalf("unexpected executor payload: %#v", task)
			}
			if task["entrypoint"] != "echo hello" {
				t.Fatalf("unexpected entrypoint payload: %#v", task)
			}
			if task["container_image"] != "alpine:3.20" {
				t.Fatalf("unexpected image payload: %#v", task)
			}
			runtimeEnv := asMap(task["runtime_env"])
			if asString(runtimeEnv["BIGCLAW_SMOKE"]) != "1" {
				t.Fatalf("unexpected runtime env payload: %#v", task)
			}
			metadata := asMap(task["metadata"])
			if asString(metadata["ticket"]) != "BIG-GO-1475" {
				t.Fatalf("unexpected metadata payload: %#v", task)
			}
			task["state"] = "queued"
			_, _ = w.Write(mustJSON(map[string]any{"task": task}))
		case r.Method == http.MethodGet && len(r.URL.Path) > len("/tasks/") && r.URL.Path[:7] == "/tasks/":
			taskID := filepath.Base(r.URL.Path)
			tasks[taskID]++
			state := "running"
			if tasks[taskID] > 1 {
				state = "succeeded"
			}
			_, _ = w.Write(mustJSON(map[string]any{
				"id":    taskID,
				"state": state,
			}))
		case r.Method == http.MethodGet && r.URL.Path == "/events":
			_, _ = w.Write(mustJSON(map[string]any{
				"events": []map[string]any{
					{"id": "evt-1", "task_id": r.URL.Query().Get("task_id"), "type": "task.completed"},
				},
			}))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	goRoot := t.TempDir()
	reportPath := "docs/reports/local-smoke-report.json"
	result, err := RunTaskSmoke(TaskSmokeOptions{
		Executor:       "local",
		Title:          "SQLite smoke",
		Entrypoint:     "echo hello",
		Image:          "alpine:3.20",
		BaseURL:        server.URL,
		GoRoot:         goRoot,
		TimeoutSeconds: 5,
		PollInterval:   10 * time.Millisecond,
		RuntimeEnvJSON: `{"BIGCLAW_SMOKE":"1"}`,
		MetadataJSON:   `{"ticket":"BIG-GO-1475"}`,
		ReportPath:     reportPath,
		TimeNow: func() time.Time {
			return time.Unix(1712360000, 0).UTC()
		},
	})
	if err != nil {
		t.Fatalf("run task smoke: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("unexpected exit code: %d", result.ExitCode)
	}
	if result.Report.BaseURL != server.URL {
		t.Fatalf("unexpected base url: %s", result.Report.BaseURL)
	}
	if result.Report.Autostarted {
		t.Fatalf("expected non-autostarted report")
	}
	if asString(result.Report.Status["state"]) != "succeeded" {
		t.Fatalf("unexpected status: %#v", result.Report.Status)
	}
	if len(result.Report.Events) != 1 || asString(result.Report.Events[0]["type"]) != "task.completed" {
		t.Fatalf("unexpected events: %#v", result.Report.Events)
	}

	reportBytes, err := os.ReadFile(filepath.Join(goRoot, reportPath))
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var written TaskSmokeReport
	if err := json.Unmarshal(reportBytes, &written); err != nil {
		t.Fatalf("unmarshal report: %v", err)
	}
	if asString(written.Task["id"]) == "" {
		t.Fatalf("expected task id in written report: %#v", written)
	}
}

func TestParseTaskSmokeCLIFlags(t *testing.T) {
	options, pretty, err := ParseTaskSmokeCLIFlags([]string{
		"--executor", "ray",
		"--title", "Ray smoke",
		"--entrypoint", "python -c \"print(1)\"",
		"--poll-interval", "0.5",
		"--timeout-seconds", "90",
		"--runtime-env-json", `{"pip":["x"]}`,
		"--metadata-json", `{"ticket":"BIG-GO-1475"}`,
		"--report-path", "docs/reports/ray-live-smoke-report.json",
		"--autostart",
		"--pretty",
	})
	if err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	if !pretty || !options.Autostart {
		t.Fatalf("expected pretty/autostart flags: %+v pretty=%t", options, pretty)
	}
	if options.Executor != "ray" || options.TimeoutSeconds != 90 {
		t.Fatalf("unexpected parsed options: %+v", options)
	}
	if options.PollInterval != 500*time.Millisecond {
		t.Fatalf("unexpected poll interval: %s", options.PollInterval)
	}
}

func mustJSON(payload any) []byte {
	contents, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return contents
}
