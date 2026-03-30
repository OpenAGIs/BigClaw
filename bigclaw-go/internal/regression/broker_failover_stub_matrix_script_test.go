package regression

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestBrokerFailoverStubMatrixBuildersStayAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	scriptPath := filepath.Join(repoRoot, "scripts", "e2e", "broker-failover-stub-matrix")
	outputDir := t.TempDir()
	reportPath := filepath.Join(outputDir, "broker-failover-stub-report.json")
	checkpointPath := filepath.Join(outputDir, "broker-checkpoint-fencing-proof-summary.json")
	retentionPath := filepath.Join(outputDir, "broker-retention-boundary-proof-summary.json")
	cmd := exec.Command(
		"bash", scriptPath,
		"--output", reportPath,
		"--checkpoint-summary-output", checkpointPath,
		"--retention-summary-output", retentionPath,
	)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run broker failover stub matrix: %v\n%s", err, output)
	}

	var report struct {
		Ticket  string `json:"ticket"`
		Summary struct {
			ScenarioCount    int `json:"scenario_count"`
			PassingScenarios int `json:"passing_scenarios"`
			FailingScenarios int `json:"failing_scenarios"`
		} `json:"summary"`
		Scenarios []struct {
			ScenarioID            string   `json:"scenario_id"`
			Result                string   `json:"result"`
			DuplicateCount        int      `json:"duplicate_count"`
			MissingEventIDs       []string `json:"missing_event_ids"`
			CheckpointBeforeFault struct {
				DurableSequence int `json:"durable_sequence"`
			} `json:"checkpoint_before_fault"`
			CheckpointAfterRecovery struct {
				DurableSequence int `json:"durable_sequence"`
			} `json:"checkpoint_after_recovery"`
		} `json:"scenarios"`
		ProofArtifacts map[string]string `json:"proof_artifacts"`
	}
	var checkpoint struct {
		Ticket      string `json:"ticket"`
		ProofFamily string `json:"proof_family"`
		Summary     struct {
			StaleWriteRejections int `json:"stale_write_rejections"`
		} `json:"summary"`
		RolloutGateStatuses []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"rollout_gate_statuses"`
	}
	var retention struct {
		Ticket      string `json:"ticket"`
		ProofFamily string `json:"proof_family"`
		Summary     struct {
			RetentionFloor int `json:"retention_floor"`
		} `json:"summary"`
		FocusScenarios []struct {
			ResetRequired bool `json:"reset_required"`
		} `json:"focus_scenarios"`
		RolloutGateStatuses []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"rollout_gate_statuses"`
	}
	readJSONFile(t, reportPath, &report)
	readJSONFile(t, checkpointPath, &checkpoint)
	readJSONFile(t, retentionPath, &retention)
	for _, path := range []string{reportPath, checkpointPath, retentionPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected generated artifact %s: %v", path, err)
		}
	}

	if report.Ticket != "OPE-272" {
		t.Fatalf("unexpected report ticket: %+v", report)
	}
	if report.Summary.ScenarioCount != 8 || report.Summary.PassingScenarios != 8 || report.Summary.FailingScenarios != 0 {
		t.Fatalf("unexpected report summary: %+v", report.Summary)
	}
	if report.ProofArtifacts["checkpoint_fencing_summary"] != "bigclaw-go/docs/reports/broker-checkpoint-fencing-proof-summary.json" ||
		report.ProofArtifacts["retention_boundary_summary"] != "bigclaw-go/docs/reports/broker-retention-boundary-proof-summary.json" {
		t.Fatalf("unexpected proof artifacts: %+v", report.ProofArtifacts)
	}

	scenarios := map[string]struct {
		Result                  string
		DuplicateCount          int
		MissingEventIDs         []string
		CheckpointBeforeFault   int
		CheckpointAfterRecovery int
	}{}
	for _, scenario := range report.Scenarios {
		scenarios[scenario.ScenarioID] = struct {
			Result                  string
			DuplicateCount          int
			MissingEventIDs         []string
			CheckpointBeforeFault   int
			CheckpointAfterRecovery int
		}{
			Result:                  scenario.Result,
			DuplicateCount:          scenario.DuplicateCount,
			MissingEventIDs:         scenario.MissingEventIDs,
			CheckpointBeforeFault:   scenario.CheckpointBeforeFault.DurableSequence,
			CheckpointAfterRecovery: scenario.CheckpointAfterRecovery.DurableSequence,
		}
	}
	if len(scenarios) != 8 {
		t.Fatalf("unexpected scenario count: %+v", scenarios)
	}
	for id, scenario := range scenarios {
		if scenario.Result != "passed" {
			t.Fatalf("scenario %s not passed: %+v", id, scenario)
		}
	}
	bf04 := scenarios["BF-04"]
	if bf04.CheckpointAfterRecovery < bf04.CheckpointBeforeFault {
		t.Fatalf("BF-04 checkpoint regressed: %+v", bf04)
	}
	bf08 := scenarios["BF-08"]
	if bf08.DuplicateCount != 1 || len(bf08.MissingEventIDs) != 0 {
		t.Fatalf("unexpected BF-08 counters: %+v", bf08)
	}

	if checkpoint.Ticket != "OPE-230" || checkpoint.ProofFamily != "checkpoint_fencing" {
		t.Fatalf("unexpected checkpoint summary identity: %+v", checkpoint)
	}
	checkpointGates := map[string]string{}
	for _, gate := range checkpoint.RolloutGateStatuses {
		checkpointGates[gate.Name] = gate.Status
	}
	if checkpointGates["replay_checkpoint_alignment"] != "passed" || checkpointGates["retention_boundary_visibility"] != "unknown" {
		t.Fatalf("unexpected checkpoint gates: %+v", checkpointGates)
	}
	if checkpoint.Summary.StaleWriteRejections != 1 {
		t.Fatalf("unexpected checkpoint summary: %+v", checkpoint.Summary)
	}

	if retention.Ticket != "OPE-230" || retention.ProofFamily != "retention_boundary" {
		t.Fatalf("unexpected retention summary identity: %+v", retention)
	}
	retentionGates := map[string]string{}
	for _, gate := range retention.RolloutGateStatuses {
		retentionGates[gate.Name] = gate.Status
	}
	if retentionGates["retention_boundary_visibility"] != "passed" {
		t.Fatalf("unexpected retention gates: %+v", retentionGates)
	}
	if len(retention.FocusScenarios) == 0 || !retention.FocusScenarios[0].ResetRequired {
		t.Fatalf("unexpected retention focus scenarios: %+v", retention.FocusScenarios)
	}
	if retention.Summary.RetentionFloor != 3 {
		t.Fatalf("unexpected retention summary: %+v", retention.Summary)
	}
}
