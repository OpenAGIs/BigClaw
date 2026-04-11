package regression

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/risk"
)

type Record struct {
	Task   domain.Task
	Events []domain.Event
}

type Summary struct {
	TotalRegressions    int    `json:"total_regressions"`
	AffectedTasks       int    `json:"affected_tasks"`
	CriticalRegressions int    `json:"critical_regressions"`
	ReworkEvents        int    `json:"rework_events"`
	TopSource           string `json:"top_source,omitempty"`
	TopWorkflow         string `json:"top_workflow,omitempty"`
}

type Breakdown struct {
	Key                 string `json:"key"`
	TotalRegressions    int    `json:"total_regressions"`
	AffectedTasks       int    `json:"affected_tasks"`
	CriticalRegressions int    `json:"critical_regressions"`
	ReworkEvents        int    `json:"rework_events"`
}

type Hotspot struct {
	Dimension           string `json:"dimension"`
	Key                 string `json:"key"`
	TotalRegressions    int    `json:"total_regressions"`
	CriticalRegressions int    `json:"critical_regressions"`
	ReworkEvents        int    `json:"rework_events"`
}

type Finding struct {
	TaskID          string     `json:"task_id"`
	TraceID         string     `json:"trace_id,omitempty"`
	Workflow        string     `json:"workflow,omitempty"`
	Team            string     `json:"team,omitempty"`
	Template        string     `json:"template,omitempty"`
	Service         string     `json:"service,omitempty"`
	Severity        string     `json:"severity"`
	RegressionCount int        `json:"regression_count"`
	ReworkEvents    int        `json:"rework_events"`
	Attribution     string     `json:"attribution"`
	Summary         string     `json:"summary"`
	Risk            risk.Score `json:"risk_score"`
	AnchorTime      time.Time  `json:"anchor_time"`
}

type Center struct {
	Summary              Summary     `json:"summary"`
	WorkflowBreakdown    []Breakdown `json:"workflow_breakdown"`
	TeamBreakdown        []Breakdown `json:"team_breakdown"`
	TemplateBreakdown    []Breakdown `json:"template_breakdown"`
	ServiceBreakdown     []Breakdown `json:"service_breakdown"`
	AttributionBreakdown []Breakdown `json:"attribution_breakdown"`
	Hotspots             []Hotspot   `json:"hotspots"`
	Findings             []Finding   `json:"findings"`
}

func Build(records []Record) Center {
	center := Center{}
	workflow := make(map[string]*Breakdown)
	team := make(map[string]*Breakdown)
	template := make(map[string]*Breakdown)
	service := make(map[string]*Breakdown)
	attribution := make(map[string]*Breakdown)
	findings := make([]Finding, 0)
	for _, record := range records {
		finding, ok := buildFinding(record)
		if !ok {
			continue
		}
		findings = append(findings, finding)
		center.Summary.TotalRegressions += finding.RegressionCount
		center.Summary.AffectedTasks++
		center.Summary.ReworkEvents += finding.ReworkEvents
		if finding.Severity == "critical" {
			center.Summary.CriticalRegressions++
		}
		accumulateBreakdown(workflow, finding.Workflow, finding)
		accumulateBreakdown(team, finding.Team, finding)
		accumulateBreakdown(template, finding.Template, finding)
		accumulateBreakdown(service, finding.Service, finding)
		accumulateBreakdown(attribution, finding.Attribution, finding)
	}
	sort.SliceStable(findings, func(i, j int) bool {
		if severityRank(findings[i].Severity) == severityRank(findings[j].Severity) {
			if findings[i].RegressionCount == findings[j].RegressionCount {
				if findings[i].AnchorTime.Equal(findings[j].AnchorTime) {
					return findings[i].TaskID < findings[j].TaskID
				}
				return findings[i].AnchorTime.After(findings[j].AnchorTime)
			}
			return findings[i].RegressionCount > findings[j].RegressionCount
		}
		return severityRank(findings[i].Severity) < severityRank(findings[j].Severity)
	})
	center.Findings = findings
	center.WorkflowBreakdown = sortedBreakdowns(workflow)
	center.TeamBreakdown = sortedBreakdowns(team)
	center.TemplateBreakdown = sortedBreakdowns(template)
	center.ServiceBreakdown = sortedBreakdowns(service)
	center.AttributionBreakdown = sortedBreakdowns(attribution)
	center.Hotspots = buildHotspots(center)
	if len(center.AttributionBreakdown) > 0 {
		center.Summary.TopSource = center.AttributionBreakdown[0].Key
	}
	if len(center.WorkflowBreakdown) > 0 {
		center.Summary.TopWorkflow = center.WorkflowBreakdown[0].Key
	}
	return center
}

func buildFinding(record Record) (Finding, bool) {
	regressionCount := regressionCount(record)
	if regressionCount <= 0 {
		return Finding{}, false
	}
	reworkEvents := reworkCount(record)
	riskScore := risk.ScoreTask(record.Task, record.Events)
	attribution := attributionFor(record)
	severity := severityFor(record, regressionCount, riskScore)
	workflow := firstNonEmpty(record.Task.Metadata["workflow"], record.Task.Metadata["workflow_id"], record.Task.Metadata["flow"], "unassigned")
	team := firstNonEmpty(record.Task.Metadata["team"], record.Task.TenantID, "unassigned")
	template := firstNonEmpty(record.Task.Metadata["template"], record.Task.Metadata["template_id"], record.Task.Metadata["prompt_template"], "unassigned")
	service := firstNonEmpty(record.Task.Metadata["service"], record.Task.Metadata["service_name"], record.Task.Metadata["system"], "unassigned")
	return Finding{
		TaskID:          record.Task.ID,
		TraceID:         record.Task.TraceID,
		Workflow:        workflow,
		Team:            team,
		Template:        template,
		Service:         service,
		Severity:        severity,
		RegressionCount: regressionCount,
		ReworkEvents:    reworkEvents,
		Attribution:     attribution,
		Summary:         summaryFor(record, regressionCount, attribution),
		Risk:            riskScore,
		AnchorTime:      anchorTime(record.Task),
	}, true
}

func Trend(findings []Finding, since, until time.Time, bucket string) []TrendPoint {
	windowStart, windowEnd := bounds(findings, since, until)
	if windowStart.IsZero() || windowEnd.IsZero() {
		return nil
	}
	step := 24 * time.Hour
	if bucket == "hour" {
		step = time.Hour
	}
	windowStart = truncate(windowStart, bucket)
	windowEnd = truncate(windowEnd, bucket)
	if !windowEnd.After(windowStart) {
		windowEnd = windowStart.Add(step)
	} else {
		windowEnd = windowEnd.Add(step)
	}
	points := make([]TrendPoint, 0)
	index := make(map[time.Time]int)
	for cursor := windowStart; cursor.Before(windowEnd); cursor = cursor.Add(step) {
		index[cursor] = len(points)
		points = append(points, TrendPoint{Start: cursor, End: cursor.Add(step), Label: formatLabel(cursor, bucket)})
	}
	for _, finding := range findings {
		anchor := truncate(finding.AnchorTime, bucket)
		position, ok := index[anchor]
		if !ok {
			continue
		}
		point := &points[position]
		point.TotalRegressions += finding.RegressionCount
		point.AffectedTasks++
		point.ReworkEvents += finding.ReworkEvents
		if finding.Severity == "critical" {
			point.CriticalRegressions++
		}
	}
	return points
}

type TrendPoint struct {
	Start               time.Time `json:"start"`
	End                 time.Time `json:"end"`
	Label               string    `json:"label"`
	TotalRegressions    int       `json:"total_regressions"`
	AffectedTasks       int       `json:"affected_tasks"`
	CriticalRegressions int       `json:"critical_regressions"`
	ReworkEvents        int       `json:"rework_events"`
}

func regressionCount(record Record) int {
	count := metadataInt(record.Task, "regression_count", "regressions")
	if count > 0 {
		return count
	}
	if metadataBool(record.Task, "regression") || hasLabel(record.Task, "regression") {
		return 1
	}
	if metadataString(record.Task, "regression_source", "regression_cause") != "" {
		return 1
	}
	return 0
}

func reworkCount(record Record) int {
	count := metadataInt(record.Task, "rework_count")
	for _, event := range record.Events {
		if event.Type == domain.EventTaskRetried {
			count++
		}
	}
	return count
}

func severityFor(record Record, regressionCount int, riskScore risk.Score) string {
	explicit := strings.ToLower(strings.TrimSpace(record.Task.Metadata["regression_severity"]))
	switch explicit {
	case "critical", "high", "medium":
		return explicit
	}
	if regressionCount >= 2 || riskScore.RequiresApproval || record.Task.State == domain.TaskDeadLetter || record.Task.State == domain.TaskCancelled || record.Task.State == domain.TaskFailed {
		return "critical"
	}
	if record.Task.State == domain.TaskBlocked || record.Task.State == domain.TaskRetrying || reworkCount(record) > 0 {
		return "high"
	}
	return "medium"
}

func attributionFor(record Record) string {
	return firstNonEmpty(
		metadataString(record.Task, "regression_source", "regression_cause", "root_cause", "failure_reason", "blocked_reason"),
		eventMessage(record.Events),
		"unknown",
	)
}

func summaryFor(record Record, regressionCount int, attribution string) string {
	return firstNonEmpty(
		metadataString(record.Task, "regression_summary", "summary"),
		attribution,
		record.Task.Title,
		"regression finding",
	) + " (regressions=" + strconv.Itoa(regressionCount) + ")"
}

func accumulateBreakdown(target map[string]*Breakdown, key string, finding Finding) {
	key = strings.TrimSpace(key)
	if key == "" {
		key = "unassigned"
	}
	entry := target[key]
	if entry == nil {
		entry = &Breakdown{Key: key}
		target[key] = entry
	}
	entry.TotalRegressions += finding.RegressionCount
	entry.AffectedTasks++
	entry.ReworkEvents += finding.ReworkEvents
	if finding.Severity == "critical" {
		entry.CriticalRegressions++
	}
}

func sortedBreakdowns(target map[string]*Breakdown) []Breakdown {
	out := make([]Breakdown, 0, len(target))
	for _, item := range target {
		out = append(out, *item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].TotalRegressions == out[j].TotalRegressions {
			if out[i].CriticalRegressions == out[j].CriticalRegressions {
				return out[i].Key < out[j].Key
			}
			return out[i].CriticalRegressions > out[j].CriticalRegressions
		}
		return out[i].TotalRegressions > out[j].TotalRegressions
	})
	return out
}

func buildHotspots(center Center) []Hotspot {
	hotspots := make([]Hotspot, 0)
	appendHotspots := func(dimension string, breakdowns []Breakdown) {
		for _, item := range breakdowns {
			hotspots = append(hotspots, Hotspot{
				Dimension:           dimension,
				Key:                 item.Key,
				TotalRegressions:    item.TotalRegressions,
				CriticalRegressions: item.CriticalRegressions,
				ReworkEvents:        item.ReworkEvents,
			})
			if len(hotspots) >= 8 {
				break
			}
		}
	}
	appendHotspots("workflow", center.WorkflowBreakdown)
	appendHotspots("team", center.TeamBreakdown)
	appendHotspots("template", center.TemplateBreakdown)
	appendHotspots("service", center.ServiceBreakdown)
	sort.SliceStable(hotspots, func(i, j int) bool {
		if hotspots[i].TotalRegressions == hotspots[j].TotalRegressions {
			if hotspots[i].CriticalRegressions == hotspots[j].CriticalRegressions {
				if hotspots[i].Dimension == hotspots[j].Dimension {
					return hotspots[i].Key < hotspots[j].Key
				}
				return hotspots[i].Dimension < hotspots[j].Dimension
			}
			return hotspots[i].CriticalRegressions > hotspots[j].CriticalRegressions
		}
		return hotspots[i].TotalRegressions > hotspots[j].TotalRegressions
	})
	if len(hotspots) > 6 {
		hotspots = hotspots[:6]
	}
	return hotspots
}

func bounds(findings []Finding, since, until time.Time) (time.Time, time.Time) {
	if !since.IsZero() && !until.IsZero() {
		return since, until
	}
	var minAnchor time.Time
	var maxAnchor time.Time
	for _, finding := range findings {
		anchor := finding.AnchorTime
		if anchor.IsZero() {
			continue
		}
		if minAnchor.IsZero() || anchor.Before(minAnchor) {
			minAnchor = anchor
		}
		if maxAnchor.IsZero() || anchor.After(maxAnchor) {
			maxAnchor = anchor
		}
	}
	if since.IsZero() {
		since = minAnchor
	}
	if until.IsZero() {
		until = maxAnchor
	}
	return since, until
}

func truncate(value time.Time, bucket string) time.Time {
	if value.IsZero() {
		return value
	}
	if bucket == "hour" {
		return value.UTC().Truncate(time.Hour)
	}
	utc := value.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}

func formatLabel(value time.Time, bucket string) string {
	if bucket == "hour" {
		return value.UTC().Format(time.RFC3339)
	}
	return value.UTC().Format("2006-01-02")
}

func anchorTime(task domain.Task) time.Time {
	if !task.UpdatedAt.IsZero() {
		return task.UpdatedAt
	}
	return task.CreatedAt
}

func eventMessage(events []domain.Event) string {
	for index := len(events) - 1; index >= 0; index-- {
		if events[index].Payload == nil {
			continue
		}
		for _, key := range []string{"reason", "message", "note"} {
			if value, ok := events[index].Payload[key].(string); ok && strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
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

func metadataString(task domain.Task, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(task.Metadata[key]); value != "" {
			return value
		}
	}
	return ""
}

func hasLabel(task domain.Task, want string) bool {
	for _, label := range task.Labels {
		if strings.EqualFold(strings.TrimSpace(label), want) {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func severityRank(severity string) int {
	switch severity {
	case "critical":
		return 0
	case "high":
		return 1
	default:
		return 2
	}
}
