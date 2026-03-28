package regression

import (
	"encoding/json"
	"strings"
	"testing"

	"bigclaw-go/internal/testharness"
)

func TestCoordinationSurfaceSummarizesCurrentLiveAndLocalProofs(t *testing.T) {
	report := runCoordinationSurfaceBuildReport(t)

	if report.Status != "local-capability-surface" {
		t.Fatalf("unexpected coordination surface status: %+v", report)
	}
	if !strings.HasPrefix(report.RuntimeReadinessLevels.LiveProven, "Shipped runtime behavior") {
		t.Fatalf("unexpected live-proven readiness description: %+v", report.RuntimeReadinessLevels)
	}
	if report.Summary.SharedQueueCrossNodeCompletions != 99 || report.Summary.TakeoverPassingScenarios != 3 {
		t.Fatalf("unexpected coordination summary: %+v", report.Summary)
	}
	found := false
	for _, item := range report.CurrentCeiling {
		if strings.Contains(item, "no partitioned topic model") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected current ceiling to mention missing partitioned topic model: %+v", report.CurrentCeiling)
	}
}

func TestCoordinationSurfaceMarksPartitionedAndBrokerModelsUnavailable(t *testing.T) {
	report := runCoordinationSurfaceBuildReport(t)

	byCapability := map[string]coordinationCapability{}
	for _, item := range report.Capabilities {
		byCapability[item.Capability] = item
	}

	if byCapability["partitioned_topic_routing"].CurrentState != "not_available" ||
		byCapability["partitioned_topic_routing"].RuntimeReadiness != "contract_only" {
		t.Fatalf("unexpected partitioned topic routing capability: %+v", byCapability["partitioned_topic_routing"])
	}
	if byCapability["broker_backed_subscriber_ownership"].CurrentState != "not_available" ||
		byCapability["broker_backed_subscriber_ownership"].RuntimeReadiness != "contract_only" {
		t.Fatalf("unexpected broker-backed ownership capability: %+v", byCapability["broker_backed_subscriber_ownership"])
	}
	if byCapability["shared_queue_task_coordination"].RuntimeReadiness != "live_proven" {
		t.Fatalf("unexpected shared queue capability: %+v", byCapability["shared_queue_task_coordination"])
	}
	if !byCapability["subscriber_takeover_semantics"].DeterministicLocalHarness ||
		byCapability["subscriber_takeover_semantics"].RuntimeReadiness != "live_proven" {
		t.Fatalf("unexpected takeover capability: %+v", byCapability["subscriber_takeover_semantics"])
	}
}

type coordinationSurfaceRuntimeReport struct {
	Status                 string `json:"status"`
	RuntimeReadinessLevels struct {
		LiveProven string `json:"live_proven"`
	} `json:"runtime_readiness_levels"`
	Summary struct {
		SharedQueueCrossNodeCompletions int `json:"shared_queue_cross_node_completions"`
		TakeoverPassingScenarios        int `json:"takeover_passing_scenarios"`
	} `json:"summary"`
	Capabilities   []coordinationCapability `json:"capabilities"`
	CurrentCeiling []string                 `json:"current_ceiling"`
}

type coordinationCapability struct {
	Capability                string `json:"capability"`
	CurrentState              string `json:"current_state"`
	RuntimeReadiness          string `json:"runtime_readiness"`
	DeterministicLocalHarness bool   `json:"deterministic_local_harness"`
}

func runCoordinationSurfaceBuildReport(t *testing.T) coordinationSurfaceRuntimeReport {
	t.Helper()
	scriptPath := testharness.JoinRepoRoot(t, "scripts", "e2e", "cross_process_coordination_surface.py")
	pythonSnippet := strings.Join([]string{
		"import importlib.util, json",
		"spec = importlib.util.spec_from_file_location('cross_process_coordination_surface', r'" + scriptPath + "')",
		"module = importlib.util.module_from_spec(spec)",
		"assert spec.loader is not None",
		"spec.loader.exec_module(module)",
		"print(json.dumps(module.build_report()))",
	}, "\n")

	cmd := testharness.PythonCommand(t, "-c", pythonSnippet)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build coordination surface report: %v (%s)", err, string(output))
	}

	var report coordinationSurfaceRuntimeReport
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("decode coordination surface report: %v (%s)", err, string(output))
	}
	return report
}
