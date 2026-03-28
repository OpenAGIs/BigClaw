package product

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

var allowedIssueCategories = []string{"ia", "metric", "permission", "ui"}
var allowedIssuePriorities = []string{"P0", "P1", "P2"}

type ArchivedIssue struct {
	FindingID string   `json:"finding_id"`
	Summary   string   `json:"summary"`
	Category  string   `json:"category"`
	Priority  string   `json:"priority"`
	Owner     string   `json:"owner,omitempty"`
	Surface   string   `json:"surface,omitempty"`
	Impact    string   `json:"impact,omitempty"`
	Status    string   `json:"status,omitempty"`
	Evidence  []string `json:"evidence,omitempty"`
}

func (issue ArchivedIssue) NormalizedCategory() string {
	return strings.ToLower(strings.TrimSpace(issue.Category))
}

func (issue ArchivedIssue) NormalizedPriority() string {
	return strings.ToUpper(strings.TrimSpace(issue.Priority))
}

func (issue ArchivedIssue) Resolved() bool {
	return strings.EqualFold(strings.TrimSpace(issue.Status), "resolved")
}

func (issue ArchivedIssue) MarshalJSON() ([]byte, error) {
	payload := map[string]any{
		"finding_id": issue.FindingID,
		"summary":    issue.Summary,
		"category":   issue.Category,
		"priority":   issue.Priority,
		"owner":      issue.Owner,
		"surface":    issue.Surface,
		"impact":     issue.Impact,
		"status":     archivedIssueStatus(issue.Status),
		"evidence":   issueArchiveStrings(issue.Evidence),
	}
	return json.Marshal(payload)
}

func (issue *ArchivedIssue) UnmarshalJSON(data []byte) error {
	type alias ArchivedIssue
	aux := struct {
		Status *string `json:"status"`
		*alias
	}{
		alias: (*alias)(issue),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Status == nil || strings.TrimSpace(*aux.Status) == "" {
		issue.Status = "open"
	} else {
		issue.Status = *aux.Status
	}
	if issue.Evidence == nil {
		issue.Evidence = []string{}
	}
	return nil
}

type IssuePriorityArchive struct {
	IssueID  string          `json:"issue_id"`
	Title    string          `json:"title"`
	Version  string          `json:"version"`
	Findings []ArchivedIssue `json:"findings,omitempty"`
}

func (archive IssuePriorityArchive) MarshalJSON() ([]byte, error) {
	payload := map[string]any{
		"issue_id": archive.IssueID,
		"title":    archive.Title,
		"version":  archive.Version,
		"findings": issueArchiveFindings(archive.Findings),
	}
	return json.Marshal(payload)
}

func (archive *IssuePriorityArchive) UnmarshalJSON(data []byte) error {
	type alias IssuePriorityArchive
	aux := struct{ *alias }{alias: (*alias)(archive)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if archive.Findings == nil {
		archive.Findings = []ArchivedIssue{}
	}
	return nil
}

type IssuePriorityArchiveAudit struct {
	Ready                bool           `json:"ready"`
	FindingCount         int            `json:"finding_count"`
	PriorityCounts       map[string]int `json:"priority_counts,omitempty"`
	CategoryCounts       map[string]int `json:"category_counts,omitempty"`
	MissingOwners        []string       `json:"missing_owners,omitempty"`
	InvalidPriorities    []string       `json:"invalid_priorities,omitempty"`
	InvalidCategories    []string       `json:"invalid_categories,omitempty"`
	UnresolvedP0Findings []string       `json:"unresolved_p0_findings,omitempty"`
}

func (audit IssuePriorityArchiveAudit) Summary() string {
	status := "HOLD"
	if audit.Ready {
		status = "READY"
	}
	return fmt.Sprintf(
		"%s: findings=%d missing_owners=%d invalid_priorities=%d invalid_categories=%d unresolved_p0=%d",
		status,
		audit.FindingCount,
		len(audit.MissingOwners),
		len(audit.InvalidPriorities),
		len(audit.InvalidCategories),
		len(audit.UnresolvedP0Findings),
	)
}

func (audit IssuePriorityArchiveAudit) MarshalJSON() ([]byte, error) {
	payload := map[string]any{
		"ready":                  audit.Ready,
		"finding_count":          audit.FindingCount,
		"priority_counts":        issueArchiveCounts(audit.PriorityCounts),
		"category_counts":        issueArchiveCounts(audit.CategoryCounts),
		"missing_owners":         issueArchiveStrings(audit.MissingOwners),
		"invalid_priorities":     issueArchiveStrings(audit.InvalidPriorities),
		"invalid_categories":     issueArchiveStrings(audit.InvalidCategories),
		"unresolved_p0_findings": issueArchiveStrings(audit.UnresolvedP0Findings),
	}
	return json.Marshal(payload)
}

func (audit *IssuePriorityArchiveAudit) UnmarshalJSON(data []byte) error {
	type alias IssuePriorityArchiveAudit
	aux := struct{ *alias }{alias: (*alias)(audit)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if audit.PriorityCounts == nil {
		audit.PriorityCounts = map[string]int{}
	}
	if audit.CategoryCounts == nil {
		audit.CategoryCounts = map[string]int{}
	}
	if audit.MissingOwners == nil {
		audit.MissingOwners = []string{}
	}
	if audit.InvalidPriorities == nil {
		audit.InvalidPriorities = []string{}
	}
	if audit.InvalidCategories == nil {
		audit.InvalidCategories = []string{}
	}
	if audit.UnresolvedP0Findings == nil {
		audit.UnresolvedP0Findings = []string{}
	}
	return nil
}

type IssuePriorityArchivist struct{}

func (IssuePriorityArchivist) Audit(archive IssuePriorityArchive) IssuePriorityArchiveAudit {
	priorityCounts := map[string]int{}
	for _, priority := range allowedIssuePriorities {
		priorityCounts[priority] = 0
	}
	categoryCounts := map[string]int{}
	for _, category := range allowedIssueCategories {
		categoryCounts[category] = 0
	}

	var missingOwners []string
	var invalidPriorities []string
	var invalidCategories []string
	var unresolvedP0 []string

	for _, finding := range archive.Findings {
		if strings.TrimSpace(finding.Owner) == "" {
			missingOwners = append(missingOwners, finding.FindingID)
		}
		if priority := finding.NormalizedPriority(); slices.Contains(allowedIssuePriorities, priority) {
			priorityCounts[priority]++
		} else {
			invalidPriorities = append(invalidPriorities, finding.FindingID)
		}
		if category := finding.NormalizedCategory(); slices.Contains(allowedIssueCategories, category) {
			categoryCounts[category]++
		} else {
			invalidCategories = append(invalidCategories, finding.FindingID)
		}
		if finding.NormalizedPriority() == "P0" && !finding.Resolved() {
			unresolvedP0 = append(unresolvedP0, finding.FindingID)
		}
	}

	slices.Sort(missingOwners)
	slices.Sort(invalidPriorities)
	slices.Sort(invalidCategories)
	slices.Sort(unresolvedP0)

	ready := len(archive.Findings) > 0 &&
		len(missingOwners) == 0 &&
		len(invalidPriorities) == 0 &&
		len(invalidCategories) == 0 &&
		len(unresolvedP0) == 0

	return IssuePriorityArchiveAudit{
		Ready:                ready,
		FindingCount:         len(archive.Findings),
		PriorityCounts:       priorityCounts,
		CategoryCounts:       categoryCounts,
		MissingOwners:        missingOwners,
		InvalidPriorities:    invalidPriorities,
		InvalidCategories:    invalidCategories,
		UnresolvedP0Findings: unresolvedP0,
	}
}

func RenderIssuePriorityArchiveReport(archive IssuePriorityArchive, audit IssuePriorityArchiveAudit) string {
	lines := []string{
		"# Issue Priority Archive",
		"",
		fmt.Sprintf("- Issue: %s %s", archive.IssueID, archive.Title),
		fmt.Sprintf("- Version: %s", archive.Version),
		fmt.Sprintf("- Audit: %s", audit.Summary()),
		fmt.Sprintf("- Priority Counts: P0=%d P1=%d P2=%d", audit.PriorityCounts["P0"], audit.PriorityCounts["P1"], audit.PriorityCounts["P2"]),
		fmt.Sprintf("- Category Counts: ui=%d ia=%d permission=%d metric=%d", audit.CategoryCounts["ui"], audit.CategoryCounts["ia"], audit.CategoryCounts["permission"], audit.CategoryCounts["metric"]),
		"",
		"## Findings",
	}
	for _, finding := range archive.Findings {
		lines = append(lines, fmt.Sprintf(
			"- %s: %s category=%s priority=%s owner=%s status=%s",
			finding.FindingID,
			finding.Summary,
			finding.NormalizedCategory(),
			finding.NormalizedPriority(),
			firstNonEmpty(strings.TrimSpace(finding.Owner), "none"),
			archivedIssueStatus(finding.Status),
		))
		lines = append(lines, fmt.Sprintf(
			"  surface=%s impact=%s evidence=%s",
			firstNonEmpty(strings.TrimSpace(finding.Surface), "none"),
			firstNonEmpty(strings.TrimSpace(finding.Impact), "none"),
			firstNonEmpty(strings.Join(finding.Evidence, ","), "none"),
		))
	}
	lines = append(lines,
		"",
		"## Audit Findings",
		fmt.Sprintf("- Missing owners: %s", firstNonEmpty(strings.Join(audit.MissingOwners, ", "), "none")),
		fmt.Sprintf("- Invalid priorities: %s", firstNonEmpty(strings.Join(audit.InvalidPriorities, ", "), "none")),
		fmt.Sprintf("- Invalid categories: %s", firstNonEmpty(strings.Join(audit.InvalidCategories, ", "), "none")),
		fmt.Sprintf("- Unresolved P0 findings: %s", firstNonEmpty(strings.Join(audit.UnresolvedP0Findings, ", "), "none")),
	)
	return strings.Join(lines, "\n")
}

func archivedIssueStatus(status string) string {
	trimmed := strings.TrimSpace(status)
	if trimmed == "" {
		return "open"
	}
	return trimmed
}

func issueArchiveStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func issueArchiveFindings(values []ArchivedIssue) []ArchivedIssue {
	if values == nil {
		return []ArchivedIssue{}
	}
	return values
}

func issueArchiveCounts(values map[string]int) map[string]int {
	if values == nil {
		return map[string]int{}
	}
	return values
}

func firstNonEmpty(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
