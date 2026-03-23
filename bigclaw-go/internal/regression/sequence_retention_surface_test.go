package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type sequenceBridgeSurface struct {
	EvidenceSources struct {
		DurabilitySpike    string   `json:"durability_spike"`
		CheckpointFencing  string   `json:"checkpoint_fencing_proof"`
		CoordinationReport string   `json:"coordination_report"`
		SupportingDocs     []string `json:"supporting_docs"`
	} `json:"evidence_sources"`
	Summary struct {
		BackendCount          int `json:"backend_count"`
		LiveProvenBackends    int `json:"live_proven_backends"`
		HarnessProvenBackends int `json:"harness_proven_backends"`
		ContractOnlyBackends  int `json:"contract_only_backends"`
		OneToOneMappings      int `json:"one_to_one_mappings"`
		ProviderEpochBridged  int `json:"provider_epoch_bridged_backends"`
	} `json:"summary"`
	Backends []struct {
		Backend           string   `json:"backend"`
		RuntimeReadiness  string   `json:"runtime_readiness"`
		SourceReportLinks []string `json:"source_report_links"`
		Notes             []string `json:"notes"`
		MappingContract   string   `json:"mapping_contract"`
	} `json:"backends"`
	CurrentCeiling   []string `json:"current_ceiling"`
	NextRuntimeHooks []string `json:"next_runtime_hooks"`
}

type retentionSurface struct {
	SourceReports []string `json:"source_reports"`
	ReviewerLinks []string `json:"reviewer_links"`
	Summary       struct {
		BackendCount             int `json:"backend_count"`
		RuntimeVisibleBackends   int `json:"runtime_visible_backends"`
		PersistedBoundaries      int `json:"persisted_boundary_backends"`
		FailClosedExpiryBackends int `json:"fail_closed_expiry_backends"`
		ContractOnlyBackends     int `json:"contract_only_backends"`
	} `json:"summary"`
	Backends []struct {
		Backend                  string   `json:"backend"`
		RuntimeReadiness         string   `json:"runtime_readiness"`
		SourceReportLinks        []string `json:"source_report_links"`
		FailClosedExpiry         bool     `json:"fail_closed_expiry"`
		Notes                    []string `json:"notes"`
		CheckpointExpiryHandling string   `json:"checkpoint_expiry_handling"`
	} `json:"backends"`
	PolicySplit []string `json:"policy_split"`
}

func TestSequenceBridgeSurfaceStaysAligned(t *testing.T) {
	root := repoRoot(t)

	var surface sequenceBridgeSurface
	readJSONFile(t, filepath.Join(root, "docs", "reports", "sequence-bridge-capability-surface.json"), &surface)

	if surface.Summary.BackendCount != 5 ||
		surface.Summary.LiveProvenBackends != 3 ||
		surface.Summary.HarnessProvenBackends != 1 ||
		surface.Summary.ContractOnlyBackends != 1 ||
		surface.Summary.OneToOneMappings != 2 ||
		surface.Summary.ProviderEpochBridged != 3 {
		t.Fatalf("unexpected sequence bridge summary: %+v", surface.Summary)
	}

	requiredSources := []string{
		surface.EvidenceSources.DurabilitySpike,
		surface.EvidenceSources.CheckpointFencing,
		surface.EvidenceSources.CoordinationReport,
	}
	requiredSources = append(requiredSources, surface.EvidenceSources.SupportingDocs...)
	for _, candidate := range requiredSources {
		if candidate == "" {
			t.Fatalf("sequence bridge evidence missing path: %+v", surface.EvidenceSources)
		}
		if _, err := os.Stat(resolveRepoPath(root, candidate)); err != nil {
			t.Fatalf("sequence bridge evidence %q missing: %v", candidate, err)
		}
	}

	contractBackendFound := false
	for _, backend := range surface.Backends {
		if backend.RuntimeReadiness == "contract_only" {
			contractBackendFound = true
			if !stringContains(backend.SourceReportLinks, "docs/reports/replicated-broker-durability-rollout-spike.md") ||
				!stringContains(backend.SourceReportLinks, "docs/reports/replicated-event-log-durability-rollout-contract.md") {
				t.Fatalf("contract-only backend missing rollout links: %+v", backend.SourceReportLinks)
			}
			if !stringContains(backend.Notes, "No live provider-backed adapter") {
				t.Fatalf("contract-only backend note missing reality check: %+v", backend.Notes)
			}
			break
		}
	}
	if !contractBackendFound {
		t.Fatal("contract-only backend entry missing from sequence bridge surface")
	}

	if !stringContains(surface.CurrentCeiling, "broker-backed mapping remains future work") ||
		!stringContains(surface.NextRuntimeHooks, "Keep broker_replicated marked contract_only") {
		t.Fatalf("sequence bridge ceiling/hooks drifted: ceiling=%+v hooks=%+v", surface.CurrentCeiling, surface.NextRuntimeHooks)
	}
}

func TestRetentionWatermarkSurfaceStaysAligned(t *testing.T) {
	root := repoRoot(t)

	var surface retentionSurface
	readJSONFile(t, filepath.Join(root, "docs", "reports", "retention-watermark-expiry-surface.json"), &surface)

	if surface.Summary.BackendCount != 5 ||
		surface.Summary.RuntimeVisibleBackends != 4 ||
		surface.Summary.PersistedBoundaries != 2 ||
		surface.Summary.FailClosedExpiryBackends != 3 ||
		surface.Summary.ContractOnlyBackends != 1 {
		t.Fatalf("unexpected retention summary: %+v", surface.Summary)
	}

	for _, path := range append(surface.SourceReports, surface.ReviewerLinks...) {
		if strings.TrimSpace(path) == "" {
			t.Fatalf("retention surface references empty path: %+v", surface.SourceReports)
		}
		if path[0] == '/' {
			continue
		}
		if _, err := os.Stat(resolveRepoPath(root, path)); err != nil {
			t.Fatalf("retention surface path missing %q: %v", path, err)
		}
	}

	contractBackendFound := false
	for _, backend := range surface.Backends {
		if backend.RuntimeReadiness == "contract_only" {
			contractBackendFound = true
			if backend.FailClosedExpiry {
				t.Fatalf("contract-only backend should not claim fail-closed expiry: %+v", backend)
			}
			if !stringContains(backend.Notes, "No real provider-backed retention watermark is wired yet") {
				t.Fatalf("contract-only backend notes drifted: %+v", backend.Notes)
			}
			break
		}
		if backend.RuntimeReadiness == "live_proven" && backend.Backend == "sqlite" {
			if !backend.FailClosedExpiry {
				t.Fatalf("sqlite backend should fail close on expiry: %+v", backend)
			}
			if !containsString(backend.SourceReportLinks, "bigclaw-go/docs/reports/broker-retention-boundary-proof-summary.json") {
				t.Fatalf("sqlite backend missing retention proof link: %+v", backend.SourceReportLinks)
			}
		}
	}
	if !contractBackendFound {
		t.Fatal("retention surface missing contract_only backend entry")
	}

	if !stringContains(surface.PolicySplit, "Retention policy advances oldest/newest retained replay boundaries") ||
		!stringContains(surface.PolicySplit, "Expired checkpoints fail closed") {
		t.Fatalf("policy split drifted: %+v", surface.PolicySplit)
	}
}

func stringContains(values []string, needle string) bool {
	for _, value := range values {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
