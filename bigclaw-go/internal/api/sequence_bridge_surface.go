package api

import (
	"encoding/json"
	"fmt"
	"os"
)

const sequenceBridgeSurfacePath = "docs/reports/sequence-bridge-capability-surface.json"

type sequenceBridgeSurface struct {
	ReportPath       string                        `json:"report_path"`
	GeneratedAt      string                        `json:"generated_at,omitempty"`
	Ticket           string                        `json:"ticket,omitempty"`
	Track            string                        `json:"track,omitempty"`
	Title            string                        `json:"title,omitempty"`
	Status           string                        `json:"status,omitempty"`
	EvidenceSources  sequenceBridgeEvidenceSources `json:"evidence_sources"`
	Summary          sequenceBridgeSummary         `json:"summary"`
	Backends         []sequenceBridgeBackendView   `json:"backends,omitempty"`
	CurrentCeiling   []string                      `json:"current_ceiling,omitempty"`
	NextRuntimeHooks []string                      `json:"next_runtime_hooks,omitempty"`
	Error            string                        `json:"error,omitempty"`
}

type sequenceBridgeEvidenceSources struct {
	DurabilitySpike        string   `json:"durability_spike,omitempty"`
	CheckpointFencingProof string   `json:"checkpoint_fencing_proof,omitempty"`
	CoordinationReport     string   `json:"coordination_report,omitempty"`
	SupportingDocs         []string `json:"supporting_docs,omitempty"`
}

type sequenceBridgeSummary struct {
	BackendCount                 int `json:"backend_count"`
	LiveProvenBackends           int `json:"live_proven_backends"`
	HarnessProvenBackends        int `json:"harness_proven_backends"`
	ContractOnlyBackends         int `json:"contract_only_backends"`
	OneToOneMappings             int `json:"one_to_one_mappings"`
	ProviderEpochBridgedBackends int `json:"provider_epoch_bridged_backends"`
}

type sequenceBridgeBackendView struct {
	Backend                string   `json:"backend"`
	RuntimeReadiness       string   `json:"runtime_readiness"`
	PortableSequenceSource string   `json:"portable_sequence_source,omitempty"`
	ProviderOffsetSource   string   `json:"provider_offset_source,omitempty"`
	OwnershipEpochSource   string   `json:"ownership_epoch_source,omitempty"`
	MappingContract        string   `json:"mapping_contract,omitempty"`
	SourceReportLinks      []string `json:"source_report_links,omitempty"`
	Notes                  []string `json:"notes,omitempty"`
}

func sequenceBridgeSurfacePayload() sequenceBridgeSurface {
	surface := sequenceBridgeSurface{ReportPath: sequenceBridgeSurfacePath}
	reportPath := resolveRepoRelativePath(sequenceBridgeSurfacePath)
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
		surface.Error = fmt.Sprintf("decode %s: %v", sequenceBridgeSurfacePath, err)
		return surface
	}
	surface.ReportPath = sequenceBridgeSurfacePath
	return surface
}
