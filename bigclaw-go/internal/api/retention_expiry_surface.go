package api

import (
	"encoding/json"
	"fmt"
	"os"
)

const retentionExpirySurfacePath = "docs/reports/retention-watermark-expiry-surface.json"

type retentionExpirySurface struct {
	ReportPath    string                        `json:"report_path"`
	GeneratedAt   string                        `json:"generated_at,omitempty"`
	Ticket        string                        `json:"ticket,omitempty"`
	Track         string                        `json:"track,omitempty"`
	Title         string                        `json:"title,omitempty"`
	Status        string                        `json:"status,omitempty"`
	SourceReports []string                      `json:"source_reports,omitempty"`
	ReviewerLinks []string                      `json:"reviewer_links,omitempty"`
	Summary       retentionExpirySurfaceSummary `json:"summary"`
	Backends      []retentionExpiryBackendView  `json:"backends,omitempty"`
	PolicySplit   []string                      `json:"policy_split,omitempty"`
	Error         string                        `json:"error,omitempty"`
}

type retentionExpirySurfaceSummary struct {
	BackendCount              int `json:"backend_count"`
	RuntimeVisibleBackends    int `json:"runtime_visible_backends"`
	PersistedBoundaryBackends int `json:"persisted_boundary_backends"`
	FailClosedExpiryBackends  int `json:"fail_closed_expiry_backends"`
	ContractOnlyBackends      int `json:"contract_only_backends"`
}

type retentionExpiryBackendView struct {
	Backend                  string   `json:"backend"`
	RuntimeReadiness         string   `json:"runtime_readiness"`
	RetainedBoundaryVisible  bool     `json:"retained_boundary_visible"`
	PersistedBoundaries      bool     `json:"persisted_boundaries"`
	FailClosedExpiry         bool     `json:"fail_closed_expiry"`
	ReplayBoundarySource     string   `json:"replay_boundary_source,omitempty"`
	CheckpointExpiryHandling string   `json:"checkpoint_expiry_handling,omitempty"`
	CheckpointCleanupPolicy  string   `json:"checkpoint_cleanup_policy,omitempty"`
	SourceReportLinks        []string `json:"source_report_links,omitempty"`
	Notes                    []string `json:"notes,omitempty"`
}

func retentionExpirySurfacePayload() retentionExpirySurface {
	surface := retentionExpirySurface{ReportPath: retentionExpirySurfacePath}
	reportPath := resolveRepoRelativePath(retentionExpirySurfacePath)
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
		surface.Error = fmt.Sprintf("decode %s: %v", retentionExpirySurfacePath, err)
		return surface
	}
	surface.ReportPath = retentionExpirySurfacePath
	return surface
}
