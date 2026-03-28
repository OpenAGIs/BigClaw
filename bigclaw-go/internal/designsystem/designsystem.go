package designsystem

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
)

var (
	foundationCategories      = []string{"color", "spacing", "typography", "motion", "radius"}
	componentReadinessOrder   = map[string]int{"draft": 0, "alpha": 1, "beta": 2, "stable": 3}
	requiredInteractionStates = []string{"default", "disabled", "hover"}
	requiredTopBarA11y        = []string{"keyboard-navigation", "screen-reader-label", "focus-visible"}
	requiredShortcuts         = []string{"cmd+k", "ctrl+k"}
)

type DesignToken struct {
	Name         string `json:"name"`
	Category     string `json:"category"`
	Value        string `json:"value"`
	SemanticRole string `json:"semantic_role,omitempty"`
	Theme        string `json:"theme,omitempty"`
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
	out := make([]string, 0)
	for _, variant := range c.Variants {
		for _, token := range variant.Tokens {
			if !slices.Contains(out, token) {
				out = append(out, token)
			}
		}
	}
	return out
}

func (c ComponentSpec) StateCoverage() []string {
	out := make([]string, 0)
	for _, variant := range c.Variants {
		for _, state := range variant.States {
			if !slices.Contains(out, state) {
				out = append(out, state)
			}
		}
	}
	return out
}

func (c ComponentSpec) MissingRequiredStates() []string {
	coverage := c.StateCoverage()
	missing := make([]string, 0)
	for _, state := range requiredInteractionStates {
		if !slices.Contains(coverage, state) {
			missing = append(missing, state)
		}
	}
	return missing
}

func (c ComponentSpec) ReleaseReady() bool {
	return componentReadinessOrder[strings.TrimSpace(c.Readiness)] >= componentReadinessOrder["beta"] &&
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

func (d DesignSystem) TokenCounts() map[string]int {
	counts := make(map[string]int, len(foundationCategories))
	for _, category := range foundationCategories {
		counts[category] = 0
	}
	for _, token := range d.Tokens {
		counts[token.Category]++
	}
	return counts
}

func (d DesignSystem) TokenIndex() map[string]DesignToken {
	index := make(map[string]DesignToken, len(d.Tokens))
	for _, token := range d.Tokens {
		index[token.Name] = token
	}
	return index
}

type DesignSystemAudit struct {
	SystemName                     string              `json:"system_name"`
	Version                        string              `json:"version"`
	TokenCountsByCategory          map[string]int      `json:"token_counts"`
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
	penalties := len(a.ComponentsMissingDocs) + len(a.ComponentsMissingAccessibility) + len(a.ComponentsMissingStates)
	score := math.Max(0, (float64(len(a.ReleaseReadyComponents))*100-float64(penalties)*10)/float64(a.ComponentCount))
	return math.Round(score*10) / 10
}

type ComponentLibrary struct{}

func (ComponentLibrary) Audit(system DesignSystem) DesignSystemAudit {
	usedTokens := make(map[string]struct{})
	releaseReady := make([]string, 0)
	missingDocs := make([]string, 0)
	missingAccessibility := make([]string, 0)
	missingStates := make([]string, 0)
	undefined := make(map[string][]string)
	tokenIndex := system.TokenIndex()

	for _, component := range system.Components {
		tokens := component.TokenNames()
		for _, token := range tokens {
			usedTokens[token] = struct{}{}
		}
		missing := make([]string, 0)
		for _, token := range tokens {
			if _, ok := tokenIndex[token]; !ok {
				missing = append(missing, token)
			}
		}
		if len(missing) > 0 {
			sort.Strings(missing)
			undefined[component.Name] = missing
		}
		if component.ReleaseReady() && len(missing) == 0 {
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
	sort.Strings(releaseReady)
	sort.Strings(missingDocs)
	sort.Strings(missingAccessibility)
	sort.Strings(missingStates)
	tokenOrphans := make([]string, 0)
	for _, token := range system.Tokens {
		if _, ok := usedTokens[token.Name]; !ok {
			tokenOrphans = append(tokenOrphans, token.Name)
		}
	}
	sort.Strings(tokenOrphans)
	return DesignSystemAudit{
		SystemName:                     system.Name,
		Version:                        system.Version,
		TokenCountsByCategory:          system.TokenCounts(),
		ComponentCount:                 len(system.Components),
		ReleaseReadyComponents:         releaseReady,
		ComponentsMissingDocs:          missingDocs,
		ComponentsMissingAccessibility: missingAccessibility,
		ComponentsMissingStates:        missingStates,
		UndefinedTokenRefs:             undefined,
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
		lines = append(lines, fmt.Sprintf("- %s: %d", category, audit.TokenCountsByCategory[category]))
	}
	lines = append(lines, "", "## Component Status", "")
	if len(system.Components) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, component := range system.Components {
			states := joinedOrNone(component.StateCoverage())
			missing := joinedOrNone(component.MissingRequiredStates())
			undefined := joinedOrNone(audit.UndefinedTokenRefs[component.Name])
			lines = append(lines,
				fmt.Sprintf("- %s: readiness=%s docs=%t a11y=%t states=%s missing_states=%s undefined_tokens=%s",
					component.Name, component.Readiness, component.DocumentationComplete, len(component.AccessibilityRequirements) > 0, states, missing, undefined))
		}
	}
	lines = append(lines, "", "## Gaps", "")
	lines = append(lines, fmt.Sprintf("- Missing docs: %s", joinedOrNone(audit.ComponentsMissingDocs)))
	lines = append(lines, fmt.Sprintf("- Missing accessibility: %s", joinedOrNone(audit.ComponentsMissingAccessibility)))
	lines = append(lines, fmt.Sprintf("- Missing interaction states: %s", joinedOrNone(audit.ComponentsMissingStates)))
	if len(audit.UndefinedTokenRefs) == 0 {
		lines = append(lines, "- Undefined token refs: none")
	} else {
		keys := sortedMapKeys(audit.UndefinedTokenRefs)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			parts = append(parts, fmt.Sprintf("%s=%s", key, strings.Join(audit.UndefinedTokenRefs[key], ", ")))
		}
		lines = append(lines, fmt.Sprintf("- Undefined token refs: %s", strings.Join(parts, "; ")))
	}
	lines = append(lines, fmt.Sprintf("- Orphan tokens: %s", joinedOrNone(audit.TokenOrphans)))
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

func (c ConsoleTopBar) HasGlobalSearch() bool      { return strings.TrimSpace(c.SearchPlaceholder) != "" }
func (c ConsoleTopBar) HasEnvironmentSwitch() bool { return len(c.EnvironmentOptions) >= 2 }
func (c ConsoleTopBar) HasTimeRangeSwitch() bool   { return len(c.TimeRangeOptions) >= 2 }
func (c ConsoleTopBar) HasAlertEntry() bool        { return len(c.AlertChannels) > 0 }
func (c ConsoleTopBar) HasCommandShell() bool {
	return strings.TrimSpace(c.CommandEntry.TriggerLabel) != "" && len(c.CommandEntry.Commands) > 0
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
	normalized := make([]string, 0)
	for _, item := range strings.Split(topBar.CommandEntry.Shortcut, "/") {
		item = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(item), " ", ""))
		if item != "" {
			normalized = append(normalized, item)
		}
	}
	accessibilityComplete := true
	for _, requirement := range requiredTopBarA11y {
		if !slices.Contains(topBar.AccessibilityRequirements, requirement) {
			accessibilityComplete = false
			break
		}
	}
	commandShortcutSupported := true
	for _, shortcut := range requiredShortcuts {
		if !slices.Contains(normalized, shortcut) {
			commandShortcutSupported = false
			break
		}
	}
	return ConsoleTopBarAudit{
		Name:                     topBar.Name,
		MissingCapabilities:      missing,
		DocumentationComplete:    topBar.DocumentationComplete,
		AccessibilityComplete:    accessibilityComplete,
		CommandShortcutSupported: commandShortcutSupported,
		CommandCount:             len(topBar.CommandEntry.Commands),
	}
}

func RenderConsoleTopBarReport(topBar ConsoleTopBar, audit ConsoleTopBarAudit) string {
	lines := []string{
		"# Console Top Bar Report",
		"",
		fmt.Sprintf("- Name: %s", topBar.Name),
		fmt.Sprintf("- Global Search: %t", topBar.HasGlobalSearch()),
		fmt.Sprintf("- Environment Switch: %s", joinedOrNone(topBar.EnvironmentOptions)),
		fmt.Sprintf("- Time Range Switch: %s", joinedOrNone(topBar.TimeRangeOptions)),
		fmt.Sprintf("- Alert Entry: %s", joinedOrNone(topBar.AlertChannels)),
		fmt.Sprintf("- Command Trigger: %s", defaultString(topBar.CommandEntry.TriggerLabel, "none")),
		fmt.Sprintf("- Command Shortcut: %s", defaultString(topBar.CommandEntry.Shortcut, "none")),
		fmt.Sprintf("- Command Count: %d", audit.CommandCount),
		fmt.Sprintf("- Release Ready: %t", audit.ReleaseReady()),
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
	lines = append(lines, "", "## Gaps", "")
	lines = append(lines, fmt.Sprintf("- Missing capabilities: %s", joinedOrNone(audit.MissingCapabilities)))
	lines = append(lines, fmt.Sprintf("- Documentation complete: %t", audit.DocumentationComplete))
	lines = append(lines, fmt.Sprintf("- Accessibility complete: %t", audit.AccessibilityComplete))
	lines = append(lines, fmt.Sprintf("- Cmd/Ctrl+K supported: %t", audit.CommandShortcutSupported))
	return strings.Join(lines, "\n") + "\n"
}

type NavigationRoute struct {
	Path      string `json:"path"`
	ScreenID  string `json:"screen_id"`
	Title     string `json:"title"`
	NavNodeID string `json:"nav_node_id,omitempty"`
	Layout    string `json:"layout,omitempty"`
}

func (n NavigationRoute) normalized() NavigationRoute {
	n.Path = normalizeRoutePath(n.Path)
	if strings.TrimSpace(n.Layout) == "" {
		n.Layout = "workspace"
	}
	return n
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

func (a InformationArchitecture) RouteIndex() map[string]NavigationRoute {
	index := make(map[string]NavigationRoute)
	for _, route := range a.Routes {
		route = route.normalized()
		if _, ok := index[route.Path]; !ok {
			index[route.Path] = route
		}
	}
	return index
}

func (a InformationArchitecture) NavigationEntries() []NavigationEntry {
	out := make([]NavigationEntry, 0)
	for _, node := range a.GlobalNav {
		out = append(out, a.flatten(node, "", 0, "")...)
	}
	return out
}

func (a InformationArchitecture) ResolveRoute(path string) (NavigationRoute, bool) {
	route, ok := a.RouteIndex()[normalizeRoutePath(path)]
	return route, ok
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

func (a InformationArchitecture) Audit() InformationArchitectureAudit {
	entries := a.NavigationEntries()
	routeCounts := make(map[string]int)
	for _, route := range a.Routes {
		routeCounts[normalizeRoutePath(route.Path)]++
	}
	duplicates := make([]string, 0)
	for path, count := range routeCounts {
		if count > 1 {
			duplicates = append(duplicates, path)
		}
	}
	sort.Strings(duplicates)
	routeIndex := a.RouteIndex()
	missingNodes := make(map[string]string)
	for _, entry := range entries {
		if _, ok := routeIndex[entry.Path]; !ok {
			missingNodes[entry.NodeID] = entry.Path
		}
	}
	secondaryGaps := make(map[string][]string)
	for _, root := range a.GlobalNav {
		missing := a.missingDescendantPaths(root, "")
		sort.Strings(missing)
		if len(missing) > 0 {
			secondaryGaps[root.Title] = missing
		}
	}
	navPaths := make(map[string]struct{}, len(entries))
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
		DuplicateRoutes:      duplicates,
		MissingRouteNodes:    missingNodes,
		SecondaryNavGaps:     secondaryGaps,
		OrphanRoutes:         orphanRoutes,
	}
}

func RenderInformationArchitectureReport(architecture InformationArchitecture, audit InformationArchitectureAudit) string {
	lines := []string{
		"# Information Architecture Report",
		"",
		fmt.Sprintf("- Navigation Nodes: %d", audit.TotalNavigationNodes),
		fmt.Sprintf("- Routes: %d", audit.TotalRoutes),
		fmt.Sprintf("- Healthy: %t", audit.Healthy()),
		"",
		"## Navigation Tree",
		"",
	}
	entries := architecture.NavigationEntries()
	if len(entries) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, entry := range entries {
			lines = append(lines, fmt.Sprintf("- %s%s (%s) screen=%s", strings.Repeat("  ", entry.Depth), entry.Title, entry.Path, defaultString(entry.ScreenID, "none")))
		}
	}
	lines = append(lines, "", "## Route Registry", "")
	if len(architecture.Routes) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, route := range architecture.Routes {
			route = route.normalized()
			lines = append(lines, fmt.Sprintf("- %s: screen=%s title=%s nav_node=%s", route.Path, route.ScreenID, route.Title, defaultString(route.NavNodeID, "none")))
		}
	}
	lines = append(lines, "", "## Audit", "")
	lines = append(lines, fmt.Sprintf("- Duplicate routes: %s", joinedOrNone(audit.DuplicateRoutes)))
	if len(audit.MissingRouteNodes) == 0 {
		lines = append(lines, "- Missing route nodes: none")
	} else {
		keys := sortedMapKeys(audit.MissingRouteNodes)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			parts = append(parts, fmt.Sprintf("%s=%s", key, audit.MissingRouteNodes[key]))
		}
		lines = append(lines, fmt.Sprintf("- Missing route nodes: %s", strings.Join(parts, ", ")))
	}
	if len(audit.SecondaryNavGaps) == 0 {
		lines = append(lines, "- Secondary nav gaps: none")
	} else {
		keys := sortedMapKeys(audit.SecondaryNavGaps)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			parts = append(parts, fmt.Sprintf("%s=%s", key, strings.Join(audit.SecondaryNavGaps[key], ", ")))
		}
		lines = append(lines, fmt.Sprintf("- Secondary nav gaps: %s", strings.Join(parts, "; ")))
	}
	lines = append(lines, fmt.Sprintf("- Orphan routes: %s", joinedOrNone(audit.OrphanRoutes)))
	return strings.Join(lines, "\n") + "\n"
}

type RolePermissionScenario struct {
	ScreenID     string   `json:"screen_id"`
	AllowedRoles []string `json:"allowed_roles,omitempty"`
	DeniedRoles  []string `json:"denied_roles,omitempty"`
	AuditEvent   string   `json:"audit_event,omitempty"`
}

func (r RolePermissionScenario) MissingCoverage() []string {
	missing := make([]string, 0)
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

func (d DataAccuracyCheck) WithinTolerance() bool {
	return math.Abs(d.ObservedDelta) <= d.Tolerance
}
func (d DataAccuracyCheck) WithinFreshnessSLO() bool {
	return d.FreshnessSLOSeconds <= 0 || d.ObservedFreshnessSeconds <= d.FreshnessSLOSeconds
}
func (d DataAccuracyCheck) Passes() bool {
	return d.WithinTolerance() && d.WithinFreshnessSLO()
}

type PerformanceBudget struct {
	SurfaceID     string `json:"surface_id"`
	Interaction   string `json:"interaction"`
	TargetP95MS   int    `json:"target_p95_ms"`
	ObservedP95MS int    `json:"observed_p95_ms"`
	TargetTTIMS   int    `json:"target_tti_ms,omitempty"`
	ObservedTTIMS int    `json:"observed_tti_ms,omitempty"`
}

func (p PerformanceBudget) WithinBudget() bool {
	return p.ObservedP95MS <= p.TargetP95MS && (p.TargetTTIMS <= 0 || p.ObservedTTIMS <= p.TargetTTIMS)
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

func (u UsabilityJourney) Passes() bool {
	return len(u.Personas) > 0 && len(u.CriticalSteps) > 0 && u.ExpectedMaxSteps > 0 &&
		u.ObservedSteps <= u.ExpectedMaxSteps && u.KeyboardAccessible && u.EmptyStateGuidance && u.RecoverySupport
}

type AuditRequirement struct {
	EventType             string   `json:"event_type"`
	RequiredFields        []string `json:"required_fields,omitempty"`
	EmittedFields         []string `json:"emitted_fields,omitempty"`
	RetentionDays         int      `json:"retention_days,omitempty"`
	ObservedRetentionDays int      `json:"observed_retention_days,omitempty"`
}

func (a AuditRequirement) MissingFields() []string {
	out := make([]string, 0)
	for _, field := range a.RequiredFields {
		if !slices.Contains(a.EmittedFields, field) {
			out = append(out, field)
		}
	}
	sort.Strings(out)
	return out
}
func (a AuditRequirement) RetentionMet() bool {
	return a.RetentionDays <= 0 || a.ObservedRetentionDays >= a.RetentionDays
}
func (a AuditRequirement) Complete() bool {
	return len(a.MissingFields()) == 0 && a.RetentionMet()
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

func (u UIAcceptanceAudit) ReleaseReady() bool {
	return len(u.PermissionGaps) == 0 && len(u.FailingDataChecks) == 0 && len(u.FailingPerformanceBudgets) == 0 &&
		len(u.FailingUsabilityJourneys) == 0 && len(u.IncompleteAuditTrails) == 0 && u.DocumentationComplete
}

func (u UIAcceptanceAudit) ReadinessScore() float64 {
	checks := []bool{
		len(u.PermissionGaps) == 0,
		len(u.FailingDataChecks) == 0,
		len(u.FailingPerformanceBudgets) == 0,
		len(u.FailingUsabilityJourneys) == 0,
		len(u.IncompleteAuditTrails) == 0,
		u.DocumentationComplete,
	}
	passed := 0
	for _, ok := range checks {
		if ok {
			passed++
		}
	}
	return math.Round((float64(passed)/float64(len(checks))*100)*10) / 10
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
		if !check.Passes() {
			failingDataChecks = append(failingDataChecks, fmt.Sprintf("%s.%s: delta=%v freshness=%ds", check.ScreenID, check.MetricID, check.ObservedDelta, check.ObservedFreshnessSeconds))
		}
	}
	failingPerformance := make([]string, 0)
	for _, budget := range suite.PerformanceBudgets {
		if !budget.WithinBudget() {
			item := fmt.Sprintf("%s.%s: p95=%dms", budget.SurfaceID, budget.Interaction, budget.ObservedP95MS)
			if budget.TargetTTIMS > 0 {
				item += fmt.Sprintf(" tti=%dms", budget.ObservedTTIMS)
			}
			failingPerformance = append(failingPerformance, item)
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
		FailingDataChecks:         failingDataChecks,
		FailingPerformanceBudgets: failingPerformance,
		FailingUsabilityJourneys:  failingUsability,
		IncompleteAuditTrails:     incompleteAudit,
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
		fmt.Sprintf("- Release Ready: %t", audit.ReleaseReady()),
		"",
		"## Coverage",
		"",
	}
	if len(suite.RolePermissions) == 0 {
		lines = append(lines, "- Role/Permission: none")
	} else {
		for _, scenario := range suite.RolePermissions {
			lines = append(lines, fmt.Sprintf("- Role/Permission %s: allow=%s deny=%s audit_event=%s",
				scenario.ScreenID, joinedOrNone(scenario.AllowedRoles), joinedOrNone(scenario.DeniedRoles), defaultString(scenario.AuditEvent, "none")))
		}
	}
	if len(suite.DataAccuracyChecks) == 0 {
		lines = append(lines, "- Data Accuracy: none")
	} else {
		for _, check := range suite.DataAccuracyChecks {
			lines = append(lines, fmt.Sprintf("- Data Accuracy %s.%s: delta=%v tolerance=%v freshness=%d/%ds",
				check.ScreenID, check.MetricID, check.ObservedDelta, check.Tolerance, check.ObservedFreshnessSeconds, check.FreshnessSLOSeconds))
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
			lines = append(lines, fmt.Sprintf("- Performance %s.%s: p95=%d/%dms%s",
				budget.SurfaceID, budget.Interaction, budget.ObservedP95MS, budget.TargetP95MS, tti))
		}
	}
	if len(suite.UsabilityJourneys) == 0 {
		lines = append(lines, "- Usability: none")
	} else {
		for _, journey := range suite.UsabilityJourneys {
			lines = append(lines, fmt.Sprintf("- Usability %s: steps=%d/%d keyboard=%t empty_state=%t recovery=%t",
				journey.JourneyID, journey.ObservedSteps, journey.ExpectedMaxSteps, journey.KeyboardAccessible, journey.EmptyStateGuidance, journey.RecoverySupport))
		}
	}
	if len(suite.AuditRequirements) == 0 {
		lines = append(lines, "- Audit: none")
	} else {
		for _, requirement := range suite.AuditRequirements {
			lines = append(lines, fmt.Sprintf("- Audit %s: fields=%d/%d retention=%d/%dd",
				requirement.EventType, len(requirement.EmittedFields), len(requirement.RequiredFields), requirement.ObservedRetentionDays, requirement.RetentionDays))
		}
	}
	lines = append(lines, "", "## Gaps", "")
	lines = append(lines, fmt.Sprintf("- Role/Permission gaps: %s", joinedOrNone(audit.PermissionGaps)))
	lines = append(lines, fmt.Sprintf("- Data accuracy failures: %s", joinedOrNone(audit.FailingDataChecks)))
	lines = append(lines, fmt.Sprintf("- Performance budget failures: %s", joinedOrNone(audit.FailingPerformanceBudgets)))
	lines = append(lines, fmt.Sprintf("- Usability journey failures: %s", joinedOrNone(audit.FailingUsabilityJourneys)))
	lines = append(lines, fmt.Sprintf("- Audit completeness gaps: %s", joinedOrNone(audit.IncompleteAuditTrails)))
	lines = append(lines, fmt.Sprintf("- Documentation complete: %t", audit.DocumentationComplete))
	return strings.Join(lines, "\n") + "\n"
}

func normalizeRoutePath(path string) string {
	stripped := strings.Trim(path, "/")
	if stripped == "" {
		return "/"
	}
	return "/" + stripped
}

func (a InformationArchitecture) flatten(node NavigationNode, parentPath string, depth int, parentID string) []NavigationEntry {
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
		entries = append(entries, a.flatten(child, path, depth+1, node.NodeID)...)
	}
	return entries
}

func (a InformationArchitecture) missingDescendantPaths(node NavigationNode, parentPath string) []string {
	path := joinPath(parentPath, node.Segment)
	missing := make([]string, 0)
	if len(node.Children) > 0 {
		if _, ok := a.RouteIndex()[path]; !ok {
			missing = append(missing, path)
		}
	}
	for _, child := range node.Children {
		missing = append(missing, a.missingDescendantPaths(child, path)...)
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

func joinedOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
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
