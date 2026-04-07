package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type coordinationCapabilitySurface struct {
	Ticket                       string `json:"ticket"`
	Title                        string `json:"title"`
	Status                       string `json:"status"`
	TargetContractSurfaceVersion string `json:"target_contract_surface_version"`
	EvidenceInputs               struct {
		SharedQueueReport     string   `json:"shared_queue_report"`
		TakeoverHarnessReport string   `json:"takeover_harness_report"`
		LiveTakeoverReport    string   `json:"live_takeover_report"`
		SupportingDocs        []string `json:"supporting_docs"`
	} `json:"evidence_inputs"`
	Summary struct {
		SharedQueueTotalTasks              int `json:"shared_queue_total_tasks"`
		SharedQueueCrossNodeCompletions    int `json:"shared_queue_cross_node_completions"`
		SharedQueueDuplicateCompletedTasks int `json:"shared_queue_duplicate_completed_tasks"`
		SharedQueueDuplicateStartedTasks   int `json:"shared_queue_duplicate_started_tasks"`
		TakeoverScenarioCount              int `json:"takeover_scenario_count"`
		TakeoverPassingScenarios           int `json:"takeover_passing_scenarios"`
		TakeoverDuplicateDeliveryCount     int `json:"takeover_duplicate_delivery_count"`
		TakeoverStaleWriteRejections       int `json:"takeover_stale_write_rejections"`
		LiveTakeoverScenarioCount          int `json:"live_takeover_scenario_count"`
		LiveTakeoverPassingScenarios       int `json:"live_takeover_passing_scenarios"`
		LiveTakeoverStaleWriteRejections   int `json:"live_takeover_stale_write_rejections"`
	} `json:"summary"`
	TargetContracts []struct {
		Capability     string `json:"capability"`
		ContractAnchor string `json:"contract_anchor"`
		RuntimeStatus  string `json:"runtime_status"`
		Partitioning   struct {
			Topic                  string   `json:"topic"`
			SupportedPartitionKeys []string `json:"supported_partition_keys"`
			OrderingScope          string   `json:"ordering_scope"`
			FilterAlignment        string   `json:"filter_alignment"`
		} `json:"partitioning"`
		Ownership struct {
			SubscriberGroup string   `json:"subscriber_group"`
			Mode            string   `json:"mode"`
			LeaseFields     []string `json:"lease_fields"`
			PartitionHints  string   `json:"partition_hints"`
		} `json:"ownership"`
		Guarantees []string `json:"guarantees"`
	} `json:"target_contracts"`
	Capabilities []struct {
		Capability            string   `json:"capability"`
		CurrentState          string   `json:"current_state"`
		RuntimeReadiness      string   `json:"runtime_readiness"`
		LiveLocalProof        bool     `json:"live_local_proof"`
		DeterministicHarness  bool     `json:"deterministic_local_harness"`
		ContractDefinedTarget bool     `json:"contract_defined_target"`
		Notes                 []string `json:"notes"`
	} `json:"capabilities"`
	CurrentCeiling   []string `json:"current_ceiling"`
	NextRuntimeHooks []string `json:"next_runtime_hooks"`
}

func TestCoordinationCapabilitySurfaceEvidenceAndSummaryStayAligned(t *testing.T) {
	root := repoRoot(t)

	var surface coordinationCapabilitySurface
	readJSONFile(t, filepath.Join(root, "docs", "reports", "cross-process-coordination-capability-surface.json"), &surface)

	if surface.Ticket != "BIG-PAR-085-local-prework" ||
		surface.Status != "local-capability-surface" ||
		surface.TargetContractSurfaceVersion != "2026-03-17" {
		t.Fatalf("unexpected coordination capability surface identity: %+v", surface)
	}
	if surface.EvidenceInputs.SharedQueueReport == "" ||
		surface.EvidenceInputs.TakeoverHarnessReport == "" ||
		surface.EvidenceInputs.LiveTakeoverReport == "" ||
		len(surface.EvidenceInputs.SupportingDocs) != 3 {
		t.Fatalf("unexpected coordination evidence inputs: %+v", surface.EvidenceInputs)
	}
	for _, candidate := range append([]string{
		surface.EvidenceInputs.SharedQueueReport,
		surface.EvidenceInputs.TakeoverHarnessReport,
		surface.EvidenceInputs.LiveTakeoverReport,
	}, surface.EvidenceInputs.SupportingDocs...) {
		if _, err := os.Stat(resolveRepoPath(root, candidate)); err != nil {
			t.Fatalf("expected coordination evidence path %q to exist: %v", candidate, err)
		}
	}

	if surface.Summary.SharedQueueTotalTasks != 200 ||
		surface.Summary.SharedQueueCrossNodeCompletions != 99 ||
		surface.Summary.SharedQueueDuplicateCompletedTasks != 0 ||
		surface.Summary.SharedQueueDuplicateStartedTasks != 0 ||
		surface.Summary.TakeoverScenarioCount != 3 ||
		surface.Summary.TakeoverPassingScenarios != 3 ||
		surface.Summary.TakeoverDuplicateDeliveryCount != 4 ||
		surface.Summary.TakeoverStaleWriteRejections != 2 ||
		surface.Summary.LiveTakeoverScenarioCount != 3 ||
		surface.Summary.LiveTakeoverPassingScenarios != 3 ||
		surface.Summary.LiveTakeoverStaleWriteRejections != 3 {
		t.Fatalf("unexpected coordination summary payload: %+v", surface.Summary)
	}
	if len(surface.CurrentCeiling) != 3 || len(surface.NextRuntimeHooks) != 3 {
		t.Fatalf("unexpected coordination ceiling/hooks payload: ceiling=%+v hooks=%+v", surface.CurrentCeiling, surface.NextRuntimeHooks)
	}
	if !containsCoordinationSnippet(surface.CurrentCeiling, "no partitioned topic model") ||
		!containsCoordinationSnippet(surface.NextRuntimeHooks, "broker-backed replay and ownership semantics") {
		t.Fatalf("unexpected coordination ceiling/hooks detail: ceiling=%+v hooks=%+v", surface.CurrentCeiling, surface.NextRuntimeHooks)
	}
}

func TestCoordinationContractOnlyTargetsStayAligned(t *testing.T) {
	root := repoRoot(t)

	var surface coordinationCapabilitySurface
	readJSONFile(t, filepath.Join(root, "docs", "reports", "cross-process-coordination-capability-surface.json"), &surface)

	if len(surface.TargetContracts) != 2 {
		t.Fatalf("expected two coordination target contracts, got %+v", surface.TargetContracts)
	}

	var partitionRoute, ownershipContract *struct {
		Capability     string `json:"capability"`
		ContractAnchor string `json:"contract_anchor"`
		RuntimeStatus  string `json:"runtime_status"`
		Partitioning   struct {
			Topic                  string   `json:"topic"`
			SupportedPartitionKeys []string `json:"supported_partition_keys"`
			OrderingScope          string   `json:"ordering_scope"`
			FilterAlignment        string   `json:"filter_alignment"`
		} `json:"partitioning"`
		Ownership struct {
			SubscriberGroup string   `json:"subscriber_group"`
			Mode            string   `json:"mode"`
			LeaseFields     []string `json:"lease_fields"`
			PartitionHints  string   `json:"partition_hints"`
		} `json:"ownership"`
		Guarantees []string `json:"guarantees"`
	}
	for i := range surface.TargetContracts {
		contract := surface.TargetContracts[i]
		switch contract.Capability {
		case "partitioned_topic_routing":
			partitionRoute = &surface.TargetContracts[i]
		case "broker_backed_subscriber_ownership":
			ownershipContract = &surface.TargetContracts[i]
		}
	}
	if partitionRoute == nil || ownershipContract == nil {
		t.Fatalf("missing contract-only coordination targets: %+v", surface.TargetContracts)
	}

	if partitionRoute.ContractAnchor != "events.SubscriptionRequest.PartitionRoute" ||
		partitionRoute.RuntimeStatus != "contract_only" ||
		partitionRoute.Partitioning.Topic != "provider-defined shared event stream" ||
		len(partitionRoute.Partitioning.SupportedPartitionKeys) != 3 ||
		partitionRoute.Partitioning.SupportedPartitionKeys[0] != "trace_id" ||
		partitionRoute.Partitioning.SupportedPartitionKeys[2] != "event_type" ||
		!strings.Contains(partitionRoute.Partitioning.OrderingScope, "portable") ||
		!strings.Contains(partitionRoute.Partitioning.FilterAlignment, "ReplayRequest") ||
		len(partitionRoute.Guarantees) != 3 {
		t.Fatalf("unexpected partition route target contract: %+v", partitionRoute)
	}

	if ownershipContract.ContractAnchor != "events.SubscriptionRequest.OwnershipContract" ||
		ownershipContract.RuntimeStatus != "contract_only" ||
		ownershipContract.Ownership.SubscriberGroup != "shared durable consumer identity" ||
		ownershipContract.Ownership.Mode != "exclusive" ||
		len(ownershipContract.Ownership.LeaseFields) != 2 ||
		ownershipContract.Ownership.LeaseFields[0] != "epoch" ||
		ownershipContract.Ownership.LeaseFields[1] != "lease_token" ||
		!strings.Contains(ownershipContract.Ownership.PartitionHints, "partition affinity") ||
		len(ownershipContract.Guarantees) != 3 {
		t.Fatalf("unexpected ownership target contract: %+v", ownershipContract)
	}

	capabilityByName := map[string]coordinationCapabilitySurfaceCapability{}
	for _, capability := range surface.Capabilities {
		capabilityByName[capability.Capability] = coordinationCapabilitySurfaceCapability{
			CurrentState:          capability.CurrentState,
			RuntimeReadiness:      capability.RuntimeReadiness,
			LiveLocalProof:        capability.LiveLocalProof,
			DeterministicHarness:  capability.DeterministicHarness,
			ContractDefinedTarget: capability.ContractDefinedTarget,
			Notes:                 capability.Notes,
		}
	}
	assertContractOnlyCoordinationCapability(t, capabilityByName["partitioned_topic_routing"], "No partitioned topic model exists yet in the runtime.")
	assertContractOnlyCoordinationCapability(t, capabilityByName["broker_backed_subscriber_ownership"], "No broker-backed cross-process subscriber ownership model exists yet.")
}

type coordinationCapabilitySurfaceCapability struct {
	CurrentState          string
	RuntimeReadiness      string
	LiveLocalProof        bool
	DeterministicHarness  bool
	ContractDefinedTarget bool
	Notes                 []string
}

func assertContractOnlyCoordinationCapability(t *testing.T, capability coordinationCapabilitySurfaceCapability, note string) {
	t.Helper()
	if capability.CurrentState != "not_available" ||
		capability.RuntimeReadiness != "contract_only" ||
		capability.LiveLocalProof ||
		capability.DeterministicHarness ||
		!capability.ContractDefinedTarget ||
		len(capability.Notes) != 2 ||
		!containsCoordinationSnippet(capability.Notes, note) {
		t.Fatalf("unexpected contract-only coordination capability: %+v", capability)
	}
}

func containsCoordinationSnippet(values []string, needle string) bool {
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), strings.ToLower(needle)) {
			return true
		}
	}
	return false
}
