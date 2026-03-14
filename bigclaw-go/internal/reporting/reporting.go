package reporting

import (
	"fmt"
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
	builder.WriteString(fmt.Sprintf("- Human interventions: %d\n", weekly.Summary.HumanInterventions))
	builder.WriteString(fmt.Sprintf("- Regressions: %d\n", weekly.Summary.RegressionFindings))
	builder.WriteString(fmt.Sprintf("- Budget cents: %d\n\n", weekly.Summary.BudgetCentsTotal))
	builder.WriteString("## Actions\n")
	for _, action := range weekly.Actions {
		builder.WriteString("- " + action + "\n")
	}
	return builder.String()
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
