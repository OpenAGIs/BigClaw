package ui

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

	if !component.ReleaseReady(map[string]struct{}{
		"color.action.primary": {},
		"spacing.control.md":   {},
	}) {
		t.Fatalf("expected component to be release ready")
	}
	if got, want := component.TokenNames(), []string{"color.action.primary", "spacing.control.md"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("token names = %v, want %v", got, want)
	}
	if got := component.MissingRequiredStates(); len(got) != 0 {
		t.Fatalf("missing required states = %v, want none", got)
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
			Name:                  "Button",
			Readiness:             "stable",
			Slots:                 []string{"icon", "label"},
			DocumentationComplete: true,
			AccessibilityRequirements: []string{
				"focus-visible",
			},
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
	if !reflect.DeepEqual(restored, system) {
		t.Fatalf("restored system = %#v, want %#v", restored, system)
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
				Variants: []ComponentVariant{{
					Name:   "primary",
					Tokens: []string{"color.action.primary", "spacing.control.md"},
					States: []string{"default", "hover", "disabled"},
				}},
			},
			{
				Name:                  "CommandBar",
				Readiness:             "beta",
				DocumentationComplete: false,
				Variants: []ComponentVariant{{
					Name:   "global",
					Tokens: []string{"spacing.control.md"},
					States: []string{"default", "hover"},
				}},
			},
		},
	}

	audit := ComponentLibrary{}.Audit(system)
	if got, want := audit.ReleaseReadyComponents, []string{"Button"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("release ready components = %v, want %v", got, want)
	}
	if got, want := audit.ComponentsMissingDocs, []string{"CommandBar"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("components missing docs = %v, want %v", got, want)
	}
	if got, want := audit.ComponentsMissingAccessibility, []string{"CommandBar"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("components missing accessibility = %v, want %v", got, want)
	}
	if got, want := audit.ComponentsMissingStates, []string{"CommandBar"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("components missing states = %v, want %v", got, want)
	}
	if len(audit.UndefinedTokenRefs) != 0 {
		t.Fatalf("undefined token refs = %v, want none", audit.UndefinedTokenRefs)
	}
	if got, want := audit.TokenOrphans, []string{"radius.md"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("token orphans = %v, want %v", got, want)
	}
	if audit.ReadinessScore != 35.0 {
		t.Fatalf("readiness score = %v, want 35.0", audit.ReadinessScore)
	}
}

func TestDesignSystemAuditFlagsUndefinedTokenReferences(t *testing.T) {
	system := DesignSystem{
		Name:    "BigClaw Console UI",
		Version: "v2",
		Tokens:  []DesignToken{{Name: "spacing.control.md", Category: "spacing", Value: "12px"}},
		Components: []ComponentSpec{{
			Name:                  "SideNav",
			Readiness:             "stable",
			DocumentationComplete: true,
			AccessibilityRequirements: []string{
				"focus-visible",
			},
			Variants: []ComponentVariant{{
				Name:   "default",
				Tokens: []string{"spacing.control.md", "color.surface.nav"},
				States: []string{"default", "hover", "disabled"},
			}},
		}},
	}

	audit := ComponentLibrary{}.Audit(system)
	if len(audit.ReleaseReadyComponents) != 0 {
		t.Fatalf("release ready components = %v, want none", audit.ReleaseReadyComponents)
	}
	if got, want := audit.UndefinedTokenRefs, map[string][]string{"SideNav": {"color.surface.nav"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("undefined token refs = %v, want %v", got, want)
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
		UndefinedTokenRefs:             map[string][]string{"SideNav": {"color.surface.nav"}},
		TokenOrphans:                   []string{"radius.md"},
	}

	var restored DesignSystemAudit
	roundTripJSON(t, audit, &restored)
	if !reflect.DeepEqual(restored, audit) {
		t.Fatalf("restored audit = %#v, want %#v", restored, audit)
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
			Name:                  "Button",
			Readiness:             "stable",
			DocumentationComplete: true,
			AccessibilityRequirements: []string{
				"focus-visible",
			},
			Variants: []ComponentVariant{{
				Name:   "primary",
				Tokens: []string{"color.action.primary", "spacing.control.md"},
				States: []string{"default", "hover", "disabled"},
			}},
		}},
	}

	audit := ComponentLibrary{}.Audit(system)
	report := RenderDesignSystemReport(system, audit)
	assertContainsAll(t, report,
		"# Design System Report",
		"- Release Ready Components: 1",
		"- color: 1",
		"- Button: readiness=stable docs=True a11y=True states=default, hover, disabled missing_states=none undefined_tokens=none",
		"- Missing interaction states: none",
		"- Undefined token refs: none",
		"- Orphan tokens: none",
	)
}

func TestConsoleTopBarRoundTripPreservesCommandEntryManifest(t *testing.T) {
	topBar := ConsoleTopBar{
		Name:              "BigClaw Global Header",
		SearchPlaceholder: "Search runs, issues, commands",
		EnvironmentOptions: []string{
			"Production", "Staging",
		},
		TimeRangeOptions:      []string{"24h", "7d", "30d"},
		AlertChannels:         []string{"approvals", "sla"},
		DocumentationComplete: true,
		AccessibilityRequirements: []string{
			"keyboard-navigation", "screen-reader-label", "focus-visible",
		},
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
		t.Fatalf("restored top bar = %#v, want %#v", restored, topBar)
	}
}

func TestConsoleTopBarAuditChecksTicketCapabilitiesAndShortcuts(t *testing.T) {
	topBar := ConsoleTopBar{
		Name:              "BigClaw Global Header",
		SearchPlaceholder: "Search runs, issues, commands",
		EnvironmentOptions: []string{
			"Production", "Staging",
		},
		TimeRangeOptions:      []string{"24h", "7d", "30d"},
		AlertChannels:         []string{"approvals", "sla"},
		DocumentationComplete: true,
		AccessibilityRequirements: []string{
			"keyboard-navigation", "screen-reader-label", "focus-visible",
		},
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

	audit := ConsoleChromeLibrary{}.AuditTopBar(topBar)
	want := ConsoleTopBarAudit{
		Name:                     "BigClaw Global Header",
		MissingCapabilities:      []string{},
		DocumentationComplete:    true,
		AccessibilityComplete:    true,
		CommandShortcutSupported: true,
		CommandCount:             2,
	}
	if !reflect.DeepEqual(audit, want) {
		t.Fatalf("top bar audit = %#v, want %#v", audit, want)
	}
	if !audit.ReleaseReady() {
		t.Fatalf("expected top bar audit to be release ready")
	}
}

func TestConsoleTopBarAuditFlagsMissingGlobalEntryCapabilities(t *testing.T) {
	topBar := ConsoleTopBar{
		Name:               "Incomplete Header",
		EnvironmentOptions: []string{"Production"},
		TimeRangeOptions:   []string{"24h"},
		CommandEntry: ConsoleCommandEntry{
			Shortcut: "Cmd+K",
		},
		DocumentationComplete:     false,
		AccessibilityRequirements: []string{"focus-visible"},
	}

	audit := ConsoleChromeLibrary{}.AuditTopBar(topBar)
	if got, want := audit.MissingCapabilities, []string{"global-search", "time-range-switch", "environment-switch", "alert-entry", "command-shell"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("missing capabilities = %v, want %v", got, want)
	}
	if audit.DocumentationComplete {
		t.Fatalf("expected documentation_complete to be false")
	}
	if audit.AccessibilityComplete {
		t.Fatalf("expected accessibility_complete to be false")
	}
	if audit.CommandShortcutSupported {
		t.Fatalf("expected command_shortcut_supported to be false")
	}
	if audit.ReleaseReady() {
		t.Fatalf("expected release_ready to be false")
	}
}

func TestRenderConsoleTopBarReportSummarizesGlobalHeaderAndShell(t *testing.T) {
	topBar := ConsoleTopBar{
		Name:              "BigClaw Global Header",
		SearchPlaceholder: "Search runs, issues, commands",
		EnvironmentOptions: []string{
			"Production", "Staging",
		},
		TimeRangeOptions:      []string{"24h", "7d", "30d"},
		AlertChannels:         []string{"approvals", "sla"},
		DocumentationComplete: true,
		AccessibilityRequirements: []string{
			"keyboard-navigation", "screen-reader-label", "focus-visible",
		},
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

	audit := ConsoleChromeLibrary{}.AuditTopBar(topBar)
	report := RenderConsoleTopBarReport(topBar, audit)
	assertContainsAll(t, report,
		"# Console Top Bar Report",
		"- Command Shortcut: Cmd+K / Ctrl+K",
		"- Release Ready: True",
		"- search-runs: Search runs [Navigate] shortcut=/",
		"- Missing capabilities: none",
		"- Cmd/Ctrl+K supported: True",
	)
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
	if !reflect.DeepEqual(restored, architecture) {
		t.Fatalf("restored architecture = %#v, want %#v", restored, architecture)
	}

	entries := architecture.NavigationEntries()
	gotPaths := []string{entries[0].Path, entries[1].Path, entries[2].Path}
	wantPaths := []string{"/operations", "/operations/queue", "/operations/triage"}
	if !reflect.DeepEqual(gotPaths, wantPaths) {
		t.Fatalf("navigation entry paths = %v, want %v", gotPaths, wantPaths)
	}

	route, ok := architecture.ResolveRoute("operations/queue")
	if !ok {
		t.Fatalf("expected route resolution to succeed")
	}
	wantRoute := NavigationRoute{
		Path:      "/operations/queue",
		ScreenID:  "queue-control",
		Title:     "Queue Control",
		NavNodeID: "ops-queue",
	}
	if !reflect.DeepEqual(route, wantRoute) {
		t.Fatalf("resolved route = %#v, want %#v", route, wantRoute)
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
	if audit.Healthy() {
		t.Fatalf("expected architecture audit to be unhealthy")
	}
	if got, want := audit.DuplicateRoutes, []string{"/workbench/runs"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("duplicate routes = %v, want %v", got, want)
	}
	if got, want := audit.MissingRouteNodes, map[string]string{"workbench": "/workbench", "workbench-replays": "/workbench/replays"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("missing route nodes = %v, want %v", got, want)
	}
	if got, want := audit.SecondaryNavGaps, map[string][]string{"Workbench": {"/workbench"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("secondary nav gaps = %v, want %v", got, want)
	}
	if got, want := audit.OrphanRoutes, []string{"/settings"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("orphan routes = %v, want %v", got, want)
	}
}

func TestInformationArchitectureAuditRoundTripAndReport(t *testing.T) {
	audit := InformationArchitectureAudit{
		TotalNavigationNodes: 3,
		TotalRoutes:          2,
		DuplicateRoutes:      []string{"/workbench/runs"},
		MissingRouteNodes:    map[string]string{"workbench": "/workbench"},
		SecondaryNavGaps:     map[string][]string{"Workbench": {"/workbench"}},
		OrphanRoutes:         []string{"/settings"},
	}

	var restored InformationArchitectureAudit
	roundTripJSON(t, audit, &restored)
	if !reflect.DeepEqual(restored, audit) {
		t.Fatalf("restored architecture audit = %#v, want %#v", restored, audit)
	}

	architecture := InformationArchitecture{
		GlobalNav: []NavigationNode{{NodeID: "workbench", Title: "Workbench", Segment: "workbench", ScreenID: "workbench-home"}},
		Routes:    []NavigationRoute{{Path: "/settings", ScreenID: "settings-home", Title: "Settings", NavNodeID: "settings"}},
	}
	report := RenderInformationArchitectureReport(architecture, audit)
	assertContainsAll(t, report,
		"# Information Architecture Report",
		"- Healthy: False",
		"- Workbench (/workbench) screen=workbench-home",
		"- /settings: screen=settings-home title=Settings nav_node=settings",
		"- Duplicate routes: /workbench/runs",
		"- Missing route nodes: workbench=/workbench",
		"- Secondary nav gaps: Workbench=/workbench",
		"- Orphan routes: /settings",
	)
}

func TestUIAcceptanceSuiteRoundTripPreservesAcceptanceManifest(t *testing.T) {
	suite := passingAcceptanceSuite()
	var restored UIAcceptanceSuite
	roundTripJSON(t, suite, &restored)
	if !reflect.DeepEqual(restored, suite) {
		t.Fatalf("restored acceptance suite = %#v, want %#v", restored, suite)
	}
}

func TestUIAcceptanceAuditDetectsPermissionAccuracyPerfUsabilityAndAuditGaps(t *testing.T) {
	suite := UIAcceptanceSuite{
		Name:    "BIG-1701 v3.0 UI Acceptance",
		Version: "v3.0",
		RolePermissions: []RolePermissionScenario{{
			ScreenID:     "operations-overview",
			AllowedRoles: []string{"admin"},
			DeniedRoles:  []string{},
			AuditEvent:   "",
		}},
		DataAccuracyChecks: []DataAccuracyCheck{{
			ScreenID:                 "sla-dashboard",
			MetricID:                 "breach-count",
			SourceOfTruth:            "warehouse.sla_daily",
			RenderedValue:            "12",
			Tolerance:                0.0,
			ObservedDelta:            2.0,
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
		DocumentationComplete: false,
	}

	audit := UIAcceptanceLibrary{}.Audit(suite)
	want := UIAcceptanceAudit{
		Name:                      "BIG-1701 v3.0 UI Acceptance",
		Version:                   "v3.0",
		PermissionGaps:            []string{"operations-overview: missing=denied-roles, audit-event"},
		FailingDataChecks:         []string{"sla-dashboard.breach-count: delta=2.0 freshness=901s"},
		FailingPerformanceBudgets: []string{"triage-center.initial-load: p95=1480ms tti=2400ms"},
		FailingUsabilityJourneys:  []string{"reassign-alert: steps=5/3"},
		IncompleteAuditTrails:     []string{"permission.override.used: missing_fields=reason_code retention=30/180d"},
		DocumentationComplete:     false,
		ReadinessScore:            0.0,
	}
	if !reflect.DeepEqual(audit, want) {
		t.Fatalf("acceptance audit = %#v, want %#v", audit, want)
	}
	if audit.ReleaseReady() {
		t.Fatalf("expected release_ready to be false")
	}
}

func TestRenderUIAcceptanceReportSummarizesReleaseReadiness(t *testing.T) {
	suite := passingAcceptanceSuite()
	audit := UIAcceptanceLibrary{}.Audit(suite)
	report := RenderUIAcceptanceReport(suite, audit)
	assertContainsAll(t, report,
		"# UI Acceptance Report",
		"- Readiness Score: 100.0",
		"- Release Ready: True",
		"- Role/Permission run-detail: allow=admin, operator deny=viewer audit_event=ui.access.denied",
		"- Data Accuracy sla-dashboard.breach-count: delta=0.0 tolerance=0.0 freshness=120/300s",
		"- Performance triage-center.initial-load: p95=980/1200ms tti=1400/1800ms",
		"- Usability approve-high-risk-run: steps=3/4 keyboard=True empty_state=True recovery=True",
		"- Audit completeness gaps: none",
	)
}

func passingAcceptanceSuite() UIAcceptanceSuite {
	return UIAcceptanceSuite{
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
			Tolerance:                0.0,
			ObservedDelta:            0.0,
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
}

func roundTripJSON(t *testing.T, input any, output any) {
	t.Helper()
	body, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if err := json.Unmarshal(body, output); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
}

func assertContainsAll(t *testing.T, report string, snippets ...string) {
	t.Helper()
	for _, snippet := range snippets {
		if !strings.Contains(report, snippet) {
			t.Fatalf("report missing %q\nreport:\n%s", snippet, report)
		}
	}
}
