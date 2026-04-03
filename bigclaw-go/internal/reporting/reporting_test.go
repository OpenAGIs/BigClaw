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

func TestRenderPilotPortfolioReportSummarizesCommercialReadiness(t *testing.T) {
	benchmarkA := 97
	benchmarkB := 88
	passed := true
	portfolio := PilotPortfolio{
		Name:   "Design Partners",
		Period: "2026-H1",
		Scorecards: []PilotScorecard{
			{
				IssueID:  "OPE-60",
				Customer: "Partner A",
				Period:   "2026-Q2",
				Metrics: []PilotMetric{
					{Name: "Coverage", Baseline: 40, Current: 85, Target: 80, Unit: "%", HigherIsBetter: true},
				},
				MonthlyBenefit:     15000,
				MonthlyCost:        3000,
				ImplementationCost: 18000,
				BenchmarkScore:     &benchmarkA,
				BenchmarkPassed:    &passed,
			},
			{
				IssueID:  "OPE-61",
				Customer: "Partner B",
				Period:   "2026-Q2",
				Metrics: []PilotMetric{
					{Name: "Cycle time", Baseline: 12, Current: 7, Target: 5, Unit: "h", HigherIsBetter: false},
				},
				MonthlyBenefit:     9000,
				MonthlyCost:        2500,
				ImplementationCost: 12000,
				BenchmarkScore:     &benchmarkB,
				BenchmarkPassed:    &passed,
			},
		},
	}

	content := RenderPilotPortfolioReport(portfolio)

	if portfolio.TotalMonthlyNetValue() != 18500 {
		t.Fatalf("unexpected monthly net value: %+v", portfolio)
	}
	if portfolio.AverageROI() != 195.2 {
		t.Fatalf("unexpected average ROI: %+v", portfolio)
	}
	if !reflect.DeepEqual(portfolio.RecommendationCounts(), map[string]int{"go": 1, "iterate": 1, "hold": 0}) {
		t.Fatalf("unexpected recommendation counts: %+v", portfolio.RecommendationCounts())
	}
	if portfolio.Recommendation() != "continue" {
		t.Fatalf("unexpected portfolio recommendation: %s", portfolio.Recommendation())
	}
	for _, fragment := range []string{
		"Recommendation Mix: go=1 iterate=1 hold=0",
		"Partner A: recommendation=go",
		"Partner B: recommendation=iterate",
	} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("expected %q in pilot portfolio report, got %s", fragment, content)
		}
	}
}

func TestRenderPilotScorecardIncludesROIAndRecommendation(t *testing.T) {
	benchmark := 96
	passed := true
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
		BenchmarkScore:     &benchmark,
		BenchmarkPassed:    &passed,
	}

	content := RenderPilotScorecard(scorecard)

	if scorecard.MetricsMet() != 2 {
		t.Fatalf("unexpected metrics met: %+v", scorecard)
	}
	if scorecard.Recommendation() != "go" {
		t.Fatalf("unexpected recommendation: %s", scorecard.Recommendation())
	}
	if payback := scorecard.PaybackMonths(); payback == nil || *payback != 1.9 {
		t.Fatalf("unexpected payback months: %+v", payback)
	}
	for _, fragment := range []string{
		"Annualized ROI: 200.0%",
		"Recommendation: go",
		"Benchmark Score: 96",
		"Automation coverage",
	} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("expected %q in pilot scorecard report, got %s", fragment, content)
		}
	}
}

func TestPilotScorecardReturnsHoldWhenValueIsNegative(t *testing.T) {
	passed := false
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
		BenchmarkPassed:    &passed,
	}
	if scorecard.MonthlyNetValue() != -2000 {
		t.Fatalf("unexpected monthly net value: %+v", scorecard)
	}
	if scorecard.PaybackMonths() != nil {
		t.Fatalf("expected nil payback months: %+v", scorecard.PaybackMonths())
	}
	if scorecard.Recommendation() != "hold" {
		t.Fatalf("unexpected recommendation: %s", scorecard.Recommendation())
	}
}

func TestLaunchChecklistTracksDocumentationAvailability(t *testing.T) {
	runbook := filepath.Join(t.TempDir(), "runbook.md")
	faq := filepath.Join(t.TempDir(), "faq.md")
	if err := WriteReport(runbook, "# Runbook\n\nready"); err != nil {
		t.Fatalf("write runbook: %v", err)
	}

	checklist := BuildLaunchChecklist(
		"BIG-1003",
		[]DocumentationArtifact{
			{Name: "runbook", Path: runbook},
			{Name: "faq", Path: faq},
		},
		[]LaunchChecklistItem{
			{Name: "Operations handoff", Evidence: []string{"runbook"}},
			{Name: "Support handoff", Evidence: []string{"faq"}},
		},
	)

	report := RenderLaunchChecklistReport(checklist)

	if !reflect.DeepEqual(checklist.DocumentationStatus(), map[string]bool{"runbook": true, "faq": false}) {
		t.Fatalf("unexpected documentation status: %+v", checklist.DocumentationStatus())
	}
	if checklist.CompletedItems() != 1 {
		t.Fatalf("unexpected completed items: %d", checklist.CompletedItems())
	}
	if !reflect.DeepEqual(checklist.MissingDocumentation(), []string{"faq"}) {
		t.Fatalf("unexpected missing documentation: %+v", checklist.MissingDocumentation())
	}
	if checklist.Ready() {
		t.Fatalf("expected checklist to be incomplete")
	}
	for _, fragment := range []string{
		"runbook: available=true",
		"faq: available=false",
		"Support handoff: completed=false evidence=faq",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in launch checklist report, got %s", fragment, report)
		}
	}
}

func TestFinalDeliveryChecklistTracksRequiredAndRecommendedArtifacts(t *testing.T) {
	root := t.TempDir()
	validationBundle := filepath.Join(root, "validation-bundle.md")
	releaseNotes := filepath.Join(root, "release-notes.md")
	if err := WriteReport(validationBundle, "# Validation Bundle\n\nready"); err != nil {
		t.Fatalf("write validation bundle: %v", err)
	}

	checklist := BuildFinalDeliveryChecklist(
		"BIG-4702",
		[]DocumentationArtifact{
			{Name: "validation-bundle", Path: validationBundle},
			{Name: "release-notes", Path: releaseNotes},
		},
		[]DocumentationArtifact{
			{Name: "runbook", Path: filepath.Join(root, "runbook.md")},
			{Name: "faq", Path: filepath.Join(root, "faq.md")},
		},
	)

	report := RenderFinalDeliveryChecklistReport(checklist)

	if !reflect.DeepEqual(checklist.RequiredOutputStatus(), map[string]bool{"validation-bundle": true, "release-notes": false}) {
		t.Fatalf("unexpected required output status: %+v", checklist.RequiredOutputStatus())
	}
	if !reflect.DeepEqual(checklist.RecommendedDocumentationStatus(), map[string]bool{"runbook": false, "faq": false}) {
		t.Fatalf("unexpected recommended documentation status: %+v", checklist.RecommendedDocumentationStatus())
	}
	if checklist.GeneratedRequiredOutputs() != 1 || checklist.GeneratedRecommendedDocumentation() != 0 {
		t.Fatalf("unexpected generated counts: required=%d recommended=%d", checklist.GeneratedRequiredOutputs(), checklist.GeneratedRecommendedDocumentation())
	}
	if !reflect.DeepEqual(checklist.MissingRequiredOutputs(), []string{"release-notes"}) {
		t.Fatalf("unexpected missing required outputs: %+v", checklist.MissingRequiredOutputs())
	}
	if !reflect.DeepEqual(checklist.MissingRecommendedDocumentation(), []string{"runbook", "faq"}) {
		t.Fatalf("unexpected missing recommended documentation: %+v", checklist.MissingRecommendedDocumentation())
	}
	if checklist.Ready() {
		t.Fatalf("expected final delivery checklist to be incomplete")
	}
	for _, fragment := range []string{
		"Required Outputs Generated: 1/2",
		"Recommended Docs Generated: 0/2",
		"release-notes: available=false",
		"runbook: available=false",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in final delivery checklist report, got %s", fragment, report)
		}
	}
}

func TestEvaluateIssueClosureValidatesReportAndChecklistRequirements(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, "validation.md")

	decision := EvaluateIssueClosure("BIG-602", reportPath, true, nil, nil)
	if decision.Allowed {
		t.Fatalf("expected missing report to block issue closure")
	}
	if decision.Reason != "validation report required before closing issue" {
		t.Fatalf("unexpected reason: %s", decision.Reason)
	}
	if ValidationReportExists(reportPath) {
		t.Fatalf("expected missing report to be unavailable")
	}

	if err := WriteReport(reportPath, "# Validation\n\nfailed"); err != nil {
		t.Fatalf("write failed report: %v", err)
	}
	decision = EvaluateIssueClosure("BIG-602", reportPath, false, nil, nil)
	if decision.Allowed {
		t.Fatalf("expected failed validation to block issue closure")
	}
	if decision.Reason != "validation failed; issue must remain open" {
		t.Fatalf("unexpected failed-validation reason: %s", decision.Reason)
	}
	if !ValidationReportExists(reportPath) {
		t.Fatalf("expected written report to exist")
	}

	if err := WriteReport(reportPath, RenderIssueValidationReport("BIG-602", "v0.1", "sandbox", "pass")); err != nil {
		t.Fatalf("write passing report: %v", err)
	}
	decision = EvaluateIssueClosure("BIG-602", reportPath, true, nil, nil)
	if !decision.Allowed {
		t.Fatalf("expected completed validation report to allow closure")
	}
	if decision.Reason != "validation report and launch checklist requirements satisfied; issue can be closed" {
		t.Fatalf("unexpected ready reason: %s", decision.Reason)
	}
	if decision.ReportPath != reportPath {
		t.Fatalf("unexpected report path: %s", decision.ReportPath)
	}
}

func TestEvaluateIssueClosureBlocksIncompleteLaunchChecklist(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, "validation.md")
	runbook := filepath.Join(root, "runbook.md")
	if err := WriteReport(reportPath, RenderIssueValidationReport("BIG-1003", "v0.2", "staging", "pass")); err != nil {
		t.Fatalf("write validation report: %v", err)
	}
	if err := WriteReport(runbook, "# Runbook\n\nready"); err != nil {
		t.Fatalf("write runbook: %v", err)
	}

	checklist := BuildLaunchChecklist(
		"BIG-1003",
		[]DocumentationArtifact{
			{Name: "runbook", Path: runbook},
			{Name: "launch-faq", Path: filepath.Join(root, "launch-faq.md")},
		},
		[]LaunchChecklistItem{{Name: "Launch comms", Evidence: []string{"runbook", "launch-faq"}}},
	)

	decision := EvaluateIssueClosure("BIG-1003", reportPath, true, &checklist, nil)
	if decision.Allowed {
		t.Fatalf("expected incomplete launch checklist to block issue closure")
	}
	if decision.Reason != "launch checklist incomplete; linked documentation missing or empty" {
		t.Fatalf("unexpected reason: %s", decision.Reason)
	}
}

func TestEvaluateIssueClosureHandlesFinalDeliveryAndReadyLaunchChecklist(t *testing.T) {
	root := t.TempDir()

	finalReportPath := filepath.Join(root, "final-validation.md")
	if err := WriteReport(finalReportPath, RenderIssueValidationReport("BIG-4702", "v0.3", "staging", "pass")); err != nil {
		t.Fatalf("write final validation report: %v", err)
	}
	finalChecklist := BuildFinalDeliveryChecklist(
		"BIG-4702",
		[]DocumentationArtifact{
			{Name: "validation-bundle", Path: filepath.Join(root, "validation-bundle.md")},
		},
		[]DocumentationArtifact{
			{Name: "runbook", Path: filepath.Join(root, "runbook.md")},
		},
	)
	decision := EvaluateIssueClosure("BIG-4702", finalReportPath, true, nil, &finalChecklist)
	if decision.Allowed {
		t.Fatalf("expected missing required final outputs to block issue closure")
	}
	if decision.Reason != "final delivery checklist incomplete; required outputs missing" {
		t.Fatalf("unexpected final delivery block reason: %s", decision.Reason)
	}

	validationBundle := filepath.Join(root, "validation-bundle.md")
	releaseNotes := filepath.Join(root, "release-notes.md")
	if err := WriteReport(validationBundle, "# Validation Bundle\n\nready"); err != nil {
		t.Fatalf("write validation bundle: %v", err)
	}
	if err := WriteReport(releaseNotes, "# Release Notes\n\nready"); err != nil {
		t.Fatalf("write release notes: %v", err)
	}
	finalChecklist = BuildFinalDeliveryChecklist(
		"BIG-4702",
		[]DocumentationArtifact{
			{Name: "validation-bundle", Path: validationBundle},
			{Name: "release-notes", Path: releaseNotes},
		},
		[]DocumentationArtifact{
			{Name: "runbook", Path: filepath.Join(root, "runbook.md")},
		},
	)
	decision = EvaluateIssueClosure("BIG-4702", finalReportPath, true, nil, &finalChecklist)
	if !decision.Allowed {
		t.Fatalf("expected final delivery checklist to allow closure when required outputs exist")
	}
	if decision.Reason != "validation report and final delivery checklist requirements satisfied; issue can be closed" {
		t.Fatalf("unexpected final delivery ready reason: %s", decision.Reason)
	}

	launchReportPath := filepath.Join(root, "launch-validation.md")
	runbook := filepath.Join(root, "launch-runbook.md")
	faq := filepath.Join(root, "launch-faq.md")
	if err := WriteReport(launchReportPath, RenderIssueValidationReport("BIG-1003", "v0.2", "staging", "pass")); err != nil {
		t.Fatalf("write launch validation report: %v", err)
	}
	if err := WriteReport(runbook, "# Runbook\n\nready"); err != nil {
		t.Fatalf("write launch runbook: %v", err)
	}
	if err := WriteReport(faq, "# FAQ\n\nready"); err != nil {
		t.Fatalf("write launch faq: %v", err)
	}
	launchChecklist := BuildLaunchChecklist(
		"BIG-1003",
		[]DocumentationArtifact{
			{Name: "runbook", Path: runbook},
			{Name: "launch-faq", Path: faq},
		},
		[]LaunchChecklistItem{{Name: "Launch comms", Evidence: []string{"runbook", "launch-faq"}}},
	)
	decision = EvaluateIssueClosure("BIG-1003", launchReportPath, true, &launchChecklist, nil)
	if !decision.Allowed {
		t.Fatalf("expected ready launch checklist to allow issue closure")
	}
	if decision.Reason != "validation report and launch checklist requirements satisfied; issue can be closed" {
		t.Fatalf("unexpected launch ready reason: %s", decision.Reason)
	}
}

func TestWriteReportWritesContentToDisk(t *testing.T) {
	output := filepath.Join(t.TempDir(), "report.md")
	content := RenderIssueValidationReport("BIG-101", "v0.1", "sandbox", "pass")
	if err := WriteReport(output, content); err != nil {
		t.Fatalf("write report: %v", err)
	}
	body, err := os.ReadFile(output)
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
		t.Fatalf("expected enabled action state, got %q", enabled.State())
	}
	if disabled.State() != "disabled" {
		t.Fatalf("expected disabled action state, got %q", disabled.State())
	}
}

func TestRenderIssueValidationReportUsesTimezoneAwareUTCTimestamp(t *testing.T) {
	content := RenderIssueValidationReport("BIG-900", "v1", "repo", "pass")
	if !strings.Contains(content, "# Issue Validation Report") {
		t.Fatalf("expected header in issue validation report, got %s", content)
	}
	var timestampValue string
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "- 生成时间: ") {
			timestampValue = strings.TrimPrefix(line, "- 生成时间: ")
			break
		}
	}
	if !strings.HasSuffix(timestampValue, "Z") {
		t.Fatalf("expected UTC timestamp suffix, got %q", timestampValue)
	}
	parsed, err := time.Parse(time.RFC3339, timestampValue)
	if err != nil {
		t.Fatalf("parse timestamp: %v", err)
	}
	if parsed.Location() != time.UTC && parsed.UTC() != parsed {
		t.Fatalf("expected UTC timestamp, got %v", parsed)
	}
}

func TestBuildOrchestrationCanvasAndTakeoverQueueAcceptCanonicalHandoffEvents(t *testing.T) {
	entry := map[string]any{
		"run_id":  "run-ope-134-canvas",
		"task_id": "OPE-134-canvas",
		"source":  "linear",
		"summary": "handoff requested",
		"audits": []any{
			map[string]any{
				"action":  "orchestration.plan",
				"actor":   "scheduler",
				"outcome": "ready",
				"details": map[string]any{
					"collaboration_mode": "cross-functional",
					"departments":        []any{"operations", "engineering"},
					"approvals":          []any{"security-review"},
				},
			},
			map[string]any{
				"action":  "execution.manual_takeover",
				"actor":   "scheduler",
				"outcome": "pending",
				"details": map[string]any{
					"task_id":            "OPE-134-canvas",
					"run_id":             "run-ope-134-canvas",
					"target_team":        "security",
					"reason":             "manual review required",
					"requested_by":       "scheduler",
					"required_approvals": []any{"security-review"},
				},
			},
		},
	}

	canvas := BuildOrchestrationCanvasFromLedgerEntry(entry)
	queue := BuildTakeoverQueueFromLedger([]map[string]any{entry}, "2026-03-11")

	if canvas.HandoffTeam != "security" || canvas.HandoffStatus != "pending" {
		t.Fatalf("unexpected orchestration canvas: %+v", canvas)
	}
	if len(queue.Requests) != 1 || !reflect.DeepEqual(queue.Requests[0].RequiredApprovals, []string{"security-review"}) {
		t.Fatalf("unexpected takeover queue: %+v", queue)
	}
}

func TestBuildTakeoverQueueFromLedgerGroupsPendingHandoffs(t *testing.T) {
	entries := []map[string]any{
		{
			"run_id":  "run-sec",
			"task_id": "OPE-66-sec",
			"source":  "linear",
			"summary": "requires approval for high-risk task",
			"audits": []any{
				map[string]any{
					"action":  "orchestration.handoff",
					"outcome": "pending",
					"details": map[string]any{
						"target_team":        "security",
						"reason":             "requires approval for high-risk task",
						"required_approvals": []any{"security-review"},
					},
				},
			},
		},
		{
			"run_id":  "run-ops",
			"task_id": "OPE-66-ops",
			"source":  "linear",
			"summary": "premium tier required for advanced cross-department orchestration",
			"audits": []any{
				map[string]any{
					"action":  "orchestration.handoff",
					"outcome": "pending",
					"details": map[string]any{
						"target_team":        "operations",
						"reason":             "premium tier required for advanced cross-department orchestration",
						"required_approvals": []any{"ops-manager"},
					},
				},
			},
		},
		{
			"run_id":  "run-ok",
			"task_id": "OPE-66-ok",
			"source":  "linear",
			"summary": "default low risk path",
			"audits": []any{
				map[string]any{
					"action":  "scheduler.decision",
					"outcome": "approved",
					"details": map[string]any{"reason": "default low risk path"},
				},
			},
		},
	}

	queue := BuildTakeoverQueueFromLedger(entries, "2026-03-10")
	totalRuns := 3
	report := RenderTakeoverQueueReport(queue, &totalRuns, nil)

	if queue.PendingRequests() != 2 {
		t.Fatalf("unexpected pending request count: %+v", queue)
	}
	if !reflect.DeepEqual(queue.TeamCounts(), map[string]int{"operations": 1, "security": 1}) {
		t.Fatalf("unexpected team counts: %+v", queue.TeamCounts())
	}
	if queue.ApprovalCount() != 2 {
		t.Fatalf("unexpected approval count: %d", queue.ApprovalCount())
	}
	if queue.Recommendation() != "expedite-security-review" {
		t.Fatalf("unexpected recommendation: %s", queue.Recommendation())
	}
	if got := []string{queue.Requests[0].RunID, queue.Requests[1].RunID}; !reflect.DeepEqual(got, []string{"run-ops", "run-sec"}) {
		t.Fatalf("unexpected request order: %+v", got)
	}
	if !queue.Requests[0].Actions[3].Enabled {
		t.Fatalf("expected operations escalation action enabled: %+v", queue.Requests[0].Actions[3])
	}
	if queue.Requests[1].Actions[3].Enabled {
		t.Fatalf("expected security escalation action disabled: %+v", queue.Requests[1].Actions[3])
	}
	for _, fragment := range []string{
		"Pending Requests: 2",
		"Team Mix: operations=1 security=1",
		"run-sec: team=security status=pending task=OPE-66-sec approvals=security-review",
		"run-ops: team=operations status=pending task=OPE-66-ops approvals=ops-manager",
		"Escalate [escalate] state=disabled target=run-sec reason=security takeovers are already escalated",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in takeover queue report, got %s", fragment, report)
		}
	}
}

func TestRenderTakeoverQueueReportRendersSharedViewErrorState(t *testing.T) {
	queue := BuildTakeoverQueueFromLedger(nil, "2026-03-10")
	resultCount := 0
	report := RenderTakeoverQueueReport(queue, nil, &SharedViewContext{
		ResultCount: &resultCount,
		Errors:      []string{"Takeover approvals service timed out."},
	})

	for _, fragment := range []string{
		"- State: error",
		"- Summary: Unable to load data for the current filters.",
		"## Errors",
		"Takeover approvals service timed out.",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in shared-view error report, got %s", fragment, report)
		}
	}
}

func TestRenderOrchestrationCanvasSummarizesPolicyAndHandoff(t *testing.T) {
	entry := map[string]any{
		"run_id":  "run-canvas",
		"task_id": "OPE-66-canvas",
		"source":  "linear",
		"summary": "Canvas run",
		"audits": []any{
			map[string]any{
				"action": "tool.invoke",
				"details": map[string]any{
					"tool": "browser",
				},
			},
			map[string]any{
				"action": "orchestration.plan",
				"details": map[string]any{
					"collaboration_mode": "tier-limited",
					"departments":        []any{"operations", "engineering"},
				},
			},
			map[string]any{
				"action":  "orchestration.policy",
				"outcome": "upgrade-required",
				"details": map[string]any{
					"tier":                 "standard",
					"reason":               "premium tier required for advanced cross-department orchestration",
					"blocked_departments":  []any{"customer-success"},
					"entitlement_status":   "upgrade-required",
					"billing_model":        "standard-blocked",
					"estimated_cost_usd":   7.0,
					"included_usage_units": 2,
					"overage_usage_units":  1,
					"overage_cost_usd":     4.0,
				},
			},
			map[string]any{
				"action":  "orchestration.handoff",
				"outcome": "pending",
				"details": map[string]any{
					"target_team":        "operations",
					"reason":             "premium tier required for advanced cross-department orchestration",
					"required_approvals": []any{"ops-manager"},
				},
			},
		},
	}

	canvas := BuildOrchestrationCanvasFromLedgerEntry(entry)
	report := RenderOrchestrationCanvas(canvas)

	if canvas.Recommendation() != "resolve-entitlement-gap" {
		t.Fatalf("unexpected recommendation: %+v", canvas)
	}
	if !reflect.DeepEqual(canvas.ActiveTools, []string{"browser"}) {
		t.Fatalf("unexpected active tools: %+v", canvas.ActiveTools)
	}
	if !canvas.Actions[3].Enabled {
		t.Fatalf("expected escalate action enabled: %+v", canvas.Actions[3])
	}
	if canvas.Actions[4].Enabled {
		t.Fatalf("expected retry action disabled for pending handoff: %+v", canvas.Actions[4])
	}
	for _, fragment := range []string{
		"# Orchestration Canvas",
		"- Tier: standard",
		"- Entitlement Status: upgrade-required",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in orchestration canvas report, got %s", fragment, report)
		}
	}
}

func TestOrchestrationCanvasReconstructsFlowCollaborationFromLedger(t *testing.T) {
	entry := map[string]any{
		"run_id":  "run-flow-1",
		"task_id": "OPE-113",
		"audits": []any{
			map[string]any{
				"action":    "orchestration.plan",
				"actor":     "scheduler",
				"outcome":   "enabled",
				"timestamp": "2026-03-11T11:00:00Z",
				"details": map[string]any{
					"collaboration_mode": "cross-functional",
					"departments":        []any{"operations", "engineering"},
					"approvals":          []any{},
				},
			},
			map[string]any{
				"action":    "orchestration.policy",
				"actor":     "scheduler",
				"outcome":   "enabled",
				"timestamp": "2026-03-11T11:01:00Z",
				"details": map[string]any{
					"tier":               "premium",
					"entitlement_status": "included",
					"billing_model":      "premium-included",
				},
			},
			map[string]any{
				"action":    "collaboration.comment",
				"actor":     "ops-lead",
				"outcome":   "recorded",
				"timestamp": "2026-03-11T11:02:00Z",
				"details": map[string]any{
					"surface":    "flow",
					"comment_id": "flow-comment-1",
					"body":       "Route @eng once the dashboard note is resolved.",
					"mentions":   []any{"eng"},
					"anchor":     "handoff-lane",
					"status":     "open",
				},
			},
			map[string]any{
				"action":    "collaboration.decision",
				"actor":     "eng-manager",
				"outcome":   "accepted",
				"timestamp": "2026-03-11T11:03:00Z",
				"details": map[string]any{
					"surface":             "flow",
					"decision_id":         "flow-decision-1",
					"summary":             "Engineering owns the next flow handoff.",
					"mentions":            []any{"ops-lead"},
					"related_comment_ids": []any{"flow-comment-1"},
					"follow_up":           "Post in the shared channel after deploy.",
				},
			},
		},
	}

	canvas := BuildOrchestrationCanvasFromLedgerEntry(entry)
	report := RenderOrchestrationCanvas(canvas)

	if canvas.Collaboration == nil {
		t.Fatalf("expected collaboration thread on canvas: %+v", canvas)
	}
	if canvas.Recommendation() != "resolve-flow-comments" {
		t.Fatalf("unexpected collaboration recommendation: %+v", canvas)
	}
	for _, fragment := range []string{
		"## Collaboration",
		"Route @eng once the dashboard note is resolved.",
		"Engineering owns the next flow handoff.",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in collaboration canvas report, got %s", fragment, report)
		}
	}
}

func TestAutoTriageCenterPrioritizesFailedAndPendingRuns(t *testing.T) {
	runs := []AutoTriageRun{
		{
			RunID:   "run-ok",
			TaskID:  "OPE-76-ok",
			Source:  "linear",
			Title:   "Healthy run",
			Medium:  "docker",
			Status:  "approved",
			Summary: "default low risk path",
			Traces:  []AutoTriageRunTrace{{Span: "scheduler.decide", Status: "ok"}},
			Audits:  []AutoTriageRunAudit{{Action: "scheduler.decision", Outcome: "approved", Details: map[string]any{"reason": "default low risk path"}}},
		},
		{
			RunID:   "run-risk",
			TaskID:  "OPE-76-risk",
			Source:  "linear",
			Title:   "Prod approval",
			Medium:  "vm",
			Status:  "needs-approval",
			Summary: "requires approval for high-risk task",
			Traces:  []AutoTriageRunTrace{{Span: "scheduler.decide", Status: "pending"}},
			Audits:  []AutoTriageRunAudit{{Action: "scheduler.decision", Outcome: "pending", Details: map[string]any{"reason": "requires approval for high-risk task"}}},
		},
		{
			RunID:   "run-browser",
			TaskID:  "OPE-76-browser",
			Source:  "linear",
			Title:   "Replay browser task",
			Medium:  "browser",
			Status:  "failed",
			Summary: "browser session crashed",
			Traces:  []AutoTriageRunTrace{{Span: "runtime.execute", Status: "failed"}},
			Audits:  []AutoTriageRunAudit{{Action: "runtime.execute", Outcome: "failed", Details: map[string]any{"reason": "browser session crashed"}}},
		},
	}
	center := BuildAutoTriageCenter(runs, "Engineering Ops", "2026-03-10", nil)
	totalRuns := 3
	report := RenderAutoTriageCenterReport(center, &totalRuns, nil)

	if center.FlaggedRuns() != 2 || center.InboxSize() != 2 {
		t.Fatalf("unexpected center sizing: %+v", center)
	}
	if !reflect.DeepEqual(center.SeverityCounts(), map[string]int{"critical": 1, "high": 1, "medium": 0}) {
		t.Fatalf("unexpected severity counts: %+v", center.SeverityCounts())
	}
	if !reflect.DeepEqual(center.OwnerCounts(), map[string]int{"security": 1, "engineering": 1, "operations": 0}) {
		t.Fatalf("unexpected owner counts: %+v", center.OwnerCounts())
	}
	if center.Recommendation() != "immediate-attention" {
		t.Fatalf("unexpected recommendation: %s", center.Recommendation())
	}
	if got := []string{center.Findings[0].RunID, center.Findings[1].RunID}; !reflect.DeepEqual(got, []string{"run-browser", "run-risk"}) {
		t.Fatalf("unexpected finding order: %+v", got)
	}
	if got := []string{center.Inbox[0].RunID, center.Inbox[1].RunID}; !reflect.DeepEqual(got, []string{"run-browser", "run-risk"}) {
		t.Fatalf("unexpected inbox order: %+v", got)
	}
	if center.Inbox[0].Suggestions[0].Label != "replay candidate" || center.Inbox[0].Suggestions[0].Confidence < 0.55 {
		t.Fatalf("unexpected browser suggestion: %+v", center.Inbox[0].Suggestions[0])
	}
	if center.Findings[0].NextAction != "replay run and inspect tool failures" || center.Findings[1].NextAction != "request approval and queue security review" {
		t.Fatalf("unexpected next actions: %+v", center.Findings)
	}
	if !center.Findings[0].Actions[4].Enabled || center.Findings[1].Actions[4].Enabled || center.Findings[1].Actions[6].Enabled {
		t.Fatalf("unexpected action states: %+v", center.Findings)
	}
	for _, fragment := range []string{
		"Flagged Runs: 2",
		"Inbox Size: 2",
		"Severity Mix: critical=1 high=1 medium=0",
		"Feedback Loop: accepted=0 rejected=0 pending=2",
		"run-browser: severity=critical owner=engineering status=failed",
		"run-risk: severity=high owner=security status=needs-approval",
		"actions=Drill Down [drill-down]",
		"Retry [retry] state=disabled target=run-risk reason=retry available after owner review",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in auto triage report, got %s", fragment, report)
		}
	}
}

func TestAutoTriageCenterReportRendersSharedViewPartialState(t *testing.T) {
	runs := []AutoTriageRun{{
		RunID:   "run-risk",
		TaskID:  "OPE-94-risk",
		Source:  "linear",
		Title:   "Prod approval",
		Medium:  "vm",
		Status:  "needs-approval",
		Summary: "requires approval for high-risk task",
		Audits:  []AutoTriageRunAudit{{Action: "scheduler.decision", Outcome: "pending", Details: map[string]any{"reason": "requires approval for high-risk task"}}},
	}}
	center := BuildAutoTriageCenter(runs, "Engineering Ops", "2026-03-10", nil)
	resultCount := 1
	report := RenderAutoTriageCenterReport(center, &resultCount, &SharedViewContext{
		Filters:     []SharedViewFilter{{Label: "Team", Value: "engineering"}, {Label: "Window", Value: "2026-03-10"}},
		ResultCount: &resultCount,
		PartialData: []string{"Replay ledger data is still backfilling."},
		LastUpdated: "2026-03-11T09:00:00Z",
	})
	for _, fragment := range []string{
		"## View State",
		"- State: partial-data",
		"- Team: engineering",
		"## Partial Data",
		"Replay ledger data is still backfilling.",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in partial-state report, got %s", fragment, report)
		}
	}
}

func TestAutoTriageCenterBuildsSimilarityEvidenceAndFeedbackLoop(t *testing.T) {
	runs := []AutoTriageRun{
		{
			RunID:   "run-browser-a",
			TaskID:  "OPE-100-browser-a",
			Source:  "linear",
			Title:   "Browser replay failure",
			Medium:  "browser",
			Status:  "failed",
			Summary: "browser session crashed",
			Traces:  []AutoTriageRunTrace{{Span: "runtime.execute", Status: "failed"}},
			Audits:  []AutoTriageRunAudit{{Action: "runtime.execute", Outcome: "failed", Details: map[string]any{"reason": "browser session crashed"}}},
		},
		{
			RunID:   "run-browser-b",
			TaskID:  "OPE-100-browser-b",
			Source:  "linear",
			Title:   "Browser replay failure",
			Medium:  "browser",
			Status:  "failed",
			Summary: "browser session crashed",
			Traces:  []AutoTriageRunTrace{{Span: "runtime.execute", Status: "failed"}},
			Audits:  []AutoTriageRunAudit{{Action: "runtime.execute", Outcome: "failed", Details: map[string]any{"reason": "browser session crashed"}}},
		},
		{
			RunID:   "run-security",
			TaskID:  "OPE-100-security",
			Source:  "linear",
			Title:   "Security approval",
			Medium:  "vm",
			Status:  "needs-approval",
			Summary: "requires approval for high-risk task",
			Traces:  []AutoTriageRunTrace{{Span: "scheduler.decide", Status: "pending"}},
			Audits:  []AutoTriageRunAudit{{Action: "scheduler.decision", Outcome: "pending", Details: map[string]any{"reason": "requires approval for high-risk task"}}},
		},
	}
	feedback := []TriageFeedbackRecord{
		{RunID: "run-browser-a", Action: "replay run and inspect tool failures", Decision: "accepted", Actor: "ops-lead", Notes: "matched previous recovery path"},
		{RunID: "run-security", Action: "request approval and queue security review", Decision: "rejected", Actor: "sec-reviewer", Notes: "approval already in flight"},
	}
	center := BuildAutoTriageCenter(runs, "Auto Triage Center", "2026-03-11", feedback)
	totalRuns := 3
	report := RenderAutoTriageCenterReport(center, &totalRuns, nil)

	var browserItem, approvalItem *TriageInboxItem
	for i := range center.Inbox {
		if center.Inbox[i].RunID == "run-browser-a" {
			browserItem = &center.Inbox[i]
		}
		if center.Inbox[i].RunID == "run-security" {
			approvalItem = &center.Inbox[i]
		}
	}
	if !reflect.DeepEqual(center.FeedbackCounts(), map[string]int{"accepted": 1, "rejected": 1, "pending": 1}) {
		t.Fatalf("unexpected feedback counts: %+v", center.FeedbackCounts())
	}
	if browserItem == nil || approvalItem == nil {
		t.Fatalf("missing inbox items: %+v", center.Inbox)
	}
	if browserItem.Suggestions[0].FeedbackStatus != "accepted" || approvalItem.Suggestions[0].FeedbackStatus != "rejected" {
		t.Fatalf("unexpected feedback status: %+v %+v", browserItem.Suggestions[0], approvalItem.Suggestions[0])
	}
	if browserItem.Suggestions[0].Evidence[0].RelatedRunID != "run-browser-b" || browserItem.Suggestions[0].Evidence[0].Score < 0.8 {
		t.Fatalf("unexpected browser evidence: %+v", browserItem.Suggestions[0].Evidence)
	}
	for _, fragment := range []string{
		"## Inbox",
		"run-browser-a: severity=critical owner=engineering status=failed",
		"similar=run-browser-b:",
		"Feedback Loop: accepted=1 rejected=1 pending=1",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in feedback report, got %s", fragment, report)
		}
	}
}

func TestOrchestrationPortfolioRollsUpCanvasAndTakeoverState(t *testing.T) {
	canvases := []OrchestrationCanvas{
		{
			TaskID:             "OPE-66-a",
			RunID:              "run-a",
			CollaborationMode:  "cross-functional",
			Departments:        []string{"operations", "engineering", "security"},
			Tier:               "premium",
			EntitlementStatus:  "included",
			BillingModel:       "premium-included",
			EstimatedCostUSD:   4.5,
			IncludedUsageUnits: 3,
			HandoffTeam:        "security",
			HandoffStatus:      "pending",
		},
		{
			TaskID:             "OPE-66-b",
			RunID:              "run-b",
			CollaborationMode:  "tier-limited",
			Departments:        []string{"operations", "engineering"},
			Tier:               "standard",
			UpgradeRequired:    true,
			EntitlementStatus:  "upgrade-required",
			BillingModel:       "standard-blocked",
			EstimatedCostUSD:   7.0,
			IncludedUsageUnits: 2,
			OverageUsageUnits:  1,
			OverageCostUSD:     4.0,
			BlockedDepartments: []string{"customer-success"},
			HandoffTeam:        "operations",
			HandoffStatus:      "pending",
		},
	}
	queue := BuildTakeoverQueueFromLedger([]map[string]any{
		{
			"run_id":  "run-a",
			"task_id": "OPE-66-a",
			"source":  "linear",
			"audits": []any{
				map[string]any{
					"action":  "orchestration.handoff",
					"outcome": "pending",
					"details": map[string]any{"target_team": "security", "reason": "risk", "required_approvals": []any{"security-review"}},
				},
			},
		},
		{
			"run_id":  "run-b",
			"task_id": "OPE-66-b",
			"source":  "linear",
			"audits": []any{
				map[string]any{
					"action":  "orchestration.handoff",
					"outcome": "pending",
					"details": map[string]any{"target_team": "operations", "reason": "entitlement", "required_approvals": []any{"ops-manager"}},
				},
			},
		},
	}, "2026-03-10")
	queue.Name = "Cross-Team Takeovers"

	portfolio := BuildOrchestrationPortfolio(canvases, "Cross-Team Portfolio", "2026-03-10", &queue)
	report := RenderOrchestrationPortfolioReport(portfolio, nil)

	if portfolio.TotalRuns() != 2 {
		t.Fatalf("unexpected total runs: %+v", portfolio)
	}
	if !reflect.DeepEqual(portfolio.CollaborationModes(), map[string]int{"cross-functional": 1, "tier-limited": 1}) {
		t.Fatalf("unexpected collaboration modes: %+v", portfolio.CollaborationModes())
	}
	if !reflect.DeepEqual(portfolio.TierCounts(), map[string]int{"premium": 1, "standard": 1}) {
		t.Fatalf("unexpected tier counts: %+v", portfolio.TierCounts())
	}
	if !reflect.DeepEqual(portfolio.EntitlementCounts(), map[string]int{"included": 1, "upgrade-required": 1}) {
		t.Fatalf("unexpected entitlement counts: %+v", portfolio.EntitlementCounts())
	}
	if !reflect.DeepEqual(portfolio.BillingModelCounts(), map[string]int{"premium-included": 1, "standard-blocked": 1}) {
		t.Fatalf("unexpected billing model counts: %+v", portfolio.BillingModelCounts())
	}
	if portfolio.TotalEstimatedCostUSD() != 11.5 || portfolio.TotalOverageCostUSD() != 4.0 {
		t.Fatalf("unexpected portfolio costs: %+v", portfolio)
	}
	if portfolio.UpgradeRequiredCount() != 1 || portfolio.ActiveHandoffs() != 2 {
		t.Fatalf("unexpected portfolio rollups: %+v", portfolio)
	}
	if portfolio.Recommendation() != "stabilize-security-takeovers" {
		t.Fatalf("unexpected recommendation: %s", portfolio.Recommendation())
	}
	for _, fragment := range []string{
		"# Orchestration Portfolio Report",
		"- Collaboration Mix: cross-functional=1 tier-limited=1",
		"- Tier Mix: premium=1 standard=1",
		"- Entitlement Mix: included=1 upgrade-required=1",
		"- Billing Models: premium-included=1 standard-blocked=1",
		"- Estimated Cost (USD): 11.50",
		"- Overage Cost (USD): 4.00",
		"- Takeover Queue: pending=2 recommendation=expedite-security-review",
		"- run-a: mode=cross-functional tier=premium entitlement=included billing=premium-included estimated_cost_usd=4.50 overage_cost_usd=0.00 upgrade_required=false handoff=security",
		"actions=Drill Down [drill-down]",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in portfolio report, got %s", fragment, report)
		}
	}
}

func TestOrchestrationPortfolioReportRendersSharedViewEmptyState(t *testing.T) {
	portfolio := BuildOrchestrationPortfolio(nil, "Cross-Team Portfolio", "2026-03-10", nil)
	resultCount := 0
	report := RenderOrchestrationPortfolioReport(portfolio, &SharedViewContext{ResultCount: &resultCount})
	for _, fragment := range []string{
		"- State: empty",
		"- Summary: No records match the current filters.",
		"## Filters",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in empty-state report, got %s", fragment, report)
		}
	}
}

func TestRenderOrchestrationOverviewPage(t *testing.T) {
	queue := BuildTakeoverQueueFromLedger([]map[string]any{
		{
			"run_id":  "run-a",
			"task_id": "OPE-66-a",
			"source":  "linear",
			"audits": []any{
				map[string]any{
					"action":  "orchestration.handoff",
					"outcome": "pending",
					"details": map[string]any{"target_team": "security", "reason": "risk", "required_approvals": []any{"security-review"}},
				},
			},
		},
	}, "2026-03-10")
	queue.Name = "Cross-Team Takeovers"
	portfolio := OrchestrationPortfolio{
		Name:   "Cross-Team Portfolio",
		Period: "2026-03-10",
		Canvases: []OrchestrationCanvas{{
			TaskID:            "OPE-66-a",
			RunID:             "run-a",
			CollaborationMode: "cross-functional",
			Departments:       []string{"operations", "engineering"},
			Tier:              "premium",
			EntitlementStatus: "included",
			BillingModel:      "premium-included",
			EstimatedCostUSD:  3.0,
			HandoffTeam:       "security",
		}},
		TakeoverQueue: &queue,
	}
	page := RenderOrchestrationOverviewPage(portfolio)
	for _, fragment := range []string{
		"<title>Orchestration Overview",
		"Cross-Team Portfolio",
		"review-security-takeover",
		"Estimated Cost",
		"premium-included",
		"pending=1 recommendation=expedite-security-review",
		"run-a",
		"actions=Drill Down [drill-down]",
	} {
		if !strings.Contains(page, fragment) {
			t.Fatalf("expected %q in overview page, got %s", fragment, page)
		}
	}
}

func TestBuildOrchestrationCanvasFromLedgerEntryExtractsAuditState(t *testing.T) {
	entry := map[string]any{
		"run_id":  "run-ledger",
		"task_id": "OPE-66-ledger",
		"audits": []any{
			map[string]any{
				"action":  "orchestration.plan",
				"outcome": "ready",
				"details": map[string]any{
					"collaboration_mode": "tier-limited",
					"departments":        []any{"operations", "engineering"},
					"approvals":          []any{"security-review"},
				},
			},
			map[string]any{
				"action":  "orchestration.policy",
				"outcome": "upgrade-required",
				"details": map[string]any{
					"tier":                 "standard",
					"entitlement_status":   "upgrade-required",
					"billing_model":        "standard-blocked",
					"estimated_cost_usd":   7.0,
					"included_usage_units": 2,
					"overage_usage_units":  1,
					"overage_cost_usd":     4.0,
					"blocked_departments":  []any{"security", "customer-success"},
				},
			},
			map[string]any{
				"action":  "orchestration.handoff",
				"outcome": "pending",
				"details": map[string]any{
					"target_team": "operations",
					"reason":      "premium tier required for advanced cross-department orchestration",
				},
			},
			map[string]any{"action": "tool.invoke", "outcome": "success", "details": map[string]any{"tool": "browser"}},
		},
	}

	canvas := BuildOrchestrationCanvasFromLedgerEntry(entry)
	if canvas.RunID != "run-ledger" || canvas.CollaborationMode != "tier-limited" {
		t.Fatalf("unexpected canvas header: %+v", canvas)
	}
	if !reflect.DeepEqual(canvas.Departments, []string{"operations", "engineering"}) {
		t.Fatalf("unexpected departments: %+v", canvas.Departments)
	}
	if !reflect.DeepEqual(canvas.RequiredApprovals, []string{"security-review"}) {
		t.Fatalf("unexpected approvals: %+v", canvas.RequiredApprovals)
	}
	if canvas.Tier != "standard" || !canvas.UpgradeRequired {
		t.Fatalf("unexpected tier state: %+v", canvas)
	}
	if canvas.EntitlementStatus != "upgrade-required" || canvas.BillingModel != "standard-blocked" {
		t.Fatalf("unexpected billing state: %+v", canvas)
	}
	if canvas.EstimatedCostUSD != 7.0 || canvas.IncludedUsageUnits != 2 || canvas.OverageUsageUnits != 1 || canvas.OverageCostUSD != 4.0 {
		t.Fatalf("unexpected cost state: %+v", canvas)
	}
	if !reflect.DeepEqual(canvas.BlockedDepartments, []string{"security", "customer-success"}) {
		t.Fatalf("unexpected blocked departments: %+v", canvas.BlockedDepartments)
	}
	if canvas.HandoffTeam != "operations" {
		t.Fatalf("unexpected handoff team: %+v", canvas)
	}
	if !reflect.DeepEqual(canvas.ActiveTools, []string{"browser"}) {
		t.Fatalf("unexpected active tools: %+v", canvas.ActiveTools)
	}
	if !canvas.Actions[3].Enabled || canvas.Actions[4].Enabled {
		t.Fatalf("unexpected action states: %+v", canvas.Actions)
	}
}

func TestBuildOrchestrationPortfolioFromLedgerRollsUpEntries(t *testing.T) {
	entries := []map[string]any{
		{
			"run_id":  "run-a",
			"task_id": "OPE-66-a",
			"audits": []any{
				map[string]any{
					"action":  "orchestration.plan",
					"outcome": "ready",
					"details": map[string]any{
						"collaboration_mode": "cross-functional",
						"departments":        []any{"operations", "engineering", "security"},
						"approvals":          []any{"security-review"},
					},
				},
				map[string]any{
					"action":  "orchestration.policy",
					"outcome": "enabled",
					"details": map[string]any{
						"tier":                 "premium",
						"entitlement_status":   "included",
						"billing_model":        "premium-included",
						"estimated_cost_usd":   4.5,
						"included_usage_units": 3,
						"blocked_departments":  []any{},
					},
				},
				map[string]any{
					"action":  "orchestration.handoff",
					"outcome": "pending",
					"details": map[string]any{"target_team": "security", "reason": "approval required", "required_approvals": []any{"security-review"}},
				},
			},
		},
		{
			"run_id":  "run-b",
			"task_id": "OPE-66-b",
			"audits": []any{
				map[string]any{
					"action":  "orchestration.plan",
					"outcome": "ready",
					"details": map[string]any{
						"collaboration_mode": "tier-limited",
						"departments":        []any{"operations", "engineering"},
						"approvals":          []any{},
					},
				},
				map[string]any{
					"action":  "orchestration.policy",
					"outcome": "upgrade-required",
					"details": map[string]any{
						"tier":                 "standard",
						"entitlement_status":   "upgrade-required",
						"billing_model":        "standard-blocked",
						"estimated_cost_usd":   7.0,
						"included_usage_units": 2,
						"overage_usage_units":  1,
						"overage_cost_usd":     4.0,
						"blocked_departments":  []any{"customer-success"},
					},
				},
				map[string]any{
					"action":  "orchestration.handoff",
					"outcome": "pending",
					"details": map[string]any{"target_team": "operations", "reason": "entitlement gap", "required_approvals": []any{"ops-manager"}},
				},
			},
		},
	}

	portfolio := BuildOrchestrationPortfolioFromLedger(entries, "Ledger Portfolio", "2026-03-10")
	if portfolio.TotalRuns() != 2 {
		t.Fatalf("unexpected total runs: %+v", portfolio)
	}
	if !reflect.DeepEqual(portfolio.CollaborationModes(), map[string]int{"cross-functional": 1, "tier-limited": 1}) {
		t.Fatalf("unexpected collaboration modes: %+v", portfolio.CollaborationModes())
	}
	if !reflect.DeepEqual(portfolio.TierCounts(), map[string]int{"premium": 1, "standard": 1}) {
		t.Fatalf("unexpected tier counts: %+v", portfolio.TierCounts())
	}
	if !reflect.DeepEqual(portfolio.EntitlementCounts(), map[string]int{"included": 1, "upgrade-required": 1}) {
		t.Fatalf("unexpected entitlement counts: %+v", portfolio.EntitlementCounts())
	}
	if portfolio.TotalEstimatedCostUSD() != 11.5 {
		t.Fatalf("unexpected estimated cost: %+v", portfolio)
	}
	if portfolio.TakeoverQueue == nil || portfolio.TakeoverQueue.PendingRequests() != 2 {
		t.Fatalf("unexpected takeover queue: %+v", portfolio.TakeoverQueue)
	}
	if portfolio.Recommendation() != "stabilize-security-takeovers" {
		t.Fatalf("unexpected recommendation: %s", portfolio.Recommendation())
	}
}

func TestBuildBillingEntitlementsPageRollsUpOrchestrationCosts(t *testing.T) {
	portfolio := OrchestrationPortfolio{
		Name:   "Revenue Ops",
		Period: "2026-03",
		Canvases: []OrchestrationCanvas{
			{
				TaskID:             "OPE-104-a",
				RunID:              "run-billing-a",
				CollaborationMode:  "cross-functional",
				Departments:        []string{"operations", "engineering", "security"},
				Tier:               "premium",
				EntitlementStatus:  "included",
				BillingModel:       "premium-included",
				EstimatedCostUSD:   4.5,
				IncludedUsageUnits: 3,
				HandoffTeam:        "security",
			},
			{
				TaskID:             "OPE-104-b",
				RunID:              "run-billing-b",
				CollaborationMode:  "tier-limited",
				Departments:        []string{"operations", "engineering"},
				Tier:               "standard",
				UpgradeRequired:    true,
				EntitlementStatus:  "upgrade-required",
				BillingModel:       "standard-blocked",
				EstimatedCostUSD:   7.0,
				IncludedUsageUnits: 2,
				OverageUsageUnits:  1,
				OverageCostUSD:     4.0,
				BlockedDepartments: []string{"customer-success"},
				HandoffTeam:        "operations",
			},
		},
	}
	page := BuildBillingEntitlementsPage(portfolio, "OpenAGI Revenue Cloud", "Standard", "2026-03")
	report := RenderBillingEntitlementsReport(page, nil)

	if page.RunCount() != 2 || page.TotalIncludedUsageUnits() != 5 || page.TotalOverageUsageUnits() != 1 {
		t.Fatalf("unexpected page rollups: %+v", page)
	}
	if page.TotalEstimatedCostUSD() != 11.5 || page.TotalOverageCostUSD() != 4.0 {
		t.Fatalf("unexpected page costs: %+v", page)
	}
	if page.UpgradeRequiredCount() != 1 {
		t.Fatalf("unexpected upgrade required count: %+v", page)
	}
	if !reflect.DeepEqual(page.EntitlementCounts(), map[string]int{"included": 1, "upgrade-required": 1}) {
		t.Fatalf("unexpected entitlement counts: %+v", page.EntitlementCounts())
	}
	if !reflect.DeepEqual(page.BillingModelCounts(), map[string]int{"premium-included": 1, "standard-blocked": 1}) {
		t.Fatalf("unexpected billing model counts: %+v", page.BillingModelCounts())
	}
	if !reflect.DeepEqual(page.BlockedCapabilities(), []string{"customer-success"}) {
		t.Fatalf("unexpected blocked capabilities: %+v", page.BlockedCapabilities())
	}
	if page.Recommendation() != "resolve-plan-gaps" {
		t.Fatalf("unexpected recommendation: %s", page.Recommendation())
	}
	for _, fragment := range []string{
		"# Billing & Entitlements Report",
		"- Workspace: OpenAGI Revenue Cloud",
		"- Overage Cost (USD): 4.00",
		"- run-billing-b: task=OPE-104-b entitlement=upgrade-required billing=standard-blocked",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in billing report, got %s", fragment, report)
		}
	}
}

func TestRenderBillingEntitlementsPageOutputsHTMLDashboard(t *testing.T) {
	page := BillingEntitlementsPage{
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
	}
	pageHTML := RenderBillingEntitlementsPage(page)
	for _, fragment := range []string{
		"<title>Billing & Entitlements",
		"OpenAGI Revenue Cloud",
		"Premium plan for 2026-03",
		"Charge Feed",
		"premium-included",
	} {
		if !strings.Contains(pageHTML, fragment) {
			t.Fatalf("expected %q in billing page HTML, got %s", fragment, pageHTML)
		}
	}
}

func TestBuildBillingEntitlementsPageFromLedgerExtractsUpgradeSignals(t *testing.T) {
	entries := []map[string]any{
		{
			"run_id":  "run-ledger-a",
			"task_id": "OPE-104-a",
			"audits": []any{
				map[string]any{
					"action":  "orchestration.plan",
					"outcome": "ready",
					"details": map[string]any{
						"collaboration_mode": "cross-functional",
						"departments":        []any{"operations", "engineering", "security"},
						"approvals":          []any{"security-review"},
					},
				},
				map[string]any{
					"action":  "orchestration.policy",
					"outcome": "enabled",
					"details": map[string]any{
						"tier":                 "premium",
						"entitlement_status":   "included",
						"billing_model":        "premium-included",
						"estimated_cost_usd":   4.5,
						"included_usage_units": 3,
						"blocked_departments":  []any{},
					},
				},
			},
		},
		{
			"run_id":  "run-ledger-b",
			"task_id": "OPE-104-b",
			"audits": []any{
				map[string]any{
					"action":  "orchestration.plan",
					"outcome": "ready",
					"details": map[string]any{
						"collaboration_mode": "tier-limited",
						"departments":        []any{"operations", "engineering"},
						"approvals":          []any{},
					},
				},
				map[string]any{
					"action":  "orchestration.policy",
					"outcome": "upgrade-required",
					"details": map[string]any{
						"tier":                 "standard",
						"entitlement_status":   "upgrade-required",
						"billing_model":        "standard-blocked",
						"estimated_cost_usd":   7.0,
						"included_usage_units": 2,
						"overage_usage_units":  1,
						"overage_cost_usd":     4.0,
						"blocked_departments":  []any{"customer-success"},
					},
				},
				map[string]any{
					"action":  "orchestration.handoff",
					"outcome": "pending",
					"details": map[string]any{"target_team": "operations", "reason": "entitlement gap", "required_approvals": []any{"ops-manager"}},
				},
			},
		},
	}

	page := BuildBillingEntitlementsPageFromLedger(entries, "OpenAGI Revenue Cloud", "Standard", "2026-03")
	if page.RunCount() != 2 {
		t.Fatalf("unexpected run count: %+v", page)
	}
	if page.Recommendation() != "resolve-plan-gaps" || page.TotalOverageCostUSD() != 4.0 {
		t.Fatalf("unexpected page recommendation or cost: %+v", page)
	}
	if !reflect.DeepEqual(page.Charges[1].BlockedCapabilities, []string{"customer-success"}) {
		t.Fatalf("unexpected blocked capabilities: %+v", page.Charges[1])
	}
	if page.Charges[1].HandoffTeam != "operations" {
		t.Fatalf("unexpected handoff team: %+v", page.Charges[1])
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
	if metrics.RepoLinkCoverage != 50.0 {
		t.Fatalf("unexpected repo link coverage: %+v", metrics)
	}
	if metrics.AcceptedCommitRate != 50.0 {
		t.Fatalf("unexpected accepted commit rate: %+v", metrics)
	}
	if metrics.DiscussionDensity != 2.0 {
		t.Fatalf("unexpected discussion density: %+v", metrics)
	}
	if metrics.AcceptedLineageDepthAvg != 3.0 {
		t.Fatalf("unexpected accepted lineage depth avg: %+v", metrics)
	}
}

func TestNormalizeDashboardLayoutClampsDimensionsAndSortsPlacements(t *testing.T) {
	widgets := []DashboardWidgetSpec{
		{
			WidgetID:   "success-rate",
			Title:      "Success Rate",
			Module:     "kpis",
			DataSource: "operations.snapshot",
			MinWidth:   3,
			MaxWidth:   6,
		},
	}
	layout := DashboardLayout{
		LayoutID: "desktop",
		Name:     "Desktop",
		Placements: []DashboardWidgetPlacement{
			{
				PlacementID: "late",
				WidgetID:    "success-rate",
				Column:      8,
				Row:         4,
				Width:       8,
				Height:      0,
			},
			{
				PlacementID: "early",
				WidgetID:    "success-rate",
				Column:      -2,
				Row:         -1,
				Width:       1,
				Height:      2,
			},
		},
	}

	normalized := NormalizeDashboardLayout(layout, widgets)
	if len(normalized.Placements) != 2 {
		t.Fatalf("expected two placements, got %+v", normalized.Placements)
	}
	if normalized.Placements[0].PlacementID != "early" || normalized.Placements[1].PlacementID != "late" {
		t.Fatalf("unexpected placement order: %+v", normalized.Placements)
	}
	if normalized.Placements[0].Column != 0 || normalized.Placements[0].Row != 0 || normalized.Placements[0].Width != 3 {
		t.Fatalf("unexpected early placement: %+v", normalized.Placements[0])
	}
	if normalized.Placements[1].Column != 6 || normalized.Placements[1].Width != 6 || normalized.Placements[1].Height != 1 {
		t.Fatalf("unexpected late placement: %+v", normalized.Placements[1])
	}
}

func TestBuildTriageClustersGroupsActionableRunsByReason(t *testing.T) {
	runs := []map[string]any{
		{
			"run_id":     "run-1",
			"task_id":    "BIG-903-1",
			"status":     "needs-approval",
			"summary":    "hold",
			"started_at": "2026-03-10T10:00:00Z",
			"ended_at":   "2026-03-10T10:05:00Z",
			"audits": []any{
				map[string]any{"details": map[string]any{"reason": "requires approval for high-risk task"}},
			},
		},
		{
			"run_id":     "run-2",
			"task_id":    "BIG-903-2",
			"status":     "failed",
			"summary":    "tool fail",
			"started_at": "2026-03-10T10:00:00Z",
			"ended_at":   "2026-03-10T10:25:00Z",
			"audits": []any{
				map[string]any{"details": map[string]any{"reason": "browser automation task"}},
			},
		},
		{
			"run_id":     "run-3",
			"task_id":    "BIG-903-3",
			"status":     "needs-approval",
			"summary":    "hold",
			"started_at": "2026-03-10T11:00:00Z",
			"ended_at":   "2026-03-10T11:15:00Z",
			"audits": []any{
				map[string]any{"details": map[string]any{"reason": "requires approval for high-risk task"}},
			},
		},
	}

	clusters := BuildTriageClusters(runs)
	if len(clusters) != 2 {
		t.Fatalf("unexpected clusters: %+v", clusters)
	}
	if clusters[0].Reason != "requires approval for high-risk task" || clusters[0].Occurrences() != 2 {
		t.Fatalf("unexpected top cluster: %+v", clusters[0])
	}
	if !reflect.DeepEqual(clusters[0].TaskIDs, []string{"BIG-903-1", "BIG-903-3"}) {
		t.Fatalf("unexpected top cluster task ids: %+v", clusters[0].TaskIDs)
	}
	if clusters[1].Reason != "browser automation task" {
		t.Fatalf("unexpected second cluster: %+v", clusters[1])
	}
}

func TestBuildOperationsSnapshotTracksSLAAndSuccessRate(t *testing.T) {
	runs := []map[string]any{
		{
			"run_id":     "run-1",
			"task_id":    "BIG-901-1",
			"status":     "approved",
			"summary":    "ok",
			"started_at": "2026-03-10T10:00:00Z",
			"ended_at":   "2026-03-10T10:20:00Z",
			"audits": []any{
				map[string]any{"details": map[string]any{"reason": "default low risk path"}},
			},
		},
		{
			"run_id":     "run-2",
			"task_id":    "BIG-901-2",
			"status":     "approved",
			"summary":    "slow",
			"started_at": "2026-03-10T11:00:00Z",
			"ended_at":   "2026-03-10T12:30:00Z",
			"audits": []any{
				map[string]any{"details": map[string]any{"reason": "browser automation task"}},
			},
		},
		{
			"run_id":     "run-3",
			"task_id":    "BIG-901-3",
			"status":     "needs-approval",
			"summary":    "approval",
			"started_at": "2026-03-10T13:00:00Z",
			"ended_at":   "2026-03-10T13:45:00Z",
			"audits": []any{
				map[string]any{"details": map[string]any{"reason": "requires approval for high-risk task"}},
			},
		},
	}

	snapshot := BuildOperationsSnapshot(runs, 60, 3)
	if snapshot.TotalRuns != 3 {
		t.Fatalf("unexpected total runs: %+v", snapshot)
	}
	if !reflect.DeepEqual(snapshot.StatusCounts, map[string]int{"approved": 2, "needs-approval": 1}) {
		t.Fatalf("unexpected status counts: %+v", snapshot.StatusCounts)
	}
	if snapshot.SuccessRate != 66.7 || snapshot.ApprovalQueueDepth != 1 || snapshot.SLABreachCount != 1 || snapshot.AverageCycleMinutes != 51.7 {
		t.Fatalf("unexpected snapshot rollup: %+v", snapshot)
	}
}

func TestAnalyzeBenchmarkRegressionsFlagsScoreDropAndPassFailure(t *testing.T) {
	baseline := BenchmarkSuiteResult{
		Version: "v0.1",
		Results: []BenchmarkCaseResult{
			{CaseID: "case-stable", Score: 100, Passed: true},
			{CaseID: "case-drop", Score: 100, Passed: true},
		},
	}
	current := BenchmarkSuiteResult{
		Version: "v0.2",
		Results: []BenchmarkCaseResult{
			{CaseID: "case-stable", Score: 100, Passed: true},
			{CaseID: "case-drop", Score: 60, Passed: false},
		},
	}

	regressions := AnalyzeBenchmarkRegressions(current, baseline)
	if len(regressions) != 1 {
		t.Fatalf("unexpected regressions: %+v", regressions)
	}
	if regressions[0].CaseID != "case-drop" || regressions[0].Delta != -40 || regressions[0].Severity != "high" {
		t.Fatalf("unexpected regression finding: %+v", regressions[0])
	}
}

func TestBuildBenchmarkRegressionCenterSeparatesRegressionsAndImprovements(t *testing.T) {
	baseline := BenchmarkSuiteResult{
		Version: "v0.1",
		Results: []BenchmarkCaseResult{
			{CaseID: "case-drop", Score: 100, Passed: true},
			{CaseID: "case-up", Score: 60, Passed: false},
			{CaseID: "case-stable", Score: 100, Passed: true},
		},
	}
	current := BenchmarkSuiteResult{
		Version: "v0.2",
		Results: []BenchmarkCaseResult{
			{CaseID: "case-drop", Score: 70, Passed: false},
			{CaseID: "case-up", Score: 100, Passed: true},
			{CaseID: "case-stable", Score: 100, Passed: true},
		},
	}

	center := BuildBenchmarkRegressionCenter(current, baseline, "Regression Analysis Center")
	if center.RegressionCount() != 1 {
		t.Fatalf("unexpected regression count: %+v", center)
	}
	if center.Regressions[0].CaseID != "case-drop" {
		t.Fatalf("unexpected regressions: %+v", center.Regressions)
	}
	if !reflect.DeepEqual(center.ImprovedCases, []string{"case-up"}) {
		t.Fatalf("unexpected improved cases: %+v", center.ImprovedCases)
	}
	if !reflect.DeepEqual(center.UnchangedCases, []string{"case-stable"}) {
		t.Fatalf("unexpected unchanged cases: %+v", center.UnchangedCases)
	}
}

func TestRenderOperationsSnapshotDashboardShowsSharedViewLoadingState(t *testing.T) {
	resultCount := 0
	view := &SharedViewContext{
		Filters: []SharedViewFilter{
			{Label: "Team", Value: "engineering"},
			{Label: "Status", Value: "needs-approval"},
		},
		ResultCount: &resultCount,
		Loading:     true,
		LastUpdated: "2026-03-11T09:00:00Z",
	}

	dashboard := RenderOperationsSnapshotDashboard(OperationsSnapshot{}, view)
	for _, fragment := range []string{
		"## View State",
		"- State: loading",
		"- Summary: Loading data for the current filters.",
		"- Team: engineering",
	} {
		if !strings.Contains(dashboard, fragment) {
			t.Fatalf("expected %q in dashboard, got %s", fragment, dashboard)
		}
	}
}

func TestRenderRegressionCenterShowsSharedViewPartialState(t *testing.T) {
	resultCount := 1
	view := &SharedViewContext{
		Filters: []SharedViewFilter{
			{Label: "Team", Value: "engineering"},
			{Label: "Status", Value: "needs-approval"},
		},
		ResultCount: &resultCount,
		PartialData: []string{"Historical baseline fetch is delayed."},
		LastUpdated: "2026-03-11T09:00:00Z",
	}
	center := regression.Center{
		Summary: regression.Summary{TotalRegressions: 1},
		Findings: []regression.Finding{
			{TaskID: "case-drop", Severity: "high", RegressionCount: 1, Summary: "score dropped"},
		},
	}

	report := RenderRegressionCenterWithView("Regression Analysis Center", center, view)
	for _, fragment := range []string{
		"- State: partial-data",
		"## Partial Data",
		"Historical baseline fetch is delayed.",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in regression report, got %s", fragment, report)
		}
	}
}

func TestRenderSharedViewContextIncludesCollaborationAnnotations(t *testing.T) {
	resultCount := 4
	view := &SharedViewContext{
		Filters:     []SharedViewFilter{{Label: "Team", Value: "ops"}},
		ResultCount: &resultCount,
		Collaboration: &CollaborationThread{
			Surface:  "dashboard",
			TargetID: "ops-overview",
			Comments: []CollaborationComment{{
				CommentID: "dashboard-comment-1",
				Author:    "pm",
				Body:      "Please review blocker copy with @ops and @eng.",
				Mentions:  []string{"ops", "eng"},
				Anchor:    "blockers",
			}},
			Decisions: []DecisionNote{{
				DecisionID: "dashboard-decision-1",
				Author:     "ops",
				Outcome:    "approved",
				Summary:    "Keep the blocker module visible for managers.",
				Mentions:   []string{"pm"},
				FollowUp:   "Recheck after next data refresh.",
			}},
		},
	}

	content := renderSharedViewContext(view)
	for _, fragment := range []string{
		"## Collaboration",
		"Surface: dashboard",
		"Please review blocker copy with @ops and @eng.",
		"Keep the blocker module visible for managers.",
	} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("expected %q in shared view context, got %s", fragment, content)
		}
	}
}

func TestTriageFeedbackRecordUsesTimezoneAwareUTCTimestamp(t *testing.T) {
	record := NewTriageFeedbackRecord("run-1", "classify", "accepted", "ops", "")
	if !strings.HasSuffix(record.Timestamp, "Z") {
		t.Fatalf("expected UTC timestamp suffix, got %q", record.Timestamp)
	}
	parsed, err := time.Parse(time.RFC3339, record.Timestamp)
	if err != nil {
		t.Fatalf("parse timestamp: %v", err)
	}
	if parsed.UTC() != parsed {
		t.Fatalf("expected UTC timestamp, got %v", parsed)
	}
}

func TestRenderPolicyPromptVersionCenterShowsSharedViewContext(t *testing.T) {
	resultCount := 1
	view := &SharedViewContext{
		Filters: []SharedViewFilter{
			{Label: "Team", Value: "engineering"},
			{Label: "Status", Value: "needs-approval"},
		},
		ResultCount: &resultCount,
		PartialData: []string{"Rollback simulation still running."},
		LastUpdated: "2026-03-11T09:00:00Z",
	}
	center := PolicyPromptVersionCenter{
		Name:        "Policy/Prompt Version Center",
		GeneratedAt: "2026-03-10T14:00:00Z",
		Histories: []VersionedArtifactHistory{
			{
				ArtifactType:     "prompt",
				ArtifactID:       "triage-system",
				CurrentVersion:   "v2",
				CurrentUpdatedAt: "2026-03-10T14:00:00Z",
				CurrentAuthor:    "ops-bot",
				CurrentSummary:   "reduce false escalations",
				RevisionCount:    2,
				Revisions: []VersionedArtifact{
					{Version: "v2", UpdatedAt: "2026-03-10T14:00:00Z", Author: "ops-bot", Summary: "reduce false escalations"},
					{Version: "v1", UpdatedAt: "2026-03-08T14:00:00Z", Author: "ops-bot", Summary: "initial prompt"},
				},
				RollbackVersion: "v1",
				RollbackReady:   true,
				ChangeSummary: &VersionChangeSummary{
					FromVersion: "v1",
					ToVersion:   "v2",
					Preview:     []string{"--- v1", "+++ v2", "+rubric: strict"},
				},
			},
		},
	}

	report := RenderPolicyPromptVersionCenterWithView(center, view)
	for _, fragment := range []string{
		"# Policy/Prompt Version Center",
		"### prompt / triage-system",
		"- Rollback Version: v1",
		"```diff",
		"- State: partial-data",
		"Rollback simulation still running.",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in version center report, got %s", fragment, report)
		}
	}
}

func TestRenderBenchmarkSuiteReport(t *testing.T) {
	suite := BenchmarkSuiteResult{
		Version: "v0.2",
		Results: []BenchmarkCaseResult{
			{CaseID: "browser-low-risk", Score: 100, Passed: true},
		},
	}
	baseline := BenchmarkSuiteResult{Version: "v0.1"}

	report := RenderBenchmarkSuiteReport(suite, &baseline)
	for _, fragment := range []string{
		"Version: v0.2",
		"Baseline Version: v0.1",
		"Score Delta: 100",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in benchmark suite report, got %s", fragment, report)
		}
	}
}

func TestRenderReplayDetailPageListsMismatches(t *testing.T) {
	expected := BenchmarkReplayRecord{TaskID: "BIG-804", RunID: "run-1", Medium: "docker", Approved: true, Status: "approved"}
	observed := BenchmarkReplayRecord{TaskID: "BIG-804", RunID: "run-1", Medium: "browser", Approved: false, Status: "needs-approval"}

	page := RenderReplayDetailPage(expected, observed, []string{"medium expected docker got browser", "approved expected True got False"})
	for _, fragment := range []string{
		"Replay Detail",
		"Timeline / Log Sync",
		"Split View",
		"Reports",
		"medium expected docker got browser",
		"needs-approval",
	} {
		if !strings.Contains(page, fragment) {
			t.Fatalf("expected %q in replay detail page, got %s", fragment, page)
		}
	}
}

func TestRenderRunReplayIndexPageLinksOutputs(t *testing.T) {
	page := RenderRunReplayIndexPage(
		"big-804-index",
		BenchmarkRunIndexRecord{TaskID: "BIG-804", Medium: "browser", Status: "approved", ReportPath: "task-run.md"},
		BenchmarkReplayOutcome{
			Matched:      true,
			ReplayRecord: BenchmarkReplayRecord{TaskID: "BIG-804", RunID: "run-1", Medium: "browser", Approved: true, Status: "approved"},
			ReportPath:   "replay.html",
		},
		[]BenchmarkCriterion{{Name: "decision-medium", Weight: 40, Passed: true, Detail: "detail"}},
	)
	for _, fragment := range []string{
		"Run Detail Index",
		"Timeline / Log Sync",
		"Acceptance",
		"Reports",
		"task-run.md",
		"replay.html",
		"decision-medium",
	} {
		if !strings.Contains(page, fragment) {
			t.Fatalf("expected %q in run replay index page, got %s", fragment, page)
		}
	}
}

func TestRenderRunReplayIndexPageWithoutReportPath(t *testing.T) {
	page := RenderRunReplayIndexPage(
		"big-804-index",
		BenchmarkRunIndexRecord{TaskID: "BIG-804", Medium: "docker", Status: "approved"},
		BenchmarkReplayOutcome{
			Matched:      true,
			ReplayRecord: BenchmarkReplayRecord{TaskID: "BIG-804", RunID: "run-1", Medium: "docker", Approved: true, Status: "approved"},
		},
		nil,
	)
	for _, fragment := range []string{"n/a", "Replay"} {
		if !strings.Contains(page, fragment) {
			t.Fatalf("expected %q in run replay index page, got %s", fragment, page)
		}
	}
}

func TestBenchmarkRunnerScoresAndReplaysCase(t *testing.T) {
	runner := BenchmarkRunner{StorageDir: t.TempDir()}
	expectedApproved := true
	result, err := runner.RunCase(BenchmarkCase{
		CaseID: "browser-low-risk",
		Task: domain.Task{
			ID:            "BIG-601",
			Source:        "linear",
			Title:         "Run browser benchmark",
			Description:   "validate routing",
			RiskLevel:     domain.RiskLow,
			RequiredTools: []string{"browser"},
		},
		ExpectedMedium:   "browser",
		ExpectedApproved: &expectedApproved,
		ExpectedStatus:   "approved",
		RequireReport:    true,
	})
	if err != nil {
		t.Fatalf("run case: %v", err)
	}
	if result.Score != 100 || !result.Passed || !result.Replay.Matched {
		t.Fatalf("unexpected benchmark result: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(runner.StorageDir, "browser-low-risk", "task-run.md")); err != nil {
		t.Fatalf("expected task report: %v", err)
	}
	if _, err := os.Stat(filepath.Join(runner.StorageDir, "benchmark-browser-low-risk", "replay.html")); err != nil {
		t.Fatalf("expected replay page: %v", err)
	}
	if _, err := os.Stat(filepath.Join(runner.StorageDir, "browser-low-risk", "run-detail.html")); err != nil {
		t.Fatalf("expected run detail page: %v", err)
	}
	if result.DetailPagePath != filepath.Join(runner.StorageDir, "browser-low-risk", "run-detail.html") {
		t.Fatalf("unexpected detail page path: %+v", result)
	}
}

func TestBenchmarkRunnerDetectsFailedExpectation(t *testing.T) {
	runner := BenchmarkRunner{StorageDir: t.TempDir()}
	expectedApproved := false
	result, err := runner.RunCase(BenchmarkCase{
		CaseID: "high-risk-gate",
		Task: domain.Task{
			ID:          "BIG-601-risk",
			Source:      "jira",
			Title:       "Prod change benchmark",
			Description: "must stop for approval",
			RiskLevel:   domain.RiskHigh,
		},
		ExpectedMedium:   "docker",
		ExpectedApproved: &expectedApproved,
		ExpectedStatus:   "needs-approval",
	})
	if err != nil {
		t.Fatalf("run case: %v", err)
	}
	if result.Passed || result.Score != 60 {
		t.Fatalf("unexpected benchmark result: %+v", result)
	}
	found := false
	for _, item := range result.Criteria {
		if item.Name == "decision-medium" && !item.Passed {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected failed decision-medium criterion, got %+v", result.Criteria)
	}
}

func TestBenchmarkReplayReportsMismatch(t *testing.T) {
	runner := BenchmarkRunner{StorageDir: t.TempDir()}
	outcome, err := runner.Replay(BenchmarkReplayRecord{
		TaskID:   "BIG-601-replay",
		RunID:    "run-1",
		Medium:   "docker",
		Approved: true,
		Status:   "approved",
	})
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if outcome.Matched {
		t.Fatalf("expected replay mismatch, got %+v", outcome)
	}
	if !reflect.DeepEqual(outcome.Mismatches, []string{"medium expected docker got browser"}) {
		t.Fatalf("unexpected replay mismatches: %+v", outcome.Mismatches)
	}
	if outcome.ReportPath == "" {
		t.Fatalf("expected replay report path, got %+v", outcome)
	}
	if _, err := os.Stat(outcome.ReportPath); err != nil {
		t.Fatalf("expected replay report file: %v", err)
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
		Name:   "Exec Builder",
		Period: "2026-W11",
		Owner:  "ops-lead",
		Permissions: EngineeringOverviewPermission{
			ViewerRole:     "engineering-manager",
			AllowedModules: []string{"kpis", "operations", "regressions", "delivery", "finance"},
		},
		Widgets: []DashboardWidgetSpec{
			{
				WidgetID:   "success-rate",
				Title:      "Success Rate",
				Module:     "kpis",
				DataSource: "operations.snapshot",
			},
		},
		Layouts: []DashboardLayout{
			{
				LayoutID: "desktop",
				Name:     "Desktop",
				Placements: []DashboardWidgetPlacement{
					{
						PlacementID: "success-rate-main",
						WidgetID:    "success-rate",
						Column:      0,
						Row:         0,
						Width:       4,
						Height:      2,
					},
				},
			},
		},
		DocumentationComplete: true,
	}

	payload, err := json.Marshal(dashboard)
	if err != nil {
		t.Fatalf("marshal dashboard builder: %v", err)
	}

	var restored DashboardBuilder
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal dashboard builder: %v", err)
	}

	if !reflect.DeepEqual(restored, dashboard) {
		t.Fatalf("dashboard builder mismatch: restored=%+v want=%+v", restored, dashboard)
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

func TestRenderEngineeringOverviewHidesModulesWithoutPermission(t *testing.T) {
	base := time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC)
	tasks := []domain.Task{
		{
			ID:        "BIG-1401-1",
			Title:     "merged",
			State:     domain.TaskSucceeded,
			CreatedAt: base,
			UpdatedAt: base.Add(20 * time.Minute),
			Metadata: map[string]string{
				"run_id":  "run-1",
				"summary": "merged",
				"team":    "platform",
			},
		},
		{
			ID:        "BIG-1401-2",
			Title:     "approval",
			State:     domain.TaskBlocked,
			CreatedAt: base.Add(time.Hour),
			UpdatedAt: base.Add(85 * time.Minute),
			Metadata: map[string]string{
				"run_id":          "run-2",
				"summary":         "approval",
				"team":            "operations",
				"approval_status": "needs-approval",
				"blocked_reason":  "requires approval for prod deploy",
			},
		},
	}
	events := []domain.Event{
		{
			ID:        "evt-approval",
			Type:      domain.EventRunAnnotated,
			TaskID:    "BIG-1401-2",
			RunID:     "run-2",
			Timestamp: base.Add(85 * time.Minute),
			Payload:   map[string]any{"reason": "requires approval for prod deploy"},
		},
	}

	executiveReport := RenderEngineeringOverview(BuildEngineeringOverview("Executive View", "2026-W11", "executive", tasks, events, 60, 0, 0))
	contributorReport := RenderEngineeringOverview(BuildEngineeringOverview("Contributor View", "2026-W11", "contributor", tasks, events, 60, 0, 0))

	if !strings.Contains(executiveReport, "## KPI Modules") ||
		!strings.Contains(executiveReport, "## Funnel Modules") ||
		!strings.Contains(executiveReport, "## Blocker Modules") ||
		strings.Contains(executiveReport, "## Activity Modules") {
		t.Fatalf("unexpected executive module visibility: %s", executiveReport)
	}
	if !strings.Contains(contributorReport, "## KPI Modules") ||
		!strings.Contains(contributorReport, "## Activity Modules") ||
		strings.Contains(contributorReport, "## Funnel Modules") ||
		strings.Contains(contributorReport, "## Blocker Modules") {
		t.Fatalf("unexpected contributor module visibility: %s", contributorReport)
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
