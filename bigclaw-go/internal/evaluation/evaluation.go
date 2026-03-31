package evaluation

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"bigclaw-go/internal/domain"
)

type BenchmarkCase struct {
	CaseID           string
	Task             domain.Task
	ExpectedMedium   string
	ExpectedApproved bool
	ExpectedStatus   string
	RequireReport    bool
}

type CriterionResult struct {
	Name     string
	Passed   bool
	Expected string
	Observed string
}

type ReplayRecord struct {
	Task     domain.Task
	RunID    string
	Medium   string
	Approved bool
	Status   string
}

type ReplayOutcome struct {
	Matched      bool
	ReplayRecord ReplayRecord
	Mismatches   []string
	ReportPath   string
}

type BenchmarkResult struct {
	CaseID         string
	Score          int
	Passed         bool
	Criteria       []CriterionResult
	Replay         ReplayOutcome
	DetailPagePath string
}

type BenchmarkSuiteResult struct {
	Results []BenchmarkResult
	Version string
	Score   int
}

type SuiteComparison struct {
	Label    string
	Delta    int
	Current  int
	Baseline int
}

type BenchmarkRunner struct {
	StorageDir string
}

func NewBenchmarkRunner(storageDir string) BenchmarkRunner {
	return BenchmarkRunner{StorageDir: storageDir}
}

func (r BenchmarkRunner) RunCase(benchmarkCase BenchmarkCase) (BenchmarkResult, error) {
	runRecord := deriveReplayRecord(benchmarkCase.Task, benchmarkCase.CaseID)
	taskDir := filepath.Join(r.StorageDir, benchmarkCase.CaseID)
	benchmarkDir := filepath.Join(r.StorageDir, "benchmark-"+benchmarkCase.CaseID)
	taskRunPath := filepath.Join(taskDir, "task-run.md")
	detailPath := filepath.Join(taskDir, "run-detail.html")
	replayPath := filepath.Join(benchmarkDir, "replay.html")

	criteria := []CriterionResult{
		buildCriterion("decision-medium", benchmarkCase.ExpectedMedium, runRecord.Medium),
		buildCriterion("approved", fmt.Sprintf("%t", benchmarkCase.ExpectedApproved), fmt.Sprintf("%t", runRecord.Approved)),
		buildCriterion("status", benchmarkCase.ExpectedStatus, runRecord.Status),
	}

	replayOutcome, err := r.replayWithPath(ReplayRecord{
		Task:     benchmarkCase.Task,
		RunID:    runRecord.RunID,
		Medium:   benchmarkCase.ExpectedMedium,
		Approved: benchmarkCase.ExpectedApproved,
		Status:   benchmarkCase.ExpectedStatus,
	}, replayPath)
	if err != nil {
		return BenchmarkResult{}, err
	}
	criteria = append(criteria, CriterionResult{
		Name:     "replay-match",
		Passed:   replayOutcome.Matched,
		Expected: "true",
		Observed: fmt.Sprintf("%t", replayOutcome.Matched),
	})

	reportObserved := "false"
	reportPassed := !benchmarkCase.RequireReport
	if benchmarkCase.RequireReport {
		reportPassed = replayOutcome.ReportPath != ""
		reportObserved = fmt.Sprintf("%t", reportPassed)
	}
	criteria = append(criteria, CriterionResult{
		Name:     "report-generated",
		Passed:   reportPassed,
		Expected: fmt.Sprintf("%t", benchmarkCase.RequireReport),
		Observed: reportObserved,
	})

	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		return BenchmarkResult{}, err
	}
	if err := os.WriteFile(taskRunPath, []byte(renderTaskRunMarkdown(benchmarkCase, runRecord, criteria)), 0o644); err != nil {
		return BenchmarkResult{}, err
	}
	if err := os.WriteFile(detailPath, []byte(RenderRunReplayIndexPage(benchmarkCase.CaseID, taskRunPath, replayOutcome, criteria)), 0o644); err != nil {
		return BenchmarkResult{}, err
	}

	score := scoreCriteria(criteria)
	return BenchmarkResult{
		CaseID:         benchmarkCase.CaseID,
		Score:          score,
		Passed:         score == 100,
		Criteria:       criteria,
		Replay:         replayOutcome,
		DetailPagePath: detailPath,
	}, nil
}

func (r BenchmarkRunner) Replay(expected ReplayRecord) (ReplayOutcome, error) {
	reportPath := ""
	if r.StorageDir != "" {
		reportPath = filepath.Join(r.StorageDir, "benchmark-"+expected.Task.ID, "replay.html")
	}
	return r.replayWithPath(expected, reportPath)
}

func (r BenchmarkRunner) replayWithPath(expected ReplayRecord, reportPath string) (ReplayOutcome, error) {
	observed := deriveReplayRecord(expected.Task, expected.RunID)
	mismatches := replayMismatches(expected, observed)
	outcome := ReplayOutcome{
		Matched:      len(mismatches) == 0,
		ReplayRecord: observed,
		Mismatches:   mismatches,
	}
	if strings.TrimSpace(reportPath) == "" {
		return outcome, nil
	}
	if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
		return ReplayOutcome{}, err
	}
	if err := os.WriteFile(reportPath, []byte(RenderReplayDetailPage(expected, observed, mismatches)), 0o644); err != nil {
		return ReplayOutcome{}, err
	}
	outcome.ReportPath = reportPath
	return outcome, nil
}

func (r BenchmarkRunner) RunSuite(cases []BenchmarkCase, version string) (BenchmarkSuiteResult, error) {
	results := make([]BenchmarkResult, 0, len(cases))
	for _, benchmarkCase := range cases {
		result, err := r.RunCase(benchmarkCase)
		if err != nil {
			return BenchmarkSuiteResult{}, err
		}
		results = append(results, result)
	}
	suite := BenchmarkSuiteResult{Results: results, Version: version}
	suite.Score = averageScore(results)
	return suite, nil
}

func (s BenchmarkSuiteResult) Compare(baseline BenchmarkSuiteResult) []SuiteComparison {
	return []SuiteComparison{{
		Label:    "suite-score",
		Delta:    s.Score - baseline.Score,
		Current:  s.Score,
		Baseline: baseline.Score,
	}}
}

func RenderBenchmarkSuiteReport(current, baseline BenchmarkSuiteResult) string {
	comparison := current.Compare(baseline)
	return strings.Join([]string{
		"Benchmark Suite Report",
		fmt.Sprintf("Version: %s", current.Version),
		fmt.Sprintf("Baseline Version: %s", baseline.Version),
		fmt.Sprintf("Score Delta: %d", comparison[0].Delta),
	}, "\n")
}

func RenderReplayDetailPage(expected, observed ReplayRecord, mismatches []string) string {
	var mismatchLines []string
	if len(mismatches) == 0 {
		mismatchLines = []string{"<li>none</li>"}
	} else {
		for _, mismatch := range mismatches {
			mismatchLines = append(mismatchLines, "<li>"+html.EscapeString(mismatch)+"</li>")
		}
	}
	return strings.Join([]string{
		"<html><body>",
		"<h1>Replay Detail</h1>",
		"<section>Timeline / Log Sync</section>",
		"<section>Split View</section>",
		"<section>Reports</section>",
		fmt.Sprintf("<div>Expected Status: %s</div>", html.EscapeString(expected.Status)),
		fmt.Sprintf("<div>Observed Status: %s</div>", html.EscapeString(observed.Status)),
		"<ul>",
		strings.Join(mismatchLines, ""),
		"</ul>",
		"</body></html>",
	}, "\n")
}

func RenderRunReplayIndexPage(caseID, taskRunPath string, replay ReplayOutcome, criteria []CriterionResult) string {
	replayRef := "n/a"
	if replay.ReportPath != "" {
		replayRef = filepath.Base(replay.ReportPath)
	}

	rows := make([]string, 0, len(criteria))
	for _, criterion := range criteria {
		rows = append(rows, fmt.Sprintf(
			"<li>%s: %t (%s -> %s)</li>",
			html.EscapeString(criterion.Name),
			criterion.Passed,
			html.EscapeString(criterion.Expected),
			html.EscapeString(criterion.Observed),
		))
	}

	return strings.Join([]string{
		"<html><body>",
		"<h1>Run Detail Index</h1>",
		fmt.Sprintf("<div>Case: %s</div>", html.EscapeString(caseID)),
		"<section>Timeline / Log Sync</section>",
		"<section>Acceptance</section>",
		"<section>Reports</section>",
		fmt.Sprintf("<div>Run Artifact: %s</div>", html.EscapeString(filepath.Base(taskRunPath))),
		fmt.Sprintf("<div>Replay: %s</div>", html.EscapeString(replayRef)),
		"<ul>",
		strings.Join(rows, ""),
		"</ul>",
		"</body></html>",
	}, "\n")
}

func deriveReplayRecord(task domain.Task, runID string) ReplayRecord {
	return ReplayRecord{
		Task:     task,
		RunID:    strings.TrimSpace(runID),
		Medium:   deriveMedium(task),
		Approved: deriveApproved(task),
		Status:   deriveStatus(task),
	}
}

func deriveMedium(task domain.Task) string {
	for _, tool := range task.RequiredTools {
		if strings.EqualFold(strings.TrimSpace(tool), "browser") {
			return "browser"
		}
	}
	if task.RiskLevel == domain.RiskHigh {
		return "kubernetes"
	}
	return "docker"
}

func deriveApproved(task domain.Task) bool {
	return task.RiskLevel != domain.RiskHigh
}

func deriveStatus(task domain.Task) string {
	if deriveApproved(task) {
		return "approved"
	}
	return "needs-approval"
}

func buildCriterion(name, expected, observed string) CriterionResult {
	return CriterionResult{
		Name:     name,
		Passed:   expected == observed,
		Expected: expected,
		Observed: observed,
	}
}

func replayMismatches(expected, observed ReplayRecord) []string {
	var mismatches []string
	if expected.Medium != observed.Medium {
		mismatches = append(mismatches, fmt.Sprintf("medium expected %s got %s", expected.Medium, observed.Medium))
	}
	if expected.Approved != observed.Approved {
		mismatches = append(mismatches, fmt.Sprintf("approved expected %t got %t", expected.Approved, observed.Approved))
	}
	if expected.Status != observed.Status {
		mismatches = append(mismatches, fmt.Sprintf("status expected %s got %s", expected.Status, observed.Status))
	}
	return mismatches
}

func scoreCriteria(criteria []CriterionResult) int {
	if len(criteria) == 0 {
		return 0
	}
	passed := 0
	for _, criterion := range criteria {
		if criterion.Passed {
			passed++
		}
	}
	return (passed * 100) / len(criteria)
}

func averageScore(results []BenchmarkResult) int {
	if len(results) == 0 {
		return 0
	}
	total := 0
	for _, result := range results {
		total += result.Score
	}
	return total / len(results)
}

func renderTaskRunMarkdown(benchmarkCase BenchmarkCase, record ReplayRecord, criteria []CriterionResult) string {
	lines := []string{
		fmt.Sprintf("# Task Run %s", benchmarkCase.CaseID),
		fmt.Sprintf("- task_id: %s", benchmarkCase.Task.ID),
		fmt.Sprintf("- medium: %s", record.Medium),
		fmt.Sprintf("- approved: %t", record.Approved),
		fmt.Sprintf("- status: %s", record.Status),
	}
	slices.SortFunc(criteria, func(left, right CriterionResult) int {
		return strings.Compare(left.Name, right.Name)
	})
	for _, criterion := range criteria {
		lines = append(lines, fmt.Sprintf("- %s: %t", criterion.Name, criterion.Passed))
	}
	return strings.Join(lines, "\n") + "\n"
}
