package reporting

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/regression"
	"github.com/pmezard/go-difflib/difflib"
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
	RootDir              string `json:"root_dir"`
	WeeklyReportPath     string `json:"weekly_report_path"`
	DashboardPath        string `json:"dashboard_path"`
	MetricSpecPath       string `json:"metric_spec_path,omitempty"`
	RegressionCenterPath string `json:"regression_center_path,omitempty"`
	QueueControlPath     string `json:"queue_control_path,omitempty"`
	VersionCenterPath    string `json:"version_center_path,omitempty"`
}

type RepoCollaborationMetrics struct {
	RepoLinkCoverage        float64 `json:"repo_link_coverage"`
	AcceptedCommitRate      float64 `json:"accepted_commit_rate"`
	DiscussionDensity       float64 `json:"discussion_density"`
	AcceptedLineageDepthAvg float64 `json:"accepted_lineage_depth_avg"`
}

type ConsoleAction struct {
	ActionID string `json:"action_id"`
	Label    string `json:"label"`
	Target   string `json:"target"`
	Enabled  bool   `json:"enabled"`
	Reason   string `json:"reason,omitempty"`
}

func (a ConsoleAction) State() string {
	if a.Enabled {
		return "enabled"
	}
	return "disabled"
}

type QueueControlCenter struct {
	QueueDepth          int                        `json:"queue_depth"`
	QueuedByPriority    map[string]int             `json:"queued_by_priority,omitempty"`
	QueuedByRisk        map[string]int             `json:"queued_by_risk,omitempty"`
	ExecutionMedia      map[string]int             `json:"execution_media,omitempty"`
	WaitingApprovalRuns int                        `json:"waiting_approval_runs"`
	BlockedTasks        []string                   `json:"blocked_tasks,omitempty"`
	QueuedTasks         []string                   `json:"queued_tasks,omitempty"`
	Actions             map[string][]ConsoleAction `json:"actions,omitempty"`
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

func NormalizeDashboardLayout(layout DashboardLayout, widgets []DashboardWidgetSpec) DashboardLayout {
	widgetIndex := make(map[string]DashboardWidgetSpec, len(widgets))
	for _, widget := range widgets {
		widgetIndex[widget.WidgetID] = widget
	}

	columnCount := layout.Columns
	if columnCount <= 0 {
		columnCount = 12
	}
	normalized := make([]DashboardWidgetPlacement, 0, len(layout.Placements))
	for _, placement := range layout.Placements {
		minWidth := 1
		maxWidth := columnCount
		if spec, ok := widgetIndex[placement.WidgetID]; ok {
			minWidth = spec.MinWidth
			if minWidth <= 0 {
				minWidth = 1
			}
			if spec.MaxWidth > 0 {
				maxWidth = minInt(spec.MaxWidth, columnCount)
			}
		}
		if maxWidth < minWidth {
			maxWidth = minWidth
		}

		width := maxInt(minWidth, minInt(placement.Width, maxWidth))
		column := maxInt(0, placement.Column)
		if column+width > columnCount {
			column = maxInt(0, columnCount-width)
		}

		normalized = append(normalized, DashboardWidgetPlacement{
			PlacementID:   placement.PlacementID,
			WidgetID:      placement.WidgetID,
			Column:        column,
			Row:           maxInt(0, placement.Row),
			Width:         width,
			Height:        maxInt(1, placement.Height),
			TitleOverride: placement.TitleOverride,
			Filters:       append([]string(nil), placement.Filters...),
		})
	}

	sort.SliceStable(normalized, func(i, j int) bool {
		if normalized[i].Row == normalized[j].Row {
			if normalized[i].Column == normalized[j].Column {
				return normalized[i].PlacementID < normalized[j].PlacementID
			}
			return normalized[i].Column < normalized[j].Column
		}
		return normalized[i].Row < normalized[j].Row
	})

	return DashboardLayout{
		LayoutID:   layout.LayoutID,
		Name:       layout.Name,
		Columns:    columnCount,
		Placements: normalized,
	}
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

type EngineeringOverviewKPI struct {
	Name      string  `json:"name"`
	Value     float64 `json:"value"`
	Target    float64 `json:"target"`
	Unit      string  `json:"unit,omitempty"`
	Direction string  `json:"direction,omitempty"`
}

func (k EngineeringOverviewKPI) Healthy() bool {
	if strings.EqualFold(strings.TrimSpace(k.Direction), "down") {
		return k.Value <= k.Target
	}
	return k.Value >= k.Target
}

type EngineeringFunnelStage struct {
	Name  string  `json:"name"`
	Count int     `json:"count"`
	Share float64 `json:"share"`
}

type EngineeringOverviewBlocker struct {
	Summary       string   `json:"summary"`
	AffectedRuns  int      `json:"affected_runs"`
	AffectedTasks []string `json:"affected_tasks,omitempty"`
	Owner         string   `json:"owner,omitempty"`
	Severity      string   `json:"severity,omitempty"`
}

type EngineeringActivity struct {
	Timestamp string `json:"timestamp"`
	RunID     string `json:"run_id,omitempty"`
	TaskID    string `json:"task_id,omitempty"`
	Status    string `json:"status"`
	Summary   string `json:"summary"`
}

type EngineeringOverview struct {
	Name        string                        `json:"name"`
	Period      string                        `json:"period"`
	Permissions EngineeringOverviewPermission `json:"permissions"`
	KPIs        []EngineeringOverviewKPI      `json:"kpis,omitempty"`
	Funnel      []EngineeringFunnelStage      `json:"funnel,omitempty"`
	Blockers    []EngineeringOverviewBlocker  `json:"blockers,omitempty"`
	Activities  []EngineeringActivity         `json:"activities,omitempty"`
}

type VersionedArtifact struct {
	ArtifactType string `json:"artifact_type"`
	ArtifactID   string `json:"artifact_id"`
	Version      string `json:"version"`
	UpdatedAt    string `json:"updated_at"`
	Author       string `json:"author"`
	Summary      string `json:"summary"`
	Content      string `json:"content"`
	ChangeTicket string `json:"change_ticket,omitempty"`
}

type VersionChangeSummary struct {
	FromVersion  string   `json:"from_version"`
	ToVersion    string   `json:"to_version"`
	Additions    int      `json:"additions"`
	Deletions    int      `json:"deletions"`
	ChangedLines int      `json:"changed_lines"`
	Preview      []string `json:"preview,omitempty"`
}

func (s VersionChangeSummary) HasChanges() bool {
	return s.ChangedLines > 0
}

type VersionedArtifactHistory struct {
	ArtifactType     string                `json:"artifact_type"`
	ArtifactID       string                `json:"artifact_id"`
	CurrentVersion   string                `json:"current_version"`
	CurrentUpdatedAt string                `json:"current_updated_at"`
	CurrentAuthor    string                `json:"current_author"`
	CurrentSummary   string                `json:"current_summary"`
	RevisionCount    int                   `json:"revision_count"`
	Revisions        []VersionedArtifact   `json:"revisions,omitempty"`
	RollbackVersion  string                `json:"rollback_version,omitempty"`
	RollbackReady    bool                  `json:"rollback_ready"`
	ChangeSummary    *VersionChangeSummary `json:"change_summary,omitempty"`
}

type PolicyPromptVersionCenter struct {
	Name        string                     `json:"name"`
	GeneratedAt string                     `json:"generated_at"`
	Histories   []VersionedArtifactHistory `json:"histories,omitempty"`
}

func (c PolicyPromptVersionCenter) ArtifactCount() int {
	return len(c.Histories)
}

func (c PolicyPromptVersionCenter) RollbackReadyCount() int {
	count := 0
	for _, history := range c.Histories {
		if history.RollbackReady {
			count++
		}
	}
	return count
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

func BuildOperationsMetricSpec(tasks []domain.Task, events []domain.Event, periodStart, periodEnd time.Time, timezoneName string, slaTargetMinutes int) OperationsMetricSpec {
	if timezoneName == "" {
		timezoneName = "UTC"
	}
	if slaTargetMinutes <= 0 {
		slaTargetMinutes = 60
	}
	interventions := interventionCounts(events)
	totalRuns := len(tasks)
	runsInWindow := 0
	cycleSum := 0.0
	cycleCount := 0
	slaCompliantRuns := 0
	intervenedRuns := 0
	regressionFindings := 0
	riskSum := 0.0
	riskCount := 0
	budgetUSDTotal := 0.0

	for _, task := range tasks {
		if within(task.CreatedAt, periodStart, periodEnd) || within(task.UpdatedAt, periodStart, periodEnd) {
			runsInWindow++
		}
		if cycle, ok := cycleMinutes(task); ok {
			cycleSum += cycle
			cycleCount++
			if cycle <= float64(slaTargetMinutes) {
				slaCompliantRuns++
			}
		}
		if interventions[task.ID] > 0 || strings.EqualFold(strings.TrimSpace(task.Metadata["approval_status"]), "needs-approval") {
			intervenedRuns++
		}
		regressionFindings += regressionCount(task)
		if score, ok := riskScoreForTask(task); ok {
			riskSum += score
			riskCount++
		}
		budgetUSDTotal += float64(task.BudgetCents) / 100.0
	}

	avgCycle := 0.0
	if cycleCount > 0 {
		avgCycle = roundTenth(cycleSum / float64(cycleCount))
	}
	interventionRate := 0.0
	if totalRuns > 0 {
		interventionRate = roundTenth((float64(intervenedRuns) / float64(totalRuns)) * 100)
	}
	slaCompliance := 0.0
	if cycleCount > 0 {
		slaCompliance = roundTenth((float64(slaCompliantRuns) / float64(cycleCount)) * 100)
	}
	avgRisk := 0.0
	if riskCount > 0 {
		avgRisk = roundTenth(riskSum / float64(riskCount))
	}
	budgetUSDTotal = math.Round(budgetUSDTotal*100) / 100

	definitions := []OperationsMetricDefinition{
		{MetricID: "runs-window", Label: "Runs In Window", Unit: "runs", Direction: "up", Formula: "count(tasks.created_at|updated_at within period)", Description: "Number of runs active inside the reporting window.", SourceFields: []string{"task.created_at", "task.updated_at"}},
		{MetricID: "avg-cycle-minutes", Label: "Avg Cycle Minutes", Unit: "m", Direction: "down", Formula: "sum(updated_at - created_at) / measured_runs", Description: "Average measured run cycle time in minutes.", SourceFields: []string{"task.created_at", "task.updated_at"}},
		{MetricID: "intervention-rate", Label: "Intervention Rate", Unit: "%", Direction: "down", Formula: "intervened_runs / total_runs", Description: "Share of runs that required manual intervention or approval handling.", SourceFields: []string{"events", "task.metadata.approval_status"}},
		{MetricID: "sla-compliance", Label: "SLA Compliance", Unit: "%", Direction: "up", Formula: "runs within SLA target / measured_runs", Description: "Measured runs that completed inside the SLA target.", SourceFields: []string{"task.created_at", "task.updated_at"}},
		{MetricID: "regression-findings", Label: "Regression Findings", Unit: "findings", Direction: "down", Formula: "sum(task.metadata.regression_count)", Description: "Regression findings attached to the reporting window.", SourceFields: []string{"task.metadata.regression_count", "task.metadata.regression"}},
		{MetricID: "avg-risk-score", Label: "Avg Risk Score", Unit: "score", Direction: "down", Formula: "avg(mapped risk_level)", Description: "Average mapped risk score using low=25, medium=60, high=90.", SourceFields: []string{"task.risk_level"}},
		{MetricID: "budget-spend", Label: "Budget Spend", Unit: "USD", Direction: "down", Formula: "sum(task.budget_cents) / 100", Description: "Budget total represented by the reporting slice.", SourceFields: []string{"task.budget_cents"}},
	}

	return OperationsMetricSpec{
		Name:         "Operations Metric Spec",
		GeneratedAt:  time.Now().UTC(),
		PeriodStart:  periodStart.UTC(),
		PeriodEnd:    periodEnd.UTC(),
		TimezoneName: timezoneName,
		Definitions:  definitions,
		Values: []OperationsMetricValue{
			{MetricID: "runs-window", Label: "Runs In Window", Value: float64(runsInWindow), DisplayValue: strconv.Itoa(runsInWindow), Numerator: float64(runsInWindow), Denominator: float64(totalRuns), Unit: "runs", Evidence: []string{fmt.Sprintf("%d of %d runs were created or updated inside the reporting window.", runsInWindow, totalRuns)}},
			{MetricID: "avg-cycle-minutes", Label: "Avg Cycle Minutes", Value: avgCycle, DisplayValue: fmt.Sprintf("%.1fm", avgCycle), Numerator: roundTenth(cycleSum), Denominator: float64(cycleCount), Unit: "m", Evidence: []string{fmt.Sprintf("%d runs had valid created_at and updated_at timestamps.", cycleCount)}},
			{MetricID: "intervention-rate", Label: "Intervention Rate", Value: interventionRate, DisplayValue: fmt.Sprintf("%.1f%%", interventionRate), Numerator: float64(intervenedRuns), Denominator: float64(totalRuns), Unit: "%", Evidence: []string{fmt.Sprintf("%d runs had intervention events or approval-required status.", intervenedRuns)}},
			{MetricID: "sla-compliance", Label: "SLA Compliance", Value: slaCompliance, DisplayValue: fmt.Sprintf("%.1f%%", slaCompliance), Numerator: float64(slaCompliantRuns), Denominator: float64(cycleCount), Unit: "%", Evidence: []string{fmt.Sprintf("SLA target: %d minutes.", slaTargetMinutes), fmt.Sprintf("%d of %d measured runs met target.", slaCompliantRuns, cycleCount)}},
			{MetricID: "regression-findings", Label: "Regression Findings", Value: float64(regressionFindings), DisplayValue: strconv.Itoa(regressionFindings), Numerator: float64(regressionFindings), Denominator: float64(totalRuns), Unit: "findings", Evidence: []string{"Regression count uses task metadata fields `regression_count`, `regressions`, or boolean `regression`."}},
			{MetricID: "avg-risk-score", Label: "Avg Risk Score", Value: avgRisk, DisplayValue: fmt.Sprintf("%.1f", avgRisk), Numerator: roundTenth(riskSum), Denominator: float64(riskCount), Unit: "score", Evidence: []string{"Risk score mapping is low=25, medium=60, high=90."}},
			{MetricID: "budget-spend", Label: "Budget Spend", Value: budgetUSDTotal, DisplayValue: fmt.Sprintf("$%.2f", budgetUSDTotal), Numerator: budgetUSDTotal, Denominator: float64(totalRuns), Unit: "USD", Evidence: []string{"Budget spend is derived from `task.budget_cents`."}},
		},
	}
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

func WriteWeeklyOperationsBundleWithVersionCenter(rootDir string, weekly Weekly, metricSpec *OperationsMetricSpec, versionCenter *PolicyPromptVersionCenter) (WeeklyArtifacts, error) {
	return WriteWeeklyOperationsBundleWithCenters(rootDir, weekly, metricSpec, "", nil, nil, versionCenter)
}

func WriteWeeklyOperationsBundleWithCenters(rootDir string, weekly Weekly, metricSpec *OperationsMetricSpec, regressionName string, regressionCenter *regression.Center, queueControl *QueueControlCenter, versionCenter *PolicyPromptVersionCenter) (WeeklyArtifacts, error) {
	artifacts, err := WriteWeeklyOperationsBundle(rootDir, weekly, metricSpec)
	if err != nil {
		return WeeklyArtifacts{}, err
	}
	if regressionCenter != nil {
		artifacts.RegressionCenterPath = filepath.Join(rootDir, "regression-center.md")
		if err := WriteReport(artifacts.RegressionCenterPath, RenderRegressionCenter(regressionName, *regressionCenter)); err != nil {
			return WeeklyArtifacts{}, err
		}
	}
	if queueControl != nil {
		artifacts.QueueControlPath = filepath.Join(rootDir, "queue-control-center.md")
		if err := WriteReport(artifacts.QueueControlPath, RenderQueueControlCenter(*queueControl)); err != nil {
			return WeeklyArtifacts{}, err
		}
	}
	if versionCenter == nil {
		return artifacts, nil
	}
	artifacts.VersionCenterPath = filepath.Join(rootDir, "policy-prompt-version-center.md")
	if err := WriteReport(artifacts.VersionCenterPath, RenderPolicyPromptVersionCenter(*versionCenter)); err != nil {
		return WeeklyArtifacts{}, err
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

func BuildEngineeringOverview(name, period, viewerRole string, tasks []domain.Task, events []domain.Event, slaTargetMinutes, topNBlockers, recentActivityLimit int) EngineeringOverview {
	if slaTargetMinutes <= 0 {
		slaTargetMinutes = 60
	}
	if topNBlockers <= 0 {
		topNBlockers = 3
	}
	if recentActivityLimit <= 0 {
		recentActivityLimit = 5
	}

	statusCounts := make(map[domain.TaskState]int)
	completed := 0
	approvalQueueDepth := 0
	slaBreachCount := 0
	totalCycleMinutes := 0.0
	cycleCount := 0
	blockerGroups := make(map[string]*EngineeringOverviewBlocker)
	recentEvents := latestEventsByTask(events)

	for _, task := range tasks {
		statusCounts[task.State]++
		if task.State == domain.TaskSucceeded {
			completed++
		}
		if strings.EqualFold(task.Metadata["approval_status"], "needs-approval") {
			approvalQueueDepth++
		}
		if minutes, ok := cycleMinutes(task); ok {
			totalCycleMinutes += minutes
			cycleCount++
			if minutes > float64(slaTargetMinutes) {
				slaBreachCount++
			}
		}
		if blockerReason, blocked := blockerReason(task, recentEvents[task.ID]); blocked {
			entry := blockerGroups[blockerReason]
			if entry == nil {
				entry = &EngineeringOverviewBlocker{
					Summary:  blockerReason,
					Owner:    blockerOwner(blockerReason),
					Severity: blockerSeverity(task.State),
				}
				blockerGroups[blockerReason] = entry
			}
			entry.AffectedRuns++
			entry.AffectedTasks = append(entry.AffectedTasks, task.ID)
			if blockerSeverity(task.State) == "high" {
				entry.Severity = "high"
			}
		}
	}

	totalRuns := len(tasks)
	successRate := 0.0
	if totalRuns > 0 {
		successRate = roundTenth((float64(completed) / float64(totalRuns)) * 100)
	}
	averageCycleMinutes := 0.0
	if cycleCount > 0 {
		averageCycleMinutes = roundTenth(totalCycleMinutes / float64(cycleCount))
	}

	overview := EngineeringOverview{
		Name:        name,
		Period:      period,
		Permissions: permissionsForRole(viewerRole),
		KPIs: []EngineeringOverviewKPI{
			{Name: "success-rate", Value: successRate, Target: 90.0, Unit: "%", Direction: "up"},
			{Name: "approval-queue-depth", Value: float64(approvalQueueDepth), Target: 2.0, Direction: "down"},
			{Name: "sla-breaches", Value: float64(slaBreachCount), Target: 0.0, Direction: "down"},
			{Name: "average-cycle-minutes", Value: averageCycleMinutes, Target: float64(slaTargetMinutes), Unit: "m", Direction: "down"},
		},
		Funnel:     buildEngineeringFunnel(statusCounts, totalRuns),
		Blockers:   topBlockers(blockerGroups, topNBlockers),
		Activities: buildRecentActivities(tasks, recentEvents, recentActivityLimit),
	}
	return overview
}

func RenderEngineeringOverview(overview EngineeringOverview) string {
	builder := strings.Builder{}
	builder.WriteString("# Engineering Overview\n\n")
	builder.WriteString(fmt.Sprintf("- Name: %s\n", overview.Name))
	builder.WriteString(fmt.Sprintf("- Period: %s\n", overview.Period))
	builder.WriteString(fmt.Sprintf("- Viewer Role: %s\n", overview.Permissions.ViewerRole))
	builder.WriteString(fmt.Sprintf("- Visible Modules: %s\n", joinOrNone(overview.Permissions.AllowedModules)))

	if overview.Permissions.CanView("kpis") {
		builder.WriteString("\n## KPI Modules\n\n")
		if len(overview.KPIs) == 0 {
			builder.WriteString("- None\n")
		} else {
			for _, kpi := range overview.KPIs {
				builder.WriteString(fmt.Sprintf("- %s: value=%.1f%s target=%.1f%s healthy=%t\n", kpi.Name, kpi.Value, kpi.Unit, kpi.Target, kpi.Unit, kpi.Healthy()))
			}
		}
	}

	if overview.Permissions.CanView("funnel") {
		builder.WriteString("\n## Funnel Modules\n\n")
		if len(overview.Funnel) == 0 {
			builder.WriteString("- None\n")
		} else {
			for _, stage := range overview.Funnel {
				builder.WriteString(fmt.Sprintf("- %s: count=%d share=%.1f%%\n", stage.Name, stage.Count, stage.Share))
			}
		}
	}

	if overview.Permissions.CanView("blockers") {
		builder.WriteString("\n## Blocker Modules\n\n")
		if len(overview.Blockers) == 0 {
			builder.WriteString("- None\n")
		} else {
			for _, blocker := range overview.Blockers {
				builder.WriteString(fmt.Sprintf("- %s: severity=%s owner=%s affected_runs=%d tasks=%s\n", blocker.Summary, blocker.Severity, blocker.Owner, blocker.AffectedRuns, joinOrNone(blocker.AffectedTasks)))
			}
		}
	}

	if overview.Permissions.CanView("activity") {
		builder.WriteString("\n## Activity Modules\n\n")
		if len(overview.Activities) == 0 {
			builder.WriteString("- None\n")
		} else {
			for _, activity := range overview.Activities {
				builder.WriteString(fmt.Sprintf("- %s: %s task=%s status=%s summary=%s\n", activity.Timestamp, firstNonEmpty(activity.RunID, "n/a"), firstNonEmpty(activity.TaskID, "n/a"), activity.Status, activity.Summary))
			}
		}
	}

	return builder.String()
}

func WriteEngineeringOverviewBundle(rootDir string, overview EngineeringOverview) (string, error) {
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(rootDir, "engineering-overview.md")
	if err := WriteReport(path, RenderEngineeringOverview(overview)); err != nil {
		return "", err
	}
	return path, nil
}

func BuildRepoCollaborationMetrics(runs []map[string]any) RepoCollaborationMetrics {
	total := len(runs)
	linked := 0
	accepted := 0
	discussionPosts := 0.0
	lineageDepthSum := 0.0
	lineageDepthCount := 0

	for _, run := range runs {
		if closeout, ok := run["closeout"].(map[string]any); ok {
			if links, ok := closeout["run_commit_links"].([]any); ok && len(links) > 0 {
				linked++
			}
			if strings.TrimSpace(anyString(closeout["accepted_commit_hash"])) != "" {
				accepted++
			}
		}
		discussionPosts += anyFloat(run["repo_discussion_posts"])
		if depth, ok := anyFloatOK(run["accepted_lineage_depth"]); ok {
			lineageDepthSum += depth
			lineageDepthCount++
		}
	}

	metrics := RepoCollaborationMetrics{}
	if total > 0 {
		metrics.RepoLinkCoverage = roundTo((float64(linked)/float64(total))*100, 1)
		metrics.AcceptedCommitRate = roundTo((float64(accepted)/float64(total))*100, 1)
		metrics.DiscussionDensity = roundTo(discussionPosts/float64(total), 2)
	}
	if lineageDepthCount > 0 {
		metrics.AcceptedLineageDepthAvg = roundTo(lineageDepthSum/float64(lineageDepthCount), 2)
	}
	return metrics
}

func BuildQueueControlCenter(tasks []domain.Task) QueueControlCenter {
	center := QueueControlCenter{
		QueuedByPriority: map[string]int{"P0": 0, "P1": 0, "P2": 0},
		QueuedByRisk:     map[string]int{"low": 0, "medium": 0, "high": 0},
		ExecutionMedia:   make(map[string]int),
		Actions:          make(map[string][]ConsoleAction),
	}
	for _, task := range tasks {
		if domain.IsActiveTaskState(task.State) {
			center.QueueDepth++
		}
		if task.State == domain.TaskBlocked || strings.EqualFold(task.Metadata["approval_status"], "needs-approval") {
			center.WaitingApprovalRuns++
			center.BlockedTasks = append(center.BlockedTasks, task.ID)
		}
		if task.State == domain.TaskQueued || task.State == domain.TaskLeased || task.State == domain.TaskRetrying {
			center.QueuedTasks = append(center.QueuedTasks, task.ID)
			center.QueuedByPriority[priorityBucket(task.Priority)]++
			center.QueuedByRisk[riskBucket(task.RiskLevel)]++
			medium := firstNonEmpty(string(task.RequiredExecutor), task.Metadata["medium"], "unknown")
			center.ExecutionMedia[medium]++
			center.Actions[task.ID] = buildConsoleActions(task.ID, task.State == domain.TaskBlocked, task.State != domain.TaskBlocked, task.State == domain.TaskBlocked)
		}
	}
	sort.Strings(center.BlockedTasks)
	sort.Strings(center.QueuedTasks)
	return center
}

func RenderQueueControlCenter(center QueueControlCenter) string {
	builder := strings.Builder{}
	builder.WriteString("# Queue Control Center\n\n")
	builder.WriteString(fmt.Sprintf("- Queue Depth: %d\n", center.QueueDepth))
	builder.WriteString(fmt.Sprintf("- Waiting Approval Runs: %d\n", center.WaitingApprovalRuns))
	builder.WriteString(fmt.Sprintf("- Queued Tasks: %s\n\n", joinOrNone(center.QueuedTasks)))
	builder.WriteString("## Queue By Priority\n\n")
	for _, priority := range []string{"P0", "P1", "P2"} {
		builder.WriteString(fmt.Sprintf("- %s: %d\n", priority, center.QueuedByPriority[priority]))
	}
	builder.WriteString("\n## Queue By Risk\n\n")
	for _, risk := range []string{"low", "medium", "high"} {
		builder.WriteString(fmt.Sprintf("- %s: %d\n", risk, center.QueuedByRisk[risk]))
	}
	builder.WriteString("\n## Execution Media\n\n")
	if len(center.ExecutionMedia) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, medium := range sortedMapKeys(center.ExecutionMedia) {
			builder.WriteString(fmt.Sprintf("- %s: %d\n", medium, center.ExecutionMedia[medium]))
		}
	}
	builder.WriteString("\n## Blocked Tasks\n\n")
	if len(center.BlockedTasks) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, taskID := range center.BlockedTasks {
			builder.WriteString(fmt.Sprintf("- %s\n", taskID))
		}
	}
	builder.WriteString("\n## Actions\n\n")
	if len(center.QueuedTasks) == 0 {
		builder.WriteString("- None\n")
		return builder.String()
	}
	for _, taskID := range center.QueuedTasks {
		builder.WriteString(fmt.Sprintf("- %s: %s\n", taskID, RenderConsoleActions(center.Actions[taskID])))
	}
	return builder.String()
}

func WriteQueueControlCenterBundle(rootDir string, center QueueControlCenter) (string, error) {
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(rootDir, "queue-control-center.md")
	if err := WriteReport(path, RenderQueueControlCenter(center)); err != nil {
		return "", err
	}
	return path, nil
}

func BuildPolicyPromptVersionCenter(name string, generatedAt time.Time, artifacts []VersionedArtifact, diffPreviewLines int) PolicyPromptVersionCenter {
	grouped := make(map[string][]VersionedArtifact)
	for _, artifact := range artifacts {
		key := artifact.ArtifactType + "\x00" + artifact.ArtifactID
		grouped[key] = append(grouped[key], artifact)
	}
	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	histories := make([]VersionedArtifactHistory, 0, len(keys))
	for _, key := range keys {
		revisions := append([]VersionedArtifact(nil), grouped[key]...)
		sort.SliceStable(revisions, func(i, j int) bool {
			left := parseRFC3339ish(revisions[i].UpdatedAt)
			right := parseRFC3339ish(revisions[j].UpdatedAt)
			if left.Equal(right) {
				if revisions[i].Version == revisions[j].Version {
					return revisions[i].Summary < revisions[j].Summary
				}
				return revisions[i].Version > revisions[j].Version
			}
			return left.After(right)
		})
		current := revisions[0]
		history := VersionedArtifactHistory{
			ArtifactType:     current.ArtifactType,
			ArtifactID:       current.ArtifactID,
			CurrentVersion:   current.Version,
			CurrentUpdatedAt: current.UpdatedAt,
			CurrentAuthor:    current.Author,
			CurrentSummary:   current.Summary,
			RevisionCount:    len(revisions),
			Revisions:        revisions,
		}
		if len(revisions) > 1 {
			previous := revisions[1]
			history.RollbackVersion = previous.Version
			history.RollbackReady = strings.TrimSpace(previous.Content) != ""
			history.ChangeSummary = pointerToChangeSummary(summarizeVersionChange(previous, current, diffPreviewLines))
		}
		histories = append(histories, history)
	}
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	return PolicyPromptVersionCenter{
		Name:        name,
		GeneratedAt: generatedAt.UTC().Format(time.RFC3339),
		Histories:   histories,
	}
}

func RenderPolicyPromptVersionCenter(center PolicyPromptVersionCenter) string {
	builder := strings.Builder{}
	builder.WriteString("# Policy/Prompt Version Center\n\n")
	builder.WriteString(fmt.Sprintf("- Name: %s\n", center.Name))
	builder.WriteString(fmt.Sprintf("- Generated At: %s\n", center.GeneratedAt))
	builder.WriteString(fmt.Sprintf("- Versioned Artifacts: %d\n", center.ArtifactCount()))
	builder.WriteString(fmt.Sprintf("- Rollback Ready Artifacts: %d\n\n", center.RollbackReadyCount()))
	builder.WriteString("## Artifact Histories\n\n")
	if len(center.Histories) == 0 {
		builder.WriteString("- None\n")
		return builder.String()
	}
	for _, history := range center.Histories {
		builder.WriteString(fmt.Sprintf("### %s / %s\n\n", history.ArtifactType, history.ArtifactID))
		builder.WriteString(fmt.Sprintf("- Current Version: %s\n", history.CurrentVersion))
		builder.WriteString(fmt.Sprintf("- Updated At: %s\n", history.CurrentUpdatedAt))
		builder.WriteString(fmt.Sprintf("- Updated By: %s\n", history.CurrentAuthor))
		builder.WriteString(fmt.Sprintf("- Summary: %s\n", history.CurrentSummary))
		builder.WriteString(fmt.Sprintf("- Revision Count: %d\n", history.RevisionCount))
		builder.WriteString(fmt.Sprintf("- Rollback Version: %s\n", firstNonEmpty(history.RollbackVersion, "none")))
		builder.WriteString(fmt.Sprintf("- Rollback Ready: %t\n", history.RollbackReady))
		if history.ChangeSummary != nil {
			builder.WriteString(fmt.Sprintf("- Diff Summary: %d additions, %d deletions\n", history.ChangeSummary.Additions, history.ChangeSummary.Deletions))
		}
		builder.WriteString("\n#### Revision History\n\n")
		for _, revision := range history.Revisions {
			builder.WriteString(fmt.Sprintf("- %s: updated_at=%s author=%s ticket=%s summary=%s\n", revision.Version, revision.UpdatedAt, revision.Author, firstNonEmpty(revision.ChangeTicket, "none"), revision.Summary))
		}
		builder.WriteString("\n#### Diff Preview\n\n")
		if history.ChangeSummary != nil && len(history.ChangeSummary.Preview) > 0 {
			builder.WriteString("```diff\n")
			for _, line := range history.ChangeSummary.Preview {
				builder.WriteString(line + "\n")
			}
			builder.WriteString("```\n\n")
			continue
		}
		builder.WriteString("- None\n\n")
	}
	return builder.String()
}

func WritePolicyPromptVersionCenterBundle(rootDir string, center PolicyPromptVersionCenter) (string, error) {
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(rootDir, "policy-prompt-version-center.md")
	if err := WriteReport(path, RenderPolicyPromptVersionCenter(center)); err != nil {
		return "", err
	}
	return path, nil
}

func RenderRegressionCenter(name string, center regression.Center) string {
	if strings.TrimSpace(name) == "" {
		name = "Regression Analysis Center"
	}
	builder := strings.Builder{}
	builder.WriteString("# Regression Analysis Center\n\n")
	builder.WriteString(fmt.Sprintf("- Name: %s\n", name))
	builder.WriteString(fmt.Sprintf("- Regressions: %d\n", len(center.Findings)))
	builder.WriteString(fmt.Sprintf("- Total Regressions: %d\n", center.Summary.TotalRegressions))
	builder.WriteString(fmt.Sprintf("- Affected Tasks: %d\n", center.Summary.AffectedTasks))
	builder.WriteString(fmt.Sprintf("- Critical Regressions: %d\n", center.Summary.CriticalRegressions))
	builder.WriteString(fmt.Sprintf("- Rework Events: %d\n", center.Summary.ReworkEvents))
	builder.WriteString(fmt.Sprintf("- Top Source: %s\n", firstNonEmpty(center.Summary.TopSource, "none")))
	builder.WriteString(fmt.Sprintf("- Top Workflow: %s\n\n", firstNonEmpty(center.Summary.TopWorkflow, "none")))

	builder.WriteString("## Findings\n\n")
	if len(center.Findings) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, finding := range center.Findings {
			builder.WriteString(fmt.Sprintf("- %s: severity=%s regressions=%d rework=%d workflow=%s team=%s summary=%s\n", finding.TaskID, finding.Severity, finding.RegressionCount, finding.ReworkEvents, finding.Workflow, finding.Team, finding.Summary))
		}
	}

	builder.WriteString("\n## Hotspots\n\n")
	if len(center.Hotspots) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, hotspot := range center.Hotspots {
			builder.WriteString(fmt.Sprintf("- %s/%s: regressions=%d critical=%d rework=%d\n", hotspot.Dimension, hotspot.Key, hotspot.TotalRegressions, hotspot.CriticalRegressions, hotspot.ReworkEvents))
		}
	}

	builder.WriteString("\n## Workflow Breakdown\n\n")
	if len(center.WorkflowBreakdown) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, item := range center.WorkflowBreakdown {
			builder.WriteString(fmt.Sprintf("- %s: regressions=%d affected_tasks=%d critical=%d rework=%d\n", item.Key, item.TotalRegressions, item.AffectedTasks, item.CriticalRegressions, item.ReworkEvents))
		}
	}

	return builder.String()
}

func WriteRegressionCenterBundle(rootDir, name string, center regression.Center) (string, error) {
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(rootDir, "regression-center.md")
	if err := WriteReport(path, RenderRegressionCenter(name, center)); err != nil {
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

func sortedMapKeys(values map[string]int) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func buildConsoleActions(target string, allowRetry, allowPause, allowEscalate bool) []ConsoleAction {
	return []ConsoleAction{
		{ActionID: "drill-down", Label: "Drill Down", Target: target, Enabled: true},
		{ActionID: "export", Label: "Export", Target: target, Enabled: true},
		{ActionID: "add-note", Label: "Add Note", Target: target, Enabled: true},
		{ActionID: "escalate", Label: "Escalate", Target: target, Enabled: allowEscalate, Reason: disabledReason(allowEscalate, "escalate is reserved for blocked queue items")},
		{ActionID: "retry", Label: "Retry", Target: target, Enabled: allowRetry, Reason: disabledReason(allowRetry, "retry is reserved for blocked queue items")},
		{ActionID: "pause", Label: "Pause", Target: target, Enabled: allowPause, Reason: disabledReason(allowPause, "approval-blocked tasks should be escalated instead of paused")},
		{ActionID: "audit", Label: "Audit Trail", Target: target, Enabled: true},
	}
}

func RenderConsoleActions(actions []ConsoleAction) string {
	if len(actions) == 0 {
		return "none"
	}
	rendered := make([]string, 0, len(actions))
	for _, action := range actions {
		detail := fmt.Sprintf("%s [%s] state=%s target=%s", action.Label, action.ActionID, action.State(), action.Target)
		if reason := strings.TrimSpace(action.Reason); reason != "" {
			detail += " reason=" + reason
		}
		rendered = append(rendered, detail)
	}
	return strings.Join(rendered, "; ")
}

func disabledReason(enabled bool, reason string) string {
	if enabled {
		return ""
	}
	return reason
}

func priorityBucket(priority int) string {
	switch {
	case priority <= 0:
		return "P0"
	case priority == 1:
		return "P1"
	default:
		return "P2"
	}
}

func riskBucket(level domain.RiskLevel) string {
	switch level {
	case domain.RiskHigh:
		return "high"
	case domain.RiskMedium:
		return "medium"
	default:
		return "low"
	}
}

func riskScoreForTask(task domain.Task) (float64, bool) {
	switch task.RiskLevel {
	case domain.RiskHigh:
		return 90, true
	case domain.RiskMedium:
		return 60, true
	case domain.RiskLow:
		return 25, true
	default:
		return 0, false
	}
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

func permissionsForRole(viewerRole string) EngineeringOverviewPermission {
	role := strings.ToLower(strings.TrimSpace(viewerRole))
	if role == "" {
		role = "contributor"
	}
	modulesByRole := map[string][]string{
		"executive":           {"kpis", "funnel", "blockers"},
		"engineering-manager": {"kpis", "funnel", "blockers", "activity"},
		"operations":          {"kpis", "funnel", "blockers", "activity"},
		"contributor":         {"kpis", "activity"},
	}
	modules, ok := modulesByRole[role]
	if !ok {
		modules = modulesByRole["contributor"]
	}
	return EngineeringOverviewPermission{
		ViewerRole:     role,
		AllowedModules: append([]string(nil), modules...),
	}
}

func buildEngineeringFunnel(statusCounts map[domain.TaskState]int, totalRuns int) []EngineeringFunnelStage {
	stages := []EngineeringFunnelStage{
		{Name: "queued", Count: statusCounts[domain.TaskQueued]},
		{Name: "in-progress", Count: statusCounts[domain.TaskRunning] + statusCounts[domain.TaskLeased] + statusCounts[domain.TaskRetrying]},
		{Name: "awaiting-approval", Count: statusCounts[domain.TaskBlocked]},
		{Name: "completed", Count: statusCounts[domain.TaskSucceeded]},
	}
	for index := range stages {
		if totalRuns > 0 {
			stages[index].Share = roundTenth((float64(stages[index].Count) / float64(totalRuns)) * 100)
		}
	}
	return stages
}

func latestEventsByTask(events []domain.Event) map[string]domain.Event {
	out := make(map[string]domain.Event)
	for _, event := range events {
		if event.TaskID == "" {
			continue
		}
		existing, ok := out[event.TaskID]
		if !ok || event.Timestamp.After(existing.Timestamp) {
			out[event.TaskID] = event
		}
	}
	return out
}

func cycleMinutes(task domain.Task) (float64, bool) {
	if task.CreatedAt.IsZero() || task.UpdatedAt.IsZero() || task.UpdatedAt.Before(task.CreatedAt) {
		return 0, false
	}
	return roundTenth(task.UpdatedAt.Sub(task.CreatedAt).Minutes()), true
}

func blockerReason(task domain.Task, event domain.Event) (string, bool) {
	switch task.State {
	case domain.TaskBlocked, domain.TaskFailed, domain.TaskDeadLetter, domain.TaskCancelled:
	default:
		return "", false
	}
	if reason := firstNonEmpty(task.Metadata["blocked_reason"], task.Metadata["failure_reason"], task.Metadata["summary"]); reason != "" {
		return reason, true
	}
	if event.Payload != nil {
		if reason, ok := event.Payload["reason"].(string); ok && strings.TrimSpace(reason) != "" {
			return strings.TrimSpace(reason), true
		}
	}
	if strings.TrimSpace(task.Title) != "" {
		return task.Title, true
	}
	return string(task.State), true
}

func blockerOwner(reason string) string {
	details := strings.ToLower(reason)
	switch {
	case strings.Contains(details, "approval"):
		return "operations"
	case strings.Contains(details, "security"):
		return "security"
	default:
		return "engineering"
	}
}

func blockerSeverity(state domain.TaskState) string {
	switch state {
	case domain.TaskFailed, domain.TaskDeadLetter, domain.TaskCancelled:
		return "high"
	default:
		return "medium"
	}
}

func topBlockers(groups map[string]*EngineeringOverviewBlocker, limit int) []EngineeringOverviewBlocker {
	out := make([]EngineeringOverviewBlocker, 0, len(groups))
	for _, blocker := range groups {
		sort.Strings(blocker.AffectedTasks)
		out = append(out, *blocker)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].AffectedRuns == out[j].AffectedRuns {
			return out[i].Summary < out[j].Summary
		}
		return out[i].AffectedRuns > out[j].AffectedRuns
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func buildRecentActivities(tasks []domain.Task, latest map[string]domain.Event, limit int) []EngineeringActivity {
	sortedTasks := append([]domain.Task(nil), tasks...)
	sort.SliceStable(sortedTasks, func(i, j int) bool {
		if sortedTasks[i].UpdatedAt.Equal(sortedTasks[j].UpdatedAt) {
			return sortedTasks[i].ID < sortedTasks[j].ID
		}
		return sortedTasks[i].UpdatedAt.After(sortedTasks[j].UpdatedAt)
	})
	if limit > 0 && len(sortedTasks) > limit {
		sortedTasks = sortedTasks[:limit]
	}
	out := make([]EngineeringActivity, 0, len(sortedTasks))
	for _, task := range sortedTasks {
		runID := task.Metadata["run_id"]
		if event, ok := latest[task.ID]; ok && strings.TrimSpace(runID) == "" {
			runID = event.RunID
		}
		summary := firstNonEmpty(task.Metadata["summary"], task.Metadata["blocked_reason"], task.Title)
		if event, ok := latest[task.ID]; ok {
			if summary == "" {
				if reason, ok := event.Payload["reason"].(string); ok {
					summary = strings.TrimSpace(reason)
				}
			}
		}
		out = append(out, EngineeringActivity{
			Timestamp: task.UpdatedAt.UTC().Format(time.RFC3339),
			RunID:     runID,
			TaskID:    task.ID,
			Status:    string(task.State),
			Summary:   firstNonEmpty(summary, string(task.State)),
		})
	}
	return out
}

func anyString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func anyFloat(value any) float64 {
	out, _ := anyFloatOK(value)
	return out
}

func anyFloatOK(value any) (float64, bool) {
	switch typed := value.(type) {
	case int:
		return float64(typed), true
	case int8:
		return float64(typed), true
	case int16:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case uint:
		return float64(typed), true
	case uint8:
		return float64(typed), true
	case uint16:
		return float64(typed), true
	case uint32:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	case float32:
		return float64(typed), true
	case float64:
		return typed, true
	default:
		return 0, false
	}
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func roundTo(value float64, places int) float64 {
	if places < 0 {
		return value
	}
	factor := math.Pow10(places)
	return math.Round(value*factor) / factor
}

func roundTenth(value float64) float64 {
	return math.Round(value*10) / 10
}

func parseRFC3339ish(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err == nil {
		return parsed.UTC()
	}
	return time.Time{}
}

func summarizeVersionChange(previous, current VersionedArtifact, previewLines int) VersionChangeSummary {
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(previous.Content),
		B:        difflib.SplitLines(current.Content),
		FromFile: previous.Version,
		ToFile:   current.Version,
		Context:  1,
	}
	text, _ := difflib.GetUnifiedDiffString(diff)
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	additions := 0
	deletions := 0
	preview := make([]string, 0, previewLines)
	for _, line := range lines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			additions++
		}
		if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			deletions++
		}
		if strings.HasPrefix(line, "@@") {
			continue
		}
		if len(preview) < previewLines {
			preview = append(preview, line)
		}
	}
	return VersionChangeSummary{
		FromVersion:  previous.Version,
		ToVersion:    current.Version,
		Additions:    additions,
		Deletions:    deletions,
		ChangedLines: additions + deletions,
		Preview:      preview,
	}
}

func pointerToChangeSummary(summary VersionChangeSummary) *VersionChangeSummary {
	return &summary
}
