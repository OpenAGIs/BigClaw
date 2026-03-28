package reporting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/domain"
)

func strPtr(value string) *string { return &value }
func boolPtr(value bool) *bool    { return &value }
func intPtr(value int) *int       { return &value }

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
	if enabled.State() != "enabled" {
		t.Fatalf("expected enabled state, got %q", enabled.State())
	}
	if disabled.State() != "disabled" {
		t.Fatalf("expected disabled state, got %q", disabled.State())
	}
}

func TestReportStudioRendersNarrativeSectionsAndBundle(t *testing.T) {
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
			{
				Heading:  "What needs attention",
				Body:     "Security takeover requests still cluster around data-export tasks and need a dedicated reviewer window.",
				Evidence: []string{"takeover-queue"},
				Callouts: []string{"Review staffing before Friday close"},
			},
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
		t.Fatalf("expected ready/publish studio")
	}
	for _, path := range []string{artifacts.MarkdownPath, artifacts.HTMLPath, artifacts.TextPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected artifact %s: %v", path, err)
		}
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
		Sections: []NarrativeSection{{Heading: "Open risks", Body: ""}},
	}
	if studio.Ready() {
		t.Fatalf("expected draft studio to be unready")
	}
	if studio.Recommendation() != "draft" {
		t.Fatalf("expected draft recommendation, got %q", studio.Recommendation())
	}
}

func TestRenderPilotScorecardIncludesROIAndRecommendation(t *testing.T) {
	benchmarkScore := 96
	benchmarkPassed := true
	scorecard := PilotScorecard{
		IssueID:            "OPE-60",
		Customer:           "Design Partner A",
		Period:             "2026-Q2",
		Metrics:            []PilotMetric{{Name: "Automation coverage", Baseline: 35, Current: 82, Target: 80, Unit: "%", HigherIsBetter: true}, {Name: "Manual review time", Baseline: 12, Current: 4, Target: 5, Unit: "h", HigherIsBetter: false}},
		MonthlyBenefit:     12000,
		MonthlyCost:        2500,
		ImplementationCost: 18000,
		BenchmarkScore:     &benchmarkScore,
		BenchmarkPassed:    &benchmarkPassed,
	}

	content := RenderPilotScorecard(scorecard)
	if scorecard.MetricsMet() != 2 || scorecard.Recommendation() != "go" {
		t.Fatalf("unexpected scorecard state: %+v", scorecard)
	}
	if payback := scorecard.PaybackMonths(); payback == nil || *payback != 1.9 {
		t.Fatalf("unexpected payback: %v", payback)
	}
	for _, fragment := range []string{"Annualized ROI: 200.0%", "Recommendation: go", "Benchmark Score: 96", "Automation coverage"} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("expected %q in content: %s", fragment, content)
		}
	}
}

func TestPilotScorecardReturnsHoldWhenNetValueNegative(t *testing.T) {
	benchmarkPassed := false
	scorecard := PilotScorecard{
		IssueID:            "OPE-60",
		Customer:           "Design Partner B",
		Period:             "2026-Q2",
		Metrics:            []PilotMetric{{Name: "Backlog aging", Baseline: 5, Current: 7, Target: 4, Unit: "d", HigherIsBetter: false}},
		MonthlyBenefit:     1000,
		MonthlyCost:        3000,
		ImplementationCost: 12000,
		BenchmarkPassed:    &benchmarkPassed,
	}
	if scorecard.MonthlyNetValue() != -2000 {
		t.Fatalf("unexpected monthly net value: %v", scorecard.MonthlyNetValue())
	}
	if scorecard.PaybackMonths() != nil {
		t.Fatalf("expected no payback")
	}
	if scorecard.Recommendation() != "hold" {
		t.Fatalf("unexpected recommendation: %s", scorecard.Recommendation())
	}
}

func TestIssueClosureRequiresNonEmptyValidationReport(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "validation.md")
	decision := EvaluateIssueClosure("BIG-602", reportPath, true, nil, nil)
	if decision.Allowed || decision.Reason != "validation report required before closing issue" || ValidationReportExists(reportPath) {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestIssueClosureBlocksFailedValidationReport(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "validation.md")
	if err := WriteReport(reportPath, "# Validation\n\nfailed"); err != nil {
		t.Fatalf("write report: %v", err)
	}
	decision := EvaluateIssueClosure("BIG-602", reportPath, false, nil, nil)
	if decision.Allowed || decision.Reason != "validation failed; issue must remain open" || !ValidationReportExists(reportPath) {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestIssueClosureAllowsCompletedValidationReport(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "validation.md")
	if err := WriteReport(reportPath, RenderIssueValidationReport("BIG-602", "v0.1", "sandbox", "pass")); err != nil {
		t.Fatalf("write report: %v", err)
	}
	decision := EvaluateIssueClosure("BIG-602", reportPath, true, nil, nil)
	if !decision.Allowed || decision.Reason != "validation report and launch checklist requirements satisfied; issue can be closed" || decision.ReportPath != reportPath {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestEnforceValidationReportPolicyBlocksMissingArtifacts(t *testing.T) {
	decision := EnforceValidationReportPolicy([]string{"task-run", "replay"})
	if decision.AllowedToClose || decision.Status != "blocked" || len(decision.MissingReports) != 1 || decision.MissingReports[0] != "benchmark-suite" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestEnforceValidationReportPolicyAllowsCompleteArtifacts(t *testing.T) {
	decision := EnforceValidationReportPolicy([]string{"task-run", "replay", "benchmark-suite"})
	if !decision.AllowedToClose || decision.Status != "ready" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestBenchmarkRunnerScoresAndReplaysCase(t *testing.T) {
	runner := BenchmarkRunner{StorageDir: t.TempDir()}
	caseDef := BenchmarkCase{
		CaseID:           "browser-low-risk",
		Task:             domain.Task{ID: "BIG-601", Source: "linear", Title: "Run browser benchmark", Description: "validate routing", RiskLevel: domain.RiskLow, RequiredTools: []string{"browser"}},
		ExpectedMedium:   strPtr("browser"),
		ExpectedApproved: boolPtr(true),
		ExpectedStatus:   strPtr("approved"),
		RequireReport:    true,
	}

	result, err := runner.RunCase(caseDef)
	if err != nil {
		t.Fatalf("run case: %v", err)
	}
	if result.Score != 100 || !result.Passed || !result.Replay.Matched {
		t.Fatalf("unexpected result: %+v", result)
	}
	for _, path := range []string{
		filepath.Join(runner.StorageDir, "browser-low-risk", "task-run.md"),
		filepath.Join(runner.StorageDir, "benchmark-browser-low-risk", "replay.html"),
		filepath.Join(runner.StorageDir, "browser-low-risk", "run-detail.html"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected artifact %s: %v", path, err)
		}
	}
	if result.DetailPagePath != filepath.Join(runner.StorageDir, "browser-low-risk", "run-detail.html") {
		t.Fatalf("unexpected detail path: %s", result.DetailPagePath)
	}
}

func TestBenchmarkRunnerDetectsFailedExpectation(t *testing.T) {
	runner := BenchmarkRunner{StorageDir: t.TempDir()}
	caseDef := BenchmarkCase{
		CaseID:           "high-risk-gate",
		Task:             domain.Task{ID: "BIG-601-risk", Source: "jira", Title: "Prod change benchmark", Description: "must stop for approval", RiskLevel: domain.RiskHigh},
		ExpectedMedium:   strPtr("docker"),
		ExpectedApproved: boolPtr(false),
		ExpectedStatus:   strPtr("needs-approval"),
	}
	result, err := runner.RunCase(caseDef)
	if err != nil {
		t.Fatalf("run case: %v", err)
	}
	if result.Passed || result.Score != 60 {
		t.Fatalf("unexpected result: %+v", result)
	}
	found := false
	for _, item := range result.Criteria {
		if item.Name == "decision-medium" && !item.Passed {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected failed decision-medium criterion: %+v", result.Criteria)
	}
}

func TestReplayOutcomeReportsMismatch(t *testing.T) {
	runner := BenchmarkRunner{StorageDir: t.TempDir()}
	replayRecord := ReplayRecord{
		Task:     domain.Task{ID: "BIG-601-replay", Source: "github", Title: "Replay browser route", Description: "compare deterministic scheduler behavior", RequiredTools: []string{"browser"}},
		RunID:    "run-1",
		Medium:   "docker",
		Approved: true,
		Status:   "approved",
	}
	outcome, err := runner.Replay(replayRecord)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if outcome.Matched || len(outcome.Mismatches) != 1 || outcome.Mismatches[0] != "medium expected docker got browser" || outcome.ReportPath == "" {
		t.Fatalf("unexpected outcome: %+v", outcome)
	}
	if _, err := os.Stat(outcome.ReportPath); err != nil {
		t.Fatalf("expected replay report: %v", err)
	}
}

func TestSuiteComparisonAndReport(t *testing.T) {
	runner := BenchmarkRunner{StorageDir: t.TempDir()}
	improved, err := runner.RunSuite([]BenchmarkCase{{
		CaseID:           "browser-low-risk",
		Task:             domain.Task{ID: "BIG-601-v2", Source: "linear", Title: "Run browser benchmark", Description: "validate routing", RequiredTools: []string{"browser"}},
		ExpectedMedium:   strPtr("browser"),
		ExpectedApproved: boolPtr(true),
		ExpectedStatus:   strPtr("approved"),
	}}, "v0.2")
	if err != nil {
		t.Fatalf("run suite: %v", err)
	}
	baseline := BenchmarkSuiteResult{Results: nil, Version: "v0.1"}
	comparison := improved.Compare(baseline)
	report := RenderBenchmarkSuiteReport(improved, &baseline)
	if len(comparison) != 1 || comparison[0].Delta != 100 || improved.Score() != 100 {
		t.Fatalf("unexpected comparison: %+v", comparison)
	}
	for _, fragment := range []string{"Version: v0.2", "Baseline Version: v0.1", "Score Delta: 100"} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report: %s", fragment, report)
		}
	}
}

func TestRenderReplayDetailPageListsMismatches(t *testing.T) {
	task := domain.Task{ID: "BIG-804", Source: "linear", Title: "Replay detail"}
	expected := ReplayRecord{Task: task, RunID: "run-1", Medium: "docker", Approved: true, Status: "approved"}
	observed := ReplayRecord{Task: task, RunID: "run-1", Medium: "browser", Approved: false, Status: "needs-approval"}
	page := RenderReplayDetailPage(expected, observed, []string{"medium expected docker got browser", "approved expected true got false"})
	for _, fragment := range []string{"Replay Detail", "Timeline / Log Sync", "Split View", "Reports", "medium expected docker got browser", "needs-approval"} {
		if !strings.Contains(page, fragment) {
			t.Fatalf("expected %q in page: %s", fragment, page)
		}
	}
}

func TestRenderRunReplayIndexPageLinksOutputs(t *testing.T) {
	runner := BenchmarkRunner{StorageDir: t.TempDir()}
	result, err := runner.RunCase(BenchmarkCase{
		CaseID:           "big-804-index",
		Task:             domain.Task{ID: "BIG-804", Source: "linear", Title: "Run detail index", Description: "single landing page", RequiredTools: []string{"browser"}},
		ExpectedMedium:   strPtr("browser"),
		ExpectedApproved: boolPtr(true),
		ExpectedStatus:   strPtr("approved"),
		RequireReport:    true,
	})
	if err != nil {
		t.Fatalf("run case: %v", err)
	}
	body, err := os.ReadFile(result.DetailPagePath)
	if err != nil {
		t.Fatalf("read detail page: %v", err)
	}
	page := string(body)
	for _, fragment := range []string{"Run Detail Index", "Timeline / Log Sync", "Acceptance", "Reports", "task-run.md", "replay.html", "decision-medium"} {
		if !strings.Contains(page, fragment) {
			t.Fatalf("expected %q in page: %s", fragment, page)
		}
	}
}

func TestRenderRunReplayIndexPageWithoutReportPath(t *testing.T) {
	task := domain.Task{ID: "BIG-804", Source: "linear", Title: "Run detail index"}
	replay := ReplayOutcome{
		Matched:      true,
		ReplayRecord: ReplayRecord{Task: task, RunID: "run-1", Medium: "docker", Approved: true, Status: "approved"},
	}
	record := BenchmarkExecutionRecord{
		Decision: BenchmarkExecutionDecision{Medium: "docker", Approved: true},
		Run:      BenchmarkExecutionRun{TaskID: task.ID, Status: "approved"},
	}
	page := RenderRunReplayIndexPage("big-804-index", record, replay, nil)
	if !strings.Contains(page, "n/a") || !strings.Contains(page, "Replay") {
		t.Fatalf("unexpected page: %s", page)
	}
}
