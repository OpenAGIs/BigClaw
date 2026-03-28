package evaluation

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/scheduler"
)

type Criterion struct {
	Name   string `json:"name"`
	Weight int    `json:"weight"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail,omitempty"`
}

type BenchmarkCase struct {
	CaseID           string      `json:"case_id"`
	Task             domain.Task `json:"task"`
	ExpectedMedium   string      `json:"expected_medium,omitempty"`
	ExpectedAccepted bool        `json:"expected_accepted"`
	ExpectedStatus   string      `json:"expected_status,omitempty"`
	RequireReport    bool        `json:"require_report,omitempty"`
}

type ReplayRecord struct {
	Task     domain.Task `json:"task"`
	RunID    string      `json:"run_id"`
	Medium   string      `json:"medium,omitempty"`
	Accepted bool        `json:"accepted"`
	Status   string      `json:"status,omitempty"`
}

type ReplayOutcome struct {
	Matched      bool         `json:"matched"`
	ReplayRecord ReplayRecord `json:"replay_record"`
	Mismatches   []string     `json:"mismatches,omitempty"`
	ReportPath   string       `json:"report_path,omitempty"`
}

type BenchmarkResult struct {
	CaseID         string        `json:"case_id"`
	Score          int           `json:"score"`
	Passed         bool          `json:"passed"`
	Criteria       []Criterion   `json:"criteria"`
	Replay         ReplayOutcome `json:"replay"`
	DetailPagePath string        `json:"detail_page_path,omitempty"`
}

type BenchmarkDelta struct {
	CaseID string `json:"case_id"`
	Delta  int    `json:"delta"`
}

type BenchmarkSuiteResult struct {
	Results []BenchmarkResult `json:"results"`
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

func (s BenchmarkSuiteResult) Compare(baseline BenchmarkSuiteResult) []BenchmarkDelta {
	index := map[string]int{}
	for _, item := range baseline.Results {
		index[item.CaseID] = item.Score
	}
	deltas := make([]BenchmarkDelta, 0, len(s.Results))
	for _, item := range s.Results {
		deltas = append(deltas, BenchmarkDelta{
			CaseID: item.CaseID,
			Delta:  item.Score - index[item.CaseID],
		})
	}
	sort.Slice(deltas, func(i, j int) bool { return deltas[i].CaseID < deltas[j].CaseID })
	return deltas
}

type Runner struct {
	scheduler  *scheduler.Scheduler
	storageDir string
}

func NewRunner(storageDir string, sched *scheduler.Scheduler) *Runner {
	if sched == nil {
		sched = scheduler.New()
	}
	return &Runner{
		scheduler:  sched,
		storageDir: storageDir,
	}
}

func (r *Runner) RunCase(item BenchmarkCase) (BenchmarkResult, error) {
	decision := r.scheduler.Decide(item.Task, scheduler.QuotaSnapshot{})
	observed := ReplayRecord{
		Task:     item.Task,
		RunID:    "run-" + item.CaseID,
		Medium:   string(decision.Assignment.Executor),
		Accepted: decision.Accepted,
		Status:   statusForDecision(decision),
	}
	replay, err := r.replayExpected(item.CaseID, ReplayRecord{
		Task:     item.Task,
		RunID:    observed.RunID,
		Medium:   item.ExpectedMedium,
		Accepted: item.ExpectedAccepted,
		Status:   item.ExpectedStatus,
	}, observed, item.RequireReport)
	if err != nil {
		return BenchmarkResult{}, err
	}
	criteria := []Criterion{
		{
			Name:   "decision-medium",
			Weight: 40,
			Passed: strings.TrimSpace(item.ExpectedMedium) == observed.Medium,
			Detail: fmt.Sprintf("expected=%s observed=%s", item.ExpectedMedium, observed.Medium),
		},
		{
			Name:   "decision-accepted",
			Weight: 30,
			Passed: item.ExpectedAccepted == observed.Accepted,
			Detail: fmt.Sprintf("expected=%t observed=%t", item.ExpectedAccepted, observed.Accepted),
		},
		{
			Name:   "decision-status",
			Weight: 30,
			Passed: strings.TrimSpace(item.ExpectedStatus) == observed.Status,
			Detail: fmt.Sprintf("expected=%s observed=%s", item.ExpectedStatus, observed.Status),
		},
	}
	score := 0
	for _, criterion := range criteria {
		if criterion.Passed {
			score += criterion.Weight
		}
	}

	detailPath := ""
	if item.RequireReport {
		root := filepath.Join(r.storageDir, item.CaseID)
		if err := os.MkdirAll(root, 0o755); err != nil {
			return BenchmarkResult{}, err
		}
		taskRunPath := filepath.Join(root, "task-run.md")
		if err := os.WriteFile(taskRunPath, []byte(renderTaskRunMarkdown(observed, decision.Reason)), 0o644); err != nil {
			return BenchmarkResult{}, err
		}
		detailPath = filepath.Join(root, "run-detail.html")
		body := RenderRunReplayIndexPage(item.CaseID, observed, replay, criteria)
		if err := os.WriteFile(detailPath, []byte(body), 0o644); err != nil {
			return BenchmarkResult{}, err
		}
	}

	return BenchmarkResult{
		CaseID:         item.CaseID,
		Score:          score,
		Passed:         score == 100,
		Criteria:       criteria,
		Replay:         replay,
		DetailPagePath: detailPath,
	}, nil
}

func (r *Runner) RunSuite(cases []BenchmarkCase, version string) (BenchmarkSuiteResult, error) {
	results := make([]BenchmarkResult, 0, len(cases))
	for _, item := range cases {
		result, err := r.RunCase(item)
		if err != nil {
			return BenchmarkSuiteResult{}, err
		}
		results = append(results, result)
	}
	return BenchmarkSuiteResult{Results: results, Version: version}, nil
}

func (r *Runner) Replay(record ReplayRecord) (ReplayOutcome, error) {
	decision := r.scheduler.Decide(record.Task, scheduler.QuotaSnapshot{})
	observed := ReplayRecord{
		Task:     record.Task,
		RunID:    record.RunID,
		Medium:   string(decision.Assignment.Executor),
		Accepted: decision.Accepted,
		Status:   statusForDecision(decision),
	}
	return r.replayExpected(strings.TrimSpace(record.Task.ID), record, observed, true)
}

func (r *Runner) replayExpected(caseID string, expected ReplayRecord, observed ReplayRecord, writeReport bool) (ReplayOutcome, error) {
	mismatches := compareReplay(expected, observed)
	outcome := ReplayOutcome{
		Matched:      len(mismatches) == 0,
		ReplayRecord: observed,
		Mismatches:   mismatches,
	}
	if !writeReport {
		return outcome, nil
	}
	root := filepath.Join(r.storageDir, "benchmark-"+slug(caseID))
	if err := os.MkdirAll(root, 0o755); err != nil {
		return ReplayOutcome{}, err
	}
	reportPath := filepath.Join(root, "replay.html")
	if err := os.WriteFile(reportPath, []byte(RenderReplayDetailPage(expected, observed, mismatches)), 0o644); err != nil {
		return ReplayOutcome{}, err
	}
	outcome.ReportPath = reportPath
	return outcome, nil
}

func RenderBenchmarkSuiteReport(current, baseline BenchmarkSuiteResult) string {
	var b strings.Builder
	b.WriteString("# Benchmark Suite Report\n\n")
	b.WriteString("Version: " + current.Version + "\n")
	b.WriteString("Baseline Version: " + baseline.Version + "\n")
	b.WriteString(fmt.Sprintf("Suite Score: %d\n\n", current.Score()))
	for _, delta := range current.Compare(baseline) {
		b.WriteString(fmt.Sprintf("- %s: Score Delta: %d\n", delta.CaseID, delta.Delta))
	}
	return b.String()
}

func RenderReplayDetailPage(expected, observed ReplayRecord, mismatches []string) string {
	var b strings.Builder
	b.WriteString("<html><head><title>Replay Detail</title></head><body>\n")
	b.WriteString("<h1>Replay Detail</h1>\n")
	b.WriteString("<h2>Timeline / Log Sync</h2>\n")
	b.WriteString("<h2>Split View</h2>\n")
	b.WriteString("<h2>Reports</h2>\n")
	b.WriteString("<p>expected medium=" + html.EscapeString(expected.Medium) + " status=" + html.EscapeString(expected.Status) + "</p>\n")
	b.WriteString("<p>observed medium=" + html.EscapeString(observed.Medium) + " status=" + html.EscapeString(observed.Status) + "</p>\n")
	if len(mismatches) > 0 {
		b.WriteString("<ul>\n")
		for _, item := range mismatches {
			b.WriteString("<li>" + html.EscapeString(item) + "</li>\n")
		}
		b.WriteString("</ul>\n")
	}
	b.WriteString("</body></html>\n")
	return b.String()
}

func RenderRunReplayIndexPage(caseID string, observed ReplayRecord, replay ReplayOutcome, criteria []Criterion) string {
	var b strings.Builder
	b.WriteString("<html><head><title>Run Detail Index</title></head><body>\n")
	b.WriteString("<h1>Run Detail Index</h1>\n")
	b.WriteString("<h2>Timeline / Log Sync</h2>\n")
	b.WriteString("<h2>Acceptance</h2>\n")
	b.WriteString("<h2>Reports</h2>\n")
	b.WriteString("<p>case=" + html.EscapeString(caseID) + "</p>\n")
	b.WriteString("<p>task-run.md</p>\n")
	if strings.TrimSpace(replay.ReportPath) == "" {
		b.WriteString("<p>Replay: n/a</p>\n")
	} else {
		b.WriteString("<p>Replay: " + html.EscapeString(filepath.Base(replay.ReportPath)) + "</p>\n")
	}
	b.WriteString("<ul>\n")
	for _, item := range criteria {
		b.WriteString("<li>" + html.EscapeString(item.Name) + "</li>\n")
	}
	b.WriteString("</ul>\n")
	b.WriteString("</body></html>\n")
	return b.String()
}

func compareReplay(expected, observed ReplayRecord) []string {
	mismatches := []string{}
	if strings.TrimSpace(expected.Medium) != strings.TrimSpace(observed.Medium) {
		mismatches = append(mismatches, fmt.Sprintf("medium expected %s got %s", expected.Medium, observed.Medium))
	}
	if expected.Accepted != observed.Accepted {
		mismatches = append(mismatches, fmt.Sprintf("accepted expected %t got %t", expected.Accepted, observed.Accepted))
	}
	if strings.TrimSpace(expected.Status) != strings.TrimSpace(observed.Status) {
		mismatches = append(mismatches, fmt.Sprintf("status expected %s got %s", expected.Status, observed.Status))
	}
	return mismatches
}

func statusForDecision(decision scheduler.Decision) string {
	if decision.Accepted {
		return "accepted"
	}
	return "needs-approval"
}

func renderTaskRunMarkdown(observed ReplayRecord, reason string) string {
	var b strings.Builder
	b.WriteString("# Task Run\n\n")
	b.WriteString("Run ID: " + observed.RunID + "\n")
	b.WriteString("Task ID: " + observed.Task.ID + "\n")
	b.WriteString("Medium: " + observed.Medium + "\n")
	b.WriteString("Status: " + observed.Status + "\n")
	b.WriteString("Reason: " + reason + "\n")
	return b.String()
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	if value == "" {
		return "case"
	}
	return value
}
