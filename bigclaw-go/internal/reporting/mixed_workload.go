package reporting

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"
)

const MixedWorkloadMatrixGenerator = "bigclaw-go/scripts/e2e/mixed_workload_matrix/main.go"

type MixedWorkloadMatrixOptions struct {
	GoRoot         string
	ReportPath     string
	TimeoutSeconds int
	Autostart      bool
	BaseURL        string
	TimeNow        func() time.Time
	HTTPClient     *http.Client
	StartBigclawd  func(string, map[string]string) (*exec.Cmd, *os.File, string, error)
}

func RunMixedWorkloadMatrix(options MixedWorkloadMatrixOptions) (map[string]any, int, error) {
	if options.GoRoot == "" {
		root, err := FindRepoRoot(".")
		if err != nil {
			return nil, 1, err
		}
		options.GoRoot = root
	}
	if options.ReportPath == "" {
		options.ReportPath = "docs/reports/mixed-workload-matrix-report.json"
	}
	if options.TimeoutSeconds <= 0 {
		options.TimeoutSeconds = 240
	}
	if options.BaseURL == "" {
		options.BaseURL = "http://127.0.0.1:8080"
	}
	if options.TimeNow == nil {
		options.TimeNow = time.Now
	}
	if options.StartBigclawd == nil {
		options.StartBigclawd = startTaskSmokeBigclawd
	}
	client := options.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	process, logFile, logPath, stateDir, activeBaseURL, err := prepareTaskSmokeService(TaskSmokeOptions{
		BaseURL:        options.BaseURL,
		GoRoot:         options.GoRoot,
		Autostart:      options.Autostart,
		HealthAttempts: 60,
		HealthInterval: time.Second,
		StartBigclawd:  options.StartBigclawd,
	})
	if err != nil {
		return nil, 1, err
	}
	defer stopTaskSmokeProcess(process, logFile)

	timestamp := options.TimeNow().Unix()
	taskSpecs := defaultMixedWorkloadTasks(timestamp)
	results := make([]map[string]any, 0, len(taskSpecs))
	allOK := true
	for _, spec := range taskSpecs {
		task := asMap(spec["task"])
		if _, err := requestTaskSmokeJSON(client, activeBaseURL, http.MethodPost, "/tasks", task); err != nil {
			return nil, 1, err
		}
		status, err := waitLocalSoakTask(client, activeBaseURL, asString(task["id"]), time.Duration(options.TimeoutSeconds)*time.Second)
		if err != nil {
			return nil, 1, err
		}
		eventsPayload, err := requestTaskSmokeJSON(client, activeBaseURL, http.MethodGet, fmt.Sprintf("/events?task_id=%s&limit=100", asString(task["id"])), nil)
		if err != nil {
			return nil, 1, err
		}
		events := anyToMapSlice(eventsPayload["events"])
		routed := mixedWorkloadRoutedEvent(events)
		routedExecutor := ""
		routedReason := ""
		if len(routed) > 0 {
			payload := asMap(routed["payload"])
			routedExecutor = asString(payload["executor"])
			routedReason = asString(payload["reason"])
		}
		ok := asString(status["state"]) == "succeeded" && routedExecutor == asString(spec["expected_executor"])
		if !ok {
			allOK = false
		}
		result := map[string]any{
			"name":              asString(spec["name"]),
			"task_id":           asString(task["id"]),
			"trace_id":          asString(task["trace_id"]),
			"expected_executor": asString(spec["expected_executor"]),
			"routed_executor":   routedExecutor,
			"routed_reason":     routedReason,
			"final_state":       asString(status["state"]),
			"latest_event_type": asString(asMap(status["latest_event"])["type"]),
			"events":            events,
			"ok":                ok,
		}
		results = append(results, result)
	}
	report := map[string]any{
		"generated_at": options.TimeNow().UTC().Format(time.RFC3339),
		"base_url":     activeBaseURL,
		"state_dir":    stateDir,
		"service_log":  logPath,
		"all_ok":       allOK,
		"tasks":        results,
	}
	if err := WriteJSON(resolveReportPath(options.GoRoot, options.ReportPath), report); err != nil {
		return nil, 1, err
	}
	exitCode := 1
	if allOK {
		exitCode = 0
	}
	return report, exitCode, nil
}

func defaultMixedWorkloadTasks(timestamp int64) []map[string]any {
	suffix := fmt.Sprintf("%d", timestamp)
	return []map[string]any{
		{
			"name":              "local-default",
			"expected_executor": "local",
			"task": map[string]any{
				"id":         "mixed-local-" + suffix,
				"trace_id":   "mixed-local-" + suffix,
				"title":      "Mixed workload local default",
				"entrypoint": "echo local default",
				"metadata":   map[string]any{"scenario": "mixed-workload", "profile": "local-default"},
			},
		},
		{
			"name":              "browser-auto",
			"expected_executor": "kubernetes",
			"task": map[string]any{
				"id":              "mixed-browser-" + suffix,
				"trace_id":        "mixed-browser-" + suffix,
				"title":           "Mixed workload browser auto-route",
				"required_tools":  []any{"browser"},
				"container_image": "alpine:3.20",
				"entrypoint":      "echo browser via kubernetes",
				"metadata":        map[string]any{"scenario": "mixed-workload", "profile": "browser-auto"},
			},
		},
		{
			"name":              "gpu-auto",
			"expected_executor": "ray",
			"task": map[string]any{
				"id":             "mixed-gpu-" + suffix,
				"trace_id":       "mixed-gpu-" + suffix,
				"title":          "Mixed workload gpu auto-route",
				"required_tools": []any{"gpu"},
				"entrypoint":     "python -c \"print('gpu via ray')\"",
				"metadata":       map[string]any{"scenario": "mixed-workload", "profile": "gpu-auto"},
			},
		},
		{
			"name":              "high-risk-auto",
			"expected_executor": "kubernetes",
			"task": map[string]any{
				"id":              "mixed-risk-" + suffix,
				"trace_id":        "mixed-risk-" + suffix,
				"title":           "Mixed workload high-risk auto-route",
				"risk_level":      "high",
				"container_image": "alpine:3.20",
				"entrypoint":      "echo high risk via kubernetes",
				"metadata":        map[string]any{"scenario": "mixed-workload", "profile": "high-risk-auto"},
			},
		},
		{
			"name":              "required-ray",
			"expected_executor": "ray",
			"task": map[string]any{
				"id":                "mixed-required-ray-" + suffix,
				"trace_id":          "mixed-required-ray-" + suffix,
				"title":             "Mixed workload explicit ray",
				"required_executor": "ray",
				"entrypoint":        "python -c \"print('required ray')\"",
				"metadata":          map[string]any{"scenario": "mixed-workload", "profile": "required-ray"},
			},
		},
	}
}

func mixedWorkloadRoutedEvent(events []map[string]any) map[string]any {
	for _, event := range events {
		if asString(event["type"]) == "scheduler.routed" {
			return event
		}
	}
	return nil
}
