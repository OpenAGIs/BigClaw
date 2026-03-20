package triage

import "encoding/json"

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

func (label Label) MarshalJSON() ([]byte, error) {
	payload := map[string]any{
		"name":       label.Name,
		"confidence": marshalConfidence(label.Confidence),
		"source":     label.Source,
	}
	return json.Marshal(payload)
}

func (label *Label) UnmarshalJSON(data []byte) error {
	type alias Label
	aux := struct {
		Confidence *float64 `json:"confidence"`
		*alias
	}{
		alias: (*alias)(label),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Confidence == nil {
		label.Confidence = 1
	} else {
		label.Confidence = *aux.Confidence
	}
	return nil
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

func (record TriageRecord) MarshalJSON() ([]byte, error) {
	payload := map[string]any{
		"triage_id":         record.TriageID,
		"task_id":           record.TaskID,
		"status":            marshalStatus(record.Status),
		"queue":             marshalQueue(record.Queue),
		"owner":             record.Owner,
		"summary":           record.Summary,
		"labels":            labelsOrEmpty(record.Labels),
		"related_run_id":    record.RelatedRunID,
		"escalation_target": record.EscalationTarget,
		"actions":           stringsOrEmpty(record.Actions),
	}
	return json.Marshal(payload)
}

func (record *TriageRecord) UnmarshalJSON(data []byte) error {
	type alias TriageRecord
	aux := struct {
		Status *Status `json:"status"`
		Queue  *string `json:"queue"`
		*alias
	}{
		alias: (*alias)(record),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Status == nil || *aux.Status == "" {
		record.Status = StatusOpen
	} else {
		record.Status = *aux.Status
	}
	if aux.Queue == nil || *aux.Queue == "" {
		record.Queue = "default"
	} else {
		record.Queue = *aux.Queue
	}
	if record.Labels == nil {
		record.Labels = []Label{}
	}
	if record.Actions == nil {
		record.Actions = []string{}
	}
	return nil
}

func marshalConfidence(confidence float64) float64 {
	if confidence == 0 {
		return 1
	}
	return confidence
}

func marshalStatus(status Status) string {
	if status == "" {
		return string(StatusOpen)
	}
	return string(status)
}

func marshalQueue(queue string) string {
	if queue == "" {
		return "default"
	}
	return queue
}

func labelsOrEmpty(values []Label) []Label {
	if values == nil {
		return []Label{}
	}
	return values
}

func stringsOrEmpty(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
