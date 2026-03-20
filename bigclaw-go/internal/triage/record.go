package triage

type Status string

const (
	StatusOpen       Status = "open"
	StatusInProgress Status = "in-progress"
	StatusEscalated  Status = "escalated"
	StatusResolved   Status = "resolved"
)

type Label struct {
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
	Source     string  `json:"source,omitempty"`
}

type TriageRecord struct {
	TriageID         string   `json:"triage_id"`
	TaskID           string   `json:"task_id"`
	Status           Status   `json:"status"`
	Queue            string   `json:"queue,omitempty"`
	Owner            string   `json:"owner,omitempty"`
	Summary          string   `json:"summary,omitempty"`
	Labels           []Label  `json:"labels,omitempty"`
	RelatedRunID     string   `json:"related_run_id,omitempty"`
	EscalationTarget string   `json:"escalation_target,omitempty"`
	Actions          []string `json:"actions,omitempty"`
}
