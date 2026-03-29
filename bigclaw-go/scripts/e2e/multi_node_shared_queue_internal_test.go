package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMultiNodeSharedQueueBuildLiveTakeoverReportSummarizesSchemaParity(t *testing.T) {
	output := runMultiNodeSharedQueuePythonSnippet(t, `
import json

scenarios = [
    {
        'id': 'scenario-a',
        'all_assertions_passed': True,
        'duplicate_delivery_count': 1,
        'stale_write_rejections': 1,
    },
    {
        'id': 'scenario-b',
        'all_assertions_passed': False,
        'duplicate_delivery_count': 2,
        'stale_write_rejections': 0,
    },
]

print(json.dumps(MODULE.build_live_takeover_report(
    scenarios,
    'docs/reports/multi-node-shared-queue-report.json',
)))
`)

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
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("unmarshal report: %v\n%s", err, output)
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

func runMultiNodeSharedQueuePythonSnippet(t *testing.T, snippet string) string {
	t.Helper()
	modulePath := filepath.Join(multiNodeSharedQueueRepoRoot(t), "scripts", "e2e", "multi_node_shared_queue.py")
	script := `
import importlib.util
import pathlib

MODULE_PATH = pathlib.Path(r"` + modulePath + `")
SPEC = importlib.util.spec_from_file_location('multi_node_shared_queue', MODULE_PATH)
if SPEC is None or SPEC.loader is None:
    raise RuntimeError(f'unable to load module from {MODULE_PATH}')
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)
` + "\n" + strings.TrimSpace(snippet) + "\n"

	cmd := exec.Command("python3", "-")
	cmd.Stdin = strings.NewReader(script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("python snippet failed: %v\n%s", err, string(output))
	}
	return string(output)
}

func multiNodeSharedQueueRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}
