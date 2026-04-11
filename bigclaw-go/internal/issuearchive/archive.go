package issuearchive

import (
	"fmt"
	"sort"
	"strings"
)

var (
	allowedIssueCategories = []string{"ia", "metric", "permission", "ui"}
	allowedIssuePriorities = []string{"P0", "P1", "P2"}
)

type ArchivedIssue struct {
	FindingID string
	Summary   string
	Category  string
	Priority  string
	Owner     string
	Surface   string
	Impact    string
	Status    string
	Evidence  []string
}

func (i ArchivedIssue) NormalizedCategory() string {
	return strings.ToLower(strings.TrimSpace(i.Category))
}

func (i ArchivedIssue) NormalizedPriority() string {
	return strings.ToUpper(strings.TrimSpace(i.Priority))
}

func (i ArchivedIssue) Resolved() bool {
	return strings.EqualFold(strings.TrimSpace(i.Status), "resolved")
}

func (i ArchivedIssue) ToMap() map[string]any {
	return map[string]any{
		"finding_id": i.FindingID,
		"summary":    i.Summary,
		"category":   i.Category,
		"priority":   i.Priority,
		"owner":      i.Owner,
		"surface":    i.Surface,
		"impact":     i.Impact,
		"status":     firstNonEmpty(i.Status, "open"),
		"evidence":   stringsToAny(i.Evidence),
	}
}

func ArchivedIssueFromMap(data map[string]any) ArchivedIssue {
	return ArchivedIssue{
		FindingID: stringValue(data["finding_id"]),
		Summary:   stringValue(data["summary"]),
		Category:  stringValue(data["category"]),
		Priority:  stringValue(data["priority"]),
		Owner:     stringValueWithDefault(data["owner"], ""),
		Surface:   stringValueWithDefault(data["surface"], ""),
		Impact:    stringValueWithDefault(data["impact"], ""),
		Status:    stringValueWithDefault(data["status"], "open"),
		Evidence:  stringSliceValue(data["evidence"]),
	}
}

type IssuePriorityArchive struct {
	IssueID  string
	Title    string
	Version  string
	Findings []ArchivedIssue
}

func (a IssuePriorityArchive) ToMap() map[string]any {
	findings := make([]any, 0, len(a.Findings))
	for _, finding := range a.Findings {
		findings = append(findings, finding.ToMap())
	}
	return map[string]any{
		"issue_id": a.IssueID,
		"title":    a.Title,
		"version":  a.Version,
		"findings": findings,
	}
}

func IssuePriorityArchiveFromMap(data map[string]any) IssuePriorityArchive {
	findings := make([]ArchivedIssue, 0)
	for _, item := range anySliceValue(data["findings"]) {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		findings = append(findings, ArchivedIssueFromMap(entry))
	}
	return IssuePriorityArchive{
		IssueID:  stringValue(data["issue_id"]),
		Title:    stringValue(data["title"]),
		Version:  stringValue(data["version"]),
		Findings: findings,
	}
}

type IssuePriorityArchiveAudit struct {
	Ready                bool
	FindingCount         int
	PriorityCounts       map[string]int
	CategoryCounts       map[string]int
	MissingOwners        []string
	InvalidPriorities    []string
	InvalidCategories    []string
	UnresolvedP0Findings []string
}

func (a IssuePriorityArchiveAudit) Summary() string {
	status := "HOLD"
	if a.Ready {
		status = "READY"
	}
	return fmt.Sprintf(
		"%s: findings=%d missing_owners=%d invalid_priorities=%d invalid_categories=%d unresolved_p0=%d",
		status,
		a.FindingCount,
		len(a.MissingOwners),
		len(a.InvalidPriorities),
		len(a.InvalidCategories),
		len(a.UnresolvedP0Findings),
	)
}

func (a IssuePriorityArchiveAudit) ToMap() map[string]any {
	return map[string]any{
		"ready":                  a.Ready,
		"finding_count":          a.FindingCount,
		"priority_counts":        intMapToAny(a.PriorityCounts),
		"category_counts":        intMapToAny(a.CategoryCounts),
		"missing_owners":         stringsToAny(a.MissingOwners),
		"invalid_priorities":     stringsToAny(a.InvalidPriorities),
		"invalid_categories":     stringsToAny(a.InvalidCategories),
		"unresolved_p0_findings": stringsToAny(a.UnresolvedP0Findings),
	}
}

func IssuePriorityArchiveAuditFromMap(data map[string]any) IssuePriorityArchiveAudit {
	return IssuePriorityArchiveAudit{
		Ready:                boolValue(data["ready"]),
		FindingCount:         intValue(data["finding_count"]),
		PriorityCounts:       intMapValue(data["priority_counts"]),
		CategoryCounts:       intMapValue(data["category_counts"]),
		MissingOwners:        stringSliceValue(data["missing_owners"]),
		InvalidPriorities:    stringSliceValue(data["invalid_priorities"]),
		InvalidCategories:    stringSliceValue(data["invalid_categories"]),
		UnresolvedP0Findings: stringSliceValue(data["unresolved_p0_findings"]),
	}
}

type Archivist struct{}

func (Archivist) Audit(archive IssuePriorityArchive) IssuePriorityArchiveAudit {
	priorityCounts := make(map[string]int, len(allowedIssuePriorities))
	for _, priority := range allowedIssuePriorities {
		priorityCounts[priority] = 0
	}
	categoryCounts := make(map[string]int, len(allowedIssueCategories))
	for _, category := range allowedIssueCategories {
		categoryCounts[category] = 0
	}

	var missingOwners []string
	var invalidPriorities []string
	var invalidCategories []string
	var unresolvedP0Findings []string

	for _, finding := range archive.Findings {
		if strings.TrimSpace(finding.Owner) == "" {
			missingOwners = append(missingOwners, finding.FindingID)
		}
		if _, ok := priorityCounts[finding.NormalizedPriority()]; ok {
			priorityCounts[finding.NormalizedPriority()]++
		} else {
			invalidPriorities = append(invalidPriorities, finding.FindingID)
		}
		if _, ok := categoryCounts[finding.NormalizedCategory()]; ok {
			categoryCounts[finding.NormalizedCategory()]++
		} else {
			invalidCategories = append(invalidCategories, finding.FindingID)
		}
		if finding.NormalizedPriority() == "P0" && !finding.Resolved() {
			unresolvedP0Findings = append(unresolvedP0Findings, finding.FindingID)
		}
	}

	sort.Strings(missingOwners)
	sort.Strings(invalidPriorities)
	sort.Strings(invalidCategories)
	sort.Strings(unresolvedP0Findings)

	return IssuePriorityArchiveAudit{
		Ready:                len(archive.Findings) > 0 && len(missingOwners) == 0 && len(invalidPriorities) == 0 && len(invalidCategories) == 0 && len(unresolvedP0Findings) == 0,
		FindingCount:         len(archive.Findings),
		PriorityCounts:       priorityCounts,
		CategoryCounts:       categoryCounts,
		MissingOwners:        missingOwners,
		InvalidPriorities:    invalidPriorities,
		InvalidCategories:    invalidCategories,
		UnresolvedP0Findings: unresolvedP0Findings,
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
		lines = append(lines, fmt.Sprintf("- %s: %s category=%s priority=%s owner=%s status=%s", finding.FindingID, finding.Summary, finding.NormalizedCategory(), finding.NormalizedPriority(), firstNonEmpty(strings.TrimSpace(finding.Owner), "none"), firstNonEmpty(strings.TrimSpace(finding.Status), "open")))
		lines = append(lines, fmt.Sprintf("  surface=%s impact=%s evidence=%s", firstNonEmpty(strings.TrimSpace(finding.Surface), "none"), firstNonEmpty(strings.TrimSpace(finding.Impact), "none"), firstNonEmpty(strings.Join(finding.Evidence, ","), "none")))
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

func stringsToAny(values []string) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}

func intMapToAny(values map[string]int) map[string]any {
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func stringValue(value any) string {
	return fmt.Sprint(value)
}

func stringValueWithDefault(value any, fallback string) string {
	if value == nil {
		return fallback
	}
	return fmt.Sprint(value)
}

func stringSliceValue(value any) []string {
	items := anySliceValue(value)
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, fmt.Sprint(item))
	}
	return out
}

func anySliceValue(value any) []any {
	switch typed := value.(type) {
	case nil:
		return []any{}
	case []any:
		return typed
	case []string:
		return stringsToAny(typed)
	default:
		return []any{}
	}
}

func intMapValue(value any) map[string]int {
	out := map[string]int{}
	typed, ok := value.(map[string]any)
	if !ok {
		return out
	}
	for key, raw := range typed {
		out[key] = intValue(raw)
	}
	return out
}

func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func boolValue(value any) bool {
	typed, ok := value.(bool)
	return ok && typed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
