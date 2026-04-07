package designsystem

import (
	"fmt"
	"math"
	"slices"
	"strings"
)

var foundationCategories = []string{"color", "spacing", "typography", "motion", "radius"}

var componentReadinessOrder = map[string]int{
	"draft":  0,
	"alpha":  1,
	"beta":   2,
	"stable": 3,
}

var requiredInteractionStates = []string{"default", "hover", "disabled"}

type DesignToken struct {
	Name         string `json:"name"`
	Category     string `json:"category"`
	Value        string `json:"value"`
	SemanticRole string `json:"semantic_role,omitempty"`
	Theme        string `json:"theme,omitempty"`
}

func (token DesignToken) withDefaults() DesignToken {
	if token.Theme == "" {
		token.Theme = "core"
	}
	return token
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

func (component ComponentSpec) withDefaults() ComponentSpec {
	if component.Readiness == "" {
		component.Readiness = "draft"
	}
	return component
}

func (component ComponentSpec) TokenNames() []string {
	names := make([]string, 0)
	seen := map[string]struct{}{}
	for _, variant := range component.Variants {
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

func (component ComponentSpec) StateCoverage() []string {
	states := make([]string, 0)
	seen := map[string]struct{}{}
	for _, variant := range component.Variants {
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

func (component ComponentSpec) MissingRequiredStates() []string {
	coverage := map[string]struct{}{}
	for _, state := range component.StateCoverage() {
		coverage[state] = struct{}{}
	}

	missing := make([]string, 0)
	for _, state := range requiredInteractionStates {
		if _, ok := coverage[state]; ok {
			continue
		}
		missing = append(missing, state)
	}
	slices.Sort(missing)
	return missing
}

func (component ComponentSpec) ReleaseReady() bool {
	component = component.withDefaults()
	return componentReadinessOrder[component.Readiness] >= componentReadinessOrder["beta"] &&
		component.DocumentationComplete &&
		len(component.AccessibilityRequirements) > 0 &&
		len(component.MissingRequiredStates()) == 0
}

type DesignSystem struct {
	Name       string          `json:"name"`
	Version    string          `json:"version"`
	Tokens     []DesignToken   `json:"tokens,omitempty"`
	Components []ComponentSpec `json:"components,omitempty"`
}

func (system DesignSystem) TokenCounts() map[string]int {
	counts := map[string]int{}
	for _, category := range foundationCategories {
		counts[category] = 0
	}
	for _, token := range system.Tokens {
		counts[token.Category]++
	}
	return counts
}

func (system DesignSystem) TokenIndex() map[string]DesignToken {
	index := make(map[string]DesignToken, len(system.Tokens))
	for _, token := range system.Tokens {
		index[token.Name] = token.withDefaults()
	}
	return index
}

type DesignSystemAudit struct {
	SystemName                     string              `json:"system_name"`
	Version                        string              `json:"version"`
	TokenCountsMap                 map[string]int      `json:"token_counts"`
	ComponentCount                 int                 `json:"component_count"`
	ReleaseReadyComponents         []string            `json:"release_ready_components,omitempty"`
	ComponentsMissingDocs          []string            `json:"components_missing_docs,omitempty"`
	ComponentsMissingAccessibility []string            `json:"components_missing_accessibility,omitempty"`
	ComponentsMissingStates        []string            `json:"components_missing_states,omitempty"`
	UndefinedTokenRefs             map[string][]string `json:"undefined_token_refs,omitempty"`
	TokenOrphans                   []string            `json:"token_orphans,omitempty"`
}

func (audit DesignSystemAudit) ReadinessScore() float64 {
	if audit.ComponentCount == 0 {
		return 0
	}
	ready := len(audit.ReleaseReadyComponents)
	penalties := len(audit.ComponentsMissingDocs) + len(audit.ComponentsMissingAccessibility) + len(audit.ComponentsMissingStates)
	score := math.Max(0, ((float64(ready)*100)-(float64(penalties)*10))/float64(audit.ComponentCount))
	return round1(score)
}

type ComponentLibrary struct{}

func (ComponentLibrary) Audit(system DesignSystem) DesignSystemAudit {
	usedTokens := map[string]struct{}{}
	releaseReadyComponents := make([]string, 0)
	componentsMissingDocs := make([]string, 0)
	componentsMissingAccessibility := make([]string, 0)
	componentsMissingStates := make([]string, 0)
	undefinedTokenRefs := map[string][]string{}
	tokenIndex := system.TokenIndex()

	for _, component := range system.Components {
		component = component.withDefaults()
		tokenNames := component.TokenNames()
		for _, token := range tokenNames {
			usedTokens[token] = struct{}{}
		}
		missingTokens := make([]string, 0)
		for _, token := range tokenNames {
			if _, ok := tokenIndex[token]; !ok {
				missingTokens = append(missingTokens, token)
			}
		}
		slices.Sort(missingTokens)
		if len(missingTokens) > 0 {
			undefinedTokenRefs[component.Name] = missingTokens
		}
		if component.ReleaseReady() && len(missingTokens) == 0 {
			releaseReadyComponents = append(releaseReadyComponents, component.Name)
		}
		if !component.DocumentationComplete {
			componentsMissingDocs = append(componentsMissingDocs, component.Name)
		}
		if len(component.AccessibilityRequirements) == 0 {
			componentsMissingAccessibility = append(componentsMissingAccessibility, component.Name)
		}
		if len(component.MissingRequiredStates()) > 0 {
			componentsMissingStates = append(componentsMissingStates, component.Name)
		}
	}

	tokenOrphans := make([]string, 0)
	for _, token := range system.Tokens {
		if _, ok := usedTokens[token.Name]; ok {
			continue
		}
		tokenOrphans = append(tokenOrphans, token.Name)
	}

	slices.Sort(releaseReadyComponents)
	slices.Sort(componentsMissingDocs)
	slices.Sort(componentsMissingAccessibility)
	slices.Sort(componentsMissingStates)
	slices.Sort(tokenOrphans)

	return DesignSystemAudit{
		SystemName:                     system.Name,
		Version:                        system.Version,
		TokenCountsMap:                 system.TokenCounts(),
		ComponentCount:                 len(system.Components),
		ReleaseReadyComponents:         releaseReadyComponents,
		ComponentsMissingDocs:          componentsMissingDocs,
		ComponentsMissingAccessibility: componentsMissingAccessibility,
		ComponentsMissingStates:        componentsMissingStates,
		UndefinedTokenRefs:             undefinedTokenRefs,
		TokenOrphans:                   tokenOrphans,
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

	for _, category := range foundationCategories {
		lines = append(lines, fmt.Sprintf("- %s: %d", category, audit.TokenCountsMap[category]))
	}
	for category, count := range audit.TokenCountsMap {
		if slices.Contains(foundationCategories, category) {
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s: %d", category, count))
	}

	lines = append(lines, "", "## Component Status", "")
	if len(system.Components) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, component := range system.Components {
			component = component.withDefaults()
			states := joinOrNone(component.StateCoverage())
			missingStates := joinOrNone(component.MissingRequiredStates())
			undefinedTokens := joinOrNone(audit.UndefinedTokenRefs[component.Name])
			lines = append(
				lines,
				fmt.Sprintf(
					"- %s: readiness=%s docs=%t a11y=%t states=%s missing_states=%s undefined_tokens=%s",
					component.Name,
					component.Readiness,
					component.DocumentationComplete,
					len(component.AccessibilityRequirements) > 0,
					states,
					missingStates,
					undefinedTokens,
				),
			)
		}
	}

	lines = append(lines, "", "## Gaps", "")
	lines = append(lines, fmt.Sprintf("- Missing docs: %s", joinOrNone(audit.ComponentsMissingDocs)))
	lines = append(lines, fmt.Sprintf("- Missing accessibility: %s", joinOrNone(audit.ComponentsMissingAccessibility)))
	lines = append(lines, fmt.Sprintf("- Missing interaction states: %s", joinOrNone(audit.ComponentsMissingStates)))
	lines = append(lines, fmt.Sprintf("- Undefined token refs: %s", formatUndefinedTokenRefs(audit.UndefinedTokenRefs)))
	lines = append(lines, fmt.Sprintf("- Orphan tokens: %s", joinOrNone(audit.TokenOrphans)))
	return strings.Join(lines, "\n") + "\n"
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

func (topBar ConsoleTopBar) HasGlobalSearch() bool {
	return strings.TrimSpace(topBar.SearchPlaceholder) != ""
}

func (topBar ConsoleTopBar) HasEnvironmentSwitch() bool {
	return len(topBar.EnvironmentOptions) >= 2
}

func (topBar ConsoleTopBar) HasTimeRangeSwitch() bool {
	return len(topBar.TimeRangeOptions) >= 2
}

func (topBar ConsoleTopBar) HasAlertEntry() bool {
	return len(topBar.AlertChannels) > 0
}

func (topBar ConsoleTopBar) HasCommandShell() bool {
	return strings.TrimSpace(topBar.CommandEntry.TriggerLabel) != "" && len(topBar.CommandEntry.Commands) > 0
}

type ConsoleTopBarAudit struct {
	Name                     string   `json:"name"`
	MissingCapabilities      []string `json:"missing_capabilities,omitempty"`
	DocumentationComplete    bool     `json:"documentation_complete,omitempty"`
	AccessibilityComplete    bool     `json:"accessibility_complete,omitempty"`
	CommandShortcutSupported bool     `json:"command_shortcut_supported,omitempty"`
	CommandCount             int      `json:"command_count"`
}

func (audit ConsoleTopBarAudit) ReleaseReady() bool {
	return len(audit.MissingCapabilities) == 0 &&
		audit.DocumentationComplete &&
		audit.AccessibilityComplete &&
		audit.CommandShortcutSupported
}

type ConsoleChromeLibrary struct{}

func (ConsoleChromeLibrary) AuditTopBar(topBar ConsoleTopBar) ConsoleTopBarAudit {
	missingCapabilities := make([]string, 0)
	if !topBar.HasGlobalSearch() {
		missingCapabilities = append(missingCapabilities, "global-search")
	}
	if !topBar.HasTimeRangeSwitch() {
		missingCapabilities = append(missingCapabilities, "time-range-switch")
	}
	if !topBar.HasEnvironmentSwitch() {
		missingCapabilities = append(missingCapabilities, "environment-switch")
	}
	if !topBar.HasAlertEntry() {
		missingCapabilities = append(missingCapabilities, "alert-entry")
	}
	if !topBar.HasCommandShell() {
		missingCapabilities = append(missingCapabilities, "command-shell")
	}

	normalizedShortcuts := map[string]struct{}{}
	for _, item := range strings.Split(topBar.CommandEntry.Shortcut, "/") {
		normalized := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(item)), " ", "")
		if normalized == "" {
			continue
		}
		normalizedShortcuts[normalized] = struct{}{}
	}
	accessibility := sliceSet(topBar.AccessibilityRequirements)

	_, hasCmdK := normalizedShortcuts["cmd+k"]
	_, hasCtrlK := normalizedShortcuts["ctrl+k"]
	_, hasKeyboardNavigation := accessibility["keyboard-navigation"]
	_, hasScreenReaderLabel := accessibility["screen-reader-label"]
	_, hasFocusVisible := accessibility["focus-visible"]

	return ConsoleTopBarAudit{
		Name:                     topBar.Name,
		MissingCapabilities:      missingCapabilities,
		DocumentationComplete:    topBar.DocumentationComplete,
		AccessibilityComplete:    hasKeyboardNavigation && hasScreenReaderLabel && hasFocusVisible,
		CommandShortcutSupported: hasCmdK && hasCtrlK,
		CommandCount:             len(topBar.CommandEntry.Commands),
	}
}

func RenderConsoleTopBarReport(topBar ConsoleTopBar, audit ConsoleTopBarAudit) string {
	lines := []string{
		"# Console Top Bar Report",
		"",
		fmt.Sprintf("- Name: %s", topBar.Name),
		fmt.Sprintf("- Global Search: %s", titleBool(topBar.HasGlobalSearch())),
		fmt.Sprintf("- Environment Switch: %s", joinOrNone(topBar.EnvironmentOptions)),
		fmt.Sprintf("- Time Range Switch: %s", joinOrNone(topBar.TimeRangeOptions)),
		fmt.Sprintf("- Alert Entry: %s", joinOrNone(topBar.AlertChannels)),
		fmt.Sprintf("- Command Trigger: %s", noneIfEmpty(topBar.CommandEntry.TriggerLabel)),
		fmt.Sprintf("- Command Shortcut: %s", noneIfEmpty(topBar.CommandEntry.Shortcut)),
		fmt.Sprintf("- Command Count: %d", audit.CommandCount),
		fmt.Sprintf("- Release Ready: %s", titleBool(audit.ReleaseReady())),
		"",
		"## Command Palette",
		"",
	}
	if len(topBar.CommandEntry.Commands) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, command := range topBar.CommandEntry.Commands {
			lines = append(lines, fmt.Sprintf("- %s: %s [%s] shortcut=%s", command.ID, command.Title, command.Section, noneIfEmpty(command.Shortcut)))
		}
	}

	lines = append(lines, "", "## Gaps", "")
	lines = append(lines, fmt.Sprintf("- Missing capabilities: %s", joinOrNone(audit.MissingCapabilities)))
	lines = append(lines, fmt.Sprintf("- Documentation complete: %s", titleBool(audit.DocumentationComplete)))
	lines = append(lines, fmt.Sprintf("- Accessibility complete: %s", titleBool(audit.AccessibilityComplete)))
	lines = append(lines, fmt.Sprintf("- Cmd/Ctrl+K supported: %s", titleBool(audit.CommandShortcutSupported)))
	return strings.Join(lines, "\n") + "\n"
}

type NavigationRoute struct {
	Path      string `json:"path"`
	ScreenID  string `json:"screen_id"`
	Title     string `json:"title"`
	NavNodeID string `json:"nav_node_id,omitempty"`
	Layout    string `json:"layout,omitempty"`
}

func (route NavigationRoute) withDefaults() NavigationRoute {
	route.Path = normalizeRoutePath(route.Path)
	if route.Layout == "" {
		route.Layout = "workspace"
	}
	return route
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

type InformationArchitecture struct {
	GlobalNav []NavigationNode  `json:"global_nav,omitempty"`
	Routes    []NavigationRoute `json:"routes,omitempty"`
}

func (architecture InformationArchitecture) RouteIndex() map[string]NavigationRoute {
	index := map[string]NavigationRoute{}
	for _, route := range architecture.Routes {
		route = route.withDefaults()
		if _, ok := index[route.Path]; ok {
			continue
		}
		index[route.Path] = route
	}
	return index
}

func (architecture InformationArchitecture) NavigationEntries() []NavigationEntry {
	entries := make([]NavigationEntry, 0)
	for _, node := range architecture.GlobalNav {
		entries = append(entries, architecture.flattenNode(node, "", 0, "")...)
	}
	return entries
}

func (architecture InformationArchitecture) ResolveRoute(path string) *NavigationRoute {
	route, ok := architecture.RouteIndex()[normalizeRoutePath(path)]
	if !ok {
		return nil
	}
	return &route
}

type InformationArchitectureAudit struct {
	TotalNavigationNodes int                 `json:"total_navigation_nodes"`
	TotalRoutes          int                 `json:"total_routes"`
	DuplicateRoutes      []string            `json:"duplicate_routes,omitempty"`
	MissingRouteNodes    map[string]string   `json:"missing_route_nodes,omitempty"`
	SecondaryNavGaps     map[string][]string `json:"secondary_nav_gaps,omitempty"`
	OrphanRoutes         []string            `json:"orphan_routes,omitempty"`
}

func (audit InformationArchitectureAudit) Healthy() bool {
	return len(audit.DuplicateRoutes) == 0 &&
		len(audit.MissingRouteNodes) == 0 &&
		len(audit.SecondaryNavGaps) == 0 &&
		len(audit.OrphanRoutes) == 0
}

func (architecture InformationArchitecture) Audit() InformationArchitectureAudit {
	entries := architecture.NavigationEntries()
	routeCounts := map[string]int{}
	for _, route := range architecture.Routes {
		routeCounts[route.withDefaults().Path]++
	}

	duplicateRoutes := make([]string, 0)
	for path, count := range routeCounts {
		if count > 1 {
			duplicateRoutes = append(duplicateRoutes, path)
		}
	}
	slices.Sort(duplicateRoutes)

	routeIndex := architecture.RouteIndex()
	missingRouteNodes := map[string]string{}
	for _, entry := range entries {
		if _, ok := routeIndex[entry.Path]; ok {
			continue
		}
		missingRouteNodes[entry.NodeID] = entry.Path
	}

	secondaryNavGaps := map[string][]string{}
	for _, root := range architecture.GlobalNav {
		gaps := architecture.missingPathsForDescendants(root, "")
		if len(gaps) == 0 {
			continue
		}
		slices.Sort(gaps)
		secondaryNavGaps[root.Title] = gaps
	}

	navPaths := map[string]struct{}{}
	for _, entry := range entries {
		navPaths[entry.Path] = struct{}{}
	}

	orphanRoutes := make([]string, 0)
	for _, route := range architecture.Routes {
		path := route.withDefaults().Path
		if _, ok := navPaths[path]; ok {
			continue
		}
		orphanRoutes = append(orphanRoutes, path)
	}
	slices.Sort(orphanRoutes)

	return InformationArchitectureAudit{
		TotalNavigationNodes: len(entries),
		TotalRoutes:          len(architecture.Routes),
		DuplicateRoutes:      duplicateRoutes,
		MissingRouteNodes:    missingRouteNodes,
		SecondaryNavGaps:     secondaryNavGaps,
		OrphanRoutes:         orphanRoutes,
	}
}

func (architecture InformationArchitecture) flattenNode(node NavigationNode, parentPath string, depth int, parentID string) []NavigationEntry {
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
		entries = append(entries, architecture.flattenNode(child, path, depth+1, node.NodeID)...)
	}
	return entries
}

func (architecture InformationArchitecture) missingPathsForDescendants(node NavigationNode, parentPath string) []string {
	path := joinPath(parentPath, node.Segment)
	missing := make([]string, 0)
	if len(node.Children) > 0 {
		if _, ok := architecture.RouteIndex()[path]; !ok {
			missing = append(missing, path)
		}
	}
	for _, child := range node.Children {
		missing = append(missing, architecture.missingPathsForDescendants(child, path)...)
	}
	return missing
}

func RenderInformationArchitectureReport(architecture InformationArchitecture, audit InformationArchitectureAudit) string {
	lines := []string{
		"# Information Architecture Report",
		"",
		fmt.Sprintf("- Navigation Nodes: %d", audit.TotalNavigationNodes),
		fmt.Sprintf("- Routes: %d", audit.TotalRoutes),
		fmt.Sprintf("- Healthy: %s", titleBool(audit.Healthy())),
		"",
		"## Navigation Tree",
		"",
	}

	entries := architecture.NavigationEntries()
	if len(entries) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, entry := range entries {
			lines = append(lines, fmt.Sprintf("- %s%s (%s) screen=%s", strings.Repeat("  ", entry.Depth), entry.Title, entry.Path, noneIfEmpty(entry.ScreenID)))
		}
	}

	lines = append(lines, "", "## Route Registry", "")
	if len(architecture.Routes) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, route := range architecture.Routes {
			route = route.withDefaults()
			lines = append(lines, fmt.Sprintf("- %s: screen=%s title=%s nav_node=%s", route.Path, route.ScreenID, route.Title, noneIfEmpty(route.NavNodeID)))
		}
	}

	lines = append(lines, "", "## Audit", "")
	lines = append(lines, fmt.Sprintf("- Duplicate routes: %s", joinOrNone(audit.DuplicateRoutes)))
	lines = append(lines, fmt.Sprintf("- Missing route nodes: %s", formatStringMap(audit.MissingRouteNodes)))
	lines = append(lines, fmt.Sprintf("- Secondary nav gaps: %s", formatStringSlicesMap(audit.SecondaryNavGaps)))
	lines = append(lines, fmt.Sprintf("- Orphan routes: %s", joinOrNone(audit.OrphanRoutes)))
	return strings.Join(lines, "\n") + "\n"
}

type RolePermissionScenario struct {
	ScreenID     string   `json:"screen_id"`
	AllowedRoles []string `json:"allowed_roles,omitempty"`
	DeniedRoles  []string `json:"denied_roles,omitempty"`
	AuditEvent   string   `json:"audit_event,omitempty"`
}

func (scenario RolePermissionScenario) MissingCoverage() []string {
	missing := make([]string, 0)
	if len(scenario.AllowedRoles) == 0 {
		missing = append(missing, "allowed-roles")
	}
	if len(scenario.DeniedRoles) == 0 {
		missing = append(missing, "denied-roles")
	}
	if strings.TrimSpace(scenario.AuditEvent) == "" {
		missing = append(missing, "audit-event")
	}
	return missing
}

type DataAccuracyCheck struct {
	ScreenID                 string  `json:"screen_id"`
	MetricID                 string  `json:"metric_id"`
	SourceOfTruth            string  `json:"source_of_truth"`
	RenderedValue            string  `json:"rendered_value"`
	Tolerance                float64 `json:"tolerance"`
	ObservedDelta            float64 `json:"observed_delta"`
	FreshnessSLOSeconds      int     `json:"freshness_slo_seconds"`
	ObservedFreshnessSeconds int     `json:"observed_freshness_seconds"`
}

func (check DataAccuracyCheck) WithinTolerance() bool {
	return math.Abs(check.ObservedDelta) <= check.Tolerance
}

func (check DataAccuracyCheck) WithinFreshnessSLO() bool {
	if check.FreshnessSLOSeconds <= 0 {
		return true
	}
	return check.ObservedFreshnessSeconds <= check.FreshnessSLOSeconds
}

func (check DataAccuracyCheck) Passes() bool {
	return check.WithinTolerance() && check.WithinFreshnessSLO()
}

type PerformanceBudget struct {
	SurfaceID     string `json:"surface_id"`
	Interaction   string `json:"interaction"`
	TargetP95MS   int    `json:"target_p95_ms"`
	ObservedP95MS int    `json:"observed_p95_ms"`
	TargetTTIMS   int    `json:"target_tti_ms"`
	ObservedTTIMS int    `json:"observed_tti_ms"`
}

func (budget PerformanceBudget) WithinBudget() bool {
	p95OK := budget.ObservedP95MS <= budget.TargetP95MS
	ttiOK := budget.TargetTTIMS <= 0 || budget.ObservedTTIMS <= budget.TargetTTIMS
	return p95OK && ttiOK
}

type UsabilityJourney struct {
	JourneyID          string   `json:"journey_id"`
	Personas           []string `json:"personas,omitempty"`
	CriticalSteps      []string `json:"critical_steps,omitempty"`
	ExpectedMaxSteps   int      `json:"expected_max_steps"`
	ObservedSteps      int      `json:"observed_steps"`
	KeyboardAccessible bool     `json:"keyboard_accessible,omitempty"`
	EmptyStateGuidance bool     `json:"empty_state_guidance,omitempty"`
	RecoverySupport    bool     `json:"recovery_support,omitempty"`
}

func (journey UsabilityJourney) Passes() bool {
	return len(journey.Personas) > 0 &&
		len(journey.CriticalSteps) > 0 &&
		journey.ExpectedMaxSteps > 0 &&
		journey.ObservedSteps <= journey.ExpectedMaxSteps &&
		journey.KeyboardAccessible &&
		journey.EmptyStateGuidance &&
		journey.RecoverySupport
}

type AuditRequirement struct {
	EventType             string   `json:"event_type"`
	RequiredFields        []string `json:"required_fields,omitempty"`
	EmittedFields         []string `json:"emitted_fields,omitempty"`
	RetentionDays         int      `json:"retention_days"`
	ObservedRetentionDays int      `json:"observed_retention_days"`
}

func (requirement AuditRequirement) MissingFields() []string {
	emitted := sliceSet(requirement.EmittedFields)
	missing := make([]string, 0)
	for _, field := range requirement.RequiredFields {
		if _, ok := emitted[field]; ok {
			continue
		}
		missing = append(missing, field)
	}
	slices.Sort(missing)
	return missing
}

func (requirement AuditRequirement) RetentionMet() bool {
	if requirement.RetentionDays <= 0 {
		return true
	}
	return requirement.ObservedRetentionDays >= requirement.RetentionDays
}

func (requirement AuditRequirement) Complete() bool {
	return len(requirement.MissingFields()) == 0 && requirement.RetentionMet()
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

func (audit UIAcceptanceAudit) ReleaseReady() bool {
	return len(audit.PermissionGaps) == 0 &&
		len(audit.FailingDataChecks) == 0 &&
		len(audit.FailingPerformanceBudgets) == 0 &&
		len(audit.FailingUsabilityJourneys) == 0 &&
		len(audit.IncompleteAuditTrails) == 0 &&
		audit.DocumentationComplete
}

func (audit UIAcceptanceAudit) ReadinessScore() float64 {
	checks := []bool{
		len(audit.PermissionGaps) == 0,
		len(audit.FailingDataChecks) == 0,
		len(audit.FailingPerformanceBudgets) == 0,
		len(audit.FailingUsabilityJourneys) == 0,
		len(audit.IncompleteAuditTrails) == 0,
		audit.DocumentationComplete,
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
	permissionGaps := make([]string, 0)
	for _, scenario := range suite.RolePermissions {
		if missing := scenario.MissingCoverage(); len(missing) > 0 {
			permissionGaps = append(permissionGaps, fmt.Sprintf("%s: missing=%s", scenario.ScreenID, strings.Join(missing, ", ")))
		}
	}

	failingDataChecks := make([]string, 0)
	for _, check := range suite.DataAccuracyChecks {
		if check.Passes() {
			continue
		}
		failingDataChecks = append(failingDataChecks, fmt.Sprintf("%s.%s: delta=%v freshness=%ds", check.ScreenID, check.MetricID, check.ObservedDelta, check.ObservedFreshnessSeconds))
	}

	failingPerformanceBudgets := make([]string, 0)
	for _, budget := range suite.PerformanceBudgets {
		if budget.WithinBudget() {
			continue
		}
		line := fmt.Sprintf("%s.%s: p95=%dms", budget.SurfaceID, budget.Interaction, budget.ObservedP95MS)
		if budget.TargetTTIMS > 0 {
			line += fmt.Sprintf(" tti=%dms", budget.ObservedTTIMS)
		}
		failingPerformanceBudgets = append(failingPerformanceBudgets, line)
	}

	failingUsabilityJourneys := make([]string, 0)
	for _, journey := range suite.UsabilityJourneys {
		if journey.Passes() {
			continue
		}
		failingUsabilityJourneys = append(failingUsabilityJourneys, fmt.Sprintf("%s: steps=%d/%d", journey.JourneyID, journey.ObservedSteps, journey.ExpectedMaxSteps))
	}

	incompleteAuditTrails := make([]string, 0)
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
		incompleteAuditTrails = append(incompleteAuditTrails, fmt.Sprintf("%s: %s", requirement.EventType, strings.Join(parts, " ")))
	}

	return UIAcceptanceAudit{
		Name:                      suite.Name,
		Version:                   suite.Version,
		PermissionGaps:            permissionGaps,
		FailingDataChecks:         failingDataChecks,
		FailingPerformanceBudgets: failingPerformanceBudgets,
		FailingUsabilityJourneys:  failingUsabilityJourneys,
		IncompleteAuditTrails:     incompleteAuditTrails,
		DocumentationComplete:     suite.DocumentationComplete,
	}
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
		fmt.Sprintf("- Release Ready: %s", titleBool(audit.ReleaseReady())),
		"",
		"## Coverage",
		"",
	}

	if len(suite.RolePermissions) == 0 {
		lines = append(lines, "- Role/Permission: none")
	} else {
		for _, scenario := range suite.RolePermissions {
			lines = append(lines, fmt.Sprintf("- Role/Permission %s: allow=%s deny=%s audit_event=%s", scenario.ScreenID, joinOrNone(scenario.AllowedRoles), joinOrNone(scenario.DeniedRoles), noneIfEmpty(scenario.AuditEvent)))
		}
	}

	if len(suite.DataAccuracyChecks) == 0 {
		lines = append(lines, "- Data Accuracy: none")
	} else {
		for _, check := range suite.DataAccuracyChecks {
			lines = append(lines, fmt.Sprintf("- Data Accuracy %s.%s: delta=%v tolerance=%v freshness=%d/%ds", check.ScreenID, check.MetricID, check.ObservedDelta, check.Tolerance, check.ObservedFreshnessSeconds, check.FreshnessSLOSeconds))
		}
	}

	if len(suite.PerformanceBudgets) == 0 {
		lines = append(lines, "- Performance: none")
	} else {
		for _, budget := range suite.PerformanceBudgets {
			ttiText := ""
			if budget.TargetTTIMS > 0 {
				ttiText = fmt.Sprintf(" tti=%d/%dms", budget.ObservedTTIMS, budget.TargetTTIMS)
			}
			lines = append(lines, fmt.Sprintf("- Performance %s.%s: p95=%d/%dms%s", budget.SurfaceID, budget.Interaction, budget.ObservedP95MS, budget.TargetP95MS, ttiText))
		}
	}

	if len(suite.UsabilityJourneys) == 0 {
		lines = append(lines, "- Usability: none")
	} else {
		for _, journey := range suite.UsabilityJourneys {
			lines = append(lines, fmt.Sprintf("- Usability %s: steps=%d/%d keyboard=%s empty_state=%s recovery=%s", journey.JourneyID, journey.ObservedSteps, journey.ExpectedMaxSteps, titleBool(journey.KeyboardAccessible), titleBool(journey.EmptyStateGuidance), titleBool(journey.RecoverySupport)))
		}
	}

	if len(suite.AuditRequirements) == 0 {
		lines = append(lines, "- Audit: none")
	} else {
		for _, requirement := range suite.AuditRequirements {
			lines = append(lines, fmt.Sprintf("- Audit %s: fields=%d/%d retention=%d/%dd", requirement.EventType, len(requirement.EmittedFields), len(requirement.RequiredFields), requirement.ObservedRetentionDays, requirement.RetentionDays))
		}
	}

	lines = append(lines, "", "## Gaps", "")
	lines = append(lines, fmt.Sprintf("- Role/Permission gaps: %s", joinOrNone(audit.PermissionGaps)))
	lines = append(lines, fmt.Sprintf("- Data accuracy failures: %s", joinOrNone(audit.FailingDataChecks)))
	lines = append(lines, fmt.Sprintf("- Performance budget failures: %s", joinOrNone(audit.FailingPerformanceBudgets)))
	lines = append(lines, fmt.Sprintf("- Usability journey failures: %s", joinOrNone(audit.FailingUsabilityJourneys)))
	lines = append(lines, fmt.Sprintf("- Audit completeness gaps: %s", joinOrNone(audit.IncompleteAuditTrails)))
	lines = append(lines, fmt.Sprintf("- Documentation complete: %s", titleBool(audit.DocumentationComplete)))
	return strings.Join(lines, "\n") + "\n"
}

func normalizeRoutePath(path string) string {
	stripped := strings.Trim(path, "/")
	if stripped == "" {
		return "/"
	}
	return "/" + stripped
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

func joinOrNone(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}

func noneIfEmpty(value string) string {
	if strings.TrimSpace(value) == "" {
		return "none"
	}
	return value
}

func titleBool(value bool) string {
	if value {
		return "True"
	}
	return "False"
}

func sliceSet(items []string) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, item := range items {
		out[item] = struct{}{}
	}
	return out
}

func formatUndefinedTokenRefs(items map[string][]string) string {
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

func formatStringMap(items map[string]string) string {
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
		parts = append(parts, fmt.Sprintf("%s=%s", key, items[key]))
	}
	return strings.Join(parts, ", ")
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

func round1(value float64) float64 {
	return math.Round(value*10) / 10
}
