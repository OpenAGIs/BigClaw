package uireview

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func RenderDecisionLog(pack UIReviewPack) string {
	lines := []string{"# UI Review Decision Log", ""}
	for _, decision := range pack.DecisionLog {
		lines = append(lines,
			fmt.Sprintf("- %s: surface=%s owner=%s status=%s", decision.DecisionID, decision.SurfaceID, decision.Owner, decision.Status),
			fmt.Sprintf("  summary=%s rationale=%s follow_up=%s", decision.Summary, decision.Rationale, firstNonEmpty(decision.FollowUp, "none")),
		)
	}
	if len(pack.DecisionLog) == 0 {
		lines = append(lines, "- none")
	}
	return strings.Join(lines, "\n")
}

func RenderRoleMatrix(pack UIReviewPack) string {
	lines := []string{"# UI Review Role Matrix", ""}
	for _, assignment := range pack.RoleMatrix {
		lines = append(lines,
			fmt.Sprintf("- %s: surface=%s role=%s status=%s", assignment.AssignmentID, assignment.SurfaceID, assignment.Role, assignment.Status),
			fmt.Sprintf("  responsibilities=%s checklist=%s decisions=%s", joinOrNone(assignment.Responsibilities), joinOrNone(assignment.ChecklistItemIDs), joinOrNone(assignment.DecisionIDs)),
		)
	}
	if len(pack.RoleMatrix) == 0 {
		lines = append(lines, "- none")
	}
	return strings.Join(lines, "\n")
}

func RenderReviewSummaryBoard(pack UIReviewPack) string {
	objectiveCounts := map[string]int{}
	for _, entry := range buildObjectiveCoverageEntries(pack) {
		objectiveCounts[entry.Coverage]++
	}
	personaCounts := map[string]int{}
	for _, entry := range buildPersonaReadinessEntries(pack) {
		personaCounts[entry.Readiness]++
	}
	interactionCovered := 0
	for _, entry := range buildInteractionCoverageEntries(pack) {
		if entry.Coverage == "covered" {
			interactionCovered++
		}
	}
	linkedQuestions := 0
	for _, entry := range buildOpenQuestionEntries(pack) {
		if entry.LinkStatus == "linked" {
			linkedQuestions++
		}
	}
	queueItems := len(buildOwnerReviewQueueEntries(pack))
	reminderItems := len(buildReminderEntries(pack))
	renewalItems := len(buildFreezeRenewalEntries(pack))
	lines := []string{
		"# UI Review Review Summary Board",
		"",
		fmt.Sprintf("- Issue: %s %s", pack.IssueID, pack.Title),
		fmt.Sprintf("- Version: %s", pack.Version),
		"- Categories: 6",
		"",
		"## Entries",
		fmt.Sprintf("- summary-objectives: category=objectives total=%d blocked=%d at-risk=%d covered=%d", len(pack.Objectives), objectiveCounts["blocked"], objectiveCounts["at-risk"], objectiveCounts["covered"]),
		fmt.Sprintf("- summary-personas: category=personas total=%d blocked=%d at-risk=%d ready=%d", len(buildPersonaReadinessEntries(pack)), personaCounts["blocked"], personaCounts["at-risk"], personaCounts["ready"]),
		fmt.Sprintf("- summary-wireframes: category=wireframes total=%d blocked=%d at-risk=%d ready=%d", len(pack.Wireframes), countWireframeReadiness(pack, "blocked"), countWireframeReadiness(pack, "at-risk"), countWireframeReadiness(pack, "ready")),
		fmt.Sprintf("- summary-interactions: category=interactions total=%d covered=%d watch=%d missing=%d", len(pack.Interactions), interactionCovered, 0, len(pack.Interactions)-interactionCovered),
		fmt.Sprintf("- summary-questions: category=questions total=%d linked=%d orphan=%d owners=%d", len(pack.OpenQuestions), linkedQuestions, len(pack.OpenQuestions)-linkedQuestions, distinctCount(questionOwners(pack))),
		fmt.Sprintf("- summary-actions: category=actions total=%d queue=%d reminder=%d renewal=%d", queueItems+reminderItems+renewalItems, queueItems, reminderItems, renewalItems),
	}
	return strings.Join(lines, "\n")
}

type objectiveCoverageEntry struct {
	ID, ObjectiveID, Persona, Priority, Coverage, Surfaces, DependencyIDs, Assignments, Checklist, Decisions, Signoffs, Blockers, Summary string
	DependencyCount                                                                                                               int
}

func buildObjectiveCoverageEntries(pack UIReviewPack) []objectiveCoverageEntry {
	assignmentsByRole := map[string][]ReviewRoleAssignment{}
	for _, assignment := range pack.RoleMatrix {
		assignmentsByRole[assignment.Role] = append(assignmentsByRole[assignment.Role], assignment)
	}
	signoffIndex := signoffByAssignment(pack)
	blockerIndex := blockersBySignoff(pack)
	var entries []objectiveCoverageEntry
	for _, objective := range pack.Objectives {
		assignments := assignmentsByRole[objective.Persona]
		surfaceSet := map[string]struct{}{}
		var assignmentIDs, checklistIDs, decisionIDs, signoffIDs, blockerIDs []string
		coverage := "covered"
		for _, assignment := range assignments {
			surfaceSet[assignment.SurfaceID] = struct{}{}
			assignmentIDs = append(assignmentIDs, assignment.AssignmentID)
			checklistIDs = append(checklistIDs, assignment.ChecklistItemIDs...)
			decisionIDs = append(decisionIDs, assignment.DecisionIDs...)
			for _, decision := range pack.DecisionLog {
				for _, decisionID := range assignment.DecisionIDs {
					if decision.DecisionID == decisionID && isOpenStatus(decision.Status) && coverage == "covered" {
						coverage = "at-risk"
					}
				}
			}
			if signoff, ok := signoffIndex[assignment.AssignmentID]; ok {
				signoffIDs = append(signoffIDs, signoff.SignoffID)
				if isOpenStatus(signoff.Status) {
					coverage = "at-risk"
				}
				for _, blocker := range blockerIndex[signoff.SignoffID] {
					blockerIDs = append(blockerIDs, blocker.BlockerID)
					coverage = "blocked"
				}
			}
			if isOpenStatus(assignment.Status) && coverage == "covered" {
				coverage = "at-risk"
			}
		}
		var surfaces []string
		for surfaceID := range surfaceSet {
			surfaces = append(surfaces, surfaceID)
		}
		sort.Strings(surfaces)
		sort.Strings(assignmentIDs)
		sort.Strings(checklistIDs)
		sort.Strings(decisionIDs)
		sort.Strings(signoffIDs)
		sort.Strings(blockerIDs)
		entries = append(entries, objectiveCoverageEntry{
			ID:              "objcov-" + objective.ObjectiveID,
			ObjectiveID:     objective.ObjectiveID,
			Persona:         objective.Persona,
			Priority:        objective.Priority,
			Coverage:        coverage,
			Surfaces:        joinOrNone(surfaces),
			DependencyCount: len(objective.Dependencies),
			DependencyIDs:   joinOrNone(objective.Dependencies),
			Assignments:     joinOrNone(assignmentIDs),
			Checklist:       joinOrNone(checklistIDs),
			Decisions:       joinOrNone(decisionIDs),
			Signoffs:        joinOrNone(signoffIDs),
			Blockers:        joinOrNone(blockerIDs),
			Summary:         objective.SuccessSignal,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Coverage > entries[j].Coverage || entries[i].ObjectiveID < entries[j].ObjectiveID
	})
	return entries
}

func RenderObjectiveCoverageBoard(pack UIReviewPack) string {
	entries := buildObjectiveCoverageEntries(pack)
	personaCounts := map[string]int{}
	statusCounts := map[string]int{}
	for _, entry := range entries {
		personaCounts[entry.Persona]++
		statusCounts[entry.Coverage]++
	}
	lines := []string{
		"# UI Review Objective Coverage Board",
		"",
		fmt.Sprintf("- Issue: %s %s", pack.IssueID, pack.Title),
		fmt.Sprintf("- Version: %s", pack.Version),
		fmt.Sprintf("- Objectives: %d", len(entries)),
		fmt.Sprintf("- Personas: %d", len(personaCounts)),
		"",
		"## By Coverage Status",
	}
	for _, key := range sortStrings(mapKeys(statusCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, statusCounts[key]))
	}
	lines = append(lines, "", "## By Persona")
	for _, key := range sortStrings(mapKeys(personaCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, personaCounts[key]))
	}
	lines = append(lines, "", "## Entries")
	for _, entry := range entries {
		lines = append(lines,
			fmt.Sprintf("- %s: objective=%s persona=%s priority=%s coverage=%s dependencies=%d surfaces=%s", entry.ID, entry.ObjectiveID, entry.Persona, entry.Priority, entry.Coverage, entry.DependencyCount, entry.Surfaces),
			fmt.Sprintf("  dependency_ids=%s assignments=%s checklist=%s decisions=%s signoffs=%s blockers=%s summary=%s", entry.DependencyIDs, entry.Assignments, entry.Checklist, entry.Decisions, entry.Signoffs, entry.Blockers, entry.Summary),
		)
	}
	return strings.Join(lines, "\n")
}

type personaReadinessEntry struct {
	ID, Persona, Readiness, ObjectiveIDs, Surfaces, QueueIDs, BlockerIDs string
	ObjectiveCount, AssignmentCount, SignoffCount, QuestionCount, QueueCount, BlockerCount int
}

func buildPersonaReadinessEntries(pack UIReviewPack) []personaReadinessEntry {
	assignmentsByRole := map[string][]ReviewRoleAssignment{}
	signoffsByRole := map[string][]ReviewSignoff{}
	objectivesByPersona := map[string][]ReviewObjective{}
	queueByOwner := map[string][]ownerQueueEntry{}
	for _, assignment := range pack.RoleMatrix {
		assignmentsByRole[assignment.Role] = append(assignmentsByRole[assignment.Role], assignment)
	}
	for _, signoff := range pack.SignoffLog {
		signoffsByRole[signoff.Role] = append(signoffsByRole[signoff.Role], signoff)
	}
	for _, objective := range pack.Objectives {
		objectivesByPersona[objective.Persona] = append(objectivesByPersona[objective.Persona], objective)
	}
	for _, item := range buildOwnerReviewQueueEntries(pack) {
		queueByOwner[item.Owner] = append(queueByOwner[item.Owner], item)
	}
	blockers := pack.BlockerLog
	var entries []personaReadinessEntry
	for persona, objectives := range objectivesByPersona {
		surfaceSet := map[string]struct{}{}
		var objectiveIDs, queueIDs, blockerIDs []string
		readiness := "ready"
		for _, objective := range objectives {
			objectiveIDs = append(objectiveIDs, objective.ObjectiveID)
		}
		for _, assignment := range assignmentsByRole[persona] {
			surfaceSet[assignment.SurfaceID] = struct{}{}
			if isOpenStatus(assignment.Status) {
				readiness = "at-risk"
			}
			for _, decision := range pack.DecisionLog {
				for _, decisionID := range assignment.DecisionIDs {
					if decision.DecisionID == decisionID && isOpenStatus(decision.Status) {
						readiness = "at-risk"
					}
				}
			}
		}
		for _, signoff := range signoffsByRole[persona] {
			surfaceSet[signoff.SurfaceID] = struct{}{}
			if isOpenStatus(signoff.Status) {
				readiness = "at-risk"
			}
		}
		for _, item := range queueByOwner[persona] {
			queueIDs = append(queueIDs, item.ID)
		}
		for _, blocker := range blockers {
			if blocker.Owner == persona || blocker.SignoffID == firstSignoffIDForRole(signoffsByRole[persona]) {
				blockerIDs = append(blockerIDs, blocker.BlockerID)
				readiness = "blocked"
			}
		}
		var surfaces []string
		for surfaceID := range surfaceSet {
			surfaces = append(surfaces, surfaceID)
		}
		sort.Strings(objectiveIDs)
		sort.Strings(queueIDs)
		sort.Strings(blockerIDs)
		sort.Strings(surfaces)
		entries = append(entries, personaReadinessEntry{
			ID:              "persona-" + slugify(persona),
			Persona:         persona,
			Readiness:       readiness,
			ObjectiveIDs:    joinOrNone(objectiveIDs),
			Surfaces:        joinOrNone(surfaces),
			QueueIDs:        joinOrNone(queueIDs),
			BlockerIDs:      joinOrNone(blockerIDs),
			ObjectiveCount:  len(objectives),
			AssignmentCount: len(assignmentsByRole[persona]),
			SignoffCount:    len(signoffsByRole[persona]),
			QuestionCount:   0,
			QueueCount:      len(queueIDs),
			BlockerCount:    len(blockerIDs),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Readiness != entries[j].Readiness {
			return entries[i].Readiness > entries[j].Readiness
		}
		return entries[i].Persona < entries[j].Persona
	})
	return entries
}

func RenderPersonaReadinessBoard(pack UIReviewPack) string {
	entries := buildPersonaReadinessEntries(pack)
	counts := map[string]int{}
	for _, entry := range entries {
		counts[entry.Readiness]++
	}
	lines := []string{
		"# UI Review Persona Readiness Board",
		"",
		fmt.Sprintf("- Issue: %s %s", pack.IssueID, pack.Title),
		fmt.Sprintf("- Version: %s", pack.Version),
		fmt.Sprintf("- Personas: %d", len(entries)),
		fmt.Sprintf("- Objectives: %d", len(pack.Objectives)),
		"",
		"## By Readiness",
	}
	for _, key := range sortStrings(mapKeys(counts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, counts[key]))
	}
	lines = append(lines, "", "## Entries")
	for _, entry := range entries {
		lines = append(lines,
			fmt.Sprintf("- %s: persona=%s readiness=%s objectives=%d assignments=%d signoffs=%d open_questions=%d queue_items=%d blockers=%d", entry.ID, entry.Persona, entry.Readiness, entry.ObjectiveCount, entry.AssignmentCount, entry.SignoffCount, entry.QuestionCount, entry.QueueCount, entry.BlockerCount),
			fmt.Sprintf("  objective_ids=%s surfaces=%s queue_ids=%s blocker_ids=%s", entry.ObjectiveIDs, entry.Surfaces, entry.QueueIDs, entry.BlockerIDs),
		)
	}
	return strings.Join(lines, "\n")
}

func countWireframeReadiness(pack UIReviewPack, state string) int {
	count := 0
	for _, entry := range buildWireframeReadinessEntries(pack) {
		if entry.Readiness == state {
			count++
		}
	}
	return count
}

type wireframeReadinessEntry struct {
	ID, SurfaceID, Device, Readiness, EntryPoint, Signoffs, Blockers, Summary string
	OpenTotal, ChecklistOpen, DecisionsOpen, AssignmentsOpen, SignoffsOpen, BlockersOpen, BlockCount, NoteCount int
}

func buildWireframeReadinessEntries(pack UIReviewPack) []wireframeReadinessEntry {
	var entries []wireframeReadinessEntry
	for _, wireframe := range pack.Wireframes {
		checklistOpen, decisionsOpen, assignmentsOpen, signoffsOpen, blockersOpen := 0, 0, 0, 0, 0
		var signoffIDs, blockerIDs []string
		for _, item := range pack.ReviewerChecklist {
			if item.SurfaceID == wireframe.SurfaceID && isOpenStatus(item.Status) {
				checklistOpen++
			}
		}
		for _, decision := range pack.DecisionLog {
			if decision.SurfaceID == wireframe.SurfaceID && isOpenStatus(decision.Status) {
				decisionsOpen++
			}
		}
		for _, assignment := range pack.RoleMatrix {
			if assignment.SurfaceID == wireframe.SurfaceID && isOpenStatus(assignment.Status) {
				assignmentsOpen++
			}
		}
		for _, signoff := range pack.SignoffLog {
			if signoff.SurfaceID == wireframe.SurfaceID {
				signoffIDs = append(signoffIDs, signoff.SignoffID)
				if isOpenStatus(signoff.Status) {
					signoffsOpen++
				}
			}
		}
		for _, blocker := range pack.BlockerLog {
			if blocker.SurfaceID == wireframe.SurfaceID {
				blockerIDs = append(blockerIDs, blocker.BlockerID)
				if isOpenStatus(blocker.Status) {
					blockersOpen++
				}
			}
		}
		readiness := "ready"
		if blockersOpen > 0 {
			readiness = "blocked"
		} else if checklistOpen+decisionsOpen+assignmentsOpen+signoffsOpen > 0 {
			readiness = "at-risk"
		}
		entries = append(entries, wireframeReadinessEntry{
			ID:             "wire-" + wireframe.SurfaceID,
			SurfaceID:      wireframe.SurfaceID,
			Device:         wireframe.Device,
			Readiness:      readiness,
			EntryPoint:     wireframe.EntryPoint,
			OpenTotal:      checklistOpen + decisionsOpen + assignmentsOpen + signoffsOpen + blockersOpen,
			ChecklistOpen:  checklistOpen,
			DecisionsOpen:  decisionsOpen,
			AssignmentsOpen: assignmentsOpen,
			SignoffsOpen:   signoffsOpen,
			BlockersOpen:   blockersOpen,
			Signoffs:       joinOrNone(sortStrings(signoffIDs)),
			Blockers:       joinOrNone(sortStrings(blockerIDs)),
			BlockCount:     len(wireframe.PrimaryBlocks),
			NoteCount:      len(wireframe.ReviewNotes),
			Summary:        wireframe.Name,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Readiness != entries[j].Readiness {
			return entries[i].Readiness > entries[j].Readiness
		}
		return entries[i].SurfaceID < entries[j].SurfaceID
	})
	return entries
}

func RenderWireframeReadinessBoard(pack UIReviewPack) string {
	entries := buildWireframeReadinessEntries(pack)
	readinessCounts := map[string]int{}
	deviceCounts := map[string]int{}
	for _, entry := range entries {
		readinessCounts[entry.Readiness]++
		deviceCounts[entry.Device]++
	}
	lines := []string{
		"# UI Review Wireframe Readiness Board",
		"",
		fmt.Sprintf("- Issue: %s %s", pack.IssueID, pack.Title),
		fmt.Sprintf("- Version: %s", pack.Version),
		fmt.Sprintf("- Wireframes: %d", len(entries)),
		fmt.Sprintf("- Devices: %d", len(deviceCounts)),
		"",
		"## By Readiness",
	}
	for _, key := range sortStrings(mapKeys(readinessCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, readinessCounts[key]))
	}
	lines = append(lines, "", "## By Device")
	for _, key := range sortStrings(mapKeys(deviceCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, deviceCounts[key]))
	}
	lines = append(lines, "", "## Entries")
	for _, entry := range entries {
		lines = append(lines,
			fmt.Sprintf("- %s: surface=%s device=%s readiness=%s open_total=%d entry=%s", entry.ID, entry.SurfaceID, entry.Device, entry.Readiness, entry.OpenTotal, entry.EntryPoint),
			fmt.Sprintf("  checklist_open=%d decisions_open=%d assignments_open=%d signoffs_open=%d blockers_open=%d signoffs=%s blockers=%s blocks=%d notes=%d summary=%s", entry.ChecklistOpen, entry.DecisionsOpen, entry.AssignmentsOpen, entry.SignoffsOpen, entry.BlockersOpen, entry.Signoffs, entry.Blockers, entry.BlockCount, entry.NoteCount, entry.Summary),
		)
	}
	return strings.Join(lines, "\n")
}

type interactionCoverageEntry struct {
	ID, FlowID, Surfaces, Owners, Coverage, Checklist, OpenChecklist, Trigger string
	StateCount, ExceptionCount int
}

func buildInteractionCoverageEntries(pack UIReviewPack) []interactionCoverageEntry {
	checklistsByFlow := map[string][]ReviewerChecklistItem{}
	for _, item := range pack.ReviewerChecklist {
		for _, link := range item.EvidenceLinks {
			if strings.HasPrefix(link, "flow-") {
				checklistsByFlow[link] = append(checklistsByFlow[link], item)
			}
		}
	}
	var entries []interactionCoverageEntry
	for _, flow := range pack.Interactions {
		items := checklistsByFlow[flow.FlowID]
		ownerSet := map[string]struct{}{}
		surfaceSet := map[string]struct{}{}
		var checklistIDs, openChecklistIDs []string
		for _, item := range items {
			ownerSet[item.Owner] = struct{}{}
			surfaceSet[item.SurfaceID] = struct{}{}
			checklistIDs = append(checklistIDs, item.ItemID)
			if isOpenStatus(item.Status) {
				openChecklistIDs = append(openChecklistIDs, item.ItemID)
			}
		}
		entries = append(entries, interactionCoverageEntry{
			ID:           "intcov-" + flow.FlowID,
			FlowID:       flow.FlowID,
			Surfaces:     joinOrNone(sortStrings(setKeys(surfaceSet))),
			Owners:       joinOrNone(sortStrings(setKeys(ownerSet))),
			Coverage:     "covered",
			Checklist:    joinOrNone(sortStrings(checklistIDs)),
			OpenChecklist: joinOrNone(sortStrings(openChecklistIDs)),
			Trigger:      flow.Trigger,
			StateCount:   len(flow.States),
			ExceptionCount: len(flow.Exceptions),
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].FlowID < entries[j].FlowID })
	return entries
}

func RenderInteractionCoverageBoard(pack UIReviewPack) string {
	entries := buildInteractionCoverageEntries(pack)
	coverageCounts := map[string]int{}
	surfaceCounts := map[string]int{}
	for _, entry := range entries {
		coverageCounts[entry.Coverage]++
		for _, surfaceID := range strings.Split(entry.Surfaces, ",") {
			if surfaceID != "" && surfaceID != "none" {
				surfaceCounts[surfaceID]++
			}
		}
	}
	lines := []string{
		"# UI Review Interaction Coverage Board",
		"",
		fmt.Sprintf("- Issue: %s %s", pack.IssueID, pack.Title),
		fmt.Sprintf("- Version: %s", pack.Version),
		fmt.Sprintf("- Interactions: %d", len(entries)),
		fmt.Sprintf("- Surfaces: %d", len(surfaceCounts)),
		"",
		"## By Coverage Status",
	}
	for _, key := range sortStrings(mapKeys(coverageCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, coverageCounts[key]))
	}
	lines = append(lines, "", "## By Surface")
	for _, key := range sortStrings(mapKeys(surfaceCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, surfaceCounts[key]))
	}
	lines = append(lines, "", "## Entries")
	for _, entry := range entries {
		lines = append(lines,
			fmt.Sprintf("- %s: flow=%s surfaces=%s owners=%s coverage=%s states=%d exceptions=%d", entry.ID, entry.FlowID, entry.Surfaces, entry.Owners, entry.Coverage, entry.StateCount, entry.ExceptionCount),
			fmt.Sprintf("  checklist=%s open_checklist=%s trigger=%s", entry.Checklist, entry.OpenChecklist, entry.Trigger),
		)
	}
	return strings.Join(lines, "\n")
}

type openQuestionEntry struct {
	ID, QuestionID, Owner, Theme, Status, LinkStatus, Surfaces, Checklist, Flows, Impact, Prompt string
}

func buildOpenQuestionEntries(pack UIReviewPack) []openQuestionEntry {
	var entries []openQuestionEntry
	for _, question := range pack.OpenQuestions {
		var surfaces, checklist []string
		for _, item := range pack.ReviewerChecklist {
			for _, link := range item.EvidenceLinks {
				if link == question.QuestionID {
					checklist = append(checklist, item.ItemID)
					surfaces = append(surfaces, item.SurfaceID)
				}
			}
		}
		entries = append(entries, openQuestionEntry{
			ID:         "qtrack-" + question.QuestionID,
			QuestionID: question.QuestionID,
			Owner:      question.Owner,
			Theme:      question.Theme,
			Status:     question.Status,
			LinkStatus: map[bool]string{true: "linked", false: "orphan"}[len(checklist) > 0],
			Surfaces:   joinOrNone(sortStrings(uniqueStrings(surfaces))),
			Checklist:  joinOrNone(sortStrings(checklist)),
			Flows:      "none",
			Impact:     question.Impact,
			Prompt:     question.Question,
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].QuestionID < entries[j].QuestionID })
	return entries
}

func RenderOpenQuestionTracker(pack UIReviewPack) string {
	entries := buildOpenQuestionEntries(pack)
	ownerCounts := map[string]int{}
	themeCounts := map[string]int{}
	for _, entry := range entries {
		ownerCounts[entry.Owner]++
		themeCounts[entry.Theme]++
	}
	lines := []string{
		"# UI Review Open Question Tracker",
		"",
		fmt.Sprintf("- Issue: %s %s", pack.IssueID, pack.Title),
		fmt.Sprintf("- Version: %s", pack.Version),
		fmt.Sprintf("- Questions: %d", len(entries)),
		fmt.Sprintf("- Owners: %d", len(ownerCounts)),
		"",
		"## By Owner",
	}
	for _, key := range sortStrings(mapKeys(ownerCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, ownerCounts[key]))
	}
	lines = append(lines, "", "## By Theme")
	for _, key := range sortStrings(mapKeys(themeCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, themeCounts[key]))
	}
	lines = append(lines, "", "## Entries")
	for _, entry := range entries {
		lines = append(lines,
			fmt.Sprintf("- %s: question=%s owner=%s theme=%s status=%s link_status=%s surfaces=%s", entry.ID, entry.QuestionID, entry.Owner, entry.Theme, entry.Status, entry.LinkStatus, entry.Surfaces),
			fmt.Sprintf("  checklist=%s flows=%s impact=%s prompt=%s", entry.Checklist, entry.Flows, entry.Impact, entry.Prompt),
		)
	}
	return strings.Join(lines, "\n")
}

type checklistTraceEntry struct {
	ID, ItemID, SurfaceID, Owner, Status, LinkedRoles, LinkedAssignments, LinkedDecisions, Evidence, Summary string
}

func buildChecklistTraceabilityEntries(pack UIReviewPack) []checklistTraceEntry {
	var entries []checklistTraceEntry
	for _, item := range pack.ReviewerChecklist {
		var roles, assignments, decisions []string
		for _, assignment := range pack.RoleMatrix {
			for _, checklistID := range assignment.ChecklistItemIDs {
				if checklistID == item.ItemID {
					roles = append(roles, assignment.Role)
					assignments = append(assignments, assignment.AssignmentID)
					decisions = append(decisions, assignment.DecisionIDs...)
					break
				}
			}
		}
		summary := item.Prompt
		if strings.TrimSpace(item.Notes) != "" {
			summary = item.Notes
		}
		entries = append(entries, checklistTraceEntry{
			ID:                "trace-" + item.ItemID,
			ItemID:            item.ItemID,
			SurfaceID:         item.SurfaceID,
			Owner:             item.Owner,
			Status:            item.Status,
			LinkedRoles:       joinOrNone(sortStrings(uniqueStrings(roles))),
			LinkedAssignments: joinOrNone(sortStrings(uniqueStrings(assignments))),
			LinkedDecisions:   joinOrNone(sortStrings(uniqueStrings(decisions))),
			Evidence:          joinOrNone(item.EvidenceLinks),
			Summary:           summary,
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ItemID < entries[j].ItemID })
	return entries
}

func RenderChecklistTraceabilityBoard(pack UIReviewPack) string {
	entries := buildChecklistTraceabilityEntries(pack)
	ownerCounts := map[string]int{}
	statusCounts := map[string]int{}
	for _, entry := range entries {
		ownerCounts[entry.Owner]++
		statusCounts[entry.Status]++
	}
	lines := []string{
		"# UI Review Checklist Traceability Board",
		"",
		fmt.Sprintf("- Issue: %s %s", pack.IssueID, pack.Title),
		fmt.Sprintf("- Version: %s", pack.Version),
		fmt.Sprintf("- Checklist items: %d", len(entries)),
		fmt.Sprintf("- Owners: %d", len(ownerCounts)),
		"",
		"## By Owner",
	}
	for _, key := range sortStrings(mapKeys(ownerCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, ownerCounts[key]))
	}
	lines = append(lines, "", "## By Status")
	for _, key := range sortStrings(mapKeys(statusCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, statusCounts[key]))
	}
	lines = append(lines, "", "## Entries")
	for _, entry := range entries {
		lines = append(lines,
			fmt.Sprintf("- %s: item=%s surface=%s owner=%s status=%s linked_roles=%s", entry.ID, entry.ItemID, entry.SurfaceID, entry.Owner, entry.Status, entry.LinkedRoles),
			fmt.Sprintf("  linked_assignments=%s linked_decisions=%s evidence=%s summary=%s", entry.LinkedAssignments, entry.LinkedDecisions, entry.Evidence, entry.Summary),
		)
	}
	return strings.Join(lines, "\n")
}

type decisionFollowupEntry struct {
	ID, DecisionID, SurfaceID, Owner, Status, LinkedRoles, LinkedAssignments, LinkedChecklists, FollowUp, Summary string
}

func buildDecisionFollowupEntries(pack UIReviewPack) []decisionFollowupEntry {
	var entries []decisionFollowupEntry
	for _, decision := range pack.DecisionLog {
		var roles, assignments, checklists []string
		for _, assignment := range pack.RoleMatrix {
			for _, decisionID := range assignment.DecisionIDs {
				if decisionID == decision.DecisionID {
					roles = append(roles, assignment.Role)
					assignments = append(assignments, assignment.AssignmentID)
					checklists = append(checklists, assignment.ChecklistItemIDs...)
					break
				}
			}
		}
		entries = append(entries, decisionFollowupEntry{
			ID:                "follow-" + decision.DecisionID,
			DecisionID:        decision.DecisionID,
			SurfaceID:         decision.SurfaceID,
			Owner:             decision.Owner,
			Status:            decision.Status,
			LinkedRoles:       joinOrNone(sortStrings(uniqueStrings(roles))),
			LinkedAssignments: joinOrNone(sortStrings(uniqueStrings(assignments))),
			LinkedChecklists:  joinOrNone(sortStrings(uniqueStrings(checklists))),
			FollowUp:          firstNonEmpty(decision.FollowUp, "none"),
			Summary:           decision.Summary,
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].DecisionID < entries[j].DecisionID })
	return entries
}

func RenderDecisionFollowupTracker(pack UIReviewPack) string {
	entries := buildDecisionFollowupEntries(pack)
	ownerCounts := map[string]int{}
	statusCounts := map[string]int{}
	for _, entry := range entries {
		ownerCounts[entry.Owner]++
		statusCounts[entry.Status]++
	}
	lines := []string{
		"# UI Review Decision Follow-up Tracker",
		"",
		fmt.Sprintf("- Issue: %s %s", pack.IssueID, pack.Title),
		fmt.Sprintf("- Version: %s", pack.Version),
		fmt.Sprintf("- Decisions: %d", len(entries)),
		fmt.Sprintf("- Owners: %d", len(ownerCounts)),
		"",
		"## By Owner",
	}
	for _, key := range sortStrings(mapKeys(ownerCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, ownerCounts[key]))
	}
	lines = append(lines, "", "## By Status")
	for _, key := range sortStrings(mapKeys(statusCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, statusCounts[key]))
	}
	lines = append(lines, "", "## Entries")
	for _, entry := range entries {
		lines = append(lines,
			fmt.Sprintf("- %s: decision=%s surface=%s owner=%s status=%s linked_roles=%s", entry.ID, entry.DecisionID, entry.SurfaceID, entry.Owner, entry.Status, entry.LinkedRoles),
			fmt.Sprintf("  linked_assignments=%s linked_checklists=%s follow_up=%s summary=%s", entry.LinkedAssignments, entry.LinkedChecklists, entry.FollowUp, entry.Summary),
		)
	}
	return strings.Join(lines, "\n")
}

type roleCoverageEntry struct {
	ID, AssignmentID, SurfaceID, Role, Status, SignoffID, SignoffStatus, Summary string
	ResponsibilityCount, ChecklistCount, DecisionCount int
}

func buildRoleCoverageEntries(pack UIReviewPack) []roleCoverageEntry {
	signoffIndex := signoffByAssignment(pack)
	var entries []roleCoverageEntry
	for _, assignment := range pack.RoleMatrix {
		signoff, ok := signoffIndex[assignment.AssignmentID]
		signoffID := "none"
		signoffStatus := "none"
		if ok {
			signoffID = signoff.SignoffID
			signoffStatus = signoff.Status
		}
		entries = append(entries, roleCoverageEntry{
			ID:                  "cover-" + assignment.AssignmentID,
			AssignmentID:        assignment.AssignmentID,
			SurfaceID:           assignment.SurfaceID,
			Role:                assignment.Role,
			Status:              assignment.Status,
			SignoffID:           signoffID,
			SignoffStatus:       signoffStatus,
			Summary:             joinOrNone(assignment.Responsibilities),
			ResponsibilityCount: len(assignment.Responsibilities),
			ChecklistCount:      len(assignment.ChecklistItemIDs),
			DecisionCount:       len(assignment.DecisionIDs),
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].AssignmentID < entries[j].AssignmentID })
	return entries
}

func RenderRoleCoverageBoard(pack UIReviewPack) string {
	entries := buildRoleCoverageEntries(pack)
	surfaceCounts := map[string]int{}
	statusCounts := map[string]int{}
	for _, entry := range entries {
		surfaceCounts[entry.SurfaceID]++
		statusCounts[entry.Status]++
	}
	lines := []string{
		"# UI Review Role Coverage Board",
		"",
		fmt.Sprintf("- Issue: %s %s", pack.IssueID, pack.Title),
		fmt.Sprintf("- Version: %s", pack.Version),
		fmt.Sprintf("- Assignments: %d", len(entries)),
		fmt.Sprintf("- Surfaces: %d", len(surfaceCounts)),
		"",
		"## By Surface",
	}
	for _, key := range sortStrings(mapKeys(surfaceCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, surfaceCounts[key]))
	}
	lines = append(lines, "", "## By Status")
	for _, key := range sortStrings(mapKeys(statusCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, statusCounts[key]))
	}
	lines = append(lines, "", "## Entries")
	for _, entry := range entries {
		lines = append(lines,
			fmt.Sprintf("- %s: assignment=%s surface=%s role=%s status=%s responsibilities=%d checklist=%d decisions=%d", entry.ID, entry.AssignmentID, entry.SurfaceID, entry.Role, entry.Status, entry.ResponsibilityCount, entry.ChecklistCount, entry.DecisionCount),
			fmt.Sprintf("  signoff=%s signoff_status=%s summary=%s", entry.SignoffID, entry.SignoffStatus, entry.Summary),
		)
	}
	return strings.Join(lines, "\n")
}

type signoffDependencyEntry struct {
	ID, SignoffID, SurfaceID, Role, Status, DependencyStatus, Blockers, Assignment, Checklist, Decisions, LatestBlockerEvent, SLA, DueAt, Cadence, Summary string
}

func buildSignoffDependencyEntries(pack UIReviewPack) []signoffDependencyEntry {
	assignmentMap := map[string]ReviewRoleAssignment{}
	for _, assignment := range pack.RoleMatrix {
		assignmentMap[assignment.AssignmentID] = assignment
	}
	blockersIndex := blockersBySignoff(pack)
	latestEvents := latestBlockerEvents(pack)
	var entries []signoffDependencyEntry
	for _, signoff := range pack.SignoffLog {
		assignment := assignmentMap[signoff.AssignmentID]
		var blockerIDs []string
		latest := "none"
		for _, blocker := range blockersIndex[signoff.SignoffID] {
			blockerIDs = append(blockerIDs, blocker.BlockerID)
			if event, ok := latestEvents[blocker.BlockerID]; ok {
				latest = fmt.Sprintf("%s/%s/%s@%s", event.EventID, event.Status, event.Actor, event.Timestamp)
			}
		}
		dependencyStatus := "clear"
		if len(blockerIDs) > 0 {
			dependencyStatus = "blocked"
		}
		entries = append(entries, signoffDependencyEntry{
			ID:               "dep-" + signoff.SignoffID,
			SignoffID:        signoff.SignoffID,
			SurfaceID:        signoff.SurfaceID,
			Role:             signoff.Role,
			Status:           signoff.Status,
			DependencyStatus: dependencyStatus,
			Blockers:         joinOrNone(sortStrings(blockerIDs)),
			Assignment:       signoff.AssignmentID,
			Checklist:        joinOrNone(assignment.ChecklistItemIDs),
			Decisions:        joinOrNone(assignment.DecisionIDs),
			LatestBlockerEvent: latest,
			SLA:              firstNonEmpty(signoff.SLAStatus, "on-track"),
			DueAt:            firstNonEmpty(signoff.DueAt, "none"),
			Cadence:          firstNonEmpty(signoff.ReminderCadence, "none"),
			Summary:          signoff.Notes,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].DependencyStatus != entries[j].DependencyStatus {
			return entries[i].DependencyStatus > entries[j].DependencyStatus
		}
		return entries[i].SignoffID < entries[j].SignoffID
	})
	return entries
}

func RenderSignoffDependencyBoard(pack UIReviewPack) string {
	entries := buildSignoffDependencyEntries(pack)
	dependencyCounts := map[string]int{}
	slaCounts := map[string]int{}
	for _, entry := range entries {
		dependencyCounts[entry.DependencyStatus]++
		slaCounts[entry.SLA]++
	}
	lines := []string{
		"# UI Review Signoff Dependency Board",
		"",
		fmt.Sprintf("- Issue: %s %s", pack.IssueID, pack.Title),
		fmt.Sprintf("- Version: %s", pack.Version),
		fmt.Sprintf("- Sign-offs: %d", len(entries)),
		fmt.Sprintf("- Dependency states: %d", len(dependencyCounts)),
		"",
		"## By Dependency Status",
	}
	for _, key := range sortStrings(mapKeys(dependencyCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, dependencyCounts[key]))
	}
	lines = append(lines, "", "## By SLA State")
	for _, key := range sortStrings(mapKeys(slaCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, slaCounts[key]))
	}
	lines = append(lines, "", "## Entries")
	for _, entry := range entries {
		lines = append(lines,
			fmt.Sprintf("- %s: signoff=%s surface=%s role=%s status=%s dependency_status=%s blockers=%s", entry.ID, entry.SignoffID, entry.SurfaceID, entry.Role, entry.Status, entry.DependencyStatus, entry.Blockers),
			fmt.Sprintf("  assignment=%s checklist=%s decisions=%s latest_blocker_event=%s sla=%s due_at=%s cadence=%s summary=%s", entry.Assignment, entry.Checklist, entry.Decisions, entry.LatestBlockerEvent, entry.SLA, entry.DueAt, entry.Cadence, entry.Summary),
		)
	}
	return strings.Join(lines, "\n")
}

func RenderSignoffLog(pack UIReviewPack) string {
	lines := []string{"# UI Review Sign-off Log", ""}
	for _, signoff := range pack.SignoffLog {
		lines = append(lines,
			fmt.Sprintf("- %s: surface=%s role=%s assignment=%s status=%s", signoff.SignoffID, signoff.SurfaceID, signoff.Role, signoff.AssignmentID, signoff.Status),
			fmt.Sprintf("  required=%s evidence=%s notes=%s waiver_owner=%s waiver_reason=%s requested_at=%s due_at=%s escalation_owner=%s sla_status=%s reminder_owner=%s reminder_channel=%s last_reminder_at=%s next_reminder_at=%s", yesNo(signoff.Required), joinOrNone(signoff.EvidenceLinks), firstNonEmpty(signoff.Notes, "none"), firstNonEmpty(signoff.WaiverOwner, "none"), firstNonEmpty(signoff.WaiverReason, "none"), firstNonEmpty(signoff.RequestedAt, "none"), firstNonEmpty(signoff.DueAt, "none"), firstNonEmpty(signoff.EscalationOwner, "none"), firstNonEmpty(signoff.SLAStatus, "on-track"), firstNonEmpty(signoff.ReminderOwner, "none"), firstNonEmpty(signoff.ReminderChannel, "none"), firstNonEmpty(signoff.LastReminderAt, "none"), firstNonEmpty(signoff.NextReminderAt, "none")),
		)
	}
	return strings.Join(lines, "\n")
}

type reminderEntry struct {
	ID, SignoffID, Role, SurfaceID, Status, SLA, Owner, Channel, LastReminderAt, NextReminderAt, DueAt, Summary, Cadence string
}

func buildReminderEntries(pack UIReviewPack) []reminderEntry {
	var entries []reminderEntry
	for _, signoff := range pack.SignoffLog {
		if !signoff.Required || !isOpenStatus(signoff.Status) {
			continue
		}
		entries = append(entries, reminderEntry{
			ID:             "rem-" + signoff.SignoffID,
			SignoffID:      signoff.SignoffID,
			Role:           signoff.Role,
			SurfaceID:      signoff.SurfaceID,
			Status:         signoff.Status,
			SLA:            firstNonEmpty(signoff.SLAStatus, "on-track"),
			Owner:          firstNonEmpty(signoff.ReminderOwner, "none"),
			Channel:        firstNonEmpty(signoff.ReminderChannel, "none"),
			LastReminderAt: firstNonEmpty(signoff.LastReminderAt, "none"),
			NextReminderAt: firstNonEmpty(signoff.NextReminderAt, "none"),
			DueAt:          firstNonEmpty(signoff.DueAt, "none"),
			Summary:        signoff.Notes,
			Cadence:        firstNonEmpty(signoff.ReminderCadence, "none"),
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].SignoffID < entries[j].SignoffID })
	return entries
}

func RenderSignoffSLADashboard(pack UIReviewPack) string {
	slaCounts := map[string]int{}
	ownerCounts := map[string]int{}
	for _, signoff := range pack.SignoffLog {
		slaCounts[firstNonEmpty(signoff.SLAStatus, "on-track")]++
		ownerCounts[firstNonEmpty(signoff.EscalationOwner, "none")]++
	}
	lines := []string{
		"# UI Review Sign-off SLA Dashboard",
		"",
		fmt.Sprintf("- Sign-offs: %d", len(pack.SignoffLog)),
		fmt.Sprintf("- Escalation owners: %d", len(ownerCounts)),
		"",
		"## SLA States",
	}
	for _, key := range sortStrings(mapKeys(slaCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, slaCounts[key]))
	}
	lines = append(lines, "", "## Escalation Owners")
	for _, key := range sortStrings(mapKeys(ownerCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, ownerCounts[key]))
	}
	lines = append(lines, "", "## Sign-offs")
	for _, signoff := range pack.SignoffLog {
		lines = append(lines,
			fmt.Sprintf("- %s: role=%s surface=%s status=%s sla=%s requested_at=%s due_at=%s escalation_owner=%s", signoff.SignoffID, signoff.Role, signoff.SurfaceID, signoff.Status, firstNonEmpty(signoff.SLAStatus, "on-track"), firstNonEmpty(signoff.RequestedAt, "none"), firstNonEmpty(signoff.DueAt, "none"), firstNonEmpty(signoff.EscalationOwner, "none")),
			fmt.Sprintf("  required=%s evidence=%s", yesNo(signoff.Required), joinOrNone(signoff.EvidenceLinks)),
		)
	}
	return strings.Join(lines, "\n")
}

func RenderSignoffReminderQueue(pack UIReviewPack) string {
	entries := buildReminderEntries(pack)
	ownerCounts := map[string]int{}
	channelCounts := map[string]int{}
	for _, entry := range entries {
		ownerCounts[entry.Owner]++
		channelCounts[entry.Channel]++
	}
	lines := []string{
		"# UI Review Sign-off Reminder Queue",
		"",
		fmt.Sprintf("- Reminders: %d", len(entries)),
		fmt.Sprintf("- Owners: %d", len(ownerCounts)),
		"",
		"## By Owner",
	}
	for _, key := range sortStrings(mapKeys(ownerCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: reminders=%d", key, ownerCounts[key]))
	}
	lines = append(lines, "", "## By Channel")
	for _, key := range sortStrings(mapKeys(channelCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, channelCounts[key]))
	}
	lines = append(lines, "", "## Items")
	for _, entry := range entries {
		lines = append(lines,
			fmt.Sprintf("- %s: signoff=%s role=%s surface=%s status=%s sla=%s owner=%s channel=%s", entry.ID, entry.SignoffID, entry.Role, entry.SurfaceID, entry.Status, entry.SLA, entry.Owner, entry.Channel),
			fmt.Sprintf("  last_reminder_at=%s next_reminder_at=%s due_at=%s summary=%s", entry.LastReminderAt, entry.NextReminderAt, entry.DueAt, entry.Summary),
		)
	}
	return strings.Join(lines, "\n")
}

func RenderReminderCadenceBoard(pack UIReviewPack) string {
	entries := buildReminderEntries(pack)
	cadenceCounts := map[string]int{}
	statusCounts := map[string]int{}
	lines := []string{
		"# UI Review Reminder Cadence Board",
		"",
		fmt.Sprintf("- Items: %d", len(entries)),
	}
	for _, entry := range entries {
		cadenceCounts[entry.Cadence]++
		statusCounts["scheduled"]++
	}
	lines = append(lines, fmt.Sprintf("- Cadences: %d", len(cadenceCounts)), "", "## By Cadence")
	for _, key := range sortStrings(mapKeys(cadenceCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, cadenceCounts[key]))
	}
	lines = append(lines, "", "## By Status")
	for _, key := range sortStrings(mapKeys(statusCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, statusCounts[key]))
	}
	lines = append(lines, "", "## Items")
	for _, entry := range entries {
		lines = append(lines,
			fmt.Sprintf("- cad-%s: signoff=%s role=%s surface=%s cadence=%s status=scheduled owner=%s", entry.ID, entry.SignoffID, entry.Role, entry.SurfaceID, entry.Cadence, entry.Owner),
			fmt.Sprintf("  sla=%s last_reminder_at=%s next_reminder_at=%s due_at=%s summary=%s", entry.SLA, entry.LastReminderAt, entry.NextReminderAt, entry.DueAt, entry.Summary),
		)
	}
	return strings.Join(lines, "\n")
}

func RenderSignoffBreachBoard(pack UIReviewPack) string {
	var entries []ReviewSignoff
	ownerCounts := map[string]int{}
	slaCounts := map[string]int{}
	for _, signoff := range pack.SignoffLog {
		if !signoff.Required || !isOpenStatus(signoff.Status) {
			continue
		}
		entries = append(entries, signoff)
		ownerCounts[firstNonEmpty(signoff.EscalationOwner, "none")]++
		slaCounts[firstNonEmpty(signoff.SLAStatus, "on-track")]++
	}
	lines := []string{
		"# UI Review Sign-off Breach Board",
		"",
		fmt.Sprintf("- Breach items: %d", len(entries)),
		fmt.Sprintf("- Escalation owners: %d", len(ownerCounts)),
		"",
		"## SLA States",
	}
	for _, key := range sortStrings(mapKeys(slaCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, slaCounts[key]))
	}
	lines = append(lines, "", "## Escalation Owners")
	for _, key := range sortStrings(mapKeys(ownerCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, ownerCounts[key]))
	}
	lines = append(lines, "", "## Items")
	blockers := blockersBySignoff(pack)
	for _, signoff := range entries {
		var blockerIDs []string
		for _, blocker := range blockers[signoff.SignoffID] {
			blockerIDs = append(blockerIDs, blocker.BlockerID)
		}
		lines = append(lines,
			fmt.Sprintf("- breach-%s: signoff=%s role=%s surface=%s status=%s sla=%s escalation_owner=%s", signoff.SignoffID, signoff.SignoffID, signoff.Role, signoff.SurfaceID, signoff.Status, firstNonEmpty(signoff.SLAStatus, "on-track"), firstNonEmpty(signoff.EscalationOwner, "none")),
			fmt.Sprintf("  requested_at=%s due_at=%s linked_blockers=%s summary=%s", firstNonEmpty(signoff.RequestedAt, "none"), firstNonEmpty(signoff.DueAt, "none"), joinOrNone(sortStrings(blockerIDs)), signoff.Notes),
		)
	}
	return strings.Join(lines, "\n")
}

func RenderEscalationDashboard(pack UIReviewPack) string {
	type ownerBucket struct{ blockers, signoffs, total int }
	ownerCounts := map[string]ownerBucket{}
	statusCounts := map[string]ownerBucket{}
	type row struct {
		ID, Owner, ItemType, SourceID, SurfaceID, Status, Priority, CurrentOwner, Summary, DueAt string
	}
	var rows []row
	for _, blocker := range pack.BlockerLog {
		owner := firstNonEmpty(blocker.EscalationOwner, "none")
		ownerCounts[owner] = ownerBucket{blockers: ownerCounts[owner].blockers + 1, signoffs: ownerCounts[owner].signoffs, total: ownerCounts[owner].total + 1}
		statusCounts[blocker.Status] = ownerBucket{blockers: statusCounts[blocker.Status].blockers + 1, signoffs: statusCounts[blocker.Status].signoffs, total: statusCounts[blocker.Status].total + 1}
		rows = append(rows, row{"esc-" + blocker.BlockerID, owner, "blocker", blocker.BlockerID, blocker.SurfaceID, blocker.Status, blocker.Severity, blocker.Owner, blocker.Summary, "none"})
	}
	for _, signoff := range pack.SignoffLog {
		if !signoff.Required || !isOpenStatus(signoff.Status) {
			continue
		}
		owner := firstNonEmpty(signoff.EscalationOwner, "none")
		ownerCounts[owner] = ownerBucket{blockers: ownerCounts[owner].blockers, signoffs: ownerCounts[owner].signoffs + 1, total: ownerCounts[owner].total + 1}
		statusCounts[signoff.Status] = ownerBucket{blockers: statusCounts[signoff.Status].blockers, signoffs: statusCounts[signoff.Status].signoffs + 1, total: statusCounts[signoff.Status].total + 1}
		rows = append(rows, row{"esc-" + signoff.SignoffID, owner, "signoff", signoff.SignoffID, signoff.SurfaceID, signoff.Status, firstNonEmpty(signoff.SLAStatus, "on-track"), signoff.Role, signoff.Notes, firstNonEmpty(signoff.DueAt, "none")})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	lines := []string{
		"# UI Review Escalation Dashboard",
		"",
		fmt.Sprintf("- Items: %d", len(rows)),
		fmt.Sprintf("- Escalation owners: %d", len(ownerCounts)),
		"",
		"## By Escalation Owner",
	}
	for _, key := range sortStrings(mapKeys(ownerCounts)) {
		bucket := ownerCounts[key]
		lines = append(lines, fmt.Sprintf("- %s: blockers=%d signoffs=%d total=%d", key, bucket.blockers, bucket.signoffs, bucket.total))
	}
	lines = append(lines, "", "## By Status")
	for _, key := range sortStrings(mapKeys(statusCounts)) {
		bucket := statusCounts[key]
		lines = append(lines, fmt.Sprintf("- %s: blockers=%d signoffs=%d total=%d", key, bucket.blockers, bucket.signoffs, bucket.total))
	}
	lines = append(lines, "", "## Escalations")
	for _, row := range rows {
		lines = append(lines,
			fmt.Sprintf("- %s: owner=%s type=%s source=%s surface=%s status=%s priority=%s current_owner=%s", row.ID, row.Owner, row.ItemType, row.SourceID, row.SurfaceID, row.Status, row.Priority, row.CurrentOwner),
			fmt.Sprintf("  summary=%s due_at=%s", row.Summary, row.DueAt),
		)
	}
	return strings.Join(lines, "\n")
}

func RenderEscalationHandoffLedger(pack UIReviewPack) string {
	var events []ReviewBlockerEvent
	channelCounts := map[string]int{}
	statusCounts := map[string]int{}
	for _, event := range pack.BlockerTimeline {
		if strings.TrimSpace(event.HandoffTo) == "" {
			continue
		}
		events = append(events, event)
		channelCounts[firstNonEmpty(event.Channel, "none")]++
		statusCounts[event.Status]++
	}
	sort.Slice(events, func(i, j int) bool { return events[i].EventID < events[j].EventID })
	lines := []string{
		"# UI Review Escalation Handoff Ledger",
		"",
		fmt.Sprintf("- Handoffs: %d", len(events)),
		fmt.Sprintf("- Channels: %d", len(channelCounts)),
		"",
		"## By Status",
	}
	for _, key := range sortStrings(mapKeys(statusCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, statusCounts[key]))
	}
	lines = append(lines, "", "## By Channel")
	for _, key := range sortStrings(mapKeys(channelCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, channelCounts[key]))
	}
	lines = append(lines, "", "## Entries")
	for _, event := range events {
		blockerSurface := blockerSurface(pack, event.BlockerID)
		lines = append(lines,
			fmt.Sprintf("- handoff-%s: event=%s blocker=%s surface=%s actor=%s status=%s at=%s", event.EventID, event.EventID, event.BlockerID, blockerSurface, event.Actor, event.Status, event.Timestamp),
			fmt.Sprintf("  from=%s to=%s channel=%s artifact=%s next_action=%s", firstNonEmpty(event.HandoffFrom, "none"), firstNonEmpty(event.HandoffTo, "none"), firstNonEmpty(event.Channel, "none"), firstNonEmpty(event.ArtifactRef, "none"), firstNonEmpty(event.NextAction, "none")),
		)
	}
	return strings.Join(lines, "\n")
}

func RenderHandoffAckLedger(pack UIReviewPack) string {
	var events []ReviewBlockerEvent
	ownerCounts := map[string]int{}
	statusCounts := map[string]int{}
	for _, event := range pack.BlockerTimeline {
		if strings.TrimSpace(event.AckOwner) == "" {
			continue
		}
		events = append(events, event)
		ownerCounts[event.AckOwner]++
		statusCounts[firstNonEmpty(event.AckStatus, "pending")]++
	}
	sort.Slice(events, func(i, j int) bool { return events[i].EventID < events[j].EventID })
	lines := []string{
		"# UI Review Handoff Ack Ledger",
		"",
		fmt.Sprintf("- Ack items: %d", len(events)),
		fmt.Sprintf("- Ack owners: %d", len(ownerCounts)),
		"",
		"## By Ack Owner",
	}
	for _, key := range sortStrings(mapKeys(ownerCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, ownerCounts[key]))
	}
	lines = append(lines, "", "## By Ack Status")
	for _, key := range sortStrings(mapKeys(statusCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, statusCounts[key]))
	}
	lines = append(lines, "", "## Entries")
	for _, event := range events {
		lines = append(lines,
			fmt.Sprintf("- ack-%s: event=%s blocker=%s surface=%s handoff_to=%s ack_owner=%s ack_status=%s ack_at=%s", event.EventID, event.EventID, event.BlockerID, blockerSurface(pack, event.BlockerID), firstNonEmpty(event.HandoffTo, "none"), event.AckOwner, firstNonEmpty(event.AckStatus, "pending"), firstNonEmpty(event.AckAt, "none")),
			fmt.Sprintf("  actor=%s status=%s channel=%s artifact=%s summary=%s", event.Actor, event.Status, firstNonEmpty(event.Channel, "none"), firstNonEmpty(event.ArtifactRef, "none"), event.Summary),
		)
	}
	return strings.Join(lines, "\n")
}

func RenderOwnerEscalationDigest(pack UIReviewPack) string {
	type row struct{ ID, Owner, ItemType, SourceID, SurfaceID, Status, Summary, Detail string }
	type bucket struct{ blockers, signoffs, reminders, freezes, handoffs, total int }
	rows := []row{}
	owners := map[string]bucket{}
	for _, blocker := range pack.BlockerLog {
		owner := blocker.EscalationOwner
		if owner != "" {
			rows = append(rows, row{"digest-esc-" + blocker.BlockerID, owner, "blocker", blocker.BlockerID, blocker.SurfaceID, blocker.Status, blocker.Summary, blocker.Severity})
			b := owners[owner]
			b.blockers++
			b.total++
			owners[owner] = b
		}
		if blocker.FreezeOwner != "" {
			rows = append(rows, row{"digest-freeze-approval-" + blocker.BlockerID, blocker.FreezeOwner, "freeze", blocker.BlockerID, blocker.SurfaceID, blocker.Status, blocker.FreezeReason, blocker.FreezeUntil})
			b := owners[blocker.FreezeOwner]
			b.freezes++
			b.total++
			owners[blocker.FreezeOwner] = b
		}
	}
	for _, signoff := range pack.SignoffLog {
		if signoff.Required && isOpenStatus(signoff.Status) && signoff.EscalationOwner != "" {
			rows = append(rows, row{"digest-esc-" + signoff.SignoffID, signoff.EscalationOwner, "signoff", signoff.SignoffID, signoff.SurfaceID, signoff.Status, signoff.Notes, firstNonEmpty(signoff.SLAStatus, "on-track")})
			b := owners[signoff.EscalationOwner]
			b.signoffs++
			b.total++
			owners[signoff.EscalationOwner] = b
		}
		if signoff.Required && isOpenStatus(signoff.Status) && signoff.ReminderOwner != "" {
			rows = append(rows, row{"digest-rem-" + signoff.SignoffID, signoff.ReminderOwner, "reminder", signoff.SignoffID, signoff.SurfaceID, signoff.Status, signoff.Notes, firstNonEmpty(signoff.NextReminderAt, "none")})
			b := owners[signoff.ReminderOwner]
			b.reminders++
			b.total++
			owners[signoff.ReminderOwner] = b
		}
	}
	for _, event := range pack.BlockerTimeline {
		if event.HandoffTo != "" {
			rows = append(rows, row{"digest-handoff-" + event.EventID, event.HandoffTo, "handoff", event.BlockerID, blockerSurface(pack, event.BlockerID), event.Status, event.Summary, event.Timestamp})
			b := owners[event.HandoffTo]
			b.handoffs++
			b.total++
			owners[event.HandoffTo] = b
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	lines := []string{
		"# UI Review Owner Escalation Digest",
		"",
		fmt.Sprintf("- Owners: %d", len(owners)),
		fmt.Sprintf("- Items: %d", len(rows)),
		"",
		"## Owners",
	}
	for _, key := range sortStrings(mapKeys(owners)) {
		b := owners[key]
		lines = append(lines, fmt.Sprintf("- %s: blockers=%d signoffs=%d reminders=%d freezes=%d handoffs=%d total=%d", key, b.blockers, b.signoffs, b.reminders, b.freezes, b.handoffs, b.total))
	}
	lines = append(lines, "", "## Items")
	for _, row := range rows {
		lines = append(lines,
			fmt.Sprintf("- %s: owner=%s type=%s source=%s surface=%s status=%s", row.ID, row.Owner, row.ItemType, row.SourceID, row.SurfaceID, row.Status),
			fmt.Sprintf("  summary=%s detail=%s", row.Summary, row.Detail),
		)
	}
	return strings.Join(lines, "\n")
}

type ownerWorkloadEntry struct {
	ID, Owner, ItemType, SourceID, SurfaceID, Status, Lane, Detail, Summary string
}

func buildOwnerWorkloadEntries(pack UIReviewPack) []ownerWorkloadEntry {
	var rows []ownerWorkloadEntry
	for _, item := range buildOwnerReviewQueueEntries(pack) {
		rows = append(rows, ownerWorkloadEntry{"load-" + item.ID, item.Owner, item.ItemType, item.SourceID, item.SurfaceID, item.Status, "queue", item.NextAction, item.Summary})
	}
	for _, item := range buildReminderEntries(pack) {
		rows = append(rows, ownerWorkloadEntry{"load-" + item.ID, item.Owner, "reminder", item.SignoffID, item.SurfaceID, item.Status, "reminder", item.NextReminderAt, item.Summary})
	}
	for _, item := range buildFreezeRenewalEntries(pack) {
		rows = append(rows, ownerWorkloadEntry{"load-" + item.ID, item.Owner, "renewal", item.BlockerID, item.SurfaceID, item.Status, "renewal", item.RenewalBy, item.Summary})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	return rows
}

func RenderOwnerWorkloadBoard(pack UIReviewPack) string {
	rows := buildOwnerWorkloadEntries(pack)
	type bucket struct{ blockers, checklist, decisions, signoffs, reminders, renewals, total int }
	owners := map[string]bucket{}
	for _, row := range rows {
		b := owners[row.Owner]
		switch row.ItemType {
		case "blocker":
			b.blockers++
		case "checklist":
			b.checklist++
		case "decision":
			b.decisions++
		case "signoff":
			b.signoffs++
		case "reminder":
			b.reminders++
		case "renewal":
			b.renewals++
		}
		b.total++
		owners[row.Owner] = b
	}
	lines := []string{
		"# UI Review Owner Workload Board",
		"",
		fmt.Sprintf("- Owners: %d", len(owners)),
		fmt.Sprintf("- Items: %d", len(rows)),
		"",
		"## Owners",
	}
	for _, key := range sortStrings(mapKeys(owners)) {
		b := owners[key]
		lines = append(lines, fmt.Sprintf("- %s: blockers=%d checklist=%d decisions=%d signoffs=%d reminders=%d renewals=%d total=%d", key, b.blockers, b.checklist, b.decisions, b.signoffs, b.reminders, b.renewals, b.total))
	}
	lines = append(lines, "", "## Items")
	for _, row := range rows {
		lines = append(lines,
			fmt.Sprintf("- %s: owner=%s type=%s source=%s surface=%s status=%s lane=%s", row.ID, row.Owner, row.ItemType, row.SourceID, row.SurfaceID, row.Status, row.Lane),
			fmt.Sprintf("  detail=%s summary=%s", row.Detail, row.Summary),
		)
	}
	return strings.Join(lines, "\n")
}

func RenderBlockerLog(pack UIReviewPack) string {
	lines := []string{"# UI Review Blocker Log", ""}
	for _, blocker := range pack.BlockerLog {
		lines = append(lines,
			fmt.Sprintf("- %s: surface=%s signoff=%s owner=%s status=%s severity=%s", blocker.BlockerID, blocker.SurfaceID, blocker.SignoffID, blocker.Owner, blocker.Status, blocker.Severity),
			fmt.Sprintf("  summary=%s escalation_owner=%s next_action=%s freeze_owner=%s freeze_until=%s freeze_approved_by=%s freeze_approved_at=%s", blocker.Summary, firstNonEmpty(blocker.EscalationOwner, "none"), firstNonEmpty(blocker.NextAction, "none"), firstNonEmpty(blocker.FreezeOwner, "none"), firstNonEmpty(blocker.FreezeUntil, "none"), firstNonEmpty(blocker.FreezeApprovedBy, "none"), firstNonEmpty(blocker.FreezeApprovedAt, "none")),
		)
	}
	return strings.Join(lines, "\n")
}

func RenderBlockerTimeline(pack UIReviewPack) string {
	lines := []string{"# UI Review Blocker Timeline", ""}
	for _, event := range pack.BlockerTimeline {
		lines = append(lines,
			fmt.Sprintf("- %s: blocker=%s actor=%s status=%s at=%s", event.EventID, event.BlockerID, event.Actor, event.Status, event.Timestamp),
			fmt.Sprintf("  summary=%s next_action=%s", event.Summary, firstNonEmpty(event.NextAction, "none")),
		)
	}
	return strings.Join(lines, "\n")
}

func RenderFreezeExceptionBoard(pack UIReviewPack) string {
	type bucket struct{ blockers, signoffs, total int }
	ownerCounts := map[string]bucket{}
	surfaceCounts := map[string]bucket{}
	lines := []string{
		"# UI Review Freeze Exception Board",
		"",
		fmt.Sprintf("- Exceptions: %d", len(pack.BlockerLog)),
		fmt.Sprintf("- Owners: %d", len(pack.BlockerLog)),
		"",
		"## By Owner",
	}
	for _, blocker := range pack.BlockerLog {
		if !blocker.FreezeException {
			continue
		}
		bo := ownerCounts[blocker.FreezeOwner]
		bo.blockers++
		bo.total++
		ownerCounts[blocker.FreezeOwner] = bo
		bs := surfaceCounts[blocker.SurfaceID]
		bs.blockers++
		bs.total++
		surfaceCounts[blocker.SurfaceID] = bs
	}
	for _, key := range sortStrings(mapKeys(ownerCounts)) {
		b := ownerCounts[key]
		lines = append(lines, fmt.Sprintf("- %s: blockers=%d signoffs=%d total=%d", key, b.blockers, b.signoffs, b.total))
	}
	lines = append(lines, "", "## By Surface")
	for _, key := range sortStrings(mapKeys(surfaceCounts)) {
		b := surfaceCounts[key]
		lines = append(lines, fmt.Sprintf("- %s: blockers=%d signoffs=%d total=%d", key, b.blockers, b.signoffs, b.total))
	}
	lines = append(lines, "", "## Entries")
	for _, blocker := range pack.BlockerLog {
		if !blocker.FreezeException {
			continue
		}
		lines = append(lines,
			fmt.Sprintf("- freeze-%s: owner=%s type=blocker source=%s surface=%s status=%s window=%s", blocker.BlockerID, blocker.FreezeOwner, blocker.BlockerID, blocker.SurfaceID, blocker.Status, blocker.FreezeUntil),
			fmt.Sprintf("  summary=%s evidence=%s next_action=%s", blocker.FreezeReason, latestEventSummary(pack, blocker.BlockerID), blocker.NextAction),
		)
	}
	return strings.Join(lines, "\n")
}

func RenderFreezeApprovalTrail(pack UIReviewPack) string {
	lines := []string{
		"# UI Review Freeze Approval Trail",
		"",
		fmt.Sprintf("- Approvals: %d", len(pack.BlockerLog)),
		fmt.Sprintf("- Approvers: %d", len(pack.BlockerLog)),
		"",
		"## By Approver",
	}
	approverCounts := map[string]int{}
	statusCounts := map[string]int{}
	for _, blocker := range pack.BlockerLog {
		if !blocker.FreezeException {
			continue
		}
		approverCounts[blocker.FreezeApprovedBy]++
		statusCounts[blocker.Status]++
	}
	for _, key := range sortStrings(mapKeys(approverCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, approverCounts[key]))
	}
	lines = append(lines, "", "## By Status")
	for _, key := range sortStrings(mapKeys(statusCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, statusCounts[key]))
	}
	lines = append(lines, "", "## Entries")
	for _, blocker := range pack.BlockerLog {
		if !blocker.FreezeException {
			continue
		}
		lines = append(lines,
			fmt.Sprintf("- freeze-approval-%s: blocker=%s surface=%s status=%s owner=%s approved_by=%s approved_at=%s window=%s", blocker.BlockerID, blocker.BlockerID, blocker.SurfaceID, blocker.Status, blocker.FreezeOwner, blocker.FreezeApprovedBy, blocker.FreezeApprovedAt, blocker.FreezeUntil),
			fmt.Sprintf("  summary=%s latest_event=%s next_action=%s", blocker.FreezeReason, latestEventSummary(pack, blocker.BlockerID), blocker.NextAction),
		)
	}
	return strings.Join(lines, "\n")
}

type freezeRenewalEntry struct {
	ID, BlockerID, SurfaceID, Status, Owner, RenewalBy, RenewalStatus, Summary string
}

func buildFreezeRenewalEntries(pack UIReviewPack) []freezeRenewalEntry {
	var entries []freezeRenewalEntry
	for _, blocker := range pack.BlockerLog {
		if !blocker.FreezeException {
			continue
		}
		entries = append(entries, freezeRenewalEntry{
			ID:            "renew-" + blocker.BlockerID,
			BlockerID:     blocker.BlockerID,
			SurfaceID:     blocker.SurfaceID,
			Status:        blocker.Status,
			Owner:         blocker.FreezeRenewalOwner,
			RenewalBy:     blocker.FreezeRenewalBy,
			RenewalStatus: blocker.FreezeRenewalStatus,
			Summary:       blocker.FreezeReason,
		})
	}
	return entries
}

func RenderFreezeRenewalTracker(pack UIReviewPack) string {
	entries := buildFreezeRenewalEntries(pack)
	ownerCounts := map[string]int{}
	statusCounts := map[string]int{}
	for _, entry := range entries {
		ownerCounts[entry.Owner]++
		statusCounts[entry.RenewalStatus]++
	}
	lines := []string{
		"# UI Review Freeze Renewal Tracker",
		"",
		fmt.Sprintf("- Renewal items: %d", len(entries)),
		fmt.Sprintf("- Renewal owners: %d", len(ownerCounts)),
		"",
		"## By Renewal Owner",
	}
	for _, key := range sortStrings(mapKeys(ownerCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, ownerCounts[key]))
	}
	lines = append(lines, "", "## By Renewal Status")
	for _, key := range sortStrings(mapKeys(statusCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, statusCounts[key]))
	}
	lines = append(lines, "", "## Entries")
	for _, blocker := range pack.BlockerLog {
		if !blocker.FreezeException {
			continue
		}
		lines = append(lines,
			fmt.Sprintf("- renew-%s: blocker=%s surface=%s status=%s renewal_owner=%s renewal_by=%s renewal_status=%s", blocker.BlockerID, blocker.BlockerID, blocker.SurfaceID, blocker.Status, blocker.FreezeRenewalOwner, blocker.FreezeRenewalBy, blocker.FreezeRenewalStatus),
			fmt.Sprintf("  freeze_owner=%s freeze_until=%s approved_by=%s summary=%s next_action=%s", blocker.FreezeOwner, blocker.FreezeUntil, blocker.FreezeApprovedBy, blocker.FreezeReason, blocker.NextAction),
		)
	}
	return strings.Join(lines, "\n")
}

func RenderExceptionLog(pack UIReviewPack) string {
	lines := []string{
		"# UI Review Exception Log",
		"",
		fmt.Sprintf("- Exceptions: %d", len(buildExceptionRows(pack))),
		"",
	}
	for _, row := range buildExceptionRows(pack) {
		lines = append(lines,
			fmt.Sprintf("- %s: type=%s source=%s surface=%s owner=%s status=%s severity=%s", row.ID, row.Category, row.SourceID, row.SurfaceID, row.Owner, row.Status, row.Severity),
			fmt.Sprintf("  summary=%s evidence=%s latest_event=%s next_action=%s", row.Summary, row.Evidence, row.LatestEvent, row.NextAction),
		)
	}
	return strings.Join(lines, "\n")
}

type exceptionRow struct {
	ID, Category, SourceID, SurfaceID, Owner, Status, Severity, Summary, Evidence, LatestEvent, NextAction string
}

func buildExceptionRows(pack UIReviewPack) []exceptionRow {
	var rows []exceptionRow
	for _, blocker := range pack.BlockerLog {
		rows = append(rows, exceptionRow{
			ID:          "exc-" + blocker.BlockerID,
			Category:    "blocker",
			SourceID:    blocker.BlockerID,
			SurfaceID:   blocker.SurfaceID,
			Owner:       blocker.Owner,
			Status:      blocker.Status,
			Severity:    blocker.Severity,
			Summary:     blocker.Summary,
			Evidence:    firstNonEmpty(blocker.EscalationOwner, "none"),
			LatestEvent: latestEventSummary(pack, blocker.BlockerID),
			NextAction:  blocker.NextAction,
		})
	}
	for _, signoff := range pack.SignoffLog {
		if strings.EqualFold(signoff.Status, "waived") {
			rows = append(rows, exceptionRow{
				ID:        "exc-" + signoff.SignoffID,
				Category:  "signoff",
				SourceID:  signoff.SignoffID,
				SurfaceID: signoff.SurfaceID,
				Owner:     signoff.WaiverOwner,
				Status:    signoff.Status,
				Severity:  "waiver",
				Summary:   signoff.Notes,
				Evidence:  firstNonEmpty(signoff.WaiverReason, "none"),
				LatestEvent: "none",
				NextAction: firstNonEmpty(signoff.WaiverReason, "none"),
			})
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	return rows
}

func RenderExceptionMatrix(pack UIReviewPack) string {
	rows := buildExceptionRows(pack)
	type bucket struct{ blockers, signoffs, total int }
	ownerCounts := map[string]bucket{}
	statusCounts := map[string]bucket{}
	surfaceCounts := map[string]bucket{}
	for _, row := range rows {
		ob := ownerCounts[row.Owner]
		sb := statusCounts[row.Status]
		pb := surfaceCounts[row.SurfaceID]
		if row.Category == "blocker" {
			ob.blockers++
			sb.blockers++
			pb.blockers++
		} else {
			ob.signoffs++
			sb.signoffs++
			pb.signoffs++
		}
		ob.total++
		sb.total++
		pb.total++
		ownerCounts[row.Owner] = ob
		statusCounts[row.Status] = sb
		surfaceCounts[row.SurfaceID] = pb
	}
	lines := []string{
		"# UI Review Exception Matrix",
		"",
		fmt.Sprintf("- Exceptions: %d", len(rows)),
		fmt.Sprintf("- Owners: %d", len(ownerCounts)),
		fmt.Sprintf("- Surfaces: %d", len(surfaceCounts)),
		"",
		"## By Owner",
	}
	for _, key := range sortStrings(mapKeys(ownerCounts)) {
		b := ownerCounts[key]
		lines = append(lines, fmt.Sprintf("- %s: blockers=%d signoffs=%d total=%d", key, b.blockers, b.signoffs, b.total))
	}
	lines = append(lines, "", "## By Status")
	for _, key := range sortStrings(mapKeys(statusCounts)) {
		b := statusCounts[key]
		lines = append(lines, fmt.Sprintf("- %s: blockers=%d signoffs=%d total=%d", key, b.blockers, b.signoffs, b.total))
	}
	lines = append(lines, "", "## By Surface")
	for _, key := range sortStrings(mapKeys(surfaceCounts)) {
		b := surfaceCounts[key]
		lines = append(lines, fmt.Sprintf("- %s: blockers=%d signoffs=%d total=%d", key, b.blockers, b.signoffs, b.total))
	}
	return strings.Join(lines, "\n")
}

func RenderAuditDensityBoard(pack UIReviewPack) string {
	type entry struct {
		ID, SurfaceID, Band string
		ArtifactTotal, OpenTotal, ChecklistCount, DecisionCount, AssignmentCount, SignoffCount, BlockerCount, TimelineCount, BlockCount, NoteCount int
	}
	var entries []entry
	bandCounts := map[string]int{}
	for _, wireframe := range pack.Wireframes {
		checklistCount := countChecklistBySurface(pack, wireframe.SurfaceID)
		decisionCount := countDecisionsBySurface(pack, wireframe.SurfaceID)
		assignmentCount := countAssignmentsBySurface(pack, wireframe.SurfaceID)
		signoffCount := countSignoffsBySurface(pack, wireframe.SurfaceID)
		blockerCount := countBlockersBySurface(pack, wireframe.SurfaceID)
		timelineCount := countTimelineBySurface(pack, wireframe.SurfaceID)
		openTotal := 0
		for _, row := range buildWireframeReadinessEntries(pack) {
			if row.SurfaceID == wireframe.SurfaceID {
				openTotal = row.OpenTotal
			}
		}
		artifactTotal := checklistCount + decisionCount + assignmentCount + signoffCount + blockerCount + timelineCount
		band := "light"
		if artifactTotal >= 8 {
			band = "dense"
		} else if artifactTotal >= 6 {
			band = "active"
		}
		bandCounts[band]++
		entries = append(entries, entry{
			ID: "density-" + wireframe.SurfaceID, SurfaceID: wireframe.SurfaceID, Band: band,
			ArtifactTotal: artifactTotal, OpenTotal: openTotal, ChecklistCount: checklistCount, DecisionCount: decisionCount, AssignmentCount: assignmentCount, SignoffCount: signoffCount, BlockerCount: blockerCount, TimelineCount: timelineCount, BlockCount: len(wireframe.PrimaryBlocks), NoteCount: len(wireframe.ReviewNotes),
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].SurfaceID < entries[j].SurfaceID })
	lines := []string{
		"# UI Review Audit Density Board",
		"",
		fmt.Sprintf("- Surfaces: %d", len(entries)),
		fmt.Sprintf("- Load bands: %d", len(bandCounts)),
		"",
		"## By Load Band",
	}
	for _, key := range sortStrings(mapKeys(bandCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, bandCounts[key]))
	}
	lines = append(lines, "", "## Entries")
	for _, entry := range entries {
		lines = append(lines,
			fmt.Sprintf("- %s: surface=%s artifact_total=%d open_total=%d band=%s", entry.ID, entry.SurfaceID, entry.ArtifactTotal, entry.OpenTotal, entry.Band),
			fmt.Sprintf("  checklist=%d decisions=%d assignments=%d signoffs=%d blockers=%d timeline=%d blocks=%d notes=%d", entry.ChecklistCount, entry.DecisionCount, entry.AssignmentCount, entry.SignoffCount, entry.BlockerCount, entry.TimelineCount, entry.BlockCount, entry.NoteCount),
		)
	}
	return strings.Join(lines, "\n")
}

type ownerQueueEntry struct {
	ID, Owner, ItemType, SourceID, SurfaceID, Status, Summary, NextAction string
}

func buildOwnerReviewQueueEntries(pack UIReviewPack) []ownerQueueEntry {
	var rows []ownerQueueEntry
	for _, item := range pack.ReviewerChecklist {
		if isOpenStatus(item.Status) {
			rows = append(rows, ownerQueueEntry{"queue-" + item.ItemID, item.Owner, "checklist", item.ItemID, item.SurfaceID, item.Status, item.Prompt, joinOrNone(item.EvidenceLinks)})
		}
	}
	for _, decision := range pack.DecisionLog {
		if isOpenStatus(decision.Status) {
			rows = append(rows, ownerQueueEntry{"queue-" + decision.DecisionID, decision.Owner, "decision", decision.DecisionID, decision.SurfaceID, decision.Status, decision.Summary, firstNonEmpty(decision.FollowUp, "none")})
		}
	}
	for _, signoff := range pack.SignoffLog {
		if signoff.Required && isOpenStatus(signoff.Status) {
			rows = append(rows, ownerQueueEntry{"queue-" + signoff.SignoffID, signoff.Role, "signoff", signoff.SignoffID, signoff.SurfaceID, signoff.Status, signoff.Notes, signoff.Notes})
		}
	}
	for _, blocker := range pack.BlockerLog {
		if isOpenStatus(blocker.Status) {
			rows = append(rows, ownerQueueEntry{"queue-" + blocker.BlockerID, blocker.Owner, "blocker", blocker.BlockerID, blocker.SurfaceID, blocker.Status, blocker.Summary, blocker.NextAction})
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	return rows
}

func RenderOwnerReviewQueue(pack UIReviewPack) string {
	rows := buildOwnerReviewQueueEntries(pack)
	type bucket struct{ blockers, checklist, decisions, signoffs, total int }
	owners := map[string]bucket{}
	for _, row := range rows {
		b := owners[row.Owner]
		switch row.ItemType {
		case "blocker":
			b.blockers++
		case "checklist":
			b.checklist++
		case "decision":
			b.decisions++
		case "signoff":
			b.signoffs++
		}
		b.total++
		owners[row.Owner] = b
	}
	lines := []string{
		"# UI Review Owner Review Queue",
		"",
		fmt.Sprintf("- Owners: %d", len(owners)),
		fmt.Sprintf("- Queue items: %d", len(rows)),
		"",
		"## Owners",
	}
	for _, key := range sortStrings(mapKeys(owners)) {
		b := owners[key]
		lines = append(lines, fmt.Sprintf("- %s: blockers=%d checklist=%d decisions=%d signoffs=%d total=%d", key, b.blockers, b.checklist, b.decisions, b.signoffs, b.total))
	}
	lines = append(lines, "", "## Items")
	for _, row := range rows {
		lines = append(lines,
			fmt.Sprintf("- %s: owner=%s type=%s source=%s surface=%s status=%s", row.ID, row.Owner, row.ItemType, row.SourceID, row.SurfaceID, row.Status),
			fmt.Sprintf("  summary=%s next_action=%s", row.Summary, row.NextAction),
		)
	}
	return strings.Join(lines, "\n")
}

func RenderBlockerTimelineSummary(pack UIReviewPack) string {
	statusCounts := map[string]int{}
	actorCounts := map[string]int{}
	for _, event := range pack.BlockerTimeline {
		statusCounts[event.Status]++
		actorCounts[event.Actor]++
	}
	lines := []string{
		"# UI Review Blocker Timeline Summary",
		"",
		fmt.Sprintf("- Events: %d", len(pack.BlockerTimeline)),
		fmt.Sprintf("- Blockers with timeline: %d", len(latestBlockerEvents(pack))),
		"- Orphan timeline blockers: none",
		"",
		"## Events by Status",
	}
	for _, key := range sortStrings(mapKeys(statusCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, statusCounts[key]))
	}
	lines = append(lines, "", "## Events by Actor")
	for _, key := range sortStrings(mapKeys(actorCounts)) {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, actorCounts[key]))
	}
	lines = append(lines, "", "## Latest Blocker Events")
	for _, blocker := range pack.BlockerLog {
		event := latestBlockerEvents(pack)[blocker.BlockerID]
		lines = append(lines, fmt.Sprintf("- %s: latest=%s actor=%s status=%s at=%s", blocker.BlockerID, event.EventID, event.Actor, event.Status, event.Timestamp))
	}
	return strings.Join(lines, "\n")
}

func RenderPackReport(pack UIReviewPack, audit UIReviewPackAudit) string {
	pack = pack.ensureDefaults()
	lines := []string{
		"# UI Review Pack",
		"",
		fmt.Sprintf("- Issue: %s %s", pack.IssueID, pack.Title),
		fmt.Sprintf("- Version: %s", pack.Version),
		fmt.Sprintf("- Audit: %s", audit.Summary()),
		"",
		"## Objectives",
	}
	for _, objective := range pack.Objectives {
		lines = append(lines,
			fmt.Sprintf("- %s: %s persona=%s priority=%s", objective.ObjectiveID, objective.Title, objective.Persona, objective.Priority),
			fmt.Sprintf("  outcome=%s success_signal=%s dependencies=%s", objective.Outcome, objective.SuccessSignal, joinOrNone(objective.Dependencies)),
		)
	}
	lines = append(lines,
		"",
		trimHeading(RenderReviewSummaryBoard(pack)),
		"",
		trimHeading(RenderObjectiveCoverageBoard(pack)),
		"",
		trimHeading(RenderPersonaReadinessBoard(pack)),
		"",
		"## Wireframes",
	)
	for _, wireframe := range pack.Wireframes {
		lines = append(lines,
			fmt.Sprintf("- %s: %s device=%s entry=%s", wireframe.SurfaceID, wireframe.Name, wireframe.Device, wireframe.EntryPoint),
			fmt.Sprintf("  blocks=%s review_notes=%s", joinOrNone(wireframe.PrimaryBlocks), joinOrNone(wireframe.ReviewNotes)),
		)
	}
	lines = append(lines,
		"",
		trimHeading(RenderWireframeReadinessBoard(pack)),
		"",
		"## Interactions",
	)
	for _, flow := range pack.Interactions {
		lines = append(lines,
			fmt.Sprintf("- %s: %s trigger=%s", flow.FlowID, flow.Name, flow.Trigger),
			fmt.Sprintf("  response=%s states=%s exceptions=%s", flow.SystemResponse, joinOrNone(flow.States), joinOrNone(flow.Exceptions)),
		)
	}
	lines = append(lines,
		"",
		trimHeading(RenderInteractionCoverageBoard(pack)),
		"",
		"## Open Questions",
	)
	for _, question := range pack.OpenQuestions {
		lines = append(lines,
			fmt.Sprintf("- %s: %s owner=%s status=%s", question.QuestionID, question.Theme, question.Owner, question.Status),
			fmt.Sprintf("  question=%s impact=%s", question.Question, question.Impact),
		)
	}
	sections := []string{
		RenderOpenQuestionTracker(pack),
		renderChecklistList(pack),
		RenderDecisionLog(pack),
		RenderRoleMatrix(pack),
		RenderChecklistTraceabilityBoard(pack),
		RenderDecisionFollowupTracker(pack),
		RenderRoleCoverageBoard(pack),
		RenderSignoffDependencyBoard(pack),
		RenderSignoffLog(pack),
		RenderSignoffSLADashboard(pack),
		RenderSignoffReminderQueue(pack),
		RenderReminderCadenceBoard(pack),
		RenderSignoffBreachBoard(pack),
		RenderEscalationDashboard(pack),
		RenderEscalationHandoffLedger(pack),
		RenderHandoffAckLedger(pack),
		RenderOwnerEscalationDigest(pack),
		RenderOwnerWorkloadBoard(pack),
		RenderBlockerLog(pack),
		RenderBlockerTimeline(pack),
		RenderExceptionLog(pack),
		RenderFreezeExceptionBoard(pack),
		RenderFreezeApprovalTrail(pack),
		RenderFreezeRenewalTracker(pack),
		RenderExceptionMatrix(pack),
		RenderAuditDensityBoard(pack),
		RenderOwnerReviewQueue(pack),
		RenderBlockerTimelineSummary(pack),
		renderAuditFindings(audit),
	}
	for _, section := range sections {
		lines = append(lines, "", trimHeading(section))
	}
	return strings.Join(lines, "\n")
}

func trimHeading(section string) string {
	if strings.HasPrefix(section, "# ") {
		return "## " + strings.TrimPrefix(section, "# ")
	}
	return section
}

func renderChecklistList(pack UIReviewPack) string {
	lines := []string{"# Reviewer Checklist", ""}
	for _, item := range pack.ReviewerChecklist {
		lines = append(lines,
			fmt.Sprintf("- %s: surface=%s owner=%s status=%s", item.ItemID, item.SurfaceID, item.Owner, item.Status),
			fmt.Sprintf("  prompt=%s evidence=%s notes=%s", item.Prompt, joinOrNone(item.EvidenceLinks), firstNonEmpty(item.Notes, "none")),
		)
	}
	return strings.Join(lines, "\n")
}

func renderAuditFindings(audit UIReviewPackAudit) string {
	lines := []string{
		"# Audit Findings",
		"",
		fmt.Sprintf("- Missing sections: %s", joinOrNone(audit.MissingSections)),
		fmt.Sprintf("- Objectives missing success signals: %s", joinOrNone(audit.ObjectivesMissingSignals)),
		fmt.Sprintf("- Wireframes missing blocks: %s", joinOrNone(audit.WireframesMissingBlocks)),
		fmt.Sprintf("- Interactions missing states: %s", joinOrNone(audit.InteractionsMissingStates)),
		fmt.Sprintf("- Unresolved questions: %s", joinOrNone(audit.UnresolvedQuestionIDs)),
		fmt.Sprintf("- Wireframes missing checklist coverage: %s", joinOrNone(audit.WireframesMissingChecklists)),
		fmt.Sprintf("- Orphan checklist surfaces: %s", joinOrNone(audit.OrphanChecklistSurfaces)),
		fmt.Sprintf("- Checklist items missing evidence: %s", joinOrNone(audit.ChecklistItemsMissingEvidence)),
		fmt.Sprintf("- Checklist items missing role links: %s", joinOrNone(audit.ChecklistItemsMissingRoleLinks)),
		fmt.Sprintf("- Wireframes missing decision coverage: %s", joinOrNone(audit.WireframesMissingDecisions)),
		fmt.Sprintf("- Orphan decision surfaces: %s", joinOrNone(audit.OrphanDecisionSurfaces)),
		fmt.Sprintf("- Unresolved decision ids: %s", joinOrNone(audit.UnresolvedDecisionIDs)),
		fmt.Sprintf("- Unresolved decisions missing follow-ups: %s", joinOrNone(audit.UnresolvedDecisionsMissingFollowUps)),
		fmt.Sprintf("- Wireframes missing role assignments: %s", joinOrNone(audit.WireframesMissingRoleAssignments)),
		fmt.Sprintf("- Wireframes missing signoff coverage: %s", joinOrNone(audit.WireframesMissingSignoffs)),
		fmt.Sprintf("- Decisions missing role links: %s", joinOrNone(audit.DecisionsMissingRoleLinks)),
		fmt.Sprintf("- Signoffs missing requested dates: %s", joinOrNone(audit.SignoffsMissingRequestedDates)),
		fmt.Sprintf("- Signoffs missing due dates: %s", joinOrNone(audit.SignoffsMissingDueDates)),
		fmt.Sprintf("- Signoffs missing escalation owners: %s", joinOrNone(audit.SignoffsMissingEscalationOwners)),
		fmt.Sprintf("- Signoffs missing reminder owners: %s", joinOrNone(audit.SignoffsMissingReminderOwners)),
		fmt.Sprintf("- Signoffs missing next reminders: %s", joinOrNone(audit.SignoffsMissingNextReminders)),
		fmt.Sprintf("- Signoffs missing reminder cadence: %s", joinOrNone(audit.SignoffsMissingReminderCadence)),
		fmt.Sprintf("- Signoffs with breached SLA: %s", joinOrNone(audit.SignoffsWithBreachedSLA)),
		fmt.Sprintf("- Unresolved required signoff ids: %s", joinOrNone(audit.UnresolvedRequiredSignoffIDs)),
		fmt.Sprintf("- Blockers missing signoff links: %s", joinOrNone(audit.BlockersMissingSignoffLinks)),
		fmt.Sprintf("- Freeze exceptions missing owners: %s", joinOrNone(audit.FreezeExceptionsMissingOwners)),
		fmt.Sprintf("- Freeze exceptions missing windows: %s", joinOrNone(audit.FreezeExceptionsMissingUntil)),
		fmt.Sprintf("- Freeze exceptions missing approvers: %s", joinOrNone(audit.FreezeExceptionsMissingApprovers)),
		fmt.Sprintf("- Freeze exceptions missing approval dates: %s", joinOrNone(audit.FreezeExceptionsMissingApprovalDates)),
		fmt.Sprintf("- Freeze exceptions missing renewal owners: %s", joinOrNone(audit.FreezeExceptionsMissingRenewalOwners)),
		fmt.Sprintf("- Freeze exceptions missing renewal dates: %s", joinOrNone(audit.FreezeExceptionsMissingRenewalDates)),
		fmt.Sprintf("- Blockers missing timeline events: %s", joinOrNone(audit.BlockersMissingTimelineEvents)),
		fmt.Sprintf("- Closed blockers missing resolution events: %s", joinOrNone(audit.ClosedBlockersMissingResolutionEvents)),
		fmt.Sprintf("- Orphan blocker timeline blocker ids: %s", joinOrNone(audit.OrphanBlockerTimelineBlockerIDs)),
		fmt.Sprintf("- Handoff events missing targets: %s", joinOrNone(audit.HandoffEventsMissingTargets)),
		fmt.Sprintf("- Handoff events missing artifacts: %s", joinOrNone(audit.HandoffEventsMissingArtifacts)),
		fmt.Sprintf("- Handoff events missing ack owners: %s", joinOrNone(audit.HandoffEventsMissingAckOwners)),
		fmt.Sprintf("- Handoff events missing ack dates: %s", joinOrNone(audit.HandoffEventsMissingAckDates)),
		fmt.Sprintf("- Unresolved required signoffs without blockers: %s", joinOrNone(audit.UnresolvedRequiredSignoffsWithoutBlockers)),
	}
	return strings.Join(lines, "\n")
}

func RenderPackHTML(pack UIReviewPack, audit UIReviewPackAudit) string {
	sections := []string{
		RenderPackReport(pack, audit),
		RenderDecisionLog(pack),
		RenderChecklistTraceabilityBoard(pack),
		RenderDecisionFollowupTracker(pack),
		RenderReviewSummaryBoard(pack),
		RenderObjectiveCoverageBoard(pack),
		RenderPersonaReadinessBoard(pack),
		RenderWireframeReadinessBoard(pack),
		RenderInteractionCoverageBoard(pack),
		RenderOpenQuestionTracker(pack),
		RenderRoleMatrix(pack),
		RenderRoleCoverageBoard(pack),
		RenderSignoffDependencyBoard(pack),
		RenderSignoffLog(pack),
		RenderSignoffSLADashboard(pack),
		RenderSignoffReminderQueue(pack),
		RenderReminderCadenceBoard(pack),
		RenderSignoffBreachBoard(pack),
		RenderEscalationDashboard(pack),
		RenderEscalationHandoffLedger(pack),
		RenderHandoffAckLedger(pack),
		RenderOwnerEscalationDigest(pack),
		RenderOwnerWorkloadBoard(pack),
		RenderBlockerLog(pack),
		RenderBlockerTimeline(pack),
		RenderFreezeExceptionBoard(pack),
		RenderFreezeApprovalTrail(pack),
		RenderFreezeRenewalTracker(pack),
		RenderExceptionLog(pack),
		RenderExceptionMatrix(pack),
		RenderAuditDensityBoard(pack),
		RenderOwnerReviewQueue(pack),
		RenderBlockerTimelineSummary(pack),
	}
	var builder strings.Builder
	builder.WriteString("<html><body>")
	for _, section := range sections {
		builder.WriteString(renderMarkdownSectionAsHTML(section))
	}
	builder.WriteString("</body></html>")
	return builder.String()
}

func renderMarkdownSectionAsHTML(section string) string {
	lines := strings.Split(section, "\n")
	var builder strings.Builder
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "# "):
			title := strings.TrimPrefix(line, "# ")
			if title == "UI Review Pack" {
				builder.WriteString("<h1>" + html.EscapeString(title) + "</h1>")
			} else {
				builder.WriteString("<h2>" + html.EscapeString(title) + "</h2>")
			}
		case strings.HasPrefix(line, "## "):
			builder.WriteString("<h2>" + html.EscapeString(strings.TrimPrefix(line, "## ")) + "</h2>")
		case strings.TrimSpace(line) == "":
			continue
		default:
			builder.WriteString("<p>" + html.EscapeString(line) + "</p>")
		}
	}
	return builder.String()
}

type Artifacts struct {
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

func WriteBundle(root string, pack UIReviewPack) (Artifacts, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return Artifacts{}, fmt.Errorf("mkdir bundle root: %w", err)
	}
	slug := strings.ToLower(strings.ReplaceAll(pack.IssueID, " ", "-"))
	audit := Auditor{}.Audit(pack)
	artifacts := Artifacts{
		RootDir:                        root,
		MarkdownPath:                   filepath.Join(root, slug+"-review-pack.md"),
		HTMLPath:                       filepath.Join(root, slug+"-review-pack.html"),
		DecisionLogPath:                filepath.Join(root, slug+"-decision-log.md"),
		ReviewSummaryBoardPath:         filepath.Join(root, slug+"-review-summary-board.md"),
		ObjectiveCoverageBoardPath:     filepath.Join(root, slug+"-objective-coverage-board.md"),
		PersonaReadinessBoardPath:      filepath.Join(root, slug+"-persona-readiness-board.md"),
		WireframeReadinessBoardPath:    filepath.Join(root, slug+"-wireframe-readiness-board.md"),
		InteractionCoverageBoardPath:   filepath.Join(root, slug+"-interaction-coverage-board.md"),
		OpenQuestionTrackerPath:        filepath.Join(root, slug+"-open-question-tracker.md"),
		ChecklistTraceabilityBoardPath: filepath.Join(root, slug+"-checklist-traceability-board.md"),
		DecisionFollowupTrackerPath:    filepath.Join(root, slug+"-decision-followup-tracker.md"),
		RoleMatrixPath:                 filepath.Join(root, slug+"-role-matrix.md"),
		RoleCoverageBoardPath:          filepath.Join(root, slug+"-role-coverage-board.md"),
		SignoffDependencyBoardPath:     filepath.Join(root, slug+"-signoff-dependency-board.md"),
		SignoffLogPath:                 filepath.Join(root, slug+"-signoff-log.md"),
		SignoffSLADashboardPath:        filepath.Join(root, slug+"-signoff-sla-dashboard.md"),
		SignoffReminderQueuePath:       filepath.Join(root, slug+"-signoff-reminder-queue.md"),
		ReminderCadenceBoardPath:       filepath.Join(root, slug+"-reminder-cadence-board.md"),
		SignoffBreachBoardPath:         filepath.Join(root, slug+"-signoff-breach-board.md"),
		EscalationDashboardPath:        filepath.Join(root, slug+"-escalation-dashboard.md"),
		EscalationHandoffLedgerPath:    filepath.Join(root, slug+"-escalation-handoff-ledger.md"),
		HandoffAckLedgerPath:           filepath.Join(root, slug+"-handoff-ack-ledger.md"),
		OwnerEscalationDigestPath:      filepath.Join(root, slug+"-owner-escalation-digest.md"),
		OwnerWorkloadBoardPath:         filepath.Join(root, slug+"-owner-workload-board.md"),
		BlockerLogPath:                 filepath.Join(root, slug+"-blocker-log.md"),
		BlockerTimelinePath:            filepath.Join(root, slug+"-blocker-timeline.md"),
		FreezeExceptionBoardPath:       filepath.Join(root, slug+"-freeze-exception-board.md"),
		FreezeApprovalTrailPath:        filepath.Join(root, slug+"-freeze-approval-trail.md"),
		FreezeRenewalTrackerPath:       filepath.Join(root, slug+"-freeze-renewal-tracker.md"),
		ExceptionLogPath:               filepath.Join(root, slug+"-exception-log.md"),
		ExceptionMatrixPath:            filepath.Join(root, slug+"-exception-matrix.md"),
		AuditDensityBoardPath:          filepath.Join(root, slug+"-audit-density-board.md"),
		OwnerReviewQueuePath:           filepath.Join(root, slug+"-owner-review-queue.md"),
		BlockerTimelineSummaryPath:     filepath.Join(root, slug+"-blocker-timeline-summary.md"),
	}
	writes := map[string]string{
		artifacts.MarkdownPath:                   RenderPackReport(pack, audit),
		artifacts.HTMLPath:                       RenderPackHTML(pack, audit),
		artifacts.DecisionLogPath:                RenderDecisionLog(pack),
		artifacts.ReviewSummaryBoardPath:         RenderReviewSummaryBoard(pack),
		artifacts.ObjectiveCoverageBoardPath:     RenderObjectiveCoverageBoard(pack),
		artifacts.PersonaReadinessBoardPath:      RenderPersonaReadinessBoard(pack),
		artifacts.WireframeReadinessBoardPath:    RenderWireframeReadinessBoard(pack),
		artifacts.InteractionCoverageBoardPath:   RenderInteractionCoverageBoard(pack),
		artifacts.OpenQuestionTrackerPath:        RenderOpenQuestionTracker(pack),
		artifacts.ChecklistTraceabilityBoardPath: RenderChecklistTraceabilityBoard(pack),
		artifacts.DecisionFollowupTrackerPath:    RenderDecisionFollowupTracker(pack),
		artifacts.RoleMatrixPath:                 RenderRoleMatrix(pack),
		artifacts.RoleCoverageBoardPath:          RenderRoleCoverageBoard(pack),
		artifacts.SignoffDependencyBoardPath:     RenderSignoffDependencyBoard(pack),
		artifacts.SignoffLogPath:                 RenderSignoffLog(pack),
		artifacts.SignoffSLADashboardPath:        RenderSignoffSLADashboard(pack),
		artifacts.SignoffReminderQueuePath:       RenderSignoffReminderQueue(pack),
		artifacts.ReminderCadenceBoardPath:       RenderReminderCadenceBoard(pack),
		artifacts.SignoffBreachBoardPath:         RenderSignoffBreachBoard(pack),
		artifacts.EscalationDashboardPath:        RenderEscalationDashboard(pack),
		artifacts.EscalationHandoffLedgerPath:    RenderEscalationHandoffLedger(pack),
		artifacts.HandoffAckLedgerPath:           RenderHandoffAckLedger(pack),
		artifacts.OwnerEscalationDigestPath:      RenderOwnerEscalationDigest(pack),
		artifacts.OwnerWorkloadBoardPath:         RenderOwnerWorkloadBoard(pack),
		artifacts.BlockerLogPath:                 RenderBlockerLog(pack),
		artifacts.BlockerTimelinePath:            RenderBlockerTimeline(pack),
		artifacts.FreezeExceptionBoardPath:       RenderFreezeExceptionBoard(pack),
		artifacts.FreezeApprovalTrailPath:        RenderFreezeApprovalTrail(pack),
		artifacts.FreezeRenewalTrackerPath:       RenderFreezeRenewalTracker(pack),
		artifacts.ExceptionLogPath:               RenderExceptionLog(pack),
		artifacts.ExceptionMatrixPath:            RenderExceptionMatrix(pack),
		artifacts.AuditDensityBoardPath:          RenderAuditDensityBoard(pack),
		artifacts.OwnerReviewQueuePath:           RenderOwnerReviewQueue(pack),
		artifacts.BlockerTimelineSummaryPath:     RenderBlockerTimelineSummary(pack),
	}
	for path, body := range writes {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return Artifacts{}, fmt.Errorf("write %s: %w", filepath.Base(path), err)
		}
	}
	return artifacts, nil
}

func slugify(value string) string {
	return strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(value, " ", "-"), "_", "-"))
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func latestEventSummary(pack UIReviewPack, blockerID string) string {
	event, ok := latestBlockerEvents(pack)[blockerID]
	if !ok {
		return "none"
	}
	return fmt.Sprintf("%s/%s/%s@%s", event.EventID, event.Status, event.Actor, event.Timestamp)
}

func blockerSurface(pack UIReviewPack, blockerID string) string {
	for _, blocker := range pack.BlockerLog {
		if blocker.BlockerID == blockerID {
			return blocker.SurfaceID
		}
	}
	return "none"
}

func mapKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

func setKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

func uniqueStrings(values []string) []string {
	set := map[string]struct{}{}
	for _, value := range values {
		if value != "" {
			set[value] = struct{}{}
		}
	}
	return setKeys(set)
}

func firstSignoffIDForRole(signoffs []ReviewSignoff) string {
	if len(signoffs) == 0 {
		return ""
	}
	return signoffs[0].SignoffID
}

func questionOwners(pack UIReviewPack) []string {
	values := make([]string, 0, len(pack.OpenQuestions))
	for _, question := range pack.OpenQuestions {
		values = append(values, question.Owner)
	}
	return values
}

func distinctCount(values []string) int {
	return len(uniqueStrings(values))
}

func countChecklistBySurface(pack UIReviewPack, surfaceID string) int {
	count := 0
	for _, item := range pack.ReviewerChecklist {
		if item.SurfaceID == surfaceID {
			count++
		}
	}
	return count
}

func countDecisionsBySurface(pack UIReviewPack, surfaceID string) int {
	count := 0
	for _, item := range pack.DecisionLog {
		if item.SurfaceID == surfaceID {
			count++
		}
	}
	return count
}

func countAssignmentsBySurface(pack UIReviewPack, surfaceID string) int {
	count := 0
	for _, item := range pack.RoleMatrix {
		if item.SurfaceID == surfaceID {
			count++
		}
	}
	return count
}

func countSignoffsBySurface(pack UIReviewPack, surfaceID string) int {
	count := 0
	for _, item := range pack.SignoffLog {
		if item.SurfaceID == surfaceID {
			count++
		}
	}
	return count
}

func countBlockersBySurface(pack UIReviewPack, surfaceID string) int {
	count := 0
	for _, item := range pack.BlockerLog {
		if item.SurfaceID == surfaceID {
			count++
		}
	}
	return count
}

func countTimelineBySurface(pack UIReviewPack, surfaceID string) int {
	count := 0
	for _, event := range pack.BlockerTimeline {
		if blockerSurface(pack, event.BlockerID) == surfaceID {
			count++
		}
	}
	return count
}
