package regression

import (
	"path/filepath"
	"strings"
	"testing"
)

type providerLiveHandoffSurface struct {
	Ticket          string   `json:"ticket"`
	Track           string   `json:"track"`
	Title           string   `json:"title"`
	Status          string   `json:"status"`
	Backend         string   `json:"backend"`
	ValidationLane  string   `json:"validation_lane"`
	ReportPath      string   `json:"report_path"`
	EvidenceSources []string `json:"evidence_sources"`
	ReviewerLinks   []string `json:"reviewer_links"`
	Summary         struct {
		ScenarioCount        int  `json:"scenario_count"`
		IsolatedScenarios    int  `json:"isolated_scenarios"`
		StalledScenarios     int  `json:"stalled_scenarios"`
		ReplayBacklogEvents  int  `json:"replay_backlog_events"`
		ReplayStepDelayMS    int  `json:"replay_step_delay_ms"`
		ReplayWindowMS       int  `json:"replay_window_ms"`
		LiveDeliveryDeadline int  `json:"live_delivery_deadline_ms"`
		IsolationMaintained  bool `json:"isolation_maintained"`
	} `json:"summary"`
	Scenarios []struct {
		Name                  string   `json:"name"`
		Status                string   `json:"status"`
		ReplayBacklogEvents   int      `json:"replay_backlog_events"`
		ReplayStepDelayMS     int      `json:"replay_step_delay_ms"`
		ReplayWindowMS        int      `json:"replay_window_ms"`
		LiveDeliveryDeadline  int      `json:"live_delivery_deadline_ms"`
		ReplayDrainsAfterLive bool     `json:"replay_drains_after_live"`
		SourceTests           []string `json:"source_tests"`
		Notes                 []string `json:"notes"`
	} `json:"scenarios"`
	Limitations []string `json:"limitations"`
}

func TestProviderLiveHandoffSurfaceStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	var surface providerLiveHandoffSurface
	readJSONFile(t, filepath.Join(repoRoot, "docs", "reports", "provider-live-handoff-isolation-evidence-pack.json"), &surface)

	if surface.Ticket != "OPE-225" || surface.Track != "BIG-DUR-104" || surface.Backend != "http_remote_service" {
		t.Fatalf("unexpected provider live handoff metadata: %+v", surface)
	}
	if surface.ValidationLane != "external_store_validation" || surface.Status != "checked_in_evidence_pack" {
		t.Fatalf("unexpected provider handoff lane posture: %+v", surface)
	}
	expectedSources := []string{
		"bigclaw-go/docs/reports/external-store-validation-report.json",
		"bigclaw-go/internal/events/http_log.go",
		"bigclaw-go/internal/api/server_test.go",
		"bigclaw-go/docs/reports/replicated-event-log-durability-rollout-contract.md",
	}
	if !matchesSubset(surface.EvidenceSources, expectedSources) {
		t.Fatalf("missing evidence sources: %+v", surface.EvidenceSources)
	}
	expectedLinks := []string{
		"docs/e2e-validation.md",
		"docs/reports/external-store-validation-report.json",
		"docs/reports/replicated-event-log-durability-rollout-contract.md",
		"docs/reports/review-readiness.md",
	}
	if !matchesSubset(surface.ReviewerLinks, expectedLinks) {
		t.Fatalf("unexpected reviewer links: %+v", surface.ReviewerLinks)
	}

	if surface.Summary.ScenarioCount != 1 || surface.Summary.IsolatedScenarios != 1 || surface.Summary.LiveDeliveryDeadline != 200 || !surface.Summary.IsolationMaintained {
		t.Fatalf("unexpected provider live handoff summary: %+v", surface.Summary)
	}
	if surface.Summary.ReplayBacklogEvents != 4 || surface.Summary.ReplayStepDelayMS != 0 || surface.Summary.ReplayWindowMS != 0 {
		t.Fatalf("unexpected backlog metrics: %+v", surface.Summary)
	}

	if len(surface.Scenarios) != 1 {
		t.Fatalf("expected exactly one scenario, got %d", len(surface.Scenarios))
	}
	scenario := surface.Scenarios[0]
	if scenario.Name != "http_remote_service_replay_handoff_keeps_live_lane_unblocked" || scenario.Status != "isolated" || scenario.ReplayBacklogEvents != 4 || !scenario.ReplayDrainsAfterLive {
		t.Fatalf("unexpected provider scenario: %+v", scenario)
	}
	if scenario.ReplayStepDelayMS != 0 || scenario.ReplayWindowMS != 0 || scenario.LiveDeliveryDeadline != 200 {
		t.Fatalf("unexpected scenario timing: %+v", scenario)
	}
	if !matchesSubset(scenario.SourceTests, []string{"internal/api/server_test.go"}) {
		t.Fatalf("missing internal API test reference: %+v", scenario.SourceTests)
	}
	if !matchesNote(scenario.Notes, "remote service") || !matchesNote(scenario.Notes, "handoff") {
		t.Fatalf("scenario notes missing key phrases, got %+v", scenario.Notes)
	}

	if len(surface.Limitations) != 2 || !matchesNote(surface.Limitations, "http_remote_service") || !matchesNote(surface.Limitations, "in-process event bus") {
		t.Fatalf("unexpected provider limitations: %+v", surface.Limitations)
	}
}

func matchesSubset(haystack []string, needles []string) bool {
	for _, needle := range needles {
		found := false
		for _, entry := range haystack {
			if entry == needle {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func matchesNote(notes []string, snippet string) bool {
	for _, note := range notes {
		if strings.Contains(strings.ToLower(note), strings.ToLower(snippet)) {
			return true
		}
	}
	return false
}
