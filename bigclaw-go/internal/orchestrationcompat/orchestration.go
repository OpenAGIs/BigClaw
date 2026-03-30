package orchestrationcompat

import (
	"fmt"
	"sort"
	"strings"
)

type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

type Task struct {
	TaskID             string
	Source             string
	Title              string
	Description        string
	Labels             []string
	Priority           int
	RiskLevel          RiskLevel
	RequiredTools      []string
	AcceptanceCriteria []string
	ValidationPlan     []string
	Budget             float64
}

type DepartmentHandoff struct {
	Department    string
	Reason        string
	RequiredTools []string
	Approvals     []string
}

type OrchestrationPlan struct {
	TaskID            string
	CollaborationMode string
	Handoffs          []DepartmentHandoff
}

func (p OrchestrationPlan) Departments() []string {
	out := make([]string, 0, len(p.Handoffs))
	for _, handoff := range p.Handoffs {
		out = append(out, handoff.Department)
	}
	return out
}

func (p OrchestrationPlan) DepartmentCount() int {
	return len(p.Handoffs)
}

func (p OrchestrationPlan) RequiredApprovals() []string {
	out := make([]string, 0)
	for _, handoff := range p.Handoffs {
		for _, approval := range handoff.Approvals {
			if approval != "" && !contains(out, approval) {
				out = append(out, approval)
			}
		}
	}
	return out
}

type OrchestrationPolicyDecision struct {
	Tier               string
	UpgradeRequired    bool
	Reason             string
	BlockedDepartments []string
	EntitlementStatus  string
	BillingModel       string
	EstimatedCostUSD   float64
	IncludedUsageUnits int
	OverageUsageUnits  int
	OverageCostUSD     float64
}

type HandoffRequest struct {
	TargetTeam        string
	Reason            string
	Status            string
	RequiredApprovals []string
}

type CrossDepartmentOrchestrator struct{}
type PremiumOrchestrationPolicy struct{}

func (CrossDepartmentOrchestrator) Plan(task Task) OrchestrationPlan {
	labels := toSet(task.Labels)
	tools := toSet(task.RequiredTools)
	textParts := []string{strings.ToLower(task.Title), strings.ToLower(task.Description)}
	for _, item := range task.AcceptanceCriteria {
		textParts = append(textParts, strings.ToLower(item))
	}
	for _, item := range task.ValidationPlan {
		textParts = append(textParts, strings.ToLower(item))
	}
	text := strings.Join(textParts, " ")

	handoffs := make([]DepartmentHandoff, 0)
	appendUniqueHandoff(&handoffs, "operations", operationsReason(task, labels, text), nil, nil)
	if len(task.RequiredTools) > 0 || strings.Contains(strings.ToLower(task.Source), "github") || intersects(tools, []string{"repo", "browser", "terminal"}) {
		appendUniqueHandoff(&handoffs, "engineering", "implements automation and tool-driven execution", sortedValues(tools), nil)
	}
	if task.RiskLevel == RiskHigh || intersects(labels, []string{"security", "compliance"}) || strings.Contains(text, "approval") {
		approvals := []string{}
		if task.RiskLevel == RiskHigh {
			approvals = []string{"security-review"}
		}
		appendUniqueHandoff(&handoffs, "security", "reviews elevated risk, compliance, or approval-sensitive work", nil, approvals)
	}
	if intersects(labels, []string{"data", "analytics"}) || intersects(tools, []string{"sql", "warehouse", "bi"}) {
		appendUniqueHandoff(&handoffs, "data", "validates analytics, warehouse, or measurement dependencies", intersectSorted(tools, []string{"sql", "warehouse", "bi"}), nil)
	}
	if intersects(labels, []string{"customer", "support", "success"}) || strings.Contains(text, "customer") || strings.Contains(text, "stakeholder") {
		appendUniqueHandoff(&handoffs, "customer-success", "coordinates customer communication and rollout readiness", nil, nil)
	}
	mode := "single-team"
	if len(handoffs) > 1 {
		mode = "cross-functional"
	}
	return OrchestrationPlan{TaskID: task.TaskID, CollaborationMode: mode, Handoffs: handoffs}
}

func (PremiumOrchestrationPolicy) Apply(task Task, plan OrchestrationPlan) (OrchestrationPlan, OrchestrationPolicyDecision) {
	requestedUnits := max(1, plan.DepartmentCount())
	if isPremium(task) {
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
	allowed := make([]DepartmentHandoff, 0, len(plan.Handoffs))
	for _, handoff := range plan.Handoffs {
		if handoff.Department == "operations" || handoff.Department == "engineering" {
			allowed = append(allowed, handoff)
		} else {
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
		Handoffs:          allowed,
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

func RenderOrchestrationPlan(plan OrchestrationPlan, policy *OrchestrationPolicyDecision, handoff *HandoffRequest) string {
	lines := []string{
		"# Cross-Department Orchestration Plan",
		"",
		fmt.Sprintf("- Task ID: %s", plan.TaskID),
		fmt.Sprintf("- Collaboration Mode: %s", plan.CollaborationMode),
		fmt.Sprintf("- Departments: %s", joinOrNone(plan.Departments())),
		fmt.Sprintf("- Required Approvals: %s", joinOrNone(plan.RequiredApprovals())),
	}
	if policy != nil {
		lines = append(lines,
			fmt.Sprintf("- Tier: %s", policy.Tier),
			fmt.Sprintf("- Upgrade Required: %t", policy.UpgradeRequired),
			fmt.Sprintf("- Entitlement Status: %s", policy.EntitlementStatus),
			fmt.Sprintf("- Billing Model: %s", policy.BillingModel),
			fmt.Sprintf("- Estimated Cost (USD): %.2f", policy.EstimatedCostUSD),
			fmt.Sprintf("- Included Usage Units: %d", policy.IncludedUsageUnits),
			fmt.Sprintf("- Overage Usage Units: %d", policy.OverageUsageUnits),
			fmt.Sprintf("- Overage Cost (USD): %.2f", policy.OverageCostUSD),
			fmt.Sprintf("- Policy Reason: %s", policy.Reason),
			fmt.Sprintf("- Blocked Departments: %s", joinOrNone(policy.BlockedDepartments)),
		)
	}
	if handoff != nil {
		lines = append(lines,
			fmt.Sprintf("- Human Handoff Team: %s", handoff.TargetTeam),
			fmt.Sprintf("- Human Handoff Status: %s", handoff.Status),
			fmt.Sprintf("- Human Handoff Reason: %s", handoff.Reason),
			fmt.Sprintf("- Human Handoff Approvals: %s", joinOrNone(handoff.RequiredApprovals)),
		)
	}
	lines = append(lines, "", "## Handoffs", "")
	if len(plan.Handoffs) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, h := range plan.Handoffs {
			lines = append(lines, fmt.Sprintf("- %s: reason=%s tools=%s approvals=%s", h.Department, h.Reason, joinOrNone(h.RequiredTools), joinOrNone(h.Approvals)))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

type SchedulerDecision struct {
	Medium   string
	Approved bool
	Reason   string
}

type TraceEntry struct {
	Span       string
	Status     string
	Attributes map[string]any
}

type AuditEntry struct {
	Action  string
	Actor   string
	Outcome string
	Details map[string]any
}

type TaskRun struct {
	RunID  string
	Medium string
	Traces []TraceEntry
	Audits []AuditEntry
}

func (r *TaskRun) Trace(span, status string, attrs map[string]any) {
	r.Traces = append(r.Traces, TraceEntry{Span: span, Status: status, Attributes: attrs})
}

func (r *TaskRun) Audit(action, actor, outcome string, details map[string]any) {
	r.Audits = append(r.Audits, AuditEntry{Action: action, Actor: actor, Outcome: outcome, Details: details})
}

type ObservabilityLedger struct {
	entries []TaskRun
}

func (l *ObservabilityLedger) Append(run TaskRun) {
	l.entries = append(l.entries, run)
}

func (l *ObservabilityLedger) Load() []map[string]any {
	out := make([]map[string]any, 0, len(l.entries))
	for _, run := range l.entries {
		traces := make([]map[string]any, 0, len(run.Traces))
		for _, trace := range run.Traces {
			traces = append(traces, map[string]any{"span": trace.Span, "status": trace.Status, "attributes": trace.Attributes})
		}
		audits := make([]map[string]any, 0, len(run.Audits))
		for _, audit := range run.Audits {
			audits = append(audits, map[string]any{"action": audit.Action, "actor": audit.Actor, "outcome": audit.Outcome, "details": audit.Details})
		}
		out = append(out, map[string]any{"run_id": run.RunID, "medium": run.Medium, "traces": traces, "audits": audits})
	}
	return out
}

type ExecutionRecord struct {
	Decision            SchedulerDecision
	Run                 TaskRun
	OrchestrationPlan   OrchestrationPlan
	OrchestrationPolicy OrchestrationPolicyDecision
	HandoffRequest      *HandoffRequest
}

type Scheduler struct {
	orchestrator        CrossDepartmentOrchestrator
	orchestrationPolicy PremiumOrchestrationPolicy
}

func (s Scheduler) Decide(task Task) SchedulerDecision {
	decision := SchedulerDecision{Medium: "docker", Approved: true, Reason: "default low risk path"}
	if task.RiskLevel == RiskHigh {
		decision = SchedulerDecision{Medium: "vm", Approved: false, Reason: "requires approval for high-risk task"}
	} else if contains(task.RequiredTools, "browser") {
		decision = SchedulerDecision{Medium: "browser", Approved: true, Reason: "browser automation task"}
	} else if task.RiskLevel == RiskMedium {
		decision = SchedulerDecision{Medium: "docker", Approved: true, Reason: "medium risk in docker"}
	}
	return applyBudgetPolicy(task, decision)
}

func (s Scheduler) Execute(task Task, runID string, ledger *ObservabilityLedger) ExecutionRecord {
	decision := s.Decide(task)
	rawPlan := s.orchestrator.Plan(task)
	plan, policy := s.orchestrationPolicy.Apply(task, rawPlan)
	handoff := buildHandoffRequest(decision, plan, policy)
	run := TaskRun{RunID: runID, Medium: decision.Medium}
	run.Trace("orchestration.plan", "ready", map[string]any{
		"collaboration_mode": plan.CollaborationMode,
		"departments":        plan.Departments(),
		"handoffs":           plan.DepartmentCount(),
	})
	run.Trace("orchestration.policy", ternary(policy.UpgradeRequired, "upgrade-required", "ok"), map[string]any{
		"tier":                 policy.Tier,
		"entitlement_status":   policy.EntitlementStatus,
		"billing_model":        policy.BillingModel,
		"estimated_cost_usd":   policy.EstimatedCostUSD,
		"included_usage_units": policy.IncludedUsageUnits,
		"overage_usage_units":  policy.OverageUsageUnits,
	})
	run.Audit("orchestration.plan", "scheduler", "ready", map[string]any{
		"collaboration_mode": plan.CollaborationMode,
		"departments":        plan.Departments(),
		"approvals":          plan.RequiredApprovals(),
	})
	run.Audit("orchestration.policy", "scheduler", ternary(policy.UpgradeRequired, "upgrade-required", "enabled"), map[string]any{
		"tier":                 policy.Tier,
		"reason":               policy.Reason,
		"entitlement_status":   policy.EntitlementStatus,
		"billing_model":        policy.BillingModel,
		"estimated_cost_usd":   policy.EstimatedCostUSD,
		"included_usage_units": policy.IncludedUsageUnits,
		"overage_usage_units":  policy.OverageUsageUnits,
		"overage_cost_usd":     policy.OverageCostUSD,
		"blocked_departments":  policy.BlockedDepartments,
	})
	if handoff != nil {
		run.Trace("orchestration.handoff", handoff.Status, map[string]any{
			"target_team":        handoff.TargetTeam,
			"required_approvals": handoff.RequiredApprovals,
		})
		run.Audit("orchestration.handoff", "scheduler", handoff.Status, map[string]any{
			"target_team":        handoff.TargetTeam,
			"reason":             handoff.Reason,
			"required_approvals": handoff.RequiredApprovals,
		})
	}
	ledger.Append(run)
	return ExecutionRecord{
		Decision:            decision,
		Run:                 run,
		OrchestrationPlan:   plan,
		OrchestrationPolicy: policy,
		HandoffRequest:      handoff,
	}
}

func buildHandoffRequest(decision SchedulerDecision, plan OrchestrationPlan, policy OrchestrationPolicyDecision) *HandoffRequest {
	if !decision.Approved {
		approvals := plan.RequiredApprovals()
		if len(approvals) == 0 {
			approvals = []string{"security-review"}
		}
		return &HandoffRequest{
			TargetTeam:        "security",
			Reason:            decision.Reason,
			Status:            "pending",
			RequiredApprovals: approvals,
		}
	}
	if policy.UpgradeRequired {
		return &HandoffRequest{
			TargetTeam:        "operations",
			Reason:            policy.Reason,
			Status:            "blocked",
			RequiredApprovals: []string{"ops-manager"},
		}
	}
	return nil
}

func applyBudgetPolicy(task Task, decision SchedulerDecision) SchedulerDecision {
	effective := task.Budget
	if effective <= 0 {
		return decision
	}
	floors := map[string]float64{"docker": 10.0, "browser": 20.0, "vm": 40.0}
	required := floors[decision.Medium]
	if effective >= required {
		return decision
	}
	if decision.Medium == "browser" && task.RiskLevel != RiskHigh && effective >= floors["docker"] {
		return SchedulerDecision{
			Medium:   "docker",
			Approved: true,
			Reason:   fmt.Sprintf("budget degraded browser route to docker (budget %.1f < required %.1f)", effective, required),
		}
	}
	return SchedulerDecision{
		Medium:   "none",
		Approved: false,
		Reason:   fmt.Sprintf("paused: budget %.1f below required %s budget %.1f", effective, decision.Medium, required),
	}
}

func operationsReason(task Task, labels map[string]struct{}, text string) string {
	if hasAny(labels, "program", "ops", "release") || strings.Contains(text, "rollout") || matchesAny(strings.ToLower(task.Source), "linear", "jira") {
		return "coordinates issue intake, handoffs, and completion tracking"
	}
	return "owns task intake and delivery coordination"
}

func appendUniqueHandoff(handoffs *[]DepartmentHandoff, department, reason string, requiredTools, approvals []string) {
	for i := range *handoffs {
		if (*handoffs)[i].Department == department {
			(*handoffs)[i].RequiredTools = uniqueAppend((*handoffs)[i].RequiredTools, requiredTools)
			(*handoffs)[i].Approvals = uniqueAppend((*handoffs)[i].Approvals, approvals)
			return
		}
	}
	*handoffs = append(*handoffs, DepartmentHandoff{
		Department:    department,
		Reason:        reason,
		RequiredTools: uniqueAppend(nil, requiredTools),
		Approvals:     uniqueAppend(nil, approvals),
	})
}

func toSet(values []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" {
			out[value] = struct{}{}
		}
	}
	return out
}

func sortedValues(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func intersectSorted(values map[string]struct{}, allowed []string) []string {
	out := make([]string, 0)
	for _, item := range allowed {
		if _, ok := values[item]; ok {
			out = append(out, item)
		}
	}
	return out
}

func intersects(values map[string]struct{}, wants []string) bool {
	for _, want := range wants {
		if _, ok := values[want]; ok {
			return true
		}
	}
	return false
}

func hasAny(values map[string]struct{}, wants ...string) bool {
	return intersects(values, wants)
}

func matchesAny(value string, wants ...string) bool {
	for _, want := range wants {
		if value == want {
			return true
		}
	}
	return false
}

func uniqueAppend(base, extra []string) []string {
	out := append([]string{}, base...)
	for _, item := range extra {
		if !contains(out, item) {
			out = append(out, item)
		}
	}
	return out
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
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

func estimateCost(units int) float64 {
	return round2(1.5 * float64(max(1, units)))
}

func estimateOverageCost(units int) float64 {
	return round2(4.0 * float64(max(0, units)))
}

func isPremium(task Task) bool {
	for _, label := range task.Labels {
		label = strings.ToLower(strings.TrimSpace(label))
		if label == "premium" || label == "enterprise" {
			return true
		}
	}
	return false
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func ternary(cond bool, yes, no string) string {
	if cond {
		return yes
	}
	return no
}
