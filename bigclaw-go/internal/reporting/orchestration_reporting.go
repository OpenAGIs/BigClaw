package reporting

import (
	"fmt"
	"html"
	"sort"
	"strings"

	"bigclaw-go/internal/observability"
	"bigclaw-go/internal/workflow"
)

type TakeoverRequest struct {
	RunID             string          `json:"run_id"`
	TaskID            string          `json:"task_id"`
	Source            string          `json:"source"`
	TargetTeam        string          `json:"target_team"`
	Status            string          `json:"status"`
	Reason            string          `json:"reason"`
	RequiredApprovals []string        `json:"required_approvals,omitempty"`
	Actions           []ConsoleAction `json:"actions,omitempty"`
}

type TakeoverQueue struct {
	Name     string            `json:"name"`
	Period   string            `json:"period"`
	Requests []TakeoverRequest `json:"requests,omitempty"`
}

func (q TakeoverQueue) PendingRequests() int { return len(q.Requests) }

func (q TakeoverQueue) TeamCounts() map[string]int {
	counts := make(map[string]int)
	for _, request := range q.Requests {
		counts[request.TargetTeam]++
	}
	return counts
}

func (q TakeoverQueue) ApprovalCount() int {
	total := 0
	for _, request := range q.Requests {
		total += len(request.RequiredApprovals)
	}
	return total
}

func (q TakeoverQueue) Recommendation() string {
	for _, request := range q.Requests {
		if request.TargetTeam == "security" {
			return "expedite-security-review"
		}
	}
	if len(q.Requests) > 0 {
		return "staff-takeover-queue"
	}
	return "monitor"
}

type OrchestrationCanvas struct {
	TaskID             string               `json:"task_id"`
	RunID              string               `json:"run_id"`
	CollaborationMode  string               `json:"collaboration_mode"`
	Departments        []string             `json:"departments,omitempty"`
	RequiredApprovals  []string             `json:"required_approvals,omitempty"`
	Tier               string               `json:"tier"`
	UpgradeRequired    bool                 `json:"upgrade_required"`
	BlockedDepartments []string             `json:"blocked_departments,omitempty"`
	HandoffTeam        string               `json:"handoff_team"`
	HandoffStatus      string               `json:"handoff_status"`
	HandoffReason      string               `json:"handoff_reason,omitempty"`
	ActiveTools        []string             `json:"active_tools,omitempty"`
	EntitlementStatus  string               `json:"entitlement_status"`
	BillingModel       string               `json:"billing_model"`
	EstimatedCostUSD   float64              `json:"estimated_cost_usd"`
	IncludedUsageUnits int                  `json:"included_usage_units"`
	OverageUsageUnits  int                  `json:"overage_usage_units"`
	OverageCostUSD     float64              `json:"overage_cost_usd"`
	Actions            []ConsoleAction      `json:"actions,omitempty"`
	Collaboration      *CollaborationThread `json:"collaboration,omitempty"`
}

func (c OrchestrationCanvas) Recommendation() string {
	if c.Collaboration != nil && c.Collaboration.OpenCommentCount() > 0 {
		return "resolve-flow-comments"
	}
	if c.HandoffTeam == "security" {
		return "review-security-takeover"
	}
	if c.UpgradeRequired {
		return "resolve-entitlement-gap"
	}
	if c.OverageCostUSD > 0 {
		return "review-billing-overage"
	}
	if len(c.Departments) > 1 {
		return "continue-cross-team-execution"
	}
	return "monitor"
}

type OrchestrationPortfolio struct {
	Name          string                `json:"name"`
	Period        string                `json:"period"`
	Canvases      []OrchestrationCanvas `json:"canvases,omitempty"`
	TakeoverQueue *TakeoverQueue        `json:"takeover_queue,omitempty"`
}

func (p OrchestrationPortfolio) TotalRuns() int { return len(p.Canvases) }

func (p OrchestrationPortfolio) CollaborationModes() map[string]int {
	counts := make(map[string]int)
	for _, canvas := range p.Canvases {
		counts[canvas.CollaborationMode]++
	}
	return counts
}

func (p OrchestrationPortfolio) TierCounts() map[string]int {
	counts := make(map[string]int)
	for _, canvas := range p.Canvases {
		counts[canvas.Tier]++
	}
	return counts
}

func (p OrchestrationPortfolio) UpgradeRequiredCount() int {
	total := 0
	for _, canvas := range p.Canvases {
		if canvas.UpgradeRequired {
			total++
		}
	}
	return total
}

func (p OrchestrationPortfolio) ActiveHandoffs() int {
	total := 0
	for _, canvas := range p.Canvases {
		if canvas.HandoffTeam != "none" {
			total++
		}
	}
	return total
}

func (p OrchestrationPortfolio) EntitlementCounts() map[string]int {
	counts := make(map[string]int)
	for _, canvas := range p.Canvases {
		counts[canvas.EntitlementStatus]++
	}
	return counts
}

func (p OrchestrationPortfolio) BillingModelCounts() map[string]int {
	counts := make(map[string]int)
	for _, canvas := range p.Canvases {
		counts[canvas.BillingModel]++
	}
	return counts
}

func (p OrchestrationPortfolio) TotalEstimatedCostUSD() float64 {
	total := 0.0
	for _, canvas := range p.Canvases {
		total += canvas.EstimatedCostUSD
	}
	return round2(total)
}

func (p OrchestrationPortfolio) TotalOverageCostUSD() float64 {
	total := 0.0
	for _, canvas := range p.Canvases {
		total += canvas.OverageCostUSD
	}
	return round2(total)
}

func (p OrchestrationPortfolio) Recommendation() string {
	if p.TakeoverQueue != nil && p.TakeoverQueue.Recommendation() == "expedite-security-review" {
		return "stabilize-security-takeovers"
	}
	if p.UpgradeRequiredCount() > 0 {
		return "close-entitlement-gaps"
	}
	if p.ActiveHandoffs() > 0 {
		return "manage-cross-team-flow"
	}
	return "monitor"
}

type BillingRunCharge struct {
	RunID               string   `json:"run_id"`
	TaskID              string   `json:"task_id"`
	BillingModel        string   `json:"billing_model"`
	EntitlementStatus   string   `json:"entitlement_status"`
	EstimatedCostUSD    float64  `json:"estimated_cost_usd"`
	IncludedUsageUnits  int      `json:"included_usage_units"`
	OverageUsageUnits   int      `json:"overage_usage_units"`
	OverageCostUSD      float64  `json:"overage_cost_usd"`
	BlockedCapabilities []string `json:"blocked_capabilities,omitempty"`
	HandoffTeam         string   `json:"handoff_team"`
	Recommendation      string   `json:"recommendation"`
}

type BillingEntitlementsPage struct {
	WorkspaceName string             `json:"workspace_name"`
	PlanName      string             `json:"plan_name"`
	BillingPeriod string             `json:"billing_period"`
	Charges       []BillingRunCharge `json:"charges,omitempty"`
}

func (p BillingEntitlementsPage) RunCount() int { return len(p.Charges) }

func (p BillingEntitlementsPage) TotalEstimatedCostUSD() float64 {
	total := 0.0
	for _, charge := range p.Charges {
		total += charge.EstimatedCostUSD
	}
	return round2(total)
}

func (p BillingEntitlementsPage) TotalIncludedUsageUnits() int {
	total := 0
	for _, charge := range p.Charges {
		total += charge.IncludedUsageUnits
	}
	return total
}

func (p BillingEntitlementsPage) TotalOverageUsageUnits() int {
	total := 0
	for _, charge := range p.Charges {
		total += charge.OverageUsageUnits
	}
	return total
}

func (p BillingEntitlementsPage) TotalOverageCostUSD() float64 {
	total := 0.0
	for _, charge := range p.Charges {
		total += charge.OverageCostUSD
	}
	return round2(total)
}

func (p BillingEntitlementsPage) UpgradeRequiredCount() int {
	total := 0
	for _, charge := range p.Charges {
		if charge.EntitlementStatus == "upgrade-required" {
			total++
		}
	}
	return total
}

func (p BillingEntitlementsPage) BillingModelCounts() map[string]int {
	counts := make(map[string]int)
	for _, charge := range p.Charges {
		counts[charge.BillingModel]++
	}
	return counts
}

func (p BillingEntitlementsPage) EntitlementCounts() map[string]int {
	counts := make(map[string]int)
	for _, charge := range p.Charges {
		counts[charge.EntitlementStatus]++
	}
	return counts
}

func (p BillingEntitlementsPage) BlockedCapabilities() []string {
	seen := make(map[string]struct{})
	out := make([]string, 0)
	for _, charge := range p.Charges {
		for _, capability := range charge.BlockedCapabilities {
			if _, ok := seen[capability]; ok {
				continue
			}
			seen[capability] = struct{}{}
			out = append(out, capability)
		}
	}
	return out
}

func (p BillingEntitlementsPage) Recommendation() string {
	if p.UpgradeRequiredCount() > 0 {
		return "resolve-plan-gaps"
	}
	if p.TotalOverageCostUSD() > 0 {
		return "optimize-billed-usage"
	}
	for _, charge := range p.Charges {
		if charge.HandoffTeam != "none" {
			return "monitor-shared-capacity"
		}
	}
	return "healthy"
}

func BuildTakeoverQueueFromLedger(entries []map[string]any, name string, period string) TakeoverQueue {
	requests := make([]TakeoverRequest, 0)
	for _, entry := range entries {
		handoffAudit := latestHandoffAudit(anyMapSlice(entry["audits"]))
		if handoffAudit == nil {
			continue
		}
		details := mapValue(handoffAudit["details"])
		runID := stringValue(entry["run_id"])
		status := firstNonEmptyString(stringValue(handoffAudit["outcome"]), "pending")
		targetTeam := firstNonEmptyString(stringValue(details["target_team"]), "operations")
		requests = append(requests, TakeoverRequest{
			RunID:             runID,
			TaskID:            stringValue(entry["task_id"]),
			Source:            stringValue(entry["source"]),
			TargetTeam:        targetTeam,
			Status:            status,
			Reason:            firstNonEmptyString(stringValue(details["reason"]), stringValue(entry["summary"]), "handoff requested"),
			RequiredApprovals: anySliceToStrings(details["required_approvals"]),
			Actions:           workflowConsoleActions(runID, false, "retry is blocked while takeover is pending", status == "pending", "only pending takeovers can be paused", true, "", targetTeam != "security", "security takeovers are already escalated"),
		})
	}
	sort.Slice(requests, func(i, j int) bool {
		if requests[i].TargetTeam == requests[j].TargetTeam {
			return requests[i].RunID < requests[j].RunID
		}
		return requests[i].TargetTeam < requests[j].TargetTeam
	})
	return TakeoverQueue{Name: name, Period: period, Requests: requests}
}

func RenderTakeoverQueueReport(queue TakeoverQueue, totalRuns *int, view *SharedViewContext) string {
	total := queue.PendingRequests()
	if totalRuns != nil {
		total = *totalRuns
	}
	lines := []string{
		"# Human Takeover Queue",
		"",
		"- Queue: " + queue.Name,
		"- Period: " + queue.Period,
		fmt.Sprintf("- Pending Requests: %d", queue.PendingRequests()),
		fmt.Sprintf("- Total Runs: %d", total),
		"- Recommendation: " + queue.Recommendation(),
		"- Team Mix: " + formatCountMap(queue.TeamCounts()),
		fmt.Sprintf("- Required Approvals: %d", queue.ApprovalCount()),
		"",
		"## Requests",
		"",
	}
	lines = append(lines, RenderSharedViewContext(view)...)
	if len(queue.Requests) == 0 {
		lines = append(lines, "- None")
		return strings.Join(lines, "\n") + "\n"
	}
	for _, request := range queue.Requests {
		approvals := "none"
		if len(request.RequiredApprovals) > 0 {
			approvals = strings.Join(request.RequiredApprovals, ",")
		}
		lines = append(lines, fmt.Sprintf("- %s: team=%s status=%s task=%s approvals=%s reason=%s actions=%s", request.RunID, request.TargetTeam, request.Status, request.TaskID, approvals, request.Reason, RenderConsoleActions(request.Actions)))
	}
	return strings.Join(lines, "\n") + "\n"
}

func BuildOrchestrationCanvas(run observability.TaskRun, plan workflow.OrchestrationPlan, policy *workflow.OrchestrationPolicyDecision, handoffRequest *workflow.HandoffRequest) OrchestrationCanvas {
	activeTools := make([]string, 0)
	seen := make(map[string]struct{})
	for _, audit := range run.Audits {
		if audit.Action != "tool.invoke" {
			continue
		}
		tool := stringValue(audit.Details["tool"])
		if tool == "" {
			continue
		}
		if _, ok := seen[tool]; ok {
			continue
		}
		seen[tool] = struct{}{}
		activeTools = append(activeTools, tool)
	}
	sort.Strings(activeTools)
	approvals := plan.RequiredApprovals()
	canvas := OrchestrationCanvas{
		TaskID:            run.TaskID,
		RunID:             run.RunID,
		CollaborationMode: plan.CollaborationMode,
		Departments:       append([]string(nil), plan.Departments()...),
		RequiredApprovals: approvals,
		Tier:              "standard",
		HandoffTeam:       "none",
		HandoffStatus:     "none",
		EntitlementStatus: "included",
		BillingModel:      "standard-included",
		ActiveTools:       activeTools,
	}
	if policy != nil {
		canvas.Tier = policy.Tier
		canvas.UpgradeRequired = policy.UpgradeRequired
		canvas.BlockedDepartments = append([]string(nil), policy.BlockedDepartments...)
		canvas.EntitlementStatus = policy.EntitlementStatus
		canvas.BillingModel = policy.BillingModel
		canvas.EstimatedCostUSD = policy.EstimatedCostUSD
		canvas.IncludedUsageUnits = policy.IncludedUsageUnits
		canvas.OverageUsageUnits = policy.OverageUsageUnits
		canvas.OverageCostUSD = policy.OverageCostUSD
	}
	if handoffRequest != nil {
		canvas.HandoffTeam = handoffRequest.TargetTeam
		canvas.HandoffStatus = handoffRequest.Status
		canvas.HandoffReason = handoffRequest.Reason
	}
	canvas.Actions = workflowConsoleActions(
		run.RunID,
		handoffRequest == nil || handoffRequest.Status != "pending",
		"pending handoff must be resolved before retry",
		run.Status != "failed" && run.Status != "completed" && run.Status != "approved",
		"completed or failed runs cannot be paused",
		handoffRequest != nil,
		"reassign is available after a handoff exists",
		policy != nil && policy.UpgradeRequired,
		"escalate when policy requires an entitlement or approval upgrade",
	)
	runAudits := make([]map[string]any, 0, len(run.Audits))
	for _, audit := range run.Audits {
		runAudits = append(runAudits, map[string]any{
			"action":    audit.Action,
			"actor":     audit.Actor,
			"outcome":   audit.Outcome,
			"timestamp": audit.Timestamp,
			"details":   audit.Details,
		})
	}
	canvas.Collaboration = BuildCollaborationThreadFromAudits(runAudits, "flow", run.RunID)
	return canvas
}

func BuildOrchestrationCanvasFromLedgerEntry(entry map[string]any) OrchestrationCanvas {
	audits := anyMapSlice(entry["audits"])
	planAudit := latestNamedAudit(audits, "orchestration.plan")
	policyAudit := latestNamedAudit(audits, "orchestration.policy")
	handoffAudit := latestHandoffAudit(audits)
	toolSet := make(map[string]struct{})
	for _, audit := range audits {
		if stringValue(audit["action"]) != "tool.invoke" {
			continue
		}
		tool := stringValue(mapValue(audit["details"])["tool"])
		if tool != "" {
			toolSet[tool] = struct{}{}
		}
	}
	activeTools := sortedStringSet(toolSet)
	planDetails := mapValue(nil)
	if planAudit != nil {
		planDetails = mapValue(planAudit["details"])
	}
	policyDetails := mapValue(nil)
	if policyAudit != nil {
		policyDetails = mapValue(policyAudit["details"])
	}
	handoffDetails := mapValue(nil)
	if handoffAudit != nil {
		handoffDetails = mapValue(handoffAudit["details"])
	}
	canvas := OrchestrationCanvas{
		TaskID:             stringValue(entry["task_id"]),
		RunID:              stringValue(entry["run_id"]),
		CollaborationMode:  firstNonEmptyString(stringValue(planDetails["collaboration_mode"]), "single-team"),
		Departments:        anySliceToStrings(planDetails["departments"]),
		RequiredApprovals:  anySliceToStrings(planDetails["approvals"]),
		Tier:               firstNonEmptyString(stringValue(policyDetails["tier"]), "standard"),
		UpgradeRequired:    policyAudit != nil && stringValue(policyAudit["outcome"]) == "upgrade-required" && stringValue(policyDetails["tier"]) != "",
		BlockedDepartments: anySliceToStrings(policyDetails["blocked_departments"]),
		HandoffTeam:        "none",
		HandoffStatus:      "none",
		HandoffReason:      "",
		ActiveTools:        activeTools,
		EntitlementStatus:  firstNonEmptyString(stringValue(policyDetails["entitlement_status"]), "included"),
		BillingModel:       firstNonEmptyString(stringValue(policyDetails["billing_model"]), "standard-included"),
		EstimatedCostUSD:   floatValue(policyDetails["estimated_cost_usd"]),
		IncludedUsageUnits: intValue(policyDetails["included_usage_units"]),
		OverageUsageUnits:  intValue(policyDetails["overage_usage_units"]),
		OverageCostUSD:     floatValue(policyDetails["overage_cost_usd"]),
		Collaboration:      BuildCollaborationThreadFromAudits(audits, "flow", stringValue(entry["run_id"])),
	}
	if handoffAudit != nil {
		canvas.HandoffTeam = firstNonEmptyString(stringValue(handoffDetails["target_team"]), "none")
		canvas.HandoffStatus = firstNonEmptyString(stringValue(handoffAudit["outcome"]), "none")
		canvas.HandoffReason = stringValue(handoffDetails["reason"])
	}
	canvas.Actions = workflowConsoleActions(
		canvas.RunID,
		handoffAudit == nil || canvas.HandoffStatus != "pending",
		"pending handoff must be resolved before retry",
		handoffAudit == nil || canvas.HandoffStatus != "completed",
		"completed handoff runs cannot be paused",
		handoffAudit != nil,
		"reassign is available after a handoff exists",
		policyAudit != nil && stringValue(policyAudit["outcome"]) == "upgrade-required",
		"escalate when policy requires an entitlement or approval upgrade",
	)
	return canvas
}

func RenderOrchestrationCanvas(canvas OrchestrationCanvas) string {
	lines := []string{
		"# Orchestration Canvas",
		"",
		"- Task ID: " + canvas.TaskID,
		"- Run ID: " + canvas.RunID,
		"- Collaboration Mode: " + canvas.CollaborationMode,
		"- Departments: " + joinListOrNone(canvas.Departments),
		"- Required Approvals: " + joinListOrNone(canvas.RequiredApprovals),
		"- Tier: " + canvas.Tier,
		"- Upgrade Required: " + pyBool(canvas.UpgradeRequired),
		"- Entitlement Status: " + canvas.EntitlementStatus,
		"- Billing Model: " + canvas.BillingModel,
		"- Blocked Departments: " + joinListOrNone(canvas.BlockedDepartments),
		"- Handoff Team: " + canvas.HandoffTeam,
		"- Handoff Status: " + canvas.HandoffStatus,
		"- Recommendation: " + canvas.Recommendation(),
		"",
		"## Execution Context",
		"",
		"- Active Tools: " + joinListOrNone(canvas.ActiveTools),
		fmt.Sprintf("- Estimated Cost (USD): %.2f", canvas.EstimatedCostUSD),
		fmt.Sprintf("- Included Usage Units: %d", canvas.IncludedUsageUnits),
		fmt.Sprintf("- Overage Usage Units: %d", canvas.OverageUsageUnits),
		fmt.Sprintf("- Overage Cost (USD): %.2f", canvas.OverageCostUSD),
		"- Handoff Reason: " + firstNonEmptyString(canvas.HandoffReason, "none"),
		"",
		"## Actions",
		"",
		"- " + RenderConsoleActions(canvas.Actions),
	}
	lines = append(lines, RenderCollaborationLines(canvas.Collaboration)...)
	return strings.Join(lines, "\n") + "\n"
}

func BuildOrchestrationPortfolio(canvases []OrchestrationCanvas, name string, period string, takeoverQueue *TakeoverQueue) OrchestrationPortfolio {
	normalized := make([]OrchestrationCanvas, 0, len(canvases))
	for _, canvas := range canvases {
		if len(canvas.Actions) == 0 {
			canvas.Actions = defaultCanvasActions(canvas)
		}
		normalized = append(normalized, canvas)
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i].RunID < normalized[j].RunID })
	return OrchestrationPortfolio{Name: name, Period: period, Canvases: normalized, TakeoverQueue: takeoverQueue}
}

func BuildOrchestrationPortfolioFromLedger(entries []map[string]any, name string, period string) OrchestrationPortfolio {
	canvases := make([]OrchestrationCanvas, 0)
	for _, entry := range entries {
		if latestNamedAudit(anyMapSlice(entry["audits"]), "orchestration.plan") == nil {
			continue
		}
		canvases = append(canvases, BuildOrchestrationCanvasFromLedgerEntry(entry))
	}
	queue := BuildTakeoverQueueFromLedger(entries, name+" Takeovers", period)
	return BuildOrchestrationPortfolio(canvases, name, period, &queue)
}

func RenderOrchestrationPortfolioReport(portfolio OrchestrationPortfolio, view *SharedViewContext) string {
	takeoverSummary := "none"
	if portfolio.TakeoverQueue != nil {
		takeoverSummary = fmt.Sprintf("pending=%d recommendation=%s", portfolio.TakeoverQueue.PendingRequests(), portfolio.TakeoverQueue.Recommendation())
	}
	lines := []string{
		"# Orchestration Portfolio Report",
		"",
		"- Portfolio: " + portfolio.Name,
		"- Period: " + portfolio.Period,
		fmt.Sprintf("- Total Runs: %d", portfolio.TotalRuns()),
		"- Recommendation: " + portfolio.Recommendation(),
		"- Collaboration Mix: " + formatCountMap(portfolio.CollaborationModes()),
		"- Tier Mix: " + formatCountMap(portfolio.TierCounts()),
		"- Entitlement Mix: " + formatCountMap(portfolio.EntitlementCounts()),
		"- Billing Models: " + formatCountMap(portfolio.BillingModelCounts()),
		fmt.Sprintf("- Upgrade Required Count: %d", portfolio.UpgradeRequiredCount()),
		fmt.Sprintf("- Estimated Cost (USD): %.2f", portfolio.TotalEstimatedCostUSD()),
		fmt.Sprintf("- Overage Cost (USD): %.2f", portfolio.TotalOverageCostUSD()),
		fmt.Sprintf("- Active Handoffs: %d", portfolio.ActiveHandoffs()),
		"- Takeover Queue: " + takeoverSummary,
		"",
		"## Runs",
		"",
	}
	lines = append(lines, RenderSharedViewContext(view)...)
	if len(portfolio.Canvases) == 0 {
		lines = append(lines, "- None")
		return strings.Join(lines, "\n") + "\n"
	}
	for _, canvas := range portfolio.Canvases {
		collaborationSummary := "comments=0 decisions=0"
		if canvas.Collaboration != nil {
			collaborationSummary = fmt.Sprintf("comments=%d decisions=%d", len(canvas.Collaboration.Comments), len(canvas.Collaboration.Decisions))
		}
		lines = append(lines, fmt.Sprintf("- %s: mode=%s tier=%s entitlement=%s billing=%s estimated_cost_usd=%.2f overage_cost_usd=%.2f upgrade_required=%s handoff=%s collaboration=%s recommendation=%s actions=%s", canvas.RunID, canvas.CollaborationMode, canvas.Tier, canvas.EntitlementStatus, canvas.BillingModel, canvas.EstimatedCostUSD, canvas.OverageCostUSD, pyBool(canvas.UpgradeRequired), canvas.HandoffTeam, collaborationSummary, canvas.Recommendation(), RenderConsoleActions(canvas.Actions)))
	}
	return strings.Join(lines, "\n") + "\n"
}

func BuildBillingEntitlementsPage(portfolio OrchestrationPortfolio, workspaceName string, planName string, billingPeriod string) BillingEntitlementsPage {
	charges := make([]BillingRunCharge, 0, len(portfolio.Canvases))
	for _, canvas := range portfolio.Canvases {
		charges = append(charges, BillingRunCharge{
			RunID:               canvas.RunID,
			TaskID:              canvas.TaskID,
			BillingModel:        canvas.BillingModel,
			EntitlementStatus:   canvas.EntitlementStatus,
			EstimatedCostUSD:    canvas.EstimatedCostUSD,
			IncludedUsageUnits:  canvas.IncludedUsageUnits,
			OverageUsageUnits:   canvas.OverageUsageUnits,
			OverageCostUSD:      canvas.OverageCostUSD,
			BlockedCapabilities: append([]string(nil), canvas.BlockedDepartments...),
			HandoffTeam:         canvas.HandoffTeam,
			Recommendation:      canvas.Recommendation(),
		})
	}
	return BillingEntitlementsPage{
		WorkspaceName: workspaceName,
		PlanName:      planName,
		BillingPeriod: billingPeriod,
		Charges:       charges,
	}
}

func BuildBillingEntitlementsPageFromLedger(entries []map[string]any, workspaceName string, planName string, billingPeriod string) BillingEntitlementsPage {
	portfolio := BuildOrchestrationPortfolioFromLedger(entries, workspaceName, billingPeriod)
	return BuildBillingEntitlementsPage(portfolio, workspaceName, planName, billingPeriod)
}

func RenderBillingEntitlementsReport(page BillingEntitlementsPage, view *SharedViewContext) string {
	blocked := page.BlockedCapabilities()
	blockedText := "none"
	if len(blocked) > 0 {
		blockedText = strings.Join(blocked, ", ")
	}
	lines := []string{
		"# Billing & Entitlements Report",
		"",
		"- Workspace: " + page.WorkspaceName,
		"- Plan: " + page.PlanName,
		"- Billing Period: " + page.BillingPeriod,
		fmt.Sprintf("- Runs: %d", page.RunCount()),
		"- Recommendation: " + page.Recommendation(),
		"- Entitlement Mix: " + formatCountMap(page.EntitlementCounts()),
		"- Billing Models: " + formatCountMap(page.BillingModelCounts()),
		fmt.Sprintf("- Included Usage Units: %d", page.TotalIncludedUsageUnits()),
		fmt.Sprintf("- Overage Usage Units: %d", page.TotalOverageUsageUnits()),
		fmt.Sprintf("- Estimated Cost (USD): %.2f", page.TotalEstimatedCostUSD()),
		fmt.Sprintf("- Overage Cost (USD): %.2f", page.TotalOverageCostUSD()),
		fmt.Sprintf("- Upgrade Required Count: %d", page.UpgradeRequiredCount()),
		"- Blocked Capabilities: " + blockedText,
		"",
		"## Charges",
		"",
	}
	lines = append(lines, RenderSharedViewContext(view)...)
	if len(page.Charges) == 0 {
		lines = append(lines, "- None")
		return strings.Join(lines, "\n") + "\n"
	}
	for _, charge := range page.Charges {
		blockedCapabilities := "none"
		if len(charge.BlockedCapabilities) > 0 {
			blockedCapabilities = strings.Join(charge.BlockedCapabilities, ", ")
		}
		lines = append(lines, fmt.Sprintf("- %s: task=%s entitlement=%s billing=%s included_units=%d overage_units=%d estimated_cost_usd=%.2f overage_cost_usd=%.2f blocked=%s handoff=%s recommendation=%s", charge.RunID, charge.TaskID, charge.EntitlementStatus, charge.BillingModel, charge.IncludedUsageUnits, charge.OverageUsageUnits, charge.EstimatedCostUSD, charge.OverageCostUSD, blockedCapabilities, charge.HandoffTeam, charge.Recommendation))
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderOrchestrationOverviewPage(portfolio OrchestrationPortfolio) string {
	takeover := "none"
	if portfolio.TakeoverQueue != nil {
		takeover = fmt.Sprintf("pending=%d recommendation=%s", portfolio.TakeoverQueue.PendingRequests(), portfolio.TakeoverQueue.Recommendation())
	}
	return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Orchestration Overview · ` + html.EscapeString(portfolio.Name) + `</title>
</head>
<body>
  <h1>Orchestration Overview</h1>
  <p>` + html.EscapeString(portfolio.Name) + ` · ` + html.EscapeString(portfolio.Period) + `</p>
  <div>Total Runs ` + fmt.Sprintf("%d", portfolio.TotalRuns()) + `</div>
  <div>Recommendation ` + html.EscapeString(portfolio.Recommendation()) + `</div>
  <div>Estimated Cost $` + fmt.Sprintf("%.2f", portfolio.TotalEstimatedCostUSD()) + `</div>
  <div>Takeover Queue ` + html.EscapeString(takeover) + `</div>
  <ul>` + renderPortfolioRunItems(portfolio.Canvases) + `</ul>
</body>
</html>
`
}

func RenderBillingEntitlementsPage(page BillingEntitlementsPage) string {
	return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Billing & Entitlements · ` + html.EscapeString(page.WorkspaceName) + `</title>
</head>
<body>
  <h1>` + html.EscapeString(page.WorkspaceName) + `</h1>
  <p>` + html.EscapeString(page.PlanName) + ` plan for ` + html.EscapeString(page.BillingPeriod) + `. Recommendation: ` + html.EscapeString(page.Recommendation()) + `.</p>
  <h2>Charge Feed</h2>
  <ul>` + renderChargeItems(page.Charges) + `</ul>
</body>
</html>
`
}

func workflowConsoleActions(target string, allowRetry bool, retryReason string, allowPause bool, pauseReason string, allowReassign bool, reassignReason string, allowEscalate bool, escalateReason string) []ConsoleAction {
	return []ConsoleAction{
		{ActionID: "drill-down", Label: "Drill Down", Target: target, Enabled: true},
		{ActionID: "export", Label: "Export", Target: target, Enabled: true},
		{ActionID: "add-note", Label: "Add Note", Target: target, Enabled: true},
		{ActionID: "escalate", Label: "Escalate", Target: target, Enabled: allowEscalate, Reason: disabledReason(allowEscalate, escalateReason)},
		{ActionID: "retry", Label: "Retry", Target: target, Enabled: allowRetry, Reason: disabledReason(allowRetry, retryReason)},
		{ActionID: "pause", Label: "Pause", Target: target, Enabled: allowPause, Reason: disabledReason(allowPause, pauseReason)},
		{ActionID: "reassign", Label: "Reassign", Target: target, Enabled: allowReassign, Reason: disabledReason(allowReassign, reassignReason)},
		{ActionID: "audit", Label: "Audit Trail", Target: target, Enabled: true},
	}
}

func defaultCanvasActions(canvas OrchestrationCanvas) []ConsoleAction {
	return workflowConsoleActions(
		canvas.RunID,
		canvas.HandoffStatus != "pending",
		"pending handoff must be resolved before retry",
		canvas.HandoffStatus != "completed",
		"completed handoff runs cannot be paused",
		canvas.HandoffTeam != "none",
		"reassign is available after a handoff exists",
		canvas.UpgradeRequired,
		"escalate when policy requires an entitlement or approval upgrade",
	)
}

func latestNamedAudit(audits []map[string]any, action string) map[string]any {
	for i := len(audits) - 1; i >= 0; i-- {
		if stringValue(audits[i]["action"]) == action {
			return audits[i]
		}
	}
	return nil
}

func latestHandoffAudit(audits []map[string]any) map[string]any {
	for _, action := range []string{"manual.takeover", "flow.handoff", "orchestration.handoff"} {
		if audit := latestNamedAudit(audits, action); audit != nil {
			return audit
		}
	}
	return nil
}

func anyMapSlice(value any) []map[string]any {
	items, ok := value.([]any)
	if !ok {
		if typed, ok := value.([]map[string]any); ok {
			return typed
		}
		return nil
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if mapped, ok := item.(map[string]any); ok {
			out = append(out, mapped)
		}
	}
	return out
}

func mapValue(value any) map[string]any {
	mapped, _ := value.(map[string]any)
	if mapped == nil {
		return map[string]any{}
	}
	return mapped
}

func floatValue(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	default:
		return 0
	}
}

func intValue(value any) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case int64:
		return int(typed)
	default:
		return 0
	}
}

func sortedStringSet(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func formatCountMap(values map[string]int) string {
	if len(values) == 0 {
		return "none"
	}
	keys := sortedMapKeys(values)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, values[key]))
	}
	return strings.Join(parts, " ")
}

func joinListOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func pyBool(value bool) string {
	if value {
		return "True"
	}
	return "False"
}

func round2(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}

func renderPortfolioRunItems(canvases []OrchestrationCanvas) string {
	if len(canvases) == 0 {
		return "<li>None</li>"
	}
	items := make([]string, 0, len(canvases))
	for _, canvas := range canvases {
		items = append(items, "<li><strong>"+html.EscapeString(canvas.RunID)+"</strong> · mode="+html.EscapeString(canvas.CollaborationMode)+" · tier="+html.EscapeString(canvas.Tier)+" · entitlement="+html.EscapeString(canvas.EntitlementStatus)+" · billing="+html.EscapeString(canvas.BillingModel)+" · cost=$"+fmt.Sprintf("%.2f", canvas.EstimatedCostUSD)+" · handoff="+html.EscapeString(canvas.HandoffTeam)+" · recommendation="+html.EscapeString(canvas.Recommendation())+" · actions="+html.EscapeString(RenderConsoleActions(defaultCanvasActions(canvas)))+"</li>")
	}
	return strings.Join(items, "")
}

func renderChargeItems(charges []BillingRunCharge) string {
	if len(charges) == 0 {
		return "<li>None</li>"
	}
	items := make([]string, 0, len(charges))
	for _, charge := range charges {
		items = append(items, "<li><strong>"+html.EscapeString(charge.RunID)+"</strong> · task="+html.EscapeString(charge.TaskID)+" · entitlement="+html.EscapeString(charge.EntitlementStatus)+" · billing="+html.EscapeString(charge.BillingModel)+" · included="+fmt.Sprintf("%d", charge.IncludedUsageUnits)+" · overage="+fmt.Sprintf("%d", charge.OverageUsageUnits)+" · cost=$"+fmt.Sprintf("%.2f", charge.EstimatedCostUSD)+" · recommendation="+html.EscapeString(charge.Recommendation)+"</li>")
	}
	return strings.Join(items, "")
}
