package consoleia

import (
	"reflect"
	"strings"
	"testing"

	"bigclaw-go/internal/designsystem"
)

func sampleTopBar() designsystem.ConsoleTopBar {
	return designsystem.ConsoleTopBar{
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
			Commands: []designsystem.CommandAction{
				{ID: "search-runs", Title: "Search runs", Section: "Navigate"},
			},
		},
	}
}

func TestConsoleIARoundTripPreservesManifestShape(t *testing.T) {
	architecture := ConsoleIA{
		Name:    "BigClaw Console IA",
		Version: "v3",
		TopBar:  sampleTopBar(),
		Navigation: []NavigationItem{
			{Name: "Overview", Route: "/overview", Section: "Operate", Icon: "dashboard", BadgeCount: 2},
		},
		Surfaces: []ConsoleSurface{
			{
				Name:              "Overview",
				Route:             "/overview",
				NavigationSection: "Operate",
				TopBarActions:     []GlobalAction{{ActionID: "refresh", Label: "Refresh", Placement: "topbar"}},
				Filters: []FilterDefinition{
					{Name: "Team", Field: "team", Control: "select", Options: []string{"all", "platform"}, DefaultValue: "all"},
				},
				States: []SurfaceState{
					{Name: "default"},
					{Name: "loading", AllowedActions: []string{"refresh"}},
					{Name: "empty", AllowedActions: []string{"refresh"}},
					{Name: "error", AllowedActions: []string{"refresh"}},
				},
			},
		},
	}

	restored := MustJSONRoundTrip(architecture)
	if !reflect.DeepEqual(restored, architecture) {
		t.Fatalf("round trip mismatch: got=%+v want=%+v", restored, architecture)
	}
}

func TestConsoleIAAuditSurfacesGlobalInteractionGaps(t *testing.T) {
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
	if !reflect.DeepEqual(audit.SurfacesMissingFilters, []string{"Queue"}) {
		t.Fatalf("unexpected filters gap: %+v", audit.SurfacesMissingFilters)
	}
	if !reflect.DeepEqual(audit.SurfacesMissingActions, []string{"Queue"}) {
		t.Fatalf("unexpected actions gap: %+v", audit.SurfacesMissingActions)
	}
	if !reflect.DeepEqual(audit.TopBarAudit.MissingCapabilities, []string{"global-search", "time-range-switch", "environment-switch", "alert-entry", "command-shell"}) {
		t.Fatalf("unexpected top bar gaps: %+v", audit.TopBarAudit.MissingCapabilities)
	}
	if audit.TopBarAudit.ReleaseReady() {
		t.Fatalf("expected top bar to be non-ready: %+v", audit.TopBarAudit)
	}
	if !reflect.DeepEqual(audit.SurfacesMissingStates, map[string][]string{"Queue": {"error"}}) {
		t.Fatalf("unexpected state gaps: %+v", audit.SurfacesMissingStates)
	}
	if !reflect.DeepEqual(audit.StatesMissingActions, map[string][]string{"Queue": {"loading"}}) {
		t.Fatalf("unexpected state action gaps: %+v", audit.StatesMissingActions)
	}
	if !reflect.DeepEqual(audit.UnresolvedStateActions, map[string]map[string][]string{"Queue": {"empty": {"retry"}}}) {
		t.Fatalf("unexpected unresolved actions: %+v", audit.UnresolvedStateActions)
	}
	if !reflect.DeepEqual(audit.OrphanNavigationRoutes, []string{"/ghost"}) {
		t.Fatalf("unexpected orphan routes: %+v", audit.OrphanNavigationRoutes)
	}
	if !reflect.DeepEqual(audit.UnnavigableSurfaces, []string{"Queue"}) {
		t.Fatalf("unexpected unnavigable surfaces: %+v", audit.UnnavigableSurfaces)
	}
	if got := audit.ReadinessScore(); got != 0.0 {
		t.Fatalf("unexpected readiness score: %v", got)
	}
}

func TestConsoleIAAuditRoundTripPreservesFindings(t *testing.T) {
	audit := ConsoleIAAudit{
		SystemName:      "BigClaw Console IA",
		Version:         "v3",
		SurfaceCount:    2,
		NavigationCount: 1,
		TopBarAudit: ConsoleIAAuditor{}.Audit(ConsoleIA{
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
		}).TopBarAudit,
		SurfacesMissingFilters: []string{"Queue"},
		SurfacesMissingActions: []string{"Queue"},
		SurfacesMissingStates:  map[string][]string{"Queue": {"error"}},
		StatesMissingActions:   map[string][]string{"Queue": {"loading"}},
		UnresolvedStateActions: map[string]map[string][]string{"Queue": {"empty": {"retry"}}},
		OrphanNavigationRoutes: []string{"/ghost"},
		UnnavigableSurfaces:    []string{"Queue"},
	}

	restored := MustJSONRoundTrip(audit)
	if !reflect.DeepEqual(restored, audit) {
		t.Fatalf("round trip mismatch: got=%+v want=%+v", restored, audit)
	}
}

func TestRenderConsoleIAReportSummarizesSurfaceCoverage(t *testing.T) {
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
			CommandEntry: designsystem.ConsoleCommandEntry{
				TriggerLabel: "Command Menu",
				Placeholder:  "Type a command or jump to a run",
				Shortcut:     "Cmd+K / Ctrl+K",
				Commands: []designsystem.CommandAction{
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
			t.Fatalf("expected %q in report:\n%s", want, report)
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
			TopBar:  sampleTopBar(),
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
			{SurfaceName: "Overview"},
			{SurfaceName: "Queue", RequiresBatchActions: true},
			{SurfaceName: "Run Detail"},
			{SurfaceName: "Triage"},
		},
	}

	restored := MustJSONRoundTrip(draft)
	want := draft
	for i := range want.Contracts {
		want.Contracts[i].RequiredStates = append([]string(nil), requiredSurfaceStates...)
	}
	if !reflect.DeepEqual(restored, want) {
		t.Fatalf("round trip mismatch: got=%+v want=%+v", restored, want)
	}
}

func TestConsoleInteractionAuditSurfacesMissingActionsPermissionsAndBatchOps(t *testing.T) {
	architecture := ConsoleIA{
		Name:    "BigClaw Console IA",
		Version: "v3",
		TopBar:  sampleTopBar(),
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
				TopBarActions: []GlobalAction{
					{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"},
					{ActionID: "export", Label: "Export", Placement: "topbar"},
					{ActionID: "audit", Label: "Audit Trail", Placement: "topbar"},
				},
				Filters: []FilterDefinition{{Name: "Team", Field: "team", Control: "select", Options: []string{"all"}}},
				States: []SurfaceState{
					{Name: "default"},
					{Name: "loading", AllowedActions: []string{"export"}},
					{Name: "empty", AllowedActions: []string{"export"}},
					{Name: "error", AllowedActions: []string{"audit"}},
				},
			},
			{
				Name:              "Queue",
				Route:             "/queue",
				NavigationSection: "Operate",
				TopBarActions: []GlobalAction{
					{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"},
					{ActionID: "audit", Label: "Audit Trail", Placement: "topbar"},
				},
				Filters: []FilterDefinition{{Name: "Status", Field: "status", Control: "select", Options: []string{"all"}}},
				States: []SurfaceState{
					{Name: "default"},
					{Name: "loading", AllowedActions: []string{"audit"}},
					{Name: "empty", AllowedActions: []string{"audit"}},
				},
			},
			{
				Name:              "Run Detail",
				Route:             "/runs/detail",
				NavigationSection: "Operate",
				TopBarActions: []GlobalAction{
					{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"},
					{ActionID: "export", Label: "Export", Placement: "topbar"},
					{ActionID: "audit", Label: "Audit Trail", Placement: "topbar"},
				},
				Filters: []FilterDefinition{{Name: "Run", Field: "run_id", Control: "search"}},
				States: []SurfaceState{
					{Name: "default"},
					{Name: "loading", AllowedActions: []string{"export"}},
					{Name: "empty", AllowedActions: []string{"drill-down"}},
					{Name: "error", AllowedActions: []string{"audit"}},
				},
			},
			{
				Name:              "Triage",
				Route:             "/triage",
				NavigationSection: "Operate",
				TopBarActions: []GlobalAction{
					{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"},
					{ActionID: "export", Label: "Export", Placement: "topbar"},
					{ActionID: "audit", Label: "Audit Trail", Placement: "topbar"},
					{ActionID: "bulk-assign", Label: "Bulk Assign", Placement: "topbar", RequiresSelection: true},
				},
				States: []SurfaceState{
					{Name: "default"},
					{Name: "loading", AllowedActions: []string{"export"}},
					{Name: "empty", AllowedActions: []string{"audit"}},
					{Name: "error", AllowedActions: []string{"audit"}},
				},
			},
		},
	}
	draft := ConsoleInteractionDraft{
		Name:         "BIG-4203 Four Critical Pages",
		Version:      "v1",
		Architecture: architecture,
		Contracts: []SurfaceInteractionContract{
			{SurfaceName: "Overview", RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiredStates: append([]string(nil), requiredSurfaceStates...), PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "overview.access.denied"}},
			{SurfaceName: "Queue", RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiresBatchActions: true, RequiredStates: append([]string(nil), requiredSurfaceStates...), PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}}},
			{SurfaceName: "Run Detail", RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiredStates: append([]string(nil), requiredSurfaceStates...), PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator", "viewer"}, AuditEvent: "run-detail.access.denied"}},
			{SurfaceName: "Triage", RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiresFilters: true, RequiresBatchActions: true, RequiredStates: append([]string(nil), requiredSurfaceStates...), PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "triage.access.denied"}},
		},
	}

	audit := ConsoleInteractionAuditor{}.Audit(draft)
	if audit.Name != "BIG-4203 Four Critical Pages" || audit.Version != "v1" || audit.ContractCount != 4 {
		t.Fatalf("unexpected audit header: %+v", audit)
	}
	if len(audit.MissingSurfaces) != 0 {
		t.Fatalf("unexpected missing surfaces: %+v", audit.MissingSurfaces)
	}
	if !reflect.DeepEqual(audit.SurfacesMissingFilters, []string{"Triage"}) {
		t.Fatalf("unexpected filter gaps: %+v", audit.SurfacesMissingFilters)
	}
	if !reflect.DeepEqual(audit.SurfacesMissingActions, map[string][]string{"Queue": {"export"}}) {
		t.Fatalf("unexpected action gaps: %+v", audit.SurfacesMissingActions)
	}
	if !reflect.DeepEqual(audit.SurfacesMissingBatchActions, []string{"Queue"}) {
		t.Fatalf("unexpected batch gaps: %+v", audit.SurfacesMissingBatchActions)
	}
	if !reflect.DeepEqual(audit.SurfacesMissingStates, map[string][]string{"Queue": {"error"}}) {
		t.Fatalf("unexpected state gaps: %+v", audit.SurfacesMissingStates)
	}
	if !reflect.DeepEqual(audit.PermissionGaps, map[string][]string{"Queue": {"audit-event"}, "Run Detail": {"denied-roles"}}) {
		t.Fatalf("unexpected permission gaps: %+v", audit.PermissionGaps)
	}
	if len(audit.UncoveredRoles) != 0 || len(audit.SurfacesMissingPrimaryPersonas) != 0 || len(audit.SurfacesMissingWireframeLinks) != 0 || len(audit.SurfacesMissingReviewFocus) != 0 || len(audit.SurfacesMissingDecisionPrompts) != 0 {
		t.Fatalf("unexpected residual frame gaps: %+v", audit)
	}
	if got := audit.ReadinessScore(); got != 0.0 {
		t.Fatalf("unexpected readiness score: %v", got)
	}
	if audit.ReleaseReady() {
		t.Fatalf("expected non-ready audit: %+v", audit)
	}
}

func TestRenderConsoleInteractionReportSummarizesCriticalPageContracts(t *testing.T) {
	draft := ConsoleInteractionDraft{
		Name:    "BIG-4203 Four Critical Pages",
		Version: "v1",
		Architecture: ConsoleIA{
			Name:    "BigClaw Console IA",
			Version: "v3",
			TopBar:  sampleTopBar(),
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
			{SurfaceName: "Overview", RequiredActionIDs: []string{"drill-down", "export", "audit"}, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "overview.access.denied"}},
			{SurfaceName: "Queue", RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiresBatchActions: true, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "queue.access.denied"}},
			{SurfaceName: "Run Detail", RequiredActionIDs: []string{"drill-down", "export", "audit"}, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator", "viewer"}, DeniedRoles: []string{"guest"}, AuditEvent: "run-detail.access.denied"}},
			{SurfaceName: "Triage", RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiresBatchActions: true, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"admin", "operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "triage.access.denied"}},
		},
	}

	report := RenderConsoleInteractionReport(draft, ConsoleInteractionAuditor{}.Audit(draft))
	for _, want := range []string{
		"# Console Interaction Draft Report",
		"- Critical Pages: 4",
		"- Required Roles: none",
		"- Readiness Score: 100.0",
		"- Release Ready: true",
		"- Overview: route=/overview required_actions=drill-down, export, audit available_actions=drill-down, export, audit filters=1 states=default, loading, empty, error batch=optional permissions=complete",
		"- Queue: route=/queue required_actions=drill-down, export, audit available_actions=drill-down, export, audit, bulk-approve filters=1 states=default, loading, empty, error batch=required permissions=complete",
		"- Permission gaps: none",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in report:\n%s", want, report)
		}
	}
}

func TestBuildBIG4203ConsoleInteractionDraftIsReleaseReady(t *testing.T) {
	draft := BuildBIG4203ConsoleInteractionDraft()
	audit := ConsoleInteractionAuditor{}.Audit(draft)
	report := RenderConsoleInteractionReport(draft, audit)

	if !reflect.DeepEqual(draft.RequiredRoles, []string{"eng-lead", "platform-admin", "vp-eng", "cross-team-operator"}) {
		t.Fatalf("unexpected required roles: %+v", draft.RequiredRoles)
	}
	if !draft.RequiresFrameContracts {
		t.Fatal("expected frame contracts to be required")
	}
	if !audit.ReleaseReady() {
		t.Fatalf("expected release ready audit: %+v", audit)
	}
	if len(audit.UncoveredRoles) != 0 {
		t.Fatalf("unexpected uncovered roles: %+v", audit.UncoveredRoles)
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
			t.Fatalf("expected %q in report:\n%s", want, report)
		}
	}
}

func TestConsoleInteractionAuditFlagsUncoveredRequiredRoles(t *testing.T) {
	draft := BuildBIG4203ConsoleInteractionDraft()
	draft.RequiredRoles = append(draft.RequiredRoles, "finance-reviewer")

	audit := ConsoleInteractionAuditor{}.Audit(draft)
	if !reflect.DeepEqual(audit.UncoveredRoles, []string{"finance-reviewer"}) {
		t.Fatalf("unexpected uncovered roles: %+v", audit.UncoveredRoles)
	}
	if audit.ReleaseReady() {
		t.Fatalf("expected non-ready audit: %+v", audit)
	}
}

func TestConsoleInteractionAuditFlagsMissingFrameContractDetails(t *testing.T) {
	draft := BuildBIG4203ConsoleInteractionDraft()
	draft.Contracts[0].PrimaryPersona = ""
	draft.Contracts[0].LinkedWireframeID = ""
	draft.Contracts[0].ReviewFocusAreas = nil
	draft.Contracts[0].DecisionPrompts = nil

	audit := ConsoleInteractionAuditor{}.Audit(draft)
	if !reflect.DeepEqual(audit.SurfacesMissingPrimaryPersonas, []string{"Overview"}) {
		t.Fatalf("unexpected persona gaps: %+v", audit.SurfacesMissingPrimaryPersonas)
	}
	if !reflect.DeepEqual(audit.SurfacesMissingWireframeLinks, []string{"Overview"}) {
		t.Fatalf("unexpected wireframe gaps: %+v", audit.SurfacesMissingWireframeLinks)
	}
	if !reflect.DeepEqual(audit.SurfacesMissingReviewFocus, []string{"Overview"}) {
		t.Fatalf("unexpected review focus gaps: %+v", audit.SurfacesMissingReviewFocus)
	}
	if !reflect.DeepEqual(audit.SurfacesMissingDecisionPrompts, []string{"Overview"}) {
		t.Fatalf("unexpected decision prompt gaps: %+v", audit.SurfacesMissingDecisionPrompts)
	}
	if audit.ReleaseReady() {
		t.Fatalf("expected non-ready audit: %+v", audit)
	}
}
