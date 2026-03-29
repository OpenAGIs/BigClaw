package designsystemparity

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

var foundationCategories = []string{"color", "spacing", "typography", "motion", "radius"}
var componentReadinessOrder = map[string]int{"draft": 0, "alpha": 1, "beta": 2, "stable": 3}
var requiredInteractionStates = []string{"default", "hover", "disabled"}
var requiredTopBarAccessibility = []string{"keyboard-navigation", "screen-reader-label", "focus-visible"}
var requiredTopBarShortcuts = []string{"cmd+k", "ctrl+k"}

type DesignToken struct {
	Name         string `json:"name"`
	Category     string `json:"category"`
	Value        string `json:"value"`
	SemanticRole string `json:"semantic_role,omitempty"`
	Theme        string `json:"theme,omitempty"`
}

func (t DesignToken) ToMap() (map[string]any, error) { return toMap(t) }
func DesignTokenFromMap(data map[string]any) (DesignToken, error) {
	var token DesignToken
	return token, fromMap(data, &token)
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
	DocumentationComplete     bool               `json:"documentation_complete"`
}

func (c ComponentSpec) TokenNames() []string {
	names := make([]string, 0)
	seen := map[string]struct{}{}
	for _, variant := range c.Variants {
		for _, token := range variant.Tokens {
			if _, ok := seen[token]; ok {
				continue
			}
			seen[token] = struct{}{}
			names = append(names, token)
		}
	}
	return names
}

func (c ComponentSpec) StateCoverage() []string {
	states := make([]string, 0)
	seen := map[string]struct{}{}
	for _, variant := range c.Variants {
		for _, state := range variant.States {
			if _, ok := seen[state]; ok {
				continue
			}
			seen[state] = struct{}{}
			states = append(states, state)
		}
	}
	return states
}

func (c ComponentSpec) MissingRequiredStates() []string {
	coverage := map[string]struct{}{}
	for _, state := range c.StateCoverage() {
		coverage[state] = struct{}{}
	}
	missing := make([]string, 0)
	for _, state := range requiredInteractionStates {
		if _, ok := coverage[state]; !ok {
			missing = append(missing, state)
		}
	}
	return missing
}

func (c ComponentSpec) ReleaseReady() bool {
	return componentReadinessOrder[c.Readiness] >= componentReadinessOrder["beta"] &&
		c.DocumentationComplete &&
		len(c.AccessibilityRequirements) > 0 &&
		len(c.MissingRequiredStates()) == 0
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
	score := maxFloat(0, (float64(ready*100)-float64(penalties*10))/float64(a.ComponentCount))
	return round1(score)
}

func (a DesignSystemAudit) ToMap() (map[string]any, error) { return toMap(a) }
func DesignSystemAuditFromMap(data map[string]any) (DesignSystemAudit, error) {
	var audit DesignSystemAudit
	return audit, fromMap(data, &audit)
}

type DesignSystem struct {
	Name       string          `json:"name"`
	Version    string          `json:"version"`
	Tokens     []DesignToken   `json:"tokens,omitempty"`
	Components []ComponentSpec `json:"components,omitempty"`
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
		index[token.Name] = token
	}
	return index
}

func (s DesignSystem) ToMap() (map[string]any, error) { return toMap(s) }
func DesignSystemFromMap(data map[string]any) (DesignSystem, error) {
	var system DesignSystem
	return system, fromMap(data, &system)
}

type ComponentLibrary struct{}

func (ComponentLibrary) Audit(system DesignSystem) DesignSystemAudit {
	usedTokens := map[string]struct{}{}
	var releaseReady, missingDocs, missingAccessibility, missingStates []string
	undefined := map[string][]string{}
	tokenIndex := system.TokenIndex()

	for _, component := range system.Components {
		for _, token := range component.TokenNames() {
			usedTokens[token] = struct{}{}
		}
		missingTokens := make([]string, 0)
		for _, token := range component.TokenNames() {
			if _, ok := tokenIndex[token]; !ok {
				missingTokens = append(missingTokens, token)
			}
		}
		sort.Strings(missingTokens)
		if len(missingTokens) > 0 {
			undefined[component.Name] = missingTokens
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

	tokenOrphans := make([]string, 0)
	for _, token := range system.Tokens {
		if _, ok := usedTokens[token.Name]; !ok {
			tokenOrphans = append(tokenOrphans, token.Name)
		}
	}
	sort.Strings(releaseReady)
	sort.Strings(missingDocs)
	sort.Strings(missingAccessibility)
	sort.Strings(missingStates)
	sort.Strings(tokenOrphans)

	return DesignSystemAudit{
		SystemName:                     system.Name,
		Version:                        system.Version,
		TokenCounts:                    system.TokenCounts(),
		ComponentCount:                 len(system.Components),
		ReleaseReadyComponents:         releaseReady,
		ComponentsMissingDocs:          missingDocs,
		ComponentsMissingAccessibility: missingAccessibility,
		ComponentsMissingStates:        missingStates,
		UndefinedTokenRefs:             undefined,
		TokenOrphans:                   tokenOrphans,
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
func (b ConsoleTopBar) ToMap() (map[string]any, error) { return toMap(b) }
func ConsoleTopBarFromMap(data map[string]any) (ConsoleTopBar, error) {
	var topBar ConsoleTopBar
	return topBar, fromMap(data, &topBar)
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
	return len(a.MissingCapabilities) == 0 && a.DocumentationComplete && a.AccessibilityComplete && a.CommandShortcutSupported
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
	for _, item := range strings.Split(topBar.CommandEntry.Shortcut, "/") {
		item = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(item), " ", ""))
		if item != "" {
			normalized[item] = struct{}{}
		}
	}
	access := map[string]struct{}{}
	for _, item := range topBar.AccessibilityRequirements {
		access[item] = struct{}{}
	}
	accessComplete := true
	for _, req := range requiredTopBarAccessibility {
		if _, ok := access[req]; !ok {
			accessComplete = false
			break
		}
	}
	shortcutComplete := true
	for _, req := range requiredTopBarShortcuts {
		if _, ok := normalized[req]; !ok {
			shortcutComplete = false
			break
		}
	}
	return ConsoleTopBarAudit{
		Name:                     topBar.Name,
		MissingCapabilities:      missing,
		DocumentationComplete:    topBar.DocumentationComplete,
		AccessibilityComplete:    accessComplete,
		CommandShortcutSupported: shortcutComplete,
		CommandCount:             len(topBar.CommandEntry.Commands),
	}
}

type NavigationRoute struct {
	Path      string `json:"path"`
	ScreenID  string `json:"screen_id"`
	Title     string `json:"title"`
	NavNodeID string `json:"nav_node_id,omitempty"`
	Layout    string `json:"layout,omitempty"`
}

func normalizeRoutePath(path string) string {
	stripped := strings.Trim(path, "/")
	if stripped == "" {
		return "/"
	}
	return "/" + stripped
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
	return len(a.DuplicateRoutes) == 0 && len(a.MissingRouteNodes) == 0 && len(a.SecondaryNavGaps) == 0 && len(a.OrphanRoutes) == 0
}

func (a InformationArchitectureAudit) ToMap() (map[string]any, error) { return toMap(a) }
func InformationArchitectureAuditFromMap(data map[string]any) (InformationArchitectureAudit, error) {
	var audit InformationArchitectureAudit
	return audit, fromMap(data, &audit)
}

type InformationArchitecture struct {
	GlobalNav []NavigationNode  `json:"global_nav,omitempty"`
	Routes    []NavigationRoute `json:"routes,omitempty"`
}

func (a InformationArchitecture) RouteIndex() map[string]NavigationRoute {
	index := map[string]NavigationRoute{}
	for _, route := range a.Routes {
		route.Path = normalizeRoutePath(route.Path)
		if _, ok := index[route.Path]; !ok {
			index[route.Path] = route
		}
	}
	return index
}

func (a InformationArchitecture) NavigationEntries() []NavigationEntry {
	entries := make([]NavigationEntry, 0)
	for _, node := range a.GlobalNav {
		entries = append(entries, a.flattenNode(node, "", 0, "")...)
	}
	return entries
}

func (a InformationArchitecture) ResolveRoute(path string) *NavigationRoute {
	route, ok := a.RouteIndex()[normalizeRoutePath(path)]
	if !ok {
		return nil
	}
	return &route
}

func (a InformationArchitecture) Audit() InformationArchitectureAudit {
	entries := a.NavigationEntries()
	routeCounts := map[string]int{}
	for _, route := range a.Routes {
		routeCounts[normalizeRoutePath(route.Path)]++
	}
	duplicateRoutes := make([]string, 0)
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
	secondaryNavGaps := map[string][]string{}
	for _, root := range a.GlobalNav {
		gaps := a.missingPathsForDescendants(root, "")
		sort.Strings(gaps)
		if len(gaps) > 0 {
			secondaryNavGaps[root.Title] = gaps
		}
	}
	navPaths := map[string]struct{}{}
	for _, entry := range entries {
		navPaths[entry.Path] = struct{}{}
	}
	orphanRoutes := make([]string, 0)
	for _, route := range a.Routes {
		path := normalizeRoutePath(route.Path)
		if _, ok := navPaths[path]; !ok {
			orphanRoutes = append(orphanRoutes, path)
		}
	}
	sort.Strings(orphanRoutes)

	return InformationArchitectureAudit{
		TotalNavigationNodes: len(entries),
		TotalRoutes:          len(a.Routes),
		DuplicateRoutes:      duplicateRoutes,
		MissingRouteNodes:    missingRouteNodes,
		SecondaryNavGaps:     secondaryNavGaps,
		OrphanRoutes:         orphanRoutes,
	}
}

func (a InformationArchitecture) ToMap() (map[string]any, error) { return toMap(a) }
func InformationArchitectureFromMap(data map[string]any) (InformationArchitecture, error) {
	var architecture InformationArchitecture
	return architecture, fromMap(data, &architecture)
}

func (a InformationArchitecture) flattenNode(node NavigationNode, parentPath string, depth int, parentID string) []NavigationEntry {
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
		entries = append(entries, a.flattenNode(child, path, depth+1, node.NodeID)...)
	}
	return entries
}

func (a InformationArchitecture) missingPathsForDescendants(node NavigationNode, parentPath string) []string {
	path := joinPath(parentPath, node.Segment)
	missing := make([]string, 0)
	if len(node.Children) > 0 {
		if _, ok := a.RouteIndex()[path]; !ok {
			missing = append(missing, path)
		}
	}
	for _, child := range node.Children {
		missing = append(missing, a.missingPathsForDescendants(child, path)...)
	}
	return missing
}

func joinPath(parentPath, segment string) string {
	base := normalizeRoutePath(parentPath)
	part := strings.Trim(segment, "/")
	if part == "" {
		return base
	}
	if base == "/" {
		return "/" + part
	}
	return base + "/" + part
}

type RolePermissionScenario struct {
	ScreenID     string   `json:"screen_id"`
	AllowedRoles []string `json:"allowed_roles,omitempty"`
	DeniedRoles  []string `json:"denied_roles,omitempty"`
	AuditEvent   string   `json:"audit_event,omitempty"`
}

func (s RolePermissionScenario) MissingCoverage() []string {
	missing := make([]string, 0)
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
	SourceOfTruth            string  `json:"source_of_truth,omitempty"`
	RenderedValue            string  `json:"rendered_value,omitempty"`
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
func (c DataAccuracyCheck) Passes() bool { return c.WithinTolerance() && c.WithinFreshnessSLO() }

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
	KeyboardAccessible bool     `json:"keyboard_accessible"`
	EmptyStateGuidance bool     `json:"empty_state_guidance"`
	RecoverySupport    bool     `json:"recovery_support"`
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
	emitted := map[string]struct{}{}
	for _, field := range r.EmittedFields {
		emitted[field] = struct{}{}
	}
	missing := make([]string, 0)
	for _, field := range r.RequiredFields {
		if _, ok := emitted[field]; !ok {
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
func (r AuditRequirement) Complete() bool { return len(r.MissingFields()) == 0 && r.RetentionMet() }

type UIAcceptanceSuite struct {
	Name                  string                   `json:"name"`
	Version               string                   `json:"version"`
	RolePermissions       []RolePermissionScenario `json:"role_permissions,omitempty"`
	DataAccuracyChecks    []DataAccuracyCheck      `json:"data_accuracy_checks,omitempty"`
	PerformanceBudgets    []PerformanceBudget      `json:"performance_budgets,omitempty"`
	UsabilityJourneys     []UsabilityJourney       `json:"usability_journeys,omitempty"`
	AuditRequirements     []AuditRequirement       `json:"audit_requirements,omitempty"`
	DocumentationComplete bool                     `json:"documentation_complete"`
}

func (s UIAcceptanceSuite) ToMap() (map[string]any, error) { return toMap(s) }
func UIAcceptanceSuiteFromMap(data map[string]any) (UIAcceptanceSuite, error) {
	var suite UIAcceptanceSuite
	return suite, fromMap(data, &suite)
}

type UIAcceptanceAudit struct {
	Name                      string   `json:"name"`
	Version                   string   `json:"version"`
	PermissionGaps            []string `json:"permission_gaps,omitempty"`
	FailingDataChecks         []string `json:"failing_data_checks,omitempty"`
	FailingPerformanceBudgets []string `json:"failing_performance_budgets,omitempty"`
	FailingUsabilityJourneys  []string `json:"failing_usability_journeys,omitempty"`
	IncompleteAuditTrails     []string `json:"incomplete_audit_trails,omitempty"`
	DocumentationComplete     bool     `json:"documentation_complete"`
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
	for _, item := range checks {
		if item {
			passed++
		}
	}
	return round1(float64(passed) / float64(len(checks)) * 100)
}

type UIAcceptanceLibrary struct{}

func (UIAcceptanceLibrary) Audit(suite UIAcceptanceSuite) UIAcceptanceAudit {
	permissionGaps := make([]string, 0)
	for _, scenario := range suite.RolePermissions {
		if gaps := scenario.MissingCoverage(); len(gaps) > 0 {
			permissionGaps = append(permissionGaps, fmt.Sprintf("%s: missing=%s", scenario.ScreenID, strings.Join(gaps, ", ")))
		}
	}
	failingData := make([]string, 0)
	for _, check := range suite.DataAccuracyChecks {
		if !check.Passes() {
			failingData = append(failingData, fmt.Sprintf("%s.%s: delta=%s freshness=%ds", check.ScreenID, check.MetricID, formatFloat(check.ObservedDelta), check.ObservedFreshnessSeconds))
		}
	}
	failingPerf := make([]string, 0)
	for _, budget := range suite.PerformanceBudgets {
		if !budget.WithinBudget() {
			line := fmt.Sprintf("%s.%s: p95=%dms", budget.SurfaceID, budget.Interaction, budget.ObservedP95MS)
			if budget.TargetTTIMS > 0 {
				line += fmt.Sprintf(" tti=%dms", budget.ObservedTTIMS)
			}
			failingPerf = append(failingPerf, line)
		}
	}
	failingUsability := make([]string, 0)
	for _, journey := range suite.UsabilityJourneys {
		if !journey.Passes() {
			failingUsability = append(failingUsability, fmt.Sprintf("%s: steps=%d/%d", journey.JourneyID, journey.ObservedSteps, journey.ExpectedMaxSteps))
		}
	}
	incompleteAudit := make([]string, 0)
	for _, requirement := range suite.AuditRequirements {
		if requirement.Complete() {
			continue
		}
		parts := make([]string, 0)
		if missing := requirement.MissingFields(); len(missing) > 0 {
			parts = append(parts, fmt.Sprintf("missing_fields=%s", strings.Join(missing, ", ")))
		}
		if !requirement.RetentionMet() {
			parts = append(parts, fmt.Sprintf("retention=%d/%dd", requirement.ObservedRetentionDays, requirement.RetentionDays))
		}
		incompleteAudit = append(incompleteAudit, fmt.Sprintf("%s: %s", requirement.EventType, strings.Join(parts, " ")))
	}
	return UIAcceptanceAudit{
		Name:                      suite.Name,
		Version:                   suite.Version,
		PermissionGaps:            permissionGaps,
		FailingDataChecks:         failingData,
		FailingPerformanceBudgets: failingPerf,
		FailingUsabilityJourneys:  failingUsability,
		IncompleteAuditTrails:     incompleteAudit,
		DocumentationComplete:     suite.DocumentationComplete,
	}
}

func RenderDesignSystemReport(system DesignSystem, audit DesignSystemAudit) string {
	lines := []string{
		"# Design System Report",
		"",
		fmt.Sprintf("- Name: %s", system.Name),
		fmt.Sprintf("- Version: %s", system.Version),
		fmt.Sprintf("- Components: %d", audit.ComponentCount),
		fmt.Sprintf("- Release Ready Components: %d", len(audit.ReleaseReadyComponents)),
		fmt.Sprintf("- Readiness Score: %.1f", audit.ReadinessScore()),
		"",
		"## Token Foundations",
		"",
	}
	counts := audit.TokenCounts
	for _, category := range foundationCategories {
		lines = append(lines, fmt.Sprintf("- %s: %d", category, counts[category]))
	}
	lines = append(lines, "", "## Component Status", "")
	if len(system.Components) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, component := range system.Components {
			lines = append(lines, fmt.Sprintf(
				"- %s: readiness=%s docs=%s a11y=%s states=%s missing_states=%s undefined_tokens=%s",
				component.Name,
				component.Readiness,
				pyBool(component.DocumentationComplete),
				pyBool(len(component.AccessibilityRequirements) > 0),
				joinOrNone(component.StateCoverage()),
				joinOrNone(component.MissingRequiredStates()),
				joinOrNone(audit.UndefinedTokenRefs[component.Name]),
			))
		}
	}
	lines = append(lines,
		"",
		"## Gaps",
		"",
		fmt.Sprintf("- Missing docs: %s", joinOrNone(audit.ComponentsMissingDocs)),
		fmt.Sprintf("- Missing accessibility: %s", joinOrNone(audit.ComponentsMissingAccessibility)),
		fmt.Sprintf("- Missing interaction states: %s", joinOrNone(audit.ComponentsMissingStates)),
		fmt.Sprintf("- Undefined token refs: %s", formatMapList(audit.UndefinedTokenRefs)),
		fmt.Sprintf("- Orphan tokens: %s", joinOrNone(audit.TokenOrphans)),
	)
	return strings.Join(lines, "\n") + "\n"
}

func RenderConsoleTopBarReport(topBar ConsoleTopBar, audit ConsoleTopBarAudit) string {
	lines := []string{
		"# Console Top Bar Report",
		"",
		fmt.Sprintf("- Name: %s", topBar.Name),
		fmt.Sprintf("- Global Search: %s", pyBool(topBar.HasGlobalSearch())),
		fmt.Sprintf("- Environment Switch: %s", joinOrNone(topBar.EnvironmentOptions)),
		fmt.Sprintf("- Time Range Switch: %s", joinOrNone(topBar.TimeRangeOptions)),
		fmt.Sprintf("- Alert Entry: %s", joinOrNone(topBar.AlertChannels)),
		fmt.Sprintf("- Command Trigger: %s", defaultString(topBar.CommandEntry.TriggerLabel, "none")),
		fmt.Sprintf("- Command Shortcut: %s", defaultString(topBar.CommandEntry.Shortcut, "none")),
		fmt.Sprintf("- Command Count: %d", audit.CommandCount),
		fmt.Sprintf("- Release Ready: %s", pyBool(audit.ReleaseReady())),
		"",
		"## Command Palette",
		"",
	}
	if len(topBar.CommandEntry.Commands) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, command := range topBar.CommandEntry.Commands {
			lines = append(lines, fmt.Sprintf("- %s: %s [%s] shortcut=%s", command.ID, command.Title, command.Section, defaultString(command.Shortcut, "none")))
		}
	}
	lines = append(lines,
		"",
		"## Gaps",
		"",
		fmt.Sprintf("- Missing capabilities: %s", joinOrNone(audit.MissingCapabilities)),
		fmt.Sprintf("- Documentation complete: %s", pyBool(audit.DocumentationComplete)),
		fmt.Sprintf("- Accessibility complete: %s", pyBool(audit.AccessibilityComplete)),
		fmt.Sprintf("- Cmd/Ctrl+K supported: %s", pyBool(audit.CommandShortcutSupported)),
	)
	return strings.Join(lines, "\n") + "\n"
}

func RenderInformationArchitectureReport(architecture InformationArchitecture, audit InformationArchitectureAudit) string {
	lines := []string{
		"# Information Architecture Report",
		"",
		fmt.Sprintf("- Navigation Nodes: %d", audit.TotalNavigationNodes),
		fmt.Sprintf("- Routes: %d", audit.TotalRoutes),
		fmt.Sprintf("- Healthy: %s", pyBool(audit.Healthy())),
		"",
		"## Navigation Tree",
		"",
	}
	entries := architecture.NavigationEntries()
	if len(entries) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, entry := range entries {
			indent := strings.Repeat("  ", entry.Depth)
			lines = append(lines, fmt.Sprintf("- %s%s (%s) screen=%s", indent, entry.Title, entry.Path, defaultString(entry.ScreenID, "none")))
		}
	}
	lines = append(lines, "", "## Route Registry", "")
	if len(architecture.Routes) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, route := range architecture.Routes {
			lines = append(lines, fmt.Sprintf("- %s: screen=%s title=%s nav_node=%s", normalizeRoutePath(route.Path), route.ScreenID, route.Title, defaultString(route.NavNodeID, "none")))
		}
	}
	lines = append(lines,
		"",
		"## Audit",
		"",
		fmt.Sprintf("- Duplicate routes: %s", joinOrNone(audit.DuplicateRoutes)),
		fmt.Sprintf("- Missing route nodes: %s", formatMapString(audit.MissingRouteNodes)),
		fmt.Sprintf("- Secondary nav gaps: %s", formatMapList(audit.SecondaryNavGaps)),
		fmt.Sprintf("- Orphan routes: %s", joinOrNone(audit.OrphanRoutes)),
	)
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIAcceptanceReport(suite UIAcceptanceSuite, audit UIAcceptanceAudit) string {
	lines := []string{
		"# UI Acceptance Report",
		"",
		fmt.Sprintf("- Name: %s", suite.Name),
		fmt.Sprintf("- Version: %s", suite.Version),
		fmt.Sprintf("- Role/Permission Scenarios: %d", len(suite.RolePermissions)),
		fmt.Sprintf("- Data Accuracy Checks: %d", len(suite.DataAccuracyChecks)),
		fmt.Sprintf("- Performance Budgets: %d", len(suite.PerformanceBudgets)),
		fmt.Sprintf("- Usability Journeys: %d", len(suite.UsabilityJourneys)),
		fmt.Sprintf("- Audit Requirements: %d", len(suite.AuditRequirements)),
		fmt.Sprintf("- Readiness Score: %.1f", audit.ReadinessScore()),
		fmt.Sprintf("- Release Ready: %s", pyBool(audit.ReleaseReady())),
		"",
		"## Coverage",
		"",
	}
	if len(suite.RolePermissions) == 0 {
		lines = append(lines, "- Role/Permission: none")
	} else {
		for _, scenario := range suite.RolePermissions {
			lines = append(lines, fmt.Sprintf("- Role/Permission %s: allow=%s deny=%s audit_event=%s", scenario.ScreenID, joinOrNone(scenario.AllowedRoles), joinOrNone(scenario.DeniedRoles), defaultString(scenario.AuditEvent, "none")))
		}
	}
	if len(suite.DataAccuracyChecks) == 0 {
		lines = append(lines, "- Data Accuracy: none")
	} else {
		for _, check := range suite.DataAccuracyChecks {
			lines = append(lines, fmt.Sprintf("- Data Accuracy %s.%s: delta=%s tolerance=%s freshness=%d/%ds", check.ScreenID, check.MetricID, formatFloat(check.ObservedDelta), formatFloat(check.Tolerance), check.ObservedFreshnessSeconds, check.FreshnessSLOSeconds))
		}
	}
	if len(suite.PerformanceBudgets) == 0 {
		lines = append(lines, "- Performance: none")
	} else {
		for _, budget := range suite.PerformanceBudgets {
			tti := ""
			if budget.TargetTTIMS > 0 {
				tti = fmt.Sprintf(" tti=%d/%dms", budget.ObservedTTIMS, budget.TargetTTIMS)
			}
			lines = append(lines, fmt.Sprintf("- Performance %s.%s: p95=%d/%dms%s", budget.SurfaceID, budget.Interaction, budget.ObservedP95MS, budget.TargetP95MS, tti))
		}
	}
	if len(suite.UsabilityJourneys) == 0 {
		lines = append(lines, "- Usability: none")
	} else {
		for _, journey := range suite.UsabilityJourneys {
			lines = append(lines, fmt.Sprintf("- Usability %s: steps=%d/%d keyboard=%s empty_state=%s recovery=%s", journey.JourneyID, journey.ObservedSteps, journey.ExpectedMaxSteps, pyBool(journey.KeyboardAccessible), pyBool(journey.EmptyStateGuidance), pyBool(journey.RecoverySupport)))
		}
	}
	if len(suite.AuditRequirements) == 0 {
		lines = append(lines, "- Audit: none")
	} else {
		for _, requirement := range suite.AuditRequirements {
			lines = append(lines, fmt.Sprintf("- Audit %s: fields=%d/%d retention=%d/%dd", requirement.EventType, len(requirement.EmittedFields), len(requirement.RequiredFields), requirement.ObservedRetentionDays, requirement.RetentionDays))
		}
	}
	lines = append(lines,
		"",
		"## Gaps",
		"",
		fmt.Sprintf("- Role/Permission gaps: %s", joinOrNone(audit.PermissionGaps)),
		fmt.Sprintf("- Data accuracy failures: %s", joinOrNone(audit.FailingDataChecks)),
		fmt.Sprintf("- Performance budget failures: %s", joinOrNone(audit.FailingPerformanceBudgets)),
		fmt.Sprintf("- Usability journey failures: %s", joinOrNone(audit.FailingUsabilityJourneys)),
		fmt.Sprintf("- Audit completeness gaps: %s", joinOrNone(audit.IncompleteAuditTrails)),
		fmt.Sprintf("- Documentation complete: %s", pyBool(audit.DocumentationComplete)),
	)
	return strings.Join(lines, "\n") + "\n"
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

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func formatMapList(m map[string][]string) string {
	if len(m) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, strings.Join(m[key], ", ")))
	}
	return strings.Join(parts, "; ")
}

func formatMapString(m map[string]string) string {
	if len(m) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, m[key]))
	}
	return strings.Join(parts, ", ")
}

func pyBool(value bool) string {
	if value {
		return "True"
	}
	return "False"
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func round1(value float64) float64 { return float64(int(value*10+0.5)) / 10 }

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func formatFloat(value float64) string {
	if value == float64(int(value)) {
		return fmt.Sprintf("%d.0", int(value))
	}
	return fmt.Sprintf("%.1f", value)
}
