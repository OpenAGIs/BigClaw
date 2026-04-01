package uireview

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestUIReviewPackRoundTripPreservesManifestShape(t *testing.T) {
	pack := buildReviewPack()

	payload, err := json.Marshal(pack)
	if err != nil {
		t.Fatalf("marshal pack: %v", err)
	}
	var restored UIReviewPack
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal pack: %v", err)
	}
	if !reflect.DeepEqual(restored, pack) {
		t.Fatalf("restored pack mismatch: got %+v want %+v", restored, pack)
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

	audit := UIReviewPackAuditor{}.Audit(pack)

	if audit.Ready {
		t.Fatalf("expected audit to be not ready")
	}
	if got, want := audit.MissingSections, []string{"open_questions"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("missing sections mismatch: got %+v want %+v", got, want)
	}
	if got, want := audit.ObjectivesMissingSignals, []string{"obj-incomplete"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("objectives missing signals mismatch: got %+v want %+v", got, want)
	}
	if got, want := audit.WireframesMissingBlocks, []string{"wf-empty"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("wireframes missing blocks mismatch: got %+v want %+v", got, want)
	}
	if got, want := audit.InteractionsMissingStates, []string{"flow-empty"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("interactions missing states mismatch: got %+v want %+v", got, want)
	}
}

func TestUIReviewPackAuditAllowsOpenQuestionsWhileMarkingPackReady(t *testing.T) {
	pack := buildReviewPack()

	audit := UIReviewPackAuditor{}.Audit(pack)

	if !audit.Ready {
		t.Fatalf("expected audit to be ready, got %+v", audit)
	}
	if got, want := audit.UnresolvedQuestionIDs, []string{"oq-mobile-depth"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unresolved question ids mismatch: got %+v want %+v", got, want)
	}
	if len(audit.MissingSections) != 0 {
		t.Fatalf("expected no missing sections, got %+v", audit.MissingSections)
	}
}

func TestRenderUIReviewPackReportSummarizesReviewShapeAndFindings(t *testing.T) {
	pack := buildReviewPack()
	audit := UIReviewPackAuditor{}.Audit(pack)

	report := RenderUIReviewPackReport(pack, audit)

	for _, needle := range []string{
		"# UI Review Pack",
		"- Issue: BIG-4204 UI评审包输出",
		"- Audit: READY: objectives=1 wireframes=1 interactions=1 open_questions=1 checklist=0 decisions=0 role_assignments=0 signoffs=0 blockers=0 timeline_events=0",
		"- obj-alignment: Align reviewers on the release-control story persona=product-experience priority=P0",
		"- Unresolved questions: oq-mobile-depth",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("expected report to contain %q, got:\n%s", needle, report)
		}
	}
}

func TestUIReviewPackAuditFlagsMissingChecklistCoverageAndEvidence(t *testing.T) {
	pack := buildBig4204ReviewPackFoundation()
	pack.ReviewerChecklist = []ReviewerChecklistItem{{
		ItemID:        "chk-overview-kpi-scan",
		SurfaceID:     "wf-overview",
		Prompt:        "Verify the KPI strip still supports one-screen executive scanning before drill-down.",
		Owner:         "VP Eng",
		Status:        "ready",
		EvidenceLinks: []string{},
	}}

	audit := UIReviewPackAuditor{}.Audit(pack)

	if audit.Ready {
		t.Fatalf("expected audit to be not ready")
	}
	if got, want := audit.WireframesMissingChecklists, []string{"wf-queue", "wf-run-detail", "wf-triage"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("wireframes missing checklists mismatch: got %+v want %+v", got, want)
	}
	if got, want := audit.ChecklistItemsMissingEvidence, []string{"chk-overview-kpi-scan"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("checklist items missing evidence mismatch: got %+v want %+v", got, want)
	}
	if len(audit.OrphanChecklistSurfaces) != 0 {
		t.Fatalf("expected no orphan checklist surfaces, got %+v", audit.OrphanChecklistSurfaces)
	}
}

func TestUIReviewPackAuditFlagsMissingDecisionCoverage(t *testing.T) {
	pack := buildBig4204ReviewPackFoundation()
	pack.DecisionLog = []ReviewDecision{{
		DecisionID: "dec-overview-alert-stack",
		SurfaceID:  "wf-overview",
		Owner:      "product-experience",
		Summary:    "Keep approval and regression alerts in one stacked priority rail.",
		Rationale:  "Reviewers need one comparison lane before jumping into queue or triage surfaces.",
		Status:     "accepted",
	}}

	audit := UIReviewPackAuditor{}.Audit(pack)

	if audit.Ready {
		t.Fatalf("expected audit to be not ready")
	}
	if got, want := audit.WireframesMissingDecisions, []string{"wf-queue", "wf-run-detail", "wf-triage"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("wireframes missing decisions mismatch: got %+v want %+v", got, want)
	}
	if len(audit.OrphanDecisionSurfaces) != 0 {
		t.Fatalf("expected no orphan decision surfaces, got %+v", audit.OrphanDecisionSurfaces)
	}
	if len(audit.UnresolvedDecisionIDs) != 0 {
		t.Fatalf("expected no unresolved decision ids, got %+v", audit.UnresolvedDecisionIDs)
	}
}

func TestUIReviewPackAuditFlagsMissingRoleMatrixCoverage(t *testing.T) {
	pack := buildBig4204ReviewPackFoundation()
	pack.RoleMatrix = []ReviewRoleAssignment{{
		AssignmentID:     "role-overview-vp-eng",
		SurfaceID:        "wf-overview",
		Role:             "VP Eng",
		Responsibilities: []string{"approve overview scan path"},
		ChecklistItemIDs: []string{"chk-overview-kpi-scan"},
		DecisionIDs:      []string{"dec-overview-alert-stack"},
		Status:           "ready",
	}}

	audit := UIReviewPackAuditor{}.Audit(pack)

	if audit.Ready {
		t.Fatalf("expected audit to be not ready")
	}
	if got, want := audit.WireframesMissingRoleAssignments, []string{"wf-queue", "wf-run-detail", "wf-triage"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("wireframes missing role assignments mismatch: got %+v want %+v", got, want)
	}
	if len(audit.OrphanRoleAssignmentSurfaces) != 0 {
		t.Fatalf("expected no orphan role assignment surfaces, got %+v", audit.OrphanRoleAssignmentSurfaces)
	}
	if len(audit.RoleAssignmentsMissingResponsibilities) != 0 {
		t.Fatalf("expected no missing responsibilities, got %+v", audit.RoleAssignmentsMissingResponsibilities)
	}
	if len(audit.RoleAssignmentsMissingChecklistLinks) != 0 {
		t.Fatalf("expected no missing checklist links, got %+v", audit.RoleAssignmentsMissingChecklistLinks)
	}
	if len(audit.RoleAssignmentsMissingDecisionLinks) != 0 {
		t.Fatalf("expected no missing decision links, got %+v", audit.RoleAssignmentsMissingDecisionLinks)
	}
	if got, want := audit.ChecklistItemsMissingRoleLinks, []string{
		"chk-overview-alert-hierarchy",
		"chk-queue-batch-approval",
		"chk-queue-role-density",
		"chk-run-audit-density",
		"chk-run-replay-context",
		"chk-triage-bulk-assign",
		"chk-triage-handoff-clarity",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("checklist items missing role links mismatch: got %+v want %+v", got, want)
	}
	if got, want := audit.DecisionsMissingRoleLinks, []string{
		"dec-queue-vp-summary",
		"dec-run-detail-audit-rail",
		"dec-triage-handoff-density",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("decisions missing role links mismatch: got %+v want %+v", got, want)
	}
}

func buildReviewPack() UIReviewPack {
	return UIReviewPack{
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
			SurfaceID:     "wf-overview",
			Name:          "Review overview board",
			Device:        "desktop",
			EntryPoint:    "Epic 11 review hub",
			PrimaryBlocks: []string{"header", "objective strip", "wireframe rail", "decision log"},
			ReviewNotes:   []string{"Highlight unresolved dependencies before approval."},
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
}

func buildBig4204ReviewPackFoundation() UIReviewPack {
	required := true
	return UIReviewPack{
		IssueID:                   "BIG-4204",
		Title:                     "UI评审包输出",
		Version:                   "v4.0-review-pack",
		RequiresReviewerChecklist: true,
		RequiresDecisionLog:       true,
		RequiresRoleMatrix:        true,
		RequiresSignoffLog:        true,
		RequiresBlockerLog:        true,
		RequiresBlockerTimeline:   true,
		Objectives: []ReviewObjective{
			{ObjectiveID: "obj-overview-alignment", Title: "Overview alignment", Persona: "VP Eng", Outcome: "Align overview critique", SuccessSignal: "Overview ready", Priority: "P0"},
			{ObjectiveID: "obj-queue-governance", Title: "Queue governance", Persona: "Platform Admin", Outcome: "Queue ready", SuccessSignal: "Queue ready", Priority: "P0"},
			{ObjectiveID: "obj-run-detail-investigation", Title: "Run detail investigation", Persona: "Eng Lead", Outcome: "Run detail ready", SuccessSignal: "Run detail ready", Priority: "P0"},
			{ObjectiveID: "obj-triage-handoff", Title: "Triage handoff", Persona: "Cross-Team Operator", Outcome: "Triage ready", SuccessSignal: "Triage ready", Priority: "P1"},
		},
		Wireframes: []WireframeSurface{
			{SurfaceID: "wf-overview", Name: "Overview board", Device: "desktop", EntryPoint: "/overview", PrimaryBlocks: []string{"header"}},
			{SurfaceID: "wf-queue", Name: "Queue board", Device: "desktop", EntryPoint: "/queue", PrimaryBlocks: []string{"header"}},
			{SurfaceID: "wf-run-detail", Name: "Run detail", Device: "desktop", EntryPoint: "/runs/detail", PrimaryBlocks: []string{"header"}},
			{SurfaceID: "wf-triage", Name: "Triage board", Device: "desktop", EntryPoint: "/triage", PrimaryBlocks: []string{"header"}},
		},
		Interactions: []InteractionFlow{
			{FlowID: "flow-overview-scan", Name: "Overview scan", Trigger: "Open overview", SystemResponse: "Overview loads", States: []string{"default"}},
			{FlowID: "flow-queue-approve", Name: "Queue approve", Trigger: "Open queue", SystemResponse: "Queue loads", States: []string{"default"}},
			{FlowID: "flow-run-replay", Name: "Run replay", Trigger: "Open run", SystemResponse: "Run detail loads", States: []string{"default"}},
			{FlowID: "flow-triage-handoff", Name: "Triage handoff", Trigger: "Open triage", SystemResponse: "Triage loads", States: []string{"default"}},
		},
		OpenQuestions: []OpenQuestion{
			{QuestionID: "oq-role-density", Theme: "role-matrix", Question: "Role density?", Owner: "product-experience", Impact: "Queue density"},
			{QuestionID: "oq-alert-priority", Theme: "alerts", Question: "Alert priority?", Owner: "VP Eng", Impact: "Overview hierarchy"},
			{QuestionID: "oq-handoff-evidence", Theme: "handoff", Question: "Handoff evidence?", Owner: "Cross-Team Operator", Impact: "Triage evidence"},
		},
		ReviewerChecklist: []ReviewerChecklistItem{
			{ItemID: "chk-overview-kpi-scan", SurfaceID: "wf-overview", Prompt: "Overview KPI scan", Owner: "VP Eng", Status: "ready", EvidenceLinks: []string{"wf-overview"}},
			{ItemID: "chk-overview-alert-hierarchy", SurfaceID: "wf-overview", Prompt: "Overview alert hierarchy", Owner: "VP Eng", Status: "open", EvidenceLinks: []string{"wf-overview"}},
			{ItemID: "chk-queue-batch-approval", SurfaceID: "wf-queue", Prompt: "Queue batch approval", Owner: "Platform Admin", Status: "ready", EvidenceLinks: []string{"wf-queue"}},
			{ItemID: "chk-queue-role-density", SurfaceID: "wf-queue", Prompt: "Queue role density", Owner: "product-experience", Status: "open", EvidenceLinks: []string{"oq-role-density"}},
			{ItemID: "chk-run-audit-density", SurfaceID: "wf-run-detail", Prompt: "Run audit density", Owner: "engineering-operations", Status: "ready", EvidenceLinks: []string{"wf-run-detail"}},
			{ItemID: "chk-run-replay-context", SurfaceID: "wf-run-detail", Prompt: "Run replay context", Owner: "Eng Lead", Status: "open", EvidenceLinks: []string{"wf-run-detail"}},
			{ItemID: "chk-triage-bulk-assign", SurfaceID: "wf-triage", Prompt: "Triage bulk assign", Owner: "Platform Admin", Status: "ready", EvidenceLinks: []string{"wf-triage"}},
			{ItemID: "chk-triage-handoff-clarity", SurfaceID: "wf-triage", Prompt: "Triage handoff clarity", Owner: "Cross-Team Operator", Status: "ready", EvidenceLinks: []string{"flow-triage-handoff"}},
		},
		DecisionLog: []ReviewDecision{
			{DecisionID: "dec-overview-alert-stack", SurfaceID: "wf-overview", Owner: "product-experience", Summary: "Overview alerts", Rationale: "Need one lane", Status: "accepted"},
			{DecisionID: "dec-queue-vp-summary", SurfaceID: "wf-queue", Owner: "VP Eng", Summary: "Queue summary", Rationale: "Need summary", Status: "proposed", FollowUp: "Resolve after next critique."},
			{DecisionID: "dec-run-detail-audit-rail", SurfaceID: "wf-run-detail", Owner: "Eng Lead", Summary: "Run audit rail", Rationale: "Need audit rail", Status: "accepted"},
			{DecisionID: "dec-triage-handoff-density", SurfaceID: "wf-triage", Owner: "Cross-Team Operator", Summary: "Triage density", Rationale: "Need handoff density", Status: "accepted"},
		},
		RoleMatrix: []ReviewRoleAssignment{
			{AssignmentID: "role-overview-vp-eng", SurfaceID: "wf-overview", Role: "VP Eng", Responsibilities: []string{"approve overview scan path"}, ChecklistItemIDs: []string{"chk-overview-kpi-scan", "chk-overview-alert-hierarchy"}, DecisionIDs: []string{"dec-overview-alert-stack"}, Status: "ready"},
			{AssignmentID: "role-queue-platform-admin", SurfaceID: "wf-queue", Role: "Platform Admin", Responsibilities: []string{"approve queue actions"}, ChecklistItemIDs: []string{"chk-queue-batch-approval"}, DecisionIDs: []string{"dec-queue-vp-summary"}, Status: "ready"},
			{AssignmentID: "role-run-detail-eng-lead", SurfaceID: "wf-run-detail", Role: "Eng Lead", Responsibilities: []string{"approve run detail", "approve replay context"}, ChecklistItemIDs: []string{"chk-run-replay-context"}, DecisionIDs: []string{"dec-run-detail-audit-rail"}, Status: "ready"},
			{AssignmentID: "role-triage-cross-team-operator", SurfaceID: "wf-triage", Role: "Cross-Team Operator", Responsibilities: []string{"approve handoff"}, ChecklistItemIDs: []string{"chk-triage-handoff-clarity"}, DecisionIDs: []string{"dec-triage-handoff-density"}, Status: "ready"},
			{AssignmentID: "role-triage-platform-admin", SurfaceID: "wf-triage", Role: "Platform Admin", Responsibilities: []string{"approve bulk assign"}, ChecklistItemIDs: []string{"chk-triage-bulk-assign"}, DecisionIDs: []string{"dec-triage-handoff-density"}, Status: "ready"},
			{AssignmentID: "role-queue-product-experience", SurfaceID: "wf-queue", Role: "product-experience", Responsibilities: []string{"approve density copy"}, ChecklistItemIDs: []string{"chk-queue-role-density"}, DecisionIDs: []string{"dec-queue-vp-summary"}, Status: "ready"},
			{AssignmentID: "role-run-audit-engineering-operations", SurfaceID: "wf-run-detail", Role: "engineering-operations", Responsibilities: []string{"approve audit density"}, ChecklistItemIDs: []string{"chk-run-audit-density"}, DecisionIDs: []string{"dec-run-detail-audit-rail"}, Status: "ready"},
			{AssignmentID: "role-overview-product-experience", SurfaceID: "wf-overview", Role: "product-experience", Responsibilities: []string{"approve overview narrative"}, ChecklistItemIDs: []string{"chk-overview-kpi-scan"}, DecisionIDs: []string{"dec-overview-alert-stack"}, Status: "ready"},
		},
		SignoffLog: []ReviewSignoff{
			{SignoffID: "sig-overview-vp-eng", AssignmentID: "role-overview-vp-eng", SurfaceID: "wf-overview", Role: "VP Eng", Status: "approved", Required: &required, EvidenceLinks: []string{"chk-overview-kpi-scan"}, RequestedAt: "2026-03-10T10:00:00Z", DueAt: "2026-03-12T18:00:00Z", EscalationOwner: "engineering-director", SLAStatus: "met"},
			{SignoffID: "sig-queue-platform-admin", AssignmentID: "role-queue-platform-admin", SurfaceID: "wf-queue", Role: "Platform Admin", Status: "approved", Required: &required, EvidenceLinks: []string{"chk-queue-batch-approval"}, RequestedAt: "2026-03-10T10:00:00Z", DueAt: "2026-03-12T18:00:00Z", EscalationOwner: "operations-director", SLAStatus: "met"},
			{SignoffID: "sig-run-detail-eng-lead", AssignmentID: "role-run-detail-eng-lead", SurfaceID: "wf-run-detail", Role: "Eng Lead", Status: "pending", Required: &required, EvidenceLinks: []string{"chk-run-replay-context", "dec-run-detail-audit-rail"}, Notes: "Waiting for final replay-state copy review.", RequestedAt: "2026-03-12T11:00:00Z", DueAt: "2026-03-15T18:00:00Z", EscalationOwner: "engineering-director", SLAStatus: "at-risk", ReminderOwner: "design-program-manager", ReminderChannel: "slack", LastReminderAt: "2026-03-14T09:45:00Z", NextReminderAt: "2026-03-15T09:45:00Z", ReminderCadence: "daily"},
			{SignoffID: "sig-triage-cross-team-operator", AssignmentID: "role-triage-cross-team-operator", SurfaceID: "wf-triage", Role: "Cross-Team Operator", Status: "approved", Required: &required, EvidenceLinks: []string{"chk-triage-handoff-clarity"}, RequestedAt: "2026-03-10T10:00:00Z", DueAt: "2026-03-12T18:00:00Z", EscalationOwner: "operations-director", SLAStatus: "met"},
		},
		BlockerLog: []ReviewBlocker{
			{BlockerID: "blk-run-detail-copy-final", SurfaceID: "wf-run-detail", SignoffID: "sig-run-detail-eng-lead", Owner: "product-experience", Summary: "Replay-state copy still needs final wording review before Eng Lead signoff can close.", Status: "open", Severity: "medium", EscalationOwner: "design-program-manager", NextAction: "Review replay-state copy with Eng Lead and update the run-detail frame in the next critique.", FreezeException: true, FreezeOwner: "release-director", FreezeUntil: "2026-03-18T18:00:00Z", FreezeReason: "Allow review pack to ship.", FreezeApprovedBy: "release-director", FreezeApprovedAt: "2026-03-14T08:30:00Z", FreezeRenewalOwner: "release-director", FreezeRenewalBy: "2026-03-17T12:00:00Z", FreezeRenewalStatus: "review-needed"},
		},
		BlockerTimeline: []ReviewBlockerEvent{
			{EventID: "evt-run-detail-copy-opened", BlockerID: "blk-run-detail-copy-final", Actor: "product-experience", Status: "opened", Summary: "Tracked wording gap.", Timestamp: "2026-03-13T10:00:00Z", NextAction: "Review wording changes."},
			{EventID: "evt-run-detail-copy-escalated", BlockerID: "blk-run-detail-copy-final", Actor: "design-program-manager", Status: "escalated", Summary: "Scheduled wording review.", Timestamp: "2026-03-14T09:30:00Z", NextAction: "Refresh frame annotations.", HandoffFrom: "product-experience", HandoffTo: "Eng Lead", Channel: "design-critique", ArtifactRef: "wf-run-detail#copy-v5", AckOwner: "Eng Lead", AckAt: "2026-03-14T10:15:00Z", AckStatus: "acknowledged"},
		},
	}
}
