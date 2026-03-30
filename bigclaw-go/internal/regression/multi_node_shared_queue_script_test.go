package regression

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestMultiNodeSharedQueueLiveTakeoverBuilderStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	scriptPath := filepath.Join(repoRoot, "scripts", "e2e", "multi_node_shared_queue.py")
	code := `
import importlib.util
import json
import pathlib

script_path = pathlib.Path(r"` + filepath.ToSlash(scriptPath) + `")
spec = importlib.util.spec_from_file_location("multi_node_shared_queue", script_path)
module = importlib.util.module_from_spec(spec)
assert spec.loader is not None
spec.loader.exec_module(module)
report = module.build_live_takeover_report(
    [
        {
            "id": "scenario-a",
            "all_assertions_passed": True,
            "duplicate_delivery_count": 1,
            "stale_write_rejections": 1,
        },
        {
            "id": "scenario-b",
            "all_assertions_passed": False,
            "duplicate_delivery_count": 2,
            "stale_write_rejections": 0,
        },
    ],
    "docs/reports/multi-node-shared-queue-report.json",
)
print(json.dumps(report))
`
	cmd := exec.Command("python3", "-c", code)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run multi-node shared queue builder: %v\n%s", err, output)
	}

	var report struct {
		Ticket            string   `json:"ticket"`
		Status            string   `json:"status"`
		RequiredSections  []string `json:"required_report_sections"`
		RemainingGaps     []string `json:"remaining_gaps"`
		CurrentPrimitives struct {
			SharedQueueEvidence []string `json:"shared_queue_evidence"`
		} `json:"current_primitives"`
		Summary struct {
			ScenarioCount          int `json:"scenario_count"`
			PassingScenarios       int `json:"passing_scenarios"`
			FailingScenarios       int `json:"failing_scenarios"`
			DuplicateDeliveryCount int `json:"duplicate_delivery_count"`
			StaleWriteRejections   int `json:"stale_write_rejections"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("decode builder output: %v\n%s", err, output)
	}

	if report.Ticket != "OPE-260" || report.Status != "live-multi-node-proof" {
		t.Fatalf("unexpected report identity: %+v", report)
	}
	if report.Summary.ScenarioCount != 2 || report.Summary.PassingScenarios != 1 || report.Summary.FailingScenarios != 1 {
		t.Fatalf("unexpected scenario summary: %+v", report.Summary)
	}
	if report.Summary.DuplicateDeliveryCount != 3 || report.Summary.StaleWriteRejections != 1 {
		t.Fatalf("unexpected aggregate counters: %+v", report.Summary)
	}
	if len(report.RequiredSections) == 0 {
		t.Fatalf("expected required report sections, got %+v", report)
	}
	if len(report.RemainingGaps) == 0 {
		t.Fatalf("expected remaining gaps, got %+v", report)
	}
	if len(report.CurrentPrimitives.SharedQueueEvidence) < 2 || report.CurrentPrimitives.SharedQueueEvidence[1] != "docs/reports/multi-node-shared-queue-report.json" {
		t.Fatalf("unexpected shared queue evidence: %+v", report.CurrentPrimitives.SharedQueueEvidence)
	}
}
