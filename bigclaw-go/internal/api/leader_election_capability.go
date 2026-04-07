package api

import (
	"encoding/json"
	"os"
	"sort"
)

const leaderElectionCapabilitySurfacePath = "docs/reports/leader-election-capability-surface.json"

type leaderElectionCapabilitySurface struct {
	ReportPath       string                                  `json:"report_path"`
	GeneratedAt      string                                  `json:"generated_at,omitempty"`
	Ticket           string                                  `json:"ticket,omitempty"`
	Title            string                                  `json:"title,omitempty"`
	Status           string                                  `json:"status,omitempty"`
	EvidenceSources  leaderElectionCapabilityEvidenceSources `json:"evidence_sources"`
	Summary          leaderElectionCapabilitySummary         `json:"summary"`
	Backends         []leaderElectionBackendView             `json:"backends"`
	EndpointCoverage []string                                `json:"endpoint_coverage,omitempty"`
}

type leaderElectionCapabilityEvidenceSources struct {
	CoordinationReport            string   `json:"coordination_report,omitempty"`
	SharedQueueReport             string   `json:"shared_queue_report,omitempty"`
	TakeoverHarnessReport         string   `json:"takeover_harness_report,omitempty"`
	LiveTakeoverReport            string   `json:"live_takeover_report,omitempty"`
	CoordinationCapabilitySurface string   `json:"coordination_capability_surface,omitempty"`
	SupportingDocs                []string `json:"supporting_docs,omitempty"`
}

type leaderElectionCapabilitySummary struct {
	BackendCount          int            `json:"backend_count"`
	RuntimeReadiness      map[string]int `json:"runtime_readiness"`
	LiveProvenBackends    int            `json:"live_proven_backends"`
	HarnessProvenBackends int            `json:"harness_proven_backends"`
	ContractOnlyBackends  int            `json:"contract_only_backends"`
	CurrentProofBackend   string         `json:"current_proof_backend,omitempty"`
}

type leaderElectionBackendView struct {
	Name             string   `json:"name"`
	RuntimeReadiness string   `json:"runtime_readiness"`
	ProofScope       string   `json:"proof_scope,omitempty"`
	BackingStore     string   `json:"backing_store,omitempty"`
	EndpointSurfaces []string `json:"endpoint_surfaces,omitempty"`
	Notes            []string `json:"notes,omitempty"`
}

type leaderElectionCapabilityDocument struct {
	GeneratedAt    string `json:"generated_at"`
	Ticket         string `json:"ticket"`
	Title          string `json:"title"`
	Status         string `json:"status"`
	EvidenceInputs struct {
		CoordinationReport            string   `json:"coordination_report"`
		SharedQueueReport             string   `json:"shared_queue_report"`
		TakeoverHarnessReport         string   `json:"takeover_harness_report"`
		LiveTakeoverReport            string   `json:"live_takeover_report"`
		CoordinationCapabilitySurface string   `json:"coordination_capability_surface"`
		SupportingDocs                []string `json:"supporting_docs"`
	} `json:"evidence_inputs"`
	Summary struct {
		BackendCount          int    `json:"backend_count"`
		LiveProvenBackends    int    `json:"live_proven_backends"`
		HarnessProvenBackends int    `json:"harness_proven_backends"`
		ContractOnlyBackends  int    `json:"contract_only_backends"`
		CurrentProofBackend   string `json:"current_proof_backend"`
	} `json:"summary"`
	Backends []struct {
		Backend          string   `json:"backend"`
		RuntimeReadiness string   `json:"runtime_readiness"`
		ProofScope       string   `json:"proof_scope"`
		BackingStore     string   `json:"backing_store"`
		EndpointSurfaces []string `json:"endpoint_surfaces"`
		Notes            []string `json:"notes"`
	} `json:"backends"`
}

func leaderElectionCapabilitySurfacePayload() any {
	reportPath := resolveRepoRelativePath(leaderElectionCapabilitySurfacePath)
	if reportPath == "" {
		return map[string]any{
			"report_path": leaderElectionCapabilitySurfacePath,
			"status":      "unavailable",
			"error":       "report path could not be resolved",
		}
	}
	contents, err := os.ReadFile(reportPath)
	if err != nil {
		return map[string]any{
			"report_path": leaderElectionCapabilitySurfacePath,
			"status":      "unavailable",
			"error":       err.Error(),
		}
	}
	var document leaderElectionCapabilityDocument
	if err := json.Unmarshal(contents, &document); err != nil {
		return map[string]any{
			"report_path": leaderElectionCapabilitySurfacePath,
			"status":      "invalid",
			"error":       err.Error(),
		}
	}
	surface := leaderElectionCapabilitySurface{
		ReportPath:  leaderElectionCapabilitySurfacePath,
		GeneratedAt: document.GeneratedAt,
		Ticket:      document.Ticket,
		Title:       document.Title,
		Status:      document.Status,
		EvidenceSources: leaderElectionCapabilityEvidenceSources{
			CoordinationReport:            document.EvidenceInputs.CoordinationReport,
			SharedQueueReport:             document.EvidenceInputs.SharedQueueReport,
			TakeoverHarnessReport:         document.EvidenceInputs.TakeoverHarnessReport,
			LiveTakeoverReport:            document.EvidenceInputs.LiveTakeoverReport,
			CoordinationCapabilitySurface: document.EvidenceInputs.CoordinationCapabilitySurface,
			SupportingDocs:                append([]string(nil), document.EvidenceInputs.SupportingDocs...),
		},
		Summary: leaderElectionCapabilitySummary{
			BackendCount:          document.Summary.BackendCount,
			RuntimeReadiness:      map[string]int{},
			LiveProvenBackends:    document.Summary.LiveProvenBackends,
			HarnessProvenBackends: document.Summary.HarnessProvenBackends,
			ContractOnlyBackends:  document.Summary.ContractOnlyBackends,
			CurrentProofBackend:   document.Summary.CurrentProofBackend,
		},
		Backends: make([]leaderElectionBackendView, 0, len(document.Backends)),
	}
	endpointCoverage := make(map[string]struct{})
	for _, backend := range document.Backends {
		surface.Summary.RuntimeReadiness[backend.RuntimeReadiness]++
		row := leaderElectionBackendView{
			Name:             backend.Backend,
			RuntimeReadiness: backend.RuntimeReadiness,
			ProofScope:       backend.ProofScope,
			BackingStore:     backend.BackingStore,
			EndpointSurfaces: append([]string(nil), backend.EndpointSurfaces...),
			Notes:            append([]string(nil), backend.Notes...),
		}
		sort.Strings(row.EndpointSurfaces)
		for _, endpoint := range row.EndpointSurfaces {
			endpointCoverage[endpoint] = struct{}{}
		}
		surface.Backends = append(surface.Backends, row)
	}
	for endpoint := range endpointCoverage {
		surface.EndpointCoverage = append(surface.EndpointCoverage, endpoint)
	}
	sort.Strings(surface.EndpointCoverage)
	return surface
}
