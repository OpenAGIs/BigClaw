package consoleia

import (
	"reflect"
	"strings"
	"testing"

	"bigclaw-go/internal/designsystem"
)

func TestConsoleIARoundTripPreservesManifestShape(t *testing.T) {
	t.Parallel()

	architecture := ConsoleIA{
		Name:    "BigClaw Console IA",
		Version: "v3",
		TopBar: designsystem.ConsoleTopBar{
			Name:                      "BigClaw Global Header",
			SearchPlaceholder:         "Search runs, issues, commands",
			EnvironmentOptions:        []string{"Production", "Staging"},
			TimeRangeOptions:          []string{"24h", "7d"},
			AlertChannels:             []string{"approvals"},
			DocumentationComplete:     true,
			AccessibilityRequirements: []string{"keyboard-navigation", "screen-reader-label", "focus-visible"},
			CommandEntry: designsystem.ConsoleCommandEntry{
				TriggerLabel: "Command Menu",
				Placeholder:  "Type a command",
				Shortcut:     "Cmd+K / Ctrl+K",
				Commands:     []designsystem.CommandAction{{ID: "search-runs", Title: "Search runs", Section: "Navigate"}},
			},
		},
		Navigation: []NavigationItem{{Name: "Overview", Route: "/overview", Section: "Operate", Icon: "dashboard", BadgeCount: 2}},
		Surfaces: []ConsoleSurface{{
			Name:              "Overview",
			Route:             "/overview",
			NavigationSection: "Operate",
			TopBarActions:     []GlobalAction{{ActionID: "refresh", Label: "Refresh", Placement: "topbar"}},
			Filters:           []FilterDefinition{{Name: "Team", Field: "team", Control: "select", Options: []string{"all", "platform"}, DefaultValue: "all"}},
			States:            []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"refresh"}}, {Name: "empty", AllowedActions: []string{"refresh"}}, {Name: "error", AllowedActions: []string{"refresh"}}},
		}},
	}
	restored, err := designsystem.DeepCopy(architecture)
	if err != nil {
		t.Fatalf("deep copy: %v", err)
	}
	if !reflect.DeepEqual(restored, architecture) {
		t.Fatalf("restored = %#v", restored)
	}
}

func TestConsoleIAAuditSurfacesGlobalInteractionGaps(t *testing.T) {
	t.Parallel()

	architecture := ConsoleIA{
		Name:    "BigClaw Console IA",
		Version: "v3",
		TopBar: designsystem.ConsoleTopBar{
			Name:                      "Incomplete Header",
			SearchPlaceholder:         "",
			EnvironmentOptions:        []string{"Production"},
			TimeRangeOptions:          []string{"24h"},
			DocumentationComplete:     false,
			AccessibilityRequirements: []string{"focus-visible"},
			CommandEntry:              designsystem.ConsoleCommandEntry{TriggerLabel: "", Placeholder: "", Shortcut: "Cmd+K"},
		},
		Navigation: []NavigationItem{{Name: "Overview", Route: "/overview", Section: "Operate"}, {Name: "Ghost", Route: "/ghost", Section: "Operate"}},
		Surfaces: []ConsoleSurface{
			{Name: "Overview", Route: "/overview", NavigationSection: "Operate", TopBarActions: []GlobalAction{{ActionID: "refresh", Label: "Refresh", Placement: "topbar"}}, Filters: []FilterDefinition{{Name: "Team", Field: "team", Control: "select", Options: []string{"all"}}}, States: []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"refresh"}}, {Name: "empty", AllowedActions: []string{"refresh"}}, {Name: "error", AllowedActions: []string{"refresh"}}}},
			{Name: "Queue", Route: "/queue", NavigationSection: "Operate", States: []SurfaceState{{Name: "default"}, {Name: "loading"}, {Name: "empty", AllowedActions: []string{"retry"}}}},
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
		audit.ReadinessScore() != 0.0 {
		t.Fatalf("unexpected audit = %#v", audit)
	}
}

func TestConsoleIAAuditRoundTripPreservesFindings(t *testing.T) {
	t.Parallel()

	topBarAudit := ConsoleIAAuditor{}.Audit(ConsoleIA{
		Name:    "BigClaw Console IA",
		Version: "v3",
		TopBar: designsystem.ConsoleTopBar{
			Name:                      "Incomplete Header",
			SearchPlaceholder:         "",
			EnvironmentOptions:        []string{"Production"},
			TimeRangeOptions:          []string{"24h"},
			DocumentationComplete:     false,
			AccessibilityRequirements: []string{"focus-visible"},
			CommandEntry:              designsystem.ConsoleCommandEntry{TriggerLabel: "", Placeholder: "", Shortcut: "Cmd+K"},
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
	restored, err := designsystem.DeepCopy(audit)
	if err != nil {
		t.Fatalf("deep copy: %v", err)
	}
	if !reflect.DeepEqual(restored, audit) {
		t.Fatalf("restored = %#v", restored)
	}
}

func TestRenderConsoleIAReportSummarizesSurfaceCoverage(t *testing.T) {
	t.Parallel()

	architecture := ConsoleIA{
		Name:    "BigClaw Console IA",
		Version: "v3",
		TopBar: designsystem.ConsoleTopBar{
			Name:                      "BigClaw Global Header",
			SearchPlaceholder:         "Search runs, issues, commands",
			EnvironmentOptions:        []string{"Production", "Staging"},
			TimeRangeOptions:          []string{"24h", "7d", "30d"},
			AlertChannels:             []string{"approvals", "sla"},
			DocumentationComplete:     true,
			AccessibilityRequirements: []string{"keyboard-navigation", "screen-reader-label", "focus-visible"},
			CommandEntry:              designsystem.ConsoleCommandEntry{TriggerLabel: "Command Menu", Placeholder: "Type a command or jump to a run", Shortcut: "Cmd+K / Ctrl+K", Commands: []designsystem.CommandAction{{ID: "search-runs", Title: "Search runs", Section: "Navigate", Shortcut: "/"}, {ID: "open-alerts", Title: "Open alerts", Section: "Monitor"}}},
		},
		Navigation: []NavigationItem{{Name: "Overview", Route: "/overview", Section: "Operate"}},
		Surfaces:   []ConsoleSurface{{Name: "Overview", Route: "/overview", NavigationSection: "Operate", TopBarActions: []GlobalAction{{ActionID: "refresh", Label: "Refresh", Placement: "topbar"}}, Filters: []FilterDefinition{{Name: "Team", Field: "team", Control: "select", Options: []string{"all"}}}, States: []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"refresh"}}, {Name: "empty", AllowedActions: []string{"refresh"}}, {Name: "error", AllowedActions: []string{"refresh"}}}}},
	}
	report := RenderConsoleIAReport(architecture, ConsoleIAAuditor{}.Audit(architecture))
	for _, want := range []string{
		"# Console Information Architecture Report",
		"- Name: BigClaw Global Header",
		"- Release Ready: true",
		"- Navigation Items: 1",
		"- Overview: route=/overview filters=Team actions=Refresh states=default, loading, empty, error missing_states=none states_without_actions=none unresolved_state_actions=none",
		"- Surfaces missing filters: none",
		"- Undefined state actions: none",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing %q:\n%s", want, report)
		}
	}
}

func TestConsoleInteractionDraftRoundTripPreservesFourPageManifest(t *testing.T) {
	t.Parallel()
	draft := ConsoleInteractionDraft{Name: "BIG-4203 Four Critical Pages", Version: "v1", Architecture: ConsoleIA{Name: "BigClaw Console IA", Version: "v3", Surfaces: []ConsoleSurface{{Name: "Overview", Route: "/overview", NavigationSection: "Operate"}, {Name: "Queue", Route: "/queue", NavigationSection: "Operate"}, {Name: "Run Detail", Route: "/runs/detail", NavigationSection: "Operate"}, {Name: "Triage", Route: "/triage", NavigationSection: "Operate"}}}, Contracts: []SurfaceInteractionContract{{SurfaceName: "Overview"}, {SurfaceName: "Queue", RequiresBatchActions: true}, {SurfaceName: "Run Detail"}, {SurfaceName: "Triage"}}}
	restored, err := designsystem.DeepCopy(draft)
	if err != nil {
		t.Fatalf("deep copy: %v", err)
	}
	if !reflect.DeepEqual(restored, draft) {
		t.Fatalf("restored = %#v", restored)
	}
}

func TestConsoleInteractionAuditSurfacesMissingActionsPermissionsAndBatchOps(t *testing.T) {
	t.Parallel()

	draft := ConsoleInteractionDraft{
		Name:    "BIG-4203 Four Critical Pages",
		Version: "v1",
		Architecture: ConsoleIA{
			Name:    "BigClaw Console IA",
			Version: "v3",
			Surfaces: []ConsoleSurface{
				{Name: "Overview", Route: "/overview", NavigationSection: "Operate", TopBarActions: []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}}, Filters: []FilterDefinition{{Name: "Team", Field: "team", Control: "select", Options: []string{"all"}}}, States: []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"export"}}, {Name: "empty", AllowedActions: []string{"export"}}, {Name: "error", AllowedActions: []string{"audit"}}}},
				{Name: "Queue", Route: "/queue", NavigationSection: "Operate", TopBarActions: []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}}, Filters: []FilterDefinition{{Name: "Status", Field: "status", Control: "select", Options: []string{"all"}}}, States: []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"audit"}}, {Name: "empty", AllowedActions: []string{"audit"}}}},
				{Name: "Run Detail", Route: "/runs/detail", NavigationSection: "Operate", TopBarActions: []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}}, Filters: []FilterDefinition{{Name: "Run", Field: "run_id", Control: "search"}}, States: []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"export"}}, {Name: "empty", AllowedActions: []string{"drill-down"}}, {Name: "error", AllowedActions: []string{"audit"}}}},
				{Name: "Triage", Route: "/triage", NavigationSection: "Operate", TopBarActions: []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}, {ActionID: "bulk-assign", Label: "Bulk Assign", Placement: "topbar", RequiresSelection: true}}, States: []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"export"}}, {Name: "empty", AllowedActions: []string{"audit"}}, {Name: "error", AllowedActions: []string{"audit"}}}},
			},
		},
		Contracts: []SurfaceInteractionContract{
			{SurfaceName: "Overview", RequiredActionIDs: []string{"drill-down", "export", "audit"}, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "overview.access.denied"}},
			{SurfaceName: "Queue", RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiresBatchActions: true, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}}},
			{SurfaceName: "Run Detail", RequiredActionIDs: []string{"drill-down", "export", "audit"}, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator", "viewer"}, DeniedRoles: nil, AuditEvent: "run-detail.access.denied"}},
			{SurfaceName: "Triage", RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiresFilters: true, RequiresBatchActions: true, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "triage.access.denied"}},
		},
	}
	audit := ConsoleInteractionAuditor{}.Audit(draft)
	want := ConsoleInteractionAudit{
		Name:                        "BIG-4203 Four Critical Pages",
		Version:                     "v1",
		ContractCount:               4,
		MissingSurfaces:             nil,
		SurfacesMissingFilters:      []string{"Triage"},
		SurfacesMissingActions:      map[string][]string{"Queue": {"export"}},
		SurfacesMissingBatchActions: []string{"Queue"},
		SurfacesMissingStates:       map[string][]string{"Queue": {"error"}},
		PermissionGaps:              map[string][]string{"Queue": {"audit-event"}, "Run Detail": {"denied-roles"}},
	}
	if !reflect.DeepEqual(audit, want) || audit.ReadinessScore() != 0.0 || audit.ReleaseReady() {
		t.Fatalf("audit = %#v", audit)
	}
}

func TestRenderConsoleInteractionReportSummarizesCriticalPageContracts(t *testing.T) {
	t.Parallel()
	draft := BuildBIG4203ConsoleInteractionDraft()
	audit := ConsoleInteractionAuditor{}.Audit(draft)
	report := RenderConsoleInteractionReport(draft, audit)
	for _, want := range []string{
		"# Console Interaction Draft Report",
		"- Critical Pages: 4",
		"- Required Roles: eng-lead, platform-admin, vp-eng, cross-team-operator",
		"- Readiness Score: 100.0",
		"- Release Ready: true",
		"- Overview: route=/overview required_actions=drill-down, export, audit available_actions=drill-down, export, audit filters=1 states=default, loading, empty, error batch=optional permissions=complete",
		"- Queue: route=/queue required_actions=drill-down, export, audit available_actions=drill-down, export, audit, bulk-approve filters=1 states=default, loading, empty, error batch=required permissions=complete",
		"- Permission gaps: none",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing %q:\n%s", want, report)
		}
	}
}

func TestBuildBig4203ConsoleInteractionDraftIsReleaseReady(t *testing.T) {
	t.Parallel()
	draft := BuildBIG4203ConsoleInteractionDraft()
	audit := ConsoleInteractionAuditor{}.Audit(draft)
	report := RenderConsoleInteractionReport(draft, audit)
	if !reflect.DeepEqual(draft.RequiredRoles, []string{"eng-lead", "platform-admin", "vp-eng", "cross-team-operator"}) || !draft.RequiresFrameContracts || !audit.ReleaseReady() || len(audit.UncoveredRoles) != 0 {
		t.Fatalf("draft/audit = %#v %#v", draft, audit)
	}
	for _, want := range []string{
		"- Required Roles: eng-lead, platform-admin, vp-eng, cross-team-operator",
		"persona=VP Eng wireframe=wf-overview",
		"review_focus=metric hierarchy,drill-down posture,alert prioritization",
		"- Uncovered roles: none",
		"- Pages missing personas: none",
		"- Pages missing wireframe links: none",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing %q:\n%s", want, report)
		}
	}
}

func TestConsoleInteractionAuditFlagsUncoveredRequiredRoles(t *testing.T) {
	t.Parallel()
	draft := BuildBIG4203ConsoleInteractionDraft()
	draft.RequiredRoles = append(draft.RequiredRoles, "finance-reviewer")
	audit := ConsoleInteractionAuditor{}.Audit(draft)
	if !reflect.DeepEqual(audit.UncoveredRoles, []string{"finance-reviewer"}) || audit.ReleaseReady() {
		t.Fatalf("audit = %#v", audit)
	}
}

func TestConsoleInteractionAuditFlagsMissingFrameContractDetails(t *testing.T) {
	t.Parallel()
	draft := BuildBIG4203ConsoleInteractionDraft()
	draft.Contracts[0].PrimaryPersona = ""
	draft.Contracts[0].LinkedWireframeID = ""
	draft.Contracts[0].ReviewFocusAreas = nil
	draft.Contracts[0].DecisionPrompts = nil
	audit := ConsoleInteractionAuditor{}.Audit(draft)
	if !reflect.DeepEqual(audit.SurfacesMissingPrimaryPersonas, []string{"Overview"}) ||
		!reflect.DeepEqual(audit.SurfacesMissingWireframeLinks, []string{"Overview"}) ||
		!reflect.DeepEqual(audit.SurfacesMissingReviewFocus, []string{"Overview"}) ||
		!reflect.DeepEqual(audit.SurfacesMissingDecisionPrompts, []string{"Overview"}) ||
		audit.ReleaseReady() {
		t.Fatalf("audit = %#v", audit)
	}
}
