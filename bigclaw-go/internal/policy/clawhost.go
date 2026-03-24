package policy

import (
	"sort"
	"strings"

	"bigclaw-go/internal/domain"
)

type ClawHostCatalog struct {
	Integration             string   `json:"integration"`
	ProviderCatalogMode     string   `json:"provider_catalog_mode"`
	OwnershipScopes         []string `json:"ownership_scopes"`
	OverrideScopes          []string `json:"override_scopes"`
	MetadataKeys            []string `json:"metadata_keys"`
	SupportsBringYourOwnKey bool     `json:"supports_bring_your_own_key"`
	ParallelSafe            bool     `json:"parallel_safe"`
}

type ClawHostTenantPolicy struct {
	TaskID                  string   `json:"task_id"`
	TenantID                string   `json:"tenant_id"`
	AppID                   string   `json:"app_id"`
	BotGroup                string   `json:"bot_group"`
	OwnershipMode           string   `json:"ownership_mode"`
	Entitlement             string   `json:"entitlement"`
	ProviderDefault         string   `json:"provider_default"`
	ModelDefault            string   `json:"model_default,omitempty"`
	ProviderMode            string   `json:"provider_mode"`
	AllowedProviders        []string `json:"allowed_providers,omitempty"`
	BlockedProviders        []string `json:"blocked_providers,omitempty"`
	ApprovalFlow            string   `json:"approval_flow"`
	RolloutLane             string   `json:"rollout_lane"`
	SupportsBringYourOwnKey bool     `json:"supports_bring_your_own_key"`
	AppDefaultLocked        bool     `json:"app_default_locked"`
	ManualReviewRequired    bool     `json:"manual_review_required"`
	TakeoverRequired        bool     `json:"takeover_required"`
	DriftStatus             string   `json:"drift_status"`
	Reason                  string   `json:"reason"`
}

func ClawHostCatalogInfo() ClawHostCatalog {
	return ClawHostCatalog{
		Integration:             "clawhost",
		ProviderCatalogMode:     "dashboard_configured",
		OwnershipScopes:         []string{"app", "tenant", "bot_group"},
		OverrideScopes:          []string{"app_default", "tenant_override", "bot_group_override"},
		MetadataKeys:            clawHostMetadataKeys(),
		SupportsBringYourOwnKey: true,
		ParallelSafe:            true,
	}
}

func ResolveClawHostTenantPolicy(task domain.Task) (ClawHostTenantPolicy, bool) {
	if !isClawHostTask(task) {
		return ClawHostTenantPolicy{}, false
	}
	policy := ClawHostTenantPolicy{
		TaskID:                  task.ID,
		TenantID:                firstNonEmpty(metadataString(task, "clawhost_tenant_id", "clawhost_tenant", "tenant"), strings.TrimSpace(task.TenantID), "unassigned"),
		AppID:                   firstNonEmpty(metadataString(task, "clawhost_app_id", "clawhost_app", "app_id", "app"), "unassigned"),
		BotGroup:                firstNonEmpty(metadataString(task, "clawhost_bot_group", "clawhost_bot_group_id", "bot_group"), "all-bots"),
		OwnershipMode:           firstNonEmpty(metadataString(task, "clawhost_ownership_mode"), "app_owner"),
		Entitlement:             firstNonEmpty(metadataString(task, "clawhost_entitlement", "plan", "tier"), "standard"),
		ProviderDefault:         firstNonEmpty(metadataString(task, "clawhost_default_provider", "clawhost_provider", "provider"), "dashboard-default"),
		ModelDefault:            metadataString(task, "clawhost_default_model", "clawhost_model", "model"),
		ProviderMode:            firstNonEmpty(metadataString(task, "clawhost_provider_mode"), "app_default"),
		AllowedProviders:        metadataCSV(task, "clawhost_allowed_providers", "clawhost_provider_allowlist"),
		BlockedProviders:        metadataCSV(task, "clawhost_blocked_providers", "clawhost_provider_blocklist"),
		ApprovalFlow:            firstNonEmpty(metadataString(task, "clawhost_approval_flow", "policy_approval_flow"), "manual-review"),
		RolloutLane:             firstNonEmpty(metadataString(task, "clawhost_rollout_lane", "policy_concurrency_profile"), "tenant_wave"),
		SupportsBringYourOwnKey: metadataBool(task, true, "clawhost_byok_allowed"),
		AppDefaultLocked:        metadataBool(task, false, "clawhost_lock_app_defaults"),
		TakeoverRequired:        metadataBool(task, false, "clawhost_takeover_required"),
	}
	policy.ManualReviewRequired = metadataBool(task, false, "clawhost_manual_review_required")
	if !policy.ManualReviewRequired && (policy.ProviderMode != "app_default" || policy.ApprovalFlow != "standard" || policy.AppDefaultLocked) {
		policy.ManualReviewRequired = true
	}
	policy.DriftStatus = clawHostDriftStatus(policy)
	policy.Reason = clawHostReason(policy)
	return policy, true
}

func isClawHostTask(task domain.Task) bool {
	if strings.EqualFold(strings.TrimSpace(task.Source), "clawhost") {
		return true
	}
	if metadataEquals(task, "integration", "clawhost") || metadataEquals(task, "connector", "clawhost") || metadataEquals(task, "control_plane", "clawhost") {
		return true
	}
	return hasLabel(task, "clawhost")
}

func clawHostDriftStatus(policy ClawHostTenantPolicy) string {
	switch {
	case containsFold(policy.BlockedProviders, policy.ProviderDefault):
		return "blocked"
	case len(policy.AllowedProviders) > 0 && !containsFold(policy.AllowedProviders, policy.ProviderDefault):
		return "out_of_policy"
	case policy.ManualReviewRequired || policy.TakeoverRequired:
		return "review_required"
	default:
		return "aligned"
	}
}

func clawHostReason(policy ClawHostTenantPolicy) string {
	reasons := make([]string, 0, 4)
	if containsFold(policy.BlockedProviders, policy.ProviderDefault) {
		reasons = append(reasons, "provider is explicitly blocked for the tenant policy")
	}
	if len(policy.AllowedProviders) > 0 && !containsFold(policy.AllowedProviders, policy.ProviderDefault) {
		reasons = append(reasons, "provider default falls outside the tenant allowlist")
	}
	if policy.ProviderMode != "app_default" {
		reasons = append(reasons, "provider settings override the shared app default")
	}
	if policy.AppDefaultLocked {
		reasons = append(reasons, "app defaults are locked and require policy review")
	}
	if policy.TakeoverRequired {
		reasons = append(reasons, "human takeover is required before rollout")
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "provider default remains aligned with the shared app policy")
	}
	return strings.Join(reasons, "; ")
}

func metadataCSV(task domain.Task, keys ...string) []string {
	seen := map[string]struct{}{}
	values := make([]string, 0, 4)
	for _, key := range keys {
		raw := metadataString(task, key)
		if raw == "" {
			continue
		}
		for _, item := range strings.Split(raw, ",") {
			value := lowerTrimClawHost(item)
			if value == "" {
				continue
			}
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			values = append(values, value)
		}
		if len(values) > 0 {
			break
		}
	}
	sort.Strings(values)
	return values
}

func containsFold(values []string, want string) bool {
	want = lowerTrimClawHost(want)
	for _, value := range values {
		if lowerTrimClawHost(value) == want {
			return true
		}
	}
	return false
}

func lowerTrimClawHost(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func clawHostMetadataKeys() []string {
	return []string{
		"clawhost_allowed_providers",
		"clawhost_app",
		"clawhost_app_id",
		"clawhost_approval_flow",
		"clawhost_blocked_providers",
		"clawhost_bot_group",
		"clawhost_byok_allowed",
		"clawhost_default_model",
		"clawhost_default_provider",
		"clawhost_entitlement",
		"clawhost_lock_app_defaults",
		"clawhost_manual_review_required",
		"clawhost_ownership_mode",
		"clawhost_provider_allowlist",
		"clawhost_provider_blocklist",
		"clawhost_provider_mode",
		"clawhost_rollout_lane",
		"clawhost_takeover_required",
		"clawhost_tenant",
		"clawhost_tenant_id",
		"integration",
	}
}
