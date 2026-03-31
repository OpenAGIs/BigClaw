package reportscompat

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/repo"
)

func makeSharedView(resultCount int, loading bool, errors, partial []string) *SharedViewContext {
	return &SharedViewContext{
		Filters:     []SharedViewFilter{{Label: "Team", Value: "engineering"}, {Label: "Window", Value: "2026-03-10"}},
		ResultCount: resultCount,
		Loading:     loading,
		Errors:      errors,
		PartialData: partial,
		LastUpdated: "2026-03-11T09:00:00Z",
	}
}

func TestRenderAndWriteReport(t *testing.T) {
	content := RenderIssueValidationReport("BIG-101", "v0.1", "sandbox", "pass")
	out := filepath.Join(t.TempDir(), "report.md")
	if err := WriteReport(out, content); err != nil {
		t.Fatalf("write report: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "BIG-101") || !strings.Contains(text, "pass") {
		t.Fatalf("unexpected report: %s", text)
	}
}

func TestConsoleActionStateReflectsEnabledFlag(t *testing.T) {
	enabled := NewConsoleAction("retry", "Retry", "run-1")
	disabled := ConsoleAction{ID: "pause", Label: "Pause", Target: "run-1", Enabled: false, Reason: "already completed"}
	if enabled.State() != "enabled" || disabled.State() != "disabled" {
		t.Fatalf("states = %s %s", enabled.State(), disabled.State())
	}
}

func TestReportStudioRendersNarrativeSectionsAndExportBundle(t *testing.T) {
	studio := ReportStudio{
		Name:          "Executive Weekly Narrative",
		IssueID:       "OPE-112",
		Audience:      "executive",
		Period:        "2026-W11",
		Summary:       "Delivery recovered after approval bottlenecks were cleared in the second half of the week.",
		Sections:      []NarrativeSection{{Heading: "What changed", Body: "Approval queue depth fell from 5 to 1.", Evidence: []string{"queue-control-center"}, Callouts: []string{"SLA risk contained"}}, {Heading: "What needs attention", Body: "Security takeover requests still cluster.", Evidence: []string{"takeover-queue"}, Callouts: []string{"Review staffing before Friday close"}}},
		ActionItems:   []string{"Publish the markdown export to leadership"},
		SourceReports: []string{"reports/weekly-operations.md"},
	}
	markdown := RenderReportStudioReport(studio)
	plain := RenderReportStudioPlainText(studio)
	html := RenderReportStudioHTML(studio)
	bundle, err := WriteReportStudioBundle(filepath.Join(t.TempDir(), "studio"), studio)
	if err != nil {
		t.Fatalf("bundle: %v", err)
	}
	if !studio.Ready() || studio.Recommendation() != "publish" || !strings.Contains(markdown, "# Report Studio") || !strings.Contains(markdown, "### What changed") || !strings.Contains(plain, "Recommendation: publish") || !strings.Contains(html, "<h1>Executive Weekly Narrative</h1>") || !strings.Contains(bundle.MarkdownPath, "executive-weekly-narrative.md") {
		t.Fatalf("unexpected studio output")
	}
	for _, path := range []string{bundle.MarkdownPath, bundle.HTMLPath, bundle.TextPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}
	}
}

func TestReportStudioRequiresSummaryAndCompleteSections(t *testing.T) {
	studio := ReportStudio{Name: "Draft Narrative", IssueID: "OPE-112", Audience: "operations", Period: "2026-W11", Summary: "", Sections: []NarrativeSection{{Heading: "Open risks", Body: ""}}}
	if studio.Ready() || studio.Recommendation() != "draft" {
		t.Fatalf("unexpected studio readiness")
	}
}

func TestRenderPilotScorecardIncludesROIAndRecommendation(t *testing.T) {
	scorecard := PilotScorecard{
		IssueID: "OPE-60", Customer: "Design Partner A", Period: "2026-Q2",
		Metrics:        []PilotMetric{{Name: "Automation coverage", Baseline: 35, Current: 82, Target: 80, Unit: "%", HigherIsBetter: true}, {Name: "Manual review time", Baseline: 12, Current: 4, Target: 5, Unit: "h", HigherIsBetter: false}},
		MonthlyBenefit: 12000, MonthlyCost: 2500, ImplementationCost: 18000, BenchmarkScore: 96, BenchmarkPassed: true,
	}
	content := RenderPilotScorecard(scorecard)
	payback := scorecard.PaybackMonths()
	if scorecard.MetricsMet() != 2 || scorecard.Recommendation() != "go" || payback == nil || *payback != 1.9 || !strings.Contains(content, "Annualized ROI: 200.0%") || !strings.Contains(content, "Recommendation: go") || !strings.Contains(content, "Benchmark Score: 96") || !strings.Contains(content, "Automation coverage") {
		t.Fatalf("unexpected scorecard content: %s", content)
	}
}

func TestPilotScorecardReturnsHoldWhenValueIsNegative(t *testing.T) {
	scorecard := PilotScorecard{IssueID: "OPE-60", Customer: "Design Partner B", Period: "2026-Q2", Metrics: []PilotMetric{{Name: "Backlog aging", Baseline: 5, Current: 7, Target: 4, Unit: "d", HigherIsBetter: false}}, MonthlyBenefit: 1000, MonthlyCost: 3000, ImplementationCost: 12000, BenchmarkPassed: false}
	if scorecard.MonthlyNetValue() != -2000 || scorecard.PaybackMonths() != nil || scorecard.Recommendation() != "hold" {
		t.Fatalf("unexpected negative scorecard")
	}
}

func TestIssueClosureValidationAndChecklists(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "validation.md")
	decision := EvaluateIssueClosure("BIG-602", reportPath, true, nil, nil)
	if decision.Allowed || decision.Reason != "validation report required before closing issue" || ValidationReportExists(reportPath) {
		t.Fatalf("unexpected missing validation behavior")
	}
	if err := WriteReport(reportPath, "# Validation\n\nfailed"); err != nil {
		t.Fatal(err)
	}
	decision = EvaluateIssueClosure("BIG-602", reportPath, false, nil, nil)
	if decision.Allowed || decision.Reason != "validation failed; issue must remain open" || !ValidationReportExists(reportPath) {
		t.Fatalf("unexpected failed validation behavior")
	}
	if err := WriteReport(reportPath, RenderIssueValidationReport("BIG-602", "v0.1", "sandbox", "pass")); err != nil {
		t.Fatal(err)
	}
	decision = EvaluateIssueClosure("BIG-602", reportPath, true, nil, nil)
	if !decision.Allowed || decision.Reason != "validation report and launch checklist requirements satisfied; issue can be closed" || decision.ReportPath != reportPath {
		t.Fatalf("unexpected success decision")
	}
}

func TestLaunchAndFinalDeliveryChecklists(t *testing.T) {
	dir := t.TempDir()
	runbook := filepath.Join(dir, "runbook.md")
	if err := WriteReport(runbook, "# Runbook\n\nready"); err != nil {
		t.Fatal(err)
	}
	checklist := BuildLaunchChecklist("BIG-1003", []DocumentationArtifact{{Name: "runbook", Path: runbook}, {Name: "faq", Path: filepath.Join(dir, "faq.md")}}, []LaunchChecklistItem{{Name: "Operations handoff", Evidence: []string{"runbook"}}, {Name: "Support handoff", Evidence: []string{"faq"}}})
	report := RenderLaunchChecklistReport(checklist)
	if !reflect.DeepEqual(checklist.DocumentationStatus, map[string]bool{"runbook": true, "faq": false}) || checklist.CompletedItems != 1 || !reflect.DeepEqual(checklist.MissingDocumentation, []string{"faq"}) || checklist.Ready() || !strings.Contains(report, "runbook: available=true") || !strings.Contains(report, "faq: available=false") || !strings.Contains(report, "Support handoff: completed=false evidence=faq") {
		t.Fatalf("unexpected launch checklist")
	}

	validationBundle := filepath.Join(dir, "validation-bundle.md")
	if err := WriteReport(validationBundle, "# Validation Bundle\n\nready"); err != nil {
		t.Fatal(err)
	}
	final := BuildFinalDeliveryChecklist("BIG-4702", []DocumentationArtifact{{Name: "validation-bundle", Path: validationBundle}, {Name: "release-notes", Path: filepath.Join(dir, "release-notes.md")}}, []DocumentationArtifact{{Name: "runbook", Path: filepath.Join(dir, "runbook.md")}, {Name: "faq", Path: filepath.Join(dir, "faq.md")}})
	finalReport := RenderFinalDeliveryChecklistReport(final)
	if !reflect.DeepEqual(final.RequiredOutputStatus, map[string]bool{"validation-bundle": true, "release-notes": false}) || !reflect.DeepEqual(final.RecommendedDocumentationStatus, map[string]bool{"runbook": true, "faq": false}) || final.GeneratedRequiredOutputs != 1 || final.GeneratedRecommendedDocs != 1 || !reflect.DeepEqual(final.MissingRequiredOutputs, []string{"release-notes"}) || !reflect.DeepEqual(final.MissingRecommendedDocs, []string{"faq"}) || final.Ready() || !strings.Contains(finalReport, "Required Outputs Generated: 1/2") || !strings.Contains(finalReport, "release-notes: available=false") {
		t.Fatalf("unexpected final delivery checklist")
	}
}

func TestIssueClosureWithLinkedChecklists(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "validation.md")
	if err := WriteReport(reportPath, RenderIssueValidationReport("BIG-1003", "v0.2", "staging", "pass")); err != nil {
		t.Fatal(err)
	}
	runbook := filepath.Join(dir, "runbook.md")
	faq := filepath.Join(dir, "launch-faq.md")
	if err := WriteReport(runbook, "# Runbook\n\nready"); err != nil {
		t.Fatal(err)
	}
	launch := BuildLaunchChecklist("BIG-1003", []DocumentationArtifact{{Name: "runbook", Path: runbook}, {Name: "launch-faq", Path: faq}}, []LaunchChecklistItem{{Name: "Launch comms", Evidence: []string{"runbook", "launch-faq"}}})
	decision := EvaluateIssueClosure("BIG-1003", reportPath, true, &launch, nil)
	if decision.Allowed || decision.Reason != "launch checklist incomplete; linked documentation missing or empty" {
		t.Fatalf("unexpected blocked launch decision")
	}
	if err := WriteReport(faq, "# FAQ\n\nready"); err != nil {
		t.Fatal(err)
	}
	launch = BuildLaunchChecklist("BIG-1003", []DocumentationArtifact{{Name: "runbook", Path: runbook}, {Name: "launch-faq", Path: faq}}, []LaunchChecklistItem{{Name: "Launch comms", Evidence: []string{"runbook", "launch-faq"}}})
	decision = EvaluateIssueClosure("BIG-1003", reportPath, true, &launch, nil)
	if !decision.Allowed || decision.Reason != "validation report and launch checklist requirements satisfied; issue can be closed" {
		t.Fatalf("unexpected allowed launch decision")
	}

	final := BuildFinalDeliveryChecklist("BIG-4702", []DocumentationArtifact{{Name: "validation-bundle", Path: filepath.Join(dir, "validation-bundle.md")}}, []DocumentationArtifact{{Name: "runbook", Path: runbook}})
	decision = EvaluateIssueClosure("BIG-4702", reportPath, true, nil, &final)
	if decision.Allowed || decision.Reason != "final delivery checklist incomplete; required outputs missing" {
		t.Fatalf("unexpected blocked final delivery decision")
	}
	if err := WriteReport(filepath.Join(dir, "validation-bundle.md"), "# Validation Bundle\n\nready"); err != nil {
		t.Fatal(err)
	}
	if err := WriteReport(filepath.Join(dir, "release-notes.md"), "# Release Notes\n\nready"); err != nil {
		t.Fatal(err)
	}
	final = BuildFinalDeliveryChecklist("BIG-4702", []DocumentationArtifact{{Name: "validation-bundle", Path: filepath.Join(dir, "validation-bundle.md")}, {Name: "release-notes", Path: filepath.Join(dir, "release-notes.md")}}, []DocumentationArtifact{{Name: "runbook", Path: runbook}})
	decision = EvaluateIssueClosure("BIG-4702", reportPath, true, nil, &final)
	if !decision.Allowed || decision.Reason != "validation report and final delivery checklist requirements satisfied; issue can be closed" {
		t.Fatalf("unexpected allowed final decision")
	}
}

func TestPilotPortfolioAndSharedViewCollaboration(t *testing.T) {
	portfolio := PilotPortfolio{
		Name: "Design Partners", Period: "2026-H1",
		Scorecards: []PilotScorecard{
			{IssueID: "OPE-60", Customer: "Partner A", Period: "2026-Q2", Metrics: []PilotMetric{{Name: "Coverage", Baseline: 40, Current: 85, Target: 80, Unit: "%", HigherIsBetter: true}}, MonthlyBenefit: 15000, MonthlyCost: 3000, ImplementationCost: 18000, BenchmarkScore: 97, BenchmarkPassed: true},
			{IssueID: "OPE-61", Customer: "Partner B", Period: "2026-Q2", Metrics: []PilotMetric{{Name: "Cycle time", Baseline: 12, Current: 7, Target: 5, Unit: "h", HigherIsBetter: false}}, MonthlyBenefit: 9000, MonthlyCost: 2500, ImplementationCost: 12000, BenchmarkScore: 88, BenchmarkPassed: true},
		},
	}
	content := RenderPilotPortfolioReport(portfolio)
	if portfolio.TotalMonthlyNetValue() != 18500 || portfolio.AverageROI() != 195.2 || !reflect.DeepEqual(portfolio.RecommendationCounts(), map[string]int{"go": 1, "iterate": 1, "hold": 0}) || portfolio.Recommendation() != "continue" || !strings.Contains(content, "Recommendation Mix: go=1 iterate=1 hold=0") || !strings.Contains(content, "Partner A: recommendation=go") || !strings.Contains(content, "Partner B: recommendation=iterate") {
		t.Fatalf("unexpected pilot portfolio")
	}

	view := SharedViewContext{
		Filters:     []SharedViewFilter{{Label: "Team", Value: "ops"}},
		ResultCount: 4,
		Collaboration: func() *repo.CollaborationThread {
			thread := repo.BuildCollaborationThread("dashboard", "ops-overview",
				[]repo.CollaborationComment{{CommentID: "dashboard-comment-1", Author: "pm", Body: "Please review blocker copy with @ops and @eng.", Anchor: "blockers"}},
				[]repo.DecisionNote{{DecisionID: "dashboard-decision-1", Author: "ops", Outcome: "approved", Summary: "Keep the blocker module visible for managers."}})
			return &thread
		}(),
	}
	content = strings.Join(RenderSharedViewContext(view), "\n")
	if !strings.Contains(content, "## Collaboration") || !strings.Contains(content, "Surface: dashboard") || !strings.Contains(content, "Please review blocker copy with @ops and @eng.") || !strings.Contains(content, "Keep the blocker module visible for managers.") {
		t.Fatalf("unexpected shared view collaboration")
	}
}

func TestAutoTriageCenterAndFeedbackLoop(t *testing.T) {
	approval := NewRunRecord(domain.Task{ID: "OPE-76-risk", Source: "linear", Title: "Prod approval"}, "run-risk", "vm")
	approval.Trace("scheduler.decide", "pending")
	approval.Audit("scheduler.decision", "scheduler", "pending", map[string]any{"reason": "requires approval for high-risk task"})
	approval.Finalize("needs-approval", "requires approval for high-risk task")

	failed := NewRunRecord(domain.Task{ID: "OPE-76-browser", Source: "linear", Title: "Replay browser task"}, "run-browser", "browser")
	failed.Trace("runtime.execute", "failed")
	failed.Audit("runtime.execute", "worker", "failed", map[string]any{"reason": "browser session crashed"})
	failed.Finalize("failed", "browser session crashed")

	healthy := NewRunRecord(domain.Task{ID: "OPE-76-ok", Source: "linear", Title: "Healthy run"}, "run-ok", "docker")
	healthy.Trace("scheduler.decide", "ok")
	healthy.Audit("scheduler.decision", "scheduler", "approved", map[string]any{"reason": "default low risk path"})
	healthy.Finalize("approved", "default low risk path")

	center := BuildAutoTriageCenter([]RunRecord{healthy, approval, failed}, "Engineering Ops", "2026-03-10", nil)
	report := RenderAutoTriageCenterReport(center, 3, nil)
	if center.FlaggedRuns != 2 || center.InboxSize != 2 || !reflect.DeepEqual(center.SeverityCounts, map[string]int{"critical": 1, "high": 1, "medium": 0}) || !reflect.DeepEqual(center.OwnerCounts, map[string]int{"security": 1, "engineering": 1, "operations": 0}) || center.Recommendation != "immediate-attention" || center.Inbox[0].Suggestions[0].Label != "replay candidate" || center.Inbox[0].Suggestions[0].Confidence < 0.55 || center.Findings[0].NextAction != "replay run and inspect tool failures" || center.Findings[1].NextAction != "request approval and queue security review" || !center.Findings[0].Actions[4].Enabled || center.Findings[1].Actions[4].Enabled || center.Findings[1].Actions[6].Enabled || !strings.Contains(report, "Flagged Runs: 2") || !strings.Contains(report, "Feedback Loop: accepted=0 rejected=0 pending=2") || !strings.Contains(report, "run-browser: severity=critical owner=engineering status=failed") || !strings.Contains(report, "Retry [retry] state=disabled target=run-risk reason=retry available after owner review") {
		t.Fatalf("unexpected auto triage")
	}

	report = RenderAutoTriageCenterReport(BuildAutoTriageCenter([]RunRecord{approval}, "Engineering Ops", "2026-03-10", nil), 1, makeSharedView(1, false, nil, []string{"Replay ledger data is still backfilling."}))
	if !strings.Contains(report, "## View State") || !strings.Contains(report, "- State: partial-data") || !strings.Contains(report, "- Team: engineering") || !strings.Contains(report, "## Partial Data") || !strings.Contains(report, "Replay ledger data is still backfilling.") {
		t.Fatalf("unexpected shared view partial report")
	}

	similar := NewRunRecord(domain.Task{ID: "OPE-100-browser-b", Source: "linear", Title: "Browser replay failure"}, "run-browser-b", "browser")
	similar.Trace("runtime.execute", "failed")
	similar.Audit("runtime.execute", "worker", "failed", map[string]any{"reason": "browser session crashed"})
	similar.Finalize("failed", "browser session crashed")
	security := NewRunRecord(domain.Task{ID: "OPE-100-security", Source: "linear", Title: "Security approval"}, "run-security", "vm")
	security.Trace("scheduler.decide", "pending")
	security.Audit("scheduler.decision", "scheduler", "pending", map[string]any{"reason": "requires approval for high-risk task"})
	security.Finalize("needs-approval", "requires approval for high-risk task")
	feedback := []TriageFeedbackRecord{
		NewTriageFeedbackRecord("run-browser", "replay run and inspect tool failures", "accepted", "ops-lead", "matched previous recovery path"),
		NewTriageFeedbackRecord("run-security", "request approval and queue security review", "rejected", "sec-reviewer", "approval already in flight"),
	}
	center = BuildAutoTriageCenter([]RunRecord{failed, similar, security}, "Auto Triage Center", "2026-03-11", feedback)
	report = RenderAutoTriageCenterReport(center, 3, nil)
	browserItem := center.Inbox[0]
	if !reflect.DeepEqual(center.FeedbackCounts, map[string]int{"accepted": 1, "rejected": 1, "pending": 1}) || browserItem.Suggestions[0].FeedbackStatus != "accepted" || browserItem.Suggestions[0].Evidence[0].RelatedRunID != "run-browser-b" || browserItem.Suggestions[0].Evidence[0].Score < 0.8 || !strings.Contains(report, "## Inbox") || !strings.Contains(report, "similar=run-browser-b:") {
		t.Fatalf("unexpected triage feedback loop")
	}
}

func TestTakeoverQueueAndOrchestration(t *testing.T) {
	entries := []map[string]any{
		{"run_id": "run-sec", "task_id": "OPE-66-sec", "source": "linear", "summary": "requires approval for high-risk task", "audits": []map[string]any{{"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "security", "reason": "requires approval for high-risk task", "required_approvals": []any{"security-review"}}}}},
		{"run_id": "run-ops", "task_id": "OPE-66-ops", "source": "linear", "summary": "premium tier required for advanced cross-department orchestration", "audits": []map[string]any{{"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "operations", "reason": "premium tier required for advanced cross-department orchestration", "required_approvals": []any{"ops-manager"}}}}},
	}
	queue := BuildTakeoverQueueFromLedger(entries, "Cross-Team Takeovers", "2026-03-10")
	report := RenderTakeoverQueueReport(queue, 3, nil)
	if queue.PendingRequests != 2 || !reflect.DeepEqual(queue.TeamCounts, map[string]int{"operations": 1, "security": 1}) || queue.ApprovalCount != 2 || queue.Recommendation != "expedite-security-review" || queue.Requests[0].RunID != "run-ops" || queue.Requests[0].Actions[3].Enabled != true || queue.Requests[1].Actions[3].Enabled != false || !strings.Contains(report, "Pending Requests: 2") || !strings.Contains(report, "run-sec: team=security status=pending task=OPE-66-sec approvals=security-review") || !strings.Contains(report, "Escalate [escalate] state=disabled target=run-sec reason=security takeovers are already escalated") {
		t.Fatalf("unexpected takeover queue")
	}
	report = RenderTakeoverQueueReport(BuildTakeoverQueueFromLedger(nil, "Cross-Team Takeovers", "2026-03-10"), 0, makeSharedView(0, false, []string{"Takeover approvals service timed out."}, nil))
	if !strings.Contains(report, "- State: error") || !strings.Contains(report, "- Summary: Unable to load data for the current filters.") || !strings.Contains(report, "## Errors") || !strings.Contains(report, "Takeover approvals service timed out.") {
		t.Fatalf("unexpected takeover error state")
	}

	run := NewRunRecord(domain.Task{ID: "OPE-66-canvas", Source: "linear", Title: "Canvas run"}, "run-canvas", "browser")
	run.Audit("tool.invoke", "worker", "success", map[string]any{"tool": "browser"})
	canvas := BuildOrchestrationCanvas(run, OrchestrationPlan{TaskID: "OPE-66-canvas", CollaborationMode: "tier-limited", Departments: []string{"operations", "engineering"}}, OrchestrationPolicyDecision{Tier: "standard", UpgradeRequired: true, Reason: "premium tier required for advanced cross-department orchestration", BlockedDepartments: []string{"customer-success"}, EntitlementStatus: "upgrade-required", BillingModel: "standard-blocked", EstimatedCostUSD: 7.0, IncludedUsageUnits: 2, OverageUsageUnits: 1, OverageCostUSD: 4.0}, &HandoffRequest{TargetTeam: "operations", Reason: "premium tier required for advanced cross-department orchestration", RequiredApprovals: []string{"ops-manager"}})
	report = RenderOrchestrationCanvas(canvas)
	if canvas.Recommendation != "resolve-entitlement-gap" || !reflect.DeepEqual(canvas.ActiveTools, []string{"browser"}) || !canvas.Actions[3].Enabled || canvas.Actions[4].Enabled || !strings.Contains(report, "# Orchestration Canvas") || !strings.Contains(report, "- Tier: standard") || !strings.Contains(report, "Escalate [escalate] state=enabled target=run-canvas") {
		t.Fatalf("unexpected orchestration canvas")
	}

	ledgerCanvas := BuildOrchestrationCanvasFromLedgerEntry(map[string]any{"run_id": "run-flow-1", "task_id": "OPE-113", "audits": []map[string]any{
		{"action": "orchestration.plan", "actor": "scheduler", "outcome": "enabled", "timestamp": "2026-03-11T11:00:00Z", "details": map[string]any{"collaboration_mode": "cross-functional", "departments": []any{"operations", "engineering"}, "approvals": []any{}}},
		{"action": "orchestration.policy", "actor": "scheduler", "outcome": "enabled", "timestamp": "2026-03-11T11:01:00Z", "details": map[string]any{"tier": "premium", "entitlement_status": "included", "billing_model": "premium-included"}},
		{"action": "collaboration.comment", "actor": "ops-lead", "outcome": "recorded", "timestamp": "2026-03-11T11:02:00Z", "details": map[string]any{"surface": "flow", "comment_id": "flow-comment-1", "body": "Route @eng once the dashboard note is resolved.", "mentions": []any{"eng"}, "anchor": "handoff-lane", "status": "open"}},
		{"action": "collaboration.decision", "actor": "eng-manager", "outcome": "accepted", "timestamp": "2026-03-11T11:03:00Z", "details": map[string]any{"surface": "flow", "decision_id": "flow-decision-1", "summary": "Engineering owns the next flow handoff."}},
	}})
	report = RenderOrchestrationCanvas(ledgerCanvas)
	if ledgerCanvas.Collaboration == nil || ledgerCanvas.Recommendation != "resolve-flow-comments" || !strings.Contains(report, "## Collaboration") || !strings.Contains(report, "Route @eng once the dashboard note is resolved.") || !strings.Contains(report, "Engineering owns the next flow handoff.") {
		t.Fatalf("unexpected ledger canvas")
	}
}

func TestOrchestrationPortfolioAndBilling(t *testing.T) {
	canvases := []OrchestrationCanvas{
		{TaskID: "OPE-66-a", RunID: "run-a", CollaborationMode: "cross-functional", Departments: []string{"operations", "engineering", "security"}, Tier: "premium", EntitlementStatus: "included", BillingModel: "premium-included", EstimatedCostUSD: 4.5, IncludedUsageUnits: 3, HandoffTeam: "security", HandoffStatus: "pending"},
		{TaskID: "OPE-66-b", RunID: "run-b", CollaborationMode: "tier-limited", Departments: []string{"operations", "engineering"}, Tier: "standard", UpgradeRequired: true, EntitlementStatus: "upgrade-required", BillingModel: "standard-blocked", EstimatedCostUSD: 7.0, IncludedUsageUnits: 2, OverageUsageUnits: 1, OverageCostUSD: 4.0, BlockedDepartments: []string{"customer-success"}, HandoffTeam: "operations", HandoffStatus: "pending"},
	}
	queue := BuildTakeoverQueueFromLedger([]map[string]any{
		{"run_id": "run-a", "task_id": "OPE-66-a", "source": "linear", "audits": []map[string]any{{"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "security", "reason": "risk", "required_approvals": []any{"security-review"}}}}},
		{"run_id": "run-b", "task_id": "OPE-66-b", "source": "linear", "audits": []map[string]any{{"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "operations", "reason": "entitlement", "required_approvals": []any{"ops-manager"}}}}},
	}, "Cross-Team Takeovers", "2026-03-10")
	portfolio := BuildOrchestrationPortfolio(canvases, "Cross-Team Portfolio", "2026-03-10", &queue)
	report := RenderOrchestrationPortfolioReport(portfolio, nil)
	if portfolio.TotalRuns != 2 || !reflect.DeepEqual(portfolio.CollaborationModes, map[string]int{"cross-functional": 1, "tier-limited": 1}) || !reflect.DeepEqual(portfolio.TierCounts, map[string]int{"premium": 1, "standard": 1}) || !reflect.DeepEqual(portfolio.EntitlementCounts, map[string]int{"included": 1, "upgrade-required": 1}) || !reflect.DeepEqual(portfolio.BillingModelCounts, map[string]int{"premium-included": 1, "standard-blocked": 1}) || portfolio.TotalEstimatedCostUSD != 11.5 || portfolio.TotalOverageCostUSD != 4.0 || portfolio.UpgradeRequiredCount != 1 || portfolio.ActiveHandoffs != 2 || portfolio.Recommendation != "stabilize-security-takeovers" || !strings.Contains(report, "- Takeover Queue: pending=2 recommendation=expedite-security-review") || !strings.Contains(report, "- run-a: mode=cross-functional tier=premium entitlement=included billing=premium-included estimated_cost_usd=4.50 overage_cost_usd=0.00 upgrade_required=false handoff=security") {
		t.Fatalf("unexpected orchestration portfolio")
	}
	report = RenderOrchestrationPortfolioReport(BuildOrchestrationPortfolio(nil, "Cross-Team Portfolio", "2026-03-10", nil), makeSharedView(0, false, nil, nil))
	if !strings.Contains(report, "- State: empty") || !strings.Contains(report, "- Summary: No records match the current filters.") || !strings.Contains(report, "## Filters") {
		t.Fatalf("unexpected portfolio empty state")
	}

	page := RenderOrchestrationOverviewPage(OrchestrationPortfolio{Name: "Cross-Team Portfolio", TakeoverQueue: &TakeoverQueue{PendingRequests: 1, Recommendation: "expedite-security-review"}, BillingModelCounts: map[string]int{"premium-included": 1}})
	if !strings.Contains(page, "<title>Orchestration Overview") || !strings.Contains(page, "Cross-Team Portfolio") || !strings.Contains(page, "review-security-takeover") || !strings.Contains(page, "Estimated Cost") || !strings.Contains(page, "premium-included") || !strings.Contains(page, "pending=1 recommendation=expedite-security-review") || !strings.Contains(page, "run-a") {
		t.Fatalf("unexpected orchestration overview page")
	}

	portfolioEntries := []map[string]any{
		{"run_id": "run-a", "task_id": "OPE-66-a", "audits": []map[string]any{{"action": "orchestration.plan", "outcome": "ready", "details": map[string]any{"collaboration_mode": "cross-functional", "departments": []any{"operations", "engineering", "security"}, "approvals": []any{"security-review"}}}, {"action": "orchestration.policy", "outcome": "enabled", "details": map[string]any{"tier": "premium", "entitlement_status": "included", "billing_model": "premium-included", "estimated_cost_usd": 4.5, "included_usage_units": 3, "blocked_departments": []any{}}}, {"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "security", "reason": "approval required", "required_approvals": []any{"security-review"}}}}},
		{"run_id": "run-b", "task_id": "OPE-66-b", "audits": []map[string]any{{"action": "orchestration.plan", "outcome": "ready", "details": map[string]any{"collaboration_mode": "tier-limited", "departments": []any{"operations", "engineering"}, "approvals": []any{}}}, {"action": "orchestration.policy", "outcome": "upgrade-required", "details": map[string]any{"tier": "standard", "entitlement_status": "upgrade-required", "billing_model": "standard-blocked", "estimated_cost_usd": 7.0, "included_usage_units": 2, "overage_usage_units": 1, "overage_cost_usd": 4.0, "blocked_departments": []any{"customer-success"}}}, {"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "operations", "reason": "entitlement gap", "required_approvals": []any{"ops-manager"}}}}},
	}
	portfolio = BuildOrchestrationPortfolioFromLedger(portfolioEntries, "Ledger Portfolio", "2026-03-10")
	if portfolio.TotalRuns != 2 || portfolio.TakeoverQueue == nil || portfolio.TakeoverQueue.PendingRequests != 2 || portfolio.Recommendation != "stabilize-security-takeovers" {
		t.Fatalf("unexpected ledger portfolio")
	}

	billingEntries := []map[string]any{
		{"run_id": "run-ledger-a", "task_id": "OPE-104-a", "audits": []map[string]any{{"action": "orchestration.plan", "outcome": "ready", "details": map[string]any{"collaboration_mode": "cross-functional", "departments": []any{"operations", "engineering", "security"}, "approvals": []any{"security-review"}}}, {"action": "orchestration.policy", "outcome": "enabled", "details": map[string]any{"tier": "premium", "entitlement_status": "included", "billing_model": "premium-included", "estimated_cost_usd": 4.5, "included_usage_units": 3, "blocked_departments": []any{}}}}},
		{"run_id": "run-ledger-b", "task_id": "OPE-104-b", "audits": []map[string]any{{"action": "orchestration.plan", "outcome": "ready", "details": map[string]any{"collaboration_mode": "tier-limited", "departments": []any{"operations", "engineering"}, "approvals": []any{}}}, {"action": "orchestration.policy", "outcome": "upgrade-required", "details": map[string]any{"tier": "standard", "entitlement_status": "upgrade-required", "billing_model": "standard-blocked", "estimated_cost_usd": 7.0, "included_usage_units": 2, "overage_usage_units": 1, "overage_cost_usd": 4.0, "blocked_departments": []any{"customer-success"}}}, {"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "operations", "reason": "entitlement gap", "required_approvals": []any{"ops-manager"}}}}},
	}
	billing := BuildBillingEntitlementsPage(OrchestrationPortfolio{Name: "Revenue Ops", Period: "2026-03", Canvases: []OrchestrationCanvas{{TaskID: "OPE-104-a", RunID: "run-billing-a", CollaborationMode: "cross-functional", Departments: []string{"operations", "engineering", "security"}, Tier: "premium", EntitlementStatus: "included", BillingModel: "premium-included", EstimatedCostUSD: 4.5, IncludedUsageUnits: 3, HandoffTeam: "security"}, {TaskID: "OPE-104-b", RunID: "run-billing-b", CollaborationMode: "tier-limited", Departments: []string{"operations", "engineering"}, Tier: "standard", UpgradeRequired: true, EntitlementStatus: "upgrade-required", BillingModel: "standard-blocked", EstimatedCostUSD: 7.0, IncludedUsageUnits: 2, OverageUsageUnits: 1, OverageCostUSD: 4.0, BlockedDepartments: []string{"customer-success"}, HandoffTeam: "operations"}}}, "OpenAGI Revenue Cloud", "Standard", "2026-03")
	report = RenderBillingEntitlementsReport(billing)
	if billing.RunCount != 2 || billing.TotalIncludedUsageUnits != 5 || billing.TotalOverageUsageUnits != 1 || billing.TotalEstimatedCostUSD != 11.5 || billing.TotalOverageCostUSD != 4.0 || billing.UpgradeRequiredCount != 1 || !reflect.DeepEqual(billing.EntitlementCounts, map[string]int{"included": 1, "upgrade-required": 1}) || !reflect.DeepEqual(billing.BillingModelCounts, map[string]int{"premium-included": 1, "standard-blocked": 1}) || !reflect.DeepEqual(billing.BlockedCapabilities, []string{"customer-success"}) || billing.Recommendation != "resolve-plan-gaps" || !strings.Contains(report, "# Billing & Entitlements Report") || !strings.Contains(report, "- Workspace: OpenAGI Revenue Cloud") || !strings.Contains(report, "- Overage Cost (USD): 4.00") || !strings.Contains(report, "- run-billing-b: task=OPE-104-b entitlement=upgrade-required billing=standard-blocked") {
		t.Fatalf("unexpected billing report")
	}
	page = RenderBillingEntitlementsPage(BillingEntitlementsPage{WorkspaceName: "OpenAGI Revenue Cloud", PlanName: "Premium", BillingPeriod: "2026-03", Charges: []BillingRunCharge{{RunID: "run-billing-a", TaskID: "OPE-104-a", EntitlementStatus: "included", BillingModel: "premium-included", EstimatedCostUSD: 4.5, IncludedUsageUnits: 3, Recommendation: "review-security-takeover"}}})
	if !strings.Contains(page, "<title>Billing & Entitlements") || !strings.Contains(page, "OpenAGI Revenue Cloud") || !strings.Contains(page, "Premium plan for 2026-03") || !strings.Contains(page, "Charge Feed") || !strings.Contains(page, "premium-included") {
		t.Fatalf("unexpected billing html")
	}
	billing = BuildBillingEntitlementsPageFromLedger(billingEntries, "OpenAGI Revenue Cloud", "Standard", "2026-03")
	if billing.RunCount != 2 || billing.Recommendation != "resolve-plan-gaps" || billing.TotalOverageCostUSD != 4.0 || !reflect.DeepEqual(billing.Charges[1].BlockedCapabilities, []string{"customer-success"}) || billing.Charges[1].HandoffTeam != "operations" {
		t.Fatalf("unexpected billing from ledger")
	}
}

func TestTimestampsAreUTC(t *testing.T) {
	record := NewTriageFeedbackRecord("run-1", "classify", "accepted", "ops", "")
	if !strings.HasSuffix(record.Timestamp, "Z") {
		t.Fatalf("timestamp missing Z: %s", record.Timestamp)
	}
	parsed, err := time.Parse(time.RFC3339, record.Timestamp)
	if err != nil || parsed.Location() != time.UTC {
		t.Fatalf("unexpected triage timestamp: %v %v", parsed, err)
	}
	content := RenderIssueValidationReport("BIG-900", "v1", "repo", "pass")
	var timestampValue string
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "- 生成时间:") {
			timestampValue = strings.TrimSpace(strings.TrimPrefix(line, "- 生成时间:"))
		}
	}
	if !strings.HasSuffix(timestampValue, "Z") {
		t.Fatalf("validation timestamp missing Z: %s", timestampValue)
	}
	parsed, err = time.Parse(time.RFC3339, timestampValue)
	if err != nil || parsed.Location() != time.UTC {
		t.Fatalf("unexpected validation timestamp: %v %v", parsed, err)
	}
}
