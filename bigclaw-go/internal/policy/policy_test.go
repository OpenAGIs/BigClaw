package policy

import (
	"reflect"
	"testing"

	"bigclaw-go/internal/domain"
)

func TestEnforceValidationReportPolicyBlocksIssueCloseWithoutRequiredReports(t *testing.T) {
	decision := EnforceValidationReportPolicy([]string{"task-run", "replay"})

	if decision.AllowedToClose || decision.Status != "blocked" {
		t.Fatalf("expected blocked decision, got %+v", decision)
	}
	if !reflect.DeepEqual(decision.MissingReports, []string{"benchmark-suite"}) {
		t.Fatalf("expected benchmark-suite missing, got %+v", decision.MissingReports)
	}
}

func TestEnforceValidationReportPolicyAllowsIssueCloseWhenReportsComplete(t *testing.T) {
	decision := EnforceValidationReportPolicy([]string{"task-run", "replay", "benchmark-suite"})

	if !decision.AllowedToClose || decision.Status != "ready" {
		t.Fatalf("expected ready decision, got %+v", decision)
	}
	if len(decision.MissingReports) != 0 {
		t.Fatalf("expected no missing reports, got %+v", decision.MissingReports)
	}
}

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

func TestResolveClawHostTenantPolicyAlignedDefaults(t *testing.T) {
	resolved, ok := ResolveClawHostTenantPolicy(domain.Task{
		ID:       "clawhost-policy-1",
		Source:   "clawhost",
		TenantID: "tenant-a",
		Labels:   []string{"integration"},
		Metadata: map[string]string{
			"clawhost_app_id":           "sales-app",
			"clawhost_bot_group":        "sales-bots",
			"clawhost_default_provider": "openai",
			"clawhost_default_model":    "gpt-5.4",
			"clawhost_approval_flow":    "standard",
		},
	})
	if !ok {
		t.Fatal("expected ClawHost task to resolve")
	}
	if resolved.DriftStatus != "aligned" || resolved.AppID != "sales-app" || resolved.BotGroup != "sales-bots" || resolved.ProviderDefault != "openai" {
		t.Fatalf("unexpected aligned ClawHost policy: %+v", resolved)
	}
	if resolved.ManualReviewRequired || resolved.TakeoverRequired {
		t.Fatalf("expected no manual review or takeover requirement, got %+v", resolved)
	}
}

func TestResolveClawHostTenantPolicyOutOfPolicyOverride(t *testing.T) {
	resolved, ok := ResolveClawHostTenantPolicy(domain.Task{
		ID:       "clawhost-policy-2",
		Labels:   []string{"clawhost"},
		TenantID: "tenant-b",
		Metadata: map[string]string{
			"clawhost_app_id":             "ops-app",
			"clawhost_default_provider":   "anthropic",
			"clawhost_provider_mode":      "tenant_override",
			"clawhost_provider_allowlist": "openai,google",
			"clawhost_lock_app_defaults":  "true",
			"clawhost_takeover_required":  "true",
		},
	})
	if !ok {
		t.Fatal("expected ClawHost label to resolve")
	}
	if resolved.DriftStatus != "out_of_policy" {
		t.Fatalf("expected out_of_policy drift, got %+v", resolved)
	}
	if !resolved.ManualReviewRequired || !resolved.TakeoverRequired {
		t.Fatalf("expected manual review and takeover requirement, got %+v", resolved)
	}
	if resolved.Reason == "" {
		t.Fatalf("expected a review reason, got %+v", resolved)
	}
}
