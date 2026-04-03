package reporting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bigclaw-go/internal/observability"
	"bigclaw-go/internal/workflow"
)

func makeSharedView(resultCount int, loading bool, errors []string, partialData []string) *SharedViewContext {
	return &SharedViewContext{
		Filters: []SharedViewFilter{
			{Label: "Team", Value: "engineering"},
			{Label: "Window", Value: "2026-03-10"},
		},
		ResultCount: resultCount,
		Loading:     loading,
		Errors:      append([]string(nil), errors...),
		PartialData: append([]string(nil), partialData...),
		LastUpdated: "2026-03-11T09:00:00Z",
	}
}

func TestWriteReportAndConsoleActionState(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "report.md")
	content := RenderIssueValidationReport("BIG-101", "v0.1", "sandbox", "pass")
	if err := WriteReport(out, content); err != nil {
		t.Fatalf("write report: %v", err)
	}
	body, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(body), "BIG-101") || !strings.Contains(string(body), "pass") {
		t.Fatalf("unexpected report body: %s", string(body))
	}

	enabled := ConsoleAction{ActionID: "retry", Label: "Retry", Target: "run-1", Enabled: true}
	disabled := ConsoleAction{ActionID: "pause", Label: "Pause", Target: "run-1", Enabled: false, Reason: "already completed"}
	if enabled.State() != "enabled" || disabled.State() != "disabled" {
		t.Fatalf("unexpected console action states: %+v %+v", enabled, disabled)
	}
}

func TestAutoTriageCenterPrioritizesFailedAndPendingRuns(t *testing.T) {
	approvalRun := observability.TaskRun{
		TaskID:  "OPE-76-risk",
		RunID:   "run-risk",
		Medium:  "vm",
		Status:  "needs-approval",
		Outcome: "requires approval for high-risk task",
		Task:    map[string]any{"source": "linear", "title": "Prod approval"},
		Traces:  []observability.TraceEntry{{Span: "scheduler.decide", Status: "pending"}},
		Audits:  []observability.AuditEntry{{Action: "scheduler.decision", Actor: "scheduler", Outcome: "pending", Details: map[string]any{"reason": "requires approval for high-risk task"}}},
	}
	failedRun := observability.TaskRun{
		TaskID:  "OPE-76-browser",
		RunID:   "run-browser",
		Medium:  "browser",
		Status:  "failed",
		Outcome: "browser session crashed",
		Task:    map[string]any{"source": "linear", "title": "Replay browser task"},
		Traces:  []observability.TraceEntry{{Span: "runtime.execute", Status: "failed"}},
		Audits:  []observability.AuditEntry{{Action: "runtime.execute", Actor: "worker", Outcome: "failed", Details: map[string]any{"reason": "browser session crashed"}}},
	}
	healthyRun := observability.TaskRun{
		TaskID:  "OPE-76-ok",
		RunID:   "run-ok",
		Medium:  "docker",
		Status:  "approved",
		Outcome: "default low risk path",
		Task:    map[string]any{"source": "linear", "title": "Healthy run"},
		Traces:  []observability.TraceEntry{{Span: "scheduler.decide", Status: "ok"}},
		Audits:  []observability.AuditEntry{{Action: "scheduler.decision", Actor: "scheduler", Outcome: "approved", Details: map[string]any{"reason": "default low risk path"}}},
	}

	center := BuildAutoTriageCenter([]observability.TaskRun{healthyRun, approvalRun, failedRun}, "Engineering Ops", "2026-03-10", nil)
	report := RenderAutoTriageCenterReport(center, 3, nil)

	if center.FlaggedRuns() != 2 || center.InboxSize() != 2 {
		t.Fatalf("unexpected triage center sizing: %+v", center)
	}
	if got := center.SeverityCounts(); got["critical"] != 1 || got["high"] != 1 || got["medium"] != 0 {
		t.Fatalf("unexpected severity counts: %+v", got)
	}
	if got := center.OwnerCounts(); got["security"] != 1 || got["engineering"] != 1 || got["operations"] != 0 {
		t.Fatalf("unexpected owner counts: %+v", got)
	}
	if center.Recommendation() != "immediate-attention" {
		t.Fatalf("unexpected recommendation: %+v", center)
	}
	if center.Findings[0].RunID != "run-browser" || center.Findings[1].RunID != "run-risk" {
		t.Fatalf("expected run-browser then run-risk, got %+v", center.Findings)
	}
	if center.Inbox[0].Suggestions[0].Label != "replay candidate" || center.Inbox[0].Suggestions[0].Confidence < 0.55 {
		t.Fatalf("expected replay suggestion, got %+v", center.Inbox[0].Suggestions)
	}
	if center.Findings[0].NextAction != "replay run and inspect tool failures" || center.Findings[1].NextAction != "request approval and queue security review" {
		t.Fatalf("unexpected next actions: %+v", center.Findings)
	}
	if !center.Findings[0].Actions[4].Enabled || center.Findings[1].Actions[4].Enabled || center.Findings[1].Actions[6].Enabled {
		t.Fatalf("unexpected action states: %+v %+v", center.Findings[0].Actions, center.Findings[1].Actions)
	}
	for _, want := range []string{
		"Flagged Runs: 2",
		"Inbox Size: 2",
		"Severity Mix: critical=1 high=1 medium=0",
		"Feedback Loop: accepted=0 rejected=0 pending=2",
		"run-browser: severity=critical owner=engineering status=failed",
		"run-risk: severity=high owner=security status=needs-approval",
		"actions=Drill Down [drill-down]",
		"Retry [retry] state=disabled target=run-risk reason=retry available after owner review",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in triage report, got %s", want, report)
		}
	}
}

func TestAutoTriageCenterReportRendersSharedViewPartialState(t *testing.T) {
	run := observability.TaskRun{
		TaskID:  "OPE-94-risk",
		RunID:   "run-risk",
		Medium:  "vm",
		Status:  "needs-approval",
		Outcome: "requires approval for high-risk task",
		Task:    map[string]any{"source": "linear", "title": "Prod approval"},
		Audits:  []observability.AuditEntry{{Action: "scheduler.decision", Actor: "scheduler", Outcome: "pending", Details: map[string]any{"reason": "requires approval for high-risk task"}}},
	}

	center := BuildAutoTriageCenter([]observability.TaskRun{run}, "Engineering Ops", "2026-03-10", nil)
	report := RenderAutoTriageCenterReport(center, 1, makeSharedView(1, false, nil, []string{"Replay ledger data is still backfilling."}))

	for _, want := range []string{
		"## View State",
		"- State: partial-data",
		"- Team: engineering",
		"## Partial Data",
		"Replay ledger data is still backfilling.",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in shared-view report, got %s", want, report)
		}
	}
}

func TestAutoTriageCenterBuildsSimilarityEvidenceAndFeedbackLoop(t *testing.T) {
	failedBrowserRun := observability.TaskRun{
		TaskID:  "OPE-100-browser-a",
		RunID:   "run-browser-a",
		Medium:  "browser",
		Status:  "failed",
		Outcome: "browser session crashed",
		Task:    map[string]any{"source": "linear", "title": "Browser replay failure"},
		Traces:  []observability.TraceEntry{{Span: "runtime.execute", Status: "failed"}},
		Audits:  []observability.AuditEntry{{Action: "runtime.execute", Actor: "worker", Outcome: "failed", Details: map[string]any{"reason": "browser session crashed"}}},
	}
	similarBrowserRun := observability.TaskRun{
		TaskID:  "OPE-100-browser-b",
		RunID:   "run-browser-b",
		Medium:  "browser",
		Status:  "failed",
		Outcome: "browser session crashed",
		Task:    map[string]any{"source": "linear", "title": "Browser replay failure"},
		Traces:  []observability.TraceEntry{{Span: "runtime.execute", Status: "failed"}},
		Audits:  []observability.AuditEntry{{Action: "runtime.execute", Actor: "worker", Outcome: "failed", Details: map[string]any{"reason": "browser session crashed"}}},
	}
	approvalRun := observability.TaskRun{
		TaskID:  "OPE-100-security",
		RunID:   "run-security",
		Medium:  "vm",
		Status:  "needs-approval",
		Outcome: "requires approval for high-risk task",
		Task:    map[string]any{"source": "linear", "title": "Security approval"},
		Traces:  []observability.TraceEntry{{Span: "scheduler.decide", Status: "pending"}},
		Audits:  []observability.AuditEntry{{Action: "scheduler.decision", Actor: "scheduler", Outcome: "pending", Details: map[string]any{"reason": "requires approval for high-risk task"}}},
	}
	feedback := []TriageFeedbackRecord{
		{RunID: "run-browser-a", Action: "replay run and inspect tool failures", Decision: "accepted", Actor: "ops-lead", Notes: "matched previous recovery path"},
		{RunID: "run-security", Action: "request approval and queue security review", Decision: "rejected", Actor: "sec-reviewer", Notes: "approval already in flight"},
	}

	center := BuildAutoTriageCenter([]observability.TaskRun{failedBrowserRun, similarBrowserRun, approvalRun}, "Auto Triage Center", "2026-03-11", feedback)
	report := RenderAutoTriageCenterReport(center, 3, nil)

	var browserItem, approvalItem *TriageInboxItem
	for i := range center.Inbox {
		switch center.Inbox[i].RunID {
		case "run-browser-a":
			browserItem = &center.Inbox[i]
		case "run-security":
			approvalItem = &center.Inbox[i]
		}
	}
	if browserItem == nil || approvalItem == nil {
		t.Fatalf("missing inbox items: %+v", center.Inbox)
	}
	if got := center.FeedbackCounts(); got["accepted"] != 1 || got["rejected"] != 1 || got["pending"] != 1 {
		t.Fatalf("unexpected feedback counts: %+v", got)
	}
	if browserItem.Suggestions[0].FeedbackStatus != "accepted" || approvalItem.Suggestions[0].FeedbackStatus != "rejected" {
		t.Fatalf("unexpected feedback statuses: %+v %+v", browserItem.Suggestions, approvalItem.Suggestions)
	}
	if browserItem.Suggestions[0].Evidence[0].RelatedRunID != "run-browser-b" || browserItem.Suggestions[0].Evidence[0].Score < 0.8 {
		t.Fatalf("unexpected similarity evidence: %+v", browserItem.Suggestions[0].Evidence)
	}
	for _, want := range []string{
		"## Inbox",
		"run-browser-a: severity=critical owner=engineering status=failed",
		"similar=run-browser-b:",
		"Feedback Loop: accepted=1 rejected=1 pending=1",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in triage report, got %s", want, report)
		}
	}
}

func TestTakeoverQueueFromLedgerGroupsPendingHandoffs(t *testing.T) {
	entries := []map[string]any{
		{
			"run_id":  "run-sec",
			"task_id": "OPE-66-sec",
			"source":  "linear",
			"summary": "requires approval for high-risk task",
			"audits": []any{
				map[string]any{"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "security", "reason": "requires approval for high-risk task", "required_approvals": []any{"security-review"}}},
			},
		},
		{
			"run_id":  "run-ops",
			"task_id": "OPE-66-ops",
			"source":  "linear",
			"summary": "premium tier required for advanced cross-department orchestration",
			"audits": []any{
				map[string]any{"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "operations", "reason": "premium tier required for advanced cross-department orchestration", "required_approvals": []any{"ops-manager"}}},
			},
		},
		{
			"run_id":  "run-ok",
			"task_id": "OPE-66-ok",
			"source":  "linear",
			"summary": "default low risk path",
			"audits": []any{
				map[string]any{"action": "scheduler.decision", "outcome": "approved", "details": map[string]any{"reason": "default low risk path"}},
			},
		},
	}

	queue := BuildTakeoverQueueFromLedger(entries, "Cross-Team Takeovers", "2026-03-10")
	report := RenderTakeoverQueueReport(queue, 3, nil)

	if queue.PendingRequests() != 2 || queue.ApprovalCount() != 2 || queue.Recommendation() != "expedite-security-review" {
		t.Fatalf("unexpected takeover queue: %+v", queue)
	}
	if got := queue.TeamCounts(); got["operations"] != 1 || got["security"] != 1 {
		t.Fatalf("unexpected team counts: %+v", got)
	}
	if queue.Requests[0].RunID != "run-ops" || queue.Requests[1].RunID != "run-sec" {
		t.Fatalf("unexpected request ordering: %+v", queue.Requests)
	}
	if !queue.Requests[0].Actions[3].Enabled || queue.Requests[1].Actions[3].Enabled {
		t.Fatalf("unexpected escalate state: %+v", queue.Requests)
	}
	for _, want := range []string{
		"Pending Requests: 2",
		"Team Mix: operations=1 security=1",
		"run-sec: team=security status=pending task=OPE-66-sec approvals=security-review",
		"run-ops: team=operations status=pending task=OPE-66-ops approvals=ops-manager",
		"Escalate [escalate] state=disabled target=run-sec reason=security takeovers are already escalated",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in takeover report, got %s", want, report)
		}
	}
}

func TestTakeoverQueueReportRendersSharedViewErrorState(t *testing.T) {
	queue := BuildTakeoverQueueFromLedger(nil, "Cross-Team Takeovers", "2026-03-10")
	report := RenderTakeoverQueueReport(queue, 0, makeSharedView(0, false, []string{"Takeover approvals service timed out."}, nil))
	for _, want := range []string{
		"- State: error",
		"- Summary: Unable to load data for the current filters.",
		"## Errors",
		"Takeover approvals service timed out.",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in takeover report, got %s", want, report)
		}
	}
}

func TestOrchestrationCanvasSummarizesPolicyAndHandoff(t *testing.T) {
	run := observability.TaskRun{
		TaskID: "OPE-66-canvas",
		RunID:  "run-canvas",
		Medium: "browser",
		Audits: []observability.AuditEntry{{Action: "tool.invoke", Actor: "worker", Outcome: "success", Details: map[string]any{"tool": "browser"}}},
	}
	plan := workflow.OrchestrationPlan{
		TaskID:            "OPE-66-canvas",
		CollaborationMode: "tier-limited",
		Handoffs: []workflow.DepartmentHandoff{
			{Department: "operations", Reason: "coordinate"},
			{Department: "engineering", Reason: "execute", RequiredTools: []string{"browser"}},
		},
	}
	policy := &workflow.OrchestrationPolicyDecision{
		Tier:               "standard",
		UpgradeRequired:    true,
		Reason:             "premium tier required for advanced cross-department orchestration",
		BlockedDepartments: []string{"customer-success"},
		EntitlementStatus:  "upgrade-required",
		BillingModel:       "standard-blocked",
		EstimatedCostUSD:   7.0,
		IncludedUsageUnits: 2,
		OverageUsageUnits:  1,
		OverageCostUSD:     4.0,
	}
	handoff := &workflow.HandoffRequest{TargetTeam: "operations", Reason: policy.Reason, Status: "pending", RequiredApprovals: []string{"ops-manager"}}

	canvas := BuildOrchestrationCanvas(run, plan, policy, handoff)
	report := RenderOrchestrationCanvas(canvas)

	if canvas.Recommendation() != "resolve-entitlement-gap" {
		t.Fatalf("unexpected canvas recommendation: %+v", canvas)
	}
	if len(canvas.ActiveTools) != 1 || canvas.ActiveTools[0] != "browser" {
		t.Fatalf("unexpected active tools: %+v", canvas.ActiveTools)
	}
	if !canvas.Actions[3].Enabled || canvas.Actions[4].Enabled {
		t.Fatalf("unexpected action states: %+v", canvas.Actions)
	}
	for _, want := range []string{
		"# Orchestration Canvas",
		"- Tier: standard",
		"- Entitlement Status: upgrade-required",
		"- Billing Model: standard-blocked",
		"- Estimated Cost (USD): 7.00",
		"- Handoff Team: operations",
		"- Recommendation: resolve-entitlement-gap",
		"## Actions",
		"Escalate [escalate] state=enabled target=run-canvas",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in canvas report, got %s", want, report)
		}
	}
}

func TestOrchestrationCanvasReconstructsFlowCollaborationFromLedger(t *testing.T) {
	entry := map[string]any{
		"run_id":  "run-flow-1",
		"task_id": "OPE-113",
		"audits": []any{
			map[string]any{"action": "orchestration.plan", "actor": "scheduler", "outcome": "enabled", "timestamp": "2026-03-11T11:00:00Z", "details": map[string]any{"collaboration_mode": "cross-functional", "departments": []any{"operations", "engineering"}, "approvals": []any{}}},
			map[string]any{"action": "orchestration.policy", "actor": "scheduler", "outcome": "enabled", "timestamp": "2026-03-11T11:01:00Z", "details": map[string]any{"tier": "premium", "entitlement_status": "included", "billing_model": "premium-included"}},
			map[string]any{"action": "collaboration.comment", "actor": "ops-lead", "outcome": "recorded", "timestamp": "2026-03-11T11:02:00Z", "details": map[string]any{"surface": "flow", "comment_id": "flow-comment-1", "body": "Route @eng once the dashboard note is resolved.", "mentions": []any{"eng"}, "anchor": "handoff-lane", "status": "open"}},
			map[string]any{"action": "collaboration.decision", "actor": "eng-manager", "outcome": "accepted", "timestamp": "2026-03-11T11:03:00Z", "details": map[string]any{"surface": "flow", "decision_id": "flow-decision-1", "summary": "Engineering owns the next flow handoff.", "mentions": []any{"ops-lead"}, "related_comment_ids": []any{"flow-comment-1"}, "follow_up": "Post in the shared channel after deploy."}},
		},
	}

	canvas := BuildOrchestrationCanvasFromLedgerEntry(entry)
	report := RenderOrchestrationCanvas(canvas)
	if canvas.Collaboration == nil || canvas.Recommendation() != "resolve-flow-comments" {
		t.Fatalf("unexpected collaboration canvas: %+v", canvas)
	}
	for _, want := range []string{
		"## Collaboration",
		"Route @eng once the dashboard note is resolved.",
		"Engineering owns the next flow handoff.",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in collaboration report, got %s", want, report)
		}
	}
}

func TestOrchestrationPortfolioAndOverviewRollUpCanvasAndTakeoverState(t *testing.T) {
	canvases := []OrchestrationCanvas{
		{
			TaskID:             "OPE-66-a",
			RunID:              "run-a",
			CollaborationMode:  "cross-functional",
			Departments:        []string{"operations", "engineering", "security"},
			Tier:               "premium",
			EntitlementStatus:  "included",
			BillingModel:       "premium-included",
			EstimatedCostUSD:   4.5,
			IncludedUsageUnits: 3,
			HandoffTeam:        "security",
			HandoffStatus:      "pending",
		},
		{
			TaskID:             "OPE-66-b",
			RunID:              "run-b",
			CollaborationMode:  "tier-limited",
			Departments:        []string{"operations", "engineering"},
			Tier:               "standard",
			UpgradeRequired:    true,
			EntitlementStatus:  "upgrade-required",
			BillingModel:       "standard-blocked",
			EstimatedCostUSD:   7.0,
			IncludedUsageUnits: 2,
			OverageUsageUnits:  1,
			OverageCostUSD:     4.0,
			BlockedDepartments: []string{"customer-success"},
			HandoffTeam:        "operations",
			HandoffStatus:      "pending",
		},
	}
	queue := BuildTakeoverQueueFromLedger([]map[string]any{
		{"run_id": "run-a", "task_id": "OPE-66-a", "source": "linear", "audits": []any{map[string]any{"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "security", "reason": "risk", "required_approvals": []any{"security-review"}}}}},
		{"run_id": "run-b", "task_id": "OPE-66-b", "source": "linear", "audits": []any{map[string]any{"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "operations", "reason": "entitlement", "required_approvals": []any{"ops-manager"}}}}},
	}, "Cross-Team Takeovers", "2026-03-10")

	portfolio := BuildOrchestrationPortfolio(canvases, "Cross-Team Portfolio", "2026-03-10", &queue)
	report := RenderOrchestrationPortfolioReport(portfolio, nil)
	page := RenderOrchestrationOverviewPage(portfolio)

	if portfolio.TotalRuns() != 2 || portfolio.TotalEstimatedCostUSD() != 11.5 || portfolio.TotalOverageCostUSD() != 4.0 {
		t.Fatalf("unexpected portfolio totals: %+v", portfolio)
	}
	if portfolio.UpgradeRequiredCount() != 1 || portfolio.ActiveHandoffs() != 2 || portfolio.Recommendation() != "stabilize-security-takeovers" {
		t.Fatalf("unexpected portfolio rollup: %+v", portfolio)
	}
	for _, want := range []string{
		"- Collaboration Mix: cross-functional=1 tier-limited=1",
		"- Tier Mix: premium=1 standard=1",
		"- Entitlement Mix: included=1 upgrade-required=1",
		"- Billing Models: premium-included=1 standard-blocked=1",
		"- Estimated Cost (USD): 11.50",
		"- Overage Cost (USD): 4.00",
		"- Takeover Queue: pending=2 recommendation=expedite-security-review",
		"- run-a: mode=cross-functional tier=premium entitlement=included billing=premium-included estimated_cost_usd=4.50 overage_cost_usd=0.00 upgrade_required=false handoff=security",
		"actions=Drill Down [drill-down]",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in portfolio report, got %s", want, report)
		}
	}
	for _, want := range []string{
		"<title>Orchestration Overview",
		"Cross-Team Portfolio",
		"review-security-takeover",
		"Estimated Cost",
		"premium-included",
		"pending=2 recommendation=expedite-security-review",
		"run-a",
		"actions=Drill Down [drill-down]",
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("expected %q in overview page, got %s", want, page)
		}
	}
}

func TestOrchestrationPortfolioReportRendersSharedViewEmptyState(t *testing.T) {
	report := RenderOrchestrationPortfolioReport(
		BuildOrchestrationPortfolio(nil, "Cross-Team Portfolio", "2026-03-10", nil),
		makeSharedView(0, false, nil, nil),
	)
	for _, want := range []string{
		"- State: empty",
		"- Summary: No records match the current filters.",
		"## Filters",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in empty-state report, got %s", want, report)
		}
	}
}

func TestBuildOrchestrationCanvasFromLedgerEntryExtractsAuditState(t *testing.T) {
	entry := map[string]any{
		"run_id":  "run-ledger",
		"task_id": "OPE-66-ledger",
		"audits": []any{
			map[string]any{"action": "orchestration.plan", "outcome": "ready", "details": map[string]any{"collaboration_mode": "tier-limited", "departments": []any{"operations", "engineering"}, "approvals": []any{"security-review"}}},
			map[string]any{"action": "orchestration.policy", "outcome": "upgrade-required", "details": map[string]any{"tier": "standard", "entitlement_status": "upgrade-required", "billing_model": "standard-blocked", "estimated_cost_usd": 7.0, "included_usage_units": 2, "overage_usage_units": 1, "overage_cost_usd": 4.0, "blocked_departments": []any{"security", "customer-success"}}},
			map[string]any{"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "operations", "reason": "premium tier required for advanced cross-department orchestration"}},
			map[string]any{"action": "tool.invoke", "outcome": "success", "details": map[string]any{"tool": "browser"}},
		},
	}

	canvas := BuildOrchestrationCanvasFromLedgerEntry(entry)
	if canvas.RunID != "run-ledger" || canvas.CollaborationMode != "tier-limited" || canvas.Tier != "standard" || !canvas.UpgradeRequired {
		t.Fatalf("unexpected canvas state: %+v", canvas)
	}
	if strings.Join(canvas.Departments, ",") != "operations,engineering" || strings.Join(canvas.RequiredApprovals, ",") != "security-review" {
		t.Fatalf("unexpected plan fields: %+v", canvas)
	}
	if canvas.EntitlementStatus != "upgrade-required" || canvas.BillingModel != "standard-blocked" || canvas.EstimatedCostUSD != 7.0 || canvas.OverageCostUSD != 4.0 {
		t.Fatalf("unexpected billing fields: %+v", canvas)
	}
	if strings.Join(canvas.BlockedDepartments, ",") != "security,customer-success" || canvas.HandoffTeam != "operations" {
		t.Fatalf("unexpected handoff fields: %+v", canvas)
	}
	if len(canvas.ActiveTools) != 1 || canvas.ActiveTools[0] != "browser" || !canvas.Actions[3].Enabled || canvas.Actions[4].Enabled {
		t.Fatalf("unexpected actions or tools: %+v", canvas)
	}
}

func TestBuildBillingEntitlementsPageAndLedgerRollUpOrchestrationCosts(t *testing.T) {
	portfolio := OrchestrationPortfolio{
		Name:   "Revenue Ops",
		Period: "2026-03",
		Canvases: []OrchestrationCanvas{
			{
				TaskID:             "OPE-104-a",
				RunID:              "run-billing-a",
				CollaborationMode:  "cross-functional",
				Departments:        []string{"operations", "engineering", "security"},
				Tier:               "premium",
				EntitlementStatus:  "included",
				BillingModel:       "premium-included",
				EstimatedCostUSD:   4.5,
				IncludedUsageUnits: 3,
				HandoffTeam:        "security",
			},
			{
				TaskID:             "OPE-104-b",
				RunID:              "run-billing-b",
				CollaborationMode:  "tier-limited",
				Departments:        []string{"operations", "engineering"},
				Tier:               "standard",
				UpgradeRequired:    true,
				EntitlementStatus:  "upgrade-required",
				BillingModel:       "standard-blocked",
				EstimatedCostUSD:   7.0,
				IncludedUsageUnits: 2,
				OverageUsageUnits:  1,
				OverageCostUSD:     4.0,
				BlockedDepartments: []string{"customer-success"},
				HandoffTeam:        "operations",
			},
		},
	}

	page := BuildBillingEntitlementsPage(portfolio, "OpenAGI Revenue Cloud", "Standard", "2026-03")
	report := RenderBillingEntitlementsReport(page, nil)
	html := RenderBillingEntitlementsPage(page)

	if page.RunCount() != 2 || page.TotalIncludedUsageUnits() != 5 || page.TotalOverageUsageUnits() != 1 {
		t.Fatalf("unexpected page usage totals: %+v", page)
	}
	if page.TotalEstimatedCostUSD() != 11.5 || page.TotalOverageCostUSD() != 4.0 || page.UpgradeRequiredCount() != 1 {
		t.Fatalf("unexpected page cost totals: %+v", page)
	}
	if got := page.EntitlementCounts(); got["included"] != 1 || got["upgrade-required"] != 1 {
		t.Fatalf("unexpected entitlement counts: %+v", got)
	}
	if got := page.BillingModelCounts(); got["premium-included"] != 1 || got["standard-blocked"] != 1 {
		t.Fatalf("unexpected billing counts: %+v", got)
	}
	if len(page.BlockedCapabilities()) != 1 || page.BlockedCapabilities()[0] != "customer-success" || page.Recommendation() != "resolve-plan-gaps" {
		t.Fatalf("unexpected blocked capabilities or recommendation: %+v", page)
	}
	for _, want := range []string{
		"# Billing & Entitlements Report",
		"- Workspace: OpenAGI Revenue Cloud",
		"- Overage Cost (USD): 4.00",
		"- run-billing-b: task=OPE-104-b entitlement=upgrade-required billing=standard-blocked",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in billing report, got %s", want, report)
		}
	}
	for _, want := range []string{
		"<title>Billing & Entitlements",
		"OpenAGI Revenue Cloud",
		"Charge Feed",
		"run-billing-b",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in billing html, got %s", want, html)
		}
	}

	entries := []map[string]any{
		{
			"run_id":  "run-ledger-a",
			"task_id": "OPE-104-a",
			"audits": []any{
				map[string]any{"action": "orchestration.plan", "outcome": "ready", "details": map[string]any{"collaboration_mode": "cross-functional", "departments": []any{"operations", "engineering", "security"}, "approvals": []any{"security-review"}}},
				map[string]any{"action": "orchestration.policy", "outcome": "enabled", "details": map[string]any{"tier": "premium", "entitlement_status": "included", "billing_model": "premium-included", "estimated_cost_usd": 4.5, "included_usage_units": 3, "blocked_departments": []any{}}},
			},
		},
		{
			"run_id":  "run-ledger-b",
			"task_id": "OPE-104-b",
			"audits": []any{
				map[string]any{"action": "orchestration.plan", "outcome": "ready", "details": map[string]any{"collaboration_mode": "tier-limited", "departments": []any{"operations", "engineering"}, "approvals": []any{}}},
				map[string]any{"action": "orchestration.policy", "outcome": "upgrade-required", "details": map[string]any{"tier": "standard", "entitlement_status": "upgrade-required", "billing_model": "standard-blocked", "estimated_cost_usd": 7.0, "included_usage_units": 2, "overage_usage_units": 1, "overage_cost_usd": 4.0, "blocked_departments": []any{"customer-success"}}},
				map[string]any{"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "operations", "reason": "entitlement gap", "required_approvals": []any{"ops-manager"}}},
			},
		},
	}

	ledgerPage := BuildBillingEntitlementsPageFromLedger(entries, "OpenAGI Revenue Cloud", "Standard", "2026-03")
	if ledgerPage.RunCount() != 2 || ledgerPage.Recommendation() != "resolve-plan-gaps" || ledgerPage.TotalOverageCostUSD() != 4.0 {
		t.Fatalf("unexpected ledger page summary: %+v", ledgerPage)
	}
	if len(ledgerPage.Charges[1].BlockedCapabilities) != 1 || ledgerPage.Charges[1].BlockedCapabilities[0] != "customer-success" || ledgerPage.Charges[1].HandoffTeam != "operations" {
		t.Fatalf("unexpected ledger charge state: %+v", ledgerPage.Charges[1])
	}
}

func TestNewTriageFeedbackRecordUsesUTCISOTime(t *testing.T) {
	record := NewTriageFeedbackRecord("run-risk", "request approval and queue security review", "pending", "sec-reviewer", "")
	if !strings.HasSuffix(record.Timestamp, "Z") {
		t.Fatalf("expected UTC timestamp, got %q", record.Timestamp)
	}
	parsed, err := time.Parse(time.RFC3339, record.Timestamp)
	if err != nil {
		t.Fatalf("parse timestamp: %v", err)
	}
	if parsed.Location() != time.UTC {
		t.Fatalf("expected UTC timestamp, got %s", parsed.Location())
	}
}
