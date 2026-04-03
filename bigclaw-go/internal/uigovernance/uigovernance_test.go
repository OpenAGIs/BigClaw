package uigovernance

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestComponentReleaseReadyRequiresDocsAccessibilityAndStates(t *testing.T) {
	component := ComponentSpec{
		Name:                      "Button",
		Readiness:                 "stable",
		DocumentationComplete:     true,
		AccessibilityRequirements: []string{"focus-visible", "keyboard-activation"},
		Variants: []ComponentVariant{{
			Name:   "primary",
			Tokens: []string{"color.action.primary", "spacing.control.md"},
			States: []string{"default", "hover", "disabled"},
		}},
	}

	if !component.ReleaseReady() {
		t.Fatalf("expected component to be release ready")
	}
	if !reflect.DeepEqual(component.TokenNames(), []string{"color.action.primary", "spacing.control.md"}) {
		t.Fatalf("unexpected token names: %+v", component.TokenNames())
	}
	if len(component.MissingRequiredStates()) != 0 {
		t.Fatalf("expected no missing states, got %+v", component.MissingRequiredStates())
	}
}

func TestDesignSystemRoundTripPreservesManifestShape(t *testing.T) {
	system := DesignSystem{
		Name:    "BigClaw Console UI",
		Version: "v2",
		Tokens: []DesignToken{{
			Name:         "color.action.primary",
			Category:     "color",
			Value:        "#4455ff",
			SemanticRole: "action-primary",
		}},
		Components: []ComponentSpec{{
			Name:                      "Button",
			Readiness:                 "stable",
			Slots:                     []string{"icon", "label"},
			DocumentationComplete:     true,
			AccessibilityRequirements: []string{"focus-visible"},
			Variants: []ComponentVariant{{
				Name:       "primary",
				Tokens:     []string{"color.action.primary"},
				States:     []string{"default", "hover", "disabled"},
				UsageNotes: "Use for primary CTA.",
			}},
		}},
	}

	var restored DesignSystem
	roundTripJSON(t, system, &restored)
	if !reflect.DeepEqual(restored.normalized(), system.normalized()) {
		t.Fatalf("round trip mismatch: got %+v want %+v", restored, system)
	}
}

func TestDesignSystemAuditSurfacesReleaseGapsAndOrphanTokens(t *testing.T) {
	system := DesignSystem{
		Name:    "BigClaw Console UI",
		Version: "v2",
		Tokens: []DesignToken{
			{Name: "color.action.primary", Category: "color", Value: "#4455ff"},
			{Name: "spacing.control.md", Category: "spacing", Value: "12px"},
			{Name: "radius.md", Category: "radius", Value: "8px"},
		},
		Components: []ComponentSpec{
			{
				Name:                      "Button",
				Readiness:                 "stable",
				DocumentationComplete:     true,
				AccessibilityRequirements: []string{"focus-visible", "keyboard-activation"},
				Variants:                  []ComponentVariant{{Name: "primary", Tokens: []string{"color.action.primary", "spacing.control.md"}, States: []string{"default", "hover", "disabled"}}},
			},
			{
				Name:      "CommandBar",
				Readiness: "beta",
				Variants:  []ComponentVariant{{Name: "global", Tokens: []string{"spacing.control.md"}, States: []string{"default", "hover"}}},
			},
		},
	}

	audit := (ComponentLibrary{}).Audit(system)
	if !reflect.DeepEqual(audit.ReleaseReadyComponents, []string{"Button"}) ||
		!reflect.DeepEqual(audit.ComponentsMissingDocs, []string{"CommandBar"}) ||
		!reflect.DeepEqual(audit.ComponentsMissingAccessibility, []string{"CommandBar"}) ||
		!reflect.DeepEqual(audit.ComponentsMissingStates, []string{"CommandBar"}) ||
		len(audit.UndefinedTokenRefs) != 0 ||
		!reflect.DeepEqual(audit.TokenOrphans, []string{"radius.md"}) ||
		audit.ReadinessScore() != 35.0 {
		t.Fatalf("unexpected audit: %+v", audit)
	}
}

func TestDesignSystemAuditFlagsUndefinedTokenReferences(t *testing.T) {
	system := DesignSystem{
		Name:    "BigClaw Console UI",
		Version: "v2",
		Tokens:  []DesignToken{{Name: "spacing.control.md", Category: "spacing", Value: "12px"}},
		Components: []ComponentSpec{{
			Name:                      "SideNav",
			Readiness:                 "stable",
			DocumentationComplete:     true,
			AccessibilityRequirements: []string{"focus-visible"},
			Variants:                  []ComponentVariant{{Name: "default", Tokens: []string{"spacing.control.md", "color.surface.nav"}, States: []string{"default", "hover", "disabled"}}},
		}},
	}

	audit := (ComponentLibrary{}).Audit(system)
	if len(audit.ReleaseReadyComponents) != 0 {
		t.Fatalf("expected no release ready components, got %+v", audit.ReleaseReadyComponents)
	}
	if !reflect.DeepEqual(audit.UndefinedTokenRefs, map[string][]string{"SideNav": []string{"color.surface.nav"}}) {
		t.Fatalf("unexpected undefined refs: %+v", audit.UndefinedTokenRefs)
	}
}

func TestDesignSystemAuditRoundTripPreservesGovernanceFindings(t *testing.T) {
	audit := DesignSystemAudit{
		SystemName:                     "BigClaw Console UI",
		Version:                        "v2",
		TokenCounts:                    map[string]int{"color": 3, "spacing": 2},
		ComponentCount:                 2,
		ReleaseReadyComponents:         []string{"Button"},
		ComponentsMissingDocs:          []string{"CommandBar"},
		ComponentsMissingAccessibility: []string{"CommandBar"},
		ComponentsMissingStates:        []string{"CommandBar"},
		UndefinedTokenRefs:             map[string][]string{"SideNav": []string{"color.surface.nav"}},
		TokenOrphans:                   []string{"radius.md"},
	}

	var restored DesignSystemAudit
	roundTripJSON(t, audit, &restored)
	if !reflect.DeepEqual(restored, audit) {
		t.Fatalf("audit round trip mismatch: got %+v want %+v", restored, audit)
	}
}

func TestRenderDesignSystemReportSummarizesInventoryAndGaps(t *testing.T) {
	system := DesignSystem{
		Name:    "BigClaw Console UI",
		Version: "v2",
		Tokens: []DesignToken{
			{Name: "color.action.primary", Category: "color", Value: "#4455ff"},
			{Name: "spacing.control.md", Category: "spacing", Value: "12px"},
		},
		Components: []ComponentSpec{{
			Name:                      "Button",
			Readiness:                 "stable",
			DocumentationComplete:     true,
			AccessibilityRequirements: []string{"focus-visible"},
			Variants:                  []ComponentVariant{{Name: "primary", Tokens: []string{"color.action.primary", "spacing.control.md"}, States: []string{"default", "hover", "disabled"}}},
		}},
	}

	report := RenderDesignSystemReport(system, (ComponentLibrary{}).Audit(system))
	for _, fragment := range []string{
		"# Design System Report",
		"- Release Ready Components: 1",
		"- color: 1",
		"- Button: readiness=stable docs=True a11y=True states=default, hover, disabled missing_states=none undefined_tokens=none",
		"- Missing interaction states: none",
		"- Undefined token refs: none",
		"- Orphan tokens: none",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}

func TestConsoleTopBarRoundTripPreservesCommandEntryManifest(t *testing.T) {
	topBar := ConsoleTopBar{
		Name:                      "BigClaw Global Header",
		SearchPlaceholder:         "Search runs, issues, commands",
		EnvironmentOptions:        []string{"Production", "Staging"},
		TimeRangeOptions:          []string{"24h", "7d", "30d"},
		AlertChannels:             []string{"approvals", "sla"},
		DocumentationComplete:     true,
		AccessibilityRequirements: []string{"keyboard-navigation", "screen-reader-label", "focus-visible"},
		CommandEntry: ConsoleCommandEntry{
			TriggerLabel:         "Command Menu",
			Placeholder:          "Type a command or jump to a run",
			Shortcut:             "Cmd+K / Ctrl+K",
			RecentQueriesEnabled: true,
			Commands: []CommandAction{
				{ID: "search-runs", Title: "Search runs", Section: "Navigate", Shortcut: "/"},
				{ID: "open-alerts", Title: "Open alerts", Section: "Monitor"},
			},
		},
	}

	var restored ConsoleTopBar
	roundTripJSON(t, topBar, &restored)
	if !reflect.DeepEqual(restored, topBar) {
		t.Fatalf("top bar round trip mismatch: got %+v want %+v", restored, topBar)
	}
}

func TestConsoleTopBarAuditChecksTicketCapabilitiesAndShortcuts(t *testing.T) {
	topBar := ConsoleTopBar{
		Name:                      "BigClaw Global Header",
		SearchPlaceholder:         "Search runs, issues, commands",
		EnvironmentOptions:        []string{"Production", "Staging"},
		TimeRangeOptions:          []string{"24h", "7d", "30d"},
		AlertChannels:             []string{"approvals", "sla"},
		DocumentationComplete:     true,
		AccessibilityRequirements: []string{"keyboard-navigation", "screen-reader-label", "focus-visible"},
		CommandEntry: ConsoleCommandEntry{
			TriggerLabel: "Command Menu",
			Placeholder:  "Type a command or jump to a run",
			Shortcut:     "Cmd+K / Ctrl+K",
			Commands: []CommandAction{
				{ID: "search-runs", Title: "Search runs", Section: "Navigate"},
				{ID: "switch-env", Title: "Switch environment", Section: "Context"},
			},
		},
	}

	audit := (ConsoleChromeLibrary{}).AuditTopBar(topBar)
	if audit.Name != "BigClaw Global Header" ||
		len(audit.MissingCapabilities) != 0 ||
		!audit.DocumentationComplete ||
		!audit.AccessibilityComplete ||
		!audit.CommandShortcutSupported ||
		audit.CommandCount != 2 {
		t.Fatalf("unexpected audit: %+v", audit)
	}
	if !audit.ReleaseReady() {
		t.Fatalf("expected release ready top bar")
	}
}

func TestConsoleTopBarAuditFlagsMissingGlobalEntryCapabilities(t *testing.T) {
	topBar := ConsoleTopBar{
		Name:                      "Incomplete Header",
		EnvironmentOptions:        []string{"Production"},
		TimeRangeOptions:          []string{"24h"},
		CommandEntry:              ConsoleCommandEntry{Shortcut: "Cmd+K"},
		DocumentationComplete:     false,
		AccessibilityRequirements: []string{"focus-visible"},
	}

	audit := (ConsoleChromeLibrary{}).AuditTopBar(topBar)
	expectedMissing := []string{"global-search", "time-range-switch", "environment-switch", "alert-entry", "command-shell"}
	if !reflect.DeepEqual(audit.MissingCapabilities, expectedMissing) ||
		audit.DocumentationComplete ||
		audit.AccessibilityComplete ||
		audit.CommandShortcutSupported ||
		audit.ReleaseReady() {
		t.Fatalf("unexpected incomplete top bar audit: %+v", audit)
	}
}

func TestRenderConsoleTopBarReportSummarizesGlobalHeaderAndShell(t *testing.T) {
	topBar := ConsoleTopBar{
		Name:                      "BigClaw Global Header",
		SearchPlaceholder:         "Search runs, issues, commands",
		EnvironmentOptions:        []string{"Production", "Staging"},
		TimeRangeOptions:          []string{"24h", "7d", "30d"},
		AlertChannels:             []string{"approvals", "sla"},
		DocumentationComplete:     true,
		AccessibilityRequirements: []string{"keyboard-navigation", "screen-reader-label", "focus-visible"},
		CommandEntry: ConsoleCommandEntry{
			TriggerLabel: "Command Menu",
			Placeholder:  "Type a command or jump to a run",
			Shortcut:     "Cmd+K / Ctrl+K",
			Commands: []CommandAction{
				{ID: "search-runs", Title: "Search runs", Section: "Navigate", Shortcut: "/"},
				{ID: "open-alerts", Title: "Open alerts", Section: "Monitor"},
			},
		},
	}

	report := RenderConsoleTopBarReport(topBar, (ConsoleChromeLibrary{}).AuditTopBar(topBar))
	for _, fragment := range []string{
		"# Console Top Bar Report",
		"- Command Shortcut: Cmd+K / Ctrl+K",
		"- Release Ready: True",
		"- search-runs: Search runs [Navigate] shortcut=/",
		"- Missing capabilities: none",
		"- Cmd/Ctrl+K supported: True",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
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

func TestUIReviewPackRoundTripPreservesManifestShape(t *testing.T) {
	pack := buildReviewPack()

	var restored UIReviewPack
	roundTripJSON(t, pack, &restored)
	if !reflect.DeepEqual(restored.normalized(), pack.normalized()) {
		t.Fatalf("review pack round trip mismatch: got %+v want %+v", restored, pack)
	}
}

func TestUIReviewPackAuditFlagsMissingSectionsAndCoverageGaps(t *testing.T) {
	pack := UIReviewPack{
		IssueID: "BIG-4204",
		Title:   "UI评审包输出",
		Version: "v4.0-review-pack",
		Objectives: []ReviewObjective{{
			ObjectiveID: "obj-incomplete",
			Title:       "Incomplete objective",
			Persona:     "product-experience",
			Outcome:     "Create a frame for review.",
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

	audit := (UIReviewPackAuditor{}).Audit(pack)
	if audit.Ready ||
		!reflect.DeepEqual(audit.MissingSections, []string{"open_questions"}) ||
		!reflect.DeepEqual(audit.ObjectivesMissingSignals, []string{"obj-incomplete"}) ||
		!reflect.DeepEqual(audit.WireframesMissingBlocks, []string{"wf-empty"}) ||
		!reflect.DeepEqual(audit.InteractionsMissingStates, []string{"flow-empty"}) {
		t.Fatalf("unexpected review pack audit: %+v", audit)
	}
}

func TestUIReviewPackAuditAllowsOpenQuestionsWhileMarkingPackReady(t *testing.T) {
	pack := buildReviewPack()

	audit := (UIReviewPackAuditor{}).Audit(pack)
	if !audit.Ready ||
		!reflect.DeepEqual(audit.UnresolvedQuestionIDs, []string{"oq-mobile-depth"}) ||
		len(audit.MissingSections) != 0 {
		t.Fatalf("unexpected ready review pack audit: %+v", audit)
	}
}

func TestRenderUIReviewPackReportSummarizesReviewShapeAndFindings(t *testing.T) {
	pack := buildReviewPack()
	audit := (UIReviewPackAuditor{}).Audit(pack)

	report := RenderUIReviewPackReport(pack, audit)
	for _, fragment := range []string{
		"# UI Review Pack",
		"- Issue: BIG-4204 UI评审包输出",
		"- Audit: READY: objectives=1 wireframes=1 interactions=1 open_questions=1 checklist=0 decisions=0 role_assignments=0 signoffs=0 blockers=0 timeline_events=0",
		"- obj-alignment: Align reviewers on the release-control story persona=product-experience priority=P0",
		"- Unresolved questions: oq-mobile-depth",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}

func TestUIReviewPackAuditFlagsMissingChecklistCoverageAndEvidence(t *testing.T) {
	pack := BuildBIG4204ReviewPack()
	pack.ReviewerChecklist = []ReviewerChecklistItem{{
		ItemID:        "chk-overview-kpi-scan",
		SurfaceID:     "wf-overview",
		Prompt:        "Verify the KPI strip still supports one-screen executive scanning before drill-down.",
		Owner:         "VP Eng",
		Status:        "ready",
		EvidenceLinks: []string{},
	}}

	audit := (UIReviewPackAuditor{}).Audit(pack)
	if audit.Ready ||
		!reflect.DeepEqual(audit.WireframesMissingChecklists, []string{"wf-queue", "wf-run-detail", "wf-triage"}) ||
		!reflect.DeepEqual(audit.ChecklistItemsMissingEvidence, []string{"chk-overview-kpi-scan"}) ||
		len(audit.OrphanChecklistSurfaces) != 0 {
		t.Fatalf("unexpected checklist audit: %+v", audit)
	}
}

func TestUIReviewPackAuditFlagsMissingDecisionCoverage(t *testing.T) {
	pack := BuildBIG4204ReviewPack()
	pack.DecisionLog = []ReviewDecision{{
		DecisionID: "dec-overview-alert-stack",
		SurfaceID:  "wf-overview",
		Owner:      "product-experience",
		Summary:    "Keep approval and regression alerts in one stacked priority rail.",
		Rationale:  "Reviewers need one comparison lane before jumping into queue or triage surfaces.",
		Status:     "accepted",
	}}

	audit := (UIReviewPackAuditor{}).Audit(pack)
	if audit.Ready ||
		!reflect.DeepEqual(audit.WireframesMissingDecisions, []string{"wf-queue", "wf-run-detail", "wf-triage"}) ||
		len(audit.OrphanDecisionSurfaces) != 0 ||
		len(audit.UnresolvedDecisionIDs) != 0 {
		t.Fatalf("unexpected decision audit: %+v", audit)
	}
}

func TestUIReviewPackAuditFlagsMissingRoleMatrixCoverage(t *testing.T) {
	pack := BuildBIG4204ReviewPack()
	pack.RoleMatrix = []ReviewRoleAssignment{{
		AssignmentID:     "role-overview-vp-eng",
		SurfaceID:        "wf-overview",
		Role:             "VP Eng",
		Responsibilities: []string{"approve overview scan path"},
		ChecklistItemIDs: []string{"chk-overview-kpi-scan"},
		DecisionIDs:      []string{"dec-overview-alert-stack"},
		Status:           "ready",
	}}

	audit := (UIReviewPackAuditor{}).Audit(pack)
	if audit.Ready ||
		!reflect.DeepEqual(audit.WireframesMissingRoleAssignments, []string{"wf-queue", "wf-run-detail", "wf-triage"}) ||
		len(audit.OrphanRoleAssignmentSurfaces) != 0 ||
		len(audit.RoleAssignmentsMissingResponsibilities) != 0 ||
		len(audit.RoleAssignmentsMissingChecklistLinks) != 0 ||
		len(audit.RoleAssignmentsMissingDecisionLinks) != 0 ||
		!reflect.DeepEqual(audit.ChecklistItemsMissingRoleLinks, []string{
			"chk-overview-alert-hierarchy",
			"chk-queue-batch-approval",
			"chk-queue-role-density",
			"chk-run-audit-density",
			"chk-run-replay-context",
			"chk-triage-bulk-assign",
			"chk-triage-handoff-clarity",
		}) ||
		!reflect.DeepEqual(audit.DecisionsMissingRoleLinks, []string{
			"dec-queue-vp-summary",
			"dec-run-detail-audit-rail",
			"dec-triage-handoff-density",
		}) {
		t.Fatalf("unexpected role audit: %+v", audit)
	}
}

func TestUIReviewPackAuditFlagsMissingSignoffCoverageAndAssignmentLinks(t *testing.T) {
	pack := BuildBIG4204ReviewPack()
	pack.SignoffLog = []ReviewSignoff{{
		SignoffID:     "sig-overview-vp-eng",
		AssignmentID:  "role-overview-missing",
		SurfaceID:     "wf-overview",
		Role:          "VP Eng",
		Status:        "approved",
		Required:      true,
		EvidenceLinks: []string{"chk-overview-kpi-scan"},
	}}

	audit := (UIReviewPackAuditor{}).Audit(pack)
	if audit.Ready ||
		!reflect.DeepEqual(audit.WireframesMissingSignoffs, []string{"wf-queue", "wf-run-detail", "wf-triage"}) ||
		len(audit.OrphanSignoffSurfaces) != 0 ||
		!reflect.DeepEqual(audit.SignoffsMissingAssignments, []string{"sig-overview-vp-eng"}) ||
		len(audit.SignoffsMissingEvidence) != 0 ||
		len(audit.UnresolvedRequiredSignoffIDs) != 0 {
		t.Fatalf("unexpected signoff coverage audit: %+v", audit)
	}
}

func TestUIReviewPackAuditFlagsMissingSignoffSLAMetadata(t *testing.T) {
	pack := BuildBIG4204ReviewPack()
	pack.SignoffLog[2] = ReviewSignoff{
		SignoffID:       "sig-run-detail-eng-lead",
		AssignmentID:    "role-run-detail-eng-lead",
		SurfaceID:       "wf-run-detail",
		Role:            "Eng Lead",
		Status:          "pending",
		Required:        true,
		EvidenceLinks:   []string{"chk-run-replay-context", "dec-run-detail-audit-rail"},
		Notes:           "Waiting for final replay-state copy review.",
		RequestedAt:     "",
		DueAt:           "",
		EscalationOwner: "",
		SLAStatus:       "breached",
		ReminderOwner:   "",
		ReminderChannel: "slack",
		LastReminderAt:  "2026-03-14T09:45:00Z",
		NextReminderAt:  "",
	}

	audit := (UIReviewPackAuditor{}).Audit(pack)
	if audit.Ready ||
		!reflect.DeepEqual(audit.SignoffsMissingRequestedDates, []string{"sig-run-detail-eng-lead"}) ||
		!reflect.DeepEqual(audit.SignoffsMissingDueDates, []string{"sig-run-detail-eng-lead"}) ||
		!reflect.DeepEqual(audit.SignoffsMissingEscalationOwners, []string{"sig-run-detail-eng-lead"}) ||
		!reflect.DeepEqual(audit.SignoffsMissingReminderOwners, []string{"sig-run-detail-eng-lead"}) ||
		!reflect.DeepEqual(audit.SignoffsMissingNextReminders, []string{"sig-run-detail-eng-lead"}) ||
		!reflect.DeepEqual(audit.SignoffsMissingReminderCadence, []string{"sig-run-detail-eng-lead"}) ||
		!reflect.DeepEqual(audit.SignoffsWithBreachedSLA, []string{"sig-run-detail-eng-lead"}) {
		t.Fatalf("unexpected signoff sla audit: %+v", audit)
	}
}

func TestUIReviewPackAuditFlagsUnresolvedDecisionWithoutFollowUp(t *testing.T) {
	pack := BuildBIG4204ReviewPack()
	pack.DecisionLog[1] = ReviewDecision{
		DecisionID: "dec-queue-vp-summary",
		SurfaceID:  "wf-queue",
		Owner:      "VP Eng",
		Summary:    "Route VP Eng to a summary-first queue state instead of operator controls.",
		Rationale:  "The VP Eng persona needs queue visibility without accidental action affordances.",
		Status:     "proposed",
		FollowUp:   "",
	}

	audit := (UIReviewPackAuditor{}).Audit(pack)
	if audit.Ready ||
		!reflect.DeepEqual(audit.UnresolvedDecisionIDs, []string{"dec-queue-vp-summary"}) ||
		!reflect.DeepEqual(audit.UnresolvedDecisionsMissingFollowUps, []string{"dec-queue-vp-summary"}) {
		t.Fatalf("unexpected unresolved decision audit: %+v", audit)
	}
}

func TestUIReviewPackAuditFlagsMissingFreezeAndHandoffMetadata(t *testing.T) {
	pack := BuildBIG4204ReviewPack()
	pack.BlockerLog[0] = ReviewBlocker{
		BlockerID:        "blk-run-detail-copy-final",
		SurfaceID:        "wf-run-detail",
		SignoffID:        "sig-run-detail-eng-lead",
		Owner:            "product-experience",
		Summary:          "Replay-state copy still needs final wording review before Eng Lead signoff can close.",
		Status:           "open",
		Severity:         "medium",
		EscalationOwner:  "design-program-manager",
		NextAction:       "Review replay-state copy with Eng Lead and update the run-detail frame in the next critique.",
		FreezeException:  true,
		FreezeOwner:      "",
		FreezeUntil:      "",
		FreezeReason:     "Allow the design sprint review pack to ship while tracked copy cleanup lands in the next critique.",
		FreezeApprovedBy: "",
		FreezeApprovedAt: "",
	}
	pack.BlockerTimeline[1] = ReviewBlockerEvent{
		EventID:     "evt-run-detail-copy-escalated",
		BlockerID:   "blk-run-detail-copy-final",
		Actor:       "design-program-manager",
		Status:      "escalated",
		Summary:     "Scheduled a joint wording review with Eng Lead and product-experience to close the signoff blocker.",
		Timestamp:   "2026-03-14T09:30:00Z",
		NextAction:  "Refresh the run-detail frame annotations after the wording review completes.",
		HandoffFrom: "product-experience",
		HandoffTo:   "",
		Channel:     "design-critique",
		ArtifactRef: "",
	}

	audit := (UIReviewPackAuditor{}).Audit(pack)
	if audit.Ready ||
		!reflect.DeepEqual(audit.FreezeExceptionsMissingOwners, []string{"blk-run-detail-copy-final"}) ||
		!reflect.DeepEqual(audit.FreezeExceptionsMissingUntil, []string{"blk-run-detail-copy-final"}) ||
		!reflect.DeepEqual(audit.FreezeExceptionsMissingApprovers, []string{"blk-run-detail-copy-final"}) ||
		!reflect.DeepEqual(audit.FreezeExceptionsMissingApprovalDates, []string{"blk-run-detail-copy-final"}) ||
		!reflect.DeepEqual(audit.FreezeExceptionsMissingRenewalOwners, []string{"blk-run-detail-copy-final"}) ||
		!reflect.DeepEqual(audit.FreezeExceptionsMissingRenewalDates, []string{"blk-run-detail-copy-final"}) ||
		!reflect.DeepEqual(audit.HandoffEventsMissingTargets, []string{"evt-run-detail-copy-escalated"}) ||
		!reflect.DeepEqual(audit.HandoffEventsMissingArtifacts, []string{"evt-run-detail-copy-escalated"}) ||
		!reflect.DeepEqual(audit.HandoffEventsMissingAckOwners, []string{"evt-run-detail-copy-escalated"}) ||
		!reflect.DeepEqual(audit.HandoffEventsMissingAckDates, []string{"evt-run-detail-copy-escalated"}) {
		t.Fatalf("unexpected freeze and handoff audit: %+v", audit)
	}
}

func TestUIReviewPackAuditFlagsUnresolvedSignoffWithoutBlocker(t *testing.T) {
	pack := BuildBIG4204ReviewPack()
	pack.BlockerLog = nil

	audit := (UIReviewPackAuditor{}).Audit(pack)
	if audit.Ready ||
		!reflect.DeepEqual(audit.UnresolvedRequiredSignoffIDs, []string{"sig-run-detail-eng-lead"}) ||
		!reflect.DeepEqual(audit.UnresolvedRequiredSignoffsWithoutBlockers, []string{"sig-run-detail-eng-lead"}) ||
		len(audit.BlockersMissingSignoffLinks) != 0 ||
		len(audit.BlockersMissingEscalationOwners) != 0 ||
		len(audit.BlockersMissingNextActions) != 0 {
		t.Fatalf("unexpected unresolved signoff audit: %+v", audit)
	}
}

func TestUIReviewPackAuditFlagsWaivedSignoffWithoutMetadata(t *testing.T) {
	pack := BuildBIG4204ReviewPack()
	pack.SignoffLog[2] = ReviewSignoff{
		SignoffID:     "sig-run-detail-eng-lead",
		AssignmentID:  "role-run-detail-eng-lead",
		SurfaceID:     "wf-run-detail",
		Role:          "Eng Lead",
		Status:        "waived",
		Required:      true,
		EvidenceLinks: []string{},
		Notes:         "Design review accepted a temporary waiver pending copy cleanup.",
	}
	pack.BlockerLog = nil
	pack.BlockerTimeline = nil

	audit := (UIReviewPackAuditor{}).Audit(pack)
	if audit.Ready ||
		!reflect.DeepEqual(audit.WaivedSignoffsMissingMetadata, []string{"sig-run-detail-eng-lead"}) ||
		len(audit.SignoffsMissingEvidence) != 0 ||
		len(audit.UnresolvedRequiredSignoffIDs) != 0 {
		t.Fatalf("unexpected waived signoff audit: %+v", audit)
	}
}

func TestUIReviewPackAuditFlagsMissingOrInvalidBlockerTimeline(t *testing.T) {
	pack := BuildBIG4204ReviewPack()
	pack.BlockerTimeline = nil

	audit := (UIReviewPackAuditor{}).Audit(pack)
	if audit.Ready ||
		!reflect.DeepEqual(audit.BlockersMissingTimelineEvents, []string{"blk-run-detail-copy-final"}) ||
		len(audit.ClosedBlockersMissingResolutionEvents) != 0 ||
		len(audit.OrphanBlockerTimelineBlockerIDs) != 0 {
		t.Fatalf("unexpected blocker timeline audit: %+v", audit)
	}
}

func TestUIReviewPackAuditFlagsClosedBlockerWithoutResolutionEventAndOrphans(t *testing.T) {
	pack := BuildBIG4204ReviewPack()
	pack.BlockerLog[0] = ReviewBlocker{
		BlockerID:       "blk-run-detail-copy-final",
		SurfaceID:       "wf-run-detail",
		SignoffID:       "sig-run-detail-eng-lead",
		Owner:           "product-experience",
		Summary:         "Replay-state copy review is closed pending audit trail confirmation.",
		Status:          "closed",
		Severity:        "medium",
		EscalationOwner: "design-program-manager",
		NextAction:      "Archive the blocker after the final review sync.",
	}
	pack.BlockerTimeline = []ReviewBlockerEvent{
		{
			EventID:    "evt-run-detail-copy-opened",
			BlockerID:  "blk-run-detail-copy-final",
			Actor:      "product-experience",
			Status:     "opened",
			Summary:    "Tracked the replay-state wording gap during review prep.",
			Timestamp:  "2026-03-13T10:00:00Z",
			NextAction: "Review wording changes with Eng Lead.",
		},
		{
			EventID:    "evt-orphan-blocker",
			BlockerID:  "blk-missing",
			Actor:      "design-program-manager",
			Status:     "escalated",
			Summary:    "Unexpected timeline entry remained after blocker merge cleanup.",
			Timestamp:  "2026-03-14T11:00:00Z",
			NextAction: "Remove orphaned timeline entry from the bundle.",
		},
	}

	audit := (UIReviewPackAuditor{}).Audit(pack)
	if audit.Ready ||
		len(audit.BlockersMissingTimelineEvents) != 0 ||
		!reflect.DeepEqual(audit.ClosedBlockersMissingResolutionEvents, []string{"blk-run-detail-copy-final"}) ||
		!reflect.DeepEqual(audit.OrphanBlockerTimelineBlockerIDs, []string{"blk-missing"}) {
		t.Fatalf("unexpected closed blocker audit: %+v", audit)
	}
}

func TestRenderUIReviewSignoffAndEscalationDashboards(t *testing.T) {
	pack := BuildBIG4204ReviewPack()

	signoffSLA := RenderUIReviewSignoffSLADashboard(pack)
	signoffReminder := RenderUIReviewSignoffReminderQueue(pack)
	signoffBreach := RenderUIReviewSignoffBreachBoard(pack)
	escalationDashboard := RenderUIReviewEscalationDashboard(pack)
	handoffLedger := RenderUIReviewEscalationHandoffLedger(pack)
	ownerDigest := RenderUIReviewOwnerEscalationDigest(pack)

	for _, tc := range []struct {
		name      string
		body      string
		fragments []string
	}{
		{
			name: "signoff sla",
			body: signoffSLA,
			fragments: []string{
				"# UI Review Sign-off SLA Dashboard",
				"- Sign-offs: 4",
				"- Escalation owners: 4",
				"- at-risk: 1",
				"- met: 3",
				"sig-run-detail-eng-lead: role=Eng Lead surface=wf-run-detail status=pending sla=at-risk requested_at=2026-03-12T11:00:00Z due_at=2026-03-15T18:00:00Z escalation_owner=engineering-director",
			},
		},
		{
			name: "signoff reminder",
			body: signoffReminder,
			fragments: []string{
				"# UI Review Sign-off Reminder Queue",
				"- Reminders: 1",
				"- design-program-manager: reminders=1",
				"rem-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail status=pending sla=at-risk owner=design-program-manager channel=slack",
			},
		},
		{
			name: "signoff breach",
			body: signoffBreach,
			fragments: []string{
				"# UI Review Sign-off Breach Board",
				"- Breach items: 1",
				"- engineering-director: 1",
				"breach-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail status=pending sla=at-risk escalation_owner=engineering-director",
			},
		},
		{
			name: "escalation dashboard",
			body: escalationDashboard,
			fragments: []string{
				"# UI Review Escalation Dashboard",
				"- Items: 2",
				"- design-program-manager: blockers=1 signoffs=0 total=1",
				"- engineering-director: blockers=0 signoffs=1 total=1",
				"esc-sig-run-detail-eng-lead: owner=engineering-director type=signoff source=sig-run-detail-eng-lead surface=wf-run-detail status=pending priority=at-risk current_owner=Eng Lead",
			},
		},
		{
			name: "handoff ledger",
			body: handoffLedger,
			fragments: []string{
				"# UI Review Escalation Handoff Ledger",
				"- Handoffs: 1",
				"- design-critique: 1",
				"handoff-evt-run-detail-copy-escalated: event=evt-run-detail-copy-escalated blocker=blk-run-detail-copy-final surface=wf-run-detail actor=design-program-manager status=escalated at=2026-03-14T09:30:00Z",
				"from=product-experience to=Eng Lead channel=design-critique artifact=wf-run-detail#copy-v5",
			},
		},
		{
			name: "owner digest",
			body: ownerDigest,
			fragments: []string{
				"# UI Review Owner Escalation Digest",
				"- design-program-manager: blockers=1 signoffs=0 reminders=1 freezes=0 handoffs=0 total=2",
				"digest-rem-sig-run-detail-eng-lead: owner=design-program-manager type=reminder source=sig-run-detail-eng-lead surface=wf-run-detail status=pending",
			},
		},
	} {
		for _, fragment := range tc.fragments {
			if !strings.Contains(tc.body, fragment) {
				t.Fatalf("%s: expected %q in report, got %s", tc.name, fragment, tc.body)
			}
		}
	}
}

func TestRenderUIReviewExceptionAndFreezeBoards(t *testing.T) {
	exceptionPack := BuildBIG4204ReviewPack()
	exceptionPack.SignoffLog[2] = ReviewSignoff{
		SignoffID:     "sig-run-detail-eng-lead",
		AssignmentID:  "role-run-detail-eng-lead",
		SurfaceID:     "wf-run-detail",
		Role:          "Eng Lead",
		Status:        "waived",
		EvidenceLinks: []string{"chk-run-replay-context", "dec-run-detail-audit-rail"},
		Notes:         "Temporary waiver approved pending copy lock.",
		WaiverOwner:   "Eng Lead",
		WaiverReason:  "Copy review is deferred to the next wording pass.",
	}
	freezePack := BuildBIG4204ReviewPack()

	exceptionMatrix := RenderUIReviewExceptionMatrix(exceptionPack)
	freezeBoard := RenderUIReviewFreezeExceptionBoard(freezePack)
	freezeTrail := RenderUIReviewFreezeApprovalTrail(freezePack)

	for _, tc := range []struct {
		name      string
		body      string
		fragments []string
	}{
		{
			name: "exception matrix",
			body: exceptionMatrix,
			fragments: []string{
				"# UI Review Exception Matrix",
				"- Exceptions: 2",
				"- Owners: 2",
				"- Surfaces: 1",
				"- Eng Lead: blockers=0 signoffs=1 total=1",
				"- product-experience: blockers=1 signoffs=0 total=1",
				"- open: blockers=1 signoffs=0 total=1",
				"- waived: blockers=0 signoffs=1 total=1",
				"- wf-run-detail: blockers=1 signoffs=1 total=2",
			},
		},
		{
			name: "freeze board",
			body: freezeBoard,
			fragments: []string{
				"# UI Review Freeze Exception Board",
				"- Exceptions: 1",
				"- release-director: blockers=1 signoffs=0 total=1",
				"- wf-run-detail: blockers=1 signoffs=0 total=1",
				"freeze-blk-run-detail-copy-final: owner=release-director type=blocker source=blk-run-detail-copy-final surface=wf-run-detail status=open window=2026-03-18T18:00:00Z",
			},
		},
		{
			name: "freeze trail",
			body: freezeTrail,
			fragments: []string{
				"# UI Review Freeze Approval Trail",
				"- Approvals: 1",
				"- release-director: 1",
				"freeze-approval-blk-run-detail-copy-final: blocker=blk-run-detail-copy-final surface=wf-run-detail status=open owner=release-director approved_by=release-director approved_at=2026-03-14T08:30:00Z window=2026-03-18T18:00:00Z",
			},
		},
	} {
		for _, fragment := range tc.fragments {
			if !strings.Contains(tc.body, fragment) {
				t.Fatalf("%s: expected %q in report, got %s", tc.name, fragment, tc.body)
			}
		}
	}
}

func TestRenderUIReviewOwnerReviewQueue(t *testing.T) {
	pack := BuildBIG4204ReviewPack()

	ownerQueue := RenderUIReviewOwnerReviewQueue(pack)
	for _, fragment := range []string{
		"# UI Review Owner Review Queue",
		"- Owners: 5",
		"- Queue items: 6",
		"- engineering-operations: blockers=0 checklist=1 decisions=0 signoffs=0 total=1",
		"- product-experience: blockers=1 checklist=1 decisions=0 signoffs=0 total=2",
		"- queue-chk-queue-role-density: owner=product-experience type=checklist source=chk-queue-role-density surface=wf-queue status=open",
		"- queue-dec-queue-vp-summary: owner=VP Eng type=decision source=dec-queue-vp-summary surface=wf-queue status=proposed",
		"- queue-sig-run-detail-eng-lead: owner=Eng Lead type=signoff source=sig-run-detail-eng-lead surface=wf-run-detail status=pending",
		"- queue-blk-run-detail-copy-final: owner=product-experience type=blocker source=blk-run-detail-copy-final surface=wf-run-detail status=open",
	} {
		if !strings.Contains(ownerQueue, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, ownerQueue)
		}
	}
}

func TestRenderUIReviewMixedReviewSurfaces(t *testing.T) {
	pack := BuildBIG4204ReviewPack()

	checklistTraceability := RenderUIReviewChecklistTraceabilityBoard(pack)
	decisionFollowup := RenderUIReviewDecisionFollowupTracker(pack)
	reminderCadence := RenderUIReviewReminderCadenceBoard(pack)
	roleCoverage := RenderUIReviewRoleCoverageBoard(pack)
	handoffAck := RenderUIReviewHandoffAckLedger(pack)
	freezeRenewal := RenderUIReviewFreezeRenewalTracker(pack)
	exceptionLog := RenderUIReviewExceptionLog(pack)
	timelineSummary := RenderUIReviewBlockerTimelineSummary(pack)

	for _, tc := range []struct {
		name      string
		body      string
		fragments []string
	}{
		{
			name: "checklist traceability",
			body: checklistTraceability,
			fragments: []string{
				"# UI Review Checklist Traceability Board",
				"- Checklist items: 8",
				"- Owners: 7",
				"trace-chk-queue-role-density: item=chk-queue-role-density surface=wf-queue owner=product-experience status=open linked_roles=product-experience",
			},
		},
		{
			name: "decision followup",
			body: decisionFollowup,
			fragments: []string{
				"# UI Review Decision Follow-up Tracker",
				"- Decisions: 4",
				"- Owners: 4",
				"follow-dec-queue-vp-summary: decision=dec-queue-vp-summary surface=wf-queue owner=VP Eng status=proposed linked_roles=Platform Admin,product-experience",
				"linked_assignments=role-queue-platform-admin,role-queue-product-experience linked_checklists=chk-queue-batch-approval,chk-queue-role-density follow_up=Resolve after the next design critique with policy owners.",
			},
		},
		{
			name: "reminder cadence",
			body: reminderCadence,
			fragments: []string{
				"# UI Review Reminder Cadence Board",
				"- Cadences: 1",
				"cad-rem-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail cadence=daily status=scheduled owner=design-program-manager",
			},
		},
		{
			name: "role coverage",
			body: roleCoverage,
			fragments: []string{
				"# UI Review Role Coverage Board",
				"- Assignments: 8",
				"- Surfaces: 4",
				"cover-role-run-detail-eng-lead: assignment=role-run-detail-eng-lead surface=wf-run-detail role=Eng Lead status=ready responsibilities=2 checklist=1 decisions=1",
				"signoff=sig-run-detail-eng-lead signoff_status=pending",
			},
		},
		{
			name: "handoff ack",
			body: handoffAck,
			fragments: []string{
				"# UI Review Handoff Ack Ledger",
				"- Ack owners: 1",
				"ack-evt-run-detail-copy-escalated: event=evt-run-detail-copy-escalated blocker=blk-run-detail-copy-final surface=wf-run-detail handoff_to=Eng Lead ack_owner=Eng Lead ack_status=acknowledged ack_at=2026-03-14T10:15:00Z",
			},
		},
		{
			name: "freeze renewal",
			body: freezeRenewal,
			fragments: []string{
				"# UI Review Freeze Renewal Tracker",
				"- Renewal owners: 1",
				"renew-blk-run-detail-copy-final: blocker=blk-run-detail-copy-final surface=wf-run-detail status=open renewal_owner=release-director renewal_by=2026-03-17T12:00:00Z renewal_status=review-needed",
			},
		},
		{
			name: "exception log",
			body: exceptionLog,
			fragments: []string{
				"# UI Review Exception Log",
				"- Exceptions: 1",
				"exc-blk-run-detail-copy-final",
				"evt-run-detail-copy-escalated/escalated/design-program-manager@2026-03-14T09:30:00Z",
			},
		},
		{
			name: "timeline summary",
			body: timelineSummary,
			fragments: []string{
				"# UI Review Blocker Timeline Summary",
				"- Events: 2",
				"- opened: 1",
				"- escalated: 1",
				"- design-program-manager: 1",
				"- blk-run-detail-copy-final: latest=evt-run-detail-copy-escalated actor=design-program-manager status=escalated at=2026-03-14T09:30:00Z",
			},
		},
	} {
		for _, fragment := range tc.fragments {
			if !strings.Contains(tc.body, fragment) {
				t.Fatalf("%s: expected %q in report, got %s", tc.name, fragment, tc.body)
			}
		}
	}
}

func TestRenderUIReviewObjectiveWireframeAndQuestionBoards(t *testing.T) {
	pack := BuildBIG4204ReviewPack()

	objectiveCoverage := RenderUIReviewObjectiveCoverageBoard(pack)
	wireframeReadiness := RenderUIReviewWireframeReadinessBoard(pack)
	questionTracker := RenderUIReviewOpenQuestionTracker(pack)

	for _, tc := range []struct {
		name      string
		body      string
		fragments []string
	}{
		{
			name: "objective coverage",
			body: objectiveCoverage,
			fragments: []string{
				"# UI Review Objective Coverage Board",
				"- Objectives: 4",
				"- Personas: 4",
				"- blocked: 1",
				"- covered: 2",
				"objcov-obj-run-detail-investigation: objective=obj-run-detail-investigation persona=Eng Lead priority=P0 coverage=blocked dependencies=3 surfaces=wf-run-detail",
				"dependency_ids=BIG-4203,OPE-72,OPE-73 assignments=role-run-detail-eng-lead checklist=chk-run-replay-context decisions=dec-run-detail-audit-rail signoffs=sig-run-detail-eng-lead blockers=blk-run-detail-copy-final",
			},
		},
		{
			name: "wireframe readiness",
			body: wireframeReadiness,
			fragments: []string{
				"# UI Review Wireframe Readiness Board",
				"- Wireframes: 4",
				"- Devices: 1",
				"- at-risk: 2",
				"- blocked: 1",
				"- ready: 1",
				"wire-wf-run-detail: surface=wf-run-detail device=desktop readiness=blocked open_total=4 entry=/runs/detail",
				"checklist_open=1 decisions_open=0 assignments_open=1 signoffs_open=1 blockers_open=1 signoffs=sig-run-detail-eng-lead blockers=blk-run-detail-copy-final blocks=4 notes=2",
			},
		},
		{
			name: "question tracker",
			body: questionTracker,
			fragments: []string{
				"# UI Review Open Question Tracker",
				"- Questions: 3",
				"- Owners: 3",
				"qtrack-oq-role-density: question=oq-role-density owner=product-experience theme=role-matrix status=open link_status=linked surfaces=wf-queue",
				"checklist=chk-queue-role-density flows=none impact=Changes denial-path copy, button placement, and review criteria for queue and triage pages.",
			},
		},
	} {
		for _, fragment := range tc.fragments {
			if !strings.Contains(tc.body, fragment) {
				t.Fatalf("%s: expected %q in report, got %s", tc.name, fragment, tc.body)
			}
		}
	}
}

func TestInformationArchitectureRoundTripAndRouteResolution(t *testing.T) {
	architecture := InformationArchitecture{
		GlobalNav: []NavigationNode{{
			NodeID:   "ops",
			Title:    "Operations",
			Segment:  "operations",
			ScreenID: "operations-overview",
			Children: []NavigationNode{
				{NodeID: "ops-queue", Title: "Queue Control", Segment: "queue", ScreenID: "queue-control"},
				{NodeID: "ops-triage", Title: "Triage Center", Segment: "triage", ScreenID: "triage-center"},
			},
		}},
		Routes: []NavigationRoute{
			{Path: "/operations", ScreenID: "operations-overview", Title: "Operations", NavNodeID: "ops"},
			{Path: "/operations/queue", ScreenID: "queue-control", Title: "Queue Control", NavNodeID: "ops-queue"},
			{Path: "/operations/triage", ScreenID: "triage-center", Title: "Triage Center", NavNodeID: "ops-triage"},
		},
	}

	var restored InformationArchitecture
	roundTripJSON(t, architecture, &restored)
	for i := range architecture.Routes {
		architecture.Routes[i].Layout = "workspace"
	}
	if !reflect.DeepEqual(restored, architecture) {
		t.Fatalf("ia round trip mismatch: got %+v want %+v", restored, architecture)
	}
	var paths []string
	for _, entry := range architecture.NavigationEntries() {
		paths = append(paths, entry.Path)
	}
	if !reflect.DeepEqual(paths, []string{"/operations", "/operations/queue", "/operations/triage"}) {
		t.Fatalf("unexpected navigation entries: %+v", paths)
	}
	route, ok := architecture.ResolveRoute("operations/queue")
	if !ok || !reflect.DeepEqual(route, NavigationRoute{Path: "/operations/queue", ScreenID: "queue-control", Title: "Queue Control", NavNodeID: "ops-queue", Layout: "workspace"}) {
		t.Fatalf("unexpected resolved route: ok=%v route=%+v", ok, route)
	}
}

func TestInformationArchitectureAuditFlagsDuplicatesSecondaryGapsAndOrphans(t *testing.T) {
	architecture := InformationArchitecture{
		GlobalNav: []NavigationNode{{
			NodeID:   "workbench",
			Title:    "Workbench",
			Segment:  "workbench",
			ScreenID: "workbench-home",
			Children: []NavigationNode{
				{NodeID: "workbench-runs", Title: "Runs", Segment: "runs", ScreenID: "run-index"},
				{NodeID: "workbench-replays", Title: "Replays", Segment: "replays", ScreenID: "replay-index"},
			},
		}},
		Routes: []NavigationRoute{
			{Path: "/workbench/runs", ScreenID: "run-index", Title: "Runs", NavNodeID: "workbench-runs"},
			{Path: "/workbench/runs", ScreenID: "run-index-v2", Title: "Runs V2", NavNodeID: "workbench-runs"},
			{Path: "/settings", ScreenID: "settings-home", Title: "Settings", NavNodeID: "settings"},
		},
	}

	audit := architecture.Audit()
	if audit.Healthy() ||
		!reflect.DeepEqual(audit.DuplicateRoutes, []string{"/workbench/runs"}) ||
		!reflect.DeepEqual(audit.MissingRouteNodes, map[string]string{"workbench": "/workbench", "workbench-replays": "/workbench/replays"}) ||
		!reflect.DeepEqual(audit.SecondaryNavGaps, map[string][]string{"Workbench": []string{"/workbench"}}) ||
		!reflect.DeepEqual(audit.OrphanRoutes, []string{"/settings"}) {
		t.Fatalf("unexpected ia audit: %+v", audit)
	}
}

func TestInformationArchitectureAuditRoundTripAndReport(t *testing.T) {
	audit := InformationArchitectureAudit{
		TotalNavigationNodes: 3,
		TotalRoutes:          2,
		DuplicateRoutes:      []string{"/workbench/runs"},
		MissingRouteNodes:    map[string]string{"workbench": "/workbench"},
		SecondaryNavGaps:     map[string][]string{"Workbench": []string{"/workbench"}},
		OrphanRoutes:         []string{"/settings"},
	}

	var restored InformationArchitectureAudit
	roundTripJSON(t, audit, &restored)
	if !reflect.DeepEqual(restored, audit) {
		t.Fatalf("audit round trip mismatch: got %+v want %+v", restored, audit)
	}

	architecture := InformationArchitecture{
		GlobalNav: []NavigationNode{{NodeID: "workbench", Title: "Workbench", Segment: "workbench", ScreenID: "workbench-home"}},
		Routes:    []NavigationRoute{{Path: "/settings", ScreenID: "settings-home", Title: "Settings", NavNodeID: "settings"}},
	}
	report := RenderInformationArchitectureReport(architecture, audit)
	for _, fragment := range []string{
		"# Information Architecture Report",
		"- Healthy: False",
		"- Workbench (/workbench) screen=workbench-home",
		"- /settings: screen=settings-home title=Settings nav_node=settings",
		"- Duplicate routes: /workbench/runs",
		"- Missing route nodes: workbench=/workbench",
		"- Secondary nav gaps: Workbench=/workbench",
		"- Orphan routes: /settings",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}

func TestUIAcceptanceSuiteRoundTripPreservesAcceptanceManifest(t *testing.T) {
	suite := UIAcceptanceSuite{
		Name:    "BIG-1701 v3.0 UI Acceptance",
		Version: "v3.0",
		RolePermissions: []RolePermissionScenario{{
			ScreenID:     "run-detail",
			AllowedRoles: []string{"admin", "operator"},
			DeniedRoles:  []string{"viewer"},
			AuditEvent:   "ui.access.denied",
		}},
		DataAccuracyChecks: []DataAccuracyCheck{{
			ScreenID:                 "sla-dashboard",
			MetricID:                 "breach-count",
			SourceOfTruth:            "warehouse.sla_daily",
			RenderedValue:            "12",
			Tolerance:                0,
			ObservedDelta:            0,
			FreshnessSLOSeconds:      300,
			ObservedFreshnessSeconds: 120,
		}},
		PerformanceBudgets: []PerformanceBudget{{
			SurfaceID:     "triage-center",
			Interaction:   "initial-load",
			TargetP95MS:   1200,
			ObservedP95MS: 980,
			TargetTTIMS:   1800,
			ObservedTTIMS: 1400,
		}},
		UsabilityJourneys: []UsabilityJourney{{
			JourneyID:          "approve-high-risk-run",
			Personas:           []string{"operator"},
			CriticalSteps:      []string{"open queue", "inspect run", "approve"},
			ExpectedMaxSteps:   4,
			ObservedSteps:      3,
			KeyboardAccessible: true,
			EmptyStateGuidance: true,
			RecoverySupport:    true,
		}},
		AuditRequirements: []AuditRequirement{{
			EventType:             "run.approval.changed",
			RequiredFields:        []string{"run_id", "actor_role", "decision"},
			EmittedFields:         []string{"run_id", "actor_role", "decision"},
			RetentionDays:         90,
			ObservedRetentionDays: 120,
		}},
		DocumentationComplete: true,
	}

	var restored UIAcceptanceSuite
	roundTripJSON(t, suite, &restored)
	if !reflect.DeepEqual(restored, suite) {
		t.Fatalf("ui acceptance round trip mismatch: got %+v want %+v", restored, suite)
	}
}

func TestUIAcceptanceAuditDetectsPermissionAccuracyPerfUsabilityAndAuditGaps(t *testing.T) {
	suite := UIAcceptanceSuite{
		Name:    "BIG-1701 v3.0 UI Acceptance",
		Version: "v3.0",
		RolePermissions: []RolePermissionScenario{{
			ScreenID:     "operations-overview",
			AllowedRoles: []string{"admin"},
		}},
		DataAccuracyChecks: []DataAccuracyCheck{{
			ScreenID:                 "sla-dashboard",
			MetricID:                 "breach-count",
			SourceOfTruth:            "warehouse.sla_daily",
			RenderedValue:            "12",
			Tolerance:                0,
			ObservedDelta:            2,
			FreshnessSLOSeconds:      300,
			ObservedFreshnessSeconds: 901,
		}},
		PerformanceBudgets: []PerformanceBudget{{
			SurfaceID:     "triage-center",
			Interaction:   "initial-load",
			TargetP95MS:   1200,
			ObservedP95MS: 1480,
			TargetTTIMS:   1800,
			ObservedTTIMS: 2400,
		}},
		UsabilityJourneys: []UsabilityJourney{{
			JourneyID:          "reassign-alert",
			Personas:           []string{"operator"},
			CriticalSteps:      []string{"open alert", "assign owner", "save"},
			ExpectedMaxSteps:   3,
			ObservedSteps:      5,
			KeyboardAccessible: false,
			EmptyStateGuidance: true,
			RecoverySupport:    false,
		}},
		AuditRequirements: []AuditRequirement{{
			EventType:             "permission.override.used",
			RequiredFields:        []string{"actor_role", "screen_id", "reason_code"},
			EmittedFields:         []string{"actor_role", "screen_id"},
			RetentionDays:         180,
			ObservedRetentionDays: 30,
		}},
	}

	audit := (UIAcceptanceLibrary{}).Audit(suite)
	expected := UIAcceptanceAudit{
		Name:                      "BIG-1701 v3.0 UI Acceptance",
		Version:                   "v3.0",
		PermissionGaps:            []string{"operations-overview: missing=denied-roles, audit-event"},
		FailingDataChecks:         []string{"sla-dashboard.breach-count: delta=2.0 freshness=901s"},
		FailingPerformanceBudgets: []string{"triage-center.initial-load: p95=1480ms tti=2400ms"},
		FailingUsabilityJourneys:  []string{"reassign-alert: steps=5/3"},
		IncompleteAuditTrails:     []string{"permission.override.used: missing_fields=reason_code retention=30/180d"},
		DocumentationComplete:     false,
	}
	if !reflect.DeepEqual(audit, expected) {
		t.Fatalf("unexpected ui acceptance audit: got %+v want %+v", audit, expected)
	}
	if audit.ReadinessScore() != 0 || audit.ReleaseReady() {
		t.Fatalf("expected non-ready audit, got %+v", audit)
	}
}

func TestRenderUIAcceptanceReportSummarizesReleaseReadiness(t *testing.T) {
	suite := UIAcceptanceSuite{
		Name:    "BIG-1701 v3.0 UI Acceptance",
		Version: "v3.0",
		RolePermissions: []RolePermissionScenario{{
			ScreenID:     "run-detail",
			AllowedRoles: []string{"admin", "operator"},
			DeniedRoles:  []string{"viewer"},
			AuditEvent:   "ui.access.denied",
		}},
		DataAccuracyChecks: []DataAccuracyCheck{{
			ScreenID:                 "sla-dashboard",
			MetricID:                 "breach-count",
			SourceOfTruth:            "warehouse.sla_daily",
			RenderedValue:            "12",
			Tolerance:                0,
			ObservedDelta:            0,
			FreshnessSLOSeconds:      300,
			ObservedFreshnessSeconds: 120,
		}},
		PerformanceBudgets: []PerformanceBudget{{
			SurfaceID:     "triage-center",
			Interaction:   "initial-load",
			TargetP95MS:   1200,
			ObservedP95MS: 980,
			TargetTTIMS:   1800,
			ObservedTTIMS: 1400,
		}},
		UsabilityJourneys: []UsabilityJourney{{
			JourneyID:          "approve-high-risk-run",
			Personas:           []string{"operator"},
			CriticalSteps:      []string{"open queue", "inspect run", "approve"},
			ExpectedMaxSteps:   4,
			ObservedSteps:      3,
			KeyboardAccessible: true,
			EmptyStateGuidance: true,
			RecoverySupport:    true,
		}},
		AuditRequirements: []AuditRequirement{{
			EventType:             "run.approval.changed",
			RequiredFields:        []string{"run_id", "actor_role", "decision"},
			EmittedFields:         []string{"run_id", "actor_role", "decision"},
			RetentionDays:         90,
			ObservedRetentionDays: 120,
		}},
		DocumentationComplete: true,
	}

	report := RenderUIAcceptanceReport(suite, (UIAcceptanceLibrary{}).Audit(suite))
	for _, fragment := range []string{
		"# UI Acceptance Report",
		"- Readiness Score: 100.0",
		"- Release Ready: True",
		"- Role/Permission run-detail: allow=admin, operator deny=viewer audit_event=ui.access.denied",
		"- Data Accuracy sla-dashboard.breach-count: delta=0.0 tolerance=0.0 freshness=120/300s",
		"- Performance triage-center.initial-load: p95=980/1200ms tti=1400/1800ms",
		"- Usability approve-high-risk-run: steps=3/4 keyboard=True empty_state=True recovery=True",
		"- Audit completeness gaps: none",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}

func TestConsoleIARoundTripPreservesManifestShape(t *testing.T) {
	architecture := ConsoleIA{
		Name:    "BigClaw Console IA",
		Version: "v3",
		TopBar: ConsoleTopBar{
			Name:                      "BigClaw Global Header",
			SearchPlaceholder:         "Search runs, issues, commands",
			EnvironmentOptions:        []string{"Production", "Staging"},
			TimeRangeOptions:          []string{"24h", "7d"},
			AlertChannels:             []string{"approvals"},
			DocumentationComplete:     true,
			AccessibilityRequirements: []string{"keyboard-navigation", "screen-reader-label", "focus-visible"},
			CommandEntry: ConsoleCommandEntry{
				TriggerLabel: "Command Menu",
				Placeholder:  "Type a command",
				Shortcut:     "Cmd+K / Ctrl+K",
				Commands:     []CommandAction{{ID: "search-runs", Title: "Search runs", Section: "Navigate"}},
			},
		},
		Navigation: []NavigationItem{{Name: "Overview", Route: "/overview", Section: "Operate", Icon: "dashboard", BadgeCount: 2}},
		Surfaces: []ConsoleSurface{{
			Name:              "Overview",
			Route:             "/overview",
			NavigationSection: "Operate",
			TopBarActions:     []GlobalAction{{ActionID: "refresh", Label: "Refresh", Placement: "topbar"}},
			Filters:           []FilterDefinition{{Name: "Team", Field: "team", Control: "select", Options: []string{"all", "platform"}, DefaultValue: "all"}},
			States: []SurfaceState{
				{Name: "default"},
				{Name: "loading", AllowedActions: []string{"refresh"}},
				{Name: "empty", AllowedActions: []string{"refresh"}},
				{Name: "error", AllowedActions: []string{"refresh"}},
			},
		}},
	}

	var restored ConsoleIA
	roundTripJSON(t, architecture, &restored)
	if !reflect.DeepEqual(restored.normalized(), architecture.normalized()) {
		t.Fatalf("console ia round trip mismatch: got %+v want %+v", restored, architecture)
	}
}

func TestConsoleIAAuditSurfacesGlobalInteractionGaps(t *testing.T) {
	architecture := ConsoleIA{
		Name:    "BigClaw Console IA",
		Version: "v3",
		TopBar: ConsoleTopBar{
			Name:                      "Incomplete Header",
			EnvironmentOptions:        []string{"Production"},
			TimeRangeOptions:          []string{"24h"},
			DocumentationComplete:     false,
			AccessibilityRequirements: []string{"focus-visible"},
			CommandEntry:              ConsoleCommandEntry{Shortcut: "Cmd+K"},
		},
		Navigation: []NavigationItem{
			{Name: "Overview", Route: "/overview", Section: "Operate"},
			{Name: "Ghost", Route: "/ghost", Section: "Operate"},
		},
		Surfaces: []ConsoleSurface{
			{
				Name:              "Overview",
				Route:             "/overview",
				NavigationSection: "Operate",
				TopBarActions:     []GlobalAction{{ActionID: "refresh", Label: "Refresh", Placement: "topbar"}},
				Filters:           []FilterDefinition{{Name: "Team", Field: "team", Control: "select", Options: []string{"all"}}},
				States:            []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"refresh"}}, {Name: "empty", AllowedActions: []string{"refresh"}}, {Name: "error", AllowedActions: []string{"refresh"}}},
			},
			{
				Name:              "Queue",
				Route:             "/queue",
				NavigationSection: "Operate",
				States:            []SurfaceState{{Name: "default"}, {Name: "loading"}, {Name: "empty", AllowedActions: []string{"retry"}}},
			},
		},
	}

	audit := (ConsoleIAAuditor{}).Audit(architecture)
	if !reflect.DeepEqual(audit.SurfacesMissingFilters, []string{"Queue"}) ||
		!reflect.DeepEqual(audit.SurfacesMissingActions, []string{"Queue"}) ||
		!reflect.DeepEqual(audit.TopBarAudit.MissingCapabilities, []string{"global-search", "time-range-switch", "environment-switch", "alert-entry", "command-shell"}) ||
		audit.TopBarAudit.ReleaseReady() ||
		!reflect.DeepEqual(audit.SurfacesMissingStates, map[string][]string{"Queue": []string{"error"}}) ||
		!reflect.DeepEqual(audit.StatesMissingActions, map[string][]string{"Queue": []string{"loading"}}) ||
		!reflect.DeepEqual(audit.UnresolvedStateActions, map[string]map[string][]string{"Queue": {"empty": []string{"retry"}}}) ||
		!reflect.DeepEqual(audit.OrphanNavigationRoutes, []string{"/ghost"}) ||
		!reflect.DeepEqual(audit.UnnavigableSurfaces, []string{"Queue"}) ||
		audit.ReadinessScore() != 0.0 {
		t.Fatalf("unexpected console ia audit: %+v", audit)
	}
}

func TestConsoleIAAuditRoundTripPreservesFindings(t *testing.T) {
	topBarAudit := (ConsoleIAAuditor{}).Audit(ConsoleIA{
		Name:    "BigClaw Console IA",
		Version: "v3",
		TopBar: ConsoleTopBar{
			Name:                      "Incomplete Header",
			EnvironmentOptions:        []string{"Production"},
			TimeRangeOptions:          []string{"24h"},
			DocumentationComplete:     false,
			AccessibilityRequirements: []string{"focus-visible"},
			CommandEntry:              ConsoleCommandEntry{Shortcut: "Cmd+K"},
		},
	}).TopBarAudit

	audit := ConsoleIAAudit{
		SystemName:             "BigClaw Console IA",
		Version:                "v3",
		SurfaceCount:           2,
		NavigationCount:        1,
		TopBarAudit:            topBarAudit,
		SurfacesMissingFilters: []string{"Queue"},
		SurfacesMissingActions: []string{"Queue"},
		SurfacesMissingStates:  map[string][]string{"Queue": []string{"error"}},
		StatesMissingActions:   map[string][]string{"Queue": []string{"loading"}},
		UnresolvedStateActions: map[string]map[string][]string{"Queue": {"empty": []string{"retry"}}},
		OrphanNavigationRoutes: []string{"/ghost"},
		UnnavigableSurfaces:    []string{"Queue"},
	}

	var restored ConsoleIAAudit
	roundTripJSON(t, audit, &restored)
	if !reflect.DeepEqual(restored, audit) {
		t.Fatalf("console ia audit round trip mismatch: got %+v want %+v", restored, audit)
	}
}

func TestRenderConsoleIAReportSummarizesSurfaceCoverage(t *testing.T) {
	architecture := ConsoleIA{
		Name:    "BigClaw Console IA",
		Version: "v3",
		TopBar: ConsoleTopBar{
			Name:                      "BigClaw Global Header",
			SearchPlaceholder:         "Search runs, issues, commands",
			EnvironmentOptions:        []string{"Production", "Staging"},
			TimeRangeOptions:          []string{"24h", "7d", "30d"},
			AlertChannels:             []string{"approvals", "sla"},
			DocumentationComplete:     true,
			AccessibilityRequirements: []string{"keyboard-navigation", "screen-reader-label", "focus-visible"},
			CommandEntry: ConsoleCommandEntry{
				TriggerLabel: "Command Menu",
				Placeholder:  "Type a command or jump to a run",
				Shortcut:     "Cmd+K / Ctrl+K",
				Commands: []CommandAction{
					{ID: "search-runs", Title: "Search runs", Section: "Navigate", Shortcut: "/"},
					{ID: "open-alerts", Title: "Open alerts", Section: "Monitor"},
				},
			},
		},
		Navigation: []NavigationItem{{Name: "Overview", Route: "/overview", Section: "Operate"}},
		Surfaces: []ConsoleSurface{{
			Name:              "Overview",
			Route:             "/overview",
			NavigationSection: "Operate",
			TopBarActions:     []GlobalAction{{ActionID: "refresh", Label: "Refresh", Placement: "topbar"}},
			Filters:           []FilterDefinition{{Name: "Team", Field: "team", Control: "select", Options: []string{"all"}}},
			States: []SurfaceState{
				{Name: "default"},
				{Name: "loading", AllowedActions: []string{"refresh"}},
				{Name: "empty", AllowedActions: []string{"refresh"}},
				{Name: "error", AllowedActions: []string{"refresh"}},
			},
		}},
	}

	report := RenderConsoleIAReport(architecture, (ConsoleIAAuditor{}).Audit(architecture))
	for _, fragment := range []string{
		"# Console Information Architecture Report",
		"- Name: BigClaw Global Header",
		"- Release Ready: True",
		"- Navigation Items: 1",
		"- Overview: route=/overview filters=Team actions=Refresh states=default, loading, empty, error missing_states=none states_without_actions=none unresolved_state_actions=none",
		"- Surfaces missing filters: none",
		"- Undefined state actions: none",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}

func TestConsoleInteractionDraftRoundTripPreservesFourPageManifest(t *testing.T) {
	draft := ConsoleInteractionDraft{
		Name:    "BIG-4203 Four Critical Pages",
		Version: "v1",
		Architecture: ConsoleIA{
			Name:    "BigClaw Console IA",
			Version: "v3",
			TopBar: ConsoleTopBar{
				Name:                      "BigClaw Global Header",
				SearchPlaceholder:         "Search runs, issues, commands",
				EnvironmentOptions:        []string{"Production", "Staging"},
				TimeRangeOptions:          []string{"24h", "7d"},
				AlertChannels:             []string{"approvals"},
				DocumentationComplete:     true,
				AccessibilityRequirements: []string{"keyboard-navigation", "screen-reader-label", "focus-visible"},
				CommandEntry: ConsoleCommandEntry{
					TriggerLabel: "Command Menu",
					Placeholder:  "Type a command",
					Shortcut:     "Cmd+K / Ctrl+K",
					Commands:     []CommandAction{{ID: "search-runs", Title: "Search runs", Section: "Navigate"}},
				},
			},
			Navigation: []NavigationItem{{Name: "Overview", Route: "/overview", Section: "Operate"}, {Name: "Queue", Route: "/queue", Section: "Operate"}, {Name: "Run Detail", Route: "/runs/detail", Section: "Operate"}, {Name: "Triage", Route: "/triage", Section: "Operate"}},
			Surfaces:   []ConsoleSurface{{Name: "Overview", Route: "/overview", NavigationSection: "Operate"}, {Name: "Queue", Route: "/queue", NavigationSection: "Operate"}, {Name: "Run Detail", Route: "/runs/detail", NavigationSection: "Operate"}, {Name: "Triage", Route: "/triage", NavigationSection: "Operate"}},
		},
		Contracts: []SurfaceInteractionContract{
			{SurfaceName: "Overview"},
			{SurfaceName: "Queue", RequiresBatchActions: true},
			{SurfaceName: "Run Detail"},
			{SurfaceName: "Triage"},
		},
	}

	var restored ConsoleInteractionDraft
	roundTripJSON(t, draft, &restored)
	for i := range draft.Contracts {
		draft.Contracts[i].RequiresFilters = true
		draft.Contracts[i].RequiredStates = []string{"default", "empty", "error", "loading"}
	}
	if !reflect.DeepEqual(restored, draft) {
		t.Fatalf("console interaction draft round trip mismatch: got %+v want %+v", restored, draft)
	}
}

func TestConsoleInteractionAuditSurfacesMissingActionsPermissionsAndBatchOps(t *testing.T) {
	architecture := ConsoleIA{
		Name:    "BigClaw Console IA",
		Version: "v3",
		TopBar: ConsoleTopBar{
			Name:                      "BigClaw Global Header",
			SearchPlaceholder:         "Search runs, issues, commands",
			EnvironmentOptions:        []string{"Production", "Staging"},
			TimeRangeOptions:          []string{"24h", "7d"},
			AlertChannels:             []string{"approvals"},
			DocumentationComplete:     true,
			AccessibilityRequirements: []string{"keyboard-navigation", "screen-reader-label", "focus-visible"},
			CommandEntry: ConsoleCommandEntry{
				TriggerLabel: "Command Menu",
				Placeholder:  "Type a command",
				Shortcut:     "Cmd+K / Ctrl+K",
				Commands:     []CommandAction{{ID: "search-runs", Title: "Search runs", Section: "Navigate"}},
			},
		},
		Navigation: []NavigationItem{{Name: "Overview", Route: "/overview", Section: "Operate"}, {Name: "Queue", Route: "/queue", Section: "Operate"}, {Name: "Run Detail", Route: "/runs/detail", Section: "Operate"}, {Name: "Triage", Route: "/triage", Section: "Operate"}},
		Surfaces: []ConsoleSurface{
			{Name: "Overview", Route: "/overview", NavigationSection: "Operate", TopBarActions: []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}}, Filters: []FilterDefinition{{Name: "Team", Field: "team", Control: "select", Options: []string{"all"}}}, States: []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"export"}}, {Name: "empty", AllowedActions: []string{"export"}}, {Name: "error", AllowedActions: []string{"audit"}}}},
			{Name: "Queue", Route: "/queue", NavigationSection: "Operate", TopBarActions: []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}}, Filters: []FilterDefinition{{Name: "Status", Field: "status", Control: "select", Options: []string{"all"}}}, States: []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"audit"}}, {Name: "empty", AllowedActions: []string{"audit"}}}},
			{Name: "Run Detail", Route: "/runs/detail", NavigationSection: "Operate", TopBarActions: []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}}, Filters: []FilterDefinition{{Name: "Run", Field: "run_id", Control: "search"}}, States: []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"export"}}, {Name: "empty", AllowedActions: []string{"drill-down"}}, {Name: "error", AllowedActions: []string{"audit"}}}},
			{Name: "Triage", Route: "/triage", NavigationSection: "Operate", TopBarActions: []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}, {ActionID: "bulk-assign", Label: "Bulk Assign", Placement: "topbar", RequiresSelection: true}}, States: []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"export"}}, {Name: "empty", AllowedActions: []string{"audit"}}, {Name: "error", AllowedActions: []string{"audit"}}}},
		},
	}
	draft := ConsoleInteractionDraft{
		Name:         "BIG-4203 Four Critical Pages",
		Version:      "v1",
		Architecture: architecture,
		Contracts: []SurfaceInteractionContract{
			{SurfaceName: "Overview", RequiredActionIDs: []string{"drill-down", "export", "audit"}, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "overview.access.denied"}},
			{SurfaceName: "Queue", RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiresBatchActions: true, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}}},
			{SurfaceName: "Run Detail", RequiredActionIDs: []string{"drill-down", "export", "audit"}, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator", "viewer"}, DeniedRoles: []string{}, AuditEvent: "run-detail.access.denied"}},
			{SurfaceName: "Triage", RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiresFilters: true, RequiresBatchActions: true, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "triage.access.denied"}},
		},
	}

	audit := (ConsoleInteractionAuditor{}).Audit(draft)
	if audit.Name != "BIG-4203 Four Critical Pages" ||
		audit.Version != "v1" ||
		audit.ContractCount != 4 ||
		len(audit.MissingSurfaces) != 0 ||
		!reflect.DeepEqual(audit.SurfacesMissingFilters, []string{"Triage"}) ||
		!reflect.DeepEqual(audit.SurfacesMissingActions, map[string][]string{"Queue": []string{"export"}}) ||
		!reflect.DeepEqual(audit.SurfacesMissingBatchActions, []string{"Queue"}) ||
		!reflect.DeepEqual(audit.SurfacesMissingStates, map[string][]string{"Queue": []string{"error"}}) ||
		!reflect.DeepEqual(audit.PermissionGaps, map[string][]string{"Queue": []string{"audit-event"}, "Run Detail": []string{"denied-roles"}}) {
		t.Fatalf("unexpected console interaction audit: %+v", audit)
	}
	if audit.ReadinessScore() != 0 || audit.ReleaseReady() {
		t.Fatalf("expected non-ready interaction audit, got %+v", audit)
	}
}

func TestRenderConsoleInteractionReportSummarizesCriticalPageContracts(t *testing.T) {
	draft := ConsoleInteractionDraft{
		Name:    "BIG-4203 Four Critical Pages",
		Version: "v1",
		Architecture: ConsoleIA{
			Name:    "BigClaw Console IA",
			Version: "v3",
			TopBar: ConsoleTopBar{
				Name:                      "BigClaw Global Header",
				SearchPlaceholder:         "Search runs, issues, commands",
				EnvironmentOptions:        []string{"Production", "Staging"},
				TimeRangeOptions:          []string{"24h", "7d"},
				AlertChannels:             []string{"approvals"},
				DocumentationComplete:     true,
				AccessibilityRequirements: []string{"keyboard-navigation", "screen-reader-label", "focus-visible"},
				CommandEntry:              ConsoleCommandEntry{TriggerLabel: "Command Menu", Placeholder: "Type a command", Shortcut: "Cmd+K / Ctrl+K", Commands: []CommandAction{{ID: "search-runs", Title: "Search runs", Section: "Navigate"}}},
			},
			Navigation: []NavigationItem{{Name: "Overview", Route: "/overview", Section: "Operate"}, {Name: "Queue", Route: "/queue", Section: "Operate"}, {Name: "Run Detail", Route: "/runs/detail", Section: "Operate"}, {Name: "Triage", Route: "/triage", Section: "Operate"}},
			Surfaces: []ConsoleSurface{
				{Name: "Overview", Route: "/overview", NavigationSection: "Operate", TopBarActions: []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}}, Filters: []FilterDefinition{{Name: "Team", Field: "team", Control: "select", Options: []string{"all"}}}, States: []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"export"}}, {Name: "empty", AllowedActions: []string{"drill-down"}}, {Name: "error", AllowedActions: []string{"audit"}}}},
				{Name: "Queue", Route: "/queue", NavigationSection: "Operate", TopBarActions: []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}, {ActionID: "bulk-approve", Label: "Bulk Approve", Placement: "topbar", RequiresSelection: true}}, Filters: []FilterDefinition{{Name: "Status", Field: "status", Control: "select", Options: []string{"all"}}}, States: []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"export"}}, {Name: "empty", AllowedActions: []string{"audit"}}, {Name: "error", AllowedActions: []string{"audit"}}}},
				{Name: "Run Detail", Route: "/runs/detail", NavigationSection: "Operate", TopBarActions: []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}}, Filters: []FilterDefinition{{Name: "Run", Field: "run_id", Control: "search"}}, States: []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"export"}}, {Name: "empty", AllowedActions: []string{"drill-down"}}, {Name: "error", AllowedActions: []string{"audit"}}}},
				{Name: "Triage", Route: "/triage", NavigationSection: "Operate", TopBarActions: []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}, {ActionID: "bulk-assign", Label: "Bulk Assign", Placement: "topbar", RequiresSelection: true}}, Filters: []FilterDefinition{{Name: "Severity", Field: "severity", Control: "select", Options: []string{"all"}}}, States: []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"export"}}, {Name: "empty", AllowedActions: []string{"audit"}}, {Name: "error", AllowedActions: []string{"audit"}}}},
			},
		},
		Contracts: []SurfaceInteractionContract{
			{SurfaceName: "Overview", RequiredActionIDs: []string{"drill-down", "export", "audit"}, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "overview.access.denied"}},
			{SurfaceName: "Queue", RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiresBatchActions: true, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "queue.access.denied"}},
			{SurfaceName: "Run Detail", RequiredActionIDs: []string{"drill-down", "export", "audit"}, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator", "viewer"}, DeniedRoles: []string{"guest"}, AuditEvent: "run-detail.access.denied"}},
			{SurfaceName: "Triage", RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiresBatchActions: true, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "triage.access.denied"}},
		},
	}
	report := RenderConsoleInteractionReport(draft, (ConsoleInteractionAuditor{}).Audit(draft))
	for _, fragment := range []string{
		"# Console Interaction Draft Report",
		"- Critical Pages: 4",
		"- Required Roles: none",
		"- Readiness Score: 100.0",
		"- Release Ready: True",
		"- Overview: route=/overview required_actions=drill-down, export, audit available_actions=drill-down, export, audit filters=1 states=default, loading, empty, error batch=optional permissions=complete",
		"- Queue: route=/queue required_actions=drill-down, export, audit available_actions=drill-down, export, audit, bulk-approve filters=1 states=default, loading, empty, error batch=required permissions=complete",
		"- Permission gaps: none",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}

func TestBuildBIG4203ConsoleInteractionDraftIsReleaseReady(t *testing.T) {
	draft := BuildBIG4203ConsoleInteractionDraft()
	audit := (ConsoleInteractionAuditor{}).Audit(draft)
	report := RenderConsoleInteractionReport(draft, audit)
	if !reflect.DeepEqual(draft.RequiredRoles, []string{"eng-lead", "platform-admin", "vp-eng", "cross-team-operator"}) ||
		!draft.RequiresFrameContracts || !audit.ReleaseReady() || len(audit.UncoveredRoles) != 0 {
		t.Fatalf("unexpected BIG-4203 draft/audit: draft=%+v audit=%+v", draft, audit)
	}
	for _, fragment := range []string{
		"- Required Roles: eng-lead, platform-admin, vp-eng, cross-team-operator",
		"persona=VP Eng wireframe=wf-overview",
		"review_focus=metric hierarchy,drill-down posture,alert prioritization",
		"- Uncovered roles: none",
		"- Pages missing personas: none",
		"- Pages missing wireframe links: none",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}

func TestConsoleInteractionAuditFlagsUncoveredRequiredRoles(t *testing.T) {
	draft := BuildBIG4203ConsoleInteractionDraft()
	draft.RequiredRoles = append(draft.RequiredRoles, "finance-reviewer")
	audit := (ConsoleInteractionAuditor{}).Audit(draft)
	if !reflect.DeepEqual(audit.UncoveredRoles, []string{"finance-reviewer"}) || audit.ReleaseReady() {
		t.Fatalf("unexpected uncovered role audit: %+v", audit)
	}
}

func TestConsoleInteractionAuditFlagsMissingFrameContractDetails(t *testing.T) {
	draft := BuildBIG4203ConsoleInteractionDraft()
	draft.Contracts[0].PrimaryPersona = ""
	draft.Contracts[0].LinkedWireframeID = ""
	draft.Contracts[0].ReviewFocusAreas = nil
	draft.Contracts[0].DecisionPrompts = nil
	audit := (ConsoleInteractionAuditor{}).Audit(draft)
	if !reflect.DeepEqual(audit.SurfacesMissingPrimaryPersonas, []string{"Overview"}) ||
		!reflect.DeepEqual(audit.SurfacesMissingWireframeLinks, []string{"Overview"}) ||
		!reflect.DeepEqual(audit.SurfacesMissingReviewFocus, []string{"Overview"}) ||
		!reflect.DeepEqual(audit.SurfacesMissingDecisionPrompts, []string{"Overview"}) ||
		audit.ReleaseReady() {
		t.Fatalf("unexpected missing frame contract audit: %+v", audit)
	}
}

func roundTripJSON(t *testing.T, input any, target any) {
	t.Helper()
	body, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
}
