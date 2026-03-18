package observability

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

func GetAuditEventSpec(eventType string) (AuditEventSpec, bool) {
	for _, spec := range P0AuditEventSpecs {
		if spec.EventType == eventType {
			return spec, true
		}
	}
	return AuditEventSpec{}, false
}

func MissingRequiredFields(eventType string, details map[string]any) []string {
	spec, ok := GetAuditEventSpec(eventType)
	if !ok {
		return nil
	}
	missing := make([]string, 0)
	for _, field := range spec.RequiredFields {
		if _, ok := details[field]; !ok {
			missing = append(missing, field)
		}
	}
	return missing
}
