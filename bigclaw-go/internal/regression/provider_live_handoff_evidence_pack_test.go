package regression

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestProviderLiveHandoffIsolationEvidencePackStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	reportPath := filepath.Join(repoRoot, "docs", "reports", "provider-live-handoff-isolation-evidence-pack.json")

	var report struct {
		Ticket         string   `json:"ticket"`
		Track          string   `json:"track"`
		Title          string   `json:"title"`
		Status         string   `json:"status"`
		Backend        string   `json:"backend"`
		ValidationLane string   `json:"validation_lane"`
		ReportPath     string   `json:"report_path"`
		EvidenceSources []string `json:"evidence_sources"`
		ReviewerLinks  []string `json:"reviewer_links"`
		Summary        struct {
			ScenarioCount          int  `json:"scenario_count"`
			IsolatedScenarios      int  `json:"isolated_scenarios"`
			StalledScenarios       int  `json:"stalled_scenarios"`
			ReplayBacklogEvents    int  `json:"replay_backlog_events"`
			ReplayStepDelayMS      int  `json:"replay_step_delay_ms"`
			ReplayWindowMS         int  `json:"replay_window_ms"`
			LiveDeliveryDeadlineMS int  `json:"live_delivery_deadline_ms"`
			IsolationMaintained    bool `json:"isolation_maintained"`
		} `json:"summary"`
		Scenarios []struct {
			Name                   string   `json:"name"`
			Status                 string   `json:"status"`
			ReplayBacklogEvents    int      `json:"replay_backlog_events"`
			ReplayStepDelayMS      int      `json:"replay_step_delay_ms"`
			ReplayWindowMS         int      `json:"replay_window_ms"`
			LiveDeliveryDeadlineMS int      `json:"live_delivery_deadline_ms"`
			ReplayDrainsAfterLive  bool     `json:"replay_drains_after_live"`
			SourceTests            []string `json:"source_tests"`
			Notes                  []string `json:"notes"`
		} `json:"scenarios"`
		Limitations []string `json:"limitations"`
	}
	readJSONFile(t, reportPath, &report)

	if report.Ticket != "OPE-225" || report.Track != "BIG-DUR-104" || report.Title != "Provider-backed live handoff isolation evidence pack" || report.Status != "checked_in_evidence_pack" {
		t.Fatalf("unexpected provider handoff report identity: %+v", report)
	}
	if report.Backend != "http_remote_service" || report.ValidationLane != "external_store_validation" || report.ReportPath != "docs/reports/provider-live-handoff-isolation-evidence-pack.json" {
		t.Fatalf("unexpected provider handoff backend posture: %+v", report)
	}
	if len(report.EvidenceSources) != 4 || len(report.ReviewerLinks) != 4 {
		t.Fatalf("unexpected provider handoff references: evidence=%+v reviewer=%+v", report.EvidenceSources, report.ReviewerLinks)
	}
	for _, candidate := range append(append([]string{}, report.EvidenceSources...), report.ReviewerLinks...) {
		if candidate == "" {
			t.Fatal("provider handoff evidence pack contains an empty reference")
		}
		if _, err := filepath.Abs(filepath.Join(repoRoot, strings.TrimPrefix(candidate, "bigclaw-go/"))); err != nil {
			t.Fatalf("resolve provider handoff reference %q: %v", candidate, err)
		}
	}
	if report.Summary.ScenarioCount != 1 || report.Summary.IsolatedScenarios != 1 || report.Summary.StalledScenarios != 0 || report.Summary.ReplayBacklogEvents != 4 || report.Summary.ReplayStepDelayMS != 0 || report.Summary.ReplayWindowMS != 0 || report.Summary.LiveDeliveryDeadlineMS != 200 || !report.Summary.IsolationMaintained {
		t.Fatalf("unexpected provider handoff summary: %+v", report.Summary)
	}
	if len(report.Scenarios) != 1 {
		t.Fatalf("expected 1 provider handoff scenario row, got %+v", report.Scenarios)
	}
	if report.Scenarios[0].Name != "http_remote_service_replay_handoff_keeps_live_lane_unblocked" || report.Scenarios[0].Status != "isolated" || report.Scenarios[0].ReplayBacklogEvents != 4 || report.Scenarios[0].ReplayStepDelayMS != 0 || report.Scenarios[0].ReplayWindowMS != 0 || report.Scenarios[0].LiveDeliveryDeadlineMS != 200 || !report.Scenarios[0].ReplayDrainsAfterLive {
		t.Fatalf("unexpected provider handoff scenario: %+v", report.Scenarios[0])
	}
	if len(report.Scenarios[0].SourceTests) != 1 || report.Scenarios[0].SourceTests[0] != "internal/api/server_test.go" || len(report.Scenarios[0].Notes) != 3 {
		t.Fatalf("unexpected provider handoff scenario evidence: %+v", report.Scenarios[0])
	}
	if len(report.Limitations) != 2 {
		t.Fatalf("unexpected provider handoff limitations: %+v", report.Limitations)
	}

	for _, tc := range []struct {
		path       string
		substrings []string
	}{
		{
			path: "docs/e2e-validation.md",
			substrings: []string{
				"provider-live-handoff-isolation-evidence-pack.json",
				"no-stall contract",
				"http_remote_service",
			},
		},
		{
			path: "docs/reports/replicated-event-log-durability-rollout-contract.md",
			substrings: []string{
				"provider-live-handoff-isolation-evidence-pack.json",
				"http_remote_service",
				"native broker-backed adapter",
			},
		},
		{
			path: "docs/reports/review-readiness.md",
			substrings: []string{
				"provider-live-handoff-isolation-evidence-pack.json",
				"runtime diagnostics",
				"distributed exports",
			},
		},
	} {
		contents := readRepoFile(t, repoRoot, tc.path)
		for _, needle := range tc.substrings {
			if !strings.Contains(contents, needle) {
				t.Fatalf("%s missing substring %q", tc.path, needle)
			}
		}
	}
}
