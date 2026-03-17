package api

import (
	"encoding/json"
	"fmt"
	"os"

	"bigclaw-go/internal/events"
)

const brokerBootstrapSurfacePath = "docs/reports/broker-validation-summary.json"

type brokerBootstrapSurface struct {
	ReportPath                    string                              `json:"report_path"`
	Enabled                       bool                                `json:"enabled"`
	Backend                       string                              `json:"backend,omitempty"`
	BundleSummaryPath             string                              `json:"bundle_summary_path,omitempty"`
	CanonicalSummaryPath          string                              `json:"canonical_summary_path,omitempty"`
	BundleBootstrapSummaryPath    string                              `json:"bundle_bootstrap_summary_path,omitempty"`
	CanonicalBootstrapSummaryPath string                              `json:"canonical_bootstrap_summary_path,omitempty"`
	ValidationPackPath            string                              `json:"validation_pack_path,omitempty"`
	ConfigurationState            string                              `json:"configuration_state,omitempty"`
	BootstrapSummary              events.BrokerBootstrapReviewSummary `json:"bootstrap_summary"`
	BootstrapReady                bool                                `json:"bootstrap_ready"`
	RuntimePosture                string                              `json:"runtime_posture,omitempty"`
	LiveAdapterImplemented        bool                                `json:"live_adapter_implemented"`
	ProofBoundary                 string                              `json:"proof_boundary,omitempty"`
	ValidationErrors              []string                            `json:"validation_errors,omitempty"`
	ConfigCompleteness            events.BrokerBootstrapCompleteness  `json:"config_completeness"`
	Status                        string                              `json:"status,omitempty"`
	Reason                        string                              `json:"reason,omitempty"`
	Error                         string                              `json:"error,omitempty"`
}

func brokerBootstrapSurfacePayload() brokerBootstrapSurface {
	surface := brokerBootstrapSurface{ReportPath: brokerBootstrapSurfacePath}
	reportPath := resolveRepoRelativePath(brokerBootstrapSurfacePath)
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
		surface.Error = fmt.Sprintf("decode %s: %v", brokerBootstrapSurfacePath, err)
		return surface
	}
	surface.ReportPath = brokerBootstrapSurfacePath
	return surface
}
