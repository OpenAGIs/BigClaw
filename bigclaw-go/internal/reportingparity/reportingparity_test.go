package reportingparity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
