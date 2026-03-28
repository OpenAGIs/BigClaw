package reporting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/repo"
	"bigclaw-go/internal/workflow"
	"bigclaw-go/internal/workflowexec"
)

func TestReportStudioBundleAndReadiness(t *testing.T) {
	studio := ReportStudio{
		Name:     "Executive Weekly Narrative",
		IssueID:  "OPE-112",
		Audience: "executive",
		Period:   "2026-W11",
		Summary:  "Delivery recovered after queue pressure eased.",
		Sections: []NarrativeSection{
			{Heading: "What changed", Body: "Approval queue depth fell.", Evidence: []string{"queue-control-center"}, Callouts: []string{"SLA risk contained"}},
			{Heading: "What needs attention", Body: "Security handoffs still bunch late-week."},
		},
	}

	artifacts, err := WriteReportStudioBundle(filepath.Join(t.TempDir(), "studio"), studio)
	if err != nil {
		t.Fatalf("write studio bundle: %v", err)
	}
	if !studio.Ready() || studio.Recommendation() != "publish" {
		t.Fatalf("expected ready publish studio, got ready=%t recommendation=%s", studio.Ready(), studio.Recommendation())
	}
	if !strings.Contains(RenderReportStudioReport(studio), "### What changed") {
		t.Fatalf("expected narrative section in markdown")
	}
	if !strings.Contains(RenderReportStudioPlainText(studio), "Recommendation: publish") {
		t.Fatalf("expected plain text recommendation")
	}
	if !strings.Contains(RenderReportStudioHTML(studio), "<h1>Executive Weekly Narrative</h1>") {
		t.Fatalf("expected html title")
	}
	for _, path := range []string{artifacts.MarkdownPath, artifacts.HTMLPath, artifacts.TextPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected artifact %s: %v", path, err)
		}
	}
	if !strings.Contains(artifacts.MarkdownPath, "executive-weekly-narrative.md") {
		t.Fatalf("unexpected artifact path: %s", artifacts.MarkdownPath)
	}
}

func TestPilotPortfolioRollup(t *testing.T) {
	benchmarkPassed := true
	portfolio := BuildPilotPortfolio("Design Partners", "2026-H1", []PilotScorecard{
		{
			IssueID:            "OPE-60",
			Customer:           "Partner A",
			Period:             "2026-Q2",
			Metrics:            []PilotMetric{{Name: "Coverage", Baseline: 40, Current: 85, Target: 80, Unit: "%", HigherIsBetter: true}},
			MonthlyBenefit:     15000,
			MonthlyCost:        3000,
			ImplementationCost: 18000,
			BenchmarkScore:     intPtr(97),
			BenchmarkPassed:    &benchmarkPassed,
		},
		{
			IssueID:            "OPE-61",
			Customer:           "Partner B",
			Period:             "2026-Q2",
			Metrics:            []PilotMetric{{Name: "Cycle time", Baseline: 12, Current: 7, Target: 5, Unit: "h", HigherIsBetter: false}},
			MonthlyBenefit:     9000,
			MonthlyCost:        2500,
			ImplementationCost: 12000,
			BenchmarkScore:     intPtr(88),
			BenchmarkPassed:    &benchmarkPassed,
		},
	})

	if portfolio.TotalMonthlyNetValue != 18500 {
		t.Fatalf("unexpected monthly net value: %+v", portfolio)
	}
	if portfolio.AverageROI != 195.2 {
		t.Fatalf("unexpected average roi: %+v", portfolio)
	}
	if portfolio.RecommendationCounts["go"] != 1 || portfolio.RecommendationCounts["iterate"] != 1 || portfolio.Recommendation != "continue" {
		t.Fatalf("unexpected recommendation mix: %+v", portfolio)
	}
	report := RenderPilotPortfolioReport(portfolio)
	for _, fragment := range []string{"Recommendation Mix: go=1 iterate=1 hold=0", "Partner A: recommendation=go", "Partner B: recommendation=iterate"} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}

func TestValidationAndClosureChecks(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "validation.md")
	decision := EvaluateIssueClosure("BIG-602", reportPath, false, nil, nil)
	if decision.Allowed || decision.Reason != "validation report required before closing issue" {
		t.Fatalf("unexpected missing-report decision: %+v", decision)
	}

	if err := WriteReport(reportPath, "# Validation\n\nfailed"); err != nil {
		t.Fatalf("write report: %v", err)
	}
	decision = EvaluateIssueClosure("BIG-602", reportPath, false, nil, nil)
	if decision.Allowed || decision.Reason != "validation failed; issue must remain open" || !ValidationReportExists(reportPath) {
		t.Fatalf("unexpected failed-report decision: %+v", decision)
	}

	if err := WriteReport(reportPath, RenderIssueValidationReport("BIG-602", "v0.1", "sandbox", "pass")); err != nil {
		t.Fatalf("write validation report: %v", err)
	}
	decision = EvaluateIssueClosure("BIG-602", reportPath, true, nil, nil)
	if !decision.Allowed || decision.Reason != "validation report and launch checklist requirements satisfied; issue can be closed" {
		t.Fatalf("unexpected pass decision: %+v", decision)
	}
	content := RenderIssueValidationReport("BIG-900", "v1", "repo", "pass")
	if !strings.Contains(content, "- 生成时间: ") || !strings.Contains(content, "Z") {
		t.Fatalf("expected utc timestamp in report: %s", content)
	}
}

func TestLaunchAndFinalDeliveryChecklistState(t *testing.T) {
	dir := t.TempDir()
	runbook := filepath.Join(dir, "runbook.md")
	validationBundle := filepath.Join(dir, "validation-bundle.md")
	releaseNotes := filepath.Join(dir, "release-notes.md")
	if err := WriteReport(runbook, "# Runbook\n\nready"); err != nil {
		t.Fatalf("write runbook: %v", err)
	}
	if err := WriteReport(validationBundle, "# Validation Bundle\n\nready"); err != nil {
		t.Fatalf("write validation bundle: %v", err)
	}

	launch := BuildLaunchChecklist("BIG-1003",
		[]DocumentationArtifact{{Name: "runbook", Path: runbook}, {Name: "faq", Path: filepath.Join(dir, "faq.md")}},
		[]LaunchChecklistItem{{Name: "Operations handoff", Evidence: []string{"runbook"}}, {Name: "Support handoff", Evidence: []string{"faq"}}},
	)
	if launch.DocumentationStatus["runbook"] != true || launch.DocumentationStatus["faq"] != false || launch.CompletedItems != 1 || launch.Ready() {
		t.Fatalf("unexpected launch checklist: %+v", launch)
	}
	launchReport := RenderLaunchChecklistReport(launch)
	for _, fragment := range []string{"runbook: available=true", "faq: available=false", "Support handoff: completed=false evidence=faq"} {
		if !strings.Contains(launchReport, fragment) {
			t.Fatalf("expected %q in launch report, got %s", fragment, launchReport)
		}
	}

	final := BuildFinalDeliveryChecklist("BIG-4702",
		[]DocumentationArtifact{{Name: "validation-bundle", Path: validationBundle}, {Name: "release-notes", Path: releaseNotes}},
		[]DocumentationArtifact{{Name: "runbook", Path: filepath.Join(dir, "runbook-2.md")}},
	)
	if final.GeneratedRequiredOutputs != 1 || final.GeneratedRecommendedDocumentation != 0 || final.Ready() {
		t.Fatalf("unexpected final delivery checklist: %+v", final)
	}
	finalReport := RenderFinalDeliveryChecklistReport(final)
	for _, fragment := range []string{"Required Outputs Generated: 1/2", "Recommended Docs Generated: 0/1", "release-notes: available=false"} {
		if !strings.Contains(finalReport, fragment) {
			t.Fatalf("expected %q in final report, got %s", fragment, finalReport)
		}
	}

	reportPath := filepath.Join(dir, "validation.md")
	if err := WriteReport(reportPath, RenderIssueValidationReport("BIG-4702", "v0.3", "staging", "pass")); err != nil {
		t.Fatalf("write validation report: %v", err)
	}
	blocked := EvaluateIssueClosure("BIG-4702", reportPath, true, nil, &final)
	if blocked.Allowed || blocked.Reason != "final delivery checklist incomplete; required outputs missing" {
		t.Fatalf("unexpected final checklist block: %+v", blocked)
	}
	if err := WriteReport(releaseNotes, "# Release Notes\n\nready"); err != nil {
		t.Fatalf("write release notes: %v", err)
	}
	readyFinal := BuildFinalDeliveryChecklist("BIG-4702",
		[]DocumentationArtifact{{Name: "validation-bundle", Path: validationBundle}, {Name: "release-notes", Path: releaseNotes}},
		nil,
	)
	allowed := EvaluateIssueClosure("BIG-4702", reportPath, true, nil, &readyFinal)
	if !allowed.Allowed || allowed.Reason != "validation report and final delivery checklist requirements satisfied; issue can be closed" {
		t.Fatalf("unexpected final checklist allow: %+v", allowed)
	}
}

func TestRenderSharedViewContextIncludesCollaboration(t *testing.T) {
	lines := RenderSharedViewContext(SharedViewContext{
		Filters:     []SharedViewFilter{{Label: "Team", Value: "ops"}},
		ResultCount: 4,
		Collaboration: &repo.CollaborationThread{
			Surface:   "dashboard",
			TargetID:  "ops-overview",
			Comments:  []repo.CollaborationComment{{CommentID: "c1", Body: "Please review blocker copy with @ops and @eng."}},
			Decisions: []repo.DecisionNote{{DecisionID: "d1", Summary: "Keep the blocker module visible for managers."}},
		},
	})
	content := strings.Join(lines, "\n")
	for _, fragment := range []string{"## Collaboration", "Surface: dashboard", "Please review blocker copy with @ops and @eng.", "Keep the blocker module visible for managers."} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("expected %q in shared view context, got %s", fragment, content)
		}
	}
}

func TestAutoTriageCenterBuildsPriorityAndFeedback(t *testing.T) {
	approvalRun := workflowexec.NewTaskRun(domain.Task{ID: "OPE-76-risk", Source: "linear", Title: "Prod approval"}, "run-risk", "vm")
	approvalRun.Trace("scheduler.decide", "pending", nil)
	approvalRun.Audit("scheduler.decision", "scheduler", "pending", map[string]any{"reason": "requires approval for high-risk task"})
	approvalRun.Finalize("needs-approval", "requires approval for high-risk task")

	failedRun := workflowexec.NewTaskRun(domain.Task{ID: "OPE-76-browser", Source: "linear", Title: "Replay browser task"}, "run-browser", "browser")
	failedRun.Trace("runtime.execute", "failed", nil)
	failedRun.Audit("runtime.execute", "worker", "failed", map[string]any{"reason": "browser session crashed"})
	failedRun.Finalize("failed", "browser session crashed")

	similarRun := workflowexec.NewTaskRun(domain.Task{ID: "OPE-100-browser-b", Source: "linear", Title: "Browser replay failure"}, "run-browser-b", "browser")
	similarRun.Trace("runtime.execute", "failed", nil)
	similarRun.Audit("runtime.execute", "worker", "failed", map[string]any{"reason": "browser session crashed"})
	similarRun.Finalize("failed", "browser session crashed")

	healthyRun := workflowexec.NewTaskRun(domain.Task{ID: "OPE-76-ok", Source: "linear", Title: "Healthy run"}, "run-ok", "docker")
	healthyRun.Finalize("approved", "default low risk path")

	center := BuildAutoTriageCenter([]workflowexec.TaskRun{healthyRun, approvalRun, failedRun, similarRun}, "Engineering Ops", "2026-03-10", []TriageFeedbackRecord{
		NewTriageFeedbackRecord("run-browser", "replay run and inspect tool failures", "accepted", "ops-lead", "matched previous recovery path"),
		NewTriageFeedbackRecord("run-risk", "request approval and queue security review", "rejected", "sec-reviewer", "approval already in flight"),
	})

	if center.FlaggedRuns != 3 || center.InboxSize != 3 {
		t.Fatalf("unexpected triage center size: %+v", center)
	}
	if center.Recommendation != "immediate-attention" || center.SeverityCounts["critical"] != 2 || center.OwnerCounts["security"] != 1 {
		t.Fatalf("unexpected triage summary: %+v", center)
	}
	if center.Findings[0].RunID != "run-browser" || center.Findings[0].Actions[4].Enabled != true {
		t.Fatalf("unexpected first finding: %+v", center.Findings[0])
	}
	if center.Findings[2].RunID != "run-risk" || center.Findings[2].Actions[4].Enabled != false || center.Findings[2].Actions[6].Enabled != false {
		t.Fatalf("unexpected approval finding actions: %+v", center.Findings[2])
	}
	browserItem := center.Inbox[0]
	if browserItem.Suggestions[0].Label != "replay candidate" || browserItem.Suggestions[0].FeedbackStatus != "accepted" {
		t.Fatalf("unexpected browser suggestion: %+v", browserItem)
	}
	if len(browserItem.Suggestions[0].Evidence) == 0 || browserItem.Suggestions[0].Evidence[0].RelatedRunID != "run-browser-b" || browserItem.Suggestions[0].Evidence[0].Score < 0.8 {
		t.Fatalf("expected similarity evidence: %+v", browserItem)
	}
	report := RenderAutoTriageCenterReport(center, 4, &SharedViewContext{
		Filters:     []SharedViewFilter{{Label: "Team", Value: "engineering"}},
		ResultCount: 1,
		PartialData: []string{"Replay ledger data is still backfilling."},
	})
	for _, fragment := range []string{"Flagged Runs: 3", "Severity Mix: critical=2 high=1 medium=0", "Feedback Loop: accepted=1 rejected=1 pending=1", "run-browser: severity=critical owner=engineering status=failed", "similar=run-browser-b:0.8", "## Partial Data"} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in triage report, got %s", fragment, report)
		}
	}
}

func TestTakeoverQueueBuildAndViewState(t *testing.T) {
	queue := BuildTakeoverQueueFromLedger([]map[string]any{
		{
			"run_id":  "run-sec",
			"task_id": "OPE-66-sec",
			"audits":  []any{map[string]any{"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "security", "required_approvals": []any{"security-review"}}}},
		},
		{
			"run_id":  "run-ops",
			"task_id": "OPE-66-ops",
			"audits":  []any{map[string]any{"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "operations", "required_approvals": []any{"ops-manager"}}}},
		},
	}, "Cross-Team Takeovers", "2026-03-10")

	if queue.PendingRequests != 2 || queue.TeamCounts["security"] != 1 || queue.Recommendation != "expedite-security-review" {
		t.Fatalf("unexpected takeover queue: %+v", queue)
	}
	if queue.Requests[0].RunID != "run-ops" || queue.Requests[0].Actions[3].Enabled != true {
		t.Fatalf("unexpected operations request: %+v", queue.Requests[0])
	}
	if queue.Requests[1].RunID != "run-sec" || queue.Requests[1].Actions[3].Enabled != false {
		t.Fatalf("unexpected security request: %+v", queue.Requests[1])
	}
	report := RenderTakeoverQueueReport(queue, 3, &SharedViewContext{
		Filters: []SharedViewFilter{{Label: "Team", Value: "engineering"}},
		Errors:  []string{"Takeover approvals service timed out."},
	})
	for _, fragment := range []string{"Pending Requests: 2", "Team Mix: operations=1 security=1", "run-sec: team=security status=pending task=OPE-66-sec approvals=security-review", "Escalate [escalate] state=disabled target=run-sec reason=security takeovers are already escalated", "## Errors"} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in takeover report, got %s", fragment, report)
		}
	}
}

func TestOrchestrationCanvasPortfolioAndBillingSurfaces(t *testing.T) {
	run := workflowexec.NewTaskRun(domain.Task{ID: "OPE-66-canvas", Source: "linear", Title: "Canvas run"}, "run-canvas", "browser")
	run.Audit("tool.invoke", "worker", "success", map[string]any{"tool": "browser"})
	plan := workflow.OrchestrationPlan{
		TaskID:            "OPE-66-canvas",
		CollaborationMode: "tier-limited",
		Handoffs: []workflow.DepartmentHandoff{
			{Department: "operations", Reason: "coordinate"},
			{Department: "engineering", Reason: "execute", RequiredTools: []string{"browser"}},
		},
	}
	policy := workflow.OrchestrationPolicyDecision{
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
	if canvas.Recommendation != "resolve-entitlement-gap" || len(canvas.ActiveTools) != 1 || canvas.Actions[3].Enabled != true || canvas.Actions[4].Enabled != false {
		t.Fatalf("unexpected canvas: %+v", canvas)
	}
	canvasReport := RenderOrchestrationCanvas(canvas)
	for _, fragment := range []string{"- Tier: standard", "- Entitlement Status: upgrade-required", "- Billing Model: standard-blocked", "- Estimated Cost (USD): 7.00", "- Handoff Team: operations", "- Recommendation: resolve-entitlement-gap"} {
		if !strings.Contains(canvasReport, fragment) {
			t.Fatalf("expected %q in canvas report, got %s", fragment, canvasReport)
		}
	}

	ledgerCanvas := BuildOrchestrationCanvasFromLedgerEntry(map[string]any{
		"run_id":  "run-flow-1",
		"task_id": "OPE-113",
		"audits": []any{
			map[string]any{"action": "orchestration.plan", "outcome": "enabled", "timestamp": "2026-03-11T11:00:00Z", "details": map[string]any{"collaboration_mode": "cross-functional", "departments": []any{"operations", "engineering"}, "approvals": []any{}}},
			map[string]any{"action": "orchestration.policy", "outcome": "enabled", "timestamp": "2026-03-11T11:01:00Z", "details": map[string]any{"tier": "premium", "entitlement_status": "included", "billing_model": "premium-included"}},
			map[string]any{"action": "collaboration.comment", "actor": "ops-lead", "outcome": "recorded", "timestamp": "2026-03-11T11:02:00Z", "details": map[string]any{"comment_id": "flow-comment-1", "body": "Route @eng once the dashboard note is resolved.", "mentions": []any{"eng"}, "anchor": "handoff-lane"}},
			map[string]any{"action": "collaboration.decision", "actor": "eng-manager", "outcome": "accepted", "timestamp": "2026-03-11T11:03:00Z", "details": map[string]any{"decision_id": "flow-decision-1", "summary": "Engineering owns the next flow handoff.", "mentions": []any{"ops-lead"}, "follow_up": "Post in the shared channel after deploy."}},
		},
	})
	if ledgerCanvas.Recommendation != "resolve-flow-comments" || ledgerCanvas.Collaboration == nil {
		t.Fatalf("unexpected ledger canvas: %+v", ledgerCanvas)
	}

	queue := queueForCanvasTests()
	portfolio := BuildOrchestrationPortfolio([]OrchestrationCanvas{
		{
			TaskID: "OPE-66-a", RunID: "run-a", CollaborationMode: "cross-functional", Departments: []string{"operations", "engineering", "security"},
			Tier: "premium", EntitlementStatus: "included", BillingModel: "premium-included", EstimatedCostUSD: 4.5, IncludedUsageUnits: 3, HandoffTeam: "security",
			Actions: []ConsoleAction{{ActionID: "drill-down", Label: "Drill Down", Target: "run-a", Enabled: true}},
		},
		canvas,
	}, "Cross-Team Portfolio", "2026-03-10", &queue)

	if portfolio.TotalRuns != 2 || portfolio.TotalEstimatedCostUSD != 11.5 || portfolio.TotalOverageCostUSD != 4.0 || portfolio.Recommendation != "stabilize-security-takeovers" {
		t.Fatalf("unexpected portfolio: %+v", portfolio)
	}
	portfolioReport := RenderOrchestrationPortfolioReport(portfolio, &SharedViewContext{Filters: []SharedViewFilter{{Label: "Team", Value: "engineering"}}, ResultCount: 0})
	for _, fragment := range []string{"- Collaboration Mix: cross-functional=1 tier-limited=1", "- Tier Mix: premium=1 standard=1", "- Entitlement Mix: included=1 upgrade-required=1", "- Billing Models: premium-included=1 standard-blocked=1", "- Takeover Queue: pending=1 recommendation=expedite-security-review"} {
		if !strings.Contains(portfolioReport, fragment) {
			t.Fatalf("expected %q in portfolio report, got %s", fragment, portfolioReport)
		}
	}
	if !strings.Contains(RenderOrchestrationOverviewPage(portfolio), "<title>Orchestration Overview") {
		t.Fatalf("expected overview page title")
	}

	page := BuildBillingEntitlementsPage(portfolio, "OpenAGI Revenue Cloud", "Standard", "2026-03")
	if page.RunCount != 2 || page.TotalIncludedUsageUnits != 5 || page.TotalOverageUsageUnits != 1 || page.TotalEstimatedCostUSD != 11.5 || page.TotalOverageCostUSD != 4.0 || page.UpgradeRequiredCount != 1 || page.Recommendation != "resolve-plan-gaps" {
		t.Fatalf("unexpected billing page: %+v", page)
	}
	if !strings.Contains(RenderBillingEntitlementsReport(page), "run-canvas: task=OPE-66-canvas entitlement=upgrade-required billing=standard-blocked") {
		t.Fatalf("expected billing report content")
	}
	if !strings.Contains(RenderBillingEntitlementsPage(page), "<title>Billing & Entitlements") {
		t.Fatalf("expected billing page html")
	}
}

func TestBuildBillingEntitlementsPageFromLedger(t *testing.T) {
	page := BuildBillingEntitlementsPageFromLedger([]map[string]any{
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
				map[string]any{"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "operations", "required_approvals": []any{"ops-manager"}}},
			},
		},
	}, "OpenAGI Revenue Cloud", "Standard", "2026-03")

	if page.RunCount != 2 || page.Recommendation != "resolve-plan-gaps" || page.TotalOverageCostUSD != 4.0 || page.Charges[1].HandoffTeam != "operations" {
		t.Fatalf("unexpected billing page from ledger: %+v", page)
	}
	if len(page.Charges[1].BlockedCapabilities) != 1 || page.Charges[1].BlockedCapabilities[0] != "customer-success" {
		t.Fatalf("unexpected blocked capabilities: %+v", page.Charges[1])
	}
}

func intPtr(value int) *int {
	return &value
}

func queueForCanvasTests() TakeoverQueue {
	return TakeoverQueue{
		Name:            "Cross-Team Takeovers",
		Period:          "2026-03-10",
		PendingRequests: 1,
		TeamCounts:      map[string]int{"security": 1},
		Recommendation:  "expedite-security-review",
	}
}
