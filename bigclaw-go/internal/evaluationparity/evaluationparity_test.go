package evaluationparity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/executionparity"
)

func boolRef(value bool) *bool { return &value }

func TestBenchmarkRunnerScoresAndReplaysCase(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	runner := BenchmarkRunner{StorageDir: dir}
	result, err := runner.RunCase(BenchmarkCase{
		CaseID: "browser-low-risk",
		Task: executionparity.Task{
			ID:            "BIG-601",
			Source:        "linear",
			Title:         "Run browser benchmark",
			Description:   "validate routing",
			RiskLevel:     executionparity.RiskLow,
			RequiredTools: []string{"browser"},
		},
		ExpectedMedium:   "browser",
		ExpectedApproved: boolRef(true),
		ExpectedStatus:   "approved",
		RequireReport:    true,
	})
	if err != nil {
		t.Fatalf("run case: %v", err)
	}

	if result.Score != 100 || !result.Passed || !result.Replay.Matched {
		t.Fatalf("unexpected result: %+v", result)
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
		t.Fatalf("detail page path = %q", result.DetailPagePath)
	}
}

func TestBenchmarkRunnerDetectsFailedExpectation(t *testing.T) {
	t.Parallel()

	runner := BenchmarkRunner{StorageDir: t.TempDir()}
	result, err := runner.RunCase(BenchmarkCase{
		CaseID: "high-risk-gate",
		Task: executionparity.Task{
			ID:          "BIG-601-risk",
			Source:      "jira",
			Title:       "Prod change benchmark",
			Description: "must stop for approval",
			RiskLevel:   executionparity.RiskHigh,
		},
		ExpectedMedium:   "docker",
		ExpectedApproved: boolRef(false),
		ExpectedStatus:   "needs-approval",
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
		t.Fatalf("expected failed decision-medium criterion: %+v", result.Criteria)
	}
}

func TestReplayOutcomeReportsMismatch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	runner := BenchmarkRunner{Scheduler: executionparity.Scheduler{}, StorageDir: dir}
	outcome, err := runner.Replay(ReplayRecord{
		Task: executionparity.Task{
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
		t.Fatalf("expected mismatch outcome: %+v", outcome)
	}
	if len(outcome.Mismatches) != 1 || outcome.Mismatches[0] != "medium expected docker got browser" {
		t.Fatalf("unexpected mismatches: %+v", outcome.Mismatches)
	}
	if outcome.ReportPath == "" {
		t.Fatal("expected replay report path")
	}
	if _, err := os.Stat(outcome.ReportPath); err != nil {
		t.Fatalf("expected replay report: %v", err)
	}
}

func TestSuiteComparisonAndReport(t *testing.T) {
	t.Parallel()

	runner := BenchmarkRunner{StorageDir: t.TempDir()}
	improved, err := runner.RunSuite([]BenchmarkCase{{
		CaseID: "browser-low-risk",
		Task: executionparity.Task{
			ID:            "BIG-601-v2",
			Source:        "linear",
			Title:         "Run browser benchmark",
			Description:   "validate routing",
			RequiredTools: []string{"browser"},
		},
		ExpectedMedium:   "browser",
		ExpectedApproved: boolRef(true),
		ExpectedStatus:   "approved",
	}}, "v0.2")
	if err != nil {
		t.Fatalf("run suite: %v", err)
	}
	baseline := BenchmarkSuiteResult{Results: nil, Version: "v0.1"}

	comparison := improved.Compare(baseline)
	report := RenderBenchmarkSuiteReport(improved, &baseline)

	if len(comparison) != 1 || comparison[0].Delta != 100 {
		t.Fatalf("unexpected comparison: %+v", comparison)
	}
	if improved.Score() != 100 {
		t.Fatalf("suite score = %d", improved.Score())
	}
	for _, fragment := range []string{"Version: v0.2", "Baseline Version: v0.1", "Score Delta: 100"} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}

func TestRenderReplayDetailPageListsMismatches(t *testing.T) {
	t.Parallel()

	task := executionparity.Task{ID: "BIG-804", Source: "linear", Title: "Replay detail", Description: ""}
	expected := ReplayRecord{Task: task, RunID: "run-1", Medium: "docker", Approved: true, Status: "approved"}
	observed := ReplayRecord{Task: task, RunID: "run-1", Medium: "browser", Approved: false, Status: "needs-approval"}

	page := RenderReplayDetailPage(expected, observed, []string{"medium expected docker got browser", "approved expected true got false"})
	for _, fragment := range []string{"Replay Detail", "Timeline / Log Sync", "Split View", "Reports", "medium expected docker got browser", "needs-approval"} {
		if !strings.Contains(page, fragment) {
			t.Fatalf("expected %q in page, got %s", fragment, page)
		}
	}
}

func TestRenderRunReplayIndexPageLinksOutputs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	runner := BenchmarkRunner{StorageDir: dir}
	result, err := runner.RunCase(BenchmarkCase{
		CaseID: "big-804-index",
		Task: executionparity.Task{
			ID:            "BIG-804",
			Source:        "linear",
			Title:         "Run detail index",
			Description:   "single landing page",
			RequiredTools: []string{"browser"},
		},
		ExpectedMedium:   "browser",
		ExpectedApproved: boolRef(true),
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
	for _, fragment := range []string{"Run Detail Index", "Timeline / Log Sync", "Acceptance", "Reports", "task-run.md", "replay.html", "decision-medium"} {
		if !strings.Contains(page, fragment) {
			t.Fatalf("expected %q in page, got %s", fragment, page)
		}
	}
}

func TestRenderRunReplayIndexPageWithoutReportPath(t *testing.T) {
	t.Parallel()

	task := executionparity.Task{ID: "BIG-804", Source: "linear", Title: "Run detail index", Description: ""}
	replay := ReplayOutcome{
		Matched:      true,
		ReplayRecord: ReplayRecord{Task: task, RunID: "run-1", Medium: "docker", Approved: true, Status: "approved"},
	}
	record, err := (executionparity.Scheduler{}).Execute(
		task,
		"run-1",
		&executionparity.Ledger{Path: filepath.Join(t.TempDir(), "ledger.json")},
		"",
	)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	page := RenderRunReplayIndexPage("big-804-index", record, replay, nil)
	if !strings.Contains(page, "n/a") || !strings.Contains(page, "Replay") {
		t.Fatalf("unexpected page: %s", page)
	}
}
