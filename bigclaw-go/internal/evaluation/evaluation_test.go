package evaluation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/domain"
)

func TestBenchmarkRunnerScoresAndReplaysCase(t *testing.T) {
	t.Parallel()

	runner := NewBenchmarkRunner(t.TempDir())
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
		ExpectedApproved: true,
		ExpectedStatus:   "approved",
		RequireReport:    true,
	})
	if err != nil {
		t.Fatalf("run case: %v", err)
	}

	if result.Score != 100 {
		t.Fatalf("score = %d, want 100", result.Score)
	}
	if !result.Passed {
		t.Fatalf("passed = false, want true")
	}
	if !result.Replay.Matched {
		t.Fatalf("replay matched = false, want true")
	}
	assertPathExists(t, filepath.Join(runner.StorageDir, "browser-low-risk", "task-run.md"))
	assertPathExists(t, filepath.Join(runner.StorageDir, "benchmark-browser-low-risk", "replay.html"))
	assertPathExists(t, filepath.Join(runner.StorageDir, "browser-low-risk", "run-detail.html"))
	if result.DetailPagePath != filepath.Join(runner.StorageDir, "browser-low-risk", "run-detail.html") {
		t.Fatalf("detail page path = %q", result.DetailPagePath)
	}
}

func TestBenchmarkRunnerDetectsFailedExpectation(t *testing.T) {
	t.Parallel()

	runner := NewBenchmarkRunner(t.TempDir())
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

	if result.Passed {
		t.Fatalf("passed = true, want false")
	}
	if result.Score != 60 {
		t.Fatalf("score = %d, want 60", result.Score)
	}
	found := false
	for _, item := range result.Criteria {
		if item.Name == "decision-medium" && !item.Passed {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected failed decision-medium criterion")
	}
}

func TestReplayOutcomeReportsMismatch(t *testing.T) {
	t.Parallel()

	runner := NewBenchmarkRunner(t.TempDir())
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

	if outcome.Matched {
		t.Fatalf("matched = true, want false")
	}
	if len(outcome.Mismatches) != 1 || outcome.Mismatches[0] != "medium expected docker got browser" {
		t.Fatalf("mismatches = %#v", outcome.Mismatches)
	}
	if outcome.ReportPath == "" {
		t.Fatalf("report path empty")
	}
	assertPathExists(t, outcome.ReportPath)
}

func TestSuiteComparisonAndReport(t *testing.T) {
	t.Parallel()

	runner := NewBenchmarkRunner(t.TempDir())
	improvedSuite, err := runner.RunSuite([]BenchmarkCase{{
		CaseID: "browser-low-risk",
		Task: domain.Task{
			ID:            "BIG-601-v2",
			Source:        "linear",
			Title:         "Run browser benchmark",
			Description:   "validate routing",
			RequiredTools: []string{"browser"},
		},
		ExpectedMedium:   "browser",
		ExpectedApproved: true,
		ExpectedStatus:   "approved",
	}}, "v0.2")
	if err != nil {
		t.Fatalf("run suite: %v", err)
	}
	baselineSuite := BenchmarkSuiteResult{Results: nil, Version: "v0.1"}
	comparison := improvedSuite.Compare(baselineSuite)
	report := RenderBenchmarkSuiteReport(improvedSuite, baselineSuite)

	if comparison[0].Delta != 100 {
		t.Fatalf("delta = %d, want 100", comparison[0].Delta)
	}
	if improvedSuite.Score != 100 {
		t.Fatalf("suite score = %d, want 100", improvedSuite.Score)
	}
	for _, want := range []string{"Version: v0.2", "Baseline Version: v0.1", "Score Delta: 100"} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing %q:\n%s", want, report)
		}
	}
}

func TestRenderReplayDetailPageListsMismatches(t *testing.T) {
	t.Parallel()

	task := domain.Task{ID: "BIG-804", Source: "linear", Title: "Replay detail"}
	expected := ReplayRecord{Task: task, RunID: "run-1", Medium: "docker", Approved: true, Status: "approved"}
	observed := ReplayRecord{Task: task, RunID: "run-1", Medium: "browser", Approved: false, Status: "needs-approval"}

	page := RenderReplayDetailPage(expected, observed, []string{
		"medium expected docker got browser",
		"approved expected true got false",
	})

	for _, want := range []string{
		"Replay Detail",
		"Timeline / Log Sync",
		"Split View",
		"Reports",
		"medium expected docker got browser",
		"needs-approval",
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("page missing %q:\n%s", want, page)
		}
	}
}

func TestRenderRunReplayIndexPageLinksOutputs(t *testing.T) {
	t.Parallel()

	runner := NewBenchmarkRunner(t.TempDir())
	result, err := runner.RunCase(BenchmarkCase{
		CaseID: "big-804-index",
		Task: domain.Task{
			ID:            "BIG-804",
			Source:        "linear",
			Title:         "Run detail index",
			Description:   "single landing page",
			RequiredTools: []string{"browser"},
		},
		ExpectedMedium:   "browser",
		ExpectedApproved: true,
		ExpectedStatus:   "approved",
		RequireReport:    true,
	})
	if err != nil {
		t.Fatalf("run case: %v", err)
	}

	pageBytes, err := os.ReadFile(result.DetailPagePath)
	if err != nil {
		t.Fatalf("read detail page: %v", err)
	}
	page := string(pageBytes)
	for _, want := range []string{
		"Run Detail Index",
		"Timeline / Log Sync",
		"Acceptance",
		"Reports",
		"task-run.md",
		"replay.html",
		"decision-medium",
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("page missing %q:\n%s", want, page)
		}
	}
}

func TestRenderRunReplayIndexPageWithoutReportPath(t *testing.T) {
	t.Parallel()

	task := domain.Task{ID: "BIG-804", Source: "linear", Title: "Run detail index"}
	replay := ReplayOutcome{
		Matched: true,
		ReplayRecord: ReplayRecord{
			Task:     task,
			RunID:    "run-1",
			Medium:   "docker",
			Approved: true,
			Status:   "approved",
		},
	}

	page := RenderRunReplayIndexPage("big-804-index", "task-run.md", replay, nil)
	for _, want := range []string{"n/a", "Replay"} {
		if !strings.Contains(page, want) {
			t.Fatalf("page missing %q:\n%s", want, page)
		}
	}
}

func assertPathExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
}
