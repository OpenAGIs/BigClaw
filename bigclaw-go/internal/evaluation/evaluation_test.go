package evaluation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/scheduler"
)

func TestRunnerScoresAndReplaysCase(t *testing.T) {
	runner := NewRunner(t.TempDir(), nil)
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
		ExpectedAccepted: true,
		ExpectedStatus:   "accepted",
		RequireReport:    true,
	})
	if err != nil {
		t.Fatalf("run case: %v", err)
	}
	if result.Score != 100 || !result.Passed || !result.Replay.Matched {
		t.Fatalf("unexpected result: %+v", result)
	}
	for _, path := range []string{
		filepath.Join(runner.storageDir, "browser-low-risk", "task-run.md"),
		filepath.Join(runner.storageDir, "benchmark-browser-low-risk", "replay.html"),
		filepath.Join(runner.storageDir, "browser-low-risk", "run-detail.html"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected artifact %s: %v", path, err)
		}
	}
}

func TestRunnerDetectsFailedExpectation(t *testing.T) {
	runner := NewRunner(t.TempDir(), nil)
	result, err := runner.RunCase(BenchmarkCase{
		CaseID: "high-risk-gate",
		Task: domain.Task{
			ID:          "BIG-601-risk",
			Source:      "jira",
			Title:       "Prod change benchmark",
			Description: "must stop for approval",
			RiskLevel:   domain.RiskHigh,
		},
		ExpectedMedium:   "local",
		ExpectedAccepted: true,
		ExpectedStatus:   "accepted",
	})
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
		t.Fatalf("expected failed medium criterion: %+v", result.Criteria)
	}
}

func TestReplayOutcomeReportsMismatch(t *testing.T) {
	runner := NewRunner(t.TempDir(), scheduler.New())
	outcome, err := runner.Replay(ReplayRecord{
		Task: domain.Task{
			ID:            "BIG-601-replay",
			Source:        "github",
			Title:         "Replay browser route",
			Description:   "compare deterministic scheduler behavior",
			RequiredTools: []string{"browser"},
		},
		RunID:    "run-1",
		Medium:   "local",
		Accepted: true,
		Status:   "accepted",
	})
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if outcome.Matched {
		t.Fatalf("expected mismatch, got %+v", outcome)
	}
	if len(outcome.Mismatches) == 0 || outcome.ReportPath == "" {
		t.Fatalf("expected mismatch details and report path, got %+v", outcome)
	}
	if _, err := os.Stat(outcome.ReportPath); err != nil {
		t.Fatalf("expected report path %s: %v", outcome.ReportPath, err)
	}
}

func TestSuiteComparisonAndReport(t *testing.T) {
	runner := NewRunner(t.TempDir(), nil)
	current, err := runner.RunSuite([]BenchmarkCase{{
		CaseID: "browser-low-risk",
		Task: domain.Task{
			ID:            "BIG-601-v2",
			Source:        "linear",
			Title:         "Run browser benchmark",
			Description:   "validate routing",
			RequiredTools: []string{"browser"},
		},
		ExpectedMedium:   "kubernetes",
		ExpectedAccepted: true,
		ExpectedStatus:   "accepted",
	}}, "v0.2")
	if err != nil {
		t.Fatalf("run suite: %v", err)
	}
	baseline := BenchmarkSuiteResult{Results: nil, Version: "v0.1"}
	comparison := current.Compare(baseline)
	report := RenderBenchmarkSuiteReport(current, baseline)
	if len(comparison) != 1 || comparison[0].Delta != 100 {
		t.Fatalf("unexpected comparison: %+v", comparison)
	}
	if current.Score() != 100 {
		t.Fatalf("unexpected suite score: %+v", current)
	}
	for _, fragment := range []string{"Version: v0.2", "Baseline Version: v0.1", "Score Delta: 100"} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report:\n%s", fragment, report)
		}
	}
}

func TestRenderReplayDetailPageListsMismatches(t *testing.T) {
	task := domain.Task{ID: "BIG-804", Source: "linear", Title: "Replay detail"}
	page := RenderReplayDetailPage(
		ReplayRecord{Task: task, RunID: "run-1", Medium: "local", Accepted: true, Status: "accepted"},
		ReplayRecord{Task: task, RunID: "run-1", Medium: "kubernetes", Accepted: false, Status: "needs-approval"},
		[]string{"medium expected local got kubernetes", "accepted expected true got false"},
	)
	for _, fragment := range []string{"Replay Detail", "Timeline / Log Sync", "Split View", "Reports", "medium expected local got kubernetes", "needs-approval"} {
		if !strings.Contains(page, fragment) {
			t.Fatalf("expected %q in page:\n%s", fragment, page)
		}
	}
}

func TestRenderRunReplayIndexPageLinksOutputs(t *testing.T) {
	runner := NewRunner(t.TempDir(), nil)
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
		ExpectedAccepted: true,
		ExpectedStatus:   "accepted",
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
			t.Fatalf("expected %q in page:\n%s", fragment, page)
		}
	}
}

func TestRenderRunReplayIndexPageWithoutReportPath(t *testing.T) {
	page := RenderRunReplayIndexPage(
		"big-804-index",
		ReplayRecord{Task: domain.Task{ID: "BIG-804", Source: "linear", Title: "Run detail index"}, RunID: "run-1", Medium: "local", Accepted: true, Status: "accepted"},
		ReplayOutcome{Matched: true, ReplayRecord: ReplayRecord{Task: domain.Task{ID: "BIG-804"}, RunID: "run-1", Medium: "local", Accepted: true, Status: "accepted"}},
		nil,
	)
	if !strings.Contains(page, "n/a") || !strings.Contains(page, "Replay") {
		t.Fatalf("unexpected page:\n%s", page)
	}
}
