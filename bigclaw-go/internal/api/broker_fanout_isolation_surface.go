package api

import (
	"encoding/json"
	"fmt"
	"os"
)

const brokerStubFanoutIsolationEvidencePackPath = "docs/reports/broker-stub-live-fanout-isolation-evidence-pack.json"

type brokerStubFanoutIsolationEvidencePack struct {
	ReportPath      string                              `json:"report_path"`
	GeneratedAt     string                              `json:"generated_at,omitempty"`
	Ticket          string                              `json:"ticket,omitempty"`
	Title           string                              `json:"title,omitempty"`
	Status          string                              `json:"status,omitempty"`
	Backend         string                              `json:"backend,omitempty"`
	EvidenceSources []string                            `json:"evidence_sources,omitempty"`
	ReviewerLinks   []string                            `json:"reviewer_links,omitempty"`
	Summary         brokerStubFanoutIsolationSummary    `json:"summary"`
	Scenarios       []brokerStubFanoutIsolationScenario `json:"scenarios,omitempty"`
	Error           string                              `json:"error,omitempty"`
}

type brokerStubFanoutIsolationSummary struct {
	ScenarioCount          int  `json:"scenario_count"`
	IsolatedScenarios      int  `json:"isolated_scenarios"`
	StalledScenarios       int  `json:"stalled_scenarios"`
	ReplayBacklogEvents    int  `json:"replay_backlog_events"`
	ReplayStepDelayMS      int  `json:"replay_step_delay_ms"`
	ReplayWindowMS         int  `json:"replay_window_ms"`
	LiveDeliveryDeadlineMS int  `json:"live_delivery_deadline_ms"`
	IsolationMaintained    bool `json:"isolation_maintained"`
}

type brokerStubFanoutIsolationScenario struct {
	Name                   string   `json:"name"`
	Status                 string   `json:"status"`
	ReplayPath             string   `json:"replay_path,omitempty"`
	LivePath               string   `json:"live_path,omitempty"`
	ReplayBacklogEvents    int      `json:"replay_backlog_events"`
	ReplayStepDelayMS      int      `json:"replay_step_delay_ms"`
	ReplayWindowMS         int      `json:"replay_window_ms"`
	LiveDeliveryDeadlineMS int      `json:"live_delivery_deadline_ms"`
	ReplayDrainsAfterLive  bool     `json:"replay_drains_after_live"`
	SourceTests            []string `json:"source_tests,omitempty"`
	Notes                  []string `json:"notes,omitempty"`
}

func brokerStubFanoutIsolationPayload() brokerStubFanoutIsolationEvidencePack {
	surface := brokerStubFanoutIsolationEvidencePack{ReportPath: brokerStubFanoutIsolationEvidencePackPath}
	reportPath := resolveRepoRelativePath(brokerStubFanoutIsolationEvidencePackPath)
	if reportPath == "" {
		surface.Status = "unavailable"
		surface.Error = "report path could not be resolved"
		return surface
	}
	contents, err := os.ReadFile(reportPath)
	if err != nil {
		surface.Status = "unavailable"
		surface.Error = err.Error()
		return surface
	}
	if err := json.Unmarshal(contents, &surface); err != nil {
		surface.Status = "invalid"
		surface.Error = fmt.Sprintf("decode %s: %v", brokerStubFanoutIsolationEvidencePackPath, err)
		return surface
	}
	surface.ReportPath = brokerStubFanoutIsolationEvidencePackPath
	return surface
}
