package executionparity

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

type Task struct {
	ID            string
	Source        string
	Title         string
	Description   string
	Priority      int
	RiskLevel     RiskLevel
	RequiredTools []string
}

type Queue struct {
	tasks []Task
}

func (q *Queue) Enqueue(task Task) {
	q.tasks = append(q.tasks, task)
}

func (q *Queue) Dequeue() *Task {
	if len(q.tasks) == 0 {
		return nil
	}
	task := q.tasks[0]
	q.tasks = q.tasks[1:]
	return &task
}

type Decision struct {
	Medium   string `json:"medium"`
	Approved bool   `json:"approved"`
	Reason   string `json:"reason"`
}

type Trace struct {
	Span   string         `json:"span"`
	Status string         `json:"status"`
	Attrs  map[string]any `json:"attributes,omitempty"`
}

type Artifact struct {
	Kind string `json:"kind"`
	Path string `json:"path"`
}

type Audit struct {
	Action  string         `json:"action"`
	Outcome string         `json:"outcome"`
	Details map[string]any `json:"details,omitempty"`
}

type Run struct {
	Status    string     `json:"status"`
	Medium    string     `json:"medium"`
	Traces    []Trace    `json:"traces,omitempty"`
	Artifacts []Artifact `json:"artifacts,omitempty"`
	Audits    []Audit    `json:"audits,omitempty"`
}

type Record struct {
	Decision Decision
	Run      Run
}

type Ledger struct {
	Path string
}

func (l Ledger) Load() ([]Run, error) {
	body, err := os.ReadFile(l.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var runs []Run
	if err := json.Unmarshal(body, &runs); err != nil {
		return nil, err
	}
	return runs, nil
}

func (l Ledger) Append(run Run) error {
	runs, err := l.Load()
	if err != nil {
		return err
	}
	runs = append(runs, run)
	if err := os.MkdirAll(filepath.Dir(l.Path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(runs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.Path, body, 0o644)
}

type Scheduler struct{}

func (Scheduler) Decide(task Task) Decision {
	if task.RiskLevel == RiskHigh {
		return Decision{Medium: "vm", Approved: false, Reason: "requires approval for high-risk task"}
	}
	for _, tool := range task.RequiredTools {
		if tool == "browser" {
			return Decision{Medium: "browser", Approved: true, Reason: "browser automation task"}
		}
	}
	return Decision{Medium: "docker", Approved: true, Reason: "default low/medium risk path"}
}

func (s Scheduler) Execute(task Task, runID string, ledger *Ledger, reportPath string) (Record, error) {
	decision := s.Decide(task)
	status := "approved"
	outcome := "approved"
	if !decision.Approved {
		status = "needs-approval"
		outcome = "pending"
	}
	run := Run{
		Status: status,
		Medium: decision.Medium,
		Traces: []Trace{
			{
				Span:   "scheduler.decide",
				Status: ternary(decision.Approved, "ok", "pending"),
				Attrs: map[string]any{
					"approved": decision.Approved,
					"medium":   decision.Medium,
				},
			},
		},
		Audits: []Audit{
			{
				Action:  "scheduler.decision",
				Outcome: outcome,
				Details: map[string]any{"reason": decision.Reason},
			},
		},
	}
	if strings.TrimSpace(reportPath) != "" {
		if err := writeReport(reportPath, status); err != nil {
			return Record{}, err
		}
		htmlPath := strings.TrimSuffix(reportPath, filepath.Ext(reportPath)) + ".html"
		if err := os.WriteFile(htmlPath, []byte("<html><body>Status: "+status+"</body></html>\n"), 0o644); err != nil {
			return Record{}, err
		}
		run.Artifacts = append(run.Artifacts,
			Artifact{Kind: "page", Path: htmlPath},
			Artifact{Kind: "report", Path: reportPath},
		)
	}
	if ledger != nil {
		if err := ledger.Append(run); err != nil {
			return Record{}, err
		}
	}
	return Record{Decision: decision, Run: run}, nil
}

func writeReport(reportPath, status string) error {
	if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(reportPath, []byte("# Task Run\n\nStatus: "+status+"\n"), 0o644)
}

func ternary(condition bool, whenTrue, whenFalse string) string {
	if condition {
		return whenTrue
	}
	return whenFalse
}
