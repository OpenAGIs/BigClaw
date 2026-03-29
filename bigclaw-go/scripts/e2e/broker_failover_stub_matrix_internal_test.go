package main

import "testing"

func TestBrokerFailoverStubMatrixBuildReportCoversAllScenariosAndSchema(t *testing.T) {
	report, err := buildBrokerFailoverReport(brokerFailoverRepoRoot(t))
	if err != nil {
		t.Fatalf("buildBrokerFailoverReport: %v", err)
	}

	if report["ticket"] != "OPE-272" {
		t.Fatalf("ticket = %v, want OPE-272", report["ticket"])
	}
	summary := asBrokerFailoverMap(report["summary"])
	if asBrokerFailoverInt(summary["scenario_count"]) != 8 ||
		asBrokerFailoverInt(summary["passing_scenarios"]) != 8 ||
		asBrokerFailoverInt(summary["failing_scenarios"]) != 0 {
		t.Fatalf("unexpected report summary: %+v", summary)
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
	for _, item := range asBrokerFailoverSlice(report["scenarios"]) {
		scenario := asBrokerFailoverMap(item)
		for _, key := range requiredKeys {
			if _, ok := scenario[key]; !ok {
				t.Fatalf("scenario missing key %q: %+v", key, scenario)
			}
		}
		if scenario["result"] != "passed" {
			t.Fatalf("scenario result = %v, want passed", scenario["result"])
		}
	}

	proofArtifacts := asBrokerFailoverMap(report["proof_artifacts"])
	wantArtifacts := map[string]string{
		"checkpoint_fencing_summary": defaultBrokerCheckpointFencingSummaryPath,
		"retention_boundary_summary": defaultBrokerRetentionBoundarySummaryPath,
	}
	if len(proofArtifacts) != len(wantArtifacts) {
		t.Fatalf("unexpected proof artifacts: %+v", proofArtifacts)
	}
	for key, want := range wantArtifacts {
		if proofArtifacts[key] != want {
			t.Fatalf("proof_artifacts[%s] = %v, want %q", key, proofArtifacts[key], want)
		}
	}

	rawArtifacts := asBrokerFailoverMap(report["raw_artifacts"])
	if len(rawArtifacts) != 8 {
		t.Fatalf("unexpected raw_artifacts length: %d", len(rawArtifacts))
	}
}

func TestBrokerFailoverStubMatrixCheckpointAndDurableSequenceAssertionsHoldForFencingScenarios(t *testing.T) {
	report, err := buildBrokerFailoverReport(brokerFailoverRepoRoot(t))
	if err != nil {
		t.Fatalf("buildBrokerFailoverReport: %v", err)
	}

	scenarios := map[string]map[string]any{}
	for _, item := range asBrokerFailoverSlice(report["scenarios"]) {
		scenario := asBrokerFailoverMap(item)
		scenarioID, _ := scenario["scenario_id"].(string)
		scenarios[scenarioID] = scenario
	}

	bf04 := scenarios["BF-04"]
	bf08 := scenarios["BF-08"]
	bf04After := asBrokerFailoverInt(asBrokerFailoverMap(bf04["checkpoint_after_recovery"])["durable_sequence"])
	bf04Before := asBrokerFailoverInt(asBrokerFailoverMap(bf04["checkpoint_before_fault"])["durable_sequence"])
	if bf04After < bf04Before {
		t.Fatalf("BF-04 checkpoint regressed: before=%d after=%d", bf04Before, bf04After)
	}

	checkpointLog := asBrokerFailoverMap(report["raw_artifacts"])["BF-04"]
	var fencedConsumerA bool
	for _, entry := range asBrokerFailoverSlice(asBrokerFailoverMap(checkpointLog)["checkpoint_transition_log"]) {
		row := asBrokerFailoverMap(entry)
		if row["owner_id"] == "consumer-a" && row["transition"] == "fenced" {
			fencedConsumerA = true
			break
		}
	}
	if !fencedConsumerA {
		t.Fatalf("expected fenced consumer-a entry in BF-04 raw artifacts")
	}
	if asBrokerFailoverInt(bf08["duplicate_count"]) != 1 {
		t.Fatalf("BF-08 duplicate_count = %v, want 1", bf08["duplicate_count"])
	}
	if len(asBrokerFailoverSlice(bf08["missing_event_ids"])) != 0 {
		t.Fatalf("BF-08 missing_event_ids = %+v, want none", bf08["missing_event_ids"])
	}
}

func TestBrokerFailoverStubMatrixProofSummariesProjectScenarioEvidenceIntoRolloutGateStatuses(t *testing.T) {
	repoRoot := brokerFailoverRepoRoot(t)
	checkpoint, err := buildBrokerCheckpointFencingSummary(repoRoot)
	if err != nil {
		t.Fatalf("buildBrokerCheckpointFencingSummary: %v", err)
	}
	retention, err := buildBrokerRetentionBoundarySummary(repoRoot)
	if err != nil {
		t.Fatalf("buildBrokerRetentionBoundarySummary: %v", err)
	}

	checkpointGates := mapBrokerProofGates(asBrokerFailoverSlice(checkpoint["rollout_gate_statuses"]))
	retentionGates := mapBrokerProofGates(asBrokerFailoverSlice(retention["rollout_gate_statuses"]))

	if checkpoint["ticket"] != "OPE-230" || checkpoint["proof_family"] != "checkpoint_fencing" {
		t.Fatalf("unexpected checkpoint summary identity: %+v", checkpoint)
	}
	if checkpointGates["replay_checkpoint_alignment"]["status"] != "passed" ||
		checkpointGates["retention_boundary_visibility"]["status"] != "unknown" ||
		asBrokerFailoverInt(asBrokerFailoverMap(checkpoint["summary"])["stale_write_rejections"]) != 1 {
		t.Fatalf("unexpected checkpoint summary payload: %+v", checkpoint)
	}

	if retention["ticket"] != "OPE-230" || retention["proof_family"] != "retention_boundary" {
		t.Fatalf("unexpected retention summary identity: %+v", retention)
	}
	if retentionGates["retention_boundary_visibility"]["status"] != "passed" {
		t.Fatalf("unexpected retention visibility gate: %+v", retentionGates["retention_boundary_visibility"])
	}
	focus := asBrokerFailoverSlice(retention["focus_scenarios"])
	firstFocus := asBrokerFailoverMap(focus[0])
	if resetRequired, _ := firstFocus["reset_required"].(bool); !resetRequired {
		t.Fatalf("expected reset_required in first retention focus scenario: %+v", firstFocus)
	}
	if asBrokerFailoverInt(asBrokerFailoverMap(retention["summary"])["retention_floor"]) != 3 {
		t.Fatalf("unexpected retention summary payload: %+v", retention)
	}
}

func brokerFailoverRepoRoot(t *testing.T) string {
	t.Helper()
	repoRoot, err := repoRootFromBrokerFailoverScript(brokerFailoverScriptFilePath())
	if err != nil {
		t.Fatalf("repoRootFromBrokerFailoverScript: %v", err)
	}
	return repoRoot
}

func mapBrokerProofGates(items []any) map[string]map[string]any {
	out := map[string]map[string]any{}
	for _, item := range items {
		gate := asBrokerFailoverMap(item)
		name, _ := gate["name"].(string)
		out[name] = gate
	}
	return out
}

func asBrokerFailoverInt(value any) int {
	switch cast := value.(type) {
	case int:
		return cast
	case int64:
		return int(cast)
	case float64:
		return int(cast)
	default:
		return 0
	}
}
