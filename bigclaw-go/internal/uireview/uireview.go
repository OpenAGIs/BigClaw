package uireview

import (
	"encoding/json"
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
	Priority      string   `json:"priority,omitempty"`
	Dependencies  []string `json:"dependencies,omitempty"`
}

type WireframeSurface struct {
	SurfaceID     string   `json:"surface_id"`
	Name          string   `json:"name"`
	Device        string   `json:"device"`
	EntryPoint    string   `json:"entry_point"`
	PrimaryBlocks []string `json:"primary_blocks,omitempty"`
	ReviewNotes   []string `json:"review_notes,omitempty"`
}

type InteractionFlow struct {
	FlowID         string   `json:"flow_id"`
	Name           string   `json:"name"`
	Trigger        string   `json:"trigger"`
	SystemResponse string   `json:"system_response"`
	States         []string `json:"states,omitempty"`
	Exceptions     []string `json:"exceptions,omitempty"`
}

type OpenQuestion struct {
	QuestionID string `json:"question_id"`
	Theme      string `json:"theme"`
	Question   string `json:"question"`
	Owner      string `json:"owner"`
	Impact     string `json:"impact"`
	Status     string `json:"status,omitempty"`
}

type ReviewerChecklistItem struct {
	ItemID        string   `json:"item_id"`
	SurfaceID     string   `json:"surface_id"`
	Prompt        string   `json:"prompt"`
	Owner         string   `json:"owner"`
	Status        string   `json:"status,omitempty"`
	EvidenceLinks []string `json:"evidence_links,omitempty"`
	Notes         string   `json:"notes,omitempty"`
}

type ReviewDecision struct {
	DecisionID string `json:"decision_id"`
	SurfaceID  string `json:"surface_id"`
	Owner      string `json:"owner"`
	Summary    string `json:"summary"`
	Rationale  string `json:"rationale"`
	Status     string `json:"status,omitempty"`
	FollowUp   string `json:"follow_up,omitempty"`
}

type ReviewRoleAssignment struct {
	AssignmentID     string   `json:"assignment_id"`
	SurfaceID        string   `json:"surface_id"`
	Role             string   `json:"role"`
	Responsibilities []string `json:"responsibilities,omitempty"`
	ChecklistItemIDs []string `json:"checklist_item_ids,omitempty"`
	DecisionIDs      []string `json:"decision_ids,omitempty"`
	Status           string   `json:"status,omitempty"`
}

type ReviewSignoff struct {
	SignoffID       string   `json:"signoff_id"`
	AssignmentID    string   `json:"assignment_id"`
	SurfaceID       string   `json:"surface_id"`
	Role            string   `json:"role"`
	Status          string   `json:"status,omitempty"`
	Required        bool     `json:"required"`
	EvidenceLinks   []string `json:"evidence_links,omitempty"`
	Notes           string   `json:"notes,omitempty"`
	WaiverOwner     string   `json:"waiver_owner,omitempty"`
	WaiverReason    string   `json:"waiver_reason,omitempty"`
	RequestedAt     string   `json:"requested_at,omitempty"`
	DueAt           string   `json:"due_at,omitempty"`
	EscalationOwner string   `json:"escalation_owner,omitempty"`
	SLAStatus       string   `json:"sla_status,omitempty"`
	ReminderOwner   string   `json:"reminder_owner,omitempty"`
	ReminderChannel string   `json:"reminder_channel,omitempty"`
	LastReminderAt  string   `json:"last_reminder_at,omitempty"`
	NextReminderAt  string   `json:"next_reminder_at,omitempty"`
	ReminderCadence string   `json:"reminder_cadence,omitempty"`
	ReminderStatus  string   `json:"reminder_status,omitempty"`
}

type ReviewBlocker struct {
	BlockerID           string `json:"blocker_id"`
	SurfaceID           string `json:"surface_id"`
	SignoffID           string `json:"signoff_id"`
	Owner               string `json:"owner"`
	Summary             string `json:"summary"`
	Status              string `json:"status,omitempty"`
	Severity            string `json:"severity,omitempty"`
	EscalationOwner     string `json:"escalation_owner,omitempty"`
	NextAction          string `json:"next_action,omitempty"`
	FreezeException     bool   `json:"freeze_exception,omitempty"`
	FreezeOwner         string `json:"freeze_owner,omitempty"`
	FreezeUntil         string `json:"freeze_until,omitempty"`
	FreezeReason        string `json:"freeze_reason,omitempty"`
	FreezeApprovedBy    string `json:"freeze_approved_by,omitempty"`
	FreezeApprovedAt    string `json:"freeze_approved_at,omitempty"`
	FreezeRenewalOwner  string `json:"freeze_renewal_owner,omitempty"`
	FreezeRenewalBy     string `json:"freeze_renewal_by,omitempty"`
	FreezeRenewalStatus string `json:"freeze_renewal_status,omitempty"`
}

type ReviewBlockerEvent struct {
	EventID     string `json:"event_id"`
	BlockerID   string `json:"blocker_id"`
	Actor       string `json:"actor"`
	Status      string `json:"status"`
	Summary     string `json:"summary"`
	Timestamp   string `json:"timestamp"`
	NextAction  string `json:"next_action,omitempty"`
	HandoffFrom string `json:"handoff_from,omitempty"`
	HandoffTo   string `json:"handoff_to,omitempty"`
	Channel     string `json:"channel,omitempty"`
	ArtifactRef string `json:"artifact_ref,omitempty"`
	AckOwner    string `json:"ack_owner,omitempty"`
	AckAt       string `json:"ack_at,omitempty"`
	AckStatus   string `json:"ack_status,omitempty"`
}

type UIReviewPack struct {
	IssueID                   string                  `json:"issue_id"`
	Title                     string                  `json:"title"`
	Version                   string                  `json:"version"`
	Objectives                []ReviewObjective       `json:"objectives,omitempty"`
	Wireframes                []WireframeSurface      `json:"wireframes,omitempty"`
	Interactions              []InteractionFlow       `json:"interactions,omitempty"`
	OpenQuestions             []OpenQuestion          `json:"open_questions,omitempty"`
	ReviewerChecklist         []ReviewerChecklistItem `json:"reviewer_checklist,omitempty"`
	RequiresReviewerChecklist bool                    `json:"requires_reviewer_checklist,omitempty"`
	DecisionLog               []ReviewDecision        `json:"decision_log,omitempty"`
	RequiresDecisionLog       bool                    `json:"requires_decision_log,omitempty"`
	RoleMatrix                []ReviewRoleAssignment  `json:"role_matrix,omitempty"`
	RequiresRoleMatrix        bool                    `json:"requires_role_matrix,omitempty"`
	SignoffLog                []ReviewSignoff         `json:"signoff_log,omitempty"`
	RequiresSignoffLog        bool                    `json:"requires_signoff_log,omitempty"`
	BlockerLog                []ReviewBlocker         `json:"blocker_log,omitempty"`
	RequiresBlockerLog        bool                    `json:"requires_blocker_log,omitempty"`
	BlockerTimeline           []ReviewBlockerEvent    `json:"blocker_timeline,omitempty"`
	RequiresBlockerTimeline   bool                    `json:"requires_blocker_timeline,omitempty"`
}

type UIReviewPackArtifacts struct {
	RootDir                        string
	MarkdownPath                   string
	HTMLPath                       string
	DecisionLogPath                string
	ReviewSummaryBoardPath         string
	ObjectiveCoverageBoardPath     string
	PersonaReadinessBoardPath      string
	WireframeReadinessBoardPath    string
	InteractionCoverageBoardPath   string
	OpenQuestionTrackerPath        string
	ChecklistTraceabilityBoardPath string
	DecisionFollowupTrackerPath    string
	RoleMatrixPath                 string
	RoleCoverageBoardPath          string
	SignoffDependencyBoardPath     string
	SignoffLogPath                 string
	SignoffSLADashboardPath        string
	SignoffReminderQueuePath       string
	ReminderCadenceBoardPath       string
	SignoffBreachBoardPath         string
	EscalationDashboardPath        string
	EscalationHandoffLedgerPath    string
	HandoffAckLedgerPath           string
	OwnerEscalationDigestPath      string
	OwnerWorkloadBoardPath         string
	BlockerLogPath                 string
	BlockerTimelinePath            string
	FreezeExceptionBoardPath       string
	FreezeApprovalTrailPath        string
	FreezeRenewalTrackerPath       string
	ExceptionLogPath               string
	ExceptionMatrixPath            string
	AuditDensityBoardPath          string
	OwnerReviewQueuePath           string
	BlockerTimelineSummaryPath     string
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
	return fmt.Sprintf("%s: objectives=%d wireframes=%d interactions=%d open_questions=%d checklist=%d decisions=%d role_assignments=%d signoffs=%d blockers=%d timeline_events=%d",
		status, a.ObjectiveCount, a.WireframeCount, a.InteractionCount, a.OpenQuestionCount, a.ChecklistCount, a.DecisionCount, a.RoleAssignmentCount, a.SignoffCount, a.BlockerCount, a.BlockerTimelineCount)
}

func (p UIReviewPack) ToMap() map[string]any {
	data, _ := json.Marshal(p)
	var out map[string]any
	_ = json.Unmarshal(data, &out)
	return out
}

func UIReviewPackFromMap(data map[string]any) (UIReviewPack, error) {
	blob, err := json.Marshal(data)
	if err != nil {
		return UIReviewPack{}, err
	}
	var out UIReviewPack
	if err := json.Unmarshal(blob, &out); err != nil {
		return UIReviewPack{}, err
	}
	return out, nil
}

type UIReviewPackAuditor struct{}

func (UIReviewPackAuditor) Audit(pack UIReviewPack) UIReviewPackAudit {
	surfaces := surfaceMap(pack.Wireframes)
	checklistsBySurface := checklistBySurface(pack.ReviewerChecklist)
	decisionsBySurface := decisionsBySurface(pack.DecisionLog)
	assignmentsBySurface := assignmentsBySurface(pack.RoleMatrix)
	signoffsBySurface := signoffsBySurface(pack.SignoffLog)
	blockersBySignoff := blockersBySignoff(pack.BlockerLog)
	timelineByBlocker := timelineByBlocker(pack.BlockerTimeline)
	assignmentIDs := setFromAssignments(pack.RoleMatrix)
	signoffIDs := setFromSignoffs(pack.SignoffLog)
	checklistIDsLinked := set[string]{}
	decisionIDsLinked := set[string]{}
	for _, assignment := range pack.RoleMatrix {
		for _, id := range assignment.ChecklistItemIDs {
			checklistIDsLinked.add(id)
		}
		for _, id := range assignment.DecisionIDs {
			decisionIDsLinked.add(id)
		}
	}
	audit := UIReviewPackAudit{
		Ready:                true,
		ObjectiveCount:       len(pack.Objectives),
		WireframeCount:       len(pack.Wireframes),
		InteractionCount:     len(pack.Interactions),
		OpenQuestionCount:    len(pack.OpenQuestions),
		ChecklistCount:       len(pack.ReviewerChecklist),
		DecisionCount:        len(pack.DecisionLog),
		RoleAssignmentCount:  len(pack.RoleMatrix),
		SignoffCount:         len(pack.SignoffLog),
		BlockerCount:         len(pack.BlockerLog),
		BlockerTimelineCount: len(pack.BlockerTimeline),
	}
	if len(pack.Objectives) == 0 {
		audit.MissingSections = append(audit.MissingSections, "objectives")
	}
	if len(pack.Wireframes) == 0 {
		audit.MissingSections = append(audit.MissingSections, "wireframes")
	}
	if len(pack.Interactions) == 0 {
		audit.MissingSections = append(audit.MissingSections, "interactions")
	}
	if len(pack.OpenQuestions) == 0 {
		audit.MissingSections = append(audit.MissingSections, "open_questions")
	}
	for _, objective := range pack.Objectives {
		if strings.TrimSpace(objective.SuccessSignal) == "" {
			audit.ObjectivesMissingSignals = append(audit.ObjectivesMissingSignals, objective.ObjectiveID)
		}
	}
	for _, wireframe := range pack.Wireframes {
		if len(wireframe.PrimaryBlocks) == 0 {
			audit.WireframesMissingBlocks = append(audit.WireframesMissingBlocks, wireframe.SurfaceID)
		}
		if pack.RequiresReviewerChecklist && len(checklistsBySurface[wireframe.SurfaceID]) == 0 {
			audit.WireframesMissingChecklists = append(audit.WireframesMissingChecklists, wireframe.SurfaceID)
		}
		if pack.RequiresDecisionLog && len(decisionsBySurface[wireframe.SurfaceID]) == 0 {
			audit.WireframesMissingDecisions = append(audit.WireframesMissingDecisions, wireframe.SurfaceID)
		}
		if pack.RequiresRoleMatrix && len(assignmentsBySurface[wireframe.SurfaceID]) == 0 {
			audit.WireframesMissingRoleAssignments = append(audit.WireframesMissingRoleAssignments, wireframe.SurfaceID)
		}
		if pack.RequiresSignoffLog && len(signoffsBySurface[wireframe.SurfaceID]) == 0 {
			audit.WireframesMissingSignoffs = append(audit.WireframesMissingSignoffs, wireframe.SurfaceID)
		}
	}
	for _, interaction := range pack.Interactions {
		if len(interaction.States) == 0 {
			audit.InteractionsMissingStates = append(audit.InteractionsMissingStates, interaction.FlowID)
		}
	}
	for _, question := range pack.OpenQuestions {
		if question.Status == "" || question.Status == "open" {
			audit.UnresolvedQuestionIDs = append(audit.UnresolvedQuestionIDs, question.QuestionID)
		}
	}
	for _, item := range pack.ReviewerChecklist {
		if _, ok := surfaces[item.SurfaceID]; !ok {
			audit.OrphanChecklistSurfaces = append(audit.OrphanChecklistSurfaces, item.SurfaceID)
		}
		if len(item.EvidenceLinks) == 0 {
			audit.ChecklistItemsMissingEvidence = append(audit.ChecklistItemsMissingEvidence, item.ItemID)
		}
		if !checklistIDsLinked.has(item.ItemID) {
			audit.ChecklistItemsMissingRoleLinks = append(audit.ChecklistItemsMissingRoleLinks, item.ItemID)
		}
	}
	for _, decision := range pack.DecisionLog {
		if _, ok := surfaces[decision.SurfaceID]; !ok {
			audit.OrphanDecisionSurfaces = append(audit.OrphanDecisionSurfaces, decision.SurfaceID)
		}
		if decision.Status != "accepted" {
			audit.UnresolvedDecisionIDs = append(audit.UnresolvedDecisionIDs, decision.DecisionID)
			if strings.TrimSpace(decision.FollowUp) == "" {
				audit.UnresolvedDecisionsMissingFollowUps = append(audit.UnresolvedDecisionsMissingFollowUps, decision.DecisionID)
			}
		}
		if !decisionIDsLinked.has(decision.DecisionID) {
			audit.DecisionsMissingRoleLinks = append(audit.DecisionsMissingRoleLinks, decision.DecisionID)
		}
	}
	checklistIDs := setFromChecklist(pack.ReviewerChecklist)
	decisionIDs := setFromDecisions(pack.DecisionLog)
	for _, assignment := range pack.RoleMatrix {
		if _, ok := surfaces[assignment.SurfaceID]; !ok {
			audit.OrphanRoleAssignmentSurfaces = append(audit.OrphanRoleAssignmentSurfaces, assignment.SurfaceID)
		}
		if len(assignment.Responsibilities) == 0 {
			audit.RoleAssignmentsMissingResponsibilities = append(audit.RoleAssignmentsMissingResponsibilities, assignment.AssignmentID)
		}
		for _, id := range assignment.ChecklistItemIDs {
			if !checklistIDs.has(id) {
				audit.RoleAssignmentsMissingChecklistLinks = appendUnique(audit.RoleAssignmentsMissingChecklistLinks, assignment.AssignmentID)
			}
		}
		for _, id := range assignment.DecisionIDs {
			if !decisionIDs.has(id) {
				audit.RoleAssignmentsMissingDecisionLinks = appendUnique(audit.RoleAssignmentsMissingDecisionLinks, assignment.AssignmentID)
			}
		}
	}
	for _, signoff := range pack.SignoffLog {
		if _, ok := surfaces[signoff.SurfaceID]; !ok {
			audit.OrphanSignoffSurfaces = append(audit.OrphanSignoffSurfaces, signoff.SurfaceID)
		}
		if !assignmentIDs.has(signoff.AssignmentID) {
			audit.SignoffsMissingAssignments = append(audit.SignoffsMissingAssignments, signoff.SignoffID)
		}
		if len(signoff.EvidenceLinks) == 0 && signoff.Status != "waived" {
			audit.SignoffsMissingEvidence = append(audit.SignoffsMissingEvidence, signoff.SignoffID)
		}
		if signoff.Status == "pending" {
			audit.UnresolvedRequiredSignoffIDs = append(audit.UnresolvedRequiredSignoffIDs, signoff.SignoffID)
			if strings.TrimSpace(signoff.RequestedAt) == "" {
				audit.SignoffsMissingRequestedDates = append(audit.SignoffsMissingRequestedDates, signoff.SignoffID)
			}
			if strings.TrimSpace(signoff.DueAt) == "" {
				audit.SignoffsMissingDueDates = append(audit.SignoffsMissingDueDates, signoff.SignoffID)
			}
			if strings.TrimSpace(signoff.EscalationOwner) == "" {
				audit.SignoffsMissingEscalationOwners = append(audit.SignoffsMissingEscalationOwners, signoff.SignoffID)
			}
			if strings.TrimSpace(signoff.ReminderOwner) == "" {
				audit.SignoffsMissingReminderOwners = append(audit.SignoffsMissingReminderOwners, signoff.SignoffID)
			}
			if strings.TrimSpace(signoff.NextReminderAt) == "" {
				audit.SignoffsMissingNextReminders = append(audit.SignoffsMissingNextReminders, signoff.SignoffID)
			}
			if strings.TrimSpace(signoff.ReminderCadence) == "" {
				audit.SignoffsMissingReminderCadence = append(audit.SignoffsMissingReminderCadence, signoff.SignoffID)
			}
			if len(blockersBySignoff[signoff.SignoffID]) == 0 && signoff.Required {
				audit.UnresolvedRequiredSignoffsWithoutBlockers = append(audit.UnresolvedRequiredSignoffsWithoutBlockers, signoff.SignoffID)
			}
		}
		if signoff.SLAStatus == "breached" {
			audit.SignoffsWithBreachedSLA = append(audit.SignoffsWithBreachedSLA, signoff.SignoffID)
		}
		if signoff.Status == "waived" && (strings.TrimSpace(signoff.WaiverOwner) == "" || strings.TrimSpace(signoff.WaiverReason) == "") {
			audit.WaivedSignoffsMissingMetadata = append(audit.WaivedSignoffsMissingMetadata, signoff.SignoffID)
		}
	}
	for _, blocker := range pack.BlockerLog {
		if !signoffIDs.has(blocker.SignoffID) {
			audit.BlockersMissingSignoffLinks = append(audit.BlockersMissingSignoffLinks, blocker.BlockerID)
		}
		if strings.TrimSpace(blocker.EscalationOwner) == "" {
			audit.BlockersMissingEscalationOwners = append(audit.BlockersMissingEscalationOwners, blocker.BlockerID)
		}
		if strings.TrimSpace(blocker.NextAction) == "" {
			audit.BlockersMissingNextActions = append(audit.BlockersMissingNextActions, blocker.BlockerID)
		}
		if blocker.FreezeException {
			if blocker.FreezeOwner == "" {
				audit.FreezeExceptionsMissingOwners = append(audit.FreezeExceptionsMissingOwners, blocker.BlockerID)
			}
			if blocker.FreezeUntil == "" {
				audit.FreezeExceptionsMissingUntil = append(audit.FreezeExceptionsMissingUntil, blocker.BlockerID)
			}
			if blocker.FreezeApprovedBy == "" {
				audit.FreezeExceptionsMissingApprovers = append(audit.FreezeExceptionsMissingApprovers, blocker.BlockerID)
			}
			if blocker.FreezeApprovedAt == "" {
				audit.FreezeExceptionsMissingApprovalDates = append(audit.FreezeExceptionsMissingApprovalDates, blocker.BlockerID)
			}
			if blocker.FreezeRenewalOwner == "" {
				audit.FreezeExceptionsMissingRenewalOwners = append(audit.FreezeExceptionsMissingRenewalOwners, blocker.BlockerID)
			}
			if blocker.FreezeRenewalBy == "" {
				audit.FreezeExceptionsMissingRenewalDates = append(audit.FreezeExceptionsMissingRenewalDates, blocker.BlockerID)
			}
		}
		events := timelineByBlocker[blocker.BlockerID]
		if pack.RequiresBlockerTimeline && len(events) == 0 {
			audit.BlockersMissingTimelineEvents = append(audit.BlockersMissingTimelineEvents, blocker.BlockerID)
		}
		if blocker.Status == "closed" && !hasResolutionEvent(events) {
			audit.ClosedBlockersMissingResolutionEvents = append(audit.ClosedBlockersMissingResolutionEvents, blocker.BlockerID)
		}
	}
	for blockerID := range timelineByBlocker {
		if !blockersBySignoff.hasBlocker(blockerID) {
			audit.OrphanBlockerTimelineBlockerIDs = append(audit.OrphanBlockerTimelineBlockerIDs, blockerID)
		}
	}
	sort.Strings(audit.OrphanBlockerTimelineBlockerIDs)
	for _, event := range pack.BlockerTimeline {
		if event.HandoffFrom != "" || event.Status == "escalated" {
			if strings.TrimSpace(event.HandoffTo) == "" {
				audit.HandoffEventsMissingTargets = append(audit.HandoffEventsMissingTargets, event.EventID)
			}
			if strings.TrimSpace(event.ArtifactRef) == "" {
				audit.HandoffEventsMissingArtifacts = append(audit.HandoffEventsMissingArtifacts, event.EventID)
			}
			if strings.TrimSpace(event.AckOwner) == "" {
				audit.HandoffEventsMissingAckOwners = append(audit.HandoffEventsMissingAckOwners, event.EventID)
			}
			if strings.TrimSpace(event.AckAt) == "" {
				audit.HandoffEventsMissingAckDates = append(audit.HandoffEventsMissingAckDates, event.EventID)
			}
		}
	}
	blocking := [][]string{
		audit.MissingSections,
		audit.ObjectivesMissingSignals,
		audit.WireframesMissingBlocks,
		audit.InteractionsMissingStates,
		audit.WireframesMissingChecklists,
		audit.OrphanChecklistSurfaces,
		audit.ChecklistItemsMissingEvidence,
		audit.WireframesMissingDecisions,
		audit.OrphanDecisionSurfaces,
		audit.UnresolvedDecisionsMissingFollowUps,
		audit.WireframesMissingRoleAssignments,
		audit.OrphanRoleAssignmentSurfaces,
		audit.RoleAssignmentsMissingResponsibilities,
		audit.RoleAssignmentsMissingChecklistLinks,
		audit.RoleAssignmentsMissingDecisionLinks,
		audit.WireframesMissingSignoffs,
		audit.OrphanSignoffSurfaces,
		audit.SignoffsMissingAssignments,
		audit.SignoffsMissingEvidence,
		audit.SignoffsMissingRequestedDates,
		audit.SignoffsMissingDueDates,
		audit.SignoffsMissingEscalationOwners,
		audit.SignoffsMissingReminderOwners,
		audit.SignoffsMissingNextReminders,
		audit.SignoffsMissingReminderCadence,
		audit.SignoffsWithBreachedSLA,
		audit.WaivedSignoffsMissingMetadata,
		audit.BlockersMissingSignoffLinks,
		audit.BlockersMissingEscalationOwners,
		audit.BlockersMissingNextActions,
		audit.FreezeExceptionsMissingOwners,
		audit.FreezeExceptionsMissingUntil,
		audit.FreezeExceptionsMissingApprovers,
		audit.FreezeExceptionsMissingApprovalDates,
		audit.FreezeExceptionsMissingRenewalOwners,
		audit.FreezeExceptionsMissingRenewalDates,
		audit.BlockersMissingTimelineEvents,
		audit.ClosedBlockersMissingResolutionEvents,
		audit.OrphanBlockerTimelineBlockerIDs,
		audit.HandoffEventsMissingTargets,
		audit.HandoffEventsMissingArtifacts,
		audit.HandoffEventsMissingAckOwners,
		audit.HandoffEventsMissingAckDates,
		audit.UnresolvedRequiredSignoffsWithoutBlockers,
	}
	for _, bucket := range blocking {
		if len(bucket) > 0 {
			audit.Ready = false
			break
		}
	}
	return audit
}

// helpers and renderers omitted from this patch chunk
