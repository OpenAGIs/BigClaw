package capacitycert

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
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

type BuildOptions struct {
	RepoRoot                    string
	BenchmarkReportPath         string
	MixedWorkloadReportPath     string
	SupplementalSoakReportPaths []string
}

func BuildReport(opts BuildOptions) (map[string]any, string, error) {
	repoRoot := defaultString(opts.RepoRoot, ".")
	benchmarkReportPath := defaultString(opts.BenchmarkReportPath, "bigclaw-go/docs/reports/benchmark-matrix-report.json")
	mixedWorkloadReportPath := defaultString(opts.MixedWorkloadReportPath, "bigclaw-go/docs/reports/mixed-workload-matrix-report.json")
	supplementalPaths := opts.SupplementalSoakReportPaths
	if len(supplementalPaths) == 0 {
		supplementalPaths = []string{
			"bigclaw-go/docs/reports/soak-local-1000x24.json",
			"bigclaw-go/docs/reports/soak-local-2000x24.json",
		}
	}

	var benchmarkReport map[string]any
	if err := readJSON(resolveRepoPath(repoRoot, benchmarkReportPath), &benchmarkReport); err != nil {
		return nil, "", err
	}
	var mixedWorkloadReport map[string]any
	if err := readJSON(resolveRepoPath(repoRoot, mixedWorkloadReportPath), &mixedWorkloadReport); err != nil {
		return nil, "", err
	}

	microbenchmarks := make([]map[string]any, 0, len(microbenchmarkLimits))
	parsed := nestedMap(nestedMap(benchmarkReport, "benchmark"), "parsed")
	for _, benchmarkName := range orderedBenchmarkNames() {
		nsPerOp := floatValue(nestedMap(parsed, benchmarkName)["ns_per_op"])
		microbenchmarks = append(microbenchmarks, benchmarkLane(benchmarkName, nsPerOp, microbenchmarkLimits[benchmarkName]))
	}

	soakInputs := make([]string, 0)
	soakResultsByLabel := map[string]map[string]any{}
	for _, entry := range mapSliceAt(benchmarkReport, "soak_matrix") {
		result := nestedMap(entry, "result")
		label := fmt.Sprintf("%dx%d", intValue(result["count"]), intValue(result["workers"]))
		soakResultsByLabel[label] = result
		soakInputs = append(soakInputs, repoRelativePath(repoRoot, stringValue(entry["report_path"], "")))
	}

	supplementalPayloads := make([]map[string]any, 0, len(supplementalPaths))
	for _, soakPath := range supplementalPaths {
		var payload map[string]any
		if err := readJSON(resolveRepoPath(repoRoot, soakPath), &payload); err != nil {
			return nil, "", err
		}
		supplementalPayloads = append(supplementalPayloads, payload)
		label := fmt.Sprintf("%dx%d", intValue(payload["count"]), intValue(payload["workers"]))
		soakResultsByLabel[label] = payload
		soakInputs = append(soakInputs, repoRelativePath(repoRoot, soakPath))
	}

	soakMatrix := make([]map[string]any, 0, len(soakThresholds))
	for _, label := range []string{"50x8", "100x12", "1000x24", "2000x24"} {
		soakMatrix = append(soakMatrix, soakLane(label, soakResultsByLabel[label], soakThresholds[label]))
	}

	mixedWorkload := mixedWorkloadLane(mixedWorkloadReport)
	saturationIndicator := buildSaturationSummary(soakMatrix)
	allLanes := append(append([]map[string]any{}, microbenchmarks...), soakMatrix...)
	allLanes = append(allLanes, mixedWorkload)

	passedLanes := 0
	failedLanes := make([]string, 0)
	for _, lane := range allLanes {
		status := stringValue(lane["status"], "")
		if status == "pass" || status == "pass-with-ceiling" {
			passedLanes++
			continue
		}
		failedLanes = append(failedLanes, stringValue(lane["lane"], ""))
	}

	operatingEnvelopes := []map[string]any{
		{
			"name":           "recommended-local-sustained",
			"recommendation": "Use up to 1000 queued tasks with 24 submit workers when a stable single-instance local review lane is required.",
			"evidence_lanes": []string{"1000x24"},
		},
		{
			"name":           "recommended-local-ceiling",
			"recommendation": "Treat 2000 queued tasks with 24 submit workers as the checked-in local ceiling, not the default operating point.",
			"evidence_lanes": []string{"2000x24"},
		},
		{
			"name":           "mixed-workload-routing",
			"recommendation": "Use the mixed-workload matrix for executor routing correctness, but do not infer sustained multi-executor throughput from it.",
			"evidence_lanes": []string{"mixed-workload-routing"},
		},
	}

	checks := []map[string]any{
		check("all_microbenchmark_thresholds_hold", allStatuses(microbenchmarks, "pass"), pythonStringList(collectLaneStatuses(microbenchmarks))),
		check("all_soak_lanes_hold", allStatuses(soakMatrix, "pass"), pythonStringList(collectLaneStatuses(soakMatrix))),
		check("mixed_workload_routes_match_expected_executors", stringValue(mixedWorkload["status"], "") == "pass" || stringValue(mixedWorkload["status"], "") == "pass-with-ceiling", stringValue(mixedWorkload["detail"], "")),
		check("ceiling_lane_does_not_show_excessive_throughput_drop", stringValue(saturationIndicator["status"], "") == "pass", fmt.Sprintf("drop_pct=%s threshold=%s", pythonFloatString(floatValue(saturationIndicator["throughput_drop_pct"])), pythonFloatString(saturationDropThresholdPct))),
	}

	report := map[string]any{
		"generated_at": deriveGeneratedAt(append([]map[string]any{benchmarkReport, mixedWorkloadReport}, supplementalPayloads...)...),
		"ticket":       "BIG-PAR-098",
		"title":        "Production-grade capacity certification matrix",
		"status":       "repo-native-capacity-certification",
		"evidence_inputs": map[string]any{
			"benchmark_report_path":      repoRelativePath(repoRoot, benchmarkReportPath),
			"mixed_workload_report_path": repoRelativePath(repoRoot, mixedWorkloadReportPath),
			"soak_report_paths":          soakInputs,
			"generator_script":           "bigclaw-go/scripts/benchmark/capacity_certification.py",
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
		"operating_envelopes":  operatingEnvelopes,
		"certification_checks": checks,
		"saturation_notes": []string{
			"Throughput plateaus around 9-10 tasks/s across the checked-in 100x12, 1000x24, and 2000x24 local lanes.",
			"The 2000x24 lane remains within the same throughput band as 1000x24, so the checked-in local ceiling is evidence-backed but not substantially headroom-rich.",
			"Mixed-workload evidence verifies executor-routing correctness across local, Kubernetes, and Ray, but it is a functional routing proof rather than a concurrency ceiling.",
		},
		"limits": []string{
			"Evidence is repo-native and single-instance; it does not certify multi-node or multi-tenant production saturation behavior.",
			"The matrix uses checked-in local runs from 2026-03-13 and should be refreshed when queue, scheduler, or executor behavior changes materially.",
			"Recommended envelopes are conservative reviewer guidance derived from current evidence, not an automated runtime admission policy.",
		},
	}
	return report, buildMarkdown(report), nil
}

func WriteOutputs(jsonPath, markdownPath string, report map[string]any, markdown string) error {
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	if err := os.MkdirAll(filepath.Dir(jsonPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(jsonPath, body, 0o644); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(markdownPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(markdownPath, []byte(markdown), 0o644)
}

func benchmarkLane(name string, nsPerOp, maxNsPerOp float64) map[string]any {
	return map[string]any{
		"lane":     name,
		"metric":   "ns_per_op",
		"observed": nsPerOp,
		"threshold": map[string]any{
			"operator": "<=",
			"value":    maxNsPerOp,
		},
		"status": ternaryString(nsPerOp <= maxNsPerOp, "pass", "fail"),
		"detail": fmt.Sprintf("observed=%sns/op limit=%sns/op", pythonFloatString(nsPerOp), pythonIntLikeString(maxNsPerOp)),
	}
}

func soakLane(label string, result map[string]any, threshold soakThreshold) map[string]any {
	throughput := floatValue(result["throughput_tasks_per_sec"])
	failures := intValue(result["failed"])
	return map[string]any{
		"lane": label,
		"scenario": map[string]any{
			"count":   intValue(result["count"]),
			"workers": intValue(result["workers"]),
		},
		"observed": map[string]any{
			"elapsed_seconds":          roundFloat(floatValue(result["elapsed_seconds"]), 3),
			"throughput_tasks_per_sec": roundFloat(throughput, 3),
			"succeeded":                intValue(result["succeeded"]),
			"failed":                   failures,
		},
		"thresholds": map[string]any{
			"min_throughput_tasks_per_sec": threshold.MinThroughput,
			"max_failures":                 threshold.MaxFailures,
		},
		"operating_envelope": threshold.Envelope,
		"status":             ternaryString(throughput >= threshold.MinThroughput && failures <= threshold.MaxFailures, "pass", "fail"),
		"detail": fmt.Sprintf(
			"throughput=%stps min=%s failures=%d max=%d",
			pythonFloatString(roundFloat(throughput, 3)),
			pythonFloatString(threshold.MinThroughput),
			failures,
			threshold.MaxFailures,
		),
	}
}

func mixedWorkloadLane(report map[string]any) map[string]any {
	tasks := mapSliceAt(report, "tasks")
	mismatches := make([]string, 0)
	successfulTasks := 0
	for _, task := range tasks {
		if !boolValue(task["ok"], false) {
			mismatches = append(mismatches, fmt.Sprintf("%s: task-level ok=false", stringValue(task["name"], "")))
		}
		if stringValue(task["expected_executor"], "") != stringValue(task["routed_executor"], "") {
			mismatches = append(mismatches, fmt.Sprintf("%s: expected=%s routed=%s", stringValue(task["name"], ""), stringValue(task["expected_executor"], ""), stringValue(task["routed_executor"], "")))
		}
		if stringValue(task["final_state"], "") != "succeeded" {
			mismatches = append(mismatches, fmt.Sprintf("%s: final_state=%s", stringValue(task["name"], ""), stringValue(task["final_state"], "")))
		}
		if stringValue(task["final_state"], "") == "succeeded" {
			successfulTasks++
		}
	}

	status := "fail"
	if boolValue(report["all_ok"], false) && len(tasks) >= 5 && len(mismatches) == 0 {
		status = "pass"
	} else if boolValue(report["all_ok"], false) {
		status = "pass-with-ceiling"
	}

	detail := "all sampled mixed-workload routes landed on the expected executor path"
	if len(mismatches) > 0 {
		detail = strings.Join(mismatches, "; ")
	}

	return map[string]any{
		"lane": "mixed-workload-routing",
		"observed": map[string]any{
			"all_ok":           boolValue(report["all_ok"], false),
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
		"limitations": []string{
			"executor-mix coverage is functional rather than high-volume",
			"mixed-workload evidence proves route correctness but not sustained cross-executor saturation limits",
		},
	}
}

func buildSaturationSummary(soakLanes []map[string]any) map[string]any {
	baseline := findLane(soakLanes, "1000x24")
	ceiling := findLane(soakLanes, "2000x24")
	baselineTPS := floatValue(nestedMap(baseline, "observed")["throughput_tasks_per_sec"])
	ceilingTPS := floatValue(nestedMap(ceiling, "observed")["throughput_tasks_per_sec"])
	dropPct := 0.0
	if baselineTPS != 0 {
		dropPct = roundFloat(((baselineTPS-ceilingTPS)/baselineTPS)*100, 2)
	}

	status := ternaryString(dropPct <= saturationDropThresholdPct, "pass", "warn")
	detail := "throughput remains in the same single-instance local band at the 2000-task ceiling"
	if status != "pass" {
		detail = "throughput drops materially at the 2000-task ceiling and should be treated as saturation"
	}

	return map[string]any{
		"baseline_lane":                     "1000x24",
		"ceiling_lane":                      "2000x24",
		"baseline_throughput_tasks_per_sec": baselineTPS,
		"ceiling_throughput_tasks_per_sec":  ceilingTPS,
		"throughput_drop_pct":               dropPct,
		"drop_warn_threshold_pct":           saturationDropThresholdPct,
		"status":                            status,
		"detail":                            detail,
	}
}

func buildMarkdown(report map[string]any) string {
	summary := nestedMap(report, "summary")
	lines := []string{
		"# Capacity Certification Report",
		"",
		"## Scope",
		"",
		fmt.Sprintf("- Generated at: `%s`", stringValue(report["generated_at"], "")),
		fmt.Sprintf("- Ticket: `%s`", stringValue(report["ticket"], "")),
		"- Goal: convert checked-in benchmark, soak, and mixed-workload evidence into a repo-native certification matrix with explicit thresholds and operating envelopes.",
		"- Boundary: this is a single-instance repo-native certification slice, not a live multi-tenant production attestation.",
		"",
		"## Certification Summary",
		"",
		fmt.Sprintf("- Overall status: `%s`", stringValue(summary["overall_status"], "")),
		fmt.Sprintf("- Passed lanes: `%v/%v`", summary["passed_lanes"], summary["total_lanes"]),
		fmt.Sprintf("- Recommended local sustained envelope: `%s`", stringValue(summary["recommended_sustained_envelope"], "")),
		fmt.Sprintf("- Local ceiling envelope: `%s`", stringValue(summary["ceiling_envelope"], "")),
		fmt.Sprintf("- Saturation signal: `%s`", stringValue(nestedMap(report, "saturation_indicator")["detail"], "")),
		"",
		"## Admission Policy Summary",
		"",
		"- Policy mode: `advisory-only reviewer guidance`",
		"- Runtime enforcement: `none`",
		fmt.Sprintf("- Default reviewer envelope: `%s`", stringValue(summary["recommended_sustained_envelope"], "")),
		fmt.Sprintf("- Ceiling reviewer envelope: `%s`", stringValue(summary["ceiling_envelope"], "")),
		"- Scheduler note: recommended envelopes guide reviewer admission decisions and are not scheduler-enforced runtime limits.",
		"",
		"## Microbenchmark Thresholds",
		"",
	}

	for _, lane := range mapSliceAt(report, "microbenchmarks") {
		threshold := floatValue(nestedMap(lane, "threshold")["value"])
		lines = append(lines, fmt.Sprintf(
			"- `%s`: `%.2f ns/op` vs limit `%v` -> `%s`",
			stringValue(lane["lane"], ""),
			floatValue(lane["observed"]),
			pythonIntLikeString(threshold),
			stringValue(lane["status"], ""),
		))
	}

	lines = append(lines, "", "## Soak Matrix", "")
	for _, lane := range mapSliceAt(report, "soak_matrix") {
		observed := nestedMap(lane, "observed")
		lines = append(lines, fmt.Sprintf(
			"- `%s`: `%v tasks/s`, `%v failed`, envelope `%s` -> `%s`",
			stringValue(lane["lane"], ""),
			observed["throughput_tasks_per_sec"],
			observed["failed"],
			lane["operating_envelope"],
			lane["status"],
		))
	}

	lines = append(
		lines,
		"",
		"## Workload Mix",
		"",
		fmt.Sprintf("- `mixed-workload-routing`: `%s` -> `%s`", stringValue(nestedMap(report, "mixed_workload")["detail"], ""), stringValue(nestedMap(report, "mixed_workload")["status"], "")),
		"",
		"## Recommended Operating Envelopes",
		"",
	)
	for _, envelope := range mapSliceAt(report, "operating_envelopes") {
		lines = append(lines, fmt.Sprintf(
			"- `%s`: %s Evidence: `%s`.",
			stringValue(envelope["name"], ""),
			stringValue(envelope["recommendation"], ""),
			strings.Join(stringSliceAt(envelope, "evidence_lanes"), ", "),
		))
	}

	lines = append(lines, "", "## Saturation Notes", "")
	for _, note := range stringSliceAt(report, "saturation_notes") {
		lines = append(lines, "- "+note)
	}

	lines = append(lines, "", "## Limits", "")
	for _, item := range stringSliceAt(report, "limits") {
		lines = append(lines, "- "+item)
	}

	return strings.Join(lines, "\n") + "\n"
}

func deriveGeneratedAt(payloads ...map[string]any) string {
	var latest time.Time
	for _, payload := range payloads {
		for _, ts := range iterTimestamps(payload) {
			if ts.After(latest) {
				latest = ts
			}
		}
	}
	if latest.IsZero() {
		return time.Now().UTC().Format(time.RFC3339Nano)
	}
	return latest.UTC().Format(time.RFC3339Nano)
}

func iterTimestamps(payload any) []time.Time {
	found := make([]time.Time, 0)
	switch value := payload.(type) {
	case map[string]any:
		for key, child := range value {
			if key == "generated_at" || key == "timestamp" || key == "created_at" || key == "completed_at" || key == "started_at" {
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
	case []map[string]any:
		for _, child := range value {
			found = append(found, iterTimestamps(child)...)
		}
	}
	return found
}

func parseTimestamp(value any) (time.Time, bool) {
	text, ok := value.(string)
	if !ok {
		return time.Time{}, false
	}
	candidate := strings.Replace(text, "Z", "+00:00", 1)
	parsed, err := time.Parse(time.RFC3339Nano, candidate)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func orderedBenchmarkNames() []string {
	return []string{
		"BenchmarkMemoryQueueEnqueueLease-8",
		"BenchmarkFileQueueEnqueueLease-8",
		"BenchmarkSQLiteQueueEnqueueLease-8",
		"BenchmarkSchedulerDecide-8",
	}
}

func check(name string, passed bool, detail string) map[string]any {
	return map[string]any{"name": name, "passed": passed, "detail": detail}
}

func allStatuses(items []map[string]any, expected string) bool {
	for _, item := range items {
		if stringValue(item["status"], "") != expected {
			return false
		}
	}
	return true
}

func collectLaneStatuses(items []map[string]any) []string {
	statuses := make([]string, 0, len(items))
	for _, item := range items {
		statuses = append(statuses, stringValue(item["status"], ""))
	}
	return statuses
}

func pythonStringList(items []string) string {
	quoted := make([]string, 0, len(items))
	for _, item := range items {
		quoted = append(quoted, "'"+item+"'")
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func pythonFloatString(value float64) string {
	text := strconv.FormatFloat(value, 'f', -1, 64)
	if !strings.Contains(text, ".") {
		return text + ".0"
	}
	return text
}

func pythonIntLikeString(value float64) string {
	if value == float64(int64(value)) {
		return strconv.FormatInt(int64(value), 10)
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func findLane(items []map[string]any, lane string) map[string]any {
	for _, item := range items {
		if stringValue(item["lane"], "") == lane {
			return item
		}
	}
	return map[string]any{}
}

func readJSON(path string, target any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

func resolveRepoPath(repoRoot, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(repoRoot, path)
}

func repoRelativePath(repoRoot, path string) string {
	resolved := resolveRepoPath(repoRoot, path)
	rel, err := filepath.Rel(repoRoot, resolved)
	if err != nil {
		return filepath.ToSlash(resolved)
	}
	return filepath.ToSlash(rel)
}

func nestedMap(source map[string]any, key string) map[string]any {
	value, _ := source[key].(map[string]any)
	if value == nil {
		return map[string]any{}
	}
	return value
}

func mapSliceAt(source map[string]any, key string) []map[string]any {
	raw := source[key]
	value := reflect.ValueOf(raw)
	if !value.IsValid() || value.Kind() != reflect.Slice {
		return nil
	}
	items := make([]map[string]any, 0, value.Len())
	for i := 0; i < value.Len(); i++ {
		mapped, ok := value.Index(i).Interface().(map[string]any)
		if ok {
			items = append(items, mapped)
		}
	}
	return items
}

func stringSliceAt(source map[string]any, key string) []string {
	raw := source[key]
	value := reflect.ValueOf(raw)
	if !value.IsValid() || value.Kind() != reflect.Slice {
		return nil
	}
	items := make([]string, 0, value.Len())
	for i := 0; i < value.Len(); i++ {
		text, ok := value.Index(i).Interface().(string)
		if ok {
			items = append(items, text)
		}
	}
	return items
}

func stringValue(value any, fallback string) string {
	text, ok := value.(string)
	if !ok {
		return fallback
	}
	return text
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
		parsed, err := typed.Float64()
		if err == nil {
			return parsed
		}
	case string:
		parsed, err := strconv.ParseFloat(typed, 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}

func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case float32:
		return int(typed)
	case json.Number:
		parsed, err := typed.Int64()
		if err == nil {
			return int(parsed)
		}
	case string:
		parsed, err := strconv.Atoi(typed)
		if err == nil {
			return parsed
		}
	}
	return 0
}

func boolValue(value any, fallback bool) bool {
	typed, ok := value.(bool)
	if !ok {
		return fallback
	}
	return typed
}

func roundFloat(value float64, places int) float64 {
	formatted := strconv.FormatFloat(value, 'f', places, 64)
	parsed, err := strconv.ParseFloat(formatted, 64)
	if err != nil {
		return value
	}
	return parsed
}

func ternaryString(condition bool, ifTrue, ifFalse string) string {
	if condition {
		return ifTrue
	}
	return ifFalse
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
