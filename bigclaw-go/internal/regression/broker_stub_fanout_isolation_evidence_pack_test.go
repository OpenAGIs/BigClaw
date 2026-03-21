package regression

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBrokerStubFanoutIsolationEvidencePackStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	reportPath := filepath.Join(repoRoot, "docs", "reports", "broker-stub-live-fanout-isolation-evidence-pack.json")

	var report struct {
		Ticket  string `json:"ticket"`
		Status  string `json:"status"`
		Backend string `json:"backend"`
		Summary struct {
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
			Name                   string `json:"name"`
			Status                 string `json:"status"`
			ReplayBacklogEvents    int    `json:"replay_backlog_events"`
			ReplayStepDelayMS      int    `json:"replay_step_delay_ms"`
			ReplayWindowMS         int    `json:"replay_window_ms"`
			LiveDeliveryDeadlineMS int    `json:"live_delivery_deadline_ms"`
			ReplayDrainsAfterLive  bool   `json:"replay_drains_after_live"`
		} `json:"scenarios"`
	}
	readJSONFile(t, reportPath, &report)
	if report.Ticket != "OPE-261" || report.Status != "checked_in_evidence_pack" || report.Backend != "broker_stub" {
		t.Fatalf("unexpected broker stub fanout report metadata: %+v", report)
	}
	if report.Summary.ScenarioCount != 1 || report.Summary.IsolatedScenarios != 1 || report.Summary.StalledScenarios != 0 || report.Summary.ReplayBacklogEvents != 4 || report.Summary.ReplayStepDelayMS != 30 || report.Summary.ReplayWindowMS != 120 || report.Summary.LiveDeliveryDeadlineMS != 50 || !report.Summary.IsolationMaintained {
		t.Fatalf("unexpected broker stub fanout summary: %+v", report.Summary)
	}
	if len(report.Scenarios) != 1 {
		t.Fatalf("expected 1 scenario row, got %+v", report.Scenarios)
	}
	if report.Scenarios[0].Name != "replay_catchup_does_not_block_live_publish" || report.Scenarios[0].Status != "isolated" || report.Scenarios[0].ReplayBacklogEvents != 4 || report.Scenarios[0].ReplayStepDelayMS != 30 || report.Scenarios[0].ReplayWindowMS != 120 || report.Scenarios[0].LiveDeliveryDeadlineMS != 50 || !report.Scenarios[0].ReplayDrainsAfterLive {
		t.Fatalf("unexpected broker stub fanout scenario: %+v", report.Scenarios[0])
	}

	contents := readRepoFile(t, repoRoot, "docs/reports/event-bus-reliability-report.md")
	for _, needle := range []string{"broker-stub-live-fanout-isolation-evidence-pack.json", "live fanout isolation for the local broker stub"} {
		if !strings.Contains(contents, needle) {
			t.Fatalf("event-bus reliability report missing substring %q", needle)
		}
	}
}
