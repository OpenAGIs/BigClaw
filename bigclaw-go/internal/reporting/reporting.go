package reporting

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
)

type Summary struct {
	TotalRuns          int   `json:"total_runs"`
	CompletedRuns      int   `json:"completed_runs"`
	BlockedRuns        int   `json:"blocked_runs"`
	HighRiskRuns       int   `json:"high_risk_runs"`
	RegressionFindings int   `json:"regression_findings"`
	HumanInterventions int   `json:"human_interventions"`
	BudgetCentsTotal   int64 `json:"budget_cents_total"`
	PremiumRuns        int   `json:"premium_runs"`
}

type TeamBreakdown struct {
	Key                string `json:"key"`
	TotalRuns          int    `json:"total_runs"`
	CompletedRuns      int    `json:"completed_runs"`
	BlockedRuns        int    `json:"blocked_runs"`
	BudgetCentsTotal   int64  `json:"budget_cents_total"`
	HumanInterventions int    `json:"human_interventions"`
}

type Weekly struct {
	WeekStart     time.Time       `json:"week_start"`
	WeekEnd       time.Time       `json:"week_end"`
	Summary       Summary         `json:"summary"`
	TeamBreakdown []TeamBreakdown `json:"team_breakdown"`
	Highlights    []string        `json:"highlights"`
	Actions       []string        `json:"actions"`
	Markdown      string          `json:"markdown"`
}

type OperationsMetricDefinition struct {
	MetricID     string   `json:"metric_id"`
	Label        string   `json:"label"`
	Unit         string   `json:"unit"`
	Direction    string   `json:"direction"`
	Formula      string   `json:"formula"`
	Description  string   `json:"description"`
	SourceFields []string `json:"source_fields,omitempty"`
}

type OperationsMetricValue struct {
	MetricID     string   `json:"metric_id"`
	Label        string   `json:"label"`
	Value        float64  `json:"value"`
	DisplayValue string   `json:"display_value"`
	Numerator    float64  `json:"numerator"`
	Denominator  float64  `json:"denominator"`
	Unit         string   `json:"unit"`
	Evidence     []string `json:"evidence,omitempty"`
}

type OperationsMetricSpec struct {
	Name         string                       `json:"name"`
	GeneratedAt  time.Time                    `json:"generated_at"`
	PeriodStart  time.Time                    `json:"period_start"`
	PeriodEnd    time.Time                    `json:"period_end"`
	TimezoneName string                       `json:"timezone_name"`
	Definitions  []OperationsMetricDefinition `json:"definitions,omitempty"`
	Values       []OperationsMetricValue      `json:"values,omitempty"`
}

type WeeklyArtifacts struct {
	RootDir          string `json:"root_dir"`
	WeeklyReportPath string `json:"weekly_report_path"`
	DashboardPath    string `json:"dashboard_path"`
	MetricSpecPath   string `json:"metric_spec_path,omitempty"`
}

type EngineeringOverviewPermission struct {
	ViewerRole     string   `json:"viewer_role"`
	AllowedModules []string `json:"allowed_modules,omitempty"`
}

func (p EngineeringOverviewPermission) CanView(module string) bool {
	module = strings.TrimSpace(module)
	for _, allowed := range p.AllowedModules {
		if strings.EqualFold(strings.TrimSpace(allowed), module) {
			return true
		}
	}
	return false
}

type DashboardWidgetSpec struct {
	WidgetID      string `json:"widget_id"`
	Title         string `json:"title"`
	Module        string `json:"module"`
	DataSource    string `json:"data_source"`
	DefaultWidth  int    `json:"default_width"`
	DefaultHeight int    `json:"default_height"`
	MinWidth      int    `json:"min_width"`
	MaxWidth      int    `json:"max_width"`
}

type DashboardWidgetPlacement struct {
	PlacementID   string   `json:"placement_id"`
	WidgetID      string   `json:"widget_id"`
	Column        int      `json:"column"`
	Row           int      `json:"row"`
	Width         int      `json:"width"`
	Height        int      `json:"height"`
	TitleOverride string   `json:"title_override,omitempty"`
	Filters       []string `json:"filters,omitempty"`
}

type DashboardLayout struct {
	LayoutID   string                     `json:"layout_id"`
	Name       string                     `json:"name"`
	Columns    int                        `json:"columns"`
	Placements []DashboardWidgetPlacement `json:"placements,omitempty"`
}

type DashboardBuilder struct {
	Name                  string                        `json:"name"`
	Period                string                        `json:"period"`
	Owner                 string                        `json:"owner"`
	Permissions           EngineeringOverviewPermission `json:"permissions"`
	Widgets               []DashboardWidgetSpec         `json:"widgets,omitempty"`
	Layouts               []DashboardLayout             `json:"layouts,omitempty"`
	DocumentationComplete bool                          `json:"documentation_complete"`
}

func (b DashboardBuilder) WidgetIndex() map[string]DashboardWidgetSpec {
	out := make(map[string]DashboardWidgetSpec, len(b.Widgets))
	for _, widget := range b.Widgets {
		out[widget.WidgetID] = widget
	}
	return out
}

type DashboardBuilderAudit struct {
	Name                  string   `json:"name"`
	TotalWidgets          int      `json:"total_widgets"`
	LayoutCount           int      `json:"layout_count"`
	PlacedWidgets         int      `json:"placed_widgets"`
	DuplicatePlacementIDs []string `json:"duplicate_placement_ids,omitempty"`
	MissingWidgetDefs     []string `json:"missing_widget_defs,omitempty"`
	InaccessibleWidgets   []string `json:"inaccessible_widgets,omitempty"`
	OverlappingPlacements []string `json:"overlapping_placements,omitempty"`
	OutOfBoundsPlacements []string `json:"out_of_bounds_placements,omitempty"`
	EmptyLayouts          []string `json:"empty_layouts,omitempty"`
	DocumentationComplete bool     `json:"documentation_complete"`
}

func (a DashboardBuilderAudit) ReleaseReady() bool {
	return len(a.DuplicatePlacementIDs) == 0 &&
		len(a.MissingWidgetDefs) == 0 &&
		len(a.InaccessibleWidgets) == 0 &&
		len(a.OverlappingPlacements) == 0 &&
		len(a.OutOfBoundsPlacements) == 0 &&
		len(a.EmptyLayouts) == 0 &&
		a.DocumentationComplete
}

func Build(tasks []domain.Task, events []domain.Event, weekStart, weekEnd time.Time) Weekly {
	weekly := Weekly{WeekStart: weekStart, WeekEnd: weekEnd}
	byTeam := make(map[string]*TeamBreakdown)
	interventions := interventionCounts(events)
	for _, task := range tasks {
		if !within(task.UpdatedAt, weekStart, weekEnd) {
			continue
		}
		weekly.Summary.TotalRuns++
		weekly.Summary.BudgetCentsTotal += task.BudgetCents
		if task.State == domain.TaskSucceeded {
			weekly.Summary.CompletedRuns++
		}
		if task.State == domain.TaskBlocked || task.State == domain.TaskDeadLetter || task.State == domain.TaskFailed {
			weekly.Summary.BlockedRuns++
		}
		if task.RiskLevel == domain.RiskHigh {
			weekly.Summary.HighRiskRuns++
		}
		if regressionCount(task) > 0 {
			weekly.Summary.RegressionFindings += regressionCount(task)
		}
		if strings.EqualFold(strings.TrimSpace(task.Metadata["plan"]), "premium") {
			weekly.Summary.PremiumRuns++
		}
		weekly.Summary.HumanInterventions += interventions[task.ID]
		team := firstNonEmpty(task.Metadata["team"], "unassigned")
		entry := byTeam[team]
		if entry == nil {
			entry = &TeamBreakdown{Key: team}
			byTeam[team] = entry
		}
		entry.TotalRuns++
		entry.BudgetCentsTotal += task.BudgetCents
		entry.HumanInterventions += interventions[task.ID]
		if task.State == domain.TaskSucceeded {
			entry.CompletedRuns++
		}
		if task.State == domain.TaskBlocked || task.State == domain.TaskDeadLetter || task.State == domain.TaskFailed {
			entry.BlockedRuns++
		}
	}
	for _, entry := range byTeam {
		weekly.TeamBreakdown = append(weekly.TeamBreakdown, *entry)
	}
	sort.SliceStable(weekly.TeamBreakdown, func(i, j int) bool {
		if weekly.TeamBreakdown[i].TotalRuns == weekly.TeamBreakdown[j].TotalRuns {
			return weekly.TeamBreakdown[i].Key < weekly.TeamBreakdown[j].Key
		}
		return weekly.TeamBreakdown[i].TotalRuns > weekly.TeamBreakdown[j].TotalRuns
	})
	weekly.Highlights = buildHighlights(weekly)
	weekly.Actions = buildActions(weekly)
	weekly.Markdown = RenderMarkdown(weekly)
	return weekly
}

func RenderMarkdown(weekly Weekly) string {
	builder := strings.Builder{}
	builder.WriteString("# BigClaw Weekly Ops Report\n\n")
	builder.WriteString(fmt.Sprintf("Window: %s -> %s\n\n", weekly.WeekStart.Format("2006-01-02"), weekly.WeekEnd.Format("2006-01-02")))
	builder.WriteString("## Summary\n")
	builder.WriteString(fmt.Sprintf("- Total runs: %d\n", weekly.Summary.TotalRuns))
	builder.WriteString(fmt.Sprintf("- Completed runs: %d\n", weekly.Summary.CompletedRuns))
	builder.WriteString(fmt.Sprintf("- Blocked runs: %d\n", weekly.Summary.BlockedRuns))
	builder.WriteString(fmt.Sprintf("- High risk runs: %d\n", weekly.Summary.HighRiskRuns))
	builder.WriteString(fmt.Sprintf("- Human interventions: %d\n", weekly.Summary.HumanInterventions))
	builder.WriteString(fmt.Sprintf("- Regressions: %d\n", weekly.Summary.RegressionFindings))
	builder.WriteString(fmt.Sprintf("- Premium runs: %d\n", weekly.Summary.PremiumRuns))
	builder.WriteString(fmt.Sprintf("- Budget cents: %d\n\n", weekly.Summary.BudgetCentsTotal))
	builder.WriteString("## Team Breakdown\n")
	if len(weekly.TeamBreakdown) == 0 {
		builder.WriteString("- None\n\n")
	} else {
		for _, team := range weekly.TeamBreakdown {
			builder.WriteString(fmt.Sprintf("- %s: total=%d completed=%d blocked=%d budget_cents=%d interventions=%d\n", team.Key, team.TotalRuns, team.CompletedRuns, team.BlockedRuns, team.BudgetCentsTotal, team.HumanInterventions))
		}
		builder.WriteString("\n")
	}
	builder.WriteString("## Highlights\n")
	for _, highlight := range weekly.Highlights {
		builder.WriteString("- " + highlight + "\n")
	}
	builder.WriteString("\n")
	builder.WriteString("## Actions\n")
	for _, action := range weekly.Actions {
		builder.WriteString("- " + action + "\n")
	}
	return builder.String()
}

func RenderOperationsDashboard(weekly Weekly) string {
	builder := strings.Builder{}
	builder.WriteString("# Operations Dashboard\n\n")
	builder.WriteString(fmt.Sprintf("- Window: %s -> %s\n", weekly.WeekStart.Format("2006-01-02"), weekly.WeekEnd.Format("2006-01-02")))
	builder.WriteString(fmt.Sprintf("- Total Runs: %d\n", weekly.Summary.TotalRuns))
	builder.WriteString(fmt.Sprintf("- Completed Runs: %d\n", weekly.Summary.CompletedRuns))
	builder.WriteString(fmt.Sprintf("- Blocked Runs: %d\n", weekly.Summary.BlockedRuns))
	builder.WriteString(fmt.Sprintf("- High Risk Runs: %d\n", weekly.Summary.HighRiskRuns))
	builder.WriteString(fmt.Sprintf("- Premium Runs: %d\n", weekly.Summary.PremiumRuns))
	builder.WriteString(fmt.Sprintf("- Human Interventions: %d\n", weekly.Summary.HumanInterventions))
	builder.WriteString(fmt.Sprintf("- Regression Findings: %d\n", weekly.Summary.RegressionFindings))
	builder.WriteString(fmt.Sprintf("- Budget Cents Total: %d\n\n", weekly.Summary.BudgetCentsTotal))
	builder.WriteString("## Team Lanes\n")
	if len(weekly.TeamBreakdown) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, team := range weekly.TeamBreakdown {
			builder.WriteString(fmt.Sprintf("- %s: total=%d completed=%d blocked=%d interventions=%d\n", team.Key, team.TotalRuns, team.CompletedRuns, team.BlockedRuns, team.HumanInterventions))
		}
	}
	return builder.String() + "\n"
}

func RenderOperationsMetricSpec(spec OperationsMetricSpec) string {
	builder := strings.Builder{}
	builder.WriteString("# Operations Metric Spec\n\n")
	builder.WriteString(fmt.Sprintf("- Name: %s\n", strings.TrimSpace(spec.Name)))
	builder.WriteString(fmt.Sprintf("- Generated At: %s\n", spec.GeneratedAt.UTC().Format(time.RFC3339)))
	builder.WriteString(fmt.Sprintf("- Period Start: %s\n", spec.PeriodStart.UTC().Format(time.RFC3339)))
	builder.WriteString(fmt.Sprintf("- Period End: %s\n", spec.PeriodEnd.UTC().Format(time.RFC3339)))
	builder.WriteString(fmt.Sprintf("- Timezone: %s\n\n", firstNonEmpty(spec.TimezoneName, "UTC")))
	builder.WriteString("## Definitions\n\n")
	if len(spec.Definitions) == 0 {
		builder.WriteString("- None\n\n")
	} else {
		for _, definition := range spec.Definitions {
			builder.WriteString(fmt.Sprintf("### %s\n\n", firstNonEmpty(definition.Label, definition.MetricID)))
			builder.WriteString(fmt.Sprintf("- Metric ID: %s\n", definition.MetricID))
			builder.WriteString(fmt.Sprintf("- Unit: %s\n", definition.Unit))
			builder.WriteString(fmt.Sprintf("- Direction: %s\n", definition.Direction))
			builder.WriteString(fmt.Sprintf("- Formula: %s\n", definition.Formula))
			builder.WriteString(fmt.Sprintf("- Description: %s\n", definition.Description))
			builder.WriteString(fmt.Sprintf("- Source Fields: %s\n\n", strings.Join(definition.SourceFields, ", ")))
		}
	}
	builder.WriteString("## Values\n")
	if len(spec.Values) == 0 {
		builder.WriteString("\n- None\n")
		return builder.String()
	}
	builder.WriteString("\n")
	for _, value := range spec.Values {
		evidence := "none"
		if len(value.Evidence) > 0 {
			evidence = strings.Join(value.Evidence, " | ")
		}
		builder.WriteString(fmt.Sprintf("- %s: value=%s numerator=%.1f denominator=%.1f unit=%s evidence=%s\n", firstNonEmpty(value.Label, value.MetricID), firstNonEmpty(value.DisplayValue, formatMetricValue(value.Value)), value.Numerator, value.Denominator, value.Unit, evidence))
	}
	return builder.String()
}

func WriteReport(path string, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func WriteWeeklyOperationsBundle(rootDir string, weekly Weekly, metricSpec *OperationsMetricSpec) (WeeklyArtifacts, error) {
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return WeeklyArtifacts{}, err
	}
	artifacts := WeeklyArtifacts{
		RootDir:          rootDir,
		WeeklyReportPath: filepath.Join(rootDir, "weekly-operations.md"),
		DashboardPath:    filepath.Join(rootDir, "operations-dashboard.md"),
	}
	if err := WriteReport(artifacts.WeeklyReportPath, RenderMarkdown(weekly)); err != nil {
		return WeeklyArtifacts{}, err
	}
	if err := WriteReport(artifacts.DashboardPath, RenderOperationsDashboard(weekly)); err != nil {
		return WeeklyArtifacts{}, err
	}
	if metricSpec != nil {
		artifacts.MetricSpecPath = filepath.Join(rootDir, "operations-metric-spec.md")
		if err := WriteReport(artifacts.MetricSpecPath, RenderOperationsMetricSpec(*metricSpec)); err != nil {
			return WeeklyArtifacts{}, err
		}
	}
	return artifacts, nil
}

func AuditDashboardBuilder(dashboard DashboardBuilder) DashboardBuilderAudit {
	widgetIndex := dashboard.WidgetIndex()
	placementCounts := make(map[string]int)
	missingWidgetDefs := make(map[string]struct{})
	inaccessibleWidgets := make(map[string]struct{})
	overlappingPlacements := make(map[string]struct{})
	outOfBoundsPlacements := make(map[string]struct{})
	emptyLayouts := make([]string, 0)
	placedWidgets := 0

	for _, layout := range dashboard.Layouts {
		if len(layout.Placements) == 0 {
			emptyLayouts = append(emptyLayouts, layout.LayoutID)
			continue
		}

		placedWidgets += len(layout.Placements)
		for _, placement := range layout.Placements {
			placementCounts[placement.PlacementID]++
			spec, ok := widgetIndex[placement.WidgetID]
			if !ok {
				missingWidgetDefs[placement.WidgetID] = struct{}{}
			} else if !dashboard.Permissions.CanView(spec.Module) {
				inaccessibleWidgets[placement.WidgetID] = struct{}{}
			}
			if placement.Column+placement.Width > layout.Columns {
				outOfBoundsPlacements[placement.PlacementID] = struct{}{}
			}
		}

		for index, placement := range layout.Placements {
			for _, other := range layout.Placements[index+1:] {
				if placementsOverlap(placement, other) {
					key := fmt.Sprintf("%s:%s<->%s", layout.LayoutID, placement.PlacementID, other.PlacementID)
					overlappingPlacements[key] = struct{}{}
				}
			}
		}
	}

	duplicateIDs := make([]string, 0)
	for placementID, count := range placementCounts {
		if count > 1 {
			duplicateIDs = append(duplicateIDs, placementID)
		}
	}
	sort.Strings(duplicateIDs)
	sort.Strings(emptyLayouts)

	return DashboardBuilderAudit{
		Name:                  dashboard.Name,
		TotalWidgets:          len(dashboard.Widgets),
		LayoutCount:           len(dashboard.Layouts),
		PlacedWidgets:         placedWidgets,
		DuplicatePlacementIDs: duplicateIDs,
		MissingWidgetDefs:     sortedKeys(missingWidgetDefs),
		InaccessibleWidgets:   sortedKeys(inaccessibleWidgets),
		OverlappingPlacements: sortedKeys(overlappingPlacements),
		OutOfBoundsPlacements: sortedKeys(outOfBoundsPlacements),
		EmptyLayouts:          emptyLayouts,
		DocumentationComplete: dashboard.DocumentationComplete,
	}
}

func RenderDashboardBuilderReport(dashboard DashboardBuilder, audit DashboardBuilderAudit) string {
	builder := strings.Builder{}
	builder.WriteString("# Dashboard Builder\n\n")
	builder.WriteString(fmt.Sprintf("- Name: %s\n", dashboard.Name))
	builder.WriteString(fmt.Sprintf("- Period: %s\n", dashboard.Period))
	builder.WriteString(fmt.Sprintf("- Owner: %s\n", dashboard.Owner))
	builder.WriteString(fmt.Sprintf("- Viewer Role: %s\n", dashboard.Permissions.ViewerRole))
	builder.WriteString(fmt.Sprintf("- Available Widgets: %d\n", len(dashboard.Widgets)))
	builder.WriteString(fmt.Sprintf("- Layouts: %d\n", len(dashboard.Layouts)))
	builder.WriteString(fmt.Sprintf("- Release Ready: %t\n\n", audit.ReleaseReady()))
	builder.WriteString("## Governance\n\n")
	builder.WriteString(fmt.Sprintf("- Documentation Complete: %t\n", audit.DocumentationComplete))
	builder.WriteString(fmt.Sprintf("- Duplicate Placement IDs: %s\n", joinOrNone(audit.DuplicatePlacementIDs)))
	builder.WriteString(fmt.Sprintf("- Missing Widget Definitions: %s\n", joinOrNone(audit.MissingWidgetDefs)))
	builder.WriteString(fmt.Sprintf("- Inaccessible Widgets: %s\n", joinOrNone(audit.InaccessibleWidgets)))
	builder.WriteString(fmt.Sprintf("- Overlaps: %s\n", joinOrNone(audit.OverlappingPlacements)))
	builder.WriteString(fmt.Sprintf("- Out Of Bounds: %s\n", joinOrNone(audit.OutOfBoundsPlacements)))
	builder.WriteString(fmt.Sprintf("- Empty Layouts: %s\n\n", joinOrNone(audit.EmptyLayouts)))
	builder.WriteString("## Layouts\n\n")

	widgetIndex := dashboard.WidgetIndex()
	if len(dashboard.Layouts) == 0 {
		builder.WriteString("- None\n")
		return builder.String()
	}
	for _, layout := range dashboard.Layouts {
		builder.WriteString(fmt.Sprintf("- %s: name=%s columns=%d placements=%d\n", layout.LayoutID, layout.Name, layout.Columns, len(layout.Placements)))
		for _, placement := range layout.Placements {
			title := placement.TitleOverride
			if strings.TrimSpace(title) == "" {
				if widget, ok := widgetIndex[placement.WidgetID]; ok {
					title = widget.Title
				} else {
					title = placement.WidgetID
				}
			}
			builder.WriteString(fmt.Sprintf("- %s: widget=%s title=%s grid=(%d,%d) size=%dx%d filters=%s\n", placement.PlacementID, placement.WidgetID, title, placement.Column, placement.Row, placement.Width, placement.Height, joinOrNone(placement.Filters)))
		}
	}
	return builder.String()
}

func WriteDashboardBuilderBundle(rootDir string, dashboard DashboardBuilder, audit DashboardBuilderAudit) (string, error) {
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(rootDir, "dashboard-builder.md")
	if err := WriteReport(path, RenderDashboardBuilderReport(dashboard, audit)); err != nil {
		return "", err
	}
	return path, nil
}

func buildHighlights(weekly Weekly) []string {
	highlights := []string{
		fmt.Sprintf("Completed %d / %d runs this week.", weekly.Summary.CompletedRuns, weekly.Summary.TotalRuns),
		fmt.Sprintf("Observed %d human interventions across active delivery lanes.", weekly.Summary.HumanInterventions),
	}
	if len(weekly.TeamBreakdown) > 0 {
		highlights = append(highlights, fmt.Sprintf("Top team by throughput: %s.", weekly.TeamBreakdown[0].Key))
	}
	return highlights
}

func buildActions(weekly Weekly) []string {
	actions := make([]string, 0)
	if weekly.Summary.BlockedRuns > 0 {
		actions = append(actions, "Reduce blocked flow count by resolving the top blocker owners first.")
	}
	if weekly.Summary.RegressionFindings > 0 {
		actions = append(actions, "Review regression hotspots and route them through the regression center.")
	}
	if weekly.Summary.HumanInterventions > 0 {
		actions = append(actions, "Audit repeated manual takeovers and convert them into policy or workflow fixes.")
	}
	if len(actions) == 0 {
		actions = append(actions, "No urgent actions detected; maintain current operating cadence.")
	}
	return actions
}

func interventionCounts(events []domain.Event) map[string]int {
	out := make(map[string]int)
	for _, event := range events {
		switch event.Type {
		case domain.EventRunTakeover, domain.EventRunReleased, domain.EventRunAnnotated, domain.EventControlPaused, domain.EventControlResumed:
			if event.TaskID != "" {
				out[event.TaskID]++
			}
		}
	}
	return out
}

func within(anchor time.Time, start time.Time, end time.Time) bool {
	if anchor.IsZero() {
		return false
	}
	if !start.IsZero() && anchor.Before(start) {
		return false
	}
	if !end.IsZero() && anchor.After(end) {
		return false
	}
	return true
}

func regressionCount(task domain.Task) int {
	for _, key := range []string{"regression_count", "regressions"} {
		if value := strings.TrimSpace(task.Metadata[key]); value != "" {
			if parsed, err := strconv.Atoi(value); err == nil {
				return parsed
			}
		}
	}
	if strings.EqualFold(strings.TrimSpace(task.Metadata["regression"]), "true") {
		return 1
	}
	return 0
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func formatMetricValue(value float64) string {
	if value == float64(int64(value)) {
		return strconv.FormatInt(int64(value), 10)
	}
	return strconv.FormatFloat(value, 'f', 1, 64)
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func sortedKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func placementsOverlap(left DashboardWidgetPlacement, right DashboardWidgetPlacement) bool {
	leftRight := left.Column + left.Width
	rightRight := right.Column + right.Width
	leftBottom := left.Row + left.Height
	rightBottom := right.Row + right.Height

	return left.Column < rightRight &&
		leftRight > right.Column &&
		left.Row < rightBottom &&
		leftBottom > right.Row
}
