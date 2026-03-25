package api

import (
	"encoding/json"
	"fmt"
	"os"
)

const clawHostTenantPolicySurfacePath = "docs/reports/clawhost-tenant-policy-surface.json"

type clawHostTenantPolicySurface struct {
	ReportPath      string                           `json:"report_path"`
	GeneratedAt     string                           `json:"generated_at,omitempty"`
	Ticket          string                           `json:"ticket,omitempty"`
	Title           string                           `json:"title,omitempty"`
	Status          string                           `json:"status,omitempty"`
	Provider        string                           `json:"provider,omitempty"`
	PolicyMode      string                           `json:"policy_mode,omitempty"`
	EvidenceSources []string                         `json:"evidence_sources,omitempty"`
	ReviewerLinks   []string                         `json:"reviewer_links,omitempty"`
	Summary         clawHostTenantPolicySummary      `json:"summary"`
	Tenants         []clawHostTenantPolicyTenant     `json:"tenants,omitempty"`
	AppDefaults     []clawHostTenantPolicyAppDefault `json:"app_defaults,omitempty"`
	Limitations     []string                         `json:"limitations,omitempty"`
	Error           string                           `json:"error,omitempty"`
}

type clawHostTenantPolicySummary struct {
	TenantCount            int `json:"tenant_count"`
	AppDefaultCount        int `json:"app_default_count"`
	MultiProviderTenants   int `json:"multi_provider_tenants"`
	EntitlementGuardrails  int `json:"entitlement_guardrails"`
	RolloutBlockedDefaults int `json:"rollout_blocked_defaults"`
	ReviewerReadyTenants   int `json:"reviewer_ready_tenants"`
}

type clawHostTenantPolicyTenant struct {
	Tenant                string   `json:"tenant"`
	DefaultProvider       string   `json:"default_provider,omitempty"`
	AllowedProviders      []string `json:"allowed_providers,omitempty"`
	EntitlementPolicy     string   `json:"entitlement_policy,omitempty"`
	ApprovalMode          string   `json:"approval_mode,omitempty"`
	RolloutGuardrail      string   `json:"rollout_guardrail,omitempty"`
	BlockedDefaultChanges int      `json:"blocked_default_changes"`
}

type clawHostTenantPolicyAppDefault struct {
	AppID             string   `json:"app_id"`
	Tenant            string   `json:"tenant,omitempty"`
	Provider          string   `json:"provider,omitempty"`
	Model             string   `json:"model,omitempty"`
	FallbackProviders []string `json:"fallback_providers,omitempty"`
	ApprovalRequired  bool     `json:"approval_required"`
	RolloutStatus     string   `json:"rollout_status,omitempty"`
}

func clawHostTenantPolicySurfacePayload() clawHostTenantPolicySurface {
	surface := clawHostTenantPolicySurface{ReportPath: clawHostTenantPolicySurfacePath}
	reportPath := resolveRepoRelativePath(clawHostTenantPolicySurfacePath)
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
		surface.Error = fmt.Sprintf("decode %s: %v", clawHostTenantPolicySurfacePath, err)
		return surface
	}
	surface.ReportPath = clawHostTenantPolicySurfacePath
	return surface
}
