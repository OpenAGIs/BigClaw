package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	e2eLatestLocalReportPath               = "docs/reports/sqlite-smoke-report.json"
	e2eLatestKubernetesReportPath          = "docs/reports/kubernetes-live-smoke-report.json"
	e2eLatestRayReportPath                 = "docs/reports/ray-live-smoke-report.json"
	e2eBrokerSummaryPath                   = "docs/reports/broker-validation-summary.json"
	e2eBrokerBootstrapSummaryPath          = "docs/reports/broker-bootstrap-review-summary.json"
	e2eBrokerValidationPackPath            = "docs/reports/broker-failover-fault-injection-validation-pack.md"
	e2eSharedQueueReportPath               = "docs/reports/multi-node-shared-queue-report.json"
	e2eSharedQueueSummaryPath              = "docs/reports/shared-queue-companion-summary.json"
	e2eContinuationScorecardPath           = "docs/reports/validation-bundle-continuation-scorecard.json"
	e2eContinuationPolicyGatePath          = "docs/reports/validation-bundle-continuation-policy-gate.json"
	e2eContinuationDigestPath              = "docs/reports/validation-bundle-continuation-digest.md"
	e2eParallelValidationMatrixPath        = "docs/reports/parallel-validation-matrix.md"
	e2eLiveValidationIndexPath             = "docs/reports/live-validation-index.md"
	e2eLiveValidationSummaryPath           = "docs/reports/live-validation-summary.json"
	e2eLiveValidationManifestPath          = "docs/reports/live-validation-index.json"
	e2eValidationBundleScorecardGenerator  = "go run ./cmd/bigclawctl automation e2e validation-bundle-continuation-scorecard"
	e2eValidationBundlePolicyGateGenerator = "go run ./cmd/bigclawctl automation e2e validation-bundle-continuation-policy-gate"
	e2eValidationBundleExportGenerator     = "go run ./cmd/bigclawctl automation e2e export-validation-bundle"
	e2eMigrationShadowCompareGenerator     = "go run ./cmd/bigclawctl automation migration shadow-compare"
	e2eContinuationTicket                  = "BIG-PAR-086-local-prework"
	e2eContinuationPolicyGateTicket        = "OPE-262"
)

var (
	e2eLatestReports = map[string]string{
		"local":      e2eLatestLocalReportPath,
		"kubernetes": e2eLatestKubernetesReportPath,
		"ray":        e2eLatestRayReportPath,
	}
	e2eLaneAliases = map[string]string{
		"local":      "local",
		"kubernetes": "k8s",
		"ray":        "ray",
	}
	e2eFailureEventTypes = map[string]struct{}{
		"task.cancelled":   {},
		"task.dead_letter": {},
		"task.failed":      {},
		"task.retried":     {},
	}
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

type automationValidationBundleContinuationScorecardOptions struct {
	Output                string
	IndexManifestPath     string
	BundleRootPath        string
	SummaryPath           string
	SharedQueueReportPath string
	Now                   func() time.Time
}

type automationValidationBundleContinuationPolicyGateOptions struct {
	ScorecardPath               string
	Output                      string
	MaxLatestAgeHours           float64
	MinRecentBundles            int
	RequireRepeatedLaneCoverage bool
	EnforcementMode             string
	LegacyEnforce               bool
	Now                         func() time.Time
}

func runAutomationExportValidationBundleCommand(args []string) error {
	flags := flag.NewFlagSet("automation e2e export-validation-bundle", flag.ContinueOnError)
	goRoot := flags.String("go-root", ".", "bigclaw-go repo root")
	runID := flags.String("run-id", "", "bundle run identifier")
	bundleDir := flags.String("bundle-dir", "", "bundle directory relative to go root")
	summaryPath := flags.String("summary-path", e2eLiveValidationSummaryPath, "summary report path")
	indexPath := flags.String("index-path", e2eLiveValidationIndexPath, "markdown index path")
	manifestPath := flags.String("manifest-path", e2eLiveValidationManifestPath, "manifest path")
	runLocal := flags.String("run-local", "1", "whether local lane ran")
	runKubernetes := flags.String("run-kubernetes", "1", "whether kubernetes lane ran")
	runRay := flags.String("run-ray", "1", "whether ray lane ran")
	validationStatus := flags.Int("validation-status", 0, "overall validation status code")
	runBroker := flags.String("run-broker", "0", "whether broker lane ran")
	brokerBackend := flags.String("broker-backend", "", "broker backend")
	brokerReportPath := flags.String("broker-report-path", "", "broker report path")
	brokerBootstrapSummaryPath := flags.String("broker-bootstrap-summary-path", "", "broker bootstrap summary path")
	localReportPath := flags.String("local-report-path", "", "local report path")
	localStdoutPath := flags.String("local-stdout-path", "", "local stdout path")
	localStderrPath := flags.String("local-stderr-path", "", "local stderr path")
	kubernetesReportPath := flags.String("kubernetes-report-path", "", "kubernetes report path")
	kubernetesStdoutPath := flags.String("kubernetes-stdout-path", "", "kubernetes stdout path")
	kubernetesStderrPath := flags.String("kubernetes-stderr-path", "", "kubernetes stderr path")
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
	required := map[string]string{
		"--local-report-path":      *localReportPath,
		"--local-stdout-path":      *localStdoutPath,
		"--local-stderr-path":      *localStderrPath,
		"--kubernetes-report-path": *kubernetesReportPath,
		"--kubernetes-stdout-path": *kubernetesStdoutPath,
		"--kubernetes-stderr-path": *kubernetesStderrPath,
		"--ray-report-path":        *rayReportPath,
		"--ray-stdout-path":        *rayStdoutPath,
		"--ray-stderr-path":        *rayStderrPath,
	}
	for name, value := range required {
		if trim(value) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	report, exitCode, err := automationExportValidationBundle(automationExportValidationBundleOptions{
		GoRoot:                     absPath(*goRoot),
		RunID:                      *runID,
		BundleDir:                  *bundleDir,
		SummaryPath:                *summaryPath,
		IndexPath:                  *indexPath,
		ManifestPath:               *manifestPath,
		RunLocal:                   *runLocal == "1",
		RunKubernetes:              *runKubernetes == "1",
		RunRay:                     *runRay == "1",
		RunBroker:                  *runBroker == "1",
		BrokerBackend:              trim(*brokerBackend),
		BrokerReportPath:           *brokerReportPath,
		BrokerBootstrapSummaryPath: *brokerBootstrapSummaryPath,
		ValidationStatus:           *validationStatus,
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
	if err != nil {
		return err
	}
	return emit(report, *asJSON, exitCode)
}

func runAutomationValidationBundleContinuationScorecardCommand(args []string) error {
	flags := flag.NewFlagSet("automation e2e validation-bundle-continuation-scorecard", flag.ContinueOnError)
	output := flags.String("output", e2eContinuationScorecardPath, "output path")
	indexManifestPath := flags.String("index-manifest-path", e2eLiveValidationManifestPath, "index manifest path")
	bundleRootPath := flags.String("bundle-root-path", "docs/reports/live-validation-runs", "bundle root path")
	summaryPath := flags.String("summary-path", e2eLiveValidationSummaryPath, "latest summary path")
	sharedQueueReportPath := flags.String("shared-queue-report-path", e2eSharedQueueReportPath, "shared queue report path")
	asJSON := flags.Bool("json", true, "json")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation e2e validation-bundle-continuation-scorecard [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	report, exitCode, err := automationValidationBundleContinuationScorecard(automationValidationBundleContinuationScorecardOptions{
		Output:                *output,
		IndexManifestPath:     *indexManifestPath,
		BundleRootPath:        *bundleRootPath,
		SummaryPath:           *summaryPath,
		SharedQueueReportPath: *sharedQueueReportPath,
	})
	if err != nil {
		return err
	}
	return emit(report, *asJSON, exitCode)
}

func runAutomationValidationBundleContinuationPolicyGateCommand(args []string) error {
	flags := flag.NewFlagSet("automation e2e validation-bundle-continuation-policy-gate", flag.ContinueOnError)
	scorecard := flags.String("scorecard", e2eContinuationScorecardPath, "scorecard path")
	output := flags.String("output", e2eContinuationPolicyGatePath, "output path")
	maxLatestAgeHours := flags.Float64("max-latest-age-hours", 72.0, "latest bundle age threshold")
	minRecentBundles := flags.Int("min-recent-bundles", 2, "minimum recent bundles")
	requireRepeatedLaneCoverage := flags.Bool("require-repeated-lane-coverage", true, "require repeated lane coverage")
	allowPartialLaneHistory := flags.Bool("allow-partial-lane-history", false, "disable repeated lane coverage requirement")
	enforcementMode := flags.String("enforcement-mode", "", "review|hold|fail")
	legacyEnforce := flags.Bool("enforce", false, "legacy alias for fail mode")
	asJSON := flags.Bool("json", true, "json")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation e2e validation-bundle-continuation-policy-gate [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	requireRepeated := *requireRepeatedLaneCoverage && !*allowPartialLaneHistory
	report, exitCode, err := automationValidationBundleContinuationPolicyGate(automationValidationBundleContinuationPolicyGateOptions{
		ScorecardPath:               *scorecard,
		Output:                      *output,
		MaxLatestAgeHours:           *maxLatestAgeHours,
		MinRecentBundles:            *minRecentBundles,
		RequireRepeatedLaneCoverage: requireRepeated,
		EnforcementMode:             *enforcementMode,
		LegacyEnforce:               *legacyEnforce,
	})
	if err != nil {
		return err
	}
	return emit(report, *asJSON, exitCode)
}

func automationExportValidationBundle(opts automationExportValidationBundleOptions) (map[string]any, int, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	root := opts.GoRoot
	bundleDir := e2eResolveInsideRoot(root, opts.BundleDir)
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return nil, 0, err
	}

	summary := map[string]any{
		"run_id":       opts.RunID,
		"generated_at": now().UTC().Format(time.RFC3339Nano),
		"status":       "succeeded",
		"bundle_path":  e2eRelPath(root, bundleDir),
		"closeout_commands": []string{
			"cd bigclaw-go && ./scripts/e2e/run_all.sh",
			"cd bigclaw-go && go test ./...",
			"git push origin <branch> && git log -1 --stat",
		},
	}
	exitCode := 0
	if opts.ValidationStatus != 0 {
		summary["status"] = "failed"
		exitCode = 1
	}

	var err error
	summary["local"], err = e2eBuildComponentSection(root, bundleDir, "local", opts.RunLocal, opts.LocalReportPath, opts.LocalStdoutPath, opts.LocalStderrPath)
	if err != nil {
		return nil, 0, err
	}
	summary["kubernetes"], err = e2eBuildComponentSection(root, bundleDir, "kubernetes", opts.RunKubernetes, opts.KubernetesReportPath, opts.KubernetesStdoutPath, opts.KubernetesStderrPath)
	if err != nil {
		return nil, 0, err
	}
	summary["ray"], err = e2eBuildComponentSection(root, bundleDir, "ray", opts.RunRay, opts.RayReportPath, opts.RayStdoutPath, opts.RayStderrPath)
	if err != nil {
		return nil, 0, err
	}
	summary["broker"], err = e2eBuildBrokerSection(root, bundleDir, opts.RunBroker, opts.BrokerBackend, opts.BrokerBootstrapSummaryPath, opts.BrokerReportPath)
	if err != nil {
		return nil, 0, err
	}
	sharedQueue, err := e2eBuildSharedQueueCompanion(root, bundleDir)
	if err != nil {
		return nil, 0, err
	}
	summary["shared_queue_companion"] = sharedQueue
	summary["validation_matrix"] = e2eBuildValidationMatrix(summary)

	if continuationGate, err := e2eBuildContinuationGateSummary(root); err == nil && continuationGate != nil {
		summary["continuation_gate"] = continuationGate
	}

	bundleSummaryPath := filepath.Join(bundleDir, "summary.json")
	summary["summary_path"] = e2eRelPath(root, bundleSummaryPath)
	if err := e2eWriteJSON(bundleSummaryPath, summary); err != nil {
		return nil, 0, err
	}
	if err := e2eWriteJSON(e2eResolveInsideRoot(root, opts.SummaryPath), summary); err != nil {
		return nil, 0, err
	}

	bundleRoot := filepath.Dir(bundleDir)
	recentRuns, err := e2eBuildRecentRuns(root, bundleRoot, 8)
	if err != nil {
		return nil, 0, err
	}
	manifest := map[string]any{
		"latest":      summary,
		"recent_runs": recentRuns,
	}
	if continuationGate, _ := summary["continuation_gate"].(map[string]any); continuationGate != nil {
		manifest["continuation_gate"] = continuationGate
	}
	if err := e2eWriteJSON(e2eResolveInsideRoot(root, opts.ManifestPath), manifest); err != nil {
		return nil, 0, err
	}

	indexText := e2eRenderIndex(summary, recentRuns, e2eMap(summary["continuation_gate"]), e2eBuildContinuationArtifacts(root), e2eBuildFollowupDigests(root))
	indexPath := e2eResolveInsideRoot(root, opts.IndexPath)
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		return nil, 0, err
	}
	if err := os.WriteFile(indexPath, []byte(indexText), 0o644); err != nil {
		return nil, 0, err
	}
	if err := os.WriteFile(filepath.Join(bundleDir, "README.md"), []byte(indexText), 0o644); err != nil {
		return nil, 0, err
	}
	return summary, exitCode, nil
}

func automationValidationBundleContinuationScorecard(opts automationValidationBundleContinuationScorecardOptions) (map[string]any, int, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	root, err := os.Getwd()
	if err != nil {
		return nil, 0, err
	}
	manifestPath := e2eResolveRepoPath(root, opts.IndexManifestPath)
	summaryPath := e2eResolveRepoPath(root, opts.SummaryPath)
	bundleRoot := e2eResolveRepoPath(root, opts.BundleRootPath)
	sharedQueueReportPath := e2eResolveRepoPath(root, opts.SharedQueueReportPath)

	var manifest map[string]any
	if err := e2eReadJSON(manifestPath, &manifest); err != nil {
		return nil, 0, err
	}
	latestMeta := e2eMap(manifest["latest"])
	recentRunMetas := e2eMapSlice(manifest["recent_runs"])
	recentRuns := make([]map[string]any, 0, len(recentRunMetas))
	recentRunInputs := make([]string, 0, len(recentRunMetas))
	for _, item := range recentRunMetas {
		summaryFile := e2eResolveEvidencePath(root, item["summary_path"])
		var runSummary map[string]any
		if err := e2eReadJSON(summaryFile, &runSummary); err != nil {
			return nil, 0, err
		}
		recentRuns = append(recentRuns, runSummary)
		recentRunInputs = append(recentRunInputs, e2eRelPath(root, summaryFile))
	}

	var latestSummary map[string]any
	if err := e2eReadJSON(summaryPath, &latestSummary); err != nil {
		return nil, 0, err
	}
	var sharedQueue map[string]any
	if err := e2eReadJSON(sharedQueueReportPath, &sharedQueue); err != nil {
		return nil, 0, err
	}
	bundledSharedQueue := e2eMap(latestSummary["shared_queue_companion"])
	laneScorecards := make([]map[string]any, 0, 3)
	for _, lane := range []string{"local", "kubernetes", "ray"} {
		laneScorecards = append(laneScorecards, e2eBuildLaneScorecard(recentRuns, lane))
	}

	latestGeneratedAt, _ := e2eParseTime(e2eString(latestMeta["generated_at"]))
	var previousGeneratedAt time.Time
	if len(recentRuns) > 1 {
		previousGeneratedAt, _ = e2eParseTime(e2eString(recentRuns[1]["generated_at"]))
	}
	currentTime := now().UTC()
	latestAgeHours := currentTime.Sub(latestGeneratedAt).Hours()
	var bundleGapMinutes any
	if !previousGeneratedAt.IsZero() {
		bundleGapMinutes = e2eRound((latestGeneratedAt.Sub(previousGeneratedAt).Minutes()), 2)
	}

	latestLaneStatuses := map[string]string{}
	latestAllSucceeded := true
	for _, lane := range []string{"local", "kubernetes", "ray"} {
		section := e2eMap(latestSummary[lane])
		status := e2eString(section["status"])
		latestLaneStatuses[lane] = status
		if status != "succeeded" {
			latestAllSucceeded = false
		}
	}
	recentAllSucceeded := true
	for _, run := range recentRuns {
		if e2eString(run["status"]) != "succeeded" {
			recentAllSucceeded = false
			break
		}
	}
	repeatedCoverage := true
	enabledRunsByLane := map[string]int{}
	for _, lane := range laneScorecards {
		laneName := e2eString(lane["lane"])
		enabledRuns := e2eInt(lane["enabled_runs"])
		enabledRunsByLane[laneName] = enabledRuns
		if enabledRuns < 2 {
			repeatedCoverage = false
		}
	}

	sharedQueueCompanion := map[string]any{
		"available":                 e2eBoolOr(bundledSharedQueue["available"], e2eBool(sharedQueue["all_ok"])),
		"report_path":               e2eStringOr(bundledSharedQueue["canonical_report_path"], opts.SharedQueueReportPath),
		"summary_path":              e2eStringOr(bundledSharedQueue["canonical_summary_path"], e2eSharedQueueSummaryPath),
		"bundle_report_path":        bundledSharedQueue["bundle_report_path"],
		"bundle_summary_path":       bundledSharedQueue["bundle_summary_path"],
		"cross_node_completions":    e2eIntOr(bundledSharedQueue["cross_node_completions"], e2eInt(sharedQueue["cross_node_completions"])),
		"duplicate_completed_tasks": e2eIntOr(bundledSharedQueue["duplicate_completed_tasks"], len(e2eAnySlice(sharedQueue["duplicate_completed_tasks"]))),
		"duplicate_started_tasks":   e2eIntOr(bundledSharedQueue["duplicate_started_tasks"], len(e2eAnySlice(sharedQueue["duplicate_started_tasks"]))),
		"mode":                      "bundle-companion-summary",
	}
	if len(bundledSharedQueue) == 0 {
		sharedQueueCompanion["mode"] = "standalone-proof"
	}

	continuationChecks := []map[string]any{
		e2eCheck("latest_bundle_all_executor_tracks_succeeded", latestAllSucceeded, fmt.Sprintf("latest lane statuses=%v", latestLaneStatuses)),
		e2eCheck("recent_bundle_chain_has_multiple_runs", len(recentRuns) >= 2, fmt.Sprintf("recent bundle count=%d", len(recentRuns))),
		e2eCheck("recent_bundle_chain_has_no_failures", recentAllSucceeded, fmt.Sprintf("recent bundle statuses=%v", e2eStatuses(recentRuns))),
		e2eCheck("all_executor_tracks_have_repeated_recent_coverage", repeatedCoverage, fmt.Sprintf("enabled_runs_by_lane=%v", enabledRunsByLane)),
		e2eCheck("shared_queue_companion_proof_available", e2eBool(sharedQueueCompanion["available"]), fmt.Sprintf("cross_node_completions=%v", sharedQueueCompanion["cross_node_completions"])),
		e2eCheck("continuation_surface_is_workflow_triggered", true, "run_all closeout now refreshes the scorecard and gate automatically, but continuation still depends on explicit workflow execution instead of an always-on service"),
	}

	currentCeiling := []string{
		"continuation across future validation bundles remains workflow-triggered",
		"shared-queue coordination proof now ships as adjacent bundle metadata rather than an executor-native lane",
		"recent history is bounded to the exported bundle index and not an always-on service",
	}
	if !repeatedCoverage {
		currentCeiling = append(currentCeiling, "not every executor lane is enabled across every indexed bundle in the current recent window")
	}
	nextRuntimeHooks := []string{
		"set BIGCLAW_E2E_CONTINUATION_GATE_MODE=hold or fail in workflow closeout when continuation holds should block or fail the run",
		"decide whether shared-queue coordination should stay as adjacent bundle metadata or gain its own executor-native validation lane",
		"extend the automatic continuation refresh beyond run_all.sh into broader workflow orchestrators",
		"extend the scorecard beyond the latest recent_runs window when more longitudinal evidence exists",
	}

	report := map[string]any{
		"generated_at": e2eUTCISO(currentTime),
		"ticket":       e2eContinuationTicket,
		"title":        "Validation bundle continuation scorecard",
		"status":       "local-continuation-scorecard",
		"evidence_inputs": map[string]any{
			"manifest_path":            opts.IndexManifestPath,
			"latest_summary_path":      opts.SummaryPath,
			"bundle_root":              opts.BundleRootPath,
			"recent_run_summaries":     recentRunInputs,
			"shared_queue_report_path": opts.SharedQueueReportPath,
			"generator_command":        e2eValidationBundleScorecardGenerator,
		},
		"summary": map[string]any{
			"recent_bundle_count":                               len(recentRuns),
			"latest_run_id":                                     e2eString(latestMeta["run_id"]),
			"latest_status":                                     e2eString(latestMeta["status"]),
			"latest_bundle_age_hours":                           e2eRound(latestAgeHours, 2),
			"latest_all_executor_tracks_succeeded":              latestAllSucceeded,
			"recent_bundle_chain_has_no_failures":               recentAllSucceeded,
			"all_executor_tracks_have_repeated_recent_coverage": repeatedCoverage,
			"bundle_gap_minutes":                                bundleGapMinutes,
			"bundle_root_exists":                                e2ePathExists(bundleRoot),
		},
		"executor_lanes":         laneScorecards,
		"shared_queue_companion": sharedQueueCompanion,
		"continuation_checks":    continuationChecks,
		"current_ceiling":        currentCeiling,
		"next_runtime_hooks":     nextRuntimeHooks,
	}
	outputPath := e2eResolveRepoPath(root, opts.Output)
	if err := e2eWriteJSON(outputPath, report); err != nil {
		return nil, 0, err
	}
	return report, 0, nil
}

func automationValidationBundleContinuationPolicyGate(opts automationValidationBundleContinuationPolicyGateOptions) (map[string]any, int, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	root, err := os.Getwd()
	if err != nil {
		return nil, 0, err
	}
	scorecardPath := e2eResolveRepoPath(root, opts.ScorecardPath)
	var scorecard map[string]any
	if err := e2eReadJSON(scorecardPath, &scorecard); err != nil {
		return nil, 0, err
	}
	summary := e2eMap(scorecard["summary"])
	sharedQueue := e2eMap(scorecard["shared_queue_companion"])
	mode, err := e2eNormalizeEnforcementMode(opts.EnforcementMode, opts.LegacyEnforce)
	if err != nil {
		return nil, 0, err
	}
	checks := []map[string]any{
		e2eCheck("latest_bundle_age_within_threshold", e2eFloat(summary["latest_bundle_age_hours"]) <= opts.MaxLatestAgeHours, fmt.Sprintf("latest_bundle_age_hours=%v threshold=%v", summary["latest_bundle_age_hours"], opts.MaxLatestAgeHours)),
		e2eCheck("recent_bundle_count_meets_floor", e2eInt(summary["recent_bundle_count"]) >= opts.MinRecentBundles, fmt.Sprintf("recent_bundle_count=%v floor=%d", summary["recent_bundle_count"], opts.MinRecentBundles)),
		e2eCheck("latest_bundle_all_executor_tracks_succeeded", e2eBool(summary["latest_all_executor_tracks_succeeded"]), fmt.Sprintf("latest_all_executor_tracks_succeeded=%v", summary["latest_all_executor_tracks_succeeded"])),
		e2eCheck("recent_bundle_chain_has_no_failures", e2eBool(summary["recent_bundle_chain_has_no_failures"]), fmt.Sprintf("recent_bundle_chain_has_no_failures=%v", summary["recent_bundle_chain_has_no_failures"])),
		e2eCheck("shared_queue_companion_available", e2eBool(sharedQueue["available"]), fmt.Sprintf("cross_node_completions=%v", sharedQueue["cross_node_completions"])),
		e2eCheck("repeated_lane_coverage_meets_policy", !opts.RequireRepeatedLaneCoverage || e2eBool(summary["all_executor_tracks_have_repeated_recent_coverage"]), fmt.Sprintf("require_repeated_lane_coverage=%v actual=%v", opts.RequireRepeatedLaneCoverage, summary["all_executor_tracks_have_repeated_recent_coverage"])),
	}
	failingChecks := make([]string, 0)
	passingChecks := 0
	for _, item := range checks {
		if e2eBool(item["passed"]) {
			passingChecks++
			continue
		}
		failingChecks = append(failingChecks, e2eString(item["name"]))
	}
	recommendation := "go"
	if len(failingChecks) > 0 {
		recommendation = "hold"
	}
	enforcement := e2eBuildEnforcementSummary(recommendation, mode)
	nextActions := make([]string, 0)
	if e2eContains(failingChecks, "latest_bundle_age_within_threshold") {
		nextActions = append(nextActions, "rerun `cd bigclaw-go && ./scripts/e2e/run_all.sh` to refresh the latest validation bundle")
	}
	if e2eContains(failingChecks, "recent_bundle_count_meets_floor") {
		nextActions = append(nextActions, "export additional validation bundles so the continuation window spans multiple indexed runs")
	}
	if e2eContains(failingChecks, "shared_queue_companion_available") {
		nextActions = append(nextActions, "rerun `python3 scripts/e2e/multi_node_shared_queue.py --report-path docs/reports/multi-node-shared-queue-report.json`")
	}
	if e2eContains(failingChecks, "repeated_lane_coverage_meets_policy") {
		nextActions = append(nextActions, "refresh another full validation bundle with `ray` enabled so each executor lane has repeated indexed coverage")
	}
	if len(nextActions) == 0 {
		nextActions = append(nextActions, "set BIGCLAW_E2E_CONTINUATION_GATE_MODE=fail when workflow closeout should stop on continuation regressions")
	}

	report := map[string]any{
		"generated_at":   e2eUTCISO(now().UTC()),
		"ticket":         e2eContinuationPolicyGateTicket,
		"title":          "Validation workflow continuation gate",
		"status":         map[bool]string{true: "policy-go", false: "policy-hold"}[recommendation == "go"],
		"recommendation": recommendation,
		"evidence_inputs": map[string]any{
			"scorecard_path":    opts.ScorecardPath,
			"generator_command": e2eValidationBundlePolicyGateGenerator,
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
			"passing_check_count":                               passingChecks,
			"failing_check_count":                               len(failingChecks),
		},
		"policy_checks":  checks,
		"failing_checks": failingChecks,
		"reviewer_path": map[string]any{
			"index_path":  e2eLiveValidationIndexPath,
			"digest_path": e2eContinuationDigestPath,
			"digest_issue": map[string]any{
				"id":   "OPE-271",
				"slug": "BIG-PAR-082",
			},
		},
		"shared_queue_companion": sharedQueue,
		"next_actions":           nextActions,
	}
	outputPath := e2eResolveRepoPath(root, opts.Output)
	if err := e2eWriteJSON(outputPath, report); err != nil {
		return nil, 0, err
	}
	return report, e2eInt(enforcement["exit_code"]), nil
}

func e2eBuildComponentSection(root string, bundleDir string, name string, enabled bool, reportPath string, stdoutPath string, stderrPath string) (map[string]any, error) {
	reportAbs := e2eResolveInsideRoot(root, reportPath)
	section := map[string]any{
		"enabled":               enabled,
		"bundle_report_path":    e2eRelPath(root, reportAbs),
		"canonical_report_path": e2eLatestReports[name],
	}
	if !enabled {
		section["status"] = "skipped"
		return section, nil
	}

	var report map[string]any
	if err := e2eReadJSONMaybe(reportAbs, &report); err != nil {
		return nil, err
	}
	section["report"] = report
	section["status"] = e2eComponentStatus(report)
	if copied, err := e2eCopyJSON(reportAbs, e2eResolveInsideRoot(root, e2eLatestReports[name])); err == nil && copied != "" {
		section["canonical_report_path"] = e2eRelPath(root, copied)
	} else if err != nil {
		return nil, err
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
	if len(report) == 0 {
		section["failure_root_cause"] = map[string]any{
			"status":     "missing_report",
			"event_type": "",
			"message":    "",
			"location":   section["bundle_report_path"],
			"event_id":   "",
			"timestamp":  "",
		}
		section["validation_matrix"] = e2eBuildValidationMatrixEntry(name, section, nil)
		return section, nil
	}

	task := e2eMap(report["task"])
	if taskID := e2eString(task["id"]); taskID != "" {
		section["task_id"] = taskID
	}
	if baseURL := e2eString(report["base_url"]); baseURL != "" {
		section["base_url"] = baseURL
	}
	if stateDir := e2eString(report["state_dir"]); stateDir != "" {
		section["state_dir"] = stateDir
		if copied, err := e2eCopyText(filepath.Join(stateDir, "audit.jsonl"), filepath.Join(bundleDir, name+".audit.jsonl")); err != nil {
			return nil, err
		} else if copied != "" {
			section["audit_log_path"] = e2eRelPath(root, copied)
		}
	}
	if serviceLog := e2eString(report["service_log"]); serviceLog != "" {
		if copied, err := e2eCopyText(serviceLog, filepath.Join(bundleDir, name+".service.log")); err != nil {
			return nil, err
		} else if copied != "" {
			section["service_log_path"] = e2eRelPath(root, copied)
		}
	}
	if latestEvent := e2eLatestReportEvent(report); latestEvent != nil {
		section["latest_event_type"] = e2eString(latestEvent["type"])
		section["latest_event_timestamp"] = e2eString(latestEvent["timestamp"])
	}
	if routingReason := e2eFindRoutingReason(report); routingReason != "" {
		section["routing_reason"] = routingReason
	}
	section["failure_root_cause"] = e2eBuildFailureRootCause(section, report)
	section["validation_matrix"] = e2eBuildValidationMatrixEntry(name, section, report)
	return section, nil
}

func e2eBuildBrokerSection(root string, bundleDir string, enabled bool, backend string, bootstrapSummaryPath string, reportPath string) (map[string]any, error) {
	bundleSummaryPath := filepath.Join(bundleDir, "broker-validation-summary.json")
	bundleBootstrapSummaryPath := filepath.Join(bundleDir, "broker-bootstrap-review-summary.json")
	section := map[string]any{
		"enabled":                          enabled,
		"bundle_summary_path":              e2eRelPath(root, bundleSummaryPath),
		"canonical_summary_path":           e2eBrokerSummaryPath,
		"bundle_bootstrap_summary_path":    e2eRelPath(root, bundleBootstrapSummaryPath),
		"canonical_bootstrap_summary_path": e2eBrokerBootstrapSummaryPath,
		"validation_pack_path":             e2eBrokerValidationPackPath,
		"configuration_state":              "not_configured",
	}
	if backend != "" {
		section["backend"] = backend
	}
	if enabled && backend != "" {
		section["configuration_state"] = "configured"
	}
	if trim(bootstrapSummaryPath) != "" {
		bootstrapAbs := e2eResolveInsideRoot(root, bootstrapSummaryPath)
		var bootstrap map[string]any
		if err := e2eReadJSONMaybe(bootstrapAbs, &bootstrap); err != nil {
			return nil, err
		}
		if len(bootstrap) > 0 {
			if copied, err := e2eCopyJSON(bootstrapAbs, bundleBootstrapSummaryPath); err != nil {
				return nil, err
			} else if copied != "" {
				section["bundle_bootstrap_summary_path"] = e2eRelPath(root, copied)
			}
			if copied, err := e2eCopyJSON(bootstrapAbs, e2eResolveInsideRoot(root, e2eBrokerBootstrapSummaryPath)); err != nil {
				return nil, err
			} else if copied != "" {
				section["canonical_bootstrap_summary_path"] = e2eRelPath(root, copied)
			}
			section["bootstrap_summary"] = bootstrap
			section["bootstrap_ready"] = e2eBool(bootstrap["ready"])
			section["runtime_posture"] = bootstrap["runtime_posture"]
			section["live_adapter_implemented"] = e2eBool(bootstrap["live_adapter_implemented"])
			section["proof_boundary"] = bootstrap["proof_boundary"]
			if validationErrors := e2eStringSlice(bootstrap["validation_errors"]); len(validationErrors) > 0 {
				section["validation_errors"] = validationErrors
			}
			if completeness := e2eMap(bootstrap["config_completeness"]); len(completeness) > 0 {
				section["config_completeness"] = completeness
			}
		}
	}
	if !enabled || backend == "" {
		section["status"] = "skipped"
		section["reason"] = "not_configured"
		return section, e2eWriteSummaryPair(root, bundleSummaryPath, e2eBrokerSummaryPath, section)
	}
	if trim(reportPath) == "" {
		section["status"] = "skipped"
		section["reason"] = "missing_report_path"
		return section, e2eWriteSummaryPair(root, bundleSummaryPath, e2eBrokerSummaryPath, section)
	}
	reportAbs := e2eResolveInsideRoot(root, reportPath)
	var report map[string]any
	if err := e2eReadJSONMaybe(reportAbs, &report); err != nil {
		return nil, err
	}
	section["canonical_report_path"] = e2eRelPath(root, reportAbs)
	section["bundle_report_path"] = e2eRelPath(root, filepath.Join(bundleDir, filepath.Base(reportAbs)))
	if len(report) == 0 {
		section["status"] = "skipped"
		section["reason"] = "not_configured"
		return section, e2eWriteSummaryPair(root, bundleSummaryPath, e2eBrokerSummaryPath, section)
	}
	if copied, err := e2eCopyJSON(reportAbs, filepath.Join(bundleDir, filepath.Base(reportAbs))); err != nil {
		return nil, err
	} else if copied != "" {
		section["bundle_report_path"] = e2eRelPath(root, copied)
	}
	section["report"] = report
	section["status"] = e2eComponentStatus(report)
	return section, e2eWriteSummaryPair(root, bundleSummaryPath, e2eBrokerSummaryPath, section)
}

func e2eBuildSharedQueueCompanion(root string, bundleDir string) (map[string]any, error) {
	canonicalReportPath := e2eResolveInsideRoot(root, e2eSharedQueueReportPath)
	canonicalSummaryPath := e2eResolveInsideRoot(root, e2eSharedQueueSummaryPath)
	bundleReportPath := filepath.Join(bundleDir, "multi-node-shared-queue-report.json")
	bundleSummaryPath := filepath.Join(bundleDir, "shared-queue-companion-summary.json")
	var report map[string]any
	if err := e2eReadJSONMaybe(canonicalReportPath, &report); err != nil {
		return nil, err
	}
	summary := map[string]any{
		"available":              len(report) > 0,
		"canonical_report_path":  e2eSharedQueueReportPath,
		"canonical_summary_path": e2eSharedQueueSummaryPath,
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
	summary["status"] = map[bool]string{true: "succeeded", false: "failed"}[e2eBool(report["all_ok"])]
	summary["generated_at"] = report["generated_at"]
	summary["count"] = report["count"]
	summary["cross_node_completions"] = report["cross_node_completions"]
	summary["duplicate_started_tasks"] = len(e2eAnySlice(report["duplicate_started_tasks"]))
	summary["duplicate_completed_tasks"] = len(e2eAnySlice(report["duplicate_completed_tasks"]))
	summary["missing_completed_tasks"] = len(e2eAnySlice(report["missing_completed_tasks"]))
	summary["submitted_by_node"] = e2eMap(report["submitted_by_node"])
	summary["completed_by_node"] = e2eMap(report["completed_by_node"])
	summary["nodes"] = e2eNodeNames(report["nodes"])
	if err := e2eWriteJSON(bundleSummaryPath, summary); err != nil {
		return nil, err
	}
	if err := e2eWriteJSON(canonicalSummaryPath, summary); err != nil {
		return nil, err
	}
	return summary, nil
}

func e2eBuildContinuationGateSummary(root string) (map[string]any, error) {
	gatePath := e2eResolveInsideRoot(root, e2eContinuationPolicyGatePath)
	var gate map[string]any
	if err := e2eReadJSONMaybe(gatePath, &gate); err != nil {
		return nil, err
	}
	if len(gate) == 0 {
		return nil, nil
	}
	return map[string]any{
		"path":           e2eContinuationPolicyGatePath,
		"status":         e2eStringOr(gate["status"], "unknown"),
		"recommendation": e2eStringOr(gate["recommendation"], "unknown"),
		"failing_checks": e2eStringSlice(gate["failing_checks"]),
		"enforcement":    e2eMap(gate["enforcement"]),
		"summary":        e2eMap(gate["summary"]),
		"reviewer_path":  e2eMap(gate["reviewer_path"]),
		"next_actions":   e2eStringSlice(gate["next_actions"]),
	}, nil
}

func e2eBuildRecentRuns(root string, bundleRoot string, limit int) ([]map[string]any, error) {
	entries, err := os.ReadDir(bundleRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	type runMeta struct {
		GeneratedAt string
		Summary     map[string]any
	}
	runs := make([]runMeta, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		summaryPath := filepath.Join(bundleRoot, entry.Name(), "summary.json")
		var summary map[string]any
		if err := e2eReadJSONMaybe(summaryPath, &summary); err != nil {
			return nil, err
		}
		if len(summary) == 0 {
			continue
		}
		runs = append(runs, runMeta{GeneratedAt: e2eString(summary["generated_at"]), Summary: summary})
	}
	sort.Slice(runs, func(i, j int) bool { return runs[i].GeneratedAt > runs[j].GeneratedAt })
	if len(runs) > limit {
		runs = runs[:limit]
	}
	items := make([]map[string]any, 0, len(runs))
	for _, item := range runs {
		summary := item.Summary
		items = append(items, map[string]any{
			"run_id":       e2eString(summary["run_id"]),
			"generated_at": e2eString(summary["generated_at"]),
			"status":       e2eStringOr(summary["status"], "unknown"),
			"bundle_path":  e2eString(summary["bundle_path"]),
			"summary_path": e2eString(summary["summary_path"]),
		})
	}
	return items, nil
}

func e2eBuildValidationMatrix(summary map[string]any) []map[string]any {
	rows := make([]map[string]any, 0, 3)
	for _, name := range []string{"local", "kubernetes", "ray"} {
		section := e2eMap(summary[name])
		if row := e2eMap(section["validation_matrix"]); len(row) > 0 {
			rows = append(rows, row)
		}
	}
	return rows
}

func e2eBuildValidationMatrixEntry(name string, section map[string]any, report map[string]any) map[string]any {
	task := e2eMap(report["task"])
	taskID := e2eString(task["id"])
	if taskID == "" {
		taskID = e2eString(section["task_id"])
	}
	executor := e2eString(task["required_executor"])
	if executor == "" {
		executor = name
	}
	rootCause := e2eMap(section["failure_root_cause"])
	return map[string]any{
		"lane":                  e2eLaneAliases[name],
		"executor":              executor,
		"enabled":               e2eBool(section["enabled"]),
		"status":                e2eStringOr(section["status"], "unknown"),
		"task_id":               taskID,
		"canonical_report_path": e2eString(section["canonical_report_path"]),
		"bundle_report_path":    e2eString(section["bundle_report_path"]),
		"latest_event_type":     e2eString(section["latest_event_type"]),
		"routing_reason":        e2eString(section["routing_reason"]),
		"root_cause_event_type": e2eString(rootCause["event_type"]),
		"root_cause_location":   e2eString(rootCause["location"]),
		"root_cause_message":    e2eString(rootCause["message"]),
	}
}

func e2eBuildFailureRootCause(section map[string]any, report map[string]any) map[string]any {
	events := e2eCollectReportEvents(report)
	latestEvent := e2eLatestReportEvent(report)
	statusMap := e2eMap(report["status"])
	taskMap := e2eMap(report["task"])
	latestStatus := e2eFirstText(statusMap["state"], taskMap["state"], e2eComponentStatus(report))
	var causeEvent map[string]any
	for i := len(events) - 1; i >= 0; i-- {
		eventType := e2eString(events[i]["type"])
		if _, ok := e2eFailureEventTypes[eventType]; ok {
			causeEvent = events[i]
			break
		}
	}
	if causeEvent == nil && latestStatus != "" && latestStatus != "succeeded" {
		causeEvent = latestEvent
	}
	location := e2eFirstText(section["stderr_path"], section["service_log_path"], section["audit_log_path"], section["bundle_report_path"])
	if causeEvent == nil {
		return map[string]any{
			"status":     "not_triggered",
			"event_type": e2eString(latestEvent["type"]),
			"message":    "",
			"location":   location,
			"event_id":   "",
			"timestamp":  "",
		}
	}
	return map[string]any{
		"status":     "captured",
		"event_type": e2eString(causeEvent["type"]),
		"message": e2eFirstText(
			e2eEventPayloadText(causeEvent, "message"),
			e2eEventPayloadText(causeEvent, "reason"),
			report["error"],
			report["failure_reason"],
		),
		"location":  location,
		"event_id":  e2eString(causeEvent["id"]),
		"timestamp": e2eString(causeEvent["timestamp"]),
	}
}

func e2eCollectReportEvents(report map[string]any) []map[string]any {
	events := make([]map[string]any, 0)
	status := e2eMap(report["status"])
	if statusEvents := e2eMapSlice(status["events"]); len(statusEvents) > 0 {
		events = append(events, statusEvents...)
	}
	if latestEvent := e2eMap(status["latest_event"]); len(latestEvent) > 0 {
		latestID := e2eString(latestEvent["id"])
		found := false
		for _, event := range events {
			if e2eString(event["id"]) == latestID && latestID != "" {
				found = true
				break
			}
		}
		if !found {
			events = append(events, latestEvent)
		}
	}
	for _, event := range e2eMapSlice(report["events"]) {
		eventID := e2eString(event["id"])
		duplicate := false
		for _, existing := range events {
			if e2eString(existing["id"]) == eventID && eventID != "" {
				duplicate = true
				break
			}
		}
		if !duplicate {
			events = append(events, event)
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
		if e2eString(events[i]["type"]) == "scheduler.routed" {
			return e2eEventPayloadText(events[i], "reason")
		}
	}
	return ""
}

func e2eBuildLaneScorecard(runs []map[string]any, lane string) map[string]any {
	statuses := make([]string, 0, len(runs))
	enabledRuns := 0
	succeededRuns := 0
	for _, run := range runs {
		section := e2eMap(run[lane])
		enabled := e2eBool(section["enabled"])
		status := "disabled"
		if enabled {
			status = e2eStringOr(section["status"], "missing")
			enabledRuns++
			if status == "succeeded" {
				succeededRuns++
			}
		}
		statuses = append(statuses, status)
	}
	latest := map[string]any{}
	if len(runs) > 0 {
		latest = e2eMap(runs[0][lane])
	}
	return map[string]any{
		"lane":                      lane,
		"latest_enabled":            e2eBool(latest["enabled"]),
		"latest_status":             e2eStringOr(latest["status"], "missing"),
		"recent_statuses":           statuses,
		"enabled_runs":              enabledRuns,
		"succeeded_runs":            succeededRuns,
		"consecutive_successes":     e2eConsecutiveSuccesses(statuses),
		"all_recent_runs_succeeded": enabledRuns > 0 && enabledRuns == succeededRuns,
	}
}

func e2eRenderIndex(summary map[string]any, recentRuns []map[string]any, continuationGate map[string]any, continuationArtifacts [][2]string, followupDigests [][2]string) string {
	lines := []string{
		"# Live Validation Index",
		"",
		fmt.Sprintf("- Latest run: `%s`", e2eString(summary["run_id"])),
		fmt.Sprintf("- Generated at: `%s`", e2eString(summary["generated_at"])),
		fmt.Sprintf("- Status: `%s`", e2eString(summary["status"])),
		fmt.Sprintf("- Bundle: `%s`", e2eString(summary["bundle_path"])),
		fmt.Sprintf("- Summary JSON: `%s`", e2eString(summary["summary_path"])),
		"",
		"## Latest bundle artifacts",
		"",
	}
	for _, name := range []string{"local", "kubernetes", "ray"} {
		section := e2eMap(summary[name])
		matrix := e2eMap(section["validation_matrix"])
		lines = append(lines,
			fmt.Sprintf("### %s", name),
			fmt.Sprintf("- Enabled: `%v`", section["enabled"]),
			fmt.Sprintf("- Status: `%s`", e2eString(section["status"])),
		)
		if lane := e2eString(matrix["lane"]); lane != "" {
			lines = append(lines, fmt.Sprintf("- Validation lane: `%s`", lane))
		}
		lines = append(lines,
			fmt.Sprintf("- Bundle report: `%s`", e2eString(section["bundle_report_path"])),
			fmt.Sprintf("- Latest report: `%s`", e2eString(section["canonical_report_path"])),
		)
		for _, label := range []struct{ Key, Title string }{
			{"stdout_path", "Stdout log"},
			{"stderr_path", "Stderr log"},
			{"service_log_path", "Service log"},
			{"audit_log_path", "Audit log"},
		} {
			if value := e2eString(section[label.Key]); value != "" {
				lines = append(lines, fmt.Sprintf("- %s: `%s`", label.Title, value))
			}
		}
		if taskID := e2eString(section["task_id"]); taskID != "" {
			lines = append(lines, fmt.Sprintf("- Task ID: `%s`", taskID))
		}
		if eventType := e2eString(section["latest_event_type"]); eventType != "" {
			lines = append(lines, fmt.Sprintf("- Latest event: `%s`", eventType))
		}
		if routingReason := e2eString(section["routing_reason"]); routingReason != "" {
			lines = append(lines, fmt.Sprintf("- Routing reason: `%s`", routingReason))
		}
		rootCause := e2eMap(section["failure_root_cause"])
		if len(rootCause) > 0 {
			lines = append(lines, fmt.Sprintf("- Failure root cause: status=`%s` event=`%s` location=`%s`", e2eStringOr(rootCause["status"], "unknown"), e2eStringOr(rootCause["event_type"], "unknown"), e2eStringOr(rootCause["location"], "n/a")))
			if message := e2eString(rootCause["message"]); message != "" {
				lines = append(lines, fmt.Sprintf("- Failure detail: `%s`", message))
			}
		}
		lines = append(lines, "")
	}
	if validationMatrix := e2eMapSlice(summary["validation_matrix"]); len(validationMatrix) > 0 {
		lines = append(lines, "## Validation matrix", "")
		for _, row := range validationMatrix {
			lines = append(lines, fmt.Sprintf("- Lane `%s` executor=`%s` status=`%s` enabled=`%v` report=`%s`", e2eStringOr(row["lane"], "unknown"), e2eStringOr(row["executor"], "unknown"), e2eStringOr(row["status"], "unknown"), row["enabled"], e2eString(row["bundle_report_path"])))
			if e2eString(row["root_cause_event_type"]) != "" || e2eString(row["root_cause_message"]) != "" {
				lines = append(lines, fmt.Sprintf("- Lane `%s` root cause: event=`%s` location=`%s` message=`%s`", e2eStringOr(row["lane"], "unknown"), e2eStringOr(row["root_cause_event_type"], "unknown"), e2eStringOr(row["root_cause_location"], "n/a"), e2eString(row["root_cause_message"])))
			}
		}
		lines = append(lines, "")
	}
	if broker := e2eMap(summary["broker"]); len(broker) > 0 {
		lines = append(lines,
			"### broker",
			fmt.Sprintf("- Enabled: `%v`", broker["enabled"]),
			fmt.Sprintf("- Status: `%s`", e2eString(broker["status"])),
			fmt.Sprintf("- Configuration state: `%s`", e2eString(broker["configuration_state"])),
			fmt.Sprintf("- Bundle summary: `%s`", e2eString(broker["bundle_summary_path"])),
			fmt.Sprintf("- Canonical summary: `%s`", e2eString(broker["canonical_summary_path"])),
			fmt.Sprintf("- Bundle bootstrap summary: `%s`", e2eString(broker["bundle_bootstrap_summary_path"])),
			fmt.Sprintf("- Canonical bootstrap summary: `%s`", e2eString(broker["canonical_bootstrap_summary_path"])),
			fmt.Sprintf("- Validation pack: `%s`", e2eString(broker["validation_pack_path"])),
		)
		if backend := e2eString(broker["backend"]); backend != "" {
			lines = append(lines, fmt.Sprintf("- Backend: `%s`", backend))
		}
		if _, ok := broker["bootstrap_ready"]; ok {
			lines = append(lines, fmt.Sprintf("- Bootstrap ready: `%v`", broker["bootstrap_ready"]))
		}
		if runtimePosture := e2eString(broker["runtime_posture"]); runtimePosture != "" {
			lines = append(lines, fmt.Sprintf("- Runtime posture: `%s`", runtimePosture))
		}
		if _, ok := broker["live_adapter_implemented"]; ok {
			lines = append(lines, fmt.Sprintf("- Live adapter implemented: `%v`", broker["live_adapter_implemented"]))
		}
		if completeness := e2eMap(broker["config_completeness"]); len(completeness) > 0 {
			lines = append(lines, fmt.Sprintf("- Config completeness: driver=`%v` urls=`%v` topic=`%v` consumer_group=`%v`", completeness["driver"], completeness["urls"], completeness["topic"], completeness["consumer_group"]))
		}
		if boundary := e2eString(broker["proof_boundary"]); boundary != "" {
			lines = append(lines, fmt.Sprintf("- Proof boundary: `%s`", boundary))
		}
		for _, errText := range e2eStringSlice(broker["validation_errors"]) {
			lines = append(lines, fmt.Sprintf("- Validation error: `%s`", errText))
		}
		if bundleReport := e2eString(broker["bundle_report_path"]); bundleReport != "" {
			lines = append(lines, fmt.Sprintf("- Bundle report: `%s`", bundleReport))
		}
		if canonicalReport := e2eString(broker["canonical_report_path"]); canonicalReport != "" {
			lines = append(lines, fmt.Sprintf("- Canonical report: `%s`", canonicalReport))
		}
		if reason := e2eString(broker["reason"]); reason != "" {
			lines = append(lines, fmt.Sprintf("- Reason: `%s`", reason))
		}
		lines = append(lines, "")
	}
	if sharedQueue := e2eMap(summary["shared_queue_companion"]); len(sharedQueue) > 0 {
		lines = append(lines,
			"### shared-queue companion",
			fmt.Sprintf("- Available: `%v`", sharedQueue["available"]),
			fmt.Sprintf("- Status: `%s`", e2eString(sharedQueue["status"])),
			fmt.Sprintf("- Bundle summary: `%s`", e2eString(sharedQueue["bundle_summary_path"])),
			fmt.Sprintf("- Canonical summary: `%s`", e2eString(sharedQueue["canonical_summary_path"])),
			fmt.Sprintf("- Bundle report: `%s`", e2eString(sharedQueue["bundle_report_path"])),
			fmt.Sprintf("- Canonical report: `%s`", e2eString(sharedQueue["canonical_report_path"])),
		)
		for _, label := range []struct{ Key, Title string }{
			{"cross_node_completions", "Cross-node completions"},
			{"duplicate_started_tasks", "Duplicate `task.started`"},
			{"duplicate_completed_tasks", "Duplicate `task.completed`"},
			{"missing_completed_tasks", "Missing terminal completions"},
		} {
			if _, ok := sharedQueue[label.Key]; ok {
				lines = append(lines, fmt.Sprintf("- %s: `%v`", label.Title, sharedQueue[label.Key]))
			}
		}
		lines = append(lines, "")
	}
	lines = append(lines, "## Workflow closeout commands", "")
	for _, command := range e2eStringSlice(summary["closeout_commands"]) {
		lines = append(lines, fmt.Sprintf("- `%s`", command))
	}
	lines = append(lines, "", "## Recent bundles", "")
	if len(recentRuns) == 0 {
		lines = append(lines, "- No previous bundles found")
	} else {
		for _, run := range recentRuns {
			lines = append(lines, fmt.Sprintf("- `%s` · `%s` · `%s` · `%s`", e2eString(run["run_id"]), e2eStringOr(run["status"], "unknown"), e2eString(run["generated_at"]), e2eString(run["bundle_path"])))
		}
	}
	lines = append(lines, "")
	if len(continuationGate) > 0 {
		enforcement := e2eMap(continuationGate["enforcement"])
		gateSummary := e2eMap(continuationGate["summary"])
		reviewerPath := e2eMap(continuationGate["reviewer_path"])
		lines = append(lines,
			"## Continuation gate",
			"",
			fmt.Sprintf("- Status: `%s`", e2eString(continuationGate["status"])),
			fmt.Sprintf("- Recommendation: `%s`", e2eString(continuationGate["recommendation"])),
			fmt.Sprintf("- Report: `%s`", e2eString(continuationGate["path"])),
		)
		if mode := e2eString(enforcement["mode"]); mode != "" {
			lines = append(lines, fmt.Sprintf("- Workflow mode: `%s`", mode))
		}
		if outcome := e2eString(enforcement["outcome"]); outcome != "" {
			lines = append(lines, fmt.Sprintf("- Workflow outcome: `%s`", outcome))
		}
		if latestRunID := e2eString(gateSummary["latest_run_id"]); latestRunID != "" {
			lines = append(lines, fmt.Sprintf("- Latest reviewed run: `%s`", latestRunID))
		}
		if _, ok := gateSummary["failing_check_count"]; ok {
			lines = append(lines, fmt.Sprintf("- Failing checks: `%v`", gateSummary["failing_check_count"]))
		}
		if _, ok := gateSummary["workflow_exit_code"]; ok {
			lines = append(lines, fmt.Sprintf("- Workflow exit code on current evidence: `%v`", gateSummary["workflow_exit_code"]))
		}
		if digest := e2eString(reviewerPath["digest_path"]); digest != "" {
			lines = append(lines, fmt.Sprintf("- Reviewer digest: `%s`", digest))
		}
		if index := e2eString(reviewerPath["index_path"]); index != "" {
			lines = append(lines, fmt.Sprintf("- Reviewer index: `%s`", index))
		}
		if digestIssue := e2eMap(reviewerPath["digest_issue"]); len(digestIssue) > 0 {
			lines = append(lines, fmt.Sprintf("- Reviewer digest issue: `%s` / `%s`", e2eString(digestIssue["id"]), e2eString(digestIssue["slug"])))
		}
		lines = append(lines, fmt.Sprintf("- Parallel validation matrix: `%s`", e2eParallelValidationMatrixPath))
		for _, action := range e2eStringSlice(continuationGate["next_actions"]) {
			lines = append(lines, fmt.Sprintf("- Next action: `%s`", action))
		}
		lines = append(lines, "")
	}
	if len(continuationArtifacts) > 0 {
		lines = append(lines, "## Continuation artifacts", "")
		for _, item := range continuationArtifacts {
			lines = append(lines, fmt.Sprintf("- `%s` %s", item[0], item[1]))
		}
		lines = append(lines, "")
	}
	if len(followupDigests) > 0 {
		lines = append(lines, "## Parallel follow-up digests", "")
		for _, item := range followupDigests {
			lines = append(lines, fmt.Sprintf("- `%s` %s", item[0], item[1]))
		}
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func e2eBuildContinuationArtifacts(root string) [][2]string {
	items := [][2]string{}
	for _, item := range [][2]string{
		{e2eContinuationScorecardPath, "summarizes the rolling readiness view across recent bundled local, Kubernetes, and Ray runs plus the shared-queue companion proof."},
		{e2eContinuationPolicyGatePath, "records the current policy decision for bundle freshness, repeated lane coverage, and shared-queue companion availability."},
	} {
		if e2ePathExists(e2eResolveInsideRoot(root, item[0])) {
			items = append(items, item)
		}
	}
	return items
}

func e2eBuildFollowupDigests(root string) [][2]string {
	items := [][2]string{}
	for _, item := range [][2]string{
		{e2eContinuationDigestPath, "Validation bundle continuation caveats are consolidated here."},
	} {
		if e2ePathExists(e2eResolveInsideRoot(root, item[0])) {
			items = append(items, item)
		}
	}
	return items
}

func e2eWriteSummaryPair(root string, bundleSummaryPath string, canonicalSummaryRel string, payload map[string]any) error {
	if err := e2eWriteJSON(bundleSummaryPath, payload); err != nil {
		return err
	}
	return e2eWriteJSON(e2eResolveInsideRoot(root, canonicalSummaryRel), payload)
}

func e2eNormalizeEnforcementMode(mode string, legacyEnforce bool) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	if normalized == "" {
		if legacyEnforce {
			return "fail", nil
		}
		return "hold", nil
	}
	switch normalized {
	case "review", "hold", "fail":
		return normalized, nil
	default:
		return "", fmt.Errorf("unsupported enforcement mode %q; expected one of review, hold, fail", mode)
	}
}

func e2eBuildEnforcementSummary(recommendation string, mode string) map[string]any {
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

func e2eCheck(name string, passed bool, detail string) map[string]any {
	return map[string]any{"name": name, "passed": passed, "detail": detail}
}

func e2eResolveRepoPath(root string, rel string) string {
	if filepath.IsAbs(rel) {
		return rel
	}
	trimmed := filepath.Clean(rel)
	candidate := filepath.Join(root, trimmed)
	if e2ePathExists(candidate) || filepath.Base(root) != "bigclaw-go" {
		return candidate
	}
	if strings.HasPrefix(trimmed, "bigclaw-go"+string(filepath.Separator)) {
		return filepath.Join(filepath.Dir(root), trimmed)
	}
	return candidate
}

func e2eResolveEvidencePath(root string, value any) string {
	path := e2eString(value)
	if filepath.IsAbs(path) {
		return path
	}
	candidate := e2eResolveRepoPath(root, path)
	if e2ePathExists(candidate) {
		return candidate
	}
	if filepath.Base(root) != "bigclaw-go" && !strings.HasPrefix(path, "bigclaw-go"+string(filepath.Separator)) {
		alt := filepath.Join(root, "bigclaw-go", path)
		if e2ePathExists(alt) {
			return alt
		}
	}
	return candidate
}

func e2eResolveInsideRoot(root string, rel string) string {
	if filepath.IsAbs(rel) {
		return rel
	}
	return filepath.Join(root, filepath.Clean(rel))
}

func e2eRelPath(root string, target string) string {
	relative, err := filepath.Rel(root, target)
	if err != nil || strings.HasPrefix(relative, "..") {
		return target
	}
	return filepath.ToSlash(relative)
}

func e2eReadJSON(path string, target any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

func e2eReadJSONMaybe(path string, target any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil
	}
	return json.Unmarshal(body, target)
}

func e2eWriteJSON(path string, payload any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	return os.WriteFile(path, body, 0o644)
}

func e2eCopyJSON(source string, destination string) (string, error) {
	if !e2ePathExists(source) {
		return "", nil
	}
	if filepath.Clean(source) == filepath.Clean(destination) {
		return destination, nil
	}
	var payload any
	if err := e2eReadJSON(source, &payload); err != nil {
		return "", err
	}
	if payload == nil {
		return "", nil
	}
	if err := e2eWriteJSON(destination, payload); err != nil {
		return "", err
	}
	return destination, nil
}

func e2eCopyText(source string, destination string) (string, error) {
	if trim(source) == "" || !e2ePathExists(source) {
		return "", nil
	}
	if filepath.Clean(source) == filepath.Clean(destination) {
		return destination, nil
	}
	body, err := os.ReadFile(source)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(destination, body, 0o644); err != nil {
		return "", err
	}
	return destination, nil
}

func e2ePathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func e2eComponentStatus(report map[string]any) string {
	if len(report) == 0 {
		return "missing_report"
	}
	if status := e2eMap(report["status"]); len(status) > 0 {
		return e2eStringOr(status["state"], "unknown")
	}
	if status, ok := report["status"].(string); ok {
		return status
	}
	if report["all_ok"] == true {
		return "succeeded"
	}
	if report["all_ok"] == false {
		return "failed"
	}
	return "unknown"
}

func e2eEventPayloadText(event map[string]any, key string) string {
	return e2eFirstText(e2eMap(event["payload"])[key])
}

func e2eFirstText(values ...any) string {
	for _, value := range values {
		if text := strings.TrimSpace(e2eString(value)); text != "" {
			return text
		}
	}
	return ""
}

func e2eStatuses(runs []map[string]any) []string {
	out := make([]string, 0, len(runs))
	for _, run := range runs {
		out = append(out, e2eStringOr(run["status"], "unknown"))
	}
	return out
}

func e2eConsecutiveSuccesses(statuses []string) int {
	count := 0
	for _, status := range statuses {
		if status == "succeeded" {
			count++
			continue
		}
		break
	}
	return count
}

func e2eUTCISO(ts time.Time) string {
	return ts.UTC().Format(time.RFC3339)
}

func e2eParseTime(value string) (time.Time, error) {
	normalized := strings.ReplaceAll(value, "Z", "+00:00")
	return time.Parse(time.RFC3339, normalized)
}

func e2eRound(value float64, digits int) float64 {
	scale := 1.0
	for i := 0; i < digits; i++ {
		scale *= 10
	}
	if value >= 0 {
		return float64(int(value*scale+0.5)) / scale
	}
	return float64(int(value*scale-0.5)) / scale
}

func e2eMap(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

func e2eMapSlice(value any) []map[string]any {
	items, ok := value.([]any)
	if !ok {
		if typed, ok := value.([]map[string]any); ok {
			return typed
		}
		return nil
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if entry, ok := item.(map[string]any); ok {
			out = append(out, entry)
		}
	}
	return out
}

func e2eAnySlice(value any) []any {
	if typed, ok := value.([]any); ok {
		return typed
	}
	return nil
}

func e2eStringSlice(value any) []string {
	items, ok := value.([]any)
	if !ok {
		if typed, ok := value.([]string); ok {
			return append([]string(nil), typed...)
		}
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if text := e2eString(item); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func e2eNodeNames(value any) []string {
	names := make([]string, 0)
	for _, item := range e2eMapSlice(value) {
		if name := e2eString(item["name"]); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func e2eString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func e2eStringOr(value any, fallback string) string {
	if text := e2eString(value); text != "" {
		return text
	}
	return fallback
}

func e2eBool(value any) bool {
	typed, ok := value.(bool)
	return ok && typed
}

func e2eBoolOr(value any, fallback bool) bool {
	if typed, ok := value.(bool); ok {
		return typed
	}
	return fallback
}

func e2eInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func e2eIntOr(value any, fallback int) int {
	if current := e2eInt(value); current != 0 {
		return current
	}
	return fallback
}

func e2eFloat(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	default:
		return 0
	}
}

func e2eContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
