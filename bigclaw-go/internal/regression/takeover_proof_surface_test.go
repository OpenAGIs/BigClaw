package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type takeoverReportSummary struct {
	ScenarioCount          int `json:"scenario_count"`
	PassingScenarios       int `json:"passing_scenarios"`
	FailingScenarios       int `json:"failing_scenarios"`
	DuplicateDeliveryCount int `json:"duplicate_delivery_count"`
	StaleWriteRejections   int `json:"stale_write_rejections"`
}

type takeoverReportScenario struct {
	ID                     string   `json:"id"`
	Title                  string   `json:"title"`
	AuditLogPaths          []string `json:"audit_log_paths"`
	DuplicateDeliveryCount int      `json:"duplicate_delivery_count"`
	StaleWriteRejections   int      `json:"stale_write_rejections"`
}

type localTakeoverReport struct {
	Ticket            string `json:"ticket"`
	Status            string `json:"status"`
	HarnessMode       string `json:"harness_mode"`
	CurrentPrimitives struct {
		LeaseAwareCheckpoints []string `json:"lease_aware_checkpoints"`
		SharedQueueEvidence   []string `json:"shared_queue_evidence"`
		TakeoverHarness       []string `json:"takeover_harness"`
	} `json:"current_primitives"`
	Summary   takeoverReportSummary    `json:"summary"`
	Scenarios []takeoverReportScenario `json:"scenarios"`
}

type liveTakeoverReport struct {
	Ticket            string `json:"ticket"`
	Status            string `json:"status"`
	HarnessMode       string `json:"harness_mode"`
	CurrentPrimitives struct {
		LeaseAwareCheckpoints []string `json:"lease_aware_checkpoints"`
		SharedQueueEvidence   []string `json:"shared_queue_evidence"`
		LiveTakeoverHarness   []string `json:"live_takeover_harness"`
	} `json:"current_primitives"`
	Summary struct {
		ScenarioCount          int `json:"scenario_count"`
		PassingScenarios       int `json:"passing_scenarios"`
		FailingScenarios       int `json:"failing_scenarios"`
		DuplicateDeliveryCount int `json:"duplicate_delivery_count"`
		StaleWriteRejections   int `json:"stale_write_rejections"`
	} `json:"summary"`
	Scenarios []takeoverReportScenario `json:"scenarios"`
}

func TestLocalTakeoverReportStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	var report localTakeoverReport
	readJSONFile(t, filepath.Join(repoRoot, "docs", "reports", "multi-subscriber-takeover-validation-report.json"), &report)

	if report.Ticket != "OPE-269" || report.Status != "local-executable" || report.HarnessMode != "deterministic_local_simulation" {
		t.Fatalf("unexpected local takeover metadata: %+v", report)
	}
	if report.Summary.ScenarioCount != len(report.Scenarios) || report.Summary.PassingScenarios != 3 || report.Summary.DuplicateDeliveryCount != 4 || report.Summary.StaleWriteRejections != 2 {
		t.Fatalf("unexpected local takeover summary: %+v", report.Summary)
	}

	for _, path := range append(append(report.CurrentPrimitives.LeaseAwareCheckpoints, report.CurrentPrimitives.SharedQueueEvidence...), report.CurrentPrimitives.TakeoverHarness...) {
		if err := assertRepoPathExists(repoRoot, path); err != nil {
			t.Fatalf("current primitive missing %q: %v", path, err)
		}
	}

	for _, scenario := range report.Scenarios {
		if scenario.ID == "" || len(scenario.AuditLogPaths) == 0 {
			t.Fatalf("scenario lacks identifiers or audit artifacts: %+v", scenario)
		}
		for _, auditPath := range scenario.AuditLogPaths {
			if !strings.HasPrefix(auditPath, "artifacts/") {
				t.Fatalf("expected local harness audit path to stay report-local, got %q", auditPath)
			}
		}
		if scenario.DuplicateDeliveryCount < 0 || scenario.StaleWriteRejections < 0 {
			t.Fatalf("unexpected counts in scenario %s: %+v", scenario.ID, scenario)
		}
	}
}

func TestLiveTakeoverReportStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	var report liveTakeoverReport
	readJSONFile(t, filepath.Join(repoRoot, "docs", "reports", "live-multi-node-subscriber-takeover-report.json"), &report)

	if report.Ticket != "OPE-260" || report.Status != "live-multi-node-proof" {
		t.Fatalf("unexpected live takeover metadata: %+v", report)
	}
	if report.Summary.ScenarioCount != len(report.Scenarios) || report.Summary.PassingScenarios != 3 || report.Summary.DuplicateDeliveryCount != 4 || report.Summary.StaleWriteRejections != 3 {
		t.Fatalf("unexpected live takeover summary: %+v", report.Summary)
	}

	for _, path := range append(append(report.CurrentPrimitives.LeaseAwareCheckpoints, report.CurrentPrimitives.SharedQueueEvidence...), report.CurrentPrimitives.LiveTakeoverHarness...) {
		if err := assertRepoPathExists(repoRoot, path); err != nil {
			t.Fatalf("live primitive missing %q: %v", path, err)
		}
	}

	for _, scenario := range report.Scenarios {
		if len(scenario.AuditLogPaths) == 0 {
			t.Fatalf("live scenario %s missing audit logs", scenario.ID)
		}
		for _, auditPath := range scenario.AuditLogPaths {
			if err := assertRepoPathExists(repoRoot, auditPath); err != nil {
				t.Fatalf("audit artifact missing %q for live scenario %s: %v", auditPath, scenario.ID, err)
			}
		}
		if scenario.DuplicateDeliveryCount < 0 || scenario.StaleWriteRejections < 0 {
			t.Fatalf("unexpected counts in live scenario %s: %+v", scenario.ID, scenario)
		}
	}
}

func TestTakeoverFollowUpDigestReferences(t *testing.T) {
	repoRoot := repoRoot(t)
	digest := readRepoFile(t, repoRoot, "docs/reports/subscriber-takeover-executability-follow-up-digest.md")
	for _, needle := range []string{
		"multi-subscriber-takeover-validation-report.json",
		"live-multi-node-subscriber-takeover-report.json",
		"shared durable SQLite scaffold exists but broker-backed ownership does not",
	} {
		if !strings.Contains(digest, needle) {
			t.Fatalf("follow-up digest missing %q", needle)
		}
	}
}

func assertRepoPathExists(repoRoot, candidate string) error {
	location := resolveRepoPath(repoRoot, candidate)
	_, err := os.Stat(location)
	return err
}
