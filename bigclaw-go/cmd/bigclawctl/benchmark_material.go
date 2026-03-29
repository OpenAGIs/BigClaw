package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

var benchmarkLinePattern = regexp.MustCompile(`^(Benchmark\S+)\s+\d+\s+([0-9.]+)\s+ns/op$`)

type benchmarkMetric struct {
	NSPerOp float64 `json:"ns_per_op"`
}

type benchmarkOutput struct {
	Stdout string                     `json:"stdout"`
	Parsed map[string]benchmarkMetric `json:"parsed"`
}

type benchmarkScenario struct {
	Count   int `json:"count"`
	Workers int `json:"workers"`
}

type benchmarkScenarioResult struct {
	Scenario   benchmarkScenario          `json:"scenario"`
	ReportPath string                     `json:"report_path"`
	Result     *automationSoakLocalReport `json:"result"`
}

type benchmarkMatrixReport struct {
	Benchmark  benchmarkOutput           `json:"benchmark"`
	SoakMatrix []benchmarkScenarioResult `json:"soak_matrix"`
}

type automationBenchmarkRunMatrixOptions struct {
	GoRoot          string
	ReportPath      string
	TimeoutSeconds  int
	Scenarios       []string
	BenchmarkRunner func(string) (benchmarkOutput, error)
	SoakRunner      func(string, int, int, int, string) (*automationSoakLocalReport, error)
}

var (
	microbenchmarkLimits = map[string]float64{
		"BenchmarkMemoryQueueEnqueueLease-8": 100_000,
		"BenchmarkFileQueueEnqueueLease-8":   40_000_000,
		"BenchmarkSQLiteQueueEnqueueLease-8": 25_000_000,
		"BenchmarkSchedulerDecide-8":         1_000,
	}
	soakThresholds = map[string]struct {
		MinThroughput float64
		MaxFailures   int
		Envelope      string
	}{
		"50x8":    {MinThroughput: 5.0, MaxFailures: 0, Envelope: "bootstrap-burst"},
		"100x12":  {MinThroughput: 8.5, MaxFailures: 0, Envelope: "bootstrap-burst"},
		"1000x24": {MinThroughput: 9.0, MaxFailures: 0, Envelope: "recommended-local-sustained"},
		"2000x24": {MinThroughput: 8.5, MaxFailures: 0, Envelope: "recommended-local-ceiling"},
	}
)

const saturationDropThresholdPct = 12.0

func runAutomationBenchmarkRunMatrixCommand(args []string) error {
	flags := flag.NewFlagSet("automation benchmark run-matrix", flag.ContinueOnError)
	goRoot := flags.String("go-root", ".", "bigclaw-go repo root")
	reportPath := flags.String("report-path", "docs/reports/benchmark-matrix-report.json", "relative or absolute report path")
	timeoutSeconds := flags.Int("timeout-seconds", 180, "task timeout seconds")
	scenarioValues := multiStringFlag{}
	flags.Var(&scenarioValues, "scenario", "count:workers; defaults to 50:8 and 100:12")
	asJSON := flags.Bool("json", true, "json")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation benchmark run-matrix [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}

	report, err := automationBenchmarkRunMatrix(automationBenchmarkRunMatrixOptions{
		GoRoot:         absPath(*goRoot),
		ReportPath:     *reportPath,
		TimeoutSeconds: *timeoutSeconds,
		Scenarios:      scenarioValues,
	})
	if err != nil {
		return err
	}
	return emit(structToMap(report), *asJSON, 0)
}

func runAutomationBenchmarkCapacityCertificationCommand(args []string) error {
	flags := flag.NewFlagSet("automation benchmark capacity-certification", flag.ContinueOnError)
	goRoot := flags.String("go-root", "..", "repo root")
	benchmarkReport := flags.String("benchmark-report", "bigclaw-go/docs/reports/benchmark-matrix-report.json", "benchmark report path")
	mixedWorkloadReport := flags.String("mixed-workload-report", "bigclaw-go/docs/reports/mixed-workload-matrix-report.json", "mixed workload report path")
	supplementalSoakReports := multiStringFlag{}
	flags.Var(&supplementalSoakReports, "supplemental-soak-report", "additional soak report path")
	outputPath := flags.String("output", "bigclaw-go/docs/reports/capacity-certification-matrix.json", "output JSON path")
	markdownOutputPath := flags.String("markdown-output", "bigclaw-go/docs/reports/capacity-certification-report.md", "output markdown path")
	pretty := flags.Bool("pretty", false, "print formatted json to stdout")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation benchmark capacity-certification [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}

	report, markdown, err := buildCapacityCertificationReport(absPath(*goRoot), *benchmarkReport, *mixedWorkloadReport, supplementalSoakReports)
	if err != nil {
		return err
	}
	if err := writeJSONFile(resolveRepoRelativePathForRoot(absPath(*goRoot), *outputPath), report); err != nil {
		return err
	}
	if err := writeTextFile(resolveRepoRelativePathForRoot(absPath(*goRoot), *markdownOutputPath), markdown); err != nil {
		return err
	}
	if *pretty {
		body, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		_, _ = os.Stdout.Write(append(body, '\n'))
	}
	return nil
}

func automationBenchmarkRunMatrix(opts automationBenchmarkRunMatrixOptions) (*benchmarkMatrixReport, error) {
	scenarios := opts.Scenarios
	if len(scenarios) == 0 {
		scenarios = []string{"50:8", "100:12"}
	}
	benchmarkRunner := opts.BenchmarkRunner
	if benchmarkRunner == nil {
		benchmarkRunner = runLocalBenchmarks
	}
	soakRunner := opts.SoakRunner
	if soakRunner == nil {
		soakRunner = func(goRoot string, count int, workers int, timeoutSeconds int, reportPath string) (*automationSoakLocalReport, error) {
			report, _, err := automationSoakLocal(automationSoakLocalOptions{
				Count:          count,
				Workers:        workers,
				BaseURL:        envOrDefault("BIGCLAW_ADDR", "http://127.0.0.1:8080"),
				GoRoot:         goRoot,
				TimeoutSeconds: timeoutSeconds,
				Autostart:      true,
				ReportPath:     reportPath,
				HTTPClient:     nil,
			})
			return report, err
		}
	}

	benchmarkResult, err := benchmarkRunner(opts.GoRoot)
	if err != nil {
		return nil, err
	}

	report := &benchmarkMatrixReport{
		Benchmark:  benchmarkResult,
		SoakMatrix: make([]benchmarkScenarioResult, 0, len(scenarios)),
	}
	for _, scenarioValue := range scenarios {
		scenario, err := parseScenarioValue(scenarioValue)
		if err != nil {
			return nil, err
		}
		reportPath := filepath.ToSlash(filepath.Join("docs", "reports", fmt.Sprintf("soak-local-%dx%d.json", scenario.Count, scenario.Workers)))
		soakReport, err := soakRunner(opts.GoRoot, scenario.Count, scenario.Workers, opts.TimeoutSeconds, reportPath)
		if err != nil {
			return nil, err
		}
		report.SoakMatrix = append(report.SoakMatrix, benchmarkScenarioResult{
			Scenario:   scenario,
			ReportPath: reportPath,
			Result:     soakReport,
		})
	}

	if err := writeJSONFile(resolveRepoRelativePathForRoot(opts.GoRoot, opts.ReportPath), report); err != nil {
		return nil, err
	}
	return report, nil
}

func runLocalBenchmarks(goRoot string) (benchmarkOutput, error) {
	cmd := exec.Command("go", "test", "-bench", ".", "./internal/queue", "./internal/scheduler")
	cmd.Dir = goRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return benchmarkOutput{}, fmt.Errorf("run benchmarks: %w\n%s", err, strings.TrimSpace(string(output)))
	}
	stdout := string(output)
	return benchmarkOutput{
		Stdout: stdout,
		Parsed: parseBenchmarkStdout(stdout),
	}, nil
}

func parseBenchmarkStdout(stdout string) map[string]benchmarkMetric {
	parsed := map[string]benchmarkMetric{}
	for _, line := range strings.Split(stdout, "\n") {
		match := benchmarkLinePattern.FindStringSubmatch(strings.TrimSpace(line))
		if len(match) != 3 {
			continue
		}
		value, err := strconv.ParseFloat(match[2], 64)
		if err != nil {
			continue
		}
		parsed[match[1]] = benchmarkMetric{NSPerOp: value}
	}
	return parsed
}

func parseScenarioValue(value string) (benchmarkScenario, error) {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return benchmarkScenario{}, fmt.Errorf("invalid scenario %q: expected count:workers", value)
	}
	count, err := strconv.Atoi(parts[0])
	if err != nil || count <= 0 {
		return benchmarkScenario{}, fmt.Errorf("invalid scenario %q: count must be > 0", value)
	}
	workers, err := strconv.Atoi(parts[1])
	if err != nil || workers <= 0 {
		return benchmarkScenario{}, fmt.Errorf("invalid scenario %q: workers must be > 0", value)
	}
	return benchmarkScenario{Count: count, Workers: workers}, nil
}

func buildCapacityCertificationReport(repoRoot string, benchmarkReportPath string, mixedWorkloadReportPath string, supplementalSoakReportPaths []string) (map[string]any, string, error) {
	benchmarkReport, err := loadJSONMap(resolveRepoRelativePathForRoot(repoRoot, benchmarkReportPath))
	if err != nil {
		return nil, "", err
	}
	mixedWorkloadReport, err := loadJSONMap(resolveRepoRelativePathForRoot(repoRoot, mixedWorkloadReportPath))
	if err != nil {
		return nil, "", err
	}
	if len(supplementalSoakReportPaths) == 0 {
		supplementalSoakReportPaths = []string{
			"bigclaw-go/docs/reports/soak-local-1000x24.json",
			"bigclaw-go/docs/reports/soak-local-2000x24.json",
		}
	}

	parsedBenchmarks := mapFrom(getMap(benchmarkReport, "benchmark"), "parsed")
	microbenchmarks := make([]map[string]any, 0, len(microbenchmarkLimits))
	benchmarkNames := make([]string, 0, len(microbenchmarkLimits))
	for name := range microbenchmarkLimits {
		benchmarkNames = append(benchmarkNames, name)
	}
	slices.Sort(benchmarkNames)
	for _, name := range benchmarkNames {
		metric := getMap(parsedBenchmarks, name)
		nsPerOp := getFloat(metric, "ns_per_op")
		microbenchmarks = append(microbenchmarks, benchmarkLane(name, nsPerOp, microbenchmarkLimits[name]))
	}

	soakResultsByLabel := map[string]map[string]any{}
	soakInputs := []string{}
	for _, rawEntry := range getSlice(benchmarkReport, "soak_matrix") {
		entry, ok := rawEntry.(map[string]any)
		if !ok {
			continue
		}
		result := getMap(entry, "result")
		label := fmt.Sprintf("%dx%d", int(getFloat(result, "count")), int(getFloat(result, "workers")))
		soakResultsByLabel[label] = result
		soakInputs = append(soakInputs, repoRelativePath(repoRoot, getString(entry, "report_path")))
	}

	supplementalReports := make([]map[string]any, 0, len(supplementalSoakReportPaths))
	for _, soakPath := range supplementalSoakReportPaths {
		result, err := loadJSONMap(resolveRepoRelativePathForRoot(repoRoot, soakPath))
		if err != nil {
			return nil, "", err
		}
		label := fmt.Sprintf("%dx%d", int(getFloat(result, "count")), int(getFloat(result, "workers")))
		soakResultsByLabel[label] = result
		supplementalReports = append(supplementalReports, result)
		soakInputs = append(soakInputs, repoRelativePath(repoRoot, soakPath))
	}

	soakMatrix := make([]map[string]any, 0, len(soakThresholds))
	soakLabels := make([]string, 0, len(soakThresholds))
	for label := range soakThresholds {
		soakLabels = append(soakLabels, label)
	}
	slices.Sort(soakLabels)
	for _, label := range soakLabels {
		threshold := soakThresholds[label]
		result, ok := soakResultsByLabel[label]
		if !ok {
			return nil, "", fmt.Errorf("missing soak evidence for lane %s", label)
		}
		soakMatrix = append(soakMatrix, soakLane(label, result, threshold.MinThroughput, threshold.MaxFailures, threshold.Envelope))
	}

	mixedWorkload := mixedWorkloadLane(mixedWorkloadReport)
	saturationIndicator, err := buildSaturationSummary(soakMatrix)
	if err != nil {
		return nil, "", err
	}

	allLanes := append([]map[string]any{}, microbenchmarks...)
	allLanes = append(allLanes, soakMatrix...)
	allLanes = append(allLanes, mixedWorkload)

	passedLanes := 0
	failedLanes := []any{}
	for _, lane := range allLanes {
		status := getString(lane, "status")
		if status == "pass" || status == "pass-with-ceiling" {
			passedLanes++
			continue
		}
		failedLanes = append(failedLanes, lane["lane"])
	}

	checks := []any{
		check("all_microbenchmark_thresholds_hold", laneStatusesAll(microbenchmarks, "pass"), fmt.Sprint(laneStatuses(microbenchmarks))),
		check("all_soak_lanes_hold", laneStatusesAll(soakMatrix, "pass"), fmt.Sprint(laneStatuses(soakMatrix))),
		check("mixed_workload_routes_match_expected_executors", getString(mixedWorkload, "status") == "pass" || getString(mixedWorkload, "status") == "pass-with-ceiling", getString(mixedWorkload, "detail")),
		check(
			"ceiling_lane_does_not_show_excessive_throughput_drop",
			getString(saturationIndicator, "status") == "pass",
			fmt.Sprintf("drop_pct=%.2f threshold=%.0f", getFloat(saturationIndicator, "throughput_drop_pct"), saturationDropThresholdPct),
		),
	}

	report := map[string]any{
		"generated_at": deriveGeneratedAt(append([]map[string]any{benchmarkReport, mixedWorkloadReport}, supplementalReports...)...),
		"ticket":       "BIG-PAR-098",
		"title":        "Production-grade capacity certification matrix",
		"status":       "repo-native-capacity-certification",
		"evidence_inputs": map[string]any{
			"benchmark_report_path":      repoRelativePath(repoRoot, benchmarkReportPath),
			"mixed_workload_report_path": repoRelativePath(repoRoot, mixedWorkloadReportPath),
			"soak_report_paths":          soakInputs,
			"generator_script":           "bigclaw-go/cmd/bigclawctl automation benchmark capacity-certification",
		},
		"summary": map[string]any{
			"overall_status":                 map[bool]string{true: "pass", false: "fail"}[len(failedLanes) == 0],
			"total_lanes":                    len(allLanes),
			"passed_lanes":                   passedLanes,
			"failed_lanes":                   failedLanes,
			"recommended_sustained_envelope": "<=1000 tasks with 24 submit workers",
			"ceiling_envelope":               "<=2000 tasks with 24 submit workers",
		},
		"microbenchmarks":      microbenchmarks,
		"soak_matrix":          soakMatrix,
		"mixed_workload":       mixedWorkload,
		"saturation_indicator": saturationIndicator,
		"operating_envelopes": []any{
			map[string]any{
				"name":           "recommended-local-sustained",
				"recommendation": "Use up to 1000 queued tasks with 24 submit workers when a stable single-instance local review lane is required.",
				"evidence_lanes": []any{"1000x24"},
			},
			map[string]any{
				"name":           "recommended-local-ceiling",
				"recommendation": "Treat 2000 queued tasks with 24 submit workers as the checked-in local ceiling, not the default operating point.",
				"evidence_lanes": []any{"2000x24"},
			},
			map[string]any{
				"name":           "mixed-workload-routing",
				"recommendation": "Use the mixed-workload matrix for executor routing correctness, but do not infer sustained multi-executor throughput from it.",
				"evidence_lanes": []any{"mixed-workload-routing"},
			},
		},
		"certification_checks": checks,
		"saturation_notes": []any{
			"Throughput plateaus around 9-10 tasks/s across the checked-in 100x12, 1000x24, and 2000x24 local lanes.",
			"The 2000x24 lane remains within the same throughput band as 1000x24, so the checked-in local ceiling is evidence-backed but not substantially headroom-rich.",
			"Mixed-workload evidence verifies executor-routing correctness across local, Kubernetes, and Ray, but it is a functional routing proof rather than a concurrency ceiling.",
		},
		"limits": []any{
			"Evidence is repo-native and single-instance; it does not certify multi-node or multi-tenant production saturation behavior.",
			"The matrix uses checked-in local runs from 2026-03-13 and should be refreshed when queue, scheduler, or executor behavior changes materially.",
			"Recommended envelopes are conservative reviewer guidance derived from current evidence, not an automated runtime admission policy.",
		},
	}
	markdown := buildCapacityCertificationMarkdown(report)
	return report, markdown, nil
}

func benchmarkLane(name string, nsPerOp float64, maxNSPerOp float64) map[string]any {
	status := "fail"
	if nsPerOp <= maxNSPerOp {
		status = "pass"
	}
	return map[string]any{
		"lane":      name,
		"metric":    "ns_per_op",
		"observed":  nsPerOp,
		"threshold": map[string]any{"operator": "<=", "value": maxNSPerOp},
		"status":    status,
		"detail":    fmt.Sprintf("observed=%gns/op limit=%gns/op", nsPerOp, maxNSPerOp),
	}
}

func soakLane(label string, result map[string]any, minThroughput float64, maxFailures int, envelope string) map[string]any {
	throughput := getFloat(result, "throughput_tasks_per_sec")
	failures := int(getFloat(result, "failed"))
	status := "fail"
	if throughput >= minThroughput && failures <= maxFailures {
		status = "pass"
	}
	return map[string]any{
		"lane": label,
		"scenario": map[string]any{
			"count":   int(getFloat(result, "count")),
			"workers": int(getFloat(result, "workers")),
		},
		"observed": map[string]any{
			"elapsed_seconds":          roundTo(getFloat(result, "elapsed_seconds"), 3),
			"throughput_tasks_per_sec": roundTo(throughput, 3),
			"succeeded":                int(getFloat(result, "succeeded")),
			"failed":                   failures,
		},
		"thresholds": map[string]any{
			"min_throughput_tasks_per_sec": minThroughput,
			"max_failures":                 maxFailures,
		},
		"operating_envelope": envelope,
		"status":             status,
		"detail":             fmt.Sprintf("throughput=%.3ftps min=%.1f failures=%d max=%d", roundTo(throughput, 3), minThroughput, failures, maxFailures),
	}
}

func mixedWorkloadLane(report map[string]any) map[string]any {
	rawTasks := getSlice(report, "tasks")
	mismatches := []string{}
	successfulTasks := 0
	for _, rawTask := range rawTasks {
		task, ok := rawTask.(map[string]any)
		if !ok {
			continue
		}
		if !getBool(task, "ok") {
			mismatches = append(mismatches, fmt.Sprintf("%s: task-level ok=false", getString(task, "name")))
		}
		if getString(task, "expected_executor") != getString(task, "routed_executor") {
			mismatches = append(mismatches, fmt.Sprintf("%s: expected=%s routed=%s", getString(task, "name"), getString(task, "expected_executor"), getString(task, "routed_executor")))
		}
		if getString(task, "final_state") != "succeeded" {
			mismatches = append(mismatches, fmt.Sprintf("%s: final_state=%s", getString(task, "name"), getString(task, "final_state")))
		} else {
			successfulTasks++
		}
	}
	status := "fail"
	if getBool(report, "all_ok") && len(rawTasks) >= 5 && len(mismatches) == 0 {
		status = "pass"
	} else if getBool(report, "all_ok") {
		status = "pass-with-ceiling"
	}
	detail := "all sampled mixed-workload routes landed on the expected executor path"
	if len(mismatches) > 0 {
		detail = strings.Join(mismatches, "; ")
	}
	return map[string]any{
		"lane": "mixed-workload-routing",
		"observed": map[string]any{
			"all_ok":           getBool(report, "all_ok"),
			"task_count":       len(rawTasks),
			"successful_tasks": successfulTasks,
		},
		"thresholds": map[string]any{
			"all_ok_required":               true,
			"minimum_task_count":            5,
			"executor_route_match_required": true,
		},
		"status": status,
		"detail": detail,
		"limitations": []any{
			"executor-mix coverage is functional rather than high-volume",
			"mixed-workload evidence proves route correctness but not sustained cross-executor saturation limits",
		},
	}
}

func buildSaturationSummary(soakLanes []map[string]any) (map[string]any, error) {
	var baseline map[string]any
	var ceiling map[string]any
	for _, lane := range soakLanes {
		switch getString(lane, "lane") {
		case "1000x24":
			baseline = lane
		case "2000x24":
			ceiling = lane
		}
	}
	if baseline == nil || ceiling == nil {
		return nil, errors.New("missing saturation lanes 1000x24/2000x24")
	}
	baselineTPS := getFloat(getMap(baseline, "observed"), "throughput_tasks_per_sec")
	ceilingTPS := getFloat(getMap(ceiling, "observed"), "throughput_tasks_per_sec")
	dropPct := 0.0
	if baselineTPS != 0 {
		dropPct = roundTo(((baselineTPS-ceilingTPS)/baselineTPS)*100, 2)
	}
	status := "warn"
	detail := "throughput drops materially at the 2000-task ceiling and should be treated as saturation"
	if dropPct <= saturationDropThresholdPct {
		status = "pass"
		detail = "throughput remains in the same single-instance local band at the 2000-task ceiling"
	}
	return map[string]any{
		"baseline_lane":                     getString(baseline, "lane"),
		"ceiling_lane":                      getString(ceiling, "lane"),
		"baseline_throughput_tasks_per_sec": baselineTPS,
		"ceiling_throughput_tasks_per_sec":  ceilingTPS,
		"throughput_drop_pct":               dropPct,
		"drop_warn_threshold_pct":           saturationDropThresholdPct,
		"status":                            status,
		"detail":                            detail,
	}, nil
}

func buildCapacityCertificationMarkdown(report map[string]any) string {
	summary := getMap(report, "summary")
	saturation := getMap(report, "saturation_indicator")
	lines := []string{
		"# Capacity Certification Report",
		"",
		"## Scope",
		"",
		fmt.Sprintf("- Generated at: `%s`", getString(report, "generated_at")),
		fmt.Sprintf("- Ticket: `%s`", getString(report, "ticket")),
		"- Goal: convert checked-in benchmark, soak, and mixed-workload evidence into a repo-native certification matrix with explicit thresholds and operating envelopes.",
		"- Boundary: this is a single-instance repo-native certification slice, not a live multi-tenant production attestation.",
		"",
		"## Certification Summary",
		"",
		fmt.Sprintf("- Overall status: `%s`", getString(summary, "overall_status")),
		fmt.Sprintf("- Passed lanes: `%d/%d`", int(getFloat(summary, "passed_lanes")), int(getFloat(summary, "total_lanes"))),
		fmt.Sprintf("- Recommended local sustained envelope: `%s`", getString(summary, "recommended_sustained_envelope")),
		fmt.Sprintf("- Local ceiling envelope: `%s`", getString(summary, "ceiling_envelope")),
		fmt.Sprintf("- Saturation signal: `%s`", getString(saturation, "detail")),
		"",
		"## Admission Policy Summary",
		"",
		"- Policy mode: `advisory-only reviewer guidance`",
		"- Runtime enforcement: `none`",
		fmt.Sprintf("- Default reviewer envelope: `%s`", getString(summary, "recommended_sustained_envelope")),
		fmt.Sprintf("- Ceiling reviewer envelope: `%s`", getString(summary, "ceiling_envelope")),
		"- Scheduler note: recommended envelopes guide reviewer admission decisions and are not scheduler-enforced runtime limits.",
		"",
		"## Microbenchmark Thresholds",
		"",
	}
	for _, rawLane := range getSlice(report, "microbenchmarks") {
		lane, ok := rawLane.(map[string]any)
		if !ok {
			continue
		}
		lines = append(lines, fmt.Sprintf("- `%s`: `%.2f ns/op` vs limit `%.0f` -> `%s`", getString(lane, "lane"), getFloat(lane, "observed"), getFloat(getMap(lane, "threshold"), "value"), getString(lane, "status")))
	}
	lines = append(lines, "", "## Soak Matrix", "")
	for _, rawLane := range getSlice(report, "soak_matrix") {
		lane, ok := rawLane.(map[string]any)
		if !ok {
			continue
		}
		observed := getMap(lane, "observed")
		lines = append(lines, fmt.Sprintf("- `%s`: `%.3f tasks/s`, `%d failed`, envelope `%s` -> `%s`", getString(lane, "lane"), getFloat(observed, "throughput_tasks_per_sec"), int(getFloat(observed, "failed")), getString(lane, "operating_envelope"), getString(lane, "status")))
	}
	mixedWorkload := getMap(report, "mixed_workload")
	lines = append(lines, "", "## Workload Mix", "", fmt.Sprintf("- `mixed-workload-routing`: `%s` -> `%s`", getString(mixedWorkload, "detail"), getString(mixedWorkload, "status")), "", "## Recommended Operating Envelopes", "")
	for _, rawEnvelope := range getSlice(report, "operating_envelopes") {
		envelope, ok := rawEnvelope.(map[string]any)
		if !ok {
			continue
		}
		lines = append(lines, fmt.Sprintf("- `%s`: %s Evidence: `%s`.", getString(envelope, "name"), getString(envelope, "recommendation"), strings.Join(stringSlice(getSlice(envelope, "evidence_lanes")), ", ")))
	}
	lines = append(lines, "", "## Saturation Notes", "")
	for _, note := range stringSlice(getSlice(report, "saturation_notes")) {
		lines = append(lines, "- "+note)
	}
	lines = append(lines, "", "## Limits", "")
	for _, limit := range stringSlice(getSlice(report, "limits")) {
		lines = append(lines, "- "+limit)
	}
	return strings.Join(lines, "\n") + "\n"
}

func check(name string, passed bool, detail string) map[string]any {
	return map[string]any{"name": name, "passed": passed, "detail": detail}
}

func laneStatuses(lanes []map[string]any) []string {
	statuses := make([]string, 0, len(lanes))
	for _, lane := range lanes {
		statuses = append(statuses, getString(lane, "status"))
	}
	return statuses
}

func laneStatusesAll(lanes []map[string]any, expected string) bool {
	for _, lane := range lanes {
		if getString(lane, "status") != expected {
			return false
		}
	}
	return true
}

func deriveGeneratedAt(payloads ...map[string]any) string {
	var latest time.Time
	for _, payload := range payloads {
		for _, candidate := range iterTimestamps(payload) {
			if candidate.After(latest) {
				latest = candidate
			}
		}
	}
	if latest.IsZero() {
		latest = time.Now().UTC()
	}
	return latest.UTC().Format(time.RFC3339Nano)
}

func iterTimestamps(payload any) []time.Time {
	found := []time.Time{}
	switch value := payload.(type) {
	case map[string]any:
		for key, child := range value {
			switch key {
			case "generated_at", "timestamp", "created_at", "completed_at", "started_at":
				if parsed, ok := parseTimestamp(child); ok {
					found = append(found, parsed)
				}
			}
			found = append(found, iterTimestamps(child)...)
		}
	case []any:
		for _, child := range value {
			found = append(found, iterTimestamps(child)...)
		}
	}
	return found
}

func parseTimestamp(value any) (time.Time, bool) {
	text, ok := value.(string)
	if !ok || trim(text) == "" {
		return time.Time{}, false
	}
	candidate := strings.ReplaceAll(text, "Z", "+00:00")
	parsed, err := time.Parse(time.RFC3339Nano, candidate)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func loadJSONMap(path string) (map[string]any, error) {
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

func writeJSONFile(path string, value any) error {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writeTextFile(path, string(body)+"\n")
}

func writeTextFile(path string, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), 0o644)
}

func resolveRepoRelativePathForRoot(repoRoot string, path string) string {
	if trim(path) == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(repoRoot, filepath.FromSlash(path))
}

func repoRelativePath(repoRoot string, path string) string {
	resolved := resolveRepoRelativePathForRoot(repoRoot, path)
	relative, err := filepath.Rel(repoRoot, resolved)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(relative)
}

func getMap(parent map[string]any, key string) map[string]any {
	value, ok := parent[key].(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return value
}

func mapFrom(parent map[string]any, key string) map[string]any {
	return getMap(parent, key)
}

func getSlice(parent map[string]any, key string) []any {
	return anySlice(parent[key])
}

func getString(parent map[string]any, key string) string {
	value, _ := parent[key].(string)
	return value
}

func getFloat(parent map[string]any, key string) float64 {
	switch value := parent[key].(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int64:
		return float64(value)
	case json.Number:
		f, _ := value.Float64()
		return f
	default:
		return 0
	}
}

func getBool(parent map[string]any, key string) bool {
	value, _ := parent[key].(bool)
	return value
}

func stringSlice(values []any) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if text, ok := value.(string); ok {
			result = append(result, text)
		}
	}
	return result
}

func anySlice(value any) []any {
	if direct, ok := value.([]any); ok {
		return direct
	}
	rv := reflect.ValueOf(value)
	if !rv.IsValid() || rv.Kind() != reflect.Slice {
		return nil
	}
	result := make([]any, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		result = append(result, rv.Index(i).Interface())
	}
	return result
}

func roundTo(value float64, decimals int) float64 {
	factor := mathPow10(decimals)
	return float64(int(value*factor+0.5)) / factor
}

func mathPow10(power int) float64 {
	result := 1.0
	for i := 0; i < power; i++ {
		result *= 10
	}
	return result
}

type multiStringFlag []string

func (m *multiStringFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiStringFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}
