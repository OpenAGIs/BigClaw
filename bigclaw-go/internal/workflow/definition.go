package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
)

var validDefinitionStepKinds = []string{
	"scheduler",
	"approval",
	"orchestration",
	"report",
	"closeout",
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
	for _, step := range d.Steps {
		kind := strings.TrimSpace(step.Kind)
		if kind == "" || slices.Contains(validDefinitionStepKinds, kind) {
			continue
		}
		if !slices.Contains(invalid, kind) {
			invalid = append(invalid, kind)
		}
	}
	if len(invalid) == 0 {
		return nil
	}
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

type DefinitionExecution struct {
	Run WorkflowRun `json:"run"`
}

type DefinitionRunResult struct {
	Execution   DefinitionExecution `json:"execution"`
	Acceptance  AcceptanceDecision  `json:"acceptance"`
	Journal     WorkpadJournal      `json:"journal"`
	JournalPath string              `json:"journal_path,omitempty"`
	ReportPath  string              `json:"report_path,omitempty"`
}

type DefinitionEngine struct {
	Gate AcceptanceGate
	Now  func() time.Time
}

func (e DefinitionEngine) RunDefinition(task domain.Task, definition Definition, runID string) (DefinitionRunResult, error) {
	if err := definition.Validate(); err != nil {
		return DefinitionRunResult{}, err
	}
	now := time.Now()
	if e.Now != nil {
		now = e.Now()
	}
	reportPath := definition.RenderReportPath(task, runID)
	if reportPath != "" {
		if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
			return DefinitionRunResult{}, err
		}
		if err := os.WriteFile(reportPath, []byte("# Workflow Definition Report\n"), 0o644); err != nil {
			return DefinitionRunResult{}, err
		}
	}

	outcome := ExecutionOutcome{Approved: true, Status: "completed"}
	runStatus := WorkflowRunSucceeded
	if task.RiskLevel == domain.RiskHigh || definitionRequiresApproval(definition) {
		outcome = ExecutionOutcome{Approved: false, Status: "needs-approval"}
		runStatus = WorkflowRunQueued
	}
	acceptance := e.Gate.Evaluate(task, outcome, definition.ValidationEvidence, definition.Approvals, "")
	closeout := BuildCloseout(CloseoutInput{
		Task:        taskWithDefinition(task, definition),
		RunID:       runID,
		Status:      runStatus,
		StartedAt:   now,
		CompletedAt: now,
	})

	journal := WorkpadJournal{TaskID: task.ID, RunID: runID, Now: e.Now}
	journal.Record("definition", "validated", map[string]any{"workflow": definition.Name})
	journal.Record("acceptance", acceptance.Status, map[string]any{"passed": acceptance.Passed})
	journalPath := definition.RenderJournalPath(task, runID)
	resolvedJournalPath := ""
	if journalPath != "" {
		written, err := journal.Write(journalPath)
		if err != nil {
			return DefinitionRunResult{}, err
		}
		resolvedJournalPath = written
	}

	return DefinitionRunResult{
		Execution:   DefinitionExecution{Run: closeout.Run},
		Acceptance:  acceptance,
		Journal:     journal,
		JournalPath: resolvedJournalPath,
		ReportPath:  reportPath,
	}, nil
}

func definitionRequiresApproval(definition Definition) bool {
	for _, step := range definition.Steps {
		if strings.TrimSpace(step.Kind) == "approval" {
			return true
		}
	}
	return false
}

func taskWithDefinition(task domain.Task, definition Definition) domain.Task {
	copy := task
	if copy.Metadata == nil {
		copy.Metadata = map[string]string{}
	}
	if raw, err := json.Marshal(definition); err == nil {
		copy.Metadata["workflow_definition"] = string(raw)
	}
	return copy
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
