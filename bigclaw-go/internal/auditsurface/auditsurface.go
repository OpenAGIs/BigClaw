package auditsurface

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sort"

	"bigclaw-go/internal/domain"
)

type AuditEventSpec struct {
	EventType      string   `json:"event_type"`
	Description    string   `json:"description"`
	Severity       string   `json:"severity"`
	RetentionDays  int      `json:"retention_days"`
	RequiredFields []string `json:"required_fields,omitempty"`
}

const (
	SchedulerDecisionEvent = "execution.scheduler_decision"
	ManualTakeoverEvent    = "execution.manual_takeover"
	ApprovalRecordedEvent  = "execution.approval_recorded"
	BudgetOverrideEvent    = "execution.budget_override"
	FlowHandoffEvent       = "execution.flow_handoff"
)

var P0AuditEventSpecs = []AuditEventSpec{
	{
		EventType:      SchedulerDecisionEvent,
		Description:    "Records the scheduler routing decision and risk context for a run.",
		Severity:       "info",
		RetentionDays:  180,
		RequiredFields: []string{"task_id", "run_id", "medium", "approved", "reason", "risk_level", "risk_score"},
	},
	{
		EventType:      ManualTakeoverEvent,
		Description:    "Captures escalation into a human takeover queue.",
		Severity:       "warn",
		RetentionDays:  365,
		RequiredFields: []string{"task_id", "run_id", "target_team", "reason", "requested_by", "required_approvals"},
	},
	{
		EventType:      ApprovalRecordedEvent,
		Description:    "Records explicit human approvals attached to a run or acceptance decision.",
		Severity:       "info",
		RetentionDays:  365,
		RequiredFields: []string{"task_id", "run_id", "approvals", "approval_count", "acceptance_status"},
	},
	{
		EventType:      BudgetOverrideEvent,
		Description:    "Captures a manual override to the run budget envelope.",
		Severity:       "warn",
		RetentionDays:  365,
		RequiredFields: []string{"task_id", "run_id", "requested_budget", "approved_budget", "override_actor", "reason"},
	},
	{
		EventType:      FlowHandoffEvent,
		Description:    "Captures ownership transfer between automated flow stages and teams.",
		Severity:       "info",
		RetentionDays:  180,
		RequiredFields: []string{"task_id", "run_id", "source_stage", "target_team", "reason", "collaboration_mode"},
	},
}

func MissingRequiredFields(eventType string, details map[string]any) []string {
	for _, spec := range P0AuditEventSpecs {
		if spec.EventType != eventType {
			continue
		}
		missing := make([]string, 0)
		for _, field := range spec.RequiredFields {
			if _, ok := details[field]; !ok {
				missing = append(missing, field)
			}
		}
		return missing
	}
	return nil
}

type AuditEntry struct {
	Action  string         `json:"action"`
	Actor   string         `json:"actor"`
	Outcome string         `json:"outcome"`
	Details map[string]any `json:"details,omitempty"`
}

type LedgerEntry struct {
	RunID   string       `json:"run_id"`
	TaskID  string       `json:"task_id"`
	Source  string       `json:"source,omitempty"`
	Summary string       `json:"summary,omitempty"`
	Audits  []AuditEntry `json:"audits,omitempty"`
}

type ObservabilityLedger struct {
	path    string
	entries []LedgerEntry
}

func NewObservabilityLedger(path string) *ObservabilityLedger {
	return &ObservabilityLedger{path: path}
}

func (l *ObservabilityLedger) Record(entry LedgerEntry) error {
	l.entries = append(l.entries, entry)
	if l.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(l.entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.path, body, 0o644)
}

func (l *ObservabilityLedger) Load() ([]LedgerEntry, error) {
	if len(l.entries) > 0 {
		out := make([]LedgerEntry, len(l.entries))
		copy(out, l.entries)
		return out, nil
	}
	if l.path == "" {
		return nil, nil
	}
	body, err := os.ReadFile(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entries []LedgerEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

type TaskRun struct {
	TaskID string
	RunID  string
	Medium string
	Audits []AuditEntry
}

func NewTaskRun(task domain.Task, runID, medium string) TaskRun {
	return TaskRun{TaskID: task.ID, RunID: runID, Medium: medium}
}

func (r *TaskRun) AuditSpecEvent(action, actor, outcome string, details map[string]any) error {
	if missing := MissingRequiredFields(action, details); len(missing) > 0 {
		return &MissingFieldsError{Action: action, Fields: missing}
	}
	r.Audits = append(r.Audits, AuditEntry{Action: action, Actor: actor, Outcome: outcome, Details: cloneMap(details)})
	return nil
}

type MissingFieldsError struct {
	Action string
	Fields []string
}

func (e *MissingFieldsError) Error() string {
	return "audit event " + e.Action + " missing required fields"
}

type SchedulerDecision struct {
	Medium   string
	Approved bool
	Reason   string
}

type HandoffRequest struct {
	TargetTeam        string
	Reason            string
	RequiredApprovals []string
}

type Run struct {
	Status string
}

type ExecutionRecord struct {
	Decision       SchedulerDecision
	Run            Run
	HandoffRequest *HandoffRequest
}

type Scheduler struct{}

func (Scheduler) Execute(task domain.Task, runID string, ledger *ObservabilityLedger) (ExecutionRecord, error) {
	riskScore := scoreTask(task)
	decision := SchedulerDecision{
		Medium:   "browser",
		Approved: true,
		Reason:   "coordinated release handling",
	}
	handoff := &HandoffRequest{
		TargetTeam:        "operations",
		Reason:            "manual review required",
		RequiredApprovals: []string{"security-review"},
	}
	entry := LedgerEntry{
		RunID:   runID,
		TaskID:  task.ID,
		Source:  task.Source,
		Summary: "handoff requested",
		Audits: []AuditEntry{
			{
				Action:  SchedulerDecisionEvent,
				Actor:   "scheduler",
				Outcome: "recorded",
				Details: map[string]any{"task_id": task.ID, "run_id": runID, "medium": decision.Medium, "approved": decision.Approved, "reason": decision.Reason, "risk_level": task.RiskLevel, "risk_score": riskScore},
			},
		},
	}
	if task.BudgetOverrideActor != "" || task.BudgetOverrideReason != "" || task.BudgetOverrideAmount > 0 {
		requested := float64(task.BudgetCents) / 100
		entry.Audits = append(entry.Audits, AuditEntry{
			Action:  BudgetOverrideEvent,
			Actor:   "scheduler",
			Outcome: "recorded",
			Details: map[string]any{"task_id": task.ID, "run_id": runID, "requested_budget": requested, "approved_budget": requested + task.BudgetOverrideAmount, "override_actor": task.BudgetOverrideActor, "reason": task.BudgetOverrideReason},
		})
	}
	entry.Audits = append(entry.Audits,
		AuditEntry{
			Action:  ManualTakeoverEvent,
			Actor:   "scheduler",
			Outcome: "pending",
			Details: map[string]any{"task_id": task.ID, "run_id": runID, "target_team": handoff.TargetTeam, "reason": handoff.Reason, "requested_by": "scheduler", "required_approvals": append([]string(nil), handoff.RequiredApprovals...)},
		},
		AuditEntry{
			Action:  FlowHandoffEvent,
			Actor:   "scheduler",
			Outcome: "ready",
			Details: map[string]any{"task_id": task.ID, "run_id": runID, "source_stage": "scheduler", "target_team": handoff.TargetTeam, "reason": handoff.Reason, "collaboration_mode": "cross-functional"},
		},
	)
	if err := ledger.Record(entry); err != nil {
		return ExecutionRecord{}, err
	}
	return ExecutionRecord{
		Decision:       decision,
		Run:            Run{Status: "approved"},
		HandoffRequest: handoff,
	}, nil
}

type WorkflowEngine struct{}

func (WorkflowEngine) Run(task domain.Task, runID string, ledger *ObservabilityLedger, approvals []string, validationEvidence []string) error {
	acceptanceStatus := "accepted"
	if len(approvals) == 0 || len(validationEvidence) == 0 {
		acceptanceStatus = "pending"
	}
	entry := LedgerEntry{
		RunID:   runID,
		TaskID:  task.ID,
		Source:  task.Source,
		Summary: "workflow acceptance evaluated",
	}
	if len(approvals) > 0 {
		sorted := append([]string(nil), approvals...)
		sort.Strings(sorted)
		entry.Audits = append(entry.Audits, AuditEntry{
			Action:  ApprovalRecordedEvent,
			Actor:   "workflow-engine",
			Outcome: "recorded",
			Details: map[string]any{"task_id": task.ID, "run_id": runID, "approvals": sorted, "approval_count": len(sorted), "acceptance_status": acceptanceStatus},
		})
	}
	return ledger.Record(entry)
}

type OrchestrationCanvas struct {
	HandoffTeam string
}

type TakeoverRequest struct {
	RunID             string
	TaskID            string
	TargetTeam        string
	Reason            string
	RequiredApprovals []string
}

type TakeoverQueue struct {
	Period   string
	Requests []TakeoverRequest
}

func BuildOrchestrationCanvasFromLedgerEntry(entry LedgerEntry) OrchestrationCanvas {
	for index := len(entry.Audits) - 1; index >= 0; index-- {
		audit := entry.Audits[index]
		if audit.Action != ManualTakeoverEvent && audit.Action != FlowHandoffEvent && audit.Action != "orchestration.handoff" {
			continue
		}
		if team, ok := audit.Details["target_team"].(string); ok {
			return OrchestrationCanvas{HandoffTeam: team}
		}
	}
	return OrchestrationCanvas{}
}

func BuildTakeoverQueueFromLedger(entries []LedgerEntry, period string) TakeoverQueue {
	queue := TakeoverQueue{Period: period, Requests: make([]TakeoverRequest, 0)}
	for _, entry := range entries {
		for index := len(entry.Audits) - 1; index >= 0; index-- {
			audit := entry.Audits[index]
			if audit.Action != ManualTakeoverEvent {
				continue
			}
			queue.Requests = append(queue.Requests, TakeoverRequest{
				RunID:             entry.RunID,
				TaskID:            entry.TaskID,
				TargetTeam:        stringValue(audit.Details["target_team"]),
				Reason:            stringValue(audit.Details["reason"]),
				RequiredApprovals: stringSliceValue(audit.Details["required_approvals"]),
			})
			break
		}
	}
	return queue
}

func scoreTask(task domain.Task) float64 {
	score := 0.0
	if task.RiskLevel == domain.RiskHigh {
		score += 60
	}
	score += float64(len(task.RequiredTools)) * 10
	score += float64(len(task.Labels)) * 5
	return math.Round(score*10) / 10
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func stringSliceValue(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}
