package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	e2eLatestReports = map[string]string{
		"local":      "docs/reports/sqlite-smoke-report.json",
		"kubernetes": "docs/reports/kubernetes-live-smoke-report.json",
		"ray":        "docs/reports/ray-live-smoke-report.json",
	}
	e2eLaneAliases = map[string]string{
		"local":      "local",
		"kubernetes": "k8s",
		"ray":        "ray",
	}
	e2eFailureEventTypes = map[string]bool{
		"task.cancelled":   true,
		"task.dead_letter": true,
		"task.failed":      true,
		"task.retried":     true,
	}
	e2eContinuationArtifacts = []struct {
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
	e2eFollowupDigests = []struct {
		Path        string
		Description string
	}{
		{
			Path:        "docs/reports/validation-bundle-continuation-digest.md",
			Description: "Validation bundle continuation caveats are consolidated here.",
		},
	}
)

const (
	e2eBrokerSummary          = "docs/reports/broker-validation-summary.json"
	e2eBrokerBootstrapSummary = "docs/reports/broker-bootstrap-review-summary.json"
	e2eBrokerValidationPack   = "docs/reports/broker-failover-fault-injection-validation-pack.md"
	e2eSharedQueueReport      = "docs/reports/multi-node-shared-queue-report.json"
	e2eSharedQueueSummary     = "docs/reports/shared-queue-companion-summary.json"
	e2eParallelEvidenceJSON   = "docs/reports/parallel-validation-evidence-bundle.json"
	e2eParallelEvidenceMD     = "docs/reports/parallel-validation-evidence-bundle.md"
)

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
	RunBroker                  bool
	BrokerBackend              string
	BrokerReportPath           string
	BrokerBootstrapSummaryPath string
	ValidationStatus           int
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

type automationContinuationScorecardOptions struct {
	GoRoot            string
	IndexManifestPath string
	BundleRootPath    string
	SummaryPath       string
	SharedQueueReport string
	OutputPath        string
	Now               func() time.Time
}

type automationContinuationPolicyGateOptions struct {
	GoRoot                        string
	ScorecardPath                 string
	OutputPath                    string
	MaxLatestAgeHours             float64
	MinRecentBundles              int
	RequireRepeatedLaneCoverage   bool
	EnforcementMode               string
	LegacyEnforceContinuationGate bool
	Now                           func() time.Time
}

func runAutomationExportValidationBundleCommand(args []string) error {
	flags := flag.NewFlagSet("automation e2e export-validation-bundle", flag.ContinueOnError)
	goRoot := flags.String("go-root", ".", "bigclaw-go repo root")
	runID := flags.String("run-id", "", "bundle run id")
	bundleDir := flags.String("bundle-dir", "", "bundle dir")
	summaryPath := flags.String("summary-path", "docs/reports/live-validation-summary.json", "summary output path")
	indexPath := flags.String("index-path", "docs/reports/live-validation-index.md", "index markdown path")
	manifestPath := flags.String("manifest-path", "docs/reports/live-validation-index.json", "manifest json path")
	runLocal := flags.Bool("run-local", true, "include local lane")
	runKubernetes := flags.Bool("run-kubernetes", true, "include kubernetes lane")
	runRay := flags.Bool("run-ray", true, "include ray lane")
	runBroker := flags.Bool("run-broker", false, "include broker lane")
	brokerBackend := flags.String("broker-backend", "", "broker backend")
	brokerReportPath := flags.String("broker-report-path", "", "broker report path")
	brokerBootstrapSummaryPath := flags.String("broker-bootstrap-summary-path", "", "broker bootstrap summary path")
	validationStatus := flags.Int("validation-status", 0, "workflow validation status")
	localReportPath := flags.String("local-report-path", "", "local report path")
	localStdoutPath := flags.String("local-stdout-path", "", "local stdout path")
	localStderrPath := flags.String("local-stderr-path", "", "local stderr path")
	k8sReportPath := flags.String("kubernetes-report-path", "", "kubernetes report path")
	k8sStdoutPath := flags.String("kubernetes-stdout-path", "", "kubernetes stdout path")
	k8sStderrPath := flags.String("kubernetes-stderr-path", "", "kubernetes stderr path")
	rayReportPath := flags.String("ray-report-path", "", "ray report path")
	rayStdoutPath := flags.String("ray-stdout-path", "", "ray stdout path")
	rayStderrPath := flags.String("ray-stderr-path", "", "ray stderr path")
	asJSON := flags.Bool("json", true, "json")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation e2e export-validation-bundle [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	if trim(*runID) == "" || trim(*bundleDir) == "" {
		return errors.New("--run-id and --bundle-dir are required")
	}
	for _, lane := range []struct {
		enabled bool
		name    string
		paths   []string
	}{
		{enabled: *runLocal, name: "local", paths: []string{*localReportPath, *localStdoutPath, *localStderrPath}},
		{enabled: *runKubernetes, name: "kubernetes", paths: []string{*k8sReportPath, *k8sStdoutPath, *k8sStderrPath}},
		{enabled: *runRay, name: "ray", paths: []string{*rayReportPath, *rayStdoutPath, *rayStderrPath}},
	} {
		if !lane.enabled {
			continue
		}
		for _, required := range lane.paths {
			if trim(required) == "" {
				return fmt.Errorf("%s lane report/stdout/stderr paths are required when enabled", lane.name)
			}
		}
	}

	report, exitCode, err := automationExportValidationBundle(automationExportValidationBundleOptions{
		GoRoot:                     absPath(*goRoot),
		RunID:                      *runID,
		BundleDir:                  *bundleDir,
		SummaryPath:                *summaryPath,
		IndexPath:                  *indexPath,
		ManifestPath:               *manifestPath,
		RunLocal:                   *runLocal,
		RunKubernetes:              *runKubernetes,
		RunRay:                     *runRay,
		RunBroker:                  *runBroker,
		BrokerBackend:              *brokerBackend,
		BrokerReportPath:           *brokerReportPath,
		BrokerBootstrapSummaryPath: *brokerBootstrapSummaryPath,
		ValidationStatus:           *validationStatus,
		LocalReportPath:            *localReportPath,
		LocalStdoutPath:            *localStdoutPath,
		LocalStderrPath:            *localStderrPath,
		KubernetesReportPath:       *k8sReportPath,
		KubernetesStdoutPath:       *k8sStdoutPath,
		KubernetesStderrPath:       *k8sStderrPath,
		RayReportPath:              *rayReportPath,
		RayStdoutPath:              *rayStdoutPath,
		RayStderrPath:              *rayStderrPath,
	})
	if err != nil {
		return err
	}
	return emit(report, *asJSON, exitCode)
}

func runAutomationContinuationScorecardCommand(args []string) error {
	flags := flag.NewFlagSet("automation e2e continuation-scorecard", flag.ContinueOnError)
	goRoot := flags.String("go-root", ".", "bigclaw-go repo root")
	indexManifestPath := flags.String("index-manifest", "docs/reports/live-validation-index.json", "index manifest path")
	bundleRootPath := flags.String("bundle-root", "docs/reports/live-validation-runs", "bundle root path")
	summaryPath := flags.String("summary", "docs/reports/live-validation-summary.json", "latest summary path")
	sharedQueueReport := flags.String("shared-queue-report", "docs/reports/multi-node-shared-queue-report.json", "shared queue report path")
	outputPath := flags.String("output", "docs/reports/validation-bundle-continuation-scorecard.json", "output path")
	asJSON := flags.Bool("json", true, "json")
	pretty := flags.Bool("pretty", false, "pretty-print report to stdout")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation e2e continuation-scorecard [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	report, err := automationContinuationScorecard(automationContinuationScorecardOptions{
		GoRoot:            absPath(*goRoot),
		IndexManifestPath: *indexManifestPath,
		BundleRootPath:    *bundleRootPath,
		SummaryPath:       *summaryPath,
		SharedQueueReport: *sharedQueueReport,
		OutputPath:        *outputPath,
	})
	if err != nil {
		return err
	}
	if *pretty {
		return emit(report, true, 0)
	}
	return emit(report, *asJSON, 0)
}

func runAutomationContinuationPolicyGateCommand(args []string) error {
	flags := flag.NewFlagSet("automation e2e continuation-policy-gate", flag.ContinueOnError)
	goRoot := flags.String("go-root", ".", "bigclaw-go repo root")
	scorecardPath := flags.String("scorecard", "docs/reports/validation-bundle-continuation-scorecard.json", "scorecard path")
	outputPath := flags.String("output", "docs/reports/validation-bundle-continuation-policy-gate.json", "output path")
	maxLatestAgeHours := flags.Float64("max-latest-age-hours", 72.0, "max latest age hours")
	minRecentBundles := flags.Int("min-recent-bundles", 2, "min recent bundles")
	requireRepeated := flags.Bool("require-repeated-lane-coverage", true, "require repeated lane coverage")
	allowPartial := flags.Bool("allow-partial-lane-history", false, "allow partial lane history")
	enforcementMode := flags.String("enforcement-mode", "", "enforcement mode: review|hold|fail")
	legacyEnforce := flags.Bool("enforce", false, "legacy alias for fail mode")
	asJSON := flags.Bool("json", true, "json")
	pretty := flags.Bool("pretty", false, "pretty-print report to stdout")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation e2e continuation-policy-gate [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	report, exitCode, err := automationContinuationPolicyGate(automationContinuationPolicyGateOptions{
		GoRoot:                        absPath(*goRoot),
		ScorecardPath:                 *scorecardPath,
		OutputPath:                    *outputPath,
		MaxLatestAgeHours:             *maxLatestAgeHours,
		MinRecentBundles:              *minRecentBundles,
		RequireRepeatedLaneCoverage:   *requireRepeated && !*allowPartial,
		EnforcementMode:               *enforcementMode,
		LegacyEnforceContinuationGate: *legacyEnforce,
	})
	if err != nil {
		return err
	}
	if *pretty {
		return emit(report, true, exitCode)
	}
	return emit(report, *asJSON, exitCode)
}

func automationExportValidationBundle(opts automationExportValidationBundleOptions) (map[string]any, int, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	root := absPath(opts.GoRoot)
	bundleDir := e2eResolvePath(root, opts.BundleDir)
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return nil, 0, err
	}

	summary := map[string]any{
		"run_id":       opts.RunID,
		"generated_at": now().UTC().Format(time.RFC3339Nano),
		"status":       ternaryString(opts.ValidationStatus == 0, "succeeded", "failed"),
		"bundle_path":  e2eRelPath(root, bundleDir),
		"closeout_commands": []any{
			"cd bigclaw-go && ./scripts/e2e/run_all.sh",
			"cd bigclaw-go && go test ./...",
			"git push origin <branch> && git log -1 --stat",
		},
	}
	var err error
	summary["local"], err = e2eBuildComponentSection("local", opts.RunLocal, root, bundleDir, opts.LocalReportPath, opts.LocalStdoutPath, opts.LocalStderrPath)
	if err != nil {
		return nil, 0, err
	}
	summary["kubernetes"], err = e2eBuildComponentSection("kubernetes", opts.RunKubernetes, root, bundleDir, opts.KubernetesReportPath, opts.KubernetesStdoutPath, opts.KubernetesStderrPath)
	if err != nil {
		return nil, 0, err
	}
	summary["ray"], err = e2eBuildComponentSection("ray", opts.RunRay, root, bundleDir, opts.RayReportPath, opts.RayStdoutPath, opts.RayStderrPath)
	if err != nil {
		return nil, 0, err
	}
	summary["broker"], err = e2eBuildBrokerSection(opts.RunBroker, opts.BrokerBackend, root, bundleDir, opts.BrokerBootstrapSummaryPath, opts.BrokerReportPath)
	if err != nil {
		return nil, 0, err
	}
	summary["shared_queue_companion"], err = e2eBuildSharedQueueCompanion(root, bundleDir)
	if err != nil {
		return nil, 0, err
	}
	summary["validation_matrix"] = e2eBuildValidationMatrix(summary)
	bundleSummaryPath := filepath.Join(bundleDir, "summary.json")
	canonicalSummaryPath := e2eResolvePath(root, opts.SummaryPath)
	summary["summary_path"] = e2eRelPath(root, bundleSummaryPath)
	summary["parallel_validation_evidence_bundle"], err = e2eWriteParallelEvidenceBundle(root, bundleDir, summary)
	if err != nil {
		return nil, 0, err
	}

	continuationGate, err := e2eBuildContinuationGateSummary(root)
	if err != nil {
		return nil, 0, err
	}
	if continuationGate != nil {
		summary["continuation_gate"] = continuationGate
	}

	if err := e2eWriteJSON(bundleSummaryPath, summary); err != nil {
		return nil, 0, err
	}
	if err := e2eWriteJSON(canonicalSummaryPath, summary); err != nil {
		return nil, 0, err
	}

	bundleRoot := filepath.Dir(bundleDir)
	recentRuns, err := e2eBuildRecentRuns(bundleRoot, root, 8)
	if err != nil {
		return nil, 0, err
	}
	manifest := map[string]any{"latest": summary, "recent_runs": recentRuns}
	if continuationGate != nil {
		manifest["continuation_gate"] = continuationGate
	}
	if err := e2eWriteJSON(e2eResolvePath(root, opts.ManifestPath), manifest); err != nil {
		return nil, 0, err
	}

	indexText := e2eRenderIndex(summary, recentRuns, continuationGate, e2eBuildArtifactList(root, e2eContinuationArtifacts), e2eBuildArtifactList(root, e2eFollowupDigests))
	if err := e2eWriteText(e2eResolvePath(root, opts.IndexPath), indexText); err != nil {
		return nil, 0, err
	}
	if err := e2eWriteText(filepath.Join(bundleDir, "README.md"), indexText); err != nil {
		return nil, 0, err
	}

	exitCode := 0
	if stringValue(summary["status"]) != "succeeded" {
		exitCode = 1
	}
	return summary, exitCode, nil
}

func automationContinuationScorecard(opts automationContinuationScorecardOptions) (map[string]any, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	root := absPath(opts.GoRoot)
	manifest, err := e2eReadJSONMap(e2eResolvePath(root, opts.IndexManifestPath))
	if err != nil {
		return nil, err
	}
	latest, _ := manifest["latest"].(map[string]any)
	recentRunsMeta, _ := manifest["recent_runs"].([]any)
	recentRuns := make([]map[string]any, 0, len(recentRunsMeta))
	recentRunInputs := make([]any, 0, len(recentRunsMeta))
	for _, rawItem := range recentRunsMeta {
		item, ok := rawItem.(map[string]any)
		if !ok {
			continue
		}
		summaryPath := e2eResolveEvidencePath(root, item["summary_path"])
		runSummary, err := e2eReadJSONMap(summaryPath)
		if err != nil {
			return nil, err
		}
		recentRuns = append(recentRuns, runSummary)
		recentRunInputs = append(recentRunInputs, e2eRelPath(root, summaryPath))
	}
	latestSummary, err := e2eReadJSONMap(e2eResolvePath(root, opts.SummaryPath))
	if err != nil {
		return nil, err
	}
	sharedQueue, err := e2eReadJSONMap(e2eResolvePath(root, opts.SharedQueueReport))
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if sharedQueue == nil {
		sharedQueue = map[string]any{}
	}
	bundleRoot := e2eResolvePath(root, opts.BundleRootPath)
	bundledSharedQueue, _ := latestSummary["shared_queue_companion"].(map[string]any)

	laneScorecards := make([]any, 0, 3)
	for _, lane := range []string{"local", "kubernetes", "ray"} {
		laneScorecards = append(laneScorecards, e2eBuildLaneScorecard(recentRuns, lane))
	}

	latestGeneratedAt, _ := e2eParseTime(stringValue(latest["generated_at"]))
	var previousGeneratedAt time.Time
	if len(recentRuns) > 1 {
		previousGeneratedAt, _ = e2eParseTime(stringValue(recentRuns[1]["generated_at"]))
	}
	currentTime := now().UTC()
	latestAgeHours := 0.0
	if !latestGeneratedAt.IsZero() {
		latestAgeHours = e2eRound(currentTime.Sub(latestGeneratedAt).Hours(), 2)
	}
	var bundleGapMinutes any
	if !previousGeneratedAt.IsZero() && !latestGeneratedAt.IsZero() {
		bundleGapMinutes = e2eRound(latestGeneratedAt.Sub(previousGeneratedAt).Minutes(), 2)
	}

	latestLaneStatuses := map[string]any{}
	latestAllSucceeded := true
	for _, lane := range []string{"local", "kubernetes", "ray"} {
		section, _ := latestSummary[lane].(map[string]any)
		status := stringValue(section["status"])
		latestLaneStatuses[lane] = status
		if status != "succeeded" {
			latestAllSucceeded = false
		}
	}

	recentAllSucceeded := true
	for _, run := range recentRuns {
		if stringValue(run["status"]) != "succeeded" {
			recentAllSucceeded = false
			break
		}
	}

	repeatedLaneCoverage := true
	enabledRunsByLane := map[string]any{}
	for _, raw := range laneScorecards {
		item, _ := raw.(map[string]any)
		enabledRuns := automationInt(item["enabled_runs"], 0)
		enabledRunsByLane[stringValue(item["lane"])] = enabledRuns
		if enabledRuns < 2 {
			repeatedLaneCoverage = false
		}
	}

	continuationChecks := []any{
		e2eCheck("latest_bundle_all_executor_tracks_succeeded", latestAllSucceeded, fmt.Sprintf("latest lane statuses=%v", latestLaneStatuses)),
		e2eCheck("recent_bundle_chain_has_multiple_runs", len(recentRuns) >= 2, fmt.Sprintf("recent bundle count=%d", len(recentRuns))),
		e2eCheck("recent_bundle_chain_has_no_failures", recentAllSucceeded, fmt.Sprintf("recent bundle statuses=%v", e2eStatuses(recentRuns))),
		e2eCheck("all_executor_tracks_have_repeated_recent_coverage", repeatedLaneCoverage, fmt.Sprintf("enabled_runs_by_lane=%v", enabledRunsByLane)),
		e2eCheck("shared_queue_companion_proof_available", automationBool(bundledSharedQueue["available"]) || automationBool(sharedQueue["all_ok"]), fmt.Sprintf("cross_node_completions=%v", firstNonZero(bundledSharedQueue["cross_node_completions"], sharedQueue["cross_node_completions"]))),
		e2eCheck("continuation_surface_is_workflow_triggered", true, "run_all closeout now refreshes the scorecard and gate automatically, but continuation still depends on explicit workflow execution instead of an always-on service"),
	}

	sharedQueueCompanion := map[string]any{
		"available":                 automationBool(bundledSharedQueue["available"]) || automationBool(sharedQueue["all_ok"]),
		"report_path":               e2eFirstText(bundledSharedQueue["canonical_report_path"], opts.SharedQueueReport),
		"summary_path":              e2eFirstText(bundledSharedQueue["canonical_summary_path"], "docs/reports/shared-queue-companion-summary.json"),
		"bundle_report_path":        bundledSharedQueue["bundle_report_path"],
		"bundle_summary_path":       bundledSharedQueue["bundle_summary_path"],
		"cross_node_completions":    firstNonZero(bundledSharedQueue["cross_node_completions"], sharedQueue["cross_node_completions"]),
		"duplicate_completed_tasks": firstNonZero(bundledSharedQueue["duplicate_completed_tasks"], lenMapSlice(sharedQueue["duplicate_completed_tasks"])),
		"duplicate_started_tasks":   firstNonZero(bundledSharedQueue["duplicate_started_tasks"], lenMapSlice(sharedQueue["duplicate_started_tasks"])),
		"mode":                      ternaryString(len(bundledSharedQueue) > 0, "bundle-companion-summary", "standalone-proof"),
	}

	currentCeiling := []any{
		"continuation across future validation bundles remains workflow-triggered",
		"shared-queue coordination proof now ships as adjacent bundle metadata rather than an executor-native lane",
		"recent history is bounded to the exported bundle index and not an always-on service",
	}
	if !repeatedLaneCoverage {
		currentCeiling = append(currentCeiling, "not every executor lane is enabled across every indexed bundle in the current recent window")
	}
	nextRuntimeHooks := []any{
		"set BIGCLAW_E2E_CONTINUATION_GATE_MODE=hold or fail in workflow closeout when continuation holds should block or fail the run",
		"decide whether shared-queue coordination should stay as adjacent bundle metadata or gain its own executor-native validation lane",
		"extend the automatic continuation refresh beyond run_all.sh into broader workflow orchestrators",
		"extend the scorecard beyond the latest recent_runs window when more longitudinal evidence exists",
	}

	report := map[string]any{
		"generated_at": e2eUTCISO(currentTime),
		"ticket":       "BIG-PAR-086-local-prework",
		"title":        "Validation bundle continuation scorecard",
		"status":       "local-continuation-scorecard",
		"evidence_inputs": map[string]any{
			"manifest_path":            opts.IndexManifestPath,
			"latest_summary_path":      opts.SummaryPath,
			"bundle_root":              opts.BundleRootPath,
			"recent_run_summaries":     recentRunInputs,
			"shared_queue_report_path": opts.SharedQueueReport,
			"generator_script":         "go run ./cmd/bigclawctl automation e2e continuation-scorecard",
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
			"bundle_root_exists":                                e2eFileExists(bundleRoot),
		},
		"executor_lanes":         laneScorecards,
		"shared_queue_companion": sharedQueueCompanion,
		"continuation_checks":    continuationChecks,
		"current_ceiling":        currentCeiling,
		"next_runtime_hooks":     nextRuntimeHooks,
	}
	if err := e2eWriteJSON(e2eResolvePath(root, opts.OutputPath), report); err != nil {
		return nil, err
	}
	return report, nil
}

func automationContinuationPolicyGate(opts automationContinuationPolicyGateOptions) (map[string]any, int, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	root := absPath(opts.GoRoot)
	scorecard, err := e2eReadJSONMap(e2eResolvePath(root, opts.ScorecardPath))
	if err != nil {
		return nil, 0, err
	}
	summary, _ := scorecard["summary"].(map[string]any)
	sharedQueue, _ := scorecard["shared_queue_companion"].(map[string]any)
	mode, err := e2eNormalizeEnforcementMode(opts.EnforcementMode, opts.LegacyEnforceContinuationGate)
	if err != nil {
		return nil, 0, err
	}
	checks := []any{
		e2eCheck("latest_bundle_age_within_threshold", number(summary["latest_bundle_age_hours"]) <= opts.MaxLatestAgeHours, fmt.Sprintf("latest_bundle_age_hours=%v threshold=%v", summary["latest_bundle_age_hours"], opts.MaxLatestAgeHours)),
		e2eCheck("recent_bundle_count_meets_floor", automationInt(summary["recent_bundle_count"], 0) >= opts.MinRecentBundles, fmt.Sprintf("recent_bundle_count=%v floor=%d", summary["recent_bundle_count"], opts.MinRecentBundles)),
		e2eCheck("latest_bundle_all_executor_tracks_succeeded", automationBool(summary["latest_all_executor_tracks_succeeded"]), fmt.Sprintf("latest_all_executor_tracks_succeeded=%v", summary["latest_all_executor_tracks_succeeded"])),
		e2eCheck("recent_bundle_chain_has_no_failures", automationBool(summary["recent_bundle_chain_has_no_failures"]), fmt.Sprintf("recent_bundle_chain_has_no_failures=%v", summary["recent_bundle_chain_has_no_failures"])),
		e2eCheck("shared_queue_companion_available", automationBool(sharedQueue["available"]), fmt.Sprintf("cross_node_completions=%v", sharedQueue["cross_node_completions"])),
		e2eCheck("repeated_lane_coverage_meets_policy", !opts.RequireRepeatedLaneCoverage || automationBool(summary["all_executor_tracks_have_repeated_recent_coverage"]), fmt.Sprintf("require_repeated_lane_coverage=%v actual=%v", opts.RequireRepeatedLaneCoverage, summary["all_executor_tracks_have_repeated_recent_coverage"])),
	}
	failingChecks := make([]any, 0)
	passingCount := 0
	for _, raw := range checks {
		item, _ := raw.(map[string]any)
		if automationBool(item["passed"]) {
			passingCount++
			continue
		}
		failingChecks = append(failingChecks, item["name"])
	}
	recommendation := ternaryString(len(failingChecks) == 0, "go", "hold")
	enforcement := e2eBuildEnforcementSummary(recommendation, mode)
	nextActions := make([]any, 0)
	if e2eContains(failingChecks, "latest_bundle_age_within_threshold") {
		nextActions = append(nextActions, "rerun `cd bigclaw-go && ./scripts/e2e/run_all.sh` to refresh the latest validation bundle")
	}
	if e2eContains(failingChecks, "recent_bundle_count_meets_floor") {
		nextActions = append(nextActions, "export additional validation bundles so the continuation window spans multiple indexed runs")
	}
	if e2eContains(failingChecks, "shared_queue_companion_available") {
		nextActions = append(nextActions, "rerun `go run ./cmd/bigclawctl automation e2e multi-node-shared-queue --report-path docs/reports/multi-node-shared-queue-report.json`")
	}
	if e2eContains(failingChecks, "repeated_lane_coverage_meets_policy") {
		nextActions = append(nextActions, "refresh another full validation bundle with `ray` enabled so each executor lane has repeated indexed coverage")
	}
	if len(nextActions) == 0 {
		nextActions = append(nextActions, "set BIGCLAW_E2E_CONTINUATION_GATE_MODE=review, hold, or fail when workflow closeout should respond to continuation regressions")
	}

	report := map[string]any{
		"generated_at":   e2eUTCISO(now().UTC()),
		"ticket":         "OPE-262",
		"title":          "Validation workflow continuation gate",
		"status":         ternaryString(recommendation == "go", "policy-go", "policy-hold"),
		"recommendation": recommendation,
		"evidence_inputs": map[string]any{
			"scorecard_path":   opts.ScorecardPath,
			"generator_script": "go run ./cmd/bigclawctl automation e2e continuation-policy-gate",
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
			"passing_check_count":                               passingCount,
			"failing_check_count":                               len(failingChecks),
		},
		"policy_checks":          checks,
		"failing_checks":         failingChecks,
		"reviewer_path":          map[string]any{"index_path": "docs/reports/live-validation-index.md", "digest_path": "docs/reports/validation-bundle-continuation-digest.md"},
		"shared_queue_companion": sharedQueue,
		"next_actions":           nextActions,
	}
	if err := e2eWriteJSON(e2eResolvePath(root, opts.OutputPath), report); err != nil {
		return nil, 0, err
	}
	return report, automationInt(enforcement["exit_code"], 0), nil
}

func e2eBuildLaneScorecard(runs []map[string]any, lane string) map[string]any {
	statuses := make([]any, 0, len(runs))
	enabledRuns := 0
	succeededRuns := 0
	for _, run := range runs {
		section, _ := run[lane].(map[string]any)
		enabled := automationBool(section["enabled"])
		status := "disabled"
		if enabled {
			enabledRuns++
			status = e2eFirstText(section["status"], "missing")
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
		"lane":                      lane,
		"latest_enabled":            automationBool(latest["enabled"]),
		"latest_status":             e2eFirstText(latest["status"], "missing"),
		"recent_statuses":           statuses,
		"enabled_runs":              enabledRuns,
		"succeeded_runs":            succeededRuns,
		"consecutive_successes":     e2eConsecutiveSuccesses(statuses),
		"all_recent_runs_succeeded": enabledRuns > 0 && enabledRuns == succeededRuns,
	}
}

func e2eBuildComponentSection(name string, enabled bool, root, bundleDir, reportPath, stdoutPath, stderrPath string) (map[string]any, error) {
	latestReportPath := e2eResolvePath(root, e2eLatestReports[name])
	reportAbs := e2eResolvePath(root, reportPath)
	section := map[string]any{
		"enabled":               enabled,
		"bundle_report_path":    e2eRelPath(root, reportAbs),
		"canonical_report_path": e2eLatestReports[name],
	}
	if !enabled {
		section["status"] = "skipped"
		return section, nil
	}
	report, err := e2eReadJSONMap(reportAbs)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	section["report"] = report
	section["status"] = e2eComponentStatus(report)
	if copied, err := e2eCopyJSON(reportAbs, latestReportPath); err != nil {
		return nil, err
	} else if copied != "" {
		section["canonical_report_path"] = e2eRelPath(root, copied)
	}
	if copied, err := e2eCopyText(stdoutPath, filepath.Join(bundleDir, name+".stdout.log")); err != nil {
		return nil, err
	} else if copied != "" {
		section["stdout_path"] = e2eRelPath(root, copied)
	}
	if copied, err := e2eCopyText(stderrPath, filepath.Join(bundleDir, name+".stderr.log")); err != nil {
		return nil, err
	} else if copied != "" {
		section["stderr_path"] = e2eRelPath(root, copied)
	}
	task, _ := report["task"].(map[string]any)
	if stringValue(task["id"]) != "" {
		section["task_id"] = task["id"]
	}
	if stringValue(report["base_url"]) != "" {
		section["base_url"] = report["base_url"]
	}
	if stateDir := stringValue(report["state_dir"]); stateDir != "" {
		section["state_dir"] = stateDir
		if copied, err := e2eCopyText(filepath.Join(stateDir, "audit.jsonl"), filepath.Join(bundleDir, name+".audit.jsonl")); err != nil {
			return nil, err
		} else if copied != "" {
			section["audit_log_path"] = e2eRelPath(root, copied)
		}
	}
	if serviceLog := stringValue(report["service_log"]); serviceLog != "" {
		if copied, err := e2eCopyText(serviceLog, filepath.Join(bundleDir, name+".service.log")); err != nil {
			return nil, err
		} else if copied != "" {
			section["service_log_path"] = e2eRelPath(root, copied)
		}
	}
	if latestEvent := e2eLatestReportEvent(report); latestEvent != nil {
		section["latest_event_type"] = latestEvent["type"]
		section["latest_event_timestamp"] = latestEvent["timestamp"]
		if payload, ok := latestEvent["payload"].(map[string]any); ok {
			if artifacts, ok := payload["artifacts"].([]any); ok {
				out := make([]any, 0, len(artifacts))
				for _, item := range artifacts {
					if stringValue(item) != "" {
						out = append(out, stringValue(item))
					}
				}
				section["artifact_paths"] = out
			}
		}
	}
	if routingReason := e2eFindRoutingReason(report); routingReason != "" {
		section["routing_reason"] = routingReason
	}
	section["failure_root_cause"] = e2eBuildFailureRootCause(section, report)
	section["validation_matrix"] = e2eBuildValidationMatrixEntry(name, section, report)
	return section, nil
}

func e2eBuildBrokerSection(enabled bool, backend, root, bundleDir, bootstrapSummaryPath, reportPath string) (map[string]any, error) {
	bundleSummaryPath := filepath.Join(bundleDir, "broker-validation-summary.json")
	bundleBootstrapSummaryPath := filepath.Join(bundleDir, "broker-bootstrap-review-summary.json")
	section := map[string]any{
		"enabled":                          enabled,
		"backend":                          nilIfEmpty(backend),
		"bundle_summary_path":              e2eRelPath(root, bundleSummaryPath),
		"canonical_summary_path":           e2eBrokerSummary,
		"bundle_bootstrap_summary_path":    e2eRelPath(root, bundleBootstrapSummaryPath),
		"canonical_bootstrap_summary_path": e2eBrokerBootstrapSummary,
		"validation_pack_path":             e2eBrokerValidationPack,
		"configuration_state":              ternaryString(enabled && trim(backend) != "", "configured", "not_configured"),
	}
	var bootstrapSummary map[string]any
	if trim(bootstrapSummaryPath) != "" {
		var err error
		bootstrapSummary, err = e2eReadJSONMap(e2eResolvePath(root, bootstrapSummaryPath))
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}
	if len(bootstrapSummary) > 0 {
		if copied, err := e2eCopyJSON(e2eResolvePath(root, bootstrapSummaryPath), bundleBootstrapSummaryPath); err != nil {
			return nil, err
		} else if copied != "" {
			section["bundle_bootstrap_summary_path"] = e2eRelPath(root, copied)
		}
		if copied, err := e2eCopyJSON(e2eResolvePath(root, bootstrapSummaryPath), e2eResolvePath(root, e2eBrokerBootstrapSummary)); err != nil {
			return nil, err
		} else if copied != "" {
			section["canonical_bootstrap_summary_path"] = e2eRelPath(root, copied)
		}
		section["bootstrap_summary"] = bootstrapSummary
		section["bootstrap_ready"] = automationBool(bootstrapSummary["ready"])
		section["runtime_posture"] = bootstrapSummary["runtime_posture"]
		section["live_adapter_implemented"] = automationBool(bootstrapSummary["live_adapter_implemented"])
		section["proof_boundary"] = bootstrapSummary["proof_boundary"]
		if errorsRaw, ok := bootstrapSummary["validation_errors"].([]any); ok {
			section["validation_errors"] = errorsRaw
		}
		if completeness, ok := bootstrapSummary["config_completeness"].(map[string]any); ok {
			section["config_completeness"] = completeness
		}
	}
	if !enabled || trim(backend) == "" {
		section["status"] = "skipped"
		section["reason"] = "not_configured"
		if err := e2eWriteJSON(bundleSummaryPath, section); err != nil {
			return nil, err
		}
		if err := e2eWriteJSON(e2eResolvePath(root, e2eBrokerSummary), section); err != nil {
			return nil, err
		}
		return section, nil
	}
	if trim(reportPath) == "" {
		section["status"] = "skipped"
		section["reason"] = "missing_report_path"
		if err := e2eWriteJSON(bundleSummaryPath, section); err != nil {
			return nil, err
		}
		if err := e2eWriteJSON(e2eResolvePath(root, e2eBrokerSummary), section); err != nil {
			return nil, err
		}
		return section, nil
	}
	reportAbs := e2eResolvePath(root, reportPath)
	report, err := e2eReadJSONMap(reportAbs)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	section["canonical_report_path"] = e2eRelPath(root, reportAbs)
	section["bundle_report_path"] = e2eRelPath(root, filepath.Join(bundleDir, filepath.Base(reportAbs)))
	if len(report) == 0 {
		section["status"] = "skipped"
		section["reason"] = "not_configured"
		if err := e2eWriteJSON(bundleSummaryPath, section); err != nil {
			return nil, err
		}
		if err := e2eWriteJSON(e2eResolvePath(root, e2eBrokerSummary), section); err != nil {
			return nil, err
		}
		return section, nil
	}
	if copied, err := e2eCopyJSON(reportAbs, filepath.Join(bundleDir, filepath.Base(reportAbs))); err != nil {
		return nil, err
	} else if copied != "" {
		section["bundle_report_path"] = e2eRelPath(root, copied)
	}
	section["report"] = report
	section["status"] = e2eComponentStatus(report)
	if err := e2eWriteJSON(bundleSummaryPath, section); err != nil {
		return nil, err
	}
	if err := e2eWriteJSON(e2eResolvePath(root, e2eBrokerSummary), section); err != nil {
		return nil, err
	}
	return section, nil
}

func e2eBuildSharedQueueCompanion(root, bundleDir string) (map[string]any, error) {
	canonicalReportPath := e2eResolvePath(root, e2eSharedQueueReport)
	canonicalSummaryPath := e2eResolvePath(root, e2eSharedQueueSummary)
	bundleReportPath := filepath.Join(bundleDir, "multi-node-shared-queue-report.json")
	bundleSummaryPath := filepath.Join(bundleDir, "shared-queue-companion-summary.json")
	report, err := e2eReadJSONMap(canonicalReportPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	summary := map[string]any{
		"available":              len(report) > 0,
		"canonical_report_path":  e2eSharedQueueReport,
		"canonical_summary_path": e2eSharedQueueSummary,
		"bundle_report_path":     e2eRelPath(root, bundleReportPath),
		"bundle_summary_path":    e2eRelPath(root, bundleSummaryPath),
	}
	if len(report) == 0 {
		summary["status"] = "missing_report"
		return summary, nil
	}
	if copied, err := e2eCopyJSON(canonicalReportPath, bundleReportPath); err != nil {
		return nil, err
	} else if copied != "" {
		summary["bundle_report_path"] = e2eRelPath(root, copied)
	}
	summary["status"] = ternaryString(automationBool(report["all_ok"]), "succeeded", "failed")
	summary["generated_at"] = report["generated_at"]
	summary["count"] = report["count"]
	summary["cross_node_completions"] = report["cross_node_completions"]
	summary["duplicate_started_tasks"] = lenMapSlice(report["duplicate_started_tasks"])
	summary["duplicate_completed_tasks"] = lenMapSlice(report["duplicate_completed_tasks"])
	summary["missing_completed_tasks"] = lenMapSlice(report["missing_completed_tasks"])
	summary["submitted_by_node"] = report["submitted_by_node"]
	summary["completed_by_node"] = report["completed_by_node"]
	if nodes, ok := report["nodes"].([]any); ok {
		out := make([]any, 0, len(nodes))
		for _, raw := range nodes {
			node, _ := raw.(map[string]any)
			if name := stringValue(node["name"]); name != "" {
				out = append(out, name)
			}
		}
		summary["nodes"] = out
	}
	if err := e2eWriteJSON(bundleSummaryPath, summary); err != nil {
		return nil, err
	}
	if err := e2eWriteJSON(canonicalSummaryPath, summary); err != nil {
		return nil, err
	}
	return summary, nil
}

func e2eBuildValidationMatrix(summary map[string]any) []any {
	rows := make([]any, 0)
	for _, name := range []string{"local", "kubernetes", "ray"} {
		section, _ := summary[name].(map[string]any)
		row, _ := section["validation_matrix"].(map[string]any)
		if len(row) > 0 {
			rows = append(rows, row)
		}
	}
	return rows
}

func e2eBuildValidationMatrixEntry(name string, section, report map[string]any) map[string]any {
	task, _ := report["task"].(map[string]any)
	taskID := firstNonEmpty(task["id"], section["task_id"])
	executor := firstNonEmpty(task["required_executor"], name)
	rootCause, _ := section["failure_root_cause"].(map[string]any)
	return map[string]any{
		"lane":                     e2eLaneAliases[name],
		"executor":                 executor,
		"enabled":                  automationBool(section["enabled"]),
		"status":                   firstNonEmpty(section["status"], "unknown"),
		"task_id":                  taskID,
		"canonical_report_path":    firstNonEmpty(section["canonical_report_path"]),
		"bundle_report_path":       firstNonEmpty(section["bundle_report_path"]),
		"latest_event_type":        firstNonEmpty(section["latest_event_type"]),
		"routing_reason":           firstNonEmpty(section["routing_reason"]),
		"root_cause_event_type":    firstNonEmpty(rootCause["event_type"]),
		"root_cause_location":      firstNonEmpty(rootCause["location"]),
		"root_cause_location_kind": firstNonEmpty(rootCause["location_kind"]),
		"root_cause_message":       firstNonEmpty(rootCause["message"]),
		"root_cause_event_id":      firstNonEmpty(rootCause["event_id"]),
		"root_cause_timestamp":     firstNonEmpty(rootCause["timestamp"]),
		"root_cause_status":        firstNonEmpty(rootCause["status"]),
	}
}

func e2eBuildFailureRootCause(section, report map[string]any) map[string]any {
	events := e2eCollectReportEvents(report)
	latestEvent := e2eLatestReportEvent(report)
	latestStatus := firstNonEmpty(nestedString(report, "status", "state"), nestedString(report, "task", "state"), e2eComponentStatus(report))
	var causeEvent map[string]any
	for i := len(events) - 1; i >= 0; i-- {
		if e2eFailureEventTypes[firstNonEmpty(events[i]["type"])] {
			causeEvent = events[i]
			break
		}
	}
	if len(causeEvent) == 0 && latestStatus != "" && latestStatus != "succeeded" {
		causeEvent = latestEvent
	}
	location, locationKind := e2eRootCauseLocation(section)
	if len(causeEvent) == 0 {
		return map[string]any{
			"status":                "not_triggered",
			"event_type":            firstNonEmpty(latestEvent["type"]),
			"message":               "",
			"location":              location,
			"location_kind":         locationKind,
			"bundle_report_path":    firstNonEmpty(section["bundle_report_path"]),
			"canonical_report_path": firstNonEmpty(section["canonical_report_path"]),
			"event_id":              "",
			"timestamp":             "",
		}
	}
	return map[string]any{
		"status":     "captured",
		"event_type": firstNonEmpty(causeEvent["type"]),
		"message": firstNonEmpty(
			e2eEventPayloadText(causeEvent, "message"),
			e2eEventPayloadText(causeEvent, "reason"),
			report["error"],
			report["failure_reason"],
		),
		"location":              location,
		"location_kind":         locationKind,
		"bundle_report_path":    firstNonEmpty(section["bundle_report_path"]),
		"canonical_report_path": firstNonEmpty(section["canonical_report_path"]),
		"event_id":              firstNonEmpty(causeEvent["id"]),
		"timestamp":             firstNonEmpty(causeEvent["timestamp"]),
	}
}

func e2eRootCauseLocation(section map[string]any) (string, string) {
	for _, candidate := range []struct {
		field string
		kind  string
	}{
		{field: "stderr_path", kind: "stderr_log"},
		{field: "service_log_path", kind: "service_log"},
		{field: "audit_log_path", kind: "audit_log"},
		{field: "bundle_report_path", kind: "bundle_report"},
	} {
		if value := firstNonEmpty(section[candidate.field]); value != "" {
			return value, candidate.kind
		}
	}
	return "", ""
}

func e2eWriteParallelEvidenceBundle(root, bundleDir string, summary map[string]any) (map[string]any, error) {
	bundleJSONPath := filepath.Join(bundleDir, "parallel-validation-evidence-bundle.json")
	bundleMDPath := filepath.Join(bundleDir, "parallel-validation-evidence-bundle.md")
	canonicalJSONPath := e2eResolvePath(root, e2eParallelEvidenceJSON)
	canonicalMDPath := e2eResolvePath(root, e2eParallelEvidenceMD)

	report := e2eBuildParallelEvidenceBundle(root, bundleDir, summary)
	markdown := e2eRenderParallelEvidenceBundle(report)

	if err := e2eWriteJSON(bundleJSONPath, report); err != nil {
		return nil, err
	}
	if err := e2eWriteJSON(canonicalJSONPath, report); err != nil {
		return nil, err
	}
	if err := e2eWriteText(bundleMDPath, markdown); err != nil {
		return nil, err
	}
	if err := e2eWriteText(canonicalMDPath, markdown); err != nil {
		return nil, err
	}

	reportSummary, _ := report["summary"].(map[string]any)
	return map[string]any{
		"status":                  report["status"],
		"lane_count":              reportSummary["lane_count"],
		"enabled_lane_count":      reportSummary["enabled_lane_count"],
		"succeeded_lane_count":    reportSummary["succeeded_lane_count"],
		"failing_lane_count":      reportSummary["failing_lane_count"],
		"canonical_json_path":     e2eParallelEvidenceJSON,
		"canonical_markdown_path": e2eParallelEvidenceMD,
		"bundle_json_path":        e2eRelPath(root, bundleJSONPath),
		"bundle_markdown_path":    e2eRelPath(root, bundleMDPath),
	}, nil
}

func e2eBuildParallelEvidenceBundle(root, bundleDir string, summary map[string]any) map[string]any {
	lanes := make([]any, 0, 3)
	validationMatrix, _ := summary["validation_matrix"].([]any)
	enabledLaneCount := 0
	succeededLaneCount := 0
	failingLaneCount := 0
	skippedLaneCount := 0
	rootCauseCount := 0
	for _, name := range []string{"local", "kubernetes", "ray"} {
		section, _ := summary[name].(map[string]any)
		matrixRow, _ := section["validation_matrix"].(map[string]any)
		rootCause, _ := section["failure_root_cause"].(map[string]any)
		status := firstNonEmpty(section["status"], "unknown")
		if automationBool(section["enabled"]) {
			enabledLaneCount++
		}
		switch status {
		case "succeeded":
			succeededLaneCount++
		case "skipped":
			skippedLaneCount++
		default:
			failingLaneCount++
		}
		if stringValue(rootCause["location"]) != "" {
			rootCauseCount++
		}
		lanes = append(lanes, map[string]any{
			"lane":                  firstNonEmpty(matrixRow["lane"], e2eLaneAliases[name]),
			"executor":              firstNonEmpty(matrixRow["executor"], name),
			"enabled":               automationBool(section["enabled"]),
			"status":                status,
			"task_id":               firstNonEmpty(section["task_id"]),
			"routing_reason":        firstNonEmpty(section["routing_reason"]),
			"latest_event_type":     firstNonEmpty(section["latest_event_type"]),
			"canonical_report_path": firstNonEmpty(section["canonical_report_path"]),
			"bundle_report_path":    firstNonEmpty(section["bundle_report_path"]),
			"stdout_path":           firstNonEmpty(section["stdout_path"]),
			"stderr_path":           firstNonEmpty(section["stderr_path"]),
			"audit_log_path":        firstNonEmpty(section["audit_log_path"]),
			"service_log_path":      firstNonEmpty(section["service_log_path"]),
			"failure_root_cause":    rootCause,
		})
	}

	return map[string]any{
		"generated_at": e2eFirstText(summary["generated_at"], e2eUTCISO(time.Now().UTC())),
		"ticket":       "BIGCLAW-173",
		"title":        "Parallel validation evidence bundle",
		"status":       firstNonEmpty(summary["status"], "unknown"),
		"run_id":       firstNonEmpty(summary["run_id"]),
		"bundle_path":  firstNonEmpty(summary["bundle_path"], e2eRelPath(root, bundleDir)),
		"evidence_inputs": map[string]any{
			"live_validation_summary": firstNonEmpty(summary["summary_path"], "docs/reports/live-validation-summary.json"),
			"live_validation_index":   "docs/reports/live-validation-index.md",
			"executors":               []any{"local", "kubernetes", "ray"},
		},
		"summary": map[string]any{
			"lane_count":           len(lanes),
			"enabled_lane_count":   enabledLaneCount,
			"succeeded_lane_count": succeededLaneCount,
			"failing_lane_count":   failingLaneCount,
			"skipped_lane_count":   skippedLaneCount,
			"root_cause_count":     rootCauseCount,
		},
		"validation_matrix": validationMatrix,
		"lanes":             lanes,
	}
}

func e2eRenderParallelEvidenceBundle(report map[string]any) string {
	lines := []string{
		"# Parallel Validation Evidence Bundle",
		"",
		fmt.Sprintf("- Run ID: `%s`", report["run_id"]),
		fmt.Sprintf("- Generated at: `%s`", report["generated_at"]),
		fmt.Sprintf("- Status: `%s`", report["status"]),
		"",
		"## Matrix",
		"",
	}
	if summary, ok := report["summary"].(map[string]any); ok {
		lines = append(lines,
			fmt.Sprintf("- Lane count: `%v`", summary["lane_count"]),
			fmt.Sprintf("- Enabled lanes: `%v`", summary["enabled_lane_count"]),
			fmt.Sprintf("- Succeeded lanes: `%v`", summary["succeeded_lane_count"]),
			fmt.Sprintf("- Failing lanes: `%v`", summary["failing_lane_count"]),
			fmt.Sprintf("- Root causes localized: `%v`", summary["root_cause_count"]),
			"",
		)
	}
	if rows, ok := report["validation_matrix"].([]any); ok {
		for _, raw := range rows {
			row, _ := raw.(map[string]any)
			lines = append(lines, fmt.Sprintf("- Lane `%s` executor=`%s` status=`%s` enabled=`%v` report=`%s`", row["lane"], row["executor"], row["status"], row["enabled"], row["bundle_report_path"]))
			lines = append(lines, fmt.Sprintf("- Lane `%s` root cause: event=`%s` location=`%s` kind=`%s` message=`%s`", row["lane"], firstNonEmpty(row["root_cause_event_type"], "not_triggered"), firstNonEmpty(row["root_cause_location"], "n/a"), firstNonEmpty(row["root_cause_location_kind"], "n/a"), firstNonEmpty(row["root_cause_message"], "")))
		}
		lines = append(lines, "")
	}
	lines = append(lines, "## Lane Details", "")
	if lanes, ok := report["lanes"].([]any); ok {
		for _, raw := range lanes {
			lane, _ := raw.(map[string]any)
			rootCause, _ := lane["failure_root_cause"].(map[string]any)
			lines = append(lines, "### "+firstNonEmpty(lane["lane"], "unknown"))
			lines = append(lines, fmt.Sprintf("- Executor: `%s`", firstNonEmpty(lane["executor"], "unknown")))
			lines = append(lines, fmt.Sprintf("- Enabled: `%v`", lane["enabled"]))
			lines = append(lines, fmt.Sprintf("- Status: `%s`", firstNonEmpty(lane["status"], "unknown")))
			if stringValue(lane["task_id"]) != "" {
				lines = append(lines, fmt.Sprintf("- Task ID: `%s`", lane["task_id"]))
			}
			if stringValue(lane["bundle_report_path"]) != "" {
				lines = append(lines, fmt.Sprintf("- Bundle report: `%s`", lane["bundle_report_path"]))
			}
			if stringValue(lane["canonical_report_path"]) != "" {
				lines = append(lines, fmt.Sprintf("- Canonical report: `%s`", lane["canonical_report_path"]))
			}
			lines = append(lines, fmt.Sprintf("- Failure root cause: status=`%s` event=`%s` location=`%s` kind=`%s`", firstNonEmpty(rootCause["status"], "unknown"), firstNonEmpty(rootCause["event_type"], "not_triggered"), firstNonEmpty(rootCause["location"], "n/a"), firstNonEmpty(rootCause["location_kind"], "n/a")))
			if stringValue(rootCause["message"]) != "" {
				lines = append(lines, fmt.Sprintf("- Failure detail: `%s`", rootCause["message"]))
			}
			lines = append(lines, "")
		}
	}
	return strings.Join(lines, "\n")
}

func e2eBuildRecentRuns(bundleRoot, root string, limit int) ([]any, error) {
	if !e2eFileExists(bundleRoot) {
		return []any{}, nil
	}
	type item struct {
		GeneratedAt string
		Summary     map[string]any
	}
	items := make([]item, 0)
	entries, err := os.ReadDir(bundleRoot)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		summaryPath := filepath.Join(bundleRoot, entry.Name(), "summary.json")
		summary, err := e2eReadJSONMap(summaryPath)
		if err != nil || len(summary) == 0 {
			continue
		}
		items = append(items, item{GeneratedAt: stringValue(summary["generated_at"]), Summary: summary})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].GeneratedAt > items[j].GeneratedAt })
	out := make([]any, 0, limit)
	for _, item := range items {
		if len(out) >= limit {
			break
		}
		out = append(out, map[string]any{
			"run_id":       item.Summary["run_id"],
			"generated_at": item.Summary["generated_at"],
			"status":       firstNonEmpty(item.Summary["status"], "unknown"),
			"bundle_path":  item.Summary["bundle_path"],
			"summary_path": item.Summary["summary_path"],
		})
	}
	return out, nil
}

func e2eBuildContinuationGateSummary(root string) (map[string]any, error) {
	path := filepath.Join(root, "docs/reports/validation-bundle-continuation-policy-gate.json")
	gate, err := e2eReadJSONMap(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if len(gate) == 0 {
		return nil, nil
	}
	enforcement, _ := gate["enforcement"].(map[string]any)
	summary, _ := gate["summary"].(map[string]any)
	reviewerPath, _ := gate["reviewer_path"].(map[string]any)
	nextActions, _ := gate["next_actions"].([]any)
	return map[string]any{
		"path":           e2eRelPath(root, path),
		"status":         firstNonEmpty(gate["status"], "unknown"),
		"recommendation": firstNonEmpty(gate["recommendation"], "unknown"),
		"failing_checks": firstNonZero(gate["failing_checks"], []any{}),
		"enforcement":    enforcement,
		"summary":        summary,
		"reviewer_path":  reviewerPath,
		"next_actions":   nextActions,
	}, nil
}

func e2eRenderIndex(summary map[string]any, recentRuns []any, continuationGate map[string]any, continuationArtifacts, followupDigests []any) string {
	lines := []string{
		"# Live Validation Index",
		"",
		fmt.Sprintf("- Latest run: `%s`", summary["run_id"]),
		fmt.Sprintf("- Generated at: `%s`", summary["generated_at"]),
		fmt.Sprintf("- Status: `%s`", summary["status"]),
		fmt.Sprintf("- Bundle: `%s`", summary["bundle_path"]),
		fmt.Sprintf("- Summary JSON: `%s`", summary["summary_path"]),
		"",
		"## Latest bundle artifacts",
		"",
	}
	for _, name := range []string{"local", "kubernetes", "ray"} {
		section, _ := summary[name].(map[string]any)
		matrix, _ := section["validation_matrix"].(map[string]any)
		lines = append(lines, "### "+name)
		lines = append(lines, fmt.Sprintf("- Enabled: `%v`", section["enabled"]))
		lines = append(lines, fmt.Sprintf("- Status: `%s`", section["status"]))
		if stringValue(matrix["lane"]) != "" {
			lines = append(lines, fmt.Sprintf("- Validation lane: `%s`", matrix["lane"]))
		}
		lines = append(lines, fmt.Sprintf("- Bundle report: `%s`", section["bundle_report_path"]))
		lines = append(lines, fmt.Sprintf("- Latest report: `%s`", section["canonical_report_path"]))
		for _, key := range []struct{ Label, Field string }{{"Stdout log", "stdout_path"}, {"Stderr log", "stderr_path"}, {"Service log", "service_log_path"}, {"Audit log", "audit_log_path"}} {
			if stringValue(section[key.Field]) != "" {
				lines = append(lines, fmt.Sprintf("- %s: `%s`", key.Label, section[key.Field]))
			}
		}
		if stringValue(section["task_id"]) != "" {
			lines = append(lines, fmt.Sprintf("- Task ID: `%s`", section["task_id"]))
		}
		if stringValue(section["latest_event_type"]) != "" {
			lines = append(lines, fmt.Sprintf("- Latest event: `%s`", section["latest_event_type"]))
		}
		if stringValue(section["routing_reason"]) != "" {
			lines = append(lines, fmt.Sprintf("- Routing reason: `%s`", section["routing_reason"]))
		}
		rootCause, _ := section["failure_root_cause"].(map[string]any)
		if len(rootCause) > 0 {
			lines = append(lines, fmt.Sprintf("- Failure root cause: status=`%s` event=`%s` location=`%s` kind=`%s`", firstNonEmpty(rootCause["status"], "unknown"), firstNonEmpty(rootCause["event_type"], "unknown"), firstNonEmpty(rootCause["location"], "n/a"), firstNonEmpty(rootCause["location_kind"], "n/a")))
			if stringValue(rootCause["message"]) != "" {
				lines = append(lines, fmt.Sprintf("- Failure detail: `%s`", rootCause["message"]))
			}
		}
		lines = append(lines, "")
	}
	if matrix, ok := summary["validation_matrix"].([]any); ok && len(matrix) > 0 {
		lines = append(lines, "## Validation matrix", "")
		for _, raw := range matrix {
			row, _ := raw.(map[string]any)
			lines = append(lines, fmt.Sprintf("- Lane `%s` executor=`%s` status=`%s` enabled=`%v` report=`%s`", row["lane"], row["executor"], row["status"], row["enabled"], row["bundle_report_path"]))
			if stringValue(row["root_cause_event_type"]) != "" || stringValue(row["root_cause_message"]) != "" {
				lines = append(lines, fmt.Sprintf("- Lane `%s` root cause: event=`%s` location=`%s` kind=`%s` message=`%s`", row["lane"], row["root_cause_event_type"], row["root_cause_location"], firstNonEmpty(row["root_cause_location_kind"], "n/a"), row["root_cause_message"]))
			}
		}
		lines = append(lines, "")
	}
	if evidence, ok := summary["parallel_validation_evidence_bundle"].(map[string]any); ok && len(evidence) > 0 {
		lines = append(lines, "## Parallel evidence bundle", "")
		lines = append(lines, fmt.Sprintf("- Status: `%s`", firstNonEmpty(evidence["status"], "unknown")))
		lines = append(lines, fmt.Sprintf("- Canonical JSON: `%s`", firstNonEmpty(evidence["canonical_json_path"])))
		lines = append(lines, fmt.Sprintf("- Canonical Markdown: `%s`", firstNonEmpty(evidence["canonical_markdown_path"])))
		lines = append(lines, fmt.Sprintf("- Bundle JSON: `%s`", firstNonEmpty(evidence["bundle_json_path"])))
		lines = append(lines, fmt.Sprintf("- Bundle Markdown: `%s`", firstNonEmpty(evidence["bundle_markdown_path"])))
		lines = append(lines, "")
	}
	if broker, ok := summary["broker"].(map[string]any); ok {
		lines = append(lines, "### broker")
		lines = append(lines, fmt.Sprintf("- Enabled: `%v`", broker["enabled"]))
		lines = append(lines, fmt.Sprintf("- Status: `%s`", broker["status"]))
		lines = append(lines, fmt.Sprintf("- Configuration state: `%s`", broker["configuration_state"]))
		lines = append(lines, fmt.Sprintf("- Bundle summary: `%s`", broker["bundle_summary_path"]))
		lines = append(lines, fmt.Sprintf("- Canonical summary: `%s`", broker["canonical_summary_path"]))
		lines = append(lines, fmt.Sprintf("- Bundle bootstrap summary: `%s`", broker["bundle_bootstrap_summary_path"]))
		lines = append(lines, fmt.Sprintf("- Canonical bootstrap summary: `%s`", broker["canonical_bootstrap_summary_path"]))
		lines = append(lines, fmt.Sprintf("- Validation pack: `%s`", broker["validation_pack_path"]))
		if stringValue(broker["backend"]) != "" {
			lines = append(lines, fmt.Sprintf("- Backend: `%s`", broker["backend"]))
		}
		if _, ok := broker["bootstrap_ready"]; ok {
			lines = append(lines, fmt.Sprintf("- Bootstrap ready: `%v`", broker["bootstrap_ready"]))
		}
		if stringValue(broker["runtime_posture"]) != "" {
			lines = append(lines, fmt.Sprintf("- Runtime posture: `%s`", broker["runtime_posture"]))
		}
		if _, ok := broker["live_adapter_implemented"]; ok {
			lines = append(lines, fmt.Sprintf("- Live adapter implemented: `%v`", broker["live_adapter_implemented"]))
		}
		if completeness, ok := broker["config_completeness"].(map[string]any); ok {
			lines = append(lines, fmt.Sprintf("- Config completeness: driver=`%v` urls=`%v` topic=`%v` consumer_group=`%v`", completeness["driver"], completeness["urls"], completeness["topic"], completeness["consumer_group"]))
		}
		if stringValue(broker["proof_boundary"]) != "" {
			lines = append(lines, fmt.Sprintf("- Proof boundary: `%s`", broker["proof_boundary"]))
		}
		if validationErrors, ok := broker["validation_errors"].([]any); ok {
			for _, item := range validationErrors {
				lines = append(lines, fmt.Sprintf("- Validation error: `%s`", item))
			}
		}
		if stringValue(broker["bundle_report_path"]) != "" {
			lines = append(lines, fmt.Sprintf("- Bundle report: `%s`", broker["bundle_report_path"]))
		}
		if stringValue(broker["canonical_report_path"]) != "" {
			lines = append(lines, fmt.Sprintf("- Canonical report: `%s`", broker["canonical_report_path"]))
		}
		if stringValue(broker["reason"]) != "" {
			lines = append(lines, fmt.Sprintf("- Reason: `%s`", broker["reason"]))
		}
		lines = append(lines, "")
	}
	if sharedQueue, ok := summary["shared_queue_companion"].(map[string]any); ok {
		lines = append(lines, "### shared-queue companion")
		lines = append(lines, fmt.Sprintf("- Available: `%v`", sharedQueue["available"]))
		lines = append(lines, fmt.Sprintf("- Status: `%s`", sharedQueue["status"]))
		lines = append(lines, fmt.Sprintf("- Bundle summary: `%s`", sharedQueue["bundle_summary_path"]))
		lines = append(lines, fmt.Sprintf("- Canonical summary: `%s`", sharedQueue["canonical_summary_path"]))
		lines = append(lines, fmt.Sprintf("- Bundle report: `%s`", sharedQueue["bundle_report_path"]))
		lines = append(lines, fmt.Sprintf("- Canonical report: `%s`", sharedQueue["canonical_report_path"]))
		for _, field := range []struct{ Label, Key string }{{"Cross-node completions", "cross_node_completions"}, {"Duplicate `task.started`", "duplicate_started_tasks"}, {"Duplicate `task.completed`", "duplicate_completed_tasks"}, {"Missing terminal completions", "missing_completed_tasks"}} {
			if _, ok := sharedQueue[field.Key]; ok {
				lines = append(lines, fmt.Sprintf("- %s: `%v`", field.Label, sharedQueue[field.Key]))
			}
		}
		lines = append(lines, "")
	}
	lines = append(lines, "## Workflow closeout commands", "")
	if commands, ok := summary["closeout_commands"].([]any); ok {
		for _, command := range commands {
			lines = append(lines, fmt.Sprintf("- `%s`", command))
		}
	}
	lines = append(lines, "", "## Recent bundles", "")
	if len(recentRuns) == 0 {
		lines = append(lines, "- No previous bundles found")
	} else {
		for _, raw := range recentRuns {
			run, _ := raw.(map[string]any)
			lines = append(lines, fmt.Sprintf("- `%s` · `%s` · `%s` · `%s`", run["run_id"], run["status"], run["generated_at"], run["bundle_path"]))
		}
	}
	lines = append(lines, "")
	if continuationGate != nil {
		lines = append(lines, "## Continuation gate", "")
		lines = append(lines, fmt.Sprintf("- Status: `%s`", continuationGate["status"]))
		lines = append(lines, fmt.Sprintf("- Recommendation: `%s`", continuationGate["recommendation"]))
		lines = append(lines, fmt.Sprintf("- Report: `%s`", continuationGate["path"]))
		enforcement, _ := continuationGate["enforcement"].(map[string]any)
		if stringValue(enforcement["mode"]) != "" {
			lines = append(lines, fmt.Sprintf("- Workflow mode: `%s`", enforcement["mode"]))
		}
		if stringValue(enforcement["outcome"]) != "" {
			lines = append(lines, fmt.Sprintf("- Workflow outcome: `%s`", enforcement["outcome"]))
		}
		gateSummary, _ := continuationGate["summary"].(map[string]any)
		if stringValue(gateSummary["latest_run_id"]) != "" {
			lines = append(lines, fmt.Sprintf("- Latest reviewed run: `%s`", gateSummary["latest_run_id"]))
		}
		if _, ok := gateSummary["failing_check_count"]; ok {
			lines = append(lines, fmt.Sprintf("- Failing checks: `%v`", gateSummary["failing_check_count"]))
		}
		if _, ok := gateSummary["workflow_exit_code"]; ok {
			lines = append(lines, fmt.Sprintf("- Workflow exit code on current evidence: `%v`", gateSummary["workflow_exit_code"]))
		}
		reviewerPath, _ := continuationGate["reviewer_path"].(map[string]any)
		if stringValue(reviewerPath["digest_path"]) != "" {
			lines = append(lines, fmt.Sprintf("- Reviewer digest: `%s`", reviewerPath["digest_path"]))
		}
		if stringValue(reviewerPath["index_path"]) != "" {
			lines = append(lines, fmt.Sprintf("- Reviewer index: `%s`", reviewerPath["index_path"]))
		}
		if actions, ok := continuationGate["next_actions"].([]any); ok {
			for _, action := range actions {
				lines = append(lines, fmt.Sprintf("- Next action: `%s`", action))
			}
		}
		lines = append(lines, "")
	}
	if len(continuationArtifacts) > 0 {
		lines = append(lines, "## Continuation artifacts", "")
		for _, raw := range continuationArtifacts {
			item, _ := raw.(map[string]any)
			lines = append(lines, fmt.Sprintf("- `%s` %s", item["path"], item["description"]))
		}
		lines = append(lines, "")
	}
	if len(followupDigests) > 0 {
		lines = append(lines, "## Parallel follow-up digests", "")
		for _, raw := range followupDigests {
			item, _ := raw.(map[string]any)
			lines = append(lines, fmt.Sprintf("- `%s` %s", item["path"], item["description"]))
		}
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func e2eCollectReportEvents(report map[string]any) []map[string]any {
	events := make([]map[string]any, 0)
	status, _ := report["status"].(map[string]any)
	if statusEvents, ok := status["events"].([]any); ok {
		for _, raw := range statusEvents {
			if event, ok := raw.(map[string]any); ok {
				events = append(events, event)
			}
		}
	}
	if latestEvent, ok := status["latest_event"].(map[string]any); ok {
		latestID := stringValue(latestEvent["id"])
		seen := false
		for _, event := range events {
			if latestID != "" && stringValue(event["id"]) == latestID {
				seen = true
				break
			}
		}
		if !seen {
			events = append(events, latestEvent)
		}
	}
	if reportEvents, ok := report["events"].([]any); ok {
		for _, raw := range reportEvents {
			event, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			eventID := stringValue(event["id"])
			seen := false
			for _, existing := range events {
				if eventID != "" && stringValue(existing["id"]) == eventID {
					seen = true
					break
				}
			}
			if !seen {
				events = append(events, event)
			}
		}
	}
	return events
}

func e2eLatestReportEvent(report map[string]any) map[string]any {
	events := e2eCollectReportEvents(report)
	if len(events) == 0 {
		return nil
	}
	return events[len(events)-1]
}

func e2eFindRoutingReason(report map[string]any) string {
	events := e2eCollectReportEvents(report)
	for i := len(events) - 1; i >= 0; i-- {
		if firstNonEmpty(events[i]["type"]) == "scheduler.routed" {
			return e2eEventPayloadText(events[i], "reason")
		}
	}
	return ""
}

func e2eEventPayloadText(event map[string]any, key string) string {
	payload, _ := event["payload"].(map[string]any)
	return e2eFirstText(payload[key])
}

func e2eComponentStatus(report map[string]any) string {
	if len(report) == 0 {
		return "missing_report"
	}
	switch status := report["status"].(type) {
	case map[string]any:
		return firstNonEmpty(status["state"], "unknown")
	case string:
		return status
	default:
		if automationBool(report["all_ok"]) {
			return "succeeded"
		}
		if value, ok := report["all_ok"].(bool); ok && !value {
			return "failed"
		}
		return "unknown"
	}
}

func e2eNormalizeEnforcementMode(mode string, legacy bool) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	if normalized == "" {
		if legacy {
			normalized = "fail"
		} else {
			normalized = "hold"
		}
	}
	switch normalized {
	case "review", "hold", "fail":
		return normalized, nil
	default:
		return "", fmt.Errorf("unsupported enforcement mode %q; expected one of review, hold, fail", mode)
	}
}

func e2eBuildEnforcementSummary(recommendation, mode string) map[string]any {
	if recommendation == "go" {
		return map[string]any{"mode": mode, "outcome": "pass", "exit_code": 0}
	}
	if mode == "review" {
		return map[string]any{"mode": mode, "outcome": "review-only", "exit_code": 0}
	}
	if mode == "hold" {
		return map[string]any{"mode": mode, "outcome": "hold", "exit_code": 2}
	}
	return map[string]any{"mode": mode, "outcome": "fail", "exit_code": 1}
}

func e2eCheck(name string, passed bool, detail string) map[string]any {
	return map[string]any{"name": name, "passed": passed, "detail": detail}
}

func e2eStatuses(runs []map[string]any) []string {
	out := make([]string, 0, len(runs))
	for _, run := range runs {
		out = append(out, firstNonEmpty(run["status"], "unknown"))
	}
	return out
}

func e2eConsecutiveSuccesses(statuses []any) int {
	count := 0
	for _, status := range statuses {
		if firstNonEmpty(status) == "succeeded" {
			count++
			continue
		}
		break
	}
	return count
}

func e2eBuildArtifactList(root string, items interface{}) []any {
	out := make([]any, 0)
	switch typed := items.(type) {
	case []struct{ Path, Description string }:
		for _, item := range typed {
			if e2eFileExists(e2eResolvePath(root, item.Path)) {
				out = append(out, map[string]any{"path": item.Path, "description": item.Description})
			}
		}
	}
	return out
}

func e2eReadJSONMap(path string) (map[string]any, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil, nil
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func e2eWriteJSON(path string, payload map[string]any) error {
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return e2eWriteText(path, string(body)+"\n")
}

func e2eWriteText(path, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), 0o644)
}

func e2eCopyText(source, destination string) (string, error) {
	if !e2eFileExists(source) {
		return "", nil
	}
	if samePath(source, destination) {
		return destination, nil
	}
	body, err := os.ReadFile(source)
	if err != nil {
		return "", err
	}
	return destination, e2eWriteText(destination, string(body))
}

func e2eCopyJSON(source, destination string) (string, error) {
	if !e2eFileExists(source) {
		return "", nil
	}
	if samePath(source, destination) {
		return destination, nil
	}
	payload, err := e2eReadJSONMap(source)
	if err != nil || len(payload) == 0 {
		return "", err
	}
	return destination, e2eWriteJSON(destination, payload)
}

func e2eResolvePath(root, target string) string {
	if filepath.IsAbs(target) {
		return target
	}
	return filepath.Join(root, target)
}

func e2eResolveEvidencePath(root string, target any) string {
	text := firstNonEmpty(target)
	if filepath.IsAbs(text) {
		return text
	}
	candidate := filepath.Join(root, text)
	if e2eFileExists(candidate) {
		return candidate
	}
	if !strings.HasPrefix(text, "bigclaw-go"+string(filepath.Separator)) {
		alt := filepath.Join(root, "bigclaw-go", text)
		if e2eFileExists(alt) {
			return alt
		}
	}
	return candidate
}

func e2eRelPath(root, target string) string {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return filepath.ToSlash(target)
	}
	return filepath.ToSlash(rel)
}

func e2eUTCISO(moment time.Time) string {
	return moment.UTC().Format(time.RFC3339Nano)
}

func e2eParseTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, strings.ReplaceAll(value, "Z", "+00:00"))
}

func e2eRound(value float64, precision int) float64 {
	pow := 1.0
	for i := 0; i < precision; i++ {
		pow *= 10
	}
	if value >= 0 {
		return float64(int(value*pow+0.5)) / pow
	}
	return float64(int(value*pow-0.5)) / pow
}

func e2eFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info != nil
}

func e2eContains(items []any, target string) bool {
	for _, item := range items {
		if firstNonEmpty(item) == target {
			return true
		}
	}
	return false
}

func e2eFirstText(values ...any) string {
	return firstNonEmpty(values...)
}

func firstNonEmpty(values ...any) string {
	for _, value := range values {
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				return strings.TrimSpace(typed)
			}
		case fmt.Stringer:
			if strings.TrimSpace(typed.String()) != "" {
				return strings.TrimSpace(typed.String())
			}
		}
	}
	return ""
}

func firstNonZero(values ...any) any {
	for _, value := range values {
		switch typed := value.(type) {
		case nil:
		case int:
			if typed != 0 {
				return typed
			}
		case float64:
			if typed != 0 {
				return typed
			}
		case []any:
			if len(typed) > 0 {
				return typed
			}
		default:
			return value
		}
	}
	return values[len(values)-1]
}

func lenMapSlice(value any) int {
	switch typed := value.(type) {
	case []any:
		return len(typed)
	default:
		return 0
	}
}

func nilIfEmpty(value string) any {
	if trim(value) == "" {
		return nil
	}
	return value
}

func samePath(left, right string) bool {
	leftAbs, leftErr := filepath.Abs(left)
	rightAbs, rightErr := filepath.Abs(right)
	return leftErr == nil && rightErr == nil && leftAbs == rightAbs
}

func boolFlagValue(value string) bool {
	parsed, _ := strconv.ParseBool(strings.TrimSpace(value))
	return parsed
}

func e2eBuildArtifactMap(path, description string) map[string]any {
	return map[string]any{"path": path, "description": description}
}

func runAllUsageBanner(writer io.Writer) {
	_, _ = io.WriteString(writer, "")
}
