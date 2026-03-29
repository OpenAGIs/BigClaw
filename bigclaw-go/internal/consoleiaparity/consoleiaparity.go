package consoleiaparity

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

var requiredSurfaceStates = []string{"default", "empty", "error", "loading"}
var requiredTopBarAccessibility = []string{"focus-visible", "keyboard-navigation", "screen-reader-label"}
var requiredShortcuts = []string{"cmd+k", "ctrl+k"}

type CommandAction struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Section  string `json:"section"`
	Shortcut string `json:"shortcut,omitempty"`
}

type ConsoleCommandEntry struct {
	TriggerLabel         string          `json:"trigger_label"`
	Placeholder          string          `json:"placeholder"`
	Shortcut             string          `json:"shortcut"`
	Commands             []CommandAction `json:"commands,omitempty"`
	RecentQueriesEnabled bool            `json:"recent_queries_enabled,omitempty"`
}

type ConsoleTopBar struct {
	Name                      string              `json:"name"`
	SearchPlaceholder         string              `json:"search_placeholder"`
	EnvironmentOptions        []string            `json:"environment_options,omitempty"`
	TimeRangeOptions          []string            `json:"time_range_options,omitempty"`
	AlertChannels             []string            `json:"alert_channels,omitempty"`
	CommandEntry              ConsoleCommandEntry `json:"command_entry"`
	DocumentationComplete     bool                `json:"documentation_complete"`
	AccessibilityRequirements []string            `json:"accessibility_requirements,omitempty"`
}

func (b ConsoleTopBar) HasGlobalSearch() bool      { return strings.TrimSpace(b.SearchPlaceholder) != "" }
func (b ConsoleTopBar) HasEnvironmentSwitch() bool { return len(b.EnvironmentOptions) >= 2 }
func (b ConsoleTopBar) HasTimeRangeSwitch() bool   { return len(b.TimeRangeOptions) >= 2 }
func (b ConsoleTopBar) HasAlertEntry() bool        { return len(b.AlertChannels) > 0 }
func (b ConsoleTopBar) HasCommandShell() bool {
	return strings.TrimSpace(b.CommandEntry.TriggerLabel) != "" && len(b.CommandEntry.Commands) > 0
}

type ConsoleTopBarAudit struct {
	Name                     string   `json:"name"`
	MissingCapabilities      []string `json:"missing_capabilities,omitempty"`
	DocumentationComplete    bool     `json:"documentation_complete"`
	AccessibilityComplete    bool     `json:"accessibility_complete"`
	CommandShortcutSupported bool     `json:"command_shortcut_supported"`
	CommandCount             int      `json:"command_count"`
}

func (a ConsoleTopBarAudit) ReleaseReady() bool {
	return len(a.MissingCapabilities) == 0 &&
		a.DocumentationComplete &&
		a.AccessibilityComplete &&
		a.CommandShortcutSupported
}

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
	Intent            string `json:"intent,omitempty"`
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
	Message        string   `json:"message,omitempty"`
	AllowedActions []string `json:"allowed_actions,omitempty"`
}

type ConsoleSurface struct {
	Name                string             `json:"name"`
	Route               string             `json:"route"`
	NavigationSection   string             `json:"navigation_section"`
	TopBarActions       []GlobalAction     `json:"top_bar_actions,omitempty"`
	Filters             []FilterDefinition `json:"filters,omitempty"`
	States              []SurfaceState     `json:"states,omitempty"`
	SupportsBulkActions bool               `json:"supports_bulk_actions,omitempty"`
}

func (s ConsoleSurface) ActionIDs() []string {
	out := make([]string, 0, len(s.TopBarActions))
	for _, action := range s.TopBarActions {
		out = append(out, action.ActionID)
	}
	return out
}

func (s ConsoleSurface) StateNames() []string {
	out := make([]string, 0, len(s.States))
	for _, state := range s.States {
		out = append(out, state.Name)
	}
	return out
}

func (s ConsoleSurface) MissingRequiredStates() []string {
	present := map[string]struct{}{}
	for _, name := range s.StateNames() {
		present[name] = struct{}{}
	}
	missing := make([]string, 0)
	for _, state := range requiredSurfaceStates {
		if _, ok := present[state]; !ok {
			missing = append(missing, state)
		}
	}
	return missing
}

func (s ConsoleSurface) UnresolvedStateActions() map[string][]string {
	available := map[string]struct{}{}
	for _, id := range s.ActionIDs() {
		available[id] = struct{}{}
	}
	unresolved := map[string][]string{}
	for _, state := range s.States {
		missing := make([]string, 0)
		for _, id := range state.AllowedActions {
			if _, ok := available[id]; !ok {
				missing = append(missing, id)
			}
		}
		sort.Strings(missing)
		if len(missing) > 0 {
			unresolved[state.Name] = missing
		}
	}
	return unresolved
}

func (s ConsoleSurface) StatesMissingActions() []string {
	missing := make([]string, 0)
	for _, state := range s.States {
		if state.Name != "default" && len(state.AllowedActions) == 0 {
			missing = append(missing, state.Name)
		}
	}
	return missing
}

type ConsoleIA struct {
	Name       string           `json:"name"`
	Version    string           `json:"version"`
	Navigation []NavigationItem `json:"navigation,omitempty"`
	Surfaces   []ConsoleSurface `json:"surfaces,omitempty"`
	TopBar     ConsoleTopBar    `json:"top_bar"`
}

func (a ConsoleIA) RouteIndex() map[string]ConsoleSurface {
	out := make(map[string]ConsoleSurface, len(a.Surfaces))
	for _, surface := range a.Surfaces {
		out[surface.Route] = surface
	}
	return out
}

type SurfacePermissionRule struct {
	AllowedRoles []string `json:"allowed_roles,omitempty"`
	DeniedRoles  []string `json:"denied_roles,omitempty"`
	AuditEvent   string   `json:"audit_event,omitempty"`
}

func (r SurfacePermissionRule) MissingCoverage() []string {
	missing := make([]string, 0, 3)
	if len(r.AllowedRoles) == 0 {
		missing = append(missing, "allowed-roles")
	}
	if len(r.DeniedRoles) == 0 {
		missing = append(missing, "denied-roles")
	}
	if strings.TrimSpace(r.AuditEvent) == "" {
		missing = append(missing, "audit-event")
	}
	return missing
}

func (r SurfacePermissionRule) Complete() bool { return len(r.MissingCoverage()) == 0 }

type SurfaceInteractionContract struct {
	SurfaceName          string                `json:"surface_name"`
	RequiredActionIDs    []string              `json:"required_action_ids,omitempty"`
	RequiresFilters      bool                  `json:"requires_filters"`
	RequiresBatchActions bool                  `json:"requires_batch_actions"`
	RequiredStates       []string              `json:"required_states,omitempty"`
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
}

func (a ConsoleInteractionAudit) ReadinessScore() float64 {
	if a.ContractCount == 0 {
		return 0
	}
	penalties := len(a.MissingSurfaces) +
		len(a.SurfacesMissingFilters) +
		len(a.SurfacesMissingActions) +
		len(a.SurfacesMissingBatchActions) +
		len(a.SurfacesMissingStates) +
		len(a.PermissionGaps) +
		len(a.UncoveredRoles) +
		len(a.SurfacesMissingPrimaryPersonas) +
		len(a.SurfacesMissingWireframeLinks) +
		len(a.SurfacesMissingReviewFocus) +
		len(a.SurfacesMissingDecisionPrompts)
	score := maxFloat(0, 100-(float64(penalties)*100)/float64(a.ContractCount))
	return round1(score)
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

type ConsoleIAAudit struct {
	SystemName             string                         `json:"system_name"`
	Version                string                         `json:"version"`
	SurfaceCount           int                            `json:"surface_count"`
	NavigationCount        int                            `json:"navigation_count"`
	TopBarAudit            ConsoleTopBarAudit             `json:"top_bar_audit"`
	SurfacesMissingFilters []string                       `json:"surfaces_missing_filters,omitempty"`
	SurfacesMissingActions []string                       `json:"surfaces_missing_actions,omitempty"`
	SurfacesMissingStates  map[string][]string            `json:"surfaces_missing_states,omitempty"`
	StatesMissingActions   map[string][]string            `json:"states_missing_actions,omitempty"`
	UnresolvedStateActions map[string]map[string][]string `json:"unresolved_state_actions,omitempty"`
	OrphanNavigationRoutes []string                       `json:"orphan_navigation_routes,omitempty"`
	UnnavigableSurfaces    []string                       `json:"unnavigable_surfaces,omitempty"`
}

func (a ConsoleIAAudit) ReadinessScore() float64 {
	if a.SurfaceCount == 0 {
		return 0
	}
	penalties := 0
	if !a.TopBarAudit.ReleaseReady() {
		penalties++
	}
	penalties += len(a.SurfacesMissingFilters) +
		len(a.SurfacesMissingActions) +
		len(a.SurfacesMissingStates) +
		len(a.StatesMissingActions) +
		len(a.UnresolvedStateActions) +
		len(a.OrphanNavigationRoutes) +
		len(a.UnnavigableSurfaces)
	score := maxFloat(0, 100-(float64(penalties)*100)/float64(a.SurfaceCount))
	return round1(score)
}

type ConsoleChromeLibrary struct{}

func (ConsoleChromeLibrary) AuditTopBar(topBar ConsoleTopBar) ConsoleTopBarAudit {
	missing := make([]string, 0)
	if !topBar.HasGlobalSearch() {
		missing = append(missing, "global-search")
	}
	if !topBar.HasTimeRangeSwitch() {
		missing = append(missing, "time-range-switch")
	}
	if !topBar.HasEnvironmentSwitch() {
		missing = append(missing, "environment-switch")
	}
	if !topBar.HasAlertEntry() {
		missing = append(missing, "alert-entry")
	}
	if !topBar.HasCommandShell() {
		missing = append(missing, "command-shell")
	}

	normalized := map[string]struct{}{}
	for _, part := range strings.Split(topBar.CommandEntry.Shortcut, "/") {
		item := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(part), " ", ""))
		if item != "" {
			normalized[item] = struct{}{}
		}
	}
	access := map[string]struct{}{}
	for _, item := range topBar.AccessibilityRequirements {
		access[item] = struct{}{}
	}
	accessibilityComplete := true
	for _, req := range requiredTopBarAccessibility {
		if _, ok := access[req]; !ok {
			accessibilityComplete = false
			break
		}
	}
	shortcutsComplete := true
	for _, req := range requiredShortcuts {
		if _, ok := normalized[req]; !ok {
			shortcutsComplete = false
			break
		}
	}

	return ConsoleTopBarAudit{
		Name:                     topBar.Name,
		MissingCapabilities:      missing,
		DocumentationComplete:    topBar.DocumentationComplete,
		AccessibilityComplete:    accessibilityComplete,
		CommandShortcutSupported: shortcutsComplete,
		CommandCount:             len(topBar.CommandEntry.Commands),
	}
}

type ConsoleIAAuditor struct{}

func (ConsoleIAAuditor) Audit(architecture ConsoleIA) ConsoleIAAudit {
	topBarAudit := (ConsoleChromeLibrary{}).AuditTopBar(architecture.TopBar)
	routeIndex := architecture.RouteIndex()
	navigationRoutes := map[string]struct{}{}
	for _, item := range architecture.Navigation {
		navigationRoutes[item.Route] = struct{}{}
	}

	var missingFilters, missingActions []string
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
		if missing := surface.MissingRequiredStates(); len(missing) > 0 {
			missingStates[surface.Name] = missing
		}
		if missing := surface.StatesMissingActions(); len(missing) > 0 {
			statesMissingActions[surface.Name] = missing
		}
		if unresolved := surface.UnresolvedStateActions(); len(unresolved) > 0 {
			unresolvedStateActions[surface.Name] = unresolved
		}
	}

	orphanRoutes := make([]string, 0)
	for route := range navigationRoutes {
		if _, ok := routeIndex[route]; !ok {
			orphanRoutes = append(orphanRoutes, route)
		}
	}
	sort.Strings(orphanRoutes)

	unnavigable := make([]string, 0)
	for _, surface := range architecture.Surfaces {
		if _, ok := navigationRoutes[surface.Route]; !ok {
			unnavigable = append(unnavigable, surface.Name)
		}
	}
	sort.Strings(unnavigable)
	sort.Strings(missingFilters)
	sort.Strings(missingActions)

	return ConsoleIAAudit{
		SystemName:             architecture.Name,
		Version:                architecture.Version,
		SurfaceCount:           len(architecture.Surfaces),
		NavigationCount:        len(architecture.Navigation),
		TopBarAudit:            topBarAudit,
		SurfacesMissingFilters: missingFilters,
		SurfacesMissingActions: missingActions,
		SurfacesMissingStates:  sortMapSlices(missingStates),
		StatesMissingActions:   sortMapSlices(statesMissingActions),
		UnresolvedStateActions: sortNestedMapSlices(unresolvedStateActions),
		OrphanNavigationRoutes: orphanRoutes,
		UnnavigableSurfaces:    unnavigable,
	}
}

type ConsoleInteractionAuditor struct{}

func (ConsoleInteractionAuditor) Audit(draft ConsoleInteractionDraft) ConsoleInteractionAudit {
	routeIndex := draft.Architecture.RouteIndex()
	var missingSurfaces, missingFilters, missingBatch, missingPersonas, missingWireframes, missingReview, missingPrompts []string
	missingActions := map[string][]string{}
	missingStates := map[string][]string{}
	permissionGaps := map[string][]string{}
	referencedRoles := map[string]struct{}{}

	for _, contract := range draft.Contracts {
		surface, ok := routeIndex[contract.SurfaceName]
		if !ok {
			for _, candidate := range draft.Architecture.Surfaces {
				if candidate.Name == contract.SurfaceName {
					surface = candidate
					ok = true
					break
				}
			}
		}
		if !ok {
			missingSurfaces = append(missingSurfaces, contract.SurfaceName)
			continue
		}
		if contract.RequiresFilters && len(surface.Filters) == 0 {
			missingFilters = append(missingFilters, contract.SurfaceName)
		}
		available := map[string]struct{}{}
		for _, id := range surface.ActionIDs() {
			available[id] = struct{}{}
		}
		actionGaps := make([]string, 0)
		for _, id := range contract.RequiredActionIDs {
			if _, ok := available[id]; !ok {
				actionGaps = append(actionGaps, id)
			}
		}
		sort.Strings(actionGaps)
		if len(actionGaps) > 0 {
			missingActions[contract.SurfaceName] = actionGaps
		}
		if contract.RequiresBatchActions {
			hasBatch := false
			for _, action := range surface.TopBarActions {
				if action.RequiresSelection {
					hasBatch = true
					break
				}
			}
			if !hasBatch {
				missingBatch = append(missingBatch, contract.SurfaceName)
			}
		}
		requiredStates := contract.RequiredStates
		if len(requiredStates) == 0 {
			requiredStates = append([]string(nil), requiredSurfaceStates...)
		}
		stateNames := map[string]struct{}{}
		for _, name := range surface.StateNames() {
			stateNames[name] = struct{}{}
		}
		stateGaps := make([]string, 0)
		for _, state := range requiredStates {
			if _, ok := stateNames[state]; !ok {
				stateGaps = append(stateGaps, state)
			}
		}
		sort.Strings(stateGaps)
		if len(stateGaps) > 0 {
			missingStates[contract.SurfaceName] = stateGaps
		}
		for _, role := range contract.PermissionRule.AllowedRoles {
			referencedRoles[role] = struct{}{}
		}
		for _, role := range contract.PermissionRule.DeniedRoles {
			referencedRoles[role] = struct{}{}
		}
		if gaps := contract.PermissionRule.MissingCoverage(); len(gaps) > 0 {
			permissionGaps[contract.SurfaceName] = gaps
		}
		if draft.RequiresFrameContracts {
			if strings.TrimSpace(contract.PrimaryPersona) == "" {
				missingPersonas = append(missingPersonas, contract.SurfaceName)
			}
			if strings.TrimSpace(contract.LinkedWireframeID) == "" {
				missingWireframes = append(missingWireframes, contract.SurfaceName)
			}
			if len(contract.ReviewFocusAreas) == 0 {
				missingReview = append(missingReview, contract.SurfaceName)
			}
			if len(contract.DecisionPrompts) == 0 {
				missingPrompts = append(missingPrompts, contract.SurfaceName)
			}
		}
	}

	uncoveredRoles := make([]string, 0)
	for _, role := range draft.RequiredRoles {
		if _, ok := referencedRoles[role]; !ok {
			uncoveredRoles = append(uncoveredRoles, role)
		}
	}
	sort.Strings(missingSurfaces)
	sort.Strings(missingFilters)
	sort.Strings(missingBatch)
	sort.Strings(missingPersonas)
	sort.Strings(missingWireframes)
	sort.Strings(missingReview)
	sort.Strings(missingPrompts)
	sort.Strings(uncoveredRoles)

	return ConsoleInteractionAudit{
		Name:                           draft.Name,
		Version:                        draft.Version,
		ContractCount:                  len(draft.Contracts),
		MissingSurfaces:                missingSurfaces,
		SurfacesMissingFilters:         missingFilters,
		SurfacesMissingActions:         sortMapSlices(missingActions),
		SurfacesMissingBatchActions:    missingBatch,
		SurfacesMissingStates:          sortMapSlices(missingStates),
		PermissionGaps:                 sortMapSlices(permissionGaps),
		UncoveredRoles:                 uncoveredRoles,
		SurfacesMissingPrimaryPersonas: missingPersonas,
		SurfacesMissingWireframeLinks:  missingWireframes,
		SurfacesMissingReviewFocus:     missingReview,
		SurfacesMissingDecisionPrompts: missingPrompts,
	}
}

func RenderConsoleIAReport(architecture ConsoleIA, audit ConsoleIAAudit) string {
	lines := []string{
		"# Console Information Architecture Report",
		"",
		fmt.Sprintf("- Name: %s", architecture.Name),
		fmt.Sprintf("- Version: %s", architecture.Version),
		fmt.Sprintf("- Navigation Items: %d", audit.NavigationCount),
		fmt.Sprintf("- Surfaces: %d", audit.SurfaceCount),
		fmt.Sprintf("- Readiness Score: %.1f", audit.ReadinessScore()),
		"",
		"## Global Header",
		"",
		fmt.Sprintf("- Name: %s", orNoneString(architecture.TopBar.Name)),
		fmt.Sprintf("- Release Ready: %s", pyBool(audit.TopBarAudit.ReleaseReady())),
		fmt.Sprintf("- Missing capabilities: %s", joinOrNone(audit.TopBarAudit.MissingCapabilities)),
		fmt.Sprintf("- Command Count: %d", audit.TopBarAudit.CommandCount),
		fmt.Sprintf("- Cmd/Ctrl+K supported: %s", pyBool(audit.TopBarAudit.CommandShortcutSupported)),
		"",
		"## Navigation",
		"",
	}
	if len(architecture.Navigation) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, item := range architecture.Navigation {
			lines = append(lines, fmt.Sprintf("- %s / %s: route=%s badge=%d icon=%s", item.Section, item.Name, item.Route, item.BadgeCount, orNoneString(item.Icon)))
		}
	}
	lines = append(lines, "", "## Surface Coverage", "")
	if len(architecture.Surfaces) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, surface := range architecture.Surfaces {
			filterNames := make([]string, 0, len(surface.Filters))
			for _, f := range surface.Filters {
				filterNames = append(filterNames, f.Name)
			}
			actionLabels := make([]string, 0, len(surface.TopBarActions))
			for _, action := range surface.TopBarActions {
				actionLabels = append(actionLabels, action.Label)
			}
			unresolved := audit.UnresolvedStateActions[surface.Name]
			unresolvedText := "none"
			if len(unresolved) > 0 {
				parts := make([]string, 0, len(unresolved))
				for _, state := range sortedKeysNested(unresolved) {
					parts = append(parts, fmt.Sprintf("%s=%s", state, strings.Join(unresolved[state], ", ")))
				}
				unresolvedText = strings.Join(parts, "; ")
			}
			lines = append(lines, fmt.Sprintf(
				"- %s: route=%s filters=%s actions=%s states=%s missing_states=%s states_without_actions=%s unresolved_state_actions=%s",
				surface.Name,
				surface.Route,
				joinOrNone(filterNames),
				joinOrNone(actionLabels),
				joinOrNone(surface.StateNames()),
				joinOrNone(surface.MissingRequiredStates()),
				joinOrNone(audit.StatesMissingActions[surface.Name]),
				unresolvedText,
			))
		}
	}
	lines = append(lines,
		"",
		"## Gaps",
		"",
		fmt.Sprintf("- Surfaces missing filters: %s", joinOrNone(audit.SurfacesMissingFilters)),
		fmt.Sprintf("- Surfaces missing top-bar actions: %s", joinOrNone(audit.SurfacesMissingActions)),
		fmt.Sprintf("- Surfaces missing required states: %s", formatMapList(audit.SurfacesMissingStates)),
		fmt.Sprintf("- States without recovery actions: %s", formatMapList(audit.StatesMissingActions)),
		fmt.Sprintf("- Undefined state actions: %s", formatNestedMapList(audit.UnresolvedStateActions)),
		fmt.Sprintf("- Orphan navigation routes: %s", joinOrNone(audit.OrphanNavigationRoutes)),
		fmt.Sprintf("- Unnavigable surfaces: %s", joinOrNone(audit.UnnavigableSurfaces)),
	)
	return strings.Join(lines, "\n") + "\n"
}

func RenderConsoleInteractionReport(draft ConsoleInteractionDraft, audit ConsoleInteractionAudit) string {
	routeIndex := draft.Architecture.RouteIndex()
	lines := []string{
		"# Console Interaction Draft Report",
		"",
		fmt.Sprintf("- Name: %s", draft.Name),
		fmt.Sprintf("- Version: %s", draft.Version),
		fmt.Sprintf("- Critical Pages: %d", len(draft.Contracts)),
		fmt.Sprintf("- Required Roles: %s", joinOrNone(draft.RequiredRoles)),
		fmt.Sprintf("- Readiness Score: %.1f", audit.ReadinessScore()),
		fmt.Sprintf("- Release Ready: %s", pyBool(audit.ReleaseReady())),
		"",
		"## Page Coverage",
		"",
	}
	if len(draft.Contracts) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, contract := range draft.Contracts {
			surface, ok := routeIndex[contract.SurfaceName]
			if !ok {
				for _, candidate := range draft.Architecture.Surfaces {
					if candidate.Name == contract.SurfaceName {
						surface = candidate
						ok = true
						break
					}
				}
			}
			if !ok {
				lines = append(lines, fmt.Sprintf("- %s: missing surface definition", contract.SurfaceName))
				continue
			}
			batchMode := "optional"
			if contract.RequiresBatchActions {
				batchMode = "required"
			}
			permissionState := "incomplete"
			if contract.PermissionRule.Complete() {
				permissionState = "complete"
			}
			lines = append(lines, fmt.Sprintf(
				"- %s: route=%s required_actions=%s available_actions=%s filters=%d states=%s batch=%s permissions=%s",
				contract.SurfaceName,
				surface.Route,
				joinOrNone(contract.RequiredActionIDs),
				joinOrNone(surface.ActionIDs()),
				len(surface.Filters),
				joinOrNone(surface.StateNames()),
				batchMode,
				permissionState,
			))
			lines = append(lines, fmt.Sprintf(
				"  persona=%s wireframe=%s review_focus=%s decision_prompts=%s",
				orNoneString(contract.PrimaryPersona),
				orNoneString(contract.LinkedWireframeID),
				joinCompactOrNone(contract.ReviewFocusAreas),
				joinCompactOrNone(contract.DecisionPrompts),
			))
		}
	}
	lines = append(lines,
		"",
		"## Gaps",
		"",
		fmt.Sprintf("- Missing surfaces: %s", joinOrNone(audit.MissingSurfaces)),
		fmt.Sprintf("- Pages missing filters: %s", joinOrNone(audit.SurfacesMissingFilters)),
		fmt.Sprintf("- Pages missing actions: %s", formatMapList(audit.SurfacesMissingActions)),
		fmt.Sprintf("- Pages missing batch actions: %s", joinOrNone(audit.SurfacesMissingBatchActions)),
		fmt.Sprintf("- Pages missing states: %s", formatMapList(audit.SurfacesMissingStates)),
		fmt.Sprintf("- Permission gaps: %s", formatMapList(audit.PermissionGaps)),
		fmt.Sprintf("- Uncovered roles: %s", joinOrNone(audit.UncoveredRoles)),
		fmt.Sprintf("- Pages missing personas: %s", joinOrNone(audit.SurfacesMissingPrimaryPersonas)),
		fmt.Sprintf("- Pages missing wireframe links: %s", joinOrNone(audit.SurfacesMissingWireframeLinks)),
		fmt.Sprintf("- Pages missing review focus: %s", joinOrNone(audit.SurfacesMissingReviewFocus)),
		fmt.Sprintf("- Pages missing decision prompts: %s", joinOrNone(audit.SurfacesMissingDecisionPrompts)),
	)
	return strings.Join(lines, "\n") + "\n"
}

func BuildBig4203ConsoleInteractionDraft() ConsoleInteractionDraft {
	return ConsoleInteractionDraft{
		Name:                   "BIG-4203 Four Critical Pages",
		Version:                "v4.0-design-sprint",
		RequiredRoles:          []string{"eng-lead", "platform-admin", "vp-eng", "cross-team-operator"},
		RequiresFrameContracts: true,
		Architecture: ConsoleIA{
			Name:    "BigClaw Console IA",
			Version: "v4.0-design-sprint",
			TopBar: ConsoleTopBar{
				Name:                      "BigClaw Global Header",
				SearchPlaceholder:         "Search runs, queues, prompts, and commands",
				EnvironmentOptions:        []string{"Production", "Staging", "Shadow"},
				TimeRangeOptions:          []string{"24h", "7d", "30d"},
				AlertChannels:             []string{"approvals", "sla", "regressions"},
				DocumentationComplete:     true,
				AccessibilityRequirements: []string{"keyboard-navigation", "screen-reader-label", "focus-visible"},
				CommandEntry: ConsoleCommandEntry{
					TriggerLabel: "Command Menu",
					Placeholder:  "Jump to a run, queue, or release control action",
					Shortcut:     "Cmd+K / Ctrl+K",
					Commands: []CommandAction{
						{ID: "search-runs", Title: "Search runs", Section: "Navigate", Shortcut: "/"},
						{ID: "open-queue", Title: "Open queue control", Section: "Operate"},
						{ID: "open-triage", Title: "Open triage center", Section: "Operate"},
					},
				},
			},
			Navigation: []NavigationItem{
				{Name: "Overview", Route: "/overview", Section: "Operate", Icon: "dashboard"},
				{Name: "Queue", Route: "/queue", Section: "Operate", Icon: "queue"},
				{Name: "Run Detail", Route: "/runs/detail", Section: "Operate", Icon: "activity"},
				{Name: "Triage", Route: "/triage", Section: "Operate", Icon: "alert"},
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
					Filters: []FilterDefinition{
						{Name: "Team", Field: "team", Control: "select", Options: []string{"all", "platform", "product"}},
						{Name: "Time", Field: "time_range", Control: "segmented", Options: []string{"24h", "7d", "30d"}, DefaultValue: "7d"},
					},
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
					Filters: []FilterDefinition{
						{Name: "Status", Field: "status", Control: "select", Options: []string{"all", "queued", "approval"}},
						{Name: "Owner", Field: "owner", Control: "search"},
					},
					States: []SurfaceState{
						{Name: "default"},
						{Name: "loading", AllowedActions: []string{"export"}},
						{Name: "empty", AllowedActions: []string{"audit"}},
						{Name: "error", AllowedActions: []string{"audit"}},
					},
					SupportsBulkActions: true,
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
					Filters: []FilterDefinition{
						{Name: "Run", Field: "run_id", Control: "search"},
						{Name: "Replay Mode", Field: "replay_mode", Control: "select", Options: []string{"latest", "failure-only"}},
					},
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
					Filters: []FilterDefinition{
						{Name: "Severity", Field: "severity", Control: "select", Options: []string{"all", "high", "critical"}},
						{Name: "Workflow", Field: "workflow", Control: "select", Options: []string{"all", "triage", "handoff"}},
					},
					States: []SurfaceState{
						{Name: "default"},
						{Name: "loading", AllowedActions: []string{"export"}},
						{Name: "empty", AllowedActions: []string{"audit"}},
						{Name: "error", AllowedActions: []string{"audit"}},
					},
					SupportsBulkActions: true,
				},
			},
		},
		Contracts: []SurfaceInteractionContract{
			{
				SurfaceName:       "Overview",
				RequiredActionIDs: []string{"drill-down", "export", "audit"},
				RequiresFilters:   true,
				PermissionRule:    SurfacePermissionRule{AllowedRoles: []string{"eng-lead", "platform-admin", "vp-eng", "cross-team-operator"}, DeniedRoles: []string{"guest"}, AuditEvent: "overview.access.denied"},
				PrimaryPersona:    "VP Eng",
				LinkedWireframeID: "wf-overview",
				ReviewFocusAreas:  []string{"metric hierarchy", "drill-down posture", "alert prioritization"},
				DecisionPrompts: []string{
					"Is the executive KPI density still scannable within one screen?",
					"Do risk and blocker cards point to the correct downstream investigation surface?",
				},
			},
			{
				SurfaceName:          "Queue",
				RequiredActionIDs:    []string{"drill-down", "export", "audit"},
				RequiresFilters:      true,
				RequiresBatchActions: true,
				PermissionRule:       SurfacePermissionRule{AllowedRoles: []string{"eng-lead", "platform-admin", "cross-team-operator"}, DeniedRoles: []string{"vp-eng", "guest"}, AuditEvent: "queue.access.denied"},
				PrimaryPersona:       "Platform Admin",
				LinkedWireframeID:    "wf-queue",
				ReviewFocusAreas:     []string{"batch approvals", "denied-role state", "audit rail"},
				DecisionPrompts: []string{
					"Does the queue clearly separate selection, confirmation, and audit outcomes?",
					"Is the denied-role treatment explicit enough for VP Eng and guest personas?",
				},
			},
			{
				SurfaceName:       "Run Detail",
				RequiredActionIDs: []string{"drill-down", "export", "audit"},
				RequiresFilters:   true,
				PermissionRule:    SurfacePermissionRule{AllowedRoles: []string{"eng-lead", "platform-admin", "vp-eng", "cross-team-operator"}, DeniedRoles: []string{"guest"}, AuditEvent: "run-detail.access.denied"},
				PrimaryPersona:    "Eng Lead",
				LinkedWireframeID: "wf-run-detail",
				ReviewFocusAreas:  []string{"replay context", "artifact evidence", "escalation path"},
				DecisionPrompts: []string{
					"Can reviewers distinguish replay, compare, and escalated states without narration?",
					"Is the audit trail visible at the moment an escalation decision is made?",
				},
			},
			{
				SurfaceName:          "Triage",
				RequiredActionIDs:    []string{"drill-down", "export", "audit"},
				RequiresFilters:      true,
				RequiresBatchActions: true,
				PermissionRule:       SurfacePermissionRule{AllowedRoles: []string{"eng-lead", "platform-admin", "cross-team-operator"}, DeniedRoles: []string{"vp-eng", "guest"}, AuditEvent: "triage.access.denied"},
				PrimaryPersona:       "Cross-Team Operator",
				LinkedWireframeID:    "wf-triage",
				ReviewFocusAreas:     []string{"handoff path", "bulk assignment", "ownership history"},
				DecisionPrompts: []string{
					"Does the triage frame explain handoff consequences before ownership changes commit?",
					"Is bulk assignment discoverable without overpowering the audit context?",
				},
			},
		},
	}
}

func (a ConsoleIA) ToMap() (map[string]any, error) { return toMap(a) }
func ConsoleIAFromMap(data map[string]any) (ConsoleIA, error) {
	var out ConsoleIA
	return out, fromMap(data, &out)
}
func (a ConsoleIAAudit) ToMap() (map[string]any, error) { return toMap(a) }
func ConsoleIAAuditFromMap(data map[string]any) (ConsoleIAAudit, error) {
	var out ConsoleIAAudit
	return out, fromMap(data, &out)
}
func (d ConsoleInteractionDraft) ToMap() (map[string]any, error) { return toMap(d) }
func ConsoleInteractionDraftFromMap(data map[string]any) (ConsoleInteractionDraft, error) {
	var out ConsoleInteractionDraft
	return out, fromMap(data, &out)
}

func toMap(value any) (map[string]any, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func fromMap(data map[string]any, target any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}

func sortMapSlices(in map[string][]string) map[string][]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string][]string, len(in))
	for key, values := range in {
		copied := append([]string(nil), values...)
		sort.Strings(copied)
		out[key] = copied
	}
	return out
}

func sortNestedMapSlices(in map[string]map[string][]string) map[string]map[string][]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]map[string][]string, len(in))
	for key, nested := range in {
		out[key] = sortMapSlices(nested)
	}
	return out
}

func sortedKeysNested(in map[string][]string) []string {
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func formatMapList(in map[string][]string) string {
	if len(in) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, strings.Join(in[key], ", ")))
	}
	return strings.Join(parts, "; ")
}

func formatNestedMapList(in map[string]map[string][]string) string {
	if len(in) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		nested := in[key]
		stateKeys := make([]string, 0, len(nested))
		for state := range nested {
			stateKeys = append(stateKeys, state)
		}
		sort.Strings(stateKeys)
		stateParts := make([]string, 0, len(stateKeys))
		for _, state := range stateKeys {
			stateParts = append(stateParts, fmt.Sprintf("%s:%s", state, strings.Join(nested[state], "/")))
		}
		parts = append(parts, fmt.Sprintf("%s=%s", key, strings.Join(stateParts, ", ")))
	}
	return strings.Join(parts, "; ")
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func joinCompactOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ",")
}

func orNoneString(value string) string {
	if strings.TrimSpace(value) == "" {
		return "none"
	}
	return value
}

func pyBool(value bool) string {
	if value {
		return "True"
	}
	return "False"
}

func round1(value float64) float64 { return float64(int(value*10+0.5)) / 10 }
func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
