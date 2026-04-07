package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const coordinationCapabilitySurfacePath = "docs/reports/cross-process-coordination-capability-surface.json"

type coordinationCapabilitySurface struct {
	ReportPath       string                               `json:"report_path"`
	GeneratedAt      string                               `json:"generated_at,omitempty"`
	Ticket           string                               `json:"ticket,omitempty"`
	Title            string                               `json:"title,omitempty"`
	Status           string                               `json:"status,omitempty"`
	EvidenceSources  coordinationCapabilityEvidenceSource `json:"evidence_sources"`
	Summary          coordinationCapabilitySummary        `json:"summary"`
	Capabilities     []coordinationCapabilityView         `json:"capabilities"`
	CurrentCeiling   []string                             `json:"current_ceiling,omitempty"`
	NextRuntimeHooks []string                             `json:"next_runtime_hooks,omitempty"`
}

type coordinationCapabilityEvidenceSource struct {
	SharedQueueReport     string   `json:"shared_queue_report,omitempty"`
	TakeoverHarnessReport string   `json:"takeover_harness_report,omitempty"`
	LiveTakeoverReport    string   `json:"live_takeover_report,omitempty"`
	SupportingDocs        []string `json:"supporting_docs,omitempty"`
}

type coordinationCapabilitySummary struct {
	CapabilityCount        int            `json:"capability_count"`
	CurrentStateCounts     map[string]int `json:"current_state_counts"`
	RuntimeReadinessCounts map[string]int `json:"runtime_readiness_counts"`
	ContractOnlyCount      int            `json:"contract_only_count"`
	HarnessProvenCount     int            `json:"harness_proven_count"`
	LiveProvenCount        int            `json:"live_proven_count"`
}

type coordinationCapabilityView struct {
	Name              string   `json:"name"`
	CurrentState      string   `json:"current_state"`
	RuntimeReadiness  string   `json:"runtime_readiness"`
	ContractOnly      bool     `json:"contract_only"`
	HarnessProven     bool     `json:"harness_proven"`
	LiveProven        bool     `json:"live_proven"`
	SourceReportLinks []string `json:"source_report_links"`
	Notes             []string `json:"notes,omitempty"`
}

type coordinationCapabilitySurfaceDocument struct {
	GeneratedAt    string `json:"generated_at"`
	Ticket         string `json:"ticket"`
	Title          string `json:"title"`
	Status         string `json:"status"`
	EvidenceInputs struct {
		SharedQueueReport     string   `json:"shared_queue_report"`
		TakeoverHarnessReport string   `json:"takeover_harness_report"`
		LiveTakeoverReport    string   `json:"live_takeover_report"`
		SupportingDocs        []string `json:"supporting_docs"`
	} `json:"evidence_inputs"`
	Capabilities []struct {
		Capability           string   `json:"capability"`
		CurrentState         string   `json:"current_state"`
		RuntimeReadiness     string   `json:"runtime_readiness"`
		LiveLocalProof       bool     `json:"live_local_proof"`
		DeterministicHarness bool     `json:"deterministic_local_harness"`
		ContractDefined      bool     `json:"contract_defined_target"`
		Notes                []string `json:"notes"`
	} `json:"capabilities"`
	CurrentCeiling   []string `json:"current_ceiling"`
	NextRuntimeHooks []string `json:"next_runtime_hooks"`
}

func coordinationCapabilitySurfacePayload() any {
	reportPath := resolveRepoRelativePath(coordinationCapabilitySurfacePath)
	if reportPath == "" {
		return map[string]any{
			"report_path": coordinationCapabilitySurfacePath,
			"status":      "unavailable",
			"error":       "report path could not be resolved",
		}
	}
	contents, err := os.ReadFile(reportPath)
	if err != nil {
		return map[string]any{
			"report_path": coordinationCapabilitySurfacePath,
			"status":      "unavailable",
			"error":       err.Error(),
		}
	}
	var document coordinationCapabilitySurfaceDocument
	if err := json.Unmarshal(contents, &document); err != nil {
		return map[string]any{
			"report_path": coordinationCapabilitySurfacePath,
			"status":      "invalid",
			"error":       err.Error(),
		}
	}
	surface := coordinationCapabilitySurface{
		ReportPath:  coordinationCapabilitySurfacePath,
		GeneratedAt: document.GeneratedAt,
		Ticket:      document.Ticket,
		Title:       document.Title,
		Status:      document.Status,
		EvidenceSources: coordinationCapabilityEvidenceSource{
			SharedQueueReport:     document.EvidenceInputs.SharedQueueReport,
			TakeoverHarnessReport: document.EvidenceInputs.TakeoverHarnessReport,
			LiveTakeoverReport:    document.EvidenceInputs.LiveTakeoverReport,
			SupportingDocs:        append([]string(nil), document.EvidenceInputs.SupportingDocs...),
		},
		Summary: coordinationCapabilitySummary{
			CapabilityCount:        len(document.Capabilities),
			CurrentStateCounts:     map[string]int{},
			RuntimeReadinessCounts: map[string]int{},
		},
		Capabilities:     make([]coordinationCapabilityView, 0, len(document.Capabilities)),
		CurrentCeiling:   append([]string(nil), document.CurrentCeiling...),
		NextRuntimeHooks: append([]string(nil), document.NextRuntimeHooks...),
	}
	for _, item := range document.Capabilities {
		surface.Summary.CurrentStateCounts[item.CurrentState]++
		surface.Summary.RuntimeReadinessCounts[item.RuntimeReadiness]++
		switch item.RuntimeReadiness {
		case "contract_only":
			surface.Summary.ContractOnlyCount++
		case "harness_proven":
			surface.Summary.HarnessProvenCount++
		case "live_proven":
			surface.Summary.LiveProvenCount++
		}
		surface.Capabilities = append(surface.Capabilities, coordinationCapabilityView{
			Name:              item.Capability,
			CurrentState:      item.CurrentState,
			RuntimeReadiness:  item.RuntimeReadiness,
			ContractOnly:      item.RuntimeReadiness == "contract_only",
			HarnessProven:     item.RuntimeReadiness == "harness_proven" || item.DeterministicHarness,
			LiveProven:        item.RuntimeReadiness == "live_proven" || item.LiveLocalProof,
			SourceReportLinks: coordinationCapabilitySources(item, document.EvidenceInputs.SharedQueueReport, document.EvidenceInputs.TakeoverHarnessReport, document.EvidenceInputs.LiveTakeoverReport, document.EvidenceInputs.SupportingDocs),
			Notes:             append([]string(nil), item.Notes...),
		})
	}
	return surface
}

func coordinationCapabilitySources(item struct {
	Capability           string   `json:"capability"`
	CurrentState         string   `json:"current_state"`
	RuntimeReadiness     string   `json:"runtime_readiness"`
	LiveLocalProof       bool     `json:"live_local_proof"`
	DeterministicHarness bool     `json:"deterministic_local_harness"`
	ContractDefined      bool     `json:"contract_defined_target"`
	Notes                []string `json:"notes"`
}, sharedQueueReport string, takeoverHarnessReport string, liveTakeoverReport string, supportingDocs []string) []string {
	seen := make(map[string]struct{})
	sources := make([]string, 0, 3+len(supportingDocs))
	appendSource := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		sources = append(sources, value)
	}
	if item.LiveLocalProof {
		if item.Capability == "shared_queue_task_coordination" {
			appendSource(sharedQueueReport)
		} else {
			appendSource(liveTakeoverReport)
		}
	}
	if item.DeterministicHarness {
		appendSource(takeoverHarnessReport)
	}
	if item.ContractDefined {
		for _, path := range supportingDocs {
			appendSource(path)
		}
	}
	if len(sources) == 0 {
		appendSource(sharedQueueReport)
		appendSource(takeoverHarnessReport)
		appendSource(liveTakeoverReport)
	}
	sort.Strings(sources)
	return sources
}

func resolveRepoRelativePath(relative string) string {
	if _, currentFile, _, ok := runtime.Caller(0); ok {
		base := filepath.Dir(currentFile)
		candidate := filepath.Clean(filepath.Join(base, "..", "..", relative))
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	if wd, err := os.Getwd(); err == nil {
		for _, root := range []string{wd, filepath.Join(wd, "bigclaw-go")} {
			candidate := filepath.Clean(filepath.Join(root, relative))
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
	}
	return ""
}
