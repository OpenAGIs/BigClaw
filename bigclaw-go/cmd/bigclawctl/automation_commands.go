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

type automationValidationBundleContinuationScorecardOptions struct {
	IndexManifestPath     string
	BundleRootPath        string
	SummaryPath           string
	SharedQueueReportPath string
	OutputPath            string
	Now                   func() time.Time
}

type automationValidationBundleContinuationPolicyGateOptions struct {
	ScorecardPath                 string
	MaxLatestAgeHours             float64
	MinRecentBundles              int
	RequireRepeatedLaneCoverage   bool
	EnforcementMode               string
	LegacyEnforceContinuationGate bool
	OutputPath                    string
	Now                           func() time.Time
}

type automationExportValidationBundleOptions struct {
	GoRoot                     string
	RunID                      string
	BundleDir                  string
	SummaryPath                string
	IndexPath                  string
	ManifestPath               string
	RunLocal                   bool
	RunKubernetes              bool
	RunRay                     bool
	ValidationStatus           int
	RunBroker                  bool
	BrokerBackend              string
	BrokerReportPath           string
	BrokerBootstrapSummaryPath string
	LocalReportPath            string
	LocalStdoutPath            string
	LocalStderrPath            string
	KubernetesReportPath       string
	KubernetesStdoutPath       string
	KubernetesStderrPath       string
	RayReportPath              string
	RayStdoutPath              string
	RayStderrPath              string
	Now                        func() time.Time
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
		_, _ = os.Stdout.WriteString("usage: bigclawctl automation e2e <run-task-smoke|export-validation-bundle|validation-bundle-continuation-scorecard|validation-bundle-continuation-policy-gate> [flags]\n")
		return nil
	}
	switch args[0] {
	case "run-task-smoke":
		return runAutomationRunTaskSmokeCommand(args[1:])
	case "export-validation-bundle":
		return runAutomationExportValidationBundleCommand(args[1:])
	case "validation-bundle-continuation-scorecard":
		return runAutomationValidationBundleContinuationScorecardCommand(args[1:])
	case "validation-bundle-continuation-policy-gate":
		return runAutomationValidationBundleContinuationPolicyGateCommand(args[1:])
	default:
		return fmt.Errorf("unknown automation e2e subcommand: %s", args[0])
	}
}

func runAutomationBenchmark(args []string) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		_, _ = os.Stdout.WriteString("usage: bigclawctl automation benchmark <soak-local> [flags]\n")
		return nil
	}
	switch args[0] {
	case "soak-local":
		return runAutomationSoakLocalCommand(args[1:])
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

func runAutomationValidationBundleContinuationScorecardCommand(args []string) error {
	flags := flag.NewFlagSet("automation e2e validation-bundle-continuation-scorecard", flag.ContinueOnError)
	indexManifest := flags.String("index-manifest", "bigclaw-go/docs/reports/live-validation-index.json", "validation bundle manifest path")
	bundleRoot := flags.String("bundle-root", "bigclaw-go/docs/reports/live-validation-runs", "validation bundle root")
	summaryPath := flags.String("summary-path", "bigclaw-go/docs/reports/live-validation-summary.json", "latest validation summary path")
	sharedQueueReport := flags.String("shared-queue-report", "bigclaw-go/docs/reports/multi-node-shared-queue-report.json", "shared queue proof report path")
	output := flags.String("output", "bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json", "output path")
	asJSON := flags.Bool("json", true, "json")
	_ = flags.Bool("pretty", false, "compatibility alias; JSON output is already formatted")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation e2e validation-bundle-continuation-scorecard [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	report, err := automationValidationBundleContinuationScorecard(automationValidationBundleContinuationScorecardOptions{
		IndexManifestPath:     *indexManifest,
		BundleRootPath:        *bundleRoot,
		SummaryPath:           *summaryPath,
		SharedQueueReportPath: *sharedQueueReport,
		OutputPath:            *output,
	})
	if err != nil {
		return err
	}
	return emit(report, *asJSON, 0)
}

func runAutomationValidationBundleContinuationPolicyGateCommand(args []string) error {
	flags := flag.NewFlagSet("automation e2e validation-bundle-continuation-policy-gate", flag.ContinueOnError)
	scorecardPath := flags.String("scorecard", "bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json", "scorecard input path")
	output := flags.String("output", "bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json", "output path")
	maxLatestAgeHours := flags.Float64("max-latest-age-hours", 72.0, "max age in hours for the latest bundle")
	minRecentBundles := flags.Int("min-recent-bundles", 2, "minimum number of recent bundles")
	requireRepeatedLaneCoverage := flags.Bool("require-repeated-lane-coverage", true, "require repeated recent coverage for every executor lane")
	allowPartialLaneHistory := flags.Bool("allow-partial-lane-history", false, "disable repeated lane coverage enforcement")
	enforcementMode := flags.String("enforcement-mode", "", "enforcement mode (review|hold|fail)")
	enforce := flags.Bool("enforce", false, "legacy alias that maps to fail mode when set")
	asJSON := flags.Bool("json", true, "json")
	_ = flags.Bool("pretty", false, "compatibility alias; JSON output is already formatted")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation e2e validation-bundle-continuation-policy-gate [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	report, exitCode, err := automationValidationBundleContinuationPolicyGate(automationValidationBundleContinuationPolicyGateOptions{
		ScorecardPath:                 *scorecardPath,
		MaxLatestAgeHours:             *maxLatestAgeHours,
		MinRecentBundles:              *minRecentBundles,
		RequireRepeatedLaneCoverage:   *requireRepeatedLaneCoverage && !*allowPartialLaneHistory,
		EnforcementMode:               *enforcementMode,
		LegacyEnforceContinuationGate: *enforce,
		OutputPath:                    *output,
	})
	if report != nil {
		return emit(report, *asJSON, exitCode)
	}
	if err != nil {
		return err
	}
	return nil
}

func runAutomationExportValidationBundleCommand(args []string) error {
	flags := flag.NewFlagSet("automation e2e export-validation-bundle", flag.ContinueOnError)
	goRoot := flags.String("go-root", ".", "bigclaw-go repo root")
	runID := flags.String("run-id", "", "bundle run id")
	bundleDir := flags.String("bundle-dir", "", "bundle directory relative to repo root")
	summaryPath := flags.String("summary-path", "docs/reports/live-validation-summary.json", "summary output path")
	indexPath := flags.String("index-path", "docs/reports/live-validation-index.md", "index markdown path")
	manifestPath := flags.String("manifest-path", "docs/reports/live-validation-index.json", "manifest output path")
	runLocal := flags.Bool("run-local", true, "whether local lane is enabled")
	runKubernetes := flags.Bool("run-kubernetes", true, "whether kubernetes lane is enabled")
	runRay := flags.Bool("run-ray", true, "whether ray lane is enabled")
	validationStatus := flags.Int("validation-status", 0, "overall validation exit status")
	runBroker := flags.Bool("run-broker", false, "whether broker lane is enabled")
	brokerBackend := flags.String("broker-backend", "", "broker backend identifier")
	brokerReportPath := flags.String("broker-report-path", "", "broker report path")
	brokerBootstrapSummaryPath := flags.String("broker-bootstrap-summary-path", "", "broker bootstrap summary path")
	localReportPath := flags.String("local-report-path", "", "local report path")
	localStdoutPath := flags.String("local-stdout-path", "", "local stdout log path")
	localStderrPath := flags.String("local-stderr-path", "", "local stderr log path")
	kubernetesReportPath := flags.String("kubernetes-report-path", "", "kubernetes report path")
	kubernetesStdoutPath := flags.String("kubernetes-stdout-path", "", "kubernetes stdout log path")
	kubernetesStderrPath := flags.String("kubernetes-stderr-path", "", "kubernetes stderr log path")
	rayReportPath := flags.String("ray-report-path", "", "ray report path")
	rayStdoutPath := flags.String("ray-stdout-path", "", "ray stdout log path")
	rayStderrPath := flags.String("ray-stderr-path", "", "ray stderr log path")
	asJSON := flags.Bool("json", true, "json")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation e2e export-validation-bundle [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	if trim(*goRoot) == "" || trim(*runID) == "" || trim(*bundleDir) == "" {
		return errors.New("--go-root, --run-id, and --bundle-dir are required")
	}
	report, exitCode, err := automationExportValidationBundle(automationExportValidationBundleOptions{
		GoRoot:                     *goRoot,
		RunID:                      *runID,
		BundleDir:                  *bundleDir,
		SummaryPath:                *summaryPath,
		IndexPath:                  *indexPath,
		ManifestPath:               *manifestPath,
		RunLocal:                   *runLocal,
		RunKubernetes:              *runKubernetes,
		RunRay:                     *runRay,
		ValidationStatus:           *validationStatus,
		RunBroker:                  *runBroker,
		BrokerBackend:              *brokerBackend,
		BrokerReportPath:           *brokerReportPath,
		BrokerBootstrapSummaryPath: *brokerBootstrapSummaryPath,
		LocalReportPath:            *localReportPath,
		LocalStdoutPath:            *localStdoutPath,
		LocalStderrPath:            *localStderrPath,
		KubernetesReportPath:       *kubernetesReportPath,
		KubernetesStdoutPath:       *kubernetesStdoutPath,
		KubernetesStderrPath:       *kubernetesStderrPath,
		RayReportPath:              *rayReportPath,
		RayStdoutPath:              *rayStdoutPath,
		RayStderrPath:              *rayStderrPath,
	})
	if report != nil {
		return emit(report, *asJSON, exitCode)
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

func automationValidationBundleContinuationScorecard(opts automationValidationBundleContinuationScorecardOptions) (map[string]any, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	manifest, err := automationShadowMatrixLoadJSON(resolveAutomationPath(opts.IndexManifestPath))
	if err != nil {
		return nil, err
	}
	latestSummary, err := automationShadowMatrixLoadJSON(resolveAutomationPath(opts.SummaryPath))
	if err != nil {
		return nil, err
	}
	sharedQueue, err := automationShadowMatrixLoadJSON(resolveAutomationPath(opts.SharedQueueReportPath))
	if err != nil {
		return nil, err
	}

	repoRoot := resolveAutomationPath("..")
	bigclawGoRoot := resolveAutomationPath(".")
	recentRunsMeta, _ := manifest["recent_runs"].([]any)
	recentRuns := make([]map[string]any, 0, len(recentRunsMeta))
	recentRunInputs := make([]any, 0, len(recentRunsMeta))
	for _, raw := range recentRunsMeta {
		item, _ := raw.(map[string]any)
		summaryFile, ok := item["summary_path"].(string)
		if !ok || trim(summaryFile) == "" {
			continue
		}
		resolved := resolveAutomationEvidencePath(repoRoot, bigclawGoRoot, summaryFile)
		report, err := automationShadowMatrixLoadJSON(resolved)
		if err != nil {
			return nil, err
		}
		recentRuns = append(recentRuns, report)
		recentRunInputs = append(recentRunInputs, relAutomationPath(resolved, repoRoot))
	}

	latest, _ := manifest["latest"].(map[string]any)
	bundleRoot := resolveAutomationPath(opts.BundleRootPath)
	bundledSharedQueue, _ := latestSummary["shared_queue_companion"].(map[string]any)
	laneScorecards := []map[string]any{
		automationValidationLaneScorecard(recentRuns, "local"),
		automationValidationLaneScorecard(recentRuns, "kubernetes"),
		automationValidationLaneScorecard(recentRuns, "ray"),
	}

	generatedAt := now().UTC()
	latestGeneratedAt := parseAutomationTime(fmt.Sprint(latest["generated_at"]))
	latestAgeHours := roundTo(generatedAt.Sub(latestGeneratedAt).Seconds()/3600, 2)
	bundleGapMinutes := any(nil)
	if len(recentRuns) > 1 {
		previousGeneratedAt := parseAutomationTime(fmt.Sprint(recentRuns[1]["generated_at"]))
		if !latestGeneratedAt.IsZero() && !previousGeneratedAt.IsZero() {
			bundleGapMinutes = roundTo(latestGeneratedAt.Sub(previousGeneratedAt).Seconds()/60, 2)
		}
	}

	latestLaneStatuses := map[string]any{}
	latestAllSucceeded := true
	for _, lane := range []string{"local", "kubernetes", "ray"} {
		section, _ := latestSummary[lane].(map[string]any)
		status := fmt.Sprint(section["status"])
		latestLaneStatuses[lane] = status
		if status != "succeeded" {
			latestAllSucceeded = false
		}
	}
	recentAllSucceeded := true
	for _, run := range recentRuns {
		if fmt.Sprint(run["status"]) != "succeeded" {
			recentAllSucceeded = false
			break
		}
	}
	repeatedLaneCoverage := true
	enabledRunsByLane := map[string]any{}
	for _, lane := range laneScorecards {
		enabledRunsByLane[fmt.Sprint(lane["lane"])] = lane["enabled_runs"]
		if automationInt(lane["enabled_runs"], 0) < 2 {
			repeatedLaneCoverage = false
		}
	}

	sharedQueueAvailable := automationBool(bundledSharedQueue["available"]) || automationBool(sharedQueue["all_ok"])
	sharedQueueCompanion := map[string]any{
		"available":                 sharedQueueAvailable,
		"report_path":               firstNonEmptyAny(bundledSharedQueue["canonical_report_path"], opts.SharedQueueReportPath),
		"summary_path":              firstNonEmptyAny(bundledSharedQueue["canonical_summary_path"], "docs/reports/shared-queue-companion-summary.json"),
		"bundle_report_path":        stringOrNil(fmt.Sprint(bundledSharedQueue["bundle_report_path"])),
		"bundle_summary_path":       stringOrNil(fmt.Sprint(bundledSharedQueue["bundle_summary_path"])),
		"cross_node_completions":    firstIntAny(bundledSharedQueue["cross_node_completions"], sharedQueue["cross_node_completions"]),
		"duplicate_completed_tasks": firstIntAny(bundledSharedQueue["duplicate_completed_tasks"], len(automationAnySlice(sharedQueue["duplicate_completed_tasks"]))),
		"duplicate_started_tasks":   firstIntAny(bundledSharedQueue["duplicate_started_tasks"], len(automationAnySlice(sharedQueue["duplicate_started_tasks"]))),
		"mode":                      "standalone-proof",
	}
	if len(bundledSharedQueue) > 0 {
		sharedQueueCompanion["mode"] = "bundle-companion-summary"
	}

	continuationChecks := []map[string]any{
		automationLiveShadowCheck(
			"latest_bundle_all_executor_tracks_succeeded",
			latestAllSucceeded,
			fmt.Sprintf("latest lane statuses=%s", automationPythonDictString(latestLaneStatuses)),
		),
		automationLiveShadowCheck(
			"recent_bundle_chain_has_multiple_runs",
			len(recentRuns) >= 2,
			fmt.Sprintf("recent bundle count=%d", len(recentRuns)),
		),
		automationLiveShadowCheck(
			"recent_bundle_chain_has_no_failures",
			recentAllSucceeded,
			fmt.Sprintf("recent bundle statuses=%s", automationPythonListString(automationCollectStatusesFromRuns(recentRuns))),
		),
		automationLiveShadowCheck(
			"all_executor_tracks_have_repeated_recent_coverage",
			repeatedLaneCoverage,
			fmt.Sprintf("enabled_runs_by_lane=%s", automationPythonDictString(enabledRunsByLane)),
		),
		automationLiveShadowCheck(
			"shared_queue_companion_proof_available",
			sharedQueueAvailable,
			fmt.Sprintf("cross_node_completions=%v", sharedQueueCompanion["cross_node_completions"]),
		),
		automationLiveShadowCheck(
			"continuation_surface_is_workflow_triggered",
			true,
			"run_all closeout now refreshes the scorecard and gate automatically, but continuation still depends on explicit workflow execution instead of an always-on service",
		),
	}

	currentCeiling := []string{
		"continuation across future validation bundles remains workflow-triggered",
		"shared-queue coordination proof now ships as adjacent bundle metadata rather than an executor-native lane",
		"recent history is bounded to the exported bundle index and not an always-on service",
	}
	if !repeatedLaneCoverage {
		currentCeiling = append(currentCeiling, "not every executor lane is enabled across every indexed bundle in the current recent window")
	}

	report := map[string]any{
		"generated_at": utcISOTime(generatedAt),
		"ticket":       "BIG-PAR-086-local-prework",
		"title":        "Validation bundle continuation scorecard",
		"status":       "local-continuation-scorecard",
		"evidence_inputs": map[string]any{
			"manifest_path":            opts.IndexManifestPath,
			"latest_summary_path":      opts.SummaryPath,
			"bundle_root":              opts.BundleRootPath,
			"recent_run_summaries":     recentRunInputs,
			"shared_queue_report_path": opts.SharedQueueReportPath,
			"generator_script":         "go run ./cmd/bigclawctl automation e2e validation-bundle-continuation-scorecard",
		},
		"summary": map[string]any{
			"recent_bundle_count":                               len(recentRuns),
			"latest_run_id":                                     latest["run_id"],
			"latest_status":                                     latest["status"],
			"latest_bundle_age_hours":                           latestAgeHours,
			"latest_all_executor_tracks_succeeded":              latestAllSucceeded,
			"recent_bundle_chain_has_no_failures":               recentAllSucceeded,
			"all_executor_tracks_have_repeated_recent_coverage": repeatedLaneCoverage,
			"bundle_gap_minutes":                                bundleGapMinutes,
			"bundle_root_exists":                                automationPathExists(bundleRoot),
		},
		"executor_lanes":         laneScorecards,
		"shared_queue_companion": sharedQueueCompanion,
		"continuation_checks":    continuationChecks,
		"current_ceiling":        currentCeiling,
		"next_runtime_hooks": []string{
			"set BIGCLAW_E2E_CONTINUATION_GATE_MODE=hold or fail in workflow closeout when continuation holds should block or fail the run",
			"decide whether shared-queue coordination should stay as adjacent bundle metadata or gain its own executor-native validation lane",
			"extend the automatic continuation refresh beyond run_all.sh into broader workflow orchestrators",
			"extend the scorecard beyond the latest recent_runs window when more longitudinal evidence exists",
		},
	}
	if err := automationWriteReport(".", opts.OutputPath, report); err != nil {
		return nil, err
	}
	return report, nil
}

func automationValidationBundleContinuationPolicyGate(opts automationValidationBundleContinuationPolicyGateOptions) (map[string]any, int, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	scorecard, err := automationShadowMatrixLoadJSON(resolveAutomationPath(opts.ScorecardPath))
	if err != nil {
		return nil, 0, err
	}
	summary, _ := scorecard["summary"].(map[string]any)
	sharedQueue, _ := scorecard["shared_queue_companion"].(map[string]any)
	normalizedMode, err := normalizeAutomationContinuationEnforcementMode(opts.EnforcementMode, opts.LegacyEnforceContinuationGate)
	if err != nil {
		return nil, 0, err
	}

	checks := []map[string]any{
		automationLiveShadowCheck(
			"latest_bundle_age_within_threshold",
			asFloat(summary["latest_bundle_age_hours"]) <= opts.MaxLatestAgeHours,
			fmt.Sprintf("latest_bundle_age_hours=%v threshold=%v", summary["latest_bundle_age_hours"], opts.MaxLatestAgeHours),
		),
		automationLiveShadowCheck(
			"recent_bundle_count_meets_floor",
			automationInt(summary["recent_bundle_count"], 0) >= opts.MinRecentBundles,
			fmt.Sprintf("recent_bundle_count=%v floor=%d", summary["recent_bundle_count"], opts.MinRecentBundles),
		),
		automationLiveShadowCheck(
			"latest_bundle_all_executor_tracks_succeeded",
			automationBool(summary["latest_all_executor_tracks_succeeded"]),
			fmt.Sprintf("latest_all_executor_tracks_succeeded=%s", automationPythonBoolString(automationBool(summary["latest_all_executor_tracks_succeeded"]))),
		),
		automationLiveShadowCheck(
			"recent_bundle_chain_has_no_failures",
			automationBool(summary["recent_bundle_chain_has_no_failures"]),
			fmt.Sprintf("recent_bundle_chain_has_no_failures=%s", automationPythonBoolString(automationBool(summary["recent_bundle_chain_has_no_failures"]))),
		),
		automationLiveShadowCheck(
			"shared_queue_companion_available",
			automationBool(sharedQueue["available"]),
			fmt.Sprintf("cross_node_completions=%v", sharedQueue["cross_node_completions"]),
		),
		automationLiveShadowCheck(
			"repeated_lane_coverage_meets_policy",
			!opts.RequireRepeatedLaneCoverage || automationBool(summary["all_executor_tracks_have_repeated_recent_coverage"]),
			fmt.Sprintf("require_repeated_lane_coverage=%s actual=%s", automationPythonBoolString(opts.RequireRepeatedLaneCoverage), automationPythonBoolString(automationBool(summary["all_executor_tracks_have_repeated_recent_coverage"]))),
		),
	}

	failingChecks := []string{}
	passingCheckCount := 0
	for _, check := range checks {
		if automationBool(check["passed"]) {
			passingCheckCount++
			continue
		}
		failingChecks = append(failingChecks, fmt.Sprint(check["name"]))
	}
	recommendation := "go"
	if len(failingChecks) > 0 {
		recommendation = "hold"
	}
	enforcement := buildAutomationContinuationEnforcementSummary(recommendation, normalizedMode)
	nextActions := []string{}
	for _, failing := range failingChecks {
		switch failing {
		case "latest_bundle_age_within_threshold":
			nextActions = append(nextActions, "rerun `cd bigclaw-go && ./scripts/e2e/run_all.sh` to refresh the latest validation bundle")
		case "recent_bundle_count_meets_floor":
			nextActions = append(nextActions, "export additional validation bundles so the continuation window spans multiple indexed runs")
		case "shared_queue_companion_available":
			nextActions = append(nextActions, "rerun `cd bigclaw-go && python3 scripts/e2e/multi_node_shared_queue.py --report-path docs/reports/multi-node-shared-queue-report.json`")
		case "repeated_lane_coverage_meets_policy":
			nextActions = append(nextActions, "refresh another full validation bundle with `ray` enabled so each executor lane has repeated indexed coverage")
		}
	}
	if len(nextActions) == 0 {
		nextActions = append(nextActions, "set BIGCLAW_E2E_CONTINUATION_GATE_MODE=fail when workflow closeout should stop on continuation regressions")
	}
	report := map[string]any{
		"generated_at":   utcISOTime(now().UTC()),
		"ticket":         "OPE-262",
		"title":          "Validation workflow continuation gate",
		"status":         map[bool]string{true: "policy-go", false: "policy-hold"}[recommendation == "go"],
		"recommendation": recommendation,
		"evidence_inputs": map[string]any{
			"scorecard_path":   opts.ScorecardPath,
			"generator_script": "go run ./cmd/bigclawctl automation e2e validation-bundle-continuation-policy-gate",
		},
		"policy_inputs": map[string]any{
			"max_latest_age_hours":           opts.MaxLatestAgeHours,
			"min_recent_bundles":             opts.MinRecentBundles,
			"require_repeated_lane_coverage": opts.RequireRepeatedLaneCoverage,
		},
		"enforcement": enforcement,
		"summary": map[string]any{
			"latest_run_id":                                     summary["latest_run_id"],
			"latest_bundle_age_hours":                           summary["latest_bundle_age_hours"],
			"recent_bundle_count":                               summary["recent_bundle_count"],
			"latest_all_executor_tracks_succeeded":              summary["latest_all_executor_tracks_succeeded"],
			"recent_bundle_chain_has_no_failures":               summary["recent_bundle_chain_has_no_failures"],
			"all_executor_tracks_have_repeated_recent_coverage": summary["all_executor_tracks_have_repeated_recent_coverage"],
			"recommendation":                                    recommendation,
			"enforcement_mode":                                  enforcement["mode"],
			"workflow_outcome":                                  enforcement["outcome"],
			"workflow_exit_code":                                enforcement["exit_code"],
			"passing_check_count":                               passingCheckCount,
			"failing_check_count":                               len(failingChecks),
		},
		"policy_checks":  checks,
		"failing_checks": failingChecks,
		"reviewer_path": map[string]any{
			"index_path":  "docs/reports/live-validation-index.md",
			"digest_path": "docs/reports/validation-bundle-continuation-digest.md",
			"digest_issue": map[string]any{
				"id":   "OPE-271",
				"slug": "BIG-PAR-082",
			},
		},
		"shared_queue_companion": sharedQueue,
		"next_actions":           nextActions,
	}
	if err := automationWriteReport(".", opts.OutputPath, report); err != nil {
		return nil, 0, err
	}
	return report, automationInt(enforcement["exit_code"], 0), nil
}

var automationLatestValidationReports = map[string]string{
	"local":      "docs/reports/sqlite-smoke-report.json",
	"kubernetes": "docs/reports/kubernetes-live-smoke-report.json",
	"ray":        "docs/reports/ray-live-smoke-report.json",
}

const (
	automationBrokerSummaryPath          = "docs/reports/broker-validation-summary.json"
	automationBrokerBootstrapSummaryPath = "docs/reports/broker-bootstrap-review-summary.json"
	automationBrokerValidationPackPath   = "docs/reports/broker-failover-fault-injection-validation-pack.md"
	automationSharedQueueReportPath      = "docs/reports/multi-node-shared-queue-report.json"
	automationSharedQueueSummaryPath     = "docs/reports/shared-queue-companion-summary.json"
)

var automationContinuationArtifacts = []struct {
	Path        string
	Description string
}{
	{
		Path:        "docs/reports/validation-bundle-continuation-scorecard.json",
		Description: "summarizes the rolling readiness view across recent bundled local, Kubernetes, and Ray runs plus the shared-queue companion proof.",
	},
	{
		Path:        "docs/reports/validation-bundle-continuation-policy-gate.json",
		Description: "records the current policy decision for bundle freshness, repeated lane coverage, and shared-queue companion availability.",
	},
}

var automationFollowupDigests = []struct {
	Path        string
	Description string
}{
	{
		Path:        "docs/reports/validation-bundle-continuation-digest.md",
		Description: "Validation bundle continuation caveats are consolidated here.",
	},
}

var automationLaneAliases = map[string]string{
	"local":      "local",
	"kubernetes": "k8s",
	"ray":        "ray",
}

var automationFailureEventTypes = map[string]struct{}{
	"task.cancelled":   {},
	"task.dead_letter": {},
	"task.failed":      {},
	"task.retried":     {},
}

func automationExportValidationBundle(opts automationExportValidationBundleOptions) (map[string]any, int, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	root := resolveAutomationGoRoot(opts.GoRoot)
	bundleDir := filepath.Join(root, opts.BundleDir)
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return nil, 0, err
	}

	summary := map[string]any{
		"run_id":       opts.RunID,
		"generated_at": now().UTC().Format(time.RFC3339Nano),
		"status":       map[bool]string{true: "succeeded", false: "failed"}[opts.ValidationStatus == 0],
		"bundle_path":  relAutomationPath(bundleDir, root),
		"closeout_commands": []string{
			"cd bigclaw-go && ./scripts/e2e/run_all.sh",
			"cd bigclaw-go && go test ./...",
			"git push origin <branch> && git log -1 --stat",
		},
	}
	var err error
	summary["local"], err = automationBuildValidationComponentSection(root, bundleDir, "local", opts.RunLocal, opts.LocalReportPath, opts.LocalStdoutPath, opts.LocalStderrPath)
	if err != nil {
		return nil, 0, err
	}
	summary["kubernetes"], err = automationBuildValidationComponentSection(root, bundleDir, "kubernetes", opts.RunKubernetes, opts.KubernetesReportPath, opts.KubernetesStdoutPath, opts.KubernetesStderrPath)
	if err != nil {
		return nil, 0, err
	}
	summary["ray"], err = automationBuildValidationComponentSection(root, bundleDir, "ray", opts.RunRay, opts.RayReportPath, opts.RayStdoutPath, opts.RayStderrPath)
	if err != nil {
		return nil, 0, err
	}
	summary["broker"], err = automationBuildValidationBrokerSection(root, bundleDir, opts)
	if err != nil {
		return nil, 0, err
	}
	summary["shared_queue_companion"], err = automationBuildSharedQueueCompanion(root, bundleDir)
	if err != nil {
		return nil, 0, err
	}
	summary["validation_matrix"] = automationBuildValidationMatrix(summary)

	continuationGate := automationBuildContinuationGateSummary(root)
	if continuationGate != nil {
		summary["continuation_gate"] = continuationGate
	}

	bundleSummaryPath := filepath.Join(bundleDir, "summary.json")
	summary["summary_path"] = relAutomationPath(bundleSummaryPath, root)
	if err := automationWriteReport(".", bundleSummaryPath, summary); err != nil {
		return nil, 0, err
	}
	if err := automationWriteReport(".", filepath.Join(root, opts.SummaryPath), summary); err != nil {
		return nil, 0, err
	}

	recentRuns, err := automationBuildRecentValidationRuns(filepath.Dir(bundleDir), root, 8)
	if err != nil {
		return nil, 0, err
	}
	manifest := map[string]any{
		"latest":      summary,
		"recent_runs": recentRuns,
	}
	if continuationGate != nil {
		manifest["continuation_gate"] = continuationGate
	}
	if err := automationWriteReport(".", filepath.Join(root, opts.ManifestPath), manifest); err != nil {
		return nil, 0, err
	}

	indexText := automationRenderValidationIndex(
		summary,
		recentRuns,
		continuationGate,
		automationBuildAvailableArtifacts(root, automationContinuationArtifacts),
		automationBuildAvailableArtifacts(root, automationFollowupDigests),
	)
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, opts.IndexPath)), 0o755); err != nil {
		return nil, 0, err
	}
	if err := os.WriteFile(filepath.Join(root, opts.IndexPath), []byte(indexText), 0o644); err != nil {
		return nil, 0, err
	}
	if err := os.WriteFile(filepath.Join(bundleDir, "README.md"), []byte(indexText), 0o644); err != nil {
		return nil, 0, err
	}
	return summary, opts.ValidationStatus, nil
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
			"digest_path":                rollbackReport["digest_path"],
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
	matrixTraceIDs, _ := latest["matrix_trace_ids"].([]any)
	lines = append(lines, fmt.Sprintf("- Matrix trace count: `%d`", len(matrixTraceIDs)))
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
	lines = append(lines, "  remaining live-shadow, rollback, and corpus-coverage follow-up digests.")
	lines = append(lines, "- Use `docs/reports/parallel-validation-matrix.md` first when a shadow review")
	lines = append(lines, "  needs the checked-in local/Kubernetes/Ray validation entrypoint alongside the")
	lines = append(lines, "  shadow evidence bundle.")
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

func resolveAutomationEvidencePath(repoRoot string, bigclawGoRoot string, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	searchRoots := []string{repoRoot}
	if !strings.HasPrefix(filepath.ToSlash(path), "bigclaw-go/") {
		searchRoots = append(searchRoots, bigclawGoRoot)
	}
	for _, root := range searchRoots {
		resolved := filepath.Join(root, path)
		if _, err := os.Stat(resolved); err == nil {
			return resolved
		}
	}
	return filepath.Join(searchRoots[0], path)
}

func automationReadJSONIfExists(path string) (map[string]any, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if info.Size() == 0 {
		return nil, nil
	}
	return automationShadowMatrixLoadJSON(path)
}

func automationCopyTextArtifact(source string, destination string) (string, error) {
	if trim(source) == "" {
		return "", nil
	}
	if _, err := os.Stat(source); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	if filepath.Clean(source) == filepath.Clean(destination) {
		return destination, nil
	}
	if err := copyTextFile(source, destination); err != nil {
		return "", err
	}
	return destination, nil
}

func automationCopyJSONArtifact(source string, destination string) (string, map[string]any, error) {
	if trim(source) == "" {
		return "", nil, nil
	}
	payload, err := automationReadJSONIfExists(source)
	if err != nil || payload == nil {
		return "", payload, err
	}
	if filepath.Clean(source) == filepath.Clean(destination) {
		return destination, payload, nil
	}
	if err := automationWriteReport(".", destination, payload); err != nil {
		return "", nil, err
	}
	return destination, payload, nil
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

func automationCollectReportEvents(report map[string]any) []map[string]any {
	events := []map[string]any{}
	status, _ := report["status"].(map[string]any)
	if statusEvents, ok := status["events"].([]any); ok {
		for _, raw := range statusEvents {
			if event, ok := raw.(map[string]any); ok {
				events = append(events, event)
			}
		}
	}
	if latestEvent, ok := status["latest_event"].(map[string]any); ok {
		latestID := trim(fmt.Sprint(latestEvent["id"]))
		found := false
		for _, event := range events {
			if trim(fmt.Sprint(event["id"])) == latestID && latestID != "" {
				found = true
				break
			}
		}
		if !found {
			events = append(events, latestEvent)
		}
	}
	if reportEvents, ok := report["events"].([]any); ok {
		for _, raw := range reportEvents {
			event, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			eventID := trim(fmt.Sprint(event["id"]))
			duplicate := false
			for _, existing := range events {
				if eventID != "" && trim(fmt.Sprint(existing["id"])) == eventID {
					duplicate = true
					break
				}
			}
			if !duplicate {
				events = append(events, event)
			}
		}
	}
	return events
}

func automationLatestReportEvent(report map[string]any) map[string]any {
	events := automationCollectReportEvents(report)
	if len(events) == 0 {
		return nil
	}
	return events[len(events)-1]
}

func automationEventPayloadText(event map[string]any, key string) string {
	payload, _ := event["payload"].(map[string]any)
	return firstStringAny(payload[key])
}

func automationFindRoutingReason(report map[string]any) string {
	events := automationCollectReportEvents(report)
	for i := len(events) - 1; i >= 0; i-- {
		if fmt.Sprint(events[i]["type"]) == "scheduler.routed" {
			return trim(automationEventPayloadText(events[i], "reason"))
		}
	}
	return ""
}

func automationComponentStatus(report map[string]any) string {
	if report == nil {
		return "missing_report"
	}
	switch status := report["status"].(type) {
	case map[string]any:
		return firstStringAny(status["state"], "unknown")
	case string:
		return status
	}
	if automationBool(report["all_ok"]) {
		return "succeeded"
	}
	if value, ok := report["all_ok"].(bool); ok && !value {
		return "failed"
	}
	return "unknown"
}

func automationBuildFailureRootCause(section map[string]any, report map[string]any) map[string]any {
	events := automationCollectReportEvents(report)
	latestEvent := automationLatestReportEvent(report)
	latestStatus := ""
	if status, ok := report["status"].(map[string]any); ok {
		latestStatus = trim(fmt.Sprint(status["state"]))
	}
	if latestStatus == "" {
		if task, ok := report["task"].(map[string]any); ok {
			latestStatus = trim(fmt.Sprint(task["state"]))
		}
	}
	if latestStatus == "" {
		latestStatus = automationComponentStatus(report)
	}
	var causeEvent map[string]any
	for i := len(events) - 1; i >= 0; i-- {
		if _, ok := automationFailureEventTypes[trim(fmt.Sprint(events[i]["type"]))]; ok {
			causeEvent = events[i]
			break
		}
	}
	if causeEvent == nil && latestStatus != "" && latestStatus != "succeeded" {
		causeEvent = latestEvent
	}
	location := trim(firstStringAny(section["stderr_path"], section["service_log_path"], section["audit_log_path"], section["bundle_report_path"]))
	if causeEvent == nil {
		return map[string]any{
			"status":     "not_triggered",
			"event_type": trim(fmt.Sprint(lookupMap(latestEvent, "type"))),
			"message":    "",
			"location":   location,
			"event_id":   "",
			"timestamp":  "",
		}
	}
	return map[string]any{
		"status":     "captured",
		"event_type": trim(fmt.Sprint(causeEvent["type"])),
		"message": trim(firstStringAny(
			automationEventPayloadText(causeEvent, "message"),
			automationEventPayloadText(causeEvent, "reason"),
			report["error"],
			report["failure_reason"],
		)),
		"location":  location,
		"event_id":  trim(fmt.Sprint(causeEvent["id"])),
		"timestamp": trim(fmt.Sprint(causeEvent["timestamp"])),
	}
}

func automationBuildValidationMatrixEntry(name string, section map[string]any, report map[string]any) map[string]any {
	var task map[string]any
	if report != nil {
		task, _ = report["task"].(map[string]any)
	}
	taskID := trim(firstStringAny(lookupMap(task, "id"), section["task_id"]))
	executor := trim(fmt.Sprint(lookupMap(task, "required_executor")))
	if executor == "" {
		executor = name
	}
	rootCause, _ := section["failure_root_cause"].(map[string]any)
	return map[string]any{
		"lane":                  automationLaneAliases[name],
		"executor":              executor,
		"enabled":               automationBool(section["enabled"]),
		"status":                firstNonEmptyAny(section["status"], "unknown"),
		"task_id":               taskID,
		"canonical_report_path": firstNonEmptyAny(section["canonical_report_path"], ""),
		"bundle_report_path":    firstNonEmptyAny(section["bundle_report_path"], ""),
		"latest_event_type":     firstNonEmptyAny(section["latest_event_type"], ""),
		"routing_reason":        firstNonEmptyAny(section["routing_reason"], ""),
		"root_cause_event_type": firstNonEmptyAny(rootCause["event_type"], ""),
		"root_cause_location":   firstNonEmptyAny(rootCause["location"], ""),
		"root_cause_message":    firstNonEmptyAny(rootCause["message"], ""),
	}
}

func automationBuildValidationMatrix(summary map[string]any) []map[string]any {
	rows := []map[string]any{}
	for _, name := range []string{"local", "kubernetes", "ray"} {
		section, _ := summary[name].(map[string]any)
		if row, ok := section["validation_matrix"].(map[string]any); ok {
			rows = append(rows, row)
		}
	}
	return rows
}

func automationBuildSharedQueueCompanion(root string, bundleDir string) (map[string]any, error) {
	canonicalReportPath := filepath.Join(root, automationSharedQueueReportPath)
	canonicalSummaryPath := filepath.Join(root, automationSharedQueueSummaryPath)
	bundleReportPath := filepath.Join(bundleDir, "multi-node-shared-queue-report.json")
	bundleSummaryPath := filepath.Join(bundleDir, "shared-queue-companion-summary.json")
	report, err := automationReadJSONIfExists(canonicalReportPath)
	if err != nil {
		return nil, err
	}
	summary := map[string]any{
		"available":              report != nil,
		"canonical_report_path":  automationSharedQueueReportPath,
		"canonical_summary_path": automationSharedQueueSummaryPath,
		"bundle_report_path":     relAutomationPath(bundleReportPath, root),
		"bundle_summary_path":    relAutomationPath(bundleSummaryPath, root),
	}
	if report == nil {
		summary["status"] = "missing_report"
		return summary, nil
	}
	if copied, _, err := automationCopyJSONArtifact(canonicalReportPath, bundleReportPath); err != nil {
		return nil, err
	} else if copied != "" {
		summary["bundle_report_path"] = relAutomationPath(copied, root)
	}
	summary["status"] = map[bool]string{true: "succeeded", false: "failed"}[automationBool(report["all_ok"])]
	summary["generated_at"] = report["generated_at"]
	summary["count"] = report["count"]
	summary["cross_node_completions"] = report["cross_node_completions"]
	summary["duplicate_started_tasks"] = len(automationAnySlice(report["duplicate_started_tasks"]))
	summary["duplicate_completed_tasks"] = len(automationAnySlice(report["duplicate_completed_tasks"]))
	summary["missing_completed_tasks"] = len(automationAnySlice(report["missing_completed_tasks"]))
	summary["submitted_by_node"] = firstMapAny(report["submitted_by_node"])
	summary["completed_by_node"] = firstMapAny(report["completed_by_node"])
	summary["nodes"] = automationExtractNodeNames(automationAnySlice(report["nodes"]))
	if err := automationWriteReport(".", bundleSummaryPath, summary); err != nil {
		return nil, err
	}
	if err := automationWriteReport(".", canonicalSummaryPath, summary); err != nil {
		return nil, err
	}
	return summary, nil
}

func automationBuildValidationComponentSection(root string, bundleDir string, name string, enabled bool, reportPath string, stdoutPath string, stderrPath string) (map[string]any, error) {
	latestReportPath := filepath.Join(root, automationLatestValidationReports[name])
	resolvedReportPath := filepath.Join(root, reportPath)
	section := map[string]any{
		"enabled":               enabled,
		"bundle_report_path":    relAutomationPath(resolvedReportPath, root),
		"canonical_report_path": automationLatestValidationReports[name],
	}
	if !enabled {
		section["status"] = "skipped"
		return section, nil
	}
	report, err := automationReadJSONIfExists(resolvedReportPath)
	if err != nil {
		return nil, err
	}
	section["report"] = report
	section["status"] = automationComponentStatus(report)
	if copied, _, err := automationCopyJSONArtifact(resolvedReportPath, latestReportPath); err != nil {
		return nil, err
	} else if copied != "" {
		section["canonical_report_path"] = relAutomationPath(copied, root)
	}
	if copied, err := automationCopyTextArtifact(stdoutPath, filepath.Join(bundleDir, name+".stdout.log")); err != nil {
		return nil, err
	} else if copied != "" {
		section["stdout_path"] = relAutomationPath(copied, root)
	}
	if copied, err := automationCopyTextArtifact(stderrPath, filepath.Join(bundleDir, name+".stderr.log")); err != nil {
		return nil, err
	} else if copied != "" {
		section["stderr_path"] = relAutomationPath(copied, root)
	}
	if report == nil {
		section["failure_root_cause"] = map[string]any{
			"status":     "missing_report",
			"event_type": "",
			"message":    "",
			"location":   section["bundle_report_path"],
			"event_id":   "",
			"timestamp":  "",
		}
		section["validation_matrix"] = automationBuildValidationMatrixEntry(name, section, nil)
		return section, nil
	}
	if task, ok := report["task"].(map[string]any); ok && trim(fmt.Sprint(task["id"])) != "" {
		section["task_id"] = task["id"]
	}
	if baseURL := trim(fmt.Sprint(report["base_url"])); baseURL != "" {
		section["base_url"] = baseURL
	}
	stateDir := trim(fmt.Sprint(report["state_dir"]))
	if stateDir != "" {
		section["state_dir"] = stateDir
		if copied, err := automationCopyTextArtifact(filepath.Join(stateDir, "audit.jsonl"), filepath.Join(bundleDir, name+".audit.jsonl")); err != nil {
			return nil, err
		} else if copied != "" {
			section["audit_log_path"] = relAutomationPath(copied, root)
		}
	}
	if serviceLog := trim(fmt.Sprint(report["service_log"])); serviceLog != "" {
		if copied, err := automationCopyTextArtifact(serviceLog, filepath.Join(bundleDir, name+".service.log")); err != nil {
			return nil, err
		} else if copied != "" {
			section["service_log_path"] = relAutomationPath(copied, root)
		}
	}
	if latestEvent := automationLatestReportEvent(report); latestEvent != nil {
		section["latest_event_type"] = trim(fmt.Sprint(latestEvent["type"]))
		section["latest_event_timestamp"] = trim(fmt.Sprint(latestEvent["timestamp"]))
		if payload, ok := latestEvent["payload"].(map[string]any); ok {
			if artifacts, ok := payload["artifacts"].([]any); ok {
				paths := []string{}
				for _, raw := range artifacts {
					if path, ok := raw.(string); ok {
						paths = append(paths, path)
					}
				}
				section["artifact_paths"] = paths
			}
		}
	}
	if routingReason := automationFindRoutingReason(report); routingReason != "" {
		section["routing_reason"] = routingReason
	}
	section["failure_root_cause"] = automationBuildFailureRootCause(section, report)
	section["validation_matrix"] = automationBuildValidationMatrixEntry(name, section, report)
	return section, nil
}

func automationBuildValidationBrokerSection(root string, bundleDir string, opts automationExportValidationBundleOptions) (map[string]any, error) {
	bundleSummaryPath := filepath.Join(bundleDir, "broker-validation-summary.json")
	bundleBootstrapSummaryPath := filepath.Join(bundleDir, "broker-bootstrap-review-summary.json")
	section := map[string]any{
		"enabled":                          opts.RunBroker,
		"backend":                          stringOrNil(trim(opts.BrokerBackend)),
		"bundle_summary_path":              relAutomationPath(bundleSummaryPath, root),
		"canonical_summary_path":           automationBrokerSummaryPath,
		"bundle_bootstrap_summary_path":    relAutomationPath(bundleBootstrapSummaryPath, root),
		"canonical_bootstrap_summary_path": automationBrokerBootstrapSummaryPath,
		"validation_pack_path":             automationBrokerValidationPackPath,
	}
	configurationState := "not_configured"
	if opts.RunBroker && trim(opts.BrokerBackend) != "" {
		configurationState = "configured"
	}
	section["configuration_state"] = configurationState
	if bootstrapPath := trim(opts.BrokerBootstrapSummaryPath); bootstrapPath != "" {
		source := filepath.Join(root, bootstrapPath)
		bootstrapSummary, err := automationReadJSONIfExists(source)
		if err != nil {
			return nil, err
		}
		if bootstrapSummary != nil {
			if copied, _, err := automationCopyJSONArtifact(source, bundleBootstrapSummaryPath); err != nil {
				return nil, err
			} else if copied != "" {
				section["bundle_bootstrap_summary_path"] = relAutomationPath(copied, root)
			}
			if copied, _, err := automationCopyJSONArtifact(source, filepath.Join(root, automationBrokerBootstrapSummaryPath)); err != nil {
				return nil, err
			} else if copied != "" {
				section["canonical_bootstrap_summary_path"] = relAutomationPath(copied, root)
			}
			section["bootstrap_summary"] = bootstrapSummary
			section["bootstrap_ready"] = automationBool(bootstrapSummary["ready"])
			section["runtime_posture"] = bootstrapSummary["runtime_posture"]
			section["live_adapter_implemented"] = automationBool(bootstrapSummary["live_adapter_implemented"])
			section["proof_boundary"] = bootstrapSummary["proof_boundary"]
			if errorsList, ok := bootstrapSummary["validation_errors"].([]any); ok {
				section["validation_errors"] = errorsList
			}
			if completeness, ok := bootstrapSummary["config_completeness"].(map[string]any); ok {
				section["config_completeness"] = completeness
			}
		}
	}
	if !opts.RunBroker || trim(opts.BrokerBackend) == "" {
		section["status"] = "skipped"
		section["reason"] = "not_configured"
		if err := automationWriteReport(".", bundleSummaryPath, section); err != nil {
			return nil, err
		}
		if err := automationWriteReport(".", filepath.Join(root, automationBrokerSummaryPath), section); err != nil {
			return nil, err
		}
		return section, nil
	}
	if trim(opts.BrokerReportPath) == "" {
		section["status"] = "skipped"
		section["reason"] = "missing_report_path"
		if err := automationWriteReport(".", bundleSummaryPath, section); err != nil {
			return nil, err
		}
		if err := automationWriteReport(".", filepath.Join(root, automationBrokerSummaryPath), section); err != nil {
			return nil, err
		}
		return section, nil
	}
	reportPath := filepath.Join(root, opts.BrokerReportPath)
	report, err := automationReadJSONIfExists(reportPath)
	if err != nil {
		return nil, err
	}
	section["canonical_report_path"] = relAutomationPath(reportPath, root)
	section["bundle_report_path"] = relAutomationPath(filepath.Join(bundleDir, filepath.Base(reportPath)), root)
	if report == nil {
		section["status"] = "skipped"
		section["reason"] = "not_configured"
		if err := automationWriteReport(".", bundleSummaryPath, section); err != nil {
			return nil, err
		}
		if err := automationWriteReport(".", filepath.Join(root, automationBrokerSummaryPath), section); err != nil {
			return nil, err
		}
		return section, nil
	}
	if copied, _, err := automationCopyJSONArtifact(reportPath, filepath.Join(bundleDir, filepath.Base(reportPath))); err != nil {
		return nil, err
	} else if copied != "" {
		section["bundle_report_path"] = relAutomationPath(copied, root)
	}
	section["report"] = report
	section["status"] = automationComponentStatus(report)
	if err := automationWriteReport(".", bundleSummaryPath, section); err != nil {
		return nil, err
	}
	if err := automationWriteReport(".", filepath.Join(root, automationBrokerSummaryPath), section); err != nil {
		return nil, err
	}
	return section, nil
}

func automationBuildRecentValidationRuns(bundleRoot string, root string, limit int) ([]map[string]any, error) {
	entries, err := os.ReadDir(bundleRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	type item struct {
		generatedAt string
		summary     map[string]any
	}
	runs := []item{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		summaryPath := filepath.Join(bundleRoot, entry.Name(), "summary.json")
		summary, err := automationReadJSONIfExists(summaryPath)
		if err != nil {
			return nil, err
		}
		if summary != nil {
			runs = append(runs, item{generatedAt: fmt.Sprint(summary["generated_at"]), summary: summary})
		}
	}
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].generatedAt > runs[j].generatedAt
	})
	if len(runs) > limit {
		runs = runs[:limit]
	}
	out := []map[string]any{}
	for _, run := range runs {
		out = append(out, map[string]any{
			"run_id":       run.summary["run_id"],
			"generated_at": run.summary["generated_at"],
			"status":       run.summary["status"],
			"bundle_path":  run.summary["bundle_path"],
			"summary_path": run.summary["summary_path"],
		})
	}
	return out, nil
}

func automationBuildContinuationGateSummary(root string) map[string]any {
	gatePath := filepath.Join(root, "docs/reports/validation-bundle-continuation-policy-gate.json")
	gate, err := automationReadJSONIfExists(gatePath)
	if err != nil || gate == nil {
		return nil
	}
	enforcement, _ := gate["enforcement"].(map[string]any)
	summary, _ := gate["summary"].(map[string]any)
	reviewerPath, _ := gate["reviewer_path"].(map[string]any)
	nextActions, _ := gate["next_actions"].([]any)
	return map[string]any{
		"path":           relAutomationPath(gatePath, root),
		"status":         firstNonEmptyAny(gate["status"], "unknown"),
		"recommendation": firstNonEmptyAny(gate["recommendation"], "unknown"),
		"failing_checks": firstAnySliceStrings(gate["failing_checks"]),
		"enforcement":    firstMapAny(enforcement),
		"summary":        firstMapAny(summary),
		"reviewer_path":  firstMapAny(reviewerPath),
		"next_actions":   firstAnySliceStrings(nextActions),
	}
}

func automationBuildAvailableArtifacts(root string, specs []struct {
	Path        string
	Description string
}) []map[string]any {
	out := []map[string]any{}
	for _, spec := range specs {
		if _, err := os.Stat(filepath.Join(root, spec.Path)); err == nil {
			out = append(out, map[string]any{"path": spec.Path, "description": spec.Description})
		}
	}
	return out
}

func automationRenderValidationIndex(summary map[string]any, recentRuns []map[string]any, continuationGate map[string]any, continuationArtifacts []map[string]any, followupDigests []map[string]any) string {
	lines := []string{
		"# Live Validation Index",
		"",
		fmt.Sprintf("- Latest run: `%v`", summary["run_id"]),
		fmt.Sprintf("- Generated at: `%v`", summary["generated_at"]),
		fmt.Sprintf("- Status: `%v`", summary["status"]),
		fmt.Sprintf("- Bundle: `%v`", summary["bundle_path"]),
		fmt.Sprintf("- Summary JSON: `%v`", summary["summary_path"]),
		"",
		"## Latest bundle artifacts",
		"",
	}
	for _, name := range []string{"local", "kubernetes", "ray"} {
		section, _ := summary[name].(map[string]any)
		matrix, _ := section["validation_matrix"].(map[string]any)
		lines = append(lines, "### "+name)
		lines = append(lines, fmt.Sprintf("- Enabled: `%v`", section["enabled"]))
		lines = append(lines, fmt.Sprintf("- Status: `%v`", section["status"]))
		if lane := trim(fmt.Sprint(matrix["lane"])); lane != "" {
			lines = append(lines, fmt.Sprintf("- Validation lane: `%s`", lane))
		}
		lines = append(lines, fmt.Sprintf("- Bundle report: `%v`", section["bundle_report_path"]))
		lines = append(lines, fmt.Sprintf("- Latest report: `%v`", section["canonical_report_path"]))
		if value := trim(fmt.Sprint(section["stdout_path"])); value != "" {
			lines = append(lines, fmt.Sprintf("- Stdout log: `%s`", value))
		}
		if value := trim(fmt.Sprint(section["stderr_path"])); value != "" {
			lines = append(lines, fmt.Sprintf("- Stderr log: `%s`", value))
		}
		if value := trim(fmt.Sprint(section["service_log_path"])); value != "" {
			lines = append(lines, fmt.Sprintf("- Service log: `%s`", value))
		}
		if value := trim(fmt.Sprint(section["audit_log_path"])); value != "" {
			lines = append(lines, fmt.Sprintf("- Audit log: `%s`", value))
		}
		if value := trim(fmt.Sprint(section["task_id"])); value != "" {
			lines = append(lines, fmt.Sprintf("- Task ID: `%s`", value))
		}
		if value := trim(fmt.Sprint(section["latest_event_type"])); value != "" {
			lines = append(lines, fmt.Sprintf("- Latest event: `%s`", value))
		}
		if value := trim(fmt.Sprint(section["routing_reason"])); value != "" {
			lines = append(lines, fmt.Sprintf("- Routing reason: `%s`", value))
		}
		rootCause, _ := section["failure_root_cause"].(map[string]any)
		lines = append(lines, fmt.Sprintf("- Failure root cause: status=`%v` event=`%v` location=`%v`", rootCause["status"], rootCause["event_type"], rootCause["location"]))
		if value := trim(fmt.Sprint(rootCause["message"])); value != "" {
			lines = append(lines, fmt.Sprintf("- Failure detail: `%s`", value))
		}
		lines = append(lines, "")
	}
	if validationMatrix, ok := summary["validation_matrix"].([]map[string]any); ok && len(validationMatrix) > 0 {
		lines = append(lines, "## Validation matrix", "")
		for _, row := range validationMatrix {
			lines = append(lines, fmt.Sprintf("- Lane `%v` executor=`%v` status=`%v` enabled=`%v` report=`%v`", row["lane"], row["executor"], row["status"], row["enabled"], row["bundle_report_path"]))
			if trim(fmt.Sprint(row["root_cause_event_type"])) != "" || trim(fmt.Sprint(row["root_cause_message"])) != "" {
				lines = append(lines, fmt.Sprintf("- Lane `%v` root cause: event=`%v` location=`%v` message=`%v`", row["lane"], row["root_cause_event_type"], row["root_cause_location"], row["root_cause_message"]))
			}
		}
		lines = append(lines, "")
	}
	if broker, ok := summary["broker"].(map[string]any); ok {
		lines = append(lines, "### broker")
		lines = append(lines, fmt.Sprintf("- Enabled: `%v`", broker["enabled"]))
		lines = append(lines, fmt.Sprintf("- Status: `%v`", broker["status"]))
		lines = append(lines, fmt.Sprintf("- Configuration state: `%v`", broker["configuration_state"]))
		lines = append(lines, fmt.Sprintf("- Bundle summary: `%v`", broker["bundle_summary_path"]))
		lines = append(lines, fmt.Sprintf("- Canonical summary: `%v`", broker["canonical_summary_path"]))
		lines = append(lines, fmt.Sprintf("- Bundle bootstrap summary: `%v`", broker["bundle_bootstrap_summary_path"]))
		lines = append(lines, fmt.Sprintf("- Canonical bootstrap summary: `%v`", broker["canonical_bootstrap_summary_path"]))
		lines = append(lines, fmt.Sprintf("- Validation pack: `%v`", broker["validation_pack_path"]))
		if value := trim(fmt.Sprint(broker["backend"])); value != "" && value != "<nil>" {
			lines = append(lines, fmt.Sprintf("- Backend: `%s`", value))
		}
		if _, ok := broker["bootstrap_ready"]; ok {
			lines = append(lines, fmt.Sprintf("- Bootstrap ready: `%v`", broker["bootstrap_ready"]))
		}
		if value := trim(fmt.Sprint(broker["runtime_posture"])); value != "" {
			lines = append(lines, fmt.Sprintf("- Runtime posture: `%s`", value))
		}
		if _, ok := broker["live_adapter_implemented"]; ok {
			lines = append(lines, fmt.Sprintf("- Live adapter implemented: `%v`", broker["live_adapter_implemented"]))
		}
		if completeness, ok := broker["config_completeness"].(map[string]any); ok {
			lines = append(lines, fmt.Sprintf("- Config completeness: driver=`%v` urls=`%v` topic=`%v` consumer_group=`%v`", completeness["driver"], completeness["urls"], completeness["topic"], completeness["consumer_group"]))
		}
		if value := trim(fmt.Sprint(broker["proof_boundary"])); value != "" {
			lines = append(lines, fmt.Sprintf("- Proof boundary: `%s`", value))
		}
		for _, raw := range automationAnySlice(broker["validation_errors"]) {
			lines = append(lines, fmt.Sprintf("- Validation error: `%v`", raw))
		}
		if value := trim(fmt.Sprint(broker["bundle_report_path"])); value != "" {
			lines = append(lines, fmt.Sprintf("- Bundle report: `%s`", value))
		}
		if value := trim(fmt.Sprint(broker["canonical_report_path"])); value != "" {
			lines = append(lines, fmt.Sprintf("- Canonical report: `%s`", value))
		}
		if value := trim(fmt.Sprint(broker["reason"])); value != "" {
			lines = append(lines, fmt.Sprintf("- Reason: `%s`", value))
		}
		lines = append(lines, "")
	}
	if sharedQueue, ok := summary["shared_queue_companion"].(map[string]any); ok {
		lines = append(lines, "### shared-queue companion")
		lines = append(lines, fmt.Sprintf("- Available: `%v`", sharedQueue["available"]))
		lines = append(lines, fmt.Sprintf("- Status: `%v`", sharedQueue["status"]))
		lines = append(lines, fmt.Sprintf("- Bundle summary: `%v`", sharedQueue["bundle_summary_path"]))
		lines = append(lines, fmt.Sprintf("- Canonical summary: `%v`", sharedQueue["canonical_summary_path"]))
		lines = append(lines, fmt.Sprintf("- Bundle report: `%v`", sharedQueue["bundle_report_path"]))
		lines = append(lines, fmt.Sprintf("- Canonical report: `%v`", sharedQueue["canonical_report_path"]))
		for _, keyLabel := range []struct{ Key, Label string }{
			{"cross_node_completions", "Cross-node completions"},
			{"duplicate_started_tasks", "Duplicate `task.started`"},
			{"duplicate_completed_tasks", "Duplicate `task.completed`"},
			{"missing_completed_tasks", "Missing terminal completions"},
		} {
			if _, ok := sharedQueue[keyLabel.Key]; ok {
				lines = append(lines, fmt.Sprintf("- %s: `%v`", keyLabel.Label, sharedQueue[keyLabel.Key]))
			}
		}
		lines = append(lines, "")
	}
	lines = append(lines, "## Workflow closeout commands", "")
	for _, command := range firstAnySliceStrings(summary["closeout_commands"]) {
		lines = append(lines, fmt.Sprintf("- `%s`", command))
	}
	lines = append(lines, "", "## Recent bundles", "")
	if len(recentRuns) == 0 {
		lines = append(lines, "- No previous bundles found")
	} else {
		for _, run := range recentRuns {
			lines = append(lines, fmt.Sprintf("- `%v` · `%v` · `%v` · `%v`", run["run_id"], run["status"], run["generated_at"], run["bundle_path"]))
		}
	}
	lines = append(lines, "")
	if continuationGate != nil {
		lines = append(lines, "## Continuation gate", "")
		lines = append(lines, fmt.Sprintf("- Status: `%v`", continuationGate["status"]))
		lines = append(lines, fmt.Sprintf("- Recommendation: `%v`", continuationGate["recommendation"]))
		lines = append(lines, fmt.Sprintf("- Report: `%v`", continuationGate["path"]))
		enforcement, _ := continuationGate["enforcement"].(map[string]any)
		if trim(fmt.Sprint(enforcement["mode"])) != "" {
			lines = append(lines, fmt.Sprintf("- Workflow mode: `%v`", enforcement["mode"]))
		}
		if trim(fmt.Sprint(enforcement["outcome"])) != "" {
			lines = append(lines, fmt.Sprintf("- Workflow outcome: `%v`", enforcement["outcome"]))
		}
		gateSummary, _ := continuationGate["summary"].(map[string]any)
		if trim(fmt.Sprint(gateSummary["latest_run_id"])) != "" {
			lines = append(lines, fmt.Sprintf("- Latest reviewed run: `%v`", gateSummary["latest_run_id"]))
		}
		if _, ok := gateSummary["failing_check_count"]; ok {
			lines = append(lines, fmt.Sprintf("- Failing checks: `%v`", gateSummary["failing_check_count"]))
		}
		if _, ok := gateSummary["workflow_exit_code"]; ok {
			lines = append(lines, fmt.Sprintf("- Workflow exit code on current evidence: `%v`", gateSummary["workflow_exit_code"]))
		}
		reviewerPath, _ := continuationGate["reviewer_path"].(map[string]any)
		if trim(fmt.Sprint(reviewerPath["digest_path"])) != "" {
			lines = append(lines, fmt.Sprintf("- Reviewer digest: `%v`", reviewerPath["digest_path"]))
		}
		if trim(fmt.Sprint(reviewerPath["index_path"])) != "" {
			lines = append(lines, fmt.Sprintf("- Reviewer index: `%v`", reviewerPath["index_path"]))
		}
		for _, action := range firstAnySliceStrings(continuationGate["next_actions"]) {
			lines = append(lines, fmt.Sprintf("- Next action: `%s`", action))
		}
		lines = append(lines, "")
	}
	if len(continuationArtifacts) > 0 {
		lines = append(lines, "## Continuation artifacts", "")
		for _, artifact := range continuationArtifacts {
			lines = append(lines, fmt.Sprintf("- `%v` %v", artifact["path"], artifact["description"]))
		}
		lines = append(lines, "")
	}
	if len(followupDigests) > 0 {
		lines = append(lines, "## Parallel follow-up digests", "")
		for _, digest := range followupDigests {
			lines = append(lines, fmt.Sprintf("- `%v` %v", digest["path"], digest["description"]))
		}
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
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

func firstNonEmptyAny(values ...any) any {
	for _, value := range values {
		text := trim(fmt.Sprint(value))
		if text != "" && text != "<nil>" {
			return text
		}
	}
	return nil
}

func firstIntAny(values ...any) int {
	for _, value := range values {
		switch typed := value.(type) {
		case int:
			return typed
		case int64:
			return int(typed)
		case float64:
			return int(typed)
		}
	}
	return 0
}

func firstStringAny(values ...any) string {
	for _, value := range values {
		if text, ok := value.(string); ok {
			text = trim(text)
			if text != "" {
				return text
			}
		}
	}
	return ""
}

func firstMapAny(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return typed
	default:
		return map[string]any{}
	}
}

func firstAnySliceStrings(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func automationAnySlice(value any) []any {
	items, _ := value.([]any)
	return items
}

func automationExtractNodeNames(nodes []any) []string {
	out := []string{}
	for _, raw := range nodes {
		node, _ := raw.(map[string]any)
		if name := trim(fmt.Sprint(node["name"])); name != "" {
			out = append(out, name)
		}
	}
	return out
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

func parseAutomationTime(value string) time.Time {
	value = trim(value)
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, strings.ReplaceAll(value, "Z", "+00:00"))
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func automationCollectStatusesFromRuns(runs []map[string]any) []string {
	statuses := make([]string, 0, len(runs))
	for _, run := range runs {
		statuses = append(statuses, fmt.Sprint(run["status"]))
	}
	return statuses
}

func automationPythonBoolString(value bool) string {
	if value {
		return "True"
	}
	return "False"
}

func automationPythonListString(values []string) string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		items = append(items, fmt.Sprintf("'%s'", value))
	}
	return "[" + strings.Join(items, ", ") + "]"
}

func automationPythonDictString(values map[string]any) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	items := make([]string, 0, len(keys))
	for _, key := range keys {
		value := values[key]
		switch typed := value.(type) {
		case string:
			items = append(items, fmt.Sprintf("'%s': '%s'", key, typed))
		default:
			items = append(items, fmt.Sprintf("'%s': %v", key, typed))
		}
	}
	return "{" + strings.Join(items, ", ") + "}"
}

func automationValidationLaneScorecard(runs []map[string]any, lane string) map[string]any {
	statuses := []string{}
	enabledRuns := 0
	succeededRuns := 0
	for _, run := range runs {
		section, _ := run[lane].(map[string]any)
		enabled := automationBool(section["enabled"])
		status := "disabled"
		if enabled {
			status = fmt.Sprint(section["status"])
			enabledRuns++
			if status == "succeeded" {
				succeededRuns++
			}
		}
		statuses = append(statuses, status)
	}
	latest := map[string]any{}
	if len(runs) > 0 {
		latest, _ = runs[0][lane].(map[string]any)
	}
	return map[string]any{
		"lane":           lane,
		"latest_enabled": automationBool(latest["enabled"]),
		"latest_status": func() string {
			if len(latest) == 0 {
				return "missing"
			}
			return fmt.Sprint(latest["status"])
		}(),
		"recent_statuses":           statuses,
		"enabled_runs":              enabledRuns,
		"succeeded_runs":            succeededRuns,
		"consecutive_successes":     automationConsecutiveSuccesses(statuses),
		"all_recent_runs_succeeded": enabledRuns > 0 && enabledRuns == succeededRuns,
	}
}

func automationConsecutiveSuccesses(statuses []string) int {
	count := 0
	for _, status := range statuses {
		if status != "succeeded" {
			break
		}
		count++
	}
	return count
}

func normalizeAutomationContinuationEnforcementMode(mode string, legacy bool) (string, error) {
	mode = strings.ToLower(trim(mode))
	if mode == "" {
		if legacy {
			mode = "fail"
		} else {
			mode = "hold"
		}
	}
	switch mode {
	case "review", "hold", "fail":
		return mode, nil
	default:
		return "", fmt.Errorf("unsupported enforcement mode %q; expected one of review, hold, fail", mode)
	}
}

func buildAutomationContinuationEnforcementSummary(recommendation string, mode string) map[string]any {
	if recommendation == "go" {
		return map[string]any{"mode": mode, "outcome": "pass", "exit_code": 0}
	}
	switch mode {
	case "review":
		return map[string]any{"mode": mode, "outcome": "review-only", "exit_code": 0}
	case "hold":
		return map[string]any{"mode": mode, "outcome": "hold", "exit_code": 2}
	default:
		return map[string]any{"mode": mode, "outcome": "fail", "exit_code": 1}
	}
}

func automationPathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
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
