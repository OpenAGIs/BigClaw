package intake

import (
	"fmt"
	"strings"

	"bigclaw-go/internal/domain"
)

func MapPriority(priority string) int {
	switch strings.ToUpper(strings.TrimSpace(priority)) {
	case "P0":
		return int(domain.PriorityP0)
	case "P1":
		return int(domain.PriorityP1)
	default:
		return int(domain.PriorityP2)
	}
}

func MapSourceState(state string) domain.TaskState {
	normalized := normalizeSourceStatus(state)
	switch {
	case strings.Contains(normalized, "progress"), strings.Contains(normalized, "running"), strings.Contains(normalized, "active"), strings.Contains(normalized, "live"):
		return domain.TaskRunning
	case strings.Contains(normalized, "approval"), strings.Contains(normalized, "review"), strings.Contains(normalized, "paused"), strings.Contains(normalized, "stop"):
		return domain.TaskBlocked
	case strings.Contains(normalized, "done"), strings.Contains(normalized, "closed"), strings.Contains(normalized, "resolved"):
		return domain.TaskSucceeded
	case strings.Contains(normalized, "block"):
		return domain.TaskBlocked
	case strings.Contains(normalized, "fail"):
		return domain.TaskFailed
	default:
		return domain.TaskQueued
	}
}

func MapSourceIssueToTask(issue SourceIssue) domain.Task {
	source := trim(issue.Source)
	if source == "" {
		source = "unknown"
	}
	identifier := trim(issue.SourceID)
	if identifier == "" {
		identifier = fmt.Sprintf("%s:%s", source, slug(issue.Title))
	}
	riskLevel := domain.RiskLow
	if strings.Contains(strings.ToLower(issue.Title), "prod") {
		riskLevel = domain.RiskHigh
	}
	requiredTools := []string{"connector"}
	if strings.EqualFold(source, "github") {
		requiredTools = []string{"github"}
	}
	if strings.EqualFold(source, "clawhost") {
		requiredTools = []string{"clawhost"}
	}
	metadata := map[string]string{
		"source_id":    trim(issue.SourceID),
		"source_state": trim(issue.State),
	}
	for key, value := range issue.Metadata {
		if trimmedKey := trim(key); trimmedKey != "" {
			metadata[trimmedKey] = trim(value)
		}
	}
	if issueURL := trim(issue.Links["issue"]); issueURL != "" {
		metadata["issue_url"] = issueURL
		metadata["source_issue_url"] = issueURL
	}
	tenantID := firstNonEmpty(issue.Metadata["tenant_id"], issue.Metadata["tenant"], issue.Metadata["tenant_slug"])
	return domain.Task{
		ID:                 identifier,
		TraceID:            identifier,
		Source:             source,
		Title:              trim(issue.Title),
		Description:        trim(issue.Description),
		Labels:             append([]string(nil), issue.Labels...),
		Priority:           MapPriority(issue.Priority),
		State:              MapSourceState(issue.State),
		RiskLevel:          riskLevel,
		RequiredTools:      append([]string(nil), requiredTools...),
		AcceptanceCriteria: []string{"Synced from source issue"},
		ValidationPlan:     []string{"mapping-test"},
		TenantID:           tenantID,
		Metadata:           metadata,
	}
}

func trim(value string) string {
	return strings.TrimSpace(value)
}

func lowerTrim(value string) string {
	return strings.ToLower(trim(value))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := trim(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func slug(value string) string {
	value = lowerTrim(value)
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, "_", "-")
	return strings.Trim(value, "-")
}
