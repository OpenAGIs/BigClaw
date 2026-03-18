package risk

import "bigclaw-go/internal/domain"

type Signal struct {
	Name     string         `json:"name"`
	Score    int            `json:"score"`
	Reason   string         `json:"reason"`
	Source   string         `json:"source,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type Assessment struct {
	AssessmentID     string           `json:"assessment_id"`
	TaskID           string           `json:"task_id"`
	Level            domain.RiskLevel `json:"level"`
	TotalScore       int              `json:"total_score"`
	RequiresApproval bool             `json:"requires_approval"`
	Signals          []Signal         `json:"signals,omitempty"`
	Mitigations      []string         `json:"mitigations,omitempty"`
	Reviewer         string           `json:"reviewer,omitempty"`
	Notes            string           `json:"notes,omitempty"`
}
