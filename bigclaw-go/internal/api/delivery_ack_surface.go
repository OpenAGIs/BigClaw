package api

import (
	"encoding/json"
	"fmt"
	"os"
)

const deliveryAckReadinessSurfacePath = "docs/reports/delivery-ack-readiness-surface.json"

type deliveryAckReadinessSurface struct {
	ReportPath      string                      `json:"report_path"`
	GeneratedAt     string                      `json:"generated_at,omitempty"`
	Ticket          string                      `json:"ticket,omitempty"`
	Title           string                      `json:"title,omitempty"`
	Status          string                      `json:"status,omitempty"`
	EvidenceSources []string                    `json:"evidence_sources,omitempty"`
	ReviewerLinks   []string                    `json:"reviewer_links,omitempty"`
	Summary         deliveryAckReadinessSummary `json:"summary"`
	Backends        []deliveryAckBackendView    `json:"backends,omitempty"`
	Error           string                      `json:"error,omitempty"`
}

type deliveryAckReadinessSummary struct {
	BackendCount         int `json:"backend_count"`
	ExplicitAckBackends  int `json:"explicit_ack_backends"`
	DurableAckBackends   int `json:"durable_ack_backends"`
	BestEffortBackends   int `json:"best_effort_backends"`
	ContractOnlyBackends int `json:"contract_only_backends"`
}

type deliveryAckBackendView struct {
	Backend                 string   `json:"backend"`
	Scope                   string   `json:"scope,omitempty"`
	PublishMode             string   `json:"publish_mode,omitempty"`
	AcknowledgementClass    string   `json:"acknowledgement_class"`
	ExplicitAcknowledgement bool     `json:"explicit_acknowledgement"`
	DurableAcknowledgement  bool     `json:"durable_acknowledgement"`
	RuntimeReadiness        string   `json:"runtime_readiness"`
	SourceReportLinks       []string `json:"source_report_links,omitempty"`
	Notes                   []string `json:"notes,omitempty"`
}

func deliveryAckReadinessPayload() deliveryAckReadinessSurface {
	surface := deliveryAckReadinessSurface{ReportPath: deliveryAckReadinessSurfacePath}
	reportPath := resolveRepoRelativePath(deliveryAckReadinessSurfacePath)
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
		surface.Error = fmt.Sprintf("decode %s: %v", deliveryAckReadinessSurfacePath, err)
		return surface
	}
	surface.ReportPath = deliveryAckReadinessSurfacePath
	return surface
}
