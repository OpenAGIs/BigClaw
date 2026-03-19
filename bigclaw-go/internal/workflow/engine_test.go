package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/observability"
)

func TestAcceptanceGateRejectsMissingEvidence(t *testing.T) {
	task := domain.Task{
		ID:                 "BIG-403",
		Source:             "linear",
		Title:              "Close acceptance gate",
		AcceptanceCriteria: []string{"report-shared"},
		ValidationPlan:     []string{"pytest"},
	}

	execution, err := DefaultExecutor{}.Execute(task, "run-gate-1")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	decision := AcceptanceGate{}.Evaluate(task, execution, []string{"pytest"}, nil)

	if decision.Passed {
		t.Fatalf("expected failed decision, got %+v", decision)
	}
	if decision.Status != "rejected" {
		t.Fatalf("expected rejected status, got %+v", decision)
	}
	if len(decision.MissingAcceptanceCriteria) != 1 || decision.MissingAcceptanceCriteria[0] != "report-shared" {
		t.Fatalf("unexpected missing acceptance: %+v", decision)
	}
	if len(decision.MissingValidationSteps) != 0 {
		t.Fatalf("unexpected missing validation: %+v", decision)
	}
}

func TestEngineRecordsJournalAndAcceptsCompleteEvidence(t *testing.T) {
	dir := t.TempDir()
	engine := NewEngine()
	task := domain.Task{
		ID:                 "BIG-402",
		Source:             "linear",
		Title:              "Record workflow journal",
		AcceptanceCriteria: []string{"report-shared"},
		ValidationPlan:     []string{"pytest"},
		RequiredTools:      []string{"browser"},
	}

	result, err := engine.Run(
		task,
		"run-wf-1",
		filepath.Join(dir, "reports", "run-wf-1.md"),
		filepath.Join(dir, "journals", "run-wf-1.json"),
		[]string{"pytest", "report-shared"},
		nil,
	)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Execution.Decision.Medium != "browser" {
		t.Fatalf("expected browser medium, got %+v", result.Execution)
	}
	if !result.Acceptance.Passed || result.Acceptance.Status != "accepted" {
		t.Fatalf("expected accepted result, got %+v", result.Acceptance)
	}
	if result.JournalPath == "" || result.ReportPath == "" {
		t.Fatalf("expected report and journal paths, got %+v", result)
	}

	body, err := os.ReadFile(result.JournalPath)
	if err != nil {
		t.Fatalf("read journal: %v", err)
	}
	var journal WorkpadJournal
	if err := json.Unmarshal(body, &journal); err != nil {
		t.Fatalf("decode journal: %v", err)
	}
	if len(journal.Entries) != 4 {
		t.Fatalf("expected 4 journal entries, got %+v", journal.Entries)
	}
	if journal.Entries[2].Step != "acceptance" || journal.Entries[2].Status != "accepted" {
		t.Fatalf("unexpected acceptance journal entry: %+v", journal.Entries[2])
	}
	if journal.Entries[3].Step != "closeout" || journal.Entries[3].Status != "complete" {
		t.Fatalf("unexpected closeout journal entry: %+v", journal.Entries[3])
	}
}

func TestEngineRunDefinitionClosesHighRiskTaskWithManualApproval(t *testing.T) {
	dir := t.TempDir()
	recorder := observability.NewRecorder()
	engine := &Engine{Executor: DefaultExecutor{}, Recorder: recorder}
	definition := Definition{
		Name:                "prod-approval",
		Steps:               []Step{{Name: "review", Kind: "approval"}},
		ValidationEvidence:  []string{"rollback-plan", "integration-test"},
		Approvals:           []string{"release-manager"},
		ReportPathTemplate:  filepath.Join(dir, "reports", "{task_id}", "{run_id}.md"),
		JournalPathTemplate: filepath.Join(dir, "journals", "{workflow}", "{run_id}.json"),
	}
	task := domain.Task{
		ID:                 "BIG-403-dsl",
		Source:             "linear",
		Title:              "Prod rollout",
		RiskLevel:          domain.RiskHigh,
		AcceptanceCriteria: []string{"rollback-plan"},
		ValidationPlan:     []string{"integration-test"},
	}

	result, err := engine.RunDefinition(task, definition, "run-dsl-2")
	if err != nil {
		t.Fatalf("run definition: %v", err)
	}
	if result.Execution.Status != "needs-approval" {
		t.Fatalf("expected execution status needs-approval, got %+v", result.Execution)
	}
	if result.Acceptance.Status != "accepted" || len(result.Acceptance.Approvals) != 1 || result.Acceptance.Approvals[0] != "release-manager" {
		t.Fatalf("unexpected acceptance result: %+v", result.Acceptance)
	}
	if _, err := os.Stat(result.ReportPath); err != nil {
		t.Fatalf("expected report path: %v", err)
	}
	if _, err := os.Stat(result.JournalPath); err != nil {
		t.Fatalf("expected journal path: %v", err)
	}

	events := recorder.EventsByTask(task.ID, 10)
	if len(events) != 2 {
		t.Fatalf("expected scheduler and approval audit events, got %+v", events)
	}
	if events[0].Type != domain.EventType(observability.SchedulerDecisionEvent) {
		t.Fatalf("unexpected scheduler event type: %+v", events[0])
	}
	if events[0].RunID != "run-dsl-2" || events[0].Payload["approved"] != false || events[0].Payload["medium"] != "vm" {
		t.Fatalf("unexpected scheduler event payload: %+v", events[0])
	}
	if events[1].Type != domain.EventType(observability.ApprovalRecordedEvent) {
		t.Fatalf("unexpected approval event type: %+v", events[1])
	}
	if events[1].RunID != "run-dsl-2" || events[1].Payload["acceptance_status"] != "accepted" {
		t.Fatalf("unexpected approval event payload: %+v", events[1])
	}
}
