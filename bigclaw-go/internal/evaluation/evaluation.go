package evaluation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/scheduler"
)

type Criterion struct {
	Name   string
	Weight int
	Passed bool
	Detail string
}

type BenchmarkCase struct {
	CaseID           string
	Task             domain.Task
	ExpectedExecutor domain.ExecutorKind
	ExpectAccepted   *bool
	ExpectedStatus   string
	RequireReport    bool
}

type ReplayRecord struct {
	Task     domain.Task
	RunID    string
	Executor domain.ExecutorKind
	Accepted bool
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
	Criteria       []Criterion
	Replay         ReplayOutcome
	ReportPath     string
	DetailPagePath string
}

type BenchmarkComparison struct {
	CaseID        string
	BaselineScore int
	CurrentScore  int
	Delta         int
	Changed       bool
}

type BenchmarkSuiteResult struct {
	Results []BenchmarkResult
	Version string
}

func (suite BenchmarkSuiteResult) Score() int {
	if len(suite.Results) == 0 {
		return 0
	}
	total := 0
	for _, result := range suite.Results {
		total += result.Score
	}
	return int(float64(total)/float64(len(suite.Results)) + 0.5)
}

func (suite BenchmarkSuiteResult) Passed() bool {
	for _, result := range suite.Results {
		if !result.Passed {
			return false
		}
	}
	return true
}

func (suite BenchmarkSuiteResult) Compare(baseline BenchmarkSuiteResult) []BenchmarkComparison {
	byCase := map[string]BenchmarkResult{}
	for _, result := range baseline.Results {
		byCase[result.CaseID] = result
	}
	comparisons := make([]BenchmarkComparison, 0, len(suite.Results))
	for _, result := range suite.Results {
		baselineScore := 0
		if existing, ok := byCase[result.CaseID]; ok {
			baselineScore = existing.Score
		}
		delta := result.Score - baselineScore
		comparisons = append(comparisons, BenchmarkComparison{
			CaseID:        result.CaseID,
			BaselineScore: baselineScore,
			CurrentScore:  result.Score,
			Delta:         delta,
			Changed:       delta != 0,
		})
	}
	return comparisons
}

type Runner struct {
	Scheduler  *scheduler.Scheduler
	StorageDir string
}

func (runner Runner) RunCase(tc BenchmarkCase) (BenchmarkResult, error) {
	runtime := runner.scheduler()
	runID := "benchmark-" + tc.CaseID
	decision := runtime.Decide(tc.Task, scheduler.QuotaSnapshot{})
	record := ReplayRecord{
		Task:     tc.Task,
		RunID:    runID,
		Executor: decision.Assignment.Executor,
		Accepted: decision.Accepted,
		Status:   decisionStatus(decision.Accepted),
	}

	reportPath := ""
	if tc.RequireReport {
		reportPath = runner.casePath(tc.CaseID, "task-run.md")
		if err := writeText(reportPath, renderTaskRunMarkdown(record)); err != nil {
			return BenchmarkResult{}, err
		}
	}

	criteria := evaluateCriteria(tc, record, reportPath)
	replay, err := runner.Replay(record)
	if err != nil {
		return BenchmarkResult{}, err
	}
	score := scoreCriteria(criteria)
	passed := replay.Matched
	for _, criterion := range criteria {
		if !criterion.Passed {
			passed = false
			break
		}
	}

	detailPath := ""
	if runner.StorageDir != "" {
		detailPath = runner.casePath(tc.CaseID, "run-detail.html")
		if err := writeText(detailPath, RenderRunReplayIndexPage(tc.CaseID, record, replay, criteria, reportPath)); err != nil {
			return BenchmarkResult{}, err
		}
	}

	return BenchmarkResult{
		CaseID:         tc.CaseID,
		Score:          score,
		Passed:         passed,
		Criteria:       criteria,
		Replay:         replay,
		ReportPath:     reportPath,
		DetailPagePath: detailPath,
	}, nil
}

func (runner Runner) RunSuite(cases []BenchmarkCase, version string) (BenchmarkSuiteResult, error) {
	results := make([]BenchmarkResult, 0, len(cases))
	for _, tc := range cases {
		result, err := runner.RunCase(tc)
		if err != nil {
			return BenchmarkSuiteResult{}, err
		}
		results = append(results, result)
	}
	return BenchmarkSuiteResult{Results: results, Version: version}, nil
}

func (runner Runner) Replay(record ReplayRecord) (ReplayOutcome, error) {
	runtime := runner.scheduler()
	decision := runtime.Decide(record.Task, scheduler.QuotaSnapshot{})
	observed := ReplayRecord{
		Task:     record.Task,
		RunID:    record.RunID,
		Executor: decision.Assignment.Executor,
		Accepted: decision.Accepted,
		Status:   decisionStatus(decision.Accepted),
	}

	mismatches := []string{}
	if observed.Executor != record.Executor {
		mismatches = append(mismatches, fmt.Sprintf("executor expected %s got %s", record.Executor, observed.Executor))
	}
	if observed.Accepted != record.Accepted {
		mismatches = append(mismatches, fmt.Sprintf("accepted expected %t got %t", record.Accepted, observed.Accepted))
	}
	if observed.Status != record.Status {
		mismatches = append(mismatches, fmt.Sprintf("status expected %s got %s", record.Status, observed.Status))
	}

	reportPath := ""
	if runner.StorageDir != "" {
		reportPath = runner.casePath(record.RunID, "replay.html")
		if err := writeText(reportPath, RenderReplayDetailPage(record, observed, mismatches)); err != nil {
			return ReplayOutcome{}, err
		}
	}

	return ReplayOutcome{
		Matched:      len(mismatches) == 0,
		ReplayRecord: observed,
		Mismatches:   mismatches,
		ReportPath:   reportPath,
	}, nil
}

func RenderBenchmarkSuiteReport(suite BenchmarkSuiteResult, baseline *BenchmarkSuiteResult) string {
	lines := []string{
		"# Benchmark Suite Report",
		"",
		fmt.Sprintf("- Version: %s", suite.Version),
		fmt.Sprintf("- Cases: %d", len(suite.Results)),
		fmt.Sprintf("- Passed: %t", suite.Passed()),
		fmt.Sprintf("- Score: %d", suite.Score()),
		"",
		"## Cases",
		"",
	}
	if len(suite.Results) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, result := range suite.Results {
			lines = append(lines, fmt.Sprintf("- %s: score=%d passed=%t replay=%t", result.CaseID, result.Score, result.Passed, result.Replay.Matched))
		}
	}
	lines = append(lines, "", "## Comparison", "")
	if baseline == nil {
		lines = append(lines, "- No baseline provided")
	} else {
		lines = append(lines, fmt.Sprintf("- Baseline Version: %s", baseline.Version))
		lines = append(lines, fmt.Sprintf("- Score Delta: %d", suite.Score()-baseline.Score()))
		for _, comparison := range suite.Compare(*baseline) {
			lines = append(lines, fmt.Sprintf("- %s: baseline=%d current=%d delta=%d", comparison.CaseID, comparison.BaselineScore, comparison.CurrentScore, comparison.Delta))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderReplayDetailPage(expected, observed ReplayRecord, mismatches []string) string {
	lines := []string{
		"<html><body>",
		"<h1>Replay Detail</h1>",
		"<h2>Timeline / Log Sync</h2>",
		"<h2>Split View</h2>",
		"<h2>Reports</h2>",
		fmt.Sprintf("<p>expected executor=%s observed executor=%s</p>", expected.Executor, observed.Executor),
		fmt.Sprintf("<p>expected status=%s observed status=%s</p>", expected.Status, observed.Status),
	}
	if len(mismatches) == 0 {
		lines = append(lines, "<p>None</p>")
	} else {
		for _, mismatch := range mismatches {
			lines = append(lines, "<p>"+mismatch+"</p>")
		}
	}
	lines = append(lines, "</body></html>")
	return strings.Join(lines, "\n")
}

func RenderRunReplayIndexPage(caseID string, record ReplayRecord, replay ReplayOutcome, criteria []Criterion, reportPath string) string {
	replayPath := replay.ReportPath
	if replayPath == "" {
		replayPath = "n/a"
	}
	if reportPath == "" {
		reportPath = "n/a"
	}
	detailPath := "n/a"
	if reportPath != "n/a" {
		detailPath = strings.TrimSuffix(reportPath, filepath.Ext(reportPath)) + ".html"
	}
	lines := []string{
		"<html><body>",
		"<h1>Run Detail Index</h1>",
		"<h2>Timeline / Log Sync</h2>",
		"<h2>Acceptance</h2>",
		"<h2>Replay</h2>",
		"<h2>Reports</h2>",
		fmt.Sprintf("<p>case=%s task=%s executor=%s status=%s</p>", caseID, record.Task.ID, record.Executor, record.Status),
		fmt.Sprintf("<p>%s</p>", reportPath),
		fmt.Sprintf("<p>%s</p>", detailPath),
		fmt.Sprintf("<p>%s</p>", replayPath),
	}
	for _, criterion := range criteria {
		lines = append(lines, fmt.Sprintf("<p>%s: %s</p>", criterion.Name, criterion.Detail))
	}
	lines = append(lines, "</body></html>")
	return strings.Join(lines, "\n")
}

func evaluateCriteria(tc BenchmarkCase, record ReplayRecord, reportPath string) []Criterion {
	criteria := []Criterion{
		criterionExecutor(tc.ExpectedExecutor, record.Executor),
		criterionAccepted(tc.ExpectAccepted, record.Accepted),
		criterionStatus(tc.ExpectedStatus, record.Status),
	}
	criteria = append(criteria, Criterion{
		Name:   "report-artifact",
		Weight: 10,
		Passed: !tc.RequireReport || reportPath != "",
		Detail: ternary(!tc.RequireReport || reportPath != "", "report emitted", "report missing"),
	})
	return criteria
}

func criterionExecutor(expected, actual domain.ExecutorKind) Criterion {
	if expected == "" {
		return Criterion{Name: "decision-executor", Weight: 40, Passed: true, Detail: "not asserted"}
	}
	return Criterion{Name: "decision-executor", Weight: 40, Passed: expected == actual, Detail: fmt.Sprintf("expected %s got %s", expected, actual)}
}

func criterionAccepted(expected *bool, actual bool) Criterion {
	if expected == nil {
		return Criterion{Name: "approval-gate", Weight: 30, Passed: true, Detail: "not asserted"}
	}
	return Criterion{Name: "approval-gate", Weight: 30, Passed: *expected == actual, Detail: fmt.Sprintf("expected %t got %t", *expected, actual)}
}

func criterionStatus(expected, actual string) Criterion {
	if expected == "" {
		return Criterion{Name: "final-status", Weight: 20, Passed: true, Detail: "not asserted"}
	}
	return Criterion{Name: "final-status", Weight: 20, Passed: expected == actual, Detail: fmt.Sprintf("expected %s got %s", expected, actual)}
}

func scoreCriteria(criteria []Criterion) int {
	totalWeight := 0
	earned := 0
	for _, item := range criteria {
		totalWeight += item.Weight
		if item.Passed {
			earned += item.Weight
		}
	}
	if totalWeight == 0 {
		return 0
	}
	return int(float64(earned)/float64(totalWeight)*100 + 0.5)
}

func decisionStatus(accepted bool) string {
	if accepted {
		return "approved"
	}
	return "needs-approval"
}

func (runner Runner) scheduler() *scheduler.Scheduler {
	if runner.Scheduler != nil {
		return runner.Scheduler
	}
	return scheduler.New()
}

func (runner Runner) casePath(caseID, name string) string {
	if runner.StorageDir == "" {
		return name
	}
	return filepath.Join(runner.StorageDir, caseID, name)
}

func renderTaskRunMarkdown(record ReplayRecord) string {
	return fmt.Sprintf("# Task Run\n\n- Task: %s\n- Executor: %s\n- Status: %s\n", record.Task.ID, record.Executor, record.Status)
}

func writeText(path, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), 0o644)
}

func ternary[T any](condition bool, yes, no T) T {
	if condition {
		return yes
	}
	return no
}
