package workflow

import (
	"encoding/json"

	"bigclaw-go/internal/domain"
)

type WorkflowTrigger string

const (
	WorkflowTriggerManual    WorkflowTrigger = "manual"
	WorkflowTriggerScheduled WorkflowTrigger = "scheduled"
	WorkflowTriggerEvent     WorkflowTrigger = "event"
)

type WorkflowRunStatus string

const (
	WorkflowRunQueued    WorkflowRunStatus = "queued"
	WorkflowRunRunning   WorkflowRunStatus = "running"
	WorkflowRunSucceeded WorkflowRunStatus = "succeeded"
	WorkflowRunFailed    WorkflowRunStatus = "failed"
	WorkflowRunCanceled  WorkflowRunStatus = "canceled"
)

type WorkflowStepStatus string

const (
	WorkflowStepPending   WorkflowStepStatus = "pending"
	WorkflowStepRunning   WorkflowStepStatus = "running"
	WorkflowStepSucceeded WorkflowStepStatus = "succeeded"
	WorkflowStepFailed    WorkflowStepStatus = "failed"
	WorkflowStepSkipped   WorkflowStepStatus = "skipped"
)

type WorkflowTemplateStep struct {
	StepID        string         `json:"step_id"`
	Name          string         `json:"name"`
	Kind          string         `json:"kind"`
	RequiredTools []string       `json:"required_tools,omitempty"`
	Approvals     []string       `json:"approvals,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type WorkflowTemplate struct {
	TemplateID  string                 `json:"template_id"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description,omitempty"`
	Trigger     WorkflowTrigger        `json:"trigger,omitempty"`
	DefaultRisk domain.RiskLevel       `json:"default_risk,omitempty"`
	Steps       []WorkflowTemplateStep `json:"steps,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Active      bool                   `json:"active"`
}

type WorkflowStepRun struct {
	StepID      string             `json:"step_id"`
	Status      WorkflowStepStatus `json:"status,omitempty"`
	Actor       string             `json:"actor,omitempty"`
	StartedAt   string             `json:"started_at,omitempty"`
	CompletedAt string             `json:"completed_at,omitempty"`
	Output      map[string]any     `json:"output,omitempty"`
}

type WorkflowRun struct {
	RunID        string            `json:"run_id"`
	TemplateID   string            `json:"template_id"`
	TaskID       string            `json:"task_id"`
	Status       WorkflowRunStatus `json:"status,omitempty"`
	TriggeredBy  string            `json:"triggered_by,omitempty"`
	StartedAt    string            `json:"started_at,omitempty"`
	CompletedAt  string            `json:"completed_at,omitempty"`
	Steps        []WorkflowStepRun `json:"steps,omitempty"`
	Outputs      map[string]any    `json:"outputs,omitempty"`
	ApprovalRefs []string          `json:"approval_refs,omitempty"`
}

func (step WorkflowTemplateStep) MarshalJSON() ([]byte, error) {
	payload := map[string]any{
		"step_id":        step.StepID,
		"name":           step.Name,
		"kind":           step.Kind,
		"required_tools": workflowStringsOrEmpty(step.RequiredTools),
		"approvals":      workflowStringsOrEmpty(step.Approvals),
		"metadata":       workflowMetadataOrEmpty(step.Metadata),
	}
	return json.Marshal(payload)
}

func (template WorkflowTemplate) MarshalJSON() ([]byte, error) {
	payload := map[string]any{
		"template_id":  template.TemplateID,
		"name":         template.Name,
		"version":      template.Version,
		"description":  template.Description,
		"trigger":      marshalWorkflowTrigger(template.Trigger),
		"default_risk": marshalWorkflowRisk(template.DefaultRisk),
		"steps":        stepTemplatesOrEmpty(template.Steps),
		"tags":         workflowStringsOrEmpty(template.Tags),
		"active":       template.Active,
	}
	return json.Marshal(payload)
}

func (run WorkflowStepRun) MarshalJSON() ([]byte, error) {
	payload := map[string]any{
		"step_id":      run.StepID,
		"status":       marshalWorkflowStepStatus(run.Status),
		"actor":        run.Actor,
		"started_at":   run.StartedAt,
		"completed_at": run.CompletedAt,
		"output":       workflowMetadataOrEmpty(run.Output),
	}
	return json.Marshal(payload)
}

func (run WorkflowRun) MarshalJSON() ([]byte, error) {
	payload := map[string]any{
		"run_id":        run.RunID,
		"template_id":   run.TemplateID,
		"task_id":       run.TaskID,
		"status":        marshalWorkflowRunStatus(run.Status),
		"triggered_by":  run.TriggeredBy,
		"started_at":    run.StartedAt,
		"completed_at":  run.CompletedAt,
		"steps":         stepRunsOrEmpty(run.Steps),
		"outputs":       workflowMetadataOrEmpty(run.Outputs),
		"approval_refs": workflowStringsOrEmpty(run.ApprovalRefs),
	}
	return json.Marshal(payload)
}

func (template *WorkflowTemplate) UnmarshalJSON(data []byte) error {
	type alias WorkflowTemplate
	aux := struct {
		Trigger     *WorkflowTrigger  `json:"trigger"`
		DefaultRisk *domain.RiskLevel `json:"default_risk"`
		Active      *bool             `json:"active"`
		*alias
	}{
		alias: (*alias)(template),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Trigger == nil {
		template.Trigger = WorkflowTriggerManual
	} else {
		template.Trigger = *aux.Trigger
	}
	if aux.DefaultRisk == nil {
		template.DefaultRisk = domain.RiskLow
	} else {
		template.DefaultRisk = *aux.DefaultRisk
	}
	if aux.Active == nil {
		template.Active = true
	} else {
		template.Active = *aux.Active
	}
	if template.Steps == nil {
		template.Steps = []WorkflowTemplateStep{}
	}
	if template.Tags == nil {
		template.Tags = []string{}
	}
	for index := range template.Steps {
		if template.Steps[index].RequiredTools == nil {
			template.Steps[index].RequiredTools = []string{}
		}
		if template.Steps[index].Approvals == nil {
			template.Steps[index].Approvals = []string{}
		}
		if template.Steps[index].Metadata == nil {
			template.Steps[index].Metadata = map[string]any{}
		}
	}
	return nil
}

func (run *WorkflowStepRun) UnmarshalJSON(data []byte) error {
	type alias WorkflowStepRun
	aux := struct {
		Status *WorkflowStepStatus `json:"status"`
		*alias
	}{
		alias: (*alias)(run),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Status == nil {
		run.Status = WorkflowStepPending
	} else {
		run.Status = *aux.Status
	}
	if run.Output == nil {
		run.Output = map[string]any{}
	}
	return nil
}

func (run *WorkflowRun) UnmarshalJSON(data []byte) error {
	type alias WorkflowRun
	aux := struct {
		Status *WorkflowRunStatus `json:"status"`
		*alias
	}{
		alias: (*alias)(run),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Status == nil {
		run.Status = WorkflowRunQueued
	} else {
		run.Status = *aux.Status
	}
	if run.Steps == nil {
		run.Steps = []WorkflowStepRun{}
	}
	if run.Outputs == nil {
		run.Outputs = map[string]any{}
	}
	if run.ApprovalRefs == nil {
		run.ApprovalRefs = []string{}
	}
	return nil
}

func marshalWorkflowTrigger(trigger WorkflowTrigger) string {
	if trigger == "" {
		return string(WorkflowTriggerManual)
	}
	return string(trigger)
}

func marshalWorkflowRisk(level domain.RiskLevel) string {
	if level == "" {
		return string(domain.RiskLow)
	}
	return string(level)
}

func marshalWorkflowStepStatus(status WorkflowStepStatus) string {
	if status == "" {
		return string(WorkflowStepPending)
	}
	return string(status)
}

func marshalWorkflowRunStatus(status WorkflowRunStatus) string {
	if status == "" {
		return string(WorkflowRunQueued)
	}
	return string(status)
}

func workflowStringsOrEmpty(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func stepTemplatesOrEmpty(values []WorkflowTemplateStep) []WorkflowTemplateStep {
	if values == nil {
		return []WorkflowTemplateStep{}
	}
	return values
}

func stepRunsOrEmpty(values []WorkflowStepRun) []WorkflowStepRun {
	if values == nil {
		return []WorkflowStepRun{}
	}
	return values
}

func workflowMetadataOrEmpty(values map[string]any) map[string]any {
	if values == nil {
		return map[string]any{}
	}
	return values
}
