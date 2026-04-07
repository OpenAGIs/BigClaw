package designsystem

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestComponentReleaseReadyRequiresDocsAccessibilityAndStates(t *testing.T) {
	component := ComponentSpec{
		Name:                  "Button",
		Readiness:             "stable",
		DocumentationComplete: true,
		AccessibilityRequirements: []string{
			"focus-visible",
			"keyboard-activation",
		},
		Variants: []ComponentVariant{{
			Name:   "primary",
			Tokens: []string{"color.action.primary", "spacing.control.md"},
			States: []string{"default", "hover", "disabled"},
		}},
	}

	if !component.ReleaseReady() {
		t.Fatalf("expected component to be release ready")
	}
	if got, want := component.TokenNames(), []string{"color.action.primary", "spacing.control.md"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("token names mismatch: got %+v want %+v", got, want)
	}
	if got := component.MissingRequiredStates(); len(got) != 0 {
		t.Fatalf("expected no missing states, got %+v", got)
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
			Theme:        "core",
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

	payload, err := json.Marshal(system)
	if err != nil {
		t.Fatalf("marshal system: %v", err)
	}
	var restored DesignSystem
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal system: %v", err)
	}
	if !reflect.DeepEqual(restored, system) {
		t.Fatalf("restored system mismatch: got %+v want %+v", restored, system)
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
				Name:                  "Button",
				Readiness:             "stable",
				DocumentationComplete: true,
				AccessibilityRequirements: []string{
					"focus-visible",
					"keyboard-activation",
				},
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
		t.Fatalf("release ready mismatch: got %+v want %+v", got, want)
	}
	if got, want := audit.ComponentsMissingDocs, []string{"CommandBar"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("missing docs mismatch: got %+v want %+v", got, want)
	}
	if got, want := audit.ComponentsMissingAccessibility, []string{"CommandBar"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("missing accessibility mismatch: got %+v want %+v", got, want)
	}
	if got, want := audit.ComponentsMissingStates, []string{"CommandBar"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("missing states mismatch: got %+v want %+v", got, want)
	}
	if len(audit.UndefinedTokenRefs) != 0 {
		t.Fatalf("expected no undefined token refs, got %+v", audit.UndefinedTokenRefs)
	}
	if got, want := audit.TokenOrphans, []string{"radius.md"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("token orphans mismatch: got %+v want %+v", got, want)
	}
	if got := audit.ReadinessScore(); got != 35.0 {
		t.Fatalf("readiness score mismatch: got %v want 35.0", got)
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
		t.Fatalf("expected no release ready components, got %+v", audit.ReleaseReadyComponents)
	}
	if got, want := audit.UndefinedTokenRefs, map[string][]string{"SideNav": {"color.surface.nav"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("undefined token refs mismatch: got %+v want %+v", got, want)
	}
}

func TestDesignSystemAuditRoundTripPreservesGovernanceFindings(t *testing.T) {
	audit := DesignSystemAudit{
		SystemName:                     "BigClaw Console UI",
		Version:                        "v2",
		TokenCountsMap:                 map[string]int{"color": 3, "spacing": 2},
		ComponentCount:                 2,
		ReleaseReadyComponents:         []string{"Button"},
		ComponentsMissingDocs:          []string{"CommandBar"},
		ComponentsMissingAccessibility: []string{"CommandBar"},
		ComponentsMissingStates:        []string{"CommandBar"},
		UndefinedTokenRefs:             map[string][]string{"SideNav": {"color.surface.nav"}},
		TokenOrphans:                   []string{"radius.md"},
	}

	payload, err := json.Marshal(audit)
	if err != nil {
		t.Fatalf("marshal audit: %v", err)
	}
	var restored DesignSystemAudit
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal audit: %v", err)
	}
	if !reflect.DeepEqual(restored, audit) {
		t.Fatalf("restored audit mismatch: got %+v want %+v", restored, audit)
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

	for _, needle := range []string{
		"# Design System Report",
		"- Release Ready Components: 1",
		"- color: 1",
		"- Button: readiness=stable docs=true a11y=true states=default, hover, disabled missing_states=none undefined_tokens=none",
		"- Missing interaction states: none",
		"- Undefined token refs: none",
		"- Orphan tokens: none",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("expected report to contain %q, got:\n%s", needle, report)
		}
	}
}

func TestConsoleTopBarRoundTripPreservesCommandEntryManifest(t *testing.T) {
	topBar := ConsoleTopBar{
		Name:              "BigClaw Global Header",
		SearchPlaceholder: "Search runs, issues, commands",
		EnvironmentOptions: []string{
			"Production",
			"Staging",
		},
		TimeRangeOptions:      []string{"24h", "7d", "30d"},
		AlertChannels:         []string{"approvals", "sla"},
		DocumentationComplete: true,
		AccessibilityRequirements: []string{
			"keyboard-navigation",
			"screen-reader-label",
			"focus-visible",
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

	payload, err := json.Marshal(topBar)
	if err != nil {
		t.Fatalf("marshal top bar: %v", err)
	}
	var restored ConsoleTopBar
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal top bar: %v", err)
	}
	if !reflect.DeepEqual(restored, topBar) {
		t.Fatalf("restored top bar mismatch: got %+v want %+v", restored, topBar)
	}
}

func TestConsoleTopBarAuditChecksTicketCapabilitiesAndShortcuts(t *testing.T) {
	topBar := ConsoleTopBar{
		Name:              "BigClaw Global Header",
		SearchPlaceholder: "Search runs, issues, commands",
		EnvironmentOptions: []string{
			"Production",
			"Staging",
		},
		TimeRangeOptions:      []string{"24h", "7d", "30d"},
		AlertChannels:         []string{"approvals", "sla"},
		DocumentationComplete: true,
		AccessibilityRequirements: []string{
			"keyboard-navigation",
			"screen-reader-label",
			"focus-visible",
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
		t.Fatalf("top bar audit mismatch: got %+v want %+v", audit, want)
	}
	if !audit.ReleaseReady() {
		t.Fatalf("expected top bar audit to be release ready")
	}
}

func TestConsoleTopBarAuditFlagsMissingGlobalEntryCapabilities(t *testing.T) {
	topBar := ConsoleTopBar{
		Name:              "Incomplete Header",
		SearchPlaceholder: "",
		EnvironmentOptions: []string{
			"Production",
		},
		TimeRangeOptions:          []string{"24h"},
		DocumentationComplete:     false,
		AccessibilityRequirements: []string{"focus-visible"},
		CommandEntry: ConsoleCommandEntry{
			TriggerLabel: "",
			Placeholder:  "",
			Shortcut:     "Cmd+K",
		},
	}

	audit := ConsoleChromeLibrary{}.AuditTopBar(topBar)

	if got, want := audit.MissingCapabilities, []string{"global-search", "time-range-switch", "environment-switch", "alert-entry", "command-shell"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("missing capabilities mismatch: got %+v want %+v", got, want)
	}
	if audit.DocumentationComplete {
		t.Fatalf("expected documentation to be incomplete")
	}
	if audit.AccessibilityComplete {
		t.Fatalf("expected accessibility to be incomplete")
	}
	if audit.CommandShortcutSupported {
		t.Fatalf("expected shortcut support to be incomplete")
	}
	if audit.ReleaseReady() {
		t.Fatalf("expected audit to be non-release-ready")
	}
}

func TestRenderConsoleTopBarReportSummarizesGlobalHeaderAndShell(t *testing.T) {
	topBar := ConsoleTopBar{
		Name:              "BigClaw Global Header",
		SearchPlaceholder: "Search runs, issues, commands",
		EnvironmentOptions: []string{
			"Production",
			"Staging",
		},
		TimeRangeOptions:      []string{"24h", "7d", "30d"},
		AlertChannels:         []string{"approvals", "sla"},
		DocumentationComplete: true,
		AccessibilityRequirements: []string{
			"keyboard-navigation",
			"screen-reader-label",
			"focus-visible",
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

	for _, needle := range []string{
		"# Console Top Bar Report",
		"- Command Shortcut: Cmd+K / Ctrl+K",
		"- Release Ready: True",
		"- search-runs: Search runs [Navigate] shortcut=/",
		"- Missing capabilities: none",
		"- Cmd/Ctrl+K supported: True",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("expected report to contain %q, got:\n%s", needle, report)
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

	payload, err := json.Marshal(architecture)
	if err != nil {
		t.Fatalf("marshal architecture: %v", err)
	}
	var restored InformationArchitecture
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal architecture: %v", err)
	}
	if !reflect.DeepEqual(restored, architecture) {
		t.Fatalf("restored architecture mismatch: got %+v want %+v", restored, architecture)
	}
	if got, want := collectPaths(architecture.NavigationEntries()), []string{"/operations", "/operations/queue", "/operations/triage"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("navigation entries mismatch: got %+v want %+v", got, want)
	}
	resolved := architecture.ResolveRoute("operations/queue")
	want := &NavigationRoute{Path: "/operations/queue", ScreenID: "queue-control", Title: "Queue Control", NavNodeID: "ops-queue", Layout: "workspace"}
	if !reflect.DeepEqual(resolved, want) {
		t.Fatalf("resolved route mismatch: got %+v want %+v", resolved, want)
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
		t.Fatalf("expected unhealthy architecture audit")
	}
	if got, want := audit.DuplicateRoutes, []string{"/workbench/runs"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("duplicate routes mismatch: got %+v want %+v", got, want)
	}
	if got, want := audit.MissingRouteNodes, map[string]string{"workbench": "/workbench", "workbench-replays": "/workbench/replays"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("missing route nodes mismatch: got %+v want %+v", got, want)
	}
	if got, want := audit.SecondaryNavGaps, map[string][]string{"Workbench": {"/workbench"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("secondary nav gaps mismatch: got %+v want %+v", got, want)
	}
	if got, want := audit.OrphanRoutes, []string{"/settings"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("orphan routes mismatch: got %+v want %+v", got, want)
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

	payload, err := json.Marshal(audit)
	if err != nil {
		t.Fatalf("marshal audit: %v", err)
	}
	var restored InformationArchitectureAudit
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal audit: %v", err)
	}
	if !reflect.DeepEqual(restored, audit) {
		t.Fatalf("restored audit mismatch: got %+v want %+v", restored, audit)
	}

	architecture := InformationArchitecture{
		GlobalNav: []NavigationNode{{NodeID: "workbench", Title: "Workbench", Segment: "workbench", ScreenID: "workbench-home"}},
		Routes:    []NavigationRoute{{Path: "/settings", ScreenID: "settings-home", Title: "Settings", NavNodeID: "settings"}},
	}

	report := RenderInformationArchitectureReport(architecture, audit)

	for _, needle := range []string{
		"# Information Architecture Report",
		"- Healthy: False",
		"- Workbench (/workbench) screen=workbench-home",
		"- /settings: screen=settings-home title=Settings nav_node=settings",
		"- Duplicate routes: /workbench/runs",
		"- Missing route nodes: workbench=/workbench",
		"- Secondary nav gaps: Workbench=/workbench",
		"- Orphan routes: /settings",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("expected report to contain %q, got:\n%s", needle, report)
		}
	}
}

func TestUIAcceptanceSuiteRoundTripPreservesAcceptanceManifest(t *testing.T) {
	suite := fullyReadyUIAcceptanceSuite()

	payload, err := json.Marshal(suite)
	if err != nil {
		t.Fatalf("marshal suite: %v", err)
	}
	var restored UIAcceptanceSuite
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal suite: %v", err)
	}
	if !reflect.DeepEqual(restored, suite) {
		t.Fatalf("restored suite mismatch: got %+v want %+v", restored, suite)
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
		FailingDataChecks:         []string{"sla-dashboard.breach-count: delta=2 freshness=901s"},
		FailingPerformanceBudgets: []string{"triage-center.initial-load: p95=1480ms tti=2400ms"},
		FailingUsabilityJourneys:  []string{"reassign-alert: steps=5/3"},
		IncompleteAuditTrails:     []string{"permission.override.used: missing_fields=reason_code retention=30/180d"},
		DocumentationComplete:     false,
	}
	if !reflect.DeepEqual(audit, want) {
		t.Fatalf("ui acceptance audit mismatch: got %+v want %+v", audit, want)
	}
	if got := audit.ReadinessScore(); got != 0.0 {
		t.Fatalf("readiness score mismatch: got %v want 0.0", got)
	}
	if audit.ReleaseReady() {
		t.Fatalf("expected audit to be non-release-ready")
	}
}

func TestRenderUIAcceptanceReportSummarizesReleaseReadiness(t *testing.T) {
	suite := fullyReadyUIAcceptanceSuite()
	audit := UIAcceptanceLibrary{}.Audit(suite)

	report := RenderUIAcceptanceReport(suite, audit)

	for _, needle := range []string{
		"# UI Acceptance Report",
		"- Readiness Score: 100.0",
		"- Release Ready: True",
		"- Role/Permission run-detail: allow=admin, operator deny=viewer audit_event=ui.access.denied",
		"- Data Accuracy sla-dashboard.breach-count: delta=0 tolerance=0 freshness=120/300s",
		"- Performance triage-center.initial-load: p95=980/1200ms tti=1400/1800ms",
		"- Usability approve-high-risk-run: steps=3/4 keyboard=True empty_state=True recovery=True",
		"- Audit completeness gaps: none",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("expected report to contain %q, got:\n%s", needle, report)
		}
	}
}

func fullyReadyUIAcceptanceSuite() UIAcceptanceSuite {
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

func collectPaths(entries []NavigationEntry) []string {
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		paths = append(paths, entry.Path)
	}
	return paths
}
