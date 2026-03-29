package controlcenterparity

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

type Task struct {
	ID          string    `json:"task_id"`
	Source      string    `json:"source"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Priority    int       `json:"priority"`
	RiskLevel   RiskLevel `json:"risk_level"`
}

type Queue struct {
	path  string
	tasks []Task
}

func NewQueue(path string) (*Queue, error) {
	return &Queue{path: path}, nil
}

func (q *Queue) Enqueue(task Task) error {
	q.tasks = append(q.tasks, task)
	sort.SliceStable(q.tasks, func(i, j int) bool {
		if q.tasks[i].Priority != q.tasks[j].Priority {
			return q.tasks[i].Priority < q.tasks[j].Priority
		}
		return q.tasks[i].ID < q.tasks[j].ID
	})
	if err := os.MkdirAll(filepath.Dir(q.path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(q.path, []byte("persisted\n"), 0o644)
}

func (q *Queue) PeekTasks() []Task {
	return append([]Task(nil), q.tasks...)
}

type Action struct {
	ActionID string
	Label    string
	Target   string
	Enabled  bool
	Reason   string
}

type Center struct {
	QueueDepth          int
	QueuedByPriority    map[string]int
	QueuedByRisk        map[string]int
	ExecutionMedia      map[string]int
	WaitingApprovalRuns int
	BlockedTasks        []string
	QueuedTasks         []string
	Actions             map[string][]Action
}

type SharedViewFilter struct {
	Label string
	Value string
}

type SharedViewContext struct {
	Filters      []SharedViewFilter
	ResultCount  int
	EmptyMessage string
}

func BuildQueueControlCenter(queue *Queue, runs []map[string]string) Center {
	center := Center{
		QueuedByPriority: map[string]int{"P0": 0, "P1": 0, "P2": 0},
		QueuedByRisk:     map[string]int{"low": 0, "medium": 0, "high": 0},
		ExecutionMedia:   map[string]int{},
		Actions:          map[string][]Action{},
	}
	for _, task := range queue.PeekTasks() {
		center.QueueDepth++
		center.QueuedTasks = append(center.QueuedTasks, task.ID)
		center.QueuedByPriority[priorityBucket(task.Priority)]++
		center.QueuedByRisk[string(task.RiskLevel)]++
	}

	blocked := map[string]bool{}
	for _, run := range runs {
		if medium := strings.TrimSpace(run["medium"]); medium != "" {
			center.ExecutionMedia[medium]++
		}
		if run["status"] == "needs-approval" {
			center.WaitingApprovalRuns++
			if taskID := strings.TrimSpace(run["task_id"]); taskID != "" {
				blocked[taskID] = true
			}
		}
	}
	for _, task := range queue.PeekTasks() {
		taskBlocked := blocked[task.ID]
		if taskBlocked {
			center.BlockedTasks = append(center.BlockedTasks, task.ID)
		}
		center.Actions[task.ID] = buildActions(task.ID, taskBlocked)
	}
	return center
}

func RenderQueueControlCenter(center Center, view *SharedViewContext) string {
	var builder strings.Builder
	builder.WriteString("# Queue Control Center\n\n")
	builder.WriteString(fmt.Sprintf("- Queue Depth: %d\n", center.QueueDepth))
	builder.WriteString(fmt.Sprintf("- Waiting Approval Runs: %d\n", center.WaitingApprovalRuns))
	builder.WriteString("\n## Queue By Priority\n\n")
	for _, bucket := range []string{"P0", "P1", "P2"} {
		builder.WriteString(fmt.Sprintf("- %s: %d\n", bucket, center.QueuedByPriority[bucket]))
	}
	builder.WriteString("\n## Queue By Risk\n\n")
	for _, bucket := range []string{"low", "medium", "high"} {
		builder.WriteString(fmt.Sprintf("- %s: %d\n", bucket, center.QueuedByRisk[bucket]))
	}
	builder.WriteString("\n## Execution Media\n\n")
	for _, medium := range sortedKeys(center.ExecutionMedia) {
		builder.WriteString(fmt.Sprintf("- %s: %d\n", medium, center.ExecutionMedia[medium]))
	}
	builder.WriteString("\n## Blocked Tasks\n\n")
	for _, taskID := range center.BlockedTasks {
		builder.WriteString(fmt.Sprintf("- %s\n", taskID))
	}
	builder.WriteString("\n## Actions\n\n")
	for _, taskID := range center.QueuedTasks {
		builder.WriteString(fmt.Sprintf("- %s: %s\n", taskID, renderActions(center.Actions[taskID])))
	}
	if view != nil {
		builder.WriteString("\n## View State\n\n")
		state := "ready"
		if view.ResultCount == 0 {
			state = "empty"
		}
		builder.WriteString(fmt.Sprintf("- State: %s\n", state))
		builder.WriteString(fmt.Sprintf("- Summary: %s\n", view.EmptyMessage))
		for _, filter := range view.Filters {
			builder.WriteString(fmt.Sprintf("- %s: %s\n", filter.Label, filter.Value))
		}
	}
	return builder.String()
}

func buildActions(target string, blocked bool) []Action {
	return []Action{
		{ActionID: "drill-down", Label: "Drill Down", Target: target, Enabled: true},
		{ActionID: "export", Label: "Export", Target: target, Enabled: true},
		{ActionID: "add-note", Label: "Add Note", Target: target, Enabled: true},
		{ActionID: "escalate", Label: "Escalate", Target: target, Enabled: blocked},
		{ActionID: "retry", Label: "Retry", Target: target, Enabled: true},
		{ActionID: "pause", Label: "Pause", Target: target, Enabled: !blocked, Reason: disabledReason(!blocked, "approval-blocked tasks should be escalated instead of paused")},
		{ActionID: "reassign", Label: "Reassign", Target: target, Enabled: true},
		{ActionID: "audit", Label: "Audit Trail", Target: target, Enabled: true},
	}
}

func renderActions(actions []Action) string {
	out := make([]string, 0, len(actions))
	for _, action := range actions {
		item := fmt.Sprintf("%s [%s] state=%s target=%s", action.Label, action.ActionID, state(action.Enabled), action.Target)
		if strings.TrimSpace(action.Reason) != "" {
			item += " reason=" + action.Reason
		}
		out = append(out, item)
	}
	return strings.Join(out, "; ")
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

func disabledReason(enabled bool, reason string) string {
	if enabled {
		return ""
	}
	return reason
}

func state(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

func sortedKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
