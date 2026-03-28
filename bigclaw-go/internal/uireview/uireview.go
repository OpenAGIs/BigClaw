package uireview

import (
	"fmt"
	"sort"
	"strings"
)

type ReviewObjective struct {
	ObjectiveID   string   `json:"objective_id"`
	Title         string   `json:"title"`
	Persona       string   `json:"persona"`
	Outcome       string   `json:"outcome"`
	SuccessSignal string   `json:"success_signal"`
	Priority      string   `json:"priority"`
	Dependencies  []string `json:"dependencies"`
}

type WireframeSurface struct {
	SurfaceID     string   `json:"surface_id"`
	Name          string   `json:"name"`
	Device        string   `json:"device"`
	EntryPoint    string   `json:"entry_point"`
	PrimaryBlocks []string `json:"primary_blocks"`
	ReviewNotes   []string `json:"review_notes"`
}

type InteractionFlow struct {
	FlowID         string   `json:"flow_id"`
	Name           string   `json:"name"`
	Trigger        string   `json:"trigger"`
	SystemResponse string   `json:"system_response"`
	States         []string `json:"states"`
	Exceptions     []string `json:"exceptions"`
}

type OpenQuestion struct {
	QuestionID string `json:"question_id"`
	Theme      string `json:"theme"`
	Question   string `json:"question"`
	Owner      string `json:"owner"`
	Impact     string `json:"impact"`
	Status     string `json:"status"`
}

type ReviewerChecklistItem struct {
	ItemID        string   `json:"item_id"`
	SurfaceID     string   `json:"surface_id"`
	Prompt        string   `json:"prompt"`
	Owner         string   `json:"owner"`
	Status        string   `json:"status"`
	EvidenceLinks []string `json:"evidence_links"`
	Notes         string   `json:"notes"`
}

type ReviewDecision struct {
	DecisionID string `json:"decision_id"`
	SurfaceID  string `json:"surface_id"`
	Owner      string `json:"owner"`
	Summary    string `json:"summary"`
	Rationale  string `json:"rationale"`
	Status     string `json:"status"`
	FollowUp   string `json:"follow_up"`
}

type ReviewRoleAssignment struct {
	AssignmentID     string   `json:"assignment_id"`
	SurfaceID        string   `json:"surface_id"`
	Role             string   `json:"role"`
	Responsibilities []string `json:"responsibilities"`
	ChecklistItemIDs []string `json:"checklist_item_ids"`
	DecisionIDs      []string `json:"decision_ids"`
	Status           string   `json:"status"`
}

type ReviewSignoff struct {
	SignoffID       string   `json:"signoff_id"`
	AssignmentID    string   `json:"assignment_id"`
	SurfaceID       string   `json:"surface_id"`
	Role            string   `json:"role"`
	Status          string   `json:"status"`
	Required        bool     `json:"required"`
	EvidenceLinks   []string `json:"evidence_links"`
	Notes           string   `json:"notes"`
	WaiverOwner     string   `json:"waiver_owner"`
	WaiverReason    string   `json:"waiver_reason"`
	RequestedAt     string   `json:"requested_at"`
	DueAt           string   `json:"due_at"`
	EscalationOwner string   `json:"escalation_owner"`
	SLAStatus       string   `json:"sla_status"`
	ReminderOwner   string   `json:"reminder_owner"`
	ReminderChannel string   `json:"reminder_channel"`
	LastReminderAt  string   `json:"last_reminder_at"`
	NextReminderAt  string   `json:"next_reminder_at"`
	ReminderCadence string   `json:"reminder_cadence"`
	ReminderStatus  string   `json:"reminder_status"`
}

type ReviewBlocker struct {
	BlockerID           string `json:"blocker_id"`
	SurfaceID           string `json:"surface_id"`
	SignoffID           string `json:"signoff_id"`
	Owner               string `json:"owner"`
	Summary             string `json:"summary"`
	Status              string `json:"status"`
	Severity            string `json:"severity"`
	EscalationOwner     string `json:"escalation_owner"`
	NextAction          string `json:"next_action"`
	FreezeException     bool   `json:"freeze_exception"`
	FreezeOwner         string `json:"freeze_owner"`
	FreezeUntil         string `json:"freeze_until"`
	FreezeReason        string `json:"freeze_reason"`
	FreezeApprovedBy    string `json:"freeze_approved_by"`
	FreezeApprovedAt    string `json:"freeze_approved_at"`
	FreezeRenewalOwner  string `json:"freeze_renewal_owner"`
	FreezeRenewalBy     string `json:"freeze_renewal_by"`
	FreezeRenewalStatus string `json:"freeze_renewal_status"`
}

type ReviewBlockerEvent struct {
	EventID     string `json:"event_id"`
	BlockerID   string `json:"blocker_id"`
	Actor       string `json:"actor"`
	Status      string `json:"status"`
	Summary     string `json:"summary"`
	Timestamp   string `json:"timestamp"`
	NextAction  string `json:"next_action"`
	HandoffFrom string `json:"handoff_from"`
	HandoffTo   string `json:"handoff_to"`
	Channel     string `json:"channel"`
	ArtifactRef string `json:"artifact_ref"`
	AckOwner    string `json:"ack_owner"`
	AckAt       string `json:"ack_at"`
	AckStatus   string `json:"ack_status"`
}

type UIReviewPack struct {
	IssueID                   string                  `json:"issue_id"`
	Title                     string                  `json:"title"`
	Version                   string                  `json:"version"`
	Objectives                []ReviewObjective       `json:"objectives"`
	Wireframes                []WireframeSurface      `json:"wireframes"`
	Interactions              []InteractionFlow       `json:"interactions"`
	OpenQuestions             []OpenQuestion          `json:"open_questions"`
	ReviewerChecklist         []ReviewerChecklistItem `json:"reviewer_checklist"`
	RequiresReviewerChecklist bool                    `json:"requires_reviewer_checklist"`
	DecisionLog               []ReviewDecision        `json:"decision_log"`
	RequiresDecisionLog       bool                    `json:"requires_decision_log"`
	RoleMatrix                []ReviewRoleAssignment  `json:"role_matrix"`
	RequiresRoleMatrix        bool                    `json:"requires_role_matrix"`
	SignoffLog                []ReviewSignoff         `json:"signoff_log"`
	RequiresSignoffLog        bool                    `json:"requires_signoff_log"`
	BlockerLog                []ReviewBlocker         `json:"blocker_log"`
	RequiresBlockerLog        bool                    `json:"requires_blocker_log"`
	BlockerTimeline           []ReviewBlockerEvent    `json:"blocker_timeline"`
	RequiresBlockerTimeline   bool                    `json:"requires_blocker_timeline"`
}

func (p UIReviewPack) ensureDefaults() UIReviewPack {
	for idx := range p.Objectives {
		if strings.TrimSpace(p.Objectives[idx].Priority) == "" {
			p.Objectives[idx].Priority = "P1"
		}
	}
	for idx := range p.OpenQuestions {
		if strings.TrimSpace(p.OpenQuestions[idx].Status) == "" {
			p.OpenQuestions[idx].Status = "open"
		}
	}
	for idx := range p.ReviewerChecklist {
		if strings.TrimSpace(p.ReviewerChecklist[idx].Status) == "" {
			p.ReviewerChecklist[idx].Status = "todo"
		}
	}
	for idx := range p.DecisionLog {
		if strings.TrimSpace(p.DecisionLog[idx].Status) == "" {
			p.DecisionLog[idx].Status = "proposed"
		}
	}
	for idx := range p.RoleMatrix {
		if strings.TrimSpace(p.RoleMatrix[idx].Status) == "" {
			p.RoleMatrix[idx].Status = "planned"
		}
	}
	for idx := range p.SignoffLog {
		if strings.TrimSpace(p.SignoffLog[idx].Status) == "" {
			p.SignoffLog[idx].Status = "pending"
		}
		if strings.TrimSpace(p.SignoffLog[idx].SLAStatus) == "" {
			p.SignoffLog[idx].SLAStatus = "on-track"
		}
		if strings.TrimSpace(p.SignoffLog[idx].ReminderStatus) == "" {
			p.SignoffLog[idx].ReminderStatus = "scheduled"
		}
	}
	for idx := range p.BlockerLog {
		if strings.TrimSpace(p.BlockerLog[idx].Status) == "" {
			p.BlockerLog[idx].Status = "open"
		}
		if strings.TrimSpace(p.BlockerLog[idx].Severity) == "" {
			p.BlockerLog[idx].Severity = "medium"
		}
		if strings.TrimSpace(p.BlockerLog[idx].FreezeRenewalStatus) == "" {
			p.BlockerLog[idx].FreezeRenewalStatus = "not-needed"
		}
	}
	for idx := range p.BlockerTimeline {
		if strings.TrimSpace(p.BlockerTimeline[idx].AckStatus) == "" {
			p.BlockerTimeline[idx].AckStatus = "pending"
		}
	}
	return p
}

type UIReviewPackAudit struct {
	Ready                                     bool
	ObjectiveCount                            int
	WireframeCount                            int
	InteractionCount                          int
	OpenQuestionCount                         int
	ChecklistCount                            int
	DecisionCount                             int
	RoleAssignmentCount                       int
	SignoffCount                              int
	BlockerCount                              int
	BlockerTimelineCount                      int
	MissingSections                           []string
	ObjectivesMissingSignals                  []string
	WireframesMissingBlocks                   []string
	InteractionsMissingStates                 []string
	UnresolvedQuestionIDs                     []string
	WireframesMissingChecklists               []string
	OrphanChecklistSurfaces                   []string
	ChecklistItemsMissingEvidence             []string
	ChecklistItemsMissingRoleLinks            []string
	WireframesMissingDecisions                []string
	OrphanDecisionSurfaces                    []string
	UnresolvedDecisionIDs                     []string
	UnresolvedDecisionsMissingFollowUps       []string
	WireframesMissingRoleAssignments          []string
	OrphanRoleAssignmentSurfaces              []string
	RoleAssignmentsMissingResponsibilities    []string
	RoleAssignmentsMissingChecklistLinks      []string
	RoleAssignmentsMissingDecisionLinks       []string
	DecisionsMissingRoleLinks                 []string
	WireframesMissingSignoffs                 []string
	OrphanSignoffSurfaces                     []string
	SignoffsMissingAssignments                []string
	SignoffsMissingEvidence                   []string
	SignoffsMissingRequestedDates             []string
	SignoffsMissingDueDates                   []string
	SignoffsMissingEscalationOwners           []string
	SignoffsMissingReminderOwners             []string
	SignoffsMissingNextReminders              []string
	SignoffsMissingReminderCadence            []string
	SignoffsWithBreachedSLA                   []string
	WaivedSignoffsMissingMetadata             []string
	UnresolvedRequiredSignoffIDs              []string
	BlockersMissingSignoffLinks               []string
	BlockersMissingEscalationOwners           []string
	BlockersMissingNextActions                []string
	FreezeExceptionsMissingOwners             []string
	FreezeExceptionsMissingUntil              []string
	FreezeExceptionsMissingApprovers          []string
	FreezeExceptionsMissingApprovalDates      []string
	FreezeExceptionsMissingRenewalOwners      []string
	FreezeExceptionsMissingRenewalDates       []string
	BlockersMissingTimelineEvents             []string
	ClosedBlockersMissingResolutionEvents     []string
	OrphanBlockerSurfaces                     []string
	OrphanBlockerTimelineBlockerIDs           []string
	HandoffEventsMissingTargets               []string
	HandoffEventsMissingArtifacts             []string
	HandoffEventsMissingAckOwners             []string
	HandoffEventsMissingAckDates              []string
	UnresolvedRequiredSignoffsWithoutBlockers []string
}

func (a UIReviewPackAudit) Summary() string {
	status := "HOLD"
	if a.Ready {
		status = "READY"
	}
	return fmt.Sprintf(
		"%s: objectives=%d wireframes=%d interactions=%d open_questions=%d checklist=%d decisions=%d role_assignments=%d signoffs=%d blockers=%d timeline_events=%d",
		status,
		a.ObjectiveCount,
		a.WireframeCount,
		a.InteractionCount,
		a.OpenQuestionCount,
		a.ChecklistCount,
		a.DecisionCount,
		a.RoleAssignmentCount,
		a.SignoffCount,
		a.BlockerCount,
		a.BlockerTimelineCount,
	)
}

type Auditor struct{}

func (Auditor) Audit(pack UIReviewPack) UIReviewPackAudit {
	pack = pack.ensureDefaults()

	missingSections := []string{}
	if len(pack.Objectives) == 0 {
		missingSections = append(missingSections, "objectives")
	}
	if len(pack.Wireframes) == 0 {
		missingSections = append(missingSections, "wireframes")
	}
	if len(pack.Interactions) == 0 {
		missingSections = append(missingSections, "interactions")
	}
	if len(pack.OpenQuestions) == 0 {
		missingSections = append(missingSections, "open_questions")
	}

	objectivesMissingSignals := []string{}
	for _, objective := range pack.Objectives {
		if strings.TrimSpace(objective.SuccessSignal) == "" {
			objectivesMissingSignals = append(objectivesMissingSignals, objective.ObjectiveID)
		}
	}
	sort.Strings(objectivesMissingSignals)

	wireframesMissingBlocks := []string{}
	for _, wireframe := range pack.Wireframes {
		if len(wireframe.PrimaryBlocks) == 0 {
			wireframesMissingBlocks = append(wireframesMissingBlocks, wireframe.SurfaceID)
		}
	}
	sort.Strings(wireframesMissingBlocks)

	interactionsMissingStates := []string{}
	for _, flow := range pack.Interactions {
		if len(flow.States) == 0 {
			interactionsMissingStates = append(interactionsMissingStates, flow.FlowID)
		}
	}
	sort.Strings(interactionsMissingStates)

	unresolvedQuestionIDs := []string{}
	for _, q := range pack.OpenQuestions {
		if strings.ToLower(strings.TrimSpace(q.Status)) != "resolved" {
			unresolvedQuestionIDs = append(unresolvedQuestionIDs, q.QuestionID)
		}
	}
	sort.Strings(unresolvedQuestionIDs)

	wireframeSet := map[string]struct{}{}
	for _, wireframe := range pack.Wireframes {
		wireframeSet[wireframe.SurfaceID] = struct{}{}
	}

	checklistBySurface := map[string][]ReviewerChecklistItem{}
	for _, item := range pack.ReviewerChecklist {
		checklistBySurface[item.SurfaceID] = append(checklistBySurface[item.SurfaceID], item)
	}

	wireframesMissingChecklists := []string{}
	orphanChecklistSurfaces := []string{}
	checklistItemsMissingEvidence := []string{}
	checklistItemsMissingRoleLinks := []string{}
	if pack.RequiresReviewerChecklist {
		for _, wireframe := range pack.Wireframes {
			if len(checklistBySurface[wireframe.SurfaceID]) == 0 {
				wireframesMissingChecklists = append(wireframesMissingChecklists, wireframe.SurfaceID)
			}
		}
		for surfaceID := range checklistBySurface {
			if _, ok := wireframeSet[surfaceID]; !ok {
				orphanChecklistSurfaces = append(orphanChecklistSurfaces, surfaceID)
			}
		}
		for _, item := range pack.ReviewerChecklist {
			if len(item.EvidenceLinks) == 0 {
				checklistItemsMissingEvidence = append(checklistItemsMissingEvidence, item.ItemID)
			}
		}
	}
	sort.Strings(wireframesMissingChecklists)
	sort.Strings(orphanChecklistSurfaces)
	sort.Strings(checklistItemsMissingEvidence)

	decisionsBySurface := map[string][]ReviewDecision{}
	for _, decision := range pack.DecisionLog {
		decisionsBySurface[decision.SurfaceID] = append(decisionsBySurface[decision.SurfaceID], decision)
	}

	wireframesMissingDecisions := []string{}
	orphanDecisionSurfaces := []string{}
	unresolvedDecisionIDs := []string{}
	unresolvedDecisionsMissingFollowUps := []string{}
	if pack.RequiresDecisionLog {
		for _, wireframe := range pack.Wireframes {
			if len(decisionsBySurface[wireframe.SurfaceID]) == 0 {
				wireframesMissingDecisions = append(wireframesMissingDecisions, wireframe.SurfaceID)
			}
		}
		for surfaceID := range decisionsBySurface {
			if _, ok := wireframeSet[surfaceID]; !ok {
				orphanDecisionSurfaces = append(orphanDecisionSurfaces, surfaceID)
			}
		}
		for _, decision := range pack.DecisionLog {
			status := strings.ToLower(strings.TrimSpace(decision.Status))
			if status != "accepted" && status != "approved" && status != "resolved" && status != "waived" {
				unresolvedDecisionIDs = append(unresolvedDecisionIDs, decision.DecisionID)
				if strings.TrimSpace(decision.FollowUp) == "" {
					unresolvedDecisionsMissingFollowUps = append(unresolvedDecisionsMissingFollowUps, decision.DecisionID)
				}
			}
		}
	}
	sort.Strings(wireframesMissingDecisions)
	sort.Strings(orphanDecisionSurfaces)
	sort.Strings(unresolvedDecisionIDs)
	sort.Strings(unresolvedDecisionsMissingFollowUps)

	checklistIDSet := map[string]struct{}{}
	for _, item := range pack.ReviewerChecklist {
		checklistIDSet[item.ItemID] = struct{}{}
	}
	decisionIDSet := map[string]struct{}{}
	for _, decision := range pack.DecisionLog {
		decisionIDSet[decision.DecisionID] = struct{}{}
	}
	assignmentIDSet := map[string]struct{}{}
	for _, assignment := range pack.RoleMatrix {
		assignmentIDSet[assignment.AssignmentID] = struct{}{}
	}

	assignmentsBySurface := map[string][]ReviewRoleAssignment{}
	for _, assignment := range pack.RoleMatrix {
		assignmentsBySurface[assignment.SurfaceID] = append(assignmentsBySurface[assignment.SurfaceID], assignment)
	}
	wireframesMissingRoleAssignments := []string{}
	orphanRoleAssignmentSurfaces := []string{}
	roleAssignmentsMissingResponsibilities := []string{}
	roleAssignmentsMissingChecklistLinks := []string{}
	roleAssignmentsMissingDecisionLinks := []string{}
	decisionsMissingRoleLinks := []string{}
	if pack.RequiresRoleMatrix {
		for _, wireframe := range pack.Wireframes {
			if len(assignmentsBySurface[wireframe.SurfaceID]) == 0 {
				wireframesMissingRoleAssignments = append(wireframesMissingRoleAssignments, wireframe.SurfaceID)
			}
		}
		for surfaceID := range assignmentsBySurface {
			if _, ok := wireframeSet[surfaceID]; !ok {
				orphanRoleAssignmentSurfaces = append(orphanRoleAssignmentSurfaces, surfaceID)
			}
		}
		linkedChecklistSet := map[string]struct{}{}
		linkedDecisionSet := map[string]struct{}{}
		for _, assignment := range pack.RoleMatrix {
			if len(assignment.Responsibilities) == 0 {
				roleAssignmentsMissingResponsibilities = append(roleAssignmentsMissingResponsibilities, assignment.AssignmentID)
			}
			checklistLinked := len(assignment.ChecklistItemIDs) > 0
			for _, itemID := range assignment.ChecklistItemIDs {
				linkedChecklistSet[itemID] = struct{}{}
				if _, ok := checklistIDSet[itemID]; !ok {
					checklistLinked = false
				}
			}
			if !checklistLinked {
				roleAssignmentsMissingChecklistLinks = append(roleAssignmentsMissingChecklistLinks, assignment.AssignmentID)
			}
			decisionLinked := len(assignment.DecisionIDs) > 0
			for _, decisionID := range assignment.DecisionIDs {
				linkedDecisionSet[decisionID] = struct{}{}
				if _, ok := decisionIDSet[decisionID]; !ok {
					decisionLinked = false
				}
			}
			if !decisionLinked {
				roleAssignmentsMissingDecisionLinks = append(roleAssignmentsMissingDecisionLinks, assignment.AssignmentID)
			}
		}
		for _, item := range pack.ReviewerChecklist {
			if _, ok := linkedChecklistSet[item.ItemID]; !ok {
				checklistItemsMissingRoleLinks = append(checklistItemsMissingRoleLinks, item.ItemID)
			}
		}
		for _, decision := range pack.DecisionLog {
			if _, ok := linkedDecisionSet[decision.DecisionID]; !ok {
				decisionsMissingRoleLinks = append(decisionsMissingRoleLinks, decision.DecisionID)
			}
		}
	}
	sort.Strings(wireframesMissingRoleAssignments)
	sort.Strings(orphanRoleAssignmentSurfaces)
	sort.Strings(roleAssignmentsMissingResponsibilities)
	sort.Strings(roleAssignmentsMissingChecklistLinks)
	sort.Strings(roleAssignmentsMissingDecisionLinks)
	sort.Strings(checklistItemsMissingRoleLinks)
	sort.Strings(decisionsMissingRoleLinks)

	signoffsBySurface := map[string][]ReviewSignoff{}
	for _, signoff := range pack.SignoffLog {
		signoffsBySurface[signoff.SurfaceID] = append(signoffsBySurface[signoff.SurfaceID], signoff)
	}
	wireframesMissingSignoffs := []string{}
	orphanSignoffSurfaces := []string{}
	signoffsMissingAssignments := []string{}
	signoffsMissingEvidence := []string{}
	signoffsMissingRequestedDates := []string{}
	signoffsMissingDueDates := []string{}
	signoffsMissingEscalationOwners := []string{}
	signoffsMissingReminderOwners := []string{}
	signoffsMissingNextReminders := []string{}
	signoffsMissingReminderCadence := []string{}
	signoffsWithBreachedSLA := []string{}
	waivedSignoffsMissingMetadata := []string{}
	unresolvedRequiredSignoffIDs := []string{}
	if pack.RequiresSignoffLog {
		for _, wireframe := range pack.Wireframes {
			if len(signoffsBySurface[wireframe.SurfaceID]) == 0 {
				wireframesMissingSignoffs = append(wireframesMissingSignoffs, wireframe.SurfaceID)
			}
		}
		for surfaceID := range signoffsBySurface {
			if _, ok := wireframeSet[surfaceID]; !ok {
				orphanSignoffSurfaces = append(orphanSignoffSurfaces, surfaceID)
			}
		}
		unresolvedStatuses := map[string]struct{}{"approved": {}, "accepted": {}, "resolved": {}, "waived": {}, "deferred": {}}
		for _, signoff := range pack.SignoffLog {
			if _, ok := assignmentIDSet[signoff.AssignmentID]; !ok {
				signoffsMissingAssignments = append(signoffsMissingAssignments, signoff.SignoffID)
			}
			status := strings.ToLower(strings.TrimSpace(signoff.Status))
			if status != "waived" && len(signoff.EvidenceLinks) == 0 {
				signoffsMissingEvidence = append(signoffsMissingEvidence, signoff.SignoffID)
			}
			if signoff.Required && strings.TrimSpace(signoff.RequestedAt) == "" {
				signoffsMissingRequestedDates = append(signoffsMissingRequestedDates, signoff.SignoffID)
			}
			if signoff.Required && strings.TrimSpace(signoff.DueAt) == "" {
				signoffsMissingDueDates = append(signoffsMissingDueDates, signoff.SignoffID)
			}
			if signoff.Required && strings.TrimSpace(signoff.EscalationOwner) == "" {
				signoffsMissingEscalationOwners = append(signoffsMissingEscalationOwners, signoff.SignoffID)
			}
			if signoff.Required {
				if _, done := unresolvedStatuses[status]; !done {
					unresolvedRequiredSignoffIDs = append(unresolvedRequiredSignoffIDs, signoff.SignoffID)
					if strings.TrimSpace(signoff.ReminderOwner) == "" {
						signoffsMissingReminderOwners = append(signoffsMissingReminderOwners, signoff.SignoffID)
					}
					if strings.TrimSpace(signoff.NextReminderAt) == "" {
						signoffsMissingNextReminders = append(signoffsMissingNextReminders, signoff.SignoffID)
					}
					if strings.TrimSpace(signoff.ReminderCadence) == "" {
						signoffsMissingReminderCadence = append(signoffsMissingReminderCadence, signoff.SignoffID)
					}
				}
			}
			if strings.ToLower(strings.TrimSpace(signoff.SLAStatus)) == "breached" && status != "approved" && status != "accepted" && status != "resolved" {
				signoffsWithBreachedSLA = append(signoffsWithBreachedSLA, signoff.SignoffID)
			}
			if status == "waived" && (strings.TrimSpace(signoff.WaiverOwner) == "" || strings.TrimSpace(signoff.WaiverReason) == "") {
				waivedSignoffsMissingMetadata = append(waivedSignoffsMissingMetadata, signoff.SignoffID)
			}
		}
	}
	sort.Strings(wireframesMissingSignoffs)
	sort.Strings(orphanSignoffSurfaces)
	sort.Strings(signoffsMissingAssignments)
	sort.Strings(signoffsMissingEvidence)
	sort.Strings(signoffsMissingRequestedDates)
	sort.Strings(signoffsMissingDueDates)
	sort.Strings(signoffsMissingEscalationOwners)
	sort.Strings(signoffsMissingReminderOwners)
	sort.Strings(signoffsMissingNextReminders)
	sort.Strings(signoffsMissingReminderCadence)
	sort.Strings(signoffsWithBreachedSLA)
	sort.Strings(waivedSignoffsMissingMetadata)
	sort.Strings(unresolvedRequiredSignoffIDs)

	blockersBySignoff := map[string][]ReviewBlocker{}
	blockerSurfaceSet := map[string]struct{}{}
	for _, blocker := range pack.BlockerLog {
		blockersBySignoff[blocker.SignoffID] = append(blockersBySignoff[blocker.SignoffID], blocker)
		blockerSurfaceSet[blocker.SurfaceID] = struct{}{}
	}

	blockersMissingSignoffLinks := []string{}
	blockersMissingEscalationOwners := []string{}
	blockersMissingNextActions := []string{}
	freezeExceptionsMissingOwners := []string{}
	freezeExceptionsMissingUntil := []string{}
	freezeExceptionsMissingApprovers := []string{}
	freezeExceptionsMissingApprovalDates := []string{}
	freezeExceptionsMissingRenewalOwners := []string{}
	freezeExceptionsMissingRenewalDates := []string{}
	orphanBlockerSurfaces := []string{}
	unresolvedRequiredSignoffsWithoutBlockers := []string{}
	if pack.RequiresBlockerLog {
		signoffSet := map[string]struct{}{}
		for _, signoff := range pack.SignoffLog {
			signoffSet[signoff.SignoffID] = struct{}{}
		}
		for _, blocker := range pack.BlockerLog {
			if _, ok := signoffSet[blocker.SignoffID]; !ok {
				blockersMissingSignoffLinks = append(blockersMissingSignoffLinks, blocker.BlockerID)
			}
			if strings.TrimSpace(blocker.EscalationOwner) == "" {
				blockersMissingEscalationOwners = append(blockersMissingEscalationOwners, blocker.BlockerID)
			}
			if strings.TrimSpace(blocker.NextAction) == "" {
				blockersMissingNextActions = append(blockersMissingNextActions, blocker.BlockerID)
			}
			if blocker.FreezeException {
				if strings.TrimSpace(blocker.FreezeOwner) == "" {
					freezeExceptionsMissingOwners = append(freezeExceptionsMissingOwners, blocker.BlockerID)
				}
				if strings.TrimSpace(blocker.FreezeUntil) == "" {
					freezeExceptionsMissingUntil = append(freezeExceptionsMissingUntil, blocker.BlockerID)
				}
				if strings.TrimSpace(blocker.FreezeApprovedBy) == "" {
					freezeExceptionsMissingApprovers = append(freezeExceptionsMissingApprovers, blocker.BlockerID)
				}
				if strings.TrimSpace(blocker.FreezeApprovedAt) == "" {
					freezeExceptionsMissingApprovalDates = append(freezeExceptionsMissingApprovalDates, blocker.BlockerID)
				}
				if strings.TrimSpace(blocker.FreezeRenewalOwner) == "" {
					freezeExceptionsMissingRenewalOwners = append(freezeExceptionsMissingRenewalOwners, blocker.BlockerID)
				}
				if strings.TrimSpace(blocker.FreezeRenewalBy) == "" {
					freezeExceptionsMissingRenewalDates = append(freezeExceptionsMissingRenewalDates, blocker.BlockerID)
				}
			}
		}
		for surfaceID := range blockerSurfaceSet {
			if _, ok := wireframeSet[surfaceID]; !ok {
				orphanBlockerSurfaces = append(orphanBlockerSurfaces, surfaceID)
			}
		}
		for _, signoffID := range unresolvedRequiredSignoffIDs {
			if len(blockersBySignoff[signoffID]) == 0 {
				unresolvedRequiredSignoffsWithoutBlockers = append(unresolvedRequiredSignoffsWithoutBlockers, signoffID)
			}
		}
	}
	sort.Strings(blockersMissingSignoffLinks)
	sort.Strings(blockersMissingEscalationOwners)
	sort.Strings(blockersMissingNextActions)
	sort.Strings(freezeExceptionsMissingOwners)
	sort.Strings(freezeExceptionsMissingUntil)
	sort.Strings(freezeExceptionsMissingApprovers)
	sort.Strings(freezeExceptionsMissingApprovalDates)
	sort.Strings(freezeExceptionsMissingRenewalOwners)
	sort.Strings(freezeExceptionsMissingRenewalDates)
	sort.Strings(orphanBlockerSurfaces)
	sort.Strings(unresolvedRequiredSignoffsWithoutBlockers)

	timelineByBlocker := map[string][]ReviewBlockerEvent{}
	for _, event := range pack.BlockerTimeline {
		timelineByBlocker[event.BlockerID] = append(timelineByBlocker[event.BlockerID], event)
	}
	blockersMissingTimelineEvents := []string{}
	closedBlockersMissingResolutionEvents := []string{}
	orphanBlockerTimelineBlockerIDs := []string{}
	handoffEventsMissingTargets := []string{}
	handoffEventsMissingArtifacts := []string{}
	handoffEventsMissingAckOwners := []string{}
	handoffEventsMissingAckDates := []string{}
	if pack.RequiresBlockerTimeline {
		blockerIDSet := map[string]struct{}{}
		for _, blocker := range pack.BlockerLog {
			blockerIDSet[blocker.BlockerID] = struct{}{}
		}
		for blockerID := range timelineByBlocker {
			if _, ok := blockerIDSet[blockerID]; !ok {
				orphanBlockerTimelineBlockerIDs = append(orphanBlockerTimelineBlockerIDs, blockerID)
			}
		}
		for _, blocker := range pack.BlockerLog {
			status := strings.ToLower(strings.TrimSpace(blocker.Status))
			if status != "resolved" && status != "closed" {
				if len(timelineByBlocker[blocker.BlockerID]) == 0 {
					blockersMissingTimelineEvents = append(blockersMissingTimelineEvents, blocker.BlockerID)
				}
				continue
			}
			resolved := false
			for _, event := range timelineByBlocker[blocker.BlockerID] {
				es := strings.ToLower(strings.TrimSpace(event.Status))
				if es == "resolved" || es == "closed" {
					resolved = true
					break
				}
			}
			if !resolved {
				closedBlockersMissingResolutionEvents = append(closedBlockersMissingResolutionEvents, blocker.BlockerID)
			}
		}
		handoffStatuses := map[string]struct{}{"escalated": {}, "handoff": {}, "reassigned": {}}
		for _, event := range pack.BlockerTimeline {
			es := strings.ToLower(strings.TrimSpace(event.Status))
			_, isHandoff := handoffStatuses[es]
			if !isHandoff {
				continue
			}
			if strings.TrimSpace(event.HandoffTo) == "" {
				handoffEventsMissingTargets = append(handoffEventsMissingTargets, event.EventID)
			}
			if strings.TrimSpace(event.ArtifactRef) == "" {
				handoffEventsMissingArtifacts = append(handoffEventsMissingArtifacts, event.EventID)
			}
			if strings.TrimSpace(event.AckOwner) == "" {
				handoffEventsMissingAckOwners = append(handoffEventsMissingAckOwners, event.EventID)
			}
			if strings.TrimSpace(event.AckAt) == "" {
				handoffEventsMissingAckDates = append(handoffEventsMissingAckDates, event.EventID)
			}
		}
	}
	sort.Strings(blockersMissingTimelineEvents)
	sort.Strings(closedBlockersMissingResolutionEvents)
	sort.Strings(orphanBlockerTimelineBlockerIDs)
	sort.Strings(handoffEventsMissingTargets)
	sort.Strings(handoffEventsMissingArtifacts)
	sort.Strings(handoffEventsMissingAckOwners)
	sort.Strings(handoffEventsMissingAckDates)

	ready := len(missingSections) == 0 &&
		len(objectivesMissingSignals) == 0 &&
		len(wireframesMissingBlocks) == 0 &&
		len(interactionsMissingStates) == 0 &&
		len(wireframesMissingChecklists) == 0 &&
		len(orphanChecklistSurfaces) == 0 &&
		len(checklistItemsMissingEvidence) == 0 &&
		len(checklistItemsMissingRoleLinks) == 0 &&
		len(wireframesMissingDecisions) == 0 &&
		len(orphanDecisionSurfaces) == 0 &&
		len(unresolvedDecisionsMissingFollowUps) == 0 &&
		len(wireframesMissingRoleAssignments) == 0 &&
		len(orphanRoleAssignmentSurfaces) == 0 &&
		len(roleAssignmentsMissingResponsibilities) == 0 &&
		len(roleAssignmentsMissingChecklistLinks) == 0 &&
		len(roleAssignmentsMissingDecisionLinks) == 0 &&
		len(decisionsMissingRoleLinks) == 0 &&
		len(wireframesMissingSignoffs) == 0 &&
		len(orphanSignoffSurfaces) == 0 &&
		len(signoffsMissingAssignments) == 0 &&
		len(signoffsMissingEvidence) == 0 &&
		len(signoffsMissingRequestedDates) == 0 &&
		len(signoffsMissingDueDates) == 0 &&
		len(signoffsMissingEscalationOwners) == 0 &&
		len(signoffsMissingReminderOwners) == 0 &&
		len(signoffsMissingNextReminders) == 0 &&
		len(signoffsMissingReminderCadence) == 0 &&
		len(waivedSignoffsMissingMetadata) == 0 &&
		len(blockersMissingSignoffLinks) == 0 &&
		len(blockersMissingEscalationOwners) == 0 &&
		len(blockersMissingNextActions) == 0 &&
		len(freezeExceptionsMissingOwners) == 0 &&
		len(freezeExceptionsMissingUntil) == 0 &&
		len(freezeExceptionsMissingApprovers) == 0 &&
		len(freezeExceptionsMissingApprovalDates) == 0 &&
		len(freezeExceptionsMissingRenewalOwners) == 0 &&
		len(freezeExceptionsMissingRenewalDates) == 0 &&
		len(blockersMissingTimelineEvents) == 0 &&
		len(closedBlockersMissingResolutionEvents) == 0 &&
		len(orphanBlockerSurfaces) == 0 &&
		len(orphanBlockerTimelineBlockerIDs) == 0 &&
		len(handoffEventsMissingTargets) == 0 &&
		len(handoffEventsMissingArtifacts) == 0 &&
		len(handoffEventsMissingAckOwners) == 0 &&
		len(handoffEventsMissingAckDates) == 0 &&
		len(unresolvedRequiredSignoffsWithoutBlockers) == 0

	return UIReviewPackAudit{
		Ready:                                     ready,
		ObjectiveCount:                            len(pack.Objectives),
		WireframeCount:                            len(pack.Wireframes),
		InteractionCount:                          len(pack.Interactions),
		OpenQuestionCount:                         len(pack.OpenQuestions),
		ChecklistCount:                            len(pack.ReviewerChecklist),
		DecisionCount:                             len(pack.DecisionLog),
		RoleAssignmentCount:                       len(pack.RoleMatrix),
		SignoffCount:                              len(pack.SignoffLog),
		BlockerCount:                              len(pack.BlockerLog),
		BlockerTimelineCount:                      len(pack.BlockerTimeline),
		MissingSections:                           missingSections,
		ObjectivesMissingSignals:                  objectivesMissingSignals,
		WireframesMissingBlocks:                   wireframesMissingBlocks,
		InteractionsMissingStates:                 interactionsMissingStates,
		UnresolvedQuestionIDs:                     unresolvedQuestionIDs,
		WireframesMissingChecklists:               wireframesMissingChecklists,
		OrphanChecklistSurfaces:                   orphanChecklistSurfaces,
		ChecklistItemsMissingEvidence:             checklistItemsMissingEvidence,
		ChecklistItemsMissingRoleLinks:            checklistItemsMissingRoleLinks,
		WireframesMissingDecisions:                wireframesMissingDecisions,
		OrphanDecisionSurfaces:                    orphanDecisionSurfaces,
		UnresolvedDecisionIDs:                     unresolvedDecisionIDs,
		UnresolvedDecisionsMissingFollowUps:       unresolvedDecisionsMissingFollowUps,
		WireframesMissingRoleAssignments:          wireframesMissingRoleAssignments,
		OrphanRoleAssignmentSurfaces:              orphanRoleAssignmentSurfaces,
		RoleAssignmentsMissingResponsibilities:    roleAssignmentsMissingResponsibilities,
		RoleAssignmentsMissingChecklistLinks:      roleAssignmentsMissingChecklistLinks,
		RoleAssignmentsMissingDecisionLinks:       roleAssignmentsMissingDecisionLinks,
		DecisionsMissingRoleLinks:                 decisionsMissingRoleLinks,
		WireframesMissingSignoffs:                 wireframesMissingSignoffs,
		OrphanSignoffSurfaces:                     orphanSignoffSurfaces,
		SignoffsMissingAssignments:                signoffsMissingAssignments,
		SignoffsMissingEvidence:                   signoffsMissingEvidence,
		SignoffsMissingRequestedDates:             signoffsMissingRequestedDates,
		SignoffsMissingDueDates:                   signoffsMissingDueDates,
		SignoffsMissingEscalationOwners:           signoffsMissingEscalationOwners,
		SignoffsMissingReminderOwners:             signoffsMissingReminderOwners,
		SignoffsMissingNextReminders:              signoffsMissingNextReminders,
		SignoffsMissingReminderCadence:            signoffsMissingReminderCadence,
		SignoffsWithBreachedSLA:                   signoffsWithBreachedSLA,
		WaivedSignoffsMissingMetadata:             waivedSignoffsMissingMetadata,
		UnresolvedRequiredSignoffIDs:              unresolvedRequiredSignoffIDs,
		BlockersMissingSignoffLinks:               blockersMissingSignoffLinks,
		BlockersMissingEscalationOwners:           blockersMissingEscalationOwners,
		BlockersMissingNextActions:                blockersMissingNextActions,
		FreezeExceptionsMissingOwners:             freezeExceptionsMissingOwners,
		FreezeExceptionsMissingUntil:              freezeExceptionsMissingUntil,
		FreezeExceptionsMissingApprovers:          freezeExceptionsMissingApprovers,
		FreezeExceptionsMissingApprovalDates:      freezeExceptionsMissingApprovalDates,
		FreezeExceptionsMissingRenewalOwners:      freezeExceptionsMissingRenewalOwners,
		FreezeExceptionsMissingRenewalDates:       freezeExceptionsMissingRenewalDates,
		BlockersMissingTimelineEvents:             blockersMissingTimelineEvents,
		ClosedBlockersMissingResolutionEvents:     closedBlockersMissingResolutionEvents,
		OrphanBlockerSurfaces:                     orphanBlockerSurfaces,
		OrphanBlockerTimelineBlockerIDs:           orphanBlockerTimelineBlockerIDs,
		HandoffEventsMissingTargets:               handoffEventsMissingTargets,
		HandoffEventsMissingArtifacts:             handoffEventsMissingArtifacts,
		HandoffEventsMissingAckOwners:             handoffEventsMissingAckOwners,
		HandoffEventsMissingAckDates:              handoffEventsMissingAckDates,
		UnresolvedRequiredSignoffsWithoutBlockers: unresolvedRequiredSignoffsWithoutBlockers,
	}
}

func firstNonEmpty(v string, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
