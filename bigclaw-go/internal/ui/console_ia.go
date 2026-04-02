package ui

import (
	"fmt"
	"sort"
	"strings"
)

type NavigationItem struct {
	Name       string `json:"name"`
	Route      string `json:"route"`
	Section    string `json:"section"`
	Icon       string `json:"icon,omitempty"`
	BadgeCount int    `json:"badge_count,omitempty"`
}

type GlobalAction struct {
	ActionID          string `json:"action_id"`
	Label             string `json:"label"`
	Placement         string `json:"placement"`
	RequiresSelection bool   `json:"requires_selection,omitempty"`
}

type FilterDefinition struct {
	Name         string   `json:"name"`
	Field        string   `json:"field"`
	Control      string   `json:"control"`
	Options      []string `json:"options,omitempty"`
	DefaultValue string   `json:"default_value,omitempty"`
}

type SurfaceState struct {
	Name           string   `json:"name"`
	AllowedActions []string `json:"allowed_actions,omitempty"`
}

type ConsoleSurface struct {
	Name              string             `json:"name"`
	Route             string             `json:"route"`
	NavigationSection string             `json:"navigation_section"`
	TopBarActions     []GlobalAction     `json:"top_bar_actions,omitempty"`
	Filters           []FilterDefinition `json:"filters,omitempty"`
	States            []SurfaceState     `json:"states,omitempty"`
}

type ConsoleIA struct {
	Name       string           `json:"name"`
	Version    string           `json:"version"`
	TopBar     ConsoleTopBar    `json:"top_bar"`
	Navigation []NavigationItem `json:"navigation"`
	Surfaces   []ConsoleSurface `json:"surfaces"`
}

type ConsoleIAAudit struct {
	SystemName             string                         `json:"system_name"`
	Version                string                         `json:"version"`
	SurfaceCount           int                            `json:"surface_count"`
	NavigationCount        int                            `json:"navigation_count"`
	TopBarAudit            ConsoleTopBarAudit             `json:"top_bar_audit"`
	SurfacesMissingFilters []string                       `json:"surfaces_missing_filters"`
	SurfacesMissingActions []string                       `json:"surfaces_missing_actions"`
	SurfacesMissingStates  map[string][]string            `json:"surfaces_missing_states"`
	StatesMissingActions   map[string][]string            `json:"states_missing_actions"`
	UnresolvedStateActions map[string]map[string][]string `json:"unresolved_state_actions"`
	OrphanNavigationRoutes []string                       `json:"orphan_navigation_routes"`
	UnnavigableSurfaces    []string                       `json:"unnavigable_surfaces"`
	ReadinessScore         float64                        `json:"readiness_score"`
}

type ConsoleIAAuditor struct{}

func (ConsoleIAAuditor) Audit(architecture ConsoleIA) ConsoleIAAudit {
	topBarAudit := ConsoleChromeLibrary{}.AuditTopBar(architecture.TopBar)
	navigationRoutes := map[string]struct{}{}
	for _, item := range architecture.Navigation {
		navigationRoutes[normalizePath(item.Route)] = struct{}{}
	}

	surfacesByRoute := map[string]ConsoleSurface{}
	for _, surface := range architecture.Surfaces {
		surfacesByRoute[normalizePath(surface.Route)] = surface
	}

	missingFilters := make([]string, 0)
	missingActions := make([]string, 0)
	missingStates := map[string][]string{}
	statesMissingActions := map[string][]string{}
	unresolvedStateActions := map[string]map[string][]string{}
	for _, surface := range architecture.Surfaces {
		if len(surface.Filters) == 0 {
			missingFilters = append(missingFilters, surface.Name)
		}
		if len(surface.TopBarActions) == 0 {
			missingActions = append(missingActions, surface.Name)
		}

		actionIDs := map[string]struct{}{}
		for _, action := range surface.TopBarActions {
			actionIDs[action.ActionID] = struct{}{}
		}

		stateNames := map[string]SurfaceState{}
		for _, state := range surface.States {
			stateNames[strings.TrimSpace(state.Name)] = state
		}

		requiredStates := []string{"default", "loading", "empty", "error"}
		componentMissingStates := make([]string, 0)
		componentStatesMissingActions := make([]string, 0)
		componentUnresolved := map[string][]string{}
		for _, required := range requiredStates {
			state, ok := stateNames[required]
			if !ok {
				componentMissingStates = append(componentMissingStates, required)
				continue
			}
			if required != "default" && len(state.AllowedActions) == 0 {
				componentStatesMissingActions = append(componentStatesMissingActions, required)
			}
			unresolved := make([]string, 0)
			for _, actionID := range state.AllowedActions {
				if _, ok := actionIDs[actionID]; !ok {
					unresolved = append(unresolved, actionID)
				}
			}
			if len(unresolved) > 0 {
				componentUnresolved[required] = unresolved
			}
		}
		if len(componentMissingStates) > 0 {
			missingStates[surface.Name] = componentMissingStates
		}
		if len(componentStatesMissingActions) > 0 {
			statesMissingActions[surface.Name] = componentStatesMissingActions
		}
		if len(componentUnresolved) > 0 {
			unresolvedStateActions[surface.Name] = componentUnresolved
		}
	}

	orphanNavigationRoutes := make([]string, 0)
	for _, item := range architecture.Navigation {
		route := normalizePath(item.Route)
		if _, ok := surfacesByRoute[route]; !ok {
			orphanNavigationRoutes = append(orphanNavigationRoutes, route)
		}
	}

	unnavigableSurfaces := make([]string, 0)
	for _, surface := range architecture.Surfaces {
		if _, ok := navigationRoutes[normalizePath(surface.Route)]; !ok {
			unnavigableSurfaces = append(unnavigableSurfaces, surface.Name)
		}
	}

	sort.Strings(missingFilters)
	sort.Strings(missingActions)
	sort.Strings(orphanNavigationRoutes)
	sort.Strings(unnavigableSurfaces)

	score := 100.0
	if !topBarAudit.ReleaseReady() || len(missingFilters) > 0 || len(missingActions) > 0 || len(missingStates) > 0 || len(statesMissingActions) > 0 || len(unresolvedStateActions) > 0 || len(orphanNavigationRoutes) > 0 || len(unnavigableSurfaces) > 0 {
		score = 0.0
	}

	return ConsoleIAAudit{
		SystemName:             architecture.Name,
		Version:                architecture.Version,
		SurfaceCount:           len(architecture.Surfaces),
		NavigationCount:        len(architecture.Navigation),
		TopBarAudit:            topBarAudit,
		SurfacesMissingFilters: missingFilters,
		SurfacesMissingActions: missingActions,
		SurfacesMissingStates:  missingStates,
		StatesMissingActions:   statesMissingActions,
		UnresolvedStateActions: unresolvedStateActions,
		OrphanNavigationRoutes: orphanNavigationRoutes,
		UnnavigableSurfaces:    unnavigableSurfaces,
		ReadinessScore:         score,
	}
}

type SurfacePermissionRule struct {
	AllowedRoles []string `json:"allowed_roles"`
	DeniedRoles  []string `json:"denied_roles"`
	AuditEvent   string   `json:"audit_event"`
}

type SurfaceInteractionContract struct {
	SurfaceName          string                `json:"surface_name"`
	RequiredActionIDs    []string              `json:"required_action_ids,omitempty"`
	RequiresBatchActions bool                  `json:"requires_batch_actions,omitempty"`
	RequiresFilters      bool                  `json:"requires_filters,omitempty"`
	PermissionRule       SurfacePermissionRule `json:"permission_rule,omitempty"`
	PrimaryPersona       string                `json:"primary_persona,omitempty"`
	LinkedWireframeID    string                `json:"linked_wireframe_id,omitempty"`
	ReviewFocusAreas     []string              `json:"review_focus_areas,omitempty"`
	DecisionPrompts      []string              `json:"decision_prompts,omitempty"`
}

type ConsoleInteractionDraft struct {
	Name                   string                       `json:"name"`
	Version                string                       `json:"version"`
	Architecture           ConsoleIA                    `json:"architecture"`
	Contracts              []SurfaceInteractionContract `json:"contracts"`
	RequiredRoles          []string                     `json:"required_roles,omitempty"`
	RequiresFrameContracts bool                         `json:"requires_frame_contracts,omitempty"`
}

type ConsoleInteractionAudit struct {
	Name                           string              `json:"name"`
	Version                        string              `json:"version"`
	ContractCount                  int                 `json:"contract_count"`
	MissingSurfaces                []string            `json:"missing_surfaces"`
	SurfacesMissingFilters         []string            `json:"surfaces_missing_filters"`
	SurfacesMissingActions         map[string][]string `json:"surfaces_missing_actions"`
	SurfacesMissingBatchActions    []string            `json:"surfaces_missing_batch_actions"`
	SurfacesMissingStates          map[string][]string `json:"surfaces_missing_states"`
	PermissionGaps                 map[string][]string `json:"permission_gaps"`
	UncoveredRoles                 []string            `json:"uncovered_roles"`
	SurfacesMissingPrimaryPersonas []string            `json:"surfaces_missing_primary_personas"`
	SurfacesMissingWireframeLinks  []string            `json:"surfaces_missing_wireframe_links"`
	SurfacesMissingReviewFocus     []string            `json:"surfaces_missing_review_focus"`
	SurfacesMissingDecisionPrompts []string            `json:"surfaces_missing_decision_prompts"`
	ReadinessScore                 float64             `json:"readiness_score"`
}

func (a ConsoleInteractionAudit) ReleaseReady() bool {
	return a.ReadinessScore == 100.0
}

type ConsoleInteractionAuditor struct{}

func (ConsoleInteractionAuditor) Audit(draft ConsoleInteractionDraft) ConsoleInteractionAudit {
	surfaces := map[string]ConsoleSurface{}
	for _, surface := range draft.Architecture.Surfaces {
		surfaces[surface.Name] = surface
	}

	missingSurfaces := make([]string, 0)
	surfacesMissingFilters := make([]string, 0)
	surfacesMissingActions := map[string][]string{}
	surfacesMissingBatchActions := make([]string, 0)
	surfacesMissingStates := map[string][]string{}
	permissionGaps := map[string][]string{}
	coveredRoles := map[string]struct{}{}
	surfacesMissingPrimaryPersonas := make([]string, 0)
	surfacesMissingWireframeLinks := make([]string, 0)
	surfacesMissingReviewFocus := make([]string, 0)
	surfacesMissingDecisionPrompts := make([]string, 0)

	for _, contract := range draft.Contracts {
		surface, ok := surfaces[contract.SurfaceName]
		if !ok {
			missingSurfaces = append(missingSurfaces, contract.SurfaceName)
			continue
		}

		actionIDs := surfaceActionIDs(surface.TopBarActions)
		missingActionIDs := make([]string, 0)
		for _, actionID := range contract.RequiredActionIDs {
			if _, ok := actionIDs[actionID]; !ok {
				missingActionIDs = append(missingActionIDs, actionID)
			}
		}
		if len(missingActionIDs) > 0 {
			surfacesMissingActions[contract.SurfaceName] = missingActionIDs
		}
		if contract.RequiresFilters && len(surface.Filters) == 0 {
			surfacesMissingFilters = append(surfacesMissingFilters, contract.SurfaceName)
		}
		if contract.RequiresBatchActions && !surfaceHasBatchAction(surface.TopBarActions) {
			surfacesMissingBatchActions = append(surfacesMissingBatchActions, contract.SurfaceName)
		}
		if missing := missingSurfaceStates(surface); len(missing) > 0 {
			surfacesMissingStates[contract.SurfaceName] = missing
		}

		gaps := make([]string, 0)
		if len(contract.PermissionRule.DeniedRoles) == 0 {
			gaps = append(gaps, "denied-roles")
		}
		if strings.TrimSpace(contract.PermissionRule.AuditEvent) == "" {
			gaps = append(gaps, "audit-event")
		}
		if len(gaps) > 0 {
			permissionGaps[contract.SurfaceName] = gaps
		}
		for _, role := range contract.PermissionRule.AllowedRoles {
			coveredRoles[role] = struct{}{}
		}

		if draft.RequiresFrameContracts {
			if strings.TrimSpace(contract.PrimaryPersona) == "" {
				surfacesMissingPrimaryPersonas = append(surfacesMissingPrimaryPersonas, contract.SurfaceName)
			}
			if strings.TrimSpace(contract.LinkedWireframeID) == "" {
				surfacesMissingWireframeLinks = append(surfacesMissingWireframeLinks, contract.SurfaceName)
			}
			if len(contract.ReviewFocusAreas) == 0 {
				surfacesMissingReviewFocus = append(surfacesMissingReviewFocus, contract.SurfaceName)
			}
			if len(contract.DecisionPrompts) == 0 {
				surfacesMissingDecisionPrompts = append(surfacesMissingDecisionPrompts, contract.SurfaceName)
			}
		}
	}

	uncoveredRoles := make([]string, 0)
	for _, role := range draft.RequiredRoles {
		if _, ok := coveredRoles[role]; !ok {
			uncoveredRoles = append(uncoveredRoles, role)
		}
	}

	sort.Strings(missingSurfaces)
	sort.Strings(surfacesMissingFilters)
	sort.Strings(surfacesMissingBatchActions)
	sort.Strings(uncoveredRoles)
	sort.Strings(surfacesMissingPrimaryPersonas)
	sort.Strings(surfacesMissingWireframeLinks)
	sort.Strings(surfacesMissingReviewFocus)
	sort.Strings(surfacesMissingDecisionPrompts)

	score := 100.0
	if len(missingSurfaces) > 0 || len(surfacesMissingFilters) > 0 || len(surfacesMissingActions) > 0 || len(surfacesMissingBatchActions) > 0 || len(surfacesMissingStates) > 0 || len(permissionGaps) > 0 || len(uncoveredRoles) > 0 || len(surfacesMissingPrimaryPersonas) > 0 || len(surfacesMissingWireframeLinks) > 0 || len(surfacesMissingReviewFocus) > 0 || len(surfacesMissingDecisionPrompts) > 0 {
		score = 0.0
	}

	return ConsoleInteractionAudit{
		Name:                           draft.Name,
		Version:                        draft.Version,
		ContractCount:                  len(draft.Contracts),
		MissingSurfaces:                missingSurfaces,
		SurfacesMissingFilters:         surfacesMissingFilters,
		SurfacesMissingActions:         surfacesMissingActions,
		SurfacesMissingBatchActions:    surfacesMissingBatchActions,
		SurfacesMissingStates:          surfacesMissingStates,
		PermissionGaps:                 permissionGaps,
		UncoveredRoles:                 uncoveredRoles,
		SurfacesMissingPrimaryPersonas: surfacesMissingPrimaryPersonas,
		SurfacesMissingWireframeLinks:  surfacesMissingWireframeLinks,
		SurfacesMissingReviewFocus:     surfacesMissingReviewFocus,
		SurfacesMissingDecisionPrompts: surfacesMissingDecisionPrompts,
		ReadinessScore:                 score,
	}
}

func RenderConsoleIAReport(architecture ConsoleIA, audit ConsoleIAAudit) string {
	lines := []string{
		"# Console Information Architecture Report",
		fmt.Sprintf("- Name: %s", architecture.TopBar.Name),
		fmt.Sprintf("- Release Ready: %s", boolWord(audit.ReadinessScore == 100.0)),
		fmt.Sprintf("- Navigation Items: %d", len(architecture.Navigation)),
	}
	for _, surface := range architecture.Surfaces {
		lines = append(lines, fmt.Sprintf("- %s: route=%s filters=%s actions=%s states=%s missing_states=%s states_without_actions=%s unresolved_state_actions=%s",
			surface.Name,
			normalizePath(surface.Route),
			renderFilterNames(surface.Filters),
			renderActionLabels(surface.TopBarActions),
			renderStateNames(surface.States),
			joinOrNone(audit.SurfacesMissingStates[surface.Name]),
			joinOrNone(audit.StatesMissingActions[surface.Name]),
			renderStateActionMap(audit.UnresolvedStateActions[surface.Name]),
		))
	}
	lines = append(lines, fmt.Sprintf("- Surfaces missing filters: %s", joinOrNone(audit.SurfacesMissingFilters)))
	lines = append(lines, fmt.Sprintf("- Undefined state actions: %s", renderNestedNestedMap(audit.UnresolvedStateActions)))
	return strings.Join(lines, "\n")
}

func RenderConsoleInteractionReport(draft ConsoleInteractionDraft, audit ConsoleInteractionAudit) string {
	lines := []string{
		"# Console Interaction Draft Report",
		fmt.Sprintf("- Critical Pages: %d", len(draft.Contracts)),
		fmt.Sprintf("- Required Roles: %s", joinOrNone(draft.RequiredRoles)),
		fmt.Sprintf("- Readiness Score: %.1f", audit.ReadinessScore),
		fmt.Sprintf("- Release Ready: %s", boolWord(audit.ReleaseReady())),
	}
	surfaces := map[string]ConsoleSurface{}
	for _, surface := range draft.Architecture.Surfaces {
		surfaces[surface.Name] = surface
	}
	for _, contract := range draft.Contracts {
		surface, ok := surfaces[contract.SurfaceName]
		if !ok {
			continue
		}
		permissionStatus := "complete"
		if gaps, ok := audit.PermissionGaps[contract.SurfaceName]; ok && len(gaps) > 0 {
			permissionStatus = strings.Join(gaps, ", ")
		}
		line := fmt.Sprintf("- %s: route=%s required_actions=%s available_actions=%s filters=%d states=%s batch=%s permissions=%s",
			contract.SurfaceName,
			normalizePath(surface.Route),
			joinOrNone(contract.RequiredActionIDs),
			renderActionIDs(surface.TopBarActions),
			len(surface.Filters),
			renderStateNames(surface.States),
			batchRequirement(contract.RequiresBatchActions),
			permissionStatus,
		)
		if draft.RequiresFrameContracts {
			line += fmt.Sprintf(" persona=%s wireframe=%s review_focus=%s", emptyFallback(contract.PrimaryPersona, "none"), emptyFallback(contract.LinkedWireframeID, "none"), joinOrNone(contract.ReviewFocusAreas))
		}
		lines = append(lines, line)
	}
	lines = append(lines, fmt.Sprintf("- Permission gaps: %s", renderStringSliceMap(audit.PermissionGaps)))
	lines = append(lines, fmt.Sprintf("- Uncovered roles: %s", joinOrNone(audit.UncoveredRoles)))
	lines = append(lines, fmt.Sprintf("- Pages missing personas: %s", joinOrNone(audit.SurfacesMissingPrimaryPersonas)))
	lines = append(lines, fmt.Sprintf("- Pages missing wireframe links: %s", joinOrNone(audit.SurfacesMissingWireframeLinks)))
	return strings.Join(lines, "\n")
}

func BuildBig4203ConsoleInteractionDraft() ConsoleInteractionDraft {
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
		Navigation: []NavigationItem{
			{Name: "Overview", Route: "/overview", Section: "Operate"},
			{Name: "Queue", Route: "/queue", Section: "Operate"},
			{Name: "Run Detail", Route: "/runs/detail", Section: "Operate"},
			{Name: "Triage", Route: "/triage", Section: "Operate"},
		},
		Surfaces: []ConsoleSurface{
			readySurface("Overview", "/overview", []FilterDefinition{{Name: "Team", Field: "team", Control: "select", Options: []string{"all"}}}, []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}}),
			readySurface("Queue", "/queue", []FilterDefinition{{Name: "Status", Field: "status", Control: "select", Options: []string{"all"}}}, []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}, {ActionID: "bulk-approve", Label: "Bulk Approve", Placement: "topbar", RequiresSelection: true}}),
			readySurface("Run Detail", "/runs/detail", []FilterDefinition{{Name: "Run", Field: "run_id", Control: "search"}}, []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}}),
			readySurface("Triage", "/triage", []FilterDefinition{{Name: "Severity", Field: "severity", Control: "select", Options: []string{"all"}}}, []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}, {ActionID: "bulk-assign", Label: "Bulk Assign", Placement: "topbar", RequiresSelection: true}}),
		},
	}

	return ConsoleInteractionDraft{
		Name:                   "BIG-4203 Four Critical Pages",
		Version:                "v1",
		Architecture:           architecture,
		RequiredRoles:          []string{"eng-lead", "platform-admin", "vp-eng", "cross-team-operator"},
		RequiresFrameContracts: true,
		Contracts: []SurfaceInteractionContract{
			{SurfaceName: "Overview", RequiredActionIDs: []string{"drill-down", "export", "audit"}, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"eng-lead", "vp-eng"}, DeniedRoles: []string{"viewer"}, AuditEvent: "overview.access.denied"}, PrimaryPersona: "VP Eng", LinkedWireframeID: "wf-overview", ReviewFocusAreas: []string{"metric hierarchy", "drill-down posture", "alert prioritization"}, DecisionPrompts: []string{"Does the KPI stack support exec triage?"}},
			{SurfaceName: "Queue", RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiresBatchActions: true, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"eng-lead", "cross-team-operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "queue.access.denied"}, PrimaryPersona: "Cross-Team Operator", LinkedWireframeID: "wf-queue", ReviewFocusAreas: []string{"queue ownership", "batch controls"}, DecisionPrompts: []string{"Is takeover control obvious?"}},
			{SurfaceName: "Run Detail", RequiredActionIDs: []string{"drill-down", "export", "audit"}, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"eng-lead", "platform-admin"}, DeniedRoles: []string{"guest"}, AuditEvent: "run-detail.access.denied"}, PrimaryPersona: "Platform Admin", LinkedWireframeID: "wf-run-detail", ReviewFocusAreas: []string{"audit trace", "artifact inspection"}, DecisionPrompts: []string{"Can evidence be reviewed without context switching?"}},
			{SurfaceName: "Triage", RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiresBatchActions: true, RequiresFilters: true, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"eng-lead", "cross-team-operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "triage.access.denied"}, PrimaryPersona: "Eng Lead", LinkedWireframeID: "wf-triage", ReviewFocusAreas: []string{"severity filtering", "batch assignment"}, DecisionPrompts: []string{"Can the lead reprioritize in one pass?"}},
		},
	}
}

func readySurface(name, route string, filters []FilterDefinition, actions []GlobalAction) ConsoleSurface {
	return ConsoleSurface{
		Name:              name,
		Route:             route,
		NavigationSection: "Operate",
		TopBarActions:     actions,
		Filters:           filters,
		States: []SurfaceState{
			{Name: "default"},
			{Name: "loading", AllowedActions: []string{"export"}},
			{Name: "empty", AllowedActions: []string{"audit"}},
			{Name: "error", AllowedActions: []string{"audit"}},
		},
	}
}

func surfaceActionIDs(actions []GlobalAction) map[string]struct{} {
	ids := map[string]struct{}{}
	for _, action := range actions {
		ids[action.ActionID] = struct{}{}
	}
	return ids
}

func surfaceHasBatchAction(actions []GlobalAction) bool {
	for _, action := range actions {
		if action.RequiresSelection {
			return true
		}
	}
	return false
}

func missingSurfaceStates(surface ConsoleSurface) []string {
	seen := map[string]struct{}{}
	for _, state := range surface.States {
		seen[state.Name] = struct{}{}
	}
	missing := make([]string, 0)
	for _, state := range []string{"default", "loading", "empty", "error"} {
		if _, ok := seen[state]; !ok {
			missing = append(missing, state)
		}
	}
	return missing
}

func renderFilterNames(filters []FilterDefinition) string {
	if len(filters) == 0 {
		return "none"
	}
	names := make([]string, 0, len(filters))
	for _, filter := range filters {
		names = append(names, filter.Name)
	}
	return strings.Join(names, ", ")
}

func renderActionLabels(actions []GlobalAction) string {
	if len(actions) == 0 {
		return "none"
	}
	labels := make([]string, 0, len(actions))
	for _, action := range actions {
		labels = append(labels, action.Label)
	}
	return strings.Join(labels, ", ")
}

func renderActionIDs(actions []GlobalAction) string {
	if len(actions) == 0 {
		return "none"
	}
	ids := make([]string, 0, len(actions))
	for _, action := range actions {
		ids = append(ids, action.ActionID)
	}
	return strings.Join(ids, ", ")
}

func renderStateNames(states []SurfaceState) string {
	if len(states) == 0 {
		return "none"
	}
	names := make([]string, 0, len(states))
	for _, state := range states {
		names = append(names, state.Name)
	}
	return strings.Join(names, ", ")
}

func renderStateActionMap(values map[string][]string) string {
	if len(values) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, strings.Join(values[key], ", ")))
	}
	return strings.Join(parts, "; ")
}

func renderNestedNestedMap(values map[string]map[string][]string) string {
	if len(values) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, renderStateActionMap(values[key])))
	}
	return strings.Join(parts, ", ")
}

func renderStringSliceMap(values map[string][]string) string {
	if len(values) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, strings.Join(values[key], ", ")))
	}
	return strings.Join(parts, "; ")
}

func batchRequirement(required bool) string {
	if required {
		return "required"
	}
	return "optional"
}

func emptyFallback(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
