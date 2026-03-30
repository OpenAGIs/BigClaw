package evaluation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/domain"
)

func TestBenchmarkRunnerScoresAndReplaysCase(t *testing.T) {
	dir := t.TempDir()
	runner := BenchmarkRunner{StorageDir: dir}
	caseInput := BenchmarkCase{
		CaseID: "browser-low-risk",
		Task: domain.Task{
			ID:            "BIG-601",
			Source:        "linear",
			Title:         "Run browser benchmark",
			Description:   "validate routing",
			RiskLevel:     domain.RiskLow,
			RequiredTools: []string{"browser"},
		},
		ExpectedMedium:   stringPtr("browser"),
		ExpectedApproved: boolPtr(true),
		ExpectedStatus:   stringPtr("approved"),
		RequireReport:    true,
	}

	result := runner.RunCase(caseInput)
	if result.Score != 100 || !result.Passed || !result.Replay.Matched {
		t.Fatalf("unexpected result: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(dir, "browser-low-risk", "task-run.md")); err != nil {
		t.Fatalf("expected task-run report: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "benchmark-browser-low-risk", "replay.html")); err != nil {
		t.Fatalf("expected replay page: %v", err)
	}
	if result.DetailPagePath != filepath.Join(dir, "browser-low-risk", "run-detail.html") {
		t.Fatalf("unexpected detail page path: %s", result.DetailPagePath)
	}
}

func TestBenchmarkRunnerDetectsFailedExpectation(t *testing.T) {
	runner := BenchmarkRunner{StorageDir: t.TempDir()}
	caseInput := BenchmarkCase{
		CaseID: "high-risk-gate",
		Task: domain.Task{
			ID:          "BIG-601-risk",
			Source:      "jira",
			Title:       "Prod change benchmark",
			Description: "must stop for approval",
			RiskLevel:   domain.RiskHigh,
		},
		ExpectedMedium:   stringPtr("docker"),
		ExpectedApproved: boolPtr(false),
		ExpectedStatus:   stringPtr("needs-approval"),
	}

	result := runner.RunCase(caseInput)
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
		t.Fatalf("expected failed decision-medium criterion, got %+v", result.Criteria)
	}
}

func TestReplayOutcomeReportsMismatch(t *testing.T) {
	runner := BenchmarkRunner{Scheduler: Scheduler{}, StorageDir: t.TempDir()}
	replayRecord := ReplayRecord{
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
	}

	outcome := runner.Replay(replayRecord)
	if outcome.Matched || len(outcome.Mismatches) != 1 || outcome.Mismatches[0] != "medium expected docker got browser" {
		t.Fatalf("unexpected outcome: %+v", outcome)
	}
	if strings.TrimSpace(outcome.ReportPath) == "" {
		t.Fatalf("expected replay report path, got %+v", outcome)
	}
	if _, err := os.Stat(outcome.ReportPath); err != nil {
		t.Fatalf("expected replay report file: %v", err)
	}
}

func TestSuiteComparisonAndReport(t *testing.T) {
	runner := BenchmarkRunner{StorageDir: t.TempDir()}
	improved := runner.RunSuite([]BenchmarkCase{{
		CaseID: "browser-low-risk",
		Task: domain.Task{
			ID:            "BIG-601-v2",
			Source:        "linear",
			Title:         "Run browser benchmark",
			Description:   "validate routing",
			RequiredTools: []string{"browser"},
		},
		ExpectedMedium:   stringPtr("browser"),
		ExpectedApproved: boolPtr(true),
		ExpectedStatus:   stringPtr("approved"),
	}}, "v0.2")
	baseline := BenchmarkSuiteResult{Results: nil, Version: "v0.1"}

	comparison := improved.Compare(baseline)
	report := RenderBenchmarkSuiteReport(improved, &baseline)
	if comparison[0].Delta != 100 || improved.Score() != 100 {
		t.Fatalf("unexpected comparison/suite score: comparison=%+v score=%d", comparison, improved.Score())
	}
	for _, fragment := range []string{"Version: v0.2", "Baseline Version: v0.1", "Score Delta: 100"} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}

func TestRenderReplayDetailPageListsMismatches(t *testing.T) {
	task := domain.Task{ID: "BIG-804", Source: "linear", Title: "Replay detail"}
	expected := ReplayRecord{Task: task, RunID: "run-1", Medium: "docker", Approved: true, Status: "approved"}
	observed := ReplayRecord{Task: task, RunID: "run-1", Medium: "browser", Approved: false, Status: "needs-approval"}
	page := RenderReplayDetailPage(expected, observed, []string{"medium expected docker got browser", "approved expected True got False"})
	for _, fragment := range []string{"Replay Detail", "Timeline / Log Sync", "Split View", "Reports", "medium expected docker got browser", "needs-approval"} {
		if !strings.Contains(page, fragment) {
			t.Fatalf("expected %q in page, got %s", fragment, page)
		}
	}
}

func TestRenderRunReplayIndexPageLinksOutputs(t *testing.T) {
	dir := t.TempDir()
	runner := BenchmarkRunner{StorageDir: dir}
	caseInput := BenchmarkCase{
		CaseID: "big-804-index",
		Task: domain.Task{
			ID:            "BIG-804",
			Source:        "linear",
			Title:         "Run detail index",
			Description:   "single landing page",
			RequiredTools: []string{"browser"},
		},
		ExpectedMedium:   stringPtr("browser"),
		ExpectedApproved: boolPtr(true),
		ExpectedStatus:   stringPtr("approved"),
		RequireReport:    true,
	}

	result := runner.RunCase(caseInput)
	body, err := os.ReadFile(result.DetailPagePath)
	if err != nil {
		t.Fatalf("read detail page: %v", err)
	}
	page := string(body)
	for _, fragment := range []string{"Run Detail Index", "Timeline / Log Sync", "Acceptance", "Reports", "task-run.md", "replay.html", "decision-medium"} {
		if !strings.Contains(page, fragment) {
			t.Fatalf("expected %q in page, got %s", fragment, page)
		}
	}
}

func TestRenderRunReplayIndexPageWithoutReportPath(t *testing.T) {
	task := domain.Task{ID: "BIG-804", Source: "linear", Title: "Run detail index"}
	replay := ReplayOutcome{
		Matched: true,
		ReplayRecord: ReplayRecord{
			Task: task, RunID: "run-1", Medium: "docker", Approved: true, Status: "approved",
		},
	}
	record := Scheduler{}.Execute(task, "run-1", "")
	page := RenderRunReplayIndexPage("big-804-index", record, replay, nil)
	for _, fragment := range []string{"n/a", "Replay"} {
		if !strings.Contains(page, fragment) {
			t.Fatalf("expected %q in page, got %s", fragment, page)
		}
	}
}
