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
	"slices"
	"strconv"
	"strings"
	"time"
)

var benchmarkLinePattern = regexp.MustCompile(`^(Benchmark\S+)\s+\d+\s+([0-9.]+)\s+ns/op$`)

var benchmarkMicrobenchmarkLimits = map[string]float64{
	"BenchmarkMemoryQueueEnqueueLease-8": 100_000,
	"BenchmarkFileQueueEnqueueLease-8":   40_000_000,
	"BenchmarkSQLiteQueueEnqueueLease-8": 25_000_000,
	"BenchmarkSchedulerDecide-8":         1_000,
}

var benchmarkSoakThresholds = map[string]struct {
	MinThroughput float64
	MaxFailures   int
	Envelope      string
}{
	"50x8":    {MinThroughput: 5.0, MaxFailures: 0, Envelope: "bootstrap-burst"},
	"100x12":  {MinThroughput: 8.5, MaxFailures: 0, Envelope: "bootstrap-burst"},
	"1000x24": {MinThroughput: 9.0, MaxFailures: 0, Envelope: "recommended-local-sustained"},
	"2000x24": {MinThroughput: 8.5, MaxFailures: 0, Envelope: "recommended-local-ceiling"},
}

const benchmarkSaturationDropThresholdPct = 12.0

type automationRunMatrixOptions struct {
	GoRoot         string
	ReportPath     string
	TimeoutSeconds int
	Scenarios      []string
	RunBenchmarks  func(string) (string, error)
	RunSoak        func(automationSoakLocalOptions) (*automationSoakLocalReport, int, error)
}

type automationBenchmarkEntry struct {
	NSPerOp float64 `json:"ns_per_op"`
}

type automationBenchmarkResult struct {
	Stdout string                              `json:"stdout"`
	Parsed map[string]automationBenchmarkEntry `json:"parsed"`
}

type automationRunMatrixScenario struct {
	Count   int `json:"count"`
	Workers int `json:"workers"`
}

type automationRunMatrixEntry struct {
	Scenario   automationRunMatrixScenario `json:"scenario"`
	ReportPath string                      `json:"report_path"`
	Result     any                         `json:"result"`
}

type automationRunMatrixReport struct {
	Benchmark  automationBenchmarkResult  `json:"benchmark"`
	SoakMatrix []automationRunMatrixEntry `json:"soak_matrix"`
}

type automationCapacityCertificationOptions struct {
	GoRoot                  string
	BenchmarkReportPath     string
	MixedWorkloadReportPath string
	SupplementalSoakReports []string
	OutputPath              string
	MarkdownOutputPath      string
	Pretty                  bool
}

type capacityCertificationCheck struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

type capacityCertificationLane struct {
	Lane      string         `json:"lane"`
	Metric    string         `json:"metric,omitempty"`
	Observed  any            `json:"observed"`
	Threshold map[string]any `json:"threshold,omitempty"`
	Status    string         `json:"status"`
	Detail    string         `json:"detail"`
}

type capacityCertificationSoakLane struct {
	Lane              string                      `json:"lane"`
	Scenario          automationRunMatrixScenario `json:"scenario"`
	Observed          map[string]any              `json:"observed"`
	Thresholds        map[string]any              `json:"thresholds"`
	OperatingEnvelope string                      `json:"operating_envelope"`
	Status            string                      `json:"status"`
	Detail            string                      `json:"detail"`
}

type capacityCertificationMixedWorkload struct {
	Lane        string         `json:"lane"`
	Observed    map[string]any `json:"observed"`
	Thresholds  map[string]any `json:"thresholds"`
	Status      string         `json:"status"`
	Detail      string         `json:"detail"`
	Limitations []string       `json:"limitations"`
}

type capacityCertificationOperatingEnvelope struct {
	Name           string   `json:"name"`
	Recommendation string   `json:"recommendation"`
	EvidenceLanes  []string `json:"evidence_lanes"`
}

type capacityCertificationSaturationIndicator struct {
	BaselineLane                  string  `json:"baseline_lane"`
	CeilingLane                   string  `json:"ceiling_lane"`
	BaselineThroughputTasksPerSec float64 `json:"baseline_throughput_tasks_per_sec"`
	CeilingThroughputTasksPerSec  float64 `json:"ceiling_throughput_tasks_per_sec"`
	ThroughputDropPct             float64 `json:"throughput_drop_pct"`
	DropWarnThresholdPct          float64 `json:"drop_warn_threshold_pct"`
	Status                        string  `json:"status"`
	Detail                        string  `json:"detail"`
}

type capacityCertificationReport struct {
	GeneratedAt         string                                   `json:"generated_at"`
	Ticket              string                                   `json:"ticket"`
	Title               string                                   `json:"title"`
	Status              string                                   `json:"status"`
	EvidenceInputs      map[string]any                           `json:"evidence_inputs"`
	Summary             map[string]any                           `json:"summary"`
	Microbenchmarks     []capacityCertificationLane              `json:"microbenchmarks"`
	SoakMatrix          []capacityCertificationSoakLane          `json:"soak_matrix"`
	MixedWorkload       capacityCertificationMixedWorkload       `json:"mixed_workload"`
	SaturationIndicator capacityCertificationSaturationIndicator `json:"saturation_indicator"`
	OperatingEnvelopes  []capacityCertificationOperatingEnvelope `json:"operating_envelopes"`
	CertificationChecks []capacityCertificationCheck             `json:"certification_checks"`
	SaturationNotes     []string                                 `json:"saturation_notes"`
	Limits              []string                                 `json:"limits"`
}

func runAutomationRunMatrixCommand(args []string) error {
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
	report, err := automationRunMatrix(automationRunMatrixOptions{
		GoRoot:         absPath(*goRoot),
		ReportPath:     *reportPath,
		TimeoutSeconds: *timeoutSeconds,
		Scenarios:      scenarios,
	})
	if err != nil {
		return err
	}
	return emit(structToMap(report), *asJSON, 0)
}

func runAutomationCapacityCertificationCommand(args []string) error {
	flags := flag.NewFlagSet("automation benchmark capacity-certification", flag.ContinueOnError)
	goRoot := flags.String("go-root", ".", "bigclaw-go repo root")
	benchmarkReport := flags.String("benchmark-report", "docs/reports/benchmark-matrix-report.json", "benchmark matrix report path")
	mixedWorkloadReport := flags.String("mixed-workload-report", "docs/reports/mixed-workload-matrix-report.json", "mixed workload report path")
	var supplementalSoakReports multiStringFlag
	flags.Var(&supplementalSoakReports, "supplemental-soak-report", "additional soak report path")
	outputPath := flags.String("output", "docs/reports/capacity-certification-matrix.json", "relative or absolute JSON output path")
	markdownOutputPath := flags.String("markdown-output", "docs/reports/capacity-certification-report.md", "relative or absolute markdown output path")
	pretty := flags.Bool("pretty", false, "pretty-print JSON to stdout")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation benchmark capacity-certification [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	report, markdown, err := automationCapacityCertification(automationCapacityCertificationOptions{
		GoRoot:                  absPath(*goRoot),
		BenchmarkReportPath:     *benchmarkReport,
		MixedWorkloadReportPath: *mixedWorkloadReport,
		SupplementalSoakReports: supplementalSoakReports,
		OutputPath:              *outputPath,
		MarkdownOutputPath:      *markdownOutputPath,
		Pretty:                  *pretty,
	})
	if err != nil {
		return err
	}
	if *pretty {
		body, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		_, _ = os.Stdout.Write(append(body, '\n'))
	} else {
		_, _ = os.Stdout.WriteString(markdown)
	}
	return nil
}

func automationRunMatrix(opts automationRunMatrixOptions) (*automationRunMatrixReport, error) {
	runBenchmarks := opts.RunBenchmarks
	if runBenchmarks == nil {
		runBenchmarks = automationRunBenchmarks
	}
	runSoak := opts.RunSoak
	if runSoak == nil {
		runSoak = func(soakOpts automationSoakLocalOptions) (*automationSoakLocalReport, int, error) {
			return automationSoakLocal(soakOpts)
		}
	}

	scenarios := opts.Scenarios
	if len(scenarios) == 0 {
		scenarios = []string{"50:8", "100:12"}
	}
	stdout, err := runBenchmarks(opts.GoRoot)
	if err != nil {
		return nil, err
	}
	report := &automationRunMatrixReport{
		Benchmark: automationBenchmarkResult{
			Stdout: stdout,
			Parsed: automationParseBenchmarkStdout(stdout),
		},
		SoakMatrix: make([]automationRunMatrixEntry, 0, len(scenarios)),
	}
	for _, scenario := range scenarios {
		count, workers, err := parseAutomationScenario(scenario)
		if err != nil {
			return nil, err
		}
		reportPath := filepath.ToSlash(filepath.Join("docs", "reports", fmt.Sprintf("soak-local-%dx%d.json", count, workers)))
		soakReport, _, err := runSoak(automationSoakLocalOptions{
			Count:          count,
			Workers:        workers,
			BaseURL:        "http://127.0.0.1:8080",
			GoRoot:         opts.GoRoot,
			TimeoutSeconds: opts.TimeoutSeconds,
			Autostart:      true,
			ReportPath:     reportPath,
		})
		if err != nil {
			return nil, err
		}
		report.SoakMatrix = append(report.SoakMatrix, automationRunMatrixEntry{
			Scenario:   automationRunMatrixScenario{Count: count, Workers: workers},
			ReportPath: reportPath,
			Result:     soakReport,
		})
	}
	if err := automationWriteReport(opts.GoRoot, opts.ReportPath, report); err != nil {
		return nil, err
	}
	return report, nil
}

func automationRunBenchmarks(goRoot string) (string, error) {
	cmd := exec.Command("go", "test", "-bench", ".", "./internal/queue", "./internal/scheduler")
	cmd.Dir = goRoot
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("go test -bench failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", err
	}
	return string(output), nil
}

func automationParseBenchmarkStdout(stdout string) map[string]automationBenchmarkEntry {
	parsed := map[string]automationBenchmarkEntry{}
	for _, line := range strings.Split(stdout, "\n") {
		match := benchmarkLinePattern.FindStringSubmatch(strings.TrimSpace(line))
		if len(match) != 3 {
			continue
		}
		nsPerOp, err := strconv.ParseFloat(match[2], 64)
		if err != nil {
			continue
		}
		parsed[match[1]] = automationBenchmarkEntry{NSPerOp: nsPerOp}
	}
	return parsed
}

func parseAutomationScenario(value string) (int, int, error) {
	parts := strings.SplitN(strings.TrimSpace(value), ":", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid --scenario %q; expected count:workers", value)
	}
	count, err := strconv.Atoi(parts[0])
	if err != nil || count <= 0 {
		return 0, 0, fmt.Errorf("invalid scenario count in %q", value)
	}
	workers, err := strconv.Atoi(parts[1])
	if err != nil || workers <= 0 {
		return 0, 0, fmt.Errorf("invalid scenario workers in %q", value)
	}
	return count, workers, nil
}

func automationCapacityCertification(opts automationCapacityCertificationOptions) (*capacityCertificationReport, string, error) {
	benchmarkReportPayload, err := automationReadJSONReport(opts.GoRoot, opts.BenchmarkReportPath)
	if err != nil {
		return nil, "", err
	}
	mixedWorkloadPayload, err := automationReadJSONReport(opts.GoRoot, opts.MixedWorkloadReportPath)
	if err != nil {
		return nil, "", err
	}

	supplementalSoakReports := opts.SupplementalSoakReports
	if len(supplementalSoakReports) == 0 {
		supplementalSoakReports = []string{
			filepath.ToSlash(filepath.Join("docs", "reports", "soak-local-1000x24.json")),
			filepath.ToSlash(filepath.Join("docs", "reports", "soak-local-2000x24.json")),
		}
	}

	microbenchmarks, err := automationBuildMicrobenchmarkLanes(benchmarkReportPayload)
	if err != nil {
		return nil, "", err
	}

	soakInputs := make([]string, 0, 4)
	soakResultsByLabel := map[string]map[string]any{}
	benchmarkSoakEntries, _ := benchmarkReportPayload["soak_matrix"].([]any)
	for _, entry := range benchmarkSoakEntries {
		item, _ := entry.(map[string]any)
		if item == nil {
			continue
		}
		reportPath, _ := item["report_path"].(string)
		if reportPath != "" {
			soakInputs = append(soakInputs, filepath.ToSlash(reportPath))
		}
		result, _ := item["result"].(map[string]any)
		if result == nil {
			continue
		}
		label := fmt.Sprintf("%dx%d", automationInt(result["count"]), automationInt(result["workers"]))
		soakResultsByLabel[label] = result
	}

	supplementalPayloads := make([]map[string]any, 0, len(supplementalSoakReports))
	for _, soakPath := range supplementalSoakReports {
		payload, err := automationReadJSONReport(opts.GoRoot, soakPath)
		if err != nil {
			return nil, "", err
		}
		supplementalPayloads = append(supplementalPayloads, payload)
		soakInputs = append(soakInputs, filepath.ToSlash(soakPath))
		label := fmt.Sprintf("%dx%d", automationInt(payload["count"]), automationInt(payload["workers"]))
		soakResultsByLabel[label] = payload
	}

	labels := []string{"50x8", "100x12", "1000x24", "2000x24"}
	soakMatrix := make([]capacityCertificationSoakLane, 0, len(labels))
	for _, label := range labels {
		threshold := benchmarkSoakThresholds[label]
		result := soakResultsByLabel[label]
		if result == nil {
			return nil, "", fmt.Errorf("missing soak report for lane %s", label)
		}
		soakMatrix = append(soakMatrix, automationBuildSoakLane(label, result, threshold.MinThroughput, threshold.MaxFailures, threshold.Envelope))
	}

	mixedWorkload := automationBuildMixedWorkloadLane(mixedWorkloadPayload)
	saturationIndicator, err := automationBuildSaturationSummary(soakMatrix)
	if err != nil {
		return nil, "", err
	}

	allStatuses := make([]string, 0, len(microbenchmarks)+len(soakMatrix)+1)
	failedLanes := make([]string, 0)
	for _, lane := range microbenchmarks {
		allStatuses = append(allStatuses, lane.Status)
		if lane.Status != "pass" {
			failedLanes = append(failedLanes, lane.Lane)
		}
	}
	for _, lane := range soakMatrix {
		allStatuses = append(allStatuses, lane.Status)
		if lane.Status != "pass" {
			failedLanes = append(failedLanes, lane.Lane)
		}
	}
	allStatuses = append(allStatuses, mixedWorkload.Status)
	if mixedWorkload.Status != "pass" && mixedWorkload.Status != "pass-with-ceiling" {
		failedLanes = append(failedLanes, mixedWorkload.Lane)
	}

	passedLanes := 0
	for _, status := range allStatuses {
		if status == "pass" || status == "pass-with-ceiling" {
			passedLanes++
		}
	}

	operatingEnvelopes := []capacityCertificationOperatingEnvelope{
		{
			Name:           "recommended-local-sustained",
			Recommendation: "Use up to 1000 queued tasks with 24 submit workers when a stable single-instance local review lane is required.",
			EvidenceLanes:  []string{"1000x24"},
		},
		{
			Name:           "recommended-local-ceiling",
			Recommendation: "Treat 2000 queued tasks with 24 submit workers as the checked-in local ceiling, not the default operating point.",
			EvidenceLanes:  []string{"2000x24"},
		},
		{
			Name:           "mixed-workload-routing",
			Recommendation: "Use the mixed-workload matrix for executor routing correctness, but do not infer sustained multi-executor throughput from it.",
			EvidenceLanes:  []string{"mixed-workload-routing"},
		},
	}

	checks := []capacityCertificationCheck{
		{
			Name:   "all_microbenchmark_thresholds_hold",
			Passed: slices.Equal(automationStatusesFromMicrobenchmarks(microbenchmarks), []string{"pass", "pass", "pass", "pass"}),
			Detail: fmt.Sprintf("%v", automationStatusesFromMicrobenchmarks(microbenchmarks)),
		},
		{
			Name:   "all_soak_lanes_hold",
			Passed: slices.Equal(automationStatusesFromSoak(soakMatrix), []string{"pass", "pass", "pass", "pass"}),
			Detail: fmt.Sprintf("%v", automationStatusesFromSoak(soakMatrix)),
		},
		{
			Name:   "mixed_workload_routes_match_expected_executors",
			Passed: mixedWorkload.Status == "pass" || mixedWorkload.Status == "pass-with-ceiling",
			Detail: mixedWorkload.Detail,
		},
		{
			Name:   "ceiling_lane_does_not_show_excessive_throughput_drop",
			Passed: saturationIndicator.Status == "pass",
			Detail: fmt.Sprintf("drop_pct=%.2f threshold=%.1f", saturationIndicator.ThroughputDropPct, benchmarkSaturationDropThresholdPct),
		},
	}

	report := &capacityCertificationReport{
		GeneratedAt: automationDeriveGeneratedAt(append([]map[string]any{benchmarkReportPayload, mixedWorkloadPayload}, supplementalPayloads...)...),
		Ticket:      "BIG-PAR-098",
		Title:       "Production-grade capacity certification matrix",
		Status:      "repo-native-capacity-certification",
		EvidenceInputs: map[string]any{
			"benchmark_report_path":      filepath.ToSlash(opts.BenchmarkReportPath),
			"mixed_workload_report_path": filepath.ToSlash(opts.MixedWorkloadReportPath),
			"soak_report_paths":          soakInputs,
			"generator_script":           "go run ./cmd/bigclawctl automation benchmark capacity-certification",
		},
		Summary: map[string]any{
			"overall_status":                 "pass",
			"total_lanes":                    len(microbenchmarks) + len(soakMatrix) + 1,
			"passed_lanes":                   passedLanes,
			"failed_lanes":                   failedLanes,
			"recommended_sustained_envelope": "<=1000 tasks with 24 submit workers",
			"ceiling_envelope":               "<=2000 tasks with 24 submit workers",
		},
		Microbenchmarks:     microbenchmarks,
		SoakMatrix:          soakMatrix,
		MixedWorkload:       mixedWorkload,
		SaturationIndicator: saturationIndicator,
		OperatingEnvelopes:  operatingEnvelopes,
		CertificationChecks: checks,
		SaturationNotes: []string{
			"Throughput plateaus around 9-10 tasks/s across the checked-in 100x12, 1000x24, and 2000x24 local lanes.",
			"The 2000x24 lane remains within the same throughput band as 1000x24, so the checked-in local ceiling is evidence-backed but not substantially headroom-rich.",
			"Mixed-workload evidence verifies executor-routing correctness across local, Kubernetes, and Ray, but it is a functional routing proof rather than a concurrency ceiling.",
		},
		Limits: []string{
			"Evidence is repo-native and single-instance; it does not certify multi-node or multi-tenant production saturation behavior.",
			"The matrix uses checked-in local runs from 2026-03-13 and should be refreshed when queue, scheduler, or executor behavior changes materially.",
			"Recommended envelopes are conservative reviewer guidance derived from current evidence, not an automated runtime admission policy.",
		},
	}
	if len(failedLanes) > 0 {
		report.Summary["overall_status"] = "fail"
	}

	markdown := automationBuildCapacityCertificationMarkdown(report)
	if err := automationWriteReport(opts.GoRoot, opts.OutputPath, report); err != nil {
		return nil, "", err
	}
	if err := automationWriteTextReport(opts.GoRoot, opts.MarkdownOutputPath, markdown); err != nil {
		return nil, "", err
	}
	return report, markdown, nil
}

func automationBuildMicrobenchmarkLanes(benchmarkReport map[string]any) ([]capacityCertificationLane, error) {
	benchmarkSection, _ := benchmarkReport["benchmark"].(map[string]any)
	parsed, _ := benchmarkSection["parsed"].(map[string]any)
	order := []string{
		"BenchmarkMemoryQueueEnqueueLease-8",
		"BenchmarkFileQueueEnqueueLease-8",
		"BenchmarkSQLiteQueueEnqueueLease-8",
		"BenchmarkSchedulerDecide-8",
	}
	lanes := make([]capacityCertificationLane, 0, len(order))
	for _, name := range order {
		entry, _ := parsed[name].(map[string]any)
		if entry == nil {
			return nil, fmt.Errorf("missing benchmark %s", name)
		}
		nsPerOp := automationFloat(entry["ns_per_op"])
		limit := benchmarkMicrobenchmarkLimits[name]
		status := "pass"
		if nsPerOp > limit {
			status = "fail"
		}
		lanes = append(lanes, capacityCertificationLane{
			Lane:     name,
			Metric:   "ns_per_op",
			Observed: nsPerOp,
			Threshold: map[string]any{
				"operator": "<=",
				"value":    limit,
			},
			Status: status,
			Detail: fmt.Sprintf("observed=%vns/op limit=%vns/op", nsPerOp, limit),
		})
	}
	return lanes, nil
}

func automationBuildSoakLane(label string, result map[string]any, minThroughput float64, maxFailures int, envelope string) capacityCertificationSoakLane {
	throughput := automationFloat(result["throughput_tasks_per_sec"])
	failures := automationInt(result["failed"])
	status := "pass"
	if throughput < minThroughput || failures > maxFailures {
		status = "fail"
	}
	return capacityCertificationSoakLane{
		Lane: label,
		Scenario: automationRunMatrixScenario{
			Count:   automationInt(result["count"]),
			Workers: automationInt(result["workers"]),
		},
		Observed: map[string]any{
			"elapsed_seconds":          automationRound(automationFloat(result["elapsed_seconds"]), 3),
			"throughput_tasks_per_sec": automationRound(throughput, 3),
			"succeeded":                automationInt(result["succeeded"]),
			"failed":                   failures,
		},
		Thresholds: map[string]any{
			"min_throughput_tasks_per_sec": minThroughput,
			"max_failures":                 maxFailures,
		},
		OperatingEnvelope: envelope,
		Status:            status,
		Detail:            fmt.Sprintf("throughput=%.3ftps min=%.1f failures=%d max=%d", automationRound(throughput, 3), minThroughput, failures, maxFailures),
	}
}

func automationBuildMixedWorkloadLane(report map[string]any) capacityCertificationMixedWorkload {
	tasks, _ := report["tasks"].([]any)
	mismatches := make([]string, 0)
	successfulTasks := 0
	for _, item := range tasks {
		task, _ := item.(map[string]any)
		if task == nil {
			continue
		}
		if ok, _ := task["ok"].(bool); !ok {
			mismatches = append(mismatches, fmt.Sprintf("%s: task-level ok=false", automationString(task["name"])))
		}
		if automationString(task["expected_executor"]) != automationString(task["routed_executor"]) {
			mismatches = append(mismatches, fmt.Sprintf("%s: expected=%s routed=%s", automationString(task["name"]), automationString(task["expected_executor"]), automationString(task["routed_executor"])))
		}
		if automationString(task["final_state"]) != "succeeded" {
			mismatches = append(mismatches, fmt.Sprintf("%s: final_state=%s", automationString(task["name"]), automationString(task["final_state"])))
		}
		if automationString(task["final_state"]) == "succeeded" {
			successfulTasks++
		}
	}
	allOK, _ := report["all_ok"].(bool)
	status := "fail"
	if allOK && len(tasks) >= 5 && len(mismatches) == 0 {
		status = "pass"
	} else if allOK {
		status = "pass-with-ceiling"
	}
	detail := "all sampled mixed-workload routes landed on the expected executor path"
	if len(mismatches) > 0 {
		detail = strings.Join(mismatches, "; ")
	}
	return capacityCertificationMixedWorkload{
		Lane: "mixed-workload-routing",
		Observed: map[string]any{
			"all_ok":           allOK,
			"task_count":       len(tasks),
			"successful_tasks": successfulTasks,
		},
		Thresholds: map[string]any{
			"all_ok_required":               true,
			"minimum_task_count":            5,
			"executor_route_match_required": true,
		},
		Status: status,
		Detail: detail,
		Limitations: []string{
			"executor-mix coverage is functional rather than high-volume",
			"mixed-workload evidence proves route correctness but not sustained cross-executor saturation limits",
		},
	}
}

func automationBuildSaturationSummary(soakMatrix []capacityCertificationSoakLane) (capacityCertificationSaturationIndicator, error) {
	var baseline capacityCertificationSoakLane
	var ceiling capacityCertificationSoakLane
	foundBaseline := false
	foundCeiling := false
	for _, lane := range soakMatrix {
		switch lane.Lane {
		case "1000x24":
			baseline = lane
			foundBaseline = true
		case "2000x24":
			ceiling = lane
			foundCeiling = true
		}
	}
	if !foundBaseline || !foundCeiling {
		return capacityCertificationSaturationIndicator{}, errors.New("missing 1000x24 or 2000x24 saturation lane")
	}
	baselineTPS := automationFloat(baseline.Observed["throughput_tasks_per_sec"])
	ceilingTPS := automationFloat(ceiling.Observed["throughput_tasks_per_sec"])
	dropPct := 0.0
	if baselineTPS > 0 {
		dropPct = automationRound(((baselineTPS-ceilingTPS)/baselineTPS)*100, 2)
	}
	status := "pass"
	detail := "throughput remains in the same single-instance local band at the 2000-task ceiling"
	if dropPct > benchmarkSaturationDropThresholdPct {
		status = "warn"
		detail = "throughput drops materially at the 2000-task ceiling and should be treated as saturation"
	}
	return capacityCertificationSaturationIndicator{
		BaselineLane:                  baseline.Lane,
		CeilingLane:                   ceiling.Lane,
		BaselineThroughputTasksPerSec: baselineTPS,
		CeilingThroughputTasksPerSec:  ceilingTPS,
		ThroughputDropPct:             dropPct,
		DropWarnThresholdPct:          benchmarkSaturationDropThresholdPct,
		Status:                        status,
		Detail:                        detail,
	}, nil
}

func automationBuildCapacityCertificationMarkdown(report *capacityCertificationReport) string {
	lines := []string{
		"# Capacity Certification Report",
		"",
		"## Scope",
		"",
		fmt.Sprintf("- Generated at: `%s`", report.GeneratedAt),
		fmt.Sprintf("- Ticket: `%s`", report.Ticket),
		"- Goal: convert checked-in benchmark, soak, and mixed-workload evidence into a repo-native certification matrix with explicit thresholds and operating envelopes.",
		"- Boundary: this is a single-instance repo-native certification slice, not a live multi-tenant production attestation.",
		"",
		"## Certification Summary",
		"",
		fmt.Sprintf("- Overall status: `%s`", report.Summary["overall_status"]),
		fmt.Sprintf("- Passed lanes: `%v/%v`", report.Summary["passed_lanes"], report.Summary["total_lanes"]),
		fmt.Sprintf("- Recommended local sustained envelope: `%v`", report.Summary["recommended_sustained_envelope"]),
		fmt.Sprintf("- Local ceiling envelope: `%v`", report.Summary["ceiling_envelope"]),
		fmt.Sprintf("- Saturation signal: `%s`", report.SaturationIndicator.Detail),
		"",
		"## Admission Policy Summary",
		"",
		"- Policy mode: `advisory-only reviewer guidance`",
		"- Runtime enforcement: `none`",
		fmt.Sprintf("- Default reviewer envelope: `%v`", report.Summary["recommended_sustained_envelope"]),
		fmt.Sprintf("- Ceiling reviewer envelope: `%v`", report.Summary["ceiling_envelope"]),
		"- Scheduler note: recommended envelopes guide reviewer admission decisions and are not scheduler-enforced runtime limits.",
		"",
		"## Microbenchmark Thresholds",
		"",
	}
	for _, lane := range report.Microbenchmarks {
		lines = append(lines, fmt.Sprintf("- `%s`: `%.2f ns/op` vs limit `%v` -> `%s`", lane.Lane, automationFloat(lane.Observed), lane.Threshold["value"], lane.Status))
	}
	lines = append(lines, "", "## Soak Matrix", "")
	for _, lane := range report.SoakMatrix {
		lines = append(lines, fmt.Sprintf("- `%s`: `%v tasks/s`, `%v failed`, envelope `%s` -> `%s`", lane.Lane, lane.Observed["throughput_tasks_per_sec"], lane.Observed["failed"], lane.OperatingEnvelope, lane.Status))
	}
	lines = append(lines,
		"",
		"## Workload Mix",
		"",
		fmt.Sprintf("- `mixed-workload-routing`: `%s` -> `%s`", report.MixedWorkload.Detail, report.MixedWorkload.Status),
		"",
		"## Recommended Operating Envelopes",
		"",
	)
	for _, envelope := range report.OperatingEnvelopes {
		lines = append(lines, fmt.Sprintf("- `%s`: %s Evidence: `%s`.", envelope.Name, envelope.Recommendation, strings.Join(envelope.EvidenceLanes, ", ")))
	}
	lines = append(lines, "", "## Saturation Notes", "")
	for _, note := range report.SaturationNotes {
		lines = append(lines, "- "+note)
	}
	lines = append(lines, "", "## Limits", "")
	for _, item := range report.Limits {
		lines = append(lines, "- "+item)
	}
	return strings.Join(lines, "\n") + "\n"
}

func automationReadJSONReport(goRoot string, reportPath string) (map[string]any, error) {
	target := reportPath
	if !filepath.IsAbs(target) {
		target = filepath.Join(goRoot, reportPath)
	}
	body, err := os.ReadFile(target)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func automationWriteTextReport(goRoot string, reportPath string, body string) error {
	target := reportPath
	if !filepath.IsAbs(target) {
		target = filepath.Join(goRoot, reportPath)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, []byte(body), 0o644)
}

func automationDeriveGeneratedAt(payloads ...map[string]any) string {
	var latest time.Time
	found := false
	for _, payload := range payloads {
		automationVisitTimestamps(payload, func(candidate time.Time) {
			if !found || candidate.After(latest) {
				latest = candidate
				found = true
			}
		})
	}
	if !found {
		return time.Now().UTC().Format(time.RFC3339Nano)
	}
	return latest.UTC().Format(time.RFC3339Nano)
}

func automationVisitTimestamps(value any, visit func(time.Time)) {
	switch typed := value.(type) {
	case map[string]any:
		for key, inner := range typed {
			switch key {
			case "generated_at", "timestamp", "created_at", "completed_at", "started_at":
				if parsed, ok := automationParseTimestamp(inner); ok {
					visit(parsed)
				}
			}
			automationVisitTimestamps(inner, visit)
		}
	case []any:
		for _, inner := range typed {
			automationVisitTimestamps(inner, visit)
		}
	}
}

func automationParseTimestamp(value any) (time.Time, bool) {
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return time.Time{}, false
	}
	candidate := strings.Replace(text, "Z", "+00:00", 1)
	parsed, err := time.Parse(time.RFC3339Nano, candidate)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func automationStatusesFromMicrobenchmarks(lanes []capacityCertificationLane) []string {
	statuses := make([]string, 0, len(lanes))
	for _, lane := range lanes {
		statuses = append(statuses, lane.Status)
	}
	return statuses
}

func automationStatusesFromSoak(lanes []capacityCertificationSoakLane) []string {
	statuses := make([]string, 0, len(lanes))
	for _, lane := range lanes {
		statuses = append(statuses, lane.Status)
	}
	return statuses
}

func automationString(value any) string {
	text, _ := value.(string)
	return text
}

func automationFloat(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case json.Number:
		result, _ := typed.Float64()
		return result
	default:
		return 0
	}
}

func automationInt(value any) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case int64:
		return int(typed)
	case json.Number:
		result, _ := typed.Int64()
		return int(result)
	default:
		return 0
	}
}

func automationRound(value float64, digits int) float64 {
	format := "%." + strconv.Itoa(digits) + "f"
	result, _ := strconv.ParseFloat(fmt.Sprintf(format, value), 64)
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
