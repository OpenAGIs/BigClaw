package regression

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCrossProcessCoordinationSurfaceBuilderStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	tmpDir := t.TempDir()
	multiNodePath := filepath.Join(tmpDir, "multi-node.json")
	takeoverPath := filepath.Join(tmpDir, "takeover.json")
	liveTakeoverPath := filepath.Join(tmpDir, "live-takeover.json")
	outputPath := filepath.Join(tmpDir, "coordination-surface.json")

	writeFile := func(path, contents string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	writeFile(multiNodePath, `{"count":200,"cross_node_completions":99,"duplicate_completed_tasks":[],"duplicate_started_tasks":[]}`)
	writeFile(takeoverPath, `{"summary":{"scenario_count":3,"passing_scenarios":3,"duplicate_delivery_count":0,"stale_write_rejections":2}}`)
	writeFile(liveTakeoverPath, `{"summary":{"scenario_count":3,"passing_scenarios":3,"stale_write_rejections":2}}`)

	scriptPath := filepath.Join(repoRoot, "scripts", "e2e", "cross-process-coordination-surface")
	cmd := exec.Command(
		"bash", scriptPath,
		"--multi-node-report", multiNodePath,
		"--takeover-report", takeoverPath,
		"--live-takeover-report", liveTakeoverPath,
		"--output", outputPath,
	)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run coordination surface builder: %v\n%s", err, output)
	}

	var report struct {
		Status                string `json:"status"`
		TargetContractVersion string `json:"target_contract_surface_version"`
		Summary               struct {
			SharedQueueCrossNodeCompletions int `json:"shared_queue_cross_node_completions"`
			TakeoverScenarioCount           int `json:"takeover_scenario_count"`
			LiveTakeoverScenarioCount       int `json:"live_takeover_scenario_count"`
			TakeoverStaleWriteRejections    int `json:"takeover_stale_write_rejections"`
		} `json:"summary"`
		Capabilities []struct {
			Capability       string `json:"capability"`
			RuntimeReadiness string `json:"runtime_readiness"`
			CurrentState     string `json:"current_state"`
		} `json:"capabilities"`
		CurrentCeiling []string `json:"current_ceiling"`
	}
	readJSONFile(t, outputPath, &report)

	if report.Status != "local-capability-surface" || report.TargetContractVersion != "2026-03-17" {
		t.Fatalf("unexpected coordination surface identity: %+v", report)
	}
	if report.Summary.SharedQueueCrossNodeCompletions != 99 || report.Summary.TakeoverScenarioCount != 3 || report.Summary.LiveTakeoverScenarioCount != 3 || report.Summary.TakeoverStaleWriteRejections != 2 {
		t.Fatalf("unexpected coordination summary: %+v", report.Summary)
	}
	if len(report.Capabilities) != 7 {
		t.Fatalf("unexpected capability count: %+v", report.Capabilities)
	}
	foundPartitioned := false
	foundSharedQueue := false
	for _, capability := range report.Capabilities {
		if capability.Capability == "partitioned_topic_routing" && capability.CurrentState == "not_available" && capability.RuntimeReadiness == "contract_only" {
			foundPartitioned = true
		}
		if capability.Capability == "shared_queue_task_coordination" && capability.RuntimeReadiness == "live_proven" {
			foundSharedQueue = true
		}
	}
	if !foundPartitioned || !foundSharedQueue {
		t.Fatalf("unexpected capability set: %+v", report.Capabilities)
	}
	if len(report.CurrentCeiling) != 3 {
		t.Fatalf("unexpected current ceiling: %+v", report.CurrentCeiling)
	}
}
