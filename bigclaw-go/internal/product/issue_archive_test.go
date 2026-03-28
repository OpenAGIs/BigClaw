package product

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestIssuePriorityArchiveRoundTripPreservesManifestShape(t *testing.T) {
	archive := IssuePriorityArchive{
		IssueID: "OPE-137",
		Title:   "BIG-4403 问题清单与优先级归档",
		Version: "v4.0-review-1",
		Findings: []ArchivedIssue{{
			FindingID: "BIG-4403-1",
			Summary:   "Global nav alert badge lacks escalation color contrast.",
			Category:  "ui",
			Priority:  "P1",
			Owner:     "product-experience",
			Surface:   "console-header",
			Impact:    "Operators can miss pending incident alerts.",
			Status:    "open",
			Evidence:  []string{"wireframe-review", "contrast-audit"},
		}},
	}

	payload, err := json.Marshal(archive)
	if err != nil {
		t.Fatalf("marshal archive: %v", err)
	}
	var restored IssuePriorityArchive
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal archive: %v", err)
	}
	if !reflect.DeepEqual(restored, archive) {
		t.Fatalf("archive mismatch: restored=%+v want=%+v", restored, archive)
	}
}

func TestIssuePriorityArchiveAuditFlagsOwnerPriorityCategoryAndOpenP0Gaps(t *testing.T) {
	archive := IssuePriorityArchive{
		IssueID: "OPE-137",
		Title:   "BIG-4403 问题清单与优先级归档",
		Version: "v4.0-review-1",
		Findings: []ArchivedIssue{
			{FindingID: "BIG-4403-1", Summary: "Primary queue screen omits inline permission guidance.", Category: "permission", Priority: "P0", Owner: "", Status: "open"},
			{FindingID: "BIG-4403-2", Summary: "Information architecture duplicates run history entry points.", Category: "ia", Priority: "P1", Owner: "product-experience", Status: "open"},
			{FindingID: "BIG-4403-3", Summary: "Latency metric label mismatches backend contract.", Category: "metrics", Priority: "P3", Owner: "engineering-operations", Status: "resolved"},
		},
	}

	audit := IssuePriorityArchivist{}.Audit(archive)

	if audit.Ready {
		t.Fatalf("expected hold audit, got %+v", audit)
	}
	if audit.FindingCount != 3 {
		t.Fatalf("expected 3 findings, got %+v", audit)
	}
	if !reflect.DeepEqual(audit.PriorityCounts, map[string]int{"P0": 1, "P1": 1, "P2": 0}) {
		t.Fatalf("unexpected priority counts: %+v", audit.PriorityCounts)
	}
	if !reflect.DeepEqual(audit.CategoryCounts, map[string]int{"ia": 1, "metric": 0, "permission": 1, "ui": 0}) {
		t.Fatalf("unexpected category counts: %+v", audit.CategoryCounts)
	}
	if !reflect.DeepEqual(audit.MissingOwners, []string{"BIG-4403-1"}) ||
		!reflect.DeepEqual(audit.InvalidPriorities, []string{"BIG-4403-3"}) ||
		!reflect.DeepEqual(audit.InvalidCategories, []string{"BIG-4403-3"}) ||
		!reflect.DeepEqual(audit.UnresolvedP0Findings, []string{"BIG-4403-1"}) {
		t.Fatalf("unexpected audit findings: %+v", audit)
	}
}

func TestIssuePriorityArchiveAuditRoundTripAndReadyState(t *testing.T) {
	audit := IssuePriorityArchiveAudit{
		Ready:                true,
		FindingCount:         2,
		PriorityCounts:       map[string]int{"P0": 0, "P1": 1, "P2": 1},
		CategoryCounts:       map[string]int{"ui": 1, "ia": 1, "permission": 0, "metric": 0},
		MissingOwners:        []string{},
		InvalidPriorities:    []string{},
		InvalidCategories:    []string{},
		UnresolvedP0Findings: []string{},
	}

	payload, err := json.Marshal(audit)
	if err != nil {
		t.Fatalf("marshal audit: %v", err)
	}
	var restored IssuePriorityArchiveAudit
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal audit: %v", err)
	}
	if !reflect.DeepEqual(restored, audit) {
		t.Fatalf("audit mismatch: restored=%+v want=%+v", restored, audit)
	}
	if restored.Summary() != "READY: findings=2 missing_owners=0 invalid_priorities=0 invalid_categories=0 unresolved_p0=0" {
		t.Fatalf("unexpected summary: %s", restored.Summary())
	}
}

func TestRenderIssuePriorityArchiveReportSummarizesFindingsAndRollups(t *testing.T) {
	archive := IssuePriorityArchive{
		IssueID: "OPE-137",
		Title:   "BIG-4403 问题清单与优先级归档",
		Version: "v4.0-review-1",
		Findings: []ArchivedIssue{
			{FindingID: "BIG-4403-1", Summary: "Queue dashboard card spacing breaks scannability under dense data.", Category: "ui", Priority: "P1", Owner: "product-experience", Surface: "operations-dashboard", Impact: "Dense incident queues become harder to triage.", Status: "resolved", Evidence: []string{"wireframe-pack"}},
			{FindingID: "BIG-4403-2", Summary: "Run detail permission matrix is not exposed in empty states.", Category: "permission", Priority: "P2", Owner: "security-platform", Surface: "run-detail", Impact: "Reviewers cannot confirm role gating behavior.", Status: "resolved", Evidence: []string{"review-notes", "permission-checklist"}},
		},
	}

	audit := IssuePriorityArchivist{}.Audit(archive)
	report := RenderIssuePriorityArchiveReport(archive, audit)

	for _, fragment := range []string{
		"# Issue Priority Archive",
		"- Audit: READY: findings=2 missing_owners=0 invalid_priorities=0 invalid_categories=0 unresolved_p0=0",
		"- Priority Counts: P0=0 P1=1 P2=1",
		"- Category Counts: ui=1 ia=0 permission=1 metric=0",
		"- BIG-4403-1: Queue dashboard card spacing breaks scannability under dense data. category=ui priority=P1 owner=product-experience status=resolved",
		"- Unresolved P0 findings: none",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}
