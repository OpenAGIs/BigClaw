package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type subscriberTakeoverReportSurface struct {
	Ticket             string   `json:"ticket"`
	Title              string   `json:"title"`
	Status             string   `json:"status"`
	HarnessMode        string   `json:"harness_mode"`
	RequiredSections   []string `json:"required_report_sections"`
	ImplementationPath []string `json:"implementation_path"`
	CurrentPrimitives  struct {
		LeaseAwareCheckpoints []string `json:"lease_aware_checkpoints"`
		SharedQueueEvidence   []string `json:"shared_queue_evidence"`
		TakeoverHarness       []string `json:"takeover_harness"`
		LiveTakeoverHarness   []string `json:"live_takeover_harness"`
	} `json:"current_primitives"`
	Summary struct {
		ScenarioCount          int `json:"scenario_count"`
		PassingScenarios       int `json:"passing_scenarios"`
		FailingScenarios       int `json:"failing_scenarios"`
		DuplicateDeliveryCount int `json:"duplicate_delivery_count"`
		StaleWriteRejections   int `json:"stale_write_rejections"`
	} `json:"summary"`
	Scenarios []subscriberTakeoverScenarioSurface `json:"scenarios"`
}

type subscriberTakeoverScenarioSurface struct {
	ID                     string   `json:"id"`
	Title                  string   `json:"title"`
	PrimarySubscriber      string   `json:"primary_subscriber"`
	TakeoverSubscriber     string   `json:"takeover_subscriber"`
	AuditLogPaths          []string `json:"audit_log_paths"`
	DuplicateEvents        []string `json:"duplicate_events"`
	DuplicateDeliveryCount int      `json:"duplicate_delivery_count"`
	StaleWriteRejections   int      `json:"stale_write_rejections"`
	AllAssertionsPassed    bool     `json:"all_assertions_passed"`
	LocalLimitations       []string `json:"local_limitations"`
	CheckpointBefore       struct {
		Owner   string `json:"owner"`
		Offset  int    `json:"offset"`
		EventID string `json:"event_id"`
	} `json:"checkpoint_before"`
	CheckpointAfter struct {
		Owner   string `json:"owner"`
		Offset  int    `json:"offset"`
		EventID string `json:"event_id"`
	} `json:"checkpoint_after"`
	ReplayStartCursor struct {
		Offset  int    `json:"offset"`
		EventID string `json:"event_id"`
	} `json:"replay_start_cursor"`
	ReplayEndCursor struct {
		Offset  int    `json:"offset"`
		EventID string `json:"event_id"`
	} `json:"replay_end_cursor"`
	AuditTimeline []struct {
		Action string `json:"action"`
	} `json:"audit_timeline"`
	EventLogExcerpt []struct {
		DeliveryKind string `json:"delivery_kind"`
	} `json:"event_log_excerpt"`
	AssertionResults struct {
		Audit      []subscriberTakeoverAssertion `json:"audit"`
		Checkpoint []subscriberTakeoverAssertion `json:"checkpoint"`
		Replay     []subscriberTakeoverAssertion `json:"replay"`
	} `json:"assertion_results"`
}

type subscriberTakeoverAssertion struct {
	Label  string `json:"label"`
	Passed bool   `json:"passed"`
}

func TestSubscriberTakeoverReportContractsStayAligned(t *testing.T) {
	root := repoRoot(t)

	var localReport subscriberTakeoverReportSurface
	readJSONFile(t, filepath.Join(root, "docs", "reports", "multi-subscriber-takeover-validation-report.json"), &localReport)

	var liveReport subscriberTakeoverReportSurface
	readJSONFile(t, filepath.Join(root, "docs", "reports", "live-multi-node-subscriber-takeover-report.json"), &liveReport)

	validateSubscriberTakeoverReport(t, root, localReport, subscriberTakeoverExpectations{
		ticket:              "OPE-269",
		status:              "local-executable",
		harnessMode:         "deterministic_local_simulation",
		takeoverHarnessPath: "scripts/e2e/subscriber_takeover_fault_matrix.py",
		summaryDuplicates:   4,
		summaryStaleWrites:  2,
		expectLiveArtifacts: false,
	})
	validateSubscriberTakeoverReport(t, root, liveReport, subscriberTakeoverExpectations{
		ticket:              "OPE-260",
		status:              "live-multi-node-proof",
		harnessMode:         "live_multi_node_bigclawd_cluster",
		takeoverHarnessPath: "scripts/e2e/multi_node_shared_queue.py",
		summaryDuplicates:   4,
		summaryStaleWrites:  3,
		expectLiveArtifacts: true,
	})

	if len(localReport.RequiredSections) != len(liveReport.RequiredSections) ||
		len(localReport.ImplementationPath) != len(liveReport.ImplementationPath) ||
		localReport.Summary.ScenarioCount != liveReport.Summary.ScenarioCount ||
		localReport.Summary.DuplicateDeliveryCount != liveReport.Summary.DuplicateDeliveryCount {
		t.Fatalf("local/live takeover report contracts drifted: local=%+v live=%+v", localReport.Summary, liveReport.Summary)
	}
	if localReport.Summary.StaleWriteRejections >= liveReport.Summary.StaleWriteRejections {
		t.Fatalf("expected live stale-write count to exceed local harness count: local=%d live=%d", localReport.Summary.StaleWriteRejections, liveReport.Summary.StaleWriteRejections)
	}
}

func TestSubscriberTakeoverFollowUpDigestLinksReports(t *testing.T) {
	root := repoRoot(t)
	contents := readRepoFile(t, root, "docs/reports/subscriber-takeover-executability-follow-up-digest.md")

	for _, needle := range []string{
		"multi-subscriber-takeover-validation-report.json",
		"live-multi-node-subscriber-takeover-report.json",
		"scripts/e2e/subscriber_takeover_fault_matrix.py",
		"scripts/e2e/multi_node_shared_queue.py",
		"broker-backed or replicated backend exists",
	} {
		if !strings.Contains(contents, needle) {
			t.Fatalf("subscriber takeover follow-up digest missing %q", needle)
		}
	}
}

type subscriberTakeoverExpectations struct {
	ticket              string
	status              string
	harnessMode         string
	takeoverHarnessPath string
	summaryDuplicates   int
	summaryStaleWrites  int
	expectLiveArtifacts bool
}

func validateSubscriberTakeoverReport(t *testing.T, root string, report subscriberTakeoverReportSurface, want subscriberTakeoverExpectations) {
	t.Helper()

	if report.Ticket != want.ticket || report.Status != want.status || report.HarnessMode != want.harnessMode {
		t.Fatalf("unexpected subscriber takeover report identity: %+v", report)
	}
	if len(report.RequiredSections) != 9 || len(report.ImplementationPath) != 4 {
		t.Fatalf("unexpected takeover report structure: sections=%d path=%d", len(report.RequiredSections), len(report.ImplementationPath))
	}

	requiredEvidence := append([]string{}, report.CurrentPrimitives.LeaseAwareCheckpoints...)
	requiredEvidence = append(requiredEvidence, report.CurrentPrimitives.SharedQueueEvidence...)
	requiredEvidence = append(requiredEvidence, want.takeoverHarnessPath)
	for _, candidate := range requiredEvidence {
		if _, err := os.Stat(resolveRepoPath(root, candidate)); err != nil {
			t.Fatalf("expected takeover evidence path %q to exist: %v", candidate, err)
		}
	}

	if report.Summary.ScenarioCount != 3 ||
		report.Summary.PassingScenarios != 3 ||
		report.Summary.FailingScenarios != 0 ||
		report.Summary.DuplicateDeliveryCount != want.summaryDuplicates ||
		report.Summary.StaleWriteRejections != want.summaryStaleWrites ||
		len(report.Scenarios) != report.Summary.ScenarioCount {
		t.Fatalf("unexpected takeover report summary: %+v", report.Summary)
	}

	totalDuplicates := 0
	totalStaleWrites := 0
	for _, scenario := range report.Scenarios {
		validateSubscriberTakeoverScenario(t, root, scenario, want.expectLiveArtifacts)
		totalDuplicates += scenario.DuplicateDeliveryCount
		totalStaleWrites += scenario.StaleWriteRejections
	}

	if totalDuplicates != report.Summary.DuplicateDeliveryCount || totalStaleWrites != report.Summary.StaleWriteRejections {
		t.Fatalf("takeover report summary drifted from scenario totals: summary=%+v duplicates=%d stale=%d", report.Summary, totalDuplicates, totalStaleWrites)
	}
}

func validateSubscriberTakeoverScenario(t *testing.T, root string, scenario subscriberTakeoverScenarioSurface, expectLiveArtifacts bool) {
	t.Helper()

	if scenario.ID == "" || scenario.Title == "" || scenario.PrimarySubscriber == "" || scenario.TakeoverSubscriber == "" {
		t.Fatalf("unexpected empty takeover scenario identity: %+v", scenario)
	}
	if scenario.PrimarySubscriber == scenario.TakeoverSubscriber {
		t.Fatalf("expected distinct takeover participants, got %+v", scenario)
	}
	if !scenario.AllAssertionsPassed {
		t.Fatalf("expected all takeover assertions to pass for %+v", scenario)
	}
	if scenario.CheckpointBefore.Owner != scenario.PrimarySubscriber || scenario.CheckpointAfter.Owner != scenario.TakeoverSubscriber {
		t.Fatalf("unexpected checkpoint ownership transfer in %+v", scenario)
	}
	if scenario.CheckpointAfter.Offset < scenario.CheckpointBefore.Offset ||
		scenario.ReplayStartCursor.Offset != scenario.CheckpointBefore.Offset ||
		scenario.ReplayStartCursor.EventID != scenario.CheckpointBefore.EventID ||
		scenario.ReplayEndCursor.Offset != scenario.CheckpointAfter.Offset ||
		scenario.ReplayEndCursor.EventID != scenario.CheckpointAfter.EventID {
		t.Fatalf("unexpected replay or checkpoint boundaries in %+v", scenario)
	}
	if len(scenario.AuditLogPaths) != 2 || len(scenario.DuplicateEvents) != scenario.DuplicateDeliveryCount || len(scenario.LocalLimitations) < 2 {
		t.Fatalf("unexpected takeover scenario artifact payload: %+v", scenario)
	}
	if len(scenario.AuditTimeline) < 6 || len(scenario.EventLogExcerpt) < 2 {
		t.Fatalf("unexpected takeover scenario timeline payload: %+v", scenario)
	}
	if !scenarioAssertionsPassed(scenario.AssertionResults.Audit) ||
		!scenarioAssertionsPassed(scenario.AssertionResults.Checkpoint) ||
		!scenarioAssertionsPassed(scenario.AssertionResults.Replay) {
		t.Fatalf("expected all scenario assertion categories to pass for %+v", scenario)
	}
	if !containsScenarioAction(scenario.AuditTimeline, "lease_acquired") ||
		!containsScenarioAction(scenario.AuditTimeline, "checkpoint_committed") {
		t.Fatalf("expected lease acquisition and checkpoint commits in %+v", scenario.AuditTimeline)
	}

	if expectLiveArtifacts {
		for _, candidate := range scenario.AuditLogPaths {
			if _, err := os.Stat(resolveRepoPath(root, candidate)); err != nil {
				t.Fatalf("expected live takeover audit artifact %q to exist: %v", candidate, err)
			}
		}
		if !containsScenarioAction(scenario.AuditTimeline, "takeover_succeeded") ||
			!containsScenarioAction(scenario.AuditTimeline, "lease_fenced") ||
			!containsScenarioDeliveryKind(scenario.EventLogExcerpt, "task.completed") {
			t.Fatalf("expected live takeover runtime markers in %+v", scenario)
		}
		return
	}

	if !strings.HasPrefix(scenario.AuditLogPaths[0], "artifacts/") ||
		!containsScenarioAction(scenario.AuditTimeline, "replay_started") {
		t.Fatalf("expected local takeover harness markers in %+v", scenario)
	}
}

func scenarioAssertionsPassed(results []subscriberTakeoverAssertion) bool {
	if len(results) == 0 {
		return false
	}
	for _, result := range results {
		if !result.Passed {
			return false
		}
	}
	return true
}

func containsScenarioAction(actions []struct {
	Action string `json:"action"`
}, want string) bool {
	for _, action := range actions {
		if action.Action == want {
			return true
		}
	}
	return false
}

func containsScenarioDeliveryKind(events []struct {
	DeliveryKind string `json:"delivery_kind"`
}, want string) bool {
	for _, event := range events {
		if event.DeliveryKind == want {
			return true
		}
	}
	return false
}
