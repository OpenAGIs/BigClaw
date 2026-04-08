package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type multiStringFlag []string

func (m *multiStringFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiStringFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

type automationRunTaskSmokeOptions struct {
	Executor       string
	Title          string
	Entrypoint     string
	Image          string
	BaseURL        string
	GoRoot         string
	TimeoutSeconds int
	PollInterval   time.Duration
	RuntimeEnvJSON string
	MetadataJSON   string
	ReportPath     string
	Autostart      bool
	HTTPClient     *http.Client
	Now            func() time.Time
	Sleep          func(time.Duration)
	StartBigClawd  func(string, map[string]string) (*exec.Cmd, string, string, string, error)
	Stdout         io.Writer
	Stderr         io.Writer
}

type automationShadowCompareOptions struct {
	PrimaryBaseURL       string
	ShadowBaseURL        string
	TaskFile             string
	TimeoutSeconds       int
	HealthTimeoutSeconds int
	ReportPath           string
	HTTPClient           *http.Client
	Now                  func() time.Time
	Sleep                func(time.Duration)
}

type automationShadowMatrixOptions struct {
	PrimaryBaseURL       string
	ShadowBaseURL        string
	TaskFiles            []string
	CorpusManifest       string
	ReplayCorpusSlices   bool
	TimeoutSeconds       int
	HealthTimeoutSeconds int
	ReportPath           string
	HTTPClient           *http.Client
	Now                  func() time.Time
	Sleep                func(time.Duration)
}

type automationLiveShadowScorecardOptions struct {
	ShadowCompareReportPath string
	ShadowMatrixReportPath  string
	OutputPath              string
	Now                     func() time.Time
}

type automationExportLiveShadowBundleOptions struct {
	GoRoot            string
	ShadowComparePath string
	ShadowMatrixPath  string
	ScorecardPath     string
	BundleRoot        string
	SummaryPath       string
	IndexPath         string
	ManifestPath      string
	RollupPath        string
	RunID             string
	Now               func() time.Time
}

type automationSoakLocalOptions struct {
	Count          int
	Workers        int
	BaseURL        string
	GoRoot         string
	TimeoutSeconds int
	Autostart      bool
	ReportPath     string
	HTTPClient     *http.Client
	Now            func() time.Time
	Sleep          func(time.Duration)
	StartBigClawd  func(string, map[string]string) (*exec.Cmd, string, string, string, error)
}

type automationTask struct {
	ID                      string                 `json:"id"`
	Title                   string                 `json:"title"`
	RequiredExecutor        string                 `json:"required_executor,omitempty"`
	Entrypoint              string                 `json:"entrypoint"`
	ContainerImage          string                 `json:"container_image,omitempty"`
	ExecutionTimeoutSeconds int                    `json:"execution_timeout_seconds"`
	RuntimeEnv              map[string]any         `json:"runtime_env,omitempty"`
	Metadata                map[string]string      `json:"metadata,omitempty"`
	TraceID                 string                 `json:"trace_id,omitempty"`
	Extra                   map[string]interface{} `json:"-"`
}

func (t automationTask) MarshalJSON() ([]byte, error) {
	payload := map[string]any{
		"id":                        t.ID,
		"title":                     t.Title,
		"entrypoint":                t.Entrypoint,
		"execution_timeout_seconds": t.ExecutionTimeoutSeconds,
	}
	if t.RequiredExecutor != "" {
		payload["required_executor"] = t.RequiredExecutor
	}
	if t.ContainerImage != "" {
		payload["container_image"] = t.ContainerImage
	}
	if len(t.RuntimeEnv) > 0 {
		payload["runtime_env"] = t.RuntimeEnv
	}
	if len(t.Metadata) > 0 {
		payload["metadata"] = t.Metadata
	}
	if t.TraceID != "" {
		payload["trace_id"] = t.TraceID
	}
	for key, value := range t.Extra {
		payload[key] = value
	}
	return json.Marshal(payload)
}

type automationRunTaskSmokeReport struct {
	Autostarted bool             `json:"autostarted"`
	BaseURL     string           `json:"base_url"`
	Task        map[string]any   `json:"task"`
	Status      map[string]any   `json:"status"`
	Events      []map[string]any `json:"events"`
	StateDir    string           `json:"state_dir,omitempty"`
	ServiceLog  string           `json:"service_log,omitempty"`
	Error       string           `json:"error,omitempty"`
}

type automationShadowCompareReport struct {
	TraceID string `json:"trace_id"`
	Primary struct {
		TaskID string           `json:"task_id"`
		Status map[string]any   `json:"status"`
		Events []map[string]any `json:"events"`
	} `json:"primary"`
	Shadow struct {
		TaskID string           `json:"task_id"`
		Status map[string]any   `json:"status"`
		Events []map[string]any `json:"events"`
	} `json:"shadow"`
	Diff struct {
		StateEqual             bool     `json:"state_equal"`
		EventCountDelta        int      `json:"event_count_delta"`
		EventTypesEqual        bool     `json:"event_types_equal"`
		PrimaryEventTypes      []string `json:"primary_event_types"`
		ShadowEventTypes       []string `json:"shadow_event_types"`
		PrimaryTimelineSeconds float64  `json:"primary_timeline_seconds"`
		ShadowTimelineSeconds  float64  `json:"shadow_timeline_seconds"`
	} `json:"diff"`
}

type automationSoakLocalReport struct {
	BaseURL               string           `json:"base_url"`
	Count                 int              `json:"count"`
	Workers               int              `json:"workers"`
	ElapsedSeconds        float64          `json:"elapsed_seconds"`
	ThroughputTasksPerSec float64          `json:"throughput_tasks_per_sec"`
	Succeeded             int              `json:"succeeded"`
	Failed                int              `json:"failed"`
	SampleStatus          []map[string]any `json:"sample_status"`
	Autostarted           bool             `json:"autostarted"`
	StateDir              string           `json:"state_dir,omitempty"`
	ServiceLog            string           `json:"service_log,omitempty"`
}

func runAutomation(args []string) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		printAutomationUsage(os.Stdout)
		return nil
	}
	switch args[0] {
	case "e2e":
		return runAutomationE2E(args[1:])
	case "benchmark":
		return runAutomationBenchmark(args[1:])
	case "migration":
		return runAutomationMigration(args[1:])
	default:
		return fmt.Errorf("unknown automation category: %s", args[0])
	}
}

func runAutomationE2E(args []string) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		_, _ = os.Stdout.WriteString("usage: bigclawctl automation e2e <run-task-smoke|export-validation-bundle|continuation-scorecard|continuation-policy-gate|broker-failover-stub-matrix|mixed-workload-matrix|cross-process-coordination-surface|subscriber-takeover-fault-matrix|external-store-validation|multi-node-shared-queue> [flags]\n")
		return nil
	}
	switch args[0] {
	case "run-task-smoke":
		return runAutomationRunTaskSmokeCommand(args[1:])
	case "export-validation-bundle":
		return runAutomationExportValidationBundleCommand(args[1:])
	case "continuation-scorecard":
		return runAutomationContinuationScorecardCommand(args[1:])
	case "continuation-policy-gate":
		return runAutomationContinuationPolicyGateCommand(args[1:])
	case "broker-failover-stub-matrix":
		return runAutomationBrokerFailoverStubMatrixCommand(args[1:])
	case "mixed-workload-matrix":
		return runAutomationMixedWorkloadMatrixCommand(args[1:])
	case "cross-process-coordination-surface":
		return runAutomationCrossProcessCoordinationSurfaceCommand(args[1:])
	case "subscriber-takeover-fault-matrix":
		return runAutomationSubscriberTakeoverFaultMatrixCommand(args[1:])
	case "external-store-validation":
		return runAutomationExternalStoreValidationCommand(args[1:])
	case "multi-node-shared-queue":
		return runAutomationMultiNodeSharedQueueCommand(args[1:])
	default:
		return fmt.Errorf("unknown automation e2e subcommand: %s", args[0])
	}
}

func runAutomationBenchmark(args []string) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		_, _ = os.Stdout.WriteString("usage: bigclawctl automation benchmark <soak-local|run-matrix|capacity-certification> [flags]\n")
		return nil
	}
	switch args[0] {
	case "soak-local":
		return runAutomationSoakLocalCommand(args[1:])
	case "run-matrix":
		return runAutomationBenchmarkRunMatrixCommand(args[1:])
	case "capacity-certification":
		return runAutomationBenchmarkCapacityCertificationCommand(args[1:])
	default:
		return fmt.Errorf("unknown automation benchmark subcommand: %s", args[0])
	}
}

func runAutomationMigration(args []string) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		_, _ = os.Stdout.WriteString("usage: bigclawctl automation migration <shadow-compare|shadow-matrix|live-shadow-scorecard|export-live-shadow-bundle> [flags]\n")
		return nil
	}
	switch args[0] {
	case "shadow-compare":
		return runAutomationShadowCompareCommand(args[1:])
	case "shadow-matrix":
		return runAutomationShadowMatrixCommand(args[1:])
	case "live-shadow-scorecard":
		return runAutomationLiveShadowScorecardCommand(args[1:])
	case "export-live-shadow-bundle":
		return runAutomationExportLiveShadowBundleCommand(args[1:])
	default:
		return fmt.Errorf("unknown automation migration subcommand: %s", args[0])
	}
}

func runAutomationRunTaskSmokeCommand(args []string) error {
	flags := flag.NewFlagSet("automation e2e run-task-smoke", flag.ContinueOnError)
	executor := flags.String("executor", "", "required executor (local|kubernetes|ray)")
	title := flags.String("title", "", "task title")
	entrypoint := flags.String("entrypoint", "", "task entrypoint")
	image := flags.String("image", "", "container image")
	baseURL := flags.String("base-url", envOrDefault("BIGCLAW_ADDR", "http://127.0.0.1:8080"), "BigClaw API base URL")
	goRoot := flags.String("go-root", ".", "bigclaw-go repo root")
	timeoutSeconds := flags.Int("timeout-seconds", 180, "task timeout seconds")
	pollInterval := flags.Duration("poll-interval", time.Second, "status poll interval")
	runtimeEnvJSON := flags.String("runtime-env-json", "", "runtime_env JSON payload")
	metadataJSON := flags.String("metadata-json", "", "metadata JSON payload")
	reportPath := flags.String("report-path", "", "relative or absolute report path")
	autostart := flags.Bool("autostart", false, "autostart bigclawd with temporary state")
	asJSON := flags.Bool("json", true, "json")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation e2e run-task-smoke [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	if trim(*executor) == "" || trim(*title) == "" || trim(*entrypoint) == "" {
		return errors.New("--executor, --title, and --entrypoint are required")
	}
	report, exitCode, err := automationRunTaskSmoke(automationRunTaskSmokeOptions{
		Executor:       trim(*executor),
		Title:          *title,
		Entrypoint:     *entrypoint,
		Image:          *image,
		BaseURL:        *baseURL,
		GoRoot:         absPath(*goRoot),
		TimeoutSeconds: *timeoutSeconds,
		PollInterval:   *pollInterval,
		RuntimeEnvJSON: *runtimeEnvJSON,
		MetadataJSON:   *metadataJSON,
		ReportPath:     *reportPath,
		Autostart:      *autostart,
		HTTPClient:     http.DefaultClient,
	})
	if report != nil {
		return emit(structToMap(report), *asJSON, exitCode)
	}
	if err != nil {
		return err
	}
	return nil
}

func runAutomationShadowCompareCommand(args []string) error {
	flags := flag.NewFlagSet("automation migration shadow-compare", flag.ContinueOnError)
	primary := flags.String("primary", "", "primary base URL")
	shadow := flags.String("shadow", "", "shadow base URL")
	taskFile := flags.String("task-file", "", "task JSON file")
	timeoutSeconds := flags.Int("timeout-seconds", 180, "task timeout seconds")
	healthTimeoutSeconds := flags.Int("health-timeout-seconds", 60, "health wait timeout seconds")
	reportPath := flags.String("report-path", "", "relative or absolute report path")
	asJSON := flags.Bool("json", true, "json")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation migration shadow-compare [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	if trim(*primary) == "" || trim(*shadow) == "" || trim(*taskFile) == "" {
		return errors.New("--primary, --shadow, and --task-file are required")
	}
	report, exitCode, err := automationShadowCompare(automationShadowCompareOptions{
		PrimaryBaseURL:       *primary,
		ShadowBaseURL:        *shadow,
		TaskFile:             *taskFile,
		TimeoutSeconds:       *timeoutSeconds,
		HealthTimeoutSeconds: *healthTimeoutSeconds,
		ReportPath:           *reportPath,
		HTTPClient:           http.DefaultClient,
	})
	if report != nil {
		return emit(structToMap(report), *asJSON, exitCode)
	}
	if err != nil {
		return err
	}
	return nil
}

func runAutomationShadowMatrixCommand(args []string) error {
	flags := flag.NewFlagSet("automation migration shadow-matrix", flag.ContinueOnError)
	primary := flags.String("primary", "", "primary base URL")
	shadow := flags.String("shadow", "", "shadow base URL")
	var taskFiles multiStringFlag
	flags.Var(&taskFiles, "task-file", "task JSON file")
	corpusManifest := flags.String("corpus-manifest", "", "corpus manifest JSON file")
	replayCorpusSlices := flags.Bool("replay-corpus-slices", false, "submit replayable corpus slices")
	timeoutSeconds := flags.Int("timeout-seconds", 180, "task timeout seconds")
	healthTimeoutSeconds := flags.Int("health-timeout-seconds", 60, "health wait timeout seconds")
	reportPath := flags.String("report-path", "", "relative or absolute report path")
	asJSON := flags.Bool("json", true, "json")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation migration shadow-matrix [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	if trim(*primary) == "" || trim(*shadow) == "" {
		return errors.New("--primary and --shadow are required")
	}
	if len(taskFiles) == 0 && trim(*corpusManifest) == "" {
		return errors.New("at least one --task-file or --corpus-manifest must be provided")
	}
	report, exitCode, err := automationShadowMatrix(automationShadowMatrixOptions{
		PrimaryBaseURL:       *primary,
		ShadowBaseURL:        *shadow,
		TaskFiles:            append([]string(nil), taskFiles...),
		CorpusManifest:       *corpusManifest,
		ReplayCorpusSlices:   *replayCorpusSlices,
		TimeoutSeconds:       *timeoutSeconds,
		HealthTimeoutSeconds: *healthTimeoutSeconds,
		ReportPath:           *reportPath,
		HTTPClient:           http.DefaultClient,
	})
	if report != nil {
		return emit(report, *asJSON, exitCode)
	}
	if err != nil {
		return err
	}
	return nil
}

func runAutomationLiveShadowScorecardCommand(args []string) error {
	flags := flag.NewFlagSet("automation migration live-shadow-scorecard", flag.ContinueOnError)
	shadowCompareReport := flags.String("shadow-compare-report", "bigclaw-go/docs/reports/shadow-compare-report.json", "shadow compare report path")
	shadowMatrixReport := flags.String("shadow-matrix-report", "bigclaw-go/docs/reports/shadow-matrix-report.json", "shadow matrix report path")
	output := flags.String("output", "bigclaw-go/docs/reports/live-shadow-mirror-scorecard.json", "output path")
	asJSON := flags.Bool("json", true, "json")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation migration live-shadow-scorecard [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	report, err := automationLiveShadowScorecard(automationLiveShadowScorecardOptions{
		ShadowCompareReportPath: *shadowCompareReport,
		ShadowMatrixReportPath:  *shadowMatrixReport,
		OutputPath:              *output,
	})
	if err != nil {
		return err
	}
	return emit(report, *asJSON, 0)
}

func runAutomationExportLiveShadowBundleCommand(args []string) error {
	flags := flag.NewFlagSet("automation migration export-live-shadow-bundle", flag.ContinueOnError)
	goRoot := flags.String("go-root", "bigclaw-go", "repo root")
	shadowCompareReport := flags.String("shadow-compare-report", "docs/reports/shadow-compare-report.json", "shadow compare report path")
	shadowMatrixReport := flags.String("shadow-matrix-report", "docs/reports/shadow-matrix-report.json", "shadow matrix report path")
	scorecardReport := flags.String("scorecard-report", "docs/reports/live-shadow-mirror-scorecard.json", "scorecard report path")
	bundleRoot := flags.String("bundle-root", "docs/reports/live-shadow-runs", "bundle root")
	summaryPath := flags.String("summary-path", "docs/reports/live-shadow-summary.json", "summary output path")
	indexPath := flags.String("index-path", "docs/reports/live-shadow-index.md", "index markdown path")
	manifestPath := flags.String("manifest-path", "docs/reports/live-shadow-index.json", "manifest json path")
	rollupPath := flags.String("rollup-path", "docs/reports/live-shadow-drift-rollup.json", "rollup path")
	runID := flags.String("run-id", "", "bundle run id")
	asJSON := flags.Bool("json", true, "json")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation migration export-live-shadow-bundle [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	report, err := automationExportLiveShadowBundle(automationExportLiveShadowBundleOptions{
		GoRoot:            *goRoot,
		ShadowComparePath: *shadowCompareReport,
		ShadowMatrixPath:  *shadowMatrixReport,
		ScorecardPath:     *scorecardReport,
		BundleRoot:        *bundleRoot,
		SummaryPath:       *summaryPath,
		IndexPath:         *indexPath,
		ManifestPath:      *manifestPath,
		RollupPath:        *rollupPath,
		RunID:             *runID,
	})
	if err != nil {
		return err
	}
	return emit(report, *asJSON, 0)
}

func runAutomationSoakLocalCommand(args []string) error {
	flags := flag.NewFlagSet("automation benchmark soak-local", flag.ContinueOnError)
	count := flags.Int("count", 50, "task count")
	workers := flags.Int("workers", 8, "concurrent workers")
	baseURL := flags.String("base-url", "http://127.0.0.1:8080", "BigClaw API base URL")
	goRoot := flags.String("go-root", ".", "bigclaw-go repo root")
	timeoutSeconds := flags.Int("timeout-seconds", 180, "task timeout seconds")
	autostart := flags.Bool("autostart", false, "autostart bigclawd with temporary state")
	reportPath := flags.String("report-path", "docs/reports/soak-local-report.json", "relative or absolute report path")
	asJSON := flags.Bool("json", true, "json")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation benchmark soak-local [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	if *count <= 0 || *workers <= 0 {
		return errors.New("--count and --workers must be > 0")
	}
	report, exitCode, err := automationSoakLocal(automationSoakLocalOptions{
		Count:          *count,
		Workers:        *workers,
		BaseURL:        *baseURL,
		GoRoot:         absPath(*goRoot),
		TimeoutSeconds: *timeoutSeconds,
		Autostart:      *autostart,
		ReportPath:     *reportPath,
		HTTPClient:     http.DefaultClient,
	})
	if report != nil {
		return emit(structToMap(report), *asJSON, exitCode)
	}
	if err != nil {
		return err
	}
	return nil
}

func automationRunTaskSmoke(opts automationRunTaskSmokeOptions) (*automationRunTaskSmokeReport, int, error) {
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
	report := &automationRunTaskSmokeReport{}

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
			if err := automationWaitForHealth(client, activeBaseURL, 60, time.Second, sleep); err != nil {
				return nil, 0, err
			}
		}
	} else if err := automationWaitForHealth(client, activeBaseURL, 60, time.Second, sleep); err != nil {
		return nil, 0, err
	}

	task := automationTask{
		ID:                      fmt.Sprintf("%s-smoke-%d", opts.Executor, now().Unix()),
		Title:                   opts.Title,
		RequiredExecutor:        opts.Executor,
		Entrypoint:              opts.Entrypoint,
		ContainerImage:          opts.Image,
		ExecutionTimeoutSeconds: opts.TimeoutSeconds,
		Metadata: map[string]string{
			"smoke_test": "true",
			"executor":   opts.Executor,
		},
	}
	if trim(opts.RuntimeEnvJSON) != "" {
		if err := json.Unmarshal([]byte(opts.RuntimeEnvJSON), &task.RuntimeEnv); err != nil {
			return nil, 0, fmt.Errorf("decode --runtime-env-json: %w", err)
		}
	}
	if trim(opts.MetadataJSON) != "" {
		extra := map[string]string{}
		if err := json.Unmarshal([]byte(opts.MetadataJSON), &extra); err != nil {
			return nil, 0, fmt.Errorf("decode --metadata-json: %w", err)
		}
		for key, value := range extra {
			task.Metadata[key] = value
		}
	}
	submitted := map[string]any{}
	if err := automationRequestJSON(client, http.MethodPost, activeBaseURL, "/tasks", task, &submitted); err != nil {
		return nil, 0, err
	}
	taskPayload, _ := submitted["task"].(map[string]any)
	if taskPayload == nil {
		taskPayload = structToMap(task)
	}
	deadline := now().Add(time.Duration(opts.TimeoutSeconds) * time.Second)
	for now().Before(deadline) {
		status := map[string]any{}
		if err := automationRequestJSON(client, http.MethodGet, activeBaseURL, "/tasks/"+taskPayload["id"].(string), nil, &status); err != nil {
			return nil, 0, err
		}
		if automationIsTerminal(status["state"]) {
			events, err := automationFetchEvents(client, activeBaseURL, taskPayload["id"].(string))
			if err != nil {
				return nil, 0, err
			}
			report.Autostarted = cmd != nil
			report.BaseURL = activeBaseURL
			report.Task = taskPayload
			report.Status = status
			report.Events = events
			report.StateDir = stateDir
			report.ServiceLog = serviceLog
			if err := automationWriteReport(opts.GoRoot, opts.ReportPath, report); err != nil {
				return nil, 0, err
			}
			if status["state"] == "succeeded" {
				return report, 0, nil
			}
			return report, 1, nil
		}
		sleep(opts.PollInterval)
	}
	status := map[string]any{}
	_ = automationRequestJSON(client, http.MethodGet, activeBaseURL, "/tasks/"+taskPayload["id"].(string), nil, &status)
	events, _ := automationFetchEvents(client, activeBaseURL, taskPayload["id"].(string))
	report.Autostarted = cmd != nil
	report.BaseURL = activeBaseURL
	report.Task = taskPayload
	report.Status = status
	report.Events = events
	report.StateDir = stateDir
	report.ServiceLog = serviceLog
	report.Error = "timeout waiting for terminal state"
	if err := automationWriteReport(opts.GoRoot, opts.ReportPath, report); err != nil {
		return nil, 0, err
	}
	return report, 1, nil
}

func automationShadowCompare(opts automationShadowCompareOptions) (*automationShadowCompareReport, int, error) {
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
	if err := automationWaitForHealth(client, opts.PrimaryBaseURL, opts.HealthTimeoutSeconds, time.Second, sleep); err != nil {
		return nil, 0, err
	}
	if err := automationWaitForHealth(client, opts.ShadowBaseURL, opts.HealthTimeoutSeconds, time.Second, sleep); err != nil {
		return nil, 0, err
	}

	var task automationTask
	body, err := os.ReadFile(opts.TaskFile)
	if err != nil {
		return nil, 0, err
	}
	if err := json.Unmarshal(body, &task); err != nil {
		return nil, 0, err
	}
	baseID := task.ID
	if trim(baseID) == "" {
		baseID = fmt.Sprintf("shadow-%d", now().Unix())
	}
	traceID := task.TraceID
	if trim(traceID) == "" {
		traceID = baseID
	}
	primaryTask := task
	shadowTask := task
	primaryTask.ID = baseID + "-primary"
	shadowTask.ID = baseID + "-shadow"
	primaryTask.TraceID = traceID
	shadowTask.TraceID = traceID

	if err := automationRequestJSON(client, http.MethodPost, opts.PrimaryBaseURL, "/tasks", primaryTask, nil); err != nil {
		return nil, 0, err
	}
	if err := automationRequestJSON(client, http.MethodPost, opts.ShadowBaseURL, "/tasks", shadowTask, nil); err != nil {
		return nil, 0, err
	}
	primaryStatus, err := automationWaitForTask(client, opts.PrimaryBaseURL, primaryTask.ID, time.Duration(opts.TimeoutSeconds)*time.Second, sleep)
	if err != nil {
		return nil, 0, err
	}
	shadowStatus, err := automationWaitForTask(client, opts.ShadowBaseURL, shadowTask.ID, time.Duration(opts.TimeoutSeconds)*time.Second, sleep)
	if err != nil {
		return nil, 0, err
	}
	primaryEvents, err := automationFetchEvents(client, opts.PrimaryBaseURL, primaryTask.ID)
	if err != nil {
		return nil, 0, err
	}
	shadowEvents, err := automationFetchEvents(client, opts.ShadowBaseURL, shadowTask.ID)
	if err != nil {
		return nil, 0, err
	}
	report := &automationShadowCompareReport{TraceID: traceID}
	report.Primary.TaskID = primaryTask.ID
	report.Primary.Status = primaryStatus
	report.Primary.Events = primaryEvents
	report.Shadow.TaskID = shadowTask.ID
	report.Shadow.Status = shadowStatus
	report.Shadow.Events = shadowEvents
	report.Diff.StateEqual = primaryStatus["state"] == shadowStatus["state"]
	report.Diff.EventCountDelta = len(primaryEvents) - len(shadowEvents)
	report.Diff.PrimaryEventTypes = automationEventTypes(primaryEvents)
	report.Diff.ShadowEventTypes = automationEventTypes(shadowEvents)
	report.Diff.EventTypesEqual = automationStringSlicesEqual(report.Diff.PrimaryEventTypes, report.Diff.ShadowEventTypes)
	report.Diff.PrimaryTimelineSeconds = automationTimelineSeconds(primaryEvents)
	report.Diff.ShadowTimelineSeconds = automationTimelineSeconds(shadowEvents)
	if err := automationWriteReport(".", opts.ReportPath, report); err != nil {
		return nil, 0, err
	}
	if report.Diff.StateEqual && report.Diff.EventTypesEqual {
		return report, 0, nil
	}
	return report, 1, nil
}

type automationShadowMatrixExecutionEntry struct {
	Task        map[string]any
	SourceKind  string
	SourceFile  string
	TaskShape   string
	CorpusSlice map[string]any
}

type automationShadowMatrixCorpusSlice struct {
	SliceID    string
	Title      string
	Weight     int
	Tags       []string
	TaskShape  string
	Task       map[string]any
	SourceFile string
	Replay     bool
	Notes      string
}

type automationShadowMatrixManifestMeta struct {
	Name        string
	GeneratedAt any
	SourceFile  string
}

func automationShadowMatrix(opts automationShadowMatrixOptions) (map[string]any, int, error) {
	fixtureEntries, err := automationShadowMatrixLoadFixtureEntries(opts.TaskFiles)
	if err != nil {
		return nil, 0, err
	}
	var manifestMeta *automationShadowMatrixManifestMeta
	corpusSlices := []automationShadowMatrixCorpusSlice{}
	replayEntries := []automationShadowMatrixExecutionEntry{}
	if trim(opts.CorpusManifest) != "" {
		var err error
		manifestMeta, replayEntries, corpusSlices, err = automationShadowMatrixLoadCorpusManifestEntries(opts.CorpusManifest, opts.ReplayCorpusSlices)
		if err != nil {
			return nil, 0, err
		}
	}

	executionEntries := append(fixtureEntries, replayEntries...)
	results := make([]map[string]any, 0, len(executionEntries))
	allMatched := true
	for index, entry := range executionEntries {
		task := cloneMap(entry.Task)
		baseID, _ := task["id"].(string)
		if trim(baseID) == "" {
			baseID = fmt.Sprintf("matrix-task-%d", index+1)
		}
		task["id"] = fmt.Sprintf("%s-m%d", baseID, index+1)
		result, exitCode, err := automationShadowMatrixCompareTask(opts, task)
		if err != nil {
			return nil, 0, err
		}
		if exitCode != 0 {
			allMatched = false
		}
		result["source_file"] = entry.SourceFile
		result["source_kind"] = entry.SourceKind
		result["task_shape"] = entry.TaskShape
		if entry.CorpusSlice != nil {
			result["corpus_slice"] = entry.CorpusSlice
		}
		results = append(results, result)
	}

	report := map[string]any{
		"total": len(results),
		"matched": func() int {
			count := 0
			for _, item := range results {
				diff, _ := item["diff"].(map[string]any)
				if diff["state_equal"] == true && diff["event_types_equal"] == true {
					count++
				}
			}
			return count
		}(),
		"mismatched": len(results),
		"inputs": map[string]any{
			"fixture_task_count": len(fixtureEntries),
			"corpus_slice_count": len(corpusSlices),
			"manifest_name": func() any {
				if manifestMeta == nil {
					return nil
				}
				return manifestMeta.Name
			}(),
		},
		"results": results,
	}
	report["mismatched"] = report["total"].(int) - report["matched"].(int)
	if manifestMeta != nil && len(corpusSlices) > 0 {
		report["corpus_coverage"] = automationShadowMatrixBuildCorpusCoverage(fixtureEntries, corpusSlices, *manifestMeta)
	}
	if err := automationWriteReport(".", opts.ReportPath, report); err != nil {
		return nil, 0, err
	}
	if allMatched {
		return report, 0, nil
	}
	return report, 1, nil
}

func automationShadowMatrixCompareTask(opts automationShadowMatrixOptions, task map[string]any) (map[string]any, int, error) {
	body, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return nil, 0, err
	}
	taskFile, err := os.CreateTemp("", "shadow-matrix-task-*.json")
	if err != nil {
		return nil, 0, err
	}
	defer os.Remove(taskFile.Name())
	if _, err := taskFile.Write(append(body, '\n')); err != nil {
		_ = taskFile.Close()
		return nil, 0, err
	}
	if err := taskFile.Close(); err != nil {
		return nil, 0, err
	}
	report, exitCode, err := automationShadowCompare(automationShadowCompareOptions{
		PrimaryBaseURL:       opts.PrimaryBaseURL,
		ShadowBaseURL:        opts.ShadowBaseURL,
		TaskFile:             taskFile.Name(),
		TimeoutSeconds:       opts.TimeoutSeconds,
		HealthTimeoutSeconds: opts.HealthTimeoutSeconds,
		HTTPClient:           opts.HTTPClient,
		Now:                  opts.Now,
		Sleep:                opts.Sleep,
	})
	if err != nil {
		return nil, 0, err
	}
	return structToMap(report), exitCode, nil
}

func automationShadowMatrixLoadFixtureEntries(taskFiles []string) ([]automationShadowMatrixExecutionEntry, error) {
	entries := make([]automationShadowMatrixExecutionEntry, 0, len(taskFiles))
	for _, taskFile := range taskFiles {
		task, err := automationShadowMatrixLoadJSON(taskFile)
		if err != nil {
			return nil, err
		}
		entries = append(entries, automationShadowMatrixMakeExecutionEntry(task, "fixture", taskFile, automationShadowMatrixDeriveTaskShape(task), nil))
	}
	return entries, nil
}

func automationShadowMatrixLoadCorpusManifestEntries(manifestPath string, replayCorpusSlices bool) (*automationShadowMatrixManifestMeta, []automationShadowMatrixExecutionEntry, []automationShadowMatrixCorpusSlice, error) {
	manifest, err := automationShadowMatrixLoadJSON(manifestPath)
	if err != nil {
		return nil, nil, nil, err
	}
	rawSlices, _ := manifest["slices"].([]any)
	if rawSlices == nil {
		return nil, nil, nil, errors.New("corpus manifest must contain a top-level slices array")
	}
	coverageSlices := make([]automationShadowMatrixCorpusSlice, 0, len(rawSlices))
	replayEntries := []automationShadowMatrixExecutionEntry{}
	for index, rawSlice := range rawSlices {
		sliceMap, _ := rawSlice.(map[string]any)
		sliceData, err := automationShadowMatrixNormalizeCorpusSlice(sliceMap, index+1, manifestPath)
		if err != nil {
			return nil, nil, nil, err
		}
		coverageSlices = append(coverageSlices, sliceData)
		if replayCorpusSlices && sliceData.Task != nil {
			replayEntries = append(replayEntries, automationShadowMatrixMakeExecutionEntry(sliceData.Task, "corpus", sliceData.SourceFile, sliceData.TaskShape, map[string]any{
				"id":     sliceData.SliceID,
				"title":  sliceData.Title,
				"weight": sliceData.Weight,
				"tags":   sliceData.Tags,
			}))
		}
	}
	name, _ := manifest["name"].(string)
	if trim(name) == "" {
		name = strings.TrimSuffix(filepath.Base(manifestPath), filepath.Ext(manifestPath))
	}
	return &automationShadowMatrixManifestMeta{
		Name:        name,
		GeneratedAt: manifest["generated_at"],
		SourceFile:  manifestPath,
	}, replayEntries, coverageSlices, nil
}

func automationShadowMatrixNormalizeCorpusSlice(sliceData map[string]any, index int, manifestPath string) (automationShadowMatrixCorpusSlice, error) {
	sliceID, _ := sliceData["slice_id"].(string)
	if trim(sliceID) == "" {
		sliceID = fmt.Sprintf("corpus-slice-%d", index)
	}
	title, _ := sliceData["title"].(string)
	if trim(title) == "" {
		title = sliceID
	}
	weight := automationInt(sliceData["weight"], 1)
	tags := automationStringSlice(sliceData["tags"])
	var task map[string]any
	var sourceFile string
	if rawTaskFile, ok := sliceData["task_file"].(string); ok && trim(rawTaskFile) != "" {
		sourceFile = rawTaskFile
		resolved := rawTaskFile
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(filepath.Dir(absPath(manifestPath)), rawTaskFile)
		}
		var err error
		task, err = automationShadowMatrixLoadJSON(resolved)
		if err != nil {
			return automationShadowMatrixCorpusSlice{}, err
		}
	} else if rawTask, ok := sliceData["task"].(map[string]any); ok {
		task = cloneMap(rawTask)
		sourceFile = fmt.Sprintf("%s#%s", filepath.Base(manifestPath), sliceID)
	}
	taskShape, _ := sliceData["task_shape"].(string)
	if trim(taskShape) == "" && task != nil {
		taskShape = automationShadowMatrixDeriveTaskShape(task)
	}
	if trim(taskShape) == "" {
		return automationShadowMatrixCorpusSlice{}, fmt.Errorf("corpus slice %s must define task_shape or provide task/task_file", sliceID)
	}
	notes, _ := sliceData["notes"].(string)
	return automationShadowMatrixCorpusSlice{
		SliceID:    sliceID,
		Title:      title,
		Weight:     weight,
		Tags:       tags,
		TaskShape:  taskShape,
		Task:       task,
		SourceFile: sourceFile,
		Replay:     automationBool(sliceData["replay"]),
		Notes:      notes,
	}, nil
}

func automationShadowMatrixBuildCorpusCoverage(fixtureEntries []automationShadowMatrixExecutionEntry, corpusSlices []automationShadowMatrixCorpusSlice, manifestMeta automationShadowMatrixManifestMeta) map[string]any {
	fixtureByShape := map[string][]automationShadowMatrixExecutionEntry{}
	for _, entry := range fixtureEntries {
		fixtureByShape[entry.TaskShape] = append(fixtureByShape[entry.TaskShape], entry)
	}
	type shapeAggregate struct {
		SliceCount      int
		ReplayableCount int
		CorpusWeight    int
		SliceIDs        []string
		Titles          []string
	}
	corpusByShape := map[string]*shapeAggregate{}
	for _, sliceData := range corpusSlices {
		aggregate := corpusByShape[sliceData.TaskShape]
		if aggregate == nil {
			aggregate = &shapeAggregate{}
			corpusByShape[sliceData.TaskShape] = aggregate
		}
		aggregate.SliceCount++
		if sliceData.Task != nil {
			aggregate.ReplayableCount++
		}
		aggregate.CorpusWeight += sliceData.Weight
		aggregate.SliceIDs = append(aggregate.SliceIDs, sliceData.SliceID)
		aggregate.Titles = append(aggregate.Titles, sliceData.Title)
	}
	taskShapes := make([]string, 0, len(corpusByShape))
	for taskShape := range corpusByShape {
		taskShapes = append(taskShapes, taskShape)
	}
	sort.Slice(taskShapes, func(i, j int) bool {
		left := corpusByShape[taskShapes[i]]
		right := corpusByShape[taskShapes[j]]
		if left.CorpusWeight != right.CorpusWeight {
			return left.CorpusWeight > right.CorpusWeight
		}
		return taskShapes[i] < taskShapes[j]
	})
	shapeScorecard := make([]map[string]any, 0, len(taskShapes))
	for _, taskShape := range taskShapes {
		aggregate := corpusByShape[taskShape]
		fixtures := fixtureByShape[taskShape]
		sources := make([]string, 0, len(fixtures))
		for _, entry := range fixtures {
			sources = append(sources, entry.SourceFile)
		}
		shapeScorecard = append(shapeScorecard, map[string]any{
			"task_shape":             taskShape,
			"fixture_task_count":     len(fixtures),
			"fixture_sources":        sources,
			"corpus_slice_count":     aggregate.SliceCount,
			"replayable_slice_count": aggregate.ReplayableCount,
			"corpus_weight":          aggregate.CorpusWeight,
			"corpus_slice_ids":       aggregate.SliceIDs,
			"corpus_titles":          aggregate.Titles,
			"covered_by_fixture":     len(fixtures) > 0,
		})
	}
	uncoveredSlices := []map[string]any{}
	for _, sliceData := range corpusSlices {
		if len(fixtureByShape[sliceData.TaskShape]) > 0 {
			continue
		}
		uncoveredSlices = append(uncoveredSlices, map[string]any{
			"slice_id":    sliceData.SliceID,
			"title":       sliceData.Title,
			"task_shape":  sliceData.TaskShape,
			"weight":      sliceData.Weight,
			"replayable":  sliceData.Task != nil,
			"source_file": sliceData.SourceFile,
			"tags":        sliceData.Tags,
			"notes":       sliceData.Notes,
		})
	}
	replayableCount := 0
	for _, sliceData := range corpusSlices {
		if sliceData.Task != nil {
			replayableCount++
		}
	}
	return map[string]any{
		"manifest_name":                 manifestMeta.Name,
		"manifest_source_file":          manifestMeta.SourceFile,
		"generated_at":                  manifestMeta.GeneratedAt,
		"fixture_task_count":            len(fixtureEntries),
		"corpus_slice_count":            len(corpusSlices),
		"corpus_replayable_slice_count": replayableCount,
		"covered_corpus_slice_count":    len(corpusSlices) - len(uncoveredSlices),
		"uncovered_corpus_slice_count":  len(uncoveredSlices),
		"shape_scorecard":               shapeScorecard,
		"uncovered_slices":              uncoveredSlices,
	}
}

func automationShadowMatrixMakeExecutionEntry(task map[string]any, sourceKind string, sourceFile string, taskShape string, corpusSlice map[string]any) automationShadowMatrixExecutionEntry {
	entryTask := cloneMap(task)
	if entryTask == nil {
		entryTask = map[string]any{}
	}
	entryTask["_source_file"] = sourceFile
	if trim(taskShape) == "" {
		taskShape = automationShadowMatrixDeriveTaskShape(entryTask)
	}
	return automationShadowMatrixExecutionEntry{
		Task:        entryTask,
		SourceKind:  sourceKind,
		SourceFile:  sourceFile,
		TaskShape:   taskShape,
		CorpusSlice: corpusSlice,
	}
}

func automationShadowMatrixDeriveTaskShape(task map[string]any) string {
	features := []string{}
	executor, _ := task["required_executor"].(string)
	if trim(executor) == "" {
		executor, _ = task["executor"].(string)
	}
	if trim(executor) == "" {
		executor = "default"
	}
	features = append(features, "executor:"+executor)
	labels := automationStringSlice(task["labels"])
	sort.Strings(labels)
	if len(labels) > 0 {
		features = append(features, "labels:"+strings.Join(labels, ","))
	}
	if task["budget_cents"] != nil {
		features = append(features, "budgeted")
	}
	if values, ok := task["acceptance_criteria"].([]any); ok && len(values) > 0 {
		features = append(features, "acceptance")
	}
	if values, ok := task["validation_plan"].([]any); ok && len(values) > 0 {
		features = append(features, "validation-plan")
	}
	if metadata, ok := task["metadata"].(map[string]any); ok {
		if scenario, _ := metadata["scenario"].(string); trim(scenario) != "" {
			features = append(features, "scenario:"+scenario)
		}
	}
	return strings.Join(features, "|")
}

func automationShadowMatrixLoadJSON(path string) (map[string]any, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func automationLiveShadowScorecard(opts automationLiveShadowScorecardOptions) (map[string]any, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	compareReport, err := automationShadowMatrixLoadJSON(resolveAutomationPath(opts.ShadowCompareReportPath))
	if err != nil {
		return nil, err
	}
	matrixReport, err := automationShadowMatrixLoadJSON(resolveAutomationPath(opts.ShadowMatrixReportPath))
	if err != nil {
		return nil, err
	}
	generatedAt := now().UTC()
	parityEntries := append([]map[string]any{automationLiveShadowBuildCompareEntry(compareReport)}, automationLiveShadowBuildMatrixEntries(matrixReport)...)
	parityOKCount := 0
	for _, item := range parityEntries {
		if parity, ok := item["parity"].(map[string]any); ok && parity["status"] == "parity-ok" {
			parityOKCount++
		}
	}
	driftDetectedCount := len(parityEntries) - parityOKCount
	freshness := []map[string]any{
		automationLiveShadowBuildFreshnessEntry("shadow-compare-report", opts.ShadowCompareReportPath, compareReport, generatedAt),
		automationLiveShadowBuildFreshnessEntry("shadow-matrix-report", opts.ShadowMatrixReportPath, matrixReport, generatedAt),
	}
	staleInputs := 0
	var latestEvidenceTimestamp string
	for _, item := range freshness {
		if item["status"] != "fresh" {
			staleInputs++
		}
		if ts, _ := item["latest_evidence_timestamp"].(string); trim(ts) != "" && ts > latestEvidenceTimestamp {
			latestEvidenceTimestamp = ts
		}
	}
	matrixCorpusCoverage, _ := matrixReport["corpus_coverage"].(map[string]any)
	cutoverCheckpoints := []map[string]any{
		automationLiveShadowCheck("single_compare_matches_terminal_state_and_event_sequence",
			lookupBool(compareReport, "diff", "state_equal") && lookupBool(compareReport, "diff", "event_types_equal"),
			fmt.Sprintf("trace_id=%v", compareReport["trace_id"])),
		automationLiveShadowCheck("matrix_reports_no_state_or_event_sequence_mismatches",
			automationInt(matrixReport["mismatched"], 0) == 0,
			fmt.Sprintf("matched=%v mismatched=%v", matrixReport["matched"], matrixReport["mismatched"])),
		automationLiveShadowCheck("scorecard_detects_no_parity_drift",
			driftDetectedCount == 0,
			fmt.Sprintf("parity_ok=%d drift_detected=%d", parityOKCount, driftDetectedCount)),
		automationLiveShadowCheck("checked_in_evidence_is_fresh_enough_for_review",
			staleInputs == 0,
			fmt.Sprintf("freshness_statuses=%v", collectStatuses(freshness))),
		automationLiveShadowCheck("matrix_includes_corpus_coverage_overlay",
			len(matrixCorpusCoverage) > 0,
			fmt.Sprintf("corpus_slice_count=%v", matrixCorpusCoverage["corpus_slice_count"])),
	}
	report := map[string]any{
		"generated_at": utcISOTime(generatedAt),
		"ticket":       "BIG-PAR-092",
		"title":        "Live shadow mirror parity drift scorecard",
		"status":       "repo-native-live-shadow-scorecard",
		"evidence_inputs": map[string]any{
			"shadow_compare_report_path": opts.ShadowCompareReportPath,
			"shadow_matrix_report_path":  opts.ShadowMatrixReportPath,
			"generator_script":           "go run ./cmd/bigclawctl automation migration live-shadow-scorecard",
		},
		"summary": map[string]any{
			"total_evidence_runs":          len(parityEntries),
			"parity_ok_count":              parityOKCount,
			"drift_detected_count":         driftDetectedCount,
			"matrix_total":                 automationInt(matrixReport["total"], 0),
			"matrix_matched":               automationInt(matrixReport["matched"], 0),
			"matrix_mismatched":            automationInt(matrixReport["mismatched"], 0),
			"corpus_coverage_present":      len(matrixCorpusCoverage) > 0,
			"corpus_uncovered_slice_count": matrixCorpusCoverage["uncovered_corpus_slice_count"],
			"latest_evidence_timestamp":    stringOrNil(latestEvidenceTimestamp),
			"fresh_inputs":                 len(freshness) - staleInputs,
			"stale_inputs":                 staleInputs,
		},
		"freshness":           freshness,
		"parity_entries":      parityEntries,
		"cutover_checkpoints": cutoverCheckpoints,
		"limitations": []string{
			"repo-native only: this scorecard summarizes checked-in shadow artifacts rather than an always-on production traffic mirror",
			"parity drift is measured from fixture-backed compare/matrix runs and optional corpus slices, not mirrored tenant traffic",
			"freshness is derived from the latest artifact event timestamps and should be treated as evidence recency, not live service health",
		},
		"future_work": []string{
			"replace offline fixture submission with a real ingress mirror or tenant-scoped shadow routing control before treating this as cutover-proof traffic parity",
			"promote parity drift review from checked-in artifacts into a continuously refreshed operational surface",
			"pair this scorecard with rollback automation only after tenant-scoped rollback guardrails exist",
		},
	}
	if err := automationWriteReport(".", opts.OutputPath, report); err != nil {
		return nil, err
	}
	return report, nil
}

func automationLiveShadowBuildFreshnessEntry(name string, path string, report map[string]any, generatedAt time.Time) map[string]any {
	latestTimestamp := automationLiveShadowLatestReportTimestamp(report)
	var latestISO any
	var ageHours any
	status := "missing-timestamps"
	if !latestTimestamp.IsZero() {
		latestISO = utcISOTime(latestTimestamp)
		ageHours = roundTo((generatedAt.Sub(latestTimestamp).Seconds())/3600, 2)
		if generatedAt.Sub(latestTimestamp).Hours() <= 168 {
			status = "fresh"
		} else {
			status = "stale"
		}
	}
	return map[string]any{
		"name":                      name,
		"report_path":               path,
		"latest_evidence_timestamp": latestISO,
		"age_hours":                 ageHours,
		"freshness_slo_hours":       168,
		"status":                    status,
	}
}

func automationLiveShadowLatestReportTimestamp(report map[string]any) time.Time {
	var timestamps []time.Time
	if results, ok := report["results"].([]any); ok {
		for _, raw := range results {
			item, _ := raw.(map[string]any)
			timestamps = append(timestamps, automationLiveShadowCollectEventTimestamps(lookupMap(item, "primary", "events"))...)
			timestamps = append(timestamps, automationLiveShadowCollectEventTimestamps(lookupMap(item, "shadow", "events"))...)
		}
	} else {
		timestamps = append(timestamps, automationLiveShadowCollectEventTimestamps(lookupMap(report, "primary", "events"))...)
		timestamps = append(timestamps, automationLiveShadowCollectEventTimestamps(lookupMap(report, "shadow", "events"))...)
	}
	latest := time.Time{}
	for _, ts := range timestamps {
		if ts.After(latest) {
			latest = ts
		}
	}
	return latest
}

func automationLiveShadowCollectEventTimestamps(events any) []time.Time {
	items, _ := events.([]any)
	out := []time.Time{}
	for _, raw := range items {
		event, _ := raw.(map[string]any)
		timestamp, _ := event["timestamp"].(string)
		if trim(timestamp) == "" {
			continue
		}
		if parsed, err := time.Parse(time.RFC3339, strings.ReplaceAll(timestamp, "Z", "+00:00")); err == nil {
			out = append(out, parsed)
		}
	}
	return out
}

func automationLiveShadowBuildCompareEntry(report map[string]any) map[string]any {
	diff, _ := lookupMap(report, "diff").(map[string]any)
	primary, _ := lookupMap(report, "primary").(map[string]any)
	shadow, _ := lookupMap(report, "shadow").(map[string]any)
	return map[string]any{
		"entry_type":      "single-compare",
		"label":           "single fixture compare",
		"trace_id":        report["trace_id"],
		"source_file":     nil,
		"source_kind":     "fixture",
		"parity":          automationLiveShadowClassifyParity(diff),
		"primary_task_id": primary["task_id"],
		"shadow_task_id":  shadow["task_id"],
	}
}

func automationLiveShadowBuildMatrixEntries(report map[string]any) []map[string]any {
	results, _ := report["results"].([]any)
	entries := make([]map[string]any, 0, len(results))
	for _, raw := range results {
		item, _ := raw.(map[string]any)
		diff, _ := lookupMap(item, "diff").(map[string]any)
		primary, _ := lookupMap(item, "primary").(map[string]any)
		shadow, _ := lookupMap(item, "shadow").(map[string]any)
		entries = append(entries, map[string]any{
			"entry_type":      "matrix-row",
			"label":           item["source_file"],
			"trace_id":        item["trace_id"],
			"source_file":     item["source_file"],
			"source_kind":     item["source_kind"],
			"task_shape":      item["task_shape"],
			"corpus_slice":    item["corpus_slice"],
			"parity":          automationLiveShadowClassifyParity(diff),
			"primary_task_id": primary["task_id"],
			"shadow_task_id":  shadow["task_id"],
		})
	}
	return entries
}

func automationLiveShadowClassifyParity(diff map[string]any) map[string]any {
	reasons := []string{}
	if !automationBool(diff["state_equal"]) {
		reasons = append(reasons, "terminal-state-mismatch")
	}
	if !automationBool(diff["event_types_equal"]) {
		reasons = append(reasons, "event-sequence-mismatch")
	}
	if automationInt(diff["event_count_delta"], 0) != 0 {
		reasons = append(reasons, "event-count-drift")
	}
	timelineDelta := roundTo(absFloat(asFloat(diff["primary_timeline_seconds"])-asFloat(diff["shadow_timeline_seconds"])), 6)
	if timelineDelta > 0.25 {
		reasons = append(reasons, "timeline-drift")
	}
	status := "parity-ok"
	if len(reasons) > 0 {
		status = "drift-detected"
	}
	return map[string]any{
		"status":                           status,
		"timeline_delta_seconds":           timelineDelta,
		"timeline_drift_tolerance_seconds": 0.25,
		"reasons":                          reasons,
	}
}

func automationLiveShadowCheck(name string, passed bool, detail string) map[string]any {
	return map[string]any{"name": name, "passed": passed, "detail": detail}
}

var liveShadowFollowupDigests = []struct {
	Path        string
	Description string
}{
	{Path: "docs/reports/live-shadow-comparison-follow-up-digest.md", Description: "Live shadow traffic comparison caveats are consolidated here."},
	{Path: "docs/reports/rollback-safeguard-follow-up-digest.md", Description: "Rollback remains operator-driven; this digest explains the guardrail visibility and trigger caveats."},
}

var liveShadowDocLinks = []struct {
	Path        string
	Description string
}{
	{Path: "docs/migration-shadow.md", Description: "Shadow helper workflow and bundle generation steps."},
	{Path: "docs/reports/migration-readiness-report.md", Description: "Migration readiness summary linked to the shadow bundle."},
	{Path: "docs/reports/migration-plan-review-notes.md", Description: "Review notes tied to the shadow bundle index."},
	{Path: "docs/reports/rollback-trigger-surface.json", Description: "Machine-readable rollback blockers, warnings, and manual-only paths linked from the shadow bundle."},
}

func automationExportLiveShadowBundle(opts automationExportLiveShadowBundleOptions) (map[string]any, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	root := resolveAutomationGoRoot(opts.GoRoot)
	compareReport, err := automationShadowMatrixLoadJSON(filepath.Join(root, opts.ShadowComparePath))
	if err != nil {
		return nil, err
	}
	matrixReport, err := automationShadowMatrixLoadJSON(filepath.Join(root, opts.ShadowMatrixPath))
	if err != nil {
		return nil, err
	}
	scorecardReport, err := automationShadowMatrixLoadJSON(filepath.Join(root, opts.ScorecardPath))
	if err != nil {
		return nil, err
	}
	generatedAt := now().UTC()
	runID := opts.RunID
	if trim(runID) == "" {
		runID = deriveLiveShadowRunID(scorecardReport, generatedAt)
	}
	bundleDir := filepath.Join(root, opts.BundleRoot, runID)
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return nil, err
	}
	latest, err := buildLiveShadowRunSummary(root, bundleDir, runID, compareReport, matrixReport, scorecardReport, generatedAt)
	if err != nil {
		return nil, err
	}
	if err := automationWriteReport(".", filepath.Join(bundleDir, "summary.json"), latest); err != nil {
		return nil, err
	}
	if err := automationWriteReport(".", filepath.Join(root, opts.SummaryPath), latest); err != nil {
		return nil, err
	}
	recentRuns, err := loadLiveShadowRecentRuns(filepath.Join(root, opts.BundleRoot))
	if err != nil {
		return nil, err
	}
	rollup := buildLiveShadowRollup(recentRuns, 5, generatedAt)
	manifest := map[string]any{
		"latest": latest,
		"recent_runs": func() []map[string]any {
			out := make([]map[string]any, 0, len(recentRuns))
			for _, item := range recentRuns {
				out = append(out, map[string]any{
					"run_id":       item["run_id"],
					"generated_at": item["generated_at"],
					"status":       item["status"],
					"severity":     item["severity"],
					"bundle_path":  item["bundle_path"],
					"summary_path": item["summary_path"],
				})
			}
			return out
		}(),
		"drift_rollup": rollup,
	}
	if err := automationWriteReport(".", filepath.Join(root, opts.RollupPath), rollup); err != nil {
		return nil, err
	}
	if err := automationWriteReport(".", filepath.Join(root, opts.ManifestPath), manifest); err != nil {
		return nil, err
	}
	indexText := renderLiveShadowIndex(latest, manifest["recent_runs"].([]map[string]any), rollup)
	if err := os.WriteFile(filepath.Join(root, opts.IndexPath), []byte(indexText), 0o644); err != nil {
		return nil, err
	}
	if err := copyTextFile(filepath.Join(root, opts.IndexPath), filepath.Join(bundleDir, "README.md")); err != nil {
		return nil, err
	}
	return manifest, nil
}

func buildLiveShadowRunSummary(root string, bundleDir string, runID string, compareReport map[string]any, matrixReport map[string]any, scorecardReport map[string]any, generatedAt time.Time) (map[string]any, error) {
	compareBundlePath, err := copyJSONFile(filepath.Join(root, "docs/reports/shadow-compare-report.json"), filepath.Join(bundleDir, "shadow-compare-report.json"))
	if err != nil {
		return nil, err
	}
	matrixBundlePath, err := copyJSONFile(filepath.Join(root, "docs/reports/shadow-matrix-report.json"), filepath.Join(bundleDir, "shadow-matrix-report.json"))
	if err != nil {
		return nil, err
	}
	scorecardBundlePath, err := copyJSONFile(filepath.Join(root, "docs/reports/live-shadow-mirror-scorecard.json"), filepath.Join(bundleDir, "live-shadow-mirror-scorecard.json"))
	if err != nil {
		return nil, err
	}
	rollbackBundlePath, err := copyJSONFile(filepath.Join(root, "docs/reports/rollback-trigger-surface.json"), filepath.Join(bundleDir, "rollback-trigger-surface.json"))
	if err != nil {
		return nil, err
	}
	rollbackReport, err := automationShadowMatrixLoadJSON(filepath.Join(root, "docs/reports/rollback-trigger-surface.json"))
	if err != nil {
		return nil, err
	}
	scorecardSummary, _ := scorecardReport["summary"].(map[string]any)
	freshness, _ := scorecardReport["freshness"].([]any)
	staleInputs := automationInt(scorecardSummary["stale_inputs"], 0)
	driftDetectedCount := automationInt(scorecardSummary["drift_detected_count"], 0)
	severity := classifyLiveShadowSeverity(scorecardReport)
	status := "parity-ok"
	if severityRank(severity) > 0 {
		status = "attention-needed"
	}
	results, _ := matrixReport["results"].([]any)
	matrixTraceIDs := []string{}
	for _, raw := range results {
		item, _ := raw.(map[string]any)
		if traceID, _ := item["trace_id"].(string); trim(traceID) != "" {
			matrixTraceIDs = append(matrixTraceIDs, traceID)
		}
	}
	return map[string]any{
		"run_id":       runID,
		"generated_at": utcISOTime(generatedAt),
		"status":       status,
		"severity":     severity,
		"bundle_path":  relAutomationPath(bundleDir, root),
		"summary_path": relAutomationPath(filepath.Join(bundleDir, "summary.json"), root),
		"artifacts": map[string]any{
			"shadow_compare_report_path":    relAutomationPath(compareBundlePath, root),
			"shadow_matrix_report_path":     relAutomationPath(matrixBundlePath, root),
			"live_shadow_scorecard_path":    relAutomationPath(scorecardBundlePath, root),
			"rollback_trigger_surface_path": relAutomationPath(rollbackBundlePath, root),
		},
		"latest_evidence_timestamp": scorecardSummary["latest_evidence_timestamp"],
		"freshness":                 freshness,
		"summary": map[string]any{
			"total_evidence_runs":  automationInt(scorecardSummary["total_evidence_runs"], 0),
			"parity_ok_count":      automationInt(scorecardSummary["parity_ok_count"], 0),
			"drift_detected_count": driftDetectedCount,
			"matrix_total":         automationInt(scorecardSummary["matrix_total"], 0),
			"matrix_mismatched":    automationInt(scorecardSummary["matrix_mismatched"], 0),
			"stale_inputs":         staleInputs,
			"fresh_inputs":         automationInt(scorecardSummary["fresh_inputs"], 0),
		},
		"rollback_trigger_surface": map[string]any{
			"status":                     lookupMap(rollbackReport, "summary", "status"),
			"automation_boundary":        lookupMap(rollbackReport, "summary", "automation_boundary"),
			"automated_rollback_trigger": automationBool(lookupMap(rollbackReport, "summary", "automated_rollback_trigger")),
			"distinctions":               lookupMap(rollbackReport, "summary", "distinctions"),
			"issue":                      lookupMap(rollbackReport, "issue"),
			"digest_path":                lookupMap(rollbackReport, "shared_guardrail_summary", "digest_path"),
			"summary_path":               relAutomationPath(filepath.Join(root, "docs/reports/rollback-trigger-surface.json"), root),
		},
		"compare_trace_id":    compareReport["trace_id"],
		"matrix_trace_ids":    matrixTraceIDs,
		"cutover_checkpoints": scorecardReport["cutover_checkpoints"],
		"closeout_commands": []string{
			"cd bigclaw-go && go run ./cmd/bigclawctl automation migration live-shadow-scorecard --pretty",
			"cd bigclaw-go && go run ./cmd/bigclawctl automation migration export-live-shadow-bundle",
			"cd bigclaw-go && go test ./internal/regression -run TestRollbackDocsStayAligned",
			"git push origin <branch> && git log -1 --stat",
		},
	}, nil
}

func loadLiveShadowRecentRuns(bundleRoot string) ([]map[string]any, error) {
	entries, err := os.ReadDir(bundleRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	recentRuns := []map[string]any{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		summaryPath := filepath.Join(bundleRoot, entry.Name(), "summary.json")
		if _, err := os.Stat(summaryPath); err != nil {
			continue
		}
		payload, err := automationShadowMatrixLoadJSON(summaryPath)
		if err != nil {
			return nil, err
		}
		recentRuns = append(recentRuns, payload)
	}
	sort.Slice(recentRuns, func(i, j int) bool {
		return fmt.Sprint(recentRuns[i]["generated_at"]) > fmt.Sprint(recentRuns[j]["generated_at"])
	})
	return recentRuns, nil
}

func buildLiveShadowRollup(recentRuns []map[string]any, limit int, generatedAt time.Time) map[string]any {
	if len(recentRuns) > limit {
		recentRuns = recentRuns[:limit]
	}
	highestSeverity := "none"
	statusCounts := map[string]any{"parity_ok": 0, "attention_needed": 0}
	staleRuns := 0
	driftDetectedRuns := 0
	entries := make([]map[string]any, 0, len(recentRuns))
	for _, item := range recentRuns {
		severity := fmt.Sprint(item["severity"])
		if severityRank(severity) > severityRank(highestSeverity) {
			highestSeverity = severity
		}
		if item["status"] == "parity-ok" {
			statusCounts["parity_ok"] = automationInt(statusCounts["parity_ok"], 0) + 1
		} else {
			statusCounts["attention_needed"] = automationInt(statusCounts["attention_needed"], 0) + 1
		}
		summary, _ := item["summary"].(map[string]any)
		staleInputs := automationInt(summary["stale_inputs"], 0)
		driftDetectedCount := automationInt(summary["drift_detected_count"], 0)
		if staleInputs > 0 {
			staleRuns++
		}
		if driftDetectedCount > 0 {
			driftDetectedRuns++
		}
		entries = append(entries, map[string]any{
			"run_id":                    item["run_id"],
			"generated_at":              item["generated_at"],
			"status":                    item["status"],
			"severity":                  severity,
			"latest_evidence_timestamp": item["latest_evidence_timestamp"],
			"drift_detected_count":      driftDetectedCount,
			"stale_inputs":              staleInputs,
			"bundle_path":               item["bundle_path"],
			"summary_path":              item["summary_path"],
		})
	}
	status := "parity-ok"
	if severityRank(highestSeverity) > 0 {
		status = "attention-needed"
	}
	latestRunID := any(nil)
	if len(recentRuns) > 0 {
		latestRunID = recentRuns[0]["run_id"]
	}
	return map[string]any{
		"generated_at": utcISOTime(generatedAt),
		"status":       status,
		"window_size":  limit,
		"summary": map[string]any{
			"recent_run_count":    len(recentRuns),
			"drift_detected_runs": driftDetectedRuns,
			"stale_runs":          staleRuns,
			"highest_severity":    highestSeverity,
			"status_counts":       statusCounts,
			"latest_run_id":       latestRunID,
		},
		"recent_runs": entries,
	}
}

func renderLiveShadowIndex(latest map[string]any, recentRuns []map[string]any, rollup map[string]any) string {
	lines := []string{
		"# Live Shadow Mirror Index",
		"",
		fmt.Sprintf("- Latest run: `%v`", latest["run_id"]),
		fmt.Sprintf("- Generated at: `%v`", latest["generated_at"]),
		fmt.Sprintf("- Status: `%v`", latest["status"]),
		fmt.Sprintf("- Severity: `%v`", latest["severity"]),
		fmt.Sprintf("- Bundle: `%v`", latest["bundle_path"]),
		fmt.Sprintf("- Summary JSON: `%v`", latest["summary_path"]),
		"",
		"## Latest bundle artifacts",
		"",
	}
	artifacts, _ := latest["artifacts"].(map[string]any)
	for _, item := range []struct{ Key, Label string }{
		{"shadow_compare_report_path", "Shadow compare report"},
		{"shadow_matrix_report_path", "Shadow matrix report"},
		{"live_shadow_scorecard_path", "Parity scorecard"},
		{"rollback_trigger_surface_path", "Rollback trigger surface"},
	} {
		lines = append(lines, fmt.Sprintf("- %s: `%v`", item.Label, artifacts[item.Key]))
	}
	lines = append(lines, "", "## Latest run summary", "")
	lines = append(lines, fmt.Sprintf("- Compare trace: `%v`", latest["compare_trace_id"]))
	matrixTraceCount := 0
	switch traceIDs := latest["matrix_trace_ids"].(type) {
	case []any:
		matrixTraceCount = len(traceIDs)
	case []string:
		matrixTraceCount = len(traceIDs)
	}
	lines = append(lines, fmt.Sprintf("- Matrix trace count: `%d`", matrixTraceCount))
	summary, _ := latest["summary"].(map[string]any)
	for _, item := range []struct{ Key, Label string }{
		{"total_evidence_runs", "Evidence runs"},
		{"parity_ok_count", "Parity-ok entries"},
		{"drift_detected_count", "Drift-detected entries"},
		{"matrix_total", "Matrix total"},
		{"matrix_mismatched", "Matrix mismatched"},
		{"fresh_inputs", "Fresh inputs"},
		{"stale_inputs", "Stale inputs"},
	} {
		lines = append(lines, fmt.Sprintf("- %s: `%v`", item.Label, summary[item.Key]))
	}
	rollback, _ := latest["rollback_trigger_surface"].(map[string]any)
	lines = append(lines, fmt.Sprintf("- Rollback trigger surface status: `%v`", rollback["status"]))
	lines = append(lines, fmt.Sprintf("- Rollback automation boundary: `%v`", rollback["automation_boundary"]))
	lines = append(lines, fmt.Sprintf("- Rollback trigger distinctions: `%v`", rollback["distinctions"]))
	lines = append(lines, "", "## Parity drift rollup", "")
	rollupSummary, _ := rollup["summary"].(map[string]any)
	lines = append(lines, fmt.Sprintf("- Status: `%v`", rollup["status"]))
	lines = append(lines, fmt.Sprintf("- Latest run: `%v`", rollupSummary["latest_run_id"]))
	lines = append(lines, fmt.Sprintf("- Highest severity: `%v`", rollupSummary["highest_severity"]))
	lines = append(lines, fmt.Sprintf("- Drift-detected runs in window: `%v`", rollupSummary["drift_detected_runs"]))
	lines = append(lines, fmt.Sprintf("- Stale runs in window: `%v`", rollupSummary["stale_runs"]))
	lines = append(lines, "", "## Workflow closeout commands", "")
	for _, raw := range latest["closeout_commands"].([]string) {
		lines = append(lines, fmt.Sprintf("- `%s`", raw))
	}
	lines = append(lines, "", "## Recent bundles", "")
	for _, item := range recentRuns {
		lines = append(lines, fmt.Sprintf("- `%v` · `%v` · `%v` · `%v` · `%v`", item["run_id"], item["status"], item["severity"], item["generated_at"], item["bundle_path"]))
	}
	lines = append(lines, "", "## Linked migration docs", "")
	for _, item := range liveShadowDocLinks {
		lines = append(lines, fmt.Sprintf("- `%s` %s", item.Path, item.Description))
	}
	lines = append(lines, "", "## Parallel Follow-up Index", "")
	lines = append(lines, "- `docs/reports/parallel-follow-up-index.md` is the canonical index for the")
	lines = append(lines, "  remaining live-shadow, rollback, and corpus-coverage follow-up digests behind")
	lines = append(lines, "  this run bundle.")
	lines = append(lines, "- For the two primary caveat tracks referenced by this bundle, see")
	lines = append(lines, "  `OPE-266` / `BIG-PAR-092` in")
	lines = append(lines, "  `docs/reports/live-shadow-comparison-follow-up-digest.md` and")
	lines = append(lines, "  `OPE-254` / `BIG-PAR-088` in")
	lines = append(lines, "  `docs/reports/rollback-safeguard-follow-up-digest.md`.")
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

func classifyLiveShadowSeverity(scorecard map[string]any) string {
	summary, _ := scorecard["summary"].(map[string]any)
	if automationInt(summary["stale_inputs"], 0) > 0 {
		return "high"
	}
	if automationInt(summary["drift_detected_count"], 0) > 0 {
		return "medium"
	}
	checkpoints, _ := scorecard["cutover_checkpoints"].([]any)
	for _, raw := range checkpoints {
		item, _ := raw.(map[string]any)
		if !automationBool(item["passed"]) {
			return "low"
		}
	}
	return "none"
}

func severityRank(value string) int {
	switch value {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func relAutomationPath(path string, root string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return path
	}
	return filepath.ToSlash(rel)
}

func copyJSONFile(source string, destination string) (string, error) {
	payload, err := automationShadowMatrixLoadJSON(source)
	if err != nil {
		return "", err
	}
	if err := automationWriteReport(".", destination, payload); err != nil {
		return "", err
	}
	return destination, nil
}

func copyTextFile(source string, destination string) error {
	body, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	return os.WriteFile(destination, body, 0o644)
}

func resolveAutomationGoRoot(value string) string {
	requested := value
	if filepath.IsAbs(requested) {
		return requested
	}
	candidate, _ := filepath.Abs(requested)
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	cwd, _ := os.Getwd()
	if filepath.Base(cwd) == requested {
		if _, err := os.Stat(filepath.Join(cwd, "docs")); err == nil {
			return cwd
		}
	}
	return candidate
}

func deriveLiveShadowRunID(scorecard map[string]any, generatedAt time.Time) string {
	summary, _ := scorecard["summary"].(map[string]any)
	if latest, _ := summary["latest_evidence_timestamp"].(string); trim(latest) != "" {
		if parsed, err := time.Parse(time.RFC3339, strings.ReplaceAll(latest, "Z", "+00:00")); err == nil {
			return parsed.UTC().Format("20060102T150405Z")
		}
	}
	return generatedAt.UTC().Format("20060102T150405Z")
}

func automationSoakLocal(opts automationSoakLocalOptions) (*automationSoakLocalReport, int, error) {
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
		var err error
		cmd, activeBaseURL, stateDir, serviceLog, err = startBigClawd(opts.GoRoot, map[string]string{})
		if err != nil {
			return nil, 0, err
		}
	}
	if err := automationWaitForHealth(client, activeBaseURL, 60, time.Second, sleep); err != nil {
		return nil, 0, err
	}
	start := now()
	tasks := make([]automationTask, 0, opts.Count)
	for i := 0; i < opts.Count; i++ {
		tasks = append(tasks, automationTask{
			ID:                      fmt.Sprintf("soak-%d-%d", i, start.Unix()),
			Title:                   fmt.Sprintf("soak task %d", i),
			RequiredExecutor:        "local",
			Entrypoint:              fmt.Sprintf("echo soak %d", i),
			ExecutionTimeoutSeconds: opts.TimeoutSeconds,
		})
	}
	statuses := make([]map[string]any, len(tasks))
	errCh := make(chan error, 1)
	workCh := make(chan int)
	var wg sync.WaitGroup
	for worker := 0; worker < opts.Workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range workCh {
				if err := automationRequestJSON(client, http.MethodPost, activeBaseURL, "/tasks", tasks[index], nil); err != nil {
					select {
					case errCh <- err:
					default:
					}
					return
				}
				status, err := automationWaitForTask(client, activeBaseURL, tasks[index].ID, time.Duration(opts.TimeoutSeconds)*time.Second, sleep)
				if err != nil {
					select {
					case errCh <- err:
					default:
					}
					return
				}
				statuses[index] = status
			}
		}()
	}
	for index := range tasks {
		select {
		case err := <-errCh:
			close(workCh)
			wg.Wait()
			return nil, 0, err
		default:
			workCh <- index
		}
	}
	close(workCh)
	wg.Wait()
	select {
	case err := <-errCh:
		return nil, 0, err
	default:
	}
	elapsed := now().Sub(start).Seconds()
	succeeded := 0
	sample := make([]map[string]any, 0, 3)
	for _, status := range statuses {
		if status["state"] == "succeeded" {
			succeeded++
		}
		if len(sample) < 3 {
			sample = append(sample, status)
		}
	}
	report := &automationSoakLocalReport{
		BaseURL:               activeBaseURL,
		Count:                 opts.Count,
		Workers:               opts.Workers,
		ElapsedSeconds:        elapsed,
		ThroughputTasksPerSec: float64(opts.Count) / elapsed,
		Succeeded:             succeeded,
		Failed:                opts.Count - succeeded,
		SampleStatus:          sample,
		Autostarted:           cmd != nil,
		StateDir:              stateDir,
		ServiceLog:            serviceLog,
	}
	if err := automationWriteReport(opts.GoRoot, opts.ReportPath, report); err != nil {
		return nil, 0, err
	}
	if succeeded == opts.Count {
		return report, 0, nil
	}
	return report, 1, nil
}

func automationRequestJSON(client *http.Client, method string, baseURL string, path string, payload any, target any) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequest(method, trim(baseURL)+path, body)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("http %d: %s", response.StatusCode, trim(string(body)))
	}
	if target == nil {
		return nil
	}
	return json.NewDecoder(response.Body).Decode(target)
}

func automationWaitForHealth(client *http.Client, baseURL string, attempts int, interval time.Duration, sleep func(time.Duration)) error {
	var lastErr error
	for i := 0; i < attempts; i++ {
		payload := map[string]any{}
		if err := automationRequestJSON(client, http.MethodGet, baseURL, "/healthz", nil, &payload); err == nil && payload["ok"] == true {
			return nil
		} else if err != nil {
			lastErr = err
		}
		sleep(interval)
	}
	if lastErr == nil {
		lastErr = errors.New("health check did not report ok=true")
	}
	return lastErr
}

func automationWaitForTask(client *http.Client, baseURL string, taskID string, timeout time.Duration, sleep func(time.Duration)) (map[string]any, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status := map[string]any{}
		if err := automationRequestJSON(client, http.MethodGet, baseURL, "/tasks/"+taskID, nil, &status); err != nil {
			return nil, err
		}
		if automationIsTerminal(status["state"]) {
			return status, nil
		}
		sleep(250 * time.Millisecond)
	}
	return nil, fmt.Errorf("timeout waiting for %s", taskID)
}

func automationFetchEvents(client *http.Client, baseURL string, taskID string) ([]map[string]any, error) {
	payload := map[string]any{}
	if err := automationRequestJSON(client, http.MethodGet, baseURL, fmt.Sprintf("/events?task_id=%s&limit=100", taskID), nil, &payload); err != nil {
		return nil, err
	}
	items, _ := payload["events"].([]any)
	events := make([]map[string]any, 0, len(items))
	for _, item := range items {
		entry, _ := item.(map[string]any)
		events = append(events, entry)
	}
	return events, nil
}

func automationIsTerminal(state any) bool {
	value, _ := state.(string)
	switch value {
	case "succeeded", "dead_letter", "cancelled", "failed":
		return true
	default:
		return false
	}
}

func automationWriteReport(root string, reportPath string, payload any) error {
	if trim(reportPath) == "" {
		return nil
	}
	target := reportPath
	if !filepath.IsAbs(target) {
		target = filepath.Join(root, reportPath)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(target, append(body, '\n'), 0o644)
}

func automationEventTypes(events []map[string]any) []string {
	types := make([]string, 0, len(events))
	for _, event := range events {
		if eventType, ok := event["type"].(string); ok {
			types = append(types, eventType)
		}
	}
	return types
}

func automationTimelineSeconds(events []map[string]any) float64 {
	if len(events) < 2 {
		return 0
	}
	start, ok := events[0]["timestamp"].(string)
	if !ok || trim(start) == "" {
		return 0
	}
	end, ok := events[len(events)-1]["timestamp"].(string)
	if !ok || trim(end) == "" {
		return 0
	}
	startTS, err := time.Parse(time.RFC3339, start)
	if err != nil {
		return 0
	}
	endTS, err := time.Parse(time.RFC3339, end)
	if err != nil {
		return 0
	}
	duration := endTS.Sub(startTS).Seconds()
	if duration < 0 {
		return 0
	}
	return duration
}

func automationStringSlicesEqual(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func resolveAutomationPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return filepath.Join("..", path)
}

func utcISOTime(moment time.Time) string {
	return moment.UTC().Format(time.RFC3339Nano)
}

func lookupMap(value map[string]any, keys ...string) any {
	current := any(value)
	for _, key := range keys {
		next, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = next[key]
	}
	return current
}

func lookupBool(value map[string]any, keys ...string) bool {
	current := lookupMap(value, keys...)
	return automationBool(current)
}

func roundTo(value float64, places int) float64 {
	pow := 1.0
	for i := 0; i < places; i++ {
		pow *= 10
	}
	if value >= 0 {
		return float64(int(value*pow+0.5)) / pow
	}
	return float64(int(value*pow-0.5)) / pow
}

func asFloat(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	default:
		return 0
	}
}

func absFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func collectStatuses(items []map[string]any) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if status, ok := item["status"].(string); ok {
			out = append(out, status)
		}
	}
	return out
}

func stringOrNil(value string) any {
	if trim(value) == "" {
		return nil
	}
	return value
}

func automationStringSlice(value any) []string {
	items, _ := value.([]any)
	if items == nil {
		if stringsValue, ok := value.([]string); ok {
			return append([]string(nil), stringsValue...)
		}
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if str, ok := item.(string); ok {
			out = append(out, str)
		}
	}
	return out
}

func automationInt(value any, fallback int) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return fallback
	}
}

func automationBool(value any) bool {
	typed, _ := value.(bool)
	return typed
}

func cloneMap(value map[string]any) map[string]any {
	if value == nil {
		return nil
	}
	out := make(map[string]any, len(value))
	for key, item := range value {
		out[key] = item
	}
	return out
}

func startAutomationBigClawd(goRoot string, extraEnv map[string]string) (*exec.Cmd, string, string, string, error) {
	stateDir, err := os.MkdirTemp("", "bigclawd-state-")
	if err != nil {
		return nil, "", "", "", err
	}
	queueBackend := envOrDefault("BIGCLAW_QUEUE_BACKEND", "file")
	baseURL, httpAddr, err := automationReserveLocalBaseURL()
	if err != nil {
		return nil, "", "", "", err
	}
	env := os.Environ()
	env = append(env, "BIGCLAW_HTTP_ADDR="+httpAddr)
	env = append(env, "BIGCLAW_AUDIT_LOG_PATH="+filepath.Join(stateDir, "audit.jsonl"))
	switch queueBackend {
	case "sqlite":
		env = append(env, "BIGCLAW_QUEUE_SQLITE_PATH="+filepath.Join(stateDir, "queue.db"))
	default:
		env = append(env, "BIGCLAW_QUEUE_FILE="+filepath.Join(stateDir, "queue.json"))
	}
	for key, value := range extraEnv {
		env = append(env, key+"="+value)
	}
	logFile, err := os.CreateTemp("", "bigclawd-automation-*.log")
	if err != nil {
		return nil, "", "", "", err
	}
	cmd := exec.Command("go", "run", "./cmd/bigclawd")
	cmd.Dir = goRoot
	cmd.Env = env
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		return nil, "", "", "", err
	}
	return cmd, baseURL, stateDir, logFile.Name(), nil
}

func automationReserveLocalBaseURL() (string, string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", "", err
	}
	defer listener.Close()
	addr := listener.Addr().String()
	return "http://" + addr, addr, nil
}

func printAutomationUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: bigclawctl automation <e2e|benchmark|migration> ...")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "categories:")
	fmt.Fprintln(w, "  e2e         end-to-end automation entrypoints")
	fmt.Fprintln(w, "  benchmark   benchmark and soak automation entrypoints")
	fmt.Fprintln(w, "  migration   migration and shadow comparison entrypoints")
}
