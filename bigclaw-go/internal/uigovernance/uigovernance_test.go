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
