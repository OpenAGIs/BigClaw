package designsystem

import (
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
		t.Fatal("expected release ready")
	}
	if got := component.TokenNames(); !reflect.DeepEqual(got, []string{"color.action.primary", "spacing.control.md"}) {
		t.Fatalf("unexpected token names: %+v", got)
	}
	if got := component.MissingRequiredStates(); len(got) != 0 {
		t.Fatalf("unexpected missing states: %+v", got)
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
	restored := MustJSONRoundTrip(system)
	if !reflect.DeepEqual(restored, system) {
		t.Fatalf("round trip mismatch: got=%+v want=%+v", restored, system)
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
				Name:                  "CommandBar",
				Readiness:             "beta",
				DocumentationComplete: false,
				Variants:              []ComponentVariant{{Name: "global", Tokens: []string{"spacing.control.md"}, States: []string{"default", "hover"}}},
			},
		},
	}
	audit := ComponentLibrary{}.Audit(system)
	if !reflect.DeepEqual(audit.ReleaseReadyComponents, []string{"Button"}) {
		t.Fatalf("unexpected release-ready components: %+v", audit.ReleaseReadyComponents)
	}
	if !reflect.DeepEqual(audit.ComponentsMissingDocs, []string{"CommandBar"}) ||
		!reflect.DeepEqual(audit.ComponentsMissingAccessibility, []string{"CommandBar"}) ||
		!reflect.DeepEqual(audit.ComponentsMissingStates, []string{"CommandBar"}) {
		t.Fatalf("unexpected component gaps: %+v", audit)
	}
	if len(audit.UndefinedTokenRefs) != 0 || !reflect.DeepEqual(audit.TokenOrphans, []string{"radius.md"}) {
		t.Fatalf("unexpected token findings: %+v", audit)
	}
	if got := audit.ReadinessScore(); got != 35.0 {
		t.Fatalf("unexpected readiness score: %v", got)
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
	audit := ComponentLibrary{}.Audit(system)
	if len(audit.ReleaseReadyComponents) != 0 {
		t.Fatalf("did not expect release-ready components: %+v", audit.ReleaseReadyComponents)
	}
	if !reflect.DeepEqual(audit.UndefinedTokenRefs, map[string][]string{"SideNav": {"color.surface.nav"}}) {
		t.Fatalf("unexpected undefined token refs: %+v", audit.UndefinedTokenRefs)
	}
}

func TestDesignSystemAuditRoundTripPreservesGovernanceFindings(t *testing.T) {
	audit := DesignSystemAudit{
		SystemName:                     "BigClaw Console UI",
		Version:                        "v2",
		TokenCountsByCategory:          map[string]int{"color": 3, "spacing": 2},
		ComponentCount:                 2,
		ReleaseReadyComponents:         []string{"Button"},
		ComponentsMissingDocs:          []string{"CommandBar"},
		ComponentsMissingAccessibility: []string{"CommandBar"},
		ComponentsMissingStates:        []string{"CommandBar"},
		UndefinedTokenRefs:             map[string][]string{"SideNav": {"color.surface.nav"}},
		TokenOrphans:                   []string{"radius.md"},
	}
	restored := MustJSONRoundTrip(audit)
	if !reflect.DeepEqual(restored, audit) {
		t.Fatalf("round trip mismatch: got=%+v want=%+v", restored, audit)
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
	report := RenderDesignSystemReport(system, ComponentLibrary{}.Audit(system))
	for _, want := range []string{
		"# Design System Report",
		"- Release Ready Components: 1",
		"- color: 1",
		"- Button: readiness=stable docs=true a11y=true states=default, hover, disabled missing_states=none undefined_tokens=none",
		"- Missing interaction states: none",
		"- Undefined token refs: none",
		"- Orphan tokens: none",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in report:\n%s", want, report)
		}
	}
}

func TestConsoleTopBarRoundTripPreservesCommandEntryManifest(t *testing.T) {
	topBar := sampleConsoleTopBar()
	restored := MustJSONRoundTrip(topBar)
	if !reflect.DeepEqual(restored, topBar) {
		t.Fatalf("round trip mismatch: got=%+v want=%+v", restored, topBar)
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
		t.Fatalf("unexpected audit: got=%+v want=%+v", audit, want)
	}
	if !audit.ReleaseReady() {
		t.Fatal("expected release ready")
	}
}

func TestConsoleTopBarAuditFlagsMissingGlobalEntryCapabilities(t *testing.T) {
	topBar := ConsoleTopBar{
		Name:                      "Incomplete Header",
		SearchPlaceholder:         "",
		EnvironmentOptions:        []string{"Production"},
		TimeRangeOptions:          []string{"24h"},
		CommandEntry:              ConsoleCommandEntry{TriggerLabel: "", Placeholder: "", Shortcut: "Cmd+K"},
		DocumentationComplete:     false,
		AccessibilityRequirements: []string{"focus-visible"},
	}
	audit := ConsoleChromeLibrary{}.AuditTopBar(topBar)
	if !reflect.DeepEqual(audit.MissingCapabilities, []string{"global-search", "time-range-switch", "environment-switch", "alert-entry", "command-shell"}) {
		t.Fatalf("unexpected missing capabilities: %+v", audit.MissingCapabilities)
	}
	if audit.DocumentationComplete || audit.AccessibilityComplete || audit.CommandShortcutSupported || audit.ReleaseReady() {
		t.Fatalf("unexpected audit booleans: %+v", audit)
	}
}

func TestRenderConsoleTopBarReportSummarizesGlobalHeaderAndShell(t *testing.T) {
	topBar := sampleConsoleTopBar()
	report := RenderConsoleTopBarReport(topBar, ConsoleChromeLibrary{}.AuditTopBar(topBar))
	for _, want := range []string{
		"# Console Top Bar Report",
		"- Command Shortcut: Cmd+K / Ctrl+K",
		"- Release Ready: true",
		"- search-runs: Search runs [Navigate] shortcut=/",
		"- Missing capabilities: none",
		"- Cmd/Ctrl+K supported: true",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in report:\n%s", want, report)
		}
	}
}

func TestInformationArchitectureRoundTripAndRouteResolution(t *testing.T) {
	architecture := InformationArchitecture{
		GlobalNav: []NavigationNode{{
			NodeID: "ops", Title: "Operations", Segment: "operations", ScreenID: "operations-overview",
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
	restored := MustJSONRoundTrip(architecture)
	if !reflect.DeepEqual(restored, architecture) {
		t.Fatalf("round trip mismatch: got=%+v want=%+v", restored, architecture)
	}
	entries := architecture.NavigationEntries()
	gotPaths := make([]string, 0, len(entries))
	for _, entry := range entries {
		gotPaths = append(gotPaths, entry.Path)
	}
	if !reflect.DeepEqual(gotPaths, []string{"/operations", "/operations/queue", "/operations/triage"}) {
		t.Fatalf("unexpected navigation paths: %+v", gotPaths)
	}
	route, ok := architecture.ResolveRoute("operations/queue")
	if !ok {
		t.Fatal("expected route")
	}
	want := (NavigationRoute{Path: "/operations/queue", ScreenID: "queue-control", Title: "Queue Control", NavNodeID: "ops-queue"}).normalized()
	if !reflect.DeepEqual(route, want) {
		t.Fatalf("unexpected route: got=%+v want=%+v", route, want)
	}
}

func TestInformationArchitectureAuditFlagsDuplicatesSecondaryGapsAndOrphans(t *testing.T) {
	architecture := InformationArchitecture{
		GlobalNav: []NavigationNode{{
			NodeID: "workbench", Title: "Workbench", Segment: "workbench", ScreenID: "workbench-home",
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
		t.Fatal("expected unhealthy audit")
	}
	if !reflect.DeepEqual(audit.DuplicateRoutes, []string{"/workbench/runs"}) {
		t.Fatalf("unexpected duplicate routes: %+v", audit.DuplicateRoutes)
	}
	if !reflect.DeepEqual(audit.MissingRouteNodes, map[string]string{"workbench": "/workbench", "workbench-replays": "/workbench/replays"}) {
		t.Fatalf("unexpected missing route nodes: %+v", audit.MissingRouteNodes)
	}
	if !reflect.DeepEqual(audit.SecondaryNavGaps, map[string][]string{"Workbench": {"/workbench"}}) {
		t.Fatalf("unexpected secondary nav gaps: %+v", audit.SecondaryNavGaps)
	}
	if !reflect.DeepEqual(audit.OrphanRoutes, []string{"/settings"}) {
		t.Fatalf("unexpected orphan routes: %+v", audit.OrphanRoutes)
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
	restored := MustJSONRoundTrip(audit)
	if !reflect.DeepEqual(restored, audit) {
		t.Fatalf("round trip mismatch: got=%+v want=%+v", restored, audit)
	}
	architecture := InformationArchitecture{
		GlobalNav: []NavigationNode{{NodeID: "workbench", Title: "Workbench", Segment: "workbench", ScreenID: "workbench-home"}},
		Routes:    []NavigationRoute{{Path: "/settings", ScreenID: "settings-home", Title: "Settings", NavNodeID: "settings"}},
	}
	report := RenderInformationArchitectureReport(architecture, audit)
	for _, want := range []string{
		"# Information Architecture Report",
		"- Healthy: false",
		"- Workbench (/workbench) screen=workbench-home",
		"- /settings: screen=settings-home title=Settings nav_node=settings",
		"- Duplicate routes: /workbench/runs",
		"- Missing route nodes: workbench=/workbench",
		"- Secondary nav gaps: Workbench=/workbench",
		"- Orphan routes: /settings",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in report:\n%s", want, report)
		}
	}
}

func TestUIAcceptanceSuiteRoundTripPreservesAcceptanceManifest(t *testing.T) {
	suite := sampleUIAcceptanceSuite()
	restored := MustJSONRoundTrip(suite)
	if !reflect.DeepEqual(restored, suite) {
		t.Fatalf("round trip mismatch: got=%+v want=%+v", restored, suite)
	}
}

func TestUIAcceptanceAuditDetectsPermissionAccuracyPerfUsabilityAndAuditGaps(t *testing.T) {
	suite := UIAcceptanceSuite{
		Name:                  "BIG-1701 v3.0 UI Acceptance",
		Version:               "v3.0",
		RolePermissions:       []RolePermissionScenario{{ScreenID: "operations-overview", AllowedRoles: []string{"admin"}, DeniedRoles: []string{}, AuditEvent: ""}},
		DataAccuracyChecks:    []DataAccuracyCheck{{ScreenID: "sla-dashboard", MetricID: "breach-count", SourceOfTruth: "warehouse.sla_daily", RenderedValue: "12", Tolerance: 0.0, ObservedDelta: 2.0, FreshnessSLOSeconds: 300, ObservedFreshnessSeconds: 901}},
		PerformanceBudgets:    []PerformanceBudget{{SurfaceID: "triage-center", Interaction: "initial-load", TargetP95MS: 1200, ObservedP95MS: 1480, TargetTTIMS: 1800, ObservedTTIMS: 2400}},
		UsabilityJourneys:     []UsabilityJourney{{JourneyID: "reassign-alert", Personas: []string{"operator"}, CriticalSteps: []string{"open alert", "assign owner", "save"}, ExpectedMaxSteps: 3, ObservedSteps: 5, KeyboardAccessible: false, EmptyStateGuidance: true, RecoverySupport: false}},
		AuditRequirements:     []AuditRequirement{{EventType: "permission.override.used", RequiredFields: []string{"actor_role", "screen_id", "reason_code"}, EmittedFields: []string{"actor_role", "screen_id"}, RetentionDays: 180, ObservedRetentionDays: 30}},
		DocumentationComplete: false,
	}
	audit := UIAcceptanceLibrary{}.Audit(suite)
	want := UIAcceptanceAudit{
		Name:                      "BIG-1701 v3.0 UI Acceptance",
		Version:                   "v3.0",
		PermissionGaps:            []string{"operations-overview: missing=denied-roles, audit-event"},
		FailingDataChecks:         []string{"sla-dashboard.breach-count: delta=2 freshness=901s"},
		FailingPerformanceBudgets: []string{"triage-center.initial-load: p95=1480ms tti=2400ms"},
		FailingUsabilityJourneys:  []string{"reassign-alert: steps=5/3"},
		IncompleteAuditTrails:     []string{"permission.override.used: missing_fields=reason_code retention=30/180d"},
		DocumentationComplete:     false,
	}
	if !reflect.DeepEqual(audit, want) {
		t.Fatalf("unexpected audit: got=%+v want=%+v", audit, want)
	}
	if got := audit.ReadinessScore(); got != 0.0 || audit.ReleaseReady() {
		t.Fatalf("unexpected readiness: score=%v ready=%t", got, audit.ReleaseReady())
	}
}

func TestRenderUIAcceptanceReportSummarizesReleaseReadiness(t *testing.T) {
	suite := sampleUIAcceptanceSuite()
	report := RenderUIAcceptanceReport(suite, UIAcceptanceLibrary{}.Audit(suite))
	for _, want := range []string{
		"# UI Acceptance Report",
		"- Readiness Score: 100.0",
		"- Release Ready: true",
		"- Role/Permission run-detail: allow=admin, operator deny=viewer audit_event=ui.access.denied",
		"- Data Accuracy sla-dashboard.breach-count: delta=0 tolerance=0 freshness=120/300s",
		"- Performance triage-center.initial-load: p95=980/1200ms tti=1400/1800ms",
		"- Usability approve-high-risk-run: steps=3/4 keyboard=true empty_state=true recovery=true",
		"- Audit completeness gaps: none",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in report:\n%s", want, report)
		}
	}
}

func sampleConsoleTopBar() ConsoleTopBar {
	return ConsoleTopBar{
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
}

func sampleUIAcceptanceSuite() UIAcceptanceSuite {
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
