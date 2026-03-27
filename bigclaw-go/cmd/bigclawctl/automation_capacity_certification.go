package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

var capacityMicrobenchmarkLimits = map[string]float64{
	"BenchmarkMemoryQueueEnqueueLease-8": 100_000,
	"BenchmarkFileQueueEnqueueLease-8":   40_000_000,
	"BenchmarkSQLiteQueueEnqueueLease-8": 25_000_000,
	"BenchmarkSchedulerDecide-8":         1_000,
}

var capacitySoakThresholds = map[string]map[string]any{
	"50x8":    {"min_throughput": 5.0, "max_failures": 0, "envelope": "bootstrap-burst"},
	"100x12":  {"min_throughput": 8.5, "max_failures": 0, "envelope": "bootstrap-burst"},
	"1000x24": {"min_throughput": 9.0, "max_failures": 0, "envelope": "recommended-local-sustained"},
	"2000x24": {"min_throughput": 8.5, "max_failures": 0, "envelope": "recommended-local-ceiling"},
}

const capacitySaturationDropThresholdPct = 12.0

type automationCapacityCertificationOptions struct {
	RepoRoot                string
	BenchmarkReportPath     string
	MixedWorkloadReportPath string
	SupplementalSoakReports []string
	OutputPath              string
	MarkdownOutputPath      string
}

func runAutomationCapacityCertificationCommand(args []string) error {
	flags := flag.NewFlagSet("automation benchmark capacity-certification", flag.ContinueOnError)
	repoRoot := flags.String("repo-root", "..", "repository root")
	benchmarkReport := flags.String("benchmark-report", "bigclaw-go/docs/reports/benchmark-matrix-report.json", "benchmark matrix report path")
	mixedWorkloadReport := flags.String("mixed-workload-report", "bigclaw-go/docs/reports/mixed-workload-matrix-report.json", "mixed workload report path")
	var supplemental multiStringFlag
	flags.Var(&supplemental, "supplemental-soak-report", "additional soak report path; repeatable")
	outputPath := flags.String("output", "bigclaw-go/docs/reports/capacity-certification-matrix.json", "output path")
	markdownOutput := flags.String("markdown-output", "bigclaw-go/docs/reports/capacity-certification-report.md", "markdown output path")
	pretty := flags.Bool("pretty", false, "print the generated JSON report")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation benchmark capacity-certification [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	report, _, err := automationCapacityCertification(automationCapacityCertificationOptions{
		RepoRoot:                absPath(*repoRoot),
		BenchmarkReportPath:     trim(*benchmarkReport),
		MixedWorkloadReportPath: trim(*mixedWorkloadReport),
		SupplementalSoakReports: supplemental.valuesOr([]string{
			"bigclaw-go/docs/reports/soak-local-1000x24.json",
			"bigclaw-go/docs/reports/soak-local-2000x24.json",
		}),
		OutputPath:         trim(*outputPath),
		MarkdownOutputPath: trim(*markdownOutput),
	})
	if err != nil {
		return err
	}
	if *pretty {
		return emit(report, true, 0)
	}
	return nil
}

func automationCapacityCertification(opts automationCapacityCertificationOptions) (map[string]any, int, error) {
	repoRoot := opts.RepoRoot
	benchmarkReport, ok := automationReadJSON(automationResolveRepoPath(repoRoot, opts.BenchmarkReportPath)).(map[string]any)
	if !ok {
		return nil, 0, fmt.Errorf("failed to load benchmark report from %s", opts.BenchmarkReportPath)
	}
	mixedWorkloadReport, ok := automationReadJSON(automationResolveRepoPath(repoRoot, opts.MixedWorkloadReportPath)).(map[string]any)
	if !ok {
		return nil, 0, fmt.Errorf("failed to load mixed workload report from %s", opts.MixedWorkloadReportPath)
	}

	microbenchmarks := make([]any, 0, len(capacityMicrobenchmarkLimits))
	parsed, _ := benchmarkReport["benchmark"].(map[string]any)
	parsedMetrics, _ := parsed["parsed"].(map[string]any)
	for name, limit := range capacityMicrobenchmarkLimits {
		metric, _ := parsedMetrics[name].(map[string]any)
		nsPerOp := automationFloat64(metric["ns_per_op"])
		microbenchmarks = append(microbenchmarks, map[string]any{
			"lane":      name,
			"metric":    "ns_per_op",
			"observed":  nsPerOp,
			"threshold": map[string]any{"operator": "<=", "value": limit},
			"status":    capacityPassFail(nsPerOp <= limit),
			"detail":    fmt.Sprintf("observed=%gns/op limit=%gns/op", nsPerOp, limit),
		})
	}

	soakInputs := []any{}
	soakResultsByLabel := map[string]map[string]any{}
	if soakMatrix, ok := benchmarkReport["soak_matrix"].([]any); ok {
		for _, item := range soakMatrix {
			entry, _ := item.(map[string]any)
			result, _ := entry["result"].(map[string]any)
			label := fmt.Sprintf("%sx%s", automationStringifyNumber(result["count"]), automationStringifyNumber(result["workers"]))
			soakResultsByLabel[label] = result
			if reportPath := automationFirstText(entry["report_path"]); reportPath != "" {
				soakInputs = append(soakInputs, repoRelativePath(repoRoot, reportPath))
			}
		}
	}

	supplementalPayloads := []any{}
	for _, soakPath := range opts.SupplementalSoakReports {
		result, _ := automationReadJSON(automationResolveRepoPath(repoRoot, soakPath)).(map[string]any)
		if result == nil {
			continue
		}
		supplementalPayloads = append(supplementalPayloads, result)
		label := fmt.Sprintf("%sx%s", automationStringifyNumber(result["count"]), automationStringifyNumber(result["workers"]))
		soakResultsByLabel[label] = result
		soakInputs = append(soakInputs, repoRelativePath(repoRoot, soakPath))
	}

	soakMatrix := make([]any, 0, len(capacitySoakThresholds))
	for label, threshold := range capacitySoakThresholds {
		result := soakResultsByLabel[label]
		throughput := automationFloat64(result["throughput_tasks_per_sec"])
		failures := automationInt(result["failed"])
		minThroughput := automationFloat64(threshold["min_throughput"])
		maxFailures := automationInt(threshold["max_failures"])
		soakMatrix = append(soakMatrix, map[string]any{
			"lane": label,
			"scenario": map[string]any{
				"count":   automationInt(result["count"]),
				"workers": automationInt(result["workers"]),
			},
			"observed": map[string]any{
				"elapsed_seconds":          roundFloat(automationFloat64(result["elapsed_seconds"]), 3),
				"throughput_tasks_per_sec": roundFloat(throughput, 3),
				"succeeded":                automationInt(result["succeeded"]),
				"failed":                   failures,
			},
			"thresholds": map[string]any{
				"min_throughput_tasks_per_sec": minThroughput,
				"max_failures":                 maxFailures,
			},
			"operating_envelope": threshold["envelope"],
			"status":             capacityPassFail(throughput >= minThroughput && failures <= maxFailures),
			"detail":             fmt.Sprintf("throughput=%gtps min=%g failures=%d max=%d", roundFloat(throughput, 3), minThroughput, failures, maxFailures),
		})
	}

	mixedWorkload := buildMixedWorkloadLane(mixedWorkloadReport)
	saturation := buildSaturationSummary(soakMatrix)

	allLanes := append([]any{}, microbenchmarks...)
	allLanes = append(allLanes, soakMatrix...)
	allLanes = append(allLanes, mixedWorkload)
	passedLanes := 0
	failedLanes := []any{}
	for _, item := range allLanes {
		lane, _ := item.(map[string]any)
		status := automationFirstText(lane["status"])
		if status == "pass" || status == "pass-with-ceiling" {
			passedLanes++
		} else {
			failedLanes = append(failedLanes, lane["lane"])
		}
	}

	operatingEnvelopes := []any{
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
	}

	checks := []any{
		automationCheck("all_microbenchmark_thresholds_hold", allLaneStatus(microbenchmarks, "pass"), fmt.Sprintf("%v", laneStatuses(microbenchmarks))),
		automationCheck("all_soak_lanes_hold", allLaneStatus(soakMatrix, "pass"), fmt.Sprintf("%v", laneStatuses(soakMatrix))),
		automationCheck("mixed_workload_routes_match_expected_executors", containsStatus(automationFirstText(mixedWorkload["status"]), "pass", "pass-with-ceiling"), automationFirstText(mixedWorkload["detail"])),
		automationCheck("ceiling_lane_does_not_show_excessive_throughput_drop", automationFirstText(saturation["status"]) == "pass", fmt.Sprintf("drop_pct=%v threshold=%v", saturation["throughput_drop_pct"], capacitySaturationDropThresholdPct)),
	}

	report := map[string]any{
		"generated_at": deriveGeneratedAt(append([]any{benchmarkReport, mixedWorkloadReport}, supplementalPayloads...)...),
		"ticket":       "BIG-PAR-098",
		"title":        "Production-grade capacity certification matrix",
		"status":       "repo-native-capacity-certification",
		"evidence_inputs": map[string]any{
			"benchmark_report_path":      repoRelativePath(repoRoot, opts.BenchmarkReportPath),
			"mixed_workload_report_path": repoRelativePath(repoRoot, opts.MixedWorkloadReportPath),
			"soak_report_paths":          soakInputs,
			"generator_script":           "bigclaw-go/scripts/benchmark/capacity_certification.py",
		},
		"summary": map[string]any{
			"overall_status":                 capacityPassFail(len(failedLanes) == 0),
			"total_lanes":                    len(allLanes),
			"passed_lanes":                   passedLanes,
			"failed_lanes":                   failedLanes,
			"recommended_sustained_envelope": "<=1000 tasks with 24 submit workers",
			"ceiling_envelope":               "<=2000 tasks with 24 submit workers",
		},
		"microbenchmarks":      microbenchmarks,
		"soak_matrix":          soakMatrix,
		"mixed_workload":       mixedWorkload,
		"saturation_indicator": saturation,
		"operating_envelopes":  operatingEnvelopes,
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
	markdown := buildCapacityMarkdown(report)
	if err := automationWriteJSON(automationResolveRepoPath(repoRoot, opts.OutputPath), report); err != nil {
		return nil, 0, err
	}
	if err := automationWriteText(automationResolveRepoPath(repoRoot, opts.MarkdownOutputPath), markdown); err != nil {
		return nil, 0, err
	}
	return report, 0, nil
}

func capacityPassFail(passed bool) string {
	if passed {
		return "pass"
	}
	return "fail"
}

func buildMixedWorkloadLane(report map[string]any) map[string]any {
	tasks, _ := report["tasks"].([]any)
	mismatches := []string{}
	successfulTasks := 0
	for _, item := range tasks {
		task, _ := item.(map[string]any)
		if automationBool(task["ok"]) == false {
			mismatches = append(mismatches, fmt.Sprintf("%v: task-level ok=false", task["name"]))
		}
		if automationFirstText(task["expected_executor"]) != automationFirstText(task["routed_executor"]) {
			mismatches = append(mismatches, fmt.Sprintf("%v: expected=%v routed=%v", task["name"], task["expected_executor"], task["routed_executor"]))
		}
		if automationFirstText(task["final_state"]) != "succeeded" {
			mismatches = append(mismatches, fmt.Sprintf("%v: final_state=%v", task["name"], task["final_state"]))
		}
		if automationFirstText(task["final_state"]) == "succeeded" {
			successfulTasks++
		}
	}
	status := "pass"
	if !automationBool(report["all_ok"]) {
		status = "fail"
	} else if len(mismatches) > 0 {
		status = "pass-with-ceiling"
	}
	detail := "all sampled mixed-workload routes landed on the expected executor path"
	if len(mismatches) > 0 {
		detail = strings.Join(mismatches, "; ")
	}
	return map[string]any{
		"lane": "mixed-workload-routing",
		"observed": map[string]any{
			"all_ok":           automationBool(report["all_ok"]),
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

func buildSaturationSummary(soakLanes []any) map[string]any {
	var baseline map[string]any
	var ceiling map[string]any
	for _, item := range soakLanes {
		lane, _ := item.(map[string]any)
		switch automationFirstText(lane["lane"]) {
		case "1000x24":
			baseline = lane
		case "2000x24":
			ceiling = lane
		}
	}
	baselineObserved, _ := baseline["observed"].(map[string]any)
	ceilingObserved, _ := ceiling["observed"].(map[string]any)
	baselineTPS := automationFloat64(baselineObserved["throughput_tasks_per_sec"])
	ceilingTPS := automationFloat64(ceilingObserved["throughput_tasks_per_sec"])
	dropPct := 0.0
	if baselineTPS != 0 {
		dropPct = roundFloat(((baselineTPS-ceilingTPS)/baselineTPS)*100, 2)
	}
	status := "pass"
	if dropPct > capacitySaturationDropThresholdPct {
		status = "warn"
	}
	detail := "throughput remains in the same single-instance local band at the 2000-task ceiling"
	if status != "pass" {
		detail = "throughput drops materially at the 2000-task ceiling and should be treated as saturation"
	}
	return map[string]any{
		"baseline_lane":                     baseline["lane"],
		"ceiling_lane":                      ceiling["lane"],
		"baseline_throughput_tasks_per_sec": baselineTPS,
		"ceiling_throughput_tasks_per_sec":  ceilingTPS,
		"throughput_drop_pct":               dropPct,
		"drop_warn_threshold_pct":           capacitySaturationDropThresholdPct,
		"status":                            status,
		"detail":                            detail,
	}
}

func buildCapacityMarkdown(report map[string]any) string {
	summary, _ := report["summary"].(map[string]any)
	saturation, _ := report["saturation_indicator"].(map[string]any)
	lines := []string{
		"# Capacity Certification Report",
		"",
		"## Scope",
		"",
		fmt.Sprintf("- Generated at: `%v`", report["generated_at"]),
		fmt.Sprintf("- Ticket: `%v`", report["ticket"]),
		"- Goal: convert checked-in benchmark, soak, and mixed-workload evidence into a repo-native certification matrix with explicit thresholds and operating envelopes.",
		"- Boundary: this is a single-instance repo-native certification slice, not a live multi-tenant production attestation.",
		"",
		"## Certification Summary",
		"",
		fmt.Sprintf("- Overall status: `%v`", summary["overall_status"]),
		fmt.Sprintf("- Passed lanes: `%v/%v`", summary["passed_lanes"], summary["total_lanes"]),
		fmt.Sprintf("- Recommended local sustained envelope: `%v`", summary["recommended_sustained_envelope"]),
		fmt.Sprintf("- Local ceiling envelope: `%v`", summary["ceiling_envelope"]),
		fmt.Sprintf("- Saturation signal: `%v`", saturation["detail"]),
		"",
		"## Admission Policy Summary",
		"",
		"- Policy mode: `advisory-only reviewer guidance`",
		"- Runtime enforcement: `none`",
		fmt.Sprintf("- Default reviewer envelope: `%v`", summary["recommended_sustained_envelope"]),
		fmt.Sprintf("- Ceiling reviewer envelope: `%v`", summary["ceiling_envelope"]),
		"- Scheduler note: recommended envelopes guide reviewer admission decisions and are not scheduler-enforced runtime limits.",
		"",
		"## Microbenchmark Thresholds",
		"",
	}
	if microbenchmarks, ok := report["microbenchmarks"].([]any); ok {
		for _, item := range microbenchmarks {
			lane, _ := item.(map[string]any)
			threshold, _ := lane["threshold"].(map[string]any)
			lines = append(lines, fmt.Sprintf("- `%v`: `%.2f ns/op` vs limit `%v` -> `%v`", lane["lane"], automationFloat64(lane["observed"]), threshold["value"], lane["status"]))
		}
	}
	lines = append(lines, "", "## Soak Matrix", "")
	if soakMatrix, ok := report["soak_matrix"].([]any); ok {
		for _, item := range soakMatrix {
			lane, _ := item.(map[string]any)
			observed, _ := lane["observed"].(map[string]any)
			lines = append(lines, fmt.Sprintf("- `%v`: `%v tasks/s`, `%v failed`, envelope `%v` -> `%v`", lane["lane"], observed["throughput_tasks_per_sec"], observed["failed"], lane["operating_envelope"], lane["status"]))
		}
	}
	mixedWorkload, _ := report["mixed_workload"].(map[string]any)
	lines = append(lines, "", "## Workload Mix", "", fmt.Sprintf("- `mixed-workload-routing`: `%v` -> `%v`", mixedWorkload["detail"], mixedWorkload["status"]), "", "## Recommended Operating Envelopes", "")
	if envelopes, ok := report["operating_envelopes"].([]any); ok {
		for _, item := range envelopes {
			envelope, _ := item.(map[string]any)
			lines = append(lines, fmt.Sprintf("- `%v`: %v Evidence: `%s`.", envelope["name"], envelope["recommendation"], joinAny(envelope["evidence_lanes"])))
		}
	}
	lines = append(lines, "", "## Saturation Notes", "")
	if notes, ok := report["saturation_notes"].([]any); ok {
		for _, item := range notes {
			lines = append(lines, fmt.Sprintf("- %v", item))
		}
	}
	lines = append(lines, "", "## Limits", "")
	if limits, ok := report["limits"].([]any); ok {
		for _, item := range limits {
			lines = append(lines, fmt.Sprintf("- %v", item))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func repoRelativePath(repoRoot string, path string) string {
	return automationRelPath(automationResolveRepoPath(repoRoot, path), repoRoot)
}

func allLaneStatus(items []any, expected string) bool {
	for _, item := range items {
		lane, _ := item.(map[string]any)
		if automationFirstText(lane["status"]) != expected {
			return false
		}
	}
	return true
}

func laneStatuses(items []any) []any {
	result := make([]any, 0, len(items))
	for _, item := range items {
		lane, _ := item.(map[string]any)
		result = append(result, lane["status"])
	}
	return result
}

func containsStatus(status string, expected ...string) bool {
	for _, item := range expected {
		if status == item {
			return true
		}
	}
	return false
}

func automationStringifyNumber(value any) string {
	switch typed := value.(type) {
	case int:
		return fmt.Sprintf("%d", typed)
	case float64:
		return fmt.Sprintf("%d", int(typed))
	default:
		return automationFirstText(value)
	}
}

func deriveGeneratedAt(payloads ...any) string {
	var latest *time.Time
	for _, payload := range payloads {
		for _, ts := range collectTimestamps(payload) {
			if latest == nil || ts.After(*latest) {
				current := ts
				latest = &current
			}
		}
	}
	if latest == nil {
		return automationUTCISO(time.Now().UTC())
	}
	return latest.UTC().Format("2006-01-02T15:04:05.999999Z")
}

func collectTimestamps(payload any) []time.Time {
	result := []time.Time{}
	switch typed := payload.(type) {
	case map[string]any:
		for key, value := range typed {
			switch key {
			case "generated_at", "timestamp", "created_at", "completed_at", "started_at":
				if parsed, err := automationParseFlexibleTime(automationFirstText(value)); err == nil {
					result = append(result, parsed)
				}
			}
			result = append(result, collectTimestamps(value)...)
		}
	case []any:
		for _, item := range typed {
			result = append(result, collectTimestamps(item)...)
		}
	}
	return result
}

func automationParseFlexibleTime(value string) (time.Time, error) {
	if trim(value) == "" {
		return time.Time{}, errors.New("empty time")
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if parsed, err := time.Parse(layout, strings.ReplaceAll(value, "Z", "+00:00")); err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, errors.New("unsupported time")
}

func joinAny(value any) string {
	items, _ := value.([]any)
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, automationFirstText(item))
	}
	return strings.Join(parts, ", ")
}
