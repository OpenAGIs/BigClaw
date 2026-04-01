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

type UIReviewPack struct {
	IssueID                   string             `json:"issue_id"`
	Title                     string             `json:"title"`
	Version                   string             `json:"version"`
	Objectives                []ReviewObjective  `json:"objectives,omitempty"`
	Wireframes                []WireframeSurface `json:"wireframes,omitempty"`
	Interactions              []InteractionFlow  `json:"interactions,omitempty"`
	OpenQuestions             []OpenQuestion     `json:"open_questions,omitempty"`
	ReviewerChecklist         []json.RawMessage  `json:"reviewer_checklist,omitempty"`
	RequiresReviewerChecklist bool               `json:"requires_reviewer_checklist,omitempty"`
	DecisionLog               []json.RawMessage  `json:"decision_log,omitempty"`
	RequiresDecisionLog       bool               `json:"requires_decision_log,omitempty"`
	RoleMatrix                []json.RawMessage  `json:"role_matrix,omitempty"`
	RequiresRoleMatrix        bool               `json:"requires_role_matrix,omitempty"`
	SignoffLog                []json.RawMessage  `json:"signoff_log,omitempty"`
	RequiresSignoffLog        bool               `json:"requires_signoff_log,omitempty"`
	BlockerLog                []json.RawMessage  `json:"blocker_log,omitempty"`
	RequiresBlockerLog        bool               `json:"requires_blocker_log,omitempty"`
	BlockerTimeline           []json.RawMessage  `json:"blocker_timeline,omitempty"`
	RequiresBlockerTimeline   bool               `json:"requires_blocker_timeline,omitempty"`
}

func (p UIReviewPack) normalized() UIReviewPack {
	out := p
	for i, objective := range out.Objectives {
		out.Objectives[i] = objective.normalized()
	}
	for i, question := range out.OpenQuestions {
		out.OpenQuestions[i] = question.normalized()
	}
	return out
}

type UIReviewPackAudit struct {
	Ready                     bool     `json:"ready"`
	ObjectiveCount            int      `json:"objective_count"`
	WireframeCount            int      `json:"wireframe_count"`
	InteractionCount          int      `json:"interaction_count"`
	OpenQuestionCount         int      `json:"open_question_count"`
	ChecklistCount            int      `json:"checklist_count,omitempty"`
	DecisionCount             int      `json:"decision_count,omitempty"`
	RoleAssignmentCount       int      `json:"role_assignment_count,omitempty"`
	SignoffCount              int      `json:"signoff_count,omitempty"`
	BlockerCount              int      `json:"blocker_count,omitempty"`
	BlockerTimelineCount      int      `json:"blocker_timeline_count,omitempty"`
	MissingSections           []string `json:"missing_sections,omitempty"`
	ObjectivesMissingSignals  []string `json:"objectives_missing_signals,omitempty"`
	WireframesMissingBlocks   []string `json:"wireframes_missing_blocks,omitempty"`
	InteractionsMissingStates []string `json:"interactions_missing_states,omitempty"`
	UnresolvedQuestionIDs     []string `json:"unresolved_question_ids,omitempty"`
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

	ready := len(missingSections) == 0 &&
		len(objectivesMissingSignals) == 0 &&
		len(wireframesMissingBlocks) == 0 &&
		len(interactionsMissingStates) == 0

	return UIReviewPackAudit{
		Ready:                     ready,
		ObjectiveCount:            len(pack.Objectives),
		WireframeCount:            len(pack.Wireframes),
		InteractionCount:          len(pack.Interactions),
		OpenQuestionCount:         len(pack.OpenQuestions),
		ChecklistCount:            len(pack.ReviewerChecklist),
		DecisionCount:             len(pack.DecisionLog),
		RoleAssignmentCount:       len(pack.RoleMatrix),
		SignoffCount:              len(pack.SignoffLog),
		BlockerCount:              len(pack.BlockerLog),
		BlockerTimelineCount:      len(pack.BlockerTimeline),
		MissingSections:           missingSections,
		ObjectivesMissingSignals:  objectivesMissingSignals,
		WireframesMissingBlocks:   wireframesMissingBlocks,
		InteractionsMissingStates: interactionsMissingStates,
		UnresolvedQuestionIDs:     unresolvedQuestionIDs,
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
