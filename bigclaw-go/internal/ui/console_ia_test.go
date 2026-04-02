package ui

import (
	"reflect"
	"testing"
)

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
	if !reflect.DeepEqual(restored, architecture) {
		t.Fatalf("restored architecture = %#v, want %#v", restored, architecture)
	}
}

func TestConsoleIAAuditSurfacesGlobalInteractionGaps(t *testing.T) {
	architecture := ConsoleIA{
		Name:    "BigClaw Console IA",
		Version: "v3",
		TopBar: ConsoleTopBar{
			Name:                      "Incomplete Header",
			SearchPlaceholder:         "",
			EnvironmentOptions:        []string{"Production"},
			TimeRangeOptions:          []string{"24h"},
			DocumentationComplete:     false,
			AccessibilityRequirements: []string{"focus-visible"},
			CommandEntry:              ConsoleCommandEntry{TriggerLabel: "", Placeholder: "", Shortcut: "Cmd+K"},
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

	audit := ConsoleIAAuditor{}.Audit(architecture)
	if got, want := audit.SurfacesMissingFilters, []string{"Queue"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("surfaces missing filters = %v, want %v", got, want)
	}
	if got, want := audit.SurfacesMissingActions, []string{"Queue"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("surfaces missing actions = %v, want %v", got, want)
	}
	if got, want := audit.TopBarAudit.MissingCapabilities, []string{"global-search", "time-range-switch", "environment-switch", "alert-entry", "command-shell"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("top bar missing capabilities = %v, want %v", got, want)
	}
	if audit.TopBarAudit.ReleaseReady() {
		t.Fatalf("expected top bar to be not release ready")
	}
	if got, want := audit.SurfacesMissingStates, map[string][]string{"Queue": {"error"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("surfaces missing states = %v, want %v", got, want)
	}
	if got, want := audit.StatesMissingActions, map[string][]string{"Queue": {"loading"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("states missing actions = %v, want %v", got, want)
	}
	if got, want := audit.UnresolvedStateActions, map[string]map[string][]string{"Queue": {"empty": {"retry"}}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unresolved state actions = %v, want %v", got, want)
	}
	if got, want := audit.OrphanNavigationRoutes, []string{"/ghost"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("orphan navigation routes = %v, want %v", got, want)
	}
	if got, want := audit.UnnavigableSurfaces, []string{"Queue"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unnavigable surfaces = %v, want %v", got, want)
	}
	if audit.ReadinessScore != 0.0 {
		t.Fatalf("readiness score = %v, want 0.0", audit.ReadinessScore)
	}
}

func TestConsoleIAAuditRoundTripPreservesFindings(t *testing.T) {
	topBarAudit := ConsoleIAAuditor{}.Audit(ConsoleIA{
		Name:    "BigClaw Console IA",
		Version: "v3",
		TopBar: ConsoleTopBar{
			Name:                      "Incomplete Header",
			SearchPlaceholder:         "",
			EnvironmentOptions:        []string{"Production"},
			TimeRangeOptions:          []string{"24h"},
			DocumentationComplete:     false,
			AccessibilityRequirements: []string{"focus-visible"},
			CommandEntry:              ConsoleCommandEntry{TriggerLabel: "", Placeholder: "", Shortcut: "Cmd+K"},
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
		SurfacesMissingStates:  map[string][]string{"Queue": {"error"}},
		StatesMissingActions:   map[string][]string{"Queue": {"loading"}},
		UnresolvedStateActions: map[string]map[string][]string{"Queue": {"empty": {"retry"}}},
		OrphanNavigationRoutes: []string{"/ghost"},
		UnnavigableSurfaces:    []string{"Queue"},
	}

	var restored ConsoleIAAudit
	roundTripJSON(t, audit, &restored)
	if !reflect.DeepEqual(restored, audit) {
		t.Fatalf("restored audit = %#v, want %#v", restored, audit)
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
			States:            []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"refresh"}}, {Name: "empty", AllowedActions: []string{"refresh"}}, {Name: "error", AllowedActions: []string{"refresh"}}},
		}},
	}

	audit := ConsoleIAAuditor{}.Audit(architecture)
	report := RenderConsoleIAReport(architecture, audit)
	assertContainsAll(t, report,
		"# Console Information Architecture Report",
		"- Name: BigClaw Global Header",
		"- Release Ready: True",
		"- Navigation Items: 1",
		"- Overview: route=/overview filters=Team actions=Refresh states=default, loading, empty, error missing_states=none states_without_actions=none unresolved_state_actions=none",
		"- Surfaces missing filters: none",
		"- Undefined state actions: none",
	)
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
	if !reflect.DeepEqual(restored, draft) {
		t.Fatalf("restored interaction draft = %#v, want %#v", restored, draft)
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

	audit := ConsoleInteractionAuditor{}.Audit(draft)
	want := ConsoleInteractionAudit{
		Name:                           "BIG-4203 Four Critical Pages",
		Version:                        "v1",
		ContractCount:                  4,
		MissingSurfaces:                []string{},
		SurfacesMissingFilters:         []string{"Triage"},
		SurfacesMissingActions:         map[string][]string{"Queue": {"export"}},
		SurfacesMissingBatchActions:    []string{"Queue"},
		SurfacesMissingStates:          map[string][]string{"Queue": {"error"}},
		PermissionGaps:                 map[string][]string{"Queue": {"audit-event"}, "Run Detail": {"denied-roles"}},
		UncoveredRoles:                 []string{},
		SurfacesMissingPrimaryPersonas: []string{},
		SurfacesMissingWireframeLinks:  []string{},
		SurfacesMissingReviewFocus:     []string{},
		SurfacesMissingDecisionPrompts: []string{},
		ReadinessScore:                 0.0,
	}
	if !reflect.DeepEqual(audit, want) {
		t.Fatalf("interaction audit = %#v, want %#v", audit, want)
	}
	if audit.ReleaseReady() {
		t.Fatalf("expected interaction audit to be not release ready")
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
				CommandEntry: ConsoleCommandEntry{
					TriggerLabel: "Command Menu",
					Placeholder:  "Type a command",
					Shortcut:     "Cmd+K / Ctrl+K",
					Commands:     []CommandAction{{ID: "search-runs", Title: "Search runs", Section: "Navigate"}},
				},
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

	audit := ConsoleInteractionAuditor{}.Audit(draft)
	report := RenderConsoleInteractionReport(draft, audit)
	assertContainsAll(t, report,
		"# Console Interaction Draft Report",
		"- Critical Pages: 4",
		"- Required Roles: none",
		"- Readiness Score: 100.0",
		"- Release Ready: True",
		"- Overview: route=/overview required_actions=drill-down, export, audit available_actions=drill-down, export, audit filters=1 states=default, loading, empty, error batch=optional permissions=complete",
		"- Queue: route=/queue required_actions=drill-down, export, audit available_actions=drill-down, export, audit, bulk-approve filters=1 states=default, loading, empty, error batch=required permissions=complete",
		"- Permission gaps: none",
	)
}

func TestBuildBig4203ConsoleInteractionDraftIsReleaseReady(t *testing.T) {
	draft := BuildBig4203ConsoleInteractionDraft()
	audit := ConsoleInteractionAuditor{}.Audit(draft)
	report := RenderConsoleInteractionReport(draft, audit)

	if got, want := draft.RequiredRoles, []string{"eng-lead", "platform-admin", "vp-eng", "cross-team-operator"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("required roles = %v, want %v", got, want)
	}
	if !draft.RequiresFrameContracts {
		t.Fatalf("expected draft to require frame contracts")
	}
	if !audit.ReleaseReady() {
		t.Fatalf("expected audit to be release ready")
	}
	if len(audit.UncoveredRoles) != 0 {
		t.Fatalf("uncovered roles = %v, want none", audit.UncoveredRoles)
	}
	assertContainsAll(t, report,
		"- Required Roles: eng-lead, platform-admin, vp-eng, cross-team-operator",
		"persona=VP Eng wireframe=wf-overview",
		"review_focus=metric hierarchy, drill-down posture, alert prioritization",
		"- Uncovered roles: none",
		"- Pages missing personas: none",
		"- Pages missing wireframe links: none",
	)
}

func TestConsoleInteractionAuditFlagsUncoveredRequiredRoles(t *testing.T) {
	draft := BuildBig4203ConsoleInteractionDraft()
	draft.RequiredRoles = append(draft.RequiredRoles, "finance-reviewer")

	audit := ConsoleInteractionAuditor{}.Audit(draft)
	if got, want := audit.UncoveredRoles, []string{"finance-reviewer"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("uncovered roles = %v, want %v", got, want)
	}
	if audit.ReleaseReady() {
		t.Fatalf("expected audit to be not release ready")
	}
}

func TestConsoleInteractionAuditFlagsMissingFrameContractDetails(t *testing.T) {
	draft := BuildBig4203ConsoleInteractionDraft()
	draft.Contracts[0].PrimaryPersona = ""
	draft.Contracts[0].LinkedWireframeID = ""
	draft.Contracts[0].ReviewFocusAreas = nil
	draft.Contracts[0].DecisionPrompts = nil

	audit := ConsoleInteractionAuditor{}.Audit(draft)
	if got, want := audit.SurfacesMissingPrimaryPersonas, []string{"Overview"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("surfaces missing personas = %v, want %v", got, want)
	}
	if got, want := audit.SurfacesMissingWireframeLinks, []string{"Overview"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("surfaces missing wireframe links = %v, want %v", got, want)
	}
	if got, want := audit.SurfacesMissingReviewFocus, []string{"Overview"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("surfaces missing review focus = %v, want %v", got, want)
	}
	if got, want := audit.SurfacesMissingDecisionPrompts, []string{"Overview"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("surfaces missing decision prompts = %v, want %v", got, want)
	}
	if audit.ReleaseReady() {
		t.Fatalf("expected audit to be not release ready")
	}
}
