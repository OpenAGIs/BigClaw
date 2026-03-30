package regression

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestBrokerFailoverStubMatrixBuildersStayAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	scriptPath := filepath.Join(repoRoot, "scripts", "e2e", "broker_failover_stub_matrix.py")
	code := `
import importlib.util
import json
import pathlib

script_path = pathlib.Path(r"` + filepath.ToSlash(scriptPath) + `")
spec = importlib.util.spec_from_file_location("broker_failover_stub_matrix", script_path)
module = importlib.util.module_from_spec(spec)
assert spec.loader is not None
spec.loader.exec_module(module)

report = module.build_report()
checkpoint = module.build_checkpoint_fencing_summary(report)
retention = module.build_retention_boundary_summary(report)

print(json.dumps({
    "report": report,
    "checkpoint": checkpoint,
    "retention": retention,
}))
`
	cmd := exec.Command("python3", "-c", code)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run broker failover builders: %v\n%s", err, output)
	}

	var payload struct {
		Report struct {
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
			RawArtifacts   map[string]struct {
				CheckpointTransitionLog []struct {
					OwnerID    string `json:"owner_id"`
					Transition string `json:"transition"`
				} `json:"checkpoint_transition_log"`
			} `json:"raw_artifacts"`
		} `json:"report"`
		Checkpoint struct {
			Ticket      string `json:"ticket"`
			ProofFamily string `json:"proof_family"`
			Summary     struct {
				StaleWriteRejections int `json:"stale_write_rejections"`
			} `json:"summary"`
			RolloutGateStatuses []struct {
				Name   string `json:"name"`
				Status string `json:"status"`
			} `json:"rollout_gate_statuses"`
		} `json:"checkpoint"`
		Retention struct {
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
		} `json:"retention"`
	}
	if err := json.Unmarshal(output, &payload); err != nil {
		t.Fatalf("decode broker failover payload: %v\n%s", err, output)
	}

	if payload.Report.Ticket != "OPE-272" {
		t.Fatalf("unexpected report ticket: %+v", payload.Report)
	}
	if payload.Report.Summary.ScenarioCount != 8 || payload.Report.Summary.PassingScenarios != 8 || payload.Report.Summary.FailingScenarios != 0 {
		t.Fatalf("unexpected report summary: %+v", payload.Report.Summary)
	}
	if payload.Report.ProofArtifacts["checkpoint_fencing_summary"] != "bigclaw-go/docs/reports/broker-checkpoint-fencing-proof-summary.json" ||
		payload.Report.ProofArtifacts["retention_boundary_summary"] != "bigclaw-go/docs/reports/broker-retention-boundary-proof-summary.json" {
		t.Fatalf("unexpected proof artifacts: %+v", payload.Report.ProofArtifacts)
	}

	scenarios := map[string]struct {
		Result                  string
		DuplicateCount          int
		MissingEventIDs         []string
		CheckpointBeforeFault   int
		CheckpointAfterRecovery int
	}{}
	for _, scenario := range payload.Report.Scenarios {
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
	foundFence := false
	for _, entry := range payload.Report.RawArtifacts["BF-04"].CheckpointTransitionLog {
		if entry.OwnerID == "consumer-a" && entry.Transition == "fenced" {
			foundFence = true
			break
		}
	}
	if !foundFence {
		t.Fatalf("BF-04 missing fence transition: %+v", payload.Report.RawArtifacts["BF-04"])
	}
	bf08 := scenarios["BF-08"]
	if bf08.DuplicateCount != 1 || len(bf08.MissingEventIDs) != 0 {
		t.Fatalf("unexpected BF-08 counters: %+v", bf08)
	}

	if payload.Checkpoint.Ticket != "OPE-230" || payload.Checkpoint.ProofFamily != "checkpoint_fencing" {
		t.Fatalf("unexpected checkpoint summary identity: %+v", payload.Checkpoint)
	}
	checkpointGates := map[string]string{}
	for _, gate := range payload.Checkpoint.RolloutGateStatuses {
		checkpointGates[gate.Name] = gate.Status
	}
	if checkpointGates["replay_checkpoint_alignment"] != "passed" || checkpointGates["retention_boundary_visibility"] != "unknown" {
		t.Fatalf("unexpected checkpoint gates: %+v", checkpointGates)
	}
	if payload.Checkpoint.Summary.StaleWriteRejections != 1 {
		t.Fatalf("unexpected checkpoint summary: %+v", payload.Checkpoint.Summary)
	}

	if payload.Retention.Ticket != "OPE-230" || payload.Retention.ProofFamily != "retention_boundary" {
		t.Fatalf("unexpected retention summary identity: %+v", payload.Retention)
	}
	retentionGates := map[string]string{}
	for _, gate := range payload.Retention.RolloutGateStatuses {
		retentionGates[gate.Name] = gate.Status
	}
	if retentionGates["retention_boundary_visibility"] != "passed" {
		t.Fatalf("unexpected retention gates: %+v", retentionGates)
	}
	if len(payload.Retention.FocusScenarios) == 0 || !payload.Retention.FocusScenarios[0].ResetRequired {
		t.Fatalf("unexpected retention focus scenarios: %+v", payload.Retention.FocusScenarios)
	}
	if payload.Retention.Summary.RetentionFloor != 3 {
		t.Fatalf("unexpected retention summary: %+v", payload.Retention.Summary)
	}
}
