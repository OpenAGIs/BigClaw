package evaluation

import (
	"fmt"
	"html"
	"math"
	"os"
	"path/filepath"
	"strings"

	"bigclaw-go/internal/domain"
)

type EvaluationCriterion struct {
	Name   string `json:"name"`
	Weight int    `json:"weight"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

type BenchmarkCase struct {
	CaseID           string
	Task             domain.Task
	ExpectedMedium   *string
	ExpectedApproved *bool
	ExpectedStatus   *string
	RequireReport    bool
}

type ReplayRecord struct {
	Task     domain.Task `json:"task"`
	RunID    string      `json:"run_id"`
	Medium   string      `json:"medium"`
	Approved bool        `json:"approved"`
	Status   string      `json:"status"`
}

type ReplayOutcome struct {
	Matched      bool         `json:"matched"`
	ReplayRecord ReplayRecord `json:"replay_record"`
	Mismatches   []string     `json:"mismatches,omitempty"`
	ReportPath   string       `json:"report_path,omitempty"`
}

type BenchmarkResult struct {
	CaseID         string                `json:"case_id"`
	Score          int                   `json:"score"`
	Passed         bool                  `json:"passed"`
	Criteria       []EvaluationCriterion `json:"criteria"`
	Record         ExecutionRecord       `json:"record"`
	Replay         ReplayOutcome         `json:"replay"`
	DetailPagePath string                `json:"detail_page_path,omitempty"`
}

type BenchmarkComparison struct {
	CaseID        string `json:"case_id"`
	BaselineScore int    `json:"baseline_score"`
	CurrentScore  int    `json:"current_score"`
	Delta         int    `json:"delta"`
	Changed       bool   `json:"changed"`
}

type BenchmarkSuiteResult struct {
	Results []BenchmarkResult `json:"results"`
	Version string            `json:"version"`
}

type SchedulerDecision struct {
	Medium   string `json:"medium"`
	Approved bool   `json:"approved"`
	Reason   string `json:"reason"`
}

type Run struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`
}

type ExecutionRecord struct {
	Decision   SchedulerDecision `json:"decision"`
	Run        Run               `json:"run"`
	ReportPath string            `json:"report_path,omitempty"`
}

type Scheduler struct{}

type BenchmarkRunner struct {
	Scheduler  Scheduler
	StorageDir string
}

func (s Scheduler) Execute(task domain.Task, runID string, reportPath string) ExecutionRecord {
	decision := s.Decide(task)
	status := "approved"
	if !decision.Approved {
		status = "needs-approval"
	}
	resolvedReportPath := ""
	if strings.TrimSpace(reportPath) != "" {
		resolvedReportPath = reportPath
		_ = os.MkdirAll(filepath.Dir(reportPath), 0o755)
		_ = os.WriteFile(reportPath, []byte("# Task Run Report\n"), 0o644)
		_ = os.WriteFile(strings.TrimSuffix(reportPath, filepath.Ext(reportPath))+".html", []byte("<title>Task Run Detail</title>"), 0o644)
	}
	return ExecutionRecord{
		Decision:   decision,
		Run:        Run{TaskID: task.ID, Status: status},
		ReportPath: resolvedReportPath,
	}
}

func (s Scheduler) Decide(task domain.Task) SchedulerDecision {
	if task.BudgetCents < 0 {
		return SchedulerDecision{Medium: "none", Approved: false, Reason: "invalid budget"}
	}
	if task.RiskLevel == domain.RiskHigh {
		return SchedulerDecision{Medium: "vm", Approved: false, Reason: "requires approval for high-risk task"}
	}
	for _, tool := range task.RequiredTools {
		if tool == "browser" {
			return SchedulerDecision{Medium: "browser", Approved: true, Reason: "browser automation task"}
		}
	}
	if task.RiskLevel == domain.RiskMedium {
		return SchedulerDecision{Medium: "docker", Approved: true, Reason: "medium risk in docker"}
	}
	return SchedulerDecision{Medium: "docker", Approved: true, Reason: "default low risk path"}
}

func (s BenchmarkSuiteResult) Score() int {
	if len(s.Results) == 0 {
		return 0
	}
	total := 0
	for _, result := range s.Results {
		total += result.Score
	}
	return int(math.Round(float64(total) / float64(len(s.Results))))
}

func (s BenchmarkSuiteResult) Passed() bool {
	for _, result := range s.Results {
		if !result.Passed {
			return false
		}
	}
	return true
}

func (s BenchmarkSuiteResult) Compare(baseline BenchmarkSuiteResult) []BenchmarkComparison {
	byCase := make(map[string]BenchmarkResult, len(baseline.Results))
	for _, result := range baseline.Results {
		byCase[result.CaseID] = result
	}
	comparisons := make([]BenchmarkComparison, 0, len(s.Results))
	for _, result := range s.Results {
		baselineResult, ok := byCase[result.CaseID]
		baselineScore := 0
		if ok {
			baselineScore = baselineResult.Score
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

func (r BenchmarkRunner) RunCase(c BenchmarkCase) BenchmarkResult {
	scheduler := r.Scheduler
	reportPath := ""
	if c.RequireReport {
		reportPath = r.casePath(c.CaseID, "task-run.md")
	}
	runID := "benchmark-" + c.CaseID
	record := scheduler.Execute(c.Task, runID, reportPath)
	criteria := r.evaluate(c, record)
	replay := r.Replay(ReplayRecord{
		Task:     c.Task,
		RunID:    runID,
		Medium:   record.Decision.Medium,
		Approved: record.Decision.Approved,
		Status:   record.Run.Status,
	})
	totalWeight := 0
	earnedWeight := 0
	passed := replay.Matched
	for _, criterion := range criteria {
		totalWeight += criterion.Weight
		if criterion.Passed {
			earnedWeight += criterion.Weight
		} else {
			passed = false
		}
	}
	score := 0
	if totalWeight > 0 {
		score = int(math.Round((float64(earnedWeight) / float64(totalWeight)) * 100))
	}
	detailPagePath := ""
	if strings.TrimSpace(r.StorageDir) != "" {
		detailPagePath = r.casePath(c.CaseID, "run-detail.html")
		_ = os.MkdirAll(filepath.Dir(detailPagePath), 0o755)
		_ = os.WriteFile(detailPagePath, []byte(RenderRunReplayIndexPage(c.CaseID, record, replay, criteria)), 0o644)
	}
	return BenchmarkResult{
		CaseID:         c.CaseID,
		Score:          score,
		Passed:         passed,
		Criteria:       criteria,
		Record:         record,
		Replay:         replay,
		DetailPagePath: detailPagePath,
	}
}

func (r BenchmarkRunner) RunSuite(cases []BenchmarkCase, version string) BenchmarkSuiteResult {
	if strings.TrimSpace(version) == "" {
		version = "current"
	}
	results := make([]BenchmarkResult, 0, len(cases))
	for _, c := range cases {
		results = append(results, r.RunCase(c))
	}
	return BenchmarkSuiteResult{Results: results, Version: version}
}

func (r BenchmarkRunner) Replay(replayRecord ReplayRecord) ReplayOutcome {
	scheduler := r.Scheduler
	record := scheduler.Execute(replayRecord.Task, replayRecord.RunID+"-replay", "")
	observed := ReplayRecord{
		Task:     replayRecord.Task,
		RunID:    replayRecord.RunID,
		Medium:   record.Decision.Medium,
		Approved: record.Decision.Approved,
		Status:   record.Run.Status,
	}
	mismatches := make([]string, 0, 3)
	if observed.Medium != replayRecord.Medium {
		mismatches = append(mismatches, fmt.Sprintf("medium expected %s got %s", replayRecord.Medium, observed.Medium))
	}
	if observed.Approved != replayRecord.Approved {
		mismatches = append(mismatches, fmt.Sprintf("approved expected %t got %t", replayRecord.Approved, observed.Approved))
	}
	if observed.Status != replayRecord.Status {
		mismatches = append(mismatches, fmt.Sprintf("status expected %s got %s", replayRecord.Status, observed.Status))
	}
	reportPath := ""
	if strings.TrimSpace(r.StorageDir) != "" {
		reportPath = r.casePath(replayRecord.RunID, "replay.html")
		_ = os.MkdirAll(filepath.Dir(reportPath), 0o755)
		_ = os.WriteFile(reportPath, []byte(RenderReplayDetailPage(replayRecord, observed, mismatches)), 0o644)
	}
	return ReplayOutcome{
		Matched:      len(mismatches) == 0,
		ReplayRecord: observed,
		Mismatches:   mismatches,
		ReportPath:   reportPath,
	}
}

func (r BenchmarkRunner) evaluate(c BenchmarkCase, record ExecutionRecord) []EvaluationCriterion {
	return []EvaluationCriterion{
		r.criterion("decision-medium", 40, c.ExpectedMedium, record.Decision.Medium),
		r.criterionBool("approval-gate", 30, c.ExpectedApproved, record.Decision.Approved),
		r.criterion("final-status", 20, c.ExpectedStatus, record.Run.Status),
		{
			Name:   "report-artifact",
			Weight: 10,
			Passed: !c.RequireReport || strings.TrimSpace(record.ReportPath) != "",
			Detail: ternary(!c.RequireReport || strings.TrimSpace(record.ReportPath) != "", "report emitted", "report missing"),
		},
	}
}

func (r BenchmarkRunner) criterion(name string, weight int, expected *string, actual string) EvaluationCriterion {
	if expected == nil {
		return EvaluationCriterion{Name: name, Weight: weight, Passed: true, Detail: "not asserted"}
	}
	return EvaluationCriterion{
		Name:   name,
		Weight: weight,
		Passed: *expected == actual,
		Detail: fmt.Sprintf("expected %s got %s", *expected, actual),
	}
}

func (r BenchmarkRunner) criterionBool(name string, weight int, expected *bool, actual bool) EvaluationCriterion {
	if expected == nil {
		return EvaluationCriterion{Name: name, Weight: weight, Passed: true, Detail: "not asserted"}
	}
	return EvaluationCriterion{
		Name:   name,
		Weight: weight,
		Passed: *expected == actual,
		Detail: fmt.Sprintf("expected %t got %t", *expected, actual),
	}
}

func (r BenchmarkRunner) casePath(caseID, fileName string) string {
	if strings.TrimSpace(r.StorageDir) == "" {
		return fileName
	}
	return filepath.Join(r.StorageDir, caseID, fileName)
}

func RenderBenchmarkSuiteReport(suite BenchmarkSuiteResult, baseline *BenchmarkSuiteResult) string {
	lines := []string{
		"# Benchmark Suite Report",
		"",
		"- Version: " + suite.Version,
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
		lines = append(lines, "- Baseline Version: "+baseline.Version)
		lines = append(lines, fmt.Sprintf("- Score Delta: %d", suite.Score()-baseline.Score()))
		for _, comparison := range suite.Compare(*baseline) {
			lines = append(lines, fmt.Sprintf("- %s: baseline=%d current=%d delta=%d", comparison.CaseID, comparison.BaselineScore, comparison.CurrentScore, comparison.Delta))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderReplayDetailPage(expected ReplayRecord, observed ReplayRecord, mismatches []string) string {
	items := "<li>None</li>"
	if len(mismatches) > 0 {
		parts := make([]string, 0, len(mismatches))
		for _, item := range mismatches {
			parts = append(parts, "<li>"+html.EscapeString(item)+"</li>")
		}
		items = strings.Join(parts, "")
	}
	return strings.Join([]string{
		"<title>Replay Detail</title>",
		"<h1>Replay Detail</h1>",
		"<h2>Timeline / Log Sync</h2>",
		"<h2>Split View</h2>",
		"<h2>Reports</h2>",
		"<p>expected " + html.EscapeString(expected.Medium) + " observed " + html.EscapeString(observed.Medium) + "</p>",
		"<p>" + html.EscapeString(observed.Status) + "</p>",
		"<ul>" + items + "</ul>",
	}, "\n")
}

func RenderRunReplayIndexPage(caseID string, record ExecutionRecord, replay ReplayOutcome, criteria []EvaluationCriterion) string {
	reportPath := record.ReportPath
	if strings.TrimSpace(reportPath) == "" {
		reportPath = "n/a"
	}
	replayPath := replay.ReportPath
	if strings.TrimSpace(replayPath) == "" {
		replayPath = "n/a"
	}
	criteriaLines := make([]string, 0, len(criteria))
	for _, item := range criteria {
		criteriaLines = append(criteriaLines, "<li>"+html.EscapeString(item.Name)+"</li>")
	}
	return strings.Join([]string{
		"<title>Run Detail Index</title>",
		"<h1>Run Detail Index</h1>",
		"<h2>Timeline / Log Sync</h2>",
		"<h2>Acceptance</h2>",
		"<ul>" + strings.Join(criteriaLines, "") + "</ul>",
		"<h2>Reports</h2>",
		"<p>" + html.EscapeString(reportPath) + "</p>",
		"<p>" + html.EscapeString(replayPath) + "</p>",
		"<p>Replay</p>",
		"<p>case=" + html.EscapeString(caseID) + "</p>",
	}, "\n")
}

func stringPtr(value string) *string { return &value }
func boolPtr(value bool) *bool       { return &value }

func ternary[T any](condition bool, left, right T) T {
	if condition {
		return left
	}
	return right
}
