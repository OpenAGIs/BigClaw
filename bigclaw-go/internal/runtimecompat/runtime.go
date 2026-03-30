package runtimecompat

import "bigclaw-go/internal/domain"

type AuditEntry struct {
	Action  string
	Outcome string
}

type Run struct {
	RunID  string
	TaskID string
	Medium string
	Audits []AuditEntry
}

func NewRun(task domain.Task, runID, medium string) *Run {
	return &Run{
		RunID:  runID,
		TaskID: task.ID,
		Medium: medium,
	}
}

func (r *Run) audit(action, outcome string) {
	r.Audits = append(r.Audits, AuditEntry{Action: action, Outcome: outcome})
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

type Handler func(action string, payload map[string]any) string

type ToolRuntime struct {
	Policy   ToolPolicy
	Handlers map[string]Handler
}

func (r ToolRuntime) Invoke(toolName, action string, payload map[string]any, run *Run) ToolCallResult {
	if !r.Policy.Allows(toolName) {
		if run != nil {
			run.audit("tool.invoke", "blocked")
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
		handler = defaultHandler
	}
	output := handler(action, payload)
	if run != nil {
		run.audit("tool.invoke", "success")
	}
	return ToolCallResult{
		ToolName: toolName,
		Action:   action,
		Success:  true,
		Output:   output,
	}
}

func defaultHandler(action string, payload map[string]any) string {
	return action
}

type Decision struct {
	Medium   string
	Approved bool
	Reason   string
}

type WorkerExecutionResult struct {
	ToolResults []ToolCallResult
}

type ClawWorkerRuntime struct {
	ToolRuntime ToolRuntime
}

func (r ClawWorkerRuntime) Execute(task domain.Task, decision Decision, run *Run, toolPayloads map[string]map[string]any) WorkerExecutionResult {
	if !decision.Approved {
		if run != nil {
			run.audit("worker.lifecycle", "waiting-approval")
		}
		return WorkerExecutionResult{}
	}
	results := make([]ToolCallResult, 0, len(task.RequiredTools))
	for _, toolName := range task.RequiredTools {
		results = append(results, r.ToolRuntime.Invoke(toolName, "execute", toolPayloads[toolName], run))
	}
	if run != nil {
		run.audit("worker.lifecycle", "completed")
	}
	return WorkerExecutionResult{ToolResults: results}
}

func RouteMedium(task domain.Task) string {
	if task.RiskLevel == domain.RiskHigh {
		return "vm"
	}
	for _, tool := range task.RequiredTools {
		if tool == "browser" {
			return "browser"
		}
	}
	return "docker"
}
