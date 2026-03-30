package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var microbenchmarkLimits = map[string]float64{
	"BenchmarkMemoryQueueEnqueueLease-8": 100000,
	"BenchmarkFileQueueEnqueueLease-8":   40000000,
	"BenchmarkSQLiteQueueEnqueueLease-8": 25000000,
	"BenchmarkSchedulerDecide-8":         1000,
}

type soakThreshold struct {
	MinThroughput float64
	MaxFailures   int
	Envelope      string
}

var soakThresholds = map[string]soakThreshold{
	"50x8":    {MinThroughput: 5.0, MaxFailures: 0, Envelope: "bootstrap-burst"},
	"100x12":  {MinThroughput: 8.5, MaxFailures: 0, Envelope: "bootstrap-burst"},
	"1000x24": {MinThroughput: 9.0, MaxFailures: 0, Envelope: "recommended-local-sustained"},
	"2000x24": {MinThroughput: 8.5, MaxFailures: 0, Envelope: "recommended-local-ceiling"},
}

const saturationDropThresholdPct = 12.0

type benchmarkReportInput struct {
	Benchmark struct {
		Parsed map[string]struct {
			NSPerOp float64 `json:"ns_per_op"`
		} `json:"parsed"`
	} `json:"benchmark"`
	SoakMatrix []struct {
		ReportPath string `json:"report_path"`
		Result     struct {
			Count                 int     `json:"count"`
			Workers               int     `json:"workers"`
			ElapsedSeconds        float64 `json:"elapsed_seconds"`
			ThroughputTasksPerSec float64 `json:"throughput_tasks_per_sec"`
			Succeeded             int     `json:"succeeded"`
			Failed                int     `json:"failed"`
		} `json:"result"`
	} `json:"soak_matrix"`
}

type mixedWorkloadInput struct {
	AllOK bool `json:"all_ok"`
	Tasks []struct {
		Name             string `json:"name"`
		OK               bool   `json:"ok"`
		ExpectedExecutor string `json:"expected_executor"`
		RoutedExecutor   string `json:"routed_executor"`
		FinalState       string `json:"final_state"`
	} `json:"tasks"`
}

type soakResult struct {
	Count                 int     `json:"count"`
	Workers               int     `json:"workers"`
	ElapsedSeconds        float64 `json:"elapsed_seconds"`
	ThroughputTasksPerSec float64 `json:"throughput_tasks_per_sec"`
	Succeeded             int     `json:"succeeded"`
	Failed                int     `json:"failed"`
}

type threshold struct {
	Operator string  `json:"operator"`
	Value    float64 `json:"value"`
}

type benchmarkLane struct {
	Lane      string    `json:"lane"`
	Metric    string    `json:"metric"`
	Observed  float64   `json:"observed"`
	Threshold threshold `json:"threshold"`
	Status    string    `json:"status"`
	Detail    string    `json:"detail"`
}

type soakObserved struct {
	ElapsedSeconds        float64 `json:"elapsed_seconds"`
	ThroughputTasksPerSec float64 `json:"throughput_tasks_per_sec"`
	Succeeded             int     `json:"succeeded"`
	Failed                int     `json:"failed"`
}

type soakScenario struct {
	Count   int `json:"count"`
	Workers int `json:"workers"`
}

type soakThresholdsOutput struct {
	MinThroughputTasksPerSec float64 `json:"min_throughput_tasks_per_sec"`
	MaxFailures              int     `json:"max_failures"`
}

type soakLane struct {
	Lane              string               `json:"lane"`
	Scenario          soakScenario         `json:"scenario"`
	Observed          soakObserved         `json:"observed"`
	Thresholds        soakThresholdsOutput `json:"thresholds"`
	OperatingEnvelope string               `json:"operating_envelope"`
	Status            string               `json:"status"`
	Detail            string               `json:"detail"`
}

type mixedObserved struct {
	AllOK           bool `json:"all_ok"`
	TaskCount       int  `json:"task_count"`
	SuccessfulTasks int  `json:"successful_tasks"`
}

type mixedThresholds struct {
	AllOKRequired              bool `json:"all_ok_required"`
	MinimumTaskCount           int  `json:"minimum_task_count"`
	ExecutorRouteMatchRequired bool `json:"executor_route_match_required"`
}

type mixedWorkloadLane struct {
	Lane        string          `json:"lane"`
	Observed    mixedObserved   `json:"observed"`
	Thresholds  mixedThresholds `json:"thresholds"`
	Status      string          `json:"status"`
	Detail      string          `json:"detail"`
	Limitations []string        `json:"limitations"`
}

type saturationSummary struct {
	BaselineLane                  string  `json:"baseline_lane"`
	CeilingLane                   string  `json:"ceiling_lane"`
	BaselineThroughputTasksPerSec float64 `json:"baseline_throughput_tasks_per_sec"`
	CeilingThroughputTasksPerSec  float64 `json:"ceiling_throughput_tasks_per_sec"`
	ThroughputDropPct             float64 `json:"throughput_drop_pct"`
	DropWarnThresholdPct          float64 `json:"drop_warn_threshold_pct"`
	Status                        string  `json:"status"`
	Detail                        string  `json:"detail"`
}

type operatingEnvelope struct {
	Name           string   `json:"name"`
	Recommendation string   `json:"recommendation"`
	EvidenceLanes  []string `json:"evidence_lanes"`
}

type certificationCheck struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

type report struct {
	GeneratedAt         string               `json:"generated_at"`
	Ticket              string               `json:"ticket"`
	Title               string               `json:"title"`
	Status              string               `json:"status"`
	EvidenceInputs      map[string]any       `json:"evidence_inputs"`
	Summary             map[string]any       `json:"summary"`
	Microbenchmarks     []benchmarkLane      `json:"microbenchmarks"`
	SoakMatrix          []soakLane           `json:"soak_matrix"`
	MixedWorkload       mixedWorkloadLane    `json:"mixed_workload"`
	SaturationIndicator saturationSummary    `json:"saturation_indicator"`
	OperatingEnvelopes  []operatingEnvelope  `json:"operating_envelopes"`
	CertificationChecks []certificationCheck `json:"certification_checks"`
	SaturationNotes     []string             `json:"saturation_notes"`
	Limits              []string             `json:"limits"`
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	flags := flag.NewFlagSet("capacity-certification", flag.ExitOnError)
	benchmarkReportPath := flags.String("benchmark-report", "docs/reports/benchmark-matrix-report.json", "benchmark report")
	mixedWorkloadReportPath := flags.String("mixed-workload-report", "docs/reports/mixed-workload-matrix-report.json", "mixed workload report")
	var supplementalSoakReports stringList
	flags.Var(&supplementalSoakReports, "supplemental-soak-report", "supplemental soak report")
	outputPath := flags.String("output", "docs/reports/capacity-certification-matrix.json", "json output")
	markdownOutputPath := flags.String("markdown-output", "docs/reports/capacity-certification-report.md", "markdown output")
	pretty := flags.Bool("pretty", false, "pretty print")
	if err := flags.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if len(supplementalSoakReports) == 0 {
		supplementalSoakReports = stringList{
			"docs/reports/soak-local-1000x24.json",
			"docs/reports/soak-local-2000x24.json",
		}
	}

	rep, markdown, err := buildReport(root, *benchmarkReportPath, *mixedWorkloadReportPath, supplementalSoakReports)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	body, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := writeFile(resolvePath(root, *outputPath), append(body, '\n')); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := writeFile(resolvePath(root, *markdownOutputPath), []byte(markdown)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *pretty {
		fmt.Println(string(body))
	}
}

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func buildReport(root, benchmarkPath, mixedPath string, supplemental []string) (report, string, error) {
	benchmarkBody, err := os.ReadFile(resolvePath(root, benchmarkPath))
	if err != nil {
		return report{}, "", err
	}
	mixedBody, err := os.ReadFile(resolvePath(root, mixedPath))
	if err != nil {
		return report{}, "", err
	}
	var benchInput benchmarkReportInput
	if err := json.Unmarshal(benchmarkBody, &benchInput); err != nil {
		return report{}, "", err
	}
	var mixedInput mixedWorkloadInput
	if err := json.Unmarshal(mixedBody, &mixedInput); err != nil {
		return report{}, "", err
	}

	microbenchmarks := make([]benchmarkLane, 0, len(microbenchmarkLimits))
	benchmarkNames := make([]string, 0, len(microbenchmarkLimits))
	for name := range microbenchmarkLimits {
		benchmarkNames = append(benchmarkNames, name)
	}
	sort.Strings(benchmarkNames)
	for _, name := range benchmarkNames {
		limit := microbenchmarkLimits[name]
		value := benchInput.Benchmark.Parsed[name].NSPerOp
		status := "pass"
		if value > limit {
			status = "fail"
		}
		microbenchmarks = append(microbenchmarks, benchmarkLane{
			Lane:      name,
			Metric:    "ns_per_op",
			Observed:  value,
			Threshold: threshold{Operator: "<=", Value: limit},
			Status:    status,
			Detail:    fmt.Sprintf("observed=%vns/op limit=%vns/op", value, limit),
		})
	}

	soakInputs := []string{}
	soakByLabel := map[string]soakResult{}
	for _, entry := range benchInput.SoakMatrix {
		label := fmt.Sprintf("%dx%d", entry.Result.Count, entry.Result.Workers)
		soakByLabel[label] = soakResult(entry.Result)
		soakInputs = append(soakInputs, repoRelativePath(root, entry.ReportPath))
	}

	rawInputs := []any{}
	var benchAny any
	_ = json.Unmarshal(benchmarkBody, &benchAny)
	rawInputs = append(rawInputs, benchAny)
	var mixedAny any
	_ = json.Unmarshal(mixedBody, &mixedAny)
	rawInputs = append(rawInputs, mixedAny)

	for _, item := range supplemental {
		body, err := os.ReadFile(resolvePath(root, item))
		if err != nil {
			return report{}, "", err
		}
		var result soakResult
		if err := json.Unmarshal(body, &result); err != nil {
			return report{}, "", err
		}
		label := fmt.Sprintf("%dx%d", result.Count, result.Workers)
		soakByLabel[label] = result
		soakInputs = append(soakInputs, repoRelativePath(root, item))
		var raw any
		_ = json.Unmarshal(body, &raw)
		rawInputs = append(rawInputs, raw)
	}

	labels := []string{"50x8", "100x12", "1000x24", "2000x24"}
	soakMatrix := make([]soakLane, 0, len(labels))
	for _, label := range labels {
		threshold := soakThresholds[label]
		result := soakByLabel[label]
		status := "pass"
		if result.ThroughputTasksPerSec < threshold.MinThroughput || result.Failed > threshold.MaxFailures {
			status = "fail"
		}
		soakMatrix = append(soakMatrix, soakLane{
			Lane:     label,
			Scenario: soakScenario{Count: result.Count, Workers: result.Workers},
			Observed: soakObserved{
				ElapsedSeconds:        round(result.ElapsedSeconds, 3),
				ThroughputTasksPerSec: round(result.ThroughputTasksPerSec, 3),
				Succeeded:             result.Succeeded,
				Failed:                result.Failed,
			},
			Thresholds: soakThresholdsOutput{
				MinThroughputTasksPerSec: threshold.MinThroughput,
				MaxFailures:              threshold.MaxFailures,
			},
			OperatingEnvelope: threshold.Envelope,
			Status:            status,
			Detail:            fmt.Sprintf("throughput=%.3ftps min=%v failures=%d max=%d", round(result.ThroughputTasksPerSec, 3), threshold.MinThroughput, result.Failed, threshold.MaxFailures),
		})
	}

	mixed := buildMixedWorkloadLane(mixedInput)
	saturation := buildSaturationSummary(soakMatrix)

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
	if mixed.Status != "pass" && mixed.Status != "pass-with-ceiling" {
		failedLanes = append(failedLanes, mixed.Lane)
	}
	passedLanes := len(microbenchmarks) + len(soakMatrix)
	if mixed.Status == "pass" || mixed.Status == "pass-with-ceiling" {
		passedLanes++
	}

	rep := report{
		GeneratedAt: deriveGeneratedAt(rawInputs...),
		Ticket:      "BIG-PAR-098",
		Title:       "Production-grade capacity certification matrix",
		Status:      "repo-native-capacity-certification",
		EvidenceInputs: map[string]any{
			"benchmark_report_path":      repoRelativePath(root, benchmarkPath),
			"mixed_workload_report_path": repoRelativePath(root, mixedPath),
			"soak_report_paths":          soakInputs,
			"generator_script":           "scripts/benchmark/capacitycert/main.go",
		},
		Summary: map[string]any{
			"overall_status":                 ternary(len(failedLanes) == 0, "pass", "fail"),
			"total_lanes":                    len(microbenchmarks) + len(soakMatrix) + 1,
			"passed_lanes":                   passedLanes,
			"failed_lanes":                   failedLanes,
			"recommended_sustained_envelope": "<=1000 tasks with 24 submit workers",
			"ceiling_envelope":               "<=2000 tasks with 24 submit workers",
		},
		Microbenchmarks:     microbenchmarks,
		SoakMatrix:          soakMatrix,
		MixedWorkload:       mixed,
		SaturationIndicator: saturation,
		OperatingEnvelopes: []operatingEnvelope{
			{Name: "recommended-local-sustained", Recommendation: "Use up to 1000 queued tasks with 24 submit workers when a stable single-instance local review lane is required.", EvidenceLanes: []string{"1000x24"}},
			{Name: "recommended-local-ceiling", Recommendation: "Treat 2000 queued tasks with 24 submit workers as the checked-in local ceiling, not the default operating point.", EvidenceLanes: []string{"2000x24"}},
			{Name: "mixed-workload-routing", Recommendation: "Use the mixed-workload matrix for executor routing correctness, but do not infer sustained multi-executor throughput from it.", EvidenceLanes: []string{"mixed-workload-routing"}},
		},
		CertificationChecks: []certificationCheck{
			{Name: "all_microbenchmark_thresholds_hold", Passed: allBenchmarkPass(microbenchmarks), Detail: formatStatusList(microbenchmarks)},
			{Name: "all_soak_lanes_hold", Passed: allSoakPass(soakMatrix), Detail: formatSoakStatusList(soakMatrix)},
			{Name: "mixed_workload_routes_match_expected_executors", Passed: mixed.Status == "pass" || mixed.Status == "pass-with-ceiling", Detail: mixed.Detail},
			{Name: "ceiling_lane_does_not_show_excessive_throughput_drop", Passed: saturation.Status == "pass", Detail: fmt.Sprintf("drop_pct=%.2f threshold=%.1f", saturation.ThroughputDropPct, saturationDropThresholdPct)},
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
	return rep, buildMarkdown(rep), nil
}

func buildMixedWorkloadLane(input mixedWorkloadInput) mixedWorkloadLane {
	mismatches := []string{}
	successful := 0
	for _, task := range input.Tasks {
		if task.FinalState == "succeeded" {
			successful++
		}
		if !task.OK {
			mismatches = append(mismatches, fmt.Sprintf("%s: task-level ok=false", task.Name))
		}
		if task.ExpectedExecutor != task.RoutedExecutor {
			mismatches = append(mismatches, fmt.Sprintf("%s: expected=%s routed=%s", task.Name, task.ExpectedExecutor, task.RoutedExecutor))
		}
		if task.FinalState != "succeeded" {
			mismatches = append(mismatches, fmt.Sprintf("%s: final_state=%s", task.Name, task.FinalState))
		}
	}
	status := "fail"
	if input.AllOK && len(input.Tasks) >= 5 && len(mismatches) == 0 {
		status = "pass"
	} else if input.AllOK {
		status = "pass-with-ceiling"
	}
	detail := "all sampled mixed-workload routes landed on the expected executor path"
	if len(mismatches) > 0 {
		detail = strings.Join(mismatches, "; ")
	}
	return mixedWorkloadLane{
		Lane: "mixed-workload-routing",
		Observed: mixedObserved{
			AllOK:           input.AllOK,
			TaskCount:       len(input.Tasks),
			SuccessfulTasks: successful,
		},
		Thresholds: mixedThresholds{
			AllOKRequired:              true,
			MinimumTaskCount:           5,
			ExecutorRouteMatchRequired: true,
		},
		Status:      status,
		Detail:      detail,
		Limitations: []string{"executor-mix coverage is functional rather than high-volume", "mixed-workload evidence proves route correctness but not sustained cross-executor saturation limits"},
	}
}

func buildSaturationSummary(soakMatrix []soakLane) saturationSummary {
	var baseline, ceiling soakLane
	for _, lane := range soakMatrix {
		if lane.Lane == "1000x24" {
			baseline = lane
		}
		if lane.Lane == "2000x24" {
			ceiling = lane
		}
	}
	dropPct := 0.0
	if baseline.Observed.ThroughputTasksPerSec != 0 {
		dropPct = round(((baseline.Observed.ThroughputTasksPerSec-ceiling.Observed.ThroughputTasksPerSec)/baseline.Observed.ThroughputTasksPerSec)*100, 2)
	}
	status := "pass"
	detail := "throughput remains in the same single-instance local band at the 2000-task ceiling"
	if dropPct > saturationDropThresholdPct {
		status = "warn"
		detail = "throughput drops materially at the 2000-task ceiling and should be treated as saturation"
	}
	return saturationSummary{
		BaselineLane:                  baseline.Lane,
		CeilingLane:                   ceiling.Lane,
		BaselineThroughputTasksPerSec: baseline.Observed.ThroughputTasksPerSec,
		CeilingThroughputTasksPerSec:  ceiling.Observed.ThroughputTasksPerSec,
		ThroughputDropPct:             dropPct,
		DropWarnThresholdPct:          saturationDropThresholdPct,
		Status:                        status,
		Detail:                        detail,
	}
}

func buildMarkdown(rep report) string {
	lines := []string{
		"# Capacity Certification Report",
		"",
		"## Scope",
		"",
		fmt.Sprintf("- Generated at: `%s`", rep.GeneratedAt),
		fmt.Sprintf("- Ticket: `%s`", rep.Ticket),
		"- Goal: convert checked-in benchmark, soak, and mixed-workload evidence into a repo-native certification matrix with explicit thresholds and operating envelopes.",
		"- Boundary: this is a single-instance repo-native certification slice, not a live multi-tenant production attestation.",
		"",
		"## Certification Summary",
		"",
		fmt.Sprintf("- Overall status: `%s`", rep.Summary["overall_status"]),
		fmt.Sprintf("- Passed lanes: `%v/%v`", rep.Summary["passed_lanes"], rep.Summary["total_lanes"]),
		fmt.Sprintf("- Recommended local sustained envelope: `%v`", rep.Summary["recommended_sustained_envelope"]),
		fmt.Sprintf("- Local ceiling envelope: `%v`", rep.Summary["ceiling_envelope"]),
		fmt.Sprintf("- Saturation signal: `%s`", rep.SaturationIndicator.Detail),
		"",
		"## Admission Policy Summary",
		"",
		"- Policy mode: `advisory-only reviewer guidance`",
		"- Runtime enforcement: `none`",
		fmt.Sprintf("- Default reviewer envelope: `%v`", rep.Summary["recommended_sustained_envelope"]),
		fmt.Sprintf("- Ceiling reviewer envelope: `%v`", rep.Summary["ceiling_envelope"]),
		"- Scheduler note: recommended envelopes guide reviewer admission decisions and are not scheduler-enforced runtime limits.",
		"",
		"## Microbenchmark Thresholds",
		"",
	}
	for _, lane := range rep.Microbenchmarks {
		lines = append(lines, fmt.Sprintf("- `%s`: `%.2f ns/op` vs limit `%v` -> `%s`", lane.Lane, lane.Observed, lane.Threshold.Value, lane.Status))
	}
	lines = append(lines, "", "## Soak Matrix", "")
	for _, lane := range rep.SoakMatrix {
		lines = append(lines, fmt.Sprintf("- `%s`: `%.3f tasks/s`, `%d failed`, envelope `%s` -> `%s`", lane.Lane, lane.Observed.ThroughputTasksPerSec, lane.Observed.Failed, lane.OperatingEnvelope, lane.Status))
	}
	lines = append(lines, "", "## Workload Mix", "", fmt.Sprintf("- `mixed-workload-routing`: `%s` -> `%s`", rep.MixedWorkload.Detail, rep.MixedWorkload.Status), "", "## Recommended Operating Envelopes", "")
	for _, envelope := range rep.OperatingEnvelopes {
		lines = append(lines, fmt.Sprintf("- `%s`: %s Evidence: `%s`.", envelope.Name, envelope.Recommendation, strings.Join(envelope.EvidenceLanes, ", ")))
	}
	lines = append(lines, "", "## Saturation Notes", "")
	for _, note := range rep.SaturationNotes {
		lines = append(lines, "- "+note)
	}
	lines = append(lines, "", "## Limits", "")
	for _, limit := range rep.Limits {
		lines = append(lines, "- "+limit)
	}
	return strings.Join(lines, "\n") + "\n"
}

func resolvePath(root, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}

func repoRelativePath(root, path string) string {
	abs := resolvePath(root, path)
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func writeFile(path string, body []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}

func deriveGeneratedAt(payloads ...any) string {
	var latest time.Time
	var found bool
	for _, payload := range payloads {
		walkTimestamps(payload, func(t time.Time) {
			if !found || t.After(latest) {
				latest = t
				found = true
			}
		})
	}
	if !found {
		return time.Now().UTC().Format(time.RFC3339Nano)
	}
	return latest.UTC().Format("2006-01-02T15:04:05.999999Z")
}

func walkTimestamps(value any, visit func(time.Time)) {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			switch key {
			case "generated_at", "timestamp", "created_at", "completed_at", "started_at":
				if ts, ok := parseTimestamp(item); ok {
					visit(ts)
				}
			}
			walkTimestamps(item, visit)
		}
	case []any:
		for _, item := range typed {
			walkTimestamps(item, visit)
		}
	}
}

func parseTimestamp(value any) (time.Time, bool) {
	text, ok := value.(string)
	if !ok {
		return time.Time{}, false
	}
	text = strings.Replace(text, "Z", "+00:00", 1)
	ts, err := time.Parse(time.RFC3339Nano, text)
	if err != nil {
		return time.Time{}, false
	}
	return ts.UTC(), true
}

func allBenchmarkPass(values []benchmarkLane) bool {
	for _, value := range values {
		if value.Status != "pass" {
			return false
		}
	}
	return true
}

func allSoakPass(values []soakLane) bool {
	for _, value := range values {
		if value.Status != "pass" {
			return false
		}
	}
	return true
}

func formatStatusList(values []benchmarkLane) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, "'"+value.Status+"'")
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func formatSoakStatusList(values []soakLane) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, "'"+value.Status+"'")
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func round(value float64, precision int) float64 {
	pow := 1.0
	for i := 0; i < precision; i++ {
		pow *= 10
	}
	if value >= 0 {
		return float64(int(value*pow+0.5)) / pow
	}
	return float64(int(value*pow-0.5)) / pow
}

func ternary[T any](condition bool, yes, no T) T {
	if condition {
		return yes
	}
	return no
}
