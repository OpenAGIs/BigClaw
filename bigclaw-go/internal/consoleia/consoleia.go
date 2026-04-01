package consoleia

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"bigclaw-go/internal/designsystem"
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
	Placement         string `json:"placement,omitempty"`
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
	Name       string                     `json:"name"`
	Version    string                     `json:"version"`
	TopBar     designsystem.ConsoleTopBar `json:"top_bar"`
	Navigation []NavigationItem           `json:"navigation,omitempty"`
	Surfaces   []ConsoleSurface           `json:"surfaces,omitempty"`
}

type ConsoleIAAudit struct {
	SystemName             string                          `json:"system_name"`
	Version                string                          `json:"version"`
	SurfaceCount           int                             `json:"surface_count"`
	NavigationCount        int                             `json:"navigation_count"`
	TopBarAudit            designsystem.ConsoleTopBarAudit `json:"top_bar_audit"`
	SurfacesMissingFilters []string                        `json:"surfaces_missing_filters,omitempty"`
	SurfacesMissingActions []string                        `json:"surfaces_missing_actions,omitempty"`
	SurfacesMissingStates  map[string][]string             `json:"surfaces_missing_states,omitempty"`
	StatesMissingActions   map[string][]string             `json:"states_missing_actions,omitempty"`
	UnresolvedStateActions map[string]map[string][]string  `json:"unresolved_state_actions,omitempty"`
	OrphanNavigationRoutes []string                        `json:"orphan_navigation_routes,omitempty"`
	UnnavigableSurfaces    []string                        `json:"unnavigable_surfaces,omitempty"`
	ReadinessScore         float64                         `json:"readiness_score"`
}

type ConsoleIAAuditor struct{}

func (ConsoleIAAuditor) Audit(architecture ConsoleIA) ConsoleIAAudit {
	audit := ConsoleIAAudit{
		SystemName:             architecture.Name,
		Version:                architecture.Version,
		SurfaceCount:           len(architecture.Surfaces),
		NavigationCount:        len(architecture.Navigation),
		TopBarAudit:            designsystem.ConsoleTopBarAuditor{}.Audit(architecture.TopBar),
		SurfacesMissingStates:  make(map[string][]string),
		StatesMissingActions:   make(map[string][]string),
		UnresolvedStateActions: make(map[string]map[string][]string),
	}
	navRoutes := make(map[string]struct{}, len(architecture.Navigation))
	for _, item := range architecture.Navigation {
		navRoutes[item.Route] = struct{}{}
	}
	surfaceRoutes := make(map[string]struct{}, len(architecture.Surfaces))
	for _, surface := range architecture.Surfaces {
		surfaceRoutes[surface.Route] = struct{}{}
		if len(surface.Filters) == 0 {
			audit.SurfacesMissingFilters = append(audit.SurfacesMissingFilters, surface.Name)
		}
		if len(surface.TopBarActions) == 0 {
			audit.SurfacesMissingActions = append(audit.SurfacesMissingActions, surface.Name)
		}
		requiredStates := []string{"default", "loading", "empty", "error"}
		stateMap := make(map[string]SurfaceState, len(surface.States))
		for _, state := range surface.States {
			stateMap[state.Name] = state
			if state.Name == "loading" && len(state.AllowedActions) == 0 {
				audit.StatesMissingActions[surface.Name] = append(audit.StatesMissingActions[surface.Name], state.Name)
			}
			unresolved := unresolvedActions(surface.TopBarActions, state.AllowedActions)
			if len(unresolved) > 0 {
				if audit.UnresolvedStateActions[surface.Name] == nil {
					audit.UnresolvedStateActions[surface.Name] = make(map[string][]string)
				}
				audit.UnresolvedStateActions[surface.Name][state.Name] = unresolved
			}
		}
		for _, required := range requiredStates {
			if _, ok := stateMap[required]; !ok {
				audit.SurfacesMissingStates[surface.Name] = append(audit.SurfacesMissingStates[surface.Name], required)
			}
		}
		if _, ok := navRoutes[surface.Route]; !ok {
			audit.UnnavigableSurfaces = append(audit.UnnavigableSurfaces, surface.Name)
		}
	}
	for _, item := range architecture.Navigation {
		if _, ok := surfaceRoutes[item.Route]; !ok {
			audit.OrphanNavigationRoutes = append(audit.OrphanNavigationRoutes, item.Route)
		}
	}
	cleanupConsoleIAAudit(&audit)
	return audit
}

type SurfacePermissionRule struct {
	AllowedRoles []string `json:"allowed_roles,omitempty"`
	DeniedRoles  []string `json:"denied_roles,omitempty"`
	AuditEvent   string   `json:"audit_event,omitempty"`
}

type SurfaceInteractionContract struct {
	SurfaceName          string                `json:"surface_name"`
	RequiredActionIDs    []string              `json:"required_action_ids,omitempty"`
	RequiresBatchActions bool                  `json:"requires_batch_actions"`
	RequiresFilters      bool                  `json:"requires_filters"`
	PermissionRule       SurfacePermissionRule `json:"permission_rule"`
	PrimaryPersona       string                `json:"primary_persona,omitempty"`
	LinkedWireframeID    string                `json:"linked_wireframe_id,omitempty"`
	ReviewFocusAreas     []string              `json:"review_focus_areas,omitempty"`
	DecisionPrompts      []string              `json:"decision_prompts,omitempty"`
}

type ConsoleInteractionDraft struct {
	Name                   string                       `json:"name"`
	Version                string                       `json:"version"`
	Architecture           ConsoleIA                    `json:"architecture"`
	Contracts              []SurfaceInteractionContract `json:"contracts,omitempty"`
	RequiredRoles          []string                     `json:"required_roles,omitempty"`
	RequiresFrameContracts bool                         `json:"requires_frame_contracts"`
}

type ConsoleInteractionAudit struct {
	Name                           string              `json:"name"`
	Version                        string              `json:"version"`
	ContractCount                  int                 `json:"contract_count"`
	MissingSurfaces                []string            `json:"missing_surfaces,omitempty"`
	SurfacesMissingFilters         []string            `json:"surfaces_missing_filters,omitempty"`
	SurfacesMissingActions         map[string][]string `json:"surfaces_missing_actions,omitempty"`
	SurfacesMissingBatchActions    []string            `json:"surfaces_missing_batch_actions,omitempty"`
	SurfacesMissingStates          map[string][]string `json:"surfaces_missing_states,omitempty"`
	PermissionGaps                 map[string][]string `json:"permission_gaps,omitempty"`
	UncoveredRoles                 []string            `json:"uncovered_roles,omitempty"`
	SurfacesMissingPrimaryPersonas []string            `json:"surfaces_missing_primary_personas,omitempty"`
	SurfacesMissingWireframeLinks  []string            `json:"surfaces_missing_wireframe_links,omitempty"`
	SurfacesMissingReviewFocus     []string            `json:"surfaces_missing_review_focus,omitempty"`
	SurfacesMissingDecisionPrompts []string            `json:"surfaces_missing_decision_prompts,omitempty"`
	ReadinessScore                 float64             `json:"readiness_score"`
	ReleaseReady                   bool                `json:"release_ready"`
}

type ConsoleInteractionAuditor struct{}

func (ConsoleInteractionAuditor) Audit(draft ConsoleInteractionDraft) ConsoleInteractionAudit {
	audit := ConsoleInteractionAudit{
		Name:                   draft.Name,
		Version:                draft.Version,
		ContractCount:          len(draft.Contracts),
		SurfacesMissingActions: make(map[string][]string),
		SurfacesMissingStates:  make(map[string][]string),
		PermissionGaps:         make(map[string][]string),
	}
	surfaceIndex := make(map[string]ConsoleSurface, len(draft.Architecture.Surfaces))
	for _, surface := range draft.Architecture.Surfaces {
		surfaceIndex[surface.Name] = surface
	}
	coveredRoles := make(map[string]struct{})
	for _, contract := range draft.Contracts {
		surface, ok := surfaceIndex[contract.SurfaceName]
		if !ok {
			audit.MissingSurfaces = append(audit.MissingSurfaces, contract.SurfaceName)
			continue
		}
		for _, role := range contract.PermissionRule.AllowedRoles {
			coveredRoles[role] = struct{}{}
		}
		if contract.RequiresFilters && len(surface.Filters) == 0 {
			audit.SurfacesMissingFilters = append(audit.SurfacesMissingFilters, surface.Name)
		}
		missingActions := missingRequiredActions(surface.TopBarActions, contract.RequiredActionIDs)
		if len(missingActions) > 0 {
			audit.SurfacesMissingActions[surface.Name] = missingActions
		}
		if contract.RequiresBatchActions && !hasBatchAction(surface.TopBarActions) {
			audit.SurfacesMissingBatchActions = append(audit.SurfacesMissingBatchActions, surface.Name)
		}
		if missingStates := missingSurfaceStates(surface.States); len(missingStates) > 0 {
			audit.SurfacesMissingStates[surface.Name] = missingStates
		}
		if strings.TrimSpace(contract.PermissionRule.AuditEvent) == "" {
			audit.PermissionGaps[surface.Name] = append(audit.PermissionGaps[surface.Name], "audit-event")
		}
		if len(contract.PermissionRule.DeniedRoles) == 0 {
			audit.PermissionGaps[surface.Name] = append(audit.PermissionGaps[surface.Name], "denied-roles")
		}
		if draft.RequiresFrameContracts {
			if strings.TrimSpace(contract.PrimaryPersona) == "" {
				audit.SurfacesMissingPrimaryPersonas = append(audit.SurfacesMissingPrimaryPersonas, surface.Name)
			}
			if strings.TrimSpace(contract.LinkedWireframeID) == "" {
				audit.SurfacesMissingWireframeLinks = append(audit.SurfacesMissingWireframeLinks, surface.Name)
			}
			if len(contract.ReviewFocusAreas) == 0 {
				audit.SurfacesMissingReviewFocus = append(audit.SurfacesMissingReviewFocus, surface.Name)
			}
			if len(contract.DecisionPrompts) == 0 {
				audit.SurfacesMissingDecisionPrompts = append(audit.SurfacesMissingDecisionPrompts, surface.Name)
			}
		}
	}
	for _, role := range draft.RequiredRoles {
		if _, ok := coveredRoles[role]; !ok {
			audit.UncoveredRoles = append(audit.UncoveredRoles, role)
		}
	}
	cleanupInteractionAudit(&audit)
	if interactionAuditHasNoGaps(audit) {
		audit.ReadinessScore = 100.0
		audit.ReleaseReady = true
	} else {
		audit.ReadinessScore = 0.0
		audit.ReleaseReady = false
	}
	return audit
}

func RenderConsoleIAReport(architecture ConsoleIA, audit ConsoleIAAudit) string {
	lines := []string{
		"# Console Information Architecture Report",
		"",
		fmt.Sprintf("- Name: %s", architecture.TopBar.Name),
		fmt.Sprintf("- Release Ready: %t", audit.TopBarAudit.ReleaseReady && audit.ReadinessScore == 100.0),
		fmt.Sprintf("- Navigation Items: %d", len(architecture.Navigation)),
	}
	for _, surface := range architecture.Surfaces {
		lines = append(lines, fmt.Sprintf(
			"- %s: route=%s filters=%s actions=%s states=%s missing_states=%s states_without_actions=%s unresolved_state_actions=%s",
			surface.Name,
			surface.Route,
			filterNames(surface.Filters),
			actionLabels(surface.TopBarActions),
			stateNames(surface.States),
			joinOrNone(audit.SurfacesMissingStates[surface.Name]),
			joinOrNone(audit.StatesMissingActions[surface.Name]),
			formatUnresolvedStates(audit.UnresolvedStateActions[surface.Name]),
		))
	}
	lines = append(lines, fmt.Sprintf("- Surfaces missing filters: %s", joinOrNone(audit.SurfacesMissingFilters)))
	lines = append(lines, fmt.Sprintf("- Undefined state actions: %s", formatAllUnresolved(audit.UnresolvedStateActions)))
	return strings.Join(lines, "\n") + "\n"
}

func RenderConsoleInteractionReport(draft ConsoleInteractionDraft, audit ConsoleInteractionAudit) string {
	lines := []string{
		"# Console Interaction Draft Report",
		"",
		fmt.Sprintf("- Critical Pages: %d", len(draft.Contracts)),
		fmt.Sprintf("- Required Roles: %s", joinOrNone(draft.RequiredRoles)),
		fmt.Sprintf("- Readiness Score: %.1f", audit.ReadinessScore),
		fmt.Sprintf("- Release Ready: %t", audit.ReleaseReady),
	}
	for _, contract := range draft.Contracts {
		surface, ok := findSurface(draft.Architecture.Surfaces, contract.SurfaceName)
		if !ok {
			continue
		}
		lines = append(lines, fmt.Sprintf(
			"- %s: route=%s required_actions=%s available_actions=%s filters=%d states=%s batch=%s permissions=%s",
			contract.SurfaceName,
			surface.Route,
			joinOrNone(contract.RequiredActionIDs),
			actionIDs(surface.TopBarActions),
			len(surface.Filters),
			stateNames(surface.States),
			batchText(contract.RequiresBatchActions),
			permissionStatus(audit.PermissionGaps[contract.SurfaceName]),
		))
		if draft.RequiresFrameContracts {
			lines = append(lines, fmt.Sprintf(
				"  persona=%s wireframe=%s review_focus=%s",
				firstNonEmpty(contract.PrimaryPersona, "none"),
				firstNonEmpty(contract.LinkedWireframeID, "none"),
				joinOrNone(contract.ReviewFocusAreas),
			))
		}
	}
	lines = append(lines, fmt.Sprintf("- Permission gaps: %s", formatPermissionGaps(audit.PermissionGaps)))
	if draft.RequiresFrameContracts {
		lines = append(lines, fmt.Sprintf("- Uncovered roles: %s", joinOrNone(audit.UncoveredRoles)))
		lines = append(lines, fmt.Sprintf("- Pages missing personas: %s", joinOrNone(audit.SurfacesMissingPrimaryPersonas)))
		lines = append(lines, fmt.Sprintf("- Pages missing wireframe links: %s", joinOrNone(audit.SurfacesMissingWireframeLinks)))
	}
	return strings.Join(lines, "\n") + "\n"
}

func BuildBIG4203ConsoleInteractionDraft() ConsoleInteractionDraft {
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
		Navigation: []NavigationItem{
			{Name: "Overview", Route: "/overview", Section: "Operate"},
			{Name: "Queue", Route: "/queue", Section: "Operate"},
			{Name: "Run Detail", Route: "/runs/detail", Section: "Operate"},
			{Name: "Triage", Route: "/triage", Section: "Operate"},
		},
		Surfaces: []ConsoleSurface{
			{Name: "Overview", Route: "/overview", NavigationSection: "Operate", TopBarActions: []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}}, Filters: []FilterDefinition{{Name: "Team", Field: "team", Control: "select", Options: []string{"all"}}}, States: standardStates("drill-down")},
			{Name: "Queue", Route: "/queue", NavigationSection: "Operate", TopBarActions: []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}, {ActionID: "bulk-approve", Label: "Bulk Approve", Placement: "topbar", RequiresSelection: true}}, Filters: []FilterDefinition{{Name: "Status", Field: "status", Control: "select", Options: []string{"all"}}}, States: standardStates("audit")},
			{Name: "Run Detail", Route: "/runs/detail", NavigationSection: "Operate", TopBarActions: []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}}, Filters: []FilterDefinition{{Name: "Run", Field: "run_id", Control: "search"}}, States: standardStates("drill-down")},
			{Name: "Triage", Route: "/triage", NavigationSection: "Operate", TopBarActions: []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}, {ActionID: "bulk-assign", Label: "Bulk Assign", Placement: "topbar", RequiresSelection: true}}, Filters: []FilterDefinition{{Name: "Severity", Field: "severity", Control: "select", Options: []string{"all"}}}, States: standardStates("audit")},
		},
	}
	return ConsoleInteractionDraft{
		Name:                   "BIG-4203 Four Critical Pages",
		Version:                "v1",
		Architecture:           architecture,
		RequiredRoles:          []string{"eng-lead", "platform-admin", "vp-eng", "cross-team-operator"},
		RequiresFrameContracts: true,
		Contracts: []SurfaceInteractionContract{
			{SurfaceName: "Overview", RequiredActionIDs: []string{"drill-down", "export", "audit"}, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"eng-lead", "platform-admin", "vp-eng"}, DeniedRoles: []string{"viewer"}, AuditEvent: "overview.access.denied"}, PrimaryPersona: "VP Eng", LinkedWireframeID: "wf-overview", ReviewFocusAreas: []string{"metric hierarchy", "drill-down posture", "alert prioritization"}, DecisionPrompts: []string{"Is the executive posture obvious?"}},
			{SurfaceName: "Queue", RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiresBatchActions: true, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"eng-lead", "platform-admin", "cross-team-operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "queue.access.denied"}, PrimaryPersona: "Cross-Team Operator", LinkedWireframeID: "wf-queue", ReviewFocusAreas: []string{"queue ownership", "batch approval safety"}, DecisionPrompts: []string{"Are bulk actions bounded?"}},
			{SurfaceName: "Run Detail", RequiredActionIDs: []string{"drill-down", "export", "audit"}, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"eng-lead", "platform-admin", "vp-eng"}, DeniedRoles: []string{"guest"}, AuditEvent: "run-detail.access.denied"}, PrimaryPersona: "Eng Lead", LinkedWireframeID: "wf-run-detail", ReviewFocusAreas: []string{"trace evidence", "closeout clarity"}, DecisionPrompts: []string{"Is validation evidence visible?"}},
			{SurfaceName: "Triage", RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiresBatchActions: true, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"platform-admin", "cross-team-operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "triage.access.denied"}, PrimaryPersona: "Platform Admin", LinkedWireframeID: "wf-triage", ReviewFocusAreas: []string{"ownership routing", "batch assignment posture"}, DecisionPrompts: []string{"Can operators resolve blocked work quickly?"}},
		},
	}
}

func standardStates(emptyAction string) []SurfaceState {
	return []SurfaceState{
		{Name: "default"},
		{Name: "loading", AllowedActions: []string{"export"}},
		{Name: "empty", AllowedActions: []string{emptyAction}},
		{Name: "error", AllowedActions: []string{"audit"}},
	}
}

func missingRequiredActions(actions []GlobalAction, required []string) []string {
	index := make(map[string]struct{}, len(actions))
	for _, action := range actions {
		index[action.ActionID] = struct{}{}
	}
	missing := make([]string, 0)
	for _, requiredAction := range required {
		if _, ok := index[requiredAction]; !ok {
			missing = append(missing, requiredAction)
		}
	}
	return missing
}

func missingSurfaceStates(states []SurfaceState) []string {
	required := []string{"default", "loading", "empty", "error"}
	index := make(map[string]struct{}, len(states))
	for _, state := range states {
		index[state.Name] = struct{}{}
	}
	missing := make([]string, 0)
	for _, state := range required {
		if _, ok := index[state]; !ok {
			missing = append(missing, state)
		}
	}
	return missing
}

func unresolvedActions(actions []GlobalAction, allowed []string) []string {
	index := make(map[string]struct{}, len(actions))
	for _, action := range actions {
		index[action.ActionID] = struct{}{}
	}
	unresolved := make([]string, 0)
	for _, item := range allowed {
		if _, ok := index[item]; !ok {
			unresolved = append(unresolved, item)
		}
	}
	return unresolved
}

func hasBatchAction(actions []GlobalAction) bool {
	for _, action := range actions {
		if action.RequiresSelection {
			return true
		}
	}
	return false
}

func cleanupConsoleIAAudit(audit *ConsoleIAAudit) {
	sort.Strings(audit.SurfacesMissingFilters)
	sort.Strings(audit.SurfacesMissingActions)
	sort.Strings(audit.OrphanNavigationRoutes)
	sort.Strings(audit.UnnavigableSurfaces)
	if len(audit.SurfacesMissingStates) == 0 {
		audit.SurfacesMissingStates = nil
	}
	if len(audit.StatesMissingActions) == 0 {
		audit.StatesMissingActions = nil
	}
	if len(audit.UnresolvedStateActions) == 0 {
		audit.UnresolvedStateActions = nil
	}
	if len(audit.SurfacesMissingFilters) == 0 && len(audit.SurfacesMissingActions) == 0 && len(audit.SurfacesMissingStates) == 0 && len(audit.StatesMissingActions) == 0 && len(audit.UnresolvedStateActions) == 0 && len(audit.OrphanNavigationRoutes) == 0 && len(audit.UnnavigableSurfaces) == 0 && audit.TopBarAudit.ReleaseReady {
		audit.ReadinessScore = 100.0
	} else {
		audit.ReadinessScore = 0.0
	}
}

func cleanupInteractionAudit(audit *ConsoleInteractionAudit) {
	sort.Strings(audit.MissingSurfaces)
	sort.Strings(audit.SurfacesMissingFilters)
	sort.Strings(audit.SurfacesMissingBatchActions)
	sort.Strings(audit.UncoveredRoles)
	sort.Strings(audit.SurfacesMissingPrimaryPersonas)
	sort.Strings(audit.SurfacesMissingWireframeLinks)
	sort.Strings(audit.SurfacesMissingReviewFocus)
	sort.Strings(audit.SurfacesMissingDecisionPrompts)
	if len(audit.SurfacesMissingActions) == 0 {
		audit.SurfacesMissingActions = nil
	}
	if len(audit.SurfacesMissingStates) == 0 {
		audit.SurfacesMissingStates = nil
	}
	if len(audit.PermissionGaps) == 0 {
		audit.PermissionGaps = nil
	}
}

func interactionAuditHasNoGaps(audit ConsoleInteractionAudit) bool {
	return len(audit.MissingSurfaces) == 0 &&
		len(audit.SurfacesMissingFilters) == 0 &&
		len(audit.SurfacesMissingActions) == 0 &&
		len(audit.SurfacesMissingBatchActions) == 0 &&
		len(audit.SurfacesMissingStates) == 0 &&
		len(audit.PermissionGaps) == 0 &&
		len(audit.UncoveredRoles) == 0 &&
		len(audit.SurfacesMissingPrimaryPersonas) == 0 &&
		len(audit.SurfacesMissingWireframeLinks) == 0 &&
		len(audit.SurfacesMissingReviewFocus) == 0 &&
		len(audit.SurfacesMissingDecisionPrompts) == 0
}

func filterNames(filters []FilterDefinition) string {
	names := make([]string, 0, len(filters))
	for _, filter := range filters {
		names = append(names, filter.Name)
	}
	return joinOrNone(names)
}

func actionLabels(actions []GlobalAction) string {
	labels := make([]string, 0, len(actions))
	for _, action := range actions {
		labels = append(labels, action.Label)
	}
	return joinOrNone(labels)
}

func actionIDs(actions []GlobalAction) string {
	ids := make([]string, 0, len(actions))
	for _, action := range actions {
		ids = append(ids, action.ActionID)
	}
	return joinOrNone(ids)
}

func stateNames(states []SurfaceState) string {
	names := make([]string, 0, len(states))
	for _, state := range states {
		names = append(names, state.Name)
	}
	return joinOrNone(names)
}

func formatUnresolvedStates(items map[string][]string) string {
	if len(items) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, strings.Join(items[key], ", ")))
	}
	return strings.Join(parts, "; ")
}

func formatAllUnresolved(items map[string]map[string][]string) string {
	if len(items) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s[%s]", key, formatUnresolvedStates(items[key])))
	}
	return strings.Join(parts, "; ")
}

func formatPermissionGaps(items map[string][]string) string {
	if len(items) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, strings.Join(items[key], ", ")))
	}
	return strings.Join(parts, "; ")
}

func permissionStatus(gaps []string) string {
	if len(gaps) == 0 {
		return "complete"
	}
	return "gapped"
}

func batchText(required bool) string {
	if required {
		return "required"
	}
	return "optional"
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func firstNonEmpty(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func findSurface(surfaces []ConsoleSurface, name string) (ConsoleSurface, bool) {
	for _, surface := range surfaces {
		if surface.Name == name {
			return surface, true
		}
	}
	return ConsoleSurface{}, false
}

func roundTrip[T any](value T) (T, error) {
	var restored T
	data, err := json.Marshal(value)
	if err != nil {
		return restored, err
	}
	if err := json.Unmarshal(data, &restored); err != nil {
		return restored, err
	}
	return restored, nil
}
