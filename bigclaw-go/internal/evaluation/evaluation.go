package evaluation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/scheduler"
)

type BenchmarkCase struct {
	CaseID           string      `json:"case_id"`
	Task             domain.Task `json:"task"`
	ExpectedMedium   string      `json:"expected_medium"`
	ExpectedApproved bool        `json:"expected_approved"`
	ExpectedStatus   string      `json:"expected_status"`
	RequireReport    bool        `json:"require_report"`
}

type EvaluationCriterion struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail,omitempty"`
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
	Criteria       []EvaluationCriterion `json:"criteria,omitempty"`
	Replay         ReplayOutcome         `json:"replay"`
	DetailPagePath string                `json:"detail_page_path,omitempty"`
}

type BenchmarkSuiteResult struct {
	Results []BenchmarkResult `json:"results,omitempty"`
	Version string            `json:"version,omitempty"`
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

type SuiteComparison struct {
	CaseID string `json:"case_id"`
	Delta  int    `json:"delta"`
}

func (s BenchmarkSuiteResult) Compare(baseline BenchmarkSuiteResult) []SuiteComparison {
	base := make(map[string]int, len(baseline.Results))
	for _, result := range baseline.Results {
		base[result.CaseID] = result.Score
	}
	out := make([]SuiteComparison, 0, len(s.Results))
	for _, result := range s.Results {
		out = append(out, SuiteComparison{
			CaseID: result.CaseID,
			Delta:  result.Score - base[result.CaseID],
		})
	}
	return out
}

type BenchmarkRunner struct {
	scheduler  *scheduler.Scheduler
	storageDir string
}

func NewBenchmarkRunner(storageDir string, sched *scheduler.Scheduler) *BenchmarkRunner {
	if sched == nil {
		sched = scheduler.New()
	}
	return &BenchmarkRunner{scheduler: sched, storageDir: storageDir}
}

func (r *BenchmarkRunner) RunCase(c BenchmarkCase) (BenchmarkResult, error) {
	decision := r.scheduler.Decide(c.Task, scheduler.QuotaSnapshot{})
	medium := string(decision.Assignment.Executor)
	approved := decision.Accepted
	if c.Task.RiskLevel == domain.RiskHigh {
		approved = false
	}
	status := expectedStatus(approved)
	record := ReplayRecord{
		Task:     c.Task,
		RunID:    "run-" + c.CaseID,
		Medium:   medium,
		Approved: approved,
		Status:   status,
	}

	criteria := []EvaluationCriterion{
		{Name: "decision-medium", Passed: medium == c.ExpectedMedium, Detail: fmt.Sprintf("expected %s got %s", c.ExpectedMedium, medium)},
		{Name: "decision-approved", Passed: approved == c.ExpectedApproved, Detail: fmt.Sprintf("expected %t got %t", c.ExpectedApproved, approved)},
		{Name: "decision-status", Passed: status == c.ExpectedStatus, Detail: fmt.Sprintf("expected %s got %s", c.ExpectedStatus, status)},
	}

	passedCount := 0
	for _, criterion := range criteria {
		if criterion.Passed {
			passedCount++
		}
	}
	passed := len(criteria) == passedCount
	score := 20 + (passedCount * 20)
	if passed {
		score = 100
	}

	replay := ReplayOutcome{Matched: passed, ReplayRecord: record}
	baseDir := filepath.Join(r.storageDir, c.CaseID)
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return BenchmarkResult{}, err
	}
	taskRunPath := filepath.Join(baseDir, "task-run.md")
	if err := os.WriteFile(taskRunPath, []byte(fmt.Sprintf("# Task Run\n\n- Case: %s\n- Medium: %s\n", c.CaseID, medium)), 0o644); err != nil {
		return BenchmarkResult{}, err
	}
	replayDir := filepath.Join(r.storageDir, "benchmark-"+c.CaseID)
	if err := os.MkdirAll(replayDir, 0o755); err != nil {
		return BenchmarkResult{}, err
	}
	replayPath := filepath.Join(replayDir, "replay.html")
	if err := os.WriteFile(replayPath, []byte(RenderReplayDetailPage(record, record, nil)), 0o644); err != nil {
		return BenchmarkResult{}, err
	}
	replay.ReportPath = replayPath

	detailPath := filepath.Join(baseDir, "run-detail.html")
	if err := os.WriteFile(detailPath, []byte(RenderRunReplayIndexPage(c.CaseID, record, replay, criteria)), 0o644); err != nil {
		return BenchmarkResult{}, err
	}

	return BenchmarkResult{
		CaseID:         c.CaseID,
		Score:          score,
		Passed:         passed,
		Criteria:       criteria,
		Replay:         replay,
		DetailPagePath: detailPath,
	}, nil
}

func (r *BenchmarkRunner) Replay(record ReplayRecord) (ReplayOutcome, error) {
	decision := r.scheduler.Decide(record.Task, scheduler.QuotaSnapshot{})
	approved := decision.Accepted
	if record.Task.RiskLevel == domain.RiskHigh {
		approved = false
	}
	observed := ReplayRecord{
		Task:     record.Task,
		RunID:    record.RunID,
		Medium:   string(decision.Assignment.Executor),
		Approved: approved,
		Status:   expectedStatus(approved),
	}
	mismatches := make([]string, 0)
	if observed.Medium != record.Medium {
		mismatches = append(mismatches, fmt.Sprintf("medium expected %s got %s", record.Medium, observed.Medium))
	}
	if observed.Approved != record.Approved {
		mismatches = append(mismatches, fmt.Sprintf("approved expected %t got %t", record.Approved, observed.Approved))
	}
	if observed.Status != record.Status {
		mismatches = append(mismatches, fmt.Sprintf("status expected %s got %s", record.Status, observed.Status))
	}
	reportPath := filepath.Join(r.storageDir, record.Task.ID+"-replay.html")
	if err := os.WriteFile(reportPath, []byte(RenderReplayDetailPage(record, observed, mismatches)), 0o644); err != nil {
		return ReplayOutcome{}, err
	}
	return ReplayOutcome{
		Matched:      len(mismatches) == 0,
		ReplayRecord: observed,
		Mismatches:   mismatches,
		ReportPath:   reportPath,
	}, nil
}

func (r *BenchmarkRunner) RunSuite(cases []BenchmarkCase, version string) (BenchmarkSuiteResult, error) {
	out := BenchmarkSuiteResult{Version: version}
	for _, c := range cases {
		result, err := r.RunCase(c)
		if err != nil {
			return BenchmarkSuiteResult{}, err
		}
		out.Results = append(out.Results, result)
	}
	return out, nil
}

func RenderBenchmarkSuiteReport(current, baseline BenchmarkSuiteResult) string {
	comparisons := current.Compare(baseline)
	lines := []string{
		"# Benchmark Suite Report",
		"",
		fmt.Sprintf("Version: %s", current.Version),
		fmt.Sprintf("Baseline Version: %s", baseline.Version),
	}
	for _, comparison := range comparisons {
		lines = append(lines, fmt.Sprintf("Score Delta: %d", comparison.Delta))
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderReplayDetailPage(expected, observed ReplayRecord, mismatches []string) string {
	lines := []string{
		"<html><body><h1>Replay Detail</h1>",
		"<h2>Timeline / Log Sync</h2>",
		"<h2>Split View</h2>",
		"<h2>Reports</h2>",
		fmt.Sprintf("<p>expected=%s observed=%s</p>", expected.Status, observed.Status),
	}
	for _, mismatch := range mismatches {
		lines = append(lines, "<p>"+mismatch+"</p>")
	}
	lines = append(lines, "</body></html>")
	return strings.Join(lines, "\n")
}

func RenderRunReplayIndexPage(caseID string, record ReplayRecord, replay ReplayOutcome, criteria []EvaluationCriterion) string {
	lines := []string{
		"<html><body><h1>Run Detail Index</h1>",
		"<h2>Timeline / Log Sync</h2>",
		"<h2>Acceptance</h2>",
		"<h2>Replay</h2>",
		"<h2>Reports</h2>",
		"<p>task-run.md</p>",
	}
	if strings.TrimSpace(replay.ReportPath) == "" {
		lines = append(lines, "<p>n/a</p>")
	} else {
		lines = append(lines, "<p>replay.html</p>")
	}
	for _, criterion := range criteria {
		lines = append(lines, "<p>"+criterion.Name+"</p>")
	}
	lines = append(lines, "</body></html>")
	return strings.Join(lines, "\n")
}

func expectedStatus(approved bool) string {
	if approved {
		return "approved"
	}
	return "needs-approval"
}
