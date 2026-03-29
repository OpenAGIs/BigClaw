package evaluationparity

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bigclaw-go/internal/executionparity"
)

type EvaluationCriterion struct {
	Name   string
	Weight int
	Passed bool
	Detail string
}

type BenchmarkCase struct {
	CaseID           string
	Task             executionparity.Task
	ExpectedMedium   string
	ExpectedApproved *bool
	ExpectedStatus   string
	RequireReport    bool
}

type ReplayRecord struct {
	Task     executionparity.Task
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
	Criteria       []EvaluationCriterion
	Record         executionparity.Record
	Replay         ReplayOutcome
	DetailPagePath string
	ReportPath     string
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

func (s BenchmarkSuiteResult) Score() int {
	if len(s.Results) == 0 {
		return 0
	}
	total := 0
	for _, result := range s.Results {
		total += result.Score
	}
	return total / len(s.Results)
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
	byCase := map[string]BenchmarkResult{}
	for _, result := range baseline.Results {
		byCase[result.CaseID] = result
	}
	comparisons := make([]BenchmarkComparison, 0, len(s.Results))
	for _, result := range s.Results {
		baselineScore := 0
		if prior, ok := byCase[result.CaseID]; ok {
			baselineScore = prior.Score
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

type BenchmarkRunner struct {
	StorageDir string
	Scheduler  executionparity.Scheduler
}

func (r BenchmarkRunner) RunCase(c BenchmarkCase) (BenchmarkResult, error) {
	ledger := &executionparity.Ledger{Path: r.casePath(c.CaseID, "ledger.json")}
	reportPath := ""
	if c.RequireReport {
		reportPath = r.casePath(c.CaseID, "task-run.md")
	}
	runID := "benchmark-" + c.CaseID
	record, err := r.Scheduler.Execute(c.Task, runID, ledger, reportPath)
	if err != nil {
		return BenchmarkResult{}, err
	}
	criteria := evaluate(c, record, reportPath)
	replay, err := r.Replay(ReplayRecord{
		Task:     c.Task,
		RunID:    runID,
		Medium:   record.Decision.Medium,
		Approved: record.Decision.Approved,
		Status:   record.Run.Status,
	})
	if err != nil {
		return BenchmarkResult{}, err
	}
	score := scoreCriteria(criteria)
	passed := replay.Matched
	for _, item := range criteria {
		if !item.Passed {
			passed = false
			break
		}
	}
	detailPagePath := ""
	if strings.TrimSpace(r.StorageDir) != "" {
		detailPagePath = r.casePath(c.CaseID, "run-detail.html")
		if err := os.WriteFile(detailPagePath, []byte(RenderRunReplayIndexPage(c.CaseID, record, replay, criteria)), 0o644); err != nil {
			return BenchmarkResult{}, err
		}
	}
	return BenchmarkResult{
		CaseID:         c.CaseID,
		Score:          score,
		Passed:         passed,
		Criteria:       criteria,
		Record:         record,
		Replay:         replay,
		DetailPagePath: detailPagePath,
		ReportPath:     reportPath,
	}, nil
}

func (r BenchmarkRunner) RunSuite(cases []BenchmarkCase, version string) (BenchmarkSuiteResult, error) {
	results := make([]BenchmarkResult, 0, len(cases))
	for _, c := range cases {
		result, err := r.RunCase(c)
		if err != nil {
			return BenchmarkSuiteResult{}, err
		}
		results = append(results, result)
	}
	return BenchmarkSuiteResult{Results: results, Version: version}, nil
}

func (r BenchmarkRunner) Replay(record ReplayRecord) (ReplayOutcome, error) {
	ledger := &executionparity.Ledger{Path: r.casePath(record.RunID, "replay-ledger.json")}
	replayed, err := r.Scheduler.Execute(record.Task, record.RunID+"-replay", ledger, "")
	if err != nil {
		return ReplayOutcome{}, err
	}
	observed := ReplayRecord{
		Task:     record.Task,
		RunID:    record.RunID,
		Medium:   replayed.Decision.Medium,
		Approved: replayed.Decision.Approved,
		Status:   replayed.Run.Status,
	}
	mismatches := make([]string, 0, 3)
	if observed.Medium != record.Medium {
		mismatches = append(mismatches, fmt.Sprintf("medium expected %s got %s", record.Medium, observed.Medium))
	}
	if observed.Approved != record.Approved {
		mismatches = append(mismatches, fmt.Sprintf("approved expected %t got %t", record.Approved, observed.Approved))
	}
	if observed.Status != record.Status {
		mismatches = append(mismatches, fmt.Sprintf("status expected %s got %s", record.Status, observed.Status))
	}
	reportPath := ""
	if strings.TrimSpace(r.StorageDir) != "" {
		reportPath = r.casePath(record.RunID, "replay.html")
		if err := os.WriteFile(reportPath, []byte(RenderReplayDetailPage(record, observed, mismatches)), 0o644); err != nil {
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
	var lines []string
	lines = append(lines,
		"# Benchmark Suite Report",
		"",
		"- Version: "+suite.Version,
		fmt.Sprintf("- Cases: %d", len(suite.Results)),
		fmt.Sprintf("- Passed: %t", suite.Passed()),
		fmt.Sprintf("- Score: %d", suite.Score()),
		"",
		"## Cases",
		"",
	)
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
		comparisons := suite.Compare(*baseline)
		if len(comparisons) == 0 {
			lines = append(lines, "- No comparable cases")
		} else {
			for _, item := range comparisons {
				lines = append(lines, fmt.Sprintf("- %s: baseline=%d current=%d delta=%d", item.CaseID, item.BaselineScore, item.CurrentScore, item.Delta))
			}
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderReplayDetailPage(expected, observed ReplayRecord, mismatches []string) string {
	var lines []string
	lines = append(lines,
		"Replay Detail",
		"Timeline / Log Sync",
		"Split View",
		"Reports",
		"Expected Medium: "+expected.Medium,
		"Observed Medium: "+observed.Medium,
		"Observed Status: "+observed.Status,
	)
	if len(mismatches) == 0 {
		lines = append(lines, "No mismatches")
	} else {
		lines = append(lines, mismatches...)
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderRunReplayIndexPage(caseID string, record executionparity.Record, replay ReplayOutcome, criteria []EvaluationCriterion) string {
	reportPath := "n/a"
	if len(record.Run.Artifacts) > 1 {
		reportPath = record.Run.Artifacts[1].Path
	}
	replayPath := replay.ReportPath
	if replayPath == "" {
		replayPath = "n/a"
	}
	lines := []string{
		"Run Detail Index",
		"Timeline / Log Sync",
		"Acceptance",
		"Reports",
		"Case: " + caseID,
		"Replay",
		reportPath,
		replayPath,
	}
	for _, item := range criteria {
		lines = append(lines, item.Name)
	}
	return strings.Join(lines, "\n") + "\n"
}

func evaluate(c BenchmarkCase, record executionparity.Record, reportPath string) []EvaluationCriterion {
	return []EvaluationCriterion{
		criterion("decision-medium", 40, c.ExpectedMedium, record.Decision.Medium),
		criterionBool("approval-gate", 30, c.ExpectedApproved, record.Decision.Approved),
		criterion("final-status", 20, c.ExpectedStatus, record.Run.Status),
		{
			Name:   "report-artifact",
			Weight: 10,
			Passed: !c.RequireReport || reportPath != "",
			Detail: func() string {
				if !c.RequireReport || reportPath != "" {
					return "report emitted"
				}
				return "report missing"
			}(),
		},
	}
}

func criterion(name string, weight int, expected string, actual string) EvaluationCriterion {
	if strings.TrimSpace(expected) == "" {
		return EvaluationCriterion{Name: name, Weight: weight, Passed: true, Detail: "not asserted"}
	}
	return EvaluationCriterion{
		Name:   name,
		Weight: weight,
		Passed: expected == actual,
		Detail: fmt.Sprintf("expected %s got %s", expected, actual),
	}
}

func criterionBool(name string, weight int, expected *bool, actual bool) EvaluationCriterion {
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

func scoreCriteria(criteria []EvaluationCriterion) int {
	totalWeight := 0
	earnedWeight := 0
	for _, item := range criteria {
		totalWeight += item.Weight
		if item.Passed {
			earnedWeight += item.Weight
		}
	}
	if totalWeight == 0 {
		return 0
	}
	return (earnedWeight * 100) / totalWeight
}

func (r BenchmarkRunner) casePath(caseID, fileName string) string {
	if strings.TrimSpace(r.StorageDir) == "" {
		return fileName
	}
	return filepath.Join(r.StorageDir, caseID, fileName)
}
