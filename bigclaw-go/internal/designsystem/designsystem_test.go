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
		Variants:                  []ComponentVariant{{Name: "primary", Tokens: []string{"color.action.primary", "spacing.control.md"}, States: []string{"default", "hover", "disabled"}}},
	}
	if !component.ReleaseReady() || !reflect.DeepEqual(component.TokenNames(), []string{"color.action.primary", "spacing.control.md"}) || len(component.MissingRequiredStates()) != 0 {
		t.Fatalf("unexpected component readiness: %+v", component)
	}
}

func TestDesignSystemRoundTripPreservesManifestShape(t *testing.T) {
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
			Variants:                  []ComponentVariant{{Name: "primary", Tokens: []string{"color.action.primary"}, States: []string{"default", "hover", "disabled"}, UsageNotes: "Use for primary CTA."}},
		}},
	}
	restored, err := roundTrip(system)
	if err != nil {
		t.Fatalf("round trip system: %v", err)
	}
	if !reflect.DeepEqual(restored, system) {
		t.Fatalf("restored system mismatch: %+v", restored)
	}
}

func TestDesignSystemAuditSurfacesReleaseGapsAndOrphanTokens(t *testing.T) {
	system := DesignSystem{
		Name:    "BigClaw Console UI",
		Version: "v2",
		Tokens:  []DesignToken{{Name: "color.action.primary", Category: "color", Value: "#4455ff"}, {Name: "spacing.control.md", Category: "spacing", Value: "12px"}, {Name: "radius.md", Category: "radius", Value: "8px"}},
		Components: []ComponentSpec{
			{Name: "Button", Readiness: "stable", DocumentationComplete: true, AccessibilityRequirements: []string{"focus-visible", "keyboard-activation"}, Variants: []ComponentVariant{{Name: "primary", Tokens: []string{"color.action.primary", "spacing.control.md"}, States: []string{"default", "hover", "disabled"}}}},
			{Name: "CommandBar", Readiness: "beta", DocumentationComplete: false, Variants: []ComponentVariant{{Name: "global", Tokens: []string{"spacing.control.md"}, States: []string{"default", "hover"}}}},
		},
	}
	audit := ComponentLibrary{}.Audit(system)
	if !reflect.DeepEqual(audit.ReleaseReadyComponents, []string{"Button"}) ||
		!reflect.DeepEqual(audit.ComponentsMissingDocs, []string{"CommandBar"}) ||
		!reflect.DeepEqual(audit.ComponentsMissingAccessibility, []string{"CommandBar"}) ||
		!reflect.DeepEqual(audit.ComponentsMissingStates, []string{"CommandBar"}) ||
		!reflect.DeepEqual(audit.UndefinedTokenRefs, map[string][]string{}) ||
		!reflect.DeepEqual(audit.TokenOrphans, []string{"radius.md"}) ||
		audit.ReadinessScore != 35.0 {
		t.Fatalf("unexpected design system audit: %+v", audit)
	}
}

func TestDesignSystemAuditFlagsUndefinedTokenReferences(t *testing.T) {
	system := DesignSystem{
		Name:       "BigClaw Console UI",
		Version:    "v2",
		Tokens:     []DesignToken{{Name: "spacing.control.md", Category: "spacing", Value: "12px"}},
		Components: []ComponentSpec{{Name: "SideNav", Readiness: "stable", DocumentationComplete: true, AccessibilityRequirements: []string{"focus-visible"}, Variants: []ComponentVariant{{Name: "default", Tokens: []string{"spacing.control.md", "color.surface.nav"}, States: []string{"default", "hover", "disabled"}}}}},
	}
	audit := ComponentLibrary{}.Audit(system)
	if !reflect.DeepEqual(audit.ReleaseReadyComponents, []string{}) || !reflect.DeepEqual(audit.UndefinedTokenRefs, map[string][]string{"SideNav": {"color.surface.nav"}}) {
		t.Fatalf("unexpected undefined token audit: %+v", audit)
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
	restored, err := roundTrip(audit)
	if err != nil {
		t.Fatalf("round trip audit: %v", err)
	}
	if !reflect.DeepEqual(restored, audit) {
		t.Fatalf("restored audit mismatch: %+v", restored)
	}
}

func TestRenderDesignSystemReportSummarizesInventoryAndGaps(t *testing.T) {
	system := DesignSystem{
		Name:       "BigClaw Console UI",
		Version:    "v2",
		Tokens:     []DesignToken{{Name: "color.action.primary", Category: "color", Value: "#4455ff"}, {Name: "spacing.control.md", Category: "spacing", Value: "12px"}},
		Components: []ComponentSpec{{Name: "Button", Readiness: "stable", DocumentationComplete: true, AccessibilityRequirements: []string{"focus-visible"}, Variants: []ComponentVariant{{Name: "primary", Tokens: []string{"color.action.primary", "spacing.control.md"}, States: []string{"default", "hover", "disabled"}}}}},
	}
	report := RenderDesignSystemReport(system, ComponentLibrary{}.Audit(system))
	for _, fragment := range []string{"# Design System Report", "- Release Ready Components: 1", "- color: 1", "- Button: readiness=stable docs=true a11y=true states=default, hover, disabled missing_states=none undefined_tokens=none", "- Missing interaction states: none", "- Undefined token refs: none", "- Orphan tokens: none"} {
		if !strings.Contains(strings.ToLower(report), strings.ToLower(fragment)) {
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
		CommandEntry:              ConsoleCommandEntry{TriggerLabel: "Command Menu", Placeholder: "Type a command or jump to a run", Shortcut: "Cmd+K / Ctrl+K", RecentQueriesEnabled: true, Commands: []CommandAction{{ID: "search-runs", Title: "Search runs", Section: "Navigate", Shortcut: "/"}, {ID: "open-alerts", Title: "Open alerts", Section: "Monitor"}}},
	}
	restored, err := roundTrip(topBar)
	if err != nil {
		t.Fatalf("round trip top bar: %v", err)
	}
	if !reflect.DeepEqual(restored, topBar) {
		t.Fatalf("restored top bar mismatch: %+v", restored)
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
		CommandEntry:              ConsoleCommandEntry{TriggerLabel: "Command Menu", Placeholder: "Type a command or jump to a run", Shortcut: "Cmd+K / Ctrl+K", Commands: []CommandAction{{ID: "search-runs", Title: "Search runs", Section: "Navigate"}, {ID: "switch-env", Title: "Switch environment", Section: "Context"}}},
	}
	audit := ConsoleChromeLibrary{}.AuditTopBar(topBar)
	expected := ConsoleTopBarAudit{Name: "BigClaw Global Header", MissingCapabilities: []string{}, DocumentationComplete: true, AccessibilityComplete: true, CommandShortcutSupported: true, CommandCount: 2, ReleaseReady: true}
	if !reflect.DeepEqual(audit, expected) {
		t.Fatalf("unexpected top bar audit: %+v", audit)
	}
}

func TestConsoleTopBarAuditFlagsMissingGlobalEntryCapabilities(t *testing.T) {
	audit := ConsoleChromeLibrary{}.AuditTopBar(ConsoleTopBar{Name: "Incomplete Header", EnvironmentOptions: []string{"Production"}, TimeRangeOptions: []string{"24h"}, CommandEntry: ConsoleCommandEntry{Shortcut: "Cmd+K"}, DocumentationComplete: false, AccessibilityRequirements: []string{"focus-visible"}})
	if !reflect.DeepEqual(audit.MissingCapabilities, []string{"global-search", "time-range-switch", "environment-switch", "alert-entry", "command-shell"}) || audit.DocumentationComplete || audit.AccessibilityComplete || audit.CommandShortcutSupported || audit.ReleaseReady {
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
		CommandEntry:              ConsoleCommandEntry{TriggerLabel: "Command Menu", Placeholder: "Type a command or jump to a run", Shortcut: "Cmd+K / Ctrl+K", Commands: []CommandAction{{ID: "search-runs", Title: "Search runs", Section: "Navigate", Shortcut: "/"}, {ID: "open-alerts", Title: "Open alerts", Section: "Monitor"}}},
	}
	report := RenderConsoleTopBarReport(topBar, ConsoleChromeLibrary{}.AuditTopBar(topBar))
	for _, fragment := range []string{"# Console Top Bar Report", "- Command Shortcut: Cmd+K / Ctrl+K", "- Release Ready: true", "- search-runs: Search runs [Navigate] shortcut=/", "- Missing capabilities: none", "- Cmd/Ctrl+K supported: true"} {
		if !strings.Contains(strings.ToLower(report), strings.ToLower(fragment)) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}

func TestInformationArchitectureRoundTripAndRouteResolution(t *testing.T) {
	architecture := InformationArchitecture{
		GlobalNav: []NavigationNode{{NodeID: "ops", Title: "Operations", Segment: "operations", ScreenID: "operations-overview", Children: []NavigationNode{{NodeID: "ops-queue", Title: "Queue Control", Segment: "queue", ScreenID: "queue-control"}, {NodeID: "ops-triage", Title: "Triage Center", Segment: "triage", ScreenID: "triage-center"}}}},
		Routes:    []NavigationRoute{{Path: "/operations", ScreenID: "operations-overview", Title: "Operations", NavNodeID: "ops"}, {Path: "/operations/queue", ScreenID: "queue-control", Title: "Queue Control", NavNodeID: "ops-queue"}, {Path: "/operations/triage", ScreenID: "triage-center", Title: "Triage Center", NavNodeID: "ops-triage"}},
	}
	restored, err := roundTrip(architecture)
	if err != nil {
		t.Fatalf("round trip information architecture: %v", err)
	}
	if !reflect.DeepEqual(restored, architecture) || !reflect.DeepEqual([]string{architecture.NavigationEntries()[0].Path, architecture.NavigationEntries()[1].Path, architecture.NavigationEntries()[2].Path}, []string{"/operations", "/operations/queue", "/operations/triage"}) || architecture.ResolveRoute("operations/queue") != (NavigationRoute{Path: "/operations/queue", ScreenID: "queue-control", Title: "Queue Control", NavNodeID: "ops-queue"}) {
		t.Fatalf("unexpected information architecture round trip or resolution")
	}
}

func TestInformationArchitectureAuditFlagsDuplicatesSecondaryGapsAndOrphans(t *testing.T) {
	architecture := InformationArchitecture{
		GlobalNav: []NavigationNode{{NodeID: "workbench", Title: "Workbench", Segment: "workbench", ScreenID: "workbench-home", Children: []NavigationNode{{NodeID: "workbench-runs", Title: "Runs", Segment: "runs", ScreenID: "run-index"}, {NodeID: "workbench-replays", Title: "Replays", Segment: "replays", ScreenID: "replay-index"}}}},
		Routes:    []NavigationRoute{{Path: "/workbench/runs", ScreenID: "run-index", Title: "Runs", NavNodeID: "workbench-runs"}, {Path: "/workbench/runs", ScreenID: "run-index-v2", Title: "Runs V2", NavNodeID: "workbench-runs"}, {Path: "/settings", ScreenID: "settings-home", Title: "Settings", NavNodeID: "settings"}},
	}
	audit := architecture.Audit()
	if audit.Healthy || !reflect.DeepEqual(audit.DuplicateRoutes, []string{"/workbench/runs"}) || !reflect.DeepEqual(audit.MissingRouteNodes, map[string]string{"workbench": "/workbench", "workbench-replays": "/workbench/replays"}) || !reflect.DeepEqual(audit.SecondaryNavGaps, map[string][]string{"Workbench": {"/workbench"}}) || !reflect.DeepEqual(audit.OrphanRoutes, []string{"/settings"}) {
		t.Fatalf("unexpected IA audit: %+v", audit)
	}
}

func TestInformationArchitectureAuditRoundTripAndReport(t *testing.T) {
	audit := InformationArchitectureAudit{TotalNavigationNodes: 3, TotalRoutes: 2, DuplicateRoutes: []string{"/workbench/runs"}, MissingRouteNodes: map[string]string{"workbench": "/workbench"}, SecondaryNavGaps: map[string][]string{"Workbench": {"/workbench"}}, OrphanRoutes: []string{"/settings"}}
	restored, err := roundTrip(audit)
	if err != nil {
		t.Fatalf("round trip IA audit: %v", err)
	}
	if !reflect.DeepEqual(restored, audit) {
		t.Fatalf("restored IA audit mismatch: %+v", restored)
	}
	architecture := InformationArchitecture{GlobalNav: []NavigationNode{{NodeID: "workbench", Title: "Workbench", Segment: "workbench", ScreenID: "workbench-home"}}, Routes: []NavigationRoute{{Path: "/settings", ScreenID: "settings-home", Title: "Settings", NavNodeID: "settings"}}}
	report := RenderInformationArchitectureReport(architecture, audit)
	for _, fragment := range []string{"# Information Architecture Report", "- Healthy: false", "- Workbench (/workbench) screen=workbench-home", "- /settings: screen=settings-home title=Settings nav_node=settings", "- Duplicate routes: /workbench/runs", "- Missing route nodes: workbench=/workbench", "- Secondary nav gaps: Workbench=/workbench", "- Orphan routes: /settings"} {
		if !strings.Contains(strings.ToLower(report), strings.ToLower(fragment)) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}

func TestUIAcceptanceSuiteRoundTripPreservesAcceptanceManifest(t *testing.T) {
	suite := UIAcceptanceSuite{
		Name:                  "BIG-1701 v3.0 UI Acceptance",
		Version:               "v3.0",
		RolePermissions:       []RolePermissionScenario{{ScreenID: "run-detail", AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "ui.access.denied"}},
		DataAccuracyChecks:    []DataAccuracyCheck{{ScreenID: "sla-dashboard", MetricID: "breach-count", SourceOfTruth: "warehouse.sla_daily", RenderedValue: "12", Tolerance: 0.0, ObservedDelta: 0.0, FreshnessSLOSeconds: 300, ObservedFreshnessSeconds: 120}},
		PerformanceBudgets:    []PerformanceBudget{{SurfaceID: "triage-center", Interaction: "initial-load", TargetP95MS: 1200, ObservedP95MS: 980, TargetTTIMS: 1800, ObservedTTIMS: 1400}},
		UsabilityJourneys:     []UsabilityJourney{{JourneyID: "approve-high-risk-run", Personas: []string{"operator"}, CriticalSteps: []string{"open queue", "inspect run", "approve"}, ExpectedMaxSteps: 4, ObservedSteps: 3, KeyboardAccessible: true, EmptyStateGuidance: true, RecoverySupport: true}},
		AuditRequirements:     []AuditRequirement{{EventType: "run.approval.changed", RequiredFields: []string{"run_id", "actor_role", "decision"}, EmittedFields: []string{"run_id", "actor_role", "decision"}, RetentionDays: 90, ObservedRetentionDays: 120}},
		DocumentationComplete: true,
	}
	restored, err := roundTrip(suite)
	if err != nil {
		t.Fatalf("round trip UI acceptance suite: %v", err)
	}
	if !reflect.DeepEqual(restored, suite) {
		t.Fatalf("restored UI suite mismatch: %+v", restored)
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
	expected := UIAcceptanceAudit{Name: "BIG-1701 v3.0 UI Acceptance", Version: "v3.0", PermissionGaps: []string{"operations-overview: missing=denied-roles, audit-event"}, FailingDataChecks: []string{"sla-dashboard.breach-count: delta=2.0 freshness=901s"}, FailingPerformanceBudgets: []string{"triage-center.initial-load: p95=1480ms tti=2400ms"}, FailingUsabilityJourneys: []string{"reassign-alert: steps=5/3"}, IncompleteAuditTrails: []string{"permission.override.used: missing_fields=reason_code retention=30/180d"}, DocumentationComplete: false, ReadinessScore: 0.0, ReleaseReady: false}
	if !reflect.DeepEqual(audit, expected) {
		t.Fatalf("unexpected UI acceptance audit: got %+v want %+v", audit, expected)
	}
}

func TestRenderUIAcceptanceReportSummarizesReleaseReadiness(t *testing.T) {
	suite := UIAcceptanceSuite{
		Name:                  "BIG-1701 v3.0 UI Acceptance",
		Version:               "v3.0",
		RolePermissions:       []RolePermissionScenario{{ScreenID: "run-detail", AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "ui.access.denied"}},
		DataAccuracyChecks:    []DataAccuracyCheck{{ScreenID: "sla-dashboard", MetricID: "breach-count", SourceOfTruth: "warehouse.sla_daily", RenderedValue: "12", Tolerance: 0.0, ObservedDelta: 0.0, FreshnessSLOSeconds: 300, ObservedFreshnessSeconds: 120}},
		PerformanceBudgets:    []PerformanceBudget{{SurfaceID: "triage-center", Interaction: "initial-load", TargetP95MS: 1200, ObservedP95MS: 980, TargetTTIMS: 1800, ObservedTTIMS: 1400}},
		UsabilityJourneys:     []UsabilityJourney{{JourneyID: "approve-high-risk-run", Personas: []string{"operator"}, CriticalSteps: []string{"open queue", "inspect run", "approve"}, ExpectedMaxSteps: 4, ObservedSteps: 3, KeyboardAccessible: true, EmptyStateGuidance: true, RecoverySupport: true}},
		AuditRequirements:     []AuditRequirement{{EventType: "run.approval.changed", RequiredFields: []string{"run_id", "actor_role", "decision"}, EmittedFields: []string{"run_id", "actor_role", "decision"}, RetentionDays: 90, ObservedRetentionDays: 120}},
		DocumentationComplete: true,
	}
	report := RenderUIAcceptanceReport(suite, UIAcceptanceLibrary{}.Audit(suite))
	for _, fragment := range []string{"# UI Acceptance Report", "- Readiness Score: 100.0", "- Release Ready: true", "- Role/Permission run-detail: allow=admin, operator deny=viewer audit_event=ui.access.denied", "- Data Accuracy sla-dashboard.breach-count: delta=0.0 tolerance=0.0 freshness=120/300s", "- Performance triage-center.initial-load: p95=980/1200ms tti=1400/1800ms", "- Usability approve-high-risk-run: steps=3/4 keyboard=true empty_state=true recovery=true", "- Audit completeness gaps: none"} {
		if !strings.Contains(strings.ToLower(report), strings.ToLower(fragment)) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}
