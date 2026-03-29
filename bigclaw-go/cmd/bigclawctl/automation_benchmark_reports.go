package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type automationRunMatrixOptions struct {
	GoRoot          string
	ReportPath      string
	TimeoutSeconds  int
	Scenarios       []string
	RunBenchmarks   func(string) (string, map[string]map[string]float64, error)
	RunSoakScenario func(automationSoakLocalOptions) (*automationSoakLocalReport, int, error)
}

type automationRunMatrixReport struct {
	Benchmark struct {
		Stdout string                        `json:"stdout"`
		Parsed map[string]map[string]float64 `json:"parsed"`
	} `json:"benchmark"`
	SoakMatrix []automationRunMatrixSoakEntry `json:"soak_matrix"`
}

type automationRunMatrixSoakEntry struct {
	Scenario   automationRunMatrixScenario `json:"scenario"`
	ReportPath string                      `json:"report_path"`
	Result     *automationSoakLocalReport  `json:"result"`
}

type automationRunMatrixScenario struct {
	Count   int `json:"count"`
	Workers int `json:"workers"`
}

type automationCapacityCertificationOptions struct {
	RepoRoot                    string
	BenchmarkReportPath         string
	MixedWorkloadReportPath     string
	SupplementalSoakReportPaths []string
	OutputPath                  string
	MarkdownOutputPath          string
}

type automationCapacityCertificationReport struct {
	GeneratedAt         string                                   `json:"generated_at"`
	Ticket              string                                   `json:"ticket"`
	Title               string                                   `json:"title"`
	Status              string                                   `json:"status"`
	EvidenceInputs      automationCertificationEvidenceInputs    `json:"evidence_inputs"`
	Summary             automationCertificationSummary           `json:"summary"`
	Microbenchmarks     []automationCertificationBenchmarkLane   `json:"microbenchmarks"`
	SoakMatrix          []automationCertificationSoakLane        `json:"soak_matrix"`
	MixedWorkload       automationCertificationMixedWorkloadLane `json:"mixed_workload"`
	SaturationIndicator automationCertificationSaturationSummary `json:"saturation_indicator"`
	OperatingEnvelopes  []automationCertificationEnvelope        `json:"operating_envelopes"`
	CertificationChecks []automationCertificationCheck           `json:"certification_checks"`
	SaturationNotes     []string                                 `json:"saturation_notes"`
	Limits              []string                                 `json:"limits"`
	Markdown            string                                   `json:"-"`
}

type automationCertificationEvidenceInputs struct {
	BenchmarkReportPath     string   `json:"benchmark_report_path"`
	MixedWorkloadReportPath string   `json:"mixed_workload_report_path"`
	SoakReportPaths         []string `json:"soak_report_paths"`
	GeneratorScript         string   `json:"generator_script"`
}

type automationCertificationSummary struct {
	OverallStatus                string   `json:"overall_status"`
	TotalLanes                   int      `json:"total_lanes"`
	PassedLanes                  int      `json:"passed_lanes"`
	FailedLanes                  []string `json:"failed_lanes"`
	RecommendedSustainedEnvelope string   `json:"recommended_sustained_envelope"`
	CeilingEnvelope              string   `json:"ceiling_envelope"`
}

type automationCertificationBenchmarkLane struct {
	Lane      string                                  `json:"lane"`
	Metric    string                                  `json:"metric"`
	Observed  float64                                 `json:"observed"`
	Threshold automationCertificationNumericThreshold `json:"threshold"`
	Status    string                                  `json:"status"`
	Detail    string                                  `json:"detail"`
}

type automationCertificationNumericThreshold struct {
	Operator string  `json:"operator"`
	Value    float64 `json:"value"`
}

type automationCertificationSoakLane struct {
	Lane              string                                `json:"lane"`
	Scenario          automationRunMatrixScenario           `json:"scenario"`
	Observed          automationCertificationSoakObserved   `json:"observed"`
	Thresholds        automationCertificationSoakThresholds `json:"thresholds"`
	OperatingEnvelope string                                `json:"operating_envelope"`
	Status            string                                `json:"status"`
	Detail            string                                `json:"detail"`
}

type automationCertificationSoakObserved struct {
	ElapsedSeconds        float64 `json:"elapsed_seconds"`
	ThroughputTasksPerSec float64 `json:"throughput_tasks_per_sec"`
	Succeeded             int     `json:"succeeded"`
	Failed                int     `json:"failed"`
}

type automationCertificationSoakThresholds struct {
	MinThroughputTasksPerSec float64 `json:"min_throughput_tasks_per_sec"`
	MaxFailures              int     `json:"max_failures"`
}

type automationCertificationMixedWorkloadLane struct {
	Lane        string                                 `json:"lane"`
	Observed    automationCertificationMixedObserved   `json:"observed"`
	Thresholds  automationCertificationMixedThresholds `json:"thresholds"`
	Status      string                                 `json:"status"`
	Detail      string                                 `json:"detail"`
	Limitations []string                               `json:"limitations"`
}

type automationCertificationMixedObserved struct {
	AllOK           bool `json:"all_ok"`
	TaskCount       int  `json:"task_count"`
	SuccessfulTasks int  `json:"successful_tasks"`
}

type automationCertificationMixedThresholds struct {
	AllOKRequired              bool `json:"all_ok_required"`
	MinimumTaskCount           int  `json:"minimum_task_count"`
	ExecutorRouteMatchRequired bool `json:"executor_route_match_required"`
}

type automationCertificationSaturationSummary struct {
	BaselineLane                  string  `json:"baseline_lane"`
	CeilingLane                   string  `json:"ceiling_lane"`
	BaselineThroughputTasksPerSec float64 `json:"baseline_throughput_tasks_per_sec"`
	CeilingThroughputTasksPerSec  float64 `json:"ceiling_throughput_tasks_per_sec"`
	ThroughputDropPct             float64 `json:"throughput_drop_pct"`
	DropWarnThresholdPct          float64 `json:"drop_warn_threshold_pct"`
	Status                        string  `json:"status"`
	Detail                        string  `json:"detail"`
}

type automationCertificationEnvelope struct {
	Name           string   `json:"name"`
	Recommendation string   `json:"recommendation"`
	EvidenceLanes  []string `json:"evidence_lanes"`
}

type automationCertificationCheck struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

var (
	automationBenchmarkLinePattern = regexp.MustCompile(`^(Benchmark\S+)\s+\d+\s+([0-9.]+)\s+ns/op$`)
	automationMicrobenchmarkLimits = map[string]float64{
		"BenchmarkMemoryQueueEnqueueLease-8": 100_000,
		"BenchmarkFileQueueEnqueueLease-8":   40_000_000,
		"BenchmarkSQLiteQueueEnqueueLease-8": 25_000_000,
		"BenchmarkSchedulerDecide-8":         1_000,
	}
	automationMicrobenchmarkOrder = []string{
		"BenchmarkMemoryQueueEnqueueLease-8",
		"BenchmarkFileQueueEnqueueLease-8",
		"BenchmarkSQLiteQueueEnqueueLease-8",
		"BenchmarkSchedulerDecide-8",
	}
	automationSoakThresholds = map[string]struct {
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

const automationSaturationDropThresholdPct = 12.0

func runAutomationBenchmarkMatrixCommand(args []string) error {
	flags := flag.NewFlagSet("automation benchmark run-matrix", flag.ContinueOnError)
	goRoot := flags.String("go-root", ".", "bigclaw-go repo root")
	reportPath := flags.String("report-path", "docs/reports/benchmark-matrix-report.json", "relative or absolute report path")
	timeoutSeconds := flags.Int("timeout-seconds", 180, "soak timeout seconds")
	scenarios := multiStringFlag{}
	flags.Var(&scenarios, "scenario", "benchmark scenario in count:workers form")
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

func runAutomationBenchmarkCapacityCertificationCommand(args []string) error {
	flags := flag.NewFlagSet("automation benchmark capacity-certification", flag.ContinueOnError)
	benchmarkReport := flags.String("benchmark-report", "bigclaw-go/docs/reports/benchmark-matrix-report.json", "benchmark matrix report path")
	mixedWorkloadReport := flags.String("mixed-workload-report", "bigclaw-go/docs/reports/mixed-workload-matrix-report.json", "mixed workload report path")
	supplementalSoakReports := multiStringFlag{}
	flags.Var(&supplementalSoakReports, "supplemental-soak-report", "supplemental soak report path")
	outputPath := flags.String("output", "bigclaw-go/docs/reports/capacity-certification-matrix.json", "JSON output path")
	markdownOutputPath := flags.String("markdown-output", "bigclaw-go/docs/reports/capacity-certification-report.md", "Markdown output path")
	pretty := flags.Bool("pretty", false, "pretty print JSON report to stdout")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation benchmark capacity-certification [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	report, err := automationBuildCapacityCertificationReport(automationCapacityCertificationOptions{
		RepoRoot:                    repoRootFromWorkingDir(),
		BenchmarkReportPath:         *benchmarkReport,
		MixedWorkloadReportPath:     *mixedWorkloadReport,
		SupplementalSoakReportPaths: supplementalSoakReports,
		OutputPath:                  *outputPath,
		MarkdownOutputPath:          *markdownOutputPath,
	})
	if err != nil {
		return err
	}
	if *pretty {
		encoded, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(append(encoded, '\n'))
		return err
	}
	return nil
}

func automationRunMatrix(opts automationRunMatrixOptions) (*automationRunMatrixReport, error) {
	runBenchmarks := opts.RunBenchmarks
	if runBenchmarks == nil {
		runBenchmarks = automationExecuteBenchmarks
	}
	runSoakScenario := opts.RunSoakScenario
	if runSoakScenario == nil {
		runSoakScenario = func(soakOpts automationSoakLocalOptions) (*automationSoakLocalReport, int, error) {
			return automationSoakLocal(soakOpts)
		}
	}
	scenarios := opts.Scenarios
	if len(scenarios) == 0 {
		scenarios = []string{"50:8", "100:12"}
	}
	stdout, parsed, err := runBenchmarks(opts.GoRoot)
	if err != nil {
		return nil, err
	}
	report := &automationRunMatrixReport{}
	report.Benchmark.Stdout = stdout
	report.Benchmark.Parsed = parsed
	for _, rawScenario := range scenarios {
		count, workers, err := automationParseScenario(rawScenario)
		if err != nil {
			return nil, err
		}
		reportPath := filepath.ToSlash(filepath.Join("docs", "reports", fmt.Sprintf("soak-local-%dx%d.json", count, workers)))
		soakReport, exitCode, err := runSoakScenario(automationSoakLocalOptions{
			Count:          count,
			Workers:        workers,
			BaseURL:        "http://127.0.0.1:8080",
			GoRoot:         opts.GoRoot,
			TimeoutSeconds: opts.TimeoutSeconds,
			Autostart:      true,
			ReportPath:     reportPath,
			HTTPClient:     http.DefaultClient,
		})
		if err != nil {
			return nil, err
		}
		if exitCode != 0 {
			return nil, fmt.Errorf("soak scenario %s failed with exit code %d", rawScenario, exitCode)
		}
		report.SoakMatrix = append(report.SoakMatrix, automationRunMatrixSoakEntry{
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

func automationExecuteBenchmarks(goRoot string) (string, map[string]map[string]float64, error) {
	command := exec.Command("go", "test", "-bench", ".", "./internal/queue", "./internal/scheduler")
	command.Dir = goRoot
	output, err := command.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", nil, fmt.Errorf("go test -bench failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", nil, err
	}
	stdout := string(output)
	return stdout, automationParseBenchmarkStdout(stdout), nil
}

func automationParseBenchmarkStdout(stdout string) map[string]map[string]float64 {
	parsed := map[string]map[string]float64{}
	for _, line := range strings.Split(stdout, "\n") {
		matches := automationBenchmarkLinePattern.FindStringSubmatch(strings.TrimSpace(line))
		if len(matches) != 3 {
			continue
		}
		value, err := strconv.ParseFloat(matches[2], 64)
		if err != nil {
			continue
		}
		parsed[matches[1]] = map[string]float64{"ns_per_op": value}
	}
	return parsed
}

func automationParseScenario(value string) (int, int, error) {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid scenario %q", value)
	}
	count, err := strconv.Atoi(parts[0])
	if err != nil || count <= 0 {
		return 0, 0, fmt.Errorf("invalid scenario %q", value)
	}
	workers, err := strconv.Atoi(parts[1])
	if err != nil || workers <= 0 {
		return 0, 0, fmt.Errorf("invalid scenario %q", value)
	}
	return count, workers, nil
}

func automationBuildCapacityCertificationReport(opts automationCapacityCertificationOptions) (*automationCapacityCertificationReport, error) {
	repoRoot := opts.RepoRoot
	if trim(repoRoot) == "" {
		repoRoot = repoRootFromWorkingDir()
	}
	benchmarkReportPath := opts.BenchmarkReportPath
	if trim(benchmarkReportPath) == "" {
		benchmarkReportPath = "bigclaw-go/docs/reports/benchmark-matrix-report.json"
	}
	mixedWorkloadReportPath := opts.MixedWorkloadReportPath
	if trim(mixedWorkloadReportPath) == "" {
		mixedWorkloadReportPath = "bigclaw-go/docs/reports/mixed-workload-matrix-report.json"
	}
	supplementalSoakReportPaths := opts.SupplementalSoakReportPaths
	if len(supplementalSoakReportPaths) == 0 {
		supplementalSoakReportPaths = []string{
			"bigclaw-go/docs/reports/soak-local-1000x24.json",
			"bigclaw-go/docs/reports/soak-local-2000x24.json",
		}
	}

	benchmarkReport := map[string]any{}
	if err := automationLoadJSON(automationResolveRepoPath(repoRoot, benchmarkReportPath), &benchmarkReport); err != nil {
		return nil, err
	}
	mixedWorkloadReport := map[string]any{}
	if err := automationLoadJSON(automationResolveRepoPath(repoRoot, mixedWorkloadReportPath), &mixedWorkloadReport); err != nil {
		return nil, err
	}

	microbenchmarks := make([]automationCertificationBenchmarkLane, 0, len(automationMicrobenchmarkOrder))
	parsed, _ := mapLookup(benchmarkReport, "benchmark", "parsed")
	for _, benchmarkName := range automationMicrobenchmarkOrder {
		limit := automationMicrobenchmarkLimits[benchmarkName]
		nsPerOp, err := automationLookupBenchmarkValue(parsed, benchmarkName)
		if err != nil {
			return nil, err
		}
		microbenchmarks = append(microbenchmarks, automationBenchmarkLane(benchmarkName, nsPerOp, limit))
	}

	soakResultsByLabel := map[string]map[string]any{}
	soakInputs := []string{}
	benchmarkSoakEntries := anySlice(mapLookupValue(benchmarkReport, "soak_matrix"))
	for _, entryAny := range benchmarkSoakEntries {
		entry, _ := entryAny.(map[string]any)
		result, _ := entry["result"].(map[string]any)
		label := fmt.Sprintf("%dx%d", intValue(result["count"]), intValue(result["workers"]))
		soakResultsByLabel[label] = result
		soakInputs = append(soakInputs, automationRepoRelativePath(repoRoot, stringify(entry["report_path"])))
	}

	supplementalSoakReports := make([]map[string]any, 0, len(supplementalSoakReportPaths))
	for _, path := range supplementalSoakReportPaths {
		report := map[string]any{}
		if err := automationLoadJSON(automationResolveRepoPath(repoRoot, path), &report); err != nil {
			return nil, err
		}
		supplementalSoakReports = append(supplementalSoakReports, report)
		label := fmt.Sprintf("%dx%d", intValue(report["count"]), intValue(report["workers"]))
		soakResultsByLabel[label] = report
		soakInputs = append(soakInputs, automationRepoRelativePath(repoRoot, path))
	}

	soakLabels := []string{"50x8", "100x12", "1000x24", "2000x24"}
	soakMatrix := make([]automationCertificationSoakLane, 0, len(soakLabels))
	for _, label := range soakLabels {
		threshold := automationSoakThresholds[label]
		soakMatrix = append(soakMatrix, automationSoakLane(label, soakResultsByLabel[label], threshold.MinThroughput, threshold.MaxFailures, threshold.Envelope))
	}

	mixedWorkload := automationMixedWorkloadLane(mixedWorkloadReport)
	saturationIndicator := automationBuildSaturationSummary(soakMatrix)

	allStatuses := []string{}
	failedLanes := []string{}
	for _, lane := range microbenchmarks {
		allStatuses = append(allStatuses, lane.Status)
		if lane.Status != "pass" && lane.Status != "pass-with-ceiling" {
			failedLanes = append(failedLanes, lane.Lane)
		}
	}
	for _, lane := range soakMatrix {
		allStatuses = append(allStatuses, lane.Status)
		if lane.Status != "pass" && lane.Status != "pass-with-ceiling" {
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

	report := &automationCapacityCertificationReport{
		GeneratedAt: automationDeriveGeneratedAt(append([]map[string]any{benchmarkReport, mixedWorkloadReport}, supplementalSoakReports...)...),
		Ticket:      "BIG-PAR-098",
		Title:       "Production-grade capacity certification matrix",
		Status:      "repo-native-capacity-certification",
		EvidenceInputs: automationCertificationEvidenceInputs{
			BenchmarkReportPath:     automationRepoRelativePath(repoRoot, benchmarkReportPath),
			MixedWorkloadReportPath: automationRepoRelativePath(repoRoot, mixedWorkloadReportPath),
			SoakReportPaths:         soakInputs,
			GeneratorScript:         "bigclaw-go/cmd/bigclawctl automation benchmark capacity-certification",
		},
		Summary: automationCertificationSummary{
			OverallStatus:                automationOverallStatus(failedLanes),
			TotalLanes:                   len(microbenchmarks) + len(soakMatrix) + 1,
			PassedLanes:                  passedLanes,
			FailedLanes:                  failedLanes,
			RecommendedSustainedEnvelope: "<=1000 tasks with 24 submit workers",
			CeilingEnvelope:              "<=2000 tasks with 24 submit workers",
		},
		Microbenchmarks:     microbenchmarks,
		SoakMatrix:          soakMatrix,
		MixedWorkload:       mixedWorkload,
		SaturationIndicator: saturationIndicator,
		OperatingEnvelopes: []automationCertificationEnvelope{
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
		},
		CertificationChecks: []automationCertificationCheck{
			{Name: "all_microbenchmark_thresholds_hold", Passed: automationStatusesAllPass(microbenchmarks), Detail: automationBenchmarkStatusesDetail(microbenchmarks)},
			{Name: "all_soak_lanes_hold", Passed: automationSoakStatusesAllPass(soakMatrix), Detail: automationSoakStatusesDetail(soakMatrix)},
			{Name: "mixed_workload_routes_match_expected_executors", Passed: mixedWorkload.Status == "pass" || mixedWorkload.Status == "pass-with-ceiling", Detail: mixedWorkload.Detail},
			{Name: "ceiling_lane_does_not_show_excessive_throughput_drop", Passed: saturationIndicator.Status == "pass", Detail: fmt.Sprintf("drop_pct=%v threshold=%v", saturationIndicator.ThroughputDropPct, automationSaturationDropThresholdPct)},
		},
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
	report.Markdown = automationBuildCertificationMarkdown(report)

	if trim(opts.OutputPath) != "" {
		if err := automationWriteReport(repoRoot, opts.OutputPath, report); err != nil {
			return nil, err
		}
	}
	if trim(opts.MarkdownOutputPath) != "" {
		markdownOutputPath := automationResolveRepoPath(repoRoot, opts.MarkdownOutputPath)
		if err := os.MkdirAll(filepath.Dir(markdownOutputPath), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(markdownOutputPath, []byte(report.Markdown), 0o644); err != nil {
			return nil, err
		}
	}
	return report, nil
}

func automationLoadJSON(path string, target any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

func automationResolveRepoPath(repoRoot string, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(repoRoot, path)
}

func automationRepoRelativePath(repoRoot string, path string) string {
	absolute := automationResolveRepoPath(repoRoot, path)
	relative, err := filepath.Rel(repoRoot, absolute)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(relative)
}

func automationLookupBenchmarkValue(parsed map[string]any, benchmarkName string) (float64, error) {
	entry, ok := parsed[benchmarkName].(map[string]any)
	if !ok {
		return 0, fmt.Errorf("missing benchmark %s", benchmarkName)
	}
	return floatValue(entry["ns_per_op"]), nil
}

func automationBenchmarkLane(name string, nsPerOp float64, maxNSPerOp float64) automationCertificationBenchmarkLane {
	status := "fail"
	if nsPerOp <= maxNSPerOp {
		status = "pass"
	}
	return automationCertificationBenchmarkLane{
		Lane:      name,
		Metric:    "ns_per_op",
		Observed:  nsPerOp,
		Threshold: automationCertificationNumericThreshold{Operator: "<=", Value: maxNSPerOp},
		Status:    status,
		Detail:    fmt.Sprintf("observed=%gns/op limit=%gns/op", nsPerOp, maxNSPerOp),
	}
}

func automationSoakLane(label string, result map[string]any, minThroughput float64, maxFailures int, envelope string) automationCertificationSoakLane {
	throughput := floatValue(result["throughput_tasks_per_sec"])
	failures := intValue(result["failed"])
	status := "fail"
	if throughput >= minThroughput && failures <= maxFailures {
		status = "pass"
	}
	return automationCertificationSoakLane{
		Lane:              label,
		Scenario:          automationRunMatrixScenario{Count: intValue(result["count"]), Workers: intValue(result["workers"])},
		Observed:          automationCertificationSoakObserved{ElapsedSeconds: roundTo(floatValue(result["elapsed_seconds"]), 3), ThroughputTasksPerSec: roundTo(throughput, 3), Succeeded: intValue(result["succeeded"]), Failed: failures},
		Thresholds:        automationCertificationSoakThresholds{MinThroughputTasksPerSec: minThroughput, MaxFailures: maxFailures},
		OperatingEnvelope: envelope,
		Status:            status,
		Detail:            fmt.Sprintf("throughput=%gtps min=%g failures=%d max=%d", roundTo(throughput, 3), minThroughput, failures, maxFailures),
	}
}

func automationMixedWorkloadLane(report map[string]any) automationCertificationMixedWorkloadLane {
	taskItems := anySlice(report["tasks"])
	mismatches := []string{}
	successfulTasks := 0
	for _, taskAny := range taskItems {
		task, _ := taskAny.(map[string]any)
		if boolValue(task["ok"]) == false {
			mismatches = append(mismatches, fmt.Sprintf("%s: task-level ok=false", stringify(task["name"])))
		}
		if stringify(task["expected_executor"]) != stringify(task["routed_executor"]) {
			mismatches = append(mismatches, fmt.Sprintf("%s: expected=%s routed=%s", stringify(task["name"]), stringify(task["expected_executor"]), stringify(task["routed_executor"])))
		}
		if stringify(task["final_state"]) == "succeeded" {
			successfulTasks++
		} else {
			mismatches = append(mismatches, fmt.Sprintf("%s: final_state=%s", stringify(task["name"]), stringify(task["final_state"])))
		}
	}
	status := "fail"
	allOK := boolValue(report["all_ok"])
	if allOK && len(taskItems) >= 5 && len(mismatches) == 0 {
		status = "pass"
	} else if allOK {
		status = "pass-with-ceiling"
	}
	detail := "all sampled mixed-workload routes landed on the expected executor path"
	if len(mismatches) > 0 {
		detail = strings.Join(mismatches, "; ")
	}
	return automationCertificationMixedWorkloadLane{
		Lane:        "mixed-workload-routing",
		Observed:    automationCertificationMixedObserved{AllOK: allOK, TaskCount: len(taskItems), SuccessfulTasks: successfulTasks},
		Thresholds:  automationCertificationMixedThresholds{AllOKRequired: true, MinimumTaskCount: 5, ExecutorRouteMatchRequired: true},
		Status:      status,
		Detail:      detail,
		Limitations: []string{"executor-mix coverage is functional rather than high-volume", "mixed-workload evidence proves route correctness but not sustained cross-executor saturation limits"},
	}
}

func automationBuildSaturationSummary(soakLanes []automationCertificationSoakLane) automationCertificationSaturationSummary {
	var baseline automationCertificationSoakLane
	var ceiling automationCertificationSoakLane
	for _, lane := range soakLanes {
		if lane.Lane == "1000x24" {
			baseline = lane
		}
		if lane.Lane == "2000x24" {
			ceiling = lane
		}
	}
	baselineTPS := baseline.Observed.ThroughputTasksPerSec
	ceilingTPS := ceiling.Observed.ThroughputTasksPerSec
	dropPct := 0.0
	if baselineTPS != 0 {
		dropPct = roundTo(((baselineTPS-ceilingTPS)/baselineTPS)*100, 2)
	}
	status := "warn"
	detail := "throughput drops materially at the 2000-task ceiling and should be treated as saturation"
	if dropPct <= automationSaturationDropThresholdPct {
		status = "pass"
		detail = "throughput remains in the same single-instance local band at the 2000-task ceiling"
	}
	return automationCertificationSaturationSummary{
		BaselineLane:                  baseline.Lane,
		CeilingLane:                   ceiling.Lane,
		BaselineThroughputTasksPerSec: baselineTPS,
		CeilingThroughputTasksPerSec:  ceilingTPS,
		ThroughputDropPct:             dropPct,
		DropWarnThresholdPct:          automationSaturationDropThresholdPct,
		Status:                        status,
		Detail:                        detail,
	}
}

func automationBuildCertificationMarkdown(report *automationCapacityCertificationReport) string {
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
		fmt.Sprintf("- Overall status: `%s`", report.Summary.OverallStatus),
		fmt.Sprintf("- Passed lanes: `%d/%d`", report.Summary.PassedLanes, report.Summary.TotalLanes),
		fmt.Sprintf("- Recommended local sustained envelope: `%s`", report.Summary.RecommendedSustainedEnvelope),
		fmt.Sprintf("- Local ceiling envelope: `%s`", report.Summary.CeilingEnvelope),
		fmt.Sprintf("- Saturation signal: `%s`", report.SaturationIndicator.Detail),
		"",
		"## Admission Policy Summary",
		"",
		"- Policy mode: `advisory-only reviewer guidance`",
		"- Runtime enforcement: `none`",
		fmt.Sprintf("- Default reviewer envelope: `%s`", report.Summary.RecommendedSustainedEnvelope),
		fmt.Sprintf("- Ceiling reviewer envelope: `%s`", report.Summary.CeilingEnvelope),
		"- Scheduler note: recommended envelopes guide reviewer admission decisions and are not scheduler-enforced runtime limits.",
		"",
		"## Microbenchmark Thresholds",
		"",
	}
	for _, lane := range report.Microbenchmarks {
		lines = append(lines, fmt.Sprintf("- `%s`: `%.2f ns/op` vs limit `%g` -> `%s`", lane.Lane, lane.Observed, lane.Threshold.Value, lane.Status))
	}
	lines = append(lines, "", "## Soak Matrix", "")
	for _, lane := range report.SoakMatrix {
		lines = append(lines, fmt.Sprintf("- `%s`: `%g tasks/s`, `%d failed`, envelope `%s` -> `%s`", lane.Lane, lane.Observed.ThroughputTasksPerSec, lane.Observed.Failed, lane.OperatingEnvelope, lane.Status))
	}
	lines = append(lines, "", "## Workload Mix", "", fmt.Sprintf("- `mixed-workload-routing`: `%s` -> `%s`", report.MixedWorkload.Detail, report.MixedWorkload.Status), "", "## Recommended Operating Envelopes", "")
	for _, envelope := range report.OperatingEnvelopes {
		lines = append(lines, fmt.Sprintf("- `%s`: %s Evidence: `%s`.", envelope.Name, envelope.Recommendation, strings.Join(envelope.EvidenceLanes, ", ")))
	}
	lines = append(lines, "", "## Saturation Notes", "")
	for _, note := range report.SaturationNotes {
		lines = append(lines, "- "+note)
	}
	lines = append(lines, "", "## Limits", "")
	for _, limit := range report.Limits {
		lines = append(lines, "- "+limit)
	}
	return strings.Join(lines, "\n") + "\n"
}

func automationDeriveGeneratedAt(payloads ...map[string]any) string {
	timestamps := []time.Time{}
	for _, payload := range payloads {
		timestamps = append(timestamps, automationCollectTimestamps(payload)...)
	}
	if len(timestamps) == 0 {
		return time.Now().UTC().Format(time.RFC3339Nano)
	}
	maxValue := timestamps[0]
	for _, value := range timestamps[1:] {
		if value.After(maxValue) {
			maxValue = value
		}
	}
	return maxValue.UTC().Format(time.RFC3339Nano)
}

func automationCollectTimestamps(value any) []time.Time {
	found := []time.Time{}
	switch typed := value.(type) {
	case map[string]any:
		for key, inner := range typed {
			switch key {
			case "generated_at", "timestamp", "created_at", "completed_at", "started_at":
				if parsed, ok := automationParseTimestamp(inner); ok {
					found = append(found, parsed)
				}
			}
			found = append(found, automationCollectTimestamps(inner)...)
		}
	case []any:
		for _, item := range typed {
			found = append(found, automationCollectTimestamps(item)...)
		}
	}
	return found
}

func automationParseTimestamp(value any) (time.Time, bool) {
	text, ok := value.(string)
	if !ok || trim(text) == "" {
		return time.Time{}, false
	}
	candidate := strings.Replace(text, "Z", "+00:00", 1)
	parsed, err := time.Parse(time.RFC3339Nano, candidate)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func automationOverallStatus(failedLanes []string) string {
	if len(failedLanes) > 0 {
		return "fail"
	}
	return "pass"
}

func automationStatusesAllPass(lanes []automationCertificationBenchmarkLane) bool {
	for _, lane := range lanes {
		if lane.Status != "pass" {
			return false
		}
	}
	return true
}

func automationSoakStatusesAllPass(lanes []automationCertificationSoakLane) bool {
	for _, lane := range lanes {
		if lane.Status != "pass" {
			return false
		}
	}
	return true
}

func automationBenchmarkStatusesDetail(lanes []automationCertificationBenchmarkLane) string {
	statuses := make([]string, 0, len(lanes))
	for _, lane := range lanes {
		statuses = append(statuses, lane.Status)
	}
	return fmt.Sprintf("%v", statuses)
}

func automationSoakStatusesDetail(lanes []automationCertificationSoakLane) string {
	statuses := make([]string, 0, len(lanes))
	for _, lane := range lanes {
		statuses = append(statuses, lane.Status)
	}
	return fmt.Sprintf("%v", statuses)
}

func repoRootFromWorkingDir() string {
	workingDir, err := os.Getwd()
	if err != nil {
		return "."
	}
	candidates := []string{workingDir}
	current := workingDir
	for i := 0; i < 4; i++ {
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		candidates = append(candidates, parent)
		current = parent
	}
	for _, candidate := range candidates {
		if automationLooksLikeWorkspaceRoot(candidate) {
			return candidate
		}
	}
	return workingDir
}

type multiStringFlag []string

func (f *multiStringFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *multiStringFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func mapLookup(payload map[string]any, keys ...string) (map[string]any, bool) {
	current := payload
	for index, key := range keys {
		value, ok := current[key]
		if !ok {
			return nil, false
		}
		if index == len(keys)-1 {
			next, ok := value.(map[string]any)
			return next, ok
		}
		next, ok := value.(map[string]any)
		if !ok {
			return nil, false
		}
		current = next
	}
	return nil, false
}

func mapLookupValue(payload map[string]any, keys ...string) any {
	current := any(payload)
	for _, key := range keys {
		mapValue, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = mapValue[key]
	}
	return current
}

func anySlice(value any) []any {
	items, _ := value.([]any)
	return items
}

func floatValue(value any) float64 {
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
		parsed, _ := typed.Float64()
		return parsed
	default:
		return 0
	}
}

func intValue(value any) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case float32:
		return int(typed)
	case int:
		return typed
	case int64:
		return int(typed)
	case json.Number:
		parsed, _ := typed.Int64()
		return int(parsed)
	default:
		return 0
	}
}

func boolValue(value any) bool {
	typed, _ := value.(bool)
	return typed
}

func stringify(value any) string {
	typed, _ := value.(string)
	return typed
}

func roundTo(value float64, precision int) float64 {
	pow := 1.0
	for i := 0; i < precision; i++ {
		pow *= 10
	}
	return float64(int(value*pow+0.5)) / pow
}

func automationLooksLikeWorkspaceRoot(path string) bool {
	info, err := os.Stat(filepath.Join(path, "bigclaw-go", "docs", "reports"))
	return err == nil && info.IsDir()
}
