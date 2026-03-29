package reporting

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/regression"
)

func TestBuildWeeklyReportRendersExpandedMarkdown(t *testing.T) {
	start := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	end := start.Add(7 * 24 * time.Hour)
	tasks := []domain.Task{
		{
			ID:          "task-1",
			Title:       "Deploy API",
			State:       domain.TaskSucceeded,
			RiskLevel:   domain.RiskHigh,
			BudgetCents: 1200,
			Metadata: map[string]string{
				"team":             "platform",
				"plan":             "premium",
				"regression_count": "2",
			},
			UpdatedAt: start.Add(2 * time.Hour),
		},
		{
			ID:          "task-2",
			Title:       "Fix flaky validation",
			State:       domain.TaskBlocked,
			BudgetCents: 800,
			Metadata: map[string]string{
				"team": "platform",
			},
			UpdatedAt: start.Add(3 * time.Hour),
		},
	}
	events := []domain.Event{
		{ID: "evt-1", Type: domain.EventRunTakeover, TaskID: "task-1", Timestamp: start.Add(30 * time.Minute)},
		{ID: "evt-2", Type: domain.EventControlPaused, TaskID: "task-2", Timestamp: start.Add(4 * time.Hour)},
	}

	weekly := Build(tasks, events, start, end)
	if weekly.Summary.TotalRuns != 2 || weekly.Summary.CompletedRuns != 1 || weekly.Summary.BlockedRuns != 1 {
		t.Fatalf("unexpected weekly summary: %+v", weekly.Summary)
	}
	if weekly.Summary.HighRiskRuns != 1 || weekly.Summary.RegressionFindings != 2 || weekly.Summary.HumanInterventions != 2 || weekly.Summary.PremiumRuns != 1 {
		t.Fatalf("unexpected weekly counters: %+v", weekly.Summary)
	}
	if len(weekly.TeamBreakdown) != 1 || weekly.TeamBreakdown[0].Key != "platform" {
		t.Fatalf("unexpected team breakdown: %+v", weekly.TeamBreakdown)
	}
	for _, fragment := range []string{"## Team Breakdown", "## Highlights", "- High risk runs: 1", "- Premium runs: 1", "platform: total=2 completed=1 blocked=1"} {
		if !strings.Contains(weekly.Markdown, fragment) {
			t.Fatalf("expected %q in weekly markdown, got %s", fragment, weekly.Markdown)
		}
	}
}

func TestRenderPilotScorecardIncludesROIAndRecommendation(t *testing.T) {
	benchmarkScore := 96
	benchmarkPassed := true
	scorecard := PilotScorecard{
		IssueID:  "OPE-60",
		Customer: "Design Partner A",
		Period:   "2026-Q2",
		Metrics: []PilotMetric{
			{Name: "Automation coverage", Baseline: 35, Current: 82, Target: 80, Unit: "%", HigherIsBetter: true},
			{Name: "Manual review time", Baseline: 12, Current: 4, Target: 5, Unit: "h", HigherIsBetter: false},
		},
		MonthlyBenefit:     12000,
		MonthlyCost:        2500,
		ImplementationCost: 18000,
		BenchmarkScore:     &benchmarkScore,
		BenchmarkPassed:    &benchmarkPassed,
	}
	content := RenderPilotScorecard(scorecard)
	if scorecard.MetricsMet() != 2 || scorecard.Recommendation() != "go" {
		t.Fatalf("unexpected scorecard rollup: %+v", scorecard)
	}
	if payback := scorecard.PaybackMonths(); payback == nil || *payback != 1.9 {
		t.Fatalf("unexpected payback months: %v", payback)
	}
	for _, want := range []string{"Annualized ROI: 200.0%", "Recommendation: go", "Benchmark Score: 96", "Automation coverage"} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected %q in scorecard content, got %s", want, content)
		}
	}
}

func TestPilotScorecardReturnsHoldWhenValueIsNegative(t *testing.T) {
	benchmarkPassed := false
	scorecard := PilotScorecard{
		IssueID:  "OPE-60",
		Customer: "Design Partner B",
		Period:   "2026-Q2",
		Metrics: []PilotMetric{
			{Name: "Backlog aging", Baseline: 5, Current: 7, Target: 4, Unit: "d", HigherIsBetter: false},
		},
		MonthlyBenefit:     1000,
		MonthlyCost:        3000,
		ImplementationCost: 12000,
		BenchmarkPassed:    &benchmarkPassed,
	}
	if scorecard.MonthlyNetValue() != -2000 || scorecard.PaybackMonths() != nil || scorecard.Recommendation() != "hold" {
		t.Fatalf("unexpected hold scorecard result: %+v", scorecard)
	}
}

func TestIssueClosureRequiresValidationReportAndChecklistEvidence(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "validation.md")
	decision := EvaluateIssueClosure("BIG-602", reportPath, true, nil, nil)
	if decision.Allowed || decision.Reason != "validation report required before closing issue" || ValidationReportExists(reportPath) {
		t.Fatalf("unexpected missing validation decision: %+v", decision)
	}
}

func TestIssueClosureBlocksFailedValidationReport(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "validation.md")
	if err := WriteReport(reportPath, "# Validation\n\nfailed"); err != nil {
		t.Fatalf("write report: %v", err)
	}
	decision := EvaluateIssueClosure("BIG-602", reportPath, false, nil, nil)
	if decision.Allowed || decision.Reason != "validation failed; issue must remain open" || !ValidationReportExists(reportPath) {
		t.Fatalf("unexpected failed validation decision: %+v", decision)
	}
}

func TestIssueClosureAllowsCompletedValidationReport(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "validation.md")
	if err := WriteReport(reportPath, RenderIssueValidationReport("BIG-602", "v0.1", "sandbox", "pass")); err != nil {
		t.Fatalf("write validation report: %v", err)
	}
	decision := EvaluateIssueClosure("BIG-602", reportPath, true, nil, nil)
	if !decision.Allowed || decision.Reason != "validation report and launch checklist requirements satisfied; issue can be closed" || decision.ReportPath != reportPath {
		t.Fatalf("unexpected completed validation decision: %+v", decision)
	}
}

func TestLaunchChecklistAutoLinksDocumentationStatus(t *testing.T) {
	tmp := t.TempDir()
	runbook := filepath.Join(tmp, "runbook.md")
	faq := filepath.Join(tmp, "faq.md")
	if err := WriteReport(runbook, "# Runbook\n\nready"); err != nil {
		t.Fatalf("write runbook: %v", err)
	}
	checklist := BuildLaunchChecklist("BIG-1003", []DocumentationArtifact{{Name: "runbook", Path: runbook}, {Name: "faq", Path: faq}}, []LaunchChecklistItem{{Name: "Operations handoff", Evidence: []string{"runbook"}}, {Name: "Support handoff", Evidence: []string{"faq"}}})
	report := RenderLaunchChecklistReport(checklist)
	if !reflect.DeepEqual(checklist.DocumentationStatus(), map[string]bool{"runbook": true, "faq": false}) || checklist.CompletedItems() != 1 || !reflect.DeepEqual(checklist.MissingDocumentation(), []string{"faq"}) || checklist.Ready() {
		t.Fatalf("unexpected launch checklist rollup: %+v", checklist)
	}
	for _, want := range []string{"runbook: available=true", "faq: available=false", "Support handoff: completed=false evidence=faq"} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in launch checklist report, got %s", want, report)
		}
	}
}

func TestFinalDeliveryChecklistTracksRequiredOutputsAndRecommendedDocs(t *testing.T) {
	tmp := t.TempDir()
	validationBundle := filepath.Join(tmp, "validation-bundle.md")
	releaseNotes := filepath.Join(tmp, "release-notes.md")
	if err := WriteReport(validationBundle, "# Validation Bundle\n\nready"); err != nil {
		t.Fatalf("write validation bundle: %v", err)
	}
	checklist := BuildFinalDeliveryChecklist("BIG-4702", []DocumentationArtifact{{Name: "validation-bundle", Path: validationBundle}, {Name: "release-notes", Path: releaseNotes}}, []DocumentationArtifact{{Name: "runbook", Path: filepath.Join(tmp, "runbook.md")}, {Name: "faq", Path: filepath.Join(tmp, "faq.md")}})
	report := RenderFinalDeliveryChecklistReport(checklist)
	if !reflect.DeepEqual(checklist.RequiredOutputStatus(), map[string]bool{"validation-bundle": true, "release-notes": false}) || !reflect.DeepEqual(checklist.RecommendedDocumentationStatus(), map[string]bool{"runbook": false, "faq": false}) || checklist.GeneratedRequiredOutputs() != 1 || checklist.GeneratedRecommendedDocumentation() != 0 || !reflect.DeepEqual(checklist.MissingRequiredOutputs(), []string{"release-notes"}) || !reflect.DeepEqual(checklist.MissingRecommendedDocumentation(), []string{"runbook", "faq"}) || checklist.Ready() {
		t.Fatalf("unexpected final delivery checklist rollup: %+v", checklist)
	}
	for _, want := range []string{"Required Outputs Generated: 1/2", "Recommended Docs Generated: 0/2", "release-notes: available=false", "runbook: available=false"} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in final delivery report, got %s", want, report)
		}
	}
}

func TestIssueClosureWithLaunchAndFinalDeliveryChecklists(t *testing.T) {
	tmp := t.TempDir()
	reportPath := filepath.Join(tmp, "validation.md")
	if err := WriteReport(reportPath, RenderIssueValidationReport("BIG-1003", "v0.2", "staging", "pass")); err != nil {
		t.Fatalf("write validation report: %v", err)
	}
	runbook := filepath.Join(tmp, "runbook.md")
	faq := filepath.Join(tmp, "launch-faq.md")
	if err := WriteReport(runbook, "# Runbook\n\nready"); err != nil {
		t.Fatalf("write runbook: %v", err)
	}
	launchChecklist := BuildLaunchChecklist("BIG-1003", []DocumentationArtifact{{Name: "runbook", Path: runbook}, {Name: "launch-faq", Path: faq}}, []LaunchChecklistItem{{Name: "Launch comms", Evidence: []string{"runbook", "launch-faq"}}})
	decision := EvaluateIssueClosure("BIG-1003", reportPath, true, &launchChecklist, nil)
	if decision.Allowed || decision.Reason != "launch checklist incomplete; linked documentation missing or empty" {
		t.Fatalf("unexpected blocked launch checklist decision: %+v", decision)
	}
	if err := WriteReport(faq, "# FAQ\n\nready"); err != nil {
		t.Fatalf("write faq: %v", err)
	}
	launchChecklist = BuildLaunchChecklist("BIG-1003", []DocumentationArtifact{{Name: "runbook", Path: runbook}, {Name: "launch-faq", Path: faq}}, []LaunchChecklistItem{{Name: "Launch comms", Evidence: []string{"runbook", "launch-faq"}}})
	decision = EvaluateIssueClosure("BIG-1003", reportPath, true, &launchChecklist, nil)
	if !decision.Allowed || decision.Reason != "validation report and launch checklist requirements satisfied; issue can be closed" {
		t.Fatalf("unexpected ready launch checklist decision: %+v", decision)
	}
	finalChecklist := BuildFinalDeliveryChecklist("BIG-4702", []DocumentationArtifact{{Name: "validation-bundle", Path: filepath.Join(tmp, "validation-bundle.md")}}, []DocumentationArtifact{{Name: "runbook", Path: filepath.Join(tmp, "runbook.md")}})
	decision = EvaluateIssueClosure("BIG-4702", reportPath, true, nil, &finalChecklist)
	if decision.Allowed || decision.Reason != "final delivery checklist incomplete; required outputs missing" {
		t.Fatalf("unexpected blocked final delivery decision: %+v", decision)
	}
	if err := WriteReport(filepath.Join(tmp, "validation-bundle.md"), "# Validation Bundle\n\nready"); err != nil {
		t.Fatalf("write validation bundle: %v", err)
	}
	finalChecklist = BuildFinalDeliveryChecklist("BIG-4702", []DocumentationArtifact{{Name: "validation-bundle", Path: filepath.Join(tmp, "validation-bundle.md")}}, []DocumentationArtifact{{Name: "runbook", Path: filepath.Join(tmp, "runbook.md")}})
	decision = EvaluateIssueClosure("BIG-4702", reportPath, true, nil, &finalChecklist)
	if !decision.Allowed || decision.Reason != "validation report and final delivery checklist requirements satisfied; issue can be closed" {
		t.Fatalf("unexpected ready final delivery decision: %+v", decision)
	}
}

func TestRenderPilotPortfolioReportSummarizesCommercialReadiness(t *testing.T) {
	scoreA := 97
	passA := true
	scoreB := 88
	passB := true
	portfolio := PilotPortfolio{
		Name:   "Design Partners",
		Period: "2026-H1",
		Scorecards: []PilotScorecard{
			{IssueID: "OPE-60", Customer: "Partner A", Period: "2026-Q2", Metrics: []PilotMetric{{Name: "Coverage", Baseline: 40, Current: 85, Target: 80, Unit: "%", HigherIsBetter: true}}, MonthlyBenefit: 15000, MonthlyCost: 3000, ImplementationCost: 18000, BenchmarkScore: &scoreA, BenchmarkPassed: &passA},
			{IssueID: "OPE-61", Customer: "Partner B", Period: "2026-Q2", Metrics: []PilotMetric{{Name: "Cycle time", Baseline: 12, Current: 7, Target: 5, Unit: "h", HigherIsBetter: false}}, MonthlyBenefit: 9000, MonthlyCost: 2500, ImplementationCost: 12000, BenchmarkScore: &scoreB, BenchmarkPassed: &passB},
		},
	}
	content := RenderPilotPortfolioReport(portfolio)
	if portfolio.TotalMonthlyNetValue() != 18500 || portfolio.AverageROI() != 195.2 || !reflect.DeepEqual(portfolio.RecommendationCounts(), map[string]int{"go": 1, "iterate": 1, "hold": 0}) || portfolio.Recommendation() != "continue" {
		t.Fatalf("unexpected pilot portfolio rollup: %+v", portfolio)
	}
	for _, want := range []string{"Recommendation Mix: go=1 iterate=1 hold=0", "Partner A: recommendation=go", "Partner B: recommendation=iterate"} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected %q in pilot portfolio report, got %s", want, content)
		}
	}
}

func TestTriageFeedbackRecordAndIssueValidationReportUseTimezoneAwareUTCTimestamps(t *testing.T) {
	record := NewTriageFeedbackRecord("run-1", "classify", "accepted", "ops", "")
	if !strings.HasSuffix(record.Timestamp, "Z") {
		t.Fatalf("expected UTC Z suffix on feedback timestamp, got %q", record.Timestamp)
	}
	parsedRecord, err := time.Parse(time.RFC3339, record.Timestamp)
	if err != nil || parsedRecord.Location() != time.UTC {
		t.Fatalf("expected UTC feedback timestamp, got %q err=%v", record.Timestamp, err)
	}
	content := RenderIssueValidationReport("BIG-900", "v1", "repo", "pass")
	var timestampValue string
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "- 生成时间:") {
			timestampValue = strings.TrimSpace(strings.TrimPrefix(line, "- 生成时间:"))
			break
		}
	}
	if !strings.HasSuffix(timestampValue, "Z") {
		t.Fatalf("expected UTC Z suffix on report timestamp, got %q", timestampValue)
	}
	parsedReport, err := time.Parse(time.RFC3339, timestampValue)
	if err != nil || parsedReport.Location() != time.UTC {
		t.Fatalf("expected UTC report timestamp, got %q err=%v", timestampValue, err)
	}
}

func TestWriteReportCreatesFileWithContent(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "report.md")
	if err := WriteReport(outputPath, "# Validation\n\npass"); err != nil {
		t.Fatalf("write report: %v", err)
	}
	body, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read written report: %v", err)
	}
	if !strings.Contains(string(body), "Validation") || !strings.Contains(string(body), "pass") {
		t.Fatalf("unexpected report body: %s", string(body))
	}
}

func TestConsoleActionStateReflectsEnabledFlag(t *testing.T) {
	enabled := ConsoleAction{ActionID: "retry", Label: "Retry", Target: "run-1", Enabled: true}
	disabled := ConsoleAction{ActionID: "pause", Label: "Pause", Target: "run-1", Enabled: false, Reason: "already completed"}
	if enabled.State() != "enabled" {
		t.Fatalf("expected enabled action state, got %q", enabled.State())
	}
	if disabled.State() != "disabled" {
		t.Fatalf("expected disabled action state, got %q", disabled.State())
	}
}

func TestBuildOperationsMetricSpec(t *testing.T) {
	start := time.Date(2026, 3, 17, 0, 0, 0, 0, time.UTC)
	end := start.Add(7 * 24 * time.Hour)
	tasks := []domain.Task{
		{ID: "task-1", State: domain.TaskSucceeded, RiskLevel: domain.RiskHigh, BudgetCents: 1200, CreatedAt: start, UpdatedAt: start.Add(31 * time.Minute), Metadata: map[string]string{"regression_count": "2"}},
		{ID: "task-2", State: domain.TaskBlocked, RiskLevel: domain.RiskLow, BudgetCents: 300, CreatedAt: start.Add(2 * time.Hour), UpdatedAt: start.Add(3 * time.Hour), Metadata: map[string]string{"approval_status": "needs-approval"}},
	}
	events := []domain.Event{{ID: "evt-1", Type: domain.EventRunTakeover, TaskID: "task-1", Timestamp: start.Add(2 * time.Hour)}}

	spec := BuildOperationsMetricSpec(tasks, events, start, end, "UTC", 60)
	if spec.Name != "Operations Metric Spec" || spec.TimezoneName != "UTC" {
		t.Fatalf("unexpected spec header: %+v", spec)
	}
	if len(spec.Definitions) != 7 || len(spec.Values) != 7 {
		t.Fatalf("expected seven metric definitions and values, got %+v", spec)
	}
	byID := map[string]OperationsMetricValue{}
	for _, value := range spec.Values {
		byID[value.MetricID] = value
	}
	if byID["runs-window"].DisplayValue != "2" {
		t.Fatalf("unexpected runs-window metric: %+v", byID["runs-window"])
	}
	if byID["avg-cycle-minutes"].DisplayValue != "45.5m" {
		t.Fatalf("unexpected avg-cycle-minutes metric: %+v", byID["avg-cycle-minutes"])
	}
	if byID["intervention-rate"].DisplayValue != "100.0%" {
		t.Fatalf("unexpected intervention-rate metric: %+v", byID["intervention-rate"])
	}
	if byID["sla-compliance"].DisplayValue != "100.0%" {
		t.Fatalf("unexpected sla-compliance metric: %+v", byID["sla-compliance"])
	}
	if byID["regression-findings"].DisplayValue != "2" {
		t.Fatalf("unexpected regression-findings metric: %+v", byID["regression-findings"])
	}
	if byID["avg-risk-score"].DisplayValue != "57.5" {
		t.Fatalf("unexpected avg-risk-score metric: %+v", byID["avg-risk-score"])
	}
	if byID["budget-spend"].DisplayValue != "$15.00" {
		t.Fatalf("unexpected budget-spend metric: %+v", byID["budget-spend"])
	}
	if len(byID["sla-compliance"].Evidence) < 2 || !strings.Contains(byID["sla-compliance"].Evidence[0], "SLA target: 60 minutes") {
		t.Fatalf("expected SLA evidence, got %+v", byID["sla-compliance"])
	}
}

func TestWriteWeeklyOperationsBundle(t *testing.T) {
	rootDir := t.TempDir()
	start := time.Date(2026, 3, 17, 0, 0, 0, 0, time.UTC)
	end := start.Add(7 * 24 * time.Hour)
	weekly := Weekly{
		WeekStart: start,
		WeekEnd:   end,
		Summary: Summary{
			TotalRuns:          4,
			CompletedRuns:      3,
			BlockedRuns:        1,
			HighRiskRuns:       2,
			RegressionFindings: 1,
			HumanInterventions: 2,
			BudgetCentsTotal:   4200,
			PremiumRuns:        1,
		},
		TeamBreakdown: []TeamBreakdown{{
			Key:                "platform",
			TotalRuns:          4,
			CompletedRuns:      3,
			BlockedRuns:        1,
			BudgetCentsTotal:   4200,
			HumanInterventions: 2,
		}},
		Highlights: []string{"Completed 3 / 4 runs this week."},
		Actions:    []string{"Review the blocked run before closeout."},
	}
	spec := &OperationsMetricSpec{
		Name:         "Weekly control-plane metrics",
		GeneratedAt:  start.Add(time.Hour),
		PeriodStart:  start,
		PeriodEnd:    end,
		TimezoneName: "UTC",
		Definitions: []OperationsMetricDefinition{{
			MetricID:     "success_rate",
			Label:        "Success Rate",
			Unit:         "%",
			Direction:    "up",
			Formula:      "completed_runs / total_runs",
			Description:  "Share of runs that completed within the reporting window.",
			SourceFields: []string{"summary.completed_runs", "summary.total_runs"},
		}},
		Values: []OperationsMetricValue{{
			MetricID:     "success_rate",
			Label:        "Success Rate",
			Value:        75,
			DisplayValue: "75.0%",
			Numerator:    3,
			Denominator:  4,
			Unit:         "%",
			Evidence:     []string{"weekly-operations.md", "operations-dashboard.md"},
		}},
	}

	artifacts, err := WriteWeeklyOperationsBundle(rootDir, weekly, spec)
	if err != nil {
		t.Fatalf("write weekly operations bundle: %v", err)
	}
	if artifacts.WeeklyReportPath != filepath.Join(rootDir, "weekly-operations.md") || artifacts.DashboardPath != filepath.Join(rootDir, "operations-dashboard.md") {
		t.Fatalf("unexpected bundle paths: %+v", artifacts)
	}
	if artifacts.MetricSpecPath != filepath.Join(rootDir, "operations-metric-spec.md") {
		t.Fatalf("expected metric spec path, got %+v", artifacts)
	}

	reportBody, err := os.ReadFile(artifacts.WeeklyReportPath)
	if err != nil {
		t.Fatalf("read weekly report: %v", err)
	}
	if !strings.Contains(string(reportBody), "## Team Breakdown") || !strings.Contains(string(reportBody), "Review the blocked run before closeout.") {
		t.Fatalf("unexpected weekly report content: %s", string(reportBody))
	}

	dashboardBody, err := os.ReadFile(artifacts.DashboardPath)
	if err != nil {
		t.Fatalf("read dashboard: %v", err)
	}
	if !strings.Contains(string(dashboardBody), "# Operations Dashboard") || !strings.Contains(string(dashboardBody), "High Risk Runs: 2") {
		t.Fatalf("unexpected dashboard content: %s", string(dashboardBody))
	}

	specBody, err := os.ReadFile(artifacts.MetricSpecPath)
	if err != nil {
		t.Fatalf("read metric spec: %v", err)
	}
	for _, fragment := range []string{"# Operations Metric Spec", "### Success Rate", "value=75.0%"} {
		if !strings.Contains(string(specBody), fragment) {
			t.Fatalf("expected %q in metric spec output, got %s", fragment, string(specBody))
		}
	}
}

func TestAuditDashboardBuilderFlagsGovernanceGaps(t *testing.T) {
	dashboard := DashboardBuilder{
		Name:   "Ops Console",
		Period: "2026-W11",
		Owner:  "operations",
		Permissions: EngineeringOverviewPermission{
			ViewerRole:     "operator",
			AllowedModules: []string{"operations"},
		},
		Widgets: []DashboardWidgetSpec{
			{WidgetID: "ops-summary", Title: "Ops Summary", Module: "operations", DataSource: "/v2/reports/weekly", DefaultWidth: 4, DefaultHeight: 3, MinWidth: 2, MaxWidth: 12},
			{WidgetID: "audit-feed", Title: "Audit Feed", Module: "audit", DataSource: "/v2/audit", DefaultWidth: 4, DefaultHeight: 3, MinWidth: 2, MaxWidth: 12},
		},
		Layouts: []DashboardLayout{
			{
				LayoutID: "primary",
				Name:     "Primary",
				Columns:  12,
				Placements: []DashboardWidgetPlacement{
					{PlacementID: "placement-1", WidgetID: "ops-summary", Column: 0, Row: 0, Width: 6, Height: 2},
					{PlacementID: "placement-1", WidgetID: "missing-widget", Column: 5, Row: 0, Width: 8, Height: 2},
					{PlacementID: "placement-3", WidgetID: "audit-feed", Column: 2, Row: 1, Width: 4, Height: 2},
				},
			},
			{
				LayoutID:   "empty-layout",
				Name:       "Empty",
				Columns:    12,
				Placements: nil,
			},
		},
		DocumentationComplete: false,
	}

	audit := AuditDashboardBuilder(dashboard)
	if audit.TotalWidgets != 2 || audit.LayoutCount != 2 || audit.PlacedWidgets != 3 {
		t.Fatalf("unexpected audit counts: %+v", audit)
	}
	if audit.ReleaseReady() {
		t.Fatalf("expected audit to block release, got %+v", audit)
	}
	for _, expected := range []string{"placement-1"} {
		if !strings.Contains(strings.Join(audit.DuplicatePlacementIDs, ","), expected) {
			t.Fatalf("expected duplicate placement %q in %+v", expected, audit.DuplicatePlacementIDs)
		}
	}
	if len(audit.MissingWidgetDefs) != 1 || audit.MissingWidgetDefs[0] != "missing-widget" {
		t.Fatalf("unexpected missing widget defs: %+v", audit.MissingWidgetDefs)
	}
	if len(audit.InaccessibleWidgets) != 1 || audit.InaccessibleWidgets[0] != "audit-feed" {
		t.Fatalf("unexpected inaccessible widgets: %+v", audit.InaccessibleWidgets)
	}
	if len(audit.OutOfBoundsPlacements) != 1 || audit.OutOfBoundsPlacements[0] != "placement-1" {
		t.Fatalf("unexpected out of bounds placements: %+v", audit.OutOfBoundsPlacements)
	}
	if len(audit.OverlappingPlacements) != 2 {
		t.Fatalf("expected overlapping placements, got %+v", audit.OverlappingPlacements)
	}
	if len(audit.EmptyLayouts) != 1 || audit.EmptyLayouts[0] != "empty-layout" {
		t.Fatalf("unexpected empty layouts: %+v", audit.EmptyLayouts)
	}
}

func TestDashboardBuilderRoundTripPreservesManifestShape(t *testing.T) {
	dashboard := DashboardBuilder{
		Name:   "Ops Console",
		Period: "2026-W11",
		Owner:  "operations",
		Permissions: EngineeringOverviewPermission{
			ViewerRole:     "operator",
			AllowedModules: []string{"operations", "audit"},
		},
		Widgets: []DashboardWidgetSpec{
			{WidgetID: "ops-summary", Title: "Ops Summary", Module: "operations", DataSource: "/v2/reports/weekly", DefaultWidth: 4, DefaultHeight: 3, MinWidth: 2, MaxWidth: 12},
		},
		Layouts: []DashboardLayout{
			{
				LayoutID: "primary",
				Name:     "Primary",
				Columns:  12,
				Placements: []DashboardWidgetPlacement{
					{PlacementID: "placement-1", WidgetID: "ops-summary", Column: 0, Row: 0, Width: 4, Height: 3, Filters: []string{"team=platform"}},
				},
			},
		},
		DocumentationComplete: true,
	}
	payload, err := json.Marshal(dashboard)
	if err != nil {
		t.Fatalf("marshal dashboard: %v", err)
	}
	var restored DashboardBuilder
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal dashboard: %v", err)
	}
	if restored.Name != dashboard.Name || restored.Period != dashboard.Period || restored.Owner != dashboard.Owner || !restored.DocumentationComplete {
		t.Fatalf("unexpected restored dashboard header: %+v", restored)
	}
	if len(restored.Widgets) != 1 || restored.Widgets[0].WidgetID != "ops-summary" {
		t.Fatalf("unexpected restored widgets: %+v", restored.Widgets)
	}
	if len(restored.Layouts) != 1 || len(restored.Layouts[0].Placements) != 1 || restored.Layouts[0].Placements[0].Filters[0] != "team=platform" {
		t.Fatalf("unexpected restored layouts: %+v", restored.Layouts)
	}
}

func TestNormalizeDashboardLayoutClampsDimensionsAndSortsPlacements(t *testing.T) {
	widgets := []DashboardWidgetSpec{
		{WidgetID: "ops-summary", Title: "Ops Summary", Module: "operations", DataSource: "/v2/reports/weekly", MinWidth: 2, MaxWidth: 6},
	}
	layout := DashboardLayout{
		LayoutID: "primary",
		Name:     "Primary",
		Columns:  6,
		Placements: []DashboardWidgetPlacement{
			{PlacementID: "placement-b", WidgetID: "ops-summary", Column: 5, Row: 2, Width: 10, Height: 0},
			{PlacementID: "placement-a", WidgetID: "ops-summary", Column: -1, Row: -1, Width: 1, Height: 2},
		},
	}
	normalized := NormalizeDashboardLayout(layout, widgets)
	if normalized.Columns != 6 {
		t.Fatalf("expected normalized columns 6, got %+v", normalized)
	}
	if len(normalized.Placements) != 2 {
		t.Fatalf("expected 2 placements, got %+v", normalized.Placements)
	}
	first := normalized.Placements[0]
	second := normalized.Placements[1]
	if first.PlacementID != "placement-a" || first.Column != 0 || first.Row != 0 || first.Width != 2 || first.Height != 2 {
		t.Fatalf("unexpected first normalized placement: %+v", first)
	}
	if second.PlacementID != "placement-b" || second.Column != 0 || second.Row != 2 || second.Width != 6 || second.Height != 1 {
		t.Fatalf("unexpected second normalized placement: %+v", second)
	}
}

func TestRenderAndWriteDashboardBuilderBundle(t *testing.T) {
	dashboard := DashboardBuilder{
		Name:   "Ops Console",
		Period: "2026-W11",
		Owner:  "operations",
		Permissions: EngineeringOverviewPermission{
			ViewerRole:     "operator",
			AllowedModules: []string{"operations", "audit"},
		},
		Widgets: []DashboardWidgetSpec{
			{WidgetID: "ops-summary", Title: "Ops Summary", Module: "operations", DataSource: "/v2/reports/weekly", DefaultWidth: 4, DefaultHeight: 3, MinWidth: 2, MaxWidth: 12},
			{WidgetID: "audit-feed", Title: "Audit Feed", Module: "audit", DataSource: "/v2/audit", DefaultWidth: 4, DefaultHeight: 3, MinWidth: 2, MaxWidth: 12},
		},
		Layouts: []DashboardLayout{
			{
				LayoutID: "primary",
				Name:     "Primary",
				Columns:  12,
				Placements: []DashboardWidgetPlacement{
					{PlacementID: "placement-1", WidgetID: "ops-summary", Column: 0, Row: 0, Width: 6, Height: 2, Filters: []string{"team=platform"}},
					{PlacementID: "placement-2", WidgetID: "audit-feed", Column: 6, Row: 0, Width: 6, Height: 2, TitleOverride: "Live Audit"},
				},
			},
		},
		DocumentationComplete: true,
	}

	audit := AuditDashboardBuilder(dashboard)
	if !audit.ReleaseReady() {
		t.Fatalf("expected release-ready dashboard, got %+v", audit)
	}

	rendered := RenderDashboardBuilderReport(dashboard, audit)
	for _, fragment := range []string{
		"# Dashboard Builder",
		"- Release Ready: true",
		"- Duplicate Placement IDs: none",
		"- primary: name=Primary columns=12 placements=2",
		"- placement-2: widget=audit-feed title=Live Audit grid=(6,0) size=6x2 filters=none",
	} {
		if !strings.Contains(rendered, fragment) {
			t.Fatalf("expected %q in rendered report, got %s", fragment, rendered)
		}
	}

	outputDir := t.TempDir()
	path, err := WriteDashboardBuilderBundle(outputDir, dashboard, audit)
	if err != nil {
		t.Fatalf("write dashboard builder bundle: %v", err)
	}
	if path != filepath.Join(outputDir, "dashboard-builder.md") {
		t.Fatalf("unexpected dashboard bundle path: %s", path)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read dashboard builder bundle: %v", err)
	}
	if !strings.Contains(string(body), "team=platform") || !strings.Contains(string(body), "Live Audit") {
		t.Fatalf("unexpected dashboard builder bundle content: %s", string(body))
	}
}

func TestBuildEngineeringOverviewFromTasksAndEvents(t *testing.T) {
	base := time.Date(2026, 3, 18, 9, 0, 0, 0, time.UTC)
	tasks := []domain.Task{
		{
			ID:        "task-success",
			Title:     "Ship release",
			State:     domain.TaskSucceeded,
			CreatedAt: base,
			UpdatedAt: base.Add(30 * time.Minute),
			Metadata: map[string]string{
				"run_id":   "run-success",
				"summary":  "release shipped",
				"team":     "platform",
				"priority": "1",
			},
		},
		{
			ID:        "task-approval",
			Title:     "Approve rollout",
			State:     domain.TaskBlocked,
			CreatedAt: base,
			UpdatedAt: base.Add(90 * time.Minute),
			Metadata: map[string]string{
				"run_id":          "run-approval",
				"approval_status": "needs-approval",
				"blocked_reason":  "approval pending for prod rollout",
				"summary":         "awaiting prod approval",
				"team":            "operations",
			},
		},
		{
			ID:        "task-failed",
			Title:     "Fix security regression",
			State:     domain.TaskFailed,
			CreatedAt: base,
			UpdatedAt: base.Add(2 * time.Hour),
			Metadata: map[string]string{
				"run_id":         "run-failed",
				"failure_reason": "security review blocked deploy",
				"team":           "security",
			},
		},
	}
	events := []domain.Event{
		{ID: "evt-approval", Type: domain.EventRunAnnotated, TaskID: "task-approval", RunID: "run-approval", Timestamp: base.Add(90 * time.Minute), Payload: map[string]any{"reason": "approval pending for prod rollout"}},
		{ID: "evt-failed", Type: domain.EventTaskDeadLetter, TaskID: "task-failed", RunID: "run-failed", Timestamp: base.Add(2 * time.Hour), Payload: map[string]any{"reason": "security review blocked deploy"}},
	}

	overview := BuildEngineeringOverview("Engineering Pulse", "2026-W12", "engineering-manager", tasks, events, 60, 3, 5)
	if overview.Permissions.ViewerRole != "engineering-manager" || !overview.Permissions.CanView("funnel") || !overview.Permissions.CanView("activity") {
		t.Fatalf("unexpected permissions: %+v", overview.Permissions)
	}
	if len(overview.KPIs) != 4 {
		t.Fatalf("expected 4 KPIs, got %+v", overview.KPIs)
	}
	if overview.KPIs[0].Value != 33.3 {
		t.Fatalf("expected success rate 33.3, got %+v", overview.KPIs[0])
	}
	if overview.KPIs[1].Value != 1 || overview.KPIs[2].Value != 2 || overview.KPIs[3].Value != 80 {
		t.Fatalf("unexpected operational KPIs: %+v", overview.KPIs)
	}
	if len(overview.Funnel) != 4 || overview.Funnel[0].Count != 0 || overview.Funnel[2].Count != 1 || overview.Funnel[3].Count != 1 {
		t.Fatalf("unexpected funnel: %+v", overview.Funnel)
	}
	if len(overview.Blockers) != 2 || overview.Blockers[0].Owner != "operations" || overview.Blockers[1].Owner != "security" {
		t.Fatalf("unexpected blockers: %+v", overview.Blockers)
	}
	if overview.Blockers[1].Severity != "high" {
		t.Fatalf("expected failed task blocker severity high, got %+v", overview.Blockers[1])
	}
	if len(overview.Activities) != 3 || overview.Activities[0].TaskID != "task-failed" || overview.Activities[0].RunID != "run-failed" {
		t.Fatalf("unexpected activities: %+v", overview.Activities)
	}
}

func TestRenderAndWriteEngineeringOverviewBundle(t *testing.T) {
	overview := EngineeringOverview{
		Name:   "Engineering Pulse",
		Period: "2026-W12",
		Permissions: EngineeringOverviewPermission{
			ViewerRole:     "operations",
			AllowedModules: []string{"kpis", "funnel", "blockers", "activity"},
		},
		KPIs: []EngineeringOverviewKPI{
			{Name: "success-rate", Value: 95, Target: 90, Unit: "%", Direction: "up"},
			{Name: "sla-breaches", Value: 1, Target: 0, Direction: "down"},
		},
		Funnel: []EngineeringFunnelStage{
			{Name: "queued", Count: 2, Share: 20},
			{Name: "completed", Count: 8, Share: 80},
		},
		Blockers: []EngineeringOverviewBlocker{
			{Summary: "approval pending for prod rollout", AffectedRuns: 2, AffectedTasks: []string{"BIG-1", "BIG-2"}, Owner: "operations", Severity: "medium"},
		},
		Activities: []EngineeringActivity{
			{Timestamp: "2026-03-18T09:30:00Z", RunID: "run-1", TaskID: "BIG-1", Status: "blocked", Summary: "approval pending for prod rollout"},
		},
	}

	rendered := RenderEngineeringOverview(overview)
	for _, fragment := range []string{
		"# Engineering Overview",
		"- Viewer Role: operations",
		"- success-rate: value=95.0% target=90.0% healthy=true",
		"- completed: count=8 share=80.0%",
		"- approval pending for prod rollout: severity=medium owner=operations affected_runs=2 tasks=BIG-1, BIG-2",
		"- 2026-03-18T09:30:00Z: run-1 task=BIG-1 status=blocked summary=approval pending for prod rollout",
	} {
		if !strings.Contains(rendered, fragment) {
			t.Fatalf("expected %q in rendered overview, got %s", fragment, rendered)
		}
	}

	outputDir := t.TempDir()
	path, err := WriteEngineeringOverviewBundle(outputDir, overview)
	if err != nil {
		t.Fatalf("write engineering overview bundle: %v", err)
	}
	if path != filepath.Join(outputDir, "engineering-overview.md") {
		t.Fatalf("unexpected overview path: %s", path)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read engineering overview bundle: %v", err)
	}
	if !strings.Contains(string(body), "## Activity Modules") || !strings.Contains(string(body), "approval pending for prod rollout") {
		t.Fatalf("unexpected overview bundle content: %s", string(body))
	}
}

func TestBuildRenderAndWriteQueueControlCenterBundle(t *testing.T) {
	tasks := []domain.Task{
		{
			ID:               "task-queued",
			Priority:         0,
			State:            domain.TaskQueued,
			RiskLevel:        domain.RiskHigh,
			RequiredExecutor: domain.ExecutorRay,
			Metadata:         map[string]string{},
		},
		{
			ID:               "task-blocked",
			Priority:         1,
			State:            domain.TaskBlocked,
			RiskLevel:        domain.RiskMedium,
			RequiredExecutor: domain.ExecutorLocal,
			Metadata: map[string]string{
				"approval_status": "needs-approval",
			},
		},
		{
			ID:               "task-retrying",
			Priority:         2,
			State:            domain.TaskRetrying,
			RiskLevel:        domain.RiskLow,
			RequiredExecutor: domain.ExecutorKubernetes,
			Metadata:         map[string]string{},
		},
	}

	center := BuildQueueControlCenter(tasks)
	if center.QueueDepth != 3 || center.WaitingApprovalRuns != 1 {
		t.Fatalf("unexpected queue center counts: %+v", center)
	}
	if len(center.BlockedTasks) != 1 || center.BlockedTasks[0] != "task-blocked" {
		t.Fatalf("unexpected blocked tasks: %+v", center.BlockedTasks)
	}
	if strings.Join(center.QueuedTasks, ",") != "task-queued,task-retrying" {
		t.Fatalf("unexpected queued tasks: %+v", center.QueuedTasks)
	}
	if center.QueuedByPriority["P0"] != 1 || center.QueuedByPriority["P2"] != 1 {
		t.Fatalf("unexpected queued by priority: %+v", center.QueuedByPriority)
	}
	if center.QueuedByRisk["high"] != 1 || center.QueuedByRisk["low"] != 1 {
		t.Fatalf("unexpected queued by risk: %+v", center.QueuedByRisk)
	}
	if center.ExecutionMedia["ray"] != 1 || center.ExecutionMedia["kubernetes"] != 1 {
		t.Fatalf("unexpected execution media: %+v", center.ExecutionMedia)
	}
	if len(center.Actions["task-queued"]) == 0 || len(center.Actions["task-retrying"]) == 0 {
		t.Fatalf("expected actions for queued tasks, got %+v", center.Actions)
	}

	rendered := RenderQueueControlCenter(center)
	for _, fragment := range []string{
		"# Queue Control Center",
		"- Queue Depth: 3",
		"- Waiting Approval Runs: 1",
		"- ray: 1",
		"- task-blocked",
		"Retry [retry] state=disabled target=task-queued",
	} {
		if !strings.Contains(rendered, fragment) {
			t.Fatalf("expected %q in rendered queue center, got %s", fragment, rendered)
		}
	}

	outputDir := t.TempDir()
	path, err := WriteQueueControlCenterBundle(outputDir, center)
	if err != nil {
		t.Fatalf("write queue control center bundle: %v", err)
	}
	if path != filepath.Join(outputDir, "queue-control-center.md") {
		t.Fatalf("unexpected queue control center path: %s", path)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read queue control center bundle: %v", err)
	}
	if !strings.Contains(string(body), "## Actions") || !strings.Contains(string(body), "task-retrying") {
		t.Fatalf("unexpected queue control center content: %s", string(body))
	}
}

func TestBuildRepoCollaborationMetrics(t *testing.T) {
	runs := []map[string]any{
		{
			"run_id": "r1",
			"closeout": map[string]any{
				"run_commit_links":     []any{map[string]any{"role": "candidate"}},
				"accepted_commit_hash": "abc123",
			},
			"repo_discussion_posts":  3,
			"accepted_lineage_depth": 2,
		},
		{
			"run_id": "r2",
			"closeout": map[string]any{
				"run_commit_links":     []any{},
				"accepted_commit_hash": "",
			},
			"repo_discussion_posts":  1,
			"accepted_lineage_depth": 4,
		},
	}
	metrics := BuildRepoCollaborationMetrics(runs)
	if metrics["repo_link_coverage"] != 50.0 || metrics["accepted_commit_rate"] != 50.0 || metrics["discussion_density"] != 2.0 || metrics["accepted_lineage_depth_avg"] != 3.0 {
		t.Fatalf("unexpected repo collaboration metrics: %+v", metrics)
	}
}

func TestBuildPolicyPromptVersionCenterSummarizesRevisionDiffs(t *testing.T) {
	center := BuildPolicyPromptVersionCenter(
		"Policy/Prompt Version Center",
		time.Date(2026, 3, 18, 15, 50, 0, 0, time.UTC),
		[]VersionedArtifact{
			{
				ArtifactType: "policy",
				ArtifactID:   "approval-gate",
				Version:      "v1",
				UpdatedAt:    "2026-03-18T14:00:00Z",
				Author:       "alice",
				Summary:      "initial gate",
				Content:      "allow=team-a\nthreshold=2\n",
			},
			{
				ArtifactType: "policy",
				ArtifactID:   "approval-gate",
				Version:      "v2",
				UpdatedAt:    "2026-03-18T15:00:00Z",
				Author:       "bob",
				Summary:      "tighten threshold",
				Content:      "allow=team-a\nthreshold=3\nnotify=ops\n",
				ChangeTicket: "BIG-GOM-304",
			},
			{
				ArtifactType: "prompt",
				ArtifactID:   "rollout-review",
				Version:      "v1",
				UpdatedAt:    "2026-03-18T13:00:00Z",
				Author:       "carol",
				Summary:      "seed prompt",
				Content:      "Summarize rollout blockers.\n",
			},
		},
		6,
	)
	if center.ArtifactCount() != 2 || center.RollbackReadyCount() != 1 {
		t.Fatalf("unexpected center counts: %+v", center)
	}
	history := center.Histories[0]
	if history.ArtifactType != "policy" || history.ArtifactID != "approval-gate" || history.CurrentVersion != "v2" {
		t.Fatalf("unexpected sorted history: %+v", history)
	}
	if !history.RollbackReady || history.RollbackVersion != "v1" || history.ChangeSummary == nil || !history.ChangeSummary.HasChanges() {
		t.Fatalf("expected rollback-ready diff summary, got %+v", history)
	}
	if history.ChangeSummary.Additions != 2 || history.ChangeSummary.Deletions != 1 {
		t.Fatalf("unexpected change summary counts: %+v", history.ChangeSummary)
	}
	if len(history.ChangeSummary.Preview) == 0 || history.ChangeSummary.Preview[0] != "--- v1" {
		t.Fatalf("unexpected diff preview: %+v", history.ChangeSummary.Preview)
	}
}

func TestRenderAndWritePolicyPromptVersionCenterBundle(t *testing.T) {
	center := PolicyPromptVersionCenter{
		Name:        "Policy/Prompt Version Center",
		GeneratedAt: "2026-03-18T15:50:00Z",
		Histories: []VersionedArtifactHistory{
			{
				ArtifactType:     "policy",
				ArtifactID:       "approval-gate",
				CurrentVersion:   "v2",
				CurrentUpdatedAt: "2026-03-18T15:00:00Z",
				CurrentAuthor:    "bob",
				CurrentSummary:   "tighten threshold",
				RevisionCount:    2,
				Revisions: []VersionedArtifact{
					{Version: "v2", UpdatedAt: "2026-03-18T15:00:00Z", Author: "bob", Summary: "tighten threshold", ChangeTicket: "BIG-GOM-304"},
					{Version: "v1", UpdatedAt: "2026-03-18T14:00:00Z", Author: "alice", Summary: "initial gate"},
				},
				RollbackVersion: "v1",
				RollbackReady:   true,
				ChangeSummary: &VersionChangeSummary{
					FromVersion:  "v1",
					ToVersion:    "v2",
					Additions:    2,
					Deletions:    1,
					ChangedLines: 3,
					Preview:      []string{"--- v1", "+++ v2", "-threshold=2", "+threshold=3"},
				},
			},
		},
	}
	rendered := RenderPolicyPromptVersionCenter(center)
	for _, fragment := range []string{
		"# Policy/Prompt Version Center",
		"- Versioned Artifacts: 1",
		"- Rollback Ready Artifacts: 1",
		"### policy / approval-gate",
		"- Diff Summary: 2 additions, 1 deletions",
		"```diff",
		"+threshold=3",
	} {
		if !strings.Contains(rendered, fragment) {
			t.Fatalf("expected %q in rendered version center, got %s", fragment, rendered)
		}
	}
	outputDir := t.TempDir()
	path, err := WritePolicyPromptVersionCenterBundle(outputDir, center)
	if err != nil {
		t.Fatalf("write version center bundle: %v", err)
	}
	if path != filepath.Join(outputDir, "policy-prompt-version-center.md") {
		t.Fatalf("unexpected version center path: %s", path)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read version center bundle: %v", err)
	}
	if !strings.Contains(string(body), "approval-gate") || !strings.Contains(string(body), "BIG-GOM-304") {
		t.Fatalf("unexpected version center bundle content: %s", string(body))
	}
}

func TestWriteWeeklyOperationsBundleWithVersionCenter(t *testing.T) {
	rootDir := t.TempDir()
	start := time.Date(2026, 3, 17, 0, 0, 0, 0, time.UTC)
	end := start.Add(7 * 24 * time.Hour)
	weekly := Weekly{
		WeekStart:  start,
		WeekEnd:    end,
		Highlights: []string{"Completed 1 / 1 runs this week."},
		Actions:    []string{"No urgent actions detected; maintain current operating cadence."},
	}
	center := BuildPolicyPromptVersionCenter("Policy Center", time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC), []VersionedArtifact{
		{
			ArtifactType: "prompt",
			ArtifactID:   "triage-summary",
			Version:      "v1",
			UpdatedAt:    "2026-03-17T10:00:00Z",
			Author:       "alice",
			Summary:      "initial prompt",
			Content:      "summarize blockers\n",
		},
		{
			ArtifactType: "prompt",
			ArtifactID:   "triage-summary",
			Version:      "v2",
			UpdatedAt:    "2026-03-18T10:00:00Z",
			Author:       "carol",
			Summary:      "latest prompt",
			Content:      "summarize blockers\ninclude owners\n",
			ChangeTicket: "BIG-GOM-304",
		},
	}, 8)

	artifacts, err := WriteWeeklyOperationsBundleWithVersionCenter(rootDir, weekly, nil, &center)
	if err != nil {
		t.Fatalf("write weekly bundle with version center: %v", err)
	}
	if artifacts.VersionCenterPath != filepath.Join(rootDir, "policy-prompt-version-center.md") {
		t.Fatalf("unexpected version center path: %+v", artifacts)
	}
	body, err := os.ReadFile(artifacts.VersionCenterPath)
	if err != nil {
		t.Fatalf("read version center bundle: %v", err)
	}
	if !strings.Contains(string(body), "triage-summary") || !strings.Contains(string(body), "latest prompt") {
		t.Fatalf("unexpected version center weekly bundle content: %s", string(body))
	}
}

func TestWriteWeeklyOperationsBundleWithCenters(t *testing.T) {
	rootDir := t.TempDir()
	start := time.Date(2026, 3, 17, 0, 0, 0, 0, time.UTC)
	end := start.Add(7 * 24 * time.Hour)
	weekly := Weekly{
		WeekStart:  start,
		WeekEnd:    end,
		Highlights: []string{"Completed 1 / 1 runs this week."},
		Actions:    []string{"No urgent actions detected; maintain current operating cadence."},
	}
	queueCenter := BuildQueueControlCenter([]domain.Task{
		{ID: "task-queued", Priority: 1, State: domain.TaskQueued, RiskLevel: domain.RiskMedium, RequiredExecutor: domain.ExecutorLocal},
	})
	versionCenter := BuildPolicyPromptVersionCenter("Policy Center", time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC), []VersionedArtifact{
		{
			ArtifactType: "prompt",
			ArtifactID:   "triage-summary",
			Version:      "v1",
			UpdatedAt:    "2026-03-18T10:00:00Z",
			Author:       "carol",
			Summary:      "latest prompt",
			Content:      "summarize blockers\ninclude owners\n",
			ChangeTicket: "BIG-GOM-304",
		},
	}, 8)

	regressionCenter := regression.Center{
		Summary: regression.Summary{
			TotalRegressions: 1,
			AffectedTasks:    1,
			TopSource:        "ops",
		},
		Findings: []regression.Finding{
			{TaskID: "task-queued", Severity: "medium", RegressionCount: 1, Summary: "retry loop detected"},
		},
	}

	artifacts, err := WriteWeeklyOperationsBundleWithCenters(rootDir, weekly, nil, "Regression Analysis Center", &regressionCenter, &queueCenter, &versionCenter)
	if err != nil {
		t.Fatalf("write weekly bundle with centers: %v", err)
	}
	if artifacts.RegressionCenterPath != filepath.Join(rootDir, "regression-center.md") {
		t.Fatalf("unexpected regression center path: %+v", artifacts)
	}
	if artifacts.QueueControlPath != filepath.Join(rootDir, "queue-control-center.md") {
		t.Fatalf("unexpected queue control path: %+v", artifacts)
	}
	if artifacts.VersionCenterPath != filepath.Join(rootDir, "policy-prompt-version-center.md") {
		t.Fatalf("unexpected version center path: %+v", artifacts)
	}
	queueBody, err := os.ReadFile(artifacts.QueueControlPath)
	if err != nil {
		t.Fatalf("read queue control bundle: %v", err)
	}
	if !strings.Contains(string(queueBody), "task-queued") {
		t.Fatalf("unexpected queue control bundle content: %s", string(queueBody))
	}
	regressionBody, err := os.ReadFile(artifacts.RegressionCenterPath)
	if err != nil {
		t.Fatalf("read regression center bundle: %v", err)
	}
	if !strings.Contains(string(regressionBody), "retry loop detected") {
		t.Fatalf("unexpected regression center bundle content: %s", string(regressionBody))
	}
	versionBody, err := os.ReadFile(artifacts.VersionCenterPath)
	if err != nil {
		t.Fatalf("read version center bundle: %v", err)
	}
	if !strings.Contains(string(versionBody), "triage-summary") {
		t.Fatalf("unexpected version center bundle content: %s", string(versionBody))
	}
}

func TestRenderAndWriteRegressionCenterBundle(t *testing.T) {
	center := regression.Center{
		Summary: regression.Summary{
			TotalRegressions:    3,
			AffectedTasks:       2,
			CriticalRegressions: 1,
			ReworkEvents:        2,
			TopSource:           "security review blocked deploy",
			TopWorkflow:         "deploy",
		},
		WorkflowBreakdown: []regression.Breakdown{
			{Key: "deploy", TotalRegressions: 3, AffectedTasks: 2, CriticalRegressions: 1, ReworkEvents: 2},
		},
		Hotspots: []regression.Hotspot{
			{Dimension: "workflow", Key: "deploy", TotalRegressions: 3, CriticalRegressions: 1, ReworkEvents: 2},
		},
		Findings: []regression.Finding{
			{TaskID: "BIG-401", Workflow: "deploy", Team: "platform", Severity: "critical", RegressionCount: 2, ReworkEvents: 1, Summary: "security review blocked deploy"},
			{TaskID: "BIG-402", Workflow: "deploy", Team: "platform", Severity: "medium", RegressionCount: 1, ReworkEvents: 1, Summary: "rollback playbook drift"},
		},
	}

	rendered := RenderRegressionCenter("Regression Console", center)
	for _, fragment := range []string{
		"# Regression Analysis Center",
		"- Name: Regression Console",
		"- Regressions: 2",
		"- Top Workflow: deploy",
		"- BIG-401: severity=critical regressions=2 rework=1 workflow=deploy team=platform summary=security review blocked deploy",
		"- workflow/deploy: regressions=3 critical=1 rework=2",
	} {
		if !strings.Contains(rendered, fragment) {
			t.Fatalf("expected %q in rendered regression center, got %s", fragment, rendered)
		}
	}

	outputDir := t.TempDir()
	path, err := WriteRegressionCenterBundle(outputDir, "Regression Console", center)
	if err != nil {
		t.Fatalf("write regression center bundle: %v", err)
	}
	if path != filepath.Join(outputDir, "regression-center.md") {
		t.Fatalf("unexpected regression center path: %s", path)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read regression center bundle: %v", err)
	}
	if !strings.Contains(string(body), "## Findings") || !strings.Contains(string(body), "rollback playbook drift") {
		t.Fatalf("unexpected regression center bundle content: %s", string(body))
	}
}
