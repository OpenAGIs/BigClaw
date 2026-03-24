package policy

import (
	"fmt"
	"strconv"
	"strings"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/risk"
)

type Quota struct {
	ConcurrentLimit     int   `json:"concurrent_limit"`
	QueueDepthLimit     int   `json:"queue_depth_limit"`
	BudgetCapCents      int64 `json:"budget_cap_cents"`
	MaxAgents           int   `json:"max_agents"`
	BrowserSessionLimit int   `json:"browser_session_limit"`
	VMSessionLimit      int   `json:"vm_session_limit"`
}

type Summary struct {
	Plan                  string   `json:"plan"`
	DedicatedQueue        string   `json:"dedicated_queue"`
	ConcurrencyProfile    string   `json:"concurrency_profile"`
	AdvancedApproval      bool     `json:"advanced_approval"`
	MultiAgentGraph       bool     `json:"multi_agent_graph"`
	DedicatedBrowserPool  bool     `json:"dedicated_browser_pool"`
	DedicatedVMPool       bool     `json:"dedicated_vm_pool"`
	BrowserPoolAccess     bool     `json:"browser_pool_access"`
	VMPoolAccess          bool     `json:"vm_pool_access"`
	Isolation             string   `json:"isolation"`
	TenantIsolationMode   string   `json:"tenant_isolation_mode"`
	TenantMetadataKeys    []string `json:"tenant_metadata_keys,omitempty"`
	OwnerMatchingRequired bool     `json:"owner_matching_required"`
	OwnerMetadataKeys     []string `json:"owner_metadata_keys,omitempty"`
	ApprovalFlow          string   `json:"approval_flow"`
	ResourcePool          string   `json:"resource_pool"`
	Reason                string   `json:"reason"`
	Quota                 Quota    `json:"quota"`
}

func Resolve(task domain.Task) Summary {
	riskScore := risk.ScoreTask(task, nil)
	premium := false
	reason := "default shared orchestration"
	switch {
	case metadataEquals(task, "plan", "premium"), metadataEquals(task, "tier", "premium"), metadataEquals(task, "orchestration", "premium"), hasLabel(task, "premium"):
		premium = true
		reason = "metadata requested premium orchestration"
	case riskScore.Level == domain.RiskHigh && (requiresTool(task, "browser") || requiresTool(task, "gpu") || requiresTool(task, "vm")):
		premium = true
		reason = "high-risk tool workload promoted to premium lane"
	}

	if premium {
		isolation := firstNonEmpty(metadataString(task, "policy_isolation"), "dedicated")
		browserAccess := true
		vmAccess := true
		approvalFlowDefault := "advanced"
		if riskScore.RequiresApproval {
			approvalFlowDefault = "risk-reviewed"
		}
		return Summary{
			Plan:                  "premium",
			DedicatedQueue:        queueName(task, "premium"),
			ConcurrencyProfile:    firstNonEmpty(metadataString(task, "policy_concurrency_profile"), "elevated"),
			AdvancedApproval:      metadataBool(task, true, "policy_advanced_approval"),
			MultiAgentGraph:       metadataBool(task, true, "policy_multi_agent_graph"),
			DedicatedBrowserPool:  browserAccess && requiresTool(task, "browser"),
			DedicatedVMPool:       vmAccess && (requiresTool(task, "vm") || task.RequiredExecutor == domain.ExecutorKubernetes || task.RequiredExecutor == domain.ExecutorRay),
			BrowserPoolAccess:     browserAccess,
			VMPoolAccess:          vmAccess,
			Isolation:             isolation,
			TenantIsolationMode:   resolveTenantIsolationMode(task, isolation),
			TenantMetadataKeys:    resolveTenantMetadataKeys(task),
			OwnerMatchingRequired: metadataBool(task, false, "policy_require_owner_match", "policy_owner_match"),
			OwnerMetadataKeys:     resolveOwnerMetadataKeys(task),
			ApprovalFlow:          firstNonEmpty(metadataString(task, "policy_approval_flow"), approvalFlowDefault),
			ResourcePool:          firstNonEmpty(metadataString(task, "policy_resource_pool"), queueName(task, "premium")),
			Reason:                reason,
			Quota: quotaForPlan(task, planDefaults{
				ConcurrentLimit:     32,
				QueueDepthLimit:     256,
				BudgetCapCents:      50000,
				MaxAgents:           8,
				BrowserSessionLimit: 4,
				VMSessionLimit:      2,
			}),
		}
	}

	approvalFlowDefault := "standard"
	if riskScore.RequiresApproval {
		approvalFlowDefault = "risk-reviewed"
		reason = firstNonEmpty(reason, "default shared orchestration") + "; risk score requires approval"
	}
	isolation := firstNonEmpty(metadataString(task, "policy_isolation"), "shared")
	return Summary{
		Plan:                  "standard",
		DedicatedQueue:        queueName(task, "shared"),
		ConcurrencyProfile:    firstNonEmpty(metadataString(task, "policy_concurrency_profile"), "shared"),
		AdvancedApproval:      metadataBool(task, riskScore.RequiresApproval, "policy_advanced_approval"),
		MultiAgentGraph:       metadataBool(task, false, "policy_multi_agent_graph"),
		DedicatedBrowserPool:  false,
		DedicatedVMPool:       false,
		BrowserPoolAccess:     false,
		VMPoolAccess:          false,
		Isolation:             isolation,
		TenantIsolationMode:   resolveTenantIsolationMode(task, isolation),
		TenantMetadataKeys:    resolveTenantMetadataKeys(task),
		OwnerMatchingRequired: metadataBool(task, false, "policy_require_owner_match", "policy_owner_match"),
		OwnerMetadataKeys:     resolveOwnerMetadataKeys(task),
		ApprovalFlow:          firstNonEmpty(metadataString(task, "policy_approval_flow"), approvalFlowDefault),
		ResourcePool:          firstNonEmpty(metadataString(task, "policy_resource_pool"), queueName(task, "shared")),
		Reason:                reason,
		Quota: quotaForPlan(task, planDefaults{
			ConcurrentLimit:     8,
			QueueDepthLimit:     64,
			BudgetCapCents:      10000,
			MaxAgents:           2,
			BrowserSessionLimit: 0,
			VMSessionLimit:      0,
		}),
	}
}

type planDefaults struct {
	ConcurrentLimit     int
	QueueDepthLimit     int
	BudgetCapCents      int64
	MaxAgents           int
	BrowserSessionLimit int
	VMSessionLimit      int
}

func quotaForPlan(task domain.Task, defaults planDefaults) Quota {
	return Quota{
		ConcurrentLimit:     metadataInt(task, defaults.ConcurrentLimit, "policy_concurrency_limit", "concurrency_limit"),
		QueueDepthLimit:     metadataInt(task, defaults.QueueDepthLimit, "policy_queue_depth_limit", "queue_depth_limit"),
		BudgetCapCents:      metadataInt64(task, defaults.BudgetCapCents, "policy_budget_cap_cents", "budget_cap_cents"),
		MaxAgents:           metadataInt(task, defaults.MaxAgents, "policy_max_agents", "max_agents"),
		BrowserSessionLimit: metadataInt(task, defaults.BrowserSessionLimit, "policy_browser_session_limit", "browser_session_limit"),
		VMSessionLimit:      metadataInt(task, defaults.VMSessionLimit, "policy_vm_session_limit", "vm_session_limit"),
	}
}

func queueName(task domain.Task, lane string) string {
	team := strings.TrimSpace(task.Metadata["team"])
	if team == "" {
		team = strings.TrimSpace(task.TenantID)
	}
	if team == "" {
		return lane + "/default"
	}
	return fmt.Sprintf("%s/%s", lane, team)
}

func metadataEquals(task domain.Task, key, want string) bool {
	return strings.EqualFold(strings.TrimSpace(task.Metadata[key]), want)
}

func requiresTool(task domain.Task, tool string) bool {
	for _, item := range task.RequiredTools {
		if strings.EqualFold(item, tool) {
			return true
		}
	}
	return false
}

func hasLabel(task domain.Task, label string) bool {
	for _, item := range task.Labels {
		if strings.EqualFold(item, label) {
			return true
		}
	}
	return false
}

func metadataString(task domain.Task, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(task.Metadata[key]); value != "" {
			return value
		}
	}
	return ""
}

func metadataInt(task domain.Task, fallback int, keys ...string) int {
	for _, key := range keys {
		if value := strings.TrimSpace(task.Metadata[key]); value != "" {
			if parsed, err := strconv.Atoi(value); err == nil {
				return parsed
			}
		}
	}
	return fallback
}

func metadataInt64(task domain.Task, fallback int64, keys ...string) int64 {
	for _, key := range keys {
		if value := strings.TrimSpace(task.Metadata[key]); value != "" {
			if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
				return parsed
			}
		}
	}
	return fallback
}

func metadataBool(task domain.Task, fallback bool, keys ...string) bool {
	for _, key := range keys {
		if value := strings.TrimSpace(task.Metadata[key]); value != "" {
			if parsed, err := strconv.ParseBool(value); err == nil {
				return parsed
			}
		}
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func resolveTenantIsolationMode(task domain.Task, isolation string) string {
	switch strings.ToLower(strings.TrimSpace(metadataString(task, "policy_tenant_isolation_mode", "policy_tenant_mode"))) {
	case "tenant", "strict":
		return "tenant"
	case "shared":
		return "shared"
	}
	if strings.EqualFold(strings.TrimSpace(isolation), "dedicated") {
		return "tenant"
	}
	return "shared"
}

func resolveOwnerMetadataKeys(task domain.Task) []string {
	return resolveMetadataKeyList(task, "policy_owner_metadata_keys")
}

func resolveTenantMetadataKeys(task domain.Task) []string {
	return resolveMetadataKeyList(task, "policy_tenant_metadata_keys")
}

func resolveMetadataKeyList(task domain.Task, key string) []string {
	raw := strings.TrimSpace(metadataString(task, key))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	seen := make(map[string]struct{}, len(parts))
	keys := make([]string, 0, len(parts))
	for _, part := range parts {
		key := strings.TrimSpace(part)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return nil
	}
	return keys
}
