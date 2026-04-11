package consoleia

import (
	"fmt"
	"math"
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
	Intent            string `json:"intent,omitempty"`
}

func (action GlobalAction) withDefaults() GlobalAction {
	if action.Intent == "" {
		action.Intent = "default"
	}
	return action
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

func (surface ConsoleSurface) ActionIDs() []string {
	ids := make([]string, 0, len(surface.TopBarActions))
	for _, action := range surface.TopBarActions {
		ids = append(ids, action.ActionID)
	}
	return ids
}

func (surface ConsoleSurface) StateNames() []string {
	names := make([]string, 0, len(surface.States))
	for _, state := range surface.States {
		names = append(names, state.Name)
	}
	return names
}

func (surface ConsoleSurface) MissingRequiredStates() []string {
	stateSet := map[string]struct{}{}
	for _, state := range surface.StateNames() {
		stateSet[state] = struct{}{}
	}

	missing := make([]string, 0)
	for _, state := range requiredSurfaceStates {
		if _, ok := stateSet[state]; ok {
			continue
		}
		missing = append(missing, state)
	}
	slices.Sort(missing)
	return missing
}

func (surface ConsoleSurface) UnresolvedStateActions() map[string][]string {
	available := sliceSet(surface.ActionIDs())
	unresolved := map[string][]string{}
	for _, state := range surface.States {
		missing := make([]string, 0)
		for _, actionID := range state.AllowedActions {
			if _, ok := available[actionID]; ok {
				continue
			}
			missing = append(missing, actionID)
		}
		slices.Sort(missing)
		if len(missing) > 0 {
			unresolved[state.Name] = missing
		}
	}
	return unresolved
}

func (surface ConsoleSurface) StatesMissingActions() []string {
	missing := make([]string, 0)
	for _, state := range surface.States {
		if state.Name != "default" && len(state.AllowedActions) == 0 {
			missing = append(missing, state.Name)
		}
	}
	return missing
}

type ConsoleIA struct {
	Name       string                     `json:"name"`
	Version    string                     `json:"version"`
	Navigation []NavigationItem           `json:"navigation,omitempty"`
	Surfaces   []ConsoleSurface           `json:"surfaces,omitempty"`
	TopBar     designsystem.ConsoleTopBar `json:"top_bar"`
}

func (architecture ConsoleIA) RouteIndex() map[string]ConsoleSurface {
	index := make(map[string]ConsoleSurface, len(architecture.Surfaces))
	for _, surface := range architecture.Surfaces {
		index[surface.Route] = surface
	}
	return index
}

type SurfacePermissionRule struct {
	AllowedRoles []string `json:"allowed_roles,omitempty"`
	DeniedRoles  []string `json:"denied_roles,omitempty"`
	AuditEvent   string   `json:"audit_event,omitempty"`
}

func (rule SurfacePermissionRule) MissingCoverage() []string {
	missing := make([]string, 0)
	if len(rule.AllowedRoles) == 0 {
		missing = append(missing, "allowed-roles")
	}
	if len(rule.DeniedRoles) == 0 {
		missing = append(missing, "denied-roles")
	}
	if strings.TrimSpace(rule.AuditEvent) == "" {
		missing = append(missing, "audit-event")
	}
	return missing
}

func (rule SurfacePermissionRule) Complete() bool {
	return len(rule.MissingCoverage()) == 0
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

func (contract SurfaceInteractionContract) requiredStatesOrDefault() []string {
	if len(contract.RequiredStates) == 0 {
		return slices.Clone(requiredSurfaceStates)
	}
	return contract.RequiredStates
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

func (audit ConsoleInteractionAudit) ReadinessScore() float64 {
	if audit.ContractCount == 0 {
		return 0
	}
	penalties := len(audit.MissingSurfaces) +
		len(audit.SurfacesMissingFilters) +
		len(audit.SurfacesMissingActions) +
		len(audit.SurfacesMissingBatchActions) +
		len(audit.SurfacesMissingStates) +
		len(audit.PermissionGaps) +
		len(audit.UncoveredRoles) +
		len(audit.SurfacesMissingPrimaryPersonas) +
		len(audit.SurfacesMissingWireframeLinks) +
		len(audit.SurfacesMissingReviewFocus) +
		len(audit.SurfacesMissingDecisionPrompts)
	score := math.Max(0, 100-((float64(penalties)*100)/float64(audit.ContractCount)))
	return round1(score)
}

func (audit ConsoleInteractionAudit) ReleaseReady() bool {
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

func (audit ConsoleIAAudit) ReadinessScore() float64 {
	if audit.SurfaceCount == 0 {
		return 0
	}
	penalties := len(audit.SurfacesMissingFilters) +
		len(audit.SurfacesMissingActions) +
		len(audit.SurfacesMissingStates) +
		len(audit.StatesMissingActions) +
		len(audit.UnresolvedStateActions) +
		len(audit.OrphanNavigationRoutes) +
		len(audit.UnnavigableSurfaces)
	if !audit.TopBarAudit.ReleaseReady() {
		penalties++
	}
	score := math.Max(0, 100-((float64(penalties)*100)/float64(audit.SurfaceCount)))
	return round1(score)
}

type ConsoleIAAuditor struct{}

func (ConsoleIAAuditor) Audit(architecture ConsoleIA) ConsoleIAAudit {
	topBarAudit := designsystem.ConsoleChromeLibrary{}.AuditTopBar(architecture.TopBar)
	routeIndex := architecture.RouteIndex()
	navigationRoutes := map[string]struct{}{}
	for _, item := range architecture.Navigation {
		navigationRoutes[item.Route] = struct{}{}
	}

	surfacesMissingFilters := make([]string, 0)
	surfacesMissingActions := make([]string, 0)
	surfacesMissingStates := map[string][]string{}
	statesMissingActions := map[string][]string{}
	unresolvedStateActions := map[string]map[string][]string{}

	for _, surface := range architecture.Surfaces {
		if len(surface.Filters) == 0 {
			surfacesMissingFilters = append(surfacesMissingFilters, surface.Name)
		}
		if len(surface.TopBarActions) == 0 {
			surfacesMissingActions = append(surfacesMissingActions, surface.Name)
		}
		if missing := surface.MissingRequiredStates(); len(missing) > 0 {
			surfacesMissingStates[surface.Name] = missing
		}
		if missing := surface.StatesMissingActions(); len(missing) > 0 {
			statesMissingActions[surface.Name] = missing
		}
		if unresolved := surface.UnresolvedStateActions(); len(unresolved) > 0 {
			unresolvedStateActions[surface.Name] = unresolved
		}
	}

	orphanNavigationRoutes := make([]string, 0)
	for route := range navigationRoutes {
		if _, ok := routeIndex[route]; ok {
			continue
		}
		orphanNavigationRoutes = append(orphanNavigationRoutes, route)
	}
	slices.Sort(orphanNavigationRoutes)

	unnavigableSurfaces := make([]string, 0)
	for _, surface := range architecture.Surfaces {
		if _, ok := navigationRoutes[surface.Route]; ok {
			continue
		}
		unnavigableSurfaces = append(unnavigableSurfaces, surface.Name)
	}
	slices.Sort(unnavigableSurfaces)
	slices.Sort(surfacesMissingFilters)
	slices.Sort(surfacesMissingActions)

	return ConsoleIAAudit{
		SystemName:             architecture.Name,
		Version:                architecture.Version,
		SurfaceCount:           len(architecture.Surfaces),
		NavigationCount:        len(architecture.Navigation),
		TopBarAudit:            topBarAudit,
		SurfacesMissingFilters: surfacesMissingFilters,
		SurfacesMissingActions: surfacesMissingActions,
		SurfacesMissingStates:  sortMapValues(surfacesMissingStates),
		StatesMissingActions:   sortMapValues(statesMissingActions),
		UnresolvedStateActions: sortNestedMapValues(unresolvedStateActions),
		OrphanNavigationRoutes: orphanNavigationRoutes,
		UnnavigableSurfaces:    unnavigableSurfaces,
	}
}

type ConsoleInteractionAuditor struct{}

func (ConsoleInteractionAuditor) Audit(draft ConsoleInteractionDraft) ConsoleInteractionAudit {
	routeIndex := draft.Architecture.RouteIndex()
	missingSurfaces := make([]string, 0)
	surfacesMissingFilters := make([]string, 0)
	surfacesMissingActions := map[string][]string{}
	surfacesMissingBatchActions := make([]string, 0)
	surfacesMissingStates := map[string][]string{}
	permissionGaps := map[string][]string{}
	referencedRoles := map[string]struct{}{}
	surfacesMissingPrimaryPersonas := make([]string, 0)
	surfacesMissingWireframeLinks := make([]string, 0)
	surfacesMissingReviewFocus := make([]string, 0)
	surfacesMissingDecisionPrompts := make([]string, 0)

	for _, contract := range draft.Contracts {
		surface, ok := routeIndex[contract.SurfaceName]
		if !ok {
			surface, ok = findSurfaceByName(draft.Architecture.Surfaces, contract.SurfaceName)
		}
		if !ok {
			missingSurfaces = append(missingSurfaces, contract.SurfaceName)
			continue
		}

		if contract.RequiresFilters && len(surface.Filters) == 0 {
			surfacesMissingFilters = append(surfacesMissingFilters, contract.SurfaceName)
		}

		availableActionIDs := sliceSet(surface.ActionIDs())
		missingActionIDs := make([]string, 0)
		for _, actionID := range contract.RequiredActionIDs {
			if _, ok := availableActionIDs[actionID]; ok {
				continue
			}
			missingActionIDs = append(missingActionIDs, actionID)
		}
		slices.Sort(missingActionIDs)
		if len(missingActionIDs) > 0 {
			surfacesMissingActions[contract.SurfaceName] = missingActionIDs
		}

		if contract.RequiresBatchActions {
			hasSelectionAction := false
			for _, action := range surface.TopBarActions {
				if action.RequiresSelection {
					hasSelectionAction = true
					break
				}
			}
			if !hasSelectionAction {
				surfacesMissingBatchActions = append(surfacesMissingBatchActions, contract.SurfaceName)
			}
		}

		missingStates := make([]string, 0)
		surfaceStates := sliceSet(surface.StateNames())
		for _, stateName := range contract.requiredStatesOrDefault() {
			if _, ok := surfaceStates[stateName]; ok {
				continue
			}
			missingStates = append(missingStates, stateName)
		}
		slices.Sort(missingStates)
		if len(missingStates) > 0 {
			surfacesMissingStates[contract.SurfaceName] = missingStates
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
		if _, ok := referencedRoles[role]; ok {
			continue
		}
		uncoveredRoles = append(uncoveredRoles, role)
	}
	slices.Sort(uncoveredRoles)
	slices.Sort(missingSurfaces)
	slices.Sort(surfacesMissingFilters)
	slices.Sort(surfacesMissingBatchActions)
	slices.Sort(surfacesMissingPrimaryPersonas)
	slices.Sort(surfacesMissingWireframeLinks)
	slices.Sort(surfacesMissingReviewFocus)
	slices.Sort(surfacesMissingDecisionPrompts)

	return ConsoleInteractionAudit{
		Name:                           draft.Name,
		Version:                        draft.Version,
		ContractCount:                  len(draft.Contracts),
		MissingSurfaces:                missingSurfaces,
		SurfacesMissingFilters:         surfacesMissingFilters,
		SurfacesMissingActions:         sortMapValues(surfacesMissingActions),
		SurfacesMissingBatchActions:    surfacesMissingBatchActions,
		SurfacesMissingStates:          sortMapValues(surfacesMissingStates),
		PermissionGaps:                 sortMapValues(permissionGaps),
		UncoveredRoles:                 uncoveredRoles,
		SurfacesMissingPrimaryPersonas: surfacesMissingPrimaryPersonas,
		SurfacesMissingWireframeLinks:  surfacesMissingWireframeLinks,
		SurfacesMissingReviewFocus:     surfacesMissingReviewFocus,
		SurfacesMissingDecisionPrompts: surfacesMissingDecisionPrompts,
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
		fmt.Sprintf("- Name: %s", noneIfEmpty(architecture.TopBar.Name)),
		fmt.Sprintf("- Release Ready: %s", titleBool(audit.TopBarAudit.ReleaseReady())),
		fmt.Sprintf("- Missing capabilities: %s", joinOrNone(audit.TopBarAudit.MissingCapabilities)),
		fmt.Sprintf("- Command Count: %d", audit.TopBarAudit.CommandCount),
		fmt.Sprintf("- Cmd/Ctrl+K supported: %s", titleBool(audit.TopBarAudit.CommandShortcutSupported)),
		"",
		"## Navigation",
		"",
	}
	if len(architecture.Navigation) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, item := range architecture.Navigation {
			lines = append(lines, fmt.Sprintf("- %s / %s: route=%s badge=%d icon=%s", item.Section, item.Name, item.Route, item.BadgeCount, noneIfEmpty(item.Icon)))
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
			unresolvedText := "none"
			if unresolved := audit.UnresolvedStateActions[surface.Name]; len(unresolved) > 0 {
				parts := make([]string, 0, len(unresolved))
				keys := make([]string, 0, len(unresolved))
				for state := range unresolved {
					keys = append(keys, state)
				}
				slices.Sort(keys)
				for _, state := range keys {
					parts = append(parts, fmt.Sprintf("%s=%s", state, strings.Join(unresolved[state], ", ")))
				}
				unresolvedText = strings.Join(parts, "; ")
			}
			lines = append(lines, fmt.Sprintf(
				"- %s: route=%s filters=%s actions=%s states=%s missing_states=%s states_without_actions=%s unresolved_state_actions=%s",
				surface.Name,
				surface.Route,
				joinOrNone(filters),
				joinOrNone(actions),
				joinOrNone(surface.StateNames()),
				joinOrNone(surface.MissingRequiredStates()),
				joinOrNone(audit.StatesMissingActions[surface.Name]),
				unresolvedText,
			))
		}
	}

	lines = append(lines, "", "## Gaps", "")
	lines = append(lines, fmt.Sprintf("- Surfaces missing filters: %s", joinOrNone(audit.SurfacesMissingFilters)))
	lines = append(lines, fmt.Sprintf("- Surfaces missing top-bar actions: %s", joinOrNone(audit.SurfacesMissingActions)))
	lines = append(lines, fmt.Sprintf("- Surfaces missing required states: %s", formatStringSlicesMap(audit.SurfacesMissingStates)))
	lines = append(lines, fmt.Sprintf("- States without recovery actions: %s", formatStringSlicesMap(audit.StatesMissingActions)))
	lines = append(lines, fmt.Sprintf("- Undefined state actions: %s", formatUnresolvedStateActions(audit.UnresolvedStateActions)))
	lines = append(lines, fmt.Sprintf("- Orphan navigation routes: %s", joinOrNone(audit.OrphanNavigationRoutes)))
	lines = append(lines, fmt.Sprintf("- Unnavigable surfaces: %s", joinOrNone(audit.UnnavigableSurfaces)))
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
		fmt.Sprintf("- Release Ready: %s", titleBool(audit.ReleaseReady())),
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
				surface, ok = findSurfaceByName(draft.Architecture.Surfaces, contract.SurfaceName)
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
				noneIfEmpty(contract.PrimaryPersona),
				noneIfEmpty(contract.LinkedWireframeID),
				joinCommaOrNone(contract.ReviewFocusAreas),
				joinCommaOrNone(contract.DecisionPrompts),
			))
		}
	}

	lines = append(lines, "", "## Gaps", "")
	lines = append(lines, fmt.Sprintf("- Missing surfaces: %s", joinOrNone(audit.MissingSurfaces)))
	lines = append(lines, fmt.Sprintf("- Pages missing filters: %s", joinOrNone(audit.SurfacesMissingFilters)))
	lines = append(lines, fmt.Sprintf("- Pages missing actions: %s", formatStringSlicesMap(audit.SurfacesMissingActions)))
	lines = append(lines, fmt.Sprintf("- Pages missing batch actions: %s", joinOrNone(audit.SurfacesMissingBatchActions)))
	lines = append(lines, fmt.Sprintf("- Pages missing states: %s", formatStringSlicesMap(audit.SurfacesMissingStates)))
	lines = append(lines, fmt.Sprintf("- Permission gaps: %s", formatStringSlicesMap(audit.PermissionGaps)))
	lines = append(lines, fmt.Sprintf("- Uncovered roles: %s", joinOrNone(audit.UncoveredRoles)))
	lines = append(lines, fmt.Sprintf("- Pages missing personas: %s", joinOrNone(audit.SurfacesMissingPrimaryPersonas)))
	lines = append(lines, fmt.Sprintf("- Pages missing wireframe links: %s", joinOrNone(audit.SurfacesMissingWireframeLinks)))
	lines = append(lines, fmt.Sprintf("- Pages missing review focus: %s", joinOrNone(audit.SurfacesMissingReviewFocus)))
	lines = append(lines, fmt.Sprintf("- Pages missing decision prompts: %s", joinOrNone(audit.SurfacesMissingDecisionPrompts)))
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
			TopBar: designsystem.ConsoleTopBar{
				Name:              "BigClaw Global Header",
				SearchPlaceholder: "Search runs, queues, prompts, and commands",
				EnvironmentOptions: []string{
					"Production",
					"Staging",
					"Shadow",
				},
				TimeRangeOptions:      []string{"24h", "7d", "30d"},
				AlertChannels:         []string{"approvals", "sla", "regressions"},
				DocumentationComplete: true,
				AccessibilityRequirements: []string{
					"keyboard-navigation",
					"screen-reader-label",
					"focus-visible",
				},
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
				PermissionRule: SurfacePermissionRule{
					AllowedRoles: []string{"eng-lead", "platform-admin", "vp-eng", "cross-team-operator"},
					DeniedRoles:  []string{"guest"},
					AuditEvent:   "overview.access.denied",
				},
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
				PermissionRule: SurfacePermissionRule{
					AllowedRoles: []string{"eng-lead", "platform-admin", "cross-team-operator"},
					DeniedRoles:  []string{"vp-eng", "guest"},
					AuditEvent:   "queue.access.denied",
				},
				PrimaryPersona:    "Platform Admin",
				LinkedWireframeID: "wf-queue",
				ReviewFocusAreas:  []string{"batch approvals", "denied-role state", "audit rail"},
				DecisionPrompts: []string{
					"Does the queue clearly separate selection, confirmation, and audit outcomes?",
					"Is the denied-role treatment explicit enough for VP Eng and guest personas?",
				},
			},
			{
				SurfaceName:       "Run Detail",
				RequiredActionIDs: []string{"drill-down", "export", "audit"},
				RequiresFilters:   true,
				PermissionRule: SurfacePermissionRule{
					AllowedRoles: []string{"eng-lead", "platform-admin", "vp-eng", "cross-team-operator"},
					DeniedRoles:  []string{"guest"},
					AuditEvent:   "run-detail.access.denied",
				},
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
				PermissionRule: SurfacePermissionRule{
					AllowedRoles: []string{"eng-lead", "platform-admin", "cross-team-operator"},
					DeniedRoles:  []string{"vp-eng", "guest"},
					AuditEvent:   "triage.access.denied",
				},
				PrimaryPersona:    "Cross-Team Operator",
				LinkedWireframeID: "wf-triage",
				ReviewFocusAreas:  []string{"handoff path", "bulk assignment", "ownership history"},
				DecisionPrompts: []string{
					"Does the triage frame explain handoff consequences before ownership changes commit?",
					"Is bulk assignment discoverable without overpowering the audit context?",
				},
			},
		},
	}
}

func findSurfaceByName(surfaces []ConsoleSurface, name string) (ConsoleSurface, bool) {
	for _, candidate := range surfaces {
		if candidate.Name == name {
			return candidate, true
		}
	}
	return ConsoleSurface{}, false
}

func sortMapValues(items map[string][]string) map[string][]string {
	if len(items) == 0 {
		return items
	}
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	out := make(map[string][]string, len(items))
	for _, key := range keys {
		values := slices.Clone(items[key])
		slices.Sort(values)
		out[key] = values
	}
	return out
}

func sortNestedMapValues(items map[string]map[string][]string) map[string]map[string][]string {
	if len(items) == 0 {
		return items
	}
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	out := make(map[string]map[string][]string, len(items))
	for _, key := range keys {
		nested := make(map[string][]string, len(items[key]))
		nestedKeys := make([]string, 0, len(items[key]))
		for nestedKey := range items[key] {
			nestedKeys = append(nestedKeys, nestedKey)
		}
		slices.Sort(nestedKeys)
		for _, nestedKey := range nestedKeys {
			values := slices.Clone(items[key][nestedKey])
			slices.Sort(values)
			nested[nestedKey] = values
		}
		out[key] = nested
	}
	return out
}

func formatStringSlicesMap(items map[string][]string) string {
	if len(items) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, strings.Join(items[key], ", ")))
	}
	return strings.Join(parts, "; ")
}

func formatUnresolvedStateActions(items map[string]map[string][]string) string {
	if len(items) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		stateKeys := make([]string, 0, len(items[key]))
		for state := range items[key] {
			stateKeys = append(stateKeys, state)
		}
		slices.Sort(stateKeys)
		stateParts := make([]string, 0, len(stateKeys))
		for _, state := range stateKeys {
			stateParts = append(stateParts, fmt.Sprintf("%s:%s", state, strings.Join(items[key][state], "/")))
		}
		parts = append(parts, fmt.Sprintf("%s=%s", key, strings.Join(stateParts, ", ")))
	}
	return strings.Join(parts, "; ")
}

func joinOrNone(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}

func joinCommaOrNone(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ",")
}

func noneIfEmpty(value string) string {
	if strings.TrimSpace(value) == "" {
		return "none"
	}
	return value
}

func sliceSet(items []string) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, item := range items {
		out[item] = struct{}{}
	}
	return out
}

func titleBool(value bool) string {
	if value {
		return "True"
	}
	return "False"
}

func round1(value float64) float64 {
	return math.Round(value*10) / 10
}
