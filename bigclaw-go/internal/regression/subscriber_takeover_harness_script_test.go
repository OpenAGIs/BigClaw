package regression

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSubscriberTakeoverHarnessScriptBuildsExpectedReport(t *testing.T) {
	repoRoot := repoRoot(t)
	outputPath := filepath.Join(t.TempDir(), "multi-subscriber-takeover-validation-report.json")
	scriptPath := filepath.Join(repoRoot, "scripts", "e2e", "subscriber-takeover-fault-matrix")

	cmd := exec.Command("bash", scriptPath, "--output", outputPath)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run takeover harness script: %v\n%s", err, output)
	}

	var report struct {
		Status  string `json:"status"`
		Summary struct {
			ScenarioCount          int `json:"scenario_count"`
			PassingScenarios       int `json:"passing_scenarios"`
			FailingScenarios       int `json:"failing_scenarios"`
			StaleWriteRejections   int `json:"stale_write_rejections"`
			DuplicateDeliveryCount int `json:"duplicate_delivery_count"`
		} `json:"summary"`
		CurrentPrimitives struct {
			TakeoverHarness []string `json:"takeover_harness"`
		} `json:"current_primitives"`
	}
	readJSONFile(t, outputPath, &report)

	if report.Status != "local-executable" ||
		report.Summary.ScenarioCount != 3 ||
		report.Summary.PassingScenarios != 3 ||
		report.Summary.FailingScenarios != 0 ||
		report.Summary.StaleWriteRejections != 2 ||
		report.Summary.DuplicateDeliveryCount != 4 {
		t.Fatalf("unexpected generated takeover report: %+v", report)
	}
	if len(report.CurrentPrimitives.TakeoverHarness) != 2 || report.CurrentPrimitives.TakeoverHarness[0] != "scripts/e2e/subscriber-takeover-fault-matrix" {
		t.Fatalf("unexpected takeover harness primitive path: %+v", report.CurrentPrimitives.TakeoverHarness)
	}
}
