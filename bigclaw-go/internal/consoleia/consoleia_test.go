package consoleia

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"bigclaw-go/internal/designsystem"
)

func TestConsoleIARoundTripPreservesManifestShape(t *testing.T) {
	architecture := ConsoleIA{
		Name:    "BigClaw Console IA",
		Version: "v3",
		TopBar:  readyTopBar(),
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

	payload, err := json.Marshal(architecture)
	if err != nil {
		t.Fatalf("marshal architecture: %v", err)
	}
	var restored ConsoleIA
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal architecture: %v", err)
	}
	if !reflect.DeepEqual(restored, architecture) {
		t.Fatalf("restored architecture mismatch: got %+v want %+v", restored, architecture)
	}
}

func TestConsoleIAAuditSurfacesGlobalInteractionGaps(t *testing.T) {
	architecture := ConsoleIA{
		Name:    "BigClaw Console IA",
		Version: "v3",
		TopBar: designsystem.ConsoleTopBar{
			Name:              "Incomplete Header",
			SearchPlaceholder: "",
			EnvironmentOptions: []string{
				"Production",
			},
			TimeRangeOptions:          []string{"24h"},
			DocumentationComplete:     false,
			AccessibilityRequirements: []string{"focus-visible"},
			CommandEntry: designsystem.ConsoleCommandEntry{
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

	if got, want := audit.SurfacesMissingFilters, []string{"Queue"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("surfaces missing filters mismatch: got %+v want %+v", got, want)
	}
	if got, want := audit.SurfacesMissingActions, []string{"Queue"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("surfaces missing actions mismatch: got %+v want %+v", got, want)
	}
	if got, want := audit.TopBarAudit.MissingCapabilities, []string{"global-search", "time-range-switch", "environment-switch", "alert-entry", "command-shell"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("top bar capabilities mismatch: got %+v want %+v", got, want)
	}
	if audit.TopBarAudit.ReleaseReady() {
		t.Fatalf("expected top bar to be non-release-ready")
	}
	if got, want := audit.SurfacesMissingStates, map[string][]string{"Queue": {"error"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("surfaces missing states mismatch: got %+v want %+v", got, want)
	}
	if got, want := audit.StatesMissingActions, map[string][]string{"Queue": {"loading"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("states missing actions mismatch: got %+v want %+v", got, want)
	}
	if got, want := audit.UnresolvedStateActions, map[string]map[string][]string{"Queue": {"empty": {"retry"}}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unresolved state actions mismatch: got %+v want %+v", got, want)
	}
	if got, want := audit.OrphanNavigationRoutes, []string{"/ghost"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("orphan navigation routes mismatch: got %+v want %+v", got, want)
	}
	if got, want := audit.UnnavigableSurfaces, []string{"Queue"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unnavigable surfaces mismatch: got %+v want %+v", got, want)
	}
	if got := audit.ReadinessScore(); got != 0.0 {
		t.Fatalf("readiness score mismatch: got %v want 0.0", got)
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
				Name:              "Incomplete Header",
				SearchPlaceholder: "",
				EnvironmentOptions: []string{
					"Production",
				},
				TimeRangeOptions:          []string{"24h"},
				DocumentationComplete:     false,
				AccessibilityRequirements: []string{"focus-visible"},
				CommandEntry: designsystem.ConsoleCommandEntry{
					TriggerLabel: "",
					Placeholder:  "",
					Shortcut:     "Cmd+K",
				},
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

	payload, err := json.Marshal(audit)
	if err != nil {
		t.Fatalf("marshal audit: %v", err)
	}
	var restored ConsoleIAAudit
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal audit: %v", err)
	}
	if !reflect.DeepEqual(restored, audit) {
		t.Fatalf("restored audit mismatch: got %+v want %+v", restored, audit)
	}
}

func TestRenderConsoleIAReportSummarizesSurfaceCoverage(t *testing.T) {
	architecture := ConsoleIA{
		Name:       "BigClaw Console IA",
		Version:    "v3",
		TopBar:     readyTopBar(),
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

	audit := ConsoleIAAuditor{}.Audit(architecture)
	report := RenderConsoleIAReport(architecture, audit)

	for _, needle := range []string{
		"# Console Information Architecture Report",
		"- Name: BigClaw Global Header",
		"- Release Ready: True",
		"- Navigation Items: 1",
		"- Overview: route=/overview filters=Team actions=Refresh states=default, loading, empty, error missing_states=none states_without_actions=none unresolved_state_actions=none",
		"- Surfaces missing filters: none",
		"- Undefined state actions: none",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("expected report to contain %q, got:\n%s", needle, report)
		}
	}
}

func TestConsoleInteractionDraftRoundTripPreservesFourPageManifest(t *testing.T) {
	draft := ConsoleInteractionDraft{
		Name:    "BIG-4203 Four Critical Pages",
		Version: "v1",
		Architecture: ConsoleIA{
			Name:       "BigClaw Console IA",
			Version:    "v3",
			TopBar:     readyTopBar(),
			Navigation: fourPageNavigation(),
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

	payload, err := json.Marshal(draft)
	if err != nil {
		t.Fatalf("marshal draft: %v", err)
	}
	var restored ConsoleInteractionDraft
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal draft: %v", err)
	}
	if !reflect.DeepEqual(restored, draft) {
		t.Fatalf("restored draft mismatch: got %+v want %+v", restored, draft)
	}
}

func TestConsoleInteractionAuditSurfacesMissingActionsPermissionsAndBatchOps(t *testing.T) {
	draft := ConsoleInteractionDraft{
		Name:    "BIG-4203 Four Critical Pages",
		Version: "v1",
		Architecture: ConsoleIA{
			Name:       "BigClaw Console IA",
			Version:    "v3",
			TopBar:     readyTopBar(),
			Navigation: fourPageNavigation(),
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
		},
		Contracts: []SurfaceInteractionContract{
			{
				SurfaceName:       "Overview",
				RequiredActionIDs: []string{"drill-down", "export", "audit"},
				RequiresFilters:   true,
				PermissionRule: SurfacePermissionRule{
					AllowedRoles: []string{"admin", "operator"},
					DeniedRoles:  []string{"viewer"},
					AuditEvent:   "overview.access.denied",
				},
			},
			{
				SurfaceName:          "Queue",
				RequiredActionIDs:    []string{"drill-down", "export", "audit"},
				RequiresFilters:      true,
				RequiresBatchActions: true,
				PermissionRule: SurfacePermissionRule{
					AllowedRoles: []string{"admin", "operator"},
					DeniedRoles:  []string{"viewer"},
				},
			},
			{
				SurfaceName:       "Run Detail",
				RequiredActionIDs: []string{"drill-down", "export", "audit"},
				RequiresFilters:   true,
				PermissionRule: SurfacePermissionRule{
					AllowedRoles: []string{"admin", "operator", "viewer"},
					DeniedRoles:  []string{},
					AuditEvent:   "run-detail.access.denied",
				},
			},
			{
				SurfaceName:          "Triage",
				RequiredActionIDs:    []string{"drill-down", "export", "audit"},
				RequiresFilters:      true,
				RequiresBatchActions: true,
				PermissionRule: SurfacePermissionRule{
					AllowedRoles: []string{"admin", "operator"},
					DeniedRoles:  []string{"viewer"},
					AuditEvent:   "triage.access.denied",
				},
			},
		},
	}

	audit := ConsoleInteractionAuditor{}.Audit(draft)

	if audit.Name != "BIG-4203 Four Critical Pages" || audit.Version != "v1" || audit.ContractCount != 4 {
		t.Fatalf("unexpected audit identity: %+v", audit)
	}
	if len(audit.MissingSurfaces) != 0 {
		t.Fatalf("expected no missing surfaces, got %+v", audit.MissingSurfaces)
	}
	if got, want := audit.SurfacesMissingFilters, []string{"Triage"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("surfaces missing filters mismatch: got %+v want %+v", got, want)
	}
	if got, want := audit.SurfacesMissingActions, map[string][]string{"Queue": {"export"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("surfaces missing actions mismatch: got %+v want %+v", got, want)
	}
	if got, want := audit.SurfacesMissingBatchActions, []string{"Queue"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("surfaces missing batch actions mismatch: got %+v want %+v", got, want)
	}
	if got, want := audit.SurfacesMissingStates, map[string][]string{"Queue": {"error"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("surfaces missing states mismatch: got %+v want %+v", got, want)
	}
	if got, want := audit.PermissionGaps, map[string][]string{"Queue": {"audit-event"}, "Run Detail": {"denied-roles"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("permission gaps mismatch: got %+v want %+v", got, want)
	}
	if got := audit.ReadinessScore(); got != 0.0 {
		t.Fatalf("readiness score mismatch: got %v want 0.0", got)
	}
	if audit.ReleaseReady() {
		t.Fatalf("expected audit to be non-release-ready")
	}
}

func TestRenderConsoleInteractionReportSummarizesCriticalPageContracts(t *testing.T) {
	draft := ConsoleInteractionDraft{
		Name:    "BIG-4203 Four Critical Pages",
		Version: "v1",
		Architecture: ConsoleIA{
			Name:       "BigClaw Console IA",
			Version:    "v3",
			TopBar:     readyTopBar(),
			Navigation: fourPageNavigation(),
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
						{Name: "empty", AllowedActions: []string{"drill-down"}},
						{Name: "error", AllowedActions: []string{"audit"}},
					},
				},
				{
					Name:              "Queue",
					Route:             "/queue",
					NavigationSection: "Operate",
					TopBarActions: []GlobalAction{
						{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"},
						{ActionID: "export", Label: "Export", Placement: "topbar"},
						{ActionID: "audit", Label: "Audit Trail", Placement: "topbar"},
						{ActionID: "bulk-approve", Label: "Bulk Approve", Placement: "topbar", RequiresSelection: true},
					},
					Filters: []FilterDefinition{{Name: "Status", Field: "status", Control: "select", Options: []string{"all"}}},
					States: []SurfaceState{
						{Name: "default"},
						{Name: "loading", AllowedActions: []string{"export"}},
						{Name: "empty", AllowedActions: []string{"audit"}},
						{Name: "error", AllowedActions: []string{"audit"}},
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
					Filters: []FilterDefinition{{Name: "Severity", Field: "severity", Control: "select", Options: []string{"all"}}},
					States: []SurfaceState{
						{Name: "default"},
						{Name: "loading", AllowedActions: []string{"export"}},
						{Name: "empty", AllowedActions: []string{"audit"}},
						{Name: "error", AllowedActions: []string{"audit"}},
					},
				},
			},
		},
		Contracts: []SurfaceInteractionContract{
			{
				SurfaceName:       "Overview",
				RequiredActionIDs: []string{"drill-down", "export", "audit"},
				RequiresFilters:   true,
				PermissionRule: SurfacePermissionRule{
					AllowedRoles: []string{"admin", "operator"},
					DeniedRoles:  []string{"viewer"},
					AuditEvent:   "overview.access.denied",
				},
			},
			{
				SurfaceName:          "Queue",
				RequiredActionIDs:    []string{"drill-down", "export", "audit"},
				RequiresFilters:      true,
				RequiresBatchActions: true,
				PermissionRule: SurfacePermissionRule{
					AllowedRoles: []string{"admin", "operator"},
					DeniedRoles:  []string{"viewer"},
					AuditEvent:   "queue.access.denied",
				},
			},
			{
				SurfaceName:       "Run Detail",
				RequiredActionIDs: []string{"drill-down", "export", "audit"},
				RequiresFilters:   true,
				PermissionRule: SurfacePermissionRule{
					AllowedRoles: []string{"admin", "operator", "viewer"},
					DeniedRoles:  []string{"guest"},
					AuditEvent:   "run-detail.access.denied",
				},
			},
			{
				SurfaceName:          "Triage",
				RequiredActionIDs:    []string{"drill-down", "export", "audit"},
				RequiresFilters:      true,
				RequiresBatchActions: true,
				PermissionRule: SurfacePermissionRule{
					AllowedRoles: []string{"admin", "operator"},
					DeniedRoles:  []string{"viewer"},
					AuditEvent:   "triage.access.denied",
				},
			},
		},
	}

	audit := ConsoleInteractionAuditor{}.Audit(draft)
	report := RenderConsoleInteractionReport(draft, audit)

	for _, needle := range []string{
		"# Console Interaction Draft Report",
		"- Critical Pages: 4",
		"- Required Roles: none",
		"- Readiness Score: 100.0",
		"- Release Ready: True",
		"- Overview: route=/overview required_actions=drill-down, export, audit available_actions=drill-down, export, audit filters=1 states=default, loading, empty, error batch=optional permissions=complete",
		"- Queue: route=/queue required_actions=drill-down, export, audit available_actions=drill-down, export, audit, bulk-approve filters=1 states=default, loading, empty, error batch=required permissions=complete",
		"- Permission gaps: none",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("expected report to contain %q, got:\n%s", needle, report)
		}
	}
}

func TestBuildBig4203ConsoleInteractionDraftIsReleaseReady(t *testing.T) {
	draft := BuildBig4203ConsoleInteractionDraft()

	audit := ConsoleInteractionAuditor{}.Audit(draft)
	report := RenderConsoleInteractionReport(draft, audit)

	if got, want := draft.RequiredRoles, []string{"eng-lead", "platform-admin", "vp-eng", "cross-team-operator"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("required roles mismatch: got %+v want %+v", got, want)
	}
	if !draft.RequiresFrameContracts {
		t.Fatalf("expected frame contracts to be required")
	}
	if !audit.ReleaseReady() {
		t.Fatalf("expected audit to be release ready, got %+v", audit)
	}
	if len(audit.UncoveredRoles) != 0 {
		t.Fatalf("expected no uncovered roles, got %+v", audit.UncoveredRoles)
	}
	for _, needle := range []string{
		"- Required Roles: eng-lead, platform-admin, vp-eng, cross-team-operator",
		"persona=VP Eng wireframe=wf-overview",
		"review_focus=metric hierarchy,drill-down posture,alert prioritization",
		"- Uncovered roles: none",
		"- Pages missing personas: none",
		"- Pages missing wireframe links: none",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("expected report to contain %q, got:\n%s", needle, report)
		}
	}
}

func TestConsoleInteractionAuditFlagsUncoveredRequiredRoles(t *testing.T) {
	draft := BuildBig4203ConsoleInteractionDraft()
	draft.RequiredRoles = append(draft.RequiredRoles, "finance-reviewer")

	audit := ConsoleInteractionAuditor{}.Audit(draft)

	if got, want := audit.UncoveredRoles, []string{"finance-reviewer"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("uncovered roles mismatch: got %+v want %+v", got, want)
	}
	if audit.ReleaseReady() {
		t.Fatalf("expected audit to be non-release-ready")
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
		t.Fatalf("missing personas mismatch: got %+v want %+v", got, want)
	}
	if got, want := audit.SurfacesMissingWireframeLinks, []string{"Overview"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("missing wireframe links mismatch: got %+v want %+v", got, want)
	}
	if got, want := audit.SurfacesMissingReviewFocus, []string{"Overview"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("missing review focus mismatch: got %+v want %+v", got, want)
	}
	if got, want := audit.SurfacesMissingDecisionPrompts, []string{"Overview"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("missing decision prompts mismatch: got %+v want %+v", got, want)
	}
	if audit.ReleaseReady() {
		t.Fatalf("expected audit to be non-release-ready")
	}
}

func readyTopBar() designsystem.ConsoleTopBar {
	return designsystem.ConsoleTopBar{
		Name:              "BigClaw Global Header",
		SearchPlaceholder: "Search runs, issues, commands",
		EnvironmentOptions: []string{
			"Production",
			"Staging",
		},
		TimeRangeOptions:      []string{"24h", "7d"},
		AlertChannels:         []string{"approvals"},
		DocumentationComplete: true,
		AccessibilityRequirements: []string{
			"keyboard-navigation",
			"screen-reader-label",
			"focus-visible",
		},
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

func fourPageNavigation() []NavigationItem {
	return []NavigationItem{
		{Name: "Overview", Route: "/overview", Section: "Operate"},
		{Name: "Queue", Route: "/queue", Section: "Operate"},
		{Name: "Run Detail", Route: "/runs/detail", Section: "Operate"},
		{Name: "Triage", Route: "/triage", Section: "Operate"},
	}
}
