package regression

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCrossProcessCoordinationReadinessDocsStayAligned(t *testing.T) {
	repoRoot := repoRoot(t)

	reportPath := filepath.Join(repoRoot, "docs", "reports", "cross-process-coordination-capability-surface.json")
	var report struct {
		RuntimeReadinessLevels       map[string]string `json:"runtime_readiness_levels"`
		TargetContractSurfaceVersion string            `json:"target_contract_surface_version"`
		TargetContracts              []struct {
			Capability     string `json:"capability"`
			ContractAnchor string `json:"contract_anchor"`
			RuntimeStatus  string `json:"runtime_status"`
		} `json:"target_contracts"`
		Capabilities []struct {
			Capability       string `json:"capability"`
			RuntimeReadiness string `json:"runtime_readiness"`
		} `json:"capabilities"`
	}
	readJSONFile(t, reportPath, &report)

	for _, readiness := range []string{"live_proven", "harness_proven", "contract_only", "supporting_surface"} {
		if report.RuntimeReadinessLevels[readiness] == "" {
			t.Fatalf("capability surface missing runtime_readiness_levels[%q]", readiness)
		}
	}
	if report.TargetContractSurfaceVersion == "" {
		t.Fatalf("capability surface missing target_contract_surface_version")
	}

	expectedCapabilities := map[string]string{
		"shared_queue_task_coordination":     "live_proven",
		"subscriber_takeover_semantics":      "live_proven",
		"cross_process_replay_coordination":  "harness_proven",
		"stale_writer_fencing":               "live_proven",
		"partitioned_topic_routing":          "contract_only",
		"broker_backed_subscriber_ownership": "contract_only",
		"operator_capability_surface":        "supporting_surface",
	}
	actualCapabilities := map[string]string{}
	for _, capability := range report.Capabilities {
		actualCapabilities[capability.Capability] = capability.RuntimeReadiness
	}
	for capability, readiness := range expectedCapabilities {
		if actualCapabilities[capability] != readiness {
			t.Fatalf("capability %s runtime_readiness = %q, want %q", capability, actualCapabilities[capability], readiness)
		}
	}

	expectedTargets := map[string]string{
		"partitioned_topic_routing":          "events.SubscriptionRequest.PartitionRoute",
		"broker_backed_subscriber_ownership": "events.SubscriptionRequest.OwnershipContract",
	}
	actualTargets := map[string]string{}
	for _, capability := range report.TargetContracts {
		if capability.RuntimeStatus != "contract_only" {
			t.Fatalf("target contract %s runtime_status = %q, want contract_only", capability.Capability, capability.RuntimeStatus)
		}
		actualTargets[capability.Capability] = capability.ContractAnchor
	}
	for capability, anchor := range expectedTargets {
		if actualTargets[capability] != anchor {
			t.Fatalf("target contract %s anchor = %q, want %q", capability, actualTargets[capability], anchor)
		}
	}

	cases := []struct {
		path       string
		substrings []string
	}{
		{
			path: "docs/reports/cross-process-coordination-boundary-digest.md",
			substrings: []string{
				"OPE-261` / `BIG-PAR-085",
				"OPE-257` / `BIG-PAR-095",
				"runtime capability matrix",
				"`live_proven`, `harness_proven`, and `contract_only`",
				"`PartitionRoute`",
				"`SubscriberOwnershipContract`",
			},
		},
		{
			path: "docs/reports/broker-event-log-adapter-contract.md",
			substrings: []string{
				"OPE-257` / `BIG-PAR-095",
				"`PartitionRoute`",
				"`SubscriberOwnershipContract`",
				"contract-only",
			},
		},
		{
			path: "docs/reports/multi-node-coordination-report.md",
			substrings: []string{
				"runtime capability matrix",
				"`live_proven` shared-queue proof",
				"dedicated coordinator leader-election lease",
				"`lease_token`",
			},
		},
		{
			path: "docs/reports/review-readiness.md",
			substrings: []string{
				"runtime capability matrix",
				"`live_proven`, `harness_proven`, and `contract_only`",
			},
		},
		{
			path: "docs/reports/issue-coverage.md",
			substrings: []string{
				"runtime capability matrix",
				"`live_proven`, `harness_proven`, and `contract_only`",
			},
		},
		{
			path: "../docs/openclaw-parallel-gap-analysis.md",
			substrings: []string{
				"OPE-261` / `BIG-PAR-085",
				"OPE-257` / `BIG-PAR-095",
				"runtime capability matrix",
			},
		},
	}

	for _, tc := range cases {
		contents := readRepoFile(t, repoRoot, tc.path)
		for _, needle := range tc.substrings {
			if !strings.Contains(contents, needle) {
				t.Fatalf("%s missing substring %q", tc.path, needle)
			}
		}
	}
}

func readJSONFile(t *testing.T, path string, dst any) {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(contents, dst); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
}
