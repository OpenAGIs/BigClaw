package designsystemparity

import (
	"reflect"
	"strings"
	"testing"
)

func TestComponentReleaseReadyRequiresDocsAccessibilityAndStates(t *testing.T) {
	t.Parallel()

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
		t.Fatal("expected component to be release ready")
	}
	if !reflect.DeepEqual(component.TokenNames(), []string{"color.action.primary", "spacing.control.md"}) {
		t.Fatalf("unexpected token names: %v", component.TokenNames())
	}
	if len(component.MissingRequiredStates()) != 0 {
		t.Fatalf("unexpected missing states: %v", component.MissingRequiredStates())
	}
}

func TestDesignSystemRoundTripPreservesManifestShape(t *testing.T) {
	t.Parallel()

	system := DesignSystem{
		Name:    "BigClaw Console UI",
		Version: "v2",
		Tokens:  []DesignToken{{Name: "color.action.primary", Category: "color", Value: "#4455ff", SemanticRole: "action-primary"}},
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

	data, err := system.ToMap()
	if err != nil {
		t.Fatalf("to map: %v", err)
	}
	restored, err := DesignSystemFromMap(data)
	if err != nil {
		t.Fatalf("from map: %v", err)
	}
	if !reflect.DeepEqual(restored, system) {
		t.Fatalf("round trip mismatch")
	}
}

func TestDesignSystemAuditSurfacesReleaseGapsAndOrphanTokens(t *testing.T) {
	t.Parallel()

	system := DesignSystem{
		Name:    "BigClaw Console UI",
		Version: "v2",
		Tokens: []DesignToken{
			{Name: "color.action.primary", Category: "color", Value: "#4455ff"},
			{Name: "spacing.control.md", Category: "spacing", Value: "12px"},
			{Name: "radius.md", Category: "radius", Value: "8px"},
		},
		Components: []ComponentSpec{
			{Name: "Button", Readiness: "stable", DocumentationComplete: true, AccessibilityRequirements: []string{"focus-visible", "keyboard-activation"}, Variants: []ComponentVariant{{Name: "primary", Tokens: []string{"color.action.primary", "spacing.control.md"}, States: []string{"default", "hover", "disabled"}}}},
			{Name: "CommandBar", Readiness: "beta", DocumentationComplete: false, Variants: []ComponentVariant{{Name: "global", Tokens: []string{"spacing.control.md"}, States: []string{"default", "hover"}}}},
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
	t.Parallel()

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
	if len(audit.ReleaseReadyComponents) != 0 || !reflect.DeepEqual(audit.UndefinedTokenRefs, map[string][]string{"SideNav": {"color.surface.nav"}}) {
		t.Fatalf("unexpected audit: %+v", audit)
	}
}

func TestDesignSystemAuditRoundTripPreservesGovernanceFindings(t *testing.T) {
	t.Parallel()

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

	data, err := audit.ToMap()
	if err != nil {
		t.Fatalf("to map: %v", err)
	}
	restored, err := DesignSystemAuditFromMap(data)
	if err != nil {
		t.Fatalf("from map: %v", err)
	}
	if !reflect.DeepEqual(restored, audit) {
		t.Fatalf("round trip mismatch")
	}
}

func TestRenderDesignSystemReportSummarizesInventoryAndGaps(t *testing.T) {
	t.Parallel()

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
			t.Fatalf("expected %q in report", fragment)
		}
	}
}

func TestConsoleTopBarRoundTripPreservesCommandEntryManifest(t *testing.T) {
	t.Parallel()

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

	data, err := topBar.ToMap()
	if err != nil {
		t.Fatalf("to map: %v", err)
	}
	restored, err := ConsoleTopBarFromMap(data)
	if err != nil {
		t.Fatalf("from map: %v", err)
	}
	if !reflect.DeepEqual(restored, topBar) {
		t.Fatalf("round trip mismatch")
	}
}

func TestConsoleTopBarAuditChecksTicketCapabilitiesAndShortcuts(t *testing.T) {
	t.Parallel()

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
	want := ConsoleTopBarAudit{
		Name:                     "BigClaw Global Header",
		MissingCapabilities:      []string{},
		DocumentationComplete:    true,
		AccessibilityComplete:    true,
		CommandShortcutSupported: true,
		CommandCount:             2,
	}
	if !reflect.DeepEqual(audit, want) || !audit.ReleaseReady() {
		t.Fatalf("unexpected audit: %+v", audit)
	}
}

func TestConsoleTopBarAuditFlagsMissingGlobalEntryCapabilities(t *testing.T) {
	t.Parallel()

	topBar := ConsoleTopBar{
		Name:                      "Incomplete Header",
		SearchPlaceholder:         "",
		EnvironmentOptions:        []string{"Production"},
		TimeRangeOptions:          []string{"24h"},
		CommandEntry:              ConsoleCommandEntry{TriggerLabel: "", Placeholder: "", Shortcut: "Cmd+K"},
		DocumentationComplete:     false,
		AccessibilityRequirements: []string{"focus-visible"},
	}

	audit := (ConsoleChromeLibrary{}).AuditTopBar(topBar)
	if !reflect.DeepEqual(audit.MissingCapabilities, []string{"global-search", "time-range-switch", "environment-switch", "alert-entry", "command-shell"}) ||
		audit.DocumentationComplete || audit.AccessibilityComplete || audit.CommandShortcutSupported || audit.ReleaseReady() {
		t.Fatalf("unexpected audit: %+v", audit)
	}
}

func TestRenderConsoleTopBarReportSummarizesGlobalHeaderAndShell(t *testing.T) {
	t.Parallel()

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
			t.Fatalf("expected %q in report", fragment)
		}
	}
}

func TestInformationArchitectureRoundTripAndRouteResolution(t *testing.T) {
	t.Parallel()

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

	data, err := architecture.ToMap()
	if err != nil {
		t.Fatalf("to map: %v", err)
	}
	restored, err := InformationArchitectureFromMap(data)
	if err != nil {
		t.Fatalf("from map: %v", err)
	}
	if !reflect.DeepEqual(restored, architecture) {
		t.Fatalf("round trip mismatch")
	}
	entries := architecture.NavigationEntries()
	got := []string{entries[0].Path, entries[1].Path, entries[2].Path}
	if !reflect.DeepEqual(got, []string{"/operations", "/operations/queue", "/operations/triage"}) {
		t.Fatalf("unexpected entries: %v", got)
	}
	route := architecture.ResolveRoute("operations/queue")
	if route == nil || !reflect.DeepEqual(*route, NavigationRoute{Path: "/operations/queue", ScreenID: "queue-control", Title: "Queue Control", NavNodeID: "ops-queue"}) {
		t.Fatalf("unexpected route: %+v", route)
	}
}

func TestInformationArchitectureAuditFlagsDuplicatesSecondaryGapsAndOrphans(t *testing.T) {
	t.Parallel()

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
		!reflect.DeepEqual(audit.SecondaryNavGaps, map[string][]string{"Workbench": {"/workbench"}}) ||
		!reflect.DeepEqual(audit.OrphanRoutes, []string{"/settings"}) {
		t.Fatalf("unexpected audit: %+v", audit)
	}
}

func TestInformationArchitectureAuditRoundTripAndReport(t *testing.T) {
	t.Parallel()

	audit := InformationArchitectureAudit{
		TotalNavigationNodes: 3,
		TotalRoutes:          2,
		DuplicateRoutes:      []string{"/workbench/runs"},
		MissingRouteNodes:    map[string]string{"workbench": "/workbench"},
		SecondaryNavGaps:     map[string][]string{"Workbench": {"/workbench"}},
		OrphanRoutes:         []string{"/settings"},
	}
	data, err := audit.ToMap()
	if err != nil {
		t.Fatalf("to map: %v", err)
	}
	restored, err := InformationArchitectureAuditFromMap(data)
	if err != nil {
		t.Fatalf("from map: %v", err)
	}
	if !reflect.DeepEqual(restored, audit) {
		t.Fatalf("round trip mismatch")
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
			t.Fatalf("expected %q in report", fragment)
		}
	}
}

func TestUIAcceptanceSuiteRoundTripPreservesAcceptanceManifest(t *testing.T) {
	t.Parallel()

	suite := UIAcceptanceSuite{
		Name:                  "BIG-1701 v3.0 UI Acceptance",
		Version:               "v3.0",
		RolePermissions:       []RolePermissionScenario{{ScreenID: "run-detail", AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "ui.access.denied"}},
		DataAccuracyChecks:    []DataAccuracyCheck{{ScreenID: "sla-dashboard", MetricID: "breach-count", SourceOfTruth: "warehouse.sla_daily", RenderedValue: "12", Tolerance: 0, ObservedDelta: 0, FreshnessSLOSeconds: 300, ObservedFreshnessSeconds: 120}},
		PerformanceBudgets:    []PerformanceBudget{{SurfaceID: "triage-center", Interaction: "initial-load", TargetP95MS: 1200, ObservedP95MS: 980, TargetTTIMS: 1800, ObservedTTIMS: 1400}},
		UsabilityJourneys:     []UsabilityJourney{{JourneyID: "approve-high-risk-run", Personas: []string{"operator"}, CriticalSteps: []string{"open queue", "inspect run", "approve"}, ExpectedMaxSteps: 4, ObservedSteps: 3, KeyboardAccessible: true, EmptyStateGuidance: true, RecoverySupport: true}},
		AuditRequirements:     []AuditRequirement{{EventType: "run.approval.changed", RequiredFields: []string{"run_id", "actor_role", "decision"}, EmittedFields: []string{"run_id", "actor_role", "decision"}, RetentionDays: 90, ObservedRetentionDays: 120}},
		DocumentationComplete: true,
	}

	data, err := suite.ToMap()
	if err != nil {
		t.Fatalf("to map: %v", err)
	}
	restored, err := UIAcceptanceSuiteFromMap(data)
	if err != nil {
		t.Fatalf("from map: %v", err)
	}
	if !reflect.DeepEqual(restored, suite) {
		t.Fatalf("round trip mismatch")
	}
}

func TestUIAcceptanceAuditDetectsPermissionAccuracyPerfUsabilityAndAuditGaps(t *testing.T) {
	t.Parallel()

	suite := UIAcceptanceSuite{
		Name:                  "BIG-1701 v3.0 UI Acceptance",
		Version:               "v3.0",
		RolePermissions:       []RolePermissionScenario{{ScreenID: "operations-overview", AllowedRoles: []string{"admin"}, DeniedRoles: []string{}, AuditEvent: ""}},
		DataAccuracyChecks:    []DataAccuracyCheck{{ScreenID: "sla-dashboard", MetricID: "breach-count", SourceOfTruth: "warehouse.sla_daily", RenderedValue: "12", Tolerance: 0, ObservedDelta: 2, FreshnessSLOSeconds: 300, ObservedFreshnessSeconds: 901}},
		PerformanceBudgets:    []PerformanceBudget{{SurfaceID: "triage-center", Interaction: "initial-load", TargetP95MS: 1200, ObservedP95MS: 1480, TargetTTIMS: 1800, ObservedTTIMS: 2400}},
		UsabilityJourneys:     []UsabilityJourney{{JourneyID: "reassign-alert", Personas: []string{"operator"}, CriticalSteps: []string{"open alert", "assign owner", "save"}, ExpectedMaxSteps: 3, ObservedSteps: 5, KeyboardAccessible: false, EmptyStateGuidance: true, RecoverySupport: false}},
		AuditRequirements:     []AuditRequirement{{EventType: "permission.override.used", RequiredFields: []string{"actor_role", "screen_id", "reason_code"}, EmittedFields: []string{"actor_role", "screen_id"}, RetentionDays: 180, ObservedRetentionDays: 30}},
		DocumentationComplete: false,
	}
	audit := (UIAcceptanceLibrary{}).Audit(suite)
	want := UIAcceptanceAudit{
		Name:                      "BIG-1701 v3.0 UI Acceptance",
		Version:                   "v3.0",
		PermissionGaps:            []string{"operations-overview: missing=denied-roles, audit-event"},
		FailingDataChecks:         []string{"sla-dashboard.breach-count: delta=2.0 freshness=901s"},
		FailingPerformanceBudgets: []string{"triage-center.initial-load: p95=1480ms tti=2400ms"},
		FailingUsabilityJourneys:  []string{"reassign-alert: steps=5/3"},
		IncompleteAuditTrails:     []string{"permission.override.used: missing_fields=reason_code retention=30/180d"},
		DocumentationComplete:     false,
	}
	if !reflect.DeepEqual(audit, want) || audit.ReadinessScore() != 0 || audit.ReleaseReady() {
		t.Fatalf("unexpected audit: %+v", audit)
	}
}

func TestRenderUIAcceptanceReportSummarizesReleaseReadiness(t *testing.T) {
	t.Parallel()

	suite := UIAcceptanceSuite{
		Name:                  "BIG-1701 v3.0 UI Acceptance",
		Version:               "v3.0",
		RolePermissions:       []RolePermissionScenario{{ScreenID: "run-detail", AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "ui.access.denied"}},
		DataAccuracyChecks:    []DataAccuracyCheck{{ScreenID: "sla-dashboard", MetricID: "breach-count", SourceOfTruth: "warehouse.sla_daily", RenderedValue: "12", Tolerance: 0, ObservedDelta: 0, FreshnessSLOSeconds: 300, ObservedFreshnessSeconds: 120}},
		PerformanceBudgets:    []PerformanceBudget{{SurfaceID: "triage-center", Interaction: "initial-load", TargetP95MS: 1200, ObservedP95MS: 980, TargetTTIMS: 1800, ObservedTTIMS: 1400}},
		UsabilityJourneys:     []UsabilityJourney{{JourneyID: "approve-high-risk-run", Personas: []string{"operator"}, CriticalSteps: []string{"open queue", "inspect run", "approve"}, ExpectedMaxSteps: 4, ObservedSteps: 3, KeyboardAccessible: true, EmptyStateGuidance: true, RecoverySupport: true}},
		AuditRequirements:     []AuditRequirement{{EventType: "run.approval.changed", RequiredFields: []string{"run_id", "actor_role", "decision"}, EmittedFields: []string{"run_id", "actor_role", "decision"}, RetentionDays: 90, ObservedRetentionDays: 120}},
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
			t.Fatalf("expected %q in report", fragment)
		}
	}
}
