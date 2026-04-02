package ui

import (
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
	Tokens     []string `json:"tokens"`
	States     []string `json:"states"`
	UsageNotes string   `json:"usage_notes,omitempty"`
}

type ComponentSpec struct {
	Name                      string             `json:"name"`
	Readiness                 string             `json:"readiness"`
	Slots                     []string           `json:"slots,omitempty"`
	DocumentationComplete     bool               `json:"documentation_complete"`
	AccessibilityRequirements []string           `json:"accessibility_requirements,omitempty"`
	Variants                  []ComponentVariant `json:"variants"`
}

func (c ComponentSpec) ReleaseReady(definedTokens map[string]struct{}) bool {
	if strings.TrimSpace(strings.ToLower(c.Readiness)) != "stable" {
		return false
	}
	if !c.DocumentationComplete {
		return false
	}
	if !c.AccessibilityComplete() {
		return false
	}
	if len(c.MissingRequiredStates()) > 0 {
		return false
	}
	return len(c.undefinedTokens(definedTokens)) == 0
}

func (c ComponentSpec) AccessibilityComplete() bool {
	return len(c.AccessibilityRequirements) > 0
}

func (c ComponentSpec) TokenNames() []string {
	seen := map[string]struct{}{}
	for _, variant := range c.Variants {
		for _, token := range variant.Tokens {
			token = strings.TrimSpace(token)
			if token == "" {
				continue
			}
			seen[token] = struct{}{}
		}
	}
	return sortedKeys(seen)
}

func (c ComponentSpec) MissingRequiredStates() []string {
	required := []string{"default", "hover", "disabled"}
	seen := map[string]struct{}{}
	for _, variant := range c.Variants {
		for _, state := range variant.States {
			state = strings.TrimSpace(strings.ToLower(state))
			if state == "" {
				continue
			}
			seen[state] = struct{}{}
		}
	}

	missing := make([]string, 0, len(required))
	for _, state := range required {
		if _, ok := seen[state]; !ok {
			missing = append(missing, state)
		}
	}
	return missing
}

func (c ComponentSpec) undefinedTokens(definedTokens map[string]struct{}) []string {
	missing := make([]string, 0)
	for _, token := range c.TokenNames() {
		if _, ok := definedTokens[token]; !ok {
			missing = append(missing, token)
		}
	}
	return missing
}

type DesignSystem struct {
	Name       string          `json:"name"`
	Version    string          `json:"version"`
	Tokens     []DesignToken   `json:"tokens"`
	Components []ComponentSpec `json:"components"`
}

type DesignSystemAudit struct {
	SystemName                     string              `json:"system_name"`
	Version                        string              `json:"version"`
	TokenCounts                    map[string]int      `json:"token_counts"`
	ComponentCount                 int                 `json:"component_count"`
	ReleaseReadyComponents         []string            `json:"release_ready_components"`
	ComponentsMissingDocs          []string            `json:"components_missing_docs"`
	ComponentsMissingAccessibility []string            `json:"components_missing_accessibility"`
	ComponentsMissingStates        []string            `json:"components_missing_states"`
	UndefinedTokenRefs             map[string][]string `json:"undefined_token_refs"`
	TokenOrphans                   []string            `json:"token_orphans"`
	ReadinessScore                 float64             `json:"readiness_score"`
}

type ComponentLibrary struct{}

func (ComponentLibrary) Audit(system DesignSystem) DesignSystemAudit {
	tokenCounts := map[string]int{}
	definedTokens := map[string]struct{}{}
	referencedTokens := map[string]struct{}{}
	releaseReady := make([]string, 0)
	missingDocs := make([]string, 0)
	missingAccessibility := make([]string, 0)
	missingStates := make([]string, 0)
	undefinedRefs := map[string][]string{}

	for _, token := range system.Tokens {
		definedTokens[token.Name] = struct{}{}
		tokenCounts[token.Category]++
	}

	for _, component := range system.Components {
		tokenNames := component.TokenNames()
		for _, token := range tokenNames {
			referencedTokens[token] = struct{}{}
		}
		if !component.DocumentationComplete {
			missingDocs = append(missingDocs, component.Name)
		}
		if !component.AccessibilityComplete() {
			missingAccessibility = append(missingAccessibility, component.Name)
		}
		if len(component.MissingRequiredStates()) > 0 {
			missingStates = append(missingStates, component.Name)
		}
		if refs := component.undefinedTokens(definedTokens); len(refs) > 0 {
			undefinedRefs[component.Name] = refs
		}
		if component.ReleaseReady(definedTokens) {
			releaseReady = append(releaseReady, component.Name)
		}
	}

	orphanTokens := make([]string, 0)
	for _, token := range system.Tokens {
		if _, ok := referencedTokens[token.Name]; !ok {
			orphanTokens = append(orphanTokens, token.Name)
		}
	}

	sort.Strings(releaseReady)
	sort.Strings(missingDocs)
	sort.Strings(missingAccessibility)
	sort.Strings(missingStates)
	sort.Strings(orphanTokens)

	return DesignSystemAudit{
		SystemName:                     system.Name,
		Version:                        system.Version,
		TokenCounts:                    tokenCounts,
		ComponentCount:                 len(system.Components),
		ReleaseReadyComponents:         releaseReady,
		ComponentsMissingDocs:          missingDocs,
		ComponentsMissingAccessibility: missingAccessibility,
		ComponentsMissingStates:        missingStates,
		UndefinedTokenRefs:             undefinedRefs,
		TokenOrphans:                   orphanTokens,
		ReadinessScore:                 designReadinessScore(missingDocs, missingAccessibility, missingStates),
	}
}

func designReadinessScore(missingDocs, missingAccessibility, missingStates []string) float64 {
	score := 100.0
	score -= 25.0 * float64(len(missingDocs))
	score -= 20.0 * float64(len(missingAccessibility))
	score -= 20.0 * float64(len(missingStates))
	if score < 0 {
		return 0
	}
	return score
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
	RecentQueriesEnabled bool            `json:"recent_queries_enabled"`
	Commands             []CommandAction `json:"commands"`
}

type ConsoleTopBar struct {
	Name                      string              `json:"name"`
	SearchPlaceholder         string              `json:"search_placeholder"`
	EnvironmentOptions        []string            `json:"environment_options"`
	TimeRangeOptions          []string            `json:"time_range_options"`
	AlertChannels             []string            `json:"alert_channels"`
	DocumentationComplete     bool                `json:"documentation_complete"`
	AccessibilityRequirements []string            `json:"accessibility_requirements,omitempty"`
	CommandEntry              ConsoleCommandEntry `json:"command_entry"`
}

type ConsoleTopBarAudit struct {
	Name                     string   `json:"name"`
	MissingCapabilities      []string `json:"missing_capabilities"`
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
	if strings.TrimSpace(topBar.CommandEntry.TriggerLabel) == "" || strings.TrimSpace(topBar.CommandEntry.Placeholder) == "" || len(topBar.CommandEntry.Commands) == 0 {
		missing = append(missing, "command-shell")
	}

	return ConsoleTopBarAudit{
		Name:                     topBar.Name,
		MissingCapabilities:      missing,
		DocumentationComplete:    topBar.DocumentationComplete,
		AccessibilityComplete:    hasAll(topBar.AccessibilityRequirements, "keyboard-navigation", "screen-reader-label", "focus-visible"),
		CommandShortcutSupported: strings.Contains(strings.ToLower(topBar.CommandEntry.Shortcut), "cmd+k") && strings.Contains(strings.ToLower(topBar.CommandEntry.Shortcut), "ctrl+k"),
		CommandCount:             len(topBar.CommandEntry.Commands),
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
	GlobalNav []NavigationNode  `json:"global_nav"`
	Routes    []NavigationRoute `json:"routes"`
}

func (a InformationArchitecture) NavigationEntries() []NavigationRoute {
	entries := make([]NavigationRoute, 0)
	for _, node := range a.GlobalNav {
		flattenNav(node, "", &entries)
	}
	return entries
}

func flattenNav(node NavigationNode, parentPath string, entries *[]NavigationRoute) {
	path := joinPath(parentPath, node.Segment)
	*entries = append(*entries, NavigationRoute{
		Path:      path,
		ScreenID:  node.ScreenID,
		Title:     node.Title,
		NavNodeID: node.NodeID,
	})
	for _, child := range node.Children {
		flattenNav(child, path, entries)
	}
}

func (a InformationArchitecture) ResolveRoute(path string) (NavigationRoute, bool) {
	normalized := normalizePath(path)
	for _, route := range a.Routes {
		if normalizePath(route.Path) == normalized {
			return route, true
		}
	}
	return NavigationRoute{}, false
}

type InformationArchitectureAudit struct {
	TotalNavigationNodes int                 `json:"total_navigation_nodes"`
	TotalRoutes          int                 `json:"total_routes"`
	DuplicateRoutes      []string            `json:"duplicate_routes"`
	MissingRouteNodes    map[string]string   `json:"missing_route_nodes"`
	SecondaryNavGaps     map[string][]string `json:"secondary_nav_gaps"`
	OrphanRoutes         []string            `json:"orphan_routes"`
}

func (a InformationArchitectureAudit) Healthy() bool {
	return len(a.DuplicateRoutes) == 0 && len(a.MissingRouteNodes) == 0 && len(a.SecondaryNavGaps) == 0 && len(a.OrphanRoutes) == 0
}

func (a InformationArchitecture) Audit() InformationArchitectureAudit {
	nodePaths := map[string]string{}
	nodeTitles := map[string]string{}
	nodeChildren := map[string]int{}
	totalNodes := 0
	for _, node := range a.GlobalNav {
		indexNav(node, "", nodePaths, nodeTitles, nodeChildren, &totalNodes)
	}

	routeCounts := map[string]int{}
	routeByNode := map[string]string{}
	orphanRoutes := make([]string, 0)
	for _, route := range a.Routes {
		path := normalizePath(route.Path)
		routeCounts[path]++
		if _, ok := nodePaths[route.NavNodeID]; ok {
			routeByNode[route.NavNodeID] = path
		} else {
			orphanRoutes = append(orphanRoutes, path)
		}
	}

	duplicates := make([]string, 0)
	for path, count := range routeCounts {
		if count > 1 {
			duplicates = append(duplicates, path)
		}
	}

	missingRouteNodes := map[string]string{}
	secondaryNavGaps := map[string][]string{}
	for nodeID, path := range nodePaths {
		if _, ok := routeByNode[nodeID]; ok {
			continue
		}
		missingRouteNodes[nodeID] = path
		if nodeChildren[nodeID] > 0 {
			secondaryNavGaps[nodeTitles[nodeID]] = append(secondaryNavGaps[nodeTitles[nodeID]], path)
		}
	}

	sort.Strings(duplicates)
	sort.Strings(orphanRoutes)
	for _, gaps := range secondaryNavGaps {
		sort.Strings(gaps)
	}

	return InformationArchitectureAudit{
		TotalNavigationNodes: totalNodes,
		TotalRoutes:          len(a.Routes),
		DuplicateRoutes:      duplicates,
		MissingRouteNodes:    missingRouteNodes,
		SecondaryNavGaps:     secondaryNavGaps,
		OrphanRoutes:         orphanRoutes,
	}
}

func indexNav(node NavigationNode, parentPath string, nodePaths map[string]string, nodeTitles map[string]string, nodeChildren map[string]int, totalNodes *int) {
	path := joinPath(parentPath, node.Segment)
	nodePaths[node.NodeID] = path
	nodeTitles[node.NodeID] = node.Title
	nodeChildren[node.NodeID] = len(node.Children)
	*totalNodes++
	for _, child := range node.Children {
		indexNav(child, path, nodePaths, nodeTitles, nodeChildren, totalNodes)
	}
}

type RolePermissionScenario struct {
	ScreenID     string   `json:"screen_id"`
	AllowedRoles []string `json:"allowed_roles"`
	DeniedRoles  []string `json:"denied_roles"`
	AuditEvent   string   `json:"audit_event"`
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
	Personas           []string `json:"personas"`
	CriticalSteps      []string `json:"critical_steps"`
	ExpectedMaxSteps   int      `json:"expected_max_steps"`
	ObservedSteps      int      `json:"observed_steps"`
	KeyboardAccessible bool     `json:"keyboard_accessible"`
	EmptyStateGuidance bool     `json:"empty_state_guidance"`
	RecoverySupport    bool     `json:"recovery_support"`
}

type AuditRequirement struct {
	EventType             string   `json:"event_type"`
	RequiredFields        []string `json:"required_fields"`
	EmittedFields         []string `json:"emitted_fields"`
	RetentionDays         int      `json:"retention_days"`
	ObservedRetentionDays int      `json:"observed_retention_days"`
}

type UIAcceptanceSuite struct {
	Name                  string                   `json:"name"`
	Version               string                   `json:"version"`
	RolePermissions       []RolePermissionScenario `json:"role_permissions"`
	DataAccuracyChecks    []DataAccuracyCheck      `json:"data_accuracy_checks"`
	PerformanceBudgets    []PerformanceBudget      `json:"performance_budgets"`
	UsabilityJourneys     []UsabilityJourney       `json:"usability_journeys"`
	AuditRequirements     []AuditRequirement       `json:"audit_requirements"`
	DocumentationComplete bool                     `json:"documentation_complete"`
}

type UIAcceptanceAudit struct {
	Name                      string   `json:"name"`
	Version                   string   `json:"version"`
	PermissionGaps            []string `json:"permission_gaps"`
	FailingDataChecks         []string `json:"failing_data_checks"`
	FailingPerformanceBudgets []string `json:"failing_performance_budgets"`
	FailingUsabilityJourneys  []string `json:"failing_usability_journeys"`
	IncompleteAuditTrails     []string `json:"incomplete_audit_trails"`
	DocumentationComplete     bool     `json:"documentation_complete"`
	ReadinessScore            float64  `json:"readiness_score"`
}

func (a UIAcceptanceAudit) ReleaseReady() bool {
	return a.ReadinessScore == 100.0
}

type UIAcceptanceLibrary struct{}

func (UIAcceptanceLibrary) Audit(suite UIAcceptanceSuite) UIAcceptanceAudit {
	permissionGaps := make([]string, 0)
	for _, scenario := range suite.RolePermissions {
		missing := make([]string, 0)
		if len(scenario.DeniedRoles) == 0 {
			missing = append(missing, "denied-roles")
		}
		if strings.TrimSpace(scenario.AuditEvent) == "" {
			missing = append(missing, "audit-event")
		}
		if len(missing) > 0 {
			permissionGaps = append(permissionGaps, fmt.Sprintf("%s: missing=%s", scenario.ScreenID, strings.Join(missing, ", ")))
		}
	}

	failingDataChecks := make([]string, 0)
	for _, check := range suite.DataAccuracyChecks {
		if check.ObservedDelta > check.Tolerance || check.ObservedFreshnessSeconds > check.FreshnessSLOSeconds {
			failingDataChecks = append(failingDataChecks, fmt.Sprintf("%s.%s: delta=%.1f freshness=%ds", check.ScreenID, check.MetricID, check.ObservedDelta, check.ObservedFreshnessSeconds))
		}
	}

	failingPerformance := make([]string, 0)
	for _, budget := range suite.PerformanceBudgets {
		if budget.ObservedP95MS > budget.TargetP95MS || budget.ObservedTTIMS > budget.TargetTTIMS {
			failingPerformance = append(failingPerformance, fmt.Sprintf("%s.%s: p95=%dms tti=%dms", budget.SurfaceID, budget.Interaction, budget.ObservedP95MS, budget.ObservedTTIMS))
		}
	}

	failingUsability := make([]string, 0)
	for _, journey := range suite.UsabilityJourneys {
		if journey.ObservedSteps > journey.ExpectedMaxSteps || !journey.KeyboardAccessible || !journey.EmptyStateGuidance || !journey.RecoverySupport {
			failingUsability = append(failingUsability, fmt.Sprintf("%s: steps=%d/%d", journey.JourneyID, journey.ObservedSteps, journey.ExpectedMaxSteps))
		}
	}

	incompleteAuditTrails := make([]string, 0)
	for _, requirement := range suite.AuditRequirements {
		missingFields := missingFields(requirement.RequiredFields, requirement.EmittedFields)
		if len(missingFields) > 0 || requirement.ObservedRetentionDays < requirement.RetentionDays {
			incompleteAuditTrails = append(incompleteAuditTrails, fmt.Sprintf("%s: missing_fields=%s retention=%d/%dd", requirement.EventType, joinOrNone(missingFields), requirement.ObservedRetentionDays, requirement.RetentionDays))
		}
	}

	score := 100.0
	if !suite.DocumentationComplete {
		score -= 20.0
	}
	if len(permissionGaps) > 0 {
		score -= 20.0
	}
	if len(failingDataChecks) > 0 {
		score -= 20.0
	}
	if len(failingPerformance) > 0 {
		score -= 20.0
	}
	if len(failingUsability) > 0 {
		score -= 10.0
	}
	if len(incompleteAuditTrails) > 0 {
		score -= 10.0
	}
	if score < 0 {
		score = 0
	}

	return UIAcceptanceAudit{
		Name:                      suite.Name,
		Version:                   suite.Version,
		PermissionGaps:            permissionGaps,
		FailingDataChecks:         failingDataChecks,
		FailingPerformanceBudgets: failingPerformance,
		FailingUsabilityJourneys:  failingUsability,
		IncompleteAuditTrails:     incompleteAuditTrails,
		DocumentationComplete:     suite.DocumentationComplete,
		ReadinessScore:            score,
	}
}

func RenderDesignSystemReport(system DesignSystem, audit DesignSystemAudit) string {
	lines := []string{
		"# Design System Report",
		fmt.Sprintf("- System: %s (%s)", system.Name, system.Version),
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
		undefined := joinOrNone(audit.UndefinedTokenRefs[component.Name])
		lines = append(lines, fmt.Sprintf("- %s: readiness=%s docs=%s a11y=%s states=%s missing_states=%s undefined_tokens=%s",
			component.Name,
			component.Readiness,
			boolWord(component.DocumentationComplete),
			boolWord(component.AccessibilityComplete()),
			strings.Join(allStates(component), ", "),
			joinOrNone(component.MissingRequiredStates()),
			undefined,
		))
	}
	lines = append(lines, fmt.Sprintf("- Missing interaction states: %s", joinOrNone(audit.ComponentsMissingStates)))
	lines = append(lines, fmt.Sprintf("- Undefined token refs: %s", renderUndefinedTokenRefs(audit.UndefinedTokenRefs)))
	lines = append(lines, fmt.Sprintf("- Orphan tokens: %s", joinOrNone(audit.TokenOrphans)))
	return strings.Join(lines, "\n")
}

func RenderConsoleTopBarReport(topBar ConsoleTopBar, audit ConsoleTopBarAudit) string {
	lines := []string{
		"# Console Top Bar Report",
		fmt.Sprintf("- Top Bar: %s", topBar.Name),
		fmt.Sprintf("- Command Shortcut: %s", topBar.CommandEntry.Shortcut),
		fmt.Sprintf("- Release Ready: %s", boolWord(audit.ReleaseReady())),
	}
	for _, command := range topBar.CommandEntry.Commands {
		shortcut := "none"
		if strings.TrimSpace(command.Shortcut) != "" {
			shortcut = command.Shortcut
		}
		lines = append(lines, fmt.Sprintf("- %s: %s [%s] shortcut=%s", command.ID, command.Title, command.Section, shortcut))
	}
	lines = append(lines, fmt.Sprintf("- Missing capabilities: %s", joinOrNone(audit.MissingCapabilities)))
	lines = append(lines, fmt.Sprintf("- Cmd/Ctrl+K supported: %s", boolWord(audit.CommandShortcutSupported)))
	return strings.Join(lines, "\n")
}

func RenderInformationArchitectureReport(architecture InformationArchitecture, audit InformationArchitectureAudit) string {
	lines := []string{
		"# Information Architecture Report",
		fmt.Sprintf("- Healthy: %s", boolWord(audit.Healthy())),
	}
	for _, entry := range architecture.NavigationEntries() {
		lines = append(lines, fmt.Sprintf("- %s (%s) screen=%s", entry.Title, entry.Path, entry.ScreenID))
	}
	for _, route := range architecture.Routes {
		lines = append(lines, fmt.Sprintf("- %s: screen=%s title=%s nav_node=%s", normalizePath(route.Path), route.ScreenID, route.Title, route.NavNodeID))
	}
	lines = append(lines, fmt.Sprintf("- Duplicate routes: %s", joinOrNone(audit.DuplicateRoutes)))
	lines = append(lines, fmt.Sprintf("- Missing route nodes: %s", renderStringMap(audit.MissingRouteNodes)))
	lines = append(lines, fmt.Sprintf("- Secondary nav gaps: %s", renderNestedMap(audit.SecondaryNavGaps)))
	lines = append(lines, fmt.Sprintf("- Orphan routes: %s", joinOrNone(audit.OrphanRoutes)))
	return strings.Join(lines, "\n")
}

func RenderUIAcceptanceReport(suite UIAcceptanceSuite, audit UIAcceptanceAudit) string {
	lines := []string{
		"# UI Acceptance Report",
		fmt.Sprintf("- Suite: %s (%s)", suite.Name, suite.Version),
		fmt.Sprintf("- Readiness Score: %.1f", audit.ReadinessScore),
		fmt.Sprintf("- Release Ready: %s", boolWord(audit.ReleaseReady())),
	}
	for _, scenario := range suite.RolePermissions {
		lines = append(lines, fmt.Sprintf("- Role/Permission %s: allow=%s deny=%s audit_event=%s", scenario.ScreenID, strings.Join(scenario.AllowedRoles, ", "), strings.Join(scenario.DeniedRoles, ", "), scenario.AuditEvent))
	}
	for _, check := range suite.DataAccuracyChecks {
		lines = append(lines, fmt.Sprintf("- Data Accuracy %s.%s: delta=%.1f tolerance=%.1f freshness=%d/%ds", check.ScreenID, check.MetricID, check.ObservedDelta, check.Tolerance, check.ObservedFreshnessSeconds, check.FreshnessSLOSeconds))
	}
	for _, budget := range suite.PerformanceBudgets {
		lines = append(lines, fmt.Sprintf("- Performance %s.%s: p95=%d/%dms tti=%d/%dms", budget.SurfaceID, budget.Interaction, budget.ObservedP95MS, budget.TargetP95MS, budget.ObservedTTIMS, budget.TargetTTIMS))
	}
	for _, journey := range suite.UsabilityJourneys {
		lines = append(lines, fmt.Sprintf("- Usability %s: steps=%d/%d keyboard=%s empty_state=%s recovery=%s", journey.JourneyID, journey.ObservedSteps, journey.ExpectedMaxSteps, boolWord(journey.KeyboardAccessible), boolWord(journey.EmptyStateGuidance), boolWord(journey.RecoverySupport)))
	}
	lines = append(lines, fmt.Sprintf("- Audit completeness gaps: %s", joinOrNone(audit.IncompleteAuditTrails)))
	return strings.Join(lines, "\n")
}

func allStates(component ComponentSpec) []string {
	seen := map[string]struct{}{}
	ordered := make([]string, 0)
	for _, variant := range component.Variants {
		for _, state := range variant.States {
			state = strings.TrimSpace(state)
			if state == "" {
				continue
			}
			if _, ok := seen[state]; ok {
				continue
			}
			seen[state] = struct{}{}
			ordered = append(ordered, state)
		}
	}
	return ordered
}

func renderUndefinedTokenRefs(refs map[string][]string) string {
	if len(refs) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(refs))
	for key := range refs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, strings.Join(refs[key], ", ")))
	}
	return strings.Join(parts, "; ")
}

func renderStringMap(values map[string]string) string {
	if len(values) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, values[key]))
	}
	return strings.Join(parts, ", ")
}

func renderNestedMap(values map[string][]string) string {
	if len(values) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, strings.Join(values[key], ", ")))
	}
	return strings.Join(parts, ", ")
}

func hasAll(values []string, required ...string) bool {
	seen := map[string]struct{}{}
	for _, value := range values {
		seen[strings.TrimSpace(strings.ToLower(value))] = struct{}{}
	}
	for _, value := range required {
		if _, ok := seen[strings.TrimSpace(strings.ToLower(value))]; !ok {
			return false
		}
	}
	return true
}

func missingFields(required, emitted []string) []string {
	seen := map[string]struct{}{}
	for _, field := range emitted {
		seen[strings.TrimSpace(field)] = struct{}{}
	}
	missing := make([]string, 0)
	for _, field := range required {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		if _, ok := seen[field]; !ok {
			missing = append(missing, field)
		}
	}
	return missing
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func sortedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func boolWord(value bool) string {
	if value {
		return "True"
	}
	return "False"
}

func joinPath(parentPath, segment string) string {
	segment = strings.Trim(segment, "/")
	if parentPath == "" {
		if segment == "" {
			return "/"
		}
		return "/" + segment
	}
	if segment == "" {
		return parentPath
	}
	return strings.TrimRight(parentPath, "/") + "/" + segment
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}
