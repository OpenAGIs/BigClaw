package controlcentercompat

import (
	"fmt"
	"sort"
	"strings"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/reporting"
)

type Queue struct {
	tasks []domain.Task
}

func (q *Queue) Enqueue(task domain.Task) {
	q.tasks = append(q.tasks, task)
}

func (q *Queue) PeekTasks() []domain.Task {
	out := append([]domain.Task(nil), q.tasks...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Priority == out[j].Priority {
			return out[i].ID < out[j].ID
		}
		return out[i].Priority < out[j].Priority
	})
	return out
}

type RunSnapshot struct {
	TaskID string
	Status string
	Medium string
}

type QueueControlCenter struct {
	QueueDepth          int
	QueuedByPriority    map[string]int
	QueuedByRisk        map[string]int
	ExecutionMedia      map[string]int
	WaitingApprovalRuns int
	BlockedTasks        []string
	QueuedTasks         []string
	Actions             map[string][]reporting.ConsoleAction
}

type SharedViewFilter struct {
	Label string
	Value string
}

type SharedViewContext struct {
	Filters      []SharedViewFilter
	ResultCount  *int
	Loading      bool
	Errors       []string
	PartialData  []string
	EmptyMessage string
}

func (v SharedViewContext) State() string {
	switch {
	case v.Loading:
		return "loading"
	case len(v.Errors) > 0 && (v.ResultCount == nil || *v.ResultCount == 0):
		return "error"
	case v.ResultCount != nil && *v.ResultCount == 0 && len(v.PartialData) == 0:
		return "empty"
	case len(v.Errors) > 0 || len(v.PartialData) > 0:
		return "partial-data"
	default:
		return "ready"
	}
}

func (v SharedViewContext) Summary() string {
	switch v.State() {
	case "loading":
		return "Loading data for the current filters."
	case "error":
		return "Unable to load data for the current filters."
	case "empty":
		if strings.TrimSpace(v.EmptyMessage) != "" {
			return v.EmptyMessage
		}
		return "No records match the current filters."
	case "partial-data":
		return "Showing partial data while one or more sources are unavailable."
	default:
		return "Data is current for the selected filters."
	}
}

type Analytics struct{}

func (Analytics) BuildQueueControlCenter(queue *Queue, runs []RunSnapshot) QueueControlCenter {
	center := QueueControlCenter{
		QueuedByPriority: map[string]int{"P0": 0, "P1": 0, "P2": 0},
		QueuedByRisk:     map[string]int{"low": 0, "medium": 0, "high": 0},
		ExecutionMedia:   make(map[string]int),
		Actions:          make(map[string][]reporting.ConsoleAction),
	}
	for _, task := range queue.PeekTasks() {
		center.QueueDepth++
		center.QueuedTasks = append(center.QueuedTasks, task.ID)
		center.QueuedByPriority[priorityBucket(task.Priority)]++
		center.QueuedByRisk[riskBucket(task.RiskLevel)]++
		center.Actions[task.ID] = buildConsoleActions(task.ID, true, false, true)
	}
	for _, run := range runs {
		if run.Medium != "" {
			center.ExecutionMedia[run.Medium]++
		}
		if strings.EqualFold(strings.TrimSpace(run.Status), "needs-approval") {
			center.WaitingApprovalRuns++
			center.BlockedTasks = append(center.BlockedTasks, run.TaskID)
		}
	}
	sort.Strings(center.BlockedTasks)
	return center
}

func RenderQueueControlCenter(center QueueControlCenter, view *SharedViewContext) string {
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
		keys := make([]string, 0, len(center.ExecutionMedia))
		for key := range center.ExecutionMedia {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			builder.WriteString(fmt.Sprintf("- %s: %d\n", key, center.ExecutionMedia[key]))
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
	} else {
		for _, taskID := range center.QueuedTasks {
			builder.WriteString(fmt.Sprintf("- %s: %s\n", taskID, reporting.RenderConsoleActions(center.Actions[taskID])))
		}
	}
	if view != nil {
		builder.WriteString("\n## View State\n\n")
		builder.WriteString(fmt.Sprintf("- State: %s\n", view.State()))
		builder.WriteString(fmt.Sprintf("- Summary: %s\n", view.Summary()))
		if view.ResultCount != nil {
			builder.WriteString(fmt.Sprintf("- Result Count: %d\n", *view.ResultCount))
		}
		builder.WriteString("\n## Filters\n\n")
		if len(view.Filters) == 0 {
			builder.WriteString("- None\n")
		} else {
			for _, filter := range view.Filters {
				builder.WriteString(fmt.Sprintf("- %s: %s\n", filter.Label, filter.Value))
			}
		}
	}
	return builder.String()
}

func buildConsoleActions(target string, allowRetry, allowPause, allowEscalate bool) []reporting.ConsoleAction {
	return []reporting.ConsoleAction{
		{ActionID: "drill-down", Label: "Drill Down", Target: target, Enabled: true},
		{ActionID: "export", Label: "Export", Target: target, Enabled: true},
		{ActionID: "add-note", Label: "Add Note", Target: target, Enabled: true},
		{ActionID: "escalate", Label: "Escalate", Target: target, Enabled: allowEscalate, Reason: disabledReason(allowEscalate, "escalate is reserved for blocked queue items")},
		{ActionID: "retry", Label: "Retry", Target: target, Enabled: allowRetry, Reason: disabledReason(allowRetry, "retry is reserved for blocked queue items")},
		{ActionID: "pause", Label: "Pause", Target: target, Enabled: allowPause, Reason: disabledReason(allowPause, "approval-blocked tasks should be escalated instead of paused")},
		{ActionID: "reassign", Label: "Reassign", Target: target, Enabled: true},
		{ActionID: "audit", Label: "Audit Trail", Target: target, Enabled: true},
	}
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

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}
