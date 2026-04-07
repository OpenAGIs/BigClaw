package governance

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestScopeFreezeBoardRoundTripPreservesManifestShape(t *testing.T) {
	board := ScopeFreezeBoard{
		Name:        "BigClaw v4.0 Freeze",
		Version:     "v4.0",
		FreezeDate:  "2026-03-11",
		FreezeOwner: "pm-director",
		BacklogItems: []GovernanceBacklogItem{
			{
				IssueID:            "OPE-116",
				Title:              "Scope freeze and task governance",
				Phase:              "step-1",
				Owner:              "pm-director",
				Status:             "ready",
				ScopeStatus:        "frozen",
				AcceptanceCriteria: []string{"Epic scope frozen", "Backlog sequencing documented"},
				ValidationPlan:     []string{"governance-audit", "report-shared"},
				RequiredCloseout:   append([]string(nil), requiredRunCloseouts...),
			},
		},
		Exceptions: []FreezeException{
			{
				IssueID:      "OPE-121",
				Reason:       "Compliance requirement discovered after freeze",
				ApprovedBy:   "cto",
				DecisionNote: "Allow as approved exception.",
			},
		},
	}
	payload, err := json.Marshal(board)
	if err != nil {
		t.Fatalf("marshal board: %v", err)
	}
	var restored ScopeFreezeBoard
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal board: %v", err)
	}
	if !reflect.DeepEqual(restored, board) {
		t.Fatalf("restored board mismatch: got %+v want %+v", restored, board)
	}
}

func TestScopeFreezeAuditFlagsBacklogGovernanceAndCloseoutGaps(t *testing.T) {
	board := ScopeFreezeBoard{
		Name:        "BigClaw v4.0 Freeze",
		Version:     "v4.0",
		FreezeDate:  "2026-03-11",
		FreezeOwner: "pm-director",
		BacklogItems: []GovernanceBacklogItem{
			{
				IssueID:          "OPE-116",
				Title:            "Scope freeze and task governance",
				Phase:            "step-1",
				ScopeStatus:      "proposed",
				RequiredCloseout: []string{"validation-evidence"},
			},
			{
				IssueID:            "OPE-118",
				Title:              "Duplicate issue record",
				Phase:              "step-1",
				Owner:              "pm-director",
				ScopeStatus:        "unknown",
				AcceptanceCriteria: []string{"Keep backlog aligned"},
				ValidationPlan:     []string{"audit"},
				RequiredCloseout:   []string{"validation-evidence", "git-push"},
			},
			{
				IssueID:            "OPE-116",
				Title:              "Mirrored tracking entry",
				Phase:              "step-1",
				Owner:              "pm-director",
				ScopeStatus:        "frozen",
				AcceptanceCriteria: []string{"Track freeze decisions"},
				ValidationPlan:     []string{"audit"},
				RequiredCloseout:   append([]string(nil), requiredRunCloseouts...),
			},
		},
		Exceptions: []FreezeException{{IssueID: "OPE-130", Reason: "Pending steering review"}},
	}
	audit := ScopeFreezeGovernance{}.Audit(board)
	if !reflect.DeepEqual(audit.DuplicateIssueIDs, []string{"OPE-116"}) {
		t.Fatalf("unexpected duplicate ids: %+v", audit)
	}
	if !reflect.DeepEqual(audit.MissingOwners, []string{"OPE-116"}) {
		t.Fatalf("unexpected missing owners: %+v", audit)
	}
	if !reflect.DeepEqual(audit.MissingAcceptance, []string{"OPE-116"}) {
		t.Fatalf("unexpected missing acceptance: %+v", audit)
	}
	if !reflect.DeepEqual(audit.MissingValidation, []string{"OPE-116"}) {
		t.Fatalf("unexpected missing validation: %+v", audit)
	}
	wantCloseout := map[string][]string{
		"OPE-116": {"git-push", "git-log-stat"},
		"OPE-118": {"git-log-stat"},
	}
	if !reflect.DeepEqual(audit.MissingCloseoutRequirements, wantCloseout) {
		t.Fatalf("unexpected missing closeout requirements: %+v", audit.MissingCloseoutRequirements)
	}
	if !reflect.DeepEqual(audit.UnauthorizedScopeChanges, []string{"OPE-116"}) {
		t.Fatalf("unexpected unauthorized scope changes: %+v", audit)
	}
	if !reflect.DeepEqual(audit.InvalidScopeStatuses, []string{"OPE-118"}) {
		t.Fatalf("unexpected invalid scope statuses: %+v", audit)
	}
	if !reflect.DeepEqual(audit.UnapprovedExceptions, []string{"OPE-130"}) {
		t.Fatalf("unexpected unapproved exceptions: %+v", audit)
	}
	if audit.ReleaseReady() {
		t.Fatalf("expected release not ready")
	}
	if audit.ReadinessScore() != 0 {
		t.Fatalf("expected readiness score 0, got %.1f", audit.ReadinessScore())
	}
}

func TestScopeFreezeAuditRoundTripAndReadyState(t *testing.T) {
	audit := ScopeFreezeAudit{
		BoardName:  "BigClaw v4.0 Freeze",
		Version:    "v4.0",
		TotalItems: 1,
	}
	payload, err := json.Marshal(audit)
	if err != nil {
		t.Fatalf("marshal audit: %v", err)
	}
	var restored ScopeFreezeAudit
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal audit: %v", err)
	}
	if !reflect.DeepEqual(restored, audit) {
		t.Fatalf("restored audit mismatch: got %+v want %+v", restored, audit)
	}
	if !restored.ReleaseReady() || restored.ReadinessScore() != 100 {
		t.Fatalf("expected ready audit, got %+v", restored)
	}
}

func TestRenderScopeFreezeReportSummarizesBoardAndRunCloseoutRequirements(t *testing.T) {
	board := ScopeFreezeBoard{
		Name:        "BigClaw v4.0 Freeze",
		Version:     "v4.0",
		FreezeDate:  "2026-03-11",
		FreezeOwner: "pm-director",
		BacklogItems: []GovernanceBacklogItem{
			{
				IssueID:            "OPE-116",
				Title:              "Scope freeze and task governance",
				Phase:              "step-1",
				Owner:              "pm-director",
				Status:             "ready",
				ScopeStatus:        "frozen",
				AcceptanceCriteria: []string{"Epic scope frozen"},
				ValidationPlan:     []string{"governance-audit"},
				RequiredCloseout:   append([]string(nil), requiredRunCloseouts...),
			},
		},
		Exceptions: []FreezeException{{IssueID: "OPE-121", Reason: "Approved scope exception", ApprovedBy: "cto"}},
	}
	report := RenderScopeFreezeReport(board, ScopeFreezeGovernance{}.Audit(board))
	for _, want := range []string{
		"# Scope Freeze Governance Report",
		"- Readiness Score: 100.0",
		"- Release Ready: true",
		"- OPE-116: phase=step-1 owner=pm-director status=ready scope=frozen closeout=validation-evidence, git-push, git-log-stat",
		"- OPE-121: approved_by=cto reason=Approved scope exception",
		"- Missing closeout requirements: none",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected report to contain %q, got %s", want, report)
		}
	}
}
