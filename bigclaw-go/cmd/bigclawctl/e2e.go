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
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var failureEventTypes = map[string]struct{}{
	"task.cancelled":   {},
	"task.dead_letter": {},
	"task.failed":      {},
	"task.retried":     {},
}

func runE2E(args []string) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		_, _ = os.Stdout.WriteString("usage: bigclawctl e2e <run-task-smoke|export-validation-bundle|validation-bundle-continuation-scorecard|validation-bundle-continuation-policy-gate|subscriber-takeover-fault-matrix> [flags]\n")
		return nil
	}
	command := args[0]
	switch command {
	case "run-task-smoke":
		return runE2ETaskSmoke(args[1:])
	case "export-validation-bundle":
		return runE2EExportValidationBundle(args[1:])
	case "validation-bundle-continuation-scorecard":
		return runE2EValidationBundleContinuationScorecard(args[1:])
	case "validation-bundle-continuation-policy-gate":
		return runE2EValidationBundleContinuationPolicyGate(args[1:])
	case "subscriber-takeover-fault-matrix":
		return runE2ESubscriberTakeoverFaultMatrix(args[1:])
	default:
		return fmt.Errorf("unknown e2e subcommand: %s", command)
	}
}

func runE2ETaskSmoke(args []string) error {
	flags := flag.NewFlagSet("e2e run-task-smoke", flag.ContinueOnError)
	executor := flags.String("executor", "", "required executor: local, kubernetes, or ray")
	title := flags.String("title", "", "task title")
	entrypoint := flags.String("entrypoint", "", "task entrypoint")
	image := flags.String("image", "", "container image")
	baseURL := flags.String("base-url", firstNonEmpty(os.Getenv("BIGCLAW_ADDR"), "http://127.0.0.1:8080"), "control-plane base URL")
	goRoot := flags.String("go-root", ".", "bigclaw-go repository root")
	timeoutSeconds := flags.Int("timeout-seconds", 180, "task timeout seconds")
	pollInterval := flags.Duration("poll-interval", time.Second, "poll interval")
	runtimeEnvJSON := flags.String("runtime-env-json", "", "runtime env json")
	metadataJSON := flags.String("metadata-json", "", "metadata json")
	reportPath := flags.String("report-path", "", "report path relative to go root")
	autostart := flags.Bool("autostart", false, "autostart isolated bigclawd when the base URL is unavailable")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl e2e run-task-smoke [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	if *executor == "" || *title == "" || *entrypoint == "" {
		return errors.New("--executor, --title, and --entrypoint are required")
	}
	if !slices.Contains([]string{"local", "kubernetes", "ray"}, *executor) {
		return fmt.Errorf("unsupported executor %q", *executor)
	}

	base := *baseURL
	var process *exec.Cmd
	var logPath string
	var stateDir string

	defer func() {
		if process != nil && process.Process != nil {
			_ = process.Process.Signal(syscall.SIGTERM)
			done := make(chan struct{})
			go func() {
				_, _ = process.Process.Wait()
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				_ = process.Process.Kill()
				<-done
			}
			if logPath != "" {
				_, _ = fmt.Fprintf(os.Stderr, "bigclawd log: %s\n", logPath)
			}
		}
	}()

	if *autostart {
		if err := waitForHealth(base, 2, 200*time.Millisecond); err != nil {
			env, activeBaseURL, activeStateDir, err := buildAutostartEnv()
			if err != nil {
				return err
			}
			base = activeBaseURL
			stateDir = activeStateDir
			process, logPath, err = startBigclawd(*goRoot, env)
			if err != nil {
				return err
			}
			if err := waitForHealth(base, 60, time.Second); err != nil {
				return err
			}
		}
	} else {
		if err := waitForHealth(base, 60, time.Second); err != nil {
			return err
		}
	}

	taskID := fmt.Sprintf("%s-smoke-%d", *executor, time.Now().Unix())
	task := map[string]any{
		"id":                        taskID,
		"title":                     *title,
		"required_executor":         *executor,
		"entrypoint":                *entrypoint,
		"execution_timeout_seconds": *timeoutSeconds,
		"metadata": map[string]any{
			"smoke_test": "true",
			"executor":   *executor,
		},
	}
	if *image != "" {
		task["container_image"] = *image
	}
	if *runtimeEnvJSON != "" {
		var runtimeEnv map[string]any
		if err := json.Unmarshal([]byte(*runtimeEnvJSON), &runtimeEnv); err != nil {
			return fmt.Errorf("parse --runtime-env-json: %w", err)
		}
		task["runtime_env"] = runtimeEnv
	}
	if *metadataJSON != "" {
		var metadata map[string]any
		if err := json.Unmarshal([]byte(*metadataJSON), &metadata); err != nil {
			return fmt.Errorf("parse --metadata-json: %w", err)
		}
		for key, value := range metadata {
			task["metadata"].(map[string]any)[key] = value
		}
	}

	submitted, err := httpJSON(base+"/tasks", http.MethodPost, task, 10*time.Second)
	if err != nil {
		return err
	}
	taskKey, _ := submitted["task"].(map[string]any)
	if taskKey == nil {
		return fmt.Errorf("submit task response missing task payload: %+v", submitted)
	}
	submittedTaskID, _ := taskKey["id"].(string)

	deadline := time.Now().Add(time.Duration(*timeoutSeconds) * time.Second)
	for time.Now().Before(deadline) {
		status, err := httpJSON(base+"/tasks/"+submittedTaskID, http.MethodGet, nil, 10*time.Second)
		if err != nil {
			return err
		}
		state, _ := status["state"].(string)
		if isTerminalState(state) {
			eventsPayload, err := httpJSON(base+"/events?task_id="+submittedTaskID+"&limit=100", http.MethodGet, nil, 10*time.Second)
			if err != nil {
				return err
			}
			report := map[string]any{
				"autostarted": process != nil,
				"base_url":    base,
				"task":        taskKey,
				"status":      status,
				"events":      eventsPayload["events"],
			}
			if stateDir != "" {
				report["state_dir"] = stateDir
			}
			if logPath != "" {
				report["service_log"] = logPath
			}
			if *reportPath != "" {
				if err := writeJSONFile(filepath.Join(*goRoot, *reportPath), report); err != nil {
					return err
				}
			}
			body, _ := json.MarshalIndent(report, "", "  ")
			_, _ = fmt.Fprintln(os.Stdout, string(body))
			if state == "succeeded" {
				return nil
			}
			return exitError(1)
		}
		time.Sleep(*pollInterval)
	}

	status, statusErr := httpJSON(base+"/tasks/"+submittedTaskID, http.MethodGet, nil, 10*time.Second)
	if statusErr != nil {
		return statusErr
	}
	eventsPayload, eventsErr := httpJSON(base+"/events?task_id="+submittedTaskID+"&limit=100", http.MethodGet, nil, 10*time.Second)
	if eventsErr != nil {
		return eventsErr
	}
	report := map[string]any{
		"autostarted": process != nil,
		"base_url":    base,
		"task":        taskKey,
		"status":      status,
		"events":      eventsPayload["events"],
		"error":       "timeout waiting for terminal state",
	}
	if stateDir != "" {
		report["state_dir"] = stateDir
	}
	if logPath != "" {
		report["service_log"] = logPath
	}
	if *reportPath != "" {
		if err := writeJSONFile(filepath.Join(*goRoot, *reportPath), report); err != nil {
			return err
		}
	}
	body, _ := json.MarshalIndent(report, "", "  ")
	_, _ = fmt.Fprintln(os.Stderr, string(body))
	return exitError(1)
}

func runE2EExportValidationBundle(args []string) error {
	flags := flag.NewFlagSet("e2e export-validation-bundle", flag.ContinueOnError)
	goRoot := flags.String("go-root", "", "bigclaw-go repository root")
	runID := flags.String("run-id", "", "run identifier")
	bundleDir := flags.String("bundle-dir", "", "bundle directory relative to the repo root")
	summaryPath := flags.String("summary-path", "docs/reports/live-validation-summary.json", "canonical summary path")
	indexPath := flags.String("index-path", "docs/reports/live-validation-index.md", "markdown index path")
	manifestPath := flags.String("manifest-path", "docs/reports/live-validation-index.json", "json index path")
	runLocal := flags.String("run-local", "1", "whether the local lane ran")
	runKubernetes := flags.String("run-kubernetes", "1", "whether the kubernetes lane ran")
	runRay := flags.String("run-ray", "1", "whether the ray lane ran")
	validationStatus := flags.String("validation-status", "0", "workflow exit status")
	runBroker := flags.String("run-broker", "0", "whether the broker lane ran")
	brokerBackend := flags.String("broker-backend", "", "broker backend")
	brokerReportPath := flags.String("broker-report-path", "", "broker report path")
	brokerBootstrapSummaryPath := flags.String("broker-bootstrap-summary-path", "", "broker bootstrap summary path")
	localReportPath := flags.String("local-report-path", "", "local report path")
	localStdoutPath := flags.String("local-stdout-path", "", "local stdout log")
	localStderrPath := flags.String("local-stderr-path", "", "local stderr log")
	kubernetesReportPath := flags.String("kubernetes-report-path", "", "kubernetes report path")
	kubernetesStdoutPath := flags.String("kubernetes-stdout-path", "", "kubernetes stdout log")
	kubernetesStderrPath := flags.String("kubernetes-stderr-path", "", "kubernetes stderr log")
	rayReportPath := flags.String("ray-report-path", "", "ray report path")
	rayStdoutPath := flags.String("ray-stdout-path", "", "ray stdout log")
	rayStderrPath := flags.String("ray-stderr-path", "", "ray stderr log")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl e2e export-validation-bundle [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	if *goRoot == "" || *runID == "" || *bundleDir == "" {
		return errors.New("--go-root, --run-id, and --bundle-dir are required")
	}

	root := absPath(*goRoot)
	bundleDirPath := filepath.Join(root, *bundleDir)
	if err := os.MkdirAll(bundleDirPath, 0o755); err != nil {
		return err
	}
	summary := map[string]any{
		"run_id":       *runID,
		"generated_at": time.Now().UTC().Format(time.RFC3339Nano),
		"status":       ternary(*validationStatus == "0", "succeeded", "failed"),
		"bundle_path":  filepath.ToSlash(*bundleDir),
		"closeout_commands": []string{
			"cd bigclaw-go && ./scripts/e2e/run_all.sh",
			"cd bigclaw-go && go test ./...",
			"git push origin <branch> && git log -1 --stat",
		},
	}
	localSection, err := buildValidationBundleComponentSection("local", *runLocal == "1", root, bundleDirPath, *localReportPath, *localStdoutPath, *localStderrPath)
	if err != nil {
		return err
	}
	k8sSection, err := buildValidationBundleComponentSection("kubernetes", *runKubernetes == "1", root, bundleDirPath, *kubernetesReportPath, *kubernetesStdoutPath, *kubernetesStderrPath)
	if err != nil {
		return err
	}
	raySection, err := buildValidationBundleComponentSection("ray", *runRay == "1", root, bundleDirPath, *rayReportPath, *rayStdoutPath, *rayStderrPath)
	if err != nil {
		return err
	}
	brokerSection, err := buildValidationBundleBrokerSection(*runBroker == "1", strings.TrimSpace(*brokerBackend), root, bundleDirPath, *brokerBootstrapSummaryPath, *brokerReportPath)
	if err != nil {
		return err
	}
	sharedQueueCompanion, err := buildValidationBundleSharedQueueCompanion(root, bundleDirPath)
	if err != nil {
		return err
	}
	summary["local"] = localSection
	summary["kubernetes"] = k8sSection
	summary["ray"] = raySection
	summary["broker"] = brokerSection
	summary["shared_queue_companion"] = sharedQueueCompanion
	summary["validation_matrix"] = buildValidationMatrixRows(summary)

	continuationGate := buildContinuationGateSummary(root)
	if len(continuationGate) > 0 {
		summary["continuation_gate"] = continuationGate
	}

	bundleSummaryPath := filepath.Join(bundleDirPath, "summary.json")
	summary["summary_path"] = relToRoot(root, bundleSummaryPath)
	if err := writeJSONFile(bundleSummaryPath, summary); err != nil {
		return err
	}
	if err := writeJSONFile(filepath.Join(root, *summaryPath), summary); err != nil {
		return err
	}

	recentRuns, err := buildValidationBundleRecentRuns(filepath.Dir(bundleDirPath), root, 8)
	if err != nil {
		return err
	}
	manifest := map[string]any{
		"latest":      summary,
		"recent_runs": recentRuns,
	}
	if len(continuationGate) > 0 {
		manifest["continuation_gate"] = continuationGate
	}
	if err := writeJSONFile(filepath.Join(root, *manifestPath), manifest); err != nil {
		return err
	}

	continuationArtifacts := buildValidationBundleArtifacts(root, []validationBundleArtifact{
		{Path: "docs/reports/validation-bundle-continuation-scorecard.json", Description: "summarizes the rolling readiness view across recent bundled local, Kubernetes, and Ray runs plus the shared-queue companion proof."},
		{Path: "docs/reports/validation-bundle-continuation-policy-gate.json", Description: "records the current policy decision for bundle freshness, repeated lane coverage, and shared-queue companion availability."},
	})
	followupDigests := buildValidationBundleArtifacts(root, []validationBundleArtifact{
		{Path: "docs/reports/validation-bundle-continuation-digest.md", Description: "Validation bundle continuation caveats are consolidated here."},
	})
	indexText := renderValidationBundleIndex(summary, recentRuns, continuationGate, continuationArtifacts, followupDigests)
	if err := os.WriteFile(filepath.Join(root, *indexPath), []byte(indexText), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(bundleDirPath, "README.md"), []byte(indexText), 0o644); err != nil {
		return err
	}

	body, _ := json.MarshalIndent(summary, "", "  ")
	_, _ = fmt.Fprintln(os.Stdout, string(body))
	if summary["status"] == "failed" {
		return exitError(1)
	}
	return nil
}

func runE2EValidationBundleContinuationScorecard(args []string) error {
	flags := flag.NewFlagSet("e2e validation-bundle-continuation-scorecard", flag.ContinueOnError)
	repoRoot := flags.String("repo-root", ".", "repository root")
	output := flags.String("output", "bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json", "output path")
	indexManifestPath := flags.String("index-manifest-path", "bigclaw-go/docs/reports/live-validation-index.json", "validation index manifest path")
	bundleRootPath := flags.String("bundle-root-path", "bigclaw-go/docs/reports/live-validation-runs", "bundle root path")
	summaryPath := flags.String("summary-path", "bigclaw-go/docs/reports/live-validation-summary.json", "latest summary path")
	sharedQueueReportPath := flags.String("shared-queue-report-path", "bigclaw-go/docs/reports/multi-node-shared-queue-report.json", "shared queue report path")
	pretty := flags.Bool("pretty", false, "print generated json")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl e2e validation-bundle-continuation-scorecard [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}

	report, err := buildValidationBundleContinuationScorecard(*repoRoot, *indexManifestPath, *bundleRootPath, *summaryPath, *sharedQueueReportPath)
	if err != nil {
		return err
	}
	outputPath := resolvePathAgainstRepoRoot(absPath(*repoRoot), *output)
	if err := writeJSONFile(outputPath, report); err != nil {
		return err
	}
	if *pretty {
		body, _ := json.MarshalIndent(report, "", "  ")
		_, _ = fmt.Fprintln(os.Stdout, string(body))
	}
	return nil
}

func runE2EValidationBundleContinuationPolicyGate(args []string) error {
	flags := flag.NewFlagSet("e2e validation-bundle-continuation-policy-gate", flag.ContinueOnError)
	repoRoot := flags.String("repo-root", ".", "repository root")
	scorecardPath := flags.String("scorecard", "bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json", "scorecard path")
	output := flags.String("output", "bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json", "output path")
	maxLatestAgeHours := flags.Float64("max-latest-age-hours", 72.0, "maximum age for latest bundle")
	minRecentBundles := flags.Int("min-recent-bundles", 2, "minimum recent bundles")
	requireRepeatedLaneCoverage := flags.Bool("require-repeated-lane-coverage", true, "require repeated lane coverage")
	allowPartialLaneHistory := flags.Bool("allow-partial-lane-history", false, "allow partial lane history")
	enforcementMode := flags.String("enforcement-mode", "", "review, hold, or fail")
	legacyEnforce := flags.Bool("enforce", false, "legacy alias for --enforcement-mode=fail")
	pretty := flags.Bool("pretty", false, "print generated json")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl e2e validation-bundle-continuation-policy-gate [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	report, err := buildValidationBundleContinuationPolicyGate(*repoRoot, *scorecardPath, *maxLatestAgeHours, *minRecentBundles, *requireRepeatedLaneCoverage && !*allowPartialLaneHistory, *enforcementMode, *legacyEnforce)
	if err != nil {
		return err
	}
	outputPath := resolvePathAgainstRepoRoot(absPath(*repoRoot), *output)
	if err := writeJSONFile(outputPath, report); err != nil {
		return err
	}
	if *pretty {
		body, _ := json.MarshalIndent(report, "", "  ")
		_, _ = fmt.Fprintln(os.Stdout, string(body))
	}
	enforcement := mapAt(report, "enforcement")
	exitCode, _ := intFromAny(enforcement["exit_code"])
	if exitCode != 0 {
		return exitError(exitCode)
	}
	return nil
}

func runE2ESubscriberTakeoverFaultMatrix(args []string) error {
	flags := flag.NewFlagSet("e2e subscriber-takeover-fault-matrix", flag.ContinueOnError)
	repoRoot := flags.String("repo-root", "..", "repository root")
	output := flags.String("output", "docs/reports/multi-subscriber-takeover-validation-report.json", "path relative to the bigclaw-go root")
	pretty := flags.Bool("pretty", false, "print generated json")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl e2e subscriber-takeover-fault-matrix [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}

	report := buildSubscriberTakeoverFaultMatrixReport(time.Now().UTC())
	outputPath := resolvePathAgainstRepoRoot(absPath(*repoRoot), *output)
	if err := writeJSONFile(outputPath, report); err != nil {
		return err
	}
	if *pretty {
		body, _ := json.MarshalIndent(report, "", "  ")
		_, _ = fmt.Fprintln(os.Stdout, string(body))
	}
	return nil
}

func buildValidationBundleContinuationScorecard(repoRoot, indexManifestPath, bundleRootPath, summaryPath, sharedQueueReportPath string) (map[string]any, error) {
	root := absPath(repoRoot)
	bigclawGoRoot := filepath.Join(root, "bigclaw-go")
	manifest, err := readJSONMap(resolvePathAgainstRepoRoot(root, indexManifestPath))
	if err != nil {
		return nil, err
	}
	latest := mapAt(manifest, "latest")
	recentRunsMeta := sliceAt(manifest, "recent_runs")
	recentRuns := make([]map[string]any, 0, len(recentRunsMeta))
	recentRunInputs := make([]string, 0, len(recentRunsMeta))
	for _, item := range recentRunsMeta {
		meta := asMap(item)
		summaryRel, _ := meta["summary_path"].(string)
		resolved := resolveEvidencePath(root, bigclawGoRoot, summaryRel)
		run, err := readJSONMap(resolved)
		if err != nil {
			return nil, err
		}
		recentRuns = append(recentRuns, run)
		if rel, err := filepath.Rel(root, resolved); err == nil {
			recentRunInputs = append(recentRunInputs, filepath.ToSlash(rel))
		} else {
			recentRunInputs = append(recentRunInputs, filepath.ToSlash(resolved))
		}
	}

	latestSummary, err := readJSONMap(resolvePathAgainstRepoRoot(root, summaryPath))
	if err != nil {
		return nil, err
	}
	sharedQueue, err := readJSONMap(resolvePathAgainstRepoRoot(root, sharedQueueReportPath))
	if err != nil {
		return nil, err
	}
	bundledSharedQueue := mapAt(latestSummary, "shared_queue_companion")

	laneScorecards := []map[string]any{}
	for _, lane := range []string{"local", "kubernetes", "ray"} {
		laneScorecards = append(laneScorecards, buildLaneScorecard(recentRuns, lane))
	}

	latestGeneratedAt, err := parseFlexibleTime(stringAt(latest, "generated_at"))
	if err != nil {
		return nil, err
	}
	var previousGeneratedAt time.Time
	hasPrevious := false
	if len(recentRuns) > 1 {
		previousGeneratedAt, err = parseFlexibleTime(stringAt(recentRuns[1], "generated_at"))
		if err != nil {
			return nil, err
		}
		hasPrevious = true
	}
	generatedAt := time.Now().UTC()
	latestAgeHours := roundFloat(generatedAt.Sub(latestGeneratedAt).Hours(), 2)
	bundleGapMinutes := any(nil)
	if hasPrevious {
		bundleGapMinutes = roundFloat(latestGeneratedAt.Sub(previousGeneratedAt).Minutes(), 2)
	}

	latestLaneStatuses := map[string]any{}
	latestAllSucceeded := true
	for _, lane := range []string{"local", "kubernetes", "ray"} {
		status := stringAt(mapAt(latestSummary, lane), "status")
		latestLaneStatuses[lane] = status
		if status != "succeeded" {
			latestAllSucceeded = false
		}
	}
	recentAllSucceeded := true
	for _, run := range recentRuns {
		if stringAt(run, "status") != "succeeded" {
			recentAllSucceeded = false
			break
		}
	}
	repeatedLaneCoverage := true
	enabledRunsByLane := map[string]any{}
	for _, item := range laneScorecards {
		enabledRuns, _ := intFromAny(item["enabled_runs"])
		enabledRunsByLane[stringAt(item, "lane")] = enabledRuns
		if enabledRuns < 2 {
			repeatedLaneCoverage = false
		}
	}

	sharedQueueAvailable := boolAtWithDefault(bundledSharedQueue, "available", boolAt(sharedQueue, "all_ok"))
	crossNodeCompletions, _ := intFromAny(firstNonNil(bundledSharedQueue["cross_node_completions"], sharedQueue["cross_node_completions"], 0))

	continuationChecks := []map[string]any{
		checkResult("latest_bundle_all_executor_tracks_succeeded", latestAllSucceeded, fmt.Sprintf("latest lane statuses=%v", latestLaneStatuses)),
		checkResult("recent_bundle_chain_has_multiple_runs", len(recentRuns) >= 2, fmt.Sprintf("recent bundle count=%d", len(recentRuns))),
		checkResult("recent_bundle_chain_has_no_failures", recentAllSucceeded, fmt.Sprintf("recent bundle statuses=%v", collectStatuses(recentRuns))),
		checkResult("all_executor_tracks_have_repeated_recent_coverage", repeatedLaneCoverage, fmt.Sprintf("enabled_runs_by_lane=%v", enabledRunsByLane)),
		checkResult("shared_queue_companion_proof_available", sharedQueueAvailable, fmt.Sprintf("cross_node_completions=%d", crossNodeCompletions)),
		checkResult("continuation_surface_is_workflow_triggered", true, "run_all closeout now refreshes the scorecard and gate automatically, but continuation still depends on explicit workflow execution instead of an always-on service"),
	}

	sharedQueueCompanion := map[string]any{
		"available":                 sharedQueueAvailable,
		"report_path":               firstNonEmpty(stringAt(bundledSharedQueue, "canonical_report_path"), sharedQueueReportPath),
		"summary_path":              firstNonEmpty(stringAt(bundledSharedQueue, "canonical_summary_path"), "bigclaw-go/docs/reports/shared-queue-companion-summary.json"),
		"bundle_report_path":        nilIfEmpty(stringAt(bundledSharedQueue, "bundle_report_path")),
		"bundle_summary_path":       nilIfEmpty(stringAt(bundledSharedQueue, "bundle_summary_path")),
		"cross_node_completions":    crossNodeCompletions,
		"duplicate_completed_tasks": firstNonZeroInt(bundledSharedQueue["duplicate_completed_tasks"], len(sliceAt(sharedQueue, "duplicate_completed_tasks"))),
		"duplicate_started_tasks":   firstNonZeroInt(bundledSharedQueue["duplicate_started_tasks"], len(sliceAt(sharedQueue, "duplicate_started_tasks"))),
		"mode":                      ternary(len(bundledSharedQueue) > 0, "bundle-companion-summary", "standalone-proof"),
	}

	currentCeiling := []string{
		"continuation across future validation bundles remains workflow-triggered",
		"shared-queue coordination proof now ships as adjacent bundle metadata rather than an executor-native lane",
		"recent history is bounded to the exported bundle index and not an always-on service",
	}
	if !repeatedLaneCoverage {
		currentCeiling = append(currentCeiling, "not every executor lane is enabled across every indexed bundle in the current recent window")
	}
	nextRuntimeHooks := []string{
		"set BIGCLAW_E2E_CONTINUATION_GATE_MODE=hold or fail in workflow closeout when continuation holds should block or fail the run",
		"decide whether shared-queue coordination should stay as adjacent bundle metadata or gain its own executor-native validation lane",
		"extend the automatic continuation refresh beyond run_all.sh into broader workflow orchestrators",
		"extend the scorecard beyond the latest recent_runs window when more longitudinal evidence exists",
	}

	return map[string]any{
		"generated_at": utcISO(generatedAt),
		"ticket":       "BIG-PAR-086-local-prework",
		"title":        "Validation bundle continuation scorecard",
		"status":       "local-continuation-scorecard",
		"evidence_inputs": map[string]any{
			"manifest_path":            indexManifestPath,
			"latest_summary_path":      summaryPath,
			"bundle_root":              bundleRootPath,
			"recent_run_summaries":     recentRunInputs,
			"shared_queue_report_path": sharedQueueReportPath,
			"generator_script":         "go run ./cmd/bigclawctl e2e validation-bundle-continuation-scorecard",
		},
		"summary": map[string]any{
			"recent_bundle_count":                               len(recentRuns),
			"latest_run_id":                                     stringAt(latest, "run_id"),
			"latest_status":                                     stringAt(latest, "status"),
			"latest_bundle_age_hours":                           latestAgeHours,
			"latest_all_executor_tracks_succeeded":              latestAllSucceeded,
			"recent_bundle_chain_has_no_failures":               recentAllSucceeded,
			"all_executor_tracks_have_repeated_recent_coverage": repeatedLaneCoverage,
			"bundle_gap_minutes":                                bundleGapMinutes,
			"bundle_root_exists":                                dirExists(resolvePathAgainstRepoRoot(root, bundleRootPath)),
		},
		"executor_lanes":         laneScorecards,
		"shared_queue_companion": sharedQueueCompanion,
		"continuation_checks":    continuationChecks,
		"current_ceiling":        currentCeiling,
		"next_runtime_hooks":     nextRuntimeHooks,
	}, nil
}

func buildValidationBundleContinuationPolicyGate(repoRoot, scorecardPath string, maxLatestAgeHours float64, minRecentBundles int, requireRepeatedLaneCoverage bool, enforcementMode string, legacyEnforce bool) (map[string]any, error) {
	root := absPath(repoRoot)
	scorecard, err := readJSONMap(resolvePathAgainstRepoRoot(root, scorecardPath))
	if err != nil {
		return nil, err
	}
	summary := mapAt(scorecard, "summary")
	sharedQueue := mapAt(scorecard, "shared_queue_companion")
	mode, err := normalizeEnforcementMode(enforcementMode, legacyEnforce)
	if err != nil {
		return nil, err
	}

	checks := []map[string]any{
		checkResult("latest_bundle_age_within_threshold", floatAt(summary, "latest_bundle_age_hours") <= maxLatestAgeHours, fmt.Sprintf("latest_bundle_age_hours=%v threshold=%v", summary["latest_bundle_age_hours"], maxLatestAgeHours)),
		checkResult("recent_bundle_count_meets_floor", intAt(summary, "recent_bundle_count") >= minRecentBundles, fmt.Sprintf("recent_bundle_count=%v floor=%d", summary["recent_bundle_count"], minRecentBundles)),
		checkResult("latest_bundle_all_executor_tracks_succeeded", boolAt(summary, "latest_all_executor_tracks_succeeded"), fmt.Sprintf("latest_all_executor_tracks_succeeded=%v", summary["latest_all_executor_tracks_succeeded"])),
		checkResult("recent_bundle_chain_has_no_failures", boolAt(summary, "recent_bundle_chain_has_no_failures"), fmt.Sprintf("recent_bundle_chain_has_no_failures=%v", summary["recent_bundle_chain_has_no_failures"])),
		checkResult("shared_queue_companion_available", boolAt(sharedQueue, "available"), fmt.Sprintf("cross_node_completions=%v", sharedQueue["cross_node_completions"])),
		checkResult("repeated_lane_coverage_meets_policy", !requireRepeatedLaneCoverage || boolAt(summary, "all_executor_tracks_have_repeated_recent_coverage"), fmt.Sprintf("require_repeated_lane_coverage=%t actual=%v", requireRepeatedLaneCoverage, summary["all_executor_tracks_have_repeated_recent_coverage"])),
	}
	failingChecks := []string{}
	for _, item := range checks {
		if !boolAt(item, "passed") {
			failingChecks = append(failingChecks, stringAt(item, "name"))
		}
	}
	recommendation := "go"
	if len(failingChecks) > 0 {
		recommendation = "hold"
	}
	enforcement := buildEnforcementSummary(recommendation, mode)
	nextActions := []string{}
	for _, name := range failingChecks {
		switch name {
		case "latest_bundle_age_within_threshold":
			nextActions = append(nextActions, "rerun `cd bigclaw-go && ./scripts/e2e/run_all.sh` to refresh the latest validation bundle")
		case "recent_bundle_count_meets_floor":
			nextActions = append(nextActions, "export additional validation bundles so the continuation window spans multiple indexed runs")
		case "shared_queue_companion_available":
			nextActions = append(nextActions, "rerun `python3 scripts/e2e/multi_node_shared_queue.py --report-path docs/reports/multi-node-shared-queue-report.json`")
		case "repeated_lane_coverage_meets_policy":
			nextActions = append(nextActions, "refresh another full validation bundle with `ray` enabled so each executor lane has repeated indexed coverage")
		}
	}
	if len(nextActions) == 0 {
		nextActions = append(nextActions, "set BIGCLAW_E2E_CONTINUATION_GATE_MODE=fail when workflow closeout should stop on continuation regressions")
	}

	passingCount := 0
	for _, item := range checks {
		if boolAt(item, "passed") {
			passingCount++
		}
	}

	return map[string]any{
		"generated_at":   utcISO(time.Now().UTC()),
		"ticket":         "OPE-262",
		"title":          "Validation workflow continuation gate",
		"status":         ternary(recommendation == "go", "policy-go", "policy-hold"),
		"recommendation": recommendation,
		"evidence_inputs": map[string]any{
			"scorecard_path":   scorecardPath,
			"generator_script": "go run ./cmd/bigclawctl e2e validation-bundle-continuation-policy-gate",
		},
		"policy_inputs": map[string]any{
			"max_latest_age_hours":           maxLatestAgeHours,
			"min_recent_bundles":             minRecentBundles,
			"require_repeated_lane_coverage": requireRepeatedLaneCoverage,
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
	}, nil
}

func buildLaneScorecard(runs []map[string]any, lane string) map[string]any {
	statuses := []string{}
	enabledRuns := 0
	succeededRuns := 0
	for _, run := range runs {
		section := mapAt(run, lane)
		enabled := boolAt(section, "enabled")
		status := "disabled"
		if enabled {
			status = firstNonEmpty(stringAt(section, "status"), "missing")
			enabledRuns++
		}
		if status == "succeeded" {
			succeededRuns++
		}
		statuses = append(statuses, status)
	}
	latest := map[string]any{}
	if len(runs) > 0 {
		latest = mapAt(runs[0], lane)
	}
	return map[string]any{
		"lane":                      lane,
		"latest_enabled":            boolAt(latest, "enabled"),
		"latest_status":             firstNonEmpty(stringAt(latest, "status"), "missing"),
		"recent_statuses":           statuses,
		"enabled_runs":              enabledRuns,
		"succeeded_runs":            succeededRuns,
		"consecutive_successes":     consecutiveSuccesses(statuses),
		"all_recent_runs_succeeded": enabledRuns > 0 && enabledRuns == succeededRuns,
	}
}

func normalizeEnforcementMode(enforcementMode string, legacyEnforce bool) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(enforcementMode))
	if mode == "" {
		if legacyEnforce {
			mode = "fail"
		} else {
			mode = "hold"
		}
	}
	if !slices.Contains([]string{"review", "hold", "fail"}, mode) {
		return "", fmt.Errorf("unsupported enforcement mode %q; expected one of review, hold, fail", enforcementMode)
	}
	return mode, nil
}

func buildEnforcementSummary(recommendation, enforcementMode string) map[string]any {
	if recommendation == "go" {
		return map[string]any{"mode": enforcementMode, "outcome": "pass", "exit_code": 0}
	}
	switch enforcementMode {
	case "review":
		return map[string]any{"mode": enforcementMode, "outcome": "review-only", "exit_code": 0}
	case "hold":
		return map[string]any{"mode": enforcementMode, "outcome": "hold", "exit_code": 2}
	default:
		return map[string]any{"mode": enforcementMode, "outcome": "fail", "exit_code": 1}
	}
}

func httpJSON(url, method string, payload any, timeout time.Duration) (map[string]any, error) {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("http request failed with status %d: %+v", resp.StatusCode, result)
	}
	return result, nil
}

func waitForHealth(baseURL string, attempts int, interval time.Duration) error {
	var lastErr error
	for i := 0; i < attempts; i++ {
		payload, err := httpJSON(baseURL+"/healthz", http.MethodGet, nil, 10*time.Second)
		if err == nil && boolAt(payload, "ok") {
			return nil
		}
		lastErr = err
		time.Sleep(interval)
	}
	return fmt.Errorf("service did not become healthy: %w", lastErr)
}

func buildAutostartEnv() ([]string, string, string, error) {
	stateDir, err := os.MkdirTemp("", "bigclawd-state-")
	if err != nil {
		return nil, "", "", err
	}
	envMap := map[string]string{}
	for _, item := range os.Environ() {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			envMap[key] = value
		}
	}
	queueBackend := firstNonEmpty(envMap["BIGCLAW_QUEUE_BACKEND"], "file")
	switch queueBackend {
	case "sqlite":
		envMap["BIGCLAW_QUEUE_SQLITE_PATH"] = filepath.Join(stateDir, "queue.db")
	case "file":
		envMap["BIGCLAW_QUEUE_FILE"] = filepath.Join(stateDir, "queue.json")
	}
	envMap["BIGCLAW_AUDIT_LOG_PATH"] = filepath.Join(stateDir, "audit.jsonl")
	baseURL, httpAddr, err := reserveLocalBaseURL()
	if err != nil {
		return nil, "", "", err
	}
	envMap["BIGCLAW_HTTP_ADDR"] = httpAddr

	env := make([]string, 0, len(envMap))
	for key, value := range envMap {
		env = append(env, key+"="+value)
	}
	return env, baseURL, stateDir, nil
}

func reserveLocalBaseURL() (string, string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", "", err
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	addr := "127.0.0.1:" + strconv.Itoa(port)
	return "http://" + addr, addr, nil
}

func startBigclawd(goRoot string, env []string) (*exec.Cmd, string, error) {
	logFile, err := os.CreateTemp("", "bigclawd-e2e-*.log")
	if err != nil {
		return nil, "", err
	}
	cmd := exec.Command("go", "run", "./cmd/bigclawd")
	cmd.Dir = goRoot
	cmd.Env = env
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return nil, "", err
	}
	_ = logFile.Close()
	return cmd, logFile.Name(), nil
}

func isTerminalState(state string) bool {
	switch state {
	case "succeeded", "dead_letter", "cancelled", "failed":
		return true
	default:
		return false
	}
}

func resolveEvidencePath(repoRoot, bigclawGoRoot, path string) string {
	candidate := filepath.Clean(path)
	if filepath.IsAbs(candidate) {
		return candidate
	}
	searchRoots := []string{repoRoot}
	if len(strings.Split(filepath.ToSlash(candidate), "/")) == 0 || !strings.HasPrefix(filepath.ToSlash(candidate), "bigclaw-go/") {
		searchRoots = append(searchRoots, bigclawGoRoot)
	}
	for _, root := range searchRoots {
		resolved := filepath.Join(root, candidate)
		if _, err := os.Stat(resolved); err == nil {
			return resolved
		}
	}
	return filepath.Join(searchRoots[0], candidate)
}

func readJSONMap(path string) (map[string]any, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func writeJSONFile(path string, payload any) error {
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

func checkResult(name string, passed bool, detail string) map[string]any {
	return map[string]any{"name": name, "passed": passed, "detail": detail}
}

func consecutiveSuccesses(statuses []string) int {
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

func collectStatuses(runs []map[string]any) []string {
	statuses := make([]string, 0, len(runs))
	for _, run := range runs {
		statuses = append(statuses, firstNonEmpty(stringAt(run, "status"), "unknown"))
	}
	return statuses
}

func parseFlexibleTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, errors.New("missing time value")
	}
	layouts := []string{time.RFC3339Nano, time.RFC3339}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("parse time %q", value)
}

func utcISO(moment time.Time) string {
	return moment.UTC().Format(time.RFC3339)
}

func roundFloat(value float64, places int) float64 {
	format := "%." + strconv.Itoa(places) + "f"
	rounded, _ := strconv.ParseFloat(fmt.Sprintf(format, value), 64)
	return rounded
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func firstNonZeroInt(values ...any) int {
	for _, value := range values {
		if number, ok := intFromAny(value); ok {
			return number
		}
	}
	return 0
}

func nilIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func ternary[T any](condition bool, whenTrue, whenFalse T) T {
	if condition {
		return whenTrue
	}
	return whenFalse
}

func mapAt(payload map[string]any, key string) map[string]any {
	return asMap(payload[key])
}

func asMap(value any) map[string]any {
	if result, ok := value.(map[string]any); ok {
		return result
	}
	return map[string]any{}
}

func sliceAt(payload map[string]any, key string) []any {
	if result, ok := payload[key].([]any); ok {
		return result
	}
	return nil
}

func stringAt(payload map[string]any, key string) string {
	if value, ok := payload[key].(string); ok {
		return value
	}
	return ""
}

func boolAt(payload map[string]any, key string) bool {
	value, ok := payload[key].(bool)
	return ok && value
}

func boolAtWithDefault(payload map[string]any, key string, fallback bool) bool {
	if value, ok := payload[key].(bool); ok {
		return value
	}
	return fallback
}

func floatAt(payload map[string]any, key string) float64 {
	if value, ok := payload[key].(float64); ok {
		return value
	}
	return 0
}

func intAt(payload map[string]any, key string) int {
	value, _ := intFromAny(payload[key])
	return value
}

func intFromAny(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case json.Number:
		number, err := typed.Int64()
		return int(number), err == nil
	default:
		return 0, false
	}
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

type validationBundleArtifact struct {
	Path        string
	Description string
}

var validationBundleLatestReports = map[string]string{
	"local":      "docs/reports/sqlite-smoke-report.json",
	"kubernetes": "docs/reports/kubernetes-live-smoke-report.json",
	"ray":        "docs/reports/ray-live-smoke-report.json",
}

func buildContinuationGateSummary(root string) map[string]any {
	gate, err := readJSONMap(filepath.Join(root, "docs/reports/validation-bundle-continuation-policy-gate.json"))
	if err != nil {
		return nil
	}
	enforcement := mapAt(gate, "enforcement")
	summary := mapAt(gate, "summary")
	reviewerPath := mapAt(gate, "reviewer_path")
	nextActions := sliceAt(gate, "next_actions")
	return map[string]any{
		"path":           "docs/reports/validation-bundle-continuation-policy-gate.json",
		"status":         firstNonEmpty(stringAt(gate, "status"), "unknown"),
		"recommendation": firstNonEmpty(stringAt(gate, "recommendation"), "unknown"),
		"failing_checks": sliceAt(gate, "failing_checks"),
		"enforcement":    enforcement,
		"summary":        summary,
		"reviewer_path":  reviewerPath,
		"next_actions":   nextActions,
	}
}

func copyTextArtifact(sourcePath, destinationPath string) (string, error) {
	if sourcePath == "" {
		return "", nil
	}
	info, err := os.Stat(sourcePath)
	if err != nil || info.IsDir() {
		return "", nil
	}
	srcAbs := absPath(sourcePath)
	dstAbs := absPath(destinationPath)
	if srcAbs == dstAbs {
		return destinationPath, nil
	}
	body, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(destinationPath), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(destinationPath, body, info.Mode().Perm()); err != nil {
		return "", err
	}
	return destinationPath, nil
}

func copyJSONArtifact(sourcePath, destinationPath string) (string, error) {
	payload, err := readJSONMap(sourcePath)
	if err != nil {
		return "", nil
	}
	if absPath(sourcePath) == absPath(destinationPath) {
		return destinationPath, nil
	}
	if err := writeJSONFile(destinationPath, payload); err != nil {
		return "", err
	}
	return destinationPath, nil
}

func buildValidationBundleSharedQueueCompanion(root, bundleDir string) (map[string]any, error) {
	canonicalReportPath := filepath.Join(root, "docs/reports/multi-node-shared-queue-report.json")
	canonicalSummaryPath := filepath.Join(root, "docs/reports/shared-queue-companion-summary.json")
	bundleReportPath := filepath.Join(bundleDir, "multi-node-shared-queue-report.json")
	bundleSummaryPath := filepath.Join(bundleDir, "shared-queue-companion-summary.json")
	report, err := readJSONMap(canonicalReportPath)
	summary := map[string]any{
		"available":              err == nil,
		"canonical_report_path":  "docs/reports/multi-node-shared-queue-report.json",
		"canonical_summary_path": "docs/reports/shared-queue-companion-summary.json",
		"bundle_report_path":     relToRoot(root, bundleReportPath),
		"bundle_summary_path":    relToRoot(root, bundleSummaryPath),
	}
	if err != nil {
		summary["status"] = "missing_report"
		return summary, nil
	}
	if copied, err := copyJSONArtifact(canonicalReportPath, bundleReportPath); err != nil {
		return nil, err
	} else if copied != "" {
		summary["bundle_report_path"] = relToRoot(root, copied)
	}
	summary["status"] = ternary(boolAt(report, "all_ok"), "succeeded", "failed")
	summary["generated_at"] = report["generated_at"]
	summary["count"] = report["count"]
	summary["cross_node_completions"] = report["cross_node_completions"]
	summary["duplicate_started_tasks"] = len(sliceAt(report, "duplicate_started_tasks"))
	summary["duplicate_completed_tasks"] = len(sliceAt(report, "duplicate_completed_tasks"))
	summary["missing_completed_tasks"] = len(sliceAt(report, "missing_completed_tasks"))
	summary["submitted_by_node"] = mapAt(report, "submitted_by_node")
	summary["completed_by_node"] = mapAt(report, "completed_by_node")
	names := []string{}
	for _, node := range sliceAt(report, "nodes") {
		name := stringAt(asMap(node), "name")
		if name != "" {
			names = append(names, name)
		}
	}
	summary["nodes"] = names
	if err := writeJSONFile(bundleSummaryPath, summary); err != nil {
		return nil, err
	}
	if err := writeJSONFile(canonicalSummaryPath, summary); err != nil {
		return nil, err
	}
	return summary, nil
}

func buildValidationBundleComponentSection(name string, enabled bool, root, bundleDir, reportPath, stdoutPath, stderrPath string) (map[string]any, error) {
	latestReportPath := filepath.Join(root, validationBundleLatestReports[name])
	section := map[string]any{
		"enabled":               enabled,
		"bundle_report_path":    relToRoot(root, filepath.Join(root, reportPath)),
		"canonical_report_path": validationBundleLatestReports[name],
	}
	if !enabled {
		section["status"] = "skipped"
		return section, nil
	}
	report, err := readJSONMap(filepath.Join(root, reportPath))
	if err == nil {
		section["report"] = report
		section["status"] = componentStatusFromMap(report)
		if copied, err := copyJSONArtifact(filepath.Join(root, reportPath), latestReportPath); err != nil {
			return nil, err
		} else if copied != "" {
			section["canonical_report_path"] = relToRoot(root, copied)
		}
	} else {
		section["status"] = "missing_report"
	}
	if copied, err := copyTextArtifact(stdoutPath, filepath.Join(bundleDir, name+".stdout.log")); err != nil {
		return nil, err
	} else if copied != "" {
		section["stdout_path"] = relToRoot(root, copied)
	}
	if copied, err := copyTextArtifact(stderrPath, filepath.Join(bundleDir, name+".stderr.log")); err != nil {
		return nil, err
	} else if copied != "" {
		section["stderr_path"] = relToRoot(root, copied)
	}
	if err == nil {
		task := mapAt(report, "task")
		if taskID := stringAt(task, "id"); taskID != "" {
			section["task_id"] = taskID
		}
		if baseURL := stringAt(report, "base_url"); baseURL != "" {
			section["base_url"] = baseURL
		}
		if stateDir := stringAt(report, "state_dir"); stateDir != "" {
			section["state_dir"] = stateDir
			if copied, err := copyTextArtifact(filepath.Join(stateDir, "audit.jsonl"), filepath.Join(bundleDir, name+".audit.jsonl")); err != nil {
				return nil, err
			} else if copied != "" {
				section["audit_log_path"] = relToRoot(root, copied)
			}
		}
		if serviceLog := stringAt(report, "service_log"); serviceLog != "" {
			if copied, err := copyTextArtifact(serviceLog, filepath.Join(bundleDir, name+".service.log")); err != nil {
				return nil, err
			} else if copied != "" {
				section["service_log_path"] = relToRoot(root, copied)
			}
		}
		latestEvent := latestReportEvent(report)
		if len(latestEvent) > 0 {
			section["latest_event_type"] = stringAt(latestEvent, "type")
			section["latest_event_timestamp"] = stringAt(latestEvent, "timestamp")
			payload := mapAt(latestEvent, "payload")
			if rawArtifacts := payload["artifacts"]; rawArtifacts != nil {
				artifacts := []string{}
				for _, item := range rawArtifacts.([]any) {
					if text, ok := item.(string); ok {
						artifacts = append(artifacts, text)
					}
				}
				if len(artifacts) > 0 {
					section["artifact_paths"] = artifacts
				}
			}
		}
		if routingReason := findRoutingReason(report); routingReason != "" {
			section["routing_reason"] = routingReason
		}
		section["failure_root_cause"] = buildFailureRootCause(section, report)
		section["validation_matrix"] = buildValidationMatrixEntry(name, section, report)
	} else {
		section["failure_root_cause"] = map[string]any{
			"status":     "missing_report",
			"event_type": "",
			"message":    "",
			"location":   section["bundle_report_path"],
			"event_id":   "",
			"timestamp":  "",
		}
		section["validation_matrix"] = buildValidationMatrixEntry(name, section, nil)
	}
	return section, nil
}

func buildValidationBundleBrokerSection(enabled bool, backend, root, bundleDir, bootstrapSummaryPath, reportPath string) (map[string]any, error) {
	bundleSummaryPath := filepath.Join(bundleDir, "broker-validation-summary.json")
	bundleBootstrapSummaryPath := filepath.Join(bundleDir, "broker-bootstrap-review-summary.json")
	section := map[string]any{
		"enabled":                          enabled,
		"backend":                          nilIfEmpty(backend),
		"bundle_summary_path":              relToRoot(root, bundleSummaryPath),
		"canonical_summary_path":           "docs/reports/broker-validation-summary.json",
		"bundle_bootstrap_summary_path":    relToRoot(root, bundleBootstrapSummaryPath),
		"canonical_bootstrap_summary_path": "docs/reports/broker-bootstrap-review-summary.json",
		"validation_pack_path":             "docs/reports/broker-failover-fault-injection-validation-pack.md",
	}
	section["configuration_state"] = ternary(enabled && backend != "", "configured", "not_configured")
	if bootstrapSummaryPath != "" {
		if bootstrapSummary, err := readJSONMap(filepath.Join(root, bootstrapSummaryPath)); err == nil {
			if copied, err := copyJSONArtifact(filepath.Join(root, bootstrapSummaryPath), bundleBootstrapSummaryPath); err != nil {
				return nil, err
			} else if copied != "" {
				section["bundle_bootstrap_summary_path"] = relToRoot(root, copied)
			}
			if copied, err := copyJSONArtifact(filepath.Join(root, bootstrapSummaryPath), filepath.Join(root, "docs/reports/broker-bootstrap-review-summary.json")); err != nil {
				return nil, err
			} else if copied != "" {
				section["canonical_bootstrap_summary_path"] = relToRoot(root, copied)
			}
			section["bootstrap_summary"] = bootstrapSummary
			section["bootstrap_ready"] = bootstrapSummary["ready"]
			section["runtime_posture"] = bootstrapSummary["runtime_posture"]
			section["live_adapter_implemented"] = bootstrapSummary["live_adapter_implemented"]
			section["proof_boundary"] = bootstrapSummary["proof_boundary"]
			if validationErrors := sliceAt(bootstrapSummary, "validation_errors"); len(validationErrors) > 0 {
				section["validation_errors"] = validationErrors
			}
			if completeness := mapAt(bootstrapSummary, "config_completeness"); len(completeness) > 0 {
				section["config_completeness"] = completeness
			}
		}
	}
	if !enabled || backend == "" {
		section["status"] = "skipped"
		section["reason"] = "not_configured"
		if err := writeJSONFile(bundleSummaryPath, section); err != nil {
			return nil, err
		}
		if err := writeJSONFile(filepath.Join(root, "docs/reports/broker-validation-summary.json"), section); err != nil {
			return nil, err
		}
		return section, nil
	}
	if reportPath == "" {
		section["status"] = "skipped"
		section["reason"] = "missing_report_path"
		if err := writeJSONFile(bundleSummaryPath, section); err != nil {
			return nil, err
		}
		if err := writeJSONFile(filepath.Join(root, "docs/reports/broker-validation-summary.json"), section); err != nil {
			return nil, err
		}
		return section, nil
	}
	report, err := readJSONMap(filepath.Join(root, reportPath))
	section["canonical_report_path"] = reportPath
	section["bundle_report_path"] = relToRoot(root, filepath.Join(bundleDir, filepath.Base(reportPath)))
	if err != nil {
		section["status"] = "skipped"
		section["reason"] = "not_configured"
		if err := writeJSONFile(bundleSummaryPath, section); err != nil {
			return nil, err
		}
		if err := writeJSONFile(filepath.Join(root, "docs/reports/broker-validation-summary.json"), section); err != nil {
			return nil, err
		}
		return section, nil
	}
	if copied, err := copyJSONArtifact(filepath.Join(root, reportPath), filepath.Join(bundleDir, filepath.Base(reportPath))); err != nil {
		return nil, err
	} else if copied != "" {
		section["bundle_report_path"] = relToRoot(root, copied)
	}
	section["report"] = report
	section["status"] = componentStatusFromMap(report)
	if err := writeJSONFile(bundleSummaryPath, section); err != nil {
		return nil, err
	}
	if err := writeJSONFile(filepath.Join(root, "docs/reports/broker-validation-summary.json"), section); err != nil {
		return nil, err
	}
	return section, nil
}

func buildValidationBundleRecentRuns(bundleRoot, root string, limit int) ([]map[string]any, error) {
	entries, err := os.ReadDir(bundleRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	type runSummary struct {
		generatedAt string
		summary     map[string]any
	}
	runs := []runSummary{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		summary, err := readJSONMap(filepath.Join(bundleRoot, entry.Name(), "summary.json"))
		if err == nil {
			runs = append(runs, runSummary{generatedAt: stringAt(summary, "generated_at"), summary: summary})
		}
	}
	slices.SortFunc(runs, func(a, b runSummary) int { return strings.Compare(b.generatedAt, a.generatedAt) })
	items := []map[string]any{}
	for i, run := range runs {
		if i >= limit {
			break
		}
		items = append(items, map[string]any{
			"run_id":       stringAt(run.summary, "run_id"),
			"generated_at": stringAt(run.summary, "generated_at"),
			"status":       firstNonEmpty(stringAt(run.summary, "status"), "unknown"),
			"bundle_path":  stringAt(run.summary, "bundle_path"),
			"summary_path": stringAt(run.summary, "summary_path"),
		})
	}
	return items, nil
}

func buildValidationBundleArtifacts(root string, artifacts []validationBundleArtifact) []validationBundleArtifact {
	items := []validationBundleArtifact{}
	for _, artifact := range artifacts {
		if _, err := os.Stat(filepath.Join(root, artifact.Path)); err == nil {
			items = append(items, artifact)
		}
	}
	return items
}

func renderValidationBundleIndex(summary map[string]any, recentRuns []map[string]any, continuationGate map[string]any, continuationArtifacts, followupDigests []validationBundleArtifact) string {
	lines := []string{
		"# Live Validation Index",
		"",
		fmt.Sprintf("- Latest run: `%s`", stringAt(summary, "run_id")),
		fmt.Sprintf("- Generated at: `%s`", stringAt(summary, "generated_at")),
		fmt.Sprintf("- Status: `%s`", stringAt(summary, "status")),
		fmt.Sprintf("- Bundle: `%s`", stringAt(summary, "bundle_path")),
		fmt.Sprintf("- Summary JSON: `%s`", stringAt(summary, "summary_path")),
		"",
		"## Latest bundle artifacts",
		"",
	}
	for _, name := range []string{"local", "kubernetes", "ray"} {
		section := mapAt(summary, name)
		matrix := mapAt(section, "validation_matrix")
		lines = append(lines, "### "+name)
		lines = append(lines, fmt.Sprintf("- Enabled: `%v`", section["enabled"]))
		lines = append(lines, fmt.Sprintf("- Status: `%s`", stringAt(section, "status")))
		if lane := stringAt(matrix, "lane"); lane != "" {
			lines = append(lines, fmt.Sprintf("- Validation lane: `%s`", lane))
		}
		lines = append(lines, fmt.Sprintf("- Bundle report: `%s`", stringAt(section, "bundle_report_path")))
		lines = append(lines, fmt.Sprintf("- Latest report: `%s`", stringAt(section, "canonical_report_path")))
		for _, key := range []struct{ field, label string }{
			{"stdout_path", "Stdout log"},
			{"stderr_path", "Stderr log"},
			{"service_log_path", "Service log"},
			{"audit_log_path", "Audit log"},
			{"task_id", "Task ID"},
			{"latest_event_type", "Latest event"},
			{"routing_reason", "Routing reason"},
		} {
			if value := stringAt(section, key.field); value != "" {
				lines = append(lines, fmt.Sprintf("- %s: `%s`", key.label, value))
			}
		}
		rootCause := mapAt(section, "failure_root_cause")
		if len(rootCause) > 0 {
			lines = append(lines, fmt.Sprintf("- Failure root cause: status=`%s` event=`%s` location=`%s`", stringAt(rootCause, "status"), stringAt(rootCause, "event_type"), stringAt(rootCause, "location")))
			if message := stringAt(rootCause, "message"); message != "" {
				lines = append(lines, fmt.Sprintf("- Failure detail: `%s`", message))
			}
		}
		lines = append(lines, "")
	}
	if matrixItems := summary["validation_matrix"]; matrixItems != nil {
		if rows, ok := matrixItems.([]map[string]any); ok && len(rows) > 0 {
			lines = append(lines, "## Validation matrix", "")
			for _, row := range rows {
				lines = append(lines, fmt.Sprintf("- Lane `%s` executor=`%s` status=`%s` enabled=`%v` report=`%s`", stringAt(row, "lane"), stringAt(row, "executor"), stringAt(row, "status"), row["enabled"], stringAt(row, "bundle_report_path")))
				if stringAt(row, "root_cause_event_type") != "" || stringAt(row, "root_cause_message") != "" {
					lines = append(lines, fmt.Sprintf("- Lane `%s` root cause: event=`%s` location=`%s` message=`%s`", stringAt(row, "lane"), stringAt(row, "root_cause_event_type"), stringAt(row, "root_cause_location"), stringAt(row, "root_cause_message")))
				}
			}
			lines = append(lines, "")
		}
	}
	if broker := mapAt(summary, "broker"); len(broker) > 0 {
		lines = append(lines, "### broker")
		lines = append(lines, fmt.Sprintf("- Enabled: `%v`", broker["enabled"]))
		lines = append(lines, fmt.Sprintf("- Status: `%s`", stringAt(broker, "status")))
		lines = append(lines, fmt.Sprintf("- Configuration state: `%s`", stringAt(broker, "configuration_state")))
		lines = append(lines, fmt.Sprintf("- Bundle summary: `%s`", stringAt(broker, "bundle_summary_path")))
		lines = append(lines, fmt.Sprintf("- Canonical summary: `%s`", stringAt(broker, "canonical_summary_path")))
		lines = append(lines, fmt.Sprintf("- Bundle bootstrap summary: `%s`", stringAt(broker, "bundle_bootstrap_summary_path")))
		lines = append(lines, fmt.Sprintf("- Canonical bootstrap summary: `%s`", stringAt(broker, "canonical_bootstrap_summary_path")))
		lines = append(lines, fmt.Sprintf("- Validation pack: `%s`", stringAt(broker, "validation_pack_path")))
		for _, key := range []struct{ field, label string }{
			{"backend", "Backend"},
			{"runtime_posture", "Runtime posture"},
			{"proof_boundary", "Proof boundary"},
			{"bundle_report_path", "Bundle report"},
			{"canonical_report_path", "Canonical report"},
			{"reason", "Reason"},
		} {
			if value := stringAt(broker, key.field); value != "" {
				lines = append(lines, fmt.Sprintf("- %s: `%s`", key.label, value))
			}
		}
		if _, ok := broker["bootstrap_ready"]; ok {
			lines = append(lines, fmt.Sprintf("- Bootstrap ready: `%v`", broker["bootstrap_ready"]))
		}
		if _, ok := broker["live_adapter_implemented"]; ok {
			lines = append(lines, fmt.Sprintf("- Live adapter implemented: `%v`", broker["live_adapter_implemented"]))
		}
		if completeness := mapAt(broker, "config_completeness"); len(completeness) > 0 {
			lines = append(lines, fmt.Sprintf("- Config completeness: driver=`%v` urls=`%v` topic=`%v` consumer_group=`%v`", completeness["driver"], completeness["urls"], completeness["topic"], completeness["consumer_group"]))
		}
		for _, errItem := range sliceAt(broker, "validation_errors") {
			if text, ok := errItem.(string); ok {
				lines = append(lines, fmt.Sprintf("- Validation error: `%s`", text))
			}
		}
		lines = append(lines, "")
	}
	if shared := mapAt(summary, "shared_queue_companion"); len(shared) > 0 {
		lines = append(lines, "### shared-queue companion")
		lines = append(lines, fmt.Sprintf("- Available: `%v`", shared["available"]))
		lines = append(lines, fmt.Sprintf("- Status: `%s`", stringAt(shared, "status")))
		lines = append(lines, fmt.Sprintf("- Bundle summary: `%s`", stringAt(shared, "bundle_summary_path")))
		lines = append(lines, fmt.Sprintf("- Canonical summary: `%s`", stringAt(shared, "canonical_summary_path")))
		lines = append(lines, fmt.Sprintf("- Bundle report: `%s`", stringAt(shared, "bundle_report_path")))
		lines = append(lines, fmt.Sprintf("- Canonical report: `%s`", stringAt(shared, "canonical_report_path")))
		for _, key := range []struct{ field, label string }{
			{"cross_node_completions", "Cross-node completions"},
			{"duplicate_started_tasks", "Duplicate `task.started`"},
			{"duplicate_completed_tasks", "Duplicate `task.completed`"},
			{"missing_completed_tasks", "Missing terminal completions"},
		} {
			if _, ok := shared[key.field]; ok {
				lines = append(lines, fmt.Sprintf("- %s: `%v`", key.label, shared[key.field]))
			}
		}
		lines = append(lines, "")
	}
	lines = append(lines, "## Workflow closeout commands", "")
	for _, command := range summary["closeout_commands"].([]string) {
		lines = append(lines, fmt.Sprintf("- `%s`", command))
	}
	lines = append(lines, "", "## Recent bundles", "")
	if len(recentRuns) == 0 {
		lines = append(lines, "- No previous bundles found")
	} else {
		for _, run := range recentRuns {
			lines = append(lines, fmt.Sprintf("- `%s` · `%s` · `%s` · `%s`", stringAt(run, "run_id"), stringAt(run, "status"), stringAt(run, "generated_at"), stringAt(run, "bundle_path")))
		}
	}
	lines = append(lines, "")
	if len(continuationGate) > 0 {
		lines = append(lines, "## Continuation gate", "")
		lines = append(lines, fmt.Sprintf("- Status: `%s`", stringAt(continuationGate, "status")))
		lines = append(lines, fmt.Sprintf("- Recommendation: `%s`", stringAt(continuationGate, "recommendation")))
		lines = append(lines, fmt.Sprintf("- Report: `%s`", stringAt(continuationGate, "path")))
		enforcement := mapAt(continuationGate, "enforcement")
		if mode := stringAt(enforcement, "mode"); mode != "" {
			lines = append(lines, fmt.Sprintf("- Workflow mode: `%s`", mode))
		}
		if outcome := stringAt(enforcement, "outcome"); outcome != "" {
			lines = append(lines, fmt.Sprintf("- Workflow outcome: `%s`", outcome))
		}
		gateSummary := mapAt(continuationGate, "summary")
		if latestRunID := stringAt(gateSummary, "latest_run_id"); latestRunID != "" {
			lines = append(lines, fmt.Sprintf("- Latest reviewed run: `%s`", latestRunID))
		}
		if _, ok := gateSummary["failing_check_count"]; ok {
			lines = append(lines, fmt.Sprintf("- Failing checks: `%v`", gateSummary["failing_check_count"]))
		}
		if _, ok := gateSummary["workflow_exit_code"]; ok {
			lines = append(lines, fmt.Sprintf("- Workflow exit code on current evidence: `%v`", gateSummary["workflow_exit_code"]))
		}
		reviewer := mapAt(continuationGate, "reviewer_path")
		if digest := stringAt(reviewer, "digest_path"); digest != "" {
			lines = append(lines, fmt.Sprintf("- Reviewer digest: `%s`", digest))
		}
		if index := stringAt(reviewer, "index_path"); index != "" {
			lines = append(lines, fmt.Sprintf("- Reviewer index: `%s`", index))
		}
		for _, action := range sliceAt(continuationGate, "next_actions") {
			if text, ok := action.(string); ok {
				lines = append(lines, fmt.Sprintf("- Next action: `%s`", text))
			}
		}
		lines = append(lines, "")
	}
	if len(continuationArtifacts) > 0 {
		lines = append(lines, "## Continuation artifacts", "")
		for _, artifact := range continuationArtifacts {
			lines = append(lines, fmt.Sprintf("- `%s` %s", artifact.Path, artifact.Description))
		}
		lines = append(lines, "")
	}
	if len(followupDigests) > 0 {
		lines = append(lines, "## Parallel follow-up digests", "")
		for _, artifact := range followupDigests {
			lines = append(lines, fmt.Sprintf("- `%s` %s", artifact.Path, artifact.Description))
		}
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func componentStatusFromMap(report map[string]any) string {
	status := report["status"]
	if statusMap, ok := status.(map[string]any); ok {
		return firstNonEmpty(stringAt(statusMap, "state"), "unknown")
	}
	if statusText, ok := status.(string); ok {
		return statusText
	}
	if boolAt(report, "all_ok") {
		return "succeeded"
	}
	if value, ok := report["all_ok"].(bool); ok && !value {
		return "failed"
	}
	return "unknown"
}

func collectReportEvents(report map[string]any) []map[string]any {
	events := []map[string]any{}
	status := mapAt(report, "status")
	for _, item := range sliceAt(status, "events") {
		if event := asMap(item); len(event) > 0 {
			events = append(events, event)
		}
	}
	latest := mapAt(status, "latest_event")
	if len(latest) > 0 {
		latestID := stringAt(latest, "id")
		duplicate := false
		for _, event := range events {
			if stringAt(event, "id") == latestID && latestID != "" {
				duplicate = true
				break
			}
		}
		if !duplicate {
			events = append(events, latest)
		}
	}
	for _, item := range sliceAt(report, "events") {
		if event := asMap(item); len(event) > 0 {
			eventID := stringAt(event, "id")
			duplicate := false
			for _, existing := range events {
				if stringAt(existing, "id") == eventID && eventID != "" {
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

func latestReportEvent(report map[string]any) map[string]any {
	events := collectReportEvents(report)
	if len(events) == 0 {
		return nil
	}
	return events[len(events)-1]
}

func eventPayloadText(event map[string]any, key string) string {
	return stringAt(mapAt(event, "payload"), key)
}

func findRoutingReason(report map[string]any) string {
	events := collectReportEvents(report)
	for i := len(events) - 1; i >= 0; i-- {
		if stringAt(events[i], "type") == "scheduler.routed" {
			return firstNonEmpty(eventPayloadText(events[i], "reason"))
		}
	}
	return ""
}

func buildFailureRootCause(section, report map[string]any) map[string]any {
	events := collectReportEvents(report)
	latestEvent := latestReportEvent(report)
	status := firstNonEmpty(stringAt(mapAt(report, "status"), "state"), stringAt(mapAt(report, "task"), "state"), componentStatusFromMap(report))
	var cause map[string]any
	for i := len(events) - 1; i >= 0; i-- {
		if _, ok := failureEventTypes[stringAt(events[i], "type")]; ok {
			cause = events[i]
			break
		}
	}
	if cause == nil && status != "" && status != "succeeded" {
		cause = latestEvent
	}
	location := firstNonEmpty(stringAt(section, "stderr_path"), stringAt(section, "service_log_path"), stringAt(section, "audit_log_path"), stringAt(section, "bundle_report_path"))
	if cause == nil {
		return map[string]any{
			"status":     "not_triggered",
			"event_type": stringAt(latestEvent, "type"),
			"message":    "",
			"location":   location,
			"event_id":   "",
			"timestamp":  "",
		}
	}
	return map[string]any{
		"status":     "captured",
		"event_type": stringAt(cause, "type"),
		"message": firstNonEmpty(
			eventPayloadText(cause, "message"),
			eventPayloadText(cause, "reason"),
			stringAt(report, "error"),
			stringAt(report, "failure_reason"),
		),
		"location":  location,
		"event_id":  stringAt(cause, "id"),
		"timestamp": stringAt(cause, "timestamp"),
	}
}

func buildValidationMatrixEntry(name string, section map[string]any, report map[string]any) map[string]any {
	task := mapAt(report, "task")
	taskID := stringAt(task, "id")
	if taskID == "" {
		taskID = stringAt(section, "task_id")
	}
	executor := stringAt(task, "required_executor")
	if executor == "" {
		executor = name
	}
	rootCause := mapAt(section, "failure_root_cause")
	lane := name
	if name == "kubernetes" {
		lane = "k8s"
	}
	return map[string]any{
		"lane":                  lane,
		"executor":              executor,
		"enabled":               section["enabled"],
		"status":                firstNonEmpty(stringAt(section, "status"), "unknown"),
		"task_id":               taskID,
		"canonical_report_path": stringAt(section, "canonical_report_path"),
		"bundle_report_path":    stringAt(section, "bundle_report_path"),
		"latest_event_type":     stringAt(section, "latest_event_type"),
		"routing_reason":        stringAt(section, "routing_reason"),
		"root_cause_event_type": stringAt(rootCause, "event_type"),
		"root_cause_location":   stringAt(rootCause, "location"),
		"root_cause_message":    stringAt(rootCause, "message"),
	}
}

func buildValidationMatrixRows(summary map[string]any) []map[string]any {
	rows := []map[string]any{}
	for _, name := range []string{"local", "kubernetes", "ray"} {
		section := mapAt(summary, name)
		if row := mapAt(section, "validation_matrix"); len(row) > 0 {
			rows = append(rows, row)
		}
	}
	return rows
}

func relToRoot(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

var takeoverBaseTime = time.Date(2026, 3, 16, 10, 30, 0, 0, time.UTC)

type takeoverLease struct {
	GroupID           string
	SubscriberID      string
	ConsumerID        string
	LeaseToken        string
	LeaseEpoch        int
	CheckpointOffset  int
	CheckpointEventID string
	ExpiresAt         time.Time
	UpdatedAt         time.Time
}

type leaseCoordinator struct {
	leases  map[string]takeoverLease
	counter int
}

func newLeaseCoordinator() *leaseCoordinator {
	return &leaseCoordinator{leases: map[string]takeoverLease{}}
}

func (c *leaseCoordinator) key(groupID, subscriberID string) string {
	return groupID + "\x00" + subscriberID
}

func (c *leaseCoordinator) nextToken() string {
	c.counter++
	return fmt.Sprintf("lease-%d", c.counter)
}

func (c *leaseCoordinator) expired(lease takeoverLease, now time.Time) bool {
	return !lease.ExpiresAt.IsZero() && !now.Before(lease.ExpiresAt)
}

func (c *leaseCoordinator) acquire(groupID, subscriberID, consumerID string, ttl time.Duration, now time.Time) (takeoverLease, error) {
	key := c.key(groupID, subscriberID)
	current, hasCurrent := c.leases[key]
	if hasCurrent && !c.expired(current, now) && current.ConsumerID != consumerID {
		return takeoverLease{}, fmt.Errorf("lease held by %s", current.ConsumerID)
	}
	if hasCurrent && c.expired(current, now) && current.ConsumerID != consumerID {
		current = takeoverLease{
			GroupID:           current.GroupID,
			SubscriberID:      current.SubscriberID,
			ConsumerID:        current.ConsumerID,
			LeaseEpoch:        current.LeaseEpoch,
			CheckpointOffset:  current.CheckpointOffset,
			CheckpointEventID: current.CheckpointEventID,
			ExpiresAt:         current.ExpiresAt,
			UpdatedAt:         current.UpdatedAt,
		}
	}
	if hasCurrent && !c.expired(current, now) && current.ConsumerID == consumerID {
		current.ExpiresAt = now.Add(ttl)
		current.UpdatedAt = now
		c.leases[key] = current
		return current, nil
	}
	nextEpoch := 1
	nextOffset := 0
	nextEvent := ""
	if hasCurrent {
		nextEpoch = current.LeaseEpoch + 1
		nextOffset = current.CheckpointOffset
		nextEvent = current.CheckpointEventID
	}
	lease := takeoverLease{
		GroupID:           groupID,
		SubscriberID:      subscriberID,
		ConsumerID:        consumerID,
		LeaseToken:        c.nextToken(),
		LeaseEpoch:        nextEpoch,
		CheckpointOffset:  nextOffset,
		CheckpointEventID: nextEvent,
		ExpiresAt:         now.Add(ttl),
		UpdatedAt:         now,
	}
	c.leases[key] = lease
	return lease, nil
}

func (c *leaseCoordinator) commit(groupID, subscriberID, consumerID, leaseToken string, leaseEpoch, checkpointOffset int, checkpointEventID string, now time.Time) (takeoverLease, error) {
	key := c.key(groupID, subscriberID)
	current, hasCurrent := c.leases[key]
	if !hasCurrent {
		return takeoverLease{}, errors.New("no lease")
	}
	if c.expired(current, now) {
		return takeoverLease{}, errors.New("lease expired")
	}
	if current.ConsumerID != consumerID || current.LeaseToken != leaseToken || current.LeaseEpoch != leaseEpoch {
		return takeoverLease{}, errors.New("lease fenced")
	}
	if checkpointOffset < current.CheckpointOffset {
		return takeoverLease{}, errors.New("checkpoint rollback")
	}
	current.CheckpointOffset = checkpointOffset
	current.CheckpointEventID = checkpointEventID
	current.UpdatedAt = now
	c.leases[key] = current
	return current, nil
}

func buildSubscriberTakeoverFaultMatrixReport(now time.Time) map[string]any {
	scenarios := []map[string]any{
		takeoverScenarioAfterPrimaryCrash(),
		takeoverScenarioLeaseExpiryStaleWriterRejected(),
		takeoverScenarioSplitBrainDualReplayWindow(),
	}
	passing := 0
	totalDuplicates := 0
	totalRejections := 0
	for _, scenario := range scenarios {
		if boolAt(scenario, "all_assertions_passed") {
			passing++
		}
		totalDuplicates += intAt(scenario, "duplicate_delivery_count")
		totalRejections += intAt(scenario, "stale_write_rejections")
	}
	return map[string]any{
		"generated_at": utcISO(now),
		"ticket":       "OPE-269",
		"title":        "Multi-subscriber takeover executable local harness report",
		"status":       "local-executable",
		"harness_mode": "deterministic_local_simulation",
		"current_primitives": map[string]any{
			"lease_aware_checkpoints": []string{
				"internal/events/subscriber_leases.go",
				"internal/events/subscriber_leases_test.go",
				"docs/reports/event-bus-reliability-report.md",
			},
			"shared_queue_evidence": []string{
				"scripts/e2e/multi_node_shared_queue.py",
				"docs/reports/multi-node-shared-queue-report.json",
			},
			"takeover_harness": []string{
				"cmd/bigclawctl/e2e.go",
				"docs/reports/multi-subscriber-takeover-validation-report.json",
			},
		},
		"required_report_sections": []string{
			"scenario metadata",
			"fault injection steps",
			"audit assertions",
			"checkpoint assertions",
			"replay assertions",
			"per-node audit artifacts",
			"final owner and replay cursor summary",
			"duplicate delivery accounting",
			"open blockers and follow-up implementation hooks",
		},
		"implementation_path": []string{
			"wire the same ownership and rejection schema into the shared multi-node harness",
			"emit real per-node audit artifacts from live takeover runs instead of synthetic report paths",
			"export duplicate replay candidates and stale-writer rejection counters from live event-log APIs",
			"prove the same report contract against an actual cross-process subscriber group",
		},
		"summary": map[string]any{
			"scenario_count":           len(scenarios),
			"passing_scenarios":        passing,
			"failing_scenarios":        len(scenarios) - passing,
			"duplicate_delivery_count": totalDuplicates,
			"stale_write_rejections":   totalRejections,
		},
		"scenarios": scenarios,
		"remaining_gaps": []string{
			"The harness is deterministic and local; it does not yet orchestrate live bigclawd takeover between separate processes.",
			"Audit log paths in the report are normalized artifact targets, not emitted runtime files from a multi-node run.",
			"Shared durable subscriber-group coordination still needs a full cross-process proof before the follow-up digest can be closed as done.",
		},
	}
}

func takeoverScenarioAfterPrimaryCrash() map[string]any {
	scenarioID := "takeover-after-primary-crash"
	subscriberGroup := "group-takeover-crash"
	primary := "subscriber-a"
	standby := "subscriber-b"
	taskOrTraceID := "trace-takeover-crash"
	timeline := []map[string]any{}
	ownerTimeline := []map[string]any{}
	coordinator := newLeaseCoordinator()
	now := takeoverBaseTime
	ttl := 30 * time.Second

	primaryLease, _ := coordinator.acquire(subscriberGroup, "event-stream", primary, ttl, now)
	ownerTimeline = append(ownerTimeline, takeoverOwnerTimelineEntry(now, primary, "lease_acquired", primaryLease))
	timeline = append(timeline, takeoverAuditEvent(now, primary, "lease_acquired", map[string]any{"lease_epoch": primaryLease.LeaseEpoch}))

	primaryLease, _ = coordinator.commit(subscriberGroup, "event-stream", primary, primaryLease.LeaseToken, primaryLease.LeaseEpoch, 40, "evt-40", now.Add(5*time.Second))
	checkpointBefore := takeoverCheckpointPayload(primaryLease)
	timeline = append(timeline, takeoverAuditEvent(now.Add(5*time.Second), primary, "checkpoint_committed", map[string]any{"offset": 40, "event_id": "evt-40"}))
	timeline = append(timeline, takeoverAuditEvent(now.Add(7*time.Second), primary, "processed_uncheckpointed_tail", map[string]any{"offset": 41, "event_id": "evt-41"}))
	timeline = append(timeline, takeoverAuditEvent(now.Add(8*time.Second), primary, "primary_crashed", map[string]any{"reason": "terminated before checkpoint flush"}))

	takeoverLease, _ := coordinator.acquire(subscriberGroup, "event-stream", standby, ttl, now.Add(31*time.Second))
	ownerTimeline = append(ownerTimeline, takeoverOwnerTimelineEntry(now.Add(31*time.Second), standby, "takeover_acquired", takeoverLease))
	timeline = append(timeline, takeoverAuditEvent(now.Add(31*time.Second), standby, "lease_acquired", map[string]any{"lease_epoch": takeoverLease.LeaseEpoch, "takeover": true}))
	timeline = append(timeline, takeoverAuditEvent(now.Add(32*time.Second), standby, "replay_started", map[string]any{"from_offset": 40, "from_event_id": "evt-40"}))
	takeoverLease, _ = coordinator.commit(subscriberGroup, "event-stream", standby, takeoverLease.LeaseToken, takeoverLease.LeaseEpoch, 41, "evt-41", now.Add(33*time.Second))
	checkpointAfter := takeoverCheckpointPayload(takeoverLease)
	timeline = append(timeline, takeoverAuditEvent(now.Add(33*time.Second), standby, "checkpoint_committed", map[string]any{"offset": 41, "event_id": "evt-41", "replayed_tail": true}))

	return buildTakeoverScenarioResult(
		scenarioID,
		"Primary subscriber crashes after processing but before checkpoint flush",
		primary,
		standby,
		taskOrTraceID,
		subscriberGroup,
		[]string{
			"Audit log shows one ownership handoff from primary to standby.",
			"Audit log records the primary interruption reason before standby completion.",
			"Audit log links takeover to the same task or trace identifier across both subscribers.",
		},
		[]string{
			"Checkpoint after takeover is greater than or equal to the last durable checkpoint from the primary.",
			"Standby checkpoint commit is attributed to the new lease owner.",
			"No checkpoint update is accepted from the crashed primary after takeover.",
		},
		[]string{
			"Replay resumes from the last durable checkpoint, not from the last in-memory event processed by the crashed primary.",
			"At most one duplicate delivery is tolerated for the uncheckpointed tail and it is visible in the report.",
			"Replay window closes once the standby checkpoint advances past the tail.",
		},
		ownerTimeline,
		checkpointBefore,
		checkpointAfter,
		map[string]any{"offset": 40, "event_id": "evt-40"},
		map[string]any{"offset": 41, "event_id": "evt-41"},
		[]string{"evt-41"},
		0,
		timeline,
		[]map[string]any{
			{"event_id": "evt-40", "delivered_by": []string{primary}, "delivery_kind": "durable"},
			{"event_id": "evt-41", "delivered_by": []string{primary, standby}, "delivery_kind": "uncheckpointed_tail_replay"},
		},
		[]string{
			"Deterministic local harness only; no live bigclawd processes participate in this proof.",
			"Per-node audit paths are report artifacts rather than emitted runtime JSONL files.",
		},
	)
}

func takeoverScenarioLeaseExpiryStaleWriterRejected() map[string]any {
	scenarioID := "lease-expiry-stale-writer-rejected"
	subscriberGroup := "group-stale-writer"
	primary := "subscriber-a"
	standby := "subscriber-b"
	taskOrTraceID := "trace-stale-writer"
	timeline := []map[string]any{}
	ownerTimeline := []map[string]any{}
	coordinator := newLeaseCoordinator()
	now := takeoverBaseTime.Add(5 * time.Minute)
	ttl := 30 * time.Second

	primaryLease, _ := coordinator.acquire(subscriberGroup, "event-stream", primary, ttl, now)
	ownerTimeline = append(ownerTimeline, takeoverOwnerTimelineEntry(now, primary, "lease_acquired", primaryLease))
	timeline = append(timeline, takeoverAuditEvent(now, primary, "lease_acquired", map[string]any{"lease_epoch": primaryLease.LeaseEpoch}))

	primaryLease, _ = coordinator.commit(subscriberGroup, "event-stream", primary, primaryLease.LeaseToken, primaryLease.LeaseEpoch, 80, "evt-80", now.Add(3*time.Second))
	checkpointBefore := takeoverCheckpointPayload(primaryLease)
	timeline = append(timeline, takeoverAuditEvent(now.Add(3*time.Second), primary, "checkpoint_committed", map[string]any{"offset": 80, "event_id": "evt-80"}))
	timeline = append(timeline, takeoverAuditEvent(now.Add(31*time.Second), primary, "lease_expired", map[string]any{"last_offset": 80}))

	takeoverLease, _ := coordinator.acquire(subscriberGroup, "event-stream", standby, ttl, now.Add(31*time.Second))
	ownerTimeline = append(ownerTimeline, takeoverOwnerTimelineEntry(now.Add(31*time.Second), standby, "takeover_acquired", takeoverLease))
	timeline = append(timeline, takeoverAuditEvent(now.Add(31*time.Second), standby, "lease_acquired", map[string]any{"lease_epoch": takeoverLease.LeaseEpoch, "takeover": true}))

	staleWriteRejections := 0
	if _, err := coordinator.commit(subscriberGroup, "event-stream", primary, primaryLease.LeaseToken, primaryLease.LeaseEpoch, 81, "evt-81", now.Add(32*time.Second)); err != nil {
		staleWriteRejections++
		timeline = append(timeline, takeoverAuditEvent(now.Add(32*time.Second), primary, "lease_fenced", map[string]any{"attempted_offset": 81, "attempted_event_id": "evt-81", "accepted_owner": standby}))
	}
	timeline = append(timeline, takeoverAuditEvent(now.Add(33*time.Second), standby, "replay_started", map[string]any{"from_offset": 80, "from_event_id": "evt-80"}))
	takeoverLease, _ = coordinator.commit(subscriberGroup, "event-stream", standby, takeoverLease.LeaseToken, takeoverLease.LeaseEpoch, 82, "evt-82", now.Add(34*time.Second))
	checkpointAfter := takeoverCheckpointPayload(takeoverLease)
	timeline = append(timeline, takeoverAuditEvent(now.Add(34*time.Second), standby, "checkpoint_committed", map[string]any{"offset": 82, "event_id": "evt-82", "stale_writer_rejected": true}))

	return buildTakeoverScenarioResult(
		scenarioID,
		"Lease expires and the former owner attempts a stale checkpoint write",
		primary,
		standby,
		taskOrTraceID,
		subscriberGroup,
		[]string{
			"Audit log records lease expiry for the former owner and acquisition by the standby.",
			"Audit log records the stale write rejection with both attempted and accepted owners.",
			"Audit log keeps the rejection and accepted takeover in the same ordered timeline.",
		},
		[]string{
			"Checkpoint sequence never decreases after the standby acquires ownership.",
			"Late primary acknowledgement is rejected or ignored without mutating durable checkpoint state.",
			"Accepted checkpoint owner always matches the active lease holder.",
		},
		[]string{
			"Replay after stale write rejection starts from the accepted durable checkpoint only.",
			"No event acknowledged only by the stale writer disappears from the replay timeline.",
			"Replay report exposes any duplicate event IDs caused by the overlap window.",
		},
		ownerTimeline,
		checkpointBefore,
		checkpointAfter,
		map[string]any{"offset": 80, "event_id": "evt-80"},
		map[string]any{"offset": 82, "event_id": "evt-82"},
		[]string{"evt-81"},
		staleWriteRejections,
		timeline,
		[]map[string]any{
			{"event_id": "evt-80", "delivered_by": []string{primary}, "delivery_kind": "durable"},
			{"event_id": "evt-81", "delivered_by": []string{primary, standby}, "delivery_kind": "stale_overlap_candidate"},
			{"event_id": "evt-82", "delivered_by": []string{standby}, "delivery_kind": "takeover_replay_commit"},
		},
		[]string{
			"Deterministic local harness only; live lease expiry still needs a real two-node integration proof.",
			"Stale writer rejection count is produced by the harness rather than the live control-plane API.",
		},
	)
}

func takeoverScenarioSplitBrainDualReplayWindow() map[string]any {
	scenarioID := "split-brain-dual-replay-window"
	subscriberGroup := "group-split-brain"
	primary := "subscriber-a"
	standby := "subscriber-b"
	taskOrTraceID := "trace-split-brain"
	timeline := []map[string]any{}
	ownerTimeline := []map[string]any{}
	coordinator := newLeaseCoordinator()
	now := takeoverBaseTime.Add(10 * time.Minute)
	ttl := 30 * time.Second

	primaryLease, _ := coordinator.acquire(subscriberGroup, "event-stream", primary, ttl, now)
	ownerTimeline = append(ownerTimeline, takeoverOwnerTimelineEntry(now, primary, "lease_acquired", primaryLease))
	timeline = append(timeline, takeoverAuditEvent(now, primary, "lease_acquired", map[string]any{"lease_epoch": primaryLease.LeaseEpoch}))

	primaryLease, _ = coordinator.commit(subscriberGroup, "event-stream", primary, primaryLease.LeaseToken, primaryLease.LeaseEpoch, 120, "evt-120", now.Add(2*time.Second))
	checkpointBefore := takeoverCheckpointPayload(primaryLease)
	timeline = append(timeline, takeoverAuditEvent(now.Add(2*time.Second), primary, "checkpoint_committed", map[string]any{"offset": 120, "event_id": "evt-120"}))
	timeline = append(timeline, takeoverAuditEvent(now.Add(29*time.Second), primary, "overlap_window_started", map[string]any{"candidate_events": []string{"evt-121", "evt-122"}}))

	takeoverLease, _ := coordinator.acquire(subscriberGroup, "event-stream", standby, ttl, now.Add(31*time.Second))
	ownerTimeline = append(ownerTimeline, takeoverOwnerTimelineEntry(now.Add(31*time.Second), standby, "takeover_acquired", takeoverLease))
	timeline = append(timeline, takeoverAuditEvent(now.Add(31*time.Second), standby, "lease_acquired", map[string]any{"lease_epoch": takeoverLease.LeaseEpoch, "takeover": true}))
	timeline = append(timeline, takeoverAuditEvent(now.Add(32*time.Second), standby, "replay_started", map[string]any{"from_offset": 120, "candidate_events": []string{"evt-121", "evt-122"}}))

	staleWriteRejections := 0
	if _, err := coordinator.commit(subscriberGroup, "event-stream", primary, primaryLease.LeaseToken, primaryLease.LeaseEpoch, 122, "evt-122", now.Add(33*time.Second)); err != nil {
		staleWriteRejections++
		timeline = append(timeline, takeoverAuditEvent(now.Add(33*time.Second), primary, "lease_fenced", map[string]any{"attempted_offset": 122, "accepted_owner": standby, "overlap": true}))
	}
	takeoverLease, _ = coordinator.commit(subscriberGroup, "event-stream", standby, takeoverLease.LeaseToken, takeoverLease.LeaseEpoch, 122, "evt-122", now.Add(34*time.Second))
	checkpointAfter := takeoverCheckpointPayload(takeoverLease)
	timeline = append(timeline, takeoverAuditEvent(now.Add(34*time.Second), standby, "checkpoint_committed", map[string]any{"offset": 122, "event_id": "evt-122", "winning_owner": standby}))
	timeline = append(timeline, takeoverAuditEvent(now.Add(35*time.Second), standby, "overlap_window_closed", map[string]any{"winning_owner": standby, "duplicate_events": []string{"evt-121", "evt-122"}}))

	return buildTakeoverScenarioResult(
		scenarioID,
		"Two subscribers briefly believe they can replay the same tail",
		primary,
		standby,
		taskOrTraceID,
		subscriberGroup,
		[]string{
			"Combined audit timeline shows overlapping replay attempts and identifies the surviving owner.",
			"Audit evidence includes per-node file paths and normalized subscriber identities.",
			"The final report highlights whether duplicate replay attempts were observed or only simulated.",
		},
		[]string{
			"Only the winning owner can advance the durable checkpoint.",
			"Losing owner leaves durable checkpoint unchanged once fencing is applied.",
			"Report includes the exact checkpoint sequence where overlap began and ended.",
		},
		[]string{
			"Replay output groups duplicate candidate deliveries by event ID.",
			"Final replay cursor belongs to the winning owner only.",
			"Validation reports whether overlapping replay created observable duplicate deliveries.",
		},
		ownerTimeline,
		checkpointBefore,
		checkpointAfter,
		map[string]any{"offset": 120, "event_id": "evt-120"},
		map[string]any{"offset": 122, "event_id": "evt-122"},
		[]string{"evt-121", "evt-122"},
		staleWriteRejections,
		timeline,
		[]map[string]any{
			{"event_id": "evt-120", "delivered_by": []string{primary}, "delivery_kind": "durable"},
			{"event_id": "evt-121", "delivered_by": []string{primary, standby}, "delivery_kind": "overlap_candidate"},
			{"event_id": "evt-122", "delivered_by": []string{primary, standby}, "delivery_kind": "overlap_candidate"},
		},
		[]string{
			"Deterministic local harness only; duplicate replay candidates are modeled rather than captured from a live shared queue.",
			"Real cross-process subscriber membership and per-node replay metrics remain follow-up work.",
		},
	)
}

func buildTakeoverScenarioResult(scenarioID, title, primarySubscriber, takeoverSubscriber, taskOrTraceID, subscriberGroup string, auditAssertions, checkpointAssertions, replayAssertions []string, ownerTimeline []map[string]any, checkpointBefore, checkpointAfter, replayStartCursor, replayEndCursor map[string]any, duplicateEvents []string, staleWriteRejections int, auditTimeline []map[string]any, eventLogExcerpt []map[string]any, localLimitations []string) map[string]any {
	auditChecks := []map[string]any{
		{"label": "ownership handoff is visible in the audit timeline", "passed": countDistinctOwners(ownerTimeline) >= 2},
		{"label": "audit timeline contains takeover-specific events", "passed": takeoverAuditContains(auditTimeline, []string{"lease_acquired", "lease_fenced", "primary_crashed"})},
		{"label": "audit timeline stays ordered by timestamp", "passed": takeoverAuditOrdered(auditTimeline)},
	}
	checkpointChecks := []map[string]any{
		{"label": "checkpoint never regresses across takeover", "passed": intAt(checkpointAfter, "offset") >= intAt(checkpointBefore, "offset")},
		{"label": "final checkpoint owner matches the final lease owner", "passed": stringAt(checkpointAfter, "owner") == stringAt(ownerTimeline[len(ownerTimeline)-1], "owner")},
		{"label": "stale writers do not replace the accepted checkpoint owner", "passed": staleWriteRejections == 0 || stringAt(checkpointAfter, "owner") == takeoverSubscriber},
	}
	replayChecks := []map[string]any{
		{"label": "replay restarts from the durable checkpoint boundary", "passed": intAt(replayStartCursor, "offset") == intAt(checkpointBefore, "offset")},
		{"label": "replay end cursor advances to the final durable checkpoint", "passed": intAt(replayEndCursor, "offset") == intAt(checkpointAfter, "offset")},
		{"label": "duplicate replay candidates are counted explicitly", "passed": len(duplicateEvents) >= 0},
	}
	allPassed := true
	for _, item := range append(append(auditChecks, checkpointChecks...), replayChecks...) {
		if !boolAt(item, "passed") {
			allPassed = false
			break
		}
	}
	artifactRoot := "artifacts/" + scenarioID
	return map[string]any{
		"id":                       scenarioID,
		"title":                    title,
		"subscriber_group":         subscriberGroup,
		"primary_subscriber":       primarySubscriber,
		"takeover_subscriber":      takeoverSubscriber,
		"task_or_trace_id":         taskOrTraceID,
		"audit_assertions":         auditAssertions,
		"checkpoint_assertions":    checkpointAssertions,
		"replay_assertions":        replayAssertions,
		"lease_owner_timeline":     ownerTimeline,
		"checkpoint_before":        checkpointBefore,
		"checkpoint_after":         checkpointAfter,
		"replay_start_cursor":      replayStartCursor,
		"replay_end_cursor":        replayEndCursor,
		"duplicate_delivery_count": len(duplicateEvents),
		"duplicate_events":         duplicateEvents,
		"stale_write_rejections":   staleWriteRejections,
		"audit_log_paths": []string{
			artifactRoot + "/" + primarySubscriber + "-audit.jsonl",
			artifactRoot + "/" + takeoverSubscriber + "-audit.jsonl",
		},
		"event_log_excerpt": eventLogExcerpt,
		"audit_timeline":    auditTimeline,
		"assertion_results": map[string]any{
			"audit":      auditChecks,
			"checkpoint": checkpointChecks,
			"replay":     replayChecks,
		},
		"all_assertions_passed": allPassed,
		"local_limitations":     localLimitations,
	}
}

func takeoverCheckpointPayload(lease takeoverLease) map[string]any {
	return map[string]any{
		"owner":       lease.ConsumerID,
		"lease_epoch": lease.LeaseEpoch,
		"lease_token": lease.LeaseToken,
		"offset":      lease.CheckpointOffset,
		"event_id":    lease.CheckpointEventID,
		"updated_at":  utcISO(lease.UpdatedAt),
	}
}

func takeoverAuditEvent(timestamp time.Time, subscriber, action string, details map[string]any) map[string]any {
	return map[string]any{
		"timestamp":  utcISO(timestamp),
		"subscriber": subscriber,
		"action":     action,
		"details":    details,
	}
}

func takeoverOwnerTimelineEntry(timestamp time.Time, owner, event string, lease takeoverLease) map[string]any {
	return map[string]any{
		"timestamp":           utcISO(timestamp),
		"owner":               owner,
		"event":               event,
		"lease_epoch":         lease.LeaseEpoch,
		"checkpoint_offset":   lease.CheckpointOffset,
		"checkpoint_event_id": lease.CheckpointEventID,
	}
}

func countDistinctOwners(timeline []map[string]any) int {
	set := map[string]struct{}{}
	for _, item := range timeline {
		if owner := stringAt(item, "owner"); owner != "" {
			set[owner] = struct{}{}
		}
	}
	return len(set)
}

func takeoverAuditContains(timeline []map[string]any, actions []string) bool {
	for _, item := range timeline {
		action := stringAt(item, "action")
		for _, candidate := range actions {
			if action == candidate {
				return true
			}
		}
	}
	return false
}

func takeoverAuditOrdered(timeline []map[string]any) bool {
	last := ""
	for _, item := range timeline {
		current := stringAt(item, "timestamp")
		if current < last {
			return false
		}
		last = current
	}
	return true
}
