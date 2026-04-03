package reporting

import (
	"fmt"
	"html"
	"sort"
	"strings"

	"bigclaw-go/internal/collaboration"
	"bigclaw-go/internal/observability"
	"bigclaw-go/internal/workflow"
)

type TriageFinding struct {
	RunID      string          `json:"run_id"`
	TaskID     string          `json:"task_id"`
	Source     string          `json:"source"`
	Severity   string          `json:"severity"`
	Owner      string          `json:"owner"`
	Status     string          `json:"status"`
	Reason     string          `json:"reason"`
	NextAction string          `json:"next_action"`
	Actions    []ConsoleAction `json:"actions,omitempty"`
}

type TriageSimilarityEvidence struct {
	RelatedRunID  string  `json:"related_run_id"`
	RelatedTaskID string  `json:"related_task_id"`
	Score         float64 `json:"score"`
	Reason        string  `json:"reason"`
}

type TriageSuggestion struct {
	Label          string                     `json:"label"`
	Action         string                     `json:"action"`
	Owner          string                     `json:"owner"`
	Confidence     float64                    `json:"confidence"`
	Evidence       []TriageSimilarityEvidence `json:"evidence,omitempty"`
	FeedbackStatus string                     `json:"feedback_status"`
}

type TriageInboxItem struct {
	RunID       string             `json:"run_id"`
	TaskID      string             `json:"task_id"`
	Source      string             `json:"source"`
	Status      string             `json:"status"`
	Severity    string             `json:"severity"`
	Owner       string             `json:"owner"`
	Summary     string             `json:"summary"`
	SubmittedAt string             `json:"submitted_at"`
	Suggestions []TriageSuggestion `json:"suggestions,omitempty"`
}

type TriageFeedbackRecord struct {
	RunID     string `json:"run_id"`
	Action    string `json:"action"`
	Decision  string `json:"decision"`
	Actor     string `json:"actor"`
	Notes     string `json:"notes,omitempty"`
	Timestamp string `json:"timestamp"`
}

func NewTriageFeedbackRecord(runID, action, decision, actor, notes string) TriageFeedbackRecord {
	return TriageFeedbackRecord{
		RunID:     runID,
		Action:    action,
		Decision:  decision,
		Actor:     actor,
		Notes:     notes,
		Timestamp: utcNowISO(),
	}
}

type AutoTriageCenter struct {
	Name     string                 `json:"name"`
	Period   string                 `json:"period"`
	Findings []TriageFinding        `json:"findings,omitempty"`
	Inbox    []TriageInboxItem      `json:"inbox,omitempty"`
	Feedback []TriageFeedbackRecord `json:"feedback,omitempty"`
}

func (c AutoTriageCenter) FlaggedRuns() int { return len(c.Findings) }
func (c AutoTriageCenter) InboxSize() int   { return len(c.Inbox) }

func (c AutoTriageCenter) SeverityCounts() map[string]int {
	counts := map[string]int{"critical": 0, "high": 0, "medium": 0}
	for _, finding := range c.Findings {
		counts[finding.Severity]++
	}
	return counts
}

func (c AutoTriageCenter) OwnerCounts() map[string]int {
	counts := map[string]int{"security": 0, "engineering": 0, "operations": 0}
	for _, finding := range c.Findings {
		counts[finding.Owner]++
	}
	return counts
}

func (c AutoTriageCenter) FeedbackCounts() map[string]int {
	counts := map[string]int{"accepted": 0, "rejected": 0, "pending": 0}
	for _, record := range c.Feedback {
		counts[record.Decision]++
	}
	pending := 0
	for _, item := range c.Inbox {
		for _, suggestion := range item.Suggestions {
			if suggestion.FeedbackStatus == "pending" {
				pending++
			}
		}
	}
	counts["pending"] = pending
	return counts
}

func (c AutoTriageCenter) Recommendation() string {
	severity := c.SeverityCounts()
	feedback := c.FeedbackCounts()
	switch {
	case severity["critical"] > 0:
		return "immediate-attention"
	case feedback["rejected"] > feedback["accepted"]:
		return "retune-suggestions"
	case severity["high"] > 0:
		return "review-queue"
	default:
		return "monitor"
	}
}

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
func (q TakeoverQueue) ApprovalCount() int {
	total := 0
	for _, request := range q.Requests {
		total += len(request.RequiredApprovals)
	}
	return total
}
func (q TakeoverQueue) TeamCounts() map[string]int {
	counts := map[string]int{}
	for _, request := range q.Requests {
		counts[request.TargetTeam]++
	}
	return counts
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
	TaskID             string                `json:"task_id"`
	RunID              string                `json:"run_id"`
	CollaborationMode  string                `json:"collaboration_mode"`
	Departments        []string              `json:"departments,omitempty"`
	RequiredApprovals  []string              `json:"required_approvals,omitempty"`
	Tier               string                `json:"tier"`
	UpgradeRequired    bool                  `json:"upgrade_required"`
	BlockedDepartments []string              `json:"blocked_departments,omitempty"`
	HandoffTeam        string                `json:"handoff_team"`
	HandoffStatus      string                `json:"handoff_status"`
	HandoffReason      string                `json:"handoff_reason,omitempty"`
	ActiveTools        []string              `json:"active_tools,omitempty"`
	EntitlementStatus  string                `json:"entitlement_status"`
	BillingModel       string                `json:"billing_model"`
	EstimatedCostUSD   float64               `json:"estimated_cost_usd"`
	IncludedUsageUnits int                   `json:"included_usage_units"`
	OverageUsageUnits  int                   `json:"overage_usage_units"`
	OverageCostUSD     float64               `json:"overage_cost_usd"`
	Actions            []ConsoleAction       `json:"actions,omitempty"`
	Collaboration      *collaboration.Thread `json:"collaboration,omitempty"`
}

func (c OrchestrationCanvas) Recommendation() string {
	switch {
	case c.Collaboration != nil && c.Collaboration.OpenCommentCount() > 0:
		return "resolve-flow-comments"
	case c.HandoffTeam == "security":
		return "review-security-takeover"
	case c.UpgradeRequired:
		return "resolve-entitlement-gap"
	case c.OverageCostUSD > 0:
		return "review-billing-overage"
	case len(c.Departments) > 1:
		return "continue-cross-team-execution"
	default:
		return "monitor"
	}
}

type OrchestrationPortfolio struct {
	Name          string                `json:"name"`
	Period        string                `json:"period"`
	Canvases      []OrchestrationCanvas `json:"canvases,omitempty"`
	TakeoverQueue *TakeoverQueue        `json:"takeover_queue,omitempty"`
}

func (p OrchestrationPortfolio) TotalRuns() int { return len(p.Canvases) }
func (p OrchestrationPortfolio) CollaborationModes() map[string]int {
	counts := map[string]int{}
	for _, canvas := range p.Canvases {
		counts[canvas.CollaborationMode]++
	}
	return counts
}
func (p OrchestrationPortfolio) TierCounts() map[string]int {
	counts := map[string]int{}
	for _, canvas := range p.Canvases {
		counts[canvas.Tier]++
	}
	return counts
}
func (p OrchestrationPortfolio) EntitlementCounts() map[string]int {
	counts := map[string]int{}
	for _, canvas := range p.Canvases {
		counts[canvas.EntitlementStatus]++
	}
	return counts
}
func (p OrchestrationPortfolio) BillingModelCounts() map[string]int {
	counts := map[string]int{}
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
	return roundTenth(total*10) / 10
}
func (p OrchestrationPortfolio) TotalOverageCostUSD() float64 {
	total := 0.0
	for _, canvas := range p.Canvases {
		total += canvas.OverageCostUSD
	}
	return roundTenth(total*10) / 10
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
func (p BillingEntitlementsPage) TotalEstimatedCostUSD() float64 {
	total := 0.0
	for _, charge := range p.Charges {
		total += charge.EstimatedCostUSD
	}
	return roundTenth(total*10) / 10
}
func (p BillingEntitlementsPage) TotalOverageCostUSD() float64 {
	total := 0.0
	for _, charge := range p.Charges {
		total += charge.OverageCostUSD
	}
	return roundTenth(total*10) / 10
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
	counts := map[string]int{}
	for _, charge := range p.Charges {
		counts[charge.BillingModel]++
	}
	return counts
}
func (p BillingEntitlementsPage) EntitlementCounts() map[string]int {
	counts := map[string]int{}
	for _, charge := range p.Charges {
		counts[charge.EntitlementStatus]++
	}
	return counts
}
func (p BillingEntitlementsPage) BlockedCapabilities() []string {
	seen := map[string]struct{}{}
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
	switch {
	case p.UpgradeRequiredCount() > 0:
		return "resolve-plan-gaps"
	case p.TotalOverageCostUSD() > 0:
		return "optimize-billed-usage"
	default:
		for _, charge := range p.Charges {
			if charge.HandoffTeam != "none" {
				return "monitor-shared-capacity"
			}
		}
		return "healthy"
	}
}

func buildSurfaceActions(target string, allowRetry bool, retryReason string, allowPause bool, pauseReason string, allowReassign bool, reassignReason string, allowEscalate bool, escalateReason string) []ConsoleAction {
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

func renderSurfaceActions(actions []ConsoleAction) string {
	if len(actions) == 0 {
		return "none"
	}
	items := make([]string, 0, len(actions))
	for _, action := range actions {
		detail := fmt.Sprintf("%s [%s] state=%s target=%s", action.Label, action.ActionID, action.State(), action.Target)
		if strings.TrimSpace(action.Reason) != "" {
			detail += " reason=" + action.Reason
		}
		items = append(items, detail)
	}
	return strings.Join(items, "; ")
}

func BuildAutoTriageCenter(runs []observability.TaskRun, name, period string, feedback []TriageFeedbackRecord) AutoTriageCenter {
	findings := make([]TriageFinding, 0)
	inbox := make([]TriageInboxItem, 0)
	for _, run := range runs {
		if !runRequiresTriage(run) {
			continue
		}
		severity := triageSeverity(run)
		owner := triageOwner(run)
		reason := triageReason(run)
		next := triageNextAction(severity, owner)
		suggestions := buildTriageSuggestions(run, runs, severity, owner, feedback)
		findings = append(findings, TriageFinding{
			RunID:      run.RunID,
			TaskID:     run.TaskID,
			Source:     taskString(run, "source"),
			Severity:   severity,
			Owner:      owner,
			Status:     run.Status,
			Reason:     reason,
			NextAction: next,
			Actions: buildSurfaceActions(
				run.RunID,
				severity == "critical" && owner != "security",
				"retry available after owner review",
				run.Status != "failed" && run.Status != "completed" && run.Status != "approved",
				"completed or failed runs cannot be paused",
				owner != "security",
				"security-owned findings stay with the security queue",
				true,
				"",
			),
		})
		inbox = append(inbox, TriageInboxItem{
			RunID:       run.RunID,
			TaskID:      run.TaskID,
			Source:      taskString(run, "source"),
			Status:      run.Status,
			Severity:    severity,
			Owner:       owner,
			Summary:     reason,
			SubmittedAt: "",
			Suggestions: suggestions,
		})
	}
	sort.Slice(findings, func(i, j int) bool {
		return triageRank(findings[i].Severity) < triageRank(findings[j].Severity) ||
			(triageRank(findings[i].Severity) == triageRank(findings[j].Severity) && findings[i].RunID < findings[j].RunID)
	})
	sort.Slice(inbox, func(i, j int) bool {
		return triageRank(inbox[i].Severity) < triageRank(inbox[j].Severity) ||
			(triageRank(inbox[i].Severity) == triageRank(inbox[j].Severity) && inbox[i].RunID < inbox[j].RunID)
	})
	return AutoTriageCenter{Name: firstNonEmpty(name, "Auto Triage Center"), Period: firstNonEmpty(period, "current"), Findings: findings, Inbox: inbox, Feedback: feedback}
}

func RenderAutoTriageCenterReport(center AutoTriageCenter, totalRuns int, view *SharedViewContext) string {
	severity := center.SeverityCounts()
	owners := center.OwnerCounts()
	feedback := center.FeedbackCounts()
	lines := []string{
		"# Auto Triage Center",
		"",
		"- Center: " + center.Name,
		"- Period: " + center.Period,
		fmt.Sprintf("- Flagged Runs: %d", center.FlaggedRuns()),
		fmt.Sprintf("- Inbox Size: %d", center.InboxSize()),
		fmt.Sprintf("- Total Runs: %d", totalRuns),
		"- Recommendation: " + center.Recommendation(),
		fmt.Sprintf("- Severity Mix: critical=%d high=%d medium=%d", severity["critical"], severity["high"], severity["medium"]),
		fmt.Sprintf("- Owner Mix: security=%d engineering=%d operations=%d", owners["security"], owners["engineering"], owners["operations"]),
		fmt.Sprintf("- Feedback Loop: accepted=%d rejected=%d pending=%d", feedback["accepted"], feedback["rejected"], feedback["pending"]),
		"",
		"## Queue",
		"",
	}
	lines = append(lines, RenderSharedViewContext(view)...)
	if len(center.Findings) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, finding := range center.Findings {
			lines = append(lines, fmt.Sprintf("- %s: severity=%s owner=%s status=%s task=%s reason=%s next=%s actions=%s", finding.RunID, finding.Severity, finding.Owner, finding.Status, finding.TaskID, finding.Reason, finding.NextAction, renderSurfaceActions(finding.Actions)))
		}
	}
	lines = append(lines, "", "## Inbox", "")
	if len(center.Inbox) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, item := range center.Inbox {
			summaries := make([]string, 0, len(item.Suggestions))
			evidence := make([]string, 0)
			for _, suggestion := range item.Suggestions {
				summaries = append(summaries, fmt.Sprintf("%s(%s, confidence=%.2f)", suggestion.Action, suggestion.FeedbackStatus, suggestion.Confidence))
				for _, match := range suggestion.Evidence {
					evidence = append(evidence, fmt.Sprintf("%s:%.2f", match.RelatedRunID, match.Score))
				}
			}
			lines = append(lines, fmt.Sprintf("- %s: severity=%s owner=%s status=%s summary=%s suggestions=%s similar=%s", item.RunID, item.Severity, item.Owner, item.Status, item.Summary, firstNonEmpty(strings.Join(summaries, "; "), "none"), firstNonEmpty(strings.Join(evidence, ", "), "none")))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func BuildTakeoverQueueFromLedger(entries []map[string]any, name, period string) TakeoverQueue {
	requests := make([]TakeoverRequest, 0)
	for _, entry := range entries {
		handoff := latestHandoffAudit(entry["audits"])
		if handoff == nil {
			continue
		}
		details, _ := handoff["details"].(map[string]any)
		targetTeam := stringValue(details["target_team"])
		requests = append(requests, TakeoverRequest{
			RunID:             stringValue(entry["run_id"]),
			TaskID:            stringValue(entry["task_id"]),
			Source:            stringValue(entry["source"]),
			TargetTeam:        firstNonEmpty(targetTeam, "operations"),
			Status:            firstNonEmpty(stringValue(handoff["outcome"]), "pending"),
			Reason:            firstNonEmpty(stringValue(details["reason"]), stringValue(entry["summary"]), "handoff requested"),
			RequiredApprovals: stringSlice(details["required_approvals"]),
			Actions: buildSurfaceActions(
				stringValue(entry["run_id"]),
				false,
				"retry is blocked while takeover is pending",
				firstNonEmpty(stringValue(handoff["outcome"]), "pending") == "pending",
				"only pending takeovers can be paused",
				true,
				"",
				targetTeam != "security",
				"security takeovers are already escalated",
			),
		})
	}
	sort.Slice(requests, func(i, j int) bool {
		if requests[i].TargetTeam == requests[j].TargetTeam {
			return requests[i].RunID < requests[j].RunID
		}
		return requests[i].TargetTeam < requests[j].TargetTeam
	})
	return TakeoverQueue{Name: firstNonEmpty(name, "Human Takeover Queue"), Period: firstNonEmpty(period, "current"), Requests: requests}
}

func RenderTakeoverQueueReport(queue TakeoverQueue, totalRuns int, view *SharedViewContext) string {
	teamCounts := queue.TeamCounts()
	teams := sortedMapKeys(teamCounts)
	mixParts := make([]string, 0, len(teams))
	for _, team := range teams {
		mixParts = append(mixParts, fmt.Sprintf("%s=%d", team, teamCounts[team]))
	}
	lines := []string{
		"# Human Takeover Queue",
		"",
		"- Queue: " + queue.Name,
		"- Period: " + queue.Period,
		fmt.Sprintf("- Pending Requests: %d", queue.PendingRequests()),
		fmt.Sprintf("- Total Runs: %d", totalRuns),
		"- Recommendation: " + queue.Recommendation(),
		"- Team Mix: " + firstNonEmpty(strings.Join(mixParts, " "), "none"),
		fmt.Sprintf("- Required Approvals: %d", queue.ApprovalCount()),
		"",
		"## Requests",
		"",
	}
	lines = append(lines, RenderSharedViewContext(view)...)
	if len(queue.Requests) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, request := range queue.Requests {
			lines = append(lines, fmt.Sprintf("- %s: team=%s status=%s task=%s approvals=%s reason=%s actions=%s", request.RunID, request.TargetTeam, request.Status, request.TaskID, firstNonEmpty(strings.Join(request.RequiredApprovals, ","), "none"), request.Reason, renderSurfaceActions(request.Actions)))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func BuildOrchestrationCanvas(run observability.TaskRun, plan workflow.OrchestrationPlan, policy *workflow.OrchestrationPolicyDecision, handoff *workflow.HandoffRequest) OrchestrationCanvas {
	canvas := OrchestrationCanvas{
		TaskID:            run.TaskID,
		RunID:             run.RunID,
		CollaborationMode: plan.CollaborationMode,
		Departments:       plan.Departments(),
		RequiredApprovals: plan.RequiredApprovals(),
		Tier:              "standard",
		HandoffTeam:       "none",
		HandoffStatus:     "none",
		EntitlementStatus: "included",
		BillingModel:      "standard-included",
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
	if handoff != nil {
		canvas.HandoffTeam = handoff.TargetTeam
		canvas.HandoffStatus = firstNonEmpty(handoff.Status, "pending")
		canvas.HandoffReason = handoff.Reason
		if len(handoff.RequiredApprovals) > 0 {
			canvas.RequiredApprovals = append([]string(nil), handoff.RequiredApprovals...)
		}
	}
	canvas.ActiveTools = activeToolsFromAudits(run.Audits)
	canvas.Actions = buildSurfaceActions(
		run.RunID,
		handoff == nil || handoff.Status != "pending",
		"pending handoff must be resolved before retry",
		run.Status != "failed" && run.Status != "completed" && run.Status != "approved",
		"completed or failed runs cannot be paused",
		handoff != nil,
		"reassign is available after a handoff exists",
		policy != nil && policy.UpgradeRequired,
		"escalate when policy requires an entitlement or approval upgrade",
	)
	canvas.Collaboration = collaboration.BuildCollaborationThreadFromAudits(auditMaps(run.Audits), "flow", run.RunID)
	return canvas
}

func BuildOrchestrationCanvasFromLedgerEntry(entry map[string]any) OrchestrationCanvas {
	audits, _ := entry["audits"].([]any)
	planAudit := latestNamedAudit(audits, "orchestration.plan")
	policyAudit := latestNamedAudit(audits, "orchestration.policy")
	handoffAudit := latestHandoffAudit(audits)
	planDetails, _ := mapValue(planAudit, "details")
	policyDetails, _ := mapValue(policyAudit, "details")
	handoffDetails, _ := mapValue(handoffAudit, "details")
	canvas := OrchestrationCanvas{
		TaskID:             stringValue(entry["task_id"]),
		RunID:              stringValue(entry["run_id"]),
		CollaborationMode:  firstNonEmpty(stringValue(planDetails["collaboration_mode"]), "single-team"),
		Departments:        stringSlice(planDetails["departments"]),
		RequiredApprovals:  stringSlice(planDetails["approvals"]),
		Tier:               firstNonEmpty(stringValue(policyDetails["tier"]), "standard"),
		UpgradeRequired:    stringValue(mapValueString(policyAudit, "outcome")) == "upgrade-required",
		BlockedDepartments: stringSlice(policyDetails["blocked_departments"]),
		HandoffTeam:        firstNonEmpty(stringValue(handoffDetails["target_team"]), "none"),
		HandoffStatus:      firstNonEmpty(stringValue(mapValueString(handoffAudit, "outcome")), "none"),
		HandoffReason:      stringValue(handoffDetails["reason"]),
		EntitlementStatus:  firstNonEmpty(stringValue(policyDetails["entitlement_status"]), "included"),
		BillingModel:       firstNonEmpty(stringValue(policyDetails["billing_model"]), "standard-included"),
		EstimatedCostUSD:   floatValue(policyDetails["estimated_cost_usd"]),
		IncludedUsageUnits: intValue(policyDetails["included_usage_units"]),
		OverageUsageUnits:  intValue(policyDetails["overage_usage_units"]),
		OverageCostUSD:     floatValue(policyDetails["overage_cost_usd"]),
		ActiveTools:        activeToolsFromLedgerAudits(audits),
		Collaboration:      collaboration.BuildCollaborationThreadFromAudits(auditsToMaps(audits), "flow", stringValue(entry["run_id"])),
	}
	canvas.Actions = buildSurfaceActions(
		canvas.RunID,
		canvas.HandoffStatus != "pending",
		"pending handoff must be resolved before retry",
		canvas.HandoffStatus != "completed",
		"completed handoff runs cannot be paused",
		handoffAudit != nil,
		"reassign is available after a handoff exists",
		canvas.UpgradeRequired,
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
		"- Departments: " + firstNonEmpty(strings.Join(canvas.Departments, ", "), "none"),
		"- Required Approvals: " + firstNonEmpty(strings.Join(canvas.RequiredApprovals, ", "), "none"),
		"- Tier: " + canvas.Tier,
		fmt.Sprintf("- Upgrade Required: %t", canvas.UpgradeRequired),
		"- Entitlement Status: " + canvas.EntitlementStatus,
		"- Billing Model: " + canvas.BillingModel,
		"- Blocked Departments: " + firstNonEmpty(strings.Join(canvas.BlockedDepartments, ", "), "none"),
		"- Handoff Team: " + canvas.HandoffTeam,
		"- Handoff Status: " + canvas.HandoffStatus,
		"- Recommendation: " + canvas.Recommendation(),
		"",
		"## Execution Context",
		"",
		"- Active Tools: " + firstNonEmpty(strings.Join(canvas.ActiveTools, ", "), "none"),
		fmt.Sprintf("- Estimated Cost (USD): %.2f", canvas.EstimatedCostUSD),
		fmt.Sprintf("- Included Usage Units: %d", canvas.IncludedUsageUnits),
		fmt.Sprintf("- Overage Usage Units: %d", canvas.OverageUsageUnits),
		fmt.Sprintf("- Overage Cost (USD): %.2f", canvas.OverageCostUSD),
		"- Handoff Reason: " + firstNonEmpty(canvas.HandoffReason, "none"),
		"",
		"## Actions",
		"",
		"- " + renderSurfaceActions(canvas.Actions),
	}
	lines = append(lines, collaboration.RenderCollaborationLines(canvas.Collaboration)...)
	return strings.Join(lines, "\n") + "\n"
}

func BuildOrchestrationPortfolio(canvases []OrchestrationCanvas, name, period string, takeoverQueue *TakeoverQueue) OrchestrationPortfolio {
	normalized := make([]OrchestrationCanvas, 0, len(canvases))
	for _, canvas := range canvases {
		if len(canvas.Actions) == 0 {
			canvas.Actions = buildSurfaceActions(canvas.RunID, canvas.HandoffStatus != "pending", "pending handoff must be resolved before retry", canvas.HandoffStatus != "completed", "completed handoff runs cannot be paused", canvas.HandoffTeam != "none", "reassign is available after a handoff exists", canvas.UpgradeRequired, "escalate when policy requires an entitlement or approval upgrade")
		}
		normalized = append(normalized, canvas)
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i].RunID < normalized[j].RunID })
	return OrchestrationPortfolio{Name: firstNonEmpty(name, "Cross-Department Orchestration"), Period: firstNonEmpty(period, "current"), Canvases: normalized, TakeoverQueue: takeoverQueue}
}

func RenderOrchestrationPortfolioReport(portfolio OrchestrationPortfolio, view *SharedViewContext) string {
	lines := []string{
		"# Orchestration Portfolio Report",
		"",
		"- Portfolio: " + portfolio.Name,
		"- Period: " + portfolio.Period,
		fmt.Sprintf("- Total Runs: %d", portfolio.TotalRuns()),
		"- Recommendation: " + portfolio.Recommendation(),
		"- Collaboration Mix: " + countsString(portfolio.CollaborationModes()),
		"- Tier Mix: " + countsString(portfolio.TierCounts()),
		"- Entitlement Mix: " + countsString(portfolio.EntitlementCounts()),
		"- Billing Models: " + countsString(portfolio.BillingModelCounts()),
		fmt.Sprintf("- Upgrade Required Count: %d", portfolio.UpgradeRequiredCount()),
		fmt.Sprintf("- Estimated Cost (USD): %.2f", portfolio.TotalEstimatedCostUSD()),
		fmt.Sprintf("- Overage Cost (USD): %.2f", portfolio.TotalOverageCostUSD()),
		fmt.Sprintf("- Active Handoffs: %d", portfolio.ActiveHandoffs()),
		"- Takeover Queue: " + func() string {
			if portfolio.TakeoverQueue == nil {
				return "none"
			}
			return fmt.Sprintf("pending=%d recommendation=%s", portfolio.TakeoverQueue.PendingRequests(), portfolio.TakeoverQueue.Recommendation())
		}(),
		"",
		"## Runs",
		"",
	}
	lines = append(lines, RenderSharedViewContext(view)...)
	if len(portfolio.Canvases) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, canvas := range portfolio.Canvases {
			collab := "comments=0 decisions=0"
			if canvas.Collaboration != nil {
				collab = fmt.Sprintf("comments=%d decisions=%d", len(canvas.Collaboration.Comments), len(canvas.Collaboration.Decisions))
			}
			lines = append(lines, fmt.Sprintf("- %s: mode=%s tier=%s entitlement=%s billing=%s estimated_cost_usd=%.2f overage_cost_usd=%.2f upgrade_required=%t handoff=%s collaboration=%s recommendation=%s actions=%s", canvas.RunID, canvas.CollaborationMode, canvas.Tier, canvas.EntitlementStatus, canvas.BillingModel, canvas.EstimatedCostUSD, canvas.OverageCostUSD, canvas.UpgradeRequired, canvas.HandoffTeam, collab, canvas.Recommendation(), renderSurfaceActions(canvas.Actions)))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderOrchestrationOverviewPage(portfolio OrchestrationPortfolio) string {
	runs := make([]string, 0, len(portfolio.Canvases))
	for _, canvas := range portfolio.Canvases {
		runs = append(runs, fmt.Sprintf("<li><strong>%s</strong> · mode=%s · tier=%s · entitlement=%s · billing=%s · cost=$%.2f · handoff=%s · recommendation=%s · actions=%s</li>", html.EscapeString(canvas.RunID), html.EscapeString(canvas.CollaborationMode), html.EscapeString(canvas.Tier), html.EscapeString(canvas.EntitlementStatus), html.EscapeString(canvas.BillingModel), canvas.EstimatedCostUSD, html.EscapeString(canvas.HandoffTeam), html.EscapeString(canvas.Recommendation()), html.EscapeString(renderSurfaceActions(canvas.Actions))))
	}
	takeover := "none"
	if portfolio.TakeoverQueue != nil {
		takeover = fmt.Sprintf("pending=%d recommendation=%s", portfolio.TakeoverQueue.PendingRequests(), portfolio.TakeoverQueue.Recommendation())
	}
	return fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Orchestration Overview · %s</title>
</head>
<body>
  <h1>Orchestration Overview</h1>
  <p>%s · %s</p>
  <div>Estimated Cost: $%.2f</div>
  <div>Takeover Queue: %s</div>
  <ul>%s</ul>
</body>
</html>
`, html.EscapeString(portfolio.Name), html.EscapeString(portfolio.Name), html.EscapeString(portfolio.Period), portfolio.TotalEstimatedCostUSD(), html.EscapeString(takeover), strings.Join(runs, ""))
}

func BuildOrchestrationPortfolioFromLedger(entries []map[string]any, name, period string) OrchestrationPortfolio {
	canvases := make([]OrchestrationCanvas, 0)
	for _, entry := range entries {
		if latestNamedAudit(entry["audits"].([]any), "orchestration.plan") == nil {
			continue
		}
		canvases = append(canvases, BuildOrchestrationCanvasFromLedgerEntry(entry))
	}
	queue := BuildTakeoverQueueFromLedger(entries, name+" Takeovers", period)
	return BuildOrchestrationPortfolio(canvases, name, period, &queue)
}

func BuildBillingEntitlementsPage(portfolio OrchestrationPortfolio, workspaceName, planName, billingPeriod string) BillingEntitlementsPage {
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
	return BillingEntitlementsPage{WorkspaceName: firstNonEmpty(workspaceName, "BigClaw Cloud"), PlanName: firstNonEmpty(planName, "Standard"), BillingPeriod: firstNonEmpty(billingPeriod, portfolio.Period), Charges: charges}
}

func RenderBillingEntitlementsReport(page BillingEntitlementsPage, view *SharedViewContext) string {
	lines := []string{
		"# Billing & Entitlements Report",
		"",
		"- Workspace: " + page.WorkspaceName,
		"- Plan: " + page.PlanName,
		"- Billing Period: " + page.BillingPeriod,
		fmt.Sprintf("- Runs: %d", page.RunCount()),
		"- Recommendation: " + page.Recommendation(),
		"- Entitlement Mix: " + countsString(page.EntitlementCounts()),
		"- Billing Models: " + countsString(page.BillingModelCounts()),
		fmt.Sprintf("- Included Usage Units: %d", page.TotalIncludedUsageUnits()),
		fmt.Sprintf("- Overage Usage Units: %d", page.TotalOverageUsageUnits()),
		fmt.Sprintf("- Estimated Cost (USD): %.2f", page.TotalEstimatedCostUSD()),
		fmt.Sprintf("- Overage Cost (USD): %.2f", page.TotalOverageCostUSD()),
		fmt.Sprintf("- Upgrade Required Count: %d", page.UpgradeRequiredCount()),
		"- Blocked Capabilities: " + firstNonEmpty(strings.Join(page.BlockedCapabilities(), ", "), "none"),
		"",
		"## Charges",
		"",
	}
	lines = append(lines, RenderSharedViewContext(view)...)
	if len(page.Charges) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, charge := range page.Charges {
			lines = append(lines, fmt.Sprintf("- %s: task=%s entitlement=%s billing=%s included_units=%d overage_units=%d estimated_cost_usd=%.2f overage_cost_usd=%.2f blocked=%s handoff=%s recommendation=%s", charge.RunID, charge.TaskID, charge.EntitlementStatus, charge.BillingModel, charge.IncludedUsageUnits, charge.OverageUsageUnits, charge.EstimatedCostUSD, charge.OverageCostUSD, firstNonEmpty(strings.Join(charge.BlockedCapabilities, ", "), "none"), charge.HandoffTeam, charge.Recommendation))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderBillingEntitlementsPage(page BillingEntitlementsPage) string {
	charges := make([]string, 0, len(page.Charges))
	for _, charge := range page.Charges {
		charges = append(charges, fmt.Sprintf("<li><strong>%s</strong> · task=%s · entitlement=%s · billing=%s · included=%d · overage=%d · cost=$%.2f · recommendation=%s</li>", html.EscapeString(charge.RunID), html.EscapeString(charge.TaskID), html.EscapeString(charge.EntitlementStatus), html.EscapeString(charge.BillingModel), charge.IncludedUsageUnits, charge.OverageUsageUnits, charge.EstimatedCostUSD, html.EscapeString(charge.Recommendation)))
	}
	return fmt.Sprintf(`<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>Billing & Entitlements · %s</title></head>
<body>
<h1>%s</h1>
<p>%s plan for %s</p>
<h2>Charge Feed</h2>
<ul>%s</ul>
</body>
</html>`, html.EscapeString(page.WorkspaceName), html.EscapeString(page.WorkspaceName), html.EscapeString(page.PlanName), html.EscapeString(page.BillingPeriod), strings.Join(charges, ""))
}

func BuildBillingEntitlementsPageFromLedger(entries []map[string]any, workspaceName, planName, billingPeriod string) BillingEntitlementsPage {
	portfolio := BuildOrchestrationPortfolioFromLedger(entries, workspaceName, billingPeriod)
	return BuildBillingEntitlementsPage(portfolio, workspaceName, planName, billingPeriod)
}

func countsString(counts map[string]int) string {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, counts[key]))
	}
	return firstNonEmpty(strings.Join(parts, " "), "none")
}

func triageRank(severity string) int {
	switch severity {
	case "critical":
		return 0
	case "high":
		return 1
	default:
		return 2
	}
}

func runRequiresTriage(run observability.TaskRun) bool {
	if run.Status == "failed" || run.Status == "needs-approval" {
		return true
	}
	for _, trace := range run.Traces {
		if trace.Status == "pending" || trace.Status == "error" || trace.Status == "failed" {
			return true
		}
	}
	for _, audit := range run.Audits {
		if audit.Outcome == "pending" || audit.Outcome == "failed" || audit.Outcome == "rejected" {
			return true
		}
	}
	return false
}

func triageSeverity(run observability.TaskRun) string {
	if run.Status == "failed" {
		return "critical"
	}
	for _, trace := range run.Traces {
		if trace.Status == "error" || trace.Status == "failed" {
			return "critical"
		}
	}
	for _, audit := range run.Audits {
		if audit.Outcome == "failed" || audit.Outcome == "rejected" {
			return "critical"
		}
	}
	if run.Status == "needs-approval" {
		return "high"
	}
	return "medium"
}

func triageOwner(run observability.TaskRun) string {
	var evidence []string
	evidence = append(evidence, strings.ToLower(taskString(run, "title")), strings.ToLower(run.Medium), strings.ToLower(run.Outcome), strings.ToLower(run.Status))
	for _, trace := range run.Traces {
		evidence = append(evidence, strings.ToLower(trace.Span), strings.ToLower(trace.Status))
	}
	for _, audit := range run.Audits {
		evidence = append(evidence, strings.ToLower(audit.Outcome), strings.ToLower(stringValue(audit.Details["reason"])))
	}
	text := strings.Join(evidence, " ")
	switch {
	case strings.Contains(text, "security") || strings.Contains(text, "high-risk"):
		return "security"
	case run.Medium == "browser":
		return "engineering"
	default:
		return "operations"
	}
}

func triageReason(run observability.TaskRun) string {
	for _, audit := range run.Audits {
		if reason := stringValue(audit.Details["reason"]); reason != "" {
			return reason
		}
	}
	return firstNonEmpty(run.Outcome, run.Status)
}

func triageNextAction(severity, owner string) string {
	if severity == "critical" {
		if owner == "engineering" {
			return "replay run and inspect tool failures"
		}
		if owner == "security" {
			return "page security reviewer and block rollout"
		}
		return "open incident review and coordinate response"
	}
	if owner == "security" {
		return "request approval and queue security review"
	}
	if owner == "engineering" {
		return "inspect execution evidence and retry when safe"
	}
	return "confirm owner and clear pending workflow gate"
}

func buildTriageSuggestions(run observability.TaskRun, runs []observability.TaskRun, severity, owner string, feedback []TriageFeedbackRecord) []TriageSuggestion {
	action := triageNextAction(severity, owner)
	label := "workflow follow-up"
	if severity == "critical" && owner == "engineering" {
		label = "replay candidate"
	} else if owner == "security" {
		label = "approval review"
	}
	evidence := similarityEvidence(run, runs)
	confidence := 0.55
	if run.Status != "needs-approval" && run.Status != "failed" {
		confidence = 0.45
	}
	if len(evidence) > 0 {
		if candidate := 0.45 + evidence[0].Score/2; candidate > confidence {
			confidence = candidate
		}
		if confidence > 0.95 {
			confidence = 0.95
		}
	}
	feedbackStatus := "pending"
	for i := len(feedback) - 1; i >= 0; i-- {
		if feedback[i].RunID == run.RunID && feedback[i].Action == action {
			feedbackStatus = feedback[i].Decision
			break
		}
	}
	return []TriageSuggestion{{
		Label:          label,
		Action:         action,
		Owner:          owner,
		Confidence:     roundTenth(confidence*100) / 100,
		Evidence:       evidence,
		FeedbackStatus: feedbackStatus,
	}}
}

func similarityEvidence(run observability.TaskRun, runs []observability.TaskRun) []TriageSimilarityEvidence {
	matches := make([]TriageSimilarityEvidence, 0)
	for _, candidate := range runs {
		if candidate.RunID == run.RunID {
			continue
		}
		score := runSimilarityScore(run, candidate)
		if score < 0.35 {
			continue
		}
		matches = append(matches, TriageSimilarityEvidence{
			RelatedRunID:  candidate.RunID,
			RelatedTaskID: candidate.TaskID,
			Score:         roundTenth(score*100) / 100,
			Reason:        similarityReason(run, candidate),
		})
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score == matches[j].Score {
			return matches[i].RelatedRunID < matches[j].RelatedRunID
		}
		return matches[i].Score > matches[j].Score
	})
	if len(matches) > 2 {
		matches = matches[:2]
	}
	return matches
}

func runSimilarityScore(run, candidate observability.TaskRun) float64 {
	score := 0.0
	if run.Status == candidate.Status {
		score += 0.45
	}
	if triageOwner(run) == triageOwner(candidate) {
		score += 0.2
	}
	if triageReason(run) == triageReason(candidate) {
		score += 0.25
	}
	if run.Medium == candidate.Medium {
		score += 0.1
	}
	if score > 1.0 {
		score = 1.0
	}
	return score
}

func similarityReason(run, candidate observability.TaskRun) string {
	parts := make([]string, 0)
	if run.Status == candidate.Status {
		parts = append(parts, "shared status "+run.Status)
	}
	if triageOwner(run) == triageOwner(candidate) {
		parts = append(parts, "shared owner "+triageOwner(run))
	}
	if triageReason(run) == triageReason(candidate) {
		parts = append(parts, "matching failure reason")
	}
	return firstNonEmpty(strings.Join(parts, ", "), "similar execution trail")
}

func activeToolsFromAudits(audits []observability.AuditEntry) []string {
	set := map[string]struct{}{}
	for _, audit := range audits {
		if audit.Action == "tool.invoke" {
			if tool := stringValue(audit.Details["tool"]); tool != "" {
				set[tool] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func activeToolsFromLedgerAudits(audits []any) []string {
	set := map[string]struct{}{}
	for _, item := range audits {
		audit, _ := item.(map[string]any)
		if stringValue(audit["action"]) == "tool.invoke" {
			details, _ := audit["details"].(map[string]any)
			if tool := stringValue(details["tool"]); tool != "" {
				set[tool] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func latestNamedAudit(audits []any, action string) map[string]any {
	for i := len(audits) - 1; i >= 0; i-- {
		audit, _ := audits[i].(map[string]any)
		if stringValue(audit["action"]) == action {
			return audit
		}
	}
	return nil
}

func latestHandoffAudit(audits any) map[string]any {
	items, _ := audits.([]any)
	for _, action := range []string{"orchestration.handoff", "flow.handoff", "manual.takeover"} {
		if audit := latestNamedAudit(items, action); audit != nil {
			return audit
		}
	}
	return nil
}

func taskString(run observability.TaskRun, key string) string {
	if run.Task == nil {
		return ""
	}
	return stringValue(run.Task[key])
}

func auditMaps(audits []observability.AuditEntry) []map[string]any {
	out := make([]map[string]any, 0, len(audits))
	for _, audit := range audits {
		out = append(out, audit.ToMap())
	}
	return out
}

func auditsToMaps(audits []any) []map[string]any {
	out := make([]map[string]any, 0, len(audits))
	for _, audit := range audits {
		if item, ok := audit.(map[string]any); ok {
			out = append(out, item)
		}
	}
	return out
}

func mapValue(audit map[string]any, key string) (map[string]any, bool) {
	if audit == nil {
		return nil, false
	}
	value, _ := audit[key].(map[string]any)
	return value, value != nil
}

func mapValueString(audit map[string]any, key string) any {
	if audit == nil {
		return ""
	}
	return audit[key]
}

func stringValue(value any) string {
	switch current := value.(type) {
	case string:
		return current
	case fmt.Stringer:
		return current.String()
	default:
		return ""
	}
}

func stringSlice(value any) []string {
	switch items := value.(type) {
	case []string:
		return append([]string(nil), items...)
	case []any:
		out := make([]string, 0, len(items))
		for _, item := range items {
			if text := strings.TrimSpace(stringValue(item)); text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func intValue(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func floatValue(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}
