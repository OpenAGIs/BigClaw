package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBrokerFailoverStubMatrixBuildReportCoversAllScenariosAndSchema(t *testing.T) {
	output := runBrokerFailoverPythonSnippet(t, `
import json
print(json.dumps(MODULE.build_report()))
`)

	var report struct {
		Ticket  string `json:"ticket"`
		Summary struct {
			ScenarioCount    int `json:"scenario_count"`
			PassingScenarios int `json:"passing_scenarios"`
			FailingScenarios int `json:"failing_scenarios"`
		} `json:"summary"`
		Scenarios      []map[string]any  `json:"scenarios"`
		ProofArtifacts map[string]string `json:"proof_artifacts"`
	}
	decodeBrokerFailoverJSON(t, output, &report)

	if report.Ticket != "OPE-272" || report.Summary.ScenarioCount != 8 || report.Summary.PassingScenarios != 8 || report.Summary.FailingScenarios != 0 {
		t.Fatalf("unexpected report summary: %+v", report)
	}
	requiredKeys := []string{
		"scenario_id",
		"backend",
		"topology",
		"fault_window",
		"published_count",
		"committed_count",
		"replayed_count",
		"duplicate_count",
		"missing_event_ids",
		"checkpoint_before_fault",
		"checkpoint_after_recovery",
		"lease_transitions",
		"publish_outcomes",
		"replay_resume_cursor",
		"artifacts",
		"result",
		"assertions",
	}
	for _, scenario := range report.Scenarios {
		for _, key := range requiredKeys {
			if _, ok := scenario[key]; !ok {
				t.Fatalf("scenario missing key %q: %+v", key, scenario)
			}
		}
		if scenario["result"] != "passed" {
			t.Fatalf("scenario result = %v, want passed", scenario["result"])
		}
	}
	wantArtifacts := map[string]string{
		"checkpoint_fencing_summary": "bigclaw-go/docs/reports/broker-checkpoint-fencing-proof-summary.json",
		"retention_boundary_summary": "bigclaw-go/docs/reports/broker-retention-boundary-proof-summary.json",
	}
	if len(report.ProofArtifacts) != len(wantArtifacts) {
		t.Fatalf("unexpected proof artifacts: %+v", report.ProofArtifacts)
	}
	for key, want := range wantArtifacts {
		if report.ProofArtifacts[key] != want {
			t.Fatalf("proof_artifacts[%s] = %q, want %q", key, report.ProofArtifacts[key], want)
		}
	}
}

func TestBrokerFailoverStubMatrixCheckpointAndDurableSequenceAssertionsHoldForFencingScenarios(t *testing.T) {
	output := runBrokerFailoverPythonSnippet(t, `
import json
report = MODULE.build_report()
scenarios = {scenario['scenario_id']: scenario for scenario in report['scenarios']}
print(json.dumps({
    'bf04_after': scenarios['BF-04']['checkpoint_after_recovery']['durable_sequence'],
    'bf04_before': scenarios['BF-04']['checkpoint_before_fault']['durable_sequence'],
    'bf04_has_fenced_consumer_a': any(
        entry['owner_id'] == 'consumer-a' and entry['transition'] == 'fenced'
        for entry in report['raw_artifacts']['BF-04']['checkpoint_transition_log']
    ),
    'bf08_duplicate_count': scenarios['BF-08']['duplicate_count'],
    'bf08_missing_event_ids': scenarios['BF-08']['missing_event_ids'],
}))
`)

	var payload struct {
		BF04After             int   `json:"bf04_after"`
		BF04Before            int   `json:"bf04_before"`
		BF04HasFencedConsumer bool  `json:"bf04_has_fenced_consumer_a"`
		BF08DuplicateCount    int   `json:"bf08_duplicate_count"`
		BF08MissingEventIDs   []any `json:"bf08_missing_event_ids"`
	}
	decodeBrokerFailoverJSON(t, output, &payload)

	if payload.BF04After < payload.BF04Before || !payload.BF04HasFencedConsumer || payload.BF08DuplicateCount != 1 || len(payload.BF08MissingEventIDs) != 0 {
		t.Fatalf("unexpected fencing payload: %+v", payload)
	}
}

func TestBrokerFailoverStubMatrixProofSummariesProjectScenarioEvidenceIntoRolloutGateStatuses(t *testing.T) {
	output := runBrokerFailoverPythonSnippet(t, `
import json
report = MODULE.build_report()
checkpoint = MODULE.build_checkpoint_fencing_summary(report)
retention = MODULE.build_retention_boundary_summary(report)
checkpoint_gates = {gate['name']: gate for gate in checkpoint['rollout_gate_statuses']}
retention_gates = {gate['name']: gate for gate in retention['rollout_gate_statuses']}
print(json.dumps({
    'checkpoint_ticket': checkpoint['ticket'],
    'checkpoint_proof_family': checkpoint['proof_family'],
    'checkpoint_replay_alignment': checkpoint_gates['replay_checkpoint_alignment']['status'],
    'checkpoint_retention_visibility': checkpoint_gates['retention_boundary_visibility']['status'],
    'checkpoint_stale_write_rejections': checkpoint['summary']['stale_write_rejections'],
    'retention_ticket': retention['ticket'],
    'retention_proof_family': retention['proof_family'],
    'retention_visibility': retention_gates['retention_boundary_visibility']['status'],
    'retention_reset_required': retention['focus_scenarios'][0]['reset_required'],
    'retention_floor': retention['summary']['retention_floor'],
}))
`)

	var payload struct {
		CheckpointTicket               string `json:"checkpoint_ticket"`
		CheckpointProofFamily          string `json:"checkpoint_proof_family"`
		CheckpointReplayAlignment      string `json:"checkpoint_replay_alignment"`
		CheckpointRetentionVisibility  string `json:"checkpoint_retention_visibility"`
		CheckpointStaleWriteRejections int    `json:"checkpoint_stale_write_rejections"`
		RetentionTicket                string `json:"retention_ticket"`
		RetentionProofFamily           string `json:"retention_proof_family"`
		RetentionVisibility            string `json:"retention_visibility"`
		RetentionResetRequired         bool   `json:"retention_reset_required"`
		RetentionFloor                 int    `json:"retention_floor"`
	}
	decodeBrokerFailoverJSON(t, output, &payload)

	if payload.CheckpointTicket != "OPE-230" || payload.CheckpointProofFamily != "checkpoint_fencing" || payload.CheckpointReplayAlignment != "passed" || payload.CheckpointRetentionVisibility != "unknown" || payload.CheckpointStaleWriteRejections != 1 {
		t.Fatalf("unexpected checkpoint payload: %+v", payload)
	}
	if payload.RetentionTicket != "OPE-230" || payload.RetentionProofFamily != "retention_boundary" || payload.RetentionVisibility != "passed" || !payload.RetentionResetRequired || payload.RetentionFloor != 3 {
		t.Fatalf("unexpected retention payload: %+v", payload)
	}
}

func runBrokerFailoverPythonSnippet(t *testing.T, snippet string) string {
	t.Helper()
	modulePath := filepath.Join(brokerFailoverRepoRoot(t), "scripts", "e2e", "broker_failover_stub_matrix.py")
	script := `
import importlib.util
import pathlib

MODULE_PATH = pathlib.Path(r"` + modulePath + `")
SPEC = importlib.util.spec_from_file_location('broker_failover_stub_matrix', MODULE_PATH)
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

func brokerFailoverRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func decodeBrokerFailoverJSON(t *testing.T, output string, target any) {
	t.Helper()
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), target); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, output)
	}
}
