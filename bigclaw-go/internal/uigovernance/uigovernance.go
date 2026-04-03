package uigovernance

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

var foundationCategories = []string{"color", "spacing", "typography", "motion", "radius"}

var componentReadinessOrder = map[string]int{
	"draft":  0,
	"alpha":  1,
	"beta":   2,
	"stable": 3,
}

var requiredInteractionStates = []string{"default", "disabled", "hover"}

type DesignToken struct {
	Name         string `json:"name"`
	Category     string `json:"category"`
	Value        string `json:"value"`
	SemanticRole string `json:"semantic_role,omitempty"`
	Theme        string `json:"theme,omitempty"`
}

func (t DesignToken) normalized() DesignToken {
	if strings.TrimSpace(t.Theme) == "" {
		t.Theme = "core"
	}
	return t
}

type ComponentVariant struct {
	Name       string   `json:"name"`
	Tokens     []string `json:"tokens,omitempty"`
	States     []string `json:"states,omitempty"`
	UsageNotes string   `json:"usage_notes,omitempty"`
}

type ComponentSpec struct {
	Name                      string             `json:"name"`
	Readiness                 string             `json:"readiness,omitempty"`
	Slots                     []string           `json:"slots,omitempty"`
	Variants                  []ComponentVariant `json:"variants,omitempty"`
	AccessibilityRequirements []string           `json:"accessibility_requirements,omitempty"`
	DocumentationComplete     bool               `json:"documentation_complete,omitempty"`
}

func (c ComponentSpec) TokenNames() []string {
	var names []string
	seen := map[string]bool{}
	for _, variant := range c.Variants {
		for _, token := range variant.Tokens {
			if !seen[token] {
				seen[token] = true
				names = append(names, token)
			}
		}
	}
	return names
}

func (c ComponentSpec) StateCoverage() []string {
	var coverage []string
	seen := map[string]bool{}
	for _, variant := range c.Variants {
		for _, state := range variant.States {
			if !seen[state] {
				seen[state] = true
				coverage = append(coverage, state)
			}
		}
	}
	return coverage
}

func (c ComponentSpec) MissingRequiredStates() []string {
	covered := map[string]bool{}
	for _, state := range c.StateCoverage() {
		covered[state] = true
	}
	var missing []string
	for _, state := range requiredInteractionStates {
		if !covered[state] {
			missing = append(missing, state)
		}
	}
	return missing
}

func (c ComponentSpec) ReleaseReady() bool {
	readiness := strings.TrimSpace(c.Readiness)
	if readiness == "" {
		readiness = "draft"
	}
	return componentReadinessOrder[readiness] >= componentReadinessOrder["beta"] &&
		c.DocumentationComplete &&
		len(c.AccessibilityRequirements) > 0 &&
		len(c.MissingRequiredStates()) == 0
}

type DesignSystem struct {
	Name       string          `json:"name"`
	Version    string          `json:"version"`
	Tokens     []DesignToken   `json:"tokens,omitempty"`
	Components []ComponentSpec `json:"components,omitempty"`
}

func (s DesignSystem) normalized() DesignSystem {
	out := s
	for i, token := range out.Tokens {
		out.Tokens[i] = token.normalized()
	}
	return out
}

func (s DesignSystem) TokenCounts() map[string]int {
	counts := map[string]int{}
	for _, category := range foundationCategories {
		counts[category] = 0
	}
	for _, token := range s.Tokens {
		counts[token.Category]++
	}
	return counts
}

func (s DesignSystem) TokenIndex() map[string]DesignToken {
	index := make(map[string]DesignToken, len(s.Tokens))
	for _, token := range s.Tokens {
		index[token.Name] = token.normalized()
	}
	return index
}

type DesignSystemAudit struct {
	SystemName                     string              `json:"system_name"`
	Version                        string              `json:"version"`
	TokenCounts                    map[string]int      `json:"token_counts"`
	ComponentCount                 int                 `json:"component_count"`
	ReleaseReadyComponents         []string            `json:"release_ready_components,omitempty"`
	ComponentsMissingDocs          []string            `json:"components_missing_docs,omitempty"`
	ComponentsMissingAccessibility []string            `json:"components_missing_accessibility,omitempty"`
	ComponentsMissingStates        []string            `json:"components_missing_states,omitempty"`
	UndefinedTokenRefs             map[string][]string `json:"undefined_token_refs,omitempty"`
	TokenOrphans                   []string            `json:"token_orphans,omitempty"`
}

func (a DesignSystemAudit) ReadinessScore() float64 {
	if a.ComponentCount == 0 {
		return 0
	}
	ready := len(a.ReleaseReadyComponents)
	penalties := len(a.ComponentsMissingDocs) + len(a.ComponentsMissingAccessibility) + len(a.ComponentsMissingStates)
	score := ((float64(ready) * 100) - (float64(penalties) * 10)) / float64(a.ComponentCount)
	if score < 0 {
		return 0
	}
	return round1(score)
}

type ComponentLibrary struct{}

func (ComponentLibrary) Audit(system DesignSystem) DesignSystemAudit {
	tokenIndex := system.TokenIndex()
	usedTokens := map[string]bool{}
	var releaseReady []string
	var missingDocs []string
	var missingAccessibility []string
	var missingStates []string
	undefinedRefs := map[string][]string{}

	for _, component := range system.Components {
		tokenNames := component.TokenNames()
		for _, token := range tokenNames {
			usedTokens[token] = true
		}
		var missingTokens []string
		for _, token := range tokenNames {
			if _, ok := tokenIndex[token]; !ok {
				missingTokens = append(missingTokens, token)
			}
		}
		sort.Strings(missingTokens)
		if len(missingTokens) > 0 {
			undefinedRefs[component.Name] = missingTokens
		}
		if component.ReleaseReady() && len(missingTokens) == 0 {
			releaseReady = append(releaseReady, component.Name)
		}
		if !component.DocumentationComplete {
			missingDocs = append(missingDocs, component.Name)
		}
		if len(component.AccessibilityRequirements) == 0 {
			missingAccessibility = append(missingAccessibility, component.Name)
		}
		if len(component.MissingRequiredStates()) > 0 {
			missingStates = append(missingStates, component.Name)
		}
	}

	var orphans []string
	for _, token := range system.Tokens {
		if !usedTokens[token.Name] {
			orphans = append(orphans, token.Name)
		}
	}
	sort.Strings(releaseReady)
	sort.Strings(missingDocs)
	sort.Strings(missingAccessibility)
	sort.Strings(missingStates)
	sort.Strings(orphans)

	return DesignSystemAudit{
		SystemName:                     system.Name,
		Version:                        system.Version,
		TokenCounts:                    system.TokenCounts(),
		ComponentCount:                 len(system.Components),
		ReleaseReadyComponents:         releaseReady,
		ComponentsMissingDocs:          missingDocs,
		ComponentsMissingAccessibility: missingAccessibility,
		ComponentsMissingStates:        missingStates,
		UndefinedTokenRefs:             undefinedRefs,
		TokenOrphans:                   orphans,
	}
}

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
	DocumentationComplete     bool                `json:"documentation_complete,omitempty"`
	AccessibilityRequirements []string            `json:"accessibility_requirements,omitempty"`
}

func (t ConsoleTopBar) HasGlobalSearch() bool {
	return strings.TrimSpace(t.SearchPlaceholder) != ""
}

func (t ConsoleTopBar) HasEnvironmentSwitch() bool {
	return len(t.EnvironmentOptions) >= 2
}

func (t ConsoleTopBar) HasTimeRangeSwitch() bool {
	return len(t.TimeRangeOptions) >= 2
}

func (t ConsoleTopBar) HasAlertEntry() bool {
	return len(t.AlertChannels) > 0
}

func (t ConsoleTopBar) HasCommandShell() bool {
	return strings.TrimSpace(t.CommandEntry.TriggerLabel) != "" && len(t.CommandEntry.Commands) > 0
}

type ConsoleTopBarAudit struct {
	Name                     string   `json:"name"`
	MissingCapabilities      []string `json:"missing_capabilities,omitempty"`
	DocumentationComplete    bool     `json:"documentation_complete,omitempty"`
	AccessibilityComplete    bool     `json:"accessibility_complete,omitempty"`
	CommandShortcutSupported bool     `json:"command_shortcut_supported,omitempty"`
	CommandCount             int      `json:"command_count"`
}

func (a ConsoleTopBarAudit) ReleaseReady() bool {
	return len(a.MissingCapabilities) == 0 &&
		a.DocumentationComplete &&
		a.AccessibilityComplete &&
		a.CommandShortcutSupported
}

type ConsoleChromeLibrary struct{}

var requiredShortcuts = map[string]bool{"cmd+k": true, "ctrl+k": true}
var requiredAccessibility = map[string]bool{"keyboard-navigation": true, "screen-reader-label": true, "focus-visible": true}

func (ConsoleChromeLibrary) AuditTopBar(topBar ConsoleTopBar) ConsoleTopBarAudit {
	var missing []string
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
	normalized := map[string]bool{}
	for _, item := range strings.Split(topBar.CommandEntry.Shortcut, "/") {
		item = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(item), " ", ""))
		if item != "" {
			normalized[item] = true
		}
	}
	accessibility := map[string]bool{}
	for _, item := range topBar.AccessibilityRequirements {
		accessibility[item] = true
	}
	return ConsoleTopBarAudit{
		Name:                     topBar.Name,
		MissingCapabilities:      missing,
		DocumentationComplete:    topBar.DocumentationComplete,
		AccessibilityComplete:    hasAllKeys(accessibility, requiredAccessibility),
		CommandShortcutSupported: hasAllKeys(normalized, requiredShortcuts),
		CommandCount:             len(topBar.CommandEntry.Commands),
	}
}

func NormalizeRoutePath(path string) string {
	stripped := strings.Trim(strings.TrimSpace(path), "/")
	if stripped == "" {
		return "/"
	}
	return "/" + stripped
}

type NavigationRoute struct {
	Path      string `json:"path"`
	ScreenID  string `json:"screen_id"`
	Title     string `json:"title"`
	NavNodeID string `json:"nav_node_id,omitempty"`
	Layout    string `json:"layout,omitempty"`
}

func (r *NavigationRoute) UnmarshalJSON(data []byte) error {
	type alias NavigationRoute
	var raw alias
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*r = NavigationRoute(raw)
	r.normalize()
	return nil
}

func (r *NavigationRoute) normalize() {
	r.Path = NormalizeRoutePath(r.Path)
	if strings.TrimSpace(r.Layout) == "" {
		r.Layout = "workspace"
	}
}

type NavigationNode struct {
	NodeID   string           `json:"node_id"`
	Title    string           `json:"title"`
	Segment  string           `json:"segment"`
	ScreenID string           `json:"screen_id,omitempty"`
	Children []NavigationNode `json:"children,omitempty"`
}

type NavigationEntry struct {
	NodeID   string
	Title    string
	Path     string
	Depth    int
	ParentID string
	ScreenID string
}

type InformationArchitectureAudit struct {
	TotalNavigationNodes int                 `json:"total_navigation_nodes"`
	TotalRoutes          int                 `json:"total_routes"`
	DuplicateRoutes      []string            `json:"duplicate_routes,omitempty"`
	MissingRouteNodes    map[string]string   `json:"missing_route_nodes,omitempty"`
	SecondaryNavGaps     map[string][]string `json:"secondary_nav_gaps,omitempty"`
	OrphanRoutes         []string            `json:"orphan_routes,omitempty"`
}

func (a InformationArchitectureAudit) Healthy() bool {
	return len(a.DuplicateRoutes) == 0 &&
		len(a.MissingRouteNodes) == 0 &&
		len(a.SecondaryNavGaps) == 0 &&
		len(a.OrphanRoutes) == 0
}

type InformationArchitecture struct {
	GlobalNav []NavigationNode  `json:"global_nav,omitempty"`
	Routes    []NavigationRoute `json:"routes,omitempty"`
}

func (a InformationArchitecture) RouteIndex() map[string]NavigationRoute {
	index := map[string]NavigationRoute{}
	for _, route := range a.Routes {
		normalized := route
		normalized.normalize()
		if _, ok := index[normalized.Path]; !ok {
			index[normalized.Path] = normalized
		}
	}
	return index
}

func (a InformationArchitecture) NavigationEntries() []NavigationEntry {
	var entries []NavigationEntry
	for _, node := range a.GlobalNav {
		entries = append(entries, flattenNode(node, "", 0, "")...)
	}
	return entries
}

func (a InformationArchitecture) ResolveRoute(path string) (NavigationRoute, bool) {
	route, ok := a.RouteIndex()[NormalizeRoutePath(path)]
	return route, ok
}

func (a InformationArchitecture) Audit() InformationArchitectureAudit {
	entries := a.NavigationEntries()
	routeCounts := map[string]int{}
	for _, route := range a.Routes {
		routeCounts[NormalizeRoutePath(route.Path)]++
	}
	var duplicateRoutes []string
	for path, count := range routeCounts {
		if count > 1 {
			duplicateRoutes = append(duplicateRoutes, path)
		}
	}
	sort.Strings(duplicateRoutes)

	routeIndex := a.RouteIndex()
	missingRouteNodes := map[string]string{}
	for _, entry := range entries {
		if _, ok := routeIndex[entry.Path]; !ok {
			missingRouteNodes[entry.NodeID] = entry.Path
		}
	}

	secondaryGaps := map[string][]string{}
	for _, root := range a.GlobalNav {
		gaps := missingPathsForDescendants(root, "", routeIndex)
		sort.Strings(gaps)
		if len(gaps) > 0 {
			secondaryGaps[root.Title] = gaps
		}
	}

	navPaths := map[string]bool{}
	for _, entry := range entries {
		navPaths[entry.Path] = true
	}
	var orphanRoutes []string
	for _, route := range a.Routes {
		path := NormalizeRoutePath(route.Path)
		if !navPaths[path] {
			orphanRoutes = append(orphanRoutes, path)
		}
	}
	sort.Strings(orphanRoutes)

	return InformationArchitectureAudit{
		TotalNavigationNodes: len(entries),
		TotalRoutes:          len(a.Routes),
		DuplicateRoutes:      duplicateRoutes,
		MissingRouteNodes:    missingRouteNodes,
		SecondaryNavGaps:     secondaryGaps,
		OrphanRoutes:         orphanRoutes,
	}
}

type RolePermissionScenario struct {
	ScreenID     string   `json:"screen_id"`
	AllowedRoles []string `json:"allowed_roles,omitempty"`
	DeniedRoles  []string `json:"denied_roles,omitempty"`
	AuditEvent   string   `json:"audit_event,omitempty"`
}

func (s RolePermissionScenario) MissingCoverage() []string {
	var missing []string
	if len(s.AllowedRoles) == 0 {
		missing = append(missing, "allowed-roles")
	}
	if len(s.DeniedRoles) == 0 {
		missing = append(missing, "denied-roles")
	}
	if strings.TrimSpace(s.AuditEvent) == "" {
		missing = append(missing, "audit-event")
	}
	return missing
}

type DataAccuracyCheck struct {
	ScreenID                 string  `json:"screen_id"`
	MetricID                 string  `json:"metric_id"`
	SourceOfTruth            string  `json:"source_of_truth"`
	RenderedValue            string  `json:"rendered_value"`
	Tolerance                float64 `json:"tolerance,omitempty"`
	ObservedDelta            float64 `json:"observed_delta,omitempty"`
	FreshnessSLOSeconds      int     `json:"freshness_slo_seconds,omitempty"`
	ObservedFreshnessSeconds int     `json:"observed_freshness_seconds,omitempty"`
}

func (c DataAccuracyCheck) WithinTolerance() bool {
	if c.ObservedDelta < 0 {
		return -c.ObservedDelta <= c.Tolerance
	}
	return c.ObservedDelta <= c.Tolerance
}

func (c DataAccuracyCheck) WithinFreshnessSLO() bool {
	if c.FreshnessSLOSeconds <= 0 {
		return true
	}
	return c.ObservedFreshnessSeconds <= c.FreshnessSLOSeconds
}

func (c DataAccuracyCheck) Passes() bool {
	return c.WithinTolerance() && c.WithinFreshnessSLO()
}

type PerformanceBudget struct {
	SurfaceID     string `json:"surface_id"`
	Interaction   string `json:"interaction"`
	TargetP95MS   int    `json:"target_p95_ms"`
	ObservedP95MS int    `json:"observed_p95_ms"`
	TargetTTIMS   int    `json:"target_tti_ms,omitempty"`
	ObservedTTIMS int    `json:"observed_tti_ms,omitempty"`
}

func (b PerformanceBudget) WithinBudget() bool {
	p95OK := b.ObservedP95MS <= b.TargetP95MS
	ttiOK := b.TargetTTIMS <= 0 || b.ObservedTTIMS <= b.TargetTTIMS
	return p95OK && ttiOK
}

type UsabilityJourney struct {
	JourneyID          string   `json:"journey_id"`
	Personas           []string `json:"personas,omitempty"`
	CriticalSteps      []string `json:"critical_steps,omitempty"`
	ExpectedMaxSteps   int      `json:"expected_max_steps,omitempty"`
	ObservedSteps      int      `json:"observed_steps,omitempty"`
	KeyboardAccessible bool     `json:"keyboard_accessible,omitempty"`
	EmptyStateGuidance bool     `json:"empty_state_guidance,omitempty"`
	RecoverySupport    bool     `json:"recovery_support,omitempty"`
}

func (j UsabilityJourney) Passes() bool {
	return len(j.Personas) > 0 &&
		len(j.CriticalSteps) > 0 &&
		j.ExpectedMaxSteps > 0 &&
		j.ObservedSteps <= j.ExpectedMaxSteps &&
		j.KeyboardAccessible &&
		j.EmptyStateGuidance &&
		j.RecoverySupport
}

type AuditRequirement struct {
	EventType             string   `json:"event_type"`
	RequiredFields        []string `json:"required_fields,omitempty"`
	EmittedFields         []string `json:"emitted_fields,omitempty"`
	RetentionDays         int      `json:"retention_days,omitempty"`
	ObservedRetentionDays int      `json:"observed_retention_days,omitempty"`
}

func (r AuditRequirement) MissingFields() []string {
	emitted := map[string]bool{}
	for _, field := range r.EmittedFields {
		emitted[field] = true
	}
	var missing []string
	for _, field := range r.RequiredFields {
		if !emitted[field] {
			missing = append(missing, field)
		}
	}
	sort.Strings(missing)
	return missing
}

func (r AuditRequirement) RetentionMet() bool {
	if r.RetentionDays <= 0 {
		return true
	}
	return r.ObservedRetentionDays >= r.RetentionDays
}

func (r AuditRequirement) Complete() bool {
	return len(r.MissingFields()) == 0 && r.RetentionMet()
}

type UIAcceptanceSuite struct {
	Name                  string                   `json:"name"`
	Version               string                   `json:"version"`
	RolePermissions       []RolePermissionScenario `json:"role_permissions,omitempty"`
	DataAccuracyChecks    []DataAccuracyCheck      `json:"data_accuracy_checks,omitempty"`
	PerformanceBudgets    []PerformanceBudget      `json:"performance_budgets,omitempty"`
	UsabilityJourneys     []UsabilityJourney       `json:"usability_journeys,omitempty"`
	AuditRequirements     []AuditRequirement       `json:"audit_requirements,omitempty"`
	DocumentationComplete bool                     `json:"documentation_complete,omitempty"`
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

func (a GlobalAction) normalized() GlobalAction {
	if strings.TrimSpace(a.Intent) == "" {
		a.Intent = "default"
	}
	return a
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

var requiredSurfaceStates = []string{"default", "empty", "error", "loading"}

func (s ConsoleSurface) ActionIDs() []string {
	ids := make([]string, 0, len(s.TopBarActions))
	for _, action := range s.TopBarActions {
		ids = append(ids, action.ActionID)
	}
	return ids
}

func (s ConsoleSurface) StateNames() []string {
	names := make([]string, 0, len(s.States))
	for _, state := range s.States {
		names = append(names, state.Name)
	}
	return names
}

func (s ConsoleSurface) MissingRequiredStates() []string {
	have := map[string]bool{}
	for _, state := range s.StateNames() {
		have[state] = true
	}
	var missing []string
	for _, state := range requiredSurfaceStates {
		if !have[state] {
			missing = append(missing, state)
		}
	}
	return missing
}

func (s ConsoleSurface) UnresolvedStateActions() map[string][]string {
	available := map[string]bool{}
	for _, actionID := range s.ActionIDs() {
		available[actionID] = true
	}
	unresolved := map[string][]string{}
	for _, state := range s.States {
		var missing []string
		for _, actionID := range state.AllowedActions {
			if !available[actionID] {
				missing = append(missing, actionID)
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
	var missing []string
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

func (a ConsoleIA) normalized() ConsoleIA {
	out := a
	for i, action := range out.TopBar.CommandEntry.Commands {
		out.TopBar.CommandEntry.Commands[i] = action
	}
	for i, surface := range out.Surfaces {
		for j, action := range surface.TopBarActions {
			surface.TopBarActions[j] = action.normalized()
		}
		out.Surfaces[i] = surface
	}
	return out
}

func (a ConsoleIA) RouteIndex() map[string]ConsoleSurface {
	index := map[string]ConsoleSurface{}
	for _, surface := range a.Surfaces {
		index[surface.Route] = surface
	}
	return index
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
	score := 100 - ((float64(penalties) * 100) / float64(a.SurfaceCount))
	if score < 0 {
		return 0
	}
	return round1(score)
}

type ConsoleIAAuditor struct{}

func (ConsoleIAAuditor) Audit(architecture ConsoleIA) ConsoleIAAudit {
	topBarAudit := (ConsoleChromeLibrary{}).AuditTopBar(architecture.TopBar)
	routeIndex := architecture.RouteIndex()
	navigationRoutes := map[string]bool{}
	for _, item := range architecture.Navigation {
		navigationRoutes[item.Route] = true
	}
	var missingFilters []string
	var missingActions []string
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
	var orphanRoutes []string
	for route := range navigationRoutes {
		if _, ok := routeIndex[route]; !ok {
			orphanRoutes = append(orphanRoutes, route)
		}
	}
	sort.Strings(orphanRoutes)
	var unnavigableSurfaces []string
	for _, surface := range architecture.Surfaces {
		if !navigationRoutes[surface.Route] {
			unnavigableSurfaces = append(unnavigableSurfaces, surface.Name)
		}
	}
	sort.Strings(unnavigableSurfaces)
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
		SurfacesMissingStates:  missingStates,
		StatesMissingActions:   statesMissingActions,
		UnresolvedStateActions: unresolvedStateActions,
		OrphanNavigationRoutes: orphanRoutes,
		UnnavigableSurfaces:    unnavigableSurfaces,
	}
}

type SurfacePermissionRule struct {
	AllowedRoles []string `json:"allowed_roles,omitempty"`
	DeniedRoles  []string `json:"denied_roles,omitempty"`
	AuditEvent   string   `json:"audit_event,omitempty"`
}

func (r SurfacePermissionRule) MissingCoverage() []string {
	var missing []string
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

func (c *SurfaceInteractionContract) UnmarshalJSON(data []byte) error {
	type alias SurfaceInteractionContract
	raw := alias{
		RequiresFilters: true,
		RequiredStates:  append([]string{}, requiredSurfaceStates...),
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*c = SurfaceInteractionContract(raw)
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

type ReviewObjective struct {
	ObjectiveID   string   `json:"objective_id"`
	Title         string   `json:"title"`
	Persona       string   `json:"persona"`
	Outcome       string   `json:"outcome"`
	SuccessSignal string   `json:"success_signal"`
	Priority      string   `json:"priority,omitempty"`
	Dependencies  []string `json:"dependencies,omitempty"`
}

func (o ReviewObjective) normalized() ReviewObjective {
	if strings.TrimSpace(o.Priority) == "" {
		o.Priority = "P1"
	}
	return o
}

type WireframeSurface struct {
	SurfaceID     string   `json:"surface_id"`
	Name          string   `json:"name"`
	Device        string   `json:"device"`
	EntryPoint    string   `json:"entry_point"`
	PrimaryBlocks []string `json:"primary_blocks,omitempty"`
	ReviewNotes   []string `json:"review_notes,omitempty"`
}

type InteractionFlow struct {
	FlowID         string   `json:"flow_id"`
	Name           string   `json:"name"`
	Trigger        string   `json:"trigger"`
	SystemResponse string   `json:"system_response"`
	States         []string `json:"states,omitempty"`
	Exceptions     []string `json:"exceptions,omitempty"`
}

type OpenQuestion struct {
	QuestionID string `json:"question_id"`
	Theme      string `json:"theme"`
	Question   string `json:"question"`
	Owner      string `json:"owner"`
	Impact     string `json:"impact"`
	Status     string `json:"status,omitempty"`
}

func (q OpenQuestion) normalized() OpenQuestion {
	if strings.TrimSpace(q.Status) == "" {
		q.Status = "open"
	}
	return q
}

type ReviewerChecklistItem struct {
	ItemID        string   `json:"item_id"`
	SurfaceID     string   `json:"surface_id"`
	Prompt        string   `json:"prompt"`
	Owner         string   `json:"owner"`
	Status        string   `json:"status,omitempty"`
	EvidenceLinks []string `json:"evidence_links,omitempty"`
	Notes         string   `json:"notes,omitempty"`
}

func (i ReviewerChecklistItem) normalized() ReviewerChecklistItem {
	if strings.TrimSpace(i.Status) == "" {
		i.Status = "todo"
	}
	return i
}

type ReviewDecision struct {
	DecisionID string `json:"decision_id"`
	SurfaceID  string `json:"surface_id"`
	Owner      string `json:"owner"`
	Summary    string `json:"summary"`
	Rationale  string `json:"rationale"`
	Status     string `json:"status,omitempty"`
	FollowUp   string `json:"follow_up,omitempty"`
}

func (d ReviewDecision) normalized() ReviewDecision {
	if strings.TrimSpace(d.Status) == "" {
		d.Status = "proposed"
	}
	return d
}

type ReviewRoleAssignment struct {
	AssignmentID     string   `json:"assignment_id"`
	SurfaceID        string   `json:"surface_id"`
	Role             string   `json:"role"`
	Responsibilities []string `json:"responsibilities,omitempty"`
	ChecklistItemIDs []string `json:"checklist_item_ids,omitempty"`
	DecisionIDs      []string `json:"decision_ids,omitempty"`
	Status           string   `json:"status,omitempty"`
}

func (a ReviewRoleAssignment) normalized() ReviewRoleAssignment {
	if strings.TrimSpace(a.Status) == "" {
		a.Status = "planned"
	}
	return a
}

type ReviewSignoff struct {
	SignoffID       string   `json:"signoff_id"`
	AssignmentID    string   `json:"assignment_id"`
	SurfaceID       string   `json:"surface_id"`
	Role            string   `json:"role"`
	Status          string   `json:"status,omitempty"`
	Required        bool     `json:"required"`
	EvidenceLinks   []string `json:"evidence_links,omitempty"`
	Notes           string   `json:"notes,omitempty"`
	WaiverOwner     string   `json:"waiver_owner,omitempty"`
	WaiverReason    string   `json:"waiver_reason,omitempty"`
	RequestedAt     string   `json:"requested_at,omitempty"`
	DueAt           string   `json:"due_at,omitempty"`
	EscalationOwner string   `json:"escalation_owner,omitempty"`
	SLAStatus       string   `json:"sla_status,omitempty"`
	ReminderOwner   string   `json:"reminder_owner,omitempty"`
	ReminderChannel string   `json:"reminder_channel,omitempty"`
	LastReminderAt  string   `json:"last_reminder_at,omitempty"`
	NextReminderAt  string   `json:"next_reminder_at,omitempty"`
	ReminderCadence string   `json:"reminder_cadence,omitempty"`
	ReminderStatus  string   `json:"reminder_status,omitempty"`
}

func (s *ReviewSignoff) UnmarshalJSON(data []byte) error {
	type alias ReviewSignoff
	raw := alias{
		Status:         "pending",
		Required:       true,
		SLAStatus:      "on-track",
		ReminderStatus: "scheduled",
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*s = ReviewSignoff(raw)
	return nil
}

func (s ReviewSignoff) normalized() ReviewSignoff {
	if strings.TrimSpace(s.Status) == "" {
		s.Status = "pending"
	}
	if strings.TrimSpace(s.SLAStatus) == "" {
		s.SLAStatus = "on-track"
	}
	if strings.TrimSpace(s.ReminderStatus) == "" {
		s.ReminderStatus = "scheduled"
	}
	return s
}

type ReviewBlocker struct {
	BlockerID           string `json:"blocker_id"`
	SurfaceID           string `json:"surface_id"`
	SignoffID           string `json:"signoff_id"`
	Owner               string `json:"owner"`
	Summary             string `json:"summary"`
	Status              string `json:"status,omitempty"`
	Severity            string `json:"severity,omitempty"`
	EscalationOwner     string `json:"escalation_owner,omitempty"`
	NextAction          string `json:"next_action,omitempty"`
	FreezeException     bool   `json:"freeze_exception,omitempty"`
	FreezeOwner         string `json:"freeze_owner,omitempty"`
	FreezeUntil         string `json:"freeze_until,omitempty"`
	FreezeReason        string `json:"freeze_reason,omitempty"`
	FreezeApprovedBy    string `json:"freeze_approved_by,omitempty"`
	FreezeApprovedAt    string `json:"freeze_approved_at,omitempty"`
	FreezeRenewalOwner  string `json:"freeze_renewal_owner,omitempty"`
	FreezeRenewalBy     string `json:"freeze_renewal_by,omitempty"`
	FreezeRenewalStatus string `json:"freeze_renewal_status,omitempty"`
}

func (b ReviewBlocker) normalized() ReviewBlocker {
	if strings.TrimSpace(b.Status) == "" {
		b.Status = "open"
	}
	if strings.TrimSpace(b.Severity) == "" {
		b.Severity = "medium"
	}
	if strings.TrimSpace(b.FreezeRenewalStatus) == "" {
		b.FreezeRenewalStatus = "not-needed"
	}
	return b
}

type ReviewBlockerEvent struct {
	EventID     string `json:"event_id"`
	BlockerID   string `json:"blocker_id"`
	Actor       string `json:"actor"`
	Status      string `json:"status"`
	Summary     string `json:"summary"`
	Timestamp   string `json:"timestamp"`
	NextAction  string `json:"next_action,omitempty"`
	HandoffFrom string `json:"handoff_from,omitempty"`
	HandoffTo   string `json:"handoff_to,omitempty"`
	Channel     string `json:"channel,omitempty"`
	ArtifactRef string `json:"artifact_ref,omitempty"`
	AckOwner    string `json:"ack_owner,omitempty"`
	AckAt       string `json:"ack_at,omitempty"`
	AckStatus   string `json:"ack_status,omitempty"`
}

func (e ReviewBlockerEvent) normalized() ReviewBlockerEvent {
	if strings.TrimSpace(e.AckStatus) == "" {
		e.AckStatus = "pending"
	}
	return e
}

type UIReviewPack struct {
	IssueID                   string                  `json:"issue_id"`
	Title                     string                  `json:"title"`
	Version                   string                  `json:"version"`
	Objectives                []ReviewObjective       `json:"objectives,omitempty"`
	Wireframes                []WireframeSurface      `json:"wireframes,omitempty"`
	Interactions              []InteractionFlow       `json:"interactions,omitempty"`
	OpenQuestions             []OpenQuestion          `json:"open_questions,omitempty"`
	ReviewerChecklist         []ReviewerChecklistItem `json:"reviewer_checklist,omitempty"`
	RequiresReviewerChecklist bool                    `json:"requires_reviewer_checklist,omitempty"`
	DecisionLog               []ReviewDecision        `json:"decision_log,omitempty"`
	RequiresDecisionLog       bool                    `json:"requires_decision_log,omitempty"`
	RoleMatrix                []ReviewRoleAssignment  `json:"role_matrix,omitempty"`
	RequiresRoleMatrix        bool                    `json:"requires_role_matrix,omitempty"`
	SignoffLog                []ReviewSignoff         `json:"signoff_log,omitempty"`
	RequiresSignoffLog        bool                    `json:"requires_signoff_log,omitempty"`
	BlockerLog                []ReviewBlocker         `json:"blocker_log,omitempty"`
	RequiresBlockerLog        bool                    `json:"requires_blocker_log,omitempty"`
	BlockerTimeline           []ReviewBlockerEvent    `json:"blocker_timeline,omitempty"`
	RequiresBlockerTimeline   bool                    `json:"requires_blocker_timeline,omitempty"`
}

func (p UIReviewPack) normalized() UIReviewPack {
	out := p
	for i, objective := range out.Objectives {
		out.Objectives[i] = objective.normalized()
	}
	for i, question := range out.OpenQuestions {
		out.OpenQuestions[i] = question.normalized()
	}
	for i, item := range out.ReviewerChecklist {
		out.ReviewerChecklist[i] = item.normalized()
	}
	for i, decision := range out.DecisionLog {
		out.DecisionLog[i] = decision.normalized()
	}
	for i, assignment := range out.RoleMatrix {
		out.RoleMatrix[i] = assignment.normalized()
	}
	for i, signoff := range out.SignoffLog {
		out.SignoffLog[i] = signoff.normalized()
	}
	for i, blocker := range out.BlockerLog {
		out.BlockerLog[i] = blocker.normalized()
	}
	for i, event := range out.BlockerTimeline {
		out.BlockerTimeline[i] = event.normalized()
	}
	return out
}

type UIReviewPackAudit struct {
	Ready                                     bool     `json:"ready"`
	ObjectiveCount                            int      `json:"objective_count"`
	WireframeCount                            int      `json:"wireframe_count"`
	InteractionCount                          int      `json:"interaction_count"`
	OpenQuestionCount                         int      `json:"open_question_count"`
	ChecklistCount                            int      `json:"checklist_count,omitempty"`
	DecisionCount                             int      `json:"decision_count,omitempty"`
	RoleAssignmentCount                       int      `json:"role_assignment_count,omitempty"`
	SignoffCount                              int      `json:"signoff_count,omitempty"`
	BlockerCount                              int      `json:"blocker_count,omitempty"`
	BlockerTimelineCount                      int      `json:"blocker_timeline_count,omitempty"`
	MissingSections                           []string `json:"missing_sections,omitempty"`
	ObjectivesMissingSignals                  []string `json:"objectives_missing_signals,omitempty"`
	WireframesMissingBlocks                   []string `json:"wireframes_missing_blocks,omitempty"`
	InteractionsMissingStates                 []string `json:"interactions_missing_states,omitempty"`
	UnresolvedQuestionIDs                     []string `json:"unresolved_question_ids,omitempty"`
	WireframesMissingChecklists               []string `json:"wireframes_missing_checklists,omitempty"`
	OrphanChecklistSurfaces                   []string `json:"orphan_checklist_surfaces,omitempty"`
	ChecklistItemsMissingEvidence             []string `json:"checklist_items_missing_evidence,omitempty"`
	ChecklistItemsMissingRoleLinks            []string `json:"checklist_items_missing_role_links,omitempty"`
	WireframesMissingDecisions                []string `json:"wireframes_missing_decisions,omitempty"`
	OrphanDecisionSurfaces                    []string `json:"orphan_decision_surfaces,omitempty"`
	UnresolvedDecisionIDs                     []string `json:"unresolved_decision_ids,omitempty"`
	UnresolvedDecisionsMissingFollowUps       []string `json:"unresolved_decisions_missing_follow_ups,omitempty"`
	WireframesMissingRoleAssignments          []string `json:"wireframes_missing_role_assignments,omitempty"`
	OrphanRoleAssignmentSurfaces              []string `json:"orphan_role_assignment_surfaces,omitempty"`
	RoleAssignmentsMissingResponsibilities    []string `json:"role_assignments_missing_responsibilities,omitempty"`
	RoleAssignmentsMissingChecklistLinks      []string `json:"role_assignments_missing_checklist_links,omitempty"`
	RoleAssignmentsMissingDecisionLinks       []string `json:"role_assignments_missing_decision_links,omitempty"`
	DecisionsMissingRoleLinks                 []string `json:"decisions_missing_role_links,omitempty"`
	WireframesMissingSignoffs                 []string `json:"wireframes_missing_signoffs,omitempty"`
	OrphanSignoffSurfaces                     []string `json:"orphan_signoff_surfaces,omitempty"`
	SignoffsMissingAssignments                []string `json:"signoffs_missing_assignments,omitempty"`
	SignoffsMissingEvidence                   []string `json:"signoffs_missing_evidence,omitempty"`
	SignoffsMissingRequestedDates             []string `json:"signoffs_missing_requested_dates,omitempty"`
	SignoffsMissingDueDates                   []string `json:"signoffs_missing_due_dates,omitempty"`
	SignoffsMissingEscalationOwners           []string `json:"signoffs_missing_escalation_owners,omitempty"`
	SignoffsMissingReminderOwners             []string `json:"signoffs_missing_reminder_owners,omitempty"`
	SignoffsMissingNextReminders              []string `json:"signoffs_missing_next_reminders,omitempty"`
	SignoffsMissingReminderCadence            []string `json:"signoffs_missing_reminder_cadence,omitempty"`
	SignoffsWithBreachedSLA                   []string `json:"signoffs_with_breached_sla,omitempty"`
	UnresolvedRequiredSignoffIDs              []string `json:"unresolved_required_signoff_ids,omitempty"`
	WaivedSignoffsMissingMetadata             []string `json:"waived_signoffs_missing_metadata,omitempty"`
	BlockersMissingSignoffLinks               []string `json:"blockers_missing_signoff_links,omitempty"`
	BlockersMissingEscalationOwners           []string `json:"blockers_missing_escalation_owners,omitempty"`
	BlockersMissingNextActions                []string `json:"blockers_missing_next_actions,omitempty"`
	FreezeExceptionsMissingOwners             []string `json:"freeze_exceptions_missing_owners,omitempty"`
	FreezeExceptionsMissingUntil              []string `json:"freeze_exceptions_missing_until,omitempty"`
	FreezeExceptionsMissingApprovers          []string `json:"freeze_exceptions_missing_approvers,omitempty"`
	FreezeExceptionsMissingApprovalDates      []string `json:"freeze_exceptions_missing_approval_dates,omitempty"`
	FreezeExceptionsMissingRenewalOwners      []string `json:"freeze_exceptions_missing_renewal_owners,omitempty"`
	FreezeExceptionsMissingRenewalDates       []string `json:"freeze_exceptions_missing_renewal_dates,omitempty"`
	BlockersMissingTimelineEvents             []string `json:"blockers_missing_timeline_events,omitempty"`
	ClosedBlockersMissingResolutionEvents     []string `json:"closed_blockers_missing_resolution_events,omitempty"`
	OrphanBlockerSurfaces                     []string `json:"orphan_blocker_surfaces,omitempty"`
	OrphanBlockerTimelineBlockerIDs           []string `json:"orphan_blocker_timeline_blocker_ids,omitempty"`
	HandoffEventsMissingTargets               []string `json:"handoff_events_missing_targets,omitempty"`
	HandoffEventsMissingArtifacts             []string `json:"handoff_events_missing_artifacts,omitempty"`
	HandoffEventsMissingAckOwners             []string `json:"handoff_events_missing_ack_owners,omitempty"`
	HandoffEventsMissingAckDates              []string `json:"handoff_events_missing_ack_dates,omitempty"`
	UnresolvedRequiredSignoffsWithoutBlockers []string `json:"unresolved_required_signoffs_without_blockers,omitempty"`
}

func (a UIReviewPackAudit) Summary() string {
	status := "HOLD"
	if a.Ready {
		status = "READY"
	}
	return status +
		": objectives=" + itoa(a.ObjectiveCount) +
		" wireframes=" + itoa(a.WireframeCount) +
		" interactions=" + itoa(a.InteractionCount) +
		" open_questions=" + itoa(a.OpenQuestionCount) +
		" checklist=" + itoa(a.ChecklistCount) +
		" decisions=" + itoa(a.DecisionCount) +
		" role_assignments=" + itoa(a.RoleAssignmentCount) +
		" signoffs=" + itoa(a.SignoffCount) +
		" blockers=" + itoa(a.BlockerCount) +
		" timeline_events=" + itoa(a.BlockerTimelineCount)
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
	score := 100 - ((float64(penalties) * 100) / float64(a.ContractCount))
	if score < 0 {
		return 0
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

type UIReviewPackAuditor struct{}

func (UIReviewPackAuditor) Audit(pack UIReviewPack) UIReviewPackAudit {
	var missingSections []string
	if len(pack.Objectives) == 0 {
		missingSections = append(missingSections, "objectives")
	}
	if len(pack.Wireframes) == 0 {
		missingSections = append(missingSections, "wireframes")
	}
	if len(pack.Interactions) == 0 {
		missingSections = append(missingSections, "interactions")
	}
	if len(pack.OpenQuestions) == 0 {
		missingSections = append(missingSections, "open_questions")
	}

	var objectivesMissingSignals []string
	for _, objective := range pack.Objectives {
		if strings.TrimSpace(objective.SuccessSignal) == "" {
			objectivesMissingSignals = append(objectivesMissingSignals, objective.ObjectiveID)
		}
	}

	var wireframesMissingBlocks []string
	for _, wireframe := range pack.Wireframes {
		if len(wireframe.PrimaryBlocks) == 0 {
			wireframesMissingBlocks = append(wireframesMissingBlocks, wireframe.SurfaceID)
		}
	}

	var interactionsMissingStates []string
	for _, interaction := range pack.Interactions {
		if len(interaction.States) == 0 {
			interactionsMissingStates = append(interactionsMissingStates, interaction.FlowID)
		}
	}

	var unresolvedQuestionIDs []string
	for _, question := range pack.OpenQuestions {
		normalized := question.normalized()
		if strings.ToLower(normalized.Status) != "resolved" {
			unresolvedQuestionIDs = append(unresolvedQuestionIDs, normalized.QuestionID)
		}
	}

	wireframeIDs := map[string]bool{}
	for _, wireframe := range pack.Wireframes {
		wireframeIDs[wireframe.SurfaceID] = true
	}

	checklistBySurface := map[string][]ReviewerChecklistItem{}
	for _, item := range pack.ReviewerChecklist {
		checklistBySurface[item.SurfaceID] = append(checklistBySurface[item.SurfaceID], item.normalized())
	}
	var wireframesMissingChecklists []string
	var orphanChecklistSurfaces []string
	var checklistItemsMissingEvidence []string
	var checklistItemsMissingRoleLinks []string
	if pack.RequiresReviewerChecklist {
		for _, wireframe := range pack.Wireframes {
			if _, ok := checklistBySurface[wireframe.SurfaceID]; !ok {
				wireframesMissingChecklists = append(wireframesMissingChecklists, wireframe.SurfaceID)
			}
		}
		for surfaceID := range checklistBySurface {
			if !wireframeIDs[surfaceID] {
				orphanChecklistSurfaces = append(orphanChecklistSurfaces, surfaceID)
			}
		}
		for _, item := range pack.ReviewerChecklist {
			if len(item.EvidenceLinks) == 0 {
				checklistItemsMissingEvidence = append(checklistItemsMissingEvidence, item.ItemID)
			}
		}
		sort.Strings(wireframesMissingChecklists)
		sort.Strings(orphanChecklistSurfaces)
		sort.Strings(checklistItemsMissingEvidence)
	}

	decisionBySurface := map[string][]ReviewDecision{}
	for _, decision := range pack.DecisionLog {
		normalized := decision.normalized()
		decisionBySurface[normalized.SurfaceID] = append(decisionBySurface[normalized.SurfaceID], normalized)
	}
	var wireframesMissingDecisions []string
	var orphanDecisionSurfaces []string
	var unresolvedDecisionIDs []string
	var unresolvedDecisionsMissingFollowUps []string
	if pack.RequiresDecisionLog {
		for _, wireframe := range pack.Wireframes {
			if _, ok := decisionBySurface[wireframe.SurfaceID]; !ok {
				wireframesMissingDecisions = append(wireframesMissingDecisions, wireframe.SurfaceID)
			}
		}
		for surfaceID := range decisionBySurface {
			if !wireframeIDs[surfaceID] {
				orphanDecisionSurfaces = append(orphanDecisionSurfaces, surfaceID)
			}
		}
		for _, decision := range pack.DecisionLog {
			normalized := decision.normalized()
			status := strings.ToLower(normalized.Status)
			if status != "accepted" && status != "approved" && status != "resolved" && status != "waived" {
				unresolvedDecisionIDs = append(unresolvedDecisionIDs, normalized.DecisionID)
				if strings.TrimSpace(normalized.FollowUp) == "" {
					unresolvedDecisionsMissingFollowUps = append(unresolvedDecisionsMissingFollowUps, normalized.DecisionID)
				}
			}
		}
		sort.Strings(wireframesMissingDecisions)
		sort.Strings(orphanDecisionSurfaces)
		sort.Strings(unresolvedDecisionIDs)
		sort.Strings(unresolvedDecisionsMissingFollowUps)
	}

	checklistItemIDs := map[string]bool{}
	for _, item := range pack.ReviewerChecklist {
		checklistItemIDs[item.ItemID] = true
	}
	decisionIDs := map[string]bool{}
	for _, decision := range pack.DecisionLog {
		decisionIDs[decision.DecisionID] = true
	}
	assignmentIDs := map[string]bool{}
	roleAssignmentsBySurface := map[string][]ReviewRoleAssignment{}
	for _, assignment := range pack.RoleMatrix {
		normalized := assignment.normalized()
		assignmentIDs[normalized.AssignmentID] = true
		roleAssignmentsBySurface[normalized.SurfaceID] = append(roleAssignmentsBySurface[normalized.SurfaceID], normalized)
	}
	var wireframesMissingRoleAssignments []string
	var orphanRoleAssignmentSurfaces []string
	var roleAssignmentsMissingResponsibilities []string
	var roleAssignmentsMissingChecklistLinks []string
	var roleAssignmentsMissingDecisionLinks []string
	var decisionsMissingRoleLinks []string
	if pack.RequiresRoleMatrix {
		for _, wireframe := range pack.Wireframes {
			if _, ok := roleAssignmentsBySurface[wireframe.SurfaceID]; !ok {
				wireframesMissingRoleAssignments = append(wireframesMissingRoleAssignments, wireframe.SurfaceID)
			}
		}
		for surfaceID := range roleAssignmentsBySurface {
			if !wireframeIDs[surfaceID] {
				orphanRoleAssignmentSurfaces = append(orphanRoleAssignmentSurfaces, surfaceID)
			}
		}
		roleLinkedChecklistIDs := map[string]bool{}
		roleLinkedDecisionIDs := map[string]bool{}
		for _, assignment := range pack.RoleMatrix {
			normalized := assignment.normalized()
			if len(normalized.Responsibilities) == 0 {
				roleAssignmentsMissingResponsibilities = append(roleAssignmentsMissingResponsibilities, normalized.AssignmentID)
			}
			if len(normalized.ChecklistItemIDs) == 0 {
				roleAssignmentsMissingChecklistLinks = append(roleAssignmentsMissingChecklistLinks, normalized.AssignmentID)
			} else {
				missingLink := false
				for _, itemID := range normalized.ChecklistItemIDs {
					roleLinkedChecklistIDs[itemID] = true
					if !checklistItemIDs[itemID] {
						missingLink = true
					}
				}
				if missingLink {
					roleAssignmentsMissingChecklistLinks = append(roleAssignmentsMissingChecklistLinks, normalized.AssignmentID)
				}
			}
			if len(normalized.DecisionIDs) == 0 {
				roleAssignmentsMissingDecisionLinks = append(roleAssignmentsMissingDecisionLinks, normalized.AssignmentID)
			} else {
				missingLink := false
				for _, decisionID := range normalized.DecisionIDs {
					roleLinkedDecisionIDs[decisionID] = true
					if !decisionIDs[decisionID] {
						missingLink = true
					}
				}
				if missingLink {
					roleAssignmentsMissingDecisionLinks = append(roleAssignmentsMissingDecisionLinks, normalized.AssignmentID)
				}
			}
		}
		for _, item := range pack.ReviewerChecklist {
			if !roleLinkedChecklistIDs[item.ItemID] {
				checklistItemsMissingRoleLinks = append(checklistItemsMissingRoleLinks, item.ItemID)
			}
		}
		for _, decision := range pack.DecisionLog {
			if !roleLinkedDecisionIDs[decision.DecisionID] {
				decisionsMissingRoleLinks = append(decisionsMissingRoleLinks, decision.DecisionID)
			}
		}
		sort.Strings(wireframesMissingRoleAssignments)
		sort.Strings(orphanRoleAssignmentSurfaces)
		sort.Strings(roleAssignmentsMissingResponsibilities)
		sort.Strings(roleAssignmentsMissingChecklistLinks)
		sort.Strings(roleAssignmentsMissingDecisionLinks)
		sort.Strings(checklistItemsMissingRoleLinks)
		sort.Strings(decisionsMissingRoleLinks)
	}

	signoffsBySurface := map[string][]ReviewSignoff{}
	for _, signoff := range pack.SignoffLog {
		normalized := signoff.normalized()
		signoffsBySurface[normalized.SurfaceID] = append(signoffsBySurface[normalized.SurfaceID], normalized)
	}
	var wireframesMissingSignoffs []string
	var orphanSignoffSurfaces []string
	var signoffsMissingAssignments []string
	var signoffsMissingEvidence []string
	var signoffsMissingRequestedDates []string
	var signoffsMissingDueDates []string
	var signoffsMissingEscalationOwners []string
	var signoffsMissingReminderOwners []string
	var signoffsMissingNextReminders []string
	var signoffsMissingReminderCadence []string
	var signoffsWithBreachedSLA []string
	var unresolvedRequiredSignoffIDs []string
	var waivedSignoffsMissingMetadata []string
	if pack.RequiresSignoffLog {
		for _, wireframe := range pack.Wireframes {
			if _, ok := signoffsBySurface[wireframe.SurfaceID]; !ok {
				wireframesMissingSignoffs = append(wireframesMissingSignoffs, wireframe.SurfaceID)
			}
		}
		for surfaceID := range signoffsBySurface {
			if !wireframeIDs[surfaceID] {
				orphanSignoffSurfaces = append(orphanSignoffSurfaces, surfaceID)
			}
		}
		unresolvedStatuses := map[string]bool{
			"approved": true,
			"accepted": true,
			"resolved": true,
			"waived":   true,
			"deferred": true,
		}
		for _, signoff := range pack.SignoffLog {
			normalized := signoff.normalized()
			status := strings.ToLower(normalized.Status)
			if !assignmentIDs[normalized.AssignmentID] {
				signoffsMissingAssignments = append(signoffsMissingAssignments, normalized.SignoffID)
			}
			if status != "waived" && len(normalized.EvidenceLinks) == 0 {
				signoffsMissingEvidence = append(signoffsMissingEvidence, normalized.SignoffID)
			}
			if normalized.Required && strings.TrimSpace(normalized.RequestedAt) == "" {
				signoffsMissingRequestedDates = append(signoffsMissingRequestedDates, normalized.SignoffID)
			}
			if normalized.Required && strings.TrimSpace(normalized.DueAt) == "" {
				signoffsMissingDueDates = append(signoffsMissingDueDates, normalized.SignoffID)
			}
			if normalized.Required && strings.TrimSpace(normalized.EscalationOwner) == "" {
				signoffsMissingEscalationOwners = append(signoffsMissingEscalationOwners, normalized.SignoffID)
			}
			if normalized.Required && !unresolvedStatuses[status] && strings.TrimSpace(normalized.ReminderOwner) == "" {
				signoffsMissingReminderOwners = append(signoffsMissingReminderOwners, normalized.SignoffID)
			}
			if normalized.Required && !unresolvedStatuses[status] && strings.TrimSpace(normalized.NextReminderAt) == "" {
				signoffsMissingNextReminders = append(signoffsMissingNextReminders, normalized.SignoffID)
			}
			if normalized.Required && !unresolvedStatuses[status] && strings.TrimSpace(normalized.ReminderCadence) == "" {
				signoffsMissingReminderCadence = append(signoffsMissingReminderCadence, normalized.SignoffID)
			}
			if strings.ToLower(normalized.SLAStatus) == "breached" && status != "approved" && status != "accepted" && status != "resolved" {
				signoffsWithBreachedSLA = append(signoffsWithBreachedSLA, normalized.SignoffID)
			}
			if status == "waived" && (strings.TrimSpace(normalized.WaiverOwner) == "" || strings.TrimSpace(normalized.WaiverReason) == "") {
				waivedSignoffsMissingMetadata = append(waivedSignoffsMissingMetadata, normalized.SignoffID)
			}
			if normalized.Required && !unresolvedStatuses[status] {
				unresolvedRequiredSignoffIDs = append(unresolvedRequiredSignoffIDs, normalized.SignoffID)
			}
		}
		sort.Strings(wireframesMissingSignoffs)
		sort.Strings(orphanSignoffSurfaces)
		sort.Strings(signoffsMissingAssignments)
		sort.Strings(signoffsMissingEvidence)
		sort.Strings(signoffsMissingRequestedDates)
		sort.Strings(signoffsMissingDueDates)
		sort.Strings(signoffsMissingEscalationOwners)
		sort.Strings(signoffsMissingReminderOwners)
		sort.Strings(signoffsMissingNextReminders)
		sort.Strings(signoffsMissingReminderCadence)
		sort.Strings(signoffsWithBreachedSLA)
		sort.Strings(waivedSignoffsMissingMetadata)
		sort.Strings(unresolvedRequiredSignoffIDs)
	}

	blockerBySignoff := map[string][]ReviewBlocker{}
	blockerSurfaces := map[string]bool{}
	for _, blocker := range pack.BlockerLog {
		normalized := blocker.normalized()
		blockerSurfaces[normalized.SurfaceID] = true
		blockerBySignoff[normalized.SignoffID] = append(blockerBySignoff[normalized.SignoffID], normalized)
	}
	var blockersMissingSignoffLinks []string
	var blockersMissingEscalationOwners []string
	var blockersMissingNextActions []string
	var freezeExceptionsMissingOwners []string
	var freezeExceptionsMissingUntil []string
	var freezeExceptionsMissingApprovers []string
	var freezeExceptionsMissingApprovalDates []string
	var freezeExceptionsMissingRenewalOwners []string
	var freezeExceptionsMissingRenewalDates []string
	var orphanBlockerSurfaces []string
	var unresolvedRequiredSignoffsWithoutBlockers []string
	if pack.RequiresBlockerLog {
		signoffIDs := map[string]bool{}
		for _, signoff := range pack.SignoffLog {
			signoffIDs[signoff.SignoffID] = true
		}
		for _, blocker := range pack.BlockerLog {
			normalized := blocker.normalized()
			if !signoffIDs[normalized.SignoffID] {
				blockersMissingSignoffLinks = append(blockersMissingSignoffLinks, normalized.BlockerID)
			}
			if strings.TrimSpace(normalized.EscalationOwner) == "" {
				blockersMissingEscalationOwners = append(blockersMissingEscalationOwners, normalized.BlockerID)
			}
			if strings.TrimSpace(normalized.NextAction) == "" {
				blockersMissingNextActions = append(blockersMissingNextActions, normalized.BlockerID)
			}
			if normalized.FreezeException && strings.TrimSpace(normalized.FreezeOwner) == "" {
				freezeExceptionsMissingOwners = append(freezeExceptionsMissingOwners, normalized.BlockerID)
			}
			if normalized.FreezeException && strings.TrimSpace(normalized.FreezeUntil) == "" {
				freezeExceptionsMissingUntil = append(freezeExceptionsMissingUntil, normalized.BlockerID)
			}
			if normalized.FreezeException && strings.TrimSpace(normalized.FreezeApprovedBy) == "" {
				freezeExceptionsMissingApprovers = append(freezeExceptionsMissingApprovers, normalized.BlockerID)
			}
			if normalized.FreezeException && strings.TrimSpace(normalized.FreezeApprovedAt) == "" {
				freezeExceptionsMissingApprovalDates = append(freezeExceptionsMissingApprovalDates, normalized.BlockerID)
			}
			if normalized.FreezeException && strings.TrimSpace(normalized.FreezeRenewalOwner) == "" {
				freezeExceptionsMissingRenewalOwners = append(freezeExceptionsMissingRenewalOwners, normalized.BlockerID)
			}
			if normalized.FreezeException && strings.TrimSpace(normalized.FreezeRenewalBy) == "" {
				freezeExceptionsMissingRenewalDates = append(freezeExceptionsMissingRenewalDates, normalized.BlockerID)
			}
		}
		for surfaceID := range blockerSurfaces {
			if !wireframeIDs[surfaceID] {
				orphanBlockerSurfaces = append(orphanBlockerSurfaces, surfaceID)
			}
		}
		for _, signoffID := range unresolvedRequiredSignoffIDs {
			if _, ok := blockerBySignoff[signoffID]; !ok {
				unresolvedRequiredSignoffsWithoutBlockers = append(unresolvedRequiredSignoffsWithoutBlockers, signoffID)
			}
		}
		sort.Strings(blockersMissingSignoffLinks)
		sort.Strings(blockersMissingEscalationOwners)
		sort.Strings(blockersMissingNextActions)
		sort.Strings(freezeExceptionsMissingOwners)
		sort.Strings(freezeExceptionsMissingUntil)
		sort.Strings(freezeExceptionsMissingApprovers)
		sort.Strings(freezeExceptionsMissingApprovalDates)
		sort.Strings(freezeExceptionsMissingRenewalOwners)
		sort.Strings(freezeExceptionsMissingRenewalDates)
		sort.Strings(orphanBlockerSurfaces)
		sort.Strings(unresolvedRequiredSignoffsWithoutBlockers)
	}

	blockerTimelineByBlocker := map[string][]ReviewBlockerEvent{}
	for _, event := range pack.BlockerTimeline {
		normalized := event.normalized()
		blockerTimelineByBlocker[normalized.BlockerID] = append(blockerTimelineByBlocker[normalized.BlockerID], normalized)
	}
	var blockersMissingTimelineEvents []string
	var closedBlockersMissingResolutionEvents []string
	var orphanBlockerTimelineBlockerIDs []string
	var handoffEventsMissingTargets []string
	var handoffEventsMissingArtifacts []string
	var handoffEventsMissingAckOwners []string
	var handoffEventsMissingAckDates []string
	if pack.RequiresBlockerTimeline {
		blockerIDs := map[string]bool{}
		for _, blocker := range pack.BlockerLog {
			blockerIDs[blocker.BlockerID] = true
		}
		for blockerID := range blockerTimelineByBlocker {
			if !blockerIDs[blockerID] {
				orphanBlockerTimelineBlockerIDs = append(orphanBlockerTimelineBlockerIDs, blockerID)
			}
		}
		for _, blocker := range pack.BlockerLog {
			normalized := blocker.normalized()
			status := strings.ToLower(normalized.Status)
			if status != "resolved" && status != "closed" {
				if _, ok := blockerTimelineByBlocker[normalized.BlockerID]; !ok {
					blockersMissingTimelineEvents = append(blockersMissingTimelineEvents, normalized.BlockerID)
				}
			}
			if status == "resolved" || status == "closed" {
				hasResolution := false
				for _, event := range blockerTimelineByBlocker[normalized.BlockerID] {
					eventStatus := strings.ToLower(event.Status)
					if eventStatus == "resolved" || eventStatus == "closed" {
						hasResolution = true
						break
					}
				}
				if !hasResolution {
					closedBlockersMissingResolutionEvents = append(closedBlockersMissingResolutionEvents, normalized.BlockerID)
				}
			}
		}
		handoffStatuses := map[string]bool{"escalated": true, "handoff": true, "reassigned": true}
		for _, event := range pack.BlockerTimeline {
			normalized := event.normalized()
			if !handoffStatuses[strings.ToLower(normalized.Status)] {
				continue
			}
			if strings.TrimSpace(normalized.HandoffTo) == "" {
				handoffEventsMissingTargets = append(handoffEventsMissingTargets, normalized.EventID)
			}
			if strings.TrimSpace(normalized.ArtifactRef) == "" {
				handoffEventsMissingArtifacts = append(handoffEventsMissingArtifacts, normalized.EventID)
			}
			if strings.TrimSpace(normalized.AckOwner) == "" {
				handoffEventsMissingAckOwners = append(handoffEventsMissingAckOwners, normalized.EventID)
			}
			if strings.TrimSpace(normalized.AckAt) == "" {
				handoffEventsMissingAckDates = append(handoffEventsMissingAckDates, normalized.EventID)
			}
		}
		sort.Strings(blockersMissingTimelineEvents)
		sort.Strings(closedBlockersMissingResolutionEvents)
		sort.Strings(orphanBlockerTimelineBlockerIDs)
		sort.Strings(handoffEventsMissingTargets)
		sort.Strings(handoffEventsMissingArtifacts)
		sort.Strings(handoffEventsMissingAckOwners)
		sort.Strings(handoffEventsMissingAckDates)
	}

	ready := len(missingSections) == 0 &&
		len(objectivesMissingSignals) == 0 &&
		len(wireframesMissingBlocks) == 0 &&
		len(interactionsMissingStates) == 0 &&
		len(wireframesMissingChecklists) == 0 &&
		len(orphanChecklistSurfaces) == 0 &&
		len(checklistItemsMissingEvidence) == 0 &&
		len(wireframesMissingDecisions) == 0 &&
		len(orphanDecisionSurfaces) == 0 &&
		len(unresolvedDecisionsMissingFollowUps) == 0 &&
		len(wireframesMissingRoleAssignments) == 0 &&
		len(orphanRoleAssignmentSurfaces) == 0 &&
		len(roleAssignmentsMissingResponsibilities) == 0 &&
		len(roleAssignmentsMissingChecklistLinks) == 0 &&
		len(roleAssignmentsMissingDecisionLinks) == 0 &&
		len(wireframesMissingSignoffs) == 0 &&
		len(orphanSignoffSurfaces) == 0 &&
		len(signoffsMissingAssignments) == 0 &&
		len(signoffsMissingEvidence) == 0 &&
		len(signoffsMissingRequestedDates) == 0 &&
		len(signoffsMissingDueDates) == 0 &&
		len(signoffsMissingEscalationOwners) == 0 &&
		len(signoffsMissingReminderOwners) == 0 &&
		len(signoffsMissingNextReminders) == 0 &&
		len(signoffsMissingReminderCadence) == 0 &&
		len(signoffsWithBreachedSLA) == 0 &&
		len(waivedSignoffsMissingMetadata) == 0 &&
		len(blockersMissingSignoffLinks) == 0 &&
		len(blockersMissingEscalationOwners) == 0 &&
		len(blockersMissingNextActions) == 0 &&
		len(freezeExceptionsMissingOwners) == 0 &&
		len(freezeExceptionsMissingUntil) == 0 &&
		len(freezeExceptionsMissingApprovers) == 0 &&
		len(freezeExceptionsMissingApprovalDates) == 0 &&
		len(freezeExceptionsMissingRenewalOwners) == 0 &&
		len(freezeExceptionsMissingRenewalDates) == 0 &&
		len(blockersMissingTimelineEvents) == 0 &&
		len(closedBlockersMissingResolutionEvents) == 0 &&
		len(orphanBlockerSurfaces) == 0 &&
		len(orphanBlockerTimelineBlockerIDs) == 0 &&
		len(handoffEventsMissingTargets) == 0 &&
		len(handoffEventsMissingArtifacts) == 0 &&
		len(handoffEventsMissingAckOwners) == 0 &&
		len(handoffEventsMissingAckDates) == 0 &&
		len(unresolvedRequiredSignoffsWithoutBlockers) == 0

	return UIReviewPackAudit{
		Ready:                                     ready,
		ObjectiveCount:                            len(pack.Objectives),
		WireframeCount:                            len(pack.Wireframes),
		InteractionCount:                          len(pack.Interactions),
		OpenQuestionCount:                         len(pack.OpenQuestions),
		ChecklistCount:                            len(pack.ReviewerChecklist),
		DecisionCount:                             len(pack.DecisionLog),
		RoleAssignmentCount:                       len(pack.RoleMatrix),
		SignoffCount:                              len(pack.SignoffLog),
		BlockerCount:                              len(pack.BlockerLog),
		BlockerTimelineCount:                      len(pack.BlockerTimeline),
		MissingSections:                           missingSections,
		ObjectivesMissingSignals:                  objectivesMissingSignals,
		WireframesMissingBlocks:                   wireframesMissingBlocks,
		InteractionsMissingStates:                 interactionsMissingStates,
		UnresolvedQuestionIDs:                     unresolvedQuestionIDs,
		WireframesMissingChecklists:               wireframesMissingChecklists,
		OrphanChecklistSurfaces:                   orphanChecklistSurfaces,
		ChecklistItemsMissingEvidence:             checklistItemsMissingEvidence,
		ChecklistItemsMissingRoleLinks:            checklistItemsMissingRoleLinks,
		WireframesMissingDecisions:                wireframesMissingDecisions,
		OrphanDecisionSurfaces:                    orphanDecisionSurfaces,
		UnresolvedDecisionIDs:                     unresolvedDecisionIDs,
		UnresolvedDecisionsMissingFollowUps:       unresolvedDecisionsMissingFollowUps,
		WireframesMissingRoleAssignments:          wireframesMissingRoleAssignments,
		OrphanRoleAssignmentSurfaces:              orphanRoleAssignmentSurfaces,
		RoleAssignmentsMissingResponsibilities:    roleAssignmentsMissingResponsibilities,
		RoleAssignmentsMissingChecklistLinks:      roleAssignmentsMissingChecklistLinks,
		RoleAssignmentsMissingDecisionLinks:       roleAssignmentsMissingDecisionLinks,
		DecisionsMissingRoleLinks:                 decisionsMissingRoleLinks,
		WireframesMissingSignoffs:                 wireframesMissingSignoffs,
		OrphanSignoffSurfaces:                     orphanSignoffSurfaces,
		SignoffsMissingAssignments:                signoffsMissingAssignments,
		SignoffsMissingEvidence:                   signoffsMissingEvidence,
		SignoffsMissingRequestedDates:             signoffsMissingRequestedDates,
		SignoffsMissingDueDates:                   signoffsMissingDueDates,
		SignoffsMissingEscalationOwners:           signoffsMissingEscalationOwners,
		SignoffsMissingReminderOwners:             signoffsMissingReminderOwners,
		SignoffsMissingNextReminders:              signoffsMissingNextReminders,
		SignoffsMissingReminderCadence:            signoffsMissingReminderCadence,
		SignoffsWithBreachedSLA:                   signoffsWithBreachedSLA,
		UnresolvedRequiredSignoffIDs:              unresolvedRequiredSignoffIDs,
		WaivedSignoffsMissingMetadata:             waivedSignoffsMissingMetadata,
		BlockersMissingSignoffLinks:               blockersMissingSignoffLinks,
		BlockersMissingEscalationOwners:           blockersMissingEscalationOwners,
		BlockersMissingNextActions:                blockersMissingNextActions,
		FreezeExceptionsMissingOwners:             freezeExceptionsMissingOwners,
		FreezeExceptionsMissingUntil:              freezeExceptionsMissingUntil,
		FreezeExceptionsMissingApprovers:          freezeExceptionsMissingApprovers,
		FreezeExceptionsMissingApprovalDates:      freezeExceptionsMissingApprovalDates,
		FreezeExceptionsMissingRenewalOwners:      freezeExceptionsMissingRenewalOwners,
		FreezeExceptionsMissingRenewalDates:       freezeExceptionsMissingRenewalDates,
		BlockersMissingTimelineEvents:             blockersMissingTimelineEvents,
		ClosedBlockersMissingResolutionEvents:     closedBlockersMissingResolutionEvents,
		OrphanBlockerSurfaces:                     orphanBlockerSurfaces,
		OrphanBlockerTimelineBlockerIDs:           orphanBlockerTimelineBlockerIDs,
		HandoffEventsMissingTargets:               handoffEventsMissingTargets,
		HandoffEventsMissingArtifacts:             handoffEventsMissingArtifacts,
		HandoffEventsMissingAckOwners:             handoffEventsMissingAckOwners,
		HandoffEventsMissingAckDates:              handoffEventsMissingAckDates,
		UnresolvedRequiredSignoffsWithoutBlockers: unresolvedRequiredSignoffsWithoutBlockers,
	}
}

type ConsoleInteractionAuditor struct{}

func (ConsoleInteractionAuditor) Audit(draft ConsoleInteractionDraft) ConsoleInteractionAudit {
	routeIndex := draft.Architecture.RouteIndex()
	var missingSurfaces []string
	var missingFilters []string
	missingActions := map[string][]string{}
	var missingBatchActions []string
	missingStates := map[string][]string{}
	permissionGaps := map[string][]string{}
	referencedRoles := map[string]bool{}
	var missingPersonas []string
	var missingWireframes []string
	var missingReviewFocus []string
	var missingDecisionPrompts []string

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

		if len(surface.Filters) == 0 {
			missingFilters = append(missingFilters, contract.SurfaceName)
		}

		availableActionIDs := map[string]bool{}
		for _, actionID := range surface.ActionIDs() {
			availableActionIDs[actionID] = true
		}
		var missingActionIDs []string
		for _, actionID := range contract.RequiredActionIDs {
			if !availableActionIDs[actionID] {
				missingActionIDs = append(missingActionIDs, actionID)
			}
		}
		if len(missingActionIDs) > 0 {
			sort.Strings(missingActionIDs)
			missingActions[contract.SurfaceName] = missingActionIDs
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
				missingBatchActions = append(missingBatchActions, contract.SurfaceName)
			}
		}

		haveStates := map[string]bool{}
		for _, stateName := range surface.StateNames() {
			haveStates[stateName] = true
		}
		requiredStates := contract.RequiredStates
		if len(requiredStates) == 0 {
			requiredStates = requiredSurfaceStates
		}
		var missingStateIDs []string
		for _, stateName := range requiredStates {
			if !haveStates[stateName] {
				missingStateIDs = append(missingStateIDs, stateName)
			}
		}
		if len(missingStateIDs) > 0 {
			sort.Strings(missingStateIDs)
			missingStates[contract.SurfaceName] = missingStateIDs
		}

		for _, role := range contract.PermissionRule.AllowedRoles {
			referencedRoles[role] = true
		}
		for _, role := range contract.PermissionRule.DeniedRoles {
			referencedRoles[role] = true
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

	var uncoveredRoles []string
	for _, role := range draft.RequiredRoles {
		if !referencedRoles[role] {
			uncoveredRoles = append(uncoveredRoles, role)
		}
	}

	sort.Strings(missingSurfaces)
	sort.Strings(missingFilters)
	sort.Strings(missingBatchActions)
	sort.Strings(uncoveredRoles)
	sort.Strings(missingPersonas)
	sort.Strings(missingWireframes)
	sort.Strings(missingReviewFocus)
	sort.Strings(missingDecisionPrompts)

	return ConsoleInteractionAudit{
		Name:                           draft.Name,
		Version:                        draft.Version,
		ContractCount:                  len(draft.Contracts),
		MissingSurfaces:                missingSurfaces,
		SurfacesMissingFilters:         missingFilters,
		SurfacesMissingActions:         missingActions,
		SurfacesMissingBatchActions:    missingBatchActions,
		SurfacesMissingStates:          missingStates,
		PermissionGaps:                 permissionGaps,
		UncoveredRoles:                 uncoveredRoles,
		SurfacesMissingPrimaryPersonas: missingPersonas,
		SurfacesMissingWireframeLinks:  missingWireframes,
		SurfacesMissingReviewFocus:     missingReviewFocus,
		SurfacesMissingDecisionPrompts: missingDecisionPrompts,
	}
}

type UIAcceptanceAudit struct {
	Name                      string   `json:"name"`
	Version                   string   `json:"version"`
	PermissionGaps            []string `json:"permission_gaps,omitempty"`
	FailingDataChecks         []string `json:"failing_data_checks,omitempty"`
	FailingPerformanceBudgets []string `json:"failing_performance_budgets,omitempty"`
	FailingUsabilityJourneys  []string `json:"failing_usability_journeys,omitempty"`
	IncompleteAuditTrails     []string `json:"incomplete_audit_trails,omitempty"`
	DocumentationComplete     bool     `json:"documentation_complete,omitempty"`
}

func (a UIAcceptanceAudit) ReleaseReady() bool {
	return len(a.PermissionGaps) == 0 &&
		len(a.FailingDataChecks) == 0 &&
		len(a.FailingPerformanceBudgets) == 0 &&
		len(a.FailingUsabilityJourneys) == 0 &&
		len(a.IncompleteAuditTrails) == 0 &&
		a.DocumentationComplete
}

func (a UIAcceptanceAudit) ReadinessScore() float64 {
	checks := []bool{
		len(a.PermissionGaps) == 0,
		len(a.FailingDataChecks) == 0,
		len(a.FailingPerformanceBudgets) == 0,
		len(a.FailingUsabilityJourneys) == 0,
		len(a.IncompleteAuditTrails) == 0,
		a.DocumentationComplete,
	}
	passed := 0
	for _, check := range checks {
		if check {
			passed++
		}
	}
	return round1((float64(passed) / float64(len(checks))) * 100)
}

type UIAcceptanceLibrary struct{}

func (UIAcceptanceLibrary) Audit(suite UIAcceptanceSuite) UIAcceptanceAudit {
	var permissionGaps []string
	for _, scenario := range suite.RolePermissions {
		if missing := scenario.MissingCoverage(); len(missing) > 0 {
			permissionGaps = append(permissionGaps, scenario.ScreenID+": missing="+strings.Join(missing, ", "))
		}
	}
	var failingDataChecks []string
	for _, check := range suite.DataAccuracyChecks {
		if !check.Passes() {
			failingDataChecks = append(failingDataChecks, check.ScreenID+"."+check.MetricID+": delta="+format1(check.ObservedDelta)+" freshness="+itoa(check.ObservedFreshnessSeconds)+"s")
		}
	}
	var failingPerformance []string
	for _, budget := range suite.PerformanceBudgets {
		if !budget.WithinBudget() {
			entry := budget.SurfaceID + "." + budget.Interaction + ": p95=" + itoa(budget.ObservedP95MS) + "ms"
			if budget.TargetTTIMS > 0 {
				entry += " tti=" + itoa(budget.ObservedTTIMS) + "ms"
			}
			failingPerformance = append(failingPerformance, entry)
		}
	}
	var failingUsability []string
	for _, journey := range suite.UsabilityJourneys {
		if !journey.Passes() {
			failingUsability = append(failingUsability, journey.JourneyID+": steps="+itoa(journey.ObservedSteps)+"/"+itoa(journey.ExpectedMaxSteps))
		}
	}
	var incompleteAudit []string
	for _, requirement := range suite.AuditRequirements {
		if requirement.Complete() {
			continue
		}
		var parts []string
		if missingFields := requirement.MissingFields(); len(missingFields) > 0 {
			parts = append(parts, "missing_fields="+strings.Join(missingFields, ", "))
		}
		if !requirement.RetentionMet() {
			parts = append(parts, "retention="+itoa(requirement.ObservedRetentionDays)+"/"+itoa(requirement.RetentionDays)+"d")
		}
		incompleteAudit = append(incompleteAudit, requirement.EventType+": "+strings.Join(parts, " "))
	}
	return UIAcceptanceAudit{
		Name:                      suite.Name,
		Version:                   suite.Version,
		PermissionGaps:            permissionGaps,
		FailingDataChecks:         failingDataChecks,
		FailingPerformanceBudgets: failingPerformance,
		FailingUsabilityJourneys:  failingUsability,
		IncompleteAuditTrails:     incompleteAudit,
		DocumentationComplete:     suite.DocumentationComplete,
	}
}

func RenderDesignSystemReport(system DesignSystem, audit DesignSystemAudit) string {
	var lines []string
	lines = append(lines,
		"# Design System Report",
		"",
		"- Name: "+system.Name,
		"- Version: "+system.Version,
		"- Components: "+itoa(audit.ComponentCount),
		"- Release Ready Components: "+itoa(len(audit.ReleaseReadyComponents)),
		"- Readiness Score: "+format1(audit.ReadinessScore()),
		"",
		"## Token Foundations",
		"",
	)
	counts := audit.TokenCounts
	for _, category := range foundationCategories {
		lines = append(lines, "- "+category+": "+itoa(counts[category]))
	}
	lines = append(lines, "", "## Component Status", "")
	if len(system.Components) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, component := range system.Components {
			states := joinOrNone(component.StateCoverage())
			missingStates := joinOrNone(component.MissingRequiredStates())
			undefinedTokens := joinOrNone(audit.UndefinedTokenRefs[component.Name])
			lines = append(lines,
				"- "+component.Name+": readiness="+component.Readiness+
					" docs="+boolString(component.DocumentationComplete)+
					" a11y="+boolString(len(component.AccessibilityRequirements) > 0)+
					" states="+states+
					" missing_states="+missingStates+
					" undefined_tokens="+undefinedTokens,
			)
		}
	}
	lines = append(lines, "", "## Gaps", "")
	lines = append(lines, "- Missing docs: "+joinOrNone(audit.ComponentsMissingDocs))
	lines = append(lines, "- Missing accessibility: "+joinOrNone(audit.ComponentsMissingAccessibility))
	lines = append(lines, "- Missing interaction states: "+joinOrNone(audit.ComponentsMissingStates))
	if len(audit.UndefinedTokenRefs) == 0 {
		lines = append(lines, "- Undefined token refs: none")
	} else {
		var refs []string
		keys := sortedKeys(audit.UndefinedTokenRefs)
		for _, component := range keys {
			refs = append(refs, component+"="+strings.Join(audit.UndefinedTokenRefs[component], ", "))
		}
		lines = append(lines, "- Undefined token refs: "+strings.Join(refs, "; "))
	}
	lines = append(lines, "- Orphan tokens: "+joinOrNone(audit.TokenOrphans))
	return strings.Join(lines, "\n") + "\n"
}

func RenderConsoleTopBarReport(topBar ConsoleTopBar, audit ConsoleTopBarAudit) string {
	var lines []string
	lines = append(lines,
		"# Console Top Bar Report",
		"",
		"- Name: "+topBar.Name,
		"- Global Search: "+boolString(topBar.HasGlobalSearch()),
		"- Environment Switch: "+joinOrNone(topBar.EnvironmentOptions),
		"- Time Range Switch: "+joinOrNone(topBar.TimeRangeOptions),
		"- Alert Entry: "+joinOrNone(topBar.AlertChannels),
		"- Command Trigger: "+fallback(topBar.CommandEntry.TriggerLabel, "none"),
		"- Command Shortcut: "+fallback(topBar.CommandEntry.Shortcut, "none"),
		"- Command Count: "+itoa(audit.CommandCount),
		"- Release Ready: "+boolString(audit.ReleaseReady()),
		"",
		"## Command Palette",
		"",
	)
	if len(topBar.CommandEntry.Commands) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, command := range topBar.CommandEntry.Commands {
			lines = append(lines, "- "+command.ID+": "+command.Title+" ["+command.Section+"] shortcut="+fallback(command.Shortcut, "none"))
		}
	}
	lines = append(lines, "", "## Gaps", "")
	lines = append(lines, "- Missing capabilities: "+joinOrNone(audit.MissingCapabilities))
	lines = append(lines, "- Documentation complete: "+boolString(audit.DocumentationComplete))
	lines = append(lines, "- Accessibility complete: "+boolString(audit.AccessibilityComplete))
	lines = append(lines, "- Cmd/Ctrl+K supported: "+boolString(audit.CommandShortcutSupported))
	return strings.Join(lines, "\n") + "\n"
}

func RenderInformationArchitectureReport(architecture InformationArchitecture, audit InformationArchitectureAudit) string {
	var lines []string
	lines = append(lines,
		"# Information Architecture Report",
		"",
		"- Navigation Nodes: "+itoa(audit.TotalNavigationNodes),
		"- Routes: "+itoa(audit.TotalRoutes),
		"- Healthy: "+boolString(audit.Healthy()),
		"",
		"## Navigation Tree",
		"",
	)
	entries := architecture.NavigationEntries()
	if len(entries) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, entry := range entries {
			lines = append(lines, "- "+strings.Repeat("  ", entry.Depth)+entry.Title+" ("+entry.Path+") screen="+fallback(entry.ScreenID, "none"))
		}
	}
	lines = append(lines, "", "## Route Registry", "")
	if len(architecture.Routes) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, route := range architecture.Routes {
			normalized := route
			normalized.normalize()
			lines = append(lines, "- "+normalized.Path+": screen="+normalized.ScreenID+" title="+normalized.Title+" nav_node="+fallback(normalized.NavNodeID, "none"))
		}
	}
	lines = append(lines, "", "## Audit", "")
	lines = append(lines, "- Duplicate routes: "+joinOrNone(audit.DuplicateRoutes))
	if len(audit.MissingRouteNodes) == 0 {
		lines = append(lines, "- Missing route nodes: none")
	} else {
		var parts []string
		for _, key := range sortedStringMapKeys(audit.MissingRouteNodes) {
			parts = append(parts, key+"="+audit.MissingRouteNodes[key])
		}
		lines = append(lines, "- Missing route nodes: "+strings.Join(parts, ", "))
	}
	if len(audit.SecondaryNavGaps) == 0 {
		lines = append(lines, "- Secondary nav gaps: none")
	} else {
		var parts []string
		for _, key := range sortedKeys(audit.SecondaryNavGaps) {
			parts = append(parts, key+"="+strings.Join(audit.SecondaryNavGaps[key], ", "))
		}
		lines = append(lines, "- Secondary nav gaps: "+strings.Join(parts, "; "))
	}
	lines = append(lines, "- Orphan routes: "+joinOrNone(audit.OrphanRoutes))
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIAcceptanceReport(suite UIAcceptanceSuite, audit UIAcceptanceAudit) string {
	var lines []string
	lines = append(lines,
		"# UI Acceptance Report",
		"",
		"- Name: "+suite.Name,
		"- Version: "+suite.Version,
		"- Role/Permission Scenarios: "+itoa(len(suite.RolePermissions)),
		"- Data Accuracy Checks: "+itoa(len(suite.DataAccuracyChecks)),
		"- Performance Budgets: "+itoa(len(suite.PerformanceBudgets)),
		"- Usability Journeys: "+itoa(len(suite.UsabilityJourneys)),
		"- Audit Requirements: "+itoa(len(suite.AuditRequirements)),
		"- Readiness Score: "+format1(audit.ReadinessScore()),
		"- Release Ready: "+boolString(audit.ReleaseReady()),
		"",
		"## Coverage",
		"",
	)
	if len(suite.RolePermissions) == 0 {
		lines = append(lines, "- Role/Permission: none")
	} else {
		for _, scenario := range suite.RolePermissions {
			lines = append(lines, "- Role/Permission "+scenario.ScreenID+": allow="+joinOrNone(scenario.AllowedRoles)+" deny="+joinOrNone(scenario.DeniedRoles)+" audit_event="+fallback(scenario.AuditEvent, "none"))
		}
	}
	if len(suite.DataAccuracyChecks) == 0 {
		lines = append(lines, "- Data Accuracy: none")
	} else {
		for _, check := range suite.DataAccuracyChecks {
			lines = append(lines, "- Data Accuracy "+check.ScreenID+"."+check.MetricID+": delta="+format1(check.ObservedDelta)+" tolerance="+format1(check.Tolerance)+" freshness="+itoa(check.ObservedFreshnessSeconds)+"/"+itoa(check.FreshnessSLOSeconds)+"s")
		}
	}
	if len(suite.PerformanceBudgets) == 0 {
		lines = append(lines, "- Performance: none")
	} else {
		for _, budget := range suite.PerformanceBudgets {
			ttiText := ""
			if budget.TargetTTIMS > 0 {
				ttiText = " tti=" + itoa(budget.ObservedTTIMS) + "/" + itoa(budget.TargetTTIMS) + "ms"
			}
			lines = append(lines, "- Performance "+budget.SurfaceID+"."+budget.Interaction+": p95="+itoa(budget.ObservedP95MS)+"/"+itoa(budget.TargetP95MS)+"ms"+ttiText)
		}
	}
	if len(suite.UsabilityJourneys) == 0 {
		lines = append(lines, "- Usability: none")
	} else {
		for _, journey := range suite.UsabilityJourneys {
			lines = append(lines, "- Usability "+journey.JourneyID+": steps="+itoa(journey.ObservedSteps)+"/"+itoa(journey.ExpectedMaxSteps)+" keyboard="+boolString(journey.KeyboardAccessible)+" empty_state="+boolString(journey.EmptyStateGuidance)+" recovery="+boolString(journey.RecoverySupport))
		}
	}
	if len(suite.AuditRequirements) == 0 {
		lines = append(lines, "- Audit: none")
	} else {
		for _, requirement := range suite.AuditRequirements {
			lines = append(lines, "- Audit "+requirement.EventType+": fields="+itoa(len(requirement.EmittedFields))+"/"+itoa(len(requirement.RequiredFields))+" retention="+itoa(requirement.ObservedRetentionDays)+"/"+itoa(requirement.RetentionDays)+"d")
		}
	}
	lines = append(lines, "", "## Gaps", "")
	lines = append(lines, "- Role/Permission gaps: "+joinOrNone(audit.PermissionGaps))
	lines = append(lines, "- Data accuracy gaps: "+joinOrNone(audit.FailingDataChecks))
	lines = append(lines, "- Performance gaps: "+joinOrNone(audit.FailingPerformanceBudgets))
	lines = append(lines, "- Usability gaps: "+joinOrNone(audit.FailingUsabilityJourneys))
	lines = append(lines, "- Audit completeness gaps: "+joinOrNone(audit.IncompleteAuditTrails))
	return strings.Join(lines, "\n") + "\n"
}

func RenderConsoleIAReport(architecture ConsoleIA, audit ConsoleIAAudit) string {
	var lines []string
	lines = append(lines,
		"# Console Information Architecture Report",
		"",
		"- Name: "+architecture.Name,
		"- Version: "+architecture.Version,
		"- Navigation Items: "+itoa(audit.NavigationCount),
		"- Surfaces: "+itoa(audit.SurfaceCount),
		"- Readiness Score: "+format1(audit.ReadinessScore()),
		"",
		"## Global Header",
		"",
		"- Name: "+fallback(architecture.TopBar.Name, "none"),
		"- Release Ready: "+boolString(audit.TopBarAudit.ReleaseReady()),
		"- Missing capabilities: "+joinOrNone(audit.TopBarAudit.MissingCapabilities),
		"- Command Count: "+itoa(audit.TopBarAudit.CommandCount),
		"- Cmd/Ctrl+K supported: "+boolString(audit.TopBarAudit.CommandShortcutSupported),
		"",
		"## Navigation",
		"",
	)
	if len(architecture.Navigation) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, item := range architecture.Navigation {
			lines = append(lines, "- "+item.Section+" / "+item.Name+": route="+item.Route+" badge="+itoa(item.BadgeCount)+" icon="+fallback(item.Icon, "none"))
		}
	}
	lines = append(lines, "", "## Surface Coverage", "")
	if len(architecture.Surfaces) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, surface := range architecture.Surfaces {
			var filterNames []string
			for _, filter := range surface.Filters {
				filterNames = append(filterNames, filter.Name)
			}
			var actionLabels []string
			for _, action := range surface.TopBarActions {
				actionLabels = append(actionLabels, action.Label)
			}
			unresolved := audit.UnresolvedStateActions[surface.Name]
			unresolvedText := "none"
			if len(unresolved) > 0 {
				var parts []string
				for _, state := range sortedKeys(unresolved) {
					parts = append(parts, state+"="+strings.Join(unresolved[state], ", "))
				}
				unresolvedText = strings.Join(parts, "; ")
			}
			lines = append(lines, "- "+surface.Name+
				": route="+surface.Route+
				" filters="+joinOrNone(filterNames)+
				" actions="+joinOrNone(actionLabels)+
				" states="+joinOrNone(surface.StateNames())+
				" missing_states="+joinOrNone(surface.MissingRequiredStates())+
				" states_without_actions="+joinOrNone(audit.StatesMissingActions[surface.Name])+
				" unresolved_state_actions="+unresolvedText)
		}
	}
	lines = append(lines, "", "## Audit", "")
	lines = append(lines, "- Surfaces missing filters: "+joinOrNone(audit.SurfacesMissingFilters))
	lines = append(lines, "- Surfaces missing actions: "+joinOrNone(audit.SurfacesMissingActions))
	if len(audit.SurfacesMissingStates) == 0 {
		lines = append(lines, "- Surfaces missing states: none")
	} else {
		var parts []string
		for _, surface := range sortedKeys(audit.SurfacesMissingStates) {
			parts = append(parts, surface+"="+strings.Join(audit.SurfacesMissingStates[surface], ", "))
		}
		lines = append(lines, "- Surfaces missing states: "+strings.Join(parts, "; "))
	}
	lines = append(lines, "- States missing actions: "+formatListMap(audit.StatesMissingActions))
	lines = append(lines, "- Undefined state actions: "+formatNestedListMap(audit.UnresolvedStateActions))
	lines = append(lines, "- Orphan navigation routes: "+joinOrNone(audit.OrphanNavigationRoutes))
	lines = append(lines, "- Unnavigable surfaces: "+joinOrNone(audit.UnnavigableSurfaces))
	return strings.Join(lines, "\n") + "\n"
}

func RenderConsoleInteractionReport(draft ConsoleInteractionDraft, audit ConsoleInteractionAudit) string {
	routeIndex := draft.Architecture.RouteIndex()
	var lines []string
	lines = append(lines,
		"# Console Interaction Draft Report",
		"",
		"- Name: "+draft.Name,
		"- Version: "+draft.Version,
		"- Critical Pages: "+itoa(len(draft.Contracts)),
		"- Required Roles: "+joinOrNone(draft.RequiredRoles),
		"- Readiness Score: "+format1(audit.ReadinessScore()),
		"- Release Ready: "+boolString(audit.ReleaseReady()),
		"",
		"## Page Coverage",
		"",
	)
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
				lines = append(lines, "- "+contract.SurfaceName+": missing surface definition")
				continue
			}
			requiredActions := joinOrNone(contract.RequiredActionIDs)
			availableActions := joinOrNone(surface.ActionIDs())
			batchMode := "optional"
			if contract.RequiresBatchActions {
				batchMode = "required"
			}
			permissionState := "incomplete"
			if contract.PermissionRule.Complete() {
				permissionState = "complete"
			}
			lines = append(lines, "- "+contract.SurfaceName+
				": route="+surface.Route+
				" required_actions="+requiredActions+
				" available_actions="+availableActions+
				" filters="+itoa(len(surface.Filters))+
				" states="+joinOrNone(surface.StateNames())+
				" batch="+batchMode+
				" permissions="+permissionState)
			lines = append(lines, "  persona="+fallback(contract.PrimaryPersona, "none")+
				" wireframe="+fallback(contract.LinkedWireframeID, "none")+
				" review_focus="+joinCSVOrNone(contract.ReviewFocusAreas)+
				" decision_prompts="+joinCSVOrNone(contract.DecisionPrompts))
		}
	}
	lines = append(lines, "", "## Gaps", "")
	lines = append(lines, "- Missing surfaces: "+joinOrNone(audit.MissingSurfaces))
	lines = append(lines, "- Pages missing filters: "+joinOrNone(audit.SurfacesMissingFilters))
	lines = append(lines, "- Pages missing actions: "+formatListMap(audit.SurfacesMissingActions))
	lines = append(lines, "- Pages missing batch actions: "+joinOrNone(audit.SurfacesMissingBatchActions))
	lines = append(lines, "- Pages missing states: "+formatListMap(audit.SurfacesMissingStates))
	lines = append(lines, "- Permission gaps: "+formatListMap(audit.PermissionGaps))
	lines = append(lines, "- Uncovered roles: "+joinOrNone(audit.UncoveredRoles))
	lines = append(lines, "- Pages missing personas: "+joinOrNone(audit.SurfacesMissingPrimaryPersonas))
	lines = append(lines, "- Pages missing wireframe links: "+joinOrNone(audit.SurfacesMissingWireframeLinks))
	lines = append(lines, "- Pages missing review focus: "+joinOrNone(audit.SurfacesMissingReviewFocus))
	lines = append(lines, "- Pages missing decision prompts: "+joinOrNone(audit.SurfacesMissingDecisionPrompts))
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewPackReport(pack UIReviewPack, audit UIReviewPackAudit) string {
	var lines []string
	lines = append(lines,
		"# UI Review Pack",
		"",
		"- Issue: "+pack.IssueID+" "+pack.Title,
		"- Version: "+pack.Version,
		"- Audit: "+audit.Summary(),
		"",
		"## Objectives",
	)
	if len(pack.Objectives) == 0 {
		lines = append(lines, "- none")
	} else {
		for _, objective := range pack.Objectives {
			normalized := objective.normalized()
			lines = append(lines,
				"- "+normalized.ObjectiveID+": "+normalized.Title+
					" persona="+normalized.Persona+
					" priority="+normalized.Priority)
			lines = append(lines,
				"  outcome="+normalized.Outcome+
					" success_signal="+normalized.SuccessSignal+
					" dependencies="+joinCSVOrNone(normalized.Dependencies))
		}
	}
	lines = append(lines, "")
	lines = append(lines, "## Open Questions")
	lines = append(lines, "- Unresolved questions: "+joinOrNone(audit.UnresolvedQuestionIDs))
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewSignoffSLADashboard(pack UIReviewPack) string {
	type entry struct {
		SignoffID       string
		Role            string
		SurfaceID       string
		Status          string
		SLAStatus       string
		RequestedAt     string
		DueAt           string
		EscalationOwner string
		Required        bool
		Evidence        string
	}
	var entries []entry
	stateCounts := map[string]int{}
	ownerCounts := map[string]int{}
	for _, signoff := range pack.SignoffLog {
		s := signoff.normalized()
		e := entry{
			SignoffID:       s.SignoffID,
			Role:            s.Role,
			SurfaceID:       s.SurfaceID,
			Status:          s.Status,
			SLAStatus:       s.SLAStatus,
			RequestedAt:     fallback(s.RequestedAt, "none"),
			DueAt:           fallback(s.DueAt, "none"),
			EscalationOwner: fallback(s.EscalationOwner, "none"),
			Required:        s.Required,
			Evidence:        joinCSVOrNone(s.EvidenceLinks),
		}
		entries = append(entries, e)
		stateCounts[e.SLAStatus]++
		ownerCounts[e.EscalationOwner]++
	}
	var lines []string
	lines = append(lines, "# UI Review Sign-off SLA Dashboard", "", "- Issue: "+pack.IssueID+" "+pack.Title, "- Version: "+pack.Version, "- Sign-offs: "+itoa(len(entries)), "- Escalation owners: "+itoa(len(ownerCounts)), "", "## SLA States")
	for _, k := range sortedStringMapKeys(intStringMap(stateCounts)) {
		lines = append(lines, "- "+k+": "+itoa(stateCounts[k]))
	}
	if len(stateCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## Escalation Owners")
	for _, k := range sortedStringMapKeys(intStringMap(ownerCounts)) {
		lines = append(lines, "- "+k+": "+itoa(ownerCounts[k]))
	}
	if len(ownerCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## Sign-offs")
	for _, e := range entries {
		lines = append(lines, "- "+e.SignoffID+": role="+e.Role+" surface="+e.SurfaceID+" status="+e.Status+" sla="+e.SLAStatus+" requested_at="+e.RequestedAt+" due_at="+e.DueAt+" escalation_owner="+e.EscalationOwner)
		lines = append(lines, "  required="+boolString(e.Required)+" evidence="+e.Evidence)
	}
	if len(entries) == 0 {
		lines = append(lines, "- none")
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewSignoffReminderQueue(pack UIReviewPack) string {
	type entry struct {
		EntryID         string
		SignoffID       string
		Role            string
		SurfaceID       string
		Status          string
		SLAStatus       string
		ReminderOwner   string
		ReminderChannel string
		LastReminderAt  string
		NextReminderAt  string
		DueAt           string
		Summary         string
	}
	var entries []entry
	ownerCounts := map[string]int{}
	channelCounts := map[string]int{}
	for _, signoff := range pack.SignoffLog {
		s := signoff.normalized()
		status := strings.ToLower(s.Status)
		if status == "approved" || status == "accepted" || status == "resolved" || status == "waived" || status == "deferred" {
			continue
		}
		if strings.TrimSpace(s.ReminderOwner) == "" {
			continue
		}
		e := entry{
			EntryID:         "rem-" + s.SignoffID,
			SignoffID:       s.SignoffID,
			Role:            s.Role,
			SurfaceID:       s.SurfaceID,
			Status:          s.Status,
			SLAStatus:       s.SLAStatus,
			ReminderOwner:   s.ReminderOwner,
			ReminderChannel: fallback(s.ReminderChannel, "none"),
			LastReminderAt:  fallback(s.LastReminderAt, "none"),
			NextReminderAt:  fallback(s.NextReminderAt, "none"),
			DueAt:           fallback(s.DueAt, "none"),
			Summary:         fallback(s.Notes, "none"),
		}
		entries = append(entries, e)
		ownerCounts[e.ReminderOwner]++
		channelCounts[e.ReminderChannel]++
	}
	var lines []string
	lines = append(lines, "# UI Review Sign-off Reminder Queue", "", "- Issue: "+pack.IssueID+" "+pack.Title, "- Version: "+pack.Version, "- Reminders: "+itoa(len(entries)), "- Owners: "+itoa(len(ownerCounts)), "", "## By Owner")
	for _, k := range sortedStringMapKeys(intStringMap(ownerCounts)) {
		lines = append(lines, "- "+k+": reminders="+itoa(ownerCounts[k]))
	}
	if len(ownerCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## By Channel")
	for _, k := range sortedStringMapKeys(intStringMap(channelCounts)) {
		lines = append(lines, "- "+k+": "+itoa(channelCounts[k]))
	}
	if len(channelCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## Items")
	for _, e := range entries {
		lines = append(lines, "- "+e.EntryID+": signoff="+e.SignoffID+" role="+e.Role+" surface="+e.SurfaceID+" status="+e.Status+" sla="+e.SLAStatus+" owner="+e.ReminderOwner+" channel="+e.ReminderChannel)
		lines = append(lines, "  last_reminder_at="+e.LastReminderAt+" next_reminder_at="+e.NextReminderAt+" due_at="+e.DueAt+" summary="+e.Summary)
	}
	if len(entries) == 0 {
		lines = append(lines, "- none")
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewSignoffBreachBoard(pack UIReviewPack) string {
	type entry struct {
		EntryID         string
		SignoffID       string
		Role            string
		SurfaceID       string
		Status          string
		SLAStatus       string
		EscalationOwner string
		RequestedAt     string
		DueAt           string
		LinkedBlockers  string
		Summary         string
	}
	blockersBySignoff := map[string][]string{}
	for _, blocker := range pack.BlockerLog {
		blockersBySignoff[blocker.SignoffID] = append(blockersBySignoff[blocker.SignoffID], blocker.BlockerID)
	}
	var entries []entry
	stateCounts := map[string]int{}
	ownerCounts := map[string]int{}
	for _, signoff := range pack.SignoffLog {
		s := signoff.normalized()
		if strings.ToLower(s.Status) == "approved" || strings.ToLower(s.Status) == "accepted" || strings.ToLower(s.Status) == "resolved" {
			continue
		}
		if strings.ToLower(s.SLAStatus) == "met" {
			continue
		}
		e := entry{
			EntryID:         "breach-" + s.SignoffID,
			SignoffID:       s.SignoffID,
			Role:            s.Role,
			SurfaceID:       s.SurfaceID,
			Status:          s.Status,
			SLAStatus:       s.SLAStatus,
			EscalationOwner: fallback(s.EscalationOwner, "none"),
			RequestedAt:     fallback(s.RequestedAt, "none"),
			DueAt:           fallback(s.DueAt, "none"),
			LinkedBlockers:  joinCSVOrNone(blockersBySignoff[s.SignoffID]),
			Summary:         fallback(s.Notes, "none"),
		}
		entries = append(entries, e)
		stateCounts[e.SLAStatus]++
		ownerCounts[e.EscalationOwner]++
	}
	var lines []string
	lines = append(lines, "# UI Review Sign-off Breach Board", "", "- Issue: "+pack.IssueID+" "+pack.Title, "- Version: "+pack.Version, "- Breach items: "+itoa(len(entries)), "- Escalation owners: "+itoa(len(ownerCounts)), "", "## SLA States")
	for _, k := range sortedStringMapKeys(intStringMap(stateCounts)) {
		lines = append(lines, "- "+k+": "+itoa(stateCounts[k]))
	}
	if len(stateCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## Escalation Owners")
	for _, k := range sortedStringMapKeys(intStringMap(ownerCounts)) {
		lines = append(lines, "- "+k+": "+itoa(ownerCounts[k]))
	}
	if len(ownerCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## Items")
	for _, e := range entries {
		lines = append(lines, "- "+e.EntryID+": signoff="+e.SignoffID+" role="+e.Role+" surface="+e.SurfaceID+" status="+e.Status+" sla="+e.SLAStatus+" escalation_owner="+e.EscalationOwner)
		lines = append(lines, "  requested_at="+e.RequestedAt+" due_at="+e.DueAt+" linked_blockers="+e.LinkedBlockers+" summary="+e.Summary)
	}
	if len(entries) == 0 {
		lines = append(lines, "- none")
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewEscalationDashboard(pack UIReviewPack) string {
	type entry struct {
		EscalationID    string
		EscalationOwner string
		ItemType        string
		SourceID        string
		SurfaceID       string
		Status          string
		Priority        string
		CurrentOwner    string
		Summary         string
		DueAt           string
	}
	var entries []entry
	ownerCounts := map[string]map[string]int{}
	statusCounts := map[string]map[string]int{}
	for _, blocker := range pack.BlockerLog {
		b := blocker.normalized()
		e := entry{
			EscalationID:    "esc-" + b.BlockerID,
			EscalationOwner: fallback(b.EscalationOwner, "none"),
			ItemType:        "blocker",
			SourceID:        b.BlockerID,
			SurfaceID:       b.SurfaceID,
			Status:          b.Status,
			Priority:        b.Severity,
			CurrentOwner:    b.Owner,
			Summary:         b.Summary,
			DueAt:           fallback(b.FreezeUntil, "none"),
		}
		entries = append(entries, e)
	}
	for _, signoff := range pack.SignoffLog {
		s := signoff.normalized()
		if strings.ToLower(s.Status) == "approved" || strings.ToLower(s.Status) == "accepted" || strings.ToLower(s.Status) == "resolved" || strings.ToLower(s.Status) == "waived" {
			continue
		}
		e := entry{
			EscalationID:    "esc-" + s.SignoffID,
			EscalationOwner: fallback(s.EscalationOwner, "none"),
			ItemType:        "signoff",
			SourceID:        s.SignoffID,
			SurfaceID:       s.SurfaceID,
			Status:          s.Status,
			Priority:        s.SLAStatus,
			CurrentOwner:    s.Role,
			Summary:         fallback(s.Notes, "none"),
			DueAt:           fallback(s.DueAt, "none"),
		}
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].EscalationID < entries[j].EscalationID })
	for _, e := range entries {
		if _, ok := ownerCounts[e.EscalationOwner]; !ok {
			ownerCounts[e.EscalationOwner] = map[string]int{"blocker": 0, "signoff": 0, "total": 0}
		}
		if _, ok := statusCounts[e.Status]; !ok {
			statusCounts[e.Status] = map[string]int{"blocker": 0, "signoff": 0, "total": 0}
		}
		ownerCounts[e.EscalationOwner][e.ItemType]++
		ownerCounts[e.EscalationOwner]["total"]++
		statusCounts[e.Status][e.ItemType]++
		statusCounts[e.Status]["total"]++
	}
	var lines []string
	lines = append(lines, "# UI Review Escalation Dashboard", "", "- Issue: "+pack.IssueID+" "+pack.Title, "- Version: "+pack.Version, "- Items: "+itoa(len(entries)), "- Escalation owners: "+itoa(len(ownerCounts)), "", "## By Escalation Owner")
	for _, k := range sortedNestedMapKeys(ownerCounts) {
		c := ownerCounts[k]
		lines = append(lines, "- "+k+": blockers="+itoa(c["blocker"])+" signoffs="+itoa(c["signoff"])+" total="+itoa(c["total"]))
	}
	if len(ownerCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## By Status")
	for _, k := range sortedNestedMapKeys(statusCounts) {
		c := statusCounts[k]
		lines = append(lines, "- "+k+": blockers="+itoa(c["blocker"])+" signoffs="+itoa(c["signoff"])+" total="+itoa(c["total"]))
	}
	if len(statusCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## Escalations")
	for _, e := range entries {
		lines = append(lines, "- "+e.EscalationID+": owner="+e.EscalationOwner+" type="+e.ItemType+" source="+e.SourceID+" surface="+e.SurfaceID+" status="+e.Status+" priority="+e.Priority+" current_owner="+e.CurrentOwner)
		lines = append(lines, "  summary="+e.Summary+" due_at="+e.DueAt)
	}
	if len(entries) == 0 {
		lines = append(lines, "- none")
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewEscalationHandoffLedger(pack UIReviewPack) string {
	type entry struct {
		LedgerID    string
		EventID     string
		BlockerID   string
		SurfaceID   string
		Actor       string
		Status      string
		Timestamp   string
		HandoffFrom string
		HandoffTo   string
		Channel     string
		ArtifactRef string
		NextAction  string
	}
	blockers := map[string]ReviewBlocker{}
	for _, blocker := range pack.BlockerLog {
		blockers[blocker.BlockerID] = blocker
	}
	var entries []entry
	channelCounts := map[string]int{}
	statusCounts := map[string]int{}
	for _, event := range pack.BlockerTimeline {
		e := event.normalized()
		status := strings.ToLower(e.Status)
		if status != "escalated" && status != "handoff" && status != "reassigned" {
			continue
		}
		surfaceID := "none"
		if blocker, ok := blockers[e.BlockerID]; ok {
			surfaceID = blocker.SurfaceID
		}
		entry := entry{
			LedgerID:    "handoff-" + e.EventID,
			EventID:     e.EventID,
			BlockerID:   e.BlockerID,
			SurfaceID:   surfaceID,
			Actor:       e.Actor,
			Status:      e.Status,
			Timestamp:   e.Timestamp,
			HandoffFrom: fallback(e.HandoffFrom, "none"),
			HandoffTo:   fallback(e.HandoffTo, "none"),
			Channel:     fallback(e.Channel, "none"),
			ArtifactRef: fallback(e.ArtifactRef, "none"),
			NextAction:  fallback(e.NextAction, "none"),
		}
		entries = append(entries, entry)
		channelCounts[entry.Channel]++
		statusCounts[entry.Status]++
	}
	var lines []string
	lines = append(lines, "# UI Review Escalation Handoff Ledger", "", "- Issue: "+pack.IssueID+" "+pack.Title, "- Version: "+pack.Version, "- Handoffs: "+itoa(len(entries)), "- Channels: "+itoa(len(channelCounts)), "", "## By Status")
	for _, k := range sortedStringMapKeys(intStringMap(statusCounts)) {
		lines = append(lines, "- "+k+": "+itoa(statusCounts[k]))
	}
	if len(statusCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## By Channel")
	for _, k := range sortedStringMapKeys(intStringMap(channelCounts)) {
		lines = append(lines, "- "+k+": "+itoa(channelCounts[k]))
	}
	if len(channelCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## Entries")
	for _, e := range entries {
		lines = append(lines, "- "+e.LedgerID+": event="+e.EventID+" blocker="+e.BlockerID+" surface="+e.SurfaceID+" actor="+e.Actor+" status="+e.Status+" at="+e.Timestamp)
		lines = append(lines, "  from="+e.HandoffFrom+" to="+e.HandoffTo+" channel="+e.Channel+" artifact="+e.ArtifactRef+" next_action="+e.NextAction)
	}
	if len(entries) == 0 {
		lines = append(lines, "- none")
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewOwnerEscalationDigest(pack UIReviewPack) string {
	type entry struct {
		DigestID  string
		Owner     string
		ItemType  string
		SourceID  string
		SurfaceID string
		Status    string
		Summary   string
		Detail    string
	}
	var entries []entry
	for _, blocker := range pack.BlockerLog {
		b := blocker.normalized()
		entries = append(entries, entry{
			DigestID:  "digest-" + b.BlockerID,
			Owner:     b.EscalationOwner,
			ItemType:  "blocker",
			SourceID:  b.BlockerID,
			SurfaceID: b.SurfaceID,
			Status:    b.Status,
			Summary:   b.Summary,
			Detail:    "next_action=" + fallback(b.NextAction, "none"),
		})
	}
	for _, signoff := range pack.SignoffLog {
		s := signoff.normalized()
		status := strings.ToLower(s.Status)
		if status != "approved" && status != "accepted" && status != "resolved" && status != "waived" && strings.TrimSpace(s.ReminderOwner) != "" {
			entries = append(entries, entry{
				DigestID:  "digest-rem-" + s.SignoffID,
				Owner:     s.ReminderOwner,
				ItemType:  "reminder",
				SourceID:  s.SignoffID,
				SurfaceID: s.SurfaceID,
				Status:    s.Status,
				Summary:   fallback(s.Notes, "none"),
				Detail:    "next_reminder_at=" + fallback(s.NextReminderAt, "none"),
			})
		}
	}
	for _, blocker := range pack.BlockerLog {
		b := blocker.normalized()
		if b.FreezeException && strings.TrimSpace(b.FreezeOwner) != "" {
			entries = append(entries, entry{
				DigestID:  "digest-freeze-" + b.BlockerID,
				Owner:     b.FreezeOwner,
				ItemType:  "freeze",
				SourceID:  b.BlockerID,
				SurfaceID: b.SurfaceID,
				Status:    b.Status,
				Summary:   b.Summary,
				Detail:    "window=" + fallback(b.FreezeUntil, "none"),
			})
		}
	}
	for _, event := range pack.BlockerTimeline {
		e := event.normalized()
		status := strings.ToLower(e.Status)
		if (status == "escalated" || status == "handoff" || status == "reassigned") && strings.TrimSpace(e.HandoffTo) != "" {
			surfaceID := "none"
			for _, blocker := range pack.BlockerLog {
				if blocker.BlockerID == e.BlockerID {
					surfaceID = blocker.SurfaceID
					break
				}
			}
			entries = append(entries, entry{
				DigestID:  "digest-handoff-" + e.EventID,
				Owner:     e.HandoffTo,
				ItemType:  "handoff",
				SourceID:  e.EventID,
				SurfaceID: surfaceID,
				Status:    e.Status,
				Summary:   e.Summary,
				Detail:    "from=" + fallback(e.HandoffFrom, "none"),
			})
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].DigestID < entries[j].DigestID })
	ownerCounts := map[string]map[string]int{}
	for _, e := range entries {
		if _, ok := ownerCounts[e.Owner]; !ok {
			ownerCounts[e.Owner] = map[string]int{"blocker": 0, "signoff": 0, "reminder": 0, "freeze": 0, "handoff": 0, "total": 0}
		}
		ownerCounts[e.Owner][e.ItemType]++
		ownerCounts[e.Owner]["total"]++
	}
	var lines []string
	lines = append(lines, "# UI Review Owner Escalation Digest", "", "- Issue: "+pack.IssueID+" "+pack.Title, "- Version: "+pack.Version, "- Owners: "+itoa(len(ownerCounts)), "- Items: "+itoa(len(entries)), "", "## Owners")
	for _, k := range sortedNestedMapKeys(ownerCounts) {
		c := ownerCounts[k]
		lines = append(lines, "- "+k+": blockers="+itoa(c["blocker"])+" signoffs="+itoa(c["signoff"])+" reminders="+itoa(c["reminder"])+" freezes="+itoa(c["freeze"])+" handoffs="+itoa(c["handoff"])+" total="+itoa(c["total"]))
	}
	if len(ownerCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## Items")
	for _, e := range entries {
		lines = append(lines, "- "+e.DigestID+": owner="+e.Owner+" type="+e.ItemType+" source="+e.SourceID+" surface="+e.SurfaceID+" status="+e.Status)
		lines = append(lines, "  summary="+e.Summary+" detail="+e.Detail)
	}
	if len(entries) == 0 {
		lines = append(lines, "- none")
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewFreezeApprovalTrail(pack UIReviewPack) string {
	type entry struct {
		EntryID          string
		BlockerID        string
		SurfaceID        string
		Status           string
		FreezeOwner      string
		FreezeUntil      string
		FreezeApprovedBy string
		FreezeApprovedAt string
		Summary          string
		LatestEvent      string
		NextAction       string
	}
	var entries []entry
	approverCounts := map[string]int{}
	statusCounts := map[string]int{}
	for _, blocker := range pack.BlockerLog {
		b := blocker.normalized()
		if !b.FreezeException {
			continue
		}
		e := entry{
			EntryID:          "freeze-approval-" + b.BlockerID,
			BlockerID:        b.BlockerID,
			SurfaceID:        b.SurfaceID,
			Status:           b.Status,
			FreezeOwner:      fallback(b.FreezeOwner, b.Owner),
			FreezeUntil:      fallback(b.FreezeUntil, "none"),
			FreezeApprovedBy: fallback(b.FreezeApprovedBy, "none"),
			FreezeApprovedAt: fallback(b.FreezeApprovedAt, "none"),
			Summary:          fallback(b.FreezeReason, b.Summary),
			LatestEvent:      latestBlockerEventLabel(pack.BlockerTimeline, b.BlockerID),
			NextAction:       fallback(b.NextAction, "none"),
		}
		entries = append(entries, e)
		approverCounts[e.FreezeApprovedBy]++
		statusCounts[e.Status]++
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].EntryID < entries[j].EntryID })
	var lines []string
	lines = append(lines, "# UI Review Freeze Approval Trail", "", "- Issue: "+pack.IssueID+" "+pack.Title, "- Version: "+pack.Version, "- Approvals: "+itoa(len(entries)), "- Approvers: "+itoa(len(approverCounts)), "", "## By Approver")
	for _, k := range sortedStringMapKeys(intStringMap(approverCounts)) {
		lines = append(lines, "- "+k+": "+itoa(approverCounts[k]))
	}
	if len(approverCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## By Status")
	for _, k := range sortedStringMapKeys(intStringMap(statusCounts)) {
		lines = append(lines, "- "+k+": "+itoa(statusCounts[k]))
	}
	if len(statusCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## Entries")
	for _, e := range entries {
		lines = append(lines, "- "+e.EntryID+": blocker="+e.BlockerID+" surface="+e.SurfaceID+" status="+e.Status+" owner="+e.FreezeOwner+" approved_by="+e.FreezeApprovedBy+" approved_at="+e.FreezeApprovedAt+" window="+e.FreezeUntil)
		lines = append(lines, "  summary="+e.Summary+" latest_event="+e.LatestEvent+" next_action="+e.NextAction)
	}
	if len(entries) == 0 {
		lines = append(lines, "- none")
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewFreezeExceptionBoard(pack UIReviewPack) string {
	type entry struct {
		EntryID    string
		ItemType   string
		SourceID   string
		SurfaceID  string
		Owner      string
		Status     string
		Window     string
		Summary    string
		Evidence   string
		NextAction string
	}
	var entries []entry
	ownerCounts := map[string]map[string]int{}
	surfaceCounts := map[string]map[string]int{}
	for _, signoff := range pack.SignoffLog {
		s := signoff.normalized()
		status := strings.ToLower(s.Status)
		if status != "waived" && status != "deferred" {
			continue
		}
		entries = append(entries, entry{
			EntryID:    "freeze-" + s.SignoffID,
			ItemType:   "signoff",
			SourceID:   s.SignoffID,
			SurfaceID:  s.SurfaceID,
			Owner:      fallback(s.WaiverOwner, s.Role),
			Status:     s.Status,
			Window:     "none",
			Summary:    fallback(s.WaiverReason, fallback(s.Notes, "none")),
			Evidence:   joinCSVOrNone(s.EvidenceLinks),
			NextAction: fallback(s.Notes, fallback(s.WaiverReason, "none")),
		})
	}
	for _, blocker := range pack.BlockerLog {
		b := blocker.normalized()
		if !b.FreezeException {
			continue
		}
		entries = append(entries, entry{
			EntryID:    "freeze-" + b.BlockerID,
			ItemType:   "blocker",
			SourceID:   b.BlockerID,
			SurfaceID:  b.SurfaceID,
			Owner:      fallback(b.FreezeOwner, b.Owner),
			Status:     b.Status,
			Window:     fallback(b.FreezeUntil, "none"),
			Summary:    fallback(b.FreezeReason, b.Summary),
			Evidence:   latestBlockerEventLabel(pack.BlockerTimeline, b.BlockerID),
			NextAction: fallback(b.NextAction, "none"),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Owner != entries[j].Owner {
			return entries[i].Owner < entries[j].Owner
		}
		if entries[i].SurfaceID != entries[j].SurfaceID {
			return entries[i].SurfaceID < entries[j].SurfaceID
		}
		if entries[i].ItemType != entries[j].ItemType {
			return entries[i].ItemType < entries[j].ItemType
		}
		return entries[i].SourceID < entries[j].SourceID
	})
	for _, e := range entries {
		if _, ok := ownerCounts[e.Owner]; !ok {
			ownerCounts[e.Owner] = map[string]int{"blocker": 0, "signoff": 0, "total": 0}
		}
		if _, ok := surfaceCounts[e.SurfaceID]; !ok {
			surfaceCounts[e.SurfaceID] = map[string]int{"blocker": 0, "signoff": 0, "total": 0}
		}
		ownerCounts[e.Owner][e.ItemType]++
		ownerCounts[e.Owner]["total"]++
		surfaceCounts[e.SurfaceID][e.ItemType]++
		surfaceCounts[e.SurfaceID]["total"]++
	}
	var lines []string
	lines = append(lines, "# UI Review Freeze Exception Board", "", "- Issue: "+pack.IssueID+" "+pack.Title, "- Version: "+pack.Version, "- Exceptions: "+itoa(len(entries)), "- Owners: "+itoa(len(ownerCounts)), "", "## By Owner")
	for _, k := range sortedNestedMapKeys(ownerCounts) {
		c := ownerCounts[k]
		lines = append(lines, "- "+k+": blockers="+itoa(c["blocker"])+" signoffs="+itoa(c["signoff"])+" total="+itoa(c["total"]))
	}
	if len(ownerCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## By Surface")
	for _, k := range sortedNestedMapKeys(surfaceCounts) {
		c := surfaceCounts[k]
		lines = append(lines, "- "+k+": blockers="+itoa(c["blocker"])+" signoffs="+itoa(c["signoff"])+" total="+itoa(c["total"]))
	}
	if len(surfaceCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## Entries")
	for _, e := range entries {
		lines = append(lines, "- "+e.EntryID+": owner="+e.Owner+" type="+e.ItemType+" source="+e.SourceID+" surface="+e.SurfaceID+" status="+e.Status+" window="+e.Window)
		lines = append(lines, "  summary="+e.Summary+" evidence="+e.Evidence+" next_action="+e.NextAction)
	}
	if len(entries) == 0 {
		lines = append(lines, "- none")
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewExceptionMatrix(pack UIReviewPack) string {
	type entry struct {
		ExceptionID string
		Category    string
		SourceID    string
		SurfaceID   string
		Owner       string
		Status      string
		Severity    string
		Summary     string
		LatestEvent string
		NextAction  string
	}
	var entries []entry
	ownerCounts := map[string]map[string]int{}
	statusCounts := map[string]map[string]int{}
	surfaceCounts := map[string]map[string]int{}
	for _, signoff := range pack.SignoffLog {
		s := signoff.normalized()
		status := strings.ToLower(s.Status)
		if status != "waived" && status != "deferred" {
			continue
		}
		entries = append(entries, entry{
			ExceptionID: "exc-" + s.SignoffID,
			Category:    "signoff",
			SourceID:    s.SignoffID,
			SurfaceID:   s.SurfaceID,
			Owner:       fallback(s.WaiverOwner, s.Role),
			Status:      s.Status,
			Severity:    "none",
			Summary:     fallback(s.WaiverReason, fallback(s.Notes, "none")),
			LatestEvent: "none",
			NextAction:  fallback(s.Notes, fallback(s.WaiverReason, "none")),
		})
	}
	for _, blocker := range pack.BlockerLog {
		b := blocker.normalized()
		status := strings.ToLower(b.Status)
		if status == "resolved" || status == "closed" {
			continue
		}
		entries = append(entries, entry{
			ExceptionID: "exc-" + b.BlockerID,
			Category:    "blocker",
			SourceID:    b.BlockerID,
			SurfaceID:   b.SurfaceID,
			Owner:       b.Owner,
			Status:      b.Status,
			Severity:    b.Severity,
			Summary:     b.Summary,
			LatestEvent: latestBlockerEventLabel(pack.BlockerTimeline, b.BlockerID),
			NextAction:  fallback(b.NextAction, "none"),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Owner != entries[j].Owner {
			return entries[i].Owner < entries[j].Owner
		}
		if entries[i].SurfaceID != entries[j].SurfaceID {
			return entries[i].SurfaceID < entries[j].SurfaceID
		}
		if entries[i].Category != entries[j].Category {
			return entries[i].Category < entries[j].Category
		}
		return entries[i].SourceID < entries[j].SourceID
	})
	for _, e := range entries {
		if _, ok := ownerCounts[e.Owner]; !ok {
			ownerCounts[e.Owner] = map[string]int{"blocker": 0, "signoff": 0, "total": 0}
		}
		if _, ok := statusCounts[e.Status]; !ok {
			statusCounts[e.Status] = map[string]int{"blocker": 0, "signoff": 0, "total": 0}
		}
		if _, ok := surfaceCounts[e.SurfaceID]; !ok {
			surfaceCounts[e.SurfaceID] = map[string]int{"blocker": 0, "signoff": 0, "total": 0}
		}
		ownerCounts[e.Owner][e.Category]++
		ownerCounts[e.Owner]["total"]++
		statusCounts[e.Status][e.Category]++
		statusCounts[e.Status]["total"]++
		surfaceCounts[e.SurfaceID][e.Category]++
		surfaceCounts[e.SurfaceID]["total"]++
	}
	var lines []string
	lines = append(lines, "# UI Review Exception Matrix", "", "- Issue: "+pack.IssueID+" "+pack.Title, "- Version: "+pack.Version, "- Exceptions: "+itoa(len(entries)), "- Owners: "+itoa(len(ownerCounts)), "- Surfaces: "+itoa(len(surfaceCounts)), "", "## By Owner")
	for _, k := range sortedNestedMapKeys(ownerCounts) {
		c := ownerCounts[k]
		lines = append(lines, "- "+k+": blockers="+itoa(c["blocker"])+" signoffs="+itoa(c["signoff"])+" total="+itoa(c["total"]))
	}
	if len(ownerCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## By Status")
	for _, k := range sortedNestedMapKeys(statusCounts) {
		c := statusCounts[k]
		lines = append(lines, "- "+k+": blockers="+itoa(c["blocker"])+" signoffs="+itoa(c["signoff"])+" total="+itoa(c["total"]))
	}
	if len(statusCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## By Surface")
	for _, k := range sortedNestedMapKeys(surfaceCounts) {
		c := surfaceCounts[k]
		lines = append(lines, "- "+k+": blockers="+itoa(c["blocker"])+" signoffs="+itoa(c["signoff"])+" total="+itoa(c["total"]))
	}
	if len(surfaceCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## Entries")
	for _, e := range entries {
		lines = append(lines, "- "+e.ExceptionID+": owner="+e.Owner+" type="+e.Category+" source="+e.SourceID+" surface="+e.SurfaceID+" status="+e.Status+" severity="+e.Severity)
		lines = append(lines, "  summary="+e.Summary+" latest_event="+e.LatestEvent+" next_action="+e.NextAction)
	}
	if len(entries) == 0 {
		lines = append(lines, "- none")
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewOwnerReviewQueue(pack UIReviewPack) string {
	type entry struct {
		QueueID    string
		Owner      string
		ItemType   string
		SourceID   string
		SurfaceID  string
		Status     string
		Summary    string
		NextAction string
	}
	checklistReadyStatuses := map[string]bool{"ready": true, "approved": true, "accepted": true, "resolved": true, "done": true}
	decisionReadyStatuses := map[string]bool{"accepted": true, "approved": true, "resolved": true, "waived": true}
	signoffReadyStatuses := map[string]bool{"approved": true, "accepted": true, "resolved": true}
	blockerDoneStatuses := map[string]bool{"resolved": true, "closed": true}
	var entries []entry
	for _, item := range pack.ReviewerChecklist {
		if checklistReadyStatuses[strings.ToLower(item.Status)] {
			continue
		}
		entries = append(entries, entry{
			QueueID:    "queue-" + item.ItemID,
			Owner:      item.Owner,
			ItemType:   "checklist",
			SourceID:   item.ItemID,
			SurfaceID:  item.SurfaceID,
			Status:     item.Status,
			Summary:    item.Prompt,
			NextAction: fallback(item.Notes, joinCSVOrNone(item.EvidenceLinks)),
		})
	}
	for _, decision := range pack.DecisionLog {
		if decisionReadyStatuses[strings.ToLower(decision.Status)] {
			continue
		}
		entries = append(entries, entry{
			QueueID:    "queue-" + decision.DecisionID,
			Owner:      decision.Owner,
			ItemType:   "decision",
			SourceID:   decision.DecisionID,
			SurfaceID:  decision.SurfaceID,
			Status:     decision.Status,
			Summary:    decision.Summary,
			NextAction: fallback(decision.FollowUp, decision.Rationale),
		})
	}
	for _, signoff := range pack.SignoffLog {
		s := signoff.normalized()
		if signoffReadyStatuses[strings.ToLower(s.Status)] {
			continue
		}
		entries = append(entries, entry{
			QueueID:    "queue-" + s.SignoffID,
			Owner:      fallback(s.WaiverOwner, s.Role),
			ItemType:   "signoff",
			SourceID:   s.SignoffID,
			SurfaceID:  s.SurfaceID,
			Status:     s.Status,
			Summary:    fallback(s.Notes, fallback(s.WaiverReason, s.Role)),
			NextAction: fallback(s.WaiverReason, fallback(s.Notes, fallback(s.DueAt, joinCSVOrNone(s.EvidenceLinks)))),
		})
	}
	for _, blocker := range pack.BlockerLog {
		b := blocker.normalized()
		if blockerDoneStatuses[strings.ToLower(b.Status)] {
			continue
		}
		entries = append(entries, entry{
			QueueID:    "queue-" + b.BlockerID,
			Owner:      b.Owner,
			ItemType:   "blocker",
			SourceID:   b.BlockerID,
			SurfaceID:  b.SurfaceID,
			Status:     b.Status,
			Summary:    b.Summary,
			NextAction: fallback(b.NextAction, "none"),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Owner != entries[j].Owner {
			return entries[i].Owner < entries[j].Owner
		}
		if entries[i].ItemType != entries[j].ItemType {
			return entries[i].ItemType < entries[j].ItemType
		}
		if entries[i].SurfaceID != entries[j].SurfaceID {
			return entries[i].SurfaceID < entries[j].SurfaceID
		}
		return entries[i].SourceID < entries[j].SourceID
	})
	ownerCounts := map[string]map[string]int{}
	for _, e := range entries {
		if _, ok := ownerCounts[e.Owner]; !ok {
			ownerCounts[e.Owner] = map[string]int{"blocker": 0, "checklist": 0, "decision": 0, "signoff": 0, "total": 0}
		}
		ownerCounts[e.Owner][e.ItemType]++
		ownerCounts[e.Owner]["total"]++
	}
	var lines []string
	lines = append(lines, "# UI Review Owner Review Queue", "", "- Issue: "+pack.IssueID+" "+pack.Title, "- Version: "+pack.Version, "- Owners: "+itoa(len(ownerCounts)), "- Queue items: "+itoa(len(entries)), "", "## Owners")
	for _, k := range sortedNestedMapKeys(ownerCounts) {
		c := ownerCounts[k]
		lines = append(lines, "- "+k+": blockers="+itoa(c["blocker"])+" checklist="+itoa(c["checklist"])+" decisions="+itoa(c["decision"])+" signoffs="+itoa(c["signoff"])+" total="+itoa(c["total"]))
	}
	if len(ownerCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## Items")
	for _, e := range entries {
		lines = append(lines, "- "+e.QueueID+": owner="+e.Owner+" type="+e.ItemType+" source="+e.SourceID+" surface="+e.SurfaceID+" status="+e.Status)
		lines = append(lines, "  summary="+e.Summary+" next_action="+e.NextAction)
	}
	if len(entries) == 0 {
		lines = append(lines, "- none")
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewChecklistTraceabilityBoard(pack UIReviewPack) string {
	type entry struct {
		EntryID           string
		ItemID            string
		SurfaceID         string
		Owner             string
		Status            string
		LinkedAssignments string
		LinkedRoles       string
		LinkedDecisions   string
		Evidence          string
		Summary           string
	}
	assignmentsByItem := map[string][]ReviewRoleAssignment{}
	for _, assignment := range pack.RoleMatrix {
		for _, itemID := range assignment.ChecklistItemIDs {
			assignmentsByItem[itemID] = append(assignmentsByItem[itemID], assignment)
		}
	}
	var entries []entry
	ownerCounts := map[string]int{}
	statusCounts := map[string]int{}
	for _, item := range pack.ReviewerChecklist {
		assignments := assignmentsByItem[item.ItemID]
		var assignmentIDs []string
		var roles []string
		decisionSet := map[string]bool{}
		for _, assignment := range assignments {
			assignmentIDs = append(assignmentIDs, assignment.AssignmentID)
			roles = append(roles, assignment.Role)
			for _, decisionID := range assignment.DecisionIDs {
				decisionSet[decisionID] = true
			}
		}
		var decisionIDs []string
		for decisionID := range decisionSet {
			decisionIDs = append(decisionIDs, decisionID)
		}
		sort.Strings(decisionIDs)
		e := entry{
			EntryID:           "trace-" + item.ItemID,
			ItemID:            item.ItemID,
			SurfaceID:         item.SurfaceID,
			Owner:             item.Owner,
			Status:            item.Status,
			LinkedAssignments: joinCSVOrNone(assignmentIDs),
			LinkedRoles:       joinCSVOrNone(roles),
			LinkedDecisions:   joinCSVOrNone(decisionIDs),
			Evidence:          joinCSVOrNone(item.EvidenceLinks),
			Summary:           fallback(item.Notes, item.Prompt),
		}
		entries = append(entries, e)
		ownerCounts[e.Owner]++
		statusCounts[e.Status]++
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Status != entries[j].Status {
			return entries[i].Status < entries[j].Status
		}
		if entries[i].Owner != entries[j].Owner {
			return entries[i].Owner < entries[j].Owner
		}
		return entries[i].ItemID < entries[j].ItemID
	})
	var lines []string
	lines = append(lines, "# UI Review Checklist Traceability Board", "", "- Issue: "+pack.IssueID+" "+pack.Title, "- Version: "+pack.Version, "- Checklist items: "+itoa(len(entries)), "- Owners: "+itoa(len(ownerCounts)), "", "## By Owner")
	for _, k := range sortedStringMapKeys(intStringMap(ownerCounts)) {
		lines = append(lines, "- "+k+": "+itoa(ownerCounts[k]))
	}
	if len(ownerCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## By Status")
	for _, k := range sortedStringMapKeys(intStringMap(statusCounts)) {
		lines = append(lines, "- "+k+": "+itoa(statusCounts[k]))
	}
	if len(statusCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## Entries")
	for _, e := range entries {
		lines = append(lines, "- "+e.EntryID+": item="+e.ItemID+" surface="+e.SurfaceID+" owner="+e.Owner+" status="+e.Status+" linked_roles="+e.LinkedRoles)
		lines = append(lines, "  linked_assignments="+e.LinkedAssignments+" linked_decisions="+e.LinkedDecisions+" evidence="+e.Evidence+" summary="+e.Summary)
	}
	if len(entries) == 0 {
		lines = append(lines, "- none")
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewDecisionFollowupTracker(pack UIReviewPack) string {
	type entry struct {
		EntryID           string
		DecisionID        string
		SurfaceID         string
		Owner             string
		Status            string
		LinkedRoles       string
		LinkedAssignments string
		LinkedChecklists  string
		FollowUp          string
		Summary           string
	}
	assignmentsByDecision := map[string][]ReviewRoleAssignment{}
	for _, assignment := range pack.RoleMatrix {
		for _, decisionID := range assignment.DecisionIDs {
			assignmentsByDecision[decisionID] = append(assignmentsByDecision[decisionID], assignment)
		}
	}
	var entries []entry
	ownerCounts := map[string]int{}
	statusCounts := map[string]int{}
	for _, decision := range pack.DecisionLog {
		assignments := assignmentsByDecision[decision.DecisionID]
		var roles []string
		var assignmentIDs []string
		checklistSet := map[string]bool{}
		for _, assignment := range assignments {
			roles = append(roles, assignment.Role)
			assignmentIDs = append(assignmentIDs, assignment.AssignmentID)
			for _, checklistID := range assignment.ChecklistItemIDs {
				checklistSet[checklistID] = true
			}
		}
		var checklistIDs []string
		for checklistID := range checklistSet {
			checklistIDs = append(checklistIDs, checklistID)
		}
		sort.Strings(checklistIDs)
		e := entry{
			EntryID:           "follow-" + decision.DecisionID,
			DecisionID:        decision.DecisionID,
			SurfaceID:         decision.SurfaceID,
			Owner:             decision.Owner,
			Status:            decision.Status,
			LinkedRoles:       joinCSVOrNone(roles),
			LinkedAssignments: joinCSVOrNone(assignmentIDs),
			LinkedChecklists:  joinCSVOrNone(checklistIDs),
			FollowUp:          fallback(decision.FollowUp, "none"),
			Summary:           decision.Summary,
		}
		entries = append(entries, e)
		ownerCounts[e.Owner]++
		statusCounts[e.Status]++
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Status != entries[j].Status {
			return entries[i].Status < entries[j].Status
		}
		if entries[i].Owner != entries[j].Owner {
			return entries[i].Owner < entries[j].Owner
		}
		return entries[i].DecisionID < entries[j].DecisionID
	})
	var lines []string
	lines = append(lines, "# UI Review Decision Follow-up Tracker", "", "- Issue: "+pack.IssueID+" "+pack.Title, "- Version: "+pack.Version, "- Decisions: "+itoa(len(entries)), "- Owners: "+itoa(len(ownerCounts)), "", "## By Owner")
	for _, k := range sortedStringMapKeys(intStringMap(ownerCounts)) {
		lines = append(lines, "- "+k+": "+itoa(ownerCounts[k]))
	}
	if len(ownerCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## By Status")
	for _, k := range sortedStringMapKeys(intStringMap(statusCounts)) {
		lines = append(lines, "- "+k+": "+itoa(statusCounts[k]))
	}
	if len(statusCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## Entries")
	for _, e := range entries {
		lines = append(lines, "- "+e.EntryID+": decision="+e.DecisionID+" surface="+e.SurfaceID+" owner="+e.Owner+" status="+e.Status+" linked_roles="+e.LinkedRoles)
		lines = append(lines, "  linked_assignments="+e.LinkedAssignments+" linked_checklists="+e.LinkedChecklists+" follow_up="+e.FollowUp+" summary="+e.Summary)
	}
	if len(entries) == 0 {
		lines = append(lines, "- none")
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewRoleCoverageBoard(pack UIReviewPack) string {
	type entry struct {
		EntryID             string
		AssignmentID        string
		SurfaceID           string
		Role                string
		Status              string
		ResponsibilityCount int
		ChecklistCount      int
		DecisionCount       int
		SignoffID           string
		SignoffStatus       string
		Summary             string
	}
	signoffsByAssignment := map[string]ReviewSignoff{}
	for _, signoff := range pack.SignoffLog {
		signoffsByAssignment[signoff.AssignmentID] = signoff
	}
	var entries []entry
	surfaceCounts := map[string]int{}
	statusCounts := map[string]int{}
	for _, assignment := range pack.RoleMatrix {
		signoff, ok := signoffsByAssignment[assignment.AssignmentID]
		signoffID := "none"
		signoffStatus := "none"
		if ok {
			signoffID = signoff.SignoffID
			signoffStatus = signoff.Status
		}
		e := entry{
			EntryID:             "cover-" + assignment.AssignmentID,
			AssignmentID:        assignment.AssignmentID,
			SurfaceID:           assignment.SurfaceID,
			Role:                assignment.Role,
			Status:              assignment.Status,
			ResponsibilityCount: len(assignment.Responsibilities),
			ChecklistCount:      len(assignment.ChecklistItemIDs),
			DecisionCount:       len(assignment.DecisionIDs),
			SignoffID:           signoffID,
			SignoffStatus:       signoffStatus,
			Summary:             joinCSVOrNone(assignment.Responsibilities),
		}
		entries = append(entries, e)
		surfaceCounts[e.SurfaceID]++
		statusCounts[e.Status]++
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].SurfaceID != entries[j].SurfaceID {
			return entries[i].SurfaceID < entries[j].SurfaceID
		}
		if entries[i].Status != entries[j].Status {
			return entries[i].Status < entries[j].Status
		}
		return entries[i].AssignmentID < entries[j].AssignmentID
	})
	var lines []string
	lines = append(lines, "# UI Review Role Coverage Board", "", "- Issue: "+pack.IssueID+" "+pack.Title, "- Version: "+pack.Version, "- Assignments: "+itoa(len(entries)), "- Surfaces: "+itoa(len(surfaceCounts)), "", "## By Surface")
	for _, k := range sortedStringMapKeys(intStringMap(surfaceCounts)) {
		lines = append(lines, "- "+k+": "+itoa(surfaceCounts[k]))
	}
	if len(surfaceCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## By Status")
	for _, k := range sortedStringMapKeys(intStringMap(statusCounts)) {
		lines = append(lines, "- "+k+": "+itoa(statusCounts[k]))
	}
	if len(statusCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## Entries")
	for _, e := range entries {
		lines = append(lines, "- "+e.EntryID+": assignment="+e.AssignmentID+" surface="+e.SurfaceID+" role="+e.Role+" status="+e.Status+" responsibilities="+itoa(e.ResponsibilityCount)+" checklist="+itoa(e.ChecklistCount)+" decisions="+itoa(e.DecisionCount))
		lines = append(lines, "  signoff="+e.SignoffID+" signoff_status="+e.SignoffStatus+" summary="+e.Summary)
	}
	if len(entries) == 0 {
		lines = append(lines, "- none")
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewReminderCadenceBoard(pack UIReviewPack) string {
	type entry struct {
		EntryID        string
		SignoffID      string
		Role           string
		SurfaceID      string
		Cadence        string
		Status         string
		Owner          string
		SLAStatus      string
		LastReminderAt string
		NextReminderAt string
		DueAt          string
		Summary        string
	}
	unresolvedStatuses := map[string]bool{"approved": true, "accepted": true, "resolved": true, "waived": true, "deferred": true}
	var entries []entry
	cadenceCounts := map[string]int{}
	statusCounts := map[string]int{}
	for _, signoff := range pack.SignoffLog {
		s := signoff.normalized()
		if !s.Required || unresolvedStatuses[strings.ToLower(s.Status)] {
			continue
		}
		e := entry{
			EntryID:        "cad-rem-" + s.SignoffID,
			SignoffID:      s.SignoffID,
			Role:           s.Role,
			SurfaceID:      s.SurfaceID,
			Cadence:        fallback(s.ReminderCadence, "none"),
			Status:         fallback(s.ReminderStatus, "scheduled"),
			Owner:          fallback(s.ReminderOwner, "none"),
			SLAStatus:      s.SLAStatus,
			LastReminderAt: fallback(s.LastReminderAt, "none"),
			NextReminderAt: fallback(s.NextReminderAt, "none"),
			DueAt:          fallback(s.DueAt, "none"),
			Summary:        fallback(s.Notes, fallback(s.WaiverReason, s.Role)),
		}
		entries = append(entries, e)
		cadenceCounts[e.Cadence]++
		statusCounts[e.Status]++
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Cadence != entries[j].Cadence {
			return entries[i].Cadence < entries[j].Cadence
		}
		if entries[i].Status != entries[j].Status {
			return entries[i].Status < entries[j].Status
		}
		return entries[i].SignoffID < entries[j].SignoffID
	})
	var lines []string
	lines = append(lines, "# UI Review Reminder Cadence Board", "", "- Issue: "+pack.IssueID+" "+pack.Title, "- Version: "+pack.Version, "- Items: "+itoa(len(entries)), "- Cadences: "+itoa(len(cadenceCounts)), "", "## By Cadence")
	for _, k := range sortedStringMapKeys(intStringMap(cadenceCounts)) {
		lines = append(lines, "- "+k+": "+itoa(cadenceCounts[k]))
	}
	if len(cadenceCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## By Status")
	for _, k := range sortedStringMapKeys(intStringMap(statusCounts)) {
		lines = append(lines, "- "+k+": "+itoa(statusCounts[k]))
	}
	if len(statusCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## Items")
	for _, e := range entries {
		lines = append(lines, "- "+e.EntryID+": signoff="+e.SignoffID+" role="+e.Role+" surface="+e.SurfaceID+" cadence="+e.Cadence+" status="+e.Status+" owner="+e.Owner)
		lines = append(lines, "  sla="+e.SLAStatus+" last_reminder_at="+e.LastReminderAt+" next_reminder_at="+e.NextReminderAt+" due_at="+e.DueAt+" summary="+e.Summary)
	}
	if len(entries) == 0 {
		lines = append(lines, "- none")
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewHandoffAckLedger(pack UIReviewPack) string {
	type entry struct {
		EntryID     string
		EventID     string
		BlockerID   string
		SurfaceID   string
		Actor       string
		Status      string
		HandoffTo   string
		AckOwner    string
		AckStatus   string
		AckAt       string
		Channel     string
		ArtifactRef string
		Summary     string
	}
	blockers := map[string]ReviewBlocker{}
	for _, blocker := range pack.BlockerLog {
		blockers[blocker.BlockerID] = blocker
	}
	var entries []entry
	ownerCounts := map[string]int{}
	statusCounts := map[string]int{}
	for _, event := range pack.BlockerTimeline {
		e := event.normalized()
		status := strings.ToLower(e.Status)
		if status != "escalated" && status != "handoff" && status != "reassigned" && strings.TrimSpace(e.HandoffTo) == "" {
			continue
		}
		blocker, ok := blockers[e.BlockerID]
		fallbackOwner := "none"
		surfaceID := "none"
		if ok {
			surfaceID = blocker.SurfaceID
			fallbackOwner = fallback(blocker.EscalationOwner, "none")
		}
		handoffTo := fallback(e.HandoffTo, fallbackOwner)
		ackOwner := fallback(e.AckOwner, handoffTo)
		entry := entry{
			EntryID:     "ack-" + e.EventID,
			EventID:     e.EventID,
			BlockerID:   e.BlockerID,
			SurfaceID:   surfaceID,
			Actor:       e.Actor,
			Status:      e.Status,
			HandoffTo:   handoffTo,
			AckOwner:    ackOwner,
			AckStatus:   fallback(e.AckStatus, "pending"),
			AckAt:       fallback(e.AckAt, "none"),
			Channel:     fallback(e.Channel, "none"),
			ArtifactRef: fallback(e.ArtifactRef, "none"),
			Summary:     e.Summary,
		}
		entries = append(entries, entry)
		ownerCounts[entry.AckOwner]++
		statusCounts[entry.AckStatus]++
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].AckStatus != entries[j].AckStatus {
			return entries[i].AckStatus < entries[j].AckStatus
		}
		if entries[i].AckOwner != entries[j].AckOwner {
			return entries[i].AckOwner < entries[j].AckOwner
		}
		return entries[i].EventID < entries[j].EventID
	})
	var lines []string
	lines = append(lines, "# UI Review Handoff Ack Ledger", "", "- Issue: "+pack.IssueID+" "+pack.Title, "- Version: "+pack.Version, "- Ack items: "+itoa(len(entries)), "- Ack owners: "+itoa(len(ownerCounts)), "", "## By Ack Owner")
	for _, k := range sortedStringMapKeys(intStringMap(ownerCounts)) {
		lines = append(lines, "- "+k+": "+itoa(ownerCounts[k]))
	}
	if len(ownerCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## By Ack Status")
	for _, k := range sortedStringMapKeys(intStringMap(statusCounts)) {
		lines = append(lines, "- "+k+": "+itoa(statusCounts[k]))
	}
	if len(statusCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## Entries")
	for _, e := range entries {
		lines = append(lines, "- "+e.EntryID+": event="+e.EventID+" blocker="+e.BlockerID+" surface="+e.SurfaceID+" handoff_to="+e.HandoffTo+" ack_owner="+e.AckOwner+" ack_status="+e.AckStatus+" ack_at="+e.AckAt)
		lines = append(lines, "  actor="+e.Actor+" status="+e.Status+" channel="+e.Channel+" artifact="+e.ArtifactRef+" summary="+e.Summary)
	}
	if len(entries) == 0 {
		lines = append(lines, "- none")
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewFreezeRenewalTracker(pack UIReviewPack) string {
	type entry struct {
		EntryID          string
		BlockerID        string
		SurfaceID        string
		Status           string
		FreezeOwner      string
		FreezeUntil      string
		RenewalOwner     string
		RenewalBy        string
		RenewalStatus    string
		FreezeApprovedBy string
		Summary          string
		NextAction       string
	}
	var entries []entry
	ownerCounts := map[string]int{}
	statusCounts := map[string]int{}
	for _, blocker := range pack.BlockerLog {
		b := blocker.normalized()
		if !b.FreezeException {
			continue
		}
		e := entry{
			EntryID:          "renew-" + b.BlockerID,
			BlockerID:        b.BlockerID,
			SurfaceID:        b.SurfaceID,
			Status:           b.Status,
			FreezeOwner:      fallback(b.FreezeOwner, b.Owner),
			FreezeUntil:      fallback(b.FreezeUntil, "none"),
			RenewalOwner:     fallback(b.FreezeRenewalOwner, "none"),
			RenewalBy:        fallback(b.FreezeRenewalBy, "none"),
			RenewalStatus:    fallback(b.FreezeRenewalStatus, "not-needed"),
			FreezeApprovedBy: fallback(b.FreezeApprovedBy, "none"),
			Summary:          fallback(b.FreezeReason, b.Summary),
			NextAction:       fallback(b.NextAction, "none"),
		}
		entries = append(entries, e)
		ownerCounts[e.RenewalOwner]++
		statusCounts[e.RenewalStatus]++
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].RenewalBy != entries[j].RenewalBy {
			return entries[i].RenewalBy < entries[j].RenewalBy
		}
		if entries[i].RenewalOwner != entries[j].RenewalOwner {
			return entries[i].RenewalOwner < entries[j].RenewalOwner
		}
		return entries[i].BlockerID < entries[j].BlockerID
	})
	var lines []string
	lines = append(lines, "# UI Review Freeze Renewal Tracker", "", "- Issue: "+pack.IssueID+" "+pack.Title, "- Version: "+pack.Version, "- Renewal items: "+itoa(len(entries)), "- Renewal owners: "+itoa(len(ownerCounts)), "", "## By Renewal Owner")
	for _, k := range sortedStringMapKeys(intStringMap(ownerCounts)) {
		lines = append(lines, "- "+k+": "+itoa(ownerCounts[k]))
	}
	if len(ownerCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## By Renewal Status")
	for _, k := range sortedStringMapKeys(intStringMap(statusCounts)) {
		lines = append(lines, "- "+k+": "+itoa(statusCounts[k]))
	}
	if len(statusCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## Entries")
	for _, e := range entries {
		lines = append(lines, "- "+e.EntryID+": blocker="+e.BlockerID+" surface="+e.SurfaceID+" status="+e.Status+" renewal_owner="+e.RenewalOwner+" renewal_by="+e.RenewalBy+" renewal_status="+e.RenewalStatus)
		lines = append(lines, "  freeze_owner="+e.FreezeOwner+" freeze_until="+e.FreezeUntil+" approved_by="+e.FreezeApprovedBy+" summary="+e.Summary+" next_action="+e.NextAction)
	}
	if len(entries) == 0 {
		lines = append(lines, "- none")
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewExceptionLog(pack UIReviewPack) string {
	type entry struct {
		ExceptionID string
		Category    string
		SourceID    string
		SurfaceID   string
		Owner       string
		Status      string
		Severity    string
		Summary     string
		Evidence    string
		LatestEvent string
		NextAction  string
	}
	var entries []entry
	for _, signoff := range pack.SignoffLog {
		s := signoff.normalized()
		status := strings.ToLower(s.Status)
		if status != "waived" && status != "deferred" {
			continue
		}
		entries = append(entries, entry{
			ExceptionID: "exc-" + s.SignoffID,
			Category:    "signoff",
			SourceID:    s.SignoffID,
			SurfaceID:   s.SurfaceID,
			Owner:       fallback(s.WaiverOwner, s.Role),
			Status:      s.Status,
			Severity:    "none",
			Summary:     fallback(s.WaiverReason, fallback(s.Notes, "none")),
			Evidence:    joinCSVOrNone(s.EvidenceLinks),
			LatestEvent: "none",
			NextAction:  fallback(s.Notes, fallback(s.WaiverReason, "none")),
		})
	}
	for _, blocker := range pack.BlockerLog {
		b := blocker.normalized()
		status := strings.ToLower(b.Status)
		if status == "resolved" || status == "closed" {
			continue
		}
		entries = append(entries, entry{
			ExceptionID: "exc-" + b.BlockerID,
			Category:    "blocker",
			SourceID:    b.BlockerID,
			SurfaceID:   b.SurfaceID,
			Owner:       b.Owner,
			Status:      b.Status,
			Severity:    b.Severity,
			Summary:     b.Summary,
			Evidence:    fallback(b.EscalationOwner, "none"),
			LatestEvent: latestBlockerEventLabel(pack.BlockerTimeline, b.BlockerID),
			NextAction:  fallback(b.NextAction, "none"),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Owner != entries[j].Owner {
			return entries[i].Owner < entries[j].Owner
		}
		if entries[i].SurfaceID != entries[j].SurfaceID {
			return entries[i].SurfaceID < entries[j].SurfaceID
		}
		if entries[i].Category != entries[j].Category {
			return entries[i].Category < entries[j].Category
		}
		return entries[i].SourceID < entries[j].SourceID
	})
	var lines []string
	lines = append(lines, "# UI Review Exception Log", "", "- Issue: "+pack.IssueID+" "+pack.Title, "- Version: "+pack.Version, "- Exceptions: "+itoa(len(entries)), "", "## Exceptions")
	for _, e := range entries {
		lines = append(lines, "- "+e.ExceptionID+": type="+e.Category+" source="+e.SourceID+" surface="+e.SurfaceID+" owner="+e.Owner+" status="+e.Status+" severity="+e.Severity)
		lines = append(lines, "  summary="+e.Summary+" evidence="+e.Evidence+" latest_event="+e.LatestEvent+" next_action="+e.NextAction)
	}
	if len(entries) == 0 {
		lines = append(lines, "- none")
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIReviewBlockerTimelineSummary(pack UIReviewPack) string {
	statusCounts := map[string]int{}
	actorCounts := map[string]int{}
	timelineIndex := map[string][]ReviewBlockerEvent{}
	for _, event := range pack.BlockerTimeline {
		e := event.normalized()
		statusCounts[e.Status]++
		actorCounts[e.Actor]++
		timelineIndex[e.BlockerID] = append(timelineIndex[e.BlockerID], e)
	}
	blockerIDs := map[string]bool{}
	for _, blocker := range pack.BlockerLog {
		blockerIDs[blocker.BlockerID] = true
	}
	var orphanIDs []string
	for blockerID := range timelineIndex {
		if !blockerIDs[blockerID] {
			orphanIDs = append(orphanIDs, blockerID)
		}
	}
	sort.Strings(orphanIDs)
	var lines []string
	lines = append(lines, "# UI Review Blocker Timeline Summary", "", "- Issue: "+pack.IssueID+" "+pack.Title, "- Version: "+pack.Version, "- Events: "+itoa(len(pack.BlockerTimeline)), "- Blockers with timeline: "+itoa(len(timelineIndex)), "- Orphan timeline blockers: "+joinCSVOrNone(orphanIDs), "", "## Events by Status")
	for _, k := range sortedStringMapKeys(intStringMap(statusCounts)) {
		lines = append(lines, "- "+k+": "+itoa(statusCounts[k]))
	}
	if len(statusCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## Events by Actor")
	for _, k := range sortedStringMapKeys(intStringMap(actorCounts)) {
		lines = append(lines, "- "+k+": "+itoa(actorCounts[k]))
	}
	if len(actorCounts) == 0 {
		lines = append(lines, "- none")
	}
	lines = append(lines, "", "## Latest Blocker Events")
	for _, blocker := range pack.BlockerLog {
		events := timelineIndex[blocker.BlockerID]
		if len(events) == 0 {
			lines = append(lines, "- "+blocker.BlockerID+": latest=none")
			continue
		}
		latest := latestBlockerEvent(events)
		lines = append(lines, "- "+blocker.BlockerID+": latest="+latest.EventID+" actor="+latest.Actor+" status="+latest.Status+" at="+latest.Timestamp)
	}
	if len(pack.BlockerLog) == 0 {
		lines = append(lines, "- none")
	}
	return strings.Join(lines, "\n") + "\n"
}

func BuildBIG4204ReviewPack() UIReviewPack {
	return UIReviewPack{
		IssueID:                   "BIG-4204",
		Title:                     "UI评审包输出",
		Version:                   "v4.0-design-sprint",
		RequiresReviewerChecklist: true,
		RequiresDecisionLog:       true,
		RequiresRoleMatrix:        true,
		RequiresSignoffLog:        true,
		RequiresBlockerLog:        true,
		RequiresBlockerTimeline:   true,
		Objectives: []ReviewObjective{
			{ObjectiveID: "obj-overview-decision", Title: "Validate the executive overview narrative and drill-down posture", Persona: "VP Eng", Outcome: "Leadership can confirm the overview page balances KPI density with investigation entry points.", SuccessSignal: "Reviewers agree the overview supports release, risk, and queue drill-down without extra walkthroughs.", Priority: "P0", Dependencies: []string{"BIG-4203", "OPE-132"}},
			{ObjectiveID: "obj-queue-governance", Title: "Confirm queue control actions and approval posture", Persona: "Platform Admin", Outcome: "Operators can assess batch approvals, audit visibility, and denial paths from one frame.", SuccessSignal: "The queue frame clearly shows allowed actions, denied roles, and audit expectations.", Priority: "P0", Dependencies: []string{"BIG-4203", "OPE-131", "OPE-132"}},
			{ObjectiveID: "obj-run-detail-investigation", Title: "Validate replay and audit investigation flow", Persona: "Eng Lead", Outcome: "Run detail reviewers can trace evidence, replay context, and escalation actions in one surface.", SuccessSignal: "The run-detail frame makes failure replay and escalation decisions reviewable without hidden dependencies.", Priority: "P0", Dependencies: []string{"BIG-4203", "OPE-72", "OPE-73"}},
			{ObjectiveID: "obj-triage-handoff", Title: "Confirm triage and cross-team handoff readiness", Persona: "Cross-Team Operator", Outcome: "Reviewers can evaluate assignment, handoff, and queue-state transitions as one operator journey.", SuccessSignal: "The triage frame exposes action states, owner switches, and handoff exceptions explicitly.", Priority: "P0", Dependencies: []string{"BIG-4203", "OPE-76", "OPE-79", "OPE-132"}},
		},
		Wireframes: []WireframeSurface{
			{SurfaceID: "wf-overview", Name: "Overview command deck", Device: "desktop", EntryPoint: "/overview", PrimaryBlocks: []string{"top bar", "kpi strip", "risk panel", "drill-down table"}, ReviewNotes: []string{"Confirm metric density and executive scan path.", "Check alert prominence versus weekly summary card."}},
			{SurfaceID: "wf-queue", Name: "Queue control center", Device: "desktop", EntryPoint: "/queue", PrimaryBlocks: []string{"approval queue", "selection toolbar", "filters", "audit rail"}, ReviewNotes: []string{"Validate batch-approve CTA hierarchy.", "Review denied-role behavior for non-operator personas."}},
			{SurfaceID: "wf-run-detail", Name: "Run detail and replay", Device: "desktop", EntryPoint: "/runs/detail", PrimaryBlocks: []string{"timeline", "artifact drawer", "replay controls", "audit notes"}, ReviewNotes: []string{"Check replay mode discoverability.", "Ensure escalation path is visible next to audit evidence."}},
			{SurfaceID: "wf-triage", Name: "Triage and handoff board", Device: "desktop", EntryPoint: "/triage", PrimaryBlocks: []string{"severity lanes", "bulk actions", "handoff panel", "ownership history"}, ReviewNotes: []string{"Validate cross-team operator workflow.", "Confirm exception path for denied escalation."}},
		},
		Interactions: []InteractionFlow{
			{FlowID: "flow-overview-drilldown", Name: "Overview to investigation drill-down", Trigger: "VP Eng selects a KPI card or blocker cluster on the overview page", SystemResponse: "The console pivots into the matching queue or run-detail slice while preserving context filters.", States: []string{"default", "focus", "handoff-ready"}, Exceptions: []string{"Warn when the requested slice is permission-denied.", "Show fallback summary when no matching runs exist."}},
			{FlowID: "flow-queue-bulk-approval", Name: "Queue batch approval review", Trigger: "Platform Admin selects multiple tasks and opens the bulk approval toolbar", SystemResponse: "The queue shows approval scope, audit consequence, and denied-role messaging before submit.", States: []string{"default", "selection", "confirming", "success"}, Exceptions: []string{"Disable submit when tasks cross unauthorized scopes.", "Route to audit timeline when approval policy changes mid-flow."}},
			{FlowID: "flow-run-replay", Name: "Run replay with evidence audit", Trigger: "Eng Lead switches replay mode on a failed run", SystemResponse: "The page updates the timeline, artifacts, and escalation controls while keeping the audit trail visible.", States: []string{"default", "replay", "compare", "escalated"}, Exceptions: []string{"Show replay-unavailable state for incomplete artifacts.", "Require escalation reason before handoff."}},
			{FlowID: "flow-triage-handoff", Name: "Triage ownership reassignment and handoff", Trigger: "Cross-Team Operator bulk-assigns a finding set or opens the handoff panel", SystemResponse: "The triage board updates owner, workflow, and handoff evidence in one acknowledgement step.", States: []string{"default", "selected", "handoff", "completed"}, Exceptions: []string{"Block handoff when reviewer coverage is incomplete.", "Record denied-role attempt in the audit summary."}},
		},
		OpenQuestions: []OpenQuestion{
			{QuestionID: "oq-role-density", Theme: "role-matrix", Question: "Should VP Eng see queue batch controls in read-only form or be routed to a summary-only state?", Owner: "product-experience", Impact: "Changes denial-path copy, button placement, and review criteria for queue and triage pages."},
			{QuestionID: "oq-alert-priority", Theme: "information-architecture", Question: "Should regression alerts outrank approval alerts in the top bar for the design sprint prototype?", Owner: "engineering-operations", Impact: "Affects alert hierarchy and the scan path used in the overview and triage reviews."},
			{QuestionID: "oq-handoff-evidence", Theme: "handoff", Question: "How much ownership history must stay visible before the run-detail and triage pages collapse older audit entries?", Owner: "orchestration-office", Impact: "Shapes the default density of the audit rail and the threshold for the review-ready packet."},
		},
		ReviewerChecklist: []ReviewerChecklistItem{
			{ItemID: "chk-overview-kpi-scan", SurfaceID: "wf-overview", Prompt: "Verify the KPI strip still supports one-screen executive scanning before drill-down.", Owner: "VP Eng", Status: "ready", EvidenceLinks: []string{"wf-overview", "flow-overview-drilldown"}, Notes: "Use the overview card hierarchy as the primary decision frame."},
			{ItemID: "chk-overview-alert-hierarchy", SurfaceID: "wf-overview", Prompt: "Confirm alert priority is readable when approvals and regressions compete for attention.", Owner: "engineering-operations", Status: "open", EvidenceLinks: []string{"wf-overview", "oq-alert-priority"}},
			{ItemID: "chk-queue-batch-approval", SurfaceID: "wf-queue", Prompt: "Check that batch approval clearly communicates scope, denial paths, and audit consequences.", Owner: "Platform Admin", Status: "ready", EvidenceLinks: []string{"wf-queue", "flow-queue-bulk-approval"}},
			{ItemID: "chk-queue-role-density", SurfaceID: "wf-queue", Prompt: "Validate whether VP Eng should get a summary-only queue variant instead of operator controls.", Owner: "product-experience", Status: "open", EvidenceLinks: []string{"wf-queue", "oq-role-density"}},
			{ItemID: "chk-run-replay-context", SurfaceID: "wf-run-detail", Prompt: "Ensure replay, compare, and escalation states remain distinguishable without narration.", Owner: "Eng Lead", Status: "ready", EvidenceLinks: []string{"wf-run-detail", "flow-run-replay"}},
			{ItemID: "chk-run-audit-density", SurfaceID: "wf-run-detail", Prompt: "Confirm the audit rail retains enough ownership history before collapsing older entries.", Owner: "orchestration-office", Status: "open", EvidenceLinks: []string{"wf-run-detail", "oq-handoff-evidence"}},
			{ItemID: "chk-triage-handoff-clarity", SurfaceID: "wf-triage", Prompt: "Check that cross-team handoff consequences are explicit before ownership changes commit.", Owner: "Cross-Team Operator", Status: "ready", EvidenceLinks: []string{"wf-triage", "flow-triage-handoff"}},
			{ItemID: "chk-triage-bulk-assign", SurfaceID: "wf-triage", Prompt: "Validate bulk assignment visibility without burying the audit context.", Owner: "Platform Admin", Status: "ready", EvidenceLinks: []string{"wf-triage", "flow-triage-handoff"}},
		},
		DecisionLog: []ReviewDecision{
			{DecisionID: "dec-overview-alert-stack", SurfaceID: "wf-overview", Owner: "product-experience", Summary: "Keep approval and regression alerts in one stacked priority rail.", Rationale: "Reviewers need one comparison lane before jumping into queue or triage surfaces.", Status: "accepted"},
			{DecisionID: "dec-queue-vp-summary", SurfaceID: "wf-queue", Owner: "VP Eng", Summary: "Route VP Eng to a summary-first queue state instead of operator controls.", Rationale: "The VP Eng persona needs queue visibility without accidental action affordances.", Status: "proposed", FollowUp: "Resolve after the next design critique with policy owners."},
			{DecisionID: "dec-run-detail-audit-rail", SurfaceID: "wf-run-detail", Owner: "Eng Lead", Summary: "Keep audit evidence visible beside replay controls in all replay states.", Rationale: "Replay decisions are inseparable from audit context and escalation evidence.", Status: "accepted"},
			{DecisionID: "dec-triage-handoff-density", SurfaceID: "wf-triage", Owner: "Cross-Team Operator", Summary: "Preserve ownership history in the triage rail until handoff is acknowledged.", Rationale: "Operators need a stable handoff trail before collapsing older events.", Status: "accepted"},
		},
		RoleMatrix: []ReviewRoleAssignment{
			{AssignmentID: "role-overview-vp-eng", SurfaceID: "wf-overview", Role: "VP Eng", Responsibilities: []string{"approve overview scan path", "validate KPI-to-drilldown narrative"}, ChecklistItemIDs: []string{"chk-overview-kpi-scan"}, DecisionIDs: []string{"dec-overview-alert-stack"}, Status: "ready"},
			{AssignmentID: "role-overview-eng-ops", SurfaceID: "wf-overview", Role: "engineering-operations", Responsibilities: []string{"review alert priority posture"}, ChecklistItemIDs: []string{"chk-overview-alert-hierarchy"}, DecisionIDs: []string{"dec-overview-alert-stack"}, Status: "open"},
			{AssignmentID: "role-queue-platform-admin", SurfaceID: "wf-queue", Role: "Platform Admin", Responsibilities: []string{"validate batch-approval copy", "confirm permission posture"}, ChecklistItemIDs: []string{"chk-queue-batch-approval"}, DecisionIDs: []string{"dec-queue-vp-summary"}, Status: "ready"},
			{AssignmentID: "role-queue-product-experience", SurfaceID: "wf-queue", Role: "product-experience", Responsibilities: []string{"tune summary-only VP variant"}, ChecklistItemIDs: []string{"chk-queue-role-density"}, DecisionIDs: []string{"dec-queue-vp-summary"}, Status: "open"},
			{AssignmentID: "role-run-detail-eng-lead", SurfaceID: "wf-run-detail", Role: "Eng Lead", Responsibilities: []string{"approve replay-state clarity", "confirm escalation adjacency"}, ChecklistItemIDs: []string{"chk-run-replay-context"}, DecisionIDs: []string{"dec-run-detail-audit-rail"}, Status: "ready"},
			{AssignmentID: "role-run-detail-orchestration-office", SurfaceID: "wf-run-detail", Role: "orchestration-office", Responsibilities: []string{"review audit density threshold"}, ChecklistItemIDs: []string{"chk-run-audit-density"}, DecisionIDs: []string{"dec-run-detail-audit-rail"}, Status: "open"},
			{AssignmentID: "role-triage-cross-team-operator", SurfaceID: "wf-triage", Role: "Cross-Team Operator", Responsibilities: []string{"approve handoff clarity", "validate ownership transition story"}, ChecklistItemIDs: []string{"chk-triage-handoff-clarity"}, DecisionIDs: []string{"dec-triage-handoff-density"}, Status: "ready"},
			{AssignmentID: "role-triage-platform-admin", SurfaceID: "wf-triage", Role: "Platform Admin", Responsibilities: []string{"confirm bulk-assign audit visibility"}, ChecklistItemIDs: []string{"chk-triage-bulk-assign"}, DecisionIDs: []string{"dec-triage-handoff-density"}, Status: "ready"},
		},
		SignoffLog: []ReviewSignoff{
			{SignoffID: "sig-overview-vp-eng", AssignmentID: "role-overview-vp-eng", SurfaceID: "wf-overview", Role: "VP Eng", Status: "approved", Required: true, EvidenceLinks: []string{"chk-overview-kpi-scan", "dec-overview-alert-stack"}, Notes: "Overview narrative approved for design sprint review.", RequestedAt: "2026-03-10T09:00:00Z", DueAt: "2026-03-12T18:00:00Z", EscalationOwner: "design-program-manager", SLAStatus: "met"},
			{SignoffID: "sig-queue-platform-admin", AssignmentID: "role-queue-platform-admin", SurfaceID: "wf-queue", Role: "Platform Admin", Status: "approved", Required: true, EvidenceLinks: []string{"chk-queue-batch-approval", "dec-queue-vp-summary"}, Notes: "Queue control actions meet operator review criteria.", RequestedAt: "2026-03-10T11:00:00Z", DueAt: "2026-03-13T18:00:00Z", EscalationOwner: "platform-ops-manager", SLAStatus: "met"},
			{SignoffID: "sig-run-detail-eng-lead", AssignmentID: "role-run-detail-eng-lead", SurfaceID: "wf-run-detail", Role: "Eng Lead", Status: "pending", Required: true, EvidenceLinks: []string{"chk-run-replay-context", "dec-run-detail-audit-rail"}, Notes: "Waiting for final replay-state copy review.", RequestedAt: "2026-03-12T11:00:00Z", DueAt: "2026-03-15T18:00:00Z", EscalationOwner: "engineering-director", SLAStatus: "at-risk", ReminderOwner: "design-program-manager", ReminderChannel: "slack", LastReminderAt: "2026-03-14T09:45:00Z", NextReminderAt: "2026-03-15T10:00:00Z", ReminderCadence: "daily", ReminderStatus: "scheduled"},
			{SignoffID: "sig-triage-cross-team-operator", AssignmentID: "role-triage-cross-team-operator", SurfaceID: "wf-triage", Role: "Cross-Team Operator", Status: "approved", Required: true, EvidenceLinks: []string{"chk-triage-handoff-clarity", "dec-triage-handoff-density"}, Notes: "Cross-team handoff flow approved for prototype review.", RequestedAt: "2026-03-11T14:00:00Z", DueAt: "2026-03-13T12:00:00Z", EscalationOwner: "cross-team-program-manager", SLAStatus: "met"},
		},
		BlockerLog: []ReviewBlocker{
			{BlockerID: "blk-run-detail-copy-final", SurfaceID: "wf-run-detail", SignoffID: "sig-run-detail-eng-lead", Owner: "product-experience", Summary: "Replay-state copy still needs final wording review before Eng Lead signoff can close.", Status: "open", Severity: "medium", EscalationOwner: "design-program-manager", NextAction: "Review replay-state copy with Eng Lead and update the run-detail frame in the next critique.", FreezeException: true, FreezeOwner: "release-director", FreezeUntil: "2026-03-18T18:00:00Z", FreezeReason: "Allow the design sprint review pack to ship while tracked copy cleanup lands in the next critique.", FreezeApprovedBy: "release-director", FreezeApprovedAt: "2026-03-14T08:30:00Z", FreezeRenewalOwner: "release-director", FreezeRenewalBy: "2026-03-17T12:00:00Z", FreezeRenewalStatus: "review-needed"},
		},
		BlockerTimeline: []ReviewBlockerEvent{
			{EventID: "evt-run-detail-copy-opened", BlockerID: "blk-run-detail-copy-final", Actor: "product-experience", Status: "opened", Summary: "Captured the final replay-state copy gap during design sprint prep.", Timestamp: "2026-03-13T10:00:00Z", NextAction: "Draft updated replay labels before the Eng Lead review."},
			{EventID: "evt-run-detail-copy-escalated", BlockerID: "blk-run-detail-copy-final", Actor: "design-program-manager", Status: "escalated", Summary: "Scheduled a joint wording review with Eng Lead and product-experience to close the signoff blocker.", Timestamp: "2026-03-14T09:30:00Z", NextAction: "Refresh the run-detail frame annotations after the wording review completes.", HandoffFrom: "product-experience", HandoffTo: "Eng Lead", Channel: "design-critique", ArtifactRef: "wf-run-detail#copy-v5", AckOwner: "Eng Lead", AckAt: "2026-03-14T10:15:00Z", AckStatus: "acknowledged"},
		},
	}
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
					States: []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"export"}}, {Name: "empty", AllowedActions: []string{"drill-down"}}, {Name: "error", AllowedActions: []string{"audit"}}},
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
					States:              []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"export"}}, {Name: "empty", AllowedActions: []string{"audit"}}, {Name: "error", AllowedActions: []string{"audit"}}},
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
					States: []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"export"}}, {Name: "empty", AllowedActions: []string{"drill-down"}}, {Name: "error", AllowedActions: []string{"audit"}}},
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
					States:              []SurfaceState{{Name: "default"}, {Name: "loading", AllowedActions: []string{"export"}}, {Name: "empty", AllowedActions: []string{"audit"}}, {Name: "error", AllowedActions: []string{"audit"}}},
					SupportsBulkActions: true,
				},
			},
		},
		Contracts: []SurfaceInteractionContract{
			{
				SurfaceName:       "Overview",
				RequiredActionIDs: []string{"drill-down", "export", "audit"},
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

func flattenNode(node NavigationNode, parentPath string, depth int, parentID string) []NavigationEntry {
	path := joinPath(parentPath, node.Segment)
	entries := []NavigationEntry{{
		NodeID:   node.NodeID,
		Title:    node.Title,
		Path:     path,
		Depth:    depth,
		ParentID: parentID,
		ScreenID: node.ScreenID,
	}}
	for _, child := range node.Children {
		entries = append(entries, flattenNode(child, path, depth+1, node.NodeID)...)
	}
	return entries
}

func missingPathsForDescendants(node NavigationNode, parentPath string, routeIndex map[string]NavigationRoute) []string {
	path := joinPath(parentPath, node.Segment)
	var missing []string
	if len(node.Children) > 0 {
		if _, ok := routeIndex[path]; !ok {
			missing = append(missing, path)
		}
	}
	for _, child := range node.Children {
		missing = append(missing, missingPathsForDescendants(child, path, routeIndex)...)
	}
	return missing
}

func joinPath(parentPath, segment string) string {
	base := NormalizeRoutePath(parentPath)
	part := strings.Trim(strings.TrimSpace(segment), "/")
	if part == "" {
		return base
	}
	if base == "/" {
		return "/" + part
	}
	return base + "/" + part
}

func hasAllKeys(actual, required map[string]bool) bool {
	for key := range required {
		if !actual[key] {
			return false
		}
	}
	return true
}

func boolString(v bool) string {
	if v {
		return "True"
	}
	return "False"
}

func joinOrNone(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}

func joinCSVOrNone(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ",")
}

func fallback(value, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return value
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedStringMapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func intStringMap(m map[string]int) map[string]string {
	out := make(map[string]string, len(m))
	for key, value := range m {
		out[key] = itoa(value)
	}
	return out
}

func sortedNestedMapKeys(m map[string]map[string]int) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func latestBlockerEventLabel(events []ReviewBlockerEvent, blockerID string) string {
	latest := ""
	for _, event := range events {
		e := event.normalized()
		if e.BlockerID != blockerID {
			continue
		}
		label := e.EventID + "/" + e.Status + "/" + e.Actor + "@" + e.Timestamp
		if latest == "" || e.Timestamp > strings.SplitN(latest, "@", 2)[1] {
			latest = label
		}
	}
	if latest == "" {
		return "none"
	}
	return latest
}

func latestBlockerEvent(events []ReviewBlockerEvent) ReviewBlockerEvent {
	var latest ReviewBlockerEvent
	for i, event := range events {
		e := event.normalized()
		if i == 0 || e.Timestamp > latest.Timestamp {
			latest = e
		}
	}
	return latest
}

func formatListMap(m map[string][]string) string {
	if len(m) == 0 {
		return "none"
	}
	var parts []string
	for _, key := range sortedKeys(m) {
		parts = append(parts, key+"="+strings.Join(m[key], ", "))
	}
	return strings.Join(parts, "; ")
}

func formatNestedListMap(m map[string]map[string][]string) string {
	if len(m) == 0 {
		return "none"
	}
	var parts []string
	for _, key := range sortedKeys(m) {
		var nested []string
		for _, state := range sortedKeys(m[key]) {
			nested = append(nested, state+"="+strings.Join(m[key][state], ", "))
		}
		parts = append(parts, key+"="+strings.Join(nested, "; "))
	}
	return strings.Join(parts, " | ")
}

func round1(v float64) float64 {
	if v >= 0 {
		return float64(int(v*10+0.5)) / 10
	}
	return float64(int(v*10-0.5)) / 10
}

func format1(v float64) string {
	return fmt.Sprintf("%.1f", v)
}

func itoa(v int) string {
	return strconv.Itoa(v)
}
