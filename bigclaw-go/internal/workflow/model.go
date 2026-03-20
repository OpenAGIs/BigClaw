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
