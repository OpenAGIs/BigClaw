package consoleiacompat

import (
	"reflect"
	"strings"
	"testing"

	"bigclaw-go/internal/designsystemcompat"
)

func testTopBar() designsystemcompat.ConsoleTopBar {
	return designsystemcompat.ConsoleTopBar{
		Name:                      "BigClaw Global Header",
		SearchPlaceholder:         "Search runs, issues, commands",
		EnvironmentOptions:        []string{"Production", "Staging"},
		TimeRangeOptions:          []string{"24h", "7d"},
		AlertChannels:             []string{"approvals"},
		DocumentationComplete:     true,
		AccessibilityRequirements: []string{"keyboard-navigation", "screen-reader-label", "focus-visible"},
		CommandEntry: designsystemcompat.ConsoleCommandEntry{
			TriggerLabel: "Command Menu",
			Placeholder:  "Type a command",
			Shortcut:     "Cmd+K / Ctrl+K",
			Commands: []designsystemcompat.CommandAction{
				{ID: "search-runs", Title: "Search runs", Section: "Navigate"},
			},
		},
	}
}

func TestConsoleIARoundTripPreservesManifestShape(t *testing.T) {
	architecture := ConsoleIA{
		Name:    "BigClaw Console IA",
		Version: "v3",
		TopBar:  testTopBar(),
		Navigation: []NavigationItem{
			{Name: "Overview", Route: "/overview", Section: "Operate", Icon: "dashboard", BadgeCount: 2},
		},
		Surfaces: []ConsoleSurface{
			{
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
			},
		},
	}

	restored := ConsoleIAFromMap(architecture.ToMap())
	if !reflect.DeepEqual(restored, architecture) {
		t.Fatalf("restored architecture mismatch: %+v", restored)
	}
}

func TestConsoleIAAuditSurfacesGlobalInteractionGaps(t *testing.T) {
	architecture := ConsoleIA{
		Name:    "BigClaw Console IA",
		Version: "v3",
		TopBar: designsystemcompat.ConsoleTopBar{
			Name:                      "Incomplete Header",
			SearchPlaceholder:         "",
			EnvironmentOptions:        []string{"Production"},
			TimeRangeOptions:          []string{"24h"},
			DocumentationComplete:     false,
			AccessibilityRequirements: []string{"focus-visible"},
			CommandEntry: designsystemcompat.ConsoleCommandEntry{
				TriggerLabel: "",
				Placeholder:  "",
				Shortcut:     "Cmd+K",
			},
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

	audit := ConsoleIAAuditor{}.Audit(architecture)
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
		t.Fatalf("unexpected IA audit: %+v", audit)
	}
}

func TestConsoleIAAuditRoundTripAndReport(t *testing.T) {
	audit := ConsoleIAAudit{
		SystemName:             "BigClaw Console IA",
		Version:                "v3",
		SurfaceCount:           2,
		NavigationCount:        1,
		TopBarAudit:            ConsoleIAAuditor{}.Audit(ConsoleIA{Name: "BigClaw Console IA", Version: "v3", TopBar: designsystemcompat.ConsoleTopBar{Name: "Incomplete Header", AccessibilityRequirements: []string{"focus-visible"}, CommandEntry: designsystemcompat.ConsoleCommandEntry{Shortcut: "Cmd+K"}, EnvironmentOptions: []string{"Production"}, TimeRangeOptions: []string{"24h"}}}).TopBarAudit,
		SurfacesMissingFilters: []string{"Queue"},
		SurfacesMissingActions: []string{"Queue"},
		SurfacesMissingStates:  map[string][]string{"Queue": {"error"}},
		StatesMissingActions:   map[string][]string{"Queue": {"loading"}},
		UnresolvedStateActions: map[string]map[string][]string{"Queue": {"empty": {"retry"}}},
		OrphanNavigationRoutes: []string{"/ghost"},
		UnnavigableSurfaces:    []string{"Queue"},
	}
	restored := ConsoleIAAuditFromMap(audit.ToMap())
	if !reflect.DeepEqual(restored, audit) {
		t.Fatalf("restored audit mismatch: %+v", restored)
	}

	report := RenderConsoleIAReport(ConsoleIA{
		Name:    "BigClaw Console IA",
		Version: "v3",
		TopBar:  testTopBar(),
		Navigation: []NavigationItem{
			{Name: "Overview", Route: "/overview", Section: "Operate"},
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
		},
	}, ConsoleIAAuditor{}.Audit(ConsoleIA{
		Name:    "BigClaw Console IA",
		Version: "v3",
		TopBar:  testTopBar(),
		Navigation: []NavigationItem{
			{Name: "Overview", Route: "/overview", Section: "Operate"},
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
		},
	}))
	for _, want := range []string{"# Console Information Architecture Report", "- Name: BigClaw Global Header", "- Release Ready: true", "- Navigation Items: 1", "- Overview: route=/overview filters=Team actions=Refresh states=default, loading, empty, error missing_states=none states_without_actions=none unresolved_state_actions=none", "- Surfaces missing filters: none", "- Undefined state actions: none"} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in report, got %s", want, report)
		}
	}
}

func TestConsoleInteractionDraftRoundTripPreservesManifest(t *testing.T) {
	draft := ConsoleInteractionDraft{
		Name:    "BIG-4203 Four Critical Pages",
		Version: "v1",
		Architecture: ConsoleIA{
			Name:    "BigClaw Console IA",
			Version: "v3",
			TopBar:  testTopBar(),
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
			NewSurfaceInteractionContract("Overview"),
			func() SurfaceInteractionContract {
				c := NewSurfaceInteractionContract("Queue")
				c.RequiresBatchActions = true
				return c
			}(),
			NewSurfaceInteractionContract("Run Detail"),
			NewSurfaceInteractionContract("Triage"),
		},
	}
	restored := ConsoleInteractionDraftFromMap(draft.ToMap())
	if !reflect.DeepEqual(restored, draft) {
		t.Fatalf("restored draft mismatch: %+v", restored)
	}
}

func TestConsoleInteractionAuditSurfacesMissingActionsPermissionsAndBatchOps(t *testing.T) {
	architecture := ConsoleIA{
		Name:    "BigClaw Console IA",
		Version: "v3",
		TopBar:  testTopBar(),
		Navigation: []NavigationItem{
			{Name: "Overview", Route: "/overview", Section: "Operate"},
			{Name: "Queue", Route: "/queue", Section: "Operate"},
			{Name: "Run Detail", Route: "/runs/detail", Section: "Operate"},
			{Name: "Triage", Route: "/triage", Section: "Operate"},
		},
		Surfaces: []ConsoleSurface{
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
		},
	}
	draft := ConsoleInteractionDraft{
		Name:         "BIG-4203 Four Critical Pages",
		Version:      "v1",
		Architecture: architecture,
		Contracts: []SurfaceInteractionContract{
			{SurfaceName: "Overview", RequiresFilters: true, RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiredStates: []string{"default", "loading", "empty", "error"}, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "overview.access.denied"}},
			{SurfaceName: "Queue", RequiresFilters: true, RequiresBatchActions: true, RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiredStates: []string{"default", "loading", "empty", "error"}, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}}},
			{SurfaceName: "Run Detail", RequiresFilters: true, RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiredStates: []string{"default", "loading", "empty", "error"}, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator", "viewer"}, AuditEvent: "run-detail.access.denied"}},
			{SurfaceName: "Triage", RequiresFilters: true, RequiresBatchActions: true, RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiredStates: []string{"default", "loading", "empty", "error"}, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "triage.access.denied"}},
		},
	}
	audit := ConsoleInteractionAuditor{}.Audit(draft)
	if audit.Name != "BIG-4203 Four Critical Pages" ||
		audit.Version != "v1" ||
		audit.ContractCount != 4 ||
		len(audit.MissingSurfaces) != 0 ||
		!reflect.DeepEqual(audit.SurfacesMissingFilters, []string{"Triage"}) ||
		!reflect.DeepEqual(audit.SurfacesMissingActions, map[string][]string{"Queue": {"export"}}) ||
		!reflect.DeepEqual(audit.SurfacesMissingBatchActions, []string{"Queue"}) ||
		!reflect.DeepEqual(audit.SurfacesMissingStates, map[string][]string{"Queue": {"error"}}) ||
		!reflect.DeepEqual(audit.PermissionGaps, map[string][]string{"Queue": {"audit-event"}, "Run Detail": {"denied-roles"}}) ||
		audit.ReadinessScore() != 0 ||
		audit.ReleaseReady() {
		t.Fatalf("unexpected interaction audit: %+v", audit)
	}
}

func TestRenderConsoleInteractionReportAndBuilder(t *testing.T) {
	draft := ConsoleInteractionDraft{
		Name:    "BIG-4203 Four Critical Pages",
		Version: "v1",
		Architecture: ConsoleIA{
			Name:       "BigClaw Console IA",
			Version:    "v3",
			TopBar:     testTopBar(),
			Navigation: []NavigationItem{{Name: "Overview", Route: "/overview", Section: "Operate"}, {Name: "Queue", Route: "/queue", Section: "Operate"}, {Name: "Run Detail", Route: "/runs/detail", Section: "Operate"}, {Name: "Triage", Route: "/triage", Section: "Operate"}},
			Surfaces: []ConsoleSurface{
				{Name: "Overview", Route: "/overview", NavigationSection: "Operate", TopBarActions: []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}}, Filters: []FilterDefinition{{Name: "Team", Field: "team", Control: "select", Options: []string{"all"}}}, States: []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"export"}}, {Name: "empty", AllowedActions: []string{"drill-down"}}, {Name: "error", AllowedActions: []string{"audit"}}}},
				{Name: "Queue", Route: "/queue", NavigationSection: "Operate", TopBarActions: []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}, {ActionID: "bulk-approve", Label: "Bulk Approve", Placement: "topbar", RequiresSelection: true}}, Filters: []FilterDefinition{{Name: "Status", Field: "status", Control: "select", Options: []string{"all"}}}, States: []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"export"}}, {Name: "empty", AllowedActions: []string{"audit"}}, {Name: "error", AllowedActions: []string{"audit"}}}},
				{Name: "Run Detail", Route: "/runs/detail", NavigationSection: "Operate", TopBarActions: []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}}, Filters: []FilterDefinition{{Name: "Run", Field: "run_id", Control: "search"}}, States: []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"export"}}, {Name: "empty", AllowedActions: []string{"drill-down"}}, {Name: "error", AllowedActions: []string{"audit"}}}},
				{Name: "Triage", Route: "/triage", NavigationSection: "Operate", TopBarActions: []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}, {ActionID: "bulk-assign", Label: "Bulk Assign", Placement: "topbar", RequiresSelection: true}}, Filters: []FilterDefinition{{Name: "Severity", Field: "severity", Control: "select", Options: []string{"all"}}}, States: []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"export"}}, {Name: "empty", AllowedActions: []string{"audit"}}, {Name: "error", AllowedActions: []string{"audit"}}}},
			},
		},
		Contracts: []SurfaceInteractionContract{
			{SurfaceName: "Overview", RequiresFilters: true, RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiredStates: []string{"default", "loading", "empty", "error"}, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "overview.access.denied"}},
			{SurfaceName: "Queue", RequiresFilters: true, RequiresBatchActions: true, RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiredStates: []string{"default", "loading", "empty", "error"}, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "queue.access.denied"}},
			{SurfaceName: "Run Detail", RequiresFilters: true, RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiredStates: []string{"default", "loading", "empty", "error"}, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator", "viewer"}, DeniedRoles: []string{"guest"}, AuditEvent: "run-detail.access.denied"}},
			{SurfaceName: "Triage", RequiresFilters: true, RequiresBatchActions: true, RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiredStates: []string{"default", "loading", "empty", "error"}, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "triage.access.denied"}},
		},
	}
	audit := ConsoleInteractionAuditor{}.Audit(draft)
	report := RenderConsoleInteractionReport(draft, audit)
	for _, want := range []string{"# Console Interaction Draft Report", "- Critical Pages: 4", "- Required Roles: none", "- Readiness Score: 100.0", "- Release Ready: true", "- Overview: route=/overview required_actions=drill-down, export, audit available_actions=drill-down, export, audit filters=1 states=default, loading, empty, error batch=optional permissions=complete", "- Queue: route=/queue required_actions=drill-down, export, audit available_actions=drill-down, export, audit, bulk-approve filters=1 states=default, loading, empty, error batch=required permissions=complete", "- Permission gaps: none"} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in report, got %s", want, report)
		}
	}

	releaseDraft := BuildBIG4203ConsoleInteractionDraft()
	releaseAudit := ConsoleInteractionAuditor{}.Audit(releaseDraft)
	releaseReport := RenderConsoleInteractionReport(releaseDraft, releaseAudit)
	if !reflect.DeepEqual(releaseDraft.RequiredRoles, []string{"eng-lead", "platform-admin", "vp-eng", "cross-team-operator"}) ||
		!releaseDraft.RequiresFrameContracts ||
		!releaseAudit.ReleaseReady() ||
		len(releaseAudit.UncoveredRoles) != 0 {
		t.Fatalf("unexpected release draft audit: %+v", releaseAudit)
	}
	for _, want := range []string{"- Required Roles: eng-lead, platform-admin, vp-eng, cross-team-operator", "persona=VP Eng wireframe=wf-overview", "review_focus=metric hierarchy,drill-down posture,alert prioritization", "- Uncovered roles: none", "- Pages missing personas: none", "- Pages missing wireframe links: none"} {
		if !strings.Contains(releaseReport, want) {
			t.Fatalf("expected %q in release report, got %s", want, releaseReport)
		}
	}
}

func TestConsoleInteractionAuditFlagsUncoveredRolesAndMissingFrameContracts(t *testing.T) {
	draft := BuildBIG4203ConsoleInteractionDraft()
	draft.RequiredRoles = append(draft.RequiredRoles, "finance-reviewer")
	audit := ConsoleInteractionAuditor{}.Audit(draft)
	if !reflect.DeepEqual(audit.UncoveredRoles, []string{"finance-reviewer"}) || audit.ReleaseReady() {
		t.Fatalf("unexpected uncovered roles audit: %+v", audit)
	}

	draft = BuildBIG4203ConsoleInteractionDraft()
	draft.Contracts[0].PrimaryPersona = ""
	draft.Contracts[0].LinkedWireframeID = ""
	draft.Contracts[0].ReviewFocusAreas = nil
	draft.Contracts[0].DecisionPrompts = nil
	audit = ConsoleInteractionAuditor{}.Audit(draft)
	if !reflect.DeepEqual(audit.SurfacesMissingPrimaryPersonas, []string{"Overview"}) ||
		!reflect.DeepEqual(audit.SurfacesMissingWireframeLinks, []string{"Overview"}) ||
		!reflect.DeepEqual(audit.SurfacesMissingReviewFocus, []string{"Overview"}) ||
		!reflect.DeepEqual(audit.SurfacesMissingDecisionPrompts, []string{"Overview"}) ||
		audit.ReleaseReady() {
		t.Fatalf("unexpected missing frame contract audit: %+v", audit)
	}
}
