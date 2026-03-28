package regression

import (
	"encoding/json"
	"testing"

	"bigclaw-go/internal/testharness"
)

func TestTakeoverHarnessReportHasThreePassingScenarios(t *testing.T) {
	report := runSubscriberTakeoverFaultMatrix(t)

	if report.Status != "local-executable" {
		t.Fatalf("unexpected takeover harness status: %+v", report)
	}
	if report.Summary.ScenarioCount != 3 ||
		report.Summary.PassingScenarios != 3 ||
		report.Summary.FailingScenarios != 0 ||
		report.Summary.StaleWriteRejections != 2 ||
		report.Summary.DuplicateDeliveryCount != 4 {
		t.Fatalf("unexpected takeover harness summary: %+v", report.Summary)
	}
}

func TestStaleWriterScenarioRecordsRejectionAndFinalOwner(t *testing.T) {
	report := runSubscriberTakeoverFaultMatrix(t)

	var scenario takeoverScenario
	found := false
	for _, item := range report.Scenarios {
		if item.ID == "lease-expiry-stale-writer-rejected" {
			scenario = item
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected stale-writer scenario in %+v", report.Scenarios)
	}

	if scenario.StaleWriteRejections != 1 ||
		scenario.CheckpointAfter.Owner != scenario.TakeoverSubscriber ||
		!containsTakeoverString(scenario.DuplicateEvents, "evt-81") ||
		!scenario.AllAssertionsPassed {
		t.Fatalf("unexpected stale-writer scenario: %+v", scenario)
	}
}

type subscriberTakeoverRuntimeReport struct {
	Status  string `json:"status"`
	Summary struct {
		ScenarioCount          int `json:"scenario_count"`
		PassingScenarios       int `json:"passing_scenarios"`
		FailingScenarios       int `json:"failing_scenarios"`
		StaleWriteRejections   int `json:"stale_write_rejections"`
		DuplicateDeliveryCount int `json:"duplicate_delivery_count"`
	} `json:"summary"`
	Scenarios []takeoverScenario `json:"scenarios"`
}

type takeoverScenario struct {
	ID                   string   `json:"id"`
	TakeoverSubscriber   string   `json:"takeover_subscriber"`
	DuplicateEvents      []string `json:"duplicate_events"`
	StaleWriteRejections int      `json:"stale_write_rejections"`
	AllAssertionsPassed  bool     `json:"all_assertions_passed"`
	CheckpointAfter      struct {
		Owner string `json:"owner"`
	} `json:"checkpoint_after"`
}

func runSubscriberTakeoverFaultMatrix(t *testing.T) subscriberTakeoverRuntimeReport {
	t.Helper()
	scriptPath := testharness.JoinRepoRoot(t, "scripts", "e2e", "subscriber_takeover_fault_matrix.py")
	pythonSnippet := "import importlib.util, json\n" +
		"spec = importlib.util.spec_from_file_location('subscriber_takeover_fault_matrix', r'" + scriptPath + "')\n" +
		"module = importlib.util.module_from_spec(spec)\n" +
		"assert spec.loader is not None\n" +
		"spec.loader.exec_module(module)\n" +
		"print(json.dumps(module.build_report()))\n"

	cmd := testharness.PythonCommand(t, "-c", pythonSnippet)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build takeover fault matrix report: %v (%s)", err, string(output))
	}

	var report subscriberTakeoverRuntimeReport
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("decode takeover fault matrix report: %v (%s)", err, string(output))
	}
	return report
}

func containsTakeoverString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
