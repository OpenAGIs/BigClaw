package designsystem

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

var requiredComponentStates = []string{"default", "hover", "disabled"}
var requiredAccessibility = []string{"keyboard-navigation", "screen-reader-label", "focus-visible"}

type DesignToken struct {
	Name         string `json:"name"`
	Category     string `json:"category"`
	Value        string `json:"value"`
	SemanticRole string `json:"semantic_role,omitempty"`
}

type ComponentVariant struct {
	Name       string   `json:"name"`
	Tokens     []string `json:"tokens,omitempty"`
	States     []string `json:"states,omitempty"`
	UsageNotes string   `json:"usage_notes,omitempty"`
}

type ComponentSpec struct {
	Name                      string             `json:"name"`
	Readiness                 string             `json:"readiness"`
	Slots                     []string           `json:"slots,omitempty"`
	DocumentationComplete     bool               `json:"documentation_complete"`
	AccessibilityRequirements []string           `json:"accessibility_requirements,omitempty"`
	Variants                  []ComponentVariant `json:"variants,omitempty"`
}

type DesignSystem struct {
	Name       string          `json:"name"`
	Version    string          `json:"version"`
	Tokens     []DesignToken   `json:"tokens,omitempty"`
	Components []ComponentSpec `json:"components,omitempty"`
}

type DesignSystemAudit struct {
	SystemName                     string              `json:"system_name"`
	Version                        string              `json:"version"`
	TokenCounts                    map[string]int      `json:"token_counts,omitempty"`
	ComponentCount                 int                 `json:"component_count"`
	ReleaseReadyComponents         []string            `json:"release_ready_components,omitempty"`
	ComponentsMissingDocs          []string            `json:"components_missing_docs,omitempty"`
	ComponentsMissingAccessibility []string            `json:"components_missing_accessibility,omitempty"`
	ComponentsMissingStates        []string            `json:"components_missing_states,omitempty"`
	UndefinedTokenRefs             map[string][]string `json:"undefined_token_refs,omitempty"`
	TokenOrphans                   []string            `json:"token_orphans,omitempty"`
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
	RecentQueriesEnabled bool            `json:"recent_queries_enabled,omitempty"`
	Commands             []CommandAction `json:"commands,omitempty"`
}

type ConsoleTopBar struct {
	Name                      string              `json:"name"`
	SearchPlaceholder         string              `json:"search_placeholder"`
	EnvironmentOptions        []string            `json:"environment_options,omitempty"`
	TimeRangeOptions          []string            `json:"time_range_options,omitempty"`
	AlertChannels             []string            `json:"alert_channels,omitempty"`
	DocumentationComplete     bool                `json:"documentation_complete"`
	AccessibilityRequirements []string            `json:"accessibility_requirements,omitempty"`
	CommandEntry              ConsoleCommandEntry `json:"command_entry"`
}

type ConsoleTopBarAudit struct {
	Name                     string   `json:"name"`
	MissingCapabilities      []string `json:"missing_capabilities,omitempty"`
	DocumentationComplete    bool     `json:"documentation_complete"`
	AccessibilityComplete    bool     `json:"accessibility_complete"`
	CommandShortcutSupported bool     `json:"command_shortcut_supported"`
	CommandCount             int      `json:"command_count"`
}

type NavigationNode struct {
	NodeID   string           `json:"node_id"`
	Title    string           `json:"title"`
	Segment  string           `json:"segment"`
	ScreenID string           `json:"screen_id"`
	Children []NavigationNode `json:"children,omitempty"`
}

type NavigationRoute struct {
	Path      string `json:"path"`
	ScreenID  string `json:"screen_id"`
	Title     string `json:"title"`
	NavNodeID string `json:"nav_node_id"`
}

type InformationArchitecture struct {
	GlobalNav []NavigationNode  `json:"global_nav,omitempty"`
	Routes    []NavigationRoute `json:"routes,omitempty"`
}

type InformationArchitectureAudit struct {
	TotalNavigationNodes int                 `json:"total_navigation_nodes"`
	TotalRoutes          int                 `json:"total_routes"`
	DuplicateRoutes      []string            `json:"duplicate_routes,omitempty"`
	MissingRouteNodes    map[string]string   `json:"missing_route_nodes,omitempty"`
	SecondaryNavGaps     map[string][]string `json:"secondary_nav_gaps,omitempty"`
	OrphanRoutes         []string            `json:"orphan_routes,omitempty"`
}

type RolePermissionScenario struct {
	ScreenID     string   `json:"screen_id"`
	AllowedRoles []string `json:"allowed_roles,omitempty"`
	DeniedRoles  []string `json:"denied_roles,omitempty"`
	AuditEvent   string   `json:"audit_event,omitempty"`
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

type PerformanceBudget struct {
	SurfaceID     string `json:"surface_id"`
	Interaction   string `json:"interaction"`
	TargetP95MS   int    `json:"target_p95_ms"`
	ObservedP95MS int    `json:"observed_p95_ms"`
	TargetTTIMS   int    `json:"target_tti_ms"`
	ObservedTTIMS int    `json:"observed_tti_ms"`
}

type UsabilityJourney struct {
	JourneyID          string   `json:"journey_id"`
	Personas           []string `json:"personas,omitempty"`
	CriticalSteps      []string `json:"critical_steps,omitempty"`
	ExpectedMaxSteps   int      `json:"expected_max_steps"`
	ObservedSteps      int      `json:"observed_steps"`
	KeyboardAccessible bool     `json:"keyboard_accessible"`
	EmptyStateGuidance bool     `json:"empty_state_guidance"`
	RecoverySupport    bool     `json:"recovery_support"`
}

type AuditRequirement struct {
	EventType             string   `json:"event_type"`
	RequiredFields        []string `json:"required_fields,omitempty"`
	EmittedFields         []string `json:"emitted_fields,omitempty"`
	RetentionDays         int      `json:"retention_days"`
	ObservedRetentionDays int      `json:"observed_retention_days"`
}

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

type ComponentLibrary struct{}
type ConsoleChromeLibrary struct{}
type UIAcceptanceLibrary struct{}

func (c ComponentSpec) ReleaseReady() bool {
	return strings.EqualFold(c.Readiness, "stable") &&
		c.DocumentationComplete &&
		containsAll(c.AccessibilityRequirements, requiredAccessibility...) &&
		len(c.MissingRequiredStates()) == 0
}

func (c ComponentSpec) TokenNames() []string {
	seen := map[string]struct{}{}
	var out []string
	for _, variant := range c.Variants {
		for _, token := range variant.Tokens {
			if _, ok := seen[token]; ok {
				continue
			}
			seen[token] = struct{}{}
			out = append(out, token)
		}
	}
	return out
}

func (c ComponentSpec) MissingRequiredStates() []string {
	seen := map[string]struct{}{}
	for _, variant := range c.Variants {
		for _, state := range variant.States {
			seen[state] = struct{}{}
		}
	}
	var missing []string
	for _, state := range requiredComponentStates {
		if _, ok := seen[state]; !ok {
			missing = append(missing, state)
		}
	}
	return missing
}

func (lib ComponentLibrary) Audit(system DesignSystem) DesignSystemAudit {
	definedTokens := map[string]struct{}{}
	tokenCounts := map[string]int{}
	tokenUsage := map[string]int{}
	audit := DesignSystemAudit{
		SystemName:         system.Name,
		Version:            system.Version,
		TokenCounts:        tokenCounts,
		ComponentCount:     len(system.Components),
		UndefinedTokenRefs: map[string][]string{},
	}
	for _, token := range system.Tokens {
		definedTokens[token.Name] = struct{}{}
		tokenCounts[token.Category]++
	}
	for _, component := range system.Components {
		if component.ReleaseReady() {
			audit.ReleaseReadyComponents = append(audit.ReleaseReadyComponents, component.Name)
		}
		if !component.DocumentationComplete {
			audit.ComponentsMissingDocs = append(audit.ComponentsMissingDocs, component.Name)
		}
		if !containsAll(component.AccessibilityRequirements, requiredAccessibility...) {
			audit.ComponentsMissingAccessibility = append(audit.ComponentsMissingAccessibility, component.Name)
		}
		if len(component.MissingRequiredStates()) > 0 {
			audit.ComponentsMissingStates = append(audit.ComponentsMissingStates, component.Name)
		}
		var undefined []string
		for _, token := range component.TokenNames() {
			if _, ok := definedTokens[token]; !ok {
				undefined = append(undefined, token)
				continue
			}
			tokenUsage[token]++
		}
		if len(undefined) > 0 {
			audit.UndefinedTokenRefs[component.Name] = undefined
		}
	}
	if len(audit.UndefinedTokenRefs) == 0 {
		audit.UndefinedTokenRefs = map[string][]string{}
	}
	for _, token := range system.Tokens {
		if tokenUsage[token.Name] == 0 {
			audit.TokenOrphans = append(audit.TokenOrphans, token.Name)
		}
	}
	return audit
}

func (a DesignSystemAudit) ReadinessScore() float64 {
	score := 100.0
	score -= float64(len(a.ComponentsMissingDocs) * 20)
	score -= float64(len(a.ComponentsMissingAccessibility) * 20)
	score -= float64(len(a.ComponentsMissingStates) * 20)
	score -= float64(len(a.TokenOrphans) * 5)
	if score < 0 {
		return 0
	}
	return score
}

func RenderDesignSystemReport(system DesignSystem, audit DesignSystemAudit) string {
	lines := []string{
		"# Design System Report",
		fmt.Sprintf("- Release Ready Components: %d", len(audit.ReleaseReadyComponents)),
	}
	for _, category := range sortedKeys(audit.TokenCounts) {
		lines = append(lines, fmt.Sprintf("- %s: %d", category, audit.TokenCounts[category]))
	}
	for _, component := range system.Components {
		missingStates := "none"
		if missing := component.MissingRequiredStates(); len(missing) > 0 {
			missingStates = strings.Join(missing, ", ")
		}
		undefined := "none"
		if refs := audit.UndefinedTokenRefs[component.Name]; len(refs) > 0 {
			undefined = strings.Join(refs, ", ")
		}
		lines = append(lines, fmt.Sprintf(
			"- %s: readiness=%s docs=%t a11y=%t states=%s missing_states=%s undefined_tokens=%s",
			component.Name,
			component.Readiness,
			component.DocumentationComplete,
			containsAll(component.AccessibilityRequirements, requiredAccessibility...),
			strings.Join(allStates(component), ", "),
			missingStates,
			undefined,
		))
	}
	lines = append(lines, "- Missing interaction states: "+noneOrJoin(audit.ComponentsMissingStates))
	lines = append(lines, "- Undefined token refs: "+renderStringMap(audit.UndefinedTokenRefs))
	lines = append(lines, "- Orphan tokens: "+noneOrJoin(audit.TokenOrphans))
	return strings.Join(lines, "\n")
}

func (lib ConsoleChromeLibrary) AuditTopBar(topBar ConsoleTopBar) ConsoleTopBarAudit {
	audit := ConsoleTopBarAudit{
		Name:                     topBar.Name,
		DocumentationComplete:    topBar.DocumentationComplete,
		AccessibilityComplete:    containsAll(topBar.AccessibilityRequirements, requiredAccessibility...),
		CommandShortcutSupported: strings.Contains(topBar.CommandEntry.Shortcut, "Cmd+K") && strings.Contains(topBar.CommandEntry.Shortcut, "Ctrl+K"),
		CommandCount:             len(topBar.CommandEntry.Commands),
	}
	if strings.TrimSpace(topBar.SearchPlaceholder) == "" {
		audit.MissingCapabilities = append(audit.MissingCapabilities, "global-search")
	}
	if len(topBar.TimeRangeOptions) < 2 {
		audit.MissingCapabilities = append(audit.MissingCapabilities, "time-range-switch")
	}
	if len(topBar.EnvironmentOptions) < 2 {
		audit.MissingCapabilities = append(audit.MissingCapabilities, "environment-switch")
	}
	if len(topBar.AlertChannels) == 0 {
		audit.MissingCapabilities = append(audit.MissingCapabilities, "alert-entry")
	}
	if strings.TrimSpace(topBar.CommandEntry.TriggerLabel) == "" || strings.TrimSpace(topBar.CommandEntry.Placeholder) == "" {
		audit.MissingCapabilities = append(audit.MissingCapabilities, "command-shell")
	}
	return audit
}

func (a ConsoleTopBarAudit) ReleaseReady() bool {
	return len(a.MissingCapabilities) == 0 && a.DocumentationComplete && a.AccessibilityComplete && a.CommandShortcutSupported
}

func RenderConsoleTopBarReport(topBar ConsoleTopBar, audit ConsoleTopBarAudit) string {
	lines := []string{
		"# Console Top Bar Report",
		fmt.Sprintf("- Command Shortcut: %s", topBar.CommandEntry.Shortcut),
		fmt.Sprintf("- Release Ready: %t", audit.ReleaseReady()),
	}
	for _, command := range topBar.CommandEntry.Commands {
		shortcut := command.Shortcut
		if shortcut == "" {
			shortcut = "none"
		}
		lines = append(lines, fmt.Sprintf("- %s: %s [%s] shortcut=%s", command.ID, command.Title, command.Section, shortcut))
	}
	lines = append(lines, "- Missing capabilities: "+noneOrJoin(audit.MissingCapabilities))
	lines = append(lines, fmt.Sprintf("- Cmd/Ctrl+K supported: %t", audit.CommandShortcutSupported))
	return strings.Join(lines, "\n")
}

func (ia InformationArchitecture) NavigationEntries() []NavigationRoute {
	var entries []NavigationRoute
	var walk func([]NavigationNode, string)
	walk = func(nodes []NavigationNode, prefix string) {
		for _, node := range nodes {
			path := prefix + "/" + strings.TrimPrefix(node.Segment, "/")
			entries = append(entries, NavigationRoute{
				Path:      path,
				ScreenID:  node.ScreenID,
				Title:     node.Title,
				NavNodeID: node.NodeID,
			})
			walk(node.Children, path)
		}
	}
	walk(ia.GlobalNav, "")
	return entries
}

func (ia InformationArchitecture) ResolveRoute(path string) (NavigationRoute, bool) {
	normalized := path
	if !strings.HasPrefix(normalized, "/") {
		normalized = "/" + normalized
	}
	for _, route := range ia.Routes {
		if route.Path == normalized {
			return route, true
		}
	}
	return NavigationRoute{}, false
}

func (ia InformationArchitecture) Audit() InformationArchitectureAudit {
	audit := InformationArchitectureAudit{
		TotalNavigationNodes: countNodes(ia.GlobalNav),
		TotalRoutes:          len(ia.Routes),
		MissingRouteNodes:    map[string]string{},
		SecondaryNavGaps:     map[string][]string{},
	}
	pathCounts := map[string]int{}
	routeByNode := map[string]NavigationRoute{}
	for _, route := range ia.Routes {
		pathCounts[route.Path]++
		routeByNode[route.NavNodeID] = route
	}
	for path, count := range pathCounts {
		if count > 1 {
			audit.DuplicateRoutes = append(audit.DuplicateRoutes, path)
		}
	}
	var walk func([]NavigationNode, string, string)
	walk = func(nodes []NavigationNode, prefix string, topTitle string) {
		for _, node := range nodes {
			path := prefix + "/" + strings.TrimPrefix(node.Segment, "/")
			if _, ok := routeByNode[node.NodeID]; !ok {
				audit.MissingRouteNodes[node.NodeID] = path
				if topTitle == "" {
					audit.SecondaryNavGaps[node.Title] = append(audit.SecondaryNavGaps[node.Title], path)
				}
			}
			nextTitle := topTitle
			if nextTitle == "" {
				nextTitle = node.Title
			}
			walk(node.Children, path, nextTitle)
		}
	}
	walk(ia.GlobalNav, "", "")
	knownNodes := map[string]struct{}{}
	for _, entry := range ia.NavigationEntries() {
		knownNodes[entry.NavNodeID] = struct{}{}
	}
	for _, route := range ia.Routes {
		if _, ok := knownNodes[route.NavNodeID]; !ok {
			audit.OrphanRoutes = append(audit.OrphanRoutes, route.Path)
		}
	}
	if len(audit.MissingRouteNodes) == 0 {
		audit.MissingRouteNodes = map[string]string{}
	}
	if len(audit.SecondaryNavGaps) == 0 {
		audit.SecondaryNavGaps = map[string][]string{}
	}
	return audit
}

func (a InformationArchitectureAudit) Healthy() bool {
	return len(a.DuplicateRoutes) == 0 && len(a.MissingRouteNodes) == 0 && len(a.SecondaryNavGaps) == 0 && len(a.OrphanRoutes) == 0
}

func RenderInformationArchitectureReport(ia InformationArchitecture, audit InformationArchitectureAudit) string {
	lines := []string{
		"# Information Architecture Report",
		fmt.Sprintf("- Healthy: %t", audit.Healthy()),
	}
	for _, entry := range ia.NavigationEntries() {
		lines = append(lines, fmt.Sprintf("- %s (%s) screen=%s", entry.Title, entry.Path, entry.ScreenID))
	}
	for _, route := range ia.Routes {
		lines = append(lines, fmt.Sprintf("- %s: screen=%s title=%s nav_node=%s", route.Path, route.ScreenID, route.Title, route.NavNodeID))
	}
	lines = append(lines, "- Duplicate routes: "+noneOrJoin(audit.DuplicateRoutes))
	lines = append(lines, "- Missing route nodes: "+renderKV(audit.MissingRouteNodes))
	lines = append(lines, "- Secondary nav gaps: "+renderSliceMap(audit.SecondaryNavGaps))
	lines = append(lines, "- Orphan routes: "+noneOrJoin(audit.OrphanRoutes))
	return strings.Join(lines, "\n")
}

func (lib UIAcceptanceLibrary) Audit(suite UIAcceptanceSuite) UIAcceptanceAudit {
	audit := UIAcceptanceAudit{
		Name:                  suite.Name,
		Version:               suite.Version,
		DocumentationComplete: suite.DocumentationComplete,
	}
	for _, item := range suite.RolePermissions {
		var missing []string
		if len(item.DeniedRoles) == 0 {
			missing = append(missing, "denied-roles")
		}
		if strings.TrimSpace(item.AuditEvent) == "" {
			missing = append(missing, "audit-event")
		}
		if len(missing) > 0 {
			audit.PermissionGaps = append(audit.PermissionGaps, fmt.Sprintf("%s: missing=%s", item.ScreenID, strings.Join(missing, ", ")))
		}
	}
	for _, item := range suite.DataAccuracyChecks {
		if item.ObservedDelta > item.Tolerance || item.ObservedFreshnessSeconds > item.FreshnessSLOSeconds {
			audit.FailingDataChecks = append(audit.FailingDataChecks, fmt.Sprintf("%s.%s: delta=%.1f freshness=%ds", item.ScreenID, item.MetricID, item.ObservedDelta, item.ObservedFreshnessSeconds))
		}
	}
	for _, item := range suite.PerformanceBudgets {
		if item.ObservedP95MS > item.TargetP95MS || item.ObservedTTIMS > item.TargetTTIMS {
			audit.FailingPerformanceBudgets = append(audit.FailingPerformanceBudgets, fmt.Sprintf("%s.%s: p95=%dms tti=%dms", item.SurfaceID, item.Interaction, item.ObservedP95MS, item.ObservedTTIMS))
		}
	}
	for _, item := range suite.UsabilityJourneys {
		if item.ObservedSteps > item.ExpectedMaxSteps || !item.KeyboardAccessible || !item.RecoverySupport {
			audit.FailingUsabilityJourneys = append(audit.FailingUsabilityJourneys, fmt.Sprintf("%s: steps=%d/%d", item.JourneyID, item.ObservedSteps, item.ExpectedMaxSteps))
		}
	}
	for _, item := range suite.AuditRequirements {
		var missing []string
		emitted := map[string]struct{}{}
		for _, field := range item.EmittedFields {
			emitted[field] = struct{}{}
		}
		for _, field := range item.RequiredFields {
			if _, ok := emitted[field]; !ok {
				missing = append(missing, field)
			}
		}
		if len(missing) > 0 || item.ObservedRetentionDays < item.RetentionDays {
			audit.IncompleteAuditTrails = append(audit.IncompleteAuditTrails, fmt.Sprintf("%s: missing_fields=%s retention=%d/%dd", item.EventType, strings.Join(missing, ", "), item.ObservedRetentionDays, item.RetentionDays))
		}
	}
	return audit
}

func (a UIAcceptanceAudit) ReadinessScore() float64 {
	if a.ReleaseReady() {
		return 100.0
	}
	return 0.0
}

func (a UIAcceptanceAudit) ReleaseReady() bool {
	return a.DocumentationComplete &&
		len(a.PermissionGaps) == 0 &&
		len(a.FailingDataChecks) == 0 &&
		len(a.FailingPerformanceBudgets) == 0 &&
		len(a.FailingUsabilityJourneys) == 0 &&
		len(a.IncompleteAuditTrails) == 0
}

func RenderUIAcceptanceReport(suite UIAcceptanceSuite, audit UIAcceptanceAudit) string {
	lines := []string{
		"# UI Acceptance Report",
		fmt.Sprintf("- Readiness Score: %.1f", audit.ReadinessScore()),
		fmt.Sprintf("- Release Ready: %t", audit.ReleaseReady()),
	}
	for _, item := range suite.RolePermissions {
		lines = append(lines, fmt.Sprintf("- Role/Permission %s: allow=%s deny=%s audit_event=%s", item.ScreenID, strings.Join(item.AllowedRoles, ", "), strings.Join(item.DeniedRoles, ", "), item.AuditEvent))
	}
	for _, item := range suite.DataAccuracyChecks {
		lines = append(lines, fmt.Sprintf("- Data Accuracy %s.%s: delta=%.1f tolerance=%.1f freshness=%d/%ds", item.ScreenID, item.MetricID, item.ObservedDelta, item.Tolerance, item.ObservedFreshnessSeconds, item.FreshnessSLOSeconds))
	}
	for _, item := range suite.PerformanceBudgets {
		lines = append(lines, fmt.Sprintf("- Performance %s.%s: p95=%d/%dms tti=%d/%dms", item.SurfaceID, item.Interaction, item.ObservedP95MS, item.TargetP95MS, item.ObservedTTIMS, item.TargetTTIMS))
	}
	for _, item := range suite.UsabilityJourneys {
		lines = append(lines, fmt.Sprintf("- Usability %s: steps=%d/%d keyboard=%t empty_state=%t recovery=%t", item.JourneyID, item.ObservedSteps, item.ExpectedMaxSteps, item.KeyboardAccessible, item.EmptyStateGuidance, item.RecoverySupport))
	}
	lines = append(lines, "- Audit completeness gaps: "+noneOrJoin(audit.IncompleteAuditTrails))
	return strings.Join(lines, "\n")
}

func DeepCopy[T any](value T) (T, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		var zero T
		return zero, err
	}
	var out T
	if err := json.Unmarshal(payload, &out); err != nil {
		var zero T
		return zero, err
	}
	return out, nil
}

func containsAll(items []string, required ...string) bool {
	seen := map[string]struct{}{}
	for _, item := range items {
		seen[item] = struct{}{}
	}
	for _, item := range required {
		if _, ok := seen[item]; !ok {
			return false
		}
	}
	return true
}

func allStates(component ComponentSpec) []string {
	seen := map[string]struct{}{}
	var states []string
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

func noneOrJoin(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}

func renderStringMap(items map[string][]string) string {
	if len(items) == 0 {
		return "none"
	}
	var keys []string
	for key := range items {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	var parts []string
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, strings.Join(items[key], ", ")))
	}
	return strings.Join(parts, "; ")
}

func renderKV(items map[string]string) string {
	if len(items) == 0 {
		return "none"
	}
	var keys []string
	for key := range items {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	var parts []string
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, items[key]))
	}
	return strings.Join(parts, ", ")
}

func renderSliceMap(items map[string][]string) string {
	if len(items) == 0 {
		return "none"
	}
	var keys []string
	for key := range items {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	var parts []string
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, strings.Join(items[key], ", ")))
	}
	return strings.Join(parts, ", ")
}

func countNodes(nodes []NavigationNode) int {
	total := 0
	for _, node := range nodes {
		total++
		total += countNodes(node.Children)
	}
	return total
}

func sortedKeys(items map[string]int) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
