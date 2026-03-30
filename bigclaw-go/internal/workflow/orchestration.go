package workflow

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"bigclaw-go/internal/domain"
)

type DepartmentHandoff struct {
	Department    string   `json:"department"`
	Reason        string   `json:"reason"`
	RequiredTools []string `json:"required_tools,omitempty"`
	Approvals     []string `json:"approvals,omitempty"`
}

type OrchestrationPlan struct {
	TaskID            string              `json:"task_id"`
	CollaborationMode string              `json:"collaboration_mode"`
	Handoffs          []DepartmentHandoff `json:"handoffs,omitempty"`
}

type HandoffRequest struct {
	TargetTeam        string   `json:"target_team"`
	Reason            string   `json:"reason"`
	Status            string   `json:"status"`
	RequiredApprovals []string `json:"required_approvals,omitempty"`
}

type OrchestrationPolicyDecision struct {
	Tier               string   `json:"tier"`
	UpgradeRequired    bool     `json:"upgrade_required"`
	Reason             string   `json:"reason"`
	BlockedDepartments []string `json:"blocked_departments,omitempty"`
	EntitlementStatus  string   `json:"entitlement_status"`
	BillingModel       string   `json:"billing_model"`
	EstimatedCostUSD   float64  `json:"estimated_cost_usd"`
	IncludedUsageUnits int      `json:"included_usage_units"`
	OverageUsageUnits  int      `json:"overage_usage_units"`
	OverageCostUSD     float64  `json:"overage_cost_usd"`
}

type CrossDepartmentOrchestrator struct{}

type PremiumOrchestrationPolicy struct{}

func (p OrchestrationPlan) Departments() []string {
	departments := make([]string, 0, len(p.Handoffs))
	for _, handoff := range p.Handoffs {
		departments = append(departments, handoff.Department)
	}
	return departments
}

func (p OrchestrationPlan) DepartmentCount() int {
	return len(p.Handoffs)
}

func (p OrchestrationPlan) RequiredApprovals() []string {
	approvals := make([]string, 0)
	for _, handoff := range p.Handoffs {
		for _, approval := range handoff.Approvals {
			approval = strings.TrimSpace(approval)
			if approval != "" && !slices.Contains(approvals, approval) {
				approvals = append(approvals, approval)
			}
		}
	}
	return approvals
}

func (o CrossDepartmentOrchestrator) Plan(task domain.Task) OrchestrationPlan {
	labels := normalizeStrings(task.Labels)
	tools := normalizeStrings(task.RequiredTools)
	textParts := []string{strings.ToLower(task.Title), strings.ToLower(task.Description)}
	for _, item := range task.AcceptanceCriteria {
		textParts = append(textParts, strings.ToLower(strings.TrimSpace(item)))
	}
	for _, item := range task.ValidationPlan {
		textParts = append(textParts, strings.ToLower(strings.TrimSpace(item)))
	}
	text := strings.Join(textParts, " ")

	handoffs := make([]DepartmentHandoff, 0)
	appendUniqueHandoff(&handoffs, "operations", operationsReason(task, labels, text), nil, nil)

	if len(tools) > 0 || strings.Contains(strings.ToLower(task.Source), "github") || hasAny(tools, "repo", "browser", "terminal") {
		appendUniqueHandoff(&handoffs, "engineering", "implements automation and tool-driven execution", tools, nil)
	}
	if task.RiskLevel == domain.RiskHigh || hasAny(labels, "security", "compliance") || strings.Contains(text, "approval") {
		approvals := []string(nil)
		if task.RiskLevel == domain.RiskHigh {
			approvals = []string{"security-review"}
		}
		appendUniqueHandoff(&handoffs, "security", "reviews elevated risk, compliance, or approval-sensitive work", nil, approvals)
	}
	if hasAny(labels, "data", "analytics") || hasAny(tools, "sql", "warehouse", "bi") {
		appendUniqueHandoff(&handoffs, "data", "validates analytics, warehouse, or measurement dependencies", intersect(tools, []string{"bi", "sql", "warehouse"}), nil)
	}
	if hasAny(labels, "customer", "support", "success") || strings.Contains(text, "customer") || strings.Contains(text, "stakeholder") {
		appendUniqueHandoff(&handoffs, "customer-success", "coordinates customer communication and rollout readiness", nil, nil)
	}

	mode := "single-team"
	if len(handoffs) > 1 {
		mode = "cross-functional"
	}
	return OrchestrationPlan{TaskID: task.ID, CollaborationMode: mode, Handoffs: handoffs}
}

func (p PremiumOrchestrationPolicy) Apply(task domain.Task, plan OrchestrationPlan) (OrchestrationPlan, OrchestrationPolicyDecision) {
	requestedUnits := max(1, plan.DepartmentCount())
	if isPremiumTask(task) {
		estimated := estimateCost(requestedUnits)
		return plan, OrchestrationPolicyDecision{
			Tier:               "premium",
			UpgradeRequired:    false,
			Reason:             "premium tier enables advanced cross-department orchestration",
			EntitlementStatus:  "included",
			BillingModel:       "premium-included",
			EstimatedCostUSD:   estimated,
			IncludedUsageUnits: requestedUnits,
		}
	}

	blocked := make([]string, 0)
	allowedHandoffs := make([]DepartmentHandoff, 0, len(plan.Handoffs))
	for _, handoff := range plan.Handoffs {
		switch handoff.Department {
		case "operations", "engineering":
			allowedHandoffs = append(allowedHandoffs, handoff)
		default:
			blocked = append(blocked, handoff.Department)
		}
	}
	if len(blocked) == 0 {
		estimated := estimateCost(requestedUnits)
		return plan, OrchestrationPolicyDecision{
			Tier:               "standard",
			UpgradeRequired:    false,
			Reason:             "standard tier supports baseline orchestration",
			EntitlementStatus:  "included",
			BillingModel:       "standard-included",
			EstimatedCostUSD:   estimated,
			IncludedUsageUnits: requestedUnits,
		}
	}

	constrained := OrchestrationPlan{
		TaskID:            plan.TaskID,
		CollaborationMode: "tier-limited",
		Handoffs:          allowedHandoffs,
	}
	included := max(1, constrained.DepartmentCount())
	overage := len(blocked)
	overageCost := estimateOverageCost(overage)
	return constrained, OrchestrationPolicyDecision{
		Tier:               "standard",
		UpgradeRequired:    true,
		Reason:             "premium tier required for advanced cross-department orchestration",
		BlockedDepartments: blocked,
		EntitlementStatus:  "upgrade-required",
		BillingModel:       "standard-blocked",
		EstimatedCostUSD:   estimateCost(included) + overageCost,
		IncludedUsageUnits: included,
		OverageUsageUnits:  overage,
		OverageCostUSD:     overageCost,
	}
}

func BuildHandoffRequest(accepted bool, plan OrchestrationPlan, policyDecision OrchestrationPolicyDecision) *HandoffRequest {
	if accepted || plan.DepartmentCount() <= 1 {
		return nil
	}
	status := "pending"
	if policyDecision.UpgradeRequired {
		status = "blocked"
	}
	targetTeam := "operations"
	if len(plan.Handoffs) > 1 {
		targetTeam = plan.Handoffs[len(plan.Handoffs)-1].Department
	}
	reason := fmt.Sprintf("%d departments require orchestration", plan.DepartmentCount())
	if policyDecision.Reason != "" {
		reason = policyDecision.Reason
	}
	return &HandoffRequest{
		TargetTeam:        targetTeam,
		Reason:            reason,
		Status:            status,
		RequiredApprovals: plan.RequiredApprovals(),
	}
}

func RenderOrchestrationPlan(plan OrchestrationPlan, policyDecision OrchestrationPolicyDecision) string {
	lines := []string{
		"# Cross-Department Orchestration Plan",
		"",
		"- Task ID: " + plan.TaskID,
		"- Collaboration Mode: " + plan.CollaborationMode,
		"- Departments: " + joinOrNone(plan.Departments()),
		"- Required Approvals: " + joinOrNone(plan.RequiredApprovals()),
		"- Tier: " + policyDecision.Tier,
		"- Entitlement Status: " + policyDecision.EntitlementStatus,
		"- Billing Model: " + policyDecision.BillingModel,
		fmt.Sprintf("- Estimated Cost (USD): %.2f", policyDecision.EstimatedCostUSD),
		"- Blocked Departments: " + joinOrNone(policyDecision.BlockedDepartments),
	}
	return strings.Join(lines, "\n") + "\n"
}

func operationsReason(task domain.Task, labels []string, text string) string {
	if hasAny(labels, "program", "ops", "release") || strings.Contains(text, "rollout") || matchesAny(strings.ToLower(task.Source), "linear", "jira") {
		return "coordinates issue intake, handoffs, and completion tracking"
	}
	return "owns task intake and delivery coordination"
}

func appendUniqueHandoff(handoffs *[]DepartmentHandoff, department, reason string, requiredTools []string, approvals []string) {
	for index := range *handoffs {
		if (*handoffs)[index].Department != department {
			continue
		}
		(*handoffs)[index].RequiredTools = uniqueAppend((*handoffs)[index].RequiredTools, requiredTools)
		(*handoffs)[index].Approvals = uniqueAppend((*handoffs)[index].Approvals, approvals)
		return
	}
	*handoffs = append(*handoffs, DepartmentHandoff{
		Department:    department,
		Reason:        reason,
		RequiredTools: uniqueAppend(nil, requiredTools),
		Approvals:     uniqueAppend(nil, approvals),
	})
}

func uniqueAppend(existing []string, extra []string) []string {
	out := append([]string(nil), existing...)
	for _, item := range extra {
		item = strings.TrimSpace(strings.ToLower(item))
		if item != "" && !slices.Contains(out, item) {
			out = append(out, item)
		}
	}
	sort.Strings(out)
	return out
}

func normalizeStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(strings.ToLower(value))
		if value != "" && !slices.Contains(out, value) {
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}

func hasAny(values []string, wants ...string) bool {
	for _, want := range wants {
		if slices.Contains(values, strings.ToLower(strings.TrimSpace(want))) {
			return true
		}
	}
	return false
}

func matchesAny(value string, wants ...string) bool {
	for _, want := range wants {
		if value == strings.ToLower(strings.TrimSpace(want)) {
			return true
		}
	}
	return false
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func intersect(values []string, allowed []string) []string {
	out := make([]string, 0)
	for _, value := range values {
		if slices.Contains(allowed, value) {
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}

func isPremiumTask(task domain.Task) bool {
	for _, label := range task.Labels {
		switch strings.ToLower(strings.TrimSpace(label)) {
		case "premium", "enterprise":
			return true
		}
	}
	return false
}

func estimateCost(units int) float64 {
	if units < 1 {
		units = 1
	}
	return float64(units) * 1.5
}

func estimateOverageCost(units int) float64 {
	if units < 1 {
		return 0
	}
	return float64(units) * 4.0
}
