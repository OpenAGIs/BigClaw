package risk

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"bigclaw-go/internal/domain"
)

type Factor struct {
	Name   string `json:"name"`
	Points int    `json:"points"`
	Reason string `json:"reason"`
}

type Score struct {
	Level            domain.RiskLevel `json:"level"`
	Total            int              `json:"total"`
	RequiresApproval bool             `json:"requires_approval"`
	Summary          string           `json:"summary"`
	Factors          []Factor         `json:"factors,omitempty"`
}

var toolPoints = map[string]int{
	"browser":   10,
	"terminal":  15,
	"shell":     15,
	"github":    10,
	"deploy":    20,
	"sql":       15,
	"warehouse": 15,
	"bi":        10,
	"vm":        15,
}

var labelPoints = map[string]int{
	"security":   20,
	"compliance": 20,
	"prod":       20,
	"release":    15,
	"ops":        10,
}

var codeImpactPoints = map[string]int{
	"medium":   10,
	"high":     20,
	"critical": 25,
}

func ScoreTask(task domain.Task, events []domain.Event) Score {
	factors := make([]Factor, 0)
	total := 0
	add := func(name string, points int, reason string) {
		if points <= 0 {
			return
		}
		total += points
		factors = append(factors, Factor{Name: name, Points: points, Reason: reason})
	}

	switch task.RiskLevel {
	case domain.RiskMedium:
		add("risk_level", 30, "declared risk level medium")
	case domain.RiskHigh:
		add("risk_level", 60, "declared risk level high")
	}

	switch {
	case task.Priority > 0 && task.Priority <= 1:
		add("priority", 10, "p0 task needs tighter controls")
	case task.Priority == 2:
		add("priority", 5, "high-priority task increases operational pressure")
	}

	labels := make([]string, 0)
	seenLabels := make(map[string]struct{})
	for _, label := range task.Labels {
		normalized := strings.ToLower(strings.TrimSpace(label))
		if normalized == "" {
			continue
		}
		if _, ok := seenLabels[normalized]; ok {
			continue
		}
		seenLabels[normalized] = struct{}{}
		labels = append(labels, normalized)
	}
	sort.Strings(labels)
	for _, label := range labels {
		if points := labelPoints[label]; points > 0 {
			add("label:"+label, points, fmt.Sprintf("label %s increases operational risk", label))
		}
	}

	tools := make([]string, 0)
	seenTools := make(map[string]struct{})
	for _, tool := range task.RequiredTools {
		normalized := strings.ToLower(strings.TrimSpace(tool))
		if normalized == "" {
			continue
		}
		if _, ok := seenTools[normalized]; ok {
			continue
		}
		seenTools[normalized] = struct{}{}
		tools = append(tools, normalized)
	}
	sort.Strings(tools)
	for _, tool := range tools {
		if points := toolPoints[tool]; points > 0 {
			add("tool:"+tool, points, fmt.Sprintf("tool %s expands execution surface", tool))
		}
	}

	if codeImpact := metadataString(task, "code_impact", "impact_scope", "change_scope"); codeImpact != "" {
		normalized := strings.ToLower(codeImpact)
		if points := codeImpactPoints[normalized]; points > 0 {
			add("code_impact", points, fmt.Sprintf("code impact %s expands blast radius", normalized))
		}
	}

	if task.BudgetCents < 0 {
		add("budget", 20, "invalid budget requires manual review")
	}

	if changedFiles := metadataInt(task, "changed_files_count", "files_changed_count", "code_files_changed"); changedFiles >= 50 {
		add("changed_files", 20, "large code delta expands review scope")
	} else if changedFiles >= 10 {
		add("changed_files", 10, "multi-file change expands review scope")
	}

	failureCount := metadataInt(task, "failure_count", "historical_failures") + countFailureEvents(events)
	if failureCount > 0 {
		points := failureCount * 5
		if points > 15 {
			points = 15
		}
		add("failure_history", points, fmt.Sprintf("%d prior failures on this task/run", failureCount))
	}

	retryCount := metadataInt(task, "retry_count", "historical_retries") + countEventType(events, domain.EventTaskRetried)
	if retryCount > 0 {
		points := retryCount * 5
		if points > 15 {
			points = 15
		}
		add("retry_history", points, fmt.Sprintf("%d retries indicate unstable execution", retryCount))
	}

	regressionCount := metadataInt(task, "regression_count", "historical_regressions")
	if regressionCount > 0 {
		points := regressionCount * 10
		if points > 20 {
			points = 20
		}
		add("regression_history", points, fmt.Sprintf("%d regressions require tighter oversight", regressionCount))
	}

	explicitApproval := metadataBool(task, "approval_required", "requires_approval")
	if explicitApproval {
		add("approval_override", 10, "task metadata requires manual approval")
	}

	level := levelForTotal(total)
	requiresApproval := explicitApproval || level == domain.RiskHigh
	return Score{
		Level:            level,
		Total:            total,
		RequiresApproval: requiresApproval,
		Summary:          summarizeFactors(factors),
		Factors:          factors,
	}
}

func levelForTotal(total int) domain.RiskLevel {
	if total >= 60 {
		return domain.RiskHigh
	}
	if total >= 25 {
		return domain.RiskMedium
	}
	return domain.RiskLow
}

func summarizeFactors(factors []Factor) string {
	if len(factors) == 0 {
		return "baseline=0"
	}
	parts := make([]string, 0, len(factors))
	for _, factor := range factors {
		parts = append(parts, fmt.Sprintf("%s=%d", factor.Name, factor.Points))
	}
	return strings.Join(parts, ", ")
}

func countFailureEvents(events []domain.Event) int {
	count := 0
	for _, event := range events {
		switch event.Type {
		case domain.EventTaskDeadLetter, domain.EventTaskCancelled:
			count++
		}
	}
	return count
}

func countEventType(events []domain.Event, want domain.EventType) int {
	count := 0
	for _, event := range events {
		if event.Type == want {
			count++
		}
	}
	return count
}

func metadataString(task domain.Task, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(task.Metadata[key]); value != "" {
			return value
		}
	}
	return ""
}

func metadataInt(task domain.Task, keys ...string) int {
	for _, key := range keys {
		if value := strings.TrimSpace(task.Metadata[key]); value != "" {
			if parsed, err := strconv.Atoi(value); err == nil {
				return parsed
			}
		}
	}
	return 0
}

func metadataBool(task domain.Task, keys ...string) bool {
	for _, key := range keys {
		if value := strings.TrimSpace(task.Metadata[key]); value != "" {
			if parsed, err := strconv.ParseBool(value); err == nil {
				return parsed
			}
		}
	}
	return false
}
