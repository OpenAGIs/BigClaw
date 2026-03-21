package workflow

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
)

type CloseoutInput struct {
	Task           domain.Task
	RunID          string
	Status         WorkflowRunStatus
	Executor       domain.ExecutorKind
	Message        string
	Artifacts      []string
	StartedAt      time.Time
	CompletedAt    time.Time
	RetryScheduled bool
}

type Closeout struct {
	Run                WorkflowRun `json:"workflow_run"`
	ReportPath         string      `json:"report_path,omitempty"`
	JournalPath        string      `json:"journal_path,omitempty"`
	ValidationEvidence []string    `json:"validation_evidence,omitempty"`
	RequiredApprovals  []string    `json:"required_approvals,omitempty"`
}

func BuildCloseout(input CloseoutInput) Closeout {
	definition := definitionForTask(input.Task)
	reportPath := definition.RenderReportPath(input.Task, input.RunID)
	journalPath := definition.RenderJournalPath(input.Task, input.RunID)
	validationEvidence := validationEvidenceForTask(input.Task, definition)
	requiredApprovals := approvalsForTask(input.Task, definition)

	outputs := map[string]any{}
	if message := strings.TrimSpace(input.Message); message != "" {
		outputs["message"] = message
	}
	if executor := strings.TrimSpace(string(input.Executor)); executor != "" {
		outputs["executor"] = input.Executor
	}
	if len(input.Artifacts) > 0 {
		outputs["artifacts"] = append([]string(nil), input.Artifacts...)
	}
	if reportPath != "" {
		outputs["report_path"] = reportPath
	}
	if journalPath != "" {
		outputs["journal_path"] = journalPath
	}
	if len(validationEvidence) > 0 {
		outputs["validation_evidence"] = append([]string(nil), validationEvidence...)
	}
	if len(requiredApprovals) > 0 {
		outputs["required_approvals"] = append([]string(nil), requiredApprovals...)
	}
	if input.RetryScheduled {
		outputs["retry_scheduled"] = true
	}
	if len(outputs) == 0 {
		outputs = nil
	}

	return Closeout{
		Run: WorkflowRun{
			RunID:        strings.TrimSpace(input.RunID),
			TemplateID:   templateIDForTask(input.Task, definition),
			TaskID:       strings.TrimSpace(input.Task.ID),
			Status:       input.Status,
			TriggeredBy:  firstNonEmpty(input.Task.Metadata["triggered_by"], input.Task.Metadata["created_by"], input.Task.Metadata["owner"]),
			StartedAt:    timestampRFC3339(input.StartedAt),
			CompletedAt:  timestampRFC3339(input.CompletedAt),
			Outputs:      outputs,
			ApprovalRefs: append([]string(nil), requiredApprovals...),
		},
		ReportPath:         reportPath,
		JournalPath:        journalPath,
		ValidationEvidence: append([]string(nil), validationEvidence...),
		RequiredApprovals:  append([]string(nil), requiredApprovals...),
	}
}

func definitionForTask(task domain.Task) Definition {
	definition, _ := parseDefinitionMetadata(task.Metadata)
	if definition.Name == "" {
		definition.Name = firstNonEmpty(task.Metadata["workflow"], task.Metadata["workflow_id"], task.Metadata["flow"], task.Metadata["template"], task.Source)
	}
	if definition.ReportPathTemplate == "" {
		definition.ReportPathTemplate = firstNonEmpty(task.Metadata["report_path_template"], task.Metadata["workflow_report_path_template"])
	}
	if definition.JournalPathTemplate == "" {
		definition.JournalPathTemplate = firstNonEmpty(task.Metadata["journal_path_template"], task.Metadata["workflow_journal_path_template"])
	}
	if len(definition.ValidationEvidence) == 0 {
		definition.ValidationEvidence = metadataStringList(task.Metadata, "validation_evidence", "workflow_validation_evidence")
	}
	if len(definition.Approvals) == 0 {
		definition.Approvals = metadataStringList(task.Metadata, "required_approvals", "workflow_approvals", "approvals", "approval")
	}
	return definition
}

func parseDefinitionMetadata(metadata map[string]string) (Definition, bool) {
	for _, key := range []string{"workflow_definition", "workflow_definition_json"} {
		raw := strings.TrimSpace(metadata[key])
		if raw == "" {
			continue
		}
		definition, err := ParseDefinition(raw)
		if err == nil {
			return definition, true
		}
	}
	return Definition{}, false
}

func validationEvidenceForTask(task domain.Task, definition Definition) []string {
	if len(definition.ValidationEvidence) > 0 {
		return append([]string(nil), definition.ValidationEvidence...)
	}
	return normalizeCloseoutStrings(append(append([]string(nil), task.ValidationPlan...), task.AcceptanceCriteria...))
}

func approvalsForTask(task domain.Task, definition Definition) []string {
	if len(definition.Approvals) > 0 {
		return append([]string(nil), definition.Approvals...)
	}
	return metadataStringList(task.Metadata, "required_approvals", "workflow_approvals", "approvals", "approval")
}

func templateIDForTask(task domain.Task, definition Definition) string {
	return firstNonEmpty(task.Metadata["template"], task.Metadata["flow_template_id"], task.Metadata["workflow_id"], definition.Name, task.Source)
}

func metadataStringList(metadata map[string]string, keys ...string) []string {
	for _, key := range keys {
		if values := parseStringList(metadata[key]); len(values) > 0 {
			return values
		}
	}
	return nil
}

func parseStringList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if strings.HasPrefix(raw, "[") {
		var values []string
		if err := json.Unmarshal([]byte(raw), &values); err == nil {
			return normalizeCloseoutStrings(values)
		}
		var generic []any
		if err := json.Unmarshal([]byte(raw), &generic); err == nil {
			values = make([]string, 0, len(generic))
			for _, value := range generic {
				values = append(values, strings.TrimSpace(toString(value)))
			}
			return normalizeCloseoutStrings(values)
		}
	}
	return normalizeCloseoutStrings(strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r'
	}))
}

func normalizeCloseoutStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func timestampRFC3339(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func toString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case nil:
		return ""
	default:
		return fmt.Sprint(typed)
	}
}
