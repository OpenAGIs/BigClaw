package runtimecompat

import "fmt"

type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

type Task struct {
	TaskID        string
	Source        string
	Title         string
	Description   string
	RequiredTools []string
	RiskLevel     RiskLevel
	Budget        float64
}

type LogEntry struct {
	Level   string
	Message string
	Fields  map[string]string
}

type TraceEntry struct {
	Span       string
	Status     string
	Attributes map[string]string
}

type AuditEntry struct {
	Action  string
	Actor   string
	Outcome string
	Details map[string]string
}

type TaskRun struct {
	TaskID  string
	RunID   string
	Medium  string
	Logs    []LogEntry
	Traces  []TraceEntry
	Audits  []AuditEntry
	Summary string
}

func NewTaskRun(task Task, runID, medium string) *TaskRun {
	return &TaskRun{
		TaskID: task.TaskID,
		RunID:  runID,
		Medium: medium,
	}
}

func (r *TaskRun) Log(level, message string, fields map[string]string) {
	r.Logs = append(r.Logs, LogEntry{Level: level, Message: message, Fields: cloneMap(fields)})
}

func (r *TaskRun) Trace(span, status string, attrs map[string]string) {
	r.Traces = append(r.Traces, TraceEntry{Span: span, Status: status, Attributes: cloneMap(attrs)})
}

func (r *TaskRun) Audit(action, actor, outcome string, details map[string]string) {
	r.Audits = append(r.Audits, AuditEntry{Action: action, Actor: actor, Outcome: outcome, Details: cloneMap(details)})
}

type ToolPolicy struct {
	AllowedTools []string
	BlockedTools []string
}

func (p ToolPolicy) Allows(toolName string) bool {
	for _, blocked := range p.BlockedTools {
		if blocked == toolName {
			return false
		}
	}
	if len(p.AllowedTools) == 0 {
		return true
	}
	for _, allowed := range p.AllowedTools {
		if allowed == toolName {
			return true
		}
	}
	return false
}

type ToolCallResult struct {
	ToolName string
	Action   string
	Success  bool
	Output   string
	Error    string
}

type ToolHandler func(action string, payload map[string]string) string

type ToolRuntime struct {
	Policy   ToolPolicy
	Handlers map[string]ToolHandler
}

func (r ToolRuntime) Invoke(toolName, action string, payload map[string]string, run *TaskRun, actor string) ToolCallResult {
	if actor == "" {
		actor = "tool-runtime"
	}
	resolvedPayload := cloneMap(payload)
	if !r.Policy.Allows(toolName) {
		if run != nil {
			fields := map[string]string{"tool": toolName, "operation": action}
			run.Log("error", "tool blocked", fields)
			run.Audit("tool.invoke", actor, "blocked", fields)
			run.Trace("tool.invoke", "blocked", fields)
		}
		return ToolCallResult{
			ToolName: toolName,
			Action:   action,
			Success:  false,
			Error:    "tool blocked by policy",
		}
	}

	handler := r.Handlers[toolName]
	if handler == nil {
		handler = defaultToolHandler
	}
	output := handler(action, resolvedPayload)
	if run != nil {
		fields := map[string]string{"tool": toolName, "operation": action}
		run.Log("info", "tool executed", fields)
		run.Audit("tool.invoke", actor, "success", fields)
		run.Trace("tool.invoke", "ok", fields)
	}
	return ToolCallResult{
		ToolName: toolName,
		Action:   action,
		Success:  true,
		Output:   output,
	}
}

func defaultToolHandler(action string, payload map[string]string) string {
	if len(payload) == 0 {
		return action
	}
	return fmt.Sprintf("%s:%v", action, payload)
}

type SandboxProfile struct {
	Medium           string
	Isolation        string
	NetworkAccess    string
	FilesystemAccess string
}

type SandboxRouter struct{}

func (SandboxRouter) ProfileFor(medium string) SandboxProfile {
	switch medium {
	case "docker":
		return SandboxProfile{Medium: "docker", Isolation: "container", NetworkAccess: "restricted", FilesystemAccess: "workspace-write"}
	case "browser":
		return SandboxProfile{Medium: "browser", Isolation: "browser", NetworkAccess: "restricted", FilesystemAccess: "workspace-write"}
	case "vm":
		return SandboxProfile{Medium: "vm", Isolation: "virtual-machine", NetworkAccess: "restricted", FilesystemAccess: "workspace-write"}
	default:
		return SandboxProfile{Medium: "none", Isolation: "disabled", NetworkAccess: "none", FilesystemAccess: "none"}
	}
}

type SchedulerDecision struct {
	Medium   string
	Approved bool
	Reason   string
}

type WorkerExecutionResult struct {
	Run            *TaskRun
	ToolResults    []ToolCallResult
	SandboxProfile SandboxProfile
}

type ClawWorkerRuntime struct {
	ToolRuntime   ToolRuntime
	SandboxRouter SandboxRouter
}

func (w ClawWorkerRuntime) Execute(task Task, decision SchedulerDecision, run *TaskRun, actor string, toolPayloads map[string]map[string]string) WorkerExecutionResult {
	if actor == "" {
		actor = "worker-runtime"
	}
	profile := w.SandboxRouter.ProfileFor(decision.Medium)
	run.Log("info", "worker assigned sandbox", map[string]string{
		"medium":         profile.Medium,
		"isolation":      profile.Isolation,
		"network_access": profile.NetworkAccess,
	})
	run.Trace("worker.sandbox", "ready", map[string]string{
		"medium":            profile.Medium,
		"isolation":         profile.Isolation,
		"filesystem_access": profile.FilesystemAccess,
	})

	if !decision.Approved {
		if profile.Medium == "none" {
			run.Log("warn", "worker paused by scheduler budget policy", map[string]string{"reason": decision.Reason})
			run.Audit("worker.lifecycle", actor, "paused", map[string]string{
				"medium": decision.Medium,
				"reason": decision.Reason,
			})
			run.Trace("worker.lifecycle", "blocked", map[string]string{"medium": decision.Medium, "reason": decision.Reason})
			return WorkerExecutionResult{Run: run, SandboxProfile: profile}
		}
		run.Log("warn", "worker waiting for approval", map[string]string{"medium": decision.Medium})
		run.Audit("worker.lifecycle", actor, "waiting-approval", map[string]string{"medium": decision.Medium})
		run.Trace("worker.lifecycle", "pending", map[string]string{"medium": decision.Medium})
		return WorkerExecutionResult{Run: run, SandboxProfile: profile}
	}

	run.Log("info", "worker started", map[string]string{
		"medium":     decision.Medium,
		"tool_count": fmt.Sprintf("%d", len(task.RequiredTools)),
	})
	run.Trace("worker.lifecycle", "started", map[string]string{
		"medium":     decision.Medium,
		"tool_count": fmt.Sprintf("%d", len(task.RequiredTools)),
	})

	results := make([]ToolCallResult, 0, len(task.RequiredTools))
	successful := make([]string, 0, len(task.RequiredTools))
	blocked := make([]string, 0)
	for _, toolName := range task.RequiredTools {
		result := w.ToolRuntime.Invoke(toolName, "execute", toolPayloads[toolName], run, actor)
		results = append(results, result)
		if result.Success {
			successful = append(successful, result.ToolName)
		} else {
			blocked = append(blocked, result.ToolName)
		}
	}
	run.Audit("worker.lifecycle", actor, "completed", map[string]string{
		"medium":           decision.Medium,
		"successful_tools": joinList(successful),
		"blocked_tools":    joinList(blocked),
	})
	run.Trace("worker.lifecycle", "completed", map[string]string{
		"medium":           decision.Medium,
		"successful_tools": fmt.Sprintf("%d", len(successful)),
		"blocked_tools":    fmt.Sprintf("%d", len(blocked)),
	})
	return WorkerExecutionResult{Run: run, ToolResults: results, SandboxProfile: profile}
}

type Scheduler struct{}

var mediumBudgetFloors = map[string]float64{
	"docker":  10.0,
	"browser": 20.0,
	"vm":      40.0,
}

func (Scheduler) Decide(task Task) SchedulerDecision {
	if task.Budget < 0 {
		return SchedulerDecision{Medium: "none", Approved: false, Reason: "invalid budget"}
	}
	if task.RiskLevel == RiskHigh {
		return applyBudgetPolicy(task, SchedulerDecision{Medium: "vm", Approved: false, Reason: "requires approval for high-risk task"})
	}
	for _, tool := range task.RequiredTools {
		if tool == "browser" {
			return applyBudgetPolicy(task, SchedulerDecision{Medium: "browser", Approved: true, Reason: "browser automation task"})
		}
	}
	if task.RiskLevel == RiskMedium {
		return applyBudgetPolicy(task, SchedulerDecision{Medium: "docker", Approved: true, Reason: "medium risk in docker"})
	}
	return applyBudgetPolicy(task, SchedulerDecision{Medium: "docker", Approved: true, Reason: "default low risk path"})
}

func applyBudgetPolicy(task Task, decision SchedulerDecision) SchedulerDecision {
	effectiveBudget := task.Budget
	if effectiveBudget <= 0 {
		return decision
	}
	requiredBudget := mediumBudgetFloors[decision.Medium]
	if effectiveBudget >= requiredBudget {
		return decision
	}
	if decision.Medium == "browser" && task.RiskLevel != RiskHigh && effectiveBudget >= mediumBudgetFloors["docker"] {
		return SchedulerDecision{
			Medium:   "docker",
			Approved: true,
			Reason:   fmt.Sprintf("budget degraded browser route to docker (budget %.1f < required %.1f)", effectiveBudget, requiredBudget),
		}
	}
	return SchedulerDecision{
		Medium:   "none",
		Approved: false,
		Reason:   fmt.Sprintf("paused: budget %.1f below required %s budget %.1f", effectiveBudget, decision.Medium, requiredBudget),
	}
}

func cloneMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func joinList(items []string) string {
	if len(items) == 0 {
		return ""
	}
	out := items[0]
	for i := 1; i < len(items); i++ {
		out += "," + items[i]
	}
	return out
}
