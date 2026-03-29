package consoleiaparity

import (
	"reflect"
	"strings"
	"testing"
)

func TestConsoleIARoundTripPreservesManifestShape(t *testing.T) {
	t.Parallel()

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

	data, err := architecture.ToMap()
	if err != nil {
		t.Fatalf("to map: %v", err)
	}
	restored, err := ConsoleIAFromMap(data)
	if err != nil {
		t.Fatalf("from map: %v", err)
	}
	if !reflect.DeepEqual(restored, architecture) {
		t.Fatalf("restored architecture mismatch")
	}
}

func TestConsoleIAAuditSurfacesGlobalInteractionGaps(t *testing.T) {
	t.Parallel()

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
				States: []SurfaceState{
					{Name: "default"},
					{Name: "loading", AllowedActions: []string{"refresh"}},
					{Name: "empty", AllowedActions: []string{"refresh"}},
					{Name: "error", AllowedActions: []string{"refresh"}},
				},
			},
			{
				Name:              "Queue",
				Route:             "/queue",
				NavigationSection: "Operate",
				States: []SurfaceState{
					{Name: "default"},
					{Name: "loading"},
					{Name: "empty", AllowedActions: []string{"retry"}},
				},
			},
		},
	}

	audit := (ConsoleIAAuditor{}).Audit(architecture)
	if !reflect.DeepEqual(audit.SurfacesMissingFilters, []string{"Queue"}) ||
		!reflect.DeepEqual(audit.SurfacesMissingActions, []string{"Queue"}) ||
		!reflect.DeepEqual(audit.TopBarAudit.MissingCapabilities, []string{"global-search", "time-range-switch", "environment-switch", "alert-entry", "command-shell"}) ||
		audit.TopBarAudit.ReleaseReady() ||
		!reflect.DeepEqual(audit.SurfacesMissingStates, map[string][]string{"Queue": {"error"}}) ||
		!reflect.DeepEqual(audit.StatesMissingActions, map[string][]string{"Queue": {"loading"}}) ||
		!reflect.DeepEqual(audit.UnresolvedStateActions, map[string]map[string][]string{"Queue": {"empty": {"retry"}}}) ||
		!reflect.DeepEqual(audit.OrphanNavigationRoutes, []string{"/ghost"}) ||
		!reflect.DeepEqual(audit.UnnavigableSurfaces, []string{"Queue"}) ||
		audit.ReadinessScore() != 0 {
		t.Fatalf("unexpected audit: %+v", audit)
	}
}

func TestConsoleIAAuditRoundTripPreservesFindings(t *testing.T) {
	t.Parallel()

	topBarAudit := (ConsoleIAAuditor{}).Audit(ConsoleIA{
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

	data, err := audit.ToMap()
	if err != nil {
		t.Fatalf("to map: %v", err)
	}
	restored, err := ConsoleIAAuditFromMap(data)
	if err != nil {
		t.Fatalf("from map: %v", err)
	}
	if !reflect.DeepEqual(restored, audit) {
		t.Fatalf("restored audit mismatch")
	}
}

func TestRenderConsoleIAReportSummarizesSurfaceCoverage(t *testing.T) {
	t.Parallel()

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
			t.Fatalf("expected %q in report", fragment)
		}
	}
}

func TestConsoleInteractionDraftRoundTripPreservesFourPageManifest(t *testing.T) {
	t.Parallel()

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
			Navigation: []NavigationItem{
				{Name: "Overview", Route: "/overview", Section: "Operate"},
				{Name: "Queue", Route: "/queue", Section: "Operate"},
				{Name: "Run Detail", Route: "/runs/detail", Section: "Operate"},
				{Name: "Triage", Route: "/triage", Section: "Operate"},
			},
			Surfaces: []ConsoleSurface{
				{Name: "Overview", Route: "/overview", NavigationSection: "Operate"},
				{Name: "Queue", Route: "/queue", NavigationSection: "Operate"},
				{Name: "Run Detail", Route: "/runs/detail", NavigationSection: "Operate"},
				{Name: "Triage", Route: "/triage", NavigationSection: "Operate"},
			},
		},
		Contracts: []SurfaceInteractionContract{
			{SurfaceName: "Overview", RequiresFilters: true},
			{SurfaceName: "Queue", RequiresFilters: true, RequiresBatchActions: true},
			{SurfaceName: "Run Detail", RequiresFilters: true},
			{SurfaceName: "Triage", RequiresFilters: true},
		},
	}

	data, err := draft.ToMap()
	if err != nil {
		t.Fatalf("to map: %v", err)
	}
	restored, err := ConsoleInteractionDraftFromMap(data)
	if err != nil {
		t.Fatalf("from map: %v", err)
	}
	if !reflect.DeepEqual(restored, draft) {
		t.Fatalf("restored draft mismatch")
	}
}

func TestConsoleInteractionAuditSurfacesMissingActionsPermissionsAndBatchOps(t *testing.T) {
	t.Parallel()

	architecture := BuildBig4203ConsoleInteractionDraft().Architecture
	architecture.Surfaces = []ConsoleSurface{
		{
			Name:              "Overview",
			Route:             "/overview",
			NavigationSection: "Operate",
			TopBarActions:     []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}},
			Filters:           []FilterDefinition{{Name: "Team", Field: "team", Control: "select", Options: []string{"all"}}},
			States:            []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"export"}}, {Name: "empty", AllowedActions: []string{"export"}}, {Name: "error", AllowedActions: []string{"audit"}}},
		},
		{
			Name:              "Queue",
			Route:             "/queue",
			NavigationSection: "Operate",
			TopBarActions:     []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}},
			Filters:           []FilterDefinition{{Name: "Status", Field: "status", Control: "select", Options: []string{"all"}}},
			States:            []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"audit"}}, {Name: "empty", AllowedActions: []string{"audit"}}},
		},
		{
			Name:              "Run Detail",
			Route:             "/runs/detail",
			NavigationSection: "Operate",
			TopBarActions:     []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}},
			Filters:           []FilterDefinition{{Name: "Run", Field: "run_id", Control: "search"}},
			States:            []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"export"}}, {Name: "empty", AllowedActions: []string{"drill-down"}}, {Name: "error", AllowedActions: []string{"audit"}}},
		},
		{
			Name:              "Triage",
			Route:             "/triage",
			NavigationSection: "Operate",
			TopBarActions:     []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}, {ActionID: "bulk-assign", Label: "Bulk Assign", Placement: "topbar", RequiresSelection: true}},
			States:            []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"export"}}, {Name: "empty", AllowedActions: []string{"audit"}}, {Name: "error", AllowedActions: []string{"audit"}}},
		},
	}
	draft := ConsoleInteractionDraft{
		Name:         "BIG-4203 Four Critical Pages",
		Version:      "v1",
		Architecture: architecture,
		Contracts: []SurfaceInteractionContract{
			{SurfaceName: "Overview", RequiredActionIDs: []string{"drill-down", "export", "audit"}, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "overview.access.denied"}},
			{SurfaceName: "Queue", RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiresBatchActions: true, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}}},
			{SurfaceName: "Run Detail", RequiredActionIDs: []string{"drill-down", "export", "audit"}, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator", "viewer"}, AuditEvent: "run-detail.access.denied"}},
			{SurfaceName: "Triage", RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiresFilters: true, RequiresBatchActions: true, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "triage.access.denied"}},
		},
	}

	audit := (ConsoleInteractionAuditor{}).Audit(draft)
	if audit.Name != "BIG-4203 Four Critical Pages" ||
		audit.Version != "v1" ||
		audit.ContractCount != 4 ||
		!reflect.DeepEqual(audit.SurfacesMissingFilters, []string{"Triage"}) ||
		!reflect.DeepEqual(audit.SurfacesMissingActions, map[string][]string{"Queue": {"export"}}) ||
		!reflect.DeepEqual(audit.SurfacesMissingBatchActions, []string{"Queue"}) ||
		!reflect.DeepEqual(audit.SurfacesMissingStates, map[string][]string{"Queue": {"error"}}) ||
		!reflect.DeepEqual(audit.PermissionGaps, map[string][]string{"Queue": {"audit-event"}, "Run Detail": {"denied-roles"}}) ||
		audit.ReadinessScore() != 0 || audit.ReleaseReady() {
		t.Fatalf("unexpected audit: %+v", audit)
	}
}

func TestRenderConsoleInteractionReportSummarizesCriticalPageContracts(t *testing.T) {
	t.Parallel()

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
			Navigation: []NavigationItem{
				{Name: "Overview", Route: "/overview", Section: "Operate"},
				{Name: "Queue", Route: "/queue", Section: "Operate"},
				{Name: "Run Detail", Route: "/runs/detail", Section: "Operate"},
				{Name: "Triage", Route: "/triage", Section: "Operate"},
			},
			Surfaces: []ConsoleSurface{
				{Name: "Overview", Route: "/overview", NavigationSection: "Operate", TopBarActions: []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}}, Filters: []FilterDefinition{{Name: "Team", Field: "team", Control: "select", Options: []string{"all"}}}, States: []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"export"}}, {Name: "empty", AllowedActions: []string{"drill-down"}}, {Name: "error", AllowedActions: []string{"audit"}}}},
				{Name: "Queue", Route: "/queue", NavigationSection: "Operate", TopBarActions: []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}, {ActionID: "bulk-approve", Label: "Bulk Approve", Placement: "topbar", RequiresSelection: true}}, Filters: []FilterDefinition{{Name: "Status", Field: "status", Control: "select", Options: []string{"all"}}}, States: []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"export"}}, {Name: "empty", AllowedActions: []string{"audit"}}, {Name: "error", AllowedActions: []string{"audit"}}}},
				{Name: "Run Detail", Route: "/runs/detail", NavigationSection: "Operate", TopBarActions: []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}}, Filters: []FilterDefinition{{Name: "Run", Field: "run_id", Control: "search"}}, States: []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"export"}}, {Name: "empty", AllowedActions: []string{"drill-down"}}, {Name: "error", AllowedActions: []string{"audit"}}}},
				{Name: "Triage", Route: "/triage", NavigationSection: "Operate", TopBarActions: []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}, {ActionID: "bulk-assign", Label: "Bulk Assign", Placement: "topbar", RequiresSelection: true}}, Filters: []FilterDefinition{{Name: "Severity", Field: "severity", Control: "select", Options: []string{"all"}}}, States: []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"export"}}, {Name: "empty", AllowedActions: []string{"audit"}}, {Name: "error", AllowedActions: []string{"audit"}}}},
			},
		},
		Contracts: []SurfaceInteractionContract{
			{SurfaceName: "Overview", RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiresFilters: true, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "overview.access.denied"}},
			{SurfaceName: "Queue", RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiresFilters: true, RequiresBatchActions: true, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "queue.access.denied"}},
			{SurfaceName: "Run Detail", RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiresFilters: true, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator", "viewer"}, DeniedRoles: []string{"guest"}, AuditEvent: "run-detail.access.denied"}},
			{SurfaceName: "Triage", RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiresFilters: true, RequiresBatchActions: true, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "triage.access.denied"}},
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
			t.Fatalf("expected %q in report", fragment)
		}
	}
}

func TestBuildBig4203ConsoleInteractionDraftIsReleaseReady(t *testing.T) {
	t.Parallel()

	draft := BuildBig4203ConsoleInteractionDraft()
	audit := (ConsoleInteractionAuditor{}).Audit(draft)
	report := RenderConsoleInteractionReport(draft, audit)
	if !reflect.DeepEqual(draft.RequiredRoles, []string{"eng-lead", "platform-admin", "vp-eng", "cross-team-operator"}) ||
		!draft.RequiresFrameContracts ||
		!audit.ReleaseReady() ||
		len(audit.UncoveredRoles) != 0 ||
		!strings.Contains(report, "- Required Roles: eng-lead, platform-admin, vp-eng, cross-team-operator") ||
		!strings.Contains(report, "persona=VP Eng wireframe=wf-overview") {
		t.Fatalf("unexpected draft/audit/report")
	}
}
