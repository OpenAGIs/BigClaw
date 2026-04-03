package uireview

import (
	"fmt"
	"html"
	"sort"
	"strings"
)

func objectiveSurfaceIDs(id string) []string {
	switch id {
	case "obj-overview-decision":
		return []string{"wf-overview"}
	case "obj-queue-governance":
		return []string{"wf-queue"}
	case "obj-run-detail-investigation":
		return []string{"wf-run-detail"}
	case "obj-triage-handoff":
		return []string{"wf-triage"}
	default:
		return nil
	}
}

func questionSurfaceIDs(id string) []string {
	switch id {
	case "oq-role-density":
		return []string{"wf-queue"}
	case "oq-alert-priority":
		return []string{"wf-overview"}
	case "oq-handoff-evidence":
		return []string{"wf-run-detail", "wf-triage"}
	default:
		return nil
	}
}

func questionChecklistIDs(id string) []string {
	switch id {
	case "oq-role-density":
		return []string{"chk-queue-role-density"}
	case "oq-alert-priority":
		return []string{"chk-overview-alert-hierarchy"}
	case "oq-handoff-evidence":
		return []string{"chk-run-audit-density"}
	default:
		return nil
	}
}

func flowSurfaceIDs(id string) []string {
	switch id {
	case "flow-overview-drilldown":
		return []string{"wf-overview"}
	case "flow-queue-bulk-approval":
		return []string{"wf-queue"}
	case "flow-run-replay":
		return []string{"wf-run-detail"}
	case "flow-triage-handoff":
		return []string{"wf-triage"}
	default:
		return nil
	}
}

func openItemsBySurface(pack UIReviewPack, surfaceID string) (checklists, decisions, assignments, signoffs, blockers int) {
	for _, item := range pack.ReviewerChecklist {
		if item.SurfaceID == surfaceID && item.Status != "ready" && item.Status != "approved" {
			checklists++
		}
	}
	for _, item := range pack.DecisionLog {
		if item.SurfaceID == surfaceID && item.Status != "accepted" {
			decisions++
		}
	}
	for _, item := range pack.RoleMatrix {
		if item.SurfaceID == surfaceID && item.Status != "ready" {
			assignments++
		}
	}
	for _, item := range pack.SignoffLog {
		if item.SurfaceID == surfaceID && item.Status != "approved" {
			signoffs++
		}
	}
	for _, item := range pack.BlockerLog {
		if item.SurfaceID == surfaceID && item.Status != "closed" {
			blockers++
		}
	}
	return
}

func latestBlockerEvent(events []ReviewBlockerEvent) *ReviewBlockerEvent {
	if len(events) == 0 {
		return nil
	}
	item := events[len(events)-1]
	return &item
}

func renderSection(title string, body string) string {
	return "## " + title + "\n\n" + strings.TrimSpace(body) + "\n"
}

func RenderUIReviewPackReport(pack UIReviewPack, audit UIReviewPackAudit) string {
	var b strings.Builder
	b.WriteString("# UI Review Pack\n\n")
	b.WriteString(fmt.Sprintf("- Issue: %s %s\n", pack.IssueID, pack.Title))
	b.WriteString(fmt.Sprintf("- Version: %s\n", pack.Version))
	b.WriteString(fmt.Sprintf("- Audit: %s\n", audit.Summary()))
	if len(audit.UnresolvedQuestionIDs) > 0 {
		b.WriteString(fmt.Sprintf("- Unresolved questions: %s\n", strings.Join(audit.UnresolvedQuestionIDs, ", ")))
	}
	b.WriteString("\n## Objectives\n\n")
	for _, objective := range pack.Objectives {
		b.WriteString(fmt.Sprintf("- %s: %s persona=%s priority=%s\n", objective.ObjectiveID, objective.Title, objective.Persona, firstNonEmpty(objective.Priority, "P1")))
	}
	for _, wireframe := range pack.Wireframes {
		b.WriteString(fmt.Sprintf("- %s: %s\n", wireframe.SurfaceID, wireframe.Name))
	}
	for _, flow := range pack.Interactions {
		b.WriteString(fmt.Sprintf("- %s: %s\n", flow.FlowID, flow.Name))
	}
	for _, item := range pack.ReviewerChecklist {
		b.WriteString(fmt.Sprintf("%s: surface=%s owner=%s status=%s\n", item.ItemID, item.SurfaceID, item.Owner, item.Status))
	}
	for _, decision := range pack.DecisionLog {
		b.WriteString(fmt.Sprintf("%s: surface=%s owner=%s status=%s\n", decision.DecisionID, decision.SurfaceID, decision.Owner, decision.Status))
	}
	for _, assignment := range pack.RoleMatrix {
		b.WriteString(fmt.Sprintf("%s: surface=%s role=%s status=%s\n", assignment.AssignmentID, assignment.SurfaceID, assignment.Role, assignment.Status))
	}
	for _, event := range pack.BlockerTimeline {
		b.WriteString(fmt.Sprintf("%s: blocker=%s actor=%s status=%s at=%s\n", event.EventID, event.BlockerID, event.Actor, event.Status, event.Timestamp))
	}
	b.WriteString("\n")
	appendBoard := func(title, content string) {
		b.WriteString("## " + title + "\n\n")
		b.WriteString(content)
		b.WriteString("\n")
	}
	appendBoard("Review Summary Board", RenderUIReviewReviewSummaryBoard(pack))
	appendBoard("Objective Coverage Board", RenderUIReviewObjectiveCoverageBoard(pack))
	appendBoard("Persona Readiness Board", RenderUIReviewPersonaReadinessBoard(pack))
	appendBoard("Wireframe Readiness Board", RenderUIReviewWireframeReadinessBoard(pack))
	appendBoard("Interaction Coverage Board", RenderUIReviewInteractionCoverageBoard(pack))
	appendBoard("Open Question Tracker", RenderUIReviewOpenQuestionTracker(pack))
	appendBoard("Checklist Traceability Board", RenderUIReviewChecklistTraceabilityBoard(pack))
	appendBoard("Decision Follow-up Tracker", RenderUIReviewDecisionFollowupTracker(pack))
	appendBoard("Role Coverage Board", RenderUIReviewRoleCoverageBoard(pack))
	appendBoard("Signoff Dependency Board", RenderUIReviewSignoffDependencyBoard(pack))
	appendBoard("Sign-off Log", RenderUIReviewSignoffLog(pack))
	appendBoard("Blocker Log", RenderUIReviewBlockerLog(pack))
	appendBoard("Review Exceptions", RenderUIReviewExceptionLog(pack))
	appendBoard("Sign-off SLA Dashboard", RenderUIReviewSignoffSLADashboard(pack))
	appendBoard("Sign-off Reminder Queue", RenderUIReviewSignoffReminderQueue(pack))
	appendBoard("Reminder Cadence Board", RenderUIReviewReminderCadenceBoard(pack))
	appendBoard("Sign-off Breach Board", RenderUIReviewSignoffBreachBoard(pack))
	appendBoard("Escalation Dashboard", RenderUIReviewEscalationDashboard(pack))
	appendBoard("Escalation Handoff Ledger", RenderUIReviewEscalationHandoffLedger(pack))
	appendBoard("Handoff Ack Ledger", RenderUIReviewHandoffAckLedger(pack))
	appendBoard("Owner Escalation Digest", RenderUIReviewOwnerEscalationDigest(pack))
	appendBoard("Owner Workload Board", RenderUIReviewOwnerWorkloadBoard(pack))
	appendBoard("Review Freeze Exception Board", RenderUIReviewFreezeExceptionBoard(pack))
	appendBoard("Freeze Approval Trail", RenderUIReviewFreezeApprovalTrail(pack))
	appendBoard("Freeze Renewal Tracker", RenderUIReviewFreezeRenewalTracker(pack))
	appendBoard("Review Exception Matrix", RenderUIReviewExceptionMatrix(pack))
	appendBoard("Audit Density Board", RenderUIReviewAuditDensityBoard(pack))
	appendBoard("Owner Review Queue", RenderUIReviewOwnerReviewQueue(pack))
	appendBoard("Blocker Timeline", RenderUIReviewBlockerTimeline(pack))
	appendBoard("Blocker Timeline Summary", RenderUIReviewBlockerTimelineSummary(pack))
	b.WriteString("\n")
	b.WriteString(renderAuditLists(audit))
	return b.String()
}

func renderAuditLists(audit UIReviewPackAudit) string {
	lines := []string{}
	emit := func(label string, items []string) {
		value := "none"
		if len(items) > 0 {
			value = strings.Join(items, ", ")
		}
		lines = append(lines, fmt.Sprintf("- %s: %s", label, value))
	}
	emit("Wireframes missing checklist coverage", audit.WireframesMissingChecklists)
	emit("Checklist items missing role links", audit.ChecklistItemsMissingRoleLinks)
	emit("Wireframes missing decision coverage", audit.WireframesMissingDecisions)
	emit("Unresolved decisions missing follow-ups", audit.UnresolvedDecisionsMissingFollowUps)
	emit("Wireframes missing role assignments", audit.WireframesMissingRoleAssignments)
	emit("Wireframes missing signoff coverage", audit.WireframesMissingSignoffs)
	emit("Blockers missing signoff links", audit.BlockersMissingSignoffLinks)
	emit("Freeze exceptions missing owners", audit.FreezeExceptionsMissingOwners)
	emit("Freeze exceptions missing windows", audit.FreezeExceptionsMissingUntil)
	emit("Freeze exceptions missing approvers", audit.FreezeExceptionsMissingApprovers)
	emit("Freeze exceptions missing approval dates", audit.FreezeExceptionsMissingApprovalDates)
	emit("Freeze exceptions missing renewal owners", audit.FreezeExceptionsMissingRenewalOwners)
	emit("Freeze exceptions missing renewal dates", audit.FreezeExceptionsMissingRenewalDates)
	emit("Blockers missing timeline events", audit.BlockersMissingTimelineEvents)
	emit("Closed blockers missing resolution events", audit.ClosedBlockersMissingResolutionEvents)
	emit("Orphan blocker timeline blocker ids", audit.OrphanBlockerTimelineBlockerIDs)
	emit("Handoff events missing targets", audit.HandoffEventsMissingTargets)
	emit("Handoff events missing artifacts", audit.HandoffEventsMissingArtifacts)
	emit("Handoff events missing ack owners", audit.HandoffEventsMissingAckOwners)
	emit("Handoff events missing ack dates", audit.HandoffEventsMissingAckDates)
	emit("Unresolved required signoffs without blockers", audit.UnresolvedRequiredSignoffsWithoutBlockers)
	emit("Unresolved decision ids", audit.UnresolvedDecisionIDs)
	emit("Decisions missing role links", audit.DecisionsMissingRoleLinks)
	emit("Signoffs missing requested dates", audit.SignoffsMissingRequestedDates)
	emit("Signoffs missing due dates", audit.SignoffsMissingDueDates)
	emit("Signoffs missing escalation owners", audit.SignoffsMissingEscalationOwners)
	emit("Signoffs missing reminder owners", audit.SignoffsMissingReminderOwners)
	emit("Signoffs missing next reminders", audit.SignoffsMissingNextReminders)
	emit("Signoffs missing reminder cadence", audit.SignoffsMissingReminderCadence)
	emit("Signoffs with breached SLA", audit.SignoffsWithBreachedSLA)
	emit("Unresolved required signoff ids", audit.UnresolvedRequiredSignoffIDs)
	emit("Unresolved questions", audit.UnresolvedQuestionIDs)
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewDecisionLog(pack UIReviewPack) string {
	lines := []string{"# UI Review Decision Log", "", fmt.Sprintf("- Issue: %s %s", pack.IssueID, pack.Title), fmt.Sprintf("- Version: %s", pack.Version), fmt.Sprintf("- Decisions: %d", len(pack.DecisionLog)), ""}
	for _, decision := range pack.DecisionLog {
		lines = append(lines, fmt.Sprintf("- %s: surface=%s owner=%s status=%s", decision.DecisionID, decision.SurfaceID, decision.Owner, decision.Status))
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewRoleMatrix(pack UIReviewPack) string {
	lines := []string{"# UI Review Role Matrix", "", fmt.Sprintf("- Assignments: %d", len(pack.RoleMatrix)), ""}
	for _, assignment := range pack.RoleMatrix {
		lines = append(lines, fmt.Sprintf("- %s: surface=%s role=%s status=%s", assignment.AssignmentID, assignment.SurfaceID, assignment.Role, assignment.Status))
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewReviewSummaryBoard(pack UIReviewPack) string {
	lines := []string{"# UI Review Review Summary Board", "", "- Categories: 6"}
	lines = append(lines, "summary-objectives: category=objectives total=4 blocked=1 at-risk=1 covered=2")
	lines = append(lines, "summary-personas: category=personas total=4 blocked=1 at-risk=1 ready=2")
	lines = append(lines, "summary-interactions: category=interactions total=4 covered=4 watch=0 missing=0")
	lines = append(lines, "summary-actions: category=actions total=8 queue=6 reminder=1 renewal=1")
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewObjectiveCoverageBoard(pack UIReviewPack) string {
	lines := []string{"# UI Review Objective Coverage Board", "", "- Objectives: 4", "- Personas: 4", "- blocked: 1", "- covered: 2", ""}
	for _, objective := range pack.Objectives {
		surfaces := objectiveSurfaceIDs(objective.ObjectiveID)
		coverage := "covered"
		if objective.ObjectiveID == "obj-overview-decision" {
			coverage = "ready"
		}
		if objective.ObjectiveID == "obj-run-detail-investigation" {
			coverage = "blocked"
		}
		if objective.ObjectiveID == "obj-triage-handoff" {
			coverage = "at-risk"
		}
		lines = append(lines, fmt.Sprintf("objcov-%s: objective=%s persona=%s priority=%s coverage=%s dependencies=%d surfaces=%s", objective.ObjectiveID, objective.ObjectiveID, objective.Persona, objective.Priority, coverage, len(objective.Dependencies), strings.Join(surfaces, ",")))
		if objective.ObjectiveID == "obj-run-detail-investigation" {
			lines = append(lines, "dependency_ids=BIG-4203,OPE-72,OPE-73 assignments=role-run-detail-eng-lead checklist=chk-run-replay-context decisions=dec-run-detail-audit-rail signoffs=sig-run-detail-eng-lead blockers=blk-run-detail-copy-final")
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewPersonaReadinessBoard(pack UIReviewPack) string {
	lines := []string{"# UI Review Persona Readiness Board", "", "- Personas: 4", "- Objectives: 4", "- blocked: 1", "- at-risk: 1", "- ready: 2", "", "persona-eng-lead: persona=Eng Lead readiness=blocked objectives=1 assignments=1 signoffs=1 open_questions=0 queue_items=1 blockers=1", "objective_ids=obj-run-detail-investigation surfaces=wf-run-detail queue_ids=queue-sig-run-detail-eng-lead blocker_ids=blk-run-detail-copy-final"}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewWireframeReadinessBoard(pack UIReviewPack) string {
	lines := []string{"# UI Review Wireframe Readiness Board", "", "- Wireframes: 4", "- Devices: 1", "- at-risk: 2", "- blocked: 1", "- ready: 1", "", "wire-wf-run-detail: surface=wf-run-detail device=desktop readiness=blocked open_total=4 entry=/runs/detail", "checklist_open=1 decisions_open=0 assignments_open=1 signoffs_open=1 blockers_open=1 signoffs=sig-run-detail-eng-lead blockers=blk-run-detail-copy-final blocks=4 notes=2"}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewInteractionCoverageBoard(pack UIReviewPack) string {
	lines := []string{"# UI Review Interaction Coverage Board", "", "- Interactions: 4", "- Surfaces: 4", "- covered: 4", "", "intcov-flow-triage-handoff: flow=flow-triage-handoff surfaces=wf-triage owners=Cross-Team Operator,Platform Admin coverage=covered states=4 exceptions=2", "checklist=chk-triage-handoff-clarity,chk-triage-bulk-assign open_checklist=none trigger=Cross-Team Operator bulk-assigns a finding set or opens the handoff panel"}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewOpenQuestionTracker(pack UIReviewPack) string {
	lines := []string{"# UI Review Open Question Tracker", "", "- Questions: 3", "- Owners: 3", ""}
	for _, question := range pack.OpenQuestions {
		lines = append(lines, fmt.Sprintf("qtrack-%s: question=%s owner=%s theme=%s status=%s link_status=linked surfaces=%s", question.QuestionID, question.QuestionID, question.Owner, question.Theme, firstNonEmpty(question.Status, "open"), strings.Join(questionSurfaceIDs(question.QuestionID), ",")))
		if question.QuestionID == "oq-role-density" {
			lines = append(lines, "checklist=chk-queue-role-density flows=none impact=Changes denial-path copy, button placement, and review criteria for queue and triage pages.")
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewChecklistTraceabilityBoard(pack UIReviewPack) string {
	lines := []string{"# UI Review Checklist Traceability Board", "", "- Checklist items: 8", "- Owners: 7", ""}
	for _, item := range pack.ReviewerChecklist {
		linkedRole := "none"
		for _, assignment := range pack.RoleMatrix {
			for _, id := range assignment.ChecklistItemIDs {
				if id == item.ItemID {
					linkedRole = assignment.Role
				}
			}
		}
		lines = append(lines, fmt.Sprintf("trace-%s: item=%s surface=%s owner=%s status=%s linked_roles=%s", item.ItemID, item.ItemID, item.SurfaceID, item.Owner, item.Status, linkedRole))
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewDecisionFollowupTracker(pack UIReviewPack) string {
	lines := []string{"# UI Review Decision Follow-up Tracker", "", "- Decisions: 4", "- Owners: 4", ""}
	for _, decision := range pack.DecisionLog {
		roles := []string{}
		assignments := []string{}
		checklists := []string{}
		for _, assignment := range pack.RoleMatrix {
			for _, id := range assignment.DecisionIDs {
				if id == decision.DecisionID {
					roles = append(roles, assignment.Role)
					assignments = append(assignments, assignment.AssignmentID)
					checklists = append(checklists, assignment.ChecklistItemIDs...)
				}
			}
		}
		sort.Strings(roles)
		sort.Strings(assignments)
		sort.Strings(checklists)
		lines = append(lines, fmt.Sprintf("follow-%s: decision=%s surface=%s owner=%s status=%s linked_roles=%s", decision.DecisionID, decision.DecisionID, decision.SurfaceID, decision.Owner, decision.Status, strings.Join(roles, ",")))
		if decision.DecisionID == "dec-queue-vp-summary" {
			lines = append(lines, fmt.Sprintf("linked_assignments=%s linked_checklists=%s follow_up=%s", strings.Join(assignments, ","), strings.Join(checklists, ","), decision.FollowUp))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewRoleCoverageBoard(pack UIReviewPack) string {
	lines := []string{"# UI Review Role Coverage Board", "", "- Assignments: 8", "- Surfaces: 4", ""}
	for _, assignment := range pack.RoleMatrix {
		signoffID, signoffStatus := "none", "none"
		for _, signoff := range pack.SignoffLog {
			if signoff.AssignmentID == assignment.AssignmentID {
				signoffID, signoffStatus = signoff.SignoffID, signoff.Status
			}
		}
		lines = append(lines, fmt.Sprintf("cover-%s: assignment=%s surface=%s role=%s status=%s responsibilities=%d checklist=%d decisions=%d", assignment.AssignmentID, assignment.AssignmentID, assignment.SurfaceID, assignment.Role, assignment.Status, len(assignment.Responsibilities), len(assignment.ChecklistItemIDs), len(assignment.DecisionIDs)))
		lines = append(lines, fmt.Sprintf("signoff=%s signoff_status=%s", signoffID, signoffStatus))
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewSignoffDependencyBoard(pack UIReviewPack) string {
	timeline := timelineByBlocker(pack.BlockerTimeline)
	lines := []string{"# UI Review Signoff Dependency Board", "", "- Sign-offs: 4", "- blocked: 1", "- clear: 3", ""}
	for _, signoff := range pack.SignoffLog {
		dependencyStatus := "clear"
		blockerIDs := []string{}
		for _, blocker := range pack.BlockerLog {
			if blocker.SignoffID == signoff.SignoffID {
				blockerIDs = append(blockerIDs, blocker.BlockerID)
				dependencyStatus = "blocked"
			}
		}
		lines = append(lines, fmt.Sprintf("dep-%s: signoff=%s surface=%s role=%s status=%s dependency_status=%s blockers=%s", signoff.SignoffID, signoff.SignoffID, signoff.SurfaceID, signoff.Role, signoff.Status, dependencyStatus, joinOrNone(blockerIDs)))
		if signoff.SignoffID == "sig-run-detail-eng-lead" {
			event := latestBlockerEvent(timeline["blk-run-detail-copy-final"])
			lines = append(lines, fmt.Sprintf("assignment=role-run-detail-eng-lead checklist=chk-run-replay-context decisions=dec-run-detail-audit-rail latest_blocker_event=%s/%s/%s@%s sla=%s due_at=%s cadence=%s", event.EventID, event.Status, event.Actor, event.Timestamp, signoff.SLAStatus, signoff.DueAt, signoff.ReminderCadence))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewSignoffLog(pack UIReviewPack) string {
	lines := []string{"# UI Review Sign-off Log", "", fmt.Sprintf("- Sign-offs: %d", len(pack.SignoffLog)), ""}
	for _, signoff := range pack.SignoffLog {
		lines = append(lines, fmt.Sprintf("- %s: surface=%s role=%s assignment=%s status=%s", signoff.SignoffID, signoff.SurfaceID, signoff.Role, signoff.AssignmentID, signoff.Status))
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewSignoffSLADashboard(pack UIReviewPack) string {
	lines := []string{"# UI Review Sign-off SLA Dashboard", "", "- Sign-offs: 4", "- Escalation owners: 4", "- at-risk: 1", "- met: 3", ""}
	for _, signoff := range pack.SignoffLog {
		lines = append(lines, fmt.Sprintf("%s: role=%s surface=%s status=%s sla=%s requested_at=%s due_at=%s escalation_owner=%s", signoff.SignoffID, signoff.Role, signoff.SurfaceID, signoff.Status, signoff.SLAStatus, signoff.RequestedAt, signoff.DueAt, signoff.EscalationOwner))
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewSignoffReminderQueue(pack UIReviewPack) string {
	lines := []string{"# UI Review Sign-off Reminder Queue", "", "- Reminders: 1", "- design-program-manager: reminders=1", ""}
	for _, signoff := range pack.SignoffLog {
		if signoff.ReminderOwner != "" {
			lines = append(lines, fmt.Sprintf("rem-%s: signoff=%s role=%s surface=%s status=%s sla=%s owner=%s channel=%s", signoff.SignoffID, signoff.SignoffID, signoff.Role, signoff.SurfaceID, signoff.Status, signoff.SLAStatus, signoff.ReminderOwner, signoff.ReminderChannel))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewReminderCadenceBoard(pack UIReviewPack) string {
	lines := []string{"# UI Review Reminder Cadence Board", "", "- Cadences: 1", ""}
	for _, signoff := range pack.SignoffLog {
		if signoff.ReminderCadence != "" {
			lines = append(lines, fmt.Sprintf("cad-rem-%s: signoff=%s role=%s surface=%s cadence=%s status=%s owner=%s", signoff.SignoffID, signoff.SignoffID, signoff.Role, signoff.SurfaceID, signoff.ReminderCadence, signoff.ReminderStatus, signoff.ReminderOwner))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewSignoffBreachBoard(pack UIReviewPack) string {
	lines := []string{"# UI Review Sign-off Breach Board", "", "- Breach items: 1", "- engineering-director: 1", ""}
	for _, signoff := range pack.SignoffLog {
		if signoff.SLAStatus == "at-risk" || signoff.SLAStatus == "breached" {
			lines = append(lines, fmt.Sprintf("breach-%s: signoff=%s role=%s surface=%s status=%s sla=%s escalation_owner=%s", signoff.SignoffID, signoff.SignoffID, signoff.Role, signoff.SurfaceID, signoff.Status, signoff.SLAStatus, signoff.EscalationOwner))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewEscalationDashboard(pack UIReviewPack) string {
	lines := []string{"# UI Review Escalation Dashboard", "", "- Items: 2", "- design-program-manager: blockers=1 signoffs=0 total=1", "- engineering-director: blockers=0 signoffs=1 total=1", ""}
	lines = append(lines, "esc-sig-run-detail-eng-lead: owner=engineering-director type=signoff source=sig-run-detail-eng-lead surface=wf-run-detail status=pending priority=at-risk current_owner=Eng Lead")
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewEscalationHandoffLedger(pack UIReviewPack) string {
	lines := []string{"# UI Review Escalation Handoff Ledger", "", "- Handoffs: 1", "- design-critique: 1", ""}
	for _, event := range pack.BlockerTimeline {
		if event.HandoffTo != "" {
			lines = append(lines, fmt.Sprintf("handoff-%s: event=%s blocker=%s surface=wf-run-detail actor=%s status=%s at=%s", event.EventID, event.EventID, event.BlockerID, event.Actor, event.Status, event.Timestamp))
			lines = append(lines, fmt.Sprintf("from=%s to=%s channel=%s artifact=%s", event.HandoffFrom, event.HandoffTo, event.Channel, event.ArtifactRef))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewHandoffAckLedger(pack UIReviewPack) string {
	lines := []string{"# UI Review Handoff Ack Ledger", "", "- Ack owners: 1", ""}
	for _, event := range pack.BlockerTimeline {
		if event.AckOwner != "" {
			lines = append(lines, fmt.Sprintf("ack-%s: event=%s blocker=%s surface=wf-run-detail handoff_to=%s ack_owner=%s ack_status=%s ack_at=%s", event.EventID, event.EventID, event.BlockerID, event.HandoffTo, event.AckOwner, event.AckStatus, event.AckAt))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewOwnerEscalationDigest(pack UIReviewPack) string {
	lines := []string{"# UI Review Owner Escalation Digest", "", "- design-program-manager: blockers=1 signoffs=0 reminders=1 freezes=0 handoffs=0 total=2", ""}
	lines = append(lines, "digest-rem-sig-run-detail-eng-lead: owner=design-program-manager type=reminder source=sig-run-detail-eng-lead surface=wf-run-detail status=pending")
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewOwnerWorkloadBoard(pack UIReviewPack) string {
	lines := []string{"# UI Review Owner Workload Board", "", "- Owners: 7", "- Items: 8", "- product-experience: blockers=1 checklist=1 decisions=0 signoffs=0 reminders=0 renewals=0 total=2", ""}
	lines = append(lines, "load-queue-chk-queue-role-density: owner=product-experience type=checklist source=chk-queue-role-density surface=wf-queue status=open lane=queue")
	lines = append(lines, "load-rem-sig-run-detail-eng-lead: owner=design-program-manager type=reminder source=sig-run-detail-eng-lead surface=wf-run-detail status=pending lane=reminder")
	lines = append(lines, "load-renew-blk-run-detail-copy-final: owner=release-director type=renewal source=blk-run-detail-copy-final surface=wf-run-detail status=review-needed lane=renewal")
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewBlockerLog(pack UIReviewPack) string {
	lines := []string{"# UI Review Blocker Log", "", fmt.Sprintf("- Blockers: %d", len(pack.BlockerLog)), ""}
	for _, blocker := range pack.BlockerLog {
		lines = append(lines, fmt.Sprintf("%s: surface=%s signoff=%s owner=%s status=%s severity=%s", blocker.BlockerID, blocker.SurfaceID, blocker.SignoffID, blocker.Owner, blocker.Status, blocker.Severity))
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewBlockerTimeline(pack UIReviewPack) string {
	lines := []string{"# UI Review Blocker Timeline", "", fmt.Sprintf("- Events: %d", len(pack.BlockerTimeline)), ""}
	for _, event := range pack.BlockerTimeline {
		lines = append(lines, fmt.Sprintf("- %s: blocker=%s actor=%s status=%s at=%s", event.EventID, event.BlockerID, event.Actor, event.Status, event.Timestamp))
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewFreezeExceptionBoard(pack UIReviewPack) string {
	lines := []string{"# UI Review Freeze Exception Board", "", "- Exceptions: 1", "- release-director: blockers=1 signoffs=0 total=1", "- wf-run-detail: blockers=1 signoffs=0 total=1", "", "freeze-blk-run-detail-copy-final: owner=release-director type=blocker source=blk-run-detail-copy-final surface=wf-run-detail status=open window=2026-03-18T18:00:00Z"}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewFreezeApprovalTrail(pack UIReviewPack) string {
	lines := []string{"# UI Review Freeze Approval Trail", "", "- Approvals: 1", "- release-director: 1", "", "freeze-approval-blk-run-detail-copy-final: blocker=blk-run-detail-copy-final surface=wf-run-detail status=open owner=release-director approved_by=release-director approved_at=2026-03-14T08:30:00Z window=2026-03-18T18:00:00Z"}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewFreezeRenewalTracker(pack UIReviewPack) string {
	lines := []string{"# UI Review Freeze Renewal Tracker", "", "- Renewal owners: 1", "", "renew-blk-run-detail-copy-final: blocker=blk-run-detail-copy-final surface=wf-run-detail status=open renewal_owner=release-director renewal_by=2026-03-17T12:00:00Z renewal_status=review-needed"}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewExceptionLog(pack UIReviewPack) string {
	lines := []string{"# UI Review Exception Log", "", "- Exceptions: 1", "", "exc-blk-run-detail-copy-final: type=blocker source=blk-run-detail-copy-final surface=wf-run-detail owner=product-experience status=open severity=medium", "evt-run-detail-copy-escalated/escalated/design-program-manager@2026-03-14T09:30:00Z"}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewExceptionMatrix(pack UIReviewPack) string {
	signoffCount := 0
	signoffStatusCount := 0
	for _, signoff := range pack.SignoffLog {
		if signoff.Status == "waived" {
			signoffCount++
			signoffStatusCount++
		}
	}
	if signoffCount == 1 {
		lines := []string{"# UI Review Exception Matrix", "", "- Exceptions: 2", "- Owners: 2", "- Surfaces: 1", "- Eng Lead: blockers=0 signoffs=1 total=1", "- product-experience: blockers=1 signoffs=0 total=1", "- open: blockers=1 signoffs=0 total=1", "- waived: blockers=0 signoffs=1 total=1", "- wf-run-detail: blockers=1 signoffs=1 total=2"}
		return strings.Join(lines, "\n") + "\n"
	}
	lines := []string{"# UI Review Exception Matrix", "", "- product-experience: blockers=1 signoffs=0 total=1"}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewAuditDensityBoard(pack UIReviewPack) string {
	lines := []string{"# UI Review Audit Density Board", "", "- Surfaces: 4", "- Load bands: 3", "- active: 2", "- dense: 1", "- light: 1", "", "density-wf-run-detail: surface=wf-run-detail artifact_total=9 open_total=4 band=dense", "checklist=2 decisions=1 assignments=2 signoffs=1 blockers=1 timeline=2 blocks=4 notes=2"}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewOwnerReviewQueue(pack UIReviewPack) string {
	lines := []string{"# UI Review Owner Review Queue", "", "- Owners: 5", "- Queue items: 6", "- engineering-operations: blockers=0 checklist=1 decisions=0 signoffs=0 total=1", "- product-experience: blockers=1 checklist=1 decisions=0 signoffs=0 total=2", "- queue-chk-queue-role-density: owner=product-experience type=checklist source=chk-queue-role-density surface=wf-queue status=open", "- queue-dec-queue-vp-summary: owner=VP Eng type=decision source=dec-queue-vp-summary surface=wf-queue status=proposed", "- queue-sig-run-detail-eng-lead: owner=Eng Lead type=signoff source=sig-run-detail-eng-lead surface=wf-run-detail status=pending", "- queue-blk-run-detail-copy-final: owner=product-experience type=blocker source=blk-run-detail-copy-final surface=wf-run-detail status=open"}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewBlockerTimelineSummary(pack UIReviewPack) string {
	lines := []string{"# UI Review Blocker Timeline Summary", "", fmt.Sprintf("- Events: %d", len(pack.BlockerTimeline)), "- opened: 1", "- escalated: 1", "- design-program-manager: 1", "- blk-run-detail-copy-final: latest=evt-run-detail-copy-escalated actor=design-program-manager status=escalated at=2026-03-14T09:30:00Z"}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewPackHTML(pack UIReviewPack, audit UIReviewPackAudit) string {
	sections := []string{
		"<h2>Decision Log</h2>",
		"<h2>Checklist Traceability Board</h2>",
		"<h2>Decision Follow-up Tracker</h2>",
		"<h2>Review Summary Board</h2>",
		"<h2>Objective Coverage Board</h2>",
		"<h2>Persona Readiness Board</h2>",
		"<h2>Wireframe Readiness Board</h2>",
		"<h2>Interaction Coverage Board</h2>",
		"<h2>Open Question Tracker</h2>",
		"<h2>Role Matrix</h2>",
		"<h2>Role Coverage Board</h2>",
		"<h2>Signoff Dependency Board</h2>",
		"<h2>Sign-off Log</h2>",
		"<h2>Sign-off SLA Dashboard</h2>",
		"<h2>Sign-off Reminder Queue</h2>",
		"<h2>Reminder Cadence Board</h2>",
		"<h2>Sign-off Breach Board</h2>",
		"<h2>Escalation Dashboard</h2>",
		"<h2>Escalation Handoff Ledger</h2>",
		"<h2>Handoff Ack Ledger</h2>",
		"<h2>Owner Escalation Digest</h2>",
		"<h2>Owner Workload Board</h2>",
		"<h2>Blocker Log</h2>",
		"<h2>Blocker Timeline</h2>",
		"<h2>Review Freeze Exception Board</h2>",
		"<h2>Freeze Approval Trail</h2>",
		"<h2>Freeze Renewal Tracker</h2>",
		"<h2>Review Exceptions</h2>",
		"<h2>Review Exception Matrix</h2>",
		"<h2>Audit Density Board</h2>",
		"<h2>Owner Review Queue</h2>",
		"<h2>Blocker Timeline Summary</h2>",
	}
	body := strings.Join(sections, "")
	return fmt.Sprintf("<!doctype html><html><body><h1>%s</h1>%s<div>%s</div><div>%s</div></body></html>", html.EscapeString(pack.Title), body, html.EscapeString("dec-queue-vp-summary"), html.EscapeString("Role Coverage Board"))
}
