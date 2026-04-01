package reporting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bigclaw-go/internal/collaboration"
)

func boolPtr(value bool) *bool { return &value }
func intPtr(value int) *int    { return &value }

func TestRenderIssueValidationReportUsesUTCISOTime(t *testing.T) {
	content := RenderIssueValidationReport("BIG-900", "v1", "repo", "pass")
	if !strings.Contains(content, "- Issue ID: BIG-900") || !strings.Contains(content, "## 结论") {
		t.Fatalf("unexpected validation report: %s", content)
	}
	var timestamp string
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "- 生成时间: ") {
			timestamp = strings.TrimPrefix(line, "- 生成时间: ")
			break
		}
	}
	if !strings.HasSuffix(timestamp, "Z") {
		t.Fatalf("expected UTC timestamp, got %q", timestamp)
	}
	if _, err := time.Parse(time.RFC3339, timestamp); err != nil {
		t.Fatalf("parse timestamp: %v", err)
	}
}

func TestReportStudioRendersBundleAndReadiness(t *testing.T) {
	studio := ReportStudio{
		Name:     "Executive Weekly Narrative",
		IssueID:  "OPE-112",
		Audience: "executive",
		Period:   "2026-W11",
		Summary:  "Delivery recovered after approval bottlenecks were cleared in the second half of the week.",
		Sections: []NarrativeSection{
			{
				Heading:  "What changed",
				Body:     "Approval queue depth fell from 5 to 1 after moving browser-heavy runs onto the shared operations lane.",
				Evidence: []string{"queue-control-center", "weekly-operations"},
				Callouts: []string{"SLA risk contained", "No new regressions opened"},
			},
		},
		ActionItems:   []string{"Publish the markdown export to leadership"},
		SourceReports: []string{"reports/weekly-operations.md"},
	}
	if !studio.Ready() || studio.Recommendation() != "publish" {
		t.Fatalf("expected ready report studio, got %+v", studio)
	}
	markdown := RenderReportStudioReport(studio)
	plain := RenderReportStudioPlainText(studio)
	html := RenderReportStudioHTML(studio)
	artifacts, err := WriteReportStudioBundle(t.TempDir(), studio)
	if err != nil {
		t.Fatalf("write report studio bundle: %v", err)
	}
	if !strings.Contains(markdown, "# Report Studio") || !strings.Contains(markdown, "### What changed") {
		t.Fatalf("unexpected markdown: %s", markdown)
	}
	if !strings.Contains(plain, "Recommendation: publish") {
		t.Fatalf("unexpected plain text: %s", plain)
	}
	if !strings.Contains(html, "<h1>Executive Weekly Narrative</h1>") {
		t.Fatalf("unexpected html: %s", html)
	}
	for _, path := range []string{artifacts.MarkdownPath, artifacts.HTMLPath, artifacts.TextPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected bundle path %s: %v", path, err)
		}
	}
	if !strings.Contains(artifacts.MarkdownPath, "executive-weekly-narrative.md") {
		t.Fatalf("unexpected markdown path: %+v", artifacts)
	}
}

func TestReportStudioRequiresSummaryAndReadySections(t *testing.T) {
	studio := ReportStudio{
		Name:     "Draft Narrative",
		IssueID:  "OPE-112",
		Audience: "operations",
		Period:   "2026-W11",
		Sections: []NarrativeSection{{Heading: "Open risks", Body: ""}},
	}
	if studio.Ready() || studio.Recommendation() != "draft" {
		t.Fatalf("expected draft studio, got %+v", studio)
	}
}

func TestLaunchChecklistAndIssueClosureDecisions(t *testing.T) {
	tmp := t.TempDir()
	reportPath := filepath.Join(tmp, "validation.md")
	runbook := filepath.Join(tmp, "runbook.md")
	if err := WriteReport(reportPath, RenderIssueValidationReport("BIG-1003", "v0.2", "staging", "pass")); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := WriteReport(runbook, "# Runbook\n\nready"); err != nil {
		t.Fatalf("write runbook: %v", err)
	}
	checklist := BuildLaunchChecklist(
		"BIG-1003",
		[]DocumentationArtifact{
			{Name: "runbook", Path: runbook},
			{Name: "launch-faq", Path: filepath.Join(tmp, "launch-faq.md")},
		},
		[]LaunchChecklistItem{{Name: "Launch comms", Evidence: []string{"runbook", "launch-faq"}}},
	)
	report := RenderLaunchChecklistReport(checklist)
	if checklist.CompletedItems() != 0 || checklist.Ready() {
		t.Fatalf("expected incomplete checklist, got %+v", checklist)
	}
	if !strings.Contains(report, "runbook: available=true") || !strings.Contains(report, "launch-faq: available=false") {
		t.Fatalf("unexpected launch checklist report: %s", report)
	}
	decision := EvaluateIssueClosure("BIG-1003", reportPath, true, &checklist, nil)
	if decision.Allowed || decision.Reason != "launch checklist incomplete; linked documentation missing or empty" {
		t.Fatalf("unexpected closure decision: %+v", decision)
	}
	if err := WriteReport(filepath.Join(tmp, "launch-faq.md"), "# FAQ\n\nready"); err != nil {
		t.Fatalf("write faq: %v", err)
	}
	checklist = BuildLaunchChecklist(
		"BIG-1003",
		[]DocumentationArtifact{
			{Name: "runbook", Path: runbook},
			{Name: "launch-faq", Path: filepath.Join(tmp, "launch-faq.md")},
		},
		[]LaunchChecklistItem{{Name: "Launch comms", Evidence: []string{"runbook", "launch-faq"}}},
	)
	decision = EvaluateIssueClosure("BIG-1003", reportPath, true, &checklist, nil)
	if !decision.Allowed || decision.Reason != "validation report and launch checklist requirements satisfied; issue can be closed" {
		t.Fatalf("unexpected ready closure decision: %+v", decision)
	}
}

func TestFinalDeliveryChecklistAndIssueClosureDecisions(t *testing.T) {
	tmp := t.TempDir()
	reportPath := filepath.Join(tmp, "validation.md")
	if err := WriteReport(reportPath, RenderIssueValidationReport("BIG-4702", "v0.3", "staging", "pass")); err != nil {
		t.Fatalf("write report: %v", err)
	}
	checklist := BuildFinalDeliveryChecklist(
		"BIG-4702",
		[]DocumentationArtifact{{Name: "validation-bundle", Path: filepath.Join(tmp, "validation-bundle.md")}},
		[]DocumentationArtifact{{Name: "runbook", Path: filepath.Join(tmp, "runbook.md")}},
	)
	report := RenderFinalDeliveryChecklistReport(checklist)
	if checklist.Ready() || len(checklist.MissingRequiredOutputs()) != 1 {
		t.Fatalf("expected missing outputs, got %+v", checklist)
	}
	if !strings.Contains(report, "Required Outputs Generated: 0/1") || !strings.Contains(report, "runbook: available=false") {
		t.Fatalf("unexpected final delivery report: %s", report)
	}
	decision := EvaluateIssueClosure("BIG-4702", reportPath, true, nil, &checklist)
	if decision.Allowed || decision.Reason != "final delivery checklist incomplete; required outputs missing" {
		t.Fatalf("unexpected closure decision: %+v", decision)
	}
	if err := WriteReport(filepath.Join(tmp, "validation-bundle.md"), "# Validation Bundle\n\nready"); err != nil {
		t.Fatalf("write bundle: %v", err)
	}
	checklist = BuildFinalDeliveryChecklist(
		"BIG-4702",
		[]DocumentationArtifact{{Name: "validation-bundle", Path: filepath.Join(tmp, "validation-bundle.md")}},
		nil,
	)
	decision = EvaluateIssueClosure("BIG-4702", reportPath, true, nil, &checklist)
	if !decision.Allowed || decision.Reason != "validation report and final delivery checklist requirements satisfied; issue can be closed" {
		t.Fatalf("unexpected ready closure decision: %+v", decision)
	}
}

func TestPilotScorecardAndPortfolioRendering(t *testing.T) {
	scorecardA := PilotScorecard{
		IssueID:            "OPE-60",
		Customer:           "Partner A",
		Period:             "2026-Q2",
		Metrics:            []PilotMetric{{Name: "Coverage", Baseline: 40, Current: 85, Target: 80, Unit: "%", HigherIsBetter: true}},
		MonthlyBenefit:     15000,
		MonthlyCost:        3000,
		ImplementationCost: 18000,
		BenchmarkScore:     intPtr(97),
		BenchmarkPassed:    boolPtr(true),
	}
	scorecardB := PilotScorecard{
		IssueID:            "OPE-61",
		Customer:           "Partner B",
		Period:             "2026-Q2",
		Metrics:            []PilotMetric{{Name: "Cycle time", Baseline: 12, Current: 7, Target: 5, Unit: "h", HigherIsBetter: false}},
		MonthlyBenefit:     9000,
		MonthlyCost:        2500,
		ImplementationCost: 12000,
		BenchmarkScore:     intPtr(88),
		BenchmarkPassed:    boolPtr(true),
	}
	if scorecardA.Recommendation() != "go" || scorecardB.Recommendation() != "iterate" {
		t.Fatalf("unexpected scorecard recommendations: %+v %+v", scorecardA, scorecardB)
	}
	content := RenderPilotScorecard(scorecardA)
	if !strings.Contains(content, "Annualized ROI: 233.3%") || !strings.Contains(content, "Recommendation: go") {
		t.Fatalf("unexpected scorecard render: %s", content)
	}
	portfolio := PilotPortfolio{Name: "Design Partners", Period: "2026-H1", Scorecards: []PilotScorecard{scorecardA, scorecardB}}
	report := RenderPilotPortfolioReport(portfolio)
	counts := portfolio.RecommendationCounts()
	if portfolio.TotalMonthlyNetValue() != 18500 || portfolio.AverageROI() != 195.2 {
		t.Fatalf("unexpected portfolio economics: %+v", portfolio)
	}
	if counts["go"] != 1 || counts["iterate"] != 1 || counts["hold"] != 0 || portfolio.Recommendation() != "continue" {
		t.Fatalf("unexpected recommendation counts: %+v", counts)
	}
	if !strings.Contains(report, "Recommendation Mix: go=1 iterate=1 hold=0") || !strings.Contains(report, "Partner B: recommendation=iterate") {
		t.Fatalf("unexpected portfolio report: %s", report)
	}
}

func TestRenderSharedViewContextIncludesCollaborationAnnotations(t *testing.T) {
	thread := collaboration.BuildCollaborationThread(
		"dashboard",
		"ops-overview",
		[]collaboration.Comment{
			{CommentID: "dashboard-comment-1", Author: "pm", Body: "Please review blocker copy with @ops and @eng.", Mentions: []string{"ops", "eng"}, Anchor: "blockers", Status: "open"},
		},
		[]collaboration.Decision{
			{DecisionID: "dashboard-decision-1", Author: "ops", Outcome: "approved", Summary: "Keep the blocker module visible for managers.", Mentions: []string{"pm"}, FollowUp: "Recheck after next data refresh."},
		},
	)
	lines := RenderSharedViewContext(&SharedViewContext{
		Filters:       []SharedViewFilter{{Label: "Team", Value: "ops"}},
		ResultCount:   4,
		LastUpdated:   "2026-03-11T09:00:00Z",
		Collaboration: &thread,
	})
	content := strings.Join(lines, "\n")
	for _, want := range []string{"## Collaboration", "Surface: dashboard", "Please review blocker copy with @ops and @eng.", "Keep the blocker module visible for managers."} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected %q in shared view context, got %s", want, content)
		}
	}
}
