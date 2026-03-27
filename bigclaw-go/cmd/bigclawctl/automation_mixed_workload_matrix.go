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
	BaseURL        string
	GoRoot         string
	ReportPath     string
	TimeoutSeconds int
	Autostart      bool
	HTTPClient     *http.Client
	Now            func() time.Time
	Sleep          func(time.Duration)
	StartBigClawd  func(string, map[string]string) (*exec.Cmd, string, string, string, error)
}

type automationMixedWorkloadTaskReport struct {
	Name             string           `json:"name"`
	TaskID           string           `json:"task_id"`
	TraceID          string           `json:"trace_id"`
	ExpectedExecutor string           `json:"expected_executor"`
	RoutedExecutor   string           `json:"routed_executor,omitempty"`
	RoutedReason     string           `json:"routed_reason,omitempty"`
	FinalState       string           `json:"final_state"`
	LatestEventType  string           `json:"latest_event_type,omitempty"`
	Events           []map[string]any `json:"events"`
	OK               bool             `json:"ok"`
}

type automationMixedWorkloadMatrixReport struct {
	GeneratedAt string                              `json:"generated_at"`
	BaseURL     string                              `json:"base_url"`
	StateDir    string                              `json:"state_dir,omitempty"`
	ServiceLog  string                              `json:"service_log,omitempty"`
	AllOK       bool                                `json:"all_ok"`
	Tasks       []automationMixedWorkloadTaskReport `json:"tasks"`
}

func runAutomationMixedWorkloadMatrixCommand(args []string) error {
	flags := flag.NewFlagSet("automation e2e mixed-workload-matrix", flag.ContinueOnError)
	baseURL := flags.String("base-url", envOrDefault("BIGCLAW_ADDR", "http://127.0.0.1:8080"), "BigClaw API base URL")
	goRoot := flags.String("go-root", ".", "bigclaw-go repo root")
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
		BaseURL:        trim(*baseURL),
		GoRoot:         absPath(*goRoot),
		ReportPath:     trim(*reportPath),
		TimeoutSeconds: *timeoutSeconds,
		Autostart:      *autostart,
	})
	if report != nil {
		return emit(structToMap(report), *asJSON, exitCode)
	}
	return err
}

func automationMixedWorkloadMatrix(opts automationMixedWorkloadMatrixOptions) (*automationMixedWorkloadMatrixReport, int, error) {
	client := opts.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	sleep := opts.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	startBigClawd := opts.StartBigClawd
	if startBigClawd == nil {
		startBigClawd = startAutomationBigClawd
	}

	activeBaseURL := trim(opts.BaseURL)
	var cmd *exec.Cmd
	var stateDir string
	var serviceLog string
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
		var err error
		cmd, activeBaseURL, stateDir, serviceLog, err = startBigClawd(opts.GoRoot, map[string]string{
			"BIGCLAW_QUEUE_BACKEND": envOrDefault("BIGCLAW_QUEUE_BACKEND", "sqlite"),
			"BIGCLAW_SERVICE_NAME":  envOrDefault("BIGCLAW_SERVICE_NAME", "bigclawd-mixed"),
		})
		if err != nil {
			return nil, 0, err
		}
	}

	if err := automationWaitForHealth(client, activeBaseURL, 60, time.Second, sleep); err != nil {
		return nil, 0, err
	}

	timestamp := now().Unix()
	specs := automationDefaultMixedWorkloadTasks(timestamp)
	results := make([]automationMixedWorkloadTaskReport, 0, len(specs))
	allOK := true

	for _, spec := range specs {
		if err := automationRequestJSON(client, http.MethodPost, activeBaseURL, "/tasks", spec.Task, nil); err != nil {
			return nil, 0, err
		}
		status, err := automationWaitForTask(client, activeBaseURL, spec.Task.ID, time.Duration(opts.TimeoutSeconds)*time.Second, sleep)
		if err != nil {
			return nil, 0, err
		}
		events, err := automationFetchEvents(client, activeBaseURL, spec.Task.ID)
		if err != nil {
			return nil, 0, err
		}
		routedExecutor, routedReason := automationFindRoutedExecutor(events)
		latestEventType := ""
		if latestEvent, ok := status["latest_event"].(map[string]any); ok {
			latestEventType, _ = latestEvent["type"].(string)
		}
		item := automationMixedWorkloadTaskReport{
			Name:             spec.Name,
			TaskID:           spec.Task.ID,
			TraceID:          spec.Task.TraceID,
			ExpectedExecutor: spec.ExpectedExecutor,
			RoutedExecutor:   routedExecutor,
			RoutedReason:     routedReason,
			FinalState:       fmt.Sprint(status["state"]),
			LatestEventType:  latestEventType,
			Events:           events,
			OK:               fmt.Sprint(status["state"]) == "succeeded" && routedExecutor == spec.ExpectedExecutor,
		}
		if !item.OK {
			allOK = false
		}
		results = append(results, item)
	}

	report := &automationMixedWorkloadMatrixReport{
		GeneratedAt: now().UTC().Format(time.RFC3339),
		BaseURL:     activeBaseURL,
		StateDir:    stateDir,
		ServiceLog:  serviceLog,
		AllOK:       allOK,
		Tasks:       results,
	}
	if err := automationWriteReport(opts.GoRoot, opts.ReportPath, report); err != nil {
		return nil, 0, err
	}
	if report.AllOK {
		return report, 0, nil
	}
	return report, 1, nil
}

type automationMixedWorkloadTaskSpec struct {
	Name             string
	ExpectedExecutor string
	Task             automationTask
}

func automationDefaultMixedWorkloadTasks(timestamp int64) []automationMixedWorkloadTaskSpec {
	suffix := fmt.Sprintf("%d", timestamp)
	return []automationMixedWorkloadTaskSpec{
		{
			Name:             "local-default",
			ExpectedExecutor: "local",
			Task: automationTask{
				ID:                      "mixed-local-" + suffix,
				TraceID:                 "mixed-local-" + suffix,
				Title:                   "Mixed workload local default",
				Entrypoint:              "echo local default",
				ExecutionTimeoutSeconds: 240,
				Metadata:                map[string]string{"scenario": "mixed-workload", "profile": "local-default"},
			},
		},
		{
			Name:             "browser-auto",
			ExpectedExecutor: "kubernetes",
			Task: automationTask{
				ID:                      "mixed-browser-" + suffix,
				TraceID:                 "mixed-browser-" + suffix,
				Title:                   "Mixed workload browser auto-route",
				Entrypoint:              "echo browser via kubernetes",
				ContainerImage:          "alpine:3.20",
				ExecutionTimeoutSeconds: 240,
				Metadata:                map[string]string{"scenario": "mixed-workload", "profile": "browser-auto"},
				Extra:                   map[string]any{"required_tools": []any{"browser"}},
			},
		},
		{
			Name:             "gpu-auto",
			ExpectedExecutor: "ray",
			Task: automationTask{
				ID:                      "mixed-gpu-" + suffix,
				TraceID:                 "mixed-gpu-" + suffix,
				Title:                   "Mixed workload gpu auto-route",
				Entrypoint:              "python -c \"print('gpu via ray')\"",
				ExecutionTimeoutSeconds: 240,
				Metadata:                map[string]string{"scenario": "mixed-workload", "profile": "gpu-auto"},
				Extra:                   map[string]any{"required_tools": []any{"gpu"}},
			},
		},
		{
			Name:             "high-risk-auto",
			ExpectedExecutor: "kubernetes",
			Task: automationTask{
				ID:                      "mixed-risk-" + suffix,
				TraceID:                 "mixed-risk-" + suffix,
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
				ID:                      "mixed-required-ray-" + suffix,
				TraceID:                 "mixed-required-ray-" + suffix,
				Title:                   "Mixed workload explicit ray",
				RequiredExecutor:        "ray",
				Entrypoint:              "python -c \"print('required ray')\"",
				ExecutionTimeoutSeconds: 240,
				Metadata:                map[string]string{"scenario": "mixed-workload", "profile": "required-ray"},
			},
		},
	}
}

func automationFindRoutedExecutor(events []map[string]any) (string, string) {
	for _, event := range events {
		if eventType, _ := event["type"].(string); eventType != "scheduler.routed" {
			continue
		}
		payload, _ := event["payload"].(map[string]any)
		if payload == nil {
			return "", ""
		}
		executor, _ := payload["executor"].(string)
		reason, _ := payload["reason"].(string)
		return executor, reason
	}
	return "", ""
}
