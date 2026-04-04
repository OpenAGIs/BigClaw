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
		_, _ = os.Stdout.WriteString("usage: bigclawctl e2e <run-task-smoke|validation-bundle-continuation-scorecard|validation-bundle-continuation-policy-gate> [flags]\n")
		return nil
	}
	command := args[0]
	switch command {
	case "run-task-smoke":
		return runE2ETaskSmoke(args[1:])
	case "validation-bundle-continuation-scorecard":
		return runE2EValidationBundleContinuationScorecard(args[1:])
	case "validation-bundle-continuation-policy-gate":
		return runE2EValidationBundleContinuationPolicyGate(args[1:])
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

func runE2EValidationBundleContinuationScorecard(args []string) error {
	flags := flag.NewFlagSet("e2e validation-bundle-continuation-scorecard", flag.ContinueOnError)
	repoRoot := flags.String("repo-root", "..", "repository root")
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
	repoRoot := flags.String("repo-root", "..", "repository root")
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
