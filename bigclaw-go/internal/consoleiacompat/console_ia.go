package consoleiacompat

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"bigclaw-go/internal/designsystemcompat"
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
	have := make(map[string]struct{}, len(s.States))
	for _, state := range s.States {
		have[state.Name] = struct{}{}
	}
	out := make([]string, 0)
	for _, state := range requiredSurfaceStates {
		if _, ok := have[state]; !ok {
			out = append(out, state)
		}
	}
	return out
}

func (s ConsoleSurface) UnresolvedStateActions() map[string][]string {
	available := make(map[string]struct{}, len(s.TopBarActions))
	for _, action := range s.TopBarActions {
		available[action.ActionID] = struct{}{}
	}
	out := map[string][]string{}
	for _, state := range s.States {
		missing := make([]string, 0)
		for _, actionID := range state.AllowedActions {
			if _, ok := available[actionID]; !ok {
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

func (s ConsoleSurface) StatesMissingActions() []string {
	out := make([]string, 0)
	for _, state := range s.States {
		if state.Name != "default" && len(state.AllowedActions) == 0 {
			out = append(out, state.Name)
		}
	}
	return out
}

type ConsoleIA struct {
	Name       string                           `json:"name"`
	Version    string                           `json:"version"`
	Navigation []NavigationItem                 `json:"navigation,omitempty"`
	Surfaces   []ConsoleSurface                 `json:"surfaces,omitempty"`
	TopBar     designsystemcompat.ConsoleTopBar `json:"top_bar"`
}

func (a ConsoleIA) ToMap() map[string]any {
	return mustToMap(a)
}

func ConsoleIAFromMap(data map[string]any) ConsoleIA {
	var out ConsoleIA
	mustFromMap(data, &out)
	return out
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
	out := make([]string, 0)
	if len(r.AllowedRoles) == 0 {
		out = append(out, "allowed-roles")
	}
	if len(r.DeniedRoles) == 0 {
		out = append(out, "denied-roles")
	}
	if strings.TrimSpace(r.AuditEvent) == "" {
		out = append(out, "audit-event")
	}
	return out
}

func (r SurfacePermissionRule) Complete() bool {
	return len(r.MissingCoverage()) == 0
}

type SurfaceInteractionContract struct {
	SurfaceName          string                `json:"surface_name"`
	RequiredActionIDs    []string              `json:"required_action_ids,omitempty"`
	RequiresFilters      bool                  `json:"requires_filters,omitempty"`
	RequiresBatchActions bool                  `json:"requires_batch_actions,omitempty"`
	RequiredStates       []string              `json:"required_states,omitempty"`
	PermissionRule       SurfacePermissionRule `json:"permission_rule"`
	PrimaryPersona       string                `json:"primary_persona,omitempty"`
	LinkedWireframeID    string                `json:"linked_wireframe_id,omitempty"`
	ReviewFocusAreas     []string              `json:"review_focus_areas,omitempty"`
	DecisionPrompts      []string              `json:"decision_prompts,omitempty"`
}

func NewSurfaceInteractionContract(surfaceName string) SurfaceInteractionContract {
	return SurfaceInteractionContract{
		SurfaceName:     surfaceName,
		RequiresFilters: true,
		RequiredStates:  append([]string(nil), requiredSurfaceStates...),
	}
}

type ConsoleInteractionDraft struct {
	Name                   string                       `json:"name"`
	Version                string                       `json:"version"`
	Architecture           ConsoleIA                    `json:"architecture"`
	Contracts              []SurfaceInteractionContract `json:"contracts,omitempty"`
	RequiredRoles          []string                     `json:"required_roles,omitempty"`
	RequiresFrameContracts bool                         `json:"requires_frame_contracts,omitempty"`
}

func (d ConsoleInteractionDraft) ToMap() map[string]any {
	return mustToMap(d)
}

func ConsoleInteractionDraftFromMap(data map[string]any) ConsoleInteractionDraft {
	var out ConsoleInteractionDraft
	mustFromMap(data, &out)
	for i := range out.Contracts {
		if len(out.Contracts[i].RequiredStates) == 0 {
			out.Contracts[i].RequiredStates = append([]string(nil), requiredSurfaceStates...)
		}
	}
	return out
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
	score := 100 - (float64(penalties*100) / float64(a.ContractCount))
	if score < 0 {
		score = 0
	}
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

func (a ConsoleInteractionAudit) ToMap() map[string]any {
	return mustToMap(a)
}

func ConsoleInteractionAuditFromMap(data map[string]any) ConsoleInteractionAudit {
	var out ConsoleInteractionAudit
	mustFromMap(data, &out)
	return out
}

type ConsoleIAAudit struct {
	SystemName             string                                `json:"system_name"`
	Version                string                                `json:"version"`
	SurfaceCount           int                                   `json:"surface_count"`
	NavigationCount        int                                   `json:"navigation_count"`
	TopBarAudit            designsystemcompat.ConsoleTopBarAudit `json:"top_bar_audit"`
	SurfacesMissingFilters []string                              `json:"surfaces_missing_filters,omitempty"`
	SurfacesMissingActions []string                              `json:"surfaces_missing_actions,omitempty"`
	SurfacesMissingStates  map[string][]string                   `json:"surfaces_missing_states,omitempty"`
	StatesMissingActions   map[string][]string                   `json:"states_missing_actions,omitempty"`
	UnresolvedStateActions map[string]map[string][]string        `json:"unresolved_state_actions,omitempty"`
	OrphanNavigationRoutes []string                              `json:"orphan_navigation_routes,omitempty"`
	UnnavigableSurfaces    []string                              `json:"unnavigable_surfaces,omitempty"`
}

func (a ConsoleIAAudit) ReadinessScore() float64 {
	if a.SurfaceCount == 0 {
		return 0
	}
	penalties := len(a.SurfacesMissingFilters) +
		len(a.SurfacesMissingActions) +
		len(a.SurfacesMissingStates) +
		len(a.StatesMissingActions) +
		len(a.UnresolvedStateActions) +
		len(a.OrphanNavigationRoutes) +
		len(a.UnnavigableSurfaces)
	if !a.TopBarAudit.ReleaseReady() {
		penalties++
	}
	score := 100 - (float64(penalties*100) / float64(a.SurfaceCount))
	if score < 0 {
		score = 0
	}
	return round1(score)
}

func (a ConsoleIAAudit) ToMap() map[string]any {
	return mustToMap(a)
}

func ConsoleIAAuditFromMap(data map[string]any) ConsoleIAAudit {
	var out ConsoleIAAudit
	mustFromMap(data, &out)
	return out
}

type ConsoleIAAuditor struct{}

func (ConsoleIAAuditor) Audit(architecture ConsoleIA) ConsoleIAAudit {
	topBarAudit := designsystemcompat.ConsoleChromeLibrary{}.AuditTopBar(architecture.TopBar)
	routeIndex := architecture.RouteIndex()
	navigationRoutes := make(map[string]struct{}, len(architecture.Navigation))
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
		if missing := surface.UnresolvedStateActions(); len(missing) > 0 {
			unresolvedStateActions[surface.Name] = missing
		}
	}

	orphanRoutes := make([]string, 0)
	for route := range navigationRoutes {
		if _, ok := routeIndex[route]; !ok {
			orphanRoutes = append(orphanRoutes, route)
		}
	}
	sort.Strings(orphanRoutes)

	unnavigableSurfaces := make([]string, 0)
	for _, surface := range architecture.Surfaces {
		if _, ok := navigationRoutes[surface.Route]; !ok {
			unnavigableSurfaces = append(unnavigableSurfaces, surface.Name)
		}
	}
	sort.Strings(unnavigableSurfaces)
	sort.Strings(surfacesMissingFilters)
	sort.Strings(surfacesMissingActions)

	return ConsoleIAAudit{
		SystemName:             architecture.Name,
		Version:                architecture.Version,
		SurfaceCount:           len(architecture.Surfaces),
		NavigationCount:        len(architecture.Navigation),
		TopBarAudit:            topBarAudit,
		SurfacesMissingFilters: surfacesMissingFilters,
		SurfacesMissingActions: surfacesMissingActions,
		SurfacesMissingStates:  surfacesMissingStates,
		StatesMissingActions:   statesMissingActions,
		UnresolvedStateActions: unresolvedStateActions,
		OrphanNavigationRoutes: orphanRoutes,
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
			surfacesMissingFilters = append(surfacesMissingFilters, contract.SurfaceName)
		}
		availableActionIDs := make(map[string]struct{}, len(surface.TopBarActions))
		for _, action := range surface.TopBarActions {
			availableActionIDs[action.ActionID] = struct{}{}
		}
		missingActionIDs := make([]string, 0)
		for _, actionID := range contract.RequiredActionIDs {
			if _, ok := availableActionIDs[actionID]; !ok {
				missingActionIDs = append(missingActionIDs, actionID)
			}
		}
		if len(missingActionIDs) > 0 {
			surfacesMissingActions[contract.SurfaceName] = missingActionIDs
		}
		if contract.RequiresBatchActions {
			hasBatchAction := false
			for _, action := range surface.TopBarActions {
				if action.RequiresSelection {
					hasBatchAction = true
					break
				}
			}
			if !hasBatchAction {
				surfacesMissingBatchActions = append(surfacesMissingBatchActions, contract.SurfaceName)
			}
		}
		requiredStates := contract.RequiredStates
		if len(requiredStates) == 0 {
			requiredStates = requiredSurfaceStates
		}
		stateIndex := make(map[string]struct{}, len(surface.States))
		for _, state := range surface.States {
			stateIndex[state.Name] = struct{}{}
		}
		missingStates := make([]string, 0)
		for _, state := range requiredStates {
			if _, ok := stateIndex[state]; !ok {
				missingStates = append(missingStates, state)
			}
		}
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
		if _, ok := referencedRoles[role]; !ok {
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
		fmt.Sprintf("- Name: %s", firstNonEmpty(architecture.TopBar.Name, "none")),
		fmt.Sprintf("- Release Ready: %t", audit.TopBarAudit.ReleaseReady()),
		fmt.Sprintf("- Missing capabilities: %s", joinOrNone(audit.TopBarAudit.MissingCapabilities)),
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
			lines = append(lines, fmt.Sprintf("- %s / %s: route=%s badge=%d icon=%s", item.Section, item.Name, item.Route, item.BadgeCount, firstNonEmpty(item.Icon, "none")))
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
				keys := sortedKeys(unresolved)
				for _, key := range keys {
					parts = append(parts, fmt.Sprintf("%s=%s", key, strings.Join(unresolved[key], ", ")))
				}
				unresolvedText = strings.Join(parts, "; ")
			}
			lines = append(lines,
				fmt.Sprintf("- %s: route=%s filters=%s actions=%s states=%s missing_states=%s states_without_actions=%s unresolved_state_actions=%s",
					surface.Name,
					surface.Route,
					joinOrNone(filters),
					joinOrNone(actions),
					joinOrNone(surface.StateNames()),
					joinOrNone(surface.MissingRequiredStates()),
					joinOrNone(audit.StatesMissingActions[surface.Name]),
					unresolvedText))
		}
	}
	lines = append(lines, "", "## Gaps", "")
	lines = append(lines, fmt.Sprintf("- Surfaces missing filters: %s", joinOrNone(audit.SurfacesMissingFilters)))
	lines = append(lines, fmt.Sprintf("- Surfaces missing top-bar actions: %s", joinOrNone(audit.SurfacesMissingActions)))
	lines = append(lines, fmt.Sprintf("- Surfaces missing required states: %s", formatMapList(audit.SurfacesMissingStates)))
	lines = append(lines, fmt.Sprintf("- States without recovery actions: %s", formatMapList(audit.StatesMissingActions)))
	lines = append(lines, fmt.Sprintf("- Undefined state actions: %s", formatNestedMapList(audit.UnresolvedStateActions)))
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
				contract.SurfaceName,
				surface.Route,
				joinOrNone(contract.RequiredActionIDs),
				joinOrNone(surface.ActionIDs()),
				len(surface.Filters),
				joinOrNone(surface.StateNames()),
				batchMode(contract.RequiresBatchActions),
				permissionState(contract.PermissionRule.Complete())))
			lines = append(lines, fmt.Sprintf("  persona=%s wireframe=%s review_focus=%s decision_prompts=%s",
				firstNonEmpty(contract.PrimaryPersona, "none"),
				firstNonEmpty(contract.LinkedWireframeID, "none"),
				joinCommaOrNone(contract.ReviewFocusAreas),
				joinCommaOrNone(contract.DecisionPrompts)))
		}
	}
	lines = append(lines, "", "## Gaps", "")
	lines = append(lines, fmt.Sprintf("- Missing surfaces: %s", joinOrNone(audit.MissingSurfaces)))
	lines = append(lines, fmt.Sprintf("- Pages missing filters: %s", joinOrNone(audit.SurfacesMissingFilters)))
	lines = append(lines, fmt.Sprintf("- Pages missing actions: %s", formatMapList(audit.SurfacesMissingActions)))
	lines = append(lines, fmt.Sprintf("- Pages missing batch actions: %s", joinOrNone(audit.SurfacesMissingBatchActions)))
	lines = append(lines, fmt.Sprintf("- Pages missing states: %s", formatMapList(audit.SurfacesMissingStates)))
	lines = append(lines, fmt.Sprintf("- Permission gaps: %s", formatMapList(audit.PermissionGaps)))
	lines = append(lines, fmt.Sprintf("- Uncovered roles: %s", joinOrNone(audit.UncoveredRoles)))
	lines = append(lines, fmt.Sprintf("- Pages missing personas: %s", joinOrNone(audit.SurfacesMissingPrimaryPersonas)))
	lines = append(lines, fmt.Sprintf("- Pages missing wireframe links: %s", joinOrNone(audit.SurfacesMissingWireframeLinks)))
	lines = append(lines, fmt.Sprintf("- Pages missing review focus: %s", joinOrNone(audit.SurfacesMissingReviewFocus)))
	lines = append(lines, fmt.Sprintf("- Pages missing decision prompts: %s", joinOrNone(audit.SurfacesMissingDecisionPrompts)))
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
			TopBar: designsystemcompat.ConsoleTopBar{
				Name:                      "BigClaw Global Header",
				SearchPlaceholder:         "Search runs, queues, prompts, and commands",
				EnvironmentOptions:        []string{"Production", "Staging", "Shadow"},
				TimeRangeOptions:          []string{"24h", "7d", "30d"},
				AlertChannels:             []string{"approvals", "sla", "regressions"},
				DocumentationComplete:     true,
				AccessibilityRequirements: []string{"keyboard-navigation", "screen-reader-label", "focus-visible"},
				CommandEntry: designsystemcompat.ConsoleCommandEntry{
					TriggerLabel: "Command Menu",
					Placeholder:  "Jump to a run, queue, or release control action",
					Shortcut:     "Cmd+K / Ctrl+K",
					Commands: []designsystemcompat.CommandAction{
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
				RequiresFilters:   true,
				RequiredActionIDs: []string{"drill-down", "export", "audit"},
				RequiredStates:    append([]string(nil), requiredSurfaceStates...),
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
				RequiresFilters:      true,
				RequiresBatchActions: true,
				RequiredActionIDs:    []string{"drill-down", "export", "audit"},
				RequiredStates:       append([]string(nil), requiredSurfaceStates...),
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
				RequiresFilters:   true,
				RequiredActionIDs: []string{"drill-down", "export", "audit"},
				RequiredStates:    append([]string(nil), requiredSurfaceStates...),
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
				RequiresFilters:      true,
				RequiresBatchActions: true,
				RequiredActionIDs:    []string{"drill-down", "export", "audit"},
				RequiredStates:       append([]string(nil), requiredSurfaceStates...),
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

func mustToMap(v any) map[string]any {
	body, _ := json.Marshal(v)
	out := map[string]any{}
	_ = json.Unmarshal(body, &out)
	return out
}

func mustFromMap(data map[string]any, out any) {
	body, _ := json.Marshal(data)
	_ = json.Unmarshal(body, out)
}

func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func joinCommaOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ",")
}

func firstNonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func round1(value float64) float64 {
	return float64(int(value*10+0.5)) / 10
}

func formatMapList(values map[string][]string) string {
	if len(values) == 0 {
		return "none"
	}
	keys := sortedKeys(values)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, strings.Join(values[key], ", ")))
	}
	return strings.Join(parts, "; ")
}

func formatNestedMapList(values map[string]map[string][]string) string {
	if len(values) == 0 {
		return "none"
	}
	keys := sortedKeys(values)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		stateKeys := sortedKeys(values[key])
		stateParts := make([]string, 0, len(stateKeys))
		for _, state := range stateKeys {
			stateParts = append(stateParts, fmt.Sprintf("%s:%s", state, strings.Join(values[key][state], "/")))
		}
		parts = append(parts, fmt.Sprintf("%s=%s", key, strings.Join(stateParts, ", ")))
	}
	return strings.Join(parts, "; ")
}

func batchMode(required bool) string {
	if required {
		return "required"
	}
	return "optional"
}

func permissionState(complete bool) string {
	if complete {
		return "complete"
	}
	return "incomplete"
}
