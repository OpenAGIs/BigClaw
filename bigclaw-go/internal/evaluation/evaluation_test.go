package evaluation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/domain"
)

func TestBenchmarkRunnerScoresAndReplaysCase(t *testing.T) {
	runner := NewBenchmarkRunner(t.TempDir(), nil)
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
		ExpectedMedium:   "kubernetes",
		ExpectedApproved: true,
		ExpectedStatus:   "approved",
		RequireReport:    true,
	})
	if err != nil {
		t.Fatalf("run case: %v", err)
	}
	if result.Score != 100 || !result.Passed || !result.Replay.Matched {
		t.Fatalf("unexpected benchmark result: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(runner.storageDir, "browser-low-risk", "task-run.md")); err != nil {
		t.Fatalf("expected task-run output: %v", err)
	}
	if _, err := os.Stat(filepath.Join(runner.storageDir, "benchmark-browser-low-risk", "replay.html")); err != nil {
		t.Fatalf("expected replay output: %v", err)
	}
	if _, err := os.Stat(filepath.Join(runner.storageDir, "browser-low-risk", "run-detail.html")); err != nil {
		t.Fatalf("expected detail output: %v", err)
	}
}

func TestBenchmarkRunnerDetectsFailedExpectation(t *testing.T) {
	runner := NewBenchmarkRunner(t.TempDir(), nil)
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
		ExpectedApproved: false,
		ExpectedStatus:   "needs-approval",
	})
	if err != nil {
		t.Fatalf("run case: %v", err)
	}
	if result.Passed || result.Score != 60 {
		t.Fatalf("unexpected failed expectation result: %+v", result)
	}
	found := false
	for _, criterion := range result.Criteria {
		if criterion.Name == "decision-medium" && !criterion.Passed {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected failed decision-medium criterion, got %+v", result.Criteria)
	}
}

func TestReplayOutcomeReportsMismatch(t *testing.T) {
	runner := NewBenchmarkRunner(t.TempDir(), nil)
	outcome, err := runner.Replay(ReplayRecord{
		Task: domain.Task{
			ID:            "BIG-601-replay",
			Source:        "github",
			Title:         "Replay browser route",
			Description:   "compare deterministic scheduler behavior",
			RequiredTools: []string{"browser"},
		},
		RunID:    "run-1",
		Medium:   "docker",
		Approved: true,
		Status:   "approved",
	})
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if outcome.Matched || len(outcome.Mismatches) == 0 || outcome.Mismatches[0] != "medium expected docker got kubernetes" {
		t.Fatalf("unexpected replay outcome: %+v", outcome)
	}
	if strings.TrimSpace(outcome.ReportPath) == "" {
		t.Fatalf("expected report path, got %+v", outcome)
	}
}

func TestSuiteComparisonAndReport(t *testing.T) {
	runner := NewBenchmarkRunner(t.TempDir(), nil)
	improved, err := runner.RunSuite([]BenchmarkCase{{
		CaseID: "browser-low-risk",
		Task: domain.Task{
			ID:            "BIG-601-v2",
			Source:        "linear",
			Title:         "Run browser benchmark",
			Description:   "validate routing",
			RequiredTools: []string{"browser"},
		},
		ExpectedMedium:   "kubernetes",
		ExpectedApproved: true,
		ExpectedStatus:   "approved",
	}}, "v0.2")
	if err != nil {
		t.Fatalf("run suite: %v", err)
	}
	baseline := BenchmarkSuiteResult{Results: nil, Version: "v0.1"}
	comparison := improved.Compare(baseline)
	report := RenderBenchmarkSuiteReport(improved, baseline)
	if comparison[0].Delta != 100 || improved.Score() != 100 {
		t.Fatalf("unexpected suite comparison: %+v %+v", comparison, improved)
	}
	for _, fragment := range []string{"Version: v0.2", "Baseline Version: v0.1", "Score Delta: 100"} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in suite report, got %s", fragment, report)
		}
	}
}

func TestRenderReplayDetailPageListsMismatches(t *testing.T) {
	task := domain.Task{ID: "BIG-804", Source: "linear", Title: "Replay detail"}
	page := RenderReplayDetailPage(
		ReplayRecord{Task: task, RunID: "run-1", Medium: "docker", Approved: true, Status: "approved"},
		ReplayRecord{Task: task, RunID: "run-1", Medium: "kubernetes", Approved: false, Status: "needs-approval"},
		[]string{"medium expected docker got kubernetes", "approved expected true got false"},
	)
	for _, fragment := range []string{"Replay Detail", "Timeline / Log Sync", "Split View", "Reports", "medium expected docker got kubernetes", "needs-approval"} {
		if !strings.Contains(page, fragment) {
			t.Fatalf("expected %q in replay detail page, got %s", fragment, page)
		}
	}
}

func TestRenderRunReplayIndexPageLinksOutputs(t *testing.T) {
	runner := NewBenchmarkRunner(t.TempDir(), nil)
	result, err := runner.RunCase(BenchmarkCase{
		CaseID: "big-804-index",
		Task: domain.Task{
			ID:            "BIG-804",
			Source:        "linear",
			Title:         "Run detail index",
			Description:   "single landing page",
			RequiredTools: []string{"browser"},
		},
		ExpectedMedium:   "kubernetes",
		ExpectedApproved: true,
		ExpectedStatus:   "approved",
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
			t.Fatalf("expected %q in run replay index page, got %s", fragment, page)
		}
	}
}

func TestRenderRunReplayIndexPageWithoutReportPath(t *testing.T) {
	page := RenderRunReplayIndexPage("big-804-index", ReplayRecord{
		Task:     domain.Task{ID: "BIG-804", Source: "linear", Title: "Run detail index"},
		RunID:    "run-1",
		Medium:   "docker",
		Approved: true,
		Status:   "approved",
	}, ReplayOutcome{
		Matched:      true,
		ReplayRecord: ReplayRecord{Task: domain.Task{ID: "BIG-804"}, RunID: "run-1", Medium: "docker", Approved: true, Status: "approved"},
		ReportPath:   "",
	}, nil)
	if !strings.Contains(page, "n/a") || !strings.Contains(page, "Replay") {
		t.Fatalf("expected fallback replay output, got %s", page)
	}
}
