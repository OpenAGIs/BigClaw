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
