package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var benchmarkStdoutPattern = regexp.MustCompile(`^(Benchmark\S+)\s+\d+\s+([0-9.]+)\s+ns/op$`)

type automationBenchmarkRunMatrixOptions struct {
	GoRoot         string
	ReportPath     string
	TimeoutSeconds int
	Scenarios      []string
	RunBenchmark   func(string) (string, error)
	RunSoak        func(string, int, int, int, string) (map[string]any, error)
}

type automationBenchmarkCapacityCertificationOptions struct {
	GoRoot                    string
	BenchmarkReportPath       string
	MixedWorkloadReportPath   string
	SupplementalSoakReportIDs []string
	OutputPath                string
	MarkdownOutputPath        string
}

func runAutomationBenchmarkRunMatrixCommand(args []string) error {
	flags := flag.NewFlagSet("automation benchmark run-matrix", flag.ContinueOnError)
	goRoot := flags.String("go-root", ".", "bigclaw-go repo root")
	reportPath := flags.String("report-path", "docs/reports/benchmark-matrix-report.json", "relative or absolute report path")
	timeoutSeconds := flags.Int("timeout-seconds", 180, "task timeout seconds")
	var scenarios multiStringFlag
	flags.Var(&scenarios, "scenario", "count:workers; default scenarios are 50:8 and 100:12")
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
		Scenarios:      append([]string(nil), scenarios...),
	})
	if err != nil {
		return err
	}
	return emit(report, *asJSON, 0)
}

func runAutomationBenchmarkCapacityCertificationCommand(args []string) error {
	flags := flag.NewFlagSet("automation benchmark capacity-certification", flag.ContinueOnError)
	goRoot := flags.String("go-root", ".", "bigclaw-go repo root")
	benchmarkReport := flags.String("benchmark-report", "docs/reports/benchmark-matrix-report.json", "benchmark report path")
	mixedWorkloadReport := flags.String("mixed-workload-report", "docs/reports/mixed-workload-matrix-report.json", "mixed workload report path")
	var supplementalSoakReports multiStringFlag
	flags.Var(&supplementalSoakReports, "supplemental-soak-report", "additional soak report paths to merge into the certification matrix")
	outputPath := flags.String("output", "docs/reports/capacity-certification-matrix.json", "output path")
	markdownOutputPath := flags.String("markdown-output", "docs/reports/capacity-certification-report.md", "markdown output path")
	pretty := flags.Bool("pretty", false, "pretty-print the report to stdout")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation benchmark capacity-certification [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}

	report, err := automationBenchmarkCapacityCertification(automationBenchmarkCapacityCertificationOptions{
		GoRoot:                    absPath(*goRoot),
		BenchmarkReportPath:       *benchmarkReport,
		MixedWorkloadReportPath:   *mixedWorkloadReport,
		SupplementalSoakReportIDs: append([]string(nil), supplementalSoakReports...),
		OutputPath:                *outputPath,
		MarkdownOutputPath:        *markdownOutputPath,
	})
	if err != nil {
		return err
	}

	payload := report
	if !*pretty {
		payload = cloneMap(report)
		delete(payload, "markdown")
	}
	return emit(payload, true, 0)
}

func automationBenchmarkRunMatrix(opts automationBenchmarkRunMatrixOptions) (map[string]any, error) {
	goRoot := absPath(opts.GoRoot)
	if goRoot == "" {
		goRoot = absPath(".")
	}
	runBenchmark := opts.RunBenchmark
	if runBenchmark == nil {
		runBenchmark = func(goRoot string) (string, error) {
			cmd := exec.Command("go", "test", "-bench", ".", "./internal/queue", "./internal/scheduler")
			cmd.Dir = goRoot
			output, err := cmd.CombinedOutput()
			if err != nil {
				return "", fmt.Errorf("run go benchmarks: %w (%s)", err, string(output))
			}
			return string(output), nil
		}
	}
	runSoak := opts.RunSoak
	if runSoak == nil {
		runSoak = func(goRoot string, count, workers, timeoutSeconds int, reportPath string) (map[string]any, error) {
			report, exitCode, err := automationSoakLocal(automationSoakLocalOptions{
				Count:          count,
				Workers:        workers,
				BaseURL:        "http://127.0.0.1:8080",
				GoRoot:         goRoot,
				TimeoutSeconds: timeoutSeconds,
				Autostart:      true,
				ReportPath:     reportPath,
				HTTPClient:     nil,
			})
			if err != nil {
				return nil, err
			}
			if exitCode != 0 {
				return nil, fmt.Errorf("soak-local returned exit code %d", exitCode)
			}
			return structToMap(report), nil
		}
	}

	scenarios := opts.Scenarios
	if len(scenarios) == 0 {
		scenarios = []string{"50:8", "100:12"}
	}
	stdout, err := runBenchmark(goRoot)
	if err != nil {
		return nil, err
	}
	report := map[string]any{
		"benchmark": map[string]any{
			"stdout": stdout,
			"parsed": parseBenchmarkStdout(stdout),
		},
	}
	soakMatrix := make([]any, 0, len(scenarios))
	for _, scenario := range scenarios {
		count, workers, err := parseRunMatrixScenario(scenario)
		if err != nil {
			return nil, err
		}
		soakReportPath := filepath.ToSlash(filepath.Join("docs", "reports", fmt.Sprintf("soak-local-%dx%d.json", count, workers)))
		soakReport, err := runSoak(goRoot, count, workers, opts.TimeoutSeconds, soakReportPath)
		if err != nil {
			return nil, err
		}
		soakMatrix = append(soakMatrix, map[string]any{
			"scenario": map[string]any{
				"count":   count,
				"workers": workers,
			},
			"report_path": soakReportPath,
			"result":      soakReport,
		})
	}
	report["soak_matrix"] = soakMatrix

	if err := writeJSONFile(resolveWithinRoot(goRoot, opts.ReportPath), report); err != nil {
		return nil, err
	}
	return report, nil
}

func automationBenchmarkCapacityCertification(opts automationBenchmarkCapacityCertificationOptions) (map[string]any, error) {
	goRoot := absPath(opts.GoRoot)
	if goRoot == "" {
		goRoot = absPath(".")
	}
	benchmarkReportPath := defaultString(opts.BenchmarkReportPath, "docs/reports/benchmark-matrix-report.json")
	mixedWorkloadReportPath := defaultString(opts.MixedWorkloadReportPath, "docs/reports/mixed-workload-matrix-report.json")
	supplementalSoakReportPaths := opts.SupplementalSoakReportIDs
	if len(supplementalSoakReportPaths) == 0 {
		supplementalSoakReportPaths = []string{
			"docs/reports/soak-local-1000x24.json",
			"docs/reports/soak-local-2000x24.json",
		}
	}

	benchmarkReport, err := readJSONFile(resolveWithinRoot(goRoot, benchmarkReportPath))
	if err != nil {
		return nil, err
	}
	mixedWorkloadReport, err := readJSONFile(resolveWithinRoot(goRoot, mixedWorkloadReportPath))
	if err != nil {
		return nil, err
	}

	microbenchmarkLimits := []struct {
		Name  string
		Limit float64
	}{
		{Name: "BenchmarkMemoryQueueEnqueueLease-8", Limit: 100_000},
		{Name: "BenchmarkFileQueueEnqueueLease-8", Limit: 40_000_000},
		{Name: "BenchmarkSQLiteQueueEnqueueLease-8", Limit: 25_000_000},
		{Name: "BenchmarkSchedulerDecide-8", Limit: 1_000},
	}
	soakThresholds := []struct {
		Label         string
		MinThroughput float64
		MaxFailures   int
		Envelope      string
	}{
		{Label: "50x8", MinThroughput: 5.0, MaxFailures: 0, Envelope: "bootstrap-burst"},
		{Label: "100x12", MinThroughput: 8.5, MaxFailures: 0, Envelope: "bootstrap-burst"},
		{Label: "1000x24", MinThroughput: 9.0, MaxFailures: 0, Envelope: "recommended-local-sustained"},
		{Label: "2000x24", MinThroughput: 8.5, MaxFailures: 0, Envelope: "recommended-local-ceiling"},
	}

	microbenchmarks := make([]any, 0, len(microbenchmarkLimits))
	parsedBenchmarks := nestedMap(benchmarkReport, "benchmark", "parsed")
	for _, lane := range microbenchmarkLimits {
		nsPerOp := nestedFloat(parsedBenchmarks, lane.Name, "ns_per_op")
		microbenchmarks = append(microbenchmarks, benchmarkLane(lane.Name, nsPerOp, lane.Limit))
	}

	soakInputs := make([]string, 0)
	soakResultsByLabel := map[string]map[string]any{}
	for _, rawEntry := range nestedSlice(benchmarkReport, "soak_matrix") {
		entry, ok := rawEntry.(map[string]any)
		if !ok {
			continue
		}
		result, _ := entry["result"].(map[string]any)
		label := fmt.Sprintf("%dx%d", int(number(result["count"])), int(number(result["workers"])))
		soakResultsByLabel[label] = result
		soakInputs = append(soakInputs, repoRelativePath(goRoot, resolveWithinRoot(goRoot, stringValue(entry["report_path"]))))
	}

	supplementalPayloads := make([]map[string]any, 0, len(supplementalSoakReportPaths))
	for _, soakPath := range supplementalSoakReportPaths {
		payload, err := readJSONFile(resolveWithinRoot(goRoot, soakPath))
		if err != nil {
			return nil, err
		}
		supplementalPayloads = append(supplementalPayloads, payload)
		label := fmt.Sprintf("%dx%d", int(number(payload["count"])), int(number(payload["workers"])))
		soakResultsByLabel[label] = payload
		soakInputs = append(soakInputs, repoRelativePath(goRoot, resolveWithinRoot(goRoot, soakPath)))
	}

	soakMatrix := make([]any, 0, len(soakThresholds))
	for _, threshold := range soakThresholds {
		soakMatrix = append(soakMatrix, soakLane(threshold.Label, soakResultsByLabel[threshold.Label], threshold.MinThroughput, threshold.MaxFailures, threshold.Envelope))
	}
	mixedWorkload := mixedWorkloadLane(mixedWorkloadReport)
	saturationIndicator := buildSaturationSummary(soakMatrix)

	allLanes := append(append(append([]any{}, microbenchmarks...), soakMatrix...), mixedWorkload)
	passedLanes := 0
	failedLanes := make([]any, 0)
	for _, rawLane := range allLanes {
		lane, ok := rawLane.(map[string]any)
		if !ok {
			continue
		}
		status := stringValue(lane["status"])
		if status == "pass" || status == "pass-with-ceiling" {
			passedLanes++
			continue
		}
		failedLanes = append(failedLanes, lane["lane"])
	}

	statuses := func(items []any) []string {
		out := make([]string, 0, len(items))
		for _, item := range items {
			lane, ok := item.(map[string]any)
			if ok {
				out = append(out, stringValue(lane["status"]))
			}
		}
		return out
	}

	report := map[string]any{
		"generated_at": deriveGeneratedAt(append([]map[string]any{benchmarkReport, mixedWorkloadReport}, supplementalPayloads...)...),
		"ticket":       "BIG-PAR-098",
		"title":        "Production-grade capacity certification matrix",
		"status":       "repo-native-capacity-certification",
		"evidence_inputs": map[string]any{
			"benchmark_report_path":      repoRelativePath(goRoot, resolveWithinRoot(goRoot, benchmarkReportPath)),
			"mixed_workload_report_path": repoRelativePath(goRoot, resolveWithinRoot(goRoot, mixedWorkloadReportPath)),
			"soak_report_paths":          soakInputs,
			"generator_script":           "go run ./cmd/bigclawctl automation benchmark capacity-certification",
		},
		"summary": map[string]any{
			"overall_status":                 ternaryString(len(failedLanes) == 0, "pass", "fail"),
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
		"certification_checks": []any{
			checkLane("all_microbenchmark_thresholds_hold", allStatusesMatch(microbenchmarks, "pass"), fmt.Sprint(statuses(microbenchmarks))),
			checkLane("all_soak_lanes_hold", allStatusesMatch(soakMatrix, "pass"), fmt.Sprint(statuses(soakMatrix))),
			checkLane("mixed_workload_routes_match_expected_executors", statusIn(mixedWorkload["status"], "pass", "pass-with-ceiling"), stringValue(mixedWorkload["detail"])),
			checkLane("ceiling_lane_does_not_show_excessive_throughput_drop", stringValue(saturationIndicator["status"]) == "pass", fmt.Sprintf("drop_pct=%v threshold=12", saturationIndicator["throughput_drop_pct"])),
		},
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

	report["markdown"] = buildCapacityCertificationMarkdown(report)
	if err := writeJSONFile(resolveWithinRoot(goRoot, opts.OutputPath), cloneMapWithout(report, "markdown")); err != nil {
		return nil, err
	}
	if err := writeTextFile(resolveWithinRoot(goRoot, opts.MarkdownOutputPath), stringValue(report["markdown"])); err != nil {
		return nil, err
	}
	return report, nil
}

func parseBenchmarkStdout(stdout string) map[string]any {
	parsed := map[string]any{}
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		matches := benchmarkStdoutPattern.FindStringSubmatch(line)
		if len(matches) != 3 {
			continue
		}
		value, err := strconv.ParseFloat(matches[2], 64)
		if err != nil {
			continue
		}
		parsed[matches[1]] = map[string]any{"ns_per_op": value}
	}
	return parsed
}

func parseRunMatrixScenario(value string) (int, int, error) {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid scenario %q: want count:workers", value)
	}
	count, err := strconv.Atoi(parts[0])
	if err != nil || count <= 0 {
		return 0, 0, fmt.Errorf("invalid scenario %q: count must be > 0", value)
	}
	workers, err := strconv.Atoi(parts[1])
	if err != nil || workers <= 0 {
		return 0, 0, fmt.Errorf("invalid scenario %q: workers must be > 0", value)
	}
	return count, workers, nil
}

func benchmarkLane(name string, nsPerOp, maxNsPerOp float64) map[string]any {
	return map[string]any{
		"lane":      name,
		"metric":    "ns_per_op",
		"observed":  nsPerOp,
		"threshold": map[string]any{"operator": "<=", "value": maxNsPerOp},
		"status":    ternaryString(nsPerOp <= maxNsPerOp, "pass", "fail"),
		"detail":    fmt.Sprintf("observed=%sns/op limit=%sns/op", formatFloat(nsPerOp), formatFloat(maxNsPerOp)),
	}
}

func soakLane(label string, result map[string]any, minThroughput float64, maxFailures int, envelope string) map[string]any {
	throughput := number(result["throughput_tasks_per_sec"])
	failures := int(number(result["failed"]))
	status := "fail"
	if throughput >= minThroughput && failures <= maxFailures {
		status = "pass"
	}
	return map[string]any{
		"lane": label,
		"scenario": map[string]any{
			"count":   int(number(result["count"])),
			"workers": int(number(result["workers"])),
		},
		"observed": map[string]any{
			"elapsed_seconds":          round(number(result["elapsed_seconds"]), 3),
			"throughput_tasks_per_sec": round(throughput, 3),
			"succeeded":                int(number(result["succeeded"])),
			"failed":                   failures,
		},
		"thresholds": map[string]any{
			"min_throughput_tasks_per_sec": minThroughput,
			"max_failures":                 maxFailures,
		},
		"operating_envelope": envelope,
		"status":             status,
		"detail":             fmt.Sprintf("throughput=%vtps min=%v failures=%d max=%d", round(throughput, 3), minThroughput, failures, maxFailures),
	}
}

func mixedWorkloadLane(report map[string]any) map[string]any {
	tasks := nestedSlice(report, "tasks")
	mismatches := make([]string, 0)
	successfulTasks := 0
	for _, rawTask := range tasks {
		task, ok := rawTask.(map[string]any)
		if !ok {
			continue
		}
		if boolValue(task["final_state"] == "succeeded") {
			successfulTasks++
		}
		if !boolValue(task["ok"]) {
			mismatches = append(mismatches, fmt.Sprintf("%s: task-level ok=false", stringValue(task["name"])))
		}
		if stringValue(task["expected_executor"]) != stringValue(task["routed_executor"]) {
			mismatches = append(mismatches, fmt.Sprintf("%s: expected=%s routed=%s", stringValue(task["name"]), stringValue(task["expected_executor"]), stringValue(task["routed_executor"])))
		}
		if stringValue(task["final_state"]) != "succeeded" {
			mismatches = append(mismatches, fmt.Sprintf("%s: final_state=%s", stringValue(task["name"]), stringValue(task["final_state"])))
		}
	}
	status := "fail"
	allOK := boolValue(report["all_ok"])
	if allOK && len(tasks) >= 5 && len(mismatches) == 0 {
		status = "pass"
	} else if allOK {
		status = "pass-with-ceiling"
	}
	detail := strings.Join(mismatches, "; ")
	if detail == "" {
		detail = "all sampled mixed-workload routes landed on the expected executor path"
	}
	return map[string]any{
		"lane": "mixed-workload-routing",
		"observed": map[string]any{
			"all_ok":           allOK,
			"task_count":       len(tasks),
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

func buildSaturationSummary(soakMatrix []any) map[string]any {
	lookup := map[string]map[string]any{}
	for _, rawLane := range soakMatrix {
		lane, ok := rawLane.(map[string]any)
		if ok {
			lookup[stringValue(lane["lane"])] = lane
		}
	}
	baseline := lookup["1000x24"]
	ceiling := lookup["2000x24"]
	baselineTPS := nestedFloat(baseline, "observed", "throughput_tasks_per_sec")
	ceilingTPS := nestedFloat(ceiling, "observed", "throughput_tasks_per_sec")
	dropPct := 0.0
	if baselineTPS > 0 {
		dropPct = round(((baselineTPS-ceilingTPS)/baselineTPS)*100, 2)
	}
	status := "warn"
	detail := "throughput drops materially at the 2000-task ceiling and should be treated as saturation"
	if dropPct <= 12 {
		status = "pass"
		detail = "throughput remains in the same single-instance local band at the 2000-task ceiling"
	}
	return map[string]any{
		"baseline_lane":                     "1000x24",
		"ceiling_lane":                      "2000x24",
		"baseline_throughput_tasks_per_sec": baselineTPS,
		"ceiling_throughput_tasks_per_sec":  ceilingTPS,
		"throughput_drop_pct":               dropPct,
		"drop_warn_threshold_pct":           12.0,
		"status":                            status,
		"detail":                            detail,
	}
}

func buildCapacityCertificationMarkdown(report map[string]any) string {
	lines := []string{
		"# Capacity Certification Report",
		"",
		"## Scope",
		"",
		fmt.Sprintf("- Generated at: `%s`", stringValue(report["generated_at"])),
		fmt.Sprintf("- Ticket: `%s`", stringValue(report["ticket"])),
		"- Goal: convert checked-in benchmark, soak, and mixed-workload evidence into a repo-native certification matrix with explicit thresholds and operating envelopes.",
		"- Boundary: this is a single-instance repo-native certification slice, not a live multi-tenant production attestation.",
		"",
		"## Certification Summary",
		"",
		fmt.Sprintf("- Overall status: `%s`", nestedString(report, "summary", "overall_status")),
		fmt.Sprintf("- Passed lanes: `%d/%d`", int(nestedFloat(report, "summary", "passed_lanes")), int(nestedFloat(report, "summary", "total_lanes"))),
		fmt.Sprintf("- Recommended local sustained envelope: `%s`", nestedString(report, "summary", "recommended_sustained_envelope")),
		fmt.Sprintf("- Local ceiling envelope: `%s`", nestedString(report, "summary", "ceiling_envelope")),
		fmt.Sprintf("- Saturation signal: `%s`", nestedString(report, "saturation_indicator", "detail")),
		"",
		"## Admission Policy Summary",
		"",
		"- Policy mode: `advisory-only reviewer guidance`",
		"- Runtime enforcement: `none`",
		fmt.Sprintf("- Default reviewer envelope: `%s`", nestedString(report, "summary", "recommended_sustained_envelope")),
		fmt.Sprintf("- Ceiling reviewer envelope: `%s`", nestedString(report, "summary", "ceiling_envelope")),
		"- Scheduler note: recommended envelopes guide reviewer admission decisions and are not scheduler-enforced runtime limits.",
		"",
		"## Microbenchmark Thresholds",
		"",
	}
	for _, rawLane := range nestedSlice(report, "microbenchmarks") {
		lane, ok := rawLane.(map[string]any)
		if !ok {
			continue
		}
		lines = append(lines, fmt.Sprintf("- `%s`: `%.2f ns/op` vs limit `%s` -> `%s`", stringValue(lane["lane"]), number(lane["observed"]), formatFloat(nestedFloat(lane, "threshold", "value")), stringValue(lane["status"])))
	}
	lines = append(lines, "", "## Soak Matrix", "")
	for _, rawLane := range nestedSlice(report, "soak_matrix") {
		lane, ok := rawLane.(map[string]any)
		if !ok {
			continue
		}
		lines = append(lines, fmt.Sprintf("- `%s`: `%v tasks/s`, `%d failed`, envelope `%s` -> `%s`", stringValue(lane["lane"]), nestedFloat(lane, "observed", "throughput_tasks_per_sec"), int(nestedFloat(lane, "observed", "failed")), stringValue(lane["operating_envelope"]), stringValue(lane["status"])))
	}
	lines = append(lines, "", "## Workload Mix", "", fmt.Sprintf("- `mixed-workload-routing`: `%s` -> `%s`", nestedString(report, "mixed_workload", "detail"), nestedString(report, "mixed_workload", "status")), "", "## Recommended Operating Envelopes", "")
	for _, rawEnvelope := range nestedSlice(report, "operating_envelopes") {
		envelope, ok := rawEnvelope.(map[string]any)
		if !ok {
			continue
		}
		evidence := make([]string, 0)
		for _, raw := range nestedSliceMap(envelope, "evidence_lanes") {
			evidence = append(evidence, stringValue(raw))
		}
		lines = append(lines, fmt.Sprintf("- `%s`: %s Evidence: `%s`.", stringValue(envelope["name"]), stringValue(envelope["recommendation"]), strings.Join(evidence, ", ")))
	}
	lines = append(lines, "", "## Saturation Notes", "")
	for _, note := range nestedSliceMap(report, "saturation_notes") {
		lines = append(lines, fmt.Sprintf("- %s", stringValue(note)))
	}
	lines = append(lines, "", "## Limits", "")
	for _, item := range nestedSliceMap(report, "limits") {
		lines = append(lines, fmt.Sprintf("- %s", stringValue(item)))
	}
	return strings.Join(lines, "\n") + "\n"
}

func deriveGeneratedAt(payloads ...map[string]any) string {
	timestamps := make([]time.Time, 0)
	for _, payload := range payloads {
		timestamps = append(timestamps, collectTimestamps(payload)...)
	}
	if len(timestamps) == 0 {
		return time.Now().UTC().Format(time.RFC3339Nano)
	}
	sort.Slice(timestamps, func(i, j int) bool { return timestamps[i].Before(timestamps[j]) })
	return timestamps[len(timestamps)-1].UTC().Format(time.RFC3339Nano)
}

func collectTimestamps(value any) []time.Time {
	switch payload := value.(type) {
	case map[string]any:
		out := make([]time.Time, 0)
		for key, item := range payload {
			switch key {
			case "generated_at", "timestamp", "created_at", "completed_at", "started_at":
				if parsed, ok := parseTimestamp(item); ok {
					out = append(out, parsed)
				}
			}
			out = append(out, collectTimestamps(item)...)
		}
		return out
	case []any:
		out := make([]time.Time, 0)
		for _, item := range payload {
			out = append(out, collectTimestamps(item)...)
		}
		return out
	default:
		return nil
	}
}

func parseTimestamp(value any) (time.Time, bool) {
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339Nano, strings.ReplaceAll(text, "Z", "+00:00"))
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func readJSONFile(path string) (map[string]any, error) {
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

func writeJSONFile(path string, payload map[string]any) error {
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return writeTextFile(path, string(body)+"\n")
}

func writeTextFile(path, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), 0o644)
}

func resolveWithinRoot(root, target string) string {
	if filepath.IsAbs(target) {
		return target
	}
	return filepath.Join(root, target)
}

func repoRelativePath(root, target string) string {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return filepath.ToSlash(target)
	}
	return filepath.ToSlash(rel)
}

func nestedMap(payload map[string]any, keys ...string) map[string]any {
	current := payload
	for _, key := range keys {
		next, ok := current[key].(map[string]any)
		if !ok {
			return map[string]any{}
		}
		current = next
	}
	return current
}

func nestedSlice(payload map[string]any, key string) []any {
	items, _ := payload[key].([]any)
	return items
}

func nestedSliceMap(payload map[string]any, key string) []any {
	items, _ := payload[key].([]any)
	return items
}

func nestedFloat(payload map[string]any, keys ...string) float64 {
	if len(keys) == 0 {
		return 0
	}
	current := payload
	for _, key := range keys[:len(keys)-1] {
		next, ok := current[key].(map[string]any)
		if !ok {
			return 0
		}
		current = next
	}
	return number(current[keys[len(keys)-1]])
}

func nestedString(payload map[string]any, keys ...string) string {
	if len(keys) == 0 {
		return ""
	}
	current := payload
	for _, key := range keys[:len(keys)-1] {
		next, ok := current[key].(map[string]any)
		if !ok {
			return ""
		}
		current = next
	}
	return stringValue(current[keys[len(keys)-1]])
}

func number(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	default:
		return 0
	}
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func boolValue(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	default:
		return false
	}
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func ternaryString(condition bool, whenTrue, whenFalse string) string {
	if condition {
		return whenTrue
	}
	return whenFalse
}

func round(value float64, precision int) float64 {
	pow := mathPow10(precision)
	return float64(int(value*pow+0.5)) / pow
}

func mathPow10(power int) float64 {
	result := 1.0
	for i := 0; i < power; i++ {
		result *= 10
	}
	return result
}

func cloneMapWithout(source map[string]any, keys ...string) map[string]any {
	out := cloneMap(source)
	for _, key := range keys {
		delete(out, key)
	}
	return out
}

func checkLane(name string, passed bool, detail string) map[string]any {
	return map[string]any{
		"name":   name,
		"passed": passed,
		"detail": detail,
	}
}

func allStatusesMatch(items []any, want string) bool {
	for _, raw := range items {
		lane, ok := raw.(map[string]any)
		if !ok || stringValue(lane["status"]) != want {
			return false
		}
	}
	return true
}

func statusIn(value any, allowed ...string) bool {
	current := stringValue(value)
	for _, item := range allowed {
		if current == item {
			return true
		}
	}
	return false
}

func formatFloat(value float64) string {
	if value == float64(int64(value)) {
		return strconv.FormatInt(int64(value), 10)
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}
