package reporting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func boolPtr(value bool) *bool { return &value }
func intPtr(value int) *int    { return &value }

func TestRenderAndWriteIssueValidationReport(t *testing.T) {
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
	enabled := ConsoleAction{ActionID: "retry", Label: "Retry", Target: "run-1", Enabled: true}
	disabled := ConsoleAction{ActionID: "pause", Label: "Pause", Target: "run-1", Enabled: false, Reason: "already completed"}
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
		t.Fatalf("write report studio bundle: %v", err)
	}

	if !studio.Ready() || studio.Recommendation() != "publish" {
		t.Fatalf("unexpected studio readiness: %+v", studio)
	}
	if !strings.Contains(markdown, "# Report Studio") || !strings.Contains(markdown, "### What changed") {
		t.Fatalf("unexpected markdown: %s", markdown)
	}
	if !strings.Contains(plainText, "Recommendation: publish") {
		t.Fatalf("unexpected plain text: %s", plainText)
	}
	if !strings.Contains(html, "<h1>Executive Weekly Narrative</h1>") {
		t.Fatalf("unexpected html: %s", html)
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
		t.Fatalf("unexpected scorecard summary: %+v", scorecard)
	}
	if payback := scorecard.PaybackMonths(); payback == nil || *payback != 1.9 {
		t.Fatalf("unexpected payback: %v", scorecard.PaybackMonths())
	}
	for _, want := range []string{"Annualized ROI: 200.0%", "Recommendation: go", "Benchmark Score: 96", "Automation coverage"} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected %q in content, got %s", want, content)
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
		t.Fatalf("unexpected scorecard outcome: %+v", scorecard)
	}
}

func TestIssueClosureAndChecklists(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "validation.md")
	decision := EvaluateIssueClosure("BIG-602", reportPath, true, nil, nil)
	if decision.Allowed || decision.Reason != "validation report required before closing issue" || ValidationReportExists(reportPath) {
		t.Fatalf("unexpected missing-report decision: %+v", decision)
	}

	if err := WriteReport(reportPath, "# Validation\n\nfailed"); err != nil {
		t.Fatalf("write report: %v", err)
	}
	decision = EvaluateIssueClosure("BIG-602", reportPath, false, nil, nil)
	if decision.Allowed || decision.Reason != "validation failed; issue must remain open" || !ValidationReportExists(reportPath) {
		t.Fatalf("unexpected failed-validation decision: %+v", decision)
	}

	if err := WriteReport(reportPath, RenderIssueValidationReport("BIG-602", "v0.1", "sandbox", "pass")); err != nil {
		t.Fatalf("write passing report: %v", err)
	}
	decision = EvaluateIssueClosure("BIG-602", reportPath, true, nil, nil)
	if !decision.Allowed || decision.Reason != "validation report and launch checklist requirements satisfied; issue can be closed" || decision.ReportPath != reportPath {
		t.Fatalf("unexpected passing decision: %+v", decision)
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
	status := checklist.DocumentationStatus()
	if status["runbook"] != true || status["faq"] != false || checklist.CompletedItems() != 1 {
		t.Fatalf("unexpected checklist status: %+v", checklist)
	}
	if got := checklist.MissingDocumentation(); len(got) != 1 || got[0] != "faq" || checklist.Ready() {
		t.Fatalf("unexpected missing docs: %+v", got)
	}
	for _, want := range []string{"runbook: available=True", "faq: available=False", "Support handoff: completed=False evidence=faq"} {
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
	required := checklist.RequiredOutputStatus()
	recommended := checklist.RecommendedDocumentationStatus()
	if !required["validation-bundle"] || required["release-notes"] || recommended["runbook"] || recommended["faq"] {
		t.Fatalf("unexpected checklist statuses: required=%+v recommended=%+v", required, recommended)
	}
	if checklist.GeneratedRequiredOutputs() != 1 || checklist.GeneratedRecommendedDocumentation() != 0 || checklist.Ready() {
		t.Fatalf("unexpected checklist counters: %+v", checklist)
	}
	for _, want := range []string{"Required Outputs Generated: 1/2", "Recommended Docs Generated: 0/2", "release-notes: available=False", "runbook: available=False"} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in report, got %s", want, report)
		}
	}
}

func TestIssueClosureHonorsLinkedChecklists(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, "validation.md")
	runbook := filepath.Join(root, "runbook.md")
	faq := filepath.Join(root, "launch-faq.md")
	if err := WriteReport(reportPath, RenderIssueValidationReport("BIG-1003", "v0.2", "staging", "pass")); err != nil {
		t.Fatalf("write validation report: %v", err)
	}
	if err := WriteReport(runbook, "# Runbook\n\nready"); err != nil {
		t.Fatalf("write runbook: %v", err)
	}

	launchChecklist := BuildLaunchChecklist("BIG-1003",
		[]DocumentationArtifact{{Name: "runbook", Path: runbook}, {Name: "launch-faq", Path: faq}},
		[]LaunchChecklistItem{{Name: "Launch comms", Evidence: []string{"runbook", "launch-faq"}}},
	)
	decision := EvaluateIssueClosure("BIG-1003", reportPath, true, &launchChecklist, nil)
	if decision.Allowed || decision.Reason != "launch checklist incomplete; linked documentation missing or empty" {
		t.Fatalf("unexpected incomplete launch decision: %+v", decision)
	}

	if err := WriteReport(faq, "# FAQ\n\nready"); err != nil {
		t.Fatalf("write faq: %v", err)
	}
	launchChecklist = BuildLaunchChecklist("BIG-1003",
		[]DocumentationArtifact{{Name: "runbook", Path: runbook}, {Name: "launch-faq", Path: faq}},
		[]LaunchChecklistItem{{Name: "Launch comms", Evidence: []string{"runbook", "launch-faq"}}},
	)
	decision = EvaluateIssueClosure("BIG-1003", reportPath, true, &launchChecklist, nil)
	if !decision.Allowed || decision.Reason != "validation report and launch checklist requirements satisfied; issue can be closed" {
		t.Fatalf("unexpected ready launch decision: %+v", decision)
	}

	finalChecklist := BuildFinalDeliveryChecklist("BIG-4702",
		[]DocumentationArtifact{{Name: "validation-bundle", Path: filepath.Join(root, "validation-bundle.md")}},
		[]DocumentationArtifact{{Name: "runbook", Path: filepath.Join(root, "runbook.md")}},
	)
	decision = EvaluateIssueClosure("BIG-4702", reportPath, true, nil, &finalChecklist)
	if decision.Allowed || decision.Reason != "final delivery checklist incomplete; required outputs missing" {
		t.Fatalf("unexpected incomplete final delivery decision: %+v", decision)
	}

	if err := WriteReport(filepath.Join(root, "validation-bundle.md"), "# Validation Bundle\n\nready"); err != nil {
		t.Fatalf("write validation bundle: %v", err)
	}
	if err := WriteReport(filepath.Join(root, "release-notes.md"), "# Release Notes\n\nready"); err != nil {
		t.Fatalf("write release notes: %v", err)
	}
	finalChecklist = BuildFinalDeliveryChecklist("BIG-4702",
		[]DocumentationArtifact{{Name: "validation-bundle", Path: filepath.Join(root, "validation-bundle.md")}, {Name: "release-notes", Path: filepath.Join(root, "release-notes.md")}},
		[]DocumentationArtifact{{Name: "runbook", Path: filepath.Join(root, "runbook.md")}},
	)
	decision = EvaluateIssueClosure("BIG-4702", reportPath, true, nil, &finalChecklist)
	if !decision.Allowed || decision.Reason != "validation report and final delivery checklist requirements satisfied; issue can be closed" {
		t.Fatalf("unexpected ready final delivery decision: %+v", decision)
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
	counts := portfolio.RecommendationCounts()
	if portfolio.TotalMonthlyNetValue() != 18500 || portfolio.AverageROI() != 195.2 || counts["go"] != 1 || counts["iterate"] != 1 || counts["hold"] != 0 || portfolio.Recommendation() != "continue" {
		t.Fatalf("unexpected portfolio summary: %+v counts=%+v", portfolio, counts)
	}
	for _, want := range []string{"Recommendation Mix: go=1 iterate=1 hold=0", "Partner A: recommendation=go", "Partner B: recommendation=iterate"} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected %q in content, got %s", want, content)
		}
	}
}

func TestUTCRecordAndValidationTimestamp(t *testing.T) {
	record := NewTriageFeedbackRecord("run-1", "classify", "accepted", "ops", "")
	if !strings.HasSuffix(record.Timestamp, "Z") {
		t.Fatalf("expected Z suffix, got %s", record.Timestamp)
	}
	parsed, err := time.Parse(time.RFC3339Nano, record.Timestamp)
	if err != nil || parsed.Location() != time.UTC {
		t.Fatalf("expected UTC timestamp, got %s err=%v", record.Timestamp, err)
	}

	content := RenderIssueValidationReport("BIG-900", "v1", "repo", "pass")
	var timestampValue string
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "- 生成时间:") {
			timestampValue = strings.SplitN(line, ": ", 2)[1]
			break
		}
	}
	if timestampValue == "" || !strings.HasSuffix(timestampValue, "Z") {
		t.Fatalf("expected timestamp line in content: %s", content)
	}
	parsed, err = time.Parse(time.RFC3339Nano, timestampValue)
	if err != nil || parsed.Location() != time.UTC {
		t.Fatalf("expected UTC validation timestamp, got %s err=%v", timestampValue, err)
	}
}
