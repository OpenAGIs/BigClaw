package workflow

import (
	"encoding/json"
	"strings"

	"bigclaw-go/internal/domain"
)

type Step struct {
	Name     string         `json:"name"`
	Kind     string         `json:"kind"`
	Required bool           `json:"required"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

func (s *Step) UnmarshalJSON(data []byte) error {
	type rawStep struct {
		Name     string         `json:"name"`
		Kind     string         `json:"kind"`
		Required *bool          `json:"required"`
		Metadata map[string]any `json:"metadata"`
	}
	var raw rawStep
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	s.Name = strings.TrimSpace(raw.Name)
	s.Kind = strings.TrimSpace(raw.Kind)
	s.Required = true
	if raw.Required != nil {
		s.Required = *raw.Required
	}
	s.Metadata = cloneMetadata(raw.Metadata)
	if s.Metadata == nil {
		s.Metadata = map[string]any{}
	}
	return nil
}

type Definition struct {
	Name                string   `json:"name"`
	Steps               []Step   `json:"steps,omitempty"`
	ReportPathTemplate  string   `json:"report_path_template,omitempty"`
	JournalPathTemplate string   `json:"journal_path_template,omitempty"`
	ValidationEvidence  []string `json:"validation_evidence,omitempty"`
	Approvals           []string `json:"approvals,omitempty"`
}

func (d *Definition) UnmarshalJSON(data []byte) error {
	type alias Definition
	aux := struct {
		*alias
	}{
		alias: (*alias)(d),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if d.Steps == nil {
		d.Steps = []Step{}
	}
	if d.ValidationEvidence == nil {
		d.ValidationEvidence = []string{}
	}
	if d.Approvals == nil {
		d.Approvals = []string{}
	}
	return nil
}

func ParseDefinition(text string) (Definition, error) {
	var definition Definition
	err := json.Unmarshal([]byte(text), &definition)
	return definition, err
}

func (d Definition) RenderPath(template string, task domain.Task, runID string) string {
	template = strings.TrimSpace(template)
	if template == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"{workflow}", d.Name,
		"{task_id}", task.ID,
		"{source}", task.Source,
		"{run_id}", runID,
	)
	return replacer.Replace(template)
}

func (d Definition) RenderReportPath(task domain.Task, runID string) string {
	return d.RenderPath(d.ReportPathTemplate, task, runID)
}

func (d Definition) RenderJournalPath(task domain.Task, runID string) string {
	return d.RenderPath(d.JournalPathTemplate, task, runID)
}

func cloneMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(metadata))
	for key, value := range metadata {
		out[key] = value
	}
	return out
}
