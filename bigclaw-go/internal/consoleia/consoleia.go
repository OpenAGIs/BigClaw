package consoleia

import (
	"fmt"
	"slices"
	"strings"

	"bigclaw-go/internal/designsystem"
)

var requiredSurfaceStates = []string{"default", "loading", "empty", "error"}

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
}

type SurfacePermissionRule struct {
	AllowedRoles []string `json:"allowed_roles,omitempty"`
	DeniedRoles  []string `json:"denied_roles,omitempty"`
	AuditEvent   string   `json:"audit_event,omitempty"`
}

type SurfaceInteractionContract struct {
	SurfaceName          string                `json:"surface_name"`
	RequiredActionIDs    []string              `json:"required_action_ids,omitempty"`
	RequiresFilters      bool                  `json:"requires_filters,omitempty"`
	RequiresBatchActions bool                  `json:"requires_batch_actions,omitempty"`
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
	RequiresFrameContracts bool                         `json:"requires_frame_contracts,omitempty"`
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
}

type ConsoleIAAuditor struct{}
type ConsoleInteractionAuditor struct{}

func (a ConsoleIAAuditor) Audit(architecture ConsoleIA) ConsoleIAAudit {
	audit := ConsoleIAAudit{
		SystemName:             architecture.Name,
		Version:                architecture.Version,
		SurfaceCount:           len(architecture.Surfaces),
		NavigationCount:        len(architecture.Navigation),
		TopBarAudit:            designsystem.ConsoleChromeLibrary{}.AuditTopBar(architecture.TopBar),
		SurfacesMissingStates:  map[string][]string{},
		StatesMissingActions:   map[string][]string{},
		UnresolvedStateActions: map[string]map[string][]string{},
	}
	navRoutes := map[string]struct{}{}
	for _, item := range architecture.Navigation {
		navRoutes[item.Route] = struct{}{}
	}
	surfaceRoutes := map[string]struct{}{}
	for _, surface := range architecture.Surfaces {
		surfaceRoutes[surface.Route] = struct{}{}
		if len(surface.Filters) == 0 {
			audit.SurfacesMissingFilters = append(audit.SurfacesMissingFilters, surface.Name)
		}
		if len(surface.TopBarActions) == 0 {
			audit.SurfacesMissingActions = append(audit.SurfacesMissingActions, surface.Name)
		}
		missingStates := missingStates(surface)
		if len(missingStates) > 0 {
			audit.SurfacesMissingStates[surface.Name] = missingStates
		}
		actionsByID := map[string]struct{}{}
		for _, action := range surface.TopBarActions {
			actionsByID[action.ActionID] = struct{}{}
		}
		for _, state := range surface.States {
			if state.Name == "default" {
				continue
			}
			if len(state.AllowedActions) == 0 {
				audit.StatesMissingActions[surface.Name] = append(audit.StatesMissingActions[surface.Name], state.Name)
				continue
			}
			var unresolved []string
			for _, action := range state.AllowedActions {
				if _, ok := actionsByID[action]; !ok {
					unresolved = append(unresolved, action)
				}
			}
			if len(unresolved) > 0 {
				if audit.UnresolvedStateActions[surface.Name] == nil {
					audit.UnresolvedStateActions[surface.Name] = map[string][]string{}
				}
				audit.UnresolvedStateActions[surface.Name][state.Name] = unresolved
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
	if len(audit.SurfacesMissingStates) == 0 {
		audit.SurfacesMissingStates = map[string][]string{}
	}
	if len(audit.StatesMissingActions) == 0 {
		audit.StatesMissingActions = map[string][]string{}
	}
	if len(audit.UnresolvedStateActions) == 0 {
		audit.UnresolvedStateActions = map[string]map[string][]string{}
	}
	return audit
}

func (a ConsoleIAAudit) ReadinessScore() float64 {
	if a.TopBarAudit.ReleaseReady() &&
		len(a.SurfacesMissingFilters) == 0 &&
		len(a.SurfacesMissingActions) == 0 &&
		len(a.SurfacesMissingStates) == 0 &&
		len(a.StatesMissingActions) == 0 &&
		len(a.UnresolvedStateActions) == 0 &&
		len(a.OrphanNavigationRoutes) == 0 &&
		len(a.UnnavigableSurfaces) == 0 {
		return 100.0
	}
	return 0.0
}

func RenderConsoleIAReport(architecture ConsoleIA, audit ConsoleIAAudit) string {
	lines := []string{
		"# Console Information Architecture Report",
		fmt.Sprintf("- Name: %s", architecture.TopBar.Name),
		fmt.Sprintf("- Release Ready: %t", audit.ReadinessScore() == 100.0),
		fmt.Sprintf("- Navigation Items: %d", len(architecture.Navigation)),
	}
	for _, surface := range architecture.Surfaces {
		filterNames := []string{}
		for _, filter := range surface.Filters {
			filterNames = append(filterNames, filter.Name)
		}
		actionNames := []string{}
		for _, action := range surface.TopBarActions {
			actionNames = append(actionNames, action.Label)
		}
		missing := audit.SurfacesMissingStates[surface.Name]
		stateMissingActions := audit.StatesMissingActions[surface.Name]
		unresolved := audit.UnresolvedStateActions[surface.Name]
		lines = append(lines, fmt.Sprintf(
			"- %s: route=%s filters=%s actions=%s states=%s missing_states=%s states_without_actions=%s unresolved_state_actions=%s",
			surface.Name,
			surface.Route,
			noneOrJoin(filterNames),
			noneOrJoin(actionNames),
			strings.Join(stateNames(surface.States), ", "),
			noneOrJoin(missing),
			noneOrJoin(stateMissingActions),
			renderNested(unresolved),
		))
	}
	lines = append(lines, "- Surfaces missing filters: "+noneOrJoin(audit.SurfacesMissingFilters))
	lines = append(lines, "- Undefined state actions: "+renderNestedMap(audit.UnresolvedStateActions))
	return strings.Join(lines, "\n")
}

func (a ConsoleInteractionAuditor) Audit(draft ConsoleInteractionDraft) ConsoleInteractionAudit {
	audit := ConsoleInteractionAudit{
		Name:                   draft.Name,
		Version:                draft.Version,
		ContractCount:          len(draft.Contracts),
		SurfacesMissingActions: map[string][]string{},
		SurfacesMissingStates:  map[string][]string{},
		PermissionGaps:         map[string][]string{},
	}
	surfaces := map[string]ConsoleSurface{}
	for _, surface := range draft.Architecture.Surfaces {
		surfaces[surface.Name] = surface
	}
	coveredRoles := map[string]struct{}{}
	for _, contract := range draft.Contracts {
		surface, ok := surfaces[contract.SurfaceName]
		if !ok {
			audit.MissingSurfaces = append(audit.MissingSurfaces, contract.SurfaceName)
			continue
		}
		for _, role := range contract.PermissionRule.AllowedRoles {
			coveredRoles[role] = struct{}{}
		}
		if contract.RequiresFilters && len(surface.Filters) == 0 {
			audit.SurfacesMissingFilters = append(audit.SurfacesMissingFilters, contract.SurfaceName)
		}
		if missing := missingActions(surface, contract.RequiredActionIDs); len(missing) > 0 {
			audit.SurfacesMissingActions[contract.SurfaceName] = missing
		}
		if contract.RequiresBatchActions && !hasBatchAction(surface) {
			audit.SurfacesMissingBatchActions = append(audit.SurfacesMissingBatchActions, contract.SurfaceName)
		}
		if missing := missingStates(surface); len(missing) > 0 {
			audit.SurfacesMissingStates[contract.SurfaceName] = missing
		}
		var permissionGaps []string
		if len(contract.PermissionRule.DeniedRoles) == 0 {
			permissionGaps = append(permissionGaps, "denied-roles")
		}
		if strings.TrimSpace(contract.PermissionRule.AuditEvent) == "" {
			permissionGaps = append(permissionGaps, "audit-event")
		}
		if len(permissionGaps) > 0 {
			audit.PermissionGaps[contract.SurfaceName] = permissionGaps
		}
		if draft.RequiresFrameContracts {
			if strings.TrimSpace(contract.PrimaryPersona) == "" {
				audit.SurfacesMissingPrimaryPersonas = append(audit.SurfacesMissingPrimaryPersonas, contract.SurfaceName)
			}
			if strings.TrimSpace(contract.LinkedWireframeID) == "" {
				audit.SurfacesMissingWireframeLinks = append(audit.SurfacesMissingWireframeLinks, contract.SurfaceName)
			}
			if len(contract.ReviewFocusAreas) == 0 {
				audit.SurfacesMissingReviewFocus = append(audit.SurfacesMissingReviewFocus, contract.SurfaceName)
			}
			if len(contract.DecisionPrompts) == 0 {
				audit.SurfacesMissingDecisionPrompts = append(audit.SurfacesMissingDecisionPrompts, contract.SurfaceName)
			}
		}
	}
	for _, role := range draft.RequiredRoles {
		if _, ok := coveredRoles[role]; !ok {
			audit.UncoveredRoles = append(audit.UncoveredRoles, role)
		}
	}
	if len(audit.SurfacesMissingActions) == 0 {
		audit.SurfacesMissingActions = map[string][]string{}
	}
	if len(audit.SurfacesMissingStates) == 0 {
		audit.SurfacesMissingStates = map[string][]string{}
	}
	if len(audit.PermissionGaps) == 0 {
		audit.PermissionGaps = map[string][]string{}
	}
	return audit
}

func (a ConsoleInteractionAudit) ReadinessScore() float64 {
	if a.ReleaseReady() {
		return 100.0
	}
	return 0.0
}

func (a ConsoleInteractionAudit) ReleaseReady() bool {
	return len(a.MissingSurfaces) == 0 &&
		len(a.SurfacesMissingFilters) == 0 &&
		len(a.SurfacesMissingActions) == 0 &&
		len(a.SurfacesMissingBatchActions) == 0 &&
		len(a.SurfacesMissingStates) == 0 &&
		len(a.PermissionGaps) == 0 &&
		len(a.UncoveredRoles) == 0 &&
		len(a.SurfacesMissingPrimaryPersonas) == 0 &&
		len(a.SurfacesMissingWireframeLinks) == 0 &&
		len(a.SurfacesMissingReviewFocus) == 0 &&
		len(a.SurfacesMissingDecisionPrompts) == 0
}

func RenderConsoleInteractionReport(draft ConsoleInteractionDraft, audit ConsoleInteractionAudit) string {
	lines := []string{
		"# Console Interaction Draft Report",
		fmt.Sprintf("- Critical Pages: %d", len(draft.Contracts)),
		fmt.Sprintf("- Required Roles: %s", noneOrJoin(draft.RequiredRoles)),
		fmt.Sprintf("- Readiness Score: %.1f", audit.ReadinessScore()),
		fmt.Sprintf("- Release Ready: %t", audit.ReleaseReady()),
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
			strings.Join(contract.RequiredActionIDs, ", "),
			strings.Join(actionIDs(surface.TopBarActions), ", "),
			len(surface.Filters),
			strings.Join(stateNames(surface.States), ", "),
			batchLabel(contract.RequiresBatchActions),
			permissionLabel(audit.PermissionGaps[contract.SurfaceName]),
		))
		if contract.PrimaryPersona != "" || contract.LinkedWireframeID != "" || len(contract.ReviewFocusAreas) > 0 {
			lines = append(lines, fmt.Sprintf("persona=%s wireframe=%s review_focus=%s", contract.PrimaryPersona, contract.LinkedWireframeID, strings.Join(contract.ReviewFocusAreas, ",")))
		}
	}
	lines = append(lines, "- Permission gaps: "+renderMapStrings(audit.PermissionGaps))
	lines = append(lines, "- Uncovered roles: "+noneOrJoin(audit.UncoveredRoles))
	lines = append(lines, "- Pages missing personas: "+noneOrJoin(audit.SurfacesMissingPrimaryPersonas))
	lines = append(lines, "- Pages missing wireframe links: "+noneOrJoin(audit.SurfacesMissingWireframeLinks))
	return strings.Join(lines, "\n")
}

func BuildBIG4203ConsoleInteractionDraft() ConsoleInteractionDraft {
	topBar := designsystem.ConsoleTopBar{
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
	}
	return ConsoleInteractionDraft{
		Name:    "BIG-4203 Four Critical Pages",
		Version: "v1",
		Architecture: ConsoleIA{
			Name:    "BigClaw Console IA",
			Version: "v3",
			TopBar:  topBar,
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
			{SurfaceName: "Overview", RequiredActionIDs: []string{"drill-down", "export", "audit"}, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"eng-lead", "platform-admin", "vp-eng"}, DeniedRoles: []string{"viewer"}, AuditEvent: "overview.access.denied"}, PrimaryPersona: "VP Eng", LinkedWireframeID: "wf-overview", ReviewFocusAreas: []string{"metric hierarchy", "drill-down posture", "alert prioritization"}, DecisionPrompts: []string{"Should the overview frame ship in v1?"}},
			{SurfaceName: "Queue", RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiresBatchActions: true, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"platform-admin"}, DeniedRoles: []string{"viewer"}, AuditEvent: "queue.access.denied"}, PrimaryPersona: "Platform Admin", LinkedWireframeID: "wf-queue", ReviewFocusAreas: []string{"queue load", "batch approval posture"}, DecisionPrompts: []string{"Is queue batching safe?"}},
			{SurfaceName: "Run Detail", RequiredActionIDs: []string{"drill-down", "export", "audit"}, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"eng-lead"}, DeniedRoles: []string{"guest"}, AuditEvent: "run-detail.access.denied"}, PrimaryPersona: "Eng Lead", LinkedWireframeID: "wf-run-detail", ReviewFocusAreas: []string{"evidence audit", "context rail"}, DecisionPrompts: []string{"Is replay context sufficient?"}},
			{SurfaceName: "Triage", RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiresBatchActions: true, PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"cross-team-operator"}, DeniedRoles: []string{"viewer"}, AuditEvent: "triage.access.denied"}, PrimaryPersona: "Cross-Team Operator", LinkedWireframeID: "wf-triage", ReviewFocusAreas: []string{"handoff timing", "severity grouping"}, DecisionPrompts: []string{"Is triage ownership clear?"}},
		},
		RequiredRoles:          []string{"eng-lead", "platform-admin", "vp-eng", "cross-team-operator"},
		RequiresFrameContracts: true,
	}
}

func missingStates(surface ConsoleSurface) []string {
	stateSet := map[string]struct{}{}
	for _, state := range surface.States {
		stateSet[state.Name] = struct{}{}
	}
	var missing []string
	for _, name := range requiredSurfaceStates {
		if _, ok := stateSet[name]; !ok {
			missing = append(missing, name)
		}
	}
	return missing
}

func missingActions(surface ConsoleSurface, required []string) []string {
	available := map[string]struct{}{}
	for _, action := range surface.TopBarActions {
		available[action.ActionID] = struct{}{}
	}
	var missing []string
	for _, action := range required {
		if _, ok := available[action]; !ok {
			missing = append(missing, action)
		}
	}
	return missing
}

func hasBatchAction(surface ConsoleSurface) bool {
	for _, action := range surface.TopBarActions {
		if action.RequiresSelection {
			return true
		}
	}
	return false
}

func stateNames(states []SurfaceState) []string {
	names := make([]string, 0, len(states))
	for _, state := range states {
		names = append(names, state.Name)
	}
	return names
}

func actionIDs(actions []GlobalAction) []string {
	names := make([]string, 0, len(actions))
	for _, action := range actions {
		names = append(names, action.ActionID)
	}
	return names
}

func noneOrJoin(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}

func renderNested(items map[string][]string) string {
	if len(items) == 0 {
		return "none"
	}
	var keys []string
	for key := range items {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	var parts []string
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, strings.Join(items[key], ", ")))
	}
	return strings.Join(parts, "; ")
}

func renderNestedMap(items map[string]map[string][]string) string {
	if len(items) == 0 {
		return "none"
	}
	var surfaceNames []string
	for name := range items {
		surfaceNames = append(surfaceNames, name)
	}
	slices.Sort(surfaceNames)
	var parts []string
	for _, name := range surfaceNames {
		parts = append(parts, fmt.Sprintf("%s=%s", name, renderNested(items[name])))
	}
	return strings.Join(parts, "; ")
}

func renderMapStrings(items map[string][]string) string {
	if len(items) == 0 {
		return "none"
	}
	var keys []string
	for key := range items {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	var parts []string
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, strings.Join(items[key], ", ")))
	}
	return strings.Join(parts, "; ")
}

func permissionLabel(gaps []string) string {
	if len(gaps) == 0 {
		return "complete"
	}
	return strings.Join(gaps, ", ")
}

func batchLabel(required bool) string {
	if required {
		return "required"
	}
	return "optional"
}

func findSurface(surfaces []ConsoleSurface, name string) (ConsoleSurface, bool) {
	for _, surface := range surfaces {
		if surface.Name == name {
			return surface, true
		}
	}
	return ConsoleSurface{}, false
}
