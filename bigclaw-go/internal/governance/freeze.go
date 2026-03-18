package governance

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

var requiredRunCloseouts = []string{"validation-evidence", "git-push", "git-log-stat"}

var allowedScopeStatuses = map[string]struct{}{
	"frozen":             {},
	"approved-exception": {},
	"proposed":           {},
}

type FreezeException struct {
	IssueID      string `json:"issue_id"`
	Reason       string `json:"reason"`
	ApprovedBy   string `json:"approved_by,omitempty"`
	DecisionNote string `json:"decision_note,omitempty"`
}

func (e FreezeException) Approved() bool {
	return strings.TrimSpace(e.ApprovedBy) != ""
}

type GovernanceBacklogItem struct {
	IssueID            string   `json:"issue_id"`
	Title              string   `json:"title"`
	Phase              string   `json:"phase"`
	Owner              string   `json:"owner,omitempty"`
	Status             string   `json:"status"`
	ScopeStatus        string   `json:"scope_status"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	ValidationPlan     []string `json:"validation_plan,omitempty"`
	RequiredCloseout   []string `json:"required_closeout,omitempty"`
	LinkedEpics        []string `json:"linked_epics,omitempty"`
	Notes              string   `json:"notes,omitempty"`
}

func (item *GovernanceBacklogItem) UnmarshalJSON(data []byte) error {
	type rawGovernanceBacklogItem struct {
		IssueID            string    `json:"issue_id"`
		Title              string    `json:"title"`
		Phase              string    `json:"phase"`
		Owner              string    `json:"owner,omitempty"`
		Status             *string   `json:"status"`
		ScopeStatus        *string   `json:"scope_status"`
		AcceptanceCriteria []string  `json:"acceptance_criteria,omitempty"`
		ValidationPlan     []string  `json:"validation_plan,omitempty"`
		RequiredCloseout   *[]string `json:"required_closeout,omitempty"`
		LinkedEpics        []string  `json:"linked_epics,omitempty"`
		Notes              string    `json:"notes,omitempty"`
	}
	var raw rawGovernanceBacklogItem
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	item.IssueID = raw.IssueID
	item.Title = raw.Title
	item.Phase = raw.Phase
	item.Owner = raw.Owner
	item.Status = "planned"
	if raw.Status != nil {
		item.Status = *raw.Status
	}
	item.ScopeStatus = "frozen"
	if raw.ScopeStatus != nil {
		item.ScopeStatus = *raw.ScopeStatus
	}
	item.AcceptanceCriteria = append([]string(nil), raw.AcceptanceCriteria...)
	item.ValidationPlan = append([]string(nil), raw.ValidationPlan...)
	item.RequiredCloseout = append([]string(nil), requiredRunCloseouts...)
	if raw.RequiredCloseout != nil {
		item.RequiredCloseout = append([]string(nil), (*raw.RequiredCloseout)...)
	}
	item.LinkedEpics = append([]string(nil), raw.LinkedEpics...)
	item.Notes = raw.Notes
	return nil
}

func (item GovernanceBacklogItem) MissingCloseoutRequirements() []string {
	present := make(map[string]struct{}, len(item.RequiredCloseout))
	for _, closeout := range item.RequiredCloseout {
		if trimmed := strings.ToLower(strings.TrimSpace(closeout)); trimmed != "" {
			present[trimmed] = struct{}{}
		}
	}
	missing := make([]string, 0)
	for _, requirement := range requiredRunCloseouts {
		if _, ok := present[requirement]; !ok {
			missing = append(missing, requirement)
		}
	}
	return missing
}

func (item GovernanceBacklogItem) GovernanceReady() bool {
	_, allowed := allowedScopeStatuses[item.ScopeStatus]
	return strings.TrimSpace(item.Owner) != "" &&
		allowed &&
		len(item.AcceptanceCriteria) > 0 &&
		len(item.ValidationPlan) > 0 &&
		len(item.MissingCloseoutRequirements()) == 0
}

type ScopeFreezeBoard struct {
	Name         string                  `json:"name"`
	Version      string                  `json:"version"`
	FreezeDate   string                  `json:"freeze_date,omitempty"`
	FreezeOwner  string                  `json:"freeze_owner,omitempty"`
	BacklogItems []GovernanceBacklogItem `json:"backlog_items,omitempty"`
	Exceptions   []FreezeException       `json:"exceptions,omitempty"`
}

type ScopeFreezeAudit struct {
	BoardName                   string              `json:"board_name"`
	Version                     string              `json:"version"`
	TotalItems                  int                 `json:"total_items"`
	DuplicateIssueIDs           []string            `json:"duplicate_issue_ids,omitempty"`
	MissingOwners               []string            `json:"missing_owners,omitempty"`
	MissingAcceptance           []string            `json:"missing_acceptance,omitempty"`
	MissingValidation           []string            `json:"missing_validation,omitempty"`
	MissingCloseoutRequirements map[string][]string `json:"missing_closeout_requirements,omitempty"`
	UnauthorizedScopeChanges    []string            `json:"unauthorized_scope_changes,omitempty"`
	InvalidScopeStatuses        []string            `json:"invalid_scope_statuses,omitempty"`
	UnapprovedExceptions        []string            `json:"unapproved_exceptions,omitempty"`
}

func (audit ScopeFreezeAudit) ReleaseReady() bool {
	return len(audit.DuplicateIssueIDs) == 0 &&
		len(audit.MissingOwners) == 0 &&
		len(audit.MissingAcceptance) == 0 &&
		len(audit.MissingValidation) == 0 &&
		len(audit.MissingCloseoutRequirements) == 0 &&
		len(audit.UnauthorizedScopeChanges) == 0 &&
		len(audit.InvalidScopeStatuses) == 0 &&
		len(audit.UnapprovedExceptions) == 0
}

func (audit ScopeFreezeAudit) ReadinessScore() float64 {
	checks := []bool{
		len(audit.DuplicateIssueIDs) == 0,
		len(audit.MissingOwners) == 0,
		len(audit.MissingAcceptance) == 0,
		len(audit.MissingValidation) == 0,
		len(audit.MissingCloseoutRequirements) == 0,
		len(audit.UnauthorizedScopeChanges) == 0,
		len(audit.InvalidScopeStatuses) == 0,
		len(audit.UnapprovedExceptions) == 0,
	}
	passed := 0
	for _, check := range checks {
		if check {
			passed++
		}
	}
	return round1(float64(passed) / float64(len(checks)) * 100)
}

type ScopeFreezeGovernance struct{}

func (ScopeFreezeGovernance) Audit(board ScopeFreezeBoard) ScopeFreezeAudit {
	counts := make(map[string]int)
	exceptionIndex := make(map[string]FreezeException, len(board.Exceptions))
	for _, exception := range board.Exceptions {
		exceptionIndex[exception.IssueID] = exception
	}
	for _, item := range board.BacklogItems {
		counts[item.IssueID]++
	}

	duplicateIssueIDs := make([]string, 0)
	for issueID, count := range counts {
		if count > 1 {
			duplicateIssueIDs = append(duplicateIssueIDs, issueID)
		}
	}
	sort.Strings(duplicateIssueIDs)

	missingOwners := make([]string, 0)
	missingAcceptance := make([]string, 0)
	missingValidation := make([]string, 0)
	missingCloseoutRequirements := make(map[string][]string)
	invalidScopeStatuses := make([]string, 0)
	unauthorizedScopeChanges := make([]string, 0)
	for _, item := range board.BacklogItems {
		if strings.TrimSpace(item.Owner) == "" {
			missingOwners = append(missingOwners, item.IssueID)
		}
		if len(item.AcceptanceCriteria) == 0 {
			missingAcceptance = append(missingAcceptance, item.IssueID)
		}
		if len(item.ValidationPlan) == 0 {
			missingValidation = append(missingValidation, item.IssueID)
		}
		if missing := item.MissingCloseoutRequirements(); len(missing) > 0 {
			missingCloseoutRequirements[item.IssueID] = missing
		}
		if _, ok := allowedScopeStatuses[item.ScopeStatus]; !ok {
			invalidScopeStatuses = append(invalidScopeStatuses, item.IssueID)
		}
		if item.ScopeStatus == "proposed" {
			exception, ok := exceptionIndex[item.IssueID]
			if !ok || !exception.Approved() {
				unauthorizedScopeChanges = append(unauthorizedScopeChanges, item.IssueID)
			}
		}
	}

	unapprovedExceptions := make([]string, 0)
	for _, exception := range board.Exceptions {
		if !exception.Approved() {
			unapprovedExceptions = append(unapprovedExceptions, exception.IssueID)
		}
	}

	sort.Strings(missingOwners)
	sort.Strings(missingAcceptance)
	sort.Strings(missingValidation)
	sort.Strings(invalidScopeStatuses)
	sort.Strings(unauthorizedScopeChanges)
	sort.Strings(unapprovedExceptions)
	if len(missingCloseoutRequirements) == 0 {
		missingCloseoutRequirements = nil
	}
	return ScopeFreezeAudit{
		BoardName:                   board.Name,
		Version:                     board.Version,
		TotalItems:                  len(board.BacklogItems),
		DuplicateIssueIDs:           duplicateIssueIDs,
		MissingOwners:               missingOwners,
		MissingAcceptance:           missingAcceptance,
		MissingValidation:           missingValidation,
		MissingCloseoutRequirements: missingCloseoutRequirements,
		UnauthorizedScopeChanges:    unauthorizedScopeChanges,
		InvalidScopeStatuses:        invalidScopeStatuses,
		UnapprovedExceptions:        unapprovedExceptions,
	}
}

func RenderScopeFreezeReport(board ScopeFreezeBoard, audit ScopeFreezeAudit) string {
	lines := []string{
		"# Scope Freeze Governance Report",
		"",
		fmt.Sprintf("- Name: %s", board.Name),
		fmt.Sprintf("- Version: %s", board.Version),
		fmt.Sprintf("- Freeze Date: %s", board.FreezeDate),
		fmt.Sprintf("- Freeze Owner: %s", board.FreezeOwner),
		fmt.Sprintf("- Backlog Items: %d", len(board.BacklogItems)),
		fmt.Sprintf("- Exceptions: %d", len(board.Exceptions)),
		fmt.Sprintf("- Readiness Score: %.1f", audit.ReadinessScore()),
		fmt.Sprintf("- Release Ready: %t", audit.ReleaseReady()),
		"",
		"## Backlog",
		"",
	}
	if len(board.BacklogItems) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, item := range board.BacklogItems {
			closeout := strings.Join(item.RequiredCloseout, ", ")
			if closeout == "" {
				closeout = "none"
			}
			lines = append(lines, fmt.Sprintf(
				"- %s: phase=%s owner=%s status=%s scope=%s closeout=%s",
				item.IssueID,
				item.Phase,
				firstNonEmpty(item.Owner, "none"),
				item.Status,
				item.ScopeStatus,
				closeout,
			))
		}
	}
	lines = append(lines, "", "## Freeze Exceptions", "")
	if len(board.Exceptions) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, exception := range board.Exceptions {
			lines = append(lines, fmt.Sprintf(
				"- %s: approved_by=%s reason=%s",
				exception.IssueID,
				firstNonEmpty(exception.ApprovedBy, "pending"),
				firstNonEmpty(exception.Reason, "none"),
			))
		}
	}
	lines = append(lines, "", "## Audit", "")
	lines = append(lines, fmt.Sprintf("- Duplicate issues: %s", joinOrNone(audit.DuplicateIssueIDs)))
	lines = append(lines, fmt.Sprintf("- Missing owners: %s", joinOrNone(audit.MissingOwners)))
	lines = append(lines, fmt.Sprintf("- Missing acceptance: %s", joinOrNone(audit.MissingAcceptance)))
	lines = append(lines, fmt.Sprintf("- Missing validation: %s", joinOrNone(audit.MissingValidation)))
	missingCloseout := "none"
	if len(audit.MissingCloseoutRequirements) > 0 {
		issueIDs := make([]string, 0, len(audit.MissingCloseoutRequirements))
		for issueID := range audit.MissingCloseoutRequirements {
			issueIDs = append(issueIDs, issueID)
		}
		sort.Strings(issueIDs)
		parts := make([]string, 0, len(issueIDs))
		for _, issueID := range issueIDs {
			parts = append(parts, fmt.Sprintf("%s=%s", issueID, strings.Join(audit.MissingCloseoutRequirements[issueID], ", ")))
		}
		missingCloseout = strings.Join(parts, "; ")
	}
	lines = append(lines, fmt.Sprintf("- Missing closeout requirements: %s", missingCloseout))
	lines = append(lines, fmt.Sprintf("- Unauthorized scope changes: %s", joinOrNone(audit.UnauthorizedScopeChanges)))
	lines = append(lines, fmt.Sprintf("- Invalid scope statuses: %s", joinOrNone(audit.InvalidScopeStatuses)))
	lines = append(lines, fmt.Sprintf("- Unapproved exceptions: %s", joinOrNone(audit.UnapprovedExceptions)))
	return strings.Join(lines, "\n") + "\n"
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func firstNonEmpty(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func round1(value float64) float64 {
	return float64(int(value*10+0.5)) / 10
}
