package workflow

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"bigclaw-go/internal/domain"
)

var validStepKinds = map[string]struct{}{
	"scheduler":     {},
	"approval":      {},
	"orchestration": {},
	"report":        {},
	"closeout":      {},
}

type Step struct {
	Name     string         `json:"name"`
	Kind     string         `json:"kind"`
	Required bool           `json:"required"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

func (s Step) MarshalJSON() ([]byte, error) {
	payload := map[string]any{
		"name":     s.Name,
		"kind":     s.Kind,
		"required": s.Required,
		"metadata": cloneMetadata(s.Metadata),
	}
	return json.Marshal(payload)
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

func (d Definition) MarshalJSON() ([]byte, error) {
	payload := map[string]any{
		"name":                  d.Name,
		"steps":                 stepsOrEmpty(d.Steps),
		"report_path_template":  d.ReportPathTemplate,
		"journal_path_template": d.JournalPathTemplate,
		"validation_evidence":   stringsOrEmpty(d.ValidationEvidence),
		"approvals":             stringsOrEmpty(d.Approvals),
	}
	return json.Marshal(payload)
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

func (d Definition) Validate() error {
	invalid := make([]string, 0)
	seen := make(map[string]struct{})
	for _, step := range d.Steps {
		kind := strings.TrimSpace(step.Kind)
		if _, ok := validStepKinds[kind]; ok {
			continue
		}
		if _, ok := seen[kind]; ok {
			continue
		}
		seen[kind] = struct{}{}
		invalid = append(invalid, kind)
	}
	if len(invalid) == 0 {
		return nil
	}
	sort.Strings(invalid)
	return fmt.Errorf("invalid workflow step kind(s): %s", strings.Join(invalid, ", "))
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

func stringsOrEmpty(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func stepsOrEmpty(values []Step) []Step {
	if values == nil {
		return []Step{}
	}
	return values
}
