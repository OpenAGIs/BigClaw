package evaluation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Task struct {
	ID            string
	Title         string
	Source        string
	RiskLevel     string
	RequiredTools []string
}

type BenchmarkCase struct {
	CaseID           string
	Task             Task
	ExpectedMedium   string
	ExpectedApproved bool
	ExpectedStatus   string
	RequireReport    bool
}

type Criterion struct {
	Name   string
	Passed bool
}

type ReplayRecord struct {
	Task     Task
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

type Result struct {
	CaseID         string
	Score          int
	Passed         bool
	Criteria       []Criterion
	Replay         ReplayOutcome
	DetailPagePath string
}

type Comparison struct {
	CaseID string
	Delta  int
}

type Suite struct {
	Results []Result
	Version string
}

func (s Suite) Score() int {
	if len(s.Results) == 0 {
		return 0
	}
	total := 0
	for _, result := range s.Results {
		total += result.Score
	}
	return total / len(s.Results)
}

func (s Suite) Compare(baseline Suite) []Comparison {
	previous := 0
	if len(baseline.Results) > 0 {
		previous = baseline.Score()
	}
	out := make([]Comparison, 0, len(s.Results))
	for _, result := range s.Results {
		out = append(out, Comparison{CaseID: result.CaseID, Delta: result.Score - previous})
	}
	return out
}

type Runner struct {
	storageDir string
}

func New(storageDir string) *Runner {
	return &Runner{storageDir: storageDir}
}

func (r *Runner) RunCase(item BenchmarkCase) (Result, error) {
	medium, approved, status := decide(item.Task)
	criteria := []Criterion{
		{Name: "decision-medium", Passed: medium == item.ExpectedMedium},
		{Name: "decision-approved", Passed: approved == item.ExpectedApproved},
		{Name: "decision-status", Passed: status == item.ExpectedStatus},
	}
	score := 60
	passed := true
	for _, criterion := range criteria {
		if !criterion.Passed {
			passed = false
		}
	}
	if passed {
		score = 100
	}

	caseDir := filepath.Join(r.storageDir, item.CaseID)
	replayDir := filepath.Join(r.storageDir, "benchmark-"+item.CaseID)
	if err := os.MkdirAll(caseDir, 0o755); err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(replayDir, 0o755); err != nil {
		return Result{}, err
	}
	taskRunPath := filepath.Join(caseDir, "task-run.md")
	if item.RequireReport {
		if err := os.WriteFile(taskRunPath, []byte("# task run\n"), 0o644); err != nil {
			return Result{}, err
		}
	}
	replayPath := filepath.Join(replayDir, "replay.html")
	if err := os.WriteFile(replayPath, []byte("Replay"), 0o644); err != nil {
		return Result{}, err
	}
	detailPath := filepath.Join(caseDir, "run-detail.html")
	index := RenderRunReplayIndexPage(item.CaseID, taskRunPath, replayPath, criteria)
	if err := os.WriteFile(detailPath, []byte(index), 0o644); err != nil {
		return Result{}, err
	}

	replayRecord := ReplayRecord{Task: item.Task, RunID: "run-1", Medium: medium, Approved: approved, Status: status}
	return Result{
		CaseID:         item.CaseID,
		Score:          score,
		Passed:         passed,
		Criteria:       criteria,
		Replay:         ReplayOutcome{Matched: true, ReplayRecord: replayRecord},
		DetailPagePath: detailPath,
	}, nil
}

func (r *Runner) Replay(record ReplayRecord) (ReplayOutcome, error) {
	observedMedium, observedApproved, observedStatus := decide(record.Task)
	mismatches := make([]string, 0)
	if observedMedium != record.Medium {
		mismatches = append(mismatches, fmt.Sprintf("medium expected %s got %s", record.Medium, observedMedium))
	}
	if observedApproved != record.Approved {
		mismatches = append(mismatches, fmt.Sprintf("approved expected %t got %t", record.Approved, observedApproved))
	}
	if observedStatus != record.Status {
		mismatches = append(mismatches, fmt.Sprintf("status expected %s got %s", record.Status, observedStatus))
	}
	reportPath := filepath.Join(r.storageDir, "replay-report.html")
	if err := os.WriteFile(reportPath, []byte(RenderReplayDetailPage(record, ReplayRecord{
		Task:     record.Task,
		RunID:    record.RunID,
		Medium:   observedMedium,
		Approved: observedApproved,
		Status:   observedStatus,
	}, mismatches)), 0o644); err != nil {
		return ReplayOutcome{}, err
	}
	return ReplayOutcome{
		Matched:      len(mismatches) == 0,
		ReplayRecord: ReplayRecord{Task: record.Task, RunID: record.RunID, Medium: observedMedium, Approved: observedApproved, Status: observedStatus},
		Mismatches:   mismatches,
		ReportPath:   reportPath,
	}, nil
}

func (r *Runner) RunSuite(items []BenchmarkCase, version string) (Suite, error) {
	results := make([]Result, 0, len(items))
	for _, item := range items {
		result, err := r.RunCase(item)
		if err != nil {
			return Suite{}, err
		}
		results = append(results, result)
	}
	return Suite{Results: results, Version: version}, nil
}

func RenderBenchmarkSuiteReport(current Suite, baseline Suite) string {
	delta := 0
	comparison := current.Compare(baseline)
	if len(comparison) > 0 {
		delta = comparison[0].Delta
	}
	return fmt.Sprintf("Version: %s\nBaseline Version: %s\nScore Delta: %d\n", current.Version, baseline.Version, delta)
}

func RenderReplayDetailPage(expected, observed ReplayRecord, mismatches []string) string {
	return strings.Join([]string{
		"<title>Replay Detail</title>",
		"Timeline / Log Sync",
		"Split View",
		"Reports",
		strings.Join(mismatches, "\n"),
		observed.Status,
	}, "\n")
}

func RenderRunReplayIndexPage(caseID, taskRunPath, replayPath string, criteria []Criterion) string {
	criteriaNames := make([]string, 0, len(criteria))
	for _, criterion := range criteria {
		criteriaNames = append(criteriaNames, criterion.Name)
	}
	taskRunLabel := taskRunPath
	if taskRunPath == "" {
		taskRunLabel = "n/a"
	} else {
		taskRunLabel = filepath.Base(taskRunPath)
	}
	return strings.Join([]string{
		"Run Detail Index",
		"Timeline / Log Sync",
		"Acceptance",
		"Replay",
		"Reports",
		taskRunLabel,
		filepath.Base(replayPath),
		strings.Join(criteriaNames, "\n"),
	}, "\n")
}

func decide(task Task) (string, bool, string) {
	for _, tool := range task.RequiredTools {
		if tool == "browser" {
			return "browser", true, "approved"
		}
	}
	if strings.EqualFold(task.RiskLevel, "high") {
		return "vm", false, "needs-approval"
	}
	return "docker", true, "approved"
}
