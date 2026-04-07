package api

import (
	"encoding/json"
	"fmt"
	"os"
)

const publishAckOutcomeSurfacePath = "docs/reports/publish-ack-outcome-surface.json"

type publishAckOutcomeSurface struct {
	ReportPath    string                   `json:"report_path"`
	GeneratedAt   string                   `json:"generated_at,omitempty"`
	Ticket        string                   `json:"ticket,omitempty"`
	Track         string                   `json:"track,omitempty"`
	Title         string                   `json:"title,omitempty"`
	Status        string                   `json:"status,omitempty"`
	Summary       publishAckOutcomeSummary `json:"summary"`
	SourceReports []string                 `json:"source_reports,omitempty"`
	ReviewerLinks []string                 `json:"reviewer_links,omitempty"`
	Outcomes      []publishAckOutcomeClass `json:"outcomes,omitempty"`
	Limitations   []string                 `json:"limitations,omitempty"`
	Error         string                   `json:"error,omitempty"`
}

type publishAckOutcomeSummary struct {
	ScenarioID         string   `json:"scenario_id,omitempty"`
	ProofStatus        string   `json:"proof_status,omitempty"`
	RequiredOutcomes   []string `json:"required_outcomes,omitempty"`
	CommittedCount     int      `json:"committed_count"`
	RejectedCount      int      `json:"rejected_count"`
	UnknownCommitCount int      `json:"unknown_commit_count"`
}

type publishAckOutcomeClass struct {
	Outcome          string   `json:"outcome"`
	ProofRule        string   `json:"proof_rule,omitempty"`
	RequiredEvidence []string `json:"required_evidence,omitempty"`
	OperatorAction   string   `json:"operator_action,omitempty"`
}

func publishAckOutcomeSurfacePayload() publishAckOutcomeSurface {
	surface := publishAckOutcomeSurface{ReportPath: publishAckOutcomeSurfacePath}
	reportPath := resolveRepoRelativePath(publishAckOutcomeSurfacePath)
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
		surface.Error = fmt.Sprintf("decode %s: %v", publishAckOutcomeSurfacePath, err)
		return surface
	}
	surface.ReportPath = publishAckOutcomeSurfacePath
	return surface
}
