package evaluation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/scheduler"
)

func boolRef(v bool) *bool { return &v }

func TestBenchmarkRunnerScoresAndReplaysCase(t *testing.T) {
	dir := t.TempDir()
	runner := Runner{StorageDir: dir}
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
		ExpectedExecutor: domain.ExecutorKubernetes,
		ExpectAccepted:   boolRef(true),
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
		filepath.Join(dir, "browser-low-risk", "task-run.md"),
		filepath.Join(dir, "benchmark-browser-low-risk", "replay.html"),
		filepath.Join(dir, "browser-low-risk", "run-detail.html"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected artifact %s: %v", path, err)
		}
	}
	if result.DetailPagePath != filepath.Join(dir, "browser-low-risk", "run-detail.html") {
		t.Fatalf("unexpected detail page path: %s", result.DetailPagePath)
	}
}

func TestBenchmarkRunnerDetectsFailedExpectation(t *testing.T) {
	runner := Runner{StorageDir: t.TempDir()}
	result, err := runner.RunCase(BenchmarkCase{
		CaseID: "high-risk-gate",
		Task: domain.Task{
			ID:          "BIG-601-risk",
			Source:      "jira",
			Title:       "Prod change benchmark",
			Description: "must stop for approval",
			RiskLevel:   domain.RiskHigh,
		},
		ExpectedExecutor: domain.ExecutorLocal,
		ExpectAccepted:   boolRef(true),
		ExpectedStatus:   "approved",
	})
	if err != nil {
		t.Fatalf("run case: %v", err)
	}
	if result.Passed || result.Score != 60 {
		t.Fatalf("unexpected failed benchmark result: %+v", result)
	}
	found := false
	for _, item := range result.Criteria {
		if item.Name == "decision-executor" && !item.Passed {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected failed executor criterion, got %+v", result.Criteria)
	}
}

func TestReplayOutcomeReportsMismatch(t *testing.T) {
	runner := Runner{Scheduler: scheduler.New(), StorageDir: t.TempDir()}
	outcome, err := runner.Replay(ReplayRecord{
		Task: domain.Task{
			ID:            "BIG-601-replay",
			Source:        "github",
			Title:         "Replay browser route",
			Description:   "compare deterministic scheduler behavior",
			RequiredTools: []string{"browser"},
		},
		RunID:    "run-1",
		Executor: domain.ExecutorLocal,
		Accepted: true,
		Status:   "approved",
	})
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if outcome.Matched {
		t.Fatalf("expected mismatch replay outcome, got %+v", outcome)
	}
	if len(outcome.Mismatches) == 0 || outcome.Mismatches[0] != "executor expected local got kubernetes" {
		t.Fatalf("unexpected mismatches: %+v", outcome.Mismatches)
	}
	if outcome.ReportPath == "" {
		t.Fatalf("expected replay report path, got %+v", outcome)
	}
	if _, err := os.Stat(outcome.ReportPath); err != nil {
		t.Fatalf("expected replay report artifact: %v", err)
	}
}

func TestSuiteComparisonAndReport(t *testing.T) {
	runner := Runner{StorageDir: t.TempDir()}
	suite, err := runner.RunSuite([]BenchmarkCase{
		{
			CaseID: "browser-low-risk",
			Task: domain.Task{
				ID:            "BIG-601-v2",
				Source:        "linear",
				Title:         "Run browser benchmark",
				Description:   "validate routing",
				RequiredTools: []string{"browser"},
			},
			ExpectedExecutor: domain.ExecutorKubernetes,
			ExpectAccepted:   boolRef(true),
			ExpectedStatus:   "approved",
		},
	}, "v0.2")
	if err != nil {
		t.Fatalf("run suite: %v", err)
	}
	baseline := BenchmarkSuiteResult{Results: nil, Version: "v0.1"}
	comparison := suite.Compare(baseline)
	report := RenderBenchmarkSuiteReport(suite, &baseline)
	if comparison[0].Delta != 100 || suite.Score() != 100 {
		t.Fatalf("unexpected suite comparison: %+v suite=%+v", comparison, suite)
	}
	for _, want := range []string{"Version: v0.2", "Baseline Version: v0.1", "Score Delta: 100"} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected report to contain %q, got %q", want, report)
		}
	}
}

func TestRenderReplayDetailPageListsMismatches(t *testing.T) {
	task := domain.Task{ID: "BIG-804", Source: "linear", Title: "Replay detail"}
	page := RenderReplayDetailPage(
		ReplayRecord{Task: task, RunID: "run-1", Executor: domain.ExecutorLocal, Accepted: true, Status: "approved"},
		ReplayRecord{Task: task, RunID: "run-1", Executor: domain.ExecutorKubernetes, Accepted: false, Status: "needs-approval"},
		[]string{"executor expected local got kubernetes", "accepted expected true got false"},
	)
	for _, want := range []string{"Replay Detail", "Timeline / Log Sync", "Split View", "Reports", "executor expected local got kubernetes", "needs-approval"} {
		if !strings.Contains(page, want) {
			t.Fatalf("expected page to contain %q, got %q", want, page)
		}
	}
}

func TestRenderRunReplayIndexPageLinksOutputs(t *testing.T) {
	dir := t.TempDir()
	runner := Runner{StorageDir: dir}
	result, err := runner.RunCase(BenchmarkCase{
		CaseID: "big-804-index",
		Task: domain.Task{
			ID:            "BIG-804",
			Source:        "linear",
			Title:         "Run detail index",
			Description:   "single landing page",
			RequiredTools: []string{"browser"},
		},
		ExpectedExecutor: domain.ExecutorKubernetes,
		ExpectAccepted:   boolRef(true),
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
	for _, want := range []string{"Run Detail Index", "Timeline / Log Sync", "Acceptance", "Reports", "task-run.md", "replay.html", "decision-executor"} {
		if !strings.Contains(string(page), want) {
			t.Fatalf("expected detail page to contain %q, got %q", want, string(page))
		}
	}
}

func TestRenderRunReplayIndexPageWithoutReportPath(t *testing.T) {
	task := domain.Task{ID: "BIG-804", Source: "linear", Title: "Run detail index"}
	page := RenderRunReplayIndexPage(
		"big-804-index",
		ReplayRecord{Task: task, RunID: "run-1", Executor: domain.ExecutorLocal, Accepted: true, Status: "approved"},
		ReplayOutcome{
			Matched:      true,
			ReplayRecord: ReplayRecord{Task: task, RunID: "run-1", Executor: domain.ExecutorLocal, Accepted: true, Status: "approved"},
			ReportPath:   "",
		},
		nil,
		"",
	)
	for _, want := range []string{"n/a", "Replay"} {
		if !strings.Contains(page, want) {
			t.Fatalf("expected page to contain %q, got %q", want, page)
		}
	}
}
