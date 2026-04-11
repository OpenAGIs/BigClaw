package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"
)

type automationMixedWorkloadMatrixOptions struct {
	GoRoot         string
	BaseURL        string
	ReportPath     string
	TimeoutSeconds int
	Autostart      bool
	HTTPClient     *http.Client
	Now            func() time.Time
	Sleep          func(time.Duration)
	StartBigClawd  func(string, map[string]string) (*exec.Cmd, string, string, string, error)
}

func runAutomationMixedWorkloadMatrixCommand(args []string) error {
	flags := flag.NewFlagSet("automation e2e mixed-workload-matrix", flag.ContinueOnError)
	goRoot := flags.String("go-root", ".", "bigclaw-go repo root")
	baseURL := flags.String("base-url", envOrDefault("BIGCLAW_ADDR", "http://127.0.0.1:8080"), "BigClaw API base URL")
	reportPath := flags.String("report-path", "docs/reports/mixed-workload-matrix-report.json", "report path")
	timeoutSeconds := flags.Int("timeout-seconds", 240, "task timeout seconds")
	autostart := flags.Bool("autostart", true, "autostart bigclawd with temporary state")
	asJSON := flags.Bool("json", true, "json")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation e2e mixed-workload-matrix [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	report, exitCode, err := automationMixedWorkloadMatrix(automationMixedWorkloadMatrixOptions{
		GoRoot:         absPath(*goRoot),
		BaseURL:        *baseURL,
		ReportPath:     *reportPath,
		TimeoutSeconds: *timeoutSeconds,
		Autostart:      *autostart,
		HTTPClient:     http.DefaultClient,
	})
	if err != nil {
		return err
	}
	return emit(report, *asJSON, exitCode)
}

func automationMixedWorkloadMatrix(opts automationMixedWorkloadMatrixOptions) (map[string]any, int, error) {
	client := opts.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	sleep := opts.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	startBigClawd := opts.StartBigClawd
	if startBigClawd == nil {
		startBigClawd = startAutomationBigClawd
	}

	activeBaseURL := trim(opts.BaseURL)
	var cmd *exec.Cmd
	var serviceLog string
	var stateDir string
	defer func() {
		if cmd != nil && cmd.Process != nil {
			_ = cmd.Process.Signal(os.Interrupt)
			done := make(chan struct{})
			go func() {
				_, _ = cmd.Process.Wait()
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				_ = cmd.Process.Kill()
			}
		}
	}()

	if opts.Autostart {
		if err := automationWaitForHealth(client, activeBaseURL, 2, 200*time.Millisecond, sleep); err != nil {
			var startErr error
			cmd, activeBaseURL, stateDir, serviceLog, startErr = startBigClawd(opts.GoRoot, map[string]string{})
			if startErr != nil {
				return nil, 0, startErr
			}
		}
	}
	if err := automationWaitForHealth(client, activeBaseURL, 60, time.Second, sleep); err != nil {
		return nil, 0, err
	}

	timestamp := now().Unix()
	taskSpecs := defaultMixedWorkloadTasks(timestamp)
	results := make([]any, 0, len(taskSpecs))
	allOK := true
	for _, spec := range taskSpecs {
		task := spec.Task
		if err := automationRequestJSON(client, http.MethodPost, activeBaseURL, "/tasks", task, nil); err != nil {
			return nil, 0, err
		}
		status, err := automationWaitForTask(client, activeBaseURL, task.ID, time.Duration(opts.TimeoutSeconds)*time.Second, sleep)
		if err != nil {
			return nil, 0, err
		}
		events, err := automationFetchEvents(client, activeBaseURL, task.ID)
		if err != nil {
			return nil, 0, err
		}
		routedExecutor := ""
		routedReason := ""
		for _, event := range events {
			if eventType, _ := event["type"].(string); eventType == "scheduler.routed" {
				payload, _ := event["payload"].(map[string]any)
				routedExecutor = firstNonEmpty(payload["executor"])
				routedReason = firstNonEmpty(payload["reason"])
				break
			}
		}
		latestEventType := ""
		if latestEvent, ok := status["latest_event"].(map[string]any); ok {
			latestEventType = firstNonEmpty(latestEvent["type"])
		}
		ok := firstNonEmpty(status["state"]) == "succeeded" && routedExecutor == spec.ExpectedExecutor
		if !ok {
			allOK = false
		}
		results = append(results, map[string]any{
			"name":              spec.Name,
			"task_id":           task.ID,
			"trace_id":          task.TraceID,
			"expected_executor": spec.ExpectedExecutor,
			"routed_executor":   routedExecutor,
			"routed_reason":     routedReason,
			"final_state":       status["state"],
			"latest_event_type": latestEventType,
			"events":            events,
			"ok":                ok,
		})
	}

	report := map[string]any{
		"generated_at": now().UTC().Format(time.RFC3339),
		"base_url":     activeBaseURL,
		"state_dir":    stateDir,
		"service_log":  serviceLog,
		"all_ok":       allOK,
		"tasks":        results,
	}
	if err := automationWriteReport(opts.GoRoot, opts.ReportPath, report); err != nil {
		return nil, 0, err
	}
	if allOK {
		return report, 0, nil
	}
	return report, 1, nil
}

type mixedWorkloadTaskSpec struct {
	Name             string
	ExpectedExecutor string
	Task             automationTask
}

func defaultMixedWorkloadTasks(timestamp int64) []mixedWorkloadTaskSpec {
	return []mixedWorkloadTaskSpec{
		{
			Name:             "local-default",
			ExpectedExecutor: "local",
			Task: automationTask{
				ID:                      fmt.Sprintf("mixed-local-%d", timestamp),
				TraceID:                 fmt.Sprintf("mixed-local-%d", timestamp),
				Title:                   "Mixed workload local default",
				Entrypoint:              "echo local default",
				Metadata:                map[string]string{"scenario": "mixed-workload", "profile": "local-default"},
				ExecutionTimeoutSeconds: 240,
			},
		},
		{
			Name:             "browser-auto",
			ExpectedExecutor: "kubernetes",
			Task: automationTask{
				ID:                      fmt.Sprintf("mixed-browser-%d", timestamp),
				TraceID:                 fmt.Sprintf("mixed-browser-%d", timestamp),
				Title:                   "Mixed workload browser auto-route",
				Entrypoint:              "echo browser via kubernetes",
				ContainerImage:          "alpine:3.20",
				ExecutionTimeoutSeconds: 240,
				Metadata:                map[string]string{"scenario": "mixed-workload", "profile": "browser-auto"},
				Extra:                   map[string]any{"required_tools": []string{"browser"}},
			},
		},
		{
			Name:             "gpu-auto",
			ExpectedExecutor: "ray",
			Task: automationTask{
				ID:                      fmt.Sprintf("mixed-gpu-%d", timestamp),
				TraceID:                 fmt.Sprintf("mixed-gpu-%d", timestamp),
				Title:                   "Mixed workload gpu auto-route",
				Entrypoint:              "echo gpu via ray",
				ExecutionTimeoutSeconds: 240,
				Metadata:                map[string]string{"scenario": "mixed-workload", "profile": "gpu-auto"},
				Extra:                   map[string]any{"required_tools": []string{"gpu"}},
			},
		},
		{
			Name:             "high-risk-auto",
			ExpectedExecutor: "kubernetes",
			Task: automationTask{
				ID:                      fmt.Sprintf("mixed-risk-%d", timestamp),
				TraceID:                 fmt.Sprintf("mixed-risk-%d", timestamp),
				Title:                   "Mixed workload high-risk auto-route",
				Entrypoint:              "echo high risk via kubernetes",
				ContainerImage:          "alpine:3.20",
				ExecutionTimeoutSeconds: 240,
				Metadata:                map[string]string{"scenario": "mixed-workload", "profile": "high-risk-auto"},
				Extra:                   map[string]any{"risk_level": "high"},
			},
		},
		{
			Name:             "required-ray",
			ExpectedExecutor: "ray",
			Task: automationTask{
				ID:                      fmt.Sprintf("mixed-required-ray-%d", timestamp),
				TraceID:                 fmt.Sprintf("mixed-required-ray-%d", timestamp),
				Title:                   "Mixed workload explicit ray",
				RequiredExecutor:        "ray",
				Entrypoint:              "echo required ray",
				ExecutionTimeoutSeconds: 240,
				Metadata:                map[string]string{"scenario": "mixed-workload", "profile": "required-ray"},
			},
		},
	}
}
