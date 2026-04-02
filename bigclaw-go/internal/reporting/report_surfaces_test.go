package reporting

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"bigclaw-go/internal/collaboration"
	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/observability"
	"bigclaw-go/internal/workflow"
)

func TestRenderAndWriteReport(t *testing.T) {
	content := RenderIssueValidationReport("BIG-101", "v0.1", "sandbox", "pass")
	out := filepath.Join(t.TempDir(), "report.md")
	if err := WriteReport(out, content); err != nil {
		t.Fatalf("write report: %v", err)
	}
	body, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(body), "BIG-101") || !strings.Contains(string(body), "pass") {
		t.Fatalf("unexpected report content: %s", string(body))
	}
}

func TestConsoleActionStateReflectsEnabledFlag(t *testing.T) {
	if got := (ConsoleAction{Enabled: true}).State(); got != "enabled" {
		t.Fatalf("state = %q, want enabled", got)
	}
	if got := (ConsoleAction{Enabled: false}).State(); got != "disabled" {
		t.Fatalf("state = %q, want disabled", got)
	}
}

func TestReportStudioRendersNarrativeSectionsAndExportBundle(t *testing.T) {
	studio := ReportStudio{
		Name:     "Executive Weekly Narrative",
		IssueID:  "OPE-112",
		Audience: "executive",
		Period:   "2026-W11",
		Summary:  "Delivery recovered after approval bottlenecks were cleared in the second half of the week.",
		Sections: []NarrativeSection{
			{Heading: "What changed", Body: "Approval queue depth fell from 5 to 1.", Evidence: []string{"queue-control-center"}},
			{Heading: "What needs attention", Body: "Security takeover requests still cluster.", Evidence: []string{"takeover-queue"}},
		},
		ActionItems:   []string{"Publish the markdown export to leadership"},
		SourceReports: []string{"reports/weekly-operations.md"},
	}
	markdown := RenderReportStudioReport(studio)
	plain := RenderReportStudioPlainText(studio)
	html := RenderReportStudioHTML(studio)
	artifacts, err := WriteReportStudioBundle(filepath.Join(t.TempDir(), "studio"), studio)
	if err != nil {
		t.Fatalf("write report studio bundle: %v", err)
	}
	if !studio.Ready() || studio.Recommendation() != "publish" {
		t.Fatalf("expected studio to be publish-ready: %+v", studio)
	}
	if !strings.Contains(markdown, "# Report Studio") || !strings.Contains(markdown, "### What changed") {
		t.Fatalf("unexpected markdown: %s", markdown)
	}
	if !strings.Contains(plain, "Recommendation: publish") || !strings.Contains(html, "<h1>Executive Weekly Narrative</h1>") {
		t.Fatalf("unexpected plain/html output")
	}
	for _, path := range []string{artifacts.MarkdownPath, artifacts.HTMLPath, artifacts.TextPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing bundle artifact %s: %v", path, err)
		}
	}
	if !strings.Contains(artifacts.MarkdownPath, "executive-weekly-narrative.md") {
		t.Fatalf("unexpected markdown path: %s", artifacts.MarkdownPath)
	}
}

func TestReportStudioRequiresSummaryAndCompleteSections(t *testing.T) {
	studio := ReportStudio{
		Name:     "Draft Narrative",
		IssueID:  "OPE-112",
		Audience: "operations",
		Period:   "2026-W11",
		Summary:  "",
		Sections: []NarrativeSection{{Heading: "Open risks", Body: ""}},
	}
	if studio.Ready() || studio.Recommendation() != "draft" {
		t.Fatalf("expected draft studio: %+v", studio)
	}
}

func TestPilotScorecardAndPortfolio(t *testing.T) {
	scorecard := PilotScorecard{
		IssueID:            "OPE-60",
		Customer:           "Design Partner A",
		Period:             "2026-Q2",
		Metrics:            []PilotMetric{{Name: "Automation coverage", Baseline: 35, Current: 82, Target: 80, Unit: "%", HigherIsBetter: true}, {Name: "Manual review time", Baseline: 12, Current: 4, Target: 5, Unit: "h"}},
		MonthlyBenefit:     12000,
		MonthlyCost:        2500,
		ImplementationCost: 18000,
		BenchmarkScore:     96,
		BenchmarkPassed:    true,
	}
	if scorecard.MetricsMet() != 2 || scorecard.Recommendation() != "go" {
		t.Fatalf("unexpected scorecard state: %+v", scorecard)
	}
	if months := scorecard.PaybackMonths(); months == nil || *months != 1.9 {
		t.Fatalf("unexpected payback months: %+v", months)
	}
	rendered := RenderPilotScorecard(scorecard)
	for _, fragment := range []string{"Annualized ROI:", "Recommendation: go", "Benchmark Score: 96", "Automation coverage"} {
		if !strings.Contains(rendered, fragment) {
			t.Fatalf("expected %q in scorecard report, got %s", fragment, rendered)
		}
	}

	hold := PilotScorecard{
		IssueID:            "OPE-60",
		Customer:           "Design Partner B",
		Period:             "2026-Q2",
		Metrics:            []PilotMetric{{Name: "Backlog aging", Baseline: 5, Current: 7, Target: 4, Unit: "d"}},
		MonthlyBenefit:     1000,
		MonthlyCost:        3000,
		ImplementationCost: 12000,
		BenchmarkPassed:    false,
	}
	if hold.MonthlyNetValue() != -2000 || hold.PaybackMonths() != nil || hold.Recommendation() != "hold" {
		t.Fatalf("unexpected hold scorecard: %+v", hold)
	}

	portfolio := PilotPortfolio{
		Name:   "Design Partners",
		Period: "2026-H1",
		Scorecards: []PilotScorecard{
			{IssueID: "OPE-60", Customer: "Partner A", Period: "2026-Q2", Metrics: []PilotMetric{{Name: "Coverage", Baseline: 40, Current: 85, Target: 80, Unit: "%", HigherIsBetter: true}}, MonthlyBenefit: 15000, MonthlyCost: 3000, ImplementationCost: 18000, BenchmarkScore: 97, BenchmarkPassed: true},
			{IssueID: "OPE-61", Customer: "Partner B", Period: "2026-Q2", Metrics: []PilotMetric{{Name: "Cycle time", Baseline: 12, Current: 7, Target: 5, Unit: "h"}}, MonthlyBenefit: 9000, MonthlyCost: 2500, ImplementationCost: 12000, BenchmarkScore: 88, BenchmarkPassed: true},
		},
	}
	if portfolio.TotalMonthlyNetValue() != 18500 || portfolio.AverageROI() <= 0 {
		t.Fatalf("unexpected portfolio rollup: %+v", portfolio)
	}
	if got, want := portfolio.RecommendationCounts(), map[string]int{"go": 1, "iterate": 1, "hold": 0}; !reflect.DeepEqual(got, want) {
		t.Fatalf("recommendation counts = %+v, want %+v", got, want)
	}
	if portfolio.Recommendation() != "continue" {
		t.Fatalf("unexpected portfolio recommendation: %+v", portfolio)
	}
	report := RenderPilotPortfolioReport(portfolio)
	for _, fragment := range []string{"Recommendation Mix: go=1 iterate=1 hold=0", "Partner A: recommendation=go", "Partner B: recommendation=iterate"} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in portfolio report, got %s", fragment, report)
		}
	}
}

func TestIssueClosureAndChecklists(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "validation.md")
	decision := EvaluateIssueClosure("BIG-602", reportPath, false, nil, nil)
	if decision.Allowed || decision.Reason != "validation report required before closing issue" || ValidationReportExists(reportPath) {
		t.Fatalf("unexpected missing validation decision: %+v", decision)
	}
	if err := WriteReport(reportPath, "# Validation\n\nfailed"); err != nil {
		t.Fatalf("write failed report: %v", err)
	}
	decision = EvaluateIssueClosure("BIG-602", reportPath, false, nil, nil)
	if decision.Allowed || decision.Reason != "validation failed; issue must remain open" || !ValidationReportExists(reportPath) {
		t.Fatalf("unexpected failed validation decision: %+v", decision)
	}
	if err := WriteReport(reportPath, RenderIssueValidationReport("BIG-602", "v0.1", "sandbox", "pass")); err != nil {
		t.Fatalf("write passing report: %v", err)
	}
	decision = EvaluateIssueClosure("BIG-602", reportPath, true, nil, nil)
	if !decision.Allowed || decision.Reason != "validation report and launch checklist requirements satisfied; issue can be closed" || decision.ReportPath != reportPath {
		t.Fatalf("unexpected completed validation decision: %+v", decision)
	}

	root := t.TempDir()
	runbook := filepath.Join(root, "runbook.md")
	faq := filepath.Join(root, "faq.md")
	if err := WriteReport(runbook, "# Runbook\n\nready"); err != nil {
		t.Fatalf("write runbook: %v", err)
	}
	launch := BuildLaunchChecklist("BIG-1003", []DocumentationArtifact{{Name: "runbook", Path: runbook}, {Name: "faq", Path: faq}}, []LaunchChecklistItem{{Name: "Operations handoff", Evidence: []string{"runbook"}}, {Name: "Support handoff", Evidence: []string{"faq"}}})
	if got, want := launch.DocumentationStatus, map[string]bool{"runbook": true, "faq": false}; !reflect.DeepEqual(got, want) {
		t.Fatalf("launch doc status = %+v, want %+v", got, want)
	}
	if launch.CompletedItems != 1 || !reflect.DeepEqual(launch.MissingDocumentation, []string{"faq"}) || launch.Ready {
		t.Fatalf("unexpected launch checklist: %+v", launch)
	}
	launchReport := RenderLaunchChecklistReport(launch)
	for _, fragment := range []string{"runbook: available=true", "faq: available=false", "Support handoff: completed=false evidence=faq"} {
		if !strings.Contains(launchReport, fragment) {
			t.Fatalf("expected %q in launch report, got %s", fragment, launchReport)
		}
	}

	validationBundle := filepath.Join(root, "validation-bundle.md")
	releaseNotes := filepath.Join(root, "release-notes.md")
	if err := WriteReport(validationBundle, "# Validation Bundle\n\nready"); err != nil {
		t.Fatalf("write validation bundle: %v", err)
	}
	finalChecklist := BuildFinalDeliveryChecklist("BIG-4702", []DocumentationArtifact{{Name: "validation-bundle", Path: validationBundle}, {Name: "release-notes", Path: releaseNotes}}, []DocumentationArtifact{{Name: "runbook", Path: filepath.Join(root, "runbook-extra.md")}, {Name: "faq", Path: filepath.Join(root, "faq-extra.md")}})
	if got, want := finalChecklist.RequiredOutputStatus, map[string]bool{"validation-bundle": true, "release-notes": false}; !reflect.DeepEqual(got, want) {
		t.Fatalf("required output status = %+v, want %+v", got, want)
	}
	if finalChecklist.GeneratedRequiredOutputs != 1 || finalChecklist.GeneratedRecommendedDocumentation != 0 || !reflect.DeepEqual(finalChecklist.MissingRequiredOutputs, []string{"release-notes"}) || finalChecklist.Ready {
		t.Fatalf("unexpected final checklist: %+v", finalChecklist)
	}
	finalReport := RenderFinalDeliveryChecklistReport(finalChecklist)
	for _, fragment := range []string{"Required Outputs Generated: 1/2", "Recommended Docs Generated: 0/2", "release-notes: available=false", "runbook: available=false"} {
		if !strings.Contains(finalReport, fragment) {
			t.Fatalf("expected %q in final delivery report, got %s", fragment, finalReport)
		}
	}

	decision = EvaluateIssueClosure("BIG-1003", reportPath, true, &launch, nil)
	if decision.Allowed || decision.Reason != "launch checklist incomplete; linked documentation missing or empty" {
		t.Fatalf("unexpected incomplete launch decision: %+v", decision)
	}
	decision = EvaluateIssueClosure("BIG-4702", reportPath, true, nil, &finalChecklist)
	if decision.Allowed || decision.Reason != "final delivery checklist incomplete; required outputs missing" {
		t.Fatalf("unexpected incomplete final delivery decision: %+v", decision)
	}

	launchFAQ := filepath.Join(root, "launch-faq.md")
	if err := WriteReport(launchFAQ, "# FAQ\n\nready"); err != nil {
		t.Fatalf("write faq: %v", err)
	}
	readyLaunch := BuildLaunchChecklist("BIG-1003", []DocumentationArtifact{{Name: "runbook", Path: runbook}, {Name: "launch-faq", Path: launchFAQ}}, []LaunchChecklistItem{{Name: "Launch comms", Evidence: []string{"runbook", "launch-faq"}}})
	decision = EvaluateIssueClosure("BIG-1003", reportPath, true, &readyLaunch, nil)
	if !decision.Allowed || decision.Reason != "validation report and launch checklist requirements satisfied; issue can be closed" {
		t.Fatalf("unexpected ready launch decision: %+v", decision)
	}
	if err := WriteReport(releaseNotes, "# Release Notes\n\nready"); err != nil {
		t.Fatalf("write release notes: %v", err)
	}
	readyFinal := BuildFinalDeliveryChecklist("BIG-4702", []DocumentationArtifact{{Name: "validation-bundle", Path: validationBundle}, {Name: "release-notes", Path: releaseNotes}}, []DocumentationArtifact{{Name: "runbook", Path: filepath.Join(root, "runbook-extra.md")}})
	decision = EvaluateIssueClosure("BIG-4702", reportPath, true, nil, &readyFinal)
	if !decision.Allowed || decision.Reason != "validation report and final delivery checklist requirements satisfied; issue can be closed" {
		t.Fatalf("unexpected ready final decision: %+v", decision)
	}
}

func TestRenderSharedViewContextIncludesCollaborationAnnotations(t *testing.T) {
	view := SharedViewContext{
		Filters:       []SharedViewFilter{{Label: "Team", Value: "ops"}},
		ResultCount:   4,
		Collaboration: collaboration.BuildThread("dashboard", "ops-overview", []collaboration.Comment{{CommentID: "dashboard-comment-1", Author: "pm", Body: "Please review blocker copy with @ops and @eng."}}, []collaboration.Decision{{DecisionID: "dashboard-decision-1", Author: "ops", Outcome: "approved", Summary: "Keep the blocker module visible for managers."}}),
	}
	content := strings.Join(RenderSharedViewContext(view), "\n")
	for _, fragment := range []string{"## Collaboration", "Surface: dashboard", "Please review blocker copy with @ops and @eng.", "Keep the blocker module visible for managers."} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("expected %q in shared view context, got %s", fragment, content)
		}
	}
}

func TestAutoTriageCenterReportsAndFeedback(t *testing.T) {
	approvalRun := makeRun("OPE-76-risk", "run-risk", "vm", "needs-approval", "requires approval for high-risk task")
	failedRun := makeRun("OPE-76-browser", "run-browser", "browser", "failed", "browser session crashed")
	healthyRun := makeRun("OPE-76-ok", "run-ok", "docker", "approved", "default low risk path")
	center := BuildAutoTriageCenter([]observability.TaskRun{healthyRun, approvalRun, failedRun}, "Engineering Ops", "2026-03-10", nil)
	report := RenderAutoTriageCenterReport(center, 3)
	if center.FlaggedRuns != 2 || center.InboxSize != 2 || !reflect.DeepEqual(center.SeverityCounts, map[string]int{"critical": 1, "high": 1, "medium": 0}) {
		t.Fatalf("unexpected triage center summary: %+v", center)
	}
	if !reflect.DeepEqual(center.OwnerCounts, map[string]int{"security": 1, "engineering": 1, "operations": 0}) || center.Recommendation != "immediate-attention" {
		t.Fatalf("unexpected owner counts/recommendation: %+v", center)
	}
	if got := []string{center.Findings[0].RunID, center.Findings[1].RunID}; !reflect.DeepEqual(got, []string{"run-browser", "run-risk"}) {
		t.Fatalf("unexpected finding order: %+v", got)
	}
	if center.Inbox[0].Suggestions[0].Label != "replay candidate" || center.Inbox[0].Suggestions[0].Confidence < 0.55 {
		t.Fatalf("unexpected inbox suggestion: %+v", center.Inbox[0])
	}
	if center.Findings[0].NextAction != "replay run and inspect tool failures" || center.Findings[1].NextAction != "request approval and queue security review" {
		t.Fatalf("unexpected next actions: %+v", center.Findings)
	}
	if !center.Findings[0].Actions[4].Enabled || center.Findings[1].Actions[4].Enabled || center.Findings[1].Actions[6].Enabled {
		t.Fatalf("unexpected action states: %+v", center.Findings)
	}
	for _, fragment := range []string{"Flagged Runs: 2", "Inbox Size: 2", "Severity Mix: critical=1 high=1 medium=0", "Feedback Loop: accepted=0 rejected=0 pending=2", "run-browser: severity=critical owner=engineering status=failed", "run-risk: severity=high owner=security status=needs-approval", "actions=Drill Down [drill-down]", "Retry [retry] state=disabled target=run-risk reason=retry available after owner review"} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in triage report, got %s", fragment, report)
		}
	}

	viewReport := RenderAutoTriageCenterReport(BuildAutoTriageCenter([]observability.TaskRun{approvalRun}, "Engineering Ops", "2026-03-10", nil), 1, SharedViewContext{Filters: []SharedViewFilter{{Label: "Team", Value: "engineering"}}, ResultCount: 1, PartialData: []string{"Replay ledger data is still backfilling."}})
	for _, fragment := range []string{"## View State", "- State: partial-data", "- Team: engineering", "## Partial Data", "Replay ledger data is still backfilling."} {
		if !strings.Contains(viewReport, fragment) {
			t.Fatalf("expected %q in partial triage report, got %s", fragment, viewReport)
		}
	}

	similarRun := makeRun("OPE-100-browser-b", "run-browser-b", "browser", "failed", "browser session crashed")
	feedback := []TriageFeedbackRecord{NewTriageFeedbackRecord("run-browser-a", "replay run and inspect tool failures", "accepted", "ops-lead", "matched previous recovery path"), NewTriageFeedbackRecord("run-security", "request approval and queue security review", "rejected", "sec-reviewer", "approval already in flight")}
	center = BuildAutoTriageCenter([]observability.TaskRun{makeRun("OPE-100-browser-a", "run-browser-a", "browser", "failed", "browser session crashed"), similarRun, makeRun("OPE-100-security", "run-security", "vm", "needs-approval", "requires approval for high-risk task")}, "Auto Triage Center", "2026-03-11", feedback)
	report = RenderAutoTriageCenterReport(center, 3)
	if !reflect.DeepEqual(center.FeedbackCounts, map[string]int{"accepted": 1, "rejected": 1, "pending": 1}) {
		t.Fatalf("unexpected feedback counts: %+v", center.FeedbackCounts)
	}
	browserItem := center.Inbox[0]
	if browserItem.RunID != "run-browser-a" {
		browserItem = center.Inbox[1]
	}
	if browserItem.Suggestions[0].FeedbackStatus != "accepted" || browserItem.Suggestions[0].Evidence[0].RelatedRunID != "run-browser-b" || browserItem.Suggestions[0].Evidence[0].Score < 0.8 {
		t.Fatalf("unexpected browser evidence: %+v", browserItem)
	}
	if !strings.Contains(report, "similar=run-browser-b:") || !strings.Contains(report, "Feedback Loop: accepted=1 rejected=1 pending=1") {
		t.Fatalf("unexpected similarity report: %s", report)
	}
}

func TestTakeoverQueueAndOrchestrationReports(t *testing.T) {
	entries := []map[string]any{
		{"run_id": "run-sec", "task_id": "OPE-66-sec", "source": "linear", "summary": "requires approval for high-risk task", "audits": []map[string]any{{"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "security", "reason": "requires approval for high-risk task", "required_approvals": []any{"security-review"}}}}},
		{"run_id": "run-ops", "task_id": "OPE-66-ops", "source": "linear", "summary": "premium tier required for advanced cross-department orchestration", "audits": []map[string]any{{"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "operations", "reason": "premium tier required for advanced cross-department orchestration", "required_approvals": []any{"ops-manager"}}}}},
	}
	queue := BuildTakeoverQueueFromLedger(entries, "Cross-Team Takeovers", "2026-03-10")
	report := RenderTakeoverQueueReport(queue, 3)
	if queue.PendingRequests != 2 || !reflect.DeepEqual(queue.TeamCounts, map[string]int{"operations": 1, "security": 1}) || queue.ApprovalCount != 2 || queue.Recommendation != "expedite-security-review" {
		t.Fatalf("unexpected takeover queue: %+v", queue)
	}
	if got := []string{queue.Requests[0].RunID, queue.Requests[1].RunID}; !reflect.DeepEqual(got, []string{"run-ops", "run-sec"}) {
		t.Fatalf("unexpected request ordering: %+v", got)
	}
	if !queue.Requests[0].Actions[3].Enabled || queue.Requests[1].Actions[3].Enabled {
		t.Fatalf("unexpected escalate states: %+v", queue.Requests)
	}
	for _, fragment := range []string{"Pending Requests: 2", "Team Mix: operations=1 security=1", "run-sec: team=security status=pending task=OPE-66-sec approvals=security-review", "run-ops: team=operations status=pending task=OPE-66-ops approvals=ops-manager", "Escalate [escalate] state=disabled target=run-sec reason=security takeovers are already escalated"} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in takeover report, got %s", fragment, report)
		}
	}

	errorReport := RenderTakeoverQueueReport(BuildTakeoverQueueFromLedger(nil, "Cross-Team Takeovers", "2026-03-10"), 0, SharedViewContext{ResultCount: 0, Errors: []string{"Takeover approvals service timed out."}})
	for _, fragment := range []string{"- State: error", "- Summary: Unable to load data for the current filters.", "## Errors", "Takeover approvals service timed out."} {
		if !strings.Contains(errorReport, fragment) {
			t.Fatalf("expected %q in error takeover report, got %s", fragment, errorReport)
		}
	}

	run := makeRun("OPE-66-canvas", "run-canvas", "browser", "", "")
	run.Audits = append(run.Audits, observability.AuditItem{Action: "tool.invoke", Actor: "worker", Outcome: "success", Details: map[string]any{"tool": "browser"}})
	plan := workflow.OrchestrationPlan{TaskID: "OPE-66-canvas", CollaborationMode: "tier-limited", Handoffs: []workflow.DepartmentHandoff{{Department: "operations", Reason: "coordinate"}, {Department: "engineering", Reason: "execute", RequiredTools: []string{"browser"}}}}
	policy := workflow.OrchestrationPolicyDecision{Tier: "standard", UpgradeRequired: true, Reason: "premium tier required for advanced cross-department orchestration", BlockedDepartments: []string{"customer-success"}, EntitlementStatus: "upgrade-required", BillingModel: "standard-blocked", EstimatedCostUSD: 7.0, IncludedUsageUnits: 2, OverageUsageUnits: 1, OverageCostUSD: 4.0}
	handoff := &workflow.HandoffRequest{TargetTeam: "operations", Reason: policy.Reason, RequiredApprovals: []string{"ops-manager"}, Status: "pending"}
	canvas := BuildOrchestrationCanvas(run, plan, policy, handoff)
	canvasReport := RenderOrchestrationCanvas(canvas)
	if canvas.Recommendation != "resolve-entitlement-gap" || !reflect.DeepEqual(canvas.ActiveTools, []string{"browser"}) || !canvas.Actions[3].Enabled || canvas.Actions[4].Enabled {
		t.Fatalf("unexpected canvas: %+v", canvas)
	}
	for _, fragment := range []string{"# Orchestration Canvas", "- Tier: standard", "- Entitlement Status: upgrade-required", "- Billing Model: standard-blocked", "- Estimated Cost (USD): 7.00", "- Handoff Team: operations", "- Recommendation: resolve-entitlement-gap", "## Actions", "Escalate [escalate] state=enabled target=run-canvas"} {
		if !strings.Contains(canvasReport, fragment) {
			t.Fatalf("expected %q in canvas report, got %s", fragment, canvasReport)
		}
	}

	ledgerCanvas := BuildOrchestrationCanvasFromLedgerEntry(map[string]any{"run_id": "run-flow-1", "task_id": "OPE-113", "audits": []map[string]any{
		{"action": "orchestration.plan", "actor": "scheduler", "outcome": "enabled", "timestamp": "2026-03-11T11:00:00Z", "details": map[string]any{"collaboration_mode": "cross-functional", "departments": []any{"operations", "engineering"}, "approvals": []any{}}},
		{"action": "orchestration.policy", "actor": "scheduler", "outcome": "enabled", "timestamp": "2026-03-11T11:01:00Z", "details": map[string]any{"tier": "premium", "entitlement_status": "included", "billing_model": "premium-included"}},
		{"action": "collaboration.comment", "actor": "ops-lead", "outcome": "recorded", "timestamp": "2026-03-11T11:02:00Z", "details": map[string]any{"surface": "flow", "comment_id": "flow-comment-1", "body": "Route @eng once the dashboard note is resolved."}},
		{"action": "collaboration.decision", "actor": "eng-manager", "outcome": "accepted", "timestamp": "2026-03-11T11:03:00Z", "details": map[string]any{"surface": "flow", "decision_id": "flow-decision-1", "summary": "Engineering owns the next flow handoff."}},
	}})
	if ledgerCanvas.CollaborationThread == nil || ledgerCanvas.Recommendation != "resolve-flow-comments" {
		t.Fatalf("unexpected ledger canvas: %+v", ledgerCanvas)
	}
	ledgerReport := RenderOrchestrationCanvas(ledgerCanvas)
	for _, fragment := range []string{"## Collaboration", "Route @eng once the dashboard note is resolved.", "Engineering owns the next flow handoff."} {
		if !strings.Contains(ledgerReport, fragment) {
			t.Fatalf("expected %q in ledger canvas report, got %s", fragment, ledgerReport)
		}
	}
}

func TestOrchestrationPortfolioAndBillingPages(t *testing.T) {
	canvases := []OrchestrationCanvas{
		{TaskID: "OPE-66-a", RunID: "run-a", CollaborationMode: "cross-functional", Departments: []string{"operations", "engineering", "security"}, Tier: "premium", EntitlementStatus: "included", BillingModel: "premium-included", EstimatedCostUSD: 4.5, IncludedUsageUnits: 3, HandoffTeam: "security", Actions: []ConsoleAction{{ActionID: "drill-down", Label: "Drill Down", Target: "run-a", Enabled: true}}},
		{TaskID: "OPE-66-b", RunID: "run-b", CollaborationMode: "tier-limited", Departments: []string{"operations", "engineering"}, Tier: "standard", UpgradeRequired: true, EntitlementStatus: "upgrade-required", BillingModel: "standard-blocked", EstimatedCostUSD: 7.0, IncludedUsageUnits: 2, OverageUsageUnits: 1, OverageCostUSD: 4.0, BlockedDepartments: []string{"customer-success"}, HandoffTeam: "operations", Actions: []ConsoleAction{{ActionID: "drill-down", Label: "Drill Down", Target: "run-b", Enabled: true}}},
	}
	queue := BuildTakeoverQueueFromLedger([]map[string]any{
		{"run_id": "run-a", "task_id": "OPE-66-a", "source": "linear", "audits": []map[string]any{{"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "security", "reason": "risk", "required_approvals": []any{"security-review"}}}}},
		{"run_id": "run-b", "task_id": "OPE-66-b", "source": "linear", "audits": []map[string]any{{"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "operations", "reason": "entitlement", "required_approvals": []any{"ops-manager"}}}}},
	}, "Cross-Team Takeovers", "2026-03-10")
	portfolio := BuildOrchestrationPortfolio(canvases, "Cross-Team Portfolio", "2026-03-10", queue)
	report := RenderOrchestrationPortfolioReport(portfolio)
	if portfolio.TotalRuns != 2 || !reflect.DeepEqual(portfolio.CollaborationModes, map[string]int{"cross-functional": 1, "tier-limited": 1}) || !reflect.DeepEqual(portfolio.TierCounts, map[string]int{"premium": 1, "standard": 1}) || !reflect.DeepEqual(portfolio.EntitlementCounts, map[string]int{"included": 1, "upgrade-required": 1}) || !reflect.DeepEqual(portfolio.BillingModelCounts, map[string]int{"premium-included": 1, "standard-blocked": 1}) || portfolio.TotalEstimatedCostUSD != 11.5 || portfolio.TotalOverageCostUSD != 4.0 || portfolio.UpgradeRequiredCount != 1 || portfolio.ActiveHandoffs != 2 || portfolio.Recommendation != "stabilize-security-takeovers" {
		t.Fatalf("unexpected portfolio: %+v", portfolio)
	}
	for _, fragment := range []string{"# Orchestration Portfolio Report", "- Collaboration Mix: cross-functional=1 tier-limited=1", "- Tier Mix: premium=1 standard=1", "- Entitlement Mix: included=1 upgrade-required=1", "- Billing Models: premium-included=1 standard-blocked=1", "- Estimated Cost (USD): 11.50", "- Overage Cost (USD): 4.00", "- Takeover Queue: pending=2 recommendation=expedite-security-review", "- run-a: mode=cross-functional tier=premium entitlement=included billing=premium-included estimated_cost_usd=4.50 overage_cost_usd=0.00 upgrade_required=false handoff=security", "actions=Drill Down [drill-down]"} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in portfolio report, got %s", fragment, report)
		}
	}
	emptyReport := RenderOrchestrationPortfolioReport(BuildOrchestrationPortfolio(nil, "Cross-Team Portfolio", "2026-03-10"), SharedViewContext{Filters: []SharedViewFilter{{Label: "Team", Value: "engineering"}}, ResultCount: 0})
	for _, fragment := range []string{"- State: empty", "- Summary: No records match the current filters.", "## Filters"} {
		if !strings.Contains(emptyReport, fragment) {
			t.Fatalf("expected %q in empty portfolio report, got %s", fragment, emptyReport)
		}
	}
	page := RenderOrchestrationOverviewPage(portfolio)
	for _, fragment := range []string{"<title>Orchestration Overview", "Cross-Team Portfolio", "stabilize-security-takeovers", "Estimated Cost", "premium-included", "pending=2 recommendation=expedite-security-review", "run-a", "actions=Drill Down [drill-down]"} {
		if !strings.Contains(page, fragment) {
			t.Fatalf("expected %q in overview page, got %s", fragment, page)
		}
	}

	ledgerPortfolio := BuildOrchestrationPortfolioFromLedger([]map[string]any{
		{"run_id": "run-a", "task_id": "OPE-66-a", "audits": []map[string]any{{"action": "orchestration.plan", "outcome": "ready", "details": map[string]any{"collaboration_mode": "cross-functional", "departments": []any{"operations", "engineering", "security"}, "approvals": []any{"security-review"}}}, {"action": "orchestration.policy", "outcome": "enabled", "details": map[string]any{"tier": "premium", "entitlement_status": "included", "billing_model": "premium-included", "estimated_cost_usd": 4.5, "included_usage_units": 3, "blocked_departments": []any{}}}, {"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "security", "reason": "approval required", "required_approvals": []any{"security-review"}}}}},
		{"run_id": "run-b", "task_id": "OPE-66-b", "audits": []map[string]any{{"action": "orchestration.plan", "outcome": "ready", "details": map[string]any{"collaboration_mode": "tier-limited", "departments": []any{"operations", "engineering"}, "approvals": []any{}}}, {"action": "orchestration.policy", "outcome": "upgrade-required", "details": map[string]any{"tier": "standard", "entitlement_status": "upgrade-required", "billing_model": "standard-blocked", "estimated_cost_usd": 7.0, "included_usage_units": 2, "overage_usage_units": 1, "overage_cost_usd": 4.0, "blocked_departments": []any{"customer-success"}}}, {"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "operations", "reason": "entitlement gap", "required_approvals": []any{"ops-manager"}}}}},
	}, "Ledger Portfolio", "2026-03-10")
	if ledgerPortfolio.TotalRuns != 2 || ledgerPortfolio.TakeoverQueue == nil || ledgerPortfolio.TakeoverQueue.PendingRequests != 2 || ledgerPortfolio.Recommendation != "stabilize-security-takeovers" {
		t.Fatalf("unexpected ledger portfolio: %+v", ledgerPortfolio)
	}

	billingPage := BuildBillingEntitlementsPage(OrchestrationPortfolio{Name: "Revenue Ops", Period: "2026-03", Canvases: []OrchestrationCanvas{
		{TaskID: "OPE-104-a", RunID: "run-billing-a", CollaborationMode: "cross-functional", Departments: []string{"operations", "engineering", "security"}, Tier: "premium", EntitlementStatus: "included", BillingModel: "premium-included", EstimatedCostUSD: 4.5, IncludedUsageUnits: 3, HandoffTeam: "security", Recommendation: "review-security-takeover"},
		{TaskID: "OPE-104-b", RunID: "run-billing-b", CollaborationMode: "tier-limited", Departments: []string{"operations", "engineering"}, Tier: "standard", UpgradeRequired: true, EntitlementStatus: "upgrade-required", BillingModel: "standard-blocked", EstimatedCostUSD: 7.0, IncludedUsageUnits: 2, OverageUsageUnits: 1, OverageCostUSD: 4.0, BlockedDepartments: []string{"customer-success"}, HandoffTeam: "operations"},
	}}, "OpenAGI Revenue Cloud", "Standard", "2026-03")
	billingReport := RenderBillingEntitlementsReport(billingPage)
	if billingPage.RunCount != 2 || billingPage.TotalIncludedUsageUnits != 5 || billingPage.TotalOverageUsageUnits != 1 || billingPage.TotalEstimatedCostUSD != 11.5 || billingPage.TotalOverageCostUSD != 4.0 || billingPage.UpgradeRequiredCount != 1 || !reflect.DeepEqual(billingPage.EntitlementCounts, map[string]int{"included": 1, "upgrade-required": 1}) || !reflect.DeepEqual(billingPage.BillingModelCounts, map[string]int{"premium-included": 1, "standard-blocked": 1}) || !reflect.DeepEqual(billingPage.BlockedCapabilities, []string{"customer-success"}) || billingPage.Recommendation != "resolve-plan-gaps" {
		t.Fatalf("unexpected billing page: %+v", billingPage)
	}
	for _, fragment := range []string{"# Billing & Entitlements Report", "- Workspace: OpenAGI Revenue Cloud", "- Overage Cost (USD): 4.00", "- run-billing-b: task=OPE-104-b entitlement=upgrade-required billing=standard-blocked"} {
		if !strings.Contains(billingReport, fragment) {
			t.Fatalf("expected %q in billing report, got %s", fragment, billingReport)
		}
	}
	billingHTML := RenderBillingEntitlementsPage(BillingEntitlementsPage{WorkspaceName: "OpenAGI Revenue Cloud", PlanName: "Premium", BillingPeriod: "2026-03", Charges: []BillingRunCharge{{RunID: "run-billing-a", TaskID: "OPE-104-a", EntitlementStatus: "included", BillingModel: "premium-included", EstimatedCostUSD: 4.5, IncludedUsageUnits: 3, Recommendation: "review-security-takeover"}}})
	for _, fragment := range []string{"<title>Billing & Entitlements", "OpenAGI Revenue Cloud", "Premium plan for 2026-03", "Charge Feed", "premium-included"} {
		if !strings.Contains(billingHTML, fragment) {
			t.Fatalf("expected %q in billing html, got %s", fragment, billingHTML)
		}
	}

	ledgerBilling := BuildBillingEntitlementsPageFromLedger([]map[string]any{
		{"run_id": "run-ledger-a", "task_id": "OPE-104-a", "audits": []map[string]any{{"action": "orchestration.plan", "outcome": "ready", "details": map[string]any{"collaboration_mode": "cross-functional", "departments": []any{"operations", "engineering", "security"}, "approvals": []any{"security-review"}}}, {"action": "orchestration.policy", "outcome": "enabled", "details": map[string]any{"tier": "premium", "entitlement_status": "included", "billing_model": "premium-included", "estimated_cost_usd": 4.5, "included_usage_units": 3, "blocked_departments": []any{}}}}},
		{"run_id": "run-ledger-b", "task_id": "OPE-104-b", "audits": []map[string]any{{"action": "orchestration.plan", "outcome": "ready", "details": map[string]any{"collaboration_mode": "tier-limited", "departments": []any{"operations", "engineering"}, "approvals": []any{}}}, {"action": "orchestration.policy", "outcome": "upgrade-required", "details": map[string]any{"tier": "standard", "entitlement_status": "upgrade-required", "billing_model": "standard-blocked", "estimated_cost_usd": 7.0, "included_usage_units": 2, "overage_usage_units": 1, "overage_cost_usd": 4.0, "blocked_departments": []any{"customer-success"}}}, {"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "operations", "reason": "entitlement gap", "required_approvals": []any{"ops-manager"}}}}},
	}, "OpenAGI Revenue Cloud", "Standard", "2026-03")
	if ledgerBilling.RunCount != 2 || ledgerBilling.Recommendation != "resolve-plan-gaps" || ledgerBilling.TotalOverageCostUSD != 4.0 || !reflect.DeepEqual(ledgerBilling.Charges[1].BlockedCapabilities, []string{"customer-success"}) || ledgerBilling.Charges[1].HandoffTeam != "operations" {
		t.Fatalf("unexpected ledger billing page: %+v", ledgerBilling)
	}
}

func TestUTCimestamps(t *testing.T) {
	record := NewTriageFeedbackRecord("run-1", "classify", "accepted", "ops", "")
	if !strings.HasSuffix(record.Timestamp, "Z") {
		t.Fatalf("expected UTC timestamp, got %s", record.Timestamp)
	}
	parsed, err := time.Parse(time.RFC3339, record.Timestamp)
	if err != nil || parsed.Location() != time.UTC {
		t.Fatalf("unexpected triage timestamp parse: %v %v", parsed, err)
	}

	content := RenderIssueValidationReport("BIG-900", "v1", "repo", "pass")
	line := ""
	for _, candidate := range strings.Split(content, "\n") {
		if strings.HasPrefix(candidate, "- 生成时间:") {
			line = candidate
			break
		}
	}
	if line == "" {
		t.Fatalf("missing timestamp line in validation report: %s", content)
	}
	value := strings.SplitN(line, ": ", 2)[1]
	if !strings.HasSuffix(value, "Z") {
		t.Fatalf("expected UTC validation timestamp, got %s", value)
	}
	parsed, err = time.Parse(time.RFC3339, value)
	if err != nil || parsed.Location() != time.UTC {
		t.Fatalf("unexpected validation timestamp parse: %v %v", parsed, err)
	}
}

func makeRun(taskID, runID, medium, status, reason string) observability.TaskRun {
	task := domain.Task{ID: taskID, Source: "linear", Title: taskID}
	run := observability.NewTaskRun(task, runID, medium)
	run.Audit("scheduler.decision", "scheduler", status, map[string]any{"reason": reason})
	run.Finalize(status, reason)
	return *run
}
