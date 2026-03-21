package api

import (
	"encoding/json"
	"fmt"
	"os"
)

const providerLiveHandoffIsolationEvidencePackPath = "docs/reports/provider-live-handoff-isolation-evidence-pack.json"

type providerLiveHandoffIsolationEvidencePack struct {
	ReportPath      string                              `json:"report_path"`
	GeneratedAt     string                              `json:"generated_at,omitempty"`
	Ticket          string                              `json:"ticket,omitempty"`
	Track           string                              `json:"track,omitempty"`
	Title           string                              `json:"title,omitempty"`
	Status          string                              `json:"status,omitempty"`
	Backend         string                              `json:"backend,omitempty"`
	ValidationLane  string                              `json:"validation_lane,omitempty"`
	EvidenceSources []string                            `json:"evidence_sources,omitempty"`
	ReviewerLinks   []string                            `json:"reviewer_links,omitempty"`
	Summary         brokerStubFanoutIsolationSummary    `json:"summary"`
	Scenarios       []brokerStubFanoutIsolationScenario `json:"scenarios,omitempty"`
	Limitations     []string                            `json:"limitations,omitempty"`
	Error           string                              `json:"error,omitempty"`
}

func providerLiveHandoffIsolationPayload() providerLiveHandoffIsolationEvidencePack {
	surface := providerLiveHandoffIsolationEvidencePack{ReportPath: providerLiveHandoffIsolationEvidencePackPath}
	reportPath := resolveRepoRelativePath(providerLiveHandoffIsolationEvidencePackPath)
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
		surface.Error = fmt.Sprintf("decode %s: %v", providerLiveHandoffIsolationEvidencePackPath, err)
		return surface
	}
	surface.ReportPath = providerLiveHandoffIsolationEvidencePackPath
	return surface
}
