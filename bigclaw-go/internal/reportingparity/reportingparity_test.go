package reportingparity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func boolPtr(v bool) *bool {
	return &v
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
	if !strings.Contains(string(data), "BIG-101") || !strings.Contains(string(data), "pass") {
		t.Fatalf("unexpected report content: %s", string(data))
	}
}

func TestConsoleActionStateReflectsEnabledFlag(t *testing.T) {
	enabled := NewConsoleAction("retry", "Retry", "run-1")
	disabled := ConsoleAction{Key: "pause", Label: "Pause", Target: "run-1", Enabled: false, Reason: "already completed"}
	if enabled.State() != "enabled" || disabled.State() != "disabled" {
		t.Fatalf("unexpected states: enabled=%s disabled=%s", enabled.State(), disabled.State())
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
			{Heading: "What changed", Body: "Approval queue depth fell from 5 to 1 after moving browser-heavy runs onto the shared operations lane.", Evidence: []string{"queue-control-center", "weekly-operations"}, Callouts: []string{"SLA risk contained", "No new regressions opened"}},
			{Heading: "What needs attention", Body: "Security takeover requests still cluster around data-export tasks and need a dedicated reviewer window.", Evidence: []string{"takeover-queue"}, Callouts: []string{"Review staffing before Friday close"}},
		},
		ActionItems:   []string{"Publish the markdown export to leadership", "Review security handoff staffing"},
		SourceReports: []string{"reports/weekly-operations.md", "reports/takeover-queue.md"},
	}

	markdown := RenderReportStudioReport(studio)
	plainText := RenderReportStudioPlainText(studio)
	html := RenderReportStudioHTML(studio)
	artifacts, err := WriteReportStudioBundle(filepath.Join(t.TempDir(), "studio"), studio)
	if err != nil {
		t.Fatalf("write studio bundle: %v", err)
	}

	if !studio.Ready() || studio.Recommendation() != "publish" {
		t.Fatalf("expected publish-ready studio, got ready=%t recommendation=%s", studio.Ready(), studio.Recommendation())
	}
	for _, fragment := range []string{"# Report Studio", "### What changed"} {
		if !strings.Contains(markdown, fragment) {
			t.Fatalf("expected %q in markdown, got %s", fragment, markdown)
		}
	}
	if !strings.Contains(plainText, "Recommendation: publish") || !strings.Contains(html, "<h1>Executive Weekly Narrative</h1>") {
		t.Fatalf("unexpected plain/html output")
	}
	if _, err := os.Stat(artifacts.MarkdownPath); err != nil {
		t.Fatalf("missing markdown artifact: %v", err)
	}
	if _, err := os.Stat(artifacts.HTMLPath); err != nil {
		t.Fatalf("missing html artifact: %v", err)
	}
	if _, err := os.Stat(artifacts.TextPath); err != nil {
		t.Fatalf("missing text artifact: %v", err)
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
		t.Fatalf("expected draft studio, got ready=%t recommendation=%s", studio.Ready(), studio.Recommendation())
	}
}

func TestRenderPilotScorecardIncludesROIAndRecommendation(t *testing.T) {
	scorecard := PilotScorecard{
		IssueID:            "OPE-60",
		Customer:           "Design Partner A",
		Period:             "2026-Q2",
		Metrics:            []PilotMetric{{Name: "Automation coverage", Baseline: 35, Current: 82, Target: 80, Unit: "%", HigherIsBetter: true}, {Name: "Manual review time", Baseline: 12, Current: 4, Target: 5, Unit: "h", HigherIsBetter: false}},
		MonthlyBenefit:     12000,
		MonthlyCost:        2500,
		ImplementationCost: 18000,
		BenchmarkScore:     96,
		BenchmarkPassed:    true,
	}
	content := RenderPilotScorecard(scorecard)
	if scorecard.MetricsMet() != 2 || scorecard.Recommendation() != "go" {
		t.Fatalf("unexpected scorecard recommendation state")
	}
	if payback := scorecard.PaybackMonths(); payback == nil || *payback != 1.9 {
		t.Fatalf("unexpected payback months: %+v", payback)
	}
	for _, fragment := range []string{"Annualized ROI: 200.0%", "Recommendation: go", "Benchmark Score: 96", "Automation coverage"} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("expected %q in scorecard output, got %s", fragment, content)
		}
	}
}

func TestPilotScorecardReturnsHoldWhenValueIsNegative(t *testing.T) {
	scorecard := PilotScorecard{
		IssueID:            "OPE-60",
		Customer:           "Design Partner B",
		Period:             "2026-Q2",
		Metrics:            []PilotMetric{{Name: "Backlog aging", Baseline: 5, Current: 7, Target: 4, Unit: "d", HigherIsBetter: false}},
		MonthlyBenefit:     1000,
		MonthlyCost:        3000,
		ImplementationCost: 12000,
		BenchmarkPassed:    false,
	}
	if scorecard.MonthlyNetValue() != -2000 || scorecard.PaybackMonths() != nil || scorecard.Recommendation() != "hold" {
		t.Fatalf("unexpected negative scorecard outcome")
	}
}

func TestIssueClosureRequiresNonEmptyValidationReport(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "validation.md")
	decision := EvaluateIssueClosure("BIG-602", reportPath, nil, nil, nil)
	if decision.Allowed || decision.Reason != "validation report required before closing issue" || ValidationReportExists(reportPath) {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestIssueClosureBlocksFailedValidationReport(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "validation.md")
	if err := WriteReport(reportPath, "# Validation\n\nfailed"); err != nil {
		t.Fatalf("write report: %v", err)
	}
	decision := EvaluateIssueClosure("BIG-602", reportPath, boolPtr(false), nil, nil)
	if decision.Allowed || decision.Reason != "validation failed; issue must remain open" || !ValidationReportExists(reportPath) {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestIssueClosureAllowsCompletedValidationReport(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "validation.md")
	if err := WriteReport(reportPath, RenderIssueValidationReport("BIG-602", "v0.1", "sandbox", "pass")); err != nil {
		t.Fatalf("write report: %v", err)
	}
	decision := EvaluateIssueClosure("BIG-602", reportPath, boolPtr(true), nil, nil)
	if !decision.Allowed || decision.Reason != "validation report and launch checklist requirements satisfied; issue can be closed" || decision.ReportPath != reportPath {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestLaunchChecklistAutoLinksDocumentationStatus(t *testing.T) {
	root := t.TempDir()
	runbook := filepath.Join(root, "runbook.md")
	faq := filepath.Join(root, "faq.md")
	if err := WriteReport(runbook, "# Runbook\n\nready"); err != nil {
		t.Fatalf("write runbook: %v", err)
	}
	checklist := BuildLaunchChecklist(
		"BIG-1003",
		[]DocumentationArtifact{{Name: "runbook", Path: runbook}, {Name: "faq", Path: faq}},
		[]LaunchChecklistItem{{Name: "Operations handoff", Evidence: []string{"runbook"}}, {Name: "Support handoff", Evidence: []string{"faq"}}},
	)
	report := RenderLaunchChecklistReport(checklist)
	if checklist.DocumentationStatus["runbook"] != true || checklist.DocumentationStatus["faq"] != false || checklist.CompletedItems != 1 || len(checklist.MissingDocumentation) != 1 || checklist.MissingDocumentation[0] != "faq" || checklist.Ready {
		t.Fatalf("unexpected checklist: %+v", checklist)
	}
	for _, fragment := range []string{"runbook: available=true", "faq: available=false", "Support handoff: completed=false evidence=faq"} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in launch checklist report, got %s", fragment, report)
		}
	}
}

func TestFinalDeliveryChecklistTracksRequiredOutputsAndRecommendedDocs(t *testing.T) {
	root := t.TempDir()
	validationBundle := filepath.Join(root, "validation-bundle.md")
	releaseNotes := filepath.Join(root, "release-notes.md")
	if err := WriteReport(validationBundle, "# Validation Bundle\n\nready"); err != nil {
		t.Fatalf("write validation bundle: %v", err)
	}
	checklist := BuildFinalDeliveryChecklist(
		"BIG-4702",
		[]DocumentationArtifact{{Name: "validation-bundle", Path: validationBundle}, {Name: "release-notes", Path: releaseNotes}},
		[]DocumentationArtifact{{Name: "runbook", Path: filepath.Join(root, "runbook.md")}, {Name: "faq", Path: filepath.Join(root, "faq.md")}},
	)
	report := RenderFinalDeliveryChecklistReport(checklist)
	if checklist.RequiredOutputStatus["validation-bundle"] != true || checklist.RequiredOutputStatus["release-notes"] != false || checklist.GeneratedRequiredOutputs != 1 || checklist.GeneratedRecommendedDocs != 0 || len(checklist.MissingRequiredOutputs) != 1 || checklist.MissingRequiredOutputs[0] != "release-notes" || checklist.Ready {
		t.Fatalf("unexpected final delivery checklist: %+v", checklist)
	}
	for _, fragment := range []string{"Required Outputs Generated: 1/2", "Recommended Docs Generated: 0/2", "release-notes: available=false", "runbook: available=false"} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in final delivery report, got %s", fragment, report)
		}
	}
}

func TestIssueClosureBlocksIncompleteLinkedLaunchChecklist(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, "validation.md")
	runbook := filepath.Join(root, "runbook.md")
	if err := WriteReport(reportPath, RenderIssueValidationReport("BIG-1003", "v0.2", "staging", "pass")); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := WriteReport(runbook, "# Runbook\n\nready"); err != nil {
		t.Fatalf("write runbook: %v", err)
	}
	checklist := BuildLaunchChecklist(
		"BIG-1003",
		[]DocumentationArtifact{{Name: "runbook", Path: runbook}, {Name: "launch-faq", Path: filepath.Join(root, "launch-faq.md")}},
		[]LaunchChecklistItem{{Name: "Launch comms", Evidence: []string{"runbook", "launch-faq"}}},
	)
	decision := EvaluateIssueClosure("BIG-1003", reportPath, boolPtr(true), &checklist, nil)
	if decision.Allowed || decision.Reason != "launch checklist incomplete; linked documentation missing or empty" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestIssueClosureBlocksMissingRequiredFinalDeliveryOutputs(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, "validation.md")
	if err := WriteReport(reportPath, RenderIssueValidationReport("BIG-4702", "v0.3", "staging", "pass")); err != nil {
		t.Fatalf("write report: %v", err)
	}
	checklist := BuildFinalDeliveryChecklist(
		"BIG-4702",
		[]DocumentationArtifact{{Name: "validation-bundle", Path: filepath.Join(root, "validation-bundle.md")}},
		[]DocumentationArtifact{{Name: "runbook", Path: filepath.Join(root, "runbook.md")}},
	)
	decision := EvaluateIssueClosure("BIG-4702", reportPath, boolPtr(true), nil, &checklist)
	if decision.Allowed || decision.Reason != "final delivery checklist incomplete; required outputs missing" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestIssueClosureAllowsWhenRequiredFinalDeliveryOutputsExist(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, "validation.md")
	validationBundle := filepath.Join(root, "validation-bundle.md")
	releaseNotes := filepath.Join(root, "release-notes.md")
	if err := WriteReport(reportPath, RenderIssueValidationReport("BIG-4702", "v0.3", "staging", "pass")); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := WriteReport(validationBundle, "# Validation Bundle\n\nready"); err != nil {
		t.Fatalf("write validation bundle: %v", err)
	}
	if err := WriteReport(releaseNotes, "# Release Notes\n\nready"); err != nil {
		t.Fatalf("write release notes: %v", err)
	}
	checklist := BuildFinalDeliveryChecklist(
		"BIG-4702",
		[]DocumentationArtifact{{Name: "validation-bundle", Path: validationBundle}, {Name: "release-notes", Path: releaseNotes}},
		[]DocumentationArtifact{{Name: "runbook", Path: filepath.Join(root, "runbook.md")}},
	)
	decision := EvaluateIssueClosure("BIG-4702", reportPath, boolPtr(true), nil, &checklist)
	if !decision.Allowed || decision.Reason != "validation report and final delivery checklist requirements satisfied; issue can be closed" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestIssueClosureAllowsWhenLinkedLaunchChecklistIsReady(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, "validation.md")
	runbook := filepath.Join(root, "runbook.md")
	faq := filepath.Join(root, "launch-faq.md")
	if err := WriteReport(reportPath, RenderIssueValidationReport("BIG-1003", "v0.2", "staging", "pass")); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := WriteReport(runbook, "# Runbook\n\nready"); err != nil {
		t.Fatalf("write runbook: %v", err)
	}
	if err := WriteReport(faq, "# FAQ\n\nready"); err != nil {
		t.Fatalf("write faq: %v", err)
	}
	checklist := BuildLaunchChecklist(
		"BIG-1003",
		[]DocumentationArtifact{{Name: "runbook", Path: runbook}, {Name: "launch-faq", Path: faq}},
		[]LaunchChecklistItem{{Name: "Launch comms", Evidence: []string{"runbook", "launch-faq"}}},
	)
	decision := EvaluateIssueClosure("BIG-1003", reportPath, boolPtr(true), &checklist, nil)
	if !decision.Allowed || decision.Reason != "validation report and launch checklist requirements satisfied; issue can be closed" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestRenderPilotPortfolioReportSummarizesCommercialReadiness(t *testing.T) {
	portfolio := PilotPortfolio{
		Name:   "Design Partners",
		Period: "2026-H1",
		Scorecards: []PilotScorecard{
			{IssueID: "OPE-60", Customer: "Partner A", Period: "2026-Q2", Metrics: []PilotMetric{{Name: "Coverage", Baseline: 40, Current: 85, Target: 80, Unit: "%", HigherIsBetter: true}}, MonthlyBenefit: 15000, MonthlyCost: 3000, ImplementationCost: 18000, BenchmarkScore: 97, BenchmarkPassed: true},
			{IssueID: "OPE-61", Customer: "Partner B", Period: "2026-Q2", Metrics: []PilotMetric{{Name: "Cycle time", Baseline: 12, Current: 7, Target: 5, Unit: "h", HigherIsBetter: false}}, MonthlyBenefit: 9000, MonthlyCost: 2500, ImplementationCost: 12000, BenchmarkScore: 88, BenchmarkPassed: true},
		},
	}
	content := RenderPilotPortfolioReport(portfolio)
	if portfolio.TotalMonthlyNetValue() != 18500 || portfolio.AverageROI() != 195.2 {
		t.Fatalf("unexpected portfolio totals")
	}
	counts := portfolio.RecommendationCounts()
	if counts["go"] != 1 || counts["iterate"] != 1 || counts["hold"] != 0 || portfolio.Recommendation() != "continue" {
		t.Fatalf("unexpected recommendation counts: %+v recommendation=%s", counts, portfolio.Recommendation())
	}
	for _, fragment := range []string{"Recommendation Mix: go=1 iterate=1 hold=0", "Partner A: recommendation=go", "Partner B: recommendation=iterate"} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("expected %q in portfolio output, got %s", fragment, content)
		}
	}
}

func TestRenderSharedViewContextIncludesCollaborationAnnotations(t *testing.T) {
	view := SharedViewContext{
		Filters:     []SharedViewFilter{{Label: "Team", Value: "ops"}},
		ResultCount: 4,
		Collaboration: func() *CollaborationThread {
			thread := BuildCollaborationThread(
				"dashboard",
				"ops-overview",
				[]CollaborationComment{{CommentID: "dashboard-comment-1", Author: "pm", Body: "Please review blocker copy with @ops and @eng.", Mentions: []string{"ops", "eng"}, Anchor: "blockers"}},
				[]DecisionNote{{DecisionID: "dashboard-decision-1", Author: "ops", Outcome: "approved", Summary: "Keep the blocker module visible for managers.", Mentions: []string{"pm"}, FollowUp: "Recheck after next data refresh."}},
			)
			return &thread
		}(),
	}
	content := strings.Join(RenderSharedViewContext(view), "\n")
	for _, fragment := range []string{"## Collaboration", "Surface: dashboard", "Please review blocker copy with @ops and @eng.", "Keep the blocker module visible for managers."} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("expected %q in shared view context, got %s", fragment, content)
		}
	}
}

func TestAutoTriageCenterPrioritizesFailedAndPendingRuns(t *testing.T) {
	center := BuildAutoTriageCenter(
		[]TriageRun{
			{RunID: "run-ok", TaskID: "OPE-76-ok", Status: "approved", Medium: "docker", Reason: "default low risk path"},
			{RunID: "run-risk", TaskID: "OPE-76-risk", Status: "needs-approval", Medium: "vm", Reason: "requires approval for high-risk task"},
			{RunID: "run-browser", TaskID: "OPE-76-browser", Status: "failed", Medium: "browser", Reason: "browser session crashed"},
		},
		"Engineering Ops",
		"2026-03-10",
		nil,
	)
	report := RenderAutoTriageCenterReport(center, 3, nil)
	if center.FlaggedRuns != 2 || center.InboxSize != 2 || center.SeverityCounts["critical"] != 1 || center.SeverityCounts["high"] != 1 {
		t.Fatalf("unexpected center counts: %+v", center)
	}
	if center.OwnerCounts["security"] != 1 || center.OwnerCounts["engineering"] != 1 || center.Recommendation != "immediate-attention" {
		t.Fatalf("unexpected owner/recommendation: %+v", center)
	}
	if center.Findings[0].RunID != "run-browser" || center.Findings[1].RunID != "run-risk" {
		t.Fatalf("unexpected finding order: %+v", center.Findings)
	}
	if center.Inbox[0].Suggestions[0].Label != "replay candidate" || center.Inbox[0].Suggestions[0].Confidence < 0.55 {
		t.Fatalf("unexpected inbox suggestion: %+v", center.Inbox[0])
	}
	for _, fragment := range []string{"Flagged Runs: 2", "Inbox Size: 2", "Severity Mix: critical=1 high=1 medium=0", "Feedback Loop: accepted=0 rejected=0 pending=2", "run-browser: severity=critical owner=engineering status=failed", "run-risk: severity=high owner=security status=needs-approval", "actions=Drill Down [drill-down]", "Retry [retry] state=disabled target=run-risk reason=retry available after owner review"} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}

func TestAutoTriageCenterReportRendersSharedViewPartialState(t *testing.T) {
	center := BuildAutoTriageCenter([]TriageRun{{RunID: "run-risk", TaskID: "OPE-94-risk", Status: "needs-approval", Medium: "vm", Reason: "requires approval for high-risk task"}}, "Engineering Ops", "2026-03-10", nil)
	report := RenderAutoTriageCenterReport(center, 1, &SharedViewContext{Filters: []SharedViewFilter{{Label: "Team", Value: "engineering"}}, PartialData: []string{"Replay ledger data is still backfilling."}})
	for _, fragment := range []string{"## View State", "- State: partial-data", "- Team: engineering", "## Partial Data", "Replay ledger data is still backfilling."} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}

func TestAutoTriageCenterBuildsSimilarityEvidenceAndFeedbackLoop(t *testing.T) {
	feedback := []TriageFeedbackRecord{
		NewTriageFeedbackRecord("run-browser-a", "replay run and inspect tool failures", "accepted", "ops-lead", "matched previous recovery path"),
		NewTriageFeedbackRecord("run-security", "request approval and queue security review", "rejected", "sec-reviewer", "approval already in flight"),
	}
	center := BuildAutoTriageCenter(
		[]TriageRun{
			{RunID: "run-browser-a", TaskID: "OPE-100-browser-a", Status: "failed", Medium: "browser", Reason: "browser session crashed"},
			{RunID: "run-browser-b", TaskID: "OPE-100-browser-b", Status: "failed", Medium: "browser", Reason: "browser session crashed"},
			{RunID: "run-security", TaskID: "OPE-100-security", Status: "needs-approval", Medium: "vm", Reason: "requires approval for high-risk task"},
		},
		"Auto Triage Center",
		"2026-03-11",
		feedback,
	)
	report := RenderAutoTriageCenterReport(center, 3, nil)
	if center.FeedbackCounts["accepted"] != 1 || center.FeedbackCounts["rejected"] != 1 || center.FeedbackCounts["pending"] != 1 {
		t.Fatalf("unexpected feedback counts: %+v", center.FeedbackCounts)
	}
	if center.Inbox[0].Suggestions[0].FeedbackStatus != "accepted" || len(center.Inbox[0].Suggestions[0].Evidence) == 0 || center.Inbox[0].Suggestions[0].Evidence[0].RelatedRunID != "run-browser-b" || center.Inbox[0].Suggestions[0].Evidence[0].Score < 0.8 {
		t.Fatalf("unexpected similarity evidence: %+v", center.Inbox[0])
	}
	for _, fragment := range []string{"## Inbox", "run-browser-a: severity=critical owner=engineering status=failed", "similar=run-browser-b:", "Feedback Loop: accepted=1 rejected=1 pending=1"} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}

func TestTakeoverQueueFromLedgerGroupsPendingHandoffs(t *testing.T) {
	queue := BuildTakeoverQueueFromLedger([]map[string]any{
		{"run_id": "run-sec", "task_id": "OPE-66-sec", "audits": []any{map[string]any{"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "security", "reason": "requires approval for high-risk task", "required_approvals": []any{"security-review"}}}}},
		{"run_id": "run-ops", "task_id": "OPE-66-ops", "audits": []any{map[string]any{"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "operations", "reason": "premium tier required for advanced cross-department orchestration", "required_approvals": []any{"ops-manager"}}}}},
		{"run_id": "run-ok", "task_id": "OPE-66-ok", "audits": []any{map[string]any{"action": "scheduler.decision", "outcome": "approved", "details": map[string]any{"reason": "default low risk path"}}}},
	}, "Cross-Team Takeovers", "2026-03-10")
	report := RenderTakeoverQueueReport(queue, 3, nil)
	if queue.PendingRequests != 2 || queue.TeamCounts["operations"] != 1 || queue.TeamCounts["security"] != 1 || queue.ApprovalCount != 2 || queue.Recommendation != "expedite-security-review" {
		t.Fatalf("unexpected queue: %+v", queue)
	}
	if queue.Requests[0].RunID != "run-ops" || queue.Requests[1].RunID != "run-sec" || !queue.Requests[0].Actions[3].Enabled || queue.Requests[1].Actions[3].Enabled {
		t.Fatalf("unexpected requests: %+v", queue.Requests)
	}
	for _, fragment := range []string{"Pending Requests: 2", "Team Mix: operations=1 security=1", "run-sec: team=security status=pending task=OPE-66-sec approvals=security-review", "run-ops: team=operations status=pending task=OPE-66-ops approvals=ops-manager", "Escalate [escalate] state=disabled target=run-sec reason=security takeovers are already escalated"} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}

func TestTakeoverQueueReportRendersSharedViewErrorState(t *testing.T) {
	queue := BuildTakeoverQueueFromLedger(nil, "Cross-Team Takeovers", "2026-03-10")
	report := RenderTakeoverQueueReport(queue, 0, &SharedViewContext{Errors: []string{"Takeover approvals service timed out."}})
	for _, fragment := range []string{"- State: error", "- Summary: Unable to load data for the current filters.", "## Errors", "Takeover approvals service timed out."} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}

func TestOrchestrationCanvasSummarizesPolicyAndHandoff(t *testing.T) {
	canvas := BuildOrchestrationCanvas("run-canvas", "OPE-66-canvas", "standard", true, "premium tier required for advanced cross-department orchestration", "upgrade-required", "standard-blocked", 7.0, 2, 1, 4.0, []string{"customer-success"}, "operations", []string{"ops-manager"}, []string{"browser"})
	report := RenderOrchestrationCanvas(canvas)
	if canvas.Recommendation != "resolve-entitlement-gap" || len(canvas.ActiveTools) != 1 || canvas.ActiveTools[0] != "browser" || !canvas.Actions[3].Enabled || canvas.Actions[4].Enabled {
		t.Fatalf("unexpected canvas: %+v", canvas)
	}
	for _, fragment := range []string{"# Orchestration Canvas", "- Tier: standard", "- Entitlement Status: upgrade-required", "- Billing Model: standard-blocked", "- Estimated Cost (USD): 7.00", "- Handoff Team: operations", "- Recommendation: resolve-entitlement-gap", "## Actions", "Escalate [escalate] state=enabled target=run-canvas"} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}

func TestOrchestrationCanvasReconstructsFlowCollaborationFromLedger(t *testing.T) {
	canvas := BuildOrchestrationCanvasFromLedgerEntry(map[string]any{
		"run_id":  "run-flow-1",
		"task_id": "OPE-113",
		"audits": []any{
			map[string]any{"action": "orchestration.plan", "outcome": "enabled", "details": map[string]any{"collaboration_mode": "cross-functional", "departments": []any{"operations", "engineering"}, "approvals": []any{}}},
			map[string]any{"action": "orchestration.policy", "outcome": "enabled", "details": map[string]any{"tier": "premium", "entitlement_status": "included", "billing_model": "premium-included"}},
			map[string]any{"action": "collaboration.comment", "outcome": "recorded", "details": map[string]any{"body": "Route @eng once the dashboard note is resolved."}},
			map[string]any{"action": "collaboration.decision", "outcome": "accepted", "details": map[string]any{"summary": "Engineering owns the next flow handoff."}},
		},
	})
	report := RenderOrchestrationCanvas(canvas)
	if canvas.Collaboration == nil || canvas.Recommendation != "resolve-flow-comments" {
		t.Fatalf("unexpected collaboration canvas: %+v", canvas)
	}
	for _, fragment := range []string{"## Collaboration", "Route @eng once the dashboard note is resolved.", "Engineering owns the next flow handoff."} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}

func TestOrchestrationPortfolioRollsUpCanvasAndTakeoverState(t *testing.T) {
	queue := BuildTakeoverQueueFromLedger([]map[string]any{
		{"run_id": "run-a", "task_id": "OPE-66-a", "audits": []any{map[string]any{"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "security", "reason": "risk", "required_approvals": []any{"security-review"}}}}},
		{"run_id": "run-b", "task_id": "OPE-66-b", "audits": []any{map[string]any{"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "operations", "reason": "entitlement", "required_approvals": []any{"ops-manager"}}}}},
	}, "Cross-Team Takeovers", "2026-03-10")
	portfolio := BuildOrchestrationPortfolio([]OrchestrationCanvas{
		{TaskID: "OPE-66-a", RunID: "run-a", CollaborationMode: "cross-functional", Departments: []string{"operations", "engineering", "security"}, Tier: "premium", EntitlementStatus: "included", BillingModel: "premium-included", EstimatedCostUSD: 4.5, IncludedUsageUnits: 3, HandoffTeam: "security", HandoffStatus: "pending", Actions: defaultActions("run-a", true)},
		{TaskID: "OPE-66-b", RunID: "run-b", CollaborationMode: "tier-limited", Departments: []string{"operations", "engineering"}, Tier: "standard", UpgradeRequired: true, EntitlementStatus: "upgrade-required", BillingModel: "standard-blocked", EstimatedCostUSD: 7.0, IncludedUsageUnits: 2, OverageUsageUnits: 1, OverageCostUSD: 4.0, BlockedDepartments: []string{"customer-success"}, HandoffTeam: "operations", HandoffStatus: "pending", Actions: defaultActions("run-b", true)},
	}, "Cross-Team Portfolio", "2026-03-10", &queue)
	report := RenderOrchestrationPortfolioReport(portfolio, nil)
	if portfolio.TotalRuns != 2 || portfolio.CollaborationModes["cross-functional"] != 1 || portfolio.CollaborationModes["tier-limited"] != 1 || portfolio.TierCounts["premium"] != 1 || portfolio.TierCounts["standard"] != 1 || portfolio.EntitlementCounts["included"] != 1 || portfolio.EntitlementCounts["upgrade-required"] != 1 || portfolio.BillingModelCounts["premium-included"] != 1 || portfolio.BillingModelCounts["standard-blocked"] != 1 || portfolio.TotalEstimatedCostUSD != 11.5 || portfolio.TotalOverageCostUSD != 4.0 || portfolio.UpgradeRequiredCount != 1 || portfolio.ActiveHandoffs != 2 || portfolio.Recommendation != "stabilize-security-takeovers" {
		t.Fatalf("unexpected portfolio: %+v", portfolio)
	}
	for _, fragment := range []string{"# Orchestration Portfolio Report", "- Collaboration Mix: cross-functional=1 tier-limited=1", "- Tier Mix: premium=1 standard=1", "- Entitlement Mix: included=1 upgrade-required=1", "- Billing Models: premium-included=1 standard-blocked=1", "- Estimated Cost (USD): 11.50", "- Overage Cost (USD): 4.00", "- Takeover Queue: pending=2 recommendation=expedite-security-review", "- run-a: mode=cross-functional tier=premium entitlement=included billing=premium-included estimated_cost_usd=4.50 overage_cost_usd=0.00 upgrade_required=false handoff=security", "actions=Drill Down [drill-down]"} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}

func TestOrchestrationPortfolioReportRendersSharedViewEmptyState(t *testing.T) {
	portfolio := BuildOrchestrationPortfolio(nil, "Cross-Team Portfolio", "2026-03-10", nil)
	report := RenderOrchestrationPortfolioReport(portfolio, &SharedViewContext{Filters: []SharedViewFilter{{Label: "Team", Value: "engineering"}}})
	for _, fragment := range []string{"- State: empty", "- Summary: No records match the current filters.", "## Filters"} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}

func TestRenderOrchestrationOverviewPage(t *testing.T) {
	queue := BuildTakeoverQueueFromLedger([]map[string]any{{"run_id": "run-a", "task_id": "OPE-66-a", "audits": []any{map[string]any{"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "security", "reason": "risk", "required_approvals": []any{"security-review"}}}}}}, "Cross-Team Takeovers", "2026-03-10")
	portfolio := BuildOrchestrationPortfolio([]OrchestrationCanvas{{TaskID: "OPE-66-a", RunID: "run-a", CollaborationMode: "cross-functional", Departments: []string{"operations", "engineering"}, Tier: "premium", EntitlementStatus: "included", BillingModel: "premium-included", EstimatedCostUSD: 3.0, HandoffTeam: "security"}}, "Cross-Team Portfolio", "2026-03-10", &queue)
	page := RenderOrchestrationOverviewPage(portfolio)
	for _, fragment := range []string{"<title>Orchestration Overview", "Cross-Team Portfolio", "Estimated Cost", "premium-included", "pending=1 recommendation=expedite-security-review", "run-a", "actions=Drill Down [drill-down]"} {
		if !strings.Contains(page, fragment) {
			t.Fatalf("expected %q in page, got %s", fragment, page)
		}
	}
}

func TestBuildOrchestrationCanvasFromLedgerEntryExtractsAuditState(t *testing.T) {
	canvas := BuildOrchestrationCanvasFromLedgerEntry(map[string]any{
		"run_id":  "run-ledger",
		"task_id": "OPE-66-ledger",
		"audits": []any{
			map[string]any{"action": "orchestration.plan", "outcome": "ready", "details": map[string]any{"collaboration_mode": "tier-limited", "departments": []any{"operations", "engineering"}, "approvals": []any{"security-review"}}},
			map[string]any{"action": "orchestration.policy", "outcome": "upgrade-required", "details": map[string]any{"tier": "standard", "entitlement_status": "upgrade-required", "billing_model": "standard-blocked", "estimated_cost_usd": 7.0, "included_usage_units": 2, "overage_usage_units": 1, "overage_cost_usd": 4.0, "blocked_departments": []any{"security", "customer-success"}}},
			map[string]any{"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "operations", "reason": "premium tier required for advanced cross-department orchestration"}},
			map[string]any{"action": "tool.invoke", "outcome": "success", "details": map[string]any{"tool": "browser"}},
		},
	})
	if canvas.RunID != "run-ledger" || canvas.CollaborationMode != "tier-limited" || strings.Join(canvas.Departments, ",") != "operations,engineering" || strings.Join(canvas.RequiredApprovals, ",") != "security-review" || canvas.Tier != "standard" || !canvas.UpgradeRequired || canvas.EntitlementStatus != "upgrade-required" || canvas.BillingModel != "standard-blocked" || canvas.EstimatedCostUSD != 7.0 || canvas.IncludedUsageUnits != 2 || canvas.OverageUsageUnits != 1 || canvas.OverageCostUSD != 4.0 || strings.Join(canvas.BlockedDepartments, ",") != "security,customer-success" || canvas.HandoffTeam != "operations" || strings.Join(canvas.ActiveTools, ",") != "browser" || !canvas.Actions[3].Enabled || canvas.Actions[4].Enabled {
		t.Fatalf("unexpected extracted canvas: %+v", canvas)
	}
}

func TestBuildOrchestrationPortfolioFromLedgerRollsUpEntries(t *testing.T) {
	portfolio := BuildOrchestrationPortfolioFromLedger([]map[string]any{
		{"run_id": "run-a", "task_id": "OPE-66-a", "audits": []any{map[string]any{"action": "orchestration.plan", "outcome": "ready", "details": map[string]any{"collaboration_mode": "cross-functional", "departments": []any{"operations", "engineering", "security"}, "approvals": []any{"security-review"}}}, map[string]any{"action": "orchestration.policy", "outcome": "enabled", "details": map[string]any{"tier": "premium", "entitlement_status": "included", "billing_model": "premium-included", "estimated_cost_usd": 4.5, "included_usage_units": 3, "blocked_departments": []any{}}}, map[string]any{"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "security", "reason": "approval required", "required_approvals": []any{"security-review"}}}}},
		{"run_id": "run-b", "task_id": "OPE-66-b", "audits": []any{map[string]any{"action": "orchestration.plan", "outcome": "ready", "details": map[string]any{"collaboration_mode": "tier-limited", "departments": []any{"operations", "engineering"}, "approvals": []any{}}}, map[string]any{"action": "orchestration.policy", "outcome": "upgrade-required", "details": map[string]any{"tier": "standard", "entitlement_status": "upgrade-required", "billing_model": "standard-blocked", "estimated_cost_usd": 7.0, "included_usage_units": 2, "overage_usage_units": 1, "overage_cost_usd": 4.0, "blocked_departments": []any{"customer-success"}}}, map[string]any{"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "operations", "reason": "entitlement gap", "required_approvals": []any{"ops-manager"}}}}},
	}, "Ledger Portfolio", "2026-03-10")
	if portfolio.TotalRuns != 2 || portfolio.CollaborationModes["cross-functional"] != 1 || portfolio.CollaborationModes["tier-limited"] != 1 || portfolio.TierCounts["premium"] != 1 || portfolio.TierCounts["standard"] != 1 || portfolio.EntitlementCounts["included"] != 1 || portfolio.EntitlementCounts["upgrade-required"] != 1 || portfolio.TotalEstimatedCostUSD != 11.5 || portfolio.TakeoverQueue == nil || portfolio.TakeoverQueue.PendingRequests != 2 || portfolio.Recommendation != "stabilize-security-takeovers" {
		t.Fatalf("unexpected ledger portfolio: %+v", portfolio)
	}
}

func TestBuildBillingEntitlementsPageRollsUpOrchestrationCosts(t *testing.T) {
	page := BuildBillingEntitlementsPage(OrchestrationPortfolio{
		Canvases: []OrchestrationCanvas{
			{TaskID: "OPE-104-a", RunID: "run-billing-a", CollaborationMode: "cross-functional", Departments: []string{"operations", "engineering", "security"}, Tier: "premium", EntitlementStatus: "included", BillingModel: "premium-included", EstimatedCostUSD: 4.5, IncludedUsageUnits: 3, HandoffTeam: "security"},
			{TaskID: "OPE-104-b", RunID: "run-billing-b", CollaborationMode: "tier-limited", Departments: []string{"operations", "engineering"}, Tier: "standard", UpgradeRequired: true, EntitlementStatus: "upgrade-required", BillingModel: "standard-blocked", EstimatedCostUSD: 7.0, IncludedUsageUnits: 2, OverageUsageUnits: 1, OverageCostUSD: 4.0, BlockedDepartments: []string{"customer-success"}, HandoffTeam: "operations"},
		},
	}, "OpenAGI Revenue Cloud", "Standard", "2026-03")
	report := RenderBillingEntitlementsReport(page)
	if page.RunCount != 2 || page.TotalIncludedUsageUnits != 5 || page.TotalOverageUsageUnits != 1 || page.TotalEstimatedCostUSD != 11.5 || page.TotalOverageCostUSD != 4.0 || page.UpgradeRequiredCount != 1 || page.EntitlementCounts["included"] != 1 || page.EntitlementCounts["upgrade-required"] != 1 || page.BillingModelCounts["premium-included"] != 1 || page.BillingModelCounts["standard-blocked"] != 1 || strings.Join(page.BlockedCapabilities, ",") != "customer-success" || page.Recommendation != "resolve-plan-gaps" {
		t.Fatalf("unexpected billing page: %+v", page)
	}
	for _, fragment := range []string{"# Billing & Entitlements Report", "- Workspace: OpenAGI Revenue Cloud", "- Overage Cost (USD): 4.00", "- run-billing-b: task=OPE-104-b entitlement=upgrade-required billing=standard-blocked"} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}

func TestRenderBillingEntitlementsPageOutputsHTMLDashboard(t *testing.T) {
	pageHTML := RenderBillingEntitlementsPage(BillingEntitlementsPage{
		WorkspaceName: "OpenAGI Revenue Cloud",
		PlanName:      "Premium",
		BillingPeriod: "2026-03",
		Charges: []BillingRunCharge{{
			RunID:              "run-billing-a",
			TaskID:             "OPE-104-a",
			EntitlementStatus:  "included",
			BillingModel:       "premium-included",
			EstimatedCostUSD:   4.5,
			IncludedUsageUnits: 3,
			Recommendation:     "review-security-takeover",
		}},
	})
	for _, fragment := range []string{"<title>Billing & Entitlements", "OpenAGI Revenue Cloud", "Premium plan for 2026-03", "Charge Feed", "premium-included"} {
		if !strings.Contains(pageHTML, fragment) {
			t.Fatalf("expected %q in html, got %s", fragment, pageHTML)
		}
	}
}

func TestBuildBillingEntitlementsPageFromLedgerExtractsUpgradeSignals(t *testing.T) {
	page := BuildBillingEntitlementsPageFromLedger([]map[string]any{
		{"run_id": "run-ledger-a", "task_id": "OPE-104-a", "audits": []any{map[string]any{"action": "orchestration.plan", "outcome": "ready", "details": map[string]any{"collaboration_mode": "cross-functional", "departments": []any{"operations", "engineering", "security"}, "approvals": []any{"security-review"}}}, map[string]any{"action": "orchestration.policy", "outcome": "enabled", "details": map[string]any{"tier": "premium", "entitlement_status": "included", "billing_model": "premium-included", "estimated_cost_usd": 4.5, "included_usage_units": 3, "blocked_departments": []any{}}}}},
		{"run_id": "run-ledger-b", "task_id": "OPE-104-b", "audits": []any{map[string]any{"action": "orchestration.plan", "outcome": "ready", "details": map[string]any{"collaboration_mode": "tier-limited", "departments": []any{"operations", "engineering"}, "approvals": []any{}}}, map[string]any{"action": "orchestration.policy", "outcome": "upgrade-required", "details": map[string]any{"tier": "standard", "entitlement_status": "upgrade-required", "billing_model": "standard-blocked", "estimated_cost_usd": 7.0, "included_usage_units": 2, "overage_usage_units": 1, "overage_cost_usd": 4.0, "blocked_departments": []any{"customer-success"}}}, map[string]any{"action": "orchestration.handoff", "outcome": "pending", "details": map[string]any{"target_team": "operations", "reason": "entitlement gap", "required_approvals": []any{"ops-manager"}}}}},
	}, "OpenAGI Revenue Cloud", "Standard", "2026-03")
	if page.RunCount != 2 || page.Recommendation != "resolve-plan-gaps" || page.TotalOverageCostUSD != 4.0 || strings.Join(page.Charges[1].BlockedCapabilities, ",") != "customer-success" || page.Charges[1].HandoffTeam != "operations" {
		t.Fatalf("unexpected billing ledger page: %+v", page)
	}
}

func TestTriageFeedbackRecordUsesTimezoneAwareUTCTimestamp(t *testing.T) {
	record := NewTriageFeedbackRecord("run-1", "classify", "accepted", "ops", "")
	if !strings.HasSuffix(record.Timestamp, "Z") {
		t.Fatalf("expected Z timestamp, got %s", record.Timestamp)
	}
	parsed, err := time.Parse(time.RFC3339, record.Timestamp)
	if err != nil || parsed.Location() != time.UTC {
		t.Fatalf("unexpected timestamp parse: %v %v", parsed, err)
	}
}

func TestIssueValidationReportUsesTimezoneAwareUTCTimestamp(t *testing.T) {
	content := RenderIssueValidationReport("BIG-900", "v1", "repo", "pass")
	var value string
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "- 生成时间:") {
			value = strings.TrimSpace(strings.TrimPrefix(line, "- 生成时间:"))
		}
	}
	if !strings.HasSuffix(value, "Z") {
		t.Fatalf("expected UTC Z timestamp, got %s", value)
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil || parsed.Location() != time.UTC {
		t.Fatalf("unexpected timestamp parse: %v %v", parsed, err)
	}
}
