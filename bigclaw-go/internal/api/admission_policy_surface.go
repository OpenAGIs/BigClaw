package api

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

const (
	capacityCertificationMatrixPath = "docs/reports/capacity-certification-matrix.json"
	capacityCertificationReportPath = "docs/reports/capacity-certification-report.md"
)

type admissionPolicySurface struct {
	ReportPath           string                              `json:"report_path"`
	MatrixPath           string                              `json:"matrix_path"`
	GeneratedAt          string                              `json:"generated_at,omitempty"`
	Ticket               string                              `json:"ticket,omitempty"`
	Title                string                              `json:"title,omitempty"`
	Status               string                              `json:"status,omitempty"`
	PolicyMode           string                              `json:"policy_mode"`
	Enforced             bool                                `json:"enforced"`
	EvidenceSources      []string                            `json:"evidence_sources,omitempty"`
	Summary              admissionPolicySummary              `json:"summary"`
	RecommendedLanes     []admissionPolicyLane               `json:"recommended_lanes,omitempty"`
	CrossBatchComparison admissionPolicyCrossBatchComparison `json:"cross_batch_comparison"`
	SupportingEvidence   []admissionPolicyEvidence           `json:"supporting_evidence,omitempty"`
	Saturation           admissionPolicySaturation           `json:"saturation"`
	Limitations          []string                            `json:"limitations,omitempty"`
	Error                string                              `json:"error,omitempty"`
}

type admissionPolicySummary struct {
	OverallStatus                string `json:"overall_status,omitempty"`
	PassedLanes                  int    `json:"passed_lanes"`
	TotalLanes                   int    `json:"total_lanes"`
	RecommendedSustainedEnvelope string `json:"recommended_sustained_envelope,omitempty"`
	CeilingEnvelope              string `json:"ceiling_envelope,omitempty"`
	AdvisoryNote                 string `json:"advisory_note,omitempty"`
}

type admissionPolicyLane struct {
	Name                  string   `json:"name"`
	Lane                  string   `json:"lane,omitempty"`
	MaxQueuedTasks        int      `json:"max_queued_tasks,omitempty"`
	SubmitWorkers         int      `json:"submit_workers,omitempty"`
	ObservedThroughputTPS float64  `json:"observed_throughput_tasks_per_sec,omitempty"`
	Recommendation        string   `json:"recommendation,omitempty"`
	EvidenceLanes         []string `json:"evidence_lanes,omitempty"`
	DefaultRecommendation bool     `json:"default_recommendation,omitempty"`
	CeilingOnly           bool     `json:"ceiling_only,omitempty"`
}

type admissionPolicyEvidence struct {
	Name          string   `json:"name"`
	Status        string   `json:"status,omitempty"`
	Detail        string   `json:"detail,omitempty"`
	EvidenceLanes []string `json:"evidence_lanes,omitempty"`
}

type admissionPolicyCrossBatchComparison struct {
	Basis                 string                           `json:"basis,omitempty"`
	DefaultLane           string                           `json:"default_lane,omitempty"`
	CeilingLane           string                           `json:"ceiling_lane,omitempty"`
	HighestThroughputLane string                           `json:"highest_throughput_lane,omitempty"`
	LowestCostLane        string                           `json:"lowest_cost_lane,omitempty"`
	Lanes                 []admissionPolicyBatchComparison `json:"lanes,omitempty"`
	Notes                 []string                         `json:"notes,omitempty"`
}

type admissionPolicyBatchComparison struct {
	Lane                        string  `json:"lane"`
	OperatingEnvelope           string  `json:"operating_envelope,omitempty"`
	Status                      string  `json:"status,omitempty"`
	MaxQueuedTasks              int     `json:"max_queued_tasks,omitempty"`
	SubmitWorkers               int     `json:"submit_workers,omitempty"`
	Succeeded                   int     `json:"succeeded,omitempty"`
	ElapsedSeconds              float64 `json:"elapsed_seconds,omitempty"`
	ThroughputTasksPerSec       float64 `json:"throughput_tasks_per_sec,omitempty"`
	TotalWorkerSecondsCost      float64 `json:"total_worker_seconds_cost,omitempty"`
	WorkerSecondsCostPerTask    float64 `json:"worker_seconds_cost_per_task,omitempty"`
	ThroughputDeltaPctVsDefault float64 `json:"throughput_delta_pct_vs_default,omitempty"`
	CostDeltaPctVsDefault       float64 `json:"cost_delta_pct_vs_default,omitempty"`
}

type admissionPolicySaturation struct {
	BaselineLane                  string  `json:"baseline_lane,omitempty"`
	CeilingLane                   string  `json:"ceiling_lane,omitempty"`
	BaselineThroughputTasksPerSec float64 `json:"baseline_throughput_tasks_per_sec,omitempty"`
	CeilingThroughputTasksPerSec  float64 `json:"ceiling_throughput_tasks_per_sec,omitempty"`
	ThroughputDropPct             float64 `json:"throughput_drop_pct,omitempty"`
	Status                        string  `json:"status,omitempty"`
	Detail                        string  `json:"detail,omitempty"`
}

type capacityCertificationDocument struct {
	GeneratedAt    string `json:"generated_at"`
	Ticket         string `json:"ticket"`
	Title          string `json:"title"`
	Status         string `json:"status"`
	EvidenceInputs struct {
		BenchmarkReportPath     string   `json:"benchmark_report_path"`
		MixedWorkloadReportPath string   `json:"mixed_workload_report_path"`
		SoakReportPaths         []string `json:"soak_report_paths"`
		GeneratorScript         string   `json:"generator_script"`
	} `json:"evidence_inputs"`
	Summary struct {
		OverallStatus                string   `json:"overall_status"`
		TotalLanes                   int      `json:"total_lanes"`
		PassedLanes                  int      `json:"passed_lanes"`
		FailedLanes                  []string `json:"failed_lanes"`
		RecommendedSustainedEnvelope string   `json:"recommended_sustained_envelope"`
		CeilingEnvelope              string   `json:"ceiling_envelope"`
	} `json:"summary"`
	SoakMatrix []struct {
		Lane     string `json:"lane"`
		Scenario struct {
			Count   int `json:"count"`
			Workers int `json:"workers"`
		} `json:"scenario"`
		Observed struct {
			ElapsedSeconds        float64 `json:"elapsed_seconds"`
			ThroughputTasksPerSec float64 `json:"throughput_tasks_per_sec"`
			Succeeded             int     `json:"succeeded"`
			Failed                int     `json:"failed"`
		} `json:"observed"`
		OperatingEnvelope string `json:"operating_envelope"`
		Status            string `json:"status"`
		Detail            string `json:"detail"`
	} `json:"soak_matrix"`
	MixedWorkload struct {
		Lane   string `json:"lane"`
		Status string `json:"status"`
		Detail string `json:"detail"`
	} `json:"mixed_workload"`
	SaturationIndicator admissionPolicySaturation `json:"saturation_indicator"`
	OperatingEnvelopes  []struct {
		Name           string   `json:"name"`
		Recommendation string   `json:"recommendation"`
		EvidenceLanes  []string `json:"evidence_lanes"`
	} `json:"operating_envelopes"`
	Limits []string `json:"limits"`
}

func admissionPolicySummaryPayload() admissionPolicySurface {
	surface := admissionPolicySurface{
		ReportPath: capacityCertificationReportPath,
		MatrixPath: capacityCertificationMatrixPath,
		PolicyMode: "advisory_only",
		Enforced:   false,
	}
	matrixPath := resolveRepoRelativePath(capacityCertificationMatrixPath)
	if matrixPath == "" {
		surface.Status = "unavailable"
		surface.Error = "matrix path could not be resolved"
		return surface
	}
	contents, err := os.ReadFile(matrixPath)
	if err != nil {
		surface.Status = "unavailable"
		surface.Error = err.Error()
		return surface
	}
	var document capacityCertificationDocument
	if err := json.Unmarshal(contents, &document); err != nil {
		surface.Status = "invalid"
		surface.Error = fmt.Sprintf("decode %s: %v", capacityCertificationMatrixPath, err)
		return surface
	}

	surface.GeneratedAt = document.GeneratedAt
	surface.Ticket = document.Ticket
	surface.Title = document.Title
	surface.Status = document.Status
	surface.EvidenceSources = admissionPolicyEvidenceSources(document)
	surface.Limitations = append([]string(nil), document.Limits...)
	surface.Summary = admissionPolicySummary{
		OverallStatus:                document.Summary.OverallStatus,
		PassedLanes:                  document.Summary.PassedLanes,
		TotalLanes:                   document.Summary.TotalLanes,
		RecommendedSustainedEnvelope: document.Summary.RecommendedSustainedEnvelope,
		CeilingEnvelope:              document.Summary.CeilingEnvelope,
		AdvisoryNote:                 admissionPolicyAdvisoryNote(document.Limits),
	}
	surface.RecommendedLanes = admissionPolicyRecommendedLanes(document)
	surface.CrossBatchComparison = admissionPolicyCrossBatchComparisonPayload(document)
	surface.SupportingEvidence = admissionPolicySupportingEvidence(document)
	surface.Saturation = document.SaturationIndicator
	return surface
}

func admissionPolicyEvidenceSources(document capacityCertificationDocument) []string {
	seen := map[string]struct{}{}
	sources := make([]string, 0, 4+len(document.EvidenceInputs.SoakReportPaths))
	appendSource := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		sources = append(sources, value)
	}
	appendSource(capacityCertificationReportPath)
	appendSource(capacityCertificationMatrixPath)
	appendSource(document.EvidenceInputs.BenchmarkReportPath)
	appendSource(document.EvidenceInputs.MixedWorkloadReportPath)
	for _, path := range document.EvidenceInputs.SoakReportPaths {
		appendSource(path)
	}
	sort.Strings(sources)
	return sources
}

func admissionPolicyAdvisoryNote(limits []string) string {
	for _, item := range limits {
		if strings.Contains(strings.ToLower(item), "not an automated runtime admission policy") {
			return item
		}
	}
	return "Recommended envelopes are advisory reviewer guidance and are not runtime enforced."
}

func admissionPolicyRecommendedLanes(document capacityCertificationDocument) []admissionPolicyLane {
	envelopesByName := make(map[string]struct {
		Recommendation string
		EvidenceLanes  []string
	}, len(document.OperatingEnvelopes))
	for _, envelope := range document.OperatingEnvelopes {
		envelopesByName[envelope.Name] = struct {
			Recommendation string
			EvidenceLanes  []string
		}{
			Recommendation: envelope.Recommendation,
			EvidenceLanes:  append([]string(nil), envelope.EvidenceLanes...),
		}
	}
	lanes := make([]admissionPolicyLane, 0, 2)
	for _, soakLane := range document.SoakMatrix {
		if soakLane.OperatingEnvelope != "recommended-local-sustained" && soakLane.OperatingEnvelope != "recommended-local-ceiling" {
			continue
		}
		envelope := envelopesByName[soakLane.OperatingEnvelope]
		lanes = append(lanes, admissionPolicyLane{
			Name:                  soakLane.OperatingEnvelope,
			Lane:                  soakLane.Lane,
			MaxQueuedTasks:        soakLane.Scenario.Count,
			SubmitWorkers:         soakLane.Scenario.Workers,
			ObservedThroughputTPS: soakLane.Observed.ThroughputTasksPerSec,
			Recommendation:        envelope.Recommendation,
			EvidenceLanes:         append([]string(nil), envelope.EvidenceLanes...),
			DefaultRecommendation: soakLane.OperatingEnvelope == "recommended-local-sustained",
			CeilingOnly:           soakLane.OperatingEnvelope == "recommended-local-ceiling",
		})
	}
	sort.Slice(lanes, func(i, j int) bool {
		if lanes[i].DefaultRecommendation != lanes[j].DefaultRecommendation {
			return lanes[i].DefaultRecommendation
		}
		return lanes[i].MaxQueuedTasks < lanes[j].MaxQueuedTasks
	})
	return lanes
}

func admissionPolicySupportingEvidence(document capacityCertificationDocument) []admissionPolicyEvidence {
	supporting := make([]admissionPolicyEvidence, 0, 1)
	if strings.TrimSpace(document.MixedWorkload.Lane) != "" {
		supporting = append(supporting, admissionPolicyEvidence{
			Name:          document.MixedWorkload.Lane,
			Status:        document.MixedWorkload.Status,
			Detail:        document.MixedWorkload.Detail,
			EvidenceLanes: []string{document.MixedWorkload.Lane},
		})
	}
	return supporting
}

func admissionPolicyCrossBatchComparisonPayload(document capacityCertificationDocument) admissionPolicyCrossBatchComparison {
	comparison := admissionPolicyCrossBatchComparison{
		Basis: "worker_seconds_cost_proxy",
		Notes: []string{
			"Cost is expressed as worker-seconds consumed by each checked-in soak lane.",
			"Worker-second cost is a deterministic proxy for cross-batch efficiency, not a billing-rate estimate.",
		},
	}

	defaultLane := ""
	for _, envelope := range document.OperatingEnvelopes {
		if envelope.Name == "recommended-local-sustained" && len(envelope.EvidenceLanes) > 0 {
			defaultLane = strings.TrimSpace(envelope.EvidenceLanes[0])
			break
		}
	}
	if defaultLane == "" {
		for _, soakLane := range document.SoakMatrix {
			if soakLane.OperatingEnvelope == "recommended-local-sustained" {
				defaultLane = strings.TrimSpace(soakLane.Lane)
				break
			}
		}
	}
	comparison.DefaultLane = defaultLane
	comparison.CeilingLane = strings.TrimSpace(document.SaturationIndicator.CeilingLane)

	defaultThroughput := 0.0
	defaultCostPerTask := 0.0
	for _, soakLane := range document.SoakMatrix {
		if strings.TrimSpace(soakLane.Lane) != defaultLane {
			continue
		}
		defaultThroughput = soakLane.Observed.ThroughputTasksPerSec
		if soakLane.Observed.Succeeded > 0 {
			defaultCostPerTask = (soakLane.Observed.ElapsedSeconds * float64(soakLane.Scenario.Workers)) / float64(soakLane.Observed.Succeeded)
		}
		break
	}

	bestThroughput := -1.0
	bestThroughputLane := ""
	lowestCost := -1.0
	lowestCostLane := ""
	comparison.Lanes = make([]admissionPolicyBatchComparison, 0, len(document.SoakMatrix))
	for _, soakLane := range document.SoakMatrix {
		totalWorkerSeconds := soakLane.Observed.ElapsedSeconds * float64(soakLane.Scenario.Workers)
		costPerTask := 0.0
		if soakLane.Observed.Succeeded > 0 {
			costPerTask = totalWorkerSeconds / float64(soakLane.Observed.Succeeded)
		}
		lane := admissionPolicyBatchComparison{
			Lane:                     soakLane.Lane,
			OperatingEnvelope:        soakLane.OperatingEnvelope,
			Status:                   soakLane.Status,
			MaxQueuedTasks:           soakLane.Scenario.Count,
			SubmitWorkers:            soakLane.Scenario.Workers,
			Succeeded:                soakLane.Observed.Succeeded,
			ElapsedSeconds:           soakLane.Observed.ElapsedSeconds,
			ThroughputTasksPerSec:    soakLane.Observed.ThroughputTasksPerSec,
			TotalWorkerSecondsCost:   totalWorkerSeconds,
			WorkerSecondsCostPerTask: costPerTask,
		}
		if defaultThroughput > 0 {
			lane.ThroughputDeltaPctVsDefault = ((lane.ThroughputTasksPerSec - defaultThroughput) / defaultThroughput) * 100
		}
		if defaultCostPerTask > 0 {
			lane.CostDeltaPctVsDefault = ((lane.WorkerSecondsCostPerTask - defaultCostPerTask) / defaultCostPerTask) * 100
		}
		comparison.Lanes = append(comparison.Lanes, lane)
		if lane.ThroughputTasksPerSec > bestThroughput {
			bestThroughput = lane.ThroughputTasksPerSec
			bestThroughputLane = lane.Lane
		}
		if lowestCost < 0 || lane.WorkerSecondsCostPerTask < lowestCost {
			lowestCost = lane.WorkerSecondsCostPerTask
			lowestCostLane = lane.Lane
		}
	}
	sort.Slice(comparison.Lanes, func(i, j int) bool {
		if comparison.Lanes[i].MaxQueuedTasks == comparison.Lanes[j].MaxQueuedTasks {
			return comparison.Lanes[i].SubmitWorkers < comparison.Lanes[j].SubmitWorkers
		}
		return comparison.Lanes[i].MaxQueuedTasks < comparison.Lanes[j].MaxQueuedTasks
	})
	comparison.HighestThroughputLane = bestThroughputLane
	comparison.LowestCostLane = lowestCostLane
	return comparison
}
