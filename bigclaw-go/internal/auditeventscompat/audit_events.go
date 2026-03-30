package auditeventscompat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	SchedulerDecisionEvent = "execution.scheduler_decision"
	ManualTakeoverEvent    = "execution.manual_takeover"
	ApprovalRecordedEvent  = "execution.approval_recorded"
	BudgetOverrideEvent    = "execution.budget_override"
	FlowHandoffEvent       = "execution.flow_handoff"
)

type AuditEventSpec struct {
	EventType      string   `json:"event_type"`
	Description    string   `json:"description"`
	Severity       string   `json:"severity"`
	RetentionDays  int      `json:"retention_days"`
	RequiredFields []string `json:"required_fields,omitempty"`
}

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

type Task struct {
	TaskID               string
	Source               string
	Title                string
	Description          string
	Labels               []string
	Priority             int
	RiskLevel            string
	RequiredTools        []string
	Budget               float64
	BudgetOverrideActor  string
	BudgetOverrideReason string
	BudgetOverrideAmount float64
	AcceptanceCriteria   []string
	ValidationPlan       []string
}

type AuditEntry struct {
	Action  string         `json:"action"`
	Actor   string         `json:"actor"`
	Outcome string         `json:"outcome"`
	Details map[string]any `json:"details,omitempty"`
}

type TaskRun struct {
	RunID   string       `json:"run_id"`
	TaskID  string       `json:"task_id"`
	Source  string       `json:"source"`
	Medium  string       `json:"medium"`
	Status  string       `json:"status"`
	Summary string       `json:"summary"`
	Audits  []AuditEntry `json:"audits,omitempty"`
}

func NewRun(task Task, runID, medium string) *TaskRun {
	return &TaskRun{
		RunID:   runID,
		TaskID:  task.TaskID,
		Source:  task.Source,
		Medium:  medium,
		Status:  "running",
		Summary: "",
	}
}

func (r *TaskRun) Finalize(status, summary string) {
	r.Status = status
	r.Summary = summary
}

func (r *TaskRun) Audit(action, actor, outcome string, details map[string]any) {
	r.Audits = append(r.Audits, AuditEntry{
		Action:  action,
		Actor:   actor,
		Outcome: outcome,
		Details: cloneMap(details),
	})
}

func (r *TaskRun) AuditSpecEvent(eventType, actor, outcome string, details map[string]any) error {
	if missing := MissingRequiredFields(eventType, details); len(missing) > 0 {
		return fmt.Errorf("audit event %s missing required fields: %s", eventType, strings.Join(missing, ", "))
	}
	r.Audit(eventType, actor, outcome, details)
	return nil
}

type ObservabilityLedger struct {
	storagePath string
	entries     []TaskRun
}

func NewLedger(path string) *ObservabilityLedger {
	return &ObservabilityLedger{storagePath: path}
}

func (l *ObservabilityLedger) Append(run *TaskRun) error {
	l.entries = append(l.entries, *run)
	if strings.TrimSpace(l.storagePath) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(l.storagePath), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(l.entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.storagePath, body, 0o644)
}

func (l *ObservabilityLedger) Load() ([]map[string]any, error) {
	if len(l.entries) > 0 {
		return serializeRuns(l.entries)
	}
	if strings.TrimSpace(l.storagePath) == "" {
		return nil, nil
	}
	body, err := os.ReadFile(l.storagePath)
	if err != nil {
		return nil, err
	}
	var entries []TaskRun
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, err
	}
	l.entries = entries
	return serializeRuns(entries)
}

type SchedulerDecision struct {
	Medium   string
	Approved bool
	Reason   string
}

type HandoffRequest struct {
	TargetTeam        string
	Reason            string
	Status            string
	RequiredApprovals []string
}

type ExecutionRecord struct {
	Decision       SchedulerDecision
	Run            *TaskRun
	HandoffRequest *HandoffRequest
}

type Scheduler struct{}

func (s Scheduler) Execute(task Task, runID string, ledger *ObservabilityLedger) (ExecutionRecord, error) {
	riskLevel, riskScore := scoreTask(task)
	decision := decide(task, riskLevel)
	plan := buildPlan(task)
	handoff := buildHandoffRequest(decision, plan)
	run := NewRun(task, runID, decision.Medium)

	if err := run.AuditSpecEvent(SchedulerDecisionEvent, "scheduler", ternary(decision.Approved, "approved", "pending"), map[string]any{
		"task_id":    task.TaskID,
		"run_id":     runID,
		"medium":     decision.Medium,
		"approved":   decision.Approved,
		"reason":     decision.Reason,
		"risk_level": riskLevel,
		"risk_score": riskScore,
	}); err != nil {
		return ExecutionRecord{}, err
	}

	if strings.TrimSpace(task.BudgetOverrideReason) != "" {
		if err := run.AuditSpecEvent(BudgetOverrideEvent, firstNonEmpty(task.BudgetOverrideActor, "scheduler"), "recorded", map[string]any{
			"task_id":          task.TaskID,
			"run_id":           runID,
			"requested_budget": task.Budget,
			"approved_budget":  task.Budget + task.BudgetOverrideAmount,
			"override_actor":   firstNonEmpty(task.BudgetOverrideActor, "scheduler"),
			"reason":           task.BudgetOverrideReason,
		}); err != nil {
			return ExecutionRecord{}, err
		}
	}

	if handoff != nil {
		manualDetails := map[string]any{
			"task_id":            task.TaskID,
			"run_id":             runID,
			"target_team":        handoff.TargetTeam,
			"reason":             handoff.Reason,
			"requested_by":       "scheduler",
			"required_approvals": append([]string(nil), handoff.RequiredApprovals...),
		}
		if err := run.AuditSpecEvent(ManualTakeoverEvent, "scheduler", handoff.Status, manualDetails); err != nil {
			return ExecutionRecord{}, err
		}
		if err := run.AuditSpecEvent(FlowHandoffEvent, "scheduler", handoff.Status, map[string]any{
			"task_id":            task.TaskID,
			"run_id":             runID,
			"source_stage":       "scheduler",
			"target_team":        handoff.TargetTeam,
			"reason":             handoff.Reason,
			"collaboration_mode": plan.CollaborationMode,
			"required_approvals": append([]string(nil), handoff.RequiredApprovals...),
		}); err != nil {
			return ExecutionRecord{}, err
		}
	}

	if decision.Approved {
		run.Finalize("approved", decision.Reason)
	} else {
		run.Finalize("needs-approval", decision.Reason)
	}
	if ledger != nil {
		if err := ledger.Append(run); err != nil {
			return ExecutionRecord{}, err
		}
	}
	return ExecutionRecord{Decision: decision, Run: run, HandoffRequest: handoff}, nil
}

type WorkflowEngine struct{}

func (WorkflowEngine) Run(task Task, runID string, ledger *ObservabilityLedger, approvals, validationEvidence []string) error {
	run := NewRun(task, runID, "docker")
	if err := run.AuditSpecEvent(ApprovalRecordedEvent, "workflow", "recorded", map[string]any{
		"task_id":           task.TaskID,
		"run_id":            runID,
		"approvals":         append([]string(nil), approvals...),
		"approval_count":    len(approvals),
		"acceptance_status": "accepted",
	}); err != nil {
		return err
	}
	run.Finalize("approved", "acceptance criteria and validation plan satisfied")
	if ledger != nil {
		return ledger.Append(run)
	}
	return nil
}

type OrchestrationCanvas struct {
	HandoffTeam string
}

type TakeoverRequest struct {
	RunID             string
	TaskID            string
	Source            string
	TargetTeam        string
	Status            string
	Reason            string
	RequiredApprovals []string
}

type TakeoverQueue struct {
	Name     string
	Period   string
	Requests []TakeoverRequest
}

func BuildOrchestrationCanvasFromLedgerEntry(entry map[string]any) OrchestrationCanvas {
	audit := latestHandoffAudit(entry["audits"])
	if audit == nil {
		return OrchestrationCanvas{HandoffTeam: "none"}
	}
	details := mapValue(audit["details"])
	return OrchestrationCanvas{HandoffTeam: asString(details["target_team"])}
}

func BuildTakeoverQueueFromLedger(entries []map[string]any, name, period string) TakeoverQueue {
	requests := make([]TakeoverRequest, 0)
	for _, entry := range entries {
		audit := latestHandoffAudit(entry["audits"])
		if audit == nil {
			continue
		}
		details := mapValue(audit["details"])
		requests = append(requests, TakeoverRequest{
			RunID:             asString(entry["run_id"]),
			TaskID:            asString(entry["task_id"]),
			Source:            asString(entry["source"]),
			TargetTeam:        firstNonEmpty(asString(details["target_team"]), "operations"),
			Status:            firstNonEmpty(asString(audit["outcome"]), "pending"),
			Reason:            firstNonEmpty(asString(details["reason"]), asString(entry["summary"])),
			RequiredApprovals: stringSlice(details["required_approvals"]),
		})
	}
	sort.Slice(requests, func(i, j int) bool {
		if requests[i].TargetTeam == requests[j].TargetTeam {
			return requests[i].RunID < requests[j].RunID
		}
		return requests[i].TargetTeam < requests[j].TargetTeam
	})
	return TakeoverQueue{Name: name, Period: period, Requests: requests}
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

type orchestrationPlan struct {
	CollaborationMode string
	RequiredApprovals []string
	UpgradeRequired   bool
}

func buildPlan(task Task) orchestrationPlan {
	needsUpgrade := false
	for _, label := range task.Labels {
		switch strings.TrimSpace(label) {
		case "customer", "data":
			needsUpgrade = true
		}
	}
	for _, tool := range task.RequiredTools {
		if strings.TrimSpace(tool) == "sql" {
			needsUpgrade = true
		}
	}
	mode := "single-team"
	if needsUpgrade {
		mode = "cross-functional"
	}
	approvals := []string{}
	if strings.EqualFold(strings.TrimSpace(task.RiskLevel), "high") {
		approvals = append(approvals, "security-review")
	}
	return orchestrationPlan{
		CollaborationMode: mode,
		RequiredApprovals: approvals,
		UpgradeRequired:   needsUpgrade,
	}
}

func buildHandoffRequest(decision SchedulerDecision, plan orchestrationPlan) *HandoffRequest {
	if !decision.Approved {
		approvals := append([]string(nil), plan.RequiredApprovals...)
		if len(approvals) == 0 {
			approvals = []string{"security-review"}
		}
		return &HandoffRequest{
			TargetTeam:        "security",
			Reason:            decision.Reason,
			Status:            "pending",
			RequiredApprovals: approvals,
		}
	}
	if plan.UpgradeRequired {
		return &HandoffRequest{
			TargetTeam:        "operations",
			Reason:            "premium tier required for advanced cross-department orchestration",
			Status:            "blocked",
			RequiredApprovals: []string{"ops-manager"},
		}
	}
	return nil
}

func decide(task Task, riskLevel string) SchedulerDecision {
	if riskLevel == "high" {
		return SchedulerDecision{Medium: "vm", Approved: false, Reason: "requires approval for high-risk task"}
	}
	for _, tool := range task.RequiredTools {
		if strings.TrimSpace(tool) == "browser" {
			return SchedulerDecision{Medium: "browser", Approved: true, Reason: "browser automation task"}
		}
	}
	if riskLevel == "medium" {
		return SchedulerDecision{Medium: "docker", Approved: true, Reason: "medium risk in docker"}
	}
	return SchedulerDecision{Medium: "docker", Approved: true, Reason: "default low risk path"}
}

func scoreTask(task Task) (string, int) {
	switch strings.ToLower(strings.TrimSpace(task.RiskLevel)) {
	case "high":
		return "high", 90
	case "medium":
		return "medium", 55
	default:
	}
	if task.Priority == 0 {
		return "medium", 60
	}
	return "low", 20
}

func serializeRuns(entries []TaskRun) ([]map[string]any, error) {
	body, err := json.Marshal(entries)
	if err != nil {
		return nil, err
	}
	var out []map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func latestHandoffAudit(raw any) map[string]any {
	audits, ok := raw.([]any)
	if !ok {
		if typed, ok := raw.([]map[string]any); ok {
			for _, action := range []string{ManualTakeoverEvent, FlowHandoffEvent, "orchestration.handoff"} {
				for i := len(typed) - 1; i >= 0; i-- {
					if asString(typed[i]["action"]) == action {
						return typed[i]
					}
				}
			}
		}
		return nil
	}
	for _, action := range []string{ManualTakeoverEvent, FlowHandoffEvent, "orchestration.handoff"} {
		for i := len(audits) - 1; i >= 0; i-- {
			item := mapValue(audits[i])
			if asString(item["action"]) == action {
				return item
			}
		}
	}
	return nil
}

func mapValue(v any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	if typed, ok := v.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

func stringSlice(v any) []string {
	switch typed := v.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, asString(item))
		}
		return out
	default:
		return nil
	}
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func ternary[T any](cond bool, yes, no T) T {
	if cond {
		return yes
	}
	return no
}
