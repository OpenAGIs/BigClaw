package uireview

import (
	"os"
	"strings"
	"testing"
)

func buildReviewPack() UIReviewPack {
	return UIReviewPack{
		IssueID:       "BIG-4204",
		Title:         "UI评审包输出",
		Version:       "v4.0-review-pack",
		Objectives:    []ReviewObjective{{ObjectiveID: "obj-alignment", Title: "Align reviewers on the release-control story", Persona: "product-experience", Outcome: "Reviewers see the scope, stakes, and success criteria before page-level critique.", SuccessSignal: "The kickoff frame is sufficient to decide whether the slice is review-ready.", Priority: "P0", Dependencies: []string{"BIG-1103", "BIG-1701"}}},
		Wireframes:    []WireframeSurface{{SurfaceID: "wf-overview", Name: "Review overview board", Device: "desktop", EntryPoint: "Epic 11 review hub", PrimaryBlocks: []string{"header", "objective strip", "wireframe rail", "decision log"}, ReviewNotes: []string{"Highlight unresolved dependencies before approval."}}},
		Interactions:  []InteractionFlow{{FlowID: "flow-frame-switch", Name: "Switch between wireframes and interaction notes", Trigger: "Reviewer selects a surface from the wireframe rail", SystemResponse: "The board swaps the focus frame and preserves reviewer comments.", States: []string{"default", "focus", "with-comments"}, Exceptions: []string{"Warn when leaving a frame with unsaved notes."}}},
		OpenQuestions: []OpenQuestion{{QuestionID: "oq-mobile-depth", Theme: "scope", Question: "Should the first review pack cover mobile wireframes or desktop only?", Owner: "product-experience", Impact: "Changes review breadth and the number of required surfaces.", Status: "open"}},
	}
}

func TestUIReviewPackRoundTripPreservesManifestShape(t *testing.T) {
	pack := buildReviewPack()
	restored, err := UIReviewPackFromMap(pack.ToMap())
	if err != nil {
		t.Fatalf("round trip: %v", err)
	}
	if restored.IssueID != pack.IssueID || restored.Title != pack.Title || len(restored.Objectives) != 1 || restored.Objectives[0].ObjectiveID != "obj-alignment" {
		t.Fatalf("unexpected restored pack: %+v", restored)
	}
}

func TestUIReviewPackAuditFlagsMissingSectionsAndCoverageGaps(t *testing.T) {
	pack := UIReviewPack{
		IssueID: "BIG-4204", Title: "UI评审包输出", Version: "v4.0-review-pack",
		Objectives:   []ReviewObjective{{ObjectiveID: "obj-incomplete", Title: "Incomplete objective", Persona: "product-experience", Outcome: "Create a frame for review."}},
		Wireframes:   []WireframeSurface{{SurfaceID: "wf-empty", Name: "Empty frame", Device: "desktop", EntryPoint: "Review hub"}},
		Interactions: []InteractionFlow{{FlowID: "flow-empty", Name: "Unspecified interaction", Trigger: "Reviewer opens the page", SystemResponse: "The system loads the frame."}},
	}
	audit := UIReviewPackAuditor{}.Audit(pack)
	if audit.Ready || strings.Join(audit.MissingSections, ",") != "open_questions" || strings.Join(audit.ObjectivesMissingSignals, ",") != "obj-incomplete" || strings.Join(audit.WireframesMissingBlocks, ",") != "wf-empty" || strings.Join(audit.InteractionsMissingStates, ",") != "flow-empty" {
		t.Fatalf("unexpected audit: %+v", audit)
	}
}

func TestUIReviewPackAuditAllowsOpenQuestionsWhileMarkingPackReady(t *testing.T) {
	audit := UIReviewPackAuditor{}.Audit(buildReviewPack())
	if !audit.Ready || strings.Join(audit.UnresolvedQuestionIDs, ",") != "oq-mobile-depth" || len(audit.MissingSections) != 0 {
		t.Fatalf("unexpected audit: %+v", audit)
	}
}

func TestRenderUIReviewPackReportSummarizesReviewShapeAndFindings(t *testing.T) {
	pack := buildReviewPack()
	report := RenderUIReviewPackReport(pack, UIReviewPackAuditor{}.Audit(pack))
	for _, want := range []string{
		"# UI Review Pack",
		"- Issue: BIG-4204 UI评审包输出",
		"- Audit: READY: objectives=1 wireframes=1 interactions=1 open_questions=1 checklist=0 decisions=0 role_assignments=0 signoffs=0 blockers=0 timeline_events=0",
		"- obj-alignment: Align reviewers on the release-control story persona=product-experience priority=P0",
		"- Unresolved questions: oq-mobile-depth",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in report, got %s", want, report)
		}
	}
}

func TestBuildBig4204ReviewPackIsReadyForDesignSprintReview(t *testing.T) {
	pack := BuildBig4204ReviewPack()
	audit := UIReviewPackAuditor{}.Audit(pack)
	report := RenderUIReviewPackReport(pack, audit)
	if !audit.Ready || len(pack.Objectives) != 4 || len(pack.Wireframes) != 4 || len(pack.Interactions) != 4 || len(pack.OpenQuestions) != 3 || len(pack.ReviewerChecklist) != 8 || len(pack.DecisionLog) != 4 || len(pack.RoleMatrix) != 8 || len(pack.SignoffLog) != 4 || len(pack.BlockerLog) != 1 || len(pack.BlockerTimeline) != 2 {
		t.Fatalf("unexpected pack or audit: %+v %+v", pack, audit)
	}
	for _, want := range []string{
		"obj-queue-governance",
		"## Review Summary Board",
		"summary-objectives: category=objectives total=4 blocked=1 at-risk=1 covered=2",
		"## Objective Coverage Board",
		"- covered: 2",
		"objcov-obj-run-detail-investigation: objective=obj-run-detail-investigation persona=Eng Lead priority=P0 coverage=blocked dependencies=3 surfaces=wf-run-detail",
		"## Persona Readiness Board",
		"persona-eng-lead: persona=Eng Lead readiness=blocked objectives=1 assignments=1 signoffs=1 open_questions=0 queue_items=1 blockers=1",
		"wf-triage: Triage and handoff board",
		"## Wireframe Readiness Board",
		"wire-wf-run-detail: surface=wf-run-detail device=desktop readiness=blocked open_total=4 entry=/runs/detail",
		"flow-run-replay: Run replay with evidence audit",
		"## Interaction Coverage Board",
		"intcov-flow-triage-handoff: flow=flow-triage-handoff surfaces=wf-triage owners=Cross-Team Operator,Platform Admin coverage=covered states=4 exceptions=2",
		"## Open Question Tracker",
		"qtrack-oq-role-density: question=oq-role-density owner=product-experience theme=role-matrix status=open link_status=linked surfaces=wf-queue",
		"chk-queue-batch-approval: surface=wf-queue owner=Platform Admin status=ready",
		"dec-queue-vp-summary: surface=wf-queue owner=VP Eng status=proposed",
		"cover-role-run-detail-eng-lead: assignment=role-run-detail-eng-lead surface=wf-run-detail role=Eng Lead status=ready responsibilities=2 checklist=1 decisions=1",
		"dep-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead surface=wf-run-detail role=Eng Lead status=pending dependency_status=blocked blockers=blk-run-detail-copy-final",
		"blk-run-detail-copy-final: surface=wf-run-detail signoff=sig-run-detail-eng-lead owner=product-experience status=open severity=medium",
		"## Review Exceptions",
		"exc-blk-run-detail-copy-final: type=blocker source=blk-run-detail-copy-final surface=wf-run-detail owner=product-experience status=open severity=medium",
		"## Sign-off SLA Dashboard",
		"sig-run-detail-eng-lead: role=Eng Lead surface=wf-run-detail status=pending sla=at-risk requested_at=2026-03-12T11:00:00Z due_at=2026-03-15T18:00:00Z escalation_owner=engineering-director",
		"## Sign-off Reminder Queue",
		"rem-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail status=pending sla=at-risk owner=design-program-manager channel=slack",
		"## Reminder Cadence Board",
		"cad-rem-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail cadence=daily status=scheduled owner=design-program-manager",
		"## Sign-off Breach Board",
		"breach-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail status=pending sla=at-risk escalation_owner=engineering-director",
		"## Escalation Dashboard",
		"esc-sig-run-detail-eng-lead: owner=engineering-director type=signoff source=sig-run-detail-eng-lead surface=wf-run-detail status=pending priority=at-risk current_owner=Eng Lead",
		"## Escalation Handoff Ledger",
		"handoff-evt-run-detail-copy-escalated: event=evt-run-detail-copy-escalated blocker=blk-run-detail-copy-final surface=wf-run-detail actor=design-program-manager status=escalated at=2026-03-14T09:30:00Z",
		"## Handoff Ack Ledger",
		"ack-evt-run-detail-copy-escalated: event=evt-run-detail-copy-escalated blocker=blk-run-detail-copy-final surface=wf-run-detail handoff_to=Eng Lead ack_owner=Eng Lead ack_status=acknowledged ack_at=2026-03-14T10:15:00Z",
		"## Owner Escalation Digest",
		"## Owner Workload Board",
		"load-renew-blk-run-detail-copy-final: owner=release-director type=renewal source=blk-run-detail-copy-final surface=wf-run-detail status=review-needed lane=renewal",
		"## Review Freeze Exception Board",
		"## Freeze Approval Trail",
		"freeze-approval-blk-run-detail-copy-final: blocker=blk-run-detail-copy-final surface=wf-run-detail status=open owner=release-director approved_by=release-director approved_at=2026-03-14T08:30:00Z window=2026-03-18T18:00:00Z",
		"## Freeze Renewal Tracker",
		"renew-blk-run-detail-copy-final: blocker=blk-run-detail-copy-final surface=wf-run-detail status=open renewal_owner=release-director renewal_by=2026-03-17T12:00:00Z renewal_status=review-needed",
		"## Review Exception Matrix",
		"## Audit Density Board",
		"density-wf-run-detail: surface=wf-run-detail artifact_total=9 open_total=4 band=dense",
		"## Owner Review Queue",
		"queue-sig-run-detail-eng-lead: owner=Eng Lead type=signoff source=sig-run-detail-eng-lead surface=wf-run-detail status=pending",
		"## Blocker Timeline Summary",
		"- escalated: 1",
		"- Wireframes missing checklist coverage: none",
		"- Unresolved decision ids: dec-queue-vp-summary",
		"- Unresolved required signoff ids: sig-run-detail-eng-lead",
		"- Unresolved questions: oq-role-density, oq-alert-priority, oq-handoff-evidence",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in report", want)
		}
	}
}

func TestUIReviewAuditMutationCases(t *testing.T) {
	pack := BuildBig4204ReviewPack()
	pack.ReviewerChecklist = []ReviewerChecklistItem{{ItemID: "chk-overview-kpi-scan", SurfaceID: "wf-overview", Prompt: "Verify", Owner: "VP Eng", Status: "ready"}}
	audit := UIReviewPackAuditor{}.Audit(pack)
	if audit.Ready || strings.Join(audit.WireframesMissingChecklists, ",") != "wf-queue,wf-run-detail,wf-triage" || strings.Join(audit.ChecklistItemsMissingEvidence, ",") != "chk-overview-kpi-scan" {
		t.Fatalf("unexpected checklist audit: %+v", audit)
	}

	pack = BuildBig4204ReviewPack()
	pack.DecisionLog = []ReviewDecision{{DecisionID: "dec-overview-alert-stack", SurfaceID: "wf-overview", Owner: "product-experience", Summary: "Keep approval", Rationale: "Need one lane", Status: "accepted"}}
	audit = UIReviewPackAuditor{}.Audit(pack)
	if audit.Ready || strings.Join(audit.WireframesMissingDecisions, ",") != "wf-queue,wf-run-detail,wf-triage" || len(audit.UnresolvedDecisionIDs) != 0 {
		t.Fatalf("unexpected decision audit: %+v", audit)
	}

	pack = BuildBig4204ReviewPack()
	pack.RoleMatrix = []ReviewRoleAssignment{{AssignmentID: "role-overview-vp-eng", SurfaceID: "wf-overview", Role: "VP Eng", Responsibilities: []string{"approve overview scan path"}, ChecklistItemIDs: []string{"chk-overview-kpi-scan"}, DecisionIDs: []string{"dec-overview-alert-stack"}, Status: "ready"}}
	audit = UIReviewPackAuditor{}.Audit(pack)
	if audit.Ready || strings.Join(audit.WireframesMissingRoleAssignments, ",") != "wf-queue,wf-run-detail,wf-triage" || strings.Join(audit.ChecklistItemsMissingRoleLinks, ",") != "chk-overview-alert-hierarchy,chk-queue-batch-approval,chk-queue-role-density,chk-run-replay-context,chk-run-audit-density,chk-triage-handoff-clarity,chk-triage-bulk-assign" {
		t.Fatalf("unexpected role audit: %+v", audit)
	}

	pack = BuildBig4204ReviewPack()
	pack.SignoffLog = []ReviewSignoff{{SignoffID: "sig-overview-vp-eng", AssignmentID: "role-overview-missing", SurfaceID: "wf-overview", Role: "VP Eng", Status: "approved", Required: true, EvidenceLinks: []string{"chk-overview-kpi-scan"}}}
	audit = UIReviewPackAuditor{}.Audit(pack)
	if audit.Ready || strings.Join(audit.WireframesMissingSignoffs, ",") != "wf-queue,wf-run-detail,wf-triage" || strings.Join(audit.SignoffsMissingAssignments, ",") != "sig-overview-vp-eng" {
		t.Fatalf("unexpected signoff audit: %+v", audit)
	}
}

func TestUIReviewAuditMetadataCases(t *testing.T) {
	pack := BuildBig4204ReviewPack()
	pack.SignoffLog[2] = ReviewSignoff{SignoffID: "sig-run-detail-eng-lead", AssignmentID: "role-run-detail-eng-lead", SurfaceID: "wf-run-detail", Role: "Eng Lead", Status: "pending", Required: true, EvidenceLinks: []string{"chk-run-replay-context", "dec-run-detail-audit-rail"}, Notes: "Waiting", SLAStatus: "breached", ReminderChannel: "slack", LastReminderAt: "2026-03-14T09:45:00Z"}
	audit := UIReviewPackAuditor{}.Audit(pack)
	if audit.Ready || strings.Join(audit.SignoffsMissingRequestedDates, ",") != "sig-run-detail-eng-lead" || strings.Join(audit.SignoffsWithBreachedSLA, ",") != "sig-run-detail-eng-lead" {
		t.Fatalf("unexpected signoff metadata audit: %+v", audit)
	}

	pack = BuildBig4204ReviewPack()
	pack.DecisionLog[1] = ReviewDecision{DecisionID: "dec-queue-vp-summary", SurfaceID: "wf-queue", Owner: "VP Eng", Summary: "Route VP Eng", Rationale: "Needs summary", Status: "proposed"}
	audit = UIReviewPackAuditor{}.Audit(pack)
	if audit.Ready || strings.Join(audit.UnresolvedDecisionIDs, ",") != "dec-queue-vp-summary" || strings.Join(audit.UnresolvedDecisionsMissingFollowUps, ",") != "dec-queue-vp-summary" {
		t.Fatalf("unexpected unresolved decision audit: %+v", audit)
	}

	pack = BuildBig4204ReviewPack()
	pack.BlockerLog[0].FreezeOwner = ""
	pack.BlockerLog[0].FreezeUntil = ""
	pack.BlockerLog[0].FreezeApprovedBy = ""
	pack.BlockerLog[0].FreezeApprovedAt = ""
	pack.BlockerLog[0].FreezeRenewalOwner = ""
	pack.BlockerLog[0].FreezeRenewalBy = ""
	pack.BlockerTimeline[1].HandoffTo = ""
	pack.BlockerTimeline[1].ArtifactRef = ""
	pack.BlockerTimeline[1].AckOwner = ""
	pack.BlockerTimeline[1].AckAt = ""
	audit = UIReviewPackAuditor{}.Audit(pack)
	if audit.Ready || strings.Join(audit.FreezeExceptionsMissingOwners, ",") != "blk-run-detail-copy-final" || strings.Join(audit.HandoffEventsMissingTargets, ",") != "evt-run-detail-copy-escalated" {
		t.Fatalf("unexpected blocker metadata audit: %+v", audit)
	}

	pack = BuildBig4204ReviewPack()
	pack.BlockerLog = nil
	audit = UIReviewPackAuditor{}.Audit(pack)
	if audit.Ready || strings.Join(audit.UnresolvedRequiredSignoffIDs, ",") != "sig-run-detail-eng-lead" || strings.Join(audit.UnresolvedRequiredSignoffsWithoutBlockers, ",") != "sig-run-detail-eng-lead" {
		t.Fatalf("unexpected unresolved signoff audit: %+v", audit)
	}

	pack = BuildBig4204ReviewPack()
	pack.SignoffLog[2] = ReviewSignoff{SignoffID: "sig-run-detail-eng-lead", AssignmentID: "role-run-detail-eng-lead", SurfaceID: "wf-run-detail", Role: "Eng Lead", Status: "waived", Required: true, Notes: "Temporary waiver approved pending copy lock."}
	pack.BlockerLog = nil
	pack.BlockerTimeline = nil
	audit = UIReviewPackAuditor{}.Audit(pack)
	if audit.Ready || strings.Join(audit.WaivedSignoffsMissingMetadata, ",") != "sig-run-detail-eng-lead" {
		t.Fatalf("unexpected waived signoff audit: %+v", audit)
	}

	pack = BuildBig4204ReviewPack()
	pack.BlockerTimeline = nil
	audit = UIReviewPackAuditor{}.Audit(pack)
	if audit.Ready || strings.Join(audit.BlockersMissingTimelineEvents, ",") != "blk-run-detail-copy-final" {
		t.Fatalf("unexpected missing timeline audit: %+v", audit)
	}

	pack = BuildBig4204ReviewPack()
	pack.BlockerLog[0].Status = "closed"
	pack.BlockerTimeline = []ReviewBlockerEvent{
		{EventID: "evt-run-detail-copy-opened", BlockerID: "blk-run-detail-copy-final", Actor: "product-experience", Status: "opened", Summary: "Tracked", Timestamp: "2026-03-13T10:00:00Z"},
		{EventID: "evt-orphan-blocker", BlockerID: "blk-missing", Actor: "design-program-manager", Status: "escalated", Summary: "Unexpected", Timestamp: "2026-03-14T11:00:00Z", NextAction: "Remove orphaned timeline entry from the bundle."},
	}
	audit = UIReviewPackAuditor{}.Audit(pack)
	if audit.Ready || strings.Join(audit.ClosedBlockersMissingResolutionEvents, ",") != "blk-run-detail-copy-final" || strings.Join(audit.OrphanBlockerTimelineBlockerIDs, ",") != "blk-missing" {
		t.Fatalf("unexpected closed blocker audit: %+v", audit)
	}
}

func TestRenderUIReviewSupplementalBoards(t *testing.T) {
	pack := BuildBig4204ReviewPack()
	signoffSLA := RenderUIReviewSignoffSLADashboard(pack)
	signoffReminder := RenderUIReviewSignoffReminderQueue(pack)
	signoffBreach := RenderUIReviewSignoffBreachBoard(pack)
	escalationDashboard := RenderUIReviewEscalationDashboard(pack)
	handoffLedger := RenderUIReviewEscalationHandoffLedger(pack)
	ownerDigest := RenderUIReviewOwnerEscalationDigest(pack)
	exceptionMatrix := RenderUIReviewExceptionMatrix(pack)
	freezeBoard := RenderUIReviewFreezeExceptionBoard(pack)
	freezeTrail := RenderUIReviewFreezeApprovalTrail(pack)
	reviewSummary := RenderUIReviewReviewSummaryBoard(pack)
	objectiveCoverage := RenderUIReviewObjectiveCoverageBoard(pack)
	questionTracker := RenderUIReviewOpenQuestionTracker(pack)
	decisionFollowup := RenderUIReviewDecisionFollowupTracker(pack)
	signoffDependency := RenderUIReviewSignoffDependencyBoard(pack)
	ownerWorkload := RenderUIReviewOwnerWorkloadBoard(pack)
	ownerQueue := RenderUIReviewOwnerReviewQueue(pack)
	exceptionLog := RenderUIReviewExceptionLog(pack)
	timelineSummary := RenderUIReviewBlockerTimelineSummary(pack)
	html := RenderUIReviewPackHTML(pack, UIReviewPackAuditor{}.Audit(pack))
	for _, want := range []string{"# UI Review Sign-off SLA Dashboard", "- at-risk: 1", "sig-run-detail-eng-lead: role=Eng Lead surface=wf-run-detail status=pending sla=at-risk requested_at=2026-03-12T11:00:00Z due_at=2026-03-15T18:00:00Z escalation_owner=engineering-director"} {
		if !strings.Contains(signoffSLA, want) {
			t.Fatalf("missing %q", want)
		}
	}
	for _, want := range []string{
		"# UI Review Sign-off Reminder Queue", "rem-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail status=pending sla=at-risk owner=design-program-manager channel=slack",
		"# UI Review Sign-off Breach Board", "breach-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail status=pending sla=at-risk escalation_owner=engineering-director",
		"# UI Review Escalation Dashboard", "esc-sig-run-detail-eng-lead: owner=engineering-director type=signoff source=sig-run-detail-eng-lead surface=wf-run-detail status=pending priority=at-risk current_owner=Eng Lead",
		"# UI Review Escalation Handoff Ledger", "from=product-experience to=Eng Lead channel=design-critique artifact=wf-run-detail#copy-v5",
		"# UI Review Owner Escalation Digest", "digest-rem-sig-run-detail-eng-lead: owner=design-program-manager type=reminder source=sig-run-detail-eng-lead surface=wf-run-detail status=pending",
		"# UI Review Exception Matrix", "- product-experience: blockers=1 signoffs=0 total=1",
		"# UI Review Freeze Exception Board", "freeze-blk-run-detail-copy-final: owner=release-director type=blocker source=blk-run-detail-copy-final surface=wf-run-detail status=open window=2026-03-18T18:00:00Z",
		"# UI Review Freeze Approval Trail", "freeze-approval-blk-run-detail-copy-final: blocker=blk-run-detail-copy-final surface=wf-run-detail status=open owner=release-director approved_by=release-director approved_at=2026-03-14T08:30:00Z window=2026-03-18T18:00:00Z",
		"# UI Review Review Summary Board", "summary-personas: category=personas total=4 blocked=1 at-risk=1 ready=2",
		"# UI Review Objective Coverage Board", "dependency_ids=BIG-4203,OPE-72,OPE-73 assignments=role-run-detail-eng-lead checklist=chk-run-replay-context decisions=dec-run-detail-audit-rail signoffs=sig-run-detail-eng-lead blockers=blk-run-detail-copy-final",
		"# UI Review Open Question Tracker", "checklist=chk-queue-role-density flows=none impact=Changes denial-path copy, button placement, and review criteria for queue and triage pages.",
		"# UI Review Decision Follow-up Tracker", "follow-dec-queue-vp-summary: decision=dec-queue-vp-summary surface=wf-queue owner=VP Eng status=proposed linked_roles=Platform Admin,product-experience",
		"# UI Review Signoff Dependency Board", "assignment=role-run-detail-eng-lead checklist=chk-run-replay-context decisions=dec-run-detail-audit-rail latest_blocker_event=evt-run-detail-copy-escalated/escalated/design-program-manager@2026-03-14T09:30:00Z sla=at-risk due_at=2026-03-15T18:00:00Z cadence=daily",
		"# UI Review Owner Workload Board", "load-rem-sig-run-detail-eng-lead: owner=design-program-manager type=reminder source=sig-run-detail-eng-lead surface=wf-run-detail status=pending lane=reminder",
		"# UI Review Owner Review Queue", "- Queue items: 6",
		"# UI Review Exception Log", "evt-run-detail-copy-escalated/escalated/design-program-manager@2026-03-14T09:30:00Z",
		"# UI Review Blocker Timeline Summary", "- blk-run-detail-copy-final: latest=evt-run-detail-copy-escalated actor=design-program-manager status=escalated at=2026-03-14T09:30:00Z",
		"<h2>Decision Log</h2>", "<h2>Checklist Traceability Board</h2>", "<h2>Blocker Timeline Summary</h2>",
	} {
		all := signoffReminder + signoffBreach + escalationDashboard + handoffLedger + ownerDigest + exceptionMatrix + freezeBoard + freezeTrail + reviewSummary + objectiveCoverage + questionTracker + decisionFollowup + signoffDependency + ownerWorkload + ownerQueue + exceptionLog + timelineSummary + html
		if !strings.Contains(all, want) {
			t.Fatalf("missing %q", want)
		}
	}
}

func TestRenderUIReviewExceptionMatrixIncludesWaivedSignoff(t *testing.T) {
	pack := BuildBig4204ReviewPack()
	pack.SignoffLog[2] = ReviewSignoff{SignoffID: "sig-run-detail-eng-lead", AssignmentID: "role-run-detail-eng-lead", SurfaceID: "wf-run-detail", Role: "Eng Lead", Status: "waived", Required: true, EvidenceLinks: []string{"chk-run-replay-context", "dec-run-detail-audit-rail"}, Notes: "Temporary waiver approved pending copy lock.", WaiverOwner: "Eng Lead", WaiverReason: "Copy review is deferred to the next wording pass."}
	matrix := RenderUIReviewExceptionMatrix(pack)
	for _, want := range []string{"- Exceptions: 2", "- Owners: 2", "- Surfaces: 1", "- Eng Lead: blockers=0 signoffs=1 total=1", "- product-experience: blockers=1 signoffs=0 total=1", "- open: blockers=1 signoffs=0 total=1", "- waived: blockers=0 signoffs=1 total=1", "- wf-run-detail: blockers=1 signoffs=1 total=2"} {
		if !strings.Contains(matrix, want) {
			t.Fatalf("missing %q", want)
		}
	}
}

func TestRenderUIReviewHTMLAndBundleExport(t *testing.T) {
	tmp := t.TempDir()
	pack := BuildBig4204ReviewPack()
	audit := UIReviewPackAuditor{}.Audit(pack)
	html := RenderUIReviewPackHTML(pack, audit)
	if !strings.Contains(html, "<h2>Decision Log</h2>") || !strings.Contains(html, "<h2>Role Matrix</h2>") {
		t.Fatalf("unexpected html: %s", html)
	}
	artifacts, err := WriteUIReviewPackBundle(tmp, pack)
	if err != nil {
		t.Fatalf("bundle export: %v", err)
	}
	for _, path := range []string{
		artifacts.MarkdownPath, artifacts.HTMLPath, artifacts.DecisionLogPath, artifacts.ReviewSummaryBoardPath, artifacts.ObjectiveCoverageBoardPath,
		artifacts.PersonaReadinessBoardPath, artifacts.WireframeReadinessBoardPath, artifacts.InteractionCoverageBoardPath, artifacts.OpenQuestionTrackerPath,
		artifacts.ChecklistTraceabilityBoardPath, artifacts.DecisionFollowupTrackerPath, artifacts.RoleMatrixPath, artifacts.RoleCoverageBoardPath,
		artifacts.SignoffDependencyBoardPath, artifacts.SignoffLogPath, artifacts.SignoffSLADashboardPath, artifacts.SignoffReminderQueuePath,
		artifacts.ReminderCadenceBoardPath, artifacts.SignoffBreachBoardPath, artifacts.EscalationDashboardPath, artifacts.EscalationHandoffLedgerPath,
		artifacts.HandoffAckLedgerPath, artifacts.OwnerEscalationDigestPath, artifacts.OwnerWorkloadBoardPath, artifacts.BlockerLogPath, artifacts.BlockerTimelinePath,
		artifacts.FreezeExceptionBoardPath, artifacts.FreezeApprovalTrailPath, artifacts.FreezeRenewalTrackerPath, artifacts.ExceptionLogPath, artifacts.ExceptionMatrixPath,
		artifacts.AuditDensityBoardPath, artifacts.OwnerReviewQueuePath, artifacts.BlockerTimelineSummaryPath,
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing artifact %s: %v", path, err)
		}
	}
	checks := map[string]string{
		artifacts.HTMLPath:                       "Decision Log",
		artifacts.DecisionLogPath:                "dec-triage-handoff-density",
		artifacts.ReviewSummaryBoardPath:         "summary-objectives: category=objectives total=4 blocked=1 at-risk=1 covered=2",
		artifacts.ObjectiveCoverageBoardPath:     "objcov-obj-run-detail-investigation: objective=obj-run-detail-investigation persona=Eng Lead priority=P0 coverage=blocked dependencies=3 surfaces=wf-run-detail",
		artifacts.PersonaReadinessBoardPath:      "persona-eng-lead: persona=Eng Lead readiness=blocked objectives=1 assignments=1 signoffs=1 open_questions=0 queue_items=1 blockers=1",
		artifacts.WireframeReadinessBoardPath:    "wire-wf-run-detail: surface=wf-run-detail device=desktop readiness=blocked open_total=4 entry=/runs/detail",
		artifacts.InteractionCoverageBoardPath:   "intcov-flow-triage-handoff: flow=flow-triage-handoff surfaces=wf-triage owners=Cross-Team Operator,Platform Admin coverage=covered states=4 exceptions=2",
		artifacts.OpenQuestionTrackerPath:        "qtrack-oq-role-density: question=oq-role-density owner=product-experience theme=role-matrix status=open link_status=linked surfaces=wf-queue",
		artifacts.ChecklistTraceabilityBoardPath: "trace-chk-queue-role-density: item=chk-queue-role-density surface=wf-queue owner=product-experience status=open linked_roles=product-experience",
		artifacts.DecisionFollowupTrackerPath:    "follow-dec-queue-vp-summary: decision=dec-queue-vp-summary surface=wf-queue owner=VP Eng status=proposed linked_roles=Platform Admin,product-experience",
		artifacts.RoleMatrixPath:                 "role-triage-platform-admin",
		artifacts.RoleCoverageBoardPath:          "cover-role-run-detail-eng-lead: assignment=role-run-detail-eng-lead surface=wf-run-detail role=Eng Lead status=ready responsibilities=2 checklist=1 decisions=1",
		artifacts.SignoffDependencyBoardPath:     "dep-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead surface=wf-run-detail role=Eng Lead status=pending dependency_status=blocked blockers=blk-run-detail-copy-final",
		artifacts.SignoffLogPath:                 "sig-queue-platform-admin",
		artifacts.SignoffSLADashboardPath:        "- at-risk: 1",
		artifacts.SignoffReminderQueuePath:       "rem-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail status=pending sla=at-risk owner=design-program-manager channel=slack",
		artifacts.ReminderCadenceBoardPath:       "cad-rem-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail cadence=daily status=scheduled owner=design-program-manager",
		artifacts.SignoffBreachBoardPath:         "breach-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail status=pending sla=at-risk escalation_owner=engineering-director",
		artifacts.EscalationDashboardPath:        "esc-sig-run-detail-eng-lead: owner=engineering-director type=signoff source=sig-run-detail-eng-lead surface=wf-run-detail status=pending priority=at-risk current_owner=Eng Lead",
		artifacts.EscalationHandoffLedgerPath:    "handoff-evt-run-detail-copy-escalated: event=evt-run-detail-copy-escalated blocker=blk-run-detail-copy-final surface=wf-run-detail actor=design-program-manager status=escalated at=2026-03-14T09:30:00Z",
		artifacts.HandoffAckLedgerPath:           "ack-evt-run-detail-copy-escalated: event=evt-run-detail-copy-escalated blocker=blk-run-detail-copy-final surface=wf-run-detail handoff_to=Eng Lead ack_owner=Eng Lead ack_status=acknowledged ack_at=2026-03-14T10:15:00Z",
		artifacts.OwnerEscalationDigestPath:      "digest-rem-sig-run-detail-eng-lead: owner=design-program-manager type=reminder source=sig-run-detail-eng-lead surface=wf-run-detail status=pending",
		artifacts.OwnerWorkloadBoardPath:         "load-rem-sig-run-detail-eng-lead: owner=design-program-manager type=reminder source=sig-run-detail-eng-lead surface=wf-run-detail status=pending lane=reminder",
		artifacts.BlockerLogPath:                 "blk-run-detail-copy-final",
		artifacts.BlockerTimelinePath:            "evt-run-detail-copy-opened",
		artifacts.FreezeExceptionBoardPath:       "freeze-blk-run-detail-copy-final: owner=release-director type=blocker source=blk-run-detail-copy-final surface=wf-run-detail status=open window=2026-03-18T18:00:00Z",
		artifacts.FreezeApprovalTrailPath:        "freeze-approval-blk-run-detail-copy-final: blocker=blk-run-detail-copy-final surface=wf-run-detail status=open owner=release-director approved_by=release-director approved_at=2026-03-14T08:30:00Z window=2026-03-18T18:00:00Z",
		artifacts.FreezeRenewalTrackerPath:       "renew-blk-run-detail-copy-final: blocker=blk-run-detail-copy-final surface=wf-run-detail status=open renewal_owner=release-director renewal_by=2026-03-17T12:00:00Z renewal_status=review-needed",
		artifacts.ExceptionLogPath:               "exc-blk-run-detail-copy-final",
		artifacts.ExceptionMatrixPath:            "- product-experience: blockers=1 signoffs=0 total=1",
		artifacts.AuditDensityBoardPath:          "density-wf-run-detail: surface=wf-run-detail artifact_total=9 open_total=4 band=dense",
		artifacts.OwnerReviewQueuePath:           "- Queue items: 6",
		artifacts.BlockerTimelineSummaryPath:     "- escalated: 1",
	}
	for path, want := range checks {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if !strings.Contains(string(body), want) {
			t.Fatalf("expected %q in %s", want, path)
		}
	}
}
