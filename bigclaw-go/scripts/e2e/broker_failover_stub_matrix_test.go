package main

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func testScriptDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		t.Fatal("failed to resolve caller file")
	}
	return filepath.Dir(file)
}

func runPythonJSON(t *testing.T, code string) map[string]any {
	t.Helper()
	cmd := exec.Command("python3", "-c", code)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("python3 failed: %v\n%s", err, string(out))
	}
	var payload map[string]any
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("failed to parse python JSON output: %v\n%s", err, string(out))
	}
	return payload
}

func requireString(t *testing.T, payload map[string]any, key string) string {
	t.Helper()
	value, ok := payload[key].(string)
	if !ok {
		t.Fatalf("expected %s to be string, got %T", key, payload[key])
	}
	return value
}

func requireNumber(t *testing.T, payload map[string]any, key string) float64 {
	t.Helper()
	value, ok := payload[key].(float64)
	if !ok {
		t.Fatalf("expected %s to be number, got %T", key, payload[key])
	}
	return value
}

func requireBool(t *testing.T, payload map[string]any, key string) bool {
	t.Helper()
	value, ok := payload[key].(bool)
	if !ok {
		t.Fatalf("expected %s to be bool, got %T", key, payload[key])
	}
	return value
}

func TestBuildReportCoversAllScenariosAndSchema(t *testing.T) {
	scriptPath := filepath.Join(testScriptDir(t), "broker_failover_stub_matrix.py")
	code := `
import importlib.util
import json
import pathlib

path = pathlib.Path(r"""` + scriptPath + `""")
spec = importlib.util.spec_from_file_location("broker_failover_stub_matrix", path)
if spec is None or spec.loader is None:
    raise RuntimeError(f"Unable to load module from {path}")
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
report = module.build_report()
required_keys = [
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
]
all_have_required = all(set(required_keys).issubset(set(item.keys())) for item in report["scenarios"])
all_passed = all(item["result"] == "passed" for item in report["scenarios"])
print(json.dumps({
    "ticket": report["ticket"],
    "scenario_count": report["summary"]["scenario_count"],
    "passing_scenarios": report["summary"]["passing_scenarios"],
    "failing_scenarios": report["summary"]["failing_scenarios"],
    "all_have_required_keys": all_have_required,
    "all_scenarios_passed": all_passed,
    "proof_artifacts": report["proof_artifacts"],
}))
`
	payload := runPythonJSON(t, code)
	if got := requireString(t, payload, "ticket"); got != "OPE-272" {
		t.Fatalf("unexpected ticket: %s", got)
	}
	if got := requireNumber(t, payload, "scenario_count"); got != 8 {
		t.Fatalf("unexpected scenario_count: %v", got)
	}
	if got := requireNumber(t, payload, "passing_scenarios"); got != 8 {
		t.Fatalf("unexpected passing_scenarios: %v", got)
	}
	if got := requireNumber(t, payload, "failing_scenarios"); got != 0 {
		t.Fatalf("unexpected failing_scenarios: %v", got)
	}
	if !requireBool(t, payload, "all_have_required_keys") {
		t.Fatal("expected all scenarios to contain required schema keys")
	}
	if !requireBool(t, payload, "all_scenarios_passed") {
		t.Fatal("expected all scenarios to pass")
	}
	proofArtifacts, ok := payload["proof_artifacts"].(map[string]any)
	if !ok {
		t.Fatalf("expected proof_artifacts object, got %T", payload["proof_artifacts"])
	}
	if got, ok := proofArtifacts["checkpoint_fencing_summary"].(string); !ok || got != "bigclaw-go/docs/reports/broker-checkpoint-fencing-proof-summary.json" {
		t.Fatalf("unexpected checkpoint_fencing_summary: %v", proofArtifacts["checkpoint_fencing_summary"])
	}
	if got, ok := proofArtifacts["retention_boundary_summary"].(string); !ok || got != "bigclaw-go/docs/reports/broker-retention-boundary-proof-summary.json" {
		t.Fatalf("unexpected retention_boundary_summary: %v", proofArtifacts["retention_boundary_summary"])
	}
}

func TestCheckpointAndDurableSequenceAssertionsHoldForFencingScenarios(t *testing.T) {
	scriptPath := filepath.Join(testScriptDir(t), "broker_failover_stub_matrix.py")
	code := `
import importlib.util
import json
import pathlib

path = pathlib.Path(r"""` + scriptPath + `""")
spec = importlib.util.spec_from_file_location("broker_failover_stub_matrix", path)
if spec is None or spec.loader is None:
    raise RuntimeError(f"Unable to load module from {path}")
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
report = module.build_report()
scenarios = {item["scenario_id"]: item for item in report["scenarios"]}
bf04 = scenarios["BF-04"]
bf08 = scenarios["BF-08"]
transition_log = report["raw_artifacts"]["BF-04"]["checkpoint_transition_log"]
has_consumer_a_fenced = any(
    entry.get("owner_id") == "consumer-a" and entry.get("transition") == "fenced"
    for entry in transition_log
)
print(json.dumps({
    "bf04_after_gte_before": bf04["checkpoint_after_recovery"]["durable_sequence"] >= bf04["checkpoint_before_fault"]["durable_sequence"],
    "bf04_has_consumer_a_fenced": has_consumer_a_fenced,
    "bf08_duplicate_count": bf08["duplicate_count"],
    "bf08_missing_event_ids": bf08["missing_event_ids"],
}))
`
	payload := runPythonJSON(t, code)
	if !requireBool(t, payload, "bf04_after_gte_before") {
		t.Fatal("expected BF-04 checkpoint_after_recovery to be >= checkpoint_before_fault")
	}
	if !requireBool(t, payload, "bf04_has_consumer_a_fenced") {
		t.Fatal("expected BF-04 checkpoint transition log to include consumer-a fenced transition")
	}
	if got := requireNumber(t, payload, "bf08_duplicate_count"); got != 1 {
		t.Fatalf("unexpected BF-08 duplicate_count: %v", got)
	}
	missing, ok := payload["bf08_missing_event_ids"].([]any)
	if !ok {
		t.Fatalf("expected bf08_missing_event_ids array, got %T", payload["bf08_missing_event_ids"])
	}
	if len(missing) != 0 {
		t.Fatalf("expected no BF-08 missing_event_ids, got %v", missing)
	}
}

func TestProofSummariesProjectScenarioEvidenceIntoRolloutGateStatuses(t *testing.T) {
	scriptPath := filepath.Join(testScriptDir(t), "broker_failover_stub_matrix.py")
	code := `
import importlib.util
import json
import pathlib

path = pathlib.Path(r"""` + scriptPath + `""")
spec = importlib.util.spec_from_file_location("broker_failover_stub_matrix", path)
if spec is None or spec.loader is None:
    raise RuntimeError(f"Unable to load module from {path}")
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
report = module.build_report()
checkpoint = module.build_checkpoint_fencing_summary(report)
retention = module.build_retention_boundary_summary(report)
checkpoint_gates = {gate["name"]: gate for gate in checkpoint["rollout_gate_statuses"]}
retention_gates = {gate["name"]: gate for gate in retention["rollout_gate_statuses"]}
print(json.dumps({
    "checkpoint_ticket": checkpoint["ticket"],
    "checkpoint_proof_family": checkpoint["proof_family"],
    "checkpoint_replay_alignment": checkpoint_gates["replay_checkpoint_alignment"]["status"],
    "checkpoint_retention_visibility": checkpoint_gates["retention_boundary_visibility"]["status"],
    "checkpoint_stale_write_rejections": checkpoint["summary"]["stale_write_rejections"],
    "retention_ticket": retention["ticket"],
    "retention_proof_family": retention["proof_family"],
    "retention_visibility": retention_gates["retention_boundary_visibility"]["status"],
    "retention_focus_reset_required": retention["focus_scenarios"][0]["reset_required"],
    "retention_floor": retention["summary"]["retention_floor"],
}))
`
	payload := runPythonJSON(t, code)
	if got := requireString(t, payload, "checkpoint_ticket"); got != "OPE-230" {
		t.Fatalf("unexpected checkpoint ticket: %s", got)
	}
	if got := requireString(t, payload, "checkpoint_proof_family"); got != "checkpoint_fencing" {
		t.Fatalf("unexpected checkpoint proof family: %s", got)
	}
	if got := requireString(t, payload, "checkpoint_replay_alignment"); got != "passed" {
		t.Fatalf("unexpected checkpoint replay alignment gate status: %s", got)
	}
	if got := requireString(t, payload, "checkpoint_retention_visibility"); got != "unknown" {
		t.Fatalf("unexpected checkpoint retention visibility gate status: %s", got)
	}
	if got := requireNumber(t, payload, "checkpoint_stale_write_rejections"); got != 1 {
		t.Fatalf("unexpected checkpoint stale_write_rejections: %v", got)
	}
	if got := requireString(t, payload, "retention_ticket"); got != "OPE-230" {
		t.Fatalf("unexpected retention ticket: %s", got)
	}
	if got := requireString(t, payload, "retention_proof_family"); got != "retention_boundary" {
		t.Fatalf("unexpected retention proof family: %s", got)
	}
	if got := requireString(t, payload, "retention_visibility"); got != "passed" {
		t.Fatalf("unexpected retention visibility gate status: %s", got)
	}
	if !requireBool(t, payload, "retention_focus_reset_required") {
		t.Fatal("expected retention focus scenario to require reset")
	}
	if got := requireNumber(t, payload, "retention_floor"); got != 3 {
		t.Fatalf("unexpected retention floor: %v", got)
	}
}
