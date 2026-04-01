package designsystem

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

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

func (c ComponentSpec) ReleaseReady() bool {
	return strings.EqualFold(c.Readiness, "stable") &&
		c.DocumentationComplete &&
		len(c.AccessibilityRequirements) > 0 &&
		len(c.MissingRequiredStates()) == 0
}

func (c ComponentSpec) TokenNames() []string {
	index := make(map[string]struct{})
	out := make([]string, 0)
	for _, variant := range c.Variants {
		for _, token := range variant.Tokens {
			if _, ok := index[token]; ok {
				continue
			}
			index[token] = struct{}{}
			out = append(out, token)
		}
	}
	sort.Strings(out)
	return out
}

func (c ComponentSpec) MissingRequiredStates() []string {
	required := []string{"default", "hover", "disabled"}
	index := make(map[string]struct{})
	for _, variant := range c.Variants {
		for _, state := range variant.States {
			index[state] = struct{}{}
		}
	}
	missing := make([]string, 0)
	for _, state := range required {
		if _, ok := index[state]; !ok {
			missing = append(missing, state)
		}
	}
	return missing
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
	ReadinessScore                 float64             `json:"readiness_score"`
}

type ComponentLibrary struct{}

func (ComponentLibrary) Audit(system DesignSystem) DesignSystemAudit {
	audit := DesignSystemAudit{
		SystemName:             system.Name,
		Version:                system.Version,
		TokenCounts:            make(map[string]int),
		ComponentCount:         len(system.Components),
		ReleaseReadyComponents: []string{},
		UndefinedTokenRefs:     make(map[string][]string),
	}
	tokenIndex := make(map[string]struct{}, len(system.Tokens))
	tokenUsage := make(map[string]int)
	for _, token := range system.Tokens {
		tokenIndex[token.Name] = struct{}{}
		audit.TokenCounts[token.Category]++
	}
	for _, component := range system.Components {
		if !component.DocumentationComplete {
			audit.ComponentsMissingDocs = append(audit.ComponentsMissingDocs, component.Name)
		}
		if len(component.AccessibilityRequirements) == 0 {
			audit.ComponentsMissingAccessibility = append(audit.ComponentsMissingAccessibility, component.Name)
		}
		if missing := component.MissingRequiredStates(); len(missing) > 0 {
			audit.ComponentsMissingStates = append(audit.ComponentsMissingStates, component.Name)
		}
		undefined := make([]string, 0)
		for _, token := range component.TokenNames() {
			if _, ok := tokenIndex[token]; !ok {
				undefined = append(undefined, token)
			} else {
				tokenUsage[token]++
			}
		}
		if len(undefined) > 0 {
			sort.Strings(undefined)
			audit.UndefinedTokenRefs[component.Name] = undefined
		} else if component.ReleaseReady() {
			audit.ReleaseReadyComponents = append(audit.ReleaseReadyComponents, component.Name)
		}
	}
	for _, token := range system.Tokens {
		if tokenUsage[token.Name] == 0 {
			audit.TokenOrphans = append(audit.TokenOrphans, token.Name)
		}
	}
	sort.Strings(audit.ReleaseReadyComponents)
	sort.Strings(audit.ComponentsMissingDocs)
	sort.Strings(audit.ComponentsMissingAccessibility)
	sort.Strings(audit.ComponentsMissingStates)
	sort.Strings(audit.TokenOrphans)
	if len(audit.UndefinedTokenRefs) == 0 {
		audit.UndefinedTokenRefs = map[string][]string{}
	}
	audit.ReadinessScore = 100.0
	if len(audit.ComponentsMissingDocs) > 0 {
		audit.ReadinessScore -= 25
	}
	if len(audit.ComponentsMissingAccessibility) > 0 {
		audit.ReadinessScore -= 20
	}
	if len(audit.ComponentsMissingStates) > 0 {
		audit.ReadinessScore -= 20
	}
	if audit.ReadinessScore < 0 {
		audit.ReadinessScore = 0
	}
	return audit
}

type ConsoleChromeLibrary struct{}

func (ConsoleChromeLibrary) AuditTopBar(topBar ConsoleTopBar) ConsoleTopBarAudit {
	missing := make([]string, 0)
	if strings.TrimSpace(topBar.SearchPlaceholder) == "" {
		missing = append(missing, "global-search")
	}
	if len(topBar.TimeRangeOptions) < 2 {
		missing = append(missing, "time-range-switch")
	}
	if len(topBar.EnvironmentOptions) < 2 {
		missing = append(missing, "environment-switch")
	}
	if len(topBar.AlertChannels) == 0 {
		missing = append(missing, "alert-entry")
	}
	if strings.TrimSpace(topBar.CommandEntry.TriggerLabel) == "" ||
		strings.TrimSpace(topBar.CommandEntry.Placeholder) == "" ||
		len(topBar.CommandEntry.Commands) == 0 {
		missing = append(missing, "command-shell")
	}
	accessibilityComplete := hasAll(topBar.AccessibilityRequirements, []string{"keyboard-navigation", "screen-reader-label", "focus-visible"})
	commandShortcutSupported := strings.Contains(topBar.CommandEntry.Shortcut, "Cmd+K") && strings.Contains(topBar.CommandEntry.Shortcut, "Ctrl+K")
	return ConsoleTopBarAudit{
		Name:                     topBar.Name,
		MissingCapabilities:      missing,
		DocumentationComplete:    topBar.DocumentationComplete,
		AccessibilityComplete:    accessibilityComplete,
		CommandShortcutSupported: commandShortcutSupported,
		CommandCount:             len(topBar.CommandEntry.Commands),
		ReleaseReady:             len(missing) == 0 && topBar.DocumentationComplete && accessibilityComplete && commandShortcutSupported,
	}
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

func (a InformationArchitecture) NavigationEntries() []NavigationRoute {
	return append([]NavigationRoute(nil), a.Routes...)
}

func (a InformationArchitecture) ResolveRoute(path string) NavigationRoute {
	normalized := path
	if !strings.HasPrefix(normalized, "/") {
		normalized = "/" + normalized
	}
	for _, route := range a.Routes {
		if route.Path == normalized {
			return route
		}
	}
	return NavigationRoute{}
}

type InformationArchitectureAudit struct {
	TotalNavigationNodes int                 `json:"total_navigation_nodes"`
	TotalRoutes          int                 `json:"total_routes"`
	DuplicateRoutes      []string            `json:"duplicate_routes,omitempty"`
	MissingRouteNodes    map[string]string   `json:"missing_route_nodes,omitempty"`
	SecondaryNavGaps     map[string][]string `json:"secondary_nav_gaps,omitempty"`
	OrphanRoutes         []string            `json:"orphan_routes,omitempty"`
	Healthy              bool                `json:"healthy"`
}

func (a InformationArchitecture) Audit() InformationArchitectureAudit {
	audit := InformationArchitectureAudit{
		TotalNavigationNodes: countNavNodes(a.GlobalNav),
		TotalRoutes:          len(a.Routes),
		MissingRouteNodes:    make(map[string]string),
		SecondaryNavGaps:     make(map[string][]string),
	}
	routeCounts := make(map[string]int)
	routePaths := make(map[string]struct{}, len(a.Routes))
	for _, route := range a.Routes {
		routeCounts[route.Path]++
		routePaths[route.Path] = struct{}{}
	}
	for path, count := range routeCounts {
		if count > 1 {
			audit.DuplicateRoutes = append(audit.DuplicateRoutes, path)
		}
	}
	for _, node := range a.GlobalNav {
		markMissingRouteNodes(node, "", routePaths, &audit, true)
	}
	validNodes := flattenNodeIDs(a.GlobalNav)
	for _, route := range a.Routes {
		if _, ok := validNodes[route.NavNodeID]; !ok {
			audit.OrphanRoutes = append(audit.OrphanRoutes, route.Path)
		}
	}
	sort.Strings(audit.DuplicateRoutes)
	sort.Strings(audit.OrphanRoutes)
	audit.Healthy = len(audit.DuplicateRoutes) == 0 && len(audit.MissingRouteNodes) == 0 && len(audit.SecondaryNavGaps) == 0 && len(audit.OrphanRoutes) == 0
	return audit
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
	ReadinessScore            float64  `json:"readiness_score"`
	ReleaseReady              bool     `json:"release_ready"`
}

type UIAcceptanceLibrary struct{}

func (UIAcceptanceLibrary) Audit(suite UIAcceptanceSuite) UIAcceptanceAudit {
	audit := UIAcceptanceAudit{
		Name:                  suite.Name,
		Version:               suite.Version,
		DocumentationComplete: suite.DocumentationComplete,
	}
	for _, scenario := range suite.RolePermissions {
		missing := make([]string, 0)
		if len(scenario.DeniedRoles) == 0 {
			missing = append(missing, "denied-roles")
		}
		if strings.TrimSpace(scenario.AuditEvent) == "" {
			missing = append(missing, "audit-event")
		}
		if len(missing) > 0 {
			audit.PermissionGaps = append(audit.PermissionGaps, fmt.Sprintf("%s: missing=%s", scenario.ScreenID, strings.Join(missing, ", ")))
		}
	}
	for _, check := range suite.DataAccuracyChecks {
		if check.ObservedDelta > check.Tolerance || check.ObservedFreshnessSeconds > check.FreshnessSLOSeconds {
			audit.FailingDataChecks = append(audit.FailingDataChecks, fmt.Sprintf("%s.%s: delta=%.1f freshness=%ds", check.ScreenID, check.MetricID, check.ObservedDelta, check.ObservedFreshnessSeconds))
		}
	}
	for _, budget := range suite.PerformanceBudgets {
		if budget.ObservedP95MS > budget.TargetP95MS || budget.ObservedTTIMS > budget.TargetTTIMS {
			audit.FailingPerformanceBudgets = append(audit.FailingPerformanceBudgets, fmt.Sprintf("%s.%s: p95=%dms tti=%dms", budget.SurfaceID, budget.Interaction, budget.ObservedP95MS, budget.ObservedTTIMS))
		}
	}
	for _, journey := range suite.UsabilityJourneys {
		if journey.ObservedSteps > journey.ExpectedMaxSteps || !journey.KeyboardAccessible || !journey.RecoverySupport {
			audit.FailingUsabilityJourneys = append(audit.FailingUsabilityJourneys, fmt.Sprintf("%s: steps=%d/%d", journey.JourneyID, journey.ObservedSteps, journey.ExpectedMaxSteps))
		}
	}
	for _, requirement := range suite.AuditRequirements {
		missingFields := diffRequiredFields(requirement.RequiredFields, requirement.EmittedFields)
		if len(missingFields) > 0 || requirement.ObservedRetentionDays < requirement.RetentionDays {
			audit.IncompleteAuditTrails = append(audit.IncompleteAuditTrails, fmt.Sprintf("%s: missing_fields=%s retention=%d/%dd", requirement.EventType, strings.Join(missingFields, ", "), requirement.ObservedRetentionDays, requirement.RetentionDays))
		}
	}
	sort.Strings(audit.PermissionGaps)
	sort.Strings(audit.FailingDataChecks)
	sort.Strings(audit.FailingPerformanceBudgets)
	sort.Strings(audit.FailingUsabilityJourneys)
	sort.Strings(audit.IncompleteAuditTrails)
	if len(audit.PermissionGaps) == 0 && len(audit.FailingDataChecks) == 0 && len(audit.FailingPerformanceBudgets) == 0 && len(audit.FailingUsabilityJourneys) == 0 && len(audit.IncompleteAuditTrails) == 0 && suite.DocumentationComplete {
		audit.ReadinessScore = 100.0
		audit.ReleaseReady = true
	} else {
		audit.ReadinessScore = 0.0
		audit.ReleaseReady = false
	}
	return audit
}

func RenderDesignSystemReport(system DesignSystem, audit DesignSystemAudit) string {
	lines := []string{
		"# Design System Report",
		"",
		fmt.Sprintf("- Release Ready Components: %d", len(audit.ReleaseReadyComponents)),
	}
	categories := make([]string, 0, len(audit.TokenCounts))
	for category := range audit.TokenCounts {
		categories = append(categories, category)
	}
	sort.Strings(categories)
	for _, category := range categories {
		lines = append(lines, fmt.Sprintf("- %s: %d", category, audit.TokenCounts[category]))
	}
	for _, component := range system.Components {
		undefined := "none"
		if refs, ok := audit.UndefinedTokenRefs[component.Name]; ok && len(refs) > 0 {
			undefined = strings.Join(refs, ", ")
		}
		lines = append(lines, fmt.Sprintf("- %s: readiness=%s docs=%t a11y=%t states=%s missing_states=%s undefined_tokens=%s",
			component.Name,
			component.Readiness,
			component.DocumentationComplete,
			len(component.AccessibilityRequirements) > 0,
			joinOrNone(component.firstVariantStates()),
			joinOrNone(component.MissingRequiredStates()),
			undefined,
		))
	}
	lines = append(lines, fmt.Sprintf("- Missing interaction states: %s", joinOrNone(audit.ComponentsMissingStates)))
	lines = append(lines, fmt.Sprintf("- Undefined token refs: %s", formatUndefinedTokenRefs(audit.UndefinedTokenRefs)))
	lines = append(lines, fmt.Sprintf("- Orphan tokens: %s", joinOrNone(audit.TokenOrphans)))
	return strings.Join(lines, "\n") + "\n"
}

func RenderConsoleTopBarReport(topBar ConsoleTopBar, audit ConsoleTopBarAudit) string {
	lines := []string{
		"# Console Top Bar Report",
		"",
		fmt.Sprintf("- Command Shortcut: %s", topBar.CommandEntry.Shortcut),
		fmt.Sprintf("- Release Ready: %t", audit.ReleaseReady),
	}
	for _, command := range topBar.CommandEntry.Commands {
		shortcut := ""
		if strings.TrimSpace(command.Shortcut) != "" {
			shortcut = " shortcut=" + command.Shortcut
		}
		lines = append(lines, fmt.Sprintf("- %s: %s [%s]%s", command.ID, command.Title, command.Section, shortcut))
	}
	lines = append(lines, fmt.Sprintf("- Missing capabilities: %s", joinOrNone(audit.MissingCapabilities)))
	lines = append(lines, fmt.Sprintf("- Cmd/Ctrl+K supported: %t", audit.CommandShortcutSupported))
	return strings.Join(lines, "\n") + "\n"
}

func RenderInformationArchitectureReport(architecture InformationArchitecture, audit InformationArchitectureAudit) string {
	lines := []string{
		"# Information Architecture Report",
		"",
		fmt.Sprintf("- Healthy: %t", audit.Healthy),
	}
	for _, node := range architecture.GlobalNav {
		lines = append(lines, renderNavigationNode(node, ""))
	}
	for _, route := range architecture.Routes {
		lines = append(lines, fmt.Sprintf("- %s: screen=%s title=%s nav_node=%s", route.Path, route.ScreenID, route.Title, route.NavNodeID))
	}
	lines = append(lines, fmt.Sprintf("- Duplicate routes: %s", joinOrNone(audit.DuplicateRoutes)))
	lines = append(lines, fmt.Sprintf("- Missing route nodes: %s", formatMissingRouteNodes(audit.MissingRouteNodes)))
	lines = append(lines, fmt.Sprintf("- Secondary nav gaps: %s", formatSecondaryNavGaps(audit.SecondaryNavGaps)))
	lines = append(lines, fmt.Sprintf("- Orphan routes: %s", joinOrNone(audit.OrphanRoutes)))
	return strings.Join(lines, "\n") + "\n"
}

func RenderUIAcceptanceReport(suite UIAcceptanceSuite, audit UIAcceptanceAudit) string {
	lines := []string{
		"# UI Acceptance Report",
		"",
		fmt.Sprintf("- Readiness Score: %.1f", audit.ReadinessScore),
		fmt.Sprintf("- Release Ready: %t", audit.ReleaseReady),
	}
	for _, scenario := range suite.RolePermissions {
		lines = append(lines, fmt.Sprintf("- Role/Permission %s: allow=%s deny=%s audit_event=%s", scenario.ScreenID, joinOrNone(scenario.AllowedRoles), joinOrNone(scenario.DeniedRoles), scenario.AuditEvent))
	}
	for _, check := range suite.DataAccuracyChecks {
		lines = append(lines, fmt.Sprintf("- Data Accuracy %s.%s: delta=%.1f tolerance=%.1f freshness=%d/%ds", check.ScreenID, check.MetricID, check.ObservedDelta, check.Tolerance, check.ObservedFreshnessSeconds, check.FreshnessSLOSeconds))
	}
	for _, budget := range suite.PerformanceBudgets {
		lines = append(lines, fmt.Sprintf("- Performance %s.%s: p95=%d/%dms tti=%d/%dms", budget.SurfaceID, budget.Interaction, budget.ObservedP95MS, budget.TargetP95MS, budget.ObservedTTIMS, budget.TargetTTIMS))
	}
	for _, journey := range suite.UsabilityJourneys {
		lines = append(lines, fmt.Sprintf("- Usability %s: steps=%d/%d keyboard=%t empty_state=%t recovery=%t", journey.JourneyID, journey.ObservedSteps, journey.ExpectedMaxSteps, journey.KeyboardAccessible, journey.EmptyStateGuidance, journey.RecoverySupport))
	}
	lines = append(lines, fmt.Sprintf("- Audit completeness gaps: %s", joinOrNone(audit.IncompleteAuditTrails)))
	return strings.Join(lines, "\n") + "\n"
}

func (c ComponentSpec) firstVariantStates() []string {
	if len(c.Variants) == 0 {
		return nil
	}
	return append([]string(nil), c.Variants[0].States...)
}

func countNavNodes(nodes []NavigationNode) int {
	count := 0
	for _, node := range nodes {
		count++
		count += countNavNodes(node.Children)
	}
	return count
}

func markMissingRouteNodes(node NavigationNode, base string, routePaths map[string]struct{}, audit *InformationArchitectureAudit, topLevel bool) {
	path := base + "/" + node.Segment
	if _, ok := routePaths[path]; !ok {
		audit.MissingRouteNodes[node.NodeID] = path
		if topLevel {
			audit.SecondaryNavGaps[node.Title] = append(audit.SecondaryNavGaps[node.Title], path)
		}
	}
	for _, child := range node.Children {
		markMissingRouteNodes(child, path, routePaths, audit, false)
	}
}

func flattenNodeIDs(nodes []NavigationNode) map[string]struct{} {
	index := make(map[string]struct{})
	var walk func([]NavigationNode)
	walk = func(current []NavigationNode) {
		for _, node := range current {
			index[node.NodeID] = struct{}{}
			walk(node.Children)
		}
	}
	walk(nodes)
	return index
}

func diffRequiredFields(required []string, emitted []string) []string {
	index := make(map[string]struct{}, len(emitted))
	for _, field := range emitted {
		index[field] = struct{}{}
	}
	missing := make([]string, 0)
	for _, field := range required {
		if _, ok := index[field]; !ok {
			missing = append(missing, field)
		}
	}
	return missing
}

func renderNavigationNode(node NavigationNode, base string) string {
	path := base + "/" + node.Segment
	return fmt.Sprintf("- %s (%s) screen=%s", node.Title, path, node.ScreenID)
}

func formatMissingRouteNodes(items map[string]string) string {
	if len(items) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, items[key]))
	}
	return strings.Join(parts, ", ")
}

func formatSecondaryNavGaps(items map[string][]string) string {
	if len(items) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, strings.Join(items[key], ", ")))
	}
	return strings.Join(parts, ", ")
}

func formatUndefinedTokenRefs(items map[string][]string) string {
	if len(items) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, strings.Join(items[key], ", ")))
	}
	return strings.Join(parts, ", ")
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func roundTrip[T any](value T) (T, error) {
	var restored T
	data, err := json.Marshal(value)
	if err != nil {
		return restored, err
	}
	if err := json.Unmarshal(data, &restored); err != nil {
		return restored, err
	}
	return restored, nil
}
