package uireview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUIReviewPackRoundTripPreservesManifestShape(t *testing.T) {
	pack := UIReviewPack{
		IssueID: "BIG-4204",
		Title:   "UI评审包输出",
		Version: "v4.0-review-pack",
		Objectives: []ReviewObjective{{
			ObjectiveID:   "obj-alignment",
			Title:         "Align reviewers on the release-control story",
			Persona:       "product-experience",
			Outcome:       "Reviewers see the scope, stakes, and success criteria before page-level critique.",
			SuccessSignal: "The kickoff frame is sufficient to decide whether the slice is review-ready.",
			Priority:      "P0",
			Dependencies:  []string{"BIG-1103", "BIG-1701"},
		}},
		Wireframes: []WireframeSurface{{
			SurfaceID:    "wf-overview",
			Name:         "Review overview board",
			Device:       "desktop",
			EntryPoint:   "Epic 11 review hub",
			PrimaryBlocks: []string{"header", "objective strip", "wireframe rail", "decision log"},
			ReviewNotes:  []string{"Highlight unresolved dependencies before approval."},
		}},
		Interactions: []InteractionFlow{{
			FlowID:         "flow-frame-switch",
			Name:           "Switch between wireframes and interaction notes",
			Trigger:        "Reviewer selects a surface from the wireframe rail",
			SystemResponse: "The board swaps the focus frame and preserves reviewer comments.",
			States:         []string{"default", "focus", "with-comments"},
			Exceptions:     []string{"Warn when leaving a frame with unsaved notes."},
		}},
		OpenQuestions: []OpenQuestion{{
			QuestionID: "oq-mobile-depth",
			Theme:      "scope",
			Question:   "Should the first review pack cover mobile wireframes or desktop only?",
			Owner:      "product-experience",
			Impact:     "Changes review breadth and the number of required surfaces.",
		}},
	}

	restored := pack.ensureDefaults()
	if restored.Objectives[0].ObjectiveID != pack.Objectives[0].ObjectiveID {
		t.Fatalf("round trip drifted: got=%+v want=%+v", restored, pack)
	}
}

func TestUIReviewPackAuditFlagsMissingSectionsAndCoverageGaps(t *testing.T) {
	pack := UIReviewPack{
		IssueID: "BIG-4204",
		Title:   "UI评审包输出",
		Version: "v4.0-review-pack",
		Objectives: []ReviewObjective{{
			ObjectiveID:   "obj-incomplete",
			Title:         "Incomplete objective",
			Persona:       "product-experience",
			Outcome:       "Create a frame for review.",
			SuccessSignal: "",
		}},
		Wireframes: []WireframeSurface{{
			SurfaceID:  "wf-empty",
			Name:       "Empty frame",
			Device:     "desktop",
			EntryPoint: "Review hub",
		}},
		Interactions: []InteractionFlow{{
			FlowID:         "flow-empty",
			Name:           "Unspecified interaction",
			Trigger:        "Reviewer opens the page",
			SystemResponse: "The system loads the frame.",
		}},
	}

	audit := Auditor{}.Audit(pack)
	if audit.Ready {
		t.Fatal("expected audit to hold incomplete pack")
	}
	if got := strings.Join(audit.MissingSections, ","); got != "open_questions" {
		t.Fatalf("unexpected missing sections: %s", got)
	}
	if got := strings.Join(audit.ObjectivesMissingSignals, ","); got != "obj-incomplete" {
		t.Fatalf("unexpected objective gaps: %s", got)
	}
	if got := strings.Join(audit.WireframesMissingBlocks, ","); got != "wf-empty" {
		t.Fatalf("unexpected wireframe gaps: %s", got)
	}
	if got := strings.Join(audit.InteractionsMissingStates, ","); got != "flow-empty" {
		t.Fatalf("unexpected interaction gaps: %s", got)
	}
}

func TestBuildBIG4204ReviewPackAndReportSurface(t *testing.T) {
	pack := BuildBIG4204ReviewPack()
	audit := Auditor{}.Audit(pack)
	report := RenderPackReport(pack, audit)

	if !audit.Ready {
		t.Fatalf("expected BIG-4204 pack to be ready: %+v", audit)
	}
	if len(pack.Objectives) != 4 || len(pack.Wireframes) != 4 || len(pack.Interactions) != 4 || len(pack.OpenQuestions) != 3 {
		t.Fatalf("unexpected pack counts: %+v", pack)
	}
	want := []string{
		"READY: objectives=4 wireframes=4 interactions=4 open_questions=3 checklist=8 decisions=4 role_assignments=8 signoffs=4 blockers=1 timeline_events=2",
		"## UI Review Review Summary Board",
		"summary-objectives: category=objectives total=4 blocked=1 at-risk=1 covered=2",
		"## UI Review Objective Coverage Board",
		"objcov-obj-run-detail-investigation: objective=obj-run-detail-investigation persona=Eng Lead priority=P0 coverage=blocked dependencies=3 surfaces=wf-run-detail",
		"## UI Review Persona Readiness Board",
		"persona-eng-lead: persona=Eng Lead readiness=blocked objectives=1 assignments=1 signoffs=1 open_questions=0 queue_items=1 blockers=1",
		"## UI Review Wireframe Readiness Board",
		"wire-wf-run-detail: surface=wf-run-detail device=desktop readiness=blocked open_total=4 entry=/runs/detail",
		"## UI Review Interaction Coverage Board",
		"intcov-flow-triage-handoff: flow=flow-triage-handoff surfaces=wf-triage owners=Cross-Team Operator,Platform Admin coverage=covered states=4 exceptions=2",
		"## UI Review Signoff Dependency Board",
		"dep-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead surface=wf-run-detail role=Eng Lead status=pending dependency_status=blocked blockers=blk-run-detail-copy-final",
		"## UI Review Exception Matrix",
		"- product-experience: blockers=1 signoffs=0 total=1",
		"## UI Review Blocker Timeline Summary",
		"- blk-run-detail-copy-final: latest=evt-run-detail-copy-escalated actor=design-program-manager status=escalated at=2026-03-14T09:30:00Z",
	}
	for _, snippet := range want {
		if !strings.Contains(report, snippet) {
			t.Fatalf("expected report to contain %q", snippet)
		}
	}
}

func TestUIReviewAuditFlagsChecklistAndMetadataGaps(t *testing.T) {
	pack := BuildBIG4204ReviewPack()
	pack.ReviewerChecklist = []ReviewerChecklistItem{{
		ItemID:    "chk-overview-kpi-scan",
		SurfaceID: "wf-overview",
		Prompt:    "Verify the KPI strip still supports one-screen executive scanning before drill-down.",
		Owner:     "VP Eng",
		Status:    "ready",
	}}
	pack.SignoffLog[2] = ReviewSignoff{
		SignoffID:       "sig-run-detail-eng-lead",
		AssignmentID:    "role-run-detail-eng-lead",
		SurfaceID:       "wf-run-detail",
		Role:            "Eng Lead",
		Status:          "pending",
		Required:        true,
		EvidenceLinks:   []string{"chk-run-replay-context", "dec-run-detail-audit-rail"},
		SLAStatus:       "breached",
		ReminderChannel: "slack",
		LastReminderAt:  "2026-03-14T09:45:00Z",
	}

	audit := Auditor{}.Audit(pack)
	if audit.Ready {
		t.Fatal("expected audit to fail with checklist and metadata gaps")
	}
	if got := strings.Join(audit.WireframesMissingChecklists, ","); got != "wf-queue,wf-run-detail,wf-triage" {
		t.Fatalf("unexpected missing checklist coverage: %s", got)
	}
	if got := strings.Join(audit.ChecklistItemsMissingEvidence, ","); got != "chk-overview-kpi-scan" {
		t.Fatalf("unexpected checklist evidence gaps: %s", got)
	}
	if got := strings.Join(audit.SignoffsMissingRequestedDates, ","); got != "sig-run-detail-eng-lead" {
		t.Fatalf("unexpected signoff requested-date gaps: %s", got)
	}
	if got := strings.Join(audit.SignoffsMissingDueDates, ","); got != "sig-run-detail-eng-lead" {
		t.Fatalf("unexpected signoff due-date gaps: %s", got)
	}
	if got := strings.Join(audit.SignoffsMissingEscalationOwners, ","); got != "sig-run-detail-eng-lead" {
		t.Fatalf("unexpected signoff escalation-owner gaps: %s", got)
	}
	if got := strings.Join(audit.SignoffsMissingReminderOwners, ","); got != "sig-run-detail-eng-lead" {
		t.Fatalf("unexpected signoff reminder-owner gaps: %s", got)
	}
	if got := strings.Join(audit.SignoffsMissingNextReminders, ","); got != "sig-run-detail-eng-lead" {
		t.Fatalf("unexpected signoff next-reminder gaps: %s", got)
	}
	if got := strings.Join(audit.SignoffsMissingReminderCadence, ","); got != "sig-run-detail-eng-lead" {
		t.Fatalf("unexpected signoff cadence gaps: %s", got)
	}
	if got := strings.Join(audit.SignoffsWithBreachedSLA, ","); got != "sig-run-detail-eng-lead" {
		t.Fatalf("unexpected breached signoffs: %s", got)
	}
}

func TestRenderHTMLAndBundleExport(t *testing.T) {
	pack := BuildBIG4204ReviewPack()
	audit := Auditor{}.Audit(pack)

	html := RenderPackHTML(pack, audit)
	if !strings.Contains(html, "<h2>UI Review Decision Log</h2>") || !strings.Contains(html, "<h2>UI Review Checklist Traceability Board</h2>") {
		t.Fatalf("expected HTML bundle headings, got %s", html)
	}

	dir := t.TempDir()
	artifacts, err := WriteBundle(dir, pack)
	if err != nil {
		t.Fatalf("write bundle: %v", err)
	}
	paths := []string{
		artifacts.MarkdownPath,
		artifacts.HTMLPath,
		artifacts.DecisionLogPath,
		artifacts.ReviewSummaryBoardPath,
		artifacts.ObjectiveCoverageBoardPath,
		artifacts.PersonaReadinessBoardPath,
		artifacts.WireframeReadinessBoardPath,
		artifacts.InteractionCoverageBoardPath,
		artifacts.OpenQuestionTrackerPath,
		artifacts.ChecklistTraceabilityBoardPath,
		artifacts.DecisionFollowupTrackerPath,
		artifacts.RoleMatrixPath,
		artifacts.RoleCoverageBoardPath,
		artifacts.SignoffDependencyBoardPath,
		artifacts.SignoffLogPath,
		artifacts.SignoffSLADashboardPath,
		artifacts.SignoffReminderQueuePath,
		artifacts.ReminderCadenceBoardPath,
		artifacts.SignoffBreachBoardPath,
		artifacts.EscalationDashboardPath,
		artifacts.EscalationHandoffLedgerPath,
		artifacts.HandoffAckLedgerPath,
		artifacts.OwnerEscalationDigestPath,
		artifacts.OwnerWorkloadBoardPath,
		artifacts.BlockerLogPath,
		artifacts.BlockerTimelinePath,
		artifacts.FreezeExceptionBoardPath,
		artifacts.FreezeApprovalTrailPath,
		artifacts.FreezeRenewalTrackerPath,
		artifacts.ExceptionLogPath,
		artifacts.ExceptionMatrixPath,
		artifacts.AuditDensityBoardPath,
		artifacts.OwnerReviewQueuePath,
		artifacts.BlockerTimelineSummaryPath,
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected artifact %s: %v", path, err)
		}
	}
	body, err := os.ReadFile(filepath.Clean(artifacts.ObjectiveCoverageBoardPath))
	if err != nil {
		t.Fatalf("read objective coverage artifact: %v", err)
	}
	if !strings.Contains(string(body), "objcov-obj-run-detail-investigation") {
		t.Fatalf("expected objective coverage artifact content, got %s", string(body))
	}
}
