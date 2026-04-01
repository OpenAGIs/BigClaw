package uireview

import (
	"fmt"
	"slices"
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

func (objective ReviewObjective) withDefaults() ReviewObjective {
	if objective.Priority == "" {
		objective.Priority = "P1"
	}
	return objective
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

func (question OpenQuestion) withDefaults() OpenQuestion {
	if question.Status == "" {
		question.Status = "open"
	}
	return question
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

func (item ReviewerChecklistItem) withDefaults() ReviewerChecklistItem {
	if item.Status == "" {
		item.Status = "todo"
	}
	return item
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

func (decision ReviewDecision) withDefaults() ReviewDecision {
	if decision.Status == "" {
		decision.Status = "proposed"
	}
	return decision
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

func (assignment ReviewRoleAssignment) withDefaults() ReviewRoleAssignment {
	if assignment.Status == "" {
		assignment.Status = "planned"
	}
	return assignment
}

type ReviewSignoff struct {
	SignoffID       string   `json:"signoff_id"`
	AssignmentID    string   `json:"assignment_id"`
	SurfaceID       string   `json:"surface_id"`
	Role            string   `json:"role"`
	Status          string   `json:"status,omitempty"`
	Required        *bool    `json:"required,omitempty"`
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

func (signoff ReviewSignoff) withDefaults() ReviewSignoff {
	if signoff.Status == "" {
		signoff.Status = "pending"
	}
	if signoff.Required == nil {
		required := true
		signoff.Required = &required
	}
	if signoff.SLAStatus == "" {
		signoff.SLAStatus = "on-track"
	}
	if signoff.ReminderStatus == "" {
		signoff.ReminderStatus = "scheduled"
	}
	return signoff
}

func (signoff ReviewSignoff) RequiredValue() bool {
	return signoff.withDefaults().Required != nil && *signoff.withDefaults().Required
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

func (blocker ReviewBlocker) withDefaults() ReviewBlocker {
	if blocker.Status == "" {
		blocker.Status = "open"
	}
	if blocker.Severity == "" {
		blocker.Severity = "medium"
	}
	if blocker.FreezeRenewalStatus == "" {
		blocker.FreezeRenewalStatus = "not-needed"
	}
	return blocker
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

func (event ReviewBlockerEvent) withDefaults() ReviewBlockerEvent {
	if event.AckStatus == "" {
		event.AckStatus = "pending"
	}
	return event
}

type UIReviewPackArtifacts struct {
	RootDir                        string `json:"root_dir"`
	MarkdownPath                   string `json:"markdown_path"`
	HTMLPath                       string `json:"html_path"`
	DecisionLogPath                string `json:"decision_log_path"`
	ReviewSummaryBoardPath         string `json:"review_summary_board_path"`
	ObjectiveCoverageBoardPath     string `json:"objective_coverage_board_path"`
	PersonaReadinessBoardPath      string `json:"persona_readiness_board_path"`
	WireframeReadinessBoardPath    string `json:"wireframe_readiness_board_path"`
	InteractionCoverageBoardPath   string `json:"interaction_coverage_board_path"`
	OpenQuestionTrackerPath        string `json:"open_question_tracker_path"`
	ChecklistTraceabilityBoardPath string `json:"checklist_traceability_board_path"`
	DecisionFollowupTrackerPath    string `json:"decision_followup_tracker_path"`
	RoleMatrixPath                 string `json:"role_matrix_path"`
	RoleCoverageBoardPath          string `json:"role_coverage_board_path"`
	SignoffDependencyBoardPath     string `json:"signoff_dependency_board_path"`
	SignoffLogPath                 string `json:"signoff_log_path"`
	SignoffSLADashboardPath        string `json:"signoff_sla_dashboard_path"`
	SignoffReminderQueuePath       string `json:"signoff_reminder_queue_path"`
	ReminderCadenceBoardPath       string `json:"reminder_cadence_board_path"`
	SignoffBreachBoardPath         string `json:"signoff_breach_board_path"`
	EscalationDashboardPath        string `json:"escalation_dashboard_path"`
	EscalationHandoffLedgerPath    string `json:"escalation_handoff_ledger_path"`
	HandoffAckLedgerPath           string `json:"handoff_ack_ledger_path"`
	OwnerEscalationDigestPath      string `json:"owner_escalation_digest_path"`
	OwnerWorkloadBoardPath         string `json:"owner_workload_board_path"`
	BlockerLogPath                 string `json:"blocker_log_path"`
	BlockerTimelinePath            string `json:"blocker_timeline_path"`
	FreezeExceptionBoardPath       string `json:"freeze_exception_board_path"`
	FreezeApprovalTrailPath        string `json:"freeze_approval_trail_path"`
	FreezeRenewalTrackerPath       string `json:"freeze_renewal_tracker_path"`
	ExceptionLogPath               string `json:"exception_log_path"`
	ExceptionMatrixPath            string `json:"exception_matrix_path"`
	AuditDensityBoardPath          string `json:"audit_density_board_path"`
	OwnerReviewQueuePath           string `json:"owner_review_queue_path"`
	BlockerTimelineSummaryPath     string `json:"blocker_timeline_summary_path"`
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

type UIReviewPackAudit struct {
	Ready                                     bool     `json:"ready"`
	ObjectiveCount                            int      `json:"objective_count"`
	WireframeCount                            int      `json:"wireframe_count"`
	InteractionCount                          int      `json:"interaction_count"`
	OpenQuestionCount                         int      `json:"open_question_count"`
	ChecklistCount                            int      `json:"checklist_count,omitempty"`
	DecisionCount                             int      `json:"decision_count,omitempty"`
	RoleAssignmentCount                       int      `json:"role_assignment_count,omitempty"`
	SignoffCount                              int      `json:"signoff_count,omitempty"`
	BlockerCount                              int      `json:"blocker_count,omitempty"`
	BlockerTimelineCount                      int      `json:"blocker_timeline_count,omitempty"`
	MissingSections                           []string `json:"missing_sections,omitempty"`
	ObjectivesMissingSignals                  []string `json:"objectives_missing_signals,omitempty"`
	WireframesMissingBlocks                   []string `json:"wireframes_missing_blocks,omitempty"`
	InteractionsMissingStates                 []string `json:"interactions_missing_states,omitempty"`
	UnresolvedQuestionIDs                     []string `json:"unresolved_question_ids,omitempty"`
	WireframesMissingChecklists               []string `json:"wireframes_missing_checklists,omitempty"`
	OrphanChecklistSurfaces                   []string `json:"orphan_checklist_surfaces,omitempty"`
	ChecklistItemsMissingEvidence             []string `json:"checklist_items_missing_evidence,omitempty"`
	ChecklistItemsMissingRoleLinks            []string `json:"checklist_items_missing_role_links,omitempty"`
	WireframesMissingDecisions                []string `json:"wireframes_missing_decisions,omitempty"`
	OrphanDecisionSurfaces                    []string `json:"orphan_decision_surfaces,omitempty"`
	UnresolvedDecisionIDs                     []string `json:"unresolved_decision_ids,omitempty"`
	UnresolvedDecisionsMissingFollowUps       []string `json:"unresolved_decisions_missing_follow_ups,omitempty"`
	WireframesMissingRoleAssignments          []string `json:"wireframes_missing_role_assignments,omitempty"`
	OrphanRoleAssignmentSurfaces              []string `json:"orphan_role_assignment_surfaces,omitempty"`
	RoleAssignmentsMissingResponsibilities    []string `json:"role_assignments_missing_responsibilities,omitempty"`
	RoleAssignmentsMissingChecklistLinks      []string `json:"role_assignments_missing_checklist_links,omitempty"`
	RoleAssignmentsMissingDecisionLinks       []string `json:"role_assignments_missing_decision_links,omitempty"`
	DecisionsMissingRoleLinks                 []string `json:"decisions_missing_role_links,omitempty"`
	WireframesMissingSignoffs                 []string `json:"wireframes_missing_signoffs,omitempty"`
	OrphanSignoffSurfaces                     []string `json:"orphan_signoff_surfaces,omitempty"`
	SignoffsMissingAssignments                []string `json:"signoffs_missing_assignments,omitempty"`
	SignoffsMissingEvidence                   []string `json:"signoffs_missing_evidence,omitempty"`
	SignoffsMissingRequestedDates             []string `json:"signoffs_missing_requested_dates,omitempty"`
	SignoffsMissingDueDates                   []string `json:"signoffs_missing_due_dates,omitempty"`
	SignoffsMissingEscalationOwners           []string `json:"signoffs_missing_escalation_owners,omitempty"`
	SignoffsMissingReminderOwners             []string `json:"signoffs_missing_reminder_owners,omitempty"`
	SignoffsMissingNextReminders              []string `json:"signoffs_missing_next_reminders,omitempty"`
	SignoffsMissingReminderCadence            []string `json:"signoffs_missing_reminder_cadence,omitempty"`
	SignoffsWithBreachedSLA                   []string `json:"signoffs_with_breached_sla,omitempty"`
	WaivedSignoffsMissingMetadata             []string `json:"waived_signoffs_missing_metadata,omitempty"`
	UnresolvedRequiredSignoffIDs              []string `json:"unresolved_required_signoff_ids,omitempty"`
	BlockersMissingSignoffLinks               []string `json:"blockers_missing_signoff_links,omitempty"`
	BlockersMissingEscalationOwners           []string `json:"blockers_missing_escalation_owners,omitempty"`
	BlockersMissingNextActions                []string `json:"blockers_missing_next_actions,omitempty"`
	FreezeExceptionsMissingOwners             []string `json:"freeze_exceptions_missing_owners,omitempty"`
	FreezeExceptionsMissingUntil              []string `json:"freeze_exceptions_missing_until,omitempty"`
	FreezeExceptionsMissingApprovers          []string `json:"freeze_exceptions_missing_approvers,omitempty"`
	FreezeExceptionsMissingApprovalDates      []string `json:"freeze_exceptions_missing_approval_dates,omitempty"`
	FreezeExceptionsMissingRenewalOwners      []string `json:"freeze_exceptions_missing_renewal_owners,omitempty"`
	FreezeExceptionsMissingRenewalDates       []string `json:"freeze_exceptions_missing_renewal_dates,omitempty"`
	BlockersMissingTimelineEvents             []string `json:"blockers_missing_timeline_events,omitempty"`
	ClosedBlockersMissingResolutionEvents     []string `json:"closed_blockers_missing_resolution_events,omitempty"`
	OrphanBlockerSurfaces                     []string `json:"orphan_blocker_surfaces,omitempty"`
	OrphanBlockerTimelineBlockerIDs           []string `json:"orphan_blocker_timeline_blocker_ids,omitempty"`
	HandoffEventsMissingTargets               []string `json:"handoff_events_missing_targets,omitempty"`
	HandoffEventsMissingArtifacts             []string `json:"handoff_events_missing_artifacts,omitempty"`
	HandoffEventsMissingAckOwners             []string `json:"handoff_events_missing_ack_owners,omitempty"`
	HandoffEventsMissingAckDates              []string `json:"handoff_events_missing_ack_dates,omitempty"`
	UnresolvedRequiredSignoffsWithoutBlockers []string `json:"unresolved_required_signoffs_without_blockers,omitempty"`
}

func (audit UIReviewPackAudit) Summary() string {
	status := "HOLD"
	if audit.Ready {
		status = "READY"
	}
	return fmt.Sprintf(
		"%s: objectives=%d wireframes=%d interactions=%d open_questions=%d checklist=%d decisions=%d role_assignments=%d signoffs=%d blockers=%d timeline_events=%d",
		status,
		audit.ObjectiveCount,
		audit.WireframeCount,
		audit.InteractionCount,
		audit.OpenQuestionCount,
		audit.ChecklistCount,
		audit.DecisionCount,
		audit.RoleAssignmentCount,
		audit.SignoffCount,
		audit.BlockerCount,
		audit.BlockerTimelineCount,
	)
}

type UIReviewPackAuditor struct{}

func (UIReviewPackAuditor) Audit(pack UIReviewPack) UIReviewPackAudit {
	missingSections := make([]string, 0)
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

	objectivesMissingSignals := make([]string, 0)
	for _, objective := range pack.Objectives {
		if strings.TrimSpace(objective.SuccessSignal) == "" {
			objectivesMissingSignals = append(objectivesMissingSignals, objective.ObjectiveID)
		}
	}
	wireframesMissingBlocks := make([]string, 0)
	for _, wireframe := range pack.Wireframes {
		if len(wireframe.PrimaryBlocks) == 0 {
			wireframesMissingBlocks = append(wireframesMissingBlocks, wireframe.SurfaceID)
		}
	}
	interactionsMissingStates := make([]string, 0)
	for _, interaction := range pack.Interactions {
		if len(interaction.States) == 0 {
			interactionsMissingStates = append(interactionsMissingStates, interaction.FlowID)
		}
	}
	unresolvedQuestionIDs := make([]string, 0)
	for _, question := range pack.OpenQuestions {
		if strings.ToLower(question.withDefaults().Status) != "resolved" {
			unresolvedQuestionIDs = append(unresolvedQuestionIDs, question.QuestionID)
		}
	}

	wireframeIDs := map[string]struct{}{}
	for _, wireframe := range pack.Wireframes {
		wireframeIDs[wireframe.SurfaceID] = struct{}{}
	}

	checklistBySurface := map[string][]ReviewerChecklistItem{}
	for _, item := range pack.ReviewerChecklist {
		item = item.withDefaults()
		checklistBySurface[item.SurfaceID] = append(checklistBySurface[item.SurfaceID], item)
	}
	wireframesMissingChecklists := make([]string, 0)
	orphanChecklistSurfaces := make([]string, 0)
	checklistItemsMissingEvidence := make([]string, 0)
	checklistItemsMissingRoleLinks := make([]string, 0)
	if pack.RequiresReviewerChecklist {
		for _, wireframe := range pack.Wireframes {
			if _, ok := checklistBySurface[wireframe.SurfaceID]; !ok {
				wireframesMissingChecklists = append(wireframesMissingChecklists, wireframe.SurfaceID)
			}
		}
		for surfaceID := range checklistBySurface {
			if _, ok := wireframeIDs[surfaceID]; !ok {
				orphanChecklistSurfaces = append(orphanChecklistSurfaces, surfaceID)
			}
		}
		for _, item := range pack.ReviewerChecklist {
			if len(item.EvidenceLinks) == 0 {
				checklistItemsMissingEvidence = append(checklistItemsMissingEvidence, item.ItemID)
			}
		}
	}

	decisionBySurface := map[string][]ReviewDecision{}
	for _, decision := range pack.DecisionLog {
		decision = decision.withDefaults()
		decisionBySurface[decision.SurfaceID] = append(decisionBySurface[decision.SurfaceID], decision)
	}
	wireframesMissingDecisions := make([]string, 0)
	orphanDecisionSurfaces := make([]string, 0)
	unresolvedDecisionIDs := make([]string, 0)
	unresolvedDecisionsMissingFollowUps := make([]string, 0)
	if pack.RequiresDecisionLog {
		for _, wireframe := range pack.Wireframes {
			if _, ok := decisionBySurface[wireframe.SurfaceID]; !ok {
				wireframesMissingDecisions = append(wireframesMissingDecisions, wireframe.SurfaceID)
			}
		}
		for surfaceID := range decisionBySurface {
			if _, ok := wireframeIDs[surfaceID]; !ok {
				orphanDecisionSurfaces = append(orphanDecisionSurfaces, surfaceID)
			}
		}
		for _, decision := range pack.DecisionLog {
			status := strings.ToLower(decision.withDefaults().Status)
			if !slices.Contains([]string{"accepted", "approved", "resolved", "waived"}, status) {
				unresolvedDecisionIDs = append(unresolvedDecisionIDs, decision.DecisionID)
				if strings.TrimSpace(decision.FollowUp) == "" {
					unresolvedDecisionsMissingFollowUps = append(unresolvedDecisionsMissingFollowUps, decision.DecisionID)
				}
			}
		}
	}

	checklistItemIDs := sliceSetMap(packChecklistIDs(pack.ReviewerChecklist))
	decisionIDs := sliceSetMap(packDecisionIDs(pack.DecisionLog))
	assignmentIDs := sliceSetMap(packAssignmentIDs(pack.RoleMatrix))

	roleAssignmentsBySurface := map[string][]ReviewRoleAssignment{}
	for _, assignment := range pack.RoleMatrix {
		assignment = assignment.withDefaults()
		roleAssignmentsBySurface[assignment.SurfaceID] = append(roleAssignmentsBySurface[assignment.SurfaceID], assignment)
	}
	wireframesMissingRoleAssignments := make([]string, 0)
	orphanRoleAssignmentSurfaces := make([]string, 0)
	roleAssignmentsMissingResponsibilities := make([]string, 0)
	roleAssignmentsMissingChecklistLinks := make([]string, 0)
	roleAssignmentsMissingDecisionLinks := make([]string, 0)
	decisionsMissingRoleLinks := make([]string, 0)
	if pack.RequiresRoleMatrix {
		for _, wireframe := range pack.Wireframes {
			if _, ok := roleAssignmentsBySurface[wireframe.SurfaceID]; !ok {
				wireframesMissingRoleAssignments = append(wireframesMissingRoleAssignments, wireframe.SurfaceID)
			}
		}
		for surfaceID := range roleAssignmentsBySurface {
			if _, ok := wireframeIDs[surfaceID]; !ok {
				orphanRoleAssignmentSurfaces = append(orphanRoleAssignmentSurfaces, surfaceID)
			}
		}
		roleLinkedChecklistIDs := map[string]struct{}{}
		roleLinkedDecisionIDs := map[string]struct{}{}
		for _, assignment := range pack.RoleMatrix {
			if len(assignment.Responsibilities) == 0 {
				roleAssignmentsMissingResponsibilities = append(roleAssignmentsMissingResponsibilities, assignment.AssignmentID)
			}
			if len(assignment.ChecklistItemIDs) == 0 || anyMissing(assignment.ChecklistItemIDs, checklistItemIDs) {
				roleAssignmentsMissingChecklistLinks = append(roleAssignmentsMissingChecklistLinks, assignment.AssignmentID)
			}
			if len(assignment.DecisionIDs) == 0 || anyMissing(assignment.DecisionIDs, decisionIDs) {
				roleAssignmentsMissingDecisionLinks = append(roleAssignmentsMissingDecisionLinks, assignment.AssignmentID)
			}
			for _, itemID := range assignment.ChecklistItemIDs {
				roleLinkedChecklistIDs[itemID] = struct{}{}
			}
			for _, decisionID := range assignment.DecisionIDs {
				roleLinkedDecisionIDs[decisionID] = struct{}{}
			}
		}
		for _, item := range pack.ReviewerChecklist {
			if _, ok := roleLinkedChecklistIDs[item.ItemID]; !ok {
				checklistItemsMissingRoleLinks = append(checklistItemsMissingRoleLinks, item.ItemID)
			}
		}
		for _, decision := range pack.DecisionLog {
			if _, ok := roleLinkedDecisionIDs[decision.DecisionID]; !ok {
				decisionsMissingRoleLinks = append(decisionsMissingRoleLinks, decision.DecisionID)
			}
		}
	}

	signoffsBySurface := map[string][]ReviewSignoff{}
	for _, signoff := range pack.SignoffLog {
		signoff = signoff.withDefaults()
		signoffsBySurface[signoff.SurfaceID] = append(signoffsBySurface[signoff.SurfaceID], signoff)
	}
	wireframesMissingSignoffs := make([]string, 0)
	orphanSignoffSurfaces := make([]string, 0)
	signoffsMissingAssignments := make([]string, 0)
	signoffsMissingEvidence := make([]string, 0)
	signoffsMissingRequestedDates := make([]string, 0)
	signoffsMissingDueDates := make([]string, 0)
	signoffsMissingEscalationOwners := make([]string, 0)
	signoffsMissingReminderOwners := make([]string, 0)
	signoffsMissingNextReminders := make([]string, 0)
	signoffsMissingReminderCadence := make([]string, 0)
	signoffsWithBreachedSLA := make([]string, 0)
	waivedSignoffsMissingMetadata := make([]string, 0)
	unresolvedRequiredSignoffIDs := make([]string, 0)
	if pack.RequiresSignoffLog {
		for _, wireframe := range pack.Wireframes {
			if _, ok := signoffsBySurface[wireframe.SurfaceID]; !ok {
				wireframesMissingSignoffs = append(wireframesMissingSignoffs, wireframe.SurfaceID)
			}
		}
		for surfaceID := range signoffsBySurface {
			if _, ok := wireframeIDs[surfaceID]; !ok {
				orphanSignoffSurfaces = append(orphanSignoffSurfaces, surfaceID)
			}
		}
		unresolvedStatuses := []string{"approved", "accepted", "resolved", "waived", "deferred"}
		for _, signoff := range pack.SignoffLog {
			signoff = signoff.withDefaults()
			if _, ok := assignmentIDs[signoff.AssignmentID]; !ok {
				signoffsMissingAssignments = append(signoffsMissingAssignments, signoff.SignoffID)
			}
			if strings.ToLower(signoff.Status) != "waived" && len(signoff.EvidenceLinks) == 0 {
				signoffsMissingEvidence = append(signoffsMissingEvidence, signoff.SignoffID)
			}
			if signoff.RequiredValue() && strings.TrimSpace(signoff.RequestedAt) == "" {
				signoffsMissingRequestedDates = append(signoffsMissingRequestedDates, signoff.SignoffID)
			}
			if signoff.RequiredValue() && strings.TrimSpace(signoff.DueAt) == "" {
				signoffsMissingDueDates = append(signoffsMissingDueDates, signoff.SignoffID)
			}
			if signoff.RequiredValue() && strings.TrimSpace(signoff.EscalationOwner) == "" {
				signoffsMissingEscalationOwners = append(signoffsMissingEscalationOwners, signoff.SignoffID)
			}
			if signoff.RequiredValue() && !slices.Contains(unresolvedStatuses, strings.ToLower(signoff.Status)) && strings.TrimSpace(signoff.ReminderOwner) == "" {
				signoffsMissingReminderOwners = append(signoffsMissingReminderOwners, signoff.SignoffID)
			}
			if signoff.RequiredValue() && !slices.Contains(unresolvedStatuses, strings.ToLower(signoff.Status)) && strings.TrimSpace(signoff.NextReminderAt) == "" {
				signoffsMissingNextReminders = append(signoffsMissingNextReminders, signoff.SignoffID)
			}
			if signoff.RequiredValue() && !slices.Contains(unresolvedStatuses, strings.ToLower(signoff.Status)) && strings.TrimSpace(signoff.ReminderCadence) == "" {
				signoffsMissingReminderCadence = append(signoffsMissingReminderCadence, signoff.SignoffID)
			}
			if strings.ToLower(signoff.SLAStatus) == "breached" && !slices.Contains([]string{"approved", "accepted", "resolved"}, strings.ToLower(signoff.Status)) {
				signoffsWithBreachedSLA = append(signoffsWithBreachedSLA, signoff.SignoffID)
			}
			if strings.ToLower(signoff.Status) == "waived" && (strings.TrimSpace(signoff.WaiverOwner) == "" || strings.TrimSpace(signoff.WaiverReason) == "") {
				waivedSignoffsMissingMetadata = append(waivedSignoffsMissingMetadata, signoff.SignoffID)
			}
			if signoff.RequiredValue() && !slices.Contains(unresolvedStatuses, strings.ToLower(signoff.Status)) {
				unresolvedRequiredSignoffIDs = append(unresolvedRequiredSignoffIDs, signoff.SignoffID)
			}
		}
	}

	blockerBySignoff := map[string][]ReviewBlocker{}
	blockerSurfaces := map[string]struct{}{}
	for _, blocker := range pack.BlockerLog {
		blocker = blocker.withDefaults()
		blockerSurfaces[blocker.SurfaceID] = struct{}{}
		blockerBySignoff[blocker.SignoffID] = append(blockerBySignoff[blocker.SignoffID], blocker)
	}
	blockersMissingSignoffLinks := make([]string, 0)
	blockersMissingEscalationOwners := make([]string, 0)
	blockersMissingNextActions := make([]string, 0)
	freezeExceptionsMissingOwners := make([]string, 0)
	freezeExceptionsMissingUntil := make([]string, 0)
	freezeExceptionsMissingApprovers := make([]string, 0)
	freezeExceptionsMissingApprovalDates := make([]string, 0)
	freezeExceptionsMissingRenewalOwners := make([]string, 0)
	freezeExceptionsMissingRenewalDates := make([]string, 0)
	orphanBlockerSurfaces := make([]string, 0)
	unresolvedRequiredSignoffsWithoutBlockers := make([]string, 0)
	if pack.RequiresBlockerLog {
		signoffIDs := sliceSetMap(packSignoffIDs(pack.SignoffLog))
		for _, blocker := range pack.BlockerLog {
			blocker = blocker.withDefaults()
			if _, ok := signoffIDs[blocker.SignoffID]; !ok {
				blockersMissingSignoffLinks = append(blockersMissingSignoffLinks, blocker.BlockerID)
			}
			if strings.TrimSpace(blocker.EscalationOwner) == "" {
				blockersMissingEscalationOwners = append(blockersMissingEscalationOwners, blocker.BlockerID)
			}
			if strings.TrimSpace(blocker.NextAction) == "" {
				blockersMissingNextActions = append(blockersMissingNextActions, blocker.BlockerID)
			}
			if blocker.FreezeException && strings.TrimSpace(blocker.FreezeOwner) == "" {
				freezeExceptionsMissingOwners = append(freezeExceptionsMissingOwners, blocker.BlockerID)
			}
			if blocker.FreezeException && strings.TrimSpace(blocker.FreezeUntil) == "" {
				freezeExceptionsMissingUntil = append(freezeExceptionsMissingUntil, blocker.BlockerID)
			}
			if blocker.FreezeException && strings.TrimSpace(blocker.FreezeApprovedBy) == "" {
				freezeExceptionsMissingApprovers = append(freezeExceptionsMissingApprovers, blocker.BlockerID)
			}
			if blocker.FreezeException && strings.TrimSpace(blocker.FreezeApprovedAt) == "" {
				freezeExceptionsMissingApprovalDates = append(freezeExceptionsMissingApprovalDates, blocker.BlockerID)
			}
			if blocker.FreezeException && strings.TrimSpace(blocker.FreezeRenewalOwner) == "" {
				freezeExceptionsMissingRenewalOwners = append(freezeExceptionsMissingRenewalOwners, blocker.BlockerID)
			}
			if blocker.FreezeException && strings.TrimSpace(blocker.FreezeRenewalBy) == "" {
				freezeExceptionsMissingRenewalDates = append(freezeExceptionsMissingRenewalDates, blocker.BlockerID)
			}
		}
		for surfaceID := range blockerSurfaces {
			if _, ok := wireframeIDs[surfaceID]; !ok {
				orphanBlockerSurfaces = append(orphanBlockerSurfaces, surfaceID)
			}
		}
		for _, signoffID := range unresolvedRequiredSignoffIDs {
			if _, ok := blockerBySignoff[signoffID]; !ok {
				unresolvedRequiredSignoffsWithoutBlockers = append(unresolvedRequiredSignoffsWithoutBlockers, signoffID)
			}
		}
	}

	blockerTimelineByBlocker := map[string][]ReviewBlockerEvent{}
	for _, event := range pack.BlockerTimeline {
		event = event.withDefaults()
		blockerTimelineByBlocker[event.BlockerID] = append(blockerTimelineByBlocker[event.BlockerID], event)
	}
	blockersMissingTimelineEvents := make([]string, 0)
	closedBlockersMissingResolutionEvents := make([]string, 0)
	orphanBlockerTimelineBlockerIDs := make([]string, 0)
	handoffEventsMissingTargets := make([]string, 0)
	handoffEventsMissingArtifacts := make([]string, 0)
	handoffEventsMissingAckOwners := make([]string, 0)
	handoffEventsMissingAckDates := make([]string, 0)
	if pack.RequiresBlockerTimeline {
		blockerIDs := sliceSetMap(packBlockerIDs(pack.BlockerLog))
		for blockerID := range blockerTimelineByBlocker {
			if _, ok := blockerIDs[blockerID]; !ok {
				orphanBlockerTimelineBlockerIDs = append(orphanBlockerTimelineBlockerIDs, blockerID)
			}
		}
		for _, blocker := range pack.BlockerLog {
			blocker = blocker.withDefaults()
			events := blockerTimelineByBlocker[blocker.BlockerID]
			if !slices.Contains([]string{"resolved", "closed"}, strings.ToLower(blocker.Status)) && len(events) == 0 {
				blockersMissingTimelineEvents = append(blockersMissingTimelineEvents, blocker.BlockerID)
			}
			if slices.Contains([]string{"resolved", "closed"}, strings.ToLower(blocker.Status)) {
				foundResolution := false
				for _, event := range events {
					if slices.Contains([]string{"resolved", "closed"}, strings.ToLower(event.Status)) {
						foundResolution = true
						break
					}
				}
				if !foundResolution {
					closedBlockersMissingResolutionEvents = append(closedBlockersMissingResolutionEvents, blocker.BlockerID)
				}
			}
		}
		handoffStatuses := []string{"escalated", "handoff", "reassigned"}
		for _, event := range pack.BlockerTimeline {
			event = event.withDefaults()
			if slices.Contains(handoffStatuses, strings.ToLower(event.Status)) && strings.TrimSpace(event.HandoffTo) == "" {
				handoffEventsMissingTargets = append(handoffEventsMissingTargets, event.EventID)
			}
			if slices.Contains(handoffStatuses, strings.ToLower(event.Status)) && strings.TrimSpace(event.ArtifactRef) == "" {
				handoffEventsMissingArtifacts = append(handoffEventsMissingArtifacts, event.EventID)
			}
			if slices.Contains(handoffStatuses, strings.ToLower(event.Status)) && strings.TrimSpace(event.AckOwner) == "" {
				handoffEventsMissingAckOwners = append(handoffEventsMissingAckOwners, event.EventID)
			}
			if slices.Contains(handoffStatuses, strings.ToLower(event.Status)) && strings.TrimSpace(event.AckAt) == "" {
				handoffEventsMissingAckDates = append(handoffEventsMissingAckDates, event.EventID)
			}
		}
	}

	sortAll(
		&missingSections,
		&objectivesMissingSignals,
		&wireframesMissingBlocks,
		&interactionsMissingStates,
		&unresolvedQuestionIDs,
		&wireframesMissingChecklists,
		&orphanChecklistSurfaces,
		&checklistItemsMissingEvidence,
		&checklistItemsMissingRoleLinks,
		&wireframesMissingDecisions,
		&orphanDecisionSurfaces,
		&unresolvedDecisionIDs,
		&unresolvedDecisionsMissingFollowUps,
		&wireframesMissingRoleAssignments,
		&orphanRoleAssignmentSurfaces,
		&roleAssignmentsMissingResponsibilities,
		&roleAssignmentsMissingChecklistLinks,
		&roleAssignmentsMissingDecisionLinks,
		&decisionsMissingRoleLinks,
		&wireframesMissingSignoffs,
		&orphanSignoffSurfaces,
		&signoffsMissingAssignments,
		&signoffsMissingEvidence,
		&signoffsMissingRequestedDates,
		&signoffsMissingDueDates,
		&signoffsMissingEscalationOwners,
		&signoffsMissingReminderOwners,
		&signoffsMissingNextReminders,
		&signoffsMissingReminderCadence,
		&signoffsWithBreachedSLA,
		&waivedSignoffsMissingMetadata,
		&unresolvedRequiredSignoffIDs,
		&blockersMissingSignoffLinks,
		&blockersMissingEscalationOwners,
		&blockersMissingNextActions,
		&freezeExceptionsMissingOwners,
		&freezeExceptionsMissingUntil,
		&freezeExceptionsMissingApprovers,
		&freezeExceptionsMissingApprovalDates,
		&freezeExceptionsMissingRenewalOwners,
		&freezeExceptionsMissingRenewalDates,
		&blockersMissingTimelineEvents,
		&closedBlockersMissingResolutionEvents,
		&orphanBlockerSurfaces,
		&orphanBlockerTimelineBlockerIDs,
		&handoffEventsMissingTargets,
		&handoffEventsMissingArtifacts,
		&handoffEventsMissingAckOwners,
		&handoffEventsMissingAckDates,
		&unresolvedRequiredSignoffsWithoutBlockers,
	)

	ready := !(len(missingSections) > 0 ||
		len(objectivesMissingSignals) > 0 ||
		len(wireframesMissingBlocks) > 0 ||
		len(interactionsMissingStates) > 0 ||
		len(wireframesMissingChecklists) > 0 ||
		len(orphanChecklistSurfaces) > 0 ||
		len(checklistItemsMissingEvidence) > 0 ||
		len(checklistItemsMissingRoleLinks) > 0 ||
		len(wireframesMissingDecisions) > 0 ||
		len(orphanDecisionSurfaces) > 0 ||
		len(unresolvedDecisionsMissingFollowUps) > 0 ||
		len(wireframesMissingRoleAssignments) > 0 ||
		len(orphanRoleAssignmentSurfaces) > 0 ||
		len(roleAssignmentsMissingResponsibilities) > 0 ||
		len(roleAssignmentsMissingChecklistLinks) > 0 ||
		len(roleAssignmentsMissingDecisionLinks) > 0 ||
		len(decisionsMissingRoleLinks) > 0 ||
		len(wireframesMissingSignoffs) > 0 ||
		len(orphanSignoffSurfaces) > 0 ||
		len(signoffsMissingAssignments) > 0 ||
		len(signoffsMissingEvidence) > 0 ||
		len(signoffsMissingRequestedDates) > 0 ||
		len(signoffsMissingDueDates) > 0 ||
		len(signoffsMissingEscalationOwners) > 0 ||
		len(signoffsMissingReminderOwners) > 0 ||
		len(signoffsMissingNextReminders) > 0 ||
		len(signoffsMissingReminderCadence) > 0 ||
		len(waivedSignoffsMissingMetadata) > 0 ||
		len(blockersMissingSignoffLinks) > 0 ||
		len(blockersMissingEscalationOwners) > 0 ||
		len(blockersMissingNextActions) > 0 ||
		len(freezeExceptionsMissingOwners) > 0 ||
		len(freezeExceptionsMissingUntil) > 0 ||
		len(freezeExceptionsMissingApprovers) > 0 ||
		len(freezeExceptionsMissingApprovalDates) > 0 ||
		len(freezeExceptionsMissingRenewalOwners) > 0 ||
		len(freezeExceptionsMissingRenewalDates) > 0 ||
		len(blockersMissingTimelineEvents) > 0 ||
		len(closedBlockersMissingResolutionEvents) > 0 ||
		len(orphanBlockerSurfaces) > 0 ||
		len(orphanBlockerTimelineBlockerIDs) > 0 ||
		len(handoffEventsMissingTargets) > 0 ||
		len(handoffEventsMissingArtifacts) > 0 ||
		len(handoffEventsMissingAckOwners) > 0 ||
		len(handoffEventsMissingAckDates) > 0 ||
		len(unresolvedRequiredSignoffsWithoutBlockers) > 0)

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

func packChecklistIDs(items []ReviewerChecklistItem) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ItemID)
	}
	return ids
}

func packDecisionIDs(items []ReviewDecision) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.DecisionID)
	}
	return ids
}

func packAssignmentIDs(items []ReviewRoleAssignment) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.AssignmentID)
	}
	return ids
}

func packSignoffIDs(items []ReviewSignoff) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.SignoffID)
	}
	return ids
}

func packBlockerIDs(items []ReviewBlocker) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.BlockerID)
	}
	return ids
}

func anyMissing(ids []string, index map[string]struct{}) bool {
	for _, id := range ids {
		if _, ok := index[id]; !ok {
			return true
		}
	}
	return false
}

func sliceSetMap(items []string) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, item := range items {
		out[item] = struct{}{}
	}
	return out
}

func sortAll(groups ...*[]string) {
	for _, group := range groups {
		slices.Sort(*group)
	}
}
