package main

import (
	"encoding/json"
	"testing"
)

func TestMultiNodeSharedQueueBuildLiveTakeoverReportSummarizesSchemaParity(t *testing.T) {
	var report struct {
		Ticket  string `json:"ticket"`
		Status  string `json:"status"`
		Summary struct {
			ScenarioCount          int `json:"scenario_count"`
			PassingScenarios       int `json:"passing_scenarios"`
			FailingScenarios       int `json:"failing_scenarios"`
			DuplicateDeliveryCount int `json:"duplicate_delivery_count"`
			StaleWriteRejections   int `json:"stale_write_rejections"`
		} `json:"summary"`
		RequiredReportSections []any `json:"required_report_sections"`
		RemainingGaps          []any `json:"remaining_gaps"`
		CurrentPrimitives      struct {
			SharedQueueEvidence []string `json:"shared_queue_evidence"`
		} `json:"current_primitives"`
	}
	raw, err := json.Marshal(buildLiveTakeoverReport([]map[string]any{
		{
			"id":                       "scenario-a",
			"all_assertions_passed":    true,
			"duplicate_delivery_count": 1,
			"stale_write_rejections":   1,
		},
		{
			"id":                       "scenario-b",
			"all_assertions_passed":    false,
			"duplicate_delivery_count": 2,
			"stale_write_rejections":   0,
		},
	}, "docs/reports/multi-node-shared-queue-report.json"))
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("unmarshal report: %v", err)
	}

	if report.Ticket != "OPE-260" || report.Status != "live-multi-node-proof" {
		t.Fatalf("unexpected report identity: %+v", report)
	}
	if report.Summary.ScenarioCount != 2 || report.Summary.PassingScenarios != 1 || report.Summary.FailingScenarios != 1 || report.Summary.DuplicateDeliveryCount != 3 || report.Summary.StaleWriteRejections != 1 {
		t.Fatalf("unexpected summary: %+v", report.Summary)
	}
	if len(report.RequiredReportSections) == 0 || len(report.RemainingGaps) == 0 {
		t.Fatalf("expected report sections and remaining gaps: %+v", report)
	}
	if len(report.CurrentPrimitives.SharedQueueEvidence) < 2 || report.CurrentPrimitives.SharedQueueEvidence[1] != "docs/reports/multi-node-shared-queue-report.json" {
		t.Fatalf("unexpected shared queue evidence: %+v", report.CurrentPrimitives.SharedQueueEvidence)
	}
}
