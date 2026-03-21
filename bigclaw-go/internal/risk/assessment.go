package risk

import (
	"encoding/json"

	"bigclaw-go/internal/domain"
)

type Signal struct {
	Name     string         `json:"name"`
	Score    int            `json:"score"`
	Reason   string         `json:"reason"`
	Source   string         `json:"source,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

func (signal Signal) MarshalJSON() ([]byte, error) {
	payload := map[string]any{
		"name":     signal.Name,
		"score":    signal.Score,
		"reason":   signal.Reason,
		"source":   signal.Source,
		"metadata": metadataOrEmpty(signal.Metadata),
	}
	return json.Marshal(payload)
}

func (signal *Signal) UnmarshalJSON(data []byte) error {
	type alias Signal
	aux := struct {
		*alias
	}{
		alias: (*alias)(signal),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if signal.Metadata == nil {
		signal.Metadata = map[string]any{}
	}
	return nil
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

func (assessment Assessment) MarshalJSON() ([]byte, error) {
	payload := map[string]any{
		"assessment_id":     assessment.AssessmentID,
		"task_id":           assessment.TaskID,
		"level":             marshalRiskLevel(assessment.Level),
		"total_score":       assessment.TotalScore,
		"requires_approval": assessment.RequiresApproval,
		"signals":           signalsOrEmpty(assessment.Signals),
		"mitigations":       stringsOrEmpty(assessment.Mitigations),
		"reviewer":          assessment.Reviewer,
		"notes":             assessment.Notes,
	}
	return json.Marshal(payload)
}

func (assessment *Assessment) UnmarshalJSON(data []byte) error {
	type alias Assessment
	aux := struct {
		Level *domain.RiskLevel `json:"level"`
		*alias
	}{
		alias: (*alias)(assessment),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Level == nil || *aux.Level == "" {
		assessment.Level = domain.RiskLow
	} else {
		assessment.Level = *aux.Level
	}
	if assessment.Signals == nil {
		assessment.Signals = []Signal{}
	}
	if assessment.Mitigations == nil {
		assessment.Mitigations = []string{}
	}
	return nil
}

func marshalRiskLevel(level domain.RiskLevel) string {
	if level == "" {
		return string(domain.RiskLow)
	}
	return string(level)
}

func signalsOrEmpty(values []Signal) []Signal {
	if values == nil {
		return []Signal{}
	}
	return values
}

func stringsOrEmpty(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func metadataOrEmpty(values map[string]any) map[string]any {
	if values == nil {
		return map[string]any{}
	}
	return values
}
