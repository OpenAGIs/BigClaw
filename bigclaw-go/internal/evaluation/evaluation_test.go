package evaluation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBenchmarkRunnerScoresAndReplaysCase(t *testing.T) {
	root := t.TempDir()
	runner := New(root)
	result, err := runner.RunCase(BenchmarkCase{
		CaseID: "browser-low-risk",
		Task: Task{
			ID:            "BIG-601",
			Source:        "linear",
			Title:         "Run browser benchmark",
			RiskLevel:     "low",
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
	if result.Score != 100 || !result.Passed || !result.Replay.Matched {
		t.Fatalf("unexpected benchmark result: %+v", result)
	}
	for _, path := range []string{
		filepath.Join(root, "browser-low-risk", "task-run.md"),
		filepath.Join(root, "benchmark-browser-low-risk", "replay.html"),
		filepath.Join(root, "browser-low-risk", "run-detail.html"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected output %s: %v", path, err)
		}
	}
	if result.DetailPagePath != filepath.Join(root, "browser-low-risk", "run-detail.html") {
		t.Fatalf("unexpected detail path: %+v", result)
	}
}

func TestBenchmarkRunnerDetectsFailedExpectation(t *testing.T) {
	runner := New(t.TempDir())
	result, err := runner.RunCase(BenchmarkCase{
		CaseID: "high-risk-gate",
		Task: Task{
			ID:        "BIG-601-risk",
			Source:    "jira",
			Title:     "Prod change benchmark",
			RiskLevel: "high",
		},
		ExpectedMedium:   "docker",
		ExpectedApproved: false,
		ExpectedStatus:   "needs-approval",
	})
	if err != nil {
		t.Fatalf("run case: %v", err)
	}
	if result.Passed || result.Score != 60 {
		t.Fatalf("expected failed 60-point result, got %+v", result)
	}
	if result.Criteria[0].Name != "decision-medium" || result.Criteria[0].Passed {
		t.Fatalf("expected failed medium criterion, got %+v", result.Criteria)
	}
}

func TestReplayOutcomeReportsMismatch(t *testing.T) {
	runner := New(t.TempDir())
	outcome, err := runner.Replay(ReplayRecord{
		Task: Task{
			ID:            "BIG-601-replay",
			Source:        "github",
			Title:         "Replay browser route",
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
	if outcome.Matched || len(outcome.Mismatches) == 0 || outcome.Mismatches[0] != "medium expected docker got browser" {
		t.Fatalf("unexpected replay outcome: %+v", outcome)
	}
	if _, err := os.Stat(outcome.ReportPath); err != nil {
		t.Fatalf("expected replay report: %v", err)
	}
}

func TestSuiteComparisonAndReport(t *testing.T) {
	runner := New(t.TempDir())
	improved, err := runner.RunSuite([]BenchmarkCase{{
		CaseID: "browser-low-risk",
		Task: Task{
			ID:            "BIG-601-v2",
			Source:        "linear",
			Title:         "Run browser benchmark",
			RequiredTools: []string{"browser"},
		},
		ExpectedMedium:   "browser",
		ExpectedApproved: true,
		ExpectedStatus:   "approved",
	}}, "v0.2")
	if err != nil {
		t.Fatalf("run suite: %v", err)
	}
	baseline := Suite{Results: nil, Version: "v0.1"}
	comparison := improved.Compare(baseline)
	report := RenderBenchmarkSuiteReport(improved, baseline)
	if comparison[0].Delta != 100 || improved.Score() != 100 {
		t.Fatalf("unexpected suite comparison: %+v suite=%+v", comparison, improved)
	}
	for _, needle := range []string{"Version: v0.2", "Baseline Version: v0.1", "Score Delta: 100"} {
		if !strings.Contains(report, needle) {
			t.Fatalf("expected %q in report, got %s", needle, report)
		}
	}
}

func TestRenderReplayDetailPageListsMismatches(t *testing.T) {
	task := Task{ID: "BIG-804", Source: "linear", Title: "Replay detail"}
	page := RenderReplayDetailPage(
		ReplayRecord{Task: task, RunID: "run-1", Medium: "docker", Approved: true, Status: "approved"},
		ReplayRecord{Task: task, RunID: "run-1", Medium: "browser", Approved: false, Status: "needs-approval"},
		[]string{"medium expected docker got browser", "approved expected True got False"},
	)
	for _, needle := range []string{"Replay Detail", "Timeline / Log Sync", "Split View", "Reports", "medium expected docker got browser", "needs-approval"} {
		if !strings.Contains(page, needle) {
			t.Fatalf("expected %q in page, got %s", needle, page)
		}
	}
}

func TestRenderRunReplayIndexPageLinksOutputs(t *testing.T) {
	root := t.TempDir()
	runner := New(root)
	result, err := runner.RunCase(BenchmarkCase{
		CaseID: "big-804-index",
		Task: Task{
			ID:            "BIG-804",
			Source:        "linear",
			Title:         "Run detail index",
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
	page, err := os.ReadFile(result.DetailPagePath)
	if err != nil {
		t.Fatalf("read detail page: %v", err)
	}
	for _, needle := range []string{"Run Detail Index", "Timeline / Log Sync", "Acceptance", "Reports", "task-run.md", "replay.html", "decision-medium"} {
		if !strings.Contains(string(page), needle) {
			t.Fatalf("expected %q in page, got %s", needle, string(page))
		}
	}
}

func TestRenderRunReplayIndexPageWithoutReportPath(t *testing.T) {
	page := RenderRunReplayIndexPage("big-804-index", "", "replay.html", nil)
	if !strings.Contains(page, "n/a") || !strings.Contains(page, "Replay") {
		t.Fatalf("unexpected page: %s", page)
	}
}
