package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/reporting"
)

func runReporting(args []string) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		_, _ = os.Stdout.WriteString("usage: bigclawctl reporting <weekly> [flags]\n")
		return nil
	}
	switch args[0] {
	case "weekly":
		return runReportingWeeklyCommand(args[1:])
	default:
		return fmt.Errorf("unknown reporting subcommand: %s", args[0])
	}
}

func runReportingWeeklyCommand(args []string) error {
	flags := flag.NewFlagSet("reporting weekly", flag.ContinueOnError)
	repoRoot := flags.String("repo", defaultRepoRoot(), "repo root")
	tasksPath := flags.String("tasks", "", "path to a JSON array or {\"tasks\": [...]} payload")
	eventsPath := flags.String("events", "", "path to a JSON array or {\"events\": [...]} payload")
	outputDir := flags.String("output-dir", "", "directory for rendered report artifacts")
	weekStartRaw := flags.String("week-start", "", "week start in RFC3339 or YYYY-MM-DD")
	weekEndRaw := flags.String("week-end", "", "week end in RFC3339 or YYYY-MM-DD")
	timezoneName := flags.String("timezone", "UTC", "display timezone name for the metric spec")
	slaMinutes := flags.Int("sla-minutes", 60, "SLA target in minutes for the metric spec")
	asJSON := flags.Bool("json", false, "json")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl reporting weekly [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}

	if trim(*tasksPath) == "" {
		return errors.New("--tasks is required")
	}
	if trim(*outputDir) == "" {
		return errors.New("--output-dir is required")
	}
	weekStart, err := parseReportingTime(*weekStartRaw)
	if err != nil {
		return fmt.Errorf("parse --week-start: %w", err)
	}
	weekEnd, err := parseReportingTime(*weekEndRaw)
	if err != nil {
		return fmt.Errorf("parse --week-end: %w", err)
	}
	if !weekEnd.After(weekStart) {
		return errors.New("--week-end must be after --week-start")
	}

	resolvedRepoRoot := absPath(*repoRoot)
	resolvedTasksPath := resolvePathAgainstRepoRoot(resolvedRepoRoot, *tasksPath)
	resolvedOutputDir := resolvePathAgainstRepoRoot(resolvedRepoRoot, *outputDir)

	tasks, err := loadReportingTasks(resolvedTasksPath)
	if err != nil {
		return err
	}

	var events []domain.Event
	if trim(*eventsPath) != "" {
		resolvedEventsPath := resolvePathAgainstRepoRoot(resolvedRepoRoot, *eventsPath)
		events, err = loadReportingEvents(resolvedEventsPath)
		if err != nil {
			return err
		}
	}

	weekly := reporting.Build(tasks, events, weekStart, weekEnd)
	metricSpec := reporting.BuildOperationsMetricSpec(tasks, events, weekStart, weekEnd, firstNonBlankString(trim(*timezoneName), "UTC"), *slaMinutes)
	artifacts, err := reporting.WriteWeeklyOperationsBundle(resolvedOutputDir, weekly, &metricSpec)
	if err != nil {
		return fmt.Errorf("write weekly operations bundle: %w", err)
	}

	payload := map[string]any{
		"status":         "ok",
		"week_start":     weekStart.UTC().Format(time.RFC3339),
		"week_end":       weekEnd.UTC().Format(time.RFC3339),
		"task_count":     len(tasks),
		"event_count":    len(events),
		"summary":        structToMap(weekly.Summary),
		"artifacts":      structToMap(artifacts),
		"metric_spec":    map[string]any{"name": metricSpec.Name, "timezone": metricSpec.TimezoneName, "definition_count": len(metricSpec.Definitions), "value_count": len(metricSpec.Values)},
		"report_surface": map[string]any{"api": "/v2/reports/weekly", "export_api": "/v2/reports/weekly/export", "cli": "bigclawctl reporting weekly"},
	}
	return emit(payload, *asJSON, 0)
}

func parseReportingTime(raw string) (time.Time, error) {
	value := trim(raw)
	if value == "" {
		return time.Time{}, errors.New("value is required")
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			if layout == "2006-01-02" {
				return parsed.UTC(), nil
			}
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported time format %q", value)
}

func loadReportingTasks(path string) ([]domain.Task, error) {
	var tasks []domain.Task
	if err := loadReportingJSONArray(path, "tasks", &tasks); err != nil {
		return nil, fmt.Errorf("load tasks %s: %w", path, err)
	}
	return tasks, nil
}

func loadReportingEvents(path string) ([]domain.Event, error) {
	var events []domain.Event
	if err := loadReportingJSONArray(path, "events", &events); err != nil {
		return nil, fmt.Errorf("load events %s: %w", path, err)
	}
	return events, nil
}

func loadReportingJSONArray(path string, wrapperKey string, destination any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return errors.New("empty file")
	}
	if strings.HasPrefix(trimmed, "[") {
		return json.Unmarshal(body, destination)
	}

	var wrapped map[string]json.RawMessage
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return err
	}
	payload, ok := wrapped[wrapperKey]
	if !ok {
		return fmt.Errorf("expected top-level key %q", wrapperKey)
	}
	return json.Unmarshal(payload, destination)
}

func firstNonBlankString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
