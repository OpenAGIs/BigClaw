package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
)

func TestWorkpadJournalRecordAndWrite(t *testing.T) {
	dir := t.TempDir()
	journal := WorkpadJournal{
		TaskID: "BIG-401",
		RunID:  "run-7",
		Now:    func() time.Time { return time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC) },
	}
	journal.Record("execution", "running", map[string]any{"medium": "kubernetes"})
	path, err := journal.Write(filepath.Join(dir, "journals", "run-7.json"))
	if err != nil {
		t.Fatalf("write journal: %v", err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read journal: %v", err)
	}
	var restored WorkpadJournal
	if err := json.Unmarshal(body, &restored); err != nil {
		t.Fatalf("unmarshal journal: %v", err)
	}
	if restored.TaskID != "BIG-401" || restored.RunID != "run-7" || len(restored.Entries) != 1 {
		t.Fatalf("unexpected restored journal: %+v", restored)
	}
	if restored.Entries[0].Timestamp != "2026-03-20T10:00:00Z" || restored.Entries[0].Details["medium"] != "kubernetes" {
		t.Fatalf("unexpected journal entry: %+v", restored.Entries[0])
	}
}

func TestWorkpadJournalReadAndReplay(t *testing.T) {
	dir := t.TempDir()
	journal := WorkpadJournal{
		TaskID: "BIG-402-replay",
		RunID:  "run-journal-1",
		Now: func() time.Time {
			return time.Date(2026, 3, 28, 8, 0, 0, 0, time.UTC)
		},
	}
	journal.Record("intake", "recorded", map[string]any{"source": "local"})
	journal.Record("execution", "approved", map[string]any{"medium": "docker"})

	path, err := journal.Write(filepath.Join(dir, "journals", "run-journal-1.json"))
	if err != nil {
		t.Fatalf("write journal: %v", err)
	}
	loaded, err := ReadWorkpadJournal(path)
	if err != nil {
		t.Fatalf("read journal: %v", err)
	}
	if loaded.TaskID != "BIG-402-replay" || loaded.RunID != "run-journal-1" {
		t.Fatalf("unexpected loaded journal: %+v", loaded)
	}
	if got := loaded.Replay(); !reflect.DeepEqual(got, []string{"intake:recorded", "execution:approved"}) {
		t.Fatalf("unexpected replay: %#v", got)
	}
}

func TestAcceptanceGateRequiresApprovalForHighRiskWithoutApprovals(t *testing.T) {
	gate := AcceptanceGate{}
	task := domain.Task{
		ID:                 "task-high-risk",
		RiskLevel:          domain.RiskHigh,
		AcceptanceCriteria: []string{"report exported"},
		ValidationPlan:     []string{"go test ./..."},
	}
	decision := gate.Evaluate(task, ExecutionOutcome{Approved: false, Status: "needs-approval"}, []string{"report exported", "go test ./..."}, nil, "")
	if decision.Passed || decision.Status != "needs-approval" || decision.Summary == "" {
		t.Fatalf("unexpected approval decision: %+v", decision)
	}
}

func TestAcceptanceGateRejectsIncompleteEvidence(t *testing.T) {
	gate := AcceptanceGate{}
	task := domain.Task{
		ID:                 "task-incomplete",
		AcceptanceCriteria: []string{"weekly bundle", "git sync"},
		ValidationPlan:     []string{"go test ./..."},
	}
	decision := gate.Evaluate(task, ExecutionOutcome{Approved: true, Status: "completed"}, []string{"weekly bundle"}, []string{"ops-review"}, "")
	if decision.Passed || decision.Status != "rejected" {
		t.Fatalf("unexpected incomplete decision: %+v", decision)
	}
	if !reflect.DeepEqual(decision.MissingAcceptanceCriteria, []string{"git sync"}) || !reflect.DeepEqual(decision.MissingValidationSteps, []string{"go test ./..."}) {
		t.Fatalf("unexpected missing evidence details: %+v", decision)
	}
}

func TestAcceptanceGateAcceptsCompleteEvidenceAndApprovals(t *testing.T) {
	gate := AcceptanceGate{}
	task := domain.Task{
		ID:                 "task-complete",
		RiskLevel:          domain.RiskHigh,
		AcceptanceCriteria: []string{"report exported"},
		ValidationPlan:     []string{"go test ./...", "git log -1 --stat"},
	}
	decision := gate.Evaluate(task, ExecutionOutcome{Approved: true, Status: "completed"}, []string{"report exported", "go test ./...", "git log -1 --stat"}, []string{"ops-review", "ops-review", "security-review"}, "")
	if !decision.Passed || decision.Status != "accepted" {
		t.Fatalf("unexpected accepted decision: %+v", decision)
	}
	if !reflect.DeepEqual(decision.Approvals, []string{"ops-review", "security-review"}) {
		t.Fatalf("unexpected approvals: %+v", decision.Approvals)
	}
}

func TestAcceptanceGateRejectsHoldRecommendation(t *testing.T) {
	gate := AcceptanceGate{}
	task := domain.Task{
		ID:                 "task-pilot",
		AcceptanceCriteria: []string{"roi met"},
		ValidationPlan:     []string{"pilot scorecard"},
	}
	decision := gate.Evaluate(task, ExecutionOutcome{Approved: true, Status: "completed"}, []string{"roi met", "pilot scorecard"}, []string{"ops-review"}, "hold")
	if decision.Passed || decision.Status != "rejected" || decision.Summary != "pilot scorecard indicates insufficient ROI or KPI progress" {
		t.Fatalf("unexpected pilot rejection: %+v", decision)
	}
}
