package policy

import (
	"testing"

	"bigclaw-go/internal/domain"
)

func TestResolvePremiumPolicyFromMetadata(t *testing.T) {
	summary := Resolve(domain.Task{
		TenantID:      "platform",
		RequiredTools: []string{"browser", "vm"},
		Metadata:      map[string]string{"plan": "premium", "team": "platform"},
	})
	if summary.Plan != "premium" || !summary.AdvancedApproval || !summary.MultiAgentGraph {
		t.Fatalf("expected premium policy, got %+v", summary)
	}
	if summary.DedicatedQueue != "premium/platform" {
		t.Fatalf("unexpected dedicated queue, got %+v", summary)
	}
	if !summary.BrowserPoolAccess || !summary.VMPoolAccess || summary.ApprovalFlow != "advanced" || summary.ResourcePool != "premium/platform" {
		t.Fatalf("expected premium capability boundary, got %+v", summary)
	}
	if summary.Quota.ConcurrentLimit != 32 || summary.Quota.QueueDepthLimit != 256 || summary.Quota.BudgetCapCents != 50000 || summary.Quota.MaxAgents != 8 {
		t.Fatalf("expected premium quota defaults, got %+v", summary.Quota)
	}
}

func TestResolveStandardPolicyDefaults(t *testing.T) {
	summary := Resolve(domain.Task{TenantID: "growth", Metadata: map[string]string{"team": "growth"}})
	if summary.Plan != "standard" || summary.AdvancedApproval || summary.MultiAgentGraph {
		t.Fatalf("expected standard policy, got %+v", summary)
	}
	if summary.DedicatedQueue != "shared/growth" || summary.ApprovalFlow != "standard" || summary.ResourcePool != "shared/growth" {
		t.Fatalf("expected standard shared routing, got %+v", summary)
	}
	if summary.Quota.ConcurrentLimit != 8 || summary.Quota.QueueDepthLimit != 64 || summary.Quota.BudgetCapCents != 10000 || summary.Quota.MaxAgents != 2 {
		t.Fatalf("expected standard quota defaults, got %+v", summary.Quota)
	}
}

func TestResolveRiskDrivenApprovalFlow(t *testing.T) {
	summary := Resolve(domain.Task{
		Priority:      1,
		Labels:        []string{"security", "prod"},
		RequiredTools: []string{"deploy"},
		Metadata:      map[string]string{"team": "platform"},
	})
	if summary.Plan != "standard" {
		t.Fatalf("expected standard plan for non-premium task, got %+v", summary)
	}
	if !summary.AdvancedApproval || summary.ApprovalFlow != "risk-reviewed" {
		t.Fatalf("expected risk-driven approval defaults, got %+v", summary)
	}
}

func TestResolvePolicyOverridesQuotaBoundaries(t *testing.T) {
	summary := Resolve(domain.Task{
		RequiredTools: []string{"browser"},
		Metadata: map[string]string{
			"plan":                         "premium",
			"team":                         "platform",
			"policy_concurrency_limit":     "64",
			"policy_queue_depth_limit":     "512",
			"policy_budget_cap_cents":      "250000",
			"policy_max_agents":            "12",
			"policy_browser_session_limit": "8",
			"policy_vm_session_limit":      "4",
			"policy_approval_flow":         "manual-gated",
			"policy_resource_pool":         "premium/gold",
			"policy_multi_agent_graph":     "false",
		},
	})
	if summary.Quota.ConcurrentLimit != 64 || summary.Quota.QueueDepthLimit != 512 || summary.Quota.BudgetCapCents != 250000 || summary.Quota.MaxAgents != 12 {
		t.Fatalf("expected quota overrides, got %+v", summary.Quota)
	}
	if summary.Quota.BrowserSessionLimit != 8 || summary.Quota.VMSessionLimit != 4 {
		t.Fatalf("expected pool overrides, got %+v", summary.Quota)
	}
	if summary.ApprovalFlow != "manual-gated" || summary.ResourcePool != "premium/gold" || summary.MultiAgentGraph {
		t.Fatalf("expected configurable capability boundaries, got %+v", summary)
	}
}
