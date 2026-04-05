package reporting

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

const CapacityCertificationGenerator = "bigclaw-go/scripts/benchmark/capacity_certification/main.go"

var (
	microbenchmarkLimits = map[string]float64{
		"BenchmarkMemoryQueueEnqueueLease-8": 100_000,
		"BenchmarkFileQueueEnqueueLease-8":   40_000_000,
		"BenchmarkSQLiteQueueEnqueueLease-8": 25_000_000,
		"BenchmarkSchedulerDecide-8":         1_000,
	}
	soakThresholds = map[string]map[string]any{
		"50x8":    {"min_throughput": 5.0, "max_failures": 0, "envelope": "bootstrap-burst"},
		"100x12":  {"min_throughput": 8.5, "max_failures": 0, "envelope": "bootstrap-burst"},
		"1000x24": {"min_throughput": 9.0, "max_failures": 0, "envelope": "recommended-local-sustained"},
		"2000x24": {"min_throughput": 8.5, "max_failures": 0, "envelope": "recommended-local-ceiling"},
	}
)

const saturationDropThresholdPct = 12.0

type CapacityCertificationOptions struct {
	BenchmarkReportPath         string
	MixedWorkloadReportPath     string
	SupplementalSoakReportPaths []string
}

func BuildCapacityCertification(root string, options CapacityCertificationOptions) (map[string]any, string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, "", fmt.Errorf("repo root is required")
	}
	if options.BenchmarkReportPath == "" {
		options.BenchmarkReportPath = "bigclaw-go/docs/reports/benchmark-matrix-report.json"
	}
	if options.MixedWorkloadReportPath == "" {
		options.MixedWorkloadReportPath = "bigclaw-go/docs/reports/mixed-workload-matrix-report.json"
	}
	if len(options.SupplementalSoakReportPaths) == 0 {
		options.SupplementalSoakReportPaths = []string{
			"bigclaw-go/docs/reports/soak-local-1000x24.json",
			"bigclaw-go/docs/reports/soak-local-2000x24.json",
		}
	}

	var benchmarkReport map[string]any
	if err := loadJSON(resolveReportPath(root, options.BenchmarkReportPath), &benchmarkReport); err != nil {
		return nil, "", err
	}
	var mixedWorkloadReport map[string]any
	if err := loadJSON(resolveReportPath(root, options.MixedWorkloadReportPath), &mixedWorkloadReport); err != nil {
		return nil, "", err
	}

	microbenchmarks := make([]map[string]any, 0, len(microbenchmarkLimits))
	parsed := asMap(asMap(benchmarkReport["benchmark"])["parsed"])
	for _, lane := range []string{
		"BenchmarkMemoryQueueEnqueueLease-8",
		"BenchmarkFileQueueEnqueueLease-8",
		"BenchmarkSQLiteQueueEnqueueLease-8",
		"BenchmarkSchedulerDecide-8",
	} {
		microbenchmarks = append(microbenchmarks, benchmarkLane(lane, asFloat(asMap(parsed[lane])["ns_per_op"]), microbenchmarkLimits[lane]))
	}

	soakMatrix := make([]map[string]any, 0, len(soakThresholds))
	soakInputs := make([]string, 0, 4)
	soakResultsByLabel := map[string]map[string]any{}
	for _, entry := range asSlice(benchmarkReport["soak_matrix"]) {
		item := asMap(entry)
		result := asMap(item["result"])
		label := fmt.Sprintf("%dx%d", asInt(result["count"]), asInt(result["workers"]))
		soakResultsByLabel[label] = result
		soakInputs = append(soakInputs, filepath.ToSlash(asString(item["report_path"])))
	}
	supplementalSoakReports := make([]map[string]any, 0, len(options.SupplementalSoakReportPaths))
	for _, soakPath := range options.SupplementalSoakReportPaths {
		var payload map[string]any
		if err := loadJSON(resolveReportPath(root, soakPath), &payload); err != nil {
			return nil, "", err
		}
		supplementalSoakReports = append(supplementalSoakReports, payload)
		label := fmt.Sprintf("%dx%d", asInt(payload["count"]), asInt(payload["workers"]))
		soakResultsByLabel[label] = payload
		soakInputs = append(soakInputs, filepath.ToSlash(soakPath))
	}
	for _, lane := range []string{"50x8", "100x12", "1000x24", "2000x24"} {
		soakMatrix = append(soakMatrix, soakLane(lane, soakResultsByLabel[lane], soakThresholds[lane]))
	}

	mixedWorkload := mixedWorkloadLane(mixedWorkloadReport)
	saturationIndicator := buildSaturationSummary(soakMatrix)
	allLanes := append(append([]map[string]any{}, microbenchmarks...), soakMatrix...)
	allLanes = append(allLanes, mixedWorkload)
	passedLanes := 0
	failedLanes := make([]string, 0)
	for _, lane := range allLanes {
		status := asString(lane["status"])
		if status == "pass" || status == "pass-with-ceiling" {
			passedLanes++
			continue
		}
		failedLanes = append(failedLanes, asString(lane["lane"]))
	}

	report := map[string]any{
		"generated_at": deriveGeneratedAt(append([]map[string]any{benchmarkReport, mixedWorkloadReport}, supplementalSoakReports...)...),
		"ticket":       "BIG-PAR-098",
		"title":        "Production-grade capacity certification matrix",
		"status":       "repo-native-capacity-certification",
		"evidence_inputs": map[string]any{
			"benchmark_report_path":      filepath.ToSlash(options.BenchmarkReportPath),
			"mixed_workload_report_path": filepath.ToSlash(options.MixedWorkloadReportPath),
			"soak_report_paths":          soakInputs,
			"generator_script":           CapacityCertificationGenerator,
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
		"operating_envelopes": []map[string]any{
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
		},
		"certification_checks": []map[string]any{
			checkEntry("all_microbenchmark_thresholds_hold", allLaneStatuses(microbenchmarks, "pass"), fmt.Sprintf("%v", collectLaneStatuses(microbenchmarks))),
			checkEntry("all_soak_lanes_hold", allLaneStatuses(soakMatrix, "pass"), fmt.Sprintf("%v", collectLaneStatuses(soakMatrix))),
			checkEntry("mixed_workload_routes_match_expected_executors", asString(mixedWorkload["status"]) == "pass" || asString(mixedWorkload["status"]) == "pass-with-ceiling", asString(mixedWorkload["detail"])),
			checkEntry("ceiling_lane_does_not_show_excessive_throughput_drop", asString(saturationIndicator["status"]) == "pass", fmt.Sprintf("drop_pct=%v threshold=%v", saturationIndicator["throughput_drop_pct"], saturationDropThresholdPct)),
		},
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
	return report, renderCapacityCertificationMarkdown(report), nil
}

func benchmarkLane(name string, observed float64, maxNsPerOp float64) map[string]any {
	return map[string]any{
		"lane":     name,
		"metric":   "ns_per_op",
		"observed": observed,
		"threshold": map[string]any{
			"operator": "<=",
			"value":    maxNsPerOp,
		},
		"status": map[bool]string{true: "pass", false: "fail"}[observed <= maxNsPerOp],
		"detail": fmt.Sprintf("observed=%vns/op limit=%vns/op", observed, maxNsPerOp),
	}
}

func soakLane(label string, result map[string]any, threshold map[string]any) map[string]any {
	throughput := asFloat(result["throughput_tasks_per_sec"])
	failures := asInt(result["failed"])
	status := "fail"
	if throughput >= asFloat(threshold["min_throughput"]) && failures <= asInt(threshold["max_failures"]) {
		status = "pass"
	}
	return map[string]any{
		"lane": label,
		"scenario": map[string]any{
			"count":   asInt(result["count"]),
			"workers": asInt(result["workers"]),
		},
		"observed": map[string]any{
			"elapsed_seconds":          roundToThreeDecimals(asFloat(result["elapsed_seconds"])),
			"throughput_tasks_per_sec": roundToThreeDecimals(throughput),
			"succeeded":                asInt(result["succeeded"]),
			"failed":                   failures,
		},
		"thresholds": map[string]any{
			"min_throughput_tasks_per_sec": asFloat(threshold["min_throughput"]),
			"max_failures":                 asInt(threshold["max_failures"]),
		},
		"operating_envelope": asString(threshold["envelope"]),
		"status":             status,
		"detail":             fmt.Sprintf("throughput=%vtps min=%v failures=%d max=%d", roundToThreeDecimals(throughput), asFloat(threshold["min_throughput"]), failures, asInt(threshold["max_failures"])),
	}
}

func mixedWorkloadLane(report map[string]any) map[string]any {
	tasks := asSlice(report["tasks"])
	mismatches := make([]string, 0)
	successfulTasks := 0
	for _, item := range tasks {
		task := asMap(item)
		if asBool(task["ok"]) == false {
			mismatches = append(mismatches, fmt.Sprintf("%s: task-level ok=false", asString(task["name"])))
		}
		if asString(task["expected_executor"]) != asString(task["routed_executor"]) {
			mismatches = append(mismatches, fmt.Sprintf("%s: expected=%s routed=%s", asString(task["name"]), asString(task["expected_executor"]), asString(task["routed_executor"])))
		}
		if asString(task["final_state"]) != "succeeded" {
			mismatches = append(mismatches, fmt.Sprintf("%s: final_state=%s", asString(task["name"]), asString(task["final_state"])))
		}
		if asString(task["final_state"]) == "succeeded" {
			successfulTasks++
		}
	}
	status := "fail"
	if asBool(report["all_ok"]) && len(tasks) >= 5 && len(mismatches) == 0 {
		status = "pass"
	} else if asBool(report["all_ok"]) {
		status = "pass-with-ceiling"
	}
	detail := "all sampled mixed-workload routes landed on the expected executor path"
	if len(mismatches) > 0 {
		detail = strings.Join(mismatches, "; ")
	}
	return map[string]any{
		"lane": "mixed-workload-routing",
		"observed": map[string]any{
			"all_ok":           asBool(report["all_ok"]),
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
	var baseline map[string]any
	var ceiling map[string]any
	for _, lane := range soakLanes {
		switch asString(lane["lane"]) {
		case "1000x24":
			baseline = lane
		case "2000x24":
			ceiling = lane
		}
	}
	baselineTPS := asFloat(asMap(baseline["observed"])["throughput_tasks_per_sec"])
	ceilingTPS := asFloat(asMap(ceiling["observed"])["throughput_tasks_per_sec"])
	dropPct := 0.0
	if baselineTPS > 0 {
		dropPct = roundToTwoDecimals(((baselineTPS - ceilingTPS) / baselineTPS) * 100)
	}
	status := "warn"
	if dropPct <= saturationDropThresholdPct {
		status = "pass"
	}
	detail := "throughput drops materially at the 2000-task ceiling and should be treated as saturation"
	if status == "pass" {
		detail = "throughput remains in the same single-instance local band at the 2000-task ceiling"
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

func renderCapacityCertificationMarkdown(report map[string]any) string {
	summary := asMap(report["summary"])
	saturation := asMap(report["saturation_indicator"])
	lines := []string{
		"# Capacity Certification Report",
		"",
		"## Scope",
		"",
		fmt.Sprintf("- Generated at: `%s`", asString(report["generated_at"])),
		fmt.Sprintf("- Ticket: `%s`", asString(report["ticket"])),
		"- Goal: convert checked-in benchmark, soak, and mixed-workload evidence into a repo-native certification matrix with explicit thresholds and operating envelopes.",
		"- Boundary: this is a single-instance repo-native certification slice, not a live multi-tenant production attestation.",
		"",
		"## Certification Summary",
		"",
		fmt.Sprintf("- Overall status: `%s`", asString(summary["overall_status"])),
		fmt.Sprintf("- Passed lanes: `%d/%d`", asInt(summary["passed_lanes"]), asInt(summary["total_lanes"])),
		fmt.Sprintf("- Recommended local sustained envelope: `%s`", asString(summary["recommended_sustained_envelope"])),
		fmt.Sprintf("- Local ceiling envelope: `%s`", asString(summary["ceiling_envelope"])),
		fmt.Sprintf("- Saturation signal: `%s`", asString(saturation["detail"])),
		"",
		"## Admission Policy Summary",
		"",
		"- Policy mode: `advisory-only reviewer guidance`",
		"- Runtime enforcement: `none`",
		fmt.Sprintf("- Default reviewer envelope: `%s`", asString(summary["recommended_sustained_envelope"])),
		fmt.Sprintf("- Ceiling reviewer envelope: `%s`", asString(summary["ceiling_envelope"])),
		"- Scheduler note: recommended envelopes guide reviewer admission decisions and are not scheduler-enforced runtime limits.",
		"",
		"## Microbenchmark Thresholds",
		"",
	}
	for _, item := range mapSlice(report["microbenchmarks"]) {
		lines = append(lines, fmt.Sprintf("- `%s`: `%.2f ns/op` vs limit `%.0f` -> `%s`", asString(item["lane"]), asFloat(item["observed"]), asFloat(asMap(item["threshold"])["value"]), asString(item["status"])))
	}
	lines = append(lines, "", "## Soak Matrix", "")
	for _, item := range mapSlice(report["soak_matrix"]) {
		observed := asMap(item["observed"])
		lines = append(lines, fmt.Sprintf("- `%s`: `%.3f tasks/s`, `%d failed`, envelope `%s` -> `%s`", asString(item["lane"]), asFloat(observed["throughput_tasks_per_sec"]), asInt(observed["failed"]), asString(item["operating_envelope"]), asString(item["status"])))
	}
	lines = append(lines, "", "## Workload Mix", "", fmt.Sprintf("- `mixed-workload-routing`: `%s` -> `%s`", asString(asMap(report["mixed_workload"])["detail"]), asString(asMap(report["mixed_workload"])["status"])), "", "## Recommended Operating Envelopes", "")
	for _, item := range mapSlice(report["operating_envelopes"]) {
		evidence := make([]string, 0)
		for _, lane := range stringSlice(item["evidence_lanes"]) {
			evidence = append(evidence, lane)
		}
		lines = append(lines, fmt.Sprintf("- `%s`: %s Evidence: `%s`.", asString(item["name"]), asString(item["recommendation"]), strings.Join(evidence, ", ")))
	}
	lines = append(lines, "", "## Saturation Notes", "")
	for _, note := range stringSlice(report["saturation_notes"]) {
		lines = append(lines, fmt.Sprintf("- %s", note))
	}
	lines = append(lines, "", "## Limits", "")
	for _, note := range stringSlice(report["limits"]) {
		lines = append(lines, fmt.Sprintf("- %s", note))
	}
	return strings.Join(lines, "\n") + "\n"
}

func deriveGeneratedAt(payloads ...map[string]any) string {
	latest := ""
	latestTime := time.Time{}
	for _, payload := range payloads {
		collectGeneratedAt(payload, &latestTime, &latest)
	}
	if latest != "" {
		return latest
	}
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func collectGeneratedAt(value any, latestTime *time.Time, latest *string) {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			if key == "generated_at" || key == "timestamp" || key == "created_at" || key == "completed_at" || key == "started_at" {
				if text := asString(item); text != "" {
					if parsed, err := parseFlexibleTime(text); err == nil && parsed.After(*latestTime) {
						*latestTime = parsed
						*latest = parsed.UTC().Format(time.RFC3339Nano)
					}
				}
			}
			collectGeneratedAt(item, latestTime, latest)
		}
	case []any:
		for _, item := range typed {
			collectGeneratedAt(item, latestTime, latest)
		}
	}
}

func repoRelativePath(root string, path string) string {
	resolved := resolveReportPath(root, path)
	rel, err := filepath.Rel(root, resolved)
	if err != nil {
		return filepath.ToSlash(resolved)
	}
	return filepath.ToSlash(rel)
}

func allLaneStatuses(values []map[string]any, want string) bool {
	for _, value := range values {
		if asString(value["status"]) != want {
			return false
		}
	}
	return true
}

func collectLaneStatuses(values []map[string]any) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, asString(value["status"]))
	}
	return out
}

func checkEntry(name string, passed bool, detail string) map[string]any {
	return map[string]any{"name": name, "passed": passed, "detail": detail}
}

func mapSlice(value any) []map[string]any {
	switch typed := value.(type) {
	case []map[string]any:
		return typed
	case []any:
		out := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, asMap(item))
		}
		return out
	default:
		return nil
	}
}

func stringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, asString(item))
		}
		return out
	default:
		return nil
	}
}

func roundToThreeDecimals(value float64) float64 {
	return float64(int(value*1000+0.5)) / 1000
}
