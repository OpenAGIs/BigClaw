package designsystemcompat

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

var foundationCategories = []string{"color", "spacing", "typography", "motion", "radius"}
var componentReadinessOrder = map[string]int{"draft": 0, "alpha": 1, "beta": 2, "stable": 3}
var requiredInteractionStates = map[string]struct{}{"default": {}, "hover": {}, "disabled": {}}

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
	names := make([]string, 0)
	for _, variant := range c.Variants {
		for _, token := range variant.Tokens {
			if !contains(names, token) {
				names = append(names, token)
			}
		}
	}
	return names
}

func (c ComponentSpec) StateCoverage() []string {
	states := make([]string, 0)
	for _, variant := range c.Variants {
		for _, state := range variant.States {
			if !contains(states, state) {
				states = append(states, state)
			}
		}
	}
	return states
}

func (c ComponentSpec) MissingRequiredStates() []string {
	missing := make([]string, 0)
	coverage := make(map[string]struct{}, len(c.StateCoverage()))
	for _, state := range c.StateCoverage() {
		coverage[state] = struct{}{}
	}
	for state := range requiredInteractionStates {
		if _, ok := coverage[state]; !ok {
			missing = append(missing, state)
		}
	}
	sort.Strings(missing)
	return missing
}

func (c ComponentSpec) ReleaseReady() bool {
	return componentReadinessOrder[c.Readiness] >= componentReadinessOrder["beta"] &&
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

func (s DesignSystem) ToMap() map[string]any {
	body, _ := json.Marshal(s)
	out := map[string]any{}
	_ = json.Unmarshal(body, &out)
	return out
}

func DesignSystemFromMap(data map[string]any) DesignSystem {
	body, _ := json.Marshal(data)
	var out DesignSystem
	_ = json.Unmarshal(body, &out)
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
		index[token.Name] = token
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
		score = 0
	}
	return round1(score)
}

func (a DesignSystemAudit) ToMap() map[string]any {
	body, _ := json.Marshal(a)
	out := map[string]any{}
	_ = json.Unmarshal(body, &out)
	return out
}

func DesignSystemAuditFromMap(data map[string]any) DesignSystemAudit {
	body, _ := json.Marshal(data)
	var out DesignSystemAudit
	_ = json.Unmarshal(body, &out)
	return out
}

type ComponentLibrary struct{}

func (ComponentLibrary) Audit(system DesignSystem) DesignSystemAudit {
	usedTokens := make(map[string]struct{})
	releaseReady := make([]string, 0)
	missingDocs := make([]string, 0)
	missingA11y := make([]string, 0)
	missingStates := make([]string, 0)
	undefined := make(map[string][]string)
	index := system.TokenIndex()
	for _, component := range system.Components {
		for _, token := range component.TokenNames() {
			usedTokens[token] = struct{}{}
			if _, ok := index[token]; !ok {
				undefined[component.Name] = append(undefined[component.Name], token)
			}
		}
		if component.ReleaseReady() && len(undefined[component.Name]) == 0 {
			releaseReady = append(releaseReady, component.Name)
		}
		if !component.DocumentationComplete {
			missingDocs = append(missingDocs, component.Name)
		}
		if len(component.AccessibilityRequirements) == 0 {
			missingA11y = append(missingA11y, component.Name)
		}
		if len(component.MissingRequiredStates()) > 0 {
			missingStates = append(missingStates, component.Name)
		}
	}
	orphanTokens := make([]string, 0)
	for _, token := range system.Tokens {
		if _, ok := usedTokens[token.Name]; !ok {
			orphanTokens = append(orphanTokens, token.Name)
		}
	}
	sort.Strings(releaseReady)
	sort.Strings(missingDocs)
	sort.Strings(missingA11y)
	sort.Strings(missingStates)
	sort.Strings(orphanTokens)
	return DesignSystemAudit{
		SystemName:                     system.Name,
		Version:                        system.Version,
		TokenCounts:                    system.TokenCounts(),
		ComponentCount:                 len(system.Components),
		ReleaseReadyComponents:         releaseReady,
		ComponentsMissingDocs:          missingDocs,
		ComponentsMissingAccessibility: missingA11y,
		ComponentsMissingStates:        missingStates,
		UndefinedTokenRefs:             undefined,
		TokenOrphans:                   orphanTokens,
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
		lines = append(lines, fmt.Sprintf("- %s: %d", category, audit.TokenCounts[category]))
	}
	lines = append(lines, "", "## Component Status", "")
	if len(system.Components) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, component := range system.Components {
			states := joinOrNone(component.StateCoverage())
			missing := joinOrNone(component.MissingRequiredStates())
			undefined := joinOrNone(audit.UndefinedTokenRefs[component.Name])
			lines = append(lines,
				fmt.Sprintf("- %s: readiness=%s docs=%t a11y=%t states=%s missing_states=%s undefined_tokens=%s",
					component.Name, component.Readiness, component.DocumentationComplete, len(component.AccessibilityRequirements) > 0, states, missing, undefined))
		}
	}
	lines = append(lines, "", "## Gaps", "")
	lines = append(lines, fmt.Sprintf("- Missing docs: %s", joinOrNone(audit.ComponentsMissingDocs)))
	lines = append(lines, fmt.Sprintf("- Missing accessibility: %s", joinOrNone(audit.ComponentsMissingAccessibility)))
	lines = append(lines, fmt.Sprintf("- Missing interaction states: %s", joinOrNone(audit.ComponentsMissingStates)))
	if len(audit.UndefinedTokenRefs) == 0 {
		lines = append(lines, "- Undefined token refs: none")
	} else {
		parts := make([]string, 0, len(audit.UndefinedTokenRefs))
		keys := make([]string, 0, len(audit.UndefinedTokenRefs))
		for key := range audit.UndefinedTokenRefs {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			parts = append(parts, fmt.Sprintf("%s=%s", key, strings.Join(audit.UndefinedTokenRefs[key], ", ")))
		}
		lines = append(lines, "- Undefined token refs: "+strings.Join(parts, "; "))
	}
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

func (t ConsoleTopBar) ToMap() map[string]any {
	body, _ := json.Marshal(t)
	out := map[string]any{}
	_ = json.Unmarshal(body, &out)
	return out
}
func ConsoleTopBarFromMap(data map[string]any) ConsoleTopBar {
	body, _ := json.Marshal(data)
	var out ConsoleTopBar
	_ = json.Unmarshal(body, &out)
	return out
}
func (t ConsoleTopBar) HasGlobalSearch() bool      { return strings.TrimSpace(t.SearchPlaceholder) != "" }
func (t ConsoleTopBar) HasEnvironmentSwitch() bool { return len(t.EnvironmentOptions) >= 2 }
func (t ConsoleTopBar) HasTimeRangeSwitch() bool   { return len(t.TimeRangeOptions) >= 2 }
func (t ConsoleTopBar) HasAlertEntry() bool        { return len(t.AlertChannels) > 0 }
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
	normalized := make(map[string]struct{})
	for _, item := range strings.Split(topBar.CommandEntry.Shortcut, "/") {
		item = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(item), " ", ""))
		if item != "" {
			normalized[item] = struct{}{}
		}
	}
	requiredA11y := []string{"keyboard-navigation", "screen-reader-label", "focus-visible"}
	a11ySet := make(map[string]struct{}, len(topBar.AccessibilityRequirements))
	for _, item := range topBar.AccessibilityRequirements {
		a11ySet[item] = struct{}{}
	}
	accessibilityComplete := true
	for _, item := range requiredA11y {
		if _, ok := a11ySet[item]; !ok {
			accessibilityComplete = false
			break
		}
	}
	_, hasCmdK := normalized["cmd+k"]
	_, hasCtrlK := normalized["ctrl+k"]
	return ConsoleTopBarAudit{
		Name:                     topBar.Name,
		MissingCapabilities:      missing,
		DocumentationComplete:    topBar.DocumentationComplete,
		AccessibilityComplete:    accessibilityComplete,
		CommandShortcutSupported: hasCmdK && hasCtrlK,
		CommandCount:             len(topBar.CommandEntry.Commands),
	}
}

func RenderConsoleTopBarReport(topBar ConsoleTopBar, audit ConsoleTopBarAudit) string {
	lines := []string{
		"# Console Top Bar Report", "",
		fmt.Sprintf("- Name: %s", topBar.Name),
		fmt.Sprintf("- Global Search: %t", topBar.HasGlobalSearch()),
		fmt.Sprintf("- Environment Switch: %s", joinOrNone(topBar.EnvironmentOptions)),
		fmt.Sprintf("- Time Range Switch: %s", joinOrNone(topBar.TimeRangeOptions)),
		fmt.Sprintf("- Alert Entry: %s", joinOrNone(topBar.AlertChannels)),
		fmt.Sprintf("- Command Trigger: %s", firstNonEmpty(topBar.CommandEntry.TriggerLabel, "none")),
		fmt.Sprintf("- Command Shortcut: %s", firstNonEmpty(topBar.CommandEntry.Shortcut, "none")),
		fmt.Sprintf("- Command Count: %d", audit.CommandCount),
		fmt.Sprintf("- Release Ready: %t", audit.ReleaseReady()),
		"", "## Command Palette", "",
	}
	if len(topBar.CommandEntry.Commands) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, cmd := range topBar.CommandEntry.Commands {
			lines = append(lines, fmt.Sprintf("- %s: %s [%s] shortcut=%s", cmd.ID, cmd.Title, cmd.Section, firstNonEmpty(cmd.Shortcut, "none")))
		}
	}
	lines = append(lines, "", "## Gaps", "")
	lines = append(lines, fmt.Sprintf("- Missing capabilities: %s", joinOrNone(audit.MissingCapabilities)))
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

type InformationArchitecture struct {
	GlobalNav []NavigationNode  `json:"global_nav,omitempty"`
	Routes    []NavigationRoute `json:"routes,omitempty"`
}

func (a InformationArchitecture) ToMap() map[string]any {
	body, _ := json.Marshal(a)
	out := map[string]any{}
	_ = json.Unmarshal(body, &out)
	return out
}
func InformationArchitectureFromMap(data map[string]any) InformationArchitecture {
	body, _ := json.Marshal(data)
	var out InformationArchitecture
	_ = json.Unmarshal(body, &out)
	for i := range out.Routes {
		out.Routes[i].Path = normalizeRoutePath(out.Routes[i].Path)
	}
	return out
}

func (a InformationArchitecture) NavigationEntries() []NavigationEntry {
	out := make([]NavigationEntry, 0)
	for _, node := range a.GlobalNav {
		out = append(out, flattenNode(node, "", 0, "")...)
	}
	return out
}

func flattenNode(node NavigationNode, parentPath string, depth int, parentID string) []NavigationEntry {
	path := joinPath(parentPath, node.Segment)
	out := []NavigationEntry{{NodeID: node.NodeID, Title: node.Title, Path: path, Depth: depth, ParentID: parentID, ScreenID: node.ScreenID}}
	for _, child := range node.Children {
		out = append(out, flattenNode(child, path, depth+1, node.NodeID)...)
	}
	return out
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

func (a InformationArchitecture) ResolveRoute(path string) *NavigationRoute {
	want := normalizeRoutePath(path)
	for _, route := range a.Routes {
		if normalizeRoutePath(route.Path) == want {
			r := route
			return &r
		}
	}
	return nil
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
func (a InformationArchitectureAudit) ToMap() map[string]any {
	body, _ := json.Marshal(a)
	out := map[string]any{}
	_ = json.Unmarshal(body, &out)
	return out
}
func InformationArchitectureAuditFromMap(data map[string]any) InformationArchitectureAudit {
	body, _ := json.Marshal(data)
	var out InformationArchitectureAudit
	_ = json.Unmarshal(body, &out)
	return out
}

func (a InformationArchitecture) Audit() InformationArchitectureAudit {
	entries := a.NavigationEntries()
	routeCounts := map[string]int{}
	routeIndex := map[string]NavigationRoute{}
	for _, route := range a.Routes {
		path := normalizeRoutePath(route.Path)
		routeCounts[path]++
		if _, ok := routeIndex[path]; !ok {
			route.Path = path
			routeIndex[path] = route
		}
	}
	dupes := make([]string, 0)
	for path, count := range routeCounts {
		if count > 1 {
			dupes = append(dupes, path)
		}
	}
	sort.Strings(dupes)
	missing := map[string]string{}
	for _, entry := range entries {
		if _, ok := routeIndex[entry.Path]; !ok {
			missing[entry.NodeID] = entry.Path
		}
	}
	secondary := map[string][]string{}
	for _, root := range a.GlobalNav {
		gaps := missingPathsForDescendants(root, "", routeIndex)
		if len(gaps) > 0 {
			secondary[root.Title] = gaps
		}
	}
	navPaths := map[string]struct{}{}
	for _, entry := range entries {
		navPaths[entry.Path] = struct{}{}
	}
	orphan := make([]string, 0)
	for _, route := range a.Routes {
		path := normalizeRoutePath(route.Path)
		if _, ok := navPaths[path]; !ok {
			orphan = append(orphan, path)
		}
	}
	sort.Strings(orphan)
	return InformationArchitectureAudit{TotalNavigationNodes: len(entries), TotalRoutes: len(a.Routes), DuplicateRoutes: dupes, MissingRouteNodes: missing, SecondaryNavGaps: secondary, OrphanRoutes: orphan}
}

func missingPathsForDescendants(node NavigationNode, parentPath string, routeIndex map[string]NavigationRoute) []string {
	path := joinPath(parentPath, node.Segment)
	missing := make([]string, 0)
	if len(node.Children) > 0 {
		if _, ok := routeIndex[path]; !ok {
			missing = append(missing, path)
		}
	}
	for _, child := range node.Children {
		missing = append(missing, missingPathsForDescendants(child, path, routeIndex)...)
	}
	sort.Strings(missing)
	return missing
}

func RenderInformationArchitectureReport(architecture InformationArchitecture, audit InformationArchitectureAudit) string {
	lines := []string{"# Information Architecture Report", "", fmt.Sprintf("- Navigation Nodes: %d", audit.TotalNavigationNodes), fmt.Sprintf("- Routes: %d", audit.TotalRoutes), fmt.Sprintf("- Healthy: %t", audit.Healthy()), "", "## Navigation Tree", ""}
	entries := architecture.NavigationEntries()
	if len(entries) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, entry := range entries {
			indent := strings.Repeat("  ", entry.Depth)
			lines = append(lines, fmt.Sprintf("- %s%s (%s) screen=%s", indent, entry.Title, entry.Path, firstNonEmpty(entry.ScreenID, "none")))
		}
	}
	lines = append(lines, "", "## Route Registry", "")
	if len(architecture.Routes) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, route := range architecture.Routes {
			lines = append(lines, fmt.Sprintf("- %s: screen=%s title=%s nav_node=%s", normalizeRoutePath(route.Path), route.ScreenID, route.Title, firstNonEmpty(route.NavNodeID, "none")))
		}
	}
	lines = append(lines, "", "## Audit", "")
	lines = append(lines, fmt.Sprintf("- Duplicate routes: %s", joinOrNone(audit.DuplicateRoutes)))
	if len(audit.MissingRouteNodes) == 0 {
		lines = append(lines, "- Missing route nodes: none")
	} else {
		keys := sortedKeys(audit.MissingRouteNodes)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			parts = append(parts, fmt.Sprintf("%s=%s", key, audit.MissingRouteNodes[key]))
		}
		lines = append(lines, "- Missing route nodes: "+strings.Join(parts, ", "))
	}
	if len(audit.SecondaryNavGaps) == 0 {
		lines = append(lines, "- Secondary nav gaps: none")
	} else {
		keys := sortedKeys(audit.SecondaryNavGaps)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			parts = append(parts, fmt.Sprintf("%s=%s", key, strings.Join(audit.SecondaryNavGaps[key], ", ")))
		}
		lines = append(lines, "- Secondary nav gaps: "+strings.Join(parts, "; "))
	}
	lines = append(lines, fmt.Sprintf("- Orphan routes: %s", joinOrNone(audit.OrphanRoutes)))
	return strings.Join(lines, "\n") + "\n"
}

type RolePermissionScenario struct {
	ScreenID     string   `json:"screen_id"`
	AllowedRoles []string `json:"allowed_roles,omitempty"`
	DeniedRoles  []string `json:"denied_roles,omitempty"`
	AuditEvent   string   `json:"audit_event,omitempty"`
}

func (s RolePermissionScenario) MissingCoverage() []string {
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

func (c DataAccuracyCheck) Passes() bool {
	return abs(c.ObservedDelta) <= c.Tolerance && (c.FreshnessSLOSeconds <= 0 || c.ObservedFreshnessSeconds <= c.FreshnessSLOSeconds)
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
	return b.ObservedP95MS <= b.TargetP95MS && (b.TargetTTIMS <= 0 || b.ObservedTTIMS <= b.TargetTTIMS)
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
	return len(j.Personas) > 0 && len(j.CriticalSteps) > 0 && j.ExpectedMaxSteps > 0 && j.ObservedSteps <= j.ExpectedMaxSteps && j.KeyboardAccessible && j.EmptyStateGuidance && j.RecoverySupport
}

type AuditRequirement struct {
	EventType             string   `json:"event_type"`
	RequiredFields        []string `json:"required_fields,omitempty"`
	EmittedFields         []string `json:"emitted_fields,omitempty"`
	RetentionDays         int      `json:"retention_days,omitempty"`
	ObservedRetentionDays int      `json:"observed_retention_days,omitempty"`
}

func (r AuditRequirement) MissingFields() []string {
	out := make([]string, 0)
	for _, field := range r.RequiredFields {
		if !contains(r.EmittedFields, field) {
			out = append(out, field)
		}
	}
	sort.Strings(out)
	return out
}
func (r AuditRequirement) RetentionMet() bool {
	return r.RetentionDays <= 0 || r.ObservedRetentionDays >= r.RetentionDays
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
	DocumentationComplete bool                     `json:"documentation_complete,omitempty"`
}

func (s UIAcceptanceSuite) ToMap() map[string]any {
	body, _ := json.Marshal(s)
	out := map[string]any{}
	_ = json.Unmarshal(body, &out)
	return out
}
func UIAcceptanceSuiteFromMap(data map[string]any) UIAcceptanceSuite {
	body, _ := json.Marshal(data)
	var out UIAcceptanceSuite
	_ = json.Unmarshal(body, &out)
	return out
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
	return len(a.PermissionGaps) == 0 && len(a.FailingDataChecks) == 0 && len(a.FailingPerformanceBudgets) == 0 && len(a.FailingUsabilityJourneys) == 0 && len(a.IncompleteAuditTrails) == 0 && a.DocumentationComplete
}
func (a UIAcceptanceAudit) ReadinessScore() float64 {
	checks := []bool{len(a.PermissionGaps) == 0, len(a.FailingDataChecks) == 0, len(a.FailingPerformanceBudgets) == 0, len(a.FailingUsabilityJourneys) == 0, len(a.IncompleteAuditTrails) == 0, a.DocumentationComplete}
	passed := 0
	for _, item := range checks {
		if item {
			passed++
		}
	}
	return round1((float64(passed) / float64(len(checks))) * 100)
}

type UIAcceptanceLibrary struct{}

func (UIAcceptanceLibrary) Audit(suite UIAcceptanceSuite) UIAcceptanceAudit {
	perm := make([]string, 0)
	for _, scenario := range suite.RolePermissions {
		if missing := scenario.MissingCoverage(); len(missing) > 0 {
			perm = append(perm, fmt.Sprintf("%s: missing=%s", scenario.ScreenID, strings.Join(missing, ", ")))
		}
	}
	dataChecks := make([]string, 0)
	for _, check := range suite.DataAccuracyChecks {
		if !check.Passes() {
			dataChecks = append(dataChecks, fmt.Sprintf("%s.%s: delta=%v freshness=%ds", check.ScreenID, check.MetricID, check.ObservedDelta, check.ObservedFreshnessSeconds))
		}
	}
	perf := make([]string, 0)
	for _, budget := range suite.PerformanceBudgets {
		if !budget.WithinBudget() {
			item := fmt.Sprintf("%s.%s: p95=%dms", budget.SurfaceID, budget.Interaction, budget.ObservedP95MS)
			if budget.TargetTTIMS > 0 {
				item += fmt.Sprintf(" tti=%dms", budget.ObservedTTIMS)
			}
			perf = append(perf, item)
		}
	}
	usability := make([]string, 0)
	for _, journey := range suite.UsabilityJourneys {
		if !journey.Passes() {
			usability = append(usability, fmt.Sprintf("%s: steps=%d/%d", journey.JourneyID, journey.ObservedSteps, journey.ExpectedMaxSteps))
		}
	}
	auditTrails := make([]string, 0)
	for _, requirement := range suite.AuditRequirements {
		if requirement.Complete() {
			continue
		}
		parts := make([]string, 0)
		if missing := requirement.MissingFields(); len(missing) > 0 {
			parts = append(parts, "missing_fields="+strings.Join(missing, ", "))
		}
		if !requirement.RetentionMet() {
			parts = append(parts, fmt.Sprintf("retention=%d/%dd", requirement.ObservedRetentionDays, requirement.RetentionDays))
		}
		auditTrails = append(auditTrails, fmt.Sprintf("%s: %s", requirement.EventType, strings.Join(parts, " ")))
	}
	return UIAcceptanceAudit{Name: suite.Name, Version: suite.Version, PermissionGaps: perm, FailingDataChecks: dataChecks, FailingPerformanceBudgets: perf, FailingUsabilityJourneys: usability, IncompleteAuditTrails: auditTrails, DocumentationComplete: suite.DocumentationComplete}
}

func RenderUIAcceptanceReport(suite UIAcceptanceSuite, audit UIAcceptanceAudit) string {
	lines := []string{"# UI Acceptance Report", "", fmt.Sprintf("- Name: %s", suite.Name), fmt.Sprintf("- Version: %s", suite.Version), fmt.Sprintf("- Role/Permission Scenarios: %d", len(suite.RolePermissions)), fmt.Sprintf("- Data Accuracy Checks: %d", len(suite.DataAccuracyChecks)), fmt.Sprintf("- Performance Budgets: %d", len(suite.PerformanceBudgets)), fmt.Sprintf("- Usability Journeys: %d", len(suite.UsabilityJourneys)), fmt.Sprintf("- Audit Requirements: %d", len(suite.AuditRequirements)), fmt.Sprintf("- Readiness Score: %.1f", audit.ReadinessScore()), fmt.Sprintf("- Release Ready: %t", audit.ReleaseReady()), "", "## Coverage", ""}
	if len(suite.RolePermissions) == 0 {
		lines = append(lines, "- Role/Permission: none")
	} else {
		for _, scenario := range suite.RolePermissions {
			lines = append(lines, fmt.Sprintf("- Role/Permission %s: allow=%s deny=%s audit_event=%s", scenario.ScreenID, joinOrNone(scenario.AllowedRoles), joinOrNone(scenario.DeniedRoles), firstNonEmpty(scenario.AuditEvent, "none")))
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
			lines = append(lines, fmt.Sprintf("- Usability %s: steps=%d/%d keyboard=%t empty_state=%t recovery=%t", journey.JourneyID, journey.ObservedSteps, journey.ExpectedMaxSteps, journey.KeyboardAccessible, journey.EmptyStateGuidance, journey.RecoverySupport))
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
	lines = append(lines, fmt.Sprintf("- Documentation complete: %t", audit.DocumentationComplete))
	return strings.Join(lines, "\n") + "\n"
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}
func firstNonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
func round1(value float64) float64 { return float64(int(value*10+0.5)) / 10 }
func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
