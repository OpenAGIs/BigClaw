package consoleia

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"bigclaw-go/internal/designsystem"
)

var requiredSurfaceStates = []string{"default", "empty", "error", "loading"}

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

func (c ConsoleSurface) ActionIDs() []string {
	out := make([]string, 0, len(c.TopBarActions))
	for _, action := range c.TopBarActions {
		out = append(out, action.ActionID)
	}
	return out
}

func (c ConsoleSurface) StateNames() []string {
	out := make([]string, 0, len(c.States))
	for _, state := range c.States {
		out = append(out, state.Name)
	}
	return out
}

func (c ConsoleSurface) MissingRequiredStates() []string {
	names := c.StateNames()
	missing := make([]string, 0)
	for _, state := range requiredSurfaceStates {
		if !contains(names, state) {
			missing = append(missing, state)
		}
	}
	return missing
}

func (c ConsoleSurface) UnresolvedStateActions() map[string][]string {
	available := c.ActionIDs()
	out := make(map[string][]string)
	for _, state := range c.States {
		missing := make([]string, 0)
		for _, actionID := range state.AllowedActions {
			if !contains(available, actionID) {
				missing = append(missing, actionID)
			}
		}
		if len(missing) > 0 {
			sort.Strings(missing)
			out[state.Name] = missing
		}
	}
	return out
}

func (c ConsoleSurface) StatesMissingActions() []string {
	out := make([]string, 0)
	for _, state := range c.States {
		if state.Name != "default" && len(state.AllowedActions) == 0 {
			out = append(out, state.Name)
		}
	}
	return out
}

type ConsoleIA struct {
	Name       string                     `json:"name"`
	Version    string                     `json:"version"`
	Navigation []NavigationItem           `json:"navigation,omitempty"`
	Surfaces   []ConsoleSurface           `json:"surfaces,omitempty"`
	TopBar     designsystem.ConsoleTopBar `json:"top_bar"`
}

func (c ConsoleIA) RouteIndex() map[string]ConsoleSurface {
	index := make(map[string]ConsoleSurface, len(c.Surfaces))
	for _, surface := range c.Surfaces {
		index[surface.Route] = surface
	}
	return index
}

type SurfacePermissionRule struct {
	AllowedRoles []string `json:"allowed_roles,omitempty"`
	DeniedRoles  []string `json:"denied_roles,omitempty"`
	AuditEvent   string   `json:"audit_event,omitempty"`
}

func (s SurfacePermissionRule) MissingCoverage() []string {
	out := make([]string, 0)
	if len(s.AllowedRoles) == 0 {
		out = append(out, "allowed-roles")
	}
	if len(s.DeniedRoles) == 0 {
		out = append(out, "denied-roles")
	}
	if strings.TrimSpace(s.AuditEvent) == "" {
		out = append(out, "audit-event")
	}
	return out
}

func (s SurfacePermissionRule) Complete() bool {
	return len(s.MissingCoverage()) == 0
}

type SurfaceInteractionContract struct {
	SurfaceName          string                `json:"surface_name"`
	RequiredActionIDs    []string              `json:"required_action_ids,omitempty"`
	RequiresFilters      bool                  `json:"requires_filters"`
	RequiresBatchActions bool                  `json:"requires_batch_actions,omitempty"`
	RequiredStates       []string              `json:"required_states,omitempty"`
	PermissionRule       SurfacePermissionRule `json:"permission_rule"`
	PrimaryPersona       string                `json:"primary_persona,omitempty"`
	LinkedWireframeID    string                `json:"linked_wireframe_id,omitempty"`
	ReviewFocusAreas     []string              `json:"review_focus_areas,omitempty"`
	DecisionPrompts      []string              `json:"decision_prompts,omitempty"`
}

func (s *SurfaceInteractionContract) UnmarshalJSON(data []byte) error {
	type alias SurfaceInteractionContract
	aux := alias{
		RequiresFilters: true,
		RequiredStates:  append([]string(nil), requiredSurfaceStates...),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*s = SurfaceInteractionContract(aux)
	if len(s.RequiredStates) == 0 {
		s.RequiredStates = append([]string(nil), requiredSurfaceStates...)
	}
	return nil
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

func (c ConsoleInteractionAudit) ReadinessScore() float64 {
	if c.ContractCount == 0 {
		return 0
	}
	penalties := len(c.MissingSurfaces) + len(c.SurfacesMissingFilters) + len(c.SurfacesMissingActions) +
		len(c.SurfacesMissingBatchActions) + len(c.SurfacesMissingStates) + len(c.PermissionGaps) +
		len(c.UncoveredRoles) + len(c.SurfacesMissingPrimaryPersonas) + len(c.SurfacesMissingWireframeLinks) +
		len(c.SurfacesMissingReviewFocus) + len(c.SurfacesMissingDecisionPrompts)
	score := math.Max(0, 100-(float64(penalties)*100/float64(c.ContractCount)))
	return math.Round(score*10) / 10
}

func (c ConsoleInteractionAudit) ReleaseReady() bool {
	return len(c.MissingSurfaces) == 0 && len(c.SurfacesMissingFilters) == 0 && len(c.SurfacesMissingActions) == 0 &&
		len(c.SurfacesMissingBatchActions) == 0 && len(c.SurfacesMissingStates) == 0 && len(c.PermissionGaps) == 0 &&
		len(c.UncoveredRoles) == 0 && len(c.SurfacesMissingPrimaryPersonas) == 0 && len(c.SurfacesMissingWireframeLinks) == 0 &&
		len(c.SurfacesMissingReviewFocus) == 0 && len(c.SurfacesMissingDecisionPrompts) == 0
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

func (c ConsoleIAAudit) ReadinessScore() float64 {
	if c.SurfaceCount == 0 {
		return 0
	}
	penalties := len(c.SurfacesMissingFilters) + len(c.SurfacesMissingActions) + len(c.SurfacesMissingStates) +
		len(c.StatesMissingActions) + len(c.UnresolvedStateActions) + len(c.OrphanNavigationRoutes) + len(c.UnnavigableSurfaces)
	if !c.TopBarAudit.ReleaseReady() {
		penalties++
	}
	score := math.Max(0, 100-(float64(penalties)*100/float64(c.SurfaceCount)))
	return math.Round(score*10) / 10
}

type ConsoleIAAuditor struct{}

func (ConsoleIAAuditor) Audit(architecture ConsoleIA) ConsoleIAAudit {
	topBarAudit := designsystem.ConsoleChromeLibrary{}.AuditTopBar(architecture.TopBar)
	routeIndex := architecture.RouteIndex()
	navigationRoutes := make([]string, 0, len(architecture.Navigation))
	for _, item := range architecture.Navigation {
		navigationRoutes = append(navigationRoutes, item.Route)
	}
	missingFilters := make([]string, 0)
	missingActions := make([]string, 0)
	missingStates := make(map[string][]string)
	statesMissingActions := make(map[string][]string)
	unresolvedStateActions := make(map[string]map[string][]string)
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
	sort.Strings(missingFilters)
	sort.Strings(missingActions)
	orphanRoutes := make([]string, 0)
	for _, route := range navigationRoutes {
		if _, ok := routeIndex[route]; !ok {
			orphanRoutes = append(orphanRoutes, route)
		}
	}
	sort.Strings(orphanRoutes)
	unnavigable := make([]string, 0)
	for _, surface := range architecture.Surfaces {
		if !contains(navigationRoutes, surface.Route) {
			unnavigable = append(unnavigable, surface.Name)
		}
	}
	sort.Strings(unnavigable)
	return ConsoleIAAudit{
		SystemName:             architecture.Name,
		Version:                architecture.Version,
		SurfaceCount:           len(architecture.Surfaces),
		NavigationCount:        len(architecture.Navigation),
		TopBarAudit:            topBarAudit,
		SurfacesMissingFilters: missingFilters,
		SurfacesMissingActions: missingActions,
		SurfacesMissingStates:  sortMapSliceValues(missingStates),
		StatesMissingActions:   sortMapSliceValues(statesMissingActions),
		UnresolvedStateActions: sortNestedMap(unresolvedStateActions),
		OrphanNavigationRoutes: orphanRoutes,
		UnnavigableSurfaces:    unnavigable,
	}
}

type ConsoleInteractionAuditor struct{}

func (ConsoleInteractionAuditor) Audit(draft ConsoleInteractionDraft) ConsoleInteractionAudit {
	routeIndex := draft.Architecture.RouteIndex()
	missingSurfaces := make([]string, 0)
	missingFilters := make([]string, 0)
	missingActions := make(map[string][]string)
	missingBatch := make([]string, 0)
	missingStates := make(map[string][]string)
	permissionGaps := make(map[string][]string)
	referencedRoles := make([]string, 0)
	missingPersonas := make([]string, 0)
	missingWireframes := make([]string, 0)
	missingReviewFocus := make([]string, 0)
	missingDecisionPrompts := make([]string, 0)

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
		availableActions := surface.ActionIDs()
		actions := make([]string, 0)
		for _, actionID := range contract.RequiredActionIDs {
			if !contains(availableActions, actionID) {
				actions = append(actions, actionID)
			}
		}
		if len(actions) > 0 {
			missingActions[contract.SurfaceName] = actions
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
		stateGaps := make([]string, 0)
		for _, state := range contract.RequiredStates {
			if !contains(surface.StateNames(), state) {
				stateGaps = append(stateGaps, state)
			}
		}
		if len(stateGaps) > 0 {
			missingStates[contract.SurfaceName] = stateGaps
		}
		for _, role := range append(append([]string{}, contract.PermissionRule.AllowedRoles...), contract.PermissionRule.DeniedRoles...) {
			if !contains(referencedRoles, role) {
				referencedRoles = append(referencedRoles, role)
			}
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
				missingReviewFocus = append(missingReviewFocus, contract.SurfaceName)
			}
			if len(contract.DecisionPrompts) == 0 {
				missingDecisionPrompts = append(missingDecisionPrompts, contract.SurfaceName)
			}
		}
	}
	sort.Strings(missingSurfaces)
	sort.Strings(missingFilters)
	sort.Strings(missingBatch)
	sort.Strings(missingPersonas)
	sort.Strings(missingWireframes)
	sort.Strings(missingReviewFocus)
	sort.Strings(missingDecisionPrompts)
	uncoveredRoles := make([]string, 0)
	for _, role := range draft.RequiredRoles {
		if !contains(referencedRoles, role) {
			uncoveredRoles = append(uncoveredRoles, role)
		}
	}
	sort.Strings(uncoveredRoles)
	return ConsoleInteractionAudit{
		Name:                           draft.Name,
		Version:                        draft.Version,
		ContractCount:                  len(draft.Contracts),
		MissingSurfaces:                missingSurfaces,
		SurfacesMissingFilters:         missingFilters,
		SurfacesMissingActions:         sortMapSliceValues(missingActions),
		SurfacesMissingBatchActions:    missingBatch,
		SurfacesMissingStates:          sortMapSliceValues(missingStates),
		PermissionGaps:                 sortMapSliceValues(permissionGaps),
		UncoveredRoles:                 uncoveredRoles,
		SurfacesMissingPrimaryPersonas: missingPersonas,
		SurfacesMissingWireframeLinks:  missingWireframes,
		SurfacesMissingReviewFocus:     missingReviewFocus,
		SurfacesMissingDecisionPrompts: missingDecisionPrompts,
	}
}

func RenderConsoleIAReport(architecture ConsoleIA, audit ConsoleIAAudit) string {
	lines := []string{
		"# Console Information Architecture Report",
		"",
		fmt.Sprintf("- Name: %s", architecture.TopBar.Name),
		fmt.Sprintf("- Version: %s", architecture.Version),
		fmt.Sprintf("- Navigation Items: %d", audit.NavigationCount),
		fmt.Sprintf("- Surfaces: %d", audit.SurfaceCount),
		fmt.Sprintf("- Readiness Score: %.1f", audit.ReadinessScore()),
		"",
		"## Global Header",
		"",
		fmt.Sprintf("- Name: %s", defaultString(architecture.TopBar.Name, "none")),
		fmt.Sprintf("- Release Ready: %t", audit.TopBarAudit.ReleaseReady()),
		fmt.Sprintf("- Missing capabilities: %s", joinedOrNone(audit.TopBarAudit.MissingCapabilities)),
		fmt.Sprintf("- Command Count: %d", audit.TopBarAudit.CommandCount),
		fmt.Sprintf("- Cmd/Ctrl+K supported: %t", audit.TopBarAudit.CommandShortcutSupported),
		"",
		"## Navigation",
		"",
	}
	if len(architecture.Navigation) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, item := range architecture.Navigation {
			lines = append(lines, fmt.Sprintf("- %s / %s: route=%s badge=%d icon=%s", item.Section, item.Name, item.Route, item.BadgeCount, defaultString(item.Icon, "none")))
		}
	}
	lines = append(lines, "", "## Surface Coverage", "")
	if len(architecture.Surfaces) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, surface := range architecture.Surfaces {
			filters := make([]string, 0, len(surface.Filters))
			for _, filter := range surface.Filters {
				filters = append(filters, filter.Name)
			}
			actions := make([]string, 0, len(surface.TopBarActions))
			for _, action := range surface.TopBarActions {
				actions = append(actions, action.Label)
			}
			unresolved := audit.UnresolvedStateActions[surface.Name]
			unresolvedText := "none"
			if len(unresolved) > 0 {
				parts := make([]string, 0, len(unresolved))
				for _, key := range sortedMapKeys(unresolved) {
					parts = append(parts, fmt.Sprintf("%s=%s", key, strings.Join(unresolved[key], ", ")))
				}
				unresolvedText = strings.Join(parts, "; ")
			}
			lines = append(lines, fmt.Sprintf("- %s: route=%s filters=%s actions=%s states=%s missing_states=%s states_without_actions=%s unresolved_state_actions=%s",
				surface.Name, surface.Route, joinedOrNone(filters), joinedOrNone(actions), joinedOrNone(surface.StateNames()), joinedOrNone(surface.MissingRequiredStates()), joinedOrNone(audit.StatesMissingActions[surface.Name]), unresolvedText))
		}
	}
	lines = append(lines, "", "## Gaps", "")
	lines = append(lines, fmt.Sprintf("- Surfaces missing filters: %s", joinedOrNone(audit.SurfacesMissingFilters)))
	lines = append(lines, fmt.Sprintf("- Surfaces missing top-bar actions: %s", joinedOrNone(audit.SurfacesMissingActions)))
	lines = append(lines, fmt.Sprintf("- Surfaces missing required states: %s", joinedNamedMap(audit.SurfacesMissingStates)))
	lines = append(lines, fmt.Sprintf("- States without recovery actions: %s", joinedNamedMap(audit.StatesMissingActions)))
	if len(audit.UnresolvedStateActions) == 0 {
		lines = append(lines, "- Undefined state actions: none")
	} else {
		parts := make([]string, 0)
		for _, surface := range sortedMapKeys(audit.UnresolvedStateActions) {
			stateParts := make([]string, 0)
			for _, state := range sortedMapKeys(audit.UnresolvedStateActions[surface]) {
				stateParts = append(stateParts, fmt.Sprintf("%s:%s", state, strings.Join(audit.UnresolvedStateActions[surface][state], "/")))
			}
			parts = append(parts, fmt.Sprintf("%s=%s", surface, strings.Join(stateParts, ", ")))
		}
		lines = append(lines, fmt.Sprintf("- Undefined state actions: %s", strings.Join(parts, "; ")))
	}
	lines = append(lines, fmt.Sprintf("- Orphan navigation routes: %s", joinedOrNone(audit.OrphanNavigationRoutes)))
	lines = append(lines, fmt.Sprintf("- Unnavigable surfaces: %s", joinedOrNone(audit.UnnavigableSurfaces)))
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
		fmt.Sprintf("- Required Roles: %s", joinedOrNone(draft.RequiredRoles)),
		fmt.Sprintf("- Readiness Score: %.1f", audit.ReadinessScore()),
		fmt.Sprintf("- Release Ready: %t", audit.ReleaseReady()),
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
			lines = append(lines, fmt.Sprintf("- %s: route=%s required_actions=%s available_actions=%s filters=%d states=%s batch=%s permissions=%s",
				contract.SurfaceName, surface.Route, joinedOrNone(contract.RequiredActionIDs), joinedOrNone(surface.ActionIDs()), len(surface.Filters), joinedOrNone(surface.StateNames()),
				map[bool]string{true: "required", false: "optional"}[contract.RequiresBatchActions],
				map[bool]string{true: "complete", false: "incomplete"}[contract.PermissionRule.Complete()]))
			lines = append(lines, "  "+fmt.Sprintf("persona=%s wireframe=%s review_focus=%s decision_prompts=%s",
				defaultString(contract.PrimaryPersona, "none"), defaultString(contract.LinkedWireframeID, "none"),
				joinedCSVOrNone(contract.ReviewFocusAreas), joinedCSVOrNone(contract.DecisionPrompts)))
		}
	}
	lines = append(lines, "", "## Gaps", "")
	lines = append(lines, fmt.Sprintf("- Missing surfaces: %s", joinedOrNone(audit.MissingSurfaces)))
	lines = append(lines, fmt.Sprintf("- Pages missing filters: %s", joinedOrNone(audit.SurfacesMissingFilters)))
	lines = append(lines, fmt.Sprintf("- Pages missing actions: %s", joinedNamedMap(audit.SurfacesMissingActions)))
	lines = append(lines, fmt.Sprintf("- Pages missing batch actions: %s", joinedOrNone(audit.SurfacesMissingBatchActions)))
	lines = append(lines, fmt.Sprintf("- Pages missing states: %s", joinedNamedMap(audit.SurfacesMissingStates)))
	lines = append(lines, fmt.Sprintf("- Permission gaps: %s", joinedNamedMap(audit.PermissionGaps)))
	lines = append(lines, fmt.Sprintf("- Uncovered roles: %s", joinedOrNone(audit.UncoveredRoles)))
	lines = append(lines, fmt.Sprintf("- Pages missing personas: %s", joinedOrNone(audit.SurfacesMissingPrimaryPersonas)))
	lines = append(lines, fmt.Sprintf("- Pages missing wireframe links: %s", joinedOrNone(audit.SurfacesMissingWireframeLinks)))
	lines = append(lines, fmt.Sprintf("- Pages missing review focus: %s", joinedOrNone(audit.SurfacesMissingReviewFocus)))
	lines = append(lines, fmt.Sprintf("- Pages missing decision prompts: %s", joinedOrNone(audit.SurfacesMissingDecisionPrompts)))
	return strings.Join(lines, "\n") + "\n"
}

func BuildBIG4203ConsoleInteractionDraft() ConsoleInteractionDraft {
	return ConsoleInteractionDraft{
		Name:                   "BIG-4203 Four Critical Pages",
		Version:                "v4.0-design-sprint",
		RequiredRoles:          []string{"eng-lead", "platform-admin", "vp-eng", "cross-team-operator"},
		RequiresFrameContracts: true,
		Architecture: ConsoleIA{
			Name:    "BigClaw Console IA",
			Version: "v4.0-design-sprint",
			TopBar: designsystem.ConsoleTopBar{
				Name:                      "BigClaw Global Header",
				SearchPlaceholder:         "Search runs, queues, prompts, and commands",
				EnvironmentOptions:        []string{"Production", "Staging", "Shadow"},
				TimeRangeOptions:          []string{"24h", "7d", "30d"},
				AlertChannels:             []string{"approvals", "sla", "regressions"},
				DocumentationComplete:     true,
				AccessibilityRequirements: []string{"keyboard-navigation", "screen-reader-label", "focus-visible"},
				CommandEntry: designsystem.ConsoleCommandEntry{
					TriggerLabel: "Command Menu",
					Placeholder:  "Jump to a run, queue, or release control action",
					Shortcut:     "Cmd+K / Ctrl+K",
					Commands: []designsystem.CommandAction{
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
				{Name: "Overview", Route: "/overview", NavigationSection: "Operate",
					TopBarActions: []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}},
					Filters:       []FilterDefinition{{Name: "Team", Field: "team", Control: "select", Options: []string{"all", "platform", "product"}}, {Name: "Time", Field: "time_range", Control: "segmented", Options: []string{"24h", "7d", "30d"}, DefaultValue: "7d"}},
					States:        []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"export"}}, {Name: "empty", AllowedActions: []string{"drill-down"}}, {Name: "error", AllowedActions: []string{"audit"}}},
				},
				{Name: "Queue", Route: "/queue", NavigationSection: "Operate", SupportsBulkActions: true,
					TopBarActions: []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}, {ActionID: "bulk-approve", Label: "Bulk Approve", Placement: "topbar", RequiresSelection: true}},
					Filters:       []FilterDefinition{{Name: "Status", Field: "status", Control: "select", Options: []string{"all", "queued", "approval"}}, {Name: "Owner", Field: "owner", Control: "search"}},
					States:        []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"export"}}, {Name: "empty", AllowedActions: []string{"audit"}}, {Name: "error", AllowedActions: []string{"audit"}}},
				},
				{Name: "Run Detail", Route: "/runs/detail", NavigationSection: "Operate",
					TopBarActions: []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}},
					Filters:       []FilterDefinition{{Name: "Run", Field: "run_id", Control: "search"}, {Name: "Replay Mode", Field: "replay_mode", Control: "select", Options: []string{"latest", "failure-only"}}},
					States:        []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"export"}}, {Name: "empty", AllowedActions: []string{"drill-down"}}, {Name: "error", AllowedActions: []string{"audit"}}},
				},
				{Name: "Triage", Route: "/triage", NavigationSection: "Operate", SupportsBulkActions: true,
					TopBarActions: []GlobalAction{{ActionID: "drill-down", Label: "Drill Down", Placement: "topbar"}, {ActionID: "export", Label: "Export", Placement: "topbar"}, {ActionID: "audit", Label: "Audit Trail", Placement: "topbar"}, {ActionID: "bulk-assign", Label: "Bulk Assign", Placement: "topbar", RequiresSelection: true}},
					Filters:       []FilterDefinition{{Name: "Severity", Field: "severity", Control: "select", Options: []string{"all", "high", "critical"}}, {Name: "Workflow", Field: "workflow", Control: "select", Options: []string{"all", "triage", "handoff"}}},
					States:        []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"export"}}, {Name: "empty", AllowedActions: []string{"audit"}}, {Name: "error", AllowedActions: []string{"audit"}}},
				},
			},
		},
		Contracts: []SurfaceInteractionContract{
			{SurfaceName: "Overview", RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiresFilters: true, RequiredStates: append([]string(nil), requiredSurfaceStates...), PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"eng-lead", "platform-admin", "vp-eng", "cross-team-operator"}, DeniedRoles: []string{"guest"}, AuditEvent: "overview.access.denied"}, PrimaryPersona: "VP Eng", LinkedWireframeID: "wf-overview", ReviewFocusAreas: []string{"metric hierarchy", "drill-down posture", "alert prioritization"}, DecisionPrompts: []string{"Is the executive KPI density still scannable within one screen?", "Do risk and blocker cards point to the correct downstream investigation surface?"}},
			{SurfaceName: "Queue", RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiresFilters: true, RequiresBatchActions: true, RequiredStates: append([]string(nil), requiredSurfaceStates...), PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"eng-lead", "platform-admin", "cross-team-operator"}, DeniedRoles: []string{"vp-eng", "guest"}, AuditEvent: "queue.access.denied"}, PrimaryPersona: "Platform Admin", LinkedWireframeID: "wf-queue", ReviewFocusAreas: []string{"batch approvals", "denied-role state", "audit rail"}, DecisionPrompts: []string{"Does the queue clearly separate selection, confirmation, and audit outcomes?", "Is the denied-role treatment explicit enough for VP Eng and guest personas?"}},
			{SurfaceName: "Run Detail", RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiresFilters: true, RequiredStates: append([]string(nil), requiredSurfaceStates...), PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"eng-lead", "platform-admin", "vp-eng", "cross-team-operator"}, DeniedRoles: []string{"guest"}, AuditEvent: "run-detail.access.denied"}, PrimaryPersona: "Eng Lead", LinkedWireframeID: "wf-run-detail", ReviewFocusAreas: []string{"replay context", "artifact evidence", "escalation path"}, DecisionPrompts: []string{"Can reviewers distinguish replay, compare, and escalated states without narration?", "Is the audit trail visible at the moment an escalation decision is made?"}},
			{SurfaceName: "Triage", RequiredActionIDs: []string{"drill-down", "export", "audit"}, RequiresFilters: true, RequiresBatchActions: true, RequiredStates: append([]string(nil), requiredSurfaceStates...), PermissionRule: SurfacePermissionRule{AllowedRoles: []string{"eng-lead", "platform-admin", "cross-team-operator"}, DeniedRoles: []string{"vp-eng", "guest"}, AuditEvent: "triage.access.denied"}, PrimaryPersona: "Cross-Team Operator", LinkedWireframeID: "wf-triage", ReviewFocusAreas: []string{"handoff path", "bulk assignment", "ownership history"}, DecisionPrompts: []string{"Does the triage frame explain handoff consequences before ownership changes commit?", "Is bulk assignment discoverable without overpowering the audit context?"}},
		},
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func joinedOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func joinedCSVOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ",")
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func sortMapSliceValues(input map[string][]string) map[string][]string {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string][]string, len(input))
	for key, value := range input {
		items := append([]string(nil), value...)
		sort.Strings(items)
		out[key] = items
	}
	return out
}

func sortNestedMap(input map[string]map[string][]string) map[string]map[string][]string {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]map[string][]string, len(input))
	for key, nested := range input {
		out[key] = sortMapSliceValues(nested)
	}
	return out
}

func joinedNamedMap(input map[string][]string) string {
	if len(input) == 0 {
		return "none"
	}
	keys := sortedMapKeys(input)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, strings.Join(input[key], ", ")))
	}
	return strings.Join(parts, "; ")
}

func sortedMapKeys[V any](input map[string]V) []string {
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func MustJSONRoundTrip[T any](value T) T {
	body, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	var out T
	if err := json.Unmarshal(body, &out); err != nil {
		panic(err)
	}
	return out
}
