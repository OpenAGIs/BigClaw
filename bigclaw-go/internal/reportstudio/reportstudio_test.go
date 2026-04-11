package reportstudio

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func boolPtr(v bool) *bool { return &v }
func intPtr(v int) *int    { return &v }

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
		t.Fatalf("unexpected report body: %s", string(body))
	}
}

func TestConsoleActionStateReflectsEnabledFlag(t *testing.T) {
	enabled := NewConsoleAction("retry", "Retry", "run-1", true, "")
	disabled := NewConsoleAction("pause", "Pause", "run-1", false, "already completed")
	if enabled.State() != "enabled" || disabled.State() != "disabled" {
		t.Fatalf("unexpected action states: %+v %+v", enabled, disabled)
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
		t.Fatalf("write report studio bundle: %v", err)
	}
	if !studio.Ready() || studio.Recommendation() != "publish" {
		t.Fatalf("unexpected studio readiness: %+v", studio)
	}
	for _, want := range []string{"# Report Studio", "### What changed"} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("expected %q in markdown, got %s", want, markdown)
		}
	}
	if !strings.Contains(plainText, "Recommendation: publish") || !strings.Contains(html, "<h1>Executive Weekly Narrative</h1>") {
		t.Fatalf("unexpected studio exports")
	}
	for _, path := range []string{artifacts.MarkdownPath, artifacts.HTMLPath, artifacts.TextPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected artifact %s: %v", path, err)
		}
	}
	if !strings.Contains(artifacts.MarkdownPath, "executive-weekly-narrative.md") {
		t.Fatalf("unexpected markdown path: %+v", artifacts)
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
		t.Fatalf("expected draft studio, got %+v", studio)
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
		BenchmarkScore:     intPtr(96),
		BenchmarkPassed:    boolPtr(true),
	}
	content := RenderPilotScorecard(scorecard)
	if scorecard.MetricsMet() != 2 || scorecard.Recommendation() != "go" {
		t.Fatalf("unexpected scorecard readiness: %+v", scorecard)
	}
	if payback := scorecard.PaybackMonths(); payback == nil || *payback != 1.9 {
		t.Fatalf("unexpected payback: %v", payback)
	}
	for _, want := range []string{"Annualized ROI: 200.0%", "Recommendation: go", "Benchmark Score: 96", "Automation coverage"} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected %q in scorecard, got %s", want, content)
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
		BenchmarkPassed:    boolPtr(false),
	}
	if scorecard.MonthlyNetValue() != -2000 || scorecard.PaybackMonths() != nil || scorecard.Recommendation() != "hold" {
		t.Fatalf("unexpected negative-value scorecard: %+v", scorecard)
	}
}

func TestIssueClosureRequiresNonEmptyValidationReport(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "validation.md")
	decision := EvaluateIssueClosure("BIG-602", reportPath, true, nil, nil)
	if decision.Allowed || decision.Reason != "validation report required before closing issue" || ValidationReportExists(reportPath) {
		t.Fatalf("unexpected closure decision: %+v", decision)
	}
}

func TestIssueClosureBlocksFailedValidationReport(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "validation.md")
	if err := WriteReport(reportPath, "# Validation\n\nfailed"); err != nil {
		t.Fatalf("write report: %v", err)
	}
	decision := EvaluateIssueClosure("BIG-602", reportPath, false, nil, nil)
	if decision.Allowed || decision.Reason != "validation failed; issue must remain open" || !ValidationReportExists(reportPath) {
		t.Fatalf("unexpected closure decision: %+v", decision)
	}
}

func TestIssueClosureAllowsCompletedValidationReport(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "validation.md")
	if err := WriteReport(reportPath, RenderIssueValidationReport("BIG-602", "v0.1", "sandbox", "pass")); err != nil {
		t.Fatalf("write report: %v", err)
	}
	decision := EvaluateIssueClosure("BIG-602", reportPath, true, nil, nil)
	if !decision.Allowed || decision.Reason != "validation report and launch checklist requirements satisfied; issue can be closed" || decision.ReportPath != reportPath {
		t.Fatalf("unexpected closure decision: %+v", decision)
	}
}

func TestLaunchChecklistAutoLinksDocumentationStatus(t *testing.T) {
	root := t.TempDir()
	runbook := filepath.Join(root, "runbook.md")
	faq := filepath.Join(root, "faq.md")
	if err := WriteReport(runbook, "# Runbook\n\nready"); err != nil {
		t.Fatalf("write runbook: %v", err)
	}
	checklist := BuildLaunchChecklist("BIG-1003",
		[]DocumentationArtifact{{Name: "runbook", Path: runbook}, {Name: "faq", Path: faq}},
		[]LaunchChecklistItem{{Name: "Operations handoff", Evidence: []string{"runbook"}}, {Name: "Support handoff", Evidence: []string{"faq"}}},
	)
	report := RenderLaunchChecklistReport(checklist)
	wantStatus := map[string]bool{"runbook": true, "faq": false}
	if got := checklist.DocumentationStatus(); got["runbook"] != wantStatus["runbook"] || got["faq"] != wantStatus["faq"] {
		t.Fatalf("unexpected doc status: %+v", got)
	}
	if checklist.CompletedItems() != 1 || strings.Join(checklist.MissingDocumentation(), ",") != "faq" || checklist.Ready() {
		t.Fatalf("unexpected checklist status: %+v", checklist)
	}
	for _, want := range []string{"runbook: available=true", "faq: available=false", "Support handoff: completed=false evidence=faq"} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in report, got %s", want, report)
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
	checklist := BuildFinalDeliveryChecklist("BIG-4702",
		[]DocumentationArtifact{{Name: "validation-bundle", Path: validationBundle}, {Name: "release-notes", Path: releaseNotes}},
		[]DocumentationArtifact{{Name: "runbook", Path: filepath.Join(root, "runbook.md")}, {Name: "faq", Path: filepath.Join(root, "faq.md")}},
	)
	report := RenderFinalDeliveryChecklistReport(checklist)
	if got := checklist.RequiredOutputStatus(); got["validation-bundle"] != true || got["release-notes"] != false {
		t.Fatalf("unexpected required output status: %+v", got)
	}
	if got := checklist.RecommendedDocumentationStatus(); got["runbook"] != false || got["faq"] != false {
		t.Fatalf("unexpected recommended status: %+v", got)
	}
	if checklist.GeneratedRequiredOutputs() != 1 || checklist.GeneratedRecommendedDocumentation() != 0 {
		t.Fatalf("unexpected generated counts: %+v", checklist)
	}
	if strings.Join(checklist.MissingRequiredOutputs(), ",") != "release-notes" || strings.Join(checklist.MissingRecommendedDocumentation(), ",") != "runbook,faq" || checklist.Ready() {
		t.Fatalf("unexpected checklist readiness: %+v", checklist)
	}
	for _, want := range []string{"Required Outputs Generated: 1/2", "Recommended Docs Generated: 0/2", "release-notes: available=false", "runbook: available=false"} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in report, got %s", want, report)
		}
	}
}

func TestIssueClosureBlocksIncompleteLinkedLaunchChecklist(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, "validation.md")
	runbook := filepath.Join(root, "runbook.md")
	_ = WriteReport(reportPath, RenderIssueValidationReport("BIG-1003", "v0.2", "staging", "pass"))
	_ = WriteReport(runbook, "# Runbook\n\nready")
	checklist := BuildLaunchChecklist("BIG-1003",
		[]DocumentationArtifact{{Name: "runbook", Path: runbook}, {Name: "launch-faq", Path: filepath.Join(root, "launch-faq.md")}},
		[]LaunchChecklistItem{{Name: "Launch comms", Evidence: []string{"runbook", "launch-faq"}}},
	)
	decision := EvaluateIssueClosure("BIG-1003", reportPath, true, &checklist, nil)
	if decision.Allowed || decision.Reason != "launch checklist incomplete; linked documentation missing or empty" {
		t.Fatalf("unexpected closure decision: %+v", decision)
	}
}

func TestIssueClosureBlocksMissingRequiredFinalDeliveryOutputs(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, "validation.md")
	_ = WriteReport(reportPath, RenderIssueValidationReport("BIG-4702", "v0.3", "staging", "pass"))
	checklist := BuildFinalDeliveryChecklist("BIG-4702",
		[]DocumentationArtifact{{Name: "validation-bundle", Path: filepath.Join(root, "validation-bundle.md")}},
		[]DocumentationArtifact{{Name: "runbook", Path: filepath.Join(root, "runbook.md")}},
	)
	decision := EvaluateIssueClosure("BIG-4702", reportPath, true, nil, &checklist)
	if decision.Allowed || decision.Reason != "final delivery checklist incomplete; required outputs missing" {
		t.Fatalf("unexpected closure decision: %+v", decision)
	}
}

func TestIssueClosureAllowsWhenRequiredFinalDeliveryOutputsExist(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, "validation.md")
	validationBundle := filepath.Join(root, "validation-bundle.md")
	releaseNotes := filepath.Join(root, "release-notes.md")
	_ = WriteReport(reportPath, RenderIssueValidationReport("BIG-4702", "v0.3", "staging", "pass"))
	_ = WriteReport(validationBundle, "# Validation Bundle\n\nready")
	_ = WriteReport(releaseNotes, "# Release Notes\n\nready")
	checklist := BuildFinalDeliveryChecklist("BIG-4702",
		[]DocumentationArtifact{{Name: "validation-bundle", Path: validationBundle}, {Name: "release-notes", Path: releaseNotes}},
		[]DocumentationArtifact{{Name: "runbook", Path: filepath.Join(root, "runbook.md")}},
	)
	decision := EvaluateIssueClosure("BIG-4702", reportPath, true, nil, &checklist)
	if !decision.Allowed || decision.Reason != "validation report and final delivery checklist requirements satisfied; issue can be closed" {
		t.Fatalf("unexpected closure decision: %+v", decision)
	}
}

func TestIssueClosureAllowsWhenLinkedLaunchChecklistIsReady(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, "validation.md")
	runbook := filepath.Join(root, "runbook.md")
	faq := filepath.Join(root, "launch-faq.md")
	_ = WriteReport(reportPath, RenderIssueValidationReport("BIG-1003", "v0.2", "staging", "pass"))
	_ = WriteReport(runbook, "# Runbook\n\nready")
	_ = WriteReport(faq, "# FAQ\n\nready")
	checklist := BuildLaunchChecklist("BIG-1003",
		[]DocumentationArtifact{{Name: "runbook", Path: runbook}, {Name: "launch-faq", Path: faq}},
		[]LaunchChecklistItem{{Name: "Launch comms", Evidence: []string{"runbook", "launch-faq"}}},
	)
	decision := EvaluateIssueClosure("BIG-1003", reportPath, true, &checklist, nil)
	if !decision.Allowed || decision.Reason != "validation report and launch checklist requirements satisfied; issue can be closed" {
		t.Fatalf("unexpected closure decision: %+v", decision)
	}
}

func TestRenderPilotPortfolioReportSummarizesCommercialReadiness(t *testing.T) {
	portfolio := PilotPortfolio{
		Name:   "Design Partners",
		Period: "2026-H1",
		Scorecards: []PilotScorecard{
			{IssueID: "OPE-60", Customer: "Partner A", Period: "2026-Q2", Metrics: []PilotMetric{{Name: "Coverage", Baseline: 40, Current: 85, Target: 80, Unit: "%", HigherIsBetter: true}}, MonthlyBenefit: 15000, MonthlyCost: 3000, ImplementationCost: 18000, BenchmarkScore: intPtr(97), BenchmarkPassed: boolPtr(true)},
			{IssueID: "OPE-61", Customer: "Partner B", Period: "2026-Q2", Metrics: []PilotMetric{{Name: "Cycle time", Baseline: 12, Current: 7, Target: 5, Unit: "h", HigherIsBetter: false}}, MonthlyBenefit: 9000, MonthlyCost: 2500, ImplementationCost: 12000, BenchmarkScore: intPtr(88), BenchmarkPassed: boolPtr(true)},
		},
	}
	content := RenderPilotPortfolioReport(portfolio)
	if portfolio.TotalMonthlyNetValue() != 18500 || portfolio.AverageROI() != 195.2 {
		t.Fatalf("unexpected portfolio metrics: %+v", portfolio)
	}
	counts := portfolio.RecommendationCounts()
	if counts["go"] != 1 || counts["iterate"] != 1 || counts["hold"] != 0 || portfolio.Recommendation() != "continue" {
		t.Fatalf("unexpected portfolio recommendation mix: %+v", counts)
	}
	for _, want := range []string{"Recommendation Mix: go=1 iterate=1 hold=0", "Partner A: recommendation=go", "Partner B: recommendation=iterate"} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected %q in portfolio report, got %s", want, content)
		}
	}
}
